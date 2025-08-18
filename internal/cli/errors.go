// Package cli provides enhanced error handling and user-friendly error messages.
package cli

import (
	"fmt"
	"strings"
)

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

The CloudWorkstation background service is not running. To fix this:

1. Start the daemon:
   cws daemon start

2. Verify it's running:
   cws daemon status

3. If problems persist:
   cws daemon stop && cws daemon start

Need help? Check: https://github.com/scttfrdmn/cloudworkstation/blob/main/TROUBLESHOOTING.md`)
}

// ConnectionErrorHandler handles connection-related errors
type ConnectionErrorHandler struct{}

func (h *ConnectionErrorHandler) CanHandle(errorMsg string) bool {
	return strings.Contains(errorMsg, "connection refused") || strings.Contains(errorMsg, "connect: connection refused")
}

func (h *ConnectionErrorHandler) Handle(err error, context string) error {
	return fmt.Errorf(`connection refused - daemon may not be running

CloudWorkstation can't connect to the background service. To fix this:

1. Check if daemon is running:
   cws daemon status

2. Start daemon if needed:
   cws daemon start

3. Check for port conflicts:
   lsof -i :8947

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
			&DaemonErrorHandler{},
			&ConnectionErrorHandler{},
			&NetworkErrorHandler{},
			&CredentialsErrorHandler{},
			&NetworkConfigErrorHandler{},
			&CapacityErrorHandler{},
			&TemplateErrorHandler{},
			&ValidationErrorHandler{},
			&DefaultErrorHandler{}, // Must be last as fallback
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
