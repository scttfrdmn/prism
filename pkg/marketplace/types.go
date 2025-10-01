// Package marketplace provides community template discovery and publishing capabilities
package marketplace

import (
	"time"

	"github.com/scttfrdmn/cloudworkstation/pkg/templates"
)

// MarketplaceRegistry defines the interface for community template operations
type MarketplaceRegistry interface {
	// Discovery operations
	SearchTemplates(query SearchQuery) ([]*CommunityTemplate, error)
	GetTemplate(templateID string) (*CommunityTemplate, error)
	ListCategories() ([]TemplateCategory, error)
	GetFeatured() ([]*CommunityTemplate, error)
	GetTrending(timeframe string) ([]*CommunityTemplate, error)

	// Publishing operations
	PublishTemplate(template *TemplatePublication) (*PublicationResult, error)
	UpdateTemplate(templateID string, update *TemplateUpdate) error
	UnpublishTemplate(templateID string) error
	GetUserPublications(userID string) ([]*CommunityTemplate, error)

	// Community operations
	AddReview(templateID string, review *TemplateReview) error
	GetReviews(templateID string, pagination *ReviewPagination) (*ReviewResponse, error)
	TrackUsage(templateID string, event *UsageEvent) error
	ForkTemplate(templateID string, fork *TemplateFork) (*CommunityTemplate, error)

	// Analytics operations
	GetTemplateAnalytics(templateID string) (*TemplateAnalytics, error)
	GetUsageStats(templateID string, timeframe string) (*UsageStats, error)
}

