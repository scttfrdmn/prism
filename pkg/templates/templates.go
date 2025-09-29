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
	"strings"

	"github.com/scttfrdmn/cloudworkstation/pkg/types"
)

// DefaultTemplateDirs returns the default template directories to scan
func DefaultTemplateDirs() []string {
	dirs := []string{}

	// HIGHEST PRIORITY: Current working directory's templates/ (for development)
	if wd, err := os.Getwd(); err == nil {
		devTemplatesPath := filepath.Join(wd, "templates")
		if _, err := os.Stat(devTemplatesPath); err == nil {
			dirs = append(dirs, devTemplatesPath)
		}
	}

	// Add project templates directory for development (binary-relative)
	if exe, err := os.Executable(); err == nil {
		exeDir := filepath.Dir(exe)

		// Development: binary is in bin/, templates are in ../templates
		devTemplatesPath := filepath.Join(exeDir, "..", "templates")
		if _, err := os.Stat(devTemplatesPath); err == nil {
			dirs = append(dirs, devTemplatesPath)
		}

		// Homebrew installation: binary is in bin/, templates are in ../share/templates
		homebrewTemplatesPath := filepath.Join(exeDir, "..", "share", "templates")
		if _, err := os.Stat(homebrewTemplatesPath); err == nil {
			dirs = append(dirs, homebrewTemplatesPath)
		}
	}

	// User templates directory
	dirs = append(dirs, filepath.Join(os.Getenv("HOME"), ".cloudworkstation", "templates"))

	// System templates directory
	dirs = append(dirs, "/etc/cloudworkstation/templates")

	// Add Homebrew installation paths
	if homebrewPrefix := os.Getenv("HOMEBREW_PREFIX"); homebrewPrefix != "" {
		dirs = append(dirs, filepath.Join(homebrewPrefix, "opt", "cloudworkstation", "share", "templates"))
	}

	// Fallback for common Homebrew installations
	commonHomebrewPaths := []string{
		"/opt/homebrew/opt/cloudworkstation/share/templates", // Apple Silicon
		"/usr/local/opt/cloudworkstation/share/templates",    // Intel
	}

	for _, path := range commonHomebrewPaths {
		if _, err := os.Stat(path); err == nil {
			dirs = append(dirs, path)
		}
	}

	return dirs
}

// GetTemplatesForRegion returns all templates for the specified region and architecture
func GetTemplatesForRegion(region, architecture string) (map[string]types.RuntimeTemplate, error) {
	registry := NewTemplateRegistry(DefaultTemplateDirs())
	resolver := NewTemplateResolver()

	// Scan for templates
	if err := registry.ScanTemplates(); err != nil {
		return nil, fmt.Errorf("failed to scan templates: %w", err)
	}

	// Convert to runtime format
	templates := make(map[string]types.RuntimeTemplate)

	for name, template := range registry.Templates {
		runtimeTemplate, err := resolver.ResolveTemplate(template, region, architecture)
		if err != nil {
			// Log the error but continue with other templates
			fmt.Printf("Warning: Failed to resolve template %s: %v\n", name, err)
			continue
		}

		// Convert to runtime template format
		var idleDetectionConfig *types.IdleDetectionConfig
		if runtimeTemplate.IdleDetection != nil {
			idleDetectionConfig = &types.IdleDetectionConfig{
				Enabled:                   runtimeTemplate.IdleDetection.Enabled,
				IdleThresholdMinutes:      runtimeTemplate.IdleDetection.IdleThresholdMinutes,
				HibernateThresholdMinutes: runtimeTemplate.IdleDetection.HibernateThresholdMinutes,
				CheckIntervalMinutes:      runtimeTemplate.IdleDetection.CheckIntervalMinutes,
			}
		}

		// Convert research user configuration (Phase 5A+)
		var researchUserConfig *types.ResearchUserTemplate
		if template.ResearchUser != nil {
			researchUserConfig = &types.ResearchUserTemplate{
				AutoCreate:     template.ResearchUser.AutoCreate,
				RequireEFS:     template.ResearchUser.RequireEFS,
				EFSMountPoint:  template.ResearchUser.EFSMountPoint,
				InstallSSHKeys: template.ResearchUser.InstallSSHKeys,
				DefaultShell:   template.ResearchUser.DefaultShell,
				DefaultGroups:  template.ResearchUser.DefaultGroups,
				DualUserIntegration: &types.DualUserIntegration{
					Strategy:             string(template.ResearchUser.UserIntegration.Strategy),
					PrimaryUser:          template.ResearchUser.UserIntegration.PrimaryUser,
					CollaborationEnabled: true, // Default enabled for research users
				},
			}
		}

		templates[name] = types.RuntimeTemplate{
			Name:                 runtimeTemplate.Name,
			Slug:                 runtimeTemplate.Slug,
			Description:          runtimeTemplate.Description,
			AMI:                  runtimeTemplate.AMI,
			InstanceType:         runtimeTemplate.InstanceType,
			UserData:             runtimeTemplate.UserData,
			Ports:                runtimeTemplate.Ports,
			EstimatedCostPerHour: runtimeTemplate.EstimatedCostPerHour,
			IdleDetection:        idleDetectionConfig,
			ResearchUser:         researchUserConfig,
		}
	}

	return templates, nil
}

