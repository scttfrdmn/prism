// Package cli provides enhanced error handling and user-friendly error messages.
package cli

import (
	"fmt"
	"strings"
)

// ErrorCategory represents the type of error for better categorization
type ErrorCategory string

const (
	ErrorCategoryDaemon      ErrorCategory = "daemon"
	ErrorCategoryNetwork     ErrorCategory = "network" 
	ErrorCategoryCredentials ErrorCategory = "credentials"
	ErrorCategoryProfile     ErrorCategory = "profile"
	ErrorCategoryLaunch      ErrorCategory = "launch"
	ErrorCategoryTemplate    ErrorCategory = "template"
	ErrorCategoryValidation  ErrorCategory = "validation"
	ErrorCategoryKeychain    ErrorCategory = "keychain"
	ErrorCategoryCapacity    ErrorCategory = "capacity"
	ErrorCategoryUnknown     ErrorCategory = "unknown"
)

// StructuredError provides categorized error information with context
type StructuredError struct {
	Category    ErrorCategory `json:"category"`
	Operation   string        `json:"operation"`
	Message     string        `json:"message"`
	Suggestions []string      `json:"suggestions"`
	OriginalErr error         `json:"original_error,omitempty"`
}

// Error implements the error interface
func (e *StructuredError) Error() string {
	return e.Message
}

// NewStructuredError creates a new structured error
func NewStructuredError(category ErrorCategory, operation, message string, suggestions []string, originalErr error) *StructuredError {
	return &StructuredError{
		Category:    category,
		Operation:   operation,
		Message:     message,
		Suggestions: suggestions,
		OriginalErr: originalErr,
	}
}

// ErrorHandler interface for different error types (Strategy Pattern - SOLID)
type ErrorHandler interface {
	CanHandle(errorMsg string) bool
	Handle(err error, context string) error
}

// DaemonErrorHandler handles daemon-related errors
type DaemonErrorHandler struct{}

func (h *DaemonErrorHandler) CanHandle(errorMsg string) bool {
	return strings.Contains(errorMsg, "daemon not running")
}

func (h *DaemonErrorHandler) Handle(err error, context string) error {
	return fmt.Errorf(`daemon not running

The CloudWorkstation background service is not responding. This is unusual since the daemon auto-starts.

🔧 Quick fixes:
1. Try your command again (daemon may be starting up)
2. Check daemon status: cws daemon status
3. If needed, restart daemon: cws daemon stop (next command will auto-start)

🔍 If this persists:
- Check for port conflicts: lsof -i :8947
- Verify binary permissions: ls -la $(which cws) $(which cwsd)
- Check logs: cws daemon logs

Need help? https://github.com/scttfrdmn/cloudworkstation/issues`)
}

// ConnectionErrorHandler handles connection-related errors
type ConnectionErrorHandler struct{}

func (h *ConnectionErrorHandler) CanHandle(errorMsg string) bool {
	return strings.Contains(errorMsg, "connection refused") || strings.Contains(errorMsg, "connect: connection refused")
}

func (h *ConnectionErrorHandler) Handle(err error, context string) error {
	return fmt.Errorf(`connection refused - daemon startup failed

CloudWorkstation's auto-start couldn't connect to the background service.

🔧 Quick fixes:
1. Wait a moment and try again (daemon may still be starting)
2. Check what's using port 8947: lsof -i :8947
3. Manual restart: cws daemon stop && cws templates

🔍 If this continues:
- Check if another CloudWorkstation is running
- Verify network permissions for localhost:8947
- Look for firewall blocking local connections

Original error: %v`, err)
}

// NetworkErrorHandler handles network connectivity errors
type NetworkErrorHandler struct{}

func (h *NetworkErrorHandler) CanHandle(errorMsg string) bool {
	return strings.Contains(errorMsg, "no such host") || strings.Contains(errorMsg, "lookup failed")
}

func (h *NetworkErrorHandler) Handle(err error, context string) error {
	return fmt.Errorf(`network connectivity issue

CloudWorkstation can't reach AWS services. To fix this:

1. Check internet connection
2. Verify AWS region is accessible:
   aws ec2 describe-availability-zones
3. Check firewall/proxy settings

Original error: %v`, err)
}

// CredentialsErrorHandler handles AWS credential errors
type CredentialsErrorHandler struct{}

func (h *CredentialsErrorHandler) CanHandle(errorMsg string) bool {
	return strings.Contains(errorMsg, "credentials not found") || strings.Contains(errorMsg, "UnauthorizedOperation")
}

