//go:build aws_integration
// +build aws_integration

// Package cli AWS integration test helper utilities
package cli

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	ec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/aws/aws-sdk-go-v2/service/efs"
	"github.com/stretchr/testify/require"

	"github.com/scttfrdmn/cloudworkstation/pkg/api/client"
	"github.com/scttfrdmn/cloudworkstation/pkg/types"
)

// AWSTestHelpers provides utility methods for AWS integration testing
type AWSTestHelpers struct {
	config    *AWSTestConfig
	ec2Client *ec2.Client
	efsClient *efs.Client
	apiClient client.CloudWorkstationAPI
	testID    string
}

// NewAWSTestHelpers creates a new AWS test helpers instance
func NewAWSTestHelpers(config *AWSTestConfig, ec2Client *ec2.Client, efsClient *efs.Client, apiClient client.CloudWorkstationAPI, testID string) *AWSTestHelpers {
	return &AWSTestHelpers{
		config:    config,
		ec2Client: ec2Client,
		efsClient: efsClient,
		apiClient: apiClient,
		testID:    testID,
	}
}

// WaitForInstanceState waits for an instance to reach the specified state
func (h *AWSTestHelpers) WaitForInstanceState(ctx context.Context, t *testing.T, instanceName, desiredState string, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()

	t.Logf("Waiting for instance %s to reach state %s (timeout: %s)", instanceName, desiredState, timeout)

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("timeout waiting for instance %s to reach state %s after %s", instanceName, desiredState, timeout)

		case <-ticker.C:
			instance, err := h.apiClient.GetInstance(ctx, instanceName)
			if err != nil {
				t.Logf("Error getting instance %s state: %v", instanceName, err)
				continue
			}

			t.Logf("Instance %s current state: %s", instanceName, instance.State)

			if instance.State == desiredState {
				t.Logf("Instance %s reached desired state: %s", instanceName, desiredState)
				return nil
			}

			// Check for failure states
			failureStates := []string{"terminated", "terminating", "shutting-down"}
			for _, failState := range failureStates {
				if instance.State == failState && desiredState != failState {
					return fmt.Errorf("instance %s entered failure state: %s", instanceName, instance.State)
				}
			}
		}
	}
}

// WaitForVolumeState waits for an EFS volume to reach the specified state
func (h *AWSTestHelpers) WaitForVolumeState(ctx context.Context, t *testing.T, volumeName, desiredState string, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	t.Logf("Waiting for volume %s to reach state %s (timeout: %s)", volumeName, desiredState, timeout)

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("timeout waiting for volume %s to reach state %s after %s", volumeName, desiredState, timeout)

		case <-ticker.C:
			volume, err := h.apiClient.GetVolume(ctx, volumeName)
			if err != nil {
				t.Logf("Error getting volume %s state: %v", volumeName, err)
				continue
			}

			t.Logf("Volume %s current state: %s", volumeName, volume.State)

			if volume.State == desiredState {
				t.Logf("Volume %s reached desired state: %s", volumeName, desiredState)
				return nil
			}
		}
	}
}

// WaitForStorageState waits for an EBS volume to reach the specified state
func (h *AWSTestHelpers) WaitForStorageState(ctx context.Context, t *testing.T, storageName, desiredState string, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	t.Logf("Waiting for storage %s to reach state %s (timeout: %s)", storageName, desiredState, timeout)

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("timeout waiting for storage %s to reach state %s after %s", storageName, desiredState, timeout)

		case <-ticker.C:
			storage, err := h.apiClient.GetStorage(ctx, storageName)
			if err != nil {
				t.Logf("Error getting storage %s state: %v", storageName, err)
				continue
			}

			t.Logf("Storage %s current state: %s", storageName, storage.State)

			if storage.State == desiredState {
				t.Logf("Storage %s reached desired state: %s", storageName, desiredState)
				return nil
			}
		}
	}
}

// VerifyInstanceProperties validates instance properties against expected values
func (h *AWSTestHelpers) VerifyInstanceProperties(t *testing.T, instance *types.Instance, expected *types.LaunchRequest) {
	require.NotNil(t, instance, "Instance should not be nil")
	require.Equal(t, expected.Name, instance.Name, "Instance name mismatch")
	require.Equal(t, expected.Template, instance.Template, "Template mismatch")
	require.NotEmpty(t, instance.ID, "Instance ID should not be empty")
	require.NotEmpty(t, instance.State, "Instance state should not be empty")

	if instance.State == "running" {
		require.NotEmpty(t, instance.PublicIP, "Running instance should have public IP")
	}

	require.True(t, instance.HourlyRate > 0, "Instance should have positive hourly rate")
	require.False(t, instance.LaunchTime.IsZero(), "Instance should have launch time")
}

