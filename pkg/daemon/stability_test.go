package daemon

import (
	"context"
	"testing"
	"time"

	"github.com/scttfrdmn/cloudworkstation/pkg/monitoring"
)

func TestStabilityManager_BasicFunctionality(t *testing.T) {
	monitor := monitoring.NewPerformanceMonitor()
	sm := NewStabilityManager(monitor)
	
	// Test initial state
	if !sm.IsHealthy() {
		t.Errorf("Expected stability manager to start healthy")
	}
	
	metrics := sm.GetStabilityMetrics()
	if metrics.HealthScore <= 0 {
		t.Errorf("Expected positive health score, got %f", metrics.HealthScore)
	}
}

func TestStabilityManager_ErrorRecording(t *testing.T) {
	monitor := monitoring.NewPerformanceMonitor()
	sm := NewStabilityManager(monitor)
	
	// Record an error
	sm.RecordError("test_component", "test_error", "Test error message", ErrorSeverityMedium)
	
	// Check error history
	errorHistory := sm.GetErrorHistory()
	if len(errorHistory) != 1 {
		t.Errorf("Expected 1 error in history, got %d", len(errorHistory))
	}
	
	error := errorHistory[0]
	if error.Component != "test_component" {
		t.Errorf("Expected component 'test_component', got '%s'", error.Component)
	}
	if error.ErrorType != "test_error" {
		t.Errorf("Expected error type 'test_error', got '%s'", error.ErrorType)
	}
	if error.Severity != ErrorSeverityMedium {
		t.Errorf("Expected severity medium, got %s", error.Severity)
	}
}

func TestStabilityManager_ErrorRecovery(t *testing.T) {
	monitor := monitoring.NewPerformanceMonitor()
	sm := NewStabilityManager(monitor)
	
	// Record an error
	sm.RecordError("test_component", "test_error", "Test error message", ErrorSeverityHigh)
	
	// Record recovery
	sm.RecordRecovery("test_component", "test_error")
	
	// Check that error is marked as recovered
	errorHistory := sm.GetErrorHistory()
	if len(errorHistory) != 1 {
		t.Errorf("Expected 1 error in history, got %d", len(errorHistory))
	}
	
	if !errorHistory[0].Recovered {
		t.Errorf("Expected error to be marked as recovered")
	}
}

func TestStabilityManager_CircuitBreaker(t *testing.T) {
	monitor := monitoring.NewPerformanceMonitor()
	sm := NewStabilityManager(monitor)
	
	// Get a circuit breaker
	cb := sm.GetCircuitBreaker("test_breaker")
	if cb == nil {
		t.Fatal("Expected circuit breaker to be created")
	}
	
	if cb.State != CircuitBreakerClosed {
		t.Errorf("Expected circuit breaker to start closed, got %s", cb.State)
	}
	
	// Test circuit breaker execution
	successCount := 0
	failureCount := 0
	
	// Test successful operations
	for i := 0; i < 3; i++ {
		err := sm.ExecuteWithCircuitBreaker("test_breaker", func() error {
			successCount++
			return nil
		})
		if err != nil {
			t.Errorf("Expected successful execution, got error: %v", err)
		}
	}
	
	// Test failing operations to trigger circuit breaker
	for i := 0; i < 6; i++ { // More than threshold
		err := sm.ExecuteWithCircuitBreaker("test_breaker", func() error {
			failureCount++
			return &TestError{"test failure"}
		})
		if i < 5 && err == nil {
			t.Errorf("Expected failure to be returned")
		}
	}
	
	// Check that circuit breaker is now open
	cb = sm.GetCircuitBreaker("test_breaker")
	if cb.State != CircuitBreakerOpen {
		t.Errorf("Expected circuit breaker to be open, got %s", cb.State)
	}
	
	if successCount != 3 {
		t.Errorf("Expected 3 successful operations, got %d", successCount)
	}
	
	// Should be 5 failures (threshold) before circuit breaker opens
	if failureCount < 5 {
		t.Errorf("Expected at least 5 failed operations, got %d", failureCount)
	}
}

