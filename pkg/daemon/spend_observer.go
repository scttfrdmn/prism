package daemon

import (
	"context"
	"fmt"
	"log"
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
//   - Billed reconciliation (authoritative, ~daily): OPT-IN (#644 follow-up). When enabled via
//     config (CostReconciliationEnabled) and a live AWS manager is present, reconcileInstance calls
//     the injected billedCost (awsManager.GetBilledCost) on the reconcileInterval and replaces the
//     estimate slice with the authoritative Cost Explorer billed slice (reversal + billed events).
//     OFF by default; skipped in test/reduced mode and when the per-instance cost-allocation tag is
//     inactive (keeps the estimate, warns).
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

	// Cost Explorer reconciliation (#644), all off unless explicitly configured.
	// billedCost is injected (bound awsManager.GetBilledCost) so the CE path is unit-testable with a
	// spy and nil in reduced mode. reconcileEnabled is the opt-in flag; reconcileInterval throttles.
	billedCost        func(ctx context.Context, name, region string, since time.Time) (*types.BilledCostResult, error)
	reconcileEnabled  bool
	reconcileInterval time.Duration
}

// spendObserverOptions configures reconciliation (Phase 2 shipped without it; this is the #644
// follow-up). Zero value = estimate-only, matching prior behavior.
type spendObserverOptions struct {
	billedCost        func(ctx context.Context, name, region string, since time.Time) (*types.BilledCostResult, error)
	reconcileEnabled  bool
	reconcileInterval time.Duration
}

func newSpendObserver(store *spendStore, loadInst func() ([]types.Instance, error), testMode bool, opts spendObserverOptions) *spendObserver {
	interval := opts.reconcileInterval
	if interval <= 0 {
		interval = spendReconcileInterval
	}
	return &spendObserver{
		store:             store,
		loadInst:          loadInst,
		testMode:          testMode,
		clock:             time.Now,
		billedCost:        opts.billedCost,
		reconcileEnabled:  opts.reconcileEnabled,
		reconcileInterval: interval,
	}
}

// reconcilesActive reports whether the CE reconciliation branch may run: opt-in enabled, a live
// billed-cost func injected (nil in reduced mode), and not test mode. This is the single guard the
// reconcile path checks — opt-in is the outermost switch.
func (o *spendObserver) reconcilesActive() bool {
	return o.reconcileEnabled && o.billedCost != nil && !o.testMode
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
		if o.reconcilesActive() {
			o.reconcileInstance(ctx, inst, now)
		}
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

// reconcileInstance replaces the estimate slice accrued since the last reconciliation with the
// authoritative Cost Explorer billed slice, so the ledger converges on real dollars. Throttled to
// reconcileInterval. Only reachable when reconcilesActive() is true.
//
// Netting (all per-instance, since cp.ReconciledBilled — the last billed baseline):
//
//	estimateSlice = cp.LastCumulative − cp.ReconciledBilled   // estimate deltas already in the ledger
//	billedSlice   = res.BilledTotal   − cp.ReconciledBilled   // authoritative for the same window
//	append  −estimateSlice (source "billed-reversal")  and  +billedSlice (source "billed")
//	→ ledger total gains (billedSlice − estimateSlice), i.e. the estimate slice is replaced by billed.
//
// Unique event IDs make a repeat at the same BilledTotal a no-op delta (idempotent).
func (o *spendObserver) reconcileInstance(ctx context.Context, inst types.Instance, now time.Time) {
	scope := projectScope(inst.ProjectID)
	cp, _ := o.store.checkpoint(ctx, scope, inst.ID)

	if !cp.LastReconcileAt.IsZero() && now.Sub(cp.LastReconcileAt) < o.reconcileInterval {
		return // throttled
	}

	res, err := o.billedCost(ctx, inst.Name, inst.Region, inst.LaunchTime)
	if err != nil {
		return // fail-open; try again next interval (checkpoint unchanged)
	}
	if res == nil || !res.TagActive {
		// Region-fallback billed cost is not per-instance; reconciling against it would mis-attribute
		// another instance's spend. Keep the estimate; nudge the operator to activate the tag.
		log.Printf("Budget reconciliation: cost-allocation tag %q not active for %s; keeping estimate (activate it in AWS Billing for authoritative per-instance cost)", "prism:instance-id", inst.Name)
		cp.LastReconcileAt = now // still advance throttle so we don't hammer CE every tick
		_ = o.store.saveCheckpoint(ctx, scope, cp)
		return
	}

	estimateSlice := cp.LastCumulative - cp.ReconciledBilled
	billedSlice := res.BilledTotal - cp.ReconciledBilled
	if estimateSlice == 0 && billedSlice == 0 {
		cp.LastReconcileAt = now
		_ = o.store.saveCheckpoint(ctx, scope, cp)
		return // nothing to correct
	}

	if estimateSlice != 0 {
		_ = o.store.AppendSpend(ctx, scope, be.SpendEvent{
			ID:           fmt.Sprintf("%s:billed-rev:%d", inst.ID, now.UnixNano()),
			AllocationID: engineAllocationID,
			Amount:       -estimateSlice,
			At:           now,
			Source:       "billed-reversal",
		})
	}
	if err := o.store.AppendSpend(ctx, scope, be.SpendEvent{
		ID:           fmt.Sprintf("%s:billed:%d", inst.ID, now.UnixNano()),
		AllocationID: engineAllocationID,
		Amount:       billedSlice,
		At:           now,
		Source:       "billed",
	}); err != nil {
		return // fail-open; the reversal above is idempotent on retry via fresh IDs + baseline unchanged
	}
	cp.InstanceID = inst.ID
	cp.ReconciledBilled = res.BilledTotal
	cp.LastCumulative = res.BilledTotal // rebase the estimate baseline to billed so future estimate deltas accrue on top
	cp.LastReconcileAt = now
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
