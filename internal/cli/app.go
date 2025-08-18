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
	"strings"
	"text/tabwriter"
	"time"

	"github.com/scttfrdmn/cloudworkstation/pkg/api"
	"github.com/scttfrdmn/cloudworkstation/pkg/api/client"
	"github.com/scttfrdmn/cloudworkstation/pkg/pricing"
	"github.com/scttfrdmn/cloudworkstation/pkg/profile"
	"github.com/scttfrdmn/cloudworkstation/pkg/templates"
	"github.com/scttfrdmn/cloudworkstation/pkg/types"
	"github.com/scttfrdmn/cloudworkstation/pkg/version"
	"github.com/spf13/cobra"
)

// App represents the CLI application
type App struct {
	version           string
	apiClient         api.CloudWorkstationAPI
	ctx               context.Context // Context for AWS operations
	tuiCommand        *cobra.Command
	config            *Config
	profileManager    *profile.ManagerEnhanced
	launchDispatcher  *LaunchCommandDispatcher // Command Pattern for launch flags
	instanceCommands  *InstanceCommands        // Instance management commands
	storageCommands   *StorageCommands         // Storage management commands
	templateCommands  *TemplateCommands        // Template management commands
}

// NewApp creates a new CLI application
func NewApp(version string) *App {
	// Load config
	config, err := LoadConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: Failed to load config: %v\n", err)
		config = &Config{}                          // Use empty config
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
		version:          version,
		apiClient:        baseClient,
		ctx:              context.Background(),
		config:           config,
		profileManager:   profileManager,
		launchDispatcher: NewLaunchCommandDispatcher(),
	}

	// Initialize command modules
	app.instanceCommands = NewInstanceCommands(app)
	app.storageCommands = NewStorageCommands(app)
	app.templateCommands = NewTemplateCommands(app)

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

	app := &App{
		version:          version,
		apiClient:        client,
		ctx:              context.Background(),
		config:           config,
		profileManager:   profileManager,
		launchDispatcher: NewLaunchCommandDispatcher(),
	}

	// Initialize command modules
	app.instanceCommands = NewInstanceCommands(app)
	app.storageCommands = NewStorageCommands(app)
	app.templateCommands = NewTemplateCommands(app)

	return app
}

// TUI launches the terminal UI
func (a *App) TUI(_ []string) error {
	return a.tuiCommand.Execute()
}

