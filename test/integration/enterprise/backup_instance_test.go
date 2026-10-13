//go:build integration
// +build integration

package enterprise_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/scttfrdmn/prism/pkg/api/client"
	"github.com/scttfrdmn/prism/pkg/types"
	"github.com/scttfrdmn/prism/test/fixtures"
	"github.com/scttfrdmn/prism/test/integration"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestBackup_CreateAndDelete validates basic backup lifecycle: create and delete
// Flow:
// 1. Launch test instance (Ubuntu Basic, size S)
// 2. Wait for instance to be running (handled by CreateTestInstance)
// 3. Create backup from instance
// 4. Wait for backup to be available (handled by CreateTestBackup)
// 5. Verify backup metadata (name, source instance, state)
// 6. Delete backup
// 7. Verify backup is deleted
func TestBackup_CreateAndDelete(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Setup
	ctx := context.Background()
	testCtx := integration.NewTestContext(t)
	registry := fixtures.NewFixtureRegistry(t, testCtx.Client)

	t.Log("=== Test 1: TestBackup_CreateAndDelete ===")

	// Step 1: Launch test instance (Ubuntu 22.04 Server, size S)
	t.Log("Step 1: Launching test instance...")
	instanceName := fmt.Sprintf("backup-test-%d", time.Now().Unix())
	instance, err := fixtures.CreateTestInstance(t, registry, fixtures.CreateTestInstanceOptions{
		Name:     instanceName,
		Template: "Ubuntu 22.04 Server",
		Size:     "S",
	})
	require.NoError(t, err, "Failed to create test instance")
	require.NotNil(t, instance, "Instance should not be nil")
	t.Logf("   ✓ Instance launched: %s (state: %s)", instance.Name, instance.State)

	// Step 2: Wait for instance to be running (already done by CreateTestInstance)
	assert.Equal(t, "running", instance.State, "Instance should be in running state")

	// Step 3: Create backup from instance
	t.Log("Step 2: Creating backup from instance...")
	backupName := fmt.Sprintf("backup-%s", instanceName)
	backup, err := fixtures.CreateTestBackup(t, registry, fixtures.CreateTestBackupOptions{
		Name:        backupName,
		InstanceID:  instanceName,
		Description: "Test backup for create/delete workflow",
	})
	require.NoError(t, err, "Failed to create backup")
	require.NotNil(t, backup, "Backup should not be nil")
	t.Logf("   ✓ Backup created: %s (state: %s)", backup.BackupName, backup.State)

	// Step 4: Wait for backup to be available (already done by CreateTestBackup)
	assert.Equal(t, "available", backup.State, "Backup should be in available state")

	// Step 5: Verify backup metadata
	t.Log("Step 3: Verifying backup metadata...")
	assert.Equal(t, backupName, backup.BackupName, "Backup name should match")
	assert.Equal(t, instanceName, backup.SourceInstance, "Source instance name should match")
	assert.NotEmpty(t, backup.BackupID, "Backup should have backup ID")
	assert.False(t, backup.CreatedAt.IsZero(), "Backup should have creation time")
	t.Logf("   ✓ Backup metadata verified: ID %s, created at %s", backup.BackupID, backup.CreatedAt.Format(time.RFC3339))

	// Step 6: Delete backup
	t.Log("Step 4: Deleting backup...")
	deleteResult, err := testCtx.Client.DeleteBackup(ctx, backupName)
	require.NoError(t, err, "Failed to delete backup")
	require.NotNil(t, deleteResult, "Delete result should not be nil")
	t.Logf("   ✓ Backup deleted: %s (savings: $%.2f/month)", backupName, deleteResult.StorageSavingsMonthly)

	// Step 7: Verify backup is deleted
	t.Log("Step 5: Verifying backup is deleted...")
	_, err = testCtx.Client.GetBackup(ctx, backupName)
	assert.Error(t, err, "Getting deleted backup should return error")
	t.Log("   ✓ Backup deletion confirmed")

	// Verify storage savings were calculated
	assert.Greater(t, deleteResult.StorageSavingsMonthly, 0.0, "Storage savings should be positive")
	t.Logf("   ✓ Storage savings: $%.2f/month", deleteResult.StorageSavingsMonthly)

	t.Log("=== Test 1 Complete: Basic backup lifecycle works correctly ===")
}

