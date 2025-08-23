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
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/scttfrdmn/cloudworkstation/pkg/api/client"
	"github.com/scttfrdmn/cloudworkstation/pkg/pricing"
	"github.com/scttfrdmn/cloudworkstation/pkg/profile"
	"github.com/scttfrdmn/cloudworkstation/pkg/project"
	"github.com/scttfrdmn/cloudworkstation/pkg/templates"
	"github.com/scttfrdmn/cloudworkstation/pkg/types"
	"github.com/spf13/cobra"
)

// App represents the CLI application
type App struct {
	version          string
	apiClient        client.CloudWorkstationAPI
	ctx              context.Context // Context for AWS operations
	tuiCommand       *cobra.Command
	config           *Config
	profileManager   *profile.ManagerEnhanced
	launchDispatcher *LaunchCommandDispatcher // Command Pattern for launch flags
	instanceCommands *InstanceCommands        // Instance management commands
	storageCommands  *StorageCommands         // Storage management commands
	templateCommands *TemplateCommands        // Template management commands
	systemCommands   *SystemCommands          // System and daemon management commands
	scalingCommands  *ScalingCommands         // Scaling and rightsizing commands
	testMode         bool                     // Skip actual SSH execution in tests
}

// NewApp creates a new CLI application
func NewApp(version string) *App {
	// Load config
	config, err := LoadConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: Failed to load config: %v\n", err)
		config = &Config{}                   // Use empty config
		config.Daemon.URL = DefaultDaemonURL // Default URL (CWS on phone keypad)
	}

	// Initialize profile manager
	profileManager, err := profile.NewManagerEnhanced()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: Failed to initialize profile manager: %v\n", err)
		// Continue without profile manager
	}

	// Initialize API client
	apiURL := config.Daemon.URL
	if envURL := os.Getenv(DaemonURLEnvVar); envURL != "" {
		apiURL = envURL
	}

	// Create API client with configuration
	baseClient := client.NewClientWithOptions(apiURL, client.Options{
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
	app.systemCommands = NewSystemCommands(app)
	app.scalingCommands = NewScalingCommands(app)

	// Initialize TUI command
	app.tuiCommand = NewTUICommand()

	return app
}

// ensureDaemonRunning checks if the daemon is running and auto-starts it if needed
func (a *App) ensureDaemonRunning() error {
	// Check if auto-start is disabled via environment variable
	if os.Getenv(AutoStartDisableEnvVar) != "" {
		// Auto-start disabled, just check if daemon is running
		if err := a.apiClient.Ping(a.ctx); err != nil {
			return fmt.Errorf("%s\n\n💡 Tip: Auto-start is disabled via %s environment variable",
				DaemonNotRunningMessage, AutoStartDisableEnvVar)
		}
		return nil
	}

	// Check if daemon is already running
	if err := a.apiClient.Ping(a.ctx); err == nil {
		return nil // Already running
	}

	// Auto-start daemon with user feedback
	fmt.Println(DaemonAutoStartMessage)
	fmt.Printf("⏳ Please wait while the daemon initializes (typically 2-3 seconds)...\n")

	// Use the systemCommands to start the daemon
	if err := a.systemCommands.Daemon([]string{"start"}); err != nil {
		fmt.Println(DaemonAutoStartFailedMessage)
		fmt.Printf("\n💡 Troubleshooting:\n")
		fmt.Printf("   • Check if 'cwsd' binary is in your PATH\n")
		fmt.Printf("   • Try manual start: cws daemon start\n")
		fmt.Printf("   • Check daemon logs for errors\n")
		return WrapAPIError("auto-start daemon", err)
	}

	fmt.Println(DaemonAutoStartSuccessMessage)
	return nil
}

// NewAppWithClient creates a new CLI application with a custom API client
func NewAppWithClient(version string, apiClient client.CloudWorkstationAPI) *App {
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
		apiClient:        apiClient,
		ctx:              context.Background(),
		config:           config,
		profileManager:   profileManager,
		launchDispatcher: NewLaunchCommandDispatcher(),
		testMode:         true, // Enable test mode when using mock client
	}

	// Initialize command modules
	app.instanceCommands = NewInstanceCommands(app)
	app.storageCommands = NewStorageCommands(app)
	app.templateCommands = NewTemplateCommands(app)
	app.systemCommands = NewSystemCommands(app)
	app.scalingCommands = NewScalingCommands(app)

	// Initialize TUI command
	app.tuiCommand = NewTUICommand()

	return app
}

// TUI launches the terminal UI
func (a *App) TUI(_ []string) error {
	// In test mode, just verify TUI command exists without running it
	if a.testMode {
		if a.tuiCommand == nil {
			return fmt.Errorf("TUI command not initialized")
		}
		return nil
	}
	return a.tuiCommand.Execute()
}

