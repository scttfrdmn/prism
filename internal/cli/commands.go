package cli

import (
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