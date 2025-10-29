package cli

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

// PolicyCobraCommands handles policy management commands using Cobra framework
type PolicyCobraCommands struct {
	app *App
}

// NewPolicyCobraCommands creates a new policy commands handler
func NewPolicyCobraCommands(app *App) *PolicyCobraCommands {
	return &PolicyCobraCommands{app: app}
}

// CreatePolicyCommand creates the main policy command with subcommands
func (p *PolicyCobraCommands) CreatePolicyCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "policy",
		Short: "Manage policy framework for template and resource access control",
		Long: `Manage Prism's policy framework for controlling access to templates,
resources, and research user operations. Policies enable fine-grained access control
for educational and research environments.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Show help if no subcommand provided
			return cmd.Help()
		},
	}

	// Add subcommands
	cmd.AddCommand(p.createStatusCommand())
	cmd.AddCommand(p.createListCommand())
	cmd.AddCommand(p.createAssignCommand())
	cmd.AddCommand(p.createEnableCommand())
	cmd.AddCommand(p.createDisableCommand())
	cmd.AddCommand(p.createCheckCommand())

	return cmd
}

// createStatusCommand creates the policy status command
func (p *PolicyCobraCommands) createStatusCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show policy enforcement status",
		Long:  "Display the current policy enforcement status and assigned policy sets.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return p.handlePolicyStatus()
		},
	}
}

// createListCommand creates the policy list command
func (p *PolicyCobraCommands) createListCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List available policy sets",
		Long:  "Display all available policy sets with their descriptions and status.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return p.handlePolicyList()
		},
	}
}

// createAssignCommand creates the policy assign command
func (p *PolicyCobraCommands) createAssignCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "assign <policy-set>",
		Short: "Assign a policy set to current user",
		Long:  "Assign a policy set (student, researcher) to the current user for access control.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return p.handlePolicyAssign(args)
		},
	}
}

// createEnableCommand creates the policy enable command
func (p *PolicyCobraCommands) createEnableCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "enable",
		Short: "Enable policy enforcement",
		Long:  "Enable policy enforcement for template and resource access control.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return p.handlePolicyEnable()
		},
	}
}

// createDisableCommand creates the policy disable command
func (p *PolicyCobraCommands) createDisableCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "disable",
		Short: "Disable policy enforcement",
		Long:  "Disable policy enforcement, allowing unrestricted access to all resources.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return p.handlePolicyDisable()
		},
	}
}

// createCheckCommand creates the policy check command
func (p *PolicyCobraCommands) createCheckCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "check <template>",
		Short: "Check template access permissions",
		Long:  "Check whether the current user has access to a specific template.",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return p.handlePolicyCheck(args)
		},
	}
}

// Command handlers that integrate with daemon-based policy service
func (p *PolicyCobraCommands) handlePolicyStatus() error {
	// Ensure daemon is running
	if err := p.app.ensureDaemonRunning(); err != nil {
		return err
	}

	// Get policy status from daemon
	response, err := p.app.apiClient.GetPolicyStatus(p.app.ctx)
	if err != nil {
		return fmt.Errorf("failed to get policy status: %v", err)
	}

	// Display policy status
	fmt.Printf("Policy Framework Status: %s\n", response.StatusIcon)
	fmt.Printf("Enforcement: %s\n", response.Status)

	if len(response.AssignedPolicies) > 0 {
		fmt.Printf("Assigned Policy Sets: %s\n", strings.Join(response.AssignedPolicies, ", "))
	} else {
		fmt.Println("Assigned Policy Sets: None (default allow)")
	}

	if response.Message != "" {
		fmt.Printf("\n%s\n", response.Message)
	}

	fmt.Println()
	fmt.Println("💡 Tip: Use 'prism policy assign <policy-set>' to configure access controls")

	return nil
}

func (p *PolicyCobraCommands) handlePolicyList() error {
	// Ensure daemon is running
	if err := p.app.ensureDaemonRunning(); err != nil {
		return err
	}

	// List policy sets from daemon
	response, err := p.app.apiClient.ListPolicySets(p.app.ctx)
	if err != nil {
		return fmt.Errorf("failed to list policy sets: %v", err)
	}

	if len(response.PolicySets) == 0 {
		fmt.Println("No policy sets available")
		return nil
	}

	fmt.Println("Available Policy Sets:")
	fmt.Println()

	// Display policy sets in a table format
	fmt.Printf("%-15s %-30s %-10s %s\n", "NAME", "DESCRIPTION", "POLICIES", "STATUS")
	fmt.Printf("%-15s %-30s %-10s %s\n", "----", "-----------", "--------", "------")

	for _, policySet := range response.PolicySets {
		fmt.Printf("%-15s %-30s %-10d %s\n",
			policySet.ID,
			policySet.Description,
			policySet.Policies,
			policySet.Status)
	}

	fmt.Println()
	fmt.Println("Use 'prism policy assign <policy-set>' to assign a policy set")

	return nil
}

func (p *PolicyCobraCommands) handlePolicyAssign(args []string) error {
	// Ensure daemon is running
	if err := p.app.ensureDaemonRunning(); err != nil {
		return err
	}

	policySet := args[0]

	// Assign policy set via daemon API
	response, err := p.app.apiClient.AssignPolicySet(p.app.ctx, policySet)
	if err != nil {
		return fmt.Errorf("failed to assign policy set: %v", err)
	}

	if response.Success {
		fmt.Printf("✅ %s\n", response.Message)
		fmt.Println()
		fmt.Printf("💡 Policy enforcement is %s. Use 'prism policy enable' to activate.\n", response.EnforcementStatus)
	} else {
		fmt.Printf("❌ Failed to assign policy set '%s'\n", policySet)
	}

	return nil
}

func (p *PolicyCobraCommands) handlePolicyEnable() error {
	// Ensure daemon is running
	if err := p.app.ensureDaemonRunning(); err != nil {
		return err
	}

	// Enable policy enforcement via daemon API
	response, err := p.app.apiClient.SetPolicyEnforcement(p.app.ctx, true)
	if err != nil {
		return fmt.Errorf("failed to enable policy enforcement: %v", err)
	}

	if response.Success {
		fmt.Printf("%s\n", response.Message)
	} else {
		fmt.Printf("❌ Failed to enable policy enforcement\n")
	}

	return nil
}

func (p *PolicyCobraCommands) handlePolicyDisable() error {
	// Ensure daemon is running
	if err := p.app.ensureDaemonRunning(); err != nil {
		return err
	}

	// Disable policy enforcement via daemon API
	response, err := p.app.apiClient.SetPolicyEnforcement(p.app.ctx, false)
	if err != nil {
		return fmt.Errorf("failed to disable policy enforcement: %v", err)
	}

	if response.Success {
		fmt.Printf("%s\n", response.Message)
		fmt.Printf("💡 All resources will be accessible when disabled\n")
	} else {
		fmt.Printf("❌ Failed to disable policy enforcement\n")
	}

	return nil
}

func (p *PolicyCobraCommands) handlePolicyCheck(args []string) error {
	// Ensure daemon is running
	if err := p.app.ensureDaemonRunning(); err != nil {
		return err
	}

	templateName := args[0]
	fmt.Printf("🔍 Checking access for template '%s'...\n", templateName)

	// Check template access via daemon API
	response, err := p.app.apiClient.CheckTemplateAccess(p.app.ctx, templateName)
	if err != nil {
		return fmt.Errorf("failed to check template access: %v", err)
	}

	// Display access result
	if response.Allowed {
		fmt.Printf("✅ Access GRANTED to template '%s'\n", response.TemplateName)
		if response.Reason != "" {
			fmt.Printf("📋 Reason: %s\n", response.Reason)
		}
	} else {
		fmt.Printf("❌ Access DENIED to template '%s'\n", response.TemplateName)
		if response.Reason != "" {
			fmt.Printf("📋 Reason: %s\n", response.Reason)
		}
	}

	// Show matched policies if any
	if len(response.MatchedPolicies) > 0 {
		fmt.Printf("📜 Matched Policies: %s\n", strings.Join(response.MatchedPolicies, ", "))
	}

	// Show suggestions if any
	if len(response.Suggestions) > 0 {
		fmt.Println("\n💡 Suggestions:")
		for _, suggestion := range response.Suggestions {
			fmt.Printf("  • %s\n", suggestion)
		}
	}

	return nil
}

// PolicyCommandFactory creates policy commands using the factory pattern
type PolicyCommandFactory struct {
	app *App
}

// CreateCommands returns all policy-related commands
func (f *PolicyCommandFactory) CreateCommands() []*cobra.Command {
	policyCobra := NewPolicyCobraCommands(f.app)
	return []*cobra.Command{
		policyCobra.CreatePolicyCommand(),
	}
}
