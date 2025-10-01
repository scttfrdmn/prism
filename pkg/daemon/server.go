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
	"github.com/scttfrdmn/cloudworkstation/pkg/connection"
	"github.com/scttfrdmn/cloudworkstation/pkg/cost"
	"github.com/scttfrdmn/cloudworkstation/pkg/marketplace"
	"github.com/scttfrdmn/cloudworkstation/pkg/monitoring"
	"github.com/scttfrdmn/cloudworkstation/pkg/policy"
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
	policyService   *policy.Service
	processManager  ProcessManager

	// Connection reliability components
	performanceMonitor *monitoring.PerformanceMonitor
	connManager        *connection.ConnectionManager
	reliabilityManager *connection.ReliabilityManager

	// Daemon stability components
	stabilityManager *StabilityManager
	recoveryManager  *RecoveryManager
	healthMonitor    *HealthMonitor

	// Cost optimization components
	budgetTracker *project.BudgetTracker
	alertManager  *cost.AlertManager

	// Template marketplace components
	marketplaceRegistry *marketplace.Registry
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
	profileManager, profileErr := profile.NewManagerEnhanced()
	if profileErr != nil {
		log.Printf("Failed to initialize profile manager: %v", profileErr)
		// Initialize AWS manager with AWS SDK defaults (no hardcoded values)
		awsManager, err = aws.NewManager(aws.ManagerOptions{
			Profile: "", // Use AWS SDK default profile resolution
			Region:  "", // Use AWS SDK default region resolution
		})
	} else {
		currentProfile, err := profileManager.GetCurrentProfile()
		if err != nil {
			log.Printf("Failed to get current profile, using AWS defaults: %v", err)
			// Initialize AWS manager with AWS SDK defaults (no hardcoded values)
			awsManager, _ = aws.NewManager(aws.ManagerOptions{
				Profile: "", // Use AWS SDK default profile resolution
				Region:  "", // Use AWS SDK default region resolution
			})
		} else {
			// Use profile values from current CloudWorkstation profile
			awsManager, _ = aws.NewManager(aws.ManagerOptions{
				Profile: currentProfile.AWSProfile,
				Region:  currentProfile.Region,
			})
		}
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

	// Initialize cost optimization components
	budgetTracker, err := project.NewBudgetTracker()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize budget tracker: %w", err)
	}
	alertManager := cost.NewAlertManager()
	alertManager.CreateDefaultRules()

	// Initialize template marketplace registry
	marketplaceConfig := &marketplace.MarketplaceConfig{
		RegistryEndpoint:      "https://marketplace.cloudworkstation.org",
		S3Bucket:              "cloudworkstation-marketplace",
		DynamoDBTable:         "marketplace-templates",
		CDNEndpoint:           "https://cdn.cloudworkstation.org",
		AutoAMIGeneration:     true,
		DefaultRegions:        []string{"us-east-1", "us-west-2", "eu-west-1"},
		RequireModeration:     false,
		MinRatingForFeatured:  4.0,
		MinReviewsForFeatured: 5,
		PublishRateLimit:      10,  // 10 publications per day
		ReviewRateLimit:       20,  // 20 reviews per day
		SearchRateLimit:       100, // 100 searches per minute
	}
	marketplaceRegistry := marketplace.NewRegistry(marketplaceConfig)
	marketplaceRegistry.LoadSampleData() // Load sample data for development
	alertManager.Start()

	// Initialize security manager
	securityConfig := security.GetDefaultSecurityConfig()
	securityManager, err := security.NewSecurityManager(securityConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize security manager: %w", err)
	}

	// Initialize policy service
	policyService := policy.NewService()
	log.Printf("Policy service initialized with framework foundation")

	// Initialize process manager
	processManager := NewProcessManager()

	// Initialize performance monitoring
	performanceMonitor := monitoring.NewPerformanceMonitor()

	// Initialize connection reliability
	connManager := connection.NewConnectionManager(performanceMonitor)
	reliabilityManager := connection.NewReliabilityManager(connManager, performanceMonitor)

	// Initialize daemon stability
	stabilityManager := NewStabilityManager(performanceMonitor)

	server := &Server{
		config:              config,
		port:                port,
		stateManager:        stateManager,
		userManager:         userManager,
		statusTracker:       statusTracker,
		versionManager:      versionManager,
		awsManager:          awsManager,
		projectManager:      projectManager,
		securityManager:     securityManager,
		policyService:       policyService,
		processManager:      processManager,
		performanceMonitor:  performanceMonitor,
		connManager:         connManager,
		reliabilityManager:  reliabilityManager,
		stabilityManager:    stabilityManager,
		budgetTracker:       budgetTracker,
		alertManager:        alertManager,
		marketplaceRegistry: marketplaceRegistry,
	}

	// Configure budget tracker with action executor
	budgetTracker.SetActionExecutor(server)

	// Initialize recovery and health monitoring (need server reference)
	server.recoveryManager = NewRecoveryManager(stabilityManager, nil) // Will be set after server creation
	server.healthMonitor = NewHealthMonitor(stateManager, stabilityManager, server.recoveryManager, performanceMonitor)

	// Initialize launch manager (if needed)
	// server.launchManager = NewLaunchManager(server)

	// Setup HTTP routes
	mux := http.NewServeMux()
	server.setupRoutes(mux)

	server.httpServer = &http.Server{
		Addr:         ":" + port,
		Handler:      mux,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	// Set HTTP server reference in recovery manager
	server.recoveryManager.HTTPServer = server.httpServer

	return server, nil
}

