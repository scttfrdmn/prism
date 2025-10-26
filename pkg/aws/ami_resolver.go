// Package aws provides AMI resolution capabilities for the Universal AMI System
package aws

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/scttfrdmn/prism/pkg/templates"
	"github.com/scttfrdmn/prism/pkg/types"
)

// UniversalAMIResolver implements multi-tier AMI resolution for Prism templates
type UniversalAMIResolver struct {
	// AWS clients
	ec2Client EC2ClientInterface

	// Configuration
	regionMapping map[string][]string // Regional fallback mapping
	cache         *AMICache
	costAnalyzer  *AMICostAnalyzer
}

// NewUniversalAMIResolver creates a new Universal AMI resolver
func NewUniversalAMIResolver(ec2Client EC2ClientInterface) *UniversalAMIResolver {
	return &UniversalAMIResolver{
		ec2Client:     ec2Client,
		regionMapping: getDefaultRegionMapping(),
		cache:         NewAMICache(),
		costAnalyzer:  NewAMICostAnalyzer(),
	}
}

// ResolveAMI performs multi-tier AMI resolution for a template in a specific region
func (r *UniversalAMIResolver) ResolveAMI(ctx context.Context, template *templates.Template, region string) (*types.AMIResolutionResult, error) {
	result := &types.AMIResolutionResult{
		FallbackChain: make([]string, 0),
		TargetRegion:  region,
	}

	// If template has no AMI config, fallback to script
	if !r.hasAMIConfig(template) {
		result.ResolutionMethod = types.ResolutionFallbackScript
		result.LaunchTime = r.estimateScriptProvisioningTime(template)
		result.Warning = "No AMI configuration found, using script provisioning"
		return result, nil
	}

	// Get AMI strategy (default to preferred)
	strategy := template.AMIConfig.Strategy
	if strategy == "" {
		strategy = templates.AMIStrategyPreferred
	}

	// Apply strategy-specific resolution logic
	switch strategy {
	case templates.AMIStrategyRequired:
		return r.resolveAMIRequired(ctx, template, region, result)
	case templates.AMIStrategyFallback:
		return r.resolveAMIFallback(ctx, template, region, result)
	case templates.AMIStrategyPreferred:
		return r.resolveAMIPreferred(ctx, template, region, result)
	default:
		return nil, fmt.Errorf("unknown AMI strategy: %s", strategy)
	}
}

// resolveAMIPreferred tries AMI first, falls back to script if unavailable
func (r *UniversalAMIResolver) resolveAMIPreferred(ctx context.Context, template *templates.Template, region string, result *types.AMIResolutionResult) (*types.AMIResolutionResult, error) {
	// Try AMI resolution
	if ami, method, err := r.tryAMIResolution(ctx, template, region, result); err == nil {
		result.AMI = ami
		result.ResolutionMethod = method
		result.LaunchTime = r.estimateAMILaunchTime(ami)
		result.EstimatedCost = r.costAnalyzer.CalculateAMICost(ami, region)
		return result, nil
	}

	// AMI resolution failed, fallback to script
	result.ResolutionMethod = types.ResolutionFallbackScript
	result.LaunchTime = r.estimateScriptProvisioningTime(template)
	result.Warning = "AMI not available, using script provisioning"
	return result, nil
}

// resolveAMIRequired requires AMI, fails if unavailable
func (r *UniversalAMIResolver) resolveAMIRequired(ctx context.Context, template *templates.Template, region string, result *types.AMIResolutionResult) (*types.AMIResolutionResult, error) {
	// Try AMI resolution
	if ami, method, err := r.tryAMIResolution(ctx, template, region, result); err == nil {
		result.AMI = ami
		result.ResolutionMethod = method
		result.LaunchTime = r.estimateAMILaunchTime(ami)
		result.EstimatedCost = r.costAnalyzer.CalculateAMICost(ami, region)
		return result, nil
	}

	// AMI required but not available
	result.ResolutionMethod = types.ResolutionFailed
	return result, fmt.Errorf("AMI required but not available in region %s: %v", region, result.FallbackChain)
}

// resolveAMIFallback tries script first, AMI if script fails
func (r *UniversalAMIResolver) resolveAMIFallback(ctx context.Context, template *templates.Template, region string, result *types.AMIResolutionResult) (*types.AMIResolutionResult, error) {
	// Try script provisioning first
	if r.canUseScriptProvisioning(template) {
		result.ResolutionMethod = types.ResolutionFallbackScript
		result.LaunchTime = r.estimateScriptProvisioningTime(template)
		return result, nil
	}

	// Script not available, try AMI
	if ami, method, err := r.tryAMIResolution(ctx, template, region, result); err == nil {
		result.AMI = ami
		result.ResolutionMethod = method
		result.LaunchTime = r.estimateAMILaunchTime(ami)
		result.Warning = "Script provisioning not available, using AMI"
		return result, nil
	}

	// Both methods failed
	result.ResolutionMethod = types.ResolutionFailed
	return result, fmt.Errorf("neither script provisioning nor AMI available")
}

