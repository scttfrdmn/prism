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
	ErrorCategorySecurity    ErrorCategory = "security"
	ErrorCategoryAccess      ErrorCategory = "access"
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

The Prism background service is not responding. This is unusual since the daemon auto-starts.

🔧 Quick fixes:
1. Try your command again (daemon may be starting up)
2. Check daemon status: prism daemon status
3. If needed, restart daemon: prism daemon stop (next command will auto-start)

🔍 If this persists:
- Check for port conflicts: lsof -i :8947
- Verify binary permissions: ls -la $(which prism) $(which prismd)
- Check logs: prism daemon logs

Need help? https://github.com/scttfrdmn/prism/issues`)
}

// ConnectionErrorHandler handles connection-related errors
type ConnectionErrorHandler struct{}

func (h *ConnectionErrorHandler) CanHandle(errorMsg string) bool {
	return strings.Contains(errorMsg, "connection refused") || strings.Contains(errorMsg, "connect: connection refused")
}

func (h *ConnectionErrorHandler) Handle(err error, context string) error {
	return fmt.Errorf(`connection refused - daemon startup failed

Prism's auto-start couldn't connect to the background service.

🔧 Quick fixes:
1. Wait a moment and try again (daemon may still be starting)
2. Check what's using port 8947: lsof -i :8947
3. Manual restart: prism daemon stop && prism templates

🔍 If this continues:
- Check if another Prism is running
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

Prism can't reach AWS services. To fix this:

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

Prism can't access your AWS account. To fix this:

1. Configure AWS credentials:
   aws configure

2. Verify access:
   aws sts get-caller-identity

3. Check IAM permissions:
   https://github.com/scttfrdmn/prism/blob/main/docs/DEMO_TESTER_SETUP.md

Original error: %v`, err)
}

// NetworkConfigErrorHandler handles VPC/subnet errors
type NetworkConfigErrorHandler struct{}

func (h *NetworkConfigErrorHandler) CanHandle(errorMsg string) bool {
	return strings.Contains(errorMsg, "VPC not found") || strings.Contains(errorMsg, "subnet not found")
}

func (h *NetworkConfigErrorHandler) Handle(err error, context string) error {
	return fmt.Errorf(`AWS network configuration issue

Prism can't find your VPC or subnet. To fix this:

1. Use auto-discovery (recommended):
   prism launch template-name instance-name

2. Create default VPC if needed:
   aws ec2 create-default-vpc

3. Or specify manually:
   prism ami build template-name --vpc vpc-12345 --subnet subnet-67890

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
   prism launch template-name instance-name --region us-east-1

2. Use a different instance size:
   prism launch template-name instance-name --size M

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
   prism templates

2. Check template name spelling
3. Refresh template cache:
   rm -rf ~/.prism/templates && prism templates

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

Need help? Check: https://github.com/scttfrdmn/prism/blob/main/TROUBLESHOOTING.md

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

Prism can't find or use the specified profile.

🔧 Quick fixes:
1. List available profiles: prism profiles list
2. Create a new profile: prism profiles add personal my-account --aws-profile default --region us-east-1
3. Switch profiles: prism profiles switch <profile-id>

🔍 Profile troubleshooting:
- Verify AWS credentials: aws sts get-caller-identity
- Check profile file: cat ~/.prism/profiles.json
- Reset to default: rm ~/.prism/profiles.json && prism profiles list

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

Prism couldn't launch your research environment.

🔧 Common solutions:
1. Try different region: prism launch template-name instance-name --region us-west-2
2. Use different size: prism launch template-name instance-name --size S
3. Check template availability: prism templates

🔍 Advanced troubleshooting:
- Verify AWS quotas: aws service-quotas get-service-quota --service-code ec2 --quota-code L-1216C47A
- Check template validation: prism templates validate
- Try different instance type: prism launch template-name instance-name --instance-type t3.medium

Need template help? Each template shows its requirements with 'cws templates'

Original error: %v`, err)
}

// KeychainErrorHandler handles macOS keychain issues
type KeychainErrorHandler struct{}

func (h *KeychainErrorHandler) CanHandle(errorMsg string) bool {
	return strings.Contains(errorMsg, "keychain") ||
		(strings.Contains(errorMsg, "password") && strings.Contains(errorMsg, "required")) ||
		(strings.Contains(errorMsg, "security") && !strings.Contains(errorMsg, "security group"))
}

func (h *KeychainErrorHandler) Handle(err error, context string) error {
	return fmt.Errorf(`keychain access issue

Prism is having trouble with macOS keychain access.

🔧 Quick fixes:
1. This shouldn't happen with basic profiles - try again
2. Check profile type: prism profiles list
3. Use AWS CLI credentials: aws configure

🔍 If keychain prompts persist:
- Basic profiles should NOT require keychain access
- This may indicate a configuration issue
- Please report this: https://github.com/scttfrdmn/prism/issues

Note: Prism v0.4.4+ eliminates keychain prompts for normal usage.

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
   https://github.com/scttfrdmn/prism/blob/main/TROUBLESHOOTING.md

2. Verify daemon status:
   prism daemon status

3. Check AWS credentials:
   aws sts get-caller-identity

4. Open an issue: https://github.com/scttfrdmn/prism/issues`, context, err)
	}
	return err
}

// IPDetectionErrorHandler handles IP detection failures
type IPDetectionErrorHandler struct{}

func (h *IPDetectionErrorHandler) CanHandle(errorMsg string) bool {
	return strings.Contains(errorMsg, "failed to detect external IP") ||
		strings.Contains(errorMsg, "IP detection failed")
}

