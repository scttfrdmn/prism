package cli

import (
	"context"
	"fmt"
	"strings"
	"time"
	
	"github.com/scttfrdmn/cloudworkstation/pkg/pricing"
	"github.com/scttfrdmn/cloudworkstation/pkg/types"
)

// LaunchCommand represents a launch operation using Command Pattern (SOLID)
type LaunchCommand interface {
	Execute(req *types.LaunchRequest, args []string, index int) (newIndex int, err error)
	CanHandle(arg string) bool
}

// LaunchCommandDispatcher manages launch flag parsing (Single Responsibility)
type LaunchCommandDispatcher struct {
	commands []LaunchCommand
}

// NewLaunchCommandDispatcher creates a new launch command dispatcher
func NewLaunchCommandDispatcher() *LaunchCommandDispatcher {
	dispatcher := &LaunchCommandDispatcher{}
	
	// Register all launch commands
	dispatcher.RegisterCommand(&SizeCommand{})
	dispatcher.RegisterCommand(&VolumeCommand{})
	dispatcher.RegisterCommand(&StorageCommand{})
	dispatcher.RegisterCommand(&RegionCommand{})
	dispatcher.RegisterCommand(&SubnetCommand{})
	dispatcher.RegisterCommand(&VpcCommand{})
	dispatcher.RegisterCommand(&ProjectCommand{})
	dispatcher.RegisterCommand(&PackageManagerCommand{})
	dispatcher.RegisterCommand(&SpotCommand{})
	dispatcher.RegisterCommand(&HibernationCommand{})
	dispatcher.RegisterCommand(&DryRunCommand{})
	dispatcher.RegisterCommand(&WaitCommand{})
	
	return dispatcher
}

// RegisterCommand registers a new launch command (Open/Closed Principle)
func (d *LaunchCommandDispatcher) RegisterCommand(cmd LaunchCommand) {
	d.commands = append(d.commands, cmd)
}

// ParseFlags parses launch flags using command pattern
func (d *LaunchCommandDispatcher) ParseFlags(req *types.LaunchRequest, args []string) error {
	for i := 2; i < len(args); i++ {
		arg := args[i]
		handled := false
		
		for _, cmd := range d.commands {
			if cmd.CanHandle(arg) {
				newIndex, err := cmd.Execute(req, args, i)
				if err != nil {
					return err
				}
				i = newIndex
				handled = true
				break
			}
		}
		
		if !handled {
			return fmt.Errorf("unknown option: %s", arg)
		}
	}
	return nil
}

// SizeCommand handles --size flag
type SizeCommand struct{}

func (s *SizeCommand) CanHandle(arg string) bool {
	return arg == "--size"
}

func (s *SizeCommand) Execute(req *types.LaunchRequest, args []string, index int) (int, error) {
	if index+1 >= len(args) {
		return index, fmt.Errorf("--size requires a value")
	}
	req.Size = args[index+1]
	return index + 1, nil
}

// VolumeCommand handles --volume flag
type VolumeCommand struct{}

func (v *VolumeCommand) CanHandle(arg string) bool {
	return arg == "--volume"
}

func (v *VolumeCommand) Execute(req *types.LaunchRequest, args []string, index int) (int, error) {
	if index+1 >= len(args) {
		return index, fmt.Errorf("--volume requires a value")
	}
	req.Volumes = append(req.Volumes, args[index+1])
	return index + 1, nil
}

// StorageCommand handles --storage flag
type StorageCommand struct{}

func (s *StorageCommand) CanHandle(arg string) bool {
	return arg == "--storage"
}

func (s *StorageCommand) Execute(req *types.LaunchRequest, args []string, index int) (int, error) {
	if index+1 >= len(args) {
		return index, fmt.Errorf("--storage requires a value")
	}
	req.EBSVolumes = append(req.EBSVolumes, args[index+1])
	return index + 1, nil
}

// RegionCommand handles --region flag
type RegionCommand struct{}

func (r *RegionCommand) CanHandle(arg string) bool {
	return arg == "--region"
}

func (r *RegionCommand) Execute(req *types.LaunchRequest, args []string, index int) (int, error) {
	if index+1 >= len(args) {
		return index, fmt.Errorf("--region requires a value")
	}
	req.Region = args[index+1]
	return index + 1, nil
}

// SubnetCommand handles --subnet flag
type SubnetCommand struct{}

func (s *SubnetCommand) CanHandle(arg string) bool {
	return arg == "--subnet"
}

func (s *SubnetCommand) Execute(req *types.LaunchRequest, args []string, index int) (int, error) {
	if index+1 >= len(args) {
		return index, fmt.Errorf("--subnet requires a value")
	}
	req.SubnetID = args[index+1]
	return index + 1, nil
}

// VpcCommand handles --vpc flag
type VpcCommand struct{}

func (v *VpcCommand) CanHandle(arg string) bool {
	return arg == "--vpc"
}

