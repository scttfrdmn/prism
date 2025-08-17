// Package security provides comprehensive audit logging for security events
package security

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// SecurityAuditLogger provides comprehensive audit logging for security events
type SecurityAuditLogger struct {
	logFile    *os.File
	logger     *log.Logger
	logPath    string
	mutex      sync.Mutex
	buffer     []SecurityEvent
	maxBuffer  int
	flushTimer *time.Timer
}

// SecurityEvent represents a security-related event to be logged
type SecurityEvent struct {
	Timestamp time.Time              `json:"timestamp"`
	EventType string                 `json:"event_type"`
	DeviceID  string                 `json:"device_id,omitempty"`
	Success   bool                   `json:"success"`
	ErrorCode string                 `json:"error_code,omitempty"`
	Details   map[string]interface{} `json:"details,omitempty"`
	Severity  string                 `json:"severity"`
	Source    string                 `json:"source"`
	UserAgent string                 `json:"user_agent,omitempty"`
	RemoteIP  string                 `json:"remote_ip,omitempty"`
}

// NewSecurityAuditLogger creates a new security audit logger
func NewSecurityAuditLogger() (*SecurityAuditLogger, error) {
	// Create audit log directory
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}

	auditDir := filepath.Join(homeDir, ".cloudworkstation", "security", "audit")
	if err := os.MkdirAll(auditDir, 0700); err != nil {
		return nil, fmt.Errorf("failed to create audit directory: %w", err)
	}

	// Create log file with date rotation
	logFileName := fmt.Sprintf("security-audit-%s.log", time.Now().Format("2006-01-02"))
	logPath := filepath.Join(auditDir, logFileName)

	logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0600)
	if err != nil {
		return nil, fmt.Errorf("failed to open audit log file: %w", err)
	}

	logger := log.New(logFile, "", 0) // We'll format our own timestamps

	auditLogger := &SecurityAuditLogger{
		logFile:    logFile,
		logger:     logger,
		logPath:    logPath,
		buffer:     make([]SecurityEvent, 0, 100),
		maxBuffer:  100,
		flushTimer: nil,
	}

	// Start periodic flush
	auditLogger.startPeriodicFlush()

	return auditLogger, nil
}

// LogSecurityEvent logs a security event with comprehensive context
func (a *SecurityAuditLogger) LogSecurityEvent(event SecurityEvent) {
	a.mutex.Lock()
	defer a.mutex.Unlock()

	// Ensure timestamp is set
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now().UTC()
	}

	// Set default severity if not provided
	if event.Severity == "" {
		if !event.Success {
			event.Severity = "ERROR"
		} else {
			event.Severity = "INFO"
		}
	}

	// Set source if not provided
	if event.Source == "" {
		event.Source = "cloudworkstation-registry"
	}

	// Add to buffer
	a.buffer = append(a.buffer, event)

	// Flush if buffer is full or event is critical
	if len(a.buffer) >= a.maxBuffer || event.Severity == "CRITICAL" {
		a.flushBuffer()
	}
}

// LogDeviceRegistration logs device registration events
func (a *SecurityAuditLogger) LogDeviceRegistration(deviceID, invitationToken string, success bool, errorCode string, details map[string]interface{}) {
	event := SecurityEvent{
		EventType: "device_registration",
		DeviceID:  deviceID,
		Success:   success,
		ErrorCode: errorCode,
		Details:   details,
	}

	// Mask sensitive data
	if event.Details == nil {
		event.Details = make(map[string]interface{})
	}
	event.Details["invitation_token"] = maskToken(invitationToken)

	a.LogSecurityEvent(event)
}

// LogAccessAttempt logs access attempt events
func (a *SecurityAuditLogger) LogAccessAttempt(deviceID string, success bool, reason string, details map[string]interface{}) {
	event := SecurityEvent{
		EventType: "access_attempt",
		DeviceID:  deviceID,
		Success:   success,
		Details:   details,
	}

	if !success {
		event.ErrorCode = "access_denied"
		event.Severity = "WARNING"
		if event.Details == nil {
			event.Details = make(map[string]interface{})
		}
		event.Details["denial_reason"] = reason
	}

	a.LogSecurityEvent(event)
}

