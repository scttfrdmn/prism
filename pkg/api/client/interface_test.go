package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/scttfrdmn/cloudworkstation/pkg/project"
	"github.com/scttfrdmn/cloudworkstation/pkg/types"
)

// TestCloudWorkstationAPIInterface tests that HTTPClient implements the interface
func TestCloudWorkstationAPIInterface(t *testing.T) {
	var _ CloudWorkstationAPI = &HTTPClient{}
}

// TestHTTPClientImplementsAllMethods tests that all interface methods are properly implemented
func TestHTTPClientImplementsAllMethods(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch r.URL.Path {
		case "/api/v1/ping":
			w.WriteHeader(http.StatusOK)
		case "/api/v1/status":
			w.WriteHeader(http.StatusOK)
			_, _ = fmt.Fprint(w, `{"status": "running", "version": "test"}`)
		case "/api/v1/instances":
			w.WriteHeader(http.StatusOK)
			_, _ = fmt.Fprint(w, `{"instances": []}`)
		case "/api/v1/templates":
			w.WriteHeader(http.StatusOK)
			_, _ = fmt.Fprint(w, `{}`)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	client := NewClient(server.URL)
	ctx := context.Background()

	// Test basic operations
	err := client.Ping(ctx)
	assert.NoError(t, err)

	status, err := client.GetStatus(ctx)
	assert.NoError(t, err)
	assert.NotNil(t, status)

	instances, err := client.ListInstances(ctx)
	assert.NoError(t, err)
	assert.NotNil(t, instances)

	templates, err := client.ListTemplates(ctx)
	assert.NoError(t, err)
	assert.NotNil(t, templates)
}

// TestRegistryStatusResponse tests RegistryStatusResponse JSON serialization
func TestRegistryStatusResponse(t *testing.T) {
	now := time.Now()

	response := RegistryStatusResponse{
		Active:        true,
		LastSync:      &now,
		TemplateCount: 5,
		AMICount:      15,
		Status:        "healthy",
	}

	// Test JSON marshaling
	data, err := json.Marshal(response)
	require.NoError(t, err)

	// Test JSON unmarshaling
	var unmarshaled RegistryStatusResponse
	err = json.Unmarshal(data, &unmarshaled)
	require.NoError(t, err)

	assert.Equal(t, response.Active, unmarshaled.Active)
	assert.Equal(t, response.TemplateCount, unmarshaled.TemplateCount)
	assert.Equal(t, response.AMICount, unmarshaled.AMICount)
	assert.Equal(t, response.Status, unmarshaled.Status)
	assert.NotNil(t, unmarshaled.LastSync)
	assert.WithinDuration(t, now, *unmarshaled.LastSync, time.Second)
}

// TestRegistryStatusResponseWithNilLastSync tests handling of nil LastSync
func TestRegistryStatusResponseWithNilLastSync(t *testing.T) {
	response := RegistryStatusResponse{
		Active:        false,
		LastSync:      nil,
		TemplateCount: 0,
		AMICount:      0,
		Status:        "inactive",
	}

	// Test JSON marshaling with nil LastSync
	data, err := json.Marshal(response)
	require.NoError(t, err)

	// Test JSON unmarshaling
	var unmarshaled RegistryStatusResponse
	err = json.Unmarshal(data, &unmarshaled)
	require.NoError(t, err)

	assert.Equal(t, response.Active, unmarshaled.Active)
	assert.Nil(t, unmarshaled.LastSync)
	assert.Equal(t, response.Status, unmarshaled.Status)
}

// TestAMIReferenceResponse tests AMIReferenceResponse JSON serialization
func TestAMIReferenceResponse(t *testing.T) {
	buildDate := time.Now()

	response := AMIReferenceResponse{
		AMIID:        "ami-12345678",
		Region:       "us-east-1",
		Architecture: "x86_64",
		TemplateName: "python-ml",
		Version:      "1.2.3",
		BuildDate:    buildDate,
		Status:       "available",
		Tags: map[string]string{
			"Environment": "test",
			"Project":     "cloudworkstation",
		},
	}

	// Test JSON marshaling
	data, err := json.Marshal(response)
	require.NoError(t, err)

	// Test JSON unmarshaling
	var unmarshaled AMIReferenceResponse
	err = json.Unmarshal(data, &unmarshaled)
	require.NoError(t, err)

	assert.Equal(t, response.AMIID, unmarshaled.AMIID)
	assert.Equal(t, response.Region, unmarshaled.Region)
	assert.Equal(t, response.Architecture, unmarshaled.Architecture)
	assert.Equal(t, response.TemplateName, unmarshaled.TemplateName)
	assert.Equal(t, response.Version, unmarshaled.Version)
	assert.WithinDuration(t, buildDate, unmarshaled.BuildDate, time.Second)
	assert.Equal(t, response.Status, unmarshaled.Status)
	assert.Equal(t, response.Tags, unmarshaled.Tags)
}

// TestAMIReferenceResponseWithEmptyTags tests handling of empty tags
func TestAMIReferenceResponseWithEmptyTags(t *testing.T) {
	response := AMIReferenceResponse{
		AMIID:        "ami-87654321",
		Region:       "us-west-2",
		Architecture: "arm64",
		TemplateName: "r-research",
		Version:      "2.1.0",
		BuildDate:    time.Now(),
		Status:       "pending",
		Tags:         nil,
	}

	// Test JSON marshaling with nil tags
	data, err := json.Marshal(response)
	require.NoError(t, err)

	// Verify omitempty works for tags
	var jsonMap map[string]interface{}
	err = json.Unmarshal(data, &jsonMap)
	require.NoError(t, err)

	_, tagsExist := jsonMap["tags"]
	assert.False(t, tagsExist, "Empty tags should be omitted from JSON")

	// Test JSON unmarshaling
	var unmarshaled AMIReferenceResponse
	err = json.Unmarshal(data, &unmarshaled)
	require.NoError(t, err)

	assert.Equal(t, response.AMIID, unmarshaled.AMIID)
	assert.Nil(t, unmarshaled.Tags)
}

// TestOptionsStructValidation tests Options struct field validation
func TestOptionsStructValidation(t *testing.T) {
	tests := []struct {
		name    string
		options Options
		valid   bool
	}{
		{
			name: "Valid complete options",
			options: Options{
				AWSProfile:      "default",
				AWSRegion:       "us-east-1",
				InvitationToken: "token123",
				OwnerAccount:    "123456789012",
				S3ConfigPath:    "/tmp/config",
			},
			valid: true,
		},
		{
			name: "Valid minimal options",
			options: Options{
				AWSProfile: "default",
				AWSRegion:  "us-east-1",
			},
			valid: true,
		},
		{
			name:    "Valid empty options",
			options: Options{},
			valid:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Options should always be valid for basic usage
			client := NewClient("http://localhost:8947")
			client.SetOptions(tt.options)

			// Basic validation - no panics, client created successfully
			assert.NotNil(t, client)
		})
	}
}

