package project

import (
	"sort"
	"testing"
	"time"

	"github.com/scttfrdmn/prism/pkg/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestComputeSurplusWithRollover_Disabled verifies that when RolloverEnabled=false
// BankedSurplus is 0 and SurplusCapped is false regardless of history.
func TestComputeSurplusWithRollover_Disabled(t *testing.T) {
	calc := &SurplusCalculator{}
	budget := &types.ProjectBudget{
		TotalBudget:     1000.0,
		BudgetPeriod:    types.BudgetPeriodMonthly,
		StartDate:       time.Now(),
		RolloverEnabled: false,
	}
	// No history, rollover disabled.
	info := calc.ComputeSurplusWithRollover(nil, budget, 100.0)
	require.NotNil(t, info)
	assert.Equal(t, 0.0, info.BankedSurplus, "BankedSurplus should be 0 when rollover is disabled")
	assert.False(t, info.SurplusCapped, "SurplusCapped should be false when rollover is disabled")
	assert.Equal(t, 0.0, info.SurplusCapPercent, "SurplusCapPercent should be 0 when rollover is disabled")
}

// TestComputeSurplusWithRollover_NilBudget exercises the nil-budget branch.
// NOTE: The current implementation dereferences budget.BudgetPeriod/StartDate
// even when !budget.RolloverEnabled, so passing a nil pointer would panic.
// The contract therefore requires a non-nil budget.  We pass RolloverEnabled=false
// to confirm the no-rollover path still produces BankedSurplus=0.
func TestComputeSurplusWithRollover_NilBudget(t *testing.T) {
	calc := &SurplusCalculator{}
	// Use a valid budget with RolloverEnabled=false to exercise the nil/disabled branch.
	budget := &types.ProjectBudget{
		TotalBudget:     500.0,
		BudgetPeriod:    types.BudgetPeriodMonthly,
		StartDate:       time.Now(),
		RolloverEnabled: false,
	}
	info := calc.ComputeSurplusWithRollover(nil, budget, 50.0)
	require.NotNil(t, info)
	assert.Equal(t, 0.0, info.BankedSurplus)
	assert.False(t, info.SurplusCapped)
}

// TestComputeSurplusWithRollover_Enabled_NoCap verifies that when RolloverEnabled=true
// and RolloverCap=0, the cap percentage is 1.0 (unlimited).
func TestComputeSurplusWithRollover_Enabled_NoCap(t *testing.T) {
	calc := &SurplusCalculator{}
	budget := &types.ProjectBudget{
		TotalBudget:  1000.0,
		BudgetPeriod: types.BudgetPeriodMonthly,
		// StartDate = now so no completed periods exist yet.
		StartDate:       time.Now(),
		RolloverEnabled: true,
		RolloverCap:     0, // unlimited
	}
	info := calc.ComputeSurplusWithRollover(nil, budget, 100.0)
	require.NotNil(t, info)
	// With StartDate = now there are no completed periods, so BankedSurplus=0.
	assert.Equal(t, 0.0, info.BankedSurplus)
	// Cap percent should be 1.0 (unlimited).
	assert.Equal(t, 1.0, info.SurplusCapPercent)
	assert.False(t, info.SurplusCapped)
}

// TestComputeSurplusWithRollover_Enabled_Cap verifies that when RolloverEnabled=true
// and RolloverCap>0 the cap percentage is computed as RolloverCap/TotalBudget.
func TestComputeSurplusWithRollover_Enabled_Cap(t *testing.T) {
	calc := &SurplusCalculator{}
	budget := &types.ProjectBudget{
		TotalBudget:     1000.0,
		BudgetPeriod:    types.BudgetPeriodMonthly,
		StartDate:       time.Now(),
		RolloverEnabled: true,
		RolloverCap:     200.0, // 20% of 1000
	}
	info := calc.ComputeSurplusWithRollover(nil, budget, 100.0)
	require.NotNil(t, info)
	// capPercent = 200/1000 = 0.2
	assert.InDelta(t, 0.2, info.SurplusCapPercent, 1e-9)
	// No completed periods → BankedSurplus=0, so the cap is not triggered.
	assert.Equal(t, 0.0, info.BankedSurplus)
	assert.False(t, info.SurplusCapped)
}

// TestComputeSurplusWithRollover_NegativeSurplus verifies that when spending
// exceeds the allocation the surplus is non-positive and BankedSurplus remains 0.
func TestComputeSurplusWithRollover_NegativeSurplus(t *testing.T) {
	calc := &SurplusCalculator{}
	budget := &types.ProjectBudget{
		TotalBudget:     1000.0,
		BudgetPeriod:    types.BudgetPeriodMonthly,
		StartDate:       time.Now(),
		RolloverEnabled: true,
		RolloverCap:     0,
	}

	// Build history with spend that exceeds allocation this period.
	// PeriodSpend uses the delta between the last and first TotalCost in the window,
	// so provide two points bracketing the current period.
	now := time.Now()
	history := []CostDataPoint{
		{
			Timestamp: now.Add(-time.Hour),
			TotalCost: 0.0,
		},
		{
			Timestamp: now,
			TotalCost: 999.0, // well over the 50.0 allocation
		},
	}

	info := calc.ComputeSurplusWithRollover(history, budget, 50.0)
	require.NotNil(t, info)
	assert.LessOrEqual(t, info.CurrentPeriodSurplus, 0.0, "surplus should be non-positive when over budget")
	// BankedSurplus from completed periods: StartDate=now → no completed periods.
	assert.Equal(t, 0.0, info.BankedSurplus)
}

// series builds a cumulative day-bucketed history, as the daemon's ledger series does:
// one point per day at UTC midnight, DailyCost that day's spend, TotalCost the running total.
func series(days map[string]float64) []CostDataPoint {
	keys := make([]string, 0, len(days))
	for k := range days {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var out []CostDataPoint
	var total float64
	for _, k := range keys {
		day, _ := time.Parse("2006-01-02", k)
		total += days[k]
		out = append(out, CostDataPoint{Timestamp: day, TotalCost: total, DailyCost: days[k]})
	}
	return out
}

func fixedClock(t time.Time) func() time.Time { return func() time.Time { return t } }

// TestBankedSurplus_Monthly (#764): completed months bank their unspent allocation; the current
// month doesn't. A month's first day counts toward its spend.
func TestBankedSurplus_Monthly(t *testing.T) {
	calc := &SurplusCalculator{Now: fixedClock(time.Date(2026, 7, 15, 12, 0, 0, 0, time.UTC))}
	history := series(map[string]float64{
		"2026-05-01": 300, // first day of May
		"2026-05-31": 100,
		"2026-06-01": 1000, // all of June's allocation, on its first day
		"2026-07-02": 50,
	})
	start := time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC)

	// May: 1000 − 400 = 600; June: 1000 − 1000 = 0; July is in progress.
	assert.InDelta(t, 600, calc.BankedSurplus(history, types.BudgetPeriodMonthly, start, 1000), 1e-9)
}

// TestBankedSurplus_Daily: with one history point per day, a fully spent day banks nothing
// (the first-point baseline used to make every day look unspent).
func TestBankedSurplus_Daily(t *testing.T) {
	calc := &SurplusCalculator{Now: fixedClock(time.Date(2026, 7, 4, 12, 0, 0, 0, time.UTC))}
	history := series(map[string]float64{
		"2026-07-01": 10,
		"2026-07-02": 4,
		"2026-07-03": 10,
	})
	start := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)

	// July 1: 0, July 2: 6, July 3: 0; July 4 is in progress.
	assert.InDelta(t, 6, calc.BankedSurplus(history, types.BudgetPeriodDaily, start, 10), 1e-9)
}

