package daemon

import (
	"context"
	cryptorand "crypto/rand"
	"encoding/hex"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	awssdk "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch"
	"github.com/scttfrdmn/prism/pkg/alerting"
	"github.com/scttfrdmn/prism/pkg/aws"
	"github.com/scttfrdmn/prism/pkg/connection"
	"github.com/scttfrdmn/prism/pkg/cost"
	"github.com/scttfrdmn/prism/pkg/course"
	"github.com/scttfrdmn/prism/pkg/daemon/logger"
	"github.com/scttfrdmn/prism/pkg/idle"
	"github.com/scttfrdmn/prism/pkg/invitation"
	"github.com/scttfrdmn/prism/pkg/marketplace"
	"github.com/scttfrdmn/prism/pkg/monitoring"
	"github.com/scttfrdmn/prism/pkg/policy"
	"github.com/scttfrdmn/prism/pkg/profile"
	"github.com/scttfrdmn/prism/pkg/project"
	"github.com/scttfrdmn/prism/pkg/rbac"
	"github.com/scttfrdmn/prism/pkg/security"
	"github.com/scttfrdmn/prism/pkg/sleepwake"
	"github.com/scttfrdmn/prism/pkg/state"
	"github.com/scttfrdmn/prism/pkg/throttle"
	"github.com/scttfrdmn/prism/pkg/types"
	"github.com/scttfrdmn/prism/pkg/workshop"
)

// Server represents the Prism daemon server
type Server struct {
	config             *Config
	port               string
	httpServer         *http.Server
	stateManager       *state.Manager
	userManager        *UserManager
	statusTracker      *StatusTracker
	versionManager     *APIVersionManager
	awsManager         *aws.Manager
	projectManager     *project.Manager
	budgetManager      *project.BudgetManager
	invitationManager  *invitation.Manager
	sharedTokenManager *invitation.SharedTokenManager
	securityManager    *security.SecurityManager
	policyService      *policy.Service
	rbacManager        *rbac.Manager
	spendLedger        *spendStore // Phase 2 (#644): budgetengine spend ledger (SpendSource/Writer)
	processManager     ProcessManager

	// Connection reliability components
	performanceMonitor *monitoring.PerformanceMonitor
	connManager        *connection.ConnectionManager
	reliabilityManager *connection.ReliabilityManager

	// Daemon stability components
	stabilityManager *StabilityManager
	recoveryManager  *RecoveryManager
	healthMonitor    *HealthMonitor

	// Background state monitoring (v0.5.8)
	stateMonitor        *StateMonitor
	storageStateMonitor *StorageStateMonitor

	// Cost optimization components
	budgetTracker   *project.BudgetTracker
	alertManager    *cost.AlertManager
	rateLimiter     *RateLimiter              // Launch rate limiting (v0.5.12)
	launchThrottler *throttle.LaunchThrottler // Advanced launch throttling (v0.6.0)

	// Sleep/wake monitoring (v0.5.7 - Issue #91)
	sleepWakeMonitor *sleepwake.Monitor // Automatic hibernation on system sleep

	// Template marketplace components
	marketplaceRegistry *marketplace.Registry

	// Web service tunneling
	tunnelManager *TunnelManager

	// CloudWatch client for rightsizing metrics
	cloudwatchClient *cloudwatch.Client

	// Test mode flag (skips AWS operations for unit testing)
	testMode bool

	// Reduced functionality mode (AWS credentials unavailable - Issue #356)
	reducedMode bool

	// Idle detection components (daemon singletons - Issue #289)
	idleScheduler *idle.Scheduler
	policyManager *idle.PolicyManager

	// Profile management singleton (prevents filesystem race conditions)
	profileManager *profile.ManagerEnhanced

	// Progress tracking for instance launches (v0.7.2 - Issue #453)
	progressTracker *ProgressTracker

	// GDEW tracker for egress waiver credit calculations (v0.11.0 - Issue #206)
	gdewTracker *project.GDEWTracker

	// Approval workflow manager (v0.12.0 - Issue #148/#149/#153/#157)
	approvalManager *project.ApprovalManager

	// Course / Education manager (v0.14.0 - Issue #45)
	courseManager *course.Manager

	// Workshop / Event manager (v0.18.0 - Issue #135/#178-#184)
	workshopManager *workshop.Manager
}

// awsInitResult bundles the outputs of initAWSManager.
type awsInitResult struct {
	manager        *aws.Manager
	profileManager *profile.ManagerEnhanced
	reducedMode    bool
}

// envAWSOptions reads AWS connection options from environment variables.
func envAWSOptions() aws.ManagerOptions {
	opts := aws.ManagerOptions{
		Profile: os.Getenv("AWS_PROFILE"),
		Region:  os.Getenv("AWS_REGION"),
	}
	if opts.Profile != "" {
		logger.Info("Using AWS_PROFILE from environment", "profile", opts.Profile)
	}
	if opts.Region != "" {
		logger.Info("Using AWS_REGION from environment", "region", opts.Region)
	}
	return opts
}

// resolveAWSOptions returns AWS options from the active Prism profile, falling back to env vars.
func resolveAWSOptions() (aws.ManagerOptions, *profile.ManagerEnhanced) {
	profileMgr, err := profile.NewManagerEnhanced()
	if err != nil {
		logger.Warn("Failed to initialize profile manager", "error", err)
		return envAWSOptions(), nil
	}
	currentProfile, err := profileMgr.GetCurrentProfile()
	if err != nil {
		logger.Warn("Failed to get current profile, using AWS defaults", "error", err)
		return envAWSOptions(), profileMgr
	}
	return aws.ManagerOptions{
		Profile: currentProfile.AWSProfile,
		Region:  currentProfile.Region,
	}, profileMgr
}

// initAWSManager initializes the AWS manager from profile or environment configuration.
// It returns a reduced-mode result (manager=nil, reducedMode=true) when credentials are
// unavailable rather than failing, allowing the daemon to start in a degraded state.
func initAWSManager() (awsInitResult, error) {
	opts, profileMgr := resolveAWSOptions()
	mgr, err := aws.NewManager(opts)
	if err != nil {
		if isCredentialError(err) {
			logger.Warn("AWS credentials unavailable, starting in reduced functionality mode")
			return awsInitResult{manager: nil, profileManager: profileMgr, reducedMode: true}, nil
		}
		return awsInitResult{}, fmt.Errorf("failed to initialize AWS manager: %w", err)
	}
	return awsInitResult{manager: mgr, profileManager: profileMgr}, nil
}

