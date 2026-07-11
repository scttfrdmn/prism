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
//     a cumulative estimate (EffectiveComputeRate × running hours). The rate is the captured spot
//     rate for spot instances (#659), else the on-demand HourlyRate — a pure function of the instance
//     record, so accrual needs no AWS call.
//   - Billed reconciliation (authoritative, ~daily): OPT-IN (#644 follow-up). For a plain on-demand
//     instance the estimate already matches the bill to the penny (known rate × known time); with the
//     spot rate captured at launch (#659), spot is now a near-no-op too. Reconciliation's remaining
//     purpose is the cases where the estimate and the actual bill still diverge: Reserved Instances /
//     Savings Plans, credits/EDP/negotiated rates, and storage nuances (snapshots, provisioned IOPS,
//     transfer). When enabled and a live AWS
//     manager is present, reconcileInstance calls billedCost (awsManager.GetBilledCost) on the
//     reconcileInterval and replaces the estimate slice with the authoritative Cost Explorer figure
//     (reversal + billed events). OFF by default; skipped in test/reduced mode and when the
//     per-instance cost-allocation tag is inactive (keeps the estimate, warns).
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
		if inst.ProjectID == "" || !accruesSpend(inst) {
			continue
		}
		o.accrueInstance(ctx, inst, now)
		if o.reconcilesActive() {
			o.reconcileInstance(ctx, inst, now)
		}
	}
}

// accruesSpend reports whether an instance still incurs cost: a running instance (compute+storage),
// or a stopped one that still has a root volume (storage persists and bills while stopped, #652).
// Terminated instances accrue nothing.
func accruesSpend(inst types.Instance) bool {
	switch inst.State {
	case "terminated", "shutting-down":
		return false
	case "running":
		return true
	default: // stopped / stopping / hibernated — still billed for storage if a volume exists
		return inst.StorageGB > 0
	}
}

// accrueInstance appends the estimate delta for one instance if the accrual interval has elapsed.
func (o *spendObserver) accrueInstance(ctx context.Context, inst types.Instance, now time.Time) {
	scope := projectScope(inst.ProjectID)
	cp, _ := o.store.checkpoint(ctx, scope, inst.ID)

	if !cp.LastAccrualAt.IsZero() && now.Sub(cp.LastAccrualAt) < spendAccrualInterval {
		return // throttled
	}

	// Per-component cumulatives on independent clocks (compute stops when the instance stops;
	// storage bills for the whole lifetime). Deltas are each component's growth since the checkpoint.
	compCum, storeCum := estimateComponents(inst, now)
	deltaCompute := compCum - cp.LastComputeCum
	deltaStorage := storeCum - cp.LastStorageCum
	if deltaCompute < 0 {
		deltaCompute = 0 // guard against any non-monotonic estimate
	}
	if deltaStorage < 0 {
		deltaStorage = 0
	}
	delta := deltaCompute + deltaStorage
	if delta <= 0 {
		// No new cost — just advance the throttle timestamp.
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
		ResourceID:   inst.ID,
		Compute:      deltaCompute,
		Storage:      deltaStorage,
		// Network unmodeled (0).
	}
	if err := o.store.AppendSpend(ctx, scope, ev); err != nil {
		return // fail-open; retry next tick (checkpoint unchanged, so no lost spend)
	}
	cp.InstanceID = inst.ID
	cp.LastComputeCum = compCum
	cp.LastStorageCum = storeCum
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

	// Slices are measured in their own units from the last reconcile: the estimate accrued into the
	// ledger since then (current total estimate − estimate at last reconcile) vs. the authoritative
	// billed for the same window (billed − billed at last reconcile). Replacing one with the other
	// nets the ledger to billed without ever conflating estimate and billed dollars.
	compCum, storeCum := estimateComponents(inst, now)
	estimateNow := compCum + storeCum
	estimateSlice := estimateNow - cp.ReconciledEstimate
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
	// Advance BOTH reconcile baselines to this point (in their own units). The per-component accrual
	// baselines (LastComputeCum/LastStorageCum) are deliberately left untouched — the estimate feed
	// keeps accruing monotonically; reconciliation only overlays a correction on top of it.
	cp.InstanceID = inst.ID
	cp.ReconciledEstimate = estimateNow
	cp.ReconciledBilled = res.BilledTotal
	cp.LastReconcileAt = now
	_ = o.store.saveCheckpoint(ctx, scope, cp)
}

// ebsGBMonthRate is the standard gp3 EBS storage price ($/GB-month), mirroring the value used in
// pkg/aws/ami_cost_analyzer.go. Coarse but fine for the estimate feed — billed reconciliation
// corrects the authoritative total. hoursPerMonth converts it to $/GB-hour.
const (
	ebsGBMonthRate = 0.10
	hoursPerMonth  = 730.0
)

// estimateComponents is a self-contained, monotonic estimate of an instance's cumulative cost since
// launch, split into compute and storage on INDEPENDENT clocks (#652):
//
//   - compute = EffectiveComputeRate × running-hours — the captured spot rate for spot instances
//     (#659), else the on-demand HourlyRate. Compute is billed only while the instance is running,
//     so it stops accruing when the instance stops (from StateHistory).
//   - storage = RootVolumeGB × ($/GB-hour) × total-hours — EBS persists and bills for the whole
//     lifetime, running or stopped.
//
// This mirrors pkg/aws/manager.go:calculateActualCosts (running-hours compute vs total-hours
// storage) but stays self-contained (no pkg/aws dependency), recomputing from the instance record so
// successive observations difference into clean per-component deltas.
func estimateComponents(inst types.Instance, now time.Time) (compute, storage float64) {
	if inst.LaunchTime.IsZero() {
		return 0, 0
	}
	totalHours := now.Sub(inst.LaunchTime).Hours()
	if totalHours <= 0 {
		return 0, 0
	}
	if rate := inst.EffectiveComputeRate(); rate > 0 {
		// EffectiveComputeRate is the captured spot rate for spot instances (#659), else on-demand.
		compute = rate * runningHours(inst, now)
	}
	if inst.StorageGB > 0 {
		storage = inst.StorageGB * (ebsGBMonthRate / hoursPerMonth) * totalHours
	}
	return compute, storage
}

// runningHours sums the time the instance has spent in the "running" state since launch, from its
// StateHistory. When there's no history, it falls back to total elapsed time if currently running,
// else 0 (a stopped instance with no history accrues no compute). Storage does not use this — it
// bills for the full lifetime regardless of state.
func runningHours(inst types.Instance, now time.Time) float64 {
	if len(inst.StateHistory) == 0 {
		if inst.State == "running" {
			return now.Sub(inst.LaunchTime).Hours()
		}
		return 0
	}
	var hours float64
	// Walk transitions; accumulate time spent in "running" between each transition and the next.
	for i, tr := range inst.StateHistory {
		if tr.ToState != "running" {
			continue
		}
		end := now
		if i+1 < len(inst.StateHistory) {
			end = inst.StateHistory[i+1].Timestamp
		}
		if end.After(tr.Timestamp) {
			hours += end.Sub(tr.Timestamp).Hours()
		}
	}
	return hours
}
