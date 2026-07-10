package daemon

import (
	"context"
	"time"

	be "github.com/scttfrdmn/budgetengine"
	"github.com/scttfrdmn/prism/pkg/seam"
)

// Phase 2 (#644): a real, persisted, append-only spend ledger backing budgetengine's SpendSource /
// SpendWriter, replacing Phase 1's synthetic single spend event. Events are seam records, so they
// survive restart and (later) can be shared with prp exactly like the other seam domains.
//
// Spend is scoped per project: the seam partition key is seam.Scope{Project: projectID}. The engine
// works in its own budgetengine.Scope (layout-identical to seam.Principal), so the adapter converts
// at the boundary.

// seamToEngineScope / engineToSeamScope are the plain conversions between the two identical Scope
// types (the whole reason the layouts were kept identical — see budgetengine/scope.go).
func engineToSeamScope(s be.Scope) seam.Scope {
	return seam.Scope(s)
}

// projectScope builds the per-project spend partition key.
func projectScope(projectID string) be.Scope {
	return be.Scope{Project: projectID}
}

// spendCheckpoint records the last cumulative spend observed for one instance, so the observer can
// convert a cumulative estimate/billed figure into an append-only delta (newCumulative − last).
// Persisted as its own seam record (durable across daemon restart) to avoid re-appending spend.
type spendCheckpoint struct {
	InstanceID       string    `json:"instance_id"`
	LastCumulative   float64   `json:"last_cumulative"`   // last cumulative estimate appended
	LastAccrualAt    time.Time `json:"last_accrual_at"`   // throttle: last estimate append
	LastReconcileAt  time.Time `json:"last_reconcile_at"` // throttle: last billed reconciliation
	ReconciledBilled float64   `json:"reconciled_billed"` // last authoritative billed total applied
}

// spendStore adapts two seam stores (spend events + per-instance checkpoints) into the engine's
// SpendSource and SpendWriter ports. One instance per daemon; scope is passed per call.
type spendStore struct {
	events      seam.Store[be.SpendEvent]
	checkpoints seam.Store[spendCheckpoint]
}

func newSpendStore(events seam.Store[be.SpendEvent], checkpoints seam.Store[spendCheckpoint]) *spendStore {
	return &spendStore{events: events, checkpoints: checkpoints}
}

// Spend implements budgetengine.SpendSource: all spend events for the scope, in append order.
func (s *spendStore) Spend(ctx context.Context, scope be.Scope) ([]be.SpendEvent, error) {
	return s.events.List(ctx, engineToSeamScope(scope))
}

// AppendSpend implements budgetengine.SpendWriter: idempotent by SpendEvent.ID (Put replaces, and
// the observer uses a unique-per-delta ID so replays are no-ops in the fold).
func (s *spendStore) AppendSpend(ctx context.Context, scope be.Scope, e be.SpendEvent) error {
	return s.events.Put(ctx, engineToSeamScope(scope), e.ID, e)
}

// checkpoint fetches an instance's checkpoint (zero value + false when absent).
func (s *spendStore) checkpoint(ctx context.Context, scope be.Scope, instanceID string) (spendCheckpoint, bool) {
	cp, err := s.checkpoints.Get(ctx, engineToSeamScope(scope), instanceID)
	if err != nil {
		return spendCheckpoint{InstanceID: instanceID}, false
	}
	return cp, true
}

// saveCheckpoint persists an instance's checkpoint.
func (s *spendStore) saveCheckpoint(ctx context.Context, scope be.Scope, cp spendCheckpoint) error {
	return s.checkpoints.Put(ctx, engineToSeamScope(scope), cp.InstanceID, cp)
}

// compile-time port checks.
var (
	_ be.SpendSource = (*spendStore)(nil)
	_ be.SpendWriter = (*spendStore)(nil)
)
