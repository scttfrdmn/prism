// Package project provides project and budget management functionality for Prism.
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
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/scttfrdmn/prism/pkg/alerting"
	"github.com/scttfrdmn/prism/pkg/seam"
	"github.com/scttfrdmn/prism/pkg/seam/filestore"
	"github.com/scttfrdmn/prism/pkg/types"
)

// Manager handles project lifecycle, budget tracking, and cost controls.
//
// Persistence runs through the seam (design §5): an in-memory map fronts a
// seam.Store[types.Project] of per-project records. Per-project records (rather than one big
// projects.json) are what make shared state safe — prp (web) and Prism (desktop) editing
// different projects don't clobber each other's writes (design §4).
type Manager struct {
	store               seam.Store[types.Project]
	mutex               sync.RWMutex
	projects            map[string]*types.Project
	budgetTracker       *BudgetTracker
	activeInstancesFunc func(projectID string) ([]string, error)
}

// projectScope is the tenancy key projects are stored under. Prism projects are single-tenant
// today, so this is the zero Scope; the cloud deployment overrides it per Principal.
var projectScope = seam.Scope{}

// SetActiveInstancesFunc registers a callback used by DeleteProject to check
// whether any running instances belong to the given project. The daemon wires
// this up against the state manager so the project manager does not need a
// direct dependency on pkg/state.
func (m *Manager) SetActiveInstancesFunc(fn func(projectID string) ([]string, error)) {
	m.activeInstancesFunc = fn
}

// NewManager creates a new project manager
func NewManager() (*Manager, error) {
	stateDir := os.Getenv("PRISM_STATE_DIR")
	if stateDir == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("failed to get home directory: %w", err)
		}
		stateDir = filepath.Join(homeDir, ".prism")
	}
	if err := os.MkdirAll(stateDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create state directory: %w", err)
	}

	budgetTracker, err := NewBudgetTracker()
	if err != nil {
		return nil, fmt.Errorf("failed to create budget tracker: %w", err)
	}

	manager := &Manager{
		store:         filestore.New[types.Project](filepath.Join(stateDir, "projects")),
		projects:      make(map[string]*types.Project),
		budgetTracker: budgetTracker,
	}

	// One-time migration of a legacy flat projects.json into the seam, then load.
	if err := manager.migrateLegacyProjects(filepath.Join(stateDir, "projects.json")); err != nil {
		return nil, fmt.Errorf("failed to migrate legacy projects: %w", err)
	}
	if err := manager.loadProjects(); err != nil {
		return nil, fmt.Errorf("failed to load projects: %w", err)
	}

	return manager, nil
}

// NewManagerWithStore builds a Manager over an injected seam store — used by tests (filestore in a
// temp dir) and the cloud (dynamostore). Same logic regardless of backend.
func NewManagerWithStore(store seam.Store[types.Project], budgetTracker *BudgetTracker) (*Manager, error) {
	m := &Manager{
		store:         store,
		projects:      make(map[string]*types.Project),
		budgetTracker: budgetTracker,
	}
	if err := m.loadProjects(); err != nil {
		return nil, fmt.Errorf("failed to load projects: %w", err)
	}
	return m, nil
}

