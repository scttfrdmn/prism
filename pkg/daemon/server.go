package daemon

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"log"
	"math/big"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/scttfrdmn/cloudworkstation/pkg/aws"
	"github.com/scttfrdmn/cloudworkstation/pkg/state"
	"github.com/scttfrdmn/cloudworkstation/pkg/types"
	"github.com/scttfrdmn/cloudworkstation/pkg/usermgmt"
)

// Server represents the CloudWorkstation daemon server
type Server struct {
	port           string
	httpServer     *http.Server
	stateManager   *state.Manager
	userManager    *UserManager
	statusTracker  *StatusTracker
	versionManager *APIVersionManager
	// Future: add cost tracker, idle monitor, etc.
}

// NewServer creates a new daemon server
func NewServer(port string) (*Server, error) {
	if port == "" {
		port = "8080"
	}

	// Initialize state manager
	stateManager, err := state.NewManager()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize state manager: %w", err)
	}

	// Initialize user manager
	userManager := NewUserManager()
	if err := userManager.Initialize(); err != nil {
		return nil, fmt.Errorf("failed to initialize user manager: %w", err)
	}

	// Initialize status tracker
	statusTracker := NewStatusTracker()

	// Initialize API version manager
	versionManager := NewAPIVersionManager("/api")

	server := &Server{
		port:           port,
		stateManager:   stateManager,
		userManager:    userManager,
		statusTracker:  statusTracker,
		versionManager: versionManager,
	}

	// Setup HTTP routes
	mux := http.NewServeMux()
	server.setupRoutes(mux)

	server.httpServer = &http.Server{
		Addr:         ":" + port,
		Handler:      mux,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	return server, nil
}

// Start starts the daemon server
func (s *Server) Start() error {
	log.Printf("Starting CloudWorkstation daemon on port %s", s.port)

	// Handle graceful shutdown
	go func() {
		c := make(chan os.Signal, 1)
		signal.Notify(c, os.Interrupt, syscall.SIGTERM)
		<-c
		log.Println("Shutting down daemon...")
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		s.httpServer.Shutdown(ctx)
	}()

	return s.httpServer.ListenAndServe()
}

// Stop stops the daemon server
func (s *Server) Stop() error {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	return s.httpServer.Shutdown(ctx)
}

// setupRoutes configures all HTTP routes
func (s *Server) setupRoutes(mux *http.ServeMux) {
	// Define middleware for JSON responses and logging
	jsonMiddleware := func(handler http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			log.Printf("%s %s", r.Method, r.URL.Path)
			// Record the request in status tracker
			s.statusTracker.RecordRequest()
			handler(w, r)
		}
	}
	
	// Operation tracking middleware
	operationTrackingMiddleware := func(handler http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			// Determine operation type from path
			opType := extractOperationType(r.URL.Path)
			
			// Start tracking this operation with type information
			opID := s.statusTracker.StartOperationWithType(opType)
			
			// Enhance logging
			log.Printf("Operation %d (%s) started: %s %s", opID, opType, r.Method, r.URL.Path)
			
			// Record start time for duration tracking
			startTime := time.Now()
			
			// Ensure operation is always marked as completed
			defer func() {
				s.statusTracker.EndOperationWithType(opType)
				log.Printf("Operation %d (%s) completed in %v", opID, opType, time.Since(startTime))
			}()
			
			// Call the handler
			handler(w, r)
		}
	}

	// Add API versioning middlewares
	versionHeaderMiddleware := func(handler http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			// Extract version from request path
			requestedVersion := s.versionManager.ExtractVersionFromPath(r.URL.Path)
			if requestedVersion == "" {
				requestedVersion = s.versionManager.GetDefaultVersion()
			}
			
			// Add version headers to response
			w.Header().Set("X-API-Version", requestedVersion)
			w.Header().Set("X-API-Latest-Version", s.versionManager.GetLatestVersion())
			w.Header().Set("X-API-Stable-Version", s.versionManager.GetStableVersion())
			
			// Add version to request context for handlers to use
			ctx := r.Context()
			ctx = setAPIVersion(ctx, requestedVersion)
			r = r.WithContext(ctx)
			
			handler(w, r)
		}
	}
	
	// Combine all middleware
	applyMiddleware := func(handler http.HandlerFunc) http.HandlerFunc {
		return s.combineMiddleware(
			handler,
			jsonMiddleware,
			operationTrackingMiddleware,
			versionHeaderMiddleware,
			s.awsHeadersMiddleware,
			s.authMiddleware,
		)
	}

	// API version information endpoint
	mux.HandleFunc("/api/versions", applyMiddleware(s.handleAPIVersions))
	
	// Register v1 endpoints
	s.registerV1Routes(mux, applyMiddleware)
	
	// API path matcher to handle any valid API request
	// This allows proper versioning of new paths that may be added in the future
	mux.HandleFunc("/api/", applyMiddleware(s.handleUnknownAPI))
}

