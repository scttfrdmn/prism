package project

import (
	"fmt"
	"time"

	"github.com/scttfrdmn/prism/pkg/types"
)

// CreateProjectRequest represents a request to create a new project
type CreateProjectRequest struct {
	// Name is the project name (required)
	Name string `json:"name"`

	// Description provides project details
	Description string `json:"description"`

	// Owner is the project owner/principal investigator
	Owner string `json:"owner"`

	// Tags for project organization
	Tags map[string]string `json:"tags,omitempty"`

	// Budget contains optional budget configuration
	Budget *CreateBudgetRequest `json:"budget,omitempty"`
}

// Validate validates the create project request
func (r *CreateProjectRequest) Validate() error {
	if r.Name == "" {
		return fmt.Errorf("project name is required")
	}

	if len(r.Name) > 100 {
		return fmt.Errorf("project name cannot exceed 100 characters")
	}

	if len(r.Description) > 1000 {
		return fmt.Errorf("project description cannot exceed 1000 characters")
	}

	if r.Budget != nil {
		if err := r.Budget.Validate(); err != nil {
			return fmt.Errorf("invalid budget configuration: %w", err)
		}
	}

	return nil
}

// CreateBudgetRequest represents a request to create a project budget
type CreateBudgetRequest struct {
	// TotalBudget is the total project budget in USD
	TotalBudget float64 `json:"total_budget"`

	// MonthlyLimit is the optional monthly spending limit in USD
	MonthlyLimit *float64 `json:"monthly_limit,omitempty"`

	// DailyLimit is the optional daily spending limit in USD
	DailyLimit *float64 `json:"daily_limit,omitempty"`

	// AlertThresholds define when to send budget alerts
	AlertThresholds []types.BudgetAlert `json:"alert_thresholds,omitempty"`

	// AutoActions define automatic actions when thresholds are reached
	AutoActions []types.BudgetAutoAction `json:"auto_actions,omitempty"`

	// BudgetPeriod defines the budget period
	BudgetPeriod types.BudgetPeriod `json:"budget_period"`

	// EndDate is when the budget period ends (optional)
	EndDate *time.Time `json:"end_date,omitempty"`
}

// Validate validates the create budget request
func (r *CreateBudgetRequest) Validate() error {
	if r.TotalBudget <= 0 {
		return fmt.Errorf("total budget must be greater than 0")
	}

	if r.MonthlyLimit != nil && *r.MonthlyLimit <= 0 {
		return fmt.Errorf("monthly limit must be greater than 0")
	}

	if r.DailyLimit != nil && *r.DailyLimit <= 0 {
		return fmt.Errorf("daily limit must be greater than 0")
	}

	// Validate alert thresholds
	for i, alert := range r.AlertThresholds {
		if alert.Threshold < 0 || alert.Threshold > 1 {
			return fmt.Errorf("alert threshold %d must be between 0.0 and 1.0", i)
		}
	}

	// Validate auto actions
	for i, action := range r.AutoActions {
		if action.Threshold < 0 || action.Threshold > 1 {
			return fmt.Errorf("auto action threshold %d must be between 0.0 and 1.0", i)
		}
	}

	return nil
}

// UpdateProjectRequest represents a request to update a project
type UpdateProjectRequest struct {
	// Name is the new project name (optional)
	Name *string `json:"name,omitempty"`

	// Description is the new project description (optional)
	Description *string `json:"description,omitempty"`

	// Tags are the new project tags (optional)
	Tags map[string]string `json:"tags,omitempty"`

	// Status is the new project status (optional)
	Status *types.ProjectStatus `json:"status,omitempty"`
}

// ProjectFilter defines filtering options for listing projects
type ProjectFilter struct {
	// Owner filters by project owner
	Owner string `json:"owner,omitempty"`

	// Status filters by project status
	Status *types.ProjectStatus `json:"status,omitempty"`

	// Tags filters by project tags (all specified tags must match)
	Tags map[string]string `json:"tags,omitempty"`

	// CreatedAfter filters projects created after this date
	CreatedAfter *time.Time `json:"created_after,omitempty"`

	// CreatedBefore filters projects created before this date
	CreatedBefore *time.Time `json:"created_before,omitempty"`

	// HasBudget filters projects with/without budgets
	HasBudget *bool `json:"has_budget,omitempty"`
}

// Matches checks if a project matches the filter criteria
func (f *ProjectFilter) Matches(project *types.Project) bool {
	// Check basic project attributes
	if !f.matchesBasicAttributes(project) {
		return false
	}

	// Check date-based filters
	if !f.matchesDateFilters(project) {
		return false
	}

	// Check tag-based filters
	if !f.matchesTagFilters(project) {
		return false
	}

	return true
}

// matchesBasicAttributes checks owner, status, and budget filters
func (f *ProjectFilter) matchesBasicAttributes(project *types.Project) bool {
	// Check owner filter
	if !f.matchesOwnerFilter(project) {
		return false
	}

	// Check status filter
	if !f.matchesStatusFilter(project) {
		return false
	}

	// Check budget filter
	if !f.matchesBudgetFilter(project) {
		return false
	}

	return true
}

// matchesOwnerFilter checks if project matches the owner filter
func (f *ProjectFilter) matchesOwnerFilter(project *types.Project) bool {
	return f.Owner == "" || project.Owner == f.Owner
}

// matchesStatusFilter checks if project matches the status filter
func (f *ProjectFilter) matchesStatusFilter(project *types.Project) bool {
	return f.Status == nil || project.Status == *f.Status
}

