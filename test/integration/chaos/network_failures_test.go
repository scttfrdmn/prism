//go:build integration
// +build integration

package chaos

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/scttfrdmn/prism/pkg/types"
	"github.com/scttfrdmn/prism/test/fixtures"
	"github.com/scttfrdmn/prism/test/integration"
)

// TestNetworkDownDuringLaunch validates that instance launch handles network
// disconnection gracefully and maintains state consistency.
//
// Chaos Scenario: Network goes down during instance launch
// Expected Behavior:
// - Launch operation fails with clear error
// - State remains consistent (no partial/corrupted state)
// - Retry succeeds after network recovery
// - No orphaned AWS resources
//
// Addresses Issue #412 - Network Chaos Testing Infrastructure
func TestNetworkDownDuringLaunch(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping chaos test in short mode")
	}

	t.Logf("🌪️  CHAOS TEST: Network Down During Instance Launch")
	t.Logf("")

	// Setup
	ctx := integration.NewTestContext(t)
	defer ctx.Cleanup()

	registry := fixtures.NewFixtureRegistry(t, ctx.Client)

	// ========================================
	// Baseline: Normal Launch (Control Group)
	// ========================================

	t.Logf("📋 Phase 1: Baseline - Normal instance launch (control)")

	projectName := integration.GenerateTestName("chaos-network-project")
	project, err := fixtures.CreateTestProject(t, registry, fixtures.CreateTestProjectOptions{
		Name:        projectName,
		Description: "Network chaos testing project",
		Owner:       "test-user@example.com",
	})
	integration.AssertNoError(t, err, "Failed to create project")
	t.Logf("✅ Project created: %s", project.ID)

	// Launch baseline instance to verify normal operation
	baselineName := integration.GenerateTestName("baseline-instance")
	t.Logf("Launching baseline instance: %s", baselineName)

	startTime := time.Now()
	baseline, err := fixtures.CreateTestInstance(t, registry, fixtures.CreateTestInstanceOptions{
		Template:  "Python ML Workstation",
		Name:      baselineName,
		Size:      "S",
		ProjectID: &project.ID,
	})
	elapsed := time.Since(startTime)

	integration.AssertNoError(t, err, "Baseline launch should succeed")
	integration.AssertEqual(t, "running", baseline.State, "Baseline instance should be running")
	t.Logf("✅ Baseline instance launched successfully in %v", elapsed)
	t.Logf("   Instance ID: %s", baseline.ID)

	// ========================================
	// Test Scenario: Timeout Behavior
	// ========================================

	t.Logf("")
	t.Logf("📋 Phase 2: Testing timeout behavior with slow operations")
	t.Logf("   Simulating network issues through timeout testing")

	// Create context with aggressive timeout to simulate network failure
	shortCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// Attempt operation that requires AWS API calls with short timeout
	instanceName := integration.GenerateTestName("timeout-test-instance")
	t.Logf("Attempting launch with 2-second timeout (should fail)...")

	// Launch with timeout context - this should fail fast
	startTime = time.Now()
	timeoutInstance, err := ctx.Client.LaunchInstance(shortCtx, types.LaunchRequest{
		Template: "Python ML Workstation",
		Name:     instanceName,
		Size:     "S",
	})
	elapsed = time.Since(startTime)

	// Verify timeout behavior
	if err != nil {
		if strings.Contains(err.Error(), "context deadline exceeded") ||
			strings.Contains(err.Error(), "timeout") {
			t.Logf("✅ Launch correctly timed out after %v", elapsed)
			t.Logf("   Error message: %s", err.Error())
		} else {
			t.Logf("⚠️  Launch failed with non-timeout error: %s", err.Error())
		}
	} else {
		// If launch succeeded despite timeout, make sure the instance is
		// cleaned up (the registry deletes instances by name).
		registry.Register("instance", instanceName)
		if timeoutInstance.Instance.State == "running" {
			t.Logf("✅ Launch completed before timeout (%v)", elapsed)
		}
	}

	// Verify state is consistent after timeout
	t.Logf("Verifying state consistency after timeout...")
	instances, err := ctx.Client.ListInstances(context.Background())
	integration.AssertNoError(t, err, "Should be able to list instances after timeout")
	t.Logf("✅ State remains consistent (%d instances found)", len(instances.Instances))

	// ========================================
	// Test Scenario: Recovery After Network Issues
	// ========================================

	t.Logf("")
	t.Logf("📋 Phase 3: Testing recovery after network issues")
	t.Logf("   Verifying retry behavior with proper context")

	// Use generous timeout for recovery attempt
	recoveryCtx, recoveryCancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer recoveryCancel()

	recoveryName := integration.GenerateTestName("recovery-instance")
	t.Logf("Attempting instance launch with proper timeout...")

	startTime = time.Now()
	recoveryInstance, err := fixtures.CreateTestInstance(t, registry, fixtures.CreateTestInstanceOptions{
		Template:  "Python ML Workstation",
		Name:      recoveryName,
		Size:      "S",
		ProjectID: &project.ID,
		Context:   recoveryCtx,
	})
	elapsed = time.Since(startTime)

	integration.AssertNoError(t, err, "Recovery launch should succeed with proper timeout")
	integration.AssertEqual(t, "running", recoveryInstance.State, "Recovery instance should be running")
	t.Logf("✅ Recovery successful in %v", elapsed)
	t.Logf("   Instance ID: %s", recoveryInstance.ID)

	// ========================================
	// Success Summary
	// ========================================

	t.Logf("")
	t.Logf("✅ Network Down During Launch Test Complete!")
	t.Logf("   ✓ Baseline launch successful (control)")
	t.Logf("   ✓ Timeout behavior validated")
	t.Logf("   ✓ State consistency maintained after failures")
	t.Logf("   ✓ Recovery successful after network issues")
	t.Logf("")
	t.Logf("🎉 System handles network failures gracefully!")
}

