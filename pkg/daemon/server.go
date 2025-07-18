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
	port         string
	httpServer   *http.Server
	stateManager *state.Manager
	userManager  *UserManager
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

	server := &Server{
		port:         port,
		stateManager: stateManager,
		userManager:  userManager,
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
			handler(w, r)
		}
	}
	
	// Combine all middleware
	applyMiddleware := func(handler http.HandlerFunc) http.HandlerFunc {
		return s.combineMiddleware(
			handler,
			jsonMiddleware,
			s.awsHeadersMiddleware,
			s.authMiddleware,
		)
	}

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

	status := types.DaemonStatus{
		Version:       "0.1.0",
		Status:        "running",
		StartTime:     time.Now(), // TODO: track actual start time
		ActiveOps:     0,          // TODO: implement operation tracking
		TotalRequests: 0,          // TODO: implement request counting
		AWSRegion:     awsRegion,
	}

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

	volume, err := s.awsManager.CreateVolume(req)
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
	err := s.awsManager.DeleteVolume(name)
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

	volume, err := s.awsManager.CreateStorage(req)
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
	err := s.awsManager.DeleteStorage(name)
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

	err := s.awsManager.AttachStorage(storageName, instanceName)
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

	err := s.awsManager.DetachStorage(storageName)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to detach storage: %v", err))
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// Helper methods

func (s *Server) writeError(w http.ResponseWriter, code int, message string) {
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(types.APIError{
		Code:    code,
		Message: message,
	})
}

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

// handleAuth handles API authentication endpoints
func (s *Server) handleAuth(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		s.handleGenerateAPIKey(w, r)
	case http.MethodGet:
		s.handleGetAuthStatus(w, r)
	case http.MethodDelete:
		s.handleRevokeAPIKey(w, r)
	default:
		s.writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
	}
}

// handleGenerateAPIKey generates a new API key
func (s *Server) handleGenerateAPIKey(w http.ResponseWriter, r *http.Request) {
	// Load current state
	state, err := s.stateManager.LoadState()
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "Failed to load state")
		return
	}

	// Generate a new random API key (64 characters)
	apiKey, err := generateAPIKey(64)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "Failed to generate API key")
		return
	}

	// Update state with the new API key
	state.Config.APIKey = apiKey
	state.Config.APIKeyCreated = time.Now()

	// Save state
	if err := s.stateManager.UpdateConfig(state.Config); err != nil {
		s.writeError(w, http.StatusInternalServerError, "Failed to save API key")
		return
	}

	// Return the new API key
	response := types.AuthResponse{
		APIKey:    apiKey,
		CreatedAt: state.Config.APIKeyCreated,
		Message:   "API key generated successfully. Keep this key secure.",
	}

	json.NewEncoder(w).Encode(response)
}

// handleGetAuthStatus returns information about the current auth status
func (s *Server) handleGetAuthStatus(w http.ResponseWriter, r *http.Request) {
	// This endpoint requires authentication
	if !isAuthenticated(r.Context()) {
		s.writeError(w, http.StatusUnauthorized, "Authentication required")
		return
	}

	// Load state to get API key info
	state, err := s.stateManager.LoadState()
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "Failed to load state")
		return
	}

	// Return auth status
	response := map[string]interface{}{
		"auth_enabled": state.Config.APIKey != "",
		"created_at":   state.Config.APIKeyCreated,
	}

	json.NewEncoder(w).Encode(response)
}

// handleRevokeAPIKey revokes the current API key
func (s *Server) handleRevokeAPIKey(w http.ResponseWriter, r *http.Request) {
	// This endpoint requires authentication
	if !isAuthenticated(r.Context()) {
		s.writeError(w, http.StatusUnauthorized, "Authentication required")
		return
	}

	// Load state
	state, err := s.stateManager.LoadState()
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "Failed to load state")
		return
	}

	// Clear API key
	state.Config.APIKey = ""
	state.Config.APIKeyCreated = time.Time{}

	// Save state
	if err := s.stateManager.UpdateConfig(state.Config); err != nil {
		s.writeError(w, http.StatusInternalServerError, "Failed to revoke API key")
		return
	}

	response := map[string]string{
		"message": "API key revoked successfully",
	}
	json.NewEncoder(w).Encode(response)
}

// generateAPIKey generates a random API key of the specified length
func generateAPIKey(length int) (string, error) {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, length)
	for i := range b {
		n, err := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
		if err != nil {
			return "", err
		}
		b[i] = charset[n.Int64()]
	}
	return string(b), nil
}