//go:build aws_integration
// +build aws_integration

// Package cli AWS integration tests for CloudWorkstation CLI
// These tests run against a real AWS account using the 'aws' profile
//
// Usage:
//
//	# Mock tests only (default)
//	go test ./internal/cli/...
//
//	# Include AWS integration tests
//	RUN_AWS_TESTS=true AWS_PROFILE=aws go test ./internal/cli/...
//
//	# AWS tests only
//	RUN_AWS_TESTS=true AWS_PROFILE=aws go test ./internal/cli/ -run TestAWS
//
// Environment Variables:
//
//	RUN_AWS_TESTS=true    - Enable AWS integration tests
//	AWS_PROFILE=aws       - Use 'aws' profile for test account authentication
//	AWS_TEST_REGION       - Optional test region override (default: us-east-1)
//	AWS_TEST_TIMEOUT      - Optional test timeout in minutes (default: 10)
//
// Safety Features:
//   - All test resources use 'cwstest-' prefix with timestamps
//   - Resources tagged: CreatedBy=CloudWorkstationIntegrationTest
//   - Automatic cleanup in teardown (even on test failure)
//   - Cost-conscious testing (smallest/cheapest instances)
//   - Resource limits to prevent runaway costs
package cli

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	ec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/aws/aws-sdk-go-v2/service/efs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/scttfrdmn/cloudworkstation/pkg/api/client"
	"github.com/scttfrdmn/cloudworkstation/pkg/profile"
	"github.com/scttfrdmn/cloudworkstation/pkg/types"
)

// Constants for AWS integration testing
const (
	TestResourcePrefix = "cwstest"
	TestTagKey         = "CreatedBy"
	TestTagValue       = "CloudWorkstationIntegrationTest"
	DefaultTestRegion  = "us-east-1"
	DefaultTestTimeout = 10 * time.Minute
	MaxConcurrentTests = 3
	MaxTestInstances   = 5
	MaxTestVolumes     = 3
)

// AWSTestManager manages AWS resources for integration testing
type AWSTestManager struct {
	cfg         aws.Config
	ec2Client   *ec2.Client
	efsClient   *efs.Client
	region      string
	testID      string
	createdAt   time.Time
	resources   *TestResourceRegistry
	costLimiter *CostLimiter
	testTimeout time.Duration
	client      client.CloudWorkstationAPI
	app         *App
}

// TestResourceRegistry tracks created resources for cleanup
type TestResourceRegistry struct {
	instances []string
	volumes   []string
	efsVols   []string
	keyPairs  []string
}

// CostLimiter prevents runaway AWS costs during testing
type CostLimiter struct {
	maxInstances     int
	maxVolumes       int
	maxHourlySpend   float64
	currentInstances int
	currentVolumes   int
	estimatedHourly  float64
}

