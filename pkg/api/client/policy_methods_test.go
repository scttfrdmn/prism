package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

// Test-specific response types that match what the interface expects

// TestGetPolicyStatus tests the GetPolicyStatus method
func TestGetPolicyStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "GET", r.Method)
		assert.Equal(t, "/api/v1/policies/status", r.URL.Path)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprint(w, `{
			"enabled": true,
			"current_policy_set": "research",
			"last_updated": "2024-01-15T10:30:00Z",
			"status": "active"
		}`)
	}))
	defer server.Close()

	client := NewClient(server.URL)
	response, err := client.GetPolicyStatus(context.Background())

	assert.NoError(t, err)
	assert.NotNil(t, response)
	assert.True(t, response.Enabled)
	assert.Equal(t, "active", response.Status)
	assert.Contains(t, response.AssignedPolicies, "research")
}

// TestGetPolicyStatusDisabled tests disabled policy enforcement
func TestGetPolicyStatusDisabled(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprint(w, `{
			"enabled": false,
			"current_policy_set": "",
			"last_updated": "2024-01-10T08:15:00Z",
			"status": "disabled"
		}`)
	}))
	defer server.Close()

	client := NewClient(server.URL)
	response, err := client.GetPolicyStatus(context.Background())

	assert.NoError(t, err)
	assert.NotNil(t, response)
	assert.False(t, response.Enabled)
	assert.Equal(t, "disabled", response.Status)
}

// TestGetPolicyStatusError tests error handling
func TestGetPolicyStatusError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = fmt.Fprint(w, `{"error": "internal server error"}`)
	}))
	defer server.Close()

	client := NewClient(server.URL)
	response, err := client.GetPolicyStatus(context.Background())

	assert.Error(t, err)
	assert.Nil(t, response)
	assert.Contains(t, err.Error(), "failed to get policy status")
}

// TestListPolicySets tests the ListPolicySets method
func TestListPolicySets(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "GET", r.Method)
		assert.Equal(t, "/api/v1/policies/sets", r.URL.Path)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprint(w, `{
			"policy_sets": {
				"research": {
					"id": "research",
					"name": "Research Policy",
					"description": "Standard research templates",
					"policies": 5,
					"status": "active"
				},
				"student": {
					"id": "student",
					"name": "Student Policy",
					"description": "Limited student access",
					"policies": 2,
					"status": "active"
				},
				"admin": {
					"id": "admin",
					"name": "Admin Policy",
					"description": "Full administrative access",
					"policies": 10,
					"status": "active"
				}
			}
		}`)
	}))
	defer server.Close()

	client := NewClient(server.URL)
	response, err := client.ListPolicySets(context.Background())

	assert.NoError(t, err)
	assert.NotNil(t, response)
	assert.Len(t, response.PolicySets, 3)

	// Verify research policy set
	researchPolicy := response.PolicySets["research"]
	assert.Equal(t, "research", researchPolicy.ID)
	assert.Equal(t, "Research Policy", researchPolicy.Name)
	assert.Equal(t, "Standard research templates", researchPolicy.Description)
	assert.Equal(t, 5, researchPolicy.Policies)
	assert.Equal(t, "active", researchPolicy.Status)

	// Verify student policy set
	studentPolicy := response.PolicySets["student"]
	assert.Equal(t, "student", studentPolicy.ID)
	assert.Equal(t, "Student Policy", studentPolicy.Name)
	assert.Equal(t, 2, studentPolicy.Policies)

	// Verify admin policy set
	adminPolicy := response.PolicySets["admin"]
	assert.Equal(t, "admin", adminPolicy.ID)
	assert.Equal(t, "Admin Policy", adminPolicy.Name)
	assert.Equal(t, 10, adminPolicy.Policies)
}

// TestListPolicySetsEmpty tests empty policy sets
func TestListPolicySetsEmpty(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprint(w, `{
			"policy_sets": {}
		}`)
	}))
	defer server.Close()

	client := NewClient(server.URL)
	response, err := client.ListPolicySets(context.Background())

	assert.NoError(t, err)
	assert.NotNil(t, response)
	assert.Empty(t, response.PolicySets)
}

// TestListPolicySetsError tests error handling
func TestListPolicySetsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = fmt.Fprint(w, `{"error": "insufficient permissions"}`)
	}))
	defer server.Close()

	client := NewClient(server.URL)
	response, err := client.ListPolicySets(context.Background())

	assert.Error(t, err)
	assert.Nil(t, response)
	assert.Contains(t, err.Error(), "failed to list policy sets")
}

