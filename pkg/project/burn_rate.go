package project

import (
	"time"

	"github.com/scttfrdmn/prism/pkg/types"
)

// PaceStatus describes whether spending is on track relative to the budget period.
type PaceStatus string

const (
	// PaceStatusOnPace means within ±10% of expected rate.
	PaceStatusOnPace PaceStatus = "on_pace"
	// PaceStatusAhead means spending faster than expected.
	PaceStatusAhead PaceStatus = "ahead"
	// PaceStatusBehind means spending slower than expected.
	PaceStatusBehind PaceStatus = "behind"
	// PaceStatusOverBudget means the period allocation is already exceeded.
	PaceStatusOverBudget PaceStatus = "over_budget"
)

// PeriodWindow represents a normalized budget period window.
type PeriodWindow struct {
	Start         time.Time          `json:"start"`
	End           time.Time          `json:"end"`
	TotalDays     float64            `json:"total_days"`
	ElapsedDays   float64            `json:"elapsed_days"`
	RemainingDays float64            `json:"remaining_days"`
	PeriodType    types.BudgetPeriod `json:"period_type"`
}

// BurnRateInfo contains period-aware burn rate analysis.
type BurnRateInfo struct {
	// DailyRate is the 7-day trailing average spend in dollars per day.
	DailyRate float64 `json:"daily_rate"`
	// WeeklyRate is projected weekly spend.
	WeeklyRate float64 `json:"weekly_rate"`
	// MonthlyRate is projected monthly spend.
	MonthlyRate float64 `json:"monthly_rate"`
	// PeriodAllocation is the budget amount for the current window.
	PeriodAllocation float64 `json:"period_allocation"`
	// PeriodSpend is how much has been spent within the current window.
	PeriodSpend float64 `json:"period_spend"`
	// PeriodRemaining is how much of the period allocation is left.
	PeriodRemaining float64 `json:"period_remaining"`
	// BurnRateRatio is actual/expected rate; >1.0 means over-paced.
	BurnRateRatio float64 `json:"burn_rate_ratio"`
	// PaceStatus characterises the current spending pace.
	PaceStatus PaceStatus `json:"pace_status"`
	// Window is the current period window.
	Window PeriodWindow `json:"window"`
}

// BurnRateCalculator computes time-bounded spending rates.
type BurnRateCalculator struct {
	// Now returns the current time; nil means time.Now. Tests set it so period windows
	// (calendar months, weeks, days) don't depend on the date the test runs.
	Now func() time.Time
}

func (b *BurnRateCalculator) now() time.Time {
	if b != nil && b.Now != nil {
		return b.Now()
	}
	return time.Now()
}

// DailyBurnRate returns dollars/day averaged over up to windowDays of history.
func (b *BurnRateCalculator) DailyBurnRate(history []CostDataPoint, windowDays int) float64 {
	if len(history) == 0 {
		return 0
	}
	cutoff := b.now().AddDate(0, 0, -windowDays)
	var recent []CostDataPoint
	for _, p := range history {
		if p.Timestamp.After(cutoff) {
			recent = append(recent, p)
		}
	}
	if len(recent) < 2 {
		// Fall back to all available data
		recent = history
	}
	if len(recent) < 2 {
		return 0
	}
	oldest := recent[0].Timestamp
	newest := recent[len(recent)-1].Timestamp
	elapsed := newest.Sub(oldest).Hours() / 24
	if elapsed <= 0 {
		return 0
	}
	delta := recent[len(recent)-1].TotalCost - recent[0].TotalCost
	if delta < 0 {
		delta = 0
	}
	return delta / elapsed
}

// WeeklyBurnRate returns projected weekly spend.
func (b *BurnRateCalculator) WeeklyBurnRate(history []CostDataPoint) float64 {
	return b.DailyBurnRate(history, 7) * 7
}

// CurrentPeriodWindow calculates the current period window given a BudgetPeriod and
// the date from which the project started being tracked.
func (b *BurnRateCalculator) CurrentPeriodWindow(period types.BudgetPeriod, startDate time.Time) PeriodWindow {
	if period == types.BudgetPeriodDaily || period == types.BudgetPeriodWeekly || period == types.BudgetPeriodMonthly {
		return b.PeriodWindowAt(period, b.now())
	}
	// BudgetPeriodProject — entire project lifetime
	return b.window(period, startDate, startDate.AddDate(10, 0, 0)) // effectively unbounded
}

