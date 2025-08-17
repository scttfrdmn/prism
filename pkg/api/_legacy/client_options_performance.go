package api

import (
	"net/http"
	"sync"
	"time"
)

// ClientPerformanceOptions configures performance-related settings for the API client
type ClientPerformanceOptions struct {
	// Maximum number of idle connections in the connection pool
	MaxIdleConnections int

	// Idle connection timeout - connections idle for longer will be closed
	IdleConnectionTimeout time.Duration

	// Connection keep-alive duration
	KeepAlive time.Duration

	// Timeout for establishing new connections
	DialTimeout time.Duration

	// Timeout for TLS handshake
	TLSHandshakeTimeout time.Duration

	// Response header timeout
	ResponseHeaderTimeout time.Duration

	// Expected request timeout - for non-context requests
	ExpectRequestTimeout time.Duration
}

// DefaultClientPerformanceOptions returns sensible default performance options
func DefaultClientPerformanceOptions() ClientPerformanceOptions {
	return ClientPerformanceOptions{
		MaxIdleConnections:    100,
		IdleConnectionTimeout: 90 * time.Second,
		KeepAlive:             30 * time.Second,
		DialTimeout:           10 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ResponseHeaderTimeout: 5 * time.Second,
		ExpectRequestTimeout:  1 * time.Second,
	}
}

// httpClientCache caches HTTP clients by profile ID to avoid creating new clients
// for each request when switching between profiles
type httpClientCache struct {
	clients map[string]*http.Client
	mu      sync.RWMutex
}

// newHTTPClientCache creates a new HTTP client cache
func newHTTPClientCache() *httpClientCache {
	return &httpClientCache{
		clients: make(map[string]*http.Client),
	}
}

// getClient returns a cached HTTP client for the given profile ID or creates a new one
// if it doesn't exist in the cache
func (c *httpClientCache) getClient(profileID string, options ClientPerformanceOptions) *http.Client {
	c.mu.RLock()
	client, exists := c.clients[profileID]
	c.mu.RUnlock()

	if !exists {
		client = createHTTPClient(options)
		c.mu.Lock()
		c.clients[profileID] = client
		c.mu.Unlock()
	}

	return client
}

// createHTTPClient creates a new HTTP client with the given performance options
func createHTTPClient(options ClientPerformanceOptions) *http.Client {
	transport := &http.Transport{
		MaxIdleConns:          options.MaxIdleConnections,
		IdleConnTimeout:       options.IdleConnectionTimeout,
		ResponseHeaderTimeout: options.ResponseHeaderTimeout,
		TLSHandshakeTimeout:   options.TLSHandshakeTimeout,
		ExpectContinueTimeout: options.ExpectRequestTimeout,
	}

	return &http.Client{
		Transport: transport,
		Timeout:   30 * time.Second, // Default timeout for non-context requests
	}
}

// profileHTTPClientCache is a global cache for HTTP clients by profile
var profileHTTPClientCache = newHTTPClientCache()

// getHTTPClientForProfile returns an HTTP client optimized for the given profile
func getHTTPClientForProfile(profileID string) *http.Client {
	return profileHTTPClientCache.getClient(profileID, DefaultClientPerformanceOptions())
}

// SetPerformanceOptions configures the performance options for the client
func (c *Client) SetPerformanceOptions(options ClientPerformanceOptions) {
	c.httpClient = createHTTPClient(options)
}

// WithPerformanceOptions returns a new client with the given performance options
func (c *Client) WithPerformanceOptions(options ClientPerformanceOptions) *Client {
	client := NewClient(c.baseURL)
	client.SetPerformanceOptions(options)

	// Copy current configuration
	client.awsProfile = c.awsProfile
	client.awsRegion = c.awsRegion
	client.invitationToken = c.invitationToken
	client.ownerAccount = c.ownerAccount
	client.s3ConfigPath = c.s3ConfigPath
	client.profileID = c.profileID

	return client
}
