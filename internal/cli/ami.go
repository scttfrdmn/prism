package cli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
	"github.com/scttfrdmn/cloudworkstation/pkg/ami"
)

// AMI processes AMI-related commands
func (a *App) AMI(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("missing AMI command (build, list, validate, publish)")
	}

	subcommand := args[0]
	subargs := args[1:]

	switch subcommand {
	case "build":
		return a.handleAMIBuild(subargs)
	case "list":
		return a.handleAMIList(subargs)
	case "validate":
		return a.handleAMIValidate(subargs)
	case "publish":
		return a.handleAMIPublish(subargs)
	default:
		return fmt.Errorf("unknown AMI command: %s", subcommand)
	}
}

// handleAMIBuild builds a new AMI from a template
func (a *App) handleAMIBuild(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("missing template name")
	}

	templateName := args[0]
	cmdArgs := parseCmdArgs(args[1:])
	fmt.Printf("DEBUG: Command args parsed: %+v\n", cmdArgs)

	// Parse command line arguments
	region := cmdArgs["region"]
	architecture := cmdArgs["arch"]
	dryRun := cmdArgs["dry-run"] != ""
	subnetID := cmdArgs["subnet"]
	vpcID := cmdArgs["vpc"]
	
	// Check required parameters
	if !dryRun {
		if subnetID == "" {
			return fmt.Errorf("subnet ID is required for AMI builds (--subnet parameter)")
		}
		if vpcID == "" {
			return fmt.Errorf("VPC ID is required for AMI builds (--vpc parameter)")
		}
	}

	if region == "" {
		region = os.Getenv("AWS_REGION")
		if region == "" {
			region = "us-east-1" // Default
		}
	}

	if architecture == "" {
		architecture = "x86_64" // Default
	}

	// Initialize AWS clients
	cfg, err := config.LoadDefaultConfig(context.TODO(), config.WithRegion(region))
	if err != nil {
		return fmt.Errorf("failed to load AWS config: %w", err)
	}

	ec2Client := ec2.NewFromConfig(cfg)
	ssmClient := ssm.NewFromConfig(cfg)

	// Create AMI registry
	registry := ami.NewRegistry(ssmClient, "")

	// Create AMI builder with configuration
	builderConfig := map[string]string{}
	if subnetID != "" {
		builderConfig["subnet_id"] = subnetID
	}
	if vpcID != "" {
		builderConfig["vpc_id"] = vpcID
	}
	builder, err := ami.NewBuilder(ec2Client, ssmClient, registry, builderConfig)
	if err != nil {
		return fmt.Errorf("failed to create AMI builder: %w", err)
	}
	fmt.Printf("DEBUG: Builder config: %+v\n", builderConfig)

	// Create template parser
	parser := ami.NewParser()

	// Find template file
	templateFile := filepath.Join("templates", templateName+".yml")
	if _, err := os.Stat(templateFile); os.IsNotExist(err) {
		// Try with .yaml extension
		templateFile = filepath.Join("templates", templateName+".yaml")
		if _, err := os.Stat(templateFile); os.IsNotExist(err) {
			return fmt.Errorf("template '%s' not found", templateName)
		}
	}

	// Parse template
	template, err := parser.ParseTemplateFile(templateFile)
	if err != nil {
		return fmt.Errorf("failed to parse template: %w", err)
	}

	// Create build request
	buildRequest := ami.BuildRequest{
		TemplateName: templateName,
		Template:     *template,
		Region:       region,
		Architecture: architecture,
		DryRun:       dryRun,
		BuildID:      fmt.Sprintf("%s-%d", templateName, time.Now().Unix()),
		BuildType:    "manual",
		VpcID:        vpcID,
		SubnetID:     subnetID,
	}

	fmt.Printf("Building AMI for template '%s' in region %s (%s)\n", templateName, region, architecture)
	if dryRun {
		fmt.Println("Running in DRY RUN mode - no AMI will be created")
	}

	// Build the AMI
	if !dryRun {
		fmt.Println("Starting build... this may take several minutes")
	} else {
		fmt.Println("Starting dry run build... simulating steps without creating resources")
	}
	
	result, err := builder.BuildAMI(context.TODO(), buildRequest)
	if err != nil {
		return fmt.Errorf("AMI build failed: %w", err)
	}

	// Print build result summary
	if result.Status == "success" {
		// No need for additional output - detailed progress is already shown during build
	} else {
		fmt.Println("\n❌ AMI build failed!")
		fmt.Printf("Error: %s\n", result.ErrorMessage)
	}

	// Print build logs if available
	if result.Logs != "" {
		logFile := fmt.Sprintf("%s-build.log", result.TemplateName)
		if err := os.WriteFile(logFile, []byte(result.Logs), 0644); err != nil {
			fmt.Printf("Warning: Failed to write build logs to %s: %v\n", logFile, err)
		} else {
			fmt.Printf("Full build logs saved to %s\n", logFile)
		}
	}

	return nil
}

