package daemon

import (
	"context"
	"fmt"
	"runtime"
	"runtime/debug"
	"sync"
	"time"

	"github.com/scttfrdmn/cloudworkstation/pkg/monitoring"
)

// StabilityManager manages daemon stability, memory, and error recovery
type StabilityManager struct {
	monitor *monitoring.PerformanceMonitor
	mu      sync.RWMutex

	// Memory management
	memoryThreshold int64 // Memory threshold in bytes
	gcInterval      time.Duration
	lastGC          time.Time
	forceGCEnabled  bool

	// Error tracking
	errorHistory    []ErrorRecord
	maxErrorHistory int
	errorCounts     map[string]int
	circuitBreakers map[string]*CircuitBreaker

	// Resource limits
	maxGoroutines int
	maxMemoryMB   int64

	// Recovery settings
	recoveryEnabled bool
	panicRecovery   bool

	// Monitoring
	stabilityMetrics StabilityMetrics
	isHealthy        bool
	lastHealthCheck  time.Time
}

// ErrorRecord tracks error occurrences for analysis
type ErrorRecord struct {
	Timestamp time.Time     `json:"timestamp"`
	ErrorType string        `json:"error_type"`
	Message   string        `json:"message"`
	Component string        `json:"component"`
	Severity  ErrorSeverity `json:"severity"`
	Count     int           `json:"count"`
	Recovered bool          `json:"recovered"`
}

// ErrorSeverity represents error severity levels
type ErrorSeverity string

const (
	ErrorSeverityLow      ErrorSeverity = "low"
	ErrorSeverityMedium   ErrorSeverity = "medium"
	ErrorSeverityHigh     ErrorSeverity = "high"
	ErrorSeverityCritical ErrorSeverity = "critical"
)

// CircuitBreaker prevents cascade failures
type CircuitBreaker struct {
	Name             string
	FailureCount     int
	LastFailure      time.Time
	State            CircuitBreakerState
	FailureThreshold int
	ResetTimeout     time.Duration
}

// CircuitBreakerState represents circuit breaker states
type CircuitBreakerState string

const (
	CircuitBreakerClosed   CircuitBreakerState = "closed"
	CircuitBreakerOpen     CircuitBreakerState = "open"
	CircuitBreakerHalfOpen CircuitBreakerState = "half_open"
)

// StabilityMetrics tracks daemon stability metrics
type StabilityMetrics struct {
	Uptime              time.Duration `json:"uptime"`
	MemoryUsageMB       float64       `json:"memory_usage_mb"`
	GoroutineCount      int           `json:"goroutine_count"`
	GCCount             uint32        `json:"gc_count"`
	LastGCPause         time.Duration `json:"last_gc_pause"`
	TotalErrors         int           `json:"total_errors"`
	CriticalErrors      int           `json:"critical_errors"`
	RecoveredErrors     int           `json:"recovered_errors"`
	CircuitBreakersOpen int           `json:"circuit_breakers_open"`
	HealthScore         float64       `json:"health_score"`
}

// NewStabilityManager creates a new stability manager
func NewStabilityManager(monitor *monitoring.PerformanceMonitor) *StabilityManager {
	return &StabilityManager{
		monitor:         monitor,
		memoryThreshold: 512 * 1024 * 1024, // 512MB default
		gcInterval:      5 * time.Minute,
		maxErrorHistory: 1000,
		errorCounts:     make(map[string]int),
		circuitBreakers: make(map[string]*CircuitBreaker),
		maxGoroutines:   1000,
		maxMemoryMB:     1024,
		recoveryEnabled: true,
		panicRecovery:   true,
		isHealthy:       true,
		lastHealthCheck: time.Now(),
	}
}

// Start begins stability monitoring
func (sm *StabilityManager) Start(ctx context.Context) {
	// Start background monitoring
	go sm.monitorResources(ctx)
	go sm.manageMemory(ctx)
	go sm.monitorCircuitBreakers(ctx)

	// Set up panic recovery if enabled
	if sm.panicRecovery {
		sm.setupPanicRecovery()
	}

	// Initial health check
	sm.performHealthCheck()
}

