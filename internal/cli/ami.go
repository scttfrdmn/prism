package cli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	ec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
	"github.com/scttfrdmn/prism/pkg/ami"
	"github.com/scttfrdmn/prism/pkg/types"
)

// AMI processes AMI-related commands
func (a *App) AMI(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("missing AMI command (build, list, validate, publish, save, resolve, test, costs, preview, create, status, cleanup, delete, snapshot)")
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
	case "save":
		return a.handleAMISave(subargs)
	// Universal AMI System commands (Phase 5.1 Week 2)
	case "resolve":
		return a.handleAMIResolve(subargs)
	case "test":
		return a.handleAMITest(subargs)
	case "costs":
		return a.handleAMICosts(subargs)
	case "preview":
		return a.handleAMIPreview(subargs)
	// AMI Creation commands (Phase 5.1 AMI Creation)
	case "create":
		return a.handleAMICreate(subargs)
	case "status":
		return a.handleAMIStatus(subargs)
	// AMI Lifecycle Management commands
	case "cleanup":
		return a.handleAMICleanup(subargs)
	case "delete":
		return a.handleAMIDelete(subargs)
	case "snapshot":
		return a.handleAMISnapshot(subargs)
	// AMI Freshness Checking command (v0.5.4 - Universal Version System)
	case "check-freshness":
		return a.handleAMICheckFreshness(subargs)
	default:
		return fmt.Errorf("unknown AMI command: %s", subcommand)
	}
}

// handleAMIBuild builds a new AMI from a template
// handleAMIBuild handles AMI build commands using Command Pattern (SOLID: Single Responsibility)
func (a *App) handleAMIBuild(args []string) error {
	// Create and execute AMI build command
	buildCmd := NewAMIBuildCommand()
	return buildCmd.Execute(args)
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
	ctx := context.Background()
	cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(region))
	if err != nil {
		return fmt.Errorf("failed to load AWS config: %w", err)
	}

	ssmClient := ssm.NewFromConfig(cfg)

	// Create AMI registry
	registry := ami.NewRegistry(ssmClient, "")

	if templateName != "" {
		// List AMIs for specific template
		amis, err := registry.ListTemplateAMIs(ctx, templateName)
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
		templates, err := registry.ListTemplates(ctx)
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
	ctx := context.Background()
	cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(region))
	if err != nil {
		return fmt.Errorf("failed to load AWS config: %w", err)
	}

	ssmClient := ssm.NewFromConfig(cfg)

	// Create AMI registry
	registry := ami.NewRegistry(ssmClient, "")

	// Create a build result to publish
	result := &ami.BuildResult{
		TemplateID:   fmt.Sprintf("%s-%d", templateName, time.Now().Unix()),
		TemplateName: templateName,
		Region:       region,
		Architecture: architecture,
		AMIID:        amiID,
		Status:       "manual",
		BuildTime:    time.Now(),
	}

	// Publish to registry
	err = registry.PublishAMI(ctx, result)
	if err != nil {
		return fmt.Errorf("failed to publish AMI: %w", err)
	}

	fmt.Printf("✅ AMI %s published to registry for template '%s' (%s, %s)\n", amiID, templateName, region, architecture)
	return nil
}

// handleAMISave saves a running instance as a new AMI template using Command Pattern (SOLID: Single Responsibility)
func (a *App) handleAMISave(args []string) error {
	// Create and execute AMI save command
	saveCmd := NewAMISaveCommand(a.apiClient)
	return saveCmd.Execute(args)
}

// discoverDefaultVPC finds the default VPC in the current region
func discoverDefaultVPC(ctx context.Context, client *ec2.Client) (string, error) {
	result, err := client.DescribeVpcs(ctx, &ec2.DescribeVpcsInput{
		Filters: []ec2types.Filter{
			{
				Name:   aws.String("is-default"),
				Values: []string{"true"},
			},
		},
	})
	if err != nil {
		return "", fmt.Errorf("failed to describe VPCs: %w", err)
	}

	if len(result.Vpcs) == 0 {
		return "", fmt.Errorf("no default VPC found - please create one or specify --vpc")
	}

	return *result.Vpcs[0].VpcId, nil
}

// discoverPublicSubnet finds a public subnet in the specified VPC
func discoverPublicSubnet(ctx context.Context, client *ec2.Client, vpcID string) (string, error) {
	// Get all subnets in the VPC
	result, err := client.DescribeSubnets(ctx, &ec2.DescribeSubnetsInput{
		Filters: []ec2types.Filter{
			{
				Name:   aws.String("vpc-id"),
				Values: []string{vpcID},
			},
		},
	})
	if err != nil {
		return "", fmt.Errorf("failed to describe subnets: %w", err)
	}

	if len(result.Subnets) == 0 {
		return "", fmt.Errorf("no subnets found in VPC %s", vpcID)
	}

	// Find a public subnet by checking route tables
	for _, subnet := range result.Subnets {
		isPublic, err := isSubnetPublic(ctx, client, *subnet.SubnetId)
		if err != nil {
			continue // Skip this subnet on error
		}
		if isPublic {
			return *subnet.SubnetId, nil
		}
	}

	// If no clearly public subnet found, use the first available subnet
	// (this handles cases where route table detection fails)
	return *result.Subnets[0].SubnetId, nil
}

// isSubnetPublic checks if a subnet is public by examining its route table
func isSubnetPublic(ctx context.Context, client *ec2.Client, subnetID string) (bool, error) {
	// Get route tables for this subnet
	result, err := client.DescribeRouteTables(ctx, &ec2.DescribeRouteTablesInput{
		Filters: []ec2types.Filter{
			{
				Name:   aws.String("association.subnet-id"),
				Values: []string{subnetID},
			},
		},
	})
	if err != nil {
		return false, err
	}

	// Check each route table for internet gateway routes
	for _, routeTable := range result.RouteTables {
		for _, route := range routeTable.Routes {
			// Look for route to 0.0.0.0/0 via internet gateway
			if route.DestinationCidrBlock != nil && *route.DestinationCidrBlock == "0.0.0.0/0" {
				if route.GatewayId != nil && strings.HasPrefix(*route.GatewayId, "igw-") {
					return true, nil
				}
			}
		}
	}

	return false, nil
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

			// Parse successful
			result[key] = value
		}
	}
	return result
}

// AMIBuildCommand handles AMI build operations using Command Pattern (SOLID: Single Responsibility)
type AMIBuildCommand struct {
	argParser      *AMIBuildArgParser
	networkService *NetworkDiscoveryService
	builderService *AMIBuilderService
}

// NewAMIBuildCommand creates a new AMI build command
func NewAMIBuildCommand() *AMIBuildCommand {
	return &AMIBuildCommand{
		argParser:      NewAMIBuildArgParser(),
		networkService: NewNetworkDiscoveryService(),
		builderService: NewAMIBuilderService(),
	}
}

// Execute executes the AMI build command (Command Pattern)
func (c *AMIBuildCommand) Execute(args []string) error {
	// Parse arguments
	config, err := c.argParser.Parse(args)
	if err != nil {
		return err
	}

	// Auto-discover network resources if needed
	if err := c.networkService.DiscoverResources(config); err != nil {
		return err
	}

	// Execute the build
	return c.builderService.BuildAMI(config)
}

// AMIBuildConfig represents AMI build configuration (Single Responsibility)
type AMIBuildConfig struct {
	TemplateName string
	Region       string
	Architecture string
	DryRun       bool
	SubnetID     string
	VpcID        string
}

// AMIBuildArgParser parses AMI build arguments using Strategy Pattern (SOLID: Single Responsibility)
type AMIBuildArgParser struct{}

// NewAMIBuildArgParser creates a new argument parser
func NewAMIBuildArgParser() *AMIBuildArgParser {
	return &AMIBuildArgParser{}
}

