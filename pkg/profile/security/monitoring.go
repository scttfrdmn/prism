// Package security provides security monitoring dashboard and alerting
package security

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"
)

// SecurityMonitor provides real-time security monitoring and alerting
type SecurityMonitor struct {
	auditLogger    *SecurityAuditLogger
	alertThresholds AlertThresholds
	alertHandler   AlertHandler
	metrics        *SecurityMetrics
}

// AlertThresholds defines thresholds for security alerts
type AlertThresholds struct {
	FailedAttemptsPerHour   int           `json:"failed_attempts_per_hour"`
	TamperAttemptsThreshold int           `json:"tamper_attempts_threshold"`
	MaxFailedDeviceBinding  int           `json:"max_failed_device_binding"`
	AlertWindow             time.Duration `json:"alert_window"`
	CriticalEventImmediate  bool          `json:"critical_event_immediate"`
}

// AlertHandler defines interface for handling security alerts
type AlertHandler interface {
	SendAlert(alert SecurityAlert) error
}

// SecurityAlert represents a security alert
type SecurityAlert struct {
	ID          string                 `json:"id"`
	Timestamp   time.Time              `json:"timestamp"`
	Severity    AlertSeverity          `json:"severity"`
	AlertType   string                 `json:"alert_type"`
	Title       string                 `json:"title"`
	Description string                 `json:"description"`
	DeviceID    string                 `json:"device_id,omitempty"`
	EventCount  int                    `json:"event_count"`
	Details     map[string]interface{} `json:"details"`
	Actions     []string               `json:"recommended_actions"`
}

// AlertSeverity defines alert severity levels
type AlertSeverity string

const (
	AlertSeverityLow      AlertSeverity = "LOW"
	AlertSeverityMedium   AlertSeverity = "MEDIUM"
	AlertSeverityHigh     AlertSeverity = "HIGH"
	AlertSeverityCritical AlertSeverity = "CRITICAL"
)

// SecurityMetrics tracks security-related metrics
type SecurityMetrics struct {
	TotalEvents           int                         `json:"total_events"`
	FailedAttempts        int                         `json:"failed_attempts"`
	SuccessfulOperations  int                         `json:"successful_operations"`
	TamperAttempts        int                         `json:"tamper_attempts"`
	DeviceRegistrations   int                         `json:"device_registrations"`
	AlertsGenerated       int                         `json:"alerts_generated"`
	EventTypeBreakdown    map[string]int              `json:"event_type_breakdown"`
	DeviceActivity        map[string]int              `json:"device_activity"`
	HourlyActivity        map[int]int                 `json:"hourly_activity"`
	LastUpdated           time.Time                   `json:"last_updated"`
	SecurityScore         int                         `json:"security_score"`
	ThreatLevel           string                      `json:"threat_level"`
	RecentCriticalEvents  []SecurityEvent             `json:"recent_critical_events"`
	KeychainProviderStats map[string]int              `json:"keychain_provider_stats"`
}

// SecurityDashboard provides consolidated security status
type SecurityDashboard struct {
	Status           string             `json:"status"`
	ThreatLevel      string             `json:"threat_level"`
	SecurityScore    int                `json:"security_score"`
	ActiveAlerts     []SecurityAlert    `json:"active_alerts"`
	Metrics          *SecurityMetrics   `json:"metrics"`
	Recommendations  []string           `json:"recommendations"`
	LastUpdate       time.Time          `json:"last_update"`
	SystemHealth     SystemHealthStatus `json:"system_health"`
}

// SystemHealthStatus provides system security health information
type SystemHealthStatus struct {
	KeychainStatus    string    `json:"keychain_status"`
	EncryptionStatus  string    `json:"encryption_status"`
	FileIntegrity     string    `json:"file_integrity"`
	DeviceBinding     string    `json:"device_binding"`
	AuditLogging      string    `json:"audit_logging"`
	LastHealthCheck   time.Time `json:"last_health_check"`
}

// NewSecurityMonitor creates a new security monitor
func NewSecurityMonitor() (*SecurityMonitor, error) {
	auditLogger, err := NewSecurityAuditLogger()
	if err != nil {
		return nil, fmt.Errorf("failed to create audit logger: %w", err)
	}

	// Default alert thresholds
	thresholds := AlertThresholds{
		FailedAttemptsPerHour:   10,
		TamperAttemptsThreshold: 1, // Any tamper attempt is critical
		MaxFailedDeviceBinding:  3,
		AlertWindow:             time.Hour,
		CriticalEventImmediate:  true,
	}

	monitor := &SecurityMonitor{
		auditLogger:     auditLogger,
		alertThresholds: thresholds,
		alertHandler:    &ConsoleAlertHandler{},
		metrics:         &SecurityMetrics{
			EventTypeBreakdown:    make(map[string]int),
			DeviceActivity:        make(map[string]int),
			HourlyActivity:        make(map[int]int),
			RecentCriticalEvents:  make([]SecurityEvent, 0),
			KeychainProviderStats: make(map[string]int),
		},
	}

	return monitor, nil
}

