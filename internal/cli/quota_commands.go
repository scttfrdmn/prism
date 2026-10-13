package cli

import (
	"context"
	"fmt"

	"github.com/scttfrdmn/prism/pkg/aws"
	"github.com/spf13/cobra"
)

// QuotaCobraCommands handles quota management commands
type QuotaCobraCommands struct {
	app *App
}

// NewQuotaCobraCommands creates a new QuotaCobraCommands instance
func NewQuotaCobraCommands(app *App) *QuotaCobraCommands {
	return &QuotaCobraCommands{app: app}
}

// CreateQuotaCommand creates the quota command with subcommands
func (c *QuotaCobraCommands) CreateQuotaCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "quota",
		Short: "AWS quota management and monitoring",
		Long: `View and manage AWS service quotas for EC2 instances.

Helps prevent launch failures due to quota limits by providing:
- Current quota usage and availability
- Proactive warnings at 90% usage
- Guidance for requesting quota increases`,
	}

	cmd.AddCommand(c.createShowCommand())
	cmd.AddCommand(c.createRequestCommand())
	cmd.AddCommand(c.createListCommand())

	return cmd
}

// createShowCommand creates the 'show' subcommand
func (c *QuotaCobraCommands) createShowCommand() *cobra.Command {
	var instanceType string

	cmd := &cobra.Command{
		Use:   "show",
		Short: "Show quota information for an instance type",
		Long: `Display detailed quota information for a specific instance type.

Shows:
- Current vCPU usage
- Quota limit
- Available capacity
- Usage percentage
- Warnings if usage is high`,
		Example: `  prism admin quota show --instance-type t3.medium
  prism admin quota show --instance-type g4dn.xlarge`,
		RunE: func(_ *cobra.Command, _ []string) error {
			if instanceType == "" {
				return fmt.Errorf("--instance-type is required")
			}

			return c.showQuota(instanceType)
		},
	}

	cmd.Flags().StringVar(&instanceType, "instance-type", "", "Instance type to check (e.g., t3.medium, m5.large)")
	_ = cmd.MarkFlagRequired("instance-type") // only errors if the flag is undefined; it is defined above

	return cmd
}

// createRequestCommand creates the 'request' subcommand
func (c *QuotaCobraCommands) createRequestCommand() *cobra.Command {
	var instanceType string

	cmd := &cobra.Command{
		Use:   "request",
		Short: "Generate quota increase request guidance",
		Long: `Generate detailed guidance for requesting a quota increase.

Provides:
- Current quota information
- Recommended increase amount
- Step-by-step instructions
- Direct AWS console links`,
		Example: `  prism admin quota request --instance-type t3.medium
  prism admin quota request --instance-type g4dn.xlarge`,
		RunE: func(_ *cobra.Command, _ []string) error {
			if instanceType == "" {
				return fmt.Errorf("--instance-type is required")
			}

			return c.requestQuotaIncrease(instanceType)
		},
	}

	cmd.Flags().StringVar(&instanceType, "instance-type", "", "Instance type for quota increase")
	_ = cmd.MarkFlagRequired("instance-type") // only errors if the flag is undefined; it is defined above

	return cmd
}

// createListCommand creates the 'list' subcommand
func (c *QuotaCobraCommands) createListCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List all relevant AWS quotas",
		Long: `Display all EC2 quotas relevant to Prism users.

Shows quotas for:
- Standard instances (A, C, D, H, I, M, R, T, Z)
- GPU instances (G, P)
- EBS volumes and storage
- Elastic IP addresses`,
		Example: `  prism admin quota list`,
		RunE: func(_ *cobra.Command, _ []string) error {
			return c.listQuotas()
		},
	}

	return cmd
}

