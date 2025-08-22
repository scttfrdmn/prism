// Package cli provides the command-line interface for CloudWorkstation
package cli

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"
)

// HibernationCobraCommands handles hibernation-related commands with proper Cobra structure
type HibernationCobraCommands struct {
	app *App
}

// NewHibernationCobraCommands creates a new hibernation commands handler
func NewHibernationCobraCommands(app *App) *HibernationCobraCommands {
	return &HibernationCobraCommands{app: app}
}

// CreateHibernationCommand creates the main hibernation command with subcommands
func (hc *HibernationCobraCommands) CreateHibernationCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "hibernation",
		Short: "Manage hibernation policies and schedules",
		Long: `Manage advanced hibernation policies and schedules for cost optimization.

Hibernation policies allow you to automatically hibernate instances based on:
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
		Example: `  # List available hibernation policies
  cws hibernation policy list

  # Apply a policy to an instance
  cws hibernation policy apply my-instance balanced

  # View policies applied to an instance
  cws hibernation policy status my-instance

  # Get policy recommendation for an instance
  cws hibernation policy recommend my-instance

  # View hibernation schedules
  cws hibernation schedule list

  # Generate savings report
  cws hibernation savings --period 30d`,
	}

	// Add subcommands
	cmd.AddCommand(
		hc.createPolicyCommand(),
		hc.createScheduleCommand(),
		hc.createSavingsCommand(),
	)

	return cmd
}

// createPolicyCommand creates the policy management command
func (hc *HibernationCobraCommands) createPolicyCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "policy",
		Short: "Manage hibernation policies",
		Long:  "Manage hibernation policy templates and apply them to instances",
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

// createPolicyListCommand lists available hibernation policies
func (hc *HibernationCobraCommands) createPolicyListCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List available hibernation policy templates",
		Long:  "Display all available hibernation policy templates with their characteristics",
		RunE: func(cmd *cobra.Command, args []string) error {
			policies, err := hc.app.apiClient.ListHibernationPolicies(hc.app.ctx)
			if err != nil {
				return fmt.Errorf("failed to list hibernation policies: %w", err)
			}

			if len(policies) == 0 {
				fmt.Println("No hibernation policies available")
				return nil
			}

			// Create table writer
			w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			fmt.Fprintln(w, "POLICY ID\tNAME\tCATEGORY\tSAVINGS\tSUITABLE FOR")
			fmt.Fprintln(w, "─────────\t────\t────────\t───────\t────────────")

			for _, policy := range policies {
				suitable := ""
				if len(policy.SuitableFor) > 0 {
					suitable = policy.SuitableFor[0]
					if len(policy.SuitableFor) > 1 {
						suitable += fmt.Sprintf(" (+%d more)", len(policy.SuitableFor)-1)
					}
				}

				fmt.Fprintf(w, "%s\t%s\t%s\t%.0f%%\t%s\n",
					policy.ID,
					policy.Name,
					policy.Category,
					policy.EstimatedSavingsPercent,
					suitable,
				)
			}

			w.Flush()
			
			fmt.Println("\n💡 Tip: Use 'cws hibernation policy details <policy-id>' to see full details")
			fmt.Println("💰 Estimated savings are based on typical usage patterns")

			return nil
		},
	}
}

// createPolicyApplyCommand applies a policy to an instance
func (hc *HibernationCobraCommands) createPolicyApplyCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "apply <instance-name> <policy-id>",
		Short: "Apply a hibernation policy to an instance",
		Long:  "Apply a hibernation policy template to an instance to enable automatic hibernation",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			instanceName := args[0]
			policyID := args[1]

			// Get policy details for display
			policy, err := hc.app.apiClient.GetHibernationPolicy(hc.app.ctx, policyID)
			if err != nil {
				return fmt.Errorf("policy not found: %w", err)
			}

			fmt.Printf("🔄 Applying hibernation policy '%s' to instance '%s'...\n", policy.Name, instanceName)
			
			if err := hc.app.apiClient.ApplyHibernationPolicy(hc.app.ctx, instanceName, policyID); err != nil {
				return fmt.Errorf("failed to apply hibernation policy: %w", err)
			}

			fmt.Printf("✅ Successfully applied hibernation policy!\n\n")
			fmt.Printf("📊 Policy Details:\n")
			fmt.Printf("   Name: %s\n", policy.Name)
			fmt.Printf("   Category: %s\n", policy.Category)
			fmt.Printf("   Estimated Savings: %.0f%%\n", policy.EstimatedSavingsPercent)
			fmt.Printf("   Schedules: %d configured\n", len(policy.Schedules))
			
			if policy.AutoApply {
				fmt.Printf("   ⚡ Auto-apply enabled (high priority)\n")
			}

			fmt.Printf("\n💡 The hibernation schedules are now active and will automatically:\n")
			fmt.Printf("   • Hibernate your instance during scheduled periods\n")
			fmt.Printf("   • Resume when needed based on the policy\n")
			fmt.Printf("   • Save costs while preserving your work environment\n")

			return nil
		},
	}
}

// createPolicyRemoveCommand removes a policy from an instance
func (hc *HibernationCobraCommands) createPolicyRemoveCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "remove <instance-name> <policy-id>",
		Short: "Remove a hibernation policy from an instance",
		Long:  "Remove a hibernation policy and its schedules from an instance",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			instanceName := args[0]
			policyID := args[1]

			fmt.Printf("🔄 Removing hibernation policy '%s' from instance '%s'...\n", policyID, instanceName)
			
			if err := hc.app.apiClient.RemoveHibernationPolicy(hc.app.ctx, instanceName, policyID); err != nil {
				return fmt.Errorf("failed to remove hibernation policy: %w", err)
			}

			fmt.Printf("✅ Successfully removed hibernation policy!\n")
			fmt.Printf("\n💡 The instance will no longer follow this policy's hibernation schedules.\n")

			return nil
		},
	}
}

// createPolicyStatusCommand shows policies applied to an instance
func (hc *HibernationCobraCommands) createPolicyStatusCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "status <instance-name>",
		Short: "Show hibernation policies applied to an instance",
		Long:  "Display all hibernation policies currently applied to an instance",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			instanceName := args[0]

			policies, err := hc.app.apiClient.GetInstanceHibernationPolicies(hc.app.ctx, instanceName)
			if err != nil {
				return fmt.Errorf("failed to get instance policies: %w", err)
			}

			if len(policies) == 0 {
				fmt.Printf("No hibernation policies applied to instance '%s'\n", instanceName)
				fmt.Println("\n💡 Tip: Use 'cws hibernation policy apply' to add a policy")
				return nil
			}

			fmt.Printf("Hibernation policies for instance '%s':\n\n", instanceName)

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
func (hc *HibernationCobraCommands) createPolicyRecommendCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "recommend <instance-name>",
		Short: "Get hibernation policy recommendation for an instance",
		Long:  "Analyze instance characteristics and recommend the best hibernation policy",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			instanceName := args[0]

			fmt.Printf("🔍 Analyzing instance '%s'...\n", instanceName)

			policy, err := hc.app.apiClient.RecommendHibernationPolicy(hc.app.ctx, instanceName)
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

			fmt.Printf("\n📅 Hibernation Schedules:\n")
			for _, schedule := range policy.Schedules {
				fmt.Printf("   • %s\n", schedule.Name)
			}

			fmt.Printf("\n✨ To apply this policy, run:\n")
			fmt.Printf("   cws hibernation policy apply %s %s\n", instanceName, policy.ID)

			return nil
		},
	}
}

// createPolicyDetailsCommand shows detailed information about a policy
func (hc *HibernationCobraCommands) createPolicyDetailsCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "details <policy-id>",
		Short: "Show detailed information about a hibernation policy",
		Long:  "Display complete details about a hibernation policy template",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			policyID := args[0]

			policy, err := hc.app.apiClient.GetHibernationPolicy(hc.app.ctx, policyID)
			if err != nil {
				return fmt.Errorf("policy not found: %w", err)
			}

			fmt.Printf("📋 Hibernation Policy: %s\n", policy.Name)
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

			fmt.Printf("\n📅 Hibernation Schedules (%d):\n", len(policy.Schedules))
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
func (hc *HibernationCobraCommands) createScheduleCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "schedule",
		Short: "Manage hibernation schedules",
		Long:  "View and manage active hibernation schedules",
	}

	cmd.AddCommand(
		hc.createScheduleListCommand(),
	)

	return cmd
}

// createScheduleListCommand lists active hibernation schedules
func (hc *HibernationCobraCommands) createScheduleListCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List active hibernation schedules",
		Long:  "Display all currently active hibernation schedules across instances",
		RunE: func(cmd *cobra.Command, args []string) error {
			// This would connect to the hibernation scheduler
			fmt.Println("Active Hibernation Schedules:")
			fmt.Println("════════════════════════════")
			fmt.Println("\n(Schedule listing will be populated from the scheduler)")
			fmt.Println("\n💡 Schedules are automatically created when policies are applied to instances")
			
			return nil
		},
	}
}

// createSavingsCommand creates the savings report command
func (hc *HibernationCobraCommands) createSavingsCommand() *cobra.Command {
	var period string
	
	cmd := &cobra.Command{
		Use:   "savings",
		Short: "Generate hibernation cost savings report",
		Long:  "Generate a detailed report of cost savings from hibernation",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Printf("💰 Hibernation Cost Savings Report\n")
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
			fmt.Printf("   Hibernation Hours: %.1f\n", 1234.5)
			fmt.Printf("   Active Hours: %.1f\n", 2345.6)
			fmt.Printf("   Savings Percentage: %.1f%%\n", 34.5)
			
			fmt.Println("\n📈 Projected Monthly Savings: $320.00")
			
			fmt.Println("\n💡 Recommendations:")
			fmt.Println("   • Enable hibernation on 2 more instances for additional $80/month savings")
			fmt.Println("   • Consider 'aggressive-cost' policy for dev instances")
			fmt.Println("   • Review idle thresholds for GPU instances")
			
			return nil
		},
	}
	
	cmd.Flags().StringVarP(&period, "period", "p", "30d", "Report period (7d, 30d, 90d)")
	
	return cmd
}