// MonitorSecurityEvents processes security events and generates alerts
func (m *SecurityMonitor) MonitorSecurityEvents() error {
	// Analyze recent audit logs
	events, err := m.loadRecentEvents()
	if err != nil {
		return fmt.Errorf("failed to load recent events: %w", err)
	}

	// Update metrics
	m.updateMetrics(events)

	// Analyze for security threats
	alerts := m.analyzeSecurityThreats(events)

	// Send alerts
	for _, alert := range alerts {
		if err := m.alertHandler.SendAlert(alert); err != nil {
			fmt.Printf("Warning: Failed to send alert %s: %v\n", alert.ID, err)
		}
		m.metrics.AlertsGenerated++
	}

	m.metrics.LastUpdated = time.Now()
	return nil
}

// GetSecurityDashboard returns comprehensive security status
func (m *SecurityMonitor) GetSecurityDashboard() (*SecurityDashboard, error) {
	// Refresh monitoring data
	if err := m.MonitorSecurityEvents(); err != nil {
		return nil, fmt.Errorf("failed to refresh monitoring data: %w", err)
	}

	// Check system health
	healthStatus, err := m.checkSystemHealth()
	if err != nil {
		return nil, fmt.Errorf("failed to check system health: %w", err)
	}

	// Get active alerts (from recent time window)
	activeAlerts := m.getActiveAlerts()

	// Calculate overall status and recommendations
	status, threatLevel := m.calculateOverallStatus()
	recommendations := m.generateRecommendations()

	dashboard := &SecurityDashboard{
		Status:          status,
		ThreatLevel:     threatLevel,
		SecurityScore:   m.metrics.SecurityScore,
		ActiveAlerts:    activeAlerts,
		Metrics:         m.metrics,
		Recommendations: recommendations,
		LastUpdate:      time.Now(),
		SystemHealth:    *healthStatus,
	}

	return dashboard, nil
}

// loadRecentEvents loads recent security events from audit logs
func (m *SecurityMonitor) loadRecentEvents() ([]SecurityEvent, error) {
	logPath := m.auditLogger.GetAuditLogPath()
	
	content, err := os.ReadFile(logPath)
	if err != nil {
		if os.IsNotExist(err) {
			return []SecurityEvent{}, nil // No events yet
		}
		return nil, fmt.Errorf("failed to read audit log: %w", err)
	}

	lines := strings.Split(strings.TrimSpace(string(content)), "\n")
	events := make([]SecurityEvent, 0, len(lines))

	// Parse recent events (last 24 hours)
	cutoff := time.Now().Add(-24 * time.Hour)

	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}

		var event SecurityEvent
		if err := json.Unmarshal([]byte(line), &event); err != nil {
			continue // Skip malformed lines
		}

		if event.Timestamp.After(cutoff) {
			events = append(events, event)
		}
	}

	return events, nil
}

// updateMetrics updates security metrics based on recent events
func (m *SecurityMonitor) updateMetrics(events []SecurityEvent) {
	// Reset counters
	m.metrics.TotalEvents = len(events)
	m.metrics.FailedAttempts = 0
	m.metrics.SuccessfulOperations = 0
	m.metrics.TamperAttempts = 0
	m.metrics.DeviceRegistrations = 0
	
	// Clear maps
	for k := range m.metrics.EventTypeBreakdown {
		delete(m.metrics.EventTypeBreakdown, k)
	}
	for k := range m.metrics.DeviceActivity {
		delete(m.metrics.DeviceActivity, k)
	}
	for k := range m.metrics.HourlyActivity {
		delete(m.metrics.HourlyActivity, k)
	}
	for k := range m.metrics.KeychainProviderStats {
		delete(m.metrics.KeychainProviderStats, k)
	}

	// Clear recent critical events
	m.metrics.RecentCriticalEvents = m.metrics.RecentCriticalEvents[:0]

	// Analyze events
	for _, event := range events {
		// Event type breakdown
		m.metrics.EventTypeBreakdown[event.EventType]++

		// Device activity
		if event.DeviceID != "" {
			m.metrics.DeviceActivity[event.DeviceID]++
		}

		// Hourly activity
		hour := event.Timestamp.Hour()
		m.metrics.HourlyActivity[hour]++

		// Success/failure tracking
		if event.Success {
			m.metrics.SuccessfulOperations++
		} else {
			m.metrics.FailedAttempts++
		}

		// Special event types
		switch event.EventType {
		case "tamper_detected":
			m.metrics.TamperAttempts++
			// tamper_detected events are automatically critical
			m.metrics.RecentCriticalEvents = append(m.metrics.RecentCriticalEvents, event)
		case "device_registration":
			m.metrics.DeviceRegistrations++
		case "keychain_operation":
			if provider, ok := event.Details["provider"].(string); ok {
				m.metrics.KeychainProviderStats[provider]++
			}
		}

		// Critical events (but avoid duplicating tamper_detected events)
		if event.Severity == "CRITICAL" && event.EventType != "tamper_detected" {
			m.metrics.RecentCriticalEvents = append(m.metrics.RecentCriticalEvents, event)
		}
	}

	// Calculate security score (0-100)
	m.metrics.SecurityScore = m.calculateSecurityScore()

	// Determine threat level
	m.metrics.ThreatLevel = m.calculateThreatLevel()
}

