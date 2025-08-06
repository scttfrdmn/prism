// Package security provides tests for security audit logging
package security

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"
)

// TestNewSecurityAuditLogger validates audit logger creation
func TestNewSecurityAuditLogger(t *testing.T) {
	logger, err := NewSecurityAuditLogger()
	if err != nil {
		t.Fatalf("Failed to create security audit logger: %v", err)
	}
	defer logger.Close()

	if logger.logPath == "" {
		t.Error("Log path should not be empty")
	}

	// Verify log file exists
	if _, err := os.Stat(logger.logPath); os.IsNotExist(err) {
		t.Error("Log file should exist after creation")
	}

	t.Logf("✅ Audit logger created successfully at: %s", logger.logPath)
}

// TestSecurityEventLogging validates security event logging
func TestSecurityEventLogging(t *testing.T) {
	logger, err := NewSecurityAuditLogger()
	if err != nil {
		t.Fatalf("Failed to create security audit logger: %v", err)
	}
	defer logger.Close()

	// Test basic security event
	event := SecurityEvent{
		EventType: "test_event",
		DeviceID:  "test-device-123",
		Success:   true,
		Details: map[string]interface{}{
			"test_key": "test_value",
		},
	}

	logger.LogSecurityEvent(event)

	// Force flush
	logger.mutex.Lock()
	logger.flushBuffer()
	logger.mutex.Unlock()

	// Read log file and verify event was written
	content, err := os.ReadFile(logger.logPath)
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	if !strings.Contains(string(content), "test_event") {
		t.Error("Log should contain test event")
	}

	if !strings.Contains(string(content), "test-device-123") {
		t.Error("Log should contain device ID")
	}

	t.Log("✅ Security event logging validated")
}

// TestDeviceRegistrationLogging validates device registration logging
func TestDeviceRegistrationLogging(t *testing.T) {
	logger, err := NewSecurityAuditLogger()
	if err != nil {
		t.Fatalf("Failed to create security audit logger: %v", err)
	}
	defer logger.Close()

	// Test successful registration
	logger.LogDeviceRegistration("test-device-456", "invitation-token-123", true, "", map[string]interface{}{
		"hostname": "test-host",
		"username": "test-user",
	})

	// Test failed registration
	logger.LogDeviceRegistration("test-device-789", "invitation-token-456", false, "invalid_token", map[string]interface{}{
		"failure_reason": "token expired",
	})

	// Force flush
	logger.mutex.Lock()
	logger.flushBuffer()
	logger.mutex.Unlock()

	// Read and verify log content
	content, err := os.ReadFile(logger.logPath)
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	logContent := string(content)

	// Verify successful registration
	if !strings.Contains(logContent, "device_registration") {
		t.Error("Log should contain device_registration events")
	}

	if !strings.Contains(logContent, "test-device-456") {
		t.Error("Log should contain successful device ID")
	}

	// Verify failed registration
	if !strings.Contains(logContent, "test-device-789") {
		t.Error("Log should contain failed device ID")
	}

	if !strings.Contains(logContent, "invalid_token") {
		t.Error("Log should contain error code")
	}

	// Verify token masking
	if strings.Contains(logContent, "invitation-token-123") {
		t.Error("Log should not contain full invitation token")
	}

	t.Log("✅ Device registration logging validated")
}

// TestAccessAttemptLogging validates access attempt logging
func TestAccessAttemptLogging(t *testing.T) {
	logger, err := NewSecurityAuditLogger()
	if err != nil {
		t.Fatalf("Failed to create security audit logger: %v", err)
	}
	defer logger.Close()

	// Test successful access
	logger.LogAccessAttempt("device-success", true, "", map[string]interface{}{
		"access_type": "device_validation",
	})

	// Test failed access
	logger.LogAccessAttempt("device-failed", false, "device_binding_violation", map[string]interface{}{
		"expected_fingerprint": "abc123",
		"actual_fingerprint":   "def456",
	})

	// Force flush
	logger.mutex.Lock()
	logger.flushBuffer()
	logger.mutex.Unlock()

	// Verify log content
	content, err := os.ReadFile(logger.logPath)
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	logContent := string(content)

	if !strings.Contains(logContent, "access_attempt") {
		t.Error("Log should contain access_attempt events")
	}

	if !strings.Contains(logContent, "device_binding_violation") {
		t.Error("Log should contain denial reason")
	}

	t.Log("✅ Access attempt logging validated")
}

