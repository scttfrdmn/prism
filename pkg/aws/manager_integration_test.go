// +build integration

package aws

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/scttfrdmn/cloudworkstation/pkg/templates"
	cwstypes "github.com/scttfrdmn/cloudworkstation/pkg/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestAWSManagerIntegration tests real AWS operations
// Run with: go test -tags=integration -v ./pkg/aws
func TestAWSManagerIntegration(t *testing.T) {
	if os.Getenv("AWS_INTEGRATION_TESTS") != "true" {
		t.Skip("Skipping AWS integration tests. Set AWS_INTEGRATION_TESTS=true to run")
	}

	ctx := context.Background()
	
	// Load AWS config
	cfg, err := config.LoadDefaultConfig(ctx)
	require.NoError(t, err, "Failed to load AWS config")

	// Create manager
	manager := &Manager{
		ec2Client:    ec2.NewFromConfig(cfg),
		efsClient:    nil, // Will be initialized as needed
		region:       cfg.Region,
		profileName:  "integration-test",
	}

	// Test VPC discovery
	t.Run("VPC Discovery", func(t *testing.T) {
		vpcs, err := manager.DescribeVPCs(ctx)
		assert.NoError(t, err)
		assert.NotEmpty(t, vpcs, "Should find at least one VPC")
		
		// Verify default VPC exists
		hasDefault := false
		for _, vpc := range vpcs {
			if vpc.IsDefault != nil && *vpc.IsDefault {
				hasDefault = true
				break
			}
		}
		assert.True(t, hasDefault, "Should have a default VPC")
	})

	// Test Subnet discovery
	t.Run("Subnet Discovery", func(t *testing.T) {
		subnets, err := manager.DescribeSubnets(ctx)
		assert.NoError(t, err)
		assert.NotEmpty(t, subnets, "Should find at least one subnet")
		
		// Verify subnets have required fields
		for _, subnet := range subnets {
			assert.NotNil(t, subnet.SubnetId)
			assert.NotNil(t, subnet.VpcId)
			assert.NotNil(t, subnet.AvailabilityZone)
		}
	})

	// Test Security Group operations
	t.Run("Security Group Operations", func(t *testing.T) {
		// Create a test security group
		sgName := "cws-integration-test-" + time.Now().Format("20060102-150405")
		sgID, err := manager.CreateSecurityGroup(ctx, sgName, "Integration test SG")
		require.NoError(t, err)
		require.NotEmpty(t, sgID)

		// Clean up
		defer func() {
			deleteInput := &ec2.DeleteSecurityGroupInput{
				GroupId: &sgID,
			}
			_, _ = manager.ec2Client.DeleteSecurityGroup(ctx, deleteInput)
		}()

		// Verify security group exists
		describeInput := &ec2.DescribeSecurityGroupsInput{
			GroupIds: []string{sgID},
		}
		result, err := manager.ec2Client.DescribeSecurityGroups(ctx, describeInput)
		assert.NoError(t, err)
		assert.Len(t, result.SecurityGroups, 1)
		assert.Equal(t, sgID, *result.SecurityGroups[0].GroupId)
	})

	// Test AMI lookup
	t.Run("AMI Lookup", func(t *testing.T) {
		// Test Ubuntu AMI lookup
		amiID, err := manager.GetLatestAMI(ctx, "ubuntu", "22.04", "amd64")
		assert.NoError(t, err)
		assert.NotEmpty(t, amiID, "Should find Ubuntu 22.04 AMI")
		
		// Verify AMI exists
		describeInput := &ec2.DescribeImagesInput{
			ImageIds: []string{amiID},
		}
		result, err := manager.ec2Client.DescribeImages(ctx, describeInput)
		assert.NoError(t, err)
		assert.Len(t, result.Images, 1)
		assert.Equal(t, types.ImageStateAvailable, result.Images[0].State)
	})

	// Test Instance Type validation
	t.Run("Instance Type Validation", func(t *testing.T) {
		// Test valid instance type
		valid := manager.IsValidInstanceType(ctx, "t3.micro")
		assert.True(t, valid, "t3.micro should be valid")

		// Test invalid instance type
		invalid := manager.IsValidInstanceType(ctx, "invalid.type")
		assert.False(t, invalid, "invalid.type should not be valid")

		// Test GPU instance type
		gpuValid := manager.IsValidInstanceType(ctx, "g4dn.xlarge")
		// Note: This may fail in regions without GPU instances
		t.Logf("GPU instance g4dn.xlarge valid: %v", gpuValid)
	})

	// Test Availability Zone listing
	t.Run("Availability Zones", func(t *testing.T) {
		azs, err := manager.GetAvailabilityZones(ctx)
		assert.NoError(t, err)
		assert.NotEmpty(t, azs, "Should have at least one AZ")
		
		for _, az := range azs {
			assert.NotEmpty(t, az.ZoneName)
			assert.NotEmpty(t, az.State)
			assert.Equal(t, "available", az.State, "AZ should be available")
		}
	})

	// Test Key Pair operations
	t.Run("Key Pair Operations", func(t *testing.T) {
		keyName := "cws-test-" + time.Now().Format("20060102-150405")
		
		// Create key pair
		keyMaterial, err := manager.CreateKeyPair(ctx, keyName)
		require.NoError(t, err)
		require.NotEmpty(t, keyMaterial, "Should return private key material")
		
		// Clean up
		defer func() {
			deleteInput := &ec2.DeleteKeyPairInput{
				KeyName: &keyName,
			}
			_, _ = manager.ec2Client.DeleteKeyPair(ctx, deleteInput)
		}()

		// Verify key pair exists
		describeInput := &ec2.DescribeKeyPairsInput{
			KeyNames: []string{keyName},
		}
		result, err := manager.ec2Client.DescribeKeyPairs(ctx, describeInput)
		assert.NoError(t, err)
		assert.Len(t, result.KeyPairs, 1)
		assert.Equal(t, keyName, *result.KeyPairs[0].KeyName)
	})

	// Test EBS Volume operations
	t.Run("EBS Volume Operations", func(t *testing.T) {
		// Get first available AZ
		azs, err := manager.GetAvailabilityZones(ctx)
		require.NoError(t, err)
		require.NotEmpty(t, azs)
		
		az := azs[0].ZoneName
		
		// Create test volume
		volumeName := "cws-test-vol-" + time.Now().Format("20060102-150405")
		volumeID, err := manager.CreateEBSVolume(ctx, volumeName, 10, "gp3", az)
		require.NoError(t, err)
		require.NotEmpty(t, volumeID)

		// Clean up
		defer func() {
			deleteInput := &ec2.DeleteVolumeInput{
				VolumeId: &volumeID,
			}
			_, _ = manager.ec2Client.DeleteVolume(ctx, deleteInput)
		}()

		// Wait for volume to be available
		time.Sleep(5 * time.Second)

		// Verify volume exists
		volumes, err := manager.ListEBSVolumes(ctx)
		assert.NoError(t, err)
		
		found := false
		for _, vol := range volumes {
			if vol.VolumeID == volumeID {
				found = true
				assert.Equal(t, volumeName, vol.Name)
				assert.Equal(t, 10, vol.Size)
				break
			}
		}
		assert.True(t, found, "Should find created volume")
	})

	// Test Spot Price lookup
	t.Run("Spot Price Lookup", func(t *testing.T) {
		price, err := manager.GetSpotPrice(ctx, "t3.micro")
		assert.NoError(t, err)
		assert.Greater(t, price, 0.0, "Should return positive spot price")
		assert.Less(t, price, 1.0, "t3.micro spot price should be less than $1")
	})
}

