package daemon

import (
	"context"
	"testing"
	"time"

	be "github.com/scttfrdmn/budgetengine"
	"github.com/scttfrdmn/prism/pkg/project"
	"github.com/scttfrdmn/prism/pkg/seam/filestore"
	"github.com/scttfrdmn/prism/pkg/types"
)

// newTestBudgetServer builds a Server with a file-backed project manager + spend ledger (no AWS), the
// minimum for exercising the Phase 3c engine-derived budgetStatus.
func newTestBudgetServer(t *testing.T) *Server {
	t.Helper()
	dir := t.TempDir()
	pm, err := project.NewManagerWithStore(filestore.New[types.Project](dir + "/projects"))
	if err != nil {
		t.Fatalf("project manager: %v", err)
	}
	return &Server{
		projectManager: pm,
		spendLedger:    newTestSpendStore(t),
	}
}

// seedBudgetedProject creates a project carrying a monthly budget and returns its ID.
func seedBudgetedProject(t *testing.T, s *Server, total, monthlyLimit float64) string {
	t.Helper()
	ctx := context.Background()
	proj, err := s.projectManager.CreateProject(ctx, &project.CreateProjectRequest{
		Name:  "Budget Test",
		Owner: "tester@example.com",
		Budget: &project.CreateProjectBudgetRequest{
			TotalBudget:  total,
			MonthlyLimit: &monthlyLimit,
			BudgetPeriod: types.BudgetPeriodMonthly,
		},
	})
	if err != nil {
		t.Fatalf("create project: %v", err)
	}
	return proj.ID
}

// TestBudgetStatus_NoBudget: a project with no budget reports BudgetEnabled=false and no spend.
func TestBudgetStatus_NoBudget(t *testing.T) {
	s := newTestBudgetServer(t)
	ctx := context.Background()
	proj, err := s.projectManager.CreateProject(ctx, &project.CreateProjectRequest{
		Name:  "No Budget",
		Owner: "tester@example.com",
	})
	if err != nil {
		t.Fatalf("create project: %v", err)
	}

	st, err := s.budgetStatus(ctx, proj.ID)
	if err != nil {
		t.Fatalf("budgetStatus: %v", err)
	}
	if st.BudgetEnabled {
		t.Fatalf("BudgetEnabled = true, want false for a budget-less project")
	}
	if st.SpentAmount != 0 || st.TotalBudget != 0 {
		t.Fatalf("expected zero budget/spend, got spent=%.2f total=%.2f", st.SpentAmount, st.TotalBudget)
	}
}

// TestBudgetStatus_MissingProject: unknown project is an error (parity with the old tracker guard).
func TestBudgetStatus_MissingProject(t *testing.T) {
	s := newTestBudgetServer(t)
	if _, err := s.budgetStatus(context.Background(), "does-not-exist"); err == nil {
		t.Fatal("expected error for missing project, got nil")
	}
}

// TestBudgetStatus_SpentFromLedger: with a budget and ledger events, SpentAmount folds the ledger and
// Remaining/SpentPercentage derive from it — NOT from the (stale, unwritten) cached SpentAmount.
func TestBudgetStatus_SpentFromLedger(t *testing.T) {
	s := newTestBudgetServer(t)
	ctx := context.Background()
	projID := seedBudgetedProject(t, s, 1000, 1000)

	// Accrue $250 of real spend into the ledger for this project.
	scope := projectScope(projID)
	now := time.Now()
	_ = s.spendLedger.AppendSpend(ctx, scope, be.SpendEvent{
		ID: "e1", AllocationID: engineAllocationID, Amount: 150, At: now.Add(-2 * time.Hour), Source: "estimate",
	})
	_ = s.spendLedger.AppendSpend(ctx, scope, be.SpendEvent{
		ID: "e2", AllocationID: engineAllocationID, Amount: 100, At: now.Add(-time.Hour), Source: "estimate",
	})

	st, err := s.budgetStatus(ctx, projID)
	if err != nil {
		t.Fatalf("budgetStatus: %v", err)
	}
	if !st.BudgetEnabled {
		t.Fatal("BudgetEnabled = false, want true")
	}
	if st.SpentAmount < 249.9 || st.SpentAmount > 250.1 {
		t.Fatalf("SpentAmount = %.2f, want ~250 (ledger fold)", st.SpentAmount)
	}
	if st.RemainingBudget < 749.9 || st.RemainingBudget > 750.1 {
		t.Fatalf("RemainingBudget = %.2f, want ~750", st.RemainingBudget)
	}
	if st.SpentPercentage < 0.249 || st.SpentPercentage > 0.251 {
		t.Fatalf("SpentPercentage = %.4f, want ~0.25", st.SpentPercentage)
	}
}

// TestBudgetStatus_EmptyLedgerFallsBackToCached: with no ledger events, spend falls back to the
// project's cached SpentAmount so a warming-up ledger never under-reports.
func TestBudgetStatus_EmptyLedgerFallsBackToCached(t *testing.T) {
	s := newTestBudgetServer(t)
	ctx := context.Background()
	projID := seedBudgetedProject(t, s, 1000, 1000)

	// Force a cached SpentAmount on the stored project (no ledger events).
	proj, _ := s.projectManager.GetProject(ctx, projID)
	proj.Budget.SpentAmount = 400
	if err := s.projectManager.SetProjectBudget(ctx, projID, proj.Budget); err != nil {
		t.Fatalf("set budget: %v", err)
	}

	st, err := s.budgetStatus(ctx, projID)
	if err != nil {
		t.Fatalf("budgetStatus: %v", err)
	}
	if st.SpentAmount < 399.9 || st.SpentAmount > 400.1 {
		t.Fatalf("SpentAmount = %.2f, want ~400 (cached fallback)", st.SpentAmount)
	}
}