// migrateLegacyProjects imports a pre-seam flat projects.json (map[id]*Project) into the store,
// then renames it aside so the import runs once. Absent file → no-op.
func (m *Manager) migrateLegacyProjects(legacyPath string) error {
	// #nosec G304 G703 -- legacyPath is the manager's own ~/.prism/projects.json, composed
	// internally from the state dir, not external input.
	data, err := os.ReadFile(legacyPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	var projects map[string]*types.Project
	if err := json.Unmarshal(data, &projects); err != nil {
		return fmt.Errorf("parse legacy projects: %w", err)
	}
	ctx := context.Background()
	for id, p := range projects {
		if p == nil {
			continue
		}
		if err := m.store.Put(ctx, projectScope, id, *p); err != nil {
			return fmt.Errorf("migrate project %q: %w", id, err)
		}
	}
	// #nosec G703 -- legacyPath is internally composed (see above), not external input.
	return os.Rename(legacyPath, legacyPath+".migrated")
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
			return nil, ErrDuplicateProjectName
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
				return nil, ErrDuplicateProjectName
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

	// Validate role
	validRoles := map[types.ProjectRole]bool{
		types.ProjectRoleOwner:  true,
		types.ProjectRoleAdmin:  true,
		types.ProjectRoleMember: true,
		types.ProjectRoleViewer: true,
	}
	if !validRoles[member.Role] {
		return fmt.Errorf("invalid role %q: must be one of owner, admin, member, or viewer", member.Role)
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

// loadProjects loads all project records from the seam into the in-memory map.
func (m *Manager) loadProjects() error {
	all, err := m.store.List(context.Background(), projectScope)
	if err != nil {
		return fmt.Errorf("failed to load projects: %w", err)
	}
	projects := make(map[string]*types.Project, len(all))
	for i := range all {
		p := all[i]
		projects[p.ID] = &p
	}
	m.projects = projects
	return nil
}

// saveProjects reconciles the seam store to the in-memory map: every project in the map is
// written, and any record in the store no longer in the map is deleted. This keeps all the
// existing mutation call sites (which mutate the map then call saveProjects) working unchanged,
// while persisting one record per project — the granularity shared state needs (design §4).
func (m *Manager) saveProjects() error {
	ctx := context.Background()

	existing, err := m.store.List(ctx, projectScope)
	if err != nil {
		return fmt.Errorf("failed to list projects for save: %w", err)
	}

	// Delete records that are no longer present in the map (handles DeleteProject).
	for i := range existing {
		id := existing[i].ID
		if _, ok := m.projects[id]; !ok {
			if err := m.store.Delete(ctx, projectScope, id); err != nil && !errorsIsNotFound(err) {
				return fmt.Errorf("failed to delete project %q: %w", id, err)
			}
		}
	}

	// Put every project currently in the map.
	for id, p := range m.projects {
		if p == nil {
			continue
		}
		if err := m.store.Put(ctx, projectScope, id, *p); err != nil {
			return fmt.Errorf("failed to save project %q: %w", id, err)
		}
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

// getActiveInstancesForProject returns the names of running instances that
// belong to projectID. It delegates to activeInstancesFunc when set (wired by
// the daemon via SetActiveInstancesFunc). Without that hook the check is a
// no-op so that the project package compiles and runs standalone.
func (m *Manager) getActiveInstancesForProject(projectID string) ([]string, error) {
	if m.activeInstancesFunc != nil {
		return m.activeInstancesFunc(projectID)
	}
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

// PreventLaunches prevents new instance launches for a project
func (m *Manager) PreventLaunches(ctx context.Context, projectID string) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	project, exists := m.projects[projectID]
	if !exists {
		return fmt.Errorf("project not found: %s", projectID)
	}

	project.LaunchPrevented = true
	project.UpdatedAt = time.Now()

	if err := m.saveProjects(); err != nil {
		return fmt.Errorf("failed to save project: %w", err)
	}

	return nil
}

// AllowLaunches allows new instance launches for a project (clears launch prevention)
func (m *Manager) AllowLaunches(ctx context.Context, projectID string) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	project, exists := m.projects[projectID]
	if !exists {
		return fmt.Errorf("project not found: %s", projectID)
	}

	project.LaunchPrevented = false
	project.UpdatedAt = time.Now()

	if err := m.saveProjects(); err != nil {
		return fmt.Errorf("failed to save project: %w", err)
	}

	return nil
}

// IsLaunchPrevented checks if launches are prevented for a project
func (m *Manager) IsLaunchPrevented(ctx context.Context, projectID string) (bool, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	project, exists := m.projects[projectID]
	if !exists {
		return false, fmt.Errorf("project not found: %s", projectID)
	}

	return project.LaunchPrevented, nil
}

// SetDefaultAllocation sets the default funding allocation for a project (v0.5.10+)
func (m *Manager) SetDefaultAllocation(ctx context.Context, projectID string, allocationID string) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	project, exists := m.projects[projectID]
	if !exists {
		return fmt.Errorf("project not found: %s", projectID)
	}

	// Set the default allocation
	project.DefaultAllocationID = &allocationID
	project.UpdatedAt = time.Now()

	// Save projects to disk
	if err := m.saveProjects(); err != nil {
		return fmt.Errorf("failed to save project: %w", err)
	}

	return nil
}

// ClearDefaultAllocation clears the default funding allocation for a project (v0.5.10+)
func (m *Manager) ClearDefaultAllocation(ctx context.Context, projectID string) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	project, exists := m.projects[projectID]
	if !exists {
		return fmt.Errorf("project not found: %s", projectID)
	}

	// Clear the default allocation
	project.DefaultAllocationID = nil
	project.UpdatedAt = time.Now()

	// Save projects to disk
	if err := m.saveProjects(); err != nil {
		return fmt.Errorf("failed to save project: %w", err)
	}

	return nil
}

// ===========================================================================
// Project Transfer and Forecast Operations (Issue #326)
// ===========================================================================

// TransferProject transfers project ownership to a new owner
func (m *Manager) TransferProject(ctx context.Context, projectID string, req *TransferProjectRequest) (*types.Project, error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	// Validate request
	if err := req.Validate(); err != nil {
		return nil, fmt.Errorf("invalid transfer request: %w", err)
	}

	// Get project
	project, exists := m.projects[projectID]
	if !exists {
		return nil, fmt.Errorf("project %q not found", projectID)
	}

	// Verify current ownership (transferredBy must be current owner)
	if project.Owner != req.TransferredBy {
		return nil, fmt.Errorf("only project owner can transfer ownership")
	}

	// Prevent transfer to same owner
	if project.Owner == req.NewOwnerID {
		return nil, fmt.Errorf("project already owned by user %q", req.NewOwnerID)
	}

	// Update owner
	oldOwner := project.Owner
	project.Owner = req.NewOwnerID
	project.UpdatedAt = time.Now()

	// Update members list - demote old owner to admin, promote new owner
	var updatedMembers []types.ProjectMember
	newOwnerFound := false

	for _, member := range project.Members {
		if member.UserID == oldOwner {
			// Demote old owner to admin
			member.Role = types.ProjectRoleAdmin
			member.AddedAt = time.Now()
			updatedMembers = append(updatedMembers, member)
		} else if member.UserID == req.NewOwnerID {
			// Promote new owner
			member.Role = types.ProjectRoleOwner
			member.AddedAt = time.Now()
			updatedMembers = append(updatedMembers, member)
			newOwnerFound = true
		} else {
			updatedMembers = append(updatedMembers, member)
		}
	}

	// If new owner wasn't already a member, add them
	if !newOwnerFound {
		updatedMembers = append(updatedMembers, types.ProjectMember{
			UserID:  req.NewOwnerID,
			Role:    types.ProjectRoleOwner,
			AddedAt: time.Now(),
			AddedBy: oldOwner,
		})
	}

	project.Members = updatedMembers

	// Save changes
	if err := m.saveProjects(); err != nil {
		return nil, fmt.Errorf("failed to save project transfer: %w", err)
	}

	projectCopy := *project
	return &projectCopy, nil
}

// GetProjectForecast generates cost forecast data for a project using linear regression.
func (m *Manager) GetProjectForecast(ctx context.Context, projectID string, req *ProjectForecastRequest) (*ProjectForecastResponse, error) {
	m.mutex.RLock()
	project, exists := m.projects[projectID]
	m.mutex.RUnlock()
	if !exists {
		return nil, fmt.Errorf("project %q not found", projectID)
	}

	months := req.Months
	if months <= 0 {
		months = 6
	}

	// Retrieve cost history for regression.
	history, err := m.budgetTracker.GetCostHistory(projectID)
	if err != nil {
		history = nil // fall back to rate-only projection
	}

	budget := project.Budget
	if budget == nil {
		budget = &types.ProjectBudget{}
	}

	predictor := NewBudgetPredictor()
	prediction, err := predictor.Predict(projectID, history, budget, months, nil)
	if err != nil {
		return nil, fmt.Errorf("forecast failed: %w", err)
	}

	// Map ShortfallPrediction → legacy ProjectForecastResponse shape.
	forecastData := make([]ForecastDataPoint, 0, len(prediction.MonthlyForecasts))
	for _, fm := range prediction.MonthlyForecasts {
		if fm.IsProjected {
			forecastData = append(forecastData, ForecastDataPoint{
				Month:          fm.Month,
				ProjectedCost:  fm.ProjectedSpend,
				CumulativeCost: fm.CumulativeSpend,
			})
		}
	}

	var historicalData []ForecastDataPoint
	if req.IncludeHistorical {
		for _, fm := range prediction.MonthlyForecasts {
			if !fm.IsProjected {
				cost := fm.ProjectedSpend
				if fm.ActualSpend != nil {
					cost = *fm.ActualSpend
				}
				historicalData = append(historicalData, ForecastDataPoint{
					Month:          fm.Month,
					ProjectedCost:  cost,
					CumulativeCost: fm.CumulativeSpend,
				})
			}
		}
	}

	confidence := 0.5
	switch prediction.ConfidenceLevel {
	case "high":
		confidence = 0.90
	case "medium":
		confidence = 0.75
	}

	return &ProjectForecastResponse{
		ProjectID:           projectID,
		GeneratedAt:         time.Now(),
		CurrentMonthlyRate:  prediction.CurrentDailyRate * 30,
		ForecastData:        forecastData,
		HistoricalData:      historicalData,
		ProjectedExhaustion: prediction.PredictedExhaustionAt,
		Confidence:          confidence,
	}, nil
}

// buildHistoricalData builds historical spending data (simplified implementation)
func (m *Manager) buildHistoricalData(projectID string, months int) []ForecastDataPoint {
	// This is a simplified implementation
	// In a full implementation, this would pull actual historical data from cost tracking
	now := time.Now()
	historicalData := make([]ForecastDataPoint, 0, months)

	// For now, return empty historical data
	// TODO: Implement actual historical data retrieval
	for i := months; i > 0; i-- {
		pastMonth := now.AddDate(0, -i, 0)
		monthStr := pastMonth.Format("2006-01")

		historicalData = append(historicalData, ForecastDataPoint{
			Month:          monthStr,
			ProjectedCost:  0,
			CumulativeCost: 0,
			ActualCost:     nil,
		})
	}

	return historicalData
}

// SetAlerter passes an AlertDispatcher through to the underlying BudgetTracker.
// This allows operators to configure notification backends (Slack, webhook, etc.)
// at daemon startup without re-creating the budget tracker.
func (m *Manager) SetAlerter(d alerting.AlertDispatcher) {
	if m.budgetTracker != nil {
		m.budgetTracker.SetAlerter(d)
	}
}

// ===========================================================================
// v0.12.0: Time-Boxed Collaborator Access (#150)
// ===========================================================================

// PruneExpiredMembers removes members whose ExpiresAt has passed.
// Intended to be called by a daemon background ticker (hourly).
// Returns the list of removed member UserIDs.
func (m *Manager) PruneExpiredMembers(ctx context.Context) ([]string, error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	now := time.Now()
	var removed []string

	for _, project := range m.projects {
		var remaining []types.ProjectMember
		for _, member := range project.Members {
			if member.ExpiresAt != nil && now.After(*member.ExpiresAt) {
				removed = append(removed, fmt.Sprintf("%s/%s", project.ID, member.UserID))
			} else {
				remaining = append(remaining, member)
			}
		}
		if len(remaining) != len(project.Members) {
			project.Members = remaining
			project.UpdatedAt = now
		}
	}

	if len(removed) == 0 {
		return nil, nil
	}

	return removed, m.saveProjects()
}

// ===========================================================================
// v0.12.0: Resource Quotas by Role (#151)
// ===========================================================================

// CheckQuota verifies whether a launch by the given user satisfies the project's
// role quotas. instanceCount is the number of instances already owned by this user.
// Returns a non-nil error if the launch would violate a quota.
func (m *Manager) CheckQuota(projectID, userID, instanceType string, instanceCount int) error {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	project, exists := m.projects[projectID]
	if !exists {
		return nil // no project = no quota constraint
	}

	if project.Budget == nil || len(project.Budget.RoleQuotas) == 0 {
		return nil // no quotas configured
	}

	// Find the user's role
	var userRole types.ProjectRole
	for _, member := range project.Members {
		if member.UserID == userID {
			userRole = member.Role
			break
		}
	}

	if userRole == "" {
		return nil // not a member, no quota
	}

	// Find the matching quota for this role
	for _, quota := range project.Budget.RoleQuotas {
		if quota.Role != userRole {
			continue
		}

		// Check instance count
		if quota.MaxInstances > 0 && instanceCount >= quota.MaxInstances {
			return fmt.Errorf("quota exceeded: role %q is limited to %d instance(s) (currently have %d)",
				userRole, quota.MaxInstances, instanceCount)
		}

		// Check instance type prefix
		if quota.MaxInstanceType != "" && !strings.HasPrefix(instanceType, quota.MaxInstanceType) {
			return fmt.Errorf("quota exceeded: role %q is limited to %s* instance types (requested %q)",
				userRole, quota.MaxInstanceType, instanceType)
		}

		return nil // quota satisfied
	}

	return nil
}

// GetRoleQuotas returns the role quotas for a project's budget.
func (m *Manager) GetRoleQuotas(projectID string) ([]types.RoleQuota, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	project, exists := m.projects[projectID]
	if !exists {
		return nil, fmt.Errorf("project %q not found", projectID)
	}

	if project.Budget == nil {
		return nil, nil
	}

	return append([]types.RoleQuota(nil), project.Budget.RoleQuotas...), nil
}

// SetRoleQuota upserts a role quota for a project.
func (m *Manager) SetRoleQuota(ctx context.Context, projectID string, quota types.RoleQuota) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	project, exists := m.projects[projectID]
	if !exists {
		return fmt.Errorf("project %q not found", projectID)
	}

	if project.Budget == nil {
		project.Budget = &types.ProjectBudget{}
	}

	// Upsert
	updated := false
	for i, q := range project.Budget.RoleQuotas {
		if q.Role == quota.Role {
			project.Budget.RoleQuotas[i] = quota
			updated = true
			break
		}
	}
	if !updated {
		project.Budget.RoleQuotas = append(project.Budget.RoleQuotas, quota)
	}

	project.UpdatedAt = time.Now()
	return m.saveProjects()
}

// ===========================================================================
// v0.12.0: Grant Period Management (#152)
// ===========================================================================

// SetGrantPeriod sets the grant period for a project budget.
func (m *Manager) SetGrantPeriod(ctx context.Context, projectID string, gp *types.GrantPeriod) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	project, exists := m.projects[projectID]
	if !exists {
		return fmt.Errorf("project %q not found", projectID)
	}

	if project.Budget == nil {
		project.Budget = &types.ProjectBudget{}
	}

	project.Budget.GrantPeriod = gp
	project.UpdatedAt = time.Now()
	return m.saveProjects()
}

