// Package security provides integrated security management for CloudWorkstation
package security

import (
	"fmt"
	"sync"
	"time"

	"github.com/scttfrdmn/cloudworkstation/pkg/profile/security"
)

// SecurityManager coordinates all security components for CloudWorkstation
type SecurityManager struct {
	auditLogger       *security.SecurityAuditLogger
	monitor           *security.SecurityMonitor
	correlationEngine *security.SecurityCorrelationEngine
	registryClient    *security.SecureRegistryClient

	// Configuration
	config SecurityConfig

	// Runtime state
	mutex           sync.RWMutex
	isEnabled       bool
	isRunning       bool
	lastHealthCheck time.Time

	// Background monitoring
	stopChan    chan struct{}
	monitorDone chan struct{}
}

// SecurityConfig provides configuration for the security manager
type SecurityConfig struct {
	// Audit logging configuration
	AuditLogEnabled  bool `json:"audit_log_enabled"`
	LogRetentionDays int  `json:"log_retention_days"`

	// Monitoring configuration
	MonitoringEnabled bool          `json:"monitoring_enabled"`
	MonitorInterval   time.Duration `json:"monitor_interval"`
	AlertThreshold    string        `json:"alert_threshold"` // LOW, MEDIUM, HIGH, CRITICAL

	// Registry security configuration
	RegistrySecurityEnabled bool   `json:"registry_security_enabled"`
	RegistryURL             string `json:"registry_url"`

	// Correlation analysis configuration
	CorrelationEnabled bool          `json:"correlation_enabled"`
	AnalysisInterval   time.Duration `json:"analysis_interval"`

	// Health check configuration
	HealthCheckEnabled  bool          `json:"health_check_enabled"`
	HealthCheckInterval time.Duration `json:"health_check_interval"`
}

// SecurityStatus provides comprehensive security status information
type SecurityStatus struct {
	Enabled         bool                           `json:"enabled"`
	Running         bool                           `json:"running"`
	LastHealthCheck time.Time                      `json:"last_health_check"`
	Dashboard       *security.SecurityDashboard    `json:"dashboard,omitempty"`
	Correlations    []security.SecurityCorrelation `json:"recent_correlations,omitempty"`
	SystemHealth    *security.SystemHealthStatus   `json:"system_health,omitempty"`
	KeychainInfo    *security.KeychainInfo         `json:"keychain_info,omitempty"`
	Configuration   SecurityConfig                 `json:"configuration"`
}

// NewSecurityManager creates and initializes a new security manager
func NewSecurityManager(config SecurityConfig) (*SecurityManager, error) {
	// Validate configuration
	if err := validateSecurityConfig(config); err != nil {
		return nil, fmt.Errorf("invalid security configuration: %w", err)
	}

	manager := &SecurityManager{
		config:      config,
		isEnabled:   config.MonitoringEnabled || config.AuditLogEnabled || config.CorrelationEnabled,
		stopChan:    make(chan struct{}),
		monitorDone: make(chan struct{}),
	}

	// Initialize components based on configuration
	if err := manager.initializeComponents(); err != nil {
		return nil, fmt.Errorf("failed to initialize security components: %w", err)
	}

	return manager, nil
}

// Start starts the security manager and all enabled components
func (m *SecurityManager) Start() error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if !m.isEnabled {
		return fmt.Errorf("security manager is disabled")
	}

	if m.isRunning {
		return fmt.Errorf("security manager is already running")
	}

	// Start background monitoring
	go m.monitoringLoop()

	m.isRunning = true

	// Log security manager startup
	if m.auditLogger != nil {
		m.auditLogger.LogSecurityEvent(security.SecurityEvent{
			EventType: "security_manager_started",
			Success:   true,
			Details: map[string]interface{}{
				"audit_enabled":       m.config.AuditLogEnabled,
				"monitoring_enabled":  m.config.MonitoringEnabled,
				"correlation_enabled": m.config.CorrelationEnabled,
				"registry_enabled":    m.config.RegistrySecurityEnabled,
			},
		})
	}

	return nil
}

// Stop gracefully stops the security manager and all components
func (m *SecurityManager) Stop() error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if !m.isRunning {
		return nil
	}

	// Stop monitoring loop
	close(m.stopChan)

	// Wait for monitoring to stop
	select {
	case <-m.monitorDone:
	case <-time.After(10 * time.Second):
		// Force stop after timeout
	}

	// Close components
	if m.correlationEngine != nil {
		_ = m.correlationEngine.Close()
	}

	if m.auditLogger != nil {
		// Log security manager shutdown
		m.auditLogger.LogSecurityEvent(security.SecurityEvent{
			EventType: "security_manager_stopped",
			Success:   true,
			Details: map[string]interface{}{
				"shutdown_reason": "graceful_stop",
			},
		})

		_ = m.auditLogger.Close()
	}

	m.isRunning = false
	return nil
}

