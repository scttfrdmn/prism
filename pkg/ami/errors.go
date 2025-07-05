// Package ami provides CloudWorkstation's AMI creation system.
package ami

import (
	"fmt"
	"strings"
)

// ErrorType represents the type of error that occurred during AMI building.
type ErrorType string

const (
	// ErrorTypeValidation indicates a validation error (template, parameters, etc.)
	ErrorTypeValidation ErrorType = "validation"
	
	// ErrorTypeInstance indicates an error with EC2 instance operations
	ErrorTypeInstance ErrorType = "instance"
	
	// ErrorTypeCommand indicates an error executing commands via SSM
	ErrorTypeCommand ErrorType = "command"
	
	// ErrorTypeImageCreation indicates an error creating or copying AMIs
	ErrorTypeImageCreation ErrorType = "image_creation"
	
	// ErrorTypeSSM indicates an error with the AWS SSM service
	ErrorTypeSSM ErrorType = "ssm"
	
	// ErrorTypeNetwork indicates a network configuration error
	ErrorTypeNetwork ErrorType = "network"
	
	// ErrorTypeConfiguration indicates a configuration error
	ErrorTypeConfiguration ErrorType = "configuration"
	
	// ErrorTypeRegistry indicates an error with the AMI registry
	ErrorTypeRegistry ErrorType = "registry"
	
	// ErrorTypeInternal indicates an unexpected internal error
	ErrorTypeInternal ErrorType = "internal"
	
	// ErrorTypeTemplateImport indicates a template import error
	ErrorTypeTemplateImport ErrorType = "template_import"
	
	// ErrorTypeTemplateExport indicates a template export error
	ErrorTypeTemplateExport ErrorType = "template_export"
	
	// ErrorTypeTemplateManagement indicates a template management error
	ErrorTypeTemplateManagement ErrorType = "template_management"
	
	// ErrorTypeDependency indicates a template dependency error
	ErrorTypeDependency ErrorType = "dependency"
)

// BuildError represents an error that occurred during AMI building.
type BuildError struct {
	Type      ErrorType
	Message   string
	Cause     error
	Context   map[string]string
	Retryable bool
}

// Error implements the error interface.
func (e *BuildError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %s: %v", e.Type, e.Message, e.Cause)
	}
	return fmt.Sprintf("%s: %s", e.Type, e.Message)
}

// Unwrap returns the underlying error.
func (e *BuildError) Unwrap() error {
	return e.Cause
}

// WithContext adds context information to the error.
func (e *BuildError) WithContext(key, value string) *BuildError {
	if e.Context == nil {
		e.Context = make(map[string]string)
	}
	e.Context[key] = value
	return e
}

// IsRetryable returns whether the error is retryable.
func (e *BuildError) IsRetryable() bool {
	return e.Retryable
}

// FormatErrorDetails returns a formatted string with error details.
func (e *BuildError) FormatErrorDetails() string {
	var details strings.Builder
	
	details.WriteString(fmt.Sprintf("Error Type: %s\n", e.Type))
	details.WriteString(fmt.Sprintf("Error Message: %s\n", e.Message))
	
	if e.Cause != nil {
		details.WriteString(fmt.Sprintf("Underlying Error: %v\n", e.Cause))
	}
	
	if len(e.Context) > 0 {
		details.WriteString("\nAdditional Context:\n")
		for k, v := range e.Context {
			details.WriteString(fmt.Sprintf("  %s: %s\n", k, v))
		}
	}
	
	details.WriteString(fmt.Sprintf("\nRetryable: %t\n", e.Retryable))
	
	details.WriteString("\nTroubleshooting:\n")
	switch e.Type {
	case ErrorTypeValidation:
		details.WriteString("  - Verify template YAML format is correct\n")
		details.WriteString("  - Check that all required fields are provided\n")
		details.WriteString("  - Ensure specified regions are supported\n")
		
	case ErrorTypeInstance:
		details.WriteString("  - Check EC2 service limits/quotas\n")
		details.WriteString("  - Verify VPC/subnet configuration\n")
		details.WriteString("  - Ensure security groups allow required traffic\n")
		
	case ErrorTypeCommand:
		details.WriteString("  - Verify command syntax is correct\n")
		details.WriteString("  - Check if command requires elevated privileges\n")
		details.WriteString("  - Verify packages mentioned in commands are available\n")
		
	case ErrorTypeImageCreation:
		details.WriteString("  - Check EC2 permissions for CreateImage API\n")
		details.WriteString("  - Ensure instance is in 'running' state when creating AMI\n")
		details.WriteString("  - Verify region has sufficient capacity for copying AMIs\n")
		
	case ErrorTypeSSM:
		details.WriteString("  - Verify SSM agent is installed and running on instance\n")
		details.WriteString("  - Check IAM permissions for SSM operations\n")
		details.WriteString("  - Ensure network connectivity between SSM and instance\n")
		
	case ErrorTypeNetwork:
		details.WriteString("  - Check VPC/subnet configuration\n")
		details.WriteString("  - Verify security group rules\n")
		details.WriteString("  - Ensure network interfaces are correctly configured\n")
		
	case ErrorTypeConfiguration:
		details.WriteString("  - Verify VPC, subnet, and security group values\n")
		details.WriteString("  - Check region and architecture compatibility\n")
		details.WriteString("  - Ensure correct AWS credentials are being used\n")
		
	case ErrorTypeRegistry:
		details.WriteString("  - Verify SSM parameter store permissions\n")
		details.WriteString("  - Check parameter path prefix configuration\n")
		
	case ErrorTypeInternal:
		details.WriteString("  - This appears to be a bug in the AMI builder\n")
		details.WriteString("  - Please report this issue with the full error details\n")
	}
	
	return details.String()
}

