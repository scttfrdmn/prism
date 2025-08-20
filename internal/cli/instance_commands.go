package cli

import (
	"fmt"

	"github.com/scttfrdmn/cloudworkstation/pkg/types"
)

// InstanceCommands handles all instance management operations
type InstanceCommands struct {
	app *App
}

// NewInstanceCommands creates instance commands handler
func NewInstanceCommands(app *App) *InstanceCommands {
	return &InstanceCommands{app: app}
}

// Connect handles the connect command
func (ic *InstanceCommands) Connect(args []string) error {
	if len(args) < 1 {
		return NewUsageError("cws connect <instance-name>", "cws connect my-workstation")
	}
	
	name := args[0]
	verbose := false

	// Parse flags
	for i := 1; i < len(args); i++ {
		switch args[i] {
		case "--verbose", "-v":
			verbose = true
		default:
			return NewValidationError("flag", args[i], "--verbose or -v")
		}
	}

	// Ensure daemon is running (auto-start if needed)
	if err := ic.app.ensureDaemonRunning(); err != nil {
		return err
	}

	connectionInfo, err := ic.app.apiClient.ConnectInstance(ic.app.ctx, name)
	if err != nil {
		return WrapAPIError("get connection info for "+name, err)
	}

	if verbose {
		fmt.Printf("🔗 SSH command for %s:\n", name)
		fmt.Printf("%s\n", connectionInfo)
	} else {
		return ic.app.executeSSHCommand(connectionInfo, name)
	}

	return nil
}

// Stop handles the stop command
func (ic *InstanceCommands) Stop(args []string) error {
	if len(args) < 1 {
		return NewUsageError("cws stop <name>", "cws stop my-workstation")
	}

	name := args[0]

	// Ensure daemon is running (auto-start if needed)
	if err := ic.app.ensureDaemonRunning(); err != nil {
		return err
	}

	err := ic.app.apiClient.StopInstance(ic.app.ctx, name)
	if err != nil {
		return WrapAPIError("stop instance "+name, err)
	}

	fmt.Printf("%s\n", FormatProgressMessage("Stopping instance", name))
	return nil
}

// Start handles the start command with intelligent state management
func (ic *InstanceCommands) Start(args []string) error {
	if len(args) < 1 {
		return NewUsageError("cws start <name>", "cws start my-workstation")
	}
	
	name := args[0]

	// Ensure daemon is running (auto-start if needed)
	if err := ic.app.ensureDaemonRunning(); err != nil {
		return err
	}

	// First, get current instance status
	listResponse, err := ic.app.apiClient.ListInstances(ic.app.ctx)
	if err != nil {
		return WrapAPIError("get instance status", err)
	}

	var targetInstance *types.Instance
	for _, instance := range listResponse.Instances {
		if instance.Name == name {
			targetInstance = &instance
			break
		}
	}

	if targetInstance == nil {
		return NewNotFoundError("instance", name, "Use 'cws list' to see available instances")
	}

	// Check current state and handle appropriately
	switch targetInstance.State {
	case "running":
		fmt.Printf("✅ Instance %s is already running\n", name)
		return nil
	case "hibernated":
		fmt.Printf("🛌 Instance %s is hibernated - use 'cws resume %s' for instant startup\n", name, name)
		fmt.Printf("   Or use 'cws start %s' for regular boot (slower)\n", name)
		fmt.Printf("   Proceeding with regular start...\n")
	case "stopped", "stopping":
		// Normal case - proceed with start
	default:
		fmt.Printf("⚠️  Instance %s is in state '%s' - attempting to start anyway\n", name, targetInstance.State)
	}

	err = ic.app.apiClient.StartInstance(ic.app.ctx, name)
	if err != nil {
		return WrapAPIError("start instance "+name, err)
	}

	fmt.Printf("%s\n", FormatProgressMessage("Starting instance", name))
	return nil
}

// Delete handles the delete command
func (ic *InstanceCommands) Delete(args []string) error {
	if len(args) < 1 {
		return NewUsageError("cws delete <name>", "cws delete my-workstation")
	}
	
	name := args[0]

	// Ensure daemon is running (auto-start if needed)
	if err := ic.app.ensureDaemonRunning(); err != nil {
		return err
	}

	err := ic.app.apiClient.DeleteInstance(ic.app.ctx, name)
	if err != nil {
		return WrapAPIError("delete instance "+name, err)
	}

	fmt.Printf("%s\n", FormatProgressMessage("Deleting instance", name))
	return nil
}

// Hibernate handles the hibernate command
func (ic *InstanceCommands) Hibernate(args []string) error {
	if len(args) < 1 {
		return NewUsageError("cws hibernate <name>", "cws hibernate my-workstation")
	}
	
	name := args[0]

	// Ensure daemon is running (auto-start if needed)
	if err := ic.app.ensureDaemonRunning(); err != nil {
		return err
	}

	// Check hibernation status first
	status, err := ic.app.apiClient.GetInstanceHibernationStatus(ic.app.ctx, name)
	if err != nil {
		return WrapAPIError("check hibernation status for "+name, err)
	}

	if !status.HibernationSupported {
		fmt.Printf("⚠️  Instance %s does not support hibernation\n", name)
		fmt.Printf("    Falling back to regular stop operation\n")
	}

	err = ic.app.apiClient.HibernateInstance(ic.app.ctx, name)
	if err != nil {
		return WrapAPIError("hibernate instance "+name, err)
	}

	if status.HibernationSupported {
		fmt.Printf("%s\n", FormatProgressMessage("Hibernating instance", name))
		fmt.Printf("   %s\n", FormatInfoMessage("RAM state preserved for instant resume"))
		fmt.Printf("   💰 Compute billing stopped, storage billing continues\n")
	} else {
		fmt.Printf("%s\n", FormatProgressMessage("Stopping instance", name))
		fmt.Printf("   %s\n", FormatInfoMessage("Consider using hibernation-capable instance types for RAM preservation"))
	}

	return nil
}

// Resume handles the resume command
func (ic *InstanceCommands) Resume(args []string) error {
	if len(args) < 1 {
		return NewUsageError("cws resume <name>", "cws resume my-workstation")
	}
	
	name := args[0]

	// Ensure daemon is running (auto-start if needed)
	if err := ic.app.ensureDaemonRunning(); err != nil {
		return err
	}

	// Check hibernation status first
	status, err := ic.app.apiClient.GetInstanceHibernationStatus(ic.app.ctx, name)
	if err != nil {
		return WrapAPIError("check hibernation status for "+name, err)
	}

	if status.IsHibernated {
		err = ic.app.apiClient.ResumeInstance(ic.app.ctx, name)
		if err != nil {
			return WrapAPIError("resume instance "+name, err)
		}
		fmt.Printf("%s\n", FormatProgressMessage("Resuming hibernated instance", name))
		fmt.Printf("   🚀 Instant startup from preserved RAM state\n")
	} else {
		// Fall back to regular start
		err = ic.app.apiClient.StartInstance(ic.app.ctx, name)
		if err != nil {
			return WrapAPIError("start instance "+name, err)
		}
		fmt.Printf("%s\n", FormatProgressMessage("Starting instance", name))
		fmt.Printf("   %s\n", FormatInfoMessage("Instance was not hibernated - performing regular start"))
	}

	return nil
}
