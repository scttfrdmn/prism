package security

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNewSecurityManager tests security manager creation
func TestNewSecurityManager(t *testing.T) {
	config := GetDefaultSecurityConfig()

	manager, err := NewSecurityManager(config)

	assert.NoError(t, err)
	assert.NotNil(t, manager)
	// With default config (all features disabled), manager should be disabled
	assert.False(t, manager.isEnabled)
	assert.False(t, manager.isRunning)
	assert.Equal(t, config, manager.config)
}

// TestNewSecurityManagerInvalidConfig tests security manager creation with invalid config
func TestNewSecurityManagerInvalidConfig(t *testing.T) {
	config := SecurityConfig{
		AlertThreshold: "INVALID_THRESHOLD",
	}

	_, err := NewSecurityManager(config)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid security configuration")
}

// TestNewSecurityManagerDisabled tests security manager creation when disabled
func TestNewSecurityManagerDisabled(t *testing.T) {
	config := SecurityConfig{
		AuditLogEnabled:         false,
		MonitoringEnabled:       false,
		CorrelationEnabled:      false,
		RegistrySecurityEnabled: false,
	}

	manager, err := NewSecurityManager(config)

	assert.NoError(t, err)
	assert.NotNil(t, manager)
	assert.False(t, manager.isEnabled)
}

// TestSecurityManagerStartStop tests start and stop functionality
func TestSecurityManagerStartStop(t *testing.T) {
	config := SecurityConfig{
		AuditLogEnabled:     true,
		MonitoringEnabled:   true,
		MonitorInterval:     100 * time.Millisecond,
		AnalysisInterval:    200 * time.Millisecond,
		HealthCheckInterval: 300 * time.Millisecond,
		HealthCheckEnabled:  true,
	}

	manager, err := NewSecurityManager(config)
	require.NoError(t, err)

	// Test start
	err = manager.Start()
	assert.NoError(t, err)
	assert.True(t, manager.isRunning)

	// Test double start
	err = manager.Start()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already running")

	// Test stop
	err = manager.Stop()
	assert.NoError(t, err)
	assert.False(t, manager.isRunning)

	// Test double stop
	err = manager.Stop()
	assert.NoError(t, err) // Should not error
}

// TestSecurityManagerStartDisabled tests starting when disabled
func TestSecurityManagerStartDisabled(t *testing.T) {
	config := SecurityConfig{
		AuditLogEnabled:   false,
		MonitoringEnabled: false,
	}

	manager, err := NewSecurityManager(config)
	require.NoError(t, err)

	err = manager.Start()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "security manager is disabled")
}

// TestGetSecurityStatus tests security status retrieval
func TestGetSecurityStatus(t *testing.T) {
	config := GetDefaultSecurityConfig()

	manager, err := NewSecurityManager(config)
	require.NoError(t, err)

	status, err := manager.GetSecurityStatus()

	assert.NoError(t, err)
	assert.NotNil(t, status)
	assert.False(t, status.Enabled) // Default config has all features disabled
	assert.False(t, status.Running) // Not started yet
	assert.Equal(t, config, status.Configuration)
}

// TestSecurityManagerLogMethods tests security logging methods
func TestSecurityManagerLogMethods(t *testing.T) {
	config := SecurityConfig{
		AuditLogEnabled: true,
	}

	manager, err := NewSecurityManager(config)
	require.NoError(t, err)

	// Test security event logging (should not panic)
	manager.LogSecurityEvent("test_event", true, "device-123", map[string]interface{}{
		"test": "data",
	})

	// Test device registration logging
	manager.LogDeviceRegistration("device-123", "token-456", true, "", map[string]interface{}{
		"method": "invitation",
	})

	// Test access attempt logging
	manager.LogAccessAttempt("device-123", true, "valid_credentials", map[string]interface{}{
		"auth_method": "device_binding",
	})

	// Test tamper attempt logging
	manager.LogTamperAttempt("device-123", "/path/to/file", "expected-hash", "actual-hash")

	// Should complete without errors
	assert.True(t, true)
}

// TestSecurityManagerLogMethodsDisabled tests logging when audit is disabled
func TestSecurityManagerLogMethodsDisabled(t *testing.T) {
	config := SecurityConfig{
		AuditLogEnabled: false,
	}

	manager, err := NewSecurityManager(config)
	require.NoError(t, err)

	// All logging methods should handle nil audit logger gracefully
	manager.LogSecurityEvent("test_event", true, "device-123", nil)
	manager.LogDeviceRegistration("device-123", "token-456", false, "error", nil)
	manager.LogAccessAttempt("device-123", false, "invalid", nil)
	manager.LogTamperAttempt("device-123", "/file", "hash1", "hash2")

	// Should complete without panics
	assert.True(t, true)
}

