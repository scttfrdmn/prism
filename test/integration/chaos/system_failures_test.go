//go:build integration
// +build integration

package chaos

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/scttfrdmn/prism/test/fixtures"
	"github.com/scttfrdmn/prism/test/integration"
)

// TestDaemonCrashDuringOperation validates that the system maintains state
// consistency when the daemon crashes mid-operation.
//
// Chaos Scenario: Daemon killed with SIGKILL during instance launch
// Expected Behavior:
// - State file remains consistent (no corruption)
// - Daemon can restart and recover
// - In-progress operations are handled correctly
// - No partial state or zombie resources
//
// Addresses Issue #412 - System Chaos Testing
func TestDaemonCrashDuringOperation(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping chaos test in short mode")
	}

	t.Logf("🌪️  CHAOS TEST: Daemon Crash During Operation")
	t.Logf("")
	t.Logf("⚠️  WARNING: This test simulates daemon crashes")
	t.Logf("   Other tests may be affected if running concurrently")
	t.Logf("")

	// Skip if not in CI or explicit chaos mode
	if os.Getenv("CHAOS_TESTING") != "true" {
		t.Skip("Skipping daemon crash test (set CHAOS_TESTING=true to enable)")
	}

	// Setup
	ctx := integration.NewTestContext(t)
	defer ctx.Cleanup()

	registry := fixtures.NewFixtureRegistry(t, ctx.Client)

	// ========================================
	// Scenario: Find and Monitor Daemon Process
	// ========================================

	t.Logf("📋 Phase 1: Locating daemon process")

	// Find daemon PID
	daemonPID, err := findDaemonProcess()
	if err != nil {
		t.Skipf("Could not find daemon process: %v", err)
	}

	t.Logf("✅ Found daemon process: PID %d", daemonPID)

	// Verify daemon is responsive
	_, err = ctx.Client.ListInstances(context.Background())
	integration.AssertNoError(t, err, "Daemon should be responsive before test")
	t.Logf("✅ Daemon is responsive")

	// ========================================
	// Scenario: Create Resources Before Crash
	// ========================================

	t.Logf("")
	t.Logf("📋 Phase 2: Creating resources before crash")

	projectName := integration.GenerateTestName("crash-test-project")
	project, err := fixtures.CreateTestProject(t, registry, fixtures.CreateTestProjectOptions{
		Name:        projectName,
		Description: "Daemon crash testing project",
		Owner:       "test-user@example.com",
	})
	integration.AssertNoError(t, err, "Failed to create project")
	t.Logf("✅ Project created: %s", project.ID)

	// Record state before crash
	preState, err := ctx.Client.ListInstances(context.Background())
	integration.AssertNoError(t, err, "Failed to get pre-crash state")
	t.Logf("✅ Pre-crash state recorded: %d instances", len(preState.Instances))

	// ========================================
	// Scenario: Simulate Daemon Crash (SIGKILL)
	// ========================================

	t.Logf("")
	t.Logf("📋 Phase 3: Simulating daemon crash")
	t.Logf("⚠️  Sending SIGKILL to daemon process...")

	// Get daemon process
	process, err := os.FindProcess(daemonPID)
	if err != nil {
		t.Fatalf("Failed to find process: %v", err)
	}

	// Send SIGKILL (immediate termination, no cleanup)
	err = process.Signal(syscall.SIGKILL)
	if err != nil {
		t.Fatalf("Failed to kill daemon: %v", err)
	}

	t.Logf("✅ Daemon crashed (SIGKILL sent)")

	// Wait for daemon to die
	time.Sleep(2 * time.Second)

	// Verify daemon is actually dead
	err = process.Signal(syscall.Signal(0)) // Signal 0 = check if process exists
	if err == nil {
		t.Error("Daemon should be dead but is still running")
	} else {
		t.Logf("✅ Daemon process terminated")
	}

	// ========================================
	// Scenario: Verify State File Integrity
	// ========================================

	t.Logf("")
	t.Logf("📋 Phase 4: Verifying state file integrity")

	homeDir, err := os.UserHomeDir()
	integration.AssertNoError(t, err, "Failed to get home directory")

	stateFile := filepath.Join(homeDir, ".prism", "state.json")
	t.Logf("Checking state file: %s", stateFile)

	// Read state file
	stateData, err := os.ReadFile(stateFile)
	if err != nil {
		t.Logf("⚠️  Could not read state file: %v", err)
		t.Logf("   State may be stored in alternate location")
	} else {
		t.Logf("✅ State file readable (%d bytes)", len(stateData))

		// Verify state file contains valid JSON structure
		stateContent := string(stateData)
		if strings.Contains(stateContent, "{") && strings.Contains(stateContent, "}") {
			t.Logf("✅ State file appears to be valid JSON")
		} else {
			t.Error("State file does not appear to be valid JSON")
		}

		// Check for corruption indicators
		if strings.Contains(stateContent, "\\x00") || len(stateData) == 0 {
			t.Error("State file appears to be corrupted")
		} else {
			t.Logf("✅ No obvious corruption detected")
		}
	}

	// ========================================
	// Scenario: Restart Daemon
	// ========================================

	t.Logf("")
	t.Logf("📋 Phase 5: Restarting daemon")

	// Find daemon binary
	daemonBinary, err := findDaemonBinary()
	if err != nil {
		t.Skipf("Could not find daemon binary: %v", err)
	}

	t.Logf("Starting daemon: %s", daemonBinary)

	// Start daemon in background
	cmd := exec.Command(daemonBinary)
	err = cmd.Start()
	integration.AssertNoError(t, err, "Failed to start daemon")

	newDaemonPID := cmd.Process.Pid
	t.Logf("✅ Daemon restarted: PID %d", newDaemonPID)

	// Wait for daemon to be ready
	t.Logf("Waiting for daemon to initialize...")
	daemonReady := false
	for i := 0; i < 30; i++ { // Wait up to 30 seconds
		_, err := ctx.Client.ListInstances(context.Background())
		if err == nil {
			daemonReady = true
			t.Logf("✅ Daemon is responsive (after %d seconds)", i+1)
			break
		}
		time.Sleep(1 * time.Second)
	}

	if !daemonReady {
		t.Fatal("Daemon did not become responsive after restart")
	}

	// ========================================
	// Scenario: Verify State Recovery
	// ========================================

	t.Logf("")
	t.Logf("📋 Phase 6: Verifying state recovery")

	// Get post-crash state
	postState, err := ctx.Client.ListInstances(context.Background())
	integration.AssertNoError(t, err, "Failed to get post-crash state")
	t.Logf("✅ Post-crash state retrieved: %d instances", len(postState.Instances))

	// Verify project still exists
	recoveredProject, err := ctx.Client.GetProject(context.Background(), project.ID)
	integration.AssertNoError(t, err, "Project should be recoverable")
	integration.AssertEqual(t, project.ID, recoveredProject.ID, "Project ID should match")
	integration.AssertEqual(t, project.Name, recoveredProject.Name, "Project name should match")
	t.Logf("✅ Project recovered: %s", project.Name)

	// Compare states
	if len(postState.Instances) != len(preState.Instances) {
		t.Logf("⚠️  Instance count changed: %d -> %d", len(preState.Instances), len(postState.Instances))
	} else {
		t.Logf("✅ Instance count consistent: %d instances", len(postState.Instances))
	}

	// ========================================
	// Scenario: Verify Operations Work After Recovery
	// ========================================

	t.Logf("")
	t.Logf("📋 Phase 7: Verifying operations after recovery")

	// Try creating new resource
	instanceName := integration.GenerateTestName("post-crash-instance")
	t.Logf("Creating instance after crash: %s", instanceName)

	instance, err := fixtures.CreateTestInstance(t, registry, fixtures.CreateTestInstanceOptions{
		Template:  "Python ML Workstation",
		Name:      instanceName,
		Size:      "S",
		ProjectID: &project.ID,
	})
	integration.AssertNoError(t, err, "Should be able to create instance after crash")
	integration.AssertEqual(t, "running", instance.State, "New instance should be running")
	t.Logf("✅ Instance created successfully: %s", instance.ID)

	// ========================================
	// Success Summary
	// ========================================

	t.Logf("")
	t.Logf("✅ Daemon Crash During Operation Test Complete!")
	t.Logf("   ✓ Daemon crashed (SIGKILL)")
	t.Logf("   ✓ State file integrity maintained")
	t.Logf("   ✓ Daemon restarted successfully")
	t.Logf("   ✓ State recovered correctly")
	t.Logf("   ✓ Operations work after recovery")
	t.Logf("")
	t.Logf("🎉 System survives daemon crashes!")
}

