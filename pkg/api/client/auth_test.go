package client

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/scttfrdmn/cloudworkstation/pkg/types"
)

// TestHTTPClientAuthHeaders tests authentication header handling
func TestHTTPClientAuthHeaders(t *testing.T) {
	// Track headers received by server
	var receivedHeaders http.Header

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedHeaders = r.Header.Clone()
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprint(w, `{"status": "running"}`)
	}))
	defer server.Close()

	// Create client with auth options
	options := Options{
		AWSProfile:      "test-profile",
		AWSRegion:       "us-east-1",
		InvitationToken: "test-token",
		OwnerAccount:    "123456789012",
		S3ConfigPath:    "/tmp/config",
	}

	client := NewClientWithOptions(server.URL, options)

	// Make request to verify headers
	_, err := client.GetStatus(context.Background())

	assert.NoError(t, err)
	assert.Equal(t, "test-profile", receivedHeaders.Get("X-AWS-Profile"))
	assert.Equal(t, "us-east-1", receivedHeaders.Get("X-AWS-Region"))
	assert.Equal(t, "application/json", receivedHeaders.Get("Content-Type"))
	assert.NotEmpty(t, receivedHeaders.Get("User-Agent"))
}

// TestHTTPClientAuthHeadersPartial tests partial authentication headers
func TestHTTPClientAuthHeadersPartial(t *testing.T) {
	var receivedHeaders http.Header

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedHeaders = r.Header.Clone()
		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprint(w, `{"status": "running"}`)
	}))
	defer server.Close()

	// Create client with partial auth options
	options := Options{
		AWSProfile: "partial-profile",
		// No AWSRegion, InvitationToken, etc.
	}

	client := NewClientWithOptions(server.URL, options)

	_, err := client.GetStatus(context.Background())

	assert.NoError(t, err)
	assert.Equal(t, "partial-profile", receivedHeaders.Get("X-AWS-Profile"))
	assert.Empty(t, receivedHeaders.Get("X-AWS-Region"))
	assert.Empty(t, receivedHeaders.Get("X-API-Key"))
}

// TestHTTPClientNoAuthHeaders tests request without authentication
func TestHTTPClientNoAuthHeaders(t *testing.T) {
	var receivedHeaders http.Header

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedHeaders = r.Header.Clone()
		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprint(w, `{"status": "running"}`)
	}))
	defer server.Close()

	// Create client without auth options
	client := NewClient(server.URL)

	_, err := client.GetStatus(context.Background())

	assert.NoError(t, err)
	assert.Empty(t, receivedHeaders.Get("X-AWS-Profile"))
	assert.Empty(t, receivedHeaders.Get("X-AWS-Region"))
	assert.Empty(t, receivedHeaders.Get("X-API-Key"))
	assert.Equal(t, "application/json", receivedHeaders.Get("Content-Type"))
}

// TestHTTPClientAPIKeyAuth tests API key authentication
func TestHTTPClientAPIKeyAuth(t *testing.T) {
	var receivedHeaders http.Header

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedHeaders = r.Header.Clone()
		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprint(w, `{"status": "running"}`)
	}))
	defer server.Close()

	client := NewClient(server.URL).(*HTTPClient)

	// Set API key directly (simulating internal auth setup)
	client.apiKey = "test-api-key-123"

	_, err := client.GetStatus(context.Background())

	assert.NoError(t, err)
	assert.Equal(t, "test-api-key-123", receivedHeaders.Get("X-API-Key"))
}