// TestExtendedOptionsConversion tests conversion from ExtendedOptions to Options
func TestExtendedOptionsConversion(t *testing.T) {
	extended := ExtendedOptions{
		AWSProfile:      "extended-profile",
		AWSRegion:       "eu-west-1",
		InvitationToken: "extended-token",
		OwnerAccount:    "987654321098",
		S3ConfigPath:    "/tmp/extended",
		ProfileID:       "profile-456",
	}

	client := NewClient("http://localhost:8947")
	result := ApplyExtendedClientOptions(client, extended)

	assert.NotNil(t, result)
	assert.Equal(t, client, result)

	// Verify basic options were applied (ProfileID is not part of basic Options)
	httpClient := result.(*HTTPClient)
	assert.Equal(t, extended.AWSProfile, httpClient.awsProfile)
	assert.Equal(t, extended.AWSRegion, httpClient.awsRegion)
	assert.Equal(t, extended.InvitationToken, httpClient.invitationToken)
	assert.Equal(t, extended.OwnerAccount, httpClient.ownerAccount)
	assert.Equal(t, extended.S3ConfigPath, httpClient.s3ConfigPath)
}

// TestNewClientDefaults tests NewClient default values
func TestNewClientDefaults(t *testing.T) {
	client := NewClient("")

	httpClient := client.(*HTTPClient)
	assert.Equal(t, "http://localhost:8080", httpClient.baseURL)
	assert.NotNil(t, httpClient.httpClient)
	assert.Equal(t, "", httpClient.awsProfile)
	assert.Equal(t, "", httpClient.awsRegion)
}