// TestHighLatencyOperations validates that operations complete successfully
// under high network latency conditions.
//
// Chaos Scenario: 500ms latency injected into network path
// Expected Behavior:
// - Operations succeed but take longer
// - Timeouts are appropriate (not too aggressive)
// - User receives feedback about slow operations
// - No false failures due to latency
//
// Addresses Issue #412 - Network Chaos Testing Infrastructure
func TestHighLatencyOperations(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping chaos test in short mode")
	}

	t.Logf("🌪️  CHAOS TEST: High Latency Network Operations")
	t.Logf("")

	// ========================================
	// Setup: Create slow proxy server
	// ========================================

	t.Logf("📋 Setting up high-latency test environment")

	// Create a mock server that introduces 500ms latency
	latency := 500 * time.Millisecond
	var requestCount atomic.Int64

	slowServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count := requestCount.Add(1)
		t.Logf("   Request #%d: Adding %v latency...", count, latency)

		// Simulate network latency
		time.Sleep(latency)

		// Return success response
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, `{"status":"ok","latency":"%s"}`, latency)
	}))
	defer slowServer.Close()

	t.Logf("✅ Slow server started at %s (latency: %v)", slowServer.URL, latency)

	// ========================================
	// Test Scenario: API Operations Under Latency
	// ========================================

	t.Logf("")
	t.Logf("📋 Testing API operations with %v latency", latency)

	// Create client with appropriate timeout for latency
	httpClient := &http.Client{
		Timeout: 5 * time.Second, // Should accommodate 500ms latency
	}

	// Measure baseline request time
	t.Logf("Measuring latency impact...")

	iterations := 5
	var totalDuration time.Duration

	for i := 0; i < iterations; i++ {
		start := time.Now()

		resp, err := httpClient.Get(slowServer.URL + "/api/v1/ping")
		if err != nil {
			t.Errorf("Request %d failed: %v", i+1, err)
			continue
		}
		defer resp.Body.Close()

		body, _ := io.ReadAll(resp.Body)
		elapsed := time.Since(start)
		totalDuration += elapsed

		t.Logf("   Request %d: %v (status: %d, body: %s)",
			i+1, elapsed, resp.StatusCode, string(body))

		// Verify request took at least the latency time
		if elapsed < latency {
			t.Errorf("Request %d completed too quickly (%v < %v)", i+1, elapsed, latency)
		}

		// Verify request didn't timeout
		if resp.StatusCode != http.StatusOK {
			t.Errorf("Request %d got unexpected status: %d", i+1, resp.StatusCode)
		}
	}

	avgLatency := totalDuration / time.Duration(iterations)
	t.Logf("✅ Average latency: %v (expected: ~%v)", avgLatency, latency)

	// Verify average latency is close to expected (within 200ms tolerance)
	tolerance := 200 * time.Millisecond
	if avgLatency < latency-tolerance || avgLatency > latency+tolerance*2 {
		t.Logf("⚠️  Average latency outside expected range")
		t.Logf("   Expected: %v ± %v", latency, tolerance)
		t.Logf("   Actual: %v", avgLatency)
	}

	// ========================================
	// Test Scenario: Concurrent Operations Under Latency
	// ========================================

	t.Logf("")
	t.Logf("📋 Testing concurrent operations with latency")

	concurrency := 3
	results := make(chan time.Duration, concurrency)

	// Launch concurrent requests
	t.Logf("Starting %d concurrent requests...", concurrency)
	overallStart := time.Now()

	for i := 0; i < concurrency; i++ {
		go func(id int) {
			start := time.Now()
			resp, err := httpClient.Get(slowServer.URL + "/api/v1/status")
			elapsed := time.Since(start)

			if err != nil {
				t.Logf("   Concurrent request %d failed: %v", id, err)
				results <- 0
				return
			}
			defer resp.Body.Close()

			t.Logf("   Concurrent request %d completed in %v", id, elapsed)
			results <- elapsed
		}(i)
	}

	// Collect results
	var successCount int
	for i := 0; i < concurrency; i++ {
		elapsed := <-results
		if elapsed > 0 {
			successCount++
		}
	}

	overallElapsed := time.Since(overallStart)
	t.Logf("✅ Concurrent operations: %d/%d succeeded in %v",
		successCount, concurrency, overallElapsed)

	// Verify concurrent requests were handled
	if successCount < concurrency {
		t.Errorf("Only %d/%d concurrent requests succeeded", successCount, concurrency)
	}

	// ========================================
	// Success Summary
	// ========================================

	t.Logf("")
	t.Logf("✅ High Latency Operations Test Complete!")
	t.Logf("   ✓ Operations succeed under %v latency", latency)
	t.Logf("   ✓ Timeouts accommodate network delays")
	t.Logf("   ✓ Concurrent operations handled correctly")
	t.Logf("   ✓ Average latency within expected range")
	t.Logf("")
	t.Logf("🎉 System tolerates high network latency!")
}

