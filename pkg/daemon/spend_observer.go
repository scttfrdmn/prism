package daemon

import (
	"context"
	"fmt"
	"time"

	be "github.com/scttfrdmn/budgetengine"
	"github.com/scttfrdmn/prism/pkg/types"
)

// Phase 2 (#644): the spend observer periodically accrues per-instance spend into the append-only
// budgetengine ledger (spendStore). It is the currently-greenfield loop that turns Prism's
// recompute-on-read cost model into a real event stream.
//
// Two accrual paths, at different cadences (both driven off StateMonitor's tick, internally
// throttled — Cost Explorer is far too costly/rate-limited to call per tick):
//
//   - Estimate (cheap, frequent): every accrualInterval, per running instance, append the delta of
//     a cumulative list-price estimate (HourlyRate × running hours). This is a pure function of the
//     instance record, so it needs no AWS call.
//   - Billed reconciliation (authoritative, ~daily): NOT YET IMPLEMENTED. The design and the
//     checkpoint carry fields for it (LastReconcileAt/ReconciledBilled, spendReconcileInterval), but
//     Phase 2 ships the estimate feed only. Reconciling against awsManager.GetBilledCost (Cost
//     Explorer) — with cumulative→delta correction and reversal semantics — lands as a follow-up so
//     the CE cost/rate-limit handling gets its own focused change. The reversal-based `Source:
//     "billed"` correction is where it will hook in.
//
// Cumulative→delta: sources are cumulative but the ledger is delta-append, so the observer keeps a
// per-instance checkpoint (last cumulative) and appends newCumulative − last, with a unique-per-delta
// event ID so a replay is a no-op in the fold.
//
// Attribution: Phase 2 is project-scoped — events are keyed by seam.Scope{Project} and carry
// AllocationID = engineAllocationID ("project"). Per-funding-allocation attribution is a later phase.

const (
	// spendAccrualInterval throttles estimate appends (billing granularity, not the 10s tick).
	spendAccrualInterval = 10 * time.Minute
	// spendReconcileInterval throttles authoritative Cost Explorer reconciliation.
	spendReconcileInterval = 24 * time.Hour
)

// spendObserver accrues spend for running instances into the ledger. It reads instances from the
// state manager and writes through the spendStore; it holds no mutable state of its own (the
// per-instance checkpoint lives in the seam, durable across restart).
type spendObserver struct {
	store    *spendStore
	loadInst func() ([]types.Instance, error) // returns current instances (from state manager)
	testMode bool                             // skip Cost Explorer entirely
	clock    func() time.Time                 // injectable for tests
}

func newSpendObserver(store *spendStore, loadInst func() ([]types.Instance, error), testMode bool) *spendObserver {
	return &spendObserver{store: store, loadInst: loadInst, testMode: testMode, clock: time.Now}
}

// Observe runs one accrual pass over all running instances. Called from StateMonitor's tick;
// per-instance throttling keeps it billing-appropriate despite the 10s cadence. Errors on a single
// instance are logged and skipped (fail-open) so one bad record never stalls the loop.
func (o *spendObserver) Observe(ctx context.Context) {
	if o.store == nil || o.loadInst == nil {
		return
	}
	instances, err := o.loadInst()
	if err != nil {
		return // no state; nothing to accrue
	}
	now := o.clock()
	for _, inst := range instances {
		if inst.ProjectID == "" || inst.State != "running" {
			continue // spend accrues only for running, project-attributed instances
		}
		o.accrueInstance(ctx, inst, now)
	}
}

// accrueInstance appends the estimate delta for one instance if the accrual interval has elapsed.
func (o *spendObserver) accrueInstance(ctx context.Context, inst types.Instance, now time.Time) {
	scope := projectScope(inst.ProjectID)
	cp, _ := o.store.checkpoint(ctx, scope, inst.ID)

	if !cp.LastAccrualAt.IsZero() && now.Sub(cp.LastAccrualAt) < spendAccrualInterval {
		return // throttled
	}

	cumulative := estimateCumulativeCost(inst, now)
	delta := cumulative - cp.LastCumulative
	if delta <= 0 {
		// No new cost (or a stale/decreasing estimate) — just advance the throttle timestamp.
		cp.LastAccrualAt = now
		_ = o.store.saveCheckpoint(ctx, scope, cp)
		return
	}

	ev := be.SpendEvent{
		ID:           fmt.Sprintf("%s:%d", inst.ID, now.UnixNano()), // unique per delta → idempotent
		AllocationID: engineAllocationID,
		Amount:       delta,
		At:           now,
		Source:       "estimate",
	}
	if err := o.store.AppendSpend(ctx, scope, ev); err != nil {
		return // fail-open; retry next tick (checkpoint unchanged, so no lost spend)
	}
	cp.InstanceID = inst.ID
	cp.LastCumulative = cumulative
	cp.LastAccrualAt = now
	_ = o.store.saveCheckpoint(ctx, scope, cp)
}

// estimateCumulativeCost is a self-contained, monotonic list-price estimate of an instance's spend
// since launch: HourlyRate × hours since LaunchTime. It intentionally does not read the possibly-
// stale cached CurrentSpend; it recomputes so successive observations produce a clean cumulative
// series the delta logic can difference. (Running-vs-stopped nuance and storage-only rates are a
// refinement for the billed-reconciliation path; this estimate is the coarse frequent feed.)
func estimateCumulativeCost(inst types.Instance, now time.Time) float64 {
	if inst.LaunchTime.IsZero() || inst.HourlyRate <= 0 {
		return 0
	}
	hours := now.Sub(inst.LaunchTime).Hours()
	if hours <= 0 {
		return 0
	}
	return inst.HourlyRate * hours
}
