package daemon

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/scttfrdmn/prism/pkg/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestHandleListInstances tests the instance listing endpoint
func TestHandleListInstances(t *testing.T) {
	server := createTestServer(t)
	handler := server.createHTTPHandler()

	tests := []struct {
		name           string
		queryParams    string
		expectedStatus int
		checkResponse  func(t *testing.T, response types.ListResponse)
	}{
		{
			name:           "list instances from state (fast mode)",
			queryParams:    "",
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, response types.ListResponse) {
				assert.NotNil(t, response.Instances)
				assert.GreaterOrEqual(t, len(response.Instances), 0)
			},
		},
		{
			name:           "list instances with refresh parameter",
			queryParams:    "?refresh=true",
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, response types.ListResponse) {
				assert.NotNil(t, response.Instances)
			},
		},
		{
			name:           "list instances with invalid refresh parameter",
			queryParams:    "?refresh=invalid",
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, response types.ListResponse) {
				assert.NotNil(t, response.Instances)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/api/v1/instances"+tt.queryParams, nil)
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.checkResponse != nil && w.Code == http.StatusOK {
				var response types.ListResponse
				err := json.Unmarshal(w.Body.Bytes(), &response)
				require.NoError(t, err)
				tt.checkResponse(t, response)
			}
		})
	}
}

// TestHandleLaunchInstance tests the instance launch endpoint
func TestHandleLaunchInstance(t *testing.T) {
	server := createTestServer(t)
	handler := server.createHTTPHandler()

	tests := []struct {
		name           string
		launchRequest  interface{}
		expectedStatus int
		checkError     func(t *testing.T, response types.APIError)
	}{
		{
			name: "missing template name",
			launchRequest: map[string]interface{}{
				"instance_name": "test-instance",
			},
			expectedStatus: http.StatusBadRequest,
			checkError: func(t *testing.T, response types.APIError) {
				assert.NotEmpty(t, response.Message)
				assert.Contains(t, response.Message, "template")
			},
		},
		{
			name: "missing instance name",
			launchRequest: map[string]interface{}{
				"template": "test-template",
			},
			expectedStatus: http.StatusBadRequest,
			checkError: func(t *testing.T, response types.APIError) {
				assert.NotEmpty(t, response.Message)
			},
		},
		{
			name:           "invalid JSON",
			launchRequest:  "invalid-json",
			expectedStatus: http.StatusBadRequest,
			checkError: func(t *testing.T, response types.APIError) {
				assert.NotEmpty(t, response.Message)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var body []byte
			var err error

			if str, ok := tt.launchRequest.(string); ok {
				body = []byte(str)
			} else {
				body, err = json.Marshal(tt.launchRequest)
				require.NoError(t, err)
			}

			req := httptest.NewRequest("POST", "/api/v1/instances", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.checkError != nil {
				var response types.APIError
				err := json.Unmarshal(w.Body.Bytes(), &response)
				require.NoError(t, err)
				tt.checkError(t, response)
			}
		})
	}
}

// TestHandleGetInstance tests the get instance endpoint
func TestHandleGetInstance(t *testing.T) {
	server := createTestServer(t)
	handler := server.createHTTPHandler()

	tests := []struct {
		name           string
		instanceName   string
		expectedStatus int
	}{
		{
			name:           "get non-existent instance",
			instanceName:   "non-existent-instance",
			expectedStatus: http.StatusNotFound,
		},
		{
			name:           "get instance with empty name",
			instanceName:   "",
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := fmt.Sprintf("/api/v1/instances/%s", tt.instanceName)
			req := httptest.NewRequest("GET", path, nil)
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
		})
	}
}

