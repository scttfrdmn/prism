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

// TestListIdlePolicies tests the ListIdlePolicies method
func TestListIdlePolicies(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "GET", r.Method)
		assert.Equal(t, "/api/v1/idle/policies", r.URL.Path)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprint(w, `[
			{
				"id": "batch",
				"name": "Batch Processing",
				"description": "Long-running batch jobs",
				"idle_minutes": 60,
				"action": "hibernate"
			},
			{
				"id": "gpu",
				"name": "GPU Workstation",
				"description": "Expensive GPU instances",
				"idle_minutes": 15,
				"action": "stop"
			}
		]`)
	}))
	defer server.Close()

	client := NewClient(server.URL)
	policies, err := client.ListIdlePolicies(context.Background())

	assert.NoError(t, err)
	assert.NotNil(t, policies)
	assert.Len(t, policies, 2)

	// Verify first policy
	assert.Equal(t, "batch", policies[0].ID)
	assert.Equal(t, "Batch Processing", policies[0].Name)
	assert.Equal(t, "Long-running batch jobs", policies[0].Description)

	// Verify second policy
	assert.Equal(t, "gpu", policies[1].ID)
	assert.Equal(t, "GPU Workstation", policies[1].Name)
}

// TestListIdlePoliciesEmpty tests empty policy list
func TestListIdlePoliciesEmpty(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprint(w, `[]`)
	}))
	defer server.Close()

	client := NewClient(server.URL)
	policies, err := client.ListIdlePolicies(context.Background())

	assert.NoError(t, err)
	assert.NotNil(t, policies)
	assert.Empty(t, policies)
}

// TestListIdlePoliciesError tests error handling
func TestListIdlePoliciesError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = fmt.Fprint(w, `{"error": "internal server error"}`)
	}))
	defer server.Close()

	client := NewClient(server.URL)
	policies, err := client.ListIdlePolicies(context.Background())

	assert.Error(t, err)
	assert.Nil(t, policies)
	assert.Contains(t, err.Error(), "failed to list idle policies")
}

// TestListIdlePoliciesInvalidJSON tests invalid JSON response
func TestListIdlePoliciesInvalidJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprint(w, `{invalid json}`)
	}))
	defer server.Close()

	client := NewClient(server.URL)
	policies, err := client.ListIdlePolicies(context.Background())

	assert.Error(t, err)
	assert.Nil(t, policies)
	assert.Contains(t, err.Error(), "failed to decode policies")
}

// TestGetIdlePolicy tests the GetIdlePolicy method
func TestGetIdlePolicy(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "GET", r.Method)
		assert.Equal(t, "/api/v1/idle/policies/batch", r.URL.Path)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprint(w, `{
			"id": "batch",
			"name": "Batch Processing",
			"description": "Long-running batch jobs",
			"idle_minutes": 60,
			"action": "hibernate",
			"cpu_threshold": 5.0,
			"memory_threshold": 10.0
		}`)
	}))
	defer server.Close()

	client := NewClient(server.URL)
	policy, err := client.GetIdlePolicy(context.Background(), "batch")

	assert.NoError(t, err)
	assert.NotNil(t, policy)
	assert.Equal(t, "batch", policy.ID)
	assert.Equal(t, "Batch Processing", policy.Name)
	assert.Equal(t, "Long-running batch jobs", policy.Description)
}

// TestGetIdlePolicyNotFound tests 404 error handling
func TestGetIdlePolicyNotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = fmt.Fprint(w, `{"error": "policy not found"}`)
	}))
	defer server.Close()

	client := NewClient(server.URL)
	policy, err := client.GetIdlePolicy(context.Background(), "nonexistent")

	assert.Error(t, err)
	assert.Nil(t, policy)
	assert.Contains(t, err.Error(), "failed to get idle policy")
}

// TestGetIdlePolicyInvalidJSON tests invalid JSON response
func TestGetIdlePolicyInvalidJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprint(w, `{invalid json}`)
	}))
	defer server.Close()

	client := NewClient(server.URL)
	policy, err := client.GetIdlePolicy(context.Background(), "batch")

	assert.Error(t, err)
	assert.Nil(t, policy)
	assert.Contains(t, err.Error(), "failed to decode policy")
}

// TestApplyIdlePolicy tests the ApplyIdlePolicy method
func TestApplyIdlePolicy(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "/api/v1/idle/policies/apply", r.URL.Path)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

		// Verify request body
		var requestBody map[string]string
		err := readJSONBody(r, &requestBody)
		assert.NoError(t, err)
		assert.Equal(t, "test-instance", requestBody["instance_name"])
		assert.Equal(t, "batch", requestBody["policy_id"])

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewClient(server.URL)
	err := client.ApplyIdlePolicy(context.Background(), "test-instance", "batch")

	assert.NoError(t, err)
}

