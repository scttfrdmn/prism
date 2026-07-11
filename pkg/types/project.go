// Package types provides project and budget management types for Prism.
//
// This file defines the core types for project-based resource organization and
// budget management, enabling researchers to organize instances, storage, and
// costs around research projects with proper financial controls.
package types

import (
	"time"
)

// OnboardingTemplate defines a template applied to new project members on joining (#154)
type OnboardingTemplate struct {
	// ID is the unique onboarding template identifier
	ID string `json:"id"`

	// Name is the onboarding template name
	Name string `json:"name"`

	// Templates lists the Prism workspace templates to provision for new members
	Templates []string `json:"templates"`

	// BudgetLimit is the per-member budget allocation created automatically on join
	BudgetLimit float64 `json:"budget_limit"`

	// Tags are optional metadata for organization
	Tags map[string]string `json:"tags,omitempty"`
}

// Project represents a research project with associated resources and budget
type Project struct {
	// ID is the unique project identifier
	ID string `json:"id"`

	// Name is the human-readable project name
	Name string `json:"name"`

	// Description provides project details
	Description string `json:"description"`

	// Owner is the project owner/principal investigator
	Owner string `json:"owner"`

	// Members are additional project members with access
	Members []ProjectMember `json:"members"`

	// Budget contains the project budget configuration (DEPRECATED in v0.5.10)
	// Use ProjectBudgetAllocation system instead
	Budget *ProjectBudget `json:"budget,omitempty"`

	// DefaultAllocationID is the default funding source for this project (v0.5.10+)
	// When users launch resources under this project without specifying --funding,
	// this allocation is automatically used
	DefaultAllocationID *string `json:"default_allocation_id,omitempty"`

	// Tags for project organization and reporting
	Tags map[string]string `json:"tags,omitempty"`

	// CreatedAt is when the project was created
	CreatedAt time.Time `json:"created_at"`

	// UpdatedAt is when the project was last modified
	UpdatedAt time.Time `json:"updated_at"`

	// Status indicates the project status
	Status ProjectStatus `json:"status"`

	// LaunchPrevented prevents new instance launches when true (set by budget actions)
	LaunchPrevented bool `json:"launch_prevented"`

	// OnboardingTemplates defines templates applied to new members on joining (#154)
	OnboardingTemplates []OnboardingTemplate `json:"onboarding_templates,omitempty"`

	// ApprovalPolicy configures budget-triggered launch approval (#495)
	ApprovalPolicy *ProjectApprovalPolicy `json:"approval_policy,omitempty"`
}

// ProjectApprovalPolicy defines when launches require PI/admin approval (#495)
type ProjectApprovalPolicy struct {
	// RequireApprovalAbove is the hourly cost threshold above which approval is required.
	// Zero means approval gating is disabled (default).
	RequireApprovalAbove float64 `json:"require_approval_above"`

	// ApproverRoles lists the project roles that can approve requests.
	// Defaults to ["admin", "owner"] when nil.
	ApproverRoles []string `json:"approver_roles,omitempty"`

	// ApprovalTimeoutHours is how long a pending request stays active before expiring.
	// Defaults to 24 when zero.
	ApprovalTimeoutHours int `json:"approval_timeout_hours,omitempty"`
}

// ProjectMember represents a project member with specific permissions
type ProjectMember struct {
	// UserID is the member's user identifier
	UserID string `json:"user_id"`

	// Role defines the member's permissions within the project
	Role ProjectRole `json:"role"`

	// AddedAt is when the member was added to the project
	AddedAt time.Time `json:"added_at"`

	// AddedBy is who added the member to the project
	AddedBy string `json:"added_by"`

	// ExpiresAt is when the member's access expires (nil = no expiry, #150)
	ExpiresAt *time.Time `json:"expires_at,omitempty"`

	// WarnedExpiry tracks whether an expiry warning has been sent (#150)
	WarnedExpiry bool `json:"warned_expiry,omitempty"`
}

// ProjectRole defines member permissions within a project
type ProjectRole string

const (
	// ProjectRoleOwner has full project control including budget and members
	ProjectRoleOwner ProjectRole = "owner"

	// ProjectRoleAdmin can manage resources and members but not budget
	ProjectRoleAdmin ProjectRole = "admin"

	// ProjectRoleMember can launch and manage their own instances
	ProjectRoleMember ProjectRole = "member"

	// ProjectRoleViewer can view project resources but not modify
	ProjectRoleViewer ProjectRole = "viewer"
)

