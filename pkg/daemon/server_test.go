package daemon

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/scttfrdmn/prism/pkg/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test helper: creates a test server with mock dependencies
func createTestServer(t *testing.T) *Server {
	// Create temporary directory for testing
	tempDir := t.TempDir()

	// Override home directory for testing
	originalHome := os.Getenv("HOME")
	t.Cleanup(func() {
		_ = os.Setenv("HOME", originalHome)
	})
	_ = os.Setenv("HOME", tempDir)

	server, err := NewServerForTesting("8948") // Use test mode to skip AWS operations
	require.NoError(t, err)
	require.NotNil(t, server)

	return server
}

// TestNewServer tests server creation
func TestNewServer(t *testing.T) {
	server := createTestServer(t)

	assert.NotNil(t, server.config)
	assert.Equal(t, "8948", server.port)
	assert.NotNil(t, server.stateManager)
	assert.NotNil(t, server.userManager)
	assert.NotNil(t, server.statusTracker)
	assert.NotNil(t, server.versionManager)
}

// TestServerHTTPHandler tests HTTP request routing
func TestServerHTTPHandler(t *testing.T) {
	server := createTestServer(t)

	tests := []struct {
		name           string
		method         string
		path           string
		expectedStatus int
		checkResponse  func(t *testing.T, body []byte)
	}{
		{
			name:           "ping endpoint",
			method:         "GET",
			path:           "/api/v1/ping",
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, body []byte) {
				assert.Contains(t, string(body), "ok")
			},
		},
		{
			name:           "version endpoint",
			method:         "GET",
			path:           "/api/versions",
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, body []byte) {
				var response map[string]interface{}
				err := json.Unmarshal(body, &response)
				assert.NoError(t, err)
				assert.Contains(t, response, "versions")
			},
		},
		{
			name:           "list instances",
			method:         "GET",
			path:           "/api/v1/instances",
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, body []byte) {
				var response types.ListResponse
				err := json.Unmarshal(body, &response)
				assert.NoError(t, err)
				assert.NotNil(t, response.Instances)
			},
		},
		{
			name:           "invalid endpoint",
			method:         "GET",
			path:           "/invalid/endpoint",
			expectedStatus: http.StatusNotFound,
			checkResponse: func(t *testing.T, body []byte) {
				assert.Contains(t, string(body), "404")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create HTTP request
			req := httptest.NewRequest(tt.method, tt.path, nil)
			w := httptest.NewRecorder()

			// Create HTTP handler
			handler := server.createHTTPHandler()

			// Execute request
			handler.ServeHTTP(w, req)

			// Check status code
			assert.Equal(t, tt.expectedStatus, w.Code)

			// Check response body if specified
			if tt.checkResponse != nil {
				tt.checkResponse(t, w.Body.Bytes())
			}
		})
	}
}

// TestServerStartStop tests server lifecycle
func TestServerStartStop(t *testing.T) {
	server := createTestServer(t)

	// Start server in background
	errChan := make(chan error, 1)
	go func() {
		errChan <- server.Start()
	}()

	// Give server time to start
	time.Sleep(100 * time.Millisecond)

	// Test that server is responding
	resp, err := http.Get(fmt.Sprintf("http://localhost:%s/api/v1/ping", server.port))
	if err == nil {
		_ = resp.Body.Close()
		assert.Equal(t, http.StatusOK, resp.StatusCode)
	}

	// Stop server
	stopErr := server.Stop()
	assert.NoError(t, stopErr)

	// Wait for server to stop
	select {
	case err := <-errChan:
		// Server should stop cleanly (accept normal shutdown errors)
		assert.True(t, err == nil ||
			strings.Contains(err.Error(), "context canceled") ||
			strings.Contains(err.Error(), "http: Server closed"),
			"Expected nil, 'context canceled', or 'http: Server closed', got: %v", err)
	case <-time.After(5 * time.Second):
		t.Fatal("Server did not stop within timeout")
	}
}

// TestServerErrorHandling tests error response formatting
func TestServerErrorHandling(t *testing.T) {
	server := createTestServer(t)
	handler := server.createHTTPHandler()

	// Test invalid JSON in request body
	invalidJSON := `{"invalid": json}`
	req := httptest.NewRequest("POST", "/api/v1/instances", strings.NewReader(invalidJSON))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	// Should return error response
	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response types.APIError
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.NotEmpty(t, response.Message)
	assert.NotEmpty(t, response.Code)
}