// Parse parses command line arguments into configuration (Single Responsibility)
func (p *AMIBuildArgParser) Parse(args []string) (*AMIBuildConfig, error) {
	if len(args) == 0 {
		return nil, fmt.Errorf("missing template name")
	}

	templateName := args[0]
	cmdArgs := parseCmdArgs(args[1:])

	// Apply defaults and parse arguments
	config := &AMIBuildConfig{
		TemplateName: templateName,
		Region:       p.parseRegion(cmdArgs),
		Architecture: p.parseArchitecture(cmdArgs),
		DryRun:       cmdArgs["dry-run"] != "",
		SubnetID:     cmdArgs["subnet"],
		VpcID:        cmdArgs["vpc"],
	}

	return config, nil
}

// parseRegion parses region with fallback (Single Responsibility)
func (p *AMIBuildArgParser) parseRegion(cmdArgs map[string]string) string {
	if region := cmdArgs["region"]; region != "" {
		return region
	}
	if region := os.Getenv("AWS_REGION"); region != "" {
		return region
	}
	return "us-east-1" // Default
}

// parseArchitecture parses architecture with default (Single Responsibility)
func (p *AMIBuildArgParser) parseArchitecture(cmdArgs map[string]string) string {
	if arch := cmdArgs["arch"]; arch != "" {
		return arch
	}
	return "x86_64" // Default
}

// NetworkDiscoveryService handles VPC/subnet discovery using Strategy Pattern (SOLID: Single Responsibility)
type NetworkDiscoveryService struct{}

// NewNetworkDiscoveryService creates a new network discovery service
func NewNetworkDiscoveryService() *NetworkDiscoveryService {
	return &NetworkDiscoveryService{}
}

// DiscoverResources auto-discovers VPC and subnet if needed (Single Responsibility)
func (s *NetworkDiscoveryService) DiscoverResources(buildConfig *AMIBuildConfig) error {
	if buildConfig.DryRun || (buildConfig.VpcID != "" && buildConfig.SubnetID != "") {
		return nil // Skip discovery if dry run or already configured
	}

	fmt.Printf("🔍 Auto-discovering default VPC and subnet...\n")

	// Initialize AWS client for discovery
	ctx := context.Background()
	cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(buildConfig.Region))
	if err != nil {
		return fmt.Errorf("failed to load AWS config: %w", err)
	}
	discoveryClient := ec2.NewFromConfig(cfg)

	// Discover VPC if needed
	if buildConfig.VpcID == "" {
		if buildConfig.VpcID, err = discoverDefaultVPC(ctx, discoveryClient); err != nil {
			return fmt.Errorf("failed to discover default VPC (you can specify with --vpc): %w", err)
		}
		fmt.Printf("   ✅ Using default VPC: %s\n", buildConfig.VpcID)
	}

	// Discover subnet if needed
	if buildConfig.SubnetID == "" {
		if buildConfig.SubnetID, err = discoverPublicSubnet(ctx, discoveryClient, buildConfig.VpcID); err != nil {
			return fmt.Errorf("failed to discover public subnet in VPC %s (you can specify with --subnet): %w", buildConfig.VpcID, err)
		}
		fmt.Printf("   ✅ Using public subnet: %s\n", buildConfig.SubnetID)
	}

	return nil
}

// AMIBuilderService handles AMI building operations using Strategy Pattern (SOLID: Single Responsibility)
type AMIBuilderService struct{}

// NewAMIBuilderService creates a new AMI builder service
func NewAMIBuilderService() *AMIBuilderService {
	return &AMIBuilderService{}
}

// BuildAMI builds the AMI using the configuration (Single Responsibility)
func (s *AMIBuilderService) BuildAMI(buildConfig *AMIBuildConfig) error {
	ctx := context.Background()

	// Setup AWS clients and builder
	builder, err := s.createBuilder(ctx, buildConfig)
	if err != nil {
		return err
	}

	// Parse template
	template, err := s.parseTemplate(buildConfig.TemplateName)
	if err != nil {
		return err
	}

	// Create and execute build request
	buildRequest := s.createBuildRequest(buildConfig, template)
	return s.executeBuild(ctx, builder, buildRequest)
}

// createBuilder creates the AMI builder (Single Responsibility)
func (s *AMIBuilderService) createBuilder(ctx context.Context, buildConfig *AMIBuildConfig) (*ami.Builder, error) {
	cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(buildConfig.Region))
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	ec2Client := ec2.NewFromConfig(cfg)
	ssmClient := ssm.NewFromConfig(cfg)
	registry := ami.NewRegistry(ssmClient, "")

	builderConfig := map[string]string{
		"subnet_id": buildConfig.SubnetID,
		"vpc_id":    buildConfig.VpcID,
	}

	return ami.NewBuilder(ec2Client, ssmClient, registry, builderConfig)
}

// parseTemplate parses the template file (Single Responsibility)
func (s *AMIBuilderService) parseTemplate(templateName string) (*ami.Template, error) {
	parser := ami.NewParser()

	// Find template file
	templateFile := filepath.Join("templates", templateName+".yml")
	if _, err := os.Stat(templateFile); os.IsNotExist(err) {
		templateFile = filepath.Join("templates", templateName+".yaml")
		if _, err := os.Stat(templateFile); os.IsNotExist(err) {
			return nil, fmt.Errorf("template '%s' not found", templateName)
		}
	}

	return parser.ParseTemplateFile(templateFile)
}

// createBuildRequest creates the build request (Single Responsibility)
func (s *AMIBuilderService) createBuildRequest(buildConfig *AMIBuildConfig, template *ami.Template) ami.BuildRequest {
	return ami.BuildRequest{
		TemplateName: buildConfig.TemplateName,
		Template:     *template,
		Region:       buildConfig.Region,
		Architecture: buildConfig.Architecture,
		DryRun:       buildConfig.DryRun,
		BuildID:      fmt.Sprintf("%s-%d", buildConfig.TemplateName, time.Now().Unix()),
		BuildType:    "manual",
		VpcID:        buildConfig.VpcID,
		SubnetID:     buildConfig.SubnetID,
	}
}

// executeBuild executes the build and handles results (Single Responsibility)
func (s *AMIBuilderService) executeBuild(ctx context.Context, builder *ami.Builder, buildRequest ami.BuildRequest) error {
	// Execute build
	result, err := builder.BuildAMI(ctx, buildRequest)
	if err != nil {
		return fmt.Errorf("AMI build failed: %w", err)
	}

	// Handle results
	return s.handleBuildResult(result)
}

