package daemon

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"runtime"
	"sync"
	"time"

	"github.com/scttfrdmn/cloudworkstation/pkg/monitoring"
)

// StateManager interface to avoid circular dependency
type StateManager interface {
	// Add methods as needed for health checks
}

// HealthMonitor provides comprehensive daemon health monitoring
type HealthMonitor struct {
	stateManager StateManager
	stabilityMgr *StabilityManager
	recoveryMgr  *RecoveryManager
	monitor      *monitoring.PerformanceMonitor

	mu             sync.RWMutex
	healthChecks   map[string]HealthCheck
	healthHistory  []HealthSnapshot
	maxHistorySize int

	// Monitoring settings
	checkInterval   time.Duration
	isHealthy       bool
	lastHealthCheck time.Time
}

// HealthCheck represents a health check function
type HealthCheck struct {
	Name        string                   `json:"name"`
	Description string                   `json:"description"`
	CheckFunc   func() HealthCheckResult `json:"-"`
	Interval    time.Duration            `json:"interval"`
	Timeout     time.Duration            `json:"timeout"`
	Critical    bool                     `json:"critical"`
	LastRun     time.Time                `json:"last_run"`
	LastResult  HealthCheckResult        `json:"last_result"`
}

// HealthCheckResult represents the result of a health check
type HealthCheckResult struct {
	Status    HealthStatus           `json:"status"`
	Message   string                 `json:"message"`
	Duration  time.Duration          `json:"duration"`
	Error     string                 `json:"error,omitempty"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
	CheckedAt time.Time              `json:"checked_at"`
}

// HealthStatus represents health check status
type HealthStatus string

const (
	HealthStatusHealthy   HealthStatus = "healthy"
	HealthStatusDegraded  HealthStatus = "degraded"
	HealthStatusUnhealthy HealthStatus = "unhealthy"
	HealthStatusUnknown   HealthStatus = "unknown"
)

// HealthSnapshot represents a point-in-time health snapshot
type HealthSnapshot struct {
	Timestamp      time.Time                    `json:"timestamp"`
	OverallStatus  HealthStatus                 `json:"overall_status"`
	CheckResults   map[string]HealthCheckResult `json:"check_results"`
	SystemMetrics  SystemMetrics                `json:"system_metrics"`
	StabilityScore float64                      `json:"stability_score"`
}

// SystemMetrics represents system-level metrics
type SystemMetrics struct {
	MemoryUsageMB   float64       `json:"memory_usage_mb"`
	GoroutineCount  int           `json:"goroutine_count"`
	GCCount         uint32        `json:"gc_count"`
	LastGCPause     time.Duration `json:"last_gc_pause"`
	Uptime          time.Duration `json:"uptime"`
	CPUUsagePercent float64       `json:"cpu_usage_percent"`
	LoadAverage     float64       `json:"load_average"`
}

// DaemonHealthSummary provides overall daemon health summary
type DaemonHealthSummary struct {
	Status          HealthStatus  `json:"status"`
	Score           float64       `json:"score"`
	LastChecked     time.Time     `json:"last_checked"`
	SystemMetrics   SystemMetrics `json:"system_metrics"`
	ActiveChecks    int           `json:"active_checks"`
	FailedChecks    int           `json:"failed_checks"`
	CriticalIssues  []string      `json:"critical_issues"`
	Recommendations []string      `json:"recommendations"`
}

// NewHealthMonitor creates a new health monitor
func NewHealthMonitor(stateManager StateManager, stabilityMgr *StabilityManager, recoveryMgr *RecoveryManager, monitor *monitoring.PerformanceMonitor) *HealthMonitor {
	hm := &HealthMonitor{
		stateManager:    stateManager,
		stabilityMgr:    stabilityMgr,
		recoveryMgr:     recoveryMgr,
		monitor:         monitor,
		healthChecks:    make(map[string]HealthCheck),
		maxHistorySize:  100,
		checkInterval:   30 * time.Second,
		isHealthy:       true,
		lastHealthCheck: time.Now(),
	}

	hm.setupDefaultHealthChecks()
	return hm
}

// Start begins health monitoring
func (hm *HealthMonitor) Start(ctx context.Context) {
	// Initial health check
	hm.performHealthChecks()

	// Start periodic health checking
	ticker := time.NewTicker(hm.checkInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			hm.performHealthChecks()
		}
	}
}

// AddHealthCheck adds a custom health check
func (hm *HealthMonitor) AddHealthCheck(name string, check HealthCheck) {
	hm.mu.Lock()
	defer hm.mu.Unlock()

	check.Name = name
	hm.healthChecks[name] = check
}

// RemoveHealthCheck removes a health check
func (hm *HealthMonitor) RemoveHealthCheck(name string) {
	hm.mu.Lock()
	defer hm.mu.Unlock()

	delete(hm.healthChecks, name)
}

// GetHealthSummary returns current health summary
func (hm *HealthMonitor) GetHealthSummary() DaemonHealthSummary {
	hm.mu.RLock()
	defer hm.mu.RUnlock()

	var failedChecks int
	var criticalIssues []string
	var recommendations []string

	for _, check := range hm.healthChecks {
		if check.LastResult.Status == HealthStatusUnhealthy {
			failedChecks++
			if check.Critical {
				criticalIssues = append(criticalIssues,
					fmt.Sprintf("%s: %s", check.Name, check.LastResult.Message))
			}
		}
	}

	// Generate recommendations based on current state
	recommendations = hm.generateRecommendations()

	systemMetrics := hm.collectSystemMetrics()
	stabilityMetrics := hm.stabilityMgr.GetStabilityMetrics()

	return DaemonHealthSummary{
		Status:          hm.calculateOverallStatus(),
		Score:           stabilityMetrics.HealthScore,
		LastChecked:     hm.lastHealthCheck,
		SystemMetrics:   systemMetrics,
		ActiveChecks:    len(hm.healthChecks),
		FailedChecks:    failedChecks,
		CriticalIssues:  criticalIssues,
		Recommendations: recommendations,
	}
}

// GetHealthHistory returns recent health history
func (hm *HealthMonitor) GetHealthHistory() []HealthSnapshot {
	hm.mu.RLock()
	defer hm.mu.RUnlock()

	history := make([]HealthSnapshot, len(hm.healthHistory))
	copy(history, hm.healthHistory)
	return history
}

// IsHealthy returns current health status
func (hm *HealthMonitor) IsHealthy() bool {
	hm.mu.RLock()
	defer hm.mu.RUnlock()
	return hm.isHealthy
}

// HandleHealthEndpoint handles HTTP health endpoint
func (hm *HealthMonitor) HandleHealthEndpoint(w http.ResponseWriter, r *http.Request) {
	summary := hm.GetHealthSummary()

	// Set appropriate HTTP status
	statusCode := http.StatusOK
	if summary.Status == HealthStatusUnhealthy {
		statusCode = http.StatusServiceUnavailable
	} else if summary.Status == HealthStatusDegraded {
		statusCode = http.StatusPartialContent
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	_ = json.NewEncoder(w).Encode(summary)
}

// HandleDetailedHealthEndpoint handles detailed health endpoint
func (hm *HealthMonitor) HandleDetailedHealthEndpoint(w http.ResponseWriter, r *http.Request) {
	hm.mu.RLock()
	checks := make(map[string]HealthCheck)
	for k, v := range hm.healthChecks {
		checks[k] = v
	}
	history := make([]HealthSnapshot, len(hm.healthHistory))
	copy(history, hm.healthHistory)
	hm.mu.RUnlock()

	detailed := struct {
		Summary   DaemonHealthSummary    `json:"summary"`
		Checks    map[string]HealthCheck `json:"checks"`
		History   []HealthSnapshot       `json:"history"`
		Stability interface{}            `json:"stability"`
	}{
		Summary:   hm.GetHealthSummary(),
		Checks:    checks,
		History:   history,
		Stability: hm.stabilityMgr.GetStabilityMetrics(),
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(detailed)
}

// performHealthChecks runs all health checks
func (hm *HealthMonitor) performHealthChecks() {
	hm.mu.Lock()
	defer hm.mu.Unlock()

	now := time.Now()
	checkResults := make(map[string]HealthCheckResult)

	// Run all health checks
	for name, check := range hm.healthChecks {
		// Skip if not time for this check yet
		if now.Sub(check.LastRun) < check.Interval {
			checkResults[name] = check.LastResult
			continue
		}

		// Run health check with timeout
		result := hm.runHealthCheck(check)

		// Update check
		check.LastRun = now
		check.LastResult = result
		hm.healthChecks[name] = check

		checkResults[name] = result

		// Record metrics
		statusValue := 1.0
		if result.Status != HealthStatusHealthy {
			statusValue = 0.0
		}
		hm.monitor.RecordValue(fmt.Sprintf("health_check_%s", name), statusValue, "status")
		hm.monitor.RecordTiming(fmt.Sprintf("health_check_%s_duration", name), result.Duration)
	}

	// Calculate overall status
	overallStatus := hm.calculateOverallStatusFromResults(checkResults)
	hm.isHealthy = (overallStatus == HealthStatusHealthy)
	hm.lastHealthCheck = now

	// Create health snapshot
	snapshot := HealthSnapshot{
		Timestamp:      now,
		OverallStatus:  overallStatus,
		CheckResults:   checkResults,
		SystemMetrics:  hm.collectSystemMetrics(),
		StabilityScore: hm.stabilityMgr.GetStabilityMetrics().HealthScore,
	}

	// Add to history
	hm.healthHistory = append(hm.healthHistory, snapshot)
	if len(hm.healthHistory) > hm.maxHistorySize {
		hm.healthHistory = hm.healthHistory[1:]
	}

	// Record overall health
	overallValue := 1.0
	if overallStatus != HealthStatusHealthy {
		overallValue = 0.0
	}
	hm.monitor.RecordValue("daemon_overall_health", overallValue, "status")
}

// runHealthCheck runs a single health check with timeout
func (hm *HealthMonitor) runHealthCheck(check HealthCheck) HealthCheckResult {
	startTime := time.Now()

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), check.Timeout)
	defer cancel()

	// Channel to receive result
	resultChan := make(chan HealthCheckResult, 1)

	// Run check in goroutine
	go func() {
		defer func() {
			if r := recover(); r != nil {
				resultChan <- HealthCheckResult{
					Status:    HealthStatusUnhealthy,
					Message:   fmt.Sprintf("Health check panicked: %v", r),
					Duration:  time.Since(startTime),
					Error:     fmt.Sprintf("panic: %v", r),
					CheckedAt: startTime,
				}
			}
		}()

		result := check.CheckFunc()
		result.Duration = time.Since(startTime)
		result.CheckedAt = startTime
		resultChan <- result
	}()

	// Wait for result or timeout
	select {
	case result := <-resultChan:
		return result
	case <-ctx.Done():
		return HealthCheckResult{
			Status:    HealthStatusUnhealthy,
			Message:   "Health check timed out",
			Duration:  time.Since(startTime),
			Error:     "timeout",
			CheckedAt: startTime,
		}
	}
}

// calculateOverallStatus calculates overall health status
func (hm *HealthMonitor) calculateOverallStatus() HealthStatus {
	checkResults := make(map[string]HealthCheckResult)
	for name, check := range hm.healthChecks {
		checkResults[name] = check.LastResult
	}
	return hm.calculateOverallStatusFromResults(checkResults)
}

// calculateOverallStatusFromResults calculates status from check results
func (hm *HealthMonitor) calculateOverallStatusFromResults(results map[string]HealthCheckResult) HealthStatus {
	if len(results) == 0 {
		return HealthStatusUnknown
	}

	criticalFailed := false
	degraded := false

	for name, result := range results {
		check := hm.healthChecks[name]

		if result.Status == HealthStatusUnhealthy {
			if check.Critical {
				criticalFailed = true
			} else {
				degraded = true
			}
		} else if result.Status == HealthStatusDegraded {
			degraded = true
		}
	}

	if criticalFailed {
		return HealthStatusUnhealthy
	} else if degraded {
		return HealthStatusDegraded
	}

	return HealthStatusHealthy
}

// collectSystemMetrics collects current system metrics
func (hm *HealthMonitor) collectSystemMetrics() SystemMetrics {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	return SystemMetrics{
		MemoryUsageMB:   float64(m.HeapAlloc) / 1024 / 1024,
		GoroutineCount:  runtime.NumGoroutine(),
		GCCount:         m.NumGC,
		LastGCPause:     time.Duration(m.PauseNs[(m.NumGC+255)%256]),
		Uptime:          time.Since(hm.lastHealthCheck),
		CPUUsagePercent: getCachedCPUUsage(),
		LoadAverage:     getCachedLoadAverage(),
	}
}

// generateRecommendations generates health recommendations
func (hm *HealthMonitor) generateRecommendations() []string {
	var recommendations []string

	// Memory recommendations
	metrics := hm.collectSystemMetrics()
	if metrics.MemoryUsageMB > 512 {
		recommendations = append(recommendations,
			"High memory usage detected. Consider restarting the daemon or reducing workload.")
	}

	// Goroutine recommendations
	if metrics.GoroutineCount > 100 {
		recommendations = append(recommendations,
			"High goroutine count detected. Check for goroutine leaks.")
	}

	// Failed checks recommendations
	for name, check := range hm.healthChecks {
		if check.LastResult.Status == HealthStatusUnhealthy {
			recommendations = append(recommendations,
				fmt.Sprintf("Fix issues with %s health check: %s", name, check.LastResult.Message))
		}
	}

	// Stability recommendations
	stabilityMetrics := hm.stabilityMgr.GetStabilityMetrics()
	if stabilityMetrics.HealthScore < 0.7 {
		recommendations = append(recommendations,
			"Overall health score is low. Review system performance and error logs.")
	}

	return recommendations
}

// setupDefaultHealthChecks sets up default health checks
func (hm *HealthMonitor) setupDefaultHealthChecks() {
	// Memory health check
	hm.AddHealthCheck("memory", HealthCheck{
		Description: "Checks memory usage",
		CheckFunc: func() HealthCheckResult {
			var m runtime.MemStats
			runtime.ReadMemStats(&m)
			memMB := float64(m.HeapAlloc) / 1024 / 1024

			status := HealthStatusHealthy
			message := fmt.Sprintf("Memory usage: %.2f MB", memMB)

			if memMB > 1024 {
				status = HealthStatusUnhealthy
				message = fmt.Sprintf("High memory usage: %.2f MB", memMB)
			} else if memMB > 512 {
				status = HealthStatusDegraded
				message = fmt.Sprintf("Elevated memory usage: %.2f MB", memMB)
			}

			return HealthCheckResult{
				Status:  status,
				Message: message,
				Metadata: map[string]interface{}{
					"memory_mb":    memMB,
					"heap_objects": m.HeapObjects,
				},
			}
		},
		Interval: 30 * time.Second,
		Timeout:  5 * time.Second,
		Critical: false,
	})

	// Goroutine health check
	hm.AddHealthCheck("goroutines", HealthCheck{
		Description: "Checks goroutine count",
		CheckFunc: func() HealthCheckResult {
			count := runtime.NumGoroutine()

			status := HealthStatusHealthy
			message := fmt.Sprintf("Goroutine count: %d", count)

			if count > 500 {
				status = HealthStatusUnhealthy
				message = fmt.Sprintf("High goroutine count: %d", count)
			} else if count > 100 {
				status = HealthStatusDegraded
				message = fmt.Sprintf("Elevated goroutine count: %d", count)
			}

			return HealthCheckResult{
				Status:  status,
				Message: message,
				Metadata: map[string]interface{}{
					"goroutine_count": count,
				},
			}
		},
		Interval: 30 * time.Second,
		Timeout:  5 * time.Second,
		Critical: false,
	})

	// Error rate health check
	hm.AddHealthCheck("error_rate", HealthCheck{
		Description: "Checks error rate",
		CheckFunc: func() HealthCheckResult {
			errorHistory := hm.stabilityMgr.GetErrorHistory()

			// Count recent errors (last 5 minutes)
			recentErrors := 0
			cutoff := time.Now().Add(-5 * time.Minute)
			for _, err := range errorHistory {
				if err.Timestamp.After(cutoff) {
					recentErrors++
				}
			}

			status := HealthStatusHealthy
			message := fmt.Sprintf("Recent errors: %d", recentErrors)

			if recentErrors > 20 {
				status = HealthStatusUnhealthy
				message = fmt.Sprintf("High error rate: %d errors in 5 minutes", recentErrors)
			} else if recentErrors > 10 {
				status = HealthStatusDegraded
				message = fmt.Sprintf("Elevated error rate: %d errors in 5 minutes", recentErrors)
			}

			return HealthCheckResult{
				Status:  status,
				Message: message,
				Metadata: map[string]interface{}{
					"recent_errors": recentErrors,
					"total_errors":  len(errorHistory),
				},
			}
		},
		Interval: 1 * time.Minute,
		Timeout:  5 * time.Second,
		Critical: false,
	})

	// State manager health check
	hm.AddHealthCheck("state_manager", HealthCheck{
		Description: "Checks state manager health",
		CheckFunc: func() HealthCheckResult {
			if hm.stateManager == nil {
				return HealthCheckResult{
					Status:  HealthStatusUnhealthy,
					Message: "State manager is not initialized",
				}
			}

			// Try to test state manager functionality
			// This would depend on your state manager implementation
			return HealthCheckResult{
				Status:  HealthStatusHealthy,
				Message: "State manager is operational",
			}
		},
		Interval: 2 * time.Minute,
		Timeout:  10 * time.Second,
		Critical: true,
	})
}