// resolvePort returns the effective port, preferring the parameter, then config, then default.
func resolvePort(param, configPort string) string {
	if param != "" {
		return param
	}
	if configPort != "" {
		return configPort
	}
	return "8947" // CWS on phone keypad
}

// initIdleComponents creates the idle scheduler and policy manager when awsMgr is available.
// NOTE: The scheduler monitoring loop is no longer started — spored handles idle detection
// on each instance directly (7-signal detection: CPU, network, disk I/O, GPU, terminals,
// users, recent activity). The scheduler remains as a data store for fleet-level policy
// management and schedule CRUD operations.
func initIdleComponents(awsMgr *aws.Manager) (*idle.Scheduler, *idle.PolicyManager) {
	if awsMgr == nil {
		return nil, nil
	}
	metricsCollector := idle.NewMetricsCollector(awsMgr.GetAWSConfig())
	scheduler := idle.NewScheduler(awsMgr, metricsCollector)
	policyMgr := idle.NewPolicyManager()
	policyMgr.SetScheduler(scheduler)
	// scheduler.Start() removed — spored handles idle detection on-instance (#588)
	logger.Info("Idle policy manager initialized (spored handles on-instance detection)")
	return scheduler, policyMgr
}

// initCloudWatchClient creates a CloudWatch client from the AWS manager, or nil if unavailable.
func initCloudWatchClient(awsMgr *aws.Manager) *cloudwatch.Client {
	if awsMgr == nil {
		return nil
	}
	return cloudwatch.NewFromConfig(awsMgr.GetAWSConfig())
}

// makeCostDataFn returns a cost data provider function backed by the given BudgetTracker.
func makeCostDataFn(bt *project.BudgetTracker) func(string) (float64, float64, float64, []float64, error) {
	return func(projectID string) (float64, float64, float64, []float64, error) {
		budgetStatus, err := bt.CheckBudgetStatus(projectID)
		if err != nil {
			return 0, 0, 0, []float64{}, nil
		}
		costHistory, err := bt.GetProjectCostHistory(projectID, 90)
		if err != nil {
			costHistory = []float64{}
		}
		dailyCost := budgetStatus.SpentAmount
		if len(costHistory) >= 2 {
			recentCost := costHistory[len(costHistory)-1]
			previousCost := costHistory[len(costHistory)-2]
			dailyCost = recentCost - previousCost
			if dailyCost < 0 {
				dailyCost = 0
			}
		}
		return budgetStatus.SpentAmount, budgetStatus.TotalBudget, dailyCost, costHistory, nil
	}
}

