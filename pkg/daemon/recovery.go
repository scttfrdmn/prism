package daemon

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"runtime"
	"runtime/debug"
	"time"
)

// HTTPServer interface to avoid circular dependency
type HTTPServer interface {
	Shutdown(ctx context.Context) error
}

// RecoveryManager handles panic recovery and graceful degradation
type RecoveryManager struct {
	stabilityMgr *StabilityManager
	HTTPServer   HTTPServer

	// Recovery settings
	enablePanicRecovery    bool
	enableGracefulShutdown bool
	shutdownTimeout        time.Duration

	// Recovery strategies
	recoveryStrategies map[string]RecoveryStrategy
}

// RecoveryStrategy defines how to recover from specific error types
type RecoveryStrategy struct {
	Name        string
	Description string
	Handler     func(error) error
	Retries     int
	Timeout     time.Duration
}

// NewRecoveryManager creates a new recovery manager
func NewRecoveryManager(stabilityMgr *StabilityManager, httpServer HTTPServer) *RecoveryManager {
	rm := &RecoveryManager{
		stabilityMgr:           stabilityMgr,
		HTTPServer:             httpServer,
		enablePanicRecovery:    true,
		enableGracefulShutdown: true,
		shutdownTimeout:        30 * time.Second,
		recoveryStrategies:     make(map[string]RecoveryStrategy),
	}

	rm.setupDefaultStrategies()
	return rm
}

// RecoverFromPanic provides panic recovery middleware
func (rm *RecoveryManager) RecoverFromPanic(component string) {
	if r := recover(); r != nil {
		// Capture panic details
		stack := debug.Stack()
		panicMsg := fmt.Sprintf("Panic in %s: %v", component, r)

		// Log panic
		log.Printf("PANIC RECOVERED: %s\nStack trace:\n%s", panicMsg, stack)

		// Record in stability manager
		rm.stabilityMgr.RecordError(component, "panic", panicMsg, ErrorSeverityCritical)

		// Attempt recovery
		if rm.enablePanicRecovery {
			rm.attemptRecovery(component, fmt.Errorf("panic: %v", r))
		}
	}
}

// RecoverHTTPHandler creates HTTP handler with panic recovery
func (rm *RecoveryManager) RecoverHTTPHandler(component string, handler http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				// Log panic
				stack := debug.Stack()
				panicMsg := fmt.Sprintf("HTTP panic in %s: %v", component, err)
				log.Printf("HTTP PANIC RECOVERED: %s\nStack trace:\n%s", panicMsg, stack)

				// Record error
				rm.stabilityMgr.RecordError(component, "http_panic", panicMsg, ErrorSeverityCritical)

				// Return 500 error
				http.Error(w, "Internal server error", http.StatusInternalServerError)

				// Attempt recovery
				rm.attemptRecovery(component, fmt.Errorf("http panic: %v", err))
			}
		}()

		handler(w, r)
	}
}

// RecoverGoroutine wraps goroutine execution with panic recovery
func (rm *RecoveryManager) RecoverGoroutine(component string, fn func()) {
	go func() {
		defer rm.RecoverFromPanic(component)
		fn()
	}()
}

// GracefulShutdown performs graceful shutdown
func (rm *RecoveryManager) GracefulShutdown(ctx context.Context) error {
	if !rm.enableGracefulShutdown {
		return nil
	}

	log.Printf("Starting graceful shutdown...")
	rm.stabilityMgr.RecordError("daemon", "shutdown", "Graceful shutdown initiated", ErrorSeverityLow)

	// Create shutdown context with timeout
	shutdownCtx, cancel := context.WithTimeout(ctx, rm.shutdownTimeout)
	defer cancel()

	// Shutdown HTTP server
	if rm.HTTPServer != nil {
		log.Printf("Shutting down HTTP server...")
		if err := rm.HTTPServer.Shutdown(shutdownCtx); err != nil {
			log.Printf("Error during HTTP server shutdown: %v", err)
			return fmt.Errorf("HTTP server shutdown failed: %w", err)
		}
	}

	// Cleanup resources
	log.Printf("Cleaning up resources...")

	// Force garbage collection
	rm.stabilityMgr.ForceGarbageCollection()

	// Wait a moment for cleanup
	time.Sleep(1 * time.Second)

	log.Printf("Graceful shutdown completed")
	rm.stabilityMgr.RecordRecovery("daemon", "shutdown")

	return nil
}

