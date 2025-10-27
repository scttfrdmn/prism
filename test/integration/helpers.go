package integration

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/scttfrdmn/prism/pkg/api/client"
	"github.com/scttfrdmn/prism/pkg/types"
)

const (
	// Test configuration
	TestAWSProfile = "aws"
	TestAWSRegion  = "us-west-2"
	DaemonURL      = "http://localhost:8947"

	// Timeouts
	DaemonStartTimeout    = 30 * time.Second
	InstanceReadyTimeout  = 10 * time.Minute
	InstanceDeleteTimeout = 5 * time.Minute
	PollInterval          = 10 * time.Second
)

// TestContext holds common test resources
type TestContext struct {
	T             *testing.T
	Client        client.PrismAPI
	DaemonCmd     *exec.Cmd
	CleanupFuncs  []func()
	InstanceNames []string
	VolumeNames   []string
}

// NewTestContext creates a new test context with daemon and API client
func NewTestContext(t *testing.T) *TestContext {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := &TestContext{
		T:             t,
		CleanupFuncs:  make([]func(), 0),
		InstanceNames: make([]string, 0),
		VolumeNames:   make([]string, 0),
	}

	// Start daemon
	ctx.StartDaemon()

	// Create API client
	ctx.Client = client.NewClientWithOptions(DaemonURL, client.Options{
		AWSProfile: TestAWSProfile,
		AWSRegion:  TestAWSRegion,
	})

	// Verify connectivity
	if err := ctx.Client.Ping(context.Background()); err != nil {
		t.Fatalf("Failed to connect to daemon: %v", err)
	}

	t.Logf("Test context initialized (profile=%s, region=%s)", TestAWSProfile, TestAWSRegion)

	return ctx
}

// StartDaemon starts the daemon process for testing
func (ctx *TestContext) StartDaemon() {
	// Find daemon binary
	daemonPath := ctx.findBinary("prismd")
	if daemonPath == "" {
		ctx.T.Fatal("Daemon binary 'prismd' not found. Run 'make build' first.")
	}

	// Create temporary state directory for isolated test environment
	tempStateDir, err := os.MkdirTemp("", "prism-test-state-*")
	if err != nil {
		ctx.T.Fatalf("Failed to create temp state dir: %v", err)
	}
	ctx.T.Logf("Using test state directory: %s", tempStateDir)

	// Add cleanup for state directory
	ctx.AddCleanup(func() {
		os.RemoveAll(tempStateDir)
		ctx.T.Logf("Cleaned up test state directory")
	})

	// Start daemon with isolated state directory
	cmd := exec.Command(daemonPath)
	// Set environment to use test state directory (matches user experience with custom state location)
	env := os.Environ()
	env = append(env, fmt.Sprintf("PRISM_STATE_DIR=%s", tempStateDir))
	cmd.Env = env
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		ctx.T.Fatalf("Failed to start daemon: %v", err)
	}

	ctx.DaemonCmd = cmd
	ctx.T.Logf("Daemon started (PID: %d)", cmd.Process.Pid)

	// Add cleanup
	ctx.AddCleanup(func() {
		if cmd.Process != nil {
			ctx.T.Log("Stopping daemon...")
			cmd.Process.Kill()
			cmd.Wait()
		}
	})

	// Wait for daemon to be ready
	ctx.waitForDaemon()
}

// waitForDaemon waits for daemon to be ready to accept connections
func (ctx *TestContext) waitForDaemon() {
	deadline := time.Now().Add(DaemonStartTimeout)
	testClient := client.NewClient(DaemonURL)

	for time.Now().Before(deadline) {
		if err := testClient.Ping(context.Background()); err == nil {
			ctx.T.Log("Daemon ready")
			return
		}
		time.Sleep(500 * time.Millisecond)
	}

	ctx.T.Fatal("Daemon failed to start within timeout")
}

// findBinary locates a binary in bin/ directory
func (ctx *TestContext) findBinary(name string) string {
	// Try relative to project root
	paths := []string{
		filepath.Join("bin", name),
		filepath.Join("..", "..", "bin", name),
		filepath.Join("../../bin", name),
	}

	for _, path := range paths {
		if _, err := os.Stat(path); err == nil {
			abs, _ := filepath.Abs(path)
			return abs
		}
	}

	return ""
}

// AddCleanup adds a cleanup function to be called at test end
func (ctx *TestContext) AddCleanup(fn func()) {
	ctx.CleanupFuncs = append(ctx.CleanupFuncs, fn)
}

// Cleanup runs all cleanup functions in reverse order
func (ctx *TestContext) Cleanup() {
	ctx.T.Log("Running cleanup...")

	// Delete all tracked instances
	for _, name := range ctx.InstanceNames {
		ctx.T.Logf("Cleaning up instance: %s", name)
		ctx.Client.DeleteInstance(context.Background(), name) // Best-effort cleanup
	}

	// Delete all tracked volumes
	for _, name := range ctx.VolumeNames {
		ctx.T.Logf("Cleaning up volume: %s", name)
		ctx.Client.DeleteVolume(context.Background(), name) // Best-effort cleanup
	}

	// Run custom cleanup functions
	for i := len(ctx.CleanupFuncs) - 1; i >= 0; i-- {
		ctx.CleanupFuncs[i]()
	}
}

// TrackInstance adds an instance name to cleanup list
func (ctx *TestContext) TrackInstance(name string) {
	ctx.InstanceNames = append(ctx.InstanceNames, name)
}

