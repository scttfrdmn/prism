package aws

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	ec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	spawn "github.com/spore-host/spawn/pkg/aws"

	"github.com/scttfrdmn/prism/pkg/launch"
)

// ec2Launcher is Prism's own EC2-backed implementation of the launch.Launcher
// port. It reproduces the exact RunInstances behavior Prism has always had —
// hand-built network interface with a public IP, gp3 root volume, spot/
// hibernation options, AZ failover — but sourced from a spawn.LaunchConfig
// instead of a types.LaunchRequest.
//
// This is the Phase-1 default: introducing the seam without changing behavior.
// In Phase 2 the Manager's launcher field is swapped to *spawn.Client (which
// also satisfies launch.Launcher), at which point spawn builds the RunInstances
// input itself and this type is retired.
type ec2Launcher struct {
	m *Manager
}

// newEC2Launcher builds the default launcher backed by the manager's EC2 client.
func newEC2Launcher(m *Manager) *ec2Launcher {
	return &ec2Launcher{m: m}
}

// compile-time proof the default impl satisfies the port.
var _ launch.Launcher = (*ec2Launcher)(nil)

// Launch reconstructs the RunInstances input from the config and runs Prism's
// existing launch pipeline (retry + AZ failover), returning a spawn.LaunchResult.
func (l *ec2Launcher) Launch(ctx context.Context, cfg spawn.LaunchConfig) (*spawn.LaunchResult, error) {
	runInput := configToRunInput(cfg)

	instance, err := l.m.runInstancesWithFailover(ctx, runInput, cfg.InstanceType)
	if err != nil {
		return nil, err
	}
	return ec2InstanceToLaunchResult(instance, cfg.Name), nil
}

// StopInstance stops (or hibernates) an instance in the given region.
func (l *ec2Launcher) StopInstance(ctx context.Context, region, instanceID string, hibernate bool) error {
	client := l.m.getRegionalEC2Client(region)
	input := &ec2.StopInstancesInput{InstanceIds: []string{instanceID}}
	if hibernate {
		input.Hibernate = aws.Bool(true)
	}
	return WithRetry(ctx, DefaultRetryConfig(), func(ctx context.Context) error {
		_, err := client.StopInstances(ctx, input)
		return err
	})
}

// StartInstance starts a stopped or hibernated instance.
func (l *ec2Launcher) StartInstance(ctx context.Context, region, instanceID string) error {
	client := l.m.getRegionalEC2Client(region)
	return WithRetry(ctx, DefaultRetryConfig(), func(ctx context.Context) error {
		_, err := client.StartInstances(ctx, &ec2.StartInstancesInput{
			InstanceIds: []string{instanceID},
		})
		return err
	})
}

// Terminate permanently deletes an instance. The raw error is returned
// unwrapped so callers can detect the "already gone" case by code/string.
func (l *ec2Launcher) Terminate(ctx context.Context, region, instanceID string) error {
	client := l.m.getRegionalEC2Client(region)
	return WithRetry(ctx, DefaultRetryConfig(), func(ctx context.Context) error {
		_, err := client.TerminateInstances(ctx, &ec2.TerminateInstancesInput{
			InstanceIds: []string{instanceID},
		})
		return err
	})
}

// configToRunInput reconstructs the ec2.RunInstancesInput Prism has always built,
// sourcing every field from the spawn.LaunchConfig. It mirrors the former
// InstanceConfigBuilder.BuildRunInstancesInput + LaunchOptionsProcessor.ProcessOptions
// exactly (public-IP network interface, gp3 root volume, hibernation/spot options).
func configToRunInput(cfg spawn.LaunchConfig) *ec2.RunInstancesInput {
	minCount := int32(1)
	maxCount := int32(1)

	runInput := &ec2.RunInstancesInput{
		ImageId:      aws.String(cfg.AMI),
		InstanceType: ec2types.InstanceType(cfg.InstanceType),
		MinCount:     &minCount,
		MaxCount:     &maxCount,
		UserData:     aws.String(cfg.UserData),
		// NetworkInterfaces (not top-level subnet/SG) so the instance always gets
		// a public IP even in subnets with MapPublicIpOnLaunch=false (Issue #439).
		NetworkInterfaces: []ec2types.InstanceNetworkInterfaceSpecification{
			{
				DeviceIndex:              aws.Int32(0),
				SubnetId:                 aws.String(cfg.SubnetID),
				Groups:                   cfg.SecurityGroupIDs,
				AssociatePublicIpAddress: aws.Bool(true),
			},
		},
		TagSpecifications: []ec2types.TagSpecification{
			{
				ResourceType: ec2types.ResourceTypeInstance,
				Tags:         tagsMapToEC2(cfg.Tags),
			},
		},
	}

	if cfg.KeyName != "" {
		runInput.KeyName = aws.String(cfg.KeyName)
	}

	// Pin to a pre-reserved EC2 Capacity Block / reservation (#63).
	if cfg.ReservationID != "" {
		runInput.CapacityReservationSpecification = &ec2types.CapacityReservationSpecification{
			CapacityReservationTarget: &ec2types.CapacityReservationTarget{
				CapacityReservationId: aws.String(cfg.ReservationID),
			},
		}
	}

	// IAM instance profile is optional — only set when it was resolved upstream.
	if cfg.IamInstanceProfile != "" {
		runInput.IamInstanceProfile = &ec2types.IamInstanceProfileSpecification{
			Name: aws.String(cfg.IamInstanceProfile),
		}
		log.Printf("Using IAM instance profile for SSM access")
	} else {
		log.Printf("IAM instance profile not found - launching without it (SSM features will be unavailable)")
	}

	// Root device name: AWS uses different names for different AMIs.
	rootDevice := "/dev/sda1"
	if strings.Contains(strings.ToLower(cfg.AMI), "amazon") || strings.Contains(strings.ToLower(cfg.AMI), "amzn") {
		rootDevice = "/dev/xvda"
	}
	runInput.BlockDeviceMappings = []ec2types.BlockDeviceMapping{
		{
			DeviceName: aws.String(rootDevice),
			Ebs: &ec2types.EbsBlockDevice{
				VolumeType:          ec2types.VolumeTypeGp3,
				VolumeSize:          aws.Int32(cfg.RootVolumeSizeGiB),
				Encrypted:           aws.Bool(cfg.Hibernate), // Only encrypt if hibernation enabled
				DeleteOnTermination: aws.Bool(true),
			},
		},
	}

	// Hibernation support.
	if cfg.Hibernate {
		runInput.HibernationOptions = &ec2types.HibernationOptionsRequest{
			Configured: aws.Bool(true),
		}
	}

	// Spot instance support.
	if cfg.Spot {
		spotOpts := &ec2types.SpotMarketOptions{
			SpotInstanceType: ec2types.SpotInstanceTypeOneTime,
		}
		if cfg.SpotMaxPrice != "" {
			spotOpts.MaxPrice = aws.String(cfg.SpotMaxPrice)
		}
		runInput.InstanceMarketOptions = &ec2types.InstanceMarketOptionsRequest{
			MarketType:  ec2types.MarketTypeSpot,
			SpotOptions: spotOpts,
		}
	}

	return runInput
}

