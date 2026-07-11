package daemon

import (
	"context"
	"sort"
	"time"

	be "github.com/scttfrdmn/budgetengine"
	"github.com/scttfrdmn/prism/pkg/project"
	"github.com/scttfrdmn/prism/pkg/types"
)

// Phase 3d (#654): the cost-analytics surfaces (cost breakdown, resource usage, cost trends, monthly
// report, and BudgetStatus.BurnRate) are aggregated here, in the daemon, from the budgetengine spend
// ledger — replacing the Phase 3c stubs that returned zeros. Everything derives from one primitive,
// ledgerCostSeries, a day-bucketed fold over s.spendLedger.Spend(projectScope(id)).
//
// The observer (#652) stamps each estimate SpendEvent with ResourceID = instance ID and an un-blended
// Compute/Storage split, so per-instance / per-day attribution is a pure fold — no AWS calls. Two
// caveats, both documented at their sites:
//   - billed / billed-reversal events (Cost Explorer reconciliation) carry NO ResourceID or component
//     split, so they fold into the day's project-level total only, not per-instance lines.
//   - instances that have since terminated are gone from state; their events fall back to the raw
//     ResourceID for a name and "unknown" for the type.

// ledgerCostSeries folds the project's spend ledger into an ascending, day-bucketed cost series over
// [start, end). Each point's TotalCost is the running cumulative spend (the burn-rate/history helpers
// difference TotalCost between points), and DailyCost is that day's spend. InstanceCosts/StorageCosts
// hold the per-instance lines for the day (estimate events only). Days with no accrual produce no
// point.
func (s *Server) ledgerCostSeries(ctx context.Context, projectID string, start, end time.Time) ([]project.CostDataPoint, error) {
	if s.spendLedger == nil || projectID == "" {
		return nil, nil
	}
	events, err := s.spendLedger.Spend(ctx, projectScope(projectID))
	if err != nil {
		return nil, err
	}
	if len(events) == 0 {
		return nil, nil
	}

	instByID := s.instancesByID()

	buckets := map[string]*dayBucket{}
	for _, ev := range events {
		if ev.At.Before(start) || !ev.At.Before(end) {
			continue
		}
		bucketForDay(buckets, ev.At.UTC().Truncate(24*time.Hour)).add(ev, instByID)
	}
	if len(buckets) == 0 {
		return nil, nil
	}
	return emitSeries(buckets), nil
}

// instancesByID loads the current instance map (ResourceID → instance) for naming/typing per-instance
// lines. Empty when no state manager is wired (unit tests) or state is unreadable.
func (s *Server) instancesByID() map[string]types.Instance {
	if s.stateManager == nil {
		return map[string]types.Instance{}
	}
	st, err := s.stateManager.LoadState()
	if err != nil {
		return map[string]types.Instance{}
	}
	return st.Instances
}

// dayBucket accumulates one UTC day's spend: the project-level dailyCost plus per-instance compute /
// storage lines (keyed by instance ID).
type dayBucket struct {
	day       time.Time
	dailyCost float64
	instances map[string]*types.InstanceCost
	storage   map[string]*types.StorageCost
}

// bucketForDay returns (creating if needed) the bucket for the given day.
func bucketForDay(buckets map[string]*dayBucket, day time.Time) *dayBucket {
	key := day.Format("2006-01-02")
	b := buckets[key]
	if b == nil {
		b = &dayBucket{
			day:       day,
			instances: map[string]*types.InstanceCost{},
			storage:   map[string]*types.StorageCost{},
		}
		buckets[key] = b
	}
	return b
}

// add folds one SpendEvent into the bucket. Amount is authoritative for the project total (spend > 0,
// reversals < 0). Only line-item-bearing events (estimate accruals, ResourceID set) attribute per
// instance; billed / billed-reversal events have no ResourceID and fold into dailyCost only.
func (b *dayBucket) add(ev be.SpendEvent, instByID map[string]types.Instance) {
	b.dailyCost += ev.Amount
	if ev.ResourceID == "" {
		return
	}

	ic := b.instances[ev.ResourceID]
	if ic == nil {
		name, typ := ev.ResourceID, "unknown"
		if inst, ok := instByID[ev.ResourceID]; ok {
			if inst.Name != "" {
				name = inst.Name
			}
			if inst.InstanceType != "" {
				typ = inst.InstanceType
			}
		}
		ic = &types.InstanceCost{InstanceName: name, InstanceType: typ}
		b.instances[ev.ResourceID] = ic
	}
	ic.ComputeCost += ev.Compute
	ic.StorageCost += ev.Storage
	ic.TotalCost += ev.Compute + ev.Storage

	if ev.Storage > 0 {
		b.storageLine(ev.ResourceID, instByID).Cost += ev.Storage
	}
}

