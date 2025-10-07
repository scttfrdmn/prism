package cli

import (
	"fmt"
	"github.com/spf13/cobra"
)

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
		pc.createInfoCommand(),
		pc.createDeleteCommand(),
		pc.createMembersCommand(),
		pc.createBudgetCommand(),
		pc.createInstancesCommand(),
		pc.createTemplatesCommand(),
	)

	return cmd
}

// createListCommand creates the list subcommand
func (pc *ProjectCobraCommands) createListCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all projects",
		RunE: func(cmd *cobra.Command, args []string) error {
			return pc.app.Project([]string{"list"})
		},
	}
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
			owner, _ := cmd.Flags().GetString("owner")

			createArgs := []string{"create", args[0]}
			if description != "" {
				createArgs = append(createArgs, "--description", description)
			}
			if budget > 0 {
				createArgs = append(createArgs, "--budget", fmt.Sprintf("%.2f", budget))
			}
			if owner != "" {
				createArgs = append(createArgs, "--owner", owner)
			}

			return pc.app.Project(createArgs)
		},
	}

	cmd.Flags().String("description", "", "Project description")
	cmd.Flags().Float64("budget", 0, "Budget limit")
	cmd.Flags().String("owner", "", "Project owner")

	return cmd
}

// createInfoCommand creates the info subcommand
func (pc *ProjectCobraCommands) createInfoCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "info <name>",
		Short: "Show detailed project information",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return pc.app.Project([]string{"info", args[0]})
		},
	}
}

// createDeleteCommand creates the delete subcommand
func (pc *ProjectCobraCommands) createDeleteCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "delete <name>",
		Short: "Delete a project",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return pc.app.Project([]string{"delete", args[0]})
		},
	}
}

// createMembersCommand creates the members management subcommand
func (pc *ProjectCobraCommands) createMembersCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "members <project>",
		Short: "Manage project members",
		Args:  cobra.ExactArgs(1),
	}

	// Members subcommands
	cmd.AddCommand(
		&cobra.Command{
			Use:   "add <email> <role>",
			Short: "Add a member to project",
			Args:  cobra.ExactArgs(2),
			RunE: func(cmd *cobra.Command, args []string) error {
				projectName := cmd.Parent().Flag("project").Value.String()
				if projectName == "" {
					// Get project from parent args
					parentArgs := cmd.Parent().Flags().Args()
					if len(parentArgs) > 0 {
						projectName = parentArgs[0]
					}
				}
				return pc.app.Project([]string{"members", projectName, "add", args[0], args[1]})
			},
		},
		&cobra.Command{
			Use:   "remove <email>",
			Short: "Remove a member from project",
			Args:  cobra.ExactArgs(1),
			RunE: func(cmd *cobra.Command, args []string) error {
				projectName := cmd.Parent().Flag("project").Value.String()
				if projectName == "" {
					// Get project from parent args
					parentArgs := cmd.Parent().Flags().Args()
					if len(parentArgs) > 0 {
						projectName = parentArgs[0]
					}
				}
				return pc.app.Project([]string{"members", projectName, "remove", args[0]})
			},
		},
		&cobra.Command{
			Use:   "list",
			Short: "List project members",
			RunE: func(cmd *cobra.Command, args []string) error {
				projectName := cmd.Parent().Flag("project").Value.String()
				if projectName == "" {
					// Get project from parent args
					parentArgs := cmd.Parent().Flags().Args()
					if len(parentArgs) > 0 {
						projectName = parentArgs[0]
					}
				}
				return pc.app.Project([]string{"members", projectName})
			},
		},
	)

	return cmd
}

// createBudgetCommand creates the budget management subcommand
func (pc *ProjectCobraCommands) createBudgetCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "budget",
		Short: "Manage project budgets and cost tracking",
		Long: `Configure project budgets, set spending limits, configure alerts,
and enable cost tracking for research projects.`,
	}

	// Budget subcommands
	cmd.AddCommand(
		&cobra.Command{
			Use:   "status <project>",
			Short: "Show budget status and spending",
			Args:  cobra.ExactArgs(1),
			RunE: func(cmd *cobra.Command, args []string) error {
				return pc.app.Project([]string{"budget", "status", args[0]})
			},
		},
		&cobra.Command{
			Use:   "set <project> <amount>",
			Short: "Set or enable project budget",
			Long:  `Set a budget for a project and enable cost tracking. Amount should be in USD.`,
			Args:  cobra.ExactArgs(2),
			RunE: func(cmd *cobra.Command, args []string) error {
				return pc.app.Project([]string{"budget", "set", args[0], args[1]})
			},
		},
		&cobra.Command{
			Use:   "disable <project>",
			Short: "Disable cost tracking for project",
			Long:  `Disable budget tracking and cost monitoring for a project.`,
			Args:  cobra.ExactArgs(1),
			RunE: func(cmd *cobra.Command, args []string) error {
				return pc.app.Project([]string{"budget", "disable", args[0]})
			},
		},
		&cobra.Command{
			Use:   "history <project>",
			Short: "Show budget spending history",
			Args:  cobra.ExactArgs(1),
			RunE: func(cmd *cobra.Command, args []string) error {
				return pc.app.Project([]string{"budget", "history", args[0]})
			},
		},
	)

	return cmd
}

// createInstancesCommand creates the instances subcommand
func (pc *ProjectCobraCommands) createInstancesCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "instances <project>",
		Short: "List instances in a project",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return pc.app.Project([]string{"instances", args[0]})
		},
	}
}

// createTemplatesCommand creates the templates subcommand
func (pc *ProjectCobraCommands) createTemplatesCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "templates <project>",
		Short: "List templates in a project",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return pc.app.Project([]string{"templates", args[0]})
		},
	}
}
