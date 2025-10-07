// Package templates provides CloudWorkstation's unified template system.
//
// This package implements a simplified, deterministic template architecture that
// leverages existing package managers (apt, conda, spack) instead of custom scripts.
// Templates are declarative YAML definitions that specify packages and services.
package templates

import (
	"fmt"
	"time"

	"github.com/scttfrdmn/cloudworkstation/pkg/research"
)

// Template represents a unified CloudWorkstation template
type Template struct {
	// Basic metadata
	Name            string `yaml:"name" json:"name"`
	Slug            string `yaml:"slug,omitempty" json:"slug,omitempty"` // Short dash-separated name for CLI
	Description     string `yaml:"description" json:"description"`
	LongDescription string `yaml:"long_description,omitempty" json:"long_description,omitempty"` // Detailed description for GUI
	Base            string `yaml:"base" json:"base"`                                             // Base OS (ubuntu-22.04, etc.) or parent template

	// Template inheritance
	Inherits []string `yaml:"inherits,omitempty" json:"inherits,omitempty"` // Parent templates to inherit from

	// Complexity and categorization
	Complexity TemplateComplexity `yaml:"complexity,omitempty" json:"complexity,omitempty"` // simple, moderate, advanced, complex
	Category   string             `yaml:"category,omitempty" json:"category,omitempty"`     // "Machine Learning", "Data Science", etc.
	Domain     string             `yaml:"domain,omitempty" json:"domain,omitempty"`         // "ml", "datascience", "bio", "web", "base"

	// Visual presentation
	Icon     string `yaml:"icon,omitempty" json:"icon,omitempty"`         // Unicode emoji or icon identifier
	Color    string `yaml:"color,omitempty" json:"color,omitempty"`       // Hex color for category theming
	Popular  bool   `yaml:"popular,omitempty" json:"popular,omitempty"`   // Popular badge display
	Featured bool   `yaml:"featured,omitempty" json:"featured,omitempty"` // Featured template (always visible)

	// Connection configuration
	ConnectionType ConnectionType `yaml:"connection_type,omitempty" json:"connection_type,omitempty"` // Explicit connection type (dcv, ssh, auto)

	// Package management strategy
	PackageManager string             `yaml:"package_manager,omitempty" json:"package_manager,omitempty"` // "auto", "apt", "dnf", "conda", "spack", "ami"
	Packages       PackageDefinitions `yaml:"packages,omitempty" json:"packages,omitempty"`

	// AMI configuration (for pre-built images)
	AMIConfig AMIConfig `yaml:"ami_config,omitempty" json:"ami_config,omitempty"`

	// Service configuration
	Services []ServiceConfig `yaml:"services,omitempty" json:"services,omitempty"`

	// User setup
	Users []UserConfig `yaml:"users,omitempty" json:"users,omitempty"`

	// Post-install script
	PostInstall string `yaml:"post_install,omitempty" json:"post_install,omitempty"`

	// User data script for instance initialization
	UserData string `yaml:"user_data,omitempty" json:"user_data,omitempty"`

	// Idle detection configuration
	IdleDetection *IdleDetectionConfig `yaml:"idle_detection,omitempty" json:"idle_detection,omitempty"`

	// Instance defaults
	InstanceDefaults InstanceDefaults `yaml:"instance_defaults,omitempty" json:"instance_defaults,omitempty"`

	// User guidance
	EstimatedLaunchTime int      `yaml:"estimated_launch_time,omitempty" json:"estimated_launch_time,omitempty"` // Launch time in minutes
	Prerequisites       []string `yaml:"prerequisites,omitempty" json:"prerequisites,omitempty"`                 // Required knowledge/skills
	LearningResources   []string `yaml:"learning_resources,omitempty" json:"learning_resources,omitempty"`       // Documentation links

	// Template parameterization
	Parameters map[string]TemplateParameter `yaml:"parameters,omitempty" json:"parameters,omitempty"` // User-configurable parameters
	Variables  map[string]string            `yaml:"variables,omitempty" json:"variables,omitempty"`   // Template-level variables

	// Research user integration (Phase 5A+)
	ResearchUser *research.ResearchUserTemplate `yaml:"research_user,omitempty" json:"research_user,omitempty"`

	// Template metadata
	Version          string            `yaml:"version,omitempty" json:"version,omitempty"`
	ValidationStatus ValidationStatus  `yaml:"validation_status,omitempty" json:"validation_status,omitempty"` // validated, testing, experimental
	Tags             map[string]string `yaml:"tags,omitempty" json:"tags,omitempty"`
	Maintainer       string            `yaml:"maintainer,omitempty" json:"maintainer,omitempty"`
	LastUpdated      time.Time         `yaml:"last_updated,omitempty" json:"last_updated,omitempty"`

	// Marketplace integration (Phase 5B+)
	Marketplace *MarketplaceConfig `yaml:"marketplace,omitempty" json:"marketplace,omitempty"`
}

