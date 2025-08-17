// Package cli implements template application commands for CloudWorkstation.
//
// These commands enable applying templates to already running instances,
// allowing for incremental environment evolution without instance recreation.
package cli

import (
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/scttfrdmn/cloudworkstation/pkg/templates"
	"github.com/scttfrdmn/cloudworkstation/pkg/types"
)

// Apply handles the apply command
func (a *App) Apply(args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("usage: cws apply <template> <instance-name> [options]\n" +
			"  options: --dry-run --force --with <package-manager>")
	}

	templateName := args[0]
	instanceName := args[1]

	// Parse options
	req := templates.ApplyRequest{
		InstanceName: instanceName,
		DryRun:       false,
		Force:        false,
	}

	// Parse additional flags
	for i := 2; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "--dry-run":
			req.DryRun = true
		case arg == "--force":
			req.Force = true
		case arg == "--with" && i+1 < len(args):
			packageManager := args[i+1]
			// Validate supported package managers
			supportedManagers := []string{"conda", "apt", "dnf", "spack", "pip", "ami"}
			supported := false
			for _, mgr := range supportedManagers {
				if packageManager == mgr {
					supported = true
					break
				}
			}
			if !supported {
				return fmt.Errorf("unsupported package manager: %s (supported: conda, apt, dnf, spack, pip, ami)", packageManager)
			}
			req.PackageManager = packageManager
			i++
		default:
			return fmt.Errorf("unknown option: %s", arg)
		}
	}

	// Check daemon is running
	if err := a.apiClient.Ping(a.ctx); err != nil {
		return fmt.Errorf("daemon not running. Start with: cws daemon start")
	}

	// Get template from API
	runtimeTemplates, err := a.apiClient.ListTemplates(a.ctx)
	if err != nil {
		return fmt.Errorf("failed to list templates: %w", err)
	}

	runtimeTemplate, exists := runtimeTemplates[templateName]
	if !exists {
		return fmt.Errorf("template '%s' not found", templateName)
	}

	// Convert runtime template to unified template for application
	// This is a placeholder - in practice, we'd need the daemon to provide
	// the unified template or convert it here
	template := &templates.Template{
		Name:        runtimeTemplate.Name,
		Description: runtimeTemplate.Description,
		// Note: This is incomplete - we'd need the daemon to provide
		// the full unified template information for application
	}

	// Set template in request
	req.Template = template

	// Apply template via API
	response, err := a.apiClient.ApplyTemplate(a.ctx, req)
	if err != nil {
		return fmt.Errorf("failed to apply template: %w", err)
	}

	if req.DryRun {
		fmt.Printf("🔍 Dry run results for applying '%s' to '%s':\n", templateName, instanceName)
		fmt.Printf("📦 Would install %d packages\n", response.PackagesInstalled)
		fmt.Printf("🔧 Would configure %d services\n", response.ServicesConfigured)
		fmt.Printf("👤 Would create %d users\n", response.UsersCreated)

		if len(response.Warnings) > 0 {
			fmt.Println("\n⚠️  Warnings:")
			for _, warning := range response.Warnings {
				fmt.Printf("   - %s\n", warning)
			}
		}

		fmt.Printf("\n⏱️  Estimated execution time: %v\n", response.ExecutionTime)
		fmt.Printf("\n💡 Run without --dry-run to apply these changes\n")
	} else {
		fmt.Printf("✅ %s\n", response.Message)
		fmt.Printf("📊 Changes applied:\n")
		fmt.Printf("   📦 Packages installed: %d\n", response.PackagesInstalled)
		fmt.Printf("   🔧 Services configured: %d\n", response.ServicesConfigured)
		fmt.Printf("   👤 Users created: %d\n", response.UsersCreated)

		if response.RollbackCheckpoint != "" {
			fmt.Printf("   📸 Rollback checkpoint: %s\n", response.RollbackCheckpoint)
		}

		if len(response.Warnings) > 0 {
			fmt.Println("\n⚠️  Warnings:")
			for _, warning := range response.Warnings {
				fmt.Printf("   - %s\n", warning)
			}
		}

		fmt.Printf("\n⏱️  Execution time: %v\n", response.ExecutionTime)
		fmt.Printf("\n💡 Use 'cws layers %s' to see all applied templates\n", instanceName)
		fmt.Printf("💡 Use 'cws rollback %s' to undo these changes if needed\n", instanceName)
	}

	return nil
}

