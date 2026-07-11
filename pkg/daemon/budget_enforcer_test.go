package daemon

import (
	"context"
	"testing"
	"time"

	"github.com/scttfrdmn/prism/pkg/alerting"
	"github.com/scttfrdmn/prism/pkg/project"
	"github.com/scttfrdmn/prism/pkg/seam/filestore"
	"github.com/scttfrdmn/prism/pkg/types"
)

// stubExecutor records which enforcement actions were invoked, per project.
type stubExecutor struct {
	hibernate []string
	stop      []string
	prevent   []string
}

func (s *stubExecutor) ExecuteHibernateAll(projectID string) error {
	s.hibernate = append(s.hibernate, projectID)
	return nil
}
func (s *stubExecutor) ExecuteStopAll(projectID string) error {
	s.stop = append(s.stop, projectID)
	return nil
}
func (s *stubExecutor) ExecutePreventLaunch(projectID string) error {
	s.prevent = append(s.prevent, projectID)
	return nil
}

// captureAlerter records sent alerts.
type captureAlerter struct{ sent []alerting.Alert }

func (c *captureAlerter) Send(_ context.Context, a alerting.Alert) error {
	c.sent = append(c.sent, a)
	return nil
}
func (c *captureAlerter) SendBatch(_ context.Context, a []alerting.Alert) error {
	c.sent = append(c.sent, a...)
	return nil
}
func (c *captureAlerter) History(int) []alerting.SentAlert { return nil }

// newEnforcerHarness builds a budgetEnforcer over an in-memory project manager with the given project,
// a stub executor, a capturing alerter, and a status fn returning the supplied status.
func newEnforcerHarness(t *testing.T, proj *types.Project, status *project.BudgetStatus) (*budgetEnforcer, *stubExecutor, *captureAlerter) {
	t.Helper()
	dir := t.TempDir()
	pm, err := project.NewManagerWithStore(filestore.New[types.Project](dir + "/projects"))
	if err != nil {
		t.Fatalf("project manager: %v", err)
	}
	exec := &stubExecutor{}
	alerter := &captureAlerter{}
	enf := &budgetEnforcer{
		enabled:     true,
		interval:    time.Minute,
		clock:       func() time.Time { return status.LastUpdated },
		executor:    exec,
		cushionEval: project.NewCushionEvaluator(exec, alerter),
		alerter:     alerter,
		status:      func(_ context.Context, _ string) (*project.BudgetStatus, error) { return status, nil },
		listBudgets: func() ([]*types.Project, error) { return []*types.Project{proj}, nil },
		saveBudget: func(ctx context.Context, id string, b *types.ProjectBudget) error {
			return pm.SetProjectBudget(ctx, id, b)
		},
	}
	return enf, exec, alerter
}

func monthlyBudgetProject(spentPct float64, actions []types.BudgetAutoAction, cushion *types.CushionBudgetConfig) (*types.Project, *project.BudgetStatus) {
	now := time.Date(2026, 7, 15, 12, 0, 0, 0, time.UTC)
	total := 1000.0
	proj := &types.Project{
		ID:   "proj-1",
		Name: "Test",
		Budget: &types.ProjectBudget{
			TotalBudget:  total,
			BudgetPeriod: types.BudgetPeriodMonthly,
			StartDate:    time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC),
			AutoActions:  actions,
			Cushion:      cushion,
		},
	}
	status := &project.BudgetStatus{
		ProjectID:       "proj-1",
		BudgetEnabled:   true,
		TotalBudget:     total,
		SpentAmount:     total * spentPct,
		RemainingBudget: total * (1 - spentPct),
		SpentPercentage: spentPct,
		LastUpdated:     now,
	}
	return proj, status
}

// TestEnforcer_AutoActionFires: an enabled hibernate action at/above its threshold fires once, sends
// an alert, and stamps LastTriggered.
func TestEnforcer_AutoActionFires(t *testing.T) {
	proj, status := monthlyBudgetProject(0.90, []types.BudgetAutoAction{
		{Threshold: 0.80, Action: types.BudgetActionHibernateAll, Enabled: true},
	}, nil)
	enf, exec, alerter := newEnforcerHarness(t, proj, status)

	enf.Enforce(context.Background())

	if len(exec.hibernate) != 1 || exec.hibernate[0] != "proj-1" {
		t.Fatalf("hibernate calls = %v, want [proj-1]", exec.hibernate)
	}
	if len(alerter.sent) != 1 {
		t.Fatalf("alerts sent = %d, want 1", len(alerter.sent))
	}
	if proj.Budget.AutoActions[0].LastTriggered == nil {
		t.Fatal("LastTriggered should be set after firing")
	}
}

