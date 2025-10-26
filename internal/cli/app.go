// Package cli implements Prism's command-line interface application.
//
// This package provides the CLI application logic for the Prism client (cws).
// It handles command parsing, API client communication, output formatting, and user
// interaction flows while maintaining Prism's core design principles.
//
// Application Structure:
//   - App: Main CLI application with command routing
//   - Command handlers for all Prism operations
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

	"github.com/scttfrdmn/prism/pkg/api/client"
	"github.com/scttfrdmn/prism/pkg/pricing"
	"github.com/scttfrdmn/prism/pkg/profile"
	"github.com/scttfrdmn/prism/pkg/project"
	"github.com/scttfrdmn/prism/pkg/templates"
	"github.com/scttfrdmn/prism/pkg/types"
	"github.com/spf13/cobra"
)

// App represents the CLI application
type App struct {
	version               string
	apiClient             client.PrismAPI
	ctx                   context.Context // Context for AWS operations
	tuiCommand            *cobra.Command
	config                *Config
	profileManager        *profile.ManagerEnhanced
	launchDispatcher      *LaunchCommandDispatcher // Command Pattern for launch flags
	instanceCommands      *InstanceCommands        // Instance management commands
	storageCommands       *StorageCommands         // Storage management commands
	templateCommands      *TemplateCommands        // Template management commands
	systemCommands        *SystemCommands          // System and daemon management commands
	scalingCommands       *ScalingCommands         // Scaling and rightsizing commands
	snapshotCommands      *SnapshotCommands        // Instance snapshot management commands
	backupCommands        *BackupCommands          // Data backup and restore management commands
	webCommands           *WebCommands             // Web service management commands
	testMode              bool                     // Skip actual SSH execution in tests
	versionCheckCompleted bool                     // Cache version compatibility check result
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

	// Load API key from daemon state if available
	apiKey := loadAPIKeyFromState()

	// Determine AWS profile and region to use
	// Priority: Current Prism profile > Config file > Environment variables
	awsProfile := config.AWS.Profile
	awsRegion := config.AWS.Region

	// Check if there's an active Prism profile
	if profileManager != nil {
		if currentProfile, err := profileManager.GetCurrentProfile(); err == nil && currentProfile != nil {
			// Use Prism profile settings
			awsProfile = currentProfile.AWSProfile
			awsRegion = currentProfile.Region
		}
	}

	// Create API client with configuration
	baseClient := client.NewClientWithOptions(apiURL, client.Options{
		AWSProfile: awsProfile,
		AWSRegion:  awsRegion,
		APIKey:     apiKey,
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
	app.snapshotCommands = NewSnapshotCommands(app)
	app.backupCommands = NewBackupCommands(app)
	app.webCommands = NewWebCommands(app)

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
		// Daemon is running, check version compatibility only if not already checked
		if !a.versionCheckCompleted {
			if err := a.checkVersionCompatibility(); err != nil {
				return fmt.Errorf("version compatibility check failed: %w", err)
			}
			a.versionCheckCompleted = true
		}
		return nil // Already running and compatible
	}

	// Auto-start daemon with user feedback
	fmt.Println(DaemonAutoStartMessage)
	fmt.Printf("⏳ Please wait while the daemon initializes (typically 2-3 seconds)...\n")

	// Use the systemCommands to start the daemon
	if err := a.systemCommands.Daemon([]string{"start"}); err != nil {
		fmt.Println(DaemonAutoStartFailedMessage)
		fmt.Printf("\n💡 Troubleshooting:\n")
		fmt.Printf("   • Check if 'prismd' binary is in your PATH\n")
		fmt.Printf("   • Try manual start: prism daemon start\n")
		fmt.Printf("   • Check daemon logs for errors\n")
		return WrapAPIError("auto-start daemon", err)
	}

	fmt.Println(DaemonAutoStartSuccessMessage)

	// Check version compatibility after successful start
	if err := a.checkVersionCompatibility(); err != nil {
		return fmt.Errorf("version compatibility check failed after daemon auto-start: %w", err)
	}
	a.versionCheckCompleted = true

	return nil
}

// checkVersionCompatibility verifies that the CLI and daemon versions are compatible
func (a *App) checkVersionCompatibility() error {
	return a.apiClient.CheckVersionCompatibility(a.ctx, a.version)
}

// NewAppWithClient creates a new CLI application with a custom API client
func NewAppWithClient(version string, apiClient client.PrismAPI) *App {
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
	app.snapshotCommands = NewSnapshotCommands(app)
	app.backupCommands = NewBackupCommands(app)
	app.webCommands = NewWebCommands(app)

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
		return NewUsageError("prism launch <template> <name>", "prism launch python-ml my-workstation")
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

	// Show immediate feedback with animated spinner
	spinner := NewSpinner(fmt.Sprintf("Launching workspace '%s' from template '%s'", req.Name, req.Template))
	spinner.Start()

	response, err := a.apiClient.LaunchInstance(a.ctx, req)

	spinner.StopWithMessage(fmt.Sprintf("✅ %s", response.Message))

	if err != nil {
		return WrapAPIError("launch workspace "+req.Name, err)
	}

	// Show project information if launched in a project
	if req.ProjectID != "" {
		fmt.Printf("📁 Project: %s\n", req.ProjectID)
		fmt.Printf("🏷️  Workspace will be tracked under project budget\n")
	}

	// Determine if we should monitor progress
	// For package-based templates, always monitor (they take 5-10 minutes)
	// For AMI templates, only monitor if --wait is specified
	shouldMonitor := req.Wait

	// Check if this is a package-based template (no AMI = needs package installation)
	if !shouldMonitor {
		templateInfo, err := a.apiClient.GetTemplate(a.ctx, req.Template)
		if err == nil && (templateInfo.AMI == nil || len(templateInfo.AMI) == 0) {
			// Package-based template - always monitor progress
			shouldMonitor = true
			fmt.Printf("\n💡 Package installation will take 5-10 minutes. Monitoring progress...\n")
			fmt.Printf("   (Use Ctrl+C to return to prompt - workspace will continue setup)\n")
		}
	}

	// Monitor launch progress if needed
	if shouldMonitor {
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
		fmt.Printf("   Running workspaces: %d\n", summary.RunningInstances)
		fmt.Printf("   Your daily cost:   $%.4f\n", summary.TotalRunningCost)
		fmt.Printf("   Your monthly est:  $%.4f\n", summary.TotalRunningCost*DaysToMonthEstimate)
		fmt.Printf("   List price daily:  $%.4f\n", summary.TotalListCost)
		fmt.Printf("   Daily savings:     $%.4f (%.1f%%)\n", totalSavings, savingsPercent)
		fmt.Printf("   Historical spend:  $%.4f\n", summary.TotalHistoricalSpend)
		if pricingConfig.Institution != "" {
			fmt.Printf("   Institution:       %s\n", pricingConfig.Institution)
		}
	} else {
		fmt.Printf("   Running workspaces: %d\n", summary.RunningInstances)
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
			fmt.Printf("⚠️  Launch monitoring timeout (%s). Workspace may still be setting up.\n",
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
				fmt.Printf("⏳ Workspace initializing...\n")
			} else {
				// After 30 seconds, show as potential issue
				fmt.Printf("⚠️  Unable to get instance status after %s\n", reporter.FormatDuration(elapsed))
				fmt.Printf("💡 Workspace may still be launching. Check with: cws list\n")
			}
			time.Sleep(5 * time.Second)
			continue
		}

		// Update progress display
		reporter.UpdateProgress(instance, elapsed)

		// Check for completion or error states
		switch instance.State {
		case "running":
			// Determine if this is an AMI-based template
			// Templates with pre-built AMIs launch immediately (30s) vs package installation (5-10min)
			isAMI := false
			if template != nil {
				isAMI = len(template.AMI) > 0
			}

			if isAMI || strings.Contains(strings.ToLower(reporter.templateName), "ami") {
				// AMI-based template - instance running means ready
				// Pre-built AMIs include all packages pre-installed
				reporter.ShowCompletion(instance)
				return nil
			}

			// Package-based template - switch to detailed progress monitoring
			// This monitors actual setup progress via SSH and cloud-init status
			fmt.Printf("\n📦 Workspace running - monitoring package installation progress...\n\n")
			return a.monitorSetupProgress(instance)

		case "stopped", "stopping":
			err := fmt.Errorf("workspace stopped during launch")
			reporter.ShowError(err, instance)
			return err

		case "terminated":
			err := fmt.Errorf("workspace terminated during launch")
			reporter.ShowError(err, instance)
			return err

		case "dry-run":
			fmt.Printf("✅ Dry-run validation successful! No actual workspace launched.\n")
			return nil
		}

		// Wait before next check
		time.Sleep(5 * time.Second)
	}
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
	fmt.Printf("🔄 Workspace starting... (%ds)\n", elapsed)
	return true, nil
}

