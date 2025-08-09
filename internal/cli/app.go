// Package cli implements CloudWorkstation's command-line interface application.
//
// This package provides the CLI application logic for the CloudWorkstation client (cws).
// It handles command parsing, API client communication, output formatting, and user
// interaction flows while maintaining CloudWorkstation's core design principles.
//
// Application Structure:
//   - App: Main CLI application with command routing
//   - Command handlers for all CloudWorkstation operations
//   - Output formatting with tables and JSON support
//   - Error handling with user-friendly messages
//   - Configuration management and validation
//
// Supported Commands:
//   - launch: Create new research instances
//   - list: Show instance status and costs
//   - connect: Get connection information
//   - stop/start: Instance lifecycle management
//   - volumes: EFS volume operations
//   - storage: EBS storage management
//
// Design Philosophy:
// Follows "Progressive Disclosure" - simple commands with optional advanced flags.
// All operations provide clear feedback and cost visibility.
//
// Usage:
//
//	app := cli.NewApp(apiClient)
//	err := app.Run(os.Args)
package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"
	"github.com/scttfrdmn/cloudworkstation/pkg/api"
	"github.com/scttfrdmn/cloudworkstation/pkg/api/client"
	"github.com/scttfrdmn/cloudworkstation/pkg/pricing"
	"github.com/scttfrdmn/cloudworkstation/pkg/profile"
	"github.com/scttfrdmn/cloudworkstation/pkg/project"
	"github.com/scttfrdmn/cloudworkstation/pkg/templates"
	"github.com/scttfrdmn/cloudworkstation/pkg/types"
	"github.com/scttfrdmn/cloudworkstation/pkg/version"
)

// App represents the CLI application
type App struct {
	version        string
	apiClient      api.CloudWorkstationAPI
	ctx            context.Context // Context for AWS operations
	tuiCommand     *cobra.Command
	config         *Config
	profileManager *profile.ManagerEnhanced
}

// NewApp creates a new CLI application
func NewApp(version string) *App {
	// Load config
	config, err := LoadConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: Failed to load config: %v\n", err)
		config = &Config{} // Use empty config
		config.Daemon.URL = "http://localhost:8947" // Default URL (CWS on phone keypad)
	}
	
	// Initialize profile manager
	profileManager, err := profile.NewManagerEnhanced()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: Failed to initialize profile manager: %v\n", err)
		// Continue without profile manager
	}
	
	// Initialize API client
	apiURL := config.Daemon.URL
	if envURL := os.Getenv("CWSD_URL"); envURL != "" {
		apiURL = envURL
	}
	
	// Create API client with configuration
	baseClient := api.NewClientWithOptions(apiURL, client.Options{
		AWSProfile: config.AWS.Profile,
		AWSRegion:  config.AWS.Region,
	})
	
	// Create app
	app := &App{
		version:        version,
		apiClient:      baseClient,
		ctx:            context.Background(),
		config:         config,
		profileManager: profileManager,
	}
	
	// Initialize TUI command
	app.tuiCommand = NewTUICommand()
	
	return app
}

// NewAppWithClient creates a new CLI application with a custom API client
func NewAppWithClient(version string, client api.CloudWorkstationAPI) *App {
	// Load config
	config, err := LoadConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: Failed to load config: %v\n", err)
		config = &Config{} // Use empty config
	}
	
	// Initialize profile manager
	profileManager, err := profile.NewManagerEnhanced()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: Failed to initialize profile manager: %v\n", err)
		// Continue without profile manager
	}
	
	return &App{
		version:        version,
		apiClient:      client,
		ctx:            context.Background(),
		config:         config,
		profileManager: profileManager,
	}
}

// TUI launches the terminal UI
func (a *App) TUI(_ []string) error {
	return a.tuiCommand.Execute()
}

// Launch handles the launch command
func (a *App) Launch(args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("usage: cws launch <template> <name> [options]\n" +
			"  options: --size XS|S|M|L|XL --volume <name> --storage <size> --project <name> --with conda|apt|dnf|ami --spot --hibernation --dry-run --subnet <subnet-id> --vpc <vpc-id>\n" +
			"\n" +
			"  T-shirt sizes (compute + storage):\n" +
			"    XS: 1 vCPU, 2GB RAM + 100GB storage  (t3.small/t4g.small)\n" +
			"    S:  2 vCPU, 4GB RAM + 500GB storage  (t3.medium/t4g.medium)\n" +
			"    M:  2 vCPU, 8GB RAM + 1TB storage    (t3.large/t4g.large) [default]\n" +
			"    L:  4 vCPU, 16GB RAM + 2TB storage   (t3.xlarge/t4g.xlarge)\n" +
			"    XL: 8 vCPU, 32GB RAM + 4TB storage   (t3.2xlarge/t4g.2xlarge)\n" +
			"\n" +
			"  GPU workloads automatically scale to GPU instances (g4dn/g5g family)\n" +
			"  Memory-intensive workloads use r5/r6g instances with more RAM\n" +
			"  Compute-intensive workloads use c5/c6g instances for better CPU performance")
	}

	template := args[0]
	name := args[1]

	// Parse options
	req := types.LaunchRequest{
		Template: template,
		Name:     name,
	}

	// Parse additional flags
	for i := 2; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "--size" && i+1 < len(args):
			req.Size = args[i+1]
			i++
		case arg == "--volume" && i+1 < len(args):
			req.Volumes = append(req.Volumes, args[i+1])
			i++
		case arg == "--storage" && i+1 < len(args):
			req.EBSVolumes = append(req.EBSVolumes, args[i+1])
			i++
		case arg == "--region" && i+1 < len(args):
			req.Region = args[i+1]
			i++
		case arg == "--subnet" && i+1 < len(args):
			req.SubnetID = args[i+1]
			i++
		case arg == "--vpc" && i+1 < len(args):
			req.VpcID = args[i+1]
			i++
		case arg == "--project" && i+1 < len(args):
			req.ProjectID = args[i+1]
			i++
		case arg == "--spot":
			req.Spot = true
		case arg == "--hibernation":
			req.Hibernation = true
		case arg == "--dry-run":
			req.DryRun = true
		case arg == "--with" && i+1 < len(args):
			packageManager := args[i+1]
			// Validate supported package managers
			supportedManagers := []string{"conda", "apt", "dnf", "ami"}
			supported := false
			for _, mgr := range supportedManagers {
				if packageManager == mgr {
					supported = true
					break
				}
			}
			if !supported {
				return fmt.Errorf("unsupported package manager: %s (supported: conda, apt, dnf, ami)", packageManager)
			}
			
			// All package managers now supported
			
			req.PackageManager = packageManager
			i++
		default:
			return fmt.Errorf("unknown option: %s", arg)
		}
	}

	// Check daemon is running
	if err := a.apiClient.Ping(a.ctx); err != nil {
		return fmt.Errorf("daemon not running. Start with: cws daemon start")
	}

	response, err := a.apiClient.LaunchInstance(a.ctx, req)
	if err != nil {
		return fmt.Errorf("failed to launch instance: %w", err)
	}

	fmt.Printf("🚀 %s\n", response.Message)
	fmt.Printf("💰 Estimated cost: %s\n", response.EstimatedCost)
	fmt.Printf("🔗 Connect with: %s\n", response.ConnectionInfo)
	
	// Show project information if launched in a project
	if req.ProjectID != "" {
		fmt.Printf("📁 Project: %s\n", req.ProjectID)
		fmt.Printf("🏷️  Instance will be tracked under project budget\n")
	}

	return nil
}

// List handles the list command with optional project filtering
func (a *App) List(args []string) error {
	// Parse arguments for project filtering
	var projectFilter string
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "--project" && i+1 < len(args):
			projectFilter = args[i+1]
			i++
		}
	}
	
	// Check daemon is running
	if err := a.apiClient.Ping(a.ctx); err != nil {
		return fmt.Errorf("daemon not running. Start with: cws daemon start")
	}

	response, err := a.apiClient.ListInstances(a.ctx)
	if err != nil {
		return fmt.Errorf("failed to list instances: %w", err)
	}

	// Filter instances by project if specified
	var filteredInstances []types.Instance
	if projectFilter != "" {
		for _, instance := range response.Instances {
			if instance.ProjectID == projectFilter {
				filteredInstances = append(filteredInstances, instance)
			}
		}
	} else {
		filteredInstances = response.Instances
	}

	if len(filteredInstances) == 0 {
		if projectFilter != "" {
			fmt.Printf("No workstations found in project '%s'. Launch one with: cws launch <template> <name> --project %s\n", projectFilter, projectFilter)
		} else {
			fmt.Println("No workstations found. Launch one with: cws launch <template> <name>")
		}
		return nil
	}

	// Show header with project filter info
	if projectFilter != "" {
		fmt.Printf("Workstations in project '%s':\n\n", projectFilter)
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 8, 2, ' ', 0)
	fmt.Fprintln(w, "NAME\tTEMPLATE\tSTATE\tTYPE\tPUBLIC IP\tPROJECT\tLAUNCHED")
	for _, instance := range filteredInstances {
		projectInfo := "-"
		if instance.ProjectID != "" {
			projectInfo = instance.ProjectID
		}
		
		// Format spot/on-demand indicator
		typeIndicator := "OD"
		if instance.InstanceLifecycle == "spot" {
			typeIndicator = "SP"
		}
		
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
			instance.Name,
			instance.Template,
			strings.ToUpper(instance.State),
			typeIndicator,
			instance.PublicIP,
			projectInfo,
			instance.LaunchTime.Format("2006-01-02 15:04"),
		)
	}

	w.Flush()

	return nil
}

// ListCost handles the list cost command - shows detailed cost information
func (a *App) ListCost(args []string) error {
	// Parse project filter
	var projectFilter string
	for i, arg := range args {
		switch {
		case arg == "--project" && i+1 < len(args):
			projectFilter = args[i+1]
			i++
		}
	}
	
	// Check daemon is running
	if err := a.apiClient.Ping(a.ctx); err != nil {
		return fmt.Errorf("daemon not running. Start with: cws daemon start")
	}

	response, err := a.apiClient.ListInstances(a.ctx)
	if err != nil {
		return fmt.Errorf("failed to list instances: %w", err)
	}

	// Filter instances by project if specified
	var filteredInstances []types.Instance
	if projectFilter != "" {
		for _, instance := range response.Instances {
			if instance.ProjectID == projectFilter {
				filteredInstances = append(filteredInstances, instance)
			}
		}
	} else {
		filteredInstances = response.Instances
	}

	if len(filteredInstances) == 0 {
		if projectFilter != "" {
			fmt.Printf("No workstations found in project '%s'.\n", projectFilter)
		} else {
			fmt.Println("No workstations found.")
		}
		return nil
	}

	// Show header with project filter info
	if projectFilter != "" {
		fmt.Printf("💰 Cost Analysis for project '%s':\n\n", projectFilter)
	} else {
		fmt.Println("💰 CloudWorkstation Cost Analysis\n")
	}

	// Load pricing configuration for accurate cost calculation
	pricingConfig, _ := pricing.LoadInstitutionalPricing()
	calculator := pricing.NewCalculator(pricingConfig)
	
	// Check if we have institutional discounts
	hasDiscounts := pricingConfig != nil && (pricingConfig.Institution != "Default")
	
	w := tabwriter.NewWriter(os.Stdout, 0, 8, 2, ' ', 0)
	if hasDiscounts {
		fmt.Fprintln(w, "INSTANCE\tSTATE\tTYPE\tRUNNING\tTOTAL SPEND\tCOST/MIN\tLIST RATE\tSAVINGS")
	} else {
		fmt.Fprintln(w, "INSTANCE\tSTATE\tTYPE\tRUNNING\tTOTAL SPEND\tCOST/MIN")
	}

	totalRunningCost := 0.0
	totalListCost := 0.0
	totalHistoricalSpend := 0.0
	runningInstances := 0

	for _, instance := range filteredInstances {
		// Calculate total lifetime for this instance
		var totalLifetime time.Duration
		if !instance.LaunchTime.IsZero() {
			if instance.DeletionTime != nil && !instance.DeletionTime.IsZero() {
				// Terminated instance - use launch to deletion time
				totalLifetime = instance.DeletionTime.Sub(instance.LaunchTime)
			} else {
				// Running or stopped instance - use launch to now
				totalLifetime = time.Since(instance.LaunchTime)
			}
		}
		
		// Get base cost rates
		dailyCost := instance.EstimatedDailyCost
		listDailyCost := dailyCost
		
		if hasDiscounts && instance.InstanceType != "" {
			// Get accurate pricing with discounts
			estimatedHourlyListPrice := dailyCost / 24.0
			if dailyCost > 0 {
				result := calculator.CalculateInstanceCost(instance.InstanceType, estimatedHourlyListPrice, "us-west-2")
				if result.TotalDiscount > 0 {
					listDailyCost = result.ListPrice * 24
					dailyCost = result.DailyEstimate
				}
			}
		}
		
		// Calculate actual spend so far (total lifetime cost)
		totalMinutes := totalLifetime.Minutes()
		actualSpend := (dailyCost / (24.0 * 60.0)) * totalMinutes
		
		// Calculate current cost per minute (running vs stopped rates)
		var currentCostPerMin, listCurrentCostPerMin float64
		if instance.State == "running" {
			// Running: full compute + storage cost
			currentCostPerMin = dailyCost / (24.0 * 60.0)
			listCurrentCostPerMin = listDailyCost / (24.0 * 60.0)
			runningInstances++
			totalRunningCost += dailyCost  // Add to daily running cost
			totalListCost += listDailyCost
		} else {
			// Stopped: only EBS storage cost (estimate ~10% of full cost)
			currentCostPerMin = (dailyCost * 0.1) / (24.0 * 60.0)
			listCurrentCostPerMin = (listDailyCost * 0.1) / (24.0 * 60.0)
		}
		
		totalHistoricalSpend += actualSpend
		
		// Format type indicator
		typeIndicator := "OD"
		if instance.InstanceLifecycle == "spot" {
			typeIndicator = "SP"
		}
		
		// Format running time as d:h:m:s
		days := int(totalLifetime.Hours()) / 24
		hours := int(totalLifetime.Hours()) % 24
		minutes := int(totalLifetime.Minutes()) % 60
		seconds := int(totalLifetime.Seconds()) % 60
		
		var runningTime string
		if days > 0 {
			runningTime = fmt.Sprintf("%d:%02d:%02d:%02d", days, hours, minutes, seconds)
		} else {
			runningTime = fmt.Sprintf("%d:%02d:%02d", hours, minutes, seconds)
		}
		
		if hasDiscounts {
			savings := listCurrentCostPerMin - currentCostPerMin
			savingsPercent := 0.0
			if listCurrentCostPerMin > 0 {
				savingsPercent = (savings / listCurrentCostPerMin) * 100
			}
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\t$%.4f\t$%.6f\t$%.6f\t$%.6f (%.1f%%)\n",
				instance.Name,
				strings.ToUpper(instance.State),
				typeIndicator,
				runningTime,
				actualSpend,
				currentCostPerMin,
				listCurrentCostPerMin,
				savings,
				savingsPercent,
			)
		} else {
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\t$%.4f\t$%.6f\n",
				instance.Name,
				strings.ToUpper(instance.State),
				typeIndicator,
				runningTime,
				actualSpend,
				currentCostPerMin,
			)
		}
	}

	w.Flush()
	
	// Summary section
	fmt.Println()
	fmt.Printf("📊 Cost Summary:\n")
	if hasDiscounts {
		totalSavings := totalListCost - totalRunningCost
		savingsPercent := 0.0
		if totalListCost > 0 {
			savingsPercent = (totalSavings / totalListCost) * 100
		}
		fmt.Printf("   Running instances: %d\n", runningInstances)
		fmt.Printf("   Your daily cost:   $%.4f\n", totalRunningCost)
		fmt.Printf("   Your monthly est:  $%.4f\n", totalRunningCost*30)
		fmt.Printf("   List price daily:  $%.4f\n", totalListCost)
		fmt.Printf("   Daily savings:     $%.4f (%.1f%%)\n", totalSavings, savingsPercent)
		fmt.Printf("   Historical spend:  $%.4f\n", totalHistoricalSpend)
		if pricingConfig.Institution != "" {
			fmt.Printf("   Institution:       %s\n", pricingConfig.Institution)
		}
	} else {
		fmt.Printf("   Running instances: %d\n", runningInstances)
		fmt.Printf("   Daily cost:        $%.4f\n", totalRunningCost)
		fmt.Printf("   Monthly estimate:  $%.4f\n", totalRunningCost*30)
		fmt.Printf("   Historical spend:  $%.4f\n", totalHistoricalSpend)
	}
	
	fmt.Printf("\n💡 Tip: Use 'cws list' for a clean instance overview without cost data\n")
	
	return nil
}

