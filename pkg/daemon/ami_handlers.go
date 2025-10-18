// AMI operations handlers for Universal AMI System (Phase 5.1 Week 2)
package daemon

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/scttfrdmn/cloudworkstation/pkg/templates"
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

	// AMI lifecycle management endpoints
	mux.HandleFunc("/api/v1/ami/cleanup", applyMiddleware(s.handleAMICleanup))
	mux.HandleFunc("/api/v1/ami/delete", applyMiddleware(s.handleAMIDelete))

	// AMI snapshot endpoints
	mux.HandleFunc("/api/v1/ami/snapshots", applyMiddleware(s.handleAMISnapshotsList))
	mux.HandleFunc("/api/v1/ami/snapshot/create", applyMiddleware(s.handleAMISnapshotCreate))
	mux.HandleFunc("/api/v1/ami/snapshot/restore", applyMiddleware(s.handleAMISnapshotRestore))
	mux.HandleFunc("/api/v1/ami/snapshot/delete", applyMiddleware(s.handleAMISnapshotDelete))

	// AMI freshness checking endpoints (v0.5.4 - Universal Version System)
	mux.HandleFunc("/api/v1/ami/check-freshness", applyMiddleware(s.handleAMICheckFreshness))
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

// AMI Lifecycle Management Handlers

// handleAMICleanup removes old and unused AMIs
// POST /api/v1/ami/cleanup
func (s *Server) handleAMICleanup(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		s.writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var request struct {
		MaxAge string `json:"max_age"`
		DryRun bool   `json:"dry_run"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		s.writeError(w, http.StatusBadRequest, fmt.Sprintf("invalid request body: %v", err))
		return
	}

	// Default max age if not specified
	if request.MaxAge == "" {
		request.MaxAge = "30d"
	}

	// Perform AMI cleanup using the AWS manager
	result, err := s.awsManager.CleanupOldAMIs(request.MaxAge, request.DryRun)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, fmt.Sprintf("AMI cleanup failed: %v", err))
		return
	}

	// Create response
	response := map[string]interface{}{
		"total_found":             result.TotalFound,
		"total_removed":           result.TotalRemoved,
		"storage_savings_monthly": result.StorageSavingsMonthly,
		"removed_amis":            result.RemovedAMIs,
		"dry_run":                 request.DryRun,
		"max_age":                 request.MaxAge,
		"cleanup_completed_at":    result.CompletedAt,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		s.writeError(w, http.StatusInternalServerError, err.Error())
	}
}

// handleAMIDelete deletes a specific AMI by ID
// POST /api/v1/ami/delete
func (s *Server) handleAMIDelete(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		s.writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var request struct {
		AMIID          string `json:"ami_id"`
		DeregisterOnly bool   `json:"deregister_only"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		s.writeError(w, http.StatusBadRequest, fmt.Sprintf("invalid request body: %v", err))
		return
	}

	if request.AMIID == "" {
		s.writeError(w, http.StatusBadRequest, "ami_id is required")
		return
	}

	// Delete AMI using the AWS manager
	result, err := s.awsManager.DeleteAMI(request.AMIID, request.DeregisterOnly)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, fmt.Sprintf("AMI deletion failed: %v", err))
		return
	}

	// Create response
	response := map[string]interface{}{
		"ami_id":                  request.AMIID,
		"status":                  result.Status,
		"deleted_snapshots":       result.DeletedSnapshots,
		"storage_savings_monthly": result.StorageSavingsMonthly,
		"deregister_only":         request.DeregisterOnly,
		"deletion_completed_at":   result.CompletedAt,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		s.writeError(w, http.StatusInternalServerError, err.Error())
	}
}

// AMI Snapshot Management Handlers

// handleAMISnapshotsList lists available snapshots
// POST /api/v1/ami/snapshots
func (s *Server) handleAMISnapshotsList(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		s.writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var filters struct {
		InstanceID string `json:"instance_id,omitempty"`
		MaxAge     string `json:"max_age,omitempty"`
		Region     string `json:"region,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&filters); err != nil {
		// Allow empty body for listing all snapshots
		filters = struct {
			InstanceID string `json:"instance_id,omitempty"`
			MaxAge     string `json:"max_age,omitempty"`
			Region     string `json:"region,omitempty"`
		}{}
	}

	// List snapshots using the AWS manager
	snapshots, err := s.awsManager.ListAMISnapshots(filters.InstanceID, filters.MaxAge)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, fmt.Sprintf("failed to list snapshots: %v", err))
		return
	}

	// Calculate total storage cost
	totalCost := 0.0
	for _, snapshot := range snapshots {
		totalCost += snapshot.StorageCostMonthly
	}

	// Create response
	response := map[string]interface{}{
		"snapshots":                  snapshots,
		"total_count":                len(snapshots),
		"total_storage_cost_monthly": totalCost,
		"filters":                    filters,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		s.writeError(w, http.StatusInternalServerError, err.Error())
	}
}

// handleAMISnapshotCreate creates a snapshot from an instance
// POST /api/v1/ami/snapshot/create
func (s *Server) handleAMISnapshotCreate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		s.writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var request struct {
		InstanceID  string `json:"instance_id"`
		Description string `json:"description"`
		NoReboot    bool   `json:"no_reboot"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		s.writeError(w, http.StatusBadRequest, fmt.Sprintf("invalid request body: %v", err))
		return
	}

	if request.InstanceID == "" {
		s.writeError(w, http.StatusBadRequest, "instance_id is required")
		return
	}

	// Create snapshot using the AWS manager
	result, err := s.awsManager.CreateInstanceSnapshot(request.InstanceID, request.Description, request.NoReboot)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, fmt.Sprintf("snapshot creation failed: %v", err))
		return
	}

	// Create response
	response := map[string]interface{}{
		"snapshot_id":                  result.SnapshotID,
		"volume_id":                    result.VolumeID,
		"volume_size":                  result.VolumeSize,
		"description":                  result.Description,
		"estimated_completion_minutes": result.EstimatedCompletionMinutes,
		"storage_cost_monthly":         result.StorageCostMonthly,
		"creation_initiated_at":        result.CreationInitiatedAt,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted) // 202 for async operation
	if err := json.NewEncoder(w).Encode(response); err != nil {
		s.writeError(w, http.StatusInternalServerError, err.Error())
	}
}

