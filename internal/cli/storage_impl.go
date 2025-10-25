// Package cli - Storage Implementation Layer
//
// ARCHITECTURE NOTE: This file contains the business logic implementation for storage commands.
// The user-facing CLI interface is defined in storage_cobra.go, which delegates to these methods.
//
// This separation follows the Facade/Adapter pattern:
//   - storage_cobra.go: CLI interface (Cobra commands, flag parsing, help text)
//   - storage_impl.go: Business logic (API calls, formatting, error handling)
//
// This architecture allows:
//   - Clean separation of concerns
//   - Reusable business logic (can be called from Cobra, TUI, or tests)
//   - Consistent API interaction patterns across all commands
//
// DO NOT REMOVE THIS FILE - it is actively used by storage_cobra.go and other components.
package cli

import (
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/scttfrdmn/cloudworkstation/pkg/types"
)

// StorageCommands handles all storage management operations (implementation layer)
type StorageCommands struct {
	app *App
}

// NewStorageCommands creates storage commands handler
func NewStorageCommands(app *App) *StorageCommands {
	return &StorageCommands{app: app}
}

// Volume handles volume commands
func (sc *StorageCommands) Volume(args []string) error {
	if len(args) < 1 {
		return NewUsageError("cws volume <action> [args]", "cws volume create my-shared-data")
	}

	action := args[0]
	volumeArgs := args[1:]

	// Ensure daemon is running (auto-start if needed)
	if err := sc.app.ensureDaemonRunning(); err != nil {
		return err
	}

	switch action {
	case "create":
		return sc.volumeCreate(volumeArgs)
	case "list":
		return sc.volumeList(volumeArgs)
	case "info":
		return sc.volumeInfo(volumeArgs)
	case "delete":
		return sc.volumeDelete(volumeArgs)
	case "mount":
		return sc.volumeMount(volumeArgs)
	case "unmount":
		return sc.volumeUnmount(volumeArgs)
	default:
		return NewValidationError("volume action", action, "create, list, info, delete, mount, unmount")
	}
}

func (sc *StorageCommands) volumeCreate(args []string) error {
	if len(args) < 1 {
		return NewUsageError("cws volume create <name> [options]", "cws volume create my-shared-data --performance generalPurpose")
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
			return NewValidationError("volume option", arg, "--performance, --throughput, --region")
		}
	}

	volume, err := sc.app.apiClient.CreateVolume(sc.app.ctx, req)
	if err != nil {
		return WrapAPIError("create shared storage "+req.Name, err)
	}

	fmt.Printf("%s\n", FormatSuccessMessage("Created Shared Storage", volume.Name, fmt.Sprintf("(%s)", volume.FileSystemID)))
	return nil
}

func (sc *StorageCommands) volumeList(_ []string) error {
	volumes, err := sc.app.apiClient.ListVolumes(sc.app.ctx)
	if err != nil {
		return WrapAPIError("list shared storage", err)
	}

	if len(volumes) == 0 {
		fmt.Println("No shared storage volumes found. Create one with 'cws volume create'.")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, TabWriterMinWidth, TabWriterTabWidth, TabWriterPadding, TabWriterPadChar, TabWriterFlags)
	_, _ = fmt.Fprintln(w, "NAME\tFILESYSTEM ID\tSTATE\tSIZE\tCOST/MONTH")

	for _, volume := range volumes {
		var sizeGB float64
		if volume.SizeBytes != nil {
			sizeGB = float64(*volume.SizeBytes) / BytesToGB
		}
		costMonth := sizeGB * volume.EstimatedCostGB
		_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%.1f GB\t$%.2f\n",
			volume.Name,
			volume.FileSystemID,
			strings.ToUpper(volume.State),
			sizeGB,
			costMonth,
		)
	}
	_ = w.Flush()

	return nil
}