// registerV1Routes registers all API v1 routes
func (s *Server) registerV1Routes(mux *http.ServeMux, applyMiddleware func(http.HandlerFunc) http.HandlerFunc) {
	// Health check
	mux.HandleFunc("/api/v1/ping", applyMiddleware(s.handlePing))
	mux.HandleFunc("/api/v1/status", applyMiddleware(s.handleStatus))

	// Authentication
	mux.HandleFunc("/api/v1/auth", applyMiddleware(s.handleAuth))
	
	// User authentication
	mux.HandleFunc("/api/v1/authenticate", applyMiddleware(s.handleAuthenticate))

	// User management
	mux.HandleFunc("/api/v1/users", applyMiddleware(s.handleUsers))
	mux.HandleFunc("/api/v1/users/", applyMiddleware(s.handleUserOperations))

	// Group management
	mux.HandleFunc("/api/v1/groups", applyMiddleware(s.handleGroups))
	mux.HandleFunc("/api/v1/groups/", applyMiddleware(s.handleGroupOperations))

	// Instance operations
	mux.HandleFunc("/api/v1/instances", applyMiddleware(s.handleInstances))
	mux.HandleFunc("/api/v1/instances/", applyMiddleware(s.handleInstanceOperations))

	// Template operations
	mux.HandleFunc("/api/v1/templates", applyMiddleware(s.handleTemplates))
	mux.HandleFunc("/api/v1/templates/", applyMiddleware(s.handleTemplateInfo))

	// Volume operations
	mux.HandleFunc("/api/v1/volumes", applyMiddleware(s.handleVolumes))
	mux.HandleFunc("/api/v1/volumes/", applyMiddleware(s.handleVolumeOperations))

	// Storage operations
	mux.HandleFunc("/api/v1/storage", applyMiddleware(s.handleStorage))
	mux.HandleFunc("/api/v1/storage/", applyMiddleware(s.handleStorageOperations))
}

// HTTP handlers

// getAPIVersion function is defined in context.go

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

func (s *Server) handlePing(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

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
	status := s.statusTracker.GetStatus("0.1.0", awsRegion)
	status.CurrentProfile = currentProfile

	json.NewEncoder(w).Encode(status)
}

func (s *Server) handleInstances(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.handleListInstances(w, r)
	case http.MethodPost:
		s.handleLaunchInstance(w, r)
	default:
		s.writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
	}
}

func (s *Server) handleListInstances(w http.ResponseWriter, r *http.Request) {
	state, err := s.stateManager.LoadState()
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "Failed to load state")
		return
	}

	instances := make([]types.Instance, 0, len(state.Instances))
	totalCost := 0.0

	for _, instance := range state.Instances {
		instances = append(instances, instance)
		if instance.State == "running" {
			totalCost += instance.EstimatedDailyCost
		}
	}

	response := types.ListResponse{
		Instances: instances,
		TotalCost: totalCost,
	}

	json.NewEncoder(w).Encode(response)
}

