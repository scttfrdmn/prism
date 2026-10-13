package cli

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/scttfrdmn/prism/pkg/api/client"
	"github.com/scttfrdmn/prism/pkg/throttle"
	"github.com/spf13/cobra"
)

// ThrottlingCobraCommands provides Cobra commands for advanced launch throttling management
type ThrottlingCobraCommands struct {
	app *App
}

// NewThrottlingCobraCommands creates a new throttling commands instance
func NewThrottlingCobraCommands(app *App) *ThrottlingCobraCommands {
	return &ThrottlingCobraCommands{app: app}
}

// CreateThrottlingCommand creates the throttling command group
func (t *ThrottlingCobraCommands) CreateThrottlingCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "throttling",
		Short: "Manage advanced launch throttling (v0.6.0)",
		Long: `Configure and monitor advanced launch throttling for cost control and resource management.

Advanced throttling provides multi-scope rate limiting (global, per-user, per-project) with
budget-aware throttling and project-specific overrides for institutional cost control.

Key Features:
  • Multi-scope throttling: Global, per-user, and per-project limits
  • Budget-aware throttling: More aggressive throttling near budget limits
  • Project overrides: Custom limits for specific projects
  • Token bucket algorithm: Allows controlled burst capacity

Examples:
  prism admin throttling status                     # View global throttling status
  prism admin throttling status --scope user:alice  # View user-specific status
  prism admin throttling configure --enabled        # Enable throttling
  prism admin throttling remaining --scope project:ml-research  # Check tokens
  prism admin throttling set-override proj-123 --max-launches 20 --time-window 2h`,
	}

	cmd.AddCommand(t.createStatusCommand())
	cmd.AddCommand(t.createConfigureCommand())
	cmd.AddCommand(t.createEnableCommand())
	cmd.AddCommand(t.createDisableCommand())
	cmd.AddCommand(t.createRemainingCommand())
	cmd.AddCommand(t.createSetOverrideCommand())
	cmd.AddCommand(t.createRemoveOverrideCommand())
	cmd.AddCommand(t.createListOverridesCommand())

	return cmd
}

// createStatusCommand creates the status subcommand
func (t *ThrottlingCobraCommands) createStatusCommand() *cobra.Command {
	var scope string

	cmd := &cobra.Command{
		Use:   "status",
		Short: "Show throttling status",
		Long: `Display the current throttling configuration and status for a scope.

Scopes:
  • global (default): System-wide throttling status
  • user:<username>: Per-user throttling status
  • project:<project-id>: Per-project throttling status

Examples:
  prism admin throttling status
  prism admin throttling status --scope user:alice
  prism admin throttling status --scope project:ml-research`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return t.handleStatus(scope)
		},
	}

	cmd.Flags().StringVar(&scope, "scope", "global", "Throttling scope (global, user:<name>, project:<id>)")

	return cmd
}