// matchesBudgetFilter checks if project matches the budget filter
func (f *ProjectFilter) matchesBudgetFilter(project *types.Project) bool {
	if f.HasBudget == nil {
		return true
	}

	hasBudget := project.Budget != nil
	return hasBudget == *f.HasBudget
}

// matchesDateFilters checks creation date filters
func (f *ProjectFilter) matchesDateFilters(project *types.Project) bool {
	// Check created after filter
	if f.CreatedAfter != nil && project.CreatedAt.Before(*f.CreatedAfter) {
		return false
	}

	// Check created before filter
	if f.CreatedBefore != nil && project.CreatedAt.After(*f.CreatedBefore) {
		return false
	}

	return true
}

// matchesTagFilters checks if all specified tags match
func (f *ProjectFilter) matchesTagFilters(project *types.Project) bool {
	if len(f.Tags) == 0 {
		return true
	}

	if project.Tags == nil {
		return false
	}

	// All specified tags must match
	for key, value := range f.Tags {
		if projectValue, exists := project.Tags[key]; !exists || projectValue != value {
			return false
		}
	}

	return true
}

// BudgetStatus represents the current budget status of a project
type BudgetStatus struct {
	// ProjectID is the project identifier
	ProjectID string `json:"project_id"`

	// BudgetEnabled indicates if budget tracking is enabled
	BudgetEnabled bool `json:"budget_enabled"`

	// TotalBudget is the total project budget
	TotalBudget float64 `json:"total_budget"`

	// SpentAmount is the current amount spent
	SpentAmount float64 `json:"spent_amount"`

	// RemainingBudget is the remaining budget
	RemainingBudget float64 `json:"remaining_budget"`

	// SpentPercentage is the percentage of budget spent (0.0-1.0)
	SpentPercentage float64 `json:"spent_percentage"`

	// ProjectedMonthlySpend is the projected monthly spending based on current usage
	ProjectedMonthlySpend float64 `json:"projected_monthly_spend"`

	// DaysUntilBudgetExhausted estimates when budget will be exhausted at current rate
	DaysUntilBudgetExhausted *int `json:"days_until_exhausted,omitempty"`

	// ActiveAlerts are currently active budget alerts
	ActiveAlerts []string `json:"active_alerts"`

	// TriggeredActions are actions that have been triggered
	TriggeredActions []string `json:"triggered_actions"`

	// LastUpdated is when this status was calculated
	LastUpdated time.Time `json:"last_updated"`
}

// ProjectSummary provides a condensed view of project information
type ProjectSummary struct {
	// ID is the project identifier
	ID string `json:"id"`

	// Name is the project name
	Name string `json:"name"`

	// Owner is the project owner
	Owner string `json:"owner"`

	// Status is the project status
	Status types.ProjectStatus `json:"status"`

	// MemberCount is the number of project members
	MemberCount int `json:"member_count"`

	// ActiveInstances is the number of currently active instances
	ActiveInstances int `json:"active_instances"`

	// TotalCost is the total project cost to date
	TotalCost float64 `json:"total_cost"`

	// BudgetStatus provides budget information if budget is enabled
	BudgetStatus *BudgetStatusSummary `json:"budget_status,omitempty"`

	// CreatedAt is when the project was created
	CreatedAt time.Time `json:"created_at"`

	// LastActivity is when the project had its last activity
	LastActivity time.Time `json:"last_activity"`
}

// BudgetStatusSummary provides a condensed view of budget status
type BudgetStatusSummary struct {
	// TotalBudget is the total project budget
	TotalBudget float64 `json:"total_budget"`

	// SpentAmount is the current amount spent
	SpentAmount float64 `json:"spent_amount"`

	// SpentPercentage is the percentage of budget spent (0.0-1.0)
	SpentPercentage float64 `json:"spent_percentage"`

	// AlertCount is the number of active alerts
	AlertCount int `json:"alert_count"`
}

// ProjectListResponse represents the response for listing projects
type ProjectListResponse struct {
	// Projects are the matching projects
	Projects []ProjectSummary `json:"projects"`

	// TotalCount is the total number of projects (before pagination)
	TotalCount int `json:"total_count"`

	// FilteredCount is the number of projects matching the filter
	FilteredCount int `json:"filtered_count"`
}

// AddMemberRequest represents a request to add a member to a project
type AddMemberRequest struct {
	// UserID is the user identifier
	UserID string `json:"user_id"`

	// Role is the project role for the member
	Role types.ProjectRole `json:"role"`

	// AddedBy is who is adding the member
	AddedBy string `json:"added_by"`
}

// Validate validates the add member request
func (r *AddMemberRequest) Validate() error {
	if r.UserID == "" {
		return fmt.Errorf("user ID is required")
	}

	if r.Role == "" {
		return fmt.Errorf("role is required")
	}

	// Validate role
	switch r.Role {
	case types.ProjectRoleOwner, types.ProjectRoleAdmin, types.ProjectRoleMember, types.ProjectRoleViewer:
		// Valid roles
	default:
		return fmt.Errorf("invalid role: %s", r.Role)
	}

	return nil
}

// UpdateMemberRequest represents a request to update a project member
type UpdateMemberRequest struct {
	// Role is the new role for the member
	Role types.ProjectRole `json:"role"`
}

// Validate validates the update member request
func (r *UpdateMemberRequest) Validate() error {
	if r.Role == "" {
		return fmt.Errorf("role is required")
	}

	// Validate role
	switch r.Role {
	case types.ProjectRoleOwner, types.ProjectRoleAdmin, types.ProjectRoleMember, types.ProjectRoleViewer:
		// Valid roles
	default:
		return fmt.Errorf("invalid role: %s", r.Role)
	}

	return nil
}
