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
		return fmt.Errorf("usage: cws connect <instance-name> [--verbose]")
	}

	name := args[0]
	verbose := false

	// Parse flags
	for i := 1; i < len(args); i++ {
		switch args[i] {
		case "--verbose", "-v":
			verbose = true
		default:
			return fmt.Errorf("unknown flag: %s", args[i])
		}
	}

	// Check daemon is running
	if err := ic.app.apiClient.Ping(ic.app.ctx); err != nil {
		return fmt.Errorf("daemon not running. Start with: cws daemon start")
	}

	connectionInfo, err := ic.app.apiClient.ConnectInstance(ic.app.ctx, name)
	if err != nil {
		return fmt.Errorf("failed to get connection info: %w", err)
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
		return fmt.Errorf("usage: cws stop <name>")
	}

	name := args[0]

	// Check daemon is running
	if err := ic.app.apiClient.Ping(ic.app.ctx); err != nil {
		return fmt.Errorf("daemon not running. Start with: cws daemon start")
	}

	err := ic.app.apiClient.StopInstance(ic.app.ctx, name)
	if err != nil {
		return fmt.Errorf("failed to stop instance: %w", err)
	}

	fmt.Printf("⏹️ Stopping instance %s...\n", name)
	return nil
}

// Start handles the start command with intelligent state management
func (ic *InstanceCommands) Start(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: cws start <name>")
	}

	name := args[0]

	// Check daemon is running
	if err := ic.app.apiClient.Ping(ic.app.ctx); err != nil {
		return fmt.Errorf("daemon not running. Start with: cws daemon start")
	}

	// First, get current instance status
	listResponse, err := ic.app.apiClient.ListInstances(ic.app.ctx)
	if err != nil {
		return fmt.Errorf("failed to get instance status: %w", err)
	}

	var targetInstance *types.Instance
	for _, instance := range listResponse.Instances {
		if instance.Name == name {
			targetInstance = &instance
			break
		}
	}

	if targetInstance == nil {
		return fmt.Errorf("instance '%s' not found", name)
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
		return fmt.Errorf("failed to start instance: %w", err)
	}

	fmt.Printf("▶️ Starting instance %s...\n", name)
	return nil
}

// Delete handles the delete command
func (ic *InstanceCommands) Delete(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: cws delete <name>")
	}

	name := args[0]

	// Check daemon is running
	if err := ic.app.apiClient.Ping(ic.app.ctx); err != nil {
		return fmt.Errorf("daemon not running. Start with: cws daemon start")
	}

	err := ic.app.apiClient.DeleteInstance(ic.app.ctx, name)
	if err != nil {
		return fmt.Errorf("failed to delete instance: %w", err)
	}

	fmt.Printf("🗑️ Deleting instance %s...\n", name)
	return nil
}

// Hibernate handles the hibernate command
func (ic *InstanceCommands) Hibernate(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: cws hibernate <name>")
	}

	name := args[0]

	// Check daemon is running
	if err := ic.app.apiClient.Ping(ic.app.ctx); err != nil {
		return fmt.Errorf("daemon not running. Start with: cws daemon start")
	}

	// Check hibernation status first
	status, err := ic.app.apiClient.GetInstanceHibernationStatus(ic.app.ctx, name)
	if err != nil {
		return fmt.Errorf("failed to check hibernation status: %w", err)
	}

	if !status.HibernationSupported {
		fmt.Printf("⚠️  Instance %s does not support hibernation\n", name)
		fmt.Printf("    Falling back to regular stop operation\n")
	}

	err = ic.app.apiClient.HibernateInstance(ic.app.ctx, name)
	if err != nil {
		return fmt.Errorf("failed to hibernate instance: %w", err)
	}

	if status.HibernationSupported {
		fmt.Printf("🛌 Hibernating instance %s...\n", name)
		fmt.Printf("   💡 RAM state preserved for instant resume\n")
		fmt.Printf("   💰 Compute billing stopped, storage billing continues\n")
	} else {
		fmt.Printf("⏹️ Stopping instance %s...\n", name)
		fmt.Printf("   💡 Consider using hibernation-capable instance types for RAM preservation\n")
	}

	return nil
}

// Resume handles the resume command
func (ic *InstanceCommands) Resume(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: cws resume <name>")
	}

	name := args[0]

	// Check daemon is running
	if err := ic.app.apiClient.Ping(ic.app.ctx); err != nil {
		return fmt.Errorf("daemon not running. Start with: cws daemon start")
	}

	// Check hibernation status first
	status, err := ic.app.apiClient.GetInstanceHibernationStatus(ic.app.ctx, name)
	if err != nil {
		return fmt.Errorf("failed to check hibernation status: %w", err)
	}

	if status.IsHibernated {
		err = ic.app.apiClient.ResumeInstance(ic.app.ctx, name)
		if err != nil {
			return fmt.Errorf("failed to resume instance: %w", err)
		}
		fmt.Printf("⏰ Resuming hibernated instance %s...\n", name)
		fmt.Printf("   🚀 Instant startup from preserved RAM state\n")
	} else {
		// Fall back to regular start
		err = ic.app.apiClient.StartInstance(ic.app.ctx, name)
		if err != nil {
			return fmt.Errorf("failed to start instance: %w", err)
		}
		fmt.Printf("▶️ Starting instance %s...\n", name)
		fmt.Printf("   💡 Instance was not hibernated - performing regular start\n")
	}

	return nil
}