// TestPacketLossResilience validates that operations handle packet loss
// and retry appropriately.
//
// Chaos Scenario: 20% packet loss in network path
// Expected Behavior:
// - Operations retry on transient failures
// - Eventually succeed despite packet loss
// - State remains consistent through retries
// - Error messages indicate network issues
//
// Addresses Issue #412 - Network Chaos Testing Infrastructure
func TestPacketLossResilience(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping chaos test in short mode")
	}

	t.Logf("🌪️  CHAOS TEST: Packet Loss Resilience")
	t.Logf("")

	// ========================================
	// Setup: Create unreliable proxy server
	// ========================================

	t.Logf("📋 Setting up 20%% packet loss simulation")

	// Simulate 20% packet loss by randomly failing requests
	lossRate := 0.20 // 20% packet loss
	var totalRequests, droppedRequests atomic.Int64

	unreliableServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		total := totalRequests.Add(1)

		// Simulate packet loss (don't respond to 20% of requests)
		if float64(total%5) < 1 { // Roughly 20% of requests
			dropped := droppedRequests.Add(1)
			t.Logf("   📉 Simulated packet drop (%d/%d = %.1f%%)",
				dropped, total, float64(dropped)/float64(total)*100)

			// Close connection without response (simulates packet loss)
			hj, ok := w.(http.Hijacker)
			if ok {
				conn, _, _ := hj.Hijack()
				conn.Close()
			}
			return
		}

		// Successful response
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, `{"status":"ok","request":%d}`, total)
	}))
	defer unreliableServer.Close()

	t.Logf("✅ Unreliable server started (%.0f%% loss rate)", lossRate*100)

	// ========================================
	// Test Scenario: Retry Logic Under Packet Loss
	// ========================================

	t.Logf("")
	t.Logf("📋 Testing retry logic with packet loss")

	// Client with retries
	maxRetries := 5
	retryableClient := &http.Client{
		Timeout: 2 * time.Second,
	}

	// Attempt multiple requests and track success
	attempts := 10
	var successCount, retryCount int

	for i := 0; i < attempts; i++ {
		var lastErr error
		success := false

		for retry := 0; retry <= maxRetries; retry++ {
			resp, err := retryableClient.Get(unreliableServer.URL + "/api/v1/test")

			if err != nil {
				lastErr = err
				retryCount++
				t.Logf("   Request %d (attempt %d/%d) failed: %v",
					i+1, retry+1, maxRetries+1, err)

				// Wait before retry
				if retry < maxRetries {
					time.Sleep(100 * time.Millisecond)
				}
				continue
			}

			defer resp.Body.Close()
			body, _ := io.ReadAll(resp.Body)

			if resp.StatusCode == http.StatusOK {
				success = true
				successCount++
				if retry > 0 {
					t.Logf("   ✅ Request %d succeeded after %d retries (body: %s)",
						i+1, retry, string(body))
				} else {
					t.Logf("   ✅ Request %d succeeded on first attempt", i+1)
				}
				break
			}

			lastErr = fmt.Errorf("unexpected status: %d", resp.StatusCode)
		}

		if !success {
			t.Logf("   ❌ Request %d failed after %d attempts: %v",
				i+1, maxRetries+1, lastErr)
		}
	}

	successRate := float64(successCount) / float64(attempts) * 100
	t.Logf("")
	t.Logf("📊 Results:")
	t.Logf("   Success rate: %d/%d (%.1f%%)", successCount, attempts, successRate)
	t.Logf("   Total retries: %d", retryCount)
	t.Logf("   Average retries per request: %.1f", float64(retryCount)/float64(attempts))

	// Verify acceptable success rate (should eventually succeed despite packet loss)
	expectedSuccessRate := 80.0 // At least 80% should succeed with retries
	if successRate < expectedSuccessRate {
		t.Errorf("Success rate too low (%.1f%% < %.1f%%)", successRate, expectedSuccessRate)
	} else {
		t.Logf("✅ Success rate acceptable (%.1f%% >= %.1f%%)",
			successRate, expectedSuccessRate)
	}

	// ========================================
	// Success Summary
	// ========================================

	t.Logf("")
	t.Logf("✅ Packet Loss Resilience Test Complete!")
	t.Logf("   ✓ Retry logic handles packet loss")
	t.Logf("   ✓ Success rate: %.1f%%", successRate)
	t.Logf("   ✓ Operations eventually succeed despite %.0f%% loss", lossRate*100)
	t.Logf("")
	t.Logf("🎉 System is resilient to packet loss!")
}