// Launch handles the launch command
func (a *App) Launch(args []string) error {
	if len(args) < 2 {
		return NewUsageError("cws launch <template> <name>", "cws launch python-ml my-workstation")
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

	// Ensure daemon is running (auto-start if needed)
	if err := a.ensureDaemonRunning(); err != nil {
		return err
	}

	response, err := a.apiClient.LaunchInstance(a.ctx, req)
	if err != nil {
		return WrapAPIError("launch instance "+req.Name, err)
	}

	fmt.Printf("🚀 %s\n", response.Message)

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
	w := tabwriter.NewWriter(os.Stdout, TabWriterMinWidth, TabWriterTabWidth, TabWriterPadding, TabWriterPadChar, TabWriterFlags)

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
		fmt.Printf("   Your monthly est:  $%.4f\n", summary.TotalRunningCost*DaysToMonthEstimate)
		fmt.Printf("   List price daily:  $%.4f\n", summary.TotalListCost)
		fmt.Printf("   Daily savings:     $%.4f (%.1f%%)\n", totalSavings, savingsPercent)
		fmt.Printf("   Historical spend:  $%.4f\n", summary.TotalHistoricalSpend)
		if pricingConfig.Institution != "" {
			fmt.Printf("   Institution:       %s\n", pricingConfig.Institution)
		}
	} else {
		fmt.Printf("   Running instances: %d\n", summary.RunningInstances)
		fmt.Printf("   Daily cost:        $%.4f\n", summary.TotalRunningCost)
		fmt.Printf("   Monthly estimate:  $%.4f\n", summary.TotalRunningCost*DaysToMonthEstimate)
		fmt.Printf("   Historical spend:  $%.4f\n", summary.TotalHistoricalSpend)
	}
}

// monitorLaunchProgress monitors and displays enhanced real-time launch progress
func (a *App) monitorLaunchProgress(instanceName, templateName string) error {
	// Get template information for enhanced progress reporting
	template, err := a.apiClient.GetTemplate(a.ctx, templateName)
	if err != nil {
		fmt.Printf("%s\n", FormatWarningMessage("Template info", "Could not get template info, using basic progress"))
	}

	// Create enhanced progress reporter
	progressReporter := NewProgressReporter(instanceName, templateName, template)
	progressReporter.ShowHeader()

	// Monitor launch with enhanced progress reporting
	return a.monitorLaunchWithEnhancedProgress(progressReporter, template)
}

// monitorLaunchWithEnhancedProgress monitors launch with enhanced progress reporting
func (a *App) monitorLaunchWithEnhancedProgress(reporter *ProgressReporter, template *types.Template) error {
	startTime := time.Now()
	maxDuration := 20 * time.Minute // Maximum monitoring time

	for {
		elapsed := time.Since(startTime)

		// Check for timeout
		if elapsed > maxDuration {
			fmt.Printf("⚠️  Launch monitoring timeout (%s). Instance may still be setting up.\n",
				reporter.FormatDuration(maxDuration))
			fmt.Printf("💡 Check status with: cws list\n")
			fmt.Printf("💡 Try connecting: cws connect %s\n", reporter.instanceName)
			return nil
		}

		// Get current instance status
		instance, err := a.apiClient.GetInstance(a.ctx, reporter.instanceName)
		if err != nil {
			// If we can't get instance info initially, show initializing
			if elapsed < 30*time.Second {
				fmt.Printf("⏳ Instance initializing...\n")
			} else {
				// After 30 seconds, show as potential issue
				fmt.Printf("⚠️  Unable to get instance status after %s\n", reporter.FormatDuration(elapsed))
				fmt.Printf("💡 Instance may still be launching. Check with: cws list\n")
			}
			time.Sleep(5 * time.Second)
			continue
		}

		// Update progress display
		reporter.UpdateProgress(instance, elapsed)

		// Check for completion or error states
		switch instance.State {
		case "running":
			// For package-based templates, verify setup is complete
			// Check if it's NOT an AMI-based template
			isAMI := false
			if template != nil {
				isAMI = len(template.AMI) > 0
			}
			if !isAMI && !strings.Contains(strings.ToLower(reporter.templateName), "ami") {
				// Check if setup is actually complete (simplified check)
				if elapsed > 2*time.Minute { // Allow some setup time
					// Try to connect to verify setup completion
					_, connErr := a.apiClient.ConnectInstance(a.ctx, reporter.instanceName)
					if connErr == nil {
						reporter.ShowCompletion(instance)
						return nil
					}
					// If connection fails but we've been running a while, consider it complete
					if elapsed > 10*time.Minute {
						reporter.ShowCompletion(instance)
						return nil
					}
				}
			} else {
				// AMI-based template - instance running means ready
				reporter.ShowCompletion(instance)
				return nil
			}

		case "stopped", "stopping":
			err := fmt.Errorf("instance stopped during launch")
			reporter.ShowError(err, instance)
			return err

		case "terminated":
			err := fmt.Errorf("instance terminated during launch")
			reporter.ShowError(err, instance)
			return err

		case "dry-run":
			fmt.Printf("✅ Dry-run validation successful! No actual instance launched.\n")
			return nil
		}

		// Wait before next check
		time.Sleep(5 * time.Second)
	}
}