// LogTamperAttempt logs tamper detection events
func (a *SecurityAuditLogger) LogTamperAttempt(deviceID, filePath, expectedHash, actualHash string) {
	event := SecurityEvent{
		EventType: "tamper_detected",
		DeviceID:  deviceID,
		Success:   false,
		Severity:  "CRITICAL",
		ErrorCode: "file_tampered",
		Details: map[string]interface{}{
			"file_path":     filePath,
			"expected_hash": expectedHash,
			"actual_hash":   actualHash,
			"action":        "access_blocked",
		},
	}

	a.LogSecurityEvent(event)
}

// LogKeychainOperation logs keychain-related security events
func (a *SecurityAuditLogger) LogKeychainOperation(operation, key string, success bool, provider string, errorCode string) {
	event := SecurityEvent{
		EventType: "keychain_operation",
		Success:   success,
		ErrorCode: errorCode,
		Details: map[string]interface{}{
			"operation": operation,
			"key":       maskKey(key),
			"provider":  provider,
		},
	}

	if !success {
		event.Severity = "WARNING"
	}

	a.LogSecurityEvent(event)
}

// LogRegistryOperation logs registry communication events
func (a *SecurityAuditLogger) LogRegistryOperation(operation, endpoint string, success bool, statusCode int, errorCode string) {
	event := SecurityEvent{
		EventType: "registry_operation",
		Success:   success,
		ErrorCode: errorCode,
		Details: map[string]interface{}{
			"operation":   operation,
			"endpoint":    endpoint,
			"status_code": statusCode,
		},
	}

	if !success {
		event.Severity = "ERROR"
	}

	a.LogSecurityEvent(event)
}

// flushBuffer writes buffered events to the log file
func (a *SecurityAuditLogger) flushBuffer() {
	if len(a.buffer) == 0 {
		return
	}

	for _, event := range a.buffer {
		eventJSON, err := json.Marshal(event)
		if err != nil {
			// Fallback to simple text logging
			a.logger.Printf("ERROR: Failed to marshal security event: %v", err)
			continue
		}

		a.logger.Println(string(eventJSON))
	}

	// Clear buffer
	a.buffer = a.buffer[:0]

	// Sync to ensure data is written
	if err := a.logFile.Sync(); err != nil {
		log.Printf("Warning: Failed to sync audit log: %v", err)
	}
}

// startPeriodicFlush starts a timer to periodically flush the buffer
func (a *SecurityAuditLogger) startPeriodicFlush() {
	a.flushTimer = time.AfterFunc(30*time.Second, func() {
		a.mutex.Lock()
		a.flushBuffer()
		a.mutex.Unlock()
		a.startPeriodicFlush() // Restart timer
	})
}

// Close closes the audit logger and flushes any remaining events
func (a *SecurityAuditLogger) Close() error {
	a.mutex.Lock()
	defer a.mutex.Unlock()

	// Stop flush timer
	if a.flushTimer != nil {
		a.flushTimer.Stop()
	}

	// Flush remaining events
	a.flushBuffer()

	// Close log file
	if a.logFile != nil {
		return a.logFile.Close()
	}

	return nil
}

// GetAuditLogPath returns the path to the current audit log file
func (a *SecurityAuditLogger) GetAuditLogPath() string {
	return a.logPath
}

// RotateLog rotates the audit log to a new file
func (a *SecurityAuditLogger) RotateLog() error {
	a.mutex.Lock()
	defer a.mutex.Unlock()

	// Flush current buffer
	a.flushBuffer()

	// Close current log file
	if err := a.logFile.Close(); err != nil {
		return fmt.Errorf("failed to close current log file: %w", err)
	}

	// Create new log file
	logFileName := fmt.Sprintf("security-audit-%s.log", time.Now().Format("2006-01-02"))
	logPath := filepath.Join(filepath.Dir(a.logPath), logFileName)

	logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0600)
	if err != nil {
		return fmt.Errorf("failed to create new log file: %w", err)
	}

	// Update logger
	a.logFile = logFile
	a.logPath = logPath
	a.logger = log.New(logFile, "", 0)

	return nil
}

// Helper functions

func maskKey(key string) string {
	if len(key) <= 8 {
		return "***"
	}
	return key[:4] + "***" + key[len(key)-4:]
}
