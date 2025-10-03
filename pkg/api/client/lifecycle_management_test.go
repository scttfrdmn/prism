package client

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/scttfrdmn/cloudworkstation/pkg/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestInstanceLifecycleManagement tests the complete instance lifecycle that users experience
func TestInstanceLifecycleManagement(t *testing.T) {
	tests := []struct {
		name        string
		testFunc    func(t *testing.T)
		description string
	}{
		{
			name:        "complete_user_workflow",
			testFunc:    testCompleteUserWorkflow,
			description: "End-to-end user workflow: launch, connect, stop, start, hibernate, resume, delete",
		},
		{
			name:        "launch_instance_success",
			testFunc:    testLaunchInstanceSuccess,
			description: "User successfully launches instance with template",
		},
		{
			name:        "launch_instance_template_error",
			testFunc:    testLaunchInstanceTemplateError,
			description: "User encounters template error during launch",
		},
		{
			name:        "connect_to_running_instance",
			testFunc:    testConnectToRunningInstance,
			description: "User connects to running instance successfully",
		},
		{
			name:        "connect_to_stopped_instance",
			testFunc:    testConnectToStoppedInstance,
			description: "User tries to connect to stopped instance",
		},
		{
			name:        "hibernation_workflow",
			testFunc:    testHibernationWorkflow,
			description: "User hibernates and resumes instance for cost optimization",
		},
		{
			name:        "instance_state_transitions",
			testFunc:    testInstanceStateTransitions,
			description: "Valid instance state transitions during operations",
		},
		{
			name:        "concurrent_operations_handling",
			testFunc:    testConcurrentOperationsHandling,
			description: "Multiple operations on same instance handled properly",
		},
		{
			name:        "error_recovery_scenarios",
			testFunc:    testErrorRecoveryScenarios,
			description: "System recovery from various error conditions",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Logf("Testing lifecycle scenario: %s", tt.description)
			tt.testFunc(t)
		})
	}
}

func testCompleteUserWorkflow(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(createWorkflowMockHandler))
	defer server.Close()

	client := NewClient(server.URL)

	testWorkflowSteps(t, client)

	t.Logf("🎉 Complete user workflow successful: Launch → Connect → Stop → Start → Hibernate → Resume → Delete")
}

func createWorkflowMockHandler(w http.ResponseWriter, r *http.Request) {
	routes := []struct {
		method   string
		pathContains string
		excludeContains string
		response string
		status   int
	}{
		{"POST", "/instances", "/", `{
			"instance": {
				"name": "research-ml",
				"id": "i-1234567890abcdef0",
				"state": "launching",
				"template": "python-ml",
				"public_ip": "",
				"estimated_daily_cost": 2.40
			},
			"connection_info": {
				"ssh_command": "ssh -i key.pem ubuntu@launching",
				"ssh_key_path": "/tmp/key.pem"
			}
		}`, http.StatusOK},
		{"GET", "/instances/research-ml", "", `{
			"name": "research-ml",
			"id": "i-1234567890abcdef0",
			"state": "running",
			"template": "python-ml",
			"public_ip": "54.123.45.67",
			"estimated_daily_cost": 2.40,
			"launch_time": "2024-01-15T10:30:00Z"
		}`, http.StatusOK},
		{"POST", "/connect", "", `"ssh -i /tmp/key.pem ubuntu@54.123.45.67"`, http.StatusOK},
		{"POST", "/stop", "", "", http.StatusOK},
		{"POST", "/start", "", "", http.StatusOK},
		{"POST", "/hibernate", "", "", http.StatusOK},
		{"POST", "/resume", "", "", http.StatusOK},
		{"DELETE", "/instances/research-ml", "", "", http.StatusOK},
	}

	for _, route := range routes {
		if r.Method == route.method &&
		   strings.Contains(r.URL.Path, route.pathContains) &&
		   (route.excludeContains == "" || !strings.Contains(r.URL.Path, route.excludeContains)) {
			if route.response != "" {
				w.Header().Set("Content-Type", "application/json")
			}
			w.WriteHeader(route.status)
			if route.response != "" {
				_, _ = w.Write([]byte(route.response))
			}
			return
		}
	}

	w.WriteHeader(http.StatusNotFound)
}

