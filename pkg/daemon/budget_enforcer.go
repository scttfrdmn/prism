package daemon

import (
	"context"
	"os"
	"time"

	"github.com/scttfrdmn/prism/pkg/alerting"
	"github.com/scttfrdmn/prism/pkg/project"
	"github.com/scttfrdmn/prism/pkg/types"
)

// Live budget enforcement (#656). The per-project auto-actions (ProjectBudget.AutoActions — threshold
// %% → hibernate/stop/prevent-launch/notify) and the cushion (ProjectBudget.Cushion — headroom → one
// mode) are persisted config, but nothing evaluated them after the BudgetTracker was retired (its
// checkBudgetAlerts loop went with it). This enforcer restores that loop, now folding the budgetengine
// spend ledger (via Server.budgetStatus) so enforcement and the status readout agree.
//
// It runs on the StateMonitor tick (alongside the spend observer), self-throttled to its own interval.
// OFF by default (config.BudgetEnforcementEnabled) because it takes destructive actions; dedup is
// fire-once-per-budget-period via the LastTriggered timestamps on each action / the cushion, so a
// breach fires once per month (or once per project-lifetime window) rather than every tick.

// enforcerStatusFn folds a project's ledger into a BudgetStatus (Server.budgetStatus). Injected so the
// enforcer is unit-testable without a live daemon.
type enforcerStatusFn func(ctx context.Context, projectID string) (*project.BudgetStatus, error)

// budgetEnforcer evaluates budgets and fires auto-actions. It depends only on small injected
// collaborators (executor, status fn, project list/save), never AWS directly.
type budgetEnforcer struct {
	enabled  bool
	interval time.Duration
	clock    func() time.Time

	executor    project.ActionExecutor           // hibernate/stop/prevent-launch (Server implements it)
	cushionEval *project.CushionEvaluator        // headroom-mode evaluator (wraps executor + alerter)
	alerter     alerting.AlertDispatcher         // budget/auto-action notifications
	status      enforcerStatusFn                 // ledger-derived per-project BudgetStatus
	listBudgets func() ([]*types.Project, error) // projects to evaluate (filtered to Budget != nil)
	saveBudget  func(ctx context.Context, projectID string, b *types.ProjectBudget) error

	lastRun time.Time // throttle
}

// newBudgetEnforcer builds the enforcer from the Server's collaborators. The alerter is a webhook
// dispatcher when PRISM_SLACK_WEBHOOK is set (reviving the retired tracker's selection), else a log
// dispatcher. The Server itself is the ActionExecutor (ExecuteHibernateAll/StopAll/PreventLaunch).
func (s *Server) newBudgetEnforcer(config *Config) *budgetEnforcer {
	alerter := budgetAlerter()
	return &budgetEnforcer{
		enabled:     config.BudgetEnforcementEnabled,
		interval:    config.GetBudgetEnforcementInterval(),
		clock:       time.Now,
		executor:    s,
		cushionEval: project.NewCushionEvaluator(s, alerter),
		alerter:     alerter,
		status:      s.budgetStatus,
		listBudgets: func() ([]*types.Project, error) {
			return s.projectManager.ListProjects(context.Background(), nil)
		},
		saveBudget: func(ctx context.Context, projectID string, b *types.ProjectBudget) error {
			return s.projectManager.SetProjectBudget(ctx, projectID, b)
		},
	}
}

// budgetAlerter selects the notification backend: a Slack webhook dispatcher when PRISM_SLACK_WEBHOOK
// is configured, otherwise the default log dispatcher.
func budgetAlerter() alerting.AlertDispatcher {
	if url := os.Getenv("PRISM_SLACK_WEBHOOK"); url != "" {
		return alerting.NewWebhookDispatcher(alerting.WebhookConfig{
			Channels: []alerting.Channel{{Type: alerting.ChannelSlack, Target: url}},
		})
	}
	return alerting.NewLogDispatcher()
}