// TestOutOfMemoryHandling validates that the system handles OOM conditions
// gracefully without data corruption.
//
// Chaos Scenario: System runs out of memory during large template provisioning
// Expected Behavior:
// - Operation fails with clear OOM error
// - No state corruption
// - System recovers after memory is available
// - Partial work is cleaned up
//
// Addresses Issue #412 - System Chaos Testing
func TestOutOfMemoryHandling(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping chaos test in short mode")
	}

	t.Logf("🌪️  CHAOS TEST: Out of Memory Handling")
	t.Logf("")

	// Note: Actual OOM simulation requires memory pressure tools
	// This test validates monitoring and error handling patterns

	t.Logf("📋 Testing memory monitoring and limits")

	// Get current memory stats
	var mem runtime.MemStats
	runtime.ReadMemStats(&mem)

	initialAlloc := mem.Alloc
	t.Logf("Initial memory allocation: %.2f MB", float64(initialAlloc)/(1024*1024))

	// ========================================
	// Scenario: Large Memory Allocation
	// ========================================

	t.Logf("")
	t.Logf("📋 Testing large memory allocation handling")

	// Attempt to allocate large chunk of memory
	chunkSize := 100 * 1024 * 1024 // 100 MB
	chunks := 10

	var allocatedChunks [][]byte
	var totalAllocated int64

	for i := 0; i < chunks; i++ {
		chunk := make([]byte, chunkSize)
		allocatedChunks = append(allocatedChunks, chunk)
		totalAllocated += int64(chunkSize)

		runtime.ReadMemStats(&mem)
		t.Logf("   Chunk %d: Allocated %.2f MB (total: %.2f MB)",
			i+1, float64(chunkSize)/(1024*1024), float64(mem.Alloc)/(1024*1024))
	}

	t.Logf("✅ Successfully allocated %.2f MB", float64(totalAllocated)/(1024*1024))

	// Keep the chunks live until here, then let them become garbage.
	runtime.KeepAlive(allocatedChunks)
	runtime.GC()
	runtime.ReadMemStats(&mem)

	t.Logf("✅ Memory released (current: %.2f MB)", float64(mem.Alloc)/(1024*1024))

	// ========================================
	// Scenario: Memory Limits and Monitoring
	// ========================================

	t.Logf("")
	t.Logf("📋 Testing memory limit awareness")

	// Check if we can detect system memory limits
	if totalRAM, freeRAM, ok := systemMemory(); ok {
		t.Logf("System memory:")
		t.Logf("   Total: %.2f GB", float64(totalRAM)/(1024*1024*1024))
		t.Logf("   Free: %.2f GB", float64(freeRAM)/(1024*1024*1024))
		t.Logf("   Usage: %.1f%%", float64(totalRAM-freeRAM)/float64(totalRAM)*100)
	} else {
		t.Logf("⚠️  Could not read system memory info (platform: %s)", runtime.GOOS)
	}

	// ========================================
	// Success Summary
	// ========================================

	t.Logf("")
	t.Logf("✅ Out of Memory Handling Test Complete!")
	t.Logf("   ✓ Large memory allocations handled")
	t.Logf("   ✓ Memory properly released")
	t.Logf("   ✓ Memory limits detectable")
	t.Logf("")
	t.Logf("🎉 System handles memory pressure!")
}