// TestAssignPolicySet tests the AssignPolicySet method
func TestAssignPolicySet(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "/api/v1/policies/assign", r.URL.Path)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

		// Verify request body
		var requestBody map[string]string
		err := json.NewDecoder(r.Body).Decode(&requestBody)
		assert.NoError(t, err)
		assert.Equal(t, "student", requestBody["policy_set"])

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprint(w, `{
			"success": true,
			"policy_set": "student",
			"message": "Policy set assigned successfully",
			"effective_date": "2024-01-15T10:30:00Z"
		}`)
	}))
	defer server.Close()

	client := NewClient(server.URL)
	response, err := client.AssignPolicySet(context.Background(), "student")

	assert.NoError(t, err)
	assert.NotNil(t, response)
	assert.True(t, response.Success)
	assert.Equal(t, "student", response.AssignedPolicySet)
	assert.Equal(t, "Policy set assigned successfully", response.Message)
}

// TestAssignPolicySetEmptyName tests empty policy set name validation
func TestAssignPolicySetEmptyName(t *testing.T) {
	client := NewClient("http://localhost:8947")
	response, err := client.AssignPolicySet(context.Background(), "")

	assert.Error(t, err)
	assert.Nil(t, response)
	assert.Contains(t, err.Error(), "policy set name cannot be empty")
}

// TestAssignPolicySetError tests error handling
func TestAssignPolicySetError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = fmt.Fprint(w, `{"error": "policy set not found"}`)
	}))
	defer server.Close()

	client := NewClient(server.URL)
	response, err := client.AssignPolicySet(context.Background(), "nonexistent")

	assert.Error(t, err)
	assert.Nil(t, response)
	assert.Contains(t, err.Error(), "failed to assign policy set 'nonexistent'")
}

// TestAssignPolicySetUnauthorized tests unauthorized access
func TestAssignPolicySetUnauthorized(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = fmt.Fprint(w, `{"error": "unauthorized"}`)
	}))
	defer server.Close()

	client := NewClient(server.URL)
	response, err := client.AssignPolicySet(context.Background(), "admin")

	assert.Error(t, err)
	assert.Nil(t, response)
	assert.Contains(t, err.Error(), "failed to assign policy set 'admin'")
}

// TestSetPolicyEnforcement tests the SetPolicyEnforcement method
func TestSetPolicyEnforcement(t *testing.T) {
	tests := []struct {
		name           string
		enabledValue   bool
		expectedBody   string
		responseBody   string
		expectedResult bool
	}{
		{
			name:         "enable_enforcement",
			enabledValue: true,
			expectedBody: `{"enabled":true}`,
			responseBody: `{
				"enabled": true,
				"previous_state": false,
				"message": "Policy enforcement enabled",
				"updated_at": "2024-01-15T10:30:00Z"
			}`,
			expectedResult: true,
		},
		{
			name:         "disable_enforcement",
			enabledValue: false,
			expectedBody: `{"enabled":false}`,
			responseBody: `{
				"enabled": false,
				"previous_state": true,
				"message": "Policy enforcement disabled",
				"updated_at": "2024-01-15T10:35:00Z"
			}`,
			expectedResult: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "POST", r.Method)
				assert.Equal(t, "/api/v1/policies/enforcement", r.URL.Path)
				assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

				// Verify request body
				var requestBody map[string]bool
				err := json.NewDecoder(r.Body).Decode(&requestBody)
				assert.NoError(t, err)
				assert.Equal(t, tt.enabledValue, requestBody["enabled"])

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				_, _ = fmt.Fprint(w, tt.responseBody)
			}))
			defer server.Close()

			client := NewClient(server.URL)
			response, err := client.SetPolicyEnforcement(context.Background(), tt.enabledValue)

			assert.NoError(t, err)
			assert.NotNil(t, response)
			assert.True(t, response.Success)
			assert.Equal(t, tt.expectedResult, response.Enabled)
			assert.NotEmpty(t, response.Message)
		})
	}
}

// TestSetPolicyEnforcementError tests error handling
func TestSetPolicyEnforcementError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = fmt.Fprint(w, `{"error": "insufficient permissions"}`)
	}))
	defer server.Close()

	client := NewClient(server.URL)
	response, err := client.SetPolicyEnforcement(context.Background(), true)

	assert.Error(t, err)
	assert.Nil(t, response)
	assert.Contains(t, err.Error(), "failed to set policy enforcement to true")
}