// TestTamperAttemptLogging validates tamper detection logging
func TestTamperAttemptLogging(t *testing.T) {
	logger, err := NewSecurityAuditLogger()
	if err != nil {
		t.Fatalf("Failed to create security audit logger: %v", err)
	}
	defer logger.Close()

	logger.LogTamperAttempt("device-tamper", "/test/file.json", "expected-hash-123", "actual-hash-456")

	// Force flush
	logger.mutex.Lock()
	logger.flushBuffer()
	logger.mutex.Unlock()

	// Verify log content
	content, err := os.ReadFile(logger.logPath)
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	logContent := string(content)

	if !strings.Contains(logContent, "tamper_detected") {
		t.Error("Log should contain tamper_detected event")
	}

	if !strings.Contains(logContent, "CRITICAL") {
		t.Error("Tamper events should be marked as CRITICAL")
	}

	if !strings.Contains(logContent, "expected-hash-123") {
		t.Error("Log should contain expected hash")
	}

	if !strings.Contains(logContent, "actual-hash-456") {
		t.Error("Log should contain actual hash")
	}

	t.Log("✅ Tamper attempt logging validated")
}

// TestKeychainOperationLogging validates keychain operation logging
func TestKeychainOperationLogging(t *testing.T) {
	logger, err := NewSecurityAuditLogger()
	if err != nil {
		t.Fatalf("Failed to create security audit logger: %v", err)
	}
	defer logger.Close()

	// Test successful keychain operation
	logger.LogKeychainOperation("store", "test.key.name", true, "macOS Keychain", "")

	// Test failed keychain operation
	logger.LogKeychainOperation("retrieve", "missing.key", false, "File Storage", "key_not_found")

	// Force flush
	logger.mutex.Lock()
	logger.flushBuffer()
	logger.mutex.Unlock()

	// Verify log content
	content, err := os.ReadFile(logger.logPath)
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	logContent := string(content)

	if !strings.Contains(logContent, "keychain_operation") {
		t.Error("Log should contain keychain_operation events")
	}

	if !strings.Contains(logContent, "macOS Keychain") {
		t.Error("Log should contain provider information")
	}

	// Verify key masking
	if strings.Contains(logContent, "test.key.name") {
		t.Error("Log should mask full key names")
	}

	t.Log("✅ Keychain operation logging validated")
}

// TestRegistryOperationLogging validates registry operation logging
func TestRegistryOperationLogging(t *testing.T) {
	logger, err := NewSecurityAuditLogger()
	if err != nil {
		t.Fatalf("Failed to create security audit logger: %v", err)
	}
	defer logger.Close()

	// Test successful registry operation
	logger.LogRegistryOperation("register", "/api/v1/register", true, 200, "")

	// Test failed registry operation
	logger.LogRegistryOperation("validate", "/api/v1/validate", false, 403, "signature_invalid")

	// Force flush
	logger.mutex.Lock()
	logger.flushBuffer()
	logger.mutex.Unlock()

	// Verify log content
	content, err := os.ReadFile(logger.logPath)
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	logContent := string(content)

	if !strings.Contains(logContent, "registry_operation") {
		t.Error("Log should contain registry_operation events")
	}

	if !strings.Contains(logContent, "signature_invalid") {
		t.Error("Log should contain error code")
	}

	if !strings.Contains(logContent, "200") {
		t.Error("Log should contain status codes")
	}

	t.Log("✅ Registry operation logging validated")
}

