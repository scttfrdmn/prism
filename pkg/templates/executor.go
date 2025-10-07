// Package templates provides remote execution capabilities for template application.
package templates

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// SSHRemoteExecutor implements RemoteExecutor using SSH connections
type SSHRemoteExecutor struct {
	keyPath string
	user    string
}

// NewSSHRemoteExecutor creates a new SSH-based remote executor
func NewSSHRemoteExecutor(keyPath, user string) *SSHRemoteExecutor {
	return &SSHRemoteExecutor{
		keyPath: keyPath,
		user:    user,
	}
}

// Execute executes a single command on the remote instance via SSH
func (e *SSHRemoteExecutor) Execute(ctx context.Context, instanceName string, command string) (*ExecutionResult, error) {
	startTime := time.Now()

	// Get instance IP (this would integrate with existing instance management)
	instanceIP, err := e.getInstanceIP(instanceName)
	if err != nil {
		return nil, fmt.Errorf("failed to get instance IP: %w", err)
	}

	// Build SSH command
	sshArgs := []string{
		"-i", e.keyPath,
		"-o", "StrictHostKeyChecking=no",
		"-o", "UserKnownHostsFile=/dev/null",
		"-o", "LogLevel=ERROR",
		fmt.Sprintf("%s@%s", e.user, instanceIP),
		command,
	}

	// Execute command with context
	cmd := exec.CommandContext(ctx, "ssh", sshArgs...)

	// Capture output
	stdout, err := cmd.Output()
	result := &ExecutionResult{
		Duration: time.Since(startTime),
		Stdout:   strings.TrimSpace(string(stdout)),
	}

	if err != nil {
		// Handle exit code errors
		if exitError, ok := err.(*exec.ExitError); ok {
			result.ExitCode = exitError.ExitCode()
			result.Stderr = strings.TrimSpace(string(exitError.Stderr))
		} else {
			return nil, fmt.Errorf("SSH execution failed: %w", err)
		}
	} else {
		result.ExitCode = 0
	}

	return result, nil
}

// ExecuteScript executes a script on the remote instance via SSH
func (e *SSHRemoteExecutor) ExecuteScript(ctx context.Context, instanceName string, script string) (*ExecutionResult, error) {
	startTime := time.Now()

	// Get instance IP
	instanceIP, err := e.getInstanceIP(instanceName)
	if err != nil {
		return nil, fmt.Errorf("failed to get instance IP: %w", err)
	}

	// Create temporary script file and execute it
	command := fmt.Sprintf("cat > /tmp/cloudworkstation-script.sh << 'EOF'\n%s\nEOF\nchmod +x /tmp/cloudworkstation-script.sh\n/tmp/cloudworkstation-script.sh\nrm -f /tmp/cloudworkstation-script.sh", script)

	// Build SSH command
	sshArgs := []string{
		"-i", e.keyPath,
		"-o", "StrictHostKeyChecking=no",
		"-o", "UserKnownHostsFile=/dev/null",
		"-o", "LogLevel=ERROR",
		fmt.Sprintf("%s@%s", e.user, instanceIP),
		command,
	}

	// Execute script with context
	cmd := exec.CommandContext(ctx, "ssh", sshArgs...)

	// Capture output
	stdout, err := cmd.Output()
	result := &ExecutionResult{
		Duration: time.Since(startTime),
		Stdout:   strings.TrimSpace(string(stdout)),
	}

	if err != nil {
		// Handle exit code errors
		if exitError, ok := err.(*exec.ExitError); ok {
			result.ExitCode = exitError.ExitCode()
			result.Stderr = strings.TrimSpace(string(exitError.Stderr))
		} else {
			return nil, fmt.Errorf("SSH script execution failed: %w", err)
		}
	} else {
		result.ExitCode = 0
	}

	return result, nil
}

