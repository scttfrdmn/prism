// Package project provides budget and allocation management for v0.5.10+
//
// This file implements the multi-budget system enabling:
//   - 1 budget → N projects (single grant funding multiple projects)
//   - N budgets → 1 project (multi-source funding)
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
	"github.com/scttfrdmn/prism/pkg/seam"
	"github.com/scttfrdmn/prism/pkg/seam/filestore"
	"github.com/scttfrdmn/prism/pkg/types"
)

// BudgetManager handles budget pools and project allocations.
//
// Persistence runs through the seam (design §5): three seam stores (budgets, allocations,
// reallocations), each fronted by an in-memory map. Per-record storage (rather than three big
// JSON files rewritten wholesale) is what makes shared state safe across Prism and prp (§4).
type BudgetManager struct {
	budgetStore       seam.Store[types.Budget]
	allocationStore   seam.Store[types.ProjectBudgetAllocation]
	reallocationStore seam.Store[ReallocationRecord]
	mutex             sync.RWMutex
	budgets           map[string]*types.Budget                  // budget_id → Budget
	allocations       map[string]*types.ProjectBudgetAllocation // allocation_id → Allocation
	reallocations     map[string]*ReallocationRecord            // reallocation_id → Record
	// Index for efficient lookups
	projectAllocations      map[string][]*types.ProjectBudgetAllocation // project_id → [Allocations]
	budgetAllocations       map[string][]*types.ProjectBudgetAllocation // budget_id → [Allocations]
	allocationReallocations map[string][]*ReallocationRecord            // allocation_id → [Reallocations]
	// Optional project manager for name/allocation enrichment (injected via SetProjectManager).
	projectManager *Manager
}

// budgetSeamScope is the tenancy key budget records are stored under. Single-tenant on the
// desktop (the zero Scope); the cloud deployment overrides it per Principal.
var budgetSeamScope = seam.Scope{}

// migrateLegacyMap imports a legacy map[id]*T JSON file into a seam store keyed by each record's
// id, then renames the file aside. The id is read back from the JSON map key (which the old code
// used as the record ID), so no per-type field accessor is needed.
func migrateLegacyMap[T any](ctx context.Context, store seam.Store[T], scope seam.Scope, legacyPath string) error {
	// #nosec G304 G703 -- legacyPath is composed internally from the state dir, not external input.
	data, err := os.ReadFile(legacyPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	var records map[string]*T
	if err := json.Unmarshal(data, &records); err != nil {
		return fmt.Errorf("parse legacy %s: %w", filepath.Base(legacyPath), err)
	}
	for id, rec := range records {
		if rec == nil {
			continue
		}
		if err := store.Put(ctx, scope, id, *rec); err != nil {
			return fmt.Errorf("migrate %s record %q: %w", filepath.Base(legacyPath), id, err)
		}
	}
	// #nosec G703 -- legacyPath is internally composed (see above), not external input.
	return os.Rename(legacyPath, legacyPath+".migrated")
}

// SetProjectManager injects a project manager so GetBudgetSummary and
// GetProjectFundingSummary can populate project names and default allocation IDs.
func (bm *BudgetManager) SetProjectManager(m *Manager) {
	bm.projectManager = m
}

// NewBudgetManager creates a new budget manager
func NewBudgetManager() (*BudgetManager, error) {
	// Check for custom state directory via environment variable (for test isolation)
	stateDir := os.Getenv("PRISM_STATE_DIR")
	if stateDir == "" {
		// Default to ~/.prism for backward compatibility
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("failed to get home directory: %w", err)
		}
		stateDir = filepath.Join(homeDir, ".prism")
	}

	if err := os.MkdirAll(stateDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create state directory: %w", err)
	}

	manager := &BudgetManager{
		budgetStore:             filestore.New[types.Budget](filepath.Join(stateDir, "budgets")),
		allocationStore:         filestore.New[types.ProjectBudgetAllocation](filepath.Join(stateDir, "budget_allocations")),
		reallocationStore:       filestore.New[ReallocationRecord](filepath.Join(stateDir, "budget_reallocations")),
		budgets:                 make(map[string]*types.Budget),
		allocations:             make(map[string]*types.ProjectBudgetAllocation),
		reallocations:           make(map[string]*ReallocationRecord),
		projectAllocations:      make(map[string][]*types.ProjectBudgetAllocation),
		budgetAllocations:       make(map[string][]*types.ProjectBudgetAllocation),
		allocationReallocations: make(map[string][]*ReallocationRecord),
	}

	// One-time migration of the legacy flat JSON files into the seam.
	if err := manager.migrateLegacy(stateDir); err != nil {
		return nil, fmt.Errorf("failed to migrate legacy budget data: %w", err)
	}

	// Load existing data
	if err := manager.loadBudgets(); err != nil {
		return nil, fmt.Errorf("failed to load budgets: %w", err)
	}

	if err := manager.loadAllocations(); err != nil {
		return nil, fmt.Errorf("failed to load allocations: %w", err)
	}

	if err := manager.loadReallocations(); err != nil {
		return nil, fmt.Errorf("failed to load reallocations: %w", err)
	}

	// Build indexes
	manager.rebuildIndexes()

	return manager, nil
}