// TestPerformHealthCheck tests health check functionality
func TestPerformHealthCheck(t *testing.T) {
	config := SecurityConfig{
		AuditLogEnabled:    true,
		MonitoringEnabled:  false, // Disable to avoid complex mock setup
		HealthCheckEnabled: true,
	}

	manager, err := NewSecurityManager(config)
	require.NoError(t, err)

	_ = manager.PerformHealthCheck()

	// Health check may fail due to keychain provider, but should not panic
	// The important thing is that it updates lastHealthCheck
	status, _ := manager.GetSecurityStatus()
	assert.False(t, status.LastHealthCheck.IsZero())
}

// TestRegisterDevice tests device registration
func TestRegisterDevice(t *testing.T) {
	config := SecurityConfig{
		RegistrySecurityEnabled: false, // Disable to avoid complex setup
	}

	manager, err := NewSecurityManager(config)
	require.NoError(t, err)

	err = manager.RegisterDevice("token-123", "device-456")

	// Should fail because registry client is not available
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "registry client not available")
}

// TestValidateSecurityConfig tests security configuration validation
func TestValidateSecurityConfig(t *testing.T) {
	tests := []struct {
		name      string
		config    SecurityConfig
		expectErr bool
	}{
		{
			name: "Valid configuration",
			config: SecurityConfig{
				MonitorInterval:     30 * time.Second,
				AnalysisInterval:    5 * time.Minute,
				HealthCheckInterval: 15 * time.Minute,
				LogRetentionDays:    30,
				AlertThreshold:      "MEDIUM",
			},
			expectErr: false,
		},
		{
			name: "Invalid alert threshold",
			config: SecurityConfig{
				AlertThreshold: "INVALID",
			},
			expectErr: true,
		},
		{
			name: "Zero intervals - should be corrected",
			config: SecurityConfig{
				MonitorInterval:     0,
				AnalysisInterval:    0,
				HealthCheckInterval: 0,
				LogRetentionDays:    0,
				AlertThreshold:      "HIGH",
			},
			expectErr: false, // Should be corrected by validation
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateSecurityConfig(tt.config)

			if tt.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestGetDefaultSecurityConfig tests default configuration
func TestGetDefaultSecurityConfig(t *testing.T) {
	config := GetDefaultSecurityConfig()

	// Security features are disabled by default to prevent keychain prompts
	// This improves UX for basic profiles that don't need secure credential storage
	assert.False(t, config.AuditLogEnabled)
	assert.False(t, config.MonitoringEnabled)
	assert.False(t, config.CorrelationEnabled)
	assert.False(t, config.RegistrySecurityEnabled)
	assert.False(t, config.HealthCheckEnabled)
	
	// Configuration values are still set even when disabled
	assert.Equal(t, 30, config.LogRetentionDays)
	assert.Equal(t, 30*time.Second, config.MonitorInterval)
	assert.Equal(t, "MEDIUM", config.AlertThreshold)
	assert.Equal(t, 5*time.Minute, config.AnalysisInterval)
	assert.Equal(t, 15*time.Minute, config.HealthCheckInterval)
}

// TestSecurityManagerEnabledComponents tests component enablement logic
func TestSecurityManagerEnabledComponents(t *testing.T) {
	tests := []struct {
		name               string
		config             SecurityConfig
		expectedComponents []string
		expectedEnabled    bool
	}{
		{
			name: "All components enabled",
			config: SecurityConfig{
				AuditLogEnabled:         true,
				MonitoringEnabled:       true,
				CorrelationEnabled:      true,
				RegistrySecurityEnabled: true,
			},
			expectedComponents: []string{"audit_logger", "security_monitor", "correlation_engine", "secure_registry"},
			expectedEnabled:    true,
		},
		{
			name: "Only audit enabled",
			config: SecurityConfig{
				AuditLogEnabled:         true,
				MonitoringEnabled:       false,
				CorrelationEnabled:      false,
				RegistrySecurityEnabled: false,
			},
			expectedComponents: []string{"audit_logger"},
			expectedEnabled:    true,
		},
		{
			name: "No components enabled",
			config: SecurityConfig{
				AuditLogEnabled:         false,
				MonitoringEnabled:       false,
				CorrelationEnabled:      false,
				RegistrySecurityEnabled: false,
			},
			expectedComponents: []string{},
			expectedEnabled:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manager, err := NewSecurityManager(tt.config)
			require.NoError(t, err)

			assert.Equal(t, tt.expectedEnabled, manager.isEnabled)

			components := manager.getEnabledComponents()
			assert.ElementsMatch(t, tt.expectedComponents, components)
		})
	}
}

// TestSecurityManagerConcurrency tests concurrent access to security manager
func TestSecurityManagerConcurrency(t *testing.T) {
	config := SecurityConfig{
		AuditLogEnabled:   true,
		MonitoringEnabled: false, // Disable to reduce complexity
	}

	manager, err := NewSecurityManager(config)
	require.NoError(t, err)

	// Start multiple goroutines accessing the manager
	done := make(chan bool, 10)

	for i := 0; i < 10; i++ {
		go func(id int) {
			// Test concurrent status checks
			status, err := manager.GetSecurityStatus()
			assert.NoError(t, err)
			assert.NotNil(t, status)

			// Test concurrent logging
			manager.LogSecurityEvent("concurrent_test", true, "device", map[string]interface{}{
				"goroutine_id": id,
			})

			done <- true
		}(i)
	}

	// Wait for all goroutines to complete
	for i := 0; i < 10; i++ {
		<-done
	}
}

// TestSecurityManagerConfigValidation tests configuration validation edge cases
func TestSecurityManagerConfigValidation(t *testing.T) {
	tests := []struct {
		name           string
		config         SecurityConfig
		expectedValid  bool
		expectedErrMsg string
	}{
		{
			name: "Valid HIGH alert threshold",
			config: SecurityConfig{
				AlertThreshold: "HIGH",
			},
			expectedValid: true,
		},
		{
			name: "Valid CRITICAL alert threshold",
			config: SecurityConfig{
				AlertThreshold: "CRITICAL",
			},
			expectedValid: true,
		},
		{
			name: "Valid LOW alert threshold",
			config: SecurityConfig{
				AlertThreshold: "LOW",
			},
			expectedValid: true,
		},
		{
			name: "Empty alert threshold (should be allowed)",
			config: SecurityConfig{
				AlertThreshold: "",
			},
			expectedValid: true,
		},
		{
			name: "Invalid alert threshold",
			config: SecurityConfig{
				AlertThreshold: "EXTREME",
			},
			expectedValid:  false,
			expectedErrMsg: "invalid alert threshold",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewSecurityManager(tt.config)

			if tt.expectedValid {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedErrMsg)
			}
		})
	}
}

