// Package cli provides budget management commands for Prism.
//
// The budget system provides comprehensive financial controls for research computing,
// enabling researchers and institutions to track spending, set limits, and automate
// cost-saving actions across individual projects and organizational accounts.
//
// Budget Architecture:
//   - Personal budgets: Individual researcher spending limits
//   - Project budgets: Collaborative project financial controls
//   - Organization budgets: Institution-wide spending management
//   - Real-time tracking: Live cost monitoring with AWS billing integration
//   - Automated actions: Hibernation, stopping, and launch prevention controls
//   - Alert system: Email, Slack, webhook notifications for budget thresholds
//
// Design Philosophy:
// Follows Prism's "Progressive Disclosure" principle - simple budget
// creation with optional advanced features like automated actions and custom alerts.
//
// Usage Examples:
//
//	prism budget list                    # Show all budgets and current status
//	prism budget create my-project 1000  # Create $1000 project budget
//	prism budget status my-project       # Show detailed budget status
//	prism budget breakdown my-project    # Cost breakdown by service/instance
package cli

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/scttfrdmn/prism/pkg/api/client"
	"github.com/scttfrdmn/prism/pkg/project"
	"github.com/scttfrdmn/prism/pkg/types"
	"github.com/spf13/cobra"
)

// BudgetCommands handles all budget-related CLI operations
type BudgetCommands struct {
	app *App
}

// NewBudgetCommands creates a new budget commands handler
func NewBudgetCommands(app *App) *BudgetCommands {
	return &BudgetCommands{app: app}
}

// CreateBudgetCommand creates the root budget command with all subcommands
func (bc *BudgetCommands) CreateBudgetCommand() *cobra.Command {
	budgetCmd := &cobra.Command{
		Use:   "budget",
		Short: "Comprehensive budget management for research computing costs",
		Long: `Manage budgets, track spending, and control costs for research computing.

Prism's budget system provides enterprise-grade financial controls
with real-time cost tracking, automated actions, and detailed analytics.

Budget Types:
  • Personal budgets: Individual researcher spending limits
  • Project budgets: Collaborative project financial controls
  • Organizational: Institution-wide spending management

Features:
  • Real-time cost tracking with AWS billing integration
  • Automated cost-saving actions (hibernation, stopping, launch prevention)
  • Multi-channel alerts (email, Slack, webhook notifications)
  • Detailed cost breakdowns and forecasting
  • Hibernation savings analytics`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return bc.budgetHelp()
		},
	}

	// Add all budget subcommands
	budgetCmd.AddCommand(bc.createListCommand())
	budgetCmd.AddCommand(bc.createCreateCommand())
	budgetCmd.AddCommand(bc.createUpdateCommand())
	budgetCmd.AddCommand(bc.createDeleteCommand())
	budgetCmd.AddCommand(bc.createInfoCommand())
	budgetCmd.AddCommand(bc.createStatusCommand())
	budgetCmd.AddCommand(bc.createUsageCommand())
	budgetCmd.AddCommand(bc.createHistoryCommand())
	budgetCmd.AddCommand(bc.createAlertsCommand())
	budgetCmd.AddCommand(bc.createForecastCommand())
	budgetCmd.AddCommand(bc.createSavingsCommand())
	budgetCmd.AddCommand(bc.createBreakdownCommand())
	budgetCmd.AddCommand(bc.createCushionCommand())
	budgetCmd.AddCommand(bc.createGDEWCommand())
	budgetCmd.AddCommand(bc.createDiscountsCommand())
	budgetCmd.AddCommand(bc.createCreditsCommand())
	// v0.12.0 governance subcommands
	budgetCmd.AddCommand(bc.createRolloverCommand())
	budgetCmd.AddCommand(bc.createShareCommand())
	budgetCmd.AddCommand(bc.createReallocateCommand())
	budgetCmd.AddCommand(bc.createBorrowCommand())
	budgetCmd.AddCommand(bc.createEmergencyCommand())
	budgetCmd.AddCommand(bc.createReportCommand())

	return budgetCmd
}

// budgetHelp displays help information and current budget overview
func (bc *BudgetCommands) budgetHelp() error {
	fmt.Printf("💰 Prism Budget Management\n\n")

	fmt.Printf("🏗️ Budget Management:\n")
	fmt.Printf("   prism budget list                    List all budgets and status\n")
	fmt.Printf("   prism budget create <project> <amt>  Create new budget\n")
	fmt.Printf("   prism budget update <budget-id>      Update budget settings\n")
	fmt.Printf("   prism budget delete <budget-id>      Delete budget\n")
	fmt.Printf("   prism budget info <budget-id>        Show detailed budget info\n")
	fmt.Printf("\n")

	fmt.Printf("📊 Budget Monitoring:\n")
	fmt.Printf("   prism budget status [budget-id]      Show current spending status\n")
	fmt.Printf("   prism budget usage <budget-id>       Show detailed usage metrics\n")
	fmt.Printf("   prism budget history <budget-id>     Show spending history\n")
	fmt.Printf("   prism budget alerts <budget-id>      Manage budget alerts\n")
	fmt.Printf("\n")

	fmt.Printf("🔍 Budget Analysis:\n")
	fmt.Printf("   prism budget forecast <budget-id>    Show spending forecast\n")
	fmt.Printf("   prism budget savings [budget-id]     Show hibernation savings\n")
	fmt.Printf("   prism budget breakdown <budget-id>   Cost breakdown by service\n")
	fmt.Printf("\n")

	// Show quick budget overview if daemon is running
	if err := bc.app.apiClient.Ping(bc.app.ctx); err == nil {
		fmt.Printf("📋 Quick Budget Overview:\n")
		if err := bc.showQuickOverview(); err != nil {
			fmt.Printf("   (Error loading budget overview: %v)\n", err)
		}
		fmt.Printf("\n")
	}

	fmt.Printf("💡 Examples:\n")
	fmt.Printf("   prism budget create my-research 500    # Create $500 project budget\n")
	fmt.Printf("   prism budget status my-research        # Check spending status\n")
	fmt.Printf("   prism budget breakdown my-research     # Detailed cost analysis\n")

	return nil
}

// showQuickOverview displays a quick overview of budget status
func (bc *BudgetCommands) showQuickOverview() error {
	projects, err := bc.app.apiClient.ListProjects(bc.app.ctx, nil)
	if err != nil {
		return err
	}

	if len(projects.Projects) == 0 {
		fmt.Printf("   No budgets found. Create one with: prism budget create <project> <amount>\n")
		return nil
	}

	totalBudget := 0.0
	totalSpent := 0.0
	budgetCount := 0

	for _, proj := range projects.Projects {
		if proj.BudgetStatus != nil && proj.BudgetStatus.TotalBudget > 0 {
			budgetCount++
			totalBudget += proj.BudgetStatus.TotalBudget
			totalSpent += proj.BudgetStatus.SpentAmount
		}
	}

	if budgetCount == 0 {
		fmt.Printf("   No active budgets. Enable budget tracking with: prism budget create\n")
		return nil
	}

	spentPercent := 0.0
	if totalBudget > 0 {
		spentPercent = (totalSpent / totalBudget) * 100
	}

	fmt.Printf("   Active Budgets: %d\n", budgetCount)
	fmt.Printf("   Total Budget: $%.2f\n", totalBudget)
	fmt.Printf("   Total Spent: $%.2f (%.1f%%)\n", totalSpent, spentPercent)
	fmt.Printf("   Remaining: $%.2f\n", totalBudget-totalSpent)

	return nil
}

// createListCommand creates the budget list command
func (bc *BudgetCommands) createListCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all budgets and their current status",
		Long: `List all budgets across personal and project accounts with current spending status.

Shows budget limits, spent amounts, remaining budget, and alert status in a
comprehensive table format. Includes cost forecasting and savings analytics.

The list command provides a complete financial overview of your research computing
infrastructure with real-time cost tracking and budget utilization metrics.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return bc.listBudgets()
		},
	}
}

// createCreateCommand creates the budget create command
func (bc *BudgetCommands) createCreateCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create <project> <amount>",
		Short: "Create a new budget with optional advanced settings",
		Long: `Create a new budget for a project with comprehensive financial controls.

Supports both simple budget creation and advanced configuration including:
  • Custom alert thresholds and notification channels
  • Automated cost-saving actions (hibernation, stopping)
  • Monthly and daily spending limits
  • Budget period configuration (project lifetime, monthly, weekly)

The budget system integrates with AWS billing for real-time cost tracking
and provides automated actions to prevent cost overruns.`,
		Args: cobra.MinimumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			return bc.createBudget(cmd, args)
		},
	}

	// Add advanced budget creation flags
	cmd.Flags().Float64("monthly-limit", 0, "Monthly spending limit")
	cmd.Flags().Float64("daily-limit", 0, "Daily spending limit")
	cmd.Flags().String("period", "project", "Budget period: project, monthly, weekly, daily")
	cmd.Flags().StringSlice("alert", []string{}, "Alert threshold in format 'percent:type:recipients' (e.g., '80:email:admin@org.edu')")
	cmd.Flags().StringSlice("action", []string{}, "Auto action in format 'percent:action' (e.g., '90:hibernate_all')")
	cmd.Flags().String("end-date", "", "Budget end date (YYYY-MM-DD)")
	cmd.Flags().String("description", "", "Budget description")

	return cmd
}

// createUpdateCommand creates the budget update command
func (bc *BudgetCommands) createUpdateCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update <budget-id>",
		Short: "Update an existing budget's settings and thresholds",
		Long: `Update budget limits, alerts, automated actions, and other settings.

Allows modification of all budget parameters without resetting spending history:
  • Total budget amount and spending limits
  • Alert thresholds and notification settings
  • Automated actions and triggers
  • Budget period and end date

Updates preserve existing cost history and maintain continuity of budget tracking
while allowing fine-tuning of financial controls as project needs evolve.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return bc.updateBudget(cmd, args)
		},
	}

	// Add update flags (same as create but optional)
	cmd.Flags().Float64("total-budget", 0, "Update total budget amount")
	cmd.Flags().Float64("monthly-limit", 0, "Update monthly spending limit")
	cmd.Flags().Float64("daily-limit", 0, "Update daily spending limit")
	cmd.Flags().StringSlice("alert", []string{}, "Replace alert thresholds")
	cmd.Flags().StringSlice("action", []string{}, "Replace auto actions")
	cmd.Flags().String("end-date", "", "Update budget end date (YYYY-MM-DD)")
	cmd.Flags().String("description", "", "Update budget description")

	return cmd
}

// createDeleteCommand creates the budget delete command
func (bc *BudgetCommands) createDeleteCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "delete <budget-id>",
		Short: "Delete a budget and disable cost tracking",
		Long: `Permanently delete a budget and disable cost tracking for the associated project.

This action:
  • Removes all budget limits and automated actions
  • Disables cost tracking and alerting
  • Preserves historical spending data for audit purposes
  • Does NOT affect running instances (they continue normally)

Deletion requires confirmation and cannot be undone. Consider updating budget
limits instead of deletion if you need to modify budget parameters.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return bc.deleteBudget(args)
		},
	}
}

// createInfoCommand creates the budget info command
func (bc *BudgetCommands) createInfoCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "info <budget-id>",
		Short: "Show detailed budget information and configuration",
		Long: `Display comprehensive budget information including configuration, spending
history, alert settings, automated actions, and cost forecasting.

Provides a complete financial overview of the budget including:
  • Current spending status and remaining budget
  • Alert thresholds and notification configuration
  • Automated action settings and trigger history
  • Cost breakdown by instance and service type
  • Hibernation savings and cost optimization metrics
  • Spending trends and forecasting analysis`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return bc.infoBudget(args)
		},
	}
}

// createStatusCommand creates the budget status command
func (bc *BudgetCommands) createStatusCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "status [budget-id]",
		Short: "Show current budget status and spending summary",
		Long: `Display current budget status with real-time spending information.

If no budget-id is provided, shows status for all active budgets.
Individual budget status includes:
  • Current spending vs. budget limits
  • Spending rate and projected monthly costs
  • Days until budget exhaustion
  • Active alerts and triggered actions
  • Cost optimization recommendations`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return bc.statusBudget(args)
		},
	}
}

// createUsageCommand creates the budget usage command
func (bc *BudgetCommands) createUsageCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "usage <budget-id>",
		Short: "Show detailed resource usage metrics and cost analysis",
		Long: `Display comprehensive resource utilization metrics for budget analysis.