// GetSecurityStatus returns comprehensive security status
func (m *SecurityManager) GetSecurityStatus() (*SecurityStatus, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	status := &SecurityStatus{
		Enabled:         m.isEnabled,
		Running:         m.isRunning,
		LastHealthCheck: m.lastHealthCheck,
		Configuration:   m.config,
	}

	// Get security dashboard if monitoring is enabled
	if m.monitor != nil {
		dashboard, err := m.monitor.GetSecurityDashboard()
		if err == nil {
			status.Dashboard = dashboard
		}
	}

	// Get recent correlations if correlation engine is enabled
	if m.correlationEngine != nil {
		correlations, err := m.correlationEngine.AnalyzeSecurityEvents()
		if err == nil {
			status.Correlations = correlations
		}
	}

	// Get keychain information
	keychainInfo, err := security.GetKeychainInfo()
	if err == nil {
		status.KeychainInfo = keychainInfo
	}

	return status, nil
}

// LogSecurityEvent logs a security event through the audit logger
func (m *SecurityManager) LogSecurityEvent(eventType string, success bool, deviceID string, details map[string]interface{}) {
	if m.auditLogger == nil {
		return
	}

	m.auditLogger.LogSecurityEvent(security.SecurityEvent{
		EventType: eventType,
		Success:   success,
		DeviceID:  deviceID,
		Details:   details,
	})
}

// LogDeviceRegistration logs device registration events
func (m *SecurityManager) LogDeviceRegistration(deviceID, invitationToken string, success bool, errorCode string, details map[string]interface{}) {
	if m.auditLogger == nil {
		return
	}

	m.auditLogger.LogDeviceRegistration(deviceID, invitationToken, success, errorCode, details)
}

// LogAccessAttempt logs access attempt events
func (m *SecurityManager) LogAccessAttempt(deviceID string, success bool, reason string, details map[string]interface{}) {
	if m.auditLogger == nil {
		return
	}

	m.auditLogger.LogAccessAttempt(deviceID, success, reason, details)
}

// LogTamperAttempt logs tamper detection events
func (m *SecurityManager) LogTamperAttempt(deviceID, filePath, expectedHash, actualHash string) {
	if m.auditLogger == nil {
		return
	}

	m.auditLogger.LogTamperAttempt(deviceID, filePath, expectedHash, actualHash)
}

// ValidateDeviceBinding validates device binding for security
func (m *SecurityManager) ValidateDeviceBinding(deviceID string, expectedFingerprint map[string]interface{}) error {
	// Generate current device fingerprint
	currentFingerprint, err := security.GenerateDeviceFingerprint()
	if err != nil {
		m.LogAccessAttempt(deviceID, false, "fingerprint_generation_failed", map[string]interface{}{
			"error": err.Error(),
		})
		return fmt.Errorf("failed to generate device fingerprint: %w", err)
	}

	// Validate fingerprint against expected
	isValid, err := security.ValidateDeviceBinding(currentFingerprint.Hash)
	if err != nil {
		m.LogAccessAttempt(deviceID, false, "validation_error", map[string]interface{}{
			"error": err.Error(),
		})
		return fmt.Errorf("device binding validation error: %w", err)
	}
	if !isValid {
		m.LogAccessAttempt(deviceID, false, "device_binding_violation", map[string]interface{}{
			"expected_hash": expectedFingerprint,
			"actual_hash":   currentFingerprint.Hash,
		})
		return fmt.Errorf("device binding validation failed")
	}

	m.LogAccessAttempt(deviceID, true, "device_binding_validated", map[string]interface{}{
		"device_hash": currentFingerprint.Hash,
	})

	return nil
}

// PerformHealthCheck performs comprehensive security health check
func (m *SecurityManager) PerformHealthCheck() error {
	m.mutex.Lock()
	m.lastHealthCheck = time.Now()
	m.mutex.Unlock()

	// Check keychain provider
	if err := security.ValidateKeychainProvider(); err != nil {
		m.LogSecurityEvent("health_check_failed", false, "", map[string]interface{}{
			"component": "keychain_provider",
			"error":     err.Error(),
		})
		return fmt.Errorf("keychain provider health check failed: %w", err)
	}

	// Check audit logging if enabled
	if m.auditLogger != nil {
		// Test logging functionality
		m.LogSecurityEvent("health_check_test", true, "", map[string]interface{}{
			"component": "audit_logger",
			"timestamp": time.Now(),
		})
	}

	// Check monitoring if enabled
	if m.monitor != nil {
		_, err := m.monitor.GetSecurityDashboard()
		if err != nil {
			m.LogSecurityEvent("health_check_failed", false, "", map[string]interface{}{
				"component": "security_monitor",
				"error":     err.Error(),
			})
			return fmt.Errorf("security monitor health check failed: %w", err)
		}
	}

	m.LogSecurityEvent("health_check_completed", true, "", map[string]interface{}{
		"components_checked": m.getEnabledComponents(),
		"timestamp":          time.Now(),
	})

	return nil
}

