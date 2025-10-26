package daemon

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/scttfrdmn/prism/pkg/types"
)

// LaunchProgressMonitor monitors instance setup progress via SSH
type LaunchProgressMonitor struct {
	instance     *types.Instance
	sshKeyPath   string
	username     string
	stages       []ProgressStage
	pollInterval time.Duration
}

// ProgressStage represents a stage in the setup process
type ProgressStage struct {
	Name        string // Internal name: "system-packages"
	DisplayName string // User-facing: "Installing system packages"
	Status      string // "pending", "running", "complete", "error"
	StartTime   time.Time
	EndTime     time.Time
	Output      string // Last output line from this stage
}

// NewLaunchProgressMonitor creates a new progress monitor
func NewLaunchProgressMonitor(instance *types.Instance, sshKeyPath, username string) *LaunchProgressMonitor {
	return &LaunchProgressMonitor{
		instance:     instance,
		sshKeyPath:   sshKeyPath,
		username:     username,
		pollInterval: 10 * time.Second,
		stages: []ProgressStage{
			{Name: "init", DisplayName: "System initialization", Status: "pending"},
			{Name: "system-packages", DisplayName: "Installing system packages", Status: "pending"},
			{Name: "conda-packages", DisplayName: "Installing conda packages", Status: "pending"},
			{Name: "pip-packages", DisplayName: "Installing pip packages", Status: "pending"},
			{Name: "service-config", DisplayName: "Configuring services", Status: "pending"},
			{Name: "ready", DisplayName: "Starting services", Status: "pending"},
		},
	}
}

// CloudInitStatus represents cloud-init status
type CloudInitStatus struct {
	Status      string // "running", "done", "error", "disabled"
	Description string
	Errors      []string
}

// Start begins monitoring the instance setup progress
func (m *LaunchProgressMonitor) Start(ctx context.Context) error {
	// Wait for SSH to become available
	if err := m.waitForSSH(ctx); err != nil {
		return fmt.Errorf("SSH not available: %w", err)
	}

	// Poll for progress
	ticker := time.NewTicker(m.pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()

		case <-ticker.C:
			// Check cloud-init status
			status, err := m.checkCloudInitStatus(ctx)
			if err != nil {
				// If we can't check status, continue polling
				continue
			}

			// Parse progress markers if available
			if err := m.parseProgressMarkers(ctx); err != nil {
				// Progress markers not available yet, just use cloud-init status
			}

			// Check if setup is complete
			if status.Status == "done" {
				m.markAllComplete()
				return nil
			} else if status.Status == "error" {
				return fmt.Errorf("cloud-init failed: %s", status.Description)
			}
		}
	}
}

// waitForSSH waits for SSH to become available on the instance
func (m *LaunchProgressMonitor) waitForSSH(ctx context.Context) error {
	timeout := time.After(5 * time.Minute)
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-timeout:
			return fmt.Errorf("timeout waiting for SSH")
		case <-ticker.C:
			if m.isSSHAvailable(ctx) {
				return nil
			}
		}
	}
}

// isSSHAvailable checks if SSH is available
func (m *LaunchProgressMonitor) isSSHAvailable(ctx context.Context) bool {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "ssh",
		"-o", "ConnectTimeout=3",
		"-o", "StrictHostKeyChecking=no",
		"-o", "UserKnownHostsFile=/dev/null",
		"-o", "LogLevel=ERROR",
		"-i", m.sshKeyPath,
		fmt.Sprintf("%s@%s", m.username, m.instance.PublicIP),
		"echo ready",
	)

	output, err := cmd.CombinedOutput()
	return err == nil && strings.TrimSpace(string(output)) == "ready"
}