Provides detailed breakdown of:
  • Compute hours by instance type and size
  • Storage usage across EFS and EBS volumes
  • Cost per service and resource type
  • Idle time analysis and hibernation opportunities
  • Resource efficiency recommendations
  • Historical usage trends and patterns`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return bc.usageBudget(cmd, args)
		},
	}

	cmd.Flags().String("period", "30d", "Analysis period: 7d, 30d, 90d")
	return cmd
}

// createHistoryCommand creates the budget history command
func (bc *BudgetCommands) createHistoryCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "history <budget-id>",
		Short: "Show spending history and trends over time",
		Long: `Display historical spending data with trend analysis and forecasting.

Shows chronological spending data including:
  • Daily/weekly/monthly spending patterns
  • Cost trends and growth rates
  • Seasonal patterns and anomalies
  • Budget utilization over time
  • Savings from hibernation and cost optimization
  • Comparative analysis across time periods`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return bc.historyBudget(cmd, args)
		},
	}

	cmd.Flags().String("period", "30d", "History period: 7d, 30d, 90d")
	cmd.Flags().String("format", "table", "Output format: table, json, csv")
	return cmd
}

// createAlertsCommand creates the budget alerts command
func (bc *BudgetCommands) createAlertsCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "alerts <budget-id>",
		Short: "Manage budget alerts and notification settings",
		Long: `Configure and manage budget alert thresholds and notifications.

Alert management includes:
  • Add/remove/modify alert thresholds
  • Configure notification channels (email, Slack, webhook)
  • Test notification delivery
  • View alert history and triggered events
  • Enable/disable individual alerts
  • Bulk alert configuration management`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return bc.alertsBudget(cmd, args)
		},
	}

	cmd.Flags().String("action", "list", "Action: list, add, remove, test")
	cmd.Flags().Float64("threshold", 0, "Alert threshold percentage (0-100)")
	cmd.Flags().String("type", "", "Alert type: email, slack, webhook")
	cmd.Flags().StringSlice("recipients", []string{}, "Alert recipients")
	cmd.Flags().String("message", "", "Custom alert message")

	return cmd
}

// createForecastCommand creates the budget forecast command
func (bc *BudgetCommands) createForecastCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "forecast <budget-id>",
		Short: "Show spending forecast and budget projections",
		Long: `Generate spending forecasts based on historical usage patterns.

Forecasting includes:
  • Projected monthly and annual spending
  • Budget exhaustion timeline
  • Seasonal trend analysis
  • Resource scaling impact projections
  • Cost optimization opportunity identification
  • Scenario analysis for different usage patterns`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return bc.forecastBudget(cmd, args)
		},
	}

	cmd.Flags().String("horizon", "3m", "Forecast horizon: 1m, 3m, 6m, 1y")
	cmd.Flags().String("scenario", "current", "Scenario: current, optimistic, conservative")
	return cmd
}

// createSavingsCommand creates the budget savings command
func (bc *BudgetCommands) createSavingsCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "savings [budget-id]",
		Short: "Show hibernation and cost optimization savings",
		Long: `Display cost savings from hibernation, right-sizing, and optimization.

Savings analysis includes:
  • Hibernation savings by instance and time period
  • Spot instance cost reductions
  • Right-sizing recommendations and potential savings
  • Idle resource identification
  • Total cost avoidance metrics
  • ROI from cost optimization features`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return bc.savingsBudget(cmd, args)
		},
	}

	cmd.Flags().String("period", "30d", "Analysis period: 7d, 30d, 90d")
	cmd.Flags().Bool("recommendations", false, "Include optimization recommendations")
	return cmd
}

// createBreakdownCommand creates the budget breakdown command
func (bc *BudgetCommands) createBreakdownCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "breakdown <budget-id>",
		Short: "Show detailed cost breakdown by service and instance",
		Long: `Display comprehensive cost breakdown across all services and resources.

