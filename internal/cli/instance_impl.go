// Package cli - Instance Implementation Layer
//
// ARCHITECTURE NOTE: This file contains the business logic implementation for instance commands.
// The user-facing CLI interface is defined in instances_cobra.go, which delegates to these methods.
//
// This separation follows the Facade/Adapter pattern:
//   - instances_cobra.go: CLI interface (Cobra commands, flag parsing, help text)
//   - instance_impl.go: Business logic (API calls, formatting, error handling)
//
// This architecture allows:
//   - Clean separation of concerns
//   - Reusable business logic (can be called from Cobra, TUI, or tests)
//   - Consistent API interaction patterns across all commands
//
// DO NOT REMOVE THIS FILE - it is actively used by instances_cobra.go and other components.
package cli

import (
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/scttfrdmn/prism/pkg/types"
)

// InstanceCommands handles all instance management operations (implementation layer)
type InstanceCommands struct {
	app *App
}

// NewInstanceCommands creates instance commands handler
func NewInstanceCommands(app *App) *InstanceCommands {
	return &InstanceCommands{app: app}
}

// Connect handles the connect command
func (ic *InstanceCommands) Connect(args []string) error {
	// Validate arguments
	if len(args) < 1 {
		return NewUsageError("prism workspace connect <workspace-name>", "prism workspace connect my-workspace")
	}

	// Parse flags
	name, verbose, userOverride, err := ic.parseConnectFlags(args)
	if err != nil {
		return err
	}

	// Ensure daemon is running (auto-start if needed)
	if err := ic.app.ensureDaemonRunning(); err != nil {
		return err
	}

	// Get instance to check connection type
	instance, err := ic.setupInstanceConnection(name)
	if err != nil {
		return err
	}

	// Route to appropriate connection handler based on connection type
	switch instance.ConnectionType {
	case "desktop":
		return ic.connectDesktopInstance(instance, name, verbose)
	case "web":
		return ic.connectWebInstance(instance, name, verbose)
	case "ssh", "":
		// Default to SSH for backwards compatibility
		return ic.connectSSHInstance(instance, name, userOverride, verbose)
	case "both":
		// For "both", prefer web if available, otherwise SSH
		if len(instance.Services) > 0 {
			return ic.connectWebInstance(instance, name, verbose)
		}
		return ic.connectSSHInstance(instance, name, userOverride, verbose)
	default:
		return fmt.Errorf("unknown connection type: %s", instance.ConnectionType)
	}
}

// parseConnectFlags parses connect command flags
func (ic *InstanceCommands) parseConnectFlags(args []string) (name string, verbose bool, userOverride string, err error) {
	name = args[0]

	for i := 1; i < len(args); i++ {
		switch args[i] {
		case "--verbose", "-v":
			verbose = true
		case "--user", "-u":
			if i+1 >= len(args) {
				return "", false, "", NewValidationError("--user", "", "requires a username")
			}
			userOverride = args[i+1]
			i++
		default:
			return "", false, "", NewValidationError("flag", args[i], "--verbose, -v, --user, or -u")
		}
	}

	return name, verbose, userOverride, nil
}

// setupInstanceConnection gets instance and creates tunnels
func (ic *InstanceCommands) setupInstanceConnection(name string) (*types.Instance, error) {
	instance, err := ic.app.apiClient.GetInstance(ic.app.ctx, name)
	if err != nil {
		return nil, WrapAPIError("get workspace details for "+name, err)
	}

	if len(instance.Services) > 0 {
		ic.createWebServiceTunnels(name)
	}

	return instance, nil
}

