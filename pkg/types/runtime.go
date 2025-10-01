package types

import (
	"time"
)

// RuntimeTemplate defines a cloud workstation template for launching instances
// This is distinct from AMI build templates (see pkg/ami package)
type RuntimeTemplate struct {
	Name                 string
	Slug                 string // CLI identifier for template (e.g., "python-ml")
	Description          string
	AMI                  map[string]map[string]string // region -> arch -> AMI ID
	InstanceType         map[string]string            // arch -> instance type
	UserData             string
	Ports                []int
	EstimatedCostPerHour map[string]float64   // arch -> cost per hour
	IdleDetection        *IdleDetectionConfig // Idle detection configuration

	// Research user integration (Phase 5A+)
	ResearchUser *ResearchUserTemplate `json:"research_user,omitempty"`
}

// ResearchUserTemplate represents research user integration configuration for templates
type ResearchUserTemplate struct {
	AutoCreate          bool                 `yaml:"auto_create" json:"auto_create"`
	RequireEFS          bool                 `yaml:"require_efs" json:"require_efs"`
	EFSMountPoint       string               `yaml:"efs_mount_point" json:"efs_mount_point"`
	InstallSSHKeys      bool                 `yaml:"install_ssh_keys" json:"install_ssh_keys"`
	DefaultShell        string               `yaml:"default_shell" json:"default_shell"`
	DefaultGroups       []string             `yaml:"default_groups" json:"default_groups"`
	DualUserIntegration *DualUserIntegration `yaml:"user_integration" json:"user_integration"`
}

// DualUserIntegration represents dual user system configuration
type DualUserIntegration struct {
	Strategy             string `yaml:"strategy" json:"strategy"`
	PrimaryUser          string `yaml:"primary_user" json:"primary_user"`
	CollaborationEnabled bool   `yaml:"collaboration_enabled" json:"collaboration_enabled"`
}

// IdleDetectionConfig represents idle detection configuration in templates
type IdleDetectionConfig struct {
	Enabled                   bool `yaml:"enabled" json:"enabled"`
	IdleThresholdMinutes      int  `yaml:"idle_threshold_minutes" json:"idle_threshold_minutes"`
	HibernateThresholdMinutes int  `yaml:"hibernate_threshold_minutes" json:"hibernate_threshold_minutes"`
	CheckIntervalMinutes      int  `yaml:"check_interval_minutes" json:"check_interval_minutes"`
}

// Instance represents a running cloud workstation
type Instance struct {
	ID                 string                  `json:"id"`
	Name               string                  `json:"name"`
	Template           string                  `json:"template"`
	PublicIP           string                  `json:"public_ip"`
	PrivateIP          string                  `json:"private_ip"`
	State              string                  `json:"state"`
	LaunchTime         time.Time               `json:"launch_time"`
	DeletionTime       *time.Time              `json:"deletion_time,omitempty"` // When user initiated deletion
	HourlyRate         float64                 `json:"hourly_rate"`             // AWS list price per hour
	CurrentSpend       float64                 `json:"current_spend"`           // Actual accumulated cost since launch
	EffectiveRate      float64                 `json:"effective_rate"`          // Current spend ÷ hours since launch
	AttachedVolumes    []string                `json:"attached_volumes"`        // EFS volume names
	AttachedEBSVolumes []string                `json:"attached_ebs_volumes"`    // EBS volume IDs
	InstanceType       string                  `json:"instance_type"`
	InstanceLifecycle  string                  `json:"instance_lifecycle"` // "spot" or "on-demand"
	Username           string                  `json:"username"`
	WebPort            int                     `json:"web_port"`
	HasWebInterface    bool                    `json:"has_web_interface"`
	ProjectID          string                  `json:"project_id,omitempty"` // Associated project ID
	IdleDetection      *IdleDetection          `json:"idle_detection,omitempty"`
	AppliedTemplates   []AppliedTemplateRecord `json:"applied_templates,omitempty"` // Template application history

	// Cost optimization fields
	EstimatedCost     float64 `json:"estimated_cost,omitempty"` // Daily cost estimate
	IdlePolicyEnabled bool    `json:"idle_policy_enabled,omitempty"`
	SpotEligible      bool    `json:"spot_eligible,omitempty"`
	IsSpot            bool    `json:"is_spot,omitempty"`
	ARMCompatible     bool    `json:"arm_compatible,omitempty"`
	Architecture      string  `json:"architecture,omitempty"`
	AlwaysOn          bool    `json:"always_on,omitempty"`
	WorkloadType      string  `json:"workload_type,omitempty"`
	Runtime           float64 `json:"runtime,omitempty"` // Hours since launch
	StorageGB         float64 `json:"storage_gb,omitempty"`
	StorageUsedGB     float64 `json:"storage_used_gb,omitempty"`

	// Universal AMI System fields (Phase 5.1)
	AMIResolutionMethod string        `json:"ami_resolution_method,omitempty"` // How AMI was resolved
	AMIID               string        `json:"ami_id,omitempty"`                // AMI used for launch
	CostSavings         float64       `json:"cost_savings,omitempty"`          // Cost savings vs script
	BootTime            time.Duration `json:"boot_time,omitempty"`             // Time to boot from AMI
}

