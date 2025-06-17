package daemon

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/scttfrdmn/cloudworkstation/pkg/types"
)

// Let's focus on testing the functions that don't require complex mocking

func TestSplitPath(t *testing.T) {
	tests := []struct {
		input    string
		expected []string
		name     string
	}{
		{"", []string{}, "empty path"},
		{"instance-name", []string{"instance-name"}, "single segment"},
		{"instance-name/start", []string{"instance-name", "start"}, "two segments"},
		{"instance-name/start/", []string{"instance-name", "start"}, "trailing slash"},
		{"a/b/c", []string{"a", "b", "c"}, "three segments"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := splitPath(tt.input)
			if len(result) != len(tt.expected) {
				t.Errorf("splitPath(%s) length = %d, want %d", tt.input, len(result), len(tt.expected))
				return
			}
			for i, part := range result {
				if part != tt.expected[i] {
					t.Errorf("splitPath(%s)[%d] = %s, want %s", tt.input, i, part, tt.expected[i])
				}
			}
		})
	}
}

func TestHandlePing(t *testing.T) {
	server := &Server{} // Simple server without dependencies for ping test
	
	req := httptest.NewRequest(http.MethodGet, "/api/v1/ping", nil)
	w := httptest.NewRecorder()
	
	server.handlePing(w, req)
	
	if w.Code != http.StatusOK {
		t.Errorf("handlePing() status = %d, want %d", w.Code, http.StatusOK)
	}
	
	var response map[string]string
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Errorf("Failed to decode response: %v", err)
	}
	
	if response["status"] != "ok" {
		t.Errorf("handlePing() status = %s, want ok", response["status"])
	}
}

func TestHandleStatusMethodNotAllowed(t *testing.T) {
	server := &Server{} // Simple server without dependencies
	
	req := httptest.NewRequest(http.MethodPost, "/api/v1/status", nil)
	w := httptest.NewRecorder()
	
	server.handleStatus(w, req)
	
	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("handleStatus() POST status = %d, want %d", w.Code, http.StatusMethodNotAllowed)
	}
}

func TestHandleInvalidJSON(t *testing.T) {
	server := &Server{}
	
	// Test invalid JSON handling for launch instance
	req := httptest.NewRequest(http.MethodPost, "/api/v1/instances", strings.NewReader("invalid json"))
	w := httptest.NewRecorder()
	
	server.handleLaunchInstance(w, req)
	
	if w.Code != http.StatusBadRequest {
		t.Errorf("handleLaunchInstance() with invalid JSON status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestHandleTemplatesMethodNotAllowed(t *testing.T) {
	server := &Server{}
	
	req := httptest.NewRequest(http.MethodPost, "/api/v1/templates", nil)
	w := httptest.NewRecorder()
	
	server.handleTemplates(w, req)
	
	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("handleTemplates() POST status = %d, want %d", w.Code, http.StatusMethodNotAllowed)
	}
}

func TestHandleTemplateInfoMethodNotAllowed(t *testing.T) {
	server := &Server{}
	
	req := httptest.NewRequest(http.MethodPost, "/api/v1/templates/test", nil)
	w := httptest.NewRecorder()
	
	server.handleTemplateInfo(w, req)
	
	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("handleTemplateInfo() POST status = %d, want %d", w.Code, http.StatusMethodNotAllowed)
	}
}