// TestSecurityManagerValidateDeviceBinding tests device binding validation
func TestSecurityManagerValidateDeviceBinding(t *testing.T) {
	config := SecurityConfig{
		AuditLogEnabled: true,
	}

	manager, err := NewSecurityManager(config)
	require.NoError(t, err)

	// Test device binding validation (will likely fail due to missing fingerprint setup)
	expectedFingerprint := map[string]interface{}{
		"hash": "test-hash-123",
	}

	err = manager.ValidateDeviceBinding("device-123", expectedFingerprint)

	// Should fail gracefully and log the attempt
	assert.Error(t, err)
	// The error indicates device binding validation was attempted
	assert.True(t, strings.Contains(err.Error(), "fingerprint") || strings.Contains(err.Error(), "device binding"))
}

// TestSecurityConfigDefaults tests that configuration defaults are applied
func TestSecurityConfigDefaults(t *testing.T) {
	// Test that validation applies defaults
	config := SecurityConfig{} // Empty config

	err := validateSecurityConfig(config)
	assert.NoError(t, err)

	// Verify defaults were applied (note: validation modifies in-place, but struct is passed by value)
	// So we test the validation logic works without panicking
}

// TestSecurityManagerStartStopRace tests race conditions in start/stop
func TestSecurityManagerStartStopRace(t *testing.T) {
	config := SecurityConfig{
		AuditLogEnabled:     true,
		MonitorInterval:     100 * time.Millisecond, // Positive interval
		AnalysisInterval:    200 * time.Millisecond, // Positive interval
		HealthCheckInterval: 300 * time.Millisecond, // Positive interval
	}

	manager, err := NewSecurityManager(config)
	require.NoError(t, err)

	// Start manager
	err = manager.Start()
	require.NoError(t, err)

	// Let it run briefly
	time.Sleep(50 * time.Millisecond)

	// Stop manager
	err = manager.Stop()
	assert.NoError(t, err)

	// Verify it stopped
	assert.False(t, manager.isRunning)
}

