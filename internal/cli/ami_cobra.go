package cli

import (
	"github.com/spf13/cobra"
)

// AMICobraCommands handles AMI management commands
type AMICobraCommands struct {
	app *App
}

// NewAMICobraCommands creates new AMI cobra commands
func NewAMICobraCommands(app *App) *AMICobraCommands {
	return &AMICobraCommands{app: app}
}

// CreateAMICommand creates the ami command with subcommands
func (ac *AMICobraCommands) CreateAMICommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "ami",
		Short: "Manage AMIs for fast instance launching",
		Long: `Build, manage, and deploy AMIs for faster CloudWorkstation deployments.

AMIs enable fast launching by pre-building template environments, reducing
launch times from 5-8 minutes to under 30 seconds.`,
	}

	// Add all AMI subcommands
	cmd.AddCommand(
		ac.createBuildCommand(),
		ac.createListCommand(),
		ac.createValidateCommand(),
		ac.createPublishCommand(),
		ac.createSaveCommand(),
		ac.createResolveCommand(),
		ac.createTestCommand(),
		ac.createCostsCommand(),
		ac.createPreviewCommand(),
		ac.createCreateCommand(),
		ac.createStatusCommand(),
		ac.createCleanupCommand(),
		ac.createDeleteCommand(),
		ac.createSnapshotCommand(),
		ac.createCheckFreshnessCommand(),
	)

	return cmd
}

func (ac *AMICobraCommands) createBuildCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "build <template-name>",
		Short: "Build a new AMI from a template",
		Long: `Build a new AMI from a CloudWorkstation template for faster launching.

This creates a pre-configured AMI that reduces launch times from 5-8 minutes
to under 30 seconds for subsequent launches.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			region, _ := cmd.Flags().GetString("region")
			force, _ := cmd.Flags().GetBool("force")

			amiArgs := []string{"build", args[0]}
			if region != "" {
				amiArgs = append(amiArgs, "--region", region)
			}
			if force {
				amiArgs = append(amiArgs, "--force")
			}

			return ac.app.AMI(amiArgs)
		},
	}

	cmd.Flags().String("region", "", "AWS region to build AMI in")
	cmd.Flags().Bool("force", false, "Force rebuild even if AMI exists")

	return cmd
}

func (ac *AMICobraCommands) createListCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list [template-name]",
		Short: "List available AMIs",
		Long:  `List AMIs available for templates, optionally filtered by template name.`,
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			region, _ := cmd.Flags().GetString("region")

			amiArgs := []string{"list"}
			if len(args) > 0 {
				amiArgs = append(amiArgs, args[0])
			}
			if region != "" {
				amiArgs = append(amiArgs, "--region", region)
			}

			return ac.app.AMI(amiArgs)
		},
	}

	cmd.Flags().String("region", "", "AWS region to list AMIs from")

	return cmd
}

func (ac *AMICobraCommands) createValidateCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "validate <template-name>",
		Short: "Validate template for AMI building",
		Long:  `Validate that a template can be successfully built into an AMI.`,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return ac.app.AMI([]string{"validate", args[0]})
		},
	}
}

func (ac *AMICobraCommands) createPublishCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "publish <ami-id>",
		Short: "Publish AMI to template registry",
		Long:  `Publish a built AMI to the template registry for community use.`,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			description, _ := cmd.Flags().GetString("description")
			public, _ := cmd.Flags().GetBool("public")

			amiArgs := []string{"publish", args[0]}
			if description != "" {
				amiArgs = append(amiArgs, "--description", description)
			}
			if public {
				amiArgs = append(amiArgs, "--public")
			}

			return ac.app.AMI(amiArgs)
		},
	}

	cmd.Flags().String("description", "", "Description for published AMI")
	cmd.Flags().Bool("public", false, "Make AMI publicly available")

	return cmd
}

func (ac *AMICobraCommands) createSaveCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "save <workspace-name> <ami-name>",
		Short: "Save instance as AMI",
		Long:  `Save a running instance as an AMI for faster future launches.`,
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			return ac.app.AMI([]string{"save", args[0], args[1]})
		},
	}
}

func (ac *AMICobraCommands) createResolveCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "resolve <template-name>",
		Short: "Resolve best AMI for template",
		Long:  `Find the best available AMI for a template in the current region.`,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return ac.app.AMI([]string{"resolve", args[0]})
		},
	}
}

func (ac *AMICobraCommands) createTestCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "test <ami-id>",
		Short: "Test AMI functionality",
		Long:  `Test an AMI by launching a temporary instance and validating functionality.`,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return ac.app.AMI([]string{"test", args[0]})
		},
	}
}

func (ac *AMICobraCommands) createCostsCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "costs <template-name>",
		Short: "Show AMI cost analysis",
		Long:  `Show cost analysis comparing AMI-based vs package-based launches.`,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return ac.app.AMI([]string{"costs", args[0]})
		},
	}
}

func (ac *AMICobraCommands) createPreviewCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "preview <template-name>",
		Short: "Preview AMI build process",
		Long:  `Preview what would be included in an AMI build without actually building.`,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return ac.app.AMI([]string{"preview", args[0]})
		},
	}
}

func (ac *AMICobraCommands) createCreateCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "create <template-name>",
		Short: "Create AMI from template",
		Long:  `Create a new AMI from a template (alias for build).`,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return ac.app.AMI([]string{"create", args[0]})
		},
	}
}

func (ac *AMICobraCommands) createStatusCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "status <ami-id>",
		Short: "Show AMI status and details",
		Long:  `Show detailed status information for a specific AMI.`,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return ac.app.AMI([]string{"status", args[0]})
		},
	}
}

func (ac *AMICobraCommands) createCleanupCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "cleanup",
		Short: "Clean up old AMIs",
		Long:  `Clean up old or unused AMIs to reduce storage costs.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			dryRun, _ := cmd.Flags().GetBool("dry-run")
			force, _ := cmd.Flags().GetBool("force")

			amiArgs := []string{"cleanup"}
			if dryRun {
				amiArgs = append(amiArgs, "--dry-run")
			}
			if force {
				amiArgs = append(amiArgs, "--force")
			}

			return ac.app.AMI(amiArgs)
		},
	}

	cmd.Flags().Bool("dry-run", false, "Show what would be cleaned up without doing it")
	cmd.Flags().Bool("force", false, "Force cleanup without confirmation")

	return cmd
}

