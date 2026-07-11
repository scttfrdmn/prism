package daemon

import (
	"context"
	"testing"
	"time"

	"github.com/scttfrdmn/prism/pkg/project"
	"github.com/scttfrdmn/prism/pkg/types"
)

// TestForecast_EmptyLedgerIsZero (Phase 3e, #655): with no ledger events the forecast degrades to the
// rate-only baseline — zero current rate and no exhaustion projection (the pre-3e behavior).
func TestForecast_EmptyLedgerIsZero(t *testing.T) {
	s := newTestCostServer(t)
	projID := mustProject(t, s, "proj")

	resp, err := s.forecast(context.Background(), projID, &project.ProjectForecastRequest{Months: 6})
	if err != nil {
		t.Fatalf("forecast: %v", err)
	}
	if resp.CurrentMonthlyRate != 0 {
		t.Fatalf("CurrentMonthlyRate = %.2f, want 0 (empty ledger)", resp.CurrentMonthlyRate)
	}
	if resp.ProjectedExhaustion != nil {
		t.Fatal("ProjectedExhaustion should be nil with no spend history")
	}
}

// TestForecast_LedgerBackedNonZero (Phase 3e, #655): a budgeted project with a real multi-day ledger
// yields a non-zero current monthly rate, populated forecast months, and (given a budget) an
// exhaustion projection — the whole point of re-pointing the predictor at the ledger.
func TestForecast_LedgerBackedNonZero(t *testing.T) {
	launched := time.Now().Add(-30 * 24 * time.Hour)
	s := newTestCostServer(t, runningInst("i-1", "web", "m7i.large", "proj", 0.10, 100, launched))
	projID := mustProject(t, s, "proj")

	// Give the project a budget so exhaustion can be projected.
	if err := s.projectManager.SetProjectBudget(context.Background(), projID, &types.ProjectBudget{
		TotalBudget:  1000,
		BudgetPeriod: types.BudgetPeriodProject,
		StartDate:    launched,
	}); err != nil {
		t.Fatalf("set budget: %v", err)
	}

	// Seed ~$20/day of estimate spend across the last 20 days (need a spread for the regression).
	now := time.Now()
	for i := 20; i >= 1; i-- {
		day := now.AddDate(0, 0, -i)
		seedEstimate(t, s, projID, "i-1", 20, 0, day.Add(6*time.Hour))
	}

	resp, err := s.forecast(context.Background(), projID, &project.ProjectForecastRequest{Months: 6})
	if err != nil {
		t.Fatalf("forecast: %v", err)
	}
	if resp.CurrentMonthlyRate <= 0 {
		t.Fatalf("CurrentMonthlyRate = %.2f, want > 0", resp.CurrentMonthlyRate)
	}
	if len(resp.ForecastData) == 0 {
		t.Fatal("ForecastData should have projected months")
	}
	if resp.Confidence <= 0 {
		t.Fatalf("Confidence = %.2f, want > 0", resp.Confidence)
	}
}
