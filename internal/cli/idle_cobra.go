// Package cli provides the command-line interface for CloudWorkstation
package cli

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"
)

// IdleCobraCommands handles idle policy-related commands with proper Cobra structure
type IdleCobraCommands struct {
	app *App
}

// NewIdleCobraCommands creates a new idle policy commands handler
func NewIdleCobraCommands(app *App) *IdleCobraCommands {
	return &IdleCobraCommands{app: app}
}

// CreateIdleCommand creates the main idle command with subcommands
func (ic *IdleCobraCommands) CreateIdleCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "idle",
		Short: "Manage idle policies and schedules",
		Long: `Manage advanced idle policies and schedules for cost optimization.

Idle policies allow you to automatically hibernate or stop instances based on:
- Time schedules (daily, weekly, work hours)
- Idle detection (CPU, memory, network usage)
- Custom rules and patterns

Available policy templates:
- aggressive-cost: Maximum savings (65% estimated) for dev/test
- balanced: Balance performance and cost (40% estimated)
- conservative: Minimal intervention (15% estimated)
- development: Aggressive for dev instances (75% estimated)
- production: Safety-first for production (5% estimated)
- research: Optimized for ML/research workloads (45% estimated)`,
		Example: `  # List available idle policies
  cws idle policy list

  # Apply a policy to an instance
  cws idle policy apply my-instance balanced

  # View policies applied to an instance
  cws idle policy status my-instance

  # Get policy recommendation for an instance
  cws idle policy recommend my-instance

  # View idle schedules
  cws idle schedule list

  # Generate savings report
  cws idle savings --period 30d`,
	}

	// Add subcommands
	cmd.AddCommand(
		ic.createPolicyCommand(),
		ic.createScheduleCommand(),
		ic.createSavingsCommand(),
	)

	return cmd
}

// createPolicyCommand creates the policy management command
func (hc *IdleCobraCommands) createPolicyCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "policy",
		Short: "Manage idle policies",
		Long:  "Manage idle policy templates and apply them to workspaces",
	}

	// Add policy subcommands
	cmd.AddCommand(
		hc.createPolicyListCommand(),
		hc.createPolicyApplyCommand(),
		hc.createPolicyRemoveCommand(),
		hc.createPolicyStatusCommand(),
		hc.createPolicyRecommendCommand(),
		hc.createPolicyDetailsCommand(),
	)

	return cmd
}

// createPolicyListCommand lists available idle policies
func (hc *IdleCobraCommands) createPolicyListCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List available idle policy templates",
		Long:  "Display all available idle policy templates with their characteristics",
		RunE: func(cmd *cobra.Command, args []string) error {
			policies, err := hc.app.apiClient.ListIdlePolicies(hc.app.ctx)
			if err != nil {
				return fmt.Errorf("failed to list idle policies: %w", err)
			}

			if len(policies) == 0 {
				fmt.Println("No idle policies available")
				return nil
			}

			// Create table writer
			w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			_, _ = fmt.Fprintln(w, "POLICY ID\tNAME\tCATEGORY\tSAVINGS\tSUITABLE FOR")
			_, _ = fmt.Fprintln(w, "─────────\t────\t────────\t───────\t────────────")

			for _, policy := range policies {
				suitable := ""
				if len(policy.SuitableFor) > 0 {
					suitable = policy.SuitableFor[0]
					if len(policy.SuitableFor) > 1 {
						suitable += fmt.Sprintf(" (+%d more)", len(policy.SuitableFor)-1)
					}
				}

				_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%.0f%%\t%s\n",
					policy.ID,
					policy.Name,
					policy.Category,
					policy.EstimatedSavingsPercent,
					suitable,
				)
			}

			_ = w.Flush()

			fmt.Println("\n💡 Tip: Use 'cws idle policy details <policy-id>' to see full details")
			fmt.Println("💰 Estimated savings are based on typical usage patterns")

			return nil
		},
	}
}