// showQuota displays quota information for an instance type
func (c *QuotaCobraCommands) showQuota(instanceType string) error {
	ctx := context.Background()

	// Get AWS manager
	manager, err := c.app.getAWSManager()
	if err != nil {
		return fmt.Errorf("failed to initialize AWS manager: %w", err)
	}

	quotaManager := manager.GetQuotaManager()

	// Validate instance launch (which provides detailed quota info)
	validation, err := quotaManager.ValidateInstanceLaunch(ctx, instanceType)
	if err != nil {
		return fmt.Errorf("failed to validate quota: %w", err)
	}

	// Format and display output
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Printf("📊 Quota Information: %s\n", instanceType)
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println()
	fmt.Printf("Quota Type:        %s\n", validation.QuotaType)
	fmt.Printf("Current Usage:     %d vCPUs\n", validation.CurrentUsage)
	fmt.Printf("Quota Limit:       %d vCPUs\n", validation.QuotaLimit)
	fmt.Printf("Available:         %d vCPUs\n", validation.AvailableCapacity)
	fmt.Printf("Usage Percent:     %.1f%%\n", validation.UsagePercent)
	fmt.Printf("Required for %s: %d vCPUs\n", instanceType, validation.RequiredCapacity)
	fmt.Println()

	// Status indicator
	if validation.IsValid {
		if validation.UsagePercent >= 90 {
			fmt.Println("⚠️  Status: AVAILABLE (High Usage)")
		} else if validation.UsagePercent >= 70 {
			fmt.Println("⚡ Status: AVAILABLE (Moderate Usage)")
		} else {
			fmt.Println("✅ Status: AVAILABLE")
		}
	} else {
		fmt.Println("❌ Status: INSUFFICIENT CAPACITY")
	}

	// Warnings and suggestions
	if validation.Warning != "" {
		fmt.Println()
		fmt.Printf("⚠️  Warning: %s\n", validation.Warning)
	}

	if validation.SuggestedAction != "" {
		fmt.Println()
		fmt.Printf("💡 Suggested Action:\n   %s\n", validation.SuggestedAction)
	}

	fmt.Println()
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	return nil
}

// requestQuotaIncrease generates quota increase request guidance
func (c *QuotaCobraCommands) requestQuotaIncrease(instanceType string) error {
	ctx := context.Background()

	// Get AWS manager
	manager, err := c.app.getAWSManager()
	if err != nil {
		return fmt.Errorf("failed to initialize AWS manager: %w", err)
	}

	quotaManager := manager.GetQuotaManager()
	helper := aws.NewQuotaRequestHelper(quotaManager, manager.GetDefaultRegion())

	// Generate quota increase request
	request, err := helper.GenerateQuotaIncreaseRequest(ctx, instanceType)
	if err != nil {
		return fmt.Errorf("failed to generate quota request: %w", err)
	}

	// Display formatted guidance
	fmt.Println(helper.FormatQuotaRequest(request))

	return nil
}

// listQuotas lists all relevant AWS quotas
func (c *QuotaCobraCommands) listQuotas() error {
	ctx := context.Background()

	// Get AWS manager
	manager, err := c.app.getAWSManager()
	if err != nil {
		return fmt.Errorf("failed to initialize AWS manager: %w", err)
	}

	quotaManager := manager.GetQuotaManager()

	// Get all relevant quotas
	quotas, err := quotaManager.ListRelevantQuotas(ctx)
	if err != nil {
		return fmt.Errorf("failed to list quotas: %w", err)
	}

	// Format and display output
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Printf("📊 AWS EC2 Quotas - Region: %s\n", manager.GetDefaultRegion())
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println()

	for _, quota := range quotas {
		fmt.Printf("📦 %s\n", quota.QuotaName)
		fmt.Printf("   Limit:      %.0f %s\n", quota.Value, quota.Unit)

		if quota.UsageValue > 0 {
			fmt.Printf("   Usage:      %.0f %s (%.1f%%)\n", quota.UsageValue, quota.Unit, quota.UsagePercent)

			// Status indicator
			if quota.UsagePercent >= 90 {
				fmt.Printf("   Status:     ⚠️  High Usage\n")
			} else if quota.UsagePercent >= 70 {
				fmt.Printf("   Status:     ⚡ Moderate Usage\n")
			} else {
				fmt.Printf("   Status:     ✅ Healthy\n")
			}
		}

		if quota.IsAdjustable {
			fmt.Printf("   Adjustable: ✓ (Can request increase)\n")
		} else {
			fmt.Printf("   Adjustable: ✗ (Fixed limit)\n")
		}

		fmt.Printf("   Code:       %s\n", quota.QuotaCode)
		fmt.Println()
	}

	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println()
	fmt.Println("💡 Tip: Use 'prism admin quota show --instance-type <type>' for detailed quota info")
	fmt.Println("   Example: prism admin quota show --instance-type t3.medium")
	fmt.Println()

	return nil
}

// getAWSManager is a helper to get the AWS manager from the App
func (a *App) getAWSManager() (*aws.Manager, error) {
	// Get current profile
	profile, err := a.profileManager.GetCurrentProfile()
	if err != nil {
		return nil, fmt.Errorf("failed to get current profile: %w", err)
	}

	// Create AWS manager with options
	manager, err := aws.NewManager(aws.ManagerOptions{
		Profile: profile.AWSProfile,
		Region:  profile.Region,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create AWS manager: %w", err)
	}

	return manager, nil
}