// Start starts the daemon server
func (s *Server) Start() error {
	log.Printf("Starting CloudWorkstation daemon on port %s", s.port)

	// Register this daemon instance
	pid := os.Getpid()
	configPath := fmt.Sprintf("%s/.cloudworkstation", os.Getenv("HOME"))
	if err := s.processManager.RegisterDaemon(pid, configPath, ""); err != nil {
		log.Printf("Warning: Failed to register daemon: %v", err)
	}

	// Start daemon stability and monitoring systems
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	log.Printf("Starting stability management systems...")
	go s.performanceMonitor.Start(ctx)
	go s.reliabilityManager.Start(ctx)
	go s.stabilityManager.Start(ctx)
	go s.healthMonitor.Start(ctx)

	// Enable memory management
	s.stabilityManager.EnableForceGC(true)
	log.Printf("Daemon stability systems started")

	// Start security manager
	if err := s.securityManager.Start(); err != nil {
		log.Printf("Warning: Failed to start security manager: %v", err)
		s.stabilityManager.RecordError("security", "startup_failed", err.Error(), ErrorSeverityHigh)
	} else {
		log.Printf("Security manager started successfully")
		s.stabilityManager.RecordRecovery("security", "startup_failed")
	}

	// Start integrated autonomous monitoring if idle detection is enabled
	s.startIntegratedMonitoring()

	// Handle graceful shutdown with recovery manager
	go func() {
		c := make(chan os.Signal, 1)
		signal.Notify(c, os.Interrupt, syscall.SIGTERM)
		<-c
		log.Println("Shutting down daemon with stability management...")

		// Use recovery manager for graceful shutdown
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		if err := s.recoveryManager.GracefulShutdown(ctx); err != nil {
			log.Printf("Warning: Graceful shutdown had issues: %v", err)
		}

		// Unregister this daemon instance
		pid := os.Getpid()
		if err := s.processManager.UnregisterDaemon(pid); err != nil {
			log.Printf("Warning: Failed to unregister daemon: %v", err)
		}

		// Stop integrated monitoring
		s.stopIntegratedMonitoring()

		// Stop security manager
		if err := s.securityManager.Stop(); err != nil {
			log.Printf("Warning: Failed to stop security manager: %v", err)
		}
	}()

	return s.httpServer.ListenAndServe()
}

// Stop stops the daemon server gracefully
func (s *Server) Stop() error {
	log.Printf("Gracefully stopping daemon server...")

	// Unregister this daemon instance
	pid := os.Getpid()
	if err := s.processManager.UnregisterDaemon(pid); err != nil {
		log.Printf("Warning: Failed to unregister daemon: %v", err)
	}

	// Stop security manager
	if err := s.securityManager.Stop(); err != nil {
		log.Printf("Warning: Failed to stop security manager: %v", err)
	}

	// Stop integrated monitoring
	s.stopIntegratedMonitoring()

	// Shutdown HTTP server with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	if err := s.httpServer.Shutdown(ctx); err != nil {
		return fmt.Errorf("daemon server shutdown failed: %w", err)
	}

	log.Printf("Daemon server stopped successfully")
	return nil
}

