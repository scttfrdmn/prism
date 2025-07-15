// Package cli implements CloudWorkstation's command-line interface application.
package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
	"github.com/scttfrdmn/cloudworkstation/pkg/ami"
)

// handleAMITemplate handles template management operations
func (a *App) handleAMITemplate(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("missing template command (import, export, list, create)")
	}

	// Initialize AWS clients for template registry
	cfg, err := config.LoadDefaultConfig(a.ctx)
	if err != nil {
		return fmt.Errorf("failed to load AWS config: %w", err)
	}

	// ec2Client := ec2.NewFromConfig(cfg)
	ssmClient := ssm.NewFromConfig(cfg)

	// Initialize AMI registry for template sharing
	registry := ami.NewRegistry(ssmClient, "/cloudworkstation/ami")

	// Create base AMIs map for template validation
	baseAMIs := map[string]map[string]string{
		"us-east-1": {
			"ubuntu-22.04-server-lts":       "ami-02029c87fa31fb148", // x86_64
			"ubuntu-22.04-server-lts-arm64": "ami-050499786ebf55a6a", // arm64
		},
	}

	// Initialize template parser
	parser := ami.NewParser(baseAMIs)

	// Determine template directory
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}
	templateDir := filepath.Join(homeDir, ".cloudworkstation", "templates")
	
	// Ensure template directory exists
	if err := os.MkdirAll(templateDir, 0755); err != nil {
		return fmt.Errorf("failed to create template directory: %w", err)
	}

	// Initialize template manager
	manager := ami.NewTemplateManager(parser, registry, templateDir)

	// Process subcommand
	subcommand := args[0]
	subargs := args[1:]

	switch subcommand {
	case "import":
		return a.handleTemplateImport(subargs, manager)
	case "export":
		return a.handleTemplateExport(subargs, manager)
	case "list":
		return a.handleTemplateList(subargs, manager)
	case "create":
		return a.handleTemplateCreate(subargs, manager)
	case "delete":
		return a.handleTemplateDelete(subargs, manager)
	case "validate":
		return a.handleTemplateValidate(subargs, manager)
	case "schema":
		return a.handleTemplateSchema(subargs, manager)
	case "share":
		return a.handleTemplateShare(subargs, manager)
	case "import-shared":
		return a.handleTemplateImportShared(subargs, manager)
	case "list-shared":
		return a.handleTemplateListShared(subargs, manager)
	default:
		return fmt.Errorf("unknown template command: %s", subcommand)
	}
}

// handleTemplateImport handles template import operations
func (a *App) handleTemplateImport(args []string, manager *ami.TemplateManager) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: cws ami template import <file-path|url> [--name <template-name>] [--force]")
	}

	source := args[0]
	cmdArgs := parseCmdArgs(args[1:])
	
	// Parse options
	options := &ami.TemplateManagerImportOptions{
		Validate:      true,
		Force:         cmdArgs["force"] != "",
		OverwriteName: cmdArgs["name"],
	}

	var template *ami.Template
	var err error

	// Check if source is a URL
	if strings.HasPrefix(source, "http://") || strings.HasPrefix(source, "https://") {
		fmt.Printf("🌐 Importing template from URL: %s\n", source)
		template, err = manager.ImportFromURL(source, options)
	} else if strings.HasPrefix(source, "github:") {
		// Parse github:<username>/<repo>/<path>[@ref]
		parts := strings.SplitN(strings.TrimPrefix(source, "github:"), "@", 2)
		repoPath := parts[0]
		ref := "main" // default ref
		if len(parts) > 1 {
			ref = parts[1]
		}

		// Split repository and path
		repoParts := strings.SplitN(repoPath, "/", 3)
		if len(repoParts) < 3 {
			return fmt.Errorf("invalid GitHub format. Use github:<username>/<repo>/<path>[@ref]")
		}

		repo := repoParts[0] + "/" + repoParts[1]
		path := repoParts[2]

		fmt.Printf("📂 Importing template from GitHub: %s/%s@%s\n", repo, path, ref)
		template, err = manager.ImportFromGitHub(repo, path, ref, options)
	} else {
		// Local file
		fmt.Printf("📄 Importing template from file: %s\n", source)
		template, err = manager.ImportFromFile(source, options)
	}

	if err != nil {
		return fmt.Errorf("template import failed: %w", err)
	}

	fmt.Printf("✅ Successfully imported template '%s'\n", template.Name)
	fmt.Printf("   Description: %s\n", template.Description)
	fmt.Printf("   Base Image: %s\n", template.Base)
	fmt.Printf("   Build Steps: %d\n", len(template.BuildSteps))
	fmt.Printf("   Validation Checks: %d\n", len(template.Validation))

	// Save to file for future use
	outputPath := filepath.Join(manager.TemplateDirectory, template.Name+".yaml")
	if err := manager.ExportToFile(template.Name, outputPath, nil); err != nil {
		fmt.Printf("⚠️  Warning: Failed to save template to %s: %v\n", outputPath, err)
	} else {
		fmt.Printf("📝 Template saved to %s\n", outputPath)
	}

	return nil
}