func (h *IPDetectionErrorHandler) Handle(err error, context string) error {
	return fmt.Errorf(`IP detection failed - web interfaces secured to SSH tunneling

Prism couldn't detect your external IP for secure direct web access.
Your instances are still fully functional, but web interfaces require SSH tunneling.

🔧 Web Interface Access:
  Jupyter:  ssh -L 8888:localhost:8888 user@<instance-ip>
  RStudio:  ssh -L 8787:localhost:8787 user@<instance-ip>
  Then open: http://localhost:8888 or http://localhost:8787

🌐 Alternative Solutions:
1. Check internet connectivity and try again
2. Use a VPN or different network
3. Manual IP refresh: prism access refresh (when available)

✅ This is secure by design - SSH tunneling provides encrypted access.

Original error: %v`, err)
}

// SecurityGroupErrorHandler handles security group access errors
type SecurityGroupErrorHandler struct{}

func (h *SecurityGroupErrorHandler) CanHandle(errorMsg string) bool {
	return strings.Contains(errorMsg, "security group") &&
		(strings.Contains(errorMsg, "UnauthorizedOperation") ||
			strings.Contains(errorMsg, "failed to add") ||
			strings.Contains(errorMsg, "access rules"))
}

func (h *SecurityGroupErrorHandler) Handle(err error, context string) error {
	return fmt.Errorf(`security group configuration failed

Prism couldn't configure secure access rules. This may be due to:

🔧 Possible Solutions:
1. Check AWS IAM permissions for EC2 security groups:
   - ec2:DescribeSecurityGroups
   - ec2:AuthorizeSecurityGroupIngress
   - ec2:RevokeSecurityGroupIngress

2. Verify your AWS credentials have sufficient permissions:
   aws sts get-caller-identity

3. Check if you're in a restricted AWS environment

💡 Fallback: Instances will use SSH-only access (secure by default)
   Connect with: ssh -L 8888:localhost:8888 user@<instance-ip>

Documentation: https://github.com/scttfrdmn/prism/blob/main/docs/DEMO_TESTER_SETUP.md

Original error: %v`, err)
}

// TemplateValidationErrorHandler handles template validation errors
type TemplateValidationErrorHandler struct{}

func (h *TemplateValidationErrorHandler) CanHandle(errorMsg string) bool {
	return strings.Contains(errorMsg, "template validation failed") ||
		strings.Contains(errorMsg, "invalid template")
}

func (h *TemplateValidationErrorHandler) Handle(err error, context string) error {
	return fmt.Errorf(`template validation failed

One or more templates have configuration issues.

🔧 Troubleshooting:
1. Check specific template: prism templates validate <template-name>
2. List all templates: prism templates
3. Check template syntax: look for YAML formatting issues

🔍 Common Issues:
- Missing required fields (name, description, base)
- Invalid package manager (must be: apt, dnf, conda, yum, apk)
- Broken inheritance chains
- Invalid port numbers or user configurations

📝 Fix Templates:
- Edit YAML files in templates/ directory
- Run validation after changes: make validate-templates
- Check template examples for proper syntax

Need help? Check the template documentation or file an issue.

Original error: %v`, err)
}

// WebAccessErrorHandler handles web interface access issues
type WebAccessErrorHandler struct{}

func (h *WebAccessErrorHandler) CanHandle(errorMsg string) bool {
	return (strings.Contains(errorMsg, "web interface") ||
		strings.Contains(errorMsg, "port 8888") ||
		strings.Contains(errorMsg, "jupyter") ||
		strings.Contains(errorMsg, "rstudio")) &&
		strings.Contains(errorMsg, "connection")
}

func (h *WebAccessErrorHandler) Handle(err error, context string) error {
	return fmt.Errorf(`web interface access issue

Having trouble accessing Jupyter, RStudio, or other web interfaces?

🌐 Access Methods (in order of preference):
1. Direct Access: http://<instance-ip>:8888
   - Available if your IP was detected during launch
   - Restricted to your specific IP for security

2. SSH Tunneling (always works):
   ssh -L 8888:localhost:8888 user@<instance-ip>
   Then: http://localhost:8888

🔍 Troubleshooting:
- Check instance is running: prism list
- Verify correct port (8888 for Jupyter, 8787 for RStudio)
- Try SSH tunnel if direct access fails
- Check your current IP: prism access status (when available)

🔧 IP Changed? 
If you moved networks or your IP changed:
- Use SSH tunneling as immediate solution
- Command available later: prism access refresh

Original error: %v`, err)
}

// ErrorHandlerRegistry manages error handlers (Strategy Pattern - SOLID)
type ErrorHandlerRegistry struct {
	handlers []ErrorHandler
}

// NewErrorHandlerRegistry creates error handler registry
func NewErrorHandlerRegistry() *ErrorHandlerRegistry {
	return &ErrorHandlerRegistry{
		handlers: []ErrorHandler{
			&KeychainErrorHandler{},           // Check keychain issues first
			&ProfileErrorHandler{},            // Check profile issues early
			&IPDetectionErrorHandler{},        // IP detection for web access
			&SecurityGroupErrorHandler{},      // Security group configuration
			&WebAccessErrorHandler{},          // Web interface access issues
			&TemplateValidationErrorHandler{}, // Template validation errors
			&LaunchErrorHandler{},             // Check launch-specific issues
			&DaemonErrorHandler{},             // Daemon connectivity issues
			&ConnectionErrorHandler{},         // Network connection issues
			&CredentialsErrorHandler{},        // AWS credential issues
			&NetworkConfigErrorHandler{},      // AWS network config issues
			&CapacityErrorHandler{},           // AWS capacity issues
			&TemplateErrorHandler{},           // Template-related issues
			&NetworkErrorHandler{},            // General network issues
			&ValidationErrorHandler{},         // Validation errors
			&DefaultErrorHandler{},            // Must be last as fallback
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
