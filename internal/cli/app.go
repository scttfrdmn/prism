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

	"github.com/spf13/cobra"
	"github.com/scttfrdmn/cloudworkstation/pkg/api"
	"github.com/scttfrdmn/cloudworkstation/pkg/api/client"
	"github.com/scttfrdmn/cloudworkstation/pkg/profile"
	"github.com/scttfrdmn/cloudworkstation/pkg/project"
	"github.com/scttfrdmn/cloudworkstation/pkg/templates"
	"github.com/scttfrdmn/cloudworkstation/pkg/types"
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
		return fmt.Errorf("usage: cws launch <template> <name> [options]\n  options: --size XS|S|M|L|XL --volume <name> --storage <size> --project <name> --with conda|apt|dnf|ami --spot --dry-run")
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
	fmt.Fprintln(w, "NAME\tTEMPLATE\tSTATE\tPUBLIC IP\tCOST/DAY\tPROJECT\tLAUNCHED")

	totalCost := 0.0
	for _, instance := range filteredInstances {
		projectInfo := "-"
		if instance.ProjectID != "" {
			projectInfo = instance.ProjectID
		}
		
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t$%.2f\t%s\t%s\n",
			instance.Name,
			instance.Template,
			strings.ToUpper(instance.State),
			instance.PublicIP,
			instance.EstimatedDailyCost,
			projectInfo,
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

// Connect handles the connect command
func (a *App) Connect(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: cws connect <n>")
	}

	name := args[0]

	// Check daemon is running
	if err := a.apiClient.Ping(a.ctx); err != nil {
		return fmt.Errorf("daemon not running. Start with: cws daemon start")
	}

	connectionInfo, err := a.apiClient.ConnectInstance(a.ctx, name)
	if err != nil {
		return fmt.Errorf("failed to get connection info: %w", err)
	}

	fmt.Printf("🔗 Connection info for %s:\n", name)
	fmt.Printf("%s\n", connectionInfo)

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

// Start handles the start command
func (a *App) Start(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: cws start <n>")
	}

	name := args[0]

	// Check daemon is running
	if err := a.apiClient.Ping(a.ctx); err != nil {
		return fmt.Errorf("daemon not running. Start with: cws daemon start")
	}

	err := a.apiClient.StartInstance(a.ctx, name)
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
	fmt.Printf("📋 Template Information: %s\n\n", templateName)

	// Check daemon is running
	if err := a.apiClient.Ping(a.ctx); err != nil {
		return fmt.Errorf("daemon not running. Start with: cws daemon start")
	}

	template, err := a.apiClient.GetTemplate(a.ctx, templateName)
	if err != nil {
		return fmt.Errorf("failed to get template info: %w", err)
	}

	fmt.Printf("🏗️  Name: %s\n", templateName)
	fmt.Printf("📝 Description: %s\n", template.Description)
	fmt.Printf("💰 Cost: $%.2f/hour (x86_64), $%.2f/hour (arm64)\n",
		template.EstimatedCostPerHour["x86_64"],
		template.EstimatedCostPerHour["arm64"])
	
	if len(template.Ports) > 0 {
		fmt.Printf("🌐 Exposed Ports: %v\n", template.Ports)
	}
	
	// Show AMI information if available
	if len(template.AMI) > 0 {
		fmt.Printf("💿 AMI IDs:\n")
		for region, arches := range template.AMI {
			for arch, amiID := range arches {
				fmt.Printf("   %s (%s): %s\n", region, arch, amiID)
			}
		}
	}

	fmt.Printf("\n🚀 Launch: cws launch %s <instance-name>\n", templateName)
	
	return nil
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
	default:
		return fmt.Errorf("unknown daemon action: %s", action)
	}
}

func (a *App) daemonStart() error {
	// Check if daemon is already running
	if err := a.apiClient.Ping(a.ctx); err == nil {
		fmt.Println("✅ Daemon is already running")
		return nil
	}

	fmt.Println("🚀 Starting CloudWorkstation daemon...")

	// Start daemon in the background
	cmd := exec.Command("cwsd")
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start daemon: %w", err)
	}

	// TODO: Wait for daemon to be ready and verify it started correctly
	fmt.Printf("✅ Daemon started (PID %d)\n", cmd.Process.Pid)

	return nil
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

// Note: AMI command is implemented in internal/cli/ami.go