// TestCheckTemplateAccess tests the CheckTemplateAccess method
func TestCheckTemplateAccess(t *testing.T) {
	tests := []struct {
		name            string
		template        string
		responseBody    string
		expectedAllowed bool
		expectedReason  string
	}{
		{
			name:     "allowed_template",
			template: "python-ml",
			responseBody: `{
				"allowed": true,
				"template": "python-ml",
				"reason": "Template allowed under research policy",
				"policy_set": "research"
			}`,
			expectedAllowed: true,
			expectedReason:  "Template allowed under research policy",
		},
		{
			name:     "blocked_template",
			template: "gpu-intensive",
			responseBody: `{
				"allowed": false,
				"template": "gpu-intensive",
				"reason": "Template not permitted for student accounts",
				"policy_set": "student",
				"restrictions": ["max_cost_exceeded", "gpu_not_allowed"]
			}`,
			expectedAllowed: false,
			expectedReason:  "Template not permitted for student accounts",
		},
		{
			name:     "admin_override",
			template: "admin-tools",
			responseBody: `{
				"allowed": true,
				"template": "admin-tools",
				"reason": "Admin policy allows all templates",
				"policy_set": "admin"
			}`,
			expectedAllowed: true,
			expectedReason:  "Admin policy allows all templates",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "POST", r.Method)
				assert.Equal(t, "/api/v1/policies/check", r.URL.Path)
				assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

				// Verify request body
				var requestBody map[string]string
				err := json.NewDecoder(r.Body).Decode(&requestBody)
				assert.NoError(t, err)
				assert.Equal(t, tt.template, requestBody["template_name"])

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				_, _ = fmt.Fprint(w, tt.responseBody)
			}))
			defer server.Close()

			client := NewClient(server.URL)
			response, err := client.CheckTemplateAccess(context.Background(), tt.template)

			assert.NoError(t, err)
			assert.NotNil(t, response)
			assert.Equal(t, tt.expectedAllowed, response.Allowed)
			assert.Equal(t, tt.template, response.TemplateName)
			assert.Equal(t, tt.expectedReason, response.Reason)
		})
	}
}

// TestCheckTemplateAccessEmptyName tests empty template name validation
func TestCheckTemplateAccessEmptyName(t *testing.T) {
	client := NewClient("http://localhost:8947")
	response, err := client.CheckTemplateAccess(context.Background(), "")

	assert.Error(t, err)
	assert.Nil(t, response)
	assert.Contains(t, err.Error(), "template name cannot be empty")
}

// TestCheckTemplateAccessError tests error handling
func TestCheckTemplateAccessError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = fmt.Fprint(w, `{"error": "template not found"}`)
	}))
	defer server.Close()

	client := NewClient(server.URL)
	response, err := client.CheckTemplateAccess(context.Background(), "nonexistent")

	assert.Error(t, err)
	assert.Nil(t, response)
	assert.Contains(t, err.Error(), "failed to check template access for 'nonexistent'")
}

// TestPolicyMethodsContextCancellation tests context cancellation
func TestPolicyMethodsContextCancellation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Delay to trigger context cancellation
		select {
		case <-r.Context().Done():
			return
		case <-context.Background().Done():
			return
		}
	}))
	defer server.Close()

	client := NewClient(server.URL)

	// Create context that cancels immediately
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	// Test all methods with cancelled context
	_, err := client.GetPolicyStatus(ctx)
	assert.Error(t, err)
	assert.Contains(t, strings.ToLower(err.Error()), "cancel")

	_, err = client.ListPolicySets(ctx)
	assert.Error(t, err)
	assert.Contains(t, strings.ToLower(err.Error()), "cancel")

	_, err = client.AssignPolicySet(ctx, "research")
	assert.Error(t, err)
	assert.Contains(t, strings.ToLower(err.Error()), "cancel")

	_, err = client.SetPolicyEnforcement(ctx, true)
	assert.Error(t, err)
	assert.Contains(t, strings.ToLower(err.Error()), "cancel")

	_, err = client.CheckTemplateAccess(ctx, "python-ml")
	assert.Error(t, err)
	assert.Contains(t, strings.ToLower(err.Error()), "cancel")
}

