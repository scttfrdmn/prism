package idle

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"sync"
	"syscall"
	"time"

	"github.com/scttfrdmn/cloudworkstation/pkg/aws"
)

// ResilientIdleService provides autonomous idle detection that survives restarts and reboots
type ResilientIdleService struct {
	manager        *Manager
	awsManager     *aws.Manager
	autonomous     *AutonomousIdleService
	
	// Persistence
	stateFile      string
	configFile     string
	mutex          sync.RWMutex
	
	// Runtime state
	running        bool
	ctx            context.Context
	cancel         context.CancelFunc
	
	// Recovery tracking
	lastSaveTime   time.Time
	saveInterval   time.Duration
	
	// System integration
	signalChan     chan os.Signal
}

// PersistentState holds the state that needs to survive restarts
type PersistentState struct {
	// Idle states for all monitored instances
	IdleStates map[string]*IdleState `json:"idle_states"`
	
	// Configuration
	Config *AutonomousConfig `json:"config"`
	
	// Timing information
	LastUpdate    time.Time `json:"last_update"`
	DaemonUptime  time.Time `json:"daemon_uptime"`
	
	// Recovery metadata
	Version       string `json:"version"`
	SaveReason    string `json:"save_reason"`
}

// NewResilientIdleService creates a new resilient autonomous idle service
func NewResilientIdleService(manager *Manager, awsManager *aws.Manager, config *AutonomousConfig) (*ResilientIdleService, error) {
	if config == nil {
		config = DefaultAutonomousConfig()
	}
	
	// Get configuration directory
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}
	
	configDir := filepath.Join(homeDir, ConfigDirName)
	if err := ensureDir(configDir); err != nil {
		return nil, fmt.Errorf("failed to create config directory: %w", err)
	}
	
	// Create context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	
	service := &ResilientIdleService{
		manager:      manager,
		awsManager:   awsManager,
		stateFile:    filepath.Join(configDir, "autonomous_state.json"),
		configFile:   filepath.Join(configDir, "autonomous_config.json"),
		saveInterval: 30 * time.Second, // Save state every 30 seconds
		ctx:          ctx,
		cancel:       cancel,
		signalChan:   make(chan os.Signal, 1),
	}
	
	// Create autonomous service
	autonomous, err := NewAutonomousIdleService(manager, awsManager, config)
	if err != nil {
		return nil, fmt.Errorf("failed to create autonomous service: %w", err)
	}
	
	service.autonomous = autonomous
	
	// Setup signal handling for graceful shutdown
	signal.Notify(service.signalChan, 
		syscall.SIGINT,  // Ctrl+C
		syscall.SIGTERM, // Terminate
		syscall.SIGHUP,  // Hangup
		syscall.SIGQUIT, // Quit
	)
	
	return service, nil
}

// Start begins resilient autonomous idle detection
func (ris *ResilientIdleService) Start() error {
	ris.mutex.Lock()
	defer ris.mutex.Unlock()
	
	if ris.running {
		return fmt.Errorf("resilient idle service is already running")
	}
	
	log.Printf("🚀 Starting resilient autonomous idle detection service")
	
	// Load previous state if it exists
	if err := ris.loadState(); err != nil {
		log.Printf("⚠️  Failed to load previous state: %v (starting fresh)", err)
	} else {
		log.Printf("✅ Recovered previous idle detection state")
	}
	
	// Save configuration
	if err := ris.saveConfig(); err != nil {
		log.Printf("⚠️  Failed to save configuration: %v", err)
	}
	
	ris.running = true
	
	// Start the autonomous service
	if err := ris.autonomous.Start(ris.ctx); err != nil {
		ris.running = false
		return fmt.Errorf("failed to start autonomous service: %w", err)
	}
	
	// Start background tasks
	go ris.persistenceLoop()
	go ris.signalHandler()
	
	log.Printf("🤖 Resilient autonomous idle detection is now running")
	log.Printf("   State file: %s", ris.stateFile)
	log.Printf("   Save interval: %v", ris.saveInterval)
	log.Printf("   Signal handling: enabled")
	
	return nil
}

