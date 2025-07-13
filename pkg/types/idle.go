package types

import (
	"time"
)

// IdlePolicy defines an idle detection policy
type IdlePolicy struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Threshold   int      `json:"threshold"` // in minutes
	Action      string   `json:"action"`    // stop, hibernate, notify
	AppliesTo   []string `json:"applies_to"` // list of instance types
}

// IdleStatus represents the current idle status of an instance
type IdleStatus struct {
	Instance       string    `json:"instance"`
	Enabled        bool      `json:"enabled"`
	Policy         string    `json:"policy,omitempty"`
	IdleTime       int       `json:"idle_time"`       // in minutes
	LastActivity   time.Time `json:"last_activity"`
	ActionSchedule time.Time `json:"action_schedule"` // when action will be taken
	ActionPending  bool      `json:"action_pending"`
}

// IdleActivity represents an activity event for an instance
type IdleActivity struct {
	Instance   string    `json:"instance"`
	ActivityID string    `json:"activity_id"`
	Type       string    `json:"type"`  // ssh, web, desktop, cpu, etc.
	Level      int       `json:"level"` // 0-100
	Time       time.Time `json:"time"`
}