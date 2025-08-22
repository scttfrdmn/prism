package monitoring

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"time"
)

// PerformanceMonitor tracks system performance metrics
type PerformanceMonitor struct {
	mu       sync.RWMutex
	metrics  map[string]*Metric
	started  time.Time
	interval time.Duration
}

// Metric represents a performance metric
type Metric struct {
	Name        string      `json:"name"`
	Value       float64     `json:"value"`
	Unit        string      `json:"unit"`
	LastUpdated time.Time   `json:"last_updated"`
	History     []DataPoint `json:"history,omitempty"`
}

// DataPoint represents a point-in-time measurement
type DataPoint struct {
	Timestamp time.Time `json:"timestamp"`
	Value     float64   `json:"value"`
}

// OperationTimer tracks timing for operations
type OperationTimer struct {
	operation string
	startTime time.Time
	monitor   *PerformanceMonitor
}

// NewPerformanceMonitor creates a new performance monitor
func NewPerformanceMonitor() *PerformanceMonitor {
	return &PerformanceMonitor{
		metrics:  make(map[string]*Metric),
		started:  time.Now(),
		interval: 30 * time.Second,
	}
}

// Start begins performance monitoring
func (pm *PerformanceMonitor) Start(ctx context.Context) {
	ticker := time.NewTicker(pm.interval)
	defer ticker.Stop()

	// Initial metrics collection
	pm.collectSystemMetrics()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			pm.collectSystemMetrics()
		}
	}
}

// StartOperation begins timing an operation
func (pm *PerformanceMonitor) StartOperation(operation string) *OperationTimer {
	return &OperationTimer{
		operation: operation,
		startTime: time.Now(),
		monitor:   pm,
	}
}

// EndOperation completes timing an operation
func (ot *OperationTimer) End() time.Duration {
	duration := time.Since(ot.startTime)

	// Record the operation timing
	ot.monitor.RecordTiming(ot.operation, duration)

	return duration
}

// RecordTiming records timing for an operation
func (pm *PerformanceMonitor) RecordTiming(operation string, duration time.Duration) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	metricName := fmt.Sprintf("timing_%s", operation)
	metric, exists := pm.metrics[metricName]
	if !exists {
		metric = &Metric{
			Name:    metricName,
			Unit:    "milliseconds",
			History: make([]DataPoint, 0),
		}
		pm.metrics[metricName] = metric
	}

	value := float64(duration.Nanoseconds()) / 1e6 // Convert to milliseconds
	metric.Value = value
	metric.LastUpdated = time.Now()

	// Add to history (keep last 100 points)
	metric.History = append(metric.History, DataPoint{
		Timestamp: time.Now(),
		Value:     value,
	})
	if len(metric.History) > 100 {
		metric.History = metric.History[1:]
	}
}

// RecordValue records a custom metric value
func (pm *PerformanceMonitor) RecordValue(name string, value float64, unit string) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	metric, exists := pm.metrics[name]
	if !exists {
		metric = &Metric{
			Name:    name,
			Unit:    unit,
			History: make([]DataPoint, 0),
		}
		pm.metrics[name] = metric
	}

	metric.Value = value
	metric.LastUpdated = time.Now()

	// Add to history
	metric.History = append(metric.History, DataPoint{
		Timestamp: time.Now(),
		Value:     value,
	})
	if len(metric.History) > 100 {
		metric.History = metric.History[1:]
	}
}

// collectSystemMetrics collects system-level metrics
func (pm *PerformanceMonitor) collectSystemMetrics() {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	pm.RecordValue("memory_heap_alloc", float64(m.HeapAlloc)/1024/1024, "MB")
	pm.RecordValue("memory_heap_sys", float64(m.HeapSys)/1024/1024, "MB")
	pm.RecordValue("memory_heap_objects", float64(m.HeapObjects), "count")
	pm.RecordValue("goroutines", float64(runtime.NumGoroutine()), "count")
	pm.RecordValue("gc_cycles", float64(m.NumGC), "count")
}

// GetMetrics returns all current metrics
func (pm *PerformanceMonitor) GetMetrics() map[string]*Metric {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	// Return a copy to avoid race conditions
	result := make(map[string]*Metric)
	for k, v := range pm.metrics {
		result[k] = v
	}

	return result
}

// GetMetric returns a specific metric
func (pm *PerformanceMonitor) GetMetric(name string) (*Metric, bool) {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	metric, exists := pm.metrics[name]
	return metric, exists
}

