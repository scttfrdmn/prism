package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/scttfrdmn/cloudworkstation/pkg/types"
	"github.com/spf13/cobra"
)

// LogsCommands handles log-related operations
type LogsCommands struct {
	app *App
}

// NewLogsCommands creates new logs commands
func NewLogsCommands(app *App) *LogsCommands {
	return &LogsCommands{
		app: app,
	}
}

// printJSON outputs data as formatted JSON
func (lc *LogsCommands) printJSON(data interface{}) error {
	return json.NewEncoder(os.Stdout).Encode(data)
}

// CreateLogsCommand creates the main logs command with subcommands
func (lc *LogsCommands) CreateLogsCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "logs [instance-name]",
		Short: "View and manage instance logs",
		Long: `View logs from CloudWorkstation instances including console output,
system logs, and application logs.

Examples:
  cws logs my-instance                    # Show console logs
  cws logs my-instance --type cloud-init # Show cloud-init logs
  cws logs my-instance --tail 50         # Show last 50 lines
  cws logs my-instance --since 1h        # Show logs from last hour
  cws logs my-instance --follow          # Follow logs in real-time
  cws logs --list                        # List all instances with log availability`,
		Args: cobra.MaximumNArgs(1),
		RunE: lc.handleLogsCommand,
	}

	// Add flags
	cmd.Flags().StringP("type", "t", "console", "Log type (console, cloud-init, cloud-init-out, messages, secure, boot, dmesg, kern, syslog)")
	cmd.Flags().IntP("tail", "n", 0, "Number of lines to show from the end of the logs")
	cmd.Flags().String("since", "", "Show logs since duration (e.g., 1h, 30m, 2h30m)")
	cmd.Flags().BoolP("follow", "f", false, "Follow log output in real-time")
	cmd.Flags().Bool("list", false, "List all instances with log availability")
	cmd.Flags().Bool("types", false, "Show available log types for instance")
	cmd.Flags().Bool("json", false, "Output in JSON format")

	return cmd
}

// handleLogsCommand handles the main logs command
func (lc *LogsCommands) handleLogsCommand(cmd *cobra.Command, args []string) error {
	list, _ := cmd.Flags().GetBool("list")
	showTypes, _ := cmd.Flags().GetBool("types")
	jsonOutput, _ := cmd.Flags().GetBool("json")

	// Handle list all instances
	if list {
		return lc.handleListLogs(jsonOutput)
	}

	// Require instance name for other operations
	if len(args) == 0 {
		return fmt.Errorf("instance name required (use --list to see all instances)")
	}

	instanceName := args[0]

	// Handle show available log types
	if showTypes {
		return lc.handleShowLogTypes(instanceName, jsonOutput)
	}

	// Handle show logs
	return lc.handleShowLogs(cmd, instanceName, jsonOutput)
}

// handleListLogs lists all instances with log availability
func (lc *LogsCommands) handleListLogs(jsonOutput bool) error {
	ctx := context.Background()
	summary, err := lc.app.apiClient.GetLogsSummary(ctx)
	if err != nil {
		return fmt.Errorf("failed to get logs summary: %w", err)
	}

	if jsonOutput {
		return lc.printJSON(summary)
	}

	fmt.Println("📋 Instance Log Availability")
	fmt.Println("===========================")

	if len(summary.Instances) == 0 {
		fmt.Println("No instances found")
		return nil
	}

	fmt.Printf("%-20s %-15s %-10s %s\n", "NAME", "INSTANCE ID", "STATE", "LOGS AVAILABLE")
	fmt.Printf("%-20s %-15s %-10s %s\n", "----", "-----------", "-----", "--------------")

	for _, instance := range summary.Instances {
		logsStatus := "❌ No"
		if instance.LogsAvailable {
			logsStatus = "✅ Yes"
		}

		fmt.Printf("%-20s %-15s %-10s %s\n",
			instance.Name,
			instance.ID[:12]+"...", // Truncate instance ID
			instance.State,
			logsStatus)
	}

	fmt.Printf("\nAvailable log types: %s\n", strings.Join(summary.AvailableLogTypes, ", "))
	fmt.Println("\nUse 'cws logs <instance-name>' to view logs for a specific instance")

	return nil
}

