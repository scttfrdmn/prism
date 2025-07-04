// Package cli implements CloudWorkstation's command-line interface application.
package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/scttfrdmn/cloudworkstation/pkg/ami"
)

// handleTemplateValidate handles template validation
func (a *App) handleTemplateValidate(args []string, manager *ami.TemplateManager) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: cws ami template validate <template-name|file-path>")
	}

	source := args[0]
	var template *ami.Template
	var err error
	
	// Check if source is a file path or template name
	if _, err := os.Stat(source); err == nil {
		// It's a file, import it
		template, err = manager.ImportFromFile(source, &ami.TemplateImportOptions{
			Validate: false, // Don't validate yet
			Force:    true,  // Allow import even if already exists
		})
		if err != nil {
			return fmt.Errorf("failed to import template file: %w", err)
		}
		
		fmt.Printf("📄 Validating template file: %s\n", source)
	} else {
		// Try as template name
		template, err = manager.GetTemplate(source)
		if err != nil {
			return fmt.Errorf("template '%s' not found: %w", source, err)
		}
		
		fmt.Printf("📄 Validating template: %s\n", source)
	}

	// Perform validation
	fmt.Printf("🔍 Validating template '%s'...\n", template.Name)
	if err := manager.ValidateTemplate(template.Name); err != nil {
		fmt.Println("❌ Validation failed")
		return err
	}

	fmt.Printf("✅ Template '%s' is valid\n", template.Name)
	fmt.Printf("   Description: %s\n", template.Description)
	fmt.Printf("   Base Image: %s\n", template.Base)
	fmt.Printf("   Build Steps: %d\n", len(template.BuildSteps))
	fmt.Printf("   Validation Checks: %d\n", len(template.Validation))

	return nil
}

// handleTemplateSchema handles template schema management
func (a *App) handleTemplateSchema(args []string, manager *ami.TemplateManager) error {
	if manager.SchemaValidator == nil {
		return fmt.Errorf("schema validator not available")
	}

	// Get the schema
	schema, err := manager.SchemaValidator.GetSchema()
	if err != nil {
		return fmt.Errorf("failed to get schema: %w", err)
	}

	// Check for output path
	var outputPath string
	if len(args) > 0 {
		outputPath = args[0]
	}

	if outputPath != "" {
		// Write to file
		if err := os.WriteFile(outputPath, schema, 0644); err != nil {
			return fmt.Errorf("failed to write schema to file: %w", err)
		}
		fmt.Printf("✅ Schema written to %s\n", outputPath)
	} else {
		// Write to stdout
		fmt.Println("📋 Template JSON Schema:")
		fmt.Println()
		fmt.Println(string(schema))
	}

	return nil
}