Cost breakdown includes:
  • Per-instance compute and storage costs
  • Service-level cost attribution (EC2, EBS, EFS, etc.)
  • Cost per research user and project member
  • Time-based cost analysis (hourly, daily, monthly)
  • Regional cost distribution
  • Cost efficiency metrics and recommendations`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return bc.breakdownBudget(cmd, args)
		},
	}

	cmd.Flags().String("period", "30d", "Analysis period: 7d, 30d, 90d")
	cmd.Flags().String("group-by", "instance", "Group by: instance, service, user, region")
	cmd.Flags().String("format", "table", "Output format: table, json, csv")
	return cmd
}

// Implementation methods

// listBudgets displays all budgets with their current status
func (bc *BudgetCommands) listBudgets() error {
	if err := bc.app.ensureDaemonRunning(); err != nil {
		return err
	}

	projects, err := bc.app.apiClient.ListProjects(bc.app.ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to list projects: %w", err)
	}

	if len(projects.Projects) == 0 {
		fmt.Printf("No projects found.\n")
		fmt.Printf("💡 Create a project with budget: prism budget create <project> <amount>\n")
		return nil
	}

	fmt.Printf("💰 Budget Overview\n\n")

	w := tabwriter.NewWriter(os.Stdout, 0, 8, 2, ' ', 0)
	fmt.Fprintf(w, "PROJECT\tBUDGET\tSPENT\tREMAINING\t%%USED\tPROJECTED/MONTH\tSTATUS\tALERTS\n")

	totalBudget := 0.0
	totalSpent := 0.0
	budgetCount := 0

	for _, proj := range projects.Projects {
		if proj.BudgetStatus == nil || proj.BudgetStatus.TotalBudget <= 0 {
			fmt.Fprintf(w, "%s\t-\t$%.2f\t-\t-\t-\tNo Budget\t-\n",
				proj.Name, proj.TotalCost)
			continue
		}

		budget := proj.BudgetStatus
		remaining := budget.TotalBudget - budget.SpentAmount
		if remaining < 0 {
			remaining = 0
		}

		usedPercent := (budget.SpentAmount / budget.TotalBudget) * 100

		// Status indicators
		status := "OK"
		if usedPercent >= 90 {
			status = "CRITICAL"
		} else if usedPercent >= 75 {
			status = "WARNING"
		}

		// Get budget details for projected spending
		budgetDetails, err := bc.app.apiClient.GetProjectBudgetStatus(bc.app.ctx, proj.ID)
		projectedMonthly := "-"
		alertStatus := "-"

		if err == nil {
			if budgetDetails.ProjectedMonthlySpend > 0 {
				projectedMonthly = fmt.Sprintf("$%.2f", budgetDetails.ProjectedMonthlySpend)
			}
			if len(budgetDetails.ActiveAlerts) > 0 {
				alertStatus = fmt.Sprintf("%d active", len(budgetDetails.ActiveAlerts))
			} else {
				alertStatus = "None"
			}
		}

		fmt.Fprintf(w, "%s\t$%.2f\t$%.2f\t$%.2f\t%.1f%%\t%s\t%s\t%s\n",
			proj.Name,
			budget.TotalBudget,
			budget.SpentAmount,
			remaining,
			usedPercent,
			projectedMonthly,
			status,
			alertStatus)

		totalBudget += budget.TotalBudget
		totalSpent += budget.SpentAmount
		budgetCount++
	}

	if budgetCount > 0 {
		overallPercent := (totalSpent / totalBudget) * 100
		fmt.Fprintf(w, "\nTOTAL (%d budgets)\t$%.2f\t$%.2f\t$%.2f\t%.1f%%\t-\t-\t-\n",
			budgetCount, totalBudget, totalSpent, totalBudget-totalSpent, overallPercent)
	}

	if err := w.Flush(); err != nil {
		return err
	}

	fmt.Printf("\n💡 Commands:\n")
	fmt.Printf("   prism budget create <project> <amount>  # Create new budget\n")
	fmt.Printf("   prism budget status <project>          # Detailed status\n")
	fmt.Printf("   prism budget breakdown <project>       # Cost breakdown\n")

	return nil
}

// createBudget creates a new budget with the specified parameters
func (bc *BudgetCommands) createBudget(cmd *cobra.Command, args []string) error {
	if err := bc.app.ensureDaemonRunning(); err != nil {
		return err
	}

	projectName, amount, err := bc.parseCreateBudgetArgs(args)
	if err != nil {
		return err
	}

	req, err := bc.buildCreateBudgetRequest(cmd, amount)
	if err != nil {
		return err
	}

	response, err := bc.app.apiClient.SetProjectBudget(bc.app.ctx, projectName, req)
	if err != nil {
		return fmt.Errorf("failed to create budget: %w", err)
	}

	bc.displayCreateBudgetSuccess(projectName, amount, req, response)
	return nil
}

// parseCreateBudgetArgs parses and validates budget creation arguments
func (bc *BudgetCommands) parseCreateBudgetArgs(args []string) (string, float64, error) {
	projectName := args[0]
	amount, err := strconv.ParseFloat(args[1], 64)
	if err != nil {
		return "", 0, fmt.Errorf("invalid budget amount '%s': must be a number", args[1])
	}

	if amount <= 0 {
		return "", 0, fmt.Errorf("budget amount must be greater than 0")
	}

	return projectName, amount, nil
}

// buildCreateBudgetRequest builds the budget creation request from command flags
func (bc *BudgetCommands) buildCreateBudgetRequest(cmd *cobra.Command, amount float64) (client.SetProjectBudgetRequest, error) {
	req := client.SetProjectBudgetRequest{
		TotalBudget:     amount,
		BudgetPeriod:    types.BudgetPeriodProject,
		AlertThresholds: []types.BudgetAlert{},
		AutoActions:     []types.BudgetAutoAction{},
	}

	if err := bc.parseCreateBudgetPeriod(cmd, &req); err != nil {
		return req, err
	}

	bc.parseCreateBudgetLimits(cmd, &req)

	if err := bc.parseCreateBudgetEndDate(cmd, &req); err != nil {
		return req, err
	}

	if err := bc.parseCreateBudgetAlerts(cmd, &req); err != nil {
		return req, err
	}

	if err := bc.parseCreateBudgetActions(cmd, &req); err != nil {
		return req, err
	}

	return req, nil
}

// parseCreateBudgetPeriod parses the budget period flag
func (bc *BudgetCommands) parseCreateBudgetPeriod(cmd *cobra.Command, req *client.SetProjectBudgetRequest) error {
	if period, _ := cmd.Flags().GetString("period"); period != "" {
		switch period {
		case "project":
			req.BudgetPeriod = types.BudgetPeriodProject
		case "monthly":
			req.BudgetPeriod = types.BudgetPeriodMonthly
		case "weekly":
			req.BudgetPeriod = types.BudgetPeriodWeekly
		case "daily":
			req.BudgetPeriod = types.BudgetPeriodDaily
		default:
			return fmt.Errorf("invalid period '%s': must be project, monthly, weekly, or daily", period)
		}
	}
	return nil
}

// parseCreateBudgetLimits parses monthly and daily limit flags
func (bc *BudgetCommands) parseCreateBudgetLimits(cmd *cobra.Command, req *client.SetProjectBudgetRequest) {
	if monthlyLimit, _ := cmd.Flags().GetFloat64("monthly-limit"); monthlyLimit > 0 {
		req.MonthlyLimit = &monthlyLimit
	}
	if dailyLimit, _ := cmd.Flags().GetFloat64("daily-limit"); dailyLimit > 0 {
		req.DailyLimit = &dailyLimit
	}
}

// parseCreateBudgetEndDate parses the end date flag
func (bc *BudgetCommands) parseCreateBudgetEndDate(cmd *cobra.Command, req *client.SetProjectBudgetRequest) error {
	if endDateStr, _ := cmd.Flags().GetString("end-date"); endDateStr != "" {
		endDate, err := time.Parse("2006-01-02", endDateStr)
		if err != nil {
			return fmt.Errorf("invalid end date format: use YYYY-MM-DD")
		}
		req.EndDate = &endDate
	}
	return nil
}

// parseCreateBudgetAlerts parses alert threshold flags
func (bc *BudgetCommands) parseCreateBudgetAlerts(cmd *cobra.Command, req *client.SetProjectBudgetRequest) error {
	if alerts, _ := cmd.Flags().GetStringSlice("alert"); len(alerts) > 0 {
		for _, alertStr := range alerts {
			alert, err := bc.parseAlertFlag(alertStr)
			if err != nil {
				return fmt.Errorf("invalid alert format '%s': %v", alertStr, err)
			}
			req.AlertThresholds = append(req.AlertThresholds, alert)
		}
	} else {
		// Add default 80% warning alert
		req.AlertThresholds = append(req.AlertThresholds, types.BudgetAlert{
			Threshold: 0.8,
			Type:      types.BudgetAlertEmail,
			Enabled:   true,
		})
	}
	return nil
}

// parseCreateBudgetActions parses automated action flags
func (bc *BudgetCommands) parseCreateBudgetActions(cmd *cobra.Command, req *client.SetProjectBudgetRequest) error {
	if actions, _ := cmd.Flags().GetStringSlice("action"); len(actions) > 0 {
		for _, actionStr := range actions {
			action, err := bc.parseActionFlag(actionStr)
			if err != nil {
				return fmt.Errorf("invalid action format '%s': %v", actionStr, err)
			}
			req.AutoActions = append(req.AutoActions, action)
		}
	}
	return nil
}

// displayCreateBudgetSuccess displays successful budget creation message
func (bc *BudgetCommands) displayCreateBudgetSuccess(projectName string, amount float64, req client.SetProjectBudgetRequest, response map[string]interface{}) {
	fmt.Printf("✅ Budget created successfully for project '%s'\n", projectName)
	fmt.Printf("   Total Budget: $%.2f\n", amount)
	fmt.Printf("   Budget Period: %s\n", req.BudgetPeriod)

	if req.MonthlyLimit != nil {
		fmt.Printf("   Monthly Limit: $%.2f\n", *req.MonthlyLimit)
	}
	if req.DailyLimit != nil {
		fmt.Printf("   Daily Limit: $%.2f\n", *req.DailyLimit)
	}

	fmt.Printf("   Alert Thresholds: %d configured\n", len(req.AlertThresholds))
	fmt.Printf("   Auto Actions: %d configured\n", len(req.AutoActions))

	if message, ok := response["message"].(string); ok {
		fmt.Printf("   %s\n", message)
	}

	fmt.Printf("\n💡 Next Steps:\n")
	fmt.Printf("   prism budget status %s     # Check budget status\n", projectName)
	fmt.Printf("   prism workspace launch <template> <instance> --project %s  # Launch with budget tracking\n", projectName)
}

// updateBudget updates an existing budget
func (bc *BudgetCommands) updateBudget(cmd *cobra.Command, args []string) error {
	if err := bc.app.ensureDaemonRunning(); err != nil {
		return err
	}

	budgetID := args[0]

	req, hasUpdates, err := bc.buildUpdateBudgetRequest(cmd)
	if err != nil {
		return err
	}

	if !hasUpdates {
		return fmt.Errorf("no updates specified. Use flags to specify what to update")
	}

	response, err := bc.app.apiClient.UpdateProjectBudget(bc.app.ctx, budgetID, req)
	if err != nil {
		return fmt.Errorf("failed to update budget: %w", err)
	}

	bc.displayUpdateBudgetSuccess(budgetID, req, response)
	return nil
}

// buildUpdateBudgetRequest builds update request from command flags
func (bc *BudgetCommands) buildUpdateBudgetRequest(cmd *cobra.Command) (client.UpdateProjectBudgetRequest, bool, error) {
	req := client.UpdateProjectBudgetRequest{}
	hasUpdates := false

	hasUpdates = bc.parseUpdateBudgetLimits(cmd, &req) || hasUpdates

	updated, err := bc.parseUpdateBudgetEndDate(cmd, &req)
	if err != nil {
		return req, false, err
	}
	hasUpdates = updated || hasUpdates

	updated, err = bc.parseUpdateBudgetAlerts(cmd, &req)
	if err != nil {
		return req, false, err
	}
	hasUpdates = updated || hasUpdates

	updated, err = bc.parseUpdateBudgetActions(cmd, &req)
	if err != nil {
		return req, false, err
	}
	hasUpdates = updated || hasUpdates

	return req, hasUpdates, nil
}

// parseUpdateBudgetLimits parses budget limit update flags
func (bc *BudgetCommands) parseUpdateBudgetLimits(cmd *cobra.Command, req *client.UpdateProjectBudgetRequest) bool {
	hasUpdates := false

	if totalBudget, _ := cmd.Flags().GetFloat64("total-budget"); totalBudget > 0 {
		req.TotalBudget = &totalBudget
		hasUpdates = true
	}

	if monthlyLimit, _ := cmd.Flags().GetFloat64("monthly-limit"); monthlyLimit > 0 {
		req.MonthlyLimit = &monthlyLimit
		hasUpdates = true
	}

	if dailyLimit, _ := cmd.Flags().GetFloat64("daily-limit"); dailyLimit > 0 {
		req.DailyLimit = &dailyLimit
		hasUpdates = true
	}

	return hasUpdates
}

// parseUpdateBudgetEndDate parses end date update flag
func (bc *BudgetCommands) parseUpdateBudgetEndDate(cmd *cobra.Command, req *client.UpdateProjectBudgetRequest) (bool, error) {
	if endDateStr, _ := cmd.Flags().GetString("end-date"); endDateStr != "" {
		endDate, err := time.Parse("2006-01-02", endDateStr)
		if err != nil {
			return false, fmt.Errorf("invalid end date format: use YYYY-MM-DD")
		}
		req.EndDate = &endDate
		return true, nil
	}
	return false, nil
}

// parseUpdateBudgetAlerts parses alert update flags
func (bc *BudgetCommands) parseUpdateBudgetAlerts(cmd *cobra.Command, req *client.UpdateProjectBudgetRequest) (bool, error) {
	if alerts, _ := cmd.Flags().GetStringSlice("alert"); len(alerts) > 0 {
		for _, alertStr := range alerts {
			alert, err := bc.parseAlertFlag(alertStr)
			if err != nil {
				return false, fmt.Errorf("invalid alert format '%s': %v", alertStr, err)
			}
			req.AlertThresholds = append(req.AlertThresholds, alert)
		}
		return true, nil
	}
	return false, nil
}

// parseUpdateBudgetActions parses action update flags
func (bc *BudgetCommands) parseUpdateBudgetActions(cmd *cobra.Command, req *client.UpdateProjectBudgetRequest) (bool, error) {
	if actions, _ := cmd.Flags().GetStringSlice("action"); len(actions) > 0 {
		for _, actionStr := range actions {
			action, err := bc.parseActionFlag(actionStr)
			if err != nil {
				return false, fmt.Errorf("invalid action format '%s': %v", actionStr, err)
			}
			req.AutoActions = append(req.AutoActions, action)
		}
		return true, nil
	}
	return false, nil
}

// displayUpdateBudgetSuccess displays budget update success message
func (bc *BudgetCommands) displayUpdateBudgetSuccess(budgetID string, req client.UpdateProjectBudgetRequest, response map[string]interface{}) {
	fmt.Printf("✅ Budget updated successfully for '%s'\n", budgetID)

	if req.TotalBudget != nil {
		fmt.Printf("   Total Budget: $%.2f\n", *req.TotalBudget)
	}
	if req.MonthlyLimit != nil {
		fmt.Printf("   Monthly Limit: $%.2f\n", *req.MonthlyLimit)
	}
	if req.DailyLimit != nil {
		fmt.Printf("   Daily Limit: $%.2f\n", *req.DailyLimit)
	}
	if len(req.AlertThresholds) > 0 {
		fmt.Printf("   Alert Thresholds: %d configured\n", len(req.AlertThresholds))
	}
	if len(req.AutoActions) > 0 {
		fmt.Printf("   Auto Actions: %d configured\n", len(req.AutoActions))
	}

	if message, ok := response["message"].(string); ok {
		fmt.Printf("   %s\n", message)
	}
}

// deleteBudget deletes a budget after confirmation
func (bc *BudgetCommands) deleteBudget(args []string) error {
	if err := bc.app.ensureDaemonRunning(); err != nil {
		return err
	}

	budgetID := args[0]

	// Get budget info for confirmation
	budgetStatus, err := bc.app.apiClient.GetProjectBudgetStatus(bc.app.ctx, budgetID)
	if err != nil {
		return fmt.Errorf("failed to get budget info: %w", err)
	}

	fmt.Printf("⚠️  WARNING: This will permanently delete the budget for '%s'\n", budgetID)
	if budgetStatus.BudgetEnabled {
		fmt.Printf("   Current Budget: $%.2f\n", budgetStatus.TotalBudget)
		fmt.Printf("   Amount Spent: $%.2f\n", budgetStatus.SpentAmount)
	}
	fmt.Printf("   This will disable cost tracking and remove all budget controls.\n")
	fmt.Printf("   Running instances will continue normally.\n\n")
	fmt.Printf("Type the budget ID to confirm deletion: ")

	var confirmation string
	_, _ = fmt.Scanln(&confirmation)

	if confirmation != budgetID {
		fmt.Println("❌ Budget ID doesn't match. Deletion cancelled.")
		return nil
	}

	response, err := bc.app.apiClient.DisableProjectBudget(bc.app.ctx, budgetID)
	if err != nil {
		return fmt.Errorf("failed to delete budget: %w", err)
	}

	fmt.Printf("🗑️ Budget deleted successfully for '%s'\n", budgetID)
	fmt.Printf("   Cost tracking disabled\n")
	fmt.Printf("   All budget alerts and actions removed\n")

	if message, ok := response["message"].(string); ok {
		fmt.Printf("   %s\n", message)
	}

	return nil
}

// infoBudget shows detailed information about a budget
func (bc *BudgetCommands) infoBudget(args []string) error {
	if err := bc.app.ensureDaemonRunning(); err != nil {
		return err
	}

	budgetID := args[0]

	budgetStatus, err := bc.app.apiClient.GetProjectBudgetStatus(bc.app.ctx, budgetID)
	if err != nil {
		return fmt.Errorf("failed to get budget status: %w", err)
	}

	project, err := bc.app.apiClient.GetProject(bc.app.ctx, budgetID)
	if err != nil {
		return fmt.Errorf("failed to get project info: %w", err)
	}

	bc.displayBudgetInfo(budgetID, project, budgetStatus)
	return nil
}

// displayBudgetInfo displays comprehensive budget information
func (bc *BudgetCommands) displayBudgetInfo(budgetID string, proj *types.Project, budgetStatus *project.BudgetStatus) {
	fmt.Printf("💰 Budget Information for '%s'\n\n", budgetID)

	bc.displayProjectDetails(proj)

	if !budgetStatus.BudgetEnabled {
		fmt.Printf("\n❌ Budget: Not enabled\n")
		fmt.Printf("💡 Enable budget tracking with: prism budget create %s <amount>\n", budgetID)
		return
	}

	bc.displayBudgetConfiguration(budgetStatus, proj.Budget)
	bc.displaySpendingProjections(budgetStatus)
	bc.displayAlertConfiguration(proj.Budget)
	bc.displayAutomatedActions(proj.Budget)
	bc.displayActiveAlertsAndActions(budgetStatus)
	bc.displayInfoCommands(budgetID)
}

// displayProjectDetails displays basic project information
func (bc *BudgetCommands) displayProjectDetails(project *types.Project) {
	fmt.Printf("🏗️ Project Details:\n")
	fmt.Printf("   Name: %s\n", project.Name)
	fmt.Printf("   ID: %s\n", project.ID)
	if project.Description != "" {
		fmt.Printf("   Description: %s\n", project.Description)
	}
	fmt.Printf("   Owner: %s\n", project.Owner)
	fmt.Printf("   Members: %d\n", len(project.Members))
	fmt.Printf("   Status: %s\n", strings.ToUpper(string(project.Status)))
}

// displayBudgetConfiguration displays budget settings
func (bc *BudgetCommands) displayBudgetConfiguration(budgetStatus *project.BudgetStatus, budget *types.ProjectBudget) {
	fmt.Printf("\n💰 Budget Configuration:\n")
	fmt.Printf("   Total Budget: $%.2f\n", budgetStatus.TotalBudget)
	fmt.Printf("   Current Spent: $%.2f (%.1f%%)\n",
		budgetStatus.SpentAmount, budgetStatus.SpentPercentage*100)
	fmt.Printf("   Remaining: $%.2f\n", budgetStatus.RemainingBudget)

	if budget == nil {
		return
	}

	if budget.MonthlyLimit != nil {
		fmt.Printf("   Monthly Limit: $%.2f\n", *budget.MonthlyLimit)
	}
	if budget.DailyLimit != nil {
		fmt.Printf("   Daily Limit: $%.2f\n", *budget.DailyLimit)
	}
	fmt.Printf("   Budget Period: %s\n", budget.BudgetPeriod)
	if budget.EndDate != nil {
		fmt.Printf("   End Date: %s\n", budget.EndDate.Format("2006-01-02"))
	}
}

// displaySpendingProjections displays spending analysis
func (bc *BudgetCommands) displaySpendingProjections(budgetStatus *project.BudgetStatus) {
	if budgetStatus.ProjectedMonthlySpend <= 0 {
		return
	}

	fmt.Printf("\n📊 Spending Analysis:\n")
	fmt.Printf("   Projected Monthly: $%.2f\n", budgetStatus.ProjectedMonthlySpend)

	if budgetStatus.DaysUntilBudgetExhausted != nil {
		fmt.Printf("   Days Until Exhausted: %d\n", *budgetStatus.DaysUntilBudgetExhausted)
	}
}

// displayAlertConfiguration displays alert settings
func (bc *BudgetCommands) displayAlertConfiguration(budget *types.ProjectBudget) {
	if budget == nil || len(budget.AlertThresholds) == 0 {
		return
	}

	fmt.Printf("\n🚨 Alert Configuration:\n")
	for i, alert := range budget.AlertThresholds {
		status := "Enabled"
		if !alert.Enabled {
			status = "Disabled"
		}
		fmt.Printf("   Alert %d: %.1f%% threshold (%s, %s)\n",
			i+1, alert.Threshold*100, alert.Type, status)
		if len(alert.Recipients) > 0 {
			fmt.Printf("     Recipients: %s\n", strings.Join(alert.Recipients, ", "))
		}
	}
}

// displayAutomatedActions displays automated action settings
func (bc *BudgetCommands) displayAutomatedActions(budget *types.ProjectBudget) {
	if budget == nil || len(budget.AutoActions) == 0 {
		return
	}

	fmt.Printf("\n⚡ Automated Actions:\n")
	for i, action := range budget.AutoActions {
		status := "Enabled"
		if !action.Enabled {
			status = "Disabled"
		}
		fmt.Printf("   Action %d: %s at %.1f%% (%s)\n",
			i+1, action.Action, action.Threshold*100, status)
	}
}

// displayActiveAlertsAndActions displays current alerts and recent actions
func (bc *BudgetCommands) displayActiveAlertsAndActions(budgetStatus *project.BudgetStatus) {
	if len(budgetStatus.ActiveAlerts) > 0 {
		fmt.Printf("\n🚨 Active Alerts:\n")
		for _, alert := range budgetStatus.ActiveAlerts {
			fmt.Printf("   • %s\n", alert)
		}
	}

	if len(budgetStatus.TriggeredActions) > 0 {
		fmt.Printf("\n⚡ Recent Actions:\n")
		for _, action := range budgetStatus.TriggeredActions {
			fmt.Printf("   • %s\n", action)
		}
	}
}

// displayInfoCommands displays helpful command suggestions
func (bc *BudgetCommands) displayInfoCommands(budgetID string) {
	fmt.Printf("\n💡 Commands:\n")
	fmt.Printf("   prism budget breakdown %s    # Detailed cost breakdown\n", budgetID)
	fmt.Printf("   prism budget usage %s       # Resource usage analysis\n", budgetID)
	fmt.Printf("   prism budget forecast %s    # Spending forecast\n", budgetID)
}

// statusBudget shows current budget status
func (bc *BudgetCommands) statusBudget(args []string) error {
	if err := bc.app.ensureDaemonRunning(); err != nil {
		return err
	}

	// If no budget ID provided, show all budgets
	if len(args) == 0 {
		return bc.listBudgets()
	}

	budgetID := args[0]
	budgetStatus, err := bc.app.apiClient.GetProjectBudgetStatus(bc.app.ctx, budgetID)
	if err != nil {
		return fmt.Errorf("failed to get budget status: %w", err)
	}

	bc.displayBudgetStatus(budgetID, budgetStatus)
	return nil
}

// displayBudgetStatus displays budget status information
func (bc *BudgetCommands) displayBudgetStatus(budgetID string, budgetStatus *project.BudgetStatus) {
	fmt.Printf("💰 Budget Status for '%s'\n\n", budgetID)

	if !budgetStatus.BudgetEnabled {
		fmt.Printf("❌ Budget: Not enabled\n")
		fmt.Printf("💡 Enable with: prism budget create %s <amount>\n", budgetID)
		return
	}

	usagePercent := budgetStatus.SpentPercentage * 100
	bc.displayCurrentStatus(budgetStatus, usagePercent)
	bc.displayBurnRate(budgetStatus)
	bc.displaySurplus(budgetStatus)
	bc.displayStatusProjections(budgetStatus)
	bc.displayStatusAlerts(budgetStatus)
	bc.displayStatusActions(budgetStatus)
	bc.displayStatusQuickActions(budgetID, usagePercent)
}

// displayCurrentStatus displays current budget metrics
func (bc *BudgetCommands) displayCurrentStatus(budgetStatus *project.BudgetStatus, usagePercent float64) {
	fmt.Printf("📊 Current Status:\n")
	fmt.Printf("   Total Budget: $%.2f\n", budgetStatus.TotalBudget)
	fmt.Printf("   Amount Spent: $%.2f\n", budgetStatus.SpentAmount)
	fmt.Printf("   Remaining: $%.2f\n", budgetStatus.RemainingBudget)
	fmt.Printf("   Usage: %.1f%%\n", usagePercent)
	fmt.Printf("   Status: %s\n", bc.getStatusIndicator(usagePercent))
}

// displayBurnRate shows period-aware spending pace information.
func (bc *BudgetCommands) displayBurnRate(budgetStatus *project.BudgetStatus) {
	br := budgetStatus.BurnRate
	if br == nil {
		return
	}

	paceIcon := "🟢"
	switch br.PaceStatus {
	case project.PaceStatusAhead:
		paceIcon = "🟡"
	case project.PaceStatusBehind:
		paceIcon = "🔵"
	case project.PaceStatusOverBudget:
		paceIcon = "🔴"
	}

	fmt.Printf("\n⚡ Burn Rate:\n")
	fmt.Printf("   Daily:   $%.2f/day\n", br.DailyRate)
	fmt.Printf("   Weekly:  $%.2f/week\n", br.WeeklyRate)
	fmt.Printf("   Monthly: $%.2f/month\n", br.MonthlyRate)
	fmt.Printf("   Pace:    %s %s (%.0f%% of allocation)\n",
		paceIcon, br.PaceStatus, br.BurnRateRatio*100)
}

// displaySurplus shows banked surplus / carry-over for periodic budgets.
func (bc *BudgetCommands) displaySurplus(budgetStatus *project.BudgetStatus) {
	s := budgetStatus.Surplus
	if s == nil || s.EffectiveBalance == 0 {
		return
	}

	fmt.Printf("\n🏦 Budget Surplus:\n")
	fmt.Printf("   This Period:    $%.2f\n", s.CurrentPeriodSurplus)
	fmt.Printf("   Banked:         $%.2f\n", s.BankedSurplus)
	fmt.Printf("   Effective Bal:  $%.2f", s.EffectiveBalance)
	if s.SurplusCapped {
		fmt.Printf(" (capped at %.0f%%)", s.SurplusCapPercent*100)
	}
	fmt.Printf("\n")
}

// getStatusIndicator returns color-coded status based on usage percentage
func (bc *BudgetCommands) getStatusIndicator(usagePercent float64) string {
	if usagePercent >= 95 {
		return "🔴 CRITICAL - Budget Nearly Exhausted"
	}
	if usagePercent >= 80 {
		return "🟡 WARNING - High Budget Usage"
	}
	if usagePercent >= 60 {
		return "🟡 MODERATE - Monitor Spending"
	}
	return "🟢 HEALTHY - On Track"
}

// displayStatusProjections displays spending projections
func (bc *BudgetCommands) displayStatusProjections(budgetStatus *project.BudgetStatus) {
	if budgetStatus.ProjectedMonthlySpend <= 0 {
		return
	}

	fmt.Printf("\n📈 Projections:\n")
	fmt.Printf("   Projected Monthly: $%.2f\n", budgetStatus.ProjectedMonthlySpend)

	if budgetStatus.DaysUntilBudgetExhausted != nil {
		days := *budgetStatus.DaysUntilBudgetExhausted
		bc.displayExhaustionWarning(days)
	}
}

// displayExhaustionWarning displays budget exhaustion timeline
func (bc *BudgetCommands) displayExhaustionWarning(days int) {
	if days <= 7 {
		fmt.Printf("   ⚠️  Budget Exhausted In: %d days (URGENT)\n", days)
	} else if days <= 30 {
		fmt.Printf("   ⚠️  Budget Exhausted In: %d days\n", days)
	} else {
		fmt.Printf("   Budget Duration: %d days remaining\n", days)
	}
}

// displayStatusAlerts displays active alerts
func (bc *BudgetCommands) displayStatusAlerts(budgetStatus *project.BudgetStatus) {
	if len(budgetStatus.ActiveAlerts) == 0 {
		return
	}

	fmt.Printf("\n🚨 Active Alerts:\n")
	for _, alert := range budgetStatus.ActiveAlerts {
		fmt.Printf("   • %s\n", alert)
	}
}

// displayStatusActions displays recent triggered actions
func (bc *BudgetCommands) displayStatusActions(budgetStatus *project.BudgetStatus) {
	if len(budgetStatus.TriggeredActions) == 0 {
		return
	}

	fmt.Printf("\n⚡ Recent Actions (last 24h):\n")
	for _, action := range budgetStatus.TriggeredActions {
		fmt.Printf("   • %s\n", action)
	}
}

// displayStatusQuickActions displays helpful command suggestions
func (bc *BudgetCommands) displayStatusQuickActions(budgetID string, usagePercent float64) {
	fmt.Printf("\n💡 Quick Actions:\n")
	fmt.Printf("   prism budget breakdown %s    # See where money is spent\n", budgetID)
	fmt.Printf("   prism budget savings %s      # Find cost optimization opportunities\n", budgetID)
	if usagePercent >= 80 {
		fmt.Printf("   prism workspace list --project %s      # Review running instances\n", budgetID)
		fmt.Printf("   prism workspace hibernate <instance>   # Hibernate idle instances\n")
	}
}

// Additional implementation methods would continue here...
// For brevity, I'm showing the structure and key methods.
// The remaining methods (usageBudget, historyBudget, alertsBudget, etc.)
// would follow similar patterns with appropriate API calls and formatting.

// Helper methods

// parseAlertFlag parses an alert flag in format "percent:type:recipients"
func (bc *BudgetCommands) parseAlertFlag(alertStr string) (types.BudgetAlert, error) {
	parts := strings.Split(alertStr, ":")
	if len(parts) < 2 {
		return types.BudgetAlert{}, fmt.Errorf("format should be 'percent:type[:recipients]'")
	}

	threshold, err := strconv.ParseFloat(parts[0], 64)
	if err != nil {
		return types.BudgetAlert{}, fmt.Errorf("invalid threshold percentage: %s", parts[0])
	}

	if threshold <= 0 || threshold > 100 {
		return types.BudgetAlert{}, fmt.Errorf("threshold must be between 1-100")
	}

	alertType := types.BudgetAlertType(parts[1])
	if alertType != types.BudgetAlertEmail &&
		alertType != types.BudgetAlertSlack &&
		alertType != types.BudgetAlertWebhook {
		return types.BudgetAlert{}, fmt.Errorf("invalid alert type: must be email, slack, or webhook")
	}

	var recipients []string
	if len(parts) >= 3 && parts[2] != "" {
		recipients = strings.Split(parts[2], ",")
	}

	return types.BudgetAlert{
		Threshold:  threshold / 100.0, // Convert percentage to decimal
		Type:       alertType,
		Recipients: recipients,
		Enabled:    true,
	}, nil
}

// parseActionFlag parses an action flag in format "percent:action"
func (bc *BudgetCommands) parseActionFlag(actionStr string) (types.BudgetAutoAction, error) {
	parts := strings.Split(actionStr, ":")
	if len(parts) != 2 {
		return types.BudgetAutoAction{}, fmt.Errorf("format should be 'percent:action'")
	}

	threshold, err := strconv.ParseFloat(parts[0], 64)
	if err != nil {
		return types.BudgetAutoAction{}, fmt.Errorf("invalid threshold percentage: %s", parts[0])
	}

	if threshold <= 0 || threshold > 100 {
		return types.BudgetAutoAction{}, fmt.Errorf("threshold must be between 1-100")
	}

	action := types.BudgetActionType(parts[1])
	if action != types.BudgetActionHibernateAll &&
		action != types.BudgetActionStopAll &&
		action != types.BudgetActionPreventLaunch &&
		action != types.BudgetActionNotifyOnly {
		return types.BudgetAutoAction{}, fmt.Errorf("invalid action: must be hibernate_all, stop_all, prevent_launch, or notify_only")
	}

	return types.BudgetAutoAction{
		Threshold: threshold / 100.0, // Convert percentage to decimal
		Action:    action,
		Enabled:   true,
	}, nil
}

// usageBudget shows detailed resource usage metrics
func (bc *BudgetCommands) usageBudget(cmd *cobra.Command, args []string) error {
	if err := bc.app.ensureDaemonRunning(); err != nil {
		return err
	}

	budgetID := args[0]
	period, _ := cmd.Flags().GetString("period")

	// Parse period
	var duration time.Duration
	switch period {
	case "7d":
		duration = 7 * 24 * time.Hour
	case "30d":
		duration = 30 * 24 * time.Hour
	case "90d":
		duration = 90 * 24 * time.Hour
	default:
		duration = 30 * 24 * time.Hour
	}

	usage, err := bc.app.apiClient.GetProjectResourceUsage(bc.app.ctx, budgetID, duration)
	if err != nil {
		return fmt.Errorf("failed to get resource usage: %w", err)
	}

	fmt.Printf("📊 Resource Usage Analysis for '%s'\n", budgetID)
	fmt.Printf("Period: Last %s\n\n", period)

	fmt.Printf("🖥️ Instance Metrics:\n")
	fmt.Printf("   Active Instances: %d\n", usage.ActiveInstances)
	fmt.Printf("   Total Instances: %d\n", usage.TotalInstances)
	fmt.Printf("   Compute Hours: %.1f\n", usage.ComputeHours)

	fmt.Printf("\n💾 Storage Metrics:\n")
	fmt.Printf("   Total Storage: %.1f GB\n", usage.TotalStorage)

	fmt.Printf("\n💡 Cost Optimization:\n")
	fmt.Printf("   Hibernation Savings: $%.2f\n", usage.IdleSavings)

	if usage.ActiveInstances > 0 {
		utilizationRate := float64(usage.ActiveInstances) / float64(usage.TotalInstances) * 100
		fmt.Printf("   Instance Utilization: %.1f%%\n", utilizationRate)
	}

	fmt.Printf("\n📈 Efficiency Recommendations:\n")
	if usage.IdleSavings > 0 {
		fmt.Printf("   • Hibernation is saving you $%.2f per month\n", usage.IdleSavings*30)
	}

	if usage.ActiveInstances < usage.TotalInstances {
		unusedCount := usage.TotalInstances - usage.ActiveInstances
		fmt.Printf("   • Consider terminating %d unused instances\n", unusedCount)
	}

	return nil
}

// historyBudget shows spending history and trends
func (bc *BudgetCommands) historyBudget(cmd *cobra.Command, args []string) error {
	if err := bc.app.ensureDaemonRunning(); err != nil {
		return err
	}

	budgetID := args[0]
	period, _ := cmd.Flags().GetString("period")
	format, _ := cmd.Flags().GetString("format")

	fmt.Printf("📈 Spending History for '%s'\n", budgetID)
	fmt.Printf("Period: Last %s\n\n", period)

	// Get cost trends (using the existing API endpoint)
	trends, err := bc.getCostTrends(budgetID, period)
	if err != nil {
		return fmt.Errorf("failed to get cost trends: %w", err)
	}

	switch format {
	case "json":
		return bc.outputJSON(trends)
	case "csv":
		return bc.outputCSV(trends)
	default:
		return bc.outputHistoryTable(trends)
	}
}

// alertsBudget manages budget alerts
func (bc *BudgetCommands) alertsBudget(cmd *cobra.Command, args []string) error {
	if err := bc.app.ensureDaemonRunning(); err != nil {
		return err
	}

	budgetID := args[0]
	action, _ := cmd.Flags().GetString("action")

	switch action {
	case "add":
		return bc.addAlert(cmd, budgetID)
	case "remove":
		return bc.removeAlert(cmd, budgetID)
	case "test":
		return bc.testAlert(cmd, budgetID)
	default:
		return bc.listAlerts(budgetID)
	}
}

// forecastBudget shows spending forecasts
func (bc *BudgetCommands) forecastBudget(cmd *cobra.Command, args []string) error {
	if err := bc.app.ensureDaemonRunning(); err != nil {
		return err
	}

	budgetID := args[0]
	horizon, _ := cmd.Flags().GetString("horizon")
	scenario, _ := cmd.Flags().GetString("scenario")

	budgetStatus, err := bc.app.apiClient.GetProjectBudgetStatus(bc.app.ctx, budgetID)
	if err != nil {
		return fmt.Errorf("failed to get budget status: %w", err)
	}

	fmt.Printf("🔮 Spending Forecast for '%s'\n", budgetID)
	fmt.Printf("Horizon: %s | Scenario: %s\n\n", horizon, scenario)

	if !budgetStatus.BudgetEnabled || budgetStatus.ProjectedMonthlySpend <= 0 {
		fmt.Printf("❌ Insufficient data for forecasting\n")
		fmt.Printf("💡 Need at least 7 days of spending data for accurate forecasts\n")
		return nil
	}

	// Calculate forecast based on current trends
	monthlySpend := budgetStatus.ProjectedMonthlySpend

	// Apply scenario adjustments
	switch scenario {
	case "optimistic":
		monthlySpend *= 0.85 // 15% reduction through optimization
	case "conservative":
		monthlySpend *= 1.25 // 25% increase for safety margin
	}

	fmt.Printf("📊 Current Status:\n")
	fmt.Printf("   Budget: $%.2f\n", budgetStatus.TotalBudget)
	fmt.Printf("   Spent: $%.2f (%.1f%%)\n", budgetStatus.SpentAmount, budgetStatus.SpentPercentage*100)
	fmt.Printf("   Current Monthly Rate: $%.2f\n", budgetStatus.ProjectedMonthlySpend)

	fmt.Printf("\n🔮 Forecast (%s scenario):\n", scenario)

	// Calculate periods based on horizon
	periods := map[string]int{"1m": 1, "3m": 3, "6m": 6, "1y": 12}
	months, ok := periods[horizon]
	if !ok {
		months = 3
	}

	totalProjected := monthlySpend * float64(months)
	fmt.Printf("   %s Projected Spend: $%.2f\n", horizon, totalProjected)

	if budgetStatus.DaysUntilBudgetExhausted != nil {
		days := *budgetStatus.DaysUntilBudgetExhausted
		if days > 0 && days <= months*30 {
			fmt.Printf("   ⚠️  Budget exhaustion in %d days\n", days)
		}
	}

	// Budget adequacy analysis
	remainingBudget := budgetStatus.RemainingBudget
	if totalProjected > remainingBudget {
		shortfall := totalProjected - remainingBudget
		fmt.Printf("   ❌ Budget Shortfall: $%.2f\n", shortfall)
		fmt.Printf("   💡 Consider increasing budget by $%.2f\n", shortfall)
	} else {
		surplus := remainingBudget - totalProjected
		fmt.Printf("   ✅ Budget Surplus: $%.2f\n", surplus)
	}

	return nil
}

// savingsBudget shows cost optimization savings
func (bc *BudgetCommands) savingsBudget(cmd *cobra.Command, args []string) error {
	if err := bc.app.ensureDaemonRunning(); err != nil {
		return err
	}

	period, _ := cmd.Flags().GetString("period")
	showRecommendations, _ := cmd.Flags().GetBool("recommendations")

	budgetIDs, err := bc.parseSavingsBudgetIDs(args)
	if err != nil {
		return err
	}

	if len(budgetIDs) == 0 {
		fmt.Printf("No budgets found for analysis.\n")
		return nil
	}

	_, totalPotentialSavings := bc.calculateAndDisplaySavings(budgetIDs, period)

	if showRecommendations {
		bc.displaySavingsRecommendations(totalPotentialSavings)
	}

	return nil
}

// parseSavingsBudgetIDs parses budget IDs from args or retrieves all budgets
func (bc *BudgetCommands) parseSavingsBudgetIDs(args []string) ([]string, error) {
	fmt.Printf("💡 Cost Savings Analysis\n")

	if len(args) > 0 {
		fmt.Printf("Project: %s\n\n", args[0])
		return []string{args[0]}, nil
	}

	projects, err := bc.app.apiClient.ListProjects(bc.app.ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to list projects: %w", err)
	}

	var budgetIDs []string
	for _, proj := range projects.Projects {
		if proj.BudgetStatus != nil && proj.BudgetStatus.TotalBudget > 0 {
			budgetIDs = append(budgetIDs, proj.ID)
		}
	}

	fmt.Printf("Analyzing %d projects with budgets\n\n", len(budgetIDs))
	return budgetIDs, nil
}

// calculateAndDisplaySavings calculates and displays savings for all budgets
func (bc *BudgetCommands) calculateAndDisplaySavings(budgetIDs []string, period string) (float64, float64) {
	fmt.Printf("Period: Last %s\n\n", period)
	fmt.Printf("🏆 Hibernation Savings:\n")

	w := tabwriter.NewWriter(os.Stdout, 0, 8, 2, ' ', 0)
	fmt.Fprintf(w, "PROJECT\tACTUAL SAVINGS\tPOTENTIAL SAVINGS\tEFFICIENCY\n")

	totalSavings := 0.0
	totalPotentialSavings := 0.0

	for _, budgetID := range budgetIDs {
		savings, potential := bc.calculateBudgetSavings(budgetID, period, w)
		totalSavings += savings
		totalPotentialSavings += potential
	}

	bc.displaySavingsTotals(w, totalSavings, totalPotentialSavings)
	return totalSavings, totalPotentialSavings
}

// calculateBudgetSavings calculates savings for a single budget
func (bc *BudgetCommands) calculateBudgetSavings(budgetID, period string, w *tabwriter.Writer) (float64, float64) {
	duration := bc.parsePeriodDuration(period)

	usage, err := bc.app.apiClient.GetProjectResourceUsage(bc.app.ctx, budgetID, duration)
	if err != nil {
		fmt.Fprintf(w, "%s\tError\tError\tError\n", budgetID)
		return 0.0, 0.0
	}

	actualSavings := usage.IdleSavings
	potentialSavings := actualSavings * 0.5 // Assume 50% more savings possible

	efficiency := bc.calculateEfficiency(actualSavings, potentialSavings)

	fmt.Fprintf(w, "%s\t$%.2f\t$%.2f\t%.1f%%\n",
		budgetID, actualSavings, potentialSavings, efficiency)

	return actualSavings, potentialSavings
}

// parsePeriodDuration converts period string to time.Duration
func (bc *BudgetCommands) parsePeriodDuration(period string) time.Duration {
	switch period {
	case "7d":
		return 7 * 24 * time.Hour
	case "30d":
		return 30 * 24 * time.Hour
	case "90d":
		return 90 * 24 * time.Hour
	default:
		return 30 * 24 * time.Hour
	}
}

// calculateEfficiency calculates savings efficiency percentage
func (bc *BudgetCommands) calculateEfficiency(actual, potential float64) float64 {
	if potential <= 0 {
		return 0.0
	}
	return (actual / (actual + potential)) * 100
}

// displaySavingsTotals displays total savings summary
func (bc *BudgetCommands) displaySavingsTotals(w *tabwriter.Writer, totalSavings, totalPotentialSavings float64) {
	totalEfficiency := bc.calculateEfficiency(totalSavings, totalPotentialSavings)
	fmt.Fprintf(w, "\nTOTAL\t$%.2f\t$%.2f\t%.1f%%\n",
		totalSavings, totalPotentialSavings, totalEfficiency)
	_ = w.Flush() // stdout table; this display helper has no error return and a failed terminal write is not actionable
}

// displaySavingsRecommendations displays cost optimization recommendations
func (bc *BudgetCommands) displaySavingsRecommendations(totalPotentialSavings float64) {
	fmt.Printf("\n🎯 Optimization Recommendations:\n")
	if totalPotentialSavings > 0 {
		fmt.Printf("   • Enable hibernation on idle instances: +$%.2f/month\n", totalPotentialSavings)
	}
	fmt.Printf("   • Review instance right-sizing opportunities\n")
	fmt.Printf("   • Consider spot instances for fault-tolerant workloads\n")
	fmt.Printf("   • Implement automated idle detection policies\n")

	fmt.Printf("\n💡 Quick Actions:\n")
	fmt.Printf("   prism idle profile create aggressive --idle-minutes 15\n")
	fmt.Printf("   prism workspace list | grep STOPPED  # Find stopped instances to terminate\n")
	fmt.Printf("   prism rightsizing analyze  # Get right-sizing recommendations\n")
}

func (bc *BudgetCommands) breakdownBudget(cmd *cobra.Command, args []string) error {
	budgetID := args[0]

	if err := bc.app.ensureDaemonRunning(); err != nil {
		return err
	}

	// Get cost breakdown from API
	endDate := time.Now()
	startDate := endDate.AddDate(0, 0, -30) // Default to 30 days

	costBreakdown, err := bc.app.apiClient.GetProjectCostBreakdown(bc.app.ctx, budgetID, startDate, endDate)
	if err != nil {
		return fmt.Errorf("failed to get cost breakdown: %w", err)
	}

	fmt.Printf("💰 Cost Breakdown for '%s'\n", budgetID)
	fmt.Printf("Period: %s to %s\n\n", startDate.Format("2006-01-02"), endDate.Format("2006-01-02"))

	if len(costBreakdown.InstanceCosts) == 0 && len(costBreakdown.StorageCosts) == 0 {
		fmt.Printf("No cost data available for the specified period.\n")
		return nil
	}

	// Instance costs
	if len(costBreakdown.InstanceCosts) > 0 {
		fmt.Printf("🖥️ Instance Costs:\n")
		w := tabwriter.NewWriter(os.Stdout, 0, 8, 2, ' ', 0)
		fmt.Fprintf(w, "INSTANCE\tTYPE\tCOMPUTE\tSTORAGE\tTOTAL\tHOURS\n")

		totalInstanceCost := 0.0
		for _, ic := range costBreakdown.InstanceCosts {
			fmt.Fprintf(w, "%s\t%s\t$%.2f\t$%.2f\t$%.2f\t%.1f\n",
				ic.InstanceName, ic.InstanceType, ic.ComputeCost, ic.StorageCost,
				ic.TotalCost, ic.RunningHours)
			totalInstanceCost += ic.TotalCost
		}
		fmt.Fprintf(w, "\nTOTAL INSTANCES\t\t\t\t$%.2f\t\n", totalInstanceCost)
		if err := w.Flush(); err != nil {
			return err
		}
		fmt.Println()
	}

	// Storage costs
	if len(costBreakdown.StorageCosts) > 0 {
		fmt.Printf("💾 Storage Costs:\n")
		w := tabwriter.NewWriter(os.Stdout, 0, 8, 2, ' ', 0)
		fmt.Fprintf(w, "VOLUME\tTYPE\tSIZE (GB)\tCOST/GB\tTOTAL\n")

		totalStorageCost := 0.0
		for _, sc := range costBreakdown.StorageCosts {
			fmt.Fprintf(w, "%s\t%s\t%.1f\t$%.4f\t$%.2f\n",
				sc.VolumeName, sc.VolumeType, sc.SizeGB, sc.CostPerGB, sc.Cost)
			totalStorageCost += sc.Cost
		}
		fmt.Fprintf(w, "\nTOTAL STORAGE\t\t\t\t$%.2f\n", totalStorageCost)
		if err := w.Flush(); err != nil {
			return err
		}
		fmt.Println()
	}

	// Summary
	fmt.Printf("📊 Summary:\n")
	fmt.Printf("   Total Cost: $%.2f\n", costBreakdown.TotalCost)
	fmt.Printf("   Generated: %s\n", costBreakdown.GeneratedAt.Format("2006-01-02 15:04:05"))

	return nil
}

// Helper methods for budget command implementation

// getCostTrends retrieves cost trends for a project using the API client
func (bc *BudgetCommands) getCostTrends(budgetID, period string) (map[string]interface{}, error) {
	ctx := context.Background()
	trends, err := bc.app.apiClient.GetCostTrends(ctx, budgetID, period)
	if err != nil {
		return nil, fmt.Errorf("failed to get cost trends: %w", err)
	}
	return trends, nil
}

// outputJSON outputs data in JSON format
func (bc *BudgetCommands) outputJSON(data interface{}) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(data)
}

// outputCSV outputs data in CSV format
func (bc *BudgetCommands) outputCSV(data interface{}) error {
	writer := csv.NewWriter(os.Stdout)
	defer writer.Flush()

	// Handle different data types
	switch v := data.(type) {
	case map[string]interface{}:
		// Write header
		if err := writer.Write([]string{"Date", "Amount", "Type"}); err != nil {
			return fmt.Errorf("failed to write CSV header: %w", err)
		}

		// Write data rows
		for key, value := range v {
			row := []string{key, fmt.Sprintf("%v", value), "spending"}
			if err := writer.Write(row); err != nil {
				return fmt.Errorf("failed to write CSV row: %w", err)
			}
		}

	case []interface{}:
		// Handle array data
		if len(v) > 0 {
			// Try to extract keys from first element
			if firstRow, ok := v[0].(map[string]interface{}); ok {
				// Write header from first row keys
				var headers []string
				for key := range firstRow {
					headers = append(headers, key)
				}
				if err := writer.Write(headers); err != nil {
					return fmt.Errorf("failed to write CSV header: %w", err)
				}

				// Write data rows
				for _, item := range v {
					if row, ok := item.(map[string]interface{}); ok {
						var values []string
						for _, key := range headers {
							values = append(values, fmt.Sprintf("%v", row[key]))
						}
						if err := writer.Write(values); err != nil {
							return fmt.Errorf("failed to write CSV row: %w", err)
						}
					}
				}
			}
		}

	default:
		return fmt.Errorf("unsupported data type for CSV output: %T", data)
	}

	return nil
}

// outputHistoryTable outputs spending history in table format with ASCII visualization
func (bc *BudgetCommands) outputHistoryTable(trends map[string]interface{}) error {
	fmt.Printf("📊 Spending History:\n\n")

	// Extract trend data
	trendsList, ok := trends["trends"].([]interface{})
	if !ok || len(trendsList) == 0 {
		fmt.Printf("No historical data available.\n")
		return nil
	}

	// Create table writer
	w := tabwriter.NewWriter(os.Stdout, 0, 8, 2, ' ', 0)
	fmt.Fprintf(w, "DATE\tSPENT\tBUDGET\tUSAGE\tVISUAL\n")

	// Find max value for scaling
	maxValue := 0.0
	for _, item := range trendsList {
		if trend, ok := item.(map[string]interface{}); ok {
			if spent, ok := trend["spent"].(float64); ok && spent > maxValue {
				maxValue = spent
			}
		}
	}

	// Display each trend with ASCII bar chart
	for _, item := range trendsList {
		if trend, ok := item.(map[string]interface{}); ok {
			date := trend["date"].(string)
			spent := trend["spent"].(float64)
			budget := trend["budget"].(float64)
			usage := (spent / budget) * 100

			// Create ASCII bar (max 40 chars)
			barLength := int((spent / maxValue) * 40)
			bar := strings.Repeat("█", barLength)

			// Color code based on usage
			symbol := "🟢"
			if usage >= 80 {
				symbol = "🔴"
			} else if usage >= 60 {
				symbol = "🟡"
			}

			fmt.Fprintf(w, "%s\t$%.2f\t$%.2f\t%.1f%%\t%s %s\n",
				date, spent, budget, usage, symbol, bar)
		}
	}

	return w.Flush()
}

// listAlerts lists all alerts for a budget
func (bc *BudgetCommands) listAlerts(budgetID string) error {
	project, err := bc.app.apiClient.GetProject(bc.app.ctx, budgetID)
	if err != nil {
		return fmt.Errorf("failed to get project info: %w", err)
	}

	fmt.Printf("🚨 Alert Configuration for '%s':\n\n", budgetID)

	if project.Budget == nil || len(project.Budget.AlertThresholds) == 0 {
		fmt.Printf("No alerts configured.\n")
		fmt.Printf("💡 Add an alert: prism budget alerts %s --action add --threshold 80 --type email\n", budgetID)
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 8, 2, ' ', 0)
	fmt.Fprintf(w, "#\tTHRESHOLD\tTYPE\tRECIPIENTS\tSTATUS\n")

	for i, alert := range project.Budget.AlertThresholds {
		status := "Enabled"
		if !alert.Enabled {
			status = "Disabled"
		}

		recipients := "-"
		if len(alert.Recipients) > 0 {
			recipients = strings.Join(alert.Recipients, ",")
		}

		fmt.Fprintf(w, "%d\t%.1f%%\t%s\t%s\t%s\n",
			i+1, alert.Threshold*100, alert.Type, recipients, status)
	}
	if err := w.Flush(); err != nil {
		return err
	}

	fmt.Printf("\n💡 Alert Actions:\n")
	fmt.Printf("   prism budget alerts %s --action add     # Add new alert\n", budgetID)
	fmt.Printf("   prism budget alerts %s --action test    # Test alert delivery\n", budgetID)

	return nil
}

// addAlert adds a new alert to a budget
func (bc *BudgetCommands) addAlert(cmd *cobra.Command, budgetID string) error {
	threshold, _ := cmd.Flags().GetFloat64("threshold")
	alertType, _ := cmd.Flags().GetString("type")
	recipients, _ := cmd.Flags().GetStringSlice("recipients")
	message, _ := cmd.Flags().GetString("message")

	if threshold <= 0 || threshold > 100 {
		return fmt.Errorf("threshold must be between 1-100")
	}

	if alertType == "" {
		return fmt.Errorf("alert type required: --type email|slack|webhook")
	}

	alert := types.BudgetAlert{
		Threshold:  threshold / 100.0, // Convert to decimal
		Type:       types.BudgetAlertType(alertType),
		Recipients: recipients,
		Enabled:    true,
	}

	if message != "" {
		alert.Message = message
	}

	// Get current project budget and add the alert
	project, err := bc.app.apiClient.GetProject(bc.app.ctx, budgetID)
	if err != nil {
		return fmt.Errorf("failed to get project info: %w", err)
	}

	if project.Budget == nil {
		return fmt.Errorf("no budget configured for project %s", budgetID)
	}

	// Add the new alert to existing alerts
	project.Budget.AlertThresholds = append(project.Budget.AlertThresholds, alert)

	// Update the budget
	updateReq := client.UpdateProjectBudgetRequest{
		AlertThresholds: project.Budget.AlertThresholds,
	}

	_, err = bc.app.apiClient.UpdateProjectBudget(bc.app.ctx, budgetID, updateReq)
	if err != nil {
		return fmt.Errorf("failed to add alert: %w", err)
	}

	fmt.Printf("✅ Alert added successfully\n")
	fmt.Printf("   Threshold: %.1f%%\n", threshold)
	fmt.Printf("   Type: %s\n", alertType)
	if len(recipients) > 0 {
		fmt.Printf("   Recipients: %s\n", strings.Join(recipients, ", "))
	}

	return nil
}

// removeAlert removes an alert from a budget
func (bc *BudgetCommands) removeAlert(cmd *cobra.Command, budgetID string) error {
	threshold, _ := cmd.Flags().GetFloat64("threshold")

	if threshold <= 0 {
		return fmt.Errorf("specify threshold to remove: --threshold <percent>")
	}

	project, err := bc.app.apiClient.GetProject(bc.app.ctx, budgetID)
	if err != nil {
		return fmt.Errorf("failed to get project info: %w", err)
	}

	if project.Budget == nil || len(project.Budget.AlertThresholds) == 0 {
		return fmt.Errorf("no alerts configured for project %s", budgetID)
	}

	// Find and remove the alert
	var newAlerts []types.BudgetAlert
	found := false

	for _, alert := range project.Budget.AlertThresholds {
		if alert.Threshold != threshold/100.0 {
			newAlerts = append(newAlerts, alert)
		} else {
			found = true
		}
	}

	if !found {
		return fmt.Errorf("alert with threshold %.1f%% not found", threshold)
	}

	// Update the budget
	updateReq := client.UpdateProjectBudgetRequest{
		AlertThresholds: newAlerts,
	}

	_, err = bc.app.apiClient.UpdateProjectBudget(bc.app.ctx, budgetID, updateReq)
	if err != nil {
		return fmt.Errorf("failed to remove alert: %w", err)
	}

	fmt.Printf("✅ Alert removed successfully\n")
	fmt.Printf("   Threshold: %.1f%%\n", threshold)

	return nil
}

// testAlert tests alert delivery
func (bc *BudgetCommands) testAlert(cmd *cobra.Command, budgetID string) error {
	threshold, _ := cmd.Flags().GetFloat64("threshold")

	if threshold <= 0 {
		threshold = 80.0 // Default test threshold
	}

	fmt.Printf("🧪 Testing alert delivery for '%s'\n", budgetID)
	fmt.Printf("   Test Threshold: %.1f%%\n", threshold)

	// Get project info to find configured alerts
	project, err := bc.app.apiClient.GetProject(bc.app.ctx, budgetID)
	if err != nil {
		return fmt.Errorf("failed to get project: %w", err)
	}

	if project.Budget == nil || len(project.Budget.AlertThresholds) == 0 {
		fmt.Printf("   ⚠️  No alerts configured. Configure alerts first:\n")
		fmt.Printf("      prism budget alerts %s --action add --threshold 80 --type email\n", budgetID)
		return nil
	}

	// Find matching alert
	var testAlert *types.BudgetAlert
	for i := range project.Budget.AlertThresholds {
		if project.Budget.AlertThresholds[i].Threshold == threshold/100.0 {
			testAlert = &project.Budget.AlertThresholds[i]
			break
		}
	}

	if testAlert == nil {
		fmt.Printf("   ⚠️  No alert configured at %.1f%% threshold\n", threshold)
		return nil
	}

	// Simulate test alert delivery
	fmt.Printf("   📧 Simulating alert delivery...\n")
	fmt.Printf("   Recipients: %v\n", testAlert.Recipients)
	fmt.Printf("   Type: %s\n", testAlert.Type)
	fmt.Printf("   Message: Budget threshold %.1f%% reached for project '%s'\n", threshold, budgetID)
	fmt.Printf("\n")
	fmt.Printf("   ✅ Test alert would be delivered to configured recipients\n")
	fmt.Printf("   💡 Actual alerts are sent automatically when thresholds are reached\n")

	return nil
}

// createCushionCommand creates the budget cushion command group.
func (bc *BudgetCommands) createCushionCommand() *cobra.Command {
	cushionCmd := &cobra.Command{
		Use:   "cushion <project-id>",
		Short: "Manage budget safety cushion (auto-action when near limit)",
		Long: `Configure a budget safety cushion that automatically takes action
