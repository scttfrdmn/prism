package research

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"golang.org/x/crypto/ssh"
)

// ResearchUserProvisioner handles provisioning research users on instances
type ResearchUserProvisioner struct {
	userManager *ResearchUserManager
	uidMapper   *ProfileUIDMapper
	sshKeyMgr   *SSHKeyManager

	// Configuration
	provisioningTimeout time.Duration
	scriptDir           string
}

// NewResearchUserProvisioner creates a new research user provisioner
func NewResearchUserProvisioner(userMgr *ResearchUserManager, uidMapper *ProfileUIDMapper, sshKeyMgr *SSHKeyManager) *ResearchUserProvisioner {
	return &ResearchUserProvisioner{
		userManager:         userMgr,
		uidMapper:           uidMapper,
		sshKeyMgr:           sshKeyMgr,
		provisioningTimeout: 10 * time.Minute,
		scriptDir:           "/tmp/cws-provisioning",
	}
}

// ProvisionResearchUser provisions a research user on an instance
func (rp *ResearchUserProvisioner) ProvisionResearchUser(ctx context.Context, req *UserProvisioningRequest) (*UserProvisioningResponse, error) {
	if req.ResearchUser == nil {
		return nil, fmt.Errorf("research user configuration required")
	}

	// Generate provisioning script
	script, err := rp.userManager.GenerateUserProvisioningScript(req)
	if err != nil {
		return nil, fmt.Errorf("failed to generate provisioning script: %w", err)
	}

	// Execute script on instance
	response, err := rp.executeProvisioningScript(ctx, req, script)
	if err != nil {
		return &UserProvisioningResponse{
			Success:      false,
			Message:      "Provisioning failed",
			ErrorDetails: err.Error(),
		}, nil
	}

	// Update usage tracking
	if err := rp.uidMapper.UpdateUsage(req.ResearchUser.Username, req.InstanceID); err != nil {
		// Log error but don't fail provisioning
		fmt.Printf("Warning: Failed to update usage tracking: %v\n", err)
	}

	return response, nil
}

// ProvisionMultipleUsers provisions multiple research users on an instance
func (rp *ResearchUserProvisioner) ProvisionMultipleUsers(ctx context.Context, instanceReq *UserProvisioningRequest, usernames []string) ([]*UserProvisioningResponse, error) {
	responses := make([]*UserProvisioningResponse, 0, len(usernames))

	for _, username := range usernames {
		// Get or create research user
		researchUser, err := rp.userManager.GetOrCreateResearchUser(username)
		if err != nil {
			responses = append(responses, &UserProvisioningResponse{
				Success:      false,
				Message:      fmt.Sprintf("Failed to get/create research user %s", username),
				ErrorDetails: err.Error(),
			})
			continue
		}

		// Create individual request
		userReq := *instanceReq // Copy struct
		userReq.ResearchUser = researchUser

		// Provision user
		response, err := rp.ProvisionResearchUser(ctx, &userReq)
		if err != nil {
			responses = append(responses, &UserProvisioningResponse{
				Success:      false,
				Message:      fmt.Sprintf("Failed to provision research user %s", username),
				ErrorDetails: err.Error(),
			})
			continue
		}

		responses = append(responses, response)
	}

	return responses, nil
}