// checkCloudInitStatus checks the cloud-init status via SSH
func (m *LaunchProgressMonitor) checkCloudInitStatus(ctx context.Context) (*CloudInitStatus, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "ssh",
		"-o", "ConnectTimeout=5",
		"-o", "StrictHostKeyChecking=no",
		"-o", "UserKnownHostsFile=/dev/null",
		"-o", "LogLevel=ERROR",
		"-i", m.sshKeyPath,
		fmt.Sprintf("%s@%s", m.username, m.instance.PublicIP),
		"cloud-init status 2>/dev/null || echo 'status: unknown'",
	)

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to check cloud-init status: %w", err)
	}

	return parseCloudInitStatus(string(output)), nil
}

// parseCloudInitStatus parses cloud-init status output
func parseCloudInitStatus(output string) *CloudInitStatus {
	status := &CloudInitStatus{
		Status: "unknown",
	}

	lines := strings.Split(strings.TrimSpace(output), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "status:") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				status.Status = strings.TrimSpace(parts[1])
			}
		}
	}

	return status
}

// parseProgressMarkers parses progress markers from the setup log
func (m *LaunchProgressMonitor) parseProgressMarkers(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "ssh",
		"-o", "ConnectTimeout=5",
		"-o", "StrictHostKeyChecking=no",
		"-o", "UserKnownHostsFile=/dev/null",
		"-o", "LogLevel=ERROR",
		"-i", m.sshKeyPath,
		fmt.Sprintf("%s@%s", m.username, m.instance.PublicIP),
		"tail -100 /var/log/cws-setup.log 2>/dev/null | grep 'CWS-PROGRESS' || echo 'NOTREADY'",
	)

	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to read progress markers: %w", err)
	}

	outputStr := string(output)
	if strings.Contains(outputStr, "NOTREADY") {
		return fmt.Errorf("progress markers not available yet")
	}

	// Parse progress markers
	// Format: [CWS-PROGRESS] STAGE:stage-name:status[:message]
	// Example: [CWS-PROGRESS] STAGE:system-packages:START
	// Example: [CWS-PROGRESS] STAGE:system-packages:COMPLETE
	lines := strings.Split(outputStr, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if !strings.Contains(line, "[CWS-PROGRESS]") {
			continue
		}

		// Extract the marker content
		parts := strings.SplitN(line, "[CWS-PROGRESS]", 2)
		if len(parts) != 2 {
			continue
		}

		marker := strings.TrimSpace(parts[1])
		if !strings.HasPrefix(marker, "STAGE:") {
			continue
		}

		// Parse: STAGE:stage-name:status[:message]
		markerParts := strings.Split(strings.TrimPrefix(marker, "STAGE:"), ":")
		if len(markerParts) < 2 {
			continue
		}

		stageName := markerParts[0]
		stageStatus := markerParts[1]
		message := ""
		if len(markerParts) > 2 {
			message = strings.Join(markerParts[2:], ":")
		}

		// Update stage status
		for i := range m.stages {
			if m.stages[i].Name == stageName {
				switch stageStatus {
				case "START":
					m.stages[i].Status = "running"
					m.stages[i].StartTime = time.Now()
				case "COMPLETE":
					m.stages[i].Status = "complete"
					m.stages[i].EndTime = time.Now()
				case "ERROR":
					m.stages[i].Status = "error"
					m.stages[i].Output = message
				default:
					m.stages[i].Output = message
				}
				break
			}
		}
	}

	return nil
}

// markAllComplete marks all stages as complete
func (m *LaunchProgressMonitor) markAllComplete() {
	for i := range m.stages {
		if m.stages[i].Status != "complete" {
			m.stages[i].Status = "complete"
			m.stages[i].EndTime = time.Now()
		}
	}
}

// GetStages returns the current progress stages
func (m *LaunchProgressMonitor) GetStages() []ProgressStage {
	return m.stages
}

// GetProgress returns the overall progress percentage
func (m *LaunchProgressMonitor) GetProgress() float64 {
	complete := 0
	for _, stage := range m.stages {
		if stage.Status == "complete" {
			complete++
		}
	}
	return float64(complete) / float64(len(m.stages)) * 100
}