// TestBackup_RestoreToTarget validates backup restoration to a target instance
// Flow:
// 1. Launch source instance
// 2. Create backup from source
// 3. Wait for backup to be available (handled by CreateTestBackup)
// 4. Launch target instance
// 5. Restore backup data to target instance (file-level restore)
// 6. Verify restore operation completed successfully
// 7. Clean up both instances + backup (handled by registry)
func TestBackup_RestoreToTarget(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Setup
	ctx := context.Background()
	testCtx := integration.NewTestContext(t)
	registry := fixtures.NewFixtureRegistry(t, testCtx.Client)

	t.Log("=== Test 2: TestBackup_RestoreToTarget ===")

	// Step 1: Launch source instance
	t.Log("Step 1: Launching source instance...")
	sourceInstanceName := fmt.Sprintf("source-instance-%d", time.Now().Unix())
	sourceInstance, err := fixtures.CreateTestInstance(t, registry, fixtures.CreateTestInstanceOptions{
		Name:     sourceInstanceName,
		Template: "Ubuntu 22.04 Server",
		Size:     "S",
	})
	require.NoError(t, err, "Failed to create source instance")
	require.NotNil(t, sourceInstance, "Source instance should not be nil")
	t.Logf("   ✓ Source instance launched: %s (state: %s)", sourceInstance.Name, sourceInstance.State)

	// Step 2: Create backup from source
	t.Log("Step 2: Creating backup from source instance...")
	backupName := fmt.Sprintf("backup-%s", sourceInstanceName)
	backup, err := fixtures.CreateTestBackup(t, registry, fixtures.CreateTestBackupOptions{
		Name:        backupName,
		InstanceID:  sourceInstanceName,
		Description: "Test backup for restoration workflow",
	})
	require.NoError(t, err, "Failed to create backup")
	require.NotNil(t, backup, "Backup should not be nil")
	t.Logf("   ✓ Backup created: %s (state: %s)", backup.BackupName, backup.State)

	// Step 3: Wait for backup to be available (already done by CreateTestBackup)
	assert.Equal(t, "available", backup.State, "Backup should be in available state")

	// Step 4: Launch target instance
	t.Log("Step 3: Launching target instance for restore...")
	targetInstanceName := fmt.Sprintf("target-instance-%d", time.Now().Unix())
	targetInstance, err := fixtures.CreateTestInstance(t, registry, fixtures.CreateTestInstanceOptions{
		Name:     targetInstanceName,
		Template: "Ubuntu 22.04 Server",
		Size:     "S",
	})
	require.NoError(t, err, "Failed to create target instance")
	require.NotNil(t, targetInstance, "Target instance should not be nil")
	t.Logf("   ✓ Target instance launched: %s (state: %s)", targetInstance.Name, targetInstance.State)

	// Step 5: Restore backup to target instance
	t.Log("Step 4: Restoring backup to target instance...")
	restoreReq := types.RestoreRequest{
		BackupName:      backupName,
		TargetInstance:  targetInstanceName,
		RestorePath:     "/", // Restore to root
		Overwrite:       true,
		PreservePerms:   true,
		VerifyIntegrity: true,
		Wait:            true,
	}

	restoreResult, err := testCtx.Client.RestoreBackup(ctx, restoreReq)
	require.NoError(t, err, "Failed to restore backup")
	require.NotNil(t, restoreResult, "Restore result should not be nil")
	t.Logf("   ✓ Restore initiated: operation ID %s", restoreResult.RestoreID)

	// Step 6: Verify restore completed successfully
	t.Log("Step 5: Verifying restore operation completed...")
	assert.Equal(t, "completed", restoreResult.State, "Restore should be completed")
	assert.Greater(t, restoreResult.RestoredBytes, int64(0), "Should have restored some bytes")
	t.Logf("   ✓ Restore completed: %d bytes restored", restoreResult.RestoredBytes)

	// Verify both instances still exist
	t.Log("Step 6: Verifying both source and target instances exist...")
	instances, err := testCtx.Client.ListInstances(ctx)
	require.NoError(t, err, "Failed to list instances")

	foundSource := false
	foundTarget := false
	for _, inst := range instances.Instances {
		if inst.Name == sourceInstanceName {
			foundSource = true
		}
		if inst.Name == targetInstanceName {
			foundTarget = true
		}
	}
	assert.True(t, foundSource, "Source instance should still exist")
	assert.True(t, foundTarget, "Target instance should exist")
	t.Log("   ✓ Both source and target instances confirmed")

	// Step 7: Cleanup handled automatically by registry
	t.Log("=== Test 2 Complete: Backup restoration to target instance works correctly ===")
}