func (sc *StorageCommands) volumeInfo(args []string) error {
	if len(args) < 1 {
		return NewUsageError("cws volume info <name>", "cws volume info my-shared-data")
	}

	name := args[0]
	volume, err := sc.app.apiClient.GetVolume(sc.app.ctx, name)
	if err != nil {
		return WrapAPIError("get volume info for "+name, err)
	}

	fmt.Printf("📁 Shared Storage: %s\n", volume.Name)
	fmt.Printf("   Filesystem ID: %s\n", volume.FileSystemID)
	fmt.Printf("   State: %s\n", strings.ToUpper(volume.State))
	fmt.Printf("   Region: %s\n", volume.Region)
	fmt.Printf("   Performance Mode: %s\n", volume.PerformanceMode)
	fmt.Printf("   Throughput Mode: %s\n", volume.ThroughputMode)
	if volume.SizeBytes != nil {
		sizeGB := float64(*volume.SizeBytes) / BytesToGB
		fmt.Printf("   Size: %.1f GB\n", sizeGB)
		fmt.Printf("   Cost: $%.2f/month\n", sizeGB*volume.EstimatedCostGB)
	}
	fmt.Printf("   Created: %s\n", volume.CreationTime.Format(StandardDateFormat))
	fmt.Printf("   AWS Service: %s\n", volume.GetTechnicalType())

	return nil
}

func (sc *StorageCommands) volumeDelete(args []string) error {
	if len(args) < 1 {
		return NewUsageError("cws volume delete <name>", "cws volume delete my-shared-data")
	}

	name := args[0]
	err := sc.app.apiClient.DeleteVolume(sc.app.ctx, name)
	if err != nil {
		return WrapAPIError("delete shared storage "+name, err)
	}

	fmt.Printf("%s\n", FormatProgressMessage("Deleting Shared Storage", name))
	return nil
}

func (sc *StorageCommands) volumeMount(args []string) error {
	if len(args) < 2 {
		return NewUsageError("cws volume mount <volume-name> <workspace-name> [mount-point]", "cws volume mount my-shared-data my-workspace")
	}

	volumeName := args[0]
	instanceName := args[1]

	// Default mount point
	mountPoint := DefaultMountPointPrefix + volumeName
	if len(args) >= 3 {
		mountPoint = args[2]
	}

	err := sc.app.apiClient.MountVolume(sc.app.ctx, volumeName, instanceName, mountPoint)
	if err != nil {
		return WrapAPIError("mount volume "+volumeName+" to "+instanceName, err)
	}

	fmt.Printf("%s\n", FormatProgressMessage("Mounting Shared Storage", fmt.Sprintf("'%s' to '%s' at %s", volumeName, instanceName, mountPoint)))
	return nil
}

func (sc *StorageCommands) volumeUnmount(args []string) error {
	if len(args) < 2 {
		return NewUsageError("cws volume unmount <volume-name> <workspace-name>", "cws volume unmount my-shared-data my-workspace")
	}

	volumeName := args[0]
	instanceName := args[1]

	err := sc.app.apiClient.UnmountVolume(sc.app.ctx, volumeName, instanceName)
	if err != nil {
		return WrapAPIError("unmount volume "+volumeName+" from "+instanceName, err)
	}

	fmt.Printf("%s\n", FormatProgressMessage("Unmounting Shared Storage", fmt.Sprintf("'%s' from '%s'", volumeName, instanceName)))
	return nil
}

// Storage handles storage commands
func (sc *StorageCommands) Storage(args []string) error {
	if len(args) < 1 {
		return NewUsageError("cws storage <action> [args]", "cws storage create my-data 100GB")
	}

	action := args[0]
	storageArgs := args[1:]

	// Ensure daemon is running (auto-start if needed)
	if err := sc.app.ensureDaemonRunning(); err != nil {
		return err
	}

	switch action {
	case "create":
		return sc.storageCreate(storageArgs)
	case "list":
		return sc.storageList(storageArgs)
	case "info":
		return sc.storageInfo(storageArgs)
	case "attach":
		return sc.storageAttach(storageArgs)
	case "detach":
		return sc.storageDetach(storageArgs)
	case "delete":
		return sc.storageDelete(storageArgs)
	default:
		return NewValidationError("storage action", action, "create, list, info, attach, detach, delete")
	}
}

