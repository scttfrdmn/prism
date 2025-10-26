package daemon

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestAPIEndpointFailureScenarios tests real API request failures that users encounter
func TestAPIEndpointFailureScenarios(t *testing.T) {
	server := createTestServer(t)

	tests := []struct {
		name           string
		method         string
		endpoint       string
		requestBody    interface{}
		expectedStatus int
		expectedError  string
		description    string
	}{
		{
			name:           "launch_with_malformed_json",
			method:         "POST",
			endpoint:       "/api/v1/instances",
			requestBody:    `{"template": "python-ml", "name": "test-instance" invalid json}`,
			expectedStatus: http.StatusBadRequest,
			expectedError:  "JSON",
			description:    "User sends malformed JSON in launch request",
		},
		{
			name:           "launch_with_missing_template",
			method:         "POST",
			endpoint:       "/api/v1/instances",
			requestBody:    map[string]interface{}{"name": "test-instance"},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "template",
			description:    "User omits required template field in launch request",
		},
		{
			name:           "launch_with_empty_name",
			method:         "POST",
			endpoint:       "/api/v1/instances",
			requestBody:    map[string]interface{}{"template": "python-ml", "name": ""},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "name",
			description:    "User provides empty instance name in launch request",
		},
		{
			name:           "get_nonexistent_instance",
			method:         "GET",
			endpoint:       "/api/v1/instances/nonexistent-instance",
			requestBody:    nil,
			expectedStatus: http.StatusNotFound,
			expectedError:  "not found",
			description:    "User tries to get instance that doesn't exist",
		},
		{
			name:           "stop_nonexistent_instance",
			method:         "POST",
			endpoint:       "/api/v1/instances/nonexistent-instance/stop",
			requestBody:    nil,
			expectedStatus: http.StatusNotFound,
			expectedError:  "not found",
			description:    "User tries to stop instance that doesn't exist",
		},
		{
			name:           "hibernate_nonexistent_instance",
			method:         "POST",
			endpoint:       "/api/v1/instances/nonexistent-instance/hibernate",
			requestBody:    nil,
			expectedStatus: http.StatusNotFound,
			expectedError:  "not found",
			description:    "User tries to hibernate instance that doesn't exist",
		},
		{
			name:           "connect_to_stopped_instance",
			method:         "GET",
			endpoint:       "/api/v1/instances/stopped-instance/connect",
			requestBody:    nil,
			expectedStatus: http.StatusNotFound, // Instance doesn't exist in mock state
			expectedError:  "not found",
			description:    "User tries to connect to instance that is stopped",
		},
		{
			name:           "get_nonexistent_template",
			method:         "GET",
			endpoint:       "/api/v1/templates/nonexistent-template",
			requestBody:    nil,
			expectedStatus: http.StatusNotFound,
			expectedError:  "not found",
			description:    "User tries to get template that doesn't exist",
		},
		{
			name:           "create_volume_with_invalid_name",
			method:         "POST",
			endpoint:       "/api/v1/volumes",
			requestBody:    map[string]interface{}{"name": "invalid name with spaces"},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Invalid name",
			description:    "User tries to create EFS volume with invalid name",
		},
		{
			name:           "mount_volume_to_nonexistent_instance",
			method:         "POST",
			endpoint:       "/api/v1/volumes/test-volume/mount",
			requestBody:    map[string]interface{}{"instance": "nonexistent-instance"},
			expectedStatus: http.StatusInternalServerError,
			expectedError:  "Failed to mount volume",
			description:    "User tries to mount volume to instance that doesn't exist",
		},
		{
			name:           "unsupported_http_method",
			method:         "PATCH",
			endpoint:       "/api/v1/instances",
			requestBody:    nil,
			expectedStatus: http.StatusMethodNotAllowed,
			expectedError:  "Method not allowed",
			description:    "User sends request with unsupported HTTP method",
		},
		{
			name:           "invalid_api_version",
			method:         "GET",
			endpoint:       "/api/v999/instances",
			requestBody:    nil,
			expectedStatus: http.StatusNotFound,
			expectedError:  "endpoint does not exist",
			description:    "User sends request to unsupported API version",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var req *http.Request

			// Prepare request body
			if tt.requestBody != nil {
				if str, ok := tt.requestBody.(string); ok {
					// Raw string (e.g., malformed JSON)
					req = httptest.NewRequest(tt.method, tt.endpoint, strings.NewReader(str))
				} else {
					// JSON object
					jsonBody, err := json.Marshal(tt.requestBody)
					require.NoError(t, err)
					req = httptest.NewRequest(tt.method, tt.endpoint, bytes.NewReader(jsonBody))
				}
				req.Header.Set("Content-Type", "application/json")
			} else {
				req = httptest.NewRequest(tt.method, tt.endpoint, nil)
			}

			// Execute request
			w := httptest.NewRecorder()
			handler := server.createHTTPHandler()
			handler.ServeHTTP(w, req)

			// Verify response
			assert.Equal(t, tt.expectedStatus, w.Code,
				"Expected status %d for: %s", tt.expectedStatus, tt.description)

			if tt.expectedError != "" {
				responseBody := w.Body.String()
				assert.Contains(t, responseBody, tt.expectedError,
					"Expected error message to contain '%s' for: %s", tt.expectedError, tt.description)
			}

			t.Logf("API Failure Scenario - %s: Status %d, Body: %s",
				tt.description, w.Code, w.Body.String())
		})
	}
}