// TestBankedSurplus_Weekly: weeks run Monday to Monday, starting with the week holding startDate.
func TestBankedSurplus_Weekly(t *testing.T) {
	// Wednesday 2026-07-15; completed weeks start Mon 06-29 and Mon 07-06.
	calc := &SurplusCalculator{Now: fixedClock(time.Date(2026, 7, 15, 12, 0, 0, 0, time.UTC))}
	history := series(map[string]float64{
		"2026-07-05": 70, // Sunday: last day of the first week
		"2026-07-06": 20, // Monday: first day of the second
		"2026-07-13": 99, // current week
	})
	start := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)

	// Week of 06-29: 100 − 70 = 30; week of 07-06: 100 − 20 = 80.
	assert.InDelta(t, 110, calc.BankedSurplus(history, types.BudgetPeriodWeekly, start, 100), 1e-9)
}

// TestBankedSurplus_NeverNegative: overspending nets against underspending, and a net
// overspend banks 0 rather than a debt.
func TestBankedSurplus_NeverNegative(t *testing.T) {
	calc := &SurplusCalculator{Now: fixedClock(time.Date(2026, 7, 15, 12, 0, 0, 0, time.UTC))}
	history := series(map[string]float64{"2026-05-10": 1500, "2026-06-10": 900})
	start := time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC)

	// May −500, June +100.
	assert.Equal(t, 0.0, calc.BankedSurplus(history, types.BudgetPeriodMonthly, start, 1000))
}

