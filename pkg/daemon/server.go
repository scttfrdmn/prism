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
	"github.com/scttfrdmn/cloudworkstation/pkg/idle"
	"github.com/scttfrdmn/cloudworkstation/pkg/profile"
	"github.com/scttfrdmn/cloudworkstation/pkg/project"
	"github.com/scttfrdmn/cloudworkstation/pkg/security"
	"github.com/scttfrdmn/cloudworkstation/pkg/state"
	"github.com/scttfrdmn/cloudworkstation/pkg/types"
)

// Server represents the CloudWorkstation daemon server
type Server struct {
	port            string
	httpServer      *http.Server
	stateManager    *state.Manager
	userManager     *UserManager
	statusTracker   *StatusTracker
	versionManager  *APIVersionManager
	awsManager      *aws.Manager
	idleManager     *idle.Manager
	idleMonitor     *idle.MonitorService
	projectManager  *project.Manager
	securityManager *security.SecurityManager
	
	// Integrated autonomous monitoring
	monitoringCancel context.CancelFunc
	monitoringTicker *time.Ticker
}

// NewServer creates a new daemon server
func NewServer(port string) (*Server, error) {
	if port == "" {
		port = "8947" // CWS on phone keypad - more unique than 8080
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

	// Initialize idle manager
	idleManager, err := idle.NewManager()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize idle manager: %w", err)
	}

	// Skip old monitoring service - using integrated monitoring instead
	var idleMonitor *idle.MonitorService // nil - not used with integrated monitoring

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
		port:            port,
		stateManager:    stateManager,
		userManager:     userManager,
		statusTracker:   statusTracker,
		versionManager:  versionManager,
		awsManager:      awsManager,
		idleManager:     idleManager,
		idleMonitor:     idleMonitor,
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

// GetIdleManager returns the idle manager for autonomous service integration
func (s *Server) GetIdleManager() *idle.Manager {
	return s.idleManager
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

	// Idle detection and hibernation policy operations
	mux.HandleFunc("/api/v1/idle/status", applyMiddleware(s.handleIdleStatus))
	mux.HandleFunc("/api/v1/idle/enable", applyMiddleware(s.handleIdleEnable))
	mux.HandleFunc("/api/v1/idle/disable", applyMiddleware(s.handleIdleDisable))
	mux.HandleFunc("/api/v1/idle/profiles", applyMiddleware(s.handleIdleProfiles))
	mux.HandleFunc("/api/v1/idle/pending-actions", applyMiddleware(s.handleIdlePendingActions))
	mux.HandleFunc("/api/v1/idle/execute-actions", applyMiddleware(s.handleIdleExecuteActions))
	mux.HandleFunc("/api/v1/idle/history", applyMiddleware(s.handleIdleHistory))

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

// startIntegratedMonitoring starts autonomous monitoring when idle detection is enabled
func (s *Server) startIntegratedMonitoring() {
	// Only start monitoring if idle detection is enabled
	if !s.idleManager.IsEnabled() {
		log.Printf("Idle detection disabled - autonomous monitoring not started")
		return
	}
	
	log.Printf("🤖 Starting integrated autonomous monitoring (1 minute intervals)")
	
	ctx, cancel := context.WithCancel(context.Background())
	s.monitoringCancel = cancel
	s.monitoringTicker = time.NewTicker(60 * time.Second) // Every minute
	
	go func() {
		defer s.monitoringTicker.Stop()
		
		// Initial run after 30 seconds to let daemon fully start
		time.Sleep(30 * time.Second)
		
		for {
			select {
			case <-ctx.Done():
				log.Printf("Integrated monitoring stopped")
				return
			case <-s.monitoringTicker.C:
				if err := s.performIntegratedMonitoringCycle(ctx); err != nil {
					log.Printf("Error in integrated monitoring: %v", err)
				}
			}
		}
	}()
}

// stopIntegratedMonitoring stops the integrated autonomous monitoring
func (s *Server) stopIntegratedMonitoring() {
	if s.monitoringCancel != nil {
		log.Printf("Stopping integrated autonomous monitoring...")
		s.monitoringCancel()
		s.monitoringCancel = nil
	}
	if s.monitoringTicker != nil {
		s.monitoringTicker.Stop()
		s.monitoringTicker = nil
	}
}

// performIntegratedMonitoringCycle performs intelligent multi-stage idle detection
func (s *Server) performIntegratedMonitoringCycle(ctx context.Context) error {
	// Smart Multi-Stage Idle Detection for Research Environments:
	// Stage 1: Fast rejection - obvious active usage (< 1 second)
	// Stage 2: Research work detection - background computation/data work  
	// Stage 3: Pattern analysis - scheduled jobs, usage patterns
	// Stage 4: Progressive action - warn before act, hibernation over termination
	
	log.Printf("🔍 Starting intelligent idle detection cycle...")
	
	// Stage 1: Fast Rejection - Immediate NOT IDLE signals
	activeInstances, err := s.detectActiveUsage(ctx)
	if err != nil {
		log.Printf("Error in active usage detection: %v", err)
		return err
	}
	
	if len(activeInstances) > 0 {
		log.Printf("🔍 %d instances have active usage - marked as non-idle", len(activeInstances))
		for _, instanceName := range activeInstances {
			s.markInstanceActive(instanceName, "active usage detected")
		}
	}
	
	// Stage 2: Research Work Detection - Background computation without user interaction
	workingInstances, err := s.detectResearchWork(ctx, activeInstances)
	if err != nil {
		log.Printf("Error in research work detection: %v", err)
		return err
	}
	
	if len(workingInstances) > 0 {
		log.Printf("🔍 %d instances doing background research work - marked as non-idle", len(workingInstances))
		for _, instanceName := range workingInstances {
			s.markInstanceActive(instanceName, "background research work")
		}
	}
	
	// Stage 3: True Idle Detection - No usage + no work + sustained period
	idleInstances, err := s.detectTrueIdleness(ctx, append(activeInstances, workingInstances...))
	if err != nil {
		log.Printf("Error in idle detection: %v", err)
		return err
	}
	
	if len(idleInstances) > 0 {
		log.Printf("🔍 %d instances are truly idle - evaluating for cost-saving actions", len(idleInstances))
		s.evaluateIdleActions(idleInstances)
	}
	
	log.Printf("🔍 Intelligent idle detection complete")
	return nil
}

// detectActiveUsage - Stage 1: Fast rejection - users actively connected
func (s *Server) detectActiveUsage(ctx context.Context) ([]string, error) {
	// Simple check: Are users actively connected?
	log.Printf("  🔍 Stage 1: Checking for active user connections...")
	
	// Get all running instances
	instances, err := s.awsManager.ListInstances()
	if err != nil {
		return nil, fmt.Errorf("failed to list instances: %w", err)
	}
	
	log.Printf("  Found %d total instances", len(instances))
	
	var runningInstancesWithIP []types.Instance
	for _, instance := range instances {
		if instance.State == "running" && instance.PublicIP != "" {
			runningInstancesWithIP = append(runningInstancesWithIP, instance)
		}
	}
	
	log.Printf("  Found %d running instances with public IPs", len(runningInstancesWithIP))
	
	var activeInstances []string
	for _, instance := range runningInstancesWithIP {
		
		// Check for active connections via simple SSH commands
		log.Printf("    → Checking connections for %s (%s)", instance.Name, instance.PublicIP)
		hasConnections, err := s.checkInstanceConnections(instance.PublicIP)
		if err != nil {
			log.Printf("    Warning: Failed to check connections for %s: %v", instance.Name, err)
			continue
		}
		
		if hasConnections {
			log.Printf("    → %s has active user connections", instance.Name)
			activeInstances = append(activeInstances, instance.Name)
		} else {
			log.Printf("    → %s has no active user connections", instance.Name)
		}
	}
	
	return activeInstances, nil
}

// checkInstanceConnections checks if an instance has active user connections
func (s *Server) checkInstanceConnections(publicIP string) (bool, error) {
	// Create SSH-based metrics collector with correct key path
	collector, err := idle.NewMetricsCollector("~/.ssh/cws-my-account-key", "ubuntu", 10*time.Second)
	if err != nil {
		return false, fmt.Errorf("failed to create metrics collector: %w", err)
	}
	
	// Check for active connections using simple commands
	hasConnections, err := collector.CheckActiveConnections(publicIP)
	if err != nil {
		return false, fmt.Errorf("failed to check connections: %w", err)
	}
	
	return hasConnections, nil
}

// detectResearchWork - Stage 2: Simple system activity check - is the system busy?
func (s *Server) detectResearchWork(ctx context.Context, skipInstances []string) ([]string, error) {
	// Simple question: Is the system doing meaningful work?
	log.Printf("  🔍 Stage 2: Checking if system is busy with any work...")
	
	// Get all running instances
	instances, err := s.awsManager.ListInstances()
	if err != nil {
		return nil, fmt.Errorf("failed to list instances: %w", err)
	}
	
	// Convert skipInstances to map for fast lookup
	skipMap := make(map[string]bool)
	for _, name := range skipInstances {
		skipMap[name] = true
	}
	
	var busyInstances []string
	for _, instance := range instances {
		if instance.State != "running" || instance.PublicIP == "" {
			continue
		}
		
		// Skip instances that already have active connections
		if skipMap[instance.Name] {
			continue
		}
		
		// Check if system is busy with any work
		log.Printf("    → Checking system activity for %s (%s)", instance.Name, instance.PublicIP)
		isBusy, err := s.checkSystemActivity(instance.PublicIP)
		if err != nil {
			log.Printf("    Warning: Failed to check system activity for %s: %v", instance.Name, err)
			continue
		}
		
		if isBusy {
			log.Printf("    → %s is busy with background work", instance.Name)
			busyInstances = append(busyInstances, instance.Name)
		} else {
			log.Printf("    → %s has low system activity", instance.Name)
		}
	}
	
	return busyInstances, nil
}

// checkSystemActivity checks if an instance is busy with background work
func (s *Server) checkSystemActivity(publicIP string) (bool, error) {
	// Create SSH-based metrics collector with correct key path
	collector, err := idle.NewMetricsCollector("~/.ssh/cws-my-account-key", "ubuntu", 10*time.Second)
	if err != nil {
		return false, fmt.Errorf("failed to create metrics collector: %w", err)
	}
	
	// Get basic system metrics
	metrics, err := collector.CollectMetrics(publicIP)
	if err != nil {
		return false, fmt.Errorf("failed to collect metrics: %w", err)
	}
	
	// Simple thresholds - if ANY resource is meaningfully active, system is busy
	if metrics.CPU > 15.0 {  // > 15% CPU usage
		return true, nil
	}
	
	if metrics.GPU != nil && *metrics.GPU > 10.0 {  // > 10% GPU usage
		return true, nil
	}
	
	// TODO: Add network and disk I/O thresholds when available in metrics
	
	return false, nil
}

// detectTrueIdleness - Stage 3: Verify sustained quiet period
func (s *Server) detectTrueIdleness(ctx context.Context, skipInstances []string) ([]string, error) {
	// Simple sustained idle check
	log.Printf("  🔍 Stage 3: Verifying sustained quiet period...")
	
	// Get all running instances
	instances, err := s.awsManager.ListInstances()
	if err != nil {
		return nil, fmt.Errorf("failed to list instances: %w", err)
	}
	
	// Convert skipInstances to map for fast lookup
	skipMap := make(map[string]bool)
	for _, name := range skipInstances {
		skipMap[name] = true
	}
	
	var idleInstances []string
	for _, instance := range instances {
		if instance.State != "running" || instance.PublicIP == "" {
			continue
		}
		
		// Skip instances that are already marked as active or busy
		if skipMap[instance.Name] {
			continue
		}
		
		// For now, consider any instance that passed stages 1 & 2 as idle
		// TODO: Add sustained time checking using idle manager history
		
		log.Printf("    → %s appears to be truly idle", instance.Name)
		idleInstances = append(idleInstances, instance.Name)
	}
	
	return idleInstances, nil
}

// markInstanceActive marks an instance as active in the idle detection system
func (s *Server) markInstanceActive(instanceName string, reason string) {
	// TODO: Update idle detection to mark this instance as active
	log.Printf("  → Instance %s marked as ACTIVE (%s)", instanceName, reason)
	
	// Record activity for future smart decisions
	s.statusTracker.RecordInstanceActivity(instanceName)
}

// evaluateIdleActions - Stage 4: Progressive cost-saving actions for truly idle instances
func (s *Server) evaluateIdleActions(idleInstances []string) {
	// Progressive action strategy for research environments:
	// 1. First time idle: Send notification/warning (don't act yet)
	// 2. Sustained idle: Hibernation (preserves state, fast resume)
	// 3. Long-term idle: Stop (more cost savings, slower resume)
	// 4. Abandoned: Terminate (only with explicit user consent)
	
	for _, instanceName := range idleInstances {
		// TODO: Implement progressive idle actions based on:
		// - How long instance has been idle
		// - Instance type and cost (GPU = act faster)  
		// - Historical usage patterns (don't act during expected work hours)
		// - User preferences (hibernation vs stop vs notification only)
		
		log.Printf("  → Evaluating cost-saving actions for idle instance: %s", instanceName)
		
		// TODO: Check idle history and determine appropriate action:
		// - First 30 minutes idle: Notification only
		// - 30-60 minutes idle: Hibernation (preserve state)
		// - 2+ hours idle: Stop (deeper cost savings)
		// - GPU instances: Faster action (higher cost)
		// - Spot instances: More conservative (termination risk)
	}
}

// Auth handlers are implemented in auth.go