// PackageDefinitions defines packages for different package managers
type PackageDefinitions struct {
	System []string `yaml:"system,omitempty" json:"system,omitempty"` // apt/dnf packages
	Conda  []string `yaml:"conda,omitempty" json:"conda,omitempty"`   // conda packages
	Spack  []string `yaml:"spack,omitempty" json:"spack,omitempty"`   // spack packages
	Pip    []string `yaml:"pip,omitempty" json:"pip,omitempty"`       // pip packages (when conda used)
}

// AMIConfig defines AMI-based template configuration for Universal AMI System
type AMIConfig struct {
	// AMI deployment strategy
	Strategy AMIStrategy `yaml:"strategy,omitempty" json:"strategy,omitempty"` // ami_preferred, ami_required, ami_fallback

	// Direct AMI mappings (highest priority resolution)
	AMIMappings map[string]string `yaml:"ami_mappings,omitempty" json:"ami_mappings,omitempty"` // region -> AMI ID

	// Dynamic AMI search configuration (second priority)
	AMISearch *AMISearchConfig `yaml:"ami_search,omitempty" json:"ami_search,omitempty"`

	// AWS Marketplace search configuration (third priority)
	MarketplaceSearch *MarketplaceSearchConfig `yaml:"marketplace_search,omitempty" json:"marketplace_search,omitempty"`

	// Fallback behavior when no AMI available
	FallbackStrategy string `yaml:"fallback_strategy,omitempty" json:"fallback_strategy,omitempty"` // script_provisioning, error, cross_region
	FallbackTimeout  string `yaml:"fallback_timeout,omitempty" json:"fallback_timeout,omitempty"`   // Max time for AMI resolution (e.g. "10m")

	// Cost and performance optimization
	PreferredArchitecture    string   `yaml:"preferred_architecture,omitempty" json:"preferred_architecture,omitempty"`         // arm64, x86_64
	InstanceFamilyPreference []string `yaml:"instance_family_preference,omitempty" json:"instance_family_preference,omitempty"` // ["t4g", "m6i", "c6i"]

	// Legacy compatibility (maintains backwards compatibility)
	AMIs           map[string]map[string]string `yaml:"amis,omitempty" json:"amis,omitempty"`                         // region -> arch -> AMI ID (legacy)
	InstanceTypes  map[string]string            `yaml:"instance_types,omitempty" json:"instance_types,omitempty"`     // arch -> instance type (legacy)
	UserDataScript string                       `yaml:"user_data_script,omitempty" json:"user_data_script,omitempty"` // Optional customization script
	SSHUser        string                       `yaml:"ssh_user,omitempty" json:"ssh_user,omitempty"`                 // SSH username for AMI
}

// AMIStrategy defines how templates handle AMI resolution
type AMIStrategy string

