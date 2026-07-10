package daemon

import (
	"testing"
	"time"

	"github.com/scttfrdmn/prism/pkg/types"
)

func ptr(f float64) *float64 { return &f }

// TestCheckMonthlyLimitViaEngine_ParityWithFlatLimit is the guardrail for Phase 1: the engine-backed
// monthly-limit gate must block exactly when the old projected>limit comparison did. Table drives
// (monthlyLimit, spent, estimatedCost) → expected Allowed().
func TestCheckMonthlyLimitViaEngine_ParityWithFlatLimit(t *testing.T) {
	cases := []struct {
		name          string
		limit         float64
		spent         float64
		estimatedCost float64
		wantAllowed   bool // old logic: (spent+est) > limit → blocked
	}{
		{"well under", 1000, 100, 50, true},
		{"exactly at limit", 1000, 900, 100, true},    // projected == limit → not > → allowed
		{"one cent over", 1000, 900, 100.01, false},   // projected > limit → blocked
		{"already over", 1000, 1200, 10, false},       // over before the launch → blocked
		{"zero spent big launch", 500, 0, 600, false}, // single launch exceeds limit → blocked
		{"zero spent small launch", 500, 0, 100, true},
	}
	s := &Server{}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			b := &types.ProjectBudget{
				MonthlyLimit: ptr(c.limit),
				SpentAmount:  c.spent,
				BudgetPeriod: types.BudgetPeriodMonthly,
			}
			// Old logic, the parity oracle.
			oldBlocked := (c.spent + c.estimatedCost) > c.limit

			dec, err := s.checkMonthlyLimitViaEngine("", b, c.estimatedCost)
			if err != nil {
				t.Fatalf("engine check errored: %v", err)
			}
			if dec.Allowed() == oldBlocked {
				t.Fatalf("parity broken: engine Allowed=%v but old logic blocked=%v (limit=%.2f spent=%.2f est=%.2f, projected=%.2f eff=%.2f verdict=%s)",
					dec.Allowed(), oldBlocked, c.limit, c.spent, c.estimatedCost, dec.Projected, dec.EffectiveBalance, dec.Verdict)
			}
			if dec.Allowed() != c.wantAllowed {
				t.Fatalf("Allowed=%v, want %v (verdict=%s)", dec.Allowed(), c.wantAllowed, dec.Verdict)
			}
		})
	}
}

// TestBudgetWindow_Monthly confirms a monthly budget yields the current calendar month window, so
// the paced ceiling at window end equals the full monthly limit (the parity requirement).
func TestBudgetWindow_Monthly(t *testing.T) {
	now := time.Date(2026, 7, 10, 12, 0, 0, 0, time.UTC)
	b := &types.ProjectBudget{BudgetPeriod: types.BudgetPeriodMonthly}
	w := budgetWindow(b, now)
	if w.Start != time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC) {
		t.Fatalf("Start = %v, want Jul 1", w.Start)
	}
	if w.End != time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC) {
		t.Fatalf("End = %v, want Aug 1", w.End)
	}
}

// TestCheckMonthlyLimitViaEngine_CeilingEqualsLimit confirms the engine's EffectiveBalance at the
// check equals the monthly limit (not a pro-rated slice) — the property that makes it a parity of
// the flat-limit comparison rather than a paced-to-date comparison.
func TestCheckMonthlyLimitViaEngine_CeilingEqualsLimit(t *testing.T) {
	s := &Server{}
	b := &types.ProjectBudget{MonthlyLimit: ptr(1000), SpentAmount: 0, BudgetPeriod: types.BudgetPeriodMonthly}
	dec, err := s.checkMonthlyLimitViaEngine("", b, 1)
	if err != nil {
		t.Fatal(err)
	}
	if dec.EffectiveBalance < 999.5 || dec.EffectiveBalance > 1000.5 {
		t.Fatalf("EffectiveBalance = %.2f, want ~1000 (full monthly limit)", dec.EffectiveBalance)
	}
}