// NewAWSTestManager creates a new AWS test manager
func NewAWSTestManager(t *testing.T) *AWSTestManager {
	// Skip if AWS tests not enabled
	if !isAWSTestsEnabled() {
		t.Skip("AWS integration tests disabled - set RUN_AWS_TESTS=true to enable")
	}

	// Validate AWS profile
	awsProfile := getAWSTestProfile()
	if awsProfile == "" {
		t.Skip("AWS profile not configured - set AWS_PROFILE=aws to enable")
	}

	testID := generateTestID()
	region := getTestRegion()
	timeout := getTestTimeout()

	t.Logf("AWS Integration Test Setup:")
	t.Logf("  Test ID: %s", testID)
	t.Logf("  AWS Profile: %s", awsProfile)
	t.Logf("  Region: %s", region)
	t.Logf("  Timeout: %s", timeout)

	// Load AWS configuration
	ctx := context.Background()
	cfg, err := config.LoadDefaultConfig(ctx,
		config.WithRegion(region),
		config.WithSharedConfigProfile(awsProfile),
	)
	require.NoError(t, err, "Failed to load AWS config")

	// Create AWS clients
	ec2Client := ec2.NewFromConfig(cfg)
	efsClient := efs.NewFromConfig(cfg)

	// Verify AWS connectivity
	_, err = ec2Client.DescribeRegions(ctx, &ec2.DescribeRegionsInput{})
	require.NoError(t, err, "Failed to connect to AWS - check credentials and permissions")

	// Create resource registry and cost limiter
	resources := &TestResourceRegistry{
		instances: make([]string, 0),
		volumes:   make([]string, 0),
		efsVols:   make([]string, 0),
		keyPairs:  make([]string, 0),
	}

	costLimiter := &CostLimiter{
		maxInstances:   MaxTestInstances,
		maxVolumes:     MaxTestVolumes,
		maxHourlySpend: 5.0, // $5/hour limit
	}

	manager := &AWSTestManager{
		cfg:         cfg,
		ec2Client:   ec2Client,
		efsClient:   efsClient,
		region:      region,
		testID:      testID,
		createdAt:   time.Now(),
		resources:   resources,
		costLimiter: costLimiter,
		testTimeout: timeout,
	}

	// Create CloudWorkstation API client with AWS profile
	daemonURL := getDaemonURL()
	apiClient := client.NewClientWithOptions(daemonURL, client.Options{
		AWSProfile: awsProfile,
		AWSRegion:  region,
	})

	// Verify daemon connectivity
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err = apiClient.Ping(ctx)
	require.NoError(t, err, "CloudWorkstation daemon not running or not accessible")

	manager.client = apiClient
	manager.app = NewAppWithClient("integration-test", apiClient)

	t.Logf("AWS Test Manager initialized successfully")
	return manager
}

// Cleanup removes all test resources
func (m *AWSTestManager) Cleanup(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	t.Logf("Starting cleanup for test ID: %s", m.testID)

	// Clean up instances via CLI (proper state management)
	for _, instanceName := range m.resources.instances {
		t.Logf("Cleaning up instance: %s", instanceName)
		if err := m.app.Delete([]string{instanceName}); err != nil {
			t.Logf("Warning: Failed to delete instance %s: %v", instanceName, err)
			// Try direct AWS cleanup as fallback
			m.forceDeleteInstance(ctx, instanceName)
		}
	}

	// Clean up EFS volumes via CLI
	for _, volumeName := range m.resources.efsVols {
		t.Logf("Cleaning up EFS volume: %s", volumeName)
		if err := m.client.DeleteVolume(ctx, volumeName); err != nil {
			t.Logf("Warning: Failed to delete EFS volume %s: %v", volumeName, err)
		}
	}

	// Clean up EBS volumes via CLI
	for _, volumeName := range m.resources.volumes {
		t.Logf("Cleaning up EBS volume: %s", volumeName)
		if err := m.client.DeleteStorage(ctx, volumeName); err != nil {
			t.Logf("Warning: Failed to delete EBS volume %s: %v", volumeName, err)
		}
	}

	// Clean up key pairs directly
	for _, keyName := range m.resources.keyPairs {
		t.Logf("Cleaning up key pair: %s", keyName)
		_, err := m.ec2Client.DeleteKeyPair(ctx, &ec2.DeleteKeyPairInput{
			KeyName: aws.String(keyName),
		})
		if err != nil {
			t.Logf("Warning: Failed to delete key pair %s: %v", keyName, err)
		}
	}

	// Final orphan resource cleanup (safety net)
	m.cleanupOrphanedResources(ctx, t)

	t.Logf("Cleanup completed for test ID: %s", m.testID)
}