// handleTemplateExport handles template export operations
func (a *App) handleTemplateExport(args []string, manager *ami.TemplateManager) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: cws ami template export <template-name> [--output <file-path>] [--format yaml|json]")
	}

	templateName := args[0]
	cmdArgs := parseCmdArgs(args[1:])
	
	// Get output file path
	outputPath := cmdArgs["output"]
	if outputPath == "" {
		outputPath = templateName + ".yaml" // Default output path
	}

	// Get format
	format := cmdArgs["format"]
	if format == "" {
		// Detect format from file extension
		ext := filepath.Ext(outputPath)
		if ext == ".json" {
			format = "json"
		} else {
			format = "yaml"
		}
	}

	// Validate format
	if format != "yaml" && format != "json" {
		return fmt.Errorf("unsupported format: %s (must be yaml or json)", format)
	}

	// Create export options
	options := &ami.TemplateExportOptions{
		Format:      format,
		PrettyPrint: true,
	}

	// Export template
	fmt.Printf("📤 Exporting template '%s' to %s format\n", templateName, format)
	if err := manager.ExportToFile(templateName, outputPath, options); err != nil {
		return fmt.Errorf("template export failed: %w", err)
	}

	fmt.Printf("✅ Template exported successfully to %s\n", outputPath)
	return nil
}

// handleTemplateList handles listing available templates
func (a *App) handleTemplateList(args []string, manager *ami.TemplateManager) error {
	cmdArgs := parseCmdArgs(args)
	includeDetails := cmdArgs["details"] != ""

	// List templates
	templates, err := manager.ListTemplates(true)
	if err != nil {
		return fmt.Errorf("failed to list templates: %w", err)
	}

	if len(templates) == 0 {
		fmt.Println("No templates available")
		fmt.Println("To import templates, use: cws ami template import <file-path|url>")
		return nil
	}

	fmt.Printf("📋 Available Templates (%d):\n\n", len(templates))
	
	for name, template := range templates {
		// Get metadata
		metadata, hasMetadata := manager.TemplateMetadata[name]
		
		fmt.Printf("📄 %s\n", name)
		fmt.Printf("   Description: %s\n", template.Description)
		fmt.Printf("   Base Image: %s\n", template.Base)
		
		if includeDetails {
			fmt.Printf("   Architecture: %s\n", template.Architecture)
			fmt.Printf("   Build Steps: %d\n", len(template.BuildSteps))
			fmt.Printf("   Validation Checks: %d\n", len(template.Validation))
			
			if template.MinDiskSize > 0 {
				fmt.Printf("   Min Disk Size: %d GB\n", template.MinDiskSize)
			}
			
			if len(template.Tags) > 0 {
				fmt.Printf("   Tags: ")
				first := true
				for k, v := range template.Tags {
					if !first {
						fmt.Print(", ")
					}
					fmt.Printf("%s: %s", k, v)
					first = false
				}
				fmt.Println()
			}
			
			if hasMetadata {
				fmt.Printf("   Last Modified: %s\n", metadata.LastModified.Format(time.RFC3339))
				fmt.Printf("   Validation Status: %s\n", metadata.ValidationStatus)
				
				if metadata.SourcePath != "" {
					fmt.Printf("   Source: %s (local file)\n", metadata.SourcePath)
				} else if metadata.SourceURL != "" {
					fmt.Printf("   Source: %s (URL)\n", metadata.SourceURL)
				}
				
				if metadata.IsBuiltIn {
					fmt.Printf("   Built-in: Yes\n")
				}
			}
		} else {
			fmt.Printf("   Build Steps: %d, Validation Checks: %d\n", 
				len(template.BuildSteps), len(template.Validation))
		}
		
		fmt.Println()
	}
	
	if !includeDetails {
		fmt.Println("Use 'cws ami template list --details' for more information")
	}

	return nil
}