// handleBuildResult processes and logs build results (Single Responsibility)
func (s *AMIBuilderService) handleBuildResult(result *ami.BuildResult) error {
	if result.Status != "success" {
		fmt.Println("\n❌ AMI build failed!")
		fmt.Printf("Error: %s\n", result.ErrorMessage)
	}

	// Save build logs if available
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

// AMI Save Command Pattern Implementation

// AMISaveCommand handles AMI save operations using Command Pattern (SOLID: Single Responsibility)
type AMISaveCommand struct {
	argParser           *AMISaveArgParser
	instanceService     *InstanceValidationService
	builderService      *AMISaveBuilderService
	confirmationService *AMISaveConfirmationService
	apiClient           interface{} // API client for instance lookups
}

// NewAMISaveCommand creates a new AMI save command
func NewAMISaveCommand(apiClient interface{}) *AMISaveCommand {
	return &AMISaveCommand{
		argParser:           NewAMISaveArgParser(),
		instanceService:     NewInstanceValidationService(apiClient),
		builderService:      NewAMISaveBuilderService(),
		confirmationService: NewAMISaveConfirmationService(),
		apiClient:           apiClient,
	}
}

// Execute executes the AMI save command (Command Pattern)
func (c *AMISaveCommand) Execute(args []string) error {
	// Parse arguments
	saveConfig, err := c.argParser.Parse(args)
	if err != nil {
		return err
	}

	// Validate instance
	instance, err := c.instanceService.ValidateInstance(saveConfig)
	if err != nil {
		return err
	}

	// Display confirmation and get user approval
	if !c.confirmationService.ConfirmSave(saveConfig, instance) {
		fmt.Println("Operation cancelled")
		return nil
	}

	// Execute the save
	return c.builderService.SaveInstanceAsAMI(saveConfig, instance)
}

// AMISaveConfig represents AMI save configuration (Single Responsibility)
type AMISaveConfig struct {
	InstanceName  string
	TemplateName  string
	Description   string
	Region        string
	ProjectID     string
	Public        bool
	CopyToRegions []string
}

// AMISaveArgParser parses AMI save arguments using Strategy Pattern (SOLID: Single Responsibility)
type AMISaveArgParser struct{}

// NewAMISaveArgParser creates a new AMI save argument parser
func NewAMISaveArgParser() *AMISaveArgParser {
	return &AMISaveArgParser{}
}

// Parse parses command line arguments into save configuration (Single Responsibility)
func (p *AMISaveArgParser) Parse(args []string) (*AMISaveConfig, error) {
	if len(args) < 2 {
		return nil, fmt.Errorf("usage: cws ami save <workspace-name> <template-name> [options]")
	}

	instanceName := args[0]
	templateName := args[1]
	cmdArgs := parseCmdArgs(args[2:])

	// Parse arguments using helper methods
	config := &AMISaveConfig{
		InstanceName:  instanceName,
		TemplateName:  templateName,
		Description:   p.parseDescription(cmdArgs, instanceName),
		Region:        p.parseRegion(cmdArgs),
		ProjectID:     cmdArgs["project"],
		Public:        cmdArgs["public"] != "",
		CopyToRegions: p.parseCopyToRegions(cmdArgs),
	}

	return config, nil
}

// parseDescription parses description with fallback (Single Responsibility)
func (p *AMISaveArgParser) parseDescription(cmdArgs map[string]string, instanceName string) string {
	if description := cmdArgs["description"]; description != "" {
		return description
	}
	return fmt.Sprintf("Custom template saved from workspace %s", instanceName)
}

// parseRegion parses region with fallback (Single Responsibility)
func (p *AMISaveArgParser) parseRegion(cmdArgs map[string]string) string {
	if region := cmdArgs["region"]; region != "" {
		return region
	}
	if region := os.Getenv("AWS_REGION"); region != "" {
		return region
	}
	return "us-east-1" // Default
}

// parseCopyToRegions parses copy-to-regions list (Single Responsibility)
func (p *AMISaveArgParser) parseCopyToRegions(cmdArgs map[string]string) []string {
	if regions := cmdArgs["copy-to-regions"]; regions != "" {
		return strings.Split(regions, ",")
	}
	return []string{}
}

// InstanceValidationService handles instance validation using Strategy Pattern (SOLID: Single Responsibility)
type InstanceValidationService struct {
	apiClient interface{} // API client for instance lookups
}

// NewInstanceValidationService creates a new instance validation service
func NewInstanceValidationService(apiClient interface{}) *InstanceValidationService {
	return &InstanceValidationService{
		apiClient: apiClient,
	}
}

// ValidateInstance validates the instance exists and is in running state (Single Responsibility)
func (s *InstanceValidationService) ValidateInstance(saveConfig *AMISaveConfig) (*types.Instance, error) {
	ctx := context.Background()

	// Check daemon is running
	if pingable, ok := s.apiClient.(interface{ Ping(context.Context) error }); ok {
		if err := pingable.Ping(ctx); err != nil {
			return nil, fmt.Errorf("daemon not running. Start with: cws daemon start")
		}
	}

	// Get instance information from daemon API
	if lister, ok := s.apiClient.(interface {
		ListInstances(context.Context) (*types.ListResponse, error)
	}); ok {
		instances, err := lister.ListInstances(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to get instance list: %w", err)
		}

		// Find the instance
		for _, inst := range instances.Instances {
			if inst.Name == saveConfig.InstanceName {
				// Validate instance state
				if inst.State != "running" {
					return nil, fmt.Errorf("workspace '%s' must be running to save as AMI (current state: %s)", saveConfig.InstanceName, inst.State)
				}
				return &inst, nil
			}
		}

		return nil, fmt.Errorf("workspace '%s' not found", saveConfig.InstanceName)
	}

	return nil, fmt.Errorf("API client does not support instance listing")
}

// AMISaveConfirmationService handles user confirmation using Strategy Pattern (SOLID: Single Responsibility)
type AMISaveConfirmationService struct{}

// NewAMISaveConfirmationService creates a new confirmation service
func NewAMISaveConfirmationService() *AMISaveConfirmationService {
	return &AMISaveConfirmationService{}
}

// ConfirmSave displays save details and gets user confirmation (Single Responsibility)
func (s *AMISaveConfirmationService) ConfirmSave(saveConfig *AMISaveConfig, instance *types.Instance) bool {
	// Display save information
	fmt.Printf("💾 Saving workspace '%s' as template '%s'\n", saveConfig.InstanceName, saveConfig.TemplateName)
	fmt.Printf("📍 Instance ID: %s\n", instance.ID)
	fmt.Printf("🏷️  Description: %s\n", saveConfig.Description)
	if len(saveConfig.CopyToRegions) > 0 {
		fmt.Printf("🌍 Will copy to regions: %s\n", strings.Join(saveConfig.CopyToRegions, ", "))
	}

	// Warning about temporary stop
	fmt.Printf("\n⚠️  WARNING: Workspace will be temporarily stopped to create a consistent AMI\n")
	fmt.Printf("   This ensures the AMI captures a clean state of the filesystem.\n")
	fmt.Printf("   The workspace will be automatically restarted after AMI creation.\n\n")

	// Get user confirmation
	fmt.Printf("Continue? (y/N): ")
	var response string
	_, _ = fmt.Scanln(&response)
	return strings.ToLower(response) == "y" || strings.ToLower(response) == "yes"
}

// AMISaveBuilderService handles AMI save operations using Strategy Pattern (SOLID: Single Responsibility)
type AMISaveBuilderService struct{}

// NewAMISaveBuilderService creates a new AMI save builder service
func NewAMISaveBuilderService() *AMISaveBuilderService {
	return &AMISaveBuilderService{}
}

// SaveInstanceAsAMI saves the instance as an AMI using the configuration (Single Responsibility)
func (s *AMISaveBuilderService) SaveInstanceAsAMI(saveConfig *AMISaveConfig, instance *types.Instance) error {
	ctx := context.Background()

	// Create AMI builder
	builder, err := s.createBuilder(ctx, saveConfig)
	if err != nil {
		return err
	}

	// Create save request
	saveRequest := s.createSaveRequest(saveConfig, instance)

	// Execute save and handle results
	return s.executeSaveAndDisplayResults(ctx, builder, saveRequest, saveConfig.TemplateName)
}

// createBuilder creates the AMI builder (Single Responsibility)
func (s *AMISaveBuilderService) createBuilder(ctx context.Context, saveConfig *AMISaveConfig) (*ami.Builder, error) {
	cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(saveConfig.Region))
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	ec2Client := ec2.NewFromConfig(cfg)
	ssmClient := ssm.NewFromConfig(cfg)
	registry := ami.NewRegistry(ssmClient, "")

	builderConfig := map[string]string{}
	builder, err := ami.NewBuilder(ec2Client, ssmClient, registry, builderConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create AMI builder: %w", err)
	}

	return builder, nil
}

// createSaveRequest creates the AMI save request (Single Responsibility)
func (s *AMISaveBuilderService) createSaveRequest(saveConfig *AMISaveConfig, instance *types.Instance) ami.InstanceSaveRequest {
	return ami.InstanceSaveRequest{
		InstanceID:    instance.ID,
		InstanceName:  saveConfig.InstanceName,
		TemplateName:  saveConfig.TemplateName,
		Description:   saveConfig.Description,
		CopyToRegions: saveConfig.CopyToRegions,
		ProjectID:     saveConfig.ProjectID,
		Public:        saveConfig.Public,
		Tags: map[string]string{
			"Name":                  saveConfig.TemplateName,
			"PrismTemplate":         saveConfig.TemplateName,
			"PrismSavedFrom":        saveConfig.InstanceName,
			"PrismOriginalTemplate": instance.Template,
		},
	}
}

// executeSaveAndDisplayResults executes the save and displays results (Single Responsibility)
func (s *AMISaveBuilderService) executeSaveAndDisplayResults(ctx context.Context, builder *ami.Builder, saveRequest ami.InstanceSaveRequest, templateName string) error {
	// Create AMI from instance
	result, err := builder.CreateAMIFromInstance(ctx, saveRequest)
	if err != nil {
		return fmt.Errorf("failed to save instance as AMI: %w", err)
	}

	// Display results
	return s.displaySaveResults(result, templateName)
}