// Cleanup performs comprehensive cleanup for uninstallation
func (s *Server) Cleanup() error {
	log.Printf("Performing comprehensive daemon cleanup...")

	// First stop the server if running
	if err := s.Stop(); err != nil {
		log.Printf("Warning: Server stop failed during cleanup: %v", err)
	}

	// Clean up all daemon processes
	if err := s.processManager.CleanupProcesses(); err != nil {
		return fmt.Errorf("failed to cleanup daemon processes: %w", err)
	}

	log.Printf("Daemon cleanup completed successfully")
	return nil
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

	// Research user operations (Phase 5A.3: REST API Integration)
	mux.HandleFunc("/api/v1/research-users", applyMiddleware(s.handleResearchUsers))
	mux.HandleFunc("/api/v1/research-users/", applyMiddleware(s.handleResearchUserOperations))

	// Idle policy operations
	s.RegisterIdleRoutes(mux, applyMiddleware)

	// Process management operations
	mux.HandleFunc("/api/v1/daemon/processes", applyMiddleware(s.handleDaemonProcesses))
	mux.HandleFunc("/api/v1/daemon/cleanup", applyMiddleware(s.handleDaemonCleanup))

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

	// Daemon stability and health endpoints (Phase 1.3: Daemon Stability)
	if s.healthMonitor != nil {
		mux.HandleFunc("/api/v1/health", s.recoveryManager.RecoverHTTPHandler("health", s.healthMonitor.HandleHealthEndpoint))
		mux.HandleFunc("/api/v1/health/detailed", s.recoveryManager.RecoverHTTPHandler("health_detailed", s.healthMonitor.HandleDetailedHealthEndpoint))
	}
	mux.HandleFunc("/api/v1/stability/metrics", applyMiddleware(s.handleStabilityMetrics))
	mux.HandleFunc("/api/v1/stability/errors", applyMiddleware(s.handleStabilityErrors))
	mux.HandleFunc("/api/v1/stability/circuit-breakers", applyMiddleware(s.handleCircuitBreakers))
	mux.HandleFunc("/api/v1/stability/recovery", applyMiddleware(s.handleRecoveryTrigger))

	// Policy management endpoints (Phase 5A.5)
	s.RegisterPolicyRoutes(mux, applyMiddleware)

	// Enhanced connection proxy endpoints (Phase 5A.5+)
	s.RegisterConnectionProxyRoutes(mux, applyMiddleware)

	// Cost optimization and budget alert endpoints
	s.RegisterCostHandlers(mux, applyMiddleware)

	// AMI management endpoints (Phase 5.1 Week 2: REST API Integration)
	s.RegisterAMIRoutes(mux, applyMiddleware)

	// Template marketplace endpoints (Phase 5.1 Week 3: Marketplace Integration)
	s.RegisterMarketplaceRoutes(mux, applyMiddleware)
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
	if len(parts[0]) == 0 {
		return "Root"
	}
	resourceType := strings.ToUpper(parts[0][:1]) + parts[0][1:]
	if len(resourceType) > 0 && resourceType[len(resourceType)-1] == 's' {
		// Convert plural to singular (instances -> instance)
		resourceType = resourceType[:len(resourceType)-1]
	}

	// If there's an ID and operation, use those
	if len(parts) >= 3 && len(parts[2]) > 0 {
		operation := strings.ToUpper(parts[2][:1]) + parts[2][1:]
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

// createHTTPHandler creates and configures the HTTP handler for testing
func (s *Server) createHTTPHandler() http.Handler {
	mux := http.NewServeMux()
	s.setupRoutes(mux)
	return mux
}

// Auth handlers are implemented in auth.go

// Budget Action Executor implementation
// The Server implements the project.ActionExecutor interface

// ExecuteHibernateAll hibernates all instances for a project
func (s *Server) ExecuteHibernateAll(projectID string) error {
	// Get all instances
	instances, err := s.awsManager.ListInstances()
	if err != nil {
		return fmt.Errorf("failed to list instances for hibernation: %w", err)
	}

	// Find instances belonging to this project
	// TODO: Proper project-instance association needs to be implemented with tags
	// For now, hibernate all running instances as a safety measure
	var hibernatedCount int
	var errors []string

	for _, instance := range instances {
		if instance.State == "running" {
			if err := s.awsManager.HibernateInstance(instance.Name); err != nil {
				errors = append(errors, fmt.Sprintf("failed to hibernate %s: %v", instance.Name, err))
			} else {
				hibernatedCount++
			}
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("hibernated %d instances but encountered errors: %s", hibernatedCount, strings.Join(errors, ", "))
	}

	log.Printf("Budget auto action: hibernated %d instances for project %s", hibernatedCount, projectID)
	return nil
}

// ExecuteStopAll stops all instances for a project
func (s *Server) ExecuteStopAll(projectID string) error {
	// Get all instances
	instances, err := s.awsManager.ListInstances()
	if err != nil {
		return fmt.Errorf("failed to list instances for stopping: %w", err)
	}

	// Find instances belonging to this project
	// TODO: Proper project-instance association needs to be implemented with tags
	// For now, stop all running instances as a safety measure
	var stoppedCount int
	var errors []string

	for _, instance := range instances {
		if instance.State == "running" {
			if err := s.awsManager.StopInstance(instance.Name); err != nil {
				errors = append(errors, fmt.Sprintf("failed to stop %s: %v", instance.Name, err))
			} else {
				stoppedCount++
			}
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("stopped %d instances but encountered errors: %s", stoppedCount, strings.Join(errors, ", "))
	}

	log.Printf("Budget auto action: stopped %d instances for project %s", stoppedCount, projectID)
	return nil
}

// ExecutePreventLaunch sets a flag to prevent new launches for a project
func (s *Server) ExecutePreventLaunch(projectID string) error {
	// TODO: Implement launch prevention mechanism
	// This would require adding a flag to the project or state manager
	// that prevents new instance launches for this project
	log.Printf("Budget auto action: prevent launch triggered for project %s (not yet implemented)", projectID)
	return fmt.Errorf("prevent launch action not yet implemented")
}
