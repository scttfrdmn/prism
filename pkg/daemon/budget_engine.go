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

// newMonthlyLimitView builds the read-only engine view (plan + fallback spend) for the monthly-limit
// budget over an explicit window. The funding "source" is the monthly limit spread over the window; a
// single synthetic spend event carries the cached SpentAmount (used only when the real ledger is
// empty). Callers construct the engine themselves with the clock and policy trio they need
// (bepolicyDefaults for the launch gate, bepolicyFor for the status read), so this returns only the
// view.
func newMonthlyLimitView(b *types.ProjectBudget, win be.Window) projectBudgetView {
	limit := 0.0
	if b.MonthlyLimit != nil {
		limit = *b.MonthlyLimit
	}

	return projectBudgetView{
		plan: []be.PlanEvent{
			{Kind: be.KindWindowExtended, Seq: 1, At: win.Start, Window: &be.Window{Start: win.Start, End: win.End}},
			{Kind: be.KindSourceAdded, Seq: 2, At: win.Start,
				Source: &be.FundingSource{ID: "monthly-limit", Amount: limit, Start: win.Start, End: win.End}},
			{Kind: be.KindAllocationChanged, Seq: 3,
				Allocation: &be.Allocation{ID: engineAllocationID, ProjectID: "", Amount: 0}}, // uncapped: draws whole source
		},
		// Fallback spend feed: the cached SpentAmount as one synthetic event, stamped at window start
		// so it counts as already-incurred. Used only when the real ledger has no events yet.
		spend: []be.SpendEvent{
			{ID: "cached-spend", AllocationID: engineAllocationID, Amount: b.SpentAmount, At: win.Start, Source: "prism-cached"},
		},
	}
}

// bepolicyDefaults is the single-source parity policy trio used by the monthly-limit gate:
// ExpiryFirst × RateAdjust × DeadlineFloatPolicy, with the warn band effectively disabled so
// block/allow decisions match the legacy flat-limit check. Shared by the synthesized-fallback and
// real-ledger engine constructions so both decide identically.
func bepolicyDefaults() []be.Option {
	return []be.Option{
		be.WithSourcing(bepolicy.ExpiryFirst{}),
		be.WithPacing(bepolicy.RateAdjust{}),
		be.WithProjection(bepolicy.DeadlineFloatPolicy{}),
		be.WithWarnThreshold(0.999),
	}
}

// isGrantBudget reports whether a budget is a multi-month grant (as opposed to a simple monthly/cloud
// budget). Grants are paced to last to a deadline; cloud budgets answer "when do I run out at this
// pace". Detected by the multi-month allocation fields (#144).
func isGrantBudget(b *types.ProjectBudget) bool {
	return b != nil && (b.MonthlyAmount > 0 || b.AllocationMonths > 1)
}

// bepolicyFor selects the pacing/projection composition for a budget's BurnState readout (Phase 3e,
// #655), per the architecture doc's two worked compositions:
//   - multi-month grant → ExpiryFirst × BankAndReserve × RateTargetPolicy (fixed-date: hold the
//     nominal rate, bank underspend, stay on track unless borrowed past tolerance).
//   - simple cloud/monthly budget → the default ExpiryFirst × RateAdjust × DeadlineFloatPolicy
//     (memoryless re-pacing: "at this pace, when do I run out?").
//
// This governs only the status/reactive read (evaluateBurnState). The launch gate
// (checkMonthlyLimitViaEngine) deliberately keeps bepolicyDefaults() — CheckLaunch never invokes the
// pacing policy (it compares projected spend against AvailableToDate), so gate decisions are unchanged
// regardless of pacing choice, preserving byte-for-byte launch parity.
func bepolicyFor(b *types.ProjectBudget) []be.Option {
	if !isGrantBudget(b) {
		return bepolicyDefaults()
	}
	return []be.Option{
		be.WithSourcing(bepolicy.ExpiryFirst{}),
		be.WithPacing(bepolicy.BankAndReserve{}),
		be.WithProjection(bepolicy.RateTargetPolicy{}),
		be.WithWarnThreshold(0.999),
	}
}

// checkMonthlyLimitViaEngine is the engine-backed replacement for the launch gate's
// projected>monthly-limit check. It returns the engine Decision for a launch of estimatedCost
// against the project's monthly budget. The caller maps Block→403 and Allow/Warn→proceed.
//
// For a monthly budget, the engine's paced ceiling at end-of-window equals the full monthly limit,
// which is the quantity the old code compared against — so with a whole-window source and the cached
// spend, Decision.Projected/EffectiveBalance mirror currentSpend+estimate vs. monthlyLimit.
func (s *Server) checkMonthlyLimitViaEngine(projectID string, b *types.ProjectBudget, estimatedCost float64) (be.Decision, error) {
	// Evaluate at window end so available_to_date == full monthly limit (paced ceiling matches the
	// flat limit the legacy gate used).
	win := budgetWindow(b, time.Now())
	view := newMonthlyLimitView(b, win)

	// Phase 2 (#644): prefer the real, persisted spend ledger as the SpendSource. The engine's plan
	// is still synthesized from the project budget (view), but spend comes from the append-only
	// ledger scoped to this project when it has data. If the ledger is empty (e.g. spend hasn't
	// accrued yet, or a fresh install), fall back to the synthesized cached-SpentAmount event so the
	// gate never becomes MORE permissive than Phase 1 while the ledger warms up.
	spendSource := be.SpendSource(view)
	scope := projectScope(projectID)
	if s.spendLedger != nil && projectID != "" {
		if evs, err := s.spendLedger.Spend(context.Background(), scope); err == nil && len(evs) > 0 {
			spendSource = s.spendLedger
		}
	}
	eng := be.New(spendSource, view, be.FixedClock{T: win.End},
		bepolicyDefaults()...)
	return eng.CheckLaunch(context.Background(), scope, engineAllocationID, estimatedCost)
}