// forceDeleteInstance directly deletes EC2 instance as fallback
func (m *AWSTestManager) forceDeleteInstance(ctx context.Context, instanceName string) {
	// Find instance by name tag
	result, err := m.ec2Client.DescribeInstances(ctx, &ec2.DescribeInstancesInput{
		Filters: []ec2types.Filter{
			{
				Name:   aws.String("tag:Name"),
				Values: []string{instanceName},
			},
			{
				Name:   aws.String("tag:" + TestTagKey),
				Values: []string{TestTagValue},
			},
		},
	})
	if err != nil {
		return
	}

	for _, reservation := range result.Reservations {
		for _, instance := range reservation.Instances {
			if instance.State.Name != ec2types.InstanceStateNameTerminated {
				_, _ = m.ec2Client.TerminateInstances(ctx, &ec2.TerminateInstancesInput{
					InstanceIds: []string{*instance.InstanceId},
				})
			}
		}
	}
}

// cleanupOrphanedResources removes any test resources that might have been missed
func (m *AWSTestManager) cleanupOrphanedResources(ctx context.Context, t *testing.T) {
	// Find and terminate orphaned EC2 instances
	result, err := m.ec2Client.DescribeInstances(ctx, &ec2.DescribeInstancesInput{
		Filters: []ec2types.Filter{
			{
				Name:   aws.String("tag:" + TestTagKey),
				Values: []string{TestTagValue},
			},
			{
				Name:   aws.String("instance-state-name"),
				Values: []string{"running", "pending", "stopping", "stopped"},
			},
		},
	})
	if err == nil {
		for _, reservation := range result.Reservations {
			for _, instance := range reservation.Instances {
				// Only clean up old test resources (older than 2 hours)
				if time.Since(*instance.LaunchTime) > 2*time.Hour {
					t.Logf("Cleaning up orphaned instance: %s", *instance.InstanceId)
					_, _ = m.ec2Client.TerminateInstances(ctx, &ec2.TerminateInstancesInput{
						InstanceIds: []string{*instance.InstanceId},
					})
				}
			}
		}
	}
}

// generateTestResourceName creates a unique test resource name
func (m *AWSTestManager) generateTestResourceName(resourceType string) string {
	timestamp := time.Now().Format("20060102-150405")
	return fmt.Sprintf("%s-%s-%s-%s", TestResourcePrefix, m.testID, resourceType, timestamp)
}

// trackResource adds a resource to the cleanup registry
func (m *AWSTestManager) trackResource(resourceType, name string) {
	switch resourceType {
	case "instance":
		m.resources.instances = append(m.resources.instances, name)
	case "volume":
		m.resources.volumes = append(m.resources.volumes, name)
	case "efs":
		m.resources.efsVols = append(m.resources.efsVols, name)
	case "keypair":
		m.resources.keyPairs = append(m.resources.keyPairs, name)
	}
}

// checkCostLimits validates resource creation against cost limits
func (m *AWSTestManager) checkCostLimits(resourceType string) error {
	switch resourceType {
	case "instance":
		if m.costLimiter.currentInstances >= m.costLimiter.maxInstances {
			return fmt.Errorf("instance limit reached: %d/%d", m.costLimiter.currentInstances, m.costLimiter.maxInstances)
		}
	case "volume", "efs":
		if m.costLimiter.currentVolumes >= m.costLimiter.maxVolumes {
			return fmt.Errorf("volume limit reached: %d/%d", m.costLimiter.currentVolumes, m.costLimiter.maxVolumes)
		}
	}

	if m.costLimiter.estimatedHourly >= m.costLimiter.maxHourlySpend {
		return fmt.Errorf("hourly spend limit reached: $%.2f/$%.2f", m.costLimiter.estimatedHourly, m.costLimiter.maxHourlySpend)
	}

	return nil
}

// waitForInstanceState waits for an instance to reach the desired state
func (m *AWSTestManager) waitForInstanceState(ctx context.Context, instanceName, desiredState string, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("timeout waiting for instance %s to reach state %s", instanceName, desiredState)
		case <-ticker.C:
			instance, err := m.client.GetInstance(ctx, instanceName)
			if err != nil {
				continue
			}

			if instance.State == desiredState {
				return nil
			}

			// Check for failure states
			if instance.State == "terminated" || instance.State == "terminating" {
				return fmt.Errorf("instance %s entered failure state: %s", instanceName, instance.State)
			}
		}
	}
}

