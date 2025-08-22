package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// TestCloudWorkstationService tests the CloudWorkstation service
func TestCloudWorkstationService(t *testing.T) {
	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/templates":
			json.NewEncoder(w).Encode([]Template{
				{Name: "python-ml", Description: "Python ML Environment"},
				{Name: "r-research", Description: "R Research Environment"},
			})
		case "/api/v1/instances":
			json.NewEncoder(w).Encode([]Instance{
				{Name: "test-instance", State: "running", IP: "1.2.3.4"},
			})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	// Create service with test server
	service := &CloudWorkstationService{
		daemonURL: server.URL,
		client:    &http.Client{Timeout: 5 * time.Second},
	}

	// Test GetTemplates
	templates, err := service.GetTemplates(context.Background())
	if err != nil {
		t.Fatalf("GetTemplates failed: %v", err)
	}
	if len(templates) != 2 {
		t.Errorf("Expected 2 templates, got %d", len(templates))
	}

	// Test GetInstances
	instances, err := service.GetInstances(context.Background())
	if err != nil {
		t.Fatalf("GetInstances failed: %v", err)
	}
	if len(instances) != 1 {
		t.Errorf("Expected 1 instance, got %d", len(instances))
	}
}

// TestLaunchInstance tests instance launching
func TestLaunchInstance(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("Expected POST, got %s", r.Method)
		}
		
		var req LaunchRequest
		json.NewDecoder(r.Body).Decode(&req)
		
		if req.Template != "python-ml" || req.Name != "test" {
			t.Errorf("Unexpected launch request: %+v", req)
		}
		
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "success"})
	}))
	defer server.Close()

	service := &CloudWorkstationService{
		daemonURL: server.URL,
		client:    &http.Client{Timeout: 5 * time.Second},
	}

	err := service.LaunchInstance(context.Background(), LaunchRequest{
		Template: "python-ml",
		Name:     "test",
		Size:     "medium",
	})
	
	if err != nil {
		t.Fatalf("LaunchInstance failed: %v", err)
	}
}

// TestInstanceActions tests instance control actions
func TestInstanceActions(t *testing.T) {
	actionCalled := ""
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/instances/test/stop":
			actionCalled = "stop"
			w.WriteHeader(http.StatusOK)
		case "/api/v1/instances/test/start":
			actionCalled = "start"
			w.WriteHeader(http.StatusOK)
		case "/api/v1/instances/test/terminate":
			actionCalled = "terminate"
			w.WriteHeader(http.StatusOK)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	service := &CloudWorkstationService{
		daemonURL: server.URL,
		client:    &http.Client{Timeout: 5 * time.Second},
	}

	// Test stop
	err := service.StopInstance(context.Background(), "test")
	if err != nil {
		t.Errorf("StopInstance failed: %v", err)
	}
	if actionCalled != "stop" {
		t.Errorf("Expected stop action, got %s", actionCalled)
	}

	// Test start
	err = service.StartInstance(context.Background(), "test")
	if err != nil {
		t.Errorf("StartInstance failed: %v", err)
	}
	if actionCalled != "start" {
		t.Errorf("Expected start action, got %s", actionCalled)
	}

	// Test terminate
	err = service.TerminateInstance(context.Background(), "test")
	if err != nil {
		t.Errorf("TerminateInstance failed: %v", err)
	}
	if actionCalled != "terminate" {
		t.Errorf("Expected terminate action, got %s", actionCalled)
	}
}

// TestGetInstanceAccess tests instance access information retrieval
func TestGetInstanceAccess(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id":                "i-123",
			"name":              "test",
			"public_ip":         "1.2.3.4",
			"has_web_interface": true,
			"web_port":          8888,
			"ports":             []int{22, 3389, 8888},
			"username":          "ubuntu",
		})
	}))
	defer server.Close()

	service := &CloudWorkstationService{
		daemonURL: server.URL,
		client:    &http.Client{Timeout: 5 * time.Second},
	}

	access, err := service.GetInstanceAccess(context.Background(), "test")
	if err != nil {
		t.Fatalf("GetInstanceAccess failed: %v", err)
	}

	// Verify access information
	if access.PublicIP != "1.2.3.4" {
		t.Errorf("Expected IP 1.2.3.4, got %s", access.PublicIP)
	}
	if access.WebPort != 8888 {
		t.Errorf("Expected web port 8888, got %d", access.WebPort)
	}
	if access.RDPPort != 3389 {
		t.Errorf("Expected RDP port 3389, got %d", access.RDPPort)
	}
	
	// Check access types
	hasDesktop := false
	hasWeb := false
	hasTerminal := false
	for _, at := range access.AccessTypes {
		switch at {
		case AccessTypeDesktop:
			hasDesktop = true
		case AccessTypeWeb:
			hasWeb = true
		case AccessTypeTerminal:
			hasTerminal = true
		}
	}
	
	if !hasDesktop || !hasWeb || !hasTerminal {
		t.Errorf("Missing access types: desktop=%v, web=%v, terminal=%v", 
			hasDesktop, hasWeb, hasTerminal)
	}
}

