package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

// WorkspaceCommandFactory creates the unified workspace command group
type WorkspaceCommandFactory struct {
	app *App
}

// CreateCommand creates the workspace command group
func (f *WorkspaceCommandFactory) CreateCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "workspace",
		Short: "Manage research computing workspaces",
		Long: `Unified workspace management for launching, monitoring, and controlling
cloud research computing environments.

Examples:
  cws workspace launch python-ml my-project    # Launch new workspace
  cws workspace list                            # List all workspaces
  cws workspace stop my-workspace               # Stop a workspace
  cws workspace connect my-workspace            # Connect via SSH`,
		GroupID: "core",
	}

	// Add subcommands
	cmd.AddCommand(f.createLaunchCommand())
	cmd.AddCommand(f.createListCommand())
	cmd.AddCommand(f.createStartCommand())
	cmd.AddCommand(f.createStopCommand())
	cmd.AddCommand(f.createDeleteCommand())
	cmd.AddCommand(f.createHibernateCommand())
	cmd.AddCommand(f.createResumeCommand())
	cmd.AddCommand(f.createConnectCommand())
	cmd.AddCommand(f.createExecCommand())
	cmd.AddCommand(f.createWebCommand())

	return cmd
}

func (f *WorkspaceCommandFactory) createLaunchCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "launch <template> <name>",
		Short: "Launch a new workspace",
		Long:  `Launch a new cloud workspace from a template with smart defaults.`,
		Args:  cobra.MinimumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			return f.buildLaunchArgs(cmd, args)
		},
	}
	f.addLaunchFlags(cmd)
	return cmd
}

func (f *WorkspaceCommandFactory) buildLaunchArgs(cmd *cobra.Command, args []string) error {
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

func (f *WorkspaceCommandFactory) addLaunchFlags(cmd *cobra.Command) {
	cmd.Flags().Bool("hibernation", false, "Enable hibernation support")
	cmd.Flags().Bool("spot", false, "Use spot instances for cost savings")
	cmd.Flags().String("size", "", "Workspace size: XS=1vCPU,2GB | S=2vCPU,4GB | M=2vCPU,8GB | L=4vCPU,16GB | XL=8vCPU,32GB")
	cmd.Flags().String("subnet", "", "Specify subnet ID")
	cmd.Flags().String("vpc", "", "Specify VPC ID")
	cmd.Flags().String("project", "", "Associate with project")
	cmd.Flags().Bool("wait", false, "Wait and display launch progress")
	cmd.Flags().Bool("dry-run", false, "Validate configuration without launching")
	cmd.Flags().StringArray("param", []string{}, "Template parameter (name=value)")
	cmd.Flags().String("research-user", "", "Automatically create and provision research user")
}

func (f *WorkspaceCommandFactory) createListCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all workspaces",
		Long:  `List all cloud workspaces with their status, costs, and metadata.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return f.app.List(args)
		},
	}
}

func (f *WorkspaceCommandFactory) createStartCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "start <name>",
		Short: "Start a stopped workspace",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return f.app.Start(args)
		},
	}
}

func (f *WorkspaceCommandFactory) createStopCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "stop <name>",
		Short: "Stop a running workspace",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return f.app.Stop(args)
		},
	}
}

func (f *WorkspaceCommandFactory) createDeleteCommand() *cobra.Command {
	return &cobra.Command{
		Use:     "delete <name>",
		Aliases: []string{"rm", "remove"},
		Short:   "Delete a workspace",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return f.app.Delete(args)
		},
	}
}

func (f *WorkspaceCommandFactory) createHibernateCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "hibernate <name>",
		Short: "Hibernate a workspace (save state, reduce costs)",
		Long: `Hibernate a workspace to save memory state to disk and stop the workspace.
This reduces costs while preserving your work session for fast resume.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return f.app.Hibernate(args)
		},
	}
}

func (f *WorkspaceCommandFactory) createResumeCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "resume <name>",
		Short: "Resume a hibernated workspace",
		Long:  `Resume a hibernated workspace, restoring memory state and continuing your work session.`,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return f.app.Resume(args)
		},
	}
}

func (f *WorkspaceCommandFactory) createConnectCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "connect <name>",
		Short: "Connect to workspace via SSH",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return f.app.Connect(args)
		},
	}
}

func (f *WorkspaceCommandFactory) createExecCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "exec <name> <command>",
		Short: "Execute a command on workspace",
		Args:  cobra.MinimumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			return f.app.Exec(args)
		},
	}
}

func (f *WorkspaceCommandFactory) createWebCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "web <name>",
		Short: "Manage workspace web services",
		Long: `Access web services running on workspace (Jupyter, RStudio, etc.).
Lists available services and provides access URLs.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return f.app.Web(args)
		},
	}
	return cmd
}

// Helper function to print deprecation warning
func printDeprecationWarning(oldCmd, newCmd string) {
	fmt.Printf("⚠️  Deprecated: Use '%s' instead of '%s'\n", newCmd, oldCmd)
	fmt.Printf("   The old command will be removed in v1.0.0\n\n")
}