// TestHandleDeleteDryRunInstance tests that dry-run instances can be deleted
// without attempting AWS operations. Dry-run instances are created by
// "prism workspace launch --dry-run" and never exist on AWS.
func TestHandleDeleteDryRunInstance(t *testing.T) {
	server := createTestServer(t)
	handler := server.createHTTPHandler()

	// Seed a dry-run instance in state
	dryRunInstance := types.Instance{
		Name:  "test-dry-run",
		State: "dry-run",
		ID:    "",
	}
	err := server.stateManager.SaveInstance(dryRunInstance)
	require.NoError(t, err)

	// Verify it exists
	state, err := server.stateManager.LoadState()
	require.NoError(t, err)
	_, exists := state.Instances["test-dry-run"]
	require.True(t, exists, "dry-run instance should exist in state before delete")

	// Delete it
	req := httptest.NewRequest("DELETE", "/api/v1/instances/test-dry-run", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	assert.Equal(t, http.StatusNoContent, w.Code)

	// Verify it was removed from state
	state, err = server.stateManager.LoadState()
	require.NoError(t, err)
	_, exists = state.Instances["test-dry-run"]
	assert.False(t, exists, "dry-run instance should be removed from state after delete")
}

// TestHandleDeleteNonExistentInstance tests that deleting a non-existent instance returns 404
func TestHandleDeleteNonExistentInstance(t *testing.T) {
	server := createTestServer(t)
	handler := server.createHTTPHandler()

	req := httptest.NewRequest("DELETE", "/api/v1/instances/does-not-exist", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

// TestHandleInstanceActions tests instance control actions (start, stop, terminate)
func TestHandleInstanceActions(t *testing.T) {
	server := createTestServer(t)
	handler := server.createHTTPHandler()

	actions := []string{"start", "stop", "reboot", "terminate"}

	for _, action := range actions {
		t.Run(fmt.Sprintf("action_%s_non_existent_instance", action), func(t *testing.T) {
			path := fmt.Sprintf("/api/v1/instances/non-existent/%s", action)
			req := httptest.NewRequest("POST", path, nil)
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, req)

			// Should return 404 for non-existent instance
			assert.Equal(t, http.StatusNotFound, w.Code)
		})
	}
}

// TestHandleTemplates tests the template listing endpoint
func TestHandleTemplates(t *testing.T) {
	server := createTestServer(t)
	handler := server.createHTTPHandler()

	tests := []struct {
		name           string
		queryParams    string
		expectedStatus int
		checkResponse  func(t *testing.T, templates map[string]types.RuntimeTemplate)
	}{
		{
			name:           "list templates with default params",
			queryParams:    "",
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, templates map[string]types.RuntimeTemplate) {
				assert.NotNil(t, templates)
				// Should have at least the test template we created
				assert.GreaterOrEqual(t, len(templates), 1)
			},
		},
		{
			name:           "list templates with specific region",
			queryParams:    "?region=us-west-2",
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, templates map[string]types.RuntimeTemplate) {
				assert.NotNil(t, templates)
			},
		},
		{
			name:           "list templates with specific architecture",
			queryParams:    "?architecture=arm64",
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, templates map[string]types.RuntimeTemplate) {
				assert.NotNil(t, templates)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/api/v1/templates"+tt.queryParams, nil)
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.checkResponse != nil && w.Code == http.StatusOK {
				var templates map[string]types.RuntimeTemplate
				err := json.Unmarshal(w.Body.Bytes(), &templates)
				require.NoError(t, err)
				tt.checkResponse(t, templates)
			}
		})
	}
}

// TestHandleTemplateInfo tests the specific template info endpoint
func TestHandleTemplateInfo(t *testing.T) {
	server := createTestServer(t)
	handler := server.createHTTPHandler()

	tests := []struct {
		name           string
		templateName   string
		expectedStatus int
	}{
		{
			name:           "get existing test template",
			templateName:   "test-template", // Use slug instead of name with space
			expectedStatus: http.StatusOK,
		},
		{
			name:           "get non-existent template",
			templateName:   "non-existent-template",
			expectedStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := fmt.Sprintf("/api/v1/templates/%s", tt.templateName)
			req := httptest.NewRequest("GET", path, nil)
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
		})
	}
}

// TestHandleStorageVolumes tests storage volume operations
func TestHandleStorageVolumes(t *testing.T) {
	server := createTestServer(t)
	handler := server.createHTTPHandler()

	t.Run("list storage volumes", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/storage", nil)
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		// Storage returns an array, not a map
		var response []interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.NotNil(t, response)
	})

	t.Run("get non-existent volume", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/storage/non-existent", nil)
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		// Should return 404 or appropriate error
		assert.True(t, w.Code == http.StatusNotFound || w.Code == http.StatusInternalServerError)
	})
}

// TestHandleEBSVolumes tests EBS volume operations
func TestHandleEBSVolumes(t *testing.T) {
	server := createTestServer(t)
	handler := server.createHTTPHandler()

	tests := []struct {
		name           string
		method         string
		path           string
		expectedStatus int
	}{
		{
			name:           "list EBS volumes",
			method:         "GET",
			path:           "/api/v1/volumes",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "get non-existent EBS volume",
			method:         "GET",
			path:           "/api/v1/volumes/non-existent",
			expectedStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, nil)
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
		})
	}
}

