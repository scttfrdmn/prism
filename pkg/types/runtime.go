package types

import (
	"time"

	"github.com/scttfrdmn/prism/pkg/research"
)

// RuntimeTemplate defines a cloud workstation template for launching instances
// This is the CANONICAL definition - pkg/templates/types.go uses a type alias to this
type RuntimeTemplate struct {
	Name                 string
	Slug                 string // CLI identifier for template (e.g., "python-ml")
	Description          string
	LongDescription      string                       // Detailed description for GUI
	AMI                  map[string]map[string]string // region -> arch -> AMI ID
	InstanceType         map[string]string            // arch -> instance type
	UserData             string                       // Generated installation script
	Ports                []int
	RootVolumeGB         int                            `json:"root_volume_gb"` // Root volume size in GB (default: 20)
	EstimatedCostPerHour map[string]float64             // arch -> cost per hour
	IdleDetection        *IdleDetectionConfig           // Idle detection configuration
	ResearchUser         *research.ResearchUserTemplate `json:"research_user,omitempty"`

	// Complexity and categorization for GUI
	Complexity TemplateComplexity `json:"complexity,omitempty"`
	Category   string             `json:"category,omitempty"`
	Domain     string             `json:"domain,omitempty"`

	// Visual presentation for GUI
	Icon     string `json:"icon,omitempty"`
	Color    string `json:"color,omitempty"`
	Popular  bool   `json:"popular,omitempty"`
	Featured bool   `json:"featured,omitempty"`

	// User guidance for GUI
	EstimatedLaunchTime int      `json:"estimated_launch_time,omitempty"`
	Prerequisites       []string `json:"prerequisites,omitempty"`
	LearningResources   []string `json:"learning_resources,omitempty"`

	// Template metadata for GUI
	ValidationStatus ValidationStatus  `json:"validation_status,omitempty"`
	Tags             map[string]string `json:"tags,omitempty"`
	Maintainer       string            `json:"maintainer,omitempty"`

	// Connection configuration
	ConnectionType ConnectionType `json:"connection_type,omitempty"`

	// Additional metadata from unified template
	Source    interface{} `json:"-"` // Reference to source template (avoid circular import)
	Generated time.Time   // When this runtime template was generated
}

// Service represents a web service running on an instance
type Service struct {
	Name        string `json:"name"`                  // Service name (e.g., "jupyter", "rstudio-server")
	Port        int    `json:"port"`                  // Remote port on instance
	LocalPort   int    `json:"local_port"`            // Local tunnel port (0 = not tunneled)
	Type        string `json:"type,omitempty"`        // Service type: "web", "api", etc.
	URL         string `json:"url,omitempty"`         // Local access URL (e.g., "http://localhost:8787")
	AuthToken   string `json:"auth_token,omitempty"`  // Authentication token if needed
	Status      string `json:"status,omitempty"`      // "running", "stopped", "unknown"
	Description string `json:"description,omitempty"` // Human-readable description
}

// TemplateComplexity represents template complexity level
type TemplateComplexity string

const (
	TemplateComplexitySimple   TemplateComplexity = "simple"   // Ready to use, perfect for getting started
	TemplateComplexityModerate TemplateComplexity = "moderate" // Some customization available, good for regular users
	TemplateComplexityAdvanced TemplateComplexity = "advanced" // Highly configurable, for experienced users
	TemplateComplexityComplex  TemplateComplexity = "complex"  // Maximum flexibility, requires technical knowledge
)

// Level returns the numeric level for sorting (1=simple, 4=complex)
func (c TemplateComplexity) Level() int {
	switch c {
	case TemplateComplexitySimple:
		return 1
	case TemplateComplexityModerate:
		return 2
	case TemplateComplexityAdvanced:
		return 3
	case TemplateComplexityComplex:
		return 4
	default:
		return 1 // Default to simple
	}
}

// Label returns the human-readable label for the complexity level
func (c TemplateComplexity) Label() string {
	switch c {
	case TemplateComplexitySimple:
		return "Simple"
	case TemplateComplexityModerate:
		return "Moderate"
	case TemplateComplexityAdvanced:
		return "Advanced"
	case TemplateComplexityComplex:
		return "Complex"
	default:
		return "Simple"
	}
}