const (
	AMIStrategyPreferred AMIStrategy = "ami_preferred" // Try AMI first, fallback to script (recommended)
	AMIStrategyRequired  AMIStrategy = "ami_required"  // AMI only, fail if unavailable
	AMIStrategyFallback  AMIStrategy = "ami_fallback"  // Script first, AMI if script fails
)

// AMISearchConfig defines dynamic AMI discovery parameters
type AMISearchConfig struct {
	Owner           string            `yaml:"owner,omitempty" json:"owner,omitempty"`                         // AWS account ID or alias
	NamePattern     string            `yaml:"name_pattern,omitempty" json:"name_pattern,omitempty"`           // AMI name pattern (e.g. "cws-python-ml-*")
	VersionTag      string            `yaml:"version_tag,omitempty" json:"version_tag,omitempty"`             // Specific version tag
	Architecture    []string          `yaml:"architecture,omitempty" json:"architecture,omitempty"`           // ["x86_64", "arm64"]
	MinCreationDate string            `yaml:"min_creation_date,omitempty" json:"min_creation_date,omitempty"` // ISO date string
	RequiredTags    map[string]string `yaml:"required_tags,omitempty" json:"required_tags,omitempty"`         // Tags that must be present
}

// MarketplaceSearchConfig defines AWS Marketplace AMI discovery
type MarketplaceSearchConfig struct {
	ProductCode       string `yaml:"product_code,omitempty" json:"product_code,omitempty"`             // Marketplace product code
	VersionConstraint string `yaml:"version_constraint,omitempty" json:"version_constraint,omitempty"` // Version constraint (e.g. ">=2.0.0")
	Publisher         string `yaml:"publisher,omitempty" json:"publisher,omitempty"`                   // Publisher filter
}

// ServiceConfig defines a service to configure and enable
type ServiceConfig struct {
	Name   string   `yaml:"name" json:"name"`
	Port   int      `yaml:"port,omitempty" json:"port,omitempty"`
	Config []string `yaml:"config,omitempty" json:"config,omitempty"` // Config file lines
	Enable bool     `yaml:"enable,omitempty" json:"enable,omitempty"` // Default: true
}

// IdleDetectionConfig represents idle detection configuration in templates
type IdleDetectionConfig struct {
	Enabled                   bool `yaml:"enabled" json:"enabled"`
	IdleThresholdMinutes      int  `yaml:"idle_threshold_minutes" json:"idle_threshold_minutes"`
	HibernateThresholdMinutes int  `yaml:"hibernate_threshold_minutes" json:"hibernate_threshold_minutes"`
	CheckIntervalMinutes      int  `yaml:"check_interval_minutes" json:"check_interval_minutes"`
}

// UserConfig defines a user to create
type UserConfig struct {
	Name   string   `yaml:"name" json:"name"`
	Groups []string `yaml:"groups,omitempty" json:"groups,omitempty"`
	Shell  string   `yaml:"shell,omitempty" json:"shell,omitempty"` // Default: /bin/bash
}

// InstanceDefaults defines default instance configuration
type InstanceDefaults struct {
	Type                 string             `yaml:"type,omitempty" json:"type,omitempty"` // Default instance type
	Ports                []int              `yaml:"ports,omitempty" json:"ports,omitempty"`
	EstimatedCostPerHour map[string]float64 `yaml:"estimated_cost_per_hour,omitempty" json:"estimated_cost_per_hour,omitempty"` // arch -> cost
}

// RuntimeTemplate represents a resolved template ready for instance launch
// This maintains compatibility with existing types.RuntimeTemplate
type RuntimeTemplate struct {
	Name                 string
	Slug                 string // CLI identifier for template (e.g., "python-ml")
	Description          string
	LongDescription      string                       // Detailed description for GUI
	AMI                  map[string]map[string]string // region -> arch -> AMI ID
	InstanceType         map[string]string            // arch -> instance type
	UserData             string                       // Generated installation script
	Ports                []int
	EstimatedCostPerHour map[string]float64   // arch -> cost per hour
	IdleDetection        *IdleDetectionConfig // Idle detection configuration

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
	Source    *Template `json:"-"` // Reference to source template
	Generated time.Time // When this runtime template was generated
}