// NewServer creates a new daemon server
func NewServer(port string) (*Server, error) {
	// Load daemon configuration
	config, err := LoadConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to load daemon configuration: %w", err)
	}

	port = resolvePort(port, config.Port)

	// Initialize state manager
	stateManager, err := state.NewManager()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize state manager: %w", err)
	}

	// Auto-generate API key if none exists (#597) — skip in test mode
	if os.Getenv("PRISM_TEST_MODE") != "true" {
		if initState, loadErr := stateManager.LoadState(); loadErr == nil && initState.Config.APIKey == "" {
			logger.Warn("No API key configured — generating one automatically")
			keyBytes := make([]byte, 32)
			if _, randErr := cryptorand.Read(keyBytes); randErr == nil {
				initState.Config.APIKey = hex.EncodeToString(keyBytes)
				initState.Config.APIKeyCreated = time.Now()
				_ = stateManager.SaveState(initState)
				logger.Info("API key generated successfully")
			}
		}
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

	// Get current profile configuration and initialize AWS manager.
	// NOTE: initAWSManager returns reduced-mode instead of failing when credentials
	// are unavailable, preventing daemon startup delays (Issue #356).
	awsInit, err := initAWSManager()
	if err != nil {
		return nil, err
	}
	awsManager := awsInit.manager
	profileManager := awsInit.profileManager
	reducedMode := awsInit.reducedMode

	// Initialize idle detection components as daemon singletons (Issue #289 fix)
	idleScheduler, policyManager := initIdleComponents(awsManager)

	// Legacy idle management removed - using universal idle detection via template resolver

	// Initialize the four seam-backed governance managers (project, budget, rbac, approval)
	// together — they share one backend decision (file-backed ~/.prism by default, or the shared
	// DynamoDB table when PRISM_SEAM_BACKEND=dynamodb).
	var seamAWSCfg *awssdk.Config
	if awsManager != nil {
		cfg := awsManager.GetAWSConfig()
		seamAWSCfg = &cfg
	}
	seamMgrs, err := initSeamManagers(seamAWSCfg)
	if err != nil {
		return nil, err
	}
	projectManager := seamMgrs.project
	budgetManager := seamMgrs.budget
	rbacManager := seamMgrs.rbac
	spendLedger := seamMgrs.spend // Phase 2 (#644): budgetengine spend ledger

	// Wire active-instance check so DeleteProject blocks while instances are running (#539)
	projectManager.SetActiveInstancesFunc(func(projectID string) ([]string, error) {
		st, err := stateManager.LoadState()
		if err != nil {
			return nil, err
		}
		var active []string
		for name, inst := range st.Instances {
			if inst.ProjectID == projectID && inst.State == "running" {
				active = append(active, name)
			}
		}
		return active, nil
	})

	budgetManager.SetProjectManager(projectManager)

	// Initialize invitation manager (v0.5.11 user invitation system)
	invitationManager, err := invitation.NewManager(invitation.NewEmailSender())
	if err != nil {
		return nil, fmt.Errorf("failed to initialize invitation manager: %w", err)
	}

	// Initialize shared token manager (v0.5.13 shared token system)
	sharedTokenManager := invitation.NewSharedTokenManager()

	// Initialize cost optimization components
	budgetTracker, err := project.NewBudgetTracker()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize budget tracker: %w", err)
	}

	// Create cost data provider adapter for alert manager
	costDataProvider := cost.NewBudgetTrackerAdapter(
		makeCostDataFn(budgetTracker),
		func() ([]string, error) {
			return budgetTracker.GetAllProjectIDs(), nil
		},
	)

	alertManager := cost.NewAlertManager(costDataProvider)
	alertManager.CreateDefaultRules()

	// Initialize launch rate limiter (v0.5.12)
	// Default: 2 launches per minute to prevent accidental cost overruns
	rateLimiter := NewRateLimiter(2, time.Minute)
	logger.Info("Launch rate limiter initialized", "max_launches", 2, "window", "1 minute")

	// Initialize advanced launch throttling (v0.6.0)
	// Default: Disabled (opt-in), can be configured via API
	throttleConfig := throttle.DefaultConfig()
	launchThrottler := throttle.NewLaunchThrottler(throttleConfig)
	if throttleConfig.Enabled {
		logger.Info("Launch throttling enabled", "max_launches", throttleConfig.MaxLaunches, "window", throttleConfig.TimeWindow)
	} else {
		logger.Info("Launch throttling initialized", "enabled", false)
	}

	// Initialize template marketplace registry
	marketplaceConfig := &marketplace.MarketplaceConfig{
		RegistryEndpoint:      "https://marketplace.prism.org",
		S3Bucket:              "prism-marketplace",
		DynamoDBTable:         "marketplace-templates",
		CDNEndpoint:           "https://cdn.prism.org",
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

	// Initialize tunnel manager for web services
	tunnelManager := NewTunnelManager(stateManager)
	logger.Info("Tunnel manager initialized")

	// Initialize security manager
	securityConfig := security.GetDefaultSecurityConfig()
	securityManager, err := security.NewSecurityManager(securityConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize security manager: %w", err)
	}

	// Initialize policy service
	policyService := policy.NewService()
	logger.Info("Policy service initialized")

	// RBAC manager comes from the seam-backed managers initialized above.
	logger.Info("RBAC manager initialized")

	// Initialize process manager
	processManager := NewProcessManager()

	// Initialize performance monitoring
	performanceMonitor := monitoring.NewPerformanceMonitor()

	// Initialize connection reliability
	connManager := connection.NewConnectionManager(performanceMonitor)
	reliabilityManager := connection.NewReliabilityManager(connManager, performanceMonitor)

	// Initialize daemon stability
	stabilityManager := NewStabilityManager(performanceMonitor)

	// Initialize state monitor for background instance monitoring (v0.5.8)
	stateMonitor := NewStateMonitor(awsManager, stateManager)

	// Initialize storage state monitor for background volume monitoring
	storageStateMonitor := NewStorageStateMonitor(awsManager, stateManager)

	// Initialize CloudWatch client for rightsizing metrics
	cloudwatchClient := initCloudWatchClient(awsManager)

	server := &Server{
		config:              config,
		port:                port,
		stateManager:        stateManager,
		userManager:         userManager,
		statusTracker:       statusTracker,
		versionManager:      versionManager,
		awsManager:          awsManager,
		projectManager:      projectManager,
		budgetManager:       budgetManager,
		invitationManager:   invitationManager,
		sharedTokenManager:  sharedTokenManager,
		securityManager:     securityManager,
		policyService:       policyService,
		rbacManager:         rbacManager,
		spendLedger:         spendLedger,
		processManager:      processManager,
		performanceMonitor:  performanceMonitor,
		connManager:         connManager,
		reliabilityManager:  reliabilityManager,
		stabilityManager:    stabilityManager,
		stateMonitor:        stateMonitor,
		storageStateMonitor: storageStateMonitor,
		budgetTracker:       budgetTracker,
		alertManager:        alertManager,
		rateLimiter:         rateLimiter,
		launchThrottler:     launchThrottler,
		marketplaceRegistry: marketplaceRegistry,
		tunnelManager:       tunnelManager,
		cloudwatchClient:    cloudwatchClient,
		idleScheduler:       idleScheduler,
		policyManager:       policyManager,
		profileManager:      profileManager,                         // Singleton for filesystem consistency
		progressTracker:     NewProgressTracker(),                   // Launch progress monitoring (v0.7.2 - Issue #453)
		reducedMode:         reducedMode,                            // Reduced functionality mode when AWS credentials unavailable (Issue #356)
		testMode:            os.Getenv("PRISM_TEST_MODE") == "true", // E2E test mode - skip AWS operations
		gdewTracker:         project.NewGDEWTracker(),               // GDEW credit tracker (v0.11.0 - Issue #206)
	}

	// Approval manager (v0.12.0 - Issue #148/#149/#153/#157) comes from the seam-backed managers.
	server.approvalManager = seamMgrs.approval

	// Phase 2 (#644): register the spend observer as a StateMonitor tick hook so per-instance spend
	// accrues into the budgetengine ledger. Self-throttled (10m estimate cadence) despite the 10s
	// tick. Loads instances from the state manager. Cost Explorer reconciliation is opt-in (config /
	// PRISM_COST_RECONCILIATION) and only wired when a live AWS manager is present; skipped in test
	// mode. The billed-cost func is bound from awsManager so the observer stays AWS-free and testable.
	if spendLedger != nil {
		opts := spendObserverOptions{
			reconcileEnabled:  config.CostReconciliationEnabled,
			reconcileInterval: config.GetCostReconciliationInterval(),
		}
		if awsManager != nil {
			opts.billedCost = awsManager.GetBilledCost
		}
		observer := newSpendObserver(spendLedger, func() ([]types.Instance, error) {
			st, err := stateManager.LoadState()
			if err != nil {
				return nil, err
			}
			out := make([]types.Instance, 0, len(st.Instances))
			for _, inst := range st.Instances {
				out = append(out, inst)
			}
			return out, nil
		}, server.testMode, opts)
		stateMonitor.SetTickHook(func() { observer.Observe(context.Background()) })
		if opts.reconcileEnabled && opts.billedCost != nil {
			logger.Info("Budget cost reconciliation enabled", "interval", opts.reconcileInterval.String())
		}
	}

	// Initialize course manager (v0.14.0 - Issue #45)
	if cm, err := course.NewManager(); err == nil {
		server.courseManager = cm
	} else {
		logger.Info(fmt.Sprintf("Warning: course manager unavailable: %v", err))
	}

	// Initialize workshop manager (v0.18.0 - Issue #135/#178-#184)
	if wm, err := workshop.NewManager(); err == nil {
		server.workshopManager = wm
	} else {
		logger.Info(fmt.Sprintf("Warning: workshop manager unavailable: %v", err))
	}

	// Load persisted idle schedules into scheduler (Issue #288)
	if server.idleScheduler != nil {
		server.loadIdleSchedules()
		logger.Info("Idle schedules loaded from persisted state")
	}

	// Print reduced mode banner if AWS credentials unavailable (Issue #356)
	if reducedMode {
		logger.Warn("Starting in reduced functionality mode")
		logger.Info(getReducedModeBanner())
	}

	// Configure budget tracker with action executor
	budgetTracker.SetActionExecutor(server)

	// Wire AlertDispatcher into budget tracker (Issue #138).
	// If PRISM_SLACK_WEBHOOK is set, use a WebhookDispatcher; otherwise keep the default LogDispatcher.
	if slackURL := os.Getenv("PRISM_SLACK_WEBHOOK"); slackURL != "" {
		webhookDisp := alerting.NewWebhookDispatcher(alerting.WebhookConfig{
			Channels: []alerting.Channel{{Type: alerting.ChannelSlack, Target: slackURL}},
		})
		server.SetAlerterOnBudgetTracker(webhookDisp)
		logger.Info("Budget alert dispatcher configured", "channel", "slack")
	}

	// Initialize recovery and health monitoring (need server reference)
	server.recoveryManager = NewRecoveryManager(stabilityManager, nil) // Will be set after server creation
	server.healthMonitor = NewHealthMonitor(stateManager, stabilityManager, server.recoveryManager, performanceMonitor)

	// Initialize launch manager (if needed)
	// server.launchManager = NewLaunchManager(server)

	// Initialize sleep/wake monitor (v0.5.7 - Issue #91)
	sleepWakeConfig := sleepwake.DefaultConfig()
	instanceManager := newInstanceManager(server)
	sleepWakeMonitor, err := sleepwake.NewMonitor(sleepWakeConfig, instanceManager)
	if err != nil {
		logger.Warn("Failed to initialize sleep/wake monitor", "error", err)
		// Non-fatal: continue without sleep/wake monitoring
	} else {
		server.sleepWakeMonitor = sleepWakeMonitor
		logger.Info("Sleep/wake monitor initialized", "platform", sleepWakeMonitor.GetStatus().Platform)
	}

	// Setup HTTP routes
	mux := http.NewServeMux()
	server.setupRoutes(mux)

	server.httpServer = &http.Server{
		Addr:    "127.0.0.1:" + port,                // Bind to localhost only (#593)
		Handler: http.MaxBytesHandler(mux, 100<<20), // 100MB request body limit (#592)
		// ReadTimeout: Keep short for receiving requests (30s is reasonable)
		ReadTimeout: 30 * time.Second,
		// WriteTimeout: Allow for long-running AWS operations (instance launch = 2-5min)
		// Set to 10 minutes to accommodate instance launch with system status checks
		WriteTimeout: 10 * time.Minute,
		// IdleTimeout: Keep connections alive for multiple requests
		IdleTimeout: 2 * time.Minute,
	}

	// Set HTTP server reference in recovery manager
	server.recoveryManager.HTTPServer = server.httpServer

	return server, nil
}