// CreateBudget creates a new budget pool
func (bm *BudgetManager) CreateBudget(ctx context.Context, req *CreateBudgetRequest) (*types.Budget, error) {
	bm.mutex.Lock()
	defer bm.mutex.Unlock()

	// Validate request
	if err := req.Validate(); err != nil {
		return nil, fmt.Errorf("invalid budget request: %w", err)
	}

	// Check for duplicate names
	for _, budget := range bm.budgets {
		if budget.Name == req.Name {
			return nil, fmt.Errorf("budget with name %q already exists", req.Name)
		}
	}

	// Create budget
	budget := &types.Budget{
		ID:              uuid.New().String(),
		Name:            req.Name,
		Description:     req.Description,
		TotalAmount:     req.TotalAmount,
		AllocatedAmount: 0.0,
		SpentAmount:     0.0,
		Period:          req.Period,
		StartDate:       req.StartDate,
		EndDate:         req.EndDate,
		AlertThreshold:  req.AlertThreshold,
		CreatedBy:       req.CreatedBy,
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
		Tags:            req.Tags,
	}

	// Store budget
	bm.budgets[budget.ID] = budget
	if err := bm.saveBudgets(); err != nil {
		delete(bm.budgets, budget.ID)
		return nil, fmt.Errorf("failed to save budget: %w", err)
	}

	return budget, nil
}

// GetBudget retrieves a budget by ID
func (bm *BudgetManager) GetBudget(ctx context.Context, budgetID string) (*types.Budget, error) {
	bm.mutex.RLock()
	defer bm.mutex.RUnlock()

	budget, exists := bm.budgets[budgetID]
	if !exists {
		return nil, fmt.Errorf("budget %q not found", budgetID)
	}

	// Return a copy to prevent external modification
	budgetCopy := *budget
	return &budgetCopy, nil
}

// GetBudgetByName retrieves a budget by name
func (bm *BudgetManager) GetBudgetByName(ctx context.Context, name string) (*types.Budget, error) {
	bm.mutex.RLock()
	defer bm.mutex.RUnlock()

	for _, budget := range bm.budgets {
		if budget.Name == name {
			budgetCopy := *budget
			return &budgetCopy, nil
		}
	}

	return nil, fmt.Errorf("budget with name %q not found", name)
}

// ListBudgets retrieves all budgets
func (bm *BudgetManager) ListBudgets(ctx context.Context) ([]*types.Budget, error) {
	bm.mutex.RLock()
	defer bm.mutex.RUnlock()

	results := make([]*types.Budget, 0, len(bm.budgets))
	for _, budget := range bm.budgets {
		budgetCopy := *budget
		results = append(results, &budgetCopy)
	}

	return results, nil
}

// UpdateBudget updates an existing budget
func (bm *BudgetManager) UpdateBudget(ctx context.Context, budgetID string, req *UpdateBudgetRequest) (*types.Budget, error) {
	bm.mutex.Lock()
	defer bm.mutex.Unlock()

	budget, exists := bm.budgets[budgetID]
	if !exists {
		return nil, fmt.Errorf("budget %q not found", budgetID)
	}

	// Update fields
	if req.Name != nil {
		// Check for duplicate names
		for id, b := range bm.budgets {
			if id != budgetID && b.Name == *req.Name {
				return nil, fmt.Errorf("budget with name %q already exists", *req.Name)
			}
		}
		budget.Name = *req.Name
	}

	if req.Description != nil {
		budget.Description = *req.Description
	}

	if req.TotalAmount != nil {
		budget.TotalAmount = *req.TotalAmount
	}

	if req.AlertThreshold != nil {
		budget.AlertThreshold = *req.AlertThreshold
	}

	if req.EndDate != nil {
		budget.EndDate = req.EndDate
	}

	if req.Tags != nil {
		budget.Tags = req.Tags
	}

	budget.UpdatedAt = time.Now()

	// Save changes
	if err := bm.saveBudgets(); err != nil {
		return nil, fmt.Errorf("failed to save budget updates: %w", err)
	}

	budgetCopy := *budget
	return &budgetCopy, nil
}

// DeleteBudget removes a budget and all associated allocations
func (bm *BudgetManager) DeleteBudget(ctx context.Context, budgetID string) error {
	bm.mutex.Lock()
	defer bm.mutex.Unlock()

	budget, exists := bm.budgets[budgetID]
	if !exists {
		return fmt.Errorf("budget %q not found", budgetID)
	}

	// Check if budget has allocations
	allocations := bm.budgetAllocations[budgetID]
	if len(allocations) > 0 {
		return fmt.Errorf("cannot delete budget with %d project allocations - remove allocations first", len(allocations))
	}

	// Remove budget
	delete(bm.budgets, budgetID)
	if err := bm.saveBudgets(); err != nil {
		// Restore budget on save failure
		bm.budgets[budgetID] = budget
		return fmt.Errorf("failed to save budget deletion: %w", err)
	}

	return nil
}