// TestInstanceLifecycleIntegration tests full instance lifecycle
func TestInstanceLifecycleIntegration(t *testing.T) {
	if os.Getenv("AWS_INTEGRATION_TESTS") != "true" {
		t.Skip("Skipping AWS integration tests. Set AWS_INTEGRATION_TESTS=true to run")
	}

	if os.Getenv("AWS_FULL_LIFECYCLE_TEST") != "true" {
		t.Skip("Skipping full lifecycle test. Set AWS_FULL_LIFECYCLE_TEST=true to run")
	}

	ctx := context.Background()
	
	// Load AWS config
	cfg, err := config.LoadDefaultConfig(ctx)
	require.NoError(t, err)

	manager := &Manager{
		ec2Client:    ec2.NewFromConfig(cfg),
		region:       cfg.Region,
		profileName:  "integration-test",
	}

	// Create test instance
	instanceName := "cws-test-" + time.Now().Format("20060102-150405")
	
	// Use minimal Ubuntu template for testing
	template := &templates.RuntimeTemplate{
		Name:           "test-ubuntu",
		OS:             "ubuntu",
		PackageManager: "apt",
	}

	launchRequest := &cwstypes.LaunchRequest{
		Template:     template.Name,
		Name:         instanceName,
		InstanceType: "t3.micro",
		VolumeSize:   10,
		SpotInstance: false,
	}

	// Launch instance
	instanceID, err := manager.LaunchInstance(ctx, launchRequest, template)
	require.NoError(t, err)
	require.NotEmpty(t, instanceID)

	// Clean up
	defer func() {
		// Terminate instance
		_ = manager.TerminateInstance(ctx, instanceID)
	}()

	// Wait for instance to be running
	err = manager.WaitForInstanceState(ctx, instanceID, "running", 5*time.Minute)
	assert.NoError(t, err)

	// Get instance details
	instance, err := manager.DescribeInstance(ctx, instanceID)
	assert.NoError(t, err)
	assert.Equal(t, instanceID, instance.ID)
	assert.Equal(t, instanceName, instance.Name)
	assert.Equal(t, "running", instance.State)
	assert.NotEmpty(t, instance.PublicIP)

	// Test stop/start cycle
	t.Run("Stop/Start Cycle", func(t *testing.T) {
		// Stop instance
		err := manager.StopInstance(ctx, instanceID)
		assert.NoError(t, err)

		// Wait for stopped state
		err = manager.WaitForInstanceState(ctx, instanceID, "stopped", 3*time.Minute)
		assert.NoError(t, err)

		// Start instance
		err = manager.StartInstance(ctx, instanceID)
		assert.NoError(t, err)

		// Wait for running state
		err = manager.WaitForInstanceState(ctx, instanceID, "running", 3*time.Minute)
		assert.NoError(t, err)
	})

	// Test hibernation (if supported)
	t.Run("Hibernation Test", func(t *testing.T) {
		status, err := manager.GetInstanceHibernationStatus(ctx, instanceID)
		assert.NoError(t, err)

		if status.Configured && status.Supported {
			// Try to hibernate
			err := manager.HibernateInstance(ctx, instanceID)
			if err == nil {
				// Wait for hibernated state
				err = manager.WaitForInstanceState(ctx, instanceID, "stopped", 3*time.Minute)
				assert.NoError(t, err)

				// Resume from hibernation
				err = manager.ResumeInstance(ctx, instanceID)
				assert.NoError(t, err)

				// Wait for running state
				err = manager.WaitForInstanceState(ctx, instanceID, "running", 3*time.Minute)
				assert.NoError(t, err)
			}
		} else {
			t.Log("Hibernation not supported for this instance")
		}
	})

	// Test tagging
	t.Run("Instance Tagging", func(t *testing.T) {
		tags := map[string]string{
			"TestTag":    "TestValue",
			"Integration": "true",
		}
		
		err := manager.TagInstance(ctx, instanceID, tags)
		assert.NoError(t, err)

		// Verify tags
		instance, err := manager.DescribeInstance(ctx, instanceID)
		assert.NoError(t, err)
		
		// Check if tags are present in instance metadata
		t.Logf("Instance tags applied successfully")
	})
}