// waitForInstanceState polls the instance until it reaches the target state or times out
func waitForInstanceState(ctx context.Context, client client.PrismAPI, instanceName, targetState string, timeout time.Duration) (*types.Instance, error) {
	startTime := time.Now()
	pollInterval := 10 * time.Second

	for time.Since(startTime) < timeout {
		instance, err := client.GetInstance(ctx, instanceName)
		if err != nil {
			// Ignore errors, keep polling
			time.Sleep(pollInterval)
			continue
		}

		if instance.State == targetState {
			return instance, nil
		}

		// Check if instance is in a terminal error state
		if instance.State == "terminated" || instance.State == "terminating" {
			return nil, fmt.Errorf("instance entered terminal state: %s", instance.State)
		}

		time.Sleep(pollInterval)
	}

	return nil, fmt.Errorf("instance %s did not reach state %s within %v", instanceName, targetState, timeout)
}

// TestBackup_MultipleBackups validates multiple backups from same instance (retention policy)
// Flow:
// 1. Launch instance
// 2. Create backup 1 ("backup-v1")
// 3. Create backup 2 ("backup-v2")
// 4. Create backup 3 ("backup-v3")
// 5. List backups for instance
// 6. Verify all 3 backups exist
// 7. Delete oldest backup (v1)
// 8. Verify only v2 and v3 remain
func TestBackup_MultipleBackups(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Setup
	ctx := context.Background()
	testCtx := integration.NewTestContext(t)
	registry := fixtures.NewFixtureRegistry(t, testCtx.Client)

	t.Log("=== Test 3: TestBackup_MultipleBackups ===")

	// Step 1: Launch instance
	t.Log("Step 1: Launching test instance...")
	instanceName := fmt.Sprintf("multi-backup-instance-%d", time.Now().Unix())
	instance, err := fixtures.CreateTestInstance(t, registry, fixtures.CreateTestInstanceOptions{
		Name:     instanceName,
		Template: "Ubuntu 22.04 Server",
		Size:     "S",
	})
	require.NoError(t, err, "Failed to create test instance")
	require.NotNil(t, instance, "Instance should not be nil")
	t.Logf("   ✓ Instance launched: %s", instance.Name)

	// Step 2-4: Create 3 backups from same instance
	t.Log("Step 2: Creating multiple backups from same instance...")
	backupNames := []string{
		fmt.Sprintf("backup-v1-%s", instanceName),
		fmt.Sprintf("backup-v2-%s", instanceName),
		fmt.Sprintf("backup-v3-%s", instanceName),
	}

	for i, backupName := range backupNames {
		t.Logf("   Creating backup %d/3: %s", i+1, backupName)
		backup, err := fixtures.CreateTestBackup(t, registry, fixtures.CreateTestBackupOptions{
			Name:        backupName,
			InstanceID:  instanceName,
			Description: fmt.Sprintf("Test backup version %d", i+1),
		})
		require.NoError(t, err, "Failed to create backup %d", i+1)
		require.NotNil(t, backup, "Backup %d should not be nil", i+1)
		require.Equal(t, "available", backup.State, "Backup %d should be available", i+1)
		t.Logf("   ✓ Backup %d/%d created: %s", i+1, 3, backupName)
	}

	// Step 5: List all backups
	t.Log("Step 3: Listing all backups...")
	backupList, err := testCtx.Client.ListBackups(ctx)
	require.NoError(t, err, "Failed to list backups")
	require.NotNil(t, backupList, "Backup list should not be nil")

	// Step 6: Verify all 3 backups exist
	t.Log("Step 4: Verifying all 3 backups exist...")
	foundBackups := 0
	for _, backup := range backupList.Backups {
		for _, backupName := range backupNames {
			if backup.BackupName == backupName {
				foundBackups++
				t.Logf("   ✓ Found backup: %s (state: %s)", backup.BackupName, backup.State)
			}
		}
	}
	assert.Equal(t, 3, foundBackups, "Should find all 3 backups in list")

	// Step 7: Delete oldest backup (v1)
	t.Log("Step 5: Deleting oldest backup (v1)...")
	deleteResult, err := testCtx.Client.DeleteBackup(ctx, backupNames[0])
	require.NoError(t, err, "Failed to delete backup v1")
	require.NotNil(t, deleteResult, "Delete result should not be nil")
	t.Logf("   ✓ Backup v1 deleted (savings: $%.2f/month)", deleteResult.StorageSavingsMonthly)

	// Step 8: Verify only v2 and v3 remain
	t.Log("Step 6: Verifying only v2 and v3 backups remain...")
	backupList, err = testCtx.Client.ListBackups(ctx)
	require.NoError(t, err, "Failed to list backups")

	foundV1 := false
	foundV2 := false
	foundV3 := false
	for _, backup := range backupList.Backups {
		if backup.BackupName == backupNames[0] {
			foundV1 = true
		}
		if backup.BackupName == backupNames[1] {
			foundV2 = true
		}
		if backup.BackupName == backupNames[2] {
			foundV3 = true
		}
	}

	assert.False(t, foundV1, "Backup v1 should not exist (deleted)")
	assert.True(t, foundV2, "Backup v2 should still exist")
	assert.True(t, foundV3, "Backup v3 should still exist")
	t.Log("   ✓ Backup retention verified: v1 deleted, v2 and v3 remain")

	t.Log("=== Test 3 Complete: Multiple backups from same instance works correctly ===")
}

