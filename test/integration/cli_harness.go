package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// CLITestContext extends TestContext with CLI binary execution capabilities
type CLITestContext struct {
	*TestContext
	PrismBin   string
	ConfigDir  string
	StateDir   string
	OutputLogs bool // If true, logs all CLI output to test logs
}

// CLIResult captures the result of a CLI command execution
type CLIResult struct {
	ExitCode int
	Stdout   string
	Stderr   string
	Duration time.Duration
	Err      error
}

// NewCLITestContext creates a test context with CLI capabilities
func NewCLITestContext(t *testing.T) *CLITestContext {
	base := NewTestContext(t)

	ctx := &CLITestContext{
		TestContext: base,
		OutputLogs:  true, // Enable by default for debugging
	}

	// Find prism CLI binary
	ctx.PrismBin = ctx.findBinary("prism")
	if ctx.PrismBin == "" {
		t.Fatal("Prism CLI binary 'prism' not found. Run 'make build' first.")
	}
	t.Logf("Found prism binary: %s", ctx.PrismBin)

	// Create temporary config directory for isolated test environment
	tempConfigDir, err := os.MkdirTemp("", "prism-test-config-*")
	if err != nil {
		t.Fatalf("Failed to create temp config dir: %v", err)
	}
	ctx.ConfigDir = tempConfigDir
	t.Logf("Using test config directory: %s", tempConfigDir)

	// Create temporary state directory (matches daemon's state isolation)
	tempStateDir, err := os.MkdirTemp("", "prism-test-cli-state-*")
	if err != nil {
		t.Fatalf("Failed to create temp state dir: %v", err)
	}
	ctx.StateDir = tempStateDir
	t.Logf("Using test state directory: %s", tempStateDir)

	// Add cleanup for directories
	ctx.AddCleanup(func() {
		for _, dir := range []string{tempConfigDir, tempStateDir} {
			if err := os.RemoveAll(dir); err != nil {
				t.Logf("Warning: failed to remove %s: %v", dir, err)
			}
		}
		t.Log("Cleaned up CLI test directories")
	})

	return ctx
}

// Prism executes the prism CLI with given arguments
// Returns CLIResult with exit code, stdout, stderr, and any execution errors
func (ctx *CLITestContext) Prism(args ...string) *CLIResult {
	start := time.Now()

	// Prepare command
	cmd := exec.Command(ctx.PrismBin, args...) //nolint:gosec // G204: test harness runs the locally built prism binary with test-supplied args (no shell)

	// Set environment with test configuration
	env := os.Environ()
	env = append(env,
		fmt.Sprintf("AWS_PROFILE=%s", TestAWSProfile),
		fmt.Sprintf("AWS_REGION=%s", TestAWSRegion),
		fmt.Sprintf("PRISM_CONFIG_DIR=%s", ctx.ConfigDir),
		fmt.Sprintf("PRISM_STATE_DIR=%s", ctx.StateDir),
		fmt.Sprintf("PRISM_DAEMON_URL=%s", DaemonURL),
		"CWS_DAEMON_AUTO_START_DISABLE=1", // Prevent CLI from auto-starting its own daemon
	)
	cmd.Env = env

	// Capture output
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// Execute command
	err := cmd.Run()
	duration := time.Since(start)

	result := &CLIResult{
		ExitCode: 0,
		Stdout:   stdout.String(),
		Stderr:   stderr.String(),
		Duration: duration,
		Err:      err,
	}

	// Extract exit code from error
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			result.ExitCode = exitErr.ExitCode()
		} else {
			result.ExitCode = 1 // Generic failure
		}
	}

	// Log output if enabled
	if ctx.OutputLogs {
		ctx.T.Logf("CLI: prism %s", strings.Join(args, " "))
		ctx.T.Logf("Exit code: %d, Duration: %s", result.ExitCode, duration)
		if result.Stdout != "" {
			ctx.T.Logf("Stdout:\n%s", result.Stdout)
		}
		if result.Stderr != "" {
			ctx.T.Logf("Stderr:\n%s", result.Stderr)
		}
	}

	return result
}

// PrismQuiet executes prism CLI without logging output (for programmatic parsing)
func (ctx *CLITestContext) PrismQuiet(args ...string) *CLIResult {
	oldLogs := ctx.OutputLogs
	ctx.OutputLogs = false
	result := ctx.Prism(args...)
	ctx.OutputLogs = oldLogs
	return result
}

// PrismJSON executes prism CLI expecting JSON output and unmarshals into target
func (ctx *CLITestContext) PrismJSON(target interface{}, args ...string) error {
	// Ensure --json flag is present
	hasJSON := false
	for _, arg := range args {
		if arg == "--json" || arg == "-j" {
			hasJSON = true
			break
		}
	}
	if !hasJSON {
		args = append(args, "--json")
	}

	result := ctx.PrismQuiet(args...)
	if result.ExitCode != 0 {
		return fmt.Errorf("command failed with exit code %d: %s", result.ExitCode, result.Stderr)
	}

	if err := json.Unmarshal([]byte(result.Stdout), target); err != nil {
		return fmt.Errorf("failed to parse JSON output: %w\nOutput: %s", err, result.Stdout)
	}

	return nil
}

// AssertSuccess fails test if CLI command did not succeed (exit code 0)
func (r *CLIResult) AssertSuccess(t *testing.T, msg string) {
	t.Helper()
	if r.ExitCode != 0 {
		t.Fatalf("%s: command failed with exit code %d\nStderr: %s\nStdout: %s",
			msg, r.ExitCode, r.Stderr, r.Stdout)
	}
}