// TestServerHeaders tests middleware header functionality
func TestServerHeaders(t *testing.T) {
	server := createTestServer(t)
	handler := server.createHTTPHandler()

	// Test JSON headers are set
	req := httptest.NewRequest("GET", "/api/v1/status", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Header().Get("Content-Type"), "application/json")
	assert.Equal(t, "v1", w.Header().Get("X-API-Version"))
}

// TestServerAPIVersioning tests API version handling
func TestServerAPIVersioning(t *testing.T) {
	server := createTestServer(t)
	handler := server.createHTTPHandler()

	tests := []struct {
		name           string
		path           string
		expectedStatus int
	}{
		{
			name:           "v1 API endpoint",
			path:           "/api/v1/instances",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "unversioned endpoint gets version required error",
			path:           "/api/instances",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "future version returns not found",
			path:           "/api/v2/instances",
			expectedStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", tt.path, nil)
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, req)
			assert.Equal(t, tt.expectedStatus, w.Code)
		})
	}
}

// TestServerConcurrency tests concurrent request handling
func TestServerConcurrency(t *testing.T) {
	server := createTestServer(t)
	handler := server.createHTTPHandler()

	// Make multiple concurrent requests
	const numRequests = 10
	responses := make(chan int, numRequests)

	for i := 0; i < numRequests; i++ {
		go func() {
			req := httptest.NewRequest("GET", "/api/v1/ping", nil)
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, req)
			responses <- w.Code
		}()
	}

	// Collect all responses
	for i := 0; i < numRequests; i++ {
		select {
		case status := <-responses:
			assert.Equal(t, http.StatusOK, status)
		case <-time.After(5 * time.Second):
			t.Fatal("Request timeout")
		}
	}
}

// TestServerConfiguration tests server configuration loading
func TestServerConfiguration(t *testing.T) {
	// Save any existing config
	configPath := GetConfigPath()
	var existingConfig []byte
	if data, err := os.ReadFile(configPath); err == nil {
		existingConfig = data
		// Remove config temporarily
		_ = os.Remove(configPath)
	}
	// Restore config after test
	defer func() {
		if existingConfig != nil {
			_ = os.WriteFile(configPath, existingConfig, 0644)
		}
	}()

	// Test with explicit port parameter
	server, err := NewServer("9999")
	require.NoError(t, err)

	// Should use provided port parameter
	assert.Equal(t, "9999", server.port)

	// Test with default port (empty parameter, default config)
	server2, err := NewServer("")
	require.NoError(t, err)

	// Should use default port
	assert.Equal(t, "8947", server2.port)
}

// TestServerStatusTracking tests status tracking functionality
func TestServerStatusTracking(t *testing.T) {
	server := createTestServer(t)

	// Test status tracker is initialized
	assert.NotNil(t, server.statusTracker)

	// Test request recording
	server.statusTracker.RecordRequest()
	server.statusTracker.RecordRequest()

	// Test operation tracking
	opID := server.statusTracker.StartOperationWithType("TestOp")
	assert.Greater(t, opID, int64(0))

	// Verify status retrieval
	status := server.statusTracker.GetStatus("test-version", "us-west-2", "test-profile")
	assert.Equal(t, "test-version", status.Version)
	assert.True(t, status.TotalRequests >= 2)
	assert.True(t, status.ActiveOps >= 1)

	// Clean up
	server.statusTracker.EndOperationWithType("TestOp")
}

// TestServerGracefulShutdown tests graceful shutdown behavior
func TestServerGracefulShutdown(t *testing.T) {
	server := createTestServer(t)

	// Start server
	errChan := make(chan error, 1)
	go func() {
		errChan <- server.Start()
	}()

	// Give server time to start
	time.Sleep(100 * time.Millisecond)

	// Stop server gracefully
	_ = server.Stop()

	// Server should shut down gracefully within timeout
	select {
	case err := <-errChan:
		// Show the actual error for debugging
		if err != nil {
			t.Logf("Server shutdown with error: %v", err)
		}
		// Should either be nil, context canceled, or http server closed (all are normal shutdown scenarios)
		assert.True(t, err == nil ||
			strings.Contains(err.Error(), "context canceled") ||
			strings.Contains(err.Error(), "http: Server closed"),
			"Expected nil, 'context canceled', or 'http: Server closed', got: %v", err)
	case <-time.After(3 * time.Second):
		t.Fatal("Server did not shut down gracefully within timeout")
	}
}