// CopyFile copies a file from local to remote instance via SCP
func (e *SSHRemoteExecutor) CopyFile(ctx context.Context, instanceName string, localPath, remotePath string) error {
	// Get instance IP
	instanceIP, err := e.getInstanceIP(instanceName)
	if err != nil {
		return fmt.Errorf("failed to get instance IP: %w", err)
	}

	// Build SCP command
	scpArgs := []string{
		"-i", e.keyPath,
		"-o", "StrictHostKeyChecking=no",
		"-o", "UserKnownHostsFile=/dev/null",
		"-o", "LogLevel=ERROR",
		localPath,
		fmt.Sprintf("%s@%s:%s", e.user, instanceIP, remotePath),
	}

	// Execute SCP with context
	cmd := exec.CommandContext(ctx, "scp", scpArgs...)

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("SCP copy failed: %w", err)
	}

	return nil
}

// GetFile copies a file from remote instance to local via SCP
func (e *SSHRemoteExecutor) GetFile(ctx context.Context, instanceName string, remotePath, localPath string) error {
	// Get instance IP
	instanceIP, err := e.getInstanceIP(instanceName)
	if err != nil {
		return fmt.Errorf("failed to get instance IP: %w", err)
	}

	// Build SCP command
	scpArgs := []string{
		"-i", e.keyPath,
		"-o", "StrictHostKeyChecking=no",
		"-o", "UserKnownHostsFile=/dev/null",
		"-o", "LogLevel=ERROR",
		fmt.Sprintf("%s@%s:%s", e.user, instanceIP, remotePath),
		localPath,
	}

	// Execute SCP with context
	cmd := exec.CommandContext(ctx, "scp", scpArgs...)

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("SCP download failed: %w", err)
	}

	return nil
}

// getInstanceIP gets the IP address for an instance from CloudWorkstation state
func (e *SSHRemoteExecutor) getInstanceIP(instanceName string) (string, error) {
	// Query CloudWorkstation state management for instance IP
	// This integrates with pkg/state.Manager to get instance metadata
	//
	// Real implementation would be:
	//
	// import (
	//     "github.com/scttfrdmn/cloudworkstation/pkg/state"
	//     "github.com/scttfrdmn/cloudworkstation/pkg/types"
	// )
	//
	// stateManager, err := state.NewManager()
	// if err != nil {
	//     return "", fmt.Errorf("failed to initialize state manager: %w", err)
	// }
	//
	// currentState, err := stateManager.LoadState()
	// if err != nil {
	//     return "", fmt.Errorf("failed to load state: %w", err)
	// }
	//
	// instance, exists := currentState.Instances[instanceName]
	// if !exists {
	//     return "", fmt.Errorf("instance %s not found in state", instanceName)
	// }
	//
	// if instance.PublicIP == "" {
	//     return "", fmt.Errorf("instance %s has no public IP", instanceName)
	// }
	//
	// return instance.PublicIP, nil

	// For SSHRemoteExecutor to work, it needs state manager dependency injection
	// Update constructor to: NewSSHRemoteExecutor(keyPath, user, stateManager)
	return "", fmt.Errorf("instance IP lookup requires state manager integration - update SSHRemoteExecutor constructor to inject state.Manager dependency")
}

// SystemsManagerExecutor implements RemoteExecutor using AWS Systems Manager
type SystemsManagerExecutor struct {
	region string
}

// NewSystemsManagerExecutor creates a new Systems Manager-based remote executor
func NewSystemsManagerExecutor(region string) *SystemsManagerExecutor {
	return &SystemsManagerExecutor{
		region: region,
	}
}

