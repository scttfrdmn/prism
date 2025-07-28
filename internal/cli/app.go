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
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"
	"github.com/scttfrdmn/cloudworkstation/pkg/api"
	"github.com/scttfrdmn/cloudworkstation/pkg/api/client"
	"github.com/scttfrdmn/cloudworkstation/pkg/profile"
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
		return fmt.Errorf("usage: cws launch <template> <name> [options]\n  options: --size XS|S|M|L|XL --volume <name> --storage <size> --with conda|apt|dnf|ami --dry-run")
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

	return nil
}

// List handles the list command
func (a *App) List(_ []string) error {
	// Check daemon is running
	if err := a.apiClient.Ping(a.ctx); err != nil {
		return fmt.Errorf("daemon not running. Start with: cws daemon start")
	}

	response, err := a.apiClient.ListInstances(a.ctx)
	if err != nil {
		return fmt.Errorf("failed to list instances: %w", err)
	}

	if len(response.Instances) == 0 {
		fmt.Println("No workstations found. Launch one with: cws launch <template> <n>")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 8, 2, ' ', 0)
	fmt.Fprintln(w, "NAME\tTEMPLATE\tSTATE\tPUBLIC IP\tCOST/DAY\tLAUNCHED")

	for _, instance := range response.Instances {
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t$%.2f\t%s\n",
			instance.Name,
			instance.Template,
			strings.ToUpper(instance.State),
			instance.PublicIP,
			instance.EstimatedDailyCost,
			instance.LaunchTime.Format("2006-01-02 15:04"),
		)
	}

	fmt.Fprintf(w, "\nTotal daily cost (running instances): $%.2f\n", response.TotalCost)
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
	// Check if this is a validation subcommand
	if len(args) > 0 && args[0] == "validate" {
		return a.validateTemplates(args[1:])
	}
	
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

// Note: AMI command is implemented in internal/cli/ami.go