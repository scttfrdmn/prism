// Package project provides project and budget management functionality for CloudWorkstation.
//
// This package implements project-based resource organization, budget tracking,
// and cost controls that enable researchers to organize instances, storage, and
// costs around research projects with proper financial oversight.
package project

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/scttfrdmn/cloudworkstation/pkg/types"
)

// Manager handles project lifecycle, budget tracking, and cost controls
type Manager struct {
	projectsPath  string
	mutex         sync.RWMutex
	projects      map[string]*types.Project
	budgetTracker *BudgetTracker
}

// NewManager creates a new project manager
func NewManager() (*Manager, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}

	stateDir := filepath.Join(homeDir, ".cloudworkstation")
	if err := os.MkdirAll(stateDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create state directory: %w", err)
	}

	projectsPath := filepath.Join(stateDir, "projects.json")

	budgetTracker, err := NewBudgetTracker()
	if err != nil {
		return nil, fmt.Errorf("failed to create budget tracker: %w", err)
	}

	manager := &Manager{
		projectsPath:  projectsPath,
		projects:      make(map[string]*types.Project),
		budgetTracker: budgetTracker,
	}

	// Load existing projects
	if err := manager.loadProjects(); err != nil {
		return nil, fmt.Errorf("failed to load projects: %w", err)
	}

	return manager, nil
}

// CreateProject creates a new research project
func (m *Manager) CreateProject(ctx context.Context, req *CreateProjectRequest) (*types.Project, error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	// Validate request
	if err := req.Validate(); err != nil {
		return nil, fmt.Errorf("invalid project request: %w", err)
	}

	// Check for duplicate names
	for _, project := range m.projects {
		if project.Name == req.Name {
			return nil, fmt.Errorf("project with name %q already exists", req.Name)
		}
	}

	// Create project
	project := &types.Project{
		ID:          uuid.New().String(),
		Name:        req.Name,
		Description: req.Description,
		Owner:       req.Owner,
		Members:     []types.ProjectMember{},
		Tags:        req.Tags,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		Status:      types.ProjectStatusActive,
	}

	// Add owner as project member
	if req.Owner != "" {
		project.Members = append(project.Members, types.ProjectMember{
			UserID:  req.Owner,
			Role:    types.ProjectRoleOwner,
			AddedAt: time.Now(),
			AddedBy: req.Owner,
		})
	}

	// Create budget if specified
	if req.Budget != nil {
		budget := &types.ProjectBudget{
			TotalBudget:     req.Budget.TotalBudget,
			SpentAmount:     0.0,
			MonthlyLimit:    req.Budget.MonthlyLimit,
			DailyLimit:      req.Budget.DailyLimit,
			AlertThresholds: req.Budget.AlertThresholds,
			AutoActions:     req.Budget.AutoActions,
			BudgetPeriod:    req.Budget.BudgetPeriod,
			StartDate:       time.Now(),
			EndDate:         req.Budget.EndDate,
			LastUpdated:     time.Now(),
		}
		project.Budget = budget

		// Initialize budget tracking
		if err := m.budgetTracker.InitializeProject(project.ID, budget); err != nil {
			return nil, fmt.Errorf("failed to initialize budget tracking: %w", err)
		}
	}

	// Store project
	m.projects[project.ID] = project
	if err := m.saveProjects(); err != nil {
		delete(m.projects, project.ID)
		return nil, fmt.Errorf("failed to save project: %w", err)
	}

	return project, nil
}

// GetProject retrieves a project by ID
func (m *Manager) GetProject(ctx context.Context, projectID string) (*types.Project, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	project, exists := m.projects[projectID]
	if !exists {
		return nil, fmt.Errorf("project %q not found", projectID)
	}

	// Return a copy to prevent external modification
	projectCopy := *project
	return &projectCopy, nil
}

// GetProjectByName retrieves a project by name
func (m *Manager) GetProjectByName(ctx context.Context, name string) (*types.Project, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	for _, project := range m.projects {
		if project.Name == name {
			projectCopy := *project
			return &projectCopy, nil
		}
	}

	return nil, fmt.Errorf("project with name %q not found", name)
}

// ListProjects retrieves projects with optional filtering
func (m *Manager) ListProjects(ctx context.Context, filter *ProjectFilter) ([]*types.Project, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	var results []*types.Project

	for _, project := range m.projects {
		if filter != nil && !filter.Matches(project) {
			continue
		}

		projectCopy := *project
		results = append(results, &projectCopy)
	}

	return results, nil
}

// UpdateProject updates an existing project
func (m *Manager) UpdateProject(ctx context.Context, projectID string, req *UpdateProjectRequest) (*types.Project, error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	project, exists := m.projects[projectID]
	if !exists {
		return nil, fmt.Errorf("project %q not found", projectID)
	}

	// Update fields
	if req.Name != nil {
		// Check for duplicate names
		for id, p := range m.projects {
			if id != projectID && p.Name == *req.Name {
				return nil, fmt.Errorf("project with name %q already exists", *req.Name)
			}
		}
		project.Name = *req.Name
	}

	if req.Description != nil {
		project.Description = *req.Description
	}

	if req.Tags != nil {
		project.Tags = req.Tags
	}

	if req.Status != nil {
		project.Status = *req.Status
	}

	project.UpdatedAt = time.Now()

	// Save changes
	if err := m.saveProjects(); err != nil {
		return nil, fmt.Errorf("failed to save project updates: %w", err)
	}

	projectCopy := *project
	return &projectCopy, nil
}

