// Package cli - Snapshot Implementation Layer
//
// ARCHITECTURE NOTE: This file contains instance snapshot command business logic.
// These commands are registered in root_command.go and called directly from the CLI.
//
// This follows CloudWorkstation's command architecture pattern:
//   - Single-layer implementation for straightforward operations
//   - Direct integration with root command structure
//   - API-driven operations with consistent error handling
//
// DO NOT REMOVE THIS FILE - it is actively used by root_command.go and App methods.
package cli

import (
	"fmt"
	"os"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/scttfrdmn/cloudworkstation/pkg/types"
)

// SnapshotCommands handles instance snapshot management operations (implementation layer)
type SnapshotCommands struct {
	app *App
}

// NewSnapshotCommands creates a new snapshot commands handler
func NewSnapshotCommands(app *App) *SnapshotCommands {
	return &SnapshotCommands{app: app}
}

// Snapshot handles the snapshot command
func (s *SnapshotCommands) Snapshot(args []string) error {
	if len(args) < 1 {
		return s.showSnapshotUsage()
	}

	action := args[0]
	actionArgs := args[1:]

	switch action {
	case "create":
		return s.createSnapshot(actionArgs)
	case "list":
		return s.listSnapshots(actionArgs)
	case "info", "show":
		return s.getSnapshotInfo(actionArgs)
	case "delete":
		return s.deleteSnapshot(actionArgs)
	case "restore":
		return s.restoreSnapshot(actionArgs)
	default:
		return fmt.Errorf("unknown snapshot action: %s\n\n%s", action, s.getSnapshotUsageText())
	}
}

// createSnapshot creates a snapshot from an instance
func (s *SnapshotCommands) createSnapshot(args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("usage: cws snapshot create <workspace-name> <snapshot-name> [options]")
	}

	instanceName := args[0]
	snapshotName := args[1]

	// Parse options
	req := types.InstanceSnapshotRequest{
		InstanceName: instanceName,
		SnapshotName: snapshotName,
	}

	for i := 2; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "--description" && i+1 < len(args):
			req.Description = args[i+1]
			i++
		case arg == "--no-reboot":
			req.NoReboot = true
		case arg == "--wait":
			req.Wait = true
		case strings.HasPrefix(arg, "--description="):
			req.Description = strings.TrimPrefix(arg, "--description=")
		default:
			return fmt.Errorf("unknown option: %s", arg)
		}
	}

	// Ensure daemon is running
	if err := s.app.ensureDaemonRunning(); err != nil {
		return err
	}

	fmt.Printf("📸 Creating snapshot '%s' from workspace '%s'...\n", snapshotName, instanceName)

	if req.NoReboot {
		fmt.Printf("⚠️  Creating snapshot without reboot (may result in inconsistent state)\n")
	}

	result, err := s.app.apiClient.CreateInstanceSnapshot(s.app.ctx, req)
	if err != nil {
		return WrapAPIError("create snapshot", err)
	}

	fmt.Printf("✅ Snapshot creation initiated\n")
	fmt.Printf("   Snapshot ID: %s\n", result.SnapshotID)
	fmt.Printf("   Source Instance: %s\n", result.SourceInstance)
	if result.Description != "" {
		fmt.Printf("   Description: %s\n", result.Description)
	}
	fmt.Printf("   Estimated Completion: %d minutes\n", result.EstimatedCompletionMinutes)
	fmt.Printf("   Monthly Storage Cost: $%.2f\n", result.StorageCostMonthly)
	fmt.Printf("   Created: %s\n", result.CreatedAt.Format("2006-01-02 15:04:05"))

	if req.Wait {
		fmt.Printf("\n⏳ Monitoring snapshot creation progress...\n")
		return s.monitorSnapshotProgress(result.SnapshotID, result.SnapshotName)
	}

	fmt.Printf("\n💡 Check progress with: cws snapshot info %s\n", snapshotName)

	return nil
}

