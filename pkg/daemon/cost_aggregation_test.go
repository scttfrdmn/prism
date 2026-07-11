package daemon

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	be "github.com/scttfrdmn/budgetengine"
	"github.com/scttfrdmn/prism/pkg/project"
	"github.com/scttfrdmn/prism/pkg/seam/filestore"
	"github.com/scttfrdmn/prism/pkg/state"
	"github.com/scttfrdmn/prism/pkg/types"
)

// newTestCostServer builds a Server with a file-backed project manager, spend ledger, and state
// manager (all under temp dirs; no AWS) — enough to exercise the Phase 3d ledger aggregators.
func newTestCostServer(t *testing.T, instances ...types.Instance) *Server {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("PRISM_STATE_DIR", dir)

	pm, err := project.NewManagerWithStore(filestore.New[types.Project](dir + "/projects"))
	if err != nil {
		t.Fatalf("project manager: %v", err)
	}
	sm, err := state.NewManager()
	if err != nil {
		t.Fatalf("state manager: %v", err)
	}
	if len(instances) > 0 {
		st, _ := sm.LoadState()
		if st.Instances == nil {
			st.Instances = map[string]types.Instance{}
		}
		for _, inst := range instances {
			st.Instances[inst.ID] = inst
		}
		if err := sm.SaveState(st); err != nil {
			t.Fatalf("save state: %v", err)
		}
	}
	return &Server{
		projectManager: pm,
		spendLedger:    newTestSpendStore(t),
		stateManager:   sm,
	}
}

// seedEstimate appends an estimate SpendEvent (carrying the per-resource line-item split) at `at`.
func seedEstimate(t *testing.T, s *Server, projectID, instanceID string, compute, storage float64, at time.Time) {
	t.Helper()
	ev := be.SpendEvent{
		ID:           instanceID + ":" + at.Format(time.RFC3339Nano),
		AllocationID: engineAllocationID,
		Amount:       compute + storage,
		At:           at,
		Source:       "estimate",
		ResourceID:   instanceID,
		Compute:      compute,
		Storage:      storage,
	}
	if err := s.spendLedger.AppendSpend(context.Background(), projectScope(projectID), ev); err != nil {
		t.Fatalf("append estimate: %v", err)
	}
}

func runningInst(id, name, typ, projectID string, hourly, storageGB float64, launched time.Time) types.Instance {
	return types.Instance{
		ID: id, Name: name, InstanceType: typ, ProjectID: projectID, State: "running",
		HourlyRate: hourly, StorageGB: storageGB, LaunchTime: launched,
	}
}

// TestLedgerCostSeries_DayBucketsAndCumulative: estimate events across 2 days / 2 instances produce
// ascending day buckets with cumulative TotalCost, per-day DailyCost, and a per-instance split that
// sums to the instance total.
func TestLedgerCostSeries_DayBucketsAndCumulative(t *testing.T) {
	launched := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	s := newTestCostServer(t,
		runningInst("i-1", "web", "m7i.large", "proj", 0.10, 100, launched),
		runningInst("i-2", "db", "r7i.xlarge", "proj", 0.20, 200, launched),
	)
	day1 := time.Date(2026, 7, 2, 6, 0, 0, 0, time.UTC)
	day2 := time.Date(2026, 7, 3, 6, 0, 0, 0, time.UTC)
	seedEstimate(t, s, "proj", "i-1", 10, 1, day1)
	seedEstimate(t, s, "proj", "i-2", 20, 2, day1)
	seedEstimate(t, s, "proj", "i-1", 5, 0.5, day2)

	series, err := s.ledgerCostSeries(context.Background(), "proj",
		time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC), time.Date(2026, 7, 10, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("series: %v", err)
	}
	if len(series) != 2 {
		t.Fatalf("want 2 day buckets, got %d", len(series))
	}
	// Day 1: 10+1+20+2 = 33; Day 2: 5+0.5 = 5.5.
	if d := series[0].DailyCost; d < 32.99 || d > 33.01 {
		t.Fatalf("day1 DailyCost = %.2f, want 33", d)
	}
	if d := series[1].DailyCost; d < 5.49 || d > 5.51 {
		t.Fatalf("day2 DailyCost = %.2f, want 5.5", d)
	}
	// TotalCost is cumulative: day2 == day1 + day2 daily.
	if tc := series[1].TotalCost; tc < 38.49 || tc > 38.51 {
		t.Fatalf("day2 TotalCost = %.2f, want cumulative 38.5", tc)
	}
	if !series[0].Timestamp.Before(series[1].Timestamp) {
		t.Fatal("series not ascending by day")
	}
	// Day 1 has two instance lines; each Compute+Storage == TotalCost.
	if len(series[0].InstanceCosts) != 2 {
		t.Fatalf("day1 want 2 instance lines, got %d", len(series[0].InstanceCosts))
	}
	for _, ic := range series[0].InstanceCosts {
		if diff := ic.TotalCost - (ic.ComputeCost + ic.StorageCost); diff > 1e-9 || diff < -1e-9 {
			t.Fatalf("%s TotalCost %.4f != Compute+Storage %.4f", ic.InstanceName, ic.TotalCost, ic.ComputeCost+ic.StorageCost)
		}
	}
}

