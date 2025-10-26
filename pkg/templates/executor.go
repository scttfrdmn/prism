// Package templates provides remote execution capabilities for template application.
package templates

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
	"github.com/scttfrdmn/prism/pkg/types"
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

// getInstanceIP gets the IP address for an instance from Prism state
func (e *SSHRemoteExecutor) getInstanceIP(instanceName string) (string, error) {
	// Query Prism state management for instance IP
	// This integrates with pkg/state.Manager to get instance metadata
	//
	// Real implementation would be:
	//
	// import (
	//     "github.com/scttfrdmn/prism/pkg/state"
	//     "github.com/scttfrdmn/prism/pkg/types"
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
	region       string
	ssmClient    SSMClientInterface
	s3Client     S3ClientInterface
	s3Bucket     string // S3 bucket for temporary file storage
	stateManager StateManager
}

// SSMClientInterface defines the interface for SSM client operations needed by executor
type SSMClientInterface interface {
	SendCommand(ctx context.Context, params *ssm.SendCommandInput, optFns ...func(*ssm.Options)) (*ssm.SendCommandOutput, error)
	GetCommandInvocation(ctx context.Context, params *ssm.GetCommandInvocationInput, optFns ...func(*ssm.Options)) (*ssm.GetCommandInvocationOutput, error)
}

// S3ClientInterface defines the interface for S3 client operations needed by executor
type S3ClientInterface interface {
	PutObject(ctx context.Context, params *s3.PutObjectInput, optFns ...func(*s3.Options)) (*s3.PutObjectOutput, error)
	GetObject(ctx context.Context, params *s3.GetObjectInput, optFns ...func(*s3.Options)) (*s3.GetObjectOutput, error)
	DeleteObject(ctx context.Context, params *s3.DeleteObjectInput, optFns ...func(*s3.Options)) (*s3.DeleteObjectOutput, error)
}

// StateManager defines the interface for state management operations needed by executor
type StateManager interface {
	LoadState() (*types.State, error)
}

// NewSystemsManagerExecutor creates a new Systems Manager-based remote executor
func NewSystemsManagerExecutor(region string, ssmClient SSMClientInterface, s3Client S3ClientInterface, s3Bucket string, stateManager StateManager) *SystemsManagerExecutor {
	return &SystemsManagerExecutor{
		region:       region,
		ssmClient:    ssmClient,
		s3Client:     s3Client,
		s3Bucket:     s3Bucket,
		stateManager: stateManager,
	}
}

// getInstanceID retrieves the instance ID for a given instance name from state manager
func (e *SystemsManagerExecutor) getInstanceID(instanceName string) (string, error) {
	state, err := e.stateManager.LoadState()
	if err != nil {
		return "", fmt.Errorf("failed to load state: %w", err)
	}

	instance, exists := state.Instances[instanceName]
	if !exists {
		return "", fmt.Errorf("instance %s not found in state", instanceName)
	}

	if instance.ID == "" {
		return "", fmt.Errorf("instance %s has no instance ID", instanceName)
	}

	return instance.ID, nil
}

