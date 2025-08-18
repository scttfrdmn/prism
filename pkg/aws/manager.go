package aws

import (
	"context"
	"encoding/base64"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	ec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/aws/aws-sdk-go-v2/service/efs"
	efsTypes "github.com/aws/aws-sdk-go-v2/service/efs/types"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
	"github.com/aws/aws-sdk-go-v2/service/sts"

	"github.com/scttfrdmn/cloudworkstation/pkg/state"
	"github.com/scttfrdmn/cloudworkstation/pkg/templates"
	ctypes "github.com/scttfrdmn/cloudworkstation/pkg/types"
)

// Manager handles all AWS operations
type Manager struct {
	cfg             aws.Config
	ec2             EC2ClientInterface
	efs             EFSClientInterface
	ssm             SSMClientInterface
	sts             STSClientInterface
	region          string
	templates       map[string]ctypes.Template
	pricingCache    map[string]float64
	lastPriceUpdate time.Time
	discountConfig  ctypes.DiscountConfig
	stateManager    StateManagerInterface
}

// ManagerOptions contains optional parameters for creating a new Manager
type ManagerOptions struct {
	Profile string // AWS profile name
	Region  string // AWS region
}

// NewManager creates a new AWS manager
func NewManager(opts ...ManagerOptions) (*Manager, error) {
	var opt ManagerOptions
	if len(opts) > 0 {
		opt = opts[0]
	}

	// Load AWS configuration with optional profile and region
	cfgOpts := []func(*config.LoadOptions) error{}

	// Set profile if specified
	if opt.Profile != "" {
		cfgOpts = append(cfgOpts, config.WithSharedConfigProfile(opt.Profile))
	}

	// Set region if specified
	if opt.Region != "" {
		cfgOpts = append(cfgOpts, config.WithRegion(opt.Region))
	}

	ctx := context.Background()
	cfg, err := config.LoadDefaultConfig(ctx, cfgOpts...)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	// Initialize state manager
	stateManager, err := state.NewManager()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize state manager: %w", err)
	}

	// Create clients
	ec2Client := ec2.NewFromConfig(cfg)
	efsClient := efs.NewFromConfig(cfg)
	ssmClient := ssm.NewFromConfig(cfg)
	stsClient := sts.NewFromConfig(cfg)

	// Use specified region or fallback to config region
	region := opt.Region
	if region == "" {
		region = cfg.Region
	}

	manager := &Manager{
		cfg:             cfg,
		ec2:             ec2Client,
		efs:             efsClient,
		ssm:             ssmClient,
		sts:             stsClient,
		region:          region,
		templates:       getTemplates(),
		pricingCache:    make(map[string]float64),
		lastPriceUpdate: time.Time{},
		discountConfig:  ctypes.DiscountConfig{}, // No discounts by default
		stateManager:    stateManager,
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
	// Detect architecture (use local for now, could be part of request)
	arch := m.getLocalArchitecture()

	// Always use unified template system with inheritance support
	return m.launchWithUnifiedTemplateSystem(req, arch)
}

// TemplateConfigExtractor extracts configuration from unified template (Single Responsibility - SOLID)
type TemplateConfigExtractor struct {
	region string
}

// ExtractConfig extracts AMI, instance type, and cost from template
func (e *TemplateConfigExtractor) ExtractConfig(template *ctypes.RuntimeTemplate, arch string) (string, string, float64, error) {
	ami, exists := template.AMI[e.region][arch]
	if !exists {
		return "", "", 0, fmt.Errorf("AMI not available for region %s and architecture %s", e.region, arch)
	}

	instanceType, exists := template.InstanceType[arch]
	if !exists {
		return "", "", 0, fmt.Errorf("instance type not available for architecture %s", arch)
	}

	dailyCost := template.EstimatedCostPerHour[arch] * 24
	return ami, instanceType, dailyCost, nil
}

// UserDataProcessor processes and configures user data (Single Responsibility - SOLID)
type UserDataProcessor struct {
	manager *Manager
	region  string
}

// ProcessUserData configures and encodes user data for instance launch
func (p *UserDataProcessor) ProcessUserData(template *ctypes.RuntimeTemplate, req ctypes.LaunchRequest) string {
	userData := template.UserData
	userData = p.manager.processIdleDetectionConfig(userData, template)
	
	// Add EFS mount if volumes specified
	if len(req.Volumes) > 0 {
		for _, volumeName := range req.Volumes {
			userData = p.manager.addEFSMountToUserData(userData, volumeName, p.region)
		}
	}

	return base64.StdEncoding.EncodeToString([]byte(userData))
}

// NetworkingResolver resolves VPC, subnet, and security group (Single Responsibility - SOLID)
type NetworkingResolver struct {
	manager *Manager
}

// ResolveNetworking determines VPC, subnet, and security group for launch
func (n *NetworkingResolver) ResolveNetworking(req ctypes.LaunchRequest) (string, string, string, error) {
	var vpcID, subnetID string

	if req.VpcID != "" {
		vpcID = req.VpcID
	} else {
		discoveredVPC, err := n.manager.DiscoverDefaultVPC()
		if err != nil {
			return "", "", "", fmt.Errorf("failed to discover VPC: %w\n\n🏗️  To fix this issue:\n  1. Create a default VPC: aws ec2 create-default-vpc\n  2. Or specify a VPC: cws launch %s %s --vpc vpc-xxxxxxxxx", err, req.Template, req.Name)
		}
		vpcID = discoveredVPC
	}

	if req.SubnetID != "" {
		subnetID = req.SubnetID
	} else {
		discoveredSubnet, err := n.manager.DiscoverPublicSubnet(vpcID)
		if err != nil {
			return "", "", "", fmt.Errorf("failed to discover subnet: %w\n\n🏗️  To fix this issue:\n  1. Create a public subnet in your VPC\n  2. Or specify a subnet: cws launch %s %s --subnet subnet-xxxxxxxxx", err, req.Template, req.Name)
		}
		subnetID = discoveredSubnet
	}

	securityGroupID, err := n.manager.GetOrCreateCloudWorkstationSecurityGroup(vpcID)
	if err != nil {
		return "", "", "", fmt.Errorf("failed to create security group: %w", err)
	}

	return vpcID, subnetID, securityGroupID, nil
}

// InstanceConfigBuilder builds EC2 RunInstances configuration (Builder Pattern - SOLID)
type InstanceConfigBuilder struct {
	manager *Manager
}

// BuildRunInstancesInput creates configured RunInstancesInput
func (b *InstanceConfigBuilder) BuildRunInstancesInput(req ctypes.LaunchRequest, ami, instanceType, userDataEncoded, subnetID, securityGroupID string) (*ec2.RunInstancesInput, error) {
	minCount := int32(1)
	maxCount := int32(1)
	
	runInput := &ec2.RunInstancesInput{
		ImageId:          &ami,
		InstanceType:     ec2types.InstanceType(instanceType),
		MinCount:         &minCount,
		MaxCount:         &maxCount,
		UserData:         &userDataEncoded,
		SubnetId:         aws.String(subnetID),
		SecurityGroupIds: []string{securityGroupID},
		IamInstanceProfile: &ec2types.IamInstanceProfileSpecification{
			Name: aws.String("CloudWorkstation-Instance-Profile"),
		},
		TagSpecifications: []ec2types.TagSpecification{
			{
				ResourceType: ec2types.ResourceTypeInstance,
				Tags: []ec2types.Tag{
					{Key: aws.String("Name"), Value: &req.Name},
					{Key: aws.String("CloudWorkstation"), Value: aws.String("true")},
					{Key: aws.String("LaunchedBy"), Value: aws.String("CloudWorkstation")},
					{Key: aws.String("Template"), Value: &req.Template},
					{Key: aws.String("PackageManager"), Value: &req.PackageManager},
				},
			},
		},
	}

	// Add SSH key pair if provided
	if req.SSHKeyName != "" {
		runInput.KeyName = aws.String(req.SSHKeyName)
	}

	return runInput, nil
}

// LaunchOptionsProcessor processes hibernation and spot options (Strategy Pattern - SOLID)
type LaunchOptionsProcessor struct {
	manager *Manager
}

