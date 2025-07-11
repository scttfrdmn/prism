// Package idle provides idle detection for CloudWorkstation instances.
//
// This package implements the core functionality for monitoring resource usage
// on CloudWorkstation instances and taking action when instances are determined
// to be idle. It is inspired by the CloudSnooze project.
package idle

import (
	"time"
)

// Action represents the action to take when an instance is determined to be idle.
type Action string

const (
	// Stop stops the instance.
	Stop Action = "stop"

	// Hibernate hibernates the instance.
	Hibernate Action = "hibernate"

	// Notify sends a notification without taking action.
	Notify Action = "notify"
)

// Profile represents an idle detection profile with thresholds.
type Profile struct {
	// Name is the profile name
	Name string `json:"name" yaml:"name"`

	// CPUThreshold is the CPU usage threshold percentage
	CPUThreshold float64 `json:"cpu_threshold" yaml:"cpu_threshold"`

	// MemoryThreshold is the memory usage threshold percentage
	MemoryThreshold float64 `json:"memory_threshold" yaml:"memory_threshold"`

	// NetworkThreshold is the network activity threshold in KBps
	NetworkThreshold float64 `json:"network_threshold" yaml:"network_threshold"`

	// DiskThreshold is the disk I/O threshold in KBps
	DiskThreshold float64 `json:"disk_threshold" yaml:"disk_threshold"`

	// GPUThreshold is the GPU usage threshold percentage
	GPUThreshold float64 `json:"gpu_threshold" yaml:"gpu_threshold"`

	// IdleMinutes is the minutes before an action is taken
	IdleMinutes int `json:"idle_minutes" yaml:"idle_minutes"`

	// Action is the action to take when idle
	Action Action `json:"action" yaml:"action"`

	// Notification indicates whether to send a notification
	Notification bool `json:"notification" yaml:"notification"`
}

// Config represents the idle detection configuration.
type Config struct {
	// Enabled indicates whether idle detection is enabled
	Enabled bool `json:"enabled" yaml:"enabled"`

	// DefaultProfile is the name of the default profile
	DefaultProfile string `json:"default_profile" yaml:"default_profile"`

	// Profiles is a map of profile names to profiles
	Profiles map[string]Profile `json:"profiles" yaml:"profiles"`

	// DomainMappings maps research domains to profiles
	DomainMappings map[string]string `json:"domain_mappings" yaml:"domain_mappings"`

	// InstanceOverrides maps instance names to profile overrides
	InstanceOverrides map[string]InstanceOverride `json:"instance_overrides" yaml:"instance_overrides"`
}

// InstanceOverride represents an instance-specific override for idle detection.
type InstanceOverride struct {
	// Profile is the profile name to use
	Profile string `json:"profile" yaml:"profile"`

	// CPUThreshold overrides the CPU threshold (optional)
	CPUThreshold *float64 `json:"cpu_threshold,omitempty" yaml:"cpu_threshold,omitempty"`

	// MemoryThreshold overrides the memory threshold (optional)
	MemoryThreshold *float64 `json:"memory_threshold,omitempty" yaml:"memory_threshold,omitempty"`

	// NetworkThreshold overrides the network threshold (optional)
	NetworkThreshold *float64 `json:"network_threshold,omitempty" yaml:"network_threshold,omitempty"`

	// DiskThreshold overrides the disk threshold (optional)
	DiskThreshold *float64 `json:"disk_threshold,omitempty" yaml:"disk_threshold,omitempty"`

	// GPUThreshold overrides the GPU threshold (optional)
	GPUThreshold *float64 `json:"gpu_threshold,omitempty" yaml:"gpu_threshold,omitempty"`

	// IdleMinutes overrides the idle minutes (optional)
	IdleMinutes *int `json:"idle_minutes,omitempty" yaml:"idle_minutes,omitempty"`

	// Action overrides the action (optional)
	Action *Action `json:"action,omitempty" yaml:"action,omitempty"`

	// Notification overrides the notification setting (optional)
	Notification *bool `json:"notification,omitempty" yaml:"notification,omitempty"`
}

