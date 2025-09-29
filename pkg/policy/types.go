package policy

import (
	"time"
)

// PolicyType represents different types of policies
type PolicyType string

const (
	// PolicyTypeTemplateAccess controls which templates users can access
	PolicyTypeTemplateAccess PolicyType = "template_access"

	// PolicyTypeResourceLimits controls resource allocation limits
	PolicyTypeResourceLimits PolicyType = "resource_limits"

	// PolicyTypeResearchUser controls research user operations
	PolicyTypeResearchUser PolicyType = "research_user"

	// PolicyTypeInstance controls instance management
	PolicyTypeInstance PolicyType = "instance"
)

// PolicyEffect determines whether a policy allows or denies access
type PolicyEffect string

const (
	PolicyEffectAllow PolicyEffect = "allow"
	PolicyEffectDeny  PolicyEffect = "deny"
)

// Policy represents a single policy rule
type Policy struct {
	ID          string                 `json:"id" yaml:"id"`
	Name        string                 `json:"name" yaml:"name"`
	Description string                 `json:"description,omitempty" yaml:"description,omitempty"`
	Type        PolicyType             `json:"type" yaml:"type"`
	Effect      PolicyEffect           `json:"effect" yaml:"effect"`
	Conditions  map[string]interface{} `json:"conditions,omitempty" yaml:"conditions,omitempty"`
	Resources   []string               `json:"resources,omitempty" yaml:"resources,omitempty"`
	Actions     []string               `json:"actions,omitempty" yaml:"actions,omitempty"`
	CreatedAt   time.Time              `json:"created_at" yaml:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at" yaml:"updated_at"`
	Enabled     bool                   `json:"enabled" yaml:"enabled"`
}

// PolicySet represents a collection of policies for a user or group
type PolicySet struct {
	ID          string            `json:"id" yaml:"id"`
	Name        string            `json:"name" yaml:"name"`
	Description string            `json:"description,omitempty" yaml:"description,omitempty"`
	Policies    []*Policy         `json:"policies" yaml:"policies"`
	Tags        map[string]string `json:"tags,omitempty" yaml:"tags,omitempty"`
	CreatedAt   time.Time         `json:"created_at" yaml:"created_at"`
	UpdatedAt   time.Time         `json:"updated_at" yaml:"updated_at"`
	Enabled     bool              `json:"enabled" yaml:"enabled"`
}

// PolicyRequest represents a request to check policy permissions
type PolicyRequest struct {
	UserID    string                 `json:"user_id"`
	Action    string                 `json:"action"`
	Resource  string                 `json:"resource"`
	Context   map[string]interface{} `json:"context,omitempty"`
	ProfileID string                 `json:"profile_id,omitempty"`
}

// PolicyResponse represents the result of a policy evaluation
type PolicyResponse struct {
	Allowed         bool     `json:"allowed"`
	Reason          string   `json:"reason,omitempty"`
	MatchedPolicies []string `json:"matched_policies,omitempty"`
	Suggestions     []string `json:"suggestions,omitempty"`
}

// TemplateAccessPolicy controls access to specific templates
type TemplateAccessPolicy struct {
	AllowedTemplates []string `json:"allowed_templates,omitempty" yaml:"allowed_templates,omitempty"`
	DeniedTemplates  []string `json:"denied_templates,omitempty" yaml:"denied_templates,omitempty"`
	RequiredDomain   string   `json:"required_domain,omitempty" yaml:"required_domain,omitempty"`
	MaxComplexity    string   `json:"max_complexity,omitempty" yaml:"max_complexity,omitempty"`
}

// ResourceLimitsPolicy controls resource allocation
type ResourceLimitsPolicy struct {
	MaxInstances     int               `json:"max_instances,omitempty" yaml:"max_instances,omitempty"`
	MaxInstanceTypes []string          `json:"max_instance_types,omitempty" yaml:"max_instance_types,omitempty"`
	MaxCostPerHour   float64           `json:"max_cost_per_hour,omitempty" yaml:"max_cost_per_hour,omitempty"`
	MaxVolumes       int               `json:"max_volumes,omitempty" yaml:"max_volumes,omitempty"`
	AllowedRegions   []string          `json:"allowed_regions,omitempty" yaml:"allowed_regions,omitempty"`
	RequireSpot      bool              `json:"require_spot,omitempty" yaml:"require_spot,omitempty"`
	Tags             map[string]string `json:"required_tags,omitempty" yaml:"required_tags,omitempty"`
}

// ResearchUserPolicy controls research user operations
type ResearchUserPolicy struct {
	AllowCreation     bool     `json:"allow_creation" yaml:"allow_creation"`
	AllowDeletion     bool     `json:"allow_deletion" yaml:"allow_deletion"`
	RequireApproval   bool     `json:"require_approval,omitempty" yaml:"require_approval,omitempty"`
	MaxUsers          int      `json:"max_users,omitempty" yaml:"max_users,omitempty"`
	AllowedShells     []string `json:"allowed_shells,omitempty" yaml:"allowed_shells,omitempty"`
	RequiredGroups    []string `json:"required_groups,omitempty" yaml:"required_groups,omitempty"`
	AllowSSHKeys      bool     `json:"allow_ssh_keys" yaml:"allow_ssh_keys"`
	AllowSudoAccess   bool     `json:"allow_sudo_access,omitempty" yaml:"allow_sudo_access,omitempty"`
	AllowDockerAccess bool     `json:"allow_docker_access,omitempty" yaml:"allow_docker_access,omitempty"`
}
