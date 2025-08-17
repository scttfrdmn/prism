package api

import (
	"context"
	"fmt"

	"github.com/scttfrdmn/cloudworkstation/pkg/types"
)

// RepositoryResponse represents the API response for repository operations
type RepositoryResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

// ListRepositories retrieves all template repositories
func (c *Client) ListRepositories(ctx context.Context) ([]types.TemplateRepository, error) {
	return c.ListRepositoriesLegacy()
}

// ListRepositoriesLegacy retrieves all template repositories without context
func (c *Client) ListRepositoriesLegacy() ([]types.TemplateRepository, error) {
	var repos []types.TemplateRepository
	err := c.get("/api/v1/repositories", &repos)
	if err != nil {
		return nil, fmt.Errorf("failed to list repositories: %w", err)
	}
	return repos, nil
}

// GetRepository retrieves a specific template repository
func (c *Client) GetRepository(ctx context.Context, name string) (*types.TemplateRepository, error) {
	return c.GetRepositoryLegacy(name)
}

// GetRepositoryLegacy retrieves a specific template repository without context
func (c *Client) GetRepositoryLegacy(name string) (*types.TemplateRepository, error) {
	var repo types.TemplateRepository
	err := c.get(fmt.Sprintf("/api/v1/repositories/%s", name), &repo)
	if err != nil {
		return nil, fmt.Errorf("failed to get repository: %w", err)
	}
	return &repo, nil
}

// AddRepository adds a new template repository
func (c *Client) AddRepository(ctx context.Context, repo types.TemplateRepositoryUpdate) error {
	return c.AddRepositoryLegacy(repo)
}

// AddRepositoryLegacy adds a new template repository without context
func (c *Client) AddRepositoryLegacy(repo types.TemplateRepositoryUpdate) error {
	var resp RepositoryResponse
	err := c.post("/api/v1/repositories", repo, &resp)
	if err != nil {
		return fmt.Errorf("failed to add repository: %w", err)
	}

	if !resp.Success {
		return fmt.Errorf("API error: %s", resp.Message)
	}

	return nil
}

// UpdateRepository updates a template repository
func (c *Client) UpdateRepository(ctx context.Context, repo types.TemplateRepositoryUpdate) error {
	return c.UpdateRepositoryLegacy(repo)
}

// UpdateRepositoryLegacy updates a template repository without context
func (c *Client) UpdateRepositoryLegacy(repo types.TemplateRepositoryUpdate) error {
	var resp RepositoryResponse
	err := c.put(fmt.Sprintf("/api/v1/repositories/%s", repo.Name), repo, &resp)
	if err != nil {
		return fmt.Errorf("failed to update repository: %w", err)
	}

	if !resp.Success {
		return fmt.Errorf("API error: %s", resp.Message)
	}

	return nil
}

// RemoveRepository removes a template repository
func (c *Client) RemoveRepository(ctx context.Context, name string) error {
	return c.RemoveRepositoryLegacy(name)
}

// RemoveRepositoryLegacy removes a template repository without context
func (c *Client) RemoveRepositoryLegacy(name string) error {
	err := c.delete(fmt.Sprintf("/api/v1/repositories/%s", name))
	if err != nil {
		return fmt.Errorf("failed to remove repository: %w", err)
	}

	return nil
}

// SyncRepositories synchronizes all template repositories
func (c *Client) SyncRepositories(ctx context.Context) error {
	return c.SyncRepositoriesLegacy()
}

// SyncRepositoriesLegacy synchronizes all template repositories without context
func (c *Client) SyncRepositoriesLegacy() error {
	var resp RepositoryResponse
	err := c.post("/api/v1/repositories/sync", nil, &resp)
	if err != nil {
		return fmt.Errorf("failed to sync repositories: %w", err)
	}

	if !resp.Success {
		return fmt.Errorf("API error: %s", resp.Message)
	}

	return nil
}

// GetRepositoryStatus retrieves the current status of template repositories
func (c *Client) GetRepositoryStatus(ctx context.Context) (*types.RepositoryStatus, error) {
	return c.GetRepositoryStatusLegacy()
}

// GetRepositoryStatusLegacy retrieves the current status of template repositories without context
func (c *Client) GetRepositoryStatusLegacy() (*types.RepositoryStatus, error) {
	var status types.RepositoryStatus
	err := c.get("/api/v1/repositories/status", &status)
	if err != nil {
		return nil, fmt.Errorf("failed to get repository status: %w", err)
	}
	return &status, nil
}

// NOTE: These methods are now implemented in client.go