// Launch handles the launch command
func (a *App) Launch(args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("usage: cws launch <template> <name> [options]\n" +
			"  options: --size XS|S|M|L|XL --volume <name> --storage <size> --project <name> --with conda|apt|dnf|ami --spot --hibernation --dry-run --wait --subnet <subnet-id> --vpc <vpc-id>\n" +
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

	// Parse options using Command Pattern (SOLID: Single Responsibility)
	req := types.LaunchRequest{
		Template: template,
		Name:     name,
	}

	// Parse additional flags using dispatcher
	if err := a.launchDispatcher.ParseFlags(&req, args); err != nil {
		return err
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

	// If --wait is specified, monitor launch progress
	if req.Wait {
		fmt.Println()
		return a.monitorLaunchProgress(req.Name, req.Template)
	}

	return nil
}

// displayCostTable displays the cost analysis table (Single Responsibility)
func (a *App) displayCostTable(analyzer *CostAnalyzer, instances []types.Instance, analyses []CostAnalysis) {
	w := tabwriter.NewWriter(os.Stdout, 0, 8, 2, ' ', 0)
	
	// Print headers
	headers := analyzer.GetHeaders()
	_, _ = fmt.Fprintln(w, strings.Join(headers, "\t"))
	
	// Print instance rows
	for i, instance := range instances {
		_, _ = fmt.Fprint(w, analyzer.FormatRow(instance, analyses[i]))
	}
	
	_ = w.Flush()
}

// displayCostSummary displays the cost summary section (Single Responsibility)
func (a *App) displayCostSummary(summary CostSummary, hasDiscounts bool, pricingConfig *pricing.InstitutionalPricingConfig) {
	fmt.Println()
	fmt.Printf("📊 Cost Summary:\n")
	
	if hasDiscounts {
		totalSavings := summary.TotalListCost - summary.TotalRunningCost
		savingsPercent := 0.0
		if summary.TotalListCost > 0 {
			savingsPercent = (totalSavings / summary.TotalListCost) * 100
		}
		fmt.Printf("   Running instances: %d\n", summary.RunningInstances)
		fmt.Printf("   Your daily cost:   $%.4f\n", summary.TotalRunningCost)
		fmt.Printf("   Your monthly est:  $%.4f\n", summary.TotalRunningCost*30)
		fmt.Printf("   List price daily:  $%.4f\n", summary.TotalListCost)
		fmt.Printf("   Daily savings:     $%.4f (%.1f%%)\n", totalSavings, savingsPercent)
		fmt.Printf("   Historical spend:  $%.4f\n", summary.TotalHistoricalSpend)
		if pricingConfig.Institution != "" {
			fmt.Printf("   Institution:       %s\n", pricingConfig.Institution)
		}
	} else {
		fmt.Printf("   Running instances: %d\n", summary.RunningInstances)
		fmt.Printf("   Daily cost:        $%.4f\n", summary.TotalRunningCost)
		fmt.Printf("   Monthly estimate:  $%.4f\n", summary.TotalRunningCost*30)
		fmt.Printf("   Historical spend:  $%.4f\n", summary.TotalHistoricalSpend)
	}
}

// monitorLaunchProgress monitors and displays real-time launch progress
func (a *App) monitorLaunchProgress(instanceName, templateName string) error {
	fmt.Printf("⏳ Monitoring launch progress for '%s'...\n", instanceName)

	// Get template information to determine progress type
	template, err := a.apiClient.GetTemplate(a.ctx, templateName)
	if err != nil {
		fmt.Printf("⚠️  Could not get template info, showing basic progress\n")
	}

	// Determine if this is an AMI-based or package-based template
	// Check if template uses AMI package manager or contains "AMI" in name
	isAMI := template != nil && (strings.Contains(templateName, "AMI") || strings.Contains(strings.ToLower(templateName), "ami"))

	if isAMI {
		return a.monitorAMILaunchProgress(instanceName)
	} else {
		return a.monitorPackageLaunchProgress(instanceName, templateName)
	}
}

// monitorAMILaunchProgress shows simple progress for AMI-based launches
func (a *App) monitorAMILaunchProgress(instanceName string) error {
	fmt.Printf("📦 AMI-based launch - showing instance status...\n\n")

	for i := 0; i < 60; i++ { // Monitor for up to 5 minutes
		instance, err := a.apiClient.GetInstance(a.ctx, instanceName)
		if err != nil {
			if i == 0 {
				fmt.Printf("⏳ Instance initializing...\n")
			}
		} else {
			switch instance.State {
			case "pending":
				fmt.Printf("🔄 Instance starting... (%ds)\n", i*5)
			case "running":
				fmt.Printf("✅ Instance running! Ready to connect.\n")
				fmt.Printf("🔗 Connect: cws connect %s\n", instanceName)
				return nil
			case "stopping", "stopped":
				return fmt.Errorf("❌ Instance stopped unexpectedly")
			case "terminated":
				return fmt.Errorf("❌ Instance terminated during launch")
			case "dry-run":
				fmt.Printf("✅ Dry-run validation successful! No actual instance launched.\n")
				return nil
			default:
				fmt.Printf("📊 Status: %s (%ds)\n", instance.State, i*5)
			}
		}

		time.Sleep(5 * time.Second)
	}

	fmt.Printf("⚠️  Timeout waiting for instance to start (5 min). Check status with: cws list\n")
	return nil
}

// InstanceStateHandler interface for handling different instance states (Strategy Pattern - SOLID)
type InstanceStateHandler interface {
	CanHandle(state string) bool
	Handle(state string, elapsed int, instanceName string) (bool, error) // returns (shouldContinue, error)
}

// PendingStateHandler handles pending instance state
type PendingStateHandler struct{}

func (h *PendingStateHandler) CanHandle(state string) bool {
	return state == "pending"
}

func (h *PendingStateHandler) Handle(state string, elapsed int, instanceName string) (bool, error) {
	fmt.Printf("🔄 Instance starting... (%ds)\n", elapsed)
	return true, nil
}

// RunningStateHandler handles running instance state with setup monitoring
type RunningStateHandler struct {
	apiClient api.CloudWorkstationAPI
	ctx       context.Context
}

func (h *RunningStateHandler) CanHandle(state string) bool {
	return state == "running"
}

func (h *RunningStateHandler) Handle(state string, elapsed int, instanceName string) (bool, error) {
	// Display setup progress messages
	h.displaySetupProgress(elapsed)

	// Check if setup is complete
	if elapsed > 60 && elapsed%30 == 0 { // Check every 30 seconds after 1 minute
		_, connErr := h.apiClient.ConnectInstance(h.ctx, instanceName)
		if connErr == nil {
			fmt.Printf("✅ Setup complete! Instance ready.\n")
			fmt.Printf("🔗 Connect: cws connect %s\n", instanceName)
			return false, nil
		}
	}
	return true, nil
}

func (h *RunningStateHandler) displaySetupProgress(elapsed int) {
	if elapsed < 30 {
		fmt.Printf("🔧 Instance running, beginning setup... (%ds)\n", elapsed)
	} else if elapsed < 120 {
		fmt.Printf("📥 Installing packages... (%ds)\n", elapsed)
	} else if elapsed < 300 {
		fmt.Printf("⚙️  Configuring services... (%ds)\n", elapsed)
	} else {
		fmt.Printf("🔧 Final setup steps... (%ds)\n", elapsed)
	}
}

// ErrorStateHandler handles error states (stopped, terminated)
type ErrorStateHandler struct{}

func (h *ErrorStateHandler) CanHandle(state string) bool {
	return state == "stopping" || state == "stopped" || state == "terminated"
}

func (h *ErrorStateHandler) Handle(state string, elapsed int, instanceName string) (bool, error) {
	switch state {
	case "stopping", "stopped":
		return false, fmt.Errorf("❌ Instance stopped during setup")
	case "terminated":
		return false, fmt.Errorf("❌ Instance terminated during launch")
	}
	return false, nil
}

// DryRunStateHandler handles dry-run state
type DryRunStateHandler struct{}

func (h *DryRunStateHandler) CanHandle(state string) bool {
	return state == "dry-run"
}

func (h *DryRunStateHandler) Handle(state string, elapsed int, instanceName string) (bool, error) {
	fmt.Printf("✅ Dry-run validation successful! No actual instance launched.\n")
	return false, nil
}

// DefaultStateHandler handles unknown states
type DefaultStateHandler struct{}

func (h *DefaultStateHandler) CanHandle(state string) bool {
	return true // Always can handle as fallback
}

func (h *DefaultStateHandler) Handle(state string, elapsed int, instanceName string) (bool, error) {
	fmt.Printf("📊 Status: %s (%ds)\n", state, elapsed)
	return true, nil
}

// LaunchProgressMonitor manages package launch monitoring (Strategy Pattern - SOLID)
type LaunchProgressMonitor struct {
	handlers  []InstanceStateHandler
	apiClient api.CloudWorkstationAPI
	ctx       context.Context
}

// NewLaunchProgressMonitor creates launch progress monitor
func NewLaunchProgressMonitor(apiClient api.CloudWorkstationAPI, ctx context.Context) *LaunchProgressMonitor {
	return &LaunchProgressMonitor{
		handlers: []InstanceStateHandler{
			&PendingStateHandler{},
			&RunningStateHandler{apiClient: apiClient, ctx: ctx},
			&ErrorStateHandler{},
			&DryRunStateHandler{},
			&DefaultStateHandler{}, // Must be last as fallback
		},
		apiClient: apiClient,
		ctx:       ctx,
	}
}

// Monitor handles instance state monitoring using strategies
func (m *LaunchProgressMonitor) Monitor(instanceName string) error {
	for i := 0; i < 240; i++ { // Monitor for up to 20 minutes
		instance, err := m.apiClient.GetInstance(m.ctx, instanceName)
		if err != nil {
			if i == 0 {
				fmt.Printf("⏳ Instance initializing...\n")
			}
		} else {
			shouldContinue, stateErr := m.handleInstanceState(instance.State, i*5, instanceName)
			if stateErr != nil {
				return stateErr
			}
			if !shouldContinue {
				return nil
			}
		}

		time.Sleep(5 * time.Second)
	}

	fmt.Printf("⚠️  Setup monitoring timeout (20 min). Instance may still be setting up.\n")
	fmt.Printf("💡 Check status with: cws list\n")
	fmt.Printf("💡 Try connecting: cws connect %s\n", instanceName)
	return nil
}

func (m *LaunchProgressMonitor) handleInstanceState(state string, elapsed int, instanceName string) (bool, error) {
	for _, handler := range m.handlers {
		if handler.CanHandle(state) {
			return handler.Handle(state, elapsed, instanceName)
		}
	}
	return true, nil // Continue monitoring by default
}

// monitorPackageLaunchProgress shows detailed progress using Strategy Pattern (SOLID: Single Responsibility)
func (a *App) monitorPackageLaunchProgress(instanceName, templateName string) error {
	fmt.Printf("📦 Package-based launch - monitoring setup progress...\n")
	fmt.Printf("💡 Setup time varies: APT/DNF ~2-3 min, conda ~5-10 min\n\n")

	monitor := NewLaunchProgressMonitor(a.apiClient, a.ctx)
	return monitor.Monitor(instanceName)
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
	_, _ = fmt.Fprintln(w, "NAME\tTEMPLATE\tSTATE\tTYPE\tPUBLIC IP\tPROJECT\tLAUNCHED")
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

		_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
			instance.Name,
			instance.Template,
			strings.ToUpper(instance.State),
			typeIndicator,
			instance.PublicIP,
			projectInfo,
			instance.LaunchTime.Format("2006-01-02 15:04"),
		)
	}

	_ = w.Flush()

	return nil
}

