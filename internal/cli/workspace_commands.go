package cli

import (
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
  prism workspace launch python-ml my-project    # Launch new workspace
  prism workspace list                            # List all workspaces
  prism workspace stop my-workspace               # Stop a workspace
  prism workspace connect my-workspace            # Connect via SSH`,
		GroupID: "core",
	}

	// Add subcommands
	cmd.AddCommand(f.createLaunchCommand())
	cmd.AddCommand(f.createListCommand())
	cmd.AddCommand(f.createCostCommand())
	cmd.AddCommand(f.createStartCommand())
	cmd.AddCommand(f.createStopCommand())
	cmd.AddCommand(f.createDeleteCommand())
	cmd.AddCommand(f.createHibernateCommand())
	cmd.AddCommand(f.createResumeCommand())
	cmd.AddCommand(f.createConnectCommand())
	cmd.AddCommand(f.createExecCommand())
	cmd.AddCommand(f.createWebCommand())

	// v0.20.0: SSM file operations (#30)
	fileOps := NewFileOpsCobraCommands(f.app)
	cmd.AddCommand(fileOps.CreateFilesCommand())

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

// appendBoolFlag appends a CLI flag to args if the named bool flag is set.
func appendBoolFlag(cmd *cobra.Command, args []string, flagName, argName string) []string {
	if v, _ := cmd.Flags().GetBool(flagName); v {
		return append(args, argName)
	}
	return args
}

// appendStringFlag appends a CLI flag + value to args if the named string flag is non-empty.
func appendStringFlag(cmd *cobra.Command, args []string, flagName, argName string) []string {
	if v, _ := cmd.Flags().GetString(flagName); v != "" {
		return append(args, argName, v)
	}
	return args
}

func (f *WorkspaceCommandFactory) buildLaunchArgs(cmd *cobra.Command, args []string) error {
	args = appendBoolFlag(cmd, args, "hibernation", "--hibernation")
	args = appendBoolFlag(cmd, args, "spot", "--spot")
	args = appendStringFlag(cmd, args, "size", "--size")
	args = appendStringFlag(cmd, args, "subnet", "--subnet")
	args = appendStringFlag(cmd, args, "vpc", "--vpc")
	args = appendStringFlag(cmd, args, "project", "--project")
	args = appendStringFlag(cmd, args, "funding", "--funding")
	args = appendBoolFlag(cmd, args, "wait", "--wait")
	args = appendBoolFlag(cmd, args, "dry-run", "--dry-run")
	if params, _ := cmd.Flags().GetStringArray("param"); len(params) > 0 {
		for _, param := range params {
			args = append(args, "--param", param)
		}
	}
	args = appendStringFlag(cmd, args, "research-user", "--research-user")
	args = appendStringFlag(cmd, args, "ssh-key", "--ssh-key")
	args = appendBoolFlag(cmd, args, "quiet", "--quiet")
	args = appendBoolFlag(cmd, args, "no-progress", "--no-progress")
	args = appendBoolFlag(cmd, args, "yes", "--yes")
	args = appendStringFlag(cmd, args, "capacity-block", "--capacity-block")
	args = appendBoolFlag(cmd, args, "request-approval", "--request-approval")
	args = appendStringFlag(cmd, args, "approval", "--approval")
	args = appendStringFlag(cmd, args, "slack-workspace", "--slack-workspace")
	args = appendStringFlag(cmd, args, "active-processes", "--active-processes")
	return f.app.Launch(args)
}

func (f *WorkspaceCommandFactory) addLaunchFlags(cmd *cobra.Command) {
	cmd.Flags().Bool("hibernation", false, "Enable hibernation support")
	cmd.Flags().Bool("spot", false, "Use spot instances for cost savings")
	cmd.Flags().String("size", "", "Workspace size: XS=1vCPU,2GB | S=2vCPU,4GB | M=2vCPU,8GB | L=4vCPU,16GB | XL=8vCPU,32GB")
	cmd.Flags().String("subnet", "", "Specify subnet ID")
	cmd.Flags().String("vpc", "", "Specify VPC ID")
	cmd.Flags().String("project", "", "Associate with project")
	cmd.Flags().String("funding", "", "Specify funding source (budget allocation) - defaults to project's default allocation")
	cmd.Flags().Bool("wait", false, "Wait and display launch progress")
	cmd.Flags().Bool("dry-run", false, "Validate configuration without launching")
	cmd.Flags().StringArray("param", []string{}, "Template parameter (name=value)")
	cmd.Flags().String("research-user", "", "Automatically create and provision research user")
	cmd.Flags().String("ssh-key", "", "SSH key name for instance access (defaults to first available key)")
	cmd.Flags().Bool("quiet", false, "Suppress progress output (for scripting)")
	cmd.Flags().Bool("no-progress", false, "Disable progress monitoring")
	cmd.Flags().BoolP("yes", "y", false, "Skip cost confirmation prompt")
	cmd.Flags().String("capacity-block", "", "Pin launch to a pre-reserved EC2 Capacity Block ID (#63)")
	cmd.Flags().Bool("request-approval", false, "Request PI/admin approval instead of launching (#495)")
	cmd.Flags().String("approval", "", "Launch using a pre-approved request ID (#495)")
	cmd.Flags().String("slack-workspace", "", "Slack workspace ID for lifecycle notifications (e.g., T0AU2S6FU86)")
	cmd.Flags().String("active-processes", "", "Processes that keep instance active, preventing idle shutdown (e.g., rsession or rsession,jupyter)")
}