// tryAMIResolution attempts multi-tier AMI resolution
func (r *UniversalAMIResolver) tryAMIResolution(ctx context.Context, template *templates.Template, region string, result *types.AMIResolutionResult) (*types.AMIInfo, types.AMIResolutionMethod, error) {
	// Tier 1: Direct AMI mapping
	if ami, err := r.tryDirectMapping(ctx, template, region, result); err == nil {
		return ami, types.ResolutionDirectMapping, nil
	}

	// Tier 2: Dynamic AMI search
	if ami, err := r.tryDynamicSearch(ctx, template, region, result); err == nil {
		return ami, types.ResolutionDynamicSearch, nil
	}

	// Tier 3: Marketplace search
	if ami, err := r.tryMarketplaceSearch(ctx, template, region, result); err == nil {
		return ami, types.ResolutionMarketplace, nil
	}

	// Tier 4: Cross-region search
	if ami, err := r.tryCrossRegionSearch(ctx, template, region, result); err == nil {
		return ami, types.ResolutionCrossRegion, nil
	}

	return nil, types.ResolutionFailed, fmt.Errorf("all AMI resolution methods failed")
}

// tryDirectMapping checks direct AMI mappings
func (r *UniversalAMIResolver) tryDirectMapping(ctx context.Context, template *templates.Template, region string, result *types.AMIResolutionResult) (*types.AMIInfo, error) {
	result.FallbackChain = append(result.FallbackChain, "direct_mapping")

	// Check new-style mappings
	if template.AMIConfig.AMIMappings != nil {
		if amiID, exists := template.AMIConfig.AMIMappings[region]; exists {
			return r.validateAndGetAMI(ctx, amiID, region)
		}
	}

	// Check legacy mappings
	if template.AMIConfig.AMIs != nil {
		if regionMap, exists := template.AMIConfig.AMIs[region]; exists {
			// Prefer the configured architecture
			preferredArch := template.AMIConfig.PreferredArchitecture
			if preferredArch == "" {
				preferredArch = "arm64" // Default to ARM64 for cost
			}

			if amiID, exists := regionMap[preferredArch]; exists {
				return r.validateAndGetAMI(ctx, amiID, region)
			}

			// Fallback to any available architecture
			for _, amiID := range regionMap {
				if ami, err := r.validateAndGetAMI(ctx, amiID, region); err == nil {
					return ami, nil
				}
			}
		}
	}

	return nil, fmt.Errorf("no direct AMI mapping found for region %s", region)
}

// tryDynamicSearch performs dynamic AMI search
func (r *UniversalAMIResolver) tryDynamicSearch(ctx context.Context, template *templates.Template, region string, result *types.AMIResolutionResult) (*types.AMIInfo, error) {
	result.FallbackChain = append(result.FallbackChain, "dynamic_search")

	if template.AMIConfig.AMISearch == nil {
		return nil, fmt.Errorf("no AMI search configuration")
	}

	search := template.AMIConfig.AMISearch

	// Build search criteria
	criteria := &AMISearchCriteria{
		Owner:        search.Owner,
		NamePattern:  search.NamePattern,
		Region:       region,
		Architecture: search.Architecture,
	}

	// Add version tag if specified
	if search.VersionTag != "" {
		if criteria.RequiredTags == nil {
			criteria.RequiredTags = make(map[string]string)
		}
		criteria.RequiredTags["Version"] = search.VersionTag
	}

	// Merge required tags
	for key, value := range search.RequiredTags {
		if criteria.RequiredTags == nil {
			criteria.RequiredTags = make(map[string]string)
		}
		criteria.RequiredTags[key] = value
	}

	return r.searchAMIByCriteria(ctx, criteria)
}

// tryMarketplaceSearch performs AWS Marketplace AMI search
func (r *UniversalAMIResolver) tryMarketplaceSearch(ctx context.Context, template *templates.Template, region string, result *types.AMIResolutionResult) (*types.AMIInfo, error) {
	result.FallbackChain = append(result.FallbackChain, "marketplace_search")

	if template.AMIConfig.MarketplaceSearch == nil {
		return nil, fmt.Errorf("no marketplace search configuration")
	}

	// Marketplace client not yet implemented
	return nil, fmt.Errorf("marketplace client not yet implemented")
}

