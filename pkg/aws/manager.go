package aws

import (
	"context"
	"encoding/base64"
	"fmt"
	"log"
	"net"
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
	"github.com/aws/aws-sdk-go-v2/service/iam"
	iamTypes "github.com/aws/aws-sdk-go-v2/service/iam/types"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
	"github.com/aws/aws-sdk-go-v2/service/sts"

	"github.com/scttfrdmn/cloudworkstation/pkg/idle"
	"github.com/scttfrdmn/cloudworkstation/pkg/security"
	"github.com/scttfrdmn/cloudworkstation/pkg/state"
	"github.com/scttfrdmn/cloudworkstation/pkg/templates"
	ctypes "github.com/scttfrdmn/cloudworkstation/pkg/types"
)

// AWS instance state constants
const (
	instanceStateRunning = "running"
	instanceStateStopped = "stopped"
	volumeTypeIO2        = "io2"
)

// Manager handles all AWS operations
type Manager struct {
	cfg            aws.Config
	ec2            EC2ClientInterface
	efs            EFSClientInterface
	iam            *iam.Client
	ssm            SSMClientInterface
	sts            STSClientInterface
	region         string
	templates      map[string]ctypes.Template
	pricingClient  *PricingClient
	discountConfig ctypes.DiscountConfig
	stateManager   StateManagerInterface
	idleScheduler  *idle.Scheduler
	policyManager  *idle.PolicyManager

	// Universal AMI System components (Phase 5.1)
	amiResolver *UniversalAMIResolver

	// Universal Version System components (v0.5.4)
	amiDiscovery *AMIDiscovery

	// Architecture cache for instance types
	architectureCache map[string]string // instance_type -> architecture
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
	iamClient := iam.NewFromConfig(cfg)
	ssmClient := ssm.NewFromConfig(cfg)
	stsClient := sts.NewFromConfig(cfg)

	// Use specified region or fallback to config region
	region := opt.Region
	if region == "" {
		region = cfg.Region
	}

	// Initialize Universal AMI System (Phase 5.1)
	amiResolver := NewUniversalAMIResolver(ec2Client)

	// Initialize Universal Version System (v0.5.4)
	amiDiscovery := NewAMIDiscovery(ssmClient)

	// Initialize AWS Pricing API client with caching
	pricingClient := NewPricingClient(cfg)

	// Create manager first (needed for adapter)
	manager := &Manager{
		cfg:               cfg,
		ec2:               ec2Client,
		efs:               efsClient,
		iam:               iamClient,
		ssm:               ssmClient,
		sts:               stsClient,
		region:            region,
		templates:         getTemplates(),
		pricingClient:     pricingClient,
		discountConfig:    ctypes.DiscountConfig{}, // No discounts by default
		stateManager:      stateManager,
		amiResolver:       amiResolver,
		amiDiscovery:      amiDiscovery,
		architectureCache: make(map[string]string), // Initialize architecture cache
	}

	// Initialize hibernation components with adapter to break circular dependency
	awsAdapter := idle.NewAWSManagerAdapter(
		manager.HibernateInstance,
		manager.ResumeInstance,
		manager.StopInstance,
		manager.StartInstance,
		func() ([]string, error) {
			// Get instance names from ListInstances
			instances, err := manager.ListInstances()
			if err != nil {
				return nil, err
			}
			names := make([]string, len(instances))
			for i, inst := range instances {
				names[i] = inst.Name
			}
			return names, nil
		},
		func(name string) (string, error) {
			// Get instance ID from name via state manager
			instances, err := manager.ListInstances()
			if err != nil {
				return "", err
			}
			for _, inst := range instances {
				if inst.Name == name {
					return inst.ID, nil
				}
			}
			return "", fmt.Errorf("instance not found: %s", name)
		},
	)

	// Create CloudWatch metrics collector for idle detection
	metricsCollector := idle.NewMetricsCollector(cfg)

	// Create scheduler with metrics collector
	idleScheduler := idle.NewScheduler(awsAdapter, metricsCollector)
	policyManager := idle.NewPolicyManager()
	policyManager.SetScheduler(idleScheduler)

	// Assign to manager
	manager.idleScheduler = idleScheduler
	manager.policyManager = policyManager

	// Start the idle scheduler
	idleScheduler.Start()

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
	// ARCHITECTURE FIX: Determine instance type first, then query its architecture
	// This fixes the critical bug where local machine architecture was used to select AMIs

	// Step 1: Get template to determine what instance type will be used
	// We need to know the instance type before we can determine architecture
	rawTemplate, err := templates.GetTemplateInfo(req.Template)
	if err != nil {
		return nil, fmt.Errorf("failed to get template info: %w", err)
	}

	// Step 2: Determine which instance type will be used
	var instanceType string
	if req.Size != "" {
		// User specified size, map it to instance type
		instanceType = m.getInstanceTypeForSize(req.Size)
	} else if rawTemplate.InstanceDefaults.Type != "" {
		// Use template's default instance type
		instanceType = rawTemplate.InstanceDefaults.Type
	} else {
		// Ultimate fallback
		instanceType = "t3.micro"
	}

	// Step 3: Query AWS for this instance type's architecture
	arch, err := m.getInstanceTypeArchitecture(instanceType)
	if err != nil {
		// This shouldn't fail due to fallbacks in getInstanceTypeArchitecture
		log.Printf("Warning: Failed to get architecture for instance type %s: %v", instanceType, err)
		arch = "x86_64" // Safe fallback
	}

	log.Printf("Instance type %s supports architecture: %s", instanceType, arch)

	// Step 4: Now launch with the correct architecture
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
func (n *NetworkingResolver) ResolveNetworking(req ctypes.LaunchRequest, instanceType string) (string, string, string, error) {
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
		// Discover subnet that supports the instance type
		discoveredSubnet, err := n.manager.DiscoverPublicSubnetForInstanceType(vpcID, instanceType)
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
func (b *InstanceConfigBuilder) BuildRunInstancesInput(req ctypes.LaunchRequest, ami, instanceType, userDataEncoded, subnetID, securityGroupID, primaryUsername string) (*ec2.RunInstancesInput, error) {
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
		// IAM Instance Profile is optional - only add if it exists
		// This makes onboarding painless for new users
		TagSpecifications: []ec2types.TagSpecification{
			{
				ResourceType: ec2types.ResourceTypeInstance,
				Tags: []ec2types.Tag{
					{Key: aws.String("Name"), Value: &req.Name},
					{Key: aws.String("CloudWorkstation"), Value: aws.String("true")},
					{Key: aws.String("LaunchedBy"), Value: aws.String("CloudWorkstation")},
					{Key: aws.String("Template"), Value: &req.Template},
					{Key: aws.String("PackageManager"), Value: &req.PackageManager},
					{Key: aws.String("PrimaryUser"), Value: aws.String(primaryUsername)},
				},
			},
		},
	}

	// Add SSH key pair if provided
	if req.SSHKeyName != "" {
		runInput.KeyName = aws.String(req.SSHKeyName)
	}

	// Optionally add IAM instance profile if it exists
	// This enables SSM access for advanced features while not blocking new users
	if b.manager.checkIAMInstanceProfileExists("CloudWorkstation-Instance-Profile") {
		runInput.IamInstanceProfile = &ec2types.IamInstanceProfileSpecification{
			Name: aws.String("CloudWorkstation-Instance-Profile"),
		}
		log.Printf("Using IAM instance profile for SSM access")
	} else {
		log.Printf("IAM instance profile not found - launching without it (SSM features will be unavailable)")
	}

	return runInput, nil
}

// LaunchOptionsProcessor processes hibernation and spot options (Strategy Pattern - SOLID)
type LaunchOptionsProcessor struct {
	manager *Manager
}

// ProcessOptions validates and applies hibernation/spot options
func (p *LaunchOptionsProcessor) ProcessOptions(req ctypes.LaunchRequest, runInput *ec2.RunInstancesInput, ami, instanceType string, rootVolumeGB int) error {
	fmt.Printf("DEBUG [ProcessOptions:379]: Received rootVolumeGB parameter: %d GB\n", rootVolumeGB)

	// Support IdlePolicy flag
	enableIdlePolicy := req.IdlePolicy

	// Validate idle policy and spot combination
	if enableIdlePolicy && req.Spot {
		return fmt.Errorf("idle policy (hibernation) and spot instances cannot be used together\n\n💡 AWS Limitation:\n  • Spot instances can be interrupted at any time\n  • Idle policies preserve instance state for later resume\n  • These features are incompatible\n\nChoose one:\n  • Use --idle-policy for cost-effective session preservation\n  • Use --spot for discounted compute pricing\n  • Use both flags separately on different instances")
	}

	// Determine root device name (AWS uses different names for different AMIs)
	rootDevice := "/dev/sda1"
	if strings.Contains(strings.ToLower(ami), "amazon") || strings.Contains(strings.ToLower(ami), "amzn") {
		rootDevice = "/dev/xvda"
	}

	// Always set root volume size from template (default 20GB if not specified)
	fmt.Printf("DEBUG [ProcessOptions:395]: Setting BlockDeviceMapping with %d GB root volume on device %s\n", rootVolumeGB, rootDevice)
	runInput.BlockDeviceMappings = []ec2types.BlockDeviceMapping{
		{
			DeviceName: aws.String(rootDevice),
			Ebs: &ec2types.EbsBlockDevice{
				VolumeType:          ec2types.VolumeTypeGp3,
				VolumeSize:          aws.Int32(int32(rootVolumeGB)),
				Encrypted:           aws.Bool(enableIdlePolicy), // Only encrypt if hibernation enabled
				DeleteOnTermination: aws.Bool(true),
			},
		},
	}

	// Add idle policy support if requested
	if enableIdlePolicy {
		if !p.manager.supportsHibernation(instanceType) {
			return fmt.Errorf("instance type %s does not support idle policy (hibernation)\n\n💡 Idle policy is supported on:\n  • General Purpose: T2, T3, T3a, M3-M7 families (including M6i, M6a, M6g, M7i, M7a, M7g)\n  • Compute Optimized: C3-C7 families (including C6i, C6a, C6g, C7i, C7a, C7g)\n  • Memory Optimized: R3-R7 families (including R6i, R6a, R6g, R7i, R7a, R7g), X1, X1e\n  • Accelerated Computing: G4dn, G4ad, G5, G5g\n\nTip: Remove --idle-policy flag or choose a different instance size", instanceType)
		}

		runInput.HibernationOptions = &ec2types.HibernationOptionsRequest{
			Configured: aws.Bool(true),
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
	region  string
}

// Helper functions for service extraction
func getServiceString(m map[string]interface{}, key string) string {
	if val, ok := m[key].(string); ok {
		return val
	}
	return ""
}

func getServiceInt(m map[string]interface{}, key string) int {
	if val, ok := m[key].(int); ok {
		return val
	}
	if val, ok := m[key].(float64); ok {
		return int(val)
	}
	return 0
}

// LaunchInstance executes EC2 instance launch and returns result
// extractServicesFromTemplate extracts service definitions from template ports
func (l *InstanceLauncher) extractServicesFromTemplate(template *ctypes.RuntimeTemplate) []ctypes.Service {
	var services []ctypes.Service

	// DEBUG LOGGING
	log.Printf("[DEBUG] LaunchInstance: template=%v", template != nil)
	if template != nil {
		log.Printf("[DEBUG] LaunchInstance: template.Ports=%v", template.Ports)
	}

	if template == nil || len(template.Ports) == 0 {
		log.Printf("[DEBUG] LaunchInstance: No ports to extract (template nil or ports empty)")
		return services
	}

	log.Printf("[DEBUG] LaunchInstance: Extracting services from %d ports", len(template.Ports))
	for _, port := range template.Ports {
		service := l.createServiceForPort(port)
		services = append(services, service)
		log.Printf("[DEBUG] LaunchInstance: Added service %s (port %d)", service.Name, service.Port)
	}

	return services
}

// createServiceForPort creates a service definition for a given port
func (l *InstanceLauncher) createServiceForPort(port int) ctypes.Service {
	service := ctypes.Service{
		Port:   port,
		Type:   "web",
		Status: "unknown",
	}

	// Infer service name and description from port
	switch port {
	case 8888:
		service.Name = "jupyter"
		service.Description = "Jupyter Lab"
	case 8787:
		service.Name = "rstudio-server"
		service.Description = "RStudio Server"
	case 3838:
		service.Name = "shiny-server"
		service.Description = "Shiny Server"
	case 8080:
		service.Name = "web"
		service.Description = "Web Application"
	default:
		service.Name = fmt.Sprintf("port-%d", port)
		service.Description = fmt.Sprintf("Service on port %d", port)
	}

	return service
}

// createDryRunInstance creates a dry-run instance response
func (l *InstanceLauncher) createDryRunInstance(req ctypes.LaunchRequest, hourlyRate float64, services []ctypes.Service, primaryUsername string) *ctypes.Instance {
	return &ctypes.Instance{
		Name:          req.Name,
		Template:      req.Template,
		Region:        l.region,
		State:         "dry-run",
		HourlyRate:    hourlyRate,
		CurrentSpend:  0.0,
		EffectiveRate: 0.0,
		Services:      services,
		Username:      primaryUsername,
	}
}

// executeInstanceLaunch performs the actual EC2 instance launch
func (l *InstanceLauncher) executeInstanceLaunch(ctx context.Context, runInput *ec2.RunInstancesInput) (*ec2types.Instance, error) {
	result, err := l.manager.ec2.RunInstances(ctx, runInput)
	if err != nil {
		return nil, fmt.Errorf("failed to launch instance: %w", err)
	}

	if len(result.Instances) == 0 {
		return nil, fmt.Errorf("no instances returned from launch")
	}

	return &result.Instances[0], nil
}

// buildInstanceFromEC2 builds CloudWorkstation instance from EC2 instance
func (l *InstanceLauncher) buildInstanceFromEC2(instance *ec2types.Instance, req ctypes.LaunchRequest, hourlyRate float64, services []ctypes.Service, primaryUsername string) *ctypes.Instance {
	instanceType := string(instance.InstanceType)
	launchTime := time.Now()

	// Get availability zone from placement
	availabilityZone := ""
	if instance.Placement != nil && instance.Placement.AvailabilityZone != nil {
		availabilityZone = *instance.Placement.AvailabilityZone
	}

	// Calculate storage costs that persist even when stopped/hibernated
	storageCostPerHour := calculateStorageCosts(req.Volumes, req.EBSVolumes)

	log.Printf("[DEBUG] Creating instance with username: %s", primaryUsername)

	// Record initial state transition for cost tracking
	initialState := string(instance.State.Name)
	stateHistory := []ctypes.StateTransition{
		{
			FromState: "",           // Empty for initial launch
			ToState:   initialState, // Usually "pending" at launch
			Timestamp: launchTime,
			Reason:    "instance_launch",
			Initiator: "user",
		},
	}

	cwsInstance := &ctypes.Instance{
		ID:                 *instance.InstanceId,
		Name:               req.Name,
		Template:           req.Template,
		Region:             l.region, // Store region for proper instance management
		AvailabilityZone:   availabilityZone,
		State:              initialState,
		InstanceType:       instanceType,
		LaunchTime:         launchTime,
		HourlyRate:         hourlyRate,
		CurrentSpend:       storageCostPerHour,              // Only storage costs at launch
		EffectiveRate:      hourlyRate + storageCostPerHour, // Will include storage
		AttachedVolumes:    req.Volumes,
		AttachedEBSVolumes: req.EBSVolumes,
		Services:           services,        // Web services from template
		Username:           primaryUsername, // Primary user from template
		StateHistory:       stateHistory,    // Initialize state history with launch event
	}

	log.Printf("[DEBUG] Instance created with username: %s", cwsInstance.Username)

	// DEBUG LOGGING
	log.Printf("[DEBUG] LaunchInstance: Created instance with %d services", len(cwsInstance.Services))
	for _, svc := range cwsInstance.Services {
		log.Printf("[DEBUG] LaunchInstance: Instance service: %s (port %d)", svc.Name, svc.Port)
	}

	return cwsInstance
}

// waitForInstanceReady waits for instance to be ready with user feedback
func (l *InstanceLauncher) waitForInstanceReady(instanceID string) {
	log.Printf("⏳ Waiting for instance to be ready for connections...")
	if err := l.manager.waitForInstanceReadyWithProgress(instanceID, l.region, func(stage string, progress float64, description string) {
		// Provide user feedback during readiness waiting
		if progress == 0.0 {
			log.Printf("  → %s", description)
		} else if progress == 1.0 {
			log.Printf("  ✓ %s", description)
		}
	}); err != nil {
		// Don't fail the launch if waiting times out - instance is still created
		log.Printf("⚠️  Warning: Instance launched but readiness check timed out: %v", err)
		log.Printf("    The instance may need a few more moments before you can connect")
	} else {
		log.Printf("✅ Instance %s is ready for SSH connections", instanceID)
	}
}

// LaunchInstance orchestrates instance launch with extracted helper methods
func (l *InstanceLauncher) LaunchInstance(req ctypes.LaunchRequest, runInput *ec2.RunInstancesInput, hourlyRate float64, template *ctypes.RuntimeTemplate, primaryUsername string) (*ctypes.Instance, error) {
	// Extract services from template
	services := l.extractServicesFromTemplate(template)

	// Handle dry run
	if req.DryRun {
		return l.createDryRunInstance(req, hourlyRate, services, primaryUsername), nil
	}

	// Launch instance
	ctx := context.Background()
	instance, err := l.executeInstanceLaunch(ctx, runInput)
	if err != nil {
		return nil, err
	}

	// Build CloudWorkstation instance from EC2 instance
	cwsInstance := l.buildInstanceFromEC2(instance, req, hourlyRate, services, primaryUsername)

	// Wait for instance to be ready for use
	l.waitForInstanceReady(cwsInstance.ID)

	return cwsInstance, nil
}

// LaunchOrchestrator coordinates instance launch using SOLID principles (Strategy Pattern - SOLID)
type LaunchOrchestrator struct {
	configExtractor    *TemplateConfigExtractor
	userDataProcessor  *UserDataProcessor
	networkingResolver *NetworkingResolver
	configBuilder      *InstanceConfigBuilder
	optionsProcessor   *LaunchOptionsProcessor
	instanceLauncher   *InstanceLauncher
}

// NewLaunchOrchestrator creates launch orchestrator
func NewLaunchOrchestrator(manager *Manager, region string) *LaunchOrchestrator {
	return &LaunchOrchestrator{
		configExtractor:    &TemplateConfigExtractor{region: region},
		userDataProcessor:  &UserDataProcessor{manager: manager, region: region},
		networkingResolver: &NetworkingResolver{manager: manager},
		configBuilder:      &InstanceConfigBuilder{manager: manager},
		optionsProcessor:   &LaunchOptionsProcessor{manager: manager},
		instanceLauncher:   &InstanceLauncher{manager: manager, region: region},
	}
}

// ExecuteLaunch performs complete instance launch using SOLID strategy pattern
func (o *LaunchOrchestrator) ExecuteLaunch(req ctypes.LaunchRequest, template *ctypes.RuntimeTemplate, arch, primaryUsername string) (*ctypes.Instance, error) {
	// Extract template configuration
	ami, instanceType, dailyCost, err := o.configExtractor.ExtractConfig(template, arch)
	if err != nil {
		return nil, err
	}

	// Process user data
	userDataEncoded := o.userDataProcessor.ProcessUserData(template, req)

	// Resolve networking (pass instance type for AZ compatibility check)
	_, subnetID, securityGroupID, err := o.networkingResolver.ResolveNetworking(req, instanceType)
	if err != nil {
		return nil, err
	}

	// Build run configuration
	runInput, err := o.configBuilder.BuildRunInstancesInput(req, ami, instanceType, userDataEncoded, subnetID, securityGroupID, primaryUsername)
	if err != nil {
		return nil, err
	}

	// Process launch options (pass root volume size from template)
	rootVolumeGB := template.RootVolumeGB
	if rootVolumeGB == 0 {
		rootVolumeGB = 20 // Default if not specified
	}
	if err := o.optionsProcessor.ProcessOptions(req, runInput, ami, instanceType, rootVolumeGB); err != nil {
		return nil, err
	}

	// Execute launch
	return o.instanceLauncher.LaunchInstance(req, runInput, dailyCost, template, primaryUsername)
}

// launchWithUnifiedTemplateSystem launches instance using unified template system with SOLID orchestration (SOLID: Single Responsibility)
func (m *Manager) launchWithUnifiedTemplateSystem(req ctypes.LaunchRequest, arch string) (*ctypes.Instance, error) {
	// Get template using unified template system
	packageManager := req.PackageManager
	if packageManager == "" {
		packageManager = ""
	}

	var template *ctypes.RuntimeTemplate
	var err error

	// Use parameter-aware template processing if parameters are provided
	if len(req.Parameters) > 0 {
		template, err = templates.GetTemplateWithParameters(req.Template, m.region, arch, packageManager, req.Size, req.Parameters)
	} else {
		template, err = templates.GetTemplateWithPackageManager(req.Template, m.region, arch, packageManager, req.Size)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to get template: %w", err)
	}

	// Get raw template for validation and username extraction
	rawTemplate, _ := templates.GetTemplateInfo(req.Template)

	// Extract primary username from template (first user in list)
	primaryUsername := "ubuntu" // Default fallback
	if rawTemplate != nil && len(rawTemplate.Users) > 0 {
		primaryUsername = rawTemplate.Users[0].Name
		log.Printf("[DEBUG] Extracted username from template: %s (from %d users)", primaryUsername, len(rawTemplate.Users))
	} else {
		log.Printf("[DEBUG] No users in template, using default: ubuntu")
	}

	// Validate template before launch (if not dry-run)
	if !req.DryRun {
		if rawTemplate != nil {
			registry := templates.NewTemplateRegistry(templates.DefaultTemplateDirs())
			registry.Templates[req.Template] = rawTemplate // Add to registry for validation

			validator := templates.NewComprehensiveValidator(registry)
			report := validator.ValidateTemplate(rawTemplate)

			if !report.Valid {
				// Build error message with details
				var errors []string
				for _, result := range report.Results {
					if result.Level == templates.ValidationError {
						errors = append(errors, fmt.Sprintf("%s: %s", result.Field, result.Message))
					}
				}
				return nil, fmt.Errorf("template validation failed: %s", strings.Join(errors, "; "))
			}
		}
	}

	// Create orchestrator and execute launch
	orchestrator := NewLaunchOrchestrator(m, m.region)
	return orchestrator.ExecuteLaunch(req, template, arch, primaryUsername)
}

// DeleteInstance terminates an EC2 instance
func (m *Manager) DeleteInstance(name string) error {
	// Get instance region
	region, err := m.getInstanceRegion(name)
	if err != nil {
		return fmt.Errorf("failed to get instance region: %w", err)
	}

	// Find instance by name tag
	instanceID, err := m.findInstanceByName(name)
	if err != nil {
		return fmt.Errorf("failed to find instance: %w", err)
	}

	// Get regional EC2 client
	regionalClient := m.getRegionalEC2Client(region)

	// Terminate the instance
	ctx := context.Background()
	_, err = regionalClient.TerminateInstances(ctx, &ec2.TerminateInstancesInput{
		InstanceIds: []string{instanceID},
	})
	if err != nil {
		return fmt.Errorf("failed to terminate instance: %w", err)
	}

	return nil
}

// StartInstance starts a stopped EC2 instance
func (m *Manager) StartInstance(name string) error {
	// Get instance region
	region, err := m.getInstanceRegion(name)
	if err != nil {
		return fmt.Errorf("failed to get instance region: %w", err)
	}

	// Find instance by name tag
	instanceID, err := m.findInstanceByName(name)
	if err != nil {
		return fmt.Errorf("failed to find instance: %w", err)
	}

	// Get regional EC2 client
	regionalClient := m.getRegionalEC2Client(region)

	// Start the instance
	ctx := context.Background()
	_, err = regionalClient.StartInstances(ctx, &ec2.StartInstancesInput{
		InstanceIds: []string{instanceID},
	})
	if err != nil {
		return fmt.Errorf("failed to start instance: %w", err)
	}

	return nil
}

// StopInstance stops a running EC2 instance
func (m *Manager) StopInstance(name string) error {
	// Get instance region
	region, err := m.getInstanceRegion(name)
	if err != nil {
		return fmt.Errorf("failed to get instance region: %w", err)
	}

	// Find instance by name tag
	instanceID, err := m.findInstanceByName(name)
	if err != nil {
		return fmt.Errorf("failed to find instance: %w", err)
	}

	// Get regional EC2 client
	regionalClient := m.getRegionalEC2Client(region)

	ctx := context.Background()
	// Stop the instance
	_, err = regionalClient.StopInstances(ctx, &ec2.StopInstancesInput{
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
	// Get instance region
	region, err := m.getInstanceRegion(name)
	if err != nil {
		return fmt.Errorf("failed to get instance region: %w", err)
	}

	// Find instance by name tag
	instanceID, err := m.findInstanceByName(name)
	if err != nil {
		return fmt.Errorf("failed to find instance: %w", err)
	}

	// Get regional EC2 client
	regionalClient := m.getRegionalEC2Client(region)

	ctx := context.Background()
	// Check if instance supports hibernation
	result, err := regionalClient.DescribeInstances(ctx, &ec2.DescribeInstancesInput{
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
	if instanceState != instanceStateRunning {
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
	_, err = regionalClient.StopInstances(ctx, &ec2.StopInstancesInput{
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

// GetInstanceHibernationStatus returns hibernation support, current state, and whether it might be hibernated
func (m *Manager) GetInstanceHibernationStatus(name string) (bool, string, bool, error) {
	// Get instance region
	region, err := m.getInstanceRegion(name)
	if err != nil {
		return false, "", false, fmt.Errorf("failed to get instance region: %w", err)
	}

	// Find instance by name tag
	instanceID, err := m.findInstanceByName(name)
	if err != nil {
		return false, "", false, fmt.Errorf("failed to find instance: %w", err)
	}

	// Get regional EC2 client
	regionalClient := m.getRegionalEC2Client(region)

	ctx := context.Background()
	// Get instance details
	result, err := regionalClient.DescribeInstances(ctx, &ec2.DescribeInstancesInput{
		InstanceIds: []string{instanceID},
	})
	if err != nil {
		return false, "", false, fmt.Errorf("failed to describe instance: %w", err)
	}

	if len(result.Reservations) == 0 || len(result.Reservations[0].Instances) == 0 {
		return false, "", false, fmt.Errorf("instance not found")
	}

	instance := result.Reservations[0].Instances[0]

	// Get current state
	currentState := "unknown"
	if instance.State != nil {
		currentState = string(instance.State.Name)
	}

	// Check hibernation configuration
	hibernationSupported := instance.HibernationOptions != nil && *instance.HibernationOptions.Configured

	// We can only infer hibernation when:
	// 1. Instance supports hibernation
	// 2. Instance is in stopped state
	// 3. (In a real scenario, we'd check StateTransitionReason, but AWS doesn't clearly distinguish)
	// Note: This is a best-guess - AWS doesn't provide a definitive "was hibernated" flag
	possiblyHibernated := hibernationSupported && currentState == instanceStateStopped

	return hibernationSupported, currentState, possiblyHibernated, nil
}

// ApplyHibernationPolicy applies a hibernation policy template to an instance
func (m *Manager) ApplyHibernationPolicy(instanceName string, policyID string) error {
	// Get the instance ID
	instanceID, err := m.findInstanceByName(instanceName)
	if err != nil {
		return fmt.Errorf("failed to find instance: %w", err)
	}

	// Apply the policy
	if err := m.policyManager.ApplyTemplate(instanceID, policyID); err != nil {
		return fmt.Errorf("failed to apply hibernation policy: %w", err)
	}

	// Add schedules to the scheduler
	template, err := m.policyManager.GetTemplate(policyID)
	if err != nil {
		return fmt.Errorf("failed to get policy template: %w", err)
	}

	// Register each schedule with the instance
	for _, schedule := range template.Schedules {
		// Create a copy of the schedule with instance-specific ID
		instanceSchedule := schedule
		instanceSchedule.ID = fmt.Sprintf("%s-%s-%s", instanceID, policyID, schedule.Name)

		if err := m.idleScheduler.AddSchedule(&instanceSchedule); err != nil {
			return fmt.Errorf("failed to add schedule %s: %w", schedule.Name, err)
		}
	}

	return nil
}

// RemoveHibernationPolicy removes a hibernation policy from an instance
func (m *Manager) RemoveHibernationPolicy(instanceName string, policyID string) error {
	// Get the instance ID
	instanceID, err := m.findInstanceByName(instanceName)
	if err != nil {
		return fmt.Errorf("failed to find instance: %w", err)
	}

	// Get the policy template to find schedules
	template, err := m.policyManager.GetTemplate(policyID)
	if err != nil {
		return fmt.Errorf("failed to get policy template: %w", err)
	}

	// Remove schedules from the scheduler
	for _, schedule := range template.Schedules {
		scheduleID := fmt.Sprintf("%s-%s-%s", instanceID, policyID, schedule.Name)
		if err := m.idleScheduler.DeleteSchedule(scheduleID); err != nil {
			// Log but don't fail if schedule doesn't exist
			fmt.Printf("Warning: failed to delete schedule %s: %v\n", scheduleID, err)
		}
	}

	// Remove the policy
	if err := m.policyManager.RemoveTemplate(instanceID, policyID); err != nil {
		return fmt.Errorf("failed to remove hibernation policy: %w", err)
	}

	return nil
}

// ListHibernationPolicies returns all available hibernation policy templates
func (m *Manager) ListIdlePolicies() []*idle.PolicyTemplate {
	return m.policyManager.ListTemplates()
}

// GetIdlePolicy gets a specific idle policy template
func (m *Manager) GetIdlePolicy(policyID string) (*idle.PolicyTemplate, error) {
	return m.policyManager.GetTemplate(policyID)
}

// GetInstancePolicies returns the idle policies applied to an instance
func (m *Manager) GetInstancePolicies(instanceName string) ([]*idle.PolicyTemplate, error) {
	// Get the instance ID
	instanceID, err := m.findInstanceByName(instanceName)
	if err != nil {
		return nil, fmt.Errorf("failed to find instance: %w", err)
	}

	return m.policyManager.GetAppliedTemplates(instanceID)
}

// RecommendIdlePolicy recommends an idle policy based on instance characteristics
func (m *Manager) RecommendIdlePolicy(instanceName string) (*idle.PolicyTemplate, error) {
	// Get instance details
	instanceID, err := m.findInstanceByName(instanceName)
	if err != nil {
		return nil, fmt.Errorf("failed to find instance: %w", err)
	}

	ctx := context.Background()
	result, err := m.ec2.DescribeInstances(ctx, &ec2.DescribeInstancesInput{
		InstanceIds: []string{instanceID},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to describe instance: %w", err)
	}

	if len(result.Reservations) == 0 || len(result.Reservations[0].Instances) == 0 {
		return nil, fmt.Errorf("instance not found")
	}

	instance := result.Reservations[0].Instances[0]

	// Extract instance type and tags
	instanceType := string(instance.InstanceType)
	tags := make(map[string]string)
	for _, tag := range instance.Tags {
		if tag.Key != nil && tag.Value != nil {
			tags[*tag.Key] = *tag.Value
		}
	}

	return m.policyManager.RecommendTemplate(instanceType, tags)
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
func (m *Manager) CreateVolume(req ctypes.VolumeCreateRequest) (*ctypes.StorageVolume, error) {
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

	sizeBytes := int64(0) // Will be updated as files are added
	volume := &ctypes.StorageVolume{
		Name:            req.Name,
		Type:            ctypes.StorageTypeShared,
		AWSService:      ctypes.AWSServiceEFS,
		Region:          m.region,
		State:           string(result.LifeCycleState),
		CreationTime:    time.Now(),
		SizeBytes:       &sizeBytes,
		FileSystemID:    *result.FileSystemId,
		MountTargets:    []string{},
		PerformanceMode: performanceMode,
		ThroughputMode:  throughputMode,
		EstimatedCostGB: m.getRegionalEFSPrice(), // Regional EFS pricing
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
	if instance.State != instanceStateRunning {
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
func (m *Manager) CreateStorage(req ctypes.StorageCreateRequest) (*ctypes.StorageVolume, error) {
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
	if volumeType == volumeTypeIO2 || volumeType == "gp3" {
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

	sizeGB32 := int32(sizeGB)
	iops32 := int32(iops)
	throughput32 := int32(throughput)

	volume := &ctypes.StorageVolume{
		Name:            req.Name,
		Type:            ctypes.StorageTypeWorkspace,
		AWSService:      ctypes.AWSServiceEBS,
		Region:          m.region,
		State:           string(result.State),
		CreationTime:    time.Now(),
		SizeGB:          &sizeGB32,
		VolumeID:        *result.VolumeId,
		VolumeType:      volumeType,
		IOPS:            &iops32,
		Throughput:      &throughput32,
		AttachedTo:      "", // Not attached initially
		EstimatedCostGB: costPerGB,
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

// getInstanceTypeArchitecture queries AWS to determine the architecture of an instance type
// This ensures AMI architecture matches the actual instance type being launched
func (m *Manager) getInstanceTypeArchitecture(instanceType string) (string, error) {
	// Check cache first
	if arch, exists := m.architectureCache[instanceType]; exists {
		return arch, nil
	}

	// Query AWS for instance type details
	ctx := context.Background()
	input := &ec2.DescribeInstanceTypesInput{
		InstanceTypes: []ec2types.InstanceType{
			ec2types.InstanceType(instanceType),
		},
	}

	result, err := m.ec2.DescribeInstanceTypes(ctx, input)
	if err != nil {
		// Fallback to x86_64 on error (most widely available)
		log.Printf("Warning: Could not query instance type %s architecture, defaulting to x86_64: %v", instanceType, err)
		return "x86_64", nil
	}

	if len(result.InstanceTypes) == 0 {
		// Instance type not found, default to x86_64
		log.Printf("Warning: Instance type %s not found, defaulting to x86_64", instanceType)
		return "x86_64", nil
	}

	// Extract architectures supported by this instance type
	instanceTypeInfo := result.InstanceTypes[0]
	if len(instanceTypeInfo.ProcessorInfo.SupportedArchitectures) == 0 {
		// No architecture info, default to x86_64
		log.Printf("Warning: No architecture info for instance type %s, defaulting to x86_64", instanceType)
		return "x86_64", nil
	}

	// Get the first supported architecture (most instance types support only one)
	arch := string(instanceTypeInfo.ProcessorInfo.SupportedArchitectures[0])

	// Normalize architecture names to match AMI conventions
	normalizedArch := arch
	if arch == "x86_64_mac" {
		normalizedArch = "x86_64"
	}

	// Cache the result
	m.architectureCache[instanceType] = normalizedArch

	return normalizedArch, nil
}

// checkIAMInstanceProfileExists checks if an IAM instance profile exists, creating it if needed
// Returns true if it exists or was created successfully, false otherwise (doesn't error - just checks)
func (m *Manager) checkIAMInstanceProfileExists(profileName string) bool {
	// Validate IAM instance profile exists before launch
	// This enables SSM access for advanced features (remote execution, file operations)
	// Auto-creates the profile if it doesn't exist for zero-configuration SSM access

	ctx := context.Background()

	// Try to get the instance profile
	_, err := m.iam.GetInstanceProfile(ctx, &iam.GetInstanceProfileInput{
		InstanceProfileName: aws.String(profileName),
	})

	if err == nil {
		// Profile exists and is accessible
		return true
	}

	// Profile doesn't exist - try to create it
	log.Printf("IAM instance profile '%s' not found - attempting to create it automatically...", profileName)

	// Create IAM role first
	roleName := profileName + "-Role"

	// Trust policy allowing EC2 to assume this role
	trustPolicy := `{
		"Version": "2012-10-17",
		"Statement": [
			{
				"Effect": "Allow",
				"Principal": {
					"Service": "ec2.amazonaws.com"
				},
				"Action": "sts:AssumeRole"
			}
		]
	}`

	// Create the role
	_, err = m.iam.CreateRole(ctx, &iam.CreateRoleInput{
		RoleName:                 aws.String(roleName),
		AssumeRolePolicyDocument: aws.String(trustPolicy),
		Description:              aws.String("CloudWorkstation instance role for SSM access and autonomous idle detection"),
		Tags: []iamTypes.Tag{
			{
				Key:   aws.String("ManagedBy"),
				Value: aws.String("CloudWorkstation"),
			},
			{
				Key:   aws.String("Purpose"),
				Value: aws.String("Instance management and SSM access"),
			},
		},
	})

	if err != nil {
		// Role creation failed - check if it already exists
		if !strings.Contains(err.Error(), "EntityAlreadyExists") {
			log.Printf("Failed to create IAM role '%s': %v", roleName, err)
			log.Printf("Continuing without IAM profile - some features (SSM, autonomous idle detection) will be unavailable")
			return false
		}
		// Role already exists, continue to attach policies
	}

	// Attach AWS managed policy for SSM
	_, err = m.iam.AttachRolePolicy(ctx, &iam.AttachRolePolicyInput{
		RoleName:  aws.String(roleName),
		PolicyArn: aws.String("arn:aws:iam::aws:policy/AmazonSSMManagedInstanceCore"),
	})
	if err != nil && !strings.Contains(err.Error(), "already attached") {
		log.Printf("Warning: Failed to attach SSM policy to role: %v", err)
	}

	// Create inline policy for autonomous idle detection (EC2 self-management)
	idleDetectionPolicy := `{
		"Version": "2012-10-17",
		"Statement": [
			{
				"Effect": "Allow",
				"Action": [
					"ec2:CreateTags",
					"ec2:DescribeTags",
					"ec2:DescribeInstances",
					"ec2:StopInstances"
				],
				"Resource": "*"
			}
		]
	}`

	_, err = m.iam.PutRolePolicy(ctx, &iam.PutRolePolicyInput{
		RoleName:       aws.String(roleName),
		PolicyName:     aws.String("CloudWorkstation-IdleDetection"),
		PolicyDocument: aws.String(idleDetectionPolicy),
	})
	if err != nil {
		log.Printf("Warning: Failed to create idle detection policy: %v", err)
	}

	// Create the instance profile
	_, err = m.iam.CreateInstanceProfile(ctx, &iam.CreateInstanceProfileInput{
		InstanceProfileName: aws.String(profileName),
		Tags: []iamTypes.Tag{
			{
				Key:   aws.String("ManagedBy"),
				Value: aws.String("CloudWorkstation"),
			},
		},
	})
	if err != nil && !strings.Contains(err.Error(), "EntityAlreadyExists") {
		log.Printf("Failed to create instance profile '%s': %v", profileName, err)
		return false
	}

	// Add role to instance profile
	_, err = m.iam.AddRoleToInstanceProfile(ctx, &iam.AddRoleToInstanceProfileInput{
		InstanceProfileName: aws.String(profileName),
		RoleName:            aws.String(roleName),
	})
	if err != nil && !strings.Contains(err.Error(), "LimitExceeded") {
		log.Printf("Failed to add role to instance profile: %v", err)
		return false
	}

	// Wait a moment for IAM changes to propagate
	time.Sleep(2 * time.Second)

	log.Printf("✅ Successfully created IAM instance profile '%s' with SSM access and idle detection permissions", profileName)
	return true
}

// getInstanceTypeForSize maps size strings to default instance types
func (m *Manager) getInstanceTypeForSize(size string) string {
	// Map sizes to reasonable default x86_64 instance types
	// These are widely available across all regions
	sizeMap := map[string]string{
		"XS": "t3.micro",  // 1 vCPU, 2GB RAM
		"S":  "t3.small",  // 2 vCPU, 4GB RAM
		"M":  "t3.medium", // 2 vCPU, 8GB RAM
		"L":  "t3.large",  // 4 vCPU, 16GB RAM
		"XL": "t3.xlarge", // 8 vCPU, 32GB RAM
	}

	if instanceType, exists := sizeMap[size]; exists {
		return instanceType
	}

	// Default fallback
	return "t3.micro"
}

// getLocalArchitecture detects the local system architecture
// DEPRECATED: Use getInstanceTypeArchitecture instead for cloud instance launches
// This method should only be used for local system detection, not for selecting cloud AMIs
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

// getRegionalEC2Client creates an EC2 client for the specified region
// Reuses the existing client if region matches, creates new client otherwise
func (m *Manager) getRegionalEC2Client(region string) EC2ClientInterface {
	if region == m.region || region == "" {
		return m.ec2
	}
	regionalCfg := m.cfg.Copy()
	regionalCfg.Region = region
	return ec2.NewFromConfig(regionalCfg)
}

// getInstanceRegion looks up the region for an instance from state
func (m *Manager) getInstanceRegion(name string) (string, error) {
	state, err := m.stateManager.LoadState()
	if err != nil {
		return "", fmt.Errorf("failed to load state: %w", err)
	}

	for _, inst := range state.Instances {
		if inst.Name == name {
			if inst.Region != "" {
				return inst.Region, nil
			}
			break
		}
	}

	// Default to manager's region
	return m.region, nil
}

// findInstanceByName finds an EC2 instance by its Name tag
func (m *Manager) findInstanceByName(name string) (string, error) {
	// Load state to get the instance ID and region (fast - no AWS API calls)
	state, err := m.stateManager.LoadState()
	if err != nil {
		return "", fmt.Errorf("failed to load state: %w", err)
	}

	// Find instance in state to get its ID and region
	var instanceID string
	var instanceRegion string
	for _, inst := range state.Instances {
		if inst.Name == name {
			instanceID = inst.ID
			instanceRegion = inst.Region
			break
		}
	}

	// If we found the instance ID in state, return it immediately (fast path)
	if instanceID != "" {
		return instanceID, nil
	}

	// Default to manager's region if not found in state
	if instanceRegion == "" {
		instanceRegion = m.region
	}

	// Create EC2 client for the instance's region
	var regionalClient EC2ClientInterface
	if instanceRegion == m.region {
		regionalClient = m.ec2
	} else {
		regionalCfg := m.cfg.Copy()
		regionalCfg.Region = instanceRegion
		regionalClient = ec2.NewFromConfig(regionalCfg)
	}

	// Fallback: Query AWS if instance ID not in state (slow but handles edge cases)
	ctx := context.Background()
	result, err := regionalClient.DescribeInstances(ctx, &ec2.DescribeInstancesInput{
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
				Values: []string{"pending", instanceStateRunning, "shutting-down", "stopping", instanceStateStopped},
			},
		},
	})
	if err != nil {
		return "", fmt.Errorf("failed to describe instances in region %s: %w", instanceRegion, err)
	}

	for _, reservation := range result.Reservations {
		for _, instance := range reservation.Instances {
			return *instance.InstanceId, nil
		}
	}

	return "", fmt.Errorf("instance '%s' not found in region %s", name, instanceRegion)
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
		if awsState == instanceStateStopped {
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
	tagExtractor   *InstanceTagExtractor
	stateConverter *InstanceStateConverter
	ec2Client      EC2ClientInterface
	pricingClient  *PricingClient
	region         string
}

// NewInstanceBuilder creates instance builder
func NewInstanceBuilder(ec2Client EC2ClientInterface, pricingClient *PricingClient, region string) *InstanceBuilder {
	return &InstanceBuilder{
		tagExtractor:   &InstanceTagExtractor{},
		stateConverter: &InstanceStateConverter{},
		ec2Client:      ec2Client,
		pricingClient:  pricingClient,
		region:         region,
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

	// Get availability zone
	availabilityZone := ""
	if ec2Instance.Placement != nil && ec2Instance.Placement.AvailabilityZone != nil {
		availabilityZone = *ec2Instance.Placement.AvailabilityZone
	}

	// Determine instance lifecycle
	instanceLifecycle := "on-demand"
	if string(ec2Instance.InstanceLifecycle) == "spot" {
		instanceLifecycle = "spot"
	}

	// Get EC2 KeyName (SSH key pair)
	keyName := ""
	if ec2Instance.KeyName != nil {
		keyName = *ec2Instance.KeyName
	}

	// Determine launch time - preserve original launch time from cache across stop/start cycles
	// AWS's LaunchTime field changes to the START time when an instance is restarted,
	// but we want to track cost from the original launch for accurate total cost tracking
	launchTime := *ec2Instance.LaunchTime
	var stateHistory []ctypes.StateTransition
	if localState != nil {
		if localInstance, exists := localState.Instances[name]; exists {
			// Use cached launch time to preserve cost tracking across stop/start cycles
			launchTime = localInstance.LaunchTime
			// Get state history for accurate cost calculation
			stateHistory = localInstance.StateHistory
		}
	}

	// Calculate cost metrics based on instance type and runtime
	instanceType := string(ec2Instance.InstanceType)

	// Get accurate hourly rate from AWS Pricing API (with fallback to estimates)
	ctx := context.Background()
	hourlyRate, err := b.pricingClient.GetInstanceHourlyRate(ctx, b.region, instanceType)
	if err != nil {
		// Fall back to hardcoded estimate on error (pricing API failure)
		log.Printf("Warning: Pricing API failed for %s in %s: %v. Using estimate.", instanceType, b.region, err)
		hourlyRate = getHourlyRate(instanceType)
	}

	// Calculate EBS storage costs using AWS API (persist when stopped/hibernated)
	ebsStorageCostPerHour := b.calculateInstanceEBSCosts(*ec2Instance.InstanceId)

	// Calculate actual costs using state history for accurate tracking
	currentSpend, effectiveRate := calculateActualCosts(hourlyRate, ebsStorageCostPerHour, launchTime, state, stateHistory)

	// Create instance
	instance := &ctypes.Instance{
		ID:                *ec2Instance.InstanceId,
		Name:              name,
		Template:          template,
		State:             state,
		PublicIP:          publicIP,
		AvailabilityZone:  availabilityZone,
		ProjectID:         project,
		InstanceLifecycle: instanceLifecycle,
		KeyName:           keyName,
		LaunchTime:        launchTime,
		InstanceType:      instanceType,
		HourlyRate:        hourlyRate,
		CurrentSpend:      currentSpend,
		EffectiveRate:     effectiveRate,
		StateHistory:      stateHistory,
	}

	// Merge remaining metadata from local state if available
	// Local state contains fields that AWS doesn't store (deletion time, etc.)
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
func NewInstanceListProcessor(ec2Client EC2ClientInterface, pricingClient *PricingClient, region string) *InstanceListProcessor {
	return &InstanceListProcessor{
		stateLoader:     &StateLoader{},
		instanceBuilder: NewInstanceBuilder(ec2Client, pricingClient, region),
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

// GetInstance retrieves real-time information for a specific instance from AWS
func (m *Manager) GetInstance(instanceID string) (*ctypes.Instance, error) {
	ctx := context.Background()

	// Query AWS for this specific instance
	result, err := m.ec2.DescribeInstances(ctx, &ec2.DescribeInstancesInput{
		InstanceIds: []string{instanceID},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to describe instance %s: %w", instanceID, err)
	}

	if len(result.Reservations) == 0 || len(result.Reservations[0].Instances) == 0 {
		return nil, fmt.Errorf("instance %s not found", instanceID)
	}

	// Process the single instance
	processor := NewInstanceListProcessor(m.ec2, m.pricingClient, m.region)
	instances := processor.ProcessReservations(result.Reservations)

	if len(instances) == 0 {
		return nil, fmt.Errorf("failed to process instance %s", instanceID)
	}

	return &instances[0], nil
}

// ListInstances returns all CloudWorkstation instances using Strategy Pattern (SOLID: Single Responsibility)
func (m *Manager) ListInstances() ([]ctypes.Instance, error) {
	// Load state to get all instances and their regions
	state, err := m.stateManager.LoadState()
	if err != nil {
		return nil, fmt.Errorf("failed to load state: %w", err)
	}

	// Collect unique regions from saved instances
	regionsMap := make(map[string]bool)
	for _, instance := range state.Instances {
		if instance.Region != "" {
			regionsMap[instance.Region] = true
		}
	}

	// If no instances or no regions saved, use default region
	if len(regionsMap) == 0 {
		regionsMap[m.region] = true
	}

	// Query each region and collect results
	var allInstances []ctypes.Instance
	ctx := context.Background()

	for region := range regionsMap {
		// Create EC2 client for this specific region
		var regionalClient EC2ClientInterface
		if region == m.region {
			// Use existing client for default region
			regionalClient = m.ec2
		} else {
			// Create temporary client for other regions
			regionalCfg := m.cfg.Copy()
			regionalCfg.Region = region
			regionalClient = ec2.NewFromConfig(regionalCfg)
		}

		// Query instances in this region
		result, err := regionalClient.DescribeInstances(ctx, &ec2.DescribeInstancesInput{
			Filters: []ec2types.Filter{
				{
					Name:   aws.String("tag:CloudWorkstation"),
					Values: []string{"true"},
				},
				{
					Name:   aws.String("instance-state-name"),
					Values: []string{"pending", instanceStateRunning, "shutting-down", "stopping", instanceStateStopped, "terminating", "terminated"},
				},
			},
		})
		if err != nil {
			// Log error but continue with other regions
			log.Printf("Warning: Failed to query instances in region %s: %v", region, err)
			continue
		}

		// Process instances from this region
		processor := NewInstanceListProcessor(regionalClient, m.pricingClient, region)
		regionalInstances := processor.ProcessReservations(result.Reservations)

		// Ensure each instance has the region set and merge cached metadata
		for i := range regionalInstances {
			if regionalInstances[i].Region == "" {
				regionalInstances[i].Region = region
			}

			// Merge cached metadata (services, username, etc.) with live AWS data
			// AWS doesn't store our custom metadata, so preserve it from cache
			if cachedInstance, exists := state.Instances[regionalInstances[i].Name]; exists {
				regionalInstances[i].Services = cachedInstance.Services
				// Always use cached username if it exists (AWS never has this metadata)
				if cachedInstance.Username != "" {
					regionalInstances[i].Username = cachedInstance.Username
				}
			}
		}

		allInstances = append(allInstances, regionalInstances...)
	}

	return allInstances, nil
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

	case volumeTypeIO2:
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

// getRegionalEBSPrice returns region-aware EBS pricing
// Note: This uses estimated pricing. For production accuracy, integrate AWS Pricing API
func (m *Manager) getRegionalEBSPrice(volumeType string) float64 {
	// Get regional pricing multiplier
	regionMultiplier := m.getRegionPricingMultiplier()

	// Base US East 1 pricing (most accurate)
	var basePrice float64
	switch volumeType {
	case "gp3":
		basePrice = 0.08 // $0.08 per GB per month in us-east-1
	case "gp2":
		basePrice = 0.10 // $0.10 per GB per month in us-east-1
	case volumeTypeIO2:
		basePrice = 0.125 // $0.125 per GB per month in us-east-1
	case "st1":
		basePrice = 0.045 // $0.045 per GB per month in us-east-1
	case "sc1":
		basePrice = 0.025 // $0.025 per GB per month in us-east-1
	default:
		basePrice = 0.10 // Default to gp2 pricing
	}

	regionalPrice := basePrice * regionMultiplier
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

// getRegionalEC2Price returns region-aware EC2 pricing
// Note: This is now primarily used as fallback. Production pricing comes from AWS Pricing API
func (m *Manager) getRegionalEC2Price(instanceType string) float64 {
	basePrice := m.getBaseInstancePrice(instanceType)
	regionalPrice := basePrice * m.getRegionPricingMultiplier()
	finalPrice := m.applyEC2Discounts(regionalPrice)
	return finalPrice
}

func (m *Manager) getBaseInstancePrice(instanceType string) float64 {
	basePrices := m.getInstanceBasePrices()

	if price, exists := basePrices[instanceType]; exists {
		return price
	}

	return m.estimateInstancePrice(instanceType)
}

func (m *Manager) getInstanceBasePrices() map[string]float64 {
	return map[string]float64{
		// General Purpose
		"t3.micro":   0.0104,
		"t3.small":   0.0208,
		"t3.medium":  0.0416,
		"t3.large":   0.0832,
		"t3.xlarge":  0.1664,
		"t3.2xlarge": 0.3328,

		// Compute Optimized
		"c5.large":   0.085,
		"c5.xlarge":  0.17,
		"c5.2xlarge": 0.34,
		"c5.4xlarge": 0.68,

		// Memory Optimized
		"r5.large":   0.126,
		"r5.xlarge":  0.252,
		"r5.2xlarge": 0.504,
		"r5.4xlarge": 1.008,

		// GPU Instances
		"g4dn.xlarge":  0.526,
		"g4dn.2xlarge": 0.752,
		"g4dn.4xlarge": 1.204,
	}
}

// estimateInstancePrice estimates pricing for unknown instance types
func (m *Manager) estimateInstancePrice(instanceType string) float64 {
	parts := strings.Split(instanceType, ".")
	if len(parts) != 2 {
		return 0.10 // Conservative fallback
	}

	familyBase := m.getInstanceFamilyBase(parts[0])
	sizeMultiplier := m.getInstanceSizeMultiplier(parts[1])

	return familyBase * sizeMultiplier
}

func (m *Manager) getInstanceFamilyBase(family string) float64 {
	familyBasePrices := map[string]float64{
		"t3":  0.0104, // t3.micro base
		"t4g": 0.0084, // ARM instances slightly cheaper
		"c5":  0.085,  // c5.large base
		"r5":  0.126,  // r5.large base
		"g4":  0.526,  // GPU base
	}

	for prefix, price := range familyBasePrices {
		if strings.HasPrefix(family, prefix) {
			return price
		}
	}

	return 0.05 // Conservative default
}

func (m *Manager) getInstanceSizeMultiplier(size string) float64 {
	sizeMultipliers := map[string]float64{
		"nano":     0.25,
		"micro":    0.5,
		"small":    1.0,
		"medium":   2.0,
		"large":    4.0,
		"xlarge":   8.0,
		"2xlarge":  16.0,
		"4xlarge":  32.0,
		"8xlarge":  64.0,
		"12xlarge": 96.0,
		"16xlarge": 128.0,
		"24xlarge": 192.0,
	}

	if multiplier, exists := sizeMultipliers[size]; exists {
		return multiplier
	}

	return 4.0 // Default to large
}

// getRegionalEFSPrice returns region-aware EFS pricing
// Note: This uses estimated pricing. For production accuracy, integrate AWS Pricing API
func (m *Manager) getRegionalEFSPrice() float64 {
	// Get regional pricing multiplier
	regionMultiplier := m.getRegionPricingMultiplier()

	// Base US East 1 EFS pricing: $0.30 per GB per month for Standard storage
	basePrice := 0.30
	regionalPrice := basePrice * regionMultiplier

	// Apply discounts
	finalPrice := m.applyEFSDiscounts(regionalPrice)
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
	// Note: Discount changes will apply on next price calculation
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
// DiscoverPublicSubnetForInstanceType finds a public subnet that supports the specified instance type
// This prevents launch failures due to instance type not being available in a randomly selected AZ
func (m *Manager) DiscoverPublicSubnetForInstanceType(vpcID, instanceType string) (string, error) {
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

	// Get availability zones that support this instance type
	offeringsResult, err := m.ec2.DescribeInstanceTypeOfferings(ctx, &ec2.DescribeInstanceTypeOfferingsInput{
		LocationType: ec2types.LocationTypeAvailabilityZone,
		Filters: []ec2types.Filter{
			{
				Name:   aws.String("instance-type"),
				Values: []string{instanceType},
			},
		},
	})
	if err != nil {
		log.Printf("Warning: Failed to check instance type availability, will try first public subnet: %v", err)
		// Fallback to old behavior
		return m.DiscoverPublicSubnet(vpcID)
	}

	// Build map of AZs that support this instance type
	supportedAZs := make(map[string]bool)
	for _, offering := range offeringsResult.InstanceTypeOfferings {
		if offering.Location != nil {
			supportedAZs[*offering.Location] = true
		}
	}

	// Find a public subnet in a supported AZ
	for _, subnet := range result.Subnets {
		// Check if subnet's AZ supports the instance type
		if subnet.AvailabilityZone != nil && supportedAZs[*subnet.AvailabilityZone] {
			// Check if subnet is public
			isPublic, err := m.isSubnetPublic(*subnet.SubnetId)
			if err != nil {
				continue // Skip this subnet on error
			}
			if isPublic {
				log.Printf("Selected subnet %s in AZ %s (supports %s)", *subnet.SubnetId, *subnet.AvailabilityZone, instanceType)
				return *subnet.SubnetId, nil
			}
		}
	}

	// If no public subnet found in supported AZ, try any subnet in supported AZ
	// (handles cases where route table detection fails)
	for _, subnet := range result.Subnets {
		if subnet.AvailabilityZone != nil && supportedAZs[*subnet.AvailabilityZone] {
			log.Printf("Selected subnet %s in AZ %s (supports %s, assuming public)", *subnet.SubnetId, *subnet.AvailabilityZone, instanceType)
			return *subnet.SubnetId, nil
		}
	}

	return "", fmt.Errorf("no subnet found that supports instance type %s in VPC %s", instanceType, vpcID)
}

// DiscoverPublicSubnet finds a public subnet (legacy method, kept for fallback)
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

	// Determine access strategy for web interfaces (handles dynamic IPs)
	accessConfig := security.DetermineAccessStrategy()
	log.Printf("🔐 Web access strategy: %s", accessConfig.Message)

	// Configure web interface access based on strategy
	webPorts := []struct {
		port        int32
		protocol    string
		description string
	}{
		{80, "tcp", "HTTP web interfaces"},
		{443, "tcp", "HTTPS web interfaces"},
		{8888, "tcp", "Jupyter notebook access"},
		{8787, "tcp", "RStudio Server access"},
	}

	switch accessConfig.Strategy {
	case security.AccessDirect:
		// Direct access from user's specific IP
		for _, webPort := range webPorts {
			_, err = m.ec2.AuthorizeSecurityGroupIngress(ctx, &ec2.AuthorizeSecurityGroupIngressInput{
				GroupId: aws.String(securityGroupID),
				IpPermissions: []ec2types.IpPermission{
					{
						IpProtocol: aws.String(webPort.protocol),
						FromPort:   aws.Int32(webPort.port),
						ToPort:     aws.Int32(webPort.port),
						IpRanges: []ec2types.IpRange{
							{
								CidrIp:      aws.String(fmt.Sprintf("%s/32", accessConfig.UserIP)),
								Description: aws.String(fmt.Sprintf("Direct %s from %s", webPort.description, accessConfig.UserIP)),
							},
						},
					},
				},
			})
			if err != nil {
				return "", fmt.Errorf("failed to add direct access rule for port %d: %w", webPort.port, err)
			}
		}
		log.Printf("✅ Direct web access configured for IP %s", accessConfig.UserIP)

	case security.AccessSubnet:
		// Subnet-based access (handles DHCP changes)
		for _, webPort := range webPorts {
			_, err = m.ec2.AuthorizeSecurityGroupIngress(ctx, &ec2.AuthorizeSecurityGroupIngressInput{
				GroupId: aws.String(securityGroupID),
				IpPermissions: []ec2types.IpPermission{
					{
						IpProtocol: aws.String(webPort.protocol),
						FromPort:   aws.Int32(webPort.port),
						ToPort:     aws.Int32(webPort.port),
						IpRanges: []ec2types.IpRange{
							{
								CidrIp:      aws.String(accessConfig.SubnetCIDR),
								Description: aws.String(fmt.Sprintf("Subnet %s for %s", accessConfig.SubnetCIDR, webPort.description)),
							},
						},
					},
				},
			})
			if err != nil {
				return "", fmt.Errorf("failed to add subnet access rule for port %d: %w", webPort.port, err)
			}
		}
		log.Printf("✅ Subnet web access configured for %s (handles DHCP changes)", accessConfig.SubnetCIDR)

	case security.AccessTunneled:
		// SSH tunneling required - no direct web access rules
		log.Println("🔒 Web interfaces secured to localhost - SSH tunneling required")
		log.Println("   Example: ssh -L 8888:localhost:8888 user@instance")
		log.Println("   Then access: http://localhost:8888")
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

// getHourlyRate returns the AWS list price per hour for an instance type
func getHourlyRate(instanceType string) float64 {
	// Instance pricing data (USD per hour) - estimated rates for us-east-1
	instancePricing := map[string]float64{
		// General Purpose
		"t3.micro":    0.0104,
		"t3.small":    0.0208,
		"t3.medium":   0.0416,
		"t3.large":    0.0832,
		"t3.xlarge":   0.1664,
		"t3.2xlarge":  0.3328,
		"t3a.micro":   0.0094,
		"t3a.small":   0.0188,
		"t3a.medium":  0.0376,
		"t3a.large":   0.0752,
		"t3a.xlarge":  0.1504,
		"t3a.2xlarge": 0.3008,

		// Compute Optimized
		"c5.large":    0.085,
		"c5.xlarge":   0.17,
		"c5.2xlarge":  0.34,
		"c5.4xlarge":  0.68,
		"c5.9xlarge":  1.53,
		"c5.12xlarge": 2.04,
		"c5.18xlarge": 3.06,
		"c5.24xlarge": 4.08,

		// Memory Optimized
		"r5.large":    0.126,
		"r5.xlarge":   0.252,
		"r5.2xlarge":  0.504,
		"r5.4xlarge":  1.008,
		"r5.8xlarge":  2.016,
		"r5.12xlarge": 3.024,
		"r5.16xlarge": 4.032,
		"r5.24xlarge": 6.048,

		// GPU Instances
		"g4dn.xlarge":   0.526,
		"g4dn.2xlarge":  0.752,
		"g4dn.4xlarge":  1.204,
		"g4dn.8xlarge":  2.176,
		"g4dn.12xlarge": 3.912,
		"g4dn.16xlarge": 4.352,
		"p3.2xlarge":    3.06,
		"p3.8xlarge":    12.24,
		"p3.16xlarge":   24.48,
		"p4d.24xlarge":  32.77,
	}

	// Look up hourly rate
	hourlyRate, exists := instancePricing[instanceType]
	if !exists {
		// Estimate for unknown instance types
		hourlyRate = estimateInstanceCost(instanceType)
	}

	return hourlyRate
}

// estimateInstanceCost estimates the hourly cost for unknown instance types
func estimateInstanceCost(instanceType string) float64 {
	// Extract instance family and size
	parts := strings.Split(instanceType, ".")
	if len(parts) != 2 {
		return 0.10 // Default fallback rate
	}

	family := parts[0]
	size := parts[1]

	// Base rates by instance family
	familyRates := map[string]float64{
		"t3":   0.0104, // t3.micro base rate
		"t3a":  0.0094, // t3a.micro base rate
		"c5":   0.085,  // c5.large base rate
		"c5n":  0.108,  // c5n.large base rate
		"r5":   0.126,  // r5.large base rate
		"r5a":  0.113,  // r5a.large base rate
		"m5":   0.096,  // m5.large base rate
		"m5a":  0.086,  // m5a.large base rate
		"g4dn": 0.526,  // g4dn.xlarge base rate
		"p3":   3.06,   // p3.2xlarge base rate
		"p4d":  32.77,  // p4d.24xlarge base rate
	}

	baseRate, exists := familyRates[family]
	if !exists {
		baseRate = 0.10 // Default rate
	}

	// Size multipliers
	sizeMultipliers := map[string]float64{
		"nano":     0.25,
		"micro":    0.5,
		"small":    1.0,
		"medium":   2.0,
		"large":    4.0,
		"xlarge":   8.0,
		"2xlarge":  16.0,
		"3xlarge":  24.0,
		"4xlarge":  32.0,
		"6xlarge":  48.0,
		"8xlarge":  64.0,
		"9xlarge":  72.0,
		"12xlarge": 96.0,
		"16xlarge": 128.0,
		"18xlarge": 144.0,
		"24xlarge": 192.0,
		"32xlarge": 256.0,
	}

	multiplier, exists := sizeMultipliers[size]
	if !exists {
		multiplier = 4.0 // Default to large equivalent
	}

	return baseRate * multiplier
}

// calculateActualCosts calculates current spend and effective rate based on actual usage
func calculateActualCosts(computeHourlyRate float64, storageHourlyRate float64, launchTime time.Time, currentState string, stateHistory []ctypes.StateTransition) (currentSpend, effectiveRate float64) {
	now := time.Now()
	totalHours := now.Sub(launchTime).Hours()

	if totalHours <= 0 {
		return 0, 0
	}

	// Calculate actual running time from state history if available
	var runningHours float64
	if len(stateHistory) > 0 {
		// Use state history for accurate cost calculation
		runningHours = calculateRunningHoursFromHistory(launchTime, currentState, stateHistory)
	} else {
		// Fallback to estimation if no state history (legacy instances or first launch)
		switch strings.ToLower(currentState) {
		case instanceStateRunning:
			// If currently running and no history, assume it's been running the whole time
			runningHours = totalHours

		case instanceStateStopped, "hibernated":
			// If currently stopped/hibernated and no history, estimate it ran for part of the time
			runningHours = totalHours * 0.6 // Conservative estimate

		case "pending", "shutting-down":
			// Transitional states - estimate partial running time
			runningHours = totalHours * 0.9

		default:
			// Default conservative estimate
			runningHours = totalHours * 0.7
		}
	}

	// Calculate current spend
	// Compute costs: only for running hours (savings from stop/hibernation!)
	computeCost := runningHours * computeHourlyRate
	// Storage costs: persist for all hours (EBS continues when stopped/hibernated)
	storageCost := totalHours * storageHourlyRate
	currentSpend = computeCost + storageCost

	// Calculate effective rate (actual spend per total hour)
	effectiveRate = currentSpend / totalHours

	return currentSpend, effectiveRate
}

// calculateRunningHoursFromHistory calculates actual running hours from state transition history
func calculateRunningHoursFromHistory(launchTime time.Time, currentState string, history []ctypes.StateTransition) float64 {
	if len(history) == 0 {
		// No history - use simple calculation based on current state
		now := time.Now()
		totalHours := now.Sub(launchTime).Hours()
		if strings.ToLower(currentState) == instanceStateRunning || currentState == "pending" {
			return totalHours
		}
		return 0
	}

	// Filter out transitions that happened before current launch time
	// This handles the case where an instance was stopped and restarted,
	// and the launch time was updated but old state history remains
	var relevantHistory []ctypes.StateTransition
	for _, transition := range history {
		if transition.Timestamp.After(launchTime) {
			relevantHistory = append(relevantHistory, transition)
		}
	}

	// If no relevant history after launch time, use simple calculation
	if len(relevantHistory) == 0 {
		now := time.Now()
		totalHours := now.Sub(launchTime).Hours()
		if strings.ToLower(currentState) == instanceStateRunning || currentState == "pending" {
			return totalHours
		}
		return 0
	}

	var runningHours float64
	lastStateTime := launchTime
	lastState := instanceStateRunning // Instances start in running/pending state

	// Process each state transition to calculate running time
	for _, transition := range relevantHistory {
		// Calculate duration in previous state
		duration := transition.Timestamp.Sub(lastStateTime).Hours()

		// Add to running hours if previous state was a "running" state
		if lastState == instanceStateRunning || lastState == "pending" {
			runningHours += duration
		}

		// Update for next iteration
		lastStateTime = transition.Timestamp
		lastState = transition.ToState
	}

	// Add time from last transition to now
	now := time.Now()
	finalDuration := now.Sub(lastStateTime).Hours()
	if lastState == instanceStateRunning || lastState == "pending" {
		runningHours += finalDuration
	}

	return runningHours
}

// calculateStorageCosts calculates hourly EBS storage costs for this instance
// Note: EFS costs are tracked separately since EFS volumes are shared across instances
func calculateStorageCosts(efsVolumes []string, ebsVolumes []string) float64 {
	var totalEBSCostPerHour float64

	// EBS pricing (monthly rates converted to hourly)
	ebsGP3PerGB := 0.08 / (30 * 24) // $0.08/GB/month → hourly
	ebsVolumeEstimatedGB := 100.0   // Default estimate per EBS volume

	// Calculate EBS costs (persist when stopped/hibernated)
	// These are instance-specific costs
	for range ebsVolumes {
		totalEBSCostPerHour += ebsVolumeEstimatedGB * ebsGP3PerGB
	}

	// Add estimated root EBS volume cost (typically 20GB gp3)
	// Every instance has a root volume
	rootVolumeGB := 20.0
	totalEBSCostPerHour += rootVolumeGB * ebsGP3PerGB

	return totalEBSCostPerHour
}

// calculateInstanceEBSCosts calculates hourly EBS costs for a specific instance using AWS API
func (b *InstanceBuilder) calculateInstanceEBSCosts(instanceId string) float64 {
	ctx := context.Background()

	// Get instance details to find attached volumes
	input := &ec2.DescribeInstancesInput{
		InstanceIds: []string{instanceId},
	}

	result, err := b.ec2Client.DescribeInstances(ctx, input)
	if err != nil {
		// If API call fails, return estimated cost for root volume (20GB gp3)
		return b.pricingClient.GetEBSVolumeHourlyRate("gp3", 20)
	}

	if len(result.Reservations) == 0 || len(result.Reservations[0].Instances) == 0 {
		// Fallback estimate (20GB gp3)
		return b.pricingClient.GetEBSVolumeHourlyRate("gp3", 20)
	}

	instance := result.Reservations[0].Instances[0]
	var totalEBSCostPerHour float64

	// Calculate costs for all attached EBS volumes using actual AWS data
	for _, blockDevice := range instance.BlockDeviceMappings {
		if blockDevice.Ebs != nil && blockDevice.Ebs.VolumeId != nil {
			volumeCost := b.getEBSVolumeCost(*blockDevice.Ebs.VolumeId)
			totalEBSCostPerHour += volumeCost
		}
	}

	return totalEBSCostPerHour
}

// getEBSVolumeCost gets the hourly cost for a specific EBS volume using actual AWS volume data
func (b *InstanceBuilder) getEBSVolumeCost(volumeId string) float64 {
	ctx := context.Background()

	input := &ec2.DescribeVolumesInput{
		VolumeIds: []string{volumeId},
	}

	result, err := b.ec2Client.DescribeVolumes(ctx, input)
	if err != nil {
		// Fallback to gp3 estimate (20GB)
		return b.pricingClient.GetEBSVolumeHourlyRate("gp3", 20)
	}

	if len(result.Volumes) == 0 {
		// Fallback to gp3 estimate (20GB)
		return b.pricingClient.GetEBSVolumeHourlyRate("gp3", 20)
	}

	volume := result.Volumes[0]
	volumeType := string(volume.VolumeType)
	volumeSize := int(*volume.Size)

	// Get accurate pricing from PricingClient
	return b.pricingClient.GetEBSVolumeHourlyRate(volumeType, volumeSize)
}

// GetInstanceConsoleOutput retrieves console output logs from an EC2 instance
func (m *Manager) GetInstanceConsoleOutput(instanceID string) (string, error) {
	input := &ec2.GetConsoleOutputInput{
		InstanceId: aws.String(instanceID),
		Latest:     aws.Bool(true),
	}

	ctx := context.Background()
	result, err := m.ec2.GetConsoleOutput(ctx, input)
	if err != nil {
		return "", fmt.Errorf("failed to get console output for instance %s: %w", instanceID, err)
	}

	if result.Output == nil {
		return "", fmt.Errorf("no console output available for instance %s", instanceID)
	}

	// Decode base64 encoded console output
	output, err := base64.StdEncoding.DecodeString(*result.Output)
	if err != nil {
		return "", fmt.Errorf("failed to decode console output: %w", err)
	}

	return string(output), nil
}

// GetInstanceSystemLogs retrieves system logs via SSM (Systems Manager)
func (m *Manager) GetInstanceSystemLogs(instanceID string, logType string) ([]string, error) {
	// Define common log paths
	logPaths := map[string]string{
		"cloud-init":     "/var/log/cloud-init.log",
		"cloud-init-out": "/var/log/cloud-init-output.log",
		"messages":       "/var/log/messages",
		"secure":         "/var/log/secure",
		"boot":           "/var/log/boot.log",
		"dmesg":          "/var/log/dmesg",
		"kern":           "/var/log/kern.log",
		"syslog":         "/var/log/syslog",
	}

	logPath, exists := logPaths[logType]
	if !exists {
		return nil, fmt.Errorf("unknown log type: %s", logType)
	}

	// Use SSM to retrieve log contents
	command := fmt.Sprintf("tail -n 1000 %s 2>/dev/null || echo 'Log file not found or accessible'", logPath)

	input := &ssm.SendCommandInput{
		InstanceIds:  []string{instanceID},
		DocumentName: aws.String("AWS-RunShellScript"),
		Parameters: map[string][]string{
			"commands": {command},
		},
	}

	ctx := context.Background()
	result, err := m.ssm.SendCommand(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to send command to instance %s: %w", instanceID, err)
	}

	// Wait for command to complete and get results
	commandID := *result.Command.CommandId
	return m.waitForSSMCommandResults(instanceID, commandID)
}

// waitForSSMCommandResults waits for SSM command to complete and returns the output
func (m *Manager) waitForSSMCommandResults(instanceID, commandID string) ([]string, error) {
	ctx := context.Background()
	maxWaitTime := 30 * time.Second
	pollInterval := 2 * time.Second
	startTime := time.Now()

	for time.Since(startTime) < maxWaitTime {
		// Check command status
		statusInput := &ssm.GetCommandInvocationInput{
			CommandId:  aws.String(commandID),
			InstanceId: aws.String(instanceID),
		}

		statusResult, err := m.ssm.GetCommandInvocation(ctx, statusInput)
		if err != nil {
			time.Sleep(pollInterval)
			continue
		}

		// Command completed
		if statusResult.Status == "Success" || statusResult.Status == "Failed" {
			if statusResult.StandardOutputContent != nil {
				lines := strings.Split(strings.TrimSpace(*statusResult.StandardOutputContent), "\n")
				return lines, nil
			}
			return []string{}, nil
		}

		// Command still running, wait
		if statusResult.Status == "InProgress" {
			time.Sleep(pollInterval)
			continue
		}

		// Command failed or cancelled
		if statusResult.Status == "Failed" || statusResult.Status == "Cancelled" {
			errorMsg := "Unknown error"
			if statusResult.StandardErrorContent != nil {
				errorMsg = *statusResult.StandardErrorContent
			}
			return nil, fmt.Errorf("SSM command failed: %s", errorMsg)
		}
	}

	return nil, fmt.Errorf("timeout waiting for SSM command to complete")
}

// ExecuteCommand executes a command on an instance via SSM
func (m *Manager) ExecuteCommand(instanceName string, execRequest ctypes.ExecRequest) (*ctypes.ExecResult, error) {
	// Get instance ID from state
	state, err := m.stateManager.LoadState()
	if err != nil {
		return nil, fmt.Errorf("failed to load state: %w", err)
	}

	instanceData, exists := state.Instances[instanceName]
	if !exists {
		return nil, fmt.Errorf("instance %s not found", instanceName)
	}

	instanceID := instanceData.ID
	startTime := time.Now()

	// Build command with user and working directory if specified
	command := execRequest.Command
	if execRequest.User != "" {
		// Use 'sudo -u <user>' to execute as different user
		command = fmt.Sprintf("sudo -u %s bash -c '%s'", execRequest.User, strings.ReplaceAll(command, "'", "'\"'\"'"))
	}
	if execRequest.WorkingDir != "" {
		// Change directory before executing command
		command = fmt.Sprintf("cd %s && %s", execRequest.WorkingDir, command)
	}

	// Add environment variables if specified
	if len(execRequest.Environment) > 0 {
		var envVars []string
		for key, value := range execRequest.Environment {
			envVars = append(envVars, fmt.Sprintf("export %s=%s", key, value))
		}
		command = strings.Join(envVars, " && ") + " && " + command
	}

	// Set timeout (default 30 seconds)
	timeout := 30
	if execRequest.TimeoutSeconds > 0 {
		timeout = execRequest.TimeoutSeconds
	}

	// Send SSM command
	input := &ssm.SendCommandInput{
		InstanceIds:  []string{instanceID},
		DocumentName: aws.String("AWS-RunShellScript"),
		Parameters: map[string][]string{
			"commands":         {command},
			"executionTimeout": {fmt.Sprintf("%d", timeout)},
		},
	}

	ctx := context.Background()
	result, err := m.ssm.SendCommand(ctx, input)
	if err != nil {
		return &ctypes.ExecResult{
			Command:  execRequest.Command,
			ExitCode: 1,
			StdErr:   fmt.Sprintf("Failed to send SSM command: %v", err),
			Status:   "failed",
		}, nil
	}

	commandID := *result.Command.CommandId

	// Wait for command to complete with enhanced result
	execResult, err := m.waitForSSMCommandExecution(instanceID, commandID, execRequest.Command, startTime)
	if err != nil {
		return &ctypes.ExecResult{
			Command:   execRequest.Command,
			ExitCode:  1,
			StdErr:    err.Error(),
			Status:    "failed",
			CommandID: commandID,
		}, nil
	}

	return execResult, nil
}

// waitForSSMCommandExecution waits for SSM command execution and returns structured results
func (m *Manager) waitForSSMCommandExecution(instanceID, commandID, originalCommand string, startTime time.Time) (*ctypes.ExecResult, error) {
	ctx := context.Background()
	maxWaitTime := 60 * time.Second // Increased timeout for exec commands
	pollInterval := 2 * time.Second
	commandStartTime := time.Now()

	for time.Since(commandStartTime) < maxWaitTime {
		// Check command status
		statusInput := &ssm.GetCommandInvocationInput{
			CommandId:  aws.String(commandID),
			InstanceId: aws.String(instanceID),
		}

		statusResult, err := m.ssm.GetCommandInvocation(ctx, statusInput)
		if err != nil {
			time.Sleep(pollInterval)
			continue
		}

		executionTime := int(time.Since(startTime).Milliseconds())

		// Command completed successfully
		if statusResult.Status == "Success" {
			stdout := ""
			if statusResult.StandardOutputContent != nil {
				stdout = *statusResult.StandardOutputContent
			}
			stderr := ""
			if statusResult.StandardErrorContent != nil {
				stderr = *statusResult.StandardErrorContent
			}

			return &ctypes.ExecResult{
				Command:       originalCommand,
				ExitCode:      0,
				StdOut:        stdout,
				StdErr:        stderr,
				Status:        "success",
				ExecutionTime: executionTime,
				CommandID:     commandID,
			}, nil
		}

		// Command failed
		if statusResult.Status == "Failed" {
			stdout := ""
			if statusResult.StandardOutputContent != nil {
				stdout = *statusResult.StandardOutputContent
			}
			stderr := ""
			if statusResult.StandardErrorContent != nil {
				stderr = *statusResult.StandardErrorContent
			}

			return &ctypes.ExecResult{
				Command:       originalCommand,
				ExitCode:      1,
				StdOut:        stdout,
				StdErr:        stderr,
				Status:        "failed",
				ExecutionTime: executionTime,
				CommandID:     commandID,
			}, nil
		}

		// Command cancelled
		if statusResult.Status == "Cancelled" {
			return &ctypes.ExecResult{
				Command:       originalCommand,
				ExitCode:      130, // Standard exit code for cancelled commands
				StdErr:        "Command was cancelled",
				Status:        "failed",
				ExecutionTime: executionTime,
				CommandID:     commandID,
			}, nil
		}

		// Command still running, wait
		if statusResult.Status == "InProgress" {
			time.Sleep(pollInterval)
			continue
		}
	}

	// Timeout occurred
	return &ctypes.ExecResult{
		Command:       originalCommand,
		ExitCode:      124, // Standard timeout exit code
		StdErr:        "Command execution timed out",
		Status:        "timeout",
		ExecutionTime: int(maxWaitTime.Milliseconds()),
		CommandID:     commandID,
	}, nil
}

// GetAvailableLogTypes returns the available log types that can be retrieved
func (m *Manager) GetAvailableLogTypes() []string {
	return []string{
		"console",        // EC2 console output
		"cloud-init",     // Cloud-init logs
		"cloud-init-out", // Cloud-init output
		"messages",       // System messages
		"secure",         // Security/auth logs
		"boot",           // Boot logs
		"dmesg",          // Kernel ring buffer
		"kern",           // Kernel logs
		"syslog",         // System logs
	}
}

// ResizeInstance resizes an EC2 instance to a new instance type
// findTargetInstance finds an instance by name
func (m *Manager) findTargetInstance(instanceName string) (*ctypes.Instance, error) {
	instances, err := m.ListInstances()
	if err != nil {
		return nil, fmt.Errorf("failed to list instances: %w", err)
	}

	for _, instance := range instances {
		if instance.Name == instanceName {
			return &instance, nil
		}
	}

	return nil, nil // Not found
}

// validateResizeRequest validates resize request and returns whether stop is needed
func (m *Manager) validateResizeRequest(targetInstance *ctypes.Instance, targetType string) (needsStop bool, err error) {
	// Check if resize is needed
	if targetInstance.InstanceType == targetType {
		return false, fmt.Errorf("instance already type %s", targetType)
	}

	// Validate instance state (must be stopped to resize)
	if targetInstance.State == "running" {
		return true, nil
	}

	if targetInstance.State != "stopped" {
		return false, fmt.Errorf("instance in state '%s', must be 'running' or 'stopped'", targetInstance.State)
	}

	return false, nil
}

// stopInstanceForResize stops an instance and waits for it to stop
func (m *Manager) stopInstanceForResize(ctx context.Context, instanceID string) error {
	_, err := m.ec2.StopInstances(ctx, &ec2.StopInstancesInput{
		InstanceIds: []string{instanceID},
	})
	if err != nil {
		return fmt.Errorf("failed to stop instance: %w", err)
	}

	// Wait for instance to stop
	waiter := ec2.NewInstanceStoppedWaiter(m.ec2)
	err = waiter.Wait(ctx, &ec2.DescribeInstancesInput{
		InstanceIds: []string{instanceID},
	}, 5*time.Minute)
	if err != nil {
		return fmt.Errorf("timeout waiting for instance to stop: %w", err)
	}

	return nil
}

// modifyInstanceType changes the instance type
func (m *Manager) modifyInstanceType(ctx context.Context, instanceID, targetType string) error {
	_, err := m.ec2.ModifyInstanceAttribute(ctx, &ec2.ModifyInstanceAttributeInput{
		InstanceId: &instanceID,
		InstanceType: &ec2types.AttributeValue{
			Value: &targetType,
		},
	})
	if err != nil {
		return fmt.Errorf("failed to modify instance type: %w", err)
	}

	return nil
}

// startInstanceAfterResize starts an instance after resize
func (m *Manager) startInstanceAfterResize(ctx context.Context, instanceID string, wait bool) error {
	_, err := m.ec2.StartInstances(ctx, &ec2.StartInstancesInput{
		InstanceIds: []string{instanceID},
	})
	if err != nil {
		return fmt.Errorf("failed to start instance: %w", err)
	}

	// Wait for instance to start if requested
	if wait {
		waiter := ec2.NewInstanceRunningWaiter(m.ec2)
		err = waiter.Wait(ctx, &ec2.DescribeInstancesInput{
			InstanceIds: []string{instanceID},
		}, 5*time.Minute)
		if err != nil {
			return fmt.Errorf("timeout waiting for instance to start: %w", err)
		}
	}

	return nil
}

// ResizeInstance orchestrates instance resize with extracted helper methods
func (m *Manager) ResizeInstance(resizeRequest ctypes.ResizeRequest) (*ctypes.ResizeResponse, error) {
	ctx := context.Background()

	// Find target instance
	targetInstance, err := m.findTargetInstance(resizeRequest.InstanceName)
	if err != nil {
		return nil, err
	}

	if targetInstance == nil {
		return &ctypes.ResizeResponse{
			Success: false,
			Message: fmt.Sprintf("Instance '%s' not found", resizeRequest.InstanceName),
		}, nil
	}

	// Validate resize request
	needsStop, err := m.validateResizeRequest(targetInstance, resizeRequest.TargetInstanceType)
	if err != nil {
		// Check if this is the "already correct type" case
		if strings.Contains(err.Error(), "already type") {
			return &ctypes.ResizeResponse{
				Success:    true,
				Message:    fmt.Sprintf("Instance '%s' is already type '%s'", resizeRequest.InstanceName, resizeRequest.TargetInstanceType),
				InstanceID: targetInstance.ID,
				OldType:    targetInstance.InstanceType,
				NewType:    resizeRequest.TargetInstanceType,
				Status:     "no-change",
			}, nil
		}
		return &ctypes.ResizeResponse{
			Success: false,
			Message: fmt.Sprintf("Validation failed: %v", err),
		}, nil
	}

	// Stop instance if needed
	if needsStop {
		if err := m.stopInstanceForResize(ctx, targetInstance.ID); err != nil {
			return &ctypes.ResizeResponse{
				Success: false,
				Message: err.Error(),
			}, nil
		}
	}

	// Modify instance type
	if err := m.modifyInstanceType(ctx, targetInstance.ID, resizeRequest.TargetInstanceType); err != nil {
		return &ctypes.ResizeResponse{
			Success: false,
			Message: err.Error(),
		}, nil
	}

	// Start instance if it was originally running
	if needsStop {
		if err := m.startInstanceAfterResize(ctx, targetInstance.ID, resizeRequest.Wait); err != nil {
			// Check if this is a timeout on restart
			if strings.Contains(err.Error(), "timeout") {
				return &ctypes.ResizeResponse{
					Success:    true,
					Message:    fmt.Sprintf("Instance resized successfully but timeout waiting for restart: %v", err),
					InstanceID: targetInstance.ID,
					OldType:    targetInstance.InstanceType,
					NewType:    resizeRequest.TargetInstanceType,
					Status:     "resize-complete-start-timeout",
				}, nil
			}
			return &ctypes.ResizeResponse{
				Success: false,
				Message: fmt.Sprintf("Instance resized successfully but failed to restart: %v", err),
			}, nil
		}
	}

	return &ctypes.ResizeResponse{
		Success:    true,
		Message:    fmt.Sprintf("Instance '%s' successfully resized from '%s' to '%s'", resizeRequest.InstanceName, targetInstance.InstanceType, resizeRequest.TargetInstanceType),
		InstanceID: targetInstance.ID,
		OldType:    targetInstance.InstanceType,
		NewType:    resizeRequest.TargetInstanceType,
		Status:     "resize-complete",
	}, nil
}

// ==========================================
// Instance Readiness Waiting
// ==========================================

// waitForInstanceReadyWithProgress waits for an instance to be fully ready for use with progress reporting
// This includes:
// 1. Instance reaching "running" state
// 2. SSH port (22) being accessible
func (m *Manager) waitForInstanceReadyWithProgress(instanceID, region string, progressCallback func(stage string, progress float64, description string)) error {
	ctx := context.Background()

	// Get regional EC2 client
	regionalClient := m.getRegionalEC2Client(region)

	// Step 1: Wait for instance to reach "running" state (typically 30-60 seconds)
	if progressCallback != nil {
		progressCallback("instance_ready", 0.0, "Waiting for instance to start...")
	}

	waiter := ec2.NewInstanceRunningWaiter(regionalClient)
	err := waiter.Wait(ctx, &ec2.DescribeInstancesInput{
		InstanceIds: []string{instanceID},
	}, 3*time.Minute) // Give it 3 minutes to start
	if err != nil {
		return fmt.Errorf("timeout waiting for instance to start: %w", err)
	}

	if progressCallback != nil {
		progressCallback("instance_ready", 1.0, "Instance is running")
	}

	// Step 2: Get public IP address for SSH check
	result, err := regionalClient.DescribeInstances(ctx, &ec2.DescribeInstancesInput{
		InstanceIds: []string{instanceID},
	})
	if err != nil {
		return fmt.Errorf("failed to get instance details: %w", err)
	}

	if len(result.Reservations) == 0 || len(result.Reservations[0].Instances) == 0 {
		return fmt.Errorf("instance not found")
	}

	instance := result.Reservations[0].Instances[0]
	if instance.PublicIpAddress == nil {
		return fmt.Errorf("instance has no public IP address")
	}

	publicIP := *instance.PublicIpAddress

	// Step 3: Wait for SSH to be accessible (typically 10-30 more seconds)
	if progressCallback != nil {
		progressCallback("ssh_ready", 0.0, "Waiting for SSH to be accessible...")
	}

	maxAttempts := 20
	attemptDelay := 3 * time.Second

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		// Try to connect to SSH port (don't actually authenticate, just check if port is open)
		conn, err := net.DialTimeout("tcp", publicIP+":22", 2*time.Second)
		if err == nil {
			conn.Close()
			if progressCallback != nil {
				progressCallback("ssh_ready", 1.0, "SSH is accepting connections")
			}
			return nil
		}

		// Report progress
		if progressCallback != nil {
			progress := float64(attempt) / float64(maxAttempts)
			description := fmt.Sprintf("Waiting for SSH (attempt %d/%d)...", attempt, maxAttempts)
			progressCallback("ssh_ready", progress, description)
		}

		if attempt < maxAttempts {
			time.Sleep(attemptDelay)
		}
	}

	return fmt.Errorf("timeout waiting for SSH to become accessible after %d attempts", maxAttempts)
}

// waitForInstanceReady is a wrapper that calls waitForInstanceReadyWithProgress without progress callbacks
// This maintains backward compatibility for code that doesn't use progress reporting yet
func (m *Manager) waitForInstanceReady(instanceID, region string) error {
	return m.waitForInstanceReadyWithProgress(instanceID, region, nil)
}

// ==========================================
// Instance Snapshot Management
// ==========================================

// CreateInstanceAMISnapshot creates an AMI snapshot from a CloudWorkstation instance
func (m *Manager) CreateInstanceAMISnapshot(instanceName, snapshotName, description string, noReboot bool) (*ctypes.InstanceSnapshotResult, error) {
	ctx := context.Background()

	// Load state to get instance information
	state, err := m.stateManager.LoadState()
	if err != nil {
		return nil, fmt.Errorf("failed to load state: %w", err)
	}

	instanceData, exists := state.Instances[instanceName]
	if !exists {
		return nil, fmt.Errorf("instance '%s' not found", instanceName)
	}

	if instanceData.State != "running" {
		return nil, fmt.Errorf("instance '%s' must be running to create snapshot (current state: %s)", instanceName, instanceData.State)
	}

	// Ensure unique snapshot name
	if snapshotName == "" {
		snapshotName = fmt.Sprintf("%s-snapshot-%d", instanceName, time.Now().Unix())
	}

	// Create AMI from instance
	input := &ec2.CreateImageInput{
		InstanceId:  aws.String(instanceData.ID),
		Name:        aws.String(snapshotName),
		Description: aws.String(description),
		NoReboot:    aws.Bool(noReboot),
		TagSpecifications: []ec2types.TagSpecification{
			{
				ResourceType: ec2types.ResourceTypeImage,
				Tags: []ec2types.Tag{
					{Key: aws.String("Name"), Value: aws.String(snapshotName)},
					{Key: aws.String("CloudWorkstation"), Value: aws.String("true")},
					{Key: aws.String("SourceInstance"), Value: aws.String(instanceName)},
					{Key: aws.String("SourceInstanceId"), Value: aws.String(instanceData.ID)},
					{Key: aws.String("SourceTemplate"), Value: aws.String(instanceData.Template)},
					{Key: aws.String("CreatedBy"), Value: aws.String("cloudworkstation-snapshot")},
				},
			},
		},
	}

	result, err := m.ec2.CreateImage(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to create snapshot: %w", err)
	}

	// Calculate estimated storage costs
	storageCost := m.calculateSnapshotStorageCost(instanceData)

	return &ctypes.InstanceSnapshotResult{
		SnapshotID:                 *result.ImageId,
		SnapshotName:               snapshotName,
		SourceInstance:             instanceName,
		SourceInstanceId:           instanceData.ID,
		Description:                description,
		State:                      "pending",
		EstimatedCompletionMinutes: 15, // Typical AMI creation time
		StorageCostMonthly:         storageCost,
		CreatedAt:                  time.Now(),
		NoReboot:                   noReboot,
	}, nil
}

// ListInstanceSnapshots lists all CloudWorkstation instance snapshots (AMIs)
func (m *Manager) ListInstanceSnapshots() ([]ctypes.InstanceSnapshotInfo, error) {
	ctx := context.Background()

	// List all AMIs created by CloudWorkstation
	input := &ec2.DescribeImagesInput{
		Owners: []string{"self"},
		Filters: []ec2types.Filter{
			{
				Name:   aws.String("tag:CloudWorkstation"),
				Values: []string{"true"},
			},
			{
				Name:   aws.String("tag:CreatedBy"),
				Values: []string{"cloudworkstation-snapshot"},
			},
		},
	}

	result, err := m.ec2.DescribeImages(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to list snapshots: %w", err)
	}

	var snapshots []ctypes.InstanceSnapshotInfo
	for _, image := range result.Images {
		// Extract tag values
		sourceInstance := ""
		sourceInstanceId := ""
		sourceTemplate := ""
		for _, tag := range image.Tags {
			switch *tag.Key {
			case "SourceInstance":
				sourceInstance = *tag.Value
			case "SourceInstanceId":
				sourceInstanceId = *tag.Value
			case "SourceTemplate":
				sourceTemplate = *tag.Value
			}
		}

		// Parse creation date
		var createdAt time.Time
		if image.CreationDate != nil {
			if parsed, err := time.Parse(time.RFC3339, *image.CreationDate); err == nil {
				createdAt = parsed
			}
		}

		// Calculate storage costs based on EBS snapshots associated with AMI
		storageCost := m.calculateAMIStorageCost(image)

		snapshots = append(snapshots, ctypes.InstanceSnapshotInfo{
			SnapshotID:         *image.ImageId,
			SnapshotName:       *image.Name,
			SourceInstance:     sourceInstance,
			SourceInstanceId:   sourceInstanceId,
			SourceTemplate:     sourceTemplate,
			Description:        aws.ToString(image.Description),
			State:              string(image.State),
			Architecture:       string(image.Architecture),
			StorageCostMonthly: storageCost,
			CreatedAt:          createdAt,
		})
	}

	return snapshots, nil
}

// RestoreInstanceFromSnapshot launches a new instance from a snapshot
func (m *Manager) RestoreInstanceFromSnapshot(snapshotName, newInstanceName string) (*ctypes.InstanceRestoreResult, error) {
	ctx := context.Background()

	// Find the snapshot AMI
	input := &ec2.DescribeImagesInput{
		Owners: []string{"self"},
		Filters: []ec2types.Filter{
			{
				Name:   aws.String("tag:CloudWorkstation"),
				Values: []string{"true"},
			},
			{
				Name:   aws.String("name"),
				Values: []string{snapshotName},
			},
		},
	}

	result, err := m.ec2.DescribeImages(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to find snapshot: %w", err)
	}

	if len(result.Images) == 0 {
		return nil, fmt.Errorf("snapshot '%s' not found", snapshotName)
	}

	if len(result.Images) > 1 {
		return nil, fmt.Errorf("multiple snapshots found with name '%s'", snapshotName)
	}

	image := result.Images[0]

	// Extract source template from tags
	sourceTemplate := ""
	for _, tag := range image.Tags {
		if *tag.Key == "SourceTemplate" {
			sourceTemplate = *tag.Value
			break
		}
	}

	if sourceTemplate == "" {
		return nil, fmt.Errorf("snapshot does not contain source template information")
	}

	// Create launch request using the snapshot as the AMI
	launchReq := ctypes.LaunchRequest{
		Name:      newInstanceName,
		Template:  sourceTemplate,
		CustomAMI: *image.ImageId, // Use snapshot AMI directly
	}

	// Launch the instance
	launchResult, err := m.LaunchInstance(launchReq)
	if err != nil {
		return nil, fmt.Errorf("failed to restore instance from snapshot: %w", err)
	}

	return &ctypes.InstanceRestoreResult{
		NewInstanceName: newInstanceName,
		InstanceID:      launchResult.ID,
		SnapshotName:    snapshotName,
		SnapshotID:      *image.ImageId,
		SourceTemplate:  sourceTemplate,
		State:           "pending",
		Message:         fmt.Sprintf("Instance %s launched from snapshot", newInstanceName),
		RestoredAt:      time.Now(),
	}, nil
}

// DeleteInstanceSnapshot deletes a CloudWorkstation instance snapshot (AMI)
func (m *Manager) DeleteInstanceSnapshot(snapshotName string) (*ctypes.InstanceSnapshotDeleteResult, error) {
	ctx := context.Background()

	// Find the snapshot AMI
	input := &ec2.DescribeImagesInput{
		Owners: []string{"self"},
		Filters: []ec2types.Filter{
			{
				Name:   aws.String("tag:CloudWorkstation"),
				Values: []string{"true"},
			},
			{
				Name:   aws.String("name"),
				Values: []string{snapshotName},
			},
		},
	}

	result, err := m.ec2.DescribeImages(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to find snapshot: %w", err)
	}

	if len(result.Images) == 0 {
		return nil, fmt.Errorf("snapshot '%s' not found", snapshotName)
	}

	image := result.Images[0]
	imageId := *image.ImageId

	// Calculate storage savings before deletion
	storageSavings := m.calculateAMIStorageCost(image)

	// Extract snapshot IDs from block device mappings before deletion
	var snapshotIds []string
	for _, blockDevice := range image.BlockDeviceMappings {
		if blockDevice.Ebs != nil && blockDevice.Ebs.SnapshotId != nil {
			snapshotIds = append(snapshotIds, *blockDevice.Ebs.SnapshotId)
		}
	}

	// Deregister the AMI first
	deregisterInput := &ec2.DeregisterImageInput{
		ImageId: aws.String(imageId),
	}

	_, err = m.ec2.DeregisterImage(ctx, deregisterInput)
	if err != nil {
		return nil, fmt.Errorf("failed to deregister AMI %s: %w", imageId, err)
	}

	// Delete associated EBS snapshots
	var deletedSnapshots []string
	for _, snapshotId := range snapshotIds {
		deleteSnapInput := &ec2.DeleteSnapshotInput{
			SnapshotId: aws.String(snapshotId),
		}

		_, err = m.ec2.DeleteSnapshot(ctx, deleteSnapInput)
		if err != nil {
			// Log error but continue deleting other snapshots
			fmt.Printf("Warning: failed to delete snapshot %s: %v\n", snapshotId, err)
			continue
		}

		deletedSnapshots = append(deletedSnapshots, snapshotId)
	}

	return &ctypes.InstanceSnapshotDeleteResult{
		SnapshotName:          snapshotName,
		SnapshotID:            imageId,
		DeletedSnapshots:      deletedSnapshots,
		StorageSavingsMonthly: storageSavings,
		DeletedAt:             time.Now(),
	}, nil
}

// GetInstanceSnapshotInfo gets detailed information about a specific snapshot
func (m *Manager) GetInstanceSnapshotInfo(snapshotName string) (*ctypes.InstanceSnapshotInfo, error) {
	ctx := context.Background()

	// Find the snapshot AMI
	input := &ec2.DescribeImagesInput{
		Owners: []string{"self"},
		Filters: []ec2types.Filter{
			{
				Name:   aws.String("tag:CloudWorkstation"),
				Values: []string{"true"},
			},
			{
				Name:   aws.String("name"),
				Values: []string{snapshotName},
			},
		},
	}

	result, err := m.ec2.DescribeImages(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to find snapshot: %w", err)
	}

	if len(result.Images) == 0 {
		return nil, fmt.Errorf("snapshot '%s' not found", snapshotName)
	}

	if len(result.Images) > 1 {
		return nil, fmt.Errorf("multiple snapshots found with name '%s'", snapshotName)
	}

	image := result.Images[0]

	// Extract tag values
	sourceInstance := ""
	sourceInstanceId := ""
	sourceTemplate := ""
	for _, tag := range image.Tags {
		switch *tag.Key {
		case "SourceInstance":
			sourceInstance = *tag.Value
		case "SourceInstanceId":
			sourceInstanceId = *tag.Value
		case "SourceTemplate":
			sourceTemplate = *tag.Value
		}
	}

	// Parse creation date
	var createdAt time.Time
	if image.CreationDate != nil {
		if parsed, err := time.Parse(time.RFC3339, *image.CreationDate); err == nil {
			createdAt = parsed
		}
	}

	// Calculate storage costs
	storageCost := m.calculateAMIStorageCost(image)

	// Get associated snapshots
	var associatedSnapshots []string
	for _, blockDevice := range image.BlockDeviceMappings {
		if blockDevice.Ebs != nil && blockDevice.Ebs.SnapshotId != nil {
			associatedSnapshots = append(associatedSnapshots, *blockDevice.Ebs.SnapshotId)
		}
	}

	return &ctypes.InstanceSnapshotInfo{
		SnapshotID:          *image.ImageId,
		SnapshotName:        *image.Name,
		SourceInstance:      sourceInstance,
		SourceInstanceId:    sourceInstanceId,
		SourceTemplate:      sourceTemplate,
		Description:         aws.ToString(image.Description),
		State:               string(image.State),
		Architecture:        string(image.Architecture),
		StorageCostMonthly:  storageCost,
		CreatedAt:           createdAt,
		AssociatedSnapshots: associatedSnapshots,
	}, nil
}

// Helper methods for snapshot cost calculation

func (m *Manager) calculateSnapshotStorageCost(instance ctypes.Instance) float64 {
	// Estimate EBS snapshot costs
	// Root volume: typically 20GB @ $0.05/GB/month
	// Additional volumes: based on instance template
	rootVolumeCost := 20.0 * 0.05 // 20GB root volume

	// Add costs for attached volumes
	additionalVolumeCost := 0.0
	for range instance.AttachedEBSVolumes {
		// Simplified estimation - would need to look up actual volume sizes
		additionalVolumeCost += 100.0 * 0.05 // Estimate 100GB per volume
	}

	return rootVolumeCost + additionalVolumeCost
}

func (m *Manager) calculateAMIStorageCost(image ec2types.Image) float64 {
	totalCost := 0.0

	// Simplified cost calculation based on typical EBS volumes
	// In a full implementation, this would query actual snapshot sizes
	for _, blockDevice := range image.BlockDeviceMappings {
		if blockDevice.Ebs != nil {
			// Estimate cost based on typical volume sizes
			if blockDevice.DeviceName != nil && *blockDevice.DeviceName == "/dev/sda1" {
				// Root volume - typically 20GB
				totalCost += 20.0 * 0.05 // $0.05 per GB per month
			} else {
				// Additional volume - estimate 100GB
				totalCost += 100.0 * 0.05 // $0.05 per GB per month
			}
		}
	}

	// Minimum cost if no block devices found
	if totalCost == 0.0 {
		totalCost = 20.0 * 0.05 // Default to 20GB root volume
	}

	return totalCost
}

// GetIdleScheduler returns the idle scheduler for direct access
func (m *Manager) GetIdleScheduler() *idle.Scheduler {
	return m.idleScheduler
}

// GetPolicyManager returns the policy manager for direct access
func (m *Manager) GetPolicyManager() *idle.PolicyManager {
	return m.policyManager
}

// GetAWSConfig returns the AWS config for creating additional service clients
func (m *Manager) GetAWSConfig() aws.Config {
	return m.cfg
}

// CheckAMIFreshness validates static AMI IDs against latest SSM values (v0.5.4)
func (m *Manager) CheckAMIFreshness(ctx context.Context, staticAMIs map[string]map[string]map[string]map[string]string) ([]AMIFreshnessResult, error) {
	return m.amiDiscovery.CheckAMIFreshness(ctx, staticAMIs, m.region)
}