// PackageManagerType represents supported package managers
type PackageManagerType string

const (
	PackageManagerApt   PackageManagerType = "apt"
	PackageManagerDnf   PackageManagerType = "dnf"
	PackageManagerConda PackageManagerType = "conda"
	PackageManagerSpack PackageManagerType = "spack"
	PackageManagerAMI   PackageManagerType = "ami"
	PackageManagerPip   PackageManagerType = "pip"
)

// TemplateComplexity represents the complexity level of a template
type TemplateComplexity string

const (
	ComplexitySimple   TemplateComplexity = "simple"   // Ready to use, perfect for getting started
	ComplexityModerate TemplateComplexity = "moderate" // Some customization available, good for regular users
	ComplexityAdvanced TemplateComplexity = "advanced" // Highly configurable, for experienced users
	ComplexityComplex  TemplateComplexity = "complex"  // Maximum flexibility, requires technical knowledge
)

// ValidationStatus represents the validation state of a template
type ValidationStatus string

const (
	ValidationValidated    ValidationStatus = "validated"    // Fully tested and verified
	ValidationTesting      ValidationStatus = "testing"      // Currently under testing
	ValidationExperimental ValidationStatus = "experimental" // Experimental, use with caution
	ValidationFailed       ValidationStatus = "failed"       // Validation failed
)

// ConnectionType represents the connection interface type for instances
type ConnectionType string

const (
	ConnectionTypeAuto ConnectionType = "auto" // Automatic detection based on template analysis (default)
	ConnectionTypeDCV  ConnectionType = "dcv"  // NICE DCV remote desktop for GUI instances
	ConnectionTypeSSH  ConnectionType = "ssh"  // SSH terminal for headless instances
	ConnectionTypeWeb  ConnectionType = "web"  // Web interface (Jupyter, RStudio, Streamlit, etc.)
	ConnectionTypeAll  ConnectionType = "all"  // Supports DCV + SSH + Web - user can choose
)

// ComplexityLevel returns the numeric level for sorting (1=simple, 4=complex)
func (c TemplateComplexity) Level() int {
	switch c {
	case ComplexitySimple:
		return 1
	case ComplexityModerate:
		return 2
	case ComplexityAdvanced:
		return 3
	case ComplexityComplex:
		return 4
	default:
		return 1 // Default to simple
	}
}

// Label returns the human-readable label for the complexity level
func (c TemplateComplexity) Label() string {
	switch c {
	case ComplexitySimple:
		return "Simple"
	case ComplexityModerate:
		return "Moderate"
	case ComplexityAdvanced:
		return "Advanced"
	case ComplexityComplex:
		return "Complex"
	default:
		return "Simple"
	}
}

// Badge returns the badge text for GUI display
func (c TemplateComplexity) Badge() string {
	switch c {
	case ComplexitySimple:
		return "Ready to Use"
	case ComplexityModerate:
		return "Some Options"
	case ComplexityAdvanced:
		return "Many Options"
	case ComplexityComplex:
		return "Full Control"
	default:
		return "Ready to Use"
	}
}

// Icon returns the emoji icon for the complexity level
func (c TemplateComplexity) Icon() string {
	switch c {
	case ComplexitySimple:
		return "🟢"
	case ComplexityModerate:
		return "🟡"
	case ComplexityAdvanced:
		return "🟠"
	case ComplexityComplex:
		return "🔴"
	default:
		return "🟢"
	}
}

// Color returns the hex color for the complexity level
func (c TemplateComplexity) Color() string {
	switch c {
	case ComplexitySimple:
		return "#059669"
	case ComplexityModerate:
		return "#d97706"
	case ComplexityAdvanced:
		return "#ea580c"
	case ComplexityComplex:
		return "#dc2626"
	default:
		return "#059669"
	}
}