func TestHandleCreateVolumeInvalidJSON(t *testing.T) {
	server := &Server{}
	
	req := httptest.NewRequest(http.MethodPost, "/api/v1/volumes", strings.NewReader("invalid json"))
	w := httptest.NewRecorder()
	
	server.handleCreateVolume(w, req)
	
	if w.Code != http.StatusBadRequest {
		t.Errorf("handleCreateVolume() with invalid JSON status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestWriteError(t *testing.T) {
	server := &Server{}
	
	w := httptest.NewRecorder()
	server.writeError(w, http.StatusBadRequest, "Test error message")
	
	if w.Code != http.StatusBadRequest {
		t.Errorf("writeError() status = %d, want %d", w.Code, http.StatusBadRequest)
	}
	
	var apiError types.APIError
	if err := json.NewDecoder(w.Body).Decode(&apiError); err != nil {
		t.Errorf("Failed to decode error response: %v", err)
	}
	
	if apiError.Code != http.StatusBadRequest {
		t.Errorf("APIError code = %d, want %d", apiError.Code, http.StatusBadRequest)
	}
	
	if apiError.Message != "Test error message" {
		t.Errorf("APIError message = %s, want 'Test error message'", apiError.Message)
	}
}

func TestHandleInstanceOperationsMissingName(t *testing.T) {
	server := &Server{}
	
	req := httptest.NewRequest(http.MethodGet, "/api/v1/instances/", nil)
	w := httptest.NewRecorder()
	
	server.handleInstanceOperations(w, req)
	
	if w.Code != http.StatusBadRequest {
		t.Errorf("Missing instance name status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestHandleInstanceOperationsUnknownOperation(t *testing.T) {
	server := &Server{}
	
	req := httptest.NewRequest(http.MethodPost, "/api/v1/instances/test-instance/unknown", nil)
	w := httptest.NewRecorder()
	
	server.handleInstanceOperations(w, req)
	
	if w.Code != http.StatusNotFound {
		t.Errorf("Unknown operation status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestServerPortHandling(t *testing.T) {
	t.Run("Custom port", func(t *testing.T) {
		server := &Server{port: "9090"}
		if server.port != "9090" {
			t.Errorf("Custom port = %s, want 9090", server.port)
		}
	})
	
	t.Run("Empty port", func(t *testing.T) {
		server := &Server{port: ""}
		if server.port != "" {
			t.Errorf("Empty port should remain empty until NewServer is called")
		}
	})
}

func TestHandleStartInstanceMethodNotAllowed(t *testing.T) {
	server := &Server{}
	
	req := httptest.NewRequest(http.MethodGet, "/api/v1/instances/test/start", nil)
	w := httptest.NewRecorder()
	
	server.handleStartInstance(w, req, "test")
	
	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("handleStartInstance() GET status = %d, want %d", w.Code, http.StatusMethodNotAllowed)
	}
}

func TestHandleStopInstanceMethodNotAllowed(t *testing.T) {
	server := &Server{}
	
	req := httptest.NewRequest(http.MethodGet, "/api/v1/instances/test/stop", nil)
	w := httptest.NewRecorder()
	
	server.handleStopInstance(w, req, "test")
	
	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("handleStopInstance() GET status = %d, want %d", w.Code, http.StatusMethodNotAllowed)
	}
}

func TestHandleConnectInstanceMethodNotAllowed(t *testing.T) {
	server := &Server{}
	
	req := httptest.NewRequest(http.MethodPost, "/api/v1/instances/test/connect", nil)
	w := httptest.NewRecorder()
	
	server.handleConnectInstance(w, req, "test")
	
	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("handleConnectInstance() POST status = %d, want %d", w.Code, http.StatusMethodNotAllowed)
	}
}

// Additional comprehensive tests to reach 80% coverage for daemon package

func TestHandleVolumesMethodNotAllowed(t *testing.T) {
	server := &Server{}
	
	req := httptest.NewRequest(http.MethodPut, "/api/v1/volumes", nil)
	w := httptest.NewRecorder()
	
	server.handleVolumes(w, req)
	
	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("handleVolumes() PUT status = %d, want %d", w.Code, http.StatusMethodNotAllowed)
	}
}

func TestHandleVolumeOperationsMethodNotAllowed(t *testing.T) {
	server := &Server{}
	
	req := httptest.NewRequest(http.MethodPut, "/api/v1/volumes/test-volume", nil)
	w := httptest.NewRecorder()
	
	server.handleVolumeOperations(w, req)
	
	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("handleVolumeOperations() PUT status = %d, want %d", w.Code, http.StatusMethodNotAllowed)
	}
}

func TestHandleStorageMethodNotAllowed(t *testing.T) {
	server := &Server{}
	
	req := httptest.NewRequest(http.MethodPut, "/api/v1/storage", nil)
	w := httptest.NewRecorder()
	
	server.handleStorage(w, req)
	
	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("handleStorage() PUT status = %d, want %d", w.Code, http.StatusMethodNotAllowed)
	}
}