// displaySaveResults displays the save operation results (Single Responsibility)
func (s *AMISaveBuilderService) displaySaveResults(result *ami.BuildResult, templateName string) error {
	fmt.Printf("\n🎉 Successfully saved workspace as AMI!\n")
	fmt.Printf("📸 AMI ID: %s\n", result.AMIID)
	fmt.Printf("🕒 Build time: %s\n", result.BuildDuration)

	if len(result.CopiedAMIs) > 0 {
		fmt.Printf("\n🌍 AMI copied to additional regions:\n")
		for region, amiID := range result.CopiedAMIs {
			fmt.Printf("   %s: %s\n", region, amiID)
		}
	}

	fmt.Printf("\n✨ Template '%s' is now available for launching new workspaces:\n", templateName)
	fmt.Printf("   cws launch %s my-new-instance\n", templateName)

	return nil
}

// Universal AMI System CLI handlers (Phase 5.1 Week 2)

// handleAMIResolve resolves AMI for a template using the Universal AMI System
func (a *App) handleAMIResolve(args []string) error {
	// Validate arguments and parse command
	templateName, cmdArgs, err := a.parseAMIResolveArgs(args)
	if err != nil {
		return err
	}

	// Prepare API parameters
	params := a.buildAMIResolveParams(cmdArgs)

	// Execute AMI resolution
	response, err := a.executeAMIResolution(templateName, params)
	if err != nil {
		return err
	}

	// Display resolution results
	a.displayAMIResolutionResults(templateName, cmdArgs, response)

	return nil
}

// parseAMIResolveArgs validates and parses AMI resolve command arguments
func (a *App) parseAMIResolveArgs(args []string) (string, map[string]string, error) {
	if len(args) == 0 {
		return "", nil, fmt.Errorf("usage: cws ami resolve <template-name> [--details] [--region <region>]")
	}

	templateName := args[0]
	cmdArgs := parseCmdArgs(args[1:])

	return templateName, cmdArgs, nil
}

// buildAMIResolveParams constructs API parameters from command arguments
func (a *App) buildAMIResolveParams(cmdArgs map[string]string) map[string]interface{} {
	params := make(map[string]interface{})

	if cmdArgs["details"] == "true" {
		params["details"] = true
	}
	if region := cmdArgs["region"]; region != "" {
		params["region"] = region
	}

	return params
}

// executeAMIResolution makes the API call to resolve AMI
func (a *App) executeAMIResolution(templateName string, params map[string]interface{}) (map[string]interface{}, error) {
	response, err := a.apiClient.ResolveAMI(a.ctx, templateName, params)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve AMI: %w", err)
	}

	return response, nil
}

// displayAMIResolutionResults formats and displays the AMI resolution results
func (a *App) displayAMIResolutionResults(templateName string, cmdArgs map[string]string, response map[string]interface{}) {
	// Display header information
	a.displayAMIHeader(templateName, cmdArgs, response)

	// Display AMI details
	a.displayAMIDetails(cmdArgs, response)

	// Display performance and cost information
	a.displayPerformanceAndCostInfo(response)

	// Display warnings and additional details
	a.displayWarningsAndExtras(cmdArgs, response)
}

// displayAMIHeader shows the header information for AMI resolution
func (a *App) displayAMIHeader(templateName string, cmdArgs map[string]string, response map[string]interface{}) {
	fmt.Printf("🔍 AMI Resolution for template '%s'\n\n", templateName)
	if region := cmdArgs["region"]; region != "" {
		fmt.Printf("📍 Target region: %s\n", region)
	}
	fmt.Printf("🏗️  Resolution method: %s\n", getString(response, "resolution_method"))
}

// displayAMIDetails shows basic and detailed AMI information
func (a *App) displayAMIDetails(cmdArgs map[string]string, response map[string]interface{}) {
	amiID := getString(response, "ami_id")
	if amiID == "" {
		return
	}

	// Basic AMI information
	fmt.Printf("📀 AMI ID: %s\n", amiID)
	fmt.Printf("🏷️  AMI Name: %s\n", getString(response, "ami_name"))
	fmt.Printf("🏛️  Architecture: %s\n", getString(response, "ami_architecture"))

	// Detailed AMI information
	if cmdArgs["details"] == "true" {
		a.displayDetailedAMIInfo(response)
	}
}

// displayDetailedAMIInfo shows comprehensive AMI details
func (a *App) displayDetailedAMIInfo(response map[string]interface{}) {
	details := getMap(response, "ami_details")
	if details == nil {
		return
	}

	fmt.Printf("\n📋 AMI Details:\n")
	fmt.Printf("   Created: %s\n", getString(details, "creation_date"))
	fmt.Printf("   Owner: %s\n", getString(details, "owner_id"))
	fmt.Printf("   Platform: %s\n", getString(details, "platform"))
	fmt.Printf("   Virtualization: %s\n", getString(details, "virtualization"))
	fmt.Printf("   Root Device: %s\n", getString(details, "root_device"))

	if cost := getFloat(details, "marketplace_cost"); cost > 0 {
		fmt.Printf("   Marketplace Cost: $%.4f/hour\n", cost)
	}
}

// displayPerformanceAndCostInfo shows launch time and cost information
func (a *App) displayPerformanceAndCostInfo(response map[string]interface{}) {
	// Launch time estimate
	if estimate := getInt(response, "launch_time_estimate_seconds"); estimate > 0 {
		fmt.Printf("⚡ Launch time estimate: %d seconds\n", estimate)
	}

	// Cost savings information
	a.displayCostSavingsInfo(response)
}

// displayCostSavingsInfo shows cost savings or additional costs
func (a *App) displayCostSavingsInfo(response map[string]interface{}) {
	savings := getFloat(response, "cost_savings")
	if savings == 0 {
		return
	}

	if savings > 0 {
		fmt.Printf("💰 Cost savings: $%.4f/hour\n", savings)
	} else {
		fmt.Printf("💸 Additional cost: $%.4f/hour\n", -savings)
	}
}

// displayWarningsAndExtras shows warnings and additional detailed information
func (a *App) displayWarningsAndExtras(cmdArgs map[string]string, response map[string]interface{}) {
	// Display warnings
	if warning := getString(response, "warning"); warning != "" {
		fmt.Printf("⚠️  Warning: %s\n", warning)
	}

	// Display fallback chain in details mode
	if cmdArgs["details"] == "true" {
		if chain := getStringSlice(response, "fallback_chain"); len(chain) > 0 {
			fmt.Printf("\n🔄 Fallback chain: %s\n", strings.Join(chain, " → "))
		}
	}
}

// handleAMITest tests AMI availability across regions for a template
func (a *App) handleAMITest(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: cws ami test <template-name> [--regions <region1,region2,...>]")
	}

	templateName := args[0]
	cmdArgs := parseCmdArgs(args[1:])

	// Prepare request body
	request := map[string]interface{}{
		"template_name": templateName,
	}

	if regions := cmdArgs["regions"]; regions != "" {
		request["regions"] = strings.Split(regions, ",")
	}

	// Make API call
	response, err := a.apiClient.TestAMIAvailability(a.ctx, request)
	if err != nil {
		return fmt.Errorf("failed to test AMI availability: %w", err)
	}

	// Display results
	fmt.Printf("🧪 AMI Availability Test for template '%s'\n\n", templateName)

	overallStatus := getString(response, "overall_status")
	totalRegions := getInt(response, "total_regions")
	availableRegions := getInt(response, "available_regions")

	fmt.Printf("📊 Overall Status: %s\n", overallStatus)
	fmt.Printf("🌍 Available in %d/%d regions\n\n", availableRegions, totalRegions)

	// Display regional results
	if regionResults := getMap(response, "region_results"); regionResults != nil {
		fmt.Println("📍 Regional Results:")
		for region, resultData := range regionResults {
			if regionData := getMap(resultData, ""); regionData != nil {
				status := getString(regionData, "status")
				statusIcon := "❌"
				if status == "passed" {
					statusIcon = "✅"
				}

				fmt.Printf("   %s %s: %s", statusIcon, region, status)

				if amiID := getString(regionData, "ami"); amiID != "" {
					fmt.Printf(" (%s via %s)", amiID, getString(regionData, "resolution_method"))
				}

				if errorMsg := getString(regionData, "error"); errorMsg != "" {
					fmt.Printf(" - %s", errorMsg)
				}
				fmt.Println()
			}
		}
	}

	return nil
}