// ProjectStatus represents the current status of a project
type ProjectStatus string

const (
	// ProjectStatusActive indicates an active project
	ProjectStatusActive ProjectStatus = "active"

	// ProjectStatusPaused indicates a temporarily paused project
	ProjectStatusPaused ProjectStatus = "paused"

	// ProjectStatusCompleted indicates a completed project
	ProjectStatusCompleted ProjectStatus = "completed"

	// ProjectStatusArchived indicates an archived project
	ProjectStatusArchived ProjectStatus = "archived"
)

// RoleQuota defines resource quotas for a specific project role (#151)
type RoleQuota struct {
	// Role is the project role this quota applies to
	Role ProjectRole `json:"role"`

	// MaxInstances is the maximum number of concurrent instances (-1 = unlimited)
	MaxInstances int `json:"max_instances"`

	// MaxInstanceType limits the instance type prefix (e.g. "t3" allows t3.* only; empty = unlimited)
	MaxInstanceType string `json:"max_instance_type,omitempty"`

	// MaxSpendDaily is the daily spend limit in USD (0 = unlimited)
	MaxSpendDaily float64 `json:"max_spend_daily"`
}

// GrantPeriod tracks a grant funding period with optional auto-freeze (#152)
type GrantPeriod struct {
	// Name is the grant period name (e.g. "NSF Year 1")
	Name string `json:"name"`

	// StartDate is when the grant period begins
	StartDate time.Time `json:"start_date"`

	// EndDate is when the grant period ends
	EndDate time.Time `json:"end_date"`

	// AutoFreeze automatically pauses the project when EndDate passes
	AutoFreeze bool `json:"auto_freeze"`

	// FrozenAt records when the project was auto-frozen (nil = not frozen)
	FrozenAt *time.Time `json:"frozen_at,omitempty"`
}

// ProjectBudget represents project budget configuration and tracking
type ProjectBudget struct {
	// TotalBudget is the total project budget in USD
	TotalBudget float64 `json:"total_budget"`

	// SpentAmount is the current amount spent in USD
	SpentAmount float64 `json:"spent_amount"`

	// MonthlyLimit is the optional monthly spending limit in USD
	MonthlyLimit *float64 `json:"monthly_limit,omitempty"`

	// DailyLimit is the optional daily spending limit in USD
	DailyLimit *float64 `json:"daily_limit,omitempty"`

	// AlertThresholds define when to send budget alerts
	AlertThresholds []BudgetAlert `json:"alert_thresholds"`

	// AutoActions define automatic actions when thresholds are reached
	AutoActions []BudgetAutoAction `json:"auto_actions"`

	// BudgetPeriod defines the budget period (project lifetime, monthly, etc.)
	BudgetPeriod BudgetPeriod `json:"budget_period"`

	// StartDate is when budget tracking began
	StartDate time.Time `json:"start_date"`

	// EndDate is when the budget period ends (optional)
	EndDate *time.Time `json:"end_date,omitempty"`

	// LastUpdated is when spending was last calculated
	LastUpdated time.Time `json:"last_updated"`

	// Cushion defines the automatic safety headroom configuration.
	// When enabled, Prism takes action (hibernate/stop/etc.) before the full
	// budget is exhausted, protecting against cost overruns.
	Cushion *CushionBudgetConfig `json:"cushion,omitempty"`

	// RoleQuotas defines per-role resource quotas (#151)
	RoleQuotas []RoleQuota `json:"role_quotas,omitempty"`

	// GrantPeriod tracks a funding grant period with auto-freeze (#152)
	GrantPeriod *GrantPeriod `json:"grant_period,omitempty"`

	// AllocationMonths is the number of months for multi-month budget allocation (#144)
	// 0 = single period (legacy)
	AllocationMonths int `json:"allocation_months,omitempty"`

	// MonthlyAmount is the per-month allocation for multi-month grants (#144)
	MonthlyAmount float64 `json:"monthly_amount,omitempty"`

	// RolloverEnabled allows unspent budget to carry over to the next period (#143)
	RolloverEnabled bool `json:"rollover_enabled,omitempty"`

	// RolloverCap is the maximum dollar amount that can roll over (0 = unlimited) (#143)
	RolloverCap float64 `json:"rollover_cap,omitempty"`
}

