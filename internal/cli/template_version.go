// Package cli implements CloudWorkstation's command-line interface application.
package cli

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/scttfrdmn/cloudworkstation/pkg/ami"
)

// handleTemplateVersion handles template version management
func (a *App) handleTemplateVersion(args []string, manager *ami.TemplateManager) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: cws ami template version <subcommand> [options]")
	}

	subcommand := args[0]
	subargs := args[1:]

	switch subcommand {
	case "get":
		return a.handleTemplateVersionGet(subargs, manager)
	case "set":
		return a.handleTemplateVersionSet(subargs, manager)
	case "increment":
		return a.handleTemplateVersionIncrement(subargs, manager)
	case "create":
		return a.handleTemplateVersionCreate(subargs, manager)
	case "list":
		return a.handleTemplateVersionList(subargs, manager)
	case "search":
		return a.handleTemplateVersionSearch(subargs, manager)
	case "compare":
		return a.handleTemplateVersionCompare(subargs, manager)
	default:
		return fmt.Errorf("unknown version command: %s", subcommand)
	}
}

// handleTemplateVersionGet gets the current version of a template
func (a *App) handleTemplateVersionGet(args []string, manager *ami.TemplateManager) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: cws ami template version get <template-name>")
	}

	templateName := args[0]
	fmt.Printf("🔍 Getting version information for template '%s'\n", templateName)

	// Get template version
	version, err := manager.GetTemplateVersion(templateName)
	if err != nil {
		return fmt.Errorf("failed to get template version: %w", err)
	}

	fmt.Printf("✅ Template: %s\n", templateName)
	fmt.Printf("   Version: %s\n", version.String())

	return nil
}

// handleTemplateVersionSet sets the version of a template
func (a *App) handleTemplateVersionSet(args []string, manager *ami.TemplateManager) error {
	if len(args) < 2 {
		return fmt.Errorf("usage: cws ami template version set <template-name> <version>")
	}

	templateName := args[0]
	versionStr := args[1]

	// Parse version
	version, err := ami.NewVersionInfo(versionStr)
	if err != nil {
		return fmt.Errorf("invalid version format: %w", err)
	}

	// Set template version
	fmt.Printf("📝 Setting version for template '%s' to %s\n", templateName, versionStr)
	if err := manager.SetTemplateVersion(templateName, version); err != nil {
		return fmt.Errorf("failed to set template version: %w", err)
	}

	fmt.Printf("✅ Version updated to %s\n", version.String())

	// Save template to file
	outputPath := filepath.Join(manager.TemplateDirectory, templateName+".yaml")
	if err := manager.ExportToFile(templateName, outputPath, nil); err != nil {
		fmt.Printf("⚠️  Warning: Failed to save template to %s: %v\n", outputPath, err)
	} else {
		fmt.Printf("📝 Template saved to %s\n", outputPath)
	}

	return nil
}

// handleTemplateVersionIncrement increments the version of a template
func (a *App) handleTemplateVersionIncrement(args []string, manager *ami.TemplateManager) error {
	if len(args) < 2 {
		return fmt.Errorf("usage: cws ami template version increment <template-name> <major|minor|patch>")
	}

	templateName := args[0]
	component := args[1]

	// Validate component
	if component != "major" && component != "minor" && component != "patch" {
		return fmt.Errorf("invalid version component: %s (must be major, minor, or patch)", component)
	}

	// Increment version
	fmt.Printf("🔄 Incrementing %s version for template '%s'\n", component, templateName)
	version, err := manager.IncrementTemplateVersion(templateName, component)
	if err != nil {
		return fmt.Errorf("failed to increment template version: %w", err)
	}

	fmt.Printf("✅ Version updated to %s\n", version.String())

	// Save template to file
	outputPath := filepath.Join(manager.TemplateDirectory, templateName+".yaml")
	if err := manager.ExportToFile(templateName, outputPath, nil); err != nil {
		fmt.Printf("⚠️  Warning: Failed to save template to %s: %v\n", outputPath, err)
	} else {
		fmt.Printf("📝 Template saved to %s\n", outputPath)
	}

	return nil
}

