package aws

import (
	"context"
	"encoding/base64"
	"fmt"
	"runtime"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/aws/aws-sdk-go-v2/service/efs"
	efsTypes "github.com/aws/aws-sdk-go-v2/service/efs/types"
	"github.com/aws/aws-sdk-go-v2/service/sts"

	ctypes "github.com/scttfrdmn/cloudworkstation/pkg/types"
)

// Manager handles all AWS operations
type Manager struct {
	cfg       aws.Config
	ec2       *ec2.Client
	efs       *efs.Client
	sts       *sts.Client
	region    string
	templates map[string]ctypes.Template
}

// NewManager creates a new AWS manager
func NewManager() (*Manager, error) {
	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	manager := &Manager{
		cfg:       cfg,
		ec2:       ec2.NewFromConfig(cfg),
		efs:       efs.NewFromConfig(cfg),
		sts:       sts.NewFromConfig(cfg),
		region:    cfg.Region,
		templates: getTemplates(),
	}

	return manager, nil
}

// GetDefaultRegion returns the default AWS region
func (m *Manager) GetDefaultRegion() string {
	return m.region
}

// GetTemplates returns all available templates
func (m *Manager) GetTemplates() map[string]ctypes.Template {
	return m.templates
}

// GetTemplate returns a specific template
func (m *Manager) GetTemplate(name string) (*ctypes.Template, error) {
	template, exists := m.templates[name]
	if !exists {
		return nil, fmt.Errorf("template %s not found", name)
	}
	return &template, nil
}

// LaunchInstance launches a new EC2 instance
func (m *Manager) LaunchInstance(req ctypes.LaunchRequest) (*ctypes.Instance, error) {
	// Get template
	template, exists := m.templates[req.Template]
	if !exists {
		return nil, fmt.Errorf("template '%s' not found", req.Template)
	}

	// Detect architecture (use local for now, could be part of request)
	arch := m.getLocalArchitecture()
	
	// Get template configuration for architecture and region
	ami, instanceType, costPerHour, err := m.getTemplateForArchitecture(template, arch, m.region)
	if err != nil {
		return nil, fmt.Errorf("failed to get template configuration: %w", err)
	}

	// Get daily cost estimate
	dailyCost := costPerHour * 24

	// Prepare UserData
	userData := template.UserData
	
	// Add EFS mount if volumes specified
	if len(req.Volumes) > 0 {
		for _, volumeName := range req.Volumes {
			// Get volume details from state manager would be needed here
			// For now, we'll include the volume name in user data
			userData = m.addEFSMountToUserData(userData, volumeName, m.region)
		}
	}

	// Encode UserData
	userDataEncoded := base64.StdEncoding.EncodeToString([]byte(userData))

	// Launch instance
	input := &ec2.RunInstancesInput{
		ImageId:      aws.String(ami),
		InstanceType: types.InstanceType(instanceType),
		MinCount:     aws.Int32(1),
		MaxCount:     aws.Int32(1),
		UserData:     aws.String(userDataEncoded),
		SecurityGroups: []string{
			"default", // For now, use default security group
		},
		TagSpecifications: []types.TagSpecification{
			{
				ResourceType: types.ResourceTypeInstance,
				Tags: []types.Tag{
					{
						Key:   aws.String("Name"),
						Value: aws.String(req.Name),
					},
					{
						Key:   aws.String("CloudWorkstation"),
						Value: aws.String("true"),
					},
					{
						Key:   aws.String("Template"),
						Value: aws.String(req.Template),
					},
				},
			},
		},
	}

	if req.DryRun {
		input.DryRun = aws.Bool(true)
	}

	result, err := m.ec2.RunInstances(context.TODO(), input)
	if err != nil {
		return nil, fmt.Errorf("failed to launch instance: %w", err)
	}

	if req.DryRun {
		return &ctypes.Instance{
			Name:     req.Name,
			Template: req.Template,
			State:    "dry-run",
		}, nil
	}

	// Get the launched instance
	instance := result.Instances[0]
	
	return &ctypes.Instance{
		ID:                 *instance.InstanceId,
		Name:               req.Name,
		Template:           req.Template,
		State:              string(instance.State.Name),
		LaunchTime:         time.Now(),
		EstimatedDailyCost: dailyCost,
		AttachedVolumes:    req.Volumes,
	}, nil
}