// handleShowLogTypes shows available log types for an instance
func (lc *LogsCommands) handleShowLogTypes(instanceName string, jsonOutput bool) error {
	ctx := context.Background()
	logTypes, err := lc.app.apiClient.GetInstanceLogTypes(ctx, instanceName)
	if err != nil {
		return fmt.Errorf("failed to get log types for instance %s: %w", instanceName, err)
	}

	if jsonOutput {
		return lc.printJSON(logTypes)
	}

	fmt.Printf("📋 Available Log Types for Instance: %s\n", instanceName)
	fmt.Println("===============================================")
	fmt.Printf("Instance ID: %s\n", logTypes.InstanceID)
	fmt.Printf("SSM Enabled: %v\n\n", logTypes.SSMEnabled)

	fmt.Println("Available log types:")
	for _, logType := range logTypes.AvailableLogTypes {
		status := "✅"
		description := lc.getLogTypeDescription(logType)
		if logType != "console" && !logTypes.SSMEnabled {
			status = "⚠️ (requires running instance)"
		}
		fmt.Printf("  %s %s - %s\n", status, logType, description)
	}

	if !logTypes.SSMEnabled {
		fmt.Println("\n⚠️  Some log types require the instance to be running for SSM access")
	}

	fmt.Printf("\nUsage: cws logs %s --type <log-type>\n", instanceName)

	return nil
}

// handleShowLogs shows logs for an instance
func (lc *LogsCommands) handleShowLogs(cmd *cobra.Command, instanceName string, jsonOutput bool) error {
	logType, _ := cmd.Flags().GetString("type")
	tail, _ := cmd.Flags().GetInt("tail")
	since, _ := cmd.Flags().GetString("since")
	follow, _ := cmd.Flags().GetBool("follow")

	// Validate since format if provided
	if since != "" {
		if _, err := time.ParseDuration(since); err != nil {
			return fmt.Errorf("invalid since duration '%s': %w (use format like 1h, 30m, 2h30m)", since, err)
		}
	}

	// Build log request
	logRequest := types.LogRequest{
		LogType: logType,
		Tail:    tail,
		Since:   since,
		Follow:  follow,
	}

	ctx := context.Background()
	logs, err := lc.app.apiClient.GetInstanceLogs(ctx, instanceName, logRequest)
	if err != nil {
		return fmt.Errorf("failed to get logs for instance %s: %w", instanceName, err)
	}

	if jsonOutput {
		return lc.printJSON(logs)
	}

	// Display logs
	lc.displayLogs(logs, follow)

	return nil
}

// displayLogs formats and displays log output
func (lc *LogsCommands) displayLogs(logs *types.LogResponse, follow bool) {
	// Header
	fmt.Printf("📋 Logs for Instance: %s (%s)\n", logs.InstanceName, logs.InstanceID)
	fmt.Println("=====================================")
	fmt.Printf("Log Type: %s\n", logs.LogType)
	fmt.Printf("Timestamp: %s\n", logs.Timestamp.Format("2006-01-02 15:04:05 MST"))
	if logs.Tail > 0 {
		fmt.Printf("Showing last %d lines\n", logs.Tail)
	}
	if follow {
		fmt.Println("Following logs (Ctrl+C to stop)")
	}
	fmt.Println()

	// Display log lines
	if len(logs.Lines) == 0 {
		fmt.Println("No logs available")
		return
	}

	for _, line := range logs.Lines {
		fmt.Println(line)
	}

	// Handle real-time following (simplified implementation)
	if follow {
		fmt.Println("\n⚠️  Real-time log following not fully implemented yet")
		fmt.Println("   Use '--tail' to get recent logs and re-run the command to refresh")
	}
}

// getLogTypeDescription returns a description for each log type
func (lc *LogsCommands) getLogTypeDescription(logType string) string {
	descriptions := map[string]string{
		"console":        "EC2 console output (boot messages, kernel logs)",
		"cloud-init":     "Cloud-init service logs",
		"cloud-init-out": "Cloud-init command output",
		"messages":       "System messages (/var/log/messages)",
		"secure":         "Security and authentication logs",
		"boot":           "Boot process logs",
		"dmesg":          "Kernel ring buffer messages",
		"kern":           "Kernel logs",
		"syslog":         "System logs (Ubuntu/Debian)",
	}

	if desc, exists := descriptions[logType]; exists {
		return desc
	}
	return "System log file"
}
