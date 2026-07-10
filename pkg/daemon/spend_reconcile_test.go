package daemon

import (
	"context"
	"testing"
	"time"

	"github.com/scttfrdmn/prism/pkg/types"
)

// billedSpy is an injectable billedCost func that records call count and returns a canned result.
type billedSpy struct {
	calls  int
	result *types.BilledCostResult
	err    error
}

func (s *billedSpy) fn(_ context.Context, _ /*name*/, _ /*region*/ string, _ time.Time) (*types.BilledCostResult, error) {
	s.calls++
	return s.result, s.err
}

func ledgerTotal(t *testing.T, s *spendStore, projectID string) float64 {
	t.Helper()
	evs, err := s.Spend(context.Background(), projectScope(projectID))
	if err != nil {
		t.Fatal(err)
	}
	var total float64
	for _, e := range evs {
		total += e.Amount
	}
	return total
}

// TestReconcile_DisabledNeverCallsCE is the opt-in guarantee: with reconciliation off (the default),
// GetBilledCost is never invoked, even with a live billedCost func present.
func TestReconcile_DisabledNeverCallsCE(t *testing.T) {
	s := newTestSpendStore(t)
	launched := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	insts := []types.Instance{runningInstance("i-1", "proj-1", 2.0, launched)}
	spy := &billedSpy{result: &types.BilledCostResult{BilledTotal: 5, TagActive: true}}

	// reconcileEnabled defaults false; even with billedCost set and past any interval, no CE call.
	obs := newSpendObserver(s, func() ([]types.Instance, error) { return insts, nil }, false,
		spendObserverOptions{billedCost: spy.fn, reconcileEnabled: false, reconcileInterval: time.Hour})
	obs.clock = func() time.Time { return launched.Add(48 * time.Hour) }
	obs.Observe(context.Background())

	if spy.calls != 0 {
		t.Fatalf("CE called %d times with reconciliation disabled, want 0", spy.calls)
	}
}

// TestReconcile_TestModeNeverCallsCE: test mode is an independent off-switch even when enabled.
func TestReconcile_TestModeNeverCallsCE(t *testing.T) {
	s := newTestSpendStore(t)
	launched := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	insts := []types.Instance{runningInstance("i-1", "proj-1", 2.0, launched)}
	spy := &billedSpy{result: &types.BilledCostResult{BilledTotal: 5, TagActive: true}}

	obs := newSpendObserver(s, func() ([]types.Instance, error) { return insts, nil }, true, // testMode
		spendObserverOptions{billedCost: spy.fn, reconcileEnabled: true, reconcileInterval: time.Hour})
	obs.clock = func() time.Time { return launched.Add(48 * time.Hour) }
	obs.Observe(context.Background())

	if spy.calls != 0 {
		t.Fatalf("CE called %d times in test mode, want 0", spy.calls)
	}
}

// TestReconcile_NilBilledCostNeverPanics: reduced mode (nil billedCost) with reconciliation enabled
// must not call/panic.
func TestReconcile_NilBilledCostSafe(t *testing.T) {
	s := newTestSpendStore(t)
	launched := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	insts := []types.Instance{runningInstance("i-1", "proj-1", 2.0, launched)}

	obs := newSpendObserver(s, func() ([]types.Instance, error) { return insts, nil }, false,
		spendObserverOptions{billedCost: nil, reconcileEnabled: true, reconcileInterval: time.Hour})
	obs.clock = func() time.Time { return launched.Add(48 * time.Hour) }
	obs.Observe(context.Background()) // must not panic
}

// TestReconcile_ReplacesEstimateWithBilled: after estimate accrual, an enabled reconcile nets the
// ledger to the authoritative BilledTotal (estimate slice reversed, billed slice added).
func TestReconcile_ReplacesEstimateWithBilled(t *testing.T) {
	s := newTestSpendStore(t)
	launched := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	insts := []types.Instance{runningInstance("i-1", "proj-1", 2.0, launched)} // $2/hr
	// AWS says the instance actually billed $15 (e.g. a discount/RI made it cheaper than the $20 est).
	spy := &billedSpy{result: &types.BilledCostResult{BilledTotal: 15, TagActive: true}}

	obs := newSpendObserver(s, func() ([]types.Instance, error) { return insts, nil }, false,
		spendObserverOptions{billedCost: spy.fn, reconcileEnabled: true, reconcileInterval: time.Hour})

	// t+10h: estimate accrues $20; then reconcile replaces it with billed $15.
	obs.clock = func() time.Time { return launched.Add(10 * time.Hour) }
	obs.Observe(context.Background())

	if spy.calls != 1 {
		t.Fatalf("CE calls = %d, want 1", spy.calls)
	}
	total := ledgerTotal(t, s, "proj-1")
	if total < 14.9 || total > 15.1 {
		t.Fatalf("ledger total = %.2f, want ~15 (billed replaces the $20 estimate)", total)
	}
}

// TestReconcile_TagInactiveKeepsEstimate: when the cost-allocation tag isn't active, do NOT reconcile
// (the region blob isn't per-instance) — the estimate stays.
func TestReconcile_TagInactiveKeepsEstimate(t *testing.T) {
	s := newTestSpendStore(t)
	launched := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	insts := []types.Instance{runningInstance("i-1", "proj-1", 2.0, launched)}
	spy := &billedSpy{result: &types.BilledCostResult{BilledTotal: 9999, TagActive: false}} // region blob

	obs := newSpendObserver(s, func() ([]types.Instance, error) { return insts, nil }, false,
		spendObserverOptions{billedCost: spy.fn, reconcileEnabled: true, reconcileInterval: time.Hour})
	obs.clock = func() time.Time { return launched.Add(10 * time.Hour) }
	obs.Observe(context.Background())

	total := ledgerTotal(t, s, "proj-1")
	if total < 19.9 || total > 20.1 {
		t.Fatalf("ledger total = %.2f, want ~20 (estimate kept; tag-inactive billed ignored)", total)
	}
}

// TestReconcile_Idempotent: a second reconcile at the same BilledTotal nets zero — no drift.
func TestReconcile_Idempotent(t *testing.T) {
	s := newTestSpendStore(t)
	launched := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	insts := []types.Instance{runningInstance("i-1", "proj-1", 2.0, launched)}
	spy := &billedSpy{result: &types.BilledCostResult{BilledTotal: 15, TagActive: true}}

	obs := newSpendObserver(s, func() ([]types.Instance, error) { return insts, nil }, false,
		spendObserverOptions{billedCost: spy.fn, reconcileEnabled: true, reconcileInterval: time.Hour})

	obs.clock = func() time.Time { return launched.Add(10 * time.Hour) }
	obs.Observe(context.Background())
	// 2h later (past the 1h reconcile interval): estimate would add ~$4 more, then reconcile back to
	// the SAME billed $15 → ledger returns to 15, no accumulation.
	obs.clock = func() time.Time { return launched.Add(12 * time.Hour) }
	obs.Observe(context.Background())

	total := ledgerTotal(t, s, "proj-1")
	if total < 14.9 || total > 15.1 {
		t.Fatalf("ledger total = %.2f, want ~15 (idempotent at same BilledTotal)", total)
	}
}
