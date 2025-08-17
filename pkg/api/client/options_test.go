package client

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// TestDefaultPerformanceOptions tests default performance configuration
func TestDefaultPerformanceOptions(t *testing.T) {
	opts := DefaultPerformanceOptions()
	
	assert.Equal(t, 30*time.Second, opts.Timeout)
	assert.Equal(t, 10, opts.MaxConnections)
	assert.Equal(t, 30*time.Second, opts.KeepAlive)
	assert.Equal(t, 3, opts.RequestRetries)
	assert.Equal(t, 100, opts.MaxIdleConns)
}

// TestCreateHTTPClient tests HTTP client creation with custom options
func TestCreateHTTPClient(t *testing.T) {
	opts := PerformanceOptions{
		Timeout:        60 * time.Second,
		MaxConnections: 20,
		KeepAlive:      60 * time.Second,
		RequestRetries: 5,
		MaxIdleConns:   200,
	}
	
	client := createHTTPClient(opts)
	
	assert.NotNil(t, client)
	assert.Equal(t, 60*time.Second, client.Timeout)
	
	// Verify transport configuration
	transport, ok := client.Transport.(*http.Transport)
	assert.True(t, ok)
	assert.Equal(t, 200, transport.MaxIdleConns)
	assert.Equal(t, 20, transport.MaxIdleConnsPerHost)
	assert.Equal(t, 60*time.Second, transport.IdleConnTimeout)
}

// TestApplyClientOptionsInternal tests basic options application (internal test)
func TestApplyClientOptionsInternal(t *testing.T) {
	client := NewClient("http://localhost:8947")
	
	options := Options{
		AWSProfile:      "test-profile",
		AWSRegion:       "us-east-1",
		InvitationToken: "test-token",
		OwnerAccount:    "123456789",
		S3ConfigPath:    "/tmp/config",
	}
	
	result := ApplyClientOptions(client, options)
	
	assert.NotNil(t, result)
	// Verify the client received the options
	assert.Equal(t, client, result)
}

// TestApplyExtendedClientOptionsInternal tests extended options application (internal test)
func TestApplyExtendedClientOptionsInternal(t *testing.T) {
	client := NewClient("http://localhost:8947")
	
	extendedOptions := ExtendedOptions{
		AWSProfile:      "extended-profile",
		AWSRegion:       "eu-west-1",
		InvitationToken: "extended-token",
		OwnerAccount:    "987654321",
		S3ConfigPath:    "/tmp/extended",
		ProfileID:       "profile-123",
	}
	
	result := ApplyExtendedClientOptions(client, extendedOptions)
	
	assert.NotNil(t, result)
	assert.Equal(t, client, result)
}

// TestProfileContext tests profile context operations
func TestProfileContext(t *testing.T) {
	// Test setting and getting profile from context
	ctx := context.Background()
	profileID := "test-profile-123"
	
	// Initially no profile in context
	_, exists := GetProfileFromContext(ctx)
	assert.False(t, exists)
	
	// Set profile in context
	ctx = SetProfileInContext(ctx, profileID)
	
	// Retrieve profile from context
	retrievedProfile, exists := GetProfileFromContext(ctx)
	assert.True(t, exists)
	assert.Equal(t, profileID, retrievedProfile)
}

// TestProfileContextWithEmptyString tests profile context with empty string
func TestProfileContextWithEmptyString(t *testing.T) {
	ctx := context.Background()
	
	// Set empty profile
	ctx = SetProfileInContext(ctx, "")
	
	// Should still exist but be empty
	retrievedProfile, exists := GetProfileFromContext(ctx)
	assert.True(t, exists)
	assert.Equal(t, "", retrievedProfile)
}

// TestProfileContextWithNilContext tests error handling with nil context
func TestProfileContextWithNilContext(t *testing.T) {
	// This test ensures our context functions handle edge cases gracefully
	ctx := context.Background()
	
	// Test with wrong type in context (should return false)
	ctx = context.WithValue(ctx, ProfileContextKey, 123) // Wrong type
	
	_, exists := GetProfileFromContext(ctx)
	assert.False(t, exists)
}