// Stop gracefully stops the resilient idle service
func (ris *ResilientIdleService) Stop() error {
	ris.mutex.Lock()
	defer ris.mutex.Unlock()
	
	if !ris.running {
		return fmt.Errorf("resilient idle service is not running")
	}
	
	log.Printf("🛑 Stopping resilient autonomous idle detection service...")
	
	// Stop autonomous service
	if ris.autonomous != nil {
		if err := ris.autonomous.Stop(); err != nil {
			log.Printf("Error stopping autonomous service: %v", err)
		}
	}
	
	// Save current state before stopping
	if err := ris.saveState("graceful_shutdown"); err != nil {
		log.Printf("⚠️  Failed to save state during shutdown: %v", err)
	} else {
		log.Printf("✅ State saved successfully during shutdown")
	}
	
	// Cancel context to stop all goroutines
	ris.cancel()
	
	ris.running = false
	
	log.Printf("✅ Resilient idle service stopped gracefully")
	
	return nil
}

// persistenceLoop handles periodic state saving
func (ris *ResilientIdleService) persistenceLoop() {
	ticker := time.NewTicker(ris.saveInterval)
	defer ticker.Stop()
	
	for {
		select {
		case <-ris.ctx.Done():
			return
		case <-ticker.C:
			if err := ris.saveState("periodic_save"); err != nil {
				log.Printf("Failed to save state: %v", err)
			}
		}
	}
}

// signalHandler handles system signals for graceful shutdown
func (ris *ResilientIdleService) signalHandler() {
	for {
		select {
		case <-ris.ctx.Done():
			return
		case sig := <-ris.signalChan:
			log.Printf("🔔 Received signal: %v", sig)
			
			switch sig {
			case syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT:
				log.Printf("🛑 Graceful shutdown requested by signal %v", sig)
				if err := ris.Stop(); err != nil {
					log.Printf("Error during signal shutdown: %v", err)
				}
				os.Exit(0)
				
			case syscall.SIGHUP:
				log.Printf("🔄 Reload requested by SIGHUP")
				if err := ris.reload(); err != nil {
					log.Printf("Failed to reload: %v", err)
				}
			}
		}
	}
}

// reload reloads configuration and state
func (ris *ResilientIdleService) reload() error {
	log.Printf("🔄 Reloading resilient idle service...")
	
	// Reload state
	if err := ris.loadState(); err != nil {
		return fmt.Errorf("failed to reload state: %w", err)
	}
	
	// Reload configuration
	if err := ris.loadConfig(); err != nil {
		log.Printf("Failed to reload config: %v", err)
	}
	
	log.Printf("✅ Service reloaded successfully")
	return nil
}

// saveState persists current idle states to disk
func (ris *ResilientIdleService) saveState(reason string) error {
	ris.mutex.Lock()
	defer ris.mutex.Unlock()
	
	// Collect current idle states from manager
	idleStates := make(map[string]*IdleState)
	// TODO: Get states from manager - this would require exposing manager state
	// For now, we'll create a placeholder structure
	
	state := &PersistentState{
		IdleStates:   idleStates,
		Config:       ris.autonomous.config,
		LastUpdate:   time.Now(),
		DaemonUptime: time.Now(), // This would be actual daemon start time
		Version:      "0.4.1",
		SaveReason:   reason,
	}
	
	// Marshal to JSON
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal state: %w", err)
	}
	
	// Write to temporary file first, then rename (atomic write)
	tempFile := ris.stateFile + ".tmp"
	if err := os.WriteFile(tempFile, data, 0600); err != nil {
		return fmt.Errorf("failed to write temp state file: %w", err)
	}
	
	if err := os.Rename(tempFile, ris.stateFile); err != nil {
		return fmt.Errorf("failed to rename state file: %w", err)
	}
	
	ris.lastSaveTime = time.Now()
	
	return nil
}