// GetGrantPeriod returns the current grant period for a project.
func (m *Manager) GetGrantPeriod(projectID string) (*types.GrantPeriod, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	project, exists := m.projects[projectID]
	if !exists {
		return nil, fmt.Errorf("project %q not found", projectID)
	}

	if project.Budget == nil {
		return nil, nil
	}

	return project.Budget.GrantPeriod, nil
}

// DeleteGrantPeriod removes the grant period from a project.
func (m *Manager) DeleteGrantPeriod(ctx context.Context, projectID string) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	project, exists := m.projects[projectID]
	if !exists {
		return fmt.Errorf("project %q not found", projectID)
	}

	if project.Budget != nil {
		project.Budget.GrantPeriod = nil
	}

	project.UpdatedAt = time.Now()
	return m.saveProjects()
}

// CheckGrantPeriods reviews all projects and auto-freezes any where the grant period
// has ended and AutoFreeze=true. Intended to be called by a daemon ticker (daily).
// Returns the list of auto-frozen project IDs.
func (m *Manager) CheckGrantPeriods(ctx context.Context) ([]string, error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	now := time.Now()
	var frozen []string

	for _, project := range m.projects {
		if project.Budget == nil || project.Budget.GrantPeriod == nil {
			continue
		}
		gp := project.Budget.GrantPeriod
		if !gp.AutoFreeze || gp.FrozenAt != nil {
			continue
		}
		if now.After(gp.EndDate) {
			project.LaunchPrevented = true
			project.Status = types.ProjectStatusPaused
			project.UpdatedAt = now
			gp.FrozenAt = &now
			frozen = append(frozen, project.ID)
		}
	}

	if len(frozen) == 0 {
		return nil, nil
	}

	return frozen, m.saveProjects()
}