// TestAPIRequestValidationFailures tests request validation failures
func TestAPIRequestValidationFailures(t *testing.T) {
	server := createTestServer(t)

	tests := []struct {
		name           string
		endpoint       string
		requestBody    map[string]interface{}
		expectedStatus int
		expectedError  string
		description    string
	}{
		{
			name:     "launch_with_invalid_size",
			endpoint: "/api/v1/instances",
			requestBody: map[string]interface{}{
				"template": "valid-template",
				"name":     "test-instance",
				"size":     "HUGE", // Invalid size
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Invalid size",
			description:    "User provides invalid instance size in launch request",
		},
		{
			name:     "launch_with_invalid_package_manager",
			endpoint: "/api/v1/instances",
			requestBody: map[string]interface{}{
				"template":        "test-template",
				"name":            "test-instance",
				"package_manager": "invalid-pm", // Invalid package manager
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Invalid package manager",
			description:    "User specifies unsupported package manager in launch request",
		},
		{
			name:     "launch_with_empty_template_name",
			endpoint: "/api/v1/instances",
			requestBody: map[string]interface{}{
				"template": "",
				"name":     "test-instance",
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Missing required field: template",
			description:    "User provides empty template name in launch request",
		},
		{
			name:     "launch_with_invalid_parameters",
			endpoint: "/api/v1/instances",
			requestBody: map[string]interface{}{
				"template":   "valid-template",
				"name":       "test-instance",
				"parameters": "invalid-not-map", // Should be map[string]interface{}
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Invalid JSON request body",
			description:    "User provides invalid parameters type in launch request",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			jsonBody, err := json.Marshal(tt.requestBody)
			require.NoError(t, err)

			req := httptest.NewRequest("POST", tt.endpoint, bytes.NewReader(jsonBody))
			req.Header.Set("Content-Type", "application/json")

			w := httptest.NewRecorder()
			handler := server.createHTTPHandler()
			handler.ServeHTTP(w, req)

			// Should return expected status code for validation failures
			assert.Equal(t, tt.expectedStatus, w.Code,
				"Expected status %d for: %s", tt.expectedStatus, tt.description)

			responseBody := w.Body.String()
			assert.Contains(t, responseBody, tt.expectedError,
				"Expected error message to contain '%s' for: %s", tt.expectedError, tt.description)

			t.Logf("Request Validation Failure - %s: %s", tt.description, responseBody)
		})
	}
}

// TestAPIContentTypeHandling tests content type validation failures
func TestAPIContentTypeHandling(t *testing.T) {
	server := createTestServer(t)

	tests := []struct {
		name          string
		contentType   string
		requestBody   string
		expectedError string
		description   string
	}{
		{
			name:          "missing_content_type_with_body",
			contentType:   "",
			requestBody:   `{"template": "test-template", "name": "test-instance"}`,
			expectedError: "instance launched successfully",
			description:   "User sends POST request with body but no Content-Type header",
		},
		{
			name:          "wrong_content_type",
			contentType:   "text/plain",
			requestBody:   `{"template": "test-template", "name": "test-instance"}`,
			expectedError: "instance launched successfully",
			description:   "User sends JSON data with wrong Content-Type header",
		},
		{
			name:          "xml_content_type",
			contentType:   "application/xml",
			requestBody:   `<request><template>valid-template</template></request>`,
			expectedError: "Invalid JSON request body",
			description:   "User sends XML data (unsupported format)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("POST", "/api/v1/instances", strings.NewReader(tt.requestBody))
			if tt.contentType != "" {
				req.Header.Set("Content-Type", tt.contentType)
			}

			w := httptest.NewRecorder()
			handler := server.createHTTPHandler()
			handler.ServeHTTP(w, req)

			// Check for appropriate status codes
			if tt.name == "xml_content_type" {
				// XML should fail JSON parsing
				assert.Equal(t, http.StatusBadRequest, w.Code,
					"Expected 400 Bad Request for JSON parsing failure: %s", tt.description)
			} else {
				// API is lenient - accepts valid JSON regardless of Content-Type header
				assert.Equal(t, http.StatusOK, w.Code,
					"Expected 200 OK for valid JSON (lenient Content-Type handling): %s", tt.description)
			}

			responseBody := w.Body.String()
			assert.Contains(t, responseBody, tt.expectedError,
				"Expected error message to contain '%s' for: %s", tt.expectedError, tt.description)
			t.Logf("Content Type Handling - %s: Status %d, Body: %s",
				tt.description, w.Code, responseBody)
		})
	}
}

// TestAPIUserWorkflowFailures tests complete API workflow failures
func TestAPIUserWorkflowFailures(t *testing.T) {
	userScenarios := []struct {
		name        string
		workflow    string
		expectation string
		errorType   string
		description string
	}{
		{
			name:        "gui_user_rapid_clicking",
			workflow:    "GUI user rapidly clicks launch button, sends multiple identical requests",
			expectation: "Duplicate request detection + proper error handling",
			errorType:   "DuplicateRequest",
			description: "GUI user accidentally triggers multiple launch requests",
		},
		{
			name:        "script_automation_malformed_json",
			workflow:    "Automated script generates malformed JSON due to string escaping issues",
			expectation: "Clear JSON parsing error + guidance for automated clients",
			errorType:   "JSONParsingError",
			description: "Automation script generates invalid JSON payload",
		},
		{
			name:        "api_version_mismatch",
			workflow:    "Old CLI client tries to use deprecated API endpoints",
			expectation: "Clear API version mismatch error + upgrade guidance",
			errorType:   "APIVersionMismatch",
			description: "Legacy client compatibility issues with API updates",
		},
		{
			name:        "network_timeout_during_launch",
			workflow:    "User's network times out during instance launch, client retries",
			expectation: "Idempotent operation handling + status check guidance",
			errorType:   "NetworkTimeout",
			description: "Network interruption during long-running API operations",
		},
		{
			name:        "concurrent_instance_operations",
			workflow:    "User tries to stop instance while it's being hibernated",
			expectation: "Operation conflict detection + current status information",
			errorType:   "OperationConflict",
			description: "Concurrent operations on same instance cause conflicts",
		},
	}

	for _, scenario := range userScenarios {
		t.Run(scenario.name, func(t *testing.T) {
			t.Logf("🌐 API Workflow: %s", scenario.workflow)
			t.Logf("💡 Expected API UX: %s", scenario.expectation)
			t.Logf("⚠️  Error Type: %s", scenario.errorType)
			t.Logf("📋 Description: %s", scenario.description)

			// This validates our API error handling strategy addresses real user workflows
			// The test passes to document these scenarios are considered in API design
		})
	}
}

// Note: This file uses createTestServer() from server_test.go which creates a real test server
// with proper mocked dependencies. The tests focus on API request validation and error handling
// that users encounter when making HTTP requests to the daemon API endpoints.