// createWebServiceTunnels creates tunnels for web services
func (ic *InstanceCommands) createWebServiceTunnels(name string) {
	fmt.Printf("🌐 Setting up tunnels for web services...\n")
	tunnelResp, err := ic.app.apiClient.CreateTunnels(ic.app.ctx, name, nil) // nil = create all
	if err != nil {
		fmt.Printf("⚠️  Warning: Could not create tunnels: %v\n", err)
		return
	}

	fmt.Printf("✅ Tunnels created:\n")
	for _, tunnel := range tunnelResp.Tunnels {
		if tunnel.AuthToken != "" {
			fmt.Printf("   • %s: %s?token=%s\n", tunnel.ServiceDesc, tunnel.LocalURL, tunnel.AuthToken)
		} else {
			fmt.Printf("   • %s: %s\n", tunnel.ServiceDesc, tunnel.LocalURL)
		}
	}
}

// getSSHConnection retrieves the instance SSH endpoint and applies any --user override
func (ic *InstanceCommands) getSSHConnection(name, userOverride string, verbose bool) (*types.SSHConnection, error) {
	conn, err := ic.app.apiClient.GetSSHConnection(ic.app.ctx, name)
	if err != nil {
		return nil, WrapAPIError("get connection info for "+name, err)
	}

	if userOverride != "" {
		// ssh receives this as an argument (after "--"), never through a shell; the check keeps
		// it a conventional login name.
		if !sshUsernamePattern.MatchString(userOverride) {
			return nil, fmt.Errorf("invalid username %q: must start with a letter or underscore and contain only letters, digits, '_', '.', or '-'", userOverride)
		}
		conn.User = userOverride
		if verbose {
			fmt.Printf("🔧 Username overridden: %s\n", userOverride)
		}
	}

	return conn, nil
}

