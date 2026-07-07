// Package cli implements Prism's command-line interface application.
//
// This package provides the CLI application logic for the Prism client.
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
	"bufio"
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
	"github.com/scttfrdmn/prism/pkg/aws"
	"github.com/scttfrdmn/prism/pkg/pricing"
	"github.com/scttfrdmn/prism/pkg/profile"
	"github.com/scttfrdmn/prism/pkg/project"
	"github.com/scttfrdmn/prism/pkg/templates"
	"github.com/scttfrdmn/prism/pkg/types"
)

// App represents the CLI application
type App struct {
	version   string
	apiClient client.PrismAPI
	ctx       context.Context // Context for AWS operations

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

	// Check environment variables if profile/region not set or is "default"
	// This ensures AWS_PROFILE and AWS_REGION environment variables are respected
	if awsProfile == "" || awsProfile == "default" {
		if envProfile := os.Getenv("AWS_PROFILE"); envProfile != "" {
			awsProfile = envProfile
		}
	}
	if awsRegion == "" {
		if envRegion := os.Getenv("AWS_REGION"); envRegion != "" {
			awsRegion = envRegion
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
		fmt.Printf("   • Try manual start: prism admin daemon start\n")
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

	return app
}

// Launch handles the launch command
func (a *App) Launch(args []string) error {
	if len(args) < 2 {
		return NewUsageError("prism workspace launch <template> <name>", "prism workspace launch python-ml my-workstation")
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

	// Show cost preview and prompt for confirmation (unless quiet/dry-run/auto-yes)
	if !req.Quiet && !req.DryRun && !req.AutoYes {
		confirmed, err := a.showCostPreview(&req)
		if err != nil {
			// Non-fatal: cost preview failed (e.g. unknown template), skip and proceed
			if !strings.Contains(err.Error(), "not found") {
				fmt.Printf("⚠️  Could not calculate cost estimate: %v\n\n", err)
			}
		} else if !confirmed {
			fmt.Println("Launch cancelled.")
			return nil
		}
	}

	// Show immediate feedback with animated spinner (unless quiet mode)
	var spinner *Spinner
	if !req.Quiet {
		spinner = NewSpinner(fmt.Sprintf("Launching workspace '%s' from template '%s'", req.Name, req.Template))
		spinner.Start()
	}

	response, err := a.apiClient.LaunchInstance(a.ctx, req)

	if err != nil {
		if spinner != nil {
			spinner.Stop()
		}
		return WrapAPIError("launch workspace "+req.Name, err)
	}

	// Approval required (#495): daemon returned HTTP 202 instead of launching
	if response.ApprovalPending {
		if spinner != nil {
			spinner.Stop()
		}
		fmt.Printf("✋ Approval required. Request created: %s\n", response.ApprovalRequestID)
		fmt.Printf("   Status:  pending\n")
		fmt.Printf("\nYour PI or admin can approve with:\n")
		fmt.Printf("   prism approval approve %s\n", response.ApprovalRequestID)
		fmt.Printf("\nOnce approved, launch with:\n")
		fmt.Printf("   prism workspace launch %s %s --approval %s\n", req.Template, req.Name, response.ApprovalRequestID)
		return nil
	}

	if spinner != nil {
		spinner.StopWithMessage(fmt.Sprintf("✅ %s", response.Message))
	}

	// Show project information if launched in a project (unless quiet mode)
	if !req.Quiet && req.ProjectID != "" {
		fmt.Printf("📁 Project: %s\n", req.ProjectID)
		fmt.Printf("🏷️  Workspace will be tracked under project budget\n")
	}

	// Skip all monitoring if --no-progress is specified
	if req.NoProgress {
		if !req.Quiet {
			fmt.Printf("\n💡 Setup is running in the background. To check progress:\n")
			fmt.Printf("   prism workspace connect %s\n", req.Name)
			fmt.Printf("   Then run: cloud-init status\n")
		}
		return nil
	}

	// Monitor launch progress if needed
	if a.shouldMonitorLaunch(&req) {
		if !req.Quiet {
			fmt.Println()
		}
		// Pass req.Wait to disable arbitrary timeouts when --wait is explicitly specified
		return a.monitorLaunchProgress(req.Name, req.Template, req.Wait)
	}

	// Instance launched but monitoring was skipped (e.g. AMI-based template).
	// Packages are pre-installed on AMI templates, but a quick cloud-init pass still
	// runs to configure SSH keys and users — let the user know how to verify readiness.
	if !req.Quiet {
		fmt.Printf("\n💡 If services seem unavailable after connecting, setup may still be\n")
		fmt.Printf("   finishing. Check with: prism workspace connect %s\n", req.Name)
		fmt.Printf("   Then run: cloud-init status\n")
	}
	return nil
}

// showCostPreview displays an estimated cost table and prompts the user to confirm.
// Returns (true, nil) if user confirms or if the terminal is not interactive.
// Returns (false, nil) if user declines.
// Returns (false, err) if template info could not be fetched.
func (a *App) showCostPreview(req *types.LaunchRequest) (bool, error) {
	// Resolve instance type from size or template defaults
	instanceType := a.resolveInstanceTypeForPreview(req)

	// Fetch root volume size and name from template
	rootVolumeGB := 20 // default
	templateDisplayName := req.Template
	templateInfo, err := a.apiClient.GetTemplate(a.ctx, req.Template)
	if err != nil {
		return false, err
	}
	if templateInfo.RootVolumeGB > 0 {
		rootVolumeGB = templateInfo.RootVolumeGB
	}
	if templateInfo.Name != "" {
		templateDisplayName = templateInfo.Name
	}

	// Get hourly rate (with institutional discounts if configured)
	pricingConfig, _ := pricing.LoadInstitutionalPricing()
	pricingCalculator := pricing.NewCalculator(pricingConfig)

	costCalc := &project.CostCalculator{}
	listPrice := costCalc.GetInstanceHourlyRate(instanceType)
	result := pricingCalculator.CalculateInstanceCost(instanceType, listPrice, "")

	// Storage cost: gp3 at $0.08/GB/month
	storageCostMonthly := float64(rootVolumeGB) * 0.08
	storageCostHourly := storageCostMonthly / (24 * 30)

	totalHourly := result.DiscountedPrice + storageCostHourly
	totalDaily := totalHourly * 24
	totalMonthly := totalHourly * 24 * 30

	// Get instance description
	instanceDesc := instanceType
	if cpuMem, ok := instanceDescriptions[instanceType]; ok {
		instanceDesc = fmt.Sprintf("%s (%s)", instanceType, cpuMem)
	}

	// Print preview box
	fmt.Printf("\n  Cost Estimate\n")
	fmt.Printf("  %s\n", strings.Repeat("─", 44))
	fmt.Printf("  Template:       %s\n", templateDisplayName)
	fmt.Printf("  Instance:       %s\n", instanceDesc)
	fmt.Printf("  Storage:        %d GB gp3\n", rootVolumeGB)
	if req.Spot {
		fmt.Printf("  Pricing:        Spot (up to 90%% savings)\n")
	}
	fmt.Printf("  %s\n", strings.Repeat("─", 44))
	fmt.Printf("  Hourly cost:    $%.4f\n", totalHourly)
	fmt.Printf("  Daily cost:     $%.2f\n", totalDaily)
	fmt.Printf("  Monthly est.:   $%.2f\n", totalMonthly)
	if pricingConfig != nil && pricingConfig.Institution != "Default" && result.TotalDiscount > 0 {
		fmt.Printf("  Discount:       %.0f%% (%s)\n", result.TotalDiscount*100, pricingConfig.Institution)
	}
	fmt.Printf("  %s\n\n", strings.Repeat("─", 44))

	// In test mode, auto-confirm
	if a.testMode {
		return true, nil
	}

	// Check if stdin is a terminal; if not (piped/scripted), auto-confirm
	fi, err2 := os.Stdin.Stat()
	if err2 != nil || (fi.Mode()&os.ModeCharDevice) == 0 {
		return true, nil
	}

	fmt.Print("  Continue with launch? [Y/n]: ")
	reader := bufio.NewReader(os.Stdin)
	answer, _ := reader.ReadString('\n')
	answer = strings.TrimSpace(strings.ToLower(answer))

	fmt.Println()
	return answer == "" || answer == "y" || answer == "yes", nil
}

// instanceDescriptions provides human-readable descriptions for common instance types.
var instanceDescriptions = map[string]string{
	"t3.micro":     "1 vCPU, 1 GB RAM",
	"t3.small":     "2 vCPU, 2 GB RAM",
	"t3.medium":    "2 vCPU, 4 GB RAM",
	"t3.large":     "2 vCPU, 8 GB RAM",
	"t3.xlarge":    "4 vCPU, 16 GB RAM",
	"t3.2xlarge":   "8 vCPU, 32 GB RAM",
	"t3a.medium":   "2 vCPU, 4 GB RAM",
	"t3a.large":    "2 vCPU, 8 GB RAM",
	"c5.large":     "2 vCPU, 4 GB RAM",
	"c5.xlarge":    "4 vCPU, 8 GB RAM",
	"c5.2xlarge":   "8 vCPU, 16 GB RAM",
	"r5.large":     "2 vCPU, 16 GB RAM",
	"r5.xlarge":    "4 vCPU, 32 GB RAM",
	"r5.2xlarge":   "8 vCPU, 64 GB RAM",
	"g4dn.xlarge":  "4 vCPU, 16 GB RAM, 1x T4 GPU",
	"g4dn.2xlarge": "8 vCPU, 32 GB RAM, 1x T4 GPU",
	"p3.2xlarge":   "8 vCPU, 61 GB RAM, 1x V100 GPU",
}

// resolveInstanceTypeForPreview returns the instance type that will be used for launch,
// mirroring the same logic as getInstanceTypeForLaunch in pkg/daemon/instance_handlers.go.
func (a *App) resolveInstanceTypeForPreview(req *types.LaunchRequest) string {
	sizeMap := map[string]string{
		"XS": "t3.micro",
		"S":  "t3.small",
		"M":  "t3.medium",
		"L":  "t3.large",
		"XL": "t3.xlarge",
	}
	if req.Size != "" {
		if t, ok := sizeMap[strings.ToUpper(req.Size)]; ok {
			return t
		}
	}
	templateInfo, err := templates.GetTemplateInfo(req.Template)
	if err == nil && templateInfo.InstanceDefaults.Type != "" {
		return templateInfo.InstanceDefaults.Type
	}
	return "t3.micro"
}

// shouldMonitorLaunch returns true if launch progress should be monitored.
// It always returns true when --wait is set, and also when the template has no
// pre-built AMI (package installation templates take 5-10 minutes).
func (a *App) shouldMonitorLaunch(req *types.LaunchRequest) bool {
	if req.Wait {
		return true
	}
	templateInfo, err := a.apiClient.GetTemplate(a.ctx, req.Template)
	if err == nil && len(templateInfo.AMI) == 0 {
		if !req.Quiet {
			fmt.Printf("\n💡 Package installation will take 5-10 minutes. Monitoring progress...\n")
			fmt.Printf("   (Use Ctrl+C to return to prompt - workspace will continue setup)\n")
		}
		return true
	}
	return false
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
func (a *App) monitorLaunchProgress(instanceName, templateName string, wait bool) error {
	// Get template information for enhanced progress reporting
	template, err := a.apiClient.GetTemplate(a.ctx, templateName)
	if err != nil {
		fmt.Printf("%s\n", FormatWarningMessage("Template info", "Could not get template info, using basic progress"))
	}

	// Create enhanced progress reporter
	progressReporter := NewProgressReporter(instanceName, templateName, template)
	progressReporter.ShowHeader()

	// Monitor launch with enhanced progress reporting
	// Pass wait flag to control timeout behavior (Issue #282)
	return a.monitorLaunchWithEnhancedProgress(progressReporter, template, wait)
}

// monitorLaunchWithEnhancedProgress monitors launch with enhanced progress reporting
func (a *App) monitorLaunchWithEnhancedProgress(reporter *ProgressReporter, template *types.Template, wait bool) error {
	startTime := time.Now()
	// Issue #282: Remove arbitrary timeout when --wait is explicitly specified
	// Keep timeout for backward compatibility when auto-monitoring package templates
	maxDuration := 20 * time.Minute // Maximum monitoring time (only used when wait=false)

	for {
		elapsed := time.Since(startTime)

		// Check for timeout (only when --wait not specified, for backward compatibility)
		if !wait && elapsed > maxDuration {
			fmt.Printf("⚠️  Launch monitoring timeout (%s). Workspace may still be setting up.\n",
				reporter.FormatDuration(maxDuration))
			fmt.Printf("💡 Check status with: prism workspace list\n")
			fmt.Printf("💡 Try connecting: prism workspace connect %s\n", reporter.instanceName)
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
				fmt.Printf("💡 Workspace may still be launching. Check with: prism workspace list\n")
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
			return a.monitorSetupProgress(instance, wait)

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
			fmt.Printf("🔗 Connect: prism workspace connect %s\n", instanceName)
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

// listFlags holds the parsed flags for the List command.
type listFlags struct {
	projectFilter string
	detailed      bool
	refresh       bool
}

// parseListFlags parses optional flag arguments for the list command.
func parseListFlags(args []string) listFlags {
	var flags listFlags
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "--project" && i+1 < len(args):
			flags.projectFilter = args[i+1]
			i++
		case arg == "--detailed" || arg == "-d":
			flags.detailed = true
		case arg == "--refresh" || arg == "-r":
			flags.refresh = true
		}
	}
	return flags
}

// printInstanceRow writes a single formatted instance row to the tabwriter.
func printInstanceRow(w *tabwriter.Writer, instance types.Instance, detailed bool) {
	projectInfo := "-"
	if instance.ProjectID != "" {
		projectInfo = instance.ProjectID
	}
	typeIndicator := "OD"
	if instance.InstanceLifecycle == "spot" {
		typeIndicator = "SP"
	}
	currentCost := fmt.Sprintf("$%.4f", instance.CurrentSpend)
	effectiveRate := fmt.Sprintf("$%.4f", instance.EffectiveRate)

	if instance.State == "stopped" || instance.State == "terminated" || instance.State == "stopping" {
		numEBSVolumes := len(instance.AttachedEBSVolumes)
		if numEBSVolumes == 0 {
			numEBSVolumes = 1
		}
		effectiveRate = fmt.Sprintf("$%.4f", 0.005*float64(numEBSVolumes))
	}

	if detailed {
		region := instance.Region
		if region == "" {
			region = "-"
		}
		az := instance.AvailabilityZone
		if az == "" {
			az = "-"
		}
		_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
			instance.Name, instance.Template, strings.ToUpper(instance.State),
			typeIndicator, region, az, instance.PublicIP, projectInfo,
			currentCost, effectiveRate, instance.LaunchTime.Format(ShortDateFormat))
	} else {
		_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
			instance.Name, instance.Template, strings.ToUpper(instance.State),
			typeIndicator, instance.PublicIP, projectInfo,
			currentCost, effectiveRate, instance.LaunchTime.Format(ShortDateFormat))
	}
}

// printRunningCostSummary prints an aggregate cost summary for running instances.
func printRunningCostSummary(instances []types.Instance) {
	running, totalCurrent, totalEffective := 0, 0.0, 0.0
	for _, inst := range instances {
		if inst.State == "running" {
			running++
			totalCurrent += inst.CurrentSpend
			totalEffective += inst.EffectiveRate
		}
	}
	if running > 0 {
		fmt.Printf("💰 Cost Summary:\n")
		fmt.Printf("   Running workspaces: %d\n", running)
		fmt.Printf("   Total accumulated:  $%.4f (since launch)\n", totalCurrent)
		fmt.Printf("   Effective rate:     $%.4f/hr (actual usage)\n", totalEffective)
		fmt.Printf("   Estimated daily:    $%.2f (at current rate)\n", totalEffective*24)
		fmt.Printf("\n💡 Tip: Use 'prism workspace cost' for detailed cost breakdown with savings analysis\n")
	}
}

// List handles the list command with optional project filtering
func (a *App) List(args []string) error {
	flags := parseListFlags(args)

	// Ensure daemon is running (auto-start if needed)
	if err := a.ensureDaemonRunning(); err != nil {
		return err
	}

	response, err := a.apiClient.ListInstancesWithRefresh(a.ctx, flags.refresh)
	if err != nil {
		return WrapAPIError("list workspaces", err)
	}

	// Filter instances by project if specified
	var filteredInstances []types.Instance
	if flags.projectFilter != "" {
		for _, instance := range response.Instances {
			if instance.ProjectID == flags.projectFilter {
				filteredInstances = append(filteredInstances, instance)
			}
		}
	} else {
		filteredInstances = response.Instances
	}

	if len(filteredInstances) == 0 {
		if flags.projectFilter != "" {
			fmt.Printf(NoInstancesFoundProjectMessage+"\n", flags.projectFilter, flags.projectFilter)
		} else {
			fmt.Println(NoInstancesFoundMessage)
		}
		return nil
	}

	if flags.projectFilter != "" {
		fmt.Printf("Workstations in project '%s':\n\n", flags.projectFilter)
	}

	w := tabwriter.NewWriter(os.Stdout, TabWriterMinWidth, TabWriterTabWidth, TabWriterPadding, TabWriterPadChar, TabWriterFlags)
	if flags.detailed {
		_, _ = fmt.Fprintln(w, "NAME\tTEMPLATE\tSTATE\tTYPE\tREGION\tAZ\tPUBLIC IP\tPROJECT\tTOTAL $\tEFF $/HR\tLAUNCHED")
	} else {
		_, _ = fmt.Fprintln(w, "NAME\tTEMPLATE\tSTATE\tTYPE\tPUBLIC IP\tPROJECT\tTOTAL $\tEFF $/HR\tLAUNCHED")
	}
	for _, instance := range filteredInstances {
		printInstanceRow(w, instance, flags.detailed)
	}
	_ = w.Flush()

	fmt.Println()
	printRunningCostSummary(filteredInstances)

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

	fmt.Printf("\n💡 Tip: Use 'prism workspace list' for a clean workspace overview without cost data\n")

	return nil
}

// ListBilledCost shows the AWS-billed ("billed so far") cost for workspaces,
// read from AWS Cost Explorer, next to prism's local estimate (CurrentSpend).
// The estimate is a model accumulated by the daemon; the billed figure is the
// metered amount AWS has charged. When name is non-empty it scopes to one
// workspace; billedOnly drops the estimate and delta columns.
func (a *App) ListBilledCost(name, projectFilter string, billedOnly bool) error {
	if err := a.ensureDaemonRunning(); err != nil {
		return err
	}

	response, err := a.apiClient.ListInstances(a.ctx)
	if err != nil {
		return WrapAPIError("list workspaces for cost analysis", err)
	}

	var instances []types.Instance
	for _, inst := range response.Instances {
		if name != "" && inst.Name != name {
			continue
		}
		if projectFilter != "" && inst.ProjectID != projectFilter {
			continue
		}
		instances = append(instances, inst)
	}

	if len(instances) == 0 {
		switch {
		case name != "":
			fmt.Printf("Workspace '%s' not found.\n", name)
		case projectFilter != "":
			fmt.Printf("No workstations found in project '%s'.\n", projectFilter)
		default:
			fmt.Println("No workstations found.")
		}
		return nil
	}

	fmt.Println("💰 Prism Billed Cost (AWS Cost Explorer)")
	fmt.Println()

	w := tabwriter.NewWriter(os.Stdout, TabWriterMinWidth, TabWriterTabWidth, TabWriterPadding, TabWriterPadChar, TabWriterFlags)
	if billedOnly {
		_, _ = fmt.Fprintln(w, "INSTANCE\tSTATE\tBILLED SO FAR")
	} else {
		_, _ = fmt.Fprintln(w, "INSTANCE\tSTATE\tBILLED SO FAR\tPRISM EST\tDELTA")
	}

	var notes []string
	var sourceLine string
	var anyTagInactive, anyEstimated bool
	for _, inst := range instances {
		billed, err := a.apiClient.GetBilledCost(a.ctx, inst.Name)
		if err != nil {
			if billedOnly {
				_, _ = fmt.Fprintf(w, "%s\t%s\t%s\n", inst.Name, strings.ToUpper(inst.State), "error")
			} else {
				_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t$%.4f\t%s\n",
					inst.Name, strings.ToUpper(inst.State), "error", inst.CurrentSpend, "n/a")
			}
			notes = append(notes, fmt.Sprintf("%s: %v", inst.Name, err))
			continue
		}

		if sourceLine == "" {
			sourceLine = billed.Source
		}
		marker := ""
		if !billed.TagActive {
			marker += "*"
			anyTagInactive = true
		}
		if billed.Estimated {
			marker += "~"
			anyEstimated = true
		}

		if billedOnly {
			_, _ = fmt.Fprintf(w, "%s\t%s\t$%.4f%s\n",
				inst.Name, strings.ToUpper(inst.State), billed.BilledTotal, marker)
		} else {
			// DELTA = prism estimate minus AWS billed; positive means prism over-counts.
			delta := inst.CurrentSpend - billed.BilledTotal
			_, _ = fmt.Fprintf(w, "%s\t%s\t$%.4f%s\t$%.4f\t%+.4f\n",
				inst.Name, strings.ToUpper(inst.State), billed.BilledTotal, marker, inst.CurrentSpend, delta)
		}

		if billed.Note != "" {
			notes = append(notes, fmt.Sprintf("%s: %s", inst.Name, billed.Note))
		}
	}
	_ = w.Flush()

	fmt.Println()
	if sourceLine != "" {
		fmt.Printf("Source: %s\n", sourceLine)
	}
	if !billedOnly {
		fmt.Println("DELTA = prism estimate minus AWS billed (positive means prism over-counts).")
	}
	if anyTagInactive {
		fmt.Println("* region fallback: per-instance cost-allocation tag not active.")
	}
	if anyEstimated {
		fmt.Println("~ includes a current period AWS has not yet finalized.")
	}
	for _, n := range notes {
		fmt.Printf("  - %s\n", n)
	}

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

// waitForInstanceState polls instance status until target state is reached
// No arbitrary timeouts - only network timeouts per API call (Issue #282)
func (a *App) waitForInstanceState(name string, targetStates []string, operation string) error {
	fmt.Printf("⏳ Waiting for %s to complete...\n", operation)
	startTime := time.Now()
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-a.ctx.Done():
			return fmt.Errorf("operation cancelled")
		case <-ticker.C:
			instance, err := a.apiClient.GetInstance(a.ctx, name)
			if err != nil {
				// For delete operations, "not found" means success
				if operation == "deletion" && (strings.Contains(err.Error(), "not found") || strings.Contains(err.Error(), "does not exist")) {
					elapsed := time.Since(startTime)
					fmt.Printf("✅ %s complete (took %s)\n", strings.Title(operation), elapsed.Round(time.Second))
					return nil
				}
				// For other operations, retry on error
				fmt.Printf("⏳ Checking status... (%s)\n", time.Since(startTime).Round(time.Second))
				continue
			}

			// Check if we've reached target state
			for _, targetState := range targetStates {
				if instance.State == targetState {
					elapsed := time.Since(startTime)
					fmt.Printf("✅ %s complete - instance is %s (took %s)\n", strings.Title(operation), instance.State, elapsed.Round(time.Second))
					return nil
				}
			}

			// Show progress
			fmt.Printf("⏳ Instance state: %s (%s)\n", instance.State, time.Since(startTime).Round(time.Second))
		}
	}
}

// StopWithWait stops an instance with optional blocking
func (a *App) StopWithWait(name string, wait bool) error {
	// Initiate the stop operation
	if err := a.instanceCommands.Stop([]string{name}); err != nil {
		return err
	}

	// If --wait flag specified, poll until stopped (no arbitrary timeout)
	if wait {
		return a.waitForInstanceState(name, []string{"stopped"}, "stop")
	}

	return nil
}

// StartWithWait starts an instance with optional blocking
func (a *App) StartWithWait(name string, wait bool) error {
	// Initiate the start operation
	if err := a.instanceCommands.Start([]string{name}); err != nil {
		return err
	}

	// If --wait flag specified, poll until running (no arbitrary timeout)
	if wait {
		return a.waitForInstanceState(name, []string{"running"}, "start")
	}

	return nil
}

// DeleteWithWait deletes an instance with optional blocking
func (a *App) DeleteWithWait(name string, wait bool) error {
	// Initiate the delete operation
	if err := a.instanceCommands.Delete([]string{name}); err != nil {
		return err
	}

	// If --wait flag specified, poll until terminated/not found (no arbitrary timeout)
	if wait {
		return a.waitForInstanceState(name, []string{"terminated"}, "deletion")
	}

	return nil
}

// HibernateWithWait hibernates an instance with optional blocking
func (a *App) HibernateWithWait(name string, wait bool) error {
	// Initiate the hibernate operation
	if err := a.instanceCommands.Hibernate([]string{name}); err != nil {
		return err
	}

	// If --wait flag specified, poll until stopped (no arbitrary timeout)
	if wait {
		return a.waitForInstanceState(name, []string{"stopped"}, "hibernation")
	}

	return nil
}

// ResumeWithWait resumes an instance with optional blocking
func (a *App) ResumeWithWait(name string, wait bool) error {
	// Initiate the resume operation
	if err := a.instanceCommands.Resume([]string{name}); err != nil {
		return err
	}

	// If --wait flag specified, poll until running (no arbitrary timeout)
	if wait {
		return a.waitForInstanceState(name, []string{"running"}, "resume")
	}

	return nil
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
	fmt.Printf("\n🛠️  To build AMIs: prism ami build <template-name>\n")

	return nil
}

// Note: AMI command is implemented in internal/cli/ami.go

// Note: Marketplace command is implemented in internal/cli/marketplace.go

// Project command implementation
func (a *App) Project(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: prism project <action> [args]")
	}

	action := args[0]
	projectArgs := args[1:]

	// Check daemon is running
	if err := a.apiClient.Ping(a.ctx); err != nil {
		return fmt.Errorf("daemon not running. Start with: prism admin daemon start")
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
	case "invite":
		return a.projectInvite(projectArgs)
	case "invitations":
		return a.projectInvitations(projectArgs)
	case "delete":
		return a.projectDelete(projectArgs)
	default:
		return fmt.Errorf("unknown project action: %s", action)
	}
}

