package types

import "time"

// Template defines a cloud workstation template
type Template struct {
	Name         string
	Description  string
	AMI          map[string]map[string]string // region -> arch -> AMI ID
	InstanceType map[string]string            // arch -> instance type
	UserData     string
	Ports        []int
	EstimatedCostPerHour map[string]float64 // arch -> cost per hour
}

// Config manages application configuration
type Config struct {
	DefaultProfile string `json:"default_profile"`
	DefaultRegion  string `json:"default_region"`
	APIKey        string `json:"api_key,omitempty"`
	APIKeyCreated time.Time `json:"api_key_created,omitempty"`
}

// Instance represents a running cloud workstation
type Instance struct {
	ID                 string          `json:"id"`
	Name               string          `json:"name"`
	Template           string          `json:"template"`
	PublicIP           string          `json:"public_ip"`
	PrivateIP          string          `json:"private_ip"`
	State              string          `json:"state"`
	LaunchTime         time.Time       `json:"launch_time"`
	EstimatedDailyCost float64         `json:"estimated_daily_cost"`
	AttachedVolumes    []string        `json:"attached_volumes"`     // EFS volume names
	AttachedEBSVolumes []string        `json:"attached_ebs_volumes"` // EBS volume IDs
	InstanceType       string          `json:"instance_type"`
	Username           string          `json:"username"`
	WebPort            int             `json:"web_port"`
	HasWebInterface    bool            `json:"has_web_interface"`
	IdleDetection      *IdleDetection  `json:"idle_detection,omitempty"`
}

// EFSVolume represents a persistent EFS file system
type EFSVolume struct {
	Name            string    `json:"name"`              // User-friendly name
	FileSystemId    string    `json:"filesystem_id"`     // AWS EFS ID
	Region          string    `json:"region"`
	CreationTime    time.Time `json:"creation_time"`
	MountTargets    []string  `json:"mount_targets"`     // Mount target IDs
	State           string    `json:"state"`             // available, creating, deleting
	PerformanceMode string    `json:"performance_mode"`  // generalPurpose, maxIO
	ThroughputMode  string    `json:"throughput_mode"`   // bursting, provisioned
	EstimatedCostGB float64   `json:"estimated_cost_gb"` // $/GB/month
	SizeBytes       int64     `json:"size_bytes"`        // Current size
}

// EBSVolume represents a secondary EBS volume for high-performance storage
type EBSVolume struct {
	Name            string    `json:"name"`              // User-friendly name
	VolumeID        string    `json:"volume_id"`         // AWS EBS volume ID
	Region          string    `json:"region"`
	CreationTime    time.Time `json:"creation_time"`
	State           string    `json:"state"`             // available, creating, in-use, deleting
	VolumeType      string    `json:"volume_type"`       // gp3, io2, etc.
	SizeGB          int32     `json:"size_gb"`           // Volume size in GB
	IOPS            int32     `json:"iops"`              // Provisioned IOPS (for io2)
	Throughput      int32     `json:"throughput"`        // Throughput in MB/s (for gp3)
	EstimatedCostGB float64   `json:"estimated_cost_gb"` // $/GB/month
	AttachedTo      string    `json:"attached_to"`       // Instance name if attached
}

// State manages the application state
type State struct {
	Instances  map[string]Instance  `json:"instances"`
	Volumes    map[string]EFSVolume `json:"volumes"`
	EBSVolumes map[string]EBSVolume `json:"ebs_volumes"`
	Config     Config               `json:"config"`
}

// API Request/Response types

// LaunchRequest represents a request to launch an instance
type LaunchRequest struct {
	Template    string   `json:"template"`
	Name        string   `json:"name"`
	Size        string   `json:"size,omitempty"`        // XS, S, M, L, XL, GPU-S, etc.
	Volumes     []string `json:"volumes,omitempty"`     // EFS volume names to attach
	EBSVolumes  []string `json:"ebs_volumes,omitempty"` // EBS volume IDs to attach
	Region      string   `json:"region,omitempty"`
	SubnetID    string   `json:"subnet_id,omitempty"`
	VpcID       string   `json:"vpc_id,omitempty"`
	Spot        bool     `json:"spot,omitempty"`
	DryRun      bool     `json:"dry_run,omitempty"`
}

// LaunchResponse represents a successful launch response
type LaunchResponse struct {
	Instance       Instance `json:"instance"`
	Message        string   `json:"message"`
	EstimatedCost  string   `json:"estimated_cost"`
	ConnectionInfo string   `json:"connection_info"`
}

// ListResponse represents a list of instances
type ListResponse struct {
	Instances []Instance `json:"instances"`
	TotalCost float64    `json:"total_cost"`
}

// VolumeCreateRequest represents a request to create an EFS volume
type VolumeCreateRequest struct {
	Name            string `json:"name"`
	PerformanceMode string `json:"performance_mode,omitempty"` // generalPurpose, maxIO
	ThroughputMode  string `json:"throughput_mode,omitempty"`  // bursting, provisioned
	Region          string `json:"region,omitempty"`
}

// StorageCreateRequest represents a request to create an EBS volume
type StorageCreateRequest struct {
	Name       string `json:"name"`
	Size       string `json:"size"`        // XS, S, M, L, XL or specific GB
	VolumeType string `json:"volume_type"` // gp3, io2
	Region     string `json:"region,omitempty"`
}

// SimpleAPIError represents a simple API error response (legacy)
type SimpleAPIError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Details string `json:"details,omitempty"`
}

func (e SimpleAPIError) Error() string {
	return e.Message
}

// DaemonStatus represents the daemon's current status
type DaemonStatus struct {
	Version       string    `json:"version"`
	Status        string    `json:"status"`
	StartTime     time.Time `json:"start_time"`
	ActiveOps     int       `json:"active_operations"`
	TotalRequests int64     `json:"total_requests"`
	AWSRegion     string    `json:"aws_region"`
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

// IdleDetection represents idle detection configuration for an instance
type IdleDetection struct {
	Enabled        bool      `json:"enabled"`
	Policy         string    `json:"policy"`
	IdleTime       int       `json:"idle_time"`       // Minutes
	Threshold      int       `json:"threshold"`       // Minutes
	ActionSchedule time.Time `json:"action_schedule"` // When action will occur
	ActionPending  bool      `json:"action_pending"`  // Whether action is pending
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

// AuthResponse represents an authentication response
type AuthResponse struct {
	APIKey       string    `json:"api_key"`
	CreatedAt    time.Time `json:"created_at"`
	ExpiresAt    time.Time `json:"expires_at,omitempty"`
	Message      string    `json:"message"`
}