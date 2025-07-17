package cli

import (
	"fmt"

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
	rootCmd.AddCommand(&cobra.Command{
		Use:   "launch <template> <name>",
		Short: "Launch a new cloud workstation",
		Long:  `Launch a new cloud workstation from a template with smart defaults.`,
		Args:  cobra.MinimumNArgs(2),
		RunE: func(_ *cobra.Command, args []string) error {
			return a.Launch(args)
		},
	})

	// List command
	rootCmd.AddCommand(&cobra.Command{
		Use:   "list",
		Short: "List workstations",
		Long:  `List all your cloud workstations and their status.`,
		RunE: func(_ *cobra.Command, args []string) error {
			return a.List(args)
		},
	})

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

	// Templates command
	rootCmd.AddCommand(&cobra.Command{
		Use:   "templates",
		Short: "List available templates",
		Long:  `List all available templates with their descriptions and costs.`,
		RunE: func(_ *cobra.Command, args []string) error {
			return a.Templates(args)
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
			case "profile":
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
	fmt.Println("CloudWorkstation Configuration:")
	fmt.Printf("   Daemon URL: %s\n", a.config.Daemon.URL)
	fmt.Printf("   AWS Profile: %s\n", valueOrEmpty(a.config.AWS.Profile))
	fmt.Printf("   AWS Region: %s\n", valueOrEmpty(a.config.AWS.Region))
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