package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"text/tabwriter"
	"time"

	"github.com/scttfrdmn/prism/pkg/sleepwake"
	"github.com/spf13/cobra"
)

// SleepWakeCobraCommands manages sleep/wake CLI commands
type SleepWakeCobraCommands struct {
	app *App
}

// NewSleepWakeCobraCommands creates a new sleep/wake command manager
func NewSleepWakeCobraCommands(app *App) *SleepWakeCobraCommands {
	return &SleepWakeCobraCommands{app: app}
}

// CreateSleepWakeCommand creates the sleep-wake command group
func (c *SleepWakeCobraCommands) CreateSleepWakeCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "sleep-wake",
		Short: "Manage automatic instance hibernation on system sleep",
		Long: `Configure and monitor automatic instance hibernation when your computer goes to sleep.

This feature helps prevent cost waste by automatically hibernating running instances
when you close your laptop or put your workstation to sleep. Instances can optionally
be resumed automatically when your system wakes up.

The system integrates with idle detection to avoid interrupting active workloads.

Examples:
  prism admin sleep-wake status           # Show current configuration and statistics
  prism admin sleep-wake enable           # Enable sleep/wake monitoring
  prism admin sleep-wake disable          # Disable sleep/wake monitoring
  prism admin sleep-wake configure        # Configure hibernation behavior`,
	}

	// Add subcommands
	cmd.AddCommand(c.createStatusCommand())
	cmd.AddCommand(c.createEnableCommand())
	cmd.AddCommand(c.createDisableCommand())
	cmd.AddCommand(c.createConfigureCommand())

	return cmd
}

// createStatusCommand creates the status subcommand
func (c *SleepWakeCobraCommands) createStatusCommand() *cobra.Command {
	var jsonOutput bool

	cmd := &cobra.Command{
		Use:   "status",
		Short: "Show sleep/wake monitoring status",
		Long:  `Display the current status, configuration, and statistics of the sleep/wake monitoring system.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Get status from daemon API
			status, err := c.getSleepWakeStatus()
			if err != nil {
				return fmt.Errorf("failed to get sleep/wake status: %w", err)
			}

			if jsonOutput {
				encoder := json.NewEncoder(os.Stdout)
				encoder.SetIndent("", "  ")
				return encoder.Encode(status)
			}

			c.printStatus(*status)
			return nil
		},
	}

	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output in JSON format")

	return cmd
}

// getSleepWakeStatus retrieves the current sleep/wake status from the daemon
func (c *SleepWakeCobraCommands) getSleepWakeStatus() (*sleepwake.Status, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	url := c.app.config.Daemon.URL + "/api/v1/sleep-wake/status"
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
	}

	var status sleepwake.Status
	if err := json.NewDecoder(resp.Body).Decode(&status); err != nil {
		return nil, err
	}

	return &status, nil
}

// createEnableCommand creates the enable subcommand
func (c *SleepWakeCobraCommands) createEnableCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "enable",
		Short: "Enable sleep/wake monitoring",
		Long: `Enable automatic instance hibernation when the system goes to sleep.

This will start monitoring system power events and automatically hibernate
running instances (based on configured hibernation mode) when your computer
enters sleep mode.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			config := sleepwake.Config{
				Enabled: true,
			}

			return c.updateConfig(config)
		},
	}
}

// createDisableCommand creates the disable subcommand
func (c *SleepWakeCobraCommands) createDisableCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "disable",
		Short: "Disable sleep/wake monitoring",
		Long: `Disable automatic instance hibernation on system sleep.

This will stop monitoring system power events. Your instances will continue
running when your computer goes to sleep.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			config := sleepwake.Config{
				Enabled: false,
			}

			return c.updateConfig(config)
		},
	}
}

// createConfigureCommand creates the configure subcommand
func (c *SleepWakeCobraCommands) createConfigureCommand() *cobra.Command {
	var (
		hibernateOnSleep  bool
		resumeOnWake      bool
		hibernationMode   string
		gracePeriod       int
		idleCheckTimeout  int
		excludedInstances []string
	)

	cmd := &cobra.Command{
		Use:   "configure",
		Short: "Configure sleep/wake behavior",
		Long: `Configure hibernation behavior and policies.

Hibernation Modes:
  idle_only    - Only hibernate idle instances (RECOMMENDED)
  all          - Hibernate all instances except excluded ones
  manual_only  - No automatic hibernation