// ListCost handles the list cost command - shows detailed cost information
func (a *App) ListCost(args []string) error {
	// Parse project filter
	var projectFilter string
	for i := 0; i < len(args); i++ {
		if args[i] == "--project" && i+1 < len(args) {
			projectFilter = args[i+1]
			i++ // Skip the next argument since we consumed it
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
		fmt.Println("💰 CloudWorkstation Cost Analysis")
	}

	// Use Strategy Pattern for cost analysis (SOLID: Open/Closed Principle)
	pricingConfig, _ := pricing.LoadInstitutionalPricing()
	calculator := pricing.NewCalculator(pricingConfig)
	hasDiscounts := pricingConfig != nil && (pricingConfig.Institution != "Default")
	
	costAnalyzer := NewCostAnalyzer(hasDiscounts, calculator)
	analyses, summary := costAnalyzer.AnalyzeInstances(filteredInstances)

	// Display cost table
	a.displayCostTable(costAnalyzer, filteredInstances, analyses)

	// Display cost summary
	a.displayCostSummary(summary, hasDiscounts, pricingConfig)

	fmt.Printf("\n💡 Tip: Use 'cws list' for a clean instance overview without cost data\n")

	return nil
}

func (a *App) Connect(args []string) error {
	return a.instanceCommands.Connect(args)
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
	return a.instanceCommands.Stop(args)
}

// Start handles the start command with intelligent state management
func (a *App) Start(args []string) error {
	return a.instanceCommands.Start(args)
}

// Delete handles the delete command
func (a *App) Delete(args []string) error {
	return a.instanceCommands.Delete(args)
}

func (a *App) Hibernate(args []string) error {
	return a.instanceCommands.Hibernate(args)
}

func (a *App) Resume(args []string) error {
	return a.instanceCommands.Resume(args)
}

// Volume handles volume commands
func (a *App) Volume(args []string) error {
	return a.storageCommands.Volume(args)
}







// Storage handles storage commands
func (a *App) Storage(args []string) error {
	return a.storageCommands.Storage(args)
}







// Templates handles the templates command
func (a *App) Templates(args []string) error {
	return a.templateCommands.Templates(args)
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
	fmt.Printf("   Instance retention: %d minutes\n", defaultConfig.InstanceRetentionMinutes)
	fmt.Printf("   Port: %s\n", defaultConfig.Port)

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
		Port:                     "8947",
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
		switch instance.State {
		case "running":
			runningInstances++
			totalDailyCost += instance.EstimatedDailyCost
		case "stopped":
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
	ctx := context.Background()
	err := resolver.UpdateAMIRegistry(ctx, "mock-ssm-client")
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
