package errors

import (
	"strings"

	"github.com/scttfrdmn/cloudworkstation/pkg/types"
)

// ExtractOperationFromPath extracts the operation name from a request path
// e.g., "/api/v1/instances/my-instance/start" -> "StartInstance"
func ExtractOperationFromPath(path string) string {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) < 1 || (len(parts) == 1 && parts[0] == "") {
		return "Unknown"
	}

	// Skip the API version prefix if present
	start := 0
	if parts[0] == "api" && len(parts) > 2 {
		start = 2
	}

	// Base resource type (e.g., "instances")
	if len(parts) <= start {
		return "Unknown"
	}
	resource := parts[start]
	singular := strings.TrimSuffix(resource, "s") // Convert plural to singular

	// Determine the operation
	var operation string
	if len(parts) == start+1 {
		// /api/v1/instances -> List
		operation = "List"
	} else if len(parts) == start+2 {
		// /api/v1/instances/my-instance -> Get
		// /volumes/create -> Create
		if parts[start+1] == "create" {
			operation = "Create"
		} else {
			operation = "Get"
		}
	} else if len(parts) == start+3 {
		// /api/v1/instances/my-instance/start -> Start
		operation = strings.ToUpper(string(parts[start+2][0])) + parts[start+2][1:]
	} else {
		// Fallback
		operation = "Operate"
	}

	// Capitalize the resource name
	resourceName := strings.ToUpper(string(singular[0])) + singular[1:]

	// Format as CamelCase operation name: ListInstances, GetInstance, StartInstance, etc.
	return operation + resourceName
}

// FromNetworkError creates an APIError from a network error
func FromNetworkError(err error, operation string) types.APIError {
	return types.NewAPIError(
		types.ErrNetworkError,
		"Network connection failed",
		err,
	).WithOperation(operation)
}

// FromTimeout creates an APIError from a timeout error
func FromTimeout(err error, operation string, resource string) types.APIError {
	return types.NewAPIError(
		types.ErrTimeout,
		"Operation timed out",
		err,
	).WithOperation(operation).WithResource(resource, "")
}

// FromAuthError creates an APIError from an authentication error
func FromAuthError(err error, detail string) types.APIError {
	message := "Authentication failed"
	if detail != "" {
		message += ": " + detail
	}

	return types.NewAPIError(
		types.ErrUnauthorized,
		message,
		err,
	)
}

// FromAWSError creates an APIError from an AWS error
func FromAWSError(err error, operation string, resource string) types.APIError {
	return types.NewAPIError(
		types.ErrAWSError,
		"AWS operation failed",
		err,
	).WithOperation(operation).WithResource(resource, "")
}

// FromValidationErrors creates an APIError from validation errors
func FromValidationErrors(validationErrors map[string]string, message string) types.APIError {
	if message == "" {
		message = "Validation failed"
	}

	return types.NewAPIError(
		types.ErrValidationFailed,
		message,
		nil,
	).WithValidation(validationErrors)
}

// FromPermissionDenied creates an APIError for permission denied
func FromPermissionDenied(operation string, resource string) types.APIError {
	return types.NewAPIError(
		types.ErrPermissionDenied,
		"Permission denied",
		nil,
	).WithOperation(operation).WithResource(resource, "")
}

// FromRequestError creates an APIError from a request formatting error
func FromRequestError(err error, operation string) types.APIError {
	return types.NewAPIError(
		types.ErrInvalidParameters,
		"Invalid request parameters",
		err,
	).WithOperation(operation)
}