// CreateAllocation creates a new project budget allocation
func (bm *BudgetManager) CreateAllocation(ctx context.Context, req *CreateAllocationRequest) (*types.ProjectBudgetAllocation, error) {
	bm.mutex.Lock()
	defer bm.mutex.Unlock()

	// Validate request
	if err := req.Validate(); err != nil {
		return nil, fmt.Errorf("invalid allocation request: %w", err)
	}

	// Verify budget exists
	budget, exists := bm.budgets[req.BudgetID]
	if !exists {
		return nil, fmt.Errorf("budget %q not found", req.BudgetID)
	}

	// Check if allocation would exceed budget
	totalAllocated := budget.AllocatedAmount + req.AllocatedAmount
	if totalAllocated > budget.TotalAmount {
		return nil, fmt.Errorf("allocation would exceed budget: $%.2f allocated + $%.2f requested > $%.2f total",
			budget.AllocatedAmount, req.AllocatedAmount, budget.TotalAmount)
	}

	// Check for duplicate budget-project allocation
	for _, allocation := range bm.allocations {
		if allocation.BudgetID == req.BudgetID && allocation.ProjectID == req.ProjectID {
			return nil, fmt.Errorf("budget %q is already allocated to project %q", req.BudgetID, req.ProjectID)
		}
	}

	// Create allocation
	allocation := &types.ProjectBudgetAllocation{
		ID:                 uuid.New().String(),
		BudgetID:           req.BudgetID,
		ProjectID:          req.ProjectID,
		AllocatedAmount:    req.AllocatedAmount,
		SpentAmount:        0.0,
		AlertThreshold:     req.AlertThreshold,
		BackupAllocationID: req.BackupAllocationID,
		Notes:              req.Notes,
		AllocatedAt:        time.Now(),
		AllocatedBy:        req.AllocatedBy,
		UpdatedAt:          time.Now(),
	}

	// Update budget allocated amount
	budget.AllocatedAmount += req.AllocatedAmount
	budget.UpdatedAt = time.Now()

	// Store allocation
	bm.allocations[allocation.ID] = allocation
	bm.rebuildIndexes()

	// Save changes
	if err := bm.saveAllocations(); err != nil {
		delete(bm.allocations, allocation.ID)
		budget.AllocatedAmount -= req.AllocatedAmount
		bm.rebuildIndexes()
		return nil, fmt.Errorf("failed to save allocation: %w", err)
	}

	if err := bm.saveBudgets(); err != nil {
		delete(bm.allocations, allocation.ID)
		budget.AllocatedAmount -= req.AllocatedAmount
		bm.rebuildIndexes()
		return nil, fmt.Errorf("failed to save budget: %w", err)
	}

	return allocation, nil
}

// GetAllocation retrieves an allocation by ID
func (bm *BudgetManager) GetAllocation(ctx context.Context, allocationID string) (*types.ProjectBudgetAllocation, error) {
	bm.mutex.RLock()
	defer bm.mutex.RUnlock()

	allocation, exists := bm.allocations[allocationID]
	if !exists {
		return nil, fmt.Errorf("allocation %q not found", allocationID)
	}

	// Return a copy to prevent external modification
	allocationCopy := *allocation
	return &allocationCopy, nil
}

// GetProjectAllocations retrieves all allocations for a project
func (bm *BudgetManager) GetProjectAllocations(ctx context.Context, projectID string) ([]*types.ProjectBudgetAllocation, error) {
	bm.mutex.RLock()
	defer bm.mutex.RUnlock()

	allocations := bm.projectAllocations[projectID]
	results := make([]*types.ProjectBudgetAllocation, len(allocations))
	for i, alloc := range allocations {
		allocCopy := *alloc
		results[i] = &allocCopy
	}

	return results, nil
}

// GetBudgetAllocations retrieves all allocations for a budget
func (bm *BudgetManager) GetBudgetAllocations(ctx context.Context, budgetID string) ([]*types.ProjectBudgetAllocation, error) {
	bm.mutex.RLock()
	defer bm.mutex.RUnlock()

	allocations := bm.budgetAllocations[budgetID]
	results := make([]*types.ProjectBudgetAllocation, len(allocations))
	for i, alloc := range allocations {
		allocCopy := *alloc
		results[i] = &allocCopy
	}

	return results, nil
}

// UpdateAllocation updates an existing allocation
func (bm *BudgetManager) UpdateAllocation(ctx context.Context, allocationID string, req *UpdateAllocationRequest) (*types.ProjectBudgetAllocation, error) {
	bm.mutex.Lock()
	defer bm.mutex.Unlock()

	allocation, exists := bm.allocations[allocationID]
	if !exists {
		return nil, fmt.Errorf("allocation %q not found", allocationID)
	}

	budget, exists := bm.budgets[allocation.BudgetID]
	if !exists {
		return nil, fmt.Errorf("budget %q not found", allocation.BudgetID)
	}

	// Handle allocated amount change
	if req.AllocatedAmount != nil {
		oldAmount := allocation.AllocatedAmount
		newAmount := *req.AllocatedAmount
		delta := newAmount - oldAmount

		// Check if new allocation would exceed budget
		if budget.AllocatedAmount+delta > budget.TotalAmount {
			return nil, fmt.Errorf("allocation change would exceed budget: $%.2f total - $%.2f allocated + $%.2f delta > $%.2f available",
				budget.TotalAmount, budget.AllocatedAmount, delta, budget.TotalAmount-budget.AllocatedAmount)
		}

		allocation.AllocatedAmount = newAmount
		budget.AllocatedAmount += delta
		budget.UpdatedAt = time.Now()
	}

	// Update other fields
	if req.AlertThreshold != nil {
		allocation.AlertThreshold = req.AlertThreshold
	}

	if req.BackupAllocationID != nil {
		allocation.BackupAllocationID = req.BackupAllocationID
	}

	if req.Notes != nil {
		allocation.Notes = *req.Notes
	}

	allocation.UpdatedAt = time.Now()

	// Save changes
	if err := bm.saveAllocations(); err != nil {
		return nil, fmt.Errorf("failed to save allocation updates: %w", err)
	}

	if err := bm.saveBudgets(); err != nil {
		return nil, fmt.Errorf("failed to save budget updates: %w", err)
	}

	allocationCopy := *allocation
	return &allocationCopy, nil
}