// TestHandleProjects tests project management endpoints
func TestHandleProjects(t *testing.T) {
	server := createTestServer(t)
	handler := server.createHTTPHandler()

	t.Run("list projects", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/projects", nil)
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.NotNil(t, response)
	})

	t.Run("create project with invalid data", func(t *testing.T) {
		invalidProject := map[string]interface{}{
			"name": "", // Empty name should fail
		}
		body, _ := json.Marshal(invalidProject)

		req := httptest.NewRequest("POST", "/api/v1/projects", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

// TestHandleBudgets tests budget management endpoints
func TestHandleBudgets(t *testing.T) {
	server := createTestServer(t)
	handler := server.createHTTPHandler()

	t.Run("list budgets", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/budgets", nil)
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.NotNil(t, response)
	})
}

// TestHandleInvitations tests invitation endpoints
func TestHandleInvitations(t *testing.T) {
	server := createTestServer(t)
	handler := server.createHTTPHandler()

	t.Run("list invitations", func(t *testing.T) {
		// Use project invitations endpoint which exists
		req := httptest.NewRequest("GET", "/api/v1/projects/test-project/invitations", nil)
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		// Expect 404 since project doesn't exist, but endpoint is valid
		assert.True(t, w.Code == http.StatusOK || w.Code == http.StatusNotFound || w.Code == http.StatusInternalServerError)
	})
}

// TestHandleProfiles tests profile management endpoints
func TestHandleProfiles(t *testing.T) {
	server := createTestServer(t)
	handler := server.createHTTPHandler()

	t.Run("list profiles", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/profiles", nil)
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		// Profiles returns an array, not a map
		var response []interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.NotNil(t, response)
	})
}

// TestHandlerMethodValidation tests that handlers reject invalid HTTP methods
func TestHandlerMethodValidation(t *testing.T) {
	server := createTestServer(t)
	handler := server.createHTTPHandler()

	tests := []struct {
		name           string
		method         string
		path           string
		expectedStatus int
	}{
		{
			name:           "DELETE on templates endpoint",
			method:         "DELETE",
			path:           "/api/v1/templates",
			expectedStatus: http.StatusMethodNotAllowed,
		},
		{
			name:           "PUT on instances without ID",
			method:         "PUT",
			path:           "/api/v1/instances",
			expectedStatus: http.StatusMethodNotAllowed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, nil)
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
		})
	}
}

// TestHandlerConcurrentRequests tests concurrent access to handlers
func TestHandlerConcurrentRequests(t *testing.T) {
	server := createTestServer(t)
	handler := server.createHTTPHandler()

	const numRequests = 20
	done := make(chan bool, numRequests)
	errors := make(chan error, numRequests)

	endpoints := []string{
		"/api/v1/templates",
		"/api/v1/instances",
		"/api/v1/projects",
		"/api/v1/status",
	}

	// Launch concurrent requests
	for i := 0; i < numRequests; i++ {
		go func(index int) {
			endpoint := endpoints[index%len(endpoints)]
			req := httptest.NewRequest("GET", endpoint, nil)
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, req)

			if w.Code != http.StatusOK {
				errors <- fmt.Errorf("request to %s failed with status %d", endpoint, w.Code)
			}
			done <- true
		}(i)
	}

	// Wait for all requests to complete
	timeout := time.After(10 * time.Second)
	for i := 0; i < numRequests; i++ {
		select {
		case <-done:
			// Request completed
		case err := <-errors:
			t.Errorf("Concurrent request failed: %v", err)
		case <-timeout:
			t.Fatal("Timeout waiting for concurrent requests")
		}
	}
}

// TestHandlerRequestValidation tests request body validation
func TestHandlerRequestValidation(t *testing.T) {
	server := createTestServer(t)
	handler := server.createHTTPHandler()

	tests := []struct {
		name           string
		method         string
		path           string
		body           string
		expectedStatus int
	}{
		{
			name:           "malformed JSON",
			method:         "POST",
			path:           "/api/v1/instances",
			body:           `{"invalid": json}`,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "empty body for POST",
			method:         "POST",
			path:           "/api/v1/instances",
			body:           "",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "wrong content type",
			method:         "POST",
			path:           "/api/v1/instances",
			body:           `name=test`,
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, bytes.NewBufferString(tt.body))
			if tt.body != "" {
				req.Header.Set("Content-Type", "application/json")
			}
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
		})
	}
}