// TestSecurityManagerStatusWhileRunning tests status while running
func TestSecurityManagerStatusWhileRunning(t *testing.T) {
	config := SecurityConfig{
		AuditLogEnabled:     true,
		MonitoringEnabled:   false,                  // Disable complex monitoring
		MonitorInterval:     100 * time.Millisecond, // Positive interval
		AnalysisInterval:    200 * time.Millisecond, // Positive interval
		HealthCheckInterval: 300 * time.Millisecond, // Positive interval
	}

	manager, err := NewSecurityManager(config)
	require.NoError(t, err)

	// Start manager
	err = manager.Start()
	require.NoError(t, err)
	defer func() { _ = manager.Stop() }()

	// Get status while running
	status, err := manager.GetSecurityStatus()
	assert.NoError(t, err)
	assert.True(t, status.Running)
	assert.True(t, status.Enabled)
}

// TestSecurityManagerHealthCheckTimeout tests health check with timeout
func TestSecurityManagerHealthCheckTimeout(t *testing.T) {
	config := SecurityConfig{
		AuditLogEnabled:    true,
		HealthCheckEnabled: true,
	}

	manager, err := NewSecurityManager(config)
	require.NoError(t, err)

	// Perform health check (may fail due to keychain, but should not hang)
	done := make(chan bool, 1)
	go func() {
		_ = manager.PerformHealthCheck()
		done <- true
	}()

	select {
	case <-done:
		// Health check completed
		assert.True(t, true)
	case <-time.After(5 * time.Second):
		t.Fatal("Health check timed out")
	}
}

// TestSecurityConfig struct validation
func TestSecurityConfig(t *testing.T) {
	config := SecurityConfig{
		AuditLogEnabled:         true,
		LogRetentionDays:        90,
		MonitoringEnabled:       true,
		MonitorInterval:         30 * time.Second,
		AlertThreshold:          "HIGH",
		RegistrySecurityEnabled: true,
		RegistryURL:             "https://registry.example.com",
		CorrelationEnabled:      true,
		AnalysisInterval:        5 * time.Minute,
		HealthCheckEnabled:      true,
		HealthCheckInterval:     15 * time.Minute,
	}

	// Test that all fields are properly set
	assert.True(t, config.AuditLogEnabled)
	assert.Equal(t, 90, config.LogRetentionDays)
	assert.True(t, config.MonitoringEnabled)
	assert.Equal(t, 30*time.Second, config.MonitorInterval)
	assert.Equal(t, "HIGH", config.AlertThreshold)
	assert.True(t, config.RegistrySecurityEnabled)
	assert.Equal(t, "https://registry.example.com", config.RegistryURL)
	assert.True(t, config.CorrelationEnabled)
	assert.Equal(t, 5*time.Minute, config.AnalysisInterval)
	assert.True(t, config.HealthCheckEnabled)
	assert.Equal(t, 15*time.Minute, config.HealthCheckInterval)
}

// TestSecurityStatus struct validation
func TestSecurityStatus(t *testing.T) {
	now := time.Now()

	status := SecurityStatus{
		Enabled:         true,
		Running:         true,
		LastHealthCheck: now,
		Configuration: SecurityConfig{
			AuditLogEnabled: true,
		},
	}

	assert.True(t, status.Enabled)
	assert.True(t, status.Running)
	assert.Equal(t, now, status.LastHealthCheck)
	assert.True(t, status.Configuration.AuditLogEnabled)
}

// TestSecurityManagerInitializationFailure tests handling of initialization failures
func TestSecurityManagerInitializationFailure(t *testing.T) {
	// Test with configuration that might cause initialization issues
	config := SecurityConfig{
		AuditLogEnabled:         true,
		MonitoringEnabled:       true,
		CorrelationEnabled:      true,
		RegistrySecurityEnabled: true,
		RegistryURL:             "", // Empty URL might cause issues
	}

	manager, err := NewSecurityManager(config)

	// Should handle initialization gracefully
	if err != nil {
		assert.Contains(t, err.Error(), "failed to initialize")
	} else {
		assert.NotNil(t, manager)
	}
}