// CushionBudgetConfig is the budget-level cushion configuration stored in ProjectBudget.
// It mirrors project.CushionConfig but lives in types to avoid circular imports.
type CushionBudgetConfig struct {
	Enabled            bool     `json:"enabled"`
	HeadroomPercent    float64  `json:"headroom_percent"`
	HeadroomFixed      *float64 `json:"headroom_fixed_usd,omitempty"`
	Mode               string   `json:"mode"`
	NotifyBeforeAction bool     `json:"notify_before_action"`
	WarnLeadHours      int      `json:"warn_lead_hours"`
	// LastTriggeredAt records when the cushion last fired, for fire-once-per-budget-period dedup by
	// the live enforcer (#656). Nil = never fired.
	LastTriggeredAt *time.Time `json:"last_triggered_at,omitempty"`
}

// BudgetAlert defines a spending threshold that triggers notifications
type BudgetAlert struct {
	// Threshold is the spending percentage (0.0-1.0) that triggers the alert
	Threshold float64 `json:"threshold"`

	// Type defines the alert type
	Type BudgetAlertType `json:"type"`

	// Recipients defines who receives the alert
	Recipients []string `json:"recipients"`

	// Message is an optional custom alert message
	Message string `json:"message,omitempty"`

	// Enabled indicates if the alert is active
	Enabled bool `json:"enabled"`

	// LastTriggered is when this alert was last sent
	LastTriggered *time.Time `json:"last_triggered,omitempty"`
}

// BudgetAlertType defines the type of budget alert
type BudgetAlertType string

const (
	// BudgetAlertEmail sends email notifications
	BudgetAlertEmail BudgetAlertType = "email"

	// BudgetAlertSlack sends Slack notifications
	BudgetAlertSlack BudgetAlertType = "slack"

	// BudgetAlertWebhook sends webhook notifications
	BudgetAlertWebhook BudgetAlertType = "webhook"
)

// BudgetAutoAction defines automatic actions when budget thresholds are reached
type BudgetAutoAction struct {
	// Threshold is the spending percentage (0.0-1.0) that triggers the action
	Threshold float64 `json:"threshold"`

	// Action defines what action to take
	Action BudgetActionType `json:"action"`

	// Enabled indicates if the auto action is active
	Enabled bool `json:"enabled"`

	// LastTriggered is when this action was last executed
	LastTriggered *time.Time `json:"last_triggered,omitempty"`
}

// BudgetActionType defines automatic budget control actions
type BudgetActionType string

const (
	// BudgetActionHibernateAll hibernates all project instances
	BudgetActionHibernateAll BudgetActionType = "hibernate_all"

	// BudgetActionStopAll stops all project instances
	BudgetActionStopAll BudgetActionType = "stop_all"

	// BudgetActionPreventLaunch prevents new instance launches
	BudgetActionPreventLaunch BudgetActionType = "prevent_launch"

	// BudgetActionNotifyOnly only sends notifications without taking action
	BudgetActionNotifyOnly BudgetActionType = "notify_only"
)

// BudgetPeriod defines how budget periods are calculated
type BudgetPeriod string

const (
	// BudgetPeriodProject tracks budget for the entire project lifetime
	BudgetPeriodProject BudgetPeriod = "project"

	// BudgetPeriodMonthly resets budget tracking monthly
	BudgetPeriodMonthly BudgetPeriod = "monthly"

	// BudgetPeriodWeekly resets budget tracking weekly
	BudgetPeriodWeekly BudgetPeriod = "weekly"

	// BudgetPeriodDaily resets budget tracking daily
	BudgetPeriodDaily BudgetPeriod = "daily"
)

// ProjectCostBreakdown provides detailed cost analysis for a project
type ProjectCostBreakdown struct {
	// ProjectID is the project identifier
	ProjectID string `json:"project_id"`

	// TotalCost is the total project cost in USD
	TotalCost float64 `json:"total_cost"`

	// InstanceCosts breaks down costs by instance
	InstanceCosts []InstanceCost `json:"instance_costs"`

	// StorageCosts breaks down storage costs
	StorageCosts []StorageCost `json:"storage_costs"`

	// PeriodStart is the start of the cost reporting period
	PeriodStart time.Time `json:"period_start"`

	// PeriodEnd is the end of the cost reporting period
	PeriodEnd time.Time `json:"period_end"`

	// GeneratedAt is when this breakdown was generated
	GeneratedAt time.Time `json:"generated_at"`
}