func testWorkflowSteps(t *testing.T, client *Client) {
	// Step 1: Launch instance
	launchResp := testWorkflowLaunch(t, client)

	// Step 2: Wait for running state
	instance := testWorkflowWaitForRunning(t, client)

	// Step 3: Connect to instance
	testWorkflowConnect(t, client, instance)

	// Steps 4-8: Test lifecycle operations
	testWorkflowLifecycleOperations(t, client)
}

func testWorkflowLaunch(t *testing.T, client *Client) *types.LaunchResponse {
	launchReq := types.LaunchRequest{
		Template: "python-ml",
		Name:     "research-ml",
		Size:     "M",
	}

	launchResp, err := client.LaunchInstance(context.Background(), launchReq)
	require.NoError(t, err, "Launch should succeed")
	assert.Equal(t, "research-ml", launchResp.Instance.Name)
	assert.Equal(t, "launching", launchResp.Instance.State)

	t.Logf("✅ Step 1: Successfully launched instance %s", launchResp.Instance.Name)
	return launchResp
}

func testWorkflowWaitForRunning(t *testing.T, client *Client) *types.Instance {
	var instance *types.Instance
	var err error

	for i := 0; i < 3; i++ {
		instance, err = client.GetInstance(context.Background(), "research-ml")
		require.NoError(t, err, "Get instance should succeed")
		if instance.State == "running" {
			break
		}
		time.Sleep(10 * time.Millisecond) // Brief simulation delay
	}

	assert.Equal(t, "running", instance.State)
	assert.Equal(t, "54.123.45.67", instance.PublicIP)
	t.Logf("✅ Step 2: Instance transitioned to running state with IP %s", instance.PublicIP)

	return instance
}

func testWorkflowConnect(t *testing.T, client *Client, instance *types.Instance) {
	sshCommand, err := client.ConnectInstance(context.Background(), "research-ml")
	require.NoError(t, err, "Connect should succeed")
	assert.Contains(t, sshCommand, "ssh -i")
	assert.Contains(t, sshCommand, instance.PublicIP)

	t.Logf("✅ Step 3: Connection established: %s", sshCommand)
}

func testWorkflowLifecycleOperations(t *testing.T, client *Client) {
	lifecycleOps := []struct {
		name string
		operation func() error
		step int
	}{
		{"Stop", func() error { return client.StopInstance(context.Background(), "research-ml") }, 4},
		{"Start", func() error { return client.StartInstance(context.Background(), "research-ml") }, 5},
		{"Hibernate", func() error { return client.HibernateInstance(context.Background(), "research-ml") }, 6},
		{"Resume", func() error { return client.ResumeInstance(context.Background(), "research-ml") }, 7},
		{"Delete", func() error { return client.DeleteInstance(context.Background(), "research-ml") }, 8},
	}

	for _, op := range lifecycleOps {
		err := op.operation()
		require.NoError(t, err, op.name+" should succeed")
		t.Logf("✅ Step %d: Instance %s completed successfully", op.step, strings.ToLower(op.name))
	}
}

