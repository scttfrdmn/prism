// Package aws provides AMI integration methods for the Universal AMI System
package aws

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	ec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/scttfrdmn/prism/pkg/templates"
	"github.com/scttfrdmn/prism/pkg/types"
)

// ResolveAMIForTemplate resolves AMI information for a template using the Universal AMI System
func (m *Manager) ResolveAMIForTemplate(templateName string) (*types.AMIResolutionResult, error) {
	ctx := context.Background()

	// Get the unified template
	rawTemplate, err := templates.GetTemplateInfo(templateName)
	if err != nil {
		return nil, fmt.Errorf("failed to get template info: %w", err)
	}

	// Use AMI resolver
	result, err := m.amiResolver.ResolveAMI(ctx, rawTemplate, m.region)
	if err != nil {
		return nil, fmt.Errorf("AMI resolution failed: %w", err)
	}

	return result, nil
}

// TestAMIAvailability tests AMI availability for a template across regions
func (m *Manager) TestAMIAvailability(templateName string, regions []string) (*types.AMITestResult, error) {
	ctx := context.Background()

	// Get the template
	rawTemplate, err := templates.GetTemplateInfo(templateName)
	if err != nil {
		return nil, fmt.Errorf("failed to get template info: %w", err)
	}

	// If no regions specified, test all common regions
	if len(regions) == 0 {
		regions = []string{"us-east-1", "us-west-2", "eu-west-1", "ap-south-1"}
	}

	result := &types.AMITestResult{
		TemplateName:     templateName,
		RegionResults:    make(map[string]*types.RegionTestResult),
		TotalRegions:     len(regions),
		AvailableRegions: 0,
	}

	// Test each region
	for _, region := range regions {
		regionResult := &types.RegionTestResult{
			Region:       region,
			TestDuration: 0, // Will be calculated
		}

		// Resolve AMI for this region
		amiResult, err := m.amiResolver.ResolveAMI(ctx, rawTemplate, region)
		if err != nil {
			regionResult.Status = types.AMITestStatusFailed
			regionResult.Error = err.Error()
		} else {
			if amiResult.ResolutionMethod == types.ResolutionFailed {
				regionResult.Status = types.AMITestStatusFailed
				regionResult.Error = "No AMI resolution method succeeded"
			} else {
				regionResult.Status = types.AMITestStatusPassed
				regionResult.AMI = amiResult.AMI
				regionResult.ResolutionMethod = amiResult.ResolutionMethod
				result.AvailableRegions++
			}
		}

		result.RegionResults[region] = regionResult
	}

	// Determine overall status
	if result.AvailableRegions == 0 {
		result.OverallStatus = types.AMITestStatusFailed
	} else if result.AvailableRegions == len(regions) {
		result.OverallStatus = types.AMITestStatusPassed
	} else {
		result.OverallStatus = types.AMITestStatusPartial
	}

	return result, nil
}

// GetAMICostAnalysis provides cost analysis for AMI vs script deployment
func (m *Manager) GetAMICostAnalysis(templateName string) (*types.AMICostAnalysis, error) {
	// Get template information
	rawTemplate, err := templates.GetTemplateInfo(templateName)
	if err != nil {
		return nil, fmt.Errorf("failed to get template info: %w", err)
	}

	// Resolve AMI information
	ctx := context.Background()
	amiResult, err := m.amiResolver.ResolveAMI(ctx, rawTemplate, m.region)
	if err != nil {
		// If AMI resolution fails, compare against script-only
		analysis := &types.AMICostAnalysis{
			TemplateName:    templateName,
			Region:          m.region,
			Recommendation:  "script_recommended",
			Reasoning:       "AMI not available, script provisioning is the only option",
			ScriptSetupTime: int(m.amiResolver.estimateScriptProvisioningTime(rawTemplate).Minutes()),
		}
		return analysis, nil
	}

	// Use cost analyzer to generate analysis
	scriptTime := m.amiResolver.estimateScriptProvisioningTime(rawTemplate)
	analysis := m.amiResolver.costAnalyzer.AnalyzeCosts(templateName, m.region, amiResult.AMI, scriptTime)

	return analysis, nil
}

