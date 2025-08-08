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
	Name        string `yaml:"name" json:"name"`
	Description string `yaml:"description" json:"description"`
	Base        string `yaml:"base" json:"base"` // Base OS (ubuntu-22.04, etc.) or parent template
	
	// Template inheritance
	Inherits []string `yaml:"inherits,omitempty" json:"inherits,omitempty"` // Parent templates to inherit from
	
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
	
	// Instance defaults
	InstanceDefaults InstanceDefaults `yaml:"instance_defaults,omitempty" json:"instance_defaults,omitempty"`
	
	// Template metadata
	Version     string            `yaml:"version,omitempty" json:"version,omitempty"`
	Tags        map[string]string `yaml:"tags,omitempty" json:"tags,omitempty"`
	Maintainer  string            `yaml:"maintainer,omitempty" json:"maintainer,omitempty"`
	LastUpdated time.Time         `yaml:"last_updated,omitempty" json:"last_updated,omitempty"`
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

// UserConfig defines a user to create
type UserConfig struct {
	Name     string   `yaml:"name" json:"name"`
	Groups   []string `yaml:"groups,omitempty" json:"groups,omitempty"`
	Shell    string   `yaml:"shell,omitempty" json:"shell,omitempty"` // Default: /bin/bash
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
	Name         string
	Description  string
	AMI          map[string]map[string]string // region -> arch -> AMI ID  
	InstanceType map[string]string            // arch -> instance type
	UserData     string                       // Generated installation script
	Ports        []int
	EstimatedCostPerHour map[string]float64 // arch -> cost per hour
	
	// Additional metadata from unified template
	Source   *Template `json:"-"` // Reference to source template
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
)

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
	
	// Cached templates
	Templates map[string]*Template
	
	// Last scan time
	LastScan time.Time
}