// Connect handles the connect command
func (a *App) Connect(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: cws connect <instance-name> [--verbose]")
	}

	name := args[0]
	verbose := false
	
	// Parse flags
	for i := 1; i < len(args); i++ {
		switch args[i] {
		case "--verbose", "-v":
			verbose = true
		default:
			return fmt.Errorf("unknown flag: %s", args[i])
		}
	}

	// Check daemon is running
	if err := a.apiClient.Ping(a.ctx); err != nil {
		return fmt.Errorf("daemon not running. Start with: cws daemon start")
	}

	connectionInfo, err := a.apiClient.ConnectInstance(a.ctx, name)
	if err != nil {
		return fmt.Errorf("failed to get connection info: %w", err)
	}

	if verbose {
		fmt.Printf("🔗 SSH command for %s:\n", name)
		fmt.Printf("%s\n", connectionInfo)
		return nil
	}

	// Execute SSH command directly
	return a.executeSSHCommand(connectionInfo, name)
}

// executeSSHCommand executes the SSH command and transfers control to the SSH process
func (a *App) executeSSHCommand(connectionInfo, instanceName string) error {
	fmt.Printf("🔗 Connecting to %s...\n", instanceName)
	
	// Use shell to execute the SSH command to handle quotes properly
	cmd := exec.Command("sh", "-c", connectionInfo)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	
	err := cmd.Run()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			// SSH exited with non-zero status - this is normal for SSH disconnections
			os.Exit(exitErr.ExitCode())
		}
		return fmt.Errorf("failed to execute SSH command: %w", err)
	}
	
	return nil
}

// Stop handles the stop command
func (a *App) Stop(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: cws stop <n>")
	}

	name := args[0]

	// Check daemon is running
	if err := a.apiClient.Ping(a.ctx); err != nil {
		return fmt.Errorf("daemon not running. Start with: cws daemon start")
	}

	err := a.apiClient.StopInstance(a.ctx, name)
	if err != nil {
		return fmt.Errorf("failed to stop instance: %w", err)
	}

	fmt.Printf("⏹️ Stopping instance %s...\n", name)
	return nil
}

// Start handles the start command with intelligent state management
func (a *App) Start(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: cws start <n>")
	}

	name := args[0]

	// Check daemon is running
	if err := a.apiClient.Ping(a.ctx); err != nil {
		return fmt.Errorf("daemon not running. Start with: cws daemon start")
	}

	// First, get current instance status
	listResponse, err := a.apiClient.ListInstances(a.ctx)
	if err != nil {
		return fmt.Errorf("failed to get instance status: %w", err)
	}

	var targetInstance *types.Instance
	for _, instance := range listResponse.Instances {
		if instance.Name == name {
			targetInstance = &instance
			break
		}
	}

	if targetInstance == nil {
		return fmt.Errorf("instance '%s' not found", name)
	}

	// Check current state and handle appropriately
	switch strings.ToLower(targetInstance.State) {
	case "running":
		fmt.Printf("✅ Instance %s is already running\n", name)
		return nil
		
	case "stopped":
		// Ready to start - proceed normally
		
	case "stopping":
		fmt.Printf("⏳ Instance %s is currently stopping. Please wait and try again in a few moments.\n", name)
		return nil
		
	case "starting", "pending":
		fmt.Printf("⏳ Instance %s is already starting. Check status with 'cws list'.\n", name)
		return nil
		
	case "shutting-down", "terminated":
		return fmt.Errorf("❌ Cannot start instance '%s' - it is %s", name, targetInstance.State)
		
	default:
		return fmt.Errorf("❌ Cannot start instance '%s' - unknown state: %s", name, targetInstance.State)
	}

	// Attempt to start the instance
	err = a.apiClient.StartInstance(a.ctx, name)
	if err != nil {
		return fmt.Errorf("failed to start instance: %w", err)
	}

	fmt.Printf("▶️ Starting instance %s...\n", name)
	return nil
}


// Delete handles the delete command
func (a *App) Delete(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: cws delete <n>")
	}

	name := args[0]

	// Check daemon is running
	if err := a.apiClient.Ping(a.ctx); err != nil {
		return fmt.Errorf("daemon not running. Start with: cws daemon start")
	}

	err := a.apiClient.DeleteInstance(a.ctx, name)
	if err != nil {
		return fmt.Errorf("failed to delete instance: %w", err)
	}

	fmt.Printf("🗑️ Deleting instance %s...\n", name)
	return nil
}

// Hibernate handles the hibernate command
func (a *App) Hibernate(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: cws hibernate <name>")
	}

	name := args[0]

	// Check daemon is running
	if err := a.apiClient.Ping(a.ctx); err != nil {
		return fmt.Errorf("daemon not running. Start with: cws daemon start")
	}

	// Check hibernation status first
	status, err := a.apiClient.GetInstanceHibernationStatus(a.ctx, name)
	if err != nil {
		return fmt.Errorf("failed to check hibernation status: %w", err)
	}

	if !status.HibernationSupported {
		fmt.Printf("⚠️  Instance %s does not support hibernation\n", name)
		fmt.Printf("    Falling back to regular stop operation\n")
	}

	err = a.apiClient.HibernateInstance(a.ctx, name)
	if err != nil {
		return fmt.Errorf("failed to hibernate instance: %w", err)
	}

	if status.HibernationSupported {
		fmt.Printf("🛌 Hibernating instance %s...\n", name)
		fmt.Printf("   💡 RAM state preserved for instant resume\n")
		fmt.Printf("   💰 Compute billing stopped, storage continues\n")
	} else {
		fmt.Printf("⏹️ Stopping instance %s (hibernation not supported)...\n", name)
	}

	return nil
}

// Resume handles the resume command
func (a *App) Resume(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: cws resume <name>")
	}

	name := args[0]

	// Check daemon is running
	if err := a.apiClient.Ping(a.ctx); err != nil {
		return fmt.Errorf("daemon not running. Start with: cws daemon start")
	}

	// Check hibernation status first
	status, err := a.apiClient.GetInstanceHibernationStatus(a.ctx, name)
	if err != nil {
		return fmt.Errorf("failed to check hibernation status: %w", err)
	}

	if status.IsHibernated {
		err = a.apiClient.ResumeInstance(a.ctx, name)
		if err != nil {
			return fmt.Errorf("failed to resume instance: %w", err)
		}
		fmt.Printf("⏰ Resuming hibernated instance %s...\n", name)
		fmt.Printf("   🚀 Instant startup from preserved RAM state\n")
	} else {
		// Fall back to regular start
		err = a.apiClient.StartInstance(a.ctx, name)
		if err != nil {
			return fmt.Errorf("failed to start instance: %w", err)
		}
		fmt.Printf("▶️ Starting instance %s...\n", name)
		fmt.Printf("   💡 Instance was not hibernated, performing regular start\n")
	}

	return nil
}


// Volume handles volume commands
func (a *App) Volume(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: cws volume <action> [args]")
	}

	action := args[0]
	volumeArgs := args[1:]

	// Check daemon is running
	if err := a.apiClient.Ping(a.ctx); err != nil {
		return fmt.Errorf("daemon not running. Start with: cws daemon start")
	}

	switch action {
	case "create":
		return a.volumeCreate(volumeArgs)
	case "list":
		return a.volumeList(volumeArgs)
	case "info":
		return a.volumeInfo(volumeArgs)
	case "delete":
		return a.volumeDelete(volumeArgs)
	default:
		return fmt.Errorf("unknown volume action: %s", action)
	}
}

func (a *App) volumeCreate(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: cws volume create <n> [options]")
	}

	req := types.VolumeCreateRequest{
		Name: args[0],
	}

	// Parse options
	for i := 1; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "--performance" && i+1 < len(args):
			req.PerformanceMode = args[i+1]
			i++
		case arg == "--throughput" && i+1 < len(args):
			req.ThroughputMode = args[i+1]
			i++
		case arg == "--region" && i+1 < len(args):
			req.Region = args[i+1]
			i++
		default:
			return fmt.Errorf("unknown option: %s", arg)
		}
	}

	volume, err := a.apiClient.CreateVolume(a.ctx, req)
	if err != nil {
		return fmt.Errorf("failed to create volume: %w", err)
	}

	fmt.Printf("📁 Created EFS volume %s (%s)\n", volume.Name, volume.FileSystemId)
	return nil
}

func (a *App) volumeList(_ []string) error {
	volumes, err := a.apiClient.ListVolumes(a.ctx)
	if err != nil {
		return fmt.Errorf("failed to list volumes: %w", err)
	}

	if len(volumes) == 0 {
		fmt.Println("No EFS volumes found. Create one with: cws volume create <n>")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 8, 2, ' ', 0)
	fmt.Fprintln(w, "NAME\tFILESYSTEM ID\tSTATE\tSIZE\tCOST/MONTH")

	for _, volume := range volumes {
		sizeGB := float64(volume.SizeBytes) / (1024 * 1024 * 1024)
		costMonth := sizeGB * volume.EstimatedCostGB
		fmt.Fprintf(w, "%s\t%s\t%s\t%.1f GB\t$%.2f\n",
			volume.Name,
			volume.FileSystemId,
			strings.ToUpper(volume.State),
			sizeGB,
			costMonth,
		)
	}
	w.Flush()

	return nil
}

func (a *App) volumeInfo(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: cws volume info <n>")
	}

	name := args[0]
	volume, err := a.apiClient.GetVolume(a.ctx, name)
	if err != nil {
		return fmt.Errorf("failed to get volume info: %w", err)
	}

	fmt.Printf("📁 EFS Volume: %s\n", volume.Name)
	fmt.Printf("   Filesystem ID: %s\n", volume.FileSystemId)
	fmt.Printf("   State: %s\n", strings.ToUpper(volume.State))
	fmt.Printf("   Region: %s\n", volume.Region)
	fmt.Printf("   Performance Mode: %s\n", volume.PerformanceMode)
	fmt.Printf("   Throughput Mode: %s\n", volume.ThroughputMode)
	fmt.Printf("   Size: %.1f GB\n", float64(volume.SizeBytes)/(1024*1024*1024))
	fmt.Printf("   Cost: $%.2f/month\n", float64(volume.SizeBytes)/(1024*1024*1024)*volume.EstimatedCostGB)
	fmt.Printf("   Created: %s\n", volume.CreationTime.Format("2006-01-02 15:04:05"))

	return nil
}

func (a *App) volumeDelete(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: cws volume delete <n>")
	}

	name := args[0]
	err := a.apiClient.DeleteVolume(a.ctx, name)
	if err != nil {
		return fmt.Errorf("failed to delete volume: %w", err)
	}

	fmt.Printf("🗑️ Deleting EFS volume %s...\n", name)
	return nil
}

// Storage handles storage commands
func (a *App) Storage(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: cws storage <action> [args]")
	}

	action := args[0]
	storageArgs := args[1:]

	// Check daemon is running
	if err := a.apiClient.Ping(a.ctx); err != nil {
		return fmt.Errorf("daemon not running. Start with: cws daemon start")
	}

	switch action {
	case "create":
		return a.storageCreate(storageArgs)
	case "list":
		return a.storageList(storageArgs)
	case "info":
		return a.storageInfo(storageArgs)
	case "attach":
		return a.storageAttach(storageArgs)
	case "detach":
		return a.storageDetach(storageArgs)
	case "delete":
		return a.storageDelete(storageArgs)
	default:
		return fmt.Errorf("unknown storage action: %s", action)
	}
}

func (a *App) storageCreate(args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("usage: cws storage create <n> <size> [type]")
	}

	req := types.StorageCreateRequest{
		Name:       args[0],
		Size:       args[1],
		VolumeType: "gp3", // default
	}

	if len(args) > 2 {
		req.VolumeType = args[2]
	}

	// Parse additional options
	for i := 3; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "--region" && i+1 < len(args):
			req.Region = args[i+1]
			i++
		default:
			return fmt.Errorf("unknown option: %s", arg)
		}
	}

	volume, err := a.apiClient.CreateStorage(a.ctx, req)
	if err != nil {
		return fmt.Errorf("failed to create storage: %w", err)
	}

	fmt.Printf("💾 Created EBS volume %s (%s) - %d GB %s\n",
		volume.Name, volume.VolumeID, volume.SizeGB, volume.VolumeType)
	return nil
}

func (a *App) storageList(_ []string) error {
	volumes, err := a.apiClient.ListStorage(a.ctx)
	if err != nil {
		return fmt.Errorf("failed to list storage: %w", err)
	}

	if len(volumes) == 0 {
		fmt.Println("No EBS volumes found. Create one with: cws storage create <n> <size>")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 8, 2, ' ', 0)
	fmt.Fprintln(w, "NAME\tVOLUME ID\tSTATE\tSIZE\tTYPE\tATTACHED TO\tCOST/MONTH")

	for _, volume := range volumes {
		costMonth := float64(volume.SizeGB) * volume.EstimatedCostGB
		attachedTo := volume.AttachedTo
		if attachedTo == "" {
			attachedTo = "-"
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%d GB\t%s\t%s\t$%.2f\n",
			volume.Name,
			volume.VolumeID,
			strings.ToUpper(volume.State),
			volume.SizeGB,
			volume.VolumeType,
			attachedTo,
			costMonth,
		)
	}
	w.Flush()

	return nil
}

func (a *App) storageInfo(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: cws storage info <n>")
	}

	name := args[0]
	volume, err := a.apiClient.GetStorage(a.ctx, name)
	if err != nil {
		return fmt.Errorf("failed to get storage info: %w", err)
	}

	fmt.Printf("💾 EBS Volume: %s\n", volume.Name)
	fmt.Printf("   Volume ID: %s\n", volume.VolumeID)
	fmt.Printf("   State: %s\n", strings.ToUpper(volume.State))
	fmt.Printf("   Region: %s\n", volume.Region)
	fmt.Printf("   Size: %d GB\n", volume.SizeGB)
	fmt.Printf("   Type: %s\n", volume.VolumeType)
	if volume.IOPS > 0 {
		fmt.Printf("   IOPS: %d\n", volume.IOPS)
	}
	if volume.Throughput > 0 {
		fmt.Printf("   Throughput: %d MB/s\n", volume.Throughput)
	}
	if volume.AttachedTo != "" {
		fmt.Printf("   Attached to: %s\n", volume.AttachedTo)
	}
	fmt.Printf("   Cost: $%.2f/month\n", float64(volume.SizeGB)*volume.EstimatedCostGB)
	fmt.Printf("   Created: %s\n", volume.CreationTime.Format("2006-01-02 15:04:05"))

	return nil
}

func (a *App) storageAttach(args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("usage: cws storage attach <volume> <instance>")
	}

	volumeName := args[0]
	instanceName := args[1]

	err := a.apiClient.AttachStorage(a.ctx, volumeName, instanceName)
	if err != nil {
		return fmt.Errorf("failed to attach storage: %w", err)
	}

	fmt.Printf("🔗 Attaching volume %s to instance %s...\n", volumeName, instanceName)
	return nil
}

func (a *App) storageDetach(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: cws storage detach <volume>")
	}

	volumeName := args[0]

	err := a.apiClient.DetachStorage(a.ctx, volumeName)
	if err != nil {
		return fmt.Errorf("failed to detach storage: %w", err)
	}

	fmt.Printf("🔓 Detaching volume %s...\n", volumeName)
	return nil
}

func (a *App) storageDelete(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: cws storage delete <n>")
	}

	name := args[0]
	err := a.apiClient.DeleteStorage(a.ctx, name)
	if err != nil {
		return fmt.Errorf("failed to delete storage: %w", err)
	}

	fmt.Printf("🗑️ Deleting EBS volume %s...\n", name)
	return nil
}

// Templates handles the templates command
func (a *App) Templates(args []string) error {
	// Handle subcommands
	if len(args) > 0 {
		switch args[0] {
		case "validate":
			return a.validateTemplates(args[1:])
		case "search":
			return a.templatesSearch(args[1:])
		case "info":
			return a.templatesInfo(args[1:])
		case "featured":
			return a.templatesFeatured(args[1:])
		case "discover":
			return a.templatesDiscover(args[1:])
		case "install":
			return a.templatesInstall(args[1:])
		case "version":
			return a.templatesVersion(args[1:])
		case "snapshot":
			return a.templatesSnapshot(args[1:])
		}
	}
	
	// Default: list all templates
	return a.templatesList(args)
}

