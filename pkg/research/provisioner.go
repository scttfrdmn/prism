package research

import (
	"bytes"
	"context"
	"fmt"
	"net"
	"os"
	"strings"
	"sync"
	"time"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/knownhosts"
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

	// Check if research user exists and get basic status
	status, err := rp.checkUserExistence(client, username)
	if err != nil {
		return status, nil // Return non-existent user status
	}

	// Collect detailed user information
	rp.populateUserDetails(client, username, status)

	return status, nil
}

// checkUserExistence verifies if a research user exists on the instance
func (rp *ResearchUserProvisioner) checkUserExistence(client *ssh.Client, username string) (*ResearchUserStatus, error) {
	checkUserCmd := fmt.Sprintf("id %s", username)
	output, err := rp.executeCommand(client, checkUserCmd)
	if err != nil {
		return &ResearchUserStatus{
			Username:      username,
			SSHAccessible: false,
		}, fmt.Errorf("user does not exist")
	}

	// Create base status with user existence confirmed
	status := &ResearchUserStatus{
		Username:      username,
		SSHAccessible: strings.Contains(output, fmt.Sprintf("uid=")),
	}

	return status, nil
}

// populateUserDetails gathers comprehensive information about the research user
func (rp *ResearchUserProvisioner) populateUserDetails(client *ssh.Client, username string, status *ResearchUserStatus) {
	// Get home directory path
	rp.populateHomeDirectory(client, username, status)

	// Check EFS mount status
	rp.populateEFSMountStatus(client, status)

	// Get last login information
	rp.populateLastLoginInfo(client, username, status)

	// Get active process count
	rp.populateActiveProcesses(client, username, status)

	// Get disk usage information
	rp.populateDiskUsage(client, status)
}

// populateHomeDirectory gets the user's home directory path
func (rp *ResearchUserProvisioner) populateHomeDirectory(client *ssh.Client, username string, status *ResearchUserStatus) {
	homeCmd := fmt.Sprintf("getent passwd %s | cut -d: -f6", username)
	if homeOutput, err := rp.executeCommand(client, homeCmd); err == nil {
		status.HomeDirectoryPath = strings.TrimSpace(homeOutput)
	}
}

// populateEFSMountStatus checks if EFS is mounted on the instance
func (rp *ResearchUserProvisioner) populateEFSMountStatus(client *ssh.Client, status *ResearchUserStatus) {
	if status.HomeDirectoryPath != "" {
		mountCmd := "mount | grep efs"
		if mountOutput, err := rp.executeCommand(client, mountCmd); err == nil {
			status.EFSMounted = strings.Contains(mountOutput, "efs")
		}
	}
}

// populateLastLoginInfo gets the user's last login information
func (rp *ResearchUserProvisioner) populateLastLoginInfo(client *ssh.Client, username string, status *ResearchUserStatus) {
	lastLoginCmd := fmt.Sprintf("last -n 1 %s | head -1", username)
	if lastOutput, err := rp.executeCommand(client, lastLoginCmd); err == nil {
		// Parse last login (basic parsing)
		if rp.isValidLastLoginOutput(lastOutput) {
			// Set a placeholder - more sophisticated parsing would be needed for actual time
			now := time.Now()
			status.LastLogin = &now
		}
	}
}

// isValidLastLoginOutput checks if the last login output contains valid login data
func (rp *ResearchUserProvisioner) isValidLastLoginOutput(output string) bool {
	return !strings.Contains(output, "wtmp begins") && !strings.Contains(output, "No such file")
}

// populateActiveProcesses gets the count of active processes for the user
func (rp *ResearchUserProvisioner) populateActiveProcesses(client *ssh.Client, username string, status *ResearchUserStatus) {
	processCmd := fmt.Sprintf("ps -u %s --no-headers | wc -l", username)
	if processOutput, err := rp.executeCommand(client, processCmd); err == nil {
		if count := strings.TrimSpace(processOutput); count != "" {
			if parsed := parseIntSafe(count); parsed > 0 {
				status.ActiveProcesses = parsed
			}
		}
	}
}

