package project

import (
	"time"

	"github.com/scttfrdmn/prism/pkg/types"
)

// This file holds cost/action types that outlive the retired BudgetTracker (#653). They were
// originally defined in budget_tracker.go but are referenced by the surviving analytics helpers
// (burn_rate.go, surplus.go, predictor.go, reports.go) and by the daemon's ActionExecutor
// implementation, so they move here when the tracker is deleted.

// ActionExecutor executes budget auto-actions for a project. The daemon (Server) implements it
// (hibernate/stop/prevent-launch). It survives the tracker retirement because CushionEvaluator and
// the eventual engine ActionSink wiring (#656) consume it.
type ActionExecutor interface {
	// ExecuteHibernateAll hibernates all instances for a project.
	ExecuteHibernateAll(projectID string) error
	// ExecuteStopAll stops all instances for a project.
	ExecuteStopAll(projectID string) error
	// ExecutePreventLaunch sets a flag to prevent new launches for a project.
	ExecutePreventLaunch(projectID string) error
}

// CostDataPoint is a point-in-time cost measurement. It is the input shape the analytics helpers
// (burn rate, surplus, predictor, monthly report) accept; post-tracker the daemon can build these
// from the budgetengine spend ledger (or pass an empty series until Phase 3d wires that).
type CostDataPoint struct {
	Timestamp     time.Time            `json:"timestamp"`
	TotalCost     float64              `json:"total_cost"`
	InstanceCosts []types.InstanceCost `json:"instance_costs"`
	StorageCosts  []types.StorageCost  `json:"storage_costs"`
	DailyCost     float64              `json:"daily_cost"`
}