// GetSummary returns a performance summary
func (pm *PerformanceMonitor) GetSummary() PerformanceSummary {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	summary := PerformanceSummary{
		Uptime:        time.Since(pm.started),
		MemoryUsageMB: float64(m.HeapAlloc) / 1024 / 1024,
		Goroutines:    runtime.NumGoroutine(),
		GCCycles:      m.NumGC,
		LastGCPause:   time.Duration(m.PauseNs[(m.NumGC+255)%256]),
		MetricCount:   len(pm.metrics),
	}

	// Calculate average operation timings
	summary.OperationTimings = make(map[string]float64)
	for name, metric := range pm.metrics {
		if len(name) > 7 && name[:7] == "timing_" {
			operation := name[7:]
			if len(metric.History) > 0 {
				total := 0.0
				for _, point := range metric.History {
					total += point.Value
				}
				summary.OperationTimings[operation] = total / float64(len(metric.History))
			}
		}
	}

	return summary
}

// PerformanceSummary provides a high-level performance overview
type PerformanceSummary struct {
	Uptime           time.Duration      `json:"uptime"`
	MemoryUsageMB    float64            `json:"memory_usage_mb"`
	Goroutines       int                `json:"goroutines"`
	GCCycles         uint32             `json:"gc_cycles"`
	LastGCPause      time.Duration      `json:"last_gc_pause"`
	MetricCount      int                `json:"metric_count"`
	OperationTimings map[string]float64 `json:"operation_timings"`
}

// LaunchSpeedTracker specifically tracks launch performance improvements
type LaunchSpeedTracker struct {
	monitor *PerformanceMonitor
}

// NewLaunchSpeedTracker creates a launch speed tracker
func NewLaunchSpeedTracker(monitor *PerformanceMonitor) *LaunchSpeedTracker {
	return &LaunchSpeedTracker{monitor: monitor}
}

// TrackLaunchPhase tracks timing for a specific launch phase
func (lst *LaunchSpeedTracker) TrackLaunchPhase(phase string, duration time.Duration) {
	lst.monitor.RecordTiming(fmt.Sprintf("launch_phase_%s", phase), duration)
}

// TrackTotalLaunchTime tracks overall launch time
func (lst *LaunchSpeedTracker) TrackTotalLaunchTime(duration time.Duration, templateName string) {
	lst.monitor.RecordTiming("launch_total", duration)
	lst.monitor.RecordTiming(fmt.Sprintf("launch_template_%s", templateName), duration)
}

// GetLaunchStats returns launch performance statistics
func (lst *LaunchSpeedTracker) GetLaunchStats() LaunchStats {
	metrics := lst.monitor.GetMetrics()

	stats := LaunchStats{
		PhaseTimings:    make(map[string]float64),
		TemplateTimings: make(map[string]float64),
	}

	// Collect phase timings
	for name, metric := range metrics {
		if len(name) > 19 && name[:19] == "timing_launch_phase" {
			phase := name[20:]
			if len(metric.History) > 0 {
				// Get average from recent history
				recent := metric.History
				if len(recent) > 10 {
					recent = recent[len(recent)-10:]
				}
				total := 0.0
				for _, point := range recent {
					total += point.Value
				}
				stats.PhaseTimings[phase] = total / float64(len(recent))
			}
		}

		// Collect template timings
		if len(name) > 23 && name[:23] == "timing_launch_template_" {
			template := name[23:]
			if len(metric.History) > 0 {
				recent := metric.History
				if len(recent) > 10 {
					recent = recent[len(recent)-10:]
				}
				total := 0.0
				for _, point := range recent {
					total += point.Value
				}
				stats.TemplateTimings[template] = total / float64(len(recent))
			}
		}

		// Get total launch timing
		if name == "timing_launch_total" && len(metric.History) > 0 {
			recent := metric.History
			if len(recent) > 10 {
				recent = recent[len(recent)-10:]
			}
			total := 0.0
			for _, point := range recent {
				total += point.Value
			}
			stats.AverageTotalTime = total / float64(len(recent))
		}
	}

	return stats
}

// LaunchStats contains launch performance statistics
type LaunchStats struct {
	AverageTotalTime float64            `json:"average_total_time_ms"`
	PhaseTimings     map[string]float64 `json:"phase_timings_ms"`
	TemplateTimings  map[string]float64 `json:"template_timings_ms"`
}