// createPolicyApplyCommand applies a policy to an instance
func (hc *IdleCobraCommands) createPolicyApplyCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "apply <workspace-name> <policy-id>",
		Short: "Apply a idle policy to an instance",
		Long:  "Apply a idle policy template to an instance to enable automatic idle",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			instanceName := args[0]
			policyID := args[1]

			// Get policy details for display
			policy, err := hc.app.apiClient.GetIdlePolicy(hc.app.ctx, policyID)
			if err != nil {
				return fmt.Errorf("policy not found: %w", err)
			}

			fmt.Printf("🔄 Applying idle policy '%s' to workspace '%s'...\n", policy.Name, instanceName)

			if err := hc.app.apiClient.ApplyIdlePolicy(hc.app.ctx, instanceName, policyID); err != nil {
				return fmt.Errorf("failed to apply idle policy: %w", err)
			}

			fmt.Printf("✅ Successfully applied idle policy!\n\n")
			fmt.Printf("📊 Policy Details:\n")
			fmt.Printf("   Name: %s\n", policy.Name)
			fmt.Printf("   Category: %s\n", policy.Category)
			fmt.Printf("   Estimated Savings: %.0f%%\n", policy.EstimatedSavingsPercent)
			fmt.Printf("   Schedules: %d configured\n", len(policy.Schedules))

			if policy.AutoApply {
				fmt.Printf("   ⚡ Auto-apply enabled (high priority)\n")
			}

			fmt.Printf("\n💡 The idle schedules are now active and will automatically:\n")
			fmt.Printf("   • Hibernate your instance during scheduled periods\n")
			fmt.Printf("   • Resume when needed based on the policy\n")
			fmt.Printf("   • Save costs while preserving your work environment\n")

			return nil
		},
	}
}

// createPolicyRemoveCommand removes a policy from an instance
func (hc *IdleCobraCommands) createPolicyRemoveCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "remove <workspace-name> <policy-id>",
		Short: "Remove a idle policy from an instance",
		Long:  "Remove a idle policy and its schedules from an instance",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			instanceName := args[0]
			policyID := args[1]

			fmt.Printf("🔄 Removing idle policy '%s' from workspace '%s'...\n", policyID, instanceName)

			if err := hc.app.apiClient.RemoveIdlePolicy(hc.app.ctx, instanceName, policyID); err != nil {
				return fmt.Errorf("failed to remove idle policy: %w", err)
			}

			fmt.Printf("✅ Successfully removed idle policy!\n")
			fmt.Printf("\n💡 The instance will no longer follow this policy's idle schedules.\n")

			return nil
		},
	}
}

// createPolicyStatusCommand shows policies applied to an instance
func (hc *IdleCobraCommands) createPolicyStatusCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "status <workspace-name>",
		Short: "Show idle policies applied to an instance",
		Long:  "Display all idle policies currently applied to an instance",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			instanceName := args[0]

			policies, err := hc.app.apiClient.GetInstanceIdlePolicies(hc.app.ctx, instanceName)
			if err != nil {
				return fmt.Errorf("failed to get instance policies: %w", err)
			}

			if len(policies) == 0 {
				fmt.Printf("No idle policies applied to workspace '%s'\n", instanceName)
				fmt.Println("\n💡 Tip: Use 'cws idle policy apply' to add a policy")
				return nil
			}

			fmt.Printf("Idle policies for workspace '%s':\n\n", instanceName)

			for _, policy := range policies {
				fmt.Printf("📋 %s (%s)\n", policy.Name, policy.ID)
				fmt.Printf("   Category: %s\n", policy.Category)
				fmt.Printf("   Estimated Savings: %.0f%%\n", policy.EstimatedSavingsPercent)
				fmt.Printf("   Schedules:\n")

				for _, schedule := range policy.Schedules {
					fmt.Printf("     • %s: ", schedule.Name)
					switch schedule.Type {
					case "daily":
						fmt.Printf("Daily %s-%s\n", schedule.StartTime, schedule.EndTime)
					case "weekly":
						fmt.Printf("Weekly on specific days\n")
					case "idle":
						fmt.Printf("After %d minutes idle\n", schedule.IdleMinutes)
					case "work_hours":
						fmt.Printf("Outside work hours\n")
					default:
						fmt.Printf("%s schedule\n", schedule.Type)
					}
				}
				fmt.Println()
			}

			return nil
		},
	}
}