func (s *Server) handleLaunchInstance(w http.ResponseWriter, r *http.Request) {
	var req types.LaunchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	
	// Use AWS manager from request and handle launch
	var instance *types.Instance
	s.withAWSManager(w, r, func(awsManager *aws.Manager) error {
		// Delegate to AWS manager
		var err error
		instance, err = awsManager.LaunchInstance(req)
		return err
	})
	
	// If instance is nil, withAWSManager already wrote an error response
	if instance == nil {
		return
	}

	// Save state
	if err := s.stateManager.SaveInstance(*instance); err != nil {
		s.writeError(w, http.StatusInternalServerError, "Failed to save instance state")
		return
	}

	response := types.LaunchResponse{
		Instance:       *instance,
		Message:        fmt.Sprintf("Instance %s launched successfully", instance.Name),
		EstimatedCost:  fmt.Sprintf("$%.2f/day", instance.EstimatedDailyCost),
		ConnectionInfo: fmt.Sprintf("ssh ubuntu@%s", instance.PublicIP),
	}

	json.NewEncoder(w).Encode(response)
}

func (s *Server) handleInstanceOperations(w http.ResponseWriter, r *http.Request) {
	// Parse instance name from path
	path := r.URL.Path[len("/api/v1/instances/"):]
	parts := splitPath(path)
	if len(parts) == 0 {
		s.writeError(w, http.StatusBadRequest, "Missing instance name")
		return
	}

	instanceName := parts[0]

	if len(parts) == 1 {
		// Operations on the instance itself
		switch r.Method {
		case http.MethodGet:
			s.handleGetInstance(w, r, instanceName)
		case http.MethodDelete:
			s.handleDeleteInstance(w, r, instanceName)
		default:
			s.writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		}
	} else if len(parts) == 2 {
		// Sub-operations
		operation := parts[1]
		switch operation {
		case "start":
			s.handleStartInstance(w, r, instanceName)
		case "stop":
			s.handleStopInstance(w, r, instanceName)
		case "connect":
			s.handleConnectInstance(w, r, instanceName)
		default:
			s.writeError(w, http.StatusNotFound, "Unknown operation")
		}
	} else {
		s.writeError(w, http.StatusNotFound, "Invalid path")
	}
}

func (s *Server) handleGetInstance(w http.ResponseWriter, r *http.Request, name string) {
	state, err := s.stateManager.LoadState()
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "Failed to load state")
		return
	}

	instance, exists := state.Instances[name]
	if !exists {
		s.writeError(w, http.StatusNotFound, "Instance not found")
		return
	}

	json.NewEncoder(w).Encode(instance)
}