// ProcessOptions validates and applies hibernation/spot options
func (p *LaunchOptionsProcessor) ProcessOptions(req ctypes.LaunchRequest, runInput *ec2.RunInstancesInput, ami, instanceType string) error {
	// Validate hibernation and spot combination
	if req.Hibernation && req.Spot {
		return fmt.Errorf("hibernation and spot instances cannot be used together\n\n💡 AWS Limitation:\n  • Spot instances can be interrupted at any time\n  • Hibernation preserves instance state for later resume\n  • These features are incompatible\n\nChoose one:\n  • Use --hibernation for cost-effective session preservation\n  • Use --spot for discounted compute pricing\n  • Use both flags separately on different instances")
	}

	// Add hibernation support if requested
	if req.Hibernation {
		if !p.manager.supportsHibernation(instanceType) {
			return fmt.Errorf("instance type %s does not support hibernation\n\n💡 Hibernation is supported on:\n  • General Purpose: T2, T3, T3a, M3-M7 families (including M6i, M6a, M6g, M7i, M7a, M7g)\n  • Compute Optimized: C3-C7 families (including C6i, C6a, C6g, C7i, C7a, C7g)\n  • Memory Optimized: R3-R7 families (including R6i, R6a, R6g, R7i, R7a, R7g), X1, X1e\n  • Accelerated Computing: G4dn, G4ad, G5, G5g\n\nTip: Remove --hibernation flag or choose a different instance size", instanceType)
		}

		runInput.HibernationOptions = &ec2types.HibernationOptionsRequest{
			Configured: aws.Bool(true),
		}

		// Enable EBS encryption for root volume (required for hibernation)
		rootDevice := "/dev/sda1"
		if strings.Contains(strings.ToLower(ami), "amazon") || strings.Contains(strings.ToLower(ami), "amzn") {
			rootDevice = "/dev/xvda"
		}

		runInput.BlockDeviceMappings = []ec2types.BlockDeviceMapping{
			{
				DeviceName: aws.String(rootDevice),
				Ebs: &ec2types.EbsBlockDevice{
					VolumeType:          ec2types.VolumeTypeGp3,
					VolumeSize:          aws.Int32(20),
					Encrypted:           aws.Bool(true),
					DeleteOnTermination: aws.Bool(true),
				},
			},
		}
	}

	// Add spot instance support if requested
	if req.Spot {
		runInput.InstanceMarketOptions = &ec2types.InstanceMarketOptionsRequest{
			MarketType: ec2types.MarketTypeSpot,
			SpotOptions: &ec2types.SpotMarketOptions{
				SpotInstanceType: ec2types.SpotInstanceTypeOneTime,
			},
		}
	}

	return nil
}

// InstanceLauncher executes the actual instance launch (Single Responsibility - SOLID)
type InstanceLauncher struct {
	manager *Manager
}

