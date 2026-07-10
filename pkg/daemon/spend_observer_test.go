package daemon

import (
	"context"
	"testing"
	"time"

	be "github.com/scttfrdmn/budgetengine"
	"github.com/scttfrdmn/prism/pkg/seam/filestore"
	"github.com/scttfrdmn/prism/pkg/types"
)

// newTestSpendStore builds a spendStore over temp-dir filestores (host-free, no AWS).
func newTestSpendStore(t *testing.T) *spendStore {
	t.Helper()
	dir := t.TempDir()
	return newSpendStore(
		filestore.New[be.SpendEvent](dir+"/events"),
		filestore.New[spendCheckpoint](dir+"/checkpoints"),
	)
}

func runningInstance(id, projectID string, hourly float64, launched time.Time) types.Instance {
	return types.Instance{ID: id, ProjectID: projectID, State: "running", HourlyRate: hourly, LaunchTime: launched}
}

// TestSpendStore_AppendListRoundTrip: append events, read them back via the SpendSource port.
func TestSpendStore_AppendListRoundTrip(t *testing.T) {
	s := newTestSpendStore(t)
	ctx := context.Background()
	scope := projectScope("proj-1")
	_ = s.AppendSpend(ctx, scope, be.SpendEvent{ID: "e1", AllocationID: engineAllocationID, Amount: 10, At: time.Now()})
	_ = s.AppendSpend(ctx, scope, be.SpendEvent{ID: "e2", AllocationID: engineAllocationID, Amount: 5, At: time.Now()})

	evs, err := s.Spend(ctx, scope)
	if err != nil {
		t.Fatal(err)
	}
	if len(evs) != 2 {
		t.Fatalf("got %d events, want 2", len(evs))
	}
	var total float64
	for _, e := range evs {
		total += e.Amount
	}
	if total != 15 {
		t.Fatalf("total = %.2f, want 15", total)
	}
	// Scope isolation: a different project sees nothing.
	other, _ := s.Spend(ctx, projectScope("proj-2"))
	if len(other) != 0 {
		t.Fatalf("proj-2 leaked %d events", len(other))
	}
}

// TestObserver_CumulativeToDelta: successive observations append deltas that sum to the cumulative
// estimate — not the cumulative value each time (the double-count guard).
func TestObserver_CumulativeToDelta(t *testing.T) {
	s := newTestSpendStore(t)
	launched := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	inst := runningInstance("i-1", "proj-1", 2.0, launched) // $2/hr

	insts := []types.Instance{inst}
	obs := newSpendObserver(s, func() ([]types.Instance, error) { return insts, nil }, true)

	// t+10h → cumulative $20. Then t+25h → cumulative $50. Deltas: $20, then $30. Total $50.
	obs.clock = func() time.Time { return launched.Add(10 * time.Hour) }
	obs.Observe(context.Background())
	obs.clock = func() time.Time { return launched.Add(25 * time.Hour) }
	obs.Observe(context.Background())

	evs, _ := s.Spend(context.Background(), projectScope("proj-1"))
	var total float64
	for _, e := range evs {
		total += e.Amount
	}
	if total < 49.9 || total > 50.1 {
		t.Fatalf("ledger total = %.2f, want ~50 (cumulative at t+25h), events=%d", total, len(evs))
	}
	if len(evs) != 2 {
		t.Fatalf("want 2 delta events, got %d", len(evs))
	}
}

// TestObserver_Throttle: a second observation inside the accrual interval appends nothing.
func TestObserver_Throttle(t *testing.T) {
	s := newTestSpendStore(t)
	launched := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	insts := []types.Instance{runningInstance("i-1", "proj-1", 2.0, launched)}
	obs := newSpendObserver(s, func() ([]types.Instance, error) { return insts, nil }, true)

	obs.clock = func() time.Time { return launched.Add(10 * time.Hour) }
	obs.Observe(context.Background())
	// 1 minute later — well inside spendAccrualInterval (10m) → no new event.
	obs.clock = func() time.Time { return launched.Add(10*time.Hour + time.Minute) }
	obs.Observe(context.Background())

	evs, _ := s.Spend(context.Background(), projectScope("proj-1"))
	if len(evs) != 1 {
		t.Fatalf("throttle failed: %d events, want 1", len(evs))
	}
}

// TestObserver_IdempotentAcrossRestart: rebuilding the observer over the SAME store (simulating a
// daemon restart) does not re-append already-accrued spend, because the checkpoint is persisted.
func TestObserver_IdempotentAcrossRestart(t *testing.T) {
	s := newTestSpendStore(t)
	launched := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	insts := []types.Instance{runningInstance("i-1", "proj-1", 2.0, launched)}

	at := launched.Add(10 * time.Hour)
	obs1 := newSpendObserver(s, func() ([]types.Instance, error) { return insts, nil }, true)
	obs1.clock = func() time.Time { return at }
	obs1.Observe(context.Background())

	// "Restart": new observer, same store. Observe again 20 min later (past the throttle). The
	// persisted checkpoint means only the genuine delta since the checkpoint accrues — NOT the whole
	// cumulative again. At t+10h20m, cumulative = $2/hr × 10.333h ≈ $20.67, so the ledger total must
	// equal the new cumulative (~$20.67), proving no double-count of the first $20.
	obs2 := newSpendObserver(s, func() ([]types.Instance, error) { return insts, nil }, true)
	obs2.clock = func() time.Time { return at.Add(20 * time.Minute) }
	obs2.Observe(context.Background())

	evs, _ := s.Spend(context.Background(), projectScope("proj-1"))
	var total float64
	for _, e := range evs {
		total += e.Amount
	}
	// Ledger total == cumulative at t+10h20m (~$20.67), NOT $20 + $20.67 (~$40.67).
	if total < 20.5 || total > 20.8 {
		t.Fatalf("restart double-counted: total = %.2f, want ~20.67 (cumulative, not summed twice)", total)
	}
}

// TestObserver_SkipsNonRunningAndUnattributed: stopped or project-less instances accrue nothing.
func TestObserver_SkipsNonRunningAndUnattributed(t *testing.T) {
	s := newTestSpendStore(t)
	launched := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	stopped := runningInstance("i-stop", "proj-1", 2.0, launched)
	stopped.State = "stopped"
	noProj := runningInstance("i-np", "", 2.0, launched)
	insts := []types.Instance{stopped, noProj}
	obs := newSpendObserver(s, func() ([]types.Instance, error) { return insts, nil }, true)
	obs.clock = func() time.Time { return launched.Add(10 * time.Hour) }
	obs.Observe(context.Background())

	evs, _ := s.Spend(context.Background(), projectScope("proj-1"))
	if len(evs) != 0 {
		t.Fatalf("stopped/unattributed accrued %d events, want 0", len(evs))
	}
}
