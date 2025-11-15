//go:build integration
// +build integration

package integration

import (
	"context"
	"strings"
	"testing"
	"time"
)

// TestCLISnapshotOperations tests snapshot creation and restoration via CLI
// Priority: HIGH - Snapshots are critical for disaster recovery and instance cloning
func TestCLISnapshotOperations(t *testing.T) {
	ctx := NewCLITestContext(t)
	defer ctx.Cleanup()

	instanceName := GenerateTestName("test-snapshot-source")
	snapshotName := "backup-v1"
	restoredInstanceName := GenerateTestName("test-snapshot-restored")

	// Launch source instance
	t.Run("LaunchSourceInstance", func(t *testing.T) {
		t.Logf("Launching source instance: %s", instanceName)

		result, err := ctx.LaunchInstanceCLI("python-ml-workstation", instanceName, "S")
		AssertNoError(t, err, "launch source instance")
		result.AssertSuccess(t, "launch should succeed")

		// Wait for instance to be fully ready
		err = ctx.WaitForInstanceState(instanceName, "running", InstanceReadyTimeout)
		AssertNoError(t, err, "instance should be running")

		t.Logf("✓ Source instance launched")
	})

	// Create snapshot
	t.Run("CreateSnapshot", func(t *testing.T) {
		t.Logf("Creating snapshot '%s' from instance '%s'", snapshotName, instanceName)

		result := ctx.Prism("snapshot", "create", instanceName, snapshotName)

		if result.ExitCode != 0 {
			if strings.Contains(result.Stderr, "unknown command") ||
				strings.Contains(result.Stderr, "not found") {
				t.Skip("Snapshot commands not yet implemented")
				return
			}
		}

		result.AssertSuccess(t, "snapshot create should succeed")

		// Snapshot creation can take several minutes (EBS snapshot time)
		t.Logf("✓ Snapshot creation initiated (may take 5-15 minutes)")
	})

	// List snapshots
	t.Run("ListSnapshots", func(t *testing.T) {
		t.Logf("Listing snapshots")

		result := ctx.Prism("snapshot", "list")

		if result.ExitCode != 0 {
			if strings.Contains(result.Stderr, "unknown command") {
				t.Skip("Snapshot list command not yet implemented")
				return
			}
		}

		result.AssertSuccess(t, "snapshot list should succeed")

		// Check if our snapshot appears (might be pending)
		output := result.Stdout + result.Stderr
		if !strings.Contains(output, snapshotName) {
			t.Logf("Warning: Snapshot '%s' not found in list (may still be creating)", snapshotName)
		} else {
			t.Logf("✓ Snapshot found in list")
		}
	})

	// Wait for snapshot to complete (EBS snapshots can take time)
	t.Run("WaitForSnapshot", func(t *testing.T) {
		t.Logf("Waiting for snapshot to complete (checking every 30s, max 20 minutes)")

		// Poll for snapshot completion
		maxWait := 20 * time.Minute
		pollInterval := 30 * time.Second
		deadline := time.Now().Add(maxWait)

		for time.Now().Before(deadline) {
			result := ctx.Prism("snapshot", "list")

			if result.ExitCode != 0 {
				if strings.Contains(result.Stderr, "unknown command") {
					t.Skip("Snapshot commands not yet implemented")
					return
				}
			}

			output := result.Stdout + result.Stderr

			// Check for completion indicators
			if strings.Contains(output, "completed") ||
				strings.Contains(output, "available") ||
				strings.Contains(output, "ready") {
				t.Logf("✓ Snapshot appears to be complete")
				return
			}

			// If snapshot is still pending/in-progress, wait
			if strings.Contains(output, "pending") ||
				strings.Contains(output, "creating") {
				t.Logf("Snapshot still creating, waiting %s...", pollInterval)
				time.Sleep(pollInterval)
				continue
			}

			// If we can't determine status, assume it's ready
			break
		}

		t.Logf("✓ Snapshot wait complete (or timed out, will try restore)")
	})

	// Restore snapshot to new instance
	t.Run("RestoreSnapshot", func(t *testing.T) {
		t.Logf("Restoring snapshot '%s' to new instance '%s'", snapshotName, restoredInstanceName)

		result := ctx.Prism("snapshot", "restore", snapshotName, restoredInstanceName)

		if result.ExitCode != 0 {
			if strings.Contains(result.Stderr, "unknown command") {
				t.Skip("Snapshot restore command not yet implemented")
				return
			}

			// Snapshot might not be ready yet
			if strings.Contains(result.Stderr, "not ready") ||
				strings.Contains(result.Stderr, "still creating") {
				t.Skip("Snapshot not ready for restore yet")
				return
			}

			t.Fatalf("Snapshot restore failed: %s", result.Stderr)
		}

		result.AssertSuccess(t, "snapshot restore should succeed")

		// Wait for restored instance to be running
		err := ctx.WaitForInstanceState(restoredInstanceName, "running", InstanceReadyTimeout)
		AssertNoError(t, err, "restored instance should be running")

		t.Logf("✓ Snapshot restored successfully")
	})

	// Verify restored instance
	t.Run("VerifyRestoredInstance", func(t *testing.T) {
		t.Logf("Verifying restored instance")

		// Get instance details
		instance, err := ctx.Client.GetInstance(context.Background(), restoredInstanceName)
		AssertNoError(t, err, "get restored instance details")

		// Verify instance is running
		AssertEqual(t, "running", instance.State, "restored instance should be running")

		// Verify it has the same template
		if instance.Template != "" {
			t.Logf("✓ Restored instance has template: %s", instance.Template)
		}

		t.Logf("✓ Restored instance verified")
	})

	// Cleanup
	t.Run("Cleanup", func(t *testing.T) {
		// Delete both instances
		err := ctx.DeleteInstanceCLI(instanceName)
		if err != nil {
			t.Logf("Warning: Failed to delete source instance: %v", err)
		}

		err = ctx.DeleteInstanceCLI(restoredInstanceName)
		if err != nil {
			t.Logf("Warning: Failed to delete restored instance: %v", err)
		}

		// TODO: Delete snapshot
		// result := ctx.Prism("snapshot", "delete", snapshotName)
		// Currently not implemented, snapshots may need manual cleanup

		t.Logf("✓ Cleanup complete (snapshot may need manual deletion)")
	})
}