// RunningStateHandler handles running instance state with setup monitoring
type RunningStateHandler struct {
	apiClient client.PrismAPI
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
			fmt.Printf("✅ Setup complete! Workspace ready.\n")
			fmt.Printf("🔗 Connect: cws connect %s\n", instanceName)
			return false, nil
		}
	}
	return true, nil
}

func (h *RunningStateHandler) displaySetupProgress(elapsed int) {
	if elapsed < 30 {
		fmt.Printf("🔧 Workspace running, beginning setup... (%ds)\n", elapsed)
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
		return false, fmt.Errorf("❌ Workspace stopped during setup")
	case "terminated":
		return false, fmt.Errorf("❌ Workspace terminated during launch")
	}
	return false, nil
}

// DryRunStateHandler handles dry-run state
type DryRunStateHandler struct{}

func (h *DryRunStateHandler) CanHandle(state string) bool {
	return state == "dry-run"
}

func (h *DryRunStateHandler) Handle(state string, elapsed int, instanceName string) (bool, error) {
	fmt.Printf("✅ Dry-run validation successful! No actual workspace launched.\n")
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
	apiClient client.PrismAPI
	ctx       context.Context
}

// NewLaunchProgressMonitor creates launch progress monitor
func NewLaunchProgressMonitor(apiClient client.PrismAPI, ctx context.Context) *LaunchProgressMonitor {
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

// List handles the list command with optional project filtering
func (a *App) List(args []string) error {
	// Parse arguments for project filtering, detailed output, and refresh
	var projectFilter string
	var detailed bool
	var refresh bool
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "--project" && i+1 < len(args):
			projectFilter = args[i+1]
			i++
		case arg == "--detailed" || arg == "-d":
			detailed = true
		case arg == "--refresh" || arg == "-r":
			refresh = true
		}
	}

	// Ensure daemon is running (auto-start if needed)
	if err := a.ensureDaemonRunning(); err != nil {
		return err
	}

	response, err := a.apiClient.ListInstancesWithRefresh(a.ctx, refresh)
	if err != nil {
		return WrapAPIError("list workspaces", err)
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

	// Show different headers based on detailed flag
	if detailed {
		_, _ = fmt.Fprintln(w, "NAME\tTEMPLATE\tSTATE\tTYPE\tREGION\tAZ\tPUBLIC IP\tPROJECT\tTOTAL $\tEFF $/HR\tLAUNCHED")
	} else {
		_, _ = fmt.Fprintln(w, "NAME\tTEMPLATE\tSTATE\tTYPE\tPUBLIC IP\tPROJECT\tTOTAL $\tEFF $/HR\tLAUNCHED")
	}

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

		// Format cost information
		currentCost := fmt.Sprintf("$%.4f", instance.CurrentSpend)
		effectiveRate := fmt.Sprintf("$%.4f", instance.EffectiveRate)

		// For stopped/terminated instances, compute cost goes to $0 but storage persists
		// Calculate EBS storage cost: $0.10/GB/month = ~$0.00014/GB/hour
		// Typical root volume is 8-100GB, so $0.001-$0.014/hour storage cost
		if instance.State == "stopped" || instance.State == "terminated" || instance.State == "stopping" {
			// Estimate EBS storage cost (rough estimate: ~$0.005/hr for typical volumes)
			// In production, this should be calculated from actual EBS volumes attached
			estimatedStorageCostPerHour := 0.005
			numEBSVolumes := len(instance.AttachedEBSVolumes)
			if numEBSVolumes == 0 {
				numEBSVolumes = 1 // At least root volume
			}
			storageCost := estimatedStorageCostPerHour * float64(numEBSVolumes)

			// CurrentSpend continues to accumulate storage costs
			// EffectiveRate shows only ongoing storage costs (EC2 compute is $0)
			effectiveRate = fmt.Sprintf("$%.4f", storageCost)

			// Note: CurrentSpend keeps its value (accumulated costs to date)
		}

		if detailed {
			// Detailed output with region and AZ
			region := instance.Region
			if region == "" {
				region = "-"
			}
			az := instance.AvailabilityZone
			if az == "" {
				az = "-"
			}

			_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
				instance.Name,
				instance.Template,
				strings.ToUpper(instance.State),
				typeIndicator,
				region,
				az,
				instance.PublicIP,
				projectInfo,
				currentCost,
				effectiveRate,
				instance.LaunchTime.Format(ShortDateFormat),
			)
		} else {
			// Standard output
			_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
				instance.Name,
				instance.Template,
				strings.ToUpper(instance.State),
				typeIndicator,
				instance.PublicIP,
				projectInfo,
				currentCost,
				effectiveRate,
				instance.LaunchTime.Format(ShortDateFormat),
			)
		}
	}

	_ = w.Flush()

	// Show cost summary for running instances
	fmt.Println()
	runningCount := 0
	totalCurrentCost := 0.0
	totalEffectiveCost := 0.0
	for _, instance := range filteredInstances {
		if instance.State == "running" {
			runningCount++
			totalCurrentCost += instance.CurrentSpend
			totalEffectiveCost += instance.EffectiveRate
		}
	}

	if runningCount > 0 {
		fmt.Printf("💰 Cost Summary:\n")
		fmt.Printf("   Running workspaces: %d\n", runningCount)
		fmt.Printf("   Total accumulated:  $%.4f (since launch)\n", totalCurrentCost)
		fmt.Printf("   Effective rate:     $%.4f/hr (actual usage)\n", totalEffectiveCost)
		fmt.Printf("   Estimated daily:    $%.2f (at current rate)\n", totalEffectiveCost*24)
		fmt.Printf("\n💡 Tip: Use 'cws list cost' for detailed cost breakdown with savings analysis\n")
	}

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
		return WrapAPIError("list workspaces for cost analysis", err)
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
		fmt.Println("💰 Prism Cost Analysis")
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

	fmt.Printf("\n💡 Tip: Use 'cws list' for a clean workspace overview without cost data\n")

	return nil
}