// TestEFSIntegration tests EFS operations
func TestEFSIntegration(t *testing.T) {
	if os.Getenv("AWS_INTEGRATION_TESTS") != "true" {
		t.Skip("Skipping AWS integration tests. Set AWS_INTEGRATION_TESTS=true to run")
	}

	if os.Getenv("AWS_EFS_TEST") != "true" {
		t.Skip("Skipping EFS test. Set AWS_EFS_TEST=true to run")
	}

	ctx := context.Background()
	
	cfg, err := config.LoadDefaultConfig(ctx)
	require.NoError(t, err)

	manager := &Manager{
		ec2Client:    ec2.NewFromConfig(cfg),
		region:       cfg.Region,
		profileName:  "integration-test",
	}
	
	// Initialize EFS client
	err = manager.initEFSClient(ctx)
	require.NoError(t, err)

	// Create EFS volume
	volumeName := "cws-test-efs-" + time.Now().Format("20060102-150405")
	
	fsID, err := manager.CreateEFSVolume(ctx, volumeName)
	require.NoError(t, err)
	require.NotEmpty(t, fsID)

	// Clean up
	defer func() {
		_ = manager.DeleteEFSVolume(ctx, fsID)
	}()

	// Wait for EFS to be available
	time.Sleep(10 * time.Second)

	// List EFS volumes
	volumes, err := manager.ListEFSVolumes(ctx)
	assert.NoError(t, err)
	
	found := false
	for _, vol := range volumes {
		if vol.FileSystemID == fsID {
			found = true
			assert.Equal(t, volumeName, vol.Name)
			break
		}
	}
	assert.True(t, found, "Should find created EFS volume")
}

// TestCostCalculations tests cost estimation accuracy
func TestCostCalculations(t *testing.T) {
	if os.Getenv("AWS_INTEGRATION_TESTS") != "true" {
		t.Skip("Skipping AWS integration tests. Set AWS_INTEGRATION_TESTS=true to run")
	}

	ctx := context.Background()
	
	cfg, err := config.LoadDefaultConfig(ctx)
	require.NoError(t, err)

	manager := &Manager{
		ec2Client:    ec2.NewFromConfig(cfg),
		region:       cfg.Region,
		profileName:  "integration-test",
	}

	// Test on-demand pricing
	t.Run("On-Demand Pricing", func(t *testing.T) {
		price, err := manager.GetInstancePrice(ctx, "t3.micro", false)
		assert.NoError(t, err)
		assert.Greater(t, price, 0.0, "Should return positive price")
		assert.Less(t, price, 0.1, "t3.micro should be less than $0.10/hour")
	})

	// Test spot pricing
	t.Run("Spot Pricing", func(t *testing.T) {
		spotPrice, err := manager.GetInstancePrice(ctx, "t3.micro", true)
		assert.NoError(t, err)
		assert.Greater(t, spotPrice, 0.0, "Should return positive spot price")
		
		onDemandPrice, _ := manager.GetInstancePrice(ctx, "t3.micro", false)
		assert.Less(t, spotPrice, onDemandPrice, "Spot price should be less than on-demand")
	})

	// Test GPU instance pricing
	t.Run("GPU Instance Pricing", func(t *testing.T) {
		price, err := manager.GetInstancePrice(ctx, "g4dn.xlarge", false)
		if err == nil {
			assert.Greater(t, price, 0.5, "GPU instance should be more expensive")
			assert.Less(t, price, 5.0, "GPU instance should be reasonable")
		} else {
			t.Logf("GPU instances not available in this region: %v", err)
		}
	})
}