// TestCostBreakdown_AggregatesWindow: the breakdown sums per-instance component costs over the window,
// synthesizes per-instance storage lines, and names/types instances from state.
func TestCostBreakdown_AggregatesWindow(t *testing.T) {
	launched := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	s := newTestCostServer(t, runningInst("i-1", "web", "m7i.large", "proj", 0.10, 100, launched))
	seedEstimate(t, s, "proj", "i-1", 10, 1, time.Date(2026, 7, 2, 6, 0, 0, 0, time.UTC))
	seedEstimate(t, s, "proj", "i-1", 5, 0.5, time.Date(2026, 7, 3, 6, 0, 0, 0, time.UTC))

	bd, err := s.costBreakdown(context.Background(), "proj",
		time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC), time.Date(2026, 7, 10, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("breakdown: %v", err)
	}
	if tc := bd.TotalCost; tc < 16.49 || tc > 16.51 {
		t.Fatalf("TotalCost = %.2f, want 16.5", tc)
	}
	if len(bd.InstanceCosts) != 1 {
		t.Fatalf("want 1 instance line, got %d", len(bd.InstanceCosts))
	}
	ic := bd.InstanceCosts[0]
	if ic.InstanceName != "web" || ic.InstanceType != "m7i.large" {
		t.Fatalf("instance identity = %q/%q, want web/m7i.large", ic.InstanceName, ic.InstanceType)
	}
	if ic.ComputeCost < 14.99 || ic.ComputeCost > 15.01 {
		t.Fatalf("ComputeCost = %.2f, want 15", ic.ComputeCost)
	}
	if len(bd.StorageCosts) != 1 || bd.StorageCosts[0].VolumeName != "web-root" {
		t.Fatalf("want 1 synthesized storage line web-root, got %+v", bd.StorageCosts)
	}
	if bd.StorageCosts[0].SizeGB != 100 {
		t.Fatalf("storage SizeGB = %.0f, want 100", bd.StorageCosts[0].SizeGB)
	}
}

// TestResourceUsage_CountsAndCompute: usage reflects active/total counts, storage, and compute hours.
func TestResourceUsage_CountsAndCompute(t *testing.T) {
	launched := time.Now().Add(-48 * time.Hour)
	running := runningInst("i-1", "web", "m7i.large", "proj", 0.10, 100, launched)
	stopped := runningInst("i-2", "db", "r7i.xlarge", "proj", 0.20, 200, launched)
	stopped.State = "stopped"
	other := runningInst("i-3", "x", "m7i.large", "other-proj", 0.10, 50, launched)
	s := newTestCostServer(t, running, stopped, other)

	usage, err := s.resourceUsage(context.Background(), "proj", 24*time.Hour)
	if err != nil {
		t.Fatalf("usage: %v", err)
	}
	if usage.TotalInstances != 2 {
		t.Fatalf("TotalInstances = %d, want 2 (other-proj excluded)", usage.TotalInstances)
	}
	if usage.ActiveInstances != 1 {
		t.Fatalf("ActiveInstances = %d, want 1", usage.ActiveInstances)
	}
	if usage.TotalStorage != 300 {
		t.Fatalf("TotalStorage = %.0f, want 300", usage.TotalStorage)
	}
	// Running instance (no StateHistory) accrues ~24h of compute over the 24h window; stopped accrues 0.
	if usage.ComputeHours < 23.9 || usage.ComputeHours > 24.1 {
		t.Fatalf("ComputeHours = %.2f, want ~24", usage.ComputeHours)
	}
}