// TestLogRotation validates log rotation functionality
func TestLogRotation(t *testing.T) {
	logger, err := NewSecurityAuditLogger()
	if err != nil {
		t.Fatalf("Failed to create security audit logger: %v", err)
	}
	defer logger.Close()

	originalPath := logger.logPath

	// Log an event
	logger.LogSecurityEvent(SecurityEvent{
		EventType: "pre_rotation",
		Success:   true,
	})

	// Force flush
	logger.mutex.Lock()
	logger.flushBuffer()
	logger.mutex.Unlock()

	// Rotate log
	if err := logger.RotateLog(); err != nil {
		t.Fatalf("Failed to rotate log: %v", err)
	}

	// Verify new log file
	if logger.logPath == originalPath {
		t.Error("Log path should change after rotation")
	}

	// Log another event
	logger.LogSecurityEvent(SecurityEvent{
		EventType: "post_rotation",
		Success:   true,
	})

	// Force flush
	logger.mutex.Lock()
	logger.flushBuffer()
	logger.mutex.Unlock()

	// Verify both files exist and contain appropriate events
	originalContent, err := os.ReadFile(originalPath)
	if err != nil {
		t.Fatalf("Failed to read original log file: %v", err)
	}

	newContent, err := os.ReadFile(logger.logPath)
	if err != nil {
		t.Fatalf("Failed to read new log file: %v", err)
	}

	if !strings.Contains(string(originalContent), "pre_rotation") {
		t.Error("Original log should contain pre-rotation event")
	}

	if !strings.Contains(string(newContent), "post_rotation") {
		t.Error("New log should contain post-rotation event")
	}

	t.Log("✅ Log rotation validated")
}

// TestAuditLogFormat validates JSON log format
func TestAuditLogFormat(t *testing.T) {
	logger, err := NewSecurityAuditLogger()
	if err != nil {
		t.Fatalf("Failed to create security audit logger: %v", err)
	}
	defer logger.Close()

	// Log a comprehensive event
	event := SecurityEvent{
		EventType: "format_test",
		DeviceID:  "format-device-123",
		Success:   true,
		Severity:  "INFO",
		Details: map[string]interface{}{
			"test_number": 42,
			"test_bool":   true,
			"test_array":  []string{"item1", "item2"},
		},
	}

	logger.LogSecurityEvent(event)

	// Force flush
	logger.mutex.Lock()
	logger.flushBuffer()
	logger.mutex.Unlock()

	// Read and parse JSON
	content, err := os.ReadFile(logger.logPath)
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(string(content)), "\n")
	if len(lines) == 0 {
		t.Fatal("Log file should contain at least one line")
	}

	// Parse the last line as JSON
	var loggedEvent SecurityEvent
	if err := json.Unmarshal([]byte(lines[len(lines)-1]), &loggedEvent); err != nil {
		t.Fatalf("Failed to parse logged event as JSON: %v", err)
	}

	// Validate structure
	if loggedEvent.EventType != "format_test" {
		t.Errorf("Expected event_type 'format_test', got '%s'", loggedEvent.EventType)
	}

	if loggedEvent.DeviceID != "format-device-123" {
		t.Errorf("Expected device_id 'format-device-123', got '%s'", loggedEvent.DeviceID)
	}

	if !loggedEvent.Success {
		t.Error("Expected success to be true")
	}

	if loggedEvent.Timestamp.IsZero() {
		t.Error("Timestamp should not be zero")
	}

	t.Log("✅ Audit log JSON format validated")
}

// TestConcurrentLogging validates thread-safe logging
func TestConcurrentLogging(t *testing.T) {
	logger, err := NewSecurityAuditLogger()
	if err != nil {
		t.Fatalf("Failed to create security audit logger: %v", err)
	}
	defer logger.Close()

	// Launch multiple goroutines to log events concurrently
	done := make(chan bool, 10)

	for i := 0; i < 10; i++ {
		go func(id int) {
			for j := 0; j < 5; j++ {
				logger.LogSecurityEvent(SecurityEvent{
					EventType: "concurrent_test",
					DeviceID:  fmt.Sprintf("device-%d", id),
					Success:   true,
					Details: map[string]interface{}{
						"goroutine": id,
						"iteration": j,
					},
				})
				time.Sleep(10 * time.Millisecond)
			}
			done <- true
		}(i)
	}

	// Wait for all goroutines to complete
	for i := 0; i < 10; i++ {
		<-done
	}

	// Force flush
	logger.mutex.Lock()
	logger.flushBuffer()
	logger.mutex.Unlock()

	// Verify log contains events from all goroutines
	content, err := os.ReadFile(logger.logPath)
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	logContent := string(content)
	eventCount := strings.Count(logContent, "concurrent_test")

	if eventCount != 50 { // 10 goroutines * 5 events each
		t.Errorf("Expected 50 events, found %d", eventCount)
	}

	t.Log("✅ Concurrent logging validated")
}