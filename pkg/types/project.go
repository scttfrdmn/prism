// Package types provides project and budget management types for CloudWorkstation.
//
// This file defines the core types for project-based resource organization and
// budget management, enabling researchers to organize instances, storage, and
// costs around research projects with proper financial controls.
package types

import (
	"time"
)

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

	// Budget contains the project budget configuration
	Budget *ProjectBudget `json:"budget,omitempty"`

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