// handleAMICosts provides cost analysis for AMI vs script deployment
func (a *App) handleAMICosts(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: cws ami costs <template-name>")
	}

	templateName := args[0]

	// Make API call
	response, err := a.apiClient.GetAMICosts(a.ctx, templateName)
	if err != nil {
		return fmt.Errorf("failed to get cost analysis: %w", err)
	}

	// Display results
	fmt.Printf("💰 Cost Analysis for template '%s'\n\n", templateName)

	recommendation := getString(response, "recommendation")
	reasoning := getString(response, "reasoning")

	fmt.Printf("🎯 Recommendation: %s\n", recommendation)
	fmt.Printf("💡 Reasoning: %s\n\n", reasoning)

	// AMI costs
	fmt.Println("📀 AMI Deployment:")
	fmt.Printf("   Setup cost: $%.4f\n", getFloat(response, "ami_setup_cost"))
	fmt.Printf("   Storage cost: $%.4f/month\n", getFloat(response, "ami_storage_cost"))
	fmt.Printf("   Launch cost: $%.4f/hour\n", getFloat(response, "ami_launch_cost"))

	// Script costs
	fmt.Println("\n📜 Script Deployment:")
	fmt.Printf("   Setup cost: $%.4f\n", getFloat(response, "script_setup_cost"))
	fmt.Printf("   Setup time: %d minutes\n", getInt(response, "script_setup_time"))
	fmt.Printf("   Launch cost: $%.4f/hour\n", getFloat(response, "script_launch_cost"))

	// Savings analysis
	fmt.Println("\n📊 Savings Analysis:")
	fmt.Printf("   1-hour session: $%.4f savings\n", getFloat(response, "cost_savings_1_hour"))
	fmt.Printf("   8-hour session: $%.4f savings\n", getFloat(response, "cost_savings_8_hour"))
	fmt.Printf("   Break-even point: %.1f hours\n", getFloat(response, "break_even_point"))
	fmt.Printf("   Time savings: %d minutes\n", getInt(response, "time_savings"))

	return nil
}

// handleAMIPreview shows what would happen during AMI resolution without executing
func (a *App) handleAMIPreview(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: cws ami preview <template-name>")
	}

	templateName := args[0]

	// Make API call
	response, err := a.apiClient.PreviewAMIResolution(a.ctx, templateName)
	if err != nil {
		return fmt.Errorf("failed to preview AMI resolution: %w", err)
	}

	// Display results
	fmt.Printf("🔮 AMI Resolution Preview for template '%s'\n\n", templateName)

	fmt.Printf("📍 Target region: %s\n", getString(response, "target_region"))
	fmt.Printf("🏗️  Resolution method: %s\n", getString(response, "resolution_method"))

	if estimate := getInt(response, "launch_time_estimate_seconds"); estimate > 0 {
		fmt.Printf("⚡ Launch time estimate: %d seconds\n", estimate)
	}

	if chain := getStringSlice(response, "fallback_chain"); len(chain) > 0 {
		fmt.Printf("🔄 Fallback chain: %s\n", strings.Join(chain, " → "))
	}

	if warning := getString(response, "warning"); warning != "" {
		fmt.Printf("⚠️  Warning: %s\n", warning)
	}

	if errorMsg := getString(response, "error"); errorMsg != "" {
		fmt.Printf("❌ Error: %s\n", errorMsg)
	}

	fmt.Println("\n💡 This is a dry-run preview. No actual AMI resolution was performed.")

	return nil
}

// Helper functions for parsing API responses

func getString(data interface{}, key string) string {
	if m, ok := data.(map[string]interface{}); ok {
		if val, exists := m[key]; exists {
			if str, ok := val.(string); ok {
				return str
			}
		}
	}
	return ""
}

func getInt(data interface{}, key string) int {
	if m, ok := data.(map[string]interface{}); ok {
		if val, exists := m[key]; exists {
			if num, ok := val.(float64); ok {
				return int(num)
			}
		}
	}
	return 0
}

func getFloat(data interface{}, key string) float64 {
	if m, ok := data.(map[string]interface{}); ok {
		if val, exists := m[key]; exists {
			if num, ok := val.(float64); ok {
				return num
			}
		}
	}
	return 0.0
}

func getMap(data interface{}, key string) map[string]interface{} {
	if m, ok := data.(map[string]interface{}); ok {
		if key == "" {
			return m
		}
		if val, exists := m[key]; exists {
			if subMap, ok := val.(map[string]interface{}); ok {
				return subMap
			}
		}
	}
	return nil
}

func getStringSlice(data interface{}, key string) []string {
	if m, ok := data.(map[string]interface{}); ok {
		if val, exists := m[key]; exists {
			if slice, ok := val.([]interface{}); ok {
				result := make([]string, len(slice))
				for i, item := range slice {
					if str, ok := item.(string); ok {
						result[i] = str
					}
				}
				return result
			}
		}
	}
	return nil
}

func getFloat64(data interface{}, key string) float64 {
	if m, ok := data.(map[string]interface{}); ok {
		if val, exists := m[key]; exists {
			if num, ok := val.(float64); ok {
				return num
			}
		}
	}
	return 0
}

func getBool(data interface{}, key string) bool {
	if m, ok := data.(map[string]interface{}); ok {
		if val, exists := m[key]; exists {
			if b, ok := val.(bool); ok {
				return b
			}
		}
	}
	return false
}

func getSlice(data interface{}, key string) []interface{} {
	if m, ok := data.(map[string]interface{}); ok {
		if val, exists := m[key]; exists {
			if slice, ok := val.([]interface{}); ok {
				return slice
			}
		}
	}
	return nil
}

// AMI Creation Commands (Phase 5.1 Enhancement)

// handleAMICreate creates an AMI from a running instance
func (a *App) handleAMICreate(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: cws ami create <workspace-name> --name <ami-name> [--description <description>] [--template <template>] [--public] [--no-reboot]")
	}

	instanceName := args[0]
	cmdArgs := parseCmdArgs(args[1:])

	// Validate required parameters
	amiName := cmdArgs["name"]
	if amiName == "" {
		return fmt.Errorf("AMI name is required (use --name <ami-name>)")
	}

	// Prepare AMI creation request
	request := types.AMICreationRequest{
		InstanceID: instanceName,
		Name:       amiName,
		Public:     cmdArgs["public"] == "true",
		NoReboot:   cmdArgs["no-reboot"] == "true",
		Tags:       make(map[string]string),
	}

	// Add template name if provided
	if template := cmdArgs["template"]; template != "" {
		request.TemplateName = template
	} else {
		request.TemplateName = "custom"
	}

	// Add description with default
	if description := cmdArgs["description"]; description != "" {
		request.Description = description
	} else {
		request.Description = fmt.Sprintf("Custom AMI created from workspace %s", instanceName)
	}

	// Add tags
	if tags := cmdArgs["tags"]; tags != "" {
		for _, tag := range strings.Split(tags, ",") {
			parts := strings.Split(tag, "=")
			if len(parts) == 2 {
				request.Tags[parts[0]] = parts[1]
			}
		}
	}

	fmt.Printf("🚀 Creating AMI from workspace '%s'...\n\n", instanceName)

	// Make API call to create AMI
	response, err := a.apiClient.CreateAMI(a.ctx, request)
	if err != nil {
		return fmt.Errorf("failed to create AMI: %w", err)
	}

	// Display results
	fmt.Printf("✅ AMI creation initiated successfully!\n\n")
	fmt.Printf("🆔 AMI ID: %s\n", getString(response, "ami_id"))
	fmt.Printf("📝 Name: %s\n", getString(response, "name"))
	fmt.Printf("🏷️  Template: %s\n", getString(response, "template_name"))
	fmt.Printf("⚡ Status: %s\n", getString(response, "status"))
	fmt.Printf("⏱️  Estimated completion: %d minutes\n", getInt(response, "estimated_completion_minutes"))
	fmt.Printf("💰 Storage cost: $%.4f/month\n", getFloat64(response, "storage_cost"))

	fmt.Printf("\n💡 Check status with: cws ami status %s\n", getString(response, "ami_id"))

	return nil
}