// listSnapshots lists all snapshots
func (s *SnapshotCommands) listSnapshots(args []string) error {
	// Ensure daemon is running
	if err := s.app.ensureDaemonRunning(); err != nil {
		return err
	}

	response, err := s.app.apiClient.ListInstanceSnapshots(s.app.ctx)
	if err != nil {
		return WrapAPIError("list snapshots", err)
	}

	if len(response.Snapshots) == 0 {
		fmt.Println("No snapshots found.")
		fmt.Println("Create one with: cws snapshot create <workspace-name> <snapshot-name>")
		return nil
	}

	fmt.Printf("📸 Instance Snapshots (%d total):\n\n", response.Count)

	w := tabwriter.NewWriter(os.Stdout, TabWriterMinWidth, TabWriterTabWidth, TabWriterPadding, TabWriterPadChar, TabWriterFlags)
	_, _ = fmt.Fprintln(w, "NAME\tSOURCE INSTANCE\tSTATE\tSIZE\tCOST/MONTH\tCREATED")

	for _, snapshot := range response.Snapshots {
		_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%s\t$%.2f\t%s\n",
			snapshot.SnapshotName,
			snapshot.SourceInstance,
			strings.ToUpper(snapshot.State),
			snapshot.Architecture,
			snapshot.StorageCostMonthly,
			snapshot.CreatedAt.Format("2006-01-02"),
		)
	}

	_ = w.Flush()

	totalCost := 0.0
	for _, snapshot := range response.Snapshots {
		totalCost += snapshot.StorageCostMonthly
	}

	fmt.Printf("\n💰 Total monthly storage cost: $%.2f\n", totalCost)
	fmt.Printf("💡 Use 'cws snapshot info <name>' for detailed information\n")

	return nil
}

// getSnapshotInfo gets detailed information about a snapshot
func (s *SnapshotCommands) getSnapshotInfo(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: cws snapshot info <snapshot-name>")
	}

	snapshotName := args[0]

	// Ensure daemon is running
	if err := s.app.ensureDaemonRunning(); err != nil {
		return err
	}

	snapshot, err := s.app.apiClient.GetInstanceSnapshot(s.app.ctx, snapshotName)
	if err != nil {
		return WrapAPIError("get snapshot info", err)
	}

	fmt.Printf("📸 Snapshot: %s\n", snapshot.SnapshotName)
	fmt.Printf("   Snapshot ID: %s\n", snapshot.SnapshotID)
	fmt.Printf("   Source Instance: %s\n", snapshot.SourceInstance)
	if snapshot.SourceInstanceId != "" {
		fmt.Printf("   Source Instance ID: %s\n", snapshot.SourceInstanceId)
	}
	if snapshot.SourceTemplate != "" {
		fmt.Printf("   Source Template: %s\n", snapshot.SourceTemplate)
	}
	if snapshot.Description != "" {
		fmt.Printf("   Description: %s\n", snapshot.Description)
	}
	fmt.Printf("   State: %s\n", strings.ToUpper(snapshot.State))
	fmt.Printf("   Architecture: %s\n", snapshot.Architecture)
	fmt.Printf("   Monthly Storage Cost: $%.2f\n", snapshot.StorageCostMonthly)
	fmt.Printf("   Created: %s\n", snapshot.CreatedAt.Format("2006-01-02 15:04:05"))

	if len(snapshot.AssociatedSnapshots) > 0 {
		fmt.Printf("   Associated EBS Snapshots: %s\n", strings.Join(snapshot.AssociatedSnapshots, ", "))
	}

	fmt.Printf("\n💡 Operations:\n")
	fmt.Printf("   Restore: cws snapshot restore %s <new-instance-name>\n", snapshot.SnapshotName)
	fmt.Printf("   Delete: cws snapshot delete %s\n", snapshot.SnapshotName)

	return nil
}