// PackageManagerStrategy handles package manager selection logic
type PackageManagerStrategy struct {
	Template *Template

	// Selection rules for "auto" mode
	Rules PackageManagerRules
}

// PackageManagerRules defines rules for automatic package manager selection
type PackageManagerRules struct {
	// If template has HPC/scientific computing packages -> spack
	HPCIndicators []string

	// If template has Python data science packages -> conda
	PythonDataScienceIndicators []string

	// If template has R packages -> conda (better R ecosystem)
	RIndicators []string

	// Default fallback -> apt (system packages)
	DefaultManager PackageManagerType
}

// TemplateParser handles parsing and validation of template YAML files
type TemplateParser struct {
	// Base AMI mappings for validation
	BaseAMIs map[string]map[string]map[string]string // base -> region -> arch -> AMI

	// Package manager strategy
	Strategy *PackageManagerStrategy
}

// TemplateResolver converts unified templates to runtime templates
type TemplateResolver struct {
	Parser      *TemplateParser
	ScriptGen   *ScriptGenerator
	AMIRegistry map[string]map[string]map[string]string // template -> region -> arch -> AMI
}

// ScriptGenerator generates installation scripts for different package managers
type ScriptGenerator struct {
	// Script templates for each package manager
	AptTemplate   string
	DnfTemplate   string
	CondaTemplate string
	SpackTemplate string
	AMITemplate   string
	PipTemplate   string
}

// TemplateValidationError represents template validation errors
type TemplateValidationError struct {
	Field   string
	Message string
}

func (e *TemplateValidationError) Error() string {
	return "template validation error in " + e.Field + ": " + e.Message
}

// TemplateRegistry manages template discovery and caching
type TemplateRegistry struct {
	// Template directories to scan
	TemplateDirs []string

	// Cached templates (indexed by name)
	Templates map[string]*Template

	// Slug index for fast lookup (slug -> template name)
	SlugIndex map[string]string

	// Last scan time
	LastScan time.Time
}

// TemplateParameter defines a configurable parameter for templates
type TemplateParameter struct {
	// Basic parameter information
	Name        string `yaml:"name" json:"name"`
	Description string `yaml:"description" json:"description"`
	Type        string `yaml:"type" json:"type"` // string, int, bool, choice

	// Value constraints
	Default  interface{}   `yaml:"default,omitempty" json:"default,omitempty"`
	Choices  []interface{} `yaml:"choices,omitempty" json:"choices,omitempty"`   // For choice type
	Min      interface{}   `yaml:"min,omitempty" json:"min,omitempty"`           // For int type
	Max      interface{}   `yaml:"max,omitempty" json:"max,omitempty"`           // For int type
	Pattern  string        `yaml:"pattern,omitempty" json:"pattern,omitempty"`   // For string type (regex)
	Required bool          `yaml:"required,omitempty" json:"required,omitempty"` // Parameter is required

	// UI presentation
	DisplayName string `yaml:"display_name,omitempty" json:"display_name,omitempty"` // Human-readable name
	Group       string `yaml:"group,omitempty" json:"group,omitempty"`               // Parameter group for UI organization
	Order       int    `yaml:"order,omitempty" json:"order,omitempty"`               // Display order within group
	Hidden      bool   `yaml:"hidden,omitempty" json:"hidden,omitempty"`             // Hide from UI (internal use)

	// Advanced features
	Conditional string `yaml:"conditional,omitempty" json:"conditional,omitempty"` // Show only if condition is met
	Impact      string `yaml:"impact,omitempty" json:"impact,omitempty"`           // cost, performance, security
}

// TemplateParameterType defines the supported parameter types
type TemplateParameterType string

const (
	ParameterTypeString TemplateParameterType = "string"
	ParameterTypeInt    TemplateParameterType = "int"
	ParameterTypeBool   TemplateParameterType = "bool"
	ParameterTypeChoice TemplateParameterType = "choice"
	ParameterTypeList   TemplateParameterType = "list"
)