func (h *CredentialsErrorHandler) Handle(err error, context string) error {
	return fmt.Errorf(`AWS credentials issue

CloudWorkstation can't access your AWS account. To fix this:

1. Configure AWS credentials:
   aws configure

2. Verify access:
   aws sts get-caller-identity

3. Check IAM permissions:
   https://github.com/scttfrdmn/cloudworkstation/blob/main/docs/DEMO_TESTER_SETUP.md

Original error: %v`, err)
}

// NetworkConfigErrorHandler handles VPC/subnet errors
type NetworkConfigErrorHandler struct{}

func (h *NetworkConfigErrorHandler) CanHandle(errorMsg string) bool {
	return strings.Contains(errorMsg, "VPC not found") || strings.Contains(errorMsg, "subnet not found")
}

func (h *NetworkConfigErrorHandler) Handle(err error, context string) error {
	return fmt.Errorf(`AWS network configuration issue

CloudWorkstation can't find your VPC or subnet. To fix this:

1. Use auto-discovery (recommended):
   cws launch template-name instance-name

2. Create default VPC if needed:
   aws ec2 create-default-vpc

3. Or specify manually:
   cws ami build template-name --vpc vpc-12345 --subnet subnet-67890

Original error: %v`, err)
}

// CapacityErrorHandler handles AWS capacity errors
type CapacityErrorHandler struct{}

func (h *CapacityErrorHandler) CanHandle(errorMsg string) bool {
	return strings.Contains(errorMsg, "insufficient capacity") || strings.Contains(errorMsg, "not available")
}

func (h *CapacityErrorHandler) Handle(err error, context string) error {
	return fmt.Errorf(`AWS capacity issue

The requested instance type is not available. To fix this:

1. Try a different region:
   cws launch template-name instance-name --region us-east-1

2. Use a different instance size:
   cws launch template-name instance-name --size M

3. Try again later (capacity changes frequently)

Original error: %v`, err)
}

// TemplateErrorHandler handles template-related errors
type TemplateErrorHandler struct{}

func (h *TemplateErrorHandler) CanHandle(errorMsg string) bool {
	return strings.Contains(errorMsg, "template not found")
}

func (h *TemplateErrorHandler) Handle(err error, context string) error {
	return fmt.Errorf(`template not found

The specified template doesn't exist. To fix this:

1. List available templates:
   cws templates

2. Check template name spelling
3. Refresh template cache:
   rm -rf ~/.cloudworkstation/templates && cws templates

Original error: %v`, err)
}

// ValidationErrorHandler handles validation errors
type ValidationErrorHandler struct{}

func (h *ValidationErrorHandler) CanHandle(errorMsg string) bool {
	return strings.Contains(errorMsg, "validation failed")
}

func (h *ValidationErrorHandler) Handle(err error, context string) error {
	return fmt.Errorf(`validation failed

The operation failed validation checks. To fix this:

1. Check the validation details above
2. Verify your inputs are correct
3. Try with different parameters

Need help? Check: https://github.com/scttfrdmn/cloudworkstation/blob/main/TROUBLESHOOTING.md

Original error: %v`, err)
}

// ProfileErrorHandler handles profile-related errors
type ProfileErrorHandler struct{}

func (h *ProfileErrorHandler) CanHandle(errorMsg string) bool {
	return strings.Contains(errorMsg, "profile not found") || 
		   strings.Contains(errorMsg, "profile") && strings.Contains(errorMsg, "not exist") ||
		   strings.Contains(errorMsg, "current profile")
}

func (h *ProfileErrorHandler) Handle(err error, context string) error {
	return fmt.Errorf(`profile configuration issue

CloudWorkstation can't find or use the specified profile.

🔧 Quick fixes:
1. List available profiles: cws profiles list
2. Create a new profile: cws profiles add personal my-account --aws-profile default --region us-east-1
3. Switch profiles: cws profiles switch <profile-id>

🔍 Profile troubleshooting:
- Verify AWS credentials: aws sts get-caller-identity
- Check profile file: cat ~/.cloudworkstation/profiles.json
- Reset to default: rm ~/.cloudworkstation/profiles.json && cws profiles list

Original error: %v`, err)
}

// LaunchErrorHandler handles instance launch errors
type LaunchErrorHandler struct{}

func (h *LaunchErrorHandler) CanHandle(errorMsg string) bool {
	return strings.Contains(errorMsg, "launch failed") ||
		   strings.Contains(errorMsg, "instance failed") ||
		   (strings.Contains(errorMsg, "UserData") && strings.Contains(errorMsg, "failed"))
}

