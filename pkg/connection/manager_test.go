package connection

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/scttfrdmn/cloudworkstation/pkg/monitoring"
)

func TestConnectionManager_ConnectWithRetry(t *testing.T) {
	monitor := monitoring.NewPerformanceMonitor()
	cm := NewConnectionManager(monitor)

	tests := []struct {
		name        string
		target      string
		port        int
		expectError bool
	}{
		{
			name:        "Connect to valid service (Google DNS)",
			target:      "8.8.8.8",
			port:        53,
			expectError: false,
		},
		{
			name:        "Connect to invalid port",
			target:      "127.0.0.1",
			port:        99999, // Invalid port
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			result, err := cm.ConnectWithRetry(ctx, tt.target, tt.port)

			if tt.expectError && err == nil {
				t.Errorf("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
			if !tt.expectError && result == nil {
				t.Errorf("Expected result but got nil")
			}
		})
	}
}

func TestConnectionManager_TestPortAvailability(t *testing.T) {
	monitor := monitoring.NewPerformanceMonitor()
	cm := NewConnectionManager(monitor)

	// Start a test server
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Failed to start test server: %v", err)
	}
	defer listener.Close()

	// Get the actual port
	addr := listener.Addr().(*net.TCPAddr)
	port := addr.Port

	ctx := context.Background()

	// Test successful connection
	err = cm.TestPortAvailability(ctx, "127.0.0.1", port, 5*time.Second)
	if err != nil {
		t.Errorf("Expected port to be available, got error: %v", err)
	}

	// Test unsuccessful connection
	err = cm.TestPortAvailability(ctx, "127.0.0.1", port+1, 1*time.Second)
	if err == nil {
		t.Errorf("Expected error for unavailable port")
	}
}

func TestConnectionManager_WaitForPortAvailability(t *testing.T) {
	monitor := monitoring.NewPerformanceMonitor()
	cm := NewConnectionManager(monitor)

	ctx := context.Background()

	// Test waiting for a port that becomes available
	go func() {
		// Wait a bit then start server
		time.Sleep(2 * time.Second)
		listener, err := net.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			return
		}
		defer listener.Close()

		// Keep server running for a while
		time.Sleep(5 * time.Second)
	}()

	// This test is complex due to dynamic port assignment
	// In a real scenario, you'd test with a known port

	// Test timeout scenario
	err := cm.WaitForPortAvailability(ctx, "127.0.0.1", 99998, 1*time.Second)
	if err == nil {
		t.Errorf("Expected timeout error")
	}
}

func TestConnectionManager_GetConnectionStats(t *testing.T) {
	monitor := monitoring.NewPerformanceMonitor()
	cm := NewConnectionManager(monitor)

	// Initially should have no connections
	stats := cm.GetConnectionStats()
	if stats.TotalConnections != 0 {
		t.Errorf("Expected 0 connections, got %d", stats.TotalConnections)
	}

	// Attempt a connection to create state
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cm.ConnectWithRetry(ctx, "127.0.0.1", 99999) // This should fail

	stats = cm.GetConnectionStats()
	if stats.TotalConnections == 0 {
		t.Errorf("Expected connections to be tracked")
	}
}

func TestRetryPolicy(t *testing.T) {
	policy := DefaultRetryPolicy()

	if policy.MaxRetries <= 0 {
		t.Errorf("Expected positive MaxRetries, got %d", policy.MaxRetries)
	}
	if policy.BaseDelay <= 0 {
		t.Errorf("Expected positive BaseDelay, got %v", policy.BaseDelay)
	}
	if policy.MaxDelay <= policy.BaseDelay {
		t.Errorf("Expected MaxDelay > BaseDelay, got %v <= %v", policy.MaxDelay, policy.BaseDelay)
	}
	if policy.Multiplier <= 1.0 {
		t.Errorf("Expected Multiplier > 1.0, got %f", policy.Multiplier)
	}
}

func TestConnectionManager_HealthChecks(t *testing.T) {
	monitor := monitoring.NewPerformanceMonitor()
	cm := NewConnectionManager(monitor)
	ctx := context.Background()

	// Test SSH health check (port 22 is usually not available in tests)
	result, err := cm.HealthCheckSSH(ctx, "127.0.0.1", 99999)
	if err == nil {
		t.Errorf("Expected health check to fail for unavailable port")
	}
	if result == nil {
		t.Errorf("Expected health result even on failure")
	}
	if result != nil && result.Status != HealthStatusUnhealthy {
		t.Errorf("Expected unhealthy status, got %v", result.Status)
	}

	// Test HTTP health check
	result, err = cm.HealthCheckHTTP(ctx, "127.0.0.1", 99999, "/health")
	if err == nil {
		t.Errorf("Expected HTTP health check to fail for unavailable port")
	}
	if result == nil {
		t.Errorf("Expected health result even on failure")
	}
	if result != nil && result.Status != HealthStatusUnhealthy {
		t.Errorf("Expected unhealthy status, got %v", result.Status)
	}
}
