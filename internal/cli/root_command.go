package cli

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/scttfrdmn/cloudworkstation/pkg/version"
	"github.com/spf13/cobra"
)

// CommandFactory interface for creating specialized commands (Factory Pattern - SOLID)
type CommandFactory interface {
	CreateCommand() *cobra.Command
}

// LaunchCommandFactory creates the launch command
type LaunchCommandFactory struct {
	app *App
}

func (f *LaunchCommandFactory) CreateCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "launch <template> <name>",
		Short:   "Launch a new cloud workstation",
		Long:    `Launch a new cloud workstation from a template with smart defaults.`,
		Args:    cobra.MinimumNArgs(2),
		GroupID: "core",
		RunE: func(cmd *cobra.Command, args []string) error {
			return f.buildLaunchArgs(cmd, args)
		},
	}
	f.addLaunchFlags(cmd)
	return cmd
}

func (f *LaunchCommandFactory) buildLaunchArgs(cmd *cobra.Command, args []string) error {
	if hibernation, _ := cmd.Flags().GetBool("hibernation"); hibernation {
		args = append(args, "--hibernation")
	}
	if spot, _ := cmd.Flags().GetBool("spot"); spot {
		args = append(args, "--spot")
	}
	if size, _ := cmd.Flags().GetString("size"); size != "" {
		args = append(args, "--size", size)
	}
	if subnet, _ := cmd.Flags().GetString("subnet"); subnet != "" {
		args = append(args, "--subnet", subnet)
	}
	if vpc, _ := cmd.Flags().GetString("vpc"); vpc != "" {
		args = append(args, "--vpc", vpc)
	}
	if project, _ := cmd.Flags().GetString("project"); project != "" {
		args = append(args, "--project", project)
	}
	if wait, _ := cmd.Flags().GetBool("wait"); wait {
		args = append(args, "--wait")
	}
	if dryRun, _ := cmd.Flags().GetBool("dry-run"); dryRun {
		args = append(args, "--dry-run")
	}
	if params, _ := cmd.Flags().GetStringArray("param"); len(params) > 0 {
		for _, param := range params {
			args = append(args, "--param", param)
		}
	}
	if researchUser, _ := cmd.Flags().GetString("research-user"); researchUser != "" {
		args = append(args, "--research-user", researchUser)
	}
	return f.app.Launch(args)
}

func (f *LaunchCommandFactory) addLaunchFlags(cmd *cobra.Command) {
	cmd.Flags().Bool("hibernation", false, "Enable hibernation support")
	cmd.Flags().Bool("spot", false, "Use spot workspaces")
	cmd.Flags().String("size", "", "Workspace size: XS=1vCPU,2GB+100GB | S=2vCPU,4GB+500GB | M=2vCPU,8GB+1TB | L=4vCPU,16GB+2TB | XL=8vCPU,32GB+4TB")
	cmd.Flags().String("subnet", "", "Specify subnet ID")
	cmd.Flags().String("vpc", "", "Specify VPC ID")
	cmd.Flags().String("project", "", "Associate with project")
	cmd.Flags().Bool("wait", false, "Wait and display launch progress in real-time")
	cmd.Flags().Bool("dry-run", false, "Validate configuration without launching")
	cmd.Flags().StringArray("param", []string{}, "Template parameter in format name=value")
	cmd.Flags().String("research-user", "", "Automatically create and provision research user on workspace")
}

// InstanceCommandFactory creates workspace management commands
type InstanceCommandFactory struct {
	app *App
}

func (f *InstanceCommandFactory) CreateCommands() []*cobra.Command {
	return []*cobra.Command{
		f.createConnectCommand(),
		f.createExecCommand(),
		f.createStopCommand(),
		f.createStartCommand(),
		f.createDeleteCommand(),
		f.createHibernateCommand(),
		f.createResumeCommand(),
		f.createResizeCommand(),
	}
}

func (f *InstanceCommandFactory) createConnectCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "connect <name>",
		Short:   "Connect to a workstation",
		Long:    `Get connection information for a cloud workstation.`,
		Args:    cobra.ExactArgs(1),
		GroupID: "core",
		RunE: func(cmd *cobra.Command, args []string) error {
			verbose, _ := cmd.Flags().GetBool("verbose")
			user, _ := cmd.Flags().GetString("user")
			if verbose {
				args = append(args, "--verbose")
			}
			if user != "" {
				args = append(args, "--user", user)
			}
			return f.app.Connect(args)
		},
	}
	cmd.Flags().BoolP("verbose", "v", false, "Show SSH connection command without executing")
	cmd.Flags().StringP("user", "u", "", "Override SSH username (e.g., ubuntu, rstats)")
	return cmd
}