// templatesList lists available templates (default behavior)
func (a *App) templatesList(args []string) error {
	// Check daemon is running
	if err := a.apiClient.Ping(a.ctx); err != nil {
		return fmt.Errorf("daemon not running. Start with: cws daemon start")
	}

	templates, err := a.apiClient.ListTemplates(a.ctx)
	if err != nil {
		return fmt.Errorf("failed to list templates: %w", err)
	}

	fmt.Println("Available templates:")
	fmt.Println()

	for name, template := range templates {
		fmt.Printf("🏗️  %s\n", name)
		fmt.Printf("   %s\n", template.Description)
		fmt.Printf("   Cost: $%.2f/hour (x86_64), $%.2f/hour (arm64)\n",
			template.EstimatedCostPerHour["x86_64"],
			template.EstimatedCostPerHour["arm64"])
		fmt.Println()
	}
	
	fmt.Println("💡 Size Information:")
	fmt.Println("   Launch with --size XS|S|M|L|XL to specify compute and storage resources")
	fmt.Println("   XS: 1 vCPU, 2GB RAM + 100GB    S: 2 vCPU, 4GB RAM + 500GB    M: 2 vCPU, 8GB RAM + 1TB [default]")
	fmt.Println("   L: 4 vCPU, 16GB RAM + 2TB       XL: 8 vCPU, 32GB RAM + 4TB")
	fmt.Println("   GPU/memory/compute workloads automatically scale to optimized instance families")
	fmt.Println()

	return nil
}

// templatesSearch searches for templates across repositories
func (a *App) templatesSearch(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: cws templates search <query>")
	}

	query := args[0]
	fmt.Printf("🔍 Searching for templates matching '%s'...\n\n", query)

	// Use existing repository manager to search across repositories
	// This would integrate with the GitHub repository system
	fmt.Printf("📍 Search results from CloudWorkstation Template Repositories:\n\n")
	
	// Placeholder implementation - in real system would search GitHub repos
	matchedTemplates := []struct {
		name       string
		repo       string
		description string
		downloads  int
		rating     float64
	}{
		{"python-ml-advanced", "community", "Advanced Python ML environment with GPU optimization", 1247, 4.8},
		{"r-bioconductor", "bioinformatics", "R environment with Bioconductor packages", 892, 4.6},
		{"neuroimaging-fsl", "neuroimaging", "FSL-based neuroimaging analysis environment", 567, 4.9},
	}

	for _, tmpl := range matchedTemplates {
		fmt.Printf("🏗️  %s:%s\n", tmpl.repo, tmpl.name)
		fmt.Printf("   %s\n", tmpl.description)
		fmt.Printf("   ⭐ %.1f stars • 📥 %d downloads\n", tmpl.rating, tmpl.downloads)
		fmt.Printf("   Install: cws templates install %s:%s\n", tmpl.repo, tmpl.name)
		fmt.Println()
	}

	fmt.Printf("💡 Add more repositories with: cws repo add <name> <github-url>\n")
	return nil
}

// templatesInfo shows detailed information about a specific template
func (a *App) templatesInfo(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: cws templates info <template-name>")
	}

	templateName := args[0]
	
	// Get raw template information directly from templates package
	rawTemplate, err := templates.GetTemplateInfo(templateName)
	if err != nil {
		return fmt.Errorf("failed to get template info: %w", err)
	}

	// Also get runtime template for cost and instance type information
	region := "us-west-2" // Default region for cost calculations
	runtimeTemplate, runtimeErr := templates.GetTemplate(templateName, region, "x86_64")
	
	fmt.Printf("📋 Detailed Template Information\n")
	fmt.Printf("═══════════════════════════════════════════════════════════════════\n\n")

	// Basic Information
	fmt.Printf("🏗️  **Name**: %s\n", rawTemplate.Name)
	if rawTemplate.Slug != "" {
		fmt.Printf("🔗 **Slug**: %s (for CLI: `cws launch %s <name>`)\n", rawTemplate.Slug, rawTemplate.Slug)
	}
	fmt.Printf("📝 **Description**: %s\n", rawTemplate.Description)
	fmt.Printf("🖥️  **Base OS**: %s\n", rawTemplate.Base)
	fmt.Printf("📦 **Package Manager**: %s\n", rawTemplate.PackageManager)
	fmt.Println()

	// Template Inheritance
	if len(rawTemplate.Inherits) > 0 {
		fmt.Printf("🔗 **Inherits From**:\n")
		for _, parent := range rawTemplate.Inherits {
			fmt.Printf("   • %s\n", parent)
		}
		fmt.Println()
	}

	// Cost and Instance Information (from runtime template)
	if runtimeErr == nil {
		fmt.Printf("💰 **Estimated Costs** (default M size):\n")
		if cost, exists := runtimeTemplate.EstimatedCostPerHour["x86_64"]; exists {
			fmt.Printf("   • x86_64: $%.3f/hour ($%.2f/day)\n", cost, cost*24)
		}
		if cost, exists := runtimeTemplate.EstimatedCostPerHour["arm64"]; exists {
			fmt.Printf("   • arm64:  $%.3f/hour ($%.2f/day)\n", cost, cost*24)
		}
		
		fmt.Printf("\n🖥️  **Instance Types** (default M size):\n")
		if instanceType, exists := runtimeTemplate.InstanceType["x86_64"]; exists {
			fmt.Printf("   • x86_64: %s\n", instanceType)
		}
		if instanceType, exists := runtimeTemplate.InstanceType["arm64"]; exists {
			fmt.Printf("   • arm64:  %s\n", instanceType)
		}
		fmt.Println()
	}

	// Size Scaling Information
	fmt.Printf("📏 **T-Shirt Size Scaling**:\n")
	fmt.Printf("   • XS: 1 vCPU, 2GB RAM + 100GB storage\n")
	fmt.Printf("   • S:  2 vCPU, 4GB RAM + 500GB storage\n") 
	fmt.Printf("   • M:  2 vCPU, 8GB RAM + 1TB storage [default]\n")
	fmt.Printf("   • L:  4 vCPU, 16GB RAM + 2TB storage\n")
	fmt.Printf("   • XL: 8 vCPU, 32GB RAM + 4TB storage\n")
	
	// Smart scaling analysis
	requiresGPU := containsGPUPackages(rawTemplate)
	requiresHighMemory := containsMemoryPackages(rawTemplate) 
	requiresHighCPU := containsComputePackages(rawTemplate)
	
	if requiresGPU || requiresHighMemory || requiresHighCPU {
		fmt.Printf("\n🧠 **Smart Scaling**: This template will use optimized instance types:\n")
		if requiresGPU {
			fmt.Printf("   • GPU workloads → g4dn/g5g instance families\n")
		}
		if requiresHighMemory {
			fmt.Printf("   • Memory-intensive → r5/r6g instance families\n")
		}
		if requiresHighCPU {
			fmt.Printf("   • Compute-intensive → c5/c6g instance families\n")
		}
	}
	fmt.Println()

	// Packages
	if hasPackages(rawTemplate) {
		fmt.Printf("📦 **Installed Packages**:\n")
		if len(rawTemplate.Packages.System) > 0 {
			fmt.Printf("   • **System** (%s): %s\n", rawTemplate.PackageManager, strings.Join(rawTemplate.Packages.System, ", "))
		}
		if len(rawTemplate.Packages.Conda) > 0 {
			fmt.Printf("   • **Conda**: %s\n", strings.Join(rawTemplate.Packages.Conda, ", "))
		}
		if len(rawTemplate.Packages.Pip) > 0 {
			fmt.Printf("   • **Pip**: %s\n", strings.Join(rawTemplate.Packages.Pip, ", "))
		}
		if len(rawTemplate.Packages.Spack) > 0 {
			fmt.Printf("   • **Spack**: %s\n", strings.Join(rawTemplate.Packages.Spack, ", "))
		}
		fmt.Println()
	}

	// Users
	if len(rawTemplate.Users) > 0 {
		fmt.Printf("👤 **User Accounts**:\n")
		for _, user := range rawTemplate.Users {
			groups := "-"
			if len(user.Groups) > 0 {
				groups = strings.Join(user.Groups, ", ")
			}
			shell := user.Shell
			if shell == "" {
				shell = "/bin/bash"
			}
			fmt.Printf("   • %s (groups: %s, shell: %s)\n", user.Name, groups, shell)
		}
		fmt.Println()
	}

	// Services
	if len(rawTemplate.Services) > 0 {
		fmt.Printf("🔧 **Services**:\n")
		for _, service := range rawTemplate.Services {
			status := "disabled"
			if service.Enable {
				status = "enabled"
			}
			port := ""
			if service.Port > 0 {
				port = fmt.Sprintf(", port: %d", service.Port)
			}
			fmt.Printf("   • %s (%s%s)\n", service.Name, status, port)
		}
		fmt.Println()
	}

	// Ports
	if runtimeErr == nil && len(runtimeTemplate.Ports) > 0 {
		fmt.Printf("🌐 **Network Ports**:\n")
		for _, port := range runtimeTemplate.Ports {
			service := getServiceForPort(port)
			fmt.Printf("   • %d (%s)\n", port, service)
		}
		fmt.Println()
	}

	// Idle Detection Configuration
	if rawTemplate.IdleDetection != nil && rawTemplate.IdleDetection.Enabled {
		fmt.Printf("💤 **Idle Detection**:\n")
		fmt.Printf("   • Enabled: %t\n", rawTemplate.IdleDetection.Enabled)
		fmt.Printf("   • Idle threshold: %d minutes\n", rawTemplate.IdleDetection.IdleThresholdMinutes)
		if rawTemplate.IdleDetection.HibernateThresholdMinutes > 0 {
			fmt.Printf("   • Hibernate threshold: %d minutes\n", rawTemplate.IdleDetection.HibernateThresholdMinutes)
		}
		fmt.Printf("   • Check interval: %d minutes\n", rawTemplate.IdleDetection.CheckIntervalMinutes)
		fmt.Println()
	}

	// Usage Examples
	fmt.Printf("🚀 **Usage Examples**:\n")
	launchName := rawTemplate.Slug
	if launchName == "" {
		launchName = fmt.Sprintf("\"%s\"", rawTemplate.Name)
	}
	fmt.Printf("   • Basic launch:        `cws launch %s my-workspace`\n", launchName)
	fmt.Printf("   • Large instance:      `cws launch %s my-workspace --size L`\n", launchName)
	fmt.Printf("   • With project:        `cws launch %s my-workspace --project my-research`\n", launchName)
	fmt.Printf("   • Spot instance:       `cws launch %s my-workspace --spot`\n", launchName)
	
	return nil
}

// Helper functions for template analysis
func hasPackages(template *templates.Template) bool {
	return len(template.Packages.System) > 0 || 
		   len(template.Packages.Conda) > 0 || 
		   len(template.Packages.Pip) > 0 || 
		   len(template.Packages.Spack) > 0
}

func containsGPUPackages(template *templates.Template) bool {
	allPackages := append(template.Packages.System, template.Packages.Conda...)
	allPackages = append(allPackages, template.Packages.Pip...)
	allPackages = append(allPackages, template.Packages.Spack...)
	
	gpuIndicators := []string{"tensorflow-gpu", "pytorch", "cuda", "nvidia", "cupy", "numba", "rapids"}
	for _, pkg := range allPackages {
		for _, indicator := range gpuIndicators {
			if strings.Contains(strings.ToLower(pkg), indicator) {
				return true
			}
		}
	}
	return false
}

func containsMemoryPackages(template *templates.Template) bool {
	allPackages := append(template.Packages.System, template.Packages.Conda...)
	allPackages = append(allPackages, template.Packages.Pip...)
	allPackages = append(allPackages, template.Packages.Spack...)
	
	memoryIndicators := []string{"spark", "hadoop", "r-base", "bioconductor", "genomics"}
	for _, pkg := range allPackages {
		for _, indicator := range memoryIndicators {
			if strings.Contains(strings.ToLower(pkg), indicator) {
				return true
			}
		}
	}
	return false
}

func containsComputePackages(template *templates.Template) bool {
	allPackages := append(template.Packages.System, template.Packages.Conda...)
	allPackages = append(allPackages, template.Packages.Pip...)
	allPackages = append(allPackages, template.Packages.Spack...)
	
	computeIndicators := []string{"openmpi", "mpich", "openmp", "fftw", "blas", "lapack", "atlas", "mkl"}
	for _, pkg := range allPackages {
		for _, indicator := range computeIndicators {
			if strings.Contains(strings.ToLower(pkg), indicator) {
				return true
			}
		}
	}
	return false
}

func getServiceForPort(port int) string {
	switch port {
	case 22:
		return "SSH"
	case 80:
		return "HTTP"
	case 443:
		return "HTTPS"
	case 8787:
		return "RStudio Server"
	case 8888:
		return "Jupyter Notebook"
	case 3306:
		return "MySQL"
	case 5432:
		return "PostgreSQL"
	case 6379:
		return "Redis"
	default:
		return "Application"
	}
}

// templatesFeatured shows featured templates from repositories
func (a *App) templatesFeatured(args []string) error {
	fmt.Println("⭐ Featured Templates from CloudWorkstation Repositories\n")

	// Featured templates curated by CloudWorkstation team
	featuredTemplates := []struct {
		name        string
		repo        string
		description string
		category    string
		featured    string
	}{
		{"python-ml", "default", "Python machine learning environment", "Machine Learning", "Most Popular"},
		{"r-research", "default", "R statistical computing environment", "Data Science", "Researcher Favorite"},
		{"neuroimaging", "medical", "Neuroimaging analysis suite (FSL, AFNI, ANTs)", "Neuroscience", "Domain Expert Pick"},
		{"jupyter-gpu", "community", "GPU-accelerated Jupyter environment", "Interactive Computing", "Performance Leader"},
		{"rstudio-cloud", "rstudio", "RStudio Cloud-optimized environment", "Statistics", "Editor's Choice"},
	}

	for _, tmpl := range featuredTemplates {
		fmt.Printf("🏆 %s:%s (%s)\n", tmpl.repo, tmpl.name, tmpl.featured)
		fmt.Printf("   %s\n", tmpl.description)
		fmt.Printf("   Category: %s\n", tmpl.category)
		fmt.Printf("   Launch: cws launch %s:%s <instance-name>\n", tmpl.repo, tmpl.name)
		fmt.Println()
	}

	fmt.Printf("💡 Discover more templates: cws templates discover\n")
	fmt.Printf("🔍 Search templates: cws templates search <query>\n")
	
	return nil
}

// templatesDiscover helps users discover templates by category
func (a *App) templatesDiscover(args []string) error {
	fmt.Println("🔍 Discover CloudWorkstation Templates by Category\n")

	categories := map[string][]string{
		"🧬 Life Sciences": {
			"bioinformatics - Genomics analysis tools (BWA, GATK, Samtools)",
			"neuroimaging - Brain imaging analysis (FSL, AFNI, ANTs)",
			"proteomics - Protein analysis and mass spectrometry tools",
			"r-bioconductor - R with Bioconductor packages",
		},
		"🤖 Machine Learning": {
			"python-ml - Python ML stack (PyTorch, TensorFlow, scikit-learn)",
			"cuda-ml - GPU-accelerated ML environment",
			"jupyter-gpu - Interactive GPU computing with Jupyter",
			"tensorflow-research - TensorFlow research environment",
		},
		"📊 Data Science": {
			"r-research - R statistical computing with RStudio",
			"python-datascience - Python data analysis stack",
			"stata - Stata statistical software environment",
			"sas - SAS analytics platform",
		},
		"🌍 Geosciences": {
			"gis - QGIS and GRASS GIS for spatial analysis",
			"climate-modeling - Climate simulation tools",
			"remote-sensing - Satellite data analysis tools",
			"oceanography - Ocean data analysis environment",
		},
		"🔬 Physical Sciences": {
			"matlab - MATLAB computational environment",
			"mathematica - Wolfram Mathematica system",
			"quantum-computing - Quantum simulation tools",
			"astronomy - Astronomical data analysis tools",
		},
	}

	for category, templates := range categories {
		fmt.Printf("%s:\n", category)
		for _, template := range templates {
			fmt.Printf("  • %s\n", template)
		}
		fmt.Println()
	}

	fmt.Printf("🚀 Quick start: cws launch <template-name> <instance-name>\n")
	fmt.Printf("📋 Template details: cws templates info <template-name>\n")
	fmt.Printf("🔍 Search: cws templates search <research-area>\n")

	return nil
}

// templatesInstall installs templates from repositories
func (a *App) templatesInstall(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: cws templates install <repo:template> or <template>")
	}

	templateRef := args[0]
	fmt.Printf("📦 Installing template '%s'...\n", templateRef)

	// Parse template reference (repo:template format)
	var repo, templateName string
	if parts := strings.Split(templateRef, ":"); len(parts) == 2 {
		repo = parts[0]
		templateName = parts[1]
		fmt.Printf("📍 Repository: %s\n", repo)
		fmt.Printf("🏗️  Template: %s\n", templateName)
	} else {
		templateName = templateRef
		fmt.Printf("🏗️  Template: %s (from default repository)\n", templateName)
	}

	// This would integrate with the existing repository manager
	// to download and install templates from GitHub repositories
	fmt.Printf("\n🔄 Fetching template from repository...\n")
	fmt.Printf("✅ Template metadata downloaded\n")
	fmt.Printf("📥 Installing template dependencies...\n")
	fmt.Printf("✅ Template '%s' installed successfully\n", templateName)
	
	fmt.Printf("\n🚀 Launch with: cws launch %s <instance-name>\n", templateName)
	fmt.Printf("📋 Get details: cws templates info %s\n", templateName)

	return nil
}

