package client

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/scttfrdmn/cloudworkstation/pkg/types"
)

// TestNewClient tests HTTP client creation
func TestNewClient(t *testing.T) {
	baseURL := "http://localhost:8947"
	client := NewClient(baseURL).(*HTTPClient)

	assert.NotNil(t, client)
	assert.Equal(t, baseURL, client.baseURL)
	assert.NotNil(t, client.httpClient)
	assert.Equal(t, 30*time.Second, client.httpClient.Timeout)
}

// TestNewClientWithOptions tests client creation with options
func TestNewClientWithOptions(t *testing.T) {
	baseURL := "http://localhost:8947"
	options := Options{
		AWSProfile: "test-profile",
		AWSRegion:  "us-east-1",
	}

	client := NewClientWithOptions(baseURL, options).(*HTTPClient)

	assert.NotNil(t, client)
	assert.Equal(t, baseURL, client.baseURL)
	assert.Equal(t, "test-profile", client.awsProfile)
	assert.Equal(t, "us-east-1", client.awsRegion)
}

// TestNewClientEmptyURL tests client creation with empty URL
func TestNewClientEmptyURL(t *testing.T) {
	client := NewClient("").(*HTTPClient)

	assert.NotNil(t, client)
	assert.Equal(t, "http://localhost:8080", client.baseURL)
}

// TestPing tests the Ping method
func TestPing(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "GET", r.Method)
		assert.Equal(t, "/api/v1/ping", r.URL.Path)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewClient(server.URL)
	err := client.Ping(context.Background())

	assert.NoError(t, err)
}

// TestGetStatus tests the GetStatus method
func TestGetStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "GET", r.Method)
		assert.Equal(t, "/api/v1/status", r.URL.Path)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprint(w, `{"status": "running", "version": "1.0.0"}`)
	}))
	defer server.Close()

	client := NewClient(server.URL)
	status, err := client.GetStatus(context.Background())

	assert.NoError(t, err)
	assert.NotNil(t, status)
	assert.Equal(t, "running", status.Status)
}

// TestLaunchInstance tests the LaunchInstance method
func TestLaunchInstance(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "/api/v1/instances", r.URL.Path)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_, _ = fmt.Fprint(w, `{"instance": {"name": "test-instance", "id": "i-123"}, "message": "Instance launched"}`)
	}))
	defer server.Close()

	client := NewClient(server.URL)
	req := types.LaunchRequest{
		Name:     "test-instance",
		Template: "python-ml",
	}

	resp, err := client.LaunchInstance(context.Background(), req)

	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, "test-instance", resp.Instance.Name)
}

// TestListInstances tests the ListInstances method
func TestListInstances(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "GET", r.Method)
		assert.Equal(t, "/api/v1/instances", r.URL.Path)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprint(w, `{"instances": [{"id": "i-123", "name": "test-instance", "state": "running"}]}`)
	}))
	defer server.Close()

	client := NewClient(server.URL)
	resp, err := client.ListInstances(context.Background())

	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Len(t, resp.Instances, 1)
}

// TestGetInstance tests the GetInstance method
func TestGetInstance(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "GET", r.Method)
		assert.Equal(t, "/api/v1/instances/test-instance", r.URL.Path)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprint(w, `{"id": "i-123", "name": "test-instance", "state": "running"}`)
	}))
	defer server.Close()

	client := NewClient(server.URL)
	instance, err := client.GetInstance(context.Background(), "test-instance")

	assert.NoError(t, err)
	assert.NotNil(t, instance)
	assert.Equal(t, "test-instance", instance.Name)
}

// TestInstanceControlOperations tests start/stop/hibernate/resume operations
func TestInstanceControlOperations(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)

		switch {
		case strings.Contains(r.URL.Path, "/start"):
			assert.Equal(t, "/api/v1/instances/test-instance/start", r.URL.Path)
		case strings.Contains(r.URL.Path, "/stop"):
			assert.Equal(t, "/api/v1/instances/test-instance/stop", r.URL.Path)
		case strings.Contains(r.URL.Path, "/hibernate"):
			assert.Equal(t, "/api/v1/instances/test-instance/hibernate", r.URL.Path)
		case strings.Contains(r.URL.Path, "/resume"):
			assert.Equal(t, "/api/v1/instances/test-instance/resume", r.URL.Path)
		}

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewClient(server.URL)
	ctx := context.Background()

	// Test all control operations
	err := client.StartInstance(ctx, "test-instance")
	assert.NoError(t, err)

	err = client.StopInstance(ctx, "test-instance")
	assert.NoError(t, err)

	err = client.HibernateInstance(ctx, "test-instance")
	assert.NoError(t, err)

	err = client.ResumeInstance(ctx, "test-instance")
	assert.NoError(t, err)
}

