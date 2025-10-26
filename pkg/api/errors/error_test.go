package errors

import (
	"errors"
	"testing"

	"github.com/scttfrdmn/prism/pkg/types"
)

func TestExtractOperationFromPath(t *testing.T) {
	tests := []struct {
		path     string
		expected string
	}{
		{"/api/v1/instances", "ListInstance"},
		{"/api/v1/instances/my-instance", "GetInstance"},
		{"/api/v1/instances/my-instance/start", "StartInstance"},
		{"/api/v1/instances/my-instance/stop", "StopInstance"},
		{"/api/v1/volumes", "ListVolume"},
		{"/api/v1/volumes/my-volume", "GetVolume"},
		{"/api/v1/templates/my-template", "GetTemplate"},
		{"/instances", "ListInstance"},
		{"/volumes/create", "CreateVolume"},
		{"/random/path/with/many/parts", "OperateRandom"},
		{"", "Unknown"},
		{"/", "Unknown"},
	}

	for _, tt := range tests {
		result := ExtractOperationFromPath(tt.path)
		if result != tt.expected {
			t.Errorf("ExtractOperationFromPath(%q) = %q, expected %q", tt.path, result, tt.expected)
		}
	}
}

func TestFromNetworkError(t *testing.T) {
	err := errors.New("connection refused")
	apiErr := FromNetworkError(err, "ListInstances")

	if apiErr.Code != types.ErrNetworkError {
		t.Errorf("Expected code %s, got %s", types.ErrNetworkError, apiErr.Code)
	}

	if apiErr.Operation != "ListInstances" {
		t.Errorf("Expected operation 'ListInstances', got '%s'", apiErr.Operation)
	}

	if errors.Unwrap(apiErr) != err {
		t.Error("Unwrap did not return original error")
	}
}

func TestFromValidationErrors(t *testing.T) {
	validationErrors := map[string]string{
		"name": "Name is required",
		"size": "Size must be between 1 and 10",
	}

	apiErr := FromValidationErrors(validationErrors, "Input validation failed")

	if apiErr.Code != types.ErrValidationFailed {
		t.Errorf("Expected code %s, got %s", types.ErrValidationFailed, apiErr.Code)
	}

	if len(apiErr.Validation) != 2 {
		t.Errorf("Expected 2 validation errors, got %d", len(apiErr.Validation))
	}

	if apiErr.Validation["name"] != "Name is required" {
		t.Errorf("Expected validation error 'Name is required', got '%s'", apiErr.Validation["name"])
	}
}

func TestFromAWSError(t *testing.T) {
	err := errors.New("AWS API error")
	apiErr := FromAWSError(err, "DescribeInstances", "Instance")

	if apiErr.Code != types.ErrAWSError {
		t.Errorf("Expected code %s, got %s", types.ErrAWSError, apiErr.Code)
	}

	if apiErr.Operation != "DescribeInstances" {
		t.Errorf("Expected operation 'DescribeInstances', got '%s'", apiErr.Operation)
	}

	if apiErr.Resource != "Instance" {
		t.Errorf("Expected resource 'Instance', got '%s'", apiErr.Resource)
	}

	if errors.Unwrap(apiErr) != err {
		t.Error("Unwrap did not return original error")
	}
}