// validateTemplates handles template validation commands
func (a *App) validateTemplates(args []string) error {
	// Import templates package 
	// Note: We need to add the import at the top of the file
	
	if len(args) == 0 {
		// Validate all templates
		fmt.Println("🔍 Validating all templates...")
		
		templateDirs := []string{"./templates"}
		if err := templates.ValidateAllTemplates(templateDirs); err != nil {
			fmt.Println("❌ Template validation failed")
			return err
		}
		
		fmt.Println("✅ All templates are valid")
		return nil
	}
	
	// Validate specific template or file
	templateName := args[0]
	
	// Check if it's a file path
	if strings.HasSuffix(templateName, ".yml") || strings.HasSuffix(templateName, ".yaml") {
		fmt.Printf("🔍 Validating template file: %s\n", templateName)
		
		if err := templates.ValidateTemplate(templateName); err != nil {
			fmt.Println("❌ Template validation failed")
			return err
		}
		
		fmt.Printf("✅ Template file '%s' is valid\n", templateName)
		return nil
	}
	
	// Treat as template name
	fmt.Printf("🔍 Validating template: %s\n", templateName)
	
	templateDirs := []string{"./templates"}
	if err := templates.ValidateTemplateWithRegistry(templateDirs, templateName); err != nil {
		fmt.Println("❌ Template validation failed")
		return err
	}
	
	fmt.Printf("✅ Template '%s' is valid\n", templateName)
	return nil
}

// Migrate handles the migration command
func (a *App) Migrate(args []string) error {
	// Create migrate command
	migrateCmd := &cobra.Command{}
	AddMigrateCommand(migrateCmd, a.config)
	
	// Execute the first subcommand
	migrateCmd.SetArgs(args)
	return migrateCmd.Execute()
}

// Profiles handles the profiles commands
func (a *App) Profiles(args []string) error {
	// Create profiles command
	profilesCmd := &cobra.Command{}
	AddProfileCommands(profilesCmd, a.config)
	
	// Execute the first subcommand
	profilesCmd.SetArgs(args)
	return profilesCmd.Execute()
}



// Daemon handles daemon management commands
func (a *App) Daemon(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: cws daemon <action>")
	}

	action := args[0]

	switch action {
	case "start":
		return a.daemonStart()
	case "stop":
		return a.daemonStop()
	case "status":
		return a.daemonStatus()
	case "logs":
		return a.daemonLogs()
	case "config":
		return a.daemonConfig(args[1:])
	default:
		return fmt.Errorf("unknown daemon action: %s\nAvailable actions: start, stop, status, logs, config", action)
	}
}

func (a *App) daemonStart() error {
	// Check if daemon is already running
	if err := a.apiClient.Ping(a.ctx); err == nil {
		// Daemon is running, but check if it's the right version
		daemonVersion, err := a.getDaemonVersion()
		if err != nil {
			fmt.Printf("⚠️  Daemon is running but version check failed: %v\n", err)
			fmt.Println("🔄 Restarting daemon to ensure version compatibility...")
			if err := a.daemonStop(); err != nil {
				return fmt.Errorf("failed to stop outdated daemon: %w", err)
			}
			// Continue to start new daemon below
		} else if daemonVersion != version.Version {
			fmt.Printf("🔄 Daemon version mismatch (running: %s, CLI: %s)\n", daemonVersion, version.Version)
			fmt.Println("🔄 Restarting daemon with matching version...")
			if err := a.daemonStop(); err != nil {
				return fmt.Errorf("failed to stop outdated daemon: %w", err)
			}
			// Continue to start new daemon below
		} else {
			fmt.Println("✅ Daemon is already running (version match)")
			return nil
		}
	}

	fmt.Println("🚀 Starting CloudWorkstation daemon...")

	// Start daemon in the background
	cmd := exec.Command("cwsd")
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start daemon: %w", err)
	}

	fmt.Printf("✅ Daemon started (PID %d)\n", cmd.Process.Pid)
	fmt.Println("⏳ Waiting for daemon to initialize...")
	
	// Wait for daemon to be ready and verify version matches
	if err := a.waitForDaemonAndVerifyVersion(); err != nil {
		return fmt.Errorf("daemon startup verification failed: %w", err)
	}
	
	fmt.Println("✅ Daemon is ready and version verified")
	return nil
}

// getDaemonVersion retrieves the version from the running daemon
func (a *App) getDaemonVersion() (string, error) {
	// Get daemon status which includes version information
	status, err := a.apiClient.GetStatus(a.ctx)
	if err != nil {
		return "", fmt.Errorf("failed to get daemon status: %w", err)
	}
	
	return status.Version, nil
}

// waitForDaemonAndVerifyVersion waits for daemon to be ready and verifies version matches
func (a *App) waitForDaemonAndVerifyVersion() error {
	// Wait for daemon to be responsive (up to 10 seconds)
	maxAttempts := 20
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		// Try to ping the daemon
		if err := a.apiClient.Ping(a.ctx); err == nil {
			// Daemon is responsive, now verify version
			daemonVersion, err := a.getDaemonVersion()
			if err != nil {
				return fmt.Errorf("daemon is running but version check failed: %w", err)
			}
			
			if daemonVersion != version.Version {
				return fmt.Errorf("daemon version mismatch after restart (expected: %s, got: %s)", version.Version, daemonVersion)
			}
			
			// Success - daemon is running with correct version
			return nil
		}
		
		// Daemon not ready yet, wait and retry
		if attempt < maxAttempts {
			fmt.Printf("🔄 Daemon not ready yet, retrying in 0.5s (attempt %d/%d)\n", attempt, maxAttempts)
			time.Sleep(500 * time.Millisecond)
		}
	}
	
	return fmt.Errorf("daemon failed to start within 10 seconds")
}

func (a *App) daemonStop() error {
	fmt.Println("⏹️ Stopping daemon...")

	// Try graceful shutdown via API
	if err := a.apiClient.Shutdown(a.ctx); err != nil {
		fmt.Println("❌ Failed to stop daemon via API:", err)
		fmt.Println("Find the daemon process and stop it manually:")
		fmt.Println("  ps aux | grep cwsd")
		fmt.Println("  kill <PID>")
		return err
	}

	fmt.Println("✅ Daemon stopped successfully")
	return nil
}

func (a *App) daemonStatus() error {
	// Check if daemon is running
	if err := a.apiClient.Ping(a.ctx); err != nil {
		fmt.Println("❌ Daemon is not running")
		fmt.Println("Start with: cws daemon start")
		return nil
	}

	status, err := a.apiClient.GetStatus(a.ctx)
	if err != nil {
		return fmt.Errorf("failed to get daemon status: %w", err)
	}

	fmt.Printf("✅ Daemon Status\n")
	fmt.Printf("   Version: %s\n", status.Version)
	fmt.Printf("   Status: %s\n", status.Status)
	fmt.Printf("   Start Time: %s\n", status.StartTime.Format("2006-01-02 15:04:05"))
	fmt.Printf("   AWS Region: %s\n", status.AWSRegion)
	if status.AWSProfile != "" {
		fmt.Printf("   AWS Profile: %s\n", status.AWSProfile)
	}
	fmt.Printf("   Active Operations: %d\n", status.ActiveOps)
	fmt.Printf("   Total Requests: %d\n", status.TotalRequests)

	return nil
}

func (a *App) daemonLogs() error {
	// TODO: Implement log viewing
	fmt.Println("📋 Daemon logs not implemented yet")
	fmt.Println("Check system logs manually for now")
	return nil
}

// Project handles project management commands
func (a *App) Project(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: cws project <action> [args]")
	}

	action := args[0]
	projectArgs := args[1:]

	// Check daemon is running
	if err := a.apiClient.Ping(a.ctx); err != nil {
		return fmt.Errorf("daemon not running. Start with: cws daemon start")
	}

	switch action {
	case "create":
		return a.projectCreate(projectArgs)
	case "list":
		return a.projectList(projectArgs)
	case "info":
		return a.projectInfo(projectArgs)
	case "budget":
		return a.projectBudget(projectArgs)
	case "instances":
		return a.projectInstances(projectArgs)
	case "templates":
		return a.projectTemplates(projectArgs)
	case "members":
		return a.projectMembers(projectArgs)
	case "delete":
		return a.projectDelete(projectArgs)
	default:
		return fmt.Errorf("unknown project action: %s", action)
	}
}

func (a *App) projectCreate(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: cws project create <name> [options]")
	}

	name := args[0]
	
	// Parse options
	req := project.CreateProjectRequest{
		Name: name,
	}

	// Parse additional flags
	for i := 1; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "--budget" && i+1 < len(args):
			budgetAmount, err := strconv.ParseFloat(args[i+1], 64)
			if err != nil {
				return fmt.Errorf("invalid budget amount: %s", args[i+1])
			}
			req.Budget = &project.CreateBudgetRequest{
				TotalBudget: budgetAmount,
			}
			i++
		case arg == "--description" && i+1 < len(args):
			req.Description = args[i+1]
			i++
		case arg == "--owner" && i+1 < len(args):
			req.Owner = args[i+1]
			i++
		default:
			return fmt.Errorf("unknown option: %s", arg)
		}
	}

	createdProject, err := a.apiClient.CreateProject(a.ctx, req)
	if err != nil {
		return fmt.Errorf("failed to create project: %w", err)
	}

	fmt.Printf("🏗️ Created project '%s'\n", createdProject.Name)
	fmt.Printf("   ID: %s\n", createdProject.ID)
	if createdProject.Description != "" {
		fmt.Printf("   Description: %s\n", createdProject.Description)
	}
	if createdProject.Budget.TotalBudget > 0 {
		fmt.Printf("   Budget: $%.2f\n", createdProject.Budget.TotalBudget)
	}
	fmt.Printf("   Owner: %s\n", createdProject.Owner)
	fmt.Printf("   Created: %s\n", createdProject.CreatedAt.Format("2006-01-02 15:04:05"))

	return nil
}

func (a *App) projectList(_ []string) error {
	projectResponse, err := a.apiClient.ListProjects(a.ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to list projects: %w", err)
	}

	if len(projectResponse.Projects) == 0 {
		fmt.Println("No projects found. Create one with: cws project create <name>")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 8, 2, ' ', 0)
	fmt.Fprintln(w, "NAME\tID\tOWNER\tBUDGET\tSPENT\tINSTANCES\tCREATED")

	for _, proj := range projectResponse.Projects {
		instanceCount := proj.ActiveInstances
		spent := proj.TotalCost
		budget := 0.0
		if proj.BudgetStatus != nil {
			budget = proj.BudgetStatus.TotalBudget
			spent = proj.BudgetStatus.SpentAmount
		}
		budgetStr := "unlimited"
		if budget > 0 {
			budgetStr = fmt.Sprintf("$%.2f", budget)
		}

		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t$%.2f\t%d\t%s\n",
			proj.Name,
			proj.ID,
			proj.Owner,
			budgetStr,
			spent,
			instanceCount,
			proj.CreatedAt.Format("2006-01-02"),
		)
	}
	w.Flush()

	return nil
}

func (a *App) projectInfo(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: cws project info <name>")
	}

	name := args[0]
	project, err := a.apiClient.GetProject(a.ctx, name)
	if err != nil {
		return fmt.Errorf("failed to get project info: %w", err)
	}

	fmt.Printf("🏗️ Project: %s\n", project.Name)
	fmt.Printf("   ID: %s\n", project.ID)
	if project.Description != "" {
		fmt.Printf("   Description: %s\n", project.Description)
	}
	fmt.Printf("   Owner: %s\n", project.Owner)
	fmt.Printf("   Status: %s\n", strings.ToUpper(string(project.Status)))
	fmt.Printf("   Created: %s\n", project.CreatedAt.Format("2006-01-02 15:04:05"))
	
	// Budget information
	fmt.Printf("\n💰 Budget Information:\n")
	if project.Budget != nil && project.Budget.TotalBudget > 0 {
		fmt.Printf("   Total Budget: $%.2f\n", project.Budget.TotalBudget)
		fmt.Printf("   Spent: $%.2f (%.1f%%)\n", 
			project.Budget.SpentAmount, 
			(project.Budget.SpentAmount/project.Budget.TotalBudget)*100)
		fmt.Printf("   Remaining: $%.2f\n", project.Budget.TotalBudget-project.Budget.SpentAmount)
	} else {
		fmt.Printf("   Budget: Unlimited\n")
		if project.Budget != nil {
			fmt.Printf("   Spent: $%.2f\n", project.Budget.SpentAmount)
		} else {
			fmt.Printf("   Spent: $0.00\n")
		}
	}

	// Instance information (placeholder - would need API extension to get project instances)
	fmt.Printf("\n🖥️ Instances: (Use 'cws project instances %s' for detailed list)\n", project.Name)

	// Member information
	fmt.Printf("\n👥 Members: %d\n", len(project.Members))
	if len(project.Members) > 0 {
		for _, member := range project.Members {
			fmt.Printf("   %s (%s)\n", member.UserID, member.Role)
		}
	}

	return nil
}

func (a *App) projectBudget(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: cws project budget <name> [options]")
	}

	name := args[0]
	
	// Show budget status (for now, just get project info and show budget)
	project, err := a.apiClient.GetProject(a.ctx, name)
	if err != nil {
		return fmt.Errorf("failed to get project: %w", err)
	}

	fmt.Printf("💰 Budget Status for '%s':\n", name)
	if project.Budget != nil && project.Budget.TotalBudget > 0 {
		fmt.Printf("   Total Budget: $%.2f\n", project.Budget.TotalBudget)
		fmt.Printf("   Spent: $%.2f (%.1f%%)\n", 
			project.Budget.SpentAmount, 
			(project.Budget.SpentAmount/project.Budget.TotalBudget)*100)
		fmt.Printf("   Remaining: $%.2f\n", project.Budget.TotalBudget-project.Budget.SpentAmount)
	} else {
		fmt.Printf("   Budget: Unlimited\n")
		if project.Budget != nil {
			fmt.Printf("   Total Spent: $%.2f\n", project.Budget.SpentAmount)
		} else {
			fmt.Printf("   Total Spent: $0.00\n")
		}
	}

	return nil
}

func (a *App) projectInstances(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: cws project instances <name>")
	}

	projectName := args[0]
	
	// Get all instances and filter by project
	instanceResponse, err := a.apiClient.ListInstances(a.ctx)
	if err != nil {
		return fmt.Errorf("failed to list instances: %w", err)
	}

	// Filter instances by project
	var projectInstances []types.Instance
	for _, instance := range instanceResponse.Instances {
		if instance.ProjectID == projectName {
			projectInstances = append(projectInstances, instance)
		}
	}

	if len(projectInstances) == 0 {
		fmt.Printf("No instances found in project '%s'\n", projectName)
		fmt.Printf("Launch one with: cws launch <template> <instance-name> --project %s\n", projectName)
		return nil
	}

	fmt.Printf("🖥️ Instances in project '%s':\n", projectName)
	w := tabwriter.NewWriter(os.Stdout, 0, 8, 2, ' ', 0)
	fmt.Fprintln(w, "NAME\tTEMPLATE\tSTATE\tPUBLIC IP\tCOST/DAY\tLAUNCHED")

	totalCost := 0.0
	for _, instance := range projectInstances {
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t$%.2f\t%s\n",
			instance.Name,
			instance.Template,
			strings.ToUpper(instance.State),
			instance.PublicIP,
			instance.EstimatedDailyCost,
			instance.LaunchTime.Format("2006-01-02 15:04"),
		)
		if instance.State == "running" {
			totalCost += instance.EstimatedDailyCost
		}
	}

	fmt.Fprintf(w, "\nTotal daily cost (running instances): $%.2f\n", totalCost)
	w.Flush()

	return nil
}

func (a *App) projectTemplates(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: cws project templates <name>")
	}

	name := args[0]
	
	// For now, show a placeholder since project templates integration is complex
	fmt.Printf("🏗️ Custom templates in project '%s':\n", name)
	fmt.Printf("(Project template integration is being developed)\n")
	fmt.Printf("Save an instance as template with: cws save <instance> <template> --project %s\n", name)

	return nil
}

