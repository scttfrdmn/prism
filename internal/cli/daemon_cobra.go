package cli

import (
	"fmt"
	
	"github.com/spf13/cobra"
)

// DaemonCobraCommands creates the daemon command with proper Cobra subcommands
type DaemonCobraCommands struct {
	app            *App
	systemCommands *SystemCommands
}

// NewDaemonCobraCommands creates new daemon cobra commands
func NewDaemonCobraCommands(app *App) *DaemonCobraCommands {
	return &DaemonCobraCommands{
		app:            app,
		systemCommands: NewSystemCommands(app),
	}
}

// CreateDaemonCommand creates the main daemon command with all subcommands
func (dc *DaemonCobraCommands) CreateDaemonCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "daemon",
		Short: "Manage CloudWorkstation daemon",
		Long: `Manage the CloudWorkstation background daemon service.
The daemon provides the API backend for all CloudWorkstation operations.`,
	}

	// Add subcommands
	cmd.AddCommand(
		dc.createStartCommand(),
		dc.createStopCommand(),
		dc.createStatusCommand(),
		dc.createRestartCommand(),
		dc.createLogsCommand(),
		dc.createConfigCommand(),
	)

	return cmd
}

// createStartCommand creates the start subcommand
func (dc *DaemonCobraCommands) createStartCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "start",
		Short: "Start the CloudWorkstation daemon",
		Long:  "Start the CloudWorkstation daemon service in the background.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return dc.systemCommands.daemonStart()
		},
	}

	// Add start-specific flags if needed
	cmd.Flags().Bool("foreground", false, "Run daemon in foreground")
	cmd.Flags().String("port", "8947", "Port to run daemon on")
	cmd.Flags().Bool("debug", false, "Enable debug logging")

	return cmd
}

// createStopCommand creates the stop subcommand
func (dc *DaemonCobraCommands) createStopCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "stop",
		Short: "Stop the CloudWorkstation daemon",
		Long:  "Stop the running CloudWorkstation daemon service.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return dc.systemCommands.daemonStop()
		},
	}
}

// createStatusCommand creates the status subcommand
func (dc *DaemonCobraCommands) createStatusCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "status",
		Short: "Check daemon status",
		Long:  "Check if the CloudWorkstation daemon is running and responsive.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return dc.systemCommands.daemonStatus()
		},
	}

	// Add status flags
	cmd.Flags().Bool("json", false, "Output status in JSON format")
	cmd.Flags().Bool("verbose", false, "Show detailed status information")

	return cmd
}

// createRestartCommand creates the restart subcommand
func (dc *DaemonCobraCommands) createRestartCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "restart",
		Short: "Restart the CloudWorkstation daemon",
		Long:  "Stop and then start the CloudWorkstation daemon service.",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := dc.systemCommands.daemonStop(); err != nil {
				// Continue even if stop fails (daemon might not be running)
			}
			return dc.systemCommands.daemonStart()
		},
	}
}

// createLogsCommand creates the logs subcommand
func (dc *DaemonCobraCommands) createLogsCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "logs",
		Short: "View daemon logs",
		Long:  "Display logs from the CloudWorkstation daemon.",
		RunE: func(cmd *cobra.Command, args []string) error {
			// Get flags
			follow, _ := cmd.Flags().GetBool("follow")
			tail, _ := cmd.Flags().GetInt("tail")
			
			// Build args for the existing logs function
			var logsArgs []string
			if follow {
				logsArgs = append(logsArgs, "--follow")
			}
			if tail > 0 {
				logsArgs = append(logsArgs, "--tail", string(tail))
			}
			
			return dc.systemCommands.daemonLogs()
		},
	}

	// Add logs flags
	cmd.Flags().BoolP("follow", "f", false, "Follow log output")
	cmd.Flags().IntP("tail", "n", 0, "Number of lines to show from the end of the logs")
	cmd.Flags().Bool("timestamps", false, "Show timestamps")

	return cmd
}

// createConfigCommand creates the config subcommand group
func (dc *DaemonCobraCommands) createConfigCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Manage daemon configuration",
		Long:  "View and manage CloudWorkstation daemon configuration settings.",
	}

	// Add config subcommands
	cmd.AddCommand(
		&cobra.Command{
			Use:   "show",
			Short: "Show current daemon configuration",
			RunE: func(cmd *cobra.Command, args []string) error {
				return dc.systemCommands.daemonConfigShow()
			},
		},
		&cobra.Command{
			Use:   "set <key> <value>",
			Short: "Set a daemon configuration value",
			Args:  cobra.ExactArgs(2),
			RunE: func(cmd *cobra.Command, args []string) error {
				return dc.systemCommands.daemonConfigSet(args)
			},
		},
		&cobra.Command{
			Use:   "get <key>",
			Short: "Get a daemon configuration value",
			Args:  cobra.ExactArgs(1),
			RunE: func(cmd *cobra.Command, args []string) error {
				// Daemon config get implementation
				key := args[0]
				switch key {
				case "url":
					fmt.Printf("%s\n", dc.app.config.Daemon.URL)
				default:
					return fmt.Errorf("unknown configuration key: %s", key)
				}
				return nil
			},
		},
		&cobra.Command{
			Use:   "reset",
			Short: "Reset daemon configuration to defaults",
			RunE: func(cmd *cobra.Command, args []string) error {
				// Reset daemon configuration to defaults
				fmt.Println("Resetting daemon configuration to defaults...")
				dc.app.config.Daemon.URL = "http://localhost:8947"
				fmt.Println("✅ Daemon configuration reset to defaults")
				return nil
			},
		},
	)

	return cmd
}