// NewServerForTesting creates a new daemon server with test mode enabled
// Test mode skips AWS operations to allow unit testing without AWS credentials
func NewServerForTesting(port string) (*Server, error) {
	server, err := NewServer(port)
	if err != nil {
		return nil, err
	}
	server.testMode = true
	return server, nil
}

// Start starts the daemon server
// acquireSingletonLock enforces the daemon singleton constraint. It discovers any
// existing daemon processes, performs a graceful takeover if needed, and registers
// the current process in the daemon registry.
func (s *Server) acquireSingletonLock() error {
	currentPID := os.Getpid()
	existingProcesses, err := s.processManager.FindDaemonProcesses()
	if err == nil && len(existingProcesses) > 0 {
		for _, proc := range existingProcesses {
			if proc.PID != currentPID && s.processManager.IsProcessRunning(proc.PID) {
				logger.Info("Found existing daemon, performing singleton takeover", "existing_pid", proc.PID)
				if err := s.processManager.GracefulShutdown(proc.PID); err != nil {
					logger.Warn("Failed to gracefully shutdown existing daemon", "error", err, "pid", proc.PID)
				}
				portReleased := false
				for i := 0; i < 10; i++ {
					time.Sleep(1 * time.Second)
					listener, listenErr := net.Listen("tcp", fmt.Sprintf(":%s", s.port))
					if listenErr == nil {
						listener.Close()
						portReleased = true
						break
					}
				}
				if !portReleased {
					return fmt.Errorf("timeout waiting for existing daemon (PID: %d) to release port %s", proc.PID, s.port)
				}
				logger.Info("Singleton lock acquired after takeover", "pid", currentPID)
			}
		}
	}
	configPath := fmt.Sprintf("%s/.prism", os.Getenv("HOME"))
	if err := s.processManager.RegisterDaemon(currentPID, configPath, ""); err != nil {
		logger.Warn("Failed to register daemon", "error", err)
	} else if len(existingProcesses) == 0 {
		logger.Info("Singleton lock acquired", "pid", currentPID)
	}
	return nil
}

// startShutdownHandler waits for OS signals and performs graceful shutdown.
func (s *Server) startShutdownHandler(cancel context.CancelFunc) {
	go func() {
		c := make(chan os.Signal, 1)
		signal.Notify(c, os.Interrupt, syscall.SIGTERM)
		<-c
		logger.Info("Shutting down daemon with stability management")
		shutCtx, shutCancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer shutCancel()
		cancel()
		if err := s.recoveryManager.GracefulShutdown(shutCtx); err != nil {
			logger.Warn("Graceful shutdown had issues", "error", err)
		}
		if err := s.processManager.UnregisterDaemon(os.Getpid()); err != nil {
			logger.Warn("Failed to unregister daemon", "error", err)
		}
		s.stopIntegratedMonitoring()
		if s.sleepWakeMonitor != nil {
			if err := s.sleepWakeMonitor.Stop(); err != nil {
				logger.Warn("Failed to stop sleep/wake monitor", "error", err)
			}
		}
		s.stateMonitor.Stop()
		s.storageStateMonitor.Stop()
		if err := s.securityManager.Stop(); err != nil {
			logger.Warn("Failed to stop security manager", "error", err)
		}
	}()
}