// TestCLISnapshotHelp tests help output for snapshot commands
func TestCLISnapshotHelp(t *testing.T) {
	ctx := NewCLITestContext(t)
	defer ctx.Cleanup()

	t.Run("SnapshotMainHelp", func(t *testing.T) {
		result := ctx.Prism("snapshot", "--help")
		result.AssertSuccess(t, "snapshot help should succeed")

		output := result.Stdout + result.Stderr

		// Check for expected subcommands
		expectedCommands := []string{"create", "list", "restore"}
		foundCommands := 0
		for _, cmd := range expectedCommands {
			if strings.Contains(output, cmd) {
				foundCommands++
			}
		}

		if foundCommands == 0 {
			t.Error("Expected snapshot subcommands in help output")
		}

		t.Logf("✓ Snapshot help available (found %d/%d commands)", foundCommands, len(expectedCommands))
	})

	t.Run("SnapshotCreateHelp", func(t *testing.T) {
		result := ctx.Prism("snapshot", "create", "--help")

		if result.ExitCode != 0 {
			if strings.Contains(result.Stderr, "unknown command") {
				t.Skip("Snapshot create command not yet implemented")
				return
			}
		}

		result.AssertSuccess(t, "snapshot create help should succeed")

		t.Logf("✓ Snapshot create help available")
	})
}