// NewBuildError creates a new BuildError.
func NewBuildError(errType ErrorType, message string, cause error) *BuildError {
	return &BuildError{
		Type:      errType,
		Message:   message,
		Cause:     cause,
		Retryable: false, // Default to not retryable
		Context:   make(map[string]string),
	}
}

// NewRetryableBuildError creates a new BuildError that is retryable.
func NewRetryableBuildError(errType ErrorType, message string, cause error) *BuildError {
	return &BuildError{
		Type:      errType,
		Message:   message,
		Cause:     cause,
		Retryable: true,
		Context:   make(map[string]string),
	}
}

// ValidationError creates a new validation error.
func ValidationError(message string, cause error) *BuildError {
	return NewBuildError(ErrorTypeValidation, message, cause)
}

// InstanceError creates a new instance error.
func InstanceError(message string, cause error) *BuildError {
	return NewBuildError(ErrorTypeInstance, message, cause)
}

// CommandError creates a new command execution error.
func CommandError(message string, cause error) *BuildError {
	return NewBuildError(ErrorTypeCommand, message, cause)
}

// ImageCreationError creates a new AMI creation error.
func ImageCreationError(message string, cause error) *BuildError {
	return NewBuildError(ErrorTypeImageCreation, message, cause)
}

// SSMError creates a new SSM service error.
func SSMError(message string, cause error) *BuildError {
	return NewBuildError(ErrorTypeSSM, message, cause)
}

// NetworkError creates a new network configuration error.
func NetworkError(message string, cause error) *BuildError {
	return NewBuildError(ErrorTypeNetwork, message, cause)
}

// ConfigurationError creates a new configuration error.
func ConfigurationError(message string, cause error) *BuildError {
	return NewBuildError(ErrorTypeConfiguration, message, cause)
}

// RegistryError creates a new registry error.
func RegistryError(message string, cause error) *BuildError {
	return NewBuildError(ErrorTypeRegistry, message, cause)
}

// InternalError creates a new internal error.
func InternalError(message string, cause error) *BuildError {
	return NewBuildError(ErrorTypeInternal, message, cause)
}

// TemplateImportError creates a new template import error.
func TemplateImportError(message string, cause error) *BuildError {
	return NewBuildError(ErrorTypeTemplateImport, message, cause)
}

// TemplateExportError creates a new template export error.
func TemplateExportError(message string, cause error) *BuildError {
	return NewBuildError(ErrorTypeTemplateExport, message, cause)
}

// TemplateManagementError creates a new template management error.
func TemplateManagementError(message string, cause error) *BuildError {
	return NewBuildError(ErrorTypeTemplateManagement, message, cause)
}

// DependencyError creates a new dependency error.
func DependencyError(message string, cause error) *BuildError {
	return NewBuildError(ErrorTypeDependency, message, cause)
}

// IsValidationError checks if the error is a validation error.
func IsValidationError(err error) bool {
	var buildErr *BuildError
	if err != nil && strings.Contains(err.Error(), string(ErrorTypeValidation)) {
		return true
	}
	if err, ok := err.(*BuildError); ok {
		buildErr = err
		return buildErr.Type == ErrorTypeValidation
	}
	return false
}

// IsNetworkError checks if the error is a network error.
func IsNetworkError(err error) bool {
	var buildErr *BuildError
	if err != nil && strings.Contains(err.Error(), string(ErrorTypeNetwork)) {
		return true
	}
	if err, ok := err.(*BuildError); ok {
		buildErr = err
		return buildErr.Type == ErrorTypeNetwork
	}
	return false
}

// IsRetryable checks if the error is retryable.
func IsRetryable(err error) bool {
	var buildErr *BuildError
	if err, ok := err.(*BuildError); ok {
		buildErr = err
		return buildErr.IsRetryable()
	}
	return false
}

// FormatError formats an error for display.
func FormatError(err error) string {
	var buildErr *BuildError
	if err, ok := err.(*BuildError); ok {
		buildErr = err
		return buildErr.FormatErrorDetails()
	}
	return fmt.Sprintf("Error: %v", err)
}