func (h *LaunchErrorHandler) Handle(err error, context string) error {
	return fmt.Errorf(`instance launch failed

CloudWorkstation couldn't launch your research environment.

🔧 Common solutions:
1. Try different region: cws launch template-name instance-name --region us-west-2
2. Use different size: cws launch template-name instance-name --size S
3. Check template availability: cws templates

🔍 Advanced troubleshooting:
- Verify AWS quotas: aws service-quotas get-service-quota --service-code ec2 --quota-code L-1216C47A
- Check template validation: cws templates validate
- Try different instance type: cws launch template-name instance-name --instance-type t3.medium

Need template help? Each template shows its requirements with 'cws templates'

Original error: %v`, err)
}

// KeychainErrorHandler handles macOS keychain issues
type KeychainErrorHandler struct{}

func (h *KeychainErrorHandler) CanHandle(errorMsg string) bool {
	return strings.Contains(errorMsg, "keychain") ||
		   strings.Contains(errorMsg, "password") && strings.Contains(errorMsg, "required") ||
		   strings.Contains(errorMsg, "security")
}

func (h *KeychainErrorHandler) Handle(err error, context string) error {
	return fmt.Errorf(`keychain access issue

CloudWorkstation is having trouble with macOS keychain access.

🔧 Quick fixes:
1. This shouldn't happen with basic profiles - try again
2. Check profile type: cws profiles list
3. Use AWS CLI credentials: aws configure

🔍 If keychain prompts persist:
- Basic profiles should NOT require keychain access
- This may indicate a configuration issue
- Please report this: https://github.com/scttfrdmn/cloudworkstation/issues

Note: CloudWorkstation v0.4.4+ eliminates keychain prompts for normal usage.

Original error: %v`, err)
}

// DefaultErrorHandler handles unknown errors
type DefaultErrorHandler struct{}

func (h *DefaultErrorHandler) CanHandle(errorMsg string) bool {
	return true // Always can handle as fallback
}

func (h *DefaultErrorHandler) Handle(err error, context string) error {
	if context != "" {
		return fmt.Errorf(`%s failed

%v

Need help?
1. Check our troubleshooting guide:
   https://github.com/scttfrdmn/cloudworkstation/blob/main/TROUBLESHOOTING.md

2. Verify daemon status:
   cws daemon status

3. Check AWS credentials:
   aws sts get-caller-identity

4. Open an issue: https://github.com/scttfrdmn/cloudworkstation/issues`, context, err)
	}
	return err
}

// ErrorHandlerRegistry manages error handlers (Strategy Pattern - SOLID)
type ErrorHandlerRegistry struct {
	handlers []ErrorHandler
}

// NewErrorHandlerRegistry creates error handler registry
func NewErrorHandlerRegistry() *ErrorHandlerRegistry {
	return &ErrorHandlerRegistry{
		handlers: []ErrorHandler{
			&KeychainErrorHandler{},    // Check keychain issues first
			&ProfileErrorHandler{},     // Check profile issues early
			&LaunchErrorHandler{},      // Check launch-specific issues
			&DaemonErrorHandler{},      // Daemon connectivity issues
			&ConnectionErrorHandler{},  // Network connection issues
			&CredentialsErrorHandler{}, // AWS credential issues
			&NetworkConfigErrorHandler{}, // AWS network config issues
			&CapacityErrorHandler{},    // AWS capacity issues
			&TemplateErrorHandler{},    // Template-related issues
			&NetworkErrorHandler{},     // General network issues
			&ValidationErrorHandler{},  // Validation errors
			&DefaultErrorHandler{},     // Must be last as fallback
		},
	}
}

// Handle processes error using appropriate handler
func (r *ErrorHandlerRegistry) Handle(err error, context string) error {
	if err == nil {
		return nil
	}

	errorMsg := err.Error()
	for _, handler := range r.handlers {
		if handler.CanHandle(errorMsg) {
			return handler.Handle(err, context)
		}
	}

	// Fallback (should never reach here due to DefaultErrorHandler)
	return err
}

// UserFriendlyError wraps errors with helpful guidance using Strategy Pattern (SOLID: Single Responsibility)
func UserFriendlyError(err error, context string) error {
	registry := NewErrorHandlerRegistry()
	return registry.Handle(err, context)
}

// FormatErrorForCLI formats an error for command-line output with helpful guidance
func FormatErrorForCLI(err error, operation string) string {
	if err == nil {
		return ""
	}

	friendlyErr := UserFriendlyError(err, operation)
	return friendlyErr.Error()
}

// WrapError wraps an error with contextual information and suggestions
func WrapError(err error, operation string) error {
	if err == nil {
		return nil
	}
	
	// Use existing error handling system to get user-friendly message
	return UserFriendlyError(err, operation)
}
