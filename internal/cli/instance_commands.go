package cli

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

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

	// Check EC2 hibernation support first
	status, err := ic.app.apiClient.GetInstanceHibernationStatus(ic.app.ctx, name)
	if err != nil {
		return WrapAPIError("check EC2 hibernation support for "+name, err)
	}

	if !status.HibernationSupported {
		fmt.Printf("⚠️  Instance %s does not support EC2 hibernation\n", name)
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
		fmt.Printf("   %s\n", FormatInfoMessage("Consider using EC2 hibernation-capable instance types for RAM preservation"))
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

	// Check EC2 hibernation status first
	status, err := ic.app.apiClient.GetInstanceHibernationStatus(ic.app.ctx, name)
	if err != nil {
		return WrapAPIError("check EC2 hibernation status for "+name, err)
	}

	if status.PossiblyHibernated {
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

// Exec handles the exec command - executes commands remotely on instances
// Note: This method is called from the Cobra command structure, so flag parsing
// is handled by Cobra. This simplified version assumes args contains only positional arguments.
func (ic *InstanceCommands) Exec(args []string) error {
	if len(args) < 2 {
		return NewUsageError("cws exec <instance-name> <command>", "cws exec my-workstation \"ls -la\"")
	}

	instanceName := args[0]
	command := args[1]

	// For now, use simple argument parsing since Cobra integration will handle flags
	// TODO: Integrate with Cobra flag system when this is called from Cobra command
	var user string
	var workingDir string
	var timeout int = 30
	environment := make(map[string]string)
	interactive := false
	verbose := false

	// Simple flag parsing for direct API usage (non-Cobra calls)
	for i := 2; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "--user" && i+1 < len(args):
			user = args[i+1]
			i++
		case arg == "--working-dir" && i+1 < len(args):
			workingDir = args[i+1]
			i++
		case arg == "--timeout" && i+1 < len(args):
			timeout, _ = strconv.Atoi(args[i+1])
			i++
		case strings.HasPrefix(arg, "--env="):
			envPart := strings.TrimPrefix(arg, "--env=")
			if envKV := strings.SplitN(envPart, "=", 2); len(envKV) == 2 {
				environment[envKV[0]] = envKV[1]
			}
		case arg == "--interactive" || arg == "-i":
			interactive = true
		case arg == "--verbose" || arg == "-v":
			verbose = true
		}
	}

	// Create exec request
	execRequest := types.ExecRequest{
		Command:        command,
		WorkingDir:     workingDir,
		User:           user,
		Environment:    environment,
		TimeoutSeconds: timeout,
		Interactive:    interactive,
	}

	// Ensure daemon is running (auto-start if needed)
	if err := ic.app.ensureDaemonRunning(); err != nil {
		return err
	}

	if verbose {
		fmt.Printf("🔧 Executing command on %s...\n", instanceName)
		fmt.Printf("   Command: %s\n", command)
		if user != "" {
			fmt.Printf("   User: %s\n", user)
		}
		if workingDir != "" {
			fmt.Printf("   Working Directory: %s\n", workingDir)
		}
		if len(environment) > 0 {
			fmt.Printf("   Environment: %v\n", environment)
		}
		fmt.Printf("   Timeout: %ds\n", timeout)
		fmt.Printf("   Interactive: %t\n", interactive)
		fmt.Println()
	}

	// Execute the command
	result, err := ic.app.apiClient.ExecInstance(ic.app.ctx, instanceName, execRequest)
	if err != nil {
		return WrapAPIError("execute command on "+instanceName, err)
	}

	// Display results based on verbosity
	if verbose {
		fmt.Printf("📊 Command execution completed:\n")
		fmt.Printf("   Exit Code: %d\n", result.ExitCode)
		fmt.Printf("   Status: %s\n", result.Status)
		fmt.Printf("   Execution Time: %dms\n", result.ExecutionTime)
		if result.CommandID != "" {
			fmt.Printf("   Command ID: %s\n", result.CommandID)
		}
		fmt.Println()
	}

	// Display stdout if available
	if result.StdOut != "" {
		if verbose {
			fmt.Printf("📤 Output:\n")
		}
		fmt.Print(result.StdOut)
		if !strings.HasSuffix(result.StdOut, "\n") {
			fmt.Println()
		}
	}

	// Display stderr if available and command failed
	if result.StdErr != "" && (result.ExitCode != 0 || verbose) {
		if verbose {
			fmt.Printf("⚠️  Error Output:\n")
		}
		fmt.Fprint(os.Stderr, result.StdErr)
		if !strings.HasSuffix(result.StdErr, "\n") {
			fmt.Fprintln(os.Stderr)
		}
	}

	// Exit with the same code as the remote command
	if result.ExitCode != 0 {
		os.Exit(result.ExitCode)
	}

	return nil
}