func (f *InstanceCommandFactory) createExecCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "exec <workspace-name> <command>",
		Short:   "Execute a command on a workspace",
		GroupID: "instance",
		Long: `Execute a command remotely on a cloud workspace via AWS Systems Manager.

This command provides powerful remote execution capabilities with support for:
• Custom user execution (--user flag)
• Working directory specification (--working-dir flag)
• Environment variable setting (--env flag)
• Command timeout configuration (--timeout flag)
• Verbose output and execution details (--verbose flag)

Examples:
  cws exec my-workspace "ls -la"                    # List directory contents
  cws exec my-workspace "python script.py" --user researcher --timeout 60
  cws exec my-workspace "cd /data && df -h" --working-dir /data
  cws exec my-workspace "export VAR=value && echo $VAR" --env=VAR=value`,
		Args: cobra.MinimumNArgs(2),
		RunE: func(_ *cobra.Command, args []string) error {
			return f.app.Exec(args)
		},
	}

	// Add flags for exec command
	cmd.Flags().String("user", "", "Execute command as specific user")
	cmd.Flags().String("working-dir", "", "Set working directory for command execution")
	cmd.Flags().Int("timeout", 30, "Command timeout in seconds")
	cmd.Flags().StringArray("env", []string{}, "Set environment variables (format: KEY=VALUE)")
	cmd.Flags().BoolP("interactive", "i", false, "Interactive execution mode")
	cmd.Flags().BoolP("verbose", "v", false, "Show verbose execution details")

	return cmd
}

func (f *InstanceCommandFactory) createStopCommand() *cobra.Command {
	return &cobra.Command{
		Use:     "stop <name>",
		Short:   "Stop a workspace",
		Long:    `Stop a running cloud workspace to save costs.`,
		Args:    cobra.ExactArgs(1),
		GroupID: "instance",
		RunE: func(_ *cobra.Command, args []string) error {
			return f.app.Stop(args)
		},
	}
}

func (f *InstanceCommandFactory) createStartCommand() *cobra.Command {
	return &cobra.Command{
		Use:     "start <name>",
		Short:   "Start a workspace",
		Long:    `Start a stopped cloud workspace.`,
		Args:    cobra.ExactArgs(1),
		GroupID: "instance",
		RunE: func(_ *cobra.Command, args []string) error {
			return f.app.Start(args)
		},
	}
}

func (f *InstanceCommandFactory) createDeleteCommand() *cobra.Command {
	return &cobra.Command{
		Use:     "delete <name>",
		Short:   "Delete a workspace",
		Long:    `Permanently delete a cloud workspace and its resources.`,
		Args:    cobra.ExactArgs(1),
		GroupID: "instance",
		RunE: func(_ *cobra.Command, args []string) error {
			return f.app.Delete(args)
		},
	}
}

func (f *InstanceCommandFactory) createHibernateCommand() *cobra.Command {
	return &cobra.Command{
		Use:     "hibernate <name>",
		Short:   "Hibernate a workspace",
		GroupID: "instance",
		Long: `Hibernate a running workspace to preserve RAM state while stopping compute billing.
If hibernation is not supported, automatically falls back to regular stop.`,
		Args: cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			return f.app.Hibernate(args)
		},
	}
}

func (f *InstanceCommandFactory) createResumeCommand() *cobra.Command {
	return &cobra.Command{
		Use:     "resume <name>",
		Short:   "Resume a workspace",
		GroupID: "instance",
		Long: `Resume a hibernated workspace with instant startup from preserved RAM state.
If not hibernated, performs regular start operation.`,
		Args: cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			return f.app.Resume(args)
		},
	}
}