// GetTemplate returns a single template for the legacy API
func GetTemplate(name, region, architecture string) (*types.RuntimeTemplate, error) {
	return GetTemplateWithPackageManager(name, region, architecture, "", "")
}

// GetTemplateWithPackageManager returns a single template with package manager override and size scaling
func GetTemplateWithPackageManager(name, region, architecture, packageManager, size string) (*types.RuntimeTemplate, error) {
	registry := NewTemplateRegistry(DefaultTemplateDirs())
	resolver := NewTemplateResolver()

	// Scan for templates
	if err := registry.ScanTemplates(); err != nil {
		return nil, fmt.Errorf("failed to scan templates: %w", err)
	}

	template, err := registry.GetTemplate(name)
	if err != nil {
		return nil, err
	}

	// Resolve with options
	runtimeTemplate, err := resolver.ResolveTemplateWithOptions(template, region, architecture, packageManager, size)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve template %s: %w", name, err)
	}

	// Convert to runtime template format
	var idleDetectionConfig *types.IdleDetectionConfig
	if runtimeTemplate.IdleDetection != nil {
		idleDetectionConfig = &types.IdleDetectionConfig{
			Enabled:                   runtimeTemplate.IdleDetection.Enabled,
			IdleThresholdMinutes:      runtimeTemplate.IdleDetection.IdleThresholdMinutes,
			HibernateThresholdMinutes: runtimeTemplate.IdleDetection.HibernateThresholdMinutes,
			CheckIntervalMinutes:      runtimeTemplate.IdleDetection.CheckIntervalMinutes,
		}
	}

	// Convert research user configuration (Phase 5A+)
	var researchUserConfig *types.ResearchUserTemplate
	if template.ResearchUser != nil {
		researchUserConfig = &types.ResearchUserTemplate{
			AutoCreate:     template.ResearchUser.AutoCreate,
			RequireEFS:     template.ResearchUser.RequireEFS,
			EFSMountPoint:  template.ResearchUser.EFSMountPoint,
			InstallSSHKeys: template.ResearchUser.InstallSSHKeys,
			DefaultShell:   template.ResearchUser.DefaultShell,
			DefaultGroups:  template.ResearchUser.DefaultGroups,
			DualUserIntegration: &types.DualUserIntegration{
				Strategy:             string(template.ResearchUser.UserIntegration.Strategy),
				PrimaryUser:          template.ResearchUser.UserIntegration.PrimaryUser,
				CollaborationEnabled: true, // Default to enabled for template integration
			},
		}
	}

	legacyTemplate := &types.RuntimeTemplate{
		Name:                 runtimeTemplate.Name,
		Slug:                 runtimeTemplate.Slug,
		Description:          runtimeTemplate.Description,
		AMI:                  runtimeTemplate.AMI,
		InstanceType:         runtimeTemplate.InstanceType,
		UserData:             runtimeTemplate.UserData,
		Ports:                runtimeTemplate.Ports,
		EstimatedCostPerHour: runtimeTemplate.EstimatedCostPerHour,
		IdleDetection:        idleDetectionConfig,
		ResearchUser:         researchUserConfig,
	}

	return legacyTemplate, nil
}