func (s *Server) Start() error {
	logger.Info("Starting Prism daemon", "port", s.port)

	// Security warnings (#595, #597)
	if os.Getenv("PRISM_TEST_MODE") == "true" {
		logger.Warn("PRISM_TEST_MODE is enabled — ALL AUTHENTICATION IS BYPASSED")
		logger.Warn("Do not use this setting in production environments")
	}

	if err := s.acquireSingletonLock(); err != nil {
		return err
	}

	// Start daemon stability and monitoring systems
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	logger.Info("Starting stability management systems")
	go s.performanceMonitor.Start(ctx)
	go s.reliabilityManager.Start(ctx)
	go s.stabilityManager.Start(ctx)
	go s.healthMonitor.Start(ctx)

	// Start background state monitor for async instance state tracking (v0.5.8)
	if err := s.stateMonitor.Start(); err != nil {
		logger.Warn("Failed to start state monitor", "error", err)
	}

	// Start background storage state monitor for async volume state tracking
	if err := s.storageStateMonitor.Start(); err != nil {
		logger.Warn("Failed to start storage state monitor", "error", err)
	}

	// Enable memory management
	s.stabilityManager.EnableForceGC(true)
	logger.Info("Daemon stability systems started")

	// Start security manager
	if err := s.securityManager.Start(); err != nil {
		logger.Warn("Failed to start security manager", "error", err)
		s.stabilityManager.RecordError("security", "startup_failed", err.Error(), ErrorSeverityHigh)
	} else {
		logger.Info("Security manager started")
		s.stabilityManager.RecordRecovery("security", "startup_failed")
	}

	// Start integrated autonomous monitoring if idle detection is enabled
	s.startIntegratedMonitoring()

	// Start sleep/wake monitor (v0.5.7 - Issue #91)
	if s.sleepWakeMonitor != nil {
		if err := s.sleepWakeMonitor.Start(); err != nil {
			logger.Warn("Failed to start sleep/wake monitor", "error", err)
			s.stabilityManager.RecordError("sleepwake", "startup_failed", err.Error(), ErrorSeverityMedium)
		} else {
			logger.Info("Sleep/wake monitor started")
			s.stabilityManager.RecordRecovery("sleepwake", "startup_failed")
		}
	}

	// Start v0.12.0 background governance tickers
	s.startGovernanceTickers(ctx)

	// Handle graceful shutdown with recovery manager
	s.startShutdownHandler(cancel)

	// Use TLS if cert/key exist in ~/.prism/tls/ (#594)
	certPath := filepath.Join(os.Getenv("HOME"), ".prism", "tls", "cert.pem")
	keyPath := filepath.Join(os.Getenv("HOME"), ".prism", "tls", "key.pem")
	if _, err := os.Stat(certPath); err == nil {
		if _, err := os.Stat(keyPath); err == nil {
			logger.Info("TLS enabled", "cert", certPath)
			return s.httpServer.ListenAndServeTLS(certPath, keyPath)
		}
	}
	logger.Info("TLS not configured (no cert at ~/.prism/tls/cert.pem) — using HTTP")
	return s.httpServer.ListenAndServe() //nolint:gosec // daemon binds localhost only; TLS optional via ~/.prism/tls/cert.pem
}

// Stop stops the daemon server gracefully
func (s *Server) Stop() error {
	logger.Info("Gracefully stopping daemon server")

	// Unregister this daemon instance
	pid := os.Getpid()
	if err := s.processManager.UnregisterDaemon(pid); err != nil {
		logger.Warn("Failed to unregister daemon", "error", err)
	}

	// Stop security manager
	if err := s.securityManager.Stop(); err != nil {
		logger.Warn("Failed to stop security manager", "error", err)
	}

	// Stop integrated monitoring
	s.stopIntegratedMonitoring()

	// Stop sleep/wake monitor (v0.5.7 - Issue #91)
	if s.sleepWakeMonitor != nil {
		if err := s.sleepWakeMonitor.Stop(); err != nil {
			logger.Warn("Failed to stop sleep/wake monitor", "error", err)
		}
	}

	// Stop state monitor (v0.5.8)
	s.stateMonitor.Stop()

	// Stop storage state monitor
	s.storageStateMonitor.Stop()

	// Stop idle scheduler to prevent goroutine leaks
	if s.idleScheduler != nil {
		s.idleScheduler.Stop()
		logger.Info("Idle scheduler stopped")
	}

	// Stop alert manager to prevent goroutine leaks
	if s.alertManager != nil {
		s.alertManager.Stop()
		logger.Info("Alert manager stopped")
	}

	// Shutdown HTTP server with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	if err := s.httpServer.Shutdown(ctx); err != nil {
		return fmt.Errorf("daemon server shutdown failed: %w", err)
	}

	logger.Info("Daemon server stopped successfully")
	return nil
}

// Cleanup performs comprehensive cleanup for uninstallation
func (s *Server) Cleanup() error {
	logger.Info("Performing comprehensive daemon cleanup")

	// First stop the server if running
	if err := s.Stop(); err != nil {
		logger.Warn("Server stop failed during cleanup", "error", err)
	}

	// Clean up all daemon processes
	if err := s.processManager.CleanupProcesses(); err != nil {
		return fmt.Errorf("failed to cleanup daemon processes: %w", err)
	}

	logger.Info("Daemon cleanup completed successfully")
	return nil
}

