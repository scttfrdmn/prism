// AMI operations handlers for Universal AMI System (Phase 5.1 Week 2)
package daemon

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/scttfrdmn/cloudworkstation/pkg/types"
)

// RegisterAMIRoutes registers all AMI-related API routes
func (s *Server) RegisterAMIRoutes(mux *http.ServeMux, applyMiddleware func(http.HandlerFunc) http.HandlerFunc) {
	// AMI resolution endpoints
	mux.HandleFunc("/api/v1/ami/resolve/", applyMiddleware(s.handleAMIResolve))
	mux.HandleFunc("/api/v1/ami/test", applyMiddleware(s.handleAMITest))
	mux.HandleFunc("/api/v1/ami/costs/", applyMiddleware(s.handleAMICosts))
	mux.HandleFunc("/api/v1/ami/preview/", applyMiddleware(s.handleAMIPreview))

	// AMI creation and management endpoints
	mux.HandleFunc("/api/v1/ami/create", applyMiddleware(s.handleAMICreate))
	mux.HandleFunc("/api/v1/ami/status/", applyMiddleware(s.handleAMIStatus))
	mux.HandleFunc("/api/v1/ami/list", applyMiddleware(s.handleAMIList))
}

// handleAMIResolve resolves AMI for a specific template
// GET /api/v1/ami/resolve/{template_name}
func (s *Server) handleAMIResolve(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	// Extract template name from URL path
	pathParts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(pathParts) < 5 || pathParts[4] == "" {
		s.writeError(w, http.StatusBadRequest, "template name required in URL path")
		return
	}

	templateName := pathParts[4]

	// Optional query parameters
	showDetails := r.URL.Query().Get("details") == "true"
	targetRegion := r.URL.Query().Get("region")

	// Use current region if none specified
	if targetRegion == "" {
		targetRegion = s.awsManager.GetDefaultRegion()
	}

	// Resolve AMI for the template
	result, err := s.awsManager.ResolveAMIForTemplate(templateName)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, fmt.Sprintf("AMI resolution failed: %v", err))
		return
	}

	// Create response
	response := map[string]interface{}{
		"template_name":                templateName,
		"target_region":                result.TargetRegion,
		"resolution_method":            result.ResolutionMethod,
		"ami_id":                       "",
		"launch_time_estimate_seconds": int(result.LaunchTime.Seconds()),
		"cost_savings":                 result.CostSavings,
	}

	if result.AMI != nil {
		response["ami_id"] = result.AMI.AMIID
		response["ami_name"] = result.AMI.Name
		response["ami_architecture"] = result.AMI.Architecture
		response["ami_description"] = result.AMI.Description

		if showDetails {
			response["ami_details"] = map[string]interface{}{
				"creation_date":    result.AMI.CreationDate,
				"owner_id":         result.AMI.Owner,
				"public":           result.AMI.Public,
				"marketplace_cost": result.AMI.MarketplaceCost,
				// Platform, virtualization, and root device info not available in current AMIInfo struct
			}
		}
	}

	if result.Warning != "" {
		response["warning"] = result.Warning
	}

	// Error information would be handled through the error return

	if showDetails && len(result.FallbackChain) > 0 {
		response["fallback_chain"] = result.FallbackChain
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		s.writeError(w, http.StatusInternalServerError, err.Error())
	}
}