// DeleteAllocation removes an allocation
func (bm *BudgetManager) DeleteAllocation(ctx context.Context, allocationID string) error {
	bm.mutex.Lock()
	defer bm.mutex.Unlock()

	allocation, exists := bm.allocations[allocationID]
	if !exists {
		return fmt.Errorf("allocation %q not found", allocationID)
	}

	budget, exists := bm.budgets[allocation.BudgetID]
	if !exists {
		return fmt.Errorf("budget %q not found", allocation.BudgetID)
	}

	// Update budget allocated amount
	budget.AllocatedAmount -= allocation.AllocatedAmount
	budget.UpdatedAt = time.Now()

	// Remove allocation
	delete(bm.allocations, allocationID)
	bm.rebuildIndexes()

	// Save changes
	if err := bm.saveAllocations(); err != nil {
		bm.allocations[allocationID] = allocation
		budget.AllocatedAmount += allocation.AllocatedAmount
		bm.rebuildIndexes()
		return fmt.Errorf("failed to save allocation deletion: %w", err)
	}

	if err := bm.saveBudgets(); err != nil {
		bm.allocations[allocationID] = allocation
		budget.AllocatedAmount += allocation.AllocatedAmount
		bm.rebuildIndexes()
		return fmt.Errorf("failed to save budget: %w", err)
	}

	return nil
}

// GetBudgetSummary generates a summary view of a budget
func (bm *BudgetManager) GetBudgetSummary(ctx context.Context, budgetID string) (*types.BudgetSummary, error) {
	bm.mutex.RLock()
	defer bm.mutex.RUnlock()

	budget, exists := bm.budgets[budgetID]
	if !exists {
		return nil, fmt.Errorf("budget %q not found", budgetID)
	}

	allocations := bm.budgetAllocations[budgetID]
	allocationCopies := make([]types.ProjectBudgetAllocation, len(allocations))
	for i, alloc := range allocations {
		allocationCopies[i] = *alloc
	}

	remainingAmount := budget.TotalAmount - budget.AllocatedAmount
	var utilizationRate float64
	if budget.AllocatedAmount > 0 {
		utilizationRate = budget.SpentAmount / budget.AllocatedAmount
	}

	projectNames := make(map[string]string)
	if bm.projectManager != nil {
		for _, alloc := range allocationCopies {
			if proj, err := bm.projectManager.GetProject(ctx, alloc.ProjectID); err == nil {
				projectNames[alloc.ProjectID] = proj.Name
			}
		}
	}

	return &types.BudgetSummary{
		Budget:          *budget,
		Allocations:     allocationCopies,
		ProjectNames:    projectNames,
		RemainingAmount: remainingAmount,
		UtilizationRate: utilizationRate,
	}, nil
}

// GetProjectFundingSummary generates a summary view of all funding for a project
func (bm *BudgetManager) GetProjectFundingSummary(ctx context.Context, projectID string) (*types.ProjectFundingSummary, error) {
	bm.mutex.RLock()
	defer bm.mutex.RUnlock()

	allocations := bm.projectAllocations[projectID]
	if len(allocations) == 0 {
		return nil, fmt.Errorf("no budget allocations found for project %q", projectID)
	}

	allocationCopies := make([]types.ProjectBudgetAllocation, len(allocations))
	budgetNames := make(map[string]string)
	var totalAllocated, totalSpent float64

	for i, alloc := range allocations {
		allocationCopies[i] = *alloc
		totalAllocated += alloc.AllocatedAmount
		totalSpent += alloc.SpentAmount

		if budget, exists := bm.budgets[alloc.BudgetID]; exists {
			budgetNames[alloc.BudgetID] = budget.Name
		}
	}

	var projectName string
	var defaultAllocationID *string
	if bm.projectManager != nil {
		if proj, err := bm.projectManager.GetProject(ctx, projectID); err == nil {
			projectName = proj.Name
			defaultAllocationID = proj.DefaultAllocationID
		}
	}

	return &types.ProjectFundingSummary{
		ProjectID:           projectID,
		ProjectName:         projectName,
		Allocations:         allocationCopies,
		BudgetNames:         budgetNames,
		TotalAllocated:      totalAllocated,
		TotalSpent:          totalSpent,
		DefaultAllocationID: defaultAllocationID,
	}, nil
}

// SpendingResult contains information about spending and potential backup activation
type SpendingResult struct {
	AllocationID        string
	AllocationExhausted bool
	BackupActivated     bool
	BackupAllocationID  string
	WarningMessage      string
}