func (sc *StorageCommands) storageCreate(args []string) error {
	if len(args) < 2 {
		return NewUsageError("cws storage create <name> <size> [type]", "cws storage create my-data 100GB gp3")
	}

	req := types.StorageCreateRequest{
		Name:       args[0],
		Size:       args[1],
		VolumeType: DefaultVolumeType, // default
	}

	// Parse volume type and options
	optionStartIndex := 2
	if len(args) > 2 && !strings.HasPrefix(args[2], "--") {
		req.VolumeType = args[2]
		optionStartIndex = 3
	}

	// Parse additional options
	for i := optionStartIndex; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "--region" && i+1 < len(args):
			req.Region = args[i+1]
			i++
		default:
			return NewValidationError("storage option", arg, "--region")
		}
	}

	volume, err := sc.app.apiClient.CreateStorage(sc.app.ctx, req)
	if err != nil {
		return WrapAPIError("create workspace storage "+req.Name, err)
	}

	sizeStr := "unknown"
	if volume.SizeGB != nil {
		sizeStr = fmt.Sprintf("%d GB", *volume.SizeGB)
	}
	fmt.Printf("%s\n", FormatSuccessMessage("Created Workspace Storage", volume.Name, fmt.Sprintf("(%s) - %s %s", volume.VolumeID, sizeStr, volume.VolumeType)))
	return nil
}

func (sc *StorageCommands) storageList(_ []string) error {
	volumes, err := sc.app.apiClient.ListStorage(sc.app.ctx)
	if err != nil {
		return WrapAPIError("list storage volumes", err)
	}

	if len(volumes) == 0 {
		fmt.Println("No storage volumes found. Create one with 'cws storage create'.")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, TabWriterMinWidth, TabWriterTabWidth, TabWriterPadding, TabWriterPadChar, TabWriterFlags)
	_, _ = fmt.Fprintln(w, "NAME\tTYPE\tSTATE\tSIZE\tDETAILS\tCOST/MONTH")

	for _, volume := range volumes {
		var sizeStr, detailsStr string
		var costMonth float64

		if volume.IsWorkspace() {
			// Workspace Storage (EBS)
			if volume.SizeGB != nil {
				sizeStr = fmt.Sprintf("%d GB", *volume.SizeGB)
				costMonth = float64(*volume.SizeGB) * volume.EstimatedCostGB
			}
			if volume.VolumeType != "" {
				detailsStr = volume.VolumeType
			}
			if volume.AttachedTo != "" {
				detailsStr += fmt.Sprintf(" → %s", volume.AttachedTo)
			}
		} else if volume.IsShared() {
			// Shared Storage (EFS)
			if volume.SizeBytes != nil {
				sizeGB := float64(*volume.SizeBytes) / BytesToGB
				sizeStr = fmt.Sprintf("%.1f GB", sizeGB)
				costMonth = sizeGB * volume.EstimatedCostGB
			}
			detailsStr = volume.PerformanceMode
		}

		if detailsStr == "" {
			detailsStr = "-"
		}

		_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t$%.2f\n",
			volume.Name,
			volume.GetDisplayType(),
			strings.ToUpper(volume.State),
			sizeStr,
			detailsStr,
			costMonth,
		)
	}
	_ = w.Flush()

	return nil
}

