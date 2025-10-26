package policy

import (
	"fmt"
	"log"
)

// Service provides policy enforcement for Prism
type Service struct {
	manager *Manager
	enabled bool
}

// NewService creates a new policy service
func NewService() *Service {
	manager := NewManager()

	// Initialize with default policy sets
	if err := manager.CreateDefaultPolicySets(); err != nil {
		log.Printf("Warning: Failed to create default policy sets: %v", err)
	}

	return &Service{
		manager: manager,
		enabled: true, // Policies are enabled by default
	}
}

// SetEnabled controls whether policy enforcement is active
func (s *Service) SetEnabled(enabled bool) {
	s.enabled = enabled
}

// IsEnabled returns whether policy enforcement is active
func (s *Service) IsEnabled() bool {
	return s.enabled
}

// CheckTemplateAccess enforces template access policies
func (s *Service) CheckTemplateAccess(templateName string) *PolicyResponse {
	if !s.enabled {
		return &PolicyResponse{
			Allowed: true,
			Reason:  "Policy enforcement disabled",
		}
	}

	userID := s.manager.GetProfileUserID()
	return s.manager.CheckTemplateAccess(userID, templateName, "")
}

// CheckResearchUserCreation enforces research user creation policies
func (s *Service) CheckResearchUserCreation(username string) *PolicyResponse {
	if !s.enabled {
		return &PolicyResponse{
			Allowed: true,
			Reason:  "Policy enforcement disabled",
		}
	}

	userID := s.manager.GetProfileUserID()
	return s.manager.CheckResearchUserAction(userID, "create", username, "")
}

// CheckResearchUserDeletion enforces research user deletion policies
func (s *Service) CheckResearchUserDeletion(username string) *PolicyResponse {
	if !s.enabled {
		return &PolicyResponse{
			Allowed: true,
			Reason:  "Policy enforcement disabled",
		}
	}

	userID := s.manager.GetProfileUserID()
	return s.manager.CheckResearchUserAction(userID, "delete", username, "")
}

// AssignStudentPolicies assigns student-level policies to current user
func (s *Service) AssignStudentPolicies() error {
	if !s.enabled {
		return nil
	}

	userID := s.manager.GetProfileUserID()
	return s.manager.AssignPolicySet(userID, "student")
}

// AssignResearcherPolicies assigns researcher-level policies to current user
func (s *Service) AssignResearcherPolicies() error {
	if !s.enabled {
		return nil
	}

	userID := s.manager.GetProfileUserID()
	return s.manager.AssignPolicySet(userID, "researcher")
}

// GetPolicyViolationMessage returns user-friendly policy violation messages
func (s *Service) GetPolicyViolationMessage(response *PolicyResponse, action string) string {
	if response.Allowed {
		return ""
	}

	baseMsg := fmt.Sprintf("Access denied: %s", response.Reason)

	if len(response.Suggestions) > 0 {
		baseMsg += "\n\nSuggestions:"
		for _, suggestion := range response.Suggestions {
			baseMsg += fmt.Sprintf("\n  • %s", suggestion)
		}
	}

	baseMsg += fmt.Sprintf("\n\nTo modify policies, use: cws policy %s", action)

	return baseMsg
}

// ListAvailablePolicySets returns available policy sets for assignment
func (s *Service) ListAvailablePolicySets() map[string]*PolicySet {
	if !s.enabled {
		return make(map[string]*PolicySet)
	}

	return s.manager.policySets
}

// GetCurrentUserPolicies returns the current user's assigned policies
func (s *Service) GetCurrentUserPolicies() []string {
	if !s.enabled {
		return []string{}
	}

	userID := s.manager.GetProfileUserID()
	return s.manager.userPolicySets[userID]
}

// ValidateTemplateAccess provides template filtering based on policies
func (s *Service) ValidateTemplateAccess(templates []string) ([]string, []string) {
	if !s.enabled {
		return templates, []string{}
	}

	var allowed []string
	var denied []string

	for _, template := range templates {
		response := s.CheckTemplateAccess(template)
		if response.Allowed {
			allowed = append(allowed, template)
		} else {
			denied = append(denied, template)
		}
	}

	return allowed, denied
}

// CreateCustomPolicy allows users to create custom policies
func (s *Service) CreateCustomPolicy(policy *Policy) error {
	if !s.enabled {
		return fmt.Errorf("policy enforcement is disabled")
	}

	// Validate policy
	if policy.Name == "" {
		return fmt.Errorf("policy name is required")
	}

	if policy.Type == "" {
		return fmt.Errorf("policy type is required")
	}

	if policy.Effect == "" {
		return fmt.Errorf("policy effect is required")
	}

	return s.manager.AddPolicy(policy)
}

// GetManager returns the underlying policy manager for advanced operations
func (s *Service) GetManager() *Manager {
	return s.manager
}