// InstanceCost represents the cost breakdown for a specific instance
type InstanceCost struct {
	// InstanceName is the instance identifier
	InstanceName string `json:"instance_name"`

	// InstanceType is the AWS instance type
	InstanceType string `json:"instance_type"`

	// ComputeCost is the EC2 compute cost
	ComputeCost float64 `json:"compute_cost"`

	// StorageCost is the EBS storage cost
	StorageCost float64 `json:"storage_cost"`

	// TotalCost is the total instance cost
	TotalCost float64 `json:"total_cost"`

	// RunningHours is the number of hours the instance was running
	RunningHours float64 `json:"running_hours"`

	// HibernatedHours is the number of hours the instance was hibernated
	HibernatedHours float64 `json:"hibernated_hours"`

	// StoppedHours is the number of hours the instance was stopped
	StoppedHours float64 `json:"stopped_hours"`
}

// StorageCost represents the cost breakdown for storage
type StorageCost struct {
	// VolumeName is the storage volume identifier
	VolumeName string `json:"volume_name"`

	// VolumeType is the storage type (EFS, EBS, etc.)
	VolumeType string `json:"volume_type"`

	// SizeGB is the storage size in gigabytes
	SizeGB float64 `json:"size_gb"`

	// Cost is the storage cost in USD
	Cost float64 `json:"cost"`

	// CostPerGB is the cost per gigabyte
	CostPerGB float64 `json:"cost_per_gb"`
}

// ProjectResourceUsage provides resource utilization metrics for a project
type ProjectResourceUsage struct {
	// ProjectID is the project identifier
	ProjectID string `json:"project_id"`

	// ActiveInstances is the number of currently active instances
	ActiveInstances int `json:"active_instances"`

	// TotalInstances is the total number of instances ever launched
	TotalInstances int `json:"total_instances"`

	// TotalStorage is the total storage in GB across all volumes
	TotalStorage float64 `json:"total_storage"`

	// ComputeHours is the total compute hours used
	ComputeHours float64 `json:"compute_hours"`

	// IdleSavings is the estimated cost savings from idle policies
	IdleSavings float64 `json:"idle_savings"`

	// MeasurementPeriod defines the period for these metrics
	MeasurementPeriod time.Duration `json:"measurement_period"`

	// LastUpdated is when these metrics were last calculated
	LastUpdated time.Time `json:"last_updated"`
}

// ============================================================================
// v0.5.10: Multi-Budget System (Issue #97)
// ============================================================================
//
// The types below implement the many-to-many budget system introduced in
// v0.5.10, enabling:
//   - 1 budget → N projects (single grant funding multiple projects)
//   - N budgets → 1 project (multi-source funding)
//
// Migration Note: The embedded `Project.Budget *ProjectBudget` field is
// deprecated in favor of the new ProjectBudgetAllocation system.

// Budget represents a standalone budget pool that can be allocated to multiple projects.
//
// Examples:
//   - "NSF Grant CISE-2024-12345" ($50,000)
//   - "CS Department Q1 2026" ($100,000)
//   - "AWS Research Credits" ($5,000)
type Budget struct {
	// ID is the unique budget identifier
	ID string `json:"id"`

	// Name is the budget name (e.g., "NSF Grant CISE-2024-12345")
	Name string `json:"name"`

	// Description provides budget details
	Description string `json:"description"`

	// TotalAmount is the total budget pool in USD
	TotalAmount float64 `json:"total_amount"`

	// AllocatedAmount is the sum of all project allocations from this budget
	AllocatedAmount float64 `json:"allocated_amount"`

	// SpentAmount is the actual amount spent across all allocations
	SpentAmount float64 `json:"spent_amount"`

	// Period defines the budget period
	Period BudgetPeriod `json:"period"`

	// StartDate is when the budget period began
	StartDate time.Time `json:"start_date"`

	// EndDate is when the budget period ends (optional for ongoing budgets)
	EndDate *time.Time `json:"end_date,omitempty"`

	// AlertThreshold is the global alert percentage (0.0-1.0)
	AlertThreshold float64 `json:"alert_threshold"`

	// CreatedBy is the user who created the budget
	CreatedBy string `json:"created_by"`

	// CreatedAt is when the budget was created
	CreatedAt time.Time `json:"created_at"`

	// UpdatedAt is when the budget was last modified
	UpdatedAt time.Time `json:"updated_at"`

	// Tags for budget organization
	Tags map[string]string `json:"tags,omitempty"`
}