// handleAMIList lists available AMIs
func (a *App) handleAMIList(args []string) error {
	cmdArgs := parseCmdArgs(args)

	// Parse command line arguments
	region := cmdArgs["region"]
	templateName := ""

	if len(args) > 0 && !strings.HasPrefix(args[0], "--") {
		templateName = args[0]
	}

	if region == "" {
		region = os.Getenv("AWS_REGION")
		if region == "" {
			region = "us-east-1" // Default
		}
	}

	// Initialize AWS clients
	cfg, err := config.LoadDefaultConfig(context.TODO(), config.WithRegion(region))
	if err != nil {
		return fmt.Errorf("failed to load AWS config: %w", err)
	}

	ssmClient := ssm.NewFromConfig(cfg)

	// Create AMI registry
	registry := ami.NewRegistry(ssmClient, "")

	if templateName != "" {
		// List AMIs for specific template
		amis, err := registry.ListTemplateAMIs(context.TODO(), templateName)
		if err != nil {
			return fmt.Errorf("failed to list AMIs: %w", err)
		}

		if len(amis) == 0 {
			fmt.Printf("No AMIs found for template '%s'\n", templateName)
			return nil
		}

		fmt.Printf("AMIs for template '%s':\n", templateName)
		for _, ami := range amis {
			fmt.Printf("- %s (%s, %s) - Created %s\n", ami.AMIID, ami.Region, ami.Architecture, ami.BuildDate.Format("2006-01-02 15:04:05"))
		}
	} else {
		// List all templates
		templates, err := registry.ListTemplates(context.TODO())
		if err != nil {
			return fmt.Errorf("failed to list templates: %w", err)
		}

		if len(templates) == 0 {
			fmt.Println("No AMI templates found in registry")
			return nil
		}

		fmt.Println("Available AMI templates:")
		for _, template := range templates {
			fmt.Printf("- %s\n", template)
		}
		fmt.Println("\nUse 'cws ami list <template>' to see AMIs for a specific template")
	}

	return nil
}

// handleAMIValidate validates a template without building an AMI
func (a *App) handleAMIValidate(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("missing template name")
	}

	templateName := args[0]

	// Create template parser
	parser := ami.NewParser()

	// Find template file
	templateFile := filepath.Join("templates", templateName+".yml")
	if _, err := os.Stat(templateFile); os.IsNotExist(err) {
		// Try with .yaml extension
		templateFile = filepath.Join("templates", templateName+".yaml")
		if _, err := os.Stat(templateFile); os.IsNotExist(err) {
			return fmt.Errorf("template '%s' not found", templateName)
		}
	}

	fmt.Printf("Validating template '%s'...\n", templateName)

	// Parse template
	template, err := parser.ParseTemplateFile(templateFile)
	if err != nil {
		return fmt.Errorf("❌ Template validation failed: %w", err)
	}

	// Check required fields
	if template.Name == "" {
		return fmt.Errorf("❌ Template validation failed: name is required")
	}

	if template.Base == "" {
		return fmt.Errorf("❌ Template validation failed: base image is required")
	}

	if len(template.BuildSteps) == 0 {
		return fmt.Errorf("❌ Template validation failed: at least one build step is required")
	}

	// Check build steps
	for i, step := range template.BuildSteps {
		if step.Name == "" {
			return fmt.Errorf("❌ Template validation failed: build step %d requires a name", i+1)
		}
		if step.Script == "" {
			return fmt.Errorf("❌ Template validation failed: build step '%s' requires a script", step.Name)
		}
	}

	fmt.Println("\n✅ Template validation successful!")
	fmt.Printf("Name: %s\n", template.Name)
	fmt.Printf("Base: %s\n", template.Base)
	fmt.Printf("Description: %s\n", template.Description)
	fmt.Printf("Build steps: %d\n", len(template.BuildSteps))
	fmt.Printf("Validation tests: %d\n", len(template.Validation))

	return nil
}

// handleAMIPublish updates the registry with a new AMI
func (a *App) handleAMIPublish(args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("usage: cws ami publish <template> <ami-id>")
	}

	templateName := args[0]
	amiID := args[1]
	cmdArgs := parseCmdArgs(args[2:])

	// Parse command line arguments
	region := cmdArgs["region"]
	architecture := cmdArgs["arch"]

	if region == "" {
		region = os.Getenv("AWS_REGION")
		if region == "" {
			region = "us-east-1" // Default
		}
	}

	if architecture == "" {
		architecture = "x86_64" // Default
	}

	// Initialize AWS clients
	cfg, err := config.LoadDefaultConfig(context.TODO(), config.WithRegion(region))
	if err != nil {
		return fmt.Errorf("failed to load AWS config: %w", err)
	}

	ssmClient := ssm.NewFromConfig(cfg)

	// Create AMI registry
	registry := ami.NewRegistry(ssmClient, "")

	// Create a build result to publish
	result := &ami.BuildResult{
		TemplateID:    fmt.Sprintf("%s-%d", templateName, time.Now().Unix()),
		TemplateName:  templateName,
		Region:        region,
		Architecture:  architecture,
		AMIID:         amiID,
		Status:        "manual",
		BuildTime:     time.Now(),
	}

	// Publish to registry
	err = registry.PublishAMI(context.TODO(), result)
	if err != nil {
		return fmt.Errorf("failed to publish AMI: %w", err)
	}

	fmt.Printf("✅ AMI %s published to registry for template '%s' (%s, %s)\n", amiID, templateName, region, architecture)
	return nil
}

// Helper function to parse command line arguments
func parseCmdArgs(args []string) map[string]string {
	result := make(map[string]string)
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if strings.HasPrefix(arg, "--") {
			// Remove leading dashes
			key := arg[2:]
			value := ""

			// Check if next arg is a value (not a flag)
			if i+1 < len(args) && !strings.HasPrefix(args[i+1], "--") {
				value = args[i+1]
				i++ // Skip value in next iteration
			} else {
				// Flag without value
				value = "true"
			}

			fmt.Printf("DEBUG: Parsed arg '%s' = '%s'\n", key, value)
			result[key] = value
		}
	}
	return result
}