// AssertFailure fails test if CLI command succeeded (expected non-zero exit code)
func (r *CLIResult) AssertFailure(t *testing.T, msg string) {
	t.Helper()
	if r.ExitCode == 0 {
		t.Fatalf("%s: command succeeded but was expected to fail\nStdout: %s",
			msg, r.Stdout)
	}
}

// AssertContains fails test if output does not contain expected string
func (r *CLIResult) AssertContains(t *testing.T, expected, msg string) {
	t.Helper()
	combined := r.Stdout + r.Stderr
	if !strings.Contains(combined, expected) {
		t.Fatalf("%s: output does not contain '%s'\nOutput: %s", msg, expected, combined)
	}
}

// AssertNotContains fails test if output contains unexpected string
func (r *CLIResult) AssertNotContains(t *testing.T, unexpected, msg string) {
	t.Helper()
	combined := r.Stdout + r.Stderr
	if strings.Contains(combined, unexpected) {
		t.Fatalf("%s: output contains unexpected '%s'\nOutput: %s", msg, unexpected, combined)
	}
}

// LaunchInstanceCLI launches an instance using the prism CLI (not API)
func (ctx *CLITestContext) LaunchInstanceCLI(templateSlug, instanceName, size string) (*CLIResult, error) {
	ctx.T.Logf("Launching instance '%s' with template '%s' (size: %s) via CLI...",
		instanceName, templateSlug, size)

	// Track for cleanup
	ctx.TrackInstance(instanceName)

	// Build command: prism workspace launch <template> <name> --size <size>
	args := []string{"workspace", "launch", templateSlug, instanceName}
	if size != "" {
		args = append(args, "--size", size)
	}

	result := ctx.Prism(args...)
	if result.ExitCode != 0 {
		return result, fmt.Errorf("launch command failed: %s", result.Stderr)
	}

	// Wait for instance to be running (using API client for verification)
	ctx.T.Logf("Waiting for instance '%s' to be running...", instanceName)
	if err := ctx.WaitForInstanceState(instanceName, "running", InstanceReadyTimeout); err != nil {
		return result, fmt.Errorf("instance failed to reach running state: %w", err)
	}

	return result, nil
}

// ListInstancesCLI lists instances using the prism CLI
func (ctx *CLITestContext) ListInstancesCLI() ([]string, error) {
	result := ctx.PrismQuiet("workspace", "list")
	if result.ExitCode != 0 {
		return nil, fmt.Errorf("list command failed: %s", result.Stderr)
	}

	// Parse instance names from output (basic parsing, expects one per line with name column)
	var names []string
	lines := strings.Split(result.Stdout, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "NAME") || strings.HasPrefix(line, "---") {
			continue // Skip header and separator lines
		}
		// Extract first field (name)
		fields := strings.Fields(line)
		if len(fields) > 0 {
			names = append(names, fields[0])
		}
	}

	return names, nil
}

// StopInstanceCLI stops an instance using the prism CLI
func (ctx *CLITestContext) StopInstanceCLI(name string) error {
	ctx.T.Logf("Stopping instance '%s' via CLI...", name)

	// Use --wait flag to ensure command doesn't return until instance is stopped
	result := ctx.Prism("workspace", "stop", "--wait", name)
	if result.ExitCode != 0 {
		return fmt.Errorf("stop command failed: %s", result.Stderr)
	}

	return nil
}

// DeleteInstanceCLI deletes an instance using the prism CLI
func (ctx *CLITestContext) DeleteInstanceCLI(name string) error {
	ctx.T.Logf("Deleting instance '%s' via CLI...", name)

	// Use --wait flag to ensure command doesn't return until instance is terminated
	result := ctx.Prism("workspace", "delete", "--wait", name)
	if result.ExitCode != 0 {
		return fmt.Errorf("delete command failed: %s", result.Stderr)
	}

	// Remove from tracking
	for i, tracked := range ctx.InstanceNames {
		if tracked == name {
			ctx.InstanceNames = append(ctx.InstanceNames[:i], ctx.InstanceNames[i+1:]...)
			break
		}
	}

	return nil
}

// VerifyInstanceInAWS verifies instance actually exists in AWS (not just in state)
func (ctx *CLITestContext) VerifyInstanceInAWS(name string) error {
	// Use API client to get actual AWS instance details
	instance, err := ctx.Client.GetInstance(context.Background(), name)
	if err != nil {
		return fmt.Errorf("instance not found in AWS: %w", err)
	}

	if instance.ID == "" {
		return fmt.Errorf("instance has no AWS ID")
	}

	ctx.T.Logf("Verified instance '%s' exists in AWS (ID: %s, State: %s)",
		name, instance.ID, instance.State)

	return nil
}

// findBinary reuses the base TestContext method to locate binaries
// This is inherited from TestContext but documented here for clarity
// Searches: bin/, ../../bin/, ../../../bin/
func (ctx *CLITestContext) findBinary(name string) string {
	// Try relative to project root
	paths := []string{
		filepath.Join("bin", name),
		filepath.Join("..", "..", "bin", name),
		filepath.Join("../../bin", name),
		filepath.Join("../../../bin", name),
	}

	for _, path := range paths {
		if _, err := os.Stat(path); err == nil {
			abs, _ := filepath.Abs(path)
			return abs
		}
	}

	return ""
}
