package daemon

import (
	"context"
	"time"

	be "github.com/scttfrdmn/budgetengine"
	bepolicy "github.com/scttfrdmn/budgetengine/policy"
	"github.com/scttfrdmn/prism/pkg/types"
)

// This file is Phase 1 of Prism's adoption of the standalone budgetengine library
// (github.com/scttfrdmn/budgetengine). It routes the launch-time monthly-limit budget gate through
// engine.CheckLaunch, WITHOUT changing Prism's budget storage or spend model.
//
// The engine is fed a read-only, in-process view synthesized from the project's existing embedded
// budget (types.ProjectBudget): a single funding source over the budget window, one allocation, and
// one synthetic spend event carrying the already-cached SpentAmount. This is the degenerate
// single-source case, which reproduces today's flat monthly-limit semantics exactly — the real
// event-sourced spend ledger arrives in Phase 2 (#644). No new persistence is introduced here.

const engineAllocationID = "project" // Phase 1 uses one implicit allocation per project

// projectBudgetView adapts a project's embedded budget into the engine's read ports (SpendSource +
// PlanSource). It is constructed per check from current data, so it needs no storage of its own.
type projectBudgetView struct {
	plan  []be.PlanEvent
	spend []be.SpendEvent
}

func (v projectBudgetView) Plan(_ context.Context, _ be.Scope) ([]be.PlanEvent, error) {
	return v.plan, nil
}

func (v projectBudgetView) Spend(_ context.Context, _ be.Scope) ([]be.SpendEvent, error) {
	return v.spend, nil
}

// budgetWindow derives the [start,end] the engine paces over from the project's budget period.
// Monthly (the limit case) → the current calendar month; explicit EndDate wins when present;
// otherwise a one-month window from StartDate (or now).
func budgetWindow(b *types.ProjectBudget, now time.Time) be.Window {
	start := b.StartDate
	if start.IsZero() {
		start = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	}
	if b.EndDate != nil && !b.EndDate.IsZero() {
		return be.Window{Start: start, End: *b.EndDate}
	}
	switch b.BudgetPeriod {
	case types.BudgetPeriodMonthly, "":
		// Current month: first of this month → first of next month.
		s := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
		return be.Window{Start: s, End: s.AddDate(0, 1, 0)}
	default:
		// Project/weekly/daily fall back to a month-long window from start (Phase 1 parity only).
		return be.Window{Start: start, End: start.AddDate(0, 1, 0)}
	}
}

// newMonthlyLimitEngine builds an engine + view for the monthly-limit launch gate over an explicit
// window, evaluated at evalAt. The funding "source" is the monthly limit spread over the window; a
// single synthetic spend event carries the cached SpentAmount. Policies are the single-source parity
// trio (ExpiryFirst × RateAdjust × DeadlineFloatPolicy): no banking, block at the ceiling — matching
// today's projected>limit→403 behavior. warnThreshold ~1.0 disables the warn band in Phase 1 so the
// gate's block/allow decisions are identical to the current code (Warn is introduced in a later
// phase once spend is a real ledger).
func newMonthlyLimitEngine(b *types.ProjectBudget, win be.Window, evalAt time.Time) (*be.Engine, projectBudgetView) {
	limit := 0.0
	if b.MonthlyLimit != nil {
		limit = *b.MonthlyLimit
	}

	view := projectBudgetView{
		plan: []be.PlanEvent{
			{Kind: be.KindWindowExtended, Seq: 1, At: win.Start, Window: &be.Window{Start: win.Start, End: win.End}},
			{Kind: be.KindSourceAdded, Seq: 2, At: win.Start,
				Source: &be.FundingSource{ID: "monthly-limit", Amount: limit, Start: win.Start, End: win.End}},
			{Kind: be.KindAllocationChanged, Seq: 3,
				Allocation: &be.Allocation{ID: engineAllocationID, ProjectID: "", Amount: 0}}, // uncapped: draws whole source
		},
		// Phase 1: cached spend as one synthetic event, stamped at window start so it counts as
		// already-incurred (available_to_date at `now` is the full paced ceiling to date).
		spend: []be.SpendEvent{
			{ID: "cached-spend", AllocationID: engineAllocationID, Amount: b.SpentAmount, At: win.Start, Source: "prism-cached"},
		},
	}

	eng := be.New(view, view, be.FixedClock{T: evalAt},
		be.WithSourcing(bepolicy.ExpiryFirst{}),
		be.WithPacing(bepolicy.RateAdjust{}),
		be.WithProjection(bepolicy.DeadlineFloatPolicy{}),
		be.WithWarnThreshold(0.999), // effectively disable the warn band for Phase 1 parity
	)
	return eng, view
}

// checkMonthlyLimitViaEngine is the engine-backed replacement for the launch gate's
// projected>monthly-limit check. It returns the engine Decision for a launch of estimatedCost
// against the project's monthly budget. The caller maps Block→403 and Allow/Warn→proceed.
//
// For a monthly budget, the engine's paced ceiling at end-of-window equals the full monthly limit,
// which is the quantity the old code compared against — so with a whole-window source and the cached
// spend, Decision.Projected/EffectiveBalance mirror currentSpend+estimate vs. monthlyLimit.
func (s *Server) checkMonthlyLimitViaEngine(b *types.ProjectBudget, estimatedCost float64) (be.Decision, error) {
	// Evaluate at window end so available_to_date == full monthly limit (paced ceiling matches the
	// flat limit the legacy gate used). This keeps Phase 1 a faithful parity of the old comparison.
	win := budgetWindow(b, time.Now())
	eng, _ := newMonthlyLimitEngine(b, win, win.End)
	return eng.CheckLaunch(context.Background(), be.Scope{}, engineAllocationID, estimatedCost)
}