// ProjectBudgetAllocation represents the many-to-many relationship between
// budgets and projects, tracking how much of a budget is allocated to each project.
//
// Examples:
//   - NSF Grant ($50k) → Climate Project ($20k allocation)
//   - Department Budget ($100k) → Lab Project A ($40k), Lab Project B ($30k)
//   - Climate Project ← NSF ($20k) + MIT Matching ($5k) + AWS ($2k) = $27k total
type ProjectBudgetAllocation struct {
	// ID is the unique allocation identifier
	ID string `json:"id"`

	// BudgetID is the parent budget pool
	BudgetID string `json:"budget_id"`

	// ProjectID is the project receiving the allocation
	ProjectID string `json:"project_id"`

	// AllocatedAmount is how much of the budget is allocated to this project
	AllocatedAmount float64 `json:"allocated_amount"`

	// SpentAmount is how much has been spent from this allocation (cached)
	SpentAmount float64 `json:"spent_amount"`

	// AlertThreshold is an optional project-specific alert threshold (overrides budget default)
	AlertThreshold *float64 `json:"alert_threshold,omitempty"`

	// BackupAllocationID is an optional backup funding source for exhaustion (#234)
	BackupAllocationID *string `json:"backup_allocation_id,omitempty"`

	// Notes provide context for this allocation
	Notes string `json:"notes,omitempty"`

	// AllocatedAt is when this allocation was created
	AllocatedAt time.Time `json:"allocated_at"`

	// AllocatedBy is the user who created the allocation
	AllocatedBy string `json:"allocated_by"`

	// UpdatedAt is when the allocation was last modified
	UpdatedAt time.Time `json:"updated_at"`
}

// BudgetSummary provides a high-level view of a budget pool.
type BudgetSummary struct {
	// Budget is the budget details
	Budget Budget `json:"budget"`

	// Allocations is the list of project allocations
	Allocations []ProjectBudgetAllocation `json:"allocations"`

	// ProjectNames maps project IDs to names for display
	ProjectNames map[string]string `json:"project_names"`

	// RemainingAmount is unallocated budget (TotalAmount - AllocatedAmount)
	RemainingAmount float64 `json:"remaining_amount"`

	// UtilizationRate is the percentage of allocated funds actually spent
	UtilizationRate float64 `json:"utilization_rate"`
}

// ProjectFundingSummary provides a view of all funding sources for a project.
type ProjectFundingSummary struct {
	// ProjectID is the project identifier
	ProjectID string `json:"project_id"`

	// ProjectName is the project name
	ProjectName string `json:"project_name"`

	// Allocations is the list of budget allocations funding this project
	Allocations []ProjectBudgetAllocation `json:"allocations"`

	// BudgetNames maps budget IDs to names for display
	BudgetNames map[string]string `json:"budget_names"`

	// TotalAllocated is the sum of all allocations to this project
	TotalAllocated float64 `json:"total_allocated"`

	// TotalSpent is the sum of spending across all allocations
	TotalSpent float64 `json:"total_spent"`

	// DefaultAllocationID is the project's default funding source
	DefaultAllocationID *string `json:"default_allocation_id,omitempty"`
}

// ============================================================================
// v0.12.0: Budget Sharing, Reallocation, Cross-Project Borrowing (#143,#145,#155,#156)
// ============================================================================

// BudgetShareRequest represents a request to share or lend budget between projects or members
type BudgetShareRequest struct {
	// FromProjectID is the source project
	FromProjectID string `json:"from_project_id"`

	// ToProjectID is the destination project (nil for member reallocation)
	ToProjectID string `json:"to_project_id,omitempty"`

	// ToMemberID is the destination member (nil for cross-project share)
	ToMemberID string `json:"to_member_id,omitempty"`

	// Amount is the USD amount to share
	Amount float64 `json:"amount"`

	// Reason is an optional note for audit purposes
	Reason string `json:"reason,omitempty"`

	// ExpiresAt is when the share expires and should be auto-reversed (nil = permanent)
	ExpiresAt *time.Time `json:"expires_at,omitempty"`
}

// BudgetShareRecord records a completed budget share operation
type BudgetShareRecord struct {
	// ID is the unique share identifier
	ID string `json:"id"`

	// Request is the original share request
	Request BudgetShareRequest `json:"request"`

	// ApprovedBy is the user who approved the share
	ApprovedBy string `json:"approved_by"`

	// CreatedAt is when the share was created
	CreatedAt time.Time `json:"created_at"`

	// ExpiresAt is when the share expires (nil = permanent)
	ExpiresAt *time.Time `json:"expires_at,omitempty"`

	// Reversed indicates whether this share has been reversed
	Reversed bool `json:"reversed"`
}