// PreviewAMIResolution shows what would happen during AMI resolution without actually resolving
func (m *Manager) PreviewAMIResolution(templateName string) (*types.AMIResolutionResult, error) {
	// This is a dry-run version that shows the resolution strategy without executing it
	rawTemplate, err := templates.GetTemplateInfo(templateName)
	if err != nil {
		return nil, fmt.Errorf("failed to get template info: %w", err)
	}

	result := &types.AMIResolutionResult{
		TargetRegion:  m.region,
		FallbackChain: make([]string, 0),
	}

	// Check what resolution methods are available
	if rawTemplate.AMIConfig.AMIMappings != nil {
		if _, exists := rawTemplate.AMIConfig.AMIMappings[m.region]; exists {
			result.FallbackChain = append(result.FallbackChain, "direct_mapping_available")
			result.ResolutionMethod = types.ResolutionDirectMapping
			result.LaunchTime = 30_000_000_000 // 30 seconds in nanoseconds
			return result, nil
		}
	}

	if rawTemplate.AMIConfig.AMISearch != nil {
		result.FallbackChain = append(result.FallbackChain, "dynamic_search_available")
		result.ResolutionMethod = types.ResolutionDynamicSearch
		result.LaunchTime = 45_000_000_000 // 45 seconds
		return result, nil
	}

	if rawTemplate.AMIConfig.MarketplaceSearch != nil {
		result.FallbackChain = append(result.FallbackChain, "marketplace_search_available")
		result.ResolutionMethod = types.ResolutionMarketplace
		result.LaunchTime = 60_000_000_000 // 60 seconds
		return result, nil
	}

	// Check if script fallback is available
	if rawTemplate.PackageManager != "" && rawTemplate.PackageManager != "ami" {
		result.ResolutionMethod = types.ResolutionFallbackScript
		result.LaunchTime = m.amiResolver.estimateScriptProvisioningTime(rawTemplate)
		result.Warning = "No AMI available, would use script provisioning"
		return result, nil
	}

	result.ResolutionMethod = types.ResolutionFailed
	result.FallbackChain = append(result.FallbackChain, "no_fallback_available")
	return result, fmt.Errorf("no resolution method available for template %s", templateName)
}

// LaunchInstanceWithAMI launches an instance using AMI resolution
func (m *Manager) LaunchInstanceWithAMI(req types.LaunchRequest) (*types.Instance, error) {
	// This method integrates AMI resolution with the existing launch process
	ctx := context.Background()

	// Get template information
	rawTemplate, err := templates.GetTemplateInfo(req.Template)
	if err != nil {
		return nil, fmt.Errorf("failed to get template info: %w", err)
	}

	// Resolve AMI for the template
	amiResult, err := m.amiResolver.ResolveAMI(ctx, rawTemplate, m.region)
	if err != nil {
		return nil, fmt.Errorf("AMI resolution failed: %w", err)
	}

	// Store AMI resolution info for launch process
	req.AMIResolutionResult = amiResult

	// ARCHITECTURE FIX: Determine instance type first, then get its architecture
	// Do NOT use local machine architecture for cloud instance selection
	var instanceType string
	if req.Size != "" {
		instanceType = m.getInstanceTypeForSize(req.Size)
	} else if rawTemplate.InstanceDefaults.Type != "" {
		instanceType = rawTemplate.InstanceDefaults.Type
	} else {
		instanceType = "t3.micro"
	}

	// Query AWS for instance type's architecture
	arch, err := m.getInstanceTypeArchitecture(instanceType)
	if err != nil {
		log.Printf("Warning: Could not determine architecture for %s, using x86_64", instanceType)
		arch = "x86_64"
	}

	// Allow template to override if explicitly specified
	if rawTemplate.AMIConfig.PreferredArchitecture != "" {
		arch = rawTemplate.AMIConfig.PreferredArchitecture
	}

	// Based on resolution method, modify launch approach
	switch amiResult.ResolutionMethod {
	case types.ResolutionDirectMapping, types.ResolutionDynamicSearch,
		types.ResolutionMarketplace, types.ResolutionCrossRegion:
		// AMI-based launch
		return m.launchWithAMI(req, amiResult, arch)

	case types.ResolutionFallbackScript:
		// Script-based launch (existing flow)
		return m.launchWithUnifiedTemplateSystem(req, arch)

	default:
		return nil, fmt.Errorf("unsupported AMI resolution method: %s", amiResult.ResolutionMethod)
	}
}