// startGovernanceTickers starts the v0.12.0 background governance maintenance routines:
//   - Hourly: prune expired project members (#150), prune expired approval requests (#149)
//   - Daily:  auto-freeze projects whose grant period has ended (#152)
func (s *Server) startGovernanceTickers(ctx context.Context) {
	// Hourly member/approval expiry pruning
	go func() {
		ticker := time.NewTicker(1 * time.Hour)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if s.projectManager != nil {
					if removed, err := s.projectManager.PruneExpiredMembers(ctx); err != nil {
						logger.Warn("PruneExpiredMembers failed", "error", err)
					} else if len(removed) > 0 {
						logger.Info("Pruned expired project members", "count", len(removed))
					}
				}
				if s.approvalManager != nil {
					if count, err := s.approvalManager.PruneExpired(); err != nil {
						logger.Warn("PruneExpiredApprovals failed", "error", err)
					} else if count > 0 {
						logger.Info("Expired approval requests pruned", "count", count)
					}
				}
			}
		}
	}()

	// Every 5 minutes: stop/hibernate instances that have passed their ExpiresAt (#146)
	go func() {
		ticker := time.NewTicker(5 * time.Minute)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				s.checkExpiredInstances(ctx)
			}
		}
	}()

	// Daily grant period auto-freeze check
	go func() {
		ticker := time.NewTicker(24 * time.Hour)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if s.projectManager != nil {
					if frozen, err := s.projectManager.CheckGrantPeriods(ctx); err != nil {
						logger.Warn("CheckGrantPeriods failed", "error", err)
					} else if len(frozen) > 0 {
						logger.Info("Auto-froze projects at grant period end", "projects", frozen)
					}
				}
			}
		}
	}()

	// Daily course archiving sweep (#162)
	go func() {
		ticker := time.NewTicker(24 * time.Hour)
		defer ticker.Stop()
		s.runCourseArchivingSweep(ctx) // run once at startup
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				s.runCourseArchivingSweep(ctx)
			}
		}
	}()
}

// runCourseArchivingSweep archives courses that have passed their grace period (#162).
func (s *Server) runCourseArchivingSweep(ctx context.Context) {
	if s.courseManager == nil {
		return
	}
	eligible, err := s.courseManager.ListArchiveEligibleCourses(ctx)
	if err != nil {
		logger.Warn("ListArchiveEligibleCourses failed", "error", err)
		return
	}
	for _, c := range eligible {
		if _, err := s.archiveCourse(ctx, c.ID); err != nil {
			logger.Warn("archiveCourse sweep failed", "course_id", c.ID, "error", err)
		} else {
			logger.Info("Auto-archived course at semester end", "course_id", c.ID, "code", c.Code)
		}
	}
}

// checkExpiredInstances stops/hibernates any instances whose ExpiresAt has passed (#146)
func (s *Server) checkExpiredInstances(ctx context.Context) {
	st, err := s.stateManager.LoadState()
	if err != nil {
		return
	}

	now := time.Now()
	for _, inst := range st.Instances {
		if inst.ExpiresAt == nil || !now.After(*inst.ExpiresAt) {
			continue
		}
		if inst.State != "running" {
			continue
		}
		logger.Info("Time-boxed instance expired, stopping", "instance", inst.Name, "expires_at", inst.ExpiresAt)
		// Use hibernate if the instance supports it, otherwise stop
		if s.awsManager != nil {
			_ = s.awsManager.StopInstance(inst.Name)
		}
	}
}

// Legacy idle management removed

// setupRoutes configures all HTTP routes
func (s *Server) setupRoutes(mux *http.ServeMux) {
	// Define middleware for JSON responses, security headers, and logging (#599)
	jsonMiddleware := func(handler http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("X-Content-Type-Options", "nosniff")
			w.Header().Set("X-Frame-Options", "DENY")
			logger.Debug("API request", "method", r.Method, "path", r.URL.Path)
			// Record the request in status tracker
			s.statusTracker.RecordRequest()
			handler(w, r)
		}
	}

	// CORS middleware — restricted to known local origins (#598)
	allowedOrigins := map[string]bool{
		"http://localhost:3000": true,
		"wails://localhost":     true,
	}
	corsMiddleware := func(handler http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")
			if allowedOrigins[origin] {
				w.Header().Set("Access-Control-Allow-Origin", origin)
				w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
				w.Header().Set("Access-Control-Allow-Headers", "Content-Type, X-API-Key")
				w.Header().Set("Access-Control-Max-Age", "3600")
			}

			if r.Method == "OPTIONS" {
				w.WriteHeader(http.StatusNoContent)
				return
			}

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
			logger.Debug("Operation started", "id", opID, "type", opType, "method", r.Method, "path", r.URL.Path)

			// Record start time for duration tracking
			startTime := time.Now()

			// Ensure operation is always marked as completed
			defer func() {
				s.statusTracker.EndOperationWithType(opType)
				logger.Debug("Operation completed", "id", opID, "type", opType, "duration", time.Since(startTime))
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
			corsMiddleware, // Add CORS first for web development
			jsonMiddleware,
			operationTrackingMiddleware,
			versionHeaderMiddleware,
			s.awsHeadersMiddleware,
			s.authMiddleware,
		)
	}

	// Middleware for AWS-requiring endpoints (Issue #356)
	applyAWSMiddleware := func(handler http.HandlerFunc) http.HandlerFunc {
		return s.combineMiddleware(
			handler,
			corsMiddleware, // Add CORS first for web development
			jsonMiddleware,
			operationTrackingMiddleware,
			versionHeaderMiddleware,
			s.awsHeadersMiddleware,
			s.authMiddleware,
			s.requireAWSMiddleware, // Check AWS credentials before proceeding
		)
	}

	// API version information endpoint
	mux.HandleFunc("/api/versions", applyMiddleware(s.handleAPIVersions))

	// Register v1 endpoints
	s.registerV1Routes(mux, applyMiddleware, applyAWSMiddleware)

	// API path matcher to handle any valid API request
	// This allows proper versioning of new paths that may be added in the future
	mux.HandleFunc("/api/", applyMiddleware(s.handleUnknownAPI))
}