// DeleteInstance terminates an EC2 instance
func (m *Manager) DeleteInstance(name string) error {
	// Find instance by name tag
	instanceID, err := m.findInstanceByName(name)
	if err != nil {
		return fmt.Errorf("failed to find instance: %w", err)
	}

	// Terminate the instance
	_, err = m.ec2.TerminateInstances(context.TODO(), &ec2.TerminateInstancesInput{
		InstanceIds: []string{instanceID},
	})
	if err != nil {
		return fmt.Errorf("failed to terminate instance: %w", err)
	}

	return nil
}

// StartInstance starts a stopped EC2 instance
func (m *Manager) StartInstance(name string) error {
	// Find instance by name tag
	instanceID, err := m.findInstanceByName(name)
	if err != nil {
		return fmt.Errorf("failed to find instance: %w", err)
	}

	// Start the instance
	_, err = m.ec2.StartInstances(context.TODO(), &ec2.StartInstancesInput{
		InstanceIds: []string{instanceID},
	})
	if err != nil {
		return fmt.Errorf("failed to start instance: %w", err)
	}

	return nil
}

// StopInstance stops a running EC2 instance
func (m *Manager) StopInstance(name string) error {
	// Find instance by name tag
	instanceID, err := m.findInstanceByName(name)
	if err != nil {
		return fmt.Errorf("failed to find instance: %w", err)
	}

	// Stop the instance
	_, err = m.ec2.StopInstances(context.TODO(), &ec2.StopInstancesInput{
		InstanceIds: []string{instanceID},
	})
	if err != nil {
		return fmt.Errorf("failed to stop instance: %w", err)
	}

	return nil
}

// GetConnectionInfo returns connection information for an instance
func (m *Manager) GetConnectionInfo(name string) (string, error) {
	// Find instance by name tag
	instanceID, err := m.findInstanceByName(name)
	if err != nil {
		return "", fmt.Errorf("failed to find instance: %w", err)
	}

	// Get instance details
	result, err := m.ec2.DescribeInstances(context.TODO(), &ec2.DescribeInstancesInput{
		InstanceIds: []string{instanceID},
	})
	if err != nil {
		return "", fmt.Errorf("failed to describe instance: %w", err)
	}

	if len(result.Reservations) == 0 || len(result.Reservations[0].Instances) == 0 {
		return "", fmt.Errorf("instance not found")
	}

	instance := result.Reservations[0].Instances[0]
	
	if instance.PublicIpAddress == nil {
		return "", fmt.Errorf("instance has no public IP address")
	}

	return fmt.Sprintf("ssh ubuntu@%s", *instance.PublicIpAddress), nil
}

// CreateVolume creates a new EFS volume
func (m *Manager) CreateVolume(req ctypes.VolumeCreateRequest) (*ctypes.EFSVolume, error) {
	// Set defaults
	performanceMode := "generalPurpose"
	if req.PerformanceMode != "" {
		performanceMode = req.PerformanceMode
	}
	
	throughputMode := "bursting"
	if req.ThroughputMode != "" {
		throughputMode = req.ThroughputMode
	}

	// Create EFS file system
	input := &efs.CreateFileSystemInput{
		PerformanceMode: efsTypes.PerformanceMode(performanceMode),
		ThroughputMode:  efsTypes.ThroughputMode(throughputMode),
		Tags: []efsTypes.Tag{
			{
				Key:   aws.String("Name"),
				Value: aws.String(req.Name),
			},
			{
				Key:   aws.String("CloudWorkstation"),
				Value: aws.String("true"),
			},
		},
	}

	result, err := m.efs.CreateFileSystem(context.TODO(), input)
	if err != nil {
		return nil, fmt.Errorf("failed to create EFS file system: %w", err)
	}

	volume := &ctypes.EFSVolume{
		Name:            req.Name,
		FileSystemId:    *result.FileSystemId,
		Region:          m.region,
		CreationTime:    time.Now(),
		State:           string(result.LifeCycleState),
		PerformanceMode: performanceMode,
		ThroughputMode:  throughputMode,
		EstimatedCostGB: 0.30, // Standard pricing
		SizeBytes:       0,     // Will be updated as files are added
	}

	return volume, nil
}

// DeleteVolume deletes an EFS volume
func (m *Manager) DeleteVolume(name string) error {
	// TODO: Implement EFS volume deletion logic
	return fmt.Errorf("not implemented yet")
}