// launchWithAMI launches an instance using resolved AMI information
func (m *Manager) launchWithAMI(req types.LaunchRequest, amiResult *types.AMIResolutionResult, arch string) (*types.Instance, error) {
	// This would implement AMI-based instance launching
	// For now, we'll integrate with the existing template system by modifying the template

	// Get the runtime template but override AMI information
	packageManager := req.PackageManager
	if packageManager == "" {
		packageManager = ""
	}

	template, err := templates.GetTemplateWithPackageManager(req.Template, m.region, arch, packageManager, req.Size)
	if err != nil {
		return nil, fmt.Errorf("failed to get template: %w", err)
	}

	// Override template AMI with resolved AMI
	if amiResult.AMI != nil {
		template.AMI[m.region] = map[string]string{
			arch: amiResult.AMI.AMIID,
		}
	}

	// Use existing launch orchestration
	return m.launchWithResolvedTemplate(req, template, amiResult)
}

// launchWithResolvedTemplate launches an instance with resolved template and AMI info
func (m *Manager) launchWithResolvedTemplate(req types.LaunchRequest, template *types.RuntimeTemplate, amiResult *types.AMIResolutionResult) (*types.Instance, error) {
	// This integrates with existing launch orchestration
	// We'll use the existing launch system but track AMI resolution metrics

	// Launch using existing orchestration
	instance, err := m.launchWithTemplate(req, template)
	if err != nil {
		return nil, err
	}

	// Enhance instance with AMI resolution information
	if instance != nil && amiResult != nil {
		// Store AMI resolution information in instance for tracking
		instance.AMIResolutionMethod = string(amiResult.ResolutionMethod)
		instance.BootTime = amiResult.LaunchTime
		instance.CostSavings = amiResult.CostSavings

		// Add warning if applicable
		if amiResult.Warning != "" {
			// In a production system, this would be logged or stored for user notification
		}
	}

	return instance, nil
}

// Helper method to launch with template (placeholder for existing integration)
func (m *Manager) launchWithTemplate(req types.LaunchRequest, template *types.RuntimeTemplate) (*types.Instance, error) {
	// ARCHITECTURE FIX: Determine instance type architecture properly
	// Get template info to determine instance type
	rawTemplate, err := templates.GetTemplateInfo(req.Template)
	if err != nil {
		// Fallback to old behavior if we can't get template info
		log.Printf("Warning: Could not get template info for architecture determination: %v", err)
		return m.launchWithUnifiedTemplateSystem(req, "x86_64")
	}

	var instanceType string
	if req.Size != "" {
		instanceType = m.getInstanceTypeForSize(req.Size)
	} else if rawTemplate.InstanceDefaults.Type != "" {
		instanceType = rawTemplate.InstanceDefaults.Type
	} else {
		instanceType = "t3.micro"
	}

	arch, err := m.getInstanceTypeArchitecture(instanceType)
	if err != nil {
		log.Printf("Warning: Could not determine architecture for %s, using x86_64", instanceType)
		arch = "x86_64"
	}

	return m.launchWithUnifiedTemplateSystem(req, arch)
}

// AMI Creation Methods (Phase 5.1 Enhancement)