func (a *App) projectCreate(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: prism project create <name> [options]")
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
			req.Budget = &project.CreateProjectBudgetRequest{
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
		fmt.Println("No projects found. Create one with: prism project create <name>")
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
		return fmt.Errorf("usage: prism project info <name>")
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

	// Instance information - filter all instances by project ID
	fmt.Printf("\n🖥️ Instances:\n")
	instanceList, err := a.apiClient.ListInstances(a.ctx)
	if err == nil {
		var projectInstances []types.Instance
		for _, inst := range instanceList.Instances {
			if inst.ProjectID == project.ID {
				projectInstances = append(projectInstances, inst)
			}
		}
		if len(projectInstances) == 0 {
			fmt.Printf("   None\n")
		} else {
			for _, inst := range projectInstances {
				fmt.Printf("   %s (%s) — %s\n", inst.Name, inst.ID, inst.State)
			}
		}
	} else {
		fmt.Printf("   (Unable to list instances: %v)\n", err)
	}

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
		return fmt.Errorf("usage: prism project budget <action> <project> [options]")
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
	case "prevent-launches":
		return a.projectBudgetPreventLaunches(remainingArgs)
	case "allow-launches":
		return a.projectBudgetAllowLaunches(remainingArgs)
	default:
		// Legacy support: if first arg is not a subcommand, assume it's a project name for status
		return a.projectBudgetStatus(args)
	}
}