// waitForCommandCompletion waits for an SSM command to complete and returns the result
func (e *SystemsManagerExecutor) waitForCommandCompletion(ctx context.Context, commandID, instanceID string) (*ExecutionResult, error) {
	maxWait := 610 // 610 seconds (slightly more than command timeout of 600s)

	for i := 0; i < maxWait; i++ {
		// Check if context is cancelled
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("context cancelled while waiting for command completion: %w", ctx.Err())
		default:
			// Continue
		}

		getInput := &ssm.GetCommandInvocationInput{
			CommandId:  aws.String(commandID),
			InstanceId: aws.String(instanceID),
		}

		getOutput, err := e.ssmClient.GetCommandInvocation(ctx, getInput)
		if err != nil {
			// Command not ready yet, wait and retry
			time.Sleep(1 * time.Second)
			continue
		}

		// Check command status
		status := string(getOutput.Status)
		if status == "Success" || status == "Failed" || status == "Cancelled" || status == "TimedOut" {
			// Command completed
			exitCode := int(getOutput.ResponseCode)
			stdout := ""
			stderr := ""

			if getOutput.StandardOutputContent != nil {
				stdout = *getOutput.StandardOutputContent
			}
			if getOutput.StandardErrorContent != nil {
				stderr = *getOutput.StandardErrorContent
			}

			result := &ExecutionResult{
				ExitCode: exitCode,
				Stdout:   stdout,
				Stderr:   stderr,
			}

			if status != "Success" {
				return result, fmt.Errorf("command failed with status %s (exit code %d)", status, exitCode)
			}

			return result, nil
		}

		// Still running, wait and retry
		time.Sleep(1 * time.Second)
	}

	return nil, fmt.Errorf("command timed out after %d seconds", maxWait)
}

// Execute executes a command via AWS Systems Manager Run Command
func (e *SystemsManagerExecutor) Execute(ctx context.Context, instanceName string, command string) (*ExecutionResult, error) {
	// Check if SSM client is configured
	if e.ssmClient == nil {
		return nil, fmt.Errorf("SSM client not configured - cannot execute commands via Systems Manager")
	}

	// Get instance ID from name via state manager
	instanceID, err := e.getInstanceID(instanceName)
	if err != nil {
		return nil, fmt.Errorf("failed to get instance ID for %s: %w", instanceName, err)
	}

	// Send command via Systems Manager
	sendInput := &ssm.SendCommandInput{
		DocumentName: aws.String("AWS-RunShellScript"),
		InstanceIds:  []string{instanceID},
		Parameters: map[string][]string{
			"commands": {command},
		},
		TimeoutSeconds: aws.Int32(600), // 10 minute timeout
	}

	sendOutput, err := e.ssmClient.SendCommand(ctx, sendInput)
	if err != nil {
		return nil, fmt.Errorf("failed to send SSM command: %w", err)
	}

	commandID := *sendOutput.Command.CommandId

	// Wait for command to complete
	return e.waitForCommandCompletion(ctx, commandID, instanceID)
}

// ExecuteScript executes a script via AWS Systems Manager
func (e *SystemsManagerExecutor) ExecuteScript(ctx context.Context, instanceName string, script string) (*ExecutionResult, error) {
	// Similar to Execute but handles multi-line scripts
	return e.Execute(ctx, instanceName, script)
}

// CopyFile copies a file using S3 as intermediate storage for Systems Manager
func (e *SystemsManagerExecutor) CopyFile(ctx context.Context, instanceName string, localPath, remotePath string) error {
	// Check if S3 client is configured
	if e.s3Client == nil || e.s3Bucket == "" {
		return fmt.Errorf("S3 client or bucket not configured - cannot copy files via Systems Manager (requires S3 intermediate storage)")
	}

	// Systems Manager doesn't have direct file copy, so we use S3 as intermediate storage:
	// 1. Upload file to S3
	// 2. Use Systems Manager to download from S3 to instance
	// 3. Clean up S3 object

	// Read local file
	fileData, err := os.ReadFile(localPath)
	if err != nil {
		return fmt.Errorf("failed to read local file %s: %w", localPath, err)
	}

	// Generate unique S3 key for temporary storage
	s3Key := fmt.Sprintf("cloudworkstation/tmp/%s/%s", instanceName, filepath.Base(localPath))

	// Upload to S3
	putInput := &s3.PutObjectInput{
		Bucket: aws.String(e.s3Bucket),
		Key:    aws.String(s3Key),
		Body:   bytes.NewReader(fileData),
	}

	_, err = e.s3Client.PutObject(ctx, putInput)
	if err != nil {
		return fmt.Errorf("failed to upload file to S3: %w", err)
	}

	// Download from S3 to instance using SSM
	downloadCommand := fmt.Sprintf("aws s3 cp s3://%s/%s %s", e.s3Bucket, s3Key, remotePath)
	result, err := e.Execute(ctx, instanceName, downloadCommand)
	if err != nil {
		// Clean up S3 object before returning error
		_ = e.cleanupS3Object(ctx, s3Key)
		return fmt.Errorf("failed to download file to instance: %w", err)
	}

	if result.ExitCode != 0 {
		// Clean up S3 object before returning error
		_ = e.cleanupS3Object(ctx, s3Key)
		return fmt.Errorf("download command failed with exit code %d: %s", result.ExitCode, result.Stderr)
	}

	// Clean up S3 object
	if err := e.cleanupS3Object(ctx, s3Key); err != nil {
		// Log warning but don't fail the operation
		fmt.Printf("Warning: failed to clean up temporary S3 object %s: %v\n", s3Key, err)
	}

	return nil
}