// handleAMISnapshotRestore creates an AMI from a snapshot
// POST /api/v1/ami/snapshot/restore
func (s *Server) handleAMISnapshotRestore(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		s.writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var request struct {
		SnapshotID   string `json:"snapshot_id"`
		Name         string `json:"name"`
		Description  string `json:"description"`
		Architecture string `json:"architecture"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		s.writeError(w, http.StatusBadRequest, fmt.Sprintf("invalid request body: %v", err))
		return
	}

	if request.SnapshotID == "" {
		s.writeError(w, http.StatusBadRequest, "snapshot_id is required")
		return
	}

	if request.Name == "" {
		s.writeError(w, http.StatusBadRequest, "name is required")
		return
	}

	// Default architecture if not specified
	if request.Architecture == "" {
		request.Architecture = "x86_64"
	}

	// Restore AMI from snapshot using the AWS manager
	result, err := s.awsManager.RestoreAMIFromSnapshot(request.SnapshotID, request.Name, request.Description, request.Architecture)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, fmt.Sprintf("AMI restore failed: %v", err))
		return
	}

	// Create response
	response := map[string]interface{}{
		"ami_id":                       result.AMIID,
		"name":                         result.Name,
		"snapshot_id":                  request.SnapshotID,
		"architecture":                 result.Architecture,
		"estimated_completion_minutes": result.EstimatedCompletionMinutes,
		"restore_initiated_at":         result.RestoreInitiatedAt,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted) // 202 for async operation
	if err := json.NewEncoder(w).Encode(response); err != nil {
		s.writeError(w, http.StatusInternalServerError, err.Error())
	}
}

// handleAMISnapshotDelete deletes a specific snapshot
// POST /api/v1/ami/snapshot/delete
func (s *Server) handleAMISnapshotDelete(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		s.writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var request struct {
		SnapshotID string `json:"snapshot_id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		s.writeError(w, http.StatusBadRequest, fmt.Sprintf("invalid request body: %v", err))
		return
	}

	if request.SnapshotID == "" {
		s.writeError(w, http.StatusBadRequest, "snapshot_id is required")
		return
	}

	// Delete snapshot using the AWS manager
	result, err := s.awsManager.DeleteSnapshot(request.SnapshotID)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, fmt.Sprintf("snapshot deletion failed: %v", err))
		return
	}

	// Create response
	response := map[string]interface{}{
		"snapshot_id":             request.SnapshotID,
		"volume_size":             result.VolumeSize,
		"storage_savings_monthly": result.StorageSavingsMonthly,
		"deletion_completed_at":   result.CompletedAt,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		s.writeError(w, http.StatusInternalServerError, err.Error())
	}
}

// handleAMICheckFreshness validates static AMI IDs against latest SSM values
// GET /api/v1/ami/check-freshness
func (s *Server) handleAMICheckFreshness(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	// Get template parser to access static AMI mappings
	parser := templates.NewTemplateParser()
	staticAMIs := parser.BaseAMIs

	// Check AMI freshness using AWS manager
	ctx := r.Context()
	results, err := s.awsManager.CheckAMIFreshness(ctx, staticAMIs)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, fmt.Sprintf("AMI freshness check failed: %v", err))
		return
	}

	// Format results
	outdatedCount := 0
	upToDateCount := 0
	noSSMSupportCount := 0

	for _, result := range results {
		if result.NeedsUpdate && result.IsOutdated {
			outdatedCount++
		} else if result.HasSSMSupport {
			upToDateCount++
		} else {
			noSSMSupportCount++
		}
	}

	// Create response
	response := map[string]interface{}{
		"total_checked":   len(results),
		"outdated":        outdatedCount,
		"up_to_date":      upToDateCount,
		"no_ssm_support":  noSSMSupportCount,
		"results":         results,
		"recommendation":  "Update outdated AMIs in pkg/templates/parser.go",
		"ssm_supported":   []string{"Ubuntu", "Amazon Linux", "Debian"},
		"static_only":     []string{"Rocky Linux", "RHEL", "Alpine"},
		"check_timestamp": time.Now().Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		s.writeError(w, http.StatusInternalServerError, err.Error())
	}
}
