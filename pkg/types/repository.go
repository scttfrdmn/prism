package types

import (
	"time"
)

// TemplateRepository represents a template repository configuration
type TemplateRepository struct {
	Name        string    `json:"name"`
	URL         string    `json:"url"`
	Priority    int       `json:"priority"`
	Enabled     bool      `json:"enabled"`
	LastSync    time.Time `json:"last_sync"`
	TemplateCount int     `json:"template_count"`
	Description string    `json:"description"`
	Owner       string    `json:"owner,omitempty"`
	AuthType    string    `json:"auth_type,omitempty"` // none, token, ssh
}

// RepositoryStatus represents the current status of template repositories
type RepositoryStatus struct {
	Repositories []TemplateRepository `json:"repositories"`
	DefaultRepo  string               `json:"default_repo"`
	SyncEnabled  bool                 `json:"sync_enabled"`
	LastSyncTime time.Time            `json:"last_sync_time"`
}

// TemplateRepositoryUpdate represents a request to update repository settings
type TemplateRepositoryUpdate struct {
	Name     string `json:"name"`
	URL      string `json:"url"`
	Priority int    `json:"priority"`
	Enabled  bool   `json:"enabled"`
}