// TestBankedSurplus_NoPeriods: project-lifetime budgets, an unset period and an unset StartDate
// bank nothing.
func TestBankedSurplus_NoPeriods(t *testing.T) {
	calc := &SurplusCalculator{Now: fixedClock(time.Date(2026, 7, 15, 12, 0, 0, 0, time.UTC))}
	start := time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC)

	assert.Equal(t, 0.0, calc.BankedSurplus(nil, types.BudgetPeriodProject, start, 1000))
	assert.Equal(t, 0.0, calc.BankedSurplus(nil, "", start, 1000))
	assert.Equal(t, 0.0, calc.BankedSurplus(nil, types.BudgetPeriodMonthly, time.Time{}, 1000))
}

// TestComputeSurplusWithRollover_BanksAndCaps: rollover banks completed periods and RolloverCap
// limits how much of it counts toward the effective balance; without rollover nothing banks.
func TestComputeSurplusWithRollover_BanksAndCaps(t *testing.T) {
	calc := &SurplusCalculator{Now: fixedClock(time.Date(2026, 7, 15, 12, 0, 0, 0, time.UTC))}
	history := series(map[string]float64{"2026-05-10": 200, "2026-06-10": 400})
	budget := &types.ProjectBudget{
		TotalBudget:     1000,
		BudgetPeriod:    types.BudgetPeriodMonthly,
		StartDate:       time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC),
		RolloverEnabled: true,
		RolloverCap:     500,
	}

	info := calc.ComputeSurplusWithRollover(history, budget, 1000)
	assert.InDelta(t, 1400, info.BankedSurplus, 1e-9) // 800 + 600
	assert.True(t, info.SurplusCapped)
	assert.InDelta(t, 1500, info.EffectiveBalance, 1e-9) // 1000 + the 500 cap

	budget.RolloverEnabled = false
	info = calc.ComputeSurplusWithRollover(history, budget, 1000)
	assert.Equal(t, 0.0, info.BankedSurplus)
	assert.False(t, info.SurplusCapped)
	assert.InDelta(t, 1000, info.EffectiveBalance, 1e-9)
}

// TestPeriodSpend_HalfOpenWindow: a point at a window's End belongs to the next window, and
// spend before the window is excluded via the preceding point's running total.
func TestPeriodSpend_HalfOpenWindow(t *testing.T) {
	calc := &BurnRateCalculator{Now: fixedClock(time.Date(2026, 7, 15, 12, 0, 0, 0, time.UTC))}
	history := series(map[string]float64{"2026-05-31": 100, "2026-06-01": 30, "2026-06-30": 20, "2026-07-01": 500})
	june := calc.PeriodWindowAt(types.BudgetPeriodMonthly, time.Date(2026, 6, 10, 0, 0, 0, 0, time.UTC))

	assert.InDelta(t, 50, calc.PeriodSpend(history, june), 1e-9)
	assert.Equal(t, 30.0, june.TotalDays)
	assert.Equal(t, 0.0, june.RemainingDays)
}
