package policy

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/scttfrdmn/cloudworkstation/pkg/profile"
)

// Manager handles policy evaluation and enforcement
type Manager struct {
	policies       map[string]*Policy
	policySets     map[string]*PolicySet
	userPolicySets map[string][]string // user -> policy set IDs
}

// NewManager creates a new policy manager
func NewManager() *Manager {
	return &Manager{
		policies:       make(map[string]*Policy),
		policySets:     make(map[string]*PolicySet),
		userPolicySets: make(map[string][]string),
	}
}

// AddPolicy adds a policy to the manager
func (m *Manager) AddPolicy(policy *Policy) error {
	if policy.ID == "" {
		return fmt.Errorf("policy ID cannot be empty")
	}
	policy.UpdatedAt = time.Now()
	m.policies[policy.ID] = policy
	return nil
}

// AddPolicySet adds a policy set to the manager
func (m *Manager) AddPolicySet(policySet *PolicySet) error {
	if policySet.ID == "" {
		return fmt.Errorf("policy set ID cannot be empty")
	}
	policySet.UpdatedAt = time.Now()
	m.policySets[policySet.ID] = policySet
	return nil
}

// AssignPolicySet assigns a policy set to a user
func (m *Manager) AssignPolicySet(userID, policySetID string) error {
	if _, exists := m.policySets[policySetID]; !exists {
		return fmt.Errorf("policy set %s not found", policySetID)
	}

	userSets := m.userPolicySets[userID]
	for _, setID := range userSets {
		if setID == policySetID {
			return nil // Already assigned
		}
	}

	m.userPolicySets[userID] = append(userSets, policySetID)
	return nil
}

// EvaluatePolicy evaluates a policy request and returns the decision
func (m *Manager) EvaluatePolicy(request *PolicyRequest) *PolicyResponse {
	response := &PolicyResponse{
		Allowed:         true, // Default allow for basic framework
		MatchedPolicies: []string{},
		Suggestions:     []string{},
	}

	// Get user's policy sets
	userSets := m.userPolicySets[request.UserID]
	if len(userSets) == 0 {
		// No policies assigned - default allow with educational message
		response.Reason = "No policies configured - using default allow"
		response.Suggestions = append(response.Suggestions, "Configure policy sets for enhanced security")
		return response
	}

	// Evaluate policies from user's policy sets
	applicablePolicies := m.getApplicablePolicies(userSets, request)

	// Evaluate each applicable policy
	for _, policy := range applicablePolicies {
		if !policy.Enabled {
			continue
		}

		matches, reason := m.evaluateSinglePolicy(policy, request)
		if matches {
			response.MatchedPolicies = append(response.MatchedPolicies, policy.ID)

			if policy.Effect == PolicyEffectDeny {
				response.Allowed = false
				response.Reason = reason
				response.Suggestions = append(response.Suggestions, m.generateSuggestions(policy, request)...)
				return response // First deny wins
			}
		}
	}

	if response.Allowed && len(response.MatchedPolicies) > 0 {
		response.Reason = "Access granted by policy evaluation"
	}

	return response
}

// getApplicablePolicies returns policies that might apply to the request
func (m *Manager) getApplicablePolicies(userSets []string, request *PolicyRequest) []*Policy {
	var policies []*Policy

	for _, setID := range userSets {
		policySet := m.policySets[setID]
		if policySet == nil || !policySet.Enabled {
			continue
		}

		for _, policy := range policySet.Policies {
			if m.policyAppliesTo(policy, request) {
				policies = append(policies, policy)
			}
		}
	}

	return policies
}

// policyAppliesTo checks if a policy applies to the given request
func (m *Manager) policyAppliesTo(policy *Policy, request *PolicyRequest) bool {
	// Check if policy type matches the request context
	if !m.policyTypeMatches(policy.Type, request.Action, request.Resource) {
		return false
	}

	// Check if action matches
	if len(policy.Actions) > 0 {
		actionMatches := false
		for _, action := range policy.Actions {
			if strings.Contains(request.Action, action) || action == "*" {
				actionMatches = true
				break
			}
		}
		if !actionMatches {
			return false
		}
	}

	// Check if resource matches
	if len(policy.Resources) > 0 {
		resourceMatches := false
		for _, resource := range policy.Resources {
			if strings.Contains(request.Resource, resource) || resource == "*" {
				resourceMatches = true
				break
			}
		}
		if !resourceMatches {
			return false
		}
	}

	return true
}

