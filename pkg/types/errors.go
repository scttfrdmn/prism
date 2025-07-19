package types

import (
	"fmt"
)

// ErrorCode defines specific error types for API operations
type ErrorCode string

const (
	// Generic error codes
	ErrUnknown          ErrorCode = "unknown_error"      // Unknown or unclassified error
	
	// HTTP-like status errors
	ErrNotFound         ErrorCode = "not_found"          // Resource not found (404)
	ErrUnauthorized     ErrorCode = "unauthorized"       // Authentication required (401)
	ErrForbidden        ErrorCode = "forbidden"          // Permission denied (403)
	ErrInvalidParameters ErrorCode = "invalid_parameters" // Bad request parameters (400)
	ErrConflict         ErrorCode = "conflict"           // Resource conflict (409)
	ErrServerError      ErrorCode = "server_error"       // Internal server error (500)
	
	// Operation errors
	ErrTimeout          ErrorCode = "timeout"            // Operation timed out
	ErrCanceled         ErrorCode = "canceled"           // Operation was canceled
	ErrRateLimited      ErrorCode = "rate_limited"       // Too many requests
	ErrNetworkError     ErrorCode = "network_error"      // Network communication error
	
	// Resource errors
	ErrResourceExists   ErrorCode = "resource_exists"    // Resource already exists
	ErrResourceInUse    ErrorCode = "resource_in_use"    // Resource is in use
	ErrResourceLocked   ErrorCode = "resource_locked"    // Resource is locked
	ErrQuotaExceeded    ErrorCode = "quota_exceeded"    // Account quota exceeded
	
	// Authentication/authorization errors
	ErrInvalidCredentials ErrorCode = "invalid_credentials" // Invalid login credentials
	ErrAccountDisabled    ErrorCode = "account_disabled"    // User account is disabled
	ErrPermissionDenied   ErrorCode = "permission_denied"   // User lacks permission
	ErrSessionExpired     ErrorCode = "session_expired"     // Session has expired
	
	// Validation errors
	ErrValidationFailed   ErrorCode = "validation_failed"   // Input validation failed
	ErrInvalidFormat      ErrorCode = "invalid_format"      // Input in wrong format
	ErrMissingRequired    ErrorCode = "missing_required"    // Required field missing
	
	// AWS specific errors
	ErrAWSError           ErrorCode = "aws_error"           // AWS API error
	ErrAWSNotConfigured   ErrorCode = "aws_not_configured"  // AWS credentials not configured
	ErrRegionNotConfigured ErrorCode = "region_not_configured" // AWS region not configured
)

// APIError represents an API error with additional context
type APIError struct {
	Code        ErrorCode            `json:"code"`
	Message     string               `json:"message"`
	Details     string               `json:"details,omitempty"`
	RequestID   string               `json:"request_id,omitempty"`
	Operation   string               `json:"operation,omitempty"`
	Resource    string               `json:"resource,omitempty"`      // Resource type (instance, volume, etc.)
	ResourceID  string               `json:"resource_id,omitempty"`   // Specific resource ID (if applicable)
	Field       string               `json:"field,omitempty"`         // Field name for validation errors
	Validation  map[string]string    `json:"validation,omitempty"`    // Field validation errors
	StatusCode  int                  `json:"status_code,omitempty"`   // HTTP status code
	Retryable   bool                 `json:"retryable,omitempty"`     // Whether operation can be retried
	Suggestions []string             `json:"suggestions,omitempty"`   // Suggested actions to resolve
	cause       error                // Not serialized - underlying error
}

// getErrorCodeFromStatusCode converts HTTP status codes to appropriate error codes
// This is kept for backwards compatibility with tests
func getErrorCodeFromStatusCode(statusCode int) ErrorCode {
	return GetErrorCodeFromStatusCode(statusCode)
}

// GetErrorCodeFromStatusCode converts HTTP status codes to appropriate error codes
func GetErrorCodeFromStatusCode(statusCode int) ErrorCode {
	switch {
	case statusCode == 404:
		return ErrNotFound
	case statusCode == 401:
		return ErrUnauthorized
	case statusCode == 403:
		return ErrForbidden
	case statusCode == 409:
		return ErrConflict
	case statusCode == 429:
		return ErrRateLimited
	case statusCode >= 400 && statusCode < 500:
		return ErrInvalidParameters
	case statusCode >= 500 && statusCode < 600:
		return ErrServerError
	default:
		return ErrUnknown
	}
}

// Error implements the error interface
func (e APIError) Error() string {
	msg := fmt.Sprintf("%s: %s", e.Code, e.Message)
	if e.Details != "" {
		msg += fmt.Sprintf(" (%s)", e.Details)
	}
	if e.Operation != "" {
		msg = fmt.Sprintf("%s in operation %s", msg, e.Operation)
	}
	if e.RequestID != "" {
		msg += fmt.Sprintf(" [request-id: %s]", e.RequestID)
	}
	return msg
}

// Unwrap implements the errors.Unwrap interface
func (e APIError) Unwrap() error {
	return e.cause
}

// NewAPIError creates a new APIError with the given code, message, and cause
func NewAPIError(code ErrorCode, message string, cause error) APIError {
	return APIError{
		Code:      code,
		Message:   message,
		cause:     cause,
		Retryable: isRetryableError(code),
	}
}

