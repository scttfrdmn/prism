// Package monitoring provides comprehensive functional tests for performance monitoring and metrics collection
package monitoring

import (
	"context"
	"testing"
	"time"
)

// TestPerformanceMonitorFunctionalWorkflow validates complete performance monitoring lifecycle
func TestPerformanceMonitorFunctionalWorkflow(t *testing.T) {
	monitor := setupPerformanceMonitor(t)

	// Test complete monitoring workflow
	testPerformanceMonitorCreation(t, monitor)
	testSystemMetricsCollection(t, monitor)
	testCustomMetricsRecording(t, monitor)
	testOperationTiming(t, monitor)
	testMetricsRetrieval(t, monitor)
	testPerformanceSummary(t, monitor)

	t.Log("✅ Performance monitor functional workflow validated")
}

// setupPerformanceMonitor creates and configures a performance monitor for testing
func setupPerformanceMonitor(t *testing.T) *PerformanceMonitor {
	monitor := NewPerformanceMonitor()
	if monitor == nil {
		t.Fatal("Failed to create performance monitor")
	}

	// Verify initial state
	if monitor.metrics == nil {
		t.Error("Performance monitor metrics map should be initialized")
	}

	if monitor.started.IsZero() {
		t.Error("Performance monitor start time should be set")
	}

	return monitor
}

// testPerformanceMonitorCreation validates monitor initialization
func testPerformanceMonitorCreation(t *testing.T, monitor *PerformanceMonitor) {
	// Test default interval
	expectedInterval := 30 * time.Second
	if monitor.interval != expectedInterval {
		t.Errorf("Expected interval %v, got %v", expectedInterval, monitor.interval)
	}

	// Test metrics map initialization
	if len(monitor.metrics) != 0 {
		t.Error("New monitor should start with empty metrics")
	}

	// Test started timestamp is recent
	timeSinceStart := time.Since(monitor.started)
	if timeSinceStart > time.Minute {
		t.Error("Monitor start time should be recent")
	}
}

// testSystemMetricsCollection validates automatic system metrics collection
func testSystemMetricsCollection(t *testing.T, monitor *PerformanceMonitor) {
	// Force system metrics collection
	monitor.collectSystemMetrics()

	// Validate expected system metrics exist
	expectedMetrics := []string{
		"memory_heap_alloc",
		"memory_heap_sys",
		"memory_heap_objects",
		"goroutines",
		"gc_cycles",
	}

	for _, metricName := range expectedMetrics {
		metric, exists := monitor.GetMetric(metricName)
		if !exists {
			t.Errorf("Expected system metric %s not found", metricName)
			continue
		}

		// Validate metric structure
		validateMetricStructure(t, metric, metricName)
	}

	t.Log("System metrics collection validated")
}

// testCustomMetricsRecording validates custom metric recording functionality
func testCustomMetricsRecording(t *testing.T, monitor *PerformanceMonitor) {
	testCases := []struct {
		name  string
		value float64
		unit  string
	}{
		{"cpu_usage", 45.6, "percent"},
		{"disk_io_rate", 1024.5, "MB/s"},
		{"network_latency", 23.4, "ms"},
		{"cache_hit_ratio", 0.95, "ratio"},
	}

	for _, tc := range testCases {
		monitor.RecordValue(tc.name, tc.value, tc.unit)

		// Verify metric was recorded correctly
		metric, exists := monitor.GetMetric(tc.name)
		if !exists {
			t.Errorf("Custom metric %s not found after recording", tc.name)
			continue
		}

		if metric.Value != tc.value {
			t.Errorf("Metric %s: expected value %f, got %f", tc.name, tc.value, metric.Value)
		}

		if metric.Unit != tc.unit {
			t.Errorf("Metric %s: expected unit %s, got %s", tc.name, tc.unit, metric.Unit)
		}

		// Validate history tracking
		if len(metric.History) != 1 {
			t.Errorf("Metric %s: expected 1 history entry, got %d", tc.name, len(metric.History))
		}
	}

	t.Log("Custom metrics recording validated")
}

// testOperationTiming validates operation timing functionality
func testOperationTiming(t *testing.T, monitor *PerformanceMonitor) {
	// Test various operations with different durations
	operations := []struct {
		name     string
		duration time.Duration
	}{
		{"database_query", 50 * time.Millisecond},
		{"api_request", 200 * time.Millisecond},
		{"file_processing", 100 * time.Millisecond},
	}

	for _, op := range operations {
		// Start operation timing
		timer := monitor.StartOperation(op.name)
		if timer == nil {
			t.Errorf("StartOperation returned nil for %s", op.name)
			continue
		}

		// Simulate operation duration
		time.Sleep(op.duration)

		// End operation timing
		actualDuration := timer.End()

		// Validate timing accuracy (allow 50ms tolerance)
		tolerance := 50 * time.Millisecond
		if actualDuration < op.duration-tolerance || actualDuration > op.duration+tolerance {
			t.Errorf("Operation %s: expected ~%v, got %v", op.name, op.duration, actualDuration)
		}

		// Verify timing metric was recorded
		expectedMetricName := "timing_" + op.name
		metric, exists := monitor.GetMetric(expectedMetricName)
		if !exists {
			t.Errorf("Timing metric %s not found", expectedMetricName)
			continue
		}

		// Validate metric properties
		if metric.Unit != "milliseconds" {
			t.Errorf("Timing metric %s: expected unit 'milliseconds', got %s", expectedMetricName, metric.Unit)
		}

		if len(metric.History) == 0 {
			t.Errorf("Timing metric %s: expected history entries", expectedMetricName)
		}
	}

	t.Log("Operation timing validated")
}