func (a *App) Connect(args []string) error {
	return a.instanceCommands.Connect(args)
}

// Exec handles the exec command
func (a *App) Exec(args []string) error {
	return a.instanceCommands.Exec(args)
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

// Snapshot handles instance snapshot commands
func (a *App) Snapshot(args []string) error {
	return a.snapshotCommands.Snapshot(args)
}

// Backup handles data backup commands
func (a *App) Backup(args []string) error {
	return a.backupCommands.Backup(args)
}

// Restore handles data restore commands
func (a *App) Restore(args []string) error {
	return a.backupCommands.Restore(args)
}

// Web handles web service commands
func (a *App) Web(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("web command requires an action: list, open, or close")
	}

	action := args[0]
	switch action {
	case "list":
		return a.webCommands.List(args[1:])
	case "open":
		return a.webCommands.Open(args[1:])
	case "close":
		return a.webCommands.Close(args[1:])
	default:
		return fmt.Errorf("unknown web action: %s (available: list, open, close)", action)
	}
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
	fmt.Printf("🔍 Prism AMI Auto-Discovery\n\n")

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

// Note: Marketplace command is implemented in internal/cli/marketplace.go

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
		return fmt.Errorf("usage: cws project budget <action> <project> [options]")
	}

	action := args[0]
	remainingArgs := args[1:]

	switch action {
	case "status":
		return a.projectBudgetStatus(remainingArgs)
	case "set", "enable":
		return a.projectBudgetSet(remainingArgs)
	case "update":
		return a.projectBudgetUpdate(remainingArgs)
	case "disable":
		return a.projectBudgetDisable(remainingArgs)
	case "history":
		return a.projectBudgetHistory(remainingArgs)
	default:
		// Legacy support: if first arg is not a subcommand, assume it's a project name for status
		return a.projectBudgetStatus(args)
	}
}