// TestBackup_VerificationWorkflow validates backup integrity verification
// Flow:
// 1. Launch instance
// 2. Create backup
// 3. Call VerifyBackup API
// 4. Verify backup verification result (success/failed)
// 5. Check verification metadata (state, file count)
func TestBackup_VerificationWorkflow(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Setup
	ctx := context.Background()
	testCtx := integration.NewTestContext(t)
	registry := fixtures.NewFixtureRegistry(t, testCtx.Client)

	t.Log("=== Test 4: TestBackup_VerificationWorkflow ===")

	// Step 1: Launch instance
	t.Log("Step 1: Launching test instance...")
	instanceName := fmt.Sprintf("verify-instance-%d", time.Now().Unix())
	instance, err := fixtures.CreateTestInstance(t, registry, fixtures.CreateTestInstanceOptions{
		Name:     instanceName,
		Template: "Ubuntu 22.04 Server",
		Size:     "S",
	})
	require.NoError(t, err, "Failed to create test instance")
	require.NotNil(t, instance, "Instance should not be nil")
	t.Logf("   ✓ Instance launched: %s", instance.Name)

	// Step 2: Create backup
	t.Log("Step 2: Creating backup...")
	backupName := fmt.Sprintf("backup-%s", instanceName)
	backup, err := fixtures.CreateTestBackup(t, registry, fixtures.CreateTestBackupOptions{
		Name:        backupName,
		InstanceID:  instanceName,
		Description: "Test backup for verification workflow",
	})
	require.NoError(t, err, "Failed to create backup")
	require.NotNil(t, backup, "Backup should not be nil")
	assert.Equal(t, "available", backup.State, "Backup should be available")
	t.Logf("   ✓ Backup created: %s", backup.BackupName)

	// Step 3: Call VerifyBackup API
	t.Log("Step 3: Verifying backup integrity...")
	verifyReq := types.BackupVerifyRequest{
		BackupName: backupName,
		QuickCheck: false, // Full verification
	}

	verifyResult, err := testCtx.Client.VerifyBackup(ctx, verifyReq)
	require.NoError(t, err, "Failed to verify backup")
	require.NotNil(t, verifyResult, "Verify result should not be nil")
	t.Logf("   ✓ Verification initiated")

	// Step 4: Verify backup verification result
	t.Log("Step 4: Checking verification result...")
	assert.Equal(t, "valid", verifyResult.VerificationState, "Backup should be valid")
	assert.Equal(t, 0, verifyResult.CorruptFileCount, "Should have no corrupt files")
	t.Logf("   ✓ Verification successful: backup is valid")

	// Step 5: Check verification metadata
	t.Log("Step 5: Checking verification metadata...")
	assert.Greater(t, verifyResult.CheckedFileCount, 0, "Should have verified some files")
	assert.Greater(t, verifyResult.VerifiedBytes, int64(0), "Should have some backup size")
	t.Logf("   ✓ Verification metadata: %d files, %d bytes", verifyResult.CheckedFileCount, verifyResult.VerifiedBytes)

	t.Log("=== Test 4 Complete: Backup verification workflow works correctly ===")
}