// TestHTTPClientAuthOptionsUpdate tests updating auth options
func TestHTTPClientAuthOptionsUpdate(t *testing.T) {
	client := NewClient("http://localhost:8947").(*HTTPClient)

	// Initial state - no auth
	assert.Empty(t, client.awsProfile)
	assert.Empty(t, client.awsRegion)
	assert.Empty(t, client.invitationToken)

	// Set initial auth options
	options1 := Options{
		AWSProfile: "profile1",
		AWSRegion:  "us-east-1",
	}
	client.SetOptions(options1)

	assert.Equal(t, "profile1", client.awsProfile)
	assert.Equal(t, "us-east-1", client.awsRegion)
	assert.Empty(t, client.invitationToken)

	// Update auth options
	options2 := Options{
		AWSProfile:      "profile2",
		AWSRegion:       "us-west-2",
		InvitationToken: "token456",
		OwnerAccount:    "987654321098",
	}
	client.SetOptions(options2)

	assert.Equal(t, "profile2", client.awsProfile)
	assert.Equal(t, "us-west-2", client.awsRegion)
	assert.Equal(t, "token456", client.invitationToken)
	assert.Equal(t, "987654321098", client.ownerAccount)

	// Clear auth options
	options3 := Options{}
	client.SetOptions(options3)

	assert.Empty(t, client.awsProfile)
	assert.Empty(t, client.awsRegion)
	assert.Empty(t, client.invitationToken)
	assert.Empty(t, client.ownerAccount)
}

// TestHTTPClientAuthenticationErrorHandling tests authentication-related error scenarios
func TestHTTPClientAuthenticationErrorHandling(t *testing.T) {
	tests := []struct {
		name           string
		serverResponse func(w http.ResponseWriter, r *http.Request)
		expectedError  string
	}{
		{
			name: "401 Unauthorized",
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusUnauthorized)
				_, _ = fmt.Fprint(w, `{"error": "invalid credentials"}`)
			},
			expectedError: "API error 401",
		},
		{
			name: "403 Forbidden",
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusForbidden)
				_, _ = fmt.Fprint(w, `{"error": "insufficient permissions"}`)
			},
			expectedError: "API error 403",
		},
		{
			name: "Missing required headers",
			serverResponse: func(w http.ResponseWriter, r *http.Request) {
				if r.Header.Get("X-AWS-Profile") == "" {
					w.WriteHeader(http.StatusBadRequest)
					_, _ = fmt.Fprint(w, `{"error": "missing AWS profile"}`)
					return
				}
				w.WriteHeader(http.StatusOK)
				_, _ = fmt.Fprint(w, `{"status": "running"}`)
			},
			expectedError: "API error 400",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(tt.serverResponse))
			defer server.Close()

			client := NewClient(server.URL)

			_, err := client.GetStatus(context.Background())

			assert.Error(t, err)
			assert.Contains(t, err.Error(), tt.expectedError)
		})
	}
}

// TestHTTPClientUserAgentHeader tests User-Agent header is set
func TestHTTPClientUserAgentHeader(t *testing.T) {
	var receivedHeaders http.Header

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedHeaders = r.Header.Clone()
		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprint(w, `{"status": "running"}`)
	}))
	defer server.Close()

	client := NewClient(server.URL)

	_, err := client.GetStatus(context.Background())

	assert.NoError(t, err)

	userAgent := receivedHeaders.Get("User-Agent")
	assert.NotEmpty(t, userAgent)
	// User-Agent should be set by Go's HTTP client
	assert.Contains(t, userAgent, "Go-http-client")
}

// TestHTTPClientMakeRequestMethod tests the public MakeRequest method
func TestHTTPClientMakeRequestMethod(t *testing.T) {
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

// TestHTTPClientMakeRequestErrorHandling tests MakeRequest error handling
func TestHTTPClientMakeRequestErrorHandling(t *testing.T) {
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

// TestHTTPClientLaunchInstanceWithAuth tests authenticated launch request
func TestHTTPClientLaunchInstanceWithAuth(t *testing.T) {
	var receivedHeaders http.Header

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedHeaders = r.Header.Clone()

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_, _ = fmt.Fprint(w, `{"name": "test-instance", "instance_id": "i-123"}`)
	}))
	defer server.Close()

	options := Options{
		AWSProfile: "auth-profile",
		AWSRegion:  "eu-west-1",
	}
	client := NewClientWithOptions(server.URL, options)

	req := types.LaunchRequest{
		Name:     "test-instance",
		Template: "python-ml",
	}

	_, err := client.LaunchInstance(context.Background(), req)

	assert.NoError(t, err)
	assert.Equal(t, "auth-profile", receivedHeaders.Get("X-AWS-Profile"))
	assert.Equal(t, "eu-west-1", receivedHeaders.Get("X-AWS-Region"))
}
