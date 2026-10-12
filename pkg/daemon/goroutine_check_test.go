package daemon

import (
	"strings"
	"testing"

	"github.com/scttfrdmn/prism/pkg/monitoring"
)

// TestCheckGoroutineLeak_HealthyCountReportsNothing: the goroutine_check recovery operation called
// HandleGoroutineLeak -- the response to a leak, which always reports one -- so every check claimed a
// leak, dumped a 1MB stack trace and recorded a high-severity error regardless of the count.
func TestCheckGoroutineLeak_HealthyCountReportsNothing(t *testing.T) {
	sm := NewStabilityManager(monitoring.NewPerformanceMonitor())
	rm := NewRecoveryManager(sm, nil)

	if err := rm.CheckGoroutineLeak(); err != nil {
		t.Fatalf("CheckGoroutineLeak with a test-sized goroutine count = %v, want nil", err)
	}
	for _, e := range sm.GetErrorHistory() {
		if strings.Contains(e.ErrorType, "goroutine") || strings.Contains(e.Message, "goroutine") {
			t.Fatalf("a healthy check recorded a stability error: %+v", e)
		}
	}
}