// isRetryableError determines if an error with the given code can be retried
func isRetryableError(code ErrorCode) bool {
	switch code {
	case ErrTimeout, ErrNetworkError, ErrServerError, ErrRateLimited:
		return true
	default:
		return false
	}
}

// WithDetails adds details to an APIError and returns it
func (e APIError) WithDetails(details string) APIError {
	e.Details = details
	return e
}

// WithRequestID adds a request ID to an APIError and returns it
func (e APIError) WithRequestID(requestID string) APIError {
	e.RequestID = requestID
	return e
}

// WithOperation adds an operation name to an APIError and returns it
func (e APIError) WithOperation(operation string) APIError {
	e.Operation = operation
	return e
}

// WithResource adds resource information to an APIError and returns it
func (e APIError) WithResource(resourceType string, resourceID string) APIError {
	e.Resource = resourceType
	e.ResourceID = resourceID
	return e
}

// WithField adds field information for validation errors and returns it
func (e APIError) WithField(field string) APIError {
	e.Field = field
	return e
}

// WithStatusCode adds HTTP status code information and returns it
func (e APIError) WithStatusCode(statusCode int) APIError {
	e.StatusCode = statusCode
	return e
}

// WithValidation adds field validation errors and returns it
func (e APIError) WithValidation(validationErrors map[string]string) APIError {
	e.Validation = validationErrors
	return e
}

// WithSuggestions adds suggested actions to resolve the error and returns it
func (e APIError) WithSuggestions(suggestions ...string) APIError {
	e.Suggestions = suggestions
	return e
}

// IsRetryable returns whether the error can be retried
func (e APIError) IsRetryable() bool {
	return e.Retryable
}

// IsNotFound returns whether this is a not found error
func (e APIError) IsNotFound() bool {
	return e.Code == ErrNotFound
}

// IsAuthError returns whether this is an authentication or authorization error
func (e APIError) IsAuthError() bool {
	return e.Code == ErrUnauthorized || e.Code == ErrForbidden || 
	       e.Code == ErrInvalidCredentials || e.Code == ErrPermissionDenied || 
	       e.Code == ErrSessionExpired
}

// NewNotFoundError creates a standard not found error for a resource
func NewNotFoundError(resource, resourceID string) APIError {
	message := fmt.Sprintf("%s not found", resource)
	if resourceID != "" {
		message = fmt.Sprintf("%s with ID '%s' not found", resource, resourceID)
	}
	
	return NewAPIError(ErrNotFound, message, nil).WithResource(resource, resourceID)
}

// NewValidationError creates a standard validation error
func NewValidationError(message string, validationErrors map[string]string) APIError {
	if message == "" {
		message = "Validation failed"
	}
	
	return NewAPIError(ErrValidationFailed, message, nil).WithValidation(validationErrors)
}

// NewPermissionError creates a standard permission error
func NewPermissionError(resource string, operation string) APIError {
	message := fmt.Sprintf("You don't have permission to %s this %s", operation, resource)
	return NewAPIError(ErrPermissionDenied, message, nil).WithResource(resource, "").WithOperation(operation)
}

// NewAwsError creates a standard AWS error wrapper
func NewAwsError(awsErr error, operation string) APIError {
	return NewAPIError(ErrAWSError, fmt.Sprintf("AWS operation failed: %v", awsErr), awsErr).WithOperation(operation)
}

// IsErrorCode checks if an error is an APIError with the specified code
func IsErrorCode(err error, code ErrorCode) bool {
	if apiErr, ok := err.(APIError); ok {
		return apiErr.Code == code
	}
	return false
}

// IsNotFoundErr checks if an error is a not found error
func IsNotFoundErr(err error) bool {
	return IsErrorCode(err, ErrNotFound)
}

// IsUnauthorizedErr checks if an error is an unauthorized error
func IsUnauthorizedErr(err error) bool {
	return IsErrorCode(err, ErrUnauthorized)
}

// IsForbiddenErr checks if an error is a forbidden error
func IsForbiddenErr(err error) bool {
	return IsErrorCode(err, ErrForbidden)
}

// IsValidationErr checks if an error is a validation error
func IsValidationErr(err error) bool {
	return IsErrorCode(err, ErrValidationFailed)
}

// IsRetryableErr checks if an error can be retried
func IsRetryableErr(err error) bool {
	if apiErr, ok := err.(APIError); ok {
		return apiErr.IsRetryable()
	}
	return false
}

// ExtractValidationErrors extracts validation errors from an APIError
// Returns nil if the error is not a validation error or has no validation details
func ExtractValidationErrors(err error) map[string]string {
	if apiErr, ok := err.(APIError); ok && apiErr.Code == ErrValidationFailed {
		return apiErr.Validation
	}
	return nil
}

// FormatErrorForDisplay formats an error for display to users
// This provides a more user-friendly message than the raw error
func FormatErrorForDisplay(err error) string {
	if apiErr, ok := err.(APIError); ok {
		msg := apiErr.Message
		
		// Add field information for validation errors
		if apiErr.Code == ErrValidationFailed && apiErr.Field != "" {
			msg = fmt.Sprintf("%s: %s", apiErr.Field, msg)
		}
		
		// Add helpful suggestions if available
		if len(apiErr.Suggestions) > 0 {
			msg = fmt.Sprintf("%s\nSuggested action: %s", msg, apiErr.Suggestions[0])
		}
		
		return msg
	}
	
	return err.Error()
}