// DeleteProject removes a project and all associated data
func (m *Manager) DeleteProject(ctx context.Context, projectID string) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	project, exists := m.projects[projectID]
	if !exists {
		return fmt.Errorf("project %q not found", projectID)
	}

	// Check for active resources before deletion
	activeInstances, err := m.getActiveInstancesForProject(projectID)
	if err != nil {
		return fmt.Errorf("failed to check active resources: %w", err)
	}
	if len(activeInstances) > 0 {
		return fmt.Errorf("cannot delete project with %d active instances - stop instances first", len(activeInstances))
	}

	// Clean up budget tracking
	if err := m.budgetTracker.RemoveProject(projectID); err != nil {
		return fmt.Errorf("failed to clean up budget tracking: %w", err)
	}

	// Remove project
	delete(m.projects, projectID)
	if err := m.saveProjects(); err != nil {
		// Restore project on save failure
		m.projects[projectID] = project
		return fmt.Errorf("failed to save project deletion: %w", err)
	}

	return nil
}

// AddProjectMember adds a member to a project
func (m *Manager) AddProjectMember(ctx context.Context, projectID string, member *types.ProjectMember) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	project, exists := m.projects[projectID]
	if !exists {
		return fmt.Errorf("project %q not found", projectID)
	}

	// Check if member already exists
	for _, existingMember := range project.Members {
		if existingMember.UserID == member.UserID {
			return fmt.Errorf("user %q is already a member of project %q", member.UserID, projectID)
		}
	}

	// Add member
	member.AddedAt = time.Now()
	project.Members = append(project.Members, *member)
	project.UpdatedAt = time.Now()

	return m.saveProjects()
}

// RemoveProjectMember removes a member from a project
func (m *Manager) RemoveProjectMember(ctx context.Context, projectID, userID string) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	project, exists := m.projects[projectID]
	if !exists {
		return fmt.Errorf("project %q not found", projectID)
	}

	// Find and remove member
	memberIndex := -1
	for i, member := range project.Members {
		if member.UserID == userID {
			memberIndex = i
			break
		}
	}

	if memberIndex == -1 {
		return fmt.Errorf("user %q is not a member of project %q", userID, projectID)
	}

	// Don't allow removal of the last owner
	if project.Members[memberIndex].Role == types.ProjectRoleOwner {
		ownerCount := 0
		for _, member := range project.Members {
			if member.Role == types.ProjectRoleOwner {
				ownerCount++
			}
		}
		if ownerCount <= 1 {
			return fmt.Errorf("cannot remove the last owner of project %q", projectID)
		}
	}

	// Remove member
	project.Members = append(project.Members[:memberIndex], project.Members[memberIndex+1:]...)
	project.UpdatedAt = time.Now()

	return m.saveProjects()
}

// UpdateProjectMember updates a member's role in a project
func (m *Manager) UpdateProjectMember(ctx context.Context, projectID, userID string, role types.ProjectRole) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	project, exists := m.projects[projectID]
	if !exists {
		return fmt.Errorf("project %q not found", projectID)
	}

	// Find member
	memberIndex := -1
	for i, member := range project.Members {
		if member.UserID == userID {
			memberIndex = i
			break
		}
	}

	if memberIndex == -1 {
		return fmt.Errorf("user %q is not a member of project %q", userID, projectID)
	}

	// Don't allow removing the last owner
	if project.Members[memberIndex].Role == types.ProjectRoleOwner && role != types.ProjectRoleOwner {
		ownerCount := 0
		for _, member := range project.Members {
			if member.Role == types.ProjectRoleOwner {
				ownerCount++
			}
		}
		if ownerCount <= 1 {
			return fmt.Errorf("cannot change the role of the last owner of project %q", projectID)
		}
	}

	// Update role
	project.Members[memberIndex].Role = role
	project.UpdatedAt = time.Now()

	return m.saveProjects()
}

// GetProjectCostBreakdown retrieves detailed cost analysis for a project
func (m *Manager) GetProjectCostBreakdown(ctx context.Context, projectID string, startDate, endDate time.Time) (*types.ProjectCostBreakdown, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	_, exists := m.projects[projectID]
	if !exists {
		return nil, fmt.Errorf("project %q not found", projectID)
	}

	return m.budgetTracker.GetCostBreakdown(projectID, startDate, endDate)
}

// GetProjectResourceUsage retrieves resource utilization metrics for a project
func (m *Manager) GetProjectResourceUsage(ctx context.Context, projectID string, period time.Duration) (*types.ProjectResourceUsage, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	_, exists := m.projects[projectID]
	if !exists {
		return nil, fmt.Errorf("project %q not found", projectID)
	}

	return m.budgetTracker.GetResourceUsage(projectID, period)
}