// deleteSnapshot deletes a snapshot
func (s *SnapshotCommands) deleteSnapshot(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: cws snapshot delete <snapshot-name>")
	}

	snapshotName := args[0]

	// Confirmation prompt
	fmt.Printf("⚠️  Are you sure you want to delete snapshot '%s'? This action cannot be undone.\n", snapshotName)
	fmt.Printf("Type 'yes' to confirm: ")

	var confirmation string
	_, _ = fmt.Scanln(&confirmation)

	if confirmation != "yes" {
		fmt.Println("❌ Deletion cancelled.")
		return nil
	}

	// Ensure daemon is running
	if err := s.app.ensureDaemonRunning(); err != nil {
		return err
	}

	fmt.Printf("🗑️  Deleting snapshot '%s'...\n", snapshotName)

	result, err := s.app.apiClient.DeleteInstanceSnapshot(s.app.ctx, snapshotName)
	if err != nil {
		return WrapAPIError("delete snapshot", err)
	}

	fmt.Printf("✅ Snapshot deleted successfully\n")
	fmt.Printf("   Snapshot ID: %s\n", result.SnapshotID)
	if len(result.DeletedSnapshots) > 0 {
		fmt.Printf("   Deleted EBS Snapshots: %s\n", strings.Join(result.DeletedSnapshots, ", "))
	}
	fmt.Printf("   Monthly Storage Savings: $%.2f\n", result.StorageSavingsMonthly)
	fmt.Printf("   Deleted At: %s\n", result.DeletedAt.Format("2006-01-02 15:04:05"))

	return nil
}

// restoreSnapshot restores an instance from a snapshot
func (s *SnapshotCommands) restoreSnapshot(args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("usage: cws snapshot restore <snapshot-name> <new-instance-name> [options]")
	}

	snapshotName := args[0]
	newInstanceName := args[1]

	// Parse options
	req := types.InstanceRestoreRequest{
		SnapshotName:    snapshotName,
		NewInstanceName: newInstanceName,
	}

	for i := 2; i < len(args); i++ {
		arg := args[i]
		switch arg {
		case "--wait":
			req.Wait = true
		default:
			return fmt.Errorf("unknown option: %s", arg)
		}
	}

	// Ensure daemon is running
	if err := s.app.ensureDaemonRunning(); err != nil {
		return err
	}

	fmt.Printf("🔄 Restoring instance '%s' from snapshot '%s'...\n", newInstanceName, snapshotName)

	result, err := s.app.apiClient.RestoreInstanceFromSnapshot(s.app.ctx, snapshotName, req)
	if err != nil {
		return WrapAPIError("restore snapshot", err)
	}

	fmt.Printf("✅ Instance restore initiated\n")
	fmt.Printf("   New Instance: %s\n", result.NewInstanceName)
	fmt.Printf("   Instance ID: %s\n", result.InstanceID)
	fmt.Printf("   Source Snapshot: %s\n", result.SnapshotName)
	fmt.Printf("   Source Template: %s\n", result.SourceTemplate)
	fmt.Printf("   Message: %s\n", result.Message)
	fmt.Printf("   Restored At: %s\n", result.RestoredAt.Format("2006-01-02 15:04:05"))

	if req.Wait {
		fmt.Printf("\n⏳ Monitoring instance launch progress...\n")
		return s.app.monitorLaunchProgress(result.NewInstanceName, result.SourceTemplate)
	}

	fmt.Printf("\n💡 Check progress with: cws list\n")
	fmt.Printf("💡 Connect when ready: cws connect %s\n", result.NewInstanceName)

	return nil
}