// createConfigureCommand creates the configure subcommand
func (t *ThrottlingCobraCommands) createConfigureCommand() *cobra.Command {
	var (
		enabled          bool
		setEnabled       bool
		maxLaunches      int
		timeWindow       string
		burstSize        int
		perUser          bool
		setPerUser       bool
		perProject       bool
		setPerProject    bool
		budgetAware      bool
		setBudgetAware   bool
		budgetThreshold  float64
		budgetMultiplier float64
		queueMode        string
	)

	cmd := &cobra.Command{
		Use:   "configure",
		Short: "Configure throttling settings",
		Long: `Update advanced throttling configuration.

Configuration Options:
  --enabled:            Enable/disable throttling
  --max-launches:       Maximum launches in time window (e.g., 10)
  --time-window:        Time window duration (e.g., "1h", "24h")
  --burst-size:         Burst capacity (0 = same as max-launches)
  --per-user:           Enable per-user throttling
  --per-project:        Enable per-project throttling
  --budget-aware:       Enable budget-aware throttling
  --budget-threshold:   Budget threshold for throttling (0.0-1.0, e.g., 0.8 = 80%)
  --budget-multiplier:  Rate reduction factor near limit (e.g., 0.5 = half rate)
  --queue-mode:         Behavior mode: "reject", "queue", or "warn"

Examples:
  prism admin throttling configure --enabled --max-launches 10 --time-window 1h
  prism admin throttling configure --per-user --per-project
  prism admin throttling configure --budget-aware --budget-threshold 0.8 --budget-multiplier 0.5`,
		RunE: func(cmd *cobra.Command, args []string) error {
			config := throttle.Config{}
			hasChanges := false

			if setEnabled {
				config.Enabled = enabled
				hasChanges = true
			}
			if maxLaunches > 0 {
				config.MaxLaunches = maxLaunches
				hasChanges = true
			}
			if timeWindow != "" {
				config.TimeWindow = timeWindow
				hasChanges = true
			}
			if burstSize > 0 {
				config.BurstSize = burstSize
				hasChanges = true
			}
			if setPerUser {
				config.PerUser = perUser
				hasChanges = true
			}
			if setPerProject {
				config.PerProject = perProject
				hasChanges = true
			}
			if setBudgetAware {
				config.BudgetAware = budgetAware
				hasChanges = true
			}
			if budgetThreshold > 0 {
				config.BudgetThreshold = budgetThreshold
				hasChanges = true
			}
			if budgetMultiplier > 0 {
				config.BudgetMultiplier = budgetMultiplier
				hasChanges = true
			}
			if queueMode != "" {
				config.QueueMode = queueMode
				hasChanges = true
			}

			if !hasChanges {
				return fmt.Errorf("no configuration changes specified")
			}

			return t.handleConfigure(config)
		},
	}

	cmd.Flags().BoolVar(&enabled, "enabled", true, "Enable throttling")
	cmd.Flags().BoolVar(&setEnabled, "set-enabled", false, "Set enabled flag")
	cmd.Flags().IntVar(&maxLaunches, "max-launches", 0, "Maximum launches in time window")
	cmd.Flags().StringVar(&timeWindow, "time-window", "", "Time window (e.g., 1h, 24h)")
	cmd.Flags().IntVar(&burstSize, "burst-size", 0, "Burst capacity")
	cmd.Flags().BoolVar(&perUser, "per-user", true, "Enable per-user throttling")
	cmd.Flags().BoolVar(&setPerUser, "set-per-user", false, "Set per-user flag")
	cmd.Flags().BoolVar(&perProject, "per-project", true, "Enable per-project throttling")
	cmd.Flags().BoolVar(&setPerProject, "set-per-project", false, "Set per-project flag")
	cmd.Flags().BoolVar(&budgetAware, "budget-aware", true, "Enable budget-aware throttling")
	cmd.Flags().BoolVar(&setBudgetAware, "set-budget-aware", false, "Set budget-aware flag")
	cmd.Flags().Float64Var(&budgetThreshold, "budget-threshold", 0, "Budget threshold (0.0-1.0)")
	cmd.Flags().Float64Var(&budgetMultiplier, "budget-multiplier", 0, "Budget multiplier (e.g., 0.5)")
	cmd.Flags().StringVar(&queueMode, "queue-mode", "", "Queue mode: reject, queue, warn")

	return cmd
}

// createEnableCommand creates the enable subcommand
func (t *ThrottlingCobraCommands) createEnableCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "enable",
		Short: "Enable throttling",
		Long:  `Enable advanced launch throttling with current configuration.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			config := throttle.Config{Enabled: true}
			return t.handleConfigure(config)
		},
	}
}

// createDisableCommand creates the disable subcommand
func (t *ThrottlingCobraCommands) createDisableCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "disable",
		Short: "Disable throttling",
		Long:  `Disable advanced launch throttling (all launches allowed).`,
		RunE: func(cmd *cobra.Command, args []string) error {
			config := throttle.Config{Enabled: false}
			return t.handleConfigure(config)
		},
	}
}

// createRemainingCommand creates the remaining subcommand
func (t *ThrottlingCobraCommands) createRemainingCommand() *cobra.Command {
	var scope string

	cmd := &cobra.Command{
		Use:   "remaining",
		Short: "Show remaining launch tokens",
		Long: `Display remaining launch tokens for a scope.

This shows how many launches are currently available before throttling occurs.