// Resize handles the resize command - changes instance type/size
func (ic *InstanceCommands) Resize(args []string) error {
	if len(args) < 2 {
		return NewUsageError("cws resize <instance-name> --size <size> [options]",
			"cws resize my-workstation --size L")
	}

	instanceName := args[0]
	var newSize string
	var instanceType string
	var dryRun bool
	var force bool
	var wait bool

	// Parse flags
	for i := 1; i < len(args); i++ {
		switch args[i] {
		case "--size":
			if i+1 >= len(args) {
				return NewValidationError("--size", "", "requires a t-shirt size (XS, S, M, L, XL)")
			}
			newSize = strings.ToUpper(args[i+1])
			if !ValidTSizes[newSize] {
				return NewValidationError("size", newSize, "XS, S, M, L, XL")
			}
			i++
		case "--instance-type":
			if i+1 >= len(args) {
				return NewValidationError("--instance-type", "", "requires an AWS instance type")
			}
			instanceType = args[i+1]
			i++
		case "--dry-run":
			dryRun = true
		case "--force":
			force = true
		case "--wait":
			wait = true
		default:
			return NewValidationError("flag", args[i], "--size, --instance-type, --dry-run, --force, or --wait")
		}
	}

	// Ensure daemon is running (auto-start if needed)
	if err := ic.app.ensureDaemonRunning(); err != nil {
		return err
	}

	// Get current instance info
	listResponse, err := ic.app.apiClient.ListInstances(ic.app.ctx)
	if err != nil {
		return WrapAPIError("get instance status", err)
	}

	var targetInstance *types.Instance
	for _, instance := range listResponse.Instances {
		if instance.Name == instanceName {
			targetInstance = &instance
			break
		}
	}

	if targetInstance == nil {
		return NewNotFoundError("instance", instanceName, "Use 'cws list' to see available instances")
	}

	// Determine target instance type
	var targetInstanceType string
	if instanceType != "" {
		targetInstanceType = instanceType
	} else if newSize != "" {
		if mappedType, exists := SizeInstanceTypeMapping[newSize]; exists {
			targetInstanceType = mappedType
		} else {
			return NewValidationError("size", newSize, "valid t-shirt size (XS, S, M, L, XL)")
		}
	} else {
		return NewUsageError("cws resize <instance-name> --size <size> OR --instance-type <type>",
			"cws resize my-workstation --size L")
	}

	// Parse current size
	currentSize := "Unknown"
	if size, exists := InstanceTypeSizeMapping[targetInstance.InstanceType]; exists {
		currentSize = size
	}

	fmt.Printf("🔄 Instance Resize Operation\n")
	fmt.Printf("═══════════════════════════\n\n")

	fmt.Printf("📋 **Resize Details**:\n")
	fmt.Printf("   Instance: %s\n", instanceName)
	fmt.Printf("   Current Type: %s (%s)\n", targetInstance.InstanceType, currentSize)
	fmt.Printf("   Target Type: %s", targetInstanceType)
	if newSize != "" {
		fmt.Printf(" (%s)", newSize)
	}
	fmt.Printf("\n")
	fmt.Printf("   Current State: %s\n\n", targetInstance.State)

	// Check if resize is needed
	if targetInstance.InstanceType == targetInstanceType {
		fmt.Printf("✅ Instance is already type %s. No resize needed.\n", targetInstanceType)
		return nil
	}

	// Validate instance state
	if targetInstance.State != "running" && targetInstance.State != "stopped" {
		return NewStateError("instance", instanceName, targetInstance.State, "running or stopped")
	}

	// Show cost comparison
	currentCost := targetInstance.HourlyRate
	newCost := ic.estimateCostForInstanceType(targetInstanceType)

	fmt.Printf("💰 **Cost Impact**:\n")
	fmt.Printf("   Current Cost: $%.2f/day\n", currentCost)
	fmt.Printf("   New Cost: $%.2f/day\n", newCost)

	if newCost > currentCost {
		fmt.Printf("   Impact: +$%.2f/day (+%.1f%%)\n", newCost-currentCost, ((newCost-currentCost)/currentCost)*100)
		fmt.Printf("   Monthly Impact: +$%.2f\n", (newCost-currentCost)*30)
	} else if newCost < currentCost {
		fmt.Printf("   Impact: -$%.2f/day (-%.1f%%)\n", currentCost-newCost, ((currentCost-newCost)/currentCost)*100)
		fmt.Printf("   Monthly Savings: $%.2f\n", (currentCost-newCost)*30)
	} else {
		fmt.Printf("   Impact: No cost change\n")
	}

	fmt.Printf("\n⚡ **Resize Process**:\n")
	if targetInstance.State == "running" {
		fmt.Printf("   1. Stop instance (preserves data)\n")
		fmt.Printf("   2. Modify instance type\n")
		fmt.Printf("   3. Start with new configuration\n")
		fmt.Printf("   4. Validate functionality\n")
		fmt.Printf("   Estimated downtime: 2-5 minutes\n\n")
	} else {
		fmt.Printf("   1. Modify instance type (instance stopped)\n")
		fmt.Printf("   2. Start with new configuration\n")
		fmt.Printf("   No additional downtime required\n\n")
	}

	if dryRun {
		fmt.Printf("🔍 **Dry Run Complete**\n")
		fmt.Printf("   Resize operation validated successfully\n")
		fmt.Printf("   Run without --dry-run to execute\n")
		return nil
	}

	// Confirmation prompt unless --force is used
	if !force {
		fmt.Printf("⚠️  **Confirmation Required**\n")
		fmt.Printf("   This will modify the instance type and require a restart.\n")
		fmt.Printf("   Type the instance name to confirm: ")

		var confirmation string
		fmt.Scanln(&confirmation)

		if confirmation != instanceName {
			fmt.Printf("❌ Instance name doesn't match. Resize cancelled.\n")
			return nil
		}
	}

	// Create resize request
	resizeRequest := types.ResizeRequest{
		InstanceName:       instanceName,
		TargetInstanceType: targetInstanceType,
		Force:              force,
		Wait:               wait,
	}

	// Execute resize
	response, err := ic.app.apiClient.ResizeInstance(ic.app.ctx, resizeRequest)
	if err != nil {
		return WrapAPIError("resize instance "+instanceName, err)
	}

	fmt.Printf("✅ %s\n", response.Message)

	if wait {
		fmt.Printf("⏳ Monitoring resize progress...\n")
		return ic.monitorResizeProgress(instanceName)
	} else {
		fmt.Printf("💡 Monitor progress with: cws list\n")
		fmt.Printf("💡 Check when ready: cws connect %s\n", instanceName)
	}

	return nil
}