// Diff handles the diff command
func (a *App) Diff(args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("usage: cws diff <template> <instance-name>")
	}

	templateName := args[0]
	instanceName := args[1]

	// Check daemon is running
	if err := a.apiClient.Ping(a.ctx); err != nil {
		return fmt.Errorf("daemon not running. Start with: cws daemon start")
	}

	// Get template from API
	runtimeTemplates, err := a.apiClient.ListTemplates(a.ctx)
	if err != nil {
		return fmt.Errorf("failed to list templates: %w", err)
	}

	runtimeTemplate, exists := runtimeTemplates[templateName]
	if !exists {
		return fmt.Errorf("template '%s' not found", templateName)
	}

	// Convert runtime template to unified template for diff
	template := &templates.Template{
		Name:        runtimeTemplate.Name,
		Description: runtimeTemplate.Description,
		// Note: This is incomplete - we'd need the daemon to provide
		// the full unified template information for diff calculation
	}

	// Get diff via API
	diff, err := a.apiClient.DiffTemplate(a.ctx, templates.DiffRequest{
		InstanceName: instanceName,
		Template:     template,
	})
	if err != nil {
		return fmt.Errorf("failed to calculate template diff: %w", err)
	}

	fmt.Printf("📋 Template diff for '%s' → '%s':\n\n", templateName, instanceName)

	// Show packages to install
	if len(diff.PackagesToInstall) > 0 {
		fmt.Println("📦 Packages to install:")
		for _, pkg := range diff.PackagesToInstall {
			if pkg.Action == "upgrade" {
				fmt.Printf("   ⬆️  %s (%s → %s) via %s\n", pkg.Name, pkg.CurrentVersion, pkg.TargetVersion, pkg.PackageManager)
			} else {
				fmt.Printf("   ➕ %s", pkg.Name)
				if pkg.TargetVersion != "" {
					fmt.Printf(" (%s)", pkg.TargetVersion)
				}
				fmt.Printf(" via %s\n", pkg.PackageManager)
			}
		}
		fmt.Println()
	}

	// Show services to configure
	if len(diff.ServicesToConfigure) > 0 {
		fmt.Println("🔧 Services to configure:")
		for _, svc := range diff.ServicesToConfigure {
			switch svc.Action {
			case "configure":
				fmt.Printf("   ➕ %s (port %d)\n", svc.Name, svc.Port)
			case "start":
				fmt.Printf("   ▶️  %s (start service)\n", svc.Name)
			case "restart":
				fmt.Printf("   🔄 %s (restart service)\n", svc.Name)
			}
		}
		fmt.Println()
	}

	// Show users to create
	if len(diff.UsersToCreate) > 0 {
		fmt.Println("👤 Users to create:")
		for _, user := range diff.UsersToCreate {
			fmt.Printf("   ➕ %s", user.Name)
			if len(user.TargetGroups) > 0 {
				fmt.Printf(" (groups: %s)", strings.Join(user.TargetGroups, ", "))
			}
			fmt.Println()
		}
		fmt.Println()
	}

	// Show users to modify
	if len(diff.UsersToModify) > 0 {
		fmt.Println("👤 Users to modify:")
		for _, user := range diff.UsersToModify {
			fmt.Printf("   🔄 %s (add to groups: %s)\n", user.Name, strings.Join(user.TargetGroups, ", "))
		}
		fmt.Println()
	}

	// Show ports to open
	if len(diff.PortsToOpen) > 0 {
		fmt.Println("🔌 Ports to open:")
		for _, port := range diff.PortsToOpen {
			fmt.Printf("   ➕ %d\n", port)
		}
		fmt.Println()
	}

	// Show conflicts
	if len(diff.ConflictsFound) > 0 {
		fmt.Println("⚠️  Conflicts detected:")
		for _, conflict := range diff.ConflictsFound {
			fmt.Printf("   ⛔ %s: %s (resolution: %s)\n", conflict.Type, conflict.Description, conflict.Resolution)
		}
		fmt.Println()
		fmt.Println("💡 Use --force to override conflicts")
	}

	// Show summary
	if !diff.HasChanges() {
		fmt.Println("✅ No changes needed - instance already matches template")
	} else {
		fmt.Printf("📊 Summary: %s\n", diff.Summary())
		fmt.Printf("\n💡 Use 'cws apply %s %s' to apply these changes\n", templateName, instanceName)
		fmt.Printf("💡 Use 'cws apply %s %s --dry-run' to preview the application\n", templateName, instanceName)
	}

	return nil
}

// Layers handles the layers command
func (a *App) Layers(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: cws layers <instance-name>")
	}

	instanceName := args[0]

	// Check daemon is running
	if err := a.apiClient.Ping(a.ctx); err != nil {
		return fmt.Errorf("daemon not running. Start with: cws daemon start")
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
		return fmt.Errorf("usage: cws rollback <instance-name> [--to-checkpoint=<checkpoint-id>]")
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

	// Check daemon is running
	if err := a.apiClient.Ping(a.ctx); err != nil {
		return fmt.Errorf("daemon not running. Start with: cws daemon start")
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
