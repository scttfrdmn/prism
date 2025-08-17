package client

import (
	"context"
	"net/http"
	"time"
)

// profileContextKey is the key used to store profile information in context
type profileContextKey struct{}

// ProfileContextKey is used to access profile information in context
var ProfileContextKey = profileContextKey{}

// ExtendedOptions extends the basic Options with additional fields
type ExtendedOptions struct {
	// AWS configuration
	AWSProfile string
	AWSRegion  string

	// Invitation details
	InvitationToken string
	OwnerAccount    string
	S3ConfigPath    string

	// Profile information
	ProfileID string
}

// PerformanceOptions represents HTTP client performance configuration
type PerformanceOptions struct {
	Timeout        time.Duration
	MaxConnections int
	KeepAlive      time.Duration
	RequestRetries int
	MaxIdleConns   int
}

// DefaultPerformanceOptions returns sensible defaults for HTTP client performance
func DefaultPerformanceOptions() PerformanceOptions {
	return PerformanceOptions{
		Timeout:        30 * time.Second,
		MaxConnections: 10,
		KeepAlive:      30 * time.Second,
		RequestRetries: 3,
		MaxIdleConns:   100,
	}
}

// ApplyClientOptions applies configuration options to a client
func ApplyClientOptions(client CloudWorkstationAPI, options Options) CloudWorkstationAPI {
	// Apply basic options
	client.SetOptions(options)
	return client
}

// ApplyExtendedClientOptions applies extended configuration options to a client
func ApplyExtendedClientOptions(client CloudWorkstationAPI, options ExtendedOptions) CloudWorkstationAPI {
	// Convert to basic options and apply
	basicOptions := Options{
		AWSProfile:      options.AWSProfile,
		AWSRegion:       options.AWSRegion,
		InvitationToken: options.InvitationToken,
		OwnerAccount:    options.OwnerAccount,
		S3ConfigPath:    options.S3ConfigPath,
	}

	client.SetOptions(basicOptions)
	return client
}

// createHTTPClient creates an HTTP client with performance optimizations
func createHTTPClient(opts PerformanceOptions) *http.Client {
	transport := &http.Transport{
		MaxIdleConns:        opts.MaxIdleConns,
		MaxIdleConnsPerHost: opts.MaxConnections,
		IdleConnTimeout:     opts.KeepAlive,
	}

	return &http.Client{
		Timeout:   opts.Timeout,
		Transport: transport,
	}
}

// GetProfileFromContext extracts profile information from context
func GetProfileFromContext(ctx context.Context) (string, bool) {
	profileID, ok := ctx.Value(ProfileContextKey).(string)
	return profileID, ok
}

// SetProfileInContext adds profile information to context
func SetProfileInContext(ctx context.Context, profileID string) context.Context {
	return context.WithValue(ctx, ProfileContextKey, profileID)
}
