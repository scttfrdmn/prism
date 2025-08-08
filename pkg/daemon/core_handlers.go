package daemon

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/scttfrdmn/cloudworkstation/pkg/types"
	"github.com/scttfrdmn/cloudworkstation/pkg/version"
)

// handleAPIVersions returns information about supported API versions
func (s *Server) handleAPIVersions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}
	
	apiVersions := s.versionManager.GetSupportedVersions()
	
	// Convert internal APIVersion to public APIVersionInfo
	versionInfos := make([]types.APIVersionInfo, 0, len(apiVersions))
	for _, v := range apiVersions {
		versionInfos = append(versionInfos, types.APIVersionInfo{
			Version:         v.Version,
			Status:          v.Status,
			IsDefault:       v.IsDefault,
			ReleaseDate:     v.ReleaseDate,
			DeprecationDate: v.DeprecationDate,
			SunsetDate:      v.SunsetDate,
			DocsURL:         v.DocsURL,
		})
	}
	
	response := types.APIVersionResponse{
		Versions:       versionInfos,
		DefaultVersion: s.versionManager.GetDefaultVersion(),
		StableVersion:  s.versionManager.GetStableVersion(),
		LatestVersion:  s.versionManager.GetLatestVersion(),
		DocsBaseURL:    "https://docs.cloudworkstation.dev/api",
	}
	
	json.NewEncoder(w).Encode(response)
}

// handleUnknownAPI handles requests to unknown API endpoints
func (s *Server) handleUnknownAPI(w http.ResponseWriter, r *http.Request) {
	// Generate request ID for tracking
	requestID := fmt.Sprintf("req_%d", time.Now().UnixNano())
	
	// Extract API version from path
	version := s.versionManager.ExtractVersionFromPath(r.URL.Path)
	
	// Check if version exists but endpoint doesn't
	if version != "" {
		// Valid version but unknown endpoint
		errorResponse := types.APIErrorResponse{
			Code:       "endpoint_not_found",
			Status:     http.StatusNotFound,
			Message:    "The requested API endpoint does not exist",
			Details:    fmt.Sprintf("No handler found for %s %s", r.Method, r.URL.Path),
			RequestID:  requestID,
			APIVersion: version,
			DocsURL:    fmt.Sprintf("https://docs.cloudworkstation.dev/api/%s", version),
		}
		
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(errorResponse)
		return
	}
	
	// No version specified, convert supported versions to APIVersionInfo
	apiVersions := s.versionManager.GetSupportedVersions()
	versionInfos := make([]types.APIVersionInfo, 0, len(apiVersions))
	for _, v := range apiVersions {
		versionInfos = append(versionInfos, types.APIVersionInfo{
			Version:         v.Version,
			Status:          v.Status,
			IsDefault:       v.IsDefault,
			ReleaseDate:     v.ReleaseDate,
			DeprecationDate: v.DeprecationDate,
			SunsetDate:      v.SunsetDate,
			DocsURL:         v.DocsURL,
		})
	}
	
	// Create error response with available versions
	errorResponse := types.APIErrorResponse{
		Code:      "version_required",
		Status:    http.StatusBadRequest,
		Message:   "No API version specified",
		Details:   "Please specify an API version in the URL path, e.g., /api/v1/...",
		RequestID: requestID,
		DocsURL:   "https://docs.cloudworkstation.dev/api",
	}
	
	w.WriteHeader(http.StatusBadRequest)
	
	// Include version information in response body
	responseBody := map[string]interface{}{
		"error":             errorResponse,
		"available_versions": versionInfos,
		"default_version":    s.versionManager.GetDefaultVersion(),
		"stable_version":     s.versionManager.GetStableVersion(),
	}
	
	json.NewEncoder(w).Encode(responseBody)
}

// handlePing handles health check requests
func (s *Server) handlePing(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// handleStatus handles status requests
func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}
	
	// Get region from context or create AWS manager to get default
	awsRegion := getAWSRegion(r.Context())
	if awsRegion == "" {
		// If no region specified in request, get default from a new AWS manager
		awsManager, err := s.createAWSManagerFromRequest(r)
		if err == nil {
			awsRegion = awsManager.GetDefaultRegion()
		} else {
			awsRegion = "unknown"
		}
	}

	// Get current profile from request
	currentProfile := getAWSProfile(r.Context())

	// Use status tracker to get current daemon status
	status := s.statusTracker.GetStatus(version.GetVersion(), awsRegion, currentProfile)
	status.CurrentProfile = currentProfile

	json.NewEncoder(w).Encode(status)
}

// handleShutdown handles daemon shutdown requests
func (s *Server) handleShutdown(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		s.writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}
	
	// Respond immediately before shutting down
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"status": "shutting_down", 
		"message": "Daemon shutdown initiated",
	})
	
	// Shutdown in a goroutine to allow response to be sent
	go func() {
		time.Sleep(100 * time.Millisecond) // Allow response to be sent
		if err := s.Stop(); err != nil {
			log.Printf("Error during shutdown: %v", err)
		}
		os.Exit(0)
	}()
}