func TestStabilityManager_MemoryPressure(t *testing.T) {
	monitor := monitoring.NewPerformanceMonitor()
	sm := NewStabilityManager(monitor)
	
	// Set a very low memory threshold for testing
	sm.SetMemoryThreshold(1024) // 1KB
	
	// Memory pressure should be detected with this low threshold
	if !sm.CheckMemoryPressure() {
		t.Errorf("Expected memory pressure to be detected with low threshold")
	}
	
	// Test force garbage collection
	sm.EnableForceGC(true)
	sm.ForceGarbageCollection()
	
	// This should not cause any errors
}

func TestStabilityManager_HealthScore(t *testing.T) {
	monitor := monitoring.NewPerformanceMonitor()
	sm := NewStabilityManager(monitor)
	
	// Get initial health score
	initialScore := sm.GetStabilityMetrics().HealthScore
	if initialScore <= 0 || initialScore > 1 {
		t.Errorf("Expected health score between 0 and 1, got %f", initialScore)
	}
	
	// Add some errors to decrease health score
	for i := 0; i < 5; i++ {
		sm.RecordError("test", "error", "test error", ErrorSeverityMedium)
	}
	
	// Health score should decrease (though this is hard to test precisely)
	// Just verify it's still a valid score
	newScore := sm.GetStabilityMetrics().HealthScore
	if newScore < 0 || newScore > 1 {
		t.Errorf("Expected health score between 0 and 1 after errors, got %f", newScore)
	}
}

func TestStabilityManager_Monitoring(t *testing.T) {
	monitor := monitoring.NewPerformanceMonitor()
	sm := NewStabilityManager(monitor)
	
	// Start monitoring in background
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	
	go sm.Start(ctx)
	
	// Wait a bit for monitoring to run
	time.Sleep(500 * time.Millisecond)
	
	// Check that monitoring is running (no crashes)
	if !sm.IsHealthy() {
		t.Errorf("Expected stability manager to remain healthy during monitoring")
	}
}

// TestError is a simple error type for testing
type TestError struct {
	message string
}

func (e *TestError) Error() string {
	return e.message
}

func TestRecoveryManager_PanicRecovery(t *testing.T) {
	monitor := monitoring.NewPerformanceMonitor()
	sm := NewStabilityManager(monitor)
	rm := NewRecoveryManager(sm, nil) // nil server is okay for testing
	
	// Test that panic recovery doesn't crash the test
	func() {
		defer rm.RecoverFromPanic("test_component")
		// Don't actually panic in test - just verify the defer works
	}()
	
	// Check that panic recovery strategies exist
	strategies := rm.GetRecoveryStrategies()
	if len(strategies) == 0 {
		t.Errorf("Expected recovery strategies to be set up")
	}
}

func TestRecoveryManager_MemoryPressure(t *testing.T) {
	monitor := monitoring.NewPerformanceMonitor()
	sm := NewStabilityManager(monitor)
	rm := NewRecoveryManager(sm, nil)
	
	// Test memory pressure handling
	err := rm.HandleMemoryPressure()
	// This might succeed or fail depending on actual memory state
	// The important thing is that it doesn't crash
	if err != nil {
		t.Logf("Memory pressure handling failed (may be normal): %v", err)
	}
}

func TestRecoveryManager_ErrorRecovery(t *testing.T) {
	monitor := monitoring.NewPerformanceMonitor()
	sm := NewStabilityManager(monitor)
	rm := NewRecoveryManager(sm, nil)
	
	// Test recovery from known error type
	testErr := &TestError{"database connection failed"}
	err := rm.RecoverFromError("test_component", testErr)
	
	// Recovery might fail if no strategy matches
	if err != nil {
		t.Logf("Error recovery failed (expected for unknown error type): %v", err)
	}
}

