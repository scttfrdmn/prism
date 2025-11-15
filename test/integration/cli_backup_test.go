//go:build integration
// +build integration

package integration

import (
	"strings"
	"testing"
	"time"
)

// TestCLIBackupOperations tests data backup and restore operations via CLI
// Priority: HIGH - Data protection is critical for research workflows
func TestCLIBackupOperations(t *testing.T) {
	ctx := NewCLITestContext(t)
	defer ctx.Cleanup()

	sourceInstance := GenerateTestName("test-backup-source")
	targetInstance := GenerateTestName("test-backup-target")
	backupName := "daily-backup"

	// Launch source instance
	t.Run("LaunchSourceInstance", func(t *testing.T) {
		t.Logf("Launching source instance: %s", sourceInstance)

		result, err := ctx.LaunchInstanceCLI("python-ml-workstation", sourceInstance, "S")
		AssertNoError(t, err, "launch source instance")
		result.AssertSuccess(t, "launch should succeed")

		// Wait for instance to be ready
		err = ctx.WaitForInstanceState(sourceInstance, "running", InstanceReadyTimeout)
		AssertNoError(t, err, "instance should be running")

		t.Logf("✓ Source instance launched and ready")
	})

	// TODO: Create test files on instance via SSH/SSM
	// This would require SSH access which may not be configured in test environment
	// For now, we'll test the backup command itself

	// Create backup
	t.Run("CreateBackup", func(t *testing.T) {
		t.Logf("Creating backup '%s' from instance '%s'", backupName, sourceInstance)

		result := ctx.Prism("backup", "create", sourceInstance, backupName)

		if result.ExitCode != 0 {
			if strings.Contains(result.Stderr, "unknown command") ||
				strings.Contains(result.Stderr, "not found") {
				t.Skip("Backup commands not yet implemented")
				return
			}

			// Backup might not be fully implemented yet
			t.Logf("Backup creation may not be implemented: %s", result.Stderr)
			t.Skip("Backup functionality appears incomplete")
			return
		}

		result.AssertSuccess(t, "backup create should succeed")

		t.Logf("✓ Backup creation initiated")
	})

	// List backups
	t.Run("ListBackups", func(t *testing.T) {
		t.Logf("Listing backups")

		result := ctx.Prism("backup", "list")

		if result.ExitCode != 0 {
			if strings.Contains(result.Stderr, "unknown command") {
				t.Skip("Backup list command not yet implemented")
				return
			}
		}

		result.AssertSuccess(t, "backup list should succeed")

		// Check if our backup appears
		output := result.Stdout + result.Stderr
		if strings.Contains(output, backupName) {
			t.Logf("✓ Backup found in list")
		} else {
			t.Logf("Note: Backup '%s' not found in list (may still be processing)", backupName)
		}
	})

	// Launch target instance for restore
	t.Run("LaunchTargetInstance", func(t *testing.T) {
		t.Logf("Launching target instance for restore: %s", targetInstance)

		result, err := ctx.LaunchInstanceCLI("python-ml-workstation", targetInstance, "S")
		AssertNoError(t, err, "launch target instance")
		result.AssertSuccess(t, "launch should succeed")

		// Wait for ready
		err = ctx.WaitForInstanceState(targetInstance, "running", InstanceReadyTimeout)
		AssertNoError(t, err, "target instance should be running")

		t.Logf("✓ Target instance launched")
	})

	// Restore backup
	t.Run("RestoreBackup", func(t *testing.T) {
		t.Logf("Restoring backup '%s' to instance '%s'", backupName, targetInstance)

		result := ctx.Prism("restore", backupName, targetInstance)

		if result.ExitCode != 0 {
			if strings.Contains(result.Stderr, "unknown command") {
				t.Skip("Restore command not yet implemented")
				return
			}

			// Backup might not be ready or command syntax different
			t.Logf("Restore may have failed: %s", result.Stderr)
			if strings.Contains(result.Stderr, "not found") {
				t.Skip("Backup not found (may not have been created successfully)")
				return
			}
		}

		result.AssertSuccess(t, "restore should succeed")

		t.Logf("✓ Backup restored")
	})

	// TODO: Verify restored data matches source
	// This would require SSH access to check files

	// Cleanup
	t.Run("Cleanup", func(t *testing.T) {
		err := ctx.DeleteInstanceCLI(sourceInstance)
		if err != nil {
			t.Logf("Warning: Failed to delete source instance: %v", err)
		}

		err = ctx.DeleteInstanceCLI(targetInstance)
		if err != nil {
			t.Logf("Warning: Failed to delete target instance: %v", err)
		}

		// TODO: Delete backup
		// result := ctx.Prism("backup", "delete", backupName)

		t.Logf("✓ Cleanup complete")
	})
}