// TestBackup_InstanceLifecycle validates backup behavior during instance operations
// Flow:
// 1. Launch instance
// 2. Create backup while instance running
// 3. Stop instance
// 4. Create second backup while instance stopped
// 5. Terminate original instance
// 6. Verify backups still exist after instance termination
// 7. Verify both backups are accessible
func TestBackup_InstanceLifecycle(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Setup
	ctx := context.Background()
	testCtx := integration.NewTestContext(t)
	registry := fixtures.NewFixtureRegistry(t, testCtx.Client)

	t.Log("=== Test 5: TestBackup_InstanceLifecycle ===")

	// Step 1: Launch instance
	t.Log("Step 1: Launching test instance...")
	instanceName := fmt.Sprintf("lifecycle-instance-%d", time.Now().Unix())
	instance, err := fixtures.CreateTestInstance(t, registry, fixtures.CreateTestInstanceOptions{
		Name:     instanceName,
		Template: "Ubuntu 22.04 Server",
		Size:     "S",
	})
	require.NoError(t, err, "Failed to create test instance")
	require.NotNil(t, instance, "Instance should not be nil")
	t.Logf("   ✓ Instance launched: %s (state: %s)", instance.Name, instance.State)

	// Step 2: Create backup while instance running
	t.Log("Step 2: Creating backup while instance running...")
	backup1Name := fmt.Sprintf("backup-running-%s", instanceName)
	backup1, err := fixtures.CreateTestBackup(t, registry, fixtures.CreateTestBackupOptions{
		Name:        backup1Name,
		InstanceID:  instanceName,
		Description: "Backup created while instance running",
	})
	require.NoError(t, err, "Failed to create backup from running instance")
	require.NotNil(t, backup1, "Backup 1 should not be nil")
	assert.Equal(t, "available", backup1.State, "Backup 1 should be available")
	t.Logf("   ✓ Backup created from running instance: %s", backup1.BackupName)

	// Step 3: Stop instance
	t.Log("Step 3: Stopping instance...")
	err = testCtx.Client.StopInstance(ctx, instanceName)
	require.NoError(t, err, "Failed to stop instance")

	// Wait for instance to be stopped
	stoppedInstance, err := waitForInstanceState(ctx, testCtx.Client, instanceName, "stopped", 5*time.Minute)
	require.NoError(t, err, "Instance did not reach stopped state")
	assert.Equal(t, "stopped", stoppedInstance.State, "Instance should be stopped")
	t.Logf("   ✓ Instance stopped: %s", instanceName)

	// Step 4: Create second backup while instance stopped
	t.Log("Step 4: Creating backup while instance stopped...")
	backup2Name := fmt.Sprintf("backup-stopped-%s", instanceName)
	backup2, err := fixtures.CreateTestBackup(t, registry, fixtures.CreateTestBackupOptions{
		Name:        backup2Name,
		InstanceID:  instanceName,
		Description: "Backup created while instance stopped",
	})
	require.NoError(t, err, "Failed to create backup from stopped instance")
	require.NotNil(t, backup2, "Backup 2 should not be nil")
	assert.Equal(t, "available", backup2.State, "Backup 2 should be available")
	t.Logf("   ✓ Backup created from stopped instance: %s", backup2.BackupName)

	// Step 5: Terminate original instance
	t.Log("Step 5: Terminating instance...")
	err = testCtx.Client.DeleteInstance(ctx, instanceName)
	require.NoError(t, err, "Failed to delete instance")
	t.Logf("   ✓ Instance deleted: %s", instanceName)

	// Step 6: Verify backups still exist after instance termination
	t.Log("Step 6: Verifying backups persist after instance termination...")
	backupList, err := testCtx.Client.ListBackups(ctx)
	require.NoError(t, err, "Failed to list backups")

	foundBackup1 := false
	foundBackup2 := false
	for _, backup := range backupList.Backups {
		if backup.BackupName == backup1Name {
			foundBackup1 = true
		}
		if backup.BackupName == backup2Name {
			foundBackup2 = true
		}
	}

	assert.True(t, foundBackup1, "Backup 1 should still exist after instance termination")
	assert.True(t, foundBackup2, "Backup 2 should still exist after instance termination")
	t.Log("   ✓ Both backups persist after instance termination")

	// Step 7: Verify both backups are accessible
	t.Log("Step 7: Verifying backup accessibility...")
	backup1Info, err := testCtx.Client.GetBackup(ctx, backup1Name)
	require.NoError(t, err, "Failed to get backup 1")
	assert.Equal(t, "available", backup1Info.State, "Backup 1 should be available")

	backup2Info, err := testCtx.Client.GetBackup(ctx, backup2Name)
	require.NoError(t, err, "Failed to get backup 2")
	assert.Equal(t, "available", backup2Info.State, "Backup 2 should be available")
	t.Log("   ✓ Both backups are accessible and available")

	t.Log("=== Test 5 Complete: Backup persists through instance lifecycle ===")
}