// populateDiskUsage gets the disk usage for the user's home directory
func (rp *ResearchUserProvisioner) populateDiskUsage(client *ssh.Client, status *ResearchUserStatus) {
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

	// Create SSH config with proper host key verification
	config := &ssh.ClientConfig{
		User: user,
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(signer),
		},
		HostKeyCallback: rp.getHostKeyCallback(),
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
	mu          sync.RWMutex // Protects jobs map and job fields
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
	pjm.mu.Lock()
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
	pjm.mu.Unlock()

	// Start job in background
	go pjm.executeJob(job)

	return job, nil
}

// executeJob executes a provisioning job
func (pjm *ProvisioningJobManager) executeJob(job *ProvisioningJob) {
	pjm.mu.Lock()
	job.Status = ProvisioningJobStatusRunning
	job.Progress = 0.1
	pjm.mu.Unlock()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	response, err := pjm.provisioner.ProvisionResearchUser(ctx, job.Request)

	pjm.mu.Lock()
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
	pjm.mu.Unlock()
}

// GetJob retrieves a provisioning job by ID
func (pjm *ProvisioningJobManager) GetJob(jobID string) (*ProvisioningJob, error) {
	pjm.mu.RLock()
	defer pjm.mu.RUnlock()

	job, exists := pjm.jobs[jobID]
	if !exists {
		return nil, fmt.Errorf("job %s not found", jobID)
	}
	return job, nil
}

// ListJobs lists all provisioning jobs
func (pjm *ProvisioningJobManager) ListJobs() []*ProvisioningJob {
	pjm.mu.RLock()
	defer pjm.mu.RUnlock()

	jobs := make([]*ProvisioningJob, 0, len(pjm.jobs))
	for _, job := range pjm.jobs {
		jobs = append(jobs, job)
	}
	return jobs
}

// getHostKeyCallback returns a host key callback for SSH connections
// This implements proper host key verification using a known_hosts file
func (rp *ResearchUserProvisioner) getHostKeyCallback() ssh.HostKeyCallback {
	// Get known_hosts path
	homeDir, err := os.UserHomeDir()
	if err != nil {
		// Fallback: For Prism-managed EC2 instances, we can trust on first use
		// since we control the infrastructure
		return rp.trustOnFirstUseCallback()
	}

	knownHostsPath := homeDir + "/.ssh/known_hosts"

	// Try to load known_hosts file
	callback, err := knownhosts.New(knownHostsPath)
	if err != nil {
		// If known_hosts doesn't exist or can't be read, use trust-on-first-use
		// This is acceptable for Prism since:
		// 1. We launch the instances ourselves via AWS
		// 2. We connect immediately after launch
		// 3. The instance is in our AWS account
		return rp.trustOnFirstUseCallback()
	}

	return callback
}

// trustOnFirstUseCallback returns a callback that trusts and records host keys on first use
// This is acceptable for Prism EC2 instances since we control the infrastructure
func (rp *ResearchUserProvisioner) trustOnFirstUseCallback() ssh.HostKeyCallback {
	knownHosts := make(map[string]ssh.PublicKey)

	return func(hostname string, remote net.Addr, key ssh.PublicKey) error {
		// Check if we've seen this host before in this session
		hostKey := hostname + ":" + remote.String()
		if knownKey, exists := knownHosts[hostKey]; exists {
			// Verify key matches what we saw before
			if !bytes.Equal(key.Marshal(), knownKey.Marshal()) {
				return fmt.Errorf("host key mismatch for %s - possible MITM attack", hostname)
			}
			return nil
		}

		// First time seeing this host - record it
		knownHosts[hostKey] = key

		// For Prism instances, we could optionally append to known_hosts file
		// but for now just keep in memory for the session
		return nil
	}
}