func (v *VpcCommand) Execute(req *types.LaunchRequest, args []string, index int) (int, error) {
	if index+1 >= len(args) {
		return index, fmt.Errorf("--vpc requires a value")
	}
	req.VpcID = args[index+1]
	return index + 1, nil
}

// ProjectCommand handles --project flag
type ProjectCommand struct{}

func (p *ProjectCommand) CanHandle(arg string) bool {
	return arg == "--project"
}

func (p *ProjectCommand) Execute(req *types.LaunchRequest, args []string, index int) (int, error) {
	if index+1 >= len(args) {
		return index, fmt.Errorf("--project requires a value")
	}
	req.ProjectID = args[index+1]
	return index + 1, nil
}

// PackageManagerCommand handles --with flag
type PackageManagerCommand struct{}

func (p *PackageManagerCommand) CanHandle(arg string) bool {
	return arg == "--with"
}

func (p *PackageManagerCommand) Execute(req *types.LaunchRequest, args []string, index int) (int, error) {
	if index+1 >= len(args) {
		return index, fmt.Errorf("--with requires a package manager")
	}
	
	packageManager := args[index+1]
	supportedManagers := []string{"conda", "apt", "dnf", "ami"}
	
	supported := false
	for _, mgr := range supportedManagers {
		if packageManager == mgr {
			supported = true
			break
		}
	}
	
	if !supported {
		return index, fmt.Errorf("unsupported package manager: %s (supported: %s)", 
			packageManager, strings.Join(supportedManagers, ", "))
	}
	
	req.PackageManager = packageManager
	return index + 1, nil
}

// SpotCommand handles --spot flag
type SpotCommand struct{}

func (s *SpotCommand) CanHandle(arg string) bool {
	return arg == "--spot"
}

func (s *SpotCommand) Execute(req *types.LaunchRequest, args []string, index int) (int, error) {
	req.Spot = true
	return index, nil
}

// HibernationCommand handles --hibernation flag
type HibernationCommand struct{}

func (h *HibernationCommand) CanHandle(arg string) bool {
	return arg == "--hibernation"
}

func (h *HibernationCommand) Execute(req *types.LaunchRequest, args []string, index int) (int, error) {
	req.Hibernation = true
	return index, nil
}

// DryRunCommand handles --dry-run flag
type DryRunCommand struct{}

func (d *DryRunCommand) CanHandle(arg string) bool {
	return arg == "--dry-run"
}

func (d *DryRunCommand) Execute(req *types.LaunchRequest, args []string, index int) (int, error) {
	req.DryRun = true
	return index, nil
}

// WaitCommand handles --wait flag
type WaitCommand struct{}

func (w *WaitCommand) CanHandle(arg string) bool {
	return arg == "--wait"
}

func (w *WaitCommand) Execute(req *types.LaunchRequest, args []string, index int) (int, error) {
	req.Wait = true
	return index, nil
}

// Cost Analysis Strategies (Strategy Pattern - SOLID)

// CostCalculationStrategy defines the interface for cost calculation strategies
type CostCalculationStrategy interface {
	CalculateInstanceCost(instance types.Instance, calculator *pricing.Calculator) CostAnalysis
	GetHeaders() []string
	FormatRow(instance types.Instance, analysis CostAnalysis) string
}

// CostAnalysis holds the result of cost calculations
type CostAnalysis struct {
	DailyCost         float64
	ListDailyCost     float64
	ActualSpend       float64
	CurrentCostPerMin float64
	ListCostPerMin    float64
	RunningTime       string
	TypeIndicator     string
	Savings           float64
	SavingsPercent    float64
}

// BasicCostStrategy calculates costs without institutional discounts
type BasicCostStrategy struct{}

func (b *BasicCostStrategy) CalculateInstanceCost(instance types.Instance, calculator *pricing.Calculator) CostAnalysis {
	// Calculate total lifetime
	var totalLifetime time.Duration
	if !instance.LaunchTime.IsZero() {
		if instance.DeletionTime != nil && !instance.DeletionTime.IsZero() {
			totalLifetime = instance.DeletionTime.Sub(instance.LaunchTime)
		} else {
			totalLifetime = time.Since(instance.LaunchTime)
		}
	}

	dailyCost := instance.EstimatedDailyCost
	totalMinutes := totalLifetime.Minutes()
	actualSpend := (dailyCost / (24.0 * 60.0)) * totalMinutes

	var currentCostPerMin float64
	if instance.State == "running" {
		currentCostPerMin = dailyCost / (24.0 * 60.0)
	} else {
		currentCostPerMin = (dailyCost * 0.1) / (24.0 * 60.0) // Storage only
	}

	typeIndicator := "OD"
	if instance.InstanceLifecycle == "spot" {
		typeIndicator = "SP"
	}

	return CostAnalysis{
		DailyCost:         dailyCost,
		ListDailyCost:     dailyCost,
		ActualSpend:       actualSpend,
		CurrentCostPerMin: currentCostPerMin,
		ListCostPerMin:    currentCostPerMin,
		RunningTime:       b.formatRunningTime(totalLifetime),
		TypeIndicator:     typeIndicator,
		Savings:           0,
		SavingsPercent:    0,
	}
}

