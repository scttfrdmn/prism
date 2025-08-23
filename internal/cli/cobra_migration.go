// Package cli provides the CloudWorkstation command-line interface.
// This file contains the migration utilities for converting internal command routing to proper Cobra subcommands.
package cli

import (
	"github.com/spf13/cobra"
)

// CobraCommandRegistry manages all migrated Cobra commands
type CobraCommandRegistry struct {
	app *App
}

// NewCobraCommandRegistry creates a new registry for Cobra commands
func NewCobraCommandRegistry(app *App) *CobraCommandRegistry {
	return &CobraCommandRegistry{app: app}
}

// RegisterAllCommands registers all commands using proper Cobra structure
func (r *CobraCommandRegistry) RegisterAllCommands(root *cobra.Command) {
	// Template commands (COMPLETED)
	templateCobra := NewTemplateCobraCommands(r.app)
	root.AddCommand(templateCobra.CreateTemplatesCommand())
	
	// Daemon commands (EXAMPLE PROVIDED)
	daemonCobra := NewDaemonCobraCommands(r.app)
	root.AddCommand(daemonCobra.CreateDaemonCommand())
	
	// Idle/Hibernation commands (TODO)
	idleCobra := NewIdleCobraCommands(r.app)
	root.AddCommand(idleCobra.CreateIdleCommand())
	
	// Project commands (TODO)
	projectCobra := NewProjectCobraCommands(r.app)
	root.AddCommand(projectCobra.CreateProjectCommand())
	
	// Storage commands (TODO)
	storageCobra := NewStorageCobraCommands(r.app)
	root.AddCommand(storageCobra.CreateStorageCommand())
	root.AddCommand(storageCobra.CreateVolumeCommand())
	
	// Repository commands (TODO)
	repoCobra := NewRepoCobraCommands(r.app)
	root.AddCommand(repoCobra.CreateRepoCommand())
}

// IdleCobraCommands is defined in idle_cobra.go

// ProjectCobraCommands handles project management commands
type ProjectCobraCommands struct {
	app *App
}

// NewProjectCobraCommands creates new project cobra commands
func NewProjectCobraCommands(app *App) *ProjectCobraCommands {
	return &ProjectCobraCommands{app: app}
}

// CreateProjectCommand creates the project command with subcommands
func (pc *ProjectCobraCommands) CreateProjectCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "project",
		Short: "Manage CloudWorkstation projects",
		Long: `Manage projects for organizing instances, tracking budgets, and collaborating
with team members.`,
	}
	
	// Add subcommands
	cmd.AddCommand(
		pc.createListCommand(),
		pc.createCreateCommand(),
		pc.createUpdateCommand(),
		pc.createDeleteCommand(),
		pc.createMembersCommand(),
		pc.createBudgetCommand(),
	)
	
	return cmd
}

// createListCommand creates the list subcommand
func (pc *ProjectCobraCommands) createListCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List all projects",
		RunE: func(cmd *cobra.Command, args []string) error {
			// TODO: Implement
			return nil
		},
	}
	
	cmd.Flags().Bool("all", false, "Show all projects including archived")
	cmd.Flags().String("filter", "", "Filter projects by name or tag")
	
	return cmd
}

// createCreateCommand creates the create subcommand
func (pc *ProjectCobraCommands) createCreateCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create <name>",
		Short: "Create a new project",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			description, _ := cmd.Flags().GetString("description")
			budget, _ := cmd.Flags().GetFloat64("budget")
			// TODO: Call actual implementation
			_ = description
			_ = budget
			return nil
		},
	}
	
	cmd.Flags().String("description", "", "Project description")
	cmd.Flags().Float64("budget", 0, "Monthly budget limit")
	cmd.Flags().StringSlice("tags", []string{}, "Project tags")
	
	return cmd
}

// createUpdateCommand creates the update subcommand
func (pc *ProjectCobraCommands) createUpdateCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update <name>",
		Short: "Update project settings",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			// TODO: Implement
			return nil
		},
	}
	
	cmd.Flags().String("description", "", "New description")
	cmd.Flags().Float64("budget", 0, "New budget limit")
	cmd.Flags().Bool("archive", false, "Archive the project")
	
	return cmd
}

