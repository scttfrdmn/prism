package connection

import (
	"context"
	"fmt"
	"math"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/scttfrdmn/cloudworkstation/pkg/monitoring"
)

// ConnectionManager manages reliable connections with retry logic
type ConnectionManager struct {
	monitor     *monitoring.PerformanceMonitor
	mu          sync.RWMutex
	connections map[string]*ConnectionState

	// Configuration
	maxRetries int
	baseDelay  time.Duration
	maxDelay   time.Duration
	multiplier float64
	jitter     bool
}

// ConnectionState tracks the state of a connection
type ConnectionState struct {
	Target      string           `json:"target"`
	Status      ConnectionStatus `json:"status"`
	Attempts    int              `json:"attempts"`
	LastAttempt time.Time        `json:"last_attempt"`
	LastError   string           `json:"last_error,omitempty"`
	NextRetry   time.Time        `json:"next_retry,omitempty"`
}

// ConnectionStatus represents connection status
type ConnectionStatus string

const (
	ConnectionPending    ConnectionStatus = "pending"
	ConnectionConnecting ConnectionStatus = "connecting"
	ConnectionConnected  ConnectionStatus = "connected"
	ConnectionFailed     ConnectionStatus = "failed"
	ConnectionTimeout    ConnectionStatus = "timeout"
)

// RetryPolicy defines retry behavior
type RetryPolicy struct {
	MaxRetries int           `json:"max_retries"`
	BaseDelay  time.Duration `json:"base_delay"`
	MaxDelay   time.Duration `json:"max_delay"`
	Multiplier float64       `json:"multiplier"`
	Jitter     bool          `json:"jitter"`
}

// DefaultRetryPolicy returns a sensible default retry policy
func DefaultRetryPolicy() RetryPolicy {
	return RetryPolicy{
		MaxRetries: 10,
		BaseDelay:  500 * time.Millisecond,
		MaxDelay:   30 * time.Second,
		Multiplier: 2.0,
		Jitter:     true,
	}
}

// NewConnectionManager creates a new connection manager
func NewConnectionManager(monitor *monitoring.PerformanceMonitor) *ConnectionManager {
	policy := DefaultRetryPolicy()

	return &ConnectionManager{
		monitor:     monitor,
		connections: make(map[string]*ConnectionState),
		maxRetries:  policy.MaxRetries,
		baseDelay:   policy.BaseDelay,
		maxDelay:    policy.MaxDelay,
		multiplier:  policy.Multiplier,
		jitter:      policy.Jitter,
	}
}

// NewConnectionManagerWithPolicy creates a connection manager with custom policy
func NewConnectionManagerWithPolicy(monitor *monitoring.PerformanceMonitor, policy RetryPolicy) *ConnectionManager {
	return &ConnectionManager{
		monitor:     monitor,
		connections: make(map[string]*ConnectionState),
		maxRetries:  policy.MaxRetries,
		baseDelay:   policy.BaseDelay,
		maxDelay:    policy.MaxDelay,
		multiplier:  policy.Multiplier,
		jitter:      policy.Jitter,
	}
}

// ConnectWithRetry attempts to connect with exponential backoff retry
func (cm *ConnectionManager) ConnectWithRetry(ctx context.Context, target string, port int) (*ConnectionResult, error) {
	timer := cm.monitor.StartOperation("connection_attempt")
	defer func() { _ = timer.End() }()

	address := fmt.Sprintf("%s:%d", target, port)

	// Initialize connection state
	cm.initConnectionState(address)

	for attempt := 0; attempt <= cm.maxRetries; attempt++ {
		select {
		case <-ctx.Done():
			cm.updateConnectionState(address, ConnectionTimeout, fmt.Errorf("context cancelled"))
			return nil, fmt.Errorf("connection cancelled: %w", ctx.Err())

		default:
			// Update state for attempt
			cm.updateConnectionAttempt(address, attempt)

			// Attempt connection
			result, err := cm.attemptConnection(ctx, address)
			if err == nil {
				cm.updateConnectionState(address, ConnectionConnected, nil)
				return result, nil
			}

			// Record failure
			cm.updateConnectionState(address, ConnectionFailed, err)
			cm.monitor.RecordValue("connection_failures", 1, "count")

			// Don't retry on final attempt
			if attempt >= cm.maxRetries {
				break
			}

			// Calculate retry delay
			delay := cm.calculateRetryDelay(attempt)
			cm.updateNextRetry(address, delay)

			// Wait before retry
			select {
			case <-ctx.Done():
				return nil, fmt.Errorf("connection cancelled during retry: %w", ctx.Err())
			case <-time.After(delay):
				continue
			}
		}
	}

	return nil, fmt.Errorf("connection failed after %d attempts to %s", cm.maxRetries+1, address)
}