// TestBackup_CostTracking validates backup cost calculations
// Flow:
// 1. Launch instance with specific volume size
// 2. Create backup
// 3. Verify backup includes storage cost estimate
// 4. Delete backup
// 5. Verify deletion returns monthly savings
func TestBackup_CostTracking(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Setup
	ctx := context.Background()
	testCtx := integration.NewTestContext(t)
	registry := fixtures.NewFixtureRegistry(t, testCtx.Client)

	t.Log("=== Test 6: TestBackup_CostTracking ===")

	// Step 1: Launch instance with specific volume size
	t.Log("Step 1: Launching test instance...")
	instanceName := fmt.Sprintf("cost-instance-%d", time.Now().Unix())
	instance, err := fixtures.CreateTestInstance(t, registry, fixtures.CreateTestInstanceOptions{
		Name:     instanceName,
		Template: "Ubuntu 22.04 Server",
		Size:     "M", // Medium size for predictable cost calculations
	})
	require.NoError(t, err, "Failed to create test instance")
	require.NotNil(t, instance, "Instance should not be nil")
	t.Logf("   ✓ Instance launched: %s", instance.Name)

	// Step 2: Create backup
	t.Log("Step 2: Creating backup...")
	backupName := fmt.Sprintf("backup-%s", instanceName)
	backup, err := fixtures.CreateTestBackup(t, registry, fixtures.CreateTestBackupOptions{
		Name:        backupName,
		InstanceID:  instanceName,
		Description: "Test backup for cost tracking",
	})
	require.NoError(t, err, "Failed to create backup")
	require.NotNil(t, backup, "Backup should not be nil")
	t.Logf("   ✓ Backup created: %s", backup.BackupName)

	// Step 3: Verify backup includes storage cost estimate
	t.Log("Step 3: Verifying storage cost estimate...")
	assert.Greater(t, backup.StorageCostMonthly, 0.0, "Storage cost should be positive")
	assert.Greater(t, backup.SizeBytes, int64(0), "Backup size should be positive")

	// Calculate expected cost (AWS snapshot pricing ~$0.05/GB/month)
	expectedCostMin := float64(backup.SizeBytes) / (1024 * 1024 * 1024) * 0.04 // $0.04/GB minimum
	expectedCostMax := float64(backup.SizeBytes) / (1024 * 1024 * 1024) * 0.06 // $0.06/GB maximum

	assert.GreaterOrEqual(t, backup.StorageCostMonthly, expectedCostMin, "Cost should be at least $0.04/GB/month")
	assert.LessOrEqual(t, backup.StorageCostMonthly, expectedCostMax, "Cost should be at most $0.06/GB/month")
	t.Logf("   ✓ Storage cost: $%.4f/month for %d bytes", backup.StorageCostMonthly, backup.SizeBytes)

	// Step 4: Delete backup
	t.Log("Step 4: Deleting backup...")
	deleteResult, err := testCtx.Client.DeleteBackup(ctx, backupName)
	require.NoError(t, err, "Failed to delete backup")
	require.NotNil(t, deleteResult, "Delete result should not be nil")
	t.Logf("   ✓ Backup deleted: %s", backupName)

	// Step 5: Verify deletion returns monthly savings
	t.Log("Step 5: Verifying deletion savings...")
	assert.Greater(t, deleteResult.StorageSavingsMonthly, 0.0, "Storage savings should be positive")
	// Savings should approximately match the original cost
	assert.InDelta(t, backup.StorageCostMonthly, deleteResult.StorageSavingsMonthly, backup.StorageCostMonthly*0.01,
		"Savings should approximately match original cost")
	t.Logf("   ✓ Storage savings: $%.4f/month", deleteResult.StorageSavingsMonthly)

	t.Log("=== Test 6 Complete: Backup cost tracking works correctly ===")
}