// TestApplyIdlePolicyError tests error handling
func TestApplyIdlePolicyError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = fmt.Fprint(w, `{"error": "invalid policy"}`)
	}))
	defer server.Close()

	client := NewClient(server.URL)
	err := client.ApplyIdlePolicy(context.Background(), "test-instance", "invalid")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to apply idle policy")
}

// TestRemoveIdlePolicy tests the RemoveIdlePolicy method
func TestRemoveIdlePolicy(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "/api/v1/idle/policies/remove", r.URL.Path)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

		// Verify request body
		var requestBody map[string]string
		err := readJSONBody(r, &requestBody)
		assert.NoError(t, err)
		assert.Equal(t, "test-instance", requestBody["instance_name"])
		assert.Equal(t, "batch", requestBody["policy_id"])

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewClient(server.URL)
	err := client.RemoveIdlePolicy(context.Background(), "test-instance", "batch")

	assert.NoError(t, err)
}

// TestRemoveIdlePolicyError tests error handling
func TestRemoveIdlePolicyError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = fmt.Fprint(w, `{"error": "policy not applied"}`)
	}))
	defer server.Close()

	client := NewClient(server.URL)
	err := client.RemoveIdlePolicy(context.Background(), "test-instance", "nonexistent")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to remove idle policy")
}

// TestGetInstanceIdlePolicies tests the GetInstanceIdlePolicies method
func TestGetInstanceIdlePolicies(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "GET", r.Method)
		assert.Equal(t, "/api/v1/instances/test-instance/idle-policies", r.URL.Path)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprint(w, `[
			{
				"id": "batch",
				"name": "Batch Processing",
				"idle_minutes": 60,
				"action": "hibernate"
			}
		]`)
	}))
	defer server.Close()

	client := NewClient(server.URL)
	policies, err := client.GetInstanceIdlePolicies(context.Background(), "test-instance")

	assert.NoError(t, err)
	assert.NotNil(t, policies)
	assert.Len(t, policies, 1)
	assert.Equal(t, "batch", policies[0].ID)
	assert.Equal(t, "Batch Processing", policies[0].Name)
}

// TestGetInstanceIdlePoliciesEmpty tests empty policy list for instance
func TestGetInstanceIdlePoliciesEmpty(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprint(w, `[]`)
	}))
	defer server.Close()

	client := NewClient(server.URL)
	policies, err := client.GetInstanceIdlePolicies(context.Background(), "test-instance")

	assert.NoError(t, err)
	assert.NotNil(t, policies)
	assert.Empty(t, policies)
}

// TestGetInstanceIdlePoliciesError tests error handling
func TestGetInstanceIdlePoliciesError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = fmt.Fprint(w, `{"error": "instance not found"}`)
	}))
	defer server.Close()

	client := NewClient(server.URL)
	policies, err := client.GetInstanceIdlePolicies(context.Background(), "nonexistent")

	assert.Error(t, err)
	assert.Nil(t, policies)
	assert.Contains(t, err.Error(), "failed to get instance idle policies")
}

// TestRecommendIdlePolicy tests the RecommendIdlePolicy method
func TestRecommendIdlePolicy(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "GET", r.Method)
		assert.Equal(t, "/api/v1/instances/gpu-instance/recommend-idle-policy", r.URL.Path)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprint(w, `{
			"id": "gpu",
			"name": "GPU Workstation",
			"description": "Recommended for expensive GPU instances",
			"idle_minutes": 15,
			"action": "stop",
			"reasoning": "GPU instances are expensive, quick stop recommended"
		}`)
	}))
	defer server.Close()

	client := NewClient(server.URL)
	policy, err := client.RecommendIdlePolicy(context.Background(), "gpu-instance")

	assert.NoError(t, err)
	assert.NotNil(t, policy)
	assert.Equal(t, "gpu", policy.ID)
	assert.Equal(t, "GPU Workstation", policy.Name)
	assert.Contains(t, policy.Description, "Recommended for expensive GPU instances")
}

// TestRecommendIdlePolicyError tests error handling
func TestRecommendIdlePolicyError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = fmt.Fprint(w, `{"error": "instance not found"}`)
	}))
	defer server.Close()

	client := NewClient(server.URL)
	policy, err := client.RecommendIdlePolicy(context.Background(), "nonexistent")

	assert.Error(t, err)
	assert.Nil(t, policy)
	assert.Contains(t, err.Error(), "failed to get idle policy recommendation")
}

// TestRecommendIdlePolicyInvalidJSON tests invalid JSON response
func TestRecommendIdlePolicyInvalidJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprint(w, `{invalid json}`)
	}))
	defer server.Close()

	client := NewClient(server.URL)
	policy, err := client.RecommendIdlePolicy(context.Background(), "test-instance")

	assert.Error(t, err)
	assert.Nil(t, policy)
	assert.Contains(t, err.Error(), "failed to decode policy")
}