// RecordError records an error for stability tracking
func (sm *StabilityManager) RecordError(component, errorType, message string, severity ErrorSeverity) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	now := time.Now()

	// Create error record
	errorRecord := ErrorRecord{
		Timestamp: now,
		ErrorType: errorType,
		Message:   message,
		Component: component,
		Severity:  severity,
		Count:     1,
		Recovered: false,
	}

	// Check if this error type already exists in recent history
	for i := len(sm.errorHistory) - 1; i >= 0 && len(sm.errorHistory) > 0; i-- {
		existing := &sm.errorHistory[i]
		if existing.ErrorType == errorType &&
			existing.Component == component &&
			time.Since(existing.Timestamp) < 5*time.Minute {
			existing.Count++
			existing.Timestamp = now
			sm.updateErrorCounts(errorType, severity)
			return
		}
	}

	// Add new error record
	sm.errorHistory = append(sm.errorHistory, errorRecord)

	// Maintain history size
	if len(sm.errorHistory) > sm.maxErrorHistory {
		sm.errorHistory = sm.errorHistory[1:]
	}

	sm.updateErrorCounts(errorType, severity)
	sm.checkCircuitBreaker(component, errorType)
	sm.monitor.RecordValue(fmt.Sprintf("error_%s", severity), 1, "count")
}

// RecordRecovery records successful error recovery
func (sm *StabilityManager) RecordRecovery(component, errorType string) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	// Mark recent errors as recovered
	for i := len(sm.errorHistory) - 1; i >= 0; i-- {
		error := &sm.errorHistory[i]
		if error.Component == component && error.ErrorType == errorType &&
			!error.Recovered && time.Since(error.Timestamp) < 10*time.Minute {
			error.Recovered = true
			sm.stabilityMetrics.RecoveredErrors++
			break
		}
	}

	sm.monitor.RecordValue("error_recovery", 1, "count")
}

// CheckMemoryPressure checks if memory usage is too high
func (sm *StabilityManager) CheckMemoryPressure() bool {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	currentMemory := int64(m.HeapAlloc)
	return currentMemory > sm.memoryThreshold
}

// ForceGarbageCollection triggers garbage collection
func (sm *StabilityManager) ForceGarbageCollection() {
	if sm.forceGCEnabled {
		runtime.GC()
		debug.FreeOSMemory()
		sm.lastGC = time.Now()
		sm.monitor.RecordValue("forced_gc", 1, "count")
	}
}

// GetCircuitBreaker gets or creates a circuit breaker
func (sm *StabilityManager) GetCircuitBreaker(name string) *CircuitBreaker {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if cb, exists := sm.circuitBreakers[name]; exists {
		return cb
	}

	cb := &CircuitBreaker{
		Name:             name,
		State:            CircuitBreakerClosed,
		FailureThreshold: 5,
		ResetTimeout:     1 * time.Minute,
	}

	sm.circuitBreakers[name] = cb
	return cb
}

// ExecuteWithCircuitBreaker executes a function with circuit breaker protection
func (sm *StabilityManager) ExecuteWithCircuitBreaker(name string, operation func() error) error {
	cb := sm.GetCircuitBreaker(name)

	// Check circuit breaker state
	switch cb.State {
	case CircuitBreakerOpen:
		if time.Since(cb.LastFailure) > cb.ResetTimeout {
			cb.State = CircuitBreakerHalfOpen
		} else {
			return fmt.Errorf("circuit breaker %s is open", name)
		}
	}

	// Execute operation
	err := operation()

	sm.mu.Lock()
	defer sm.mu.Unlock()

	if err != nil {
		cb.FailureCount++
		cb.LastFailure = time.Now()

		if cb.FailureCount >= cb.FailureThreshold {
			cb.State = CircuitBreakerOpen
			sm.monitor.RecordValue("circuit_breaker_open", 1, "count")
		}

		return err
	}

	// Success - reset circuit breaker
	if cb.State == CircuitBreakerHalfOpen {
		cb.State = CircuitBreakerClosed
		cb.FailureCount = 0
		sm.monitor.RecordValue("circuit_breaker_reset", 1, "count")
	}

	return nil
}

