package daemon

import (
	"testing"
	"time"
)

// TestCheckAndRecordLaunches_AdmitsBatch confirms a job array of N is admitted as
// a unit even when N exceeds maxLaunches — the batch limiter guards against a
// saturated window, not against an intentional array being larger than the
// per-launch accidental-runaway limit.
func TestCheckAndRecordLaunches_AdmitsBatch(t *testing.T) {
	rl := NewRateLimiter(2, time.Minute) // per-launch limit of 2/min

	if err := rl.CheckAndRecordLaunches(5); err != nil {
		t.Fatalf("batch of 5 should be admitted on an empty window, got: %v", err)
	}

	// All 5 timestamps should now be recorded so the window reflects the footprint.
	if got := rl.GetStatus().Current; got != 5 {
		t.Errorf("recorded launches = %d; want 5", got)
	}
}

// TestCheckAndRecordLaunches_RejectsWhenWindowSaturated confirms that once the
// window is already at the limit, a subsequent batch is rejected (retry later).
func TestCheckAndRecordLaunches_RejectsWhenWindowSaturated(t *testing.T) {
	rl := NewRateLimiter(2, time.Minute)

	// Saturate the window with single launches.
	if err := rl.CheckAndRecordLaunch(); err != nil {
		t.Fatalf("first launch: %v", err)
	}
	if err := rl.CheckAndRecordLaunch(); err != nil {
		t.Fatalf("second launch: %v", err)
	}

	err := rl.CheckAndRecordLaunches(3)
	if err == nil {
		t.Fatal("batch should be rejected when the window is already saturated")
	}
	if _, ok := err.(*RateLimitError); !ok {
		t.Errorf("expected *RateLimitError, got %T", err)
	}
}

// TestCheckAndRecordLaunches_DelegatesForSingle confirms n<=1 behaves exactly like
// the per-launch limiter (so single launches still hit the accidental-runaway guard).
func TestCheckAndRecordLaunches_DelegatesForSingle(t *testing.T) {
	rl := NewRateLimiter(1, time.Minute)

	if err := rl.CheckAndRecordLaunches(1); err != nil {
		t.Fatalf("first single launch should pass: %v", err)
	}
	if err := rl.CheckAndRecordLaunches(1); err == nil {
		t.Fatal("second single launch should be rate-limited (limit 1/min)")
	}
}

// TestCheckAndRecordLaunches_DisabledAlwaysAllows confirms a disabled limiter
// admits any batch.
func TestCheckAndRecordLaunches_DisabledAlwaysAllows(t *testing.T) {
	rl := NewRateLimiter(1, time.Minute)
	rl.SetEnabled(false)

	if err := rl.CheckAndRecordLaunches(100); err != nil {
		t.Fatalf("disabled limiter should admit any batch, got: %v", err)
	}
}
