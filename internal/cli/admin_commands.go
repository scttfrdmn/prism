package cli

import (
	"github.com/spf13/cobra"
)

// AdminCommandFactory creates the unified admin command group
type AdminCommandFactory struct {
	app *App
}

// CreateCommand creates the admin command group
func (f *AdminCommandFactory) CreateCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "admin",
		Short: "Administrative and system management",
		Long: `Administrative commands for system management, daemon control, and advanced configuration.

Examples:
  prism admin daemon status          # Check daemon status
  prism admin daemon start           # Start daemon
  prism admin policy list            # List policies
  prism admin rightsizing analyze    # Analyze instance sizing`,
		GroupID: "system",
	}

	// Add subcommands
	cmd.AddCommand(f.createDaemonCommand())
	cmd.AddCommand(f.createPolicyCommand())
	cmd.AddCommand(f.createRightsizingCommand())
	cmd.AddCommand(f.createScalingCommand())

	return cmd
}

// createDaemonCommand creates the daemon subcommand
func (f *AdminCommandFactory) createDaemonCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "daemon <action>",
		Short: "Manage Prism daemon",
		Long:  `Control the Prism daemon service (start, stop, status, logs).`,
		RunE: func(_ *cobra.Command, args []string) error {
			return f.app.Daemon(args)
		},
	}
}

// createPolicyCommand creates the policy subcommand
func (f *AdminCommandFactory) createPolicyCommand() *cobra.Command {
	policyCobra := NewPolicyCobraCommands(f.app)
	return policyCobra.CreatePolicyCommand()
}

// createRightsizingCommand creates the rightsizing subcommand
func (f *AdminCommandFactory) createRightsizingCommand() *cobra.Command {
	rightsizingCobra := NewRightsizingCobraCommands(f.app)
	return rightsizingCobra.CreateRightsizingCommand()
}

// createScalingCommand creates the scaling subcommand
func (f *AdminCommandFactory) createScalingCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "scaling <action>",
		Short: "Dynamic workspace scaling operations",
		Long:  `Dynamically scale workspaces to different sizes based on usage patterns and requirements.`,
		RunE: func(_ *cobra.Command, args []string) error {
			return f.app.Scaling(args)
		},
	}
}