func (a *App) projectBudgetStatus(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: cws project budget status <project>")
	}

	projectName := args[0]

	// Get detailed budget status
	budgetStatus, err := a.apiClient.GetProjectBudgetStatus(a.ctx, projectName)
	if err != nil {
		return fmt.Errorf("failed to get budget status: %w", err)
	}

	fmt.Printf("💰 Budget Status for '%s':\n", projectName)

	if !budgetStatus.BudgetEnabled {
		fmt.Printf("   Budget: Not enabled\n")
		fmt.Printf("   💡 Enable cost tracking with: cws project budget set %s <amount>\n", projectName)
		return nil
	}

	fmt.Printf("   Total Budget: $%.2f\n", budgetStatus.TotalBudget)
	fmt.Printf("   Spent: $%.2f (%.1f%%)\n",
		budgetStatus.SpentAmount,
		budgetStatus.SpentPercentage*100)
	fmt.Printf("   Remaining: $%.2f\n", budgetStatus.RemainingBudget)

	if budgetStatus.ProjectedMonthlySpend > 0 {
		fmt.Printf("   Projected Monthly: $%.2f\n", budgetStatus.ProjectedMonthlySpend)
	}

	if budgetStatus.DaysUntilBudgetExhausted != nil {
		fmt.Printf("   Days Until Exhausted: %d\n", *budgetStatus.DaysUntilBudgetExhausted)
	}

	if len(budgetStatus.ActiveAlerts) > 0 {
		fmt.Printf("   🚨 Active Alerts:\n")
		for _, alert := range budgetStatus.ActiveAlerts {
			fmt.Printf("      • %s\n", alert)
		}
	}

	if len(budgetStatus.TriggeredActions) > 0 {
		fmt.Printf("   ⚡ Recent Actions:\n")
		for _, action := range budgetStatus.TriggeredActions {
			fmt.Printf("      • %s\n", action)
		}
	}

	return nil
}

