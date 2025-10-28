// Package cli implements template application commands for Prism.
//
// These commands enable applying templates to already running instances,
// allowing for incremental environment evolution without instance recreation.
package cli

import (
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/scttfrdmn/prism/pkg/types"
)

// Apply handles the apply command using Command Pattern (SOLID: Single Responsibility)
func (a *App) Apply(args []string) error {
	// Create and execute template apply command
	applyCmd := NewTemplateApplyCommand(a.apiClient)
	return applyCmd.Execute(args)
}

// Diff handles the diff command using Command Pattern (SOLID: Single Responsibility)
func (a *App) Diff(args []string) error {
	// Create and execute template diff command
	diffCmd := NewTemplateDiffCommand(a.apiClient)
	return diffCmd.Execute(args)
}

// Layers handles the layers command
func (a *App) Layers(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: cws layers <workspace-name>")
	}

	instanceName := args[0]

	// Ensure daemon is running (auto-start if needed)
	if err := a.ensureDaemonRunning(); err != nil {
		return err
	}

	// Get applied templates via API
	layers, err := a.apiClient.GetInstanceLayers(a.ctx, instanceName)
	if err != nil {
		return fmt.Errorf("failed to get instance layers: %w", err)
	}

	if len(layers) == 0 {
		fmt.Printf("📋 No templates applied to instance '%s'\n", instanceName)
		fmt.Printf("💡 Apply a template with: cws apply <template> %s\n", instanceName)
		return nil
	}

	fmt.Printf("📋 Applied templates for instance '%s':\n\n", instanceName)

	w := tabwriter.NewWriter(os.Stdout, 0, 8, 2, ' ', 0)
	_, _ = fmt.Fprintln(w, "LAYER\tTEMPLATE\tAPPLIED\tPACKAGE MANAGER\tPACKAGES\tSERVICES\tUSERS\tCHECKPOINT")

	for i, layer := range layers {
		_, _ = fmt.Fprintf(w, "%d\t%s\t%s\t%s\t%d\t%d\t%d\t%s\n",
			i+1,
			layer.Name,
			layer.AppliedAt.Format("2006-01-02 15:04"),
			layer.PackageManager,
			len(layer.PackagesInstalled),
			len(layer.ServicesConfigured),
			len(layer.UsersCreated),
			layer.RollbackCheckpoint,
		)
	}

	_ = w.Flush()

	fmt.Printf("\n💡 Use 'cws rollback %s --to-checkpoint=<checkpoint>' to rollback to a specific layer\n", instanceName)
	fmt.Printf("💡 Use 'cws rollback %s' to rollback to the previous checkpoint\n", instanceName)

	return nil
}

// Rollback handles the rollback command
func (a *App) Rollback(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: cws rollback <workspace-name> [--to-checkpoint=<checkpoint-id>]")
	}

	instanceName := args[0]
	var checkpointID string

	// Parse options
	for i := 1; i < len(args); i++ {
		arg := args[i]
		if strings.HasPrefix(arg, "--to-checkpoint=") {
			checkpointID = strings.TrimPrefix(arg, "--to-checkpoint=")
		} else {
			return fmt.Errorf("unknown option: %s", arg)
		}
	}

	// Ensure daemon is running (auto-start if needed)
	if err := a.ensureDaemonRunning(); err != nil {
		return err
	}

	// If no checkpoint specified, get the latest one
	if checkpointID == "" {
		layers, err := a.apiClient.GetInstanceLayers(a.ctx, instanceName)
		if err != nil {
			return fmt.Errorf("failed to get instance layers: %w", err)
		}

		if len(layers) == 0 {
			return fmt.Errorf("no templates applied to instance '%s' - nothing to rollback", instanceName)
		}

		// Get the second-to-last checkpoint (rollback from current state)
		if len(layers) == 1 {
			return fmt.Errorf("only one template applied to instance '%s' - no previous state to rollback to", instanceName)
		}

		checkpointID = layers[len(layers)-2].RollbackCheckpoint
		fmt.Printf("🔄 Rolling back to checkpoint: %s\n", checkpointID)
	}

	// Perform rollback via API
	err := a.apiClient.RollbackInstance(a.ctx, types.RollbackRequest{
		InstanceName: instanceName,
		CheckpointID: checkpointID,
	})
	if err != nil {
		return fmt.Errorf("failed to rollback instance: %w", err)
	}

	fmt.Printf("✅ Successfully rolled back instance '%s' to checkpoint '%s'\n", instanceName, checkpointID)
	fmt.Printf("💡 Use 'cws layers %s' to see the current state\n", instanceName)
	fmt.Printf("💡 Use 'cws list' to verify the instance is healthy\n")

	return nil
}
