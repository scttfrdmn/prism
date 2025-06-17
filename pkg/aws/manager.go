package aws

import (
	"context"
	"encoding/base64"
	"fmt"
	"runtime"
	"strings"
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
	cfg         aws.Config
	ec2         *ec2.Client
	efs         *efs.Client
	sts         *sts.Client
	region      string
	templates   map[string]ctypes.Template
	pricingCache map[string]float64
	lastPriceUpdate time.Time
	discountConfig ctypes.DiscountConfig
}

// NewManager creates a new AWS manager
func NewManager() (*Manager, error) {
	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	manager := &Manager{
		cfg:         cfg,
		ec2:         ec2.NewFromConfig(cfg),
		efs:         efs.NewFromConfig(cfg),
		sts:         sts.NewFromConfig(cfg),
		region:      cfg.Region,
		templates:   getTemplates(),
		pricingCache: make(map[string]float64),
		lastPriceUpdate: time.Time{},
		discountConfig: ctypes.DiscountConfig{}, // No discounts by default
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
	ami, instanceType, _, err := m.getTemplateForArchitecture(template, arch, m.region)
	if err != nil {
		return nil, fmt.Errorf("failed to get template configuration: %w", err)
	}

	// Get regional pricing for instance
	regionalCostPerHour := m.getRegionalEC2Price(instanceType)
	dailyCost := regionalCostPerHour * 24

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
		EstimatedCostGB: m.getRegionalEFSPrice(), // Regional EFS pricing
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
		Size:         aws.Int32(int32(sizeGB)),
		VolumeType:   types.VolumeType(volumeType),
		AvailabilityZone: aws.String(m.region + "a"), // Use first AZ
		TagSpecifications: []types.TagSpecification{
			{
				ResourceType: types.ResourceTypeVolume,
				Tags: []types.Tag{
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
	
	result, err := m.ec2.CreateVolume(context.TODO(), input)
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
	
	// Delete the volume
	_, err = m.ec2.DeleteVolume(context.TODO(), &ec2.DeleteVolumeInput{
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
	
	// Attach volume to instance
	_, err = m.ec2.AttachVolume(context.TODO(), &ec2.AttachVolumeInput{
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
	
	// Detach the volume
	_, err = m.ec2.DetachVolume(context.TODO(), &ec2.DetachVolumeInput{
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
	result, err := m.ec2.DescribeVolumes(context.TODO(), &ec2.DescribeVolumesInput{
		Filters: []types.Filter{
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