// TestEnforcer_DedupWithinPeriod: a second pass in the same budget period does not re-fire; advancing
// the clock into the next period re-arms it.
func TestEnforcer_DedupWithinPeriod(t *testing.T) {
	proj, status := monthlyBudgetProject(0.90, []types.BudgetAutoAction{
		{Threshold: 0.80, Action: types.BudgetActionHibernateAll, Enabled: true},
	}, nil)
	enf, exec, _ := newEnforcerHarness(t, proj, status)

	enf.Enforce(context.Background())
	// Second pass, same period — advance the clock past the throttle but stay in July.
	status.LastUpdated = status.LastUpdated.Add(2 * time.Minute)
	enf.Enforce(context.Background())
	if len(exec.hibernate) != 1 {
		t.Fatalf("hibernate calls = %d, want 1 (deduped within period)", len(exec.hibernate))
	}

	// Advance into the next month → re-armed.
	status.LastUpdated = time.Date(2026, 8, 2, 12, 0, 0, 0, time.UTC)
	enf.Enforce(context.Background())
	if len(exec.hibernate) != 2 {
		t.Fatalf("hibernate calls = %d, want 2 (re-armed next period)", len(exec.hibernate))
	}
}

// TestEnforcer_NotifyOnlyNoExecutor: a notify-only action sends an alert but calls no executor method.
func TestEnforcer_NotifyOnlyNoExecutor(t *testing.T) {
	proj, status := monthlyBudgetProject(0.95, []types.BudgetAutoAction{
		{Threshold: 0.90, Action: types.BudgetActionNotifyOnly, Enabled: true},
	}, nil)
	enf, exec, alerter := newEnforcerHarness(t, proj, status)

	enf.Enforce(context.Background())
	if len(exec.hibernate)+len(exec.stop)+len(exec.prevent) != 0 {
		t.Fatal("notify-only must not call any executor method")
	}
	if len(alerter.sent) != 1 {
		t.Fatalf("alerts sent = %d, want 1", len(alerter.sent))
	}
}

// TestEnforcer_CushionTriggers: a cushion whose headroom is breached fires its mode (prevent-launch)
// and stamps Cushion.LastTriggeredAt.
func TestEnforcer_CushionTriggers(t *testing.T) {
	// 95%% spent → $50 remaining; headroom 10%% of $1000 = $100 → remaining <= headroom → triggered.
	proj, status := monthlyBudgetProject(0.95, nil, &types.CushionBudgetConfig{
		Enabled: true, HeadroomPercent: 0.10, Mode: string(project.CushionModePreventLaunch),
	})
	enf, exec, _ := newEnforcerHarness(t, proj, status)

	enf.Enforce(context.Background())
	if len(exec.prevent) != 1 {
		t.Fatalf("prevent-launch calls = %d, want 1 (cushion breached)", len(exec.prevent))
	}
	if proj.Budget.Cushion.LastTriggeredAt == nil {
		t.Fatal("Cushion.LastTriggeredAt should be set after firing")
	}
}

// TestEnforcer_DisabledNoOp: with enabled=false nothing fires regardless of config.
func TestEnforcer_DisabledNoOp(t *testing.T) {
	proj, status := monthlyBudgetProject(0.99, []types.BudgetAutoAction{
		{Threshold: 0.50, Action: types.BudgetActionStopAll, Enabled: true},
	}, nil)
	enf, exec, alerter := newEnforcerHarness(t, proj, status)
	enf.enabled = false

	enf.Enforce(context.Background())
	if len(exec.stop) != 0 || len(alerter.sent) != 0 {
		t.Fatal("disabled enforcer must be a no-op")
	}
}

// TestEnforcer_BelowThreshold: spend under the action threshold does not fire.
func TestEnforcer_BelowThreshold(t *testing.T) {
	proj, status := monthlyBudgetProject(0.40, []types.BudgetAutoAction{
		{Threshold: 0.80, Action: types.BudgetActionHibernateAll, Enabled: true},
	}, nil)
	enf, exec, _ := newEnforcerHarness(t, proj, status)

	enf.Enforce(context.Background())
	if len(exec.hibernate) != 0 {
		t.Fatalf("hibernate calls = %d, want 0 (below threshold)", len(exec.hibernate))
	}
}

// TestEnforcer_DisabledActionSkipped: an action with Enabled=false does not fire even over threshold.
func TestEnforcer_DisabledActionSkipped(t *testing.T) {
	proj, status := monthlyBudgetProject(0.90, []types.BudgetAutoAction{
		{Threshold: 0.80, Action: types.BudgetActionHibernateAll, Enabled: false},
	}, nil)
	enf, exec, _ := newEnforcerHarness(t, proj, status)

	enf.Enforce(context.Background())
	if len(exec.hibernate) != 0 {
		t.Fatalf("hibernate calls = %d, want 0 (action disabled)", len(exec.hibernate))
	}
}
