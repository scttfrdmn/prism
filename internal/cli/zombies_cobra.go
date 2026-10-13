package cli

import (
	"context"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

// ZombiesCobraCommands provides zombie resource detection commands
type ZombiesCobraCommands struct {
	app *App
}

// NewZombiesCobraCommands creates a new zombies command handler
func NewZombiesCobraCommands(app *App) *ZombiesCobraCommands {
	return &ZombiesCobraCommands{app: app}
}

// CreateZombiesCommand creates the zombies command group
func (z *ZombiesCobraCommands) CreateZombiesCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "zombies",
		Short: "Detect and cleanup zombie resources",
		Long: `Zombie resource detection identifies AWS resources (EC2 instances, EBS volumes) without Prism management tags.

These "zombie" resources may be:
- Manually launched instances outside of Prism
- Resources from failed cleanup operations
- Orphaned resources after state file corruption

Zombie detection helps prevent runaway AWS costs by identifying resources you may have forgotten about.`,
		Example: `  # Scan for zombie resources
  prism admin zombies scan

  # Scan with detailed output
  prism admin zombies scan --verbose

  # Cleanup zombies (dry-run by default)
  prism admin zombies cleanup

  # Actually terminate zombies (requires confirmation)
  prism admin zombies cleanup --force

  # Show cost estimate only
  prism admin zombies cost-estimate`,
	}

	cmd.AddCommand(z.createScanCommand())
	cmd.AddCommand(z.createCleanupCommand())
	cmd.AddCommand(z.createCostEstimateCommand())

	return cmd
}

// createScanCommand creates the scan subcommand
func (z *ZombiesCobraCommands) createScanCommand() *cobra.Command {
	var verbose bool

	cmd := &cobra.Command{
		Use:   "scan",
		Short: "Scan for zombie resources",
		Long: `Scans the current AWS region for EC2 instances and EBS volumes without Prism management tags.

Zombie resources are identified by the absence of either:
- prism:managed=true tag (current standard)
- Prism=true tag (legacy compatibility)`,
		Example: `  # Basic scan
  prism admin zombies scan

  # Detailed output
  prism admin zombies scan --verbose`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return z.runScan(cmd.Context(), verbose)
		},
	}

	cmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "Show detailed information for each resource")

	return cmd
}

// createCleanupCommand creates the cleanup subcommand
func (z *ZombiesCobraCommands) createCleanupCommand() *cobra.Command {
	var force bool
	var dryRun bool

	cmd := &cobra.Command{
		Use:   "cleanup",
		Short: "Cleanup zombie resources",
		Long: `Terminates EC2 instances and deletes EBS volumes identified as zombies.

By default, this command runs in DRY-RUN mode and will not make any changes.
Use --force to actually terminate resources (requires confirmation).

⚠️  WARNING: This action is IRREVERSIBLE. Terminated instances cannot be recovered.`,
		Example: `  # Dry run (safe, no changes)
  prism admin zombies cleanup

  # Actually cleanup (requires confirmation)
  prism admin zombies cleanup --force

  # Skip confirmation (automation)
  prism admin zombies cleanup --force --no-dry-run`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return z.runCleanup(cmd.Context(), force, dryRun)
		},
	}

	cmd.Flags().BoolVarP(&force, "force", "f", false, "Actually terminate resources (not dry-run)")
	cmd.Flags().BoolVar(&dryRun, "dry-run", true, "Preview actions without making changes")

	return cmd
}

// createCostEstimateCommand creates the cost-estimate subcommand
func (z *ZombiesCobraCommands) createCostEstimateCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "cost-estimate",
		Short: "Estimate monthly cost of zombie resources",
		Long: `Calculates the estimated monthly AWS cost of all detected zombie resources.

This provides a clear ROI for cleanup - shows how much you could save by terminating zombies.`,
		Example: `  prism admin zombies cost-estimate`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return z.runCostEstimate(cmd.Context())
		},
	}
}

// runScan executes the zombie resource scan
func (z *ZombiesCobraCommands) runScan(ctx context.Context, verbose bool) error {
	fmt.Println("🔍 Scanning for zombie resources...")
	fmt.Println()

	// Get AWS manager from daemon or direct
	awsManager, err := z.app.GetAWSManager()
	if err != nil {
		return fmt.Errorf("failed to get AWS manager: %w", err)
	}

	// Scan for zombies
	result, err := awsManager.ScanZombieResources(ctx)
	if err != nil {
		return fmt.Errorf("scan failed: %w", err)
	}

	// Display results
	fmt.Printf("Region: %s\n", result.Region)
	fmt.Printf("Scan Time: %s\n", result.ScanTime.Format("2006-01-02 15:04:05 MST"))
	fmt.Println()

	// Instance summary
	if len(result.Instances) > 0 {
		fmt.Printf("❌ Zombie Instances: %d\n", len(result.Instances))
		if verbose {
			for _, instance := range result.Instances {
				fmt.Printf("  • %s (%s)\n", instance.InstanceID, instance.InstanceType)
				fmt.Printf("    State: %s\n", instance.State)
				fmt.Printf("    Name: %s\n", stringOrDefault(instance.Name, "<unnamed>"))
				fmt.Printf("    Launched: %s\n", instance.LaunchTime.Format("2006-01-02 15:04"))
				fmt.Printf("    Monthly Cost: $%.2f\n", instance.MonthlyCost)
			}
		}
	} else {
		fmt.Println("✅ No zombie instances found")
	}
	fmt.Println()

	// Volume summary
	if len(result.Volumes) > 0 {
		fmt.Printf("❌ Unattached Volumes: %d\n", len(result.Volumes))
		if verbose {
			for _, volume := range result.Volumes {
				fmt.Printf("  • %s (%d GB)\n", volume.VolumeID, volume.SizeGB)
				fmt.Printf("    Created: %s\n", volume.CreateTime.Format("2006-01-02 15:04"))
				fmt.Printf("    Monthly Cost: $%.2f\n", volume.MonthlyCost)
			}
		}
	} else {
		fmt.Println("✅ No unattached volumes found")
	}
	fmt.Println()

	// Cost summary
	fmt.Printf("💰 Total Monthly Cost: $%.2f\n", result.TotalMonthlyCost)
	fmt.Println()

	// Show prism-managed count
	fmt.Printf("✅ Prism-Managed Instances: %d\n", result.PrismInstancesCount)

	return nil
}