// CreateAMIFromInstance creates an AMI from a running instance
func (m *Manager) CreateAMIFromInstance(request *types.AMICreationRequest) (*types.AMICreationResult, error) {
	ctx := context.Background()

	// Validate instance exists in our state
	instances, err := m.ListInstances()
	if err != nil {
		return nil, fmt.Errorf("failed to list instances: %w", err)
	}

	// Find the instance to create AMI from
	var sourceInstance *types.Instance
	for _, instance := range instances {
		if instance.ID == request.InstanceID || instance.Name == request.InstanceID {
			sourceInstance = &instance
			break
		}
	}

	if sourceInstance == nil {
		return nil, fmt.Errorf("instance not found: %s", request.InstanceID)
	}

	// Ensure instance is running
	if sourceInstance.State != "running" {
		return nil, fmt.Errorf("instance must be running to create AMI, current state: %s", sourceInstance.State)
	}

	// Set instance ID to AWS instance ID for AMI creation
	request.InstanceID = sourceInstance.ID

	// Use the AMI resolver to create the AMI
	result, err := m.amiResolver.CreateAMIFromInstance(ctx, request)
	if err != nil {
		return nil, fmt.Errorf("AMI creation failed: %w", err)
	}

	return result, nil
}

// GetAMICreationStatus checks the status of AMI creation
func (m *Manager) GetAMICreationStatus(creationID string) (*types.AMICreationResult, error) {
	ctx := context.Background()

	// Extract AMI ID from creation ID (format: ami-creation-{template}-{timestamp} or direct ami-id)
	amiID := creationID
	if strings.HasPrefix(creationID, "ami-creation-") {
		// For now, simulate AMI ID extraction from creation ID
		// In production, this would be stored in a creation tracking system
		amiID = fmt.Sprintf("ami-%s", strings.Split(creationID, "-")[2])
	}

	// Query AMI status
	result, err := m.amiResolver.GetAMICreationStatus(ctx, amiID)
	if err != nil {
		return nil, fmt.Errorf("failed to get AMI creation status: %w", err)
	}

	return result, nil
}

// ListUserAMIs lists AMIs created by the user
func (m *Manager) ListUserAMIs() ([]*types.AMIInfo, error) {
	ctx := context.Background()

	// Use EC2 DescribeImages API with Owner=self filter and Prism tags
	input := &ec2.DescribeImagesInput{
		Owners: []string{"self"},
		Filters: []ec2types.Filter{
			{
				Name:   aws.String("tag:Prism"),
				Values: []string{"true"},
			},
		},
	}

	result, err := m.ec2.DescribeImages(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to describe images: %w", err)
	}

	// Convert EC2 images to AMIInfo
	userAMIs := make([]*types.AMIInfo, 0, len(result.Images))
	for _, image := range result.Images {
		if image.ImageId == nil {
			continue
		}

		// Extract tags into map
		tags := make(map[string]string)
		for _, tag := range image.Tags {
			if tag.Key != nil && tag.Value != nil {
				tags[*tag.Key] = *tag.Value
			}
		}

		// Parse creation date
		var creationDate time.Time
		if image.CreationDate != nil {
			creationDate, _ = time.Parse(time.RFC3339, *image.CreationDate)
		}

		amiInfo := &types.AMIInfo{
			AMIID:        *image.ImageId,
			Name:         aws.ToString(image.Name),
			Description:  aws.ToString(image.Description),
			Architecture: string(image.Architecture),
			Owner:        aws.ToString(image.OwnerId),
			CreationDate: creationDate,
			Public:       aws.ToBool(image.Public),
			Tags:         tags,
		}

		userAMIs = append(userAMIs, amiInfo)
	}

	return userAMIs, nil
}