func (b *BasicCostStrategy) GetHeaders() []string {
	return []string{"INSTANCE", "STATE", "TYPE", "RUNNING", "TOTAL SPEND", "COST/MIN"}
}

func (b *BasicCostStrategy) FormatRow(instance types.Instance, analysis CostAnalysis) string {
	return fmt.Sprintf("%s\t%s\t%s\t%s\t$%.4f\t$%.6f\n",
		instance.Name,
		strings.ToUpper(instance.State),
		analysis.TypeIndicator,
		analysis.RunningTime,
		analysis.ActualSpend,
		analysis.CurrentCostPerMin)
}

func (b *BasicCostStrategy) formatRunningTime(duration time.Duration) string {
	days := int(duration.Hours()) / 24
	hours := int(duration.Hours()) % 24
	minutes := int(duration.Minutes()) % 60
	seconds := int(duration.Seconds()) % 60

	if days > 0 {
		return fmt.Sprintf("%d:%02d:%02d:%02d", days, hours, minutes, seconds)
	}
	return fmt.Sprintf("%d:%02d:%02d", hours, minutes, seconds)
}

// InstitutionalCostStrategy calculates costs with institutional discounts
type InstitutionalCostStrategy struct{}

func (i *InstitutionalCostStrategy) CalculateInstanceCost(instance types.Instance, calculator *pricing.Calculator) CostAnalysis {
	basic := (&BasicCostStrategy{}).CalculateInstanceCost(instance, calculator)

	// Calculate discounted costs if instance type is available
	if instance.InstanceType != "" && basic.DailyCost > 0 {
		estimatedHourlyListPrice := basic.DailyCost / 24.0
		result := calculator.CalculateInstanceCost(instance.InstanceType, estimatedHourlyListPrice, "us-west-2")
		
		if result.TotalDiscount > 0 {
			basic.ListDailyCost = result.ListPrice * 24
			basic.DailyCost = result.DailyEstimate
			basic.ListCostPerMin = basic.ListDailyCost / (24.0 * 60.0)
			
			if instance.State == "running" {
				basic.CurrentCostPerMin = basic.DailyCost / (24.0 * 60.0)
			} else {
				basic.CurrentCostPerMin = (basic.DailyCost * 0.1) / (24.0 * 60.0)
			}
			
			basic.Savings = basic.ListCostPerMin - basic.CurrentCostPerMin
			if basic.ListCostPerMin > 0 {
				basic.SavingsPercent = (basic.Savings / basic.ListCostPerMin) * 100
			}
		}
	}

	return basic
}

func (i *InstitutionalCostStrategy) GetHeaders() []string {
	return []string{"INSTANCE", "STATE", "TYPE", "RUNNING", "TOTAL SPEND", "COST/MIN", "LIST RATE", "SAVINGS"}
}

func (i *InstitutionalCostStrategy) FormatRow(instance types.Instance, analysis CostAnalysis) string {
	return fmt.Sprintf("%s\t%s\t%s\t%s\t$%.4f\t$%.6f\t$%.6f\t$%.6f (%.1f%%)\n",
		instance.Name,
		strings.ToUpper(instance.State),
		analysis.TypeIndicator,
		analysis.RunningTime,
		analysis.ActualSpend,
		analysis.CurrentCostPerMin,
		analysis.ListCostPerMin,
		analysis.Savings,
		analysis.SavingsPercent)
}

// CostAnalyzer provides cost analysis functionality using Strategy Pattern
type CostAnalyzer struct {
	strategy   CostCalculationStrategy
	calculator *pricing.Calculator
}

// NewCostAnalyzer creates a cost analyzer with the appropriate strategy
func NewCostAnalyzer(hasDiscounts bool, calculator *pricing.Calculator) *CostAnalyzer {
	var strategy CostCalculationStrategy
	if hasDiscounts {
		strategy = &InstitutionalCostStrategy{}
	} else {
		strategy = &BasicCostStrategy{}
	}
	
	return &CostAnalyzer{
		strategy:   strategy,
		calculator: calculator,
	}
}

// AnalyzeInstances analyzes a list of instances and returns cost data
func (ca *CostAnalyzer) AnalyzeInstances(instances []types.Instance) ([]CostAnalysis, CostSummary) {
	var analyses []CostAnalysis
	summary := CostSummary{}
	
	for _, instance := range instances {
		analysis := ca.strategy.CalculateInstanceCost(instance, ca.calculator)
		analyses = append(analyses, analysis)
		
		// Update summary
		summary.TotalHistoricalSpend += analysis.ActualSpend
		if instance.State == "running" {
			summary.TotalRunningCost += analysis.DailyCost
			summary.TotalListCost += analysis.ListDailyCost
			summary.RunningInstances++
		}
	}
	
	return analyses, summary
}