func (a *App) projectBudgetSet(args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("usage: cws project budget set <project> <amount> [options]")
	}

	projectName := args[0]
	budgetAmount, err := strconv.ParseFloat(args[1], 64)
	if err != nil {
		return fmt.Errorf("invalid budget amount: %s", args[1])
	}

	if budgetAmount <= 0 {
		return fmt.Errorf("budget amount must be greater than 0")
	}

	// NOTE: This is a simplified legacy function for backwards compatibility
	// For full budget creation with all flags (--monthly-limit, --daily-limit, --alert, --action, etc.),
	// use the Cobra-based command: `prism budget create <project> <amount> [flags]`
	// See internal/cli/budget_commands.go for complete implementation with all features

	req := client.SetProjectBudgetRequest{
		TotalBudget:     budgetAmount,
		BudgetPeriod:    types.BudgetPeriodProject,
		AlertThresholds: []types.BudgetAlert{},
		AutoActions:     []types.BudgetAutoAction{},
	}

	// Add a default 80% warning alert (for full alert customization, use `prism budget create`)
	req.AlertThresholds = append(req.AlertThresholds, types.BudgetAlert{
		Threshold: 0.8,
		Type:      types.BudgetAlertEmail,
		Enabled:   true,
	})

	response, err := a.apiClient.SetProjectBudget(a.ctx, projectName, req)
	if err != nil {
		return fmt.Errorf("failed to set budget: %w", err)
	}

	fmt.Printf("✅ Budget configured for project '%s'\n", projectName)
	fmt.Printf("   Total Budget: $%.2f\n", budgetAmount)
	fmt.Printf("   Cost tracking enabled\n")

	if message, ok := response["message"].(string); ok {
		fmt.Printf("   %s\n", message)
	}

	return nil
}

func (a *App) projectBudgetUpdate(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: cws project budget update <project> [options]")
	}

	projectName := args[0]

	// For now, implement basic update - this would be enhanced with flag parsing
	req := client.UpdateProjectBudgetRequest{}

	response, err := a.apiClient.UpdateProjectBudget(a.ctx, projectName, req)
	if err != nil {
		return fmt.Errorf("failed to update budget: %w", err)
	}

	fmt.Printf("✅ Budget updated for project '%s'\n", projectName)

	if message, ok := response["message"].(string); ok {
		fmt.Printf("   %s\n", message)
	}

	return nil
}

func (a *App) projectBudgetDisable(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: cws project budget disable <project>")
	}

	projectName := args[0]

	response, err := a.apiClient.DisableProjectBudget(a.ctx, projectName)
	if err != nil {
		return fmt.Errorf("failed to disable budget: %w", err)
	}

	fmt.Printf("✅ Budget disabled for project '%s'\n", projectName)
	fmt.Printf("   Cost tracking stopped\n")

	if message, ok := response["message"].(string); ok {
		fmt.Printf("   %s\n", message)
	}

	return nil
}

