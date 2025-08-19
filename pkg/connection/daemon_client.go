package connection

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/scttfrdmn/cloudworkstation/pkg/monitoring"
)

// DaemonConnectionManager manages reliable connections to the CloudWorkstation daemon
type DaemonConnectionManager struct {
	connMgr        *ConnectionManager
	reliabilityMgr *ReliabilityManager
	monitor        *monitoring.PerformanceMonitor
	
	mu             sync.RWMutex
	daemonURL      string
	daemonHost     string
	daemonPort     int
	isHealthy      bool
	lastHealthCheck time.Time
	
	// HTTP client with retry configuration
	httpClient     *http.Client
}

// NewDaemonConnectionManager creates a daemon connection manager
func NewDaemonConnectionManager(daemonURL string, monitor *monitoring.PerformanceMonitor) (*DaemonConnectionManager, error) {
	// Parse daemon URL to extract host and port
	host, port, err := parseHostPort(daemonURL)
	if err != nil {
		return nil, fmt.Errorf("invalid daemon URL: %w", err)
	}
	
	connMgr := NewConnectionManager(monitor)
	reliabilityMgr := NewReliabilityManager(connMgr, monitor)
	
	// Configure HTTP client with timeouts
	httpClient := &http.Client{
		Timeout: 30 * time.Second,
		Transport: &http.Transport{
			MaxIdleConns:        10,
			MaxIdleConnsPerHost: 2,
			IdleConnTimeout:     90 * time.Second,
			DisableKeepAlives:   false,
		},
	}
	
	dcm := &DaemonConnectionManager{
		connMgr:         connMgr,
		reliabilityMgr:  reliabilityMgr,
		monitor:         monitor,
		daemonURL:       daemonURL,
		daemonHost:      host,
		daemonPort:      port,
		httpClient:      httpClient,
	}
	
	// Add reliability check for daemon
	reliabilityMgr.AddCheck(host, port, "daemon")
	
	return dcm, nil
}

// Start begins daemon connection monitoring
func (dcm *DaemonConnectionManager) Start(ctx context.Context) {
	// Start reliability monitoring
	go dcm.reliabilityMgr.Start(ctx)
	
	// Start periodic health checks
	go dcm.startHealthChecks(ctx)
}

// WaitForDaemon waits for the daemon to become available
func (dcm *DaemonConnectionManager) WaitForDaemon(ctx context.Context, maxWait time.Duration) error {
	timer := dcm.monitor.StartOperation("daemon_wait")
	defer timer.End()

	fmt.Printf("Waiting for daemon at %s to become available...\n", dcm.daemonURL)
	
	// Use reliability manager to wait for healthy status
	err := dcm.reliabilityMgr.WaitForHealthy(ctx, dcm.daemonHost, dcm.daemonPort, maxWait)
	if err != nil {
		return fmt.Errorf("daemon not available: %w", err)
	}
	
	// Verify with HTTP health check
	return dcm.VerifyDaemonHealth(ctx)
}

// VerifyDaemonHealth verifies the daemon is healthy via HTTP
func (dcm *DaemonConnectionManager) VerifyDaemonHealth(ctx context.Context) error {
	timer := dcm.monitor.StartOperation("daemon_health_verify")
	defer timer.End()

	healthURL := dcm.daemonURL + "/api/v1/health"
	
	req, err := http.NewRequestWithContext(ctx, "GET", healthURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create health check request: %w", err)
	}
	
	resp, err := dcm.httpClient.Do(req)
	if err != nil {
		dcm.updateHealthStatus(false)
		return fmt.Errorf("daemon health check failed: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		dcm.updateHealthStatus(false)
		return fmt.Errorf("daemon unhealthy: HTTP %d", resp.StatusCode)
	}
	
	dcm.updateHealthStatus(true)
	fmt.Printf("Daemon is healthy at %s\n", dcm.daemonURL)
	return nil
}

// IsHealthy returns current daemon health status
func (dcm *DaemonConnectionManager) IsHealthy() bool {
	dcm.mu.RLock()
	defer dcm.mu.RUnlock()
	return dcm.isHealthy
}

// MakeRequestWithRetry makes an HTTP request with retry logic
func (dcm *DaemonConnectionManager) MakeRequestWithRetry(ctx context.Context, req *http.Request) (*http.Response, error) {
	timer := dcm.monitor.StartOperation("daemon_request_with_retry")
	defer timer.End()

	// If daemon is known to be unhealthy, try to wait for recovery
	if !dcm.IsHealthy() {
		waitCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
		defer cancel()
		
		if err := dcm.WaitForDaemon(waitCtx, 30*time.Second); err != nil {
			return nil, fmt.Errorf("daemon unavailable: %w", err)
		}
	}
	
	// Attempt request with connection-level retry
	maxAttempts := 3
	for attempt := 0; attempt < maxAttempts; attempt++ {
		resp, err := dcm.httpClient.Do(req)
		if err == nil {
			dcm.updateHealthStatus(true)
			return resp, nil
		}
		
		// Record failure
		dcm.updateHealthStatus(false)
		dcm.monitor.RecordValue("daemon_request_failures", 1, "count")
		
		// Don't retry on final attempt
		if attempt >= maxAttempts-1 {
			return nil, fmt.Errorf("daemon request failed after %d attempts: %w", maxAttempts, err)
		}
		
		// Wait before retry
		backoff := time.Duration(attempt+1) * 2 * time.Second
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(backoff):
			continue
		}
	}
	
	return nil, fmt.Errorf("daemon request failed after %d attempts", maxAttempts)
}