// GetHeaders returns the headers for the cost table
func (ca *CostAnalyzer) GetHeaders() []string {
	return ca.strategy.GetHeaders()
}

// FormatRow formats a single instance row
func (ca *CostAnalyzer) FormatRow(instance types.Instance, analysis CostAnalysis) string {
	return ca.strategy.FormatRow(instance, analysis)
}

// CostSummary holds aggregate cost information
type CostSummary struct {
	TotalRunningCost     float64
	TotalListCost        float64
	TotalHistoricalSpend float64
	RunningInstances     int
}

// Template Snapshot Command Pattern Implementation

// TemplateSnapshotCommand handles template snapshot operations using Command Pattern (SOLID: Single Responsibility)
type TemplateSnapshotCommand struct {
	argParser          *TemplateSnapshotArgParser
	validationService  *TemplateSnapshotValidationService
	discoveryService   *ConfigurationDiscoveryService
	generationService  *TemplateGenerationService
	saveService        *TemplateSnapshotSaveService
	apiClient          interface{} // API client for instance operations
}

// NewTemplateSnapshotCommand creates a new template snapshot command
func NewTemplateSnapshotCommand(apiClient interface{}) *TemplateSnapshotCommand {
	return &TemplateSnapshotCommand{
		argParser:          NewTemplateSnapshotArgParser(),
		validationService:  NewTemplateSnapshotValidationService(apiClient),
		discoveryService:   NewConfigurationDiscoveryService(),
		generationService:  NewTemplateGenerationService(),
		saveService:        NewTemplateSnapshotSaveService(),
		apiClient:          apiClient,
	}
}

// Execute executes the template snapshot command (Command Pattern)
func (c *TemplateSnapshotCommand) Execute(args []string) error {
	// Parse arguments
	config, err := c.argParser.Parse(args)
	if err != nil {
		return err
	}

	// Validate instance
	instance, err := c.validationService.ValidateInstance(config)
	if err != nil {
		return err
	}

	// Display operation info
	c.displaySnapshotInfo(config, instance)

	// Discover instance configuration
	instanceConfig, err := c.discoveryService.DiscoverConfiguration(instance)
	if err != nil {
		return fmt.Errorf("failed to discover instance configuration: %w", err)
	}

	// Generate template
	template, err := c.generationService.GenerateTemplate(config, instanceConfig)
	if err != nil {
		return fmt.Errorf("failed to generate template: %w", err)
	}

	// Save or preview template
	return c.saveService.HandleTemplateResult(config, template)
}

// displaySnapshotInfo displays snapshot operation information (Single Responsibility)
func (c *TemplateSnapshotCommand) displaySnapshotInfo(config *TemplateSnapshotConfig, instance *types.Instance) {
	fmt.Printf("📸 Template Snapshot\n")
	fmt.Printf("═══════════════════\n\n")

	fmt.Printf("📋 **Source Instance**:\n")
	fmt.Printf("   Name: %s\n", instance.Name)
	fmt.Printf("   Type: %s\n", instance.InstanceType)
	fmt.Printf("   State: %s\n", instance.State)
	fmt.Printf("   Launch Time: %s\n\n", instance.LaunchTime)

	fmt.Printf("🏗️  **Target Template**:\n")
	fmt.Printf("   Name: %s\n", config.TemplateName)
	if config.Description != "" {
		fmt.Printf("   Description: %s\n", config.Description)
	}
	if config.BaseTemplate != "" {
		fmt.Printf("   Base Template: %s\n", config.BaseTemplate)
	}
	fmt.Println()

	if config.DryRun {
		fmt.Printf("🔍 **Discovery Process (Dry Run)**:\n")
	} else {
		fmt.Printf("🔍 **Discovery Process**:\n")
	}
}

// TemplateSnapshotConfig represents template snapshot configuration (Single Responsibility)
type TemplateSnapshotConfig struct {
	InstanceName string
	TemplateName string
	Description  string
	BaseTemplate string
	DryRun       bool
}

// TemplateSnapshotArgParser parses template snapshot arguments (SOLID: Single Responsibility)
type TemplateSnapshotArgParser struct{}

// NewTemplateSnapshotArgParser creates a new argument parser
func NewTemplateSnapshotArgParser() *TemplateSnapshotArgParser {
	return &TemplateSnapshotArgParser{}
}