// storageLine returns (creating if needed) the synthesized per-instance root-volume line. Events
// don't key by volume, so size comes from state when the instance still exists.
func (b *dayBucket) storageLine(resourceID string, instByID map[string]types.Instance) *types.StorageCost {
	sc := b.storage[resourceID]
	if sc != nil {
		return sc
	}
	name := resourceID + "-root"
	var sizeGB float64
	if inst, ok := instByID[resourceID]; ok {
		if inst.Name != "" {
			name = inst.Name + "-root"
		}
		sizeGB = inst.StorageGB
	}
	sc = &types.StorageCost{VolumeName: name, VolumeType: "gp3", SizeGB: sizeGB, CostPerGB: ebsGBMonthRate}
	b.storage[resourceID] = sc
	return sc
}

// emitSeries orders the buckets ascending by day and emits points with a running cumulative
// TotalCost (the burn-rate/history helpers difference TotalCost between points).
func emitSeries(buckets map[string]*dayBucket) []project.CostDataPoint {
	keys := make([]string, 0, len(buckets))
	for k := range buckets {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	series := make([]project.CostDataPoint, 0, len(keys))
	var cumulative float64
	for _, k := range keys {
		b := buckets[k]
		cumulative += b.dailyCost

		instances := make([]types.InstanceCost, 0, len(b.instances))
		for _, ic := range b.instances {
			instances = append(instances, *ic)
		}
		sort.Slice(instances, func(i, j int) bool { return instances[i].InstanceName < instances[j].InstanceName })

		storage := make([]types.StorageCost, 0, len(b.storage))
		for _, sc := range b.storage {
			storage = append(storage, *sc)
		}
		sort.Slice(storage, func(i, j int) bool { return storage[i].VolumeName < storage[j].VolumeName })

		series = append(series, project.CostDataPoint{
			Timestamp:     b.day,
			TotalCost:     cumulative,
			DailyCost:     b.dailyCost,
			InstanceCosts: instances,
			StorageCosts:  storage,
		})
	}
	return series
}

// costBreakdown aggregates the day-bucketed series across [start, end) into a single
// ProjectCostBreakdown: one InstanceCost per instance (component costs + running/stopped hours from
// StateHistory) and one synthesized per-instance StorageCost line.
func (s *Server) costBreakdown(ctx context.Context, projectID string, start, end time.Time) (*types.ProjectCostBreakdown, error) {
	series, err := s.ledgerCostSeries(ctx, projectID, start, end)
	if err != nil {
		return nil, err
	}

	instByID := s.instancesByID()

	// Aggregate per instance name (breakdown lines are keyed by name, matching the DTO) and per volume.
	instAgg := map[string]*types.InstanceCost{}
	storageAgg := map[string]*types.StorageCost{}
	var total float64
	for _, pt := range series {
		total += pt.DailyCost
		for _, ic := range pt.InstanceCosts {
			agg := instAgg[ic.InstanceName]
			if agg == nil {
				agg = &types.InstanceCost{InstanceName: ic.InstanceName, InstanceType: ic.InstanceType}
				instAgg[ic.InstanceName] = agg
			}
			agg.ComputeCost += ic.ComputeCost
			agg.StorageCost += ic.StorageCost
			agg.TotalCost += ic.TotalCost
		}
		for _, sc := range pt.StorageCosts {
			agg := storageAgg[sc.VolumeName]
			if agg == nil {
				agg = &types.StorageCost{VolumeName: sc.VolumeName, VolumeType: sc.VolumeType, SizeGB: sc.SizeGB, CostPerGB: sc.CostPerGB}
				storageAgg[sc.VolumeName] = agg
			}
			agg.Cost += sc.Cost
		}
	}

	// Fill running/stopped hours from each instance's StateHistory clipped to the window. Matched by
	// name (the breakdown key); instances gone from state simply report zero hours.
	windowHours := end.Sub(start).Hours()
	for _, inst := range instByID {
		agg := instAgg[inst.Name]
		if agg == nil {
			continue
		}
		rh := runningHoursInWindow(inst, start, end)
		agg.RunningHours = rh
		stopped := windowHours - rh
		if stopped < 0 {
			stopped = 0
		}
		agg.StoppedHours = stopped
		// HibernatedHours is not separately tracked from StateHistory; left 0 (best-effort).
	}

	instances := make([]types.InstanceCost, 0, len(instAgg))
	for _, ic := range instAgg {
		instances = append(instances, *ic)
	}
	sort.Slice(instances, func(i, j int) bool { return instances[i].InstanceName < instances[j].InstanceName })

	storage := make([]types.StorageCost, 0, len(storageAgg))
	for _, sc := range storageAgg {
		storage = append(storage, *sc)
	}
	sort.Slice(storage, func(i, j int) bool { return storage[i].VolumeName < storage[j].VolumeName })

	return &types.ProjectCostBreakdown{
		ProjectID:     projectID,
		TotalCost:     total,
		InstanceCosts: instances,
		StorageCosts:  storage,
		PeriodStart:   start,
		PeriodEnd:     end,
		GeneratedAt:   time.Now(),
	}, nil
}

// resourceUsage summarizes a project's fleet over the trailing period from local state + the ledger
// series. Compute/idle figures come from StateHistory (running vs. stopped hours).
func (s *Server) resourceUsage(ctx context.Context, projectID string, period time.Duration) (*types.ProjectResourceUsage, error) {
	now := time.Now()
	start := now.Add(-period)

	usage := &types.ProjectResourceUsage{
		ProjectID:         projectID,
		MeasurementPeriod: period,
		LastUpdated:       now,
	}

	if s.stateManager == nil {
		return usage, nil
	}
	st, err := s.stateManager.LoadState()
	if err != nil {
		return usage, nil // no state → empty usage, not an error (matches prior lenient behavior)
	}

	for _, inst := range st.Instances {
		if inst.ProjectID != projectID {
			continue
		}
		if inst.State == "terminated" || inst.State == "shutting-down" {
			continue
		}
		usage.TotalInstances++
		if inst.State == "running" {
			usage.ActiveInstances++
		}
		usage.TotalStorage += inst.StorageGB

		running := runningHoursInWindow(inst, start, now)
		usage.ComputeHours += running
		// IdleSavings: compute that WOULD have accrued had the instance run the whole period but did
		// not (i.e. the avoided on-demand cost of the stopped time).
		windowHours := now.Sub(start).Hours()
		stopped := windowHours - running
		if rate := inst.EffectiveComputeRate(); stopped > 0 && rate > 0 {
			// Avoided compute cost of the stopped time — spot rate for spot instances (#659).
			usage.IdleSavings += stopped * rate
		}
	}

	return usage, nil
}

// costTrends returns the per-period spending series in the shape the CLI history table renders
// (budget_commands.go outputHistoryTable): trends is a list of {date, spent, budget} objects. This is
// deliberately NOT the CostDataPoint JSON shape — the pre-3c stub emitted CostDataPoint, which the CLI
// never parsed, so the history table was always empty. period is "7d" / "30d" / "90d".
func (s *Server) costTrends(ctx context.Context, projectID, period string) (map[string]interface{}, error) {
	days := 30
	switch period {
	case "7d":
		days = 7
	case "90d":
		days = 90
	}
	now := time.Now()
	start := now.AddDate(0, 0, -days)

	series, err := s.ledgerCostSeries(ctx, projectID, start, now)
	if err != nil {
		return nil, err
	}

	// Daily allocation for the budget column: MonthlyLimit/30 when set, else TotalBudget prorated over
	// the window, else 0. Best-effort — the CLI uses it only for the usage% and bar scaling.
	dailyBudget := s.dailyAllocation(ctx, projectID, days)

	trends := make([]map[string]interface{}, 0, len(series))
	for _, pt := range series {
		trends = append(trends, map[string]interface{}{
			"date":   pt.Timestamp.Format("2006-01-02"),
			"spent":  pt.DailyCost,
			"budget": dailyBudget,
		})
	}

	return map[string]interface{}{
		"project_id": projectID,
		"period":     period,
		"days":       days,
		"trends":     trends,
		"count":      len(trends),
	}, nil
}

// dailyAllocation estimates a per-day budget figure for the trends "budget" column. MonthlyLimit
// divided by 30 when present; otherwise the total budget spread across the requested window; 0 when
// the project has no budget.
func (s *Server) dailyAllocation(ctx context.Context, projectID string, days int) float64 {
	proj, err := s.projectManager.GetProject(ctx, projectID)
	if err != nil || proj.Budget == nil {
		return 0
	}
	if proj.Budget.MonthlyLimit != nil && *proj.Budget.MonthlyLimit > 0 {
		return *proj.Budget.MonthlyLimit / 30.0
	}
	if proj.Budget.TotalBudget > 0 && days > 0 {
		return proj.Budget.TotalBudget / float64(days)
	}
	return 0
}

// runningHoursInWindow sums the time an instance spent in the "running" state within [start, end),
// from its StateHistory. Mirrors spend_observer.go's runningHours but clips to an explicit window so
// the breakdown/usage reflect only the requested period. With no history it falls back to the full
// window if currently running, else 0.
func runningHoursInWindow(inst types.Instance, start, end time.Time) float64 {
	clip := func(a, b time.Time) float64 {
		if a.Before(start) {
			a = start
		}
		if b.After(end) {
			b = end
		}
		if b.After(a) {
			return b.Sub(a).Hours()
		}
		return 0
	}

	if len(inst.StateHistory) == 0 {
		if inst.State == "running" {
			base := inst.LaunchTime
			if base.IsZero() || base.Before(start) {
				base = start
			}
			return clip(base, end)
		}
		return 0
	}

	var hours float64
	for i, tr := range inst.StateHistory {
		if tr.ToState != "running" {
			continue
		}
		segEnd := end
		if i+1 < len(inst.StateHistory) {
			segEnd = inst.StateHistory[i+1].Timestamp
		}
		hours += clip(tr.Timestamp, segEnd)
	}
	return hours
}