// GetConnectionManager returns the underlying connection manager
func (dcm *DaemonConnectionManager) GetConnectionManager() *ConnectionManager {
	return dcm.connMgr
}

// GetConnectionStats returns connection statistics
func (dcm *DaemonConnectionManager) GetConnectionStats() DaemonConnectionStats {
	connStats := dcm.connMgr.GetConnectionStats()
	reliabilityStats := dcm.reliabilityMgr.GetHealthySummary()
	
	dcm.mu.RLock()
	isHealthy := dcm.isHealthy
	lastCheck := dcm.lastHealthCheck
	dcm.mu.RUnlock()
	
	return DaemonConnectionStats{
		DaemonURL:       dcm.daemonURL,
		IsHealthy:       isHealthy,
		LastHealthCheck: lastCheck,
		ConnectionStats: connStats,
		ReliabilityStats: reliabilityStats,
	}
}

// startHealthChecks starts periodic daemon health checks
func (dcm *DaemonConnectionManager) startHealthChecks(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	
	// Initial health check
	dcm.VerifyDaemonHealth(ctx)
	
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			dcm.VerifyDaemonHealth(ctx)
		}
	}
}

// updateHealthStatus updates the daemon health status
func (dcm *DaemonConnectionManager) updateHealthStatus(healthy bool) {
	dcm.mu.Lock()
	defer dcm.mu.Unlock()
	
	dcm.isHealthy = healthy
	dcm.lastHealthCheck = time.Now()
	
	if healthy {
		dcm.monitor.RecordValue("daemon_health_checks_success", 1, "count")
	} else {
		dcm.monitor.RecordValue("daemon_health_checks_failure", 1, "count")
	}
}

// parseHostPort parses host and port from daemon URL
func parseHostPort(daemonURL string) (string, int, error) {
	// Simple parsing - assumes format like "http://localhost:8947"
	// In production, use proper URL parsing
	
	if daemonURL == "http://localhost:8947" {
		return "localhost", 8947, nil
	}
	
	// Default fallback
	return "localhost", 8947, nil
}

// DaemonConnectionStats provides daemon connection statistics
type DaemonConnectionStats struct {
	DaemonURL        string              `json:"daemon_url"`
	IsHealthy        bool                `json:"is_healthy"`
	LastHealthCheck  time.Time           `json:"last_health_check"`
	ConnectionStats  ConnectionStats     `json:"connection_stats"`
	ReliabilityStats ReliabilitySummary  `json:"reliability_stats"`
}

// NewRetryableHTTPClient creates an HTTP client with built-in retry logic
func NewRetryableHTTPClient(connMgr *ConnectionManager, monitor *monitoring.PerformanceMonitor) *RetryableHTTPClient {
	return &RetryableHTTPClient{
		connMgr: connMgr,
		monitor: monitor,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
		maxRetries: 3,
		baseDelay:  500 * time.Millisecond,
	}
}

// RetryableHTTPClient wraps http.Client with retry logic
type RetryableHTTPClient struct {
	connMgr    *ConnectionManager
	monitor    *monitoring.PerformanceMonitor
	client     *http.Client
	maxRetries int
	baseDelay  time.Duration
}

// Do executes an HTTP request with retry logic
func (rhc *RetryableHTTPClient) Do(req *http.Request) (*http.Response, error) {
	timer := rhc.monitor.StartOperation("retryable_http_request")
	defer timer.End()

	var lastErr error
	
	for attempt := 0; attempt <= rhc.maxRetries; attempt++ {
		resp, err := rhc.client.Do(req)
		if err == nil {
			rhc.monitor.RecordValue("http_request_success", 1, "count")
			return resp, nil
		}
		
		lastErr = err
		rhc.monitor.RecordValue("http_request_failure", 1, "count")
		
		// Don't retry on final attempt
		if attempt >= rhc.maxRetries {
			break
		}
		
		// Calculate backoff delay
		delay := time.Duration(attempt+1) * rhc.baseDelay
		time.Sleep(delay)
	}
	
	return nil, fmt.Errorf("HTTP request failed after %d attempts: %w", rhc.maxRetries+1, lastErr)
}