package connection

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/scttfrdmn/cloudworkstation/pkg/monitoring"
)

// ReliabilityManager manages connection reliability and automatic recovery
type ReliabilityManager struct {
	connMgr *ConnectionManager
	monitor *monitoring.PerformanceMonitor

	mu     sync.RWMutex
	checks map[string]*ReliabilityCheck

	// Configuration
	checkInterval      time.Duration
	unhealthyThreshold int
	recoveryThreshold  int
	enabled            bool
}

// ReliabilityCheck tracks reliability for a specific target
type ReliabilityCheck struct {
	Target               string            `json:"target"`
	Port                 int               `json:"port"`
	Service              string            `json:"service"`
	Status               ReliabilityStatus `json:"status"`
	ConsecutiveFailures  int               `json:"consecutive_failures"`
	ConsecutiveSuccesses int               `json:"consecutive_successes"`
	TotalChecks          int               `json:"total_checks"`
	SuccessRate          float64           `json:"success_rate"`
	LastCheck            time.Time         `json:"last_check"`
	LastSuccess          time.Time         `json:"last_success"`
	LastFailure          time.Time         `json:"last_failure"`
	ErrorMessage         string            `json:"error_message,omitempty"`
}

// ReliabilityStatus represents the reliability status
type ReliabilityStatus string

const (
	ReliabilityStatusHealthy    ReliabilityStatus = "healthy"
	ReliabilityStatusDegraded   ReliabilityStatus = "degraded"
	ReliabilityStatusUnhealthy  ReliabilityStatus = "unhealthy"
	ReliabilityStatusRecovering ReliabilityStatus = "recovering"
)

// NewReliabilityManager creates a new reliability manager
func NewReliabilityManager(connMgr *ConnectionManager, monitor *monitoring.PerformanceMonitor) *ReliabilityManager {
	return &ReliabilityManager{
		connMgr:            connMgr,
		monitor:            monitor,
		checks:             make(map[string]*ReliabilityCheck),
		checkInterval:      30 * time.Second,
		unhealthyThreshold: 3,
		recoveryThreshold:  2,
		enabled:            true,
	}
}

// Start begins reliability monitoring
func (rm *ReliabilityManager) Start(ctx context.Context) {
	if !rm.enabled {
		return
	}

	ticker := time.NewTicker(rm.checkInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			rm.performReliabilityChecks(ctx)
		}
	}
}

// AddCheck adds a reliability check for a target
func (rm *ReliabilityManager) AddCheck(target string, port int, service string) {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	checkKey := fmt.Sprintf("%s:%d", target, port)
	rm.checks[checkKey] = &ReliabilityCheck{
		Target:  target,
		Port:    port,
		Service: service,
		Status:  ReliabilityStatusHealthy,
	}
}

// RemoveCheck removes a reliability check
func (rm *ReliabilityManager) RemoveCheck(target string, port int) {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	checkKey := fmt.Sprintf("%s:%d", target, port)
	delete(rm.checks, checkKey)
}

// GetReliabilityStatus gets the status for a specific target
func (rm *ReliabilityManager) GetReliabilityStatus(target string, port int) (*ReliabilityCheck, bool) {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	checkKey := fmt.Sprintf("%s:%d", target, port)
	check, exists := rm.checks[checkKey]
	return check, exists
}

// GetAllChecks returns all reliability checks
func (rm *ReliabilityManager) GetAllChecks() map[string]*ReliabilityCheck {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	// Return a copy to avoid race conditions
	result := make(map[string]*ReliabilityCheck)
	for k, v := range rm.checks {
		result[k] = v
	}
	return result
}

// WaitForHealthy waits for a target to become healthy
func (rm *ReliabilityManager) WaitForHealthy(ctx context.Context, target string, port int, maxWait time.Duration) error {
	timer := rm.monitor.StartOperation("wait_for_healthy")
	defer timer.End()

	timeout := time.NewTimer(maxWait)
	defer timeout.Stop()

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	// Add check if it doesn't exist
	checkKey := fmt.Sprintf("%s:%d", target, port)
	if _, exists := rm.GetReliabilityStatus(target, port); !exists {
		rm.AddCheck(target, port, "generic")
	}

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("cancelled while waiting for %s:%d to become healthy: %w", target, port, ctx.Err())

		case <-timeout.C:
			return fmt.Errorf("timeout waiting for %s:%d to become healthy after %v", target, port, maxWait)

		case <-ticker.C:
			// Perform immediate check
			rm.performSingleReliabilityCheck(ctx, checkKey)

			// Check status
			if check, exists := rm.GetReliabilityStatus(target, port); exists {
				if check.Status == ReliabilityStatusHealthy {
					return nil
				}
			}
		}
	}
}

// IsHealthy checks if a target is currently healthy
func (rm *ReliabilityManager) IsHealthy(target string, port int) bool {
	if check, exists := rm.GetReliabilityStatus(target, port); exists {
		return check.Status == ReliabilityStatusHealthy
	}
	return false
}

// GetHealthySummary returns a summary of healthy/unhealthy services
func (rm *ReliabilityManager) GetHealthySummary() ReliabilitySummary {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	summary := ReliabilitySummary{
		StatusCounts: make(map[ReliabilityStatus]int),
	}

	for _, check := range rm.checks {
		summary.TotalChecks++
		summary.StatusCounts[check.Status]++

		if check.Status == ReliabilityStatusHealthy {
			summary.HealthyChecks++
		}
	}

	if summary.TotalChecks > 0 {
		summary.OverallHealthiness = float64(summary.HealthyChecks) / float64(summary.TotalChecks)
	}

	return summary
}