// TestPortAvailability tests if a port is available on the target
func (cm *ConnectionManager) TestPortAvailability(ctx context.Context, target string, port int, timeout time.Duration) error {
	timer := cm.monitor.StartOperation("port_test")
	defer func() { _ = timer.End() }()

	address := fmt.Sprintf("%s:%d", target, port)

	// Create dialer with timeout
	dialer := &net.Dialer{
		Timeout: timeout,
	}

	// Attempt connection
	conn, err := dialer.DialContext(ctx, "tcp", address)
	if err != nil {
		cm.monitor.RecordValue("port_test_failures", 1, "count")
		return fmt.Errorf("port %d not available on %s: %w", port, target, err)
	}

	// Close connection immediately
	_ = conn.Close()
	cm.monitor.RecordValue("port_test_successes", 1, "count")
	return nil
}

// WaitForPortAvailability waits for a port to become available
func (cm *ConnectionManager) WaitForPortAvailability(ctx context.Context, target string, port int, maxWait time.Duration) error {
	timer := cm.monitor.StartOperation("port_wait")
	defer func() { _ = timer.End() }()

	timeout := time.NewTimer(maxWait)
	defer timeout.Stop()

	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("cancelled while waiting for port %d: %w", port, ctx.Err())

		case <-timeout.C:
			return fmt.Errorf("timeout waiting for port %d on %s after %v", port, target, maxWait)

		case <-ticker.C:
			if err := cm.TestPortAvailability(ctx, target, port, 5*time.Second); err == nil {
				return nil // Port is available
			}
			// Continue waiting
		}
	}
}

// HealthCheckSSH performs an SSH health check
func (cm *ConnectionManager) HealthCheckSSH(ctx context.Context, target string, port int) (*HealthResult, error) {
	timer := cm.monitor.StartOperation("ssh_health_check")
	defer func() { _ = timer.End() }()

	startTime := time.Now()

	// Test SSH port availability
	err := cm.TestPortAvailability(ctx, target, port, 10*time.Second)
	if err != nil {
		return &HealthResult{
			Service:   "ssh",
			Status:    HealthStatusUnhealthy,
			Error:     err.Error(),
			CheckedAt: startTime,
			Duration:  time.Since(startTime),
		}, err
	}

	// Additional SSH-specific checks could go here
	// For now, port availability is sufficient

	return &HealthResult{
		Service:   "ssh",
		Status:    HealthStatusHealthy,
		CheckedAt: startTime,
		Duration:  time.Since(startTime),
	}, nil
}

// HealthCheckHTTP performs an HTTP health check
func (cm *ConnectionManager) HealthCheckHTTP(ctx context.Context, target string, port int, path string) (*HealthResult, error) {
	timer := cm.monitor.StartOperation("http_health_check")
	defer func() { _ = timer.End() }()

	startTime := time.Now()

	// Test HTTP port availability
	err := cm.TestPortAvailability(ctx, target, port, 10*time.Second)
	if err != nil {
		return &HealthResult{
			Service:   fmt.Sprintf("http:%d", port),
			Status:    HealthStatusUnhealthy,
			Error:     err.Error(),
			CheckedAt: startTime,
			Duration:  time.Since(startTime),
		}, err
	}

	// Perform actual HTTP request to verify service is responding
	url := fmt.Sprintf("http://%s:%d%s", target, port, path)

	// Create HTTP client with timeout
	httpClient := &http.Client{
		Timeout: 10 * time.Second,
	}

	// Make GET request
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return &HealthResult{
			Service:   fmt.Sprintf("http:%d", port),
			Status:    HealthStatusUnhealthy,
			Error:     fmt.Sprintf("failed to create HTTP request: %v", err),
			CheckedAt: startTime,
			Duration:  time.Since(startTime),
		}, err
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return &HealthResult{
			Service:   fmt.Sprintf("http:%d", port),
			Status:    HealthStatusUnhealthy,
			Error:     fmt.Sprintf("HTTP request failed: %v", err),
			CheckedAt: startTime,
			Duration:  time.Since(startTime),
		}, err
	}
	defer resp.Body.Close()

	// Check for successful response (2xx or 3xx status code)
	if resp.StatusCode >= 400 {
		return &HealthResult{
			Service:   fmt.Sprintf("http:%d", port),
			Status:    HealthStatusUnhealthy,
			Error:     fmt.Sprintf("HTTP %d: %s", resp.StatusCode, resp.Status),
			CheckedAt: startTime,
			Duration:  time.Since(startTime),
		}, fmt.Errorf("HTTP status %d", resp.StatusCode)
	}

	return &HealthResult{
		Service:   fmt.Sprintf("http:%d", port),
		Status:    HealthStatusHealthy,
		CheckedAt: startTime,
		Duration:  time.Since(startTime),
	}, nil
}

