// Package security provides tests for security monitoring and alerting
package security

import (
	"testing"
	"time"
)

// TestNewSecurityMonitor validates security monitor creation
func TestNewSecurityMonitor(t *testing.T) {
	monitor, err := NewSecurityMonitor()
	if err != nil {
		t.Fatalf("Failed to create security monitor: %v", err)
	}
	defer monitor.auditLogger.Close()

	if monitor.auditLogger == nil {
		t.Error("Audit logger should not be nil")
	}

	if monitor.metrics == nil {
		t.Error("Metrics should not be nil")
	}

	if monitor.alertHandler == nil {
		t.Error("Alert handler should not be nil")
	}

	// Test default thresholds
	if monitor.alertThresholds.FailedAttemptsPerHour != 10 {
		t.Errorf("Expected failed attempts threshold 10, got %d", monitor.alertThresholds.FailedAttemptsPerHour)
	}

	if monitor.alertThresholds.TamperAttemptsThreshold != 1 {
		t.Errorf("Expected tamper attempts threshold 1, got %d", monitor.alertThresholds.TamperAttemptsThreshold)
	}

	t.Log("✅ Security monitor created successfully")
}

// TestSecurityMetricsUpdate validates metrics updating
func TestSecurityMetricsUpdate(t *testing.T) {
	// Setup test monitor
	monitor := setupSecurityMonitor(t)
	defer monitor.auditLogger.Close()

	// Create and process test events
	events := createTestSecurityEvents()
	monitor.updateMetrics(events)

	// Validate all metrics categories
	validateBasicMetrics(t, monitor)
	validateEventTypeBreakdown(t, monitor)
	validateDeviceActivity(t, monitor)
	validateKeychainProviderStats(t, monitor)
	validateCriticalEventTracking(t, monitor)
	validateSecurityScoring(t, monitor)

	t.Log("✅ Security metrics update validated")
}

// setupSecurityMonitor creates and configures a security monitor for testing
func setupSecurityMonitor(t *testing.T) *SecurityMonitor {
	monitor, err := NewSecurityMonitor()
	if err != nil {
		t.Fatalf("Failed to create security monitor: %v", err)
	}
	return monitor
}

// createTestSecurityEvents generates a comprehensive set of test events
func createTestSecurityEvents() []SecurityEvent {
	return []SecurityEvent{
		{
			EventType: "device_registration",
			Success:   true,
			DeviceID:  "device1",
			Timestamp: time.Now(),
		},
		{
			EventType: "access_attempt",
			Success:   false,
			DeviceID:  "device2",
			Timestamp: time.Now(),
			Severity:  "WARNING",
		},
		{
			EventType: "tamper_detected",
			Success:   false,
			DeviceID:  "device1",
			Timestamp: time.Now(),
			Severity:  "CRITICAL",
		},
		{
			EventType: "keychain_operation",
			Success:   true,
			Timestamp: time.Now(),
			Details: map[string]interface{}{
				"provider": "macOS Keychain",
			},
		},
	}
}

// validateBasicMetrics checks fundamental metric counters
func validateBasicMetrics(t *testing.T, monitor *SecurityMonitor) {
	if monitor.metrics.TotalEvents != 4 {
		t.Errorf("Expected 4 total events, got %d", monitor.metrics.TotalEvents)
	}

	if monitor.metrics.SuccessfulOperations != 2 {
		t.Errorf("Expected 2 successful operations, got %d", monitor.metrics.SuccessfulOperations)
	}

	if monitor.metrics.FailedAttempts != 2 {
		t.Errorf("Expected 2 failed attempts, got %d", monitor.metrics.FailedAttempts)
	}

	if monitor.metrics.TamperAttempts != 1 {
		t.Errorf("Expected 1 tamper attempt, got %d", monitor.metrics.TamperAttempts)
	}

	if monitor.metrics.DeviceRegistrations != 1 {
		t.Errorf("Expected 1 device registration, got %d", monitor.metrics.DeviceRegistrations)
	}
}