// CurrentMonthAllocation returns the per-month allocation for a multi-month budget (#144).
// Returns TotalBudget if AllocationMonths <= 1.
func (m *Manager) CurrentMonthAllocation(projectID string) (float64, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	project, exists := m.projects[projectID]
	if !exists {
		return 0, fmt.Errorf("project %q not found", projectID)
	}

	if project.Budget == nil {
		return 0, nil
	}

	if project.Budget.MonthlyAmount > 0 {
		return project.Budget.MonthlyAmount, nil
	}

	if project.Budget.AllocationMonths > 1 {
		return project.Budget.TotalBudget / float64(project.Budget.AllocationMonths), nil
	}

	return project.Budget.TotalBudget, nil
}

// ===========================================================================
// v0.12.0: Budget Sharing / Reallocation / Cross-Project Borrowing (#143,#145,#155,#156)
// ===========================================================================

// ShareBudget executes a budget share between projects or members.
// The amount is subtracted from the source project's TotalBudget and added to
// the destination, with a BudgetShareRecord created for audit purposes.
func (m *Manager) ShareBudget(ctx context.Context, req *types.BudgetShareRequest, approvedBy string) (*types.BudgetShareRecord, error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	// Validate source project
	srcProject, exists := m.projects[req.FromProjectID]
	if !exists {
		return nil, fmt.Errorf("source project %q not found", req.FromProjectID)
	}

	if req.Amount <= 0 {
		return nil, fmt.Errorf("share amount must be positive")
	}

	// Verify sufficient budget
	if srcProject.Budget != nil {
		available := srcProject.Budget.TotalBudget - srcProject.Budget.SpentAmount
		if req.Amount > available {
			return nil, fmt.Errorf("insufficient budget: requested $%.2f but only $%.2f available", req.Amount, available)
		}

		// Deduct from source
		srcProject.Budget.TotalBudget -= req.Amount
		srcProject.UpdatedAt = time.Now()
	}

	// Credit destination project if specified
	if req.ToProjectID != "" {
		dstProject, ok := m.projects[req.ToProjectID]
		if !ok {
			// Rollback source
			if srcProject.Budget != nil {
				srcProject.Budget.TotalBudget += req.Amount
			}
			return nil, fmt.Errorf("destination project %q not found", req.ToProjectID)
		}
		if dstProject.Budget == nil {
			dstProject.Budget = &types.ProjectBudget{}
		}
		dstProject.Budget.TotalBudget += req.Amount
		dstProject.UpdatedAt = time.Now()
	}

	record := &types.BudgetShareRecord{
		ID:         uuid.New().String(),
		Request:    *req,
		ApprovedBy: approvedBy,
		CreatedAt:  time.Now(),
		ExpiresAt:  req.ExpiresAt,
	}

	if err := m.saveProjects(); err != nil {
		return nil, fmt.Errorf("failed to save share: %w", err)
	}

	return record, nil
}