// GetConnectionStats returns connection statistics
func (cm *ConnectionManager) GetConnectionStats() ConnectionStats {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	stats := ConnectionStats{
		TotalConnections:    len(cm.connections),
		ConnectionsByStatus: make(map[ConnectionStatus]int),
	}

	for _, conn := range cm.connections {
		stats.ConnectionsByStatus[conn.Status]++
	}

	return stats
}

// attemptConnection performs a single connection attempt
func (cm *ConnectionManager) attemptConnection(ctx context.Context, address string) (*ConnectionResult, error) {
	dialer := &net.Dialer{
		Timeout: 10 * time.Second,
	}

	conn, err := dialer.DialContext(ctx, "tcp", address)
	if err != nil {
		return nil, err
	}

	// Close connection immediately - we just wanted to test connectivity
	_ = conn.Close()

	return &ConnectionResult{
		Address:   address,
		Connected: true,
		Latency:   0, // Could measure actual latency here
	}, nil
}

// calculateRetryDelay calculates the delay before next retry using exponential backoff
func (cm *ConnectionManager) calculateRetryDelay(attempt int) time.Duration {
	delay := float64(cm.baseDelay) * math.Pow(cm.multiplier, float64(attempt))

	// Apply maximum delay cap
	if delay > float64(cm.maxDelay) {
		delay = float64(cm.maxDelay)
	}

	// Apply jitter if enabled
	if cm.jitter {
		jitter := delay * 0.1 // 10% jitter
		delay = delay + (jitter * (2.0*float64(time.Now().UnixNano()%1000)/1000.0 - 1.0))
	}

	return time.Duration(delay)
}

// initConnectionState initializes connection tracking
func (cm *ConnectionManager) initConnectionState(address string) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	cm.connections[address] = &ConnectionState{
		Target:      address,
		Status:      ConnectionPending,
		Attempts:    0,
		LastAttempt: time.Time{},
	}
}

// updateConnectionAttempt updates attempt count and timestamp
func (cm *ConnectionManager) updateConnectionAttempt(address string, attempt int) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	if state, exists := cm.connections[address]; exists {
		state.Status = ConnectionConnecting
		state.Attempts = attempt + 1
		state.LastAttempt = time.Now()
	}
}

// updateConnectionState updates connection state
func (cm *ConnectionManager) updateConnectionState(address string, status ConnectionStatus, err error) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	if state, exists := cm.connections[address]; exists {
		state.Status = status
		if err != nil {
			state.LastError = err.Error()
		} else {
			state.LastError = ""
		}
	}
}

// updateNextRetry updates the next retry time
func (cm *ConnectionManager) updateNextRetry(address string, delay time.Duration) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	if state, exists := cm.connections[address]; exists {
		state.NextRetry = time.Now().Add(delay)
	}
}

// ConnectionResult represents a successful connection result
type ConnectionResult struct {
	Address   string        `json:"address"`
	Connected bool          `json:"connected"`
	Latency   time.Duration `json:"latency"`
}

// HealthResult represents a health check result
type HealthResult struct {
	Service   string        `json:"service"`
	Status    HealthStatus  `json:"status"`
	Error     string        `json:"error,omitempty"`
	CheckedAt time.Time     `json:"checked_at"`
	Duration  time.Duration `json:"duration"`
}

// HealthStatus represents health status
type HealthStatus string

const (
	HealthStatusHealthy   HealthStatus = "healthy"
	HealthStatusUnhealthy HealthStatus = "unhealthy"
	HealthStatusUnknown   HealthStatus = "unknown"
)

// ConnectionStats provides connection statistics
type ConnectionStats struct {
	TotalConnections    int                      `json:"total_connections"`
	ConnectionsByStatus map[ConnectionStatus]int `json:"connections_by_status"`
}