// handleAMIStatus checks the status of AMI creation
func (a *App) handleAMIStatus(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: cws ami status <creation-id|ami-id>")
	}

	creationID := args[0]

	// Make API call to get status
	response, err := a.apiClient.GetAMIStatus(a.ctx, creationID)
	if err != nil {
		return fmt.Errorf("failed to get AMI status: %w", err)
	}

	// Display results
	fmt.Printf("📊 AMI Creation Status\n\n")
	fmt.Printf("🆔 AMI ID: %s\n", getString(response, "ami_id"))
	fmt.Printf("⚡ Status: %s\n", getString(response, "status"))

	progress := getInt(response, "progress")
	fmt.Printf("📈 Progress: %d%%\n", progress)

	// Show progress bar
	barWidth := 30
	filledWidth := int(float64(barWidth) * float64(progress) / 100.0)
	fmt.Print("   [")
	for i := 0; i < barWidth; i++ {
		if i < filledWidth {
			fmt.Print("█")
		} else {
			fmt.Print("░")
		}
	}
	fmt.Printf("]\n\n")

	fmt.Printf("⏱️  Elapsed time: %d minutes\n", getInt(response, "elapsed_time_minutes"))
	fmt.Printf("⏰ Estimated completion: %d minutes remaining\n", getInt(response, "estimated_completion_minutes"))
	fmt.Printf("💰 Storage cost: $%.4f/month\n", getFloat64(response, "storage_cost"))

	// Show completion message if done
	status := getString(response, "status")
	if status == "completed" {
		fmt.Printf("\n🎉 AMI creation completed successfully!\n")
		fmt.Printf("   Use this AMI by creating a template or launching directly\n")
	} else if status == "failed" {
		fmt.Printf("\n❌ AMI creation failed\n")
		if message := getString(response, "message"); message != "" {
			fmt.Printf("   Error: %s\n", message)
		}
	} else {
		fmt.Printf("\n⏳ AMI creation is still in progress...\n")
		fmt.Printf("   Check again in a few minutes\n")
	}

	return nil
}

// handleAMIList lists user's AMIs
func (a *App) handleAMIListUser(args []string) error {
	// Make API call to list user AMIs
	response, err := a.apiClient.ListUserAMIs(a.ctx)
	if err != nil {
		return fmt.Errorf("failed to list user AMIs: %w", err)
	}

	fmt.Printf("🖼️  Your Custom AMIs\n\n")

	amis := getSlice(response, "amis")
	if len(amis) == 0 {
		fmt.Printf("No custom AMIs found.\n")
		fmt.Printf("💡 Create one with: cws ami create <workspace-name> --name <ami-name>\n")
		return nil
	}

	for i, ami := range amis {
		amiMap := ami.(map[string]interface{})
		fmt.Printf("🖼️  AMI %d:\n", i+1)
		fmt.Printf("   🆔 ID: %s\n", getString(amiMap, "ami_id"))
		fmt.Printf("   📝 Name: %s\n", getString(amiMap, "name"))
		fmt.Printf("   📖 Description: %s\n", getString(amiMap, "description"))
		fmt.Printf("   🏗️  Architecture: %s\n", getString(amiMap, "architecture"))
		fmt.Printf("   📅 Created: %s\n", getString(amiMap, "creation_date"))

		if getBool(amiMap, "public") {
			fmt.Printf("   🌍 Visibility: Public\n")
		} else {
			fmt.Printf("   🔒 Visibility: Private\n")
		}

		fmt.Printf("\n")
	}

	fmt.Printf("💡 Use an AMI by creating a template or referencing in launch commands\n")

	return nil
}

// AMI Lifecycle Management Commands

// handleAMICleanup removes old and unused AMIs
func (a *App) handleAMICleanup(args []string) error {
	cmdArgs := parseCmdArgs(args)

	// Parse command line arguments
	dryRun := cmdArgs["dry-run"] != ""
	force := cmdArgs["force"] != ""
	maxAge := cmdArgs["max-age"]
	if maxAge == "" {
		maxAge = "30d" // Default to 30 days
	}

	// Display confirmation warning
	if !dryRun && !force {
		fmt.Printf("⚠️  AMI Cleanup Operation\n")
		fmt.Printf("This will identify and remove AMIs older than %s\n", maxAge)
		fmt.Printf("💡 Use --dry-run to preview which AMIs would be deleted\n")
		fmt.Printf("💡 Use --force to skip this confirmation\n\n")

		fmt.Printf("Continue? (y/N): ")
		var response string
		_, _ = fmt.Scanln(&response)
		if !(strings.ToLower(response) == "y" || strings.ToLower(response) == "yes") {
			fmt.Println("Operation cancelled")
			return nil
		}
	}

	// Prepare cleanup request
	request := map[string]interface{}{
		"max_age": maxAge,
		"dry_run": dryRun,
	}

	// Make API call
	response, err := a.apiClient.CleanupAMIs(a.ctx, request)
	if err != nil {
		return fmt.Errorf("failed to cleanup AMIs: %w", err)
	}

	// Display results
	fmt.Printf("🧹 AMI Cleanup Results\n\n")

	if dryRun {
		fmt.Printf("🔍 DRY RUN - No AMIs were actually deleted\n\n")
	}

	totalFound := getInt(response, "total_found")
	totalRemoved := getInt(response, "total_removed")
	costSavings := getFloat(response, "storage_savings_monthly")

	fmt.Printf("📊 Summary:\n")
	fmt.Printf("   AMIs found: %d\n", totalFound)
	if dryRun {
		fmt.Printf("   AMIs would be removed: %d\n", totalRemoved)
		fmt.Printf("   Estimated monthly savings: $%.2f\n", costSavings)
	} else {
		fmt.Printf("   AMIs removed: %d\n", totalRemoved)
		fmt.Printf("   Monthly storage savings: $%.2f\n", costSavings)
	}

	// Display detailed results if available
	if removedAMIs := getSlice(response, "removed_amis"); len(removedAMIs) > 0 {
		fmt.Printf("\n📀 %s AMIs:\n", map[bool]string{true: "Would remove", false: "Removed"}[dryRun])
		for _, ami := range removedAMIs {
			if amiMap := getMap(ami, ""); amiMap != nil {
				fmt.Printf("   • %s (%s) - Created %s\n",
					getString(amiMap, "ami_id"),
					getString(amiMap, "name"),
					getString(amiMap, "creation_date"))
			}
		}
	}

	if !dryRun && totalRemoved > 0 {
		fmt.Printf("\n✅ Cleanup completed successfully!\n")
	} else if dryRun {
		fmt.Printf("\n💡 Run without --dry-run to perform the actual cleanup\n")
	}

	return nil
}