// createDeleteCommand creates the delete subcommand
func (pc *ProjectCobraCommands) createDeleteCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delete <name>",
		Short: "Delete a project",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			force, _ := cmd.Flags().GetBool("force")
			// TODO: Call actual implementation
			_ = force
			return nil
		},
	}
	
	cmd.Flags().Bool("force", false, "Force delete without confirmation")
	
	return cmd
}

// createMembersCommand creates the members management subcommand
func (pc *ProjectCobraCommands) createMembersCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "members",
		Short: "Manage project members",
	}
	
	// Members subcommands
	cmd.AddCommand(
		&cobra.Command{
			Use:   "add <project> <email>",
			Short: "Add a member to project",
			Args:  cobra.ExactArgs(2),
			RunE: func(cmd *cobra.Command, args []string) error {
				role, _ := cmd.Flags().GetString("role")
				// TODO: Call actual implementation
				_ = role
				return nil
			},
		},
		&cobra.Command{
			Use:   "remove <project> <email>",
			Short: "Remove a member from project",
			Args:  cobra.ExactArgs(2),
			RunE: func(cmd *cobra.Command, args []string) error {
				// TODO: Implement
				return nil
			},
		},
		&cobra.Command{
			Use:   "list <project>",
			Short: "List project members",
			Args:  cobra.ExactArgs(1),
			RunE: func(cmd *cobra.Command, args []string) error {
				// TODO: Implement
				return nil
			},
		},
	)
	
	// Add flags to add command
	addCmd := cmd.Commands()[0]
	addCmd.Flags().String("role", "member", "Member role (owner/admin/member/viewer)")
	
	return cmd
}

// createBudgetCommand creates the budget management subcommand
func (pc *ProjectCobraCommands) createBudgetCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "budget",
		Short: "Manage project budgets",
	}
	
	// Budget subcommands
	cmd.AddCommand(
		&cobra.Command{
			Use:   "status <project>",
			Short: "Show budget status",
			Args:  cobra.ExactArgs(1),
			RunE: func(cmd *cobra.Command, args []string) error {
				// TODO: Implement
				return nil
			},
		},
		&cobra.Command{
			Use:   "history <project>",
			Short: "Show budget history",
			Args:  cobra.ExactArgs(1),
			RunE: func(cmd *cobra.Command, args []string) error {
				days, _ := cmd.Flags().GetInt("days")
				// TODO: Call actual implementation
				_ = days
				return nil
			},
		},
		&cobra.Command{
			Use:   "alert <project>",
			Short: "Configure budget alerts",
			Args:  cobra.ExactArgs(1),
			RunE: func(cmd *cobra.Command, args []string) error {
				threshold, _ := cmd.Flags().GetInt("threshold")
				email, _ := cmd.Flags().GetString("email")
				// TODO: Call actual implementation
				_ = threshold
				_ = email
				return nil
			},
		},
	)
	
	// Add flags
	historyCmd := cmd.Commands()[1]
	historyCmd.Flags().Int("days", 30, "Number of days of history")
	
	alertCmd := cmd.Commands()[2]
	alertCmd.Flags().Int("threshold", 80, "Alert threshold percentage")
	alertCmd.Flags().String("email", "", "Email for alerts")
	
	return cmd
}

// StorageCobraCommands handles storage/volume commands
type StorageCobraCommands struct {
	app *App
}

// NewStorageCobraCommands creates new storage cobra commands
func NewStorageCobraCommands(app *App) *StorageCobraCommands {
	return &StorageCobraCommands{app: app}
}