// RecordSpending records spending against an allocation with backup funding support (v0.5.10+)
// Returns SpendingResult indicating if backup was activated
func (bm *BudgetManager) RecordSpending(ctx context.Context, allocationID string, amount float64) (*SpendingResult, error) {
	bm.mutex.Lock()
	defer bm.mutex.Unlock()

	result := &SpendingResult{
		AllocationID: allocationID,
	}

	allocation, exists := bm.allocations[allocationID]
	if !exists {
		return nil, fmt.Errorf("allocation %q not found", allocationID)
	}

	budget, exists := bm.budgets[allocation.BudgetID]
	if !exists {
		return nil, fmt.Errorf("budget %q not found", allocation.BudgetID)
	}

	// Update spending
	allocation.SpentAmount += amount
	allocation.UpdatedAt = time.Now()

	budget.SpentAmount += amount
	budget.UpdatedAt = time.Now()

	// Check if allocation is exhausted (v0.5.10+ backup funding)
	if allocation.SpentAmount >= allocation.AllocatedAmount {
		result.AllocationExhausted = true

		// Check for backup allocation (Issue #234)
		if allocation.BackupAllocationID != nil && *allocation.BackupAllocationID != "" {
			backupID := *allocation.BackupAllocationID

			// Verify backup allocation exists and is not exhausted
			backupAlloc, exists := bm.allocations[backupID]
			if exists && backupAlloc.SpentAmount < backupAlloc.AllocatedAmount {
				result.BackupActivated = true
				result.BackupAllocationID = backupID

				// Get budget names for messaging
				budgetName := budget.Name
				backupBudget := bm.budgets[backupAlloc.BudgetID]
				backupBudgetName := backupBudget.Name

				result.WarningMessage = fmt.Sprintf(
					"⚠️  Primary funding exhausted: %s ($%.2f spent / $%.2f allocated)\n"+
						"✅ Automatically switched to backup funding: %s ($%.2f available)\n"+
						"   Project will continue using backup allocation.",
					budgetName, allocation.SpentAmount, allocation.AllocatedAmount,
					backupBudgetName, backupAlloc.AllocatedAmount-backupAlloc.SpentAmount)
			} else {
				// Backup exists but is also exhausted or invalid
				result.WarningMessage = fmt.Sprintf(
					"❌ Primary funding exhausted: %s ($%.2f spent / $%.2f allocated)\n"+
						"❌ Backup funding unavailable or also exhausted\n"+
						"   No additional funds available for this project.",
					budget.Name, allocation.SpentAmount, allocation.AllocatedAmount)
			}
		} else {
			// No backup configured
			result.WarningMessage = fmt.Sprintf(
				"❌ Allocation exhausted: %s ($%.2f spent / $%.2f allocated)\n"+
					"   No backup funding configured for this allocation.",
				budget.Name, allocation.SpentAmount, allocation.AllocatedAmount)
		}
	}

	// Save changes
	if err := bm.saveAllocations(); err != nil {
		return nil, fmt.Errorf("failed to save allocation: %w", err)
	}

	if err := bm.saveBudgets(); err != nil {
		return nil, fmt.Errorf("failed to save budget: %w", err)
	}

	return result, nil
}

// CheckAllocationStatus checks if an allocation is exhausted or nearing exhaustion
func (bm *BudgetManager) CheckAllocationStatus(ctx context.Context, allocationID string) (exhausted bool, remaining float64, err error) {
	bm.mutex.RLock()
	defer bm.mutex.RUnlock()

	allocation, exists := bm.allocations[allocationID]
	if !exists {
		return false, 0, fmt.Errorf("allocation %q not found", allocationID)
	}

	remaining = allocation.AllocatedAmount - allocation.SpentAmount
	exhausted = remaining <= 0

	return exhausted, remaining, nil
}

// ActivateBackupFunding switches a project to its backup funding allocation
// This is called when the primary allocation is exhausted and backup exists
func (bm *BudgetManager) ActivateBackupFunding(ctx context.Context, projectID string, primaryAllocationID string, backupAllocationID string) error {
	bm.mutex.Lock()
	defer bm.mutex.Unlock()

	// Verify allocations exist
	primaryAlloc, exists := bm.allocations[primaryAllocationID]
	if !exists {
		return fmt.Errorf("primary allocation %q not found", primaryAllocationID)
	}

	backupAlloc, exists := bm.allocations[backupAllocationID]
	if !exists {
		return fmt.Errorf("backup allocation %q not found", backupAllocationID)
	}

	// Verify both belong to the same project
	if primaryAlloc.ProjectID != projectID || backupAlloc.ProjectID != projectID {
		return fmt.Errorf("allocations do not belong to project %q", projectID)
	}

	// Verify backup has available funds
	if backupAlloc.SpentAmount >= backupAlloc.AllocatedAmount {
		return fmt.Errorf("backup allocation %q is exhausted", backupAllocationID)
	}

	// Note: Updating project's default allocation is handled by the project manager
	// This method just validates the backup activation is possible

	return nil
}

// loadBudgets loads all budget records from the seam into the in-memory map.
func (bm *BudgetManager) loadBudgets() error {
	all, err := bm.budgetStore.List(context.Background(), budgetSeamScope)
	if err != nil {
		return fmt.Errorf("failed to load budgets: %w", err)
	}
	budgets := make(map[string]*types.Budget, len(all))
	for i := range all {
		b := all[i]
		budgets[b.ID] = &b
	}
	bm.budgets = budgets
	return nil
}

// saveBudgets reconciles the budget store to the in-memory map (Put all in-map, Delete dropped).
func (bm *BudgetManager) saveBudgets() error {
	ctx := context.Background()
	existing, err := bm.budgetStore.List(ctx, budgetSeamScope)
	if err != nil {
		return fmt.Errorf("failed to list budgets for save: %w", err)
	}
	for i := range existing {
		id := existing[i].ID
		if _, ok := bm.budgets[id]; !ok {
			if err := bm.budgetStore.Delete(ctx, budgetSeamScope, id); err != nil && !errorsIsNotFound(err) {
				return fmt.Errorf("failed to delete budget %q: %w", id, err)
			}
		}
	}
	for id, b := range bm.budgets {
		if b == nil {
			continue
		}
		if err := bm.budgetStore.Put(ctx, budgetSeamScope, id, *b); err != nil {
			return fmt.Errorf("failed to save budget %q: %w", id, err)
		}
	}
	return nil
}