Examples:
  # Safe default: only hibernate idle instances
  prism admin sleep-wake configure --mode idle_only

  # Aggressive: hibernate all except critical instances
  prism admin sleep-wake configure --mode all --exclude prod-database

  # Conservative: no automatic hibernation
  prism admin sleep-wake configure --mode manual_only

  # Enable automatic resume on wake
  prism admin sleep-wake configure --resume-on-wake`,
		RunE: func(cmd *cobra.Command, args []string) error {
			config := sleepwake.Config{
				Enabled:           true,
				HibernateOnSleep:  hibernateOnSleep,
				ResumeOnWake:      resumeOnWake,
				HibernationMode:   sleepwake.HibernationMode(hibernationMode),
				GracePeriod:       time.Duration(gracePeriod) * time.Second,
				IdleCheckTimeout:  time.Duration(idleCheckTimeout) * time.Second,
				ExcludedInstances: excludedInstances,
			}

			if err := c.updateConfig(config); err != nil {
				return err
			}

			fmt.Println("✅ Sleep/wake configuration updated successfully")
			return nil
		},
	}

	cmd.Flags().BoolVar(&hibernateOnSleep, "hibernate-on-sleep", true, "Hibernate instances when system sleeps")
	cmd.Flags().BoolVar(&resumeOnWake, "resume-on-wake", false, "Resume instances when system wakes (default: manual resume)")
	cmd.Flags().StringVar(&hibernationMode, "mode", "idle_only", "Hibernation mode: idle_only, all, manual_only")
	cmd.Flags().IntVar(&gracePeriod, "grace-period", 30, "Grace period in seconds before hibernation")
	cmd.Flags().IntVar(&idleCheckTimeout, "idle-timeout", 10, "Idle check timeout in seconds")
	cmd.Flags().StringSliceVar(&excludedInstances, "exclude", []string{}, "Instance names to exclude from auto-hibernation")

	return cmd
}

// updateConfig sends configuration update to daemon
func (c *SleepWakeCobraCommands) updateConfig(config sleepwake.Config) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	configJSON, err := json.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal configuration: %w", err)
	}

	url := c.app.config.Daemon.URL + "/api/v1/sleep-wake/configure"
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(configJSON))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to update configuration: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// printStatus displays the sleep/wake status in a user-friendly format
func (c *SleepWakeCobraCommands) printStatus(status sleepwake.Status) {
	fmt.Println("🌙 Sleep/Wake Monitoring Status")
	fmt.Println("=" + string(make([]byte, 50)))
	fmt.Println()

	// Configuration
	fmt.Println("Configuration:")
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
	fmt.Fprintf(w, "  Enabled:\t%v\n", status.Enabled)
	fmt.Fprintf(w, "  Running:\t%v\n", status.Running)
	fmt.Fprintf(w, "  Platform:\t%s\n", status.Platform)
	fmt.Fprintf(w, "  Hibernate on Sleep:\t%v\n", status.HibernateOnSleep)
	fmt.Fprintf(w, "  Hibernation Mode:\t%s\n", status.HibernationMode)
	fmt.Fprintf(w, "  Idle Check Timeout:\t%s\n", status.IdleCheckTimeout)
	fmt.Fprintf(w, "  Resume on Wake:\t%v\n", status.ResumeOnWake)
	fmt.Fprintf(w, "  Grace Period:\t%s\n", status.GracePeriod)
	_ = w.Flush() // stdout table; printStatus has no error return and a failed terminal write is not actionable

	if len(status.ExcludedInstances) > 0 {
		fmt.Println("\nExcluded Instances:")
		for _, instance := range status.ExcludedInstances {
			fmt.Printf("  • %s\n", instance)
		}
	}

	// Statistics
	fmt.Println("\nStatistics:")
	w = tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
	if !status.Stats.LastSleepTime.IsZero() {
		fmt.Fprintf(w, "  Last Sleep:\t%s\n", status.Stats.LastSleepTime.Format(time.RFC3339))
	}
	if !status.Stats.LastWakeTime.IsZero() {
		fmt.Fprintf(w, "  Last Wake:\t%s\n", status.Stats.LastWakeTime.Format(time.RFC3339))
	}
	fmt.Fprintf(w, "  Total Sleep Events:\t%d\n", status.Stats.TotalSleepEvents)
	fmt.Fprintf(w, "  Total Wake Events:\t%d\n", status.Stats.TotalWakeEvents)
	fmt.Fprintf(w, "  Total Hibernated:\t%d\n", status.Stats.TotalHibernated)
	fmt.Fprintf(w, "  Total Resumed:\t%d\n", status.Stats.TotalResumed)
	fmt.Fprintf(w, "  Currently Tracked:\t%d\n", status.Stats.ActivelyTracked)
	_ = w.Flush() // stdout table; printStatus has no error return and a failed terminal write is not actionable

	// Currently hibernated instances
	if len(status.HibernatedInstances) > 0 {
		fmt.Println("\nCurrently Hibernated Instances:")
		for name, timestamp := range status.HibernatedInstances {
			hibernatedTime, _ := time.Parse(time.RFC3339, timestamp)
			duration := time.Since(hibernatedTime).Round(time.Second)
			fmt.Printf("  • %s (hibernated %s ago)\n", name, duration)
		}
	}

	fmt.Println()

	// Helpful information based on configuration
	if !status.Enabled {
		fmt.Println("💡 Tip: Enable sleep/wake monitoring with 'prism admin sleep-wake enable'")
	} else if status.HibernationMode == "manual_only" {
		fmt.Println("ℹ️  Manual hibernation mode: Instances will not be automatically hibernated")
	} else if status.HibernationMode == "idle_only" {
		fmt.Println("✅ Safe mode: Only idle instances will be hibernated on system sleep")
	} else if status.HibernationMode == "all" {
		fmt.Println("⚠️  Aggressive mode: All instances (except excluded) will be hibernated on sleep")
		if len(status.ExcludedInstances) == 0 {
			fmt.Println("   Consider excluding critical instances with --exclude flag")
		}
	}
}