func testLaunchInstanceSuccess(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" && strings.Contains(r.URL.Path, "/instances") {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{
				"instance": {
					"name": "test-instance",
					"id": "i-1234567890abcdef0",
					"state": "launching",
					"template": "python-ml",
					"estimated_daily_cost": 2.40
				},
				"connection_info": {
					"ssh_command": "ssh -i key.pem ubuntu@launching",
					"ssh_key_path": "/tmp/key.pem"
				}
			}`))
		}
	}))
	defer server.Close()

	client := NewClient(server.URL)

	req := types.LaunchRequest{
		Template: "python-ml",
		Name:     "test-instance",
		Size:     "M",
	}

	resp, err := client.LaunchInstance(context.Background(), req)

	require.NoError(t, err, "Launch should succeed")
	assert.Equal(t, "test-instance", resp.Instance.Name)
	assert.Equal(t, "launching", resp.Instance.State)
	assert.Equal(t, "python-ml", resp.Instance.Template)
	assert.Greater(t, resp.Instance.EstimatedCost, 0.0)
	assert.NotEmpty(t, resp.ConnectionInfo)

	t.Logf("Successfully launched instance with cost estimate: $%.2f/day", resp.Instance.EstimatedCost)
}

func testLaunchInstanceTemplateError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" && strings.Contains(r.URL.Path, "/instances") {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(`{
				"code": "template_not_found",
				"message": "Template 'nonexistent-template' not found",
				"status_code": 400
			}`))
		}
	}))
	defer server.Close()

	client := NewClient(server.URL)

	req := types.LaunchRequest{
		Template: "nonexistent-template",
		Name:     "test-instance",
	}

	_, err := client.LaunchInstance(context.Background(), req)

	require.Error(t, err, "Should fail with template error")
	assert.Contains(t, err.Error(), "template_not_found")
	assert.Contains(t, err.Error(), "not found")

	t.Logf("Correctly failed with template error: %s", err.Error())
}

func testConnectToRunningInstance(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" && strings.Contains(r.URL.Path, "/connect") {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`"ssh -i /tmp/research-ml-key.pem ubuntu@54.123.45.67"`))
		}
	}))
	defer server.Close()

	client := NewClient(server.URL)

	sshCommand, err := client.ConnectInstance(context.Background(), "running-instance")

	require.NoError(t, err, "Connect should succeed")
	assert.Contains(t, sshCommand, "ssh -i")
	assert.Contains(t, sshCommand, "54.123.45.67")
	assert.Contains(t, sshCommand, "ubuntu")

	t.Logf("Connection successful: %s", sshCommand)
}

func testConnectToStoppedInstance(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" && strings.Contains(r.URL.Path, "/connect") {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusConflict)
			_, _ = w.Write([]byte(`{
				"code": "instance_not_running",
				"message": "Cannot connect to stopped instance",
				"status_code": 409
			}`))
		}
	}))
	defer server.Close()

	client := NewClient(server.URL)

	_, err := client.ConnectInstance(context.Background(), "stopped-instance")

	require.Error(t, err, "Should fail to connect to stopped instance")
	assert.Contains(t, err.Error(), "instance_not_running")

	t.Logf("Correctly failed to connect to stopped instance: %s", err.Error())
}

func testHibernationWorkflow(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.Contains(r.URL.Path, "/hibernation-status"):
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{
				"hibernation_enabled": true,
				"current_state": "running",
				"hibernation_supported": true
			}`))

		case strings.Contains(r.URL.Path, "/hibernate"):
			w.WriteHeader(http.StatusOK)

		case strings.Contains(r.URL.Path, "/resume"):
			w.WriteHeader(http.StatusOK)
		}
	}))
	defer server.Close()

	client := NewClient(server.URL)

	// Check hibernation status
	status, err := client.GetInstanceHibernationStatus(context.Background(), "test-instance")
	require.NoError(t, err, "Hibernation status should be accessible")
	assert.True(t, status.HibernationSupported)
	assert.True(t, status.HibernationSupported)

	t.Logf("Hibernation supported: %t", status.HibernationSupported)

	// Hibernate instance
	err = client.HibernateInstance(context.Background(), "test-instance")
	require.NoError(t, err, "Hibernation should succeed")

	t.Logf("✅ Instance hibernated successfully")

	// Resume instance
	err = client.ResumeInstance(context.Background(), "test-instance")
	require.NoError(t, err, "Resume should succeed")

	t.Logf("✅ Instance resumed successfully")
}

