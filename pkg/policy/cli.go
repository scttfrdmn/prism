package policy

import (
	"fmt"
	"os"
	"strings"
	"text/tabwriter"
)

// CLIHandler provides command-line interface for policy management
type CLIHandler struct {
	service *Service
}

// NewCLIHandler creates a new CLI handler for policies
func NewCLIHandler(service *Service) *CLIHandler {
	return &CLIHandler{service: service}
}

// HandlePolicyCommand processes policy-related CLI commands
func (h *CLIHandler) HandlePolicyCommand(args []string) error {
	if len(args) == 0 {
		return h.showPolicyHelp()
	}

	switch args[0] {
	case "status":
		return h.showPolicyStatus()
	case "list":
		return h.listPolicySets()
	case "assign":
		return h.assignPolicySet(args[1:])
	case "enable":
		h.service.SetEnabled(true)
		fmt.Println("✅ Policy enforcement enabled")
		return nil
	case "disable":
		h.service.SetEnabled(false)
		fmt.Println("⚠️  Policy enforcement disabled")
		return nil
	case "check":
		return h.checkTemplateAccess(args[1:])
	default:
		return fmt.Errorf("unknown policy command: %s", args[0])
	}
}

// showPolicyHelp displays help information for policy commands
func (h *CLIHandler) showPolicyHelp() error {
	fmt.Println("Policy Framework Commands:")
	fmt.Println("")
	fmt.Println("  cws policy status              Show policy enforcement status")
	fmt.Println("  cws policy list                List available policy sets")
	fmt.Println("  cws policy assign <policy-set> Assign a policy set to current user")
	fmt.Println("  cws policy enable              Enable policy enforcement")
	fmt.Println("  cws policy disable             Disable policy enforcement")
	fmt.Println("  cws policy check <template>    Check template access permissions")
	fmt.Println("")
	fmt.Println("Available Policy Sets:")
	fmt.Println("  student     - Restricted access for educational environments")
	fmt.Println("  researcher  - Full access for research users")
	fmt.Println("")
	fmt.Println("Examples:")
	fmt.Println("  cws policy assign student")
	fmt.Println("  cws policy check \"Python ML Research\"")
	fmt.Println("")
	return nil
}

// showPolicyStatus displays the current policy enforcement status
func (h *CLIHandler) showPolicyStatus() error {
	fmt.Printf("Policy Framework Status: %s\n", h.getStatusIcon())
	fmt.Printf("Enforcement: %s\n", h.getEnforcementStatus())

	currentPolicies := h.service.GetCurrentUserPolicies()
	if len(currentPolicies) > 0 {
		fmt.Printf("Assigned Policy Sets: %s\n", strings.Join(currentPolicies, ", "))
	} else {
		fmt.Println("Assigned Policy Sets: None (default allow)")
	}

	fmt.Println()
	fmt.Println("💡 Tip: Use 'cws policy assign <policy-set>' to configure access controls")
	return nil
}

// listPolicySets displays available policy sets
func (h *CLIHandler) listPolicySets() error {
	policySets := h.service.ListAvailablePolicySets()

	if len(policySets) == 0 {
		fmt.Println("No policy sets available")
		return nil
	}

	fmt.Println("Available Policy Sets:")
	fmt.Println()

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "NAME\tDESCRIPTION\tPOLICIES\tSTATUS")
	fmt.Fprintln(w, "----\t-----------\t--------\t------")

	for id, policySet := range policySets {
		status := "Enabled"
		if !policySet.Enabled {
			status = "Disabled"
		}

		fmt.Fprintf(w, "%s\t%s\t%d\t%s\n",
			id,
			policySet.Description,
			len(policySet.Policies),
			status)
	}

	w.Flush()
	fmt.Println()
	fmt.Println("Use 'cws policy assign <policy-set>' to assign a policy set")
	return nil
}

// assignPolicySet assigns a policy set to the current user
func (h *CLIHandler) assignPolicySet(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("policy set name required. Use 'cws policy list' to see available sets")
	}

	policySetName := args[0]

	// Validate policy set exists
	policySets := h.service.ListAvailablePolicySets()
	if _, exists := policySets[policySetName]; !exists {
		return fmt.Errorf("policy set '%s' not found. Use 'cws policy list' to see available sets", policySetName)
	}

	var err error
	switch policySetName {
	case "student":
		err = h.service.AssignStudentPolicies()
	case "researcher":
		err = h.service.AssignResearcherPolicies()
	default:
		return fmt.Errorf("assignment not implemented for policy set: %s", policySetName)
	}

	if err != nil {
		return fmt.Errorf("failed to assign policy set: %v", err)
	}

	fmt.Printf("✅ Successfully assigned '%s' policy set\n", policySetName)
	fmt.Println()
	fmt.Printf("💡 Policy enforcement is %s. Use 'cws policy enable' to activate.\n",
		h.getEnforcementStatus())
	return nil
}

// checkTemplateAccess checks access permissions for a specific template
func (h *CLIHandler) checkTemplateAccess(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("template name required")
	}

	templateName := strings.Join(args, " ")
	response := h.service.CheckTemplateAccess(templateName)

	if response.Allowed {
		fmt.Printf("✅ Access ALLOWED for template: %s\n", templateName)
		if response.Reason != "" {
			fmt.Printf("Reason: %s\n", response.Reason)
		}
	} else {
		fmt.Printf("❌ Access DENIED for template: %s\n", templateName)
		fmt.Printf("Reason: %s\n", response.Reason)

		if len(response.Suggestions) > 0 {
			fmt.Println("\nSuggestions:")
			for _, suggestion := range response.Suggestions {
				fmt.Printf("  • %s\n", suggestion)
			}
		}
	}

	if len(response.MatchedPolicies) > 0 {
		fmt.Printf("\nMatched Policies: %s\n", strings.Join(response.MatchedPolicies, ", "))
	}

	return nil
}

// getStatusIcon returns a status icon for policy enforcement
func (h *CLIHandler) getStatusIcon() string {
	if h.service.IsEnabled() {
		return "🔒 Active"
	}
	return "🔓 Inactive"
}

// getEnforcementStatus returns the enforcement status as a string
func (h *CLIHandler) getEnforcementStatus() string {
	if h.service.IsEnabled() {
		return "Enabled"
	}
	return "Disabled"
}
