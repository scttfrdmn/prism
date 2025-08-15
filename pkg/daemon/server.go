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

	"github.com/scttfrdmn/cloudworkstation/pkg/aws"
	"github.com/scttfrdmn/cloudworkstation/pkg/profile"
	"github.com/scttfrdmn/cloudworkstation/pkg/project"
	"github.com/scttfrdmn/cloudworkstation/pkg/security"
	"github.com/scttfrdmn/cloudworkstation/pkg/state"
)

// Server represents the CloudWorkstation daemon server
type Server struct {
	config          *Config
	port            string
	httpServer      *http.Server
	stateManager    *state.Manager
	userManager     *UserManager
	statusTracker   *StatusTracker
	versionManager  *APIVersionManager
	awsManager      *aws.Manager
	projectManager  *project.Manager
	securityManager *security.SecurityManager

	// Integrated autonomous monitoring
	monitoringCancel context.CancelFunc
	monitoringTicker *time.Ticker
}

// NewServer creates a new daemon server
func NewServer(port string) (*Server, error) {
	// Load daemon configuration
	config, err := LoadConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to load daemon configuration: %w", err)
	}

	// Use port from parameter or config, with fallback to default
	if port == "" {
		if config.Port != "" {
			port = config.Port
		} else {
			port = "8947" // CWS on phone keypad - more unique than 8080
		}
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

	// Get current profile configuration and initialize AWS manager
	var awsManager *aws.Manager
	currentProfile, err := profile.GetCurrentProfile()
	if err != nil {
		log.Printf("Failed to get current profile, using defaults: %v", err)
		// Initialize AWS manager with default profile 'aws' as requested
		awsManager, err = aws.NewManager(aws.ManagerOptions{
			Profile: "aws",
			Region:  "us-west-2",
		})
	} else {
		// Use profile values but force 'aws' profile as requested
		awsManager, err = aws.NewManager(aws.ManagerOptions{
			Profile: "aws", // Always use 'aws' profile as requested
			Region:  currentProfile.Region,
		})
	}
	if err != nil {
		return nil, fmt.Errorf("failed to initialize AWS manager: %w", err)
	}

	// Legacy idle management removed - using universal idle detection via template resolver

	// Initialize project manager
	projectManager, err := project.NewManager()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize project manager: %w", err)
	}

	// Initialize security manager
	securityConfig := security.GetDefaultSecurityConfig()
	securityManager, err := security.NewSecurityManager(securityConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize security manager: %w", err)
	}

	server := &Server{
		config:          config,
		port:            port,
		stateManager:    stateManager,
		userManager:     userManager,
		statusTracker:   statusTracker,
		versionManager:  versionManager,
		awsManager:      awsManager,
		projectManager:  projectManager,
		securityManager: securityManager,
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

	// Start security manager
	if err := s.securityManager.Start(); err != nil {
		log.Printf("Warning: Failed to start security manager: %v", err)
	} else {
		log.Printf("Security manager started successfully")
	}

	// Start integrated autonomous monitoring if idle detection is enabled
	s.startIntegratedMonitoring()

	// Handle graceful shutdown
	go func() {
		c := make(chan os.Signal, 1)
		signal.Notify(c, os.Interrupt, syscall.SIGTERM)
		<-c
		log.Println("Shutting down daemon...")

		// Stop integrated monitoring
		s.stopIntegratedMonitoring()

		// Stop security manager
		if err := s.securityManager.Stop(); err != nil {
			log.Printf("Warning: Failed to stop security manager: %v", err)
		}

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

// Legacy idle management removed

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
	mux.HandleFunc("/api/v1/shutdown", applyMiddleware(s.handleShutdown))

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

	// Template application operations
	mux.HandleFunc("/api/v1/templates/apply", applyMiddleware(s.handleTemplateApply))
	mux.HandleFunc("/api/v1/templates/diff", applyMiddleware(s.handleTemplateDiff))

	// Volume operations
	mux.HandleFunc("/api/v1/volumes", applyMiddleware(s.handleVolumes))
	mux.HandleFunc("/api/v1/volumes/", applyMiddleware(s.handleVolumeOperations))

	// Storage operations
	mux.HandleFunc("/api/v1/storage", applyMiddleware(s.handleStorage))
	mux.HandleFunc("/api/v1/storage/", applyMiddleware(s.handleStorageOperations))

	// Legacy idle detection removed - using universal idle detection via template resolver

	// Project management operations
	mux.HandleFunc("/api/v1/projects", applyMiddleware(s.handleProjectOperations))
	mux.HandleFunc("/api/v1/projects/", applyMiddleware(s.handleProjectByID))

	// Security management endpoints (Phase 4: Security integration)
	mux.HandleFunc("/api/v1/security/status", applyMiddleware(s.handleSecurityStatus))
	mux.HandleFunc("/api/v1/security/health", applyMiddleware(s.handleSecurityHealth))
	mux.HandleFunc("/api/v1/security/dashboard", applyMiddleware(s.handleSecurityDashboard))
	mux.HandleFunc("/api/v1/security/correlations", applyMiddleware(s.handleSecurityCorrelations))
	mux.HandleFunc("/api/v1/security/keychain", applyMiddleware(s.handleSecurityKeychain))
	mux.HandleFunc("/api/v1/security/config", applyMiddleware(s.handleSecurityConfig))
	// AWS Compliance validation endpoints
	mux.HandleFunc("/api/v1/security/compliance/validate/{framework}", applyMiddleware(s.handleAWSComplianceValidate))
	mux.HandleFunc("/api/v1/security/compliance/report/{framework}", applyMiddleware(s.handleAWSComplianceReport))
	mux.HandleFunc("/api/v1/security/compliance/scp/{framework}", applyMiddleware(s.handleAWSComplianceSCP))
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

// startIntegratedMonitoring removed - using universal idle detection via template resolver
func (s *Server) startIntegratedMonitoring() {
	log.Printf("Legacy monitoring removed - using universal idle detection")
}

// stopIntegratedMonitoring removed - using universal idle detection
func (s *Server) stopIntegratedMonitoring() {
	log.Printf("Legacy monitoring removed - using universal idle detection")
}

// performIntegratedMonitoringCycle removed - using universal idle detection
func (s *Server) performIntegratedMonitoringCycle(ctx context.Context) error {
	return nil
}

// Auth handlers are implemented in auth.go