func (sc *StorageCommands) storageInfo(args []string) error {
	if len(args) < 1 {
		return NewUsageError("cws storage info <name>", "cws storage info my-data")
	}

	name := args[0]
	volume, err := sc.app.apiClient.GetStorage(sc.app.ctx, name)
	if err != nil {
		return WrapAPIError("get storage info for "+name, err)
	}

	// Display common fields
	icon := "💾"
	if volume.IsShared() {
		icon = "📁"
	}
	fmt.Printf("%s %s: %s\n", icon, volume.GetDisplayType(), volume.Name)
	fmt.Printf("   State: %s\n", strings.ToUpper(volume.State))
	fmt.Printf("   Region: %s\n", volume.Region)

	// Display type-specific fields
	if volume.IsWorkspace() {
		// Workspace Storage (EBS) fields
		fmt.Printf("   Volume ID: %s\n", volume.VolumeID)
		if volume.SizeGB != nil {
			fmt.Printf("   Size: %d GB\n", *volume.SizeGB)
		}
		fmt.Printf("   Type: %s\n", volume.VolumeType)
		if volume.IOPS != nil && *volume.IOPS > 0 {
			fmt.Printf("   IOPS: %d\n", *volume.IOPS)
		}
		if volume.Throughput != nil && *volume.Throughput > 0 {
			fmt.Printf("   Throughput: %d MB/s\n", *volume.Throughput)
		}
		if volume.AttachedTo != "" {
			fmt.Printf("   Attached to: %s\n", volume.AttachedTo)
		}
		if volume.SizeGB != nil {
			fmt.Printf("   Cost: $%.2f/month\n", float64(*volume.SizeGB)*volume.EstimatedCostGB)
		}
	} else if volume.IsShared() {
		// Shared Storage (EFS) fields
		fmt.Printf("   Filesystem ID: %s\n", volume.FileSystemID)
		if volume.SizeBytes != nil {
			sizeGB := float64(*volume.SizeBytes) / BytesToGB
			fmt.Printf("   Size: %.1f GB\n", sizeGB)
			fmt.Printf("   Cost: $%.2f/month\n", sizeGB*volume.EstimatedCostGB)
		}
		fmt.Printf("   Performance Mode: %s\n", volume.PerformanceMode)
		fmt.Printf("   Throughput Mode: %s\n", volume.ThroughputMode)
	}

	fmt.Printf("   Created: %s\n", volume.CreationTime.Format(StandardDateFormat))
	fmt.Printf("   AWS Service: %s\n", volume.GetTechnicalType())

	return nil
}

func (sc *StorageCommands) storageAttach(args []string) error {
	if len(args) < 2 {
		return NewUsageError("cws storage attach <volume> <workspace>", "cws storage attach my-data my-workspace")
	}

	volumeName := args[0]
	instanceName := args[1]

	err := sc.app.apiClient.AttachStorage(sc.app.ctx, volumeName, instanceName)
	if err != nil {
		return WrapAPIError("attach storage "+volumeName+" to "+instanceName, err)
	}

	fmt.Printf("%s\n", FormatProgressMessage("Attaching volume", fmt.Sprintf("%s to workspace %s", volumeName, instanceName)))
	return nil
}

func (sc *StorageCommands) storageDetach(args []string) error {
	if len(args) < 1 {
		return NewUsageError("cws storage detach <volume>", "cws storage detach my-data")
	}

	volumeName := args[0]

	err := sc.app.apiClient.DetachStorage(sc.app.ctx, volumeName)
	if err != nil {
		return WrapAPIError("detach storage "+volumeName, err)
	}

	fmt.Printf("%s\n", FormatProgressMessage("Detaching volume", volumeName))
	return nil
}

func (sc *StorageCommands) storageDelete(args []string) error {
	if len(args) < 1 {
		return NewUsageError("cws storage delete <name>", "cws storage delete my-data")
	}

	name := args[0]
	err := sc.app.apiClient.DeleteStorage(sc.app.ctx, name)
	if err != nil {
		return WrapAPIError("delete storage "+name, err)
	}

	fmt.Printf("%s\n", FormatProgressMessage("Deleting storage", name))
	return nil
}