Examples:
  prism admin throttling remaining
  prism admin throttling remaining --scope user:alice
  prism admin throttling remaining --scope project:ml-research`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return t.handleRemaining(scope)
		},
	}

	cmd.Flags().StringVar(&scope, "scope", "global", "Throttling scope (global, user:<name>, project:<id>)")

	return cmd
}

// createSetOverrideCommand creates the set-override subcommand
func (t *ThrottlingCobraCommands) createSetOverrideCommand() *cobra.Command {
	var (
		maxLaunches int
		timeWindow  string
		reason      string
	)

	cmd := &cobra.Command{
		Use:   "set-override <project-id>",
		Short: "Set project throttling override",
		Long: `Set a project-specific throttling override.

This allows administrators to set custom throttling limits for specific projects,
overriding the global configuration.

Examples:
  prism admin throttling set-override proj-123 --max-launches 20 --time-window 2h
  prism admin throttling set-override ml-research --max-launches 50 --time-window 24h --reason "High-priority research"`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			projectID := args[0]

			if maxLaunches == 0 || timeWindow == "" {
				return fmt.Errorf("both --max-launches and --time-window are required")
			}

			override := throttle.Override{
				ProjectID:   projectID,
				MaxLaunches: maxLaunches,
				TimeWindow:  timeWindow,
				Reason:      reason,
				CreatedAt:   time.Now(),
			}

			return t.handleSetOverride(projectID, override)
		},
	}

	cmd.Flags().IntVar(&maxLaunches, "max-launches", 0, "Maximum launches in time window")
	cmd.Flags().StringVar(&timeWindow, "time-window", "", "Time window (e.g., 1h, 24h)")
	cmd.Flags().StringVar(&reason, "reason", "", "Reason for override")
	_ = cmd.MarkFlagRequired("max-launches") // only errors if the flag is undefined; it is defined above
	_ = cmd.MarkFlagRequired("time-window")  // only errors if the flag is undefined; it is defined above

	return cmd
}

// createRemoveOverrideCommand creates the remove-override subcommand
func (t *ThrottlingCobraCommands) createRemoveOverrideCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "remove-override <project-id>",
		Short: "Remove project throttling override",
		Long: `Remove a project-specific throttling override, reverting to global configuration.

Examples:
  prism admin throttling remove-override proj-123
  prism admin throttling remove-override ml-research`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			projectID := args[0]
			return t.handleRemoveOverride(projectID)
		},
	}
}

// createListOverridesCommand creates the list-overrides subcommand
func (t *ThrottlingCobraCommands) createListOverridesCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "list-overrides",
		Short: "List all project overrides",
		Long:  `Display all project-specific throttling overrides.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return t.handleListOverrides()
		},
	}
}

// handleStatus displays the current throttling status
func (t *ThrottlingCobraCommands) handleStatus(scope string) error {
	apiClient := client.NewClientWithOptions(t.app.config.Daemon.URL, client.Options{})
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	status, err := apiClient.GetThrottlingStatus(ctx, scope)
	if err != nil {
		return fmt.Errorf("failed to get throttling status: %w", err)
	}

	// Display status
	fmt.Println("🚦 Advanced Launch Throttling Status")
	fmt.Println(strings.Repeat("=", 70))
	fmt.Printf("Scope:              %s\n", status.Scope)
	fmt.Printf("Enabled:            %v\n", status.Enabled)
	fmt.Printf("Max Launches:       %d per %s\n", status.MaxLaunches, status.TimeWindow)
	fmt.Printf("Current Tokens:     %.2f\n", status.CurrentTokens)
	fmt.Printf("Launches in Window: %d\n", status.LaunchesInWindow)

	if !status.NextTokenRefill.IsZero() {
		fmt.Printf("Next Token Refill:  %s\n", status.NextTokenRefill.Format(time.RFC3339))
		fmt.Printf("Time Until Refill:  %s\n", status.TimeUntilRefill.String())
	}

	if status.BudgetAdjusted {
		fmt.Printf("\n💰 Budget-Aware Throttling:\n")
		fmt.Printf("Configured Rate:    %.2f launches/hour\n", status.ConfiguredRate)
		fmt.Printf("Effective Rate:     %.2f launches/hour (adjusted)\n", status.EffectiveRate)
	}

	fmt.Printf("\n📊 Statistics:\n")
	fmt.Printf("Total Allowed:      %d launches\n", status.AllowedLaunches)
	fmt.Printf("Total Throttled:    %d launches\n", status.TotalThrottled)
	if status.AllowedLaunches+status.TotalThrottled > 0 {
		fmt.Printf("Success Rate:       %.1f%%\n", status.SuccessRate*100)
	}

	if !status.LastLaunchTime.IsZero() {
		fmt.Printf("Last Launch:        %s\n", status.LastLaunchTime.Format(time.RFC3339))
	}

	return nil
}