// UsageMetrics represents resource usage metrics for an instance.
type UsageMetrics struct {
	// Timestamp is the time when the metrics were collected
	Timestamp time.Time `json:"timestamp" yaml:"timestamp"`

	// CPU usage percentage (0-100)
	CPU float64 `json:"cpu" yaml:"cpu"`

	// Memory usage percentage (0-100)
	Memory float64 `json:"memory" yaml:"memory"`

	// Network activity in KBps
	Network float64 `json:"network" yaml:"network"`

	// Disk I/O in KBps
	Disk float64 `json:"disk" yaml:"disk"`

	// GPU usage percentage (0-100), if available
	GPU *float64 `json:"gpu,omitempty" yaml:"gpu,omitempty"`

	// HasActivity indicates user activity (keyboard, mouse, etc.)
	HasActivity bool `json:"has_activity" yaml:"has_activity"`
}

// IdleState represents the idle state of an instance.
type IdleState struct {
	// InstanceID is the AWS instance ID
	InstanceID string `json:"instance_id" yaml:"instance_id"`

	// InstanceName is the CloudWorkstation instance name
	InstanceName string `json:"instance_name" yaml:"instance_name"`

	// Profile is the idle detection profile being used
	Profile string `json:"profile" yaml:"profile"`

	// IsIdle indicates whether the instance is currently idle
	IsIdle bool `json:"is_idle" yaml:"is_idle"`

	// IdleSince is the time when the instance became idle
	IdleSince *time.Time `json:"idle_since,omitempty" yaml:"idle_since,omitempty"`

	// LastActivity is the last time activity was detected
	LastActivity time.Time `json:"last_activity" yaml:"last_activity"`

	// NextAction is the next scheduled action
	NextAction *ScheduledAction `json:"next_action,omitempty" yaml:"next_action,omitempty"`

	// LastMetrics contains the most recent usage metrics
	LastMetrics *UsageMetrics `json:"last_metrics,omitempty" yaml:"last_metrics,omitempty"`
}

// ScheduledAction represents a scheduled idle action.
type ScheduledAction struct {
	// Action is the action to take
	Action Action `json:"action" yaml:"action"`

	// Time is when the action will be taken
	Time time.Time `json:"time" yaml:"time"`
}

// HistoryEntry represents an entry in the idle history.
type HistoryEntry struct {
	// InstanceID is the AWS instance ID
	InstanceID string `json:"instance_id" yaml:"instance_id"`

	// InstanceName is the CloudWorkstation instance name
	InstanceName string `json:"instance_name" yaml:"instance_name"`

	// Action is the action that was taken
	Action Action `json:"action" yaml:"action"`

	// Time is when the action was taken
	Time time.Time `json:"time" yaml:"time"`

	// IdleDuration is how long the instance was idle
	IdleDuration time.Duration `json:"idle_duration" yaml:"idle_duration"`

	// Metrics contains the usage metrics at the time of the action
	Metrics *UsageMetrics `json:"metrics,omitempty" yaml:"metrics,omitempty"`
}

// History represents the idle detection history.
type History struct {
	// Entries is a list of history entries
	Entries []HistoryEntry `json:"entries" yaml:"entries"`
}

// Notification represents an idle detection notification.
type Notification struct {
	// InstanceID is the AWS instance ID
	InstanceID string `json:"instance_id" yaml:"instance_id"`

	// InstanceName is the CloudWorkstation instance name
	InstanceName string `json:"instance_name" yaml:"instance_name"`

	// Message is the notification message
	Message string `json:"message" yaml:"message"`

	// Action is the action that will be taken
	Action Action `json:"action" yaml:"action"`

	// Time is when the action will be taken
	Time time.Time `json:"time" yaml:"time"`

	// IsWarning indicates whether this is a warning notification
	IsWarning bool `json:"is_warning" yaml:"is_warning"`
}