// TestNewClientWithOptionsInterface tests NewClientWithOptions interface
func TestNewClientWithOptionsInterface(t *testing.T) {
	options := Options{
		AWSProfile: "test-profile",
		AWSRegion:  "us-west-2",
	}

	client := NewClientWithOptions("http://test:9000", options).(*HTTPClient)

	assert.Equal(t, "http://test:9000", client.baseURL)
	assert.Equal(t, "test-profile", client.awsProfile)
	assert.Equal(t, "us-west-2", client.awsRegion)
}

// TestSetOptionsMultipleCalls tests multiple SetOptions calls
func TestSetOptionsMultipleCalls(t *testing.T) {
	client := NewClient("http://localhost:8947")
	httpClient := client.(*HTTPClient)

	// First set of options
	options1 := Options{
		AWSProfile: "profile1",
		AWSRegion:  "us-east-1",
	}
	client.SetOptions(options1)

	assert.Equal(t, "profile1", httpClient.awsProfile)
	assert.Equal(t, "us-east-1", httpClient.awsRegion)

	// Second set of options should override
	options2 := Options{
		AWSProfile:      "profile2",
		AWSRegion:       "us-west-2",
		InvitationToken: "token123",
	}
	client.SetOptions(options2)

	assert.Equal(t, "profile2", httpClient.awsProfile)
	assert.Equal(t, "us-west-2", httpClient.awsRegion)
	assert.Equal(t, "token123", httpClient.invitationToken)
}

// Integration test for critical API methods
func TestCriticalAPIMethodsIntegration(t *testing.T) {
	server := createInstanceAPITestServer()
	defer server.Close()

	client := NewClient(server.URL)
	ctx := context.Background()

	t.Run("instance_lifecycle_operations", func(t *testing.T) {
		testInstanceLifecycleOperations(t, client, ctx)
	})

	t.Run("instance_power_management", func(t *testing.T) {
		testInstancePowerManagement(t, client, ctx)
	})

	t.Run("instance_hibernation_and_connection", func(t *testing.T) {
		testInstanceHibernationAndConnection(t, client, ctx)
	})
}

// TestTemplateOperationsIntegration tests template-related API methods
func TestTemplateOperationsIntegration(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch {
		case r.URL.Path == "/api/v1/templates" && r.Method == "GET":
			w.WriteHeader(http.StatusOK)
			_, _ = fmt.Fprint(w, `{"python-ml": {"name": "python-ml", "description": "Python ML environment"}}`)
		case r.URL.Path == "/api/v1/templates/python-ml" && r.Method == "GET":
			w.WriteHeader(http.StatusOK)
			_, _ = fmt.Fprint(w, `{"name": "python-ml", "description": "Python ML environment"}`)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	client := NewClient(server.URL)
	ctx := context.Background()

	// Test basic template operations
	templatesMap, err := client.ListTemplates(ctx)
	assert.NoError(t, err)
	assert.Contains(t, templatesMap, "python-ml")

	template, err := client.GetTemplate(ctx, "python-ml")
	assert.NoError(t, err)
	assert.Equal(t, "python-ml", template.Name)
}

// TestIdleDetectionOperationsIntegration tests idle detection API methods
func TestIdleDetectionOperationsIntegration(t *testing.T) {
	server := createIdleAPITestServer()
	defer server.Close()

	client := NewClient(server.URL)
	ctx := context.Background()

	t.Run("idle_policy_management", func(t *testing.T) {
		testIdlePolicyManagement(t, client, ctx)
	})

	t.Run("idle_actions_execution", func(t *testing.T) {
		testIdleActionsExecution(t, client, ctx)
	})

	t.Run("idle_reporting_and_history", func(t *testing.T) {
		testIdleReportingAndHistory(t, client, ctx)
	})
}

// TestProjectManagementIntegration tests project management API methods
func TestProjectManagementIntegration(t *testing.T) {
	server := createProjectAPITestServer()
	defer server.Close()

	client := NewClient(server.URL)
	ctx := context.Background()

	t.Run("project_lifecycle_management", func(t *testing.T) {
		testProjectLifecycleManagement(t, client, ctx)
	})

	t.Run("project_member_management", func(t *testing.T) {
		testProjectMemberManagement(t, client, ctx)
	})

	t.Run("project_budget_operations", func(t *testing.T) {
		testProjectBudgetOperations(t, client, ctx)
	})
}

// Helper functions for idle detection testing

func createIdleAPITestServer() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		routes := getIdleAPIRoutes()
		for _, route := range routes {
			if route.matches(r) {
				w.WriteHeader(route.status)
				if route.response != "" {
					_, _ = fmt.Fprint(w, route.response)
				}
				return
			}
		}
		w.WriteHeader(http.StatusNotFound)
	}))
}