// TestConnectionInfo tests connection info retrieval
func TestConnectionInfo(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(ConnectionInfo{
			InstanceName: "test",
			HasDesktop:   true,
			HasDisplay:   true,
			Services:     []string{"jupyter", "rstudio"},
			Ports:        []int{22, 8888, 8787},
		})
	}))
	defer server.Close()

	service := &CloudWorkstationService{
		daemonURL: server.URL,
		client:    &http.Client{Timeout: 5 * time.Second},
	}

	info, err := service.GetConnectionInfo(context.Background(), "test")
	if err != nil {
		t.Fatalf("GetConnectionInfo failed: %v", err)
	}

	if !info.HasDesktop {
		t.Error("Expected HasDesktop to be true")
	}
	if len(info.Services) != 2 {
		t.Errorf("Expected 2 services, got %d", len(info.Services))
	}
	if len(info.Ports) != 3 {
		t.Errorf("Expected 3 ports, got %d", len(info.Ports))
	}
}

// TestTemplateHelpers tests template helper functions
func TestTemplateHelpers(t *testing.T) {
	tests := []struct {
		name     string
		template string
		wantIcon string
		wantCat  string
	}{
		{"Python ML", "python-ml", "🐍", "Machine Learning"},
		{"R Research", "r-research", "📊", "Data Science"},
		{"Ubuntu", "ubuntu", "🐧", "General"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			icon := getTemplateIcon(tt.template)
			if icon != tt.wantIcon {
				t.Errorf("getTemplateIcon() = %v, want %v", icon, tt.wantIcon)
			}

			cat := getTemplateCategory(tt.template)
			if cat != tt.wantCat {
				t.Errorf("getTemplateCategory() = %v, want %v", cat, tt.wantCat)
			}
		})
	}
}

// TestEmbeddedWebView tests embedded web view configuration
func TestEmbeddedWebView(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"name":              "test",
			"has_web_interface": true,
			"web_port":          8888,
		})
	}))
	defer server.Close()

	service := &CloudWorkstationService{
		daemonURL: server.URL,
		client:    &http.Client{Timeout: 5 * time.Second},
	}

	webView, err := service.CreateEmbeddedWebView(context.Background(), "test")
	if err != nil {
		t.Fatalf("CreateEmbeddedWebView failed: %v", err)
	}

	if !strings.Contains(webView.URL, "/proxy/test") {
		t.Errorf("Expected proxy URL, got %s", webView.URL)
	}
	if webView.Width != 1200 || webView.Height != 800 {
		t.Errorf("Unexpected dimensions: %dx%d", webView.Width, webView.Height)
	}
}

// TestErrorHandling tests error handling
func TestErrorHandling(t *testing.T) {
	// Test server that always returns errors
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "test error"})
	}))
	defer server.Close()

	service := &CloudWorkstationService{
		daemonURL: server.URL,
		client:    &http.Client{Timeout: 5 * time.Second},
	}

	// Test that methods handle errors gracefully
	_, err := service.GetTemplates(context.Background())
	if err == nil {
		t.Error("Expected error from GetTemplates")
	}

	_, err = service.GetInstances(context.Background())
	if err == nil {
		t.Error("Expected error from GetInstances")
	}

	err = service.LaunchInstance(context.Background(), LaunchRequest{})
	if err == nil {
		t.Error("Expected error from LaunchInstance")
	}
}

// TestTimeout tests request timeout handling
func TestTimeout(t *testing.T) {
	// Server that delays response
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	service := &CloudWorkstationService{
		daemonURL: server.URL,
		client:    &http.Client{Timeout: 100 * time.Millisecond},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	_, err := service.GetTemplates(ctx)
	if err == nil {
		t.Error("Expected timeout error")
	}
}

// BenchmarkGetTemplates benchmarks template fetching
func BenchmarkGetTemplates(b *testing.B) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		templates := make([]Template, 100)
		for i := 0; i < 100; i++ {
			templates[i] = Template{
				Name:        fmt.Sprintf("template-%d", i),
				Description: fmt.Sprintf("Description %d", i),
			}
		}
		json.NewEncoder(w).Encode(templates)
	}))
	defer server.Close()

	service := &CloudWorkstationService{
		daemonURL: server.URL,
		client:    &http.Client{Timeout: 5 * time.Second},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = service.GetTemplates(context.Background())
	}
}

// TestConcurrentRequests tests concurrent API requests
func TestConcurrentRequests(t *testing.T) {
	requestCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		json.NewEncoder(w).Encode([]Template{})
	}))
	defer server.Close()

	service := &CloudWorkstationService{
		daemonURL: server.URL,
		client:    &http.Client{Timeout: 5 * time.Second},
	}

	// Make concurrent requests
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func() {
			service.GetTemplates(context.Background())
			done <- true
		}()
	}

	// Wait for all requests
	for i := 0; i < 10; i++ {
		<-done
	}

	if requestCount != 10 {
		t.Errorf("Expected 10 requests, got %d", requestCount)
	}
}