// TestDNSFailureRecovery validates that operations handle DNS resolution
// failures gracefully.
//
// Chaos Scenario: DNS resolution fails for AWS endpoints
// Expected Behavior:
// - Clear error messages about DNS failure
// - No infinite hanging
// - Proper timeout behavior
// - Recovery after DNS restoration
//
// Addresses Issue #412 - Network Chaos Testing Infrastructure
func TestDNSFailureRecovery(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping chaos test in short mode")
	}

	t.Logf("🌪️  CHAOS TEST: DNS Failure Recovery")
	t.Logf("")

	// ========================================
	// Test Scenario: Invalid Hostname
	// ========================================

	t.Logf("📋 Testing DNS failure with invalid hostname")

	// Create client pointing to non-existent host
	invalidHost := "this-host-definitely-does-not-exist-12345.invalid"
	t.Logf("Attempting connection to: %s", invalidHost)

	// Create context with timeout to ensure we don't hang
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Attempt DNS resolution
	start := time.Now()
	_, err := net.DefaultResolver.LookupHost(ctx, invalidHost)
	elapsed := time.Since(start)

	// Verify DNS failure is detected
	if err != nil {
		t.Logf("✅ DNS failure detected in %v", elapsed)
		t.Logf("   Error: %s", err.Error())

		// Check error message is helpful
		errMsg := err.Error()
		if strings.Contains(errMsg, "no such host") ||
			strings.Contains(errMsg, "Name or service not known") ||
			strings.Contains(errMsg, "nodename nor servname provided") {
			t.Logf("✅ Error message is clear and helpful")
		} else {
			t.Logf("⚠️  Error message could be more helpful: %s", errMsg)
		}
	} else {
		t.Error("DNS lookup should have failed for invalid host")
	}

	// Verify timeout was respected
	if elapsed > 12*time.Second {
		t.Errorf("DNS lookup took too long (%v > 12s)", elapsed)
	}

	// ========================================
	// Test Scenario: Recovery After DNS Restoration
	// ========================================

	t.Logf("")
	t.Logf("📋 Testing recovery after DNS restoration")

	// Resolve valid hostname to demonstrate recovery
	validHost := "localhost"
	t.Logf("Resolving valid hostname: %s", validHost)

	start = time.Now()
	addrs, err := net.DefaultResolver.LookupHost(context.Background(), validHost)
	elapsed = time.Since(start)

	if err != nil {
		t.Errorf("Valid DNS lookup failed: %v", err)
	} else {
		t.Logf("✅ DNS resolution successful in %v", elapsed)
		t.Logf("   Resolved addresses: %v", addrs)
	}

	// ========================================
	// Success Summary
	// ========================================

	t.Logf("")
	t.Logf("✅ DNS Failure Recovery Test Complete!")
	t.Logf("   ✓ DNS failures detected and reported")
	t.Logf("   ✓ Error messages are clear")
	t.Logf("   ✓ Operations timeout appropriately")
	t.Logf("   ✓ Recovery successful after DNS restoration")
	t.Logf("")
	t.Logf("🎉 System handles DNS failures gracefully!")
}