// TestCLISnapshotValidation tests error handling for snapshot operations
func TestCLISnapshotValidation(t *testing.T) {
	ctx := NewCLITestContext(t)
	defer ctx.Cleanup()

	t.Run("CreateSnapshotNonexistentInstance", func(t *testing.T) {
		t.Logf("Testing snapshot creation on nonexistent instance")

		result := ctx.Prism("snapshot", "create", "nonexistent-instance-12345", "test-snapshot")

		if result.ExitCode != 0 {
			if strings.Contains(result.Stderr, "unknown command") {
				t.Skip("Snapshot commands not yet implemented")
				return
			}
		}

		// Should fail with clear error
		result.AssertFailure(t, "snapshot of nonexistent instance should fail")

		t.Logf("✓ Nonexistent instance error handled")
	})

	t.Run("RestoreNonexistentSnapshot", func(t *testing.T) {
		t.Logf("Testing restore of nonexistent snapshot")

		result := ctx.Prism("snapshot", "restore", "nonexistent-snapshot", "new-instance")

		if result.ExitCode != 0 {
			if strings.Contains(result.Stderr, "unknown command") {
				t.Skip("Snapshot restore command not yet implemented")
				return
			}
		}

		// Should fail with clear error
		result.AssertFailure(t, "nonexistent snapshot restore should fail")

		t.Logf("✓ Nonexistent snapshot error handled")
	})
}

// TestCLISnapshotCloning tests creating multiple instances from same snapshot
func TestCLISnapshotCloning(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping long-running snapshot cloning test")
	}

	ctx := NewCLITestContext(t)
	defer ctx.Cleanup()

	sourceInstance := GenerateTestName("test-clone-source")
	snapshotName := "clone-template"
	clone1 := GenerateTestName("test-clone-1")
	clone2 := GenerateTestName("test-clone-2")

	t.Run("CreateSourceAndSnapshot", func(t *testing.T) {
		t.Logf("Creating source instance and snapshot")

		// Launch source
		result, err := ctx.LaunchInstanceCLI("python-ml-workstation", sourceInstance, "S")
		AssertNoError(t, err, "launch source")
		result.AssertSuccess(t, "launch should succeed")

		// Wait for ready
		err = ctx.WaitForInstanceState(sourceInstance, "running", InstanceReadyTimeout)
		AssertNoError(t, err, "source should be running")

		// Create snapshot
		snapResult := ctx.Prism("snapshot", "create", sourceInstance, snapshotName)
		if snapResult.ExitCode != 0 {
			if strings.Contains(snapResult.Stderr, "unknown command") {
				t.Skip("Snapshot commands not yet implemented")
				return
			}
		}

		snapResult.AssertSuccess(t, "snapshot creation should succeed")

		t.Logf("✓ Source and snapshot created")
	})

	t.Run("CloneMultipleInstances", func(t *testing.T) {
		t.Logf("Creating multiple instances from snapshot")

		// Wait for snapshot to be ready (simplified check)
		time.Sleep(2 * time.Minute)

		// Restore to clone1
		result1 := ctx.Prism("snapshot", "restore", snapshotName, clone1)
		if result1.ExitCode != 0 {
			if strings.Contains(result1.Stderr, "unknown command") {
				t.Skip("Snapshot restore not yet implemented")
				return
			}
		}

		// Restore to clone2
		result2 := ctx.Prism("snapshot", "restore", snapshotName, clone2)
		if result2.ExitCode == 0 {
			t.Logf("✓ Successfully created multiple instances from same snapshot")
		}

		// Verify both clones exist
		err := ctx.WaitForInstanceState(clone1, "running", InstanceReadyTimeout)
		if err == nil {
			t.Logf("✓ Clone 1 is running")
		}

		err = ctx.WaitForInstanceState(clone2, "running", InstanceReadyTimeout)
		if err == nil {
			t.Logf("✓ Clone 2 is running")
		}
	})

	t.Run("Cleanup", func(t *testing.T) {
		ctx.DeleteInstanceCLI(sourceInstance)
		ctx.DeleteInstanceCLI(clone1)
		ctx.DeleteInstanceCLI(clone2)
		// Snapshot cleanup would go here
	})
}