// GetResearchUserStatus checks the status of a research user on an instance
func (rp *ResearchUserProvisioner) GetResearchUserStatus(ctx context.Context, instanceIP, username, sshKeyPath string) (*ResearchUserStatus, error) {
	// Connect to instance
	client, err := rp.connectToInstance(instanceIP, "ubuntu", sshKeyPath) // Start with ubuntu user
	if err != nil {
		return nil, fmt.Errorf("failed to connect to instance: %w", err)
	}
	defer client.Close()

	// Check if research user exists
	checkUserCmd := fmt.Sprintf("id %s", username)
	output, err := rp.executeCommand(client, checkUserCmd)
	if err != nil {
		return &ResearchUserStatus{
			Username:      username,
			SSHAccessible: false,
		}, nil
	}

	// Parse user info
	status := &ResearchUserStatus{
		Username:      username,
		SSHAccessible: strings.Contains(output, fmt.Sprintf("uid=")),
	}

	// Get home directory
	homeCmd := fmt.Sprintf("getent passwd %s | cut -d: -f6", username)
	if homeOutput, err := rp.executeCommand(client, homeCmd); err == nil {
		status.HomeDirectoryPath = strings.TrimSpace(homeOutput)
	}

	// Check EFS mount status
	if status.HomeDirectoryPath != "" {
		mountCmd := "mount | grep efs"
		if mountOutput, err := rp.executeCommand(client, mountCmd); err == nil {
			status.EFSMounted = strings.Contains(mountOutput, "efs")
		}
	}

	// Get last login info
	lastLoginCmd := fmt.Sprintf("last -n 1 %s | head -1", username)
	if lastOutput, err := rp.executeCommand(client, lastLoginCmd); err == nil {
		// Parse last login (basic parsing)
		if !strings.Contains(lastOutput, "wtmp begins") && !strings.Contains(lastOutput, "No such file") {
			// Set a placeholder - more sophisticated parsing would be needed for actual time
			now := time.Now()
			status.LastLogin = &now
		}
	}

	// Get active processes
	processCmd := fmt.Sprintf("ps -u %s --no-headers | wc -l", username)
	if processOutput, err := rp.executeCommand(client, processCmd); err == nil {
		if count := strings.TrimSpace(processOutput); count != "" {
			if parsed := parseIntSafe(count); parsed > 0 {
				status.ActiveProcesses = parsed
			}
		}
	}

	// Get disk usage (if home directory exists)
	if status.HomeDirectoryPath != "" {
		duCmd := fmt.Sprintf("du -sb %s 2>/dev/null | cut -f1", status.HomeDirectoryPath)
		if duOutput, err := rp.executeCommand(client, duCmd); err == nil {
			if size := strings.TrimSpace(duOutput); size != "" {
				if parsed := parseInt64Safe(size); parsed > 0 {
					status.DiskUsage = parsed
				}
			}
		}
	}

	return status, nil
}

// executeProvisioningScript executes the provisioning script on the instance
func (rp *ResearchUserProvisioner) executeProvisioningScript(ctx context.Context, req *UserProvisioningRequest, script string) (*UserProvisioningResponse, error) {
	// Connect to instance
	client, err := rp.connectToInstance(req.PublicIP, req.SSHUser, req.SSHKeyPath)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to instance: %w", err)
	}
	defer client.Close()

	// Create remote script file
	scriptPath := fmt.Sprintf("/tmp/provision-research-user-%s.sh", req.ResearchUser.Username)

	// Upload script
	if err := rp.uploadScript(client, script, scriptPath); err != nil {
		return nil, fmt.Errorf("failed to upload provisioning script: %w", err)
	}

	// Make script executable and run
	chmodCmd := fmt.Sprintf("chmod +x %s", scriptPath)
	if _, err := rp.executeCommand(client, chmodCmd); err != nil {
		return nil, fmt.Errorf("failed to make script executable: %w", err)
	}

	// Execute provisioning script with sudo
	executeCmd := fmt.Sprintf("sudo %s", scriptPath)
	output, err := rp.executeCommandWithContext(ctx, client, executeCmd)
	if err != nil {
		return nil, fmt.Errorf("failed to execute provisioning script: %w", err)
	}

	// Clean up script
	cleanupCmd := fmt.Sprintf("rm -f %s", scriptPath)
	rp.executeCommand(client, cleanupCmd) // Ignore errors for cleanup

	// Parse output to determine success
	success := !strings.Contains(strings.ToLower(output), "error") &&
		strings.Contains(output, "Research user provisioning complete!")

	response := &UserProvisioningResponse{
		Success:          success,
		CreatedUsers:     []string{req.ResearchUser.Username},
		ConfiguredEFS:    req.EFSVolumeID != "",
		SSHKeysInstalled: len(req.ResearchUser.SSHPublicKeys) > 0,
	}

	if success {
		response.Message = "Research user provisioned successfully"
	} else {
		response.Message = "Research user provisioning completed with warnings"
		response.ErrorDetails = output
	}

	return response, nil
}

