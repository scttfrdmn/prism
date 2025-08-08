package types

import "time"

// RuntimeTemplate defines a cloud workstation template for launching instances
// This is distinct from AMI build templates (see pkg/ami package)
type RuntimeTemplate struct {
	Name         string
	Description  string
	AMI          map[string]map[string]string // region -> arch -> AMI ID
	InstanceType map[string]string            // arch -> instance type
	UserData     string
	Ports        []int
	EstimatedCostPerHour map[string]float64 // arch -> cost per hour
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
	EstimatedDailyCost float64                 `json:"estimated_daily_cost"`
	AttachedVolumes    []string                `json:"attached_volumes"`     // EFS volume names
	AttachedEBSVolumes []string                `json:"attached_ebs_volumes"` // EBS volume IDs
	InstanceType       string                  `json:"instance_type"`
	InstanceLifecycle  string                  `json:"instance_lifecycle"`  // "spot" or "on-demand"
	Username           string                  `json:"username"`
	WebPort            int                     `json:"web_port"`
	HasWebInterface    bool                    `json:"has_web_interface"`
	ProjectID          string                  `json:"project_id,omitempty"` // Associated project ID
	IdleDetection      *IdleDetection          `json:"idle_detection,omitempty"`
	AppliedTemplates   []AppliedTemplateRecord `json:"applied_templates,omitempty"` // Template application history
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
	CreditType       string     `json:"credit_type"`  // "AWS Promotional", "AWS Educate", etc.
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
	EC2Discount         float64 `json:"ec2_discount"`         // Percentage discount (0.0-1.0)
	EBSDiscount         float64 `json:"ebs_discount"`         // Percentage discount (0.0-1.0)
	EFSDiscount         float64 `json:"efs_discount"`         // Percentage discount (0.0-1.0)
	SavingsPlansDiscount float64 `json:"savings_plans_discount"` // Additional savings plan discount
	ReservedInstanceDiscount float64 `json:"reserved_instance_discount"` // RI discount
	SpotDiscount        float64 `json:"spot_discount"`        // Spot instance discount
	VolumeDiscount      float64 `json:"volume_discount"`      // Volume discount for large usage
	EducationalDiscount float64 `json:"educational_discount"` // Educational institution discount
	StartupDiscount     float64 `json:"startup_discount"`     // AWS Activate/startup credits
	EnterpriseDiscount  float64 `json:"enterprise_discount"`  // Enterprise agreement discount
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
	IsHibernated        bool   `json:"is_hibernated"`
	InstanceName        string `json:"instance_name"`
}

// IdleStatusResponse represents the response from GetIdleStatus
type IdleStatusResponse struct {
	Enabled        bool                    `json:"enabled"`
	DefaultProfile string                  `json:"default_profile"`
	Profiles       map[string]IdleProfile  `json:"profiles"`
	DomainMappings map[string]string       `json:"domain_mappings"`
}

// IdleProfile represents an idle detection profile
type IdleProfile struct {
	Name             string  `json:"name"`
	CPUThreshold     float64 `json:"cpu_threshold"`
	MemoryThreshold  float64 `json:"memory_threshold"`
	NetworkThreshold float64 `json:"network_threshold"`
	DiskThreshold    float64 `json:"disk_threshold"`
	GPUThreshold     float64 `json:"gpu_threshold"`
	IdleMinutes      int     `json:"idle_minutes"`
	Action           string  `json:"action"`
	Notification     bool    `json:"notification"`
}

// IdleState represents the idle state of an instance
type IdleState struct {
	InstanceID   string                  `json:"instance_id"`
	InstanceName string                  `json:"instance_name"`
	Profile      string                  `json:"profile"`
	IsIdle       bool                    `json:"is_idle"`
	IdleSince    *time.Time              `json:"idle_since,omitempty"`
	LastActivity time.Time               `json:"last_activity"`
	NextAction   *IdleScheduledAction    `json:"next_action,omitempty"`
	LastMetrics  *IdleUsageMetrics       `json:"last_metrics,omitempty"`
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
	InstanceID   string             `json:"instance_id"`
	InstanceName string             `json:"instance_name"`
	Action       string             `json:"action"`
	Time         time.Time          `json:"time"`
	IdleDuration time.Duration      `json:"idle_duration"`
	Metrics      *IdleUsageMetrics  `json:"metrics,omitempty"`
}

// IdleExecutionResponse represents the response from executing idle actions
type IdleExecutionResponse struct {
	Executed int      `json:"executed"`
	Errors   []string `json:"errors"`
	Total    int      `json:"total"`
}