type apiRoute struct {
	method   string
	path     string
	pathFunc func(string) bool
	status   int
	response string
}

func (r apiRoute) matches(req *http.Request) bool {
	if req.Method != r.method {
		return false
	}
	if r.pathFunc != nil {
		return r.pathFunc(req.URL.Path)
	}
	return req.URL.Path == r.path
}

func getIdleAPIRoutes() []apiRoute {
	return []apiRoute{
		{"GET", "/api/v1/idle/status", nil, http.StatusOK, `{"enabled": true, "active_profiles": 2}`},
		{"POST", "/api/v1/idle/enable", nil, http.StatusOK, ""},
		{"POST", "/api/v1/idle/disable", nil, http.StatusOK, ""},
		{"GET", "/api/v1/idle/policies", nil, http.StatusOK, `[{"id": "batch", "name": "batch", "idle_minutes": 60, "action": "hibernate"}]`},
		{"GET", "", func(path string) bool { return strings.HasPrefix(path, "/api/v1/idle/policies/") }, http.StatusOK, `{"id": "batch", "name": "batch", "idle_minutes": 60, "action": "hibernate"}`},
		{"POST", "/api/v1/idle/policies/apply", nil, http.StatusOK, ""},
		{"POST", "/api/v1/idle/policies/remove", nil, http.StatusOK, ""},
		{"GET", "", func(path string) bool { return strings.HasPrefix(path, "/api/v1/instances/") && strings.HasSuffix(path, "/idle-policies") }, http.StatusOK, `[{"id": "batch", "name": "batch", "idle_minutes": 60, "action": "hibernate"}]`},
		{"GET", "/api/v1/idle/pending-actions", nil, http.StatusOK, `[{"instance_name": "test", "action": "hibernate"}]`},
		{"POST", "/api/v1/idle/execute-actions", nil, http.StatusOK, `{"executed": 1, "errors": [], "total": 1}`},
		{"GET", "/api/v1/idle/history", nil, http.StatusOK, `[{"instance_name": "test", "action": "hibernate", "timestamp": "2024-01-01T00:00:00Z"}]`},
		{"GET", "", func(path string) bool { return strings.HasPrefix(path, "/api/v1/idle/savings") }, http.StatusOK, `{"total_saved": 100.50, "period": "7d", "instances": 5}`},
	}
}

func testIdlePolicyManagement(t *testing.T, client *Client, ctx context.Context) {
	// Test list policies
	policies, err := client.ListIdlePolicies(ctx)
	assert.NoError(t, err)
	assert.NotNil(t, policies)
	assert.Greater(t, len(policies), 0)

	// Test get policy by ID
	policy, err := client.GetIdlePolicy(ctx, "batch")
	assert.NoError(t, err)
	assert.NotNil(t, policy)

	// Test policy application and removal
	policyTests := []struct {
		action   string
		testFunc func() error
	}{
		{"apply", func() error { return client.ApplyIdlePolicy(ctx, "test-instance", "batch") }},
		{"remove", func() error { return client.RemoveIdlePolicy(ctx, "test-instance", "batch") }},
	}

	for _, test := range policyTests {
		t.Run("policy_"+test.action, func(t *testing.T) {
			err := test.testFunc()
			assert.NoError(t, err)
		})
	}

	// Test get instance policies
	instancePolicies, err := client.GetInstanceIdlePolicies(ctx, "test-instance")
	assert.NoError(t, err)
	assert.NotNil(t, instancePolicies)
}

func testIdleActionsExecution(t *testing.T, client *Client, ctx context.Context) {
	// Test pending actions
	pendingActions, err := client.GetIdlePendingActions(ctx)
	assert.NoError(t, err)
	assert.Len(t, pendingActions, 1)

	// Test action execution
	execResp, err := client.ExecuteIdleActions(ctx)
	assert.NoError(t, err)
	assert.Equal(t, 1, execResp.Executed)
}