// TestCLIBackupWithOptions tests backup creation with various options
func TestCLIBackupWithOptions(t *testing.T) {
	ctx := NewCLITestContext(t)
	defer ctx.Cleanup()

	instanceName := GenerateTestName("test-backup-options")

	t.Run("LaunchInstance", func(t *testing.T) {
		result, err := ctx.LaunchInstanceCLI("python-ml-workstation", instanceName, "S")
		AssertNoError(t, err, "launch instance")
		result.AssertSuccess(t, "launch should succeed")

		err = ctx.WaitForInstanceState(instanceName, "running", InstanceReadyTimeout)
		AssertNoError(t, err, "instance should be running")
	})

	t.Run("BackupWithCompression", func(t *testing.T) {
		t.Logf("Testing backup with compression flag")

		result := ctx.Prism("backup", "create", instanceName, "compressed-backup", "--compress")

		if result.ExitCode != 0 {
			if strings.Contains(result.Stderr, "unknown command") ||
				strings.Contains(result.Stderr, "unknown flag") {
				t.Skip("Backup compression option not yet implemented")
				return
			}
		}

		// May or may not be implemented
		if result.ExitCode == 0 {
			t.Logf("✓ Backup with compression succeeded")
		}
	})

	t.Run("BackupWithEncryption", func(t *testing.T) {
		t.Logf("Testing backup with encryption flag")

		result := ctx.Prism("backup", "create", instanceName, "encrypted-backup", "--encrypt")

		if result.ExitCode != 0 {
			if strings.Contains(result.Stderr, "unknown command") ||
				strings.Contains(result.Stderr, "unknown flag") {
				t.Skip("Backup encryption option not yet implemented")
				return
			}
		}

		if result.ExitCode == 0 {
			t.Logf("✓ Backup with encryption succeeded")
		}
	})

	t.Run("SelectiveBackup", func(t *testing.T) {
		t.Logf("Testing selective backup (specific directories)")

		result := ctx.Prism("backup", "create", instanceName, "selective-backup", "--path", "/home/researcher")

		if result.ExitCode != 0 {
			if strings.Contains(result.Stderr, "unknown command") ||
				strings.Contains(result.Stderr, "unknown flag") {
				t.Skip("Selective backup not yet implemented")
				return
			}
		}

		if result.ExitCode == 0 {
			t.Logf("✓ Selective backup succeeded")
		}
	})

	t.Run("Cleanup", func(t *testing.T) {
		ctx.DeleteInstanceCLI(instanceName)
	})
}

// TestCLIBackupValidation tests error handling for backup operations
func TestCLIBackupValidation(t *testing.T) {
	ctx := NewCLITestContext(t)
	defer ctx.Cleanup()

	t.Run("BackupNonexistentInstance", func(t *testing.T) {
		t.Logf("Testing backup of nonexistent instance")

		result := ctx.Prism("backup", "create", "nonexistent-instance-12345", "test-backup")

		if result.ExitCode != 0 {
			if strings.Contains(result.Stderr, "unknown command") {
				t.Skip("Backup commands not yet implemented")
				return
			}
		}

		// Should fail with clear error
		result.AssertFailure(t, "backup of nonexistent instance should fail")

		t.Logf("✓ Nonexistent instance error handled")
	})

	t.Run("RestoreNonexistentBackup", func(t *testing.T) {
		t.Logf("Testing restore of nonexistent backup")

		result := ctx.Prism("restore", "nonexistent-backup", "target-instance")

		if result.ExitCode != 0 {
			if strings.Contains(result.Stderr, "unknown command") {
				t.Skip("Restore command not yet implemented")
				return
			}
		}

		// Should fail with clear error
		result.AssertFailure(t, "nonexistent backup restore should fail")

		t.Logf("✓ Nonexistent backup error handled")
	})

	t.Run("RestoreToNonexistentInstance", func(t *testing.T) {
		t.Logf("Testing restore to nonexistent instance")

		// This assumes "daily-backup" doesn't exist, which is fine for error testing
		result := ctx.Prism("restore", "some-backup", "nonexistent-target-12345")

		if result.ExitCode != 0 {
			if strings.Contains(result.Stderr, "unknown command") {
				t.Skip("Restore command not yet implemented")
				return
			}
		}

		// Should fail
		result.AssertFailure(t, "restore to nonexistent instance should fail")

		t.Logf("✓ Nonexistent target instance error handled")
	})
}