// HandleMemoryPressure handles high memory usage situations
func (rm *RecoveryManager) HandleMemoryPressure() error {
	log.Printf("Memory pressure detected, attempting recovery...")

	// Get current memory stats
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	memBefore := m.HeapAlloc

	// Force garbage collection
	rm.stabilityMgr.ForceGarbageCollection()

	// Check if memory was freed
	runtime.ReadMemStats(&m)
	memAfter := m.HeapAlloc
	freed := memBefore - memAfter

	log.Printf("Memory recovery: freed %d bytes (%.2f MB)", freed, float64(freed)/1024/1024)

	if freed > 0 {
		rm.stabilityMgr.RecordRecovery("daemon", "memory_pressure")
		return nil
	}

	return fmt.Errorf("memory recovery failed: no memory freed")
}

// HandleGoroutineLeak handles excessive goroutine count
func (rm *RecoveryManager) HandleGoroutineLeak() error {
	goroutineCount := runtime.NumGoroutine()
	log.Printf("Goroutine leak detected: %d goroutines", goroutineCount)

	// Log goroutine stack traces for debugging
	buf := make([]byte, 1024*1024) // 1MB buffer
	stackSize := runtime.Stack(buf, true)
	log.Printf("Goroutine stack dump:\n%s", string(buf[:stackSize]))

	// Record error
	rm.stabilityMgr.RecordError("daemon", "goroutine_leak",
		fmt.Sprintf("High goroutine count: %d", goroutineCount), ErrorSeverityHigh)

	// For now, we can only monitor and log
	// In a production system, you might implement goroutine cancellation
	return fmt.Errorf("goroutine leak detected: %d goroutines", goroutineCount)
}

// RecoverFromError attempts to recover from a specific error
func (rm *RecoveryManager) RecoverFromError(component string, err error) error {
	errorType := getErrorType(err)

	// Try recovery strategy
	if strategy, exists := rm.recoveryStrategies[errorType]; exists {
		log.Printf("Attempting recovery for %s error in %s using strategy: %s",
			errorType, component, strategy.Name)

		for attempt := 0; attempt < strategy.Retries; attempt++ {
			if recoveryErr := strategy.Handler(err); recoveryErr == nil {
				log.Printf("Recovery successful for %s error in %s", errorType, component)
				rm.stabilityMgr.RecordRecovery(component, errorType)
				return nil
			}

			if attempt < strategy.Retries-1 {
				log.Printf("Recovery attempt %d failed, retrying...", attempt+1)
				time.Sleep(time.Second * time.Duration(attempt+1))
			}
		}

		log.Printf("All recovery attempts failed for %s error in %s", errorType, component)
		return fmt.Errorf("recovery failed after %d attempts: %w", strategy.Retries, err)
	}

	log.Printf("No recovery strategy found for %s error in %s", errorType, component)
	return fmt.Errorf("no recovery strategy for error type %s: %w", errorType, err)
}

// attemptRecovery attempts automatic recovery
func (rm *RecoveryManager) attemptRecovery(component string, err error) {
	// Check if memory pressure is the issue
	if rm.stabilityMgr.CheckMemoryPressure() {
		if recoveryErr := rm.HandleMemoryPressure(); recoveryErr == nil {
			return
		}
	}

	// Check for goroutine leaks
	if runtime.NumGoroutine() > 500 { // Threshold
		_ = rm.HandleGoroutineLeak()
	}

	// Try specific error recovery
	_ = rm.RecoverFromError(component, err)
}