before the full budget is exhausted.

Cushion modes:
  hibernate      — hibernate all running instances
  stop           — stop all running instances (no resume)
  prevent_launch — block new launches without stopping running instances
  notify_only    — send an alert but take no automated action

Examples:
  prism budget cushion my-project                      # show current config
  prism budget cushion my-project --enable --headroom 0.10 --mode hibernate
  prism budget cushion my-project --disable`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return bc.manageCushion(cmd, args)
		},
	}

	cushionCmd.Flags().Bool("enable", false, "Enable the cushion")
	cushionCmd.Flags().Bool("disable", false, "Disable the cushion")
	cushionCmd.Flags().Float64("headroom", 0.10, "Headroom as a fraction of total budget (0.0–1.0)")
	cushionCmd.Flags().Float64("headroom-fixed", 0, "Fixed-dollar headroom (overrides --headroom when > 0)")
	cushionCmd.Flags().String("mode", "hibernate", "Action mode: hibernate|stop|prevent_launch|notify_only")
	cushionCmd.Flags().Bool("notify-before", false, "Send warning alert before executing the action")
	cushionCmd.Flags().Int("warn-hours", 6, "Hours before projected action to send the warning")

	return cushionCmd
}

func (bc *BudgetCommands) manageCushion(cmd *cobra.Command, args []string) error {
	projectID := args[0]
	ctx := context.Background()

	disable, _ := cmd.Flags().GetBool("disable")
	if disable {
		if err := bc.app.apiClient.DeleteProjectCushion(ctx, projectID); err != nil {
			return fmt.Errorf("failed to remove cushion: %w", err)
		}
		fmt.Printf("✅ Budget cushion disabled for project '%s'\n", projectID)
		return nil
	}

	enable, _ := cmd.Flags().GetBool("enable")
	if enable {
		headroom, _ := cmd.Flags().GetFloat64("headroom")
		headroomFixed, _ := cmd.Flags().GetFloat64("headroom-fixed")
		mode, _ := cmd.Flags().GetString("mode")
		notifyBefore, _ := cmd.Flags().GetBool("notify-before")
		warnHours, _ := cmd.Flags().GetInt("warn-hours")

		var fixedPtr *float64
		if headroomFixed > 0 {
			fixedPtr = &headroomFixed
		}

		cfg := map[string]interface{}{
			"enabled":              true,
			"headroom_percent":     headroom,
			"headroom_fixed_usd":   fixedPtr,
			"mode":                 mode,
			"notify_before_action": notifyBefore,
			"warn_lead_hours":      warnHours,
		}
		result, err := bc.app.apiClient.SetProjectCushion(ctx, projectID, cfg)
		if err != nil {
			return fmt.Errorf("failed to set cushion: %w", err)
		}
		fmt.Printf("✅ Budget cushion enabled for project '%s'\n", projectID)
		fmt.Printf("   Headroom: %.0f%%  Mode: %s\n",
			result["headroom_percent"].(float64)*100, result["mode"])
		return nil
	}

	// Default: show current cushion config
	cushion, err := bc.app.apiClient.GetProjectCushion(ctx, projectID)
	if err != nil {
		return fmt.Errorf("failed to get cushion config: %w", err)
	}

	fmt.Printf("🛡️  Budget Cushion for '%s'\n\n", projectID)
	if enabled, ok := cushion["enabled"].(bool); !ok || !enabled {
		fmt.Printf("   Status: Disabled\n")
		fmt.Printf("💡 Enable with: prism budget cushion %s --enable --headroom 0.10 --mode hibernate\n", projectID)
		return nil
	}

	fmt.Printf("   Status:       Enabled\n")
	if pct, ok := cushion["headroom_percent"].(float64); ok {
		fmt.Printf("   Headroom:     %.0f%%\n", pct*100)
	}
	if mode, ok := cushion["mode"].(string); ok {
		fmt.Printf("   Mode:         %s\n", mode)
	}
	if nb, ok := cushion["notify_before_action"].(bool); ok && nb {
		if wh, ok := cushion["warn_lead_hours"].(float64); ok {
			fmt.Printf("   Warn Before:  %.0f hours\n", wh)
		}
	}
	return nil
}

// createGDEWCommand creates the `prism budget gdew <project>` command (Issue #206).
func (bc *BudgetCommands) createGDEWCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "gdew <project-id>",
		Short: "Show AWS Global Data Egress Waiver (GDEW) credit status",
		Long: `Display GDEW credit status for a project.