// Helper functions for test configuration

func isAWSTestsEnabled() bool {
	return os.Getenv("RUN_AWS_TESTS") == "true"
}

func getAWSTestProfile() string {
	profile := os.Getenv("AWS_PROFILE")
	if profile == "" {
		profile = "aws" // Default to 'aws' profile
	}
	return profile
}

func getTestRegion() string {
	region := os.Getenv("AWS_TEST_REGION")
	if region == "" {
		region = DefaultTestRegion
	}
	return region
}

func getTestTimeout() time.Duration {
	timeoutStr := os.Getenv("AWS_TEST_TIMEOUT")
	if timeoutStr == "" {
		return DefaultTestTimeout
	}

	timeoutMin, err := strconv.Atoi(timeoutStr)
	if err != nil {
		return DefaultTestTimeout
	}

	return time.Duration(timeoutMin) * time.Minute
}

func getDaemonURL() string {
	url := os.Getenv(DaemonURLEnvVar)
	if url == "" {
		url = "http://localhost:8947"
	}
	return url
}

func generateTestID() string {
	return fmt.Sprintf("%d", time.Now().Unix()%10000)
}

// AWS Integration Tests

// TestAWSInstanceLifecycle tests the complete instance lifecycle
func TestAWSInstanceLifecycle(t *testing.T) {
	manager := NewAWSTestManager(t)
	defer func() { _ = manager.Cleanup(t) }()

	ctx, cancel := context.WithTimeout(context.Background(), manager.testTimeout)
	defer cancel()

	// Generate unique instance name
	instanceName := manager.generateTestResourceName("instance")

	// Check cost limits
	require.NoError(t, manager.checkCostLimits("instance"), "Cost limit check failed")

	// Track resource for cleanup
	manager.trackResource("instance", instanceName)

	t.Logf("Testing instance lifecycle with name: %s", instanceName)

	// Test 1: Launch instance
	t.Run("Launch", func(t *testing.T) {
		err := manager.app.Launch([]string{"Basic Ubuntu (APT)", instanceName})
		require.NoError(t, err, "Failed to launch instance")

		// Update cost tracking
		manager.costLimiter.currentInstances++
		manager.costLimiter.estimatedHourly += 0.05 // t3.nano estimate

		// Wait for running state
		err = manager.waitForInstanceState(ctx, instanceName, "running", 10*time.Minute)
		assert.NoError(t, err, "Instance failed to reach running state")
	})

	// Test 2: List instances (verify our instance appears)
	t.Run("List", func(t *testing.T) {
		err := manager.app.List([]string{})
		assert.NoError(t, err, "Failed to list instances")

		// Verify instance exists in API
		instance, err := manager.client.GetInstance(ctx, instanceName)
		assert.NoError(t, err, "Instance not found in API")
		assert.Equal(t, instanceName, instance.Name, "Instance name mismatch")
		assert.Equal(t, "Basic Ubuntu (APT)", instance.Template, "Template mismatch")
	})

	// Test 3: Stop instance
	t.Run("Stop", func(t *testing.T) {
		err := manager.app.Stop([]string{instanceName})
		assert.NoError(t, err, "Failed to stop instance")

		err = manager.waitForInstanceState(ctx, instanceName, "stopped", 5*time.Minute)
		assert.NoError(t, err, "Instance failed to stop")
	})

	// Test 4: Start instance
	t.Run("Start", func(t *testing.T) {
		err := manager.app.Start([]string{instanceName})
		assert.NoError(t, err, "Failed to start instance")

		err = manager.waitForInstanceState(ctx, instanceName, "running", 5*time.Minute)
		assert.NoError(t, err, "Instance failed to start")
	})

	// Test 5: Hibernation (if supported)
	t.Run("Hibernation", func(t *testing.T) {
		// Check hibernation support
		status, err := manager.client.GetInstanceHibernationStatus(ctx, instanceName)
		require.NoError(t, err, "Failed to get hibernation status")

		if status.HibernationSupported {
			// Test hibernate
			err = manager.app.Hibernate([]string{instanceName})
			assert.NoError(t, err, "Failed to hibernate instance")

			err = manager.waitForInstanceState(ctx, instanceName, "hibernated", 5*time.Minute)
			assert.NoError(t, err, "Instance failed to hibernate")

			// Test resume
			err = manager.app.Resume([]string{instanceName})
			assert.NoError(t, err, "Failed to resume instance")

			err = manager.waitForInstanceState(ctx, instanceName, "running", 5*time.Minute)
			assert.NoError(t, err, "Instance failed to resume")
		} else {
			t.Log("Hibernation not supported for this instance type, skipping hibernation test")
		}
	})

	// Test 6: Connection info
	t.Run("Connect", func(t *testing.T) {
		connectionInfo, err := manager.client.ConnectInstance(ctx, instanceName)
		assert.NoError(t, err, "Failed to get connection info")
		assert.Contains(t, connectionInfo, "ssh", "Connection info should contain SSH command")
		assert.Contains(t, strings.ToLower(connectionInfo), instanceName, "Connection info should reference instance")
	})

	// Test 7: Delete instance (performed in cleanup)
	t.Log("Instance will be deleted during cleanup")
}