// estimateCostForInstanceType estimates daily cost for an instance type
func (ic *InstanceCommands) estimateCostForInstanceType(instanceType string) float64 {
	// Try to map instance type to t-shirt size for cost estimation
	if size, exists := InstanceTypeSizeMapping[instanceType]; exists {
		if specs, exists := TSizeSpecifications[size]; exists {
			return specs.Cost
		}
	}

	// Fallback cost estimation based on instance family
	switch {
	case strings.Contains(instanceType, "nano"):
		return 0.25
	case strings.Contains(instanceType, "micro"):
		return 0.50
	case strings.Contains(instanceType, "small"):
		return 1.00
	case strings.Contains(instanceType, "medium"):
		return 2.00
	case strings.Contains(instanceType, "large"):
		return 4.00
	case strings.Contains(instanceType, "xlarge"):
		return 8.00
	case strings.Contains(instanceType, "2xlarge"):
		return 16.00
	case strings.Contains(instanceType, "4xlarge"):
		return 32.00
	default:
		return 2.00 // Default estimate
	}
}

// monitorResizeProgress monitors resize operation progress
func (ic *InstanceCommands) monitorResizeProgress(instanceName string) error {
	fmt.Printf("🔄 Monitoring resize progress for %s...\n", instanceName)

	maxAttempts := 60 // 5 minutes max
	for i := 0; i < maxAttempts; i++ {
		// Check current status
		instance, err := ic.app.apiClient.GetInstance(ic.app.ctx, instanceName)
		if err != nil {
			fmt.Printf("⚠️  Unable to get instance status: %v\n", err)
			return nil
		}

		switch instance.State {
		case "running":
			fmt.Printf("✅ Resize complete! Instance is running with new configuration.\n")
			fmt.Printf("🔗 Connect: cws connect %s\n", instanceName)
			return nil
		case "stopped", "stopping":
			fmt.Printf("⏳ Instance stopping for resize... (%ds)\n", i*5)
		case "pending":
			fmt.Printf("⏳ Instance starting with new configuration... (%ds)\n", i*5)
		default:
			fmt.Printf("📊 Status: %s (%ds)\n", instance.State, i*5)
		}

		if i < maxAttempts-1 {
			time.Sleep(5 * time.Second)
		}
	}

	fmt.Printf("⚠️  Resize monitoring timeout. Instance may still be resizing.\n")
	fmt.Printf("💡 Check status with: cws list\n")
	return nil
}