// TrackVolume adds a volume name to cleanup list
func (ctx *TestContext) TrackVolume(name string) {
	ctx.VolumeNames = append(ctx.VolumeNames, name)
}

// WaitForInstanceState waits for instance to reach desired state
func (ctx *TestContext) WaitForInstanceState(name, desiredState string, timeout time.Duration) error {
	ctx.T.Logf("Waiting for instance '%s' to reach state '%s'...", name, desiredState)
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		instance, err := ctx.Client.GetInstance(context.Background(), name)
		if err != nil {
			return fmt.Errorf("failed to get instance: %w", err)
		}

		ctx.T.Logf("Instance '%s' current state: %s", name, instance.State)

		if instance.State == desiredState {
			ctx.T.Logf("Instance '%s' reached state '%s'", name, desiredState)
			return nil
		}

		time.Sleep(PollInterval)
	}

	return fmt.Errorf("timeout waiting for instance '%s' to reach state '%s'", name, desiredState)
}

// WaitForInstanceRunning waits for instance to be running and returns instance details
func (ctx *TestContext) WaitForInstanceRunning(name string) (*types.Instance, error) {
	if err := ctx.WaitForInstanceState(name, "running", InstanceReadyTimeout); err != nil {
		return nil, err
	}
	return ctx.Client.GetInstance(context.Background(), name)
}

// GenerateTestName creates a unique test resource name
func GenerateTestName(prefix string) string {
	return fmt.Sprintf("%s-%d", prefix, time.Now().Unix())
}

// AssertNoError fails test if error is not nil
func AssertNoError(t *testing.T, err error, msg string) {
	t.Helper()
	if err != nil {
		t.Fatalf("%s: %v", msg, err)
	}
}

// AssertEqual fails test if values are not equal
func AssertEqual(t *testing.T, expected, actual interface{}, msg string) {
	t.Helper()
	if expected != actual {
		t.Fatalf("%s: expected %v, got %v", msg, expected, actual)
	}
}

// AssertNotEmpty fails test if string is empty
func AssertNotEmpty(t *testing.T, value string, msg string) {
	t.Helper()
	if value == "" {
		t.Fatalf("%s: value is empty", msg)
	}
}

// AssertInstanceExists verifies instance can be retrieved
func (ctx *TestContext) AssertInstanceExists(name string) *types.Instance {
	instance, err := ctx.Client.GetInstance(context.Background(), name)
	AssertNoError(ctx.T, err, fmt.Sprintf("Instance '%s' should exist", name))
	AssertNotEmpty(ctx.T, instance.ID, fmt.Sprintf("Instance '%s' should have ID", name))
	return instance
}

// AssertInstanceState verifies instance is in expected state
func (ctx *TestContext) AssertInstanceState(name, expectedState string) {
	instance := ctx.AssertInstanceExists(name)
	AssertEqual(ctx.T, expectedState, instance.State,
		fmt.Sprintf("Instance '%s' state", name))
}

// LaunchInstance launches an instance and waits for it to be ready
func (ctx *TestContext) LaunchInstance(templateSlug, instanceName, size string) (*types.Instance, error) {
	ctx.T.Logf("Launching instance '%s' with template '%s' (size: %s)...",
		instanceName, templateSlug, size)

	// Track for cleanup
	ctx.TrackInstance(instanceName)

	// Launch instance
	launchRequest := types.LaunchRequest{
		Template: templateSlug,
		Name:     instanceName,
		Size:     size,
	}

	_, err := ctx.Client.LaunchInstance(context.Background(), launchRequest)
	if err != nil {
		return nil, fmt.Errorf("failed to launch instance: %w", err)
	}

	// Wait for running state
	return ctx.WaitForInstanceRunning(instanceName)
}

// StopInstance stops an instance and waits for stopped state
func (ctx *TestContext) StopInstance(name string) error {
	ctx.T.Logf("Stopping instance '%s'...", name)

	if err := ctx.Client.StopInstance(context.Background(), name); err != nil {
		return fmt.Errorf("failed to stop instance: %w", err)
	}

	return ctx.WaitForInstanceState(name, "stopped", InstanceDeleteTimeout)
}

// HibernateInstance hibernates an instance
func (ctx *TestContext) HibernateInstance(name string) error {
	ctx.T.Logf("Hibernating instance '%s'...", name)

	if err := ctx.Client.HibernateInstance(context.Background(), name); err != nil {
		return fmt.Errorf("failed to hibernate instance: %w", err)
	}

	return ctx.WaitForInstanceState(name, "stopped", InstanceDeleteTimeout)
}

// StartInstance starts a stopped/hibernated instance
func (ctx *TestContext) StartInstance(name string) error {
	ctx.T.Logf("Starting instance '%s'...", name)

	if err := ctx.Client.StartInstance(context.Background(), name); err != nil {
		return fmt.Errorf("failed to start instance: %w", err)
	}

	return ctx.WaitForInstanceState(name, "running", InstanceReadyTimeout)
}

// DeleteInstance deletes an instance and removes from tracking
func (ctx *TestContext) DeleteInstance(name string) error {
	ctx.T.Logf("Deleting instance '%s'...", name)

	if err := ctx.Client.DeleteInstance(context.Background(), name); err != nil {
		return fmt.Errorf("failed to delete instance: %w", err)
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