// TestBudgetStatus_OverspentClampsRemaining: spend beyond the budget clamps RemainingBudget at 0.
func TestBudgetStatus_OverspentClampsRemaining(t *testing.T) {
	s := newTestBudgetServer(t)
	ctx := context.Background()
	projID := seedBudgetedProject(t, s, 100, 100)

	_ = s.spendLedger.AppendSpend(ctx, projectScope(projID), be.SpendEvent{
		ID: "e1", AllocationID: engineAllocationID, Amount: 150, At: time.Now().Add(-time.Hour), Source: "estimate",
	})

	st, err := s.budgetStatus(ctx, projID)
	if err != nil {
		t.Fatalf("budgetStatus: %v", err)
	}
	if st.RemainingBudget != 0 {
		t.Fatalf("RemainingBudget = %.2f, want 0 (clamped)", st.RemainingBudget)
	}
}

// TestBudgetStatus_SurplusForPeriodicBudget (Phase 3e, #655): a periodic (monthly) budget with a
// seeded ledger gets a populated Surplus. StartDate = start of the current month so there are no
// completed prior periods — isolating CurrentPeriodSurplus = allocation − spent.
func TestBudgetStatus_SurplusForPeriodicBudget(t *testing.T) {
	s := newTestBudgetServer(t)
	ctx := context.Background()
	projID := seedBudgetedProject(t, s, 1000, 1000)

	// Anchor StartDate to the first of the current month; enable rollover (unlimited cap).
	now := time.Now().UTC()
	monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	proj, _ := s.projectManager.GetProject(ctx, projID)
	proj.Budget.StartDate = monthStart
	proj.Budget.RolloverEnabled = true
	if err := s.projectManager.SetProjectBudget(ctx, projID, proj.Budget); err != nil {
		t.Fatalf("set budget: %v", err)
	}

	// Spend $200 across two days this month (need >=2 points for the series).
	_ = s.spendLedger.AppendSpend(ctx, projectScope(projID), be.SpendEvent{
		ID: "e1", AllocationID: engineAllocationID, Amount: 120, At: monthStart.Add(24 * time.Hour), Source: "estimate", ResourceID: "i-1", Compute: 120,
	})
	_ = s.spendLedger.AppendSpend(ctx, projectScope(projID), be.SpendEvent{
		ID: "e2", AllocationID: engineAllocationID, Amount: 80, At: monthStart.Add(48 * time.Hour), Source: "estimate", ResourceID: "i-1", Compute: 80,
	})

	st, err := s.budgetStatus(ctx, projID)
	if err != nil {
		t.Fatalf("budgetStatus: %v", err)
	}
	if st.Surplus == nil {
		t.Fatal("Surplus should be populated for a periodic budget")
	}
	// Allocation falls back to TotalBudget (1000) since MonthlyAmount isn't set. PeriodSpend differences
	// the cumulative series endpoints within the window (200 − 120 = 80; the first in-window point is
	// the baseline — the established BurnRateCalculator.PeriodSpend behavior), so surplus = 1000 − 80.
	if st.Surplus.CurrentPeriodSurplus < 919 || st.Surplus.CurrentPeriodSurplus > 921 {
		t.Fatalf("CurrentPeriodSurplus = %.2f, want ~920", st.Surplus.CurrentPeriodSurplus)
	}
	// No completed prior periods → banked 0; effective balance = allocation + banked (0).
	if st.Surplus.BankedSurplus != 0 {
		t.Fatalf("BankedSurplus = %.2f, want 0 (no completed prior periods)", st.Surplus.BankedSurplus)
	}
	if st.Surplus.EffectiveBalance < 999 || st.Surplus.EffectiveBalance > 1001 {
		t.Fatalf("EffectiveBalance = %.2f, want ~1000", st.Surplus.EffectiveBalance)
	}
}

// TestBudgetStatus_NoSurplusForProjectLifetime (Phase 3e, #655): a project-lifetime budget does not
// bank across periods, so Surplus stays nil.
func TestBudgetStatus_NoSurplusForProjectLifetime(t *testing.T) {
	s := newTestBudgetServer(t)
	ctx := context.Background()
	proj, err := s.projectManager.CreateProject(ctx, &project.CreateProjectRequest{
		Name:  "Lifetime Budget",
		Owner: "tester@example.com",
		Budget: &project.CreateProjectBudgetRequest{
			TotalBudget:  1000,
			BudgetPeriod: types.BudgetPeriodProject,
		},
	})
	if err != nil {
		t.Fatalf("create project: %v", err)
	}
	now := time.Now()
	_ = s.spendLedger.AppendSpend(ctx, projectScope(proj.ID), be.SpendEvent{
		ID: "e1", AllocationID: engineAllocationID, Amount: 100, At: now.Add(-48 * time.Hour), Source: "estimate", ResourceID: "i-1", Compute: 100,
	})
	_ = s.spendLedger.AppendSpend(ctx, projectScope(proj.ID), be.SpendEvent{
		ID: "e2", AllocationID: engineAllocationID, Amount: 50, At: now.Add(-24 * time.Hour), Source: "estimate", ResourceID: "i-1", Compute: 50,
	})

	st, err := s.budgetStatus(ctx, proj.ID)
	if err != nil {
		t.Fatalf("budgetStatus: %v", err)
	}
	if st.Surplus != nil {
		t.Fatalf("Surplus should be nil for a project-lifetime budget, got %+v", st.Surplus)
	}
}
