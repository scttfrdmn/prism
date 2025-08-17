package daemon

import (
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"
)

// APIVersion represents a specific API version
type APIVersion struct {
	// Version identifier (e.g., "v1", "v2")
	Version string

	// Status of this version (stable, beta, alpha, deprecated)
	Status string

	// IsDefault indicates if this is the default version to use when no version is specified
	IsDefault bool

	// ReleaseDate is when this version was released
	ReleaseDate time.Time

	// DeprecationDate is when this version was deprecated (if applicable)
	DeprecationDate *time.Time

	// SunsetDate is when this version will be removed (if applicable)
	SunsetDate *time.Time

	// DocsURL is the documentation URL for this version
	DocsURL string
}

// APIVersionManager manages API versions and routing
type APIVersionManager struct {
	// Supported API versions, keyed by version string (e.g., "v1")
	versions map[string]APIVersion

	// Default version to use when no version is specified
	defaultVersion string

	// Current stable version
	stableVersion string

	// Latest version (may be alpha/beta)
	latestVersion string

	// Base path for API endpoints (typically "/api")
	basePath string
}

// NewAPIVersionManager creates a new API version manager
func NewAPIVersionManager(basePath string) *APIVersionManager {
	manager := &APIVersionManager{
		versions: make(map[string]APIVersion),
		basePath: basePath,
	}

	// Add initial v1 version as default and stable
	manager.AddVersion(APIVersion{
		Version:     "v1",
		Status:      "stable",
		IsDefault:   true,
		ReleaseDate: time.Now().AddDate(0, -6, 0), // Assume it was released 6 months ago
		DocsURL:     "https://docs.cloudworkstation.dev/api/v1",
	})

	return manager
}

// AddVersion adds a new API version to the manager
func (m *APIVersionManager) AddVersion(version APIVersion) {
	m.versions[version.Version] = version

	// Update default version if specified
	if version.IsDefault {
		m.defaultVersion = version.Version
	}

	// Update stable/latest version pointers based on version naming and status
	versionVal := extractVersionNumber(version.Version)

	// Latest version is the highest version number that's not deprecated
	currentLatestVal := extractVersionNumber(m.latestVersion)
	if versionVal > currentLatestVal && version.Status != "deprecated" {
		m.latestVersion = version.Version
	}

	// Stable version is the highest version with "stable" status
	if version.Status == "stable" {
		currentStableVal := extractVersionNumber(m.stableVersion)
		if versionVal > currentStableVal {
			m.stableVersion = version.Version
		}
	}
}

// GetVersion returns details about an API version
func (m *APIVersionManager) GetVersion(version string) (APIVersion, bool) {
	v, found := m.versions[version]
	return v, found
}

// GetDefaultVersion returns the default API version
func (m *APIVersionManager) GetDefaultVersion() string {
	return m.defaultVersion
}

// GetStableVersion returns the current stable API version
func (m *APIVersionManager) GetStableVersion() string {
	return m.stableVersion
}

// GetLatestVersion returns the latest API version (may be alpha/beta)
func (m *APIVersionManager) GetLatestVersion() string {
	return m.latestVersion
}

// GetSupportedVersions returns all supported API versions
func (m *APIVersionManager) GetSupportedVersions() []APIVersion {
	versions := make([]APIVersion, 0, len(m.versions))
	for _, v := range m.versions {
		versions = append(versions, v)
	}
	return versions
}

// ExtractVersionFromPath extracts the version string from a URL path
// Example: "/api/v1/instances" -> "v1"
func (m *APIVersionManager) ExtractVersionFromPath(path string) string {
	parts := strings.Split(strings.TrimPrefix(path, m.basePath+"/"), "/")
	if len(parts) == 0 {
		return ""
	}

	// Check if first part matches version pattern (e.g., "v1", "v2", "v2.1")
	if isVersionString(parts[0]) {
		return parts[0]
	}

	return ""
}

// TranslateRequestPath translates a request path based on API version
// This allows API evolution by mapping newer API calls to older implementations when needed
// Example: "/api/v2/instances" might map to a different handler than "/api/v1/instances"
func (m *APIVersionManager) TranslateRequestPath(path, version string) string {
	// In the initial implementation, we just return the original path
	// Future versions can implement more complex translation logic
	return path
}