func (a *App) projectBudgetStatus(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: prism project budget status <project>")
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
		fmt.Printf("   💡 Enable cost tracking with: prism project budget set %s <amount>\n", projectName)
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
		return fmt.Errorf("usage: prism project budget set <project> <amount> [options]")
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
		return fmt.Errorf("usage: prism project budget update <project> [options]")
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
		return fmt.Errorf("usage: prism project budget disable <project>")
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
		return fmt.Errorf("usage: prism project budget history <project> [--days N]")
	}

	projectName := args[0]

	// Parse optional --days flag (default 30)
	days := 30
	for i := 1; i < len(args)-1; i++ {
		if args[i] == "--days" {
			if n, err := strconv.Atoi(args[i+1]); err == nil && n > 0 {
				days = n
			}
		}
	}

	// Resolve project name → ID
	projectResponse, err := a.apiClient.ListProjects(a.ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to list projects: %w", err)
	}

	var projectID string
	for _, proj := range projectResponse.Projects {
		if proj.Name == projectName {
			projectID = proj.ID
			break
		}
	}
	if projectID == "" {
		return fmt.Errorf("project not found: %s", projectName)
	}

	history, err := a.apiClient.GetProjectBudgetHistory(a.ctx, projectID, days)
	if err != nil {
		return fmt.Errorf("failed to get budget history: %w", err)
	}

	fmt.Printf("📊 Budget History for '%s' (last %d days):\n\n", projectName, days)

	if len(history) == 0 {
		fmt.Printf("   No cost data recorded yet.\n")
		fmt.Printf("   💡 Cost data is recorded as instances run.\n")
		return nil
	}

	// Show each data point as a simple bar chart using ASCII
	var maxCost float64
	for _, c := range history {
		if c > maxCost {
			maxCost = c
		}
	}

	for i, cost := range history {
		barLen := 0
		if maxCost > 0 {
			barLen = int((cost / maxCost) * 30)
		}
		bar := strings.Repeat("█", barLen)
		fmt.Printf("   [%3d] $%7.2f  %s\n", i+1, cost, bar)
	}

	fmt.Printf("\n   Total data points: %d\n", len(history))

	return nil
}

