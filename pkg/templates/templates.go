// Package templates provides CloudWorkstation's unified template system.
//
// This package replaces the fragmented template definitions across the codebase
// with a single, simplified system that leverages existing package managers
// (apt, conda, spack) instead of custom bash scripts.
package templates

import (
	"fmt"
	"os"
	"path/filepath"
	
	"github.com/scttfrdmn/cloudworkstation/pkg/types"
)

// DefaultTemplateDirs returns the default template directories to scan
func DefaultTemplateDirs() []string {
	return []string{
		"templates",
		filepath.Join(os.Getenv("HOME"), ".cloudworkstation", "templates"),
		"/etc/cloudworkstation/templates",
	}
}

// GetTemplatesForRegion returns all templates formatted for the legacy API
// This maintains backward compatibility with existing code
func GetTemplatesForRegion(region, architecture string) (map[string]types.RuntimeTemplate, error) {
	manager := NewCompatibilityManager(DefaultTemplateDirs())
	return manager.GetLegacyTemplates(region, architecture)
}

// GetTemplate returns a single template for the legacy API
func GetTemplate(name, region, architecture string) (*types.RuntimeTemplate, error) {
	return GetTemplateWithPackageManager(name, region, architecture, "")
}

// GetTemplateWithPackageManager returns a single template with package manager override
func GetTemplateWithPackageManager(name, region, architecture, packageManager string) (*types.RuntimeTemplate, error) {
	manager := NewCompatibilityManager(DefaultTemplateDirs())
	return manager.GetLegacyTemplateWithPackageManager(name, region, architecture, packageManager)
}

// ValidateTemplate validates a template file
func ValidateTemplate(filename string) error {
	parser := NewTemplateParser()
	_, err := parser.ParseTemplateFile(filename)
	return err
}

// ListAvailableTemplates lists all available templates
func ListAvailableTemplates() ([]string, error) {
	registry := NewTemplateRegistry(DefaultTemplateDirs())
	if err := registry.ScanTemplates(); err != nil {
		return nil, err
	}
	
	names := make([]string, 0, len(registry.Templates))
	for name := range registry.Templates {
		names = append(names, name)
	}
	
	return names, nil
}

// GetTemplateInfo returns detailed information about a template
func GetTemplateInfo(name string) (*Template, error) {
	registry := NewTemplateRegistry(DefaultTemplateDirs())
	if err := registry.ScanTemplates(); err != nil {
		return nil, err
	}
	
	return registry.GetTemplate(name)
}

// GenerateScript generates an installation script for a template
func GenerateScript(templateName, packageManager string) (string, error) {
	registry := NewTemplateRegistry(DefaultTemplateDirs())
	if err := registry.ScanTemplates(); err != nil {
		return "", err
	}
	
	template, err := registry.GetTemplate(templateName)
	if err != nil {
		return "", err
	}
	
	generator := NewScriptGenerator()
	return generator.GenerateScript(template, PackageManagerType(packageManager))
}

// Examples and utilities for development and testing

// CreateExampleTemplate creates an example template file
func CreateExampleTemplate(filename string) error {
	_ = &Template{
		Name:        "Example Research Environment",
		Description: "An example template showing the simplified template system",
		Base:        "ubuntu-22.04",
		PackageManager: "auto",
		
		Packages: PackageDefinitions{
			System: []string{"build-essential", "curl", "wget"},
			Conda:  []string{"python=3.11", "jupyter", "numpy", "pandas"},
		},
		
		Services: []ServiceConfig{
			{
				Name:   "jupyter",
				Port:   8888,
				Enable: true,
			},
		},
		
		Users: []UserConfig{
			{
				Name:     "researcher",
				Password: "auto-generated",
				Groups:   []string{"sudo"},
				Shell:    "/bin/bash",
			},
		},
		
		InstanceDefaults: InstanceDefaults{
			Type:  "t3.medium",
			Ports: []int{22, 8888},
			EstimatedCostPerHour: map[string]float64{
				"x86_64": 0.0464,
				"arm64":  0.0368,
			},
		},
		
		Version: "1.0.0",
		Tags: map[string]string{
			"type":     "research",
			"language": "python",
			"example":  "true",
		},
		Maintainer: "CloudWorkstation Team",
	}
	
	// In a real implementation, this would marshal to YAML
	// For now, just create a placeholder
	return fmt.Errorf("YAML marshaling not implemented in this example")
}

// MigrateFromLegacy migrates existing hardcoded templates to YAML format
func MigrateFromLegacy(outputDir string) error {
	manager := NewCompatibilityManager(DefaultTemplateDirs())
	legacyTemplates := getHardcodedLegacyTemplates()
	
	for name, template := range legacyTemplates {
		if err := manager.MigrateTemplateToYAML(template, outputDir); err != nil {
			return fmt.Errorf("failed to migrate template %s: %w", name, err)
		}
	}
	
	return nil
}

// Configuration and settings

// TemplateConfig represents global template system configuration
type TemplateConfig struct {
	// Template directories to scan
	TemplateDirs []string `json:"template_dirs"`
	
	// Default package manager preference
	DefaultPackageManager string `json:"default_package_manager"`
	
	// Cache settings
	CacheEnabled     bool `json:"cache_enabled"`
	CacheTTLMinutes  int  `json:"cache_ttl_minutes"`
	
	// Package manager paths
	PackageManagerPaths map[string]string `json:"package_manager_paths"`
}

// DefaultTemplateConfig returns the default template configuration
func DefaultTemplateConfig() *TemplateConfig {
	return &TemplateConfig{
		TemplateDirs:      DefaultTemplateDirs(),
		DefaultPackageManager: "auto",
		CacheEnabled:      true,
		CacheTTLMinutes:   30,
		PackageManagerPaths: map[string]string{
			"apt":   "/usr/bin/apt-get",
			"conda": "/opt/miniforge/bin/conda",
			"spack": "/opt/spack/bin/spack",
		},
	}
}

// Integration helpers for existing code

// ReplaceAWSTemplatesFunction replaces the existing aws.getTemplates() function
func ReplaceAWSTemplatesFunction() func() map[string]types.RuntimeTemplate {
	return func() map[string]types.RuntimeTemplate {
		templates, err := GetTemplatesForRegion("us-east-1", "x86_64")
		if err != nil {
			// Fallback to hardcoded templates if unified system fails
			return getHardcodedLegacyTemplates()
		}
		return templates
	}
}

// Integration with daemon handlers
func GetTemplatesForDaemonHandler(region, architecture string) (map[string]types.RuntimeTemplate, error) {
	return GetTemplatesForRegion(region, architecture)
}