func (a *App) projectBudgetHistory(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: cws project budget history <project> [--days N]")
	}

	projectName := args[0]

	// For now, show a placeholder - this would be enhanced with actual cost history data
	fmt.Printf("📊 Budget History for '%s':\n", projectName)
	fmt.Printf("   (Cost history functionality would be implemented here)\n")
	fmt.Printf("   💡 Use 'cws project budget status %s' for current spending\n", projectName)

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
		return fmt.Errorf("failed to list workspaces: %w", err)
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
		fmt.Printf("Launch one with: cws launch <template> <workspace-name> --project %s\n", projectName)
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

			// Get current user from profile context
			currentUser := "system"
			if a.profileManager != nil {
				if profile, err := a.profileManager.GetCurrentProfile(); err == nil {
					currentUser = profile.Name
				}
			}

			req := project.AddMemberRequest{
				UserID:  email,
				Role:    types.ProjectRole(role),
				AddedBy: currentUser,
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

// loadAPIKeyFromState attempts to load the API key from daemon state
func loadAPIKeyFromState() string {
	// Try to load daemon state to get API key
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "" // No API key available
	}

	stateFile := filepath.Join(homeDir, ".prism", "state.json")
	data, err := os.ReadFile(stateFile)
	if err != nil {
		return "" // No state file or can't read it
	}

	// Parse state to extract API key
	var state struct {
		Config struct {
			APIKey string `json:"api_key"`
		} `json:"config"`
	}

	if err := json.Unmarshal(data, &state); err != nil {
		return "" // Invalid state format
	}

	return state.Config.APIKey
}

// monitorSetupProgress monitors setup progress using SSH-based monitoring
func (a *App) monitorSetupProgress(instance *types.Instance) error {
	// We need the SSH key to monitor progress
	// Use the tunnel manager's key resolution logic
	sshKeyPath, err := a.findSSHKey(instance.Region)
	if err != nil {
		// Fall back to basic monitoring if no SSH key
		fmt.Printf("⚠️  SSH key not found - using basic progress monitoring\n")
		fmt.Printf("💡 Expected key: cws-test-%s-key in ~/.ssh/\n", instance.Region)
		return a.basicSetupMonitoring(instance)
	}

	// Determine username (ubuntu for standard AMIs)
	username := instance.Username
	if username == "" {
		username = "ubuntu"
	}

	// Create progress monitor (from daemon package, need to import it properly)
	// For now, use basic monitoring with cloud-init status check
	fmt.Printf("🔍 Monitoring setup with SSH key: %s\n", filepath.Base(sshKeyPath))
	fmt.Printf("👤 Username: %s\n", username)
	fmt.Printf("🌐 IP: %s\n\n", instance.PublicIP)

	// Poll for setup completion
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	startTime := time.Now()
	lastStage := ""

	for {
		select {
		case <-a.ctx.Done():
			return a.ctx.Err()
		case <-ticker.C:
			elapsed := time.Since(startTime)

			// Check cloud-init status via SSH
			status := a.checkSetupStatus(sshKeyPath, username, instance.PublicIP)

			if status != lastStage {
				fmt.Printf("📦 %s (%s)\n", status, elapsed.Round(time.Second))
				lastStage = status
			}

			// Check if complete
			if strings.Contains(status, "Complete") || strings.Contains(status, "ready") {
				fmt.Printf("\n✅ Setup complete! Workspace ready.\n")
				fmt.Printf("⏱️  Total setup time: %s\n", elapsed.Round(time.Second))
				fmt.Printf("🔗 Connect: cws connect %s\n", instance.Name)
				return nil
			}

			// Timeout after 15 minutes
			if elapsed > 15*time.Minute {
				fmt.Printf("\n⚠️  Setup taking longer than expected\n")
				fmt.Printf("💡 Workspace may still be configuring. Check with: cws list\n")
				return nil
			}
		}
	}
}

// findSSHKey finds the SSH key for a region
func (a *App) findSSHKey(region string) (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	// Standard format: cws-test-{region}-key
	keyPaths := []string{
		filepath.Join(homeDir, ".ssh", fmt.Sprintf("cws-test-%s-key", region)),
		filepath.Join(homeDir, ".prism", "profiles", "test", "ssh", fmt.Sprintf("cws-test-%s-key", region)),
	}

	for _, path := range keyPaths {
		if _, err := os.Stat(path); err == nil {
			return path, nil
		}
	}

	return "", fmt.Errorf("SSH key not found for region %s", region)
}