// loadAllocations loads all allocation records from the seam into the in-memory map.
func (bm *BudgetManager) loadAllocations() error {
	all, err := bm.allocationStore.List(context.Background(), budgetSeamScope)
	if err != nil {
		return fmt.Errorf("failed to load allocations: %w", err)
	}
	allocations := make(map[string]*types.ProjectBudgetAllocation, len(all))
	for i := range all {
		a := all[i]
		allocations[a.ID] = &a
	}
	bm.allocations = allocations
	return nil
}

// saveAllocations reconciles the allocation store to the in-memory map.
func (bm *BudgetManager) saveAllocations() error {
	ctx := context.Background()
	existing, err := bm.allocationStore.List(ctx, budgetSeamScope)
	if err != nil {
		return fmt.Errorf("failed to list allocations for save: %w", err)
	}
	for i := range existing {
		id := existing[i].ID
		if _, ok := bm.allocations[id]; !ok {
			if err := bm.allocationStore.Delete(ctx, budgetSeamScope, id); err != nil && !errorsIsNotFound(err) {
				return fmt.Errorf("failed to delete allocation %q: %w", id, err)
			}
		}
	}
	for id, a := range bm.allocations {
		if a == nil {
			continue
		}
		if err := bm.allocationStore.Put(ctx, budgetSeamScope, id, *a); err != nil {
			return fmt.Errorf("failed to save allocation %q: %w", id, err)
		}
	}
	return nil
}

// rebuildIndexes rebuilds the lookup indexes for efficient queries
func (bm *BudgetManager) rebuildIndexes() {
	bm.projectAllocations = make(map[string][]*types.ProjectBudgetAllocation)
	bm.budgetAllocations = make(map[string][]*types.ProjectBudgetAllocation)
	bm.allocationReallocations = make(map[string][]*ReallocationRecord)

	for _, allocation := range bm.allocations {
		bm.projectAllocations[allocation.ProjectID] = append(bm.projectAllocations[allocation.ProjectID], allocation)
		bm.budgetAllocations[allocation.BudgetID] = append(bm.budgetAllocations[allocation.BudgetID], allocation)
	}

	for _, reallocation := range bm.reallocations {
		bm.allocationReallocations[reallocation.SourceAllocationID] = append(bm.allocationReallocations[reallocation.SourceAllocationID], reallocation)
		bm.allocationReallocations[reallocation.DestinationAllocationID] = append(bm.allocationReallocations[reallocation.DestinationAllocationID], reallocation)
	}
}

// loadReallocations loads all reallocation records from the seam into the in-memory map.
func (bm *BudgetManager) loadReallocations() error {
	all, err := bm.reallocationStore.List(context.Background(), budgetSeamScope)
	if err != nil {
		return fmt.Errorf("failed to load reallocations: %w", err)
	}
	reallocations := make(map[string]*ReallocationRecord, len(all))
	for i := range all {
		r := all[i]
		reallocations[r.ID] = &r
	}
	bm.reallocations = reallocations
	return nil
}

// saveReallocations reconciles the reallocation store to the in-memory map.
func (bm *BudgetManager) saveReallocations() error {
	ctx := context.Background()
	existing, err := bm.reallocationStore.List(ctx, budgetSeamScope)
	if err != nil {
		return fmt.Errorf("failed to list reallocations for save: %w", err)
	}
	for i := range existing {
		id := existing[i].ID
		if _, ok := bm.reallocations[id]; !ok {
			if err := bm.reallocationStore.Delete(ctx, budgetSeamScope, id); err != nil && !errorsIsNotFound(err) {
				return fmt.Errorf("failed to delete reallocation %q: %w", id, err)
			}
		}
	}
	for id, r := range bm.reallocations {
		if r == nil {
			continue
		}
		if err := bm.reallocationStore.Put(ctx, budgetSeamScope, id, *r); err != nil {
			return fmt.Errorf("failed to save reallocation %q: %w", id, err)
		}
	}
	return nil
}

// migrateLegacy folds the pre-seam flat JSON files (budgets.json, budget_allocations.json,
// budget_reallocations.json) into the seam stores, then retires each. Absent files → no-op.
func (bm *BudgetManager) migrateLegacy(stateDir string) error {
	ctx := context.Background()
	if err := migrateLegacyMap[types.Budget](ctx, bm.budgetStore, budgetSeamScope, filepath.Join(stateDir, "budgets.json")); err != nil {
		return err
	}
	if err := migrateLegacyMap[types.ProjectBudgetAllocation](ctx, bm.allocationStore, budgetSeamScope, filepath.Join(stateDir, "budget_allocations.json")); err != nil {
		return err
	}
	return migrateLegacyMap[ReallocationRecord](ctx, bm.reallocationStore, budgetSeamScope, filepath.Join(stateDir, "budget_reallocations.json"))
}