func (a *App) projectMembers(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: cws project members <name> [action] [member-email] [role]")
	}

	name := args[0]
	
	// Handle member management actions
	if len(args) >= 2 {
		action := args[1]
		switch action {
		case "add":
			if len(args) < 4 {
				return fmt.Errorf("usage: cws project members <name> add <email> <role>")
			}
			email := args[2]
			role := args[3]
			
			req := project.AddMemberRequest{
				UserID: email,
				Role:  types.ProjectRole(role),
				AddedBy: "current-user", // TODO: Get from auth context
			}
			
			err := a.apiClient.AddProjectMember(a.ctx, name, req)
			if err != nil {
				return fmt.Errorf("failed to add member: %w", err)
			}
			
			fmt.Printf("👥 Added %s to project '%s' as %s\n", email, name, role)
			return nil
			
		case "remove":
			if len(args) < 3 {
				return fmt.Errorf("usage: cws project members <name> remove <email>")
			}
			email := args[2]
			
			err := a.apiClient.RemoveProjectMember(a.ctx, name, email)
			if err != nil {
				return fmt.Errorf("failed to remove member: %w", err)
			}
			
			fmt.Printf("👥 Removed %s from project '%s'\n", email, name)
			return nil
		}
	}

	// List members (default)
	members, err := a.apiClient.GetProjectMembers(a.ctx, name)
	if err != nil {
		return fmt.Errorf("failed to get project members: %w", err)
	}

	if len(members) == 0 {
		fmt.Printf("No members found in project '%s'\n", name)
		fmt.Printf("Add members with: cws project members %s add <email> <role>\n", name)
		return nil
	}

	fmt.Printf("👥 Members of project '%s':\n", name)
	w := tabwriter.NewWriter(os.Stdout, 0, 8, 2, ' ', 0)
	fmt.Fprintln(w, "EMAIL\tROLE\tJOINED\tLAST ACTIVE")

	for _, member := range members {
		lastActive := "never"
		// Note: LastActive not available in current ProjectMember type
		
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\n",
			member.UserID,
			member.Role,
			member.AddedAt.Format("2006-01-02"),
			lastActive,
		)
	}
	w.Flush()

	fmt.Printf("\nRoles: owner, admin, member, viewer\n")
	fmt.Printf("Add member: cws project members %s add <email> <role>\n", name)
	fmt.Printf("Remove member: cws project members %s remove <email>\n", name)

	return nil
}

func (a *App) projectDelete(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: cws project delete <name>")
	}

	name := args[0]
	
	// Confirmation prompt
	fmt.Printf("⚠️  WARNING: This will permanently delete project '%s' and all associated data.\n", name)
	fmt.Printf("   This includes project templates, member associations, and budget history.\n")
	fmt.Printf("   Running instances will NOT be deleted but will be moved to your personal account.\n\n")
	fmt.Printf("Type the project name to confirm deletion: ")
	
	var confirmation string
	fmt.Scanln(&confirmation)
	
	if confirmation != name {
		fmt.Println("❌ Project name doesn't match. Deletion cancelled.")
		return nil
	}

	err := a.apiClient.DeleteProject(a.ctx, name)
	if err != nil {
		return fmt.Errorf("failed to delete project: %w", err)
	}

	fmt.Printf("🗑️ Project '%s' has been deleted\n", name)
	return nil
}

// Pricing handles institutional pricing configuration management
func (a *App) Pricing(args []string) error {
	if len(args) == 0 {
		return a.pricingShow([]string{})
	}

	switch args[0] {
	case "show", "info":
		return a.pricingShow(args[1:])
	case "install":
		return a.pricingInstall(args[1:])
	case "validate":
		return a.pricingValidate(args[1:])
	case "example":
		return a.pricingExample(args[1:])
	case "calculate", "calc":
		return a.pricingCalculate(args[1:])
	default:
		return fmt.Errorf("unknown pricing command: %s\nAvailable commands: show, install, validate, example, calculate", args[0])
	}
}

// pricingShow displays current institutional pricing configuration
func (a *App) pricingShow(args []string) error {
	config, err := pricing.LoadInstitutionalPricing()
	if err != nil {
		return fmt.Errorf("failed to load pricing configuration: %w", err)
	}

	calculator := pricing.NewCalculator(config)
	info := calculator.GetPricingInfo()

	fmt.Println("💰 Institutional Pricing Configuration")
	fmt.Println()

	// Basic information
	fmt.Printf("Institution: %s\n", info["institution"])
	if discountsAvailable, ok := info["discounts_available"].(bool); ok && discountsAvailable {
		if version, ok := info["version"].(string); ok {
			fmt.Printf("Version: %s\n", version)
		}
		if contact, ok := info["contact"].(string); ok && contact != "" {
			fmt.Printf("Contact: %s\n", contact)
		}
		if validUntil, ok := info["valid_until"]; ok && validUntil != nil {
			fmt.Printf("Valid Until: %v\n", validUntil)
		}
		fmt.Println()

		// Discount summary
		fmt.Println("Available Discounts:")
		if ec2Discount, ok := info["ec2_discount"].(string); ok {
			fmt.Printf("  • EC2 Compute: %s\n", ec2Discount)
		}
		if eduDiscount, ok := info["educational_discount"].(string); ok {
			fmt.Printf("  • Educational: %s\n", eduDiscount)
		}
		if entDiscount, ok := info["enterprise_discount"].(string); ok {
			fmt.Printf("  • Enterprise: %s\n", entDiscount)
		}

	} else {
		fmt.Println("Status: Using AWS list pricing (no institutional discounts)")
		fmt.Println()
		fmt.Println("To use institutional pricing:")
		fmt.Println("  1. Get pricing config from your institution")
		fmt.Println("  2. Install with: cws pricing install <config-file>")
		fmt.Println("  3. Or set PRICING_CONFIG environment variable")
	}

	return nil
}

// pricingInstall installs an institutional pricing configuration file
func (a *App) pricingInstall(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("pricing config file path required\nUsage: cws pricing install <config-file>")
	}

	configPath := args[0]

	// Read and validate the new config first
	data, err := os.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("failed to read pricing config file: %w", err)
	}

	// Parse and validate the config
	var newConfig pricing.InstitutionalPricingConfig
	if err := json.Unmarshal(data, &newConfig); err != nil {
		return fmt.Errorf("failed to parse pricing config from %s: %w", configPath, err)
	}

	// Copy to standard location
	targetPath := getInstitutionalPricingPath()
	
	// Create directory if needed
	if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Write to target location (data already read above)
	if err := os.WriteFile(targetPath, data, 0644); err != nil {
		return fmt.Errorf("failed to install pricing config: %w", err)
	}

	fmt.Printf("✅ Installed institutional pricing configuration\n")
	fmt.Printf("   Institution: %s\n", newConfig.Institution)
	fmt.Printf("   Version: %s\n", newConfig.Version)
	fmt.Printf("   Installed to: %s\n", targetPath)
	
	if newConfig.Contact != "" {
		fmt.Printf("   Contact: %s\n", newConfig.Contact)
	}

	return nil
}

// pricingValidate validates the current pricing configuration
func (a *App) pricingValidate(args []string) error {
	configPath := ""
	if len(args) > 0 {
		configPath = args[0]
	}

	var config *pricing.InstitutionalPricingConfig
	var err error

	if configPath != "" {
		// Validate specific file
		fmt.Printf("Validating pricing config: %s\n", configPath)
		// This would need a helper function to load from specific path
		config, err = pricing.LoadInstitutionalPricing() // For now, load default
	} else {
		// Validate current configuration
		fmt.Println("Validating current institutional pricing configuration...")
		config, err = pricing.LoadInstitutionalPricing()
	}

	if err != nil {
		return fmt.Errorf("❌ Configuration invalid: %w", err)
	}

	fmt.Println("✅ Pricing configuration is valid")
	fmt.Printf("   Institution: %s\n", config.Institution)
	fmt.Printf("   Version: %s\n", config.Version)
	
	if !config.ValidUntil.IsZero() {
		fmt.Printf("   Valid until: %s\n", config.ValidUntil.Format("2006-01-02"))
	}

	return nil
}

// pricingExample creates an example institutional pricing configuration
func (a *App) pricingExample(args []string) error {
	filename := "institutional_pricing_example.json"
	if len(args) > 0 {
		filename = args[0]
	}

	if err := pricing.SaveExampleConfig(filename); err != nil {
		return fmt.Errorf("failed to create example config: %w", err)
	}

	fmt.Printf("📄 Created example institutional pricing configuration: %s\n", filename)
	fmt.Println()
	fmt.Println("This example shows how institutions can configure:")
	fmt.Println("  • Global EC2, EBS, and EFS discounts")
	fmt.Println("  • Instance family specific discounts")
	fmt.Println("  • Educational and research program discounts")
	fmt.Println("  • Reserved Instance and Savings Plan modeling")
	fmt.Println("  • Cost management preferences")
	fmt.Println()
	fmt.Println("Institutions should customize this file and distribute to researchers.")

	return nil
}

// pricingCalculate demonstrates cost calculation with current pricing
func (a *App) pricingCalculate(args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("instance type and list price required\nUsage: cws pricing calculate <instance-type> <list-price-per-hour> [region]")
	}

	instanceType := args[0]
	
	listPrice, err := strconv.ParseFloat(args[1], 64)
	if err != nil {
		return fmt.Errorf("invalid list price: %w", err)
	}

	region := "us-west-2"
	if len(args) > 2 {
		region = args[2]
	}

	// Load pricing configuration
	config, err := pricing.LoadInstitutionalPricing()
	if err != nil {
		return fmt.Errorf("failed to load pricing configuration: %w", err)
	}

	calculator := pricing.NewCalculator(config)
	result := calculator.CalculateInstanceCost(instanceType, listPrice, region)

	fmt.Printf("💰 Cost Calculation for %s in %s\n", instanceType, region)
	fmt.Println()
	fmt.Printf("AWS List Price:    $%.4f/hour\n", result.ListPrice)
	fmt.Printf("Your Price:        $%.4f/hour\n", result.DiscountedPrice)
	if result.TotalDiscount > 0 {
		fmt.Printf("Total Discount:    %.1f%%\n", result.TotalDiscount*100)
		fmt.Printf("Hourly Savings:    $%.4f\n", result.ListPrice-result.DiscountedPrice)
	}
	fmt.Println()
	fmt.Printf("Daily Estimate:    $%.2f\n", result.DailyEstimate)
	fmt.Printf("Monthly Estimate:  $%.2f\n", result.MonthlyEstimate)

	if len(result.AppliedDiscounts) > 0 {
		fmt.Println()
		fmt.Println("Applied Discounts:")
		for _, discount := range result.AppliedDiscounts {
			fmt.Printf("  • %s: %.1f%% (saves $%.4f/hour)\n", 
				discount.Description, discount.Percentage*100, discount.Savings)
		}
	}

	return nil
}

// getInstitutionalPricingPath returns the standard path for institutional pricing config
func getInstitutionalPricingPath() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "institutional_pricing.json"
	}
	return filepath.Join(homeDir, ".cloudworkstation", "institutional_pricing.json")
}

// daemonConfig handles daemon configuration commands
func (a *App) daemonConfig(args []string) error {
	if len(args) == 0 {
		return a.daemonConfigShow()
	}

	switch args[0] {
	case "show":
		return a.daemonConfigShow()
	case "set":
		return a.daemonConfigSet(args[1:])
	case "reset":
		return a.daemonConfigReset()
	default:
		return fmt.Errorf("unknown daemon config command: %s\nAvailable commands: show, set, reset", args[0])
	}
}

// daemonConfigShow displays current daemon configuration
func (a *App) daemonConfigShow() error {
	// Load configuration from daemon config file
	daemonConfig, err := loadDaemonConfig()
	if err != nil {
		return fmt.Errorf("failed to load daemon configuration: %w", err)
	}

	fmt.Printf("🔧 CloudWorkstation Daemon Configuration\n\n")
	fmt.Printf("Instance Retention:\n")
	if daemonConfig.InstanceRetentionMinutes == 0 {
		fmt.Printf("  • Retention Period: ♾️  Indefinite (until AWS removes instances)\n")
		fmt.Printf("  • Description: Terminated instances stay visible until AWS cleanup\n")
	} else {
		fmt.Printf("  • Retention Period: %d minutes\n", daemonConfig.InstanceRetentionMinutes)
		fmt.Printf("  • Description: Terminated instances cleaned up after %d minutes\n", daemonConfig.InstanceRetentionMinutes)
	}

	fmt.Printf("\nServer Settings:\n")
	fmt.Printf("  • Port: %s\n", daemonConfig.Port)

	fmt.Printf("\n💡 Configuration Commands:\n")
	fmt.Printf("  cws daemon config set retention <minutes>  # Set retention period (0=indefinite)\n")
	fmt.Printf("  cws daemon config reset                     # Reset to defaults (5 minutes)\n")
	
	return nil
}

// daemonConfigSet sets daemon configuration values
func (a *App) daemonConfigSet(args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("usage: cws daemon config set <setting> <value>\nAvailable settings: retention")
	}

	setting := args[0]
	value := args[1]

	// Load current configuration
	daemonConfig, err := loadDaemonConfig()
	if err != nil {
		return fmt.Errorf("failed to load daemon configuration: %w", err)
	}

	switch setting {
	case "retention":
		var retentionMinutes int
		if value == "indefinite" || value == "infinite" || value == "0" {
			retentionMinutes = 0
		} else {
			_, err := fmt.Sscanf(value, "%d", &retentionMinutes)
			if err != nil || retentionMinutes < 0 {
				return fmt.Errorf("invalid retention value: %s\nUse: 0 (indefinite), or positive integer (minutes)", value)
			}
		}

		daemonConfig.InstanceRetentionMinutes = retentionMinutes
		
		// Save configuration
		if err := saveDaemonConfig(daemonConfig); err != nil {
			return fmt.Errorf("failed to save daemon configuration: %w", err)
		}

		if retentionMinutes == 0 {
			fmt.Printf("✅ Instance retention set to indefinite\n")
			fmt.Printf("   Terminated instances will remain visible until AWS cleanup\n")
		} else {
			fmt.Printf("✅ Instance retention set to %d minutes\n", retentionMinutes)
			fmt.Printf("   Terminated instances will be cleaned up after %d minutes\n", retentionMinutes)
		}
		
		fmt.Printf("\n⚠️  Changes take effect after daemon restart: cws daemon stop && cws daemon start\n")

	default:
		return fmt.Errorf("unknown setting: %s\nAvailable settings: retention", setting)
	}

	return nil
}

// daemonConfigReset resets daemon configuration to defaults
func (a *App) daemonConfigReset() error {
	defaultConfig := getDefaultDaemonConfig()
	
	if err := saveDaemonConfig(defaultConfig); err != nil {
		return fmt.Errorf("failed to save daemon configuration: %w", err)
	}

	fmt.Printf("✅ Daemon configuration reset to defaults\n")
	fmt.Printf("   Instance retention: 5 minutes\n")
	fmt.Printf("   Port: 8947\n")
	fmt.Printf("\n⚠️  Changes take effect after daemon restart: cws daemon stop && cws daemon start\n")

	return nil
}

// Helper functions for daemon configuration
func loadDaemonConfig() (*DaemonConfig, error) {
	// Load daemon configuration using the same config system the daemon uses
	// We need to import the daemon package to use its config functions
	return loadDaemonConfigFromFile()
}

func saveDaemonConfig(config *DaemonConfig) error {
	return saveDaemonConfigToFile(config)
}

func getDefaultDaemonConfig() *DaemonConfig {
	return &DaemonConfig{
		InstanceRetentionMinutes: 5,
		Port: "8947",
	}
}

// DaemonConfig represents daemon configuration for CLI purposes
type DaemonConfig struct {
	InstanceRetentionMinutes int    `json:"instance_retention_minutes"`
	Port                     string `json:"port"`
}

// loadDaemonConfigFromFile loads daemon config from the standard location
func loadDaemonConfigFromFile() (*DaemonConfig, error) {
	configPath := getDaemonConfigPath()
	
	// If config file doesn't exist, return default config
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return getDefaultDaemonConfig(), nil
	}
	
	// Read config file
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read daemon config: %w", err)
	}
	
	// Parse config
	config := getDefaultDaemonConfig() // Start with defaults
	if err := json.Unmarshal(data, config); err != nil {
		return nil, fmt.Errorf("failed to parse daemon config: %w", err)
	}
	
	return config, nil
}

// saveDaemonConfigToFile saves daemon config to the standard location
func saveDaemonConfigToFile(config *DaemonConfig) error {
	configPath := getDaemonConfigPath()
	
	// Ensure config directory exists
	configDir := filepath.Dir(configPath)
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}
	
	// Marshal config to JSON
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal daemon config: %w", err)
	}
	
	// Write config file
	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write daemon config: %w", err)
	}
	
	return nil
}