func testIdleReportingAndHistory(t *testing.T, client *Client, ctx context.Context) {
	// Test savings report
	report, err := client.GetIdleSavingsReport(ctx, "7d")
	assert.NoError(t, err)
	assert.NotNil(t, report)

	// Test action history
	history, err := client.GetIdleHistory(ctx)
	assert.NoError(t, err)
	assert.Len(t, history, 1)
}

// Helper functions for project management testing

func createProjectAPITestServer() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		routes := getProjectAPIRoutes()
		for _, route := range routes {
			if route.matches(r) {
				w.WriteHeader(route.status)
				if route.response != "" {
					_, _ = fmt.Fprint(w, route.response)
				}
				return
			}
		}
		w.WriteHeader(http.StatusNotFound)
	}))
}

func getProjectAPIRoutes() []apiRoute {
	return []apiRoute{
		{"POST", "/api/v1/projects", nil, http.StatusCreated, `{"id": "proj-123", "name": "test-project", "status": "active"}`},
		{"GET", "/api/v1/projects", nil, http.StatusOK, `{"projects": [{"id": "proj-123", "name": "test-project"}], "total": 1}`},
		{"GET", "/api/v1/projects/proj-123", nil, http.StatusOK, `{"id": "proj-123", "name": "test-project", "status": "active"}`},
		{"PUT", "/api/v1/projects/proj-123", nil, http.StatusOK, `{"id": "proj-123", "name": "updated-project", "status": "active"}`},
		{"DELETE", "/api/v1/projects/proj-123", nil, http.StatusNoContent, ""},
		{"POST", "/api/v1/projects/proj-123/members", nil, http.StatusCreated, ""},
		{"PUT", "/api/v1/projects/proj-123/members/user-456", nil, http.StatusOK, ""},
		{"DELETE", "/api/v1/projects/proj-123/members/user-456", nil, http.StatusNoContent, ""},
		{"GET", "/api/v1/projects/proj-123/members", nil, http.StatusOK, `[{"user_id": "user-456", "role": "member"}]`},
		{"GET", "/api/v1/projects/proj-123/budget", nil, http.StatusOK, `{"total_budget": 1000, "used_budget": 250}`},
	}
}

func testProjectLifecycleManagement(t *testing.T, client *Client, ctx context.Context) {
	// Create project
	createReq := project.CreateProjectRequest{
		Name:        "test-project",
		Description: "Test project",
	}
	proj, err := client.CreateProject(ctx, createReq)
	assert.NoError(t, err)
	assert.Equal(t, "test-project", proj.Name)

	// List projects
	projects, err := client.ListProjects(ctx, nil)
	assert.NoError(t, err)
	assert.Len(t, projects.Projects, 1)

	// Get project
	retrievedProj, err := client.GetProject(ctx, "proj-123")
	assert.NoError(t, err)
	assert.Equal(t, "proj-123", retrievedProj.ID)

	// Update project
	updateReq := project.UpdateProjectRequest{
		Name: func() *string { s := "updated-project"; return &s }(),
	}
	updatedProj, err := client.UpdateProject(ctx, "proj-123", updateReq)
	assert.NoError(t, err)
	assert.Equal(t, "updated-project", updatedProj.Name)

	// Delete project
	err = client.DeleteProject(ctx, "proj-123")
	assert.NoError(t, err)
}

func testProjectMemberManagement(t *testing.T, client *Client, ctx context.Context) {
	memberOperations := []struct {
		name string
		op   func() error
	}{
		{"add_member", func() error {
			return client.AddProjectMember(ctx, "proj-123", project.AddMemberRequest{
				UserID: "user-456",
				Role:   "member",
			})
		}},
		{"update_member", func() error {
			return client.UpdateProjectMember(ctx, "proj-123", "user-456", project.UpdateMemberRequest{
				Role: "admin",
			})
		}},
		{"remove_member", func() error {
			return client.RemoveProjectMember(ctx, "proj-123", "user-456")
		}},
	}

	for _, op := range memberOperations {
		t.Run(op.name, func(t *testing.T) {
			err := op.op()
			assert.NoError(t, err)
		})
	}

	// Get members
	members, err := client.GetProjectMembers(ctx, "proj-123")
	assert.NoError(t, err)
	assert.Len(t, members, 1)
}