// ReallocateFunds moves funds between allocations atomically (v0.5.10+ Issue #99)
func (bm *BudgetManager) ReallocateFunds(ctx context.Context, req *ReallocateFundsRequest) (*ReallocationRecord, error) {
	bm.mutex.Lock()
	defer bm.mutex.Unlock()

	// Validate request
	if err := req.Validate(); err != nil {
		return nil, fmt.Errorf("invalid reallocation request: %w", err)
	}

	// Get source and destination allocations
	sourceAlloc, exists := bm.allocations[req.SourceAllocationID]
	if !exists {
		return nil, fmt.Errorf("source allocation %q not found", req.SourceAllocationID)
	}

	destAlloc, exists := bm.allocations[req.DestinationAllocationID]
	if !exists {
		return nil, fmt.Errorf("destination allocation %q not found", req.DestinationAllocationID)
	}

	// Get source and destination budgets
	sourceBudget, exists := bm.budgets[sourceAlloc.BudgetID]
	if !exists {
		return nil, fmt.Errorf("source budget %q not found", sourceAlloc.BudgetID)
	}

	destBudget, exists := bm.budgets[destAlloc.BudgetID]
	if !exists {
		return nil, fmt.Errorf("destination budget %q not found", destAlloc.BudgetID)
	}

	// Validate reallocation amount
	availableInSource := sourceAlloc.AllocatedAmount - sourceAlloc.SpentAmount
	if req.Amount > availableInSource {
		return nil, fmt.Errorf("insufficient unspent funds in source allocation: $%.2f available, $%.2f requested",
			availableInSource, req.Amount)
	}

	// For cross-budget reallocations, check destination budget has capacity
	if sourceAlloc.BudgetID != destAlloc.BudgetID {
		availableInDestBudget := destBudget.TotalAmount - destBudget.AllocatedAmount
		if req.Amount > availableInDestBudget {
			return nil, fmt.Errorf("insufficient capacity in destination budget: $%.2f available, $%.2f requested",
				availableInDestBudget, req.Amount)
		}
	}

	// Create reallocation record
	record := &ReallocationRecord{
		ID:                      uuid.New().String(),
		SourceAllocationID:      req.SourceAllocationID,
		DestinationAllocationID: req.DestinationAllocationID,
		SourceBudgetID:          sourceAlloc.BudgetID,
		DestinationBudgetID:     destAlloc.BudgetID,
		Amount:                  req.Amount,
		Reason:                  req.Reason,
		PerformedBy:             req.PerformedBy,
		Timestamp:               time.Now(),
	}

	// Perform the reallocation atomically
	sourceAlloc.AllocatedAmount -= req.Amount
	destAlloc.AllocatedAmount += req.Amount

	// Update budget allocated amounts if cross-budget
	if sourceAlloc.BudgetID != destAlloc.BudgetID {
		sourceBudget.AllocatedAmount -= req.Amount
		destBudget.AllocatedAmount += req.Amount
		sourceBudget.UpdatedAt = time.Now()
		destBudget.UpdatedAt = time.Now()
	}

	sourceAlloc.UpdatedAt = time.Now()
	destAlloc.UpdatedAt = time.Now()

	// Store reallocation record
	bm.reallocations[record.ID] = record
	bm.rebuildIndexes()

	// Save all changes; roll back on any failure
	isCrossBudget := sourceAlloc.BudgetID != destAlloc.BudgetID
	rollback := func() {
		sourceAlloc.AllocatedAmount += req.Amount
		destAlloc.AllocatedAmount -= req.Amount
		if isCrossBudget {
			sourceBudget.AllocatedAmount += req.Amount
			destBudget.AllocatedAmount -= req.Amount
		}
		delete(bm.reallocations, record.ID)
		bm.rebuildIndexes()
	}

	if err := bm.saveAllocations(); err != nil {
		rollback()
		return nil, fmt.Errorf("failed to save allocations: %w", err)
	}
	if err := bm.saveBudgets(); err != nil {
		rollback()
		return nil, fmt.Errorf("failed to save budgets: %w", err)
	}
	if err := bm.saveReallocations(); err != nil {
		rollback()
		return nil, fmt.Errorf("failed to save reallocation record: %w", err)
	}

	recordCopy := *record
	return &recordCopy, nil
}

// GetReallocationHistory retrieves reallocation history for a specific allocation (v0.5.10+ Issue #99)
func (bm *BudgetManager) GetReallocationHistory(ctx context.Context, allocationID string) ([]*ReallocationRecord, error) {
	bm.mutex.RLock()
	defer bm.mutex.RUnlock()

	// Verify allocation exists
	if _, exists := bm.allocations[allocationID]; !exists {
		return nil, fmt.Errorf("allocation %q not found", allocationID)
	}

	records := bm.allocationReallocations[allocationID]
	results := make([]*ReallocationRecord, len(records))
	for i, record := range records {
		recordCopy := *record
		results[i] = &recordCopy
	}

	return results, nil
}

// GetBudgetReallocationHistory retrieves all reallocations for a budget (v0.5.10+ Issue #99)
func (bm *BudgetManager) GetBudgetReallocationHistory(ctx context.Context, budgetID string) ([]*ReallocationRecord, error) {
	bm.mutex.RLock()
	defer bm.mutex.RUnlock()

	// Verify budget exists
	if _, exists := bm.budgets[budgetID]; !exists {
		return nil, fmt.Errorf("budget %q not found", budgetID)
	}

	// Collect all reallocations involving this budget
	var results []*ReallocationRecord
	for _, record := range bm.reallocations {
		if record.SourceBudgetID == budgetID || record.DestinationBudgetID == budgetID {
			recordCopy := *record
			results = append(results, &recordCopy)
		}
	}

	return results, nil
}

// ============================================================================
// Multi-Project Cost Rollup and Reporting (v0.5.10+ Issue #100)
// ============================================================================