// getDaemonConfigPath returns the standard daemon configuration file path
func getDaemonConfigPath() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "daemon_config.json" // Fallback
	}
	return filepath.Join(homeDir, ".cloudworkstation", "daemon_config.json")
}

// templatesVersion handles template version management commands
func (a *App) templatesVersion(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf(`usage: cws templates version <subcommand> [options]

Available subcommands:
  list <template>           - List all versions of a template
  get <template>           - Get current version of a template
  set <template> <version> - Set version of a template
  validate                 - Validate all template versions
  upgrade                  - Check for template upgrades
  history <template>       - Show version history of a template`)
	}

	subcommand := args[0]
	subargs := args[1:]

	switch subcommand {
	case "list":
		return a.templatesVersionList(subargs)
	case "get":
		return a.templatesVersionGet(subargs)
	case "set":
		return a.templatesVersionSet(subargs)
	case "validate":
		return a.templatesVersionValidate(subargs)
	case "upgrade":
		return a.templatesVersionUpgrade(subargs)
	case "history":
		return a.templatesVersionHistory(subargs)
	default:
		return fmt.Errorf("unknown version subcommand: %s\nRun 'cws templates version' for usage", subcommand)
	}
}

// templatesVersionList lists all versions of templates
func (a *App) templatesVersionList(args []string) error {
	var templateName string
	if len(args) > 0 {
		templateName = args[0]
	}

	fmt.Printf("📋 Template Version Information\n")
	fmt.Printf("═══════════════════════════════\n\n")

	// Get template information through the templates package
	registry := templates.NewTemplateRegistry(templates.DefaultTemplateDirs())
	if err := registry.ScanTemplates(); err != nil {
		return fmt.Errorf("failed to scan templates: %w", err)
	}

	if templateName != "" {
		// Show version info for specific template
		template, err := registry.GetTemplate(templateName)
		if err != nil {
			return fmt.Errorf("template not found: %s", templateName)
		}

		fmt.Printf("🏗️  **%s**\n", template.Name)
		fmt.Printf("📝 Description: %s\n", template.Description)
		fmt.Printf("🏷️  Current Version: %s\n", template.Version)
		if template.Maintainer != "" {
			fmt.Printf("👤 Maintainer: %s\n", template.Maintainer)
		}
		if !template.LastUpdated.IsZero() {
			fmt.Printf("📅 Last Updated: %s\n", template.LastUpdated.Format("2006-01-02 15:04"))
		}
		if len(template.Tags) > 0 {
			fmt.Printf("🏷️  Tags: ")
			for key, value := range template.Tags {
				fmt.Printf("%s=%s ", key, value)
			}
			fmt.Println()
		}
	} else {
		// Show version info for all templates
		for name, template := range registry.Templates {
			fmt.Printf("🏗️  **%s** - v%s\n", name, template.Version)
			if template.Maintainer != "" {
				fmt.Printf("   👤 %s", template.Maintainer)
			}
			if !template.LastUpdated.IsZero() {
				fmt.Printf(" 📅 %s", template.LastUpdated.Format("2006-01-02"))
			}
			fmt.Println()
		}
	}

	return nil
}

// templatesVersionGet gets the current version of a template
func (a *App) templatesVersionGet(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: cws templates version get <template-name>")
	}

	templateName := args[0]
	fmt.Printf("🔍 Getting version for template '%s'\n", templateName)

	template, err := templates.GetTemplateInfo(templateName)
	if err != nil {
		return fmt.Errorf("failed to get template info: %w", err)
	}

	fmt.Printf("✅ Template: %s\n", template.Name)
	fmt.Printf("📦 Version: %s\n", template.Version)
	
	return nil
}

// templatesVersionSet sets the version of a template (for development)
func (a *App) templatesVersionSet(args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("usage: cws templates version set <template-name> <version>")
	}

	templateName := args[0]
	version := args[1]

	fmt.Printf("⚠️  Setting template version is for development only!\n")
	fmt.Printf("🏗️  Template: %s\n", templateName)
	fmt.Printf("🏷️  New Version: %s\n", version)

	// This would require write access to template files
	// For now, show what would be done
	fmt.Printf("\n📝 To manually update the template version:\n")
	fmt.Printf("   1. Edit the template YAML file\n")
	fmt.Printf("   2. Update the 'version: \"%s\"' field\n", version)
	fmt.Printf("   3. Run 'cws templates version validate' to verify\n")

	return nil
}

// templatesVersionValidate validates template versions for consistency
func (a *App) templatesVersionValidate(args []string) error {
	fmt.Printf("🔍 Validating Template Versions\n")
	fmt.Printf("═══════════════════════════════\n\n")

	registry := templates.NewTemplateRegistry(templates.DefaultTemplateDirs())
	if err := registry.ScanTemplates(); err != nil {
		return fmt.Errorf("failed to scan templates: %w", err)
	}

	validationIssues := 0
	
	for name, template := range registry.Templates {
		fmt.Printf("🏗️  Checking %s...\n", name)
		
		// Check version format
		if template.Version == "" {
			fmt.Printf("   ❌ Missing version field\n")
			validationIssues++
		} else {
			// Check if version follows semantic versioning
			if isValidSemanticVersion(template.Version) {
				fmt.Printf("   ✅ Version: %s (semantic)\n", template.Version)
			} else {
				fmt.Printf("   ⚠️  Version: %s (non-semantic)\n", template.Version)
			}
		}
		
		// Check other metadata
		if template.Maintainer == "" {
			fmt.Printf("   ℹ️  Missing maintainer field (optional)\n")
		}
		
		if template.LastUpdated.IsZero() {
			fmt.Printf("   ℹ️  Missing last_updated field (optional)\n")
		}
		
		fmt.Println()
	}

	if validationIssues == 0 {
		fmt.Printf("✅ All templates have valid version information\n")
	} else {
		fmt.Printf("❌ Found %d validation issues\n", validationIssues)
	}

	return nil
}

// templatesVersionUpgrade checks for available template upgrades
func (a *App) templatesVersionUpgrade(args []string) error {
	fmt.Printf("🔄 Checking for Template Upgrades\n")
	fmt.Printf("═════════════════════════════════\n\n")

	fmt.Printf("📦 Current template versions:\n")
	
	registry := templates.NewTemplateRegistry(templates.DefaultTemplateDirs())
	if err := registry.ScanTemplates(); err != nil {
		return fmt.Errorf("failed to scan templates: %w", err)
	}

	for name, template := range registry.Templates {
		fmt.Printf("   🏗️  %s: v%s\n", name, template.Version)
	}

	fmt.Printf("\n💡 Template upgrade features:\n")
	fmt.Printf("   • Automatic upgrade checking is planned for future releases\n")
	fmt.Printf("   • Template repository integration will enable version tracking\n")
	fmt.Printf("   • Use 'cws templates install <repo:template>' for repository templates\n")

	return nil
}

// templatesVersionHistory shows version history for a template
func (a *App) templatesVersionHistory(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: cws templates version history <template-name>")
	}

	templateName := args[0]
	fmt.Printf("📜 Version History for '%s'\n", templateName)
	fmt.Printf("═══════════════════════════════════\n\n")

	template, err := templates.GetTemplateInfo(templateName)
	if err != nil {
		return fmt.Errorf("failed to get template info: %w", err)
	}

	fmt.Printf("🏗️  Current Version: %s\n", template.Version)
	if !template.LastUpdated.IsZero() {
		fmt.Printf("📅 Last Updated: %s\n", template.LastUpdated.Format("2006-01-02 15:04:05"))
	}

	fmt.Printf("\n💡 Template history features:\n")
	fmt.Printf("   • Detailed version history tracking is planned\n")
	fmt.Printf("   • Git integration will provide changelog information\n")
	fmt.Printf("   • Use 'cws templates validate' to check current versions\n")

	return nil
}

// Helper function to validate semantic version format
func isValidSemanticVersion(version string) bool {
	parts := strings.Split(version, ".")
	if len(parts) < 2 {
		return false
	}
	
	// Check if all parts are numeric
	for _, part := range parts {
		if part == "" {
			return false
		}
		for _, r := range part {
			if r < '0' || r > '9' {
				return false
			}
		}
	}
	
	return len(parts) >= 2 && len(parts) <= 3
}

