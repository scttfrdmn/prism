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
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
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
	Version       string // Semantic version in format major.minor.patch
	DryRun        bool
	BuildID       string
	BuildType     string // "scheduled", "manual", "ci"
	VpcID         string
	SubnetID      string
	SecurityGroup string
	CopyToRegions []string // Regions to copy the AMI to after building
	SetAsDefault  bool     // Whether to set this AMI as the default latest
}

// InstanceSaveRequest contains parameters for saving a running instance as an AMI
type InstanceSaveRequest struct {
	InstanceID    string            // EC2 instance ID to save
	InstanceName  string            // CloudWorkstation instance name
	TemplateName  string            // Name for the new template
	Description   string            // Template description
	CopyToRegions []string          // Regions to copy AMI
	Tags          map[string]string // Custom tags
	ProjectID     string            // Associated project (Phase 4)
	Public        bool              // Allow public sharing
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

// NewParser creates a new AMI template parser
func NewParser() *Parser {
	return &Parser{
		BaseAMIs: make(map[string]map[string]string),
	}
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

// Validator handles AMI validation
type Validator struct {
	SSMClient *ssm.Client
	Options   ValidatorOptions
}

// NewValidator creates a new AMI validator
func NewValidator(ssmClient *ssm.Client, options ValidatorOptions) *Validator {
	return &Validator{
		SSMClient: ssmClient,
		Options:   options,
	}
}

// ValidateAMI validates an AMI build
func (v *Validator) ValidateAMI(instanceID string, template *Template) (*ValidationResult, error) {
	result := &ValidationResult{
		Details: make(map[string]string),
	}

	for i, validation := range template.Validation {
		if v.Options.LogProgress {
			fmt.Printf("Running validation %d: %s\n", i+1, validation.Name)
		}

		// Execute validation command via SSM
		passed, detail, err := v.executeValidationCommand(instanceID, validation)
		if err != nil {
			result.FailedChecks = append(result.FailedChecks, validation.Name)
			result.Details[validation.Name] = fmt.Sprintf("ERROR: %v", err)
			continue
		}

		if passed {
			result.SuccessfulTests++
			result.Details[validation.Name] = detail
		} else {
			result.FailedChecks = append(result.FailedChecks, validation.Name)
			result.Details[validation.Name] = detail
		}
	}

	result.TotalTests = len(template.Validation)
	result.Successful = result.SuccessfulTests == result.TotalTests

	return result, nil
}

// executeValidationCommand executes a single validation command via SSM
func (v *Validator) executeValidationCommand(instanceID string, validation Validation) (bool, string, error) {
	if v.SSMClient == nil {
		return false, "SSM client not configured", fmt.Errorf("SSM client required for validation")
	}

	// Send command via SSM
	sendInput := &ssm.SendCommandInput{
		InstanceIds:  []string{instanceID},
		DocumentName: aws.String("AWS-RunShellScript"),
		Parameters: map[string][]string{
			"commands": {validation.Command},
		},
		TimeoutSeconds: aws.Int32(60), // 60 second timeout for validation commands
	}

	sendOutput, err := v.SSMClient.SendCommand(context.Background(), sendInput)
	if err != nil {
		return false, "", fmt.Errorf("failed to send SSM command: %w", err)
	}

	commandID := sendOutput.Command.CommandId

	// Wait for command completion (max 70 seconds)
	maxWait := 70
	for i := 0; i < maxWait; i++ {
		getInput := &ssm.GetCommandInvocationInput{
			CommandId:  commandID,
			InstanceId: aws.String(instanceID),
		}

		getOutput, err := v.SSMClient.GetCommandInvocation(context.Background(), getInput)
		if err != nil {
			// Command not ready yet, wait and retry
			time.Sleep(1 * time.Second)
			continue
		}

		// Check command status
		status := string(getOutput.Status)
		if status == "Success" || status == "Failed" {
			// Command completed
			exitCode := int(getOutput.ResponseCode)
			output := ""
			if getOutput.StandardOutputContent != nil {
				output = *getOutput.StandardOutputContent
			}

			// Determine if validation passed
			passed := false
			detail := ""

			if validation.Success {
				// Check exit code
				if exitCode == 0 {
					passed = true
					detail = "PASS: Command exited with code 0"
				} else {
					detail = fmt.Sprintf("FAIL: Command exited with code %d", exitCode)
				}
			}

			if validation.Contains != "" {
				// Check output contains string
				if strings.Contains(output, validation.Contains) {
					passed = true
					detail = fmt.Sprintf("PASS: Output contains '%s'", validation.Contains)
				} else {
					detail = fmt.Sprintf("FAIL: Output does not contain '%s'. Got: %s", validation.Contains, output)
				}
			}

			// If both checks specified, both must pass
			if validation.Success && validation.Contains != "" {
				if exitCode == 0 && strings.Contains(output, validation.Contains) {
					passed = true
					detail = fmt.Sprintf("PASS: Exit code 0 and output contains '%s'", validation.Contains)
				} else {
					detail = fmt.Sprintf("FAIL: Exit code %d or missing '%s'", exitCode, validation.Contains)
				}
			}

			return passed, detail, nil
		}

		// Still running, wait and retry
		time.Sleep(1 * time.Second)
	}

	return false, "TIMEOUT: Command did not complete in time", fmt.Errorf("command timed out after %d seconds", maxWait)
}

// FormatValidationResult formats a validation result for output
func (v *Validator) FormatValidationResult(result *ValidationResult) string {
	if result.Successful {
		return fmt.Sprintf("✅ All %d validations passed", result.TotalTests)
	} else {
		return fmt.Sprintf("❌ %d/%d validations failed: %v",
			result.TotalTests-result.SuccessfulTests,
			result.TotalTests,
			result.FailedChecks)
	}
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