// Execute executes a command via AWS Systems Manager Run Command
func (e *SystemsManagerExecutor) Execute(ctx context.Context, instanceName string, command string) (*ExecutionResult, error) {
	// This would integrate with AWS Systems Manager to execute commands
	// without requiring SSH access. This is particularly useful for
	// instances in private subnets or with restricted security groups.

	// Example implementation using AWS SDK:
	// instanceID, err := e.getInstanceID(instanceName)
	// if err != nil {
	//     return nil, err
	// }
	//
	// sess := session.Must(session.NewSession(&aws.Config{
	//     Region: aws.String(e.region),
	// }))
	//
	// ssmClient := ssm.New(sess)
	//
	// input := &ssm.SendCommandInput{
	//     DocumentName: aws.String("AWS-RunShellScript"),
	//     InstanceIds: []*string{aws.String(instanceID)},
	//     Parameters: map[string][]*string{
	//         "commands": {aws.String(command)},
	//     },
	// }
	//
	// result, err := ssmClient.SendCommandWithContext(ctx, input)
	// if err != nil {
	//     return nil, err
	// }
	//
	// // Wait for command to complete and get output
	// commandID := *result.Command.CommandId
	// return e.waitForCommandCompletion(ctx, ssmClient, commandID, instanceID)

	return nil, fmt.Errorf("systems Manager executor not implemented (placeholder)")
}

// ExecuteScript executes a script via AWS Systems Manager
func (e *SystemsManagerExecutor) ExecuteScript(ctx context.Context, instanceName string, script string) (*ExecutionResult, error) {
	// Similar to Execute but handles multi-line scripts
	return e.Execute(ctx, instanceName, script)
}

// CopyFile copies a file using S3 as intermediate storage for Systems Manager
func (e *SystemsManagerExecutor) CopyFile(ctx context.Context, instanceName string, localPath, remotePath string) error {
	// Systems Manager doesn't have direct file copy, so this would:
	// 1. Upload file to S3
	// 2. Use Systems Manager to download from S3 to instance
	// 3. Clean up S3 object

	return fmt.Errorf("systems Manager file copy not implemented (placeholder)")
}

// GetFile gets a file using S3 as intermediate storage for Systems Manager
func (e *SystemsManagerExecutor) GetFile(ctx context.Context, instanceName string, remotePath, localPath string) error {
	// Systems Manager doesn't have direct file copy, so this would:
	// 1. Use Systems Manager to upload file from instance to S3
	// 2. Download file from S3 to local
	// 3. Clean up S3 object

	return fmt.Errorf("systems Manager file download not implemented (placeholder)")
}

// MockRemoteExecutor implements RemoteExecutor for testing
type MockRemoteExecutor struct {
	commands []string
	results  map[string]*ExecutionResult
}

// NewMockRemoteExecutor creates a new mock executor for testing
func NewMockRemoteExecutor() *MockRemoteExecutor {
	return &MockRemoteExecutor{
		commands: []string{},
		results:  make(map[string]*ExecutionResult),
	}
}

// SetResult sets the expected result for a command
func (e *MockRemoteExecutor) SetResult(command string, result *ExecutionResult) {
	e.results[command] = result
}

// Execute records the command and returns a predefined result
func (e *MockRemoteExecutor) Execute(ctx context.Context, instanceName string, command string) (*ExecutionResult, error) {
	e.commands = append(e.commands, command)

	if result, exists := e.results[command]; exists {
		return result, nil
	}

	// Default success result
	return &ExecutionResult{
		ExitCode: 0,
		Stdout:   "",
		Stderr:   "",
		Duration: time.Millisecond * 100,
	}, nil
}

// ExecuteScript records the script and returns a predefined result
func (e *MockRemoteExecutor) ExecuteScript(ctx context.Context, instanceName string, script string) (*ExecutionResult, error) {
	return e.Execute(ctx, instanceName, script)
}

// CopyFile records the file copy operation
func (e *MockRemoteExecutor) CopyFile(ctx context.Context, instanceName string, localPath, remotePath string) error {
	e.commands = append(e.commands, fmt.Sprintf("copy %s -> %s", localPath, remotePath))
	return nil
}

// GetFile records the file download operation
func (e *MockRemoteExecutor) GetFile(ctx context.Context, instanceName string, remotePath, localPath string) error {
	e.commands = append(e.commands, fmt.Sprintf("download %s -> %s", remotePath, localPath))
	return nil
}

// GetCommands returns all recorded commands for testing verification
func (e *MockRemoteExecutor) GetCommands() []string {
	return e.commands
}