// CheckBudgetStatus checks the current budget status and triggers alerts if needed
func (m *Manager) CheckBudgetStatus(ctx context.Context, projectID string) (*BudgetStatus, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	project, exists := m.projects[projectID]
	if !exists {
		return nil, fmt.Errorf("project %q not found", projectID)
	}

	if project.Budget == nil {
		return &BudgetStatus{
			ProjectID:     projectID,
			BudgetEnabled: false,
		}, nil
	}

	return m.budgetTracker.CheckBudgetStatus(projectID)
}

// loadProjects loads projects from disk
func (m *Manager) loadProjects() error {
	// Check if projects file exists
	if _, err := os.Stat(m.projectsPath); os.IsNotExist(err) {
		// No projects file exists yet, start with empty map
		return nil
	}

	data, err := os.ReadFile(m.projectsPath)
	if err != nil {
		return fmt.Errorf("failed to read projects file: %w", err)
	}

	var projects map[string]*types.Project
	if err := json.Unmarshal(data, &projects); err != nil {
		return fmt.Errorf("failed to parse projects file: %w", err)
	}

	m.projects = projects
	return nil
}

// saveProjects saves projects to disk
func (m *Manager) saveProjects() error {
	data, err := json.MarshalIndent(m.projects, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal projects: %w", err)
	}

	// Write to temporary file first, then rename for atomicity
	tempPath := m.projectsPath + ".tmp"
	if err := os.WriteFile(tempPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write temporary projects file: %w", err)
	}

	if err := os.Rename(tempPath, m.projectsPath); err != nil {
		return fmt.Errorf("failed to rename projects file: %w", err)
	}

	return nil
}

// Close cleanly shuts down the project manager
func (m *Manager) Close() error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if m.budgetTracker != nil {
		if err := m.budgetTracker.Close(); err != nil {
			return fmt.Errorf("failed to close budget tracker: %w", err)
		}
	}

	return nil
}

// getActiveInstancesForProject checks for active instances in a project
// Currently returns empty slice - project-instance association needs to be implemented
// at the daemon level where both state manager and AWS manager are available
func (m *Manager) getActiveInstancesForProject(projectID string) ([]string, error) {
	// This method is called during project deletion to check if there are active instances
	// For now, return empty slice to allow project operations to proceed
	//
	// Proper implementation would require:
	// 1. Access to state manager or AWS manager (not available in project manager)
	// 2. Standardized project tagging on EC2 instances
	// 3. Query instances with Project tag = projectID and State = running
	//
	// The actual instance counting is now implemented in daemon/project_handlers.go
	// where both awsManager and budgetTracker are available

	return []string{}, nil
}

// SetProjectBudget sets or enables budget tracking for a project
func (m *Manager) SetProjectBudget(ctx context.Context, projectID string, budget *types.ProjectBudget) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	project, exists := m.projects[projectID]
	if !exists {
		return fmt.Errorf("project not found: %s", projectID)
	}

	// Set the budget on the project
	project.Budget = budget
	project.UpdatedAt = time.Now()

	// Initialize budget tracking
	if m.budgetTracker != nil {
		if err := m.budgetTracker.InitializeProject(projectID, budget); err != nil {
			return fmt.Errorf("failed to initialize budget tracking: %w", err)
		}
	}

	// Save projects to disk
	if err := m.saveProjects(); err != nil {
		return fmt.Errorf("failed to save projects: %w", err)
	}

	return nil
}

// UpdateProjectBudget updates an existing project budget
func (m *Manager) UpdateProjectBudget(ctx context.Context, projectID string, budget *types.ProjectBudget) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	project, exists := m.projects[projectID]
	if !exists {
		return fmt.Errorf("project not found: %s", projectID)
	}

	if project.Budget == nil {
		return fmt.Errorf("no budget configured for project: %s", projectID)
	}

	// Update the budget on the project
	project.Budget = budget
	project.UpdatedAt = time.Now()

	// Re-initialize budget tracking with updated configuration
	if m.budgetTracker != nil {
		if err := m.budgetTracker.InitializeProject(projectID, budget); err != nil {
			return fmt.Errorf("failed to update budget tracking: %w", err)
		}
	}

	// Save projects to disk
	if err := m.saveProjects(); err != nil {
		return fmt.Errorf("failed to save projects: %w", err)
	}

	return nil
}

// DisableProjectBudget disables budget tracking for a project
func (m *Manager) DisableProjectBudget(ctx context.Context, projectID string) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	project, exists := m.projects[projectID]
	if !exists {
		return fmt.Errorf("project not found: %s", projectID)
	}

	// Remove budget from project
	project.Budget = nil
	project.UpdatedAt = time.Now()

	// Remove from budget tracker
	if m.budgetTracker != nil {
		if err := m.budgetTracker.RemoveProject(projectID); err != nil {
			return fmt.Errorf("failed to remove budget tracking: %w", err)
		}
	}

	// Save projects to disk
	if err := m.saveProjects(); err != nil {
		return fmt.Errorf("failed to save projects: %w", err)
	}

	return nil
}
