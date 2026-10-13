// Package templates provides Prism's unified template system.
//
// This package replaces the fragmented template definitions across the codebase
// with a single, simplified system that leverages existing package managers
// (apt, conda, spack) instead of custom bash scripts.
package templates

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/scttfrdmn/prism/pkg/types"
	"gopkg.in/yaml.v3"
)

// appendIfDir appends path to dirs if it exists and is a directory.
func appendIfDir(dirs []string, path string) []string {
	if info, err := os.Stat(path); err == nil && info.IsDir() { //nolint:gosec // G703: candidate template dirs from local env/home/binary location, only stat'd
		return append(dirs, path)
	}
	return dirs
}

// binaryRelativeTemplateDirs returns template dirs relative to the running executable.
func binaryRelativeTemplateDirs() []string {
	exe, err := os.Executable()
	if err != nil {
		return nil
	}
	exeDir := filepath.Dir(exe)
	var dirs []string
	dirs = appendIfDir(dirs, filepath.Clean(filepath.Join(exeDir, "..", "templates")))
	dirs = appendIfDir(dirs, filepath.Clean(filepath.Join(exeDir, "..", "share", "templates")))
	return dirs
}

// workdirTemplateDirs returns the cwd-relative templates directory if it exists and is not a duplicate.
func workdirTemplateDirs(existing []string) []string {
	wd, err := os.Getwd()
	if err != nil {
		return nil
	}
	path := filepath.Join(wd, "templates")
	info, err := os.Stat(path)
	if err != nil || !info.IsDir() {
		return nil
	}
	for _, dir := range existing {
		if dir == path {
			return nil // already present
		}
	}
	return []string{path}
}