GDEW provides credits up to 15% of total monthly AWS spend that can be applied
against data egress charges. This command shows the current coverage percentage,
available credit, and any warnings about nearing the credit limit.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return bc.showGDEWStatus(cmd, args)
		},
	}
}

func (bc *BudgetCommands) showGDEWStatus(cmd *cobra.Command, args []string) error {
	if err := bc.app.ensureDaemonRunning(); err != nil {
		return err
	}
	ctx := bc.app.ctx
	projectID := args[0]

	status, err := bc.app.apiClient.GetProjectGDEW(ctx, projectID)
	if err != nil {
		return fmt.Errorf("failed to get GDEW status: %w", err)
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintf(w, "🌐 GDEW Credit Status — %s\n\n", projectID)
	fmt.Fprintf(w, "  Total Spend MTD:\t$%.2f\n", floatFromMap(status, "total_spend_mtd"))
	fmt.Fprintf(w, "  Egress Charges MTD:\t$%.2f\n", floatFromMap(status, "egress_charges_mtd"))
	fmt.Fprintf(w, "  Available Credit (15%%):\t$%.2f\n", floatFromMap(status, "available_credit"))
	fmt.Fprintf(w, "  Used Credit:\t$%.2f\n", floatFromMap(status, "used_credit"))
	fmt.Fprintf(w, "  Remaining Credit:\t$%.2f\n", floatFromMap(status, "remaining_credit"))
	fmt.Fprintf(w, "  Net Egress Cost:\t$%.2f\n", floatFromMap(status, "net_egress_cost"))
	fmt.Fprintf(w, "  Coverage:\t%.0f%%\n", floatFromMap(status, "coverage_percent"))
	_ = w.Flush()

	if msg, ok := status["status_message"].(string); ok && msg != "" {
		fmt.Printf("\n  %s\n", msg)
	}
	return nil
}

// floatFromMap is a helper to safely extract a float64 from a map[string]interface{}.
func floatFromMap(m map[string]interface{}, key string) float64 {
	if v, ok := m[key].(float64); ok {
		return v
	}
	return 0
}

// createDiscountsCommand creates `prism budget discounts <project>` (Issue #207).
func (bc *BudgetCommands) createDiscountsCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "discounts <project-id>",
		Short: "Show AWS discount discovery results for a project",
		Long: `Display active EDP/PPA discounts discovered for a project's AWS account.

Discount discovery queries the AWS Cost Explorer API. If Cost Explorer is not
enabled on the account, a graceful empty result is returned.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return bc.showDiscounts(cmd, args)
		},
	}
}

func (bc *BudgetCommands) showDiscounts(cmd *cobra.Command, args []string) error {
	if err := bc.app.ensureDaemonRunning(); err != nil {
		return err
	}
	ctx := bc.app.ctx
	projectID := args[0]

	adj, err := bc.app.apiClient.GetProjectDiscounts(ctx, projectID)
	if err != nil {
		return fmt.Errorf("failed to get discounts: %w", err)
	}

	fmt.Printf("💸 Discount Discovery — %s\n\n", projectID)
	discounts, _ := adj["discounts"].([]interface{})
	if len(discounts) == 0 {
		fmt.Printf("  No active discounts discovered.\n")
		fmt.Printf("  Enable AWS Cost Explorer to discover EDP/PPA discounts.\n")
	} else {
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintf(w, "  TYPE\tDISCOUNT\tSTATUS\n")
		for _, d := range discounts {
			if dm, ok := d.(map[string]interface{}); ok {
				fmt.Fprintf(w, "  %s\t%.0f%%\t%v\n", dm["type"], floatFromMap(dm, "percentage"), dm["active"])
			}
		}
		_ = w.Flush()
	}

	if rate, ok := adj["effective_cost_rate"].(float64); ok && rate < 1.0 {
		fmt.Printf("\n  Effective cost rate: %.0f%% (%.0f%% discount applied)\n", rate*100, (1-rate)*100)
	}
	return nil
}

// createCreditsCommand creates `prism budget credits <project>` (Issue #207).
func (bc *BudgetCommands) createCreditsCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "credits <project-id>",
		Short: "Show AWS credit balances and expiry warnings for a project",
		Long: `Display time-limited AWS credits for a project.

Shows remaining credit balances and warns about credits expiring within 30 days.
Requires AWS Cost Explorer access for real data; returns empty result otherwise.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return bc.showCredits(cmd, args)
		},
	}
}