// PeriodWindowAt returns the daily, weekly or monthly window that contains t. Elapsed and
// remaining days are measured against the current time, so a past window is fully elapsed.
func (b *BurnRateCalculator) PeriodWindowAt(period types.BudgetPeriod, t time.Time) PeriodWindow {
	var windowStart, windowEnd time.Time

	switch period {
	case types.BudgetPeriodDaily:
		windowStart = time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
		windowEnd = windowStart.AddDate(0, 0, 1)

	case types.BudgetPeriodWeekly:
		// Align to Monday of t's week.
		weekday := int(t.Weekday())
		if weekday == 0 {
			weekday = 7 // Sunday → 7
		}
		windowStart = time.Date(t.Year(), t.Month(), t.Day()-weekday+1, 0, 0, 0, 0, t.Location())
		windowEnd = windowStart.AddDate(0, 0, 7)

	default: // BudgetPeriodMonthly
		windowStart = time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, t.Location())
		windowEnd = windowStart.AddDate(0, 1, 0)
	}

	return b.window(period, windowStart, windowEnd)
}

// window fills in a PeriodWindow's day counts relative to the current time.
func (b *BurnRateCalculator) window(period types.BudgetPeriod, windowStart, windowEnd time.Time) PeriodWindow {
	totalDays := windowEnd.Sub(windowStart).Hours() / 24
	elapsedDays := b.now().Sub(windowStart).Hours() / 24
	if elapsedDays < 0 {
		elapsedDays = 0
	}
	if elapsedDays > totalDays {
		elapsedDays = totalDays
	}

	return PeriodWindow{
		Start:         windowStart,
		End:           windowEnd,
		TotalDays:     totalDays,
		ElapsedDays:   elapsedDays,
		RemainingDays: totalDays - elapsedDays,
		PeriodType:    period,
	}
}

// PeriodSpend returns the total spent within the window [Start, End). History is cumulative
// (TotalCost is a running total), so spend is the last in-window total minus the total just
// before the window: the last earlier point's TotalCost, or else the first in-window point's
// TotalCost less its own DailyCost, so the window's first day counts too.
func (b *BurnRateCalculator) PeriodSpend(history []CostDataPoint, window PeriodWindow) float64 {
	var before, first, last *CostDataPoint
	for i := range history {
		p := &history[i]
		switch {
		case p.Timestamp.Before(window.Start):
			before = p
		case p.Timestamp.Before(window.End):
			if first == nil {
				first = p
			}
			last = p
		}
	}
	if last == nil {
		return 0
	}
	baseline := first.TotalCost - first.DailyCost
	if before != nil {
		baseline = before.TotalCost
	}
	delta := last.TotalCost - baseline
	if delta < 0 {
		return 0
	}
	return delta
}

// BurnRateVsAllocation compares the current daily burn rate to the pro-rated daily
// allocation for the current period.
//
// Returns (actualDailyRate, expectedDailyRate, ratio) where ratio > 1.0 means
// spending faster than the period allocation allows.
func (b *BurnRateCalculator) BurnRateVsAllocation(history []CostDataPoint, window PeriodWindow, totalAllocation float64) (float64, float64, float64) {
	actualRate := b.DailyBurnRate(history, 7)

	if window.TotalDays <= 0 {
		return actualRate, 0, 0
	}
	expectedRate := totalAllocation / window.TotalDays

	ratio := 0.0
	if expectedRate > 0 {
		ratio = actualRate / expectedRate
	}
	return actualRate, expectedRate, ratio
}

// paceStatus computes the pace status from a burn rate ratio.
func paceStatus(ratio float64, periodSpend, periodAllocation float64) PaceStatus {
	if periodAllocation > 0 && periodSpend > periodAllocation {
		return PaceStatusOverBudget
	}
	if ratio <= 0 {
		return PaceStatusBehind
	}
	if ratio > 1.10 {
		return PaceStatusAhead
	}
	if ratio < 0.90 {
		return PaceStatusBehind
	}
	return PaceStatusOnPace
}

// ComputeBurnRate builds a fully populated BurnRateInfo for a project budget.
func (b *BurnRateCalculator) ComputeBurnRate(
	history []CostDataPoint,
	period types.BudgetPeriod,
	startDate time.Time,
	totalAllocation float64,
) *BurnRateInfo {
	window := b.CurrentPeriodWindow(period, startDate)
	periodSpend := b.PeriodSpend(history, window)
	dailyRate, expectedRate, ratio := b.BurnRateVsAllocation(history, window, totalAllocation)

	periodRemaining := totalAllocation - periodSpend
	if periodRemaining < 0 {
		periodRemaining = 0
	}

	_ = expectedRate // available for future display

	return &BurnRateInfo{
		DailyRate:        dailyRate,
		WeeklyRate:       dailyRate * 7,
		MonthlyRate:      dailyRate * 30,
		PeriodAllocation: totalAllocation,
		PeriodSpend:      periodSpend,
		PeriodRemaining:  periodRemaining,
		BurnRateRatio:    ratio,
		PaceStatus:       paceStatus(ratio, periodSpend, totalAllocation),
		Window:           window,
	}
}