// VerifyVolumeProperties validates EFS volume properties
func (h *AWSTestHelpers) VerifyVolumeProperties(t *testing.T, volume *types.EFSVolume, expectedName string) {
	require.NotNil(t, volume, "Volume should not be nil")
	require.Equal(t, expectedName, volume.Name, "Volume name mismatch")
	require.NotEmpty(t, volume.FileSystemId, "Volume should have filesystem ID")
	require.NotEmpty(t, volume.State, "Volume state should not be empty")
	require.False(t, volume.CreationTime.IsZero(), "Volume should have creation time")
}

// VerifyStorageProperties validates EBS storage properties
func (h *AWSTestHelpers) VerifyStorageProperties(t *testing.T, storage *types.EBSVolume, expected *types.StorageCreateRequest) {
	require.NotNil(t, storage, "Storage should not be nil")
	require.Equal(t, expected.Name, storage.Name, "Storage name mismatch")
	require.NotEmpty(t, storage.VolumeID, "Storage should have volume ID")
	require.Equal(t, expected.VolumeType, storage.VolumeType, "Storage type mismatch")
	require.NotEmpty(t, storage.State, "Storage state should not be empty")
	require.True(t, storage.SizeGB > 0, "Storage should have positive size")
	require.False(t, storage.CreationTime.IsZero(), "Storage should have creation time")

	// Verify GP3 specific properties
	if expected.VolumeType == "gp3" {
		require.True(t, storage.IOPS > 0, "GP3 storage should have positive IOPS")
		require.True(t, storage.Throughput > 0, "GP3 storage should have positive throughput")
	}
}

// CleanupOrphanedEC2Instances removes orphaned test instances
func (h *AWSTestHelpers) CleanupOrphanedEC2Instances(ctx context.Context, t *testing.T, maxAge time.Duration) {
	t.Log("Cleaning up orphaned EC2 instances...")

	result, err := h.ec2Client.DescribeInstances(ctx, &ec2.DescribeInstancesInput{
		Filters: []ec2types.Filter{
			{
				Name:   aws.String("tag:CreatedBy"),
				Values: []string{"CloudWorkstationIntegrationTest"},
			},
			{
				Name:   aws.String("instance-state-name"),
				Values: []string{"running", "pending", "stopping", "stopped"},
			},
		},
	})
	if err != nil {
		t.Logf("Error describing instances for cleanup: %v", err)
		return
	}

	var orphanedInstances []string
	for _, reservation := range result.Reservations {
		for _, instance := range reservation.Instances {
			// Only clean up instances older than maxAge
			if instance.LaunchTime != nil && time.Since(*instance.LaunchTime) > maxAge {
				orphanedInstances = append(orphanedInstances, *instance.InstanceId)
				t.Logf("Found orphaned instance: %s (age: %s)", *instance.InstanceId, time.Since(*instance.LaunchTime))
			}
		}
	}

	if len(orphanedInstances) > 0 {
		t.Logf("Terminating %d orphaned instances", len(orphanedInstances))
		_, err := h.ec2Client.TerminateInstances(ctx, &ec2.TerminateInstancesInput{
			InstanceIds: orphanedInstances,
		})
		if err != nil {
			t.Logf("Error terminating orphaned instances: %v", err)
		}
	}
}

// CleanupOrphanedEFSVolumes removes orphaned test EFS volumes
func (h *AWSTestHelpers) CleanupOrphanedEFSVolumes(ctx context.Context, t *testing.T, maxAge time.Duration) {
	t.Log("Cleaning up orphaned EFS volumes...")

	result, err := h.efsClient.DescribeFileSystems(ctx, &efs.DescribeFileSystemsInput{})
	if err != nil {
		t.Logf("Error describing file systems for cleanup: %v", err)
		return
	}

	for _, fs := range result.FileSystems {
		// Check if it's a test file system
		isTestFS := false
		for _, tag := range fs.Tags {
			if *tag.Key == "CreatedBy" && *tag.Value == "CloudWorkstationIntegrationTest" {
				isTestFS = true
				break
			}
		}

		// Only clean up test file systems older than maxAge
		if isTestFS && fs.CreationTime != nil && time.Since(*fs.CreationTime) > maxAge {
			t.Logf("Found orphaned EFS volume: %s (age: %s)", *fs.FileSystemId, time.Since(*fs.CreationTime))

			// Delete the file system
			_, err := h.efsClient.DeleteFileSystem(ctx, &efs.DeleteFileSystemInput{
				FileSystemId: fs.FileSystemId,
			})
			if err != nil {
				t.Logf("Error deleting orphaned EFS volume %s: %v", *fs.FileSystemId, err)
			}
		}
	}
}

