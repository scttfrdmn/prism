// Package types provides AMI-related data structures for the Universal AMI System
package types

import (
	"time"
)

// AMIResolutionResult represents the result of AMI resolution for a template
type AMIResolutionResult struct {
	// Resolved AMI information
	AMI *AMIInfo `json:"ami,omitempty"`

	// How the AMI was resolved
	ResolutionMethod AMIResolutionMethod `json:"resolution_method"`

	// Chain of fallback methods attempted
	FallbackChain []string `json:"fallback_chain"`

	// User warning about resolution (if applicable)
	Warning string `json:"warning,omitempty"`

	// Cost analysis
	EstimatedCost float64       `json:"estimated_cost"`
	LaunchTime    time.Duration `json:"launch_time"`
	CostSavings   float64       `json:"cost_savings,omitempty"` // vs script provisioning

	// Region information
	TargetRegion string `json:"target_region"`
	SourceRegion string `json:"source_region,omitempty"` // Different if cross-region copy
}

// AMIResolutionMethod indicates how an AMI was resolved
type AMIResolutionMethod string

const (
	ResolutionDirectMapping   AMIResolutionMethod = "direct_mapping"
	ResolutionDynamicSearch   AMIResolutionMethod = "dynamic_search"
	ResolutionMarketplace     AMIResolutionMethod = "marketplace"
	ResolutionCrossRegion     AMIResolutionMethod = "cross_region"
	ResolutionFallbackScript  AMIResolutionMethod = "fallback_script"
	ResolutionFailed          AMIResolutionMethod = "failed"
)

// AMIInfo contains detailed information about an AMI
type AMIInfo struct {
	// Basic AMI identification
	AMIID        string `json:"ami_id"`
	Name         string `json:"name"`
	Description  string `json:"description"`
	Region       string `json:"region"`
	Architecture string `json:"architecture"` // x86_64, arm64

	// AMI metadata
	Owner         string            `json:"owner"`
	CreationDate  time.Time         `json:"creation_date"`
	State         string            `json:"state"`
	Public        bool              `json:"public"`
	Tags          map[string]string `json:"tags,omitempty"`

	// Performance and cost information
	LaunchTime      time.Duration `json:"launch_time"`        // Expected launch time
	MarketplaceCost float64       `json:"marketplace_cost"`   // Marketplace hourly cost (if applicable)
	StorageCost     float64       `json:"storage_cost"`       // Monthly storage cost
	SourceRegion    string        `json:"source_region"`      // Different from Region if copied

	// Community AMI information (if applicable)
	CommunityInfo *CommunityAMIInfo `json:"community_info,omitempty"`

	// Validation and security
	SignatureValid bool      `json:"signature_valid"`
	SecurityScore  float64   `json:"security_score"` // 0.0-10.0
	LastTested     time.Time `json:"last_tested"`
}

// CommunityAMIInfo contains community-specific AMI metadata
type CommunityAMIInfo struct {
	Creator       string    `json:"creator"`
	Version       string    `json:"version"`
	Rating        float64   `json:"rating"`        // 0.0-5.0
	ReviewCount   int       `json:"review_count"`
	DownloadCount int       `json:"download_count"`
	Verified      bool      `json:"verified"`
	LastUpdated   time.Time `json:"last_updated"`
	Reviews       []AMIReview `json:"reviews,omitempty"`
}

// AMIReview represents a community review of an AMI
type AMIReview struct {
	UserID      string    `json:"user_id"`
	Rating      int       `json:"rating"`      // 1-5 stars
	Review      string    `json:"review"`
	Helpful     int       `json:"helpful"`     // Helpful votes
	CreatedAt   time.Time `json:"created_at"`
}

// AMICreationRequest represents a request to create a new AMI
type AMICreationRequest struct {
	// Source instance information
	InstanceID string `json:"instance_id"`
	TemplateName string `json:"template_name"`

	// AMI metadata
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Version     string            `json:"version,omitempty"`
	Tags        map[string]string `json:"tags,omitempty"`

	// Sharing configuration
	Public      bool     `json:"public"`
	ShareWith   []string `json:"share_with,omitempty"`   // AWS account IDs
	Community   string   `json:"community,omitempty"`    // Community to share with

	// Multi-region deployment
	MultiRegion     []string `json:"multi_region,omitempty"`
	RegionPriority  []string `json:"region_priority,omitempty"` // Preferred order for deployment

	// Creation options
	NoReboot        bool   `json:"no_reboot,omitempty"`
	BlockDeviceMapping bool `json:"block_device_mapping,omitempty"`
}

// AMICreationResult represents the result of AMI creation
type AMICreationResult struct {
	// Primary AMI information
	AMIID         string                     `json:"ami_id"`
	Name          string                     `json:"name"`
	CreationTime  time.Duration             `json:"creation_time"`
	Status        AMICreationStatus         `json:"status"`

	// Multi-region deployment results
	RegionResults map[string]*RegionAMIResult `json:"region_results,omitempty"`

	// Community sharing results
	CommunitySharing *CommunityShareResult `json:"community_sharing,omitempty"`

	// Cost information
	StorageCost     float64 `json:"storage_cost"`     // Monthly storage cost
	CreationCost    float64 `json:"creation_cost"`    // One-time creation cost
	TransferCost    float64 `json:"transfer_cost"`    // Multi-region copy cost
}

// AMICreationStatus represents the status of AMI creation
type AMICreationStatus string

const (
	AMICreationPending    AMICreationStatus = "pending"
	AMICreationInProgress AMICreationStatus = "in_progress"
	AMICreationCompleted  AMICreationStatus = "completed"
	AMICreationFailed     AMICreationStatus = "failed"
)

