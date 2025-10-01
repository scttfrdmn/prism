// Package cli implements admin commands for CloudWorkstation CLI.
//
// This module provides comprehensive system administration functionality including
// configuration management, security settings, policy management, and daemon control.
//
// Commands:
//   - admin config                   # System configuration management
//   - admin security                 # Security settings and management
//   - admin policy                   # Policy management and enforcement
//   - admin profiles                 # Profile management
//   - admin daemon                   # Daemon lifecycle management
//   - admin uninstall               # Complete system uninstallation
//
// Examples:
//
//	cws admin config --check
//	cws admin security scan
//	cws admin policy enable
//	cws admin daemon status
package cli

import (
	"github.com/spf13/cobra"
)

// AdminCommands provides system administration functionality
type AdminCommands struct {
	app *App
}

// NewAdminCommands creates a new admin commands handler
func NewAdminCommands(app *App) *AdminCommands {
	return &AdminCommands{
		app: app,
	}
}

// AdminCommandFactory creates admin commands using factory pattern
type AdminCommandFactory struct {
	app *App
}

// CreateCommands creates all admin commands
func (f *AdminCommandFactory) CreateCommands() []*cobra.Command {
	commands := NewAdminCommands(f.app)
	return []*cobra.Command{
		commands.createMainCommand(),
	}
}

// createMainCommand creates the main "admin" command with subcommands
func (r *AdminCommands) createMainCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "admin",
		Short: "System administration commands",
		Long: `System administration commands for CloudWorkstation configuration and management.

Provides centralized access to system configuration, security management, policy
enforcement, profile administration, and daemon lifecycle operations.

Examples:
  cws admin config --check          # Check system configuration
  cws admin security scan           # Run security scan
  cws admin policy enable           # Enable policy enforcement
  cws admin daemon status           # Check daemon status`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}

	// Add admin subcommands (these will call the existing implementations)
	cmd.AddCommand(r.createConfigCommand())
	cmd.AddCommand(r.createSecurityCommand())
	cmd.AddCommand(r.createPolicyCommand())
	cmd.AddCommand(r.createProfilesCommand())
	cmd.AddCommand(r.createDaemonCommand())
	cmd.AddCommand(r.createUninstallCommand())

	return cmd
}

// Admin subcommands (these delegate to existing root commands for compatibility)

func (r *AdminCommands) createConfigCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "config <action> [args]",
		Short: "Configure CloudWorkstation",
		Long:  `View and update CloudWorkstation configuration.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Use the existing config implementation
			factory := &CommandFactoryRegistry{app: r.app}
			return factory.handleConfigCommand(args)
		},
	}
}

func (r *AdminCommands) createSecurityCommand() *cobra.Command {
	// Use the existing security command implementation
	return r.app.SecurityCommand()
}

func (r *AdminCommands) createPolicyCommand() *cobra.Command {
	// Create the policy command using the existing factory
	policyFactory := &PolicyCommandFactory{app: r.app}
	policyCommands := policyFactory.CreateCommands()
	if len(policyCommands) > 0 {
		return policyCommands[0] // Return the main policy command
	}

	// Fallback
	return &cobra.Command{
		Use:   "policy",
		Short: "Manage policy framework for template and resource access control",
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}
}

func (r *AdminCommands) createProfilesCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "profiles",
		Short: "Manage CloudWorkstation profiles",
		Long:  `Manage CloudWorkstation profiles for different AWS accounts and configurations.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}

	// Add profile subcommands (integration with existing profile system)
	if r.app.profileManager != nil {
		// This will add all the profile subcommands under admin profiles
		AddProfileCommands(cmd, r.app.config)
	}

	return cmd
}

func (r *AdminCommands) createDaemonCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "daemon <action>",
		Short: "Manage the daemon",
		Long:  `Control the CloudWorkstation daemon process.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return r.app.Daemon(args)
		},
	}
}

func (r *AdminCommands) createUninstallCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "uninstall",
		Short: "Uninstall CloudWorkstation completely",
		RunE: func(cmd *cobra.Command, args []string) error {
			// Use the existing uninstall implementation
			factory := &CommandFactoryRegistry{app: r.app}
			return factory.handleUninstallCommand(cmd, args)
		},
	}
}
