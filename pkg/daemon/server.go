package daemon

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/scttfrdmn/cloudworkstation/pkg/state"
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

// Handler functions are now organized in separate files:
// - core_handlers.go: API versioning, ping, status, unknown API
// - instance_handlers.go: Instance CRUD and lifecycle operations
// - template_handlers.go: Template listing and information
// - volume_handlers.go: EFS volume management
// - storage_handlers.go: EBS volume management
// - user_handlers.go: User and group management (already separate)

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