func (a *App) projectBudgetPreventLaunches(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: prism project budget prevent-launches <project>")
	}

	projectName := args[0]

	// Call API to prevent launches
	response, err := a.apiClient.PreventProjectLaunches(a.ctx, projectName)
	if err != nil {
		return fmt.Errorf("failed to prevent launches: %w", err)
	}

	fmt.Printf("🚫 Launch prevention enabled for project '%s'\n", projectName)
	fmt.Printf("   New instance launches are now blocked\n")
	fmt.Printf("   💡 To allow launches again: prism project budget allow-launches %s\n", projectName)

	if message, ok := response["message"].(string); ok {
		fmt.Printf("   %s\n", message)
	}

	return nil
}

func (a *App) projectBudgetAllowLaunches(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: prism project budget allow-launches <project>")
	}

	projectName := args[0]

	// Call API to allow launches
	response, err := a.apiClient.AllowProjectLaunches(a.ctx, projectName)
	if err != nil {
		return fmt.Errorf("failed to allow launches: %w", err)
	}

	fmt.Printf("✅ Launch prevention cleared for project '%s'\n", projectName)
	fmt.Printf("   New instance launches are now allowed\n")

	if message, ok := response["message"].(string); ok {
		fmt.Printf("   %s\n", message)
	}

	return nil
}