// TestGetIdleSavingsReport tests the GetIdleSavingsReport method
func TestGetIdleSavingsReport(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "GET", r.Method)
		assert.Equal(t, "/api/v1/idle/savings", r.URL.Path)
		assert.Equal(t, "period=monthly", r.URL.RawQuery)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprint(w, `{
			"period": "monthly",
			"total_savings": 245.67,
			"hibernation_savings": 198.43,
			"stop_savings": 47.24,
			"instances_managed": 12,
			"total_idle_hours": 487.5,
			"currency": "USD"
		}`)
	}))
	defer server.Close()

	client := NewClient(server.URL)
	report, err := client.GetIdleSavingsReport(context.Background(), "monthly")

	assert.NoError(t, err)
	assert.NotNil(t, report)
	assert.Equal(t, "monthly", report["period"])
	assert.Equal(t, 245.67, report["total_savings"])
	assert.Equal(t, 198.43, report["hibernation_savings"])
	assert.Equal(t, 47.24, report["stop_savings"])
	assert.Equal(t, float64(12), report["instances_managed"])
	assert.Equal(t, 487.5, report["total_idle_hours"])
	assert.Equal(t, "USD", report["currency"])
}

// TestGetIdleSavingsReportDifferentPeriods tests different time periods
func TestGetIdleSavingsReportDifferentPeriods(t *testing.T) {
	periods := []string{"daily", "weekly", "monthly", "yearly"}

	for _, period := range periods {
		t.Run(period, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, fmt.Sprintf("period=%s", period), r.URL.RawQuery)

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				_, _ = fmt.Fprintf(w, `{
					"period": "%s",
					"total_savings": 50.00,
					"instances_managed": 5
				}`, period)
			}))
			defer server.Close()

			client := NewClient(server.URL)
			report, err := client.GetIdleSavingsReport(context.Background(), period)

			assert.NoError(t, err)
			assert.NotNil(t, report)
			assert.Equal(t, period, report["period"])
		})
	}
}

// TestGetIdleSavingsReportError tests error handling
func TestGetIdleSavingsReportError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = fmt.Fprint(w, `{"error": "invalid period"}`)
	}))
	defer server.Close()

	client := NewClient(server.URL)
	report, err := client.GetIdleSavingsReport(context.Background(), "invalid")

	assert.Error(t, err)
	assert.Nil(t, report)
	assert.Contains(t, err.Error(), "failed to get idle savings report")
}

// TestGetIdleSavingsReportInvalidJSON tests invalid JSON response
func TestGetIdleSavingsReportInvalidJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprint(w, `{invalid json}`)
	}))
	defer server.Close()

	client := NewClient(server.URL)
	report, err := client.GetIdleSavingsReport(context.Background(), "monthly")

	assert.Error(t, err)
	assert.Nil(t, report)
	assert.Contains(t, err.Error(), "failed to decode report")
}

// TestIdlePolicyContextCancellation tests context cancellation
func TestIdlePolicyContextCancellation(t *testing.T) {
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

	_, err := client.ListIdlePolicies(ctx)

	assert.Error(t, err)
	assert.Contains(t, strings.ToLower(err.Error()), "cancel")
}

// TestIdlePolicyMethods tests method combinations
func TestIdlePolicyMethods(t *testing.T) {
	requestCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		w.Header().Set("Content-Type", "application/json")

		switch {
		case strings.Contains(r.URL.Path, "/policies") && !strings.Contains(r.URL.Path, "/apply"):
			w.WriteHeader(http.StatusOK)
			_, _ = fmt.Fprint(w, `[{"id": "batch", "name": "Batch", "idle_minutes": 60}]`)
		case strings.Contains(r.URL.Path, "/apply"):
			w.WriteHeader(http.StatusOK)
		case strings.Contains(r.URL.Path, "/savings"):
			w.WriteHeader(http.StatusOK)
			_, _ = fmt.Fprint(w, `{"total_savings": 100.0}`)
		default:
			w.WriteHeader(http.StatusOK)
			_, _ = fmt.Fprint(w, `{"id": "batch"}`)
		}
	}))
	defer server.Close()

	client := NewClient(server.URL)
	ctx := context.Background()

	// Test method sequence
	policies, err := client.ListIdlePolicies(ctx)
	assert.NoError(t, err)
	assert.NotEmpty(t, policies)

	err = client.ApplyIdlePolicy(ctx, "test-instance", "batch")
	assert.NoError(t, err)

	report, err := client.GetIdleSavingsReport(ctx, "monthly")
	assert.NoError(t, err)
	assert.NotNil(t, report)

	// Verify all requests were made
	assert.Equal(t, 3, requestCount)
}

// Helper function to read JSON body from request (for testing)
func readJSONBody(r *http.Request, v interface{}) error {
	decoder := json.NewDecoder(r.Body)
	return decoder.Decode(v)
}
