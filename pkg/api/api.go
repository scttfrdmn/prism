// Package api provides the CloudWorkstation REST API client implementation.
//
// This package maintains backward compatibility while internally using
// the reorganized client package structure. New code should prefer
// using pkg/api/client directly.
package api

import (
	"github.com/scttfrdmn/cloudworkstation/pkg/api/client"
)

// Backward compatibility aliases - these maintain the existing API
type CloudWorkstationAPI = client.CloudWorkstationAPI
type ClientOptions = client.Options

// NewClient creates a new API client (backward compatibility)
func NewClient(baseURL string) CloudWorkstationAPI {
	return client.NewClient(baseURL)
}

// NewClientWithOptions creates a new API client with options (backward compatibility)
func NewClientWithOptions(baseURL string, opts ClientOptions) CloudWorkstationAPI {
	return client.NewClientWithOptions(baseURL, opts)
}

// Registry-specific response types for API operations (backward compatibility)
type RegistryStatusResponse = client.RegistryStatusResponse
type AMIReferenceResponse = client.AMIReferenceResponse
