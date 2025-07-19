package types

import (
	"time"
)

// APIVersionInfo represents information about an API version
type APIVersionInfo struct {
	// Version string (e.g., "v1", "v2.1", "v2-beta")
	Version string `json:"version"`

	// Status of this version (stable, beta, alpha, deprecated)
	Status string `json:"status"`

	// IsDefault indicates if this is the default version
	IsDefault bool `json:"is_default"`

	// ReleaseDate when this version was released
	ReleaseDate time.Time `json:"release_date,omitempty"`

	// DeprecationDate when this version was deprecated (if applicable)
	DeprecationDate *time.Time `json:"deprecation_date,omitempty"`

	// SunsetDate when this version will be removed (if applicable)
	SunsetDate *time.Time `json:"sunset_date,omitempty"`

	// DocsURL is the URL to the documentation for this API version
	DocsURL string `json:"docs_url,omitempty"`
}

// APIVersionResponse represents the response from the API versions endpoint
type APIVersionResponse struct {
	// All supported API versions
	Versions []APIVersionInfo `json:"versions"`

	// Default version to use when no version is specified
	DefaultVersion string `json:"default_version"`

	// Current stable version
	StableVersion string `json:"stable_version"`

	// Latest version (may be alpha/beta)
	LatestVersion string `json:"latest_version"`

	// Base URL for API documentation
	DocsBaseURL string `json:"docs_base_url,omitempty"`
}

// APIErrorResponse represents a standard error response format for API errors
type APIErrorResponse struct {
	// Error code (e.g., "not_found", "validation_error")
	Code string `json:"code"`

	// HTTP status code
	Status int `json:"status"`

	// User-friendly error message
	Message string `json:"message"`

	// More detailed error information
	Details string `json:"details,omitempty"`

	// Request ID for tracking
	RequestID string `json:"request_id,omitempty"`

	// API version used for the request
	APIVersion string `json:"api_version,omitempty"`

	// Documentation URL for this error type
	DocsURL string `json:"docs_url,omitempty"`
}