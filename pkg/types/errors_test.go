package types

import (
	"errors"
	"strings"
	"testing"
)

func TestAPIError(t *testing.T) {
	// Test basic error
	err := APIError{
		Code:    ErrNotFound,
		Message: "Resource not found",
	}
	
	if err.Error() != "not_found: Resource not found" {
		t.Errorf("Unexpected error message: %s", err.Error())
	}
	
	// Test with details
	err = APIError{
		Code:    ErrUnauthorized,
		Message: "Unauthorized",
		Details: "Invalid credentials",
	}
	
	if !strings.Contains(err.Error(), "Invalid credentials") {
		t.Errorf("Error message missing details: %s", err.Error())
	}
	
	// Test with operation
	err = APIError{
		Code:      ErrInvalidParameters,
		Message:   "Invalid parameters",
		Operation: "LaunchInstance",
	}
	
	if !strings.Contains(err.Error(), "operation LaunchInstance") {
		t.Errorf("Error message missing operation: %s", err.Error())
	}
	
	// Test with request ID
	err = APIError{
		Code:      ErrServerError,
		Message:   "Server error",
		RequestID: "req-12345",
	}
	
	if !strings.Contains(err.Error(), "req-12345") {
		t.Errorf("Error message missing request ID: %s", err.Error())
	}
	
	// Test with underlying error
	originalErr := errors.New("original error")
	err = APIError{
		Code:    ErrNetworkError,
		Message: "Network error",
		cause:   originalErr,
	}
	
	if errors.Unwrap(err) != originalErr {
		t.Errorf("Unwrap did not return original error")
	}
}

func TestNewAPIError(t *testing.T) {
	// Test creating new error
	originalErr := errors.New("underlying error")
	err := NewAPIError(ErrTimeout, "Operation timed out", originalErr)
	
	if err.Code != ErrTimeout {
		t.Errorf("Expected code %s, got %s", ErrTimeout, err.Code)
	}
	
	if err.Message != "Operation timed out" {
		t.Errorf("Expected message 'Operation timed out', got '%s'", err.Message)
	}
	
	if errors.Unwrap(err) != originalErr {
		t.Errorf("Unwrap did not return original error")
	}
}

func TestAPIErrorChaining(t *testing.T) {
	// Test method chaining for building errors
	err := NewAPIError(ErrNotFound, "Resource not found", nil).
		WithDetails("The specified instance does not exist").
		WithOperation("GetInstance").
		WithRequestID("req-12345")
	
	expectedSubstrings := []string{
		"not_found",
		"Resource not found",
		"The specified instance does not exist",
		"operation GetInstance",
		"req-12345",
	}
	
	errorMsg := err.Error()
	for _, substr := range expectedSubstrings {
		if !strings.Contains(errorMsg, substr) {
			t.Errorf("Error message missing expected substring '%s': %s", substr, errorMsg)
		}
	}
}

func TestNewSpecializedErrors(t *testing.T) {
	// Test NewNotFoundError
	notFoundErr := NewNotFoundError("Instance", "i-12345")
	if notFoundErr.Code != ErrNotFound {
		t.Errorf("Expected code %s, got %s", ErrNotFound, notFoundErr.Code)
	}
	if notFoundErr.Resource != "Instance" {
		t.Errorf("Expected resource 'Instance', got '%s'", notFoundErr.Resource)
	}
	if notFoundErr.ResourceID != "i-12345" {
		t.Errorf("Expected resource ID 'i-12345', got '%s'", notFoundErr.ResourceID)
	}
	if !strings.Contains(notFoundErr.Message, "Instance with ID 'i-12345' not found") {
		t.Errorf("Unexpected message: %s", notFoundErr.Message)
	}

	// Test NewValidationError
	validationErrors := map[string]string{
		"name": "Name is required",
		"size": "Size must be between 1 and 10",
	}
	validationErr := NewValidationError("Validation failed", validationErrors)
	if validationErr.Code != ErrValidationFailed {
		t.Errorf("Expected code %s, got %s", ErrValidationFailed, validationErr.Code)
	}
	if len(validationErr.Validation) != 2 {
		t.Errorf("Expected 2 validation errors, got %d", len(validationErr.Validation))
	}
	if validationErr.Validation["name"] != "Name is required" {
		t.Errorf("Expected validation error 'Name is required', got '%s'", validationErr.Validation["name"])
	}

	// Test NewPermissionError
	permissionErr := NewPermissionError("Instance", "delete")
	if permissionErr.Code != ErrPermissionDenied {
		t.Errorf("Expected code %s, got %s", ErrPermissionDenied, permissionErr.Code)
	}
	if permissionErr.Resource != "Instance" {
		t.Errorf("Expected resource 'Instance', got '%s'", permissionErr.Resource)
	}
	if permissionErr.Operation != "delete" {
		t.Errorf("Expected operation 'delete', got '%s'", permissionErr.Operation)
	}
}

func TestErrorTypeChecking(t *testing.T) {
	// Create different error types
	notFoundErr := NewNotFoundError("Instance", "i-12345")
	validationErr := NewValidationError("Invalid input", nil)
	_ = NewAwsError(errors.New("AWS API error"), "DescribeInstances")
	_ = NewPermissionError("Volume", "attach")
	standardErr := errors.New("standard error")
	
	// Test IsErrorCode
	if !IsErrorCode(notFoundErr, ErrNotFound) {
		t.Error("Expected IsErrorCode to return true for not found error")
	}
	if IsErrorCode(notFoundErr, ErrServerError) {
		t.Error("Expected IsErrorCode to return false for incorrect error code")
	}
	if IsErrorCode(standardErr, ErrNotFound) {
		t.Error("Expected IsErrorCode to return false for standard error")
	}
	
	// Test specialized type checking
	if !IsNotFoundErr(notFoundErr) {
		t.Error("Expected IsNotFoundErr to return true for not found error")
	}
	if IsNotFoundErr(validationErr) {
		t.Error("Expected IsNotFoundErr to return false for validation error")
	}
	
	// Test validation error extraction
	validationWithDetails := NewValidationError("", map[string]string{
		"name": "Name is required",
		"age": "Age must be positive",
	})
	
	validationMap := ExtractValidationErrors(validationWithDetails)
	if len(validationMap) != 2 {
		t.Errorf("Expected 2 validation errors, got %d", len(validationMap))
	}
	
	// Test non-validation error returns nil
	if ExtractValidationErrors(notFoundErr) != nil {
		t.Error("Expected ExtractValidationErrors to return nil for non-validation error")
	}
}

func TestErrorCodeFromStatusCode(t *testing.T) {
	tests := []struct {
		statusCode int
		expected   ErrorCode
	}{
		{404, ErrNotFound},
		{401, ErrUnauthorized},
		{403, ErrForbidden},  // Updated to match new mapping
		{400, ErrInvalidParameters},
		{409, ErrConflict},
		{429, ErrRateLimited},
		{500, ErrServerError},
		{502, ErrServerError},
		{418, ErrUnknown}, // I'm a teapot
	}
	
	for _, test := range tests {
		code := GetErrorCodeFromStatusCode(test.statusCode)
		if code != test.expected {
			t.Errorf("For status code %d, expected error code %s, got %s", 
				test.statusCode, test.expected, code)
		}
	}
}