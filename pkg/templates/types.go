// Package templates provides CloudWorkstation's unified template system.
//
// This package implements a simplified, deterministic template architecture that
// leverages existing package managers (apt, conda, spack) instead of custom scripts.
// Templates are declarative YAML definitions that specify packages and services.
package templates

import (
	"time"
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

	// Template metadata
	Version          string            `yaml:"version,omitempty" json:"version,omitempty"`
	ValidationStatus ValidationStatus  `yaml:"validation_status,omitempty" json:"validation_status,omitempty"` // validated, testing, experimental
	Tags             map[string]string `yaml:"tags,omitempty" json:"tags,omitempty"`
	Maintainer       string            `yaml:"maintainer,omitempty" json:"maintainer,omitempty"`
	LastUpdated      time.Time         `yaml:"last_updated,omitempty" json:"last_updated,omitempty"`
}

// PackageDefinitions defines packages for different package managers
type PackageDefinitions struct {
	System []string `yaml:"system,omitempty" json:"system,omitempty"` // apt/dnf packages
	Conda  []string `yaml:"conda,omitempty" json:"conda,omitempty"`   // conda packages
	Spack  []string `yaml:"spack,omitempty" json:"spack,omitempty"`   // spack packages
	Pip    []string `yaml:"pip,omitempty" json:"pip,omitempty"`       // pip packages (when conda used)
}

// AMIConfig defines AMI-based template configuration
type AMIConfig struct {
	// AMI IDs for different regions and architectures
	AMIs map[string]map[string]string `yaml:"amis" json:"amis"` // region -> arch -> AMI ID

	// Instance type overrides for different architectures
	InstanceTypes map[string]string `yaml:"instance_types,omitempty" json:"instance_types,omitempty"` // arch -> instance type

	// Optional user data script for AMI customization
	UserDataScript string `yaml:"user_data_script,omitempty" json:"user_data_script,omitempty"`

	// SSH username for the AMI (varies by image)
	SSHUser string `yaml:"ssh_user,omitempty" json:"ssh_user,omitempty"`
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