// GetFile gets a file using S3 as intermediate storage for Systems Manager
func (e *SystemsManagerExecutor) GetFile(ctx context.Context, instanceName string, remotePath, localPath string) error {
	// Check if S3 client is configured
	if e.s3Client == nil || e.s3Bucket == "" {
		return fmt.Errorf("S3 client or bucket not configured - cannot get files via Systems Manager (requires S3 intermediate storage)")
	}

	// Systems Manager doesn't have direct file copy, so we use S3 as intermediate storage:
	// 1. Use Systems Manager to upload file from instance to S3
	// 2. Download file from S3 to local
	// 3. Clean up S3 object

	// Generate unique S3 key for temporary storage
	s3Key := fmt.Sprintf("cloudworkstation/tmp/%s/%s", instanceName, filepath.Base(remotePath))

	// Upload from instance to S3 using SSM
	uploadCommand := fmt.Sprintf("aws s3 cp %s s3://%s/%s", remotePath, e.s3Bucket, s3Key)
	result, err := e.Execute(ctx, instanceName, uploadCommand)
	if err != nil {
		return fmt.Errorf("failed to upload file from instance: %w", err)
	}

	if result.ExitCode != 0 {
		return fmt.Errorf("upload command failed with exit code %d: %s", result.ExitCode, result.Stderr)
	}

	// Download from S3 to local
	getInput := &s3.GetObjectInput{
		Bucket: aws.String(e.s3Bucket),
		Key:    aws.String(s3Key),
	}

	getOutput, err := e.s3Client.GetObject(ctx, getInput)
	if err != nil {
		// Clean up S3 object before returning error
		_ = e.cleanupS3Object(ctx, s3Key)
		return fmt.Errorf("failed to download file from S3: %w", err)
	}
	defer getOutput.Body.Close()

	// Read S3 object body
	fileData, err := io.ReadAll(getOutput.Body)
	if err != nil {
		// Clean up S3 object before returning error
		_ = e.cleanupS3Object(ctx, s3Key)
		return fmt.Errorf("failed to read S3 object body: %w", err)
	}

	// Write to local file
	if err := os.WriteFile(localPath, fileData, 0644); err != nil {
		// Clean up S3 object before returning error
		_ = e.cleanupS3Object(ctx, s3Key)
		return fmt.Errorf("failed to write local file %s: %w", localPath, err)
	}

	// Clean up S3 object
	if err := e.cleanupS3Object(ctx, s3Key); err != nil {
		// Log warning but don't fail the operation
		fmt.Printf("Warning: failed to clean up temporary S3 object %s: %v\n", s3Key, err)
	}

	return nil
}

// cleanupS3Object removes a temporary S3 object
func (e *SystemsManagerExecutor) cleanupS3Object(ctx context.Context, s3Key string) error {
	deleteInput := &s3.DeleteObjectInput{
		Bucket: aws.String(e.s3Bucket),
		Key:    aws.String(s3Key),
	}

	_, err := e.s3Client.DeleteObject(ctx, deleteInput)
	return err
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
