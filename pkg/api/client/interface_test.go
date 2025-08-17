package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
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
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch {
		case r.URL.Path == "/api/v1/instances" && r.Method == "POST":
			// Launch instance
			w.WriteHeader(http.StatusCreated)
			_, _ = fmt.Fprint(w, `{"instance": {"name": "test-instance", "id": "i-123"}, "message": "Instance launched"}`)
		case r.URL.Path == "/api/v1/instances" && r.Method == "GET":
			// List instances
			w.WriteHeader(http.StatusOK)
			_, _ = fmt.Fprint(w, `{"instances": [{"id": "i-123", "name": "test-instance", "state": "running"}]}`)
		case r.URL.Path == "/api/v1/instances/test-instance" && r.Method == "GET":
			// Get instance
			w.WriteHeader(http.StatusOK)
			_, _ = fmt.Fprint(w, `{"id": "i-123", "name": "test-instance", "state": "running"}`)
		case r.URL.Path == "/api/v1/instances/test-instance/start" && r.Method == "POST":
			// Start instance
			w.WriteHeader(http.StatusOK)
		case r.URL.Path == "/api/v1/instances/test-instance/stop" && r.Method == "POST":
			// Stop instance
			w.WriteHeader(http.StatusOK)
		case r.URL.Path == "/api/v1/instances/test-instance/hibernate" && r.Method == "POST":
			// Hibernate instance
			w.WriteHeader(http.StatusOK)
		case r.URL.Path == "/api/v1/instances/test-instance/resume" && r.Method == "POST":
			// Resume instance
			w.WriteHeader(http.StatusOK)
		case r.URL.Path == "/api/v1/instances/test-instance/hibernation-status" && r.Method == "GET":
			// Get hibernation status
			w.WriteHeader(http.StatusOK)
			_, _ = fmt.Fprint(w, `{"hibernation_supported": true, "is_hibernated": false, "instance_name": "test-instance"}`)
		case r.URL.Path == "/api/v1/instances/test-instance" && r.Method == "DELETE":
			// Delete instance
			w.WriteHeader(http.StatusNoContent)
		case r.URL.Path == "/api/v1/instances/test-instance/connect" && r.Method == "GET":
			// Connect instance
			w.WriteHeader(http.StatusOK)
			_, _ = fmt.Fprint(w, `{"connection_info": "ssh user@1.2.3.4"}`)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	client := NewClient(server.URL)
	ctx := context.Background()

	// Test instance operations
	launchReq := types.LaunchRequest{
		Name:     "test-instance",
		Template: "python-ml",
	}
	launchResp, err := client.LaunchInstance(ctx, launchReq)
	assert.NoError(t, err)
	assert.Equal(t, "test-instance", launchResp.Instance.Name)

	listResp, err := client.ListInstances(ctx)
	assert.NoError(t, err)
	assert.NotNil(t, listResp)

	instance, err := client.GetInstance(ctx, "test-instance")
	assert.NoError(t, err)
	assert.Equal(t, "test-instance", instance.Name)

	err = client.StartInstance(ctx, "test-instance")
	assert.NoError(t, err)

	err = client.StopInstance(ctx, "test-instance")
	assert.NoError(t, err)

	err = client.HibernateInstance(ctx, "test-instance")
	assert.NoError(t, err)

	err = client.ResumeInstance(ctx, "test-instance")
	assert.NoError(t, err)

	hibStatus, err := client.GetInstanceHibernationStatus(ctx, "test-instance")
	assert.NoError(t, err)
	assert.True(t, hibStatus.HibernationSupported)

	connInfo, err := client.ConnectInstance(ctx, "test-instance")
	assert.NoError(t, err)
	assert.Equal(t, "ssh user@1.2.3.4", connInfo)

	err = client.DeleteInstance(ctx, "test-instance")
	assert.NoError(t, err)
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
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch {
		case r.URL.Path == "/api/v1/idle/status" && r.Method == "GET":
			w.WriteHeader(http.StatusOK)
			_, _ = fmt.Fprint(w, `{"enabled": true, "active_profiles": 2}`)
		case r.URL.Path == "/api/v1/idle/enable" && r.Method == "POST":
			w.WriteHeader(http.StatusOK)
		case r.URL.Path == "/api/v1/idle/disable" && r.Method == "POST":
			w.WriteHeader(http.StatusOK)
		case r.URL.Path == "/api/v1/idle/profiles" && r.Method == "GET":
			w.WriteHeader(http.StatusOK)
			_, _ = fmt.Fprint(w, `{"batch": {"name": "batch", "idle_minutes": 60}}`)
		case r.URL.Path == "/api/v1/idle/profiles" && r.Method == "POST":
			w.WriteHeader(http.StatusCreated)
		case r.URL.Path == "/api/v1/idle/pending-actions" && r.Method == "GET":
			w.WriteHeader(http.StatusOK)
			_, _ = fmt.Fprint(w, `[{"instance_name": "test", "action": "hibernate"}]`)
		case r.URL.Path == "/api/v1/idle/execute-actions" && r.Method == "POST":
			w.WriteHeader(http.StatusOK)
			_, _ = fmt.Fprint(w, `{"executed": 1, "errors": [], "total": 1}`)
		case r.URL.Path == "/api/v1/idle/history" && r.Method == "GET":
			w.WriteHeader(http.StatusOK)
			_, _ = fmt.Fprint(w, `[{"instance_name": "test", "action": "hibernate", "timestamp": "2024-01-01T00:00:00Z"}]`)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	client := NewClient(server.URL)
	ctx := context.Background()

	// Test idle detection operations
	status, err := client.GetIdleStatus(ctx)
	assert.NoError(t, err)
	assert.True(t, status.Enabled)

	err = client.EnableIdleDetection(ctx)
	assert.NoError(t, err)

	err = client.DisableIdleDetection(ctx)
	assert.NoError(t, err)

	profiles, err := client.GetIdleProfiles(ctx)
	assert.NoError(t, err)
	assert.Contains(t, profiles, "batch")

	profile := types.IdleProfile{
		Name:        "test-profile",
		IdleMinutes: 30,
		Action:      "hibernate",
	}
	err = client.AddIdleProfile(ctx, profile)
	assert.NoError(t, err)

	pendingActions, err := client.GetIdlePendingActions(ctx)
	assert.NoError(t, err)
	assert.Len(t, pendingActions, 1)

	execResp, err := client.ExecuteIdleActions(ctx)
	assert.NoError(t, err)
	assert.Equal(t, 1, execResp.Executed)

	history, err := client.GetIdleHistory(ctx)
	assert.NoError(t, err)
	assert.Len(t, history, 1)
}

// TestProjectManagementIntegration tests project management API methods
func TestProjectManagementIntegration(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch {
		case r.URL.Path == "/api/v1/projects" && r.Method == "POST":
			w.WriteHeader(http.StatusCreated)
			_, _ = fmt.Fprint(w, `{"id": "proj-123", "name": "test-project", "status": "active"}`)
		case r.URL.Path == "/api/v1/projects" && r.Method == "GET":
			w.WriteHeader(http.StatusOK)
			_, _ = fmt.Fprint(w, `{"projects": [{"id": "proj-123", "name": "test-project"}], "total": 1}`)
		case r.URL.Path == "/api/v1/projects/proj-123" && r.Method == "GET":
			w.WriteHeader(http.StatusOK)
			_, _ = fmt.Fprint(w, `{"id": "proj-123", "name": "test-project", "status": "active"}`)
		case r.URL.Path == "/api/v1/projects/proj-123" && r.Method == "PUT":
			w.WriteHeader(http.StatusOK)
			_, _ = fmt.Fprint(w, `{"id": "proj-123", "name": "updated-project", "status": "active"}`)
		case r.URL.Path == "/api/v1/projects/proj-123" && r.Method == "DELETE":
			w.WriteHeader(http.StatusNoContent)
		case r.URL.Path == "/api/v1/projects/proj-123/members" && r.Method == "POST":
			w.WriteHeader(http.StatusCreated)
		case r.URL.Path == "/api/v1/projects/proj-123/members/user-456" && r.Method == "PUT":
			w.WriteHeader(http.StatusOK)
		case r.URL.Path == "/api/v1/projects/proj-123/members/user-456" && r.Method == "DELETE":
			w.WriteHeader(http.StatusNoContent)
		case r.URL.Path == "/api/v1/projects/proj-123/members" && r.Method == "GET":
			w.WriteHeader(http.StatusOK)
			_, _ = fmt.Fprint(w, `[{"user_id": "user-456", "role": "member"}]`)
		case r.URL.Path == "/api/v1/projects/proj-123/budget" && r.Method == "GET":
			w.WriteHeader(http.StatusOK)
			_, _ = fmt.Fprint(w, `{"total_budget": 1000, "used_budget": 250}`)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	client := NewClient(server.URL)
	ctx := context.Background()

	// Test project operations
	createReq := project.CreateProjectRequest{
		Name:        "test-project",
		Description: "Test project",
	}
	proj, err := client.CreateProject(ctx, createReq)
	assert.NoError(t, err)
	assert.Equal(t, "test-project", proj.Name)

	projects, err := client.ListProjects(ctx, nil)
	assert.NoError(t, err)
	assert.Len(t, projects.Projects, 1)

	retrievedProj, err := client.GetProject(ctx, "proj-123")
	assert.NoError(t, err)
	assert.Equal(t, "proj-123", retrievedProj.ID)

	updateReq := project.UpdateProjectRequest{
		Name: func() *string { s := "updated-project"; return &s }(),
	}
	updatedProj, err := client.UpdateProject(ctx, "proj-123", updateReq)
	assert.NoError(t, err)
	assert.Equal(t, "updated-project", updatedProj.Name)

	err = client.DeleteProject(ctx, "proj-123")
	assert.NoError(t, err)

	// Test member operations
	addMemberReq := project.AddMemberRequest{
		UserID: "user-456",
		Role:   "member",
	}
	err = client.AddProjectMember(ctx, "proj-123", addMemberReq)
	assert.NoError(t, err)

	updateMemberReq := project.UpdateMemberRequest{
		Role: "admin",
	}
	err = client.UpdateProjectMember(ctx, "proj-123", "user-456", updateMemberReq)
	assert.NoError(t, err)

	err = client.RemoveProjectMember(ctx, "proj-123", "user-456")
	assert.NoError(t, err)

	members, err := client.GetProjectMembers(ctx, "proj-123")
	assert.NoError(t, err)
	assert.Len(t, members, 1)

	budgetStatus, err := client.GetProjectBudgetStatus(ctx, "proj-123")
	assert.NoError(t, err)
	assert.Equal(t, float64(1000), budgetStatus.TotalBudget)
}