func (s *Server) handleDeleteInstance(w http.ResponseWriter, r *http.Request, name string) {
	s.withAWSManager(w, r, func(awsManager *aws.Manager) error {
		return awsManager.DeleteInstance(name)
	})

	// Remove from state - only if we didn't error out above
	if err := s.stateManager.RemoveInstance(name); err != nil {
		s.writeError(w, http.StatusInternalServerError, "Failed to update state")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleStartInstance(w http.ResponseWriter, r *http.Request, name string) {
	if r.Method != http.MethodPost {
		s.writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	s.withAWSManager(w, r, func(awsManager *aws.Manager) error {
		return awsManager.StartInstance(name)
	})

	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleStopInstance(w http.ResponseWriter, r *http.Request, name string) {
	if r.Method != http.MethodPost {
		s.writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	s.withAWSManager(w, r, func(awsManager *aws.Manager) error {
		return awsManager.StopInstance(name)
	})

	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleConnectInstance(w http.ResponseWriter, r *http.Request, name string) {
	if r.Method != http.MethodGet {
		s.writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	var connectionInfo string
	s.withAWSManager(w, r, func(awsManager *aws.Manager) error {
		var err error
		connectionInfo, err = awsManager.GetConnectionInfo(name)
		return err
	})

	if connectionInfo == "" {
		// Error was already handled by withAWSManager
		return
	}

	response := map[string]string{
		"connection_info": connectionInfo,
	}
	json.NewEncoder(w).Encode(response)
}

func (s *Server) handleTemplates(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	var templates map[string]types.Template
	s.withAWSManager(w, r, func(awsManager *aws.Manager) error {
		templates = awsManager.GetTemplates()
		return nil
	})

	if templates != nil {
		json.NewEncoder(w).Encode(templates)
	}
}

func (s *Server) handleTemplateInfo(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	templateName := r.URL.Path[len("/api/v1/templates/"):]
	
	var template *types.Template
	s.withAWSManager(w, r, func(awsManager *aws.Manager) error {
		var err error
		template, err = awsManager.GetTemplate(templateName)
		return err
	})

	if template != nil {
		json.NewEncoder(w).Encode(template)
	}
}

func (s *Server) handleVolumes(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.handleListVolumes(w, r)
	case http.MethodPost:
		s.handleCreateVolume(w, r)
	default:
		s.writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
	}
}

func (s *Server) handleListVolumes(w http.ResponseWriter, r *http.Request) {
	state, err := s.stateManager.LoadState()
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "Failed to load state")
		return
	}

	json.NewEncoder(w).Encode(state.Volumes)
}

func (s *Server) handleCreateVolume(w http.ResponseWriter, r *http.Request) {
	var req types.VolumeCreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	awsManager, err := s.createAWSManagerFromRequest(r)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to create AWS manager: %v", err))
		return
	}

	volume, err := awsManager.CreateVolume(req)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to create volume: %v", err))
		return
	}

	// Save state
	if err := s.stateManager.SaveVolume(*volume); err != nil {
		s.writeError(w, http.StatusInternalServerError, "Failed to save volume state")
		return
	}

	json.NewEncoder(w).Encode(volume)
}

func (s *Server) handleVolumeOperations(w http.ResponseWriter, r *http.Request) {
	volumeName := r.URL.Path[len("/api/v1/volumes/"):]
	
	switch r.Method {
	case http.MethodGet:
		s.handleGetVolume(w, r, volumeName)
	case http.MethodDelete:
		s.handleDeleteVolume(w, r, volumeName)
	default:
		s.writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
	}
}

func (s *Server) handleGetVolume(w http.ResponseWriter, r *http.Request, name string) {
	state, err := s.stateManager.LoadState()
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "Failed to load state")
		return
	}

	volume, exists := state.Volumes[name]
	if !exists {
		s.writeError(w, http.StatusNotFound, "Volume not found")
		return
	}

	json.NewEncoder(w).Encode(volume)
}

func (s *Server) handleDeleteVolume(w http.ResponseWriter, r *http.Request, name string) {
	awsManager, err := s.createAWSManagerFromRequest(r)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to create AWS manager: %v", err))
		return
	}

	err = awsManager.DeleteVolume(name)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to delete volume: %v", err))
		return
	}

	// Remove from state
	if err := s.stateManager.RemoveVolume(name); err != nil {
		s.writeError(w, http.StatusInternalServerError, "Failed to update state")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleStorage(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.handleListStorage(w, r)
	case http.MethodPost:
		s.handleCreateStorage(w, r)
	default:
		s.writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
	}
}

func (s *Server) handleListStorage(w http.ResponseWriter, r *http.Request) {
	state, err := s.stateManager.LoadState()
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "Failed to load state")
		return
	}

	json.NewEncoder(w).Encode(state.EBSVolumes)
}

func (s *Server) handleCreateStorage(w http.ResponseWriter, r *http.Request) {
	var req types.StorageCreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	awsManager, err := s.createAWSManagerFromRequest(r)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to create AWS manager: %v", err))
		return
	}

	volume, err := awsManager.CreateStorage(req)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to create storage: %v", err))
		return
	}

	// Save state
	if err := s.stateManager.SaveEBSVolume(*volume); err != nil {
		s.writeError(w, http.StatusInternalServerError, "Failed to save storage state")
		return
	}

	json.NewEncoder(w).Encode(volume)
}