// TestGetInstanceHibernationStatus tests hibernation status checking
func TestGetInstanceHibernationStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "GET", r.Method)
		assert.Equal(t, "/api/v1/instances/test-instance/hibernation-status", r.URL.Path)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprint(w, `{"hibernation_supported": true, "is_hibernated": false, "instance_name": "test-instance"}`)
	}))
	defer server.Close()

	client := NewClient(server.URL)
	status, err := client.GetInstanceHibernationStatus(context.Background(), "test-instance")

	assert.NoError(t, err)
	assert.NotNil(t, status)
	assert.True(t, status.HibernationSupported)
	assert.False(t, status.PossiblyHibernated)
}

// TestDeleteInstance tests the DeleteInstance method
func TestDeleteInstance(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "DELETE", r.Method)
		assert.Equal(t, "/api/v1/instances/test-instance", r.URL.Path)
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	client := NewClient(server.URL)
	err := client.DeleteInstance(context.Background(), "test-instance")

	assert.NoError(t, err)
}

// TestConnectInstance tests the ConnectInstance method
func TestConnectInstance(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "GET", r.Method)
		assert.Equal(t, "/api/v1/instances/test-instance/connect", r.URL.Path)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprint(w, `{"connection_info": "ssh user@1.2.3.4"}`)
	}))
	defer server.Close()

	client := NewClient(server.URL)
	connInfo, err := client.ConnectInstance(context.Background(), "test-instance")

	assert.NoError(t, err)
	assert.Equal(t, "ssh user@1.2.3.4", connInfo)
}

// TestListTemplates tests the ListTemplates method
func TestListTemplates(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "GET", r.Method)
		assert.Equal(t, "/api/v1/templates", r.URL.Path)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprint(w, `{"python-ml": {"name": "python-ml", "description": "Python ML"}}`)
	}))
	defer server.Close()

	client := NewClient(server.URL)
	templates, err := client.ListTemplates(context.Background())

	assert.NoError(t, err)
	assert.NotNil(t, templates)
	assert.Contains(t, templates, "python-ml")
}

// TestGetTemplate tests the GetTemplate method
func TestGetTemplate(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "GET", r.Method)
		assert.Equal(t, "/api/v1/templates/python-ml", r.URL.Path)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprint(w, `{"name": "python-ml", "description": "Python ML environment"}`)
	}))
	defer server.Close()

	client := NewClient(server.URL)
	template, err := client.GetTemplate(context.Background(), "python-ml")

	assert.NoError(t, err)
	assert.NotNil(t, template)
	assert.Equal(t, "python-ml", template.Name)
}

// TestHTTPClientErrorHandling tests error response handling
func TestHTTPClientErrorHandling(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = fmt.Fprint(w, `{"error": "invalid request"}`)
	}))
	defer server.Close()

	client := NewClient(server.URL)

	err := client.Ping(context.Background())

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "API error 400")
}

// TestHTTPClientTimeout tests timeout handling
func TestHTTPClientTimeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(200 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	// Create client and set very short timeout
	client := NewClient(server.URL).(*HTTPClient)
	client.httpClient.Timeout = 10 * time.Millisecond

	err := client.Ping(context.Background())

	assert.Error(t, err)
	assert.Contains(t, strings.ToLower(err.Error()), "timeout")
}

// TestHTTPClientContextCancellation tests context cancellation
func TestHTTPClientContextCancellation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewClient(server.URL)

	// Create context that cancels quickly
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(10 * time.Millisecond)
		cancel()
	}()

	err := client.Ping(ctx)

	assert.Error(t, err)
	assert.Contains(t, strings.ToLower(err.Error()), "cancel")
}

// TestHTTPClientInvalidJSON tests invalid JSON response handling
func TestHTTPClientInvalidJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprint(w, `{"invalid": json}`)
	}))
	defer server.Close()

	client := NewClient(server.URL)

	_, err := client.GetStatus(context.Background())

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to decode response")
}