func (f *InstanceCommandFactory) createResizeCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "resize <name>",
		Short:   "Resize a workspace type or size",
		GroupID: "instance",
		Long: `Resize a cloud workspace to change its instance type, CPU, memory, or storage.

This command provides flexible resizing capabilities with support for:
• T-shirt sizes (--size XS, S, M, L, XL) for simple scaling
• Direct instance type specification (--instance-type c5.large)
• Dry-run preview of resize operations (--dry-run)
• Force execution without confirmation prompts (--force)
• Wait for resize completion with progress monitoring (--wait)

The resize operation requires instance shutdown and will cause 2-5 minutes of downtime.
All data and configuration are preserved during the resize operation.

Examples:
  cws resize my-workspace --size L                  # Resize to Large t-shirt size
  cws resize gpu-training --instance-type p3.2xlarge # Resize to specific GPU instance
  cws resize my-analysis --size XL --dry-run       # Preview resize to Extra Large
  cws resize my-server --size M --wait             # Resize and wait for completion`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			instanceCommands := NewInstanceCommands(f.app)

			// Convert Cobra flags to args format expected by Resize method
			resizeArgs := []string{args[0]} // instance name

			if size, _ := cmd.Flags().GetString("size"); size != "" {
				resizeArgs = append(resizeArgs, "--size", size)
			}
			if instanceType, _ := cmd.Flags().GetString("instance-type"); instanceType != "" {
				resizeArgs = append(resizeArgs, "--instance-type", instanceType)
			}
			if dryRun, _ := cmd.Flags().GetBool("dry-run"); dryRun {
				resizeArgs = append(resizeArgs, "--dry-run")
			}
			if force, _ := cmd.Flags().GetBool("force"); force {
				resizeArgs = append(resizeArgs, "--force")
			}
			if wait, _ := cmd.Flags().GetBool("wait"); wait {
				resizeArgs = append(resizeArgs, "--wait")
			}

			return instanceCommands.Resize(resizeArgs)
		},
	}

	// Add resize-specific flags
	cmd.Flags().String("size", "", "T-shirt size: XS, S, M, L, XL")
	cmd.Flags().String("instance-type", "", "AWS instance type (e.g., c5.large, m5.xlarge)")
	cmd.Flags().Bool("dry-run", false, "Preview resize operation without executing")
	cmd.Flags().Bool("force", false, "Skip confirmation prompts")
	cmd.Flags().Bool("wait", false, "Wait for resize completion with progress monitoring")

	return cmd
}

// TemplateCommandFactory creates template management commands
type TemplateCommandFactory struct {
	app *App
}

func (f *TemplateCommandFactory) CreateCommands() []*cobra.Command {
	return []*cobra.Command{
		f.createTemplatesCommand(),
		f.createApplyCommand(),
		f.createDiffCommand(),
		f.createLayersCommand(),
		f.createRollbackCommand(),
	}
}

func (f *TemplateCommandFactory) createTemplatesCommand() *cobra.Command {
	// Use the new Cobra-based templates command with proper subcommands
	templateCobra := NewTemplateCobraCommands(f.app)
	return templateCobra.CreateTemplatesCommand()
}

func (f *TemplateCommandFactory) createApplyCommand() *cobra.Command {
	return &cobra.Command{
		Use:     "apply <template> <workspace-name>",
		Short:   "Apply template to running workspace",
		GroupID: "templates",
		Long: `Apply a template to an already running workspace, enabling incremental
environment evolution without requiring workspace recreation.

This allows you to add packages, services, and users to existing workspaces
while maintaining rollback capabilities.`,
		Args: cobra.MinimumNArgs(2),
		RunE: func(_ *cobra.Command, args []string) error {
			return f.app.Apply(args)
		},
	}
}

func (f *TemplateCommandFactory) createDiffCommand() *cobra.Command {
	return &cobra.Command{
		Use:     "diff <template> <workspace-name>",
		Short:   "Show template differences",
		GroupID: "templates",
		Long: `Show what changes would be made when applying a template to a running workspace.
This provides a preview of packages, services, users, and ports that would be modified.`,
		Args: cobra.ExactArgs(2),
		RunE: func(_ *cobra.Command, args []string) error {
			return f.app.Diff(args)
		},
	}
}

func (f *TemplateCommandFactory) createLayersCommand() *cobra.Command {
	return &cobra.Command{
		Use:     "layers <workspace-name>",
		Short:   "List applied template layers",
		GroupID: "templates",
		Long: `List all templates that have been applied to a workspace, showing the
layer history with rollback checkpoints.`,
		Args: cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			return f.app.Layers(args)
		},
	}
}