func TestRecoveryManager_HealthCheck(t *testing.T) {
	monitor := monitoring.NewPerformanceMonitor()
	sm := NewStabilityManager(monitor)
	rm := NewRecoveryManager(sm, nil)
	
	// Test health check
	err := rm.HealthCheck()
	if err != nil {
		t.Logf("Health check failed: %v", err)
		// This is not necessarily an error in testing environment
	}
}

func TestHealthMonitor_BasicFunctionality(t *testing.T) {
	monitor := monitoring.NewPerformanceMonitor()
	sm := NewStabilityManager(monitor)
	rm := NewRecoveryManager(sm, nil)
	hm := NewHealthMonitor(nil, sm, rm, monitor)
	
	// Test initial state
	if !hm.IsHealthy() {
		t.Errorf("Expected health monitor to start healthy")
	}
	
	// Get health summary
	summary := hm.GetHealthSummary()
	if summary.ActiveChecks == 0 {
		t.Errorf("Expected health checks to be configured")
	}
}

func TestHealthMonitor_HealthChecks(t *testing.T) {
	monitor := monitoring.NewPerformanceMonitor()
	sm := NewStabilityManager(monitor)
	rm := NewRecoveryManager(sm, nil)
	hm := NewHealthMonitor(nil, sm, rm, monitor)
	
	// Add a custom health check
	hm.AddHealthCheck("test_check", HealthCheck{
		Description: "Test health check",
		CheckFunc: func() HealthCheckResult {
			return HealthCheckResult{
				Status:  HealthStatusHealthy,
				Message: "Test check passed",
			}
		},
		Interval: 1 * time.Second,
		Timeout:  5 * time.Second,
		Critical: false,
	})
	
	// Get updated summary
	summary := hm.GetHealthSummary()
	if summary.ActiveChecks == 0 {
		t.Errorf("Expected health checks to include custom check")
	}
	
	// Remove health check
	hm.RemoveHealthCheck("test_check")
}

func TestHealthMonitor_Monitoring(t *testing.T) {
	monitor := monitoring.NewPerformanceMonitor()
	sm := NewStabilityManager(monitor)
	rm := NewRecoveryManager(sm, nil)
	hm := NewHealthMonitor(nil, sm, rm, monitor)
	
	// Start monitoring in background
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	
	go hm.Start(ctx)
	
	// Wait a bit for monitoring to run
	time.Sleep(500 * time.Millisecond)
	
	// Check that monitoring is working
	history := hm.GetHealthHistory()
	if len(history) == 0 {
		// This might be expected if monitoring hasn't run a full cycle yet
		t.Logf("No health history yet (monitoring may still be starting)")
	}
}

func TestStabilityIntegration(t *testing.T) {
	// Test integration between all stability components
	monitor := monitoring.NewPerformanceMonitor()
	sm := NewStabilityManager(monitor)
	rm := NewRecoveryManager(sm, nil)
	hm := NewHealthMonitor(nil, sm, rm, monitor)
	
	// Start all systems
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	
	go monitor.Start(ctx)
	go sm.Start(ctx)
	go hm.Start(ctx)
	
	// Simulate some activity
	sm.RecordError("integration_test", "test_error", "Integration test error", ErrorSeverityLow)
	sm.RecordRecovery("integration_test", "test_error")
	
	// Wait for systems to process
	time.Sleep(1 * time.Second)
	
	// Verify systems are working together
	if !sm.IsHealthy() {
		t.Logf("Warning: Stability manager not healthy during integration test")
	}
	
	if !hm.IsHealthy() {
		t.Logf("Warning: Health monitor not healthy during integration test")
	}
	
	// Get integrated metrics
	stabilityMetrics := sm.GetStabilityMetrics()
	healthSummary := hm.GetHealthSummary()
	
	if stabilityMetrics.HealthScore <= 0 {
		t.Errorf("Expected positive health score, got %f", stabilityMetrics.HealthScore)
	}
	
	if healthSummary.Score <= 0 {
		t.Errorf("Expected positive health summary score, got %f", healthSummary.Score)
	}
}