// TestHandlerResponseFormat tests that all handlers return proper JSON
func TestHandlerResponseFormat(t *testing.T) {
	server := createTestServer(t)
	handler := server.createHTTPHandler()

	endpoints := []string{
		"/api/v1/templates",
		"/api/v1/instances",
		"/api/v1/projects",
		"/api/v1/budgets",
		"/api/v1/profiles",
		"/api/v1/status",
	}

	for _, endpoint := range endpoints {
		t.Run(endpoint, func(t *testing.T) {
			req := httptest.NewRequest("GET", endpoint, nil)
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, req)

			// Should return 200
			assert.Equal(t, http.StatusOK, w.Code)

			// Should have JSON content type
			contentType := w.Header().Get("Content-Type")
			assert.Contains(t, contentType, "application/json")

			// Should be valid JSON
			var response interface{}
			err := json.Unmarshal(w.Body.Bytes(), &response)
			assert.NoError(t, err, "Response should be valid JSON")
		})
	}
}

// TestCheckInstanceNameUniqueness verifies that duplicate-name detection works via
// local state (fast path) without requiring a live AWS call.
func TestCheckInstanceNameUniqueness(t *testing.T) {
	server := createTestServer(t)

	saveInstance := func(name, id, state string) {
		err := server.stateManager.SaveInstance(types.Instance{
			Name:  name,
			ID:    id,
			State: state,
		})
		require.NoError(t, err)
	}

	callCheck := func(name string) *httptest.ResponseRecorder {
		req := httptest.NewRequest("POST", "/api/v1/instances", nil)
		w := httptest.NewRecorder()
		server.checkInstanceNameUniqueness(
			&types.LaunchRequest{Name: name, Template: "test-template"},
			w, req,
		)
		return w
	}

	t.Run("available name returns false (no conflict)", func(t *testing.T) {
		conflict := server.checkInstanceNameUniqueness(
			&types.LaunchRequest{Name: "brand-new", Template: "test-template"},
			httptest.NewRecorder(),
			httptest.NewRequest("POST", "/", nil),
		)
		assert.False(t, conflict)
	})

	t.Run("running instance blocks reuse", func(t *testing.T) {
		saveInstance("my-project", "i-aaa111", "running")
		w := callCheck("my-project")
		assert.Equal(t, http.StatusConflict, w.Code)
		assert.Contains(t, w.Body.String(), "my-project")
		assert.Contains(t, w.Body.String(), "i-aaa111")
		assert.Contains(t, w.Body.String(), "running")
	})

	t.Run("stopped instance blocks reuse", func(t *testing.T) {
		saveInstance("stopped-proj", "i-bbb222", "stopped")
		w := callCheck("stopped-proj")
		assert.Equal(t, http.StatusConflict, w.Code)
		assert.Contains(t, w.Body.String(), "stopped-proj")
	})

	t.Run("dry-run record allows name reuse", func(t *testing.T) {
		// A dry run creates nothing on AWS (#703). Older versions still persisted a
		// synthetic record for it, so the name check must not let such a record
		// reserve a name -- otherwise the real launch of that name comes back 409.
		saveInstance("dry-proj", "", "dry-run")
		conflict := server.checkInstanceNameUniqueness(
			&types.LaunchRequest{Name: "dry-proj", Template: "test-template"},
			httptest.NewRecorder(),
			httptest.NewRequest("POST", "/", nil),
		)
		assert.False(t, conflict)
	})

	t.Run("terminated instance allows name reuse", func(t *testing.T) {
		saveInstance("old-proj", "i-ccc333", "terminated")
		conflict := server.checkInstanceNameUniqueness(
			&types.LaunchRequest{Name: "old-proj", Template: "test-template"},
			httptest.NewRecorder(),
			httptest.NewRequest("POST", "/", nil),
		)
		assert.False(t, conflict)
	})

	t.Run("terminating instance allows name reuse", func(t *testing.T) {
		// "terminating" is treated the same as "terminated" — instance is going away
		saveInstance("dying-proj", "i-ddd444", "terminating")
		conflict := server.checkInstanceNameUniqueness(
			&types.LaunchRequest{Name: "dying-proj", Template: "test-template"},
			httptest.NewRecorder(),
			httptest.NewRequest("POST", "/", nil),
		)
		assert.False(t, conflict)
	})
}