// validateEventTypeBreakdown checks event type categorization
func validateEventTypeBreakdown(t *testing.T, monitor *SecurityMonitor) {
	if monitor.metrics.EventTypeBreakdown["device_registration"] != 1 {
		t.Error("Event type breakdown should track device_registration")
	}

	if monitor.metrics.EventTypeBreakdown["tamper_detected"] != 1 {
		t.Error("Event type breakdown should track tamper_detected")
	}
}

// validateDeviceActivity checks per-device activity tracking
func validateDeviceActivity(t *testing.T, monitor *SecurityMonitor) {
	if monitor.metrics.DeviceActivity["device1"] != 2 {
		t.Error("Device activity should track device1 activity")
	}

	if monitor.metrics.DeviceActivity["device2"] != 1 {
		t.Error("Device activity should track device2 activity")
	}
}

// validateKeychainProviderStats checks keychain provider statistics
func validateKeychainProviderStats(t *testing.T, monitor *SecurityMonitor) {
	if monitor.metrics.KeychainProviderStats["macOS Keychain"] != 1 {
		t.Error("Keychain provider stats should track macOS Keychain usage")
	}
}

// validateCriticalEventTracking checks critical event monitoring
func validateCriticalEventTracking(t *testing.T, monitor *SecurityMonitor) {
	if len(monitor.metrics.RecentCriticalEvents) != 1 {
		t.Error("Should track recent critical events")
	}
}

// validateSecurityScoring checks security score and threat level calculation
func validateSecurityScoring(t *testing.T, monitor *SecurityMonitor) {
	if monitor.metrics.SecurityScore < 0 || monitor.metrics.SecurityScore > 100 {
		t.Errorf("Security score should be 0-100, got %d", monitor.metrics.SecurityScore)
	}

	if monitor.metrics.ThreatLevel == "" {
		t.Error("Threat level should not be empty")
	}
}

// TestSecurityThreatAnalysis validates threat analysis and alerting
func TestSecurityThreatAnalysis(t *testing.T) {
	monitor, err := NewSecurityMonitor()
	if err != nil {
		t.Fatalf("Failed to create security monitor: %v", err)
	}
	defer monitor.auditLogger.Close()

	// Create events that should trigger alerts
	events := make([]SecurityEvent, 0)

	// Add multiple failed attempts (should trigger excessive failed attempts alert)
	for i := 0; i < 15; i++ {
		events = append(events, SecurityEvent{
			EventType: "access_attempt",
			Success:   false,
			DeviceID:  "suspicious-device",
			Timestamp: time.Now(),
		})
	}

	// Add tamper attempt (should trigger critical alert)
	events = append(events, SecurityEvent{
		EventType: "tamper_detected",
		Success:   false,
		DeviceID:  "compromised-device",
		Timestamp: time.Now(),
		Severity:  "CRITICAL",
	})

	// Update metrics and analyze threats
	monitor.updateMetrics(events)
	alerts := monitor.analyzeSecurityThreats(events)

	// Should have at least 2 alerts (failed attempts + tamper)
	if len(alerts) < 2 {
		t.Errorf("Expected at least 2 alerts, got %d", len(alerts))
	}

	// Check for excessive failed attempts alert
	foundFailedAttemptsAlert := false
	for _, alert := range alerts {
		if alert.AlertType == "excessive_failed_attempts" {
			foundFailedAttemptsAlert = true
			if alert.Severity != AlertSeverityHigh {
				t.Error("Failed attempts alert should be HIGH severity")
			}
			if alert.EventCount < 15 {
				t.Error("Failed attempts alert should report correct event count")
			}
		}
	}
	if !foundFailedAttemptsAlert {
		t.Error("Should generate alert for excessive failed attempts")
	}

	// Check for tamper alert
	foundTamperAlert := false
	for _, alert := range alerts {
		if alert.AlertType == "tamper_detected" {
			foundTamperAlert = true
			if alert.Severity != AlertSeverityCritical {
				t.Error("Tamper alert should be CRITICAL severity")
			}
		}
	}
	if !foundTamperAlert {
		t.Error("Should generate alert for tamper attempts")
	}

	t.Log("✅ Security threat analysis validated")
}