// GetStabilityMetrics returns current stability metrics
func (sm *StabilityManager) GetStabilityMetrics() StabilityMetrics {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	metrics := sm.stabilityMetrics
	metrics.MemoryUsageMB = float64(m.HeapAlloc) / 1024 / 1024
	metrics.GoroutineCount = runtime.NumGoroutine()
	metrics.GCCount = m.NumGC
	metrics.LastGCPause = time.Duration(m.PauseNs[(m.NumGC+255)%256])
	metrics.Uptime = time.Since(sm.lastHealthCheck)

	// Calculate health score
	metrics.HealthScore = sm.calculateHealthScore(metrics)

	return metrics
}

// IsHealthy returns current health status
func (sm *StabilityManager) IsHealthy() bool {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return sm.isHealthy
}

// GetErrorHistory returns recent error history
func (sm *StabilityManager) GetErrorHistory() []ErrorRecord {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	// Return a copy to avoid race conditions
	history := make([]ErrorRecord, len(sm.errorHistory))
	copy(history, sm.errorHistory)
	return history
}

// GetCircuitBreakerStatus returns all circuit breaker statuses
func (sm *StabilityManager) GetCircuitBreakerStatus() map[string]*CircuitBreaker {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	status := make(map[string]*CircuitBreaker)
	for name, cb := range sm.circuitBreakers {
		// Return a copy
		status[name] = &CircuitBreaker{
			Name:             cb.Name,
			FailureCount:     cb.FailureCount,
			LastFailure:      cb.LastFailure,
			State:            cb.State,
			FailureThreshold: cb.FailureThreshold,
			ResetTimeout:     cb.ResetTimeout,
		}
	}

	return status
}

// monitorResources monitors system resources
func (sm *StabilityManager) monitorResources(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			sm.performHealthCheck()
		}
	}
}

// manageMemory manages memory usage and garbage collection
func (sm *StabilityManager) manageMemory(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			// Check memory pressure
			if sm.CheckMemoryPressure() {
				sm.ForceGarbageCollection()
			}

			// Periodic GC if interval has passed
			if time.Since(sm.lastGC) > sm.gcInterval {
				sm.ForceGarbageCollection()
			}
		}
	}
}

// monitorCircuitBreakers monitors and manages circuit breakers
func (sm *StabilityManager) monitorCircuitBreakers(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			sm.mu.Lock()
			openBreakers := 0
			for _, cb := range sm.circuitBreakers {
				if cb.State == CircuitBreakerOpen {
					openBreakers++

					// Try to reset if timeout has passed
					if time.Since(cb.LastFailure) > cb.ResetTimeout {
						cb.State = CircuitBreakerHalfOpen
						sm.monitor.RecordValue("circuit_breaker_half_open", 1, "count")
					}
				}
			}
			sm.stabilityMetrics.CircuitBreakersOpen = openBreakers
			sm.mu.Unlock()
		}
	}
}

// performHealthCheck performs comprehensive health check
func (sm *StabilityManager) performHealthCheck() {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	// Check memory usage
	memoryUsageMB := float64(m.HeapAlloc) / 1024 / 1024
	memoryHealthy := memoryUsageMB < float64(sm.maxMemoryMB)

	// Check goroutine count
	goroutineCount := runtime.NumGoroutine()
	goroutineHealthy := goroutineCount < sm.maxGoroutines

	// Check error rate
	recentErrors := sm.countRecentErrors(5 * time.Minute)
	errorHealthy := recentErrors < 10

	// Overall health
	sm.isHealthy = memoryHealthy && goroutineHealthy && errorHealthy
	sm.lastHealthCheck = time.Now()

	// Update metrics
	sm.stabilityMetrics.MemoryUsageMB = memoryUsageMB
	sm.stabilityMetrics.GoroutineCount = goroutineCount
	sm.stabilityMetrics.TotalErrors = len(sm.errorHistory)
	sm.stabilityMetrics.CriticalErrors = sm.countErrorsBySeverity(ErrorSeverityCritical)

	// Record health metrics
	sm.monitor.RecordValue("daemon_health_score", sm.calculateHealthScore(sm.stabilityMetrics), "score")
	sm.monitor.RecordValue("daemon_memory_mb", memoryUsageMB, "MB")
	sm.monitor.RecordValue("daemon_goroutines", float64(goroutineCount), "count")
}