// TestHTTPClientNoContent tests 204 No Content responses
func TestHTTPClientNoContent(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	client := NewClient(server.URL)
	err := client.Ping(context.Background())

	assert.NoError(t, err)
}

// TestHTTPClientHeaders tests request headers
func TestHTTPClientHeaders(t *testing.T) {
	var receivedHeaders http.Header

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedHeaders = r.Header.Clone()

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_, _ = fmt.Fprint(w, `{"name": "test-instance"}`)
	}))
	defer server.Close()

	options := Options{
		AWSProfile: "test-profile",
		AWSRegion:  "us-east-1",
	}
	client := NewClientWithOptions(server.URL, options)

	req := types.LaunchRequest{
		Name:     "test-instance",
		Template: "python-ml",
	}

	_, err := client.LaunchInstance(context.Background(), req)

	assert.NoError(t, err)
	assert.Equal(t, "application/json", receivedHeaders.Get("Content-Type"))
	assert.Equal(t, "test-profile", receivedHeaders.Get("X-AWS-Profile"))
	assert.Equal(t, "us-east-1", receivedHeaders.Get("X-AWS-Region"))
	assert.NotEmpty(t, receivedHeaders.Get("User-Agent"))
}

// TestShutdown tests the Shutdown method
func TestShutdown(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "/api/v1/shutdown", r.URL.Path)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewClient(server.URL)
	err := client.Shutdown(context.Background())

	assert.NoError(t, err)
}

// TestMakeRequest tests the generic MakeRequest method
func TestMakeRequest(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "/custom/endpoint", r.URL.Path)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprint(w, `{"result": "success"}`)
	}))
	defer server.Close()

	client := NewClient(server.URL)

	requestData := map[string]string{"key": "value"}
	responseData, err := client.MakeRequest("POST", "/custom/endpoint", requestData)

	assert.NoError(t, err)
	assert.Contains(t, string(responseData), "success")
}

// TestMakeRequestErrorHandling tests MakeRequest error handling
func TestMakeRequestErrorHandling(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = fmt.Fprint(w, `{"error": "internal server error"}`)
	}))
	defer server.Close()

	client := NewClient(server.URL)

	_, err := client.MakeRequest("GET", "/error", nil)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "API error 500")
	assert.Contains(t, err.Error(), "GET /error")
}

// TestSetOptions tests options configuration
func TestSetOptions(t *testing.T) {
	client := NewClient("http://localhost:8947").(*HTTPClient)

	// Initially empty
	assert.Empty(t, client.awsProfile)
	assert.Empty(t, client.awsRegion)

	options := Options{
		AWSProfile:      "test-profile",
		AWSRegion:       "us-west-2",
		InvitationToken: "token123",
		OwnerAccount:    "123456789012",
		S3ConfigPath:    "/tmp/config",
	}

	client.SetOptions(options)

	assert.Equal(t, "test-profile", client.awsProfile)
	assert.Equal(t, "us-west-2", client.awsRegion)
	assert.Equal(t, "token123", client.invitationToken)
	assert.Equal(t, "123456789012", client.ownerAccount)
	assert.Equal(t, "/tmp/config", client.s3ConfigPath)
}

// TestServerDown tests handling when server is down
func TestServerDown(t *testing.T) {
	// Use invalid URL that will fail to connect
	client := NewClient("http://localhost:99999")

	err := client.Ping(context.Background())

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "dial tcp")
}

// TestMalformedURL tests handling of malformed URLs
func TestMalformedURL(t *testing.T) {
	// Test that client creation doesn't panic with unusual URLs
	tests := []string{
		"not-a-url",
		"http://",
		"https://valid-host.com:8947",
	}

	for _, url := range tests {
		client := NewClient(url)
		assert.NotNil(t, client)
	}
}

// TestHTTPClientThreadSafety tests concurrent access to client
func TestHTTPClientThreadSafety(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewClient(server.URL)

	// Run multiple concurrent requests
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func() {
			err := client.Ping(context.Background())
			assert.NoError(t, err)
			done <- true
		}()
	}

	// Wait for all goroutines to complete
	for i := 0; i < 10; i++ {
		<-done
	}
}
