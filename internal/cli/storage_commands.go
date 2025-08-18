package cli

import (
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/scttfrdmn/cloudworkstation/pkg/types"
)

// StorageCommands handles all storage management operations
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

	// Check daemon is running
	if err := sc.app.apiClient.Ping(sc.app.ctx); err != nil {
		return WrapDaemonError(err)
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
		return WrapAPIError("create EFS volume "+req.Name, err)
	}

	fmt.Printf("%s\n", FormatSuccessMessage("Created EFS volume", volume.Name, fmt.Sprintf("(%s)", volume.FileSystemId)))
	return nil
}

func (sc *StorageCommands) volumeList(_ []string) error {
	volumes, err := sc.app.apiClient.ListVolumes(sc.app.ctx)
	if err != nil {
		return WrapAPIError("list EFS volumes", err)
	}

	if len(volumes) == 0 {
		fmt.Println(NoEFSVolumesFoundMessage)
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, TabWriterMinWidth, TabWriterTabWidth, TabWriterPadding, TabWriterPadChar, TabWriterFlags)
	_, _ = fmt.Fprintln(w, "NAME\tFILESYSTEM ID\tSTATE\tSIZE\tCOST/MONTH")

	for _, volume := range volumes {
		sizeGB := float64(volume.SizeBytes) / BytesToGB
		costMonth := sizeGB * volume.EstimatedCostGB
		_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%.1f GB\t$%.2f\n",
			volume.Name,
			volume.FileSystemId,
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

	fmt.Printf("📁 EFS Volume: %s\n", volume.Name)
	fmt.Printf("   Filesystem ID: %s\n", volume.FileSystemId)
	fmt.Printf("   State: %s\n", strings.ToUpper(volume.State))
	fmt.Printf("   Region: %s\n", volume.Region)
	fmt.Printf("   Performance Mode: %s\n", volume.PerformanceMode)
	fmt.Printf("   Throughput Mode: %s\n", volume.ThroughputMode)
	fmt.Printf("   Size: %.1f GB\n", float64(volume.SizeBytes)/BytesToGB)
	fmt.Printf("   Cost: $%.2f/month\n", float64(volume.SizeBytes)/BytesToGB*volume.EstimatedCostGB)
	fmt.Printf("   Created: %s\n", volume.CreationTime.Format(StandardDateFormat))

	return nil
}

func (sc *StorageCommands) volumeDelete(args []string) error {
	if len(args) < 1 {
		return NewUsageError("cws volume delete <name>", "cws volume delete my-shared-data")
	}

	name := args[0]
	err := sc.app.apiClient.DeleteVolume(sc.app.ctx, name)
	if err != nil {
		return WrapAPIError("delete EFS volume "+name, err)
	}

	fmt.Printf("%s\n", FormatProgressMessage("Deleting EFS volume", name))
	return nil
}

func (sc *StorageCommands) volumeMount(args []string) error {
	if len(args) < 2 {
		return NewUsageError("cws volume mount <volume-name> <instance-name> [mount-point]", "cws volume mount my-shared-data my-workstation")
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

	fmt.Printf("%s\n", FormatProgressMessage("Mounting EFS volume", fmt.Sprintf("'%s' to '%s' at %s", volumeName, instanceName, mountPoint)))
	return nil
}

func (sc *StorageCommands) volumeUnmount(args []string) error {
	if len(args) < 2 {
		return NewUsageError("cws volume unmount <volume-name> <instance-name>", "cws volume unmount my-shared-data my-workstation")
	}

	volumeName := args[0]
	instanceName := args[1]

	err := sc.app.apiClient.UnmountVolume(sc.app.ctx, volumeName, instanceName)
	if err != nil {
		return WrapAPIError("unmount volume "+volumeName+" from "+instanceName, err)
	}

	fmt.Printf("%s\n", FormatProgressMessage("Unmounting EFS volume", fmt.Sprintf("'%s' from '%s'", volumeName, instanceName)))
	return nil
}

// Storage handles storage commands
func (sc *StorageCommands) Storage(args []string) error {
	if len(args) < 1 {
		return NewUsageError("cws storage <action> [args]", "cws storage create my-data 100GB")
	}

	action := args[0]
	storageArgs := args[1:]

	// Check daemon is running
	if err := sc.app.apiClient.Ping(sc.app.ctx); err != nil {
		return WrapDaemonError(err)
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
			return NewValidationError("storage option", arg, "--region")
		}
	}

	volume, err := sc.app.apiClient.CreateStorage(sc.app.ctx, req)
	if err != nil {
		return WrapAPIError("create EBS volume "+req.Name, err)
	}

	fmt.Printf("%s\n", FormatSuccessMessage("Created EBS volume", volume.Name, fmt.Sprintf("(%s) - %d GB %s", volume.VolumeID, volume.SizeGB, volume.VolumeType)))
	return nil
}

func (sc *StorageCommands) storageList(_ []string) error {
	volumes, err := sc.app.apiClient.ListStorage(sc.app.ctx)
	if err != nil {
		return WrapAPIError("list EBS volumes", err)
	}

	if len(volumes) == 0 {
		fmt.Println(NoEBSVolumesFoundMessage)
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, TabWriterMinWidth, TabWriterTabWidth, TabWriterPadding, TabWriterPadChar, TabWriterFlags)
	_, _ = fmt.Fprintln(w, "NAME\tVOLUME ID\tSTATE\tSIZE\tTYPE\tATTACHED TO\tCOST/MONTH")

	for _, volume := range volumes {
		costMonth := float64(volume.SizeGB) * volume.EstimatedCostGB
		attachedTo := volume.AttachedTo
		if attachedTo == "" {
			attachedTo = "-"
		}
		_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%d GB\t%s\t%s\t$%.2f\n",
			volume.Name,
			volume.VolumeID,
			strings.ToUpper(volume.State),
			volume.SizeGB,
			volume.VolumeType,
			attachedTo,
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
	fmt.Printf("   Created: %s\n", volume.CreationTime.Format(StandardDateFormat))

	return nil
}

func (sc *StorageCommands) storageAttach(args []string) error {
	if len(args) < 2 {
		return NewUsageError("cws storage attach <volume> <instance>", "cws storage attach my-data my-workstation")
	}

	volumeName := args[0]
	instanceName := args[1]

	err := sc.app.apiClient.AttachStorage(sc.app.ctx, volumeName, instanceName)
	if err != nil {
		return WrapAPIError("attach storage "+volumeName+" to "+instanceName, err)
	}

	fmt.Printf("%s\n", FormatProgressMessage("Attaching volume", fmt.Sprintf("%s to instance %s", volumeName, instanceName)))
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
		return WrapAPIError("delete EBS volume "+name, err)
	}

	fmt.Printf("%s\n", FormatProgressMessage("Deleting EBS volume", name))
	return nil
}