// policyTypeMatches determines if a policy type is relevant to the request
func (m *Manager) policyTypeMatches(policyType PolicyType, action, resource string) bool {
	switch policyType {
	case PolicyTypeTemplateAccess:
		return strings.Contains(action, "template") || strings.Contains(resource, "template")
	case PolicyTypeResourceLimits:
		return strings.Contains(action, "launch") || strings.Contains(action, "create")
	case PolicyTypeResearchUser:
		return strings.Contains(action, "research_user") || strings.Contains(resource, "research_user")
	case PolicyTypeInstance:
		return strings.Contains(action, "instance") || strings.Contains(resource, "instance")
	default:
		return true
	}
}

// evaluateSinglePolicy evaluates a single policy against the request
func (m *Manager) evaluateSinglePolicy(policy *Policy, request *PolicyRequest) (bool, string) {
	// Parse policy conditions based on type
	switch policy.Type {
	case PolicyTypeTemplateAccess:
		return m.evaluateTemplateAccessPolicy(policy, request)
	case PolicyTypeResourceLimits:
		return m.evaluateResourceLimitsPolicy(policy, request)
	case PolicyTypeResearchUser:
		return m.evaluateResearchUserPolicy(policy, request)
	default:
		// For basic policies, match if no specific conditions
		return len(policy.Conditions) == 0, "Basic policy evaluation"
	}
}

// evaluateTemplateAccessPolicy evaluates template access policies
func (m *Manager) evaluateTemplateAccessPolicy(policy *Policy, request *PolicyRequest) (bool, string) {
	if len(policy.Conditions) == 0 {
		return true, "No template restrictions"
	}

	// Parse template access conditions
	conditionsJSON, _ := json.Marshal(policy.Conditions)
	var templatePolicy TemplateAccessPolicy
	json.Unmarshal(conditionsJSON, &templatePolicy)

	templateName := request.Resource

	// Check denied templates first
	for _, denied := range templatePolicy.DeniedTemplates {
		if strings.Contains(templateName, denied) {
			return true, fmt.Sprintf("Template %s is denied by policy", templateName)
		}
	}

	// Check allowed templates
	if len(templatePolicy.AllowedTemplates) > 0 {
		allowed := false
		for _, allowedTemplate := range templatePolicy.AllowedTemplates {
			if strings.Contains(templateName, allowedTemplate) || allowedTemplate == "*" {
				allowed = true
				break
			}
		}
		if !allowed {
			return true, fmt.Sprintf("Template %s is not in allowed list", templateName)
		}
	}

	return false, "Template access allowed"
}

// evaluateResourceLimitsPolicy evaluates resource limit policies
func (m *Manager) evaluateResourceLimitsPolicy(policy *Policy, request *PolicyRequest) (bool, string) {
	// Basic resource limit evaluation - can be extended
	return false, "Resource limits evaluation not implemented"
}

// evaluateResearchUserPolicy evaluates research user policies
func (m *Manager) evaluateResearchUserPolicy(policy *Policy, request *PolicyRequest) (bool, string) {
	if len(policy.Conditions) == 0 {
		return false, "No research user restrictions"
	}

	// Parse research user conditions
	conditionsJSON, _ := json.Marshal(policy.Conditions)
	var researchPolicy ResearchUserPolicy
	json.Unmarshal(conditionsJSON, &researchPolicy)

	action := request.Action

	// Check creation permissions
	if strings.Contains(action, "create") && !researchPolicy.AllowCreation {
		return true, "Research user creation is not allowed by policy"
	}

	// Check deletion permissions
	if strings.Contains(action, "delete") && !researchPolicy.AllowDeletion {
		return true, "Research user deletion is not allowed by policy"
	}

	return false, "Research user action allowed"
}

// generateSuggestions provides helpful suggestions when access is denied
func (m *Manager) generateSuggestions(policy *Policy, request *PolicyRequest) []string {
	var suggestions []string

	switch policy.Type {
	case PolicyTypeTemplateAccess:
		suggestions = append(suggestions, "Try using a different template from the allowed list")
		suggestions = append(suggestions, "Contact your administrator to request access to this template")
	case PolicyTypeResourceLimits:
		suggestions = append(suggestions, "Consider using a smaller instance type")
		suggestions = append(suggestions, "Use spot instances to reduce costs")
	case PolicyTypeResearchUser:
		suggestions = append(suggestions, "Contact your administrator to enable research user creation")
		suggestions = append(suggestions, "Use existing research users instead of creating new ones")
	}

	return suggestions
}