// GenerateBudgetRollupReport generates a comprehensive rollup report across all budgets (v0.5.10+ Issue #100)
func (bm *BudgetManager) GenerateBudgetRollupReport(ctx context.Context) (*BudgetRollupReport, error) {
	bm.mutex.RLock()
	defer bm.mutex.RUnlock()

	report := &BudgetRollupReport{
		ReportID:        uuid.New().String(),
		GeneratedAt:     time.Now(),
		BudgetSummaries: make([]BudgetSummaryReport, 0),
	}

	projectsMap := make(map[string]bool)

	// Generate summary for each budget
	for _, budget := range bm.budgets {
		summary := bm.generateBudgetSummary(budget)
		report.BudgetSummaries = append(report.BudgetSummaries, summary)

		// Track unique projects
		for _, proj := range summary.Projects {
			projectsMap[proj.ProjectID] = true
		}

		// Aggregate totals
		report.TotalAllocated += budget.AllocatedAmount
		report.TotalSpent += budget.SpentAmount
	}

	report.TotalBudgets = len(bm.budgets)
	report.TotalRemaining = report.TotalAllocated - report.TotalSpent
	if report.TotalAllocated > 0 {
		report.OverallUtilization = report.TotalSpent / report.TotalAllocated
	}
	report.ProjectCount = len(projectsMap)

	return report, nil
}

// generateBudgetSummary creates a detailed summary for a single budget
func (bm *BudgetManager) generateBudgetSummary(budget *types.Budget) BudgetSummaryReport {
	summary := BudgetSummaryReport{
		BudgetID:        budget.ID,
		BudgetName:      budget.Name,
		Period:          string(budget.Period),
		TotalAmount:     budget.TotalAmount,
		AllocatedAmount: budget.AllocatedAmount,
		SpentAmount:     budget.SpentAmount,
		RemainingAmount: budget.TotalAmount - budget.AllocatedAmount,
		Projects:        make([]ProjectCostSummary, 0),
	}

	if budget.AllocatedAmount > 0 {
		summary.Utilization = budget.SpentAmount / budget.AllocatedAmount
	}

	// Get allocations for this budget
	allocations := bm.budgetAllocations[budget.ID]
	summary.AllocationCount = len(allocations)

	// Group allocations by project
	projectAllocations := make(map[string][]*types.ProjectBudgetAllocation)
	for _, alloc := range allocations {
		projectAllocations[alloc.ProjectID] = append(projectAllocations[alloc.ProjectID], alloc)
	}

	summary.ProjectCount = len(projectAllocations)

	// Generate project summaries
	for projectID, projAllocs := range projectAllocations {
		projectSummary := bm.generateProjectCostSummary(projectID, projAllocs)
		summary.Projects = append(summary.Projects, projectSummary)
	}

	return summary
}

// generateProjectCostSummary creates a cost summary for a project
func (bm *BudgetManager) generateProjectCostSummary(projectID string, allocations []*types.ProjectBudgetAllocation) ProjectCostSummary {
	summary := ProjectCostSummary{
		ProjectID:      projectID,
		ProjectName:    projectID, // Will be populated by caller if project manager available
		FundingSources: make([]AllocationSummary, 0),
	}

	for _, alloc := range allocations {
		allocSummary := AllocationSummary{
			AllocationID:    alloc.ID,
			BudgetID:        alloc.BudgetID,
			BudgetName:      bm.budgets[alloc.BudgetID].Name,
			AllocatedAmount: alloc.AllocatedAmount,
			SpentAmount:     alloc.SpentAmount,
			RemainingAmount: alloc.AllocatedAmount - alloc.SpentAmount,
			HasBackup:       alloc.BackupAllocationID != nil && *alloc.BackupAllocationID != "",
		}

		if alloc.AllocatedAmount > 0 {
			allocSummary.Utilization = alloc.SpentAmount / alloc.AllocatedAmount
		}

		summary.FundingSources = append(summary.FundingSources, allocSummary)
		summary.TotalAllocated += alloc.AllocatedAmount
		summary.TotalSpent += alloc.SpentAmount
	}

	summary.TotalRemaining = summary.TotalAllocated - summary.TotalSpent
	if summary.TotalAllocated > 0 {
		summary.Utilization = summary.TotalSpent / summary.TotalAllocated
	}

	return summary
}

// GetBudgetSummaryReport generates a detailed summary for a specific budget (v0.5.10+ Issue #100)
func (bm *BudgetManager) GetBudgetSummaryReport(ctx context.Context, budgetID string) (*BudgetSummaryReport, error) {
	bm.mutex.RLock()
	defer bm.mutex.RUnlock()

	budget, exists := bm.budgets[budgetID]
	if !exists {
		return nil, fmt.Errorf("budget %q not found", budgetID)
	}

	summary := bm.generateBudgetSummary(budget)
	return &summary, nil
}

// GetProjectCostRollup generates cost rollup for specific projects (v0.5.10+ Issue #100)
func (bm *BudgetManager) GetProjectCostRollup(ctx context.Context, projectIDs []string) ([]ProjectCostSummary, error) {
	bm.mutex.RLock()
	defer bm.mutex.RUnlock()

	summaries := make([]ProjectCostSummary, 0, len(projectIDs))

	for _, projectID := range projectIDs {
		allocations := bm.projectAllocations[projectID]
		if len(allocations) == 0 {
			continue // Skip projects with no allocations
		}

		summary := bm.generateProjectCostSummary(projectID, allocations)
		summaries = append(summaries, summary)
	}

	return summaries, nil
}

// Close cleanly shuts down the budget manager
func (bm *BudgetManager) Close() error {
	bm.mutex.Lock()
	defer bm.mutex.Unlock()

	// Nothing to clean up currently, but method exists for future needs
	return nil
}