func (f *TemplateCommandFactory) createRollbackCommand() *cobra.Command {
	return &cobra.Command{
		Use:     "rollback <workspace-name>",
		Short:   "Rollback template applications",
		GroupID: "templates",
		Long: `Rollback a workspace to a previous state by undoing template applications.
Can rollback to the previous checkpoint or a specific checkpoint ID.`,
		Args: cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			return f.app.Rollback(args)
		},
	}
}

// CommandFactoryRegistry manages all command factories (Factory Pattern - SOLID)
type CommandFactoryRegistry struct {
	app *App
}

// NewCommandFactoryRegistry creates command factory registry
func NewCommandFactoryRegistry(app *App) *CommandFactoryRegistry {
	return &CommandFactoryRegistry{app: app}
}

// RegisterAllCommands adds all commands to root using factories
func (r *CommandFactoryRegistry) RegisterAllCommands(rootCmd *cobra.Command) {
	// NEW: Unified workspace command group (Phase 5.0.3 - CLI Consistency)
	workspaceFactory := &WorkspaceCommandFactory{app: r.app}
	rootCmd.AddCommand(workspaceFactory.CreateCommand())

	// Logs command
	logsCommands := NewLogsCommands(r.app)
	rootCmd.AddCommand(logsCommands.CreateLogsCommand())

	// Template commands
	templateFactory := &TemplateCommandFactory{app: r.app}
	for _, cmd := range templateFactory.CreateCommands() {
		rootCmd.AddCommand(cmd)
	}

	// Idle commands (using new Cobra structure)
	idleCobra := NewIdleCobraCommands(r.app)
	rootCmd.AddCommand(idleCobra.CreateIdleCommand())

	// Project commands (using new Cobra structure)
	projectCobra := NewProjectCobraCommands(r.app)
	rootCmd.AddCommand(projectCobra.CreateProjectCommand())

	// Budget commands (comprehensive financial management)
	budgetCommands := NewBudgetCommands(r.app)
	rootCmd.AddCommand(budgetCommands.CreateBudgetCommand())

	// Research User commands (Phase 5A Multi-User Foundation)
	researchUserCobra := NewResearchUserCobraCommands(r.app)
	rootCmd.AddCommand(researchUserCobra.CreateResearchUserCommand())

	// User commands are now handled via research-user command above
	// Admin commands are now handled via daemon command above

	// Profile commands (user-friendly interface)
	AddProfileCommands(rootCmd, r.app.config)

	// Keys commands (SSH key management)
	keysCobra := NewKeysCobraCommands(r.app)
	rootCmd.AddCommand(keysCobra.CreateKeysCommand())

	// Policy commands moved to admin

	// Storage commands (using proper Cobra structure)
	storageCobra := NewStorageCobraCommands(r.app)
	rootCmd.AddCommand(storageCobra.CreateVolumeCommand())
	rootCmd.AddCommand(storageCobra.CreateStorageCommand())

	// Snapshot commands
	rootCmd.AddCommand(r.createSnapshotCommand())

	// Backup and Restore commands
	rootCmd.AddCommand(r.createBackupCommand())
	rootCmd.AddCommand(r.createRestoreCommand())

	// Web Services command
	rootCmd.AddCommand(r.createWebCommand())

	// System commands (kept at root level)
	rootCmd.AddCommand(r.app.tuiCommand)
	rootCmd.AddCommand(NewGUICommand())
	rootCmd.AddCommand(NewAboutCommand())

	// Other commands (removed duplicate idle command - using Cobra version instead)

	// AMI commands (using new Cobra structure)
	amiCobra := NewAMICobraCommands(r.app)
	rootCmd.AddCommand(amiCobra.CreateAMICommand())
	rootCmd.AddCommand(r.createAMIDiscoverCommand()) // Keep legacy ami-discover for now

	// Marketplace commands (using new Cobra structure)
	marketplaceCobra := NewMarketplaceCobraCommands(r.app)
	rootCmd.AddCommand(marketplaceCobra.CreateMarketplaceCommand())

	// Repository commands (using new Cobra structure)
	repoCobra := NewRepoCobraCommands(r.app)
	rootCmd.AddCommand(repoCobra.CreateRepoCommand())

	// NEW: Unified admin command group (Phase 5.0.3 - CLI Consistency)
	adminFactory := &AdminCommandFactory{app: r.app}
	rootCmd.AddCommand(adminFactory.CreateCommand())
}

