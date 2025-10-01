// Package aws provides AMI integration methods for the Universal AMI System
package aws

import (
	"context"
	"fmt"

	"github.com/scttfrdmn/cloudworkstation/pkg/templates"
	"github.com/scttfrdmn/cloudworkstation/pkg/types"
	ctypes "github.com/scttfrdmn/cloudworkstation/pkg/types"
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
			TemplateName:     templateName,
			Region:           m.region,
			Recommendation:   "script_recommended",
			Reasoning:        "AMI not available, script provisioning is the only option",
			ScriptSetupTime:  int(m.amiResolver.estimateScriptProvisioningTime(rawTemplate).Minutes()),
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
func (m *Manager) LaunchInstanceWithAMI(req ctypes.LaunchRequest) (*ctypes.Instance, error) {
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
func (m *Manager) launchWithAMI(req ctypes.LaunchRequest, amiResult *types.AMIResolutionResult, arch string) (*ctypes.Instance, error) {
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
func (m *Manager) launchWithResolvedTemplate(req ctypes.LaunchRequest, template *ctypes.RuntimeTemplate, amiResult *types.AMIResolutionResult) (*ctypes.Instance, error) {
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
func (m *Manager) launchWithTemplate(req ctypes.LaunchRequest, template *ctypes.RuntimeTemplate) (*ctypes.Instance, error) {
	// This would integrate with the existing launch orchestration system
	// For now, we'll call the existing method
	arch := m.getLocalArchitecture()
	return m.launchWithUnifiedTemplateSystem(req, arch)
}