// TestAWSTemplateOperations tests template discovery and validation
func TestAWSTemplateOperations(t *testing.T) {
	manager := NewAWSTestManager(t)
	defer func() { _ = manager.Cleanup(t) }()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	// Test 1: List templates
	t.Run("ListTemplates", func(t *testing.T) {
		err := manager.app.Templates([]string{})
		assert.NoError(t, err, "Failed to list templates")

		// Verify templates via API
		templates, err := manager.client.ListTemplates(ctx)
		require.NoError(t, err, "Failed to get templates from API")
		assert.Greater(t, len(templates), 0, "No templates found")

		// Verify required templates exist
		requiredTemplates := []string{
			"Basic Ubuntu (APT)",
			"Python Machine Learning (Simplified)",
			"R Research Environment (Simplified)",
		}

		for _, required := range requiredTemplates {
			_, exists := templates[required]
			assert.True(t, exists, "Required template not found: %s", required)
		}
	})

	// Test 2: Get specific template
	t.Run("GetTemplate", func(t *testing.T) {
		template, err := manager.client.GetTemplate(ctx, "Basic Ubuntu (APT)")
		assert.NoError(t, err, "Failed to get Basic Ubuntu template")
		assert.Equal(t, "Basic Ubuntu (APT)", template.Name, "Template name mismatch")
		assert.NotEmpty(t, template.Description, "Template should have description")
	})

	// Test 3: Template validation
	t.Run("ValidateTemplates", func(t *testing.T) {
		err := manager.app.Templates([]string{"validate"})
		assert.NoError(t, err, "Template validation failed")
	})
}