func testProjectBudgetOperations(t *testing.T, client *Client, ctx context.Context) {
	budgetStatus, err := client.GetProjectBudgetStatus(ctx, "proj-123")
	assert.NoError(t, err)
	assert.Equal(t, float64(1000), budgetStatus.TotalBudget)
}

// Helper functions for instance management testing

func createInstanceAPITestServer() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		routes := getInstanceAPIRoutes()
		for _, route := range routes {
			if route.matches(r) {
				w.WriteHeader(route.status)
				if route.response != "" {
					_, _ = fmt.Fprint(w, route.response)
				}
				return
			}
		}
		w.WriteHeader(http.StatusNotFound)
	}))
}

func getInstanceAPIRoutes() []apiRoute {
	return []apiRoute{
		{"POST", "/api/v1/instances", nil, http.StatusCreated, `{"instance": {"name": "test-instance", "id": "i-123"}, "message": "Instance launched"}`},
		{"GET", "/api/v1/instances", nil, http.StatusOK, `{"instances": [{"id": "i-123", "name": "test-instance", "state": "running"}]}`},
		{"GET", "/api/v1/instances/test-instance", nil, http.StatusOK, `{"id": "i-123", "name": "test-instance", "state": "running"}`},
		{"POST", "/api/v1/instances/test-instance/start", nil, http.StatusOK, ""},
		{"POST", "/api/v1/instances/test-instance/stop", nil, http.StatusOK, ""},
		{"POST", "/api/v1/instances/test-instance/hibernate", nil, http.StatusOK, ""},
		{"POST", "/api/v1/instances/test-instance/resume", nil, http.StatusOK, ""},
		{"GET", "/api/v1/instances/test-instance/hibernation-status", nil, http.StatusOK, `{"hibernation_supported": true, "is_hibernated": false, "instance_name": "test-instance"}`},
		{"DELETE", "/api/v1/instances/test-instance", nil, http.StatusNoContent, ""},
		{"GET", "/api/v1/instances/test-instance/connect", nil, http.StatusOK, `{"connection_info": "ssh user@1.2.3.4"}`},
	}
}

func testInstanceLifecycleOperations(t *testing.T, client *Client, ctx context.Context) {
	// Launch instance
	launchReq := types.LaunchRequest{
		Name:     "test-instance",
		Template: "python-ml",
	}
	launchResp, err := client.LaunchInstance(ctx, launchReq)
	assert.NoError(t, err)
	assert.Equal(t, "test-instance", launchResp.Instance.Name)

	// List instances
	listResp, err := client.ListInstances(ctx)
	assert.NoError(t, err)
	assert.NotNil(t, listResp)

	// Get instance
	instance, err := client.GetInstance(ctx, "test-instance")
	assert.NoError(t, err)
	assert.Equal(t, "test-instance", instance.Name)

	// Delete instance
	err = client.DeleteInstance(ctx, "test-instance")
	assert.NoError(t, err)
}

func testInstancePowerManagement(t *testing.T, client *Client, ctx context.Context) {
	powerOperations := []struct {
		name string
		op   func() error
	}{
		{"start", func() error { return client.StartInstance(ctx, "test-instance") }},
		{"stop", func() error { return client.StopInstance(ctx, "test-instance") }},
	}

	for _, op := range powerOperations {
		t.Run(op.name, func(t *testing.T) {
			err := op.op()
			assert.NoError(t, err)
		})
	}
}

func testInstanceHibernationAndConnection(t *testing.T, client *Client, ctx context.Context) {
	// Test hibernation operations
	err := client.HibernateInstance(ctx, "test-instance")
	assert.NoError(t, err)

	err = client.ResumeInstance(ctx, "test-instance")
	assert.NoError(t, err)

	// Test hibernation status
	hibStatus, err := client.GetInstanceHibernationStatus(ctx, "test-instance")
	assert.NoError(t, err)
	assert.True(t, hibStatus.HibernationSupported)

	// Test connection
	connInfo, err := client.ConnectInstance(ctx, "test-instance")
	assert.NoError(t, err)
	assert.Equal(t, "ssh user@1.2.3.4", connInfo)
}
