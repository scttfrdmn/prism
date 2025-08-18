package cli

import (
	"fmt"
	"os"
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
		Use:   "launch <template> <name>",
		Short: "Launch a new cloud workstation",
		Long:  `Launch a new cloud workstation from a template with smart defaults.`,
		Args:  cobra.MinimumNArgs(2),
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
	return f.app.Launch(args)
}

func (f *LaunchCommandFactory) addLaunchFlags(cmd *cobra.Command) {
	cmd.Flags().Bool("hibernation", false, "Enable hibernation support")
	cmd.Flags().Bool("spot", false, "Use spot instances")
	cmd.Flags().String("size", "", "Instance size: XS=1vCPU,2GB+100GB | S=2vCPU,4GB+500GB | M=2vCPU,8GB+1TB | L=4vCPU,16GB+2TB | XL=8vCPU,32GB+4TB")
	cmd.Flags().String("subnet", "", "Specify subnet ID")
	cmd.Flags().String("vpc", "", "Specify VPC ID")
	cmd.Flags().String("project", "", "Associate with project")
	cmd.Flags().Bool("wait", false, "Wait and display launch progress in real-time")
	cmd.Flags().Bool("dry-run", false, "Validate configuration without launching")
}

// InstanceCommandFactory creates instance management commands
type InstanceCommandFactory struct {
	app *App
}

func (f *InstanceCommandFactory) CreateCommands() []*cobra.Command {
	return []*cobra.Command{
		f.createConnectCommand(),
		f.createStopCommand(),
		f.createStartCommand(),
		f.createDeleteCommand(),
		f.createHibernateCommand(),
		f.createResumeCommand(),
	}
}

func (f *InstanceCommandFactory) createConnectCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "connect <name>",
		Short: "Connect to a workstation",
		Long:  `Get connection information for a cloud workstation.`,
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			return f.app.Connect(args)
		},
	}
}

func (f *InstanceCommandFactory) createStopCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "stop <name>",
		Short: "Stop a workstation",
		Long:  `Stop a running cloud workstation to save costs.`,
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			return f.app.Stop(args)
		},
	}
}

func (f *InstanceCommandFactory) createStartCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "start <name>",
		Short: "Start a workstation",
		Long:  `Start a stopped cloud workstation.`,
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			return f.app.Start(args)
		},
	}
}

func (f *InstanceCommandFactory) createDeleteCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "delete <name>",
		Short: "Delete a workstation",
		Long:  `Permanently delete a cloud workstation and its resources.`,
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			return f.app.Delete(args)
		},
	}
}

func (f *InstanceCommandFactory) createHibernateCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "hibernate <name>",
		Short: "Hibernate a workstation",
		Long: `Hibernate a running workstation to preserve RAM state while stopping compute billing.
If hibernation is not supported, automatically falls back to regular stop.`,
		Args: cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			return f.app.Hibernate(args)
		},
	}
}

func (f *InstanceCommandFactory) createResumeCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "resume <name>",
		Short: "Resume a workstation",
		Long: `Resume a hibernated workstation with instant startup from preserved RAM state.
If not hibernated, performs regular start operation.`,
		Args: cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			return f.app.Resume(args)
		},
	}
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
	return &cobra.Command{
		Use:   "templates",
		Short: "List available templates",
		Long:  `List all available templates with their descriptions and costs.`,
		RunE: func(_ *cobra.Command, args []string) error {
			return f.app.Templates(args)
		},
	}
}

func (f *TemplateCommandFactory) createApplyCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "apply <template> <instance-name>",
		Short: "Apply template to running instance",
		Long: `Apply a template to an already running instance, enabling incremental 
environment evolution without requiring instance recreation.

This allows you to add packages, services, and users to existing instances
while maintaining rollback capabilities.`,
		Args: cobra.MinimumNArgs(2),
		RunE: func(_ *cobra.Command, args []string) error {
			return f.app.Apply(args)
		},
	}
}

func (f *TemplateCommandFactory) createDiffCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "diff <template> <instance-name>",
		Short: "Show template differences",
		Long: `Show what changes would be made when applying a template to a running instance.
This provides a preview of packages, services, users, and ports that would be modified.`,
		Args: cobra.ExactArgs(2),
		RunE: func(_ *cobra.Command, args []string) error {
			return f.app.Diff(args)
		},
	}
}

func (f *TemplateCommandFactory) createLayersCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "layers <instance-name>",
		Short: "List applied template layers",
		Long: `List all templates that have been applied to an instance, showing the
layer history with rollback checkpoints.`,
		Args: cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			return f.app.Layers(args)
		},
	}
}

