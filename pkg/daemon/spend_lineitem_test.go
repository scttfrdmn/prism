package daemon

import (
	"context"
	"testing"
	"time"

	"github.com/scttfrdmn/prism/pkg/types"
)

// instanceWithStorage builds a running instance carrying a root volume, for line-item tests.
func instanceWithStorage(id, projectID string, hourly, storageGB float64, launched time.Time) types.Instance {
	return types.Instance{
		ID: id, ProjectID: projectID, State: "running",
		HourlyRate: hourly, StorageGB: storageGB, LaunchTime: launched,
	}
}

// TestObserver_LineItemsSplitComputeStorage: a running instance with a root volume produces events
// carrying non-zero Compute AND Storage, ResourceID = instance ID, and Compute+Storage == Amount.
func TestObserver_LineItemsSplitComputeStorage(t *testing.T) {
	s := newTestSpendStore(t)
	launched := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	inst := instanceWithStorage("i-1", "proj-1", 2.0, 100, launched) // $2/hr compute, 100GB
	obs := newSpendObserver(s, func() ([]types.Instance, error) { return []types.Instance{inst}, nil }, true, spendObserverOptions{})

	obs.clock = func() time.Time { return launched.Add(10 * time.Hour) }
	obs.Observe(context.Background())

	evs, _ := s.Spend(context.Background(), projectScope("proj-1"))
	if len(evs) != 1 {
		t.Fatalf("want 1 event, got %d", len(evs))
	}
	e := evs[0]
	if e.ResourceID != "i-1" {
		t.Fatalf("ResourceID = %q, want i-1", e.ResourceID)
	}
	// compute = $2 × 10h = $20; storage = 100GB × $0.10/730 × 10h ≈ $0.137
	if e.Compute < 19.9 || e.Compute > 20.1 {
		t.Fatalf("Compute = %.4f, want ~20", e.Compute)
	}
	if e.Storage <= 0 {
		t.Fatalf("Storage = %.4f, want > 0", e.Storage)
	}
	if diff := e.Amount - (e.Compute + e.Storage); diff > 1e-9 || diff < -1e-9 {
		t.Fatalf("Amount (%.6f) != Compute+Storage (%.6f)", e.Amount, e.Compute+e.Storage)
	}
}

// TestObserver_StorageAccruesWhileStopped: a STOPPED instance with a volume still accrues storage
// (EBS bills while stopped) but zero compute — the independent-clock guarantee.
func TestObserver_StorageAccruesWhileStopped(t *testing.T) {
	s := newTestSpendStore(t)
	launched := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	inst := instanceWithStorage("i-1", "proj-1", 2.0, 100, launched)
	inst.State = "stopped" // no compute; storage persists
	obs := newSpendObserver(s, func() ([]types.Instance, error) { return []types.Instance{inst}, nil }, true, spendObserverOptions{})

	obs.clock = func() time.Time { return launched.Add(10 * time.Hour) }
	obs.Observe(context.Background())

	evs, _ := s.Spend(context.Background(), projectScope("proj-1"))
	if len(evs) != 1 {
		t.Fatalf("stopped instance with storage should accrue 1 event, got %d", len(evs))
	}
	e := evs[0]
	if e.Compute != 0 {
		t.Fatalf("Compute = %.4f, want 0 (instance stopped)", e.Compute)
	}
	if e.Storage <= 0 {
		t.Fatalf("Storage = %.4f, want > 0 (EBS bills while stopped)", e.Storage)
	}
	if e.Amount != e.Storage {
		t.Fatalf("Amount = %.4f, want == Storage %.4f", e.Amount, e.Storage)
	}
}

// TestObserver_StoppedNoStorageAccruesNothing: a stopped instance with no volume accrues nothing
// (matches the pre-3b skip behavior for the compute-only case).
func TestObserver_StoppedNoStorageAccruesNothing(t *testing.T) {
	s := newTestSpendStore(t)
	launched := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	inst := instanceWithStorage("i-1", "proj-1", 2.0, 0, launched)
	inst.State = "stopped"
	obs := newSpendObserver(s, func() ([]types.Instance, error) { return []types.Instance{inst}, nil }, true, spendObserverOptions{})
	obs.clock = func() time.Time { return launched.Add(10 * time.Hour) }
	obs.Observe(context.Background())

	evs, _ := s.Spend(context.Background(), projectScope("proj-1"))
	if len(evs) != 0 {
		t.Fatalf("stopped + no storage should accrue nothing, got %d events", len(evs))
	}
}

// TestObserver_ComputeStopsStorageContinues: running-hours from StateHistory — compute freezes at the
// stop transition while storage keeps growing on the total-hours clock.
func TestObserver_ComputeStopsStorageContinues(t *testing.T) {
	s := newTestSpendStore(t)
	launched := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	inst := instanceWithStorage("i-1", "proj-1", 2.0, 100, launched)
	inst.State = "stopped"
	// Ran for the first 4h, then stopped.
	inst.StateHistory = []types.StateTransition{
		{ToState: "running", Timestamp: launched},
		{ToState: "stopped", Timestamp: launched.Add(4 * time.Hour)},
	}
	obs := newSpendObserver(s, func() ([]types.Instance, error) { return []types.Instance{inst}, nil }, true, spendObserverOptions{})

	obs.clock = func() time.Time { return launched.Add(10 * time.Hour) } // 10h elapsed, 4h running
	obs.Observe(context.Background())

	evs, _ := s.Spend(context.Background(), projectScope("proj-1"))
	e := evs[0]
	// compute = $2 × 4 running-hours = $8 (NOT $20); storage = 100GB × rate × 10 total-hours.
	if e.Compute < 7.9 || e.Compute > 8.1 {
		t.Fatalf("Compute = %.4f, want ~8 (4 running-hours, not 10 wall-clock)", e.Compute)
	}
	if e.Storage <= 0 {
		t.Fatalf("Storage = %.4f, want > 0 (accrues full 10h)", e.Storage)
	}
}
