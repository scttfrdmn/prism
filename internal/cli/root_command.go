package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

// NewRootCommand creates the root command for the CLI
func (a *App) NewRootCommand() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "cws",
		Short: "CloudWorkstation - Launch research environments in seconds",
		Long: `CloudWorkstation helps researchers quickly launch cloud environments
for scientific computing without configuration hassles.

Default to Success: Every template works out of the box
Optimize by Default: Templates choose the best instance types
Smart Fallbacks:     Graceful degradation when necessary
Progressive Disclosure: Simple by default, detailed when needed`,
		Version: a.version,
	}

	// Launch command
	launchCmd := &cobra.Command{
		Use:   "launch <template> <name>",
		Short: "Launch a new cloud workstation",
		Long:  `Launch a new cloud workstation from a template with smart defaults.`,
		Args:  cobra.MinimumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			// Get flags and add them to args for compatibility with existing Launch function
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
			return a.Launch(args)
		},
	}
	launchCmd.Flags().Bool("hibernation", false, "Enable hibernation support")
	launchCmd.Flags().Bool("spot", false, "Use spot instances") 
	launchCmd.Flags().String("size", "", "Instance size: XS=1vCPU,2GB+100GB | S=2vCPU,4GB+500GB | M=2vCPU,8GB+1TB | L=4vCPU,16GB+2TB | XL=8vCPU,32GB+4TB")
	launchCmd.Flags().String("subnet", "", "Specify subnet ID")
	launchCmd.Flags().String("vpc", "", "Specify VPC ID")
	launchCmd.Flags().String("project", "", "Associate with project")
	rootCmd.AddCommand(launchCmd)

	// List command with subcommands
	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List workstations",
		Long:  `List all your cloud workstations and their status.`,
		RunE: func(_ *cobra.Command, args []string) error {
			return a.List(args)
		},
	}
	
	// List cost subcommand
	listCostCmd := &cobra.Command{
		Use:   "cost",
		Short: "Show detailed cost information",
		Long:  `Show detailed cost information for all workstations with four decimal precision.`,
		RunE: func(_ *cobra.Command, args []string) error {
			return a.ListCost(args)
		},
	}
	
	listCmd.AddCommand(listCostCmd)
	rootCmd.AddCommand(listCmd)

	// Connect command
	rootCmd.AddCommand(&cobra.Command{
		Use:   "connect <name>",
		Short: "Connect to a workstation",
		Long:  `Get connection information for a cloud workstation.`,
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			return a.Connect(args)
		},
	})

	// Stop command
	rootCmd.AddCommand(&cobra.Command{
		Use:   "stop <name>",
		Short: "Stop a workstation",
		Long:  `Stop a running cloud workstation to save costs.`,
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			return a.Stop(args)
		},
	})

	// Start command
	rootCmd.AddCommand(&cobra.Command{
		Use:   "start <name>",
		Short: "Start a workstation",
		Long:  `Start a stopped cloud workstation.`,
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			return a.Start(args)
		},
	})

	// Delete command
	rootCmd.AddCommand(&cobra.Command{
		Use:   "delete <name>",
		Short: "Delete a workstation",
		Long:  `Permanently delete a cloud workstation and its resources.`,
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			return a.Delete(args)
		},
	})

	// Hibernate command
	rootCmd.AddCommand(&cobra.Command{
		Use:   "hibernate <name>",
		Short: "Hibernate a workstation",
		Long: `Hibernate a running workstation to preserve RAM state while stopping compute billing.
If hibernation is not supported, automatically falls back to regular stop.`,
		Args: cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			return a.Hibernate(args)
		},
	})

	// Resume command
	rootCmd.AddCommand(&cobra.Command{
		Use:   "resume <name>",
		Short: "Resume a workstation",
		Long: `Resume a hibernated workstation with instant startup from preserved RAM state.
If not hibernated, performs regular start operation.`,
		Args: cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			return a.Resume(args)
		},
	})

	// Templates command
	rootCmd.AddCommand(&cobra.Command{
		Use:   "templates",
		Short: "List available templates",
		Long:  `List all available templates with their descriptions and costs.`,
		RunE: func(_ *cobra.Command, args []string) error {
			return a.Templates(args)
		},
	})

	// Apply command
	rootCmd.AddCommand(&cobra.Command{
		Use:   "apply <template> <instance-name>",
		Short: "Apply template to running instance",
		Long: `Apply a template to an already running instance, enabling incremental 
environment evolution without requiring instance recreation.

This allows you to add packages, services, and users to existing instances
while maintaining rollback capabilities.`,
		Args: cobra.MinimumNArgs(2),
		RunE: func(_ *cobra.Command, args []string) error {
			return a.Apply(args)
		},
	})

	// Diff command
	rootCmd.AddCommand(&cobra.Command{
		Use:   "diff <template> <instance-name>",
		Short: "Show template differences",
		Long: `Show what changes would be made when applying a template to a running instance.
This provides a preview of packages, services, users, and ports that would be modified.`,
		Args: cobra.ExactArgs(2),
		RunE: func(_ *cobra.Command, args []string) error {
			return a.Diff(args)
		},
	})

	// Layers command
	rootCmd.AddCommand(&cobra.Command{
		Use:   "layers <instance-name>",
		Short: "List applied template layers",
		Long: `List all templates that have been applied to an instance, showing the
layer history with rollback checkpoints.`,
		Args: cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			return a.Layers(args)
		},
	})

	// Rollback command
	rootCmd.AddCommand(&cobra.Command{
		Use:   "rollback <instance-name>",
		Short: "Rollback template applications",
		Long: `Rollback an instance to a previous state by undoing template applications.
Can rollback to the previous checkpoint or a specific checkpoint ID.`,
		Args: cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			return a.Rollback(args)
		},
	})

	// Volume command
	volumeCmd := &cobra.Command{
		Use:   "volume <action>",
		Short: "Manage EFS volumes",
		Long:  `Create and manage shared EFS volumes for your workstations.`,
		RunE: func(_ *cobra.Command, args []string) error {
			return a.Volume(args)
		},
	}
	rootCmd.AddCommand(volumeCmd)

	// Storage command
	storageCmd := &cobra.Command{
		Use:   "storage <action>",
		Short: "Manage EBS storage",
		Long:  `Create and manage EBS storage volumes for your workstations.`,
		RunE: func(_ *cobra.Command, args []string) error {
			return a.Storage(args)
		},
	}
	rootCmd.AddCommand(storageCmd)

	// Daemon command
	daemonCmd := &cobra.Command{
		Use:   "daemon <action>",
		Short: "Manage the daemon",
		Long:  `Control the CloudWorkstation daemon process.`,
		RunE: func(_ *cobra.Command, args []string) error {
			return a.Daemon(args)
		},
	}
	rootCmd.AddCommand(daemonCmd)

	// TUI command
	rootCmd.AddCommand(a.tuiCommand)

	// Config command
	configCmd := &cobra.Command{
		Use:   "config <action> [args]",
		Short: "Configure CloudWorkstation",
		Long:  `View and update CloudWorkstation configuration.`,
		RunE: func(_ *cobra.Command, args []string) error {
			if len(args) == 0 {
				return fmt.Errorf("usage: cws config <action> [args]")
			}

			action := args[0]
			configArgs := args[1:]

			switch action {
			case "show":
				return a.configShow()
			case "profile", "set-aws-profile":
				if len(configArgs) != 1 {
					return fmt.Errorf("usage: cws config profile <aws-profile>")
				}
				return a.configSetProfile(configArgs[0])
			case "region":
				if len(configArgs) != 1 {
					return fmt.Errorf("usage: cws config region <aws-region>")
				}
				return a.configSetRegion(configArgs[0])
			default:
				return fmt.Errorf("unknown config action: %s", action)
			}
		},
	}
	rootCmd.AddCommand(configCmd)

	// Add profile commands
	if a.profileManager != nil {
		AddProfileCommands(rootCmd, a.config)
		// Add migration command
		AddMigrateCommand(rootCmd, a.config)
	}

	// Add security command
	rootCmd.AddCommand(a.SecurityCommand())

	// Idle command
	idleCmd := &cobra.Command{
		Use:   "idle",
		Short: "Configure idle detection on running instances",
		Long:  "Configure runtime idle detection parameters on running CloudWorkstation instances.",
	}
	
	// Idle configure subcommand
	idleConfigureCmd := &cobra.Command{
		Use:   "configure <instance-name>",
		Short: "Configure idle thresholds on running instance",
		Long:  "Configure runtime idle detection parameters on a running CloudWorkstation instance.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			instanceName := args[0]
			
			// Get flag values
			enable, _ := cmd.Flags().GetBool("enable")
			disable, _ := cmd.Flags().GetBool("disable")
			idleMinutes, _ := cmd.Flags().GetInt("idle-minutes")
			hibernateMinutes, _ := cmd.Flags().GetInt("hibernate-minutes") 
			checkInterval, _ := cmd.Flags().GetInt("check-interval")
			
			return a.configureIdleDetection(instanceName, enable, disable, idleMinutes, hibernateMinutes, checkInterval)
		},
	}
	
	// Add flags to the configure command
	idleConfigureCmd.Flags().Bool("enable", false, "Enable idle detection")
	idleConfigureCmd.Flags().Bool("disable", false, "Disable idle detection")
	idleConfigureCmd.Flags().Int("idle-minutes", 0, "Minutes before considered idle")
	idleConfigureCmd.Flags().Int("hibernate-minutes", 0, "Minutes before hibernation/stop") 
	idleConfigureCmd.Flags().Int("check-interval", 0, "Check interval in minutes")
	
	idleCmd.AddCommand(idleConfigureCmd)
	rootCmd.AddCommand(idleCmd)

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