// handleAMIDelete deletes a specific AMI by ID
func (a *App) handleAMIDelete(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: cws ami delete <ami-id> [--force] [--deregister-only]")
	}

	amiID := args[0]
	cmdArgs := parseCmdArgs(args[1:])
	force := cmdArgs["force"] != ""
	deregisterOnly := cmdArgs["deregister-only"] != ""

	// Display confirmation warning
	if !force {
		fmt.Printf("⚠️  AMI Deletion Warning\n")
		fmt.Printf("AMI ID: %s\n", amiID)
		if deregisterOnly {
			fmt.Printf("Operation: Deregister AMI (keep snapshots)\n")
		} else {
			fmt.Printf("Operation: Delete AMI and associated snapshots\n")
		}
		fmt.Printf("\n💡 This operation cannot be undone\n")
		fmt.Printf("💡 Use --force to skip this confirmation\n\n")

		fmt.Printf("Continue? (y/N): ")
		var response string
		_, _ = fmt.Scanln(&response)
		if !(strings.ToLower(response) == "y" || strings.ToLower(response) == "yes") {
			fmt.Println("Operation cancelled")
			return nil
		}
	}

	// Prepare deletion request
	request := map[string]interface{}{
		"ami_id":          amiID,
		"deregister_only": deregisterOnly,
	}

	fmt.Printf("🗑️  Deleting AMI %s...\n", amiID)

	// Make API call
	response, err := a.apiClient.DeleteAMI(a.ctx, request)
	if err != nil {
		return fmt.Errorf("failed to delete AMI: %w", err)
	}

	// Display results
	fmt.Printf("\n✅ AMI deletion completed successfully!\n\n")
	fmt.Printf("📋 Details:\n")
	fmt.Printf("   AMI ID: %s\n", getString(response, "ami_id"))
	fmt.Printf("   Status: %s\n", getString(response, "status"))

	if deletedSnapshots := getSlice(response, "deleted_snapshots"); len(deletedSnapshots) > 0 && !deregisterOnly {
		fmt.Printf("   Deleted snapshots: %d\n", len(deletedSnapshots))
		fmt.Printf("   Storage savings: $%.2f/month\n", getFloat(response, "storage_savings_monthly"))
	} else if deregisterOnly {
		fmt.Printf("   Snapshots: Preserved (deregister-only mode)\n")
	}

	return nil
}

// handleAMISnapshot manages AMI snapshots and creates AMIs from snapshots
func (a *App) handleAMISnapshot(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: cws ami snapshot <subcommand>\n" +
			"Subcommands:\n" +
			"  list                    List available snapshots\n" +
			"  create <instance-id>    Create snapshot from instance\n" +
			"  restore <snapshot-id>   Create AMI from snapshot\n" +
			"  delete <snapshot-id>    Delete a snapshot")
	}

	subcommand := args[0]
	subargs := args[1:]

	switch subcommand {
	case "list":
		return a.handleAMISnapshotList(subargs)
	case "create":
		return a.handleAMISnapshotCreate(subargs)
	case "restore":
		return a.handleAMISnapshotRestore(subargs)
	case "delete":
		return a.handleAMISnapshotDelete(subargs)
	default:
		return fmt.Errorf("unknown snapshot command: %s", subcommand)
	}
}

// handleAMISnapshotList lists available snapshots
func (a *App) handleAMISnapshotList(args []string) error {
	cmdArgs := parseCmdArgs(args)

	// Optional filters
	filters := make(map[string]interface{})
	if instanceID := cmdArgs["instance-id"]; instanceID != "" {
		filters["instance_id"] = instanceID
	}
	if maxAge := cmdArgs["max-age"]; maxAge != "" {
		filters["max_age"] = maxAge
	}

	// Make API call
	response, err := a.apiClient.ListAMISnapshots(a.ctx, filters)
	if err != nil {
		return fmt.Errorf("failed to list snapshots: %w", err)
	}

	// Display results
	fmt.Printf("📸 AMI Snapshots\n\n")

	snapshots := getSlice(response, "snapshots")
	if len(snapshots) == 0 {
		fmt.Printf("No snapshots found.\n")
		fmt.Printf("💡 Create one with: cws ami snapshot create <instance-id>\n")
		return nil
	}

	fmt.Printf("Found %d snapshots:\n\n", len(snapshots))

	for i, snapshot := range snapshots {
		if snapshotMap := getMap(snapshot, ""); snapshotMap != nil {
			fmt.Printf("📸 Snapshot %d:\n", i+1)
			fmt.Printf("   🆔 ID: %s\n", getString(snapshotMap, "snapshot_id"))
			fmt.Printf("   💾 Volume ID: %s\n", getString(snapshotMap, "volume_id"))
			fmt.Printf("   📊 Size: %d GB\n", getInt(snapshotMap, "volume_size"))
			fmt.Printf("   🏷️  Description: %s\n", getString(snapshotMap, "description"))
			fmt.Printf("   📅 Created: %s\n", getString(snapshotMap, "start_time"))
			fmt.Printf("   ⚡ State: %s\n", getString(snapshotMap, "state"))
			fmt.Printf("   📈 Progress: %s\n", getString(snapshotMap, "progress"))
			fmt.Printf("   💰 Storage cost: $%.3f/month\n", getFloat(snapshotMap, "storage_cost_monthly"))
			fmt.Printf("\n")
		}
	}

	totalCost := getFloat(response, "total_storage_cost_monthly")
	fmt.Printf("💰 Total monthly storage cost: $%.2f\n", totalCost)

	return nil
}

// handleAMISnapshotCreate creates a snapshot from an instance
func (a *App) handleAMISnapshotCreate(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: cws ami snapshot create <instance-id> [--description <desc>] [--no-reboot]")
	}

	instanceID := args[0]
	cmdArgs := parseCmdArgs(args[1:])

	// Prepare snapshot creation request
	request := map[string]interface{}{
		"instance_id": instanceID,
		"no_reboot":   cmdArgs["no-reboot"] != "",
	}

	if description := cmdArgs["description"]; description != "" {
		request["description"] = description
	} else {
		request["description"] = fmt.Sprintf("Snapshot of workspace %s created on %s",
			instanceID, time.Now().Format("2006-01-02 15:04:05"))
	}

	fmt.Printf("📸 Creating snapshot of workspace %s...\n", instanceID)

	if request["no_reboot"] == false {
		fmt.Printf("⚠️  Instance will be temporarily stopped to ensure consistent snapshot\n")
	}

	// Make API call
	response, err := a.apiClient.CreateAMISnapshot(a.ctx, request)
	if err != nil {
		return fmt.Errorf("failed to create snapshot: %w", err)
	}

	// Display results
	fmt.Printf("\n✅ Snapshot creation initiated successfully!\n\n")
	fmt.Printf("📋 Details:\n")
	fmt.Printf("   🆔 Snapshot ID: %s\n", getString(response, "snapshot_id"))
	fmt.Printf("   💾 Volume ID: %s\n", getString(response, "volume_id"))
	fmt.Printf("   📊 Volume Size: %d GB\n", getInt(response, "volume_size"))
	fmt.Printf("   ⏱️  Estimated completion: %d minutes\n", getInt(response, "estimated_completion_minutes"))
	fmt.Printf("   💰 Storage cost: $%.3f/month\n", getFloat(response, "storage_cost_monthly"))

	fmt.Printf("\n💡 Check progress with: cws ami snapshot list --instance-id %s\n", instanceID)

	return nil
}

// handleAMISnapshotRestore creates an AMI from a snapshot
func (a *App) handleAMISnapshotRestore(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: cws ami snapshot restore <snapshot-id> --name <ami-name> [--description <desc>] [--architecture <arch>]")
	}

	snapshotID := args[0]
	cmdArgs := parseCmdArgs(args[1:])

	// Validate required parameters
	amiName := cmdArgs["name"]
	if amiName == "" {
		return fmt.Errorf("AMI name is required (use --name <ami-name>)")
	}

	// Prepare restore request
	request := map[string]interface{}{
		"snapshot_id": snapshotID,
		"name":        amiName,
	}

	if description := cmdArgs["description"]; description != "" {
		request["description"] = description
	} else {
		request["description"] = fmt.Sprintf("AMI restored from snapshot %s", snapshotID)
	}

	if architecture := cmdArgs["architecture"]; architecture != "" {
		request["architecture"] = architecture
	} else {
		request["architecture"] = "x86_64" // Default
	}

	fmt.Printf("🔄 Restoring AMI from snapshot %s...\n", snapshotID)

	// Make API call
	response, err := a.apiClient.RestoreAMIFromSnapshot(a.ctx, request)
	if err != nil {
		return fmt.Errorf("failed to restore AMI from snapshot: %w", err)
	}

	// Display results
	fmt.Printf("\n✅ AMI restore initiated successfully!\n\n")
	fmt.Printf("📋 Details:\n")
	fmt.Printf("   🆔 AMI ID: %s\n", getString(response, "ami_id"))
	fmt.Printf("   📝 Name: %s\n", getString(response, "name"))
	fmt.Printf("   📸 Source Snapshot: %s\n", getString(response, "snapshot_id"))
	fmt.Printf("   🏛️  Architecture: %s\n", getString(response, "architecture"))
	fmt.Printf("   ⏱️  Estimated completion: %d minutes\n", getInt(response, "estimated_completion_minutes"))

	fmt.Printf("\n💡 Check status with: cws ami status %s\n", getString(response, "ami_id"))

	return nil
}