// NormalizePath standardizes the path format for a given version
// Example: "/instances" -> "/api/v1/instances" (for default version v1)
func (m *APIVersionManager) NormalizePath(path, version string) string {
	// If path already starts with version, return as is
	if strings.HasPrefix(path, m.basePath+"/"+version+"/") {
		return path
	}

	// If path starts with base path but no version, add version
	if strings.HasPrefix(path, m.basePath+"/") {
		pathWithoutBase := strings.TrimPrefix(path, m.basePath+"/")
		parts := strings.Split(pathWithoutBase, "/")

		// Check if first part is a version; if not, insert version
		if len(parts) > 0 && !isVersionString(parts[0]) {
			return fmt.Sprintf("%s/%s/%s", m.basePath, version, pathWithoutBase)
		}
		return path
	}

	// If path doesn't start with base path, add base path and version
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	return fmt.Sprintf("%s/%s%s", m.basePath, version, path)
}

// BuildVersionedPath builds a properly versioned path
// Example: BuildVersionedPath("instances", "v2") -> "/api/v2/instances"
func (m *APIVersionManager) BuildVersionedPath(path, version string) string {
	// Remove leading slash if present
	path = strings.TrimPrefix(path, "/")

	return fmt.Sprintf("%s/%s/%s", m.basePath, version, path)
}

// VersionHeaderMiddleware adds versioning information to response headers
func (m *APIVersionManager) VersionHeaderMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Extract version from request path
		requestedVersion := m.ExtractVersionFromPath(r.URL.Path)
		if requestedVersion == "" {
			requestedVersion = m.defaultVersion
		}

		// Add version headers to response
		w.Header().Set("X-API-Version", requestedVersion)
		w.Header().Set("X-API-Latest-Version", m.latestVersion)
		w.Header().Set("X-API-Stable-Version", m.stableVersion)

		// Add deprecation warning header if applicable
		version, found := m.versions[requestedVersion]
		if found && version.Status == "deprecated" {
			w.Header().Set("X-API-Deprecated", "true")
			if version.SunsetDate != nil {
				w.Header().Set("X-API-Sunset-Date", version.SunsetDate.Format(time.RFC3339))
			}
			if version.DocsURL != "" {
				w.Header().Set("X-API-Docs-URL", version.DocsURL)
			}
		}

		next.ServeHTTP(w, r)
	})
}

// VersionRoutingMiddleware handles routing based on API version
func (m *APIVersionManager) VersionRoutingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Extract version from request path
		requestedVersion := m.ExtractVersionFromPath(r.URL.Path)

		// If no version specified, use default
		if requestedVersion == "" {
			// Rewrite URL to include default version
			r.URL.Path = m.NormalizePath(r.URL.Path, m.defaultVersion)
			requestedVersion = m.defaultVersion
		}

		// Check if requested version is supported
		_, found := m.versions[requestedVersion]
		if !found {
			http.Error(w, fmt.Sprintf("Unsupported API version: %s", requestedVersion), http.StatusNotFound)
			return
		}

		// Add version to request context for handlers to use
		ctx := r.Context()
		ctx = setAPIVersion(ctx, requestedVersion)
		r = r.WithContext(ctx)

		// Translate path for version compatibility if needed
		translatedPath := m.TranslateRequestPath(r.URL.Path, requestedVersion)
		if translatedPath != r.URL.Path {
			r.URL.Path = translatedPath
		}

		next.ServeHTTP(w, r)
	})
}

// Helper functions

// isVersionString checks if a string matches version pattern (e.g., "v1", "v2.1", "v3-alpha")
func isVersionString(s string) bool {
	pattern := regexp.MustCompile(`^v\d+(\.\d+)?(-\w+)?$`)
	return pattern.MatchString(s)
}

// extractVersionNumber extracts a numeric value from version string for comparison
// "v1" -> 1, "v2.5" -> 2.5, "v3-alpha" -> 3
func extractVersionNumber(version string) float64 {
	if version == "" {
		return 0
	}

	// Remove 'v' prefix
	version = strings.TrimPrefix(version, "v")

	// Remove any suffix after '-'
	if idx := strings.Index(version, "-"); idx != -1 {
		version = version[:idx]
	}

	// Parse as float
	var val float64
	_, _ = fmt.Sscanf(version, "%f", &val)
	return val
}