// LaunchInstance executes EC2 instance launch and returns result
func (l *InstanceLauncher) LaunchInstance(req ctypes.LaunchRequest, runInput *ec2.RunInstancesInput, dailyCost float64) (*ctypes.Instance, error) {
	// Handle dry run
	if req.DryRun {
		return &ctypes.Instance{
			Name:               req.Name,
			Template:           req.Template,
			State:              "dry-run",
			EstimatedDailyCost: dailyCost,
		}, nil
	}

	// Launch instance
	ctx := context.Background()
	result, err := l.manager.ec2.RunInstances(ctx, runInput)
	if err != nil {
		return nil, fmt.Errorf("failed to launch instance: %w", err)
	}

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

// LaunchOrchestrator coordinates instance launch using SOLID principles (Strategy Pattern - SOLID)
type LaunchOrchestrator struct {
	configExtractor     *TemplateConfigExtractor
	userDataProcessor   *UserDataProcessor
	networkingResolver  *NetworkingResolver
	configBuilder       *InstanceConfigBuilder
	optionsProcessor    *LaunchOptionsProcessor
	instanceLauncher    *InstanceLauncher
}

// NewLaunchOrchestrator creates launch orchestrator
func NewLaunchOrchestrator(manager *Manager, region string) *LaunchOrchestrator {
	return &LaunchOrchestrator{
		configExtractor:    &TemplateConfigExtractor{region: region},
		userDataProcessor:  &UserDataProcessor{manager: manager, region: region},
		networkingResolver: &NetworkingResolver{manager: manager},
		configBuilder:      &InstanceConfigBuilder{manager: manager},
		optionsProcessor:   &LaunchOptionsProcessor{manager: manager},
		instanceLauncher:   &InstanceLauncher{manager: manager},
	}
}

// ExecuteLaunch performs complete instance launch using SOLID strategy pattern
func (o *LaunchOrchestrator) ExecuteLaunch(req ctypes.LaunchRequest, template *ctypes.RuntimeTemplate, arch string) (*ctypes.Instance, error) {
	// Extract template configuration
	ami, instanceType, dailyCost, err := o.configExtractor.ExtractConfig(template, arch)
	if err != nil {
		return nil, err
	}

	// Process user data
	userDataEncoded := o.userDataProcessor.ProcessUserData(template, req)

	// Resolve networking
	_, subnetID, securityGroupID, err := o.networkingResolver.ResolveNetworking(req)
	if err != nil {
		return nil, err
	}

	// Build run configuration
	runInput, err := o.configBuilder.BuildRunInstancesInput(req, ami, instanceType, userDataEncoded, subnetID, securityGroupID)
	if err != nil {
		return nil, err
	}

	// Process launch options
	if err := o.optionsProcessor.ProcessOptions(req, runInput, ami, instanceType); err != nil {
		return nil, err
	}

	// Execute launch
	return o.instanceLauncher.LaunchInstance(req, runInput, dailyCost)
}

// launchWithUnifiedTemplateSystem launches instance using unified template system with SOLID orchestration (SOLID: Single Responsibility)
func (m *Manager) launchWithUnifiedTemplateSystem(req ctypes.LaunchRequest, arch string) (*ctypes.Instance, error) {
	// Get template using unified template system
	packageManager := req.PackageManager
	if packageManager == "" {
		packageManager = ""
	}

	template, err := templates.GetTemplateWithPackageManager(req.Template, m.region, arch, packageManager, req.Size)
	if err != nil {
		return nil, fmt.Errorf("failed to get template: %w", err)
	}

	// Create orchestrator and execute launch
	orchestrator := NewLaunchOrchestrator(m, m.region)
	return orchestrator.ExecuteLaunch(req, template, arch)
}

// DeleteInstance terminates an EC2 instance
func (m *Manager) DeleteInstance(name string) error {
	// Find instance by name tag
	instanceID, err := m.findInstanceByName(name)
	if err != nil {
		return fmt.Errorf("failed to find instance: %w", err)
	}

	// Terminate the instance
	ctx := context.Background()
	_, err = m.ec2.TerminateInstances(ctx, &ec2.TerminateInstancesInput{
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
	ctx := context.Background()
	_, err = m.ec2.StartInstances(ctx, &ec2.StartInstancesInput{
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

	ctx := context.Background()
	// Stop the instance
	_, err = m.ec2.StopInstances(ctx, &ec2.StopInstancesInput{
		InstanceIds: []string{instanceID},
	})
	if err != nil {
		return fmt.Errorf("failed to stop instance: %w", err)
	}

	return nil
}

// HibernateInstance hibernates (pauses) a running EC2 instance
// This preserves the RAM state to storage for faster resume than regular stop/start
func (m *Manager) HibernateInstance(name string) error {
	// Find instance by name tag
	instanceID, err := m.findInstanceByName(name)
	if err != nil {
		return fmt.Errorf("failed to find instance: %w", err)
	}

	ctx := context.Background()
	// Check if instance supports hibernation
	result, err := m.ec2.DescribeInstances(ctx, &ec2.DescribeInstancesInput{
		InstanceIds: []string{instanceID},
	})
	if err != nil {
		return fmt.Errorf("failed to describe instance: %w", err)
	}

	if len(result.Reservations) == 0 || len(result.Reservations[0].Instances) == 0 {
		return fmt.Errorf("instance not found")
	}

	instance := result.Reservations[0].Instances[0]

	// Check if hibernation is enabled for this instance
	if instance.HibernationOptions == nil {
		fmt.Printf("⚠️  Instance %s does not support hibernation (hibernation options not found)\n", name)
		fmt.Printf("    Falling back to regular stop operation\n")
		return m.StopInstance(name)
	}

	if !*instance.HibernationOptions.Configured {
		fmt.Printf("⚠️  Instance %s does not support hibernation (hibernation not configured)\n", name)
		fmt.Printf("    Falling back to regular stop operation\n")
		return m.StopInstance(name)
	}

	// Check if instance is in a state that can be hibernated
	if instance.State == nil {
		return fmt.Errorf("instance state unknown")
	}

	instanceState := string(instance.State.Name)
	if instanceState != "running" {
		return fmt.Errorf("instance must be in 'running' state to hibernate (current state: %s)", instanceState)
	}

	// Check if instance has been running long enough for hibernation agent to be ready
	// AWS hibernation agent typically needs 2-3 minutes after launch to be ready
	if instance.LaunchTime != nil {
		timeSinceLaunch := time.Since(*instance.LaunchTime)
		minReadyTime := 3 * time.Minute

		if timeSinceLaunch < minReadyTime {
			remainingTime := minReadyTime - timeSinceLaunch
			return fmt.Errorf("instance not ready for hibernation yet (launched %v ago, need %v). Wait %v more",
				timeSinceLaunch.Round(time.Second),
				minReadyTime,
				remainingTime.Round(time.Second))
		}
	}

	// Stop the instance with hibernation
	_, err = m.ec2.StopInstances(ctx, &ec2.StopInstancesInput{
		InstanceIds: []string{instanceID},
		Hibernate:   aws.Bool(true), // This enables hibernation
	})
	if err != nil {
		return fmt.Errorf("failed to hibernate instance: %w", err)
	}

	return nil
}

// ResumeInstance resumes a hibernated instance (same as StartInstance for hibernated instances)
func (m *Manager) ResumeInstance(name string) error {
	// Resume is the same as start for hibernated instances
	return m.StartInstance(name)
}

// GetInstanceHibernationStatus returns whether the instance supports and is configured for hibernation
func (m *Manager) GetInstanceHibernationStatus(name string) (bool, bool, error) {
	// Find instance by name tag
	instanceID, err := m.findInstanceByName(name)
	if err != nil {
		return false, false, fmt.Errorf("failed to find instance: %w", err)
	}

	ctx := context.Background()
	// Get instance details
	result, err := m.ec2.DescribeInstances(ctx, &ec2.DescribeInstancesInput{
		InstanceIds: []string{instanceID},
	})
	if err != nil {
		return false, false, fmt.Errorf("failed to describe instance: %w", err)
	}

	if len(result.Reservations) == 0 || len(result.Reservations[0].Instances) == 0 {
		return false, false, fmt.Errorf("instance not found")
	}

	instance := result.Reservations[0].Instances[0]

	// Check hibernation configuration
	hibernationSupported := instance.HibernationOptions != nil && *instance.HibernationOptions.Configured

	// Check if currently hibernated (stopped with hibernation)
	isHibernated := hibernationSupported && instance.State != nil && string(instance.State.Name) == "stopped"

	return hibernationSupported, isHibernated, nil
}

// GetConnectionInfo returns connection information for an instance with SSH key path
func (m *Manager) GetConnectionInfo(name string) (string, error) {
	// Find instance by name tag
	instanceID, err := m.findInstanceByName(name)
	if err != nil {
		return "", fmt.Errorf("failed to find instance: %w", err)
	}

	ctx := context.Background()
	// Get instance details
	result, err := m.ec2.DescribeInstances(ctx, &ec2.DescribeInstancesInput{
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

	// Get SSH key information
	sshKeyInfo := ""
	if instance.KeyName != nil {
		keyPath, err := m.getSSHKeyPathFromKeyName(*instance.KeyName)
		if err == nil {
			sshKeyInfo = fmt.Sprintf(" -i \"%s\"", keyPath)
		}
	}

	return fmt.Sprintf("ssh%s ubuntu@%s", sshKeyInfo, *instance.PublicIpAddress), nil
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

	ctx := context.Background()
	result, err := m.efs.CreateFileSystem(ctx, input)
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
		EstimatedCostGB: m.getRegionalEFSPrice(), // Regional EFS pricing
		SizeBytes:       0,                       // Will be updated as files are added
	}

	return volume, nil
}

// DeleteVolume deletes an EFS volume
func (m *Manager) DeleteVolume(name string) error {
	// Get volume state to find the FileSystemId
	state, err := m.stateManager.LoadState()
	if err != nil {
		return fmt.Errorf("failed to load state: %w", err)
	}

	volume, exists := state.Volumes[name]
	if !exists {
		return fmt.Errorf("volume '%s' not found in state", name)
	}

	// Check if the filesystem exists
	fsId := volume.FileSystemId
	if fsId == "" {
		return fmt.Errorf("no filesystem ID found for volume '%s'", name)
	}

	log.Printf("Deleting EFS volume '%s' (filesystem ID: %s)...", name, fsId)

	ctx := context.Background()
	// 1. Delete all mount targets first
	// List mount targets for the file system
	mtResp, err := m.efs.DescribeMountTargets(ctx, &efs.DescribeMountTargetsInput{
		FileSystemId: aws.String(fsId),
	})
	if err != nil {
		return fmt.Errorf("failed to list mount targets: %w", err)
	}

	// Delete all mount targets
	for _, mt := range mtResp.MountTargets {
		mountTargetId := *mt.MountTargetId
		log.Printf("Deleting mount target %s...", mountTargetId)

		_, err := m.efs.DeleteMountTarget(ctx, &efs.DeleteMountTargetInput{
			MountTargetId: aws.String(mountTargetId),
		})
		if err != nil {
			return fmt.Errorf("failed to delete mount target %s: %w", mountTargetId, err)
		}
	}

	// 2. Wait for mount targets to be deleted
	// The file system can't be deleted until all mount targets are deleted
	if len(mtResp.MountTargets) > 0 {
		log.Printf("Waiting for mount targets to be deleted...")

		// Poll until all mount targets are gone
		for i := 0; i < 30; i++ { // Try for up to 5 minutes (30 * 10 seconds)
			// Check if mount targets still exist
			mtCheck, err := m.efs.DescribeMountTargets(ctx, &efs.DescribeMountTargetsInput{
				FileSystemId: aws.String(fsId),
			})
			if err != nil {
				// If the file system itself is gone, we don't care about mount targets
				if strings.Contains(err.Error(), "FileSystemNotFound") {
					break
				}
				return fmt.Errorf("error checking mount targets: %w", err)
			}

			// If no mount targets remain, we can proceed
			if len(mtCheck.MountTargets) == 0 {
				break
			}

			// Wait before checking again
			time.Sleep(10 * time.Second)
		}
	}

	// 3. Delete the file system
	log.Printf("Deleting EFS file system %s...", fsId)
	_, err = m.efs.DeleteFileSystem(ctx, &efs.DeleteFileSystemInput{
		FileSystemId: aws.String(fsId),
	})
	if err != nil {
		return fmt.Errorf("failed to delete file system: %w", err)
	}

	// 4. Remove from state
	return m.stateManager.RemoveVolume(name)
}

// MountVolume mounts an EFS volume to an instance
func (m *Manager) MountVolume(volumeName, instanceName, mountPoint string) error {
	// Get volume state to find the FileSystemId
	state, err := m.stateManager.LoadState()
	if err != nil {
		return fmt.Errorf("failed to load state: %w", err)
	}

	volume, exists := state.Volumes[volumeName]
	if !exists {
		return fmt.Errorf("volume '%s' not found in state", volumeName)
	}

	// Get instance information
	instance, exists := state.Instances[instanceName]
	if !exists {
		return fmt.Errorf("instance '%s' not found in state", instanceName)
	}

	// Check if instance is running
	if instance.State != "running" {
		return fmt.Errorf("instance '%s' is not running (state: %s)", instanceName, instance.State)
	}

	fsId := volume.FileSystemId
	if fsId == "" {
		return fmt.Errorf("no filesystem ID found for volume '%s'", volumeName)
	}

	log.Printf("Mounting EFS volume '%s' (filesystem ID: %s) to instance '%s' at %s...", volumeName, fsId, instanceName, mountPoint)

	// Create mount command script with shared group setup
	mountScript := fmt.Sprintf(`#!/bin/bash
set -e

# Install EFS utils if not already installed
if ! command -v mount.efs &> /dev/null; then
    if command -v yum &> /dev/null; then
        sudo yum install -y amazon-efs-utils
    elif command -v apt &> /dev/null; then
        sudo apt-get update && sudo apt-get install -y amazon-efs-utils
    else
        echo "Unsupported package manager"
        exit 1
    fi
fi

# Create CloudWorkstation shared group if it doesn't exist
if ! getent group cloudworkstation-shared >/dev/null 2>&1; then
    sudo groupadd -g 3000 cloudworkstation-shared
    echo "Created cloudworkstation-shared group (gid: 3000)"
fi

# Add current user to shared group if not already a member
CURRENT_USER=$(whoami)
if ! groups "$CURRENT_USER" | grep -q cloudworkstation-shared; then
    sudo usermod -a -G cloudworkstation-shared "$CURRENT_USER"
    echo "Added $CURRENT_USER to cloudworkstation-shared group"
fi

# Create mount directory
sudo mkdir -p %s

# Mount the EFS volume with shared group ownership
sudo mount -t efs %s:/ %s -o tls,_netdev,gid=3000

# Set proper permissions for shared access
sudo chmod 2775 %s  # Group sticky bit + group writable
sudo chgrp cloudworkstation-shared %s

# Create shared subdirectories with proper permissions
sudo mkdir -p %s/shared %s/users
sudo chmod 2775 %s/shared
sudo chmod 2755 %s/users
sudo chgrp cloudworkstation-shared %s/shared %s/users

# Create user-specific directory if it doesn't exist
sudo mkdir -p %s/users/$CURRENT_USER
sudo chown $CURRENT_USER:cloudworkstation-shared %s/users/$CURRENT_USER
sudo chmod 755 %s/users/$CURRENT_USER

# Add to fstab for persistence with group mount option
if ! grep -q "%s" /etc/fstab; then
    echo "%s:/ %s efs tls,_netdev,gid=3000" | sudo tee -a /etc/fstab
fi

# Set default umask for group-friendly permissions
echo "umask 002" | sudo tee -a /etc/bash.bashrc >/dev/null 2>&1 || true
echo "umask 002" | sudo tee -a /etc/zsh/zshrc >/dev/null 2>&1 || true

echo "EFS volume mounted successfully at %s with shared group access"
echo "  - Shared files: %s/shared (all users)"
echo "  - User files: %s/users/$CURRENT_USER (personal)" 
echo "  - Group: cloudworkstation-shared (gid: 3000)"
`, mountPoint, fsId, mountPoint, mountPoint, mountPoint, mountPoint, mountPoint, mountPoint, mountPoint, mountPoint, mountPoint, mountPoint, mountPoint, mountPoint, fsId, fsId, mountPoint, mountPoint, mountPoint, mountPoint)

	// Execute mount script on the instance via SSM
	err = m.executeScriptOnInstance(instance.ID, mountScript)
	if err != nil {
		return fmt.Errorf("failed to mount EFS volume: %w", err)
	}

	log.Printf("Successfully mounted EFS volume '%s' to instance '%s' at %s", volumeName, instanceName, mountPoint)
	return nil
}

// UnmountVolume unmounts an EFS volume from an instance
func (m *Manager) UnmountVolume(volumeName, instanceName string) error {
	// Get volume and instance state
	state, err := m.stateManager.LoadState()
	if err != nil {
		return fmt.Errorf("failed to load state: %w", err)
	}

	volume, exists := state.Volumes[volumeName]
	if !exists {
		return fmt.Errorf("volume '%s' not found in state", volumeName)
	}

	instance, exists := state.Instances[instanceName]
	if !exists {
		return fmt.Errorf("instance '%s' not found in state", instanceName)
	}

	fsId := volume.FileSystemId
	if fsId == "" {
		return fmt.Errorf("no filesystem ID found for volume '%s'", volumeName)
	}

	log.Printf("Unmounting EFS volume '%s' (filesystem ID: %s) from instance '%s'...", volumeName, fsId, instanceName)

	// Create unmount command script
	unmountScript := fmt.Sprintf(`#!/bin/bash
set -e

# Find mount points for this EFS volume
MOUNT_POINTS=$(mount | grep %s | awk '{print $3}' || true)

if [ -z "$MOUNT_POINTS" ]; then
    echo "No mount points found for EFS volume %s"
    exit 0
fi

# Unmount each mount point
for MOUNT_POINT in $MOUNT_POINTS; do
    echo "Unmounting $MOUNT_POINT..."
    sudo umount "$MOUNT_POINT" || true
    
    # Remove from fstab
    sudo sed -i "\|%s:/|d" /etc/fstab || true
    
    echo "Successfully unmounted $MOUNT_POINT"
done

echo "EFS volume unmounted successfully"
`, fsId, fsId, fsId)

	// Execute unmount script on the instance via SSM
	err = m.executeScriptOnInstance(instance.ID, unmountScript)
	if err != nil {
		return fmt.Errorf("failed to unmount EFS volume: %w", err)
	}

	log.Printf("Successfully unmounted EFS volume '%s' from instance '%s'", volumeName, instanceName)
	return nil
}

// executeScriptOnInstance executes a shell script on an instance using SSM
func (m *Manager) executeScriptOnInstance(instanceID, script string) error {
	ctx := context.Background()
	// Send command via SSM
	output, err := m.ssm.SendCommand(ctx, &ssm.SendCommandInput{
		InstanceIds:  []string{instanceID},
		DocumentName: aws.String("AWS-RunShellScript"),
		Parameters: map[string][]string{
			"commands": {script},
		},
		Comment: aws.String("CloudWorkstation EFS mount/unmount operation"),
	})
	if err != nil {
		return fmt.Errorf("failed to send SSM command: %w", err)
	}

	commandID := *output.Command.CommandId
	log.Printf("Sent SSM command %s to instance %s", commandID, instanceID)

	// Wait for command completion with timeout
	maxAttempts := 30 // Wait up to 5 minutes (30 * 10 seconds)
	for attempt := 0; attempt < maxAttempts; attempt++ {
		// Get command invocation status
		invocation, err := m.ssm.GetCommandInvocation(ctx, &ssm.GetCommandInvocationInput{
			CommandId:  aws.String(commandID),
			InstanceId: aws.String(instanceID),
		})
		if err != nil {
			time.Sleep(10 * time.Second)
			continue
		}

		status := string(invocation.Status)
		switch status {
		case "Success":
			log.Printf("SSM command %s completed successfully", commandID)
			if invocation.StandardOutputContent != nil {
				log.Printf("Command output: %s", *invocation.StandardOutputContent)
			}
			return nil
		case "Failed":
			errorMsg := "Unknown error"
			if invocation.StandardErrorContent != nil {
				errorMsg = *invocation.StandardErrorContent
			}
			return fmt.Errorf("SSM command failed: %s", errorMsg)
		case "Cancelled", "TimedOut":
			return fmt.Errorf("SSM command %s: %s", status, commandID)
		case "InProgress", "Pending", "Delayed":
			// Continue waiting
			time.Sleep(10 * time.Second)
			continue
		default:
			log.Printf("Unknown SSM command status: %s", status)
			time.Sleep(10 * time.Second)
			continue
		}
	}

	return fmt.Errorf("SSM command %s timed out after waiting 5 minutes", commandID)
}

// CreateStorage creates a new EBS volume
func (m *Manager) CreateStorage(req ctypes.StorageCreateRequest) (*ctypes.EBSVolume, error) {
	// Parse size from t-shirt sizes or use direct GB value
	sizeGB, err := m.parseSizeToGB(req.Size)
	if err != nil {
		return nil, fmt.Errorf("invalid size: %w", err)
	}

	// Set defaults
	volumeType := "gp3"
	if req.VolumeType != "" {
		volumeType = req.VolumeType
	}

	// Calculate IOPS and throughput for gp3 volumes
	iops, throughput := m.calculatePerformanceParams(volumeType, sizeGB)

	// Create EBS volume
	input := &ec2.CreateVolumeInput{
		Size:             aws.Int32(int32(sizeGB)),
		VolumeType:       ec2types.VolumeType(volumeType),
		AvailabilityZone: aws.String(m.region + "a"), // Use first AZ
		TagSpecifications: []ec2types.TagSpecification{
			{
				ResourceType: ec2types.ResourceTypeVolume,
				Tags: []ec2types.Tag{
					{
						Key:   aws.String("Name"),
						Value: aws.String(req.Name),
					},
					{
						Key:   aws.String("CloudWorkstation"),
						Value: aws.String("true"),
					},
				},
			},
		},
	}

	// Set IOPS for io2 and gp3 volumes
	if volumeType == "io2" || volumeType == "gp3" {
		input.Iops = aws.Int32(int32(iops))
	}

	// Set throughput for gp3 volumes
	if volumeType == "gp3" {
		input.Throughput = aws.Int32(int32(throughput))
	}

	ctx := context.Background()
	result, err := m.ec2.CreateVolume(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to create EBS volume: %w", err)
	}

	// Calculate cost per GB per month
	costPerGB := m.getEBSCostPerGB(volumeType)

	volume := &ctypes.EBSVolume{
		Name:            req.Name,
		VolumeID:        *result.VolumeId,
		Region:          m.region,
		CreationTime:    time.Now(),
		State:           string(result.State),
		VolumeType:      volumeType,
		SizeGB:          int32(sizeGB),
		IOPS:            int32(iops),
		Throughput:      int32(throughput),
		EstimatedCostGB: costPerGB,
		AttachedTo:      "", // Not attached initially
	}

	return volume, nil
}

// DeleteStorage deletes an EBS volume
func (m *Manager) DeleteStorage(name string) error {
	// Find volume by name tag
	volumeID, err := m.findVolumeByName(name)
	if err != nil {
		return fmt.Errorf("failed to find volume: %w", err)
	}

	ctx := context.Background()
	// Delete the volume
	_, err = m.ec2.DeleteVolume(ctx, &ec2.DeleteVolumeInput{
		VolumeId: aws.String(volumeID),
	})
	if err != nil {
		return fmt.Errorf("failed to delete volume: %w", err)
	}

	return nil
}

// AttachStorage attaches an EBS volume to an instance
func (m *Manager) AttachStorage(volumeName, instanceName string) error {
	// Find volume by name tag
	volumeID, err := m.findVolumeByName(volumeName)
	if err != nil {
		return fmt.Errorf("failed to find volume: %w", err)
	}

	// Find instance by name tag
	instanceID, err := m.findInstanceByName(instanceName)
	if err != nil {
		return fmt.Errorf("failed to find instance: %w", err)
	}

	// Find next available device name (start with /dev/sdf)
	deviceName := "/dev/sdf"

	ctx := context.Background()
	// Attach volume to instance
	_, err = m.ec2.AttachVolume(ctx, &ec2.AttachVolumeInput{
		VolumeId:   aws.String(volumeID),
		InstanceId: aws.String(instanceID),
		Device:     aws.String(deviceName),
	})
	if err != nil {
		return fmt.Errorf("failed to attach volume: %w", err)
	}

	return nil
}

// DetachStorage detaches an EBS volume from an instance
func (m *Manager) DetachStorage(volumeName string) error {
	// Find volume by name tag
	volumeID, err := m.findVolumeByName(volumeName)
	if err != nil {
		return fmt.Errorf("failed to find volume: %w", err)
	}

	ctx := context.Background()
	// Detach the volume
	_, err = m.ec2.DetachVolume(ctx, &ec2.DetachVolumeInput{
		VolumeId: aws.String(volumeID),
	})
	if err != nil {
		return fmt.Errorf("failed to detach volume: %w", err)
	}

	return nil
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

// processIdleDetectionConfig replaces idle detection configuration placeholders in UserData
func (m *Manager) processIdleDetectionConfig(userData string, template *ctypes.RuntimeTemplate) string {
	// Check if template has idle detection configuration
	if template.IdleDetection == nil {
		return userData
	}

	// Replace configuration placeholders
	userData = strings.ReplaceAll(userData, "{{IDLE_THRESHOLD_MINUTES}}", fmt.Sprintf("%d", template.IdleDetection.IdleThresholdMinutes))
	userData = strings.ReplaceAll(userData, "{{HIBERNATE_THRESHOLD_MINUTES}}", fmt.Sprintf("%d", template.IdleDetection.HibernateThresholdMinutes))
	userData = strings.ReplaceAll(userData, "{{CHECK_INTERVAL_MINUTES}}", fmt.Sprintf("%d", template.IdleDetection.CheckIntervalMinutes))

	return userData
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
	ctx := context.Background()
	result, err := m.ec2.DescribeInstances(ctx, &ec2.DescribeInstancesInput{
		Filters: []ec2types.Filter{
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

// StateLoader loads local state for instance merging (Single Responsibility - SOLID)
type StateLoader struct{}

// LoadLocalState attempts to load local state for deletion timestamp merging
func (l *StateLoader) LoadLocalState() *ctypes.State {
	stateManager, err := state.NewManager()
	if err != nil {
		return nil // Continue without state merging if unavailable
	}

	localState, _ := stateManager.LoadState()
	return localState
}

// InstanceTagExtractor extracts tags from EC2 instances (Single Responsibility - SOLID)
type InstanceTagExtractor struct{}

// ExtractTags extracts CloudWorkstation-specific tags from EC2 instance
func (e *InstanceTagExtractor) ExtractTags(ec2Instance ec2types.Instance) (name, template, project string) {
	for _, tag := range ec2Instance.Tags {
		if tag.Key != nil && tag.Value != nil {
			switch *tag.Key {
			case "Name":
				name = *tag.Value
			case "Template":
				template = *tag.Value
			case "Project":
				project = *tag.Value
			}
		}
	}
	return name, template, project
}

// InstanceStateConverter converts AWS states to CloudWorkstation states (Single Responsibility - SOLID)
type InstanceStateConverter struct{}

// ConvertState converts EC2 instance state to CloudWorkstation state
func (c *InstanceStateConverter) ConvertState(ec2Instance ec2types.Instance) string {
	if ec2Instance.State == nil {
		return "unknown"
	}

	awsState := string(ec2Instance.State.Name)

	// Check for hibernation states
	if c.isHibernationConfigured(ec2Instance) {
		if awsState == "stopped" {
			return "hibernated"
		} else if awsState == "stopping" {
			return "hibernating"
		}
	}

	return awsState
}

func (c *InstanceStateConverter) isHibernationConfigured(ec2Instance ec2types.Instance) bool {
	return ec2Instance.HibernationOptions != nil && *ec2Instance.HibernationOptions.Configured
}

// InstanceBuilder builds CloudWorkstation instance objects (Builder Pattern - SOLID)
type InstanceBuilder struct {
	tagExtractor    *InstanceTagExtractor
	stateConverter  *InstanceStateConverter
}

// NewInstanceBuilder creates instance builder
func NewInstanceBuilder() *InstanceBuilder {
	return &InstanceBuilder{
		tagExtractor:   &InstanceTagExtractor{},
		stateConverter: &InstanceStateConverter{},
	}
}

// BuildInstance creates CloudWorkstation instance from EC2 instance
func (b *InstanceBuilder) BuildInstance(ec2Instance ec2types.Instance, localState *ctypes.State) *ctypes.Instance {
	// Extract tags
	name, template, project := b.tagExtractor.ExtractTags(ec2Instance)

	// Skip instances without names
	if name == "" {
		return nil
	}

	// Convert state
	state := b.stateConverter.ConvertState(ec2Instance)

	// Get public IP
	publicIP := ""
	if ec2Instance.PublicIpAddress != nil {
		publicIP = *ec2Instance.PublicIpAddress
	}

	// Determine instance lifecycle
	instanceLifecycle := "on-demand"
	if string(ec2Instance.InstanceLifecycle) == "spot" {
		instanceLifecycle = "spot"
	}

	// Create instance
	instance := &ctypes.Instance{
		ID:                 *ec2Instance.InstanceId,
		Name:               name,
		Template:           template,
		State:              state,
		PublicIP:           publicIP,
		ProjectID:          project,
		InstanceLifecycle:  instanceLifecycle,
		LaunchTime:         *ec2Instance.LaunchTime,
		EstimatedDailyCost: 0.0, // TODO: Calculate based on instance type
	}

	// Merge deletion time from local state if available
	if localState != nil {
		if localInstance, exists := localState.Instances[name]; exists {
			instance.DeletionTime = localInstance.DeletionTime
		}
	}

	return instance
}

// InstanceListProcessor processes EC2 reservations into CloudWorkstation instances (Strategy Pattern - SOLID)
type InstanceListProcessor struct {
	stateLoader     *StateLoader
	instanceBuilder *InstanceBuilder
}

// NewInstanceListProcessor creates instance list processor
func NewInstanceListProcessor() *InstanceListProcessor {
	return &InstanceListProcessor{
		stateLoader:     &StateLoader{},
		instanceBuilder: NewInstanceBuilder(),
	}
}

// ProcessReservations converts EC2 reservations to CloudWorkstation instances
func (p *InstanceListProcessor) ProcessReservations(reservations []ec2types.Reservation) []ctypes.Instance {
	localState := p.stateLoader.LoadLocalState()
	var instances []ctypes.Instance

	for _, reservation := range reservations {
		for _, ec2Instance := range reservation.Instances {
			if instance := p.instanceBuilder.BuildInstance(ec2Instance, localState); instance != nil {
				instances = append(instances, *instance)
			}
		}
	}

	return instances
}

// ListInstances returns all CloudWorkstation instances using Strategy Pattern (SOLID: Single Responsibility)
func (m *Manager) ListInstances() ([]ctypes.Instance, error) {
	ctx := context.Background()
	result, err := m.ec2.DescribeInstances(ctx, &ec2.DescribeInstancesInput{
		Filters: []ec2types.Filter{
			{
				Name:   aws.String("tag:CloudWorkstation"),
				Values: []string{"true"},
			},
			{
				Name:   aws.String("instance-state-name"),
				Values: []string{"pending", "running", "shutting-down", "stopping", "stopped", "terminating", "terminated"},
			},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to describe instances: %w", err)
	}

	processor := NewInstanceListProcessor()
	return processor.ProcessReservations(result.Reservations), nil
}

// parseSizeToGB converts t-shirt sizes to GB
func (m *Manager) parseSizeToGB(size string) (int, error) {
	switch size {
	case "XS", "xs":
		return 100, nil
	case "S", "s":
		return 500, nil
	case "M", "m":
		return 1000, nil
	case "L", "l":
		return 2000, nil
	case "XL", "xl":
		return 4000, nil
	default:
		// Try to parse as direct GB value
		var gb int
		if _, err := fmt.Sscanf(size, "%d", &gb); err == nil && gb > 0 {
			return gb, nil
		}
		return 0, fmt.Errorf("invalid size '%s'. Use XS/S/M/L/XL or GB value", size)
	}
}

// calculatePerformanceParams calculates optimal IOPS and throughput for EBS volumes
func (m *Manager) calculatePerformanceParams(volumeType string, sizeGB int) (int, int) {
	switch volumeType {
	case "gp3":
		// gp3: 3 IOPS per GB baseline, max 16,000 IOPS
		// 125 MB/s baseline throughput, max 1,000 MB/s
		iops := sizeGB * 3
		if iops > 16000 {
			iops = 16000
		}
		if iops < 3000 {
			iops = 3000 // Minimum for gp3
		}

		throughput := sizeGB / 4 // Rough approximation
		if throughput > 1000 {
			throughput = 1000
		}
		if throughput < 125 {
			throughput = 125
		}

		return iops, throughput

	case "io2":
		// io2: Up to 500 IOPS per GB, max 64,000 IOPS
		iops := sizeGB * 10 // Conservative for cost
		if iops > 64000 {
			iops = 64000
		}
		if iops < 100 {
			iops = 100
		}
		return iops, 0 // throughput not applicable for io2

	default:
		return 0, 0 // No IOPS/throughput configuration for other types
	}
}

// getEBSCostPerGB returns the cost per GB per month for different EBS volume types
func (m *Manager) getEBSCostPerGB(volumeType string) float64 {
	basePrice := m.getRegionalEBSPrice(volumeType)
	return m.applyEBSDiscounts(basePrice)
}

// getRegionalEBSPrice returns region-aware EBS pricing with smart caching
func (m *Manager) getRegionalEBSPrice(volumeType string) float64 {
	// Check cache first (24-hour TTL)
	cacheKey := fmt.Sprintf("ebs-%s-%s", volumeType, m.region)
	if cachedPrice, exists := m.pricingCache[cacheKey]; exists {
		if time.Since(m.lastPriceUpdate) < 24*time.Hour {
			return cachedPrice
		}
	}

	// Get regional pricing multiplier
	regionMultiplier := m.getRegionPricingMultiplier()

	// Base US East 1 pricing (most accurate)
	var basePrice float64
	switch volumeType {
	case "gp3":
		basePrice = 0.08 // $0.08 per GB per month in us-east-1
	case "gp2":
		basePrice = 0.10 // $0.10 per GB per month in us-east-1
	case "io2":
		basePrice = 0.125 // $0.125 per GB per month in us-east-1
	case "st1":
		basePrice = 0.045 // $0.045 per GB per month in us-east-1
	case "sc1":
		basePrice = 0.025 // $0.025 per GB per month in us-east-1
	default:
		basePrice = 0.10 // Default to gp2 pricing
	}

	regionalPrice := basePrice * regionMultiplier

	// Cache the result
	m.pricingCache[cacheKey] = regionalPrice
	m.lastPriceUpdate = time.Now()

	return regionalPrice
}

// getRegionPricingMultiplier returns pricing multiplier for different AWS regions
func (m *Manager) getRegionPricingMultiplier() float64 {
	// Regional pricing multipliers based on AWS public pricing patterns
	switch {
	case strings.HasPrefix(m.region, "us-east-1"):
		return 1.0 // Base pricing
	case strings.HasPrefix(m.region, "us-east-2"):
		return 0.98 // Slightly cheaper
	case strings.HasPrefix(m.region, "us-west-"):
		return 1.05 // West coast premium
	case strings.HasPrefix(m.region, "eu-west-1"):
		return 1.10 // Ireland
	case strings.HasPrefix(m.region, "eu-west-2"):
		return 1.12 // London
	case strings.HasPrefix(m.region, "eu-west-3"):
		return 1.15 // Paris
	case strings.HasPrefix(m.region, "eu-central-1"):
		return 1.18 // Frankfurt
	case strings.HasPrefix(m.region, "ap-southeast-1"):
		return 1.20 // Singapore
	case strings.HasPrefix(m.region, "ap-southeast-2"):
		return 1.25 // Sydney
	case strings.HasPrefix(m.region, "ap-northeast-1"):
		return 1.22 // Tokyo
	case strings.HasPrefix(m.region, "ap-northeast-2"):
		return 1.18 // Seoul
	case strings.HasPrefix(m.region, "ap-south-1"):
		return 1.05 // Mumbai
	case strings.HasPrefix(m.region, "ca-central-1"):
		return 1.08 // Canada
	case strings.HasPrefix(m.region, "sa-east-1"):
		return 1.30 // São Paulo
	default:
		return 1.15 // Conservative default for other regions
	}
}

// getRegionalEC2Price returns region-aware EC2 pricing with smart caching
func (m *Manager) getRegionalEC2Price(instanceType string) float64 {
	// Check cache first (24-hour TTL)
	cacheKey := fmt.Sprintf("ec2-%s-%s", instanceType, m.region)
	if cachedPrice, exists := m.pricingCache[cacheKey]; exists {
		if time.Since(m.lastPriceUpdate) < 24*time.Hour {
			return cachedPrice
		}
	}

	// Get regional pricing multiplier
	regionMultiplier := m.getRegionPricingMultiplier()

	// Base US East 1 pricing for common instance types
	var basePrice float64
	switch instanceType {
	// General Purpose
	case "t3.micro":
		basePrice = 0.0104 // $0.0104 per hour
	case "t3.small":
		basePrice = 0.0208 // $0.0208 per hour
	case "t3.medium":
		basePrice = 0.0416 // $0.0416 per hour
	case "t3.large":
		basePrice = 0.0832 // $0.0832 per hour
	case "t3.xlarge":
		basePrice = 0.1664 // $0.1664 per hour
	case "t3.2xlarge":
		basePrice = 0.3328 // $0.3328 per hour

	// Compute Optimized
	case "c5.large":
		basePrice = 0.085 // $0.085 per hour
	case "c5.xlarge":
		basePrice = 0.17 // $0.17 per hour
	case "c5.2xlarge":
		basePrice = 0.34 // $0.34 per hour
	case "c5.4xlarge":
		basePrice = 0.68 // $0.68 per hour

	// Memory Optimized
	case "r5.large":
		basePrice = 0.126 // $0.126 per hour
	case "r5.xlarge":
		basePrice = 0.252 // $0.252 per hour
	case "r5.2xlarge":
		basePrice = 0.504 // $0.504 per hour
	case "r5.4xlarge":
		basePrice = 1.008 // $1.008 per hour

	// GPU Instances
	case "g4dn.xlarge":
		basePrice = 0.526 // $0.526 per hour
	case "g4dn.2xlarge":
		basePrice = 0.752 // $0.752 per hour
	case "g4dn.4xlarge":
		basePrice = 1.204 // $1.204 per hour

	default:
		// Estimate based on instance family and size
		basePrice = m.estimateInstancePrice(instanceType)
	}

	regionalPrice := basePrice * regionMultiplier

	// Apply discounts
	finalPrice := m.applyEC2Discounts(regionalPrice)

	// Cache the result
	m.pricingCache[cacheKey] = finalPrice
	m.lastPriceUpdate = time.Now()

	return finalPrice
}

// estimateInstancePrice estimates pricing for unknown instance types
func (m *Manager) estimateInstancePrice(instanceType string) float64 {
	// Extract instance family and size
	parts := strings.Split(instanceType, ".")
	if len(parts) != 2 {
		return 0.10 // Conservative fallback
	}

	family := parts[0]
	size := parts[1]

	// Base pricing by family (rough estimates)
	var familyBase float64
	switch {
	case strings.HasPrefix(family, "t3"):
		familyBase = 0.0104 // t3.micro base
	case strings.HasPrefix(family, "t4g"):
		familyBase = 0.0084 // ARM instances slightly cheaper
	case strings.HasPrefix(family, "c5"):
		familyBase = 0.085 // c5.large base
	case strings.HasPrefix(family, "r5"):
		familyBase = 0.126 // r5.large base
	case strings.HasPrefix(family, "g4"):
		familyBase = 0.526 // GPU base
	default:
		familyBase = 0.05 // Conservative default
	}

	// Size multiplier
	var sizeMultiplier float64
	switch size {
	case "nano":
		sizeMultiplier = 0.25
	case "micro":
		sizeMultiplier = 0.5
	case "small":
		sizeMultiplier = 1.0
	case "medium":
		sizeMultiplier = 2.0
	case "large":
		sizeMultiplier = 4.0
	case "xlarge":
		sizeMultiplier = 8.0
	case "2xlarge":
		sizeMultiplier = 16.0
	case "4xlarge":
		sizeMultiplier = 32.0
	case "8xlarge":
		sizeMultiplier = 64.0
	case "12xlarge":
		sizeMultiplier = 96.0
	case "16xlarge":
		sizeMultiplier = 128.0
	case "24xlarge":
		sizeMultiplier = 192.0
	default:
		sizeMultiplier = 4.0 // Default to large
	}

	return familyBase * sizeMultiplier
}

// getRegionalEFSPrice returns region-aware EFS pricing with smart caching
func (m *Manager) getRegionalEFSPrice() float64 {
	// Check cache first (24-hour TTL)
	cacheKey := fmt.Sprintf("efs-%s", m.region)
	if cachedPrice, exists := m.pricingCache[cacheKey]; exists {
		if time.Since(m.lastPriceUpdate) < 24*time.Hour {
			return cachedPrice
		}
	}

	// Get regional pricing multiplier
	regionMultiplier := m.getRegionPricingMultiplier()

	// Base US East 1 EFS pricing: $0.30 per GB per month for Standard storage
	basePrice := 0.30
	regionalPrice := basePrice * regionMultiplier

	// Apply discounts
	finalPrice := m.applyEFSDiscounts(regionalPrice)

	// Cache the result
	m.pricingCache[cacheKey] = finalPrice
	m.lastPriceUpdate = time.Now()

	return finalPrice
}

// GetBillingInfo retrieves current billing and credit information
func (m *Manager) GetBillingInfo() (*ctypes.BillingInfo, error) {
	// Note: AWS doesn't provide direct credit APIs, so this is a simplified approach
	// In practice, this would require parsing billing reports or using Cost Explorer

	info := &ctypes.BillingInfo{
		MonthToDateSpend: 0.0, // Would need Cost Explorer API
		ForecastedSpend:  0.0, // Would need Cost Explorer API
		Credits:          []ctypes.CreditInfo{},
		BillingPeriod:    time.Now().Format("2006-01"),
		LastUpdated:      time.Now(),
	}

	// Check for common credit scenarios based on account type
	credits := m.detectPotentialCredits()
	info.Credits = credits

	return info, nil
}

// detectPotentialCredits attempts to detect potential credit allocations
func (m *Manager) detectPotentialCredits() []ctypes.CreditInfo {
	var credits []ctypes.CreditInfo

	// Check account alias or organization info for common credit programs
	// This is a simplified approach - real implementation would need billing API access

	// Example: Check if account is part of AWS Educate
	// (In practice, this would require additional AWS APIs)

	// Mock credit for educational/startup accounts
	credits = append(credits, ctypes.CreditInfo{
		TotalCredits:     0.0,
		RemainingCredits: 0.0,
		UsedCredits:      0.0,
		CreditType:       "AWS Credits",
		Description:      "Credit information requires AWS billing API access",
	})

	return credits
}

// SetDiscountConfig configures pricing discounts for various AWS services
func (m *Manager) SetDiscountConfig(config ctypes.DiscountConfig) {
	m.discountConfig = config
	// Clear pricing cache to force recalculation with new discounts
	m.pricingCache = make(map[string]float64)
	m.lastPriceUpdate = time.Time{}
}

// GetDiscountConfig returns the current discount configuration
func (m *Manager) GetDiscountConfig() ctypes.DiscountConfig {
	return m.discountConfig
}

// applyEC2Discounts applies all applicable discounts to EC2 pricing
func (m *Manager) applyEC2Discounts(basePrice float64) float64 {
	price := basePrice

	// Apply individual discounts sequentially
	if m.discountConfig.EC2Discount > 0 {
		price *= (1.0 - m.discountConfig.EC2Discount)
	}

	if m.discountConfig.SavingsPlansDiscount > 0 {
		price *= (1.0 - m.discountConfig.SavingsPlansDiscount)
	}

	if m.discountConfig.ReservedInstanceDiscount > 0 {
		price *= (1.0 - m.discountConfig.ReservedInstanceDiscount)
	}

	if m.discountConfig.EducationalDiscount > 0 {
		price *= (1.0 - m.discountConfig.EducationalDiscount)
	}

	if m.discountConfig.StartupDiscount > 0 {
		price *= (1.0 - m.discountConfig.StartupDiscount)
	}

	if m.discountConfig.EnterpriseDiscount > 0 {
		price *= (1.0 - m.discountConfig.EnterpriseDiscount)
	}

	return price
}

// applyEBSDiscounts applies all applicable discounts to EBS pricing
func (m *Manager) applyEBSDiscounts(basePrice float64) float64 {
	price := basePrice

	if m.discountConfig.EBSDiscount > 0 {
		price *= (1.0 - m.discountConfig.EBSDiscount)
	}

	if m.discountConfig.VolumeDiscount > 0 {
		price *= (1.0 - m.discountConfig.VolumeDiscount)
	}

	if m.discountConfig.EnterpriseDiscount > 0 {
		price *= (1.0 - m.discountConfig.EnterpriseDiscount)
	}

	return price
}

// applyEFSDiscounts applies all applicable discounts to EFS pricing
func (m *Manager) applyEFSDiscounts(basePrice float64) float64 {
	price := basePrice

	if m.discountConfig.EFSDiscount > 0 {
		price *= (1.0 - m.discountConfig.EFSDiscount)
	}

	if m.discountConfig.EnterpriseDiscount > 0 {
		price *= (1.0 - m.discountConfig.EnterpriseDiscount)
	}

	return price
}

// findVolumeByName finds an EBS volume by its Name tag
func (m *Manager) findVolumeByName(name string) (string, error) {
	ctx := context.Background()
	result, err := m.ec2.DescribeVolumes(ctx, &ec2.DescribeVolumesInput{
		Filters: []ec2types.Filter{
			{
				Name:   aws.String("tag:Name"),
				Values: []string{name},
			},
			{
				Name:   aws.String("tag:CloudWorkstation"),
				Values: []string{"true"},
			},
		},
	})
	if err != nil {
		return "", fmt.Errorf("failed to describe volumes: %w", err)
	}

	if len(result.Volumes) == 0 {
		return "", fmt.Errorf("volume '%s' not found", name)
	}

	return *result.Volumes[0].VolumeId, nil
}

// EnsureKeyPairExists ensures the SSH key pair exists in AWS, creating it if necessary
func (m *Manager) EnsureKeyPairExists(keyName, publicKeyContent string) error {
	ctx := context.Background()
	// Check if key pair already exists
	_, err := m.ec2.DescribeKeyPairs(ctx, &ec2.DescribeKeyPairsInput{
		KeyNames: []string{keyName},
	})

	if err == nil {
		// Key pair already exists
		return nil
	}

	// Key pair doesn't exist, import it
	_, err = m.ec2.ImportKeyPair(ctx, &ec2.ImportKeyPairInput{
		KeyName:           aws.String(keyName),
		PublicKeyMaterial: []byte(publicKeyContent),
		TagSpecifications: []ec2types.TagSpecification{
			{
				ResourceType: ec2types.ResourceTypeKeyPair,
				Tags: []ec2types.Tag{
					{
						Key:   aws.String("CloudWorkstation"),
						Value: aws.String("true"),
					},
					{
						Key:   aws.String("ManagedBy"),
						Value: aws.String("cws"),
					},
				},
			},
		},
	})
	if err != nil {
		return fmt.Errorf("failed to import key pair: %w", err)
	}

	return nil
}

// DeleteKeyPair deletes an SSH key pair from AWS
func (m *Manager) DeleteKeyPair(keyName string) error {
	ctx := context.Background()
	_, err := m.ec2.DeleteKeyPair(ctx, &ec2.DeleteKeyPairInput{
		KeyName: aws.String(keyName),
	})
	if err != nil {
		return fmt.Errorf("failed to delete key pair: %w", err)
	}

	return nil
}

// ListCloudWorkstationKeyPairs lists all SSH key pairs managed by CloudWorkstation
func (m *Manager) ListCloudWorkstationKeyPairs() ([]string, error) {
	ctx := context.Background()
	result, err := m.ec2.DescribeKeyPairs(ctx, &ec2.DescribeKeyPairsInput{
		Filters: []ec2types.Filter{
			{
				Name:   aws.String("tag:CloudWorkstation"),
				Values: []string{"true"},
			},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to describe key pairs: %w", err)
	}

	var keyNames []string
	for _, keyPair := range result.KeyPairs {
		if keyPair.KeyName != nil {
			keyNames = append(keyNames, *keyPair.KeyName)
		}
	}

	return keyNames, nil
}

// getSSHKeyPathFromKeyName maps an AWS key pair name to local SSH key path
func (m *Manager) getSSHKeyPathFromKeyName(keyName string) (string, error) {
	// CloudWorkstation key naming pattern: cws-<profile>-key
	if strings.HasPrefix(keyName, "cws-") && strings.HasSuffix(keyName, "-key") {
		// Extract safe name from key name (it's already safe for filesystem)
		safeName := strings.TrimPrefix(keyName, "cws-")
		safeName = strings.TrimSuffix(safeName, "-key")

		// Get home directory
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("failed to get home directory: %w", err)
		}

		// Construct key path using the same naming
		keyPath := filepath.Join(homeDir, ".ssh", fmt.Sprintf("cws-%s-key", safeName))

		// Check if key exists
		if _, err := os.Stat(keyPath); os.IsNotExist(err) {
			return "", fmt.Errorf("SSH key not found at %s", keyPath)
		}

		return keyPath, nil
	}

	// For non-CloudWorkstation keys, try to find default SSH keys
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}

	// Check common SSH key locations
	commonKeys := []string{"id_ed25519", "id_rsa", "id_ecdsa"}
	for _, keyType := range commonKeys {
		keyPath := filepath.Join(homeDir, ".ssh", keyType)
		if _, err := os.Stat(keyPath); err == nil {
			return keyPath, nil
		}
	}

	return "", fmt.Errorf("no SSH key found for key name: %s", keyName)
}

// ===== NETWORKING FUNCTIONS =====

// DiscoverDefaultVPC finds the default VPC in the current region
func (m *Manager) DiscoverDefaultVPC() (string, error) {
	ctx := context.Background()
	result, err := m.ec2.DescribeVpcs(ctx, &ec2.DescribeVpcsInput{
		Filters: []ec2types.Filter{
			{
				Name:   aws.String("is-default"),
				Values: []string{"true"},
			},
		},
	})
	if err != nil {
		return "", fmt.Errorf("failed to describe VPCs: %w", err)
	}

	if len(result.Vpcs) == 0 {
		return "", fmt.Errorf("no default VPC found in region %s - please create one or specify --vpc", m.region)
	}

	return *result.Vpcs[0].VpcId, nil
}

// DiscoverPublicSubnet finds a public subnet in the specified VPC
func (m *Manager) DiscoverPublicSubnet(vpcID string) (string, error) {
	ctx := context.Background()
	// Get all subnets in the VPC
	result, err := m.ec2.DescribeSubnets(ctx, &ec2.DescribeSubnetsInput{
		Filters: []ec2types.Filter{
			{
				Name:   aws.String("vpc-id"),
				Values: []string{vpcID},
			},
		},
	})
	if err != nil {
		return "", fmt.Errorf("failed to describe subnets in VPC %s: %w", vpcID, err)
	}

	if len(result.Subnets) == 0 {
		return "", fmt.Errorf("no subnets found in VPC %s", vpcID)
	}

	// Find a public subnet by checking route tables
	for _, subnet := range result.Subnets {
		isPublic, err := m.isSubnetPublic(*subnet.SubnetId)
		if err != nil {
			continue // Skip this subnet on error
		}
		if isPublic {
			return *subnet.SubnetId, nil
		}
	}

	// If no clearly public subnet found, use the first available subnet
	// (this handles cases where route table detection fails)
	return *result.Subnets[0].SubnetId, nil
}

// isSubnetPublic checks if a subnet is public by examining its route table
func (m *Manager) isSubnetPublic(subnetID string) (bool, error) {
	ctx := context.Background()
	// Get route tables for this subnet
	result, err := m.ec2.DescribeRouteTables(ctx, &ec2.DescribeRouteTablesInput{
		Filters: []ec2types.Filter{
			{
				Name:   aws.String("association.subnet-id"),
				Values: []string{subnetID},
			},
		},
	})
	if err != nil {
		return false, err
	}

	// Check each route table for internet gateway routes
	for _, routeTable := range result.RouteTables {
		for _, route := range routeTable.Routes {
			// Look for route to 0.0.0.0/0 via internet gateway
			if route.DestinationCidrBlock != nil && *route.DestinationCidrBlock == "0.0.0.0/0" {
				if route.GatewayId != nil && strings.HasPrefix(*route.GatewayId, "igw-") {
					return true, nil
				}
			}
		}
	}

	return false, nil
}

// GetOrCreateCloudWorkstationSecurityGroup creates or finds the CloudWorkstation security group
func (m *Manager) GetOrCreateCloudWorkstationSecurityGroup(vpcID string) (string, error) {
	securityGroupName := "cloudworkstation-access"

	ctx := context.Background()
	// Try to find existing security group
	result, err := m.ec2.DescribeSecurityGroups(ctx, &ec2.DescribeSecurityGroupsInput{
		Filters: []ec2types.Filter{
			{
				Name:   aws.String("group-name"),
				Values: []string{securityGroupName},
			},
			{
				Name:   aws.String("vpc-id"),
				Values: []string{vpcID},
			},
		},
	})
	if err != nil {
		return "", fmt.Errorf("failed to describe security groups: %w", err)
	}

	// Return existing security group if found
	if len(result.SecurityGroups) > 0 {
		return *result.SecurityGroups[0].GroupId, nil
	}

	// Create new security group
	createResult, err := m.ec2.CreateSecurityGroup(ctx, &ec2.CreateSecurityGroupInput{
		GroupName:   aws.String(securityGroupName),
		Description: aws.String("CloudWorkstation SSH and web access"),
		VpcId:       aws.String(vpcID),
		TagSpecifications: []ec2types.TagSpecification{
			{
				ResourceType: ec2types.ResourceTypeSecurityGroup,
				Tags: []ec2types.Tag{
					{Key: aws.String("Name"), Value: aws.String(securityGroupName)},
					{Key: aws.String("CloudWorkstation"), Value: aws.String("true")},
					{Key: aws.String("Purpose"), Value: aws.String("Research workstation access")},
				},
			},
		},
	})
	if err != nil {
		return "", fmt.Errorf("failed to create security group: %w", err)
	}

	securityGroupID := *createResult.GroupId

	// Add SSH rule (port 22)
	_, err = m.ec2.AuthorizeSecurityGroupIngress(ctx, &ec2.AuthorizeSecurityGroupIngressInput{
		GroupId: aws.String(securityGroupID),
		IpPermissions: []ec2types.IpPermission{
			{
				IpProtocol: aws.String("tcp"),
				FromPort:   aws.Int32(22),
				ToPort:     aws.Int32(22),
				IpRanges: []ec2types.IpRange{
					{
						CidrIp:      aws.String("0.0.0.0/0"),
						Description: aws.String("SSH access"),
					},
				},
			},
		},
	})
	if err != nil {
		return "", fmt.Errorf("failed to add SSH rule to security group: %w", err)
	}

	// Add HTTP rule (port 80) for web interfaces
	_, err = m.ec2.AuthorizeSecurityGroupIngress(ctx, &ec2.AuthorizeSecurityGroupIngressInput{
		GroupId: aws.String(securityGroupID),
		IpPermissions: []ec2types.IpPermission{
			{
				IpProtocol: aws.String("tcp"),
				FromPort:   aws.Int32(80),
				ToPort:     aws.Int32(80),
				IpRanges: []ec2types.IpRange{
					{
						CidrIp:      aws.String("0.0.0.0/0"),
						Description: aws.String("HTTP access for web interfaces"),
					},
				},
			},
		},
	})
	if err != nil {
		return "", fmt.Errorf("failed to add HTTP rule to security group: %w", err)
	}

	// Add HTTPS rule (port 443) for secure web interfaces
	_, err = m.ec2.AuthorizeSecurityGroupIngress(ctx, &ec2.AuthorizeSecurityGroupIngressInput{
		GroupId: aws.String(securityGroupID),
		IpPermissions: []ec2types.IpPermission{
			{
				IpProtocol: aws.String("tcp"),
				FromPort:   aws.Int32(443),
				ToPort:     aws.Int32(443),
				IpRanges: []ec2types.IpRange{
					{
						CidrIp:      aws.String("0.0.0.0/0"),
						Description: aws.String("HTTPS access for secure web interfaces"),
					},
				},
			},
		},
	})
	if err != nil {
		return "", fmt.Errorf("failed to add HTTPS rule to security group: %w", err)
	}

	// Add Jupyter rule (port 8888) for notebook interfaces
	_, err = m.ec2.AuthorizeSecurityGroupIngress(ctx, &ec2.AuthorizeSecurityGroupIngressInput{
		GroupId: aws.String(securityGroupID),
		IpPermissions: []ec2types.IpPermission{
			{
				IpProtocol: aws.String("tcp"),
				FromPort:   aws.Int32(8888),
				ToPort:     aws.Int32(8888),
				IpRanges: []ec2types.IpRange{
					{
						CidrIp:      aws.String("0.0.0.0/0"),
						Description: aws.String("Jupyter notebook access"),
					},
				},
			},
		},
	})
	if err != nil {
		return "", fmt.Errorf("failed to add Jupyter rule to security group: %w", err)
	}

	// Add RStudio rule (port 8787) for R interfaces
	_, err = m.ec2.AuthorizeSecurityGroupIngress(ctx, &ec2.AuthorizeSecurityGroupIngressInput{
		GroupId: aws.String(securityGroupID),
		IpPermissions: []ec2types.IpPermission{
			{
				IpProtocol: aws.String("tcp"),
				FromPort:   aws.Int32(8787),
				ToPort:     aws.Int32(8787),
				IpRanges: []ec2types.IpRange{
					{
						CidrIp:      aws.String("0.0.0.0/0"),
						Description: aws.String("RStudio server access"),
					},
				},
			},
		},
	})
	if err != nil {
		return "", fmt.Errorf("failed to add RStudio rule to security group: %w", err)
	}

	// Add ICMP rule for ping
	_, err = m.ec2.AuthorizeSecurityGroupIngress(ctx, &ec2.AuthorizeSecurityGroupIngressInput{
		GroupId: aws.String(securityGroupID),
		IpPermissions: []ec2types.IpPermission{
			{
				IpProtocol: aws.String("icmp"),
				FromPort:   aws.Int32(-1),
				ToPort:     aws.Int32(-1),
				IpRanges: []ec2types.IpRange{
					{
						CidrIp:      aws.String("0.0.0.0/0"),
						Description: aws.String("ICMP ping access"),
					},
				},
			},
		},
	})
	if err != nil {
		return "", fmt.Errorf("failed to add ICMP rule to security group: %w", err)
	}

	return securityGroupID, nil
}

// supportsHibernation checks if an instance type supports hibernation
func (m *Manager) supportsHibernation(instanceType string) bool {
	// AWS hibernation support is based on instance families and generations
	// Reference: https://docs.aws.amazon.com/AWSEC2/latest/UserGuide/hibernating-instances.html

	supportedFamilies := map[string]bool{
		// General Purpose
		"t2": true, "t3": true, "t3a": true,
		"m3": true, "m4": true, "m5": true, "m5a": true, "m5n": true, "m5zn": true,
		"m6i": true, "m6a": true, "m6g": true, "m7i": true, "m7a": true, "m7g": true,

		// Compute Optimized
		"c3": true, "c4": true, "c5": true, "c5n": true,
		"c6i": true, "c6a": true, "c6g": true, "c7i": true, "c7a": true, "c7g": true,

		// Memory Optimized
		"r3": true, "r4": true, "r5": true, "r5a": true, "r5n": true,
		"r6i": true, "r6a": true, "r6g": true, "r7i": true, "r7a": true, "r7g": true,
		"x1": true, "x1e": true,

		// Accelerated Computing (GPU)
		"g4dn": true, "g4ad": true, "g5": true, "g5g": true,
	}

	// Extract instance family from instance type (e.g., "c6g.large" -> "c6g", "t3.micro" -> "t3")
	dotIndex := strings.Index(instanceType, ".")
	if dotIndex == -1 {
		// No dot found, use entire string
		return supportedFamilies[instanceType]
	}

	family := instanceType[:dotIndex]
	return supportedFamilies[family]
}