// monitorAMILaunchProgress shows simple progress for AMI-based launches
func (a *App) monitorAMILaunchProgress(instanceName string) error {
	fmt.Printf(LaunchProgressAMIMessage + "\n\n")

	for i := 0; i < 60; i++ { // Monitor for up to 5 minutes
		instance, err := a.apiClient.GetInstance(a.ctx, instanceName)
		if err != nil {
			if i == 0 {
				fmt.Printf(StateMessageInitializing + "\n")
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
	apiClient client.CloudWorkstationAPI
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
	apiClient client.CloudWorkstationAPI
	ctx       context.Context
}

// NewLaunchProgressMonitor creates launch progress monitor
func NewLaunchProgressMonitor(apiClient client.CloudWorkstationAPI, ctx context.Context) *LaunchProgressMonitor {
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
				fmt.Printf(StateMessageInitializing + "\n")
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

	fmt.Printf(SetupTimeoutMessage + "\n")
	fmt.Printf(SetupTimeoutHelpMessage + "\n")
	fmt.Printf(SetupTimeoutConnectMessage+"\n", instanceName)
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
	fmt.Printf(LaunchProgressPackageMessage + "\n")
	fmt.Printf(LaunchProgressPackageTiming + "\n\n")

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

	// Ensure daemon is running (auto-start if needed)
	if err := a.ensureDaemonRunning(); err != nil {
		return err
	}

	response, err := a.apiClient.ListInstances(a.ctx)
	if err != nil {
		return WrapAPIError("list instances", err)
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
			fmt.Printf(NoInstancesFoundProjectMessage+"\n", projectFilter, projectFilter)
		} else {
			fmt.Println(NoInstancesFoundMessage)
		}
		return nil
	}

	// Show header with project filter info
	if projectFilter != "" {
		fmt.Printf("Workstations in project '%s':\n\n", projectFilter)
	}

	w := tabwriter.NewWriter(os.Stdout, TabWriterMinWidth, TabWriterTabWidth, TabWriterPadding, TabWriterPadChar, TabWriterFlags)
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
			instance.LaunchTime.Format(ShortDateFormat),
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

	// Ensure daemon is running (auto-start if needed)
	if err := a.ensureDaemonRunning(); err != nil {
		return err
	}

	response, err := a.apiClient.ListInstances(a.ctx)
	if err != nil {
		return WrapAPIError("list instances for cost analysis", err)
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

	// In test mode, skip actual SSH execution
	if a.testMode {
		fmt.Printf("Test mode: would execute: %s\n", connectionInfo)
		return nil
	}

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
		return WrapAPIError("execute SSH command", err)
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
	return a.systemCommands.Daemon(args)
}

// Rightsizing handles rightsizing analysis and recommendations
func (a *App) Rightsizing(args []string) error {
	return a.scalingCommands.Rightsizing(args)
}

// Scaling handles dynamic instance scaling operations
func (a *App) Scaling(args []string) error {
	return a.scalingCommands.Scaling(args)
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
		return WrapAPIError("update AMI registry", err)
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

	fmt.Printf("\n💡 Templates with ✅ use pre-built AMIs for faster deployment\n")
	fmt.Printf("💡 Templates with ⏱️ will take several minutes to install packages\n")
	fmt.Printf("\n🛠️  To build AMIs: cws ami build <template-name>\n")

	return nil
}

// Note: AMI command is implemented in internal/cli/ami.go

// Project command implementation
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
	_, _ = fmt.Fprintln(w, "NAME\tID\tOWNER\tBUDGET\tSPENT\tINSTANCES\tCREATED")

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

		_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%s\t$%.2f\t%d\t%s\n",
			proj.Name,
			proj.ID,
			proj.Owner,
			budgetStr,
			spent,
			instanceCount,
			proj.CreatedAt.Format("2006-01-02"),
		)
	}
	_ = w.Flush()

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
	_, _ = fmt.Fprintln(w, "NAME\tTEMPLATE\tSTATE\tPUBLIC IP\tCOST/DAY\tLAUNCHED")

	totalCost := 0.0
	for _, instance := range projectInstances {
		_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%s\t$%.2f\t%s\n",
			instance.Name,
			instance.Template,
			strings.ToUpper(instance.State),
			instance.PublicIP,
			instance.HourlyRate*24,
			instance.LaunchTime.Format("2006-01-02 15:04"),
		)
		if instance.State == "running" {
			totalCost += instance.HourlyRate * 24
		}
	}

	_, _ = fmt.Fprintf(w, "\nTotal daily cost (running instances): $%.2f\n", totalCost)
	_ = w.Flush()

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
				UserID:  email,
				Role:    types.ProjectRole(role),
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
	_, _ = fmt.Fprintln(w, "EMAIL\tROLE\tJOINED\tLAST ACTIVE")

	for _, member := range members {
		lastActive := "never"
		// Note: LastActive not available in current ProjectMember type

		_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%s\n",
			member.UserID,
			member.Role,
			member.AddedAt.Format("2006-01-02"),
			lastActive,
		)
	}
	_ = w.Flush()

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
	_, _ = fmt.Scanln(&confirmation)

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