// handleAMITest tests AMI availability across regions for a template
// POST /api/v1/ami/test
func (s *Server) handleAMITest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		s.writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var request struct {
		TemplateName string   `json:"template_name"`
		Regions      []string `json:"regions,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		s.writeError(w, http.StatusBadRequest, fmt.Sprintf("invalid request body: %v", err))
		return
	}

	if request.TemplateName == "" {
		s.writeError(w, http.StatusBadRequest, "template_name is required")
		return
	}

	// Test AMI availability
	result, err := s.awsManager.TestAMIAvailability(request.TemplateName, request.Regions)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, fmt.Sprintf("AMI availability test failed: %v", err))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(result); err != nil {
		s.writeError(w, http.StatusInternalServerError, err.Error())
	}
}

// handleAMICosts provides cost analysis for AMI vs script deployment
// GET /api/v1/ami/costs/{template_name}
func (s *Server) handleAMICosts(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	// Extract template name from URL path
	pathParts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(pathParts) < 5 || pathParts[4] == "" {
		s.writeError(w, http.StatusBadRequest, "template name required in URL path")
		return
	}

	templateName := pathParts[4]

	// Get cost analysis
	analysis, err := s.awsManager.GetAMICostAnalysis(templateName)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, fmt.Sprintf("cost analysis failed: %v", err))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(analysis); err != nil {
		s.writeError(w, http.StatusInternalServerError, err.Error())
	}
}

// handleAMIPreview shows what would happen during AMI resolution without executing
// GET /api/v1/ami/preview/{template_name}
func (s *Server) handleAMIPreview(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	// Extract template name from URL path
	pathParts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(pathParts) < 5 || pathParts[4] == "" {
		s.writeError(w, http.StatusBadRequest, "template name required in URL path")
		return
	}

	templateName := pathParts[4]

	// Get preview of AMI resolution
	preview, err := s.awsManager.PreviewAMIResolution(templateName)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, fmt.Sprintf("AMI preview failed: %v", err))
		return
	}

	// Create response
	response := map[string]interface{}{
		"template_name":                templateName,
		"target_region":                preview.TargetRegion,
		"resolution_method":            preview.ResolutionMethod,
		"launch_time_estimate_seconds": int(preview.LaunchTime.Seconds()),
		"fallback_chain":               preview.FallbackChain,
	}

	if preview.Warning != "" {
		response["warning"] = preview.Warning
	}

	// Error information would be handled through the error return

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		s.writeError(w, http.StatusInternalServerError, err.Error())
	}
}

// handleAMICreate initiates AMI creation for a template
// POST /api/v1/ami/create
func (s *Server) handleAMICreate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		s.writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var request types.AMICreationRequest

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		s.writeError(w, http.StatusBadRequest, fmt.Sprintf("invalid request body: %v", err))
		return
	}

	// Validate request
	if request.TemplateName == "" {
		s.writeError(w, http.StatusBadRequest, "template_name is required")
		return
	}

	if request.InstanceID == "" {
		s.writeError(w, http.StatusBadRequest, "instance_id is required")
		return
	}

	// Create AMI from instance using the AWS manager
	result, err := s.awsManager.CreateAMIFromInstance(&request)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, fmt.Sprintf("AMI creation failed: %v", err))
		return
	}

	// Create response from result
	response := map[string]interface{}{
		"creation_id":                  result.AMIID,
		"ami_id":                       result.AMIID,
		"template_name":                request.TemplateName,
		"instance_id":                  request.InstanceID,
		"target_regions":               request.MultiRegion,
		"status":                       string(result.Status),
		"message":                      "AMI creation initiated successfully",
		"estimated_completion_minutes": 12, // Typical AMI creation time
		"storage_cost":                 result.StorageCost,
		"creation_cost":                result.CreationCost,
	}

	// Add region results if multi-region deployment
	if result.RegionResults != nil {
		response["region_results"] = result.RegionResults
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted) // 202 for async operation
	if err := json.NewEncoder(w).Encode(response); err != nil {
		s.writeError(w, http.StatusInternalServerError, err.Error())
	}
}

// handleAMIStatus checks the status of AMI creation
// GET /api/v1/ami/status/{creation_id}
func (s *Server) handleAMIStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	// Extract creation ID from URL path
	pathParts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(pathParts) < 5 || pathParts[4] == "" {
		s.writeError(w, http.StatusBadRequest, "creation ID required in URL path")
		return
	}

	creationID := pathParts[4]

	// Get AMI creation status using the AWS manager
	result, err := s.awsManager.GetAMICreationStatus(creationID)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, fmt.Sprintf("failed to get AMI status: %v", err))
		return
	}

	// Calculate progress percentage based on status
	progress := 0
	switch result.Status {
	case types.AMICreationPending:
		progress = 10
	case types.AMICreationInProgress:
		progress = 50
	case types.AMICreationCompleted:
		progress = 100
	case types.AMICreationFailed:
		progress = 0
	}

	// Create response
	response := map[string]interface{}{
		"creation_id":                  creationID,
		"ami_id":                       result.AMIID,
		"status":                       string(result.Status),
		"progress":                     progress,
		"message":                      "AMI creation in progress",
		"estimated_completion_minutes": 8, // Remaining time estimate
		"elapsed_time_minutes":         int(result.CreationTime.Minutes()),
		"storage_cost":                 result.StorageCost,
		"creation_cost":                result.CreationCost,
	}

	// Add region results if available
	if result.RegionResults != nil {
		response["region_results"] = result.RegionResults
	}

	// Add community sharing results if available
	if result.CommunitySharing != nil {
		response["community_sharing"] = result.CommunitySharing
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		s.writeError(w, http.StatusInternalServerError, err.Error())
	}
}

// handleAMIList lists user-created AMIs
// GET /api/v1/ami/list
func (s *Server) handleAMIList(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	// Get user AMIs using the AWS manager
	userAMIs, err := s.awsManager.ListUserAMIs()
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, fmt.Sprintf("failed to list user AMIs: %v", err))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(userAMIs); err != nil {
		s.writeError(w, http.StatusInternalServerError, err.Error())
	}
}