func (ac *AMICobraCommands) createDeleteCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delete <ami-id>",
		Short: "Delete a specific AMI",
		Long:  `Delete a specific AMI and its associated snapshots.`,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			force, _ := cmd.Flags().GetBool("force")

			amiArgs := []string{"delete", args[0]}
			if force {
				amiArgs = append(amiArgs, "--force")
			}

			return ac.app.AMI(amiArgs)
		},
	}

	cmd.Flags().Bool("force", false, "Force delete without confirmation")

	return cmd
}

func (ac *AMICobraCommands) createSnapshotCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "snapshot",
		Short: "Manage AMI snapshots",
		Long:  `Create, list, and manage AMI snapshots.`,
	}

	// Add snapshot subcommands
	cmd.AddCommand(
		&cobra.Command{
			Use:   "list",
			Short: "List AMI snapshots",
			RunE: func(cmd *cobra.Command, args []string) error {
				return ac.app.AMI([]string{"snapshot", "list"})
			},
		},
		&cobra.Command{
			Use:   "create <ami-id> <snapshot-name>",
			Short: "Create snapshot of AMI",
			Args:  cobra.ExactArgs(2),
			RunE: func(cmd *cobra.Command, args []string) error {
				return ac.app.AMI([]string{"snapshot", "create", args[0], args[1]})
			},
		},
		&cobra.Command{
			Use:   "restore <snapshot-id> <new-ami-name>",
			Short: "Restore AMI from snapshot",
			Args:  cobra.ExactArgs(2),
			RunE: func(cmd *cobra.Command, args []string) error {
				return ac.app.AMI([]string{"snapshot", "restore", args[0], args[1]})
			},
		},
		&cobra.Command{
			Use:   "delete <snapshot-id>",
			Short: "Delete AMI snapshot",
			Args:  cobra.ExactArgs(1),
			RunE: func(cmd *cobra.Command, args []string) error {
				return ac.app.AMI([]string{"snapshot", "delete", args[0]})
			},
		},
	)

	return cmd
}

func (ac *AMICobraCommands) createCheckFreshnessCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "check-freshness",
		Short: "Check AMI freshness against latest versions",
		Long: `Check static AMI IDs against latest versions from AWS SSM Parameter Store.

This command validates all static AMI mappings to identify outdated AMIs that
should be updated to the latest version for optimal security and performance.

SSM-supported distributions (automatically updated):
  - Ubuntu 24.04, 22.04, 20.04
  - Amazon Linux 2023, 2
  - Debian 12

Static-only distributions (manual updates required):
  - Rocky Linux 10, 9
  - RHEL 9
  - Alpine 3.20`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return ac.app.AMI([]string{"check-freshness"})
		},
	}

	return cmd
}
