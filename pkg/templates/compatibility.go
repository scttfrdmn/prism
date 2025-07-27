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
		legacyTemplate := types.RuntimeTemplate{
			Name:         runtimeTemplate.Name,
			Description:  runtimeTemplate.Description,
			AMI:          runtimeTemplate.AMI,
			InstanceType: runtimeTemplate.InstanceType,
			UserData:     runtimeTemplate.UserData,
			Ports:        runtimeTemplate.Ports,
			EstimatedCostPerHour: runtimeTemplate.EstimatedCostPerHour,
		}
		
		legacyTemplates[name] = legacyTemplate
	}
	
	// Merge with hardcoded legacy templates for backward compatibility
	hardcodedTemplates := getHardcodedLegacyTemplates()
	for name, template := range hardcodedTemplates {
		// Only use hardcoded if no YAML version exists
		if _, exists := legacyTemplates[name]; !exists {
			legacyTemplates[name] = template
		}
	}
	
	return legacyTemplates, nil
}

// GetLegacyTemplate returns a single template in legacy format
func (cm *CompatibilityManager) GetLegacyTemplate(name, region, architecture string) (*types.RuntimeTemplate, error) {
	templates, err := cm.GetLegacyTemplates(region, architecture)
	if err != nil {
		return nil, err
	}
	
	template, exists := templates[name]
	if !exists {
		return nil, fmt.Errorf("template not found: %s", name)
	}
	
	return &template, nil
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
				Name:     "ubuntu",
				Password: "auto-generated",
				Groups:   []string{"sudo"},
				Shell:    "/bin/bash",
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
	return map[string]types.RuntimeTemplate{
		"r-research": {
			Name:        "R Research Environment",
			Description: "R + RStudio Server + tidyverse packages",
			AMI: map[string]map[string]string{
				"us-east-1": {
					"x86_64": "ami-02029c87fa31fb148",
					"arm64":  "ami-050499786ebf55a6a",
				},
				"us-east-2": {
					"x86_64": "ami-0b05d988257befbbe",
					"arm64":  "ami-010755a3881216bba",
				},
				"us-west-1": {
					"x86_64": "ami-043b59f1d11f8f189",
					"arm64":  "ami-0d3e8bea392f79ebb",
				},
				"us-west-2": {
					"x86_64": "ami-016d360a89daa11ba",
					"arm64":  "ami-09f6c9efbf93542be",
				},
			},
			InstanceType: map[string]string{
				"x86_64": "t3.medium",
				"arm64":  "t4g.medium",
			},
			UserData: `#!/bin/bash
apt update -y
apt install -y r-base r-base-dev
# ... (rest of legacy script)
`,
			Ports: []int{22, 8787},
			EstimatedCostPerHour: map[string]float64{
				"x86_64": 0.0464,
				"arm64":  0.0368,
			},
		},
		"python-research": {
			Name:        "Python Research Environment",
			Description: "Python + Jupyter + data science packages",
			AMI: map[string]map[string]string{
				"us-east-1": {
					"x86_64": "ami-02029c87fa31fb148",
					"arm64":  "ami-050499786ebf55a6a",
				},
				"us-east-2": {
					"x86_64": "ami-0b05d988257befbbe",
					"arm64":  "ami-010755a3881216bba",
				},
				"us-west-1": {
					"x86_64": "ami-043b59f1d11f8f189",
					"arm64":  "ami-0d3e8bea392f79ebb",
				},
				"us-west-2": {
					"x86_64": "ami-016d360a89daa11ba",
					"arm64":  "ami-09f6c9efbf93542be",
				},
			},
			InstanceType: map[string]string{
				"x86_64": "t3.medium",
				"arm64":  "t4g.medium",
			},
			UserData: `#!/bin/bash
apt update -y
apt install -y python3 python3-pip
pip3 install jupyter pandas numpy matplotlib seaborn scikit-learn
# ... (rest of legacy script)
`,
			Ports: []int{22, 8888},
			EstimatedCostPerHour: map[string]float64{
				"x86_64": 0.0464,
				"arm64":  0.0368,
			},
		},
		"basic-ubuntu": {
			Name:        "Basic Ubuntu",
			Description: "Plain Ubuntu 22.04 for general use",
			AMI: map[string]map[string]string{
				"us-east-1": {
					"x86_64": "ami-02029c87fa31fb148",
					"arm64":  "ami-050499786ebf55a6a",
				},
				"us-east-2": {
					"x86_64": "ami-0b05d988257befbbe",
					"arm64":  "ami-010755a3881216bba",
				},
				"us-west-1": {
					"x86_64": "ami-043b59f1d11f8f189",
					"arm64":  "ami-0d3e8bea392f79ebb",
				},
				"us-west-2": {
					"x86_64": "ami-016d360a89daa11ba",
					"arm64":  "ami-09f6c9efbf93542be",
				},
			},
			InstanceType: map[string]string{
				"x86_64": "t3.micro",
				"arm64":  "t4g.micro",
			},
			UserData: `#!/bin/bash
apt update -y
echo "Setup complete" > /var/log/cws-setup.log
`,
			Ports: []int{22},
			EstimatedCostPerHour: map[string]float64{
				"x86_64": 0.0116,
				"arm64":  0.0092,
			},
		},
	}
}