// CreateStorage creates a new EBS volume
func (m *Manager) CreateStorage(req ctypes.StorageCreateRequest) (*ctypes.EBSVolume, error) {
	// TODO: Implement EBS volume creation logic
	return nil, fmt.Errorf("not implemented yet")
}

// DeleteStorage deletes an EBS volume
func (m *Manager) DeleteStorage(name string) error {
	// TODO: Implement EBS volume deletion logic
	return fmt.Errorf("not implemented yet")
}

// AttachStorage attaches an EBS volume to an instance
func (m *Manager) AttachStorage(volumeName, instanceName string) error {
	// TODO: Implement EBS volume attachment logic
	return fmt.Errorf("not implemented yet")
}

// DetachStorage detaches an EBS volume from an instance
func (m *Manager) DetachStorage(volumeName string) error {
	// TODO: Implement EBS volume detachment logic
	return fmt.Errorf("not implemented yet")
}

// Helper functions

// getLocalArchitecture detects the local system architecture
func (m *Manager) getLocalArchitecture() string {
	arch := runtime.GOARCH
	switch arch {
	case "amd64":
		return "x86_64"
	case "arm64":
		return "arm64"
	default:
		return "x86_64" // Default fallback
	}
}

// getTemplateForArchitecture gets AMI, instance type and cost for a specific architecture
func (m *Manager) getTemplateForArchitecture(template ctypes.Template, arch, region string) (string, string, float64, error) {
	// Check if template supports the region
	regionAmis, regionExists := template.AMI[region]
	if !regionExists {
		return "", "", 0, fmt.Errorf("template '%s' does not support region '%s'", template.Name, region)
	}
	
	// Check if template supports the architecture in this region
	ami, archExists := regionAmis[arch]
	if !archExists {
		return "", "", 0, fmt.Errorf("template '%s' does not support architecture '%s' in region '%s'", template.Name, arch, region)
	}
	
	// Get instance type for architecture
	instanceType, typeExists := template.InstanceType[arch]
	if !typeExists {
		return "", "", 0, fmt.Errorf("template '%s' does not have instance type for architecture '%s'", template.Name, arch)
	}
	
	// Get cost for architecture
	costPerHour, costExists := template.EstimatedCostPerHour[arch]
	if !costExists {
		return "", "", 0, fmt.Errorf("template '%s' does not have cost information for architecture '%s'", template.Name, arch)
	}
	
	return ami, instanceType, costPerHour, nil
}

// addEFSMountToUserData adds EFS mount commands to UserData script
func (m *Manager) addEFSMountToUserData(originalUserData, volumeName, region string) string {
	// This is a simplified version - in practice, we'd need to get the filesystem ID
	// from the state manager or EFS service
	efsMount := fmt.Sprintf(`

# Mount EFS volume: %s
mkdir -p /mnt/%s
apt-get update && apt-get install -y nfs-common
mount -t nfs4 -o nfsvers=4.1,rsize=1048576,wsize=1048576,hard,intr,timeo=600 %s.efs.%s.amazonaws.com:/ /mnt/%s
echo "%s.efs.%s.amazonaws.com:/ /mnt/%s nfs4 nfsvers=4.1,rsize=1048576,wsize=1048576,hard,intr,timeo=600,_netdev 0 0" >> /etc/fstab
chown -R ubuntu:ubuntu /mnt/%s
`, volumeName, volumeName, volumeName, region, volumeName, volumeName, region, volumeName, volumeName)
	
	return originalUserData + efsMount
}

// findInstanceByName finds an EC2 instance by its Name tag
func (m *Manager) findInstanceByName(name string) (string, error) {
	result, err := m.ec2.DescribeInstances(context.TODO(), &ec2.DescribeInstancesInput{
		Filters: []types.Filter{
			{
				Name:   aws.String("tag:Name"),
				Values: []string{name},
			},
			{
				Name:   aws.String("tag:CloudWorkstation"),
				Values: []string{"true"},
			},
			{
				Name:   aws.String("instance-state-name"),
				Values: []string{"pending", "running", "shutting-down", "stopping", "stopped"},
			},
		},
	})
	if err != nil {
		return "", fmt.Errorf("failed to describe instances: %w", err)
	}

	for _, reservation := range result.Reservations {
		for _, instance := range reservation.Instances {
			return *instance.InstanceId, nil
		}
	}

	return "", fmt.Errorf("instance '%s' not found", name)
}