// PublishAMIToCommunity makes an AMI available to the community
func (m *Manager) PublishAMIToCommunity(amiID string, public bool, tags map[string]string) error {
	ctx := context.Background()

	// Validate AMI exists and is owned by user
	if !strings.HasPrefix(amiID, "ami-") {
		return fmt.Errorf("invalid AMI ID format: %s", amiID)
	}

	// 1. Update AMI permissions to make it public if requested
	if public {
		launchPermissionInput := &ec2.ModifyImageAttributeInput{
			ImageId: aws.String(amiID),
			LaunchPermission: &ec2types.LaunchPermissionModifications{
				Add: []ec2types.LaunchPermission{
					{
						Group: ec2types.PermissionGroupAll,
					},
				},
			},
		}

		_, err := m.ec2.ModifyImageAttribute(ctx, launchPermissionInput)
		if err != nil {
			return fmt.Errorf("failed to make AMI public: %w", err)
		}
	}

	// 2. Add community tags for discoverability
	if tags != nil && len(tags) > 0 {
		// Add Prism community tag
		tags["Prism-Community"] = "published"

		// Convert tags to EC2 tag format
		ec2Tags := make([]ec2types.Tag, 0, len(tags))
		for key, value := range tags {
			ec2Tags = append(ec2Tags, ec2types.Tag{
				Key:   aws.String(key),
				Value: aws.String(value),
			})
		}

		createTagsInput := &ec2.CreateTagsInput{
			Resources: []string{amiID},
			Tags:      ec2Tags,
		}

		_, err := m.ec2.CreateTags(ctx, createTagsInput)
		if err != nil {
			return fmt.Errorf("failed to add community tags: %w", err)
		}
	}

	// 3. Log publication (marketplace integration would happen here in full implementation)
	log.Printf("Published AMI %s to community (public: %v) with %d tags", amiID, public, len(tags))

	return nil
}

// AMI Lifecycle Management Methods

// CleanupOldAMIs removes old and unused AMIs
func (m *Manager) CleanupOldAMIs(maxAge string, dryRun bool) (*types.AMICleanupResult, error) {
	// 1. Parse maxAge duration (e.g., "30d", "7d", "1y")
	duration, err := parseDuration(maxAge)
	if err != nil {
		return nil, fmt.Errorf("invalid maxAge format: %w", err)
	}

	cutoffTime := time.Now().Add(-duration)

	// 2. Query EC2 for all user AMIs with Prism tags
	userAMIs, err := m.ListUserAMIs()
	if err != nil {
		return nil, fmt.Errorf("failed to list user AMIs: %w", err)
	}

	// 3. Get all running instances to check which AMIs are in use
	instances, err := m.ListInstances()
	if err != nil {
		return nil, fmt.Errorf("failed to list instances: %w", err)
	}

	// Build set of AMIs currently in use
	amisInUse := make(map[string]bool)
	for _, instance := range instances {
		// Get AMI from instance (would need to query EC2 for actual AMI ID)
		// For now, assume we can't delete if any instances exist
		if instance.State == "running" || instance.State == "stopped" {
			amisInUse[instance.ID] = true
		}
	}

	// 4. Find old AMIs not in use
	var oldAMIs []types.AMIInfo
	var totalSavings float64

	for _, ami := range userAMIs {
		// Check if AMI is older than cutoff
		if ami.CreationDate.Before(cutoffTime) {
			// Check if not in use (conservative: skip if any instances exist)
			if len(amisInUse) == 0 {
				oldAMIs = append(oldAMIs, *ami)

				// Calculate storage cost (estimate $0.05/GB-month, assume 50GB per AMI)
				totalSavings += 2.50 // $0.05 * 50GB
			}
		}
	}

	result := &types.AMICleanupResult{
		TotalFound:            len(oldAMIs),
		TotalRemoved:          0,
		StorageSavingsMonthly: 0,
		CompletedAt:           time.Now(),
		RemovedAMIs:           []types.AMIInfo{},
	}

	// 5. Delete AMIs if not dry run
	if !dryRun && len(oldAMIs) > 0 {
		for _, ami := range oldAMIs {
			// Delete the AMI using our DeleteAMI method
			deleteResult, err := m.DeleteAMI(ami.AMIID, false)
			if err != nil {
				log.Printf("Warning: failed to delete AMI %s: %v", ami.AMIID, err)
				continue
			}

			result.TotalRemoved++
			result.StorageSavingsMonthly += deleteResult.StorageSavingsMonthly
			result.RemovedAMIs = append(result.RemovedAMIs, ami)
		}
	} else if dryRun {
		result.StorageSavingsMonthly = totalSavings
	}

	return result, nil
}