// registerV1Routes registers all API v1 routes
func (s *Server) registerV1Routes(mux *http.ServeMux, applyMiddleware func(http.HandlerFunc) http.HandlerFunc, applyAWSMiddleware func(http.HandlerFunc) http.HandlerFunc) {
	// Health check (no AWS required)
	mux.HandleFunc("/api/v1/ping", applyMiddleware(s.handlePing))
	mux.HandleFunc("/api/v1/status", applyMiddleware(s.handleStatus))
	mux.HandleFunc("/api/v1/shutdown", applyMiddleware(s.handleShutdown))
	mux.HandleFunc("/api/v1/update/check", applyMiddleware(s.handleUpdateCheck))

	// Authentication (no AWS required)
	mux.HandleFunc("/api/v1/auth", applyMiddleware(s.handleAuth))

	// User authentication (no AWS required)
	mux.HandleFunc("/api/v1/authenticate", applyMiddleware(s.handleAuthenticate))

	// Profile management (no AWS required - local profiles)
	mux.HandleFunc("/api/v1/profiles", applyMiddleware(s.handleProfiles))
	mux.HandleFunc("/api/v1/profiles/current", applyMiddleware(s.handleGetCurrentProfile))
	mux.HandleFunc("/api/v1/profiles/", applyMiddleware(s.handleProfileOperations))

	// Group management (requires AWS)
	mux.HandleFunc("/api/v1/groups", applyAWSMiddleware(s.handleGroups))
	mux.HandleFunc("/api/v1/groups/", applyAWSMiddleware(s.handleGroupOperations))

	// Instance operations (requires AWS)
	mux.HandleFunc("/api/v1/instances", applyAWSMiddleware(s.handleInstances))
	mux.HandleFunc("/api/v1/instances/", applyAWSMiddleware(s.handleInstanceOperations))

	// EC2 Capacity Blocks (#63)
	mux.HandleFunc("/api/v1/capacity-blocks", applyAWSMiddleware(s.handleCapacityBlocks))
	mux.HandleFunc("/api/v1/capacity-blocks/", applyAWSMiddleware(s.handleCapacityBlockByID))

	// Storage analytics (#23)
	mux.HandleFunc("/api/v1/storage/analytics", applyAWSMiddleware(s.handleStorageAnalytics))
	mux.HandleFunc("/api/v1/storage/analytics/", applyAWSMiddleware(s.handleStorageAnalytics))

	// Tunnel operations (requires AWS - needs instances)
	mux.HandleFunc("/api/v1/tunnels", applyAWSMiddleware(s.handleTunnels))

	// Log operations (requires AWS - needs instances)
	mux.HandleFunc("/api/v1/logs", applyAWSMiddleware(s.handleLogs))
	mux.HandleFunc("/api/v1/logs/", applyAWSMiddleware(s.handleLogOperations))

	// Template operations (no AWS required - local operations)
	mux.HandleFunc("/api/v1/templates", applyMiddleware(s.handleTemplates))
	mux.HandleFunc("/api/v1/templates/", applyMiddleware(s.handleTemplateInfo))

	// Template application operations (requires AWS - launches instances)
	mux.HandleFunc("/api/v1/templates/apply", applyAWSMiddleware(s.handleTemplateApply))
	mux.HandleFunc("/api/v1/templates/diff", applyAWSMiddleware(s.handleTemplateDiff))

	// Volume operations (requires AWS)
	mux.HandleFunc("/api/v1/volumes", applyAWSMiddleware(s.handleVolumes))
	mux.HandleFunc("/api/v1/volumes/", applyAWSMiddleware(s.handleVolumeOperations))

	// Storage operations (requires AWS)
	mux.HandleFunc("/api/v1/storage", applyAWSMiddleware(s.handleStorage))
	mux.HandleFunc("/api/v1/storage/", applyAWSMiddleware(s.handleStorageOperations))

	// Volume and storage sync operations (requires AWS)
	mux.HandleFunc("/api/v1/volumes/sync", applyAWSMiddleware(s.handleSyncAllVolumes))
	mux.HandleFunc("/api/v1/storage/sync", applyAWSMiddleware(s.handleSyncAllStorage))

	// Storage transfer operations (requires AWS - S3-backed file transfers) (v0.5.7)
	mux.HandleFunc("/api/v1/storage/transfer", applyAWSMiddleware(s.handleStorageTransfer))
	mux.HandleFunc("/api/v1/storage/transfer/", applyMiddleware(s.handleStorageTransferOperations))

	// Instance snapshot operations (AMI-based full backups)
	mux.HandleFunc("/api/v1/snapshots", applyMiddleware(s.handleSnapshots))
	mux.HandleFunc("/api/v1/snapshots/", applyMiddleware(s.handleSnapshotOperations))

	// S3 file-level backup operations (Issue #478)
	mux.HandleFunc("/api/v1/backups", applyAWSMiddleware(s.handleBackups))
	mux.HandleFunc("/api/v1/backups/", applyAWSMiddleware(s.handleBackupOperations))

	// User operations (Phase 5A.3: REST API Integration)
	mux.HandleFunc("/api/v1/users", applyMiddleware(s.handleResearchUsers))
	mux.HandleFunc("/api/v1/users/", applyMiddleware(s.handleResearchUserOperations))

	// Idle policy operations
	s.RegisterIdleRoutes(mux, applyMiddleware)

	// Rightsizing analysis operations
	s.registerRightsizingRoutes(mux, applyMiddleware)

	// Process management operations
	mux.HandleFunc("/api/v1/daemon/processes", applyMiddleware(s.handleDaemonProcesses))
	mux.HandleFunc("/api/v1/daemon/cleanup", applyMiddleware(s.handleDaemonCleanup))

	// Project management operations
	mux.HandleFunc("/api/v1/projects", applyMiddleware(s.handleProjectOperations))
	mux.HandleFunc("/api/v1/projects/", applyMiddleware(s.handleProjectByID))

	// Admin dashboard endpoints (v0.12.0)
	mux.HandleFunc("/api/v1/admin/approvals", applyMiddleware(s.handleAdminApprovals))

	// Budget management operations (v0.5.10 multi-budget system)
	mux.HandleFunc("/api/v1/budgets", applyMiddleware(s.handleBudgetOperations))
	mux.HandleFunc("/api/v1/budgets/", applyMiddleware(s.handleBudgetByID))

	// Allocation management operations (v0.5.10 multi-budget system)
	mux.HandleFunc("/api/v1/allocations", applyMiddleware(s.handleAllocationOperations))
	mux.HandleFunc("/api/v1/allocations/", applyMiddleware(s.handleAllocationByID))

	// Reallocation operations (v0.5.10 Issue #99)
	mux.HandleFunc("/api/v1/reallocations", applyMiddleware(s.handleReallocationOperations))

	// Cost rollup and reporting operations (v0.5.10 Issue #100)
	mux.HandleFunc("/api/v1/reports/rollup", applyMiddleware(s.handleBudgetRollupReport))
	mux.HandleFunc("/api/v1/reports/projects", applyMiddleware(s.handleProjectCostRollup))

	// Invitation management operations (v0.5.11 user invitation system)
	// Note: Project invitation routes handled by handleProjectByID
	mux.HandleFunc("/api/v1/invitations/", applyMiddleware(s.handleInvitationOperations))

	// Course / Education management (v0.14.0 - Issue #45)
	mux.HandleFunc("/api/v1/courses", applyMiddleware(s.handleCourseOperations))
	mux.HandleFunc("/api/v1/courses/", applyMiddleware(s.handleCourseByID))

	// Workshop / Event management (v0.18.0 - Issue #135/#178-#184)
	mux.HandleFunc("/api/v1/workshops", applyMiddleware(s.handleWorkshopOperations))
	mux.HandleFunc("/api/v1/workshops/", applyMiddleware(s.handleWorkshopByID))

	// Security management endpoints (Phase 4: Security integration)
	mux.HandleFunc("/api/v1/security/status", applyMiddleware(s.handleSecurityStatus))
	mux.HandleFunc("/api/v1/security/health", applyMiddleware(s.handleSecurityHealth))

	// IAM validation (Issue #35)
	mux.HandleFunc("/api/v1/iam/validate", applyAWSMiddleware(s.handleIAMValidate))
	mux.HandleFunc("/api/v1/iam/permissions", applyMiddleware(s.handleIAMPermissions))

	// RBAC (Issue #36)
	mux.HandleFunc("/api/v1/rbac/", applyMiddleware(s.handleRBACOperations))

	// Audit log (Issue #26)
	mux.HandleFunc("/api/v1/audit/events", applyMiddleware(s.handleAuditEvents))
	mux.HandleFunc("/api/v1/security/dashboard", applyMiddleware(s.handleSecurityDashboard))
	mux.HandleFunc("/api/v1/security/correlations", applyMiddleware(s.handleSecurityCorrelations))

	// Rate limiting management endpoints (v0.5.12 Issue #107)
	mux.HandleFunc("/api/v1/rate-limit/status", applyMiddleware(s.handleRateLimitStatus))
	mux.HandleFunc("/api/v1/rate-limit/configure", applyMiddleware(s.handleRateLimitConfigure))
	mux.HandleFunc("/api/v1/rate-limit/reset", applyMiddleware(s.handleRateLimitReset))

	// Launch throttling endpoints (v0.6.0 Issue #90)
	mux.HandleFunc("/api/v1/throttling/status", applyMiddleware(s.handleThrottlingStatus))
	mux.HandleFunc("/api/v1/throttling/configure", applyMiddleware(s.handleThrottlingConfigure))
	mux.HandleFunc("/api/v1/throttling/remaining", applyMiddleware(s.handleThrottlingRemaining))
	mux.HandleFunc("/api/v1/throttling/projects/overrides", applyMiddleware(s.handleListProjectOverrides))
	mux.HandleFunc("/api/v1/throttling/projects/", applyMiddleware(s.handleProjectThrottlingOperations))

	// Sleep/wake monitoring endpoints (v0.5.7 Issue #91)
	s.RegisterSleepWakeRoutes(mux, applyMiddleware)

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
	logger.Debug("Legacy monitoring removed - using universal idle detection")
}