// ===========================================================================
// v0.12.0: Onboarding Templates (#154)
// ===========================================================================

// SetOnboardingTemplate upserts an onboarding template for a project.
func (m *Manager) SetOnboardingTemplate(ctx context.Context, projectID string, tmpl types.OnboardingTemplate) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	project, exists := m.projects[projectID]
	if !exists {
		return fmt.Errorf("project %q not found", projectID)
	}

	if tmpl.ID == "" {
		tmpl.ID = uuid.New().String()
	}

	for i, t := range project.OnboardingTemplates {
		if t.ID == tmpl.ID || t.Name == tmpl.Name {
			project.OnboardingTemplates[i] = tmpl
			project.UpdatedAt = time.Now()
			return m.saveProjects()
		}
	}

	project.OnboardingTemplates = append(project.OnboardingTemplates, tmpl)
	project.UpdatedAt = time.Now()
	return m.saveProjects()
}

// DeleteOnboardingTemplate removes an onboarding template by name or ID.
func (m *Manager) DeleteOnboardingTemplate(ctx context.Context, projectID, nameOrID string) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	project, exists := m.projects[projectID]
	if !exists {
		return fmt.Errorf("project %q not found", projectID)
	}

	var remaining []types.OnboardingTemplate
	for _, t := range project.OnboardingTemplates {
		if t.ID != nameOrID && t.Name != nameOrID {
			remaining = append(remaining, t)
		}
	}

	project.OnboardingTemplates = remaining
	project.UpdatedAt = time.Now()
	return m.saveProjects()
}