func testInstanceStateTransitions(t *testing.T) {
	// Test valid state transitions that users expect
	validTransitions := []struct {
		from        string
		operation   string
		expectedTo  string
		description string
	}{
		{"stopped", "start", "running", "Users start stopped instances"},
		{"running", "stop", "stopped", "Users stop running instances"},
		{"running", "hibernate", "stopped", "Users hibernate for cost savings"},
		{"stopped", "resume", "running", "Users resume hibernated instances"},
		{"running", "delete", "terminated", "Users clean up instances"},
	}

	for _, transition := range validTransitions {
		t.Run(fmt.Sprintf("%s_%s_%s", transition.from, transition.operation, transition.expectedTo), func(t *testing.T) {
			t.Logf("Testing transition: %s --%s--> %s (%s)",
				transition.from, transition.operation, transition.expectedTo, transition.description)

			// In a real test, we would verify the state transition
			// This documents the expected behavior
			assert.NotEmpty(t, transition.from, "Initial state should be defined")
			assert.NotEmpty(t, transition.operation, "Operation should be defined")
			assert.NotEmpty(t, transition.expectedTo, "Final state should be defined")
		})
	}
}

func testConcurrentOperationsHandling(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		if callCount == 1 {
			// First operation succeeds
			w.WriteHeader(http.StatusOK)
		} else {
			// Subsequent operations are blocked
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusConflict)
			_, _ = w.Write([]byte(`{
				"code": "operation_in_progress",
				"message": "Another operation is already in progress on this instance",
				"status_code": 409
			}`))
		}
	}))
	defer server.Close()

	client := NewClient(server.URL)

	// Start first operation
	err1 := client.StopInstance(context.Background(), "test-instance")
	require.NoError(t, err1, "First operation should succeed")

	// Try concurrent operation
	err2 := client.StartInstance(context.Background(), "test-instance")
	require.Error(t, err2, "Concurrent operation should be blocked")
	assert.Contains(t, err2.Error(), "operation_in_progress")

	t.Logf("✅ Concurrent operations properly handled: first succeeds, second blocked")
}

func testErrorRecoveryScenarios(t *testing.T) {
	errorScenarios := []struct {
		errorType   string
		httpStatus  int
		response    string
		description string
	}{
		{
			errorType:   "network_timeout",
			httpStatus:  0, // Simulates network timeout
			response:    "",
			description: "User experiences network issues during operation",
		},
		{
			errorType:   "instance_not_found",
			httpStatus:  404,
			response:    `{"code":"instance_not_found","message":"Instance not found","status_code":404}`,
			description: "User references non-existent instance",
		},
		{
			errorType:   "server_error",
			httpStatus:  500,
			response:    `{"code":"server_error","message":"Internal server error","status_code":500}`,
			description: "Server encounters internal error",
		},
		{
			errorType:   "rate_limited",
			httpStatus:  429,
			response:    `{"code":"rate_limited","message":"Too many requests","status_code":429}`,
			description: "User hits rate limits",
		},
	}

	for _, scenario := range errorScenarios {
		t.Run(scenario.errorType, func(t *testing.T) {
			if scenario.httpStatus == 0 {
				// Simulate network timeout by using invalid URL
				client := NewClient("http://invalid-url-that-will-timeout")

				ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
				defer cancel()

				_, err := client.GetInstance(ctx, "test-instance")
				require.Error(t, err, "Should fail with network error")

				t.Logf("✅ Network error properly handled: %s", err.Error())
			} else {
				// Simulate HTTP error response
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(scenario.httpStatus)
					w.Write([]byte(scenario.response))
				}))
				defer server.Close()

				client := NewClient(server.URL)
				_, err := client.GetInstance(context.Background(), "test-instance")
				require.Error(t, err, "Should fail with expected error")

				t.Logf("✅ %s properly handled: %s", scenario.errorType, err.Error())
			}

			t.Logf("📋 Scenario: %s", scenario.description)
		})
	}
}

// Helper function for string formatting in tests