// handleConfigure updates the throttling configuration
func (t *ThrottlingCobraCommands) handleConfigure(config throttle.Config) error {
	apiClient := client.NewClientWithOptions(t.app.config.Daemon.URL, client.Options{})
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := apiClient.ConfigureThrottling(ctx, config); err != nil {
		return fmt.Errorf("failed to configure throttling: %w", err)
	}

	fmt.Println("✅ Throttling configuration updated successfully")

	// Show current status
	return t.handleStatus("global")
}

// handleRemaining shows remaining launch tokens
func (t *ThrottlingCobraCommands) handleRemaining(scope string) error {
	apiClient := client.NewClientWithOptions(t.app.config.Daemon.URL, client.Options{})
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	remaining, err := apiClient.GetThrottlingRemaining(ctx, scope)
	if err != nil {
		return fmt.Errorf("failed to get remaining tokens: %w", err)
	}

	fmt.Println("🎫 Remaining Launch Tokens")
	fmt.Println(strings.Repeat("=", 60))
	fmt.Printf("Scope:           %v\n", remaining["scope"])
	fmt.Printf("Enabled:         %v\n", remaining["enabled"])
	fmt.Printf("Current Tokens:  %v\n", remaining["current_tokens"])
	fmt.Printf("Max Launches:    %v per %v\n", remaining["max_launches"], remaining["time_window"])

	if nextRefill, ok := remaining["next_refill"]; ok && nextRefill != nil {
		fmt.Printf("Next Refill:     %v\n", nextRefill)
		fmt.Printf("Time Until:      %v\n", remaining["time_until_refill"])
	}

	return nil
}

// handleSetOverride sets a project-specific override
func (t *ThrottlingCobraCommands) handleSetOverride(projectID string, override throttle.Override) error {
	apiClient := client.NewClientWithOptions(t.app.config.Daemon.URL, client.Options{})
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := apiClient.SetProjectThrottleOverride(ctx, projectID, override); err != nil {
		return fmt.Errorf("failed to set project override: %w", err)
	}

	fmt.Println("✅ Project throttling override set successfully")
	fmt.Printf("Project:        %s\n", projectID)
	fmt.Printf("Max Launches:   %d per %s\n", override.MaxLaunches, override.TimeWindow)
	if override.Reason != "" {
		fmt.Printf("Reason:         %s\n", override.Reason)
	}

	return nil
}

// handleRemoveOverride removes a project-specific override
func (t *ThrottlingCobraCommands) handleRemoveOverride(projectID string) error {
	apiClient := client.NewClientWithOptions(t.app.config.Daemon.URL, client.Options{})
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := apiClient.RemoveProjectThrottleOverride(ctx, projectID); err != nil {
		return fmt.Errorf("failed to remove project override: %w", err)
	}

	fmt.Println("✅ Project throttling override removed successfully")
	fmt.Printf("Project %s now uses global throttling configuration\n", projectID)

	return nil
}

// handleListOverrides lists all project overrides
func (t *ThrottlingCobraCommands) handleListOverrides() error {
	apiClient := client.NewClientWithOptions(t.app.config.Daemon.URL, client.Options{})
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	overrides, err := apiClient.ListProjectThrottleOverrides(ctx)
	if err != nil {
		return fmt.Errorf("failed to list project overrides: %w", err)
	}

	if len(overrides) == 0 {
		fmt.Println("No project throttling overrides configured")
		return nil
	}

	fmt.Printf("📋 Project Throttling Overrides (%d)\n", len(overrides))
	fmt.Println(strings.Repeat("=", 80))

	for _, override := range overrides {
		fmt.Printf("\nProject:        %s\n", override.ProjectID)
		fmt.Printf("Max Launches:   %d per %s\n", override.MaxLaunches, override.TimeWindow)
		fmt.Printf("Created:        %s\n", override.CreatedAt.Format(time.RFC3339))
		if override.CreatedBy != "" {
			fmt.Printf("Created By:     %s\n", override.CreatedBy)
		}
		if override.Reason != "" {
			fmt.Printf("Reason:         %s\n", override.Reason)
		}
	}

	return nil
}