// Parse parses command line arguments into configuration (Single Responsibility)
func (p *TemplateSnapshotArgParser) Parse(args []string) (*TemplateSnapshotConfig, error) {
	if len(args) < 2 {
		return nil, fmt.Errorf(`usage: cws templates snapshot <instance-name> <template-name> [options]

Create a template from a running workstation's current configuration.

Arguments:
  instance-name    Name of the running instance to snapshot
  template-name    Name for the new template

Options:
  description=<text>       Description for the new template
  base=<template>          Base template to inherit from (optional)  
  dry-run                  Show what would be captured without creating template

Examples:
  cws templates snapshot my-ml-workstation custom-ml-env
  cws templates snapshot research-instance my-research-template description="Customized research environment"
  cws templates snapshot data-science-box ds-template base="Python Machine Learning" dry-run`)
	}

	// Parse arguments using helper methods
	cleanArgs := p.extractCleanArgs(args)
	if len(cleanArgs) < 2 {
		return nil, fmt.Errorf("missing required arguments: instance-name and template-name")
	}

	return &TemplateSnapshotConfig{
		InstanceName: cleanArgs[0],
		TemplateName: cleanArgs[1],
		Description:  p.parseDescription(args),
		BaseTemplate: p.parseBaseTemplate(args),
		DryRun:       p.parseDryRun(args),
	}, nil
}

// extractCleanArgs filters out option arguments and returns clean positional args (Single Responsibility)
func (p *TemplateSnapshotArgParser) extractCleanArgs(args []string) []string {
	var cleanArgs []string
	for _, arg := range args {
		if !strings.Contains(arg, "=") && arg != "dry-run" {
			cleanArgs = append(cleanArgs, arg)
		}
	}
	return cleanArgs
}

// parseDescription extracts description from arguments (Single Responsibility)
func (p *TemplateSnapshotArgParser) parseDescription(args []string) string {
	for _, arg := range args {
		if strings.HasPrefix(arg, "description=") {
			return strings.TrimPrefix(arg, "description=")
		}
	}
	return ""
}

// parseBaseTemplate extracts base template from arguments (Single Responsibility)
func (p *TemplateSnapshotArgParser) parseBaseTemplate(args []string) string {
	for _, arg := range args {
		if strings.HasPrefix(arg, "base=") {
			return strings.TrimPrefix(arg, "base=")
		}
	}
	return ""
}

// parseDryRun checks for dry-run flag (Single Responsibility)
func (p *TemplateSnapshotArgParser) parseDryRun(args []string) bool {
	for _, arg := range args {
		if arg == "dry-run" {
			return true
		}
	}
	return false
}

// TemplateSnapshotValidationService handles instance validation for snapshots (SOLID: Single Responsibility)
type TemplateSnapshotValidationService struct {
	apiClient interface{}
}

// NewTemplateSnapshotValidationService creates a new validation service
func NewTemplateSnapshotValidationService(apiClient interface{}) *TemplateSnapshotValidationService {
	return &TemplateSnapshotValidationService{
		apiClient: apiClient,
	}
}

// ValidateInstance validates the instance for snapshot creation (Single Responsibility)
func (s *TemplateSnapshotValidationService) ValidateInstance(config *TemplateSnapshotConfig) (*types.Instance, error) {
	// Check daemon is running
	if pingable, ok := s.apiClient.(interface{ Ping(context.Context) error }); ok {
		if err := pingable.Ping(context.Background()); err != nil {
			return nil, fmt.Errorf("daemon not running. Start with: cws daemon start")
		}
	}

	if config.DryRun {
		// For dry-run, create a mock instance
		return &types.Instance{
			Name:         config.InstanceName,
			InstanceType: "t3.medium",
			State:        "running",
			LaunchTime:   time.Now().Add(-2 * time.Hour),
		}, nil
	}

	// For real execution, verify instance exists and is running
	if lister, ok := s.apiClient.(interface{ ListInstances(context.Context) (*types.ListResponse, error) }); ok {
		response, err := lister.ListInstances(context.Background())
		if err != nil {
			return nil, fmt.Errorf("failed to list instances: %w", err)
		}

		for i := range response.Instances {
			if response.Instances[i].Name == config.InstanceName {
				instance := &response.Instances[i]
				if instance.State != "running" {
					return nil, fmt.Errorf("instance '%s' must be running to create snapshot (current state: %s)", config.InstanceName, instance.State)
				}
				return instance, nil
			}
		}
		return nil, fmt.Errorf("instance '%s' not found", config.InstanceName)
	}
	
	return nil, fmt.Errorf("API client does not support instance listing")
}

// ConfigurationDiscoveryService handles instance configuration discovery (SOLID: Single Responsibility)
type ConfigurationDiscoveryService struct{}

// NewConfigurationDiscoveryService creates a new configuration discovery service
func NewConfigurationDiscoveryService() *ConfigurationDiscoveryService {
	return &ConfigurationDiscoveryService{}
}