// checkSetupStatus checks the setup status via SSH
func (a *App) checkSetupStatus(sshKeyPath, username, ip string) string {
	ctx, cancel := context.WithTimeout(a.ctx, 5*time.Second)
	defer cancel()

	// Try to get cloud-init status
	cmd := exec.CommandContext(ctx, "ssh",
		"-o", "ConnectTimeout=3",
		"-o", "StrictHostKeyChecking=no",
		"-o", "UserKnownHostsFile=/dev/null",
		"-o", "LogLevel=ERROR",
		"-i", sshKeyPath,
		fmt.Sprintf("%s@%s", username, ip),
		"cloud-init status 2>/dev/null || echo 'status: unknown'",
	)

	output, err := cmd.Output()
	if err != nil {
		return "Waiting for SSH..."
	}

	statusStr := strings.TrimSpace(string(output))

	// Parse cloud-init status
	if strings.Contains(statusStr, "status: done") {
		return "Setup complete"
	} else if strings.Contains(statusStr, "status: running") {
		// Try to get more detail from progress markers
		return a.getDetailedProgress(sshKeyPath, username, ip)
	} else if strings.Contains(statusStr, "status: error") {
		return "Setup error detected"
	}

	return "Initializing..."
}

// getDetailedProgress gets detailed progress from setup log
func (a *App) getDetailedProgress(sshKeyPath, username, ip string) string {
	ctx, cancel := context.WithTimeout(a.ctx, 5*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "ssh",
		"-o", "ConnectTimeout=3",
		"-o", "StrictHostKeyChecking=no",
		"-o", "UserKnownHostsFile=/dev/null",
		"-o", "LogLevel=ERROR",
		"-i", sshKeyPath,
		fmt.Sprintf("%s@%s", username, ip),
		"tail -5 /var/log/cws-setup.log 2>/dev/null | grep CWS-PROGRESS | tail -1 || echo ''",
	)

	output, err := cmd.Output()
	if err != nil || len(output) == 0 {
		return "Installing packages"
	}

	// Parse progress marker
	// Format: [CWS-PROGRESS] STAGE:stage-name:status
	line := strings.TrimSpace(string(output))
	if strings.Contains(line, "STAGE:") {
		parts := strings.Split(line, "STAGE:")
		if len(parts) > 1 {
			stageParts := strings.Split(parts[1], ":")
			if len(stageParts) >= 2 {
				stageName := stageParts[0]
				stageStatus := stageParts[1]

				// Human-readable stage names
				stageNames := map[string]string{
					"init":            "Initializing system",
					"system-packages": "Installing system packages",
					"conda-packages":  "Installing conda packages",
					"pip-packages":    "Installing pip packages",
					"service-config":  "Configuring services",
					"ready":           "Starting services",
				}

				displayName := stageNames[stageName]
				if displayName == "" {
					displayName = stageName
				}

				if stageStatus == "COMPLETE" {
					return fmt.Sprintf("✅ %s", displayName)
				} else {
					return fmt.Sprintf("🔄 %s", displayName)
				}
			}
		}
	}

	return "Installing packages"
}

// basicSetupMonitoring provides basic time-based monitoring when SSH not available
func (a *App) basicSetupMonitoring(instance *types.Instance) error {
	fmt.Printf("📦 Monitoring setup progress (estimated 5-8 minutes)...\n\n")

	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()

	startTime := time.Now()
	minWaitTime := 5 * time.Minute

	for {
		select {
		case <-a.ctx.Done():
			return a.ctx.Err()
		case <-ticker.C:
			elapsed := time.Since(startTime)

			if elapsed < 2*time.Minute {
				fmt.Printf("🔧 System initialization (%s)\n", elapsed.Round(time.Second))
			} else if elapsed < 5*time.Minute {
				fmt.Printf("📦 Installing packages (%s)\n", elapsed.Round(time.Second))
			} else if elapsed < 7*time.Minute {
				fmt.Printf("⚙️  Configuring services (%s)\n", elapsed.Round(time.Second))
			} else {
				fmt.Printf("🔧 Finalizing setup (%s)\n", elapsed.Round(time.Second))
			}

			// After minimum wait, try to connect
			if elapsed > minWaitTime {
				_, err := a.apiClient.ConnectInstance(a.ctx, instance.Name)
				if err == nil {
					fmt.Printf("\n✅ Setup complete!\n")
					fmt.Printf("⏱️  Total time: %s\n", elapsed.Round(time.Second))
					return nil
				}
			}

			// Timeout after 15 minutes
			if elapsed > 15*time.Minute {
				fmt.Printf("\n⚠️  Setup taking longer than expected\n")
				return nil
			}
		}
	}
}