func (r *CommandFactoryRegistry) createListCommand() *cobra.Command {
	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List workstations",
		Long: `List all your cloud workstations and their status from local cache (fast).
Use --refresh to query AWS for real-time status (slower).`,
		GroupID: "core",
		RunE: func(cmd *cobra.Command, args []string) error {
			// Get flag values and prepend to args for App.List() to parse
			detailed, _ := cmd.Flags().GetBool("detailed")
			refresh, _ := cmd.Flags().GetBool("refresh")
			if detailed {
				args = append([]string{"--detailed"}, args...)
			}
			if refresh {
				args = append([]string{"--refresh"}, args...)
			}
			return r.app.List(args)
		},
	}

	// Add flags
	listCmd.Flags().BoolP("detailed", "d", false, "Show detailed information including region and availability zone")
	listCmd.Flags().BoolP("refresh", "r", false, "Refresh from AWS for real-time status (slower)")

	listCostCmd := &cobra.Command{
		Use:   "cost",
		Short: "Show detailed cost information",
		Long:  `Show detailed cost information for all workstations with four decimal precision.`,
		RunE: func(_ *cobra.Command, args []string) error {
			return r.app.ListCost(args)
		},
	}
	listCmd.AddCommand(listCostCmd)
	return listCmd
}

func (r *CommandFactoryRegistry) createSnapshotCommand() *cobra.Command {
	return &cobra.Command{
		Use:     "snapshot <action>",
		Short:   "Manage workspace snapshots",
		GroupID: "storage",
		Long: `Create and manage CloudWorkstation workspace snapshots for backup, cloning, and disaster recovery.

Snapshots capture the complete state of your workspaces including:
• Operating system and all installed software
• User data and configuration files
• Template metadata for easy restoration

Examples:
  cws snapshot create my-workspace backup-v1
  cws snapshot list
  cws snapshot restore backup-v1 my-new-workspace`,
		RunE: func(_ *cobra.Command, args []string) error {
			return r.app.Snapshot(args)
		},
	}
}

func (r *CommandFactoryRegistry) createBackupCommand() *cobra.Command {
	return &cobra.Command{
		Use:     "backup <action>",
		Short:   "Manage data backups",
		GroupID: "storage",
		Long: `Create and manage CloudWorkstation data backups for user files, configurations, and research data.

Data backups provide granular backup capabilities with:
• Selective file and directory backup
• Incremental backup support
• Multiple storage options (S3, EFS, EBS)
• Compression and encryption
• Cost-effective storage with deduplication

Examples:
  cws backup create my-workspace daily-backup
  cws backup list
  cws backup restore daily-backup target-workspace`,
		RunE: func(_ *cobra.Command, args []string) error {
			return r.app.Backup(args)
		},
	}
}

func (r *CommandFactoryRegistry) createRestoreCommand() *cobra.Command {
	return &cobra.Command{
		Use:     "restore <backup-name> <target-workspace>",
		Short:   "Restore data from backups",
		GroupID: "storage",
		Long: `Restore data from CloudWorkstation backups with granular control over restore operations.

Restore capabilities include:
• Cross-workspace restoration
• Selective file/directory restoration
• Multiple restore modes (safe, merge, overwrite)
• Integrity verification
• Progress monitoring and dry-run preview

Examples:
  cws restore daily-backup my-workspace
  cws restore daily-backup my-workspace --path /data --selective /home/user
  cws restore daily-backup my-workspace --dry-run`,
		RunE: func(_ *cobra.Command, args []string) error {
			return r.app.Restore(args)
		},
	}
}

func (r *CommandFactoryRegistry) createWebCommand() *cobra.Command {
	return &cobra.Command{
		Use:     "web <action>",
		Short:   "Manage workspace web services",
		GroupID: "instance",
		Long: `Access and manage web services running on CloudWorkstation workspaces.

Web service management provides seamless access to:
• Jupyter Lab and Jupyter Notebook
• RStudio Server
• Shiny Server
• Custom web applications

Examples:
  cws web list my-jupyter         # List all web services for workspace
  cws web open my-jupyter jupyter # Open Jupyter in browser with auto-tunneling
  cws web close my-jupyter         # Close all tunnels for workspace
  cws web close my-jupyter jupyter # Close specific service tunnel`,
		RunE: func(_ *cobra.Command, args []string) error {
			return r.app.Web(args)
		},
	}
}