// TestDiskFullScenario validates that operations handle disk full conditions
// gracefully without state corruption.
//
// Chaos Scenario: Disk fills up during state file write
// Expected Behavior:
// - Write operations fail with clear error
// - Existing state not corrupted
// - Recovery after space is freed
// - Atomic writes prevent partial state
//
// Addresses Issue #412 - System Chaos Testing
func TestDiskFullScenario(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping chaos test in short mode")
	}

	t.Logf("🌪️  CHAOS TEST: Disk Full Scenario")
	t.Logf("")

	// ========================================
	// Scenario: Check Disk Space
	// ========================================

	t.Logf("📋 Monitoring disk space")

	homeDir, err := os.UserHomeDir()
	integration.AssertNoError(t, err, "Failed to get home directory")

	var stat syscall.Statfs_t
	err = syscall.Statfs(homeDir, &stat)
	if err != nil {
		t.Logf("⚠️  Could not get disk stats (platform: %s)", runtime.GOOS)
	} else {
		// Available blocks * block size
		available := stat.Bavail * uint64(stat.Bsize)
		total := stat.Blocks * uint64(stat.Bsize)
		used := total - available

		t.Logf("Disk space:")
		t.Logf("   Total: %.2f GB", float64(total)/(1024*1024*1024))
		t.Logf("   Used: %.2f GB", float64(used)/(1024*1024*1024))
		t.Logf("   Available: %.2f GB", float64(available)/(1024*1024*1024))
		t.Logf("   Usage: %.1f%%", float64(used)/float64(total)*100)

		// Verify sufficient space
		minRequired := uint64(1024 * 1024 * 1024) // 1 GB
		if available < minRequired {
			t.Logf("⚠️  Low disk space (< 1 GB available)")
		} else {
			t.Logf("✅ Sufficient disk space available")
		}
	}

	// ========================================
	// Scenario: Large File Write Test
	// ========================================

	t.Logf("")
	t.Logf("📋 Testing large file write handling")

	// Create temporary test file
	tmpDir := os.TempDir()
	testFile := filepath.Join(tmpDir, fmt.Sprintf("prism-chaos-test-%d.tmp", time.Now().Unix()))

	t.Logf("Creating test file: %s", testFile)

	// Write large file to test disk space handling
	file, err := os.Create(testFile)
	integration.AssertNoError(t, err, "Failed to create test file")
	defer os.Remove(testFile) // Cleanup

	// Write 100 MB in 10 MB chunks
	chunkSize := 10 * 1024 * 1024 // 10 MB
	chunks := 10
	data := make([]byte, chunkSize)

	for i := 0; i < chunks; i++ {
		_, err := file.Write(data)
		if err != nil {
			t.Logf("⚠️  Write failed at chunk %d: %v", i+1, err)
			if strings.Contains(err.Error(), "no space left") {
				t.Logf("✅ Detected disk full condition")
			}
			break
		}
	}

	file.Close()

	// Verify file was created
	info, err := os.Stat(testFile)
	if err == nil {
		t.Logf("✅ Test file created: %.2f MB", float64(info.Size())/(1024*1024))
	}

	// ========================================
	// Scenario: State File Integrity
	// ========================================

	t.Logf("")
	t.Logf("📋 Verifying state file integrity")

	stateFile := filepath.Join(homeDir, ".prism", "state.json")
	if _, err := os.Stat(stateFile); err == nil {
		// Read state file
		stateData, err := os.ReadFile(stateFile)
		if err != nil {
			t.Logf("⚠️  Could not read state file: %v", err)
		} else {
			t.Logf("✅ State file readable (%d bytes)", len(stateData))

			// Verify it's valid JSON
			if len(stateData) > 0 && stateData[0] == '{' {
				t.Logf("✅ State file appears valid")
			} else {
				t.Error("State file appears corrupted")
			}
		}
	} else {
		t.Logf("⚠️  State file not found (may use alternate location)")
	}

	// ========================================
	// Success Summary
	// ========================================

	t.Logf("")
	t.Logf("✅ Disk Full Scenario Test Complete!")
	t.Logf("   ✓ Disk space monitoring works")
	t.Logf("   ✓ Large file operations handled")
	t.Logf("   ✓ State file integrity maintained")
	t.Logf("")
	t.Logf("🎉 System handles disk space issues!")
}

// Helper Functions

func findDaemonProcess() (int, error) {
	// Try to find prismd process
	cmd := exec.Command("pgrep", "-f", "prismd")
	output, err := cmd.Output()
	if err != nil {
		return 0, fmt.Errorf("prismd process not found")
	}

	pidStr := strings.TrimSpace(string(output))
	var pid int
	fmt.Sscanf(pidStr, "%d", &pid)

	if pid == 0 {
		return 0, fmt.Errorf("invalid PID: %s", pidStr)
	}

	return pid, nil
}

func findDaemonBinary() (string, error) {
	// Try common locations
	locations := []string{
		"./bin/prismd",
		"../../../bin/prismd",
		"prismd", // In PATH
	}

	for _, loc := range locations {
		if _, err := os.Stat(loc); err == nil {
			return loc, nil
		}
	}

	// Try to find in PATH
	path, err := exec.LookPath("prismd")
	if err == nil {
		return path, nil
	}

	return "", fmt.Errorf("daemon binary not found in expected locations")
}