// ValidateAWSConnectivity checks basic AWS connectivity and permissions
func (h *AWSTestHelpers) ValidateAWSConnectivity(ctx context.Context, t *testing.T) {
	t.Log("Validating AWS connectivity and permissions...")

	// Test EC2 connectivity
	t.Run("EC2Connectivity", func(t *testing.T) {
		_, err := h.ec2Client.DescribeRegions(ctx, &ec2.DescribeRegionsInput{})
		require.NoError(t, err, "Failed to connect to EC2 - check AWS credentials and permissions")
	})

	// Test EFS connectivity
	t.Run("EFSConnectivity", func(t *testing.T) {
		_, err := h.efsClient.DescribeFileSystems(ctx, &efs.DescribeFileSystemsInput{})
		require.NoError(t, err, "Failed to connect to EFS - check AWS permissions")
	})

	// Test CloudWorkstation daemon connectivity
	t.Run("DaemonConnectivity", func(t *testing.T) {
		err := h.apiClient.Ping(ctx)
		require.NoError(t, err, "Failed to connect to CloudWorkstation daemon")
	})

	t.Log("AWS connectivity validation completed successfully")
}

// GetTestInstanceTypes returns appropriate instance types for testing
func (h *AWSTestHelpers) GetTestInstanceTypes() []string {
	return []string{
		"t3.nano",  // Cheapest option
		"t3.micro", // Free tier eligible
	}
}

// GetTestVolumeTypes returns appropriate EBS volume types for testing
func (h *AWSTestHelpers) GetTestVolumeTypes() []string {
	return []string{
		"gp3", // Modern general purpose
		"gp2", // Legacy general purpose (often cheaper)
	}
}

// EstimateCost estimates the hourly cost of running test resources
func (h *AWSTestHelpers) EstimateCost(instances int, volumes int, storageGB int) float64 {
	// Conservative estimates for us-east-1
	instanceCost := float64(instances) * 0.0052          // t3.nano
	volumeCost := float64(volumes) * (0.10 / 24 / 30)    // EFS per GB-month
	storageCost := float64(storageGB) * (0.10 / 24 / 30) // GP3 per GB-month

	return instanceCost + volumeCost + storageCost
}

// GenerateTestName creates a unique test resource name
func (h *AWSTestHelpers) GenerateTestName(resourceType string) string {
	return h.config.GetResourceName(h.testID, resourceType)
}

// GetTestTags returns standard tags for test resources
func (h *AWSTestHelpers) GetTestTags() map[string]string {
	tags := h.config.GetTestTags()
	tags["TestID"] = h.testID
	tags["CreatedAt"] = time.Now().Format(time.RFC3339)
	return tags
}

// LogResourceCreation logs resource creation for tracking
func (h *AWSTestHelpers) LogResourceCreation(t *testing.T, resourceType, name, id string) {
	t.Logf("Created %s: %s (ID: %s)", resourceType, name, id)
}

// LogResourceDeletion logs resource deletion for tracking
func (h *AWSTestHelpers) LogResourceDeletion(t *testing.T, resourceType, name string) {
	t.Logf("Deleted %s: %s", resourceType, name)
}

// CheckTestLimits validates that test execution won't exceed safety limits
func (h *AWSTestHelpers) CheckTestLimits(t *testing.T, plannedInstances, plannedVolumes int) {
	require.LessOrEqual(t, plannedInstances, h.config.MaxInstances,
		"Planned instances (%d) exceed limit (%d)", plannedInstances, h.config.MaxInstances)

	require.LessOrEqual(t, plannedVolumes, h.config.MaxVolumes,
		"Planned volumes (%d) exceed limit (%d)", plannedVolumes, h.config.MaxVolumes)

	estimatedCost := h.EstimateCost(plannedInstances, plannedVolumes, 100) // Assume 100GB per volume
	require.LessOrEqual(t, estimatedCost, h.config.MaxHourlyCost,
		"Estimated hourly cost ($%.2f) exceeds limit ($%.2f)", estimatedCost, h.config.MaxHourlyCost)
}

// IsRetryableError checks if an AWS error is retryable
func (h *AWSTestHelpers) IsRetryableError(err error) bool {
	if err == nil {
		return false
	}

	errStr := strings.ToLower(err.Error())
	retryableErrors := []string{
		"throttling",
		"rate exceeded",
		"service unavailable",
		"internal error",
		"timeout",
		"connection",
	}

	for _, retryable := range retryableErrors {
		if strings.Contains(errStr, retryable) {
			return true
		}
	}

	return false
}

// RetryWithBackoff executes a function with exponential backoff for retryable errors
func (h *AWSTestHelpers) RetryWithBackoff(ctx context.Context, operation func() error, maxRetries int) error {
	backoff := 1 * time.Second

	for i := 0; i < maxRetries; i++ {
		err := operation()
		if err == nil {
			return nil
		}

		if !h.IsRetryableError(err) {
			return err // Non-retryable error
		}

		if i < maxRetries-1 { // Don't sleep on the last attempt
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(backoff):
				backoff *= 2 // Exponential backoff
			}
		}
	}

	return fmt.Errorf("operation failed after %d retries", maxRetries)
}