// Badge returns the badge text for GUI display
func (c TemplateComplexity) Badge() string {
	switch c {
	case TemplateComplexitySimple:
		return "Ready to Use"
	case TemplateComplexityModerate:
		return "Some Options"
	case TemplateComplexityAdvanced:
		return "Many Options"
	case TemplateComplexityComplex:
		return "Full Control"
	default:
		return "Ready to Use"
	}
}

// Icon returns the emoji icon for the complexity level
func (c TemplateComplexity) Icon() string {
	switch c {
	case TemplateComplexitySimple:
		return "🟢"
	case TemplateComplexityModerate:
		return "🟡"
	case TemplateComplexityAdvanced:
		return "🟠"
	case TemplateComplexityComplex:
		return "🔴"
	default:
		return "🟢"
	}
}

// Color returns the hex color for the complexity level
func (c TemplateComplexity) Color() string {
	switch c {
	case TemplateComplexitySimple:
		return "#059669"
	case TemplateComplexityModerate:
		return "#d97706"
	case TemplateComplexityAdvanced:
		return "#ea580c"
	case TemplateComplexityComplex:
		return "#dc2626"
	default:
		return "#059669"
	}
}

// ValidationStatus represents template validation status
type ValidationStatus string

const (
	ValidationStatusValid   ValidationStatus = "valid"
	ValidationStatusInvalid ValidationStatus = "invalid"
	ValidationStatusUnknown ValidationStatus = "unknown"
)

// ConnectionType represents how users connect to instances
type ConnectionType string

const (
	ConnectionTypeSSH  ConnectionType = "ssh"
	ConnectionTypeWeb  ConnectionType = "web"
	ConnectionTypeBoth ConnectionType = "both"
)

// IdleDetectionConfig represents idle detection configuration in templates
type IdleDetectionConfig struct {
	Enabled                   bool `yaml:"enabled" json:"enabled"`
	IdleThresholdMinutes      int  `yaml:"idle_threshold_minutes" json:"idle_threshold_minutes"`
	HibernateThresholdMinutes int  `yaml:"hibernate_threshold_minutes" json:"hibernate_threshold_minutes"`
	CheckIntervalMinutes      int  `yaml:"check_interval_minutes" json:"check_interval_minutes"`
}

// Type aliases pointing to canonical definitions in research package
type ResearchUserTemplate = research.ResearchUserTemplate
type DualUserIntegration = research.DualUserIntegration

// Instance represents a running cloud workstation
type Instance struct {
	ID                    string                  `json:"id"`
	Name                  string                  `json:"name"`
	Template              string                  `json:"template"`
	Region                string                  `json:"region"`            // AWS region where instance is running
	AvailabilityZone      string                  `json:"availability_zone"` // AWS availability zone within region
	PublicIP              string                  `json:"public_ip"`
	PrivateIP             string                  `json:"private_ip"`
	State                 string                  `json:"state"`
	LaunchTime            time.Time               `json:"launch_time"`
	RunningStateStartTime *time.Time              `json:"running_state_start_time,omitempty"` // When instance entered running state (billing starts)
	DeletionTime          *time.Time              `json:"deletion_time,omitempty"`            // When user initiated deletion
	HourlyRate            float64                 `json:"hourly_rate"`                        // AWS list price per hour
	CurrentSpend          float64                 `json:"current_spend"`                      // Actual accumulated cost since launch
	EffectiveRate         float64                 `json:"effective_rate"`                     // Current spend ÷ hours since launch
	AttachedVolumes       []string                `json:"attached_volumes"`                   // EFS volume names
	AttachedEBSVolumes    []string                `json:"attached_ebs_volumes"`               // EBS volume IDs
	InstanceType          string                  `json:"instance_type"`
	InstanceLifecycle     string                  `json:"instance_lifecycle"` // "spot" or "on-demand"
	KeyName               string                  `json:"key_name"`           // EC2 key pair name
	Username              string                  `json:"username"`
	WebPort               int                     `json:"web_port"`             // Deprecated: Use Services instead
	HasWebInterface       bool                    `json:"has_web_interface"`    // Deprecated: Use Services instead
	Services              []Service               `json:"services,omitempty"`   // Web services available on this instance
	ProjectID             string                  `json:"project_id,omitempty"` // Associated project ID
	IdleDetection         *IdleDetection          `json:"idle_detection,omitempty"`
	AppliedTemplates      []AppliedTemplateRecord `json:"applied_templates,omitempty"` // Template application history

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

	// State transition history for accurate cost tracking
	StateHistory []StateTransition `json:"state_history,omitempty"` // History of all state changes
}

