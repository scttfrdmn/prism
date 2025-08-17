package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/scttfrdmn/cloudworkstation/pkg/types"
)

func TestNewClient(t *testing.T) {
	client := NewClient("")
	if client.baseURL != "http://localhost:8080" {
		t.Errorf("Expected default baseURL to be http://localhost:8080, got %s", client.baseURL)
	}

	client = NewClient("http://example.com:9000")
	if client.baseURL != "http://example.com:9000" {
		t.Errorf("Expected baseURL to be http://example.com:9000, got %s", client.baseURL)
	}
}

func TestPing(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/ping" {
			t.Errorf("Expected path /api/v1/ping, got %s", r.URL.Path)
		}
		if r.Method != http.MethodGet {
			t.Errorf("Expected GET method, got %s", r.Method)
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	}))
	defer server.Close()

	client := NewClient(server.URL)
	err := client.Ping()
	if err != nil {
		t.Errorf("Ping should succeed: %v", err)
	}
}

func TestGetStatus(t *testing.T) {
	expectedStatus := types.DaemonStatus{
		Version:       "0.1.0",
		Status:        "running",
		ActiveOps:     0,
		TotalRequests: 42,
		AWSRegion:     "us-east-1",
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/status" {
			t.Errorf("Expected path /api/v1/status, got %s", r.URL.Path)
		}
		if r.Method != http.MethodGet {
			t.Errorf("Expected GET method, got %s", r.Method)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(expectedStatus)
	}))
	defer server.Close()

	client := NewClient(server.URL)
	status, err := client.GetStatus()
	if err != nil {
		t.Fatalf("GetStatus failed: %v", err)
	}

	if status.Version != expectedStatus.Version {
		t.Errorf("Version mismatch: got %s, want %s", status.Version, expectedStatus.Version)
	}
	if status.Status != expectedStatus.Status {
		t.Errorf("Status mismatch: got %s, want %s", status.Status, expectedStatus.Status)
	}
	if status.TotalRequests != expectedStatus.TotalRequests {
		t.Errorf("TotalRequests mismatch: got %d, want %d", status.TotalRequests, expectedStatus.TotalRequests)
	}
}

func TestListInstances(t *testing.T) {
	expectedResponse := types.ListResponse{
		Instances: []types.Instance{
			{
				ID:       "i-123",
				Name:     "test-instance",
				Template: "r-research",
				State:    "running",
			},
		},
		TotalCost: 2.40,
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/instances" {
			t.Errorf("Expected path /api/v1/instances, got %s", r.URL.Path)
		}
		if r.Method != http.MethodGet {
			t.Errorf("Expected GET method, got %s", r.Method)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(expectedResponse)
	}))
	defer server.Close()

	client := NewClient(server.URL)
	response, err := client.ListInstances()
	if err != nil {
		t.Fatalf("ListInstances failed: %v", err)
	}

	if len(response.Instances) != 1 {
		t.Errorf("Expected 1 instance, got %d", len(response.Instances))
	}
	if response.TotalCost != 2.40 {
		t.Errorf("TotalCost mismatch: got %f, want %f", response.TotalCost, 2.40)
	}
	if response.Instances[0].Name != "test-instance" {
		t.Errorf("Instance name mismatch: got %s, want test-instance", response.Instances[0].Name)
	}
}

func TestLaunchInstance(t *testing.T) {
	request := types.LaunchRequest{
		Template: "r-research",
		Name:     "my-instance",
		Size:     "M",
	}

	expectedResponse := types.LaunchResponse{
		Instance: types.Instance{
			ID:       "i-123",
			Name:     "my-instance",
			Template: "r-research",
			State:    "pending",
		},
		Message:        "Instance launched successfully",
		EstimatedCost:  "$2.40/day",
		ConnectionInfo: "ssh ubuntu@54.123.45.67",
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/instances" {
			t.Errorf("Expected path /api/v1/instances, got %s", r.URL.Path)
		}
		if r.Method != http.MethodPost {
			t.Errorf("Expected POST method, got %s", r.Method)
		}

		// Verify request body
		var receivedRequest types.LaunchRequest
		err := json.NewDecoder(r.Body).Decode(&receivedRequest)
		if err != nil {
			t.Errorf("Failed to decode request body: %v", err)
		}
		if receivedRequest.Template != request.Template {
			t.Errorf("Template mismatch: got %s, want %s", receivedRequest.Template, request.Template)
		}
		if receivedRequest.Name != request.Name {
			t.Errorf("Name mismatch: got %s, want %s", receivedRequest.Name, request.Name)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(expectedResponse)
	}))
	defer server.Close()

	client := NewClient(server.URL)
	response, err := client.LaunchInstance(request)
	if err != nil {
		t.Fatalf("LaunchInstance failed: %v", err)
	}

	if response.Instance.Name != "my-instance" {
		t.Errorf("Instance name mismatch: got %s, want my-instance", response.Instance.Name)
	}
	if response.Message != "Instance launched successfully" {
		t.Errorf("Message mismatch: got %s", response.Message)
	}
}