// tryCrossRegionSearch searches for AMIs in other regions
func (r *UniversalAMIResolver) tryCrossRegionSearch(ctx context.Context, template *templates.Template, region string, result *types.AMIResolutionResult) (*types.AMIInfo, error) {
	result.FallbackChain = append(result.FallbackChain, "cross_region_search")

	// Check if cross-region is allowed by fallback strategy
	if template.AMIConfig.FallbackStrategy != "cross_region" &&
		template.AMIConfig.FallbackStrategy != "" {
		return nil, fmt.Errorf("cross-region search not enabled")
	}

	fallbackRegions := r.regionMapping[region]
	for _, sourceRegion := range fallbackRegions {
		// Try to find AMI in source region
		tempResult := &types.AMIResolutionResult{TargetRegion: sourceRegion}
		if ami, _, err := r.tryAMIResolution(ctx, template, sourceRegion, tempResult); err == nil {
			// Found AMI in source region, initiate copy
			copiedAMI, err := r.copyAMIToRegion(ctx, ami, sourceRegion, region)
			if err == nil {
				result.SourceRegion = sourceRegion
				result.Warning = fmt.Sprintf("AMI copied from %s (additional cost: $%.3f)", sourceRegion, r.estimateCopyICost(ami, sourceRegion, region))
				return copiedAMI, nil
			}
		}
	}

	return nil, fmt.Errorf("no AMI found in fallback regions")
}

// validateAndGetAMI validates an AMI ID and returns AMI information
func (r *UniversalAMIResolver) validateAndGetAMI(ctx context.Context, amiID, region string) (*types.AMIInfo, error) {
	// Check cache first
	if cachedAMI := r.cache.GetAMI(amiID, region); cachedAMI != nil {
		return cachedAMI, nil
	}

	// Query AWS for AMI information - placeholder for now
	// ami, err := r.ec2Client.DescribeImages(ctx, &ec2.DescribeImagesInput{...})
	ami, err := r.validateAMIExists(ctx, amiID)
	if err != nil {
		return nil, fmt.Errorf("failed to validate AMI %s: %w", amiID, err)
	}

	// Convert to our AMI info structure
	amiInfo := &types.AMIInfo{
		AMIID:        ami.AMIID,
		Name:         ami.Name,
		Description:  ami.Description,
		Region:       region,
		Architecture: ami.Architecture,
		Owner:        ami.Owner,
		CreationDate: ami.CreationDate,
		State:        ami.State,
		Public:       ami.Public,
		LaunchTime:   30 * time.Second, // Estimated AMI launch time
	}

	// Cache the result
	r.cache.SetAMI(amiID, region, amiInfo)

	return amiInfo, nil
}

// Helper functions

func (r *UniversalAMIResolver) hasAMIConfig(template *templates.Template) bool {
	return template.AMIConfig.Strategy != "" ||
		template.AMIConfig.AMIMappings != nil ||
		template.AMIConfig.AMISearch != nil ||
		template.AMIConfig.MarketplaceSearch != nil ||
		template.AMIConfig.AMIs != nil
}

func (r *UniversalAMIResolver) canUseScriptProvisioning(template *templates.Template) bool {
	return template.PackageManager != "" && template.PackageManager != "ami"
}

func (r *UniversalAMIResolver) estimateAMILaunchTime(ami *types.AMIInfo) time.Duration {
	if ami != nil && ami.LaunchTime > 0 {
		return ami.LaunchTime
	}
	return 30 * time.Second // Default AMI launch time
}

func (r *UniversalAMIResolver) estimateScriptProvisioningTime(template *templates.Template) time.Duration {
	// Estimate based on package count and complexity
	baseTime := 3 * time.Minute

	if template.Packages.System != nil {
		baseTime += time.Duration(len(template.Packages.System)) * 10 * time.Second
	}
	if template.Packages.Conda != nil {
		baseTime += time.Duration(len(template.Packages.Conda)) * 20 * time.Second
	}
	if template.Packages.Pip != nil {
		baseTime += time.Duration(len(template.Packages.Pip)) * 5 * time.Second
	}

	return baseTime
}

func (r *UniversalAMIResolver) estimateCopyICost(ami *types.AMIInfo, sourceRegion, targetRegion string) float64 {
	// Simplified cost estimation for cross-region AMI copy
	// In reality, this would use AWS pricing APIs
	return 0.02 // $0.02 for typical AMI copy
}