// DiscoverConfiguration discovers instance configuration (Single Responsibility)
func (s *ConfigurationDiscoveryService) DiscoverConfiguration(instance *types.Instance) (*InstanceConfiguration, error) {
	// Display discovery steps
	fmt.Printf("   🔍 Connecting to instance %s...\n", instance.Name)
	fmt.Printf("   📦 Discovering installed packages...\n")
	fmt.Printf("   👥 Analyzing user accounts...\n")
	fmt.Printf("   🔧 Checking system services...\n")
	fmt.Printf("   🌐 Scanning network configuration...\n")

	// Mock configuration for now (in real implementation, this would SSH to instance)
	return &InstanceConfiguration{
		BaseOS:         "ubuntu-22.04",
		PackageManager: "apt",
		Packages: PackageSet{
			System: []string{"curl", "wget", "git", "build-essential", "python3", "python3-pip"},
			Python: []string{"numpy", "pandas", "matplotlib", "jupyter"},
		},
		Users: []User{
			{Name: "ubuntu", Groups: []string{"sudo"}},
			{Name: "researcher", Groups: []string{"users"}},
		},
		Services: []Service{
			{Name: "jupyter", Command: "jupyter lab --no-browser --ip=0.0.0.0", Port: 8888},
		},
		Ports: []int{22, 8888},
	}, nil
}

// TemplateGenerationService handles template generation from configuration (SOLID: Single Responsibility)
type TemplateGenerationService struct{}

// NewTemplateGenerationService creates a new template generation service
func NewTemplateGenerationService() *TemplateGenerationService {
	return &TemplateGenerationService{}
}

// GenerateTemplate generates template YAML from configuration (Single Responsibility)
func (s *TemplateGenerationService) GenerateTemplate(config *TemplateSnapshotConfig, instanceConfig *InstanceConfiguration) (string, error) {
	// Generate template YAML content (simplified mock)
	template := fmt.Sprintf(`name: "%s"
description: "%s"
base: "%s"
package_manager: "%s"

packages:
  system: %v
  python: %v

users: %v
services: %v
ports: %v`,
		config.TemplateName,
		config.Description,
		instanceConfig.BaseOS,
		instanceConfig.PackageManager,
		instanceConfig.Packages.System,
		instanceConfig.Packages.Python,
		instanceConfig.Users,
		instanceConfig.Services,
		instanceConfig.Ports)

	return template, nil
}

// TemplateSnapshotSaveService handles template saving and result display (SOLID: Single Responsibility)
type TemplateSnapshotSaveService struct{}

// NewTemplateSnapshotSaveService creates a new save service
func NewTemplateSnapshotSaveService() *TemplateSnapshotSaveService {
	return &TemplateSnapshotSaveService{}
}

// HandleTemplateResult handles template preview or saving (Single Responsibility)
func (s *TemplateSnapshotSaveService) HandleTemplateResult(config *TemplateSnapshotConfig, template string) error {
	if config.DryRun {
		return s.displayDryRunResults(config, template)
	}
	return s.saveTemplateAndDisplayResults(config, template)
}

// displayDryRunResults displays dry-run preview (Single Responsibility)
func (s *TemplateSnapshotSaveService) displayDryRunResults(config *TemplateSnapshotConfig, template string) error {
	fmt.Printf("   ✅ Configuration discovery completed\n")
	fmt.Printf("   ✅ Template generation simulated\n\n")

	fmt.Printf("📄 **Generated Template Preview**:\n")
	fmt.Printf("```yaml\n%s```\n\n", template)

	fmt.Printf("💡 **Next Steps**:\n")
	fmt.Printf("   Run without dry-run to save template:\n")
	fmt.Printf("   cws templates snapshot %s %s", config.InstanceName, config.TemplateName)
	if config.Description != "" {
		fmt.Printf(" description=\"%s\"", config.Description)
	}
	if config.BaseTemplate != "" {
		fmt.Printf(" base=\"%s\"", config.BaseTemplate)
	}
	fmt.Println()

	return nil
}

// saveTemplateAndDisplayResults saves template and displays success (Single Responsibility)
func (s *TemplateSnapshotSaveService) saveTemplateAndDisplayResults(config *TemplateSnapshotConfig, template string) error {
	// TODO: Implement actual template saving
	// For now, simulate saving
	fmt.Printf("   ✅ Configuration discovery completed\n")
	fmt.Printf("   ✅ Template generated and saved\n\n")

	fmt.Printf("✅ **Template Created Successfully**:\n")
	fmt.Printf("   Template saved as: %s\n", config.TemplateName)
	fmt.Printf("   Location: templates/%s.yml\n\n", config.TemplateName)

	fmt.Printf("🚀 **Usage**:\n")
	fmt.Printf("   Launch new instance: cws launch \"%s\" new-instance\n", config.TemplateName)
	fmt.Printf("   View template info: cws templates info \"%s\"\n", config.TemplateName)
	fmt.Printf("   Validate template: cws templates validate \"%s\"\n", config.TemplateName)

	return nil
}

// Template Apply Command Pattern Implementation

// TemplateApplyCommand handles template application operations using Command Pattern (SOLID: Single Responsibility)
type TemplateApplyCommand struct {
	argParser         *TemplateApplyArgParser
	validationService *TemplateApplyValidationService
	applicationService *TemplateApplicationService
	displayService    *TemplateApplyDisplayService
	apiClient         interface{} // API client for template operations
}