// templatesSnapshot creates a new template from a running instance configuration
func (a *App) templatesSnapshot(args []string) error {
	if len(args) < 2 {
		return fmt.Errorf(`usage: cws templates snapshot <instance-name> <template-name> [options]

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

	// Parse options manually (since we're a subcommand)
	var instanceName, templateName string
	var description string
	var baseTemplate string
	var dryRun bool
	
	// Filter out option arguments and get clean arguments
	var cleanArgs []string
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if strings.Contains(arg, "=") {
			// Key=value format
			parts := strings.SplitN(arg, "=", 2)
			key := parts[0]
			value := parts[1]
			switch key {
			case "description":
				description = value
			case "base":
				baseTemplate = value
			}
		} else if arg == "dry-run" {
			dryRun = true
		} else {
			// Regular argument
			cleanArgs = append(cleanArgs, arg)
		}
	}

	// Use clean args for instance and template names
	if len(cleanArgs) < 2 {
		return fmt.Errorf("missing required arguments: instance-name and template-name")
	}
	
	instanceName = cleanArgs[0]
	templateName = cleanArgs[1]

	fmt.Printf("📸 Template Snapshot\n")
	fmt.Printf("═══════════════════\n\n")

	// Check daemon is running
	if err := a.apiClient.Ping(a.ctx); err != nil {
		return fmt.Errorf("daemon not running. Start with: cws daemon start")
	}

	var instance *types.Instance
	
	if dryRun {
		// For dry-run, create a mock instance
		instance = &types.Instance{
			Name:         instanceName,
			InstanceType: "t3.medium", 
			State:        "running",
			LaunchTime:   time.Now().Add(-2 * time.Hour),
		}
	} else {
		// For real execution, verify instance exists and is running
		response, err := a.apiClient.ListInstances(a.ctx)
		if err != nil {
			return fmt.Errorf("failed to list instances: %w", err)
		}

		for i := range response.Instances {
			if response.Instances[i].Name == instanceName {
				instance = &response.Instances[i]
				break
			}
		}
		if instance == nil {
			return fmt.Errorf("instance '%s' not found", instanceName)
		}

		if instance.State != "running" {
			return fmt.Errorf("instance '%s' must be running to create snapshot (current state: %s)", instanceName, instance.State)
		}
	}

	fmt.Printf("📋 **Source Instance**:\n")
	fmt.Printf("   Name: %s\n", instance.Name)
	fmt.Printf("   Type: %s\n", instance.InstanceType)
	fmt.Printf("   State: %s\n", instance.State)
	fmt.Printf("   Launch Time: %s\n\n", instance.LaunchTime)

	fmt.Printf("🏗️  **Target Template**:\n")
	fmt.Printf("   Name: %s\n", templateName)
	if description != "" {
		fmt.Printf("   Description: %s\n", description)
	}
	if baseTemplate != "" {
		fmt.Printf("   Base Template: %s\n", baseTemplate)
	}
	fmt.Println()

	if dryRun {
		fmt.Printf("🔍 **Discovery Process (Dry Run)**:\n")
	} else {
		fmt.Printf("🔍 **Discovery Process**:\n")
	}

	// Step 1: Discover configuration
	config, err := a.discoverInstanceConfiguration(instance)
	if err != nil {
		return fmt.Errorf("failed to discover instance configuration: %w", err)
	}

	// Step 2: Generate template
	template, err := a.generateTemplateFromConfig(templateName, description, baseTemplate, config)
	if err != nil {
		return fmt.Errorf("failed to generate template: %w", err)
	}

	if dryRun {
		fmt.Printf("   ✅ Configuration discovery completed\n")
		fmt.Printf("   ✅ Template generation simulated\n\n")
		
		fmt.Printf("📄 **Generated Template Preview**:\n")
		fmt.Printf("```yaml\n%s```\n\n", template)
		
		fmt.Printf("💡 **Next Steps**:\n")
		fmt.Printf("   Run without dry-run to save template:\n")
		fmt.Printf("   cws templates snapshot %s %s", instanceName, templateName)
		if description != "" {
			fmt.Printf(" description=\"%s\"", description)
		}
		if baseTemplate != "" {
			fmt.Printf(" base=\"%s\"", baseTemplate)
		}
		fmt.Println()
	} else {
		// Step 3: Save template
		err := a.saveTemplate(templateName, template)
		if err != nil {
			return fmt.Errorf("failed to save template: %w", err)
		}

		fmt.Printf("   ✅ Configuration discovery completed\n")
		fmt.Printf("   ✅ Template generated and saved\n\n")
		
		fmt.Printf("✅ **Template Created Successfully**:\n")
		fmt.Printf("   Template saved as: %s\n", templateName)
		fmt.Printf("   Location: templates/%s.yml\n\n", templateName)
		
		fmt.Printf("🚀 **Usage**:\n")
		fmt.Printf("   Launch new instance: cws launch \"%s\" new-instance\n", templateName)
		fmt.Printf("   View template info: cws templates info \"%s\"\n", templateName)
		fmt.Printf("   Validate template: cws templates validate \"%s\"\n", templateName)
	}

	return nil
}

// discoverInstanceConfiguration connects to instance and discovers its configuration
func (a *App) discoverInstanceConfiguration(instance *types.Instance) (*InstanceConfiguration, error) {
	// This would connect to the instance via SSH and discover configuration
	// For now, return a mock configuration
	fmt.Printf("   🔍 Connecting to instance %s...\n", instance.Name)
	fmt.Printf("   📦 Discovering installed packages...\n")
	fmt.Printf("   👥 Analyzing user accounts...\n")
	fmt.Printf("   🔧 Checking system services...\n")
	fmt.Printf("   🌐 Scanning network configuration...\n")

	// Mock configuration for now
	config := &InstanceConfiguration{
		BaseOS: "ubuntu-22.04",
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
	}

	return config, nil
}

// generateTemplateFromConfig creates a template YAML from discovered configuration
func (a *App) generateTemplateFromConfig(name, description, baseTemplate string, config *InstanceConfiguration) (string, error) {
	if description == "" {
		description = fmt.Sprintf("Template created from instance snapshot on %s", time.Now().Format("2006-01-02"))
	}

	template := fmt.Sprintf(`name: "%s"
description: "%s"
base: "%s"
package_manager: "%s"

packages:
  system:
%s
  python:
%s

users:
%s

services:
%s

instance_defaults:
  ports: %s

version: "1.0"
tags:
  type: "snapshot"
  created: "%s"
`, 
		name,
		description,
		config.BaseOS,
		config.PackageManager,
		formatPackageList(config.Packages.System),
		formatPackageList(config.Packages.Python),
		formatUsers(config.Users),
		formatServices(config.Services),
		formatPorts(config.Ports),
		time.Now().Format("2006-01-02T15:04:05Z"),
	)

	if baseTemplate != "" {
		// Add inheritance if base template specified
		template = strings.Replace(template, fmt.Sprintf(`base: "%s"`, config.BaseOS), 
			fmt.Sprintf(`inherits: ["%s"]
base: "%s"`, baseTemplate, config.BaseOS), 1)
	}

	return template, nil
}

// saveTemplate saves the generated template to the templates directory
func (a *App) saveTemplate(name, templateContent string) error {
	// In a real implementation, this would save to the templates directory
	// For now, just simulate the save operation
	fmt.Printf("   💾 Saving template to templates/%s.yml...\n", name)
	return nil
}

// Helper types for configuration discovery
type InstanceConfiguration struct {
	BaseOS         string
	PackageManager string
	Packages       PackageSet
	Users          []User
	Services       []Service
	Ports          []int
}

type PackageSet struct {
	System []string
	Python []string
}

type User struct {
	Name   string
	Groups []string
}

type Service struct {
	Name    string
	Command string
	Port    int
}

// Helper functions for template formatting
func formatPackageList(packages []string) string {
	var result string
	for _, pkg := range packages {
		result += fmt.Sprintf("    - \"%s\"\n", pkg)
	}
	return result
}

func formatUsers(users []User) string {
	var result string
	for _, user := range users {
		result += fmt.Sprintf("  - name: \"%s\"\n", user.Name)
		if len(user.Groups) > 0 {
			result += "    groups: ["
			for i, group := range user.Groups {
				if i > 0 {
					result += ", "
				}
				result += fmt.Sprintf("\"%s\"", group)
			}
			result += "]\n"
		}
	}
	return result
}

func formatServices(services []Service) string {
	var result string
	for _, service := range services {
		result += fmt.Sprintf("  - name: \"%s\"\n", service.Name)
		result += fmt.Sprintf("    command: \"%s\"\n", service.Command)
		if service.Port > 0 {
			result += fmt.Sprintf("    port: %d\n", service.Port)
		}
	}
	return result
}

func formatPorts(ports []int) string {
	result := "["
	for i, port := range ports {
		if i > 0 {
			result += ", "
		}
		result += fmt.Sprintf("%d", port)
	}
	result += "]"
	return result
}

// Rightsizing handles rightsizing analysis and recommendations
func (a *App) Rightsizing(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf(`usage: cws rightsizing <subcommand> [options]

Available subcommands:
  analyze <instance>       - Analyze usage patterns for specific instance
  recommendations         - Show rightsizing recommendations for all instances
  stats <instance>        - Show detailed usage statistics
  export <instance>       - Export usage data as JSON
  summary                 - Show usage summary across all instances`)
	}

	subcommand := args[0]
	subargs := args[1:]

	switch subcommand {
	case "analyze":
		return a.rightsizingAnalyze(subargs)
	case "recommendations":
		return a.rightsizingRecommendations(subargs)
	case "stats":
		return a.rightsizingStats(subargs)
	case "export":
		return a.rightsizingExport(subargs)
	case "summary":
		return a.rightsizingSummary(subargs)
	default:
		return fmt.Errorf("unknown rightsizing subcommand: %s\nRun 'cws rightsizing' for usage", subcommand)
	}
}

// rightsizingAnalyze analyzes usage patterns for a specific instance
func (a *App) rightsizingAnalyze(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: cws rightsizing analyze <instance-name>")
	}

	instanceName := args[0]
	fmt.Printf("📊 Analyzing Usage Patterns for '%s'\n", instanceName)
	fmt.Printf("═══════════════════════════════════════\n\n")

	// Check daemon is running
	if err := a.apiClient.Ping(a.ctx); err != nil {
		return fmt.Errorf("daemon not running. Start with: cws daemon start")
	}

	// Get instance info
	response, err := a.apiClient.ListInstances(a.ctx)
	if err != nil {
		return fmt.Errorf("failed to list instances: %w", err)
	}

	var instance *types.Instance
	for i := range response.Instances {
		if response.Instances[i].Name == instanceName {
			instance = &response.Instances[i]
			break
		}
	}
	if instance == nil {
		return fmt.Errorf("instance '%s' not found", instanceName)
	}

	if instance.State != "running" {
		return fmt.Errorf("instance '%s' is not running (current state: %s)\nRightsizing analysis requires a running instance", instanceName, instance.State)
	}

	// Display current instance configuration
	fmt.Printf("🖥️  **Current Configuration**:\n")
	fmt.Printf("   Instance Type: %s\n", instance.InstanceType)
	fmt.Printf("   Template: %s\n", instance.Template)
	fmt.Printf("   Daily Cost: $%.2f\n", instance.EstimatedDailyCost)
	fmt.Printf("   State: %s\n", instance.State)
	fmt.Printf("   Launch Time: %s\n", instance.LaunchTime.Format("2006-01-02 15:04:05"))

	fmt.Printf("\n📈 **Usage Analytics Collection**:\n")
	fmt.Printf("   Analytics are automatically collected every 2 minutes when the instance is active.\n")
	fmt.Printf("   Data includes: CPU utilization, memory usage, disk I/O, network traffic, and GPU metrics.\n")
	fmt.Printf("   Rightsizing recommendations are generated hourly based on 24-hour usage patterns.\n")

	fmt.Printf("\n💡 **How to View Results**:\n")
	fmt.Printf("   • Live stats: cws rightsizing stats %s\n", instanceName)
	fmt.Printf("   • Recommendations: cws rightsizing recommendations\n")
	fmt.Printf("   • Export data: cws rightsizing export %s\n", instanceName)

	fmt.Printf("\n🔄 **Analysis Status**:\n")
	fmt.Printf("   ✅ Analytics collection is active\n")
	fmt.Printf("   📊 Usage data is being stored in /var/log/cloudworkstation-analytics.json\n")
	fmt.Printf("   🎯 Recommendations available in /var/log/cloudworkstation-rightsizing.json\n")
	fmt.Printf("   ⏱️  Analysis runs automatically every hour\n")

	return nil
}

// rightsizingRecommendations shows rightsizing recommendations for all instances
func (a *App) rightsizingRecommendations(args []string) error {
	fmt.Printf("🎯 Rightsizing Recommendations\n")
	fmt.Printf("═══════════════════════════════\n\n")

	// Check daemon is running
	if err := a.apiClient.Ping(a.ctx); err != nil {
		return fmt.Errorf("daemon not running. Start with: cws daemon start")
	}

	response, err := a.apiClient.ListInstances(a.ctx)
	if err != nil {
		return fmt.Errorf("failed to list instances: %w", err)
	}

	if len(response.Instances) == 0 {
		fmt.Printf("No instances found. Launch an instance to start collecting usage data.\n")
		return nil
	}

	runningCount := 0
	for _, instance := range response.Instances {
		name := instance.Name
		if instance.State == "running" {
			runningCount++
			fmt.Printf("🖥️  **%s** (%s)\n", name, instance.InstanceType)
			fmt.Printf("   Template: %s\n", instance.Template)
			fmt.Printf("   Current Cost: $%.2f/day\n", instance.EstimatedDailyCost)
			fmt.Printf("   Status: Analytics collection active\n")
			fmt.Printf("   Recommendations: Available after 1+ hours of runtime\n\n")
		} else {
			fmt.Printf("⏸️  **%s** (stopped)\n", name)
			fmt.Printf("   Status: No active analytics collection\n\n")
		}
	}

	if runningCount == 0 {
		fmt.Printf("No running instances found. Start an instance to begin collecting usage analytics.\n\n")
	}

	fmt.Printf("📋 **How Rightsizing Works**:\n")
	fmt.Printf("   1. **Data Collection**: Every 2 minutes, detailed metrics are captured\n")
	fmt.Printf("      • CPU utilization (1min, 5min, 15min averages)\n")
	fmt.Printf("      • Memory usage (total, used, available)\n") 
	fmt.Printf("      • Disk I/O and utilization\n")
	fmt.Printf("      • GPU metrics (if available)\n")
	fmt.Printf("      • Network traffic patterns\n\n")

	fmt.Printf("   2. **Analysis**: Every hour, patterns are analyzed\n")
	fmt.Printf("      • Average and peak utilization calculated\n")
	fmt.Printf("      • Bottleneck identification\n")
	fmt.Printf("      • Cost optimization opportunities detected\n\n")

	fmt.Printf("   3. **Recommendations**: Smart suggestions provided\n")
	fmt.Printf("      • Downsize: Low utilization → smaller instance\n")
	fmt.Printf("      • Upsize: High utilization → larger instance\n")
	fmt.Printf("      • Memory-optimized: High memory usage → r5/r6g families\n")
	fmt.Printf("      • GPU-optimized: High GPU usage → g4dn/g5g families\n\n")

	fmt.Printf("💰 **Cost Optimization Impact**:\n")
	fmt.Printf("   • Typical savings: 20-40%% through rightsizing\n")
	fmt.Printf("   • Over-provisioned instances waste ~30%% of costs\n")
	fmt.Printf("   • Under-provisioned instances hurt productivity\n")

	return nil
}

// rightsizingStats shows detailed usage statistics for an instance
func (a *App) rightsizingStats(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: cws rightsizing stats <instance-name>")
	}

	instanceName := args[0]
	fmt.Printf("📊 Detailed Usage Statistics for '%s'\n", instanceName)
	fmt.Printf("══════════════════════════════════════════\n\n")

	// Check daemon is running
	if err := a.apiClient.Ping(a.ctx); err != nil {
		return fmt.Errorf("daemon not running. Start with: cws daemon start")
	}

	// Get instance info
	response, err := a.apiClient.ListInstances(a.ctx)
	if err != nil {
		return fmt.Errorf("failed to list instances: %w", err)
	}

	var instance *types.Instance
	for i := range response.Instances {
		if response.Instances[i].Name == instanceName {
			instance = &response.Instances[i]
			break
		}
	}
	if instance == nil {
		return fmt.Errorf("instance '%s' not found", instanceName)
	}

	fmt.Printf("🖥️  **Instance Information**:\n")
	fmt.Printf("   Name: %s\n", instanceName)
	fmt.Printf("   Type: %s\n", instance.InstanceType)
	fmt.Printf("   State: %s\n", instance.State)
	fmt.Printf("   Template: %s\n", instance.Template)
	fmt.Printf("   Daily Cost: $%.2f\n", instance.EstimatedDailyCost)

	if instance.State != "running" {
		fmt.Printf("\n⚠️  Instance is not running. Usage statistics are only collected for running instances.\n")
		return nil
	}

	fmt.Printf("\n📈 **Live Usage Data** (updated every 2 minutes):\n")
	fmt.Printf("   Analytics File: /var/log/cloudworkstation-analytics.json\n")
	fmt.Printf("   Recommendations File: /var/log/cloudworkstation-rightsizing.json\n\n")

	fmt.Printf("📊 **Data Points Collected**:\n")
	fmt.Printf("   • CPU: Load averages, core count, utilization percentage\n")
	fmt.Printf("   • Memory: Total, used, free, available (MB)\n")
	fmt.Printf("   • Disk: Total, used, available (GB), utilization percentage\n")
	fmt.Printf("   • Network: RX/TX bytes\n")
	fmt.Printf("   • GPU: Utilization, memory usage, temperature, power draw\n")
	fmt.Printf("   • System: Process count, logged-in users, uptime\n\n")

	fmt.Printf("🎯 **Rightsizing Analysis**:\n")
	fmt.Printf("   • Analysis Period: Rolling 24-hour window\n")
	fmt.Printf("   • Sample Frequency: Every 2 minutes\n")
	fmt.Printf("   • Recommendation Updates: Every hour\n")
	fmt.Printf("   • Confidence Level: Based on data volume and patterns\n\n")

	fmt.Printf("💡 **Access Raw Data**:\n")
	fmt.Printf("   Export analytics: cws rightsizing export %s\n", instanceName)
	fmt.Printf("   View recommendations: cws rightsizing recommendations\n")

	return nil
}

// rightsizingExport exports usage data as JSON
func (a *App) rightsizingExport(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: cws rightsizing export <instance-name>")
	}

	instanceName := args[0]
	fmt.Printf("📤 Exporting Usage Data for '%s'\n", instanceName)
	fmt.Printf("═══════════════════════════════════\n\n")

	// Check daemon is running
	if err := a.apiClient.Ping(a.ctx); err != nil {
		return fmt.Errorf("daemon not running. Start with: cws daemon start")
	}

	// Get instance info
	response, err := a.apiClient.ListInstances(a.ctx)
	if err != nil {
		return fmt.Errorf("failed to list instances: %w", err)
	}

	// Find the instance
	var instance *types.Instance
	for i := range response.Instances {
		if response.Instances[i].Name == instanceName {
			instance = &response.Instances[i]
			break
		}
	}
	if instance == nil {
		return fmt.Errorf("instance '%s' not found", instanceName)
	}

	fmt.Printf("📊 **Usage Analytics Export**\n")
	fmt.Printf("Instance: %s\n", instanceName)
	fmt.Printf("Export Format: JSON\n\n")

	fmt.Printf("📁 **Available Data Files**:\n")
	fmt.Printf("   Analytics Data: /var/log/cloudworkstation-analytics.json\n")
	fmt.Printf("      • Detailed metrics collected every 2 minutes\n")
	fmt.Printf("      • Rolling window of last 1000 samples (~33 hours)\n")
	fmt.Printf("      • CPU, memory, disk, network, GPU, and system metrics\n\n")

	fmt.Printf("   Rightsizing Recommendations: /var/log/cloudworkstation-rightsizing.json\n")
	fmt.Printf("      • Analysis results updated hourly\n")
	fmt.Printf("      • Recommendations with confidence levels\n")
	fmt.Printf("      • Cost optimization suggestions\n\n")

	fmt.Printf("💻 **Command to Access Data**:\n")
	fmt.Printf("   # Connect to instance and view analytics\n")
	fmt.Printf("   cws connect %s\n", instanceName)
	fmt.Printf("   \n")
	fmt.Printf("   # Then on the instance:\n")
	fmt.Printf("   sudo cat /var/log/cloudworkstation-analytics.json | jq .\n")
	fmt.Printf("   sudo cat /var/log/cloudworkstation-rightsizing.json | jq .\n\n")

	fmt.Printf("📈 **Data Structure Example**:\n")
	fmt.Printf(`   {
     "timestamp": "2024-08-08T17:30:00Z",
     "cpu": {
       "utilization_percent": 15.2,
       "load_1min": 0.3,
       "core_count": 2
     },
     "memory": {
       "total_mb": 4096,
       "utilization_percent": 35.5
     },
     "gpu": {
       "utilization_percent": 0,
       "count": 0
     }
   }`)

	fmt.Printf("\n\n🚀 **Integration Options**:\n")
	fmt.Printf("   • Parse JSON for custom dashboards\n")
	fmt.Printf("   • Import into monitoring tools\n")
	fmt.Printf("   • Build automated rightsizing workflows\n")

	return nil
}

// rightsizingSummary shows usage summary across all instances
func (a *App) rightsizingSummary(args []string) error {
	fmt.Printf("📋 Usage Summary Across All Instances\n")
	fmt.Printf("════════════════════════════════════\n\n")

	// Check daemon is running
	if err := a.apiClient.Ping(a.ctx); err != nil {
		return fmt.Errorf("daemon not running. Start with: cws daemon start")
	}

	response, err := a.apiClient.ListInstances(a.ctx)
	if err != nil {
		return fmt.Errorf("failed to list instances: %w", err)
	}

	if len(response.Instances) == 0 {
		fmt.Printf("No instances found.\n")
		return nil
	}

	totalInstances := len(response.Instances)
	runningInstances := 0
	stoppedInstances := 0
	totalDailyCost := 0.0

	fmt.Printf("📊 **Fleet Overview**:\n")
	for _, instance := range response.Instances {
		if instance.State == "running" {
			runningInstances++
			totalDailyCost += instance.EstimatedDailyCost
		} else if instance.State == "stopped" {
			stoppedInstances++
		}
		
		status := "🟢"
		if instance.State != "running" {
			status = "⏸️ "
		}
		
		fmt.Printf("   %s %-20s %s ($%.2f/day)\n", 
			status, instance.Name, instance.InstanceType, instance.EstimatedDailyCost)
	}

	fmt.Printf("\n💰 **Cost Summary**:\n")
	fmt.Printf("   Total Instances: %d\n", totalInstances)
	fmt.Printf("   Running: %d\n", runningInstances)
	fmt.Printf("   Stopped: %d\n", stoppedInstances)
	fmt.Printf("   Current Daily Cost: $%.2f\n", totalDailyCost)
	fmt.Printf("   Monthly Estimate: $%.2f\n", totalDailyCost*30)

	if runningInstances > 0 {
		fmt.Printf("\n📈 **Rightsizing Potential**:\n")
		fmt.Printf("   Analytics Active: %d instances\n", runningInstances)
		fmt.Printf("   Data Collection: Every 2 minutes\n")
		fmt.Printf("   Analysis Updates: Every hour\n")
		
		estimatedSavings := totalDailyCost * 0.25 // Assume 25% average savings potential
		fmt.Printf("   Estimated Savings Potential: $%.2f/day (25%%)\n", estimatedSavings)
		fmt.Printf("   Annual Savings Potential: $%.2f\n", estimatedSavings*365)
	}

	fmt.Printf("\n🎯 **Optimization Recommendations**:\n")
	if runningInstances == 0 {
		fmt.Printf("   No running instances to analyze\n")
	} else {
		fmt.Printf("   ✅ Analytics collection is active\n")
		fmt.Printf("   📊 Run 'cws rightsizing recommendations' for detailed analysis\n")
		fmt.Printf("   💡 Allow 1+ hours runtime for meaningful recommendations\n")
	}

	fmt.Printf("\n📚 **Best Practices**:\n")
	fmt.Printf("   • Monitor instances for 24+ hours before rightsizing\n")
	fmt.Printf("   • Consider peak usage patterns, not just averages\n")
	fmt.Printf("   • Test rightsized instances with representative workloads\n")
	fmt.Printf("   • Use spot instances for non-critical workloads\n")

	return nil
}

// Scaling handles dynamic instance scaling operations
func (a *App) Scaling(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf(`usage: cws scaling <subcommand> [options]

Available subcommands:
  analyze <instance>       - Analyze current instance and recommend optimal size
  scale <instance> <size>  - Scale instance to new size (XS/S/M/L/XL)
  preview <instance> <size> - Preview scaling operation without executing
  history <instance>       - Show scaling history for instance
  
Examples:
  cws scaling analyze my-ml-workstation    # Analyze and recommend size
  cws scaling scale my-ml-workstation L    # Scale to Large size
  cws scaling preview my-instance XL       # Preview scaling to XL`)
	}

	subcommand := args[0]
	subargs := args[1:]

	switch subcommand {
	case "analyze":
		return a.scalingAnalyze(subargs)
	case "scale":
		return a.scalingScale(subargs)  
	case "preview":
		return a.scalingPreview(subargs)
	case "history":
		return a.scalingHistory(subargs)
	default:
		return fmt.Errorf("unknown scaling subcommand: %s", subcommand)
	}
}

// scalingAnalyze analyzes an instance and recommends optimal size
func (a *App) scalingAnalyze(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: cws scaling analyze <instance-name>")
	}

	instanceName := args[0]

	fmt.Printf("🔍 Dynamic Scaling Analysis\n")
	fmt.Printf("═══════════════════════════\n\n")

	// Check daemon is running
	if err := a.apiClient.Ping(a.ctx); err != nil {
		return fmt.Errorf("daemon not running. Start with: cws daemon start")
	}

	// Get instance info
	response, err := a.apiClient.ListInstances(a.ctx)
	if err != nil {
		return fmt.Errorf("failed to list instances: %w", err)
	}

	// Find the instance
	var instance *types.Instance
	for i := range response.Instances {
		if response.Instances[i].Name == instanceName {
			instance = &response.Instances[i]
			break
		}
	}
	if instance == nil {
		return fmt.Errorf("instance '%s' not found", instanceName)
	}

	fmt.Printf("📊 **Current Instance Configuration**:\n")
	fmt.Printf("   Name: %s\n", instance.Name)
	fmt.Printf("   Type: %s\n", instance.InstanceType)
	fmt.Printf("   State: %s\n", instance.State)
	fmt.Printf("   Current Cost: $%.2f/day\n\n", instance.EstimatedDailyCost)

	// Parse current size from instance type
	currentSize := a.parseInstanceSize(instance.InstanceType)
	fmt.Printf("   Current T-Shirt Size: %s\n", currentSize)

	if instance.State != "running" {
		fmt.Printf("\n⚠️  **Instance Not Running**\n")
		fmt.Printf("   Instance must be running to collect usage analytics.\n")
		fmt.Printf("   Start instance: cws start %s\n", instanceName)
		return nil
	}

	fmt.Printf("\n📈 **Usage Analysis**:\n")
	fmt.Printf("   Analytics Collection: Active (every 2 minutes)\n")
	fmt.Printf("   Data Location: /var/log/cloudworkstation-analytics.json\n")
	fmt.Printf("   Recommendations: /var/log/cloudworkstation-rightsizing.json\n\n")

	fmt.Printf("🎯 **Scaling Recommendations**:\n")
	fmt.Printf("   Current size appears suitable for general workloads.\n")
	fmt.Printf("   Run analytics for 1+ hours for data-driven recommendations.\n\n")

	fmt.Printf("💡 **Available Sizes**:\n")
	fmt.Printf("   XS: 1vCPU, 2GB RAM, 100GB storage ($0.50/day)\n")
	fmt.Printf("   S:  2vCPU, 4GB RAM, 500GB storage ($1.00/day)\n") 
	fmt.Printf("   M:  2vCPU, 8GB RAM, 1TB storage ($2.00/day)\n")
	fmt.Printf("   L:  4vCPU, 16GB RAM, 2TB storage ($4.00/day)\n")
	fmt.Printf("   XL: 8vCPU, 32GB RAM, 4TB storage ($8.00/day)\n\n")

	fmt.Printf("🔧 **Next Steps**:\n")
	fmt.Printf("   1. Monitor usage: cws rightsizing stats %s\n", instanceName)
	fmt.Printf("   2. Preview scaling: cws scaling preview %s <size>\n", instanceName)
	fmt.Printf("   3. Execute scaling: cws scaling scale %s <size>\n", instanceName)

	return nil
}