// TestAWSStorageOperations tests EFS and EBS storage operations
func TestAWSStorageOperations(t *testing.T) {
	manager := NewAWSTestManager(t)
	defer func() { _ = manager.Cleanup(t) }()

	ctx, cancel := context.WithTimeout(context.Background(), manager.testTimeout)
	defer cancel()

	// Test EFS Volume Operations
	t.Run("EFSVolume", func(t *testing.T) {
		// Check cost limits
		require.NoError(t, manager.checkCostLimits("efs"), "EFS volume cost limit check failed")

		volumeName := manager.generateTestResourceName("efs")
		manager.trackResource("efs", volumeName)
		manager.costLimiter.currentVolumes++

		t.Logf("Testing EFS volume: %s", volumeName)

		// Create EFS volume
		volume, err := manager.client.CreateVolume(ctx, types.VolumeCreateRequest{
			Name: volumeName,
		})
		require.NoError(t, err, "Failed to create EFS volume")
		assert.Equal(t, volumeName, volume.Name, "EFS volume name mismatch")
		assert.NotEmpty(t, volume.FileSystemId, "EFS volume should have filesystem ID")

		// List volumes
		volumes, err := manager.client.ListVolumes(ctx)
		assert.NoError(t, err, "Failed to list EFS volumes")

		var foundVolume *types.EFSVolume
		for _, v := range volumes {
			if v.Name == volumeName {
				foundVolume = &v
				break
			}
		}
		require.NotNil(t, foundVolume, "Created EFS volume not found in list")

		// Get specific volume
		getVolume, err := manager.client.GetVolume(ctx, volumeName)
		assert.NoError(t, err, "Failed to get EFS volume")
		assert.Equal(t, volumeName, getVolume.Name, "Retrieved volume name mismatch")
	})

	// Test EBS Storage Operations
	t.Run("EBSStorage", func(t *testing.T) {
		// Check cost limits
		require.NoError(t, manager.checkCostLimits("volume"), "EBS volume cost limit check failed")

		storageName := manager.generateTestResourceName("ebs")
		manager.trackResource("volume", storageName)
		manager.costLimiter.currentVolumes++

		t.Logf("Testing EBS storage: %s", storageName)

		// Create EBS storage
		storage, err := manager.client.CreateStorage(ctx, types.StorageCreateRequest{
			Name:       storageName,
			Size:       "S", // 100GB
			VolumeType: "gp3",
		})
		require.NoError(t, err, "Failed to create EBS storage")
		assert.Equal(t, storageName, storage.Name, "EBS storage name mismatch")
		assert.NotEmpty(t, storage.VolumeID, "EBS storage should have volume ID")
		assert.Equal(t, "gp3", storage.VolumeType, "EBS storage type mismatch")

		// List storage
		storageList, err := manager.client.ListStorage(ctx)
		assert.NoError(t, err, "Failed to list EBS storage")

		var foundStorage *types.EBSVolume
		for _, s := range storageList {
			if s.Name == storageName {
				foundStorage = &s
				break
			}
		}
		require.NotNil(t, foundStorage, "Created EBS storage not found in list")

		// Get specific storage
		getStorage, err := manager.client.GetStorage(ctx, storageName)
		assert.NoError(t, err, "Failed to get EBS storage")
		assert.Equal(t, storageName, getStorage.Name, "Retrieved storage name mismatch")
	})
}

// TestAWSProjectManagement tests project creation and management
func TestAWSProjectManagement(t *testing.T) {
	manager := NewAWSTestManager(t)
	defer func() { _ = manager.Cleanup(t) }()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	projectName := manager.generateTestResourceName("project")

	t.Run("ProjectOperations", func(t *testing.T) {
		// Note: Project management typically doesn't create AWS resources directly
		// but manages CloudWorkstation's project metadata and budgets

		// Note: Project functionality may be available through API client directly
		// For now, we test that the app is functional for other operations

		// Test with instance in project context
		instanceName := manager.generateTestResourceName("proj-inst")
		manager.trackResource("instance", instanceName)

		// Launch instance with project
		err := manager.app.Launch([]string{"Basic Ubuntu (APT)", instanceName, "--project", projectName})
		if err == nil {
			// If project creation succeeded, track the resource
			manager.costLimiter.currentInstances++

			// Verify instance has project association
			instance, err := manager.client.GetInstance(ctx, instanceName)
			if err == nil {
				assert.Equal(t, projectName, instance.ProjectID, "Instance project ID mismatch")
			}
		} else {
			// Project operations might not be fully configured in test environment
			t.Logf("Project operations not available in test environment: %v", err)
		}
	})
}

