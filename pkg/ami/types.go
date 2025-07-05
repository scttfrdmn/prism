// Package ami provides CloudWorkstation's AMI creation system.
//
// This package implements automated building of Amazon Machine Images (AMIs) for
// CloudWorkstation templates. It handles YAML template parsing, EC2 instance
// management for building, AMI creation, and validation.
//
// Key Components:
//   - Builder: Core AMI creation service with EC2 orchestration
//   - Parser: YAML template parser and validator
//   - Registry: AMI version management and lookup service
//   - Validator: AMI build validation framework
//
// The AMI builder implements CloudWorkstation's core principle of "Default to Success"
// by ensuring every template has reliable, pre-built AMIs available across all
// supported regions and architectures.
package ami

import (
	"time"

	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
)

// Template defines the YAML structure for an AMI template
type Template struct {
	Name        string       `yaml:"name"`
	Base        string       `yaml:"base"`
	Description string       `yaml:"description"`
	BuildSteps  []BuildStep  `yaml:"build_steps"`
	Validation  []Validation `yaml:"validation"`
	// Optional fields
	Tags         map[string]string `yaml:"tags,omitempty"`
	MinDiskSize  int               `yaml:"min_disk_size,omitempty"` // GB
	Architecture string            `yaml:"architecture,omitempty"`  // Default is both
	// Dependency management
	Dependencies []TemplateDependency `yaml:"dependencies,omitempty"` // Template dependencies
}

// BuildStep represents a single step in the AMI build process
type BuildStep struct {
	Name   string `yaml:"name"`
	Script string `yaml:"script"`
	// Optional timeout in seconds (default: 600)
	TimeoutSeconds int `yaml:"timeout_seconds,omitempty"`
}

// Validation represents a test to validate the AMI build
type Validation struct {
	Name    string `yaml:"name"`
	Command string `yaml:"command"`
	// Validation options (at least one required)
	Success  bool   `yaml:"success,omitempty"`  // Command must exit with code 0
	Contains string `yaml:"contains,omitempty"` // Output must contain string
	Equals   string `yaml:"equals,omitempty"`   // Output must exactly match
}

// BuildRequest contains parameters for building an AMI
type BuildRequest struct {
	TemplateName  string
	Template      Template
	Region        string
	Architecture  string
	Version       string    // Semantic version in format major.minor.patch
	DryRun        bool
	BuildID       string
	BuildType     string   // "scheduled", "manual", "ci"
	VpcID         string
	SubnetID      string
	SecurityGroup string
	CopyToRegions []string // Regions to copy the AMI to after building
	SetAsDefault  bool     // Whether to set this AMI as the default latest
}

// BuildResult contains the outcome of an AMI build
type BuildResult struct {
	TemplateID    string
	TemplateName  string
	Region        string
	Architecture  string
	AMIID         string
	CopiedAMIs    map[string]string // Region -> AMI ID map of copied AMIs
	BuildTime     time.Time
	BuildDuration time.Duration
	Status        string
	ErrorMessage  string
	Logs          string
	BuilderID     string
	ValidationLog string
	SourceAMI     string // Base AMI used as the source for this build
	Version       string // Semantic version of the template
}

// IsSuccessful returns true if the build was successful
func (b *BuildResult) IsSuccessful() bool {
	return b.Status == "completed" || b.Status == "dry-run"
}

// Builder handles the AMI creation process
type Builder struct {
	EC2Client       *ec2.Client
	SSMClient       *ssm.Client
	RegistryClient  *Registry
	BaseAMIs        map[string]map[string]string // region -> arch -> ami
	DefaultVPC      string
	DefaultSubnet   string
	BuilderRole     string
	BuilderProfile  string
	SecurityGroupID string
}

// Registry handles AMI version tracking and lookup
type Registry struct {
	SSMClient *ssm.Client
	// SSM parameter path prefix for AMI registry
	ParameterPrefix string
}

// Parser handles YAML template parsing and validation
type Parser struct {
	// Base AMI mappings for validation
	BaseAMIs map[string]map[string]string
}

// ValidatorOptions configures the validation process
type ValidatorOptions struct {
	FailFast     bool
	LogProgress  bool
	OutputFormat string
}

// ValidationResult contains the outcome of AMI validation
type ValidationResult struct {
	Successful      bool
	FailedChecks    []string
	SuccessfulTests int
	TotalTests      int
	Details         map[string]string
}

// Reference contains details for referencing an AMI
type Reference struct {
	AMIID        string
	Region       string
	Architecture string
	TemplateName string
	Version      string
	BuildDate    time.Time
	Tags         map[string]string
}