// RegisterDevice securely registers a device with the registry
func (m *SecurityManager) RegisterDevice(invitationToken, deviceID string) error {
	if m.registryClient == nil {
		return fmt.Errorf("registry client not available")
	}

	return m.registryClient.RegisterDevice(invitationToken, deviceID)
}

// Private methods

func (m *SecurityManager) initializeComponents() error {
	var err error

	// Initialize audit logger if enabled
	if m.config.AuditLogEnabled {
		m.auditLogger, err = security.NewSecurityAuditLogger()
		if err != nil {
			return fmt.Errorf("failed to initialize audit logger: %w", err)
		}
	}

	// Initialize security monitor if enabled
	if m.config.MonitoringEnabled {
		m.monitor, err = security.NewSecurityMonitor()
		if err != nil {
			return fmt.Errorf("failed to initialize security monitor: %w", err)
		}
	}

	// Initialize correlation engine if enabled
	if m.config.CorrelationEnabled {
		m.correlationEngine, err = security.NewSecurityCorrelationEngine()
		if err != nil {
			return fmt.Errorf("failed to initialize correlation engine: %w", err)
		}
	}

	// Initialize secure registry client if enabled
	if m.config.RegistrySecurityEnabled {
		registryConfig := security.S3RegistryConfig{
			BucketName: "cloudworkstation-registry", // Default bucket name
			Region:     "us-west-2",                 // Default region
			Enabled:    true,
		}

		m.registryClient, err = security.NewSecureRegistryClient(registryConfig)
		if err != nil {
			return fmt.Errorf("failed to initialize secure registry client: %w", err)
		}
	}

	return nil
}

func (m *SecurityManager) monitoringLoop() {
	defer close(m.monitorDone)

	monitorTicker := time.NewTicker(m.config.MonitorInterval)
	defer monitorTicker.Stop()

	analysisTicker := time.NewTicker(m.config.AnalysisInterval)
	defer analysisTicker.Stop()

	healthTicker := time.NewTicker(m.config.HealthCheckInterval)
	defer healthTicker.Stop()

	for {
		select {
		case <-m.stopChan:
			return

		case <-monitorTicker.C:
			if m.monitor != nil {
				if err := m.monitor.MonitorSecurityEvents(); err != nil {
					m.LogSecurityEvent("monitoring_error", false, "", map[string]interface{}{
						"error": err.Error(),
					})
				}
			}

		case <-analysisTicker.C:
			if m.correlationEngine != nil {
				if _, err := m.correlationEngine.AnalyzeSecurityEvents(); err != nil {
					m.LogSecurityEvent("correlation_error", false, "", map[string]interface{}{
						"error": err.Error(),
					})
				}
			}

		case <-healthTicker.C:
			if m.config.HealthCheckEnabled {
				if err := m.PerformHealthCheck(); err != nil {
					m.LogSecurityEvent("health_check_failed", false, "", map[string]interface{}{
						"error": err.Error(),
					})
				}
			}
		}
	}
}

func (m *SecurityManager) getEnabledComponents() []string {
	components := make([]string, 0)

	if m.config.AuditLogEnabled {
		components = append(components, "audit_logger")
	}
	if m.config.MonitoringEnabled {
		components = append(components, "security_monitor")
	}
	if m.config.CorrelationEnabled {
		components = append(components, "correlation_engine")
	}
	if m.config.RegistrySecurityEnabled {
		components = append(components, "secure_registry")
	}

	return components
}

// Helper functions

func validateSecurityConfig(config SecurityConfig) error {
	if config.MonitorInterval <= 0 {
		config.MonitorInterval = 30 * time.Second
	}

	if config.AnalysisInterval <= 0 {
		config.AnalysisInterval = 5 * time.Minute
	}

	if config.HealthCheckInterval <= 0 {
		config.HealthCheckInterval = 15 * time.Minute
	}

	if config.LogRetentionDays <= 0 {
		config.LogRetentionDays = 30
	}

	validThresholds := map[string]bool{
		"LOW": true, "MEDIUM": true, "HIGH": true, "CRITICAL": true,
	}

	if config.AlertThreshold != "" && !validThresholds[config.AlertThreshold] {
		return fmt.Errorf("invalid alert threshold: %s", config.AlertThreshold)
	}

	return nil
}

// GetDefaultSecurityConfig returns default security configuration
func GetDefaultSecurityConfig() SecurityConfig {
	return SecurityConfig{
		AuditLogEnabled:         true,
		LogRetentionDays:        30,
		MonitoringEnabled:       true,
		MonitorInterval:         30 * time.Second,
		AlertThreshold:          "MEDIUM",
		RegistrySecurityEnabled: true,
		CorrelationEnabled:      true,
		AnalysisInterval:        5 * time.Minute,
		HealthCheckEnabled:      true,
		HealthCheckInterval:     15 * time.Minute,
	}
}