// TestPolicyWorkflowIntegration tests a complete policy workflow
func TestPolicyWorkflowIntegration(t *testing.T) {
	requestCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		w.Header().Set("Content-Type", "application/json")

		switch {
		case strings.Contains(r.URL.Path, "/status"):
			w.WriteHeader(http.StatusOK)
			_, _ = fmt.Fprint(w, `{"enabled": false, "current_policy_set": ""}`)

		case strings.Contains(r.URL.Path, "/sets"):
			w.WriteHeader(http.StatusOK)
			_, _ = fmt.Fprint(w, `{
				"policy_sets": [{"name": "research", "description": "Research access"}],
				"default": "research"
			}`)

		case strings.Contains(r.URL.Path, "/assign"):
			w.WriteHeader(http.StatusOK)
			_, _ = fmt.Fprint(w, `{
				"success": true,
				"policy_set": "research",
				"message": "Assigned successfully"
			}`)

		case strings.Contains(r.URL.Path, "/enforcement"):
			w.WriteHeader(http.StatusOK)
			_, _ = fmt.Fprint(w, `{
				"enabled": true,
				"previous_state": false,
				"message": "Enforcement enabled"
			}`)

		case strings.Contains(r.URL.Path, "/check"):
			w.WriteHeader(http.StatusOK)
			_, _ = fmt.Fprint(w, `{
				"allowed": true,
				"template": "python-ml",
				"reason": "Template allowed",
				"policy_set": "research"
			}`)

		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	client := NewClient(server.URL)
	ctx := context.Background()

	// Workflow: Check status -> List sets -> Assign set -> Enable enforcement -> Check access
	status, err := client.GetPolicyStatus(ctx)
	assert.NoError(t, err)
	assert.False(t, status.Enabled)

	sets, err := client.ListPolicySets(ctx)
	assert.NoError(t, err)
	assert.NotEmpty(t, sets.PolicySets)

	assign, err := client.AssignPolicySet(ctx, "research")
	assert.NoError(t, err)
	assert.True(t, assign.Success)

	enforcement, err := client.SetPolicyEnforcement(ctx, true)
	assert.NoError(t, err)
	assert.True(t, enforcement.Enabled)

	check, err := client.CheckTemplateAccess(ctx, "python-ml")
	assert.NoError(t, err)
	assert.True(t, check.Allowed)

	// Verify all requests were made
	assert.Equal(t, 5, requestCount)
}

// TestPolicyMethodsWithAuth tests authenticated policy requests
func TestPolicyMethodsWithAuth(t *testing.T) {
	var receivedHeaders http.Header

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedHeaders = r.Header.Clone()
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprint(w, `{"enabled": true}`)
	}))
	defer server.Close()

	options := Options{
		AWSProfile: "policy-profile",
		AWSRegion:  "us-east-1",
	}
	client := NewClientWithOptions(server.URL, options)

	_, err := client.GetPolicyStatus(context.Background())

	assert.NoError(t, err)
	assert.Equal(t, "policy-profile", receivedHeaders.Get("X-AWS-Profile"))
	assert.Equal(t, "us-east-1", receivedHeaders.Get("X-AWS-Region"))
}

// TestPolicyMethodsErrorScenarios tests various error scenarios
func TestPolicyMethodsErrorScenarios(t *testing.T) {
	errorTests := []struct {
		name       string
		statusCode int
		method     func(client PrismAPI) error
	}{
		{
			name:       "GetPolicyStatus_500",
			statusCode: http.StatusInternalServerError,
			method: func(client PrismAPI) error {
				_, err := client.GetPolicyStatus(context.Background())
				return err
			},
		},
		{
			name:       "ListPolicySets_403",
			statusCode: http.StatusForbidden,
			method: func(client PrismAPI) error {
				_, err := client.ListPolicySets(context.Background())
				return err
			},
		},
		{
			name:       "AssignPolicySet_404",
			statusCode: http.StatusNotFound,
			method: func(client PrismAPI) error {
				_, err := client.AssignPolicySet(context.Background(), "invalid")
				return err
			},
		},
		{
			name:       "SetPolicyEnforcement_401",
			statusCode: http.StatusUnauthorized,
			method: func(client PrismAPI) error {
				_, err := client.SetPolicyEnforcement(context.Background(), true)
				return err
			},
		},
		{
			name:       "CheckTemplateAccess_400",
			statusCode: http.StatusBadRequest,
			method: func(client PrismAPI) error {
				_, err := client.CheckTemplateAccess(context.Background(), "invalid")
				return err
			},
		},
	}

	for _, tt := range errorTests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.statusCode)
				_, _ = fmt.Fprint(w, `{"error": "test error"}`)
			}))
			defer server.Close()

			client := NewClient(server.URL)
			err := tt.method(client)

			assert.Error(t, err)
			assert.Contains(t, err.Error(), "failed to")
		})
	}
}

// TestPolicyMethodsJSONErrorHandling tests invalid JSON response handling
func TestPolicyMethodsJSONErrorHandling(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprint(w, `{invalid json}`)
	}))
	defer server.Close()

	client := NewClient(server.URL)
	ctx := context.Background()

	// All methods should handle JSON decode errors gracefully
	_, err := client.GetPolicyStatus(ctx)
	assert.Error(t, err)

	_, err = client.ListPolicySets(ctx)
	assert.Error(t, err)

	_, err = client.AssignPolicySet(ctx, "test")
	assert.Error(t, err)

	_, err = client.SetPolicyEnforcement(ctx, true)
	assert.Error(t, err)

	_, err = client.CheckTemplateAccess(ctx, "test")
	assert.Error(t, err)
}