// parseDuration parses duration strings like "30d", "7d", "1y"
func parseDuration(s string) (time.Duration, error) {
	if len(s) < 2 {
		return 0, fmt.Errorf("invalid duration format")
	}

	valueStr := s[:len(s)-1]
	unit := s[len(s)-1]

	value := 0
	_, err := fmt.Sscanf(valueStr, "%d", &value)
	if err != nil {
		return 0, err
	}

	switch unit {
	case 'd':
		return time.Duration(value) * 24 * time.Hour, nil
	case 'w':
		return time.Duration(value) * 7 * 24 * time.Hour, nil
	case 'm':
		return time.Duration(value) * 30 * 24 * time.Hour, nil
	case 'y':
		return time.Duration(value) * 365 * 24 * time.Hour, nil
	default:
		return 0, fmt.Errorf("unsupported duration unit: %c (use d, w, m, y)", unit)
	}
}

// DeleteAMI deletes a specific AMI by ID
func (m *Manager) DeleteAMI(amiID string, deregisterOnly bool) (*types.AMIDeletionResult, error) {
	ctx := context.Background()

	// 1. Validate AMI exists and get its details
	describeInput := &ec2.DescribeImagesInput{
		ImageIds: []string{amiID},
	}

	describeResult, err := m.ec2.DescribeImages(ctx, describeInput)
	if err != nil {
		return nil, fmt.Errorf("failed to describe AMI: %w", err)
	}

	if len(describeResult.Images) == 0 {
		return nil, fmt.Errorf("AMI %s not found", amiID)
	}

	image := describeResult.Images[0]

	// Calculate storage cost before deletion
	storageCost := m.calculateAMIStorageCost(image)

	// Extract snapshot IDs before deregistration
	var snapshotIds []string
	for _, blockDevice := range image.BlockDeviceMappings {
		if blockDevice.Ebs != nil && blockDevice.Ebs.SnapshotId != nil {
			snapshotIds = append(snapshotIds, *blockDevice.Ebs.SnapshotId)
		}
	}

	// 3. Deregister the AMI
	deregisterInput := &ec2.DeregisterImageInput{
		ImageId: aws.String(amiID),
	}

	_, err = m.ec2.DeregisterImage(ctx, deregisterInput)
	if err != nil {
		return nil, fmt.Errorf("failed to deregister AMI: %w", err)
	}

	result := &types.AMIDeletionResult{
		AMIID:                 amiID,
		Status:                "deregistered",
		StorageSavingsMonthly: storageCost,
		CompletedAt:           time.Now(),
		DeletedSnapshots:      []string{},
	}

	// 4. Optionally delete associated EBS snapshots
	if !deregisterOnly {
		for _, snapshotId := range snapshotIds {
			deleteSnapInput := &ec2.DeleteSnapshotInput{
				SnapshotId: aws.String(snapshotId),
			}

			_, err = m.ec2.DeleteSnapshot(ctx, deleteSnapInput)
			if err != nil {
				log.Printf("Warning: failed to delete snapshot %s: %v", snapshotId, err)
				continue
			}

			result.DeletedSnapshots = append(result.DeletedSnapshots, snapshotId)
		}

		if len(result.DeletedSnapshots) > 0 {
			result.Status = "deleted"
		}
	}

	return result, nil
}

// AMI Snapshot Management Methods