// calculateHealthScore calculates overall health score (0-1)
func (sm *StabilityManager) calculateHealthScore(metrics StabilityMetrics) float64 {
	// Memory score (penalty for high memory usage)
	memoryScore := 1.0 - (metrics.MemoryUsageMB / float64(sm.maxMemoryMB))
	if memoryScore < 0 {
		memoryScore = 0
	}

	// Goroutine score (penalty for too many goroutines)
	goroutineScore := 1.0 - (float64(metrics.GoroutineCount) / float64(sm.maxGoroutines))
	if goroutineScore < 0 {
		goroutineScore = 0
	}

	// Error score (penalty for recent errors)
	recentErrors := sm.countRecentErrors(5 * time.Minute)
	errorScore := 1.0 - (float64(recentErrors) / 10.0)
	if errorScore < 0 {
		errorScore = 0
	}

	// Circuit breaker score (penalty for open circuit breakers)
	cbScore := 1.0 - (float64(metrics.CircuitBreakersOpen) / 5.0)
	if cbScore < 0 {
		cbScore = 0
	}

	// Weighted average
	score := (memoryScore*0.3 + goroutineScore*0.2 + errorScore*0.3 + cbScore*0.2)

	return score
}

// updateErrorCounts updates error count tracking
func (sm *StabilityManager) updateErrorCounts(errorType string, severity ErrorSeverity) {
	sm.errorCounts[errorType]++

	switch severity {
	case ErrorSeverityCritical:
		sm.stabilityMetrics.CriticalErrors++
	}
}

// countRecentErrors counts errors within a time window
func (sm *StabilityManager) countRecentErrors(window time.Duration) int {
	count := 0
	cutoff := time.Now().Add(-window)

	for _, error := range sm.errorHistory {
		if error.Timestamp.After(cutoff) {
			count++
		}
	}

	return count
}

// countErrorsBySeverity counts errors by severity level
func (sm *StabilityManager) countErrorsBySeverity(severity ErrorSeverity) int {
	count := 0
	for _, error := range sm.errorHistory {
		if error.Severity == severity {
			count++
		}
	}
	return count
}

// checkCircuitBreaker checks if circuit breaker should be triggered
// NOTE: This method assumes the caller already holds the lock
func (sm *StabilityManager) checkCircuitBreaker(component, errorType string) {
	cbName := fmt.Sprintf("%s_%s", component, errorType)

	// Get circuit breaker without locking (we already hold the lock)
	cb, exists := sm.circuitBreakers[cbName]
	if !exists {
		cb = &CircuitBreaker{
			Name:             cbName,
			State:            CircuitBreakerClosed,
			FailureThreshold: 5,
			ResetTimeout:     1 * time.Minute,
		}
		sm.circuitBreakers[cbName] = cb
	}

	cb.FailureCount++
	cb.LastFailure = time.Now()

	if cb.FailureCount >= cb.FailureThreshold && cb.State == CircuitBreakerClosed {
		cb.State = CircuitBreakerOpen
		sm.monitor.RecordValue("circuit_breaker_triggered", 1, "count")
	}
}

// setupPanicRecovery sets up global panic recovery
func (sm *StabilityManager) setupPanicRecovery() {
	// This would typically be called in HTTP handlers and goroutines
	// Implementation would depend on specific daemon architecture
}

// EnableForceGC enables/disables forced garbage collection
func (sm *StabilityManager) EnableForceGC(enabled bool) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.forceGCEnabled = enabled
}

// SetMemoryThreshold sets memory threshold for triggering GC
func (sm *StabilityManager) SetMemoryThreshold(bytes int64) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.memoryThreshold = bytes
}
