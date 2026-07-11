package daemon

import (
	"context"
	"time"

	be "github.com/scttfrdmn/budgetengine"
	"github.com/scttfrdmn/prism/pkg/project"
	"github.com/scttfrdmn/prism/pkg/types"
)

// Phase 3c (#653): budget status is derived here, in the daemon, from the budgetengine spend ledger
// + the project's embedded budget plan — NOT from the retired BudgetTracker (which kept a parallel
// budget_data.json model). This makes the ledger the single source of truth for spend: the launch
// gate (checkMonthlyLimitViaEngine) and the status readout now fold the same events.
//
// The engine is fed the same synthesized plan view as the launch gate (budgetWindow +
// newMonthlyLimitEngine) but reads spend from the persisted ledger when it has events, falling back
// to the project's cached SpentAmount otherwise. We evaluate BurnState at "now" (the status view of
// the world) rather than at window-end (the gate's ceiling view).
//
// BurnRate/Surplus analytics stay nil in 3c — they need a cost-history series the ledger does not yet
// expose in the shape the analytics helpers accept; that arrives with the ledger-derived breakdown in
// Phase 3d (#654) and banking/rollover in 3e (#655).

// budgetStatus derives a project's BudgetStatus from the spend ledger and its budget plan. It is the
// daemon-layer replacement for the old Manager.CheckBudgetStatus / BudgetTracker.CheckBudgetStatus.
//
// A missing project is an error; a project with no budget returns {BudgetEnabled:false} (preserving
// the tracker's guard behavior). The returned struct is byte-compatible with the DTO the CLI and GUI
// consume — the only change is where the numbers come from.
func (s *Server) budgetStatus(ctx context.Context, projectID string) (*project.BudgetStatus, error) {
	proj, err := s.projectManager.GetProject(ctx, projectID)
	if err != nil {
		return nil, err
	}

	if proj.Budget == nil {
		return &project.BudgetStatus{
			ProjectID:        projectID,
			BudgetEnabled:    false,
			ActiveAlerts:     []string{},
			TriggeredActions: []string{},
			LastUpdated:      time.Now(),
		}, nil
	}

	budget := proj.Budget
	now := time.Now()
	spent := s.ledgerSpent(ctx, projectID, budget, now)

	totalBudget := budget.TotalBudget
	remaining := totalBudget - spent
	if remaining < 0 {
		remaining = 0
	}
	spentPct := 0.0
	if totalBudget > 0 {
		spentPct = spent / totalBudget
	}

	status := &project.BudgetStatus{
		ProjectID:       projectID,
		BudgetEnabled:   true,
		TotalBudget:     totalBudget,
		SpentAmount:     spent,
		RemainingBudget: remaining,
		SpentPercentage: spentPct,
		// Surplus (banking/rollover) stays nil — Phase 3e (#655). BurnRate is populated below (3d).
		ActiveAlerts:     []string{},
		TriggeredActions: []string{},
		LastUpdated:      now,
	}

	// Fixed-rate projection from the engine's BurnState: sustainable rate → projected monthly spend,
	// projected-zero-date → days until exhausted. This is the engine's read of the same ledger.
	if bs, ok := s.evaluateBurnState(ctx, projectID, budget, now); ok {
		status.ProjectedMonthlySpend = bs.SustainableRate * hoursPerMonth * 3600 // rate is USD/second
		if !bs.ProjectedZeroDate.IsZero() && bs.ProjectedZeroDate.After(now) {
			days := int(bs.ProjectedZeroDate.Sub(now).Hours() / 24)
			status.DaysUntilBudgetExhausted = &days
		}
	}

	// Phase 3d (#654): period-aware burn rate from the ledger-derived cost-history series. Reuses the
	// existing BurnRateCalculator (the same analytics the retired tracker fed). The 7-day trailing rate
	// needs a few weeks of history, so pull a 90-day window; ComputeBurnRate re-clips to the period.
	if series, err := s.ledgerCostSeries(ctx, projectID, now.AddDate(0, 0, -90), now); err == nil && len(series) >= 2 {
		allocation := totalBudget
		if budget.MonthlyAmount > 0 {
			allocation = budget.MonthlyAmount
		}
		calc := &project.BurnRateCalculator{}
		status.BurnRate = calc.ComputeBurnRate(series, budget.BudgetPeriod, budget.StartDate, allocation)
	}

	return status, nil
}

// ledgerSpent returns authoritative cumulative spend for the project.
func (s *Server) ledgerSpent(ctx context.Context, projectID string, budget *types.ProjectBudget, now time.Time) float64 {
	return ledgerSpent(ctx, s.spendLedger, projectID, budget, now)
}

// ledgerSpent folds the persisted ledger via the engine (CheckLaunch with zero cost yields
// Decision.Spent) when the ledger has events; otherwise it falls back to the project's cached
// SpentAmount so a warming-up ledger never reports less than the retired tracker did. Package-level so
// the alert-manager cost-data feed (server.go) can reuse it without a *Server.
func ledgerSpent(ctx context.Context, ledger *spendStore, projectID string, budget *types.ProjectBudget, now time.Time) float64 {
	if ledger != nil && projectID != "" {
		if evs, err := ledger.Spend(ctx, projectScope(projectID)); err == nil && len(evs) > 0 {
			win := budgetWindow(budget, now)
			_, view := newMonthlyLimitEngine(budget, win, now)
			eng := be.New(ledger, view, be.FixedClock{T: now}, bepolicyDefaults()...)
			if d, err := eng.CheckLaunch(ctx, projectScope(projectID), engineAllocationID, 0); err == nil {
				return d.Spent
			}
		}
	}
	return budget.SpentAmount
}

// evaluateBurnState folds the ledger + plan into a BurnState at now (the status/reactive view). It
// prefers the persisted ledger and falls back to the synthesized cached-spend view, mirroring the
// launch gate's source selection. ok is false only when Evaluate errors.
func (s *Server) evaluateBurnState(ctx context.Context, projectID string, budget *types.ProjectBudget, now time.Time) (be.BurnState, bool) {
	win := budgetWindow(budget, now)
	_, view := newMonthlyLimitEngine(budget, win, now)

	spendSource := be.SpendSource(view)
	if s.spendLedger != nil && projectID != "" {
		if evs, err := s.spendLedger.Spend(ctx, projectScope(projectID)); err == nil && len(evs) > 0 {
			spendSource = s.spendLedger
		}
	}
	eng := be.New(spendSource, view, be.FixedClock{T: now}, bepolicyDefaults()...)
	bs, err := eng.Evaluate(ctx, projectScope(projectID), engineAllocationID)
	if err != nil {
		return be.BurnState{}, false
	}
	return bs, true
}