// handleTemplateVersionCreate creates a new version of a template
func (a *App) handleTemplateVersionCreate(args []string, manager *ami.TemplateManager) error {
	if len(args) < 2 {
		return fmt.Errorf("usage: cws ami template version create <template-name> <major|minor|patch> [--output <file-path>]")
	}

	templateName := args[0]
	component := args[1]
	cmdArgs := parseCmdArgs(args[2:])

	// Validate component
	if component != "major" && component != "minor" && component != "patch" {
		return fmt.Errorf("invalid version component: %s (must be major, minor, or patch)", component)
	}

	// Get output file path
	outputPath := cmdArgs["output"]
	if outputPath == "" {
		// Automatically generate output path based on template name and version
		version, err := manager.GetTemplateVersion(templateName)
		if err != nil {
			return fmt.Errorf("failed to get template version: %w", err)
		}

		// Increment version based on component
		newVersion := *version
		switch component {
		case "major":
			newVersion.IncrementMajor()
		case "minor":
			newVersion.IncrementMinor()
		case "patch":
			newVersion.IncrementPatch()
		}

		outputPath = filepath.Join(manager.TemplateDirectory, fmt.Sprintf("%s-v%s.yaml", templateName, newVersion.String()))
	}

	// Create new template version
	fmt.Printf("🔄 Creating new %s version of template '%s'\n", component, templateName)
	builder, err := manager.CreateTemplateVersion(templateName, component)
	if err != nil {
		return fmt.Errorf("failed to create template version: %w", err)
	}

	// Build the new template
	template, err := builder.Build()
	if err != nil {
		return fmt.Errorf("failed to build versioned template: %w", err)
	}

	// Export the new template
	fmt.Printf("📤 Saving new template version to %s\n", outputPath)
	if err := manager.ExportToFile(template.Name, outputPath, nil); err != nil {
		return fmt.Errorf("failed to export template: %w", err)
	}

	// Get the new version
	version, _ := manager.GetTemplateVersion(template.Name)
	fmt.Printf("✅ Created template version %s\n", version.String())

	return nil
}

// handleTemplateVersionList lists all versions of a template in the registry
func (a *App) handleTemplateVersionList(args []string, manager *ami.TemplateManager) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: cws ami template version list <template-name>")
	}

	if manager.Registry == nil {
		return fmt.Errorf("template registry not configured")
	}

	templateName := args[0]
	fmt.Printf("🔍 Listing versions for template '%s'\n", templateName)

	// List versions from registry
	ctx := context.Background()
	versions, err := manager.Registry.ListSharedTemplateVersions(ctx, templateName)
	if err != nil {
		return fmt.Errorf("failed to list template versions: %w", err)
	}

	if len(versions) == 0 {
		fmt.Printf("No versions found for template '%s'\n", templateName)
		return nil
	}

	fmt.Printf("📋 Available versions for '%s':\n\n", templateName)
	for _, version := range versions {
		fmt.Printf("📄 %s\n", version)
		// TODO: Add additional information about each version
	}

	return nil
}

// handleTemplateDependency handles template dependency management
func (a *App) handleTemplateDependency(args []string, manager *ami.TemplateManager) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: cws ami template dependency <subcommand> [options]")
	}

	subcommand := args[0]
	subargs := args[1:]

	switch subcommand {
	case "add":
		return a.handleTemplateDependencyAdd(subargs, manager)
	case "remove":
		return a.handleTemplateDependencyRemove(subargs, manager)
	case "list":
		return a.handleTemplateDependencyList(subargs, manager)
	case "check":
		return a.handleTemplateDependencyCheck(subargs, manager)
	case "graph":
		return a.handleTemplateDependencyGraph(subargs, manager)
	case "resolve":
		return a.handleTemplateDependencyResolve(subargs, manager)
	case "analyze":
		return a.handleTemplateDependencyAnalyze(subargs, manager)
	default:
		return fmt.Errorf("unknown dependency command: %s", subcommand)
	}
}

// handleTemplateDependencyAdd adds a dependency to a template
func (a *App) handleTemplateDependencyAdd(args []string, manager *ami.TemplateManager) error {
	if len(args) < 2 {
		return fmt.Errorf("usage: cws ami template dependency add <template-name> <dependency-name> [--version <version>] [--operator <operator>] [--optional]")
	}

	templateName := args[0]
	dependencyName := args[1]
	cmdArgs := parseCmdArgs(args[2:])

	// Create dependency
	dependency := ami.TemplateDependency{
		Name:            dependencyName,
		Version:         cmdArgs["version"],
		VersionOperator: cmdArgs["operator"],
		Optional:        cmdArgs["optional"] != "",
	}

	// Add dependency
	fmt.Printf("📝 Adding dependency '%s' to template '%s'\n", dependencyName, templateName)
	if err := manager.AddDependency(templateName, dependency); err != nil {
		return fmt.Errorf("failed to add dependency: %w", err)
	}

	fmt.Printf("✅ Dependency added successfully\n")

	// Save template to file
	outputPath := filepath.Join(manager.TemplateDirectory, templateName+".yaml")
	if err := manager.ExportToFile(templateName, outputPath, nil); err != nil {
		fmt.Printf("⚠️  Warning: Failed to save template to %s: %v\n", outputPath, err)
	} else {
		fmt.Printf("📝 Template saved to %s\n", outputPath)
	}

	return nil
}