// CommunityTemplate represents a template published to the marketplace
type CommunityTemplate struct {
	// Core identification
	TemplateID  string    `json:"template_id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Author      string    `json:"author"`
	AuthorName  string    `json:"author_name,omitempty"`
	Version     string    `json:"version"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`

	// Categorization and discovery
	Category       string   `json:"category"`
	Tags           []string `json:"tags"`
	Keywords       []string `json:"keywords,omitempty"`
	ResearchDomain string   `json:"research_domain,omitempty"`

	// Technical specifications
	Architecture      []string              `json:"architecture"` // ["x86_64", "arm64"]
	SupportedRegions  []string              `json:"supported_regions"`
	RequiredResources *ResourceRequirements `json:"required_resources,omitempty"`
	EstimatedCost     *CostEstimate         `json:"estimated_cost,omitempty"`

	// Community metrics
	Rating        float64 `json:"rating"`         // Average rating (0-5)
	ReviewCount   int     `json:"review_count"`   // Total number of reviews
	DownloadCount int     `json:"download_count"` // Total downloads
	LaunchCount   int     `json:"launch_count"`   // Total launches
	ForkCount     int     `json:"fork_count"`     // Number of forks

	// Quality indicators
	Verified          bool      `json:"verified"`           // Verified by CloudWorkstation team
	Featured          bool      `json:"featured"`           // Featured template
	LastTested        time.Time `json:"last_tested"`        // Last automated testing
	TestStatus        string    `json:"test_status"`        // "passed", "failed", "unknown"
	SecurityScore     int       `json:"security_score"`     // 0-100 security score
	MaintenanceStatus string    `json:"maintenance_status"` // "active", "deprecated", "archived"

	// Content and documentation
	Documentation string          `json:"documentation"`        // Markdown documentation
	Screenshots   []string        `json:"screenshots"`          // Screenshot URLs
	VideoDemo     string          `json:"video_demo,omitempty"` // Demo video URL
	ExampleUsage  []UsageExample  `json:"example_usage"`        // Usage examples
	RelatedPapers []ResearchPaper `json:"related_papers"`       // Associated publications

	// Publication metadata
	Publication *PublicationMetadata `json:"publication"`
	AMIInfo     *AMIAvailability     `json:"ami_info,omitempty"`

	// Underlying template definition
	Template *templates.Template `json:"template"` // Full template specification
}

// TemplatePublication represents a template being published to the marketplace
type TemplatePublication struct {
	// Source information
	SourceInstanceID string `json:"source_instance_id,omitempty"` // Create from running instance
	SourceTemplateID string `json:"source_template_id,omitempty"` // Publish existing template

	// Publication details
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Category    string   `json:"category"`
	Tags        []string `json:"tags"`
	Keywords    []string `json:"keywords,omitempty"`

	// Documentation and media
	Documentation string   `json:"documentation"`
	Screenshots   []string `json:"screenshots,omitempty"`
	VideoDemo     string   `json:"video_demo,omitempty"`

	// Publication settings
	Visibility    string            `json:"visibility"`     // "public", "private", "organization"
	License       string            `json:"license"`        // "MIT", "Apache-2.0", etc.
	GenerateAMI   bool              `json:"generate_ami"`   // Whether to create AMIs
	TargetRegions []string          `json:"target_regions"` // Regions for AMI generation
	Metadata      map[string]string `json:"metadata"`       // Additional metadata

	// Research context
	ResearchDomain string          `json:"research_domain,omitempty"`
	PaperDOI       string          `json:"paper_doi,omitempty"`
	FundingSource  string          `json:"funding_source,omitempty"`
	RelatedPapers  []ResearchPaper `json:"related_papers,omitempty"`
}

// PublicationResult contains the result of a template publication
type PublicationResult struct {
	TemplateID     string    `json:"template_id"`
	PublicationURL string    `json:"publication_url"`
	Status         string    `json:"status"` // "published", "pending_review", "rejected"
	Message        string    `json:"message"`
	CreatedAt      time.Time `json:"created_at"`
	AMICreationIDs []string  `json:"ami_creation_ids,omitempty"` // AMI creation tracking
}

// TemplateUpdate represents updates to a published template
type TemplateUpdate struct {
	Name          string            `json:"name,omitempty"`
	Description   string            `json:"description,omitempty"`
	Documentation string            `json:"documentation,omitempty"`
	Tags          []string          `json:"tags,omitempty"`
	Keywords      []string          `json:"keywords,omitempty"`
	Screenshots   []string          `json:"screenshots,omitempty"`
	VideoDemo     string            `json:"video_demo,omitempty"`
	Metadata      map[string]string `json:"metadata,omitempty"`
	Version       string            `json:"version,omitempty"` // New version number
}

// SearchQuery defines parameters for template search operations
type SearchQuery struct {
	// Text search
	Query    string   `json:"query,omitempty"`    // Free text search
	Keywords []string `json:"keywords,omitempty"` // Specific keywords
	Category string   `json:"category,omitempty"` // Filter by category
	Tags     []string `json:"tags,omitempty"`     // Filter by tags
	Author   string   `json:"author,omitempty"`   // Filter by author

	// Technical filters
	Architecture string `json:"architecture,omitempty"`  // "x86_64", "arm64"
	Region       string `json:"region,omitempty"`        // Must be available in region
	AMIAvailable bool   `json:"ami_available,omitempty"` // Filter for AMI-enabled templates

	// Quality filters
	MinRating    float64 `json:"min_rating,omitempty"`    // Minimum rating (0-5)
	VerifiedOnly bool    `json:"verified_only,omitempty"` // Only verified templates
	FeaturedOnly bool    `json:"featured_only,omitempty"` // Only featured templates
	MinDownloads int     `json:"min_downloads,omitempty"` // Minimum download count

	// Sorting and pagination
	SortBy    string `json:"sort_by,omitempty"`    // "rating", "downloads", "updated", "created"
	SortOrder string `json:"sort_order,omitempty"` // "asc", "desc"
	Limit     int    `json:"limit,omitempty"`      // Results per page
	Offset    int    `json:"offset,omitempty"`     // Pagination offset
}

// TemplateCategory represents a template category
type TemplateCategory struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	Description   string `json:"description"`
	Icon          string `json:"icon,omitempty"`
	Color         string `json:"color,omitempty"`
	TemplateCount int    `json:"template_count"`
	Featured      bool   `json:"featured"`
}

// TemplateReview represents a user review of a template
type TemplateReview struct {
	ReviewID     string `json:"review_id"`
	TemplateID   string `json:"template_id"`
	Reviewer     string `json:"reviewer"`      // User ID
	ReviewerName string `json:"reviewer_name"` // Display name
	Rating       int    `json:"rating"`        // 1-5 stars
	Title        string `json:"title"`
	Content      string `json:"content"`
	UseCase      string `json:"use_case,omitempty"` // How they used the template

	// Verification and validation
	VerifiedUsage bool   `json:"verified_usage"`        // Confirmed actual usage
	VerifiedBy    string `json:"verified_by,omitempty"` // Who verified the review

	// Community interaction
	HelpfulVotes   int     `json:"helpful_votes"`     // "Helpful" votes
	UnhelpfulVotes int     `json:"unhelpful_votes"`   // "Not helpful" votes
	Replies        []Reply `json:"replies,omitempty"` // Replies to review

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Reply represents a reply to a review
type Reply struct {
	ReplyID     string    `json:"reply_id"`
	Replier     string    `json:"replier"`      // User ID
	ReplierName string    `json:"replier_name"` // Display name
	Content     string    `json:"content"`
	CreatedAt   time.Time `json:"created_at"`
}

// ReviewPagination defines pagination parameters for reviews
type ReviewPagination struct {
	Limit  int    `json:"limit,omitempty"`   // Results per page
	Offset int    `json:"offset,omitempty"`  // Pagination offset
	SortBy string `json:"sort_by,omitempty"` // "rating", "helpful", "recent"
}

// ReviewResponse contains paginated review results
type ReviewResponse struct {
	Reviews    []*TemplateReview `json:"reviews"`
	TotalCount int               `json:"total_count"`
	Page       int               `json:"page"`
	TotalPages int               `json:"total_pages"`
	HasMore    bool              `json:"has_more"`
}

// UsageEvent represents a template usage event for analytics
type UsageEvent struct {
	EventType    string            `json:"event_type"` // "download", "launch", "success", "failure"
	TemplateID   string            `json:"template_id"`
	UserID       string            `json:"user_id"`
	InstanceID   string            `json:"instance_id,omitempty"`
	Region       string            `json:"region"`
	Architecture string            `json:"architecture"`
	LaunchTime   time.Duration     `json:"launch_time,omitempty"`   // Time to successful launch
	ErrorDetails string            `json:"error_details,omitempty"` // Error information
	Metadata     map[string]string `json:"metadata,omitempty"`
	Timestamp    time.Time         `json:"timestamp"`
}

// TemplateFork represents a template fork operation
type TemplateFork struct {
	NewName        string         `json:"new_name"`
	NewDescription string         `json:"new_description"`
	Modifications  []Modification `json:"modifications,omitempty"` // Changes made to original
	Private        bool           `json:"private"`                 // Whether fork is private
}

// Modification represents a change made when forking
type Modification struct {
	Type        string `json:"type"`        // "package_add", "package_remove", "config_change", etc.
	Description string `json:"description"` // Human-readable description
	Details     string `json:"details"`     // Technical details
}

// Supporting structures

// ResourceRequirements defines minimum resource requirements
type ResourceRequirements struct {
	MinCPU       int    `json:"min_cpu"`              // Minimum CPU cores
	MinMemoryGB  int    `json:"min_memory_gb"`        // Minimum memory in GB
	MinStorageGB int    `json:"min_storage_gb"`       // Minimum storage in GB
	RequiresGPU  bool   `json:"requires_gpu"`         // Whether GPU is required
	GPUType      string `json:"gpu_type,omitempty"`   // Specific GPU type if needed
	NetworkBW    string `json:"network_bw,omitempty"` // Network bandwidth requirements
}

// CostEstimate provides cost estimation information
type CostEstimate struct {
	HourlyCost   float64   `json:"hourly_cost"`   // Estimated cost per hour
	DailyCost    float64   `json:"daily_cost"`    // Estimated cost per day
	MonthlyCost  float64   `json:"monthly_cost"`  // Estimated cost per month
	Region       string    `json:"region"`        // Region for cost estimate
	InstanceType string    `json:"instance_type"` // Recommended instance type
	Currency     string    `json:"currency"`      // Cost currency (USD)
	LastUpdated  time.Time `json:"last_updated"`  // When estimate was calculated
}

// PublicationMetadata contains publication-specific metadata
type PublicationMetadata struct {
	License          string `json:"license"`
	Visibility       string `json:"visibility"`
	PaperDOI         string `json:"paper_doi,omitempty"`
	FundingSource    string `json:"funding_source,omitempty"`
	DocumentationURL string `json:"documentation_url,omitempty"`
	RepositoryURL    string `json:"repository_url,omitempty"`
	ContactEmail     string `json:"contact_email,omitempty"`
}

// AMIAvailability tracks AMI availability across regions
type AMIAvailability struct {
	Available      bool              `json:"available"`
	Regions        map[string]string `json:"regions"` // region -> AMI ID mapping
	LastUpdated    time.Time         `json:"last_updated"`
	CreationStatus string            `json:"creation_status"` // "creating", "available", "failed"
}

// UsageExample provides template usage examples
type UsageExample struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	Command     string `json:"command"`
	UseCase     string `json:"use_case"`
	Difficulty  string `json:"difficulty"` // "beginner", "intermediate", "advanced"
}

// ResearchPaper represents associated research publications
type ResearchPaper struct {
	DOI         string    `json:"doi"`
	Title       string    `json:"title"`
	Authors     []string  `json:"authors"`
	Journal     string    `json:"journal,omitempty"`
	Year        int       `json:"year"`
	URL         string    `json:"url,omitempty"`
	Abstract    string    `json:"abstract,omitempty"`
	PublishedAt time.Time `json:"published_at,omitempty"`
}

// TemplateAnalytics provides comprehensive analytics for a template
type TemplateAnalytics struct {
	TemplateID string `json:"template_id"`

	// Usage metrics
	TotalDownloads int     `json:"total_downloads"`
	TotalLaunches  int     `json:"total_launches"`
	SuccessRate    float64 `json:"success_rate"` // Successful launches / total launches

	// Community metrics
	AverageRating float64 `json:"average_rating"`
	TotalReviews  int     `json:"total_reviews"`
	TotalForks    int     `json:"total_forks"`

	// Performance metrics
	AverageLaunchTime time.Duration   `json:"average_launch_time"`
	FailureReasons    []FailureReason `json:"failure_reasons"`

	// Geographic distribution
	RegionUsage       map[string]int `json:"region_usage"`       // region -> usage count
	ArchitectureUsage map[string]int `json:"architecture_usage"` // arch -> usage count

	// Temporal patterns
	DailyUsage   []DailyUsagePoint   `json:"daily_usage"`
	MonthlyTrend []MonthlyTrendPoint `json:"monthly_trend"`

	LastUpdated time.Time `json:"last_updated"`
}

// FailureReason tracks common failure patterns
type FailureReason struct {
	Reason         string    `json:"reason"`
	Count          int       `json:"count"`
	Percentage     float64   `json:"percentage"`
	LastOccurrence time.Time `json:"last_occurrence"`
}

// DailyUsagePoint represents daily usage statistics
type DailyUsagePoint struct {
	Date      time.Time `json:"date"`
	Downloads int       `json:"downloads"`
	Launches  int       `json:"launches"`
	Successes int       `json:"successes"`
}

// MonthlyTrendPoint represents monthly trend data
type MonthlyTrendPoint struct {
	Month     time.Time `json:"month"`
	Downloads int       `json:"downloads"`
	Launches  int       `json:"launches"`
	Rating    float64   `json:"rating"`
}

// UsageStats provides usage statistics for a specific timeframe
type UsageStats struct {
	TemplateID string    `json:"template_id"`
	Timeframe  string    `json:"timeframe"` // "day", "week", "month", "year"
	StartDate  time.Time `json:"start_date"`
	EndDate    time.Time `json:"end_date"`

	// Core metrics
	Downloads int `json:"downloads"`
	Launches  int `json:"launches"`
	Successes int `json:"successes"`
	Failures  int `json:"failures"`

	// Derived metrics
	SuccessRate       float64       `json:"success_rate"`
	AverageLaunchTime time.Duration `json:"average_launch_time"`

	// Comparisons
	PeriodComparison *PeriodComparison `json:"period_comparison,omitempty"`
}

// PeriodComparison compares current period to previous period
type PeriodComparison struct {
	DownloadChange    float64 `json:"download_change"`     // Percentage change
	LaunchChange      float64 `json:"launch_change"`       // Percentage change
	SuccessRateChange float64 `json:"success_rate_change"` // Percentage point change
}

// MarketplaceConfig defines configuration for the marketplace system
type MarketplaceConfig struct {
	// Registry configuration
	RegistryEndpoint string `json:"registry_endpoint"`
	S3Bucket         string `json:"s3_bucket"`
	DynamoDBTable    string `json:"dynamodb_table"`
	CDNEndpoint      string `json:"cdn_endpoint"`

	// Publication settings
	AutoAMIGeneration bool     `json:"auto_ami_generation"`
	DefaultRegions    []string `json:"default_regions"`
	RequireModeration bool     `json:"require_moderation"`

	// Quality controls
	MinRatingForFeatured  float64 `json:"min_rating_for_featured"`
	MinReviewsForFeatured int     `json:"min_reviews_for_featured"`

	// Rate limiting
	PublishRateLimit int `json:"publish_rate_limit"` // Publications per user per day
	ReviewRateLimit  int `json:"review_rate_limit"`  // Reviews per user per day
	SearchRateLimit  int `json:"search_rate_limit"`  // Searches per user per minute
}