// monitorSnapshotProgress monitors snapshot creation progress
func (s *SnapshotCommands) monitorSnapshotProgress(snapshotID, snapshotName string) error {
	startTime := time.Now()
	maxDuration := 30 * time.Minute // Maximum monitoring time for snapshots

	fmt.Printf("📊 Monitoring snapshot creation: %s (%s)\n", snapshotName, snapshotID)

	for {
		elapsed := time.Since(startTime)

		// Check for timeout
		if elapsed > maxDuration {
			fmt.Printf("⚠️  Snapshot monitoring timeout (%s). Snapshot may still be creating.\n",
				s.formatDuration(maxDuration))
			fmt.Printf("💡 Check status with: cws snapshot info %s\n", snapshotName)
			return nil
		}

		// Get current snapshot status
		snapshot, err := s.app.apiClient.GetInstanceSnapshot(s.app.ctx, snapshotName)
		if err != nil {
			fmt.Printf("⚠️  Unable to get snapshot status: %v\n", err)
			time.Sleep(10 * time.Second)
			continue
		}

		// Update progress display
		fmt.Printf("⏳ Snapshot state: %s (%s elapsed)\n", strings.ToUpper(snapshot.State), s.formatDuration(elapsed))

		// Check for completion
		if snapshot.State == "available" {
			fmt.Printf("✅ Snapshot creation completed!\n")
			fmt.Printf("   Snapshot: %s\n", snapshot.SnapshotName)
			fmt.Printf("   Total Time: %s\n", s.formatDuration(elapsed))
			fmt.Printf("   Monthly Cost: $%.2f\n", snapshot.StorageCostMonthly)
			return nil
		}

		// Check for error states
		if snapshot.State == "error" || snapshot.State == "failed" {
			fmt.Printf("❌ Snapshot creation failed\n")
			return fmt.Errorf("snapshot creation failed with state: %s", snapshot.State)
		}

		// Wait before next check
		time.Sleep(15 * time.Second)
	}
}

// formatDuration formats a duration for display
func (s *SnapshotCommands) formatDuration(d time.Duration) string {
	minutes := int(d.Minutes())
	seconds := int(d.Seconds()) % 60

	if minutes > 0 {
		return fmt.Sprintf("%dm%ds", minutes, seconds)
	}
	return fmt.Sprintf("%ds", seconds)
}

// showSnapshotUsage shows snapshot command usage
func (s *SnapshotCommands) showSnapshotUsage() error {
	fmt.Print(s.getSnapshotUsageText())
	return nil
}

// getSnapshotUsageText returns the snapshot usage text
func (s *SnapshotCommands) getSnapshotUsageText() string {
	return `Usage: cws snapshot <action> [arguments]

Actions:
  create <workspace-name> <snapshot-name> [options]   Create a snapshot from an instance
  list                                              List all snapshots
  info <snapshot-name>                              Show detailed snapshot information
  delete <snapshot-name>                            Delete a snapshot
  restore <snapshot-name> <new-instance-name>      Create new instance from snapshot

Create Options:
  --description <text>    Add a description to the snapshot
  --no-reboot            Create snapshot without rebooting instance (may be inconsistent)
  --wait                 Wait and monitor snapshot creation progress

Restore Options:
  --wait                 Wait and monitor instance launch progress

Examples:
  cws snapshot create my-workspace backup-v1
  cws snapshot create gpu-training checkpoint-epoch-10 --description "Training checkpoint after 10 epochs"
  cws snapshot list
  cws snapshot info backup-v1
  cws snapshot restore backup-v1 my-new-workstation
  cws snapshot delete old-backup

Cost Information:
  Snapshots are stored as AMIs with associated EBS snapshots
  Cost: ~$0.05/GB/month for EBS snapshot storage
  Use 'cws snapshot list' to see total monthly costs

💡 Snapshots preserve the complete instance state including:
   • Operating system and configuration
   • Installed packages and applications
   • User data and files
   • Template configuration metadata

⚠️  Important Notes:
   • Snapshots are region-specific
   • Creating snapshots may cause brief I/O pause (use --no-reboot to avoid)
   • Restored instances launch with the same template configuration
`
}