// testMetricsRetrieval validates metrics retrieval functionality
func testMetricsRetrieval(t *testing.T, monitor *PerformanceMonitor) {
	// Add test metrics
	monitor.RecordValue("test_metric_1", 100.0, "units")
	monitor.RecordValue("test_metric_2", 200.0, "units")

	// Test GetMetrics returns all metrics
	allMetrics := monitor.GetMetrics()
	if len(allMetrics) < 2 {
		t.Error("GetMetrics should return all recorded metrics")
	}

	// Test GetMetric returns specific metric
	metric, exists := monitor.GetMetric("test_metric_1")
	if !exists {
		t.Error("GetMetric should find existing metric")
	}

	if metric.Value != 100.0 {
		t.Error("GetMetric should return correct metric value")
	}

	// Test GetMetric returns false for non-existent metric
	_, exists = monitor.GetMetric("non_existent_metric")
	if exists {
		t.Error("GetMetric should return false for non-existent metric")
	}

	t.Log("Metrics retrieval validated")
}

// testPerformanceSummary validates performance summary generation
func testPerformanceSummary(t *testing.T, monitor *PerformanceMonitor) {
	// Add some operation timings to test summary calculation
	monitor.RecordTiming("operation_1", 100*time.Millisecond)
	monitor.RecordTiming("operation_1", 200*time.Millisecond)
	monitor.RecordTiming("operation_2", 150*time.Millisecond)

	// Generate performance summary
	summary := monitor.GetSummary()

	// Validate summary structure
	if summary.Uptime <= 0 {
		t.Error("Summary uptime should be positive")
	}

	if summary.MemoryUsageMB < 0 {
		t.Error("Summary memory usage should be non-negative")
	}

	if summary.Goroutines <= 0 {
		t.Error("Summary goroutines count should be positive")
	}

	if summary.MetricCount <= 0 {
		t.Error("Summary metric count should be positive")
	}

	// Validate operation timings in summary
	if len(summary.OperationTimings) == 0 {
		t.Error("Summary should include operation timings")
	}

	// Check specific operation average
	if timing, exists := summary.OperationTimings["operation_1"]; exists {
		// Should be average of 100ms and 200ms = 150ms
		expectedAvg := 150.0
		tolerance := 10.0
		if timing < expectedAvg-tolerance || timing > expectedAvg+tolerance {
			t.Errorf("Operation_1 average: expected ~%f, got %f", expectedAvg, timing)
		}
	} else {
		t.Error("Summary should include operation_1 timing")
	}

	t.Log("Performance summary validated")
}

// validateMetricStructure validates that a metric has proper structure
func validateMetricStructure(t *testing.T, metric *Metric, name string) {
	if metric.Name != name {
		t.Errorf("Metric %s: expected name %s, got %s", name, name, metric.Name)
	}

	if metric.LastUpdated.IsZero() {
		t.Errorf("Metric %s: LastUpdated should be set", name)
	}

	if metric.Unit == "" {
		t.Errorf("Metric %s: Unit should not be empty", name)
	}

	if metric.History == nil {
		t.Errorf("Metric %s: History should be initialized", name)
	}
}

// TestLaunchSpeedTrackerFunctionalWorkflow validates launch performance tracking
func TestLaunchSpeedTrackerFunctionalWorkflow(t *testing.T) {
	monitor := NewPerformanceMonitor()
	tracker := NewLaunchSpeedTracker(monitor)

	if tracker == nil {
		t.Fatal("Failed to create launch speed tracker")
	}

	// Test launch phase tracking
	testLaunchPhaseTracking(t, tracker)

	// Test total launch time tracking
	testTotalLaunchTimeTracking(t, tracker)

	// Test launch statistics generation
	testLaunchStatisticsGeneration(t, tracker)

	t.Log("✅ Launch speed tracker functional workflow validated")
}

// testLaunchPhaseTracking validates launch phase timing tracking
func testLaunchPhaseTracking(t *testing.T, tracker *LaunchSpeedTracker) {
	phases := []struct {
		name     string
		duration time.Duration
	}{
		{"ami_resolution", 2 * time.Second},
		{"instance_launch", 5 * time.Second},
		{"configuration", 3 * time.Second},
		{"health_check", 1 * time.Second},
	}

	for _, phase := range phases {
		tracker.TrackLaunchPhase(phase.name, phase.duration)

		// Verify phase timing was recorded
		expectedMetricName := "timing_launch_phase_" + phase.name
		metric, exists := tracker.monitor.GetMetric(expectedMetricName)
		if !exists {
			t.Errorf("Launch phase metric %s not found", expectedMetricName)
			continue
		}

		expectedValue := float64(phase.duration.Nanoseconds()) / 1e6
		if metric.Value != expectedValue {
			t.Errorf("Phase %s: expected %f ms, got %f ms", phase.name, expectedValue, metric.Value)
		}
	}

	t.Log("Launch phase tracking validated")
}

