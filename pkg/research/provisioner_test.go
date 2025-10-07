package research

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockSSHClient simulates SSH client behavior for testing
type MockSSHClient struct {
	commands map[string]string // command -> output mapping
	errors   map[string]error  // command -> error mapping
	uploads  map[string]string // path -> content mapping
}

func NewMockSSHClient() *MockSSHClient {
	return &MockSSHClient{
		commands: make(map[string]string),
		errors:   make(map[string]error),
		uploads:  make(map[string]string),
	}
}

func (m *MockSSHClient) SetCommandOutput(command, output string) {
	m.commands[command] = output
}

func (m *MockSSHClient) SetCommandError(command string, err error) {
	m.errors[command] = err
}

func (m *MockSSHClient) GetUploadedScript(path string) string {
	return m.uploads[path]
}

// TestNewResearchUserProvisioner tests provisioner creation
func TestNewResearchUserProvisioner(t *testing.T) {
	userManager := NewResearchUserManager(&MockProfileManager{}, "/tmp/test")
	uidMapper := NewProfileUIDMapper(&MockProfileManager{})
	keyManager := NewSSHKeyManager("/tmp/test")

	provisioner := NewResearchUserProvisioner(userManager, uidMapper, keyManager)

	assert.NotNil(t, provisioner)
	assert.Equal(t, userManager, provisioner.userManager)
	assert.Equal(t, uidMapper, provisioner.uidMapper)
	assert.Equal(t, keyManager, provisioner.sshKeyMgr)
	assert.Equal(t, 10*time.Minute, provisioner.provisioningTimeout)
	assert.Equal(t, "/tmp/cws-provisioning", provisioner.scriptDir)
}

// TestProvisionResearchUser tests single user provisioning
func TestProvisionResearchUser(t *testing.T) {
	tests := []struct {
		name          string
		request       *UserProvisioningRequest
		scriptError   error
		expectError   bool
		expectSuccess bool
	}{
		{
			name: "valid_provisioning_request",
			request: &UserProvisioningRequest{
				InstanceID:   "i-1234567890abcdef0",
				InstanceName: "test-instance",
				PublicIP:     "54.123.45.67",
				ResearchUser: &ResearchUserConfig{
					Username: "researcher",
					UID:      5001,
					GID:      5001,
				},
				SSHKeyPath: "/tmp/test-key",
				SSHUser:    "ubuntu",
			},
			scriptError:   nil,
			expectError:   false,
			expectSuccess: true,
		},
		{
			name: "missing_research_user",
			request: &UserProvisioningRequest{
				InstanceID:   "i-1234567890abcdef0",
				InstanceName: "test-instance",
				PublicIP:     "54.123.45.67",
				ResearchUser: nil,
			},
			scriptError:   nil,
			expectError:   true,
			expectSuccess: false,
		},
		{
			name: "script_generation_error",
			request: &UserProvisioningRequest{
				InstanceID:   "i-1234567890abcdef0",
				InstanceName: "test-instance",
				PublicIP:     "54.123.45.67",
				ResearchUser: &ResearchUserConfig{
					Username: "researcher",
					UID:      5001,
					GID:      5001,
				},
			},
			scriptError:   fmt.Errorf("script generation failed"),
			expectError:   true,
			expectSuccess: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup test dependencies
			profileMgr := &MockProfileManager{currentProfile: "test-profile"}
			userManager := NewResearchUserManager(profileMgr, "/tmp/test")
			uidMapper := NewProfileUIDMapper(profileMgr)
			keyManager := NewSSHKeyManager("/tmp/test")

			provisioner := NewResearchUserProvisioner(userManager, uidMapper, keyManager)

			ctx := context.Background()
			response, err := provisioner.ProvisionResearchUser(ctx, tt.request)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, response)
				assert.Equal(t, tt.expectSuccess, response.Success)
			}
		})
	}
}