// DefaultTemplateDirs returns the default template directories to scan
func DefaultTemplateDirs() []string {
	log.Printf("🔍 DEFAULT TEMPLATE DIRS CALLED - STARTING DISCOVERY")
	dirs := []string{}

	// HIGHEST PRIORITY: Environment variable override (for tests and custom setups)
	if envDir := os.Getenv("PRISM_TEMPLATE_DIR"); envDir != "" {
		envDir = filepath.Clean(envDir)
		if info, err := os.Stat(envDir); err == nil && info.IsDir() { //nolint:gosec // G703: PRISM_TEMPLATE_DIR is the local user's own config, only stat'd
			return []string{envDir} // Use ONLY the environment variable path when set
		}
	}

	dirs = append(dirs, binaryRelativeTemplateDirs()...)
	dirs = append(dirs, workdirTemplateDirs(dirs)...)
	dirs = appendIfDir(dirs, filepath.Join(os.Getenv("HOME"), ".prism", "templates"))
	dirs = appendIfDir(dirs, "/etc/prism/templates")

	if homebrewPrefix := os.Getenv("HOMEBREW_PREFIX"); homebrewPrefix != "" {
		dirs = append(dirs, filepath.Join(homebrewPrefix, "share", "prism", "templates"))
	}
	for _, path := range []string{
		"/opt/homebrew/share/prism/templates",
		"/usr/local/share/prism/templates",
	} {
		if _, err := os.Stat(path); err == nil {
			dirs = append(dirs, path)
		}
	}

	log.Printf("[TEMPLATE DISCOVERY] Total template directories found: %d", len(dirs)) //nolint:gosec // G706: integer count only, cannot inject log content
	if len(dirs) == 0 {
		log.Println("[TEMPLATE DISCOVERY] ⚠️  WARNING: NO template directories found! Templates will not be available!")
	} else {
		log.Println("[TEMPLATE DISCOVERY] Template directories:")
		for i, dir := range dirs {
			log.Printf("[TEMPLATE DISCOVERY]   %d. %s", i+1, dir) //nolint:gosec // G706: dir comes from the local user's env/filesystem, not remote input
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
	LoadCommunityTemplatesIntoRegistry(registry)

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
				UserIntegration: types.DualUserIntegration{
					Strategy:    template.ResearchUser.UserIntegration.Strategy,
					PrimaryUser: template.ResearchUser.UserIntegration.PrimaryUser,
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
	return GetTemplateWithDiscovery(name, region, architecture, packageManager, size, nil)
}

// GetTemplateWithDiscovery returns a single template with AMI discovery service (Issue #436)
// This function integrates dynamic AMI lookup via AWS SSM Parameter Store
func GetTemplateWithDiscovery(name, region, architecture, packageManager, size string, amiDiscovery AMIDiscoveryService) (*types.RuntimeTemplate, error) {
	registry := NewTemplateRegistry(DefaultTemplateDirs())

	// Create resolver with or without AMI discovery
	var resolver *TemplateResolver
	if amiDiscovery != nil {
		resolver = NewTemplateResolverWithDiscovery(amiDiscovery)
	} else {
		resolver = NewTemplateResolver()
	}

	// Scan for templates
	if err := registry.ScanTemplates(); err != nil {
		return nil, fmt.Errorf("failed to scan templates: %w", err)
	}
	LoadCommunityTemplatesIntoRegistry(registry)

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
			UserIntegration: types.DualUserIntegration{
				Strategy:    template.ResearchUser.UserIntegration.Strategy,
				PrimaryUser: template.ResearchUser.UserIntegration.PrimaryUser,
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
		RootVolumeGB:         runtimeTemplate.RootVolumeGB,
		EstimatedCostPerHour: runtimeTemplate.EstimatedCostPerHour,
		IdleDetection:        idleDetectionConfig,
		ResearchUser:         researchUserConfig,
	}

	return legacyTemplate, nil
}

// GetTemplateWithParameters returns a template with parameter processing applied
func GetTemplateWithParameters(name, region, architecture, packageManager, size string, parameters map[string]interface{}) (*types.RuntimeTemplate, error) {
	return GetTemplateWithParametersAndDiscovery(name, region, architecture, packageManager, size, parameters, nil)
}

// GetTemplateWithParametersAndDiscovery returns a template with parameter processing and AMI discovery (Issue #436)
func GetTemplateWithParametersAndDiscovery(name, region, architecture, packageManager, size string, parameters map[string]interface{}, amiDiscovery AMIDiscoveryService) (*types.RuntimeTemplate, error) {
	// First, get the raw template
	registry := NewTemplateRegistry(DefaultTemplateDirs())
	if err := registry.ScanTemplates(); err != nil {
		return nil, fmt.Errorf("failed to scan templates: %w", err)
	}
	LoadCommunityTemplatesIntoRegistry(registry)

	template, err := registry.GetTemplate(name)
	if err != nil {
		return nil, fmt.Errorf("template not found: %w", err)
	}

	// Process parameters if the template has them
	// Issue #444: Always process parameters if template defines them, even if user doesn't provide any
	// This ensures default parameter values are applied
	var processedTemplate *Template
	if len(template.Parameters) > 0 {
		// DEBUG: Log parameter processing for Issue #450
		log.Printf("DEBUG [Issue #450]: Template '%s' has %d parameters", template.Name, len(template.Parameters))
		log.Printf("DEBUG [Issue #450]: User provided %d parameter values: %+v", len(parameters), parameters)

		processor := NewParameterProcessor(template, parameters)

		// Validate parameters
		if validationErrors := processor.ValidateParameters(); len(validationErrors) > 0 {
			var errorMessages []string
			for _, vErr := range validationErrors {
				errorMessages = append(errorMessages, vErr.Error())
			}
			// DEBUG: Log validation errors
			log.Printf("DEBUG [Issue #450]: Validation FAILED for '%s': %v", template.Name, errorMessages)
			return nil, fmt.Errorf("parameter validation failed: %s", strings.Join(errorMessages, ", "))
		}

		log.Printf("DEBUG [Issue #450]: Validation PASSED for '%s', processing template...", template.Name)

		// Process template with parameters
		processedTemplate, err = processor.ProcessTemplate()
		if err != nil {
			log.Printf("DEBUG [Issue #450]: Processing FAILED for '%s': %v", template.Name, err)
			return nil, fmt.Errorf("parameter processing failed: %w", err)
		}

		log.Printf("DEBUG [Issue #450]: Processing SUCCEEDED for '%s', post_install length: %d",
			template.Name, len(processedTemplate.PostInstall))
		log.Printf("DEBUG [Issue #450]: Post-install preview (first 200 chars): %.200s",
			processedTemplate.PostInstall)
	} else {
		processedTemplate = template
	}

	// Convert to runtime template using resolver with AMI discovery
	var resolver *TemplateResolver
	if amiDiscovery != nil {
		resolver = NewTemplateResolverWithDiscovery(amiDiscovery)
	} else {
		resolver = NewTemplateResolver()
	}

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
			UserIntegration: types.DualUserIntegration{
				Strategy:    template.ResearchUser.UserIntegration.Strategy,
				PrimaryUser: template.ResearchUser.UserIntegration.PrimaryUser,
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
		RootVolumeGB:         runtimeTemplate.RootVolumeGB,
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
	LoadCommunityTemplatesIntoRegistry(registry)

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
	LoadCommunityTemplatesIntoRegistry(registry)

	return registry.GetTemplate(name)
}

// GenerateScript generates an installation script for a template
func GenerateScript(templateName, packageManager string) (string, error) {
	registry := NewTemplateRegistry(DefaultTemplateDirs())
	if err := registry.ScanTemplates(); err != nil {
		return "", err
	}
	LoadCommunityTemplatesIntoRegistry(registry)

	template, err := registry.GetTemplate(templateName)
	if err != nil {
		return "", err
	}

	// Create plugin registry and load plugins
	pluginRegistry := NewPluginRegistry()
	for _, dir := range DefaultPluginDirs() {
		// Silently ignore errors loading plugins (plugins are optional)
		_ = pluginRegistry.LoadPluginsFromDirectory(dir)
	}

	generator := NewScriptGeneratorWithPlugins(pluginRegistry)
	return generator.GenerateScript(template, PackageManagerType(packageManager))
}

// Examples and utilities for development and testing

// CreateExampleTemplate creates an example template file
func CreateExampleTemplate(filename string) error {
	template := &Template{
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
		Maintainer: "Prism Team",
	}

	// Marshal template to YAML
	data, err := yaml.Marshal(template)
	if err != nil {
		return fmt.Errorf("failed to marshal template to YAML: %w", err)
	}

	// Write to file
	if err := os.WriteFile(filename, data, 0644); err != nil { //nolint:gosec // G306: example template is a non-secret, shareable user file
		return fmt.Errorf("failed to write template file: %w", err)
	}

	fmt.Printf("✅ Created example template: %s\n", filename)
	return nil
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
	return GetTemplatesForRegionWithDiscovery(region, architecture, nil)
}

// GetTemplatesForRegionWithDiscovery returns all templates for the specified region and architecture
// with optional dynamic AMI discovery (Issue #436)
func GetTemplatesForRegionWithDiscovery(region, architecture string, amiDiscovery AMIDiscoveryService) (map[string]types.RuntimeTemplate, error) {
	registry := NewTemplateRegistry(DefaultTemplateDirs())

	// Create resolver with or without AMI discovery
	var resolver *TemplateResolver
	if amiDiscovery != nil {
		resolver = NewTemplateResolverWithDiscovery(amiDiscovery)
	} else {
		resolver = NewTemplateResolver()
	}

	// Scan for templates
	if err := registry.ScanTemplates(); err != nil {
		return nil, fmt.Errorf("failed to scan templates: %w", err)
	}
	LoadCommunityTemplatesIntoRegistry(registry)

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
				UserIntegration: types.DualUserIntegration{
					Strategy:    template.ResearchUser.UserIntegration.Strategy,
					PrimaryUser: template.ResearchUser.UserIntegration.PrimaryUser,
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
