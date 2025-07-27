package api

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAddRequestHeaders(t *testing.T) {
	tests := []struct {
		name           string
		clientConfig   Client
		expectedHeaders map[string]string
	}{
		{
			name: "All headers",
			clientConfig: Client{
				awsProfile:      "test-profile",
				awsRegion:       "us-west-2",
				invitationToken: "test-token",
				ownerAccount:    "123456789012",
				s3ConfigPath:    "s3://bucket/path",
				profileID:       "profile-123",
			},
			expectedHeaders: map[string]string{
				"X-AWS-Profile":      "test-profile",
				"X-AWS-Region":       "us-west-2",
				"X-Invitation-Token": "test-token",
				"X-Owner-Account":    "123456789012",
				"X-S3-Config-Path":   "s3://bucket/path",
				"X-Profile-ID":       "profile-123",
			},
		},
		{
			name: "AWS headers only",
			clientConfig: Client{
				awsProfile: "test-profile",
				awsRegion:  "us-west-2",
			},
			expectedHeaders: map[string]string{
				"X-AWS-Profile": "test-profile",
				"X-AWS-Region":  "us-west-2",
			},
		},
		{
			name: "Invitation headers only",
			clientConfig: Client{
				invitationToken: "test-token",
				ownerAccount:    "123456789012",
			},
			expectedHeaders: map[string]string{
				"X-Invitation-Token": "test-token",
				"X-Owner-Account":    "123456789012",
			},
		},
		{
			name: "Invitation headers with S3 path",
			clientConfig: Client{
				invitationToken: "test-token",
				ownerAccount:    "123456789012",
				s3ConfigPath:    "s3://bucket/path",
			},
			expectedHeaders: map[string]string{
				"X-Invitation-Token": "test-token",
				"X-Owner-Account":    "123456789012",
				"X-S3-Config-Path":   "s3://bucket/path",
			},
		},
		{
			name: "Profile ID only",
			clientConfig: Client{
				profileID: "profile-123",
			},
			expectedHeaders: map[string]string{
				"X-Profile-ID": "profile-123",
			},
		},
		{
			name:           "No headers",
			clientConfig:   Client{},
			expectedHeaders: map[string]string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, _ := http.NewRequest("GET", "http://example.com", nil)
			client := tt.clientConfig
			
			// Call the method to test
			client.addRequestHeaders(req)
			
			// Check headers
			for key, value := range tt.expectedHeaders {
				assert.Equal(t, value, req.Header.Get(key), "Header %s should be %s", key, value)
			}
			
			// Check no unexpected headers
			assert.Equal(t, len(tt.expectedHeaders), len(req.Header), "Should not have unexpected headers")
		})
	}
}