// CreateStorageCommand creates the storage command with subcommands
func (sc *StorageCobraCommands) CreateStorageCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "storage",
		Short: "Manage CloudWorkstation storage (EBS volumes)",
		Long:  "Create, attach, detach, and manage EBS storage volumes.",
	}
	
	// Add subcommands
	cmd.AddCommand(
		&cobra.Command{
			Use:   "create <name>",
			Short: "Create a new EBS volume",
			Args:  cobra.ExactArgs(1),
			RunE: func(cmd *cobra.Command, args []string) error {
				size, _ := cmd.Flags().GetString("size")
				volumeType, _ := cmd.Flags().GetString("type")
				// TODO: Call actual implementation
				_ = size
				_ = volumeType
				return nil
			},
		},
		&cobra.Command{
			Use:   "attach <volume> <instance>",
			Short: "Attach volume to instance",
			Args:  cobra.ExactArgs(2),
			RunE: func(cmd *cobra.Command, args []string) error {
				// TODO: Implement
				return nil
			},
		},
		&cobra.Command{
			Use:   "detach <volume>",
			Short: "Detach volume from instance",
			Args:  cobra.ExactArgs(1),
			RunE: func(cmd *cobra.Command, args []string) error {
				// TODO: Implement
				return nil
			},
		},
		&cobra.Command{
			Use:   "list",
			Short: "List all storage volumes",
			RunE: func(cmd *cobra.Command, args []string) error {
				// TODO: Implement
				return nil
			},
		},
		&cobra.Command{
			Use:   "delete <volume>",
			Short: "Delete a storage volume",
			Args:  cobra.ExactArgs(1),
			RunE: func(cmd *cobra.Command, args []string) error {
				force, _ := cmd.Flags().GetBool("force")
				// TODO: Call actual implementation
				_ = force
				return nil
			},
		},
	)
	
	// Add flags
	createCmd := cmd.Commands()[0]
	createCmd.Flags().String("size", "L", "Volume size (S/M/L/XL or custom like 100GB)")
	createCmd.Flags().String("type", "gp3", "Volume type (gp3/io2/st1)")
	
	deleteCmd := cmd.Commands()[4]
	deleteCmd.Flags().Bool("force", false, "Force delete without confirmation")
	
	return cmd
}

// CreateVolumeCommand creates the volume command (alias for storage)
func (sc *StorageCobraCommands) CreateVolumeCommand() *cobra.Command {
	// Volume is an alias for storage, reuse the same command structure
	cmd := sc.CreateStorageCommand()
	cmd.Use = "volume"
	cmd.Aliases = []string{"vol"}
	cmd.Short = "Manage CloudWorkstation volumes (EFS shared storage)"
	cmd.Long = "Create, attach, detach, and manage EFS shared storage volumes."
	
	return cmd
}

// RepoCobraCommands handles repository commands
type RepoCobraCommands struct {
	app *App
}

// NewRepoCobraCommands creates new repository cobra commands
func NewRepoCobraCommands(app *App) *RepoCobraCommands {
	return &RepoCobraCommands{app: app}
}

// CreateRepoCommand creates the repo command with subcommands
func (rc *RepoCobraCommands) CreateRepoCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "repo",
		Short: "Manage template repositories",
		Long:  "Add, remove, and manage CloudWorkstation template repositories.",
	}
	
	// Add subcommands
	cmd.AddCommand(
		&cobra.Command{
			Use:   "list",
			Short: "List all repositories",
			RunE: func(cmd *cobra.Command, args []string) error {
				// TODO: Implement
				return nil
			},
		},
		&cobra.Command{
			Use:   "add <name> <url>",
			Short: "Add a new repository",
			Args:  cobra.ExactArgs(2),
			RunE: func(cmd *cobra.Command, args []string) error {
				// TODO: Implement
				return nil
			},
		},
		&cobra.Command{
			Use:   "remove <name>",
			Short: "Remove a repository",
			Args:  cobra.ExactArgs(1),
			RunE: func(cmd *cobra.Command, args []string) error {
				// TODO: Implement
				return nil
			},
		},
		&cobra.Command{
			Use:   "sync [name]",
			Short: "Sync repository templates",
			RunE: func(cmd *cobra.Command, args []string) error {
				force, _ := cmd.Flags().GetBool("force")
				// TODO: Call actual implementation
				_ = force
				return nil
			},
		},
	)
	
	// Add flags
	syncCmd := cmd.Commands()[3]
	syncCmd.Flags().Bool("force", false, "Force sync even if up to date")
	
	return cmd
}