// handleAMISnapshotDelete deletes a specific snapshot
func (a *App) handleAMISnapshotDelete(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: cws ami snapshot delete <snapshot-id> [--force]")
	}

	snapshotID := args[0]
	cmdArgs := parseCmdArgs(args[1:])
	force := cmdArgs["force"] != ""

	// Display confirmation warning
	if !force {
		fmt.Printf("⚠️  Snapshot Deletion Warning\n")
		fmt.Printf("Snapshot ID: %s\n", snapshotID)
		fmt.Printf("\n💡 This operation cannot be undone\n")
		fmt.Printf("💡 Use --force to skip this confirmation\n\n")

		fmt.Printf("Continue? (y/N): ")
		var response string
		_, _ = fmt.Scanln(&response)
		if !(strings.ToLower(response) == "y" || strings.ToLower(response) == "yes") {
			fmt.Println("Operation cancelled")
			return nil
		}
	}

	// Prepare deletion request
	request := map[string]interface{}{
		"snapshot_id": snapshotID,
	}

	fmt.Printf("🗑️  Deleting snapshot %s...\n", snapshotID)

	// Make API call
	response, err := a.apiClient.DeleteAMISnapshot(a.ctx, request)
	if err != nil {
		return fmt.Errorf("failed to delete snapshot: %w", err)
	}

	// Display results
	fmt.Printf("\n✅ Snapshot deletion completed successfully!\n\n")
	fmt.Printf("📋 Details:\n")
	fmt.Printf("   🆔 Snapshot ID: %s\n", getString(response, "snapshot_id"))
	fmt.Printf("   📊 Size: %d GB\n", getInt(response, "volume_size"))
	fmt.Printf("   💰 Monthly savings: $%.3f\n", getFloat(response, "storage_savings_monthly"))

	return nil
}

// handleAMICheckFreshness checks AMI freshness against latest versions (v0.5.4 - Universal Version System)
func (a *App) handleAMICheckFreshness(args []string) error {
	// Ensure daemon is running (auto-start if needed)
	if err := a.ensureDaemonRunning(); err != nil {
		return err
	}

	fmt.Printf("🔍 Checking AMI freshness against latest AWS SSM versions...\n\n")

	// Make API call to daemon
	response, err := a.apiClient.CheckAMIFreshness(a.ctx)
	if err != nil {
		return fmt.Errorf("failed to check AMI freshness: %w", err)
	}

	// Display results using helper methods
	a.displayFreshnessSummary(response)
	a.displayFreshnessResults(response)
	a.displayFreshnessRecommendations(response)
	a.displayFreshnessFooter(response)

	return nil
}

// displayFreshnessSummary shows the summary statistics
func (a *App) displayFreshnessSummary(response map[string]interface{}) {
	totalChecked := getInt(response, "total_checked")
	outdated := getInt(response, "outdated")
	upToDate := getInt(response, "up_to_date")
	noSSM := getInt(response, "no_ssm_support")

	fmt.Printf("📊 Summary:\n")
	fmt.Printf("   Total AMIs checked: %d\n", totalChecked)
	fmt.Printf("   Up to date: %d\n", upToDate)
	fmt.Printf("   Outdated: %d\n", outdated)
	fmt.Printf("   No SSM support: %d\n\n", noSSM)
}

// displayFreshnessResults displays categorized AMI results
func (a *App) displayFreshnessResults(response map[string]interface{}) {
	results := getSlice(response, "results")
	if results == nil {
		return
	}

	outdated := getInt(response, "outdated")
	outdatedResults, ssmResults, staticResults := a.categorizeAMIResults(results)

	a.displayOutdatedAMIs(outdatedResults)
	a.displaySSMDistros(ssmResults, outdated)
	a.displayStaticDistros(staticResults)
}

// categorizeAMIResults groups results by status
func (a *App) categorizeAMIResults(results []interface{}) (outdated, ssm, static []interface{}) {
	for _, result := range results {
		resultMap := getMap(result, "")
		if resultMap == nil {
			continue
		}

		needsUpdate := getBool(resultMap, "needs_update")
		hasSSMSupport := getBool(resultMap, "has_ssm_support")

		if needsUpdate && getBool(resultMap, "is_outdated") {
			outdated = append(outdated, result)
		} else if hasSSMSupport {
			ssm = append(ssm, result)
		} else {
			static = append(static, result)
		}
	}
	return
}

// displayOutdatedAMIs shows outdated AMIs that need updates
func (a *App) displayOutdatedAMIs(outdatedResults []interface{}) {
	if len(outdatedResults) == 0 {
		return
	}

	fmt.Printf("⚠️  Outdated AMIs (need updates):\n\n")
	for _, result := range outdatedResults {
		resultMap := getMap(result, "")
		distro := getString(resultMap, "distro")
		version := getString(resultMap, "version")
		region := getString(resultMap, "region")
		arch := getString(resultMap, "architecture")
		current := getString(resultMap, "current_ami")
		latest := getString(resultMap, "latest_ami")

		fmt.Printf("  📀 %s %s (%s/%s)\n", distro, version, region, arch)
		fmt.Printf("     Current: %s\n", current)
		fmt.Printf("     Latest:  %s\n", latest)
		if message := getString(resultMap, "message"); message != "" {
			fmt.Printf("     Note: %s\n", message)
		}
		fmt.Printf("\n")
	}
}

// displaySSMDistros shows SSM-supported distributions
func (a *App) displaySSMDistros(ssmResults []interface{}, outdated int) {
	if len(ssmResults) == 0 || outdated > 0 {
		return
	}

	fmt.Printf("✅ SSM-supported distributions (automatically updated):\n")
	ssmDistros := make(map[string]bool)
	for _, result := range ssmResults {
		resultMap := getMap(result, "")
		distro := getString(resultMap, "distro")
		ssmDistros[distro] = true
	}
	for distro := range ssmDistros {
		fmt.Printf("   • %s\n", distro)
	}
	fmt.Printf("\n")
}

// displayStaticDistros shows static distributions
func (a *App) displayStaticDistros(staticResults []interface{}) {
	if len(staticResults) == 0 {
		return
	}

	fmt.Printf("ℹ️  Static distributions (manual updates required):\n")
	staticDistros := make(map[string]bool)
	for _, result := range staticResults {
		resultMap := getMap(result, "")
		distro := getString(resultMap, "distro")
		staticDistros[distro] = true
	}
	for distro := range staticDistros {
		fmt.Printf("   • %s\n", distro)
	}
	fmt.Printf("\n")
}

// displayFreshnessRecommendations shows recommendations
func (a *App) displayFreshnessRecommendations(response map[string]interface{}) {
	outdated := getInt(response, "outdated")
	if outdated > 0 {
		fmt.Printf("💡 Recommendation: %s\n", getString(response, "recommendation"))
		fmt.Printf("\n📝 Update static AMI mappings in pkg/templates/parser.go\n")
	} else {
		fmt.Printf("✅ All AMIs are up to date!\n")
	}
}

// displayFreshnessFooter shows supported distributions and timestamp
func (a *App) displayFreshnessFooter(response map[string]interface{}) {
	if ssmSupported := getStringSlice(response, "ssm_supported"); len(ssmSupported) > 0 {
		fmt.Printf("\n🔄 SSM-supported distributions: %s\n", strings.Join(ssmSupported, ", "))
	}
	if staticOnly := getStringSlice(response, "static_only"); len(staticOnly) > 0 {
		fmt.Printf("📌 Static-only distributions: %s\n", strings.Join(staticOnly, ", "))
	}

	if timestamp := getString(response, "check_timestamp"); timestamp != "" {
		fmt.Printf("\n⏰ Check completed at: %s\n", timestamp)
	}
}