// handleTemplateDependencyRemove removes a dependency from a template
func (a *App) handleTemplateDependencyRemove(args []string, manager *ami.TemplateManager) error {
	if len(args) < 2 {
		return fmt.Errorf("usage: cws ami template dependency remove <template-name> <dependency-name>")
	}

	templateName := args[0]
	dependencyName := args[1]

	// Remove dependency
	fmt.Printf("🗑️  Removing dependency '%s' from template '%s'\n", dependencyName, templateName)
	removed, err := manager.RemoveDependency(templateName, dependencyName)
	if err != nil {
		return fmt.Errorf("failed to remove dependency: %w", err)
	}

	if !removed {
		fmt.Printf("⚠️  Dependency '%s' not found in template '%s'\n", dependencyName, templateName)
		return nil
	}

	fmt.Printf("✅ Dependency removed successfully\n")

	// Save template to file
	outputPath := filepath.Join(manager.TemplateDirectory, templateName+".yaml")
	if err := manager.ExportToFile(templateName, outputPath, nil); err != nil {
		fmt.Printf("⚠️  Warning: Failed to save template to %s: %v\n", outputPath, err)
	} else {
		fmt.Printf("📝 Template saved to %s\n", outputPath)
	}

	return nil
}

// handleTemplateDependencyList lists dependencies for a template
func (a *App) handleTemplateDependencyList(args []string, manager *ami.TemplateManager) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: cws ami template dependency list <template-name>")
	}

	templateName := args[0]
	fmt.Printf("🔍 Listing dependencies for template '%s'\n", templateName)

	// Get template
	template, err := manager.GetTemplate(templateName)
	if err != nil {
		return fmt.Errorf("failed to get template: %w", err)
	}

	if len(template.Dependencies) == 0 {
		fmt.Printf("No dependencies for template '%s'\n", templateName)
		return nil
	}

	fmt.Printf("📋 Dependencies for '%s':\n\n", templateName)
	for _, dep := range template.Dependencies {
		fmt.Printf("📄 %s", dep.Name)
		
		if dep.Version != "" {
			operator := dep.VersionOperator
			if operator == "" {
				operator = ">="
			}
			fmt.Printf(" (%s %s)", operator, dep.Version)
		}
		
		if dep.Optional {
			fmt.Printf(" [optional]")
		}
		
		fmt.Println()
	}

	return nil
}

// handleTemplateDependencyCheck validates dependencies for a template
func (a *App) handleTemplateDependencyCheck(args []string, manager *ami.TemplateManager) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: cws ami template dependency check <template-name>")
	}

	templateName := args[0]
	fmt.Printf("🔍 Checking dependencies for template '%s'\n", templateName)

	// Get template
	template, err := manager.GetTemplate(templateName)
	if err != nil {
		return fmt.Errorf("failed to get template: %w", err)
	}

	if len(template.Dependencies) == 0 {
		fmt.Printf("✅ No dependencies to check for template '%s'\n", templateName)
		return nil
	}

	// Validate dependencies
	if err := manager.ValidateTemplateDependencies(templateName, template.Dependencies); err != nil {
		return fmt.Errorf("dependency validation failed: %w", err)
	}

	fmt.Printf("✅ All dependencies valid for template '%s'\n", templateName)
	return nil
}

// handleTemplateDependencyGraph shows the dependency graph for a template
func (a *App) handleTemplateDependencyGraph(args []string, manager *ami.TemplateManager) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: cws ami template dependency graph <template-name>")
	}

	templateName := args[0]
	fmt.Printf("🔍 Generating dependency graph for template '%s'\n", templateName)

	// Get dependency graph
	graph, err := manager.GetDependencyGraph(templateName)
	if err != nil {
		return fmt.Errorf("failed to get dependency graph: %w", err)
	}

	if len(graph) <= 1 {
		fmt.Printf("Template '%s' has no dependencies\n", templateName)
		return nil
	}

	fmt.Printf("📋 Build order for template '%s':\n\n", templateName)
	for i, name := range graph {
		if i == len(graph)-1 {
			fmt.Printf("%d. %s (target template)\n", i+1, name)
		} else {
			fmt.Printf("%d. %s\n", i+1, name)
		}
	}

	return nil
}