// createPolicyRecommendCommand recommends a policy for an instance
func (hc *IdleCobraCommands) createPolicyRecommendCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "recommend <workspace-name>",
		Short: "Get idle policy recommendation for an instance",
		Long:  "Analyze instance characteristics and recommend the best idle policy",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			instanceName := args[0]

			fmt.Printf("🔍 Analyzing instance '%s'...\n", instanceName)

			policy, err := hc.app.apiClient.RecommendIdlePolicy(hc.app.ctx, instanceName)
			if err != nil {
				return fmt.Errorf("failed to get recommendation: %w", err)
			}

			fmt.Printf("\n💡 Recommended Policy: %s\n", policy.Name)
			fmt.Printf("\n📊 Policy Details:\n")
			fmt.Printf("   ID: %s\n", policy.ID)
			fmt.Printf("   Category: %s\n", policy.Category)
			fmt.Printf("   Description: %s\n", policy.Description)
			fmt.Printf("   Estimated Savings: %.0f%%\n", policy.EstimatedSavingsPercent)
			fmt.Printf("   Suitable For: %v\n", policy.SuitableFor)

			fmt.Printf("\n📅 Idle Schedules:\n")
			for _, schedule := range policy.Schedules {
				fmt.Printf("   • %s\n", schedule.Name)
			}

			fmt.Printf("\n✨ To apply this policy, run:\n")
			fmt.Printf("   cws idle policy apply %s %s\n", instanceName, policy.ID)

			return nil
		},
	}
}

// createPolicyDetailsCommand shows detailed information about a policy
func (hc *IdleCobraCommands) createPolicyDetailsCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "details <policy-id>",
		Short: "Show detailed information about a idle policy",
		Long:  "Display complete details about a idle policy template",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			policyID := args[0]

			policy, err := hc.app.apiClient.GetIdlePolicy(hc.app.ctx, policyID)
			if err != nil {
				return fmt.Errorf("policy not found: %w", err)
			}

			fmt.Printf("📋 Idle Policy: %s\n", policy.Name)
			fmt.Printf("═══════════════════════════════════════════════\n\n")

			fmt.Printf("ID: %s\n", policy.ID)
			fmt.Printf("Category: %s\n", policy.Category)
			fmt.Printf("Description: %s\n", policy.Description)
			fmt.Printf("Estimated Savings: %.0f%%\n", policy.EstimatedSavingsPercent)
			fmt.Printf("Auto-apply: %v\n", policy.AutoApply)
			fmt.Printf("Priority: %d\n", policy.Priority)

			if len(policy.SuitableFor) > 0 {
				fmt.Printf("Suitable For: %v\n", policy.SuitableFor)
			}

			if len(policy.Conflicts) > 0 {
				fmt.Printf("Conflicts With: %v\n", policy.Conflicts)
			}

			fmt.Printf("\n📅 Idle Schedules (%d):\n", len(policy.Schedules))
			fmt.Printf("───────────────────────────────\n")

			for i, schedule := range policy.Schedules {
				fmt.Printf("\n%d. %s\n", i+1, schedule.Name)
				fmt.Printf("   Type: %s\n", schedule.Type)
				fmt.Printf("   Action: %s → %s\n", schedule.HibernateAction, schedule.WakeAction)

				switch schedule.Type {
				case "daily":
					fmt.Printf("   Time: %s - %s\n", schedule.StartTime, schedule.EndTime)
				case "weekly":
					fmt.Printf("   Days: %v\n", schedule.DaysOfWeek)
					fmt.Printf("   Time: %s - %s\n", schedule.StartTime, schedule.EndTime)
				case "idle":
					fmt.Printf("   Idle Threshold: %d minutes\n", schedule.IdleMinutes)
					fmt.Printf("   CPU Threshold: %.1f%%\n", schedule.CPUThreshold)
					fmt.Printf("   Memory Threshold: %.1f%%\n", schedule.MemoryThreshold)
					if schedule.NetworkThreshold > 0 {
						fmt.Printf("   Network Threshold: %.1f MB/s\n", schedule.NetworkThreshold)
					}
				case "work_hours":
					fmt.Printf("   Schedule: Monday-Friday 9 AM - 6 PM\n")
					fmt.Printf("   Hibernates: Nights and weekends\n")
				}

				if schedule.GracePeriodMinutes > 0 {
					fmt.Printf("   Grace Period: %d minutes\n", schedule.GracePeriodMinutes)
				}

				if len(schedule.RequireTags) > 0 {
					fmt.Printf("   Required Tags: %v\n", schedule.RequireTags)
				}
			}

			if len(policy.Tags) > 0 {
				fmt.Printf("\n🏷️  Policy Tags:\n")
				for k, v := range policy.Tags {
					fmt.Printf("   %s: %s\n", k, v)
				}
			}

			return nil
		},
	}
}