// ListAMISnapshots lists available snapshots
func (m *Manager) ListAMISnapshots(instanceID, maxAge string) ([]types.SnapshotInfo, error) {
	// In production, this would:
	// 1. Query EC2 DescribeSnapshots API
	// 2. Filter by Prism tags and optional instanceID
	// 3. Apply maxAge filter if specified
	// 4. Calculate storage costs for each snapshot

	// Placeholder implementation
	snapshots := []types.SnapshotInfo{
		{
			SnapshotID:         "snap-0123456789abcdef0",
			VolumeID:           "vol-0123456789abcdef0",
			VolumeSize:         20,
			Description:        "Root volume snapshot for python-ml instance",
			StartTime:          time.Now().Add(-2 * time.Hour),
			State:              "completed",
			Progress:           "100%",
			StorageCostMonthly: 1.00, // $0.05 per GB-month * 20 GB
		},
		{
			SnapshotID:         "snap-0987654321fedcba0",
			VolumeID:           "vol-0987654321fedcba0",
			VolumeSize:         100,
			Description:        "Data volume snapshot for research project",
			StartTime:          time.Now().Add(-24 * time.Hour),
			State:              "completed",
			Progress:           "100%",
			StorageCostMonthly: 5.00, // $0.05 per GB-month * 100 GB
		},
	}

	// Apply instanceID filter if specified
	if instanceID != "" {
		filtered := make([]types.SnapshotInfo, 0)
		for _, snapshot := range snapshots {
			// In production, would check if snapshot was created from the specified instance
			if strings.Contains(snapshot.Description, instanceID) {
				filtered = append(filtered, snapshot)
			}
		}
		snapshots = filtered
	}

	return snapshots, nil
}

// CreateInstanceSnapshot creates a snapshot from an instance
func (m *Manager) CreateInstanceSnapshot(instanceID, description string, noReboot bool) (*types.SnapshotCreationResult, error) {
	// In production, this would:
	// 1. Validate instance exists and user owns it
	// 2. Get instance root volume ID
	// 3. Optionally stop instance if noReboot is false
	// 4. Create EBS snapshot with proper tags
	// 5. Start instance if it was stopped
	// 6. Calculate estimated completion time and cost

	// Placeholder implementation
	result := &types.SnapshotCreationResult{
		SnapshotID:                 "snap-new" + instanceID[2:],
		VolumeID:                   "vol-" + instanceID[2:],
		VolumeSize:                 20,
		Description:                description,
		EstimatedCompletionMinutes: 15,   // Typical snapshot creation time
		StorageCostMonthly:         1.00, // $0.05 per GB-month * 20 GB
		CreationInitiatedAt:        time.Now(),
	}

	return result, nil
}

// RestoreAMIFromSnapshot creates an AMI from a snapshot
func (m *Manager) RestoreAMIFromSnapshot(snapshotID, name, description, architecture string) (*types.AMIRestoreResult, error) {
	// In production, this would:
	// 1. Validate snapshot exists and user owns it
	// 2. Create AMI from the snapshot using RegisterImage API
	// 3. Add proper Prism tags
	// 4. Return AMI creation details

	// Placeholder implementation
	result := &types.AMIRestoreResult{
		AMIID:                      "ami-restored" + snapshotID[5:],
		Name:                       name,
		Description:                description,
		Architecture:               architecture,
		EstimatedCompletionMinutes: 8, // AMI registration is typically faster than creation
		RestoreInitiatedAt:         time.Now(),
	}

	return result, nil
}

// DeleteSnapshot deletes a specific snapshot
func (m *Manager) DeleteSnapshot(snapshotID string) (*types.SnapshotDeletionResult, error) {
	// In production, this would:
	// 1. Validate snapshot exists and user owns it
	// 2. Check if snapshot is used by any AMIs
	// 3. Delete snapshot using DeleteSnapshot API
	// 4. Calculate storage cost savings

	// Placeholder implementation
	result := &types.SnapshotDeletionResult{
		SnapshotID:            snapshotID,
		VolumeSize:            20,
		StorageSavingsMonthly: 1.00, // $0.05 per GB-month * 20 GB
		CompletedAt:           time.Now(),
	}

	return result, nil
}