func (s *Server) handleStorageOperations(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path[len("/api/v1/storage/"):]
	parts := splitPath(path)
	if len(parts) == 0 {
		s.writeError(w, http.StatusBadRequest, "Missing storage name")
		return
	}

	storageName := parts[0]

	if len(parts) == 1 {
		switch r.Method {
		case http.MethodGet:
			s.handleGetStorage(w, r, storageName)
		case http.MethodDelete:
			s.handleDeleteStorage(w, r, storageName)
		default:
			s.writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		}
	} else if len(parts) == 2 {
		operation := parts[1]
		switch operation {
		case "attach":
			s.handleAttachStorage(w, r, storageName)
		case "detach":
			s.handleDetachStorage(w, r, storageName)
		default:
			s.writeError(w, http.StatusNotFound, "Unknown operation")
		}
	}
}

func (s *Server) handleGetStorage(w http.ResponseWriter, r *http.Request, name string) {
	state, err := s.stateManager.LoadState()
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "Failed to load state")
		return
	}

	storage, exists := state.EBSVolumes[name]
	if !exists {
		s.writeError(w, http.StatusNotFound, "Storage not found")
		return
	}

	json.NewEncoder(w).Encode(storage)
}

func (s *Server) handleDeleteStorage(w http.ResponseWriter, r *http.Request, name string) {
	awsManager, err := s.createAWSManagerFromRequest(r)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to create AWS manager: %v", err))
		return
	}

	err = awsManager.DeleteStorage(name)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to delete storage: %v", err))
		return
	}

	// Remove from state
	if err := s.stateManager.RemoveEBSVolume(name); err != nil {
		s.writeError(w, http.StatusInternalServerError, "Failed to update state")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleAttachStorage(w http.ResponseWriter, r *http.Request, storageName string) {
	if r.Method != http.MethodPost {
		s.writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	var req map[string]string
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	instanceName, ok := req["instance"]
	if !ok {
		s.writeError(w, http.StatusBadRequest, "Missing instance name")
		return
	}

	awsManager, err := s.createAWSManagerFromRequest(r)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to create AWS manager: %v", err))
		return
	}

	err = awsManager.AttachStorage(storageName, instanceName)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to attach storage: %v", err))
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleDetachStorage(w http.ResponseWriter, r *http.Request, storageName string) {
	if r.Method != http.MethodPost {
		s.writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	awsManager, err := s.createAWSManagerFromRequest(r)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to create AWS manager: %v", err))
		return
	}

	err = awsManager.DetachStorage(storageName)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to detach storage: %v", err))
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// Helper methods

// writeError method is implemented in error_handler.go

func splitPath(path string) []string {
	if path == "" {
		return []string{}
	}
	// Remove trailing slash and split
	if path[len(path)-1] == '/' {
		path = path[:len(path)-1]
	}
	return strings.Split(path, "/")
}

// extractOperationType extracts an operation type string from a URL path
// Example: /api/v1/instances/create -> "InstanceCreate"
func extractOperationType(path string) string {
	parts := splitPath(path)
	
	if len(parts) < 3 {
		return "Unknown"
	}
	
	// Skip the /api/v1 prefix
	if parts[0] == "" && parts[1] == "api" && parts[2] == "v1" {
		parts = parts[3:]
	} else if parts[0] == "api" && parts[1] == "v1" {
		parts = parts[2:]
	}
	
	if len(parts) == 0 {
		return "Root"
	}
	
	// Extract resource type (first part)
	resourceType := strings.Title(parts[0])
	if len(resourceType) > 0 && resourceType[len(resourceType)-1] == 's' {
		// Convert plural to singular (instances -> instance)
		resourceType = resourceType[:len(resourceType)-1]
	}
	
	// If there's an ID and operation, use those
	if len(parts) >= 3 {
		operation := strings.Title(parts[2])
		return resourceType + operation
	}
	
	// If there's just an ID, determine operation based on HTTP method
	if len(parts) == 2 {
		return resourceType + "Operation"
	}
	
	// Otherwise just return the resource type
	return resourceType
}

// Auth handlers are implemented in auth.go