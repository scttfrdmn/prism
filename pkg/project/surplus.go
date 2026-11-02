package project

import (
	"time"

	"github.com/scttfrdmn/prism/pkg/types"
)

// SurplusInfo contains banking and carry-over information for a project budget.
type SurplusInfo struct {
	// CurrentPeriodSurplus is unspent amount in the current period (negative = over budget).
	CurrentPeriodSurplus float64 `json:"current_period_surplus"`
	// BankedSurplus is accumulated carry-over from prior completed periods.
	BankedSurplus float64 `json:"banked_surplus"`
	// EffectiveBalance is the current period allocation plus banked surplus (capped).
	EffectiveBalance float64 `json:"effective_balance"`
	// SurplusCapPercent is the maximum carry-over as a fraction of the total budget.
	SurplusCapPercent float64 `json:"surplus_cap_percent"`
	// SurplusCapped is true when the cap was applied.
	SurplusCapped bool `json:"surplus_capped"`
}

// SurplusCalculator handles budget banking and carry-over across periods.
type SurplusCalculator struct {
	// Now returns the current time; nil means time.Now (see BurnRateCalculator.Now).
	Now func() time.Time
}

// PeriodSurplus returns the unspent amount in the given window relative to its allocation.
// A negative value means the window was over-budget.
func (s *SurplusCalculator) PeriodSurplus(history []CostDataPoint, window PeriodWindow, allocation float64) float64 {
	calc := &BurnRateCalculator{Now: s.Now}
	spent := calc.PeriodSpend(history, window)
	return allocation - spent
}

// BankedSurplus returns the total accumulated surplus across all completed periods
// prior to the current one.  Only meaningful for monthly/weekly/daily periods.
func (s *SurplusCalculator) BankedSurplus(
	history []CostDataPoint,
	period types.BudgetPeriod,
	startDate time.Time,
	periodAllocation float64,
) float64 {
	if period == types.BudgetPeriodProject {
		return 0 // project-lifetime budgets don't bank across periods
	}

	calc := &BurnRateCalculator{Now: s.Now}
	now := calc.now()
	var banked float64

	// Walk through all completed periods from startDate to now.
	cursor := startDate
	for {
		window := calc.CurrentPeriodWindow(period, cursor)
		// Only count fully elapsed windows.
		if window.End.After(now) {
			break
		}
		surplus := s.PeriodSurplus(history, window, periodAllocation)
		banked += surplus

		// Advance cursor to the start of the next period.
		cursor = window.End
		// Safety: avoid infinite loop on misconfigured period.
		if cursor.Equal(startDate) || cursor.Before(startDate) {
			break
		}
	}
	if banked < 0 {
		return 0 // never carry negative balance
	}
	return banked
}

// EffectiveBalance returns the allocation for the current period plus banked surplus,
// subject to the surplus cap.
//
// surplusCapPercent=0 means banking is disabled; surplusCapPercent=0.20 means at
// most 20% of totalBudget can carry over.
func (s *SurplusCalculator) EffectiveBalance(periodAllocation, bankedSurplus, totalBudget, surplusCapPercent float64) float64 {
	if surplusCapPercent <= 0 {
		return periodAllocation
	}
	cap := totalBudget * surplusCapPercent
	carried := bankedSurplus
	if carried > cap {
		carried = cap
	}
	return periodAllocation + carried
}

// ComputeSurplus builds a fully populated SurplusInfo for a project budget.
func (s *SurplusCalculator) ComputeSurplus(
	history []CostDataPoint,
	period types.BudgetPeriod,
	startDate time.Time,
	totalBudget float64,
	periodAllocation float64,
	surplusCapPercent float64,
) *SurplusInfo {
	calc := &BurnRateCalculator{Now: s.Now}
	window := calc.CurrentPeriodWindow(period, startDate)
	currentSurplus := s.PeriodSurplus(history, window, periodAllocation)

	banked := s.BankedSurplus(history, period, startDate, periodAllocation)

	cap := totalBudget * surplusCapPercent
	capped := surplusCapPercent > 0 && banked > cap
	effective := s.EffectiveBalance(periodAllocation, banked, totalBudget, surplusCapPercent)

	return &SurplusInfo{
		CurrentPeriodSurplus: currentSurplus,
		BankedSurplus:        banked,
		EffectiveBalance:     effective,
		SurplusCapPercent:    surplusCapPercent,
		SurplusCapped:        capped,
	}
}

// ComputeSurplusWithRollover builds a SurplusInfo respecting the v0.12.0 rollover
// configuration fields on ProjectBudget (#143).
//
// When RolloverEnabled=false the behaviour is identical to ComputeSurplus with
// surplusCapPercent=0.  When RolloverEnabled=true the cap is derived from
// RolloverCap (absolute dollar amount) unless it is 0, in which case there is
// no cap (surplusCapPercent=1.0 acts as 100%).
func (s *SurplusCalculator) ComputeSurplusWithRollover(
	history []CostDataPoint,
	budget *types.ProjectBudget,
	periodAllocation float64,
) *SurplusInfo {
	if budget == nil || !budget.RolloverEnabled {
		return s.ComputeSurplus(history, budget.BudgetPeriod, budget.StartDate,
			budget.TotalBudget, periodAllocation, 0)
	}

	// Determine effective cap percentage
	capPercent := 1.0 // unlimited by default when rollover enabled
	if budget.RolloverCap > 0 && budget.TotalBudget > 0 {
		capPercent = budget.RolloverCap / budget.TotalBudget
	}

	return s.ComputeSurplus(history, budget.BudgetPeriod, budget.StartDate,
		budget.TotalBudget, periodAllocation, capPercent)
}