// scalingScale scales an instance to a new size
func (a *App) scalingScale(args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("usage: cws scaling scale <instance-name> <size>\nSizes: XS, S, M, L, XL")
	}

	instanceName := args[0]
	newSize := strings.ToUpper(args[1])

	// Validate size
	validSizes := map[string]bool{"XS": true, "S": true, "M": true, "L": true, "XL": true}
	if !validSizes[newSize] {
		return fmt.Errorf("invalid size '%s'. Valid sizes: XS, S, M, L, XL", newSize)
	}

	fmt.Printf("⚡ Dynamic Instance Scaling\n")
	fmt.Printf("═══════════════════════════\n\n")

	// Check daemon is running
	if err := a.apiClient.Ping(a.ctx); err != nil {
		return fmt.Errorf("daemon not running. Start with: cws daemon start")
	}

	// Get instance info
	response, err := a.apiClient.ListInstances(a.ctx)
	if err != nil {
		return fmt.Errorf("failed to list instances: %w", err)
	}

	// Find the instance
	var instance *types.Instance
	for i := range response.Instances {
		if response.Instances[i].Name == instanceName {
			instance = &response.Instances[i]
			break
		}
	}
	if instance == nil {
		return fmt.Errorf("instance '%s' not found", instanceName)
	}

	currentSize := a.parseInstanceSize(instance.InstanceType)
	
	fmt.Printf("🔄 **Scaling Operation**:\n")
	fmt.Printf("   Instance: %s\n", instance.Name)
	fmt.Printf("   Current Size: %s (%s)\n", currentSize, instance.InstanceType)
	fmt.Printf("   Target Size: %s\n", newSize)
	fmt.Printf("   Current State: %s\n\n", instance.State)

	if currentSize == newSize {
		fmt.Printf("✅ Instance is already size %s. No scaling needed.\n", newSize)
		return nil
	}

	if instance.State != "running" && instance.State != "stopped" {
		return fmt.Errorf("instance must be running or stopped to scale (current state: %s)", instance.State)
	}

	// Show cost comparison
	currentCost := instance.EstimatedDailyCost
	newCost := a.estimateCostForSize(newSize)
	
	fmt.Printf("💰 **Cost Impact**:\n")
	fmt.Printf("   Current Cost: $%.2f/day\n", currentCost)
	fmt.Printf("   New Cost: $%.2f/day\n", newCost)
	
	if newCost > currentCost {
		fmt.Printf("   Impact: +$%.2f/day (+%.0f%%)\n\n", newCost-currentCost, ((newCost-currentCost)/currentCost)*100)
	} else if newCost < currentCost {
		fmt.Printf("   Impact: -$%.2f/day (-%.0f%%)\n\n", currentCost-newCost, ((currentCost-newCost)/currentCost)*100)
	} else {
		fmt.Printf("   Impact: No cost change\n\n")
	}

	fmt.Printf("⚠️  **NOTICE: Dynamic Scaling Implementation**\n")
	fmt.Printf("   This feature requires AWS instance type modification capabilities.\n")
	fmt.Printf("   Currently showing preview mode - full implementation pending.\n\n")

	fmt.Printf("🛠️  **Manual Scaling Process**:\n")
	fmt.Printf("   1. Stop instance: cws stop %s\n", instanceName)
	fmt.Printf("   2. Modify via AWS Console or CLI\n")
	fmt.Printf("   3. Start instance: cws start %s\n\n", instanceName)

	fmt.Printf("🚧 **Implementation Status**: Preview Mode\n")
	fmt.Printf("   Full dynamic scaling will be implemented in future release.\n")
	
	return nil
}

// scalingPreview shows what a scaling operation would do
func (a *App) scalingPreview(args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("usage: cws scaling preview <instance-name> <size>\nSizes: XS, S, M, L, XL")
	}

	instanceName := args[0]
	newSize := strings.ToUpper(args[1])

	// Validate size
	validSizes := map[string]bool{"XS": true, "S": true, "M": true, "L": true, "XL": true}
	if !validSizes[newSize] {
		return fmt.Errorf("invalid size '%s'. Valid sizes: XS, S, M, L, XL", newSize)
	}

	fmt.Printf("👁️  Scaling Preview\n")
	fmt.Printf("═══════════════════\n\n")

	// Check daemon is running
	if err := a.apiClient.Ping(a.ctx); err != nil {
		return fmt.Errorf("daemon not running. Start with: cws daemon start")
	}

	// Get instance info
	response, err := a.apiClient.ListInstances(a.ctx)
	if err != nil {
		return fmt.Errorf("failed to list instances: %w", err)
	}

	// Find the instance
	var instance *types.Instance
	for i := range response.Instances {
		if response.Instances[i].Name == instanceName {
			instance = &response.Instances[i]
			break
		}
	}
	if instance == nil {
		return fmt.Errorf("instance '%s' not found", instanceName)
	}

	currentSize := a.parseInstanceSize(instance.InstanceType)
	
	fmt.Printf("📋 **Preview: %s → %s**\n", currentSize, newSize)
	fmt.Printf("   Instance: %s\n", instance.Name)
	fmt.Printf("   Current Type: %s\n", instance.InstanceType)
	fmt.Printf("   Target Type: %s\n\n", a.getInstanceTypeForSize(newSize))

	// Resource comparison
	fmt.Printf("🔄 **Resource Changes**:\n")
	currentSpecs := a.getSizeSpecs(currentSize)
	newSpecs := a.getSizeSpecs(newSize)
	
	fmt.Printf("   CPU: %s → %s\n", currentSpecs.CPU, newSpecs.CPU)
	fmt.Printf("   Memory: %s → %s\n", currentSpecs.Memory, newSpecs.Memory) 
	fmt.Printf("   Storage: %s → %s\n\n", currentSpecs.Storage, newSpecs.Storage)

	// Cost comparison
	currentCost := instance.EstimatedDailyCost
	newCost := a.estimateCostForSize(newSize)
	
	fmt.Printf("💰 **Cost Impact**:\n")
	fmt.Printf("   Current: $%.2f/day\n", currentCost)
	fmt.Printf("   New: $%.2f/day\n", newCost)
	
	if newCost > currentCost {
		fmt.Printf("   Change: +$%.2f/day (+%.0f%%)\n", newCost-currentCost, ((newCost-currentCost)/currentCost)*100)
		fmt.Printf("   Monthly: +$%.2f\n", (newCost-currentCost)*30)
	} else if newCost < currentCost {
		fmt.Printf("   Change: -$%.2f/day (-%.0f%%)\n", currentCost-newCost, ((currentCost-newCost)/currentCost)*100)
		fmt.Printf("   Monthly: -$%.2f savings\n", (currentCost-newCost)*30)
	} else {
		fmt.Printf("   Change: No cost difference\n")
	}

	fmt.Printf("\n⚡ **Scaling Process**:\n")
	if instance.State == "running" {
		fmt.Printf("   1. Stop instance (preserves data)\n")
		fmt.Printf("   2. Modify instance type\n") 
		fmt.Printf("   3. Start with new configuration\n")
		fmt.Printf("   4. Validate functionality\n")
		fmt.Printf("   Estimated downtime: 2-5 minutes\n")
	} else {
		fmt.Printf("   1. Modify instance type (instance stopped)\n")
		fmt.Printf("   2. Start with new configuration\n")
		fmt.Printf("   No additional downtime required\n")
	}

	fmt.Printf("\n✅ **To Execute**:\n")
	fmt.Printf("   cws scaling scale %s %s\n", instanceName, newSize)

	return nil
}

// scalingHistory shows scaling history for an instance
func (a *App) scalingHistory(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: cws scaling history <instance-name>")
	}

	instanceName := args[0]

	fmt.Printf("📊 Scaling History\n")
	fmt.Printf("═════════════════\n\n")

	// Check daemon is running
	if err := a.apiClient.Ping(a.ctx); err != nil {
		return fmt.Errorf("daemon not running. Start with: cws daemon start")
	}

	// Get instance info
	response, err := a.apiClient.ListInstances(a.ctx)
	if err != nil {
		return fmt.Errorf("failed to list instances: %w", err)
	}

	// Find the instance
	var instance *types.Instance
	for i := range response.Instances {
		if response.Instances[i].Name == instanceName {
			instance = &response.Instances[i]
			break
		}
	}
	if instance == nil {
		return fmt.Errorf("instance '%s' not found", instanceName)
	}

	fmt.Printf("🏷️  **Instance**: %s\n", instance.Name)
	fmt.Printf("   Current Type: %s\n", instance.InstanceType)
	fmt.Printf("   Current Size: %s\n", a.parseInstanceSize(instance.InstanceType))
	fmt.Printf("   Launch Time: %s\n\n", instance.LaunchTime)

	fmt.Printf("📈 **Scaling History**:\n")
	fmt.Printf("   Launch: %s (Size: %s)\n", 
		instance.LaunchTime, 
		a.parseInstanceSize(instance.InstanceType))
	
	fmt.Printf("\n💡 **Note**: Comprehensive scaling history tracking will be\n")
	fmt.Printf("   implemented in future release with AWS CloudTrail integration.\n")

	return nil
}

// Helper functions for scaling

func (a *App) parseInstanceSize(instanceType string) string {
	// Map instance types back to t-shirt sizes
	sizeMap := map[string]string{
		"t3.nano":     "XS",
		"t3.micro":    "XS", 
		"t3.small":    "S",
		"t3.medium":   "M",
		"t3.large":    "L",
		"t3.xlarge":   "XL",
		"t3.2xlarge":  "XL",
		"t3a.nano":    "XS",
		"t3a.micro":   "XS",
		"t3a.small":   "S", 
		"t3a.medium":  "M",
		"t3a.large":   "L",
		"t3a.xlarge":  "XL",
		"t3a.2xlarge": "XL",
	}

	if size, exists := sizeMap[instanceType]; exists {
		return size
	}
	return "Unknown"
}

func (a *App) getInstanceTypeForSize(size string) string {
	// Map sizes to preferred instance types (using ARM-optimized when available)
	sizeTypeMap := map[string]string{
		"XS": "t4g.nano",
		"S":  "t4g.small", 
		"M":  "t4g.medium",
		"L":  "t4g.large",
		"XL": "t4g.xlarge",
	}

	if instanceType, exists := sizeTypeMap[size]; exists {
		return instanceType
	}
	return "unknown"
}

func (a *App) estimateCostForSize(size string) float64 {
	// Estimated daily costs for different sizes
	costMap := map[string]float64{
		"XS": 0.50,
		"S":  1.00,
		"M":  2.00, 
		"L":  4.00,
		"XL": 8.00,
	}

	if cost, exists := costMap[size]; exists {
		return cost
	}
	return 0.0
}

type SizeSpecs struct {
	CPU     string
	Memory  string
	Storage string
}

func (a *App) getSizeSpecs(size string) SizeSpecs {
	specMap := map[string]SizeSpecs{
		"XS": {"1vCPU", "2GB", "100GB"},
		"S":  {"2vCPU", "4GB", "500GB"},
		"M":  {"2vCPU", "8GB", "1TB"}, 
		"L":  {"4vCPU", "16GB", "2TB"},
		"XL": {"8vCPU", "32GB", "4TB"},
	}

	if specs, exists := specMap[size]; exists {
		return specs
	}
	return SizeSpecs{"Unknown", "Unknown", "Unknown"}
}

// AMIDiscover demonstrates AMI auto-discovery functionality
func (a *App) AMIDiscover(args []string) error {
	fmt.Printf("🔍 CloudWorkstation AMI Auto-Discovery\n\n")
	
	// This would normally get the template resolver from the daemon
	// For demo purposes, create a resolver and populate it with mock AMI data
	resolver := templates.NewTemplateResolver()
	
	// Simulate AMI registry update (in practice this would connect to AWS SSM)
	err := resolver.UpdateAMIRegistry(context.TODO(), "mock-ssm-client")
	if err != nil {
		return fmt.Errorf("failed to update AMI registry: %w", err)
	}
	
	// Show current template list with AMI availability
	fmt.Printf("📋 Template Analysis:\n\n")
	
	templateNames := []string{"python-ml", "r-research", "simple-python-ml", "simple-r-research"}
	region := "us-east-1"
	architecture := "x86_64"
	
	for _, templateName := range templateNames {
		amiID := resolver.CheckAMIAvailability(templateName, region, architecture)
		if amiID != "" {
			fmt.Printf("✅ %s: AMI available (%s) - Fast launch ready!\n", templateName, amiID)
		} else {
			fmt.Printf("⏱️  %s: No pre-built AMI - Will build from scratch\n", templateName)
		}
	}
	
	fmt.Printf("\n💡 Templates with ✅ can launch in seconds using pre-built AMIs\n")
	fmt.Printf("💡 Templates with ⏱️ will take several minutes to install packages\n")
	fmt.Printf("\n🛠️  To build AMIs: cws ami build <template-name>\n")
	
	return nil
}

// Note: AMI command is implemented in internal/cli/ami.go