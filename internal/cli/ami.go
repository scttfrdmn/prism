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
	"github.com/scttfrdmn/cloudworkstation/pkg/ami"
	"github.com/scttfrdmn/cloudworkstation/pkg/types"
)

// AMI processes AMI-related commands
func (a *App) AMI(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("missing AMI command (build, list, validate, publish, save)")
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

			fmt.Printf("DEBUG: Parsed arg '%s' = '%s'\n", key, value)
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
		return nil, fmt.Errorf("usage: cws ami save <instance-name> <template-name> [options]")
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
	return fmt.Sprintf("Custom template saved from instance %s", instanceName)
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
					return nil, fmt.Errorf("instance '%s' must be running to save as AMI (current state: %s)", saveConfig.InstanceName, inst.State)
				}
				return &inst, nil
			}
		}

		return nil, fmt.Errorf("instance '%s' not found", saveConfig.InstanceName)
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
	fmt.Printf("💾 Saving instance '%s' as template '%s'\n", saveConfig.InstanceName, saveConfig.TemplateName)
	fmt.Printf("📍 Instance ID: %s\n", instance.ID)
	fmt.Printf("🏷️  Description: %s\n", saveConfig.Description)
	if len(saveConfig.CopyToRegions) > 0 {
		fmt.Printf("🌍 Will copy to regions: %s\n", strings.Join(saveConfig.CopyToRegions, ", "))
	}

	// Warning about temporary stop
	fmt.Printf("\n⚠️  WARNING: Instance will be temporarily stopped to create a consistent AMI\n")
	fmt.Printf("   This ensures the AMI captures a clean state of the filesystem.\n")
	fmt.Printf("   The instance will be automatically restarted after AMI creation.\n\n")

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
			"Name":                             saveConfig.TemplateName,
			"CloudWorkstationTemplate":         saveConfig.TemplateName,
			"CloudWorkstationSavedFrom":        saveConfig.InstanceName,
			"CloudWorkstationOriginalTemplate": instance.Template,
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
	fmt.Printf("\n🎉 Successfully saved instance as AMI!\n")
	fmt.Printf("📸 AMI ID: %s\n", result.AMIID)
	fmt.Printf("🕒 Build time: %s\n", result.BuildDuration)

	if len(result.CopiedAMIs) > 0 {
		fmt.Printf("\n🌍 AMI copied to additional regions:\n")
		for region, amiID := range result.CopiedAMIs {
			fmt.Printf("   %s: %s\n", region, amiID)
		}
	}

	fmt.Printf("\n✨ Template '%s' is now available for launching new instances:\n", templateName)
	fmt.Printf("   cws launch %s my-new-instance\n", templateName)

	return nil
}