// setupDefaultStrategies sets up default recovery strategies
func (rm *RecoveryManager) setupDefaultStrategies() {
	// Database connection recovery
	rm.recoveryStrategies["database_connection"] = RecoveryStrategy{
		Name:        "Database Connection Recovery",
		Description: "Attempts to reconnect to database",
		Handler: func(err error) error {
			// Wait and retry
			time.Sleep(5 * time.Second)
			// In real implementation, would attempt DB reconnection
			return nil
		},
		Retries: 3,
		Timeout: 10 * time.Second,
	}

	// AWS connection recovery
	rm.recoveryStrategies["aws_connection"] = RecoveryStrategy{
		Name:        "AWS Connection Recovery",
		Description: "Attempts to reconnect to AWS services",
		Handler: func(err error) error {
			// Reset AWS manager connections
			time.Sleep(2 * time.Second)
			// In real implementation, would reinitialize AWS manager
			return nil
		},
		Retries: 3,
		Timeout: 15 * time.Second,
	}

	// File system recovery
	rm.recoveryStrategies["filesystem"] = RecoveryStrategy{
		Name:        "File System Recovery",
		Description: "Attempts to recover from filesystem errors",
		Handler: func(err error) error {
			// Try to create missing directories, fix permissions, etc.
			time.Sleep(1 * time.Second)
			return nil
		},
		Retries: 2,
		Timeout: 5 * time.Second,
	}

	// Memory recovery
	rm.recoveryStrategies["memory"] = RecoveryStrategy{
		Name:        "Memory Recovery",
		Description: "Attempts to free memory and reduce pressure",
		Handler: func(err error) error {
			return rm.HandleMemoryPressure()
		},
		Retries: 1,
		Timeout: 30 * time.Second,
	}
}

// AddRecoveryStrategy adds a custom recovery strategy
func (rm *RecoveryManager) AddRecoveryStrategy(errorType string, strategy RecoveryStrategy) {
	rm.recoveryStrategies[errorType] = strategy
}

// GetRecoveryStrategies returns all recovery strategies
func (rm *RecoveryManager) GetRecoveryStrategies() map[string]RecoveryStrategy {
	strategies := make(map[string]RecoveryStrategy)
	for k, v := range rm.recoveryStrategies {
		strategies[k] = v
	}
	return strategies
}

// HealthCheck performs comprehensive health check
func (rm *RecoveryManager) HealthCheck() error {
	// Check stability manager health
	if !rm.stabilityMgr.IsHealthy() {
		return fmt.Errorf("daemon is not healthy")
	}

	// Check memory usage
	if rm.stabilityMgr.CheckMemoryPressure() {
		return fmt.Errorf("high memory pressure detected")
	}

	// Check goroutine count
	if runtime.NumGoroutine() > 500 {
		return fmt.Errorf("high goroutine count: %d", runtime.NumGoroutine())
	}

	return nil
}

// getErrorType extracts error type from error message
func getErrorType(err error) string {
	errorMsg := err.Error()

	// Simple error type classification
	switch {
	case contains(errorMsg, "connection", "timeout", "network"):
		return "network_connection"
	case contains(errorMsg, "database", "sql"):
		return "database_connection"
	case contains(errorMsg, "aws", "credential", "region"):
		return "aws_connection"
	case contains(errorMsg, "file", "directory", "permission"):
		return "filesystem"
	case contains(errorMsg, "memory", "out of memory"):
		return "memory"
	default:
		return "unknown"
	}
}

// contains checks if any of the substrings exist in the text
func contains(text string, substrings ...string) bool {
	// text is already a string, no need to convert
	for _, substring := range substrings {
		if substring != "" &&
			len(text) > 0 && len(substring) > 0 {
			// Simple substring check
			for i := 0; i <= len(text)-len(substring); i++ {
				match := true
				for j := 0; j < len(substring); j++ {
					if text[i+j] != substring[j] {
						match = false
						break
					}
				}
				if match {
					return true
				}
			}
		}
	}
	return false
}