// analyzeSecurityThreats analyzes events for security threats and generates alerts
func (m *SecurityMonitor) analyzeSecurityThreats(events []SecurityEvent) []SecurityAlert {
	alerts := make([]SecurityAlert, 0)

	// Check for excessive failed attempts
	recentFailed := m.countRecentFailedAttempts(events)
	if recentFailed >= m.alertThresholds.FailedAttemptsPerHour {
		alert := SecurityAlert{
			ID:          fmt.Sprintf("failed-attempts-%d", time.Now().Unix()),
			Timestamp:   time.Now(),
			Severity:    AlertSeverityHigh,
			AlertType:   "excessive_failed_attempts",
			Title:       "Excessive Failed Authentication Attempts",
			Description: fmt.Sprintf("Detected %d failed attempts in the last hour", recentFailed),
			EventCount:  recentFailed,
			Details: map[string]interface{}{
				"threshold": m.alertThresholds.FailedAttemptsPerHour,
				"actual":    recentFailed,
			},
			Actions: []string{
				"Review device registration logs",
				"Check for unauthorized access attempts",
				"Consider increasing security measures",
			},
		}
		alerts = append(alerts, alert)
	}

	// Check for tamper attempts
	if m.metrics.TamperAttempts > 0 {
		alert := SecurityAlert{
			ID:          fmt.Sprintf("tamper-detected-%d", time.Now().Unix()),
			Timestamp:   time.Now(),
			Severity:    AlertSeverityCritical,
			AlertType:   "tamper_detected",
			Title:       "File Tampering Detected",
			Description: fmt.Sprintf("Detected %d tamper attempts", m.metrics.TamperAttempts),
			EventCount:  m.metrics.TamperAttempts,
			Details: map[string]interface{}{
				"tamper_count": m.metrics.TamperAttempts,
				"action_taken": "access_blocked",
			},
			Actions: []string{
				"Investigate tampered files immediately",
				"Run full system integrity check",
				"Consider reinstalling affected components",
				"Check for malware or unauthorized modifications",
			},
		}
		alerts = append(alerts, alert)
	}

	// Check for unusual device activity
	suspiciousDevices := m.identifySuspiciousDevices(events)
	for _, deviceID := range suspiciousDevices {
		alert := SecurityAlert{
			ID:          fmt.Sprintf("suspicious-device-%s-%d", deviceID, time.Now().Unix()),
			Timestamp:   time.Now(),
			Severity:    AlertSeverityMedium,
			AlertType:   "suspicious_device_activity",
			Title:       "Suspicious Device Activity",
			Description: fmt.Sprintf("Device %s showing unusual activity patterns", deviceID),
			DeviceID:    deviceID,
			Details: map[string]interface{}{
				"device_id": deviceID,
				"activity_count": m.metrics.DeviceActivity[deviceID],
			},
			Actions: []string{
				"Review device access logs",
				"Verify device identity and binding",
				"Consider revoking device access if unauthorized",
			},
		}
		alerts = append(alerts, alert)
	}

	return alerts
}

// checkSystemHealth performs comprehensive system health checks
func (m *SecurityMonitor) checkSystemHealth() (*SystemHealthStatus, error) {
	health := &SystemHealthStatus{
		LastHealthCheck: time.Now(),
	}

	// Check keychain status
	if err := ValidateKeychainProvider(); err != nil {
		health.KeychainStatus = fmt.Sprintf("ERROR: %v", err)
	} else {
		health.KeychainStatus = "OK"
	}

	// Check encryption status
	health.EncryptionStatus = "OK" // Assume OK if no errors detected

	// Check file integrity
	if m.metrics.TamperAttempts > 0 {
		health.FileIntegrity = "COMPROMISED"
	} else {
		health.FileIntegrity = "OK"
	}

	// Check device binding
	failedBindings := m.countFailedDeviceBindings()
	if failedBindings > m.alertThresholds.MaxFailedDeviceBinding {
		health.DeviceBinding = "WARNING"
	} else {
		health.DeviceBinding = "OK"
	}

	// Check audit logging
	if _, err := os.Stat(m.auditLogger.GetAuditLogPath()); err != nil {
		health.AuditLogging = "ERROR"
	} else {
		health.AuditLogging = "OK"
	}

	return health, nil
}