// testTotalLaunchTimeTracking validates total launch time tracking
func testTotalLaunchTimeTracking(t *testing.T, tracker *LaunchSpeedTracker) {
	totalDuration := 15 * time.Second
	templateName := "python-ml"

	tracker.TrackTotalLaunchTime(totalDuration, templateName)

	// Verify total launch metric
	totalMetric, exists := tracker.monitor.GetMetric("timing_launch_total")
	if !exists {
		t.Error("Total launch time metric not found")
	} else {
		expectedValue := float64(totalDuration.Nanoseconds()) / 1e6
		if totalMetric.Value != expectedValue {
			t.Errorf("Total launch time: expected %f ms, got %f ms", expectedValue, totalMetric.Value)
		}
	}

	// Verify template-specific metric
	templateMetric, exists := tracker.monitor.GetMetric("timing_launch_template_" + templateName)
	if !exists {
		t.Error("Template launch time metric not found")
	} else {
		expectedValue := float64(totalDuration.Nanoseconds()) / 1e6
		if templateMetric.Value != expectedValue {
			t.Errorf("Template launch time: expected %f ms, got %f ms", expectedValue, templateMetric.Value)
		}
	}

	t.Log("Total launch time tracking validated")
}

// testLaunchStatisticsGeneration validates launch statistics generation
func testLaunchStatisticsGeneration(t *testing.T, tracker *LaunchSpeedTracker) {
	// Add multiple data points for better statistics
	tracker.TrackLaunchPhase("ami_resolution", 1*time.Second)
	tracker.TrackLaunchPhase("ami_resolution", 2*time.Second)
	tracker.TrackTotalLaunchTime(10*time.Second, "test-template")
	tracker.TrackTotalLaunchTime(12*time.Second, "test-template")

	// Generate launch statistics
	stats := tracker.GetLaunchStats()

	// Validate statistics structure
	if stats.PhaseTimings == nil {
		t.Error("Launch stats phase timings should be initialized")
	}

	if stats.TemplateTimings == nil {
		t.Error("Launch stats template timings should be initialized")
	}

	// Validate phase timing statistics
	if timing, exists := stats.PhaseTimings["ami_resolution"]; exists {
		// Should be average of 1s and 2s = 1.5s = 1500ms
		expectedAvg := 1500.0
		tolerance := 200.0 // Allow more tolerance for averaging calculations
		if timing < expectedAvg-tolerance || timing > expectedAvg+tolerance {
			t.Errorf("AMI resolution average: expected ~%f ms, got %f ms", expectedAvg, timing)
		}
	} else {
		t.Error("Launch stats should include ami_resolution timing")
	}

	// Validate template timing statistics
	if timing, exists := stats.TemplateTimings["test-template"]; exists {
		// Should be average of 10s and 12s = 11s = 11000ms
		expectedAvg := 11000.0
		tolerance := 500.0
		if timing < expectedAvg-tolerance || timing > expectedAvg+tolerance {
			t.Errorf("Test template average: expected ~%f ms, got %f ms", expectedAvg, timing)
		}
	} else {
		t.Error("Launch stats should include test-template timing")
	}

	// Validate average total time
	if stats.AverageTotalTime <= 0 {
		t.Error("Launch stats should include average total time")
	}

	t.Log("Launch statistics generation validated")
}

// TestPerformanceMonitorConcurrency validates thread-safe operations
func TestPerformanceMonitorConcurrency(t *testing.T) {
	monitor := NewPerformanceMonitor()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// Start background monitoring
	go monitor.Start(ctx)

	// Perform concurrent operations
	done := make(chan bool, 3)

	// Concurrent metric recording
	go func() {
		for i := 0; i < 100; i++ {
			monitor.RecordValue("concurrent_metric", float64(i), "count")
			time.Sleep(time.Millisecond)
		}
		done <- true
	}()

	// Concurrent operation timing
	go func() {
		for i := 0; i < 50; i++ {
			timer := monitor.StartOperation("concurrent_operation")
			time.Sleep(time.Millisecond)
			timer.End()
		}
		done <- true
	}()

	// Concurrent metric retrieval
	go func() {
		for i := 0; i < 50; i++ {
			monitor.GetMetrics()
			monitor.GetSummary()
			time.Sleep(time.Millisecond)
		}
		done <- true
	}()

	// Wait for all goroutines to complete
	for i := 0; i < 3; i++ {
		select {
		case <-done:
			// Success
		case <-time.After(5 * time.Second):
			t.Error("Concurrent operations timed out")
		}
	}

	t.Log("✅ Performance monitor concurrency validated")
}