func TestGetInstance(t *testing.T) {
	expectedInstance := types.Instance{
		ID:       "i-123",
		Name:     "test-instance",
		Template: "python-research",
		State:    "running",
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/instances/test-instance" {
			t.Errorf("Expected path /api/v1/instances/test-instance, got %s", r.URL.Path)
		}
		if r.Method != http.MethodGet {
			t.Errorf("Expected GET method, got %s", r.Method)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(expectedInstance)
	}))
	defer server.Close()

	client := NewClient(server.URL)
	instance, err := client.GetInstance("test-instance")
	if err != nil {
		t.Fatalf("GetInstance failed: %v", err)
	}

	if instance.Name != "test-instance" {
		t.Errorf("Instance name mismatch: got %s, want test-instance", instance.Name)
	}
	if instance.Template != "python-research" {
		t.Errorf("Template mismatch: got %s, want python-research", instance.Template)
	}
}

func TestDeleteInstance(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/instances/test-instance" {
			t.Errorf("Expected path /api/v1/instances/test-instance, got %s", r.URL.Path)
		}
		if r.Method != http.MethodDelete {
			t.Errorf("Expected DELETE method, got %s", r.Method)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	client := NewClient(server.URL)
	err := client.DeleteInstance("test-instance")
	if err != nil {
		t.Errorf("DeleteInstance should succeed: %v", err)
	}
}

func TestAPIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(types.APIError{
			Code:    404,
			Message: "Instance not found",
			Details: "The specified instance does not exist",
		})
	}))
	defer server.Close()

	client := NewClient(server.URL)
	_, err := client.GetInstance("nonexistent")
	if err == nil {
		t.Error("Expected error for nonexistent instance")
	}

	apiErr, ok := err.(types.APIError)
	if !ok {
		t.Errorf("Expected APIError, got %T", err)
	} else {
		if apiErr.Code != 404 {
			t.Errorf("Expected code 404, got %d", apiErr.Code)
		}
		if apiErr.Message != "Instance not found" {
			t.Errorf("Expected message 'Instance not found', got %s", apiErr.Message)
		}
	}
}

func TestListTemplates(t *testing.T) {
	expectedTemplates := map[string]types.Template{
		"r-research": {
			Name:        "R Research Environment",
			Description: "R + RStudio Server + tidyverse packages",
		},
		"python-research": {
			Name:        "Python Research Environment",
			Description: "Python + Jupyter + data science packages",
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/templates" {
			t.Errorf("Expected path /api/v1/templates, got %s", r.URL.Path)
		}
		if r.Method != http.MethodGet {
			t.Errorf("Expected GET method, got %s", r.Method)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(expectedTemplates)
	}))
	defer server.Close()

	client := NewClient(server.URL)
	templates, err := client.ListTemplates()
	if err != nil {
		t.Fatalf("ListTemplates failed: %v", err)
	}

	if len(templates) != 2 {
		t.Errorf("Expected 2 templates, got %d", len(templates))
	}
	if templates["r-research"].Name != "R Research Environment" {
		t.Errorf("Template name mismatch: got %s", templates["r-research"].Name)
	}
}

func TestCreateVolume(t *testing.T) {
	request := types.VolumeCreateRequest{
		Name:            "test-volume",
		PerformanceMode: "generalPurpose",
		ThroughputMode:  "bursting",
		Region:          "us-east-1",
	}

	expectedVolume := types.EFSVolume{
		Name:            "test-volume",
		FileSystemId:    "fs-123",
		State:           "creating",
		PerformanceMode: "generalPurpose",
		ThroughputMode:  "bursting",
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/volumes" {
			t.Errorf("Expected path /api/v1/volumes, got %s", r.URL.Path)
		}
		if r.Method != http.MethodPost {
			t.Errorf("Expected POST method, got %s", r.Method)
		}

		var receivedRequest types.VolumeCreateRequest
		err := json.NewDecoder(r.Body).Decode(&receivedRequest)
		if err != nil {
			t.Errorf("Failed to decode request body: %v", err)
		}
		if receivedRequest.Name != request.Name {
			t.Errorf("Name mismatch: got %s, want %s", receivedRequest.Name, request.Name)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(expectedVolume)
	}))
	defer server.Close()

	client := NewClient(server.URL)
	volume, err := client.CreateVolume(request)
	if err != nil {
		t.Fatalf("CreateVolume failed: %v", err)
	}

	if volume.Name != "test-volume" {
		t.Errorf("Volume name mismatch: got %s, want test-volume", volume.Name)
	}
	if volume.FileSystemId != "fs-123" {
		t.Errorf("FileSystemId mismatch: got %s, want fs-123", volume.FileSystemId)
	}
}