// Helper methods

func (m *SecurityMonitor) countRecentFailedAttempts(events []SecurityEvent) int {
	cutoff := time.Now().Add(-m.alertThresholds.AlertWindow)
	count := 0

	for _, event := range events {
		if !event.Success && event.Timestamp.After(cutoff) {
			count++
		}
	}

	return count
}

func (m *SecurityMonitor) identifySuspiciousDevices(events []SecurityEvent) []string {
	suspicious := make([]string, 0)

	for deviceID, count := range m.metrics.DeviceActivity {
		// Device with unusually high activity (threshold: 50 events)
		if count > 50 {
			suspicious = append(suspicious, deviceID)
		}
	}

	return suspicious
}

func (m *SecurityMonitor) countFailedDeviceBindings() int {
	count := 0
	for _, event := range m.metrics.RecentCriticalEvents {
		if event.EventType == "access_attempt" && !event.Success {
			if reason, ok := event.Details["denial_reason"].(string); ok {
				if strings.Contains(reason, "device_binding") {
					count++
				}
			}
		}
	}
	return count
}

func (m *SecurityMonitor) calculateSecurityScore() int {
	score := 100

	// Deduct points for security issues
	score -= m.metrics.TamperAttempts * 20      // -20 per tamper attempt
	score -= (m.metrics.FailedAttempts / 10) * 5 // -5 per 10 failed attempts
	
	// Bonus points for successful operations
	if m.metrics.SuccessfulOperations > m.metrics.FailedAttempts {
		score += 5
	}

	// Ensure score is within bounds
	if score < 0 {
		score = 0
	}
	if score > 100 {
		score = 100
	}

	return score
}

func (m *SecurityMonitor) calculateThreatLevel() string {
	if m.metrics.TamperAttempts > 0 {
		return "CRITICAL"
	}
	if m.metrics.FailedAttempts > m.metrics.SuccessfulOperations {
		return "HIGH"
	}
	if m.metrics.FailedAttempts > 10 {
		return "MEDIUM"
	}
	return "LOW"
}

func (m *SecurityMonitor) calculateOverallStatus() (string, string) {
	score := m.metrics.SecurityScore
	threatLevel := m.metrics.ThreatLevel

	switch {
	case score >= 90 && threatLevel == "LOW":
		return "SECURE", threatLevel
	case score >= 70 && threatLevel != "CRITICAL":
		return "MONITORING", threatLevel
	case score >= 50:
		return "CAUTION", threatLevel
	default:
		return "ALERT", threatLevel
	}
}

func (m *SecurityMonitor) generateRecommendations() []string {
	recommendations := make([]string, 0)

	if m.metrics.SecurityScore < 70 {
		recommendations = append(recommendations, "Review and strengthen security configurations")
	}

	if m.metrics.TamperAttempts > 0 {
		recommendations = append(recommendations, "Investigate file integrity issues immediately")
	}

	if m.metrics.FailedAttempts > 20 {
		recommendations = append(recommendations, "Review authentication logs for suspicious activity")
	}

	// Check keychain provider distribution
	if len(m.metrics.KeychainProviderStats) > 1 {
		// If multiple providers are in use, suggest standardization
		recommendations = append(recommendations, "Consider standardizing keychain provider usage")
	}

	if len(recommendations) == 0 {
		recommendations = append(recommendations, "Security posture is good - continue monitoring")
	}

	return recommendations
}

func (m *SecurityMonitor) getActiveAlerts() []SecurityAlert {
	// For now, return empty slice - in a full implementation,
	// this would track active alerts from recent time windows
	return make([]SecurityAlert, 0)
}

// ConsoleAlertHandler provides console-based alert handling
type ConsoleAlertHandler struct{}

func (h *ConsoleAlertHandler) SendAlert(alert SecurityAlert) error {
	fmt.Printf("\n🚨 SECURITY ALERT [%s] - %s\n", alert.Severity, alert.Title)
	fmt.Printf("📅 Time: %s\n", alert.Timestamp.Format(time.RFC3339))
	fmt.Printf("📝 Description: %s\n", alert.Description)
	
	if alert.DeviceID != "" {
		fmt.Printf("📱 Device: %s\n", alert.DeviceID)
	}

	if alert.EventCount > 0 {
		fmt.Printf("📊 Event Count: %d\n", alert.EventCount)
	}

	if len(alert.Actions) > 0 {
		fmt.Printf("🔧 Recommended Actions:\n")
		for _, action := range alert.Actions {
			fmt.Printf("   • %s\n", action)
		}
	}
	
	fmt.Println("─────────────────────────────────────")
	return nil
}