// TestBackup_ConcurrentBackups validates concurrent backup creation (stress test)
// Flow:
// 1. Launch 2 instances concurrently
// 2. Create backups from both instances simultaneously
// 3. Wait for both backups to complete
// 4. List all backups
// 5. Verify both backups exist
// 6. Cleanup handled by registry
func TestBackup_ConcurrentBackups(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Setup
	ctx := context.Background()
	testCtx := integration.NewTestContext(t)
	registry := fixtures.NewFixtureRegistry(t, testCtx.Client)

	t.Log("=== Test 7: TestBackup_ConcurrentBackups ===")

	// Step 1: Launch 2 instances concurrently
	t.Log("Step 1: Launching 2 instances concurrently...")
	timestamp := time.Now().Unix()
	instance1Name := fmt.Sprintf("concurrent-1-%d", timestamp)
	instance2Name := fmt.Sprintf("concurrent-2-%d", timestamp)

	// Launch instances sequentially (fixture manages concurrency with slot manager)
	instance1, err := fixtures.CreateTestInstance(t, registry, fixtures.CreateTestInstanceOptions{
		Name:     instance1Name,
		Template: "Ubuntu 22.04 Server",
		Size:     "S",
	})
	require.NoError(t, err, "Failed to create instance 1")
	t.Logf("   ✓ Instance 1 launched: %s", instance1.Name)

	instance2, err := fixtures.CreateTestInstance(t, registry, fixtures.CreateTestInstanceOptions{
		Name:     instance2Name,
		Template: "Ubuntu 22.04 Server",
		Size:     "S",
	})
	require.NoError(t, err, "Failed to create instance 2")
	t.Logf("   ✓ Instance 2 launched: %s", instance2.Name)

	// Step 2: Create backups from both instances simultaneously
	t.Log("Step 2: Creating backups from both instances...")
	backup1Name := fmt.Sprintf("backup-%s", instance1Name)
	backup2Name := fmt.Sprintf("backup-%s", instance2Name)

	// Create backups concurrently using goroutines
	type backupResult struct {
		backup *types.BackupInfo
		err    error
		name   string
	}

	results := make(chan backupResult, 2)

	// Start backup 1
	go func() {
		backup, err := fixtures.CreateTestBackup(t, registry, fixtures.CreateTestBackupOptions{
			Name:        backup1Name,
			InstanceID:  instance1Name,
			Description: "Concurrent backup test 1",
		})
		results <- backupResult{backup: backup, err: err, name: backup1Name}
	}()

	// Start backup 2
	go func() {
		backup, err := fixtures.CreateTestBackup(t, registry, fixtures.CreateTestBackupOptions{
			Name:        backup2Name,
			InstanceID:  instance2Name,
			Description: "Concurrent backup test 2",
		})
		results <- backupResult{backup: backup, err: err, name: backup2Name}
	}()

	// Step 3: Wait for both backups to complete
	t.Log("Step 3: Waiting for both backups to complete...")
	var backup1, backup2 *types.BackupInfo
	for i := 0; i < 2; i++ {
		result := <-results
		require.NoError(t, result.err, "Backup %s failed", result.name)
		require.NotNil(t, result.backup, "Backup %s should not be nil", result.name)
		assert.Equal(t, "available", result.backup.State, "Backup %s should be available", result.name)

		if result.name == backup1Name {
			backup1 = result.backup
			t.Logf("   ✓ Backup 1 completed: %s", backup1.BackupName)
		} else {
			backup2 = result.backup
			t.Logf("   ✓ Backup 2 completed: %s", backup2.BackupName)
		}
	}

	// Step 4: List all backups
	t.Log("Step 4: Listing all backups...")
	backupList, err := testCtx.Client.ListBackups(ctx)
	require.NoError(t, err, "Failed to list backups")

	// Step 5: Verify both backups exist
	t.Log("Step 5: Verifying both backups exist...")
	foundBackup1 := false
	foundBackup2 := false
	for _, backup := range backupList.Backups {
		if backup.BackupName == backup1Name {
			foundBackup1 = true
		}
		if backup.BackupName == backup2Name {
			foundBackup2 = true
		}
	}

	assert.True(t, foundBackup1, "Backup 1 should exist in list")
	assert.True(t, foundBackup2, "Backup 2 should exist in list")
	t.Log("   ✓ Both backups confirmed in backup list")

	// Verify no naming conflicts
	assert.NotEqual(t, backup1.BackupID, backup2.BackupID, "Backups should have different IDs")
	t.Log("   ✓ No naming conflicts detected")

	// Step 6: Cleanup handled automatically by registry
	t.Log("=== Test 7 Complete: Concurrent backup operations work correctly ===")
}