func (bc *BudgetCommands) showCredits(cmd *cobra.Command, args []string) error {
	if err := bc.app.ensureDaemonRunning(); err != nil {
		return err
	}
	ctx := bc.app.ctx
	projectID := args[0]

	adj, err := bc.app.apiClient.GetProjectCredits(ctx, projectID)
	if err != nil {
		return fmt.Errorf("failed to get credits: %w", err)
	}

	fmt.Printf("🎟️  AWS Credit Balances — %s\n\n", projectID)
	credits, _ := adj["credits"].([]interface{})
	if len(credits) == 0 {
		fmt.Printf("  No active credits discovered.\n")
		fmt.Printf("  Enable AWS Cost Explorer to discover time-limited credits.\n")
	} else {
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintf(w, "  NAME\tREMAINING\tEXPIRES (DAYS)\tWARNING\n")
		for _, c := range credits {
			if cm, ok := c.(map[string]interface{}); ok {
				warn := ""
				if expWarn, _ := cm["expires_in_warning"].(bool); expWarn {
					warn = "⚠️  EXPIRING SOON"
				}
				fmt.Fprintf(w, "  %s\t$%.2f\t%v\t%s\n",
					cm["name"], floatFromMap(cm, "remaining_amount"),
					cm["days_until_expiry"], warn)
			}
		}
		_ = w.Flush()
	}

	if bal := floatFromMap(adj, "total_credit_balance"); bal > 0 {
		fmt.Printf("\n  Total credit balance: $%.2f\n", bal)
	}
	return nil
}