// sshUsernamePattern matches a conventional POSIX login name.
var sshUsernamePattern = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_.-]{0,31}$`)

// executeConnection executes or displays connection info
func (ic *InstanceCommands) executeConnection(conn *types.SSHConnection, name string, verbose bool) error {
	if verbose {
		fmt.Printf("🔗 SSH command for %s:\n", name)
		fmt.Printf("%s\n", conn.String())
		return nil
	}
	return ic.app.executeSSHCommand(conn, name)
}

// connectSSHInstance handles SSH connections
func (ic *InstanceCommands) connectSSHInstance(instance *types.Instance, name, userOverride string, verbose bool) error {
	conn, err := ic.getSSHConnection(name, userOverride, verbose)
	if err != nil {
		return err
	}
	return ic.executeConnection(conn, name, verbose)
}

// connectWebInstance handles web-based connections
func (ic *InstanceCommands) connectWebInstance(instance *types.Instance, name string, verbose bool) error {
	// Setup web service tunnels (already done in setupInstanceConnection)
	// Just display connection information
	if verbose {
		fmt.Printf("🌐 Web services available for %s\n", name)
		if len(instance.Services) > 0 {
			fmt.Printf("   Services will be tunneled automatically when accessed\n")
			fmt.Printf("   Use: prism workspace connect %s\n", name)
		}
	}
	return nil
}

// connectDesktopInstance handles DCV desktop connections
func (ic *InstanceCommands) connectDesktopInstance(instance *types.Instance, name string, verbose bool) error {
	fmt.Printf("🖥️  Connecting to desktop workspace: %s\n", name)

	// Check instance state
	if instance.State != "running" {
		return fmt.Errorf("workspace must be running to connect (current state: %s)", instance.State)
	}

	// Get DCV port from instance metadata (default to 8443)
	dcvPort := 8443
	if instance.WebPort != 0 {
		dcvPort = instance.WebPort
	}

	// Start SSM port forwarding
	localPort := 8443
	fmt.Printf("📡 Starting secure tunnel to DCV server (port %d → localhost:%d)...\n", dcvPort, localPort)

	// Display connection details if verbose
	if verbose {
		fmt.Printf("\n🔧 Connection Details:\n")
		fmt.Printf("   Remote DCV Port: %d\n", dcvPort)
		fmt.Printf("   Local Port: %d\n", localPort)
		fmt.Printf("   Instance ID: %s\n", instance.ID)
		fmt.Printf("   DCV URL: https://localhost:%d\n", localPort)
	}

	// Start SSM port forwarding session in background
	if err := ic.startSSMPortForwarding(instance.ID, dcvPort, localPort, verbose); err != nil {
		return fmt.Errorf("failed to start SSM port forwarding: %w", err)
	}

	// Wait a moment for tunnel to establish
	fmt.Printf("⏳ Waiting for tunnel to establish...\n")
	time.Sleep(2 * time.Second)

	// Display credentials
	fmt.Printf("\n🔑 DCV Connection Credentials:\n")
	fmt.Printf("   Username: %s\n", instance.Username)
	if instance.DCVPassword != "" {
		fmt.Printf("   Password: %s\n", instance.DCVPassword)
	} else {
		fmt.Printf("   Password: (check ~/.prism/state.json — dcv_password field for this instance)\n")
	}

	// Open browser to DCV
	dcvURL := fmt.Sprintf("https://localhost:%d", localPort)
	fmt.Printf("\n🌐 Opening DCV session in browser...\n")
	fmt.Printf("   URL: %s\n", dcvURL)

	// Open browser
	if err := openBrowser(dcvURL); err != nil {
		fmt.Printf("⚠️  Could not auto-open browser: %v\n", err)
		fmt.Printf("   Please open manually: %s\n", dcvURL)
	} else {
		fmt.Printf("✅ Browser opened\n")
	}

	// Display usage instructions
	fmt.Printf("\n💡 Connection Instructions:\n")
	fmt.Printf("   1. Your browser will warn about self-signed certificate — click 'Advanced' and proceed\n")
	fmt.Printf("   2. Login with username: %s and the password shown above\n", instance.Username)
	fmt.Printf("   3. Your desktop environment will appear in the browser window\n")
	fmt.Printf("   4. The tunnel will stay open until you close it with Ctrl+C\n")

	if verbose {
		fmt.Printf("\n📚 DCV Features:\n")
		fmt.Printf("   • Browser-based remote desktop (no client needed)\n")
		fmt.Printf("   • Full desktop environment with GPU support\n")
		fmt.Printf("   • Copy/paste between local and remote\n")
		fmt.Printf("   • File transfer via browser\n")
	}

	// Keep the connection open
	fmt.Printf("\n🔒 Tunnel is active. Press Ctrl+C to disconnect...\n")

	// Block until user interrupts (Ctrl+C)
	// The SSM session will be terminated when the process exits
	select {}
}

// startSSMPortForwarding starts an AWS SSM port forwarding session in the current process
func (ic *InstanceCommands) startSSMPortForwarding(instanceID string, remotePort, localPort int, verbose bool) error {
	// Build SSM parameters JSON
	parameters := fmt.Sprintf(`{"portNumber":["%d"],"localPortNumber":["%d"]}`, remotePort, localPort)

	// Build AWS SSM command
	cmd := exec.Command("aws", "ssm", "start-session", //nolint:gosec // G204: aws CLI, args are argv not shell
		"--target", instanceID,
		"--document-name", "AWS-StartPortForwardingSession",
		"--parameters", parameters,
	)

	if verbose {
		fmt.Printf("🔧 SSM Command: %s %s\n", cmd.Path, strings.Join(cmd.Args[1:], " "))
	}

	// Set up output
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// Start the command
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start SSM session: %w", err)
	}

	// Wait for the command to complete (this will block until Ctrl+C)
	go func() {
		if err := cmd.Wait(); err != nil {
			// Only print error if it's not due to user interrupt
			if !strings.Contains(err.Error(), "signal: interrupt") {
				fmt.Printf("\n⚠️  SSM session ended: %v\n", err)
			}
		}
	}()

	fmt.Printf("✅ SSM tunnel started\n")
	return nil
}

// Stop handles the stop command
func (ic *InstanceCommands) Stop(args []string) error {
	if len(args) < 1 {
		return NewUsageError("prism workspace stop <name>", "prism workspace stop my-workspace")
	}

	name := args[0]

	// Ensure daemon is running (auto-start if needed)
	if err := ic.app.ensureDaemonRunning(); err != nil {
		return err
	}

	err := ic.app.apiClient.StopInstance(ic.app.ctx, name)
	if err != nil {
		return WrapAPIError("stop workspace "+name, err)
	}

	fmt.Printf("%s\n", FormatProgressMessage("Stopping workspace", name))
	return nil
}

// Start handles the start command with intelligent state management
func (ic *InstanceCommands) Start(args []string) error {
	if len(args) < 1 {
		return NewUsageError("prism workspace start <name>", "prism workspace start my-workspace")
	}

	name := args[0]

	// Ensure daemon is running (auto-start if needed)
	if err := ic.app.ensureDaemonRunning(); err != nil {
		return err
	}

	// First, get current instance status
	listResponse, err := ic.app.apiClient.ListInstances(ic.app.ctx)
	if err != nil {
		return WrapAPIError("get workspace status", err)
	}

	var targetInstance *types.Instance
	for _, instance := range listResponse.Instances {
		if instance.Name == name {
			targetInstance = &instance
			break
		}
	}

	if targetInstance == nil {
		return NewNotFoundError("workspace", name, "Use 'prism workspace list' to see available instances")
	}

	// Check current state and handle appropriately
	switch targetInstance.State {
	case "running":
		fmt.Printf("✅ Workspace %s is already running\n", name)
		return nil
	case "hibernated":
		fmt.Printf("🛌 Workspace %s is hibernated - use 'prism workspace resume %s' for instant startup\n", name, name)
		fmt.Printf("   Or use 'prism workspace start %s' for regular boot (slower)\n", name)
		fmt.Printf("   Proceeding with regular start...\n")
	case "stopped", "stopping":
		// Normal case - proceed with start
	default:
		fmt.Printf("⚠️  Workspace %s is in state '%s' - attempting to start anyway\n", name, targetInstance.State)
	}

	err = ic.app.apiClient.StartInstance(ic.app.ctx, name)
	if err != nil {
		return WrapAPIError("start workspace "+name, err)
	}

	fmt.Printf("%s\n", FormatProgressMessage("Starting workspace", name))
	return nil
}

// Delete handles the delete command
func (ic *InstanceCommands) Delete(args []string) error {
	if len(args) < 1 {
		return NewUsageError("prism workspace delete <name>", "prism workspace delete my-workspace")
	}

	name := args[0]

	// Ensure daemon is running (auto-start if needed)
	if err := ic.app.ensureDaemonRunning(); err != nil {
		return err
	}

	err := ic.app.apiClient.DeleteInstance(ic.app.ctx, name)
	if err != nil {
		return WrapAPIError("delete workspace "+name, err)
	}

	fmt.Printf("%s\n", FormatProgressMessage("Deleting workspace", name))
	return nil
}

// Hibernate handles the hibernate command
func (ic *InstanceCommands) Hibernate(args []string) error {
	if len(args) < 1 {
		return NewUsageError("prism workspace hibernate <name>", "prism workspace hibernate my-workspace")
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
		fmt.Printf("⚠️  Workspace %s does not support EC2 hibernation\n", name)
		fmt.Printf("    Falling back to regular stop operation\n")
	}

	err = ic.app.apiClient.HibernateInstance(ic.app.ctx, name)
	if err != nil {
		return WrapAPIError("hibernate workspace "+name, err)
	}

	if status.HibernationSupported {
		fmt.Printf("%s\n", FormatProgressMessage("Hibernating workspace", name))
		fmt.Printf("   %s\n", FormatInfoMessage("RAM state preserved for instant resume"))
		fmt.Printf("   💰 Compute billing stopped, storage billing continues\n")
	} else {
		fmt.Printf("%s\n", FormatProgressMessage("Stopping workspace", name))
		fmt.Printf("   %s\n", FormatInfoMessage("Consider using EC2 hibernation-capable instance types for RAM preservation"))
	}

	return nil
}

// Resume handles the resume command
func (ic *InstanceCommands) Resume(args []string) error {
	if len(args) < 1 {
		return NewUsageError("prism workspace resume <name>", "prism workspace resume my-workspace")
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
			return WrapAPIError("resume workspace "+name, err)
		}
		fmt.Printf("%s\n", FormatProgressMessage("Resuming hibernated workspace", name))
		fmt.Printf("   🚀 Instant startup from preserved RAM state\n")
	} else {
		// Fall back to regular start
		err = ic.app.apiClient.StartInstance(ic.app.ctx, name)
		if err != nil {
			return WrapAPIError("start workspace "+name, err)
		}
		fmt.Printf("%s\n", FormatProgressMessage("Starting workspace", name))
		fmt.Printf("   %s\n", FormatInfoMessage("Workspace was not hibernated - performing regular start"))
	}

	return nil
}

// Exec handles the exec command - executes commands remotely on instances
// Note: This method is called from the Cobra command structure, so flag parsing
// is handled by Cobra. This simplified version assumes args contains only positional arguments.
func (ic *InstanceCommands) Exec(args []string) error {
	// Validate arguments
	if len(args) < 2 {
		return NewUsageError("prism workspace exec <workspace-name> <command>", "prism workspace exec my-workspace \"ls -la\"")
	}

	// Parse command arguments and flags
	instanceName := args[0]
	command := args[1]
	execRequest, verbose := ic.parseExecFlags(args[2:], command)

	// Ensure daemon is running
	if err := ic.app.ensureDaemonRunning(); err != nil {
		return err
	}

	// Display verbose execution info
	if verbose {
		ic.displayExecInfo(instanceName, execRequest)
	}

	// Execute command and display results
	return ic.executeAndDisplayResult(instanceName, execRequest, verbose)
}

// parseExecFlags parses exec command flags
func (ic *InstanceCommands) parseExecFlags(args []string, command string) (types.ExecRequest, bool) {
	// Design Note: This function supports both Cobra and direct API usage
	// When called via Cobra command (instance_cobra.go), flags are pre-parsed by Cobra
	// When called directly (legacy/API usage), manual flag parsing below handles arguments
	execRequest := types.ExecRequest{
		Command:        command,
		TimeoutSeconds: 30,
		Environment:    make(map[string]string),
	}
	verbose := false

	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "--user" && i+1 < len(args):
			execRequest.User = args[i+1]
			i++
		case arg == "--working-dir" && i+1 < len(args):
			execRequest.WorkingDir = args[i+1]
			i++
		case arg == "--timeout" && i+1 < len(args):
			execRequest.TimeoutSeconds, _ = strconv.Atoi(args[i+1])
			i++
		case strings.HasPrefix(arg, "--env="):
			envPart := strings.TrimPrefix(arg, "--env=")
			if envKV := strings.SplitN(envPart, "=", 2); len(envKV) == 2 {
				execRequest.Environment[envKV[0]] = envKV[1]
			}
		case arg == "--interactive" || arg == "-i":
			execRequest.Interactive = true
		case arg == "--verbose" || arg == "-v":
			verbose = true
		}
	}

	return execRequest, verbose
}

// displayExecInfo displays execution information in verbose mode
func (ic *InstanceCommands) displayExecInfo(instanceName string, req types.ExecRequest) {
	fmt.Printf("🔧 Executing command on %s...\n", instanceName)
	fmt.Printf("   Command: %s\n", req.Command)
	if req.User != "" {
		fmt.Printf("   User: %s\n", req.User)
	}
	if req.WorkingDir != "" {
		fmt.Printf("   Working Directory: %s\n", req.WorkingDir)
	}
	if len(req.Environment) > 0 {
		fmt.Printf("   Environment: %v\n", req.Environment)
	}
	fmt.Printf("   Timeout: %ds\n", req.TimeoutSeconds)
	fmt.Printf("   Interactive: %t\n", req.Interactive)
	fmt.Println()
}

// executeAndDisplayResult executes command and displays results
func (ic *InstanceCommands) executeAndDisplayResult(instanceName string, req types.ExecRequest, verbose bool) error {
	result, err := ic.app.apiClient.ExecInstance(ic.app.ctx, instanceName, req)
	if err != nil {
		return WrapAPIError("execute command on "+instanceName, err)
	}

	ic.displayExecResult(result, verbose)

	if result.ExitCode != 0 {
		os.Exit(result.ExitCode)
	}

	return nil
}

// displayExecResult displays command execution results
func (ic *InstanceCommands) displayExecResult(result *types.ExecResult, verbose bool) {
	if verbose {
		ic.displayExecSummary(result)
	}

	ic.displayStdOut(result.StdOut, verbose)
	ic.displayStdErr(result.StdErr, result.ExitCode, verbose)
}

// displayExecSummary displays execution summary in verbose mode
func (ic *InstanceCommands) displayExecSummary(result *types.ExecResult) {
	fmt.Printf("📊 Command execution completed:\n")
	fmt.Printf("   Exit Code: %d\n", result.ExitCode)
	fmt.Printf("   Status: %s\n", result.Status)
	fmt.Printf("   Execution Time: %dms\n", result.ExecutionTime)
	if result.CommandID != "" {
		fmt.Printf("   Command ID: %s\n", result.CommandID)
	}
	fmt.Println()
}

// displayStdOut displays standard output
func (ic *InstanceCommands) displayStdOut(stdout string, verbose bool) {
	if stdout == "" {
		return
	}

	if verbose {
		fmt.Printf("📤 Output:\n")
	}
	fmt.Print(stdout)
	if !strings.HasSuffix(stdout, "\n") {
		fmt.Println()
	}
}

// displayStdErr displays standard error
func (ic *InstanceCommands) displayStdErr(stderr string, exitCode int, verbose bool) {
	if stderr == "" || (exitCode == 0 && !verbose) {
		return
	}

	if verbose {
		fmt.Printf("⚠️  Error Output:\n")
	}
	fmt.Fprint(os.Stderr, stderr)
	if !strings.HasSuffix(stderr, "\n") {
		fmt.Fprintln(os.Stderr)
	}
}

// Resize handles the resize command - changes instance type/size
func (ic *InstanceCommands) Resize(args []string) error {
	// Validate arguments
	if len(args) < 2 {
		return NewUsageError("prism resize <workspace-name> --size <size> [options]",
			"prism resize my-workspace --size L")
	}

	// Parse flags
	instanceName, resizeOpts, err := ic.parseResizeFlags(args)
	if err != nil {
		return err
	}

	// Ensure daemon is running
	if err := ic.app.ensureDaemonRunning(); err != nil {
		return err
	}

	// Get and validate instance
	targetInstance, err := ic.getInstanceForResize(instanceName)
	if err != nil {
		return err
	}

	// Determine target workspace type
	targetInstanceType, err := ic.resolveTargetInstanceType(resizeOpts)
	if err != nil {
		return err
	}

	// Display resize information and handle dry-run
	if err := ic.displayResizeInfo(instanceName, targetInstance, targetInstanceType, resizeOpts); err != nil {
		return err
	}

	// Execute resize
	return ic.executeResize(instanceName, targetInstanceType, resizeOpts)
}

// resizeOptions holds parsed resize command options
type resizeOptions struct {
	newSize      string
	instanceType string
	dryRun       bool
	force        bool
	wait         bool
}

// parseResizeFlags parses resize command flags
func (ic *InstanceCommands) parseResizeFlags(args []string) (string, resizeOptions, error) {
	instanceName := args[0]
	opts := resizeOptions{}

	for i := 1; i < len(args); i++ {
		switch args[i] {
		case "--size":
			if i+1 >= len(args) {
				return "", opts, NewValidationError("--size", "", "requires a t-shirt size (XS, S, M, L, XL)")
			}
			opts.newSize = strings.ToUpper(args[i+1])
			if !ValidTSizes[opts.newSize] {
				return "", opts, NewValidationError("size", opts.newSize, "XS, S, M, L, XL")
			}
			i++
		case "--instance-type":
			if i+1 >= len(args) {
				return "", opts, NewValidationError("--instance-type", "", "requires an AWS instance type")
			}
			opts.instanceType = args[i+1]
			i++
		case "--dry-run":
			opts.dryRun = true
		case "--force":
			opts.force = true
		case "--wait":
			opts.wait = true
		default:
			return "", opts, NewValidationError("flag", args[i], "--size, --instance-type, --dry-run, --force, or --wait")
		}
	}

	return instanceName, opts, nil
}

// getInstanceForResize retrieves and validates the target workspace
func (ic *InstanceCommands) getInstanceForResize(instanceName string) (*types.Instance, error) {
	listResponse, err := ic.app.apiClient.ListInstances(ic.app.ctx)
	if err != nil {
		return nil, WrapAPIError("get workspace status", err)
	}

	for _, instance := range listResponse.Instances {
		if instance.Name == instanceName {
			return &instance, nil
		}
	}

	return nil, NewNotFoundError("workspace", instanceName, "Use 'prism workspace list' to see available instances")
}

// resolveTargetInstanceType determines the target workspace type from options
func (ic *InstanceCommands) resolveTargetInstanceType(opts resizeOptions) (string, error) {
	if opts.instanceType != "" {
		return opts.instanceType, nil
	}

	if opts.newSize != "" {
		if mappedType, exists := SizeInstanceTypeMapping[opts.newSize]; exists {
			return mappedType, nil
		}
		return "", NewValidationError("size", opts.newSize, "valid t-shirt size (XS, S, M, L, XL)")
	}

	return "", NewUsageError("prism resize <workspace-name> --size <size> OR --instance-type <type>",
		"prism resize my-workspace --size L")
}

// displayResizeInfo displays resize operation details and handles validation
func (ic *InstanceCommands) displayResizeInfo(instanceName string, targetInstance *types.Instance, targetInstanceType string, opts resizeOptions) error {
	currentSize := ic.getCurrentSize(targetInstance.InstanceType)

	ic.displayResizeHeader(instanceName, targetInstance, targetInstanceType, currentSize, opts.newSize)

	if targetInstance.InstanceType == targetInstanceType {
		fmt.Printf("✅ Instance is already type %s. No resize needed.\n", targetInstanceType)
		return fmt.Errorf("no resize needed")
	}

	if targetInstance.State != "running" && targetInstance.State != "stopped" {
		return NewStateError("workspace", instanceName, targetInstance.State, "running or stopped")
	}

	ic.displayCostImpact(targetInstance, targetInstanceType)
	ic.displayResizeProcess(targetInstance)

	if opts.dryRun {
		fmt.Printf("🔍 **Dry Run Complete**\n")
		fmt.Printf("   Resize operation validated successfully\n")
		fmt.Printf("   Run without --dry-run to execute\n")
		return fmt.Errorf("dry-run complete")
	}

	return ic.confirmResize(instanceName, opts.force)
}

// getCurrentSize gets current t-shirt size for instance type
func (ic *InstanceCommands) getCurrentSize(instanceType string) string {
	if size, exists := InstanceTypeSizeMapping[instanceType]; exists {
		return size
	}
	return "Unknown"
}

// displayResizeHeader displays resize operation header
func (ic *InstanceCommands) displayResizeHeader(instanceName string, instance *types.Instance, targetType, currentSize, newSize string) {
	fmt.Printf("🔄 Instance Resize Operation\n")
	fmt.Printf("═══════════════════════════\n\n")
	fmt.Printf("📋 **Resize Details**:\n")
	fmt.Printf("   Instance: %s\n", instanceName)
	fmt.Printf("   Current Type: %s (%s)\n", instance.InstanceType, currentSize)
	fmt.Printf("   Target Type: %s", targetType)
	if newSize != "" {
		fmt.Printf(" (%s)", newSize)
	}
	fmt.Printf("\n   Current State: %s\n\n", instance.State)
}

// displayCostImpact displays cost comparison
func (ic *InstanceCommands) displayCostImpact(instance *types.Instance, targetType string) {
	currentCost := instance.HourlyRate
	newCost := ic.estimateCostForInstanceType(targetType)

	fmt.Printf("💰 **Cost Impact**:\n")
	fmt.Printf("   Current Cost: $%.2f/day\n", currentCost)
	fmt.Printf("   New Cost: $%.2f/day\n", newCost)

	if newCost > currentCost {
		diff := newCost - currentCost
		fmt.Printf("   Impact: +$%.2f/day (+%.1f%%)\n", diff, (diff/currentCost)*100)
		fmt.Printf("   Monthly Impact: +$%.2f\n", diff*30)
	} else if newCost < currentCost {
		diff := currentCost - newCost
		fmt.Printf("   Impact: -$%.2f/day (-%.1f%%)\n", diff, (diff/currentCost)*100)
		fmt.Printf("   Monthly Savings: $%.2f\n", diff*30)
	} else {
		fmt.Printf("   Impact: No cost change\n")
	}
	fmt.Println()
}

// displayResizeProcess displays resize process steps
func (ic *InstanceCommands) displayResizeProcess(instance *types.Instance) {
	fmt.Printf("⚡ **Resize Process**:\n")
	if instance.State == "running" {
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
}

// confirmResize prompts for user confirmation unless forced
func (ic *InstanceCommands) confirmResize(instanceName string, force bool) error {
	if force {
		return nil
	}

	fmt.Printf("⚠️  **Confirmation Required**\n")
	fmt.Printf("   This will modify the instance type and require a restart.\n")
	fmt.Printf("   Type the instance name to confirm: ")

	var confirmation string
	_, _ = fmt.Scanln(&confirmation) // empty or invalid input leaves confirmation unmatched, which is treated as "no"

	if confirmation != instanceName {
		fmt.Printf("❌ Instance name doesn't match. Resize cancelled.\n")
		return fmt.Errorf("confirmation failed")
	}

	return nil
}

// executeResize executes the resize operation
func (ic *InstanceCommands) executeResize(instanceName, targetType string, opts resizeOptions) error {
	resizeRequest := types.ResizeRequest{
		InstanceName:       instanceName,
		TargetInstanceType: targetType,
		Force:              opts.force,
		Wait:               opts.wait,
	}

	response, err := ic.app.apiClient.ResizeInstance(ic.app.ctx, resizeRequest)
	if err != nil {
		return WrapAPIError("resize instance "+instanceName, err)
	}

	fmt.Printf("✅ %s\n", response.Message)

	if opts.wait {
		fmt.Printf("⏳ Monitoring resize progress...\n")
		return ic.monitorResizeProgress(instanceName)
	}

	fmt.Printf("💡 Monitor progress with: prism workspace list\n")
	fmt.Printf("💡 Check when ready: prism workspace connect %s\n", instanceName)
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
			fmt.Printf("⚠️  Unable to get workspace status: %v\n", err)
			return nil
		}

		switch instance.State {
		case "running":
			fmt.Printf("✅ Resize complete! Instance is running with new configuration.\n")
			fmt.Printf("🔗 Connect: prism workspace connect %s\n", instanceName)
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
	fmt.Printf("💡 Check status with: prism workspace list\n")
	return nil
}