// TestSystemHealthCheck validates system health monitoring
func TestSystemHealthCheck(t *testing.T) {
	monitor, err := NewSecurityMonitor()
	if err != nil {
		t.Fatalf("Failed to create security monitor: %v", err)
	}
	defer monitor.auditLogger.Close()

	health, err := monitor.checkSystemHealth()
	if err != nil {
		t.Fatalf("Failed to check system health: %v", err)
	}

	// Validate health check fields
	if health.KeychainStatus == "" {
		t.Error("Keychain status should not be empty")
	}

	if health.EncryptionStatus == "" {
		t.Error("Encryption status should not be empty")
	}

	if health.FileIntegrity == "" {
		t.Error("File integrity status should not be empty")
	}

	if health.DeviceBinding == "" {
		t.Error("Device binding status should not be empty")
	}

	if health.AuditLogging == "" {
		t.Error("Audit logging status should not be empty")
	}

	if health.LastHealthCheck.IsZero() {
		t.Error("Last health check timestamp should not be zero")
	}

	t.Logf("Health Status:")
	t.Logf("  Keychain: %s", health.KeychainStatus)
	t.Logf("  Encryption: %s", health.EncryptionStatus)
	t.Logf("  File Integrity: %s", health.FileIntegrity)
	t.Logf("  Device Binding: %s", health.DeviceBinding)
	t.Logf("  Audit Logging: %s", health.AuditLogging)

	t.Log("✅ System health check validated")
}

// TestSecurityDashboard validates security dashboard generation
func TestSecurityDashboard(t *testing.T) {
	monitor, err := NewSecurityMonitor()
	if err != nil {
		t.Fatalf("Failed to create security monitor: %v", err)
	}
	defer monitor.auditLogger.Close()

	// Generate some test events first
	testEvents := []SecurityEvent{
		{
			EventType: "device_registration",
			Success:   true,
			DeviceID:  "test-device",
			Timestamp: time.Now(),
		},
		{
			EventType: "access_attempt",
			Success:   true,
			DeviceID:  "test-device",
			Timestamp: time.Now(),
		},
	}

	monitor.updateMetrics(testEvents)

	dashboard, err := monitor.GetSecurityDashboard()
	if err != nil {
		t.Fatalf("Failed to get security dashboard: %v", err)
	}

	// Validate dashboard fields
	if dashboard.Status == "" {
		t.Error("Dashboard status should not be empty")
	}

	if dashboard.ThreatLevel == "" {
		t.Error("Dashboard threat level should not be empty")
	}

	if dashboard.SecurityScore < 0 || dashboard.SecurityScore > 100 {
		t.Errorf("Security score should be 0-100, got %d", dashboard.SecurityScore)
	}

	if dashboard.Metrics == nil {
		t.Error("Dashboard metrics should not be nil")
	}

	if dashboard.Recommendations == nil {
		t.Error("Dashboard recommendations should not be nil")
	}

	if dashboard.LastUpdate.IsZero() {
		t.Error("Dashboard last update should not be zero")
	}

	// Validate system health in dashboard
	if dashboard.SystemHealth.KeychainStatus == "" {
		t.Error("System health keychain status should not be empty")
	}

	t.Logf("Security Dashboard:")
	t.Logf("  Status: %s", dashboard.Status)
	t.Logf("  Threat Level: %s", dashboard.ThreatLevel)
	t.Logf("  Security Score: %d", dashboard.SecurityScore)
	t.Logf("  Active Alerts: %d", len(dashboard.ActiveAlerts))
	t.Logf("  Recommendations: %d", len(dashboard.Recommendations))

	t.Log("✅ Security dashboard validated")
}