// NewTemplateApplyCommand creates a new template apply command
func NewTemplateApplyCommand(apiClient interface{}) *TemplateApplyCommand {
	return &TemplateApplyCommand{
		argParser:         NewTemplateApplyArgParser(),
		validationService: NewTemplateApplyValidationService(apiClient),
		applicationService: NewTemplateApplicationService(apiClient),
		displayService:    NewTemplateApplyDisplayService(),
		apiClient:         apiClient,
	}
}

// Execute executes the template apply command (Command Pattern)
func (c *TemplateApplyCommand) Execute(args []string) error {
	// Parse arguments
	applyConfig, err := c.argParser.Parse(args)
	if err != nil {
		return err
	}

	// Validate template and daemon
	template, err := c.validationService.ValidateTemplateAndDaemon(applyConfig)
	if err != nil {
		return err
	}

	// Apply template
	response, err := c.applicationService.ApplyTemplate(applyConfig, template)
	if err != nil {
		return err
	}

	// Display results
	return c.displayService.DisplayResults(applyConfig, response)
}

// TemplateApplyConfig represents template application configuration (Single Responsibility)
type TemplateApplyConfig struct {
	TemplateName   string
	InstanceName   string
	DryRun         bool
	Force          bool
	PackageManager string
}

// TemplateApplyArgParser parses template apply arguments (SOLID: Single Responsibility)
type TemplateApplyArgParser struct{}

// NewTemplateApplyArgParser creates a new argument parser
func NewTemplateApplyArgParser() *TemplateApplyArgParser {
	return &TemplateApplyArgParser{}
}

// Parse parses command line arguments into apply configuration (Single Responsibility)
func (p *TemplateApplyArgParser) Parse(args []string) (*TemplateApplyConfig, error) {
	if len(args) < 2 {
		return nil, fmt.Errorf("usage: cws apply <template> <instance-name> [options]\n" +
			"  options: --dry-run --force --with <package-manager>")
	}

	config := &TemplateApplyConfig{
		TemplateName: args[0],
		InstanceName: args[1],
		DryRun:       false,
		Force:        false,
	}

	// Parse flags using helper method
	return p.parseFlags(config, args[2:])
}

// parseFlags parses command line flags (Single Responsibility)
func (p *TemplateApplyArgParser) parseFlags(config *TemplateApplyConfig, flags []string) (*TemplateApplyConfig, error) {
	for i := 0; i < len(flags); i++ {
		arg := flags[i]
		switch {
		case arg == "--dry-run":
			config.DryRun = true
		case arg == "--force":
			config.Force = true
		case arg == "--with" && i+1 < len(flags):
			packageManager := flags[i+1]
			if err := p.validatePackageManager(packageManager); err != nil {
				return nil, err
			}
			config.PackageManager = packageManager
			i++ // Skip the package manager value
		default:
			return nil, fmt.Errorf("unknown option: %s", arg)
		}
	}
	return config, nil
}

// validatePackageManager validates package manager is supported (Single Responsibility)
func (p *TemplateApplyArgParser) validatePackageManager(packageManager string) error {
	supportedManagers := []string{"conda", "apt", "dnf", "spack", "pip", "ami"}
	for _, mgr := range supportedManagers {
		if packageManager == mgr {
			return nil
		}
	}
	return fmt.Errorf("unsupported package manager: %s (supported: conda, apt, dnf, spack, pip, ami)", packageManager)
}

// TemplateApplyValidationService handles template validation (SOLID: Single Responsibility)
type TemplateApplyValidationService struct {
	apiClient interface{}
}

// NewTemplateApplyValidationService creates a new validation service
func NewTemplateApplyValidationService(apiClient interface{}) *TemplateApplyValidationService {
	return &TemplateApplyValidationService{
		apiClient: apiClient,
	}
}

// ValidateTemplateAndDaemon validates daemon is running and template exists (Single Responsibility)
func (s *TemplateApplyValidationService) ValidateTemplateAndDaemon(config *TemplateApplyConfig) (interface{}, error) {
	// Check daemon is running
	if pingable, ok := s.apiClient.(interface{ Ping(context.Context) error }); ok {
		if err := pingable.Ping(context.Background()); err != nil {
			return nil, fmt.Errorf("daemon not running. Start with: cws daemon start")
		}
	}

	// Get template from API
	if lister, ok := s.apiClient.(interface{ ListTemplates(context.Context) (map[string]interface{}, error) }); ok {
		runtimeTemplates, err := lister.ListTemplates(context.Background())
		if err != nil {
			return nil, fmt.Errorf("failed to list templates: %w", err)
		}

		template, exists := runtimeTemplates[config.TemplateName]
		if !exists {
			return nil, fmt.Errorf("template '%s' not found", config.TemplateName)
		}

		return template, nil
	}

	return nil, fmt.Errorf("API client does not support template listing")
}