// getDefaultRegionMapping returns the default regional fallback mapping
func getDefaultRegionMapping() map[string][]string {
	return map[string][]string{
		"us-east-1":      {"us-east-2", "us-west-2", "us-west-1"},
		"us-east-2":      {"us-east-1", "us-west-2", "us-west-1"},
		"us-west-1":      {"us-west-2", "us-east-1", "us-east-2"},
		"us-west-2":      {"us-west-1", "us-east-1", "us-east-2"},
		"ca-central-1":   {"us-east-1", "us-east-2"},
		"eu-west-1":      {"eu-west-2", "eu-central-1", "us-east-1"},
		"eu-west-2":      {"eu-west-1", "eu-central-1", "us-east-1"},
		"eu-west-3":      {"eu-west-1", "eu-west-2", "eu-central-1"},
		"eu-central-1":   {"eu-west-1", "eu-west-2", "us-east-1"},
		"eu-north-1":     {"eu-west-1", "eu-central-1"},
		"ap-south-1":     {"ap-southeast-1", "ap-northeast-1", "us-east-1"},
		"ap-southeast-1": {"ap-southeast-2", "ap-northeast-1", "us-east-1"},
		"ap-southeast-2": {"ap-southeast-1", "ap-northeast-1", "us-east-1"},
		"ap-northeast-1": {"ap-northeast-2", "ap-southeast-1", "us-east-1"},
		"ap-northeast-2": {"ap-northeast-1", "ap-southeast-1", "us-east-1"},
		"ap-northeast-3": {"ap-northeast-1", "ap-northeast-2", "us-east-1"},
		"sa-east-1":      {"us-east-1", "us-east-2"},
	}
}

// AMISearchCriteria defines search criteria for dynamic AMI discovery
type AMISearchCriteria struct {
	Owner        string
	NamePattern  string
	Region       string
	Architecture []string
	RequiredTags map[string]string
}

// Placeholder methods - these would be implemented with full AWS integration
func (r *UniversalAMIResolver) searchAMIByCriteria(ctx context.Context, criteria *AMISearchCriteria) (*types.AMIInfo, error) {
	// This would implement actual AWS EC2 DescribeImages call with filters
	return nil, fmt.Errorf("dynamic AMI search not yet implemented")
}

func (r *UniversalAMIResolver) searchMarketplaceAMI(ctx context.Context, search *templates.MarketplaceSearchConfig, region string) (*types.AMIInfo, error) {
	// This would implement AWS Marketplace API calls
	return nil, fmt.Errorf("marketplace AMI search not yet implemented")
}

func (r *UniversalAMIResolver) copyAMIToRegion(ctx context.Context, ami *types.AMIInfo, sourceRegion, targetRegion string) (*types.AMIInfo, error) {
	// This would implement AWS EC2 CopyImage API call
	return nil, fmt.Errorf("cross-region AMI copy not yet implemented")
}

// validateAMIExists is a placeholder method for AMI validation
func (r *UniversalAMIResolver) validateAMIExists(ctx context.Context, amiID string) (*types.AMIInfo, error) {
	// Placeholder implementation - in production this would use DescribeImages
	return &types.AMIInfo{
		AMIID:        amiID,
		Name:         "Placeholder AMI",
		Description:  "Placeholder description",
		Architecture: "x86_64",
		Public:       false,
	}, nil
}

// AMI Creation Methods (Phase 5.1 Enhancement)

// CreateAMIFromInstance creates an AMI from a running instance
func (r *UniversalAMIResolver) CreateAMIFromInstance(ctx context.Context, request *types.AMICreationRequest) (*types.AMICreationResult, error) {
	// Validate the instance exists and is in a good state for AMI creation
	if err := r.validateInstanceForAMICreation(ctx, request.InstanceID); err != nil {
		return nil, fmt.Errorf("instance validation failed: %w", err)
	}

	// Generate unique AMI name if not provided
	amiName := request.Name
	if amiName == "" {
		amiName = fmt.Sprintf("cws-%s-%d", request.TemplateName, time.Now().Unix())
	}

	// Prepare AMI creation parameters
	createParams := &AMICreateParams{
		InstanceID:         request.InstanceID,
		Name:               amiName,
		Description:        request.Description,
		NoReboot:           request.NoReboot,
		BlockDeviceMapping: request.BlockDeviceMapping,
		Tags:               request.Tags,
	}

	// Initiate AMI creation via EC2 API
	amiID, err := r.initiateAMICreation(ctx, createParams)
	if err != nil {
		return nil, fmt.Errorf("failed to initiate AMI creation: %w", err)
	}

	// Create initial result
	result := &types.AMICreationResult{
		AMIID:        amiID,
		Name:         amiName,
		Status:       types.AMICreationInProgress,
		CreationTime: 0,                                                // Will be updated when complete
		StorageCost:  r.costAnalyzer.getStorageCost("us-east-1") * 8.0, // Estimate 8GB
		CreationCost: 0.05,                                             // Small instance cost during creation
	}

	// Handle multi-region deployment if requested
	if len(request.MultiRegion) > 0 {
		result.RegionResults = make(map[string]*types.RegionAMIResult)
		for _, region := range request.MultiRegion {
			result.RegionResults[region] = &types.RegionAMIResult{
				Region: region,
				Status: types.AMICreationPending,
			}
		}
	}

	return result, nil
}