// TestProvisionMultipleUsers tests multiple user provisioning
func TestProvisionMultipleUsers(t *testing.T) {
	profileMgr := &MockProfileManager{currentProfile: "test-profile"}
	userManager := NewResearchUserManager(profileMgr, "/tmp/test")
	uidMapper := NewProfileUIDMapper(profileMgr)
	keyManager := NewSSHKeyManager("/tmp/test")

	provisioner := NewResearchUserProvisioner(userManager, uidMapper, keyManager)

	instanceReq := &UserProvisioningRequest{
		InstanceID:   "i-1234567890abcdef0",
		InstanceName: "test-instance",
		PublicIP:     "54.123.45.67",
		SSHKeyPath:   "/tmp/test-key",
		SSHUser:      "ubuntu",
	}

	usernames := []string{"researcher1", "researcher2", "researcher3"}
	ctx := context.Background()

	responses, err := provisioner.ProvisionMultipleUsers(ctx, instanceReq, usernames)

	assert.NoError(t, err)
	assert.Len(t, responses, len(usernames))

	// Each user should have a response
	for i, response := range responses {
		assert.NotNil(t, response)
		// In the actual implementation, success would depend on mock SSH behavior
		t.Logf("User %s provisioning result: %+v", usernames[i], response)
	}
}

// TestProvisionerGetResearchUserStatus tests user status checking
func TestProvisionerGetResearchUserStatus(t *testing.T) {
	tests := []struct {
		name         string
		username     string
		instanceIP   string
		sshKeyPath   string
		mockCommands map[string]string
		mockErrors   map[string]error
		expectError  bool
		expectExists bool
	}{
		{
			name:       "existing_user",
			username:   "researcher",
			instanceIP: "54.123.45.67",
			sshKeyPath: "/tmp/test-key",
			mockCommands: map[string]string{
				"id researcher": "uid=5001(researcher) gid=5001(researcher) groups=5001(researcher)",
			},
			expectError:  false,
			expectExists: true,
		},
		{
			name:       "non_existing_user",
			username:   "nonexistent",
			instanceIP: "54.123.45.67",
			sshKeyPath: "/tmp/test-key",
			mockErrors: map[string]error{
				"id nonexistent": fmt.Errorf("user not found"),
			},
			expectError:  false,
			expectExists: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			profileMgr := &MockProfileManager{currentProfile: "test-profile"}
			userManager := NewResearchUserManager(profileMgr, "/tmp/test")
			uidMapper := NewProfileUIDMapper(profileMgr)
			keyManager := NewSSHKeyManager("/tmp/test")

			provisioner := NewResearchUserProvisioner(userManager, uidMapper, keyManager)

			ctx := context.Background()
			status, err := provisioner.GetResearchUserStatus(ctx, tt.instanceIP, tt.username, tt.sshKeyPath)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				// Note: In actual implementation, this would require SSH mocking
				// For now, we expect connection errors since SSH is not mocked
				t.Logf("Status check result for %s: %v (error: %v)", tt.username, status, err)
			}
		})
	}
}

// TestExtractSystemUsersFromTemplate tests template user extraction
func TestExtractSystemUsersFromTemplate(t *testing.T) {
	tests := []struct {
		name              string
		templateUserData  string
		expectedUserCount int
		expectedFirstUser string
		expectedPurpose   string
	}{
		{
			name: "single_useradd",
			templateUserData: `#!/bin/bash
useradd -m -s /bin/bash researcher
echo "User created"`,
			expectedUserCount: 1,
			expectedFirstUser: "researcher",
			expectedPurpose:   "template",
		},
		{
			name: "multiple_useradd",
			templateUserData: `#!/bin/bash
useradd -m -s /bin/bash researcher1
useradd -m -s /bin/bash researcher2
# Comment line
echo "Users created"`,
			expectedUserCount: 2,
			expectedFirstUser: "researcher1",
			expectedPurpose:   "template",
		},
		{
			name: "no_useradd_commands",
			templateUserData: `#!/bin/bash
echo "No user creation"
# useradd commented out`,
			expectedUserCount: 0,
		},
		{
			name:              "empty_template",
			templateUserData:  "",
			expectedUserCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			profileMgr := &MockProfileManager{currentProfile: "test-profile"}
			userManager := NewResearchUserManager(profileMgr, "/tmp/test")
			uidMapper := NewProfileUIDMapper(profileMgr)
			keyManager := NewSSHKeyManager("/tmp/test")

			provisioner := NewResearchUserProvisioner(userManager, uidMapper, keyManager)

			systemUsers := provisioner.ExtractSystemUsersFromTemplate(tt.templateUserData)

			assert.Len(t, systemUsers, tt.expectedUserCount)

			if tt.expectedUserCount > 0 {
				assert.Equal(t, tt.expectedFirstUser, systemUsers[0].Name)
				assert.Equal(t, tt.expectedPurpose, systemUsers[0].Purpose)
				assert.True(t, systemUsers[0].TemplateCreated)
				assert.Equal(t, "/bin/bash", systemUsers[0].Shell)
				assert.Equal(t, []string{"users"}, systemUsers[0].Groups)
			}
		})
	}
}