// IdleDetection represents idle detection configuration for an instance
type IdleDetection struct {
	Enabled        bool      `json:"enabled"`
	Policy         string    `json:"policy"`
	IdleTime       int       `json:"idle_time"`       // Minutes
	Threshold      int       `json:"threshold"`       // Minutes
	ActionSchedule time.Time `json:"action_schedule"` // When action will occur
	ActionPending  bool      `json:"action_pending"`  // Whether action is pending
}

// CreditInfo represents AWS credit information
type CreditInfo struct {
	TotalCredits     float64    `json:"total_credits"`
	RemainingCredits float64    `json:"remaining_credits"`
	UsedCredits      float64    `json:"used_credits"`
	CreditType       string     `json:"credit_type"` // "AWS Promotional", "AWS Educate", etc.
	ExpirationDate   *time.Time `json:"expiration_date,omitempty"`
	Description      string     `json:"description"`
}

// BillingInfo represents current billing and cost information
type BillingInfo struct {
	MonthToDateSpend float64      `json:"month_to_date_spend"`
	ForecastedSpend  float64      `json:"forecasted_spend"`
	Credits          []CreditInfo `json:"credits"`
	BillingPeriod    string       `json:"billing_period"`
	LastUpdated      time.Time    `json:"last_updated"`
}

// DiscountConfig represents pricing discount configuration
type DiscountConfig struct {
	EC2Discount              float64 `json:"ec2_discount"`               // Percentage discount (0.0-1.0)
	EBSDiscount              float64 `json:"ebs_discount"`               // Percentage discount (0.0-1.0)
	EFSDiscount              float64 `json:"efs_discount"`               // Percentage discount (0.0-1.0)
	SavingsPlansDiscount     float64 `json:"savings_plans_discount"`     // Additional savings plan discount
	ReservedInstanceDiscount float64 `json:"reserved_instance_discount"` // RI discount
	SpotDiscount             float64 `json:"spot_discount"`              // Spot instance discount
	VolumeDiscount           float64 `json:"volume_discount"`            // Volume discount for large usage
	EducationalDiscount      float64 `json:"educational_discount"`       // Educational institution discount
	StartupDiscount          float64 `json:"startup_discount"`           // AWS Activate/startup credits
	EnterpriseDiscount       float64 `json:"enterprise_discount"`        // Enterprise agreement discount
}

// AppliedTemplateRecord represents a template that has been applied to an instance
type AppliedTemplateRecord struct {
	TemplateName       string    `json:"template_name"`
	AppliedAt          time.Time `json:"applied_at"`
	PackageManager     string    `json:"package_manager"`
	PackagesInstalled  []string  `json:"packages_installed"`
	ServicesConfigured []string  `json:"services_configured"`
	UsersCreated       []string  `json:"users_created"`
	RollbackCheckpoint string    `json:"rollback_checkpoint"`
}

// HibernationStatus represents the hibernation status of an instance
type HibernationStatus struct {
	HibernationSupported bool   `json:"hibernation_supported"`
	InstanceState        string `json:"instance_state"`      // Current AWS state: running, stopped, stopping, etc.
	PossiblyHibernated   bool   `json:"possibly_hibernated"` // True if stopped with hibernation support
	InstanceName         string `json:"instance_name"`
	Note                 string `json:"note,omitempty"` // Explanatory note about the status

	// Deprecated: Use PossiblyHibernated instead
	IsHibernated bool `json:"is_hibernated,omitempty"` // Kept for backward compatibility
}

// Moved to idle_legacy.go - legacy idle system removed

// Moved to idle_legacy.go - legacy idle profile removed

// IdleState represents the idle state of an instance
type IdleState struct {
	InstanceID   string               `json:"instance_id"`
	InstanceName string               `json:"instance_name"`
	Profile      string               `json:"profile"`
	IsIdle       bool                 `json:"is_idle"`
	IdleSince    *time.Time           `json:"idle_since,omitempty"`
	LastActivity time.Time            `json:"last_activity"`
	NextAction   *IdleScheduledAction `json:"next_action,omitempty"`
	LastMetrics  *IdleUsageMetrics    `json:"last_metrics,omitempty"`
}

// IdleScheduledAction represents a scheduled idle action
type IdleScheduledAction struct {
	Action string    `json:"action"`
	Time   time.Time `json:"time"`
}

// IdleUsageMetrics represents usage metrics for idle detection
type IdleUsageMetrics struct {
	Timestamp   time.Time `json:"timestamp"`
	CPU         float64   `json:"cpu"`
	Memory      float64   `json:"memory"`
	Network     float64   `json:"network"`
	Disk        float64   `json:"disk"`
	GPU         *float64  `json:"gpu,omitempty"`
	HasActivity bool      `json:"has_activity"`
}

// IdleHistoryEntry represents an entry in the idle history
type IdleHistoryEntry struct {
	InstanceID   string            `json:"instance_id"`
	InstanceName string            `json:"instance_name"`
	Action       string            `json:"action"`
	Time         time.Time         `json:"time"`
	IdleDuration time.Duration     `json:"idle_duration"`
	Metrics      *IdleUsageMetrics `json:"metrics,omitempty"`
}

// IdleExecutionResponse represents the response from executing idle actions
type IdleExecutionResponse struct {
	Executed int      `json:"executed"`
	Errors   []string `json:"errors"`
	Total    int      `json:"total"`
}