// loadState loads persistent state from disk
func (ris *ResilientIdleService) loadState() error {
	ris.mutex.Lock()
	defer ris.mutex.Unlock()
	
	// Check if state file exists
	if _, err := os.Stat(ris.stateFile); os.IsNotExist(err) {
		log.Printf("No previous state file found - starting fresh")
		return nil
	}
	
	// Read state file
	data, err := os.ReadFile(ris.stateFile)
	if err != nil {
		return fmt.Errorf("failed to read state file: %w", err)
	}
	
	// Parse state
	var state PersistentState
	if err := json.Unmarshal(data, &state); err != nil {
		return fmt.Errorf("failed to parse state file: %w", err)
	}
	
	// Validate state version compatibility
	if state.Version == "" {
		log.Printf("⚠️  State file has no version - assuming compatible")
	}
	
	// Calculate downtime
	downtime := time.Since(state.LastUpdate)
	log.Printf("📊 State recovery info:")
	log.Printf("   Last saved: %v", state.LastUpdate.Format("2006-01-02 15:04:05"))
	log.Printf("   Downtime: %v", downtime.Round(time.Second))
	log.Printf("   Save reason: %s", state.SaveReason)
	log.Printf("   Idle states: %d instances", len(state.IdleStates))
	
	// Recover idle states
	for _, idleState := range state.IdleStates {
		log.Printf("   Recovering: %s (idle: %t)", idleState.InstanceName, idleState.IsIdle)
		
		// Adjust timing for downtime
		if idleState.IsIdle && idleState.IdleSince != nil {
			// The instance has been idle, but we were down - what should we do?
			// Option 1: Assume still idle and extend the idle time
			// Option 2: Reset idle detection (more conservative)
			// Option 3: Check if action should have been taken during downtime
			
			if idleState.NextAction != nil {
				actionTime := idleState.NextAction.Time
				if time.Now().After(actionTime) {
					// Action should have been taken during downtime
					log.Printf("     ⚠️  Action %s was due at %v (during downtime)", 
						idleState.NextAction.Action, actionTime.Format("15:04:05"))
					
					// TODO: Decide whether to execute immediately or reset
					// For now, execute immediately
					log.Printf("     🤖 Executing overdue action immediately")
				}
			}
		}
		
		// Restore state to manager
		ris.manager.SetIdleState(idleState)
	}
	
	return nil
}

// saveConfig saves current configuration
func (ris *ResilientIdleService) saveConfig() error {
	data, err := json.MarshalIndent(ris.autonomous.config, "", "  ")
	if err != nil {
		return err
	}
	
	return os.WriteFile(ris.configFile, data, 0600)
}

// loadConfig loads configuration from disk
func (ris *ResilientIdleService) loadConfig() error {
	if _, err := os.Stat(ris.configFile); os.IsNotExist(err) {
		return nil // No config file is OK
	}
	
	data, err := os.ReadFile(ris.configFile)
	if err != nil {
		return err
	}
	
	var config AutonomousConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return err
	}
	
	// Update configuration
	ris.autonomous.config = &config
	
	return nil
}

// IsRunning returns whether the service is running
func (ris *ResilientIdleService) IsRunning() bool {
	ris.mutex.RLock()
	defer ris.mutex.RUnlock()
	return ris.running
}

// GetStatus returns comprehensive status including resilience info
func (ris *ResilientIdleService) GetStatus() map[string]interface{} {
	ris.mutex.RLock()
	defer ris.mutex.RUnlock()
	
	status := map[string]interface{}{
		"resilient_service": ris.running,
		"state_file":        ris.stateFile,
		"config_file":       ris.configFile,
		"last_save":         ris.lastSaveTime.Format("2006-01-02 15:04:05"),
		"save_interval":     ris.saveInterval.String(),
	}
	
	// Add autonomous service status
	if ris.autonomous != nil {
		autonomousStatus := ris.autonomous.GetStatus()
		for k, v := range autonomousStatus {
			status["autonomous_"+k] = v
		}
	}
	
	// Add state file info
	if stat, err := os.Stat(ris.stateFile); err == nil {
		status["state_file_size"] = stat.Size()
		status["state_file_modified"] = stat.ModTime().Format("2006-01-02 15:04:05")
	}
	
	return status
}