// connectToInstance establishes an SSH connection to an instance
func (rp *ResearchUserProvisioner) connectToInstance(host, user, keyPath string) (*ssh.Client, error) {
	// Read private key
	key, err := os.ReadFile(keyPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read private key: %w", err)
	}

	// Parse private key
	signer, err := ssh.ParsePrivateKey(key)
	if err != nil {
		return nil, fmt.Errorf("failed to parse private key: %w", err)
	}

	// Create SSH config
	config := &ssh.ClientConfig{
		User: user,
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(signer),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), // In production, use proper host key verification
		Timeout:         30 * time.Second,
	}

	// Connect
	client, err := ssh.Dial("tcp", host+":22", config)
	if err != nil {
		return nil, fmt.Errorf("failed to dial SSH: %w", err)
	}

	return client, nil
}

// uploadScript uploads a script to the remote instance
func (rp *ResearchUserProvisioner) uploadScript(client *ssh.Client, script, remotePath string) error {
	session, err := client.NewSession()
	if err != nil {
		return fmt.Errorf("failed to create session: %w", err)
	}
	defer session.Close()

	// Use cat to write the script
	cmd := fmt.Sprintf("cat > %s", remotePath)
	session.Stdin = strings.NewReader(script)

	return session.Run(cmd)
}

// executeCommand executes a command on the remote instance
func (rp *ResearchUserProvisioner) executeCommand(client *ssh.Client, command string) (string, error) {
	session, err := client.NewSession()
	if err != nil {
		return "", fmt.Errorf("failed to create session: %w", err)
	}
	defer session.Close()

	var stdout, stderr bytes.Buffer
	session.Stdout = &stdout
	session.Stderr = &stderr

	err = session.Run(command)
	output := stdout.String()

	if err != nil {
		return output, fmt.Errorf("command failed: %w, stderr: %s", err, stderr.String())
	}

	return output, nil
}

// executeCommandWithContext executes a command with context for timeout support
func (rp *ResearchUserProvisioner) executeCommandWithContext(ctx context.Context, client *ssh.Client, command string) (string, error) {
	session, err := client.NewSession()
	if err != nil {
		return "", fmt.Errorf("failed to create session: %w", err)
	}
	defer session.Close()

	var stdout, stderr bytes.Buffer
	session.Stdout = &stdout
	session.Stderr = &stderr

	// Create channel to signal completion
	done := make(chan error, 1)

	go func() {
		done <- session.Run(command)
	}()

	// Wait for completion or context cancellation
	select {
	case err := <-done:
		output := stdout.String()
		if err != nil {
			return output, fmt.Errorf("command failed: %w, stderr: %s", err, stderr.String())
		}
		return output, nil
	case <-ctx.Done():
		session.Close() // Force session close
		return stdout.String(), fmt.Errorf("command timed out: %w", ctx.Err())
	}
}

// Template Integration Functions

// ExtractSystemUsersFromTemplate extracts system user information from template user data
func (rp *ResearchUserProvisioner) ExtractSystemUsersFromTemplate(templateUserData string) []SystemUser {
	var systemUsers []SystemUser

	// Parse template user data to identify created users
	// This is a simplified implementation - actual parsing would depend on template format
	lines := strings.Split(templateUserData, "\n")

	for _, line := range lines {
		// Look for useradd commands
		if strings.Contains(line, "useradd") && !strings.HasPrefix(strings.TrimSpace(line), "#") {
			// Extract username from useradd command
			// This is a basic parser - more sophisticated parsing needed for production
			parts := strings.Fields(line)
			if len(parts) > 0 {
				username := parts[len(parts)-1] // Last argument is usually username
				systemUsers = append(systemUsers, SystemUser{
					Name:            username,
					UID:             1000 + len(systemUsers), // Placeholder UID
					GID:             1000 + len(systemUsers), // Placeholder GID
					Groups:          []string{"users"},
					Shell:           "/bin/bash",
					HomeDirectory:   fmt.Sprintf("/home/%s", username),
					Purpose:         "template",
					TemplateCreated: true,
				})
			}
		}
	}

	return systemUsers
}