func (a *App) projectInstances(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: prism project instances <name>")
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
		fmt.Printf("Launch one with: prism workspace launch <template> <workspace-name> --project %s\n", projectName)
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
		return fmt.Errorf("usage: prism project templates <name>")
	}

	name := args[0]

	// Templates don't have per-project scope yet (#122) — show the global library.
	templateMap, err := a.apiClient.ListTemplates(a.ctx)
	if err != nil {
		return fmt.Errorf("failed to list templates: %w", err)
	}

	fmt.Printf("🏗️ Templates available to project '%s':\n\n", name)
	fmt.Printf("   ℹ️  Templates are shared globally. Project-specific templates are tracked in issue #122.\n\n")

	if len(templateMap) == 0 {
		fmt.Printf("   No templates found.\n")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "   NAME\tCATEGORY\tDESCRIPTION")
	fmt.Fprintln(w, "   ----\t--------\t-----------")
	for tName, t := range templateMap {
		desc := t.Description
		if len(desc) > 60 {
			desc = desc[:57] + "..."
		}
		category := t.Category
		if category == "" {
			category = t.Domain
		}
		fmt.Fprintf(w, "   %s\t%s\t%s\n", tName, category, desc)
	}
	w.Flush()

	return nil
}

func (a *App) projectMembers(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: prism project members <name> [action] [member-email] [role]")
	}

	name := args[0]

	// Handle member management actions
	if len(args) >= 2 {
		action := args[1]
		switch action {
		case "add":
			if len(args) < 4 {
				return fmt.Errorf("usage: prism project members <name> add <email> <role>")
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
				return fmt.Errorf("usage: prism project members <name> remove <email>")
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
		fmt.Printf("Add members with: prism project members %s add <email> <role>\n", name)
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
	fmt.Printf("Add member: prism project members %s add <email> <role>\n", name)
	fmt.Printf("Remove member: prism project members %s remove <email>\n", name)

	return nil
}

func (a *App) projectDelete(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: prism project delete <name>")
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

// GetAWSManager creates an AWS manager using the current profile settings
func (a *App) GetAWSManager() (*aws.Manager, error) {
	// Determine AWS profile and region to use
	awsProfile := a.config.AWS.Profile
	awsRegion := a.config.AWS.Region

	// Check if there's an active Prism profile that overrides config
	if a.profileManager != nil {
		if currentProfile, err := a.profileManager.GetCurrentProfile(); err == nil && currentProfile != nil {
			awsProfile = currentProfile.AWSProfile
			awsRegion = currentProfile.Region
		}
	}

	// Create AWS manager with options
	return aws.NewManager(aws.ManagerOptions{
		Profile: awsProfile,
		Region:  awsRegion,
	})
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
func (a *App) monitorSetupProgress(instance *types.Instance, wait bool) error {
	// Use new progress API (v0.7.2) instead of SSH checks
	fmt.Printf("🔍 Monitoring template installation progress...\n\n")

	// Poll for setup completion using the progress API
	ticker := time.NewTicker(5 * time.Second) // Poll every 5 seconds
	defer ticker.Stop()

	startTime := time.Now()
	lastProgressPercent := float64(-1)
	lastStageIndex := -1

	for {
		select {
		case <-a.ctx.Done():
			return a.ctx.Err()
		case <-ticker.C:
			elapsed := time.Since(startTime)

			// Get progress from API
			progress, err := a.apiClient.GetProgress(a.ctx, instance.Name)
			if err != nil {
				// Progress monitoring might not be available yet
				if elapsed < 2*time.Minute {
					fmt.Printf("⏳ Initializing progress monitoring... (%s)\n", elapsed.Round(time.Second))
					continue
				}
				// After 2 minutes, if still no progress API, fall back to basic monitoring
				fmt.Printf("⚠️  Progress API not available, falling back to basic monitoring\n")
				return a.basicSetupMonitoring(instance, wait)
			}

			// Clear previous output (simple approach - just update on changes)
			if progress.OverallProgress != lastProgressPercent || progress.CurrentStageIndex != lastStageIndex {
				// Show progress bar
				a.displayProgressBar(progress.OverallProgress)

				// Show current stage with details
				fmt.Printf("📦 %s\n", progress.CurrentStage)

				// Show stage list with status
				a.displayStageList(progress.Stages)

				// Show time estimate
				if progress.EstimatedTimeLeft != "" && !progress.IsComplete {
					fmt.Printf("\n⏱️  Elapsed: %s | Remaining: %s\n",
						elapsed.Round(time.Second),
						progress.EstimatedTimeLeft)
				}

				fmt.Println() // Add spacing

				lastProgressPercent = progress.OverallProgress
				lastStageIndex = progress.CurrentStageIndex
			}

			// Check if complete
			if progress.IsComplete || progress.OverallProgress >= 100 {
				fmt.Printf("✅ Setup complete! Workspace ready.\n")
				fmt.Printf("⏱️  Total setup time: %s\n", elapsed.Round(time.Second))
				fmt.Printf("🔗 Connect: prism workspace connect %s\n", instance.Name)
				return nil
			}

			// Issue #282: Timeout only when --wait not specified (for backward compatibility)
			// When --wait is specified, poll indefinitely until complete
			if !wait && elapsed > 30*time.Minute {
				fmt.Printf("\n⚠️  Setup taking longer than expected\n")
				fmt.Printf("💡 Progress: %.1f%% - Workspace may still be configuring\n", progress.OverallProgress)
				fmt.Printf("💡 Check status with: prism workspace list\n")
				fmt.Printf("💡 To verify setup progress after connecting:\n")
				fmt.Printf("   prism workspace connect %s  # then run: cloud-init status\n", instance.Name)
				return nil
			}
		}
	}
}

// displayProgressBar shows a visual progress bar
func (a *App) displayProgressBar(percent float64) {
	barWidth := 30
	filled := int(percent / 100 * float64(barWidth))

	bar := "["
	for i := 0; i < barWidth; i++ {
		if i < filled {
			bar += "█"
		} else if i == filled && percent < 100 {
			bar += "▌"
		} else {
			bar += "░"
		}
	}
	bar += fmt.Sprintf("] %.1f%%", percent)

	fmt.Printf("  %s\n\n", bar)
}

// displayStageList shows the list of stages with status icons
func (a *App) displayStageList(stages []types.ProgressStage) {
	for _, stage := range stages {
		var icon string
		switch stage.Status {
		case "complete":
			icon = "✓"
		case "running":
			icon = "⏳"
		case "error":
			icon = "❌"
		default: // pending
			icon = "⏸"
		}

		// Calculate elapsed time for completed stages
		timeStr := ""
		if stage.Status == "complete" && !stage.EndTime.IsZero() && !stage.StartTime.IsZero() {
			elapsed := stage.EndTime.Sub(stage.StartTime)
			timeStr = fmt.Sprintf(" (%s)", elapsed.Round(time.Second))
		} else if stage.Status == "running" && !stage.StartTime.IsZero() {
			elapsed := time.Since(stage.StartTime)
			timeStr = fmt.Sprintf(" (%s elapsed)", elapsed.Round(time.Second))
		}

		fmt.Printf("  %s %s%s\n", icon, stage.DisplayName, timeStr)
	}
}

// findSSHKey finds the SSH key for a region
func (a *App) findSSHKey(region string) (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	// Standard format: prism-test-{region}-key
	keyPaths := []string{
		filepath.Join(homeDir, ".ssh", fmt.Sprintf("prism-test-%s-key", region)),
		filepath.Join(homeDir, ".prism", "profiles", "test", "ssh", fmt.Sprintf("prism-test-%s-key", region)),
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
		"tail -5 /var/log/prism-setup.log 2>/dev/null | grep PRISM-PROGRESS | tail -1 || echo ''",
	)

	output, err := cmd.Output()
	if err != nil || len(output) == 0 {
		return "Installing packages"
	}

	// Parse progress marker
	// Format: [PRISM-PROGRESS] STAGE:stage-name:status
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
func (a *App) basicSetupMonitoring(instance *types.Instance, wait bool) error {
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

			// Issue #282: Timeout only when --wait not specified (for backward compatibility)
			// When --wait is specified, poll indefinitely until complete
			if !wait && elapsed > 15*time.Minute {
				fmt.Printf("\n⚠️  Setup taking longer than expected\n")
				return nil
			}
		}
	}
}