// handleTemplateCreate handles template creation
func (a *App) handleTemplateCreate(args []string, manager *ami.TemplateManager) error {
	if len(args) < 2 {
		return fmt.Errorf("usage: cws ami template create <template-name> <description> --base <base-image> [options]")
	}

	name := args[0]
	description := args[1]
	cmdArgs := parseCmdArgs(args[2:])
	
	// Validate required parameters
	if cmdArgs["base"] == "" {
		return fmt.Errorf("base image is required (--base <base-image>)")
	}

	// Initialize template builder
	builder := manager.CreateTemplate(name, description).
		WithBase(cmdArgs["base"])
	
	// Add optional parameters
	if cmdArgs["arch"] != "" {
		builder.WithArchitecture(cmdArgs["arch"])
	}
	
	if cmdArgs["min-disk-size"] != "" {
		sizeGB := 0
		fmt.Sscanf(cmdArgs["min-disk-size"], "%d", &sizeGB)
		if sizeGB > 0 {
			builder.WithMinDiskSize(sizeGB)
		}
	}
	
	// Parse tags (format: --tag key=value)
	for k, v := range cmdArgs {
		if strings.HasPrefix(k, "tag-") {
			tagKey := strings.TrimPrefix(k, "tag-")
			builder.WithTag(tagKey, v)
		}
	}

	// Build and validate template
	template, err := builder.Build()
	if err != nil {
		return fmt.Errorf("template creation failed: %w", err)
	}

	fmt.Printf("✅ Template '%s' created successfully\n", template.Name)
	fmt.Printf("   Description: %s\n", template.Description)
	fmt.Printf("   Base Image: %s\n", template.Base)
	
	// Save to file for future use
	outputPath := filepath.Join(manager.TemplateDirectory, template.Name+".yaml")
	if err := manager.ExportToFile(template.Name, outputPath, nil); err != nil {
		fmt.Printf("⚠️  Warning: Failed to save template to %s: %v\n", outputPath, err)
	} else {
		fmt.Printf("📝 Template saved to %s\n", outputPath)
	}
	
	// Provide guidance on next steps
	fmt.Println("\nNext steps:")
	fmt.Printf("1. Add build steps: cws ami template build-step %s <step-name> <script>\n", template.Name)
	fmt.Printf("2. Add validation checks: cws ami template validation %s <check-name> <command>\n", template.Name)
	fmt.Printf("3. Build AMI from template: cws ami build %s\n", template.Name)

	return nil
}

// handleTemplateDelete handles template deletion
func (a *App) handleTemplateDelete(args []string, manager *ami.TemplateManager) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: cws ami template delete <template-name> [--keep-file]")
	}

	name := args[0]
	cmdArgs := parseCmdArgs(args[1:])
	
	// Determine whether to delete the file
	keepFile := cmdArgs["keep-file"] != ""
	
	// Delete template
	fmt.Printf("🗑️  Deleting template '%s'...\n", name)
	if err := manager.DeleteTemplate(name, !keepFile); err != nil {
		return fmt.Errorf("template deletion failed: %w", err)
	}

	fmt.Printf("✅ Template '%s' deleted successfully\n", name)
	if keepFile {
		fmt.Println("Note: Template file was preserved")
	}

	return nil
}