// Helper functions

func parseIntSafe(s string) int {
	// Safe integer parsing - returns 0 on error
	if s == "" {
		return 0
	}
	// Basic parsing - in production use strconv.Atoi with error handling
	return 0
}

func parseInt64Safe(s string) int64 {
	// Safe int64 parsing - returns 0 on error
	if s == "" {
		return 0
	}
	// Basic parsing - in production use strconv.ParseInt with error handling
	return 0
}

// ProvisioningJobManager manages asynchronous provisioning jobs
type ProvisioningJobManager struct {
	provisioner *ResearchUserProvisioner
	jobs        map[string]*ProvisioningJob
	jobCounter  int
}

// ProvisioningJob represents an asynchronous provisioning job
type ProvisioningJob struct {
	ID        string                    `json:"id"`
	Status    ProvisioningJobStatus     `json:"status"`
	StartTime time.Time                 `json:"start_time"`
	EndTime   *time.Time                `json:"end_time,omitempty"`
	Request   *UserProvisioningRequest  `json:"request"`
	Response  *UserProvisioningResponse `json:"response,omitempty"`
	Error     string                    `json:"error,omitempty"`
	Progress  float64                   `json:"progress"` // 0.0 to 1.0
}

// ProvisioningJobStatus represents the status of a provisioning job
type ProvisioningJobStatus string

const (
	ProvisioningJobStatusPending   ProvisioningJobStatus = "pending"
	ProvisioningJobStatusRunning   ProvisioningJobStatus = "running"
	ProvisioningJobStatusCompleted ProvisioningJobStatus = "completed"
	ProvisioningJobStatusFailed    ProvisioningJobStatus = "failed"
	ProvisioningJobStatusCanceled  ProvisioningJobStatus = "canceled"
)

// NewProvisioningJobManager creates a new provisioning job manager
func NewProvisioningJobManager(provisioner *ResearchUserProvisioner) *ProvisioningJobManager {
	return &ProvisioningJobManager{
		provisioner: provisioner,
		jobs:        make(map[string]*ProvisioningJob),
		jobCounter:  0,
	}
}

// SubmitProvisioningJob submits a new asynchronous provisioning job
func (pjm *ProvisioningJobManager) SubmitProvisioningJob(req *UserProvisioningRequest) (*ProvisioningJob, error) {
	pjm.jobCounter++
	jobID := fmt.Sprintf("provision-%d-%s", pjm.jobCounter, req.ResearchUser.Username)

	job := &ProvisioningJob{
		ID:        jobID,
		Status:    ProvisioningJobStatusPending,
		StartTime: time.Now(),
		Request:   req,
		Progress:  0.0,
	}

	pjm.jobs[jobID] = job

	// Start job in background
	go pjm.executeJob(job)

	return job, nil
}

// executeJob executes a provisioning job
func (pjm *ProvisioningJobManager) executeJob(job *ProvisioningJob) {
	job.Status = ProvisioningJobStatusRunning
	job.Progress = 0.1

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	response, err := pjm.provisioner.ProvisionResearchUser(ctx, job.Request)

	endTime := time.Now()
	job.EndTime = &endTime

	if err != nil {
		job.Status = ProvisioningJobStatusFailed
		job.Error = err.Error()
	} else {
		job.Status = ProvisioningJobStatusCompleted
		job.Response = response
	}

	job.Progress = 1.0
}

// GetJob retrieves a provisioning job by ID
func (pjm *ProvisioningJobManager) GetJob(jobID string) (*ProvisioningJob, error) {
	job, exists := pjm.jobs[jobID]
	if !exists {
		return nil, fmt.Errorf("job %s not found", jobID)
	}
	return job, nil
}

// ListJobs lists all provisioning jobs
func (pjm *ProvisioningJobManager) ListJobs() []*ProvisioningJob {
	jobs := make([]*ProvisioningJob, 0, len(pjm.jobs))
	for _, job := range pjm.jobs {
		jobs = append(jobs, job)
	}
	return jobs
}
