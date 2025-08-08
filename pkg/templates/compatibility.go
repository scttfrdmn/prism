package templates

import (
	"fmt"
	"path/filepath"
	
	"github.com/scttfrdmn/cloudworkstation/pkg/types"
)

// CompatibilityManager provides backward compatibility with existing template systems
type CompatibilityManager struct {
	Registry *TemplateRegistry
	Resolver *TemplateResolver
	
	// Legacy template cache for performance
	legacyTemplates map[string]types.RuntimeTemplate
}

// NewCompatibilityManager creates a new compatibility manager
func NewCompatibilityManager(templateDirs []string) *CompatibilityManager {
	registry := NewTemplateRegistry(templateDirs)
	resolver := NewTemplateResolver()
	
	return &CompatibilityManager{
		Registry:        registry,
		Resolver:        resolver,
		legacyTemplates: make(map[string]types.RuntimeTemplate),
	}
}

// GetLegacyTemplates returns templates in the legacy format for backward compatibility
func (cm *CompatibilityManager) GetLegacyTemplates(region, architecture string) (map[string]types.RuntimeTemplate, error) {
	// Scan for new templates
	if err := cm.Registry.ScanTemplates(); err != nil {
		return nil, fmt.Errorf("failed to scan templates: %w", err)
	}
	
	// Convert to legacy format
	legacyTemplates := make(map[string]types.RuntimeTemplate)
	
	for name, template := range cm.Registry.Templates {
		runtimeTemplate, err := cm.Resolver.ResolveTemplate(template, region, architecture)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve template %s: %w", name, err)
		}
		
		// Convert to legacy format
		var idleDetectionConfig *types.IdleDetectionConfig
		if runtimeTemplate.IdleDetection != nil {
			idleDetectionConfig = &types.IdleDetectionConfig{
				Enabled:                  runtimeTemplate.IdleDetection.Enabled,
				IdleThresholdMinutes:     runtimeTemplate.IdleDetection.IdleThresholdMinutes,
				HibernateThresholdMinutes: runtimeTemplate.IdleDetection.HibernateThresholdMinutes,
				CheckIntervalMinutes:     runtimeTemplate.IdleDetection.CheckIntervalMinutes,
			}
		}
		
		legacyTemplate := types.RuntimeTemplate{
			Name:         runtimeTemplate.Name,
			Description:  runtimeTemplate.Description,
			AMI:          runtimeTemplate.AMI,
			InstanceType: runtimeTemplate.InstanceType,
			UserData:     runtimeTemplate.UserData,
			Ports:        runtimeTemplate.Ports,
			EstimatedCostPerHour: runtimeTemplate.EstimatedCostPerHour,
			IdleDetection: idleDetectionConfig,
		}
		
		legacyTemplates[name] = legacyTemplate
	}
	
	// No fallback to hardcoded templates - use only YAML templates
	
	return legacyTemplates, nil
}

// GetLegacyTemplate returns a single template in legacy format
func (cm *CompatibilityManager) GetLegacyTemplate(name, region, architecture string) (*types.RuntimeTemplate, error) {
	return cm.GetLegacyTemplateWithPackageManager(name, region, architecture, "", "")
}

// GetLegacyTemplateWithPackageManager returns a single template with package manager override and size scaling
func (cm *CompatibilityManager) GetLegacyTemplateWithPackageManager(name, region, architecture, packageManager, size string) (*types.RuntimeTemplate, error) {
	// Scan for new templates
	if err := cm.Registry.ScanTemplates(); err != nil {
		return nil, fmt.Errorf("failed to scan templates: %w", err)
	}
	
	template, err := cm.Registry.GetTemplate(name)
	if err != nil {
		return nil, err // GetTemplate already returns "template not found: %s" error
	}
	
	// Resolve with package manager override and size scaling
	runtimeTemplate, err := cm.Resolver.ResolveTemplateWithOptions(template, region, architecture, packageManager, size)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve template %s: %w", name, err)
	}
	
	// Convert to legacy format
	var idleDetectionConfig *types.IdleDetectionConfig
	if runtimeTemplate.IdleDetection != nil {
		idleDetectionConfig = &types.IdleDetectionConfig{
			Enabled:                  runtimeTemplate.IdleDetection.Enabled,
			IdleThresholdMinutes:     runtimeTemplate.IdleDetection.IdleThresholdMinutes,
			HibernateThresholdMinutes: runtimeTemplate.IdleDetection.HibernateThresholdMinutes,
			CheckIntervalMinutes:     runtimeTemplate.IdleDetection.CheckIntervalMinutes,
		}
	}
	
	legacyTemplate := types.RuntimeTemplate{
		Name:         runtimeTemplate.Name,
		Description:  runtimeTemplate.Description,
		AMI:          runtimeTemplate.AMI,
		InstanceType: runtimeTemplate.InstanceType,
		UserData:     runtimeTemplate.UserData,
		Ports:        runtimeTemplate.Ports,
		EstimatedCostPerHour: runtimeTemplate.EstimatedCostPerHour,
		IdleDetection: idleDetectionConfig,
	}
	
	return &legacyTemplate, nil
}