func (f *TemplateCommandFactory) createRollbackCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "rollback <instance-name>",
		Short: "Rollback template applications",
		Long: `Rollback an instance to a previous state by undoing template applications.
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
	// Launch command
	launchFactory := &LaunchCommandFactory{app: r.app}
	rootCmd.AddCommand(launchFactory.CreateCommand())

	// List command
	rootCmd.AddCommand(r.createListCommand())

	// Instance commands
	instanceFactory := &InstanceCommandFactory{app: r.app}
	for _, cmd := range instanceFactory.CreateCommands() {
		rootCmd.AddCommand(cmd)
	}

	// Template commands
	templateFactory := &TemplateCommandFactory{app: r.app}
	for _, cmd := range templateFactory.CreateCommands() {
		rootCmd.AddCommand(cmd)
	}

	// Storage commands
	rootCmd.AddCommand(r.createVolumeCommand())
	rootCmd.AddCommand(r.createStorageCommand())

	// System commands
	rootCmd.AddCommand(r.createDaemonCommand())
	rootCmd.AddCommand(r.app.tuiCommand)
	rootCmd.AddCommand(r.createConfigCommand())

	// Profile commands
	if r.app.profileManager != nil {
		AddProfileCommands(rootCmd, r.app.config)
		AddMigrateCommand(rootCmd, r.app.config)
	}

	// Security and idle commands
	rootCmd.AddCommand(r.app.SecurityCommand())
	rootCmd.AddCommand(r.createIdleCommand())

	// Advanced commands
	rootCmd.AddCommand(r.createRightsizingCommand())
	rootCmd.AddCommand(r.createScalingCommand())
	rootCmd.AddCommand(r.createAMICommand())
	rootCmd.AddCommand(r.createAMIDiscoverCommand())
}

func (r *CommandFactoryRegistry) createListCommand() *cobra.Command {
	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List workstations",
		Long:  `List all your cloud workstations and their status.`,
		RunE: func(_ *cobra.Command, args []string) error {
			return r.app.List(args)
		},
	}
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

func (r *CommandFactoryRegistry) createVolumeCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "volume <action>",
		Short: "Manage EFS volumes",
		Long:  `Create and manage shared EFS volumes for your workstations.`,
		RunE: func(_ *cobra.Command, args []string) error {
			return r.app.Volume(args)
		},
	}
}

func (r *CommandFactoryRegistry) createStorageCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "storage <action>",
		Short: "Manage EBS storage",
		Long:  `Create and manage EBS storage volumes for your workstations.`,
		RunE: func(_ *cobra.Command, args []string) error {
			return r.app.Storage(args)
		},
	}
}

func (r *CommandFactoryRegistry) createDaemonCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "daemon <action>",
		Short: "Manage the daemon",
		Long:  `Control the CloudWorkstation daemon process.`,
		RunE: func(_ *cobra.Command, args []string) error {
			return r.app.Daemon(args)
		},
	}
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
		Short: "Configure idle detection on running instances",
		Long:  "Configure runtime idle detection parameters on running CloudWorkstation instances.",
	}

	idleConfigureCmd := &cobra.Command{
		Use:   "configure <instance-name>",
		Short: "Configure idle thresholds on running instance",
		Long:  "Configure runtime idle detection parameters on a running CloudWorkstation instance.",
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
		Use:   "rightsizing <action>",
		Short: "Analyze and optimize instance sizes",
		Long:  `Analyze usage patterns and provide rightsizing recommendations for cost optimization.`,
		RunE: func(_ *cobra.Command, args []string) error {
			return r.app.Rightsizing(args)
		},
	}
}

func (r *CommandFactoryRegistry) createScalingCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "scaling <action>",
		Short: "Dynamic instance scaling operations",
		Long:  `Dynamically scale instances to different sizes based on usage patterns and requirements.`,
		RunE: func(_ *cobra.Command, args []string) error {
			return r.app.Scaling(args)
		},
	}
}

func (r *CommandFactoryRegistry) createAMICommand() *cobra.Command {
	return &cobra.Command{
		Use:                "ami <action>",
		Short:              "AMI management operations",
		Long:               `Build, manage, and deploy AMIs for fast instance launching.`,
		DisableFlagParsing: true, // Allow AMI command to handle its own flags
		RunE: func(_ *cobra.Command, args []string) error {
			return r.app.AMI(args)
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
		Short: "CloudWorkstation - Launch research environments in seconds",
		Long: fmt.Sprintf(`%s

CloudWorkstation helps researchers quickly launch cloud environments
for research computing without configuration hassles.

`, version.GetVersionInfo()),
		Version: a.version,
	}

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