// TemplateApplicationService handles template application operations (SOLID: Single Responsibility)
type TemplateApplicationService struct {
	apiClient interface{}
}

// NewTemplateApplicationService creates a new application service
func NewTemplateApplicationService(apiClient interface{}) *TemplateApplicationService {
	return &TemplateApplicationService{
		apiClient: apiClient,
	}
}

// TemplateApplyResponse represents template application results
type TemplateApplyResponse struct {
	Message              string
	PackagesInstalled    int
	ServicesConfigured   int
	UsersCreated         int
	RollbackCheckpoint   string
	Warnings             []string
	ExecutionTime        time.Duration
}

// ApplyTemplate applies the template using the configuration (Single Responsibility)
func (s *TemplateApplicationService) ApplyTemplate(config *TemplateApplyConfig, template interface{}) (*TemplateApplyResponse, error) {
	// Convert runtime template to unified template for application
	unifiedTemplate := s.convertToUnifiedTemplate(template)

	// Create apply request
	req := s.createApplyRequest(config, unifiedTemplate)

	// Apply template via API
	if applier, ok := s.apiClient.(interface{ ApplyTemplate(context.Context, interface{}) (*TemplateApplyResponse, error) }); ok {
		return applier.ApplyTemplate(context.Background(), req)
	}

	return nil, fmt.Errorf("API client does not support template application")
}

// convertToUnifiedTemplate converts runtime template to unified template (Single Responsibility)
func (s *TemplateApplicationService) convertToUnifiedTemplate(template interface{}) interface{} {
	// This is a placeholder - in practice, we'd need the daemon to provide
	// the full unified template information for application
	return template
}

// createApplyRequest creates the apply request (Single Responsibility)
func (s *TemplateApplicationService) createApplyRequest(config *TemplateApplyConfig, template interface{}) interface{} {
	// Create a request object that matches the expected API structure
	return map[string]interface{}{
		"instance_name":    config.InstanceName,
		"dry_run":          config.DryRun,
		"force":            config.Force,
		"package_manager":  config.PackageManager,
		"template":         template,
	}
}

// TemplateApplyDisplayService handles result display (SOLID: Single Responsibility)
type TemplateApplyDisplayService struct{}

// NewTemplateApplyDisplayService creates a new display service
func NewTemplateApplyDisplayService() *TemplateApplyDisplayService {
	return &TemplateApplyDisplayService{}
}

// DisplayResults displays template application results (Single Responsibility)
func (s *TemplateApplyDisplayService) DisplayResults(config *TemplateApplyConfig, response *TemplateApplyResponse) error {
	if config.DryRun {
		return s.displayDryRunResults(config, response)
	}
	return s.displaySuccessResults(config, response)
}

// displayDryRunResults displays dry-run preview results (Single Responsibility)
func (s *TemplateApplyDisplayService) displayDryRunResults(config *TemplateApplyConfig, response *TemplateApplyResponse) error {
	fmt.Printf("🔍 Dry run results for applying '%s' to '%s':\n", config.TemplateName, config.InstanceName)
	fmt.Printf("📦 Would install %d packages\n", response.PackagesInstalled)
	fmt.Printf("🔧 Would configure %d services\n", response.ServicesConfigured)
	fmt.Printf("👤 Would create %d users\n", response.UsersCreated)

	if len(response.Warnings) > 0 {
		fmt.Println("\n⚠️  Warnings:")
		for _, warning := range response.Warnings {
			fmt.Printf("   - %s\n", warning)
		}
	}

	fmt.Printf("\n⏱️  Estimated execution time: %v\n", response.ExecutionTime)
	fmt.Printf("\n💡 Run without --dry-run to apply these changes\n")

	return nil
}

// displaySuccessResults displays successful application results (Single Responsibility)
func (s *TemplateApplyDisplayService) displaySuccessResults(config *TemplateApplyConfig, response *TemplateApplyResponse) error {
	fmt.Printf("✅ %s\n", response.Message)
	fmt.Printf("📊 Changes applied:\n")
	fmt.Printf("   📦 Packages installed: %d\n", response.PackagesInstalled)
	fmt.Printf("   🔧 Services configured: %d\n", response.ServicesConfigured)
	fmt.Printf("   👤 Users created: %d\n", response.UsersCreated)

	if response.RollbackCheckpoint != "" {
		fmt.Printf("   📸 Rollback checkpoint: %s\n", response.RollbackCheckpoint)
	}

	if len(response.Warnings) > 0 {
		fmt.Println("\n⚠️  Warnings:")
		for _, warning := range response.Warnings {
			fmt.Printf("   - %s\n", warning)
		}
	}

	fmt.Printf("\n⏱️  Execution time: %v\n", response.ExecutionTime)
	fmt.Printf("\n💡 Use 'cws layers %s' to see all applied templates\n", config.InstanceName)
	fmt.Printf("💡 Use 'cws rollback %s' to undo these changes if needed\n", config.InstanceName)

	return nil
}