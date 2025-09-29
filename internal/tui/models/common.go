// Package models contains the models for the TUI components.
package models

import (
	"context"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/scttfrdmn/cloudworkstation/internal/tui/api"
)

// Common message types used across multiple models

// RefreshMsg is sent when data should be refreshed
type RefreshMsg struct{}

// ErrorMsg represents an error message
type ErrorMsg struct {
	Error error
}

// Common interface for API clients that ensures proper mocking in tests
type apiClient interface {
	// Instance operations
	ListInstances(ctx context.Context) (*api.ListInstancesResponse, error)
	GetInstance(ctx context.Context, name string) (*api.InstanceResponse, error)
	LaunchInstance(ctx context.Context, req api.LaunchInstanceRequest) (*api.LaunchInstanceResponse, error)
	StartInstance(ctx context.Context, name string) error
	StopInstance(ctx context.Context, name string) error
	DeleteInstance(ctx context.Context, name string) error

	// Template operations
	ListTemplates(ctx context.Context) (*api.ListTemplatesResponse, error)
	GetTemplate(ctx context.Context, name string) (*api.TemplateResponse, error)

	// Storage operations
	ListVolumes(ctx context.Context) (*api.ListVolumesResponse, error)
	ListStorage(ctx context.Context) (*api.ListStorageResponse, error)
	MountVolume(ctx context.Context, volumeName, instanceName, mountPoint string) error
	UnmountVolume(ctx context.Context, volumeName, instanceName string) error

	// Idle detection operations
	ListIdlePolicies(ctx context.Context) (*api.ListIdlePoliciesResponse, error)
	UpdateIdlePolicy(ctx context.Context, req api.IdlePolicyUpdateRequest) error
	GetInstanceIdleStatus(ctx context.Context, name string) (*api.IdleDetectionResponse, error)
	EnableIdleDetection(ctx context.Context, name, policy string) error
	DisableIdleDetection(ctx context.Context, name string) error
}

// refreshRoutine schedules periodic refresh operations
func refreshRoutine(interval time.Duration) tea.Cmd {
	return tea.Tick(interval, func(t time.Time) tea.Msg {
		return RefreshMsg{}
	})
}