// runCleanup executes the zombie resource cleanup
func (z *ZombiesCobraCommands) runCleanup(ctx context.Context, force bool, dryRun bool) error {
	// First, scan to find zombies
	awsManager, err := z.app.GetAWSManager()
	if err != nil {
		return fmt.Errorf("failed to get AWS manager: %w", err)
	}

	result, err := awsManager.ScanZombieResources(ctx)
	if err != nil {
		return fmt.Errorf("scan failed: %w", err)
	}

	// Check if any zombies found
	totalZombies := len(result.Instances) + len(result.Volumes)
	if totalZombies == 0 {
		fmt.Println("✅ No zombie resources found - nothing to cleanup!")
		return nil
	}

	// Show what will be cleaned up
	fmt.Printf("Found %d zombie resources:\n", totalZombies)
	fmt.Printf("  • %d EC2 instances\n", len(result.Instances))
	fmt.Printf("  • %d EBS volumes\n", len(result.Volumes))
	fmt.Printf("  Monthly cost savings: $%.2f\n", result.TotalMonthlyCost)
	fmt.Println()

	// Dry run mode (default)
	if dryRun || !force {
		fmt.Println("🔒 DRY RUN MODE - No resources will be terminated")
		fmt.Println()
		fmt.Println("To actually cleanup these resources, run:")
		fmt.Println("  prism admin zombies cleanup --force --no-dry-run")
		return nil
	}

	// Force mode - require confirmation
	fmt.Println("⚠️  WARNING: This will PERMANENTLY TERMINATE:")
	for _, instance := range result.Instances {
		fmt.Printf("  • Instance %s (%s)\n", instance.InstanceID, instance.Name)
	}
	for _, volume := range result.Volumes {
		fmt.Printf("  • Volume %s (%d GB)\n", volume.VolumeID, volume.SizeGB)
	}
	fmt.Println()

	// Get confirmation
	fmt.Print("Type 'yes' to confirm: ")
	var confirmation string
	_, _ = fmt.Scanln(&confirmation) // empty or invalid input leaves confirmation unmatched, which is treated as "no"

	if strings.ToLower(strings.TrimSpace(confirmation)) != "yes" {
		fmt.Println("Cleanup cancelled.")
		return nil
	}

	// Perform cleanup
	fmt.Println()
	fmt.Println("🗑️  Cleaning up zombie resources...")

	instanceIDs := make([]string, len(result.Instances))
	for i, instance := range result.Instances {
		instanceIDs[i] = instance.InstanceID
	}

	volumeIDs := make([]string, len(result.Volumes))
	for i, volume := range result.Volumes {
		volumeIDs[i] = volume.VolumeID
	}

	err = awsManager.CleanupZombieResources(ctx, instanceIDs, volumeIDs)
	if err != nil {
		return fmt.Errorf("cleanup failed: %w", err)
	}

	fmt.Println("✅ Cleanup complete!")
	fmt.Printf("💰 Estimated monthly savings: $%.2f\n", result.TotalMonthlyCost)

	return nil
}

// runCostEstimate shows cost estimate for zombie resources
func (z *ZombiesCobraCommands) runCostEstimate(ctx context.Context) error {
	awsManager, err := z.app.GetAWSManager()
	if err != nil {
		return fmt.Errorf("failed to get AWS manager: %w", err)
	}

	result, err := awsManager.ScanZombieResources(ctx)
	if err != nil {
		return fmt.Errorf("scan failed: %w", err)
	}

	fmt.Println("💰 Zombie Resource Cost Estimate")
	fmt.Println("================================")
	fmt.Printf("Region: %s\n", result.Region)
	fmt.Println()

	// Instance costs
	instanceCost := 0.0
	for _, instance := range result.Instances {
		instanceCost += instance.MonthlyCost
	}

	// Volume costs
	volumeCost := 0.0
	for _, volume := range result.Volumes {
		volumeCost += volume.MonthlyCost
	}

	fmt.Printf("EC2 Instances (%d): $%.2f/month\n", len(result.Instances), instanceCost)
	fmt.Printf("EBS Volumes (%d):   $%.2f/month\n", len(result.Volumes), volumeCost)
	fmt.Println("--------------------------------")
	fmt.Printf("Total:              $%.2f/month\n", result.TotalMonthlyCost)
	fmt.Printf("Annual Cost:        $%.2f/year\n", result.TotalMonthlyCost*12)
	fmt.Println()

	if result.TotalMonthlyCost > 0 {
		fmt.Printf("💡 You could save $%.2f/month by cleaning up these resources.\n", result.TotalMonthlyCost)
		fmt.Println()
		fmt.Println("Run 'prism admin zombies cleanup --force' to terminate them.")
	}

	return nil
}

// stringOrDefault returns the string or a default if empty
func stringOrDefault(s, defaultVal string) string {
	if s == "" {
		return defaultVal
	}
	return s
}