// TemplateParameterValues holds user-provided parameter values
type TemplateParameterValues map[string]interface{}

// Validate checks if parameter values meet the template parameter constraints
func (values TemplateParameterValues) Validate(parameters map[string]TemplateParameter) []TemplateValidationError {
	var errors []TemplateValidationError

	// Check required parameters
	for name, param := range parameters {
		if param.Required {
			if _, exists := values[name]; !exists {
				errors = append(errors, TemplateValidationError{
					Field:   "parameters." + name,
					Message: "required parameter missing",
				})
			}
		}
	}

	// Validate provided values
	for name, value := range values {
		param, exists := parameters[name]
		if !exists {
			errors = append(errors, TemplateValidationError{
				Field:   "parameters." + name,
				Message: "unknown parameter",
			})
			continue
		}

		// Type-specific validation
		if err := validateParameterValue(name, value, param); err != nil {
			errors = append(errors, *err)
		}
	}

	return errors
}

// validateParameterValue validates a single parameter value
func validateParameterValue(name string, value interface{}, param TemplateParameter) *TemplateValidationError {
	switch param.Type {
	case "string":
		str, ok := value.(string)
		if !ok {
			return &TemplateValidationError{
				Field:   "parameters." + name,
				Message: "must be a string",
			}
		}
		if param.Pattern != "" {
			// Pattern validation would go here (regex)
			_ = str // Use the validated string
		}

	case "int":
		// JSON unmarshaling converts numbers to float64 by default
		// Accept both int and float64, validate they're whole numbers
		switch v := value.(type) {
		case int:
			// Already an integer, valid
		case float64:
			// Check if it's a whole number
			if v != float64(int64(v)) {
				return &TemplateValidationError{
					Field:   "parameters." + name,
					Message: "must be an integer (whole number)",
				}
			}
		default:
			return &TemplateValidationError{
				Field:   "parameters." + name,
				Message: "must be an integer",
			}
		}
		// Min/max validation would go here

	case "bool":
		_, ok := value.(bool)
		if !ok {
			return &TemplateValidationError{
				Field:   "parameters." + name,
				Message: "must be a boolean",
			}
		}

	case "choice":
		if len(param.Choices) > 0 {
			found := false
			// Convert value to string for comparison since choices are strings
			valueStr := fmt.Sprintf("%v", value)
			for _, choice := range param.Choices {
				choiceStr := fmt.Sprintf("%v", choice)
				if valueStr == choiceStr {
					found = true
					break
				}
			}
			if !found {
				return &TemplateValidationError{
					Field:   "parameters." + name,
					Message: fmt.Sprintf("must be one of the allowed choices: %v", param.Choices),
				}
			}
		}
	}

	return nil
}