// tagsMapToEC2 converts a tag map into the EC2 tag slice RunInstances expects.
func tagsMapToEC2(tags map[string]string) []ec2types.Tag {
	out := make([]ec2types.Tag, 0, len(tags))
	for k, v := range tags {
		out = append(out, ec2types.Tag{Key: aws.String(k), Value: aws.String(v)})
	}
	return out
}

// ec2InstanceToLaunchResult adapts a launched EC2 instance into a spawn.LaunchResult.
func ec2InstanceToLaunchResult(instance *ec2types.Instance, name string) *spawn.LaunchResult {
	res := &spawn.LaunchResult{Name: name}
	if instance == nil {
		return res
	}
	if instance.InstanceId != nil {
		res.InstanceID = *instance.InstanceId
	}
	if instance.PublicIpAddress != nil {
		res.PublicIP = *instance.PublicIpAddress
	}
	if instance.PrivateIpAddress != nil {
		res.PrivateIP = *instance.PrivateIpAddress
	}
	if instance.Placement != nil && instance.Placement.AvailabilityZone != nil {
		res.AvailabilityZone = *instance.Placement.AvailabilityZone
	}
	if instance.State != nil {
		res.State = string(instance.State.Name)
	}
	if instance.KeyName != nil {
		res.KeyName = *instance.KeyName
	}
	return res
}

// runInstancesWithFailover performs the actual EC2 launch with intelligent AZ
// failover. It is the former InstanceLauncher.executeInstanceLaunch, relocated
// onto Manager so the launcher implementation can drive it.
func (m *Manager) runInstancesWithFailover(ctx context.Context, runInput *ec2.RunInstancesInput, instanceType string) (*ec2types.Instance, error) {
	// When a subnet is specified, AWS uses the subnet's AZ — don't override
	// placement (Issue #427, #439). Prism's pipeline always resolves a subnet,
	// so this is the path taken in practice.
	if hasSubnetConfig(runInput) {
		var result *ec2.RunInstancesOutput
		err := WithRetry(ctx, DefaultRetryConfig(), func(ctx context.Context) error {
			var runErr error
			result, runErr = m.ec2.RunInstances(ctx, runInput)
			return runErr
		})
		if err != nil {
			return nil, EnhanceError(err, "Launch instance")
		}
		if len(result.Instances) == 0 {
			return nil, fmt.Errorf("no instances returned from launch")
		}

		instance := &result.Instances[0]
		selectedAZ := ""
		if instance.Placement != nil && instance.Placement.AvailabilityZone != nil {
			selectedAZ = *instance.Placement.AvailabilityZone
		}
		log.Printf("✅ Successfully launched instance %s in AZ %s (from subnet)", *instance.InstanceId, selectedAZ)
		return instance, nil
	}

	// No subnet specified — use AvailabilityManager for AZ selection + failover.
	instanceID, selectedAZ, err := m.availabilityManager.AttemptLaunchWithFailover(
		ctx,
		instanceType,
		func(ctx context.Context, az string) (string, error) {
			if runInput.Placement == nil {
				runInput.Placement = &ec2types.Placement{}
			}
			runInput.Placement.AvailabilityZone = aws.String(az)

			var result *ec2.RunInstancesOutput
			err := WithRetry(ctx, DefaultRetryConfig(), func(ctx context.Context) error {
				var runErr error
				result, runErr = m.ec2.RunInstances(ctx, runInput)
				return runErr
			})
			if err != nil {
				return "", err
			}
			if len(result.Instances) == 0 {
				return "", fmt.Errorf("no instances returned from launch")
			}
			return *result.Instances[0].InstanceId, nil
		},
	)
	if err != nil {
		return nil, EnhanceError(err, "Launch instance")
	}

	describeOutput, err := m.ec2.DescribeInstances(ctx, &ec2.DescribeInstancesInput{
		InstanceIds: []string{instanceID},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve launched instance details: %w", err)
	}
	if len(describeOutput.Reservations) == 0 || len(describeOutput.Reservations[0].Instances) == 0 {
		return nil, fmt.Errorf("launched instance not found: %s", instanceID)
	}

	log.Printf("✅ Successfully launched instance %s in AZ %s", instanceID, selectedAZ)
	return &describeOutput.Reservations[0].Instances[0], nil
}