// TestCostTrends_CLIShape: trends elements carry exactly date/spent/budget, and days/count match.
func TestCostTrends_CLIShape(t *testing.T) {
	launched := time.Now().Add(-72 * time.Hour)
	s := newTestCostServer(t, runningInst("i-1", "web", "m7i.large", "proj", 0.10, 100, launched))
	// Give the project a budget so the "budget" column is non-zero.
	ml := 300.0
	if err := s.projectManager.SetProjectBudget(context.Background(), mustProject(t, s, "proj"),
		&types.ProjectBudget{TotalBudget: 300, MonthlyLimit: &ml, BudgetPeriod: types.BudgetPeriodMonthly}); err != nil {
		t.Fatalf("set budget: %v", err)
	}
	seedEstimate(t, s, mustProject(t, s, "proj"), "i-1", 10, 1, time.Now().Add(-24*time.Hour))

	trends, err := s.costTrends(context.Background(), mustProject(t, s, "proj"), "7d")
	if err != nil {
		t.Fatalf("trends: %v", err)
	}
	if trends["days"] != 7 {
		t.Fatalf("days = %v, want 7", trends["days"])
	}
	list, _ := trends["trends"].([]map[string]interface{})
	if len(list) != 1 {
		t.Fatalf("want 1 trend element, got %d", len(list))
	}
	// Round-trip through JSON to assert the exact wire keys the CLI reads.
	raw, _ := json.Marshal(list[0])
	var elem map[string]interface{}
	_ = json.Unmarshal(raw, &elem)
	for _, k := range []string{"date", "spent", "budget"} {
		if _, ok := elem[k]; !ok {
			t.Fatalf("trend element missing key %q; got %v", k, elem)
		}
	}
	if elem["spent"].(float64) < 10.9 || elem["spent"].(float64) > 11.1 {
		t.Fatalf("spent = %v, want ~11", elem["spent"])
	}
}

// mustProject creates a project named after the given ID marker and returns its real ID; a small
// helper so seed calls read naturally. It caches nothing — callers pass the returned ID.
func mustProject(t *testing.T, s *Server, marker string) string {
	t.Helper()
	// Reuse an existing project with this marker name if present (idempotent within a test).
	projects, _ := s.projectManager.ListProjects(context.Background(), nil)
	for _, p := range projects {
		if p.Name == marker {
			return p.ID
		}
	}
	p, err := s.projectManager.CreateProject(context.Background(), &project.CreateProjectRequest{
		Name: marker, Owner: "tester@example.com",
	})
	if err != nil {
		t.Fatalf("create project: %v", err)
	}
	return p.ID
}

// TestLedgerCostSeries_TerminatedInstanceFallback: an event whose instance is gone from state falls
// back to the raw ResourceID for a name and "unknown" for the type.
func TestLedgerCostSeries_TerminatedInstanceFallback(t *testing.T) {
	s := newTestCostServer(t) // no instances in state
	seedEstimate(t, s, "proj", "i-ghost", 7, 0, time.Date(2026, 7, 2, 6, 0, 0, 0, time.UTC))

	series, err := s.ledgerCostSeries(context.Background(), "proj",
		time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC), time.Date(2026, 7, 10, 0, 0, 0, 0, time.UTC))
	if err != nil || len(series) != 1 {
		t.Fatalf("series err=%v len=%d, want 1", err, len(series))
	}
	ic := series[0].InstanceCosts[0]
	if ic.InstanceName != "i-ghost" || ic.InstanceType != "unknown" {
		t.Fatalf("fallback identity = %q/%q, want i-ghost/unknown", ic.InstanceName, ic.InstanceType)
	}
}

// TestLedgerCostSeries_BilledFoldsIntoProjectTotal: billed / billed-reversal events (no ResourceID)
// affect the day's DailyCost but produce no per-instance line.
func TestLedgerCostSeries_BilledFoldsIntoProjectTotal(t *testing.T) {
	s := newTestCostServer(t, runningInst("i-1", "web", "m7i.large", "proj", 0.10, 100, time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)))
	day := time.Date(2026, 7, 2, 6, 0, 0, 0, time.UTC)
	seedEstimate(t, s, "proj", "i-1", 10, 0, day)
	// Reconciliation: reverse the estimate, add a billed figure — neither carries ResourceID.
	ctx := context.Background()
	_ = s.spendLedger.AppendSpend(ctx, projectScope("proj"), be.SpendEvent{ID: "rev", AllocationID: engineAllocationID, Amount: -10, At: day.Add(time.Hour), Source: "billed-reversal"})
	_ = s.spendLedger.AppendSpend(ctx, projectScope("proj"), be.SpendEvent{ID: "bill", AllocationID: engineAllocationID, Amount: 8, At: day.Add(time.Hour), Source: "billed"})

	series, err := s.ledgerCostSeries(ctx, "proj",
		time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC), time.Date(2026, 7, 10, 0, 0, 0, 0, time.UTC))
	if err != nil || len(series) != 1 {
		t.Fatalf("series err=%v len=%d, want 1", err, len(series))
	}
	// DailyCost nets to 10 - 10 + 8 = 8; the per-instance line reflects only the $10 estimate.
	if d := series[0].DailyCost; d < 7.99 || d > 8.01 {
		t.Fatalf("DailyCost = %.2f, want 8 (billed nets over estimate)", d)
	}
	if len(series[0].InstanceCosts) != 1 || series[0].InstanceCosts[0].ComputeCost < 9.99 {
		t.Fatalf("per-instance line should reflect the $10 estimate only, got %+v", series[0].InstanceCosts)
	}
}