// GetTemplateWithParameters returns a template with parameter processing applied
func GetTemplateWithParameters(name, region, architecture, packageManager, size string, parameters map[string]interface{}) (*types.RuntimeTemplate, error) {
	// First, get the raw template
	registry := NewTemplateRegistry(DefaultTemplateDirs())
	if err := registry.ScanTemplates(); err != nil {
		return nil, fmt.Errorf("failed to scan templates: %w", err)
	}

	template, err := registry.GetTemplate(name)
	if err != nil {
		return nil, fmt.Errorf("template not found: %w", err)
	}

	// Process parameters if the template has them
	var processedTemplate *Template
	if len(template.Parameters) > 0 && parameters != nil {
		processor := NewParameterProcessor(template, parameters)

		// Validate parameters
		if validationErrors := processor.ValidateParameters(); len(validationErrors) > 0 {
			var errorMessages []string
			for _, vErr := range validationErrors {
				errorMessages = append(errorMessages, vErr.Error())
			}
			return nil, fmt.Errorf("parameter validation failed: %s", strings.Join(errorMessages, ", "))
		}

		// Process template with parameters
		processedTemplate, err = processor.ProcessTemplate()
		if err != nil {
			return nil, fmt.Errorf("parameter processing failed: %w", err)
		}
	} else {
		processedTemplate = template
	}

	// Convert to runtime template using resolver
	resolver := NewTemplateResolver()
	runtimeTemplate, err := resolver.ResolveTemplateWithOptions(processedTemplate, region, architecture, packageManager, size)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve template: %w", err)
	}

	// Convert to legacy RuntimeTemplate format
	var idleDetectionConfig *types.IdleDetectionConfig
	if runtimeTemplate.IdleDetection != nil {
		idleDetectionConfig = &types.IdleDetectionConfig{
			Enabled:                   runtimeTemplate.IdleDetection.Enabled,
			IdleThresholdMinutes:      runtimeTemplate.IdleDetection.IdleThresholdMinutes,
			HibernateThresholdMinutes: runtimeTemplate.IdleDetection.HibernateThresholdMinutes,
			CheckIntervalMinutes:      runtimeTemplate.IdleDetection.CheckIntervalMinutes,
		}
	}

	// Convert research user configuration (Phase 5A+)
	var researchUserConfig *types.ResearchUserTemplate
	if template.ResearchUser != nil {
		researchUserConfig = &types.ResearchUserTemplate{
			AutoCreate:     template.ResearchUser.AutoCreate,
			RequireEFS:     template.ResearchUser.RequireEFS,
			EFSMountPoint:  template.ResearchUser.EFSMountPoint,
			InstallSSHKeys: template.ResearchUser.InstallSSHKeys,
			DefaultShell:   template.ResearchUser.DefaultShell,
			DefaultGroups:  template.ResearchUser.DefaultGroups,
			DualUserIntegration: &types.DualUserIntegration{
				Strategy:             string(template.ResearchUser.UserIntegration.Strategy),
				PrimaryUser:          template.ResearchUser.UserIntegration.PrimaryUser,
				CollaborationEnabled: true, // Default to enabled for template integration
			},
		}
	}

	legacyTemplate := &types.RuntimeTemplate{
		Name:                 runtimeTemplate.Name,
		Slug:                 runtimeTemplate.Slug,
		Description:          runtimeTemplate.Description,
		AMI:                  runtimeTemplate.AMI,
		InstanceType:         runtimeTemplate.InstanceType,
		UserData:             runtimeTemplate.UserData,
		Ports:                runtimeTemplate.Ports,
		EstimatedCostPerHour: runtimeTemplate.EstimatedCostPerHour,
		IdleDetection:        idleDetectionConfig,
		ResearchUser:         researchUserConfig,
	}

	return legacyTemplate, nil
}

// ValidateTemplate validates a template file
func ValidateTemplate(filename string) error {
	parser := NewTemplateParser()
	_, err := parser.ParseTemplateFile(filename)
	return err
}

// ValidateTemplateWithRegistry validates a template with inheritance resolution
func ValidateTemplateWithRegistry(templateDirs []string, templateName string) error {
	registry := NewTemplateRegistry(templateDirs)

	// Scan templates to load all templates and resolve inheritance
	if err := registry.ScanTemplates(); err != nil {
		return fmt.Errorf("failed to scan templates: %w", err)
	}

	// Check if template exists
	_, err := registry.GetTemplate(templateName)
	if err != nil {
		return fmt.Errorf("template validation failed: %w", err)
	}

	return nil
}

// ValidateAllTemplates validates all templates in the given directories
func ValidateAllTemplates(templateDirs []string) error {
	registry := NewTemplateRegistry(templateDirs)

	// Scan templates - this will validate all templates and resolve inheritance
	if err := registry.ScanTemplates(); err != nil {
		return fmt.Errorf("template validation failed: %w", err)
	}

	return nil
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
		Name:           "Example Research Environment",
		Description:    "An example template showing the simplified template system",
		Base:           "ubuntu-22.04",
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
				Name:   "researcher",
				Groups: []string{"sudo"},
				Shell:  "/bin/bash",
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
	// No hardcoded templates to migrate - all templates are now YAML-based
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
	CacheEnabled    bool `json:"cache_enabled"`
	CacheTTLMinutes int  `json:"cache_ttl_minutes"`

	// Package manager paths
	PackageManagerPaths map[string]string `json:"package_manager_paths"`
}

// DefaultTemplateConfig returns the default template configuration
func DefaultTemplateConfig() *TemplateConfig {
	return &TemplateConfig{
		TemplateDirs:          DefaultTemplateDirs(),
		DefaultPackageManager: "auto",
		CacheEnabled:          true,
		CacheTTLMinutes:       30,
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
			// No fallback - return empty map if YAML templates fail
			return make(map[string]types.RuntimeTemplate)
		}
		return templates
	}
}

// Integration with daemon handlers
func GetTemplatesForDaemonHandler(region, architecture string) (map[string]types.RuntimeTemplate, error) {
	return GetTemplatesForRegion(region, architecture)
}