// stopIntegratedMonitoring removed - using universal idle detection
func (s *Server) stopIntegratedMonitoring() {
	logger.Debug("Legacy monitoring removed - using universal idle detection")
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

	// Find instances belonging to this project by filtering on ProjectID field
	var hibernatedCount int
	var errors []string
	var skippedCount int

	for _, instance := range instances {
		// Only process instances that belong to this project
		if instance.ProjectID != projectID {
			skippedCount++
			continue
		}

		// Only hibernate running instances
		if instance.State == "running" {
			if err := s.awsManager.HibernateInstance(instance.Name); err != nil {
				errors = append(errors, fmt.Sprintf("failed to hibernate %s: %v", instance.Name, err))
			} else {
				hibernatedCount++
				logger.Info("Budget auto action hibernated instance", "instance", instance.Name, "project", projectID)
			}
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("hibernated %d instances but encountered errors: %s", hibernatedCount, strings.Join(errors, ", "))
	}

	logger.Info("Budget auto action completed hibernation", "hibernated", hibernatedCount, "project", projectID, "skipped", skippedCount)
	return nil
}

// ExecuteStopAll stops all instances for a project
func (s *Server) ExecuteStopAll(projectID string) error {
	// Get all instances
	instances, err := s.awsManager.ListInstances()
	if err != nil {
		return fmt.Errorf("failed to list instances for stopping: %w", err)
	}

	// Find instances belonging to this project by filtering on ProjectID field
	var stoppedCount int
	var errors []string
	var skippedCount int

	for _, instance := range instances {
		// Only process instances that belong to this project
		if instance.ProjectID != projectID {
			skippedCount++
			continue
		}

		// Only stop running instances
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

	logger.Info("Budget auto action completed stop", "stopped", stoppedCount, "project", projectID, "skipped", skippedCount)
	return nil
}

// ExecutePreventLaunch sets a flag to prevent new launches for a project
func (s *Server) ExecutePreventLaunch(projectID string) error {
	// Prevent new launches via project manager
	ctx := context.Background()
	if err := s.projectManager.PreventLaunches(ctx, projectID); err != nil {
		return fmt.Errorf("failed to prevent launches: %w", err)
	}

	logger.Info("Budget auto action prevented new launches", "project", projectID)
	return nil
}

// SetAlerterOnBudgetTracker configures an AlertDispatcher on the budget tracker.
// Called at startup when PRISM_SLACK_WEBHOOK or another notifier is configured.
func (s *Server) SetAlerterOnBudgetTracker(d alerting.AlertDispatcher) {
	if s.budgetTracker != nil {
		s.budgetTracker.SetAlerter(d)
	}
	if s.projectManager != nil {
		s.projectManager.SetAlerter(d)
	}
}