// CheckTemplateAccess is a convenience method for checking template access
func (m *Manager) CheckTemplateAccess(userID, templateName string, profileID string) *PolicyResponse {
	request := &PolicyRequest{
		UserID:    userID,
		Action:    "template_access",
		Resource:  templateName,
		ProfileID: profileID,
		Context: map[string]interface{}{
			"template_name": templateName,
		},
	}
	return m.EvaluatePolicy(request)
}

// CheckResearchUserAction is a convenience method for checking research user actions
func (m *Manager) CheckResearchUserAction(userID, action, username string, profileID string) *PolicyResponse {
	request := &PolicyRequest{
		UserID:    userID,
		Action:    fmt.Sprintf("research_user_%s", action),
		Resource:  fmt.Sprintf("research_user/%s", username),
		ProfileID: profileID,
		Context: map[string]interface{}{
			"username": username,
			"action":   action,
		},
	}
	return m.EvaluatePolicy(request)
}

// CreateDefaultPolicySets creates example policy sets for different user types
func (m *Manager) CreateDefaultPolicySets() error {
	// Student policy set - restricted access
	studentPolicy := &Policy{
		ID:          "student-template-access",
		Name:        "Student Template Access",
		Description: "Restricts students to basic and educational templates",
		Type:        PolicyTypeTemplateAccess,
		Effect:      PolicyEffectDeny,
		Conditions: map[string]interface{}{
			"denied_templates": []string{"GPU", "Enterprise", "Production"},
			"max_complexity":   "moderate",
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Enabled:   true,
	}

	studentResearchPolicy := &Policy{
		ID:          "student-research-user",
		Name:        "Student Research User Policy",
		Description: "Allows basic research user operations for students",
		Type:        PolicyTypeResearchUser,
		Effect:      PolicyEffectAllow,
		Conditions: map[string]interface{}{
			"allow_creation":      true,
			"allow_deletion":      false,
			"max_users":           1,
			"allowed_shells":      []string{"/bin/bash", "/bin/sh"},
			"allow_ssh_keys":      true,
			"allow_sudo_access":   false,
			"allow_docker_access": false,
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Enabled:   true,
	}

	studentPolicySet := &PolicySet{
		ID:          "student",
		Name:        "Student Policy Set",
		Description: "Default policies for students",
		Policies:    []*Policy{studentPolicy, studentResearchPolicy},
		Tags: map[string]string{
			"type": "educational",
			"role": "student",
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Enabled:   true,
	}

	// Researcher policy set - more permissive
	researcherPolicy := &Policy{
		ID:          "researcher-full-access",
		Name:        "Researcher Template Access",
		Description: "Full access to research templates",
		Type:        PolicyTypeTemplateAccess,
		Effect:      PolicyEffectAllow,
		Actions:     []string{"*"},
		Resources:   []string{"*"},
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		Enabled:     true,
	}

	researcherUserPolicy := &Policy{
		ID:          "researcher-research-user",
		Name:        "Researcher Research User Policy",
		Description: "Full research user management for researchers",
		Type:        PolicyTypeResearchUser,
		Effect:      PolicyEffectAllow,
		Conditions: map[string]interface{}{
			"allow_creation":      true,
			"allow_deletion":      true,
			"max_users":           5,
			"allow_ssh_keys":      true,
			"allow_sudo_access":   true,
			"allow_docker_access": true,
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Enabled:   true,
	}

	researcherPolicySet := &PolicySet{
		ID:          "researcher",
		Name:        "Researcher Policy Set",
		Description: "Full access policies for researchers",
		Policies:    []*Policy{researcherPolicy, researcherUserPolicy},
		Tags: map[string]string{
			"type": "research",
			"role": "researcher",
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Enabled:   true,
	}

	// Add policy sets
	if err := m.AddPolicySet(studentPolicySet); err != nil {
		return err
	}
	if err := m.AddPolicySet(researcherPolicySet); err != nil {
		return err
	}

	return nil
}

// GetProfileUserID returns a user identifier for policy evaluation
// This integrates with the existing profile system
func (m *Manager) GetProfileUserID() string {
	// Create a profile manager to get current profile
	profileManager, err := profile.NewManagerEnhanced()
	if err != nil {
		return "default_user"
	}

	// Get current profile for user identification
	currentProfile, err := profileManager.GetCurrentProfile()
	if err != nil {
		return "default_user"
	}

	// Use profile name as user identifier for policy evaluation
	return currentProfile.Name
}