// TestAlertSeverityLevels validates alert severity classification
func TestAlertSeverityLevels(t *testing.T) {
	monitor, err := NewSecurityMonitor()
	if err != nil {
		t.Fatalf("Failed to create security monitor: %v", err)
	}
	defer monitor.auditLogger.Close()

	// Test different types of events and their alert severity
	testCases := []struct {
		events         []SecurityEvent
		expectedAlerts int
		expectedMaxSev AlertSeverity
		description    string
	}{
		{
			events: []SecurityEvent{
				{EventType: "tamper_detected", Success: false, Timestamp: time.Now(), Severity: "CRITICAL"},
			},
			expectedAlerts: 1,
			expectedMaxSev: AlertSeverityCritical,
			description:    "Tamper detection should generate critical alert",
		},
		{
			events: func() []SecurityEvent {
				events := make([]SecurityEvent, 12)
				for i := range events {
					events[i] = SecurityEvent{
						EventType: "access_attempt",
						Success:   false,
						Timestamp: time.Now(),
					}
				}
				return events
			}(),
			expectedAlerts: 1,
			expectedMaxSev: AlertSeverityHigh,
			description:    "Multiple failed attempts should generate high severity alert",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			monitor.updateMetrics(tc.events)
			alerts := monitor.analyzeSecurityThreats(tc.events)

			if len(alerts) != tc.expectedAlerts {
				t.Errorf("Expected %d alerts, got %d", tc.expectedAlerts, len(alerts))
			}

			if len(alerts) > 0 {
				maxSeverity := AlertSeverityLow
				for _, alert := range alerts {
					if alert.Severity == AlertSeverityCritical {
						maxSeverity = AlertSeverityCritical
					} else if alert.Severity == AlertSeverityHigh && maxSeverity != AlertSeverityCritical {
						maxSeverity = AlertSeverityHigh
					} else if alert.Severity == AlertSeverityMedium && maxSeverity == AlertSeverityLow {
						maxSeverity = AlertSeverityMedium
					}
				}

				if maxSeverity != tc.expectedMaxSev {
					t.Errorf("Expected max severity %s, got %s", tc.expectedMaxSev, maxSeverity)
				}
			}
		})
	}

	t.Log("✅ Alert severity levels validated")
}

// TestConsoleAlertHandler validates console alert output
func TestConsoleAlertHandler(t *testing.T) {
	handler := &ConsoleAlertHandler{}

	alert := SecurityAlert{
		ID:          "test-alert-123",
		Timestamp:   time.Now(),
		Severity:    AlertSeverityHigh,
		AlertType:   "test_alert",
		Title:       "Test Security Alert",
		Description: "This is a test alert for validation",
		DeviceID:    "test-device-456",
		EventCount:  5,
		Actions: []string{
			"Review logs",
			"Check device status",
		},
	}

	// Test sending alert (should not error)
	if err := handler.SendAlert(alert); err != nil {
		t.Errorf("Console alert handler should not error: %v", err)
	}

	t.Log("✅ Console alert handler validated")
}

// TestSecurityScoreCalculation validates security score calculation
func TestSecurityScoreCalculation(t *testing.T) {
	monitor, err := NewSecurityMonitor()
	if err != nil {
		t.Fatalf("Failed to create security monitor: %v", err)
	}
	defer monitor.auditLogger.Close()

	testCases := []struct {
		tamperAttempts       int
		failedAttempts       int
		successfulOperations int
		expectedRange        [2]int // min, max expected score
		description          string
	}{
		{
			tamperAttempts:       0,
			failedAttempts:       0,
			successfulOperations: 10,
			expectedRange:        [2]int{95, 100},
			description:          "Perfect security should have high score",
		},
		{
			tamperAttempts:       1,
			failedAttempts:       5,
			successfulOperations: 15,
			expectedRange:        [2]int{70, 85},
			description:          "Some issues should reduce score moderately",
		},
		{
			tamperAttempts:       3,
			failedAttempts:       50,
			successfulOperations: 10,
			expectedRange:        [2]int{0, 40},
			description:          "Major issues should significantly reduce score",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			monitor.metrics.TamperAttempts = tc.tamperAttempts
			monitor.metrics.FailedAttempts = tc.failedAttempts
			monitor.metrics.SuccessfulOperations = tc.successfulOperations

			score := monitor.calculateSecurityScore()

			if score < tc.expectedRange[0] || score > tc.expectedRange[1] {
				t.Errorf("Security score %d not in expected range [%d, %d]",
					score, tc.expectedRange[0], tc.expectedRange[1])
			}

			if score < 0 || score > 100 {
				t.Errorf("Security score %d out of valid range [0, 100]", score)
			}

			t.Logf("Score: %d (tamper: %d, failed: %d, success: %d)",
				score, tc.tamperAttempts, tc.failedAttempts, tc.successfulOperations)
		})
	}

	t.Log("✅ Security score calculation validated")
}