// performReliabilityChecks performs all configured reliability checks
func (rm *ReliabilityManager) performReliabilityChecks(ctx context.Context) {
	rm.mu.RLock()
	checks := make([]string, 0, len(rm.checks))
	for key := range rm.checks {
		checks = append(checks, key)
	}
	rm.mu.RUnlock()

	// Perform checks in parallel
	var wg sync.WaitGroup
	for _, checkKey := range checks {
		wg.Add(1)
		go func(key string) {
			defer wg.Done()
			rm.performSingleReliabilityCheck(ctx, key)
		}(checkKey)
	}
	wg.Wait()
}

// performSingleReliabilityCheck performs a single reliability check
func (rm *ReliabilityManager) performSingleReliabilityCheck(ctx context.Context, checkKey string) {
	rm.mu.RLock()
	check, exists := rm.checks[checkKey]
	if !exists {
		rm.mu.RUnlock()
		return
	}
	// Create a copy to avoid holding the lock during the actual check
	checkCopy := *check
	rm.mu.RUnlock()

	// Perform the actual check
	err := rm.connMgr.TestPortAvailability(ctx, checkCopy.Target, checkCopy.Port, 10*time.Second)

	// Update check results
	rm.mu.Lock()
	defer rm.mu.Unlock()

	// Re-get the check in case it was deleted
	check, exists = rm.checks[checkKey]
	if !exists {
		return
	}

	now := time.Now()
	check.LastCheck = now
	check.TotalChecks++

	if err == nil {
		// Success
		check.ConsecutiveFailures = 0
		check.ConsecutiveSuccesses++
		check.LastSuccess = now
		check.ErrorMessage = ""

		// Update status based on consecutive successes
		if check.Status == ReliabilityStatusUnhealthy && check.ConsecutiveSuccesses >= rm.recoveryThreshold {
			check.Status = ReliabilityStatusRecovering
		} else if check.Status == ReliabilityStatusRecovering && check.ConsecutiveSuccesses >= rm.recoveryThreshold*2 {
			check.Status = ReliabilityStatusHealthy
		} else if check.Status != ReliabilityStatusUnhealthy && check.Status != ReliabilityStatusRecovering {
			check.Status = ReliabilityStatusHealthy
		}

		rm.monitor.RecordValue("reliability_check_success", 1, "count")
	} else {
		// Failure
		check.ConsecutiveSuccesses = 0
		check.ConsecutiveFailures++
		check.LastFailure = now
		check.ErrorMessage = err.Error()

		// Update status based on consecutive failures
		if check.ConsecutiveFailures >= rm.unhealthyThreshold {
			check.Status = ReliabilityStatusUnhealthy
		} else if check.ConsecutiveFailures > 1 {
			check.Status = ReliabilityStatusDegraded
		}

		rm.monitor.RecordValue("reliability_check_failure", 1, "count")
	}

	// Calculate success rate (last 100 checks)
	if check.TotalChecks > 0 {
		successCount := check.TotalChecks - check.ConsecutiveFailures
		if successCount < 0 {
			successCount = 0
		}

		// For simplicity, using current consecutive failures
		// In production, you'd maintain a sliding window
		recent := check.TotalChecks
		if recent > 100 {
			recent = 100
		}

		recentSuccesses := recent - check.ConsecutiveFailures
		if recentSuccesses < 0 {
			recentSuccesses = 0
		}

		check.SuccessRate = float64(recentSuccesses) / float64(recent)
	}
}

// HTTPReliabilityChecker provides HTTP-specific reliability checking
type HTTPReliabilityChecker struct {
	client *http.Client
	rm     *ReliabilityManager
}

// NewHTTPReliabilityChecker creates an HTTP reliability checker
func NewHTTPReliabilityChecker(rm *ReliabilityManager) *HTTPReliabilityChecker {
	return &HTTPReliabilityChecker{
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
		rm: rm,
	}
}

// CheckHTTPEndpoint checks an HTTP endpoint for reliability
func (hrc *HTTPReliabilityChecker) CheckHTTPEndpoint(ctx context.Context, url string) (*HealthResult, error) {
	timer := hrc.rm.monitor.StartOperation("http_endpoint_check")
	defer timer.End()

	startTime := time.Now()

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return &HealthResult{
			Service:   "http",
			Status:    HealthStatusUnhealthy,
			Error:     fmt.Sprintf("failed to create request: %v", err),
			CheckedAt: startTime,
			Duration:  time.Since(startTime),
		}, err
	}

	resp, err := hrc.client.Do(req)
	if err != nil {
		return &HealthResult{
			Service:   "http",
			Status:    HealthStatusUnhealthy,
			Error:     fmt.Sprintf("request failed: %v", err),
			CheckedAt: startTime,
			Duration:  time.Since(startTime),
		}, err
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	// Consider 2xx and 3xx as healthy
	if resp.StatusCode >= 200 && resp.StatusCode < 400 {
		return &HealthResult{
			Service:   "http",
			Status:    HealthStatusHealthy,
			CheckedAt: startTime,
			Duration:  time.Since(startTime),
		}, nil
	}

	return &HealthResult{
		Service:   "http",
		Status:    HealthStatusUnhealthy,
		Error:     fmt.Sprintf("HTTP %d", resp.StatusCode),
		CheckedAt: startTime,
		Duration:  time.Since(startTime),
	}, fmt.Errorf("HTTP %d", resp.StatusCode)
}

// ReliabilitySummary provides an overview of reliability status
type ReliabilitySummary struct {
	TotalChecks        int                       `json:"total_checks"`
	HealthyChecks      int                       `json:"healthy_checks"`
	OverallHealthiness float64                   `json:"overall_healthiness"`
	StatusCounts       map[ReliabilityStatus]int `json:"status_counts"`
}
