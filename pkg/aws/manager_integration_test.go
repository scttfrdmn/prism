//go:build integration
// +build integration

package aws

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"os"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/aws/aws-sdk-go-v2/service/efs"
	cwstypes "github.com/scttfrdmn/prism/pkg/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/ssh"
)

// newRealAWSManager builds a Manager the same way production does (NewManager),
// honouring AWS_PROFILE / AWS_REGION from the environment.
func newRealAWSManager(t *testing.T) *Manager {
	t.Helper()
	manager, err := NewManager(ManagerOptions{
		Profile: os.Getenv("AWS_PROFILE"),
		Region:  os.Getenv("AWS_REGION"),
	})
	require.NoError(t, err, "Failed to create AWS manager")
	return manager
}

// sdkEC2 returns a raw EC2 SDK client built from the manager's own aws.Config,
// used for verification/cleanup steps that have no Manager equivalent.
func sdkEC2(manager *Manager) *ec2.Client {
	return ec2.NewFromConfig(manager.GetAWSConfig())
}

// TestAWSManagerIntegration tests real AWS operations
// Run with: go test -tags=integration -v ./pkg/aws
func TestAWSManagerIntegration(t *testing.T) {
	if os.Getenv("AWS_INTEGRATION_TESTS") != "true" {
		t.Skip("Skipping AWS integration tests. Set AWS_INTEGRATION_TESTS=true to run")
	}

	ctx := context.Background()
	manager := newRealAWSManager(t)
	ec2Client := sdkEC2(manager)

	// Test VPC discovery
	t.Run("VPC Discovery", func(t *testing.T) {
		vpcID, err := manager.DiscoverDefaultVPC()
		require.NoError(t, err)
		require.NotEmpty(t, vpcID, "Should find the default VPC")

		result, err := ec2Client.DescribeVpcs(ctx, &ec2.DescribeVpcsInput{VpcIds: []string{vpcID}})
		require.NoError(t, err)
		require.Len(t, result.Vpcs, 1)
		assert.True(t, aws.ToBool(result.Vpcs[0].IsDefault), "Discovered VPC should be the default VPC")
	})

	// Test Subnet discovery
	t.Run("Subnet Discovery", func(t *testing.T) {
		vpcID, err := manager.DiscoverDefaultVPC()
		require.NoError(t, err)

		subnetID, err := manager.DiscoverPublicSubnet(vpcID)
		require.NoError(t, err)
		require.NotEmpty(t, subnetID, "Should find a public subnet")

		result, err := ec2Client.DescribeSubnets(ctx, &ec2.DescribeSubnetsInput{SubnetIds: []string{subnetID}})
		require.NoError(t, err)
		require.Len(t, result.Subnets, 1)
		subnet := result.Subnets[0]
		assert.Equal(t, vpcID, aws.ToString(subnet.VpcId))
		assert.NotEmpty(t, aws.ToString(subnet.AvailabilityZone))
	})

	// Test Security Group operations. GetOrCreatePrismSecurityGroup returns the
	// shared "prism-access" group used by real workspaces, so it is not deleted.
	t.Run("Security Group Operations", func(t *testing.T) {
		vpcID, err := manager.DiscoverDefaultVPC()
		require.NoError(t, err)

		sgID, err := manager.GetOrCreatePrismSecurityGroup(vpcID)
		require.NoError(t, err)
		require.NotEmpty(t, sgID)

		result, err := ec2Client.DescribeSecurityGroups(ctx, &ec2.DescribeSecurityGroupsInput{
			GroupIds: []string{sgID},
		})
		require.NoError(t, err)
		require.Len(t, result.SecurityGroups, 1)
		assert.Equal(t, sgID, aws.ToString(result.SecurityGroups[0].GroupId))
		assert.Equal(t, vpcID, aws.ToString(result.SecurityGroups[0].VpcId))

		// Idempotent: a second call returns the same group.
		again, err := manager.GetOrCreatePrismSecurityGroup(vpcID)
		require.NoError(t, err)
		assert.Equal(t, sgID, again)
	})

	// Test AMI lookup
	t.Run("AMI Lookup", func(t *testing.T) {
		amiID, err := manager.GetAMIDiscovery().GetLatestAMI(ctx, "ubuntu", "22.04", manager.GetDefaultRegion(), "x86_64")
		require.NoError(t, err)
		require.NotEmpty(t, amiID, "Should find Ubuntu 22.04 AMI")

		result, err := ec2Client.DescribeImages(ctx, &ec2.DescribeImagesInput{ImageIds: []string{amiID}})
		require.NoError(t, err)
		require.Len(t, result.Images, 1)
		assert.Equal(t, types.ImageStateAvailable, result.Images[0].State)
	})

	// Test Instance Type validation (instance type offered in this region)
	t.Run("Instance Type Validation", func(t *testing.T) {
		am := manager.GetAvailabilityManager()

		azs, err := am.GetAvailableAZsForInstanceType(ctx, "t3.micro")
		assert.NoError(t, err)
		assert.NotEmpty(t, azs, "t3.micro should be offered")

		_, err = am.GetAvailableAZsForInstanceType(ctx, "invalid.type")
		assert.Error(t, err, "invalid.type should not be offered")

		// Note: This may fail in regions without GPU instances
		gpuAZs, gpuErr := am.GetAvailableAZsForInstanceType(ctx, "g4dn.xlarge")
		t.Logf("GPU instance g4dn.xlarge AZs: %v (err: %v)", gpuAZs, gpuErr)
	})

	// Test Availability Zone listing
	t.Run("Availability Zones", func(t *testing.T) {
		azs, err := manager.GetAvailabilityManager().GetAvailableAZsForInstanceType(ctx, "t3.micro")
		require.NoError(t, err)
		require.NotEmpty(t, azs, "Should have at least one AZ")

		result, err := ec2Client.DescribeAvailabilityZones(ctx, &ec2.DescribeAvailabilityZonesInput{ZoneNames: azs})
		require.NoError(t, err)
		for _, az := range result.AvailabilityZones {
			assert.NotEmpty(t, aws.ToString(az.ZoneName))
			assert.Equal(t, types.AvailabilityZoneStateAvailable, az.State, "AZ should be available")
		}
	})

	// Test Key Pair operations
	t.Run("Key Pair Operations", func(t *testing.T) {
		keyName := "prism-test-" + time.Now().Format("20060102-150405")

		pub, _, err := ed25519.GenerateKey(rand.Reader)
		require.NoError(t, err)
		sshPub, err := ssh.NewPublicKey(pub)
		require.NoError(t, err)

		err = manager.EnsureKeyPairExists(keyName, string(ssh.MarshalAuthorizedKey(sshPub)))
		require.NoError(t, err)

		defer func() {
			_ = manager.DeleteKeyPair(keyName)
		}()

		result, err := ec2Client.DescribeKeyPairs(ctx, &ec2.DescribeKeyPairsInput{KeyNames: []string{keyName}})
		require.NoError(t, err)
		require.Len(t, result.KeyPairs, 1)
		assert.Equal(t, keyName, aws.ToString(result.KeyPairs[0].KeyName))

		names, err := manager.ListPrismKeyPairs()
		require.NoError(t, err)
		assert.Contains(t, names, keyName)
	})

	// Test EBS Volume operations
	t.Run("EBS Volume Operations", func(t *testing.T) {
		volumeName := "prism-test-vol-" + time.Now().Format("20060102-150405")
		volume, err := manager.CreateStorage(cwstypes.StorageCreateRequest{
			Name:       volumeName,
			Size:       "10",
			VolumeType: "gp3",
		})
		require.NoError(t, err)
		require.NotEmpty(t, volume.VolumeID)

		defer func() {
			_ = manager.DeleteStorage(volumeName)
		}()

		volumes, err := manager.ListEBSVolumes()
		require.NoError(t, err)

		found := false
		for _, vol := range volumes {
			if vol.VolumeID == volume.VolumeID {
				found = true
				assert.Equal(t, volumeName, vol.Name)
				require.NotNil(t, vol.SizeGB)
				assert.Equal(t, int32(10), *vol.SizeGB)
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

// waitForInstanceState blocks until the instance reaches the given state using
// the AWS SDK waiters (Manager has no public wait helper).
func waitForInstanceState(ctx context.Context, t *testing.T, client *ec2.Client, instanceID, state string, timeout time.Duration) {
	t.Helper()
	input := &ec2.DescribeInstancesInput{InstanceIds: []string{instanceID}}
	var err error
	switch state {
	case "running":
		err = ec2.NewInstanceRunningWaiter(client).Wait(ctx, input, timeout)
	case "stopped":
		err = ec2.NewInstanceStoppedWaiter(client).Wait(ctx, input, timeout)
	default:
		t.Fatalf("unsupported wait state %q", state)
	}
	assert.NoError(t, err, "instance %s did not reach %s", instanceID, state)
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
	manager := newRealAWSManager(t)
	ec2Client := sdkEC2(manager)

	instanceName := "prism-test-" + time.Now().Format("20060102-150405")

	// Minimal Ubuntu base template; Size XS maps to t3.micro.
	launchRequest := cwstypes.LaunchRequest{
		Template: "ubuntu-22-04-x86",
		Name:     instanceName,
		Size:     "XS",
		Spot:     false,
	}

	instance, err := manager.LaunchInstance(launchRequest)
	require.NoError(t, err)
	require.NotNil(t, instance)
	require.NotEmpty(t, instance.ID)
	instanceID := instance.ID

	defer func() {
		_ = manager.DeleteInstance(instanceName)
	}()

	waitForInstanceState(ctx, t, ec2Client, instanceID, "running", 5*time.Minute)

	described, err := manager.GetInstance(instanceID)
	require.NoError(t, err)
	assert.Equal(t, instanceID, described.ID)
	assert.Equal(t, instanceName, described.Name)
	assert.Equal(t, "running", described.State)
	assert.NotEmpty(t, described.PublicIP)

	// Test stop/start cycle
	t.Run("Stop/Start Cycle", func(t *testing.T) {
		assert.NoError(t, manager.StopInstance(instanceName))
		waitForInstanceState(ctx, t, ec2Client, instanceID, "stopped", 3*time.Minute)

		assert.NoError(t, manager.StartInstance(instanceName))
		waitForInstanceState(ctx, t, ec2Client, instanceID, "running", 3*time.Minute)
	})

	// Test hibernation (if supported)
	t.Run("Hibernation Test", func(t *testing.T) {
		supported, state, _, err := manager.GetInstanceHibernationStatus(instanceName)
		assert.NoError(t, err)
		t.Logf("Hibernation supported=%v state=%s", supported, state)

		if !supported {
			t.Log("Hibernation not supported for this instance")
			return
		}
		if err := manager.HibernateInstance(instanceName); err != nil {
			t.Logf("HibernateInstance failed: %v", err)
			return
		}
		waitForInstanceState(ctx, t, ec2Client, instanceID, "stopped", 3*time.Minute)

		assert.NoError(t, manager.ResumeInstance(instanceName))
		waitForInstanceState(ctx, t, ec2Client, instanceID, "running", 3*time.Minute)
	})

	// Test tagging
	t.Run("Instance Tagging", func(t *testing.T) {
		tags := map[string]string{
			"TestTag":     "TestValue",
			"Integration": "true",
		}

		for k, v := range tags {
			assert.NoError(t, manager.TagInstance(instanceID, k, v))
		}

		result, err := ec2Client.DescribeInstances(ctx, &ec2.DescribeInstancesInput{InstanceIds: []string{instanceID}})
		require.NoError(t, err)
		require.NotEmpty(t, result.Reservations)
		require.NotEmpty(t, result.Reservations[0].Instances)

		got := map[string]string{}
		for _, tag := range result.Reservations[0].Instances[0].Tags {
			got[aws.ToString(tag.Key)] = aws.ToString(tag.Value)
		}
		for k, v := range tags {
			assert.Equal(t, v, got[k], "tag %s", k)
		}
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
	manager := newRealAWSManager(t)

	volumeName := "prism-test-efs-" + time.Now().Format("20060102-150405")

	// CreateVolume waits for the file system to become available.
	volume, err := manager.CreateVolume(cwstypes.VolumeCreateRequest{Name: volumeName})
	require.NoError(t, err)
	require.NotEmpty(t, volume.FileSystemID)
	fsID := volume.FileSystemID

	// Clean up directly via the SDK: Manager.DeleteVolume resolves the file
	// system from daemon state, which this test does not write to.
	defer func() {
		_, _ = efs.NewFromConfig(manager.GetAWSConfig()).DeleteFileSystem(ctx, &efs.DeleteFileSystemInput{
			FileSystemId: aws.String(fsID),
		})
	}()

	volumes, err := manager.ListEFSVolumes()
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
	manager := newRealAWSManager(t)
	region := manager.GetDefaultRegion()

	onDemandPrice := func(instanceType string) (float64, error) {
		return manager.onDemandPricingClient.GetOnDemandHourlyRate(ctx, instanceType, region)
	}

	// Test on-demand pricing
	t.Run("On-Demand Pricing", func(t *testing.T) {
		price, err := onDemandPrice("t3.micro")
		assert.NoError(t, err)
		assert.Greater(t, price, 0.0, "Should return positive price")
		assert.Less(t, price, 0.1, "t3.micro should be less than $0.10/hour")
	})

	// Test spot pricing
	t.Run("Spot Pricing", func(t *testing.T) {
		spotPrice, err := manager.GetSpotPrice(ctx, "t3.micro")
		assert.NoError(t, err)
		assert.Greater(t, spotPrice, 0.0, "Should return positive spot price")

		onDemand, err := onDemandPrice("t3.micro")
		require.NoError(t, err)
		assert.Less(t, spotPrice, onDemand, "Spot price should be less than on-demand")
	})

	// Test GPU instance pricing
	t.Run("GPU Instance Pricing", func(t *testing.T) {
		price, err := onDemandPrice("g4dn.xlarge")
		if err == nil {
			assert.Greater(t, price, 0.5, "GPU instance should be more expensive")
			assert.Less(t, price, 5.0, "GPU instance should be reasonable")
		} else {
			t.Logf("GPU instances not available in this region: %v", err)
		}
	})
}