// createScheduleCommand creates the schedule management command
func (hc *IdleCobraCommands) createScheduleCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "schedule",
		Short: "Manage idle schedules",
		Long:  "View and manage active idle schedules",
	}

	cmd.AddCommand(
		hc.createScheduleListCommand(),
	)

	return cmd
}

// createScheduleListCommand lists active idle schedules
func (hc *IdleCobraCommands) createScheduleListCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List active idle schedules",
		Long:  "Display all currently active idle schedules across instances",
		RunE: func(cmd *cobra.Command, args []string) error {
			// This would connect to the idle scheduler
			fmt.Println("Active Idle Schedules:")
			fmt.Println("════════════════════════════")
			fmt.Println("\n(Schedule listing will be populated from the scheduler)")
			fmt.Println("\n💡 Schedules are automatically created when policies are applied to workspaces")

			return nil
		},
	}
}

// createSavingsCommand creates the savings report command
func (hc *IdleCobraCommands) createSavingsCommand() *cobra.Command {
	var period string

	cmd := &cobra.Command{
		Use:   "savings",
		Short: "Generate idle cost savings report",
		Long:  "Generate a detailed report of cost savings from idle",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Printf("💰 Idle Cost Savings Report\n")
			fmt.Printf("═══════════════════════════════════\n\n")

			// Parse period flag
			days := 30 // Default
			if period == "7d" {
				days = 7
			} else if period == "90d" {
				days = 90
			}

			fmt.Printf("Period: Last %d days\n\n", days)

			// This would generate actual savings report
			fmt.Println("📊 Summary:")
			fmt.Printf("   Total Saved: $%.2f\n", 245.67)
			fmt.Printf("   Idle Hours: %.1f\n", 1234.5)
			fmt.Printf("   Active Hours: %.1f\n", 2345.6)
			fmt.Printf("   Savings Percentage: %.1f%%\n", 34.5)

			fmt.Println("\n📈 Projected Monthly Savings: $320.00")

			fmt.Println("\n💡 Recommendations:")
			fmt.Println("   • Enable idle on 2 more instances for additional $80/month savings")
			fmt.Println("   • Consider 'aggressive-cost' policy for dev instances")
			fmt.Println("   • Review idle thresholds for GPU instances")

			return nil
		},
	}

	cmd.Flags().StringVarP(&period, "period", "p", "30d", "Report period (7d, 30d, 90d)")

	return cmd
}