// Enforce runs one enforcement pass over all budgeted projects, if enabled and the throttle interval
// has elapsed. Errors on a single project are logged and skipped (fail-open) so one bad project never
// stalls the loop.
func (e *budgetEnforcer) Enforce(ctx context.Context) {
	if e == nil || !e.enabled || e.status == nil || e.listBudgets == nil {
		return
	}
	now := e.clock()
	if !e.lastRun.IsZero() && now.Sub(e.lastRun) < e.interval {
		return // throttled
	}
	e.lastRun = now

	projects, err := e.listBudgets()
	if err != nil {
		return
	}
	for _, proj := range projects {
		if proj.Budget == nil {
			continue
		}
		e.enforceProject(ctx, proj, now)
	}
}

// enforceProject evaluates one project's budget against the ledger and fires any due actions. It
// mutates the project's budget copy (LastTriggered stamps) and persists only when something fired.
func (e *budgetEnforcer) enforceProject(ctx context.Context, proj *types.Project, now time.Time) {
	status, err := e.status(ctx, proj.ID)
	if err != nil || status == nil || !status.BudgetEnabled {
		return
	}
	budget := proj.Budget
	dirty := false

	// Threshold-based auto-actions: fire when spent%% has reached the action's threshold and it hasn't
	// already fired this budget period.
	for i := range budget.AutoActions {
		action := &budget.AutoActions[i]
		if !action.Enabled || status.SpentPercentage < action.Threshold {
			continue
		}
		if firedThisPeriod(action.LastTriggered, now, budget) {
			continue
		}
		e.fireAutoAction(ctx, proj, *action, status)
		action.LastTriggered = ptrTime(now)
		dirty = true
	}

	// Cushion: fire when remaining budget is at/below the configured headroom.
	if budget.Cushion != nil && budget.Cushion.Enabled && !firedThisPeriod(budget.Cushion.LastTriggeredAt, now, budget) {
		cfg := toCushionConfig(budget.Cushion)
		if e.cushionEval.Evaluate(status, cfg).Triggered {
			_ = e.cushionEval.Execute(ctx, proj.ID, proj.Name, cfg, *status)
			budget.Cushion.LastTriggeredAt = ptrTime(now)
			dirty = true
		}
	}

	if dirty && e.saveBudget != nil {
		_ = e.saveBudget(ctx, proj.ID, budget)
	}
}

// fireAutoAction dispatches a single auto-action's effect (via the executor) and sends a notification.
// notify-only actions send an alert without touching instances.
func (e *budgetEnforcer) fireAutoAction(ctx context.Context, proj *types.Project, action types.BudgetAutoAction, status *project.BudgetStatus) {
	if e.alerter != nil {
		alert := alerting.FormatAutoActionAlert(proj.ID, proj.Name, action.Action, status.SpentAmount, status.TotalBudget)
		_ = e.alerter.Send(ctx, alert)
	}
	if e.executor == nil {
		return
	}
	switch action.Action {
	case types.BudgetActionHibernateAll:
		_ = e.executor.ExecuteHibernateAll(proj.ID)
	case types.BudgetActionStopAll:
		_ = e.executor.ExecuteStopAll(proj.ID)
	case types.BudgetActionPreventLaunch:
		_ = e.executor.ExecutePreventLaunch(proj.ID)
	case types.BudgetActionNotifyOnly:
		// alert already sent above
	}
}

// firedThisPeriod reports whether a LastTriggered timestamp falls within the current budget period, so
// an action fires at most once per period. Reuses budgetWindow (the same window the engine paces over)
// so a monthly budget re-arms each month and a project-lifetime budget re-arms per its rolling window.
func firedThisPeriod(last *time.Time, now time.Time, budget *types.ProjectBudget) bool {
	if last == nil {
		return false
	}
	win := budgetWindow(budget, now)
	return !last.Before(win.Start)
}

// toCushionConfig maps the persisted types.CushionBudgetConfig onto the project.CushionConfig the
// CushionEvaluator consumes (the mapping the retired tracker did).
func toCushionConfig(c *types.CushionBudgetConfig) project.CushionConfig {
	return project.CushionConfig{
		Enabled:            c.Enabled,
		HeadroomPercent:    c.HeadroomPercent,
		HeadroomFixed:      c.HeadroomFixed,
		Mode:               project.CushionMode(c.Mode),
		NotifyBeforeAction: c.NotifyBeforeAction,
		WarnLeadHours:      c.WarnLeadHours,
	}
}

func ptrTime(t time.Time) *time.Time { return &t }
