package types

import (
	"time"
)

// InstanceExtended adds monitoring information to Instance
type InstanceExtended struct {
	Instance
	IdleDetection *InstanceIdleStatus `json:"idle_detection,omitempty"`
}

// InstanceIdleStatus represents idle detection status for an instance
type InstanceIdleStatus struct {
	Enabled        bool      `json:"enabled"`
	Policy         string    `json:"policy"`
	IdleTime       int       `json:"idle_time"`
	Threshold      int       `json:"threshold"`
	ActionSchedule time.Time `json:"action_schedule"`
	ActionPending  bool      `json:"action_pending"`
}