// GetAMICreationStatus checks the status of AMI creation
func (r *UniversalAMIResolver) GetAMICreationStatus(ctx context.Context, amiID string) (*types.AMICreationResult, error) {
	// Query AMI status via EC2 API
	status, err := r.queryAMIStatus(ctx, amiID)
	if err != nil {
		return nil, fmt.Errorf("failed to query AMI status: %w", err)
	}

	result := &types.AMICreationResult{
		AMIID:        amiID,
		Status:       status.Status,
		CreationTime: status.ElapsedTime,
	}

	// Update status based on AMI state
	switch status.State {
	case "available":
		result.Status = types.AMICreationCompleted
	case "pending":
		result.Status = types.AMICreationInProgress
	case "failed":
		result.Status = types.AMICreationFailed
	}

	return result, nil
}

// Private helper methods for AMI creation

type AMICreateParams struct {
	InstanceID         string
	Name               string
	Description        string
	NoReboot           bool
	BlockDeviceMapping bool
	Tags               map[string]string
}

type AMIStatusInfo struct {
	State       string
	Status      types.AMICreationStatus
	ElapsedTime time.Duration
}

// validateInstanceForAMICreation ensures instance is ready for AMI creation
func (r *UniversalAMIResolver) validateInstanceForAMICreation(ctx context.Context, instanceID string) error {
	// In production, this would:
	// 1. Check instance exists and is running
	// 2. Verify instance is not already being used for AMI creation
	// 3. Ensure instance has no pending operations
	// 4. Validate instance is in a stable state

	// Placeholder validation
	if instanceID == "" {
		return fmt.Errorf("instance ID is required")
	}

	if len(instanceID) < 10 || !strings.HasPrefix(instanceID, "i-") {
		return fmt.Errorf("invalid instance ID format: %s", instanceID)
	}

	return nil
}

// initiateAMICreation starts the AMI creation process via AWS EC2 API
func (r *UniversalAMIResolver) initiateAMICreation(ctx context.Context, params *AMICreateParams) (string, error) {
	// In production, this would use EC2 CreateImage API:
	//
	// input := &ec2.CreateImageInput{
	//     InstanceId:   aws.String(params.InstanceID),
	//     Name:         aws.String(params.Name),
	//     Description:  aws.String(params.Description),
	//     NoReboot:     aws.Bool(params.NoReboot),
	// }
	//
	// result, err := r.ec2Client.CreateImage(ctx, input)
	// if err != nil {
	//     return "", err
	// }
	//
	// return *result.ImageId, nil

	// Placeholder implementation
	amiID := fmt.Sprintf("ami-%016x", time.Now().UnixNano()&0xffffffffffff)

	// Simulate API call delay
	time.Sleep(100 * time.Millisecond)

	return amiID, nil
}

// queryAMIStatus checks the current status of AMI creation
func (r *UniversalAMIResolver) queryAMIStatus(ctx context.Context, amiID string) (*AMIStatusInfo, error) {
	// In production, this would use EC2 DescribeImages API:
	//
	// input := &ec2.DescribeImagesInput{
	//     ImageIds: []string{amiID},
	// }
	//
	// result, err := r.ec2Client.DescribeImages(ctx, input)
	// if err != nil {
	//     return nil, err
	// }
	//
	// if len(result.Images) == 0 {
	//     return nil, fmt.Errorf("AMI not found: %s", amiID)
	// }
	//
	// image := result.Images[0]
	// return &AMIStatusInfo{
	//     State:       *image.State,
	//     ElapsedTime: time.Since(*image.CreationDate),
	// }, nil

	// Placeholder implementation - simulate AMI creation progress
	info := &AMIStatusInfo{
		State:       "pending", // Simulate in-progress
		Status:      types.AMICreationInProgress,
		ElapsedTime: time.Duration(5) * time.Minute, // Simulate 5 minutes elapsed
	}

	return info, nil
}