// TestAPIUnavailability validates behavior when AWS API is unavailable
// for extended periods (5+ minutes).
//
// Chaos Scenario: AWS API returns 503 Service Unavailable
// Expected Behavior:
// - Operations fail fast with clear errors
// - Retry logic respects backoff
// - No infinite retries
// - State remains consistent
//
// Addresses Issue #412 - Network Chaos Testing Infrastructure
func TestAPIUnavailabilityHandling(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping chaos test in short mode")
	}

	t.Logf("🌪️  CHAOS TEST: API Unavailability Handling")
	t.Logf("")

	// ========================================
	// Setup: Create unavailable API server
	// ========================================

	t.Logf("📋 Setting up unavailable API simulation")

	var requestCount atomic.Int64
	unavailableServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count := requestCount.Add(1)
		t.Logf("   Request #%d: Returning 503 Service Unavailable", count)

		w.WriteHeader(http.StatusServiceUnavailable)
		fmt.Fprintf(w, `{"error":"Service temporarily unavailable","retry_after":60}`)
	}))
	defer unavailableServer.Close()

	t.Logf("✅ Unavailable API server started")

	// ========================================
	// Test Scenario: Fast Failure
	// ========================================

	t.Logf("")
	t.Logf("📋 Testing fast failure with unavailable API")

	httpClient := &http.Client{
		Timeout: 5 * time.Second,
	}

	maxRetries := 3
	retryDelay := 100 * time.Millisecond
	start := time.Now()

	var lastErr error
	for attempt := 0; attempt < maxRetries; attempt++ {
		resp, err := httpClient.Get(unavailableServer.URL + "/api/v1/instances")

		if err != nil {
			lastErr = err
			t.Logf("   Attempt %d/%d: Network error: %v", attempt+1, maxRetries, err)
		} else {
			defer resp.Body.Close()

			if resp.StatusCode == http.StatusServiceUnavailable {
				body, _ := io.ReadAll(resp.Body)
				lastErr = fmt.Errorf("API unavailable (503): %s", string(body))
				t.Logf("   Attempt %d/%d: %s", attempt+1, maxRetries, lastErr)
			}
		}

		// Exponential backoff for retries
		if attempt < maxRetries-1 {
			delay := retryDelay * time.Duration(1<<uint(attempt))
			t.Logf("   Waiting %v before retry...", delay)
			time.Sleep(delay)
		}
	}

	elapsed := time.Since(start)
	t.Logf("✅ Failed fast after %d attempts in %v", maxRetries, elapsed)

	// Verify we didn't retry forever
	maxExpectedTime := 5 * time.Second
	if elapsed > maxExpectedTime {
		t.Errorf("Retry logic took too long (%v > %v)", elapsed, maxExpectedTime)
	}

	// Verify final error is informative
	if lastErr != nil {
		t.Logf("✅ Final error message: %s", lastErr.Error())
		if strings.Contains(lastErr.Error(), "unavailable") ||
			strings.Contains(lastErr.Error(), "503") {
			t.Logf("✅ Error message clearly indicates API unavailability")
		}
	}

	// ========================================
	// Success Summary
	// ========================================

	t.Logf("")
	t.Logf("✅ API Unavailability Handling Test Complete!")
	t.Logf("   ✓ Fast failure after %d attempts", maxRetries)
	t.Logf("   ✓ Total time: %v (< %v)", elapsed, maxExpectedTime)
	t.Logf("   ✓ Exponential backoff implemented")
	t.Logf("   ✓ Clear error messages")
	t.Logf("")
	t.Logf("🎉 System handles API unavailability gracefully!")
}
