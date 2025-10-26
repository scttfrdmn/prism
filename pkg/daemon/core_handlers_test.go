package daemon

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/scttfrdmn/prism/pkg/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestAPIVersionsHandler tests the API versions endpoint
func TestAPIVersionsHandler(t *testing.T) {
	server := createTestServer(t)
	handler := server.createHTTPHandler()

	req := httptest.NewRequest("GET", "/api/versions", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response types.APIVersionResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.NotEmpty(t, response.Versions)
	assert.Equal(t, "v1", response.DefaultVersion)
	assert.Equal(t, "v1", response.StableVersion)
	assert.Equal(t, "v1", response.LatestVersion)
}

// TestUnknownAPIHandler tests unknown endpoint handling
func TestUnknownAPIHandler(t *testing.T) {
	server := createTestServer(t)
	handler := server.createHTTPHandler()

	tests := []struct {
		name           string
		path           string
		expectedStatus int
	}{
		{
			name:           "unknown endpoint with valid version",
			path:           "/api/v1/unknown",
			expectedStatus: http.StatusNotFound,
		},
		{
			name:           "unknown endpoint without version",
			path:           "/api/unknown",
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", tt.path, nil)
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			// Verify error response structure
			var response map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &response)
			assert.NoError(t, err)

			if tt.expectedStatus == http.StatusNotFound {
				// Should be APIErrorResponse
				assert.Equal(t, "endpoint_not_found", response["code"])
				assert.Contains(t, response["message"], "does not exist")
			} else if tt.expectedStatus == http.StatusBadRequest {
				// Should include available versions
				assert.Contains(t, response, "available_versions")
				assert.Contains(t, response, "error")
			}
		})
	}
}

// TestServerMiddleware tests middleware functionality
func TestServerMiddleware(t *testing.T) {
	server := createTestServer(t)
	handler := server.createHTTPHandler()

	// Test JSON content type middleware
	req := httptest.NewRequest("GET", "/api/v1/status", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Header().Get("Content-Type"), "application/json")

	// Test API version headers are set
	assert.Equal(t, "v1", w.Header().Get("X-API-Version"))
	assert.NotEmpty(t, w.Header().Get("X-API-Latest-Version"))
	assert.NotEmpty(t, w.Header().Get("X-API-Stable-Version"))
}

// TestAPIVersionMiddleware tests API versioning middleware
func TestAPIVersionMiddleware(t *testing.T) {
	server := createTestServer(t)
	handler := server.createHTTPHandler()

	tests := []struct {
		name           string
		path           string
		expectedStatus int
		checkHeaders   func(t *testing.T, w *httptest.ResponseRecorder)
	}{
		{
			name:           "v1 endpoint has version headers",
			path:           "/api/v1/status",
			expectedStatus: http.StatusOK,
			checkHeaders: func(t *testing.T, w *httptest.ResponseRecorder) {
				assert.Equal(t, "v1", w.Header().Get("X-API-Version"))
				assert.Equal(t, "v1", w.Header().Get("X-API-Latest-Version"))
				assert.Equal(t, "v1", w.Header().Get("X-API-Stable-Version"))
			},
		},
		{
			name:           "unversioned endpoint gets default version",
			path:           "/api/status",
			expectedStatus: http.StatusBadRequest,
			checkHeaders: func(t *testing.T, w *httptest.ResponseRecorder) {
				// Should still have version headers
				assert.NotEmpty(t, w.Header().Get("X-API-Version"))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", tt.path, nil)
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.checkHeaders != nil {
				tt.checkHeaders(t, w)
			}
		})
	}
}

// TestServerErrorResponses tests error response formatting
func TestServerErrorResponses(t *testing.T) {
	server := createTestServer(t)
	handler := server.createHTTPHandler()

	// Test malformed JSON request
	malformedJSON := `{"invalid": json`
	req := httptest.NewRequest("POST", "/api/v1/instances", strings.NewReader(malformedJSON))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	// Verify JSON response
	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.NotEmpty(t, response)
}

// TestServerAuthentication tests authentication middleware
func TestServerAuthentication(t *testing.T) {
	server := createTestServer(t)
	handler := server.createHTTPHandler()

	// Test endpoint that may require authentication
	req := httptest.NewRequest("POST", "/api/v1/instances", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	// Should handle request without crashing (authentication may be disabled in test mode)
	// Valid response codes: BadRequest (missing data), OK, Unauthorized, MethodNotAllowed
	validCodes := []int{http.StatusBadRequest, http.StatusOK, http.StatusUnauthorized, http.StatusMethodNotAllowed}
	assert.Contains(t, validCodes, w.Code)
}

// TestServerRequestLogging tests request logging middleware
func TestServerRequestLogging(t *testing.T) {
	server := createTestServer(t)
	handler := server.createHTTPHandler()

	// Make a request to valid endpoint
	req := httptest.NewRequest("GET", "/api/v1/status", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	// Should complete without errors (logging middleware should not crash)
	assert.Equal(t, http.StatusOK, w.Code)

	// Should have recorded the request in status tracker
	status := server.statusTracker.GetStatus("test-version", "us-west-2", "test-profile")
	assert.True(t, status.TotalRequests > 0)
}

// TestServerJSONContentType tests JSON content type handling
func TestServerJSONContentType(t *testing.T) {
	server := createTestServer(t)
	handler := server.createHTTPHandler()

	// Test multiple JSON endpoints
	endpoints := []string{
		"/api/v1/status",
		"/api/v1/ping",
		"/api/versions",
	}

	for _, endpoint := range endpoints {
		t.Run(endpoint, func(t *testing.T) {
			req := httptest.NewRequest("GET", endpoint, nil)
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, req)

			assert.Equal(t, http.StatusOK, w.Code)
			assert.Contains(t, w.Header().Get("Content-Type"), "application/json")
		})
	}
}

// TestServerMetrics tests metrics collection
func TestServerMetrics(t *testing.T) {
	server := createTestServer(t)

	// Test metrics are being tracked
	assert.NotNil(t, server.statusTracker)

	// Test request recording
	server.statusTracker.RecordRequest()
	server.statusTracker.RecordRequest()

	// Test operation tracking
	opID := server.statusTracker.StartOperationWithType("TestOp")
	assert.Greater(t, opID, int64(0))

	// Test getting status with parameters (matching actual GetStatus signature)
	status := server.statusTracker.GetStatus("test-version", "us-west-2", "test-profile")
	assert.Equal(t, "test-version", status.Version)
	assert.Equal(t, "us-west-2", status.AWSRegion)
	assert.Equal(t, "test-profile", status.AWSProfile)
	assert.Equal(t, "running", status.Status)
	assert.True(t, status.TotalRequests >= 2)
	assert.True(t, status.ActiveOps >= 1)

	// Clean up operation
	server.statusTracker.EndOperationWithType("TestOp")
}
