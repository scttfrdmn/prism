// Package cli - Storage Cobra Command Layer
//
// ARCHITECTURE NOTE: This file defines the user-facing CLI interface for storage commands.
// The actual business logic is in storage_impl.go (StorageCommands).
//
// This separation follows the Facade/Adapter pattern:
//   - storage_cobra.go: CLI interface (THIS FILE - Cobra commands, flag parsing, help text)
//   - storage_impl.go: Business logic (API calls, formatting, error handling)
//
// This Cobra layer is responsible for:
//   - Defining command structure and subcommands
//   - Parsing and validating flags
//   - Providing help text and examples
//   - Delegating to StorageCommands for execution
package cli

import (
	"github.com/spf13/cobra"
)

// StorageCobraCommands handles storage/volume commands (Cobra layer)
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
		Short: "Manage CloudWorkstation storage (all types)",
		Long: `Manage all storage types (local and shared).

'cws storage list' shows all storage volumes (both local EBS and shared EFS).
'cws storage create' creates local storage (EBS volumes).

For shared storage (EFS), use 'cws volume' commands.`,
	}

	// Create commands separately to add flags
	createCmd := &cobra.Command{
		Use:   "create <name>",
		Short: "Create a new EBS volume",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			size, _ := cmd.Flags().GetString("size")
			volumeType, _ := cmd.Flags().GetString("type")
			createArgs := []string{"create", args[0]}
			if size != "" {
				createArgs = append(createArgs, "--size", size)
			}
			if volumeType != "" {
				createArgs = append(createArgs, "--type", volumeType)
			}
			return sc.app.Storage(createArgs)
		},
	}
	createCmd.Flags().String("size", "L", "Volume size (S/M/L/XL or custom like 100GB)")
	createCmd.Flags().String("type", "gp3", "Volume type (gp3/io2/st1)")

	deleteCmd := &cobra.Command{
		Use:   "delete <volume>",
		Short: "Delete a storage volume",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			force, _ := cmd.Flags().GetBool("force")
			deleteArgs := []string{"delete", args[0]}
			if force {
				deleteArgs = append(deleteArgs, "--force")
			}
			return sc.app.Storage(deleteArgs)
		},
	}
	deleteCmd.Flags().Bool("force", false, "Force delete without confirmation")

	// Add subcommands
	cmd.AddCommand(
		createCmd,
		&cobra.Command{
			Use:   "attach <volume> <instance>",
			Short: "Attach volume to instance",
			Args:  cobra.ExactArgs(2),
			RunE: func(cmd *cobra.Command, args []string) error {
				return sc.app.Storage([]string{"attach", args[0], args[1]})
			},
		},
		&cobra.Command{
			Use:   "detach <volume>",
			Short: "Detach volume from instance",
			Args:  cobra.ExactArgs(1),
			RunE: func(cmd *cobra.Command, args []string) error {
				return sc.app.Storage([]string{"detach", args[0]})
			},
		},
		&cobra.Command{
			Use:   "list",
			Short: "List all storage volumes",
			RunE: func(cmd *cobra.Command, args []string) error {
				return sc.app.Storage([]string{"list"})
			},
		},
		&cobra.Command{
			Use:   "info <name>",
			Short: "Show detailed storage volume information",
			Args:  cobra.ExactArgs(1),
			RunE: func(cmd *cobra.Command, args []string) error {
				return sc.app.Storage([]string{"info", args[0]})
			},
		},
		deleteCmd,
	)

	return cmd
}

// CreateVolumeCommand creates the volume command for EFS volumes
func (sc *StorageCobraCommands) CreateVolumeCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "volume",
		Short: "Manage shared storage volumes (EFS)",
		Long: `Create, mount, unmount, and manage shared storage (EFS volumes).

Shared storage can be mounted to multiple workspaces simultaneously,
making it ideal for collaborative projects and shared datasets.

Use 'cws storage' for local storage (EBS volumes).`,
	}

	// Create command with flags
	createCmd := &cobra.Command{
		Use:   "create <name>",
		Short: "Create a new EFS volume",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			performance, _ := cmd.Flags().GetString("performance")
			throughput, _ := cmd.Flags().GetString("throughput")
			createArgs := []string{"create", args[0]}
			if performance != "" {
				createArgs = append(createArgs, "--performance", performance)
			}
			if throughput != "" {
				createArgs = append(createArgs, "--throughput", throughput)
			}
			return sc.app.Volume(createArgs)
		},
	}
	createCmd.Flags().String("performance", "generalPurpose", "Performance mode (generalPurpose/maxIO)")
	createCmd.Flags().String("throughput", "bursting", "Throughput mode (bursting/provisioned)")

	// Add subcommands
	cmd.AddCommand(
		createCmd,
		&cobra.Command{
			Use:   "list",
			Short: "List all EFS volumes",
			RunE: func(cmd *cobra.Command, args []string) error {
				return sc.app.Volume([]string{"list"})
			},
		},
		&cobra.Command{
			Use:   "info <name>",
			Short: "Show detailed volume information",
			Args:  cobra.ExactArgs(1),
			RunE: func(cmd *cobra.Command, args []string) error {
				return sc.app.Volume([]string{"info", args[0]})
			},
		},
		&cobra.Command{
			Use:   "mount <volume> <instance> [mount-point]",
			Short: "Mount volume to instance",
			Args:  cobra.RangeArgs(2, 3),
			RunE: func(cmd *cobra.Command, args []string) error {
				mountArgs := []string{"mount", args[0], args[1]}
				if len(args) == 3 {
					mountArgs = append(mountArgs, args[2])
				}
				return sc.app.Volume(mountArgs)
			},
		},
		&cobra.Command{
			Use:   "unmount <volume> <instance>",
			Short: "Unmount volume from instance",
			Args:  cobra.ExactArgs(2),
			RunE: func(cmd *cobra.Command, args []string) error {
				return sc.app.Volume([]string{"unmount", args[0], args[1]})
			},
		},
		&cobra.Command{
			Use:   "delete <volume>",
			Short: "Delete an EFS volume",
			Args:  cobra.ExactArgs(1),
			RunE: func(cmd *cobra.Command, args []string) error {
				return sc.app.Volume([]string{"delete", args[0]})
			},
		},
	)

	return cmd
}