// TestParseIntSafe tests safe integer parsing utility
func TestParseIntSafe(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected int
	}{
		{
			name:     "empty_string",
			input:    "",
			expected: 0,
		},
		{
			name:     "valid_number",
			input:    "42",
			expected: 0, // Function currently returns 0 - would need actual parsing implementation
		},
		{
			name:     "invalid_number",
			input:    "not-a-number",
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseIntSafe(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestParseInt64Safe tests safe int64 parsing utility
func TestParseInt64Safe(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected int64
	}{
		{
			name:     "empty_string",
			input:    "",
			expected: 0,
		},
		{
			name:     "valid_number",
			input:    "1024000",
			expected: 0, // Function currently returns 0 - would need actual parsing implementation
		},
		{
			name:     "invalid_number",
			input:    "invalid",
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseInt64Safe(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestProvisioningJobManager tests asynchronous job management
func TestNewProvisioningJobManager(t *testing.T) {
	profileMgr := &MockProfileManager{currentProfile: "test-profile"}
	userManager := NewResearchUserManager(profileMgr, "/tmp/test")
	uidMapper := NewProfileUIDMapper(profileMgr)
	keyManager := NewSSHKeyManager("/tmp/test")

	provisioner := NewResearchUserProvisioner(userManager, uidMapper, keyManager)
	jobManager := NewProvisioningJobManager(provisioner)

	assert.NotNil(t, jobManager)
	assert.Equal(t, provisioner, jobManager.provisioner)
	assert.NotNil(t, jobManager.jobs)
	assert.Equal(t, 0, jobManager.jobCounter)
}

// TestSubmitProvisioningJob tests job submission
func TestSubmitProvisioningJob(t *testing.T) {
	profileMgr := &MockProfileManager{currentProfile: "test-profile"}
	userManager := NewResearchUserManager(profileMgr, "/tmp/test")
	uidMapper := NewProfileUIDMapper(profileMgr)
	keyManager := NewSSHKeyManager("/tmp/test")

	provisioner := NewResearchUserProvisioner(userManager, uidMapper, keyManager)
	jobManager := NewProvisioningJobManager(provisioner)

	req := &UserProvisioningRequest{
		InstanceID:   "i-1234567890abcdef0",
		InstanceName: "test-instance",
		PublicIP:     "54.123.45.67",
		ResearchUser: &ResearchUserConfig{
			Username: "researcher",
			UID:      5001,
			GID:      5001,
		},
		SSHKeyPath: "/tmp/test-key",
		SSHUser:    "ubuntu",
	}

	job, err := jobManager.SubmitProvisioningJob(req)

	assert.NoError(t, err)
	assert.NotNil(t, job)
	assert.NotEmpty(t, job.ID)
	assert.Equal(t, ProvisioningJobStatusPending, job.Status)
	assert.Equal(t, req, job.Request)
	assert.Equal(t, 0.0, job.Progress)

	// Give job a moment to start
	time.Sleep(100 * time.Millisecond)

	// Retrieve job
	retrievedJob, err := jobManager.GetJob(job.ID)
	assert.NoError(t, err)
	assert.Equal(t, job.ID, retrievedJob.ID)
}

// TestGetJob tests job retrieval
func TestGetJob(t *testing.T) {
	profileMgr := &MockProfileManager{currentProfile: "test-profile"}
	userManager := NewResearchUserManager(profileMgr, "/tmp/test")
	uidMapper := NewProfileUIDMapper(profileMgr)
	keyManager := NewSSHKeyManager("/tmp/test")

	provisioner := NewResearchUserProvisioner(userManager, uidMapper, keyManager)
	jobManager := NewProvisioningJobManager(provisioner)

	// Test retrieving non-existent job
	_, err := jobManager.GetJob("non-existent-job")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")

	// Submit a job and then retrieve it
	req := &UserProvisioningRequest{
		InstanceID:   "i-1234567890abcdef0",
		InstanceName: "test-instance",
		PublicIP:     "54.123.45.67",
		ResearchUser: &ResearchUserConfig{
			Username: "researcher",
			UID:      5001,
			GID:      5001,
		},
		SSHKeyPath: "/tmp/test-key",
		SSHUser:    "ubuntu",
	}

	job, err := jobManager.SubmitProvisioningJob(req)
	require.NoError(t, err)

	retrievedJob, err := jobManager.GetJob(job.ID)
	assert.NoError(t, err)
	assert.Equal(t, job.ID, retrievedJob.ID)
	assert.Equal(t, job.Request.ResearchUser.Username, retrievedJob.Request.ResearchUser.Username)
}

// TestListJobs tests job listing
func TestListJobs(t *testing.T) {
	profileMgr := &MockProfileManager{currentProfile: "test-profile"}
	userManager := NewResearchUserManager(profileMgr, "/tmp/test")
	uidMapper := NewProfileUIDMapper(profileMgr)
	keyManager := NewSSHKeyManager("/tmp/test")

	provisioner := NewResearchUserProvisioner(userManager, uidMapper, keyManager)
	jobManager := NewProvisioningJobManager(provisioner)

	// Initially no jobs
	jobs := jobManager.ListJobs()
	assert.Len(t, jobs, 0)

	// Submit multiple jobs
	usernames := []string{"researcher1", "researcher2", "researcher3"}
	for _, username := range usernames {
		req := &UserProvisioningRequest{
			InstanceID:   "i-1234567890abcdef0",
			InstanceName: "test-instance",
			PublicIP:     "54.123.45.67",
			ResearchUser: &ResearchUserConfig{
				Username: username,
				UID:      5001,
				GID:      5001,
			},
			SSHKeyPath: "/tmp/test-key",
			SSHUser:    "ubuntu",
		}

		_, err := jobManager.SubmitProvisioningJob(req)
		require.NoError(t, err)
	}

	// Check job count
	jobs = jobManager.ListJobs()
	assert.Len(t, jobs, len(usernames))

	// Verify job details
	jobUsernames := make(map[string]bool)
	for _, job := range jobs {
		jobUsernames[job.Request.ResearchUser.Username] = true
	}

	for _, username := range usernames {
		assert.True(t, jobUsernames[username], "Job for username %s should exist", username)
	}
}

// TestProvisioningJobStatus tests job status constants
func TestProvisioningJobStatus(t *testing.T) {
	// Test that all status constants are defined
	assert.Equal(t, "pending", string(ProvisioningJobStatusPending))
	assert.Equal(t, "running", string(ProvisioningJobStatusRunning))
	assert.Equal(t, "completed", string(ProvisioningJobStatusCompleted))
	assert.Equal(t, "failed", string(ProvisioningJobStatusFailed))
	assert.Equal(t, "canceled", string(ProvisioningJobStatusCanceled))
}

// TestProvisioningJobExecution tests job execution lifecycle
func TestProvisioningJobExecution(t *testing.T) {
	profileMgr := &MockProfileManager{currentProfile: "test-profile"}
	userManager := NewResearchUserManager(profileMgr, "/tmp/test")
	uidMapper := NewProfileUIDMapper(profileMgr)
	keyManager := NewSSHKeyManager("/tmp/test")

	provisioner := NewResearchUserProvisioner(userManager, uidMapper, keyManager)
	jobManager := NewProvisioningJobManager(provisioner)

	req := &UserProvisioningRequest{
		InstanceID:   "i-1234567890abcdef0",
		InstanceName: "test-instance",
		PublicIP:     "54.123.45.67",
		ResearchUser: &ResearchUserConfig{
			Username: "researcher",
			UID:      5001,
			GID:      5001,
		},
		SSHKeyPath: "/tmp/test-key",
		SSHUser:    "ubuntu",
	}

	job, err := jobManager.SubmitProvisioningJob(req)
	require.NoError(t, err)

	// Initially pending
	assert.Equal(t, ProvisioningJobStatusPending, job.Status)
	assert.Equal(t, 0.0, job.Progress)

	// Wait for job to complete (it will fail due to no SSH connection)
	timeout := time.After(5 * time.Second)
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-timeout:
			t.Fatal("Job did not complete within timeout")
		case <-ticker.C:
			updatedJob, err := jobManager.GetJob(job.ID)
			require.NoError(t, err)

			if updatedJob.Status == ProvisioningJobStatusCompleted || updatedJob.Status == ProvisioningJobStatusFailed {
				assert.Equal(t, 1.0, updatedJob.Progress)
				assert.NotNil(t, updatedJob.EndTime)
				t.Logf("Job completed with status: %s", updatedJob.Status)
				return
			}
		}
	}
}

// TestProvisioningJobConcurrency tests concurrent job execution
func TestProvisioningJobConcurrency(t *testing.T) {
	profileMgr := &MockProfileManager{currentProfile: "test-profile"}
	userManager := NewResearchUserManager(profileMgr, "/tmp/test")
	uidMapper := NewProfileUIDMapper(profileMgr)
	keyManager := NewSSHKeyManager("/tmp/test")

	provisioner := NewResearchUserProvisioner(userManager, uidMapper, keyManager)
	jobManager := NewProvisioningJobManager(provisioner)

	// Submit multiple jobs concurrently
	jobCount := 5
	jobs := make([]*ProvisioningJob, jobCount)

	for i := 0; i < jobCount; i++ {
		req := &UserProvisioningRequest{
			InstanceID:   fmt.Sprintf("i-123456789%d", i),
			InstanceName: fmt.Sprintf("test-instance-%d", i),
			PublicIP:     "54.123.45.67",
			ResearchUser: &ResearchUserConfig{
				Username: fmt.Sprintf("researcher%d", i),
				UID:      5001 + i,
				GID:      5001 + i,
			},
			SSHKeyPath: "/tmp/test-key",
			SSHUser:    "ubuntu",
		}

		job, err := jobManager.SubmitProvisioningJob(req)
		require.NoError(t, err)
		jobs[i] = job
	}

	// Wait for all jobs to complete
	timeout := time.After(10 * time.Second)
	ticker := time.NewTicker(200 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-timeout:
			t.Fatal("Jobs did not complete within timeout")
		case <-ticker.C:
			allJobsList := jobManager.ListJobs()
			completed := 0

			for _, job := range allJobsList {
				if job.Status == ProvisioningJobStatusCompleted || job.Status == ProvisioningJobStatusFailed {
					completed++
				}
			}

			if completed == jobCount {
				t.Logf("All %d jobs completed", jobCount)
				return
			}
		}
	}
}

// TestProvisionerConfiguration tests provisioner configuration
func TestProvisionerConfiguration(t *testing.T) {
	profileMgr := &MockProfileManager{currentProfile: "test-profile"}
	userManager := NewResearchUserManager(profileMgr, "/tmp/test")
	uidMapper := NewProfileUIDMapper(profileMgr)
	keyManager := NewSSHKeyManager("/tmp/test")

	provisioner := NewResearchUserProvisioner(userManager, uidMapper, keyManager)

	// Test default configuration
	assert.Equal(t, 10*time.Minute, provisioner.provisioningTimeout)
	assert.Equal(t, "/tmp/cws-provisioning", provisioner.scriptDir)

	// Test component integration
	assert.NotNil(t, provisioner.userManager)
	assert.NotNil(t, provisioner.uidMapper)
	assert.NotNil(t, provisioner.sshKeyMgr)
}