// RegionAMIResult represents AMI deployment result in a specific region
type RegionAMIResult struct {
	Region      string            `json:"region"`
	AMIID       string            `json:"ami_id"`
	Status      AMICreationStatus `json:"status"`
	Error       string            `json:"error,omitempty"`
	CopyTime    time.Duration     `json:"copy_time,omitempty"`
	CopyCost    float64           `json:"copy_cost,omitempty"`
}

// CommunityShareResult represents the result of sharing an AMI with community
type CommunityShareResult struct {
	Community   string    `json:"community"`
	Status      string    `json:"status"`      // submitted, approved, rejected
	SubmittedAt time.Time `json:"submitted_at"`
	ReviewURL   string    `json:"review_url,omitempty"`
	Error       string    `json:"error,omitempty"`
}

// AMITestRequest represents a request to test AMI availability
type AMITestRequest struct {
	TemplateName string   `json:"template_name"`
	Regions      []string `json:"regions,omitempty"`      // Test specific regions, empty = all
	TestLaunch   bool     `json:"test_launch,omitempty"`  // Actually launch instance for testing
}

// AMITestResult represents the result of AMI availability testing
type AMITestResult struct {
	TemplateName    string                      `json:"template_name"`
	OverallStatus   AMITestStatus              `json:"overall_status"`
	RegionResults   map[string]*RegionTestResult `json:"region_results"`
	TestedAt        time.Time                  `json:"tested_at"`
	TotalRegions    int                        `json:"total_regions"`
	AvailableRegions int                       `json:"available_regions"`
}

// AMITestStatus represents overall AMI test status
type AMITestStatus string

const (
	AMITestStatusPassed    AMITestStatus = "passed"    // Available in all requested regions
	AMITestStatusPartial   AMITestStatus = "partial"   // Available in some regions
	AMITestStatusFailed    AMITestStatus = "failed"    // Not available in any region
)

// RegionTestResult represents AMI test result for a specific region
type RegionTestResult struct {
	Region           string                `json:"region"`
	Status           AMITestStatus         `json:"status"`
	AMI              *AMIInfo             `json:"ami,omitempty"`
	ResolutionMethod AMIResolutionMethod  `json:"resolution_method,omitempty"`
	LaunchTest       *LaunchTestResult    `json:"launch_test,omitempty"`
	Error            string               `json:"error,omitempty"`
	TestDuration     time.Duration        `json:"test_duration"`
}

// LaunchTestResult represents the result of actual launch testing
type LaunchTestResult struct {
	InstanceID    string        `json:"instance_id,omitempty"`
	LaunchTime    time.Duration `json:"launch_time"`
	Success       bool          `json:"success"`
	Error         string        `json:"error,omitempty"`
	CleanedUp     bool          `json:"cleaned_up"`
}

// AMICostAnalysis represents cost analysis for AMI vs script deployment
type AMICostAnalysis struct {
	TemplateName string  `json:"template_name"`
	Region       string  `json:"region"`

	// AMI deployment costs
	AMILaunchCost    float64 `json:"ami_launch_cost"`     // Per hour
	AMIStorageCost   float64 `json:"ami_storage_cost"`    // Per month
	AMISetupCost     float64 `json:"ami_setup_cost"`      // One-time (minimal)

	// Script deployment costs
	ScriptLaunchCost  float64 `json:"script_launch_cost"`  // Per hour (same as AMI)
	ScriptSetupCost   float64 `json:"script_setup_cost"`   // One-time setup cost (higher)
	ScriptSetupTime   int     `json:"script_setup_time"`   // Setup time in minutes

	// Cost comparison
	BreakEvenPoint   float64 `json:"break_even_point"`   // Hours where costs are equal
	CostSavings1Hour float64 `json:"cost_savings_1h"`    // Savings for 1-hour session
	CostSavings8Hour float64 `json:"cost_savings_8h"`    // Savings for 8-hour session
	TimeSavings      int     `json:"time_savings"`       // Time saved in minutes

	// Recommendations
	Recommendation string `json:"recommendation"` // "ami_recommended", "script_recommended", "neutral"
	Reasoning      string `json:"reasoning"`
}

// AMIUsageMetrics represents usage metrics for AMI system
type AMIUsageMetrics struct {
	// Resolution metrics
	ResolutionAttempts  map[AMIResolutionMethod]int64 `json:"resolution_attempts"`
	ResolutionSuccesses map[AMIResolutionMethod]int64 `json:"resolution_successes"`
	AverageResolutionTime map[AMIResolutionMethod]time.Duration `json:"average_resolution_time"`

	// Launch metrics
	AMILaunches     int64         `json:"ami_launches"`
	ScriptLaunches  int64         `json:"script_launches"`
	LaunchSuccessRate float64     `json:"launch_success_rate"`
	AverageLaunchTime time.Duration `json:"average_launch_time"`

	// Cost metrics
	TotalCostSavings    float64 `json:"total_cost_savings"`
	AverageCostSavings  float64 `json:"average_cost_savings"`
	TotalTimeSavings    time.Duration `json:"total_time_savings"`

	// Regional metrics
	RegionalAvailability map[string]float64 `json:"regional_availability"` // region -> availability percentage
	CrossRegionCopies    int64              `json:"cross_region_copies"`

	// Community metrics
	CommunityAMIUsage    int64 `json:"community_ami_usage"`
	CommunityContributions int64 `json:"community_contributions"`

	// Time period
	Period    string    `json:"period"`     // "daily", "weekly", "monthly"
	StartTime time.Time `json:"start_time"`
	EndTime   time.Time `json:"end_time"`
}