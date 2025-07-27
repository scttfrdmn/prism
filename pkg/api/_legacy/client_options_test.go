package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	
	"github.com/scttfrdmn/cloudworkstation/pkg/profile"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWithProfile(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check that headers are correctly set based on profile type
		if r.URL.Path == "/personal" {
			assert.Equal(t, "personal-aws", r.Header.Get("X-AWS-Profile"))
			assert.Equal(t, "us-west-2", r.Header.Get("X-AWS-Region"))
			assert.Equal(t, "personal-id", r.Header.Get("X-Profile-ID"))
			assert.Equal(t, "", r.Header.Get("X-Invitation-Token"))
			w.WriteHeader(http.StatusOK)
		} else if r.URL.Path == "/invitation" {
			assert.Equal(t, "", r.Header.Get("X-AWS-Profile"))
			assert.Equal(t, "us-east-1", r.Header.Get("X-AWS-Region"))
			assert.Equal(t, "invitation-id", r.Header.Get("X-Profile-ID"))
			assert.Equal(t, "test-token", r.Header.Get("X-Invitation-Token"))
			assert.Equal(t, "test-account", r.Header.Get("X-Owner-Account"))
			assert.Equal(t, "s3://config/path", r.Header.Get("X-S3-Config-Path"))
			w.WriteHeader(http.StatusOK)
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()
	
	// Create base client
	client := NewClient(server.URL)
	
	// Create profile manager mock
	profiles := map[string]profile.Profile{
		"personal-id": {
			Type:       profile.ProfileTypePersonal,
			Name:       "Personal Profile",
			AWSProfile: "personal-aws",
			Region:     "us-west-2",
		},
		"invitation-id": {
			Type:            profile.ProfileTypeInvitation,
			Name:            "Invitation Profile",
			Region:          "us-east-1",
			InvitationToken: "test-token",
			OwnerAccount:    "test-account",
			S3ConfigPath:    "s3://config/path",
		},
	}
	
	mockProfileManager := &mockProfileManager{
		profiles: profiles,
	}
	
	t.Run("WithPersonalProfile", func(t *testing.T) {
		// Get client with personal profile
		profileClient, err := client.WithProfile(mockProfileManager, "personal-id")
		require.NoError(t, err)
		
		// Make request
		req, err := http.NewRequest("GET", server.URL+"/personal", nil)
		require.NoError(t, err)
		
		profileClient.addRequestHeaders(req)
		
		// Send request
		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()
		
		// Check response
		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})
	
	t.Run("WithInvitationProfile", func(t *testing.T) {
		// Get client with invitation profile
		profileClient, err := client.WithProfile(mockProfileManager, "invitation-id")
		require.NoError(t, err)
		
		// Make request
		req, err := http.NewRequest("GET", server.URL+"/invitation", nil)
		require.NoError(t, err)
		
		profileClient.addRequestHeaders(req)
		
		// Send request
		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()
		
		// Check response
		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})
	
	t.Run("WithNonExistentProfile", func(t *testing.T) {
		// Get client with non-existent profile
		_, err := client.WithProfile(mockProfileManager, "non-existent")
		require.Error(t, err)
	})
}

func TestContextClient_WithProfile(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check that the context client correctly passes profile information
		if r.URL.Path == "/api/test" {
			assert.Equal(t, "test-profile", r.Header.Get("X-AWS-Profile"))
			assert.Equal(t, "us-west-2", r.Header.Get("X-AWS-Region"))
			assert.Equal(t, "profile-id", r.Header.Get("X-Profile-ID"))
			w.WriteHeader(http.StatusOK)
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()
	
	// Create base client
	baseClient := NewClient(server.URL)
	
	// Create context client
	client := &ContextClient{
		client: baseClient,
	}
	
	// Create profile manager mock
	profiles := map[string]profile.Profile{
		"profile-id": {
			Type:       profile.ProfileTypePersonal,
			Name:       "Test Profile",
			AWSProfile: "test-profile",
			Region:     "us-west-2",
		},
	}
	
	mockProfileManager := &mockProfileManager{
		profiles: profiles,
	}
	
	// Get client with profile
	contextClient, err := client.WithProfile(mockProfileManager, "profile-id")
	require.NoError(t, err)
	
	// Test that the context client works with the profile
	ctx := context.Background()
	req, _ := http.NewRequestWithContext(ctx, "GET", server.URL+"/api/test", nil)
	
	// Get the underlying client and add headers
	baseClient = contextClient.client
	baseClient.addRequestHeaders(req)
	
	// Send request
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	
	// Check response
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestWithProfileContext(t *testing.T) {
	// Create a profile
	prof := &profile.Profile{
		Type:       profile.ProfileTypePersonal,
		Name:       "Test Profile",
		AWSProfile: "test-profile",
		Region:     "us-west-2",
	}
	
	// Create context with profile
	ctx := context.Background()
	ctxWithProfile := WithProfileContext(ctx, prof)
	
	// Get profile from context
	retrievedProf, ok := GetProfileFromContext(ctxWithProfile)
	
	// Check that profile was retrieved correctly
	assert.True(t, ok)
	assert.Equal(t, prof, retrievedProf)
	
	// Try getting profile from a context without one
	emptyCtx := context.Background()
	emptyProf, ok := GetProfileFromContext(emptyCtx)
	assert.False(t, ok)
	assert.Nil(t, emptyProf)
}

// Mock profile manager for testing
type mockProfileManager struct {
	profiles map[string]profile.Profile
}

func (m *mockProfileManager) GetProfile(id string) (*profile.Profile, error) {
	prof, exists := m.profiles[id]
	if !exists {
		return nil, profile.ErrProfileNotFound
	}
	return &prof, nil
}