// MarketplaceConfig defines template marketplace integration settings (Phase 5B+)
type MarketplaceConfig struct {
	// Registry information
	Registry     string `yaml:"registry,omitempty" json:"registry,omitempty"`           // Registry URL or identifier (community, institutional, private)
	RegistryType string `yaml:"registry_type,omitempty" json:"registry_type,omitempty"` // community, institutional, private, official

	// Publication metadata
	PublishedAt      *time.Time `yaml:"published_at,omitempty" json:"published_at,omitempty"`
	Publisher        string     `yaml:"publisher,omitempty" json:"publisher,omitempty"`   // Organization or user who published
	License          string     `yaml:"license,omitempty" json:"license,omitempty"`       // Template license (MIT, Apache-2.0, etc.)
	SourceURL        string     `yaml:"source_url,omitempty" json:"source_url,omitempty"` // Git repository or source location
	DocumentationURL string     `yaml:"documentation_url,omitempty" json:"documentation_url,omitempty"`

	// Community metrics
	Downloads   int64   `yaml:"downloads,omitempty" json:"downloads,omitempty"`       // Download/usage count
	Rating      float64 `yaml:"rating,omitempty" json:"rating,omitempty"`             // Average user rating (0-5)
	RatingCount int     `yaml:"rating_count,omitempty" json:"rating_count,omitempty"` // Number of ratings

	// Security and validation
	SecurityScan    *SecurityScanResult `yaml:"security_scan,omitempty" json:"security_scan,omitempty"`
	ValidationTests []ValidationTest    `yaml:"validation_tests,omitempty" json:"validation_tests,omitempty"`

	// Marketplace categories and discoverability
	Keywords   []string           `yaml:"keywords,omitempty" json:"keywords,omitempty"`     // Search keywords
	Categories []string           `yaml:"categories,omitempty" json:"categories,omitempty"` // Marketplace categories
	Badges     []MarketplaceBadge `yaml:"badges,omitempty" json:"badges,omitempty"`         // Quality badges
	Verified   bool               `yaml:"verified,omitempty" json:"verified,omitempty"`     // Official verification status

	// Dependency information
	Dependencies  []TemplateDependency `yaml:"dependencies,omitempty" json:"dependencies,omitempty"`   // Other templates this depends on
	Dependents    []string             `yaml:"dependents,omitempty" json:"dependents,omitempty"`       // Templates that depend on this
	Compatibility []string             `yaml:"compatibility,omitempty" json:"compatibility,omitempty"` // Compatible CloudWorkstation versions
}

// SecurityScanResult contains security scanning information
type SecurityScanResult struct {
	Status    string            `yaml:"status" json:"status"` // passed, failed, warning, pending
	ScanDate  time.Time         `yaml:"scan_date" json:"scan_date"`
	Scanner   string            `yaml:"scanner" json:"scanner"` // Tool used for scanning
	Findings  []SecurityFinding `yaml:"findings,omitempty" json:"findings,omitempty"`
	Score     float64           `yaml:"score,omitempty" json:"score,omitempty"`           // Security score (0-100)
	ReportURL string            `yaml:"report_url,omitempty" json:"report_url,omitempty"` // Detailed report link
}

// SecurityFinding represents a security issue found during scanning
type SecurityFinding struct {
	Severity    string `yaml:"severity" json:"severity"` // critical, high, medium, low, info
	Category    string `yaml:"category" json:"category"` // vulnerability, misconfiguration, secret, etc.
	Description string `yaml:"description" json:"description"`
	Remediation string `yaml:"remediation,omitempty" json:"remediation,omitempty"`
	CVEID       string `yaml:"cve_id,omitempty" json:"cve_id,omitempty"`
}

// ValidationTest represents automated template validation results
type ValidationTest struct {
	Name       string    `yaml:"name" json:"name"`
	Status     string    `yaml:"status" json:"status"` // passed, failed, skipped
	ExecutedAt time.Time `yaml:"executed_at" json:"executed_at"`
	Duration   string    `yaml:"duration,omitempty" json:"duration,omitempty"`
	Message    string    `yaml:"message,omitempty" json:"message,omitempty"`
	Details    string    `yaml:"details,omitempty" json:"details,omitempty"`
}

// MarketplaceBadge represents quality or feature badges
type MarketplaceBadge struct {
	Type        string     `yaml:"type" json:"type"`   // verified, trending, editor_choice, community_favorite
	Label       string     `yaml:"label" json:"label"` // Display text
	Description string     `yaml:"description,omitempty" json:"description,omitempty"`
	EarnedAt    *time.Time `yaml:"earned_at,omitempty" json:"earned_at,omitempty"`
}

// TemplateDependency represents a dependency on another template
type TemplateDependency struct {
	Name    string `yaml:"name" json:"name"`                           // Template name or slug
	Version string `yaml:"version,omitempty" json:"version,omitempty"` // Version requirement (semver)
	Source  string `yaml:"source,omitempty" json:"source,omitempty"`   // Registry or source location
	Type    string `yaml:"type,omitempty" json:"type,omitempty"`       // inherits, runtime, build
}