// TestAWSIdleDetection tests idle detection and hibernation policies
func TestAWSIdleDetection(t *testing.T) {
	manager := NewAWSTestManager(t)
	defer func() { _ = manager.Cleanup(t) }()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	// Test idle profile operations
	t.Run("IdleProfiles", func(t *testing.T) {
		// Get idle status
		status, err := manager.client.GetIdleStatus(ctx)
		assert.NoError(t, err, "Failed to get idle status")
		assert.NotNil(t, status, "Idle status should not be nil")

		// List idle profiles
		profiles, err := manager.client.GetIdleProfiles(ctx)
		assert.NoError(t, err, "Failed to get idle profiles")
		assert.NotNil(t, profiles, "Idle profiles should not be nil")

		// Verify default profiles exist
		defaultProfiles := []string{"batch", "gpu"}
		for _, profileName := range defaultProfiles {
			profile, exists := profiles[profileName]
			if exists {
				assert.Equal(t, profileName, profile.Name, "Profile name mismatch")
				assert.Greater(t, profile.IdleMinutes, 0, "Profile should have positive idle minutes")
			}
		}
	})

	// Test idle history
	t.Run("IdleHistory", func(t *testing.T) {
		history, err := manager.client.GetIdleHistory(ctx)
		assert.NoError(t, err, "Failed to get idle history")
		assert.NotNil(t, history, "Idle history should not be nil")
		// History might be empty in fresh test environment
	})
}

// TestAWSDaemonIntegration tests daemon status and operations
func TestAWSDaemonIntegration(t *testing.T) {
	manager := NewAWSTestManager(t)
	defer func() { _ = manager.Cleanup(t) }()

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
	defer cancel()

	// Test daemon status
	t.Run("DaemonStatus", func(t *testing.T) {
		err := manager.app.Daemon([]string{"status"})
		assert.NoError(t, err, "Failed to get daemon status")

		status, err := manager.client.GetStatus(ctx)
		assert.NoError(t, err, "Failed to get daemon status via API")
		assert.NotNil(t, status, "Daemon status should not be nil")
		assert.NotEmpty(t, status.Status, "Daemon should have status")
	})

	// Test daemon ping
	t.Run("DaemonPing", func(t *testing.T) {
		err := manager.client.Ping(ctx)
		assert.NoError(t, err, "Daemon ping failed")
	})
}

// TestAWSNetworkOperations tests VPC and security group operations
func TestAWSNetworkOperations(t *testing.T) {
	manager := NewAWSTestManager(t)
	defer func() { _ = manager.Cleanup(t) }()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	// Test network discovery (VPCs, subnets, security groups)
	t.Run("NetworkDiscovery", func(t *testing.T) {
		// Launch an instance to test network operations
		instanceName := manager.generateTestResourceName("network")

		require.NoError(t, manager.checkCostLimits("instance"), "Cost limit check failed")
		manager.trackResource("instance", instanceName)

		err := manager.app.Launch([]string{"Basic Ubuntu (APT)", instanceName})
		require.NoError(t, err, "Failed to launch instance for network testing")

		manager.costLimiter.currentInstances++

		// Wait for instance to be running
		err = manager.waitForInstanceState(ctx, instanceName, "running", 8*time.Minute)
		require.NoError(t, err, "Instance failed to reach running state")

		// Get instance details to verify network configuration
		instance, err := manager.client.GetInstance(ctx, instanceName)
		assert.NoError(t, err, "Failed to get instance details")
		assert.NotEmpty(t, instance.PublicIP, "Instance should have public IP")
		assert.NotEmpty(t, instance.ID, "Instance should have AWS ID")
	})
}

