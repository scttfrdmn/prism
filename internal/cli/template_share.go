// Package cli implements CloudWorkstation's command-line interface application.
package cli

import (
	"fmt"
	"time"

	"github.com/scttfrdmn/cloudworkstation/pkg/ami"
)

// handleTemplateShare handles template sharing operations
func (a *App) handleTemplateShare(args []string, manager *ami.TemplateManager) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: cws ami template share <template-name> [--version <version>]")
	}

	templateName := args[0]
	cmdArgs := parseCmdArgs(args[1:])
	
	// Get version if specified
	version := cmdArgs["version"]
	if version == "" {
		version = "1.0.0" // Default version
	}
	
	// Validate template first
	fmt.Printf("🔍 Validating template '%s' before sharing...\n", templateName)
	if err := manager.ValidateTemplate(templateName); err != nil {
		return fmt.Errorf("template validation failed: %w", err)
	}

	// Share template
	fmt.Printf("🌐 Sharing template '%s' (version %s)...\n", templateName, version)
	if err := manager.ShareTemplate(templateName, a.ctx); err != nil {
		return fmt.Errorf("failed to share template: %w", err)
	}

	fmt.Printf("✅ Template '%s' successfully shared\n", templateName)
	return nil
}

// handleTemplateImportShared handles importing shared templates from the registry
func (a *App) handleTemplateImportShared(args []string, manager *ami.TemplateManager) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: cws ami template import-shared <template-name> [--version <version>] [--force]")
	}

	templateName := args[0]
	cmdArgs := parseCmdArgs(args[1:])
	
	// Parse options
	version := cmdArgs["version"] // Empty for latest
	force := cmdArgs["force"] != ""
	
	// Check if registry is available
	if manager.Registry == nil {
		return fmt.Errorf("registry not configured")
	}

	// Get template from registry
	fmt.Printf("🔄 Importing shared template '%s'%s from registry...\n", 
		templateName, version != "" ? " version " + version : "")
	
	entry, err := manager.Registry.GetSharedTemplate(a.ctx, templateName, version)
	if err != nil {
		return fmt.Errorf("failed to retrieve template from registry: %w", err)
	}

	// Parse the template data
	template, err := manager.Parser.ParseTemplate([]byte(entry.TemplateData))
	if err != nil {
		return fmt.Errorf("failed to parse template data: %w", err)
	}

	// Import into local cache
	options := &ami.TemplateImportOptions{
		Validate: true,
		Force:    force,
	}
	
	// Import template
	_, err = manager.ImportFromTemplate(template, options)
	if err != nil {
		return fmt.Errorf("failed to import template: %w", err)
	}

	// Save to file for future use
	outputPath := fmt.Sprintf("%s/%s.yaml", manager.TemplateDirectory, template.Name)
	if err := manager.ExportToFile(template.Name, outputPath, nil); err != nil {
		fmt.Printf("⚠️  Warning: Failed to save template to %s: %v\n", outputPath, err)
	} else {
		fmt.Printf("📝 Template saved to %s\n", outputPath)
	}

	fmt.Printf("✅ Successfully imported shared template '%s'\n", template.Name)
	fmt.Printf("   Description: %s\n", template.Description)
	fmt.Printf("   Publisher: %s\n", entry.PublishedBy)
	fmt.Printf("   Version: %s\n", entry.Version)
	fmt.Printf("   Published: %s\n", entry.PublishedAt.Format(time.RFC3339))
	fmt.Printf("   Build Steps: %d\n", len(template.BuildSteps))
	fmt.Printf("   Validation Checks: %d\n", len(template.Validation))

	return nil
}

// handleTemplateListShared lists templates available in the registry
func (a *App) handleTemplateListShared(args []string, manager *ami.TemplateManager) error {
	// Check if registry is available
	if manager.Registry == nil {
		return fmt.Errorf("registry not configured")
	}
	
	cmdArgs := parseCmdArgs(args)
	includeDetails := cmdArgs["details"] != ""
	
	// List shared templates
	fmt.Println("🔍 Searching registry for shared templates...")
	templates, err := manager.Registry.ListSharedTemplates(a.ctx)
	if err != nil {
		return fmt.Errorf("failed to list shared templates: %w", err)
	}
	
	if len(templates) == 0 {
		fmt.Println("No shared templates found in registry")
		return nil
	}
	
	fmt.Printf("📋 Available Shared Templates (%d):\n\n", len(templates))
	
	for name, entry := range templates {
		fmt.Printf("📄 %s\n", name)
		fmt.Printf("   Description: %s\n", entry.Description)
		fmt.Printf("   Version: %s\n", entry.Version)
		
		if includeDetails {
			fmt.Printf("   Publisher: %s\n", entry.PublishedBy)
			fmt.Printf("   Published: %s\n", entry.PublishedAt.Format(time.RFC3339))
			
			if entry.Architecture != "" {
				fmt.Printf("   Architecture: %s\n", entry.Architecture)
			}
			
			if len(entry.Tags) > 0 {
				fmt.Printf("   Tags: ")
				first := true
				for k, v := range entry.Tags {
					if !first {
						fmt.Print(", ")
					}
					fmt.Printf("%s: %s", k, v)
					first = false
				}
				fmt.Println()
			}
		}
		
		fmt.Println()
	}
	
	fmt.Println("To import a shared template, use: cws ami template import-shared <template-name>")
	
	return nil
}