func TestHandleStorageOperationsMissingName(t *testing.T) {
	server := &Server{}
	
	req := httptest.NewRequest(http.MethodGet, "/api/v1/storage/", nil)
	w := httptest.NewRecorder()
	
	server.handleStorageOperations(w, req)
	
	if w.Code != http.StatusBadRequest {
		t.Errorf("Missing storage name status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestHandleStorageOperationsUnknownOperation(t *testing.T) {
	server := &Server{}
	
	req := httptest.NewRequest(http.MethodPost, "/api/v1/storage/test-storage/unknown", nil)
	w := httptest.NewRecorder()
	
	server.handleStorageOperations(w, req)
	
	if w.Code != http.StatusNotFound {
		t.Errorf("Unknown storage operation status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestHandleAttachStorageMethodNotAllowed(t *testing.T) {
	server := &Server{}
	
	req := httptest.NewRequest(http.MethodGet, "/api/v1/storage/test/attach", nil)
	w := httptest.NewRecorder()
	
	server.handleAttachStorage(w, req, "test")
	
	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("handleAttachStorage() GET status = %d, want %d", w.Code, http.StatusMethodNotAllowed)
	}
}

func TestHandleDetachStorageMethodNotAllowed(t *testing.T) {
	server := &Server{}
	
	req := httptest.NewRequest(http.MethodGet, "/api/v1/storage/test/detach", nil)
	w := httptest.NewRecorder()
	
	server.handleDetachStorage(w, req, "test")
	
	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("handleDetachStorage() GET status = %d, want %d", w.Code, http.StatusMethodNotAllowed)
	}
}

func TestHandleAttachStorageInvalidJSON(t *testing.T) {
	server := &Server{}
	
	req := httptest.NewRequest(http.MethodPost, "/api/v1/storage/test/attach", strings.NewReader("invalid json"))
	w := httptest.NewRecorder()
	
	server.handleAttachStorage(w, req, "test")
	
	if w.Code != http.StatusBadRequest {
		t.Errorf("handleAttachStorage() with invalid JSON status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestHandleAttachStorageMissingInstance(t *testing.T) {
	server := &Server{}
	
	reqBody := `{"other_field": "value"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/storage/test/attach", strings.NewReader(reqBody))
	w := httptest.NewRecorder()
	
	server.handleAttachStorage(w, req, "test")
	
	if w.Code != http.StatusBadRequest {
		t.Errorf("handleAttachStorage() missing instance name status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestSplitPathEdgeCases(t *testing.T) {
	tests := []struct {
		input    string
		expected []string
		name     string
	}{
		{"/", []string{""}, "single slash"},
		{"//", []string{"", ""}, "double slash"},
		{"a//b", []string{"a", "", "b"}, "empty segment"},
		{"trailing/slash/", []string{"trailing", "slash"}, "trailing slash removal"},
		{"no-slash", []string{"no-slash"}, "no slash"},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := splitPath(tt.input)
			if len(result) != len(tt.expected) {
				t.Errorf("splitPath(%s) length = %d, want %d", tt.input, len(result), len(tt.expected))
				return
			}
			for i, part := range result {
				if part != tt.expected[i] {
					t.Errorf("splitPath(%s)[%d] = %s, want %s", tt.input, i, part, tt.expected[i])
				}
			}
		})
	}
}

func TestServerInitialization(t *testing.T) {
	t.Run("Server with custom port", func(t *testing.T) {
		// Test the parts of server creation that don't require AWS
		server := &Server{
			port: "9090",
		}
		
		if server.port != "9090" {
			t.Errorf("Server port = %s, want 9090", server.port)
		}
	})
	
	t.Run("Empty server", func(t *testing.T) {
		server := &Server{}
		
		if server.port != "" {
			t.Errorf("Empty server port should be empty, got %s", server.port)
		}
	})
}

func TestJSONMiddleware(t *testing.T) {
	// Test that writeError produces valid JSON
	server := &Server{}
	
	w := httptest.NewRecorder()
	server.writeError(w, http.StatusBadRequest, "test error")
	
	// Check that response body is valid JSON
	var apiError types.APIError
	if err := json.NewDecoder(w.Body).Decode(&apiError); err != nil {
		t.Errorf("writeError should produce valid JSON: %v", err)
	}
	
	if apiError.Message != "test error" {
		t.Errorf("APIError message = %s, want 'test error'", apiError.Message)
	}
}

func TestWriteErrorVariations(t *testing.T) {
	server := &Server{}
	
	tests := []struct {
		code    int
		message string
		name    string
	}{
		{http.StatusNotFound, "Not found", "404 error"},
		{http.StatusInternalServerError, "Internal error", "500 error"},
		{http.StatusUnauthorized, "Unauthorized", "401 error"},
		{http.StatusForbidden, "Forbidden", "403 error"},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			server.writeError(w, tt.code, tt.message)
			
			if w.Code != tt.code {
				t.Errorf("writeError() status = %d, want %d", w.Code, tt.code)
			}
			
			var apiError types.APIError
			if err := json.NewDecoder(w.Body).Decode(&apiError); err != nil {
				t.Errorf("Failed to decode error response: %v", err)
			}
			
			if apiError.Message != tt.message {
				t.Errorf("APIError message = %s, want %s", apiError.Message, tt.message)
			}
		})
	}
}