// ============================================================================
// v0.12.0: Rollover, Share, Reallocate, Borrow, Emergency, Report commands
// ============================================================================

// createRolloverCommand creates the `prism budget rollover` command (#143)
func (bc *BudgetCommands) createRolloverCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "rollover <project>",
		Short: "Configure budget rollover for a project",
		Long: `Enable or disable budget rollover so unspent funds carry into the next period.

Examples:
  prism budget rollover my-project --enable
  prism budget rollover my-project --enable --cap 500
  prism budget rollover my-project --disable`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			projectName := args[0]
			enable, _ := cmd.Flags().GetBool("enable")
			disable, _ := cmd.Flags().GetBool("disable")
			cap, _ := cmd.Flags().GetFloat64("cap")

			proj, err := bc.app.apiClient.GetProject(bc.app.ctx, projectName)
			if err != nil {
				return fmt.Errorf("project %q not found: %w", projectName, err)
			}

			budget := proj.Budget
			if budget == nil {
				budget = &types.ProjectBudget{}
			}

			if enable {
				budget.RolloverEnabled = true
				budget.RolloverCap = cap
			} else if disable {
				budget.RolloverEnabled = false
				budget.RolloverCap = 0
			}

			req := client.UpdateProjectBudgetRequest{
				TotalBudget:     &budget.TotalBudget,
				RolloverEnabled: &budget.RolloverEnabled,
				RolloverCap:     &budget.RolloverCap,
			}
			if _, err := bc.app.apiClient.UpdateProjectBudget(bc.app.ctx, proj.ID, req); err != nil {
				return fmt.Errorf("failed to update rollover settings: %w", err)
			}

			if budget.RolloverEnabled {
				if budget.RolloverCap > 0 {
					fmt.Printf("✅ Rollover enabled for %s (cap: $%.2f)\n", projectName, budget.RolloverCap)
				} else {
					fmt.Printf("✅ Rollover enabled for %s (no cap)\n", projectName)
				}
			} else {
				fmt.Printf("✅ Rollover disabled for %s\n", projectName)
			}
			return nil
		},
	}
	cmd.Flags().Bool("enable", false, "Enable budget rollover")
	cmd.Flags().Bool("disable", false, "Disable budget rollover")
	cmd.Flags().Float64("cap", 0, "Maximum USD amount that can roll over (0 = unlimited)")
	return cmd
}