// MigrateTemplateToYAML converts a legacy hardcoded template to YAML format
func (cm *CompatibilityManager) MigrateTemplateToYAML(legacyTemplate types.RuntimeTemplate, outputDir string) error {
	// Convert legacy template to unified template format
	unifiedTemplate := &Template{
		Name:        legacyTemplate.Name,
		Description: legacyTemplate.Description,
		Base:        "ubuntu-22.04", // Assume Ubuntu 22.04 for legacy templates
		PackageManager: "auto",
		
		// Extract packages from UserData (basic parsing)
		Packages: extractPackagesFromUserData(legacyTemplate.UserData),
		
		// Extract services from ports
		Services: extractServicesFromPorts(legacyTemplate.Ports),
		
		// Default user setup
		Users: []UserConfig{
			{
				Name:   "ubuntu",
				Groups: []string{"sudo"},
				Shell:  "/bin/bash",
			},
		},
		
		// Instance defaults
		InstanceDefaults: InstanceDefaults{
			Ports: legacyTemplate.Ports,
			EstimatedCostPerHour: legacyTemplate.EstimatedCostPerHour,
		},
		
		// Basic tags
		Tags: map[string]string{
			"migrated": "true",
			"source":   "legacy",
		},
	}
	
	// Write YAML file
	filename := filepath.Join(outputDir, legacyTemplate.Name+".yml")
	return cm.writeTemplateYAML(unifiedTemplate, filename)
}

// extractPackagesFromUserData attempts to extract package names from legacy UserData scripts
func extractPackagesFromUserData(userData string) PackageDefinitions {
	packages := PackageDefinitions{}
	
	// Basic pattern matching for apt-get install commands
	// This is a simplified implementation - in practice, would need more sophisticated parsing
	if contains(userData, "apt-get install") || contains(userData, "apt install") {
		// Extract common packages
		if contains(userData, "r-base") {
			packages.System = append(packages.System, "r-base", "r-base-dev")
		}
		if contains(userData, "python3") {
			packages.System = append(packages.System, "python3", "python3-pip")
		}
		if contains(userData, "build-essential") {
			packages.System = append(packages.System, "build-essential")
		}
	}
	
	return packages
}

// extractServicesFromPorts attempts to determine services from port configuration
func extractServicesFromPorts(ports []int) []ServiceConfig {
	services := make([]ServiceConfig, 0)
	
	for _, port := range ports {
		switch port {
		case 8787:
			services = append(services, ServiceConfig{
				Name:   "rstudio-server",
				Port:   8787,
				Enable: true,
				Config: []string{"www-port=8787"},
			})
		case 8888:
			services = append(services, ServiceConfig{
				Name:   "jupyter",
				Port:   8888,
				Enable: true,
			})
		case 80, 443:
			services = append(services, ServiceConfig{
				Name:   "nginx",
				Port:   port,
				Enable: true,
			})
		// Skip port 22 (SSH) as it's default
		case 22:
			continue
		}
	}
	
	return services
}

// writeTemplateYAML writes a template to a YAML file
func (cm *CompatibilityManager) writeTemplateYAML(template *Template, filename string) error {
	// This would use the YAML marshaling functionality
	// Implementation left as placeholder - would use gopkg.in/yaml.v3
	return fmt.Errorf("YAML writing not implemented in this example")
}

// getHardcodedLegacyTemplates returns the existing hardcoded templates for fallback
func getHardcodedLegacyTemplates() map[string]types.RuntimeTemplate {
	// No hardcoded templates - all templates are now YAML-based
	return make(map[string]types.RuntimeTemplate)
}