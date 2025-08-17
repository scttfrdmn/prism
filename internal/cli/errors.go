// Package cli provides enhanced error handling and user-friendly error messages.
package cli

import (
	"fmt"
	"strings"
)

// UserFriendlyError wraps errors with helpful guidance for common issues
func UserFriendlyError(err error, context string) error {
	if err == nil {
		return nil
	}

	errorMsg := err.Error()

	// Transform common error patterns into helpful messages
	switch {
	case strings.Contains(errorMsg, "daemon not running"):
		return fmt.Errorf(`daemon not running

The CloudWorkstation background service is not running. To fix this:

1. Start the daemon:
   cws daemon start

2. Verify it's running:
   cws daemon status

3. If problems persist:
   cws daemon stop && cws daemon start

Need help? Check: https://github.com/scttfrdmn/cloudworkstation/blob/main/TROUBLESHOOTING.md`)

	case strings.Contains(errorMsg, "connection refused") || strings.Contains(errorMsg, "connect: connection refused"):
		return fmt.Errorf(`connection refused - daemon may not be running

CloudWorkstation can't connect to the background service. To fix this:

1. Check if daemon is running:
   cws daemon status

2. Start daemon if needed:
   cws daemon start

3. Check for port conflicts:
   lsof -i :8947

Original error: %v`, err)

	case strings.Contains(errorMsg, "no such host") || strings.Contains(errorMsg, "lookup failed"):
		return fmt.Errorf(`network connectivity issue

CloudWorkstation can't reach AWS services. To fix this:

1. Check internet connection
2. Verify AWS region is accessible:
   aws ec2 describe-availability-zones
3. Check firewall/proxy settings

Original error: %v`, err)

	case strings.Contains(errorMsg, "credentials not found") || strings.Contains(errorMsg, "UnauthorizedOperation"):
		return fmt.Errorf(`AWS credentials issue

CloudWorkstation can't access your AWS account. To fix this:

1. Configure AWS credentials:
   aws configure

2. Verify access:
   aws sts get-caller-identity

3. Check IAM permissions:
   https://github.com/scttfrdmn/cloudworkstation/blob/main/docs/DEMO_TESTER_SETUP.md

Original error: %v`, err)

	case strings.Contains(errorMsg, "VPC not found") || strings.Contains(errorMsg, "subnet not found"):
		return fmt.Errorf(`AWS network configuration issue

CloudWorkstation can't find your VPC or subnet. To fix this:

1. Use auto-discovery (recommended):
   cws launch template-name instance-name

2. Create default VPC if needed:
   aws ec2 create-default-vpc

3. Or specify manually:
   cws ami build template-name --vpc vpc-12345 --subnet subnet-67890

Original error: %v`, err)

	case strings.Contains(errorMsg, "insufficient capacity") || strings.Contains(errorMsg, "not available"):
		return fmt.Errorf(`AWS capacity issue

The requested instance type is not available. To fix this:

1. Try a different region:
   cws launch template-name instance-name --region us-east-1

2. Use a different instance size:
   cws launch template-name instance-name --size M

3. Try again later (capacity changes frequently)

Original error: %v`, err)

	case strings.Contains(errorMsg, "template not found"):
		return fmt.Errorf(`template not found

The specified template doesn't exist. To fix this:

1. List available templates:
   cws templates

2. Check template name spelling
3. Refresh template cache:
   rm -rf ~/.cloudworkstation/templates && cws templates

Original error: %v`, err)

	case strings.Contains(errorMsg, "validation failed"):
		return fmt.Errorf(`validation failed

The operation failed validation checks. To fix this:

1. Check the validation details above
2. Verify your inputs are correct
3. Try with different parameters

Need help? Check: https://github.com/scttfrdmn/cloudworkstation/blob/main/TROUBLESHOOTING.md

Original error: %v`, err)

	default:
		// For unknown errors, provide general guidance
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
}

// FormatErrorForCLI formats an error for command-line output with helpful guidance
func FormatErrorForCLI(err error, operation string) string {
	if err == nil {
		return ""
	}

	friendlyErr := UserFriendlyError(err, operation)
	return friendlyErr.Error()
}
