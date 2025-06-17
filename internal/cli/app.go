package cli

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"text/tabwriter"

	"github.com/scttfrdmn/cloudworkstation/pkg/api"
	"github.com/scttfrdmn/cloudworkstation/pkg/types"
)

// App represents the CLI application
type App struct {
	version   string
	apiClient api.CloudWorkstationAPI
}

// NewApp creates a new CLI application
func NewApp(version string) *App {
	return &App{
		version:   version,
		apiClient: api.NewClient(""), // Uses default localhost:8080
	}
}

// Launch handles the launch command
func (a *App) Launch(args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("usage: cws launch <template> <name> [options]")
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
		case arg == "--spot":
			req.Spot = true
		case arg == "--dry-run":
			req.DryRun = true
		default:
			return fmt.Errorf("unknown option: %s", arg)
		}
	}

	// Check daemon is running
	if err := a.apiClient.Ping(); err != nil {
		return fmt.Errorf("daemon not running. Start with: cws daemon start")
	}

	response, err := a.apiClient.LaunchInstance(req)
	if err != nil {
		return fmt.Errorf("failed to launch instance: %w", err)
	}

	fmt.Printf("🚀 %s\n", response.Message)
	fmt.Printf("💰 Estimated cost: %s\n", response.EstimatedCost)
	fmt.Printf("🔗 Connect with: %s\n", response.ConnectionInfo)
	
	return nil
}

// List handles the list command
func (a *App) List(args []string) error {
	// Check daemon is running
	if err := a.apiClient.Ping(); err != nil {
		return fmt.Errorf("daemon not running. Start with: cws daemon start")
	}

	response, err := a.apiClient.ListInstances()
	if err != nil {
		return fmt.Errorf("failed to list instances: %w", err)
	}

	if len(response.Instances) == 0 {
		fmt.Println("No workstations found. Launch one with: cws launch <template> <name>")
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
		return fmt.Errorf("usage: cws connect <name>")
	}

	name := args[0]

	// Check daemon is running
	if err := a.apiClient.Ping(); err != nil {
		return fmt.Errorf("daemon not running. Start with: cws daemon start")
	}

	connectionInfo, err := a.apiClient.ConnectInstance(name)
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
		return fmt.Errorf("usage: cws stop <name>")
	}

	name := args[0]

	// Check daemon is running
	if err := a.apiClient.Ping(); err != nil {
		return fmt.Errorf("daemon not running. Start with: cws daemon start")
	}

	err := a.apiClient.StopInstance(name)
	if err != nil {
		return fmt.Errorf("failed to stop instance: %w", err)
	}

	fmt.Printf("⏹️ Stopping instance %s...\n", name)
	return nil
}

// Start handles the start command
func (a *App) Start(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: cws start <name>")
	}

	name := args[0]

	// Check daemon is running
	if err := a.apiClient.Ping(); err != nil {
		return fmt.Errorf("daemon not running. Start with: cws daemon start")
	}

	err := a.apiClient.StartInstance(name)
	if err != nil {
		return fmt.Errorf("failed to start instance: %w", err)
	}

	fmt.Printf("▶️ Starting instance %s...\n", name)
	return nil
}

// Delete handles the delete command
func (a *App) Delete(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: cws delete <name>")
	}

	name := args[0]

	// Check daemon is running
	if err := a.apiClient.Ping(); err != nil {
		return fmt.Errorf("daemon not running. Start with: cws daemon start")
	}

	err := a.apiClient.DeleteInstance(name)
	if err != nil {
		return fmt.Errorf("failed to delete instance: %w", err)
	}

	fmt.Printf("🗑️ Deleting instance %s...\n", name)
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
	if err := a.apiClient.Ping(); err != nil {
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
		return fmt.Errorf("usage: cws volume create <name> [options]")
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

	volume, err := a.apiClient.CreateVolume(req)
	if err != nil {
		return fmt.Errorf("failed to create volume: %w", err)
	}

	fmt.Printf("📁 Created EFS volume %s (%s)\n", volume.Name, volume.FileSystemId)
	return nil
}

func (a *App) volumeList(args []string) error {
	volumes, err := a.apiClient.ListVolumes()
	if err != nil {
		return fmt.Errorf("failed to list volumes: %w", err)
	}

	if len(volumes) == 0 {
		fmt.Println("No EFS volumes found. Create one with: cws volume create <name>")
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
		return fmt.Errorf("usage: cws volume info <name>")
	}

	name := args[0]
	volume, err := a.apiClient.GetVolume(name)
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
		return fmt.Errorf("usage: cws volume delete <name>")
	}

	name := args[0]
	err := a.apiClient.DeleteVolume(name)
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
	if err := a.apiClient.Ping(); err != nil {
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
		return fmt.Errorf("usage: cws storage create <name> <size> [type]")
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

	volume, err := a.apiClient.CreateStorage(req)
	if err != nil {
		return fmt.Errorf("failed to create storage: %w", err)
	}

	fmt.Printf("💾 Created EBS volume %s (%s) - %d GB %s\n", 
		volume.Name, volume.VolumeID, volume.SizeGB, volume.VolumeType)
	return nil
}

func (a *App) storageList(args []string) error {
	volumes, err := a.apiClient.ListStorage()
	if err != nil {
		return fmt.Errorf("failed to list storage: %w", err)
	}

	if len(volumes) == 0 {
		fmt.Println("No EBS volumes found. Create one with: cws storage create <name> <size>")
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
		return fmt.Errorf("usage: cws storage info <name>")
	}

	name := args[0]
	volume, err := a.apiClient.GetStorage(name)
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

	err := a.apiClient.AttachStorage(volumeName, instanceName)
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

	err := a.apiClient.DetachStorage(volumeName)
	if err != nil {
		return fmt.Errorf("failed to detach storage: %w", err)
	}

	fmt.Printf("🔓 Detaching volume %s...\n", volumeName)
	return nil
}

func (a *App) storageDelete(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: cws storage delete <name>")
	}

	name := args[0]
	err := a.apiClient.DeleteStorage(name)
	if err != nil {
		return fmt.Errorf("failed to delete storage: %w", err)
	}

	fmt.Printf("🗑️ Deleting EBS volume %s...\n", name)
	return nil
}

// Templates handles the templates command
func (a *App) Templates(args []string) error {
	// Check daemon is running
	if err := a.apiClient.Ping(); err != nil {
		return fmt.Errorf("daemon not running. Start with: cws daemon start")
	}

	templates, err := a.apiClient.ListTemplates()
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
	if err := a.apiClient.Ping(); err == nil {
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
	// TODO: Implement graceful daemon shutdown
	fmt.Println("⏹️ Stopping daemon...")
	
	// For now, just inform user how to stop manually
	fmt.Println("Find the daemon process and stop it manually:")
	fmt.Println("  ps aux | grep cwsd")
	fmt.Println("  kill <PID>")
	
	return nil
}

func (a *App) daemonStatus() error {
	// Check if daemon is running
	if err := a.apiClient.Ping(); err != nil {
		fmt.Println("❌ Daemon is not running")
		fmt.Println("Start with: cws daemon start")
		return nil
	}

	status, err := a.apiClient.GetStatus()
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