func (r *CommandFactoryRegistry) createDaemonCommand() *cobra.Command {
	return &cobra.Command{
		Use:     "daemon <action>",
		Short:   "Manage the daemon",
		GroupID: "system",
		Long:    `Control the CloudWorkstation daemon process.`,
		RunE: func(_ *cobra.Command, args []string) error {
			return r.app.Daemon(args)
		},
	}
}

func (r *CommandFactoryRegistry) createUninstallCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "uninstall",
		Short: "Uninstall CloudWorkstation completely",
		Long: `Completely uninstall CloudWorkstation from your system.
		
This command performs comprehensive cleanup including:
• Stop all running daemon processes
• Remove all configuration files and data
• Clean up log files and temporary data
• Remove service files and system integrations

Use with caution - this will remove ALL CloudWorkstation data.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return r.handleUninstallCommand(cmd, args)
		},
	}
}

func (r *CommandFactoryRegistry) handleUninstallCommand(cmd *cobra.Command, args []string) error {
	fmt.Println("🗑️  CloudWorkstation Uninstaller")
	fmt.Println("=================================")
	fmt.Println()
	fmt.Println("⚠️  This will completely remove CloudWorkstation from your system!")
	fmt.Println()
	fmt.Println("The following will be removed:")
	fmt.Println("  • All daemon processes")
	fmt.Println("  • Configuration files (~/.cloudworkstation)")
	fmt.Println("  • Log files and temporary data")
	fmt.Println("  • Service files and system integrations")
	fmt.Println()
	fmt.Println("🔒 AWS credentials and profiles will remain unchanged")
	fmt.Println()

	// Confirmation
	fmt.Print("Are you sure you want to completely uninstall CloudWorkstation? [y/N]: ")
	var response string
	_, _ = fmt.Scanln(&response) // Error ignored - user input validation happens below

	if response != "y" && response != "Y" && response != "yes" {
		fmt.Println("❌ Uninstallation cancelled")
		return nil
	}

	fmt.Println()
	fmt.Println("🚀 Starting uninstallation...")

	// Find script path
	scriptPath, err := r.findUninstallScript()
	if err != nil {
		fmt.Printf("⚠️  Uninstall script not found: %v\n", err)
		fmt.Println("🔧 Falling back to manual cleanup...")
		return r.performManualCleanup()
	}

	// Execute uninstall script
	fmt.Printf("📜 Executing uninstall script: %s\n", scriptPath)
	execCmd := exec.Command("bash", scriptPath, "--force")
	execCmd.Stdout = os.Stdout
	execCmd.Stderr = os.Stderr

	if err := execCmd.Run(); err != nil {
		fmt.Printf("⚠️  Uninstall script failed: %v\n", err)
		fmt.Println("🔧 Falling back to manual cleanup...")
		return r.performManualCleanup()
	}

	fmt.Println()
	fmt.Println("✅ CloudWorkstation has been successfully uninstalled!")
	fmt.Println("   Thank you for using CloudWorkstation! 👋")

	return nil
}

func (r *CommandFactoryRegistry) findUninstallScript() (string, error) {
	// Try to find the uninstall script in various locations
	candidates := []string{
		"./scripts/uninstall-manager.sh",
		"../scripts/uninstall-manager.sh",
		"/usr/local/share/cloudworkstation/uninstall-manager.sh",
		"/opt/homebrew/share/cloudworkstation/uninstall-manager.sh",
	}

	for _, path := range candidates {
		if _, err := os.Stat(path); err == nil {
			return path, nil
		}
	}

	return "", fmt.Errorf("uninstall script not found")
}

func (r *CommandFactoryRegistry) performManualCleanup() error {
	fmt.Println("🧹 Performing manual cleanup...")

	// Stop daemon processes
	fmt.Println("🛑 Stopping daemon processes...")
	if err := r.app.systemCommands.daemonCleanup([]string{"--yes", "--force"}); err != nil {
		fmt.Printf("⚠️  Daemon cleanup failed: %v\n", err)
	}

	// Remove configuration directory
	homeDir, err := os.UserHomeDir()
	if err == nil {
		configDir := filepath.Join(homeDir, ".cloudworkstation")
		if _, err := os.Stat(configDir); err == nil {
			fmt.Printf("🗂️  Removing configuration directory: %s\n", configDir)
			if err := os.RemoveAll(configDir); err != nil {
				fmt.Printf("⚠️  Failed to remove config directory: %v\n", err)
			} else {
				fmt.Println("✅ Configuration directory removed")
			}
		}
	}

	fmt.Println()
	fmt.Println("✅ Manual cleanup completed")
	fmt.Println("💡 You may need to manually remove:")
	fmt.Println("   • Binary files (cws, cwsd) from your PATH")
	fmt.Println("   • System service files")
	fmt.Println("   • Homebrew package: brew uninstall cloudworkstation")

	return nil
}

func (r *CommandFactoryRegistry) createConfigCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "config <action> [args]",
		Short: "Configure CloudWorkstation",
		Long:  `View and update CloudWorkstation configuration.`,
		RunE: func(_ *cobra.Command, args []string) error {
			return r.handleConfigCommand(args)
		},
	}
}

func (r *CommandFactoryRegistry) handleConfigCommand(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: cws config <action> [args]")
	}

	action := args[0]
	configArgs := args[1:]

	switch action {
	case "show":
		return r.app.configShow()
	case "profile", "set-aws-profile":
		if len(configArgs) != 1 {
			return fmt.Errorf("usage: cws config profile <aws-profile>")
		}
		return r.app.configSetProfile(configArgs[0])
	case "region":
		if len(configArgs) != 1 {
			return fmt.Errorf("usage: cws config region <aws-region>")
		}
		return r.app.configSetRegion(configArgs[0])
	default:
		return fmt.Errorf("unknown config action: %s", action)
	}
}

func (r *CommandFactoryRegistry) createIdleCommand() *cobra.Command {
	idleCmd := &cobra.Command{
		Use:   "idle",
		Short: "Configure idle detection on running workspaces",
		Long:  "Configure runtime idle detection parameters on running CloudWorkstation workspaces.",
	}

	idleConfigureCmd := &cobra.Command{
		Use:   "configure <workspace-name>",
		Short: "Configure idle thresholds on running workspace",
		Long:  "Configure runtime idle detection parameters on a running CloudWorkstation workspace.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return r.handleIdleConfigureCommand(cmd, args)
		},
	}

	r.addIdleFlags(idleConfigureCmd)
	idleCmd.AddCommand(idleConfigureCmd)
	return idleCmd
}

func (r *CommandFactoryRegistry) handleIdleConfigureCommand(cmd *cobra.Command, args []string) error {
	instanceName := args[0]
	enable, _ := cmd.Flags().GetBool("enable")
	disable, _ := cmd.Flags().GetBool("disable")
	idleMinutes, _ := cmd.Flags().GetInt("idle-minutes")
	hibernateMinutes, _ := cmd.Flags().GetInt("hibernate-minutes")
	checkInterval, _ := cmd.Flags().GetInt("check-interval")
	return r.app.configureIdleDetection(instanceName, enable, disable, idleMinutes, hibernateMinutes, checkInterval)
}

func (r *CommandFactoryRegistry) addIdleFlags(cmd *cobra.Command) {
	cmd.Flags().Bool("enable", false, "Enable idle detection")
	cmd.Flags().Bool("disable", false, "Disable idle detection")
	cmd.Flags().Int("idle-minutes", 0, "Minutes before considered idle")
	cmd.Flags().Int("hibernate-minutes", 0, "Minutes before hibernation/stop")
	cmd.Flags().Int("check-interval", 0, "Check interval in minutes")
}

func (r *CommandFactoryRegistry) createRightsizingCommand() *cobra.Command {
	return &cobra.Command{
		Use:     "rightsizing <action>",
		Short:   "Analyze and optimize workspace sizes",
		GroupID: "cost",
		Long:    `Analyze usage patterns and provide rightsizing recommendations for cost optimization.`,
		RunE: func(_ *cobra.Command, args []string) error {
			return r.app.Rightsizing(args)
		},
	}
}

func (r *CommandFactoryRegistry) createScalingCommand() *cobra.Command {
	return &cobra.Command{
		Use:     "scaling <action>",
		Short:   "Dynamic workspace scaling operations",
		GroupID: "cost",
		Long:    `Dynamically scale workspaces to different sizes based on usage patterns and requirements.`,
		RunE: func(_ *cobra.Command, args []string) error {
			return r.app.Scaling(args)
		},
	}
}

func (r *CommandFactoryRegistry) createAMIDiscoverCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "ami-discover",
		Short: "Demonstrate AMI auto-discovery functionality",
		Long:  `Show which templates have pre-built AMIs available for faster launching.`,
		RunE: func(_ *cobra.Command, args []string) error {
			return r.app.AMIDiscover(args)
		},
	}
}

// NewRootCommand creates the root command for the CLI
func (a *App) NewRootCommand() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "cws",
		Short: "CloudWorkstation - Launch research computing environments",
		Long: fmt.Sprintf(`%s

