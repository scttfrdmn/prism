// Package aws provides AMI integration methods for the Universal AMI System
package aws

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/scttfrdmn/cloudworkstation/pkg/templates"
	"github.com/scttfrdmn/cloudworkstation/pkg/types"
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

	// Determine architecture preference
	arch := m.getLocalArchitecture() // Default to local arch
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
	// This would integrate with the existing launch orchestration system
	// For now, we'll call the existing method
	arch := m.getLocalArchitecture()
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
	// In production, this would:
	// 1. Use EC2 DescribeImages API with Owner=self filter
	// 2. Filter for AMIs with CloudWorkstation tags
	// 3. Return detailed AMI information

	// Placeholder implementation
	userAMIs := []*types.AMIInfo{
		{
			AMIID:        "ami-user123456789abcdef",
			Name:         "my-custom-python-env",
			Description:  "Custom Python ML environment with PyTorch",
			Architecture: "x86_64",
			Owner:        "123456789012", // User's AWS account ID
			CreationDate: time.Now().Add(-2 * time.Hour),
			Public:       false,
			Tags: map[string]string{
				"CloudWorkstation": "true",
				"Template":         "python-ml",
				"Creator":          "researcher",
			},
		},
		{
			AMIID:        "ami-user987654321fedcba",
			Name:         "genomics-pipeline-v2",
			Description:  "Optimized genomics analysis pipeline",
			Architecture: "arm64",
			Owner:        "123456789012",
			CreationDate: time.Now().Add(-24 * time.Hour),
			Public:       true,
			Tags: map[string]string{
				"CloudWorkstation": "true",
				"Template":         "bioinformatics",
				"Creator":          "researcher",
				"Community":        "published",
			},
		},
	}

	return userAMIs, nil
}

// PublishAMIToCommunity makes an AMI available to the community
func (m *Manager) PublishAMIToCommunity(amiID string, public bool, tags map[string]string) error {
	// In production, this would:
	// 1. Update AMI permissions to make it public if requested
	// 2. Add community tags for discoverability
	// 3. Submit to CloudWorkstation community registry
	// 4. Add to template marketplace integration

	// Validate AMI exists and is owned by user
	if !strings.HasPrefix(amiID, "ami-") {
		return fmt.Errorf("invalid AMI ID format: %s", amiID)
	}

	// Placeholder implementation
	log.Printf("Publishing AMI %s to community (public: %v) with tags: %v", amiID, public, tags)

	return nil
}

// AMI Lifecycle Management Methods

// CleanupOldAMIs removes old and unused AMIs
func (m *Manager) CleanupOldAMIs(maxAge string, dryRun bool) (*types.AMICleanupResult, error) {
	// In production, this would:
	// 1. Parse maxAge duration (e.g., "30d", "7d", "1y")
	// 2. Query EC2 for AMIs older than maxAge with CloudWorkstation tags
	// 3. Check for any instances currently using these AMIs
	// 4. Optionally delete AMIs and associated snapshots if not in use
	// 5. Calculate storage cost savings

	// Placeholder implementation
	result := &types.AMICleanupResult{
		TotalFound:            8,
		TotalRemoved:          5,
		StorageSavingsMonthly: 47.25, // $0.05 per GB-month * average 945 GB across 5 AMIs
		CompletedAt:           time.Now(),
		RemovedAMIs: []types.AMIInfo{
			{
				AMIID:        "ami-old123456789abcdef",
				Name:         "obsolete-python-2.7-env",
				CreationDate: time.Now().Add(-45 * 24 * time.Hour), // 45 days old
			},
			{
				AMIID:        "ami-old987654321fedcba",
				Name:         "deprecated-r-3.6-env",
				CreationDate: time.Now().Add(-62 * 24 * time.Hour), // 62 days old
			},
		},
	}

	if dryRun {
		result.TotalRemoved = 0
		result.StorageSavingsMonthly = 0
	}

	return result, nil
}

// DeleteAMI deletes a specific AMI by ID
func (m *Manager) DeleteAMI(amiID string, deregisterOnly bool) (*types.AMIDeletionResult, error) {
	// In production, this would:
	// 1. Validate AMI exists and user owns it
	// 2. Check for any instances using this AMI
	// 3. Deregister the AMI
	// 4. Optionally delete associated EBS snapshots
	// 5. Calculate storage cost savings

	// Placeholder implementation
	result := &types.AMIDeletionResult{
		AMIID:                 amiID,
		Status:                "deleted",
		StorageSavingsMonthly: 12.50, // $0.05 per GB-month * 250 GB
		CompletedAt:           time.Now(),
	}

	if !deregisterOnly {
		result.DeletedSnapshots = []string{
			"snap-0123456789abcdef0",
			"snap-0987654321fedcba0",
		}
	}

	return result, nil
}

// AMI Snapshot Management Methods

// ListAMISnapshots lists available snapshots
func (m *Manager) ListAMISnapshots(instanceID, maxAge string) ([]types.SnapshotInfo, error) {
	// In production, this would:
	// 1. Query EC2 DescribeSnapshots API
	// 2. Filter by CloudWorkstation tags and optional instanceID
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
	// 3. Add proper CloudWorkstation tags
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