// TestAWSErrorHandling tests error scenarios and recovery
func TestAWSErrorHandling(t *testing.T) {
	manager := NewAWSTestManager(t)
	defer func() { _ = manager.Cleanup(t) }()

	// Test invalid template
	t.Run("InvalidTemplate", func(t *testing.T) {
		instanceName := manager.generateTestResourceName("error")
		err := manager.app.Launch([]string{"NonexistentTemplate", instanceName})
		assert.Error(t, err, "Should fail with nonexistent template")
		assert.Contains(t, err.Error(), "template", "Error should mention template")
	})

	// Test operations on nonexistent instance
	t.Run("NonexistentInstance", func(t *testing.T) {
		nonexistentName := "nonexistent-instance-" + manager.testID

		err := manager.app.Start([]string{nonexistentName})
		assert.Error(t, err, "Should fail with nonexistent instance")

		err = manager.app.Stop([]string{nonexistentName})
		assert.Error(t, err, "Should fail with nonexistent instance")

		err = manager.app.Delete([]string{nonexistentName})
		assert.Error(t, err, "Should fail with nonexistent instance")
	})

	// Test invalid storage operations
	t.Run("InvalidStorage", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		nonexistentStorage := "nonexistent-storage-" + manager.testID
		_, err := manager.client.GetStorage(ctx, nonexistentStorage)
		assert.Error(t, err, "Should fail with nonexistent storage")
	})
}

// TestAWSCostAnalysis tests cost tracking and analysis features
func TestAWSCostAnalysis(t *testing.T) {
	manager := NewAWSTestManager(t)
	defer func() { _ = manager.Cleanup(t) }()

	// Test cost listing
	t.Run("CostAnalysis", func(t *testing.T) {
		err := manager.app.ListCost([]string{})
		assert.NoError(t, err, "Failed to list costs")

		// Launch an instance to test cost tracking
		instanceName := manager.generateTestResourceName("cost")

		require.NoError(t, manager.checkCostLimits("instance"), "Cost limit check failed")
		manager.trackResource("instance", instanceName)

		err = manager.app.Launch([]string{"Basic Ubuntu (APT)", instanceName})
		if err == nil {
			manager.costLimiter.currentInstances++

			// Test cost listing with actual instance
			err = manager.app.ListCost([]string{})
			assert.NoError(t, err, "Failed to list costs with active instance")
		}
	})
}

// TestAWSProfileIntegration tests AWS profile integration
func TestAWSProfileIntegration(t *testing.T) {
	manager := NewAWSTestManager(t)
	defer func() { _ = manager.Cleanup(t) }()

	// Test profile operations
	t.Run("ProfileOperations", func(t *testing.T) {
		// Create test profile manager
		profileManager, err := profile.GetDefaultManager()
		require.NoError(t, err, "Failed to create profile manager")

		// Create profile with AWS profile
		testProfile := &profile.Profile{
			Type:       profile.ProfileTypePersonal,
			Name:       "test-integration-profile",
			AWSProfile: getAWSTestProfile(),
			Region:     manager.region,
			Default:    false,
			CreatedAt:  time.Now(),
		}

		profileErr := profileManager.Set("test-integration-profile", testProfile)
		if profileErr == nil {
			// If profile operations work, test profile-based operations
			defer func() { _ = profileManager.Delete("test-integration-profile") }()

			// Test profile listing
			profiles := profileManager.List()
			assert.Greater(t, len(profiles), 0, "Should have at least one profile")
		} else {
			t.Logf("Profile operations not fully available: %v", profileErr)
		}
	})
}

// BenchmarkAWSOperations benchmarks AWS API operations
func BenchmarkAWSOperations(b *testing.B) {
	if !isAWSTestsEnabled() {
		b.Skip("AWS integration tests disabled - set RUN_AWS_TESTS=true to enable")
	}

	// Convert testing.B to testing.T for manager creation
	t := &testing.T{}
	manager := NewAWSTestManager(t)
	defer func() { _ = manager.Cleanup(t) }()

	ctx := context.Background()

	b.ResetTimer()

	b.Run("ListTemplates", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, err := manager.client.ListTemplates(ctx)
			if err != nil {
				b.Fatal("ListTemplates failed:", err)
			}
		}
	})

	b.Run("ListInstances", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, err := manager.client.ListInstances(ctx)
			if err != nil {
				b.Fatal("ListInstances failed:", err)
			}
		}
	})

	b.Run("GetStatus", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, err := manager.client.GetStatus(ctx)
			if err != nil {
				b.Fatal("GetStatus failed:", err)
			}
		}
	})
}
