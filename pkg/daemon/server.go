package daemon

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/scttfrdmn/cloudworkstation/pkg/aws"
	"github.com/scttfrdmn/cloudworkstation/pkg/state"
	"github.com/scttfrdmn/cloudworkstation/pkg/types"
)

// Server represents the CloudWorkstation daemon server
type Server struct {
	port         string
	httpServer   *http.Server
	awsManager   *aws.Manager
	stateManager *state.Manager
	// Future: add cost tracker, idle monitor, etc.
}

// NewServer creates a new daemon server
func NewServer(port string) (*Server, error) {
	if port == "" {
		port = "8080"
	}

	// Initialize AWS manager
	awsManager, err := aws.NewManager()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize AWS manager: %w", err)
	}

	// Initialize state manager
	stateManager, err := state.NewManager()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize state manager: %w", err)
	}

	server := &Server{
		port:         port,
		awsManager:   awsManager,
		stateManager: stateManager,
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
	// Middleware for JSON responses and logging
	jsonMiddleware := func(handler http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			log.Printf("%s %s", r.Method, r.URL.Path)
			handler(w, r)
		}
	}

	// Health check
	mux.HandleFunc("/api/v1/ping", jsonMiddleware(s.handlePing))
	mux.HandleFunc("/api/v1/status", jsonMiddleware(s.handleStatus))

	// Instance operations
	mux.HandleFunc("/api/v1/instances", jsonMiddleware(s.handleInstances))
	mux.HandleFunc("/api/v1/instances/", jsonMiddleware(s.handleInstanceOperations))

	// Template operations
	mux.HandleFunc("/api/v1/templates", jsonMiddleware(s.handleTemplates))
	mux.HandleFunc("/api/v1/templates/", jsonMiddleware(s.handleTemplateInfo))

	// Volume operations
	mux.HandleFunc("/api/v1/volumes", jsonMiddleware(s.handleVolumes))
	mux.HandleFunc("/api/v1/volumes/", jsonMiddleware(s.handleVolumeOperations))

	// Storage operations
	mux.HandleFunc("/api/v1/storage", jsonMiddleware(s.handleStorage))
	mux.HandleFunc("/api/v1/storage/", jsonMiddleware(s.handleStorageOperations))
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

	status := types.DaemonStatus{
		Version:       "0.1.0",
		Status:        "running",
		StartTime:     time.Now(), // TODO: track actual start time
		ActiveOps:     0,          // TODO: implement operation tracking
		TotalRequests: 0,          // TODO: implement request counting
		AWSRegion:     s.awsManager.GetDefaultRegion(),
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

	// Delegate to AWS manager
	instance, err := s.awsManager.LaunchInstance(req)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to launch instance: %v", err))
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
	err := s.awsManager.DeleteInstance(name)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to delete instance: %v", err))
		return
	}

	// Remove from state
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

	err := s.awsManager.StartInstance(name)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to start instance: %v", err))
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleStopInstance(w http.ResponseWriter, r *http.Request, name string) {
	if r.Method != http.MethodPost {
		s.writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	err := s.awsManager.StopInstance(name)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to stop instance: %v", err))
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleConnectInstance(w http.ResponseWriter, r *http.Request, name string) {
	if r.Method != http.MethodGet {
		s.writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	connectionInfo, err := s.awsManager.GetConnectionInfo(name)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to get connection info: %v", err))
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

	templates := s.awsManager.GetTemplates()
	json.NewEncoder(w).Encode(templates)
}

func (s *Server) handleTemplateInfo(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	templateName := r.URL.Path[len("/api/v1/templates/"):]
	template, err := s.awsManager.GetTemplate(templateName)
	if err != nil {
		s.writeError(w, http.StatusNotFound, "Template not found")
		return
	}

	json.NewEncoder(w).Encode(template)
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