// StateTransition records when an instance changes state for cost tracking
// This enables accurate billing calculations based on actual runtime vs stopped time
type StateTransition struct {
	FromState string    `json:"from_state"`          // Previous state (or empty for launch)
	ToState   string    `json:"to_state"`            // New state
	Timestamp time.Time `json:"timestamp"`           // When transition occurred
	Reason    string    `json:"reason,omitempty"`    // Why transition happened (user action, idle detection, etc.)
	Initiator string    `json:"initiator,omitempty"` // Who/what initiated (user, system, policy)
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

// =============================================================================
// Rightsizing Types
// =============================================================================

// InstanceMetrics represents collected performance metrics for an instance
type InstanceMetrics struct {
	InstanceID   string         `json:"instance_id"`
	InstanceName string         `json:"instance_name"`
	Timestamp    time.Time      `json:"timestamp"`
	CPU          CPUMetrics     `json:"cpu"`
	Memory       MemoryMetrics  `json:"memory"`
	Storage      StorageMetrics `json:"storage"`
	Network      NetworkMetrics `json:"network"`
	GPU          *GPUMetrics    `json:"gpu,omitempty"`
	System       SystemMetrics  `json:"system"`
}

// CPUMetrics represents CPU performance metrics
type CPUMetrics struct {
	UtilizationPercent float64 `json:"utilization_percent"`
	Load1Min           float64 `json:"load_1min"`
	Load5Min           float64 `json:"load_5min"`
	Load15Min          float64 `json:"load_15min"`
	CoreCount          int     `json:"core_count"`
	IdlePercent        float64 `json:"idle_percent"`
	WaitPercent        float64 `json:"wait_percent"`
}

// MemoryMetrics represents memory performance metrics
type MemoryMetrics struct {
	TotalMB            float64 `json:"total_mb"`
	UsedMB             float64 `json:"used_mb"`
	FreeMB             float64 `json:"free_mb"`
	AvailableMB        float64 `json:"available_mb"`
	CachedMB           float64 `json:"cached_mb"`
	BufferedMB         float64 `json:"buffered_mb"`
	UtilizationPercent float64 `json:"utilization_percent"`
	SwapTotalMB        float64 `json:"swap_total_mb"`
	SwapUsedMB         float64 `json:"swap_used_mb"`
}

// StorageMetrics represents storage performance metrics
type StorageMetrics struct {
	TotalGB             float64 `json:"total_gb"`
	UsedGB              float64 `json:"used_gb"`
	AvailableGB         float64 `json:"available_gb"`
	UtilizationPercent  float64 `json:"utilization_percent"`
	ReadIOPS            float64 `json:"read_iops"`
	WriteIOPS           float64 `json:"write_iops"`
	ReadThroughputMBps  float64 `json:"read_throughput_mbps"`
	WriteThroughputMBps float64 `json:"write_throughput_mbps"`
}

// NetworkMetrics represents network performance metrics
type NetworkMetrics struct {
	RxBytesPerSec   float64 `json:"rx_bytes_per_sec"`
	TxBytesPerSec   float64 `json:"tx_bytes_per_sec"`
	RxPacketsPerSec float64 `json:"rx_packets_per_sec"`
	TxPacketsPerSec float64 `json:"tx_packets_per_sec"`
	TotalRxBytes    float64 `json:"total_rx_bytes"`
	TotalTxBytes    float64 `json:"total_tx_bytes"`
}

// GPUMetrics represents GPU performance metrics (optional)
type GPUMetrics struct {
	Count                    int     `json:"count"`
	UtilizationPercent       float64 `json:"utilization_percent"`
	MemoryTotalMB            float64 `json:"memory_total_mb"`
	MemoryUsedMB             float64 `json:"memory_used_mb"`
	MemoryUtilizationPercent float64 `json:"memory_utilization_percent"`
	TemperatureCelsius       float64 `json:"temperature_celsius"`
	PowerDrawWatts           float64 `json:"power_draw_watts"`
}

// SystemMetrics represents system-level metrics
type SystemMetrics struct {
	ProcessCount    int       `json:"process_count"`
	LoggedInUsers   int       `json:"logged_in_users"`
	UptimeSeconds   float64   `json:"uptime_seconds"`
	LastActivity    time.Time `json:"last_activity"`
	LoadAverage1Min float64   `json:"load_average_1min"`
}

// RightsizingRecommendation represents a rightsizing recommendation
type RightsizingRecommendation struct {
	InstanceID              string           `json:"instance_id"`
	InstanceName            string           `json:"instance_name"`
	CurrentInstanceType     string           `json:"current_instance_type"`
	CurrentSize             string           `json:"current_size"`
	RecommendedInstanceType string           `json:"recommended_instance_type"`
	RecommendedSize         string           `json:"recommended_size"`
	RecommendationType      RightsizingType  `json:"recommendation_type"`
	Confidence              ConfidenceLevel  `json:"confidence"`
	Reasoning               string           `json:"reasoning"`
	CostImpact              CostImpact       `json:"cost_impact"`
	ResourceAnalysis        ResourceAnalysis `json:"resource_analysis"`
	CreatedAt               time.Time        `json:"created_at"`
	DataPointsAnalyzed      int              `json:"data_points_analyzed"`
	AnalysisPeriodHours     float64          `json:"analysis_period_hours"`
}

// RightsizingType represents the type of rightsizing recommendation
type RightsizingType string

const (
	RightsizingDownsize         RightsizingType = "downsize"
	RightsizingUpsize           RightsizingType = "upsize"
	RightsizingOptimal          RightsizingType = "optimal"
	RightsizingMemoryOptimized  RightsizingType = "memory_optimized"
	RightsizingComputeOptimized RightsizingType = "compute_optimized"
	RightsizingGPUOptimized     RightsizingType = "gpu_optimized"
)

// ConfidenceLevel represents the confidence level of a recommendation
type ConfidenceLevel string

const (
	ConfidenceLow      ConfidenceLevel = "low"
	ConfidenceMedium   ConfidenceLevel = "medium"
	ConfidenceHigh     ConfidenceLevel = "high"
	ConfidenceVeryHigh ConfidenceLevel = "very_high"
)

// CostImpact represents the cost impact of a rightsizing recommendation
type CostImpact struct {
	CurrentDailyCost     float64 `json:"current_daily_cost"`
	RecommendedDailyCost float64 `json:"recommended_daily_cost"`
	DailyDifference      float64 `json:"daily_difference"`
	PercentageChange     float64 `json:"percentage_change"`
	MonthlySavings       float64 `json:"monthly_savings"`
	AnnualSavings        float64 `json:"annual_savings"`
	IsIncrease           bool    `json:"is_increase"`
	PaybackPeriodDays    float64 `json:"payback_period_days,omitempty"`
}

// ResourceAnalysis represents detailed resource utilization analysis
type ResourceAnalysis struct {
	CPUAnalysis     CPUAnalysis     `json:"cpu_analysis"`
	MemoryAnalysis  MemoryAnalysis  `json:"memory_analysis"`
	StorageAnalysis StorageAnalysis `json:"storage_analysis"`
	NetworkAnalysis NetworkAnalysis `json:"network_analysis"`
	GPUAnalysis     *GPUAnalysis    `json:"gpu_analysis,omitempty"`
	WorkloadPattern WorkloadPattern `json:"workload_pattern"`
}

// CPUAnalysis represents CPU utilization analysis
type CPUAnalysis struct {
	AverageUtilization float64 `json:"average_utilization"`
	PeakUtilization    float64 `json:"peak_utilization"`
	P95Utilization     float64 `json:"p95_utilization"`
	P99Utilization     float64 `json:"p99_utilization"`
	IdlePercentage     float64 `json:"idle_percentage"`
	IsBottleneck       bool    `json:"is_bottleneck"`
	Recommendation     string  `json:"recommendation"`
}

// MemoryAnalysis represents memory utilization analysis
type MemoryAnalysis struct {
	AverageUtilization float64 `json:"average_utilization"`
	PeakUtilization    float64 `json:"peak_utilization"`
	P95Utilization     float64 `json:"p95_utilization"`
	P99Utilization     float64 `json:"p99_utilization"`
	SwapUsage          float64 `json:"swap_usage"`
	IsBottleneck       bool    `json:"is_bottleneck"`
	Recommendation     string  `json:"recommendation"`
}

// StorageAnalysis represents storage utilization analysis
type StorageAnalysis struct {
	AverageIOPS       float64 `json:"average_iops"`
	PeakIOPS          float64 `json:"peak_iops"`
	AverageThroughput float64 `json:"average_throughput"`
	PeakThroughput    float64 `json:"peak_throughput"`
	SpaceUtilization  float64 `json:"space_utilization"`
	IsBottleneck      bool    `json:"is_bottleneck"`
	Recommendation    string  `json:"recommendation"`
}

// NetworkAnalysis represents network utilization analysis
type NetworkAnalysis struct {
	AverageThroughput float64 `json:"average_throughput"`
	PeakThroughput    float64 `json:"peak_throughput"`
	PacketRate        float64 `json:"packet_rate"`
	IsBottleneck      bool    `json:"is_bottleneck"`
	Recommendation    string  `json:"recommendation"`
}

// GPUAnalysis represents GPU utilization analysis
type GPUAnalysis struct {
	AverageUtilization float64 `json:"average_utilization"`
	PeakUtilization    float64 `json:"peak_utilization"`
	MemoryUtilization  float64 `json:"memory_utilization"`
	TemperatureAverage float64 `json:"temperature_average"`
	PowerUsageAverage  float64 `json:"power_usage_average"`
	IsBottleneck       bool    `json:"is_bottleneck"`
	IsUnderutilized    bool    `json:"is_underutilized"`
	Recommendation     string  `json:"recommendation"`
}

// WorkloadPattern represents workload pattern analysis
type WorkloadPattern struct {
	Type                WorkloadPatternType `json:"type"`
	ConsistencyScore    float64             `json:"consistency_score"`
	PeakHours           []int               `json:"peak_hours"`
	SeasonalityDetected bool                `json:"seasonality_detected"`
	GrowthTrend         float64             `json:"growth_trend"`
	BurstFrequency      float64             `json:"burst_frequency"`
	Description         string              `json:"description"`
}

// WorkloadPatternType represents different types of workload patterns
type WorkloadPatternType string

const (
	WorkloadPatternSteady        WorkloadPatternType = "steady"
	WorkloadPatternBursty        WorkloadPatternType = "bursty"
	WorkloadPatternSeasonal      WorkloadPatternType = "seasonal"
	WorkloadPatternGrowing       WorkloadPatternType = "growing"
	WorkloadPatternDeclining     WorkloadPatternType = "declining"
	WorkloadPatternUnpredictable WorkloadPatternType = "unpredictable"
)

// RightsizingAnalysisRequest represents a request for rightsizing analysis
type RightsizingAnalysisRequest struct {
	InstanceName        string  `json:"instance_name"`
	AnalysisPeriodHours float64 `json:"analysis_period_hours,omitempty"`
	IncludeDetails      bool    `json:"include_details,omitempty"`
	ForceRefresh        bool    `json:"force_refresh,omitempty"`
}

// RightsizingAnalysisResponse represents the response from rightsizing analysis
type RightsizingAnalysisResponse struct {
	Recommendation      *RightsizingRecommendation `json:"recommendation"`
	MetricsAvailable    bool                       `json:"metrics_available"`
	DataPointsCount     int                        `json:"data_points_count"`
	AnalysisPeriodHours float64                    `json:"analysis_period_hours"`
	LastUpdated         time.Time                  `json:"last_updated"`
	Message             string                     `json:"message,omitempty"`
}

// RightsizingStatsResponse represents detailed statistics for an instance
type RightsizingStatsResponse struct {
	InstanceName         string                     `json:"instance_name"`
	CurrentConfiguration InstanceConfiguration      `json:"current_configuration"`
	MetricsSummary       MetricsSummary             `json:"metrics_summary"`
	RecentMetrics        []InstanceMetrics          `json:"recent_metrics"`
	Recommendation       *RightsizingRecommendation `json:"recommendation,omitempty"`
	CollectionStatus     MetricsCollectionStatus    `json:"collection_status"`
}

// InstanceConfiguration represents current instance configuration
type InstanceConfiguration struct {
	InstanceType       string  `json:"instance_type"`
	Size               string  `json:"size"`
	VCPUs              int     `json:"vcpus"`
	MemoryGB           float64 `json:"memory_gb"`
	StorageGB          float64 `json:"storage_gb"`
	NetworkPerformance string  `json:"network_performance"`
	DailyCost          float64 `json:"daily_cost"`
}

// MetricsSummary represents aggregated metrics summary
type MetricsSummary struct {
	CPUSummary     ResourceSummary  `json:"cpu_summary"`
	MemorySummary  ResourceSummary  `json:"memory_summary"`
	StorageSummary ResourceSummary  `json:"storage_summary"`
	NetworkSummary ResourceSummary  `json:"network_summary"`
	GPUSummary     *ResourceSummary `json:"gpu_summary,omitempty"`
}

// ResourceSummary represents summary statistics for a resource
type ResourceSummary struct {
	Average           float64 `json:"average"`
	Peak              float64 `json:"peak"`
	P95               float64 `json:"p95"`
	P99               float64 `json:"p99"`
	Minimum           float64 `json:"minimum"`
	StandardDeviation float64 `json:"standard_deviation"`
	TrendDirection    string  `json:"trend_direction"` // "increasing", "decreasing", "stable"
	Bottleneck        bool    `json:"bottleneck"`
	Underutilized     bool    `json:"underutilized"`
}

// MetricsCollectionStatus represents the status of metrics collection
type MetricsCollectionStatus struct {
	IsActive           bool      `json:"is_active"`
	LastCollectionTime time.Time `json:"last_collection_time"`
	CollectionInterval string    `json:"collection_interval"`
	TotalDataPoints    int       `json:"total_data_points"`
	DataRetentionDays  int       `json:"data_retention_days"`
	StorageLocation    string    `json:"storage_location"`
}

// RightsizingRecommendationsResponse represents multiple recommendations
type RightsizingRecommendationsResponse struct {
	Recommendations  []RightsizingRecommendation `json:"recommendations"`
	TotalInstances   int                         `json:"total_instances"`
	ActiveInstances  int                         `json:"active_instances"`
	PotentialSavings float64                     `json:"potential_savings"`
	GeneratedAt      time.Time                   `json:"generated_at"`
}

// RightsizingSummaryResponse represents fleet-wide rightsizing summary
type RightsizingSummaryResponse struct {
	FleetOverview       FleetOverview              `json:"fleet_overview"`
	CostOptimization    CostOptimizationSummary    `json:"cost_optimization"`
	ResourceUtilization ResourceUtilizationSummary `json:"resource_utilization"`
	Recommendations     RecommendationsSummary     `json:"recommendations"`
	GeneratedAt         time.Time                  `json:"generated_at"`
}

// FleetOverview represents overview of all instances
type FleetOverview struct {
	TotalInstances       int     `json:"total_instances"`
	RunningInstances     int     `json:"running_instances"`
	StoppedInstances     int     `json:"stopped_instances"`
	TotalDailyCost       float64 `json:"total_daily_cost"`
	TotalMonthlyCost     float64 `json:"total_monthly_cost"`
	InstancesWithMetrics int     `json:"instances_with_metrics"`
}

// CostOptimizationSummary represents cost optimization opportunities
type CostOptimizationSummary struct {
	PotentialDailySavings         float64 `json:"potential_daily_savings"`
	PotentialMonthlySavings       float64 `json:"potential_monthly_savings"`
	PotentialAnnualSavings        float64 `json:"potential_annual_savings"`
	SavingsPercentage             float64 `json:"savings_percentage"`
	OverprovisionedInstances      int     `json:"overprovisioned_instances"`
	UnderprovisionedInstances     int     `json:"underprovisioned_instances"`
	OptimallyProvisionedInstances int     `json:"optimally_provisioned_instances"`
}

// ResourceUtilizationSummary represents fleet-wide resource utilization
type ResourceUtilizationSummary struct {
	AverageCPUUtilization     float64 `json:"average_cpu_utilization"`
	AverageMemoryUtilization  float64 `json:"average_memory_utilization"`
	AverageStorageUtilization float64 `json:"average_storage_utilization"`
	InstancesWithLowCPU       int     `json:"instances_with_low_cpu"`
	InstancesWithHighCPU      int     `json:"instances_with_high_cpu"`
	InstancesWithLowMemory    int     `json:"instances_with_low_memory"`
	InstancesWithHighMemory   int     `json:"instances_with_high_memory"`
}

// RecommendationsSummary represents summary of recommendations
type RecommendationsSummary struct {
	TotalRecommendations    int `json:"total_recommendations"`
	DownsizeRecommendations int `json:"downsize_recommendations"`
	UpsizeRecommendations   int `json:"upsize_recommendations"`
	OptimizeRecommendations int `json:"optimize_recommendations"`
	HighConfidenceCount     int `json:"high_confidence_count"`
	MediumConfidenceCount   int `json:"medium_confidence_count"`
	LowConfidenceCount      int `json:"low_confidence_count"`
}