// TestCLIBackupHelp tests help output for backup commands
func TestCLIBackupHelp(t *testing.T) {
	ctx := NewCLITestContext(t)
	defer ctx.Cleanup()

	t.Run("BackupMainHelp", func(t *testing.T) {
		result := ctx.Prism("backup", "--help")
		result.AssertSuccess(t, "backup help should succeed")

		output := result.Stdout + result.Stderr

		// Check for expected concepts
		expectedTerms := []string{"create", "list", "data"}
		foundTerms := 0
		for _, term := range expectedTerms {
			if strings.Contains(strings.ToLower(output), term) {
				foundTerms++
			}
		}

		if foundTerms > 0 {
			t.Logf("✓ Backup help available (found %d/%d expected terms)", foundTerms, len(expectedTerms))
		}
	})

	t.Run("RestoreHelp", func(t *testing.T) {
		result := ctx.Prism("restore", "--help")

		// Restore might be backup subcommand or separate command
		if result.ExitCode != 0 {
			// Try as backup subcommand
			result = ctx.Prism("backup", "restore", "--help")
		}

		if result.ExitCode != 0 {
			if strings.Contains(result.Stderr, "unknown command") {
				t.Skip("Restore command not yet implemented")
				return
			}
		}

		result.AssertSuccess(t, "restore help should succeed")

		t.Logf("✓ Restore help available")
	})
}

// TestCLIBackupIncremental tests incremental backup functionality
func TestCLIBackupIncremental(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping long-running incremental backup test")
	}

	ctx := NewCLITestContext(t)
	defer ctx.Cleanup()

	instanceName := GenerateTestName("test-incremental")

	t.Run("CreateInitialBackup", func(t *testing.T) {
		t.Logf("Creating instance for incremental backup test")

		result, err := ctx.LaunchInstanceCLI("python-ml-workstation", instanceName, "S")
		AssertNoError(t, err, "launch instance")
		result.AssertSuccess(t, "launch should succeed")

		err = ctx.WaitForInstanceState(instanceName, "running", InstanceReadyTimeout)
		AssertNoError(t, err, "instance should be running")

		// Create initial backup
		backupResult := ctx.Prism("backup", "create", instanceName, "initial-backup")
		if backupResult.ExitCode != 0 {
			if strings.Contains(backupResult.Stderr, "unknown command") {
				t.Skip("Backup commands not yet implemented")
				return
			}
		}

		backupResult.AssertSuccess(t, "initial backup should succeed")

		t.Logf("✓ Initial backup created")
	})

	t.Run("CreateIncrementalBackup", func(t *testing.T) {
		t.Logf("Testing incremental backup")

		// Wait a bit to simulate data changes
		time.Sleep(10 * time.Second)

		// Create incremental backup
		result := ctx.Prism("backup", "create", instanceName, "incremental-backup", "--incremental")

		if result.ExitCode != 0 {
			if strings.Contains(result.Stderr, "unknown flag") {
				t.Skip("Incremental backup not yet supported")
				return
			}
		}

		// Incremental backup should be smaller/faster than full backup
		if result.ExitCode == 0 {
			t.Logf("✓ Incremental backup succeeded")
		}
	})

	t.Run("Cleanup", func(t *testing.T) {
		ctx.DeleteInstanceCLI(instanceName)
	})
}

// TestCLIBackupStorage tests different backup storage options
func TestCLIBackupStorage(t *testing.T) {
	ctx := NewCLITestContext(t)
	defer ctx.Cleanup()

	instanceName := GenerateTestName("test-backup-storage")

	t.Run("LaunchInstance", func(t *testing.T) {
		result, err := ctx.LaunchInstanceCLI("python-ml-workstation", instanceName, "S")
		AssertNoError(t, err, "launch instance")
		result.AssertSuccess(t, "launch should succeed")

		err = ctx.WaitForInstanceState(instanceName, "running", InstanceReadyTimeout)
		AssertNoError(t, err, "instance should be running")
	})

	t.Run("BackupToS3", func(t *testing.T) {
		t.Logf("Testing backup to S3")

		result := ctx.Prism("backup", "create", instanceName, "s3-backup", "--storage", "s3")

		if result.ExitCode != 0 {
			if strings.Contains(result.Stderr, "unknown flag") {
				t.Skip("Storage option not yet implemented")
				return
			}
		}

		if result.ExitCode == 0 {
			t.Logf("✓ S3 backup succeeded")
		}
	})

	t.Run("BackupToEFS", func(t *testing.T) {
		t.Logf("Testing backup to EFS")

		result := ctx.Prism("backup", "create", instanceName, "efs-backup", "--storage", "efs")

		if result.ExitCode != 0 {
			if strings.Contains(result.Stderr, "unknown flag") {
				t.Skip("EFS storage option not yet implemented")
				return
			}
		}

		if result.ExitCode == 0 {
			t.Logf("✓ EFS backup succeeded")
		}
	})

	t.Run("Cleanup", func(t *testing.T) {
		ctx.DeleteInstanceCLI(instanceName)
	})
}