func (f *WorkspaceCommandFactory) createListCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List all workspaces",
		Long:  `List all cloud workspaces with their status, costs, and metadata.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Collect flags into args for parseListFlags
			var flagArgs []string
			if refresh, _ := cmd.Flags().GetBool("refresh"); refresh {
				flagArgs = append(flagArgs, "--refresh")
			}
			if detailed, _ := cmd.Flags().GetBool("detailed"); detailed {
				flagArgs = append(flagArgs, "--detailed")
			}
			if project, _ := cmd.Flags().GetString("project"); project != "" {
				flagArgs = append(flagArgs, "--project", project)
			}
			return f.app.List(flagArgs)
		},
	}
	cmd.Flags().BoolP("refresh", "r", false, "Refresh instance data from AWS (slower but accurate)")
	cmd.Flags().BoolP("detailed", "d", false, "Show detailed instance information")
	cmd.Flags().String("project", "", "Filter by project ID")
	return cmd
}

func (f *WorkspaceCommandFactory) createCostCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "cost",
		Short: "Detailed cost breakdown with savings analysis",
		Long: `Show detailed cost analysis for all workspaces, including running time,
per-minute costs, monthly estimates, and institutional discount savings.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			var flagArgs []string
			if project, _ := cmd.Flags().GetString("project"); project != "" {
				flagArgs = append(flagArgs, "--project", project)
			}
			return f.app.ListCost(flagArgs)
		},
	}
	cmd.Flags().String("project", "", "Filter by project ID")
	return cmd
}

func (f *WorkspaceCommandFactory) createStartCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "start <name>",
		Short: "Start a stopped workspace",
		Long: `Start a stopped workspace.

By default, the command initiates the start and returns immediately.
Use --wait to block until the workspace is running.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			wait, _ := cmd.Flags().GetBool("wait")
			return f.app.StartWithWait(args[0], wait)
		},
	}
	cmd.Flags().Bool("wait", false, "Wait for the start operation to complete")
	return cmd
}

func (f *WorkspaceCommandFactory) createStopCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "stop <name>",
		Short: "Stop a running workspace",
		Long: `Stop a running workspace.

By default, the command initiates the stop and returns immediately.
Use --wait to block until the operation completes.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			wait, _ := cmd.Flags().GetBool("wait")
			return f.app.StopWithWait(args[0], wait)
		},
	}
	cmd.Flags().Bool("wait", false, "Wait for the stop operation to complete")
	return cmd
}

func (f *WorkspaceCommandFactory) createDeleteCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "delete <name>",
		Aliases: []string{"rm", "remove"},
		Short:   "Delete a workspace",
		Long: `Delete a workspace and its associated resources.

By default, the command initiates the deletion and returns immediately.
Use --wait to block until the workspace is fully terminated.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			wait, _ := cmd.Flags().GetBool("wait")
			return f.app.DeleteWithWait(args[0], wait)
		},
	}
	cmd.Flags().Bool("wait", false, "Wait for the deletion to complete")
	return cmd
}

func (f *WorkspaceCommandFactory) createHibernateCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "hibernate <name>",
		Short: "Hibernate a workspace (save state, reduce costs)",
		Long: `Hibernate a workspace to save memory state to disk and stop the workspace.
This reduces costs while preserving your work session for fast resume.

By default, the command initiates hibernation and returns immediately.
Use --wait to block until the workspace is hibernated.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			wait, _ := cmd.Flags().GetBool("wait")
			return f.app.HibernateWithWait(args[0], wait)
		},
	}
	cmd.Flags().Bool("wait", false, "Wait for the hibernation to complete")
	return cmd
}

func (f *WorkspaceCommandFactory) createResumeCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "resume <name>",
		Short: "Resume a hibernated workspace",
		Long: `Resume a hibernated workspace, restoring memory state and continuing your work session.

By default, the command initiates the resume and returns immediately.
Use --wait to block until the workspace is running.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			wait, _ := cmd.Flags().GetBool("wait")
			return f.app.ResumeWithWait(args[0], wait)
		},
	}
	cmd.Flags().Bool("wait", false, "Wait for the resume operation to complete")
	return cmd
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