CloudWorkstation provides researchers with pre-configured cloud computing
environments for data analysis, machine learning, and research computing.

`, version.GetVersionInfo()),
		Version: version.GetCLIVersionInfo(),
	}

	// Add command groups for better organization
	rootCmd.AddGroup(&cobra.Group{ID: "core", Title: "Core Commands:"})
	rootCmd.AddGroup(&cobra.Group{ID: "instance", Title: "Workspace Management:"})
	rootCmd.AddGroup(&cobra.Group{ID: "storage", Title: "Storage & Data:"})
	rootCmd.AddGroup(&cobra.Group{ID: "cost", Title: "Cost Management:"})
	rootCmd.AddGroup(&cobra.Group{ID: "templates", Title: "Templates & Marketplace:"})
	rootCmd.AddGroup(&cobra.Group{ID: "collab", Title: "Collaboration:"})
	rootCmd.AddGroup(&cobra.Group{ID: "system", Title: "System & Configuration:"})

	// Register all commands using factory pattern
	factory := NewCommandFactoryRegistry(a)
	factory.RegisterAllCommands(rootCmd)

	return rootCmd
}

// Run executes the application with the given arguments
func (a *App) Run(args []string) error {
	rootCmd := a.NewRootCommand()
	rootCmd.SetArgs(args[1:]) // Skip the first argument (program name)
	return rootCmd.Execute()
}

// Config command implementations

func (a *App) configShow() error {
	fmt.Printf("📋 CloudWorkstation Configuration\n\n")

	// Show current effective configuration
	fmt.Printf("🔧 Current Configuration:\n")
	fmt.Printf("   Daemon URL: %s\n", a.config.Daemon.URL)
	fmt.Printf("   AWS Profile: %s\n", valueOrEmpty(a.config.AWS.Profile))
	fmt.Printf("   AWS Region: %s\n", valueOrEmpty(a.config.AWS.Region))

	// Show environment variable overrides
	fmt.Printf("\n🌍 Environment Variables:\n")
	if profile := os.Getenv("AWS_PROFILE"); profile != "" {
		fmt.Printf("   AWS_PROFILE: %s (overrides config file)\n", profile)
	} else {
		fmt.Printf("   AWS_PROFILE: (not set)\n")
	}
	if region := os.Getenv("AWS_REGION"); region != "" {
		fmt.Printf("   AWS_REGION: %s (overrides config file)\n", region)
	} else if region := os.Getenv("AWS_DEFAULT_REGION"); region != "" {
		fmt.Printf("   AWS_DEFAULT_REGION: %s (overrides config file)\n", region)
	} else {
		fmt.Printf("   AWS_REGION/AWS_DEFAULT_REGION: (not set)\n")
	}

	// Show config file location
	homeDir, _ := os.UserHomeDir()
	configFile := filepath.Join(homeDir, ".cloudworkstation", "config.json")
	fmt.Printf("\n📁 Config File: %s\n", configFile)
	if _, err := os.Stat(configFile); os.IsNotExist(err) {
		fmt.Printf("   (file does not exist - using defaults)\n")
	}

	fmt.Printf("\n💡 Usage:\n")
	fmt.Printf("   cws config profile <aws-profile>  # Set default AWS profile\n")
	fmt.Printf("   cws config region <aws-region>    # Set default AWS region\n")
	fmt.Printf("   export AWS_PROFILE=profile        # Override profile for session\n")

	return nil
}

func (a *App) configSetProfile(awsProfile string) error {
	a.config.AWS.Profile = awsProfile
	err := saveConfig(a.config)
	if err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}
	fmt.Printf("✅ AWS Profile set to '%s'\n", awsProfile)
	return nil
}

func (a *App) configSetRegion(region string) error {
	a.config.AWS.Region = region
	err := saveConfig(a.config)
	if err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}
	fmt.Printf("✅ AWS Region set to '%s'\n", region)
	return nil
}