// createShareCommand creates the `prism budget share` command (#145)
func (bc *BudgetCommands) createShareCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "share",
		Short: "Share budget between projects",
		Long: `Transfer a portion of one project's budget to another project.

Examples:
  prism budget share --from project-a --to project-b --amount 200
  prism budget share --from grant-pool --to summer-intern --amount 150 --expires 30d`,
		RunE: func(cmd *cobra.Command, args []string) error {
			from, _ := cmd.Flags().GetString("from")
			to, _ := cmd.Flags().GetString("to")
			amount, _ := cmd.Flags().GetFloat64("amount")
			expires, _ := cmd.Flags().GetString("expires")
			reason, _ := cmd.Flags().GetString("reason")

			if from == "" || to == "" || amount <= 0 {
				return fmt.Errorf("--from, --to, and --amount are required")
			}

			shareReq := types.BudgetShareRequest{
				FromProjectID: from,
				ToProjectID:   to,
				Amount:        amount,
				Reason:        reason,
			}

			if expires != "" {
				d, err := parseBudgetDuration(expires)
				if err != nil {
					return fmt.Errorf("invalid --expires value: %w", err)
				}
				t := time.Now().Add(d)
				shareReq.ExpiresAt = &t
			}

			// Resolve project name → UUID before submitting
			srcProj, err := bc.app.apiClient.GetProject(bc.app.ctx, from)
			if err != nil {
				return fmt.Errorf("source project %q not found: %w", from, err)
			}
			shareReq.FromProjectID = srcProj.ID

			result, err := bc.app.apiClient.ShareProjectBudget(bc.app.ctx, shareReq)
			if err != nil {
				return fmt.Errorf("budget share failed: %w", err)
			}

			fmt.Printf("✅ Budget share created (ID: %s)\n", result.ID)
			fmt.Printf("   $%.2f transferred from %s → %s\n", amount, from, to)
			if result.ExpiresAt != nil {
				fmt.Printf("   Expires: %s\n", result.ExpiresAt.Format("2006-01-02"))
			}
			return nil
		},
	}
	cmd.Flags().String("from", "", "Source project name or ID")
	cmd.Flags().String("to", "", "Destination project name or ID")
	cmd.Flags().Float64("amount", 0, "Amount in USD to share")
	cmd.Flags().String("expires", "", "Auto-reverse after duration (e.g. 30d, 2w)")
	cmd.Flags().String("reason", "", "Optional note for audit log")
	return cmd
}

// createReallocateCommand creates the `prism budget reallocate` command (#155)
func (bc *BudgetCommands) createReallocateCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "reallocate",
		Short: "Reallocate budget between project members",
		Long: `Move a portion of budget from one member's allocation to another.

Example:
  prism budget reallocate --from postdoc1 --to postdoc2 --amount 100 --project my-lab`,
		RunE: func(cmd *cobra.Command, args []string) error {
			from, _ := cmd.Flags().GetString("from")
			to, _ := cmd.Flags().GetString("to")
			amount, _ := cmd.Flags().GetFloat64("amount")
			projName, _ := cmd.Flags().GetString("project")

			if from == "" || to == "" || amount <= 0 {
				return fmt.Errorf("--from, --to, and --amount are required")
			}

			proj, err := bc.app.apiClient.GetProject(bc.app.ctx, projName)
			if err != nil {
				return fmt.Errorf("project %q not found: %w", projName, err)
			}

			shareReq := types.BudgetShareRequest{
				FromProjectID: proj.ID,
				ToMemberID:    to,
				Amount:        amount,
				Reason:        fmt.Sprintf("reallocated from %s", from),
			}

			_, err = bc.app.apiClient.ShareProjectBudget(bc.app.ctx, shareReq)
			if err != nil {
				return fmt.Errorf("reallocation failed: %w", err)
			}

			fmt.Printf("✅ $%.2f reallocated from %s → %s in project %s\n", amount, from, to, projName)
			return nil
		},
	}
	cmd.Flags().String("from", "", "Source member user ID")
	cmd.Flags().String("to", "", "Destination member user ID")
	cmd.Flags().Float64("amount", 0, "Amount in USD to reallocate")
	cmd.Flags().String("project", "", "Project name or ID")
	return cmd
}

// createBorrowCommand creates the `prism budget borrow` command (#156)
func (bc *BudgetCommands) createBorrowCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "borrow",
		Short: "Temporarily borrow budget from another project",
		Long: `Borrow a temporary budget allocation from a source project.
The borrowed amount is automatically reversed when the expiry passes.

Example:
  prism budget borrow --from discretionary --amount 500 --expires 14d`,
		RunE: func(cmd *cobra.Command, args []string) error {
			from, _ := cmd.Flags().GetString("from")
			amount, _ := cmd.Flags().GetFloat64("amount")
			expires, _ := cmd.Flags().GetString("expires")
			reason, _ := cmd.Flags().GetString("reason")

			if from == "" || amount <= 0 || expires == "" {
				return fmt.Errorf("--from, --amount, and --expires are required")
			}

			d, err := parseBudgetDuration(expires)
			if err != nil {
				return fmt.Errorf("invalid --expires: %w", err)
			}
			t := time.Now().Add(d)

			srcProj, err := bc.app.apiClient.GetProject(bc.app.ctx, from)
			if err != nil {
				return fmt.Errorf("source project %q not found: %w", from, err)
			}

			shareReq := types.BudgetShareRequest{
				FromProjectID: srcProj.ID,
				Amount:        amount,
				Reason:        reason,
				ExpiresAt:     &t,
			}

			result, err := bc.app.apiClient.ShareProjectBudget(bc.app.ctx, shareReq)
			if err != nil {
				return fmt.Errorf("borrow request failed: %w", err)
			}

			fmt.Printf("✅ Borrowed $%.2f from %s (expires %s, ID: %s)\n",
				amount, from, t.Format("2006-01-02"), result.ID)
			return nil
		},
	}
	cmd.Flags().String("from", "", "Source project to borrow from")
	cmd.Flags().Float64("amount", 0, "Amount in USD to borrow")
	cmd.Flags().String("expires", "", "Auto-reverse after duration (e.g. 14d, 1w)")
	cmd.Flags().String("reason", "", "Optional reason for audit log")
	return cmd
}

// createEmergencyCommand creates the `prism budget emergency` command (#157)
func (bc *BudgetCommands) createEmergencyCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "emergency <project>",
		Short: "Request emergency budget increase",
		Long: `Submit an emergency budget increase request for a project.
The request requires approval from the project owner or admin.

Example:
  prism budget emergency my-project --amount 500 --reason "critical paper deadline"`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			projectName := args[0]
			amount, _ := cmd.Flags().GetFloat64("amount")
			reason, _ := cmd.Flags().GetString("reason")

			if amount <= 0 || reason == "" {
				return fmt.Errorf("--amount and --reason are required")
			}

			proj, err := bc.app.apiClient.GetProject(bc.app.ctx, projectName)
			if err != nil {
				return fmt.Errorf("project %q not found: %w", projectName, err)
			}

			req, err := bc.app.apiClient.SubmitApproval(bc.app.ctx, proj.ID, project.ApprovalTypeEmergency,
				map[string]interface{}{"amount": amount}, reason)
			if err != nil {
				return fmt.Errorf("failed to submit emergency request: %w", err)
			}

			fmt.Printf("✅ Emergency budget request submitted\n")
			fmt.Printf("   Request ID: %s\n", req.ID)
			fmt.Printf("   Amount:     $%.2f\n", amount)
			fmt.Printf("   Status:     %s (pending approval)\n", req.Status)
			fmt.Printf("\nTo approve: prism approve %s\n", req.ID)
			return nil
		},
	}
	cmd.Flags().Float64("amount", 0, "USD amount to request")
	cmd.Flags().String("reason", "", "Justification for the emergency increase")
	return cmd
}

// createReportCommand creates the `prism budget report` command (#141)
func (bc *BudgetCommands) createReportCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "report <project>",
		Short: "Generate monthly budget report",
		Long: `Generate a monthly budget report for a project.

Supported formats: json (default), text, csv

Examples:
  prism budget report my-project --month 2026-02
  prism budget report my-project --month 2026-02 --format text
  prism budget report my-project --month 2026-02 --format csv > report.csv`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			projectName := args[0]
			month, _ := cmd.Flags().GetString("month")
			format, _ := cmd.Flags().GetString("format")

			if month == "" {
				month = time.Now().Format("2006-01")
			}

			proj, err := bc.app.apiClient.GetProject(bc.app.ctx, projectName)
			if err != nil {
				return fmt.Errorf("project %q not found: %w", projectName, err)
			}

			output, err := bc.app.apiClient.GetMonthlyReport(bc.app.ctx, proj.ID, month, format)
			if err != nil {
				return fmt.Errorf("failed to get report: %w", err)
			}

			fmt.Print(output)
			return nil
		},
	}
	cmd.Flags().String("month", "", "Month in YYYY-MM format (default: current month)")
	cmd.Flags().String("format", "text", "Output format: json, text, csv")
	return cmd
}

// parseBudgetDuration parses human-readable durations like "30d", "2w", "4h"
func parseBudgetDuration(s string) (time.Duration, error) {
	if len(s) < 2 {
		return 0, fmt.Errorf("invalid duration: %q", s)
	}

	unit := s[len(s)-1]
	valueStr := s[:len(s)-1]
	value, err := strconv.ParseFloat(valueStr, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid duration value %q: %w", valueStr, err)
	}

	switch unit {
	case 'h':
		return time.Duration(value * float64(time.Hour)), nil
	case 'd':
		return time.Duration(value * float64(24*time.Hour)), nil
	case 'w':
		return time.Duration(value * float64(7*24*time.Hour)), nil
	case 'm':
		return time.Duration(value * float64(30*24*time.Hour)), nil
	default:
		return 0, fmt.Errorf("unknown duration unit %q (use h, d, w, or m)", string(unit))
	}
}

// Ensure unused imports are referenced
var _ = csv.NewWriter
var _ = json.NewEncoder
var _ = strconv.ParseFloat
var _ = strings.Contains
var _ = context.Background
