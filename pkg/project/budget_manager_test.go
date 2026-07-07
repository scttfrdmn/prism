package project

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNewBudgetManager tests budget manager creation
func TestNewBudgetManager(t *testing.T) {
	// Create temporary directory for testing
	tempDir, err := os.MkdirTemp("", "prism-budget-test-*")
	require.NoError(t, err)
	defer func() { _ = os.RemoveAll(tempDir) }()

	// Mock home directory
	originalHome := os.Getenv("HOME")
	defer func() { _ = os.Setenv("HOME", originalHome) }()
	_ = os.Setenv("HOME", tempDir)

	manager, err := NewBudgetManager()
	assert.NoError(t, err)
	assert.NotNil(t, manager)
	assert.NotNil(t, manager.budgets)
	assert.NotNil(t, manager.allocations)
	assert.NotNil(t, manager.reallocations)
}

// TestBudgetManager_CreateBudget tests budget creation
func TestBudgetManager_CreateBudget(t *testing.T) {
	manager := setupTestBudgetManager(t)
	ctx := context.Background()

	tests := []struct {
		name    string
		req     *CreateBudgetRequest
		wantErr bool
		errMsg  string
	}{
		{
			name: "create valid budget",
			req: &CreateBudgetRequest{
				Name:           "NSF Grant 2024",
				Description:    "National Science Foundation grant",
				TotalAmount:    100000.0,
				Period:         "yearly",
				StartDate:      time.Now(),
				AlertThreshold: 0.8,
				CreatedBy:      "test-user",
			},
			wantErr: false,
		},
		{
			name: "create budget with tags",
			req: &CreateBudgetRequest{
				Name:           "DOE Grant",
				Description:    "Department of Energy grant",
				TotalAmount:    50000.0,
				Period:         "monthly",
				StartDate:      time.Now(),
				AlertThreshold: 0.9,
				CreatedBy:      "test-user",
				Tags: map[string]string{
					"agency":     "DOE",
					"grant_type": "research",
				},
			},
			wantErr: false,
		},
		{
			name: "create budget with end date",
			req: &CreateBudgetRequest{
				Name:           "Temporary Budget",
				Description:    "Time-limited budget",
				TotalAmount:    25000.0,
				Period:         "project",
				StartDate:      time.Now(),
				EndDate:        timePtr(time.Now().Add(365 * 24 * time.Hour)),
				AlertThreshold: 0.75,
				CreatedBy:      "test-user",
			},
			wantErr: false,
		},
		{
			name: "create budget with empty name",
			req: &CreateBudgetRequest{
				Name:           "",
				Description:    "Invalid budget",
				TotalAmount:    10000.0,
				Period:         "monthly",
				StartDate:      time.Now(),
				AlertThreshold: 0.8,
				CreatedBy:      "test-user",
			},
			wantErr: true,
			errMsg:  "name is required",
		},
		{
			name: "create budget with zero amount",
			req: &CreateBudgetRequest{
				Name:           "Zero Budget",
				Description:    "Invalid budget",
				TotalAmount:    0.0,
				Period:         "monthly",
				StartDate:      time.Now(),
				AlertThreshold: 0.8,
				CreatedBy:      "test-user",
			},
			wantErr: true,
			errMsg:  "total amount must be greater than 0",
		},
		{
			name: "create budget with negative amount",
			req: &CreateBudgetRequest{
				Name:           "Negative Budget",
				Description:    "Invalid budget",
				TotalAmount:    -1000.0,
				Period:         "monthly",
				StartDate:      time.Now(),
				AlertThreshold: 0.8,
				CreatedBy:      "test-user",
			},
			wantErr: true,
			errMsg:  "total amount must be greater than 0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			budget, err := manager.CreateBudget(ctx, tt.req)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, budget)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, budget)
				assert.NotEmpty(t, budget.ID)
				assert.Equal(t, tt.req.Name, budget.Name)
				assert.Equal(t, tt.req.TotalAmount, budget.TotalAmount)
				assert.Equal(t, 0.0, budget.AllocatedAmount)
				assert.Equal(t, 0.0, budget.SpentAmount)
				assert.WithinDuration(t, time.Now(), budget.CreatedAt, time.Second)
			}
		})
	}
}

// TestBudgetManager_CreateBudget_DuplicateName tests duplicate budget names
func TestBudgetManager_CreateBudget_DuplicateName(t *testing.T) {
	manager := setupTestBudgetManager(t)
	ctx := context.Background()

	// Create first budget
	req := &CreateBudgetRequest{
		Name:           "Test Budget",
		Description:    "First budget",
		TotalAmount:    10000.0,
		Period:         "monthly",
		StartDate:      time.Now(),
		AlertThreshold: 0.8,
		CreatedBy:      "user1",
	}

	budget1, err := manager.CreateBudget(ctx, req)
	require.NoError(t, err)
	require.NotNil(t, budget1)

	// Try to create second budget with same name
	req2 := &CreateBudgetRequest{
		Name:           "Test Budget", // Same name
		Description:    "Second budget",
		TotalAmount:    20000.0,
		Period:         "yearly",
		StartDate:      time.Now(),
		AlertThreshold: 0.9,
		CreatedBy:      "user2",
	}

	budget2, err := manager.CreateBudget(ctx, req2)
	assert.Error(t, err)
	assert.Nil(t, budget2)
	assert.Contains(t, err.Error(), "already exists")
}

// TestBudgetManager_GetBudget tests budget retrieval
func TestBudgetManager_GetBudget(t *testing.T) {
	manager := setupTestBudgetManager(t)
	ctx := context.Background()

	// Create test budget
	req := &CreateBudgetRequest{
		Name:           "Test Budget",
		Description:    "Budget for retrieval testing",
		TotalAmount:    50000.0,
		Period:         "yearly",
		StartDate:      time.Now(),
		AlertThreshold: 0.85,
		CreatedBy:      "test-user",
		Tags: map[string]string{
			"type": "test",
		},
	}

	createdBudget, err := manager.CreateBudget(ctx, req)
	require.NoError(t, err)

	tests := []struct {
		name     string
		budgetID string
		wantErr  bool
		wantName string
	}{
		{
			name:     "get existing budget",
			budgetID: createdBudget.ID,
			wantErr:  false,
			wantName: "Test Budget",
		},
		{
			name:     "get non-existent budget",
			budgetID: uuid.New().String(),
			wantErr:  true,
		},
		{
			name:     "get budget with empty ID",
			budgetID: "",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			budget, err := manager.GetBudget(ctx, tt.budgetID)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, budget)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, budget)
				assert.Equal(t, tt.wantName, budget.Name)
				assert.Equal(t, tt.budgetID, budget.ID)

				// Verify it's a copy (changes don't affect original)
				budget.Name = "Modified"
				retrievedAgain, err := manager.GetBudget(ctx, tt.budgetID)
				require.NoError(t, err)
				assert.Equal(t, tt.wantName, retrievedAgain.Name)
			}
		})
	}
}

// TestBudgetManager_GetBudgetByName tests budget retrieval by name
func TestBudgetManager_GetBudgetByName(t *testing.T) {
	manager := setupTestBudgetManager(t)
	ctx := context.Background()

	// Create test budgets
	budget1, err := manager.CreateBudget(ctx, &CreateBudgetRequest{
		Name:           "Budget Alpha",
		Description:    "First test budget",
		TotalAmount:    10000.0,
		Period:         "monthly",
		StartDate:      time.Now(),
		AlertThreshold: 0.8,
		CreatedBy:      "user1",
	})
	require.NoError(t, err)

	budget2, err := manager.CreateBudget(ctx, &CreateBudgetRequest{
		Name:           "Budget Beta",
		Description:    "Second test budget",
		TotalAmount:    20000.0,
		Period:         "yearly",
		StartDate:      time.Now(),
		AlertThreshold: 0.9,
		CreatedBy:      "user2",
	})
	require.NoError(t, err)

	tests := []struct {
		name       string
		budgetName string
		wantErr    bool
		wantID     string
	}{
		{
			name:       "get existing budget by name",
			budgetName: "Budget Alpha",
			wantErr:    false,
			wantID:     budget1.ID,
		},
		{
			name:       "get second budget by name",
			budgetName: "Budget Beta",
			wantErr:    false,
			wantID:     budget2.ID,
		},
		{
			name:       "get non-existent budget by name",
			budgetName: "Non-existent Budget",
			wantErr:    true,
		},
		{
			name:       "get budget with empty name",
			budgetName: "",
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			budget, err := manager.GetBudgetByName(ctx, tt.budgetName)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, budget)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, budget)
				assert.Equal(t, tt.budgetName, budget.Name)
				assert.Equal(t, tt.wantID, budget.ID)
			}
		})
	}
}

// TestBudgetManager_ListBudgets tests listing budgets
func TestBudgetManager_ListBudgets(t *testing.T) {
	manager := setupTestBudgetManager(t)
	ctx := context.Background()

	// Create test budgets
	_, err := manager.CreateBudget(ctx, &CreateBudgetRequest{
		Name:           "Budget 1",
		Description:    "First budget",
		TotalAmount:    10000.0,
		Period:         "monthly",
		StartDate:      time.Now(),
		AlertThreshold: 0.8,
		CreatedBy:      "user1",
		Tags: map[string]string{
			"department": "research",
		},
	})
	require.NoError(t, err)

	_, err = manager.CreateBudget(ctx, &CreateBudgetRequest{
		Name:           "Budget 2",
		Description:    "Second budget",
		TotalAmount:    20000.0,
		Period:         "yearly",
		StartDate:      time.Now(),
		AlertThreshold: 0.9,
		CreatedBy:      "user2",
		Tags: map[string]string{
			"department": "engineering",
		},
	})
	require.NoError(t, err)

	// List all budgets
	budgets, err := manager.ListBudgets(ctx)
	assert.NoError(t, err)
	assert.Len(t, budgets, 2)

	// Verify budgets are returned
	names := []string{budgets[0].Name, budgets[1].Name}
	assert.Contains(t, names, "Budget 1")
	assert.Contains(t, names, "Budget 2")
}

// TestBudgetManager_UpdateBudget tests budget updates
func TestBudgetManager_UpdateBudget(t *testing.T) {
	manager := setupTestBudgetManager(t)
	ctx := context.Background()

	// Create test budget
	budget, err := manager.CreateBudget(ctx, &CreateBudgetRequest{
		Name:           "Original Budget",
		Description:    "Original description",
		TotalAmount:    10000.0,
		Period:         "monthly",
		StartDate:      time.Now(),
		AlertThreshold: 0.8,
		CreatedBy:      "test-user",
	})
	require.NoError(t, err)

	tests := []struct {
		name      string
		budgetID  string
		updateReq *UpdateBudgetRequest
		wantErr   bool
		errMsg    string
	}{
		{
			name:     "update budget name",
			budgetID: budget.ID,
			updateReq: &UpdateBudgetRequest{
				Name: stringPtr("Updated Budget"),
			},
			wantErr: false,
		},
		{
			name:     "update budget amount",
			budgetID: budget.ID,
			updateReq: &UpdateBudgetRequest{
				TotalAmount: floatPtr(20000.0),
			},
			wantErr: false,
		},
		{
			name:     "update non-existent budget",
			budgetID: uuid.New().String(),
			updateReq: &UpdateBudgetRequest{
				Name: stringPtr("Should Fail"),
			},
			wantErr: true,
			errMsg:  "not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			updatedBudget, err := manager.UpdateBudget(ctx, tt.budgetID, tt.updateReq)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, updatedBudget)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, updatedBudget)

				if tt.updateReq.Name != nil {
					assert.Equal(t, *tt.updateReq.Name, updatedBudget.Name)
				}
				if tt.updateReq.TotalAmount != nil {
					assert.Equal(t, *tt.updateReq.TotalAmount, updatedBudget.TotalAmount)
				}
			}
		})
	}
}

// TestBudgetManager_DeleteBudget tests budget deletion
func TestBudgetManager_DeleteBudget(t *testing.T) {
	manager := setupTestBudgetManager(t)
	ctx := context.Background()

	// Create test budgets
	budget1, err := manager.CreateBudget(ctx, &CreateBudgetRequest{
		Name:           "Budget to Delete",
		Description:    "This will be deleted",
		TotalAmount:    10000.0,
		Period:         "monthly",
		StartDate:      time.Now(),
		AlertThreshold: 0.8,
		CreatedBy:      "test-user",
	})
	require.NoError(t, err)

	budget2, err := manager.CreateBudget(ctx, &CreateBudgetRequest{
		Name:           "Budget to Keep",
		Description:    "This will remain",
		TotalAmount:    20000.0,
		Period:         "yearly",
		StartDate:      time.Now(),
		AlertThreshold: 0.9,
		CreatedBy:      "test-user",
	})
	require.NoError(t, err)

	// Delete first budget
	err = manager.DeleteBudget(ctx, budget1.ID)
	assert.NoError(t, err)

	// Verify it's deleted
	_, err = manager.GetBudget(ctx, budget1.ID)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")

	// Verify second budget still exists
	retrieved, err := manager.GetBudget(ctx, budget2.ID)
	assert.NoError(t, err)
	assert.NotNil(t, retrieved)
	assert.Equal(t, "Budget to Keep", retrieved.Name)

	// Test deleting non-existent budget
	err = manager.DeleteBudget(ctx, uuid.New().String())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

// Helper functions
func setupTestBudgetManager(t *testing.T) *BudgetManager {
	t.Helper()

	// Create temporary directory for testing
	tempDir, err := os.MkdirTemp("", "prism-budget-test-*")
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = os.RemoveAll(tempDir)
	})

	// Mock home directory
	originalHome := os.Getenv("HOME")
	t.Cleanup(func() {
		_ = os.Setenv("HOME", originalHome)
	})
	_ = os.Setenv("HOME", tempDir)

	manager, err := NewBudgetManager()
	require.NoError(t, err)
	require.NotNil(t, manager)

	return manager
}

// TestBudgetManager_CreateAllocation tests creating allocations
func TestBudgetManager_CreateAllocation(t *testing.T) {
	manager := setupTestBudgetManager(t)
	ctx := context.Background()

	// Create test budget
	budget, err := manager.CreateBudget(ctx, &CreateBudgetRequest{
		Name:           "Test Budget",
		Description:    "Budget for allocation testing",
		TotalAmount:    100000.0,
		Period:         "yearly",
		StartDate:      time.Now(),
		AlertThreshold: 0.8,
		CreatedBy:      "test-user",
	})
	require.NoError(t, err)

	projectID := uuid.New().String()

	tests := []struct {
		name    string
		req     *CreateAllocationRequest
		wantErr bool
		errMsg  string
	}{
		{
			name: "create valid allocation",
			req: &CreateAllocationRequest{
				BudgetID:        budget.ID,
				ProjectID:       projectID,
				AllocatedAmount: 25000.0,
				AlertThreshold:  floatPtr(0.85),
				Notes:           "Q1 allocation",
				AllocatedBy:     "test-user",
			},
			wantErr: false,
		},
		{
			name: "create allocation with zero amount",
			req: &CreateAllocationRequest{
				BudgetID:        budget.ID,
				ProjectID:       uuid.New().String(),
				AllocatedAmount: 0.0,
				AllocatedBy:     "test-user",
			},
			wantErr: true,
			errMsg:  "allocated amount must be greater than 0",
		},
		{
			name: "create allocation for non-existent budget",
			req: &CreateAllocationRequest{
				BudgetID:        uuid.New().String(),
				ProjectID:       uuid.New().String(),
				AllocatedAmount: 10000.0,
				AllocatedBy:     "test-user",
			},
			wantErr: true,
			errMsg:  "budget",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			allocation, err := manager.CreateAllocation(ctx, tt.req)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, allocation)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, allocation)
				assert.NotEmpty(t, allocation.ID)
				assert.Equal(t, tt.req.BudgetID, allocation.BudgetID)
				assert.Equal(t, tt.req.ProjectID, allocation.ProjectID)
				assert.Equal(t, tt.req.AllocatedAmount, allocation.AllocatedAmount)
				assert.Equal(t, 0.0, allocation.SpentAmount)
				assert.WithinDuration(t, time.Now(), allocation.AllocatedAt, time.Second)
			}
		})
	}
}

// TestBudgetManager_GetAllocation tests allocation retrieval
func TestBudgetManager_GetAllocation(t *testing.T) {
	manager := setupTestBudgetManager(t)
	ctx := context.Background()

	// Create test budget and allocation
	budget, err := manager.CreateBudget(ctx, &CreateBudgetRequest{
		Name:           "Test Budget",
		Description:    "Budget for testing",
		TotalAmount:    50000.0,
		Period:         "yearly",
		StartDate:      time.Now(),
		AlertThreshold: 0.8,
		CreatedBy:      "test-user",
	})
	require.NoError(t, err)

	allocation, err := manager.CreateAllocation(ctx, &CreateAllocationRequest{
		BudgetID:        budget.ID,
		ProjectID:       uuid.New().String(),
		AllocatedAmount: 10000.0,
		Notes:           "Test allocation",
		AllocatedBy:     "test-user",
	})
	require.NoError(t, err)

	// Test retrieval
	retrieved, err := manager.GetAllocation(ctx, allocation.ID)
	assert.NoError(t, err)
	assert.NotNil(t, retrieved)
	assert.Equal(t, allocation.ID, retrieved.ID)
	assert.Equal(t, allocation.AllocatedAmount, retrieved.AllocatedAmount)

	// Test non-existent allocation
	_, err = manager.GetAllocation(ctx, uuid.New().String())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

// TestBudgetManager_GetProjectAllocations tests getting all allocations for a project
func TestBudgetManager_GetProjectAllocations(t *testing.T) {
	manager := setupTestBudgetManager(t)
	ctx := context.Background()

	// Create test budgets
	budget1, err := manager.CreateBudget(ctx, &CreateBudgetRequest{
		Name:           "Budget 1",
		Description:    "First budget",
		TotalAmount:    50000.0,
		Period:         "yearly",
		StartDate:      time.Now(),
		AlertThreshold: 0.8,
		CreatedBy:      "test-user",
	})
	require.NoError(t, err)

	budget2, err := manager.CreateBudget(ctx, &CreateBudgetRequest{
		Name:           "Budget 2",
		Description:    "Second budget",
		TotalAmount:    30000.0,
		Period:         "yearly",
		StartDate:      time.Now(),
		AlertThreshold: 0.9,
		CreatedBy:      "test-user",
	})
	require.NoError(t, err)

	projectID := uuid.New().String()

	// Create allocations for the same project from different budgets
	_, err = manager.CreateAllocation(ctx, &CreateAllocationRequest{
		BudgetID:        budget1.ID,
		ProjectID:       projectID,
		AllocatedAmount: 15000.0,
		AllocatedBy:     "test-user",
	})
	require.NoError(t, err)

	_, err = manager.CreateAllocation(ctx, &CreateAllocationRequest{
		BudgetID:        budget2.ID,
		ProjectID:       projectID,
		AllocatedAmount: 10000.0,
		AllocatedBy:     "test-user",
	})
	require.NoError(t, err)

	// Get all allocations for the project
	allocations, err := manager.GetProjectAllocations(ctx, projectID)
	assert.NoError(t, err)
	assert.Len(t, allocations, 2)

	// Verify allocations
	totalAllocated := allocations[0].AllocatedAmount + allocations[1].AllocatedAmount
	assert.Equal(t, 25000.0, totalAllocated)
}

// TestBudgetManager_GetBudgetAllocations tests getting all allocations for a budget
func TestBudgetManager_GetBudgetAllocations(t *testing.T) {
	manager := setupTestBudgetManager(t)
	ctx := context.Background()

	// Create test budget
	budget, err := manager.CreateBudget(ctx, &CreateBudgetRequest{
		Name:           "Test Budget",
		Description:    "Budget for allocation testing",
		TotalAmount:    100000.0,
		Period:         "yearly",
		StartDate:      time.Now(),
		AlertThreshold: 0.8,
		CreatedBy:      "test-user",
	})
	require.NoError(t, err)

	// Create allocations for different projects
	project1 := uuid.New().String()
	project2 := uuid.New().String()

	_, err = manager.CreateAllocation(ctx, &CreateAllocationRequest{
		BudgetID:        budget.ID,
		ProjectID:       project1,
		AllocatedAmount: 20000.0,
		AllocatedBy:     "test-user",
	})
	require.NoError(t, err)

	_, err = manager.CreateAllocation(ctx, &CreateAllocationRequest{
		BudgetID:        budget.ID,
		ProjectID:       project2,
		AllocatedAmount: 30000.0,
		AllocatedBy:     "test-user",
	})
	require.NoError(t, err)

	// Get all allocations for the budget
	allocations, err := manager.GetBudgetAllocations(ctx, budget.ID)
	assert.NoError(t, err)
	assert.Len(t, allocations, 2)

	// Verify total allocated
	totalAllocated := allocations[0].AllocatedAmount + allocations[1].AllocatedAmount
	assert.Equal(t, 50000.0, totalAllocated)
}

// TestBudgetManager_UpdateAllocation tests updating allocations
func TestBudgetManager_UpdateAllocation(t *testing.T) {
	manager := setupTestBudgetManager(t)
	ctx := context.Background()

	// Create test budget and allocation
	budget, err := manager.CreateBudget(ctx, &CreateBudgetRequest{
		Name:           "Test Budget",
		Description:    "Budget for testing",
		TotalAmount:    50000.0,
		Period:         "yearly",
		StartDate:      time.Now(),
		AlertThreshold: 0.8,
		CreatedBy:      "test-user",
	})
	require.NoError(t, err)

	allocation, err := manager.CreateAllocation(ctx, &CreateAllocationRequest{
		BudgetID:        budget.ID,
		ProjectID:       uuid.New().String(),
		AllocatedAmount: 10000.0,
		AlertThreshold:  floatPtr(0.8),
		Notes:           "Original notes",
		AllocatedBy:     "test-user",
	})
	require.NoError(t, err)

	// Update allocation
	updateReq := &UpdateAllocationRequest{
		AllocatedAmount: floatPtr(15000.0),
		AlertThreshold:  floatPtr(0.85),
		Notes:           stringPtr("Updated notes"),
	}

	updatedAllocation, err := manager.UpdateAllocation(ctx, allocation.ID, updateReq)
	assert.NoError(t, err)
	assert.NotNil(t, updatedAllocation)
	assert.Equal(t, 15000.0, updatedAllocation.AllocatedAmount)
	assert.Equal(t, 0.85, *updatedAllocation.AlertThreshold)
	assert.Equal(t, "Updated notes", updatedAllocation.Notes)

	// Test updating non-existent allocation
	_, err = manager.UpdateAllocation(ctx, uuid.New().String(), updateReq)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

// TestBudgetManager_DeleteAllocation tests allocation deletion
func TestBudgetManager_DeleteAllocation(t *testing.T) {
	manager := setupTestBudgetManager(t)
	ctx := context.Background()

	// Create test budget and allocation
	budget, err := manager.CreateBudget(ctx, &CreateBudgetRequest{
		Name:           "Test Budget",
		Description:    "Budget for testing",
		TotalAmount:    50000.0,
		Period:         "yearly",
		StartDate:      time.Now(),
		AlertThreshold: 0.8,
		CreatedBy:      "test-user",
	})
	require.NoError(t, err)

	allocation, err := manager.CreateAllocation(ctx, &CreateAllocationRequest{
		BudgetID:        budget.ID,
		ProjectID:       uuid.New().String(),
		AllocatedAmount: 10000.0,
		AllocatedBy:     "test-user",
	})
	require.NoError(t, err)

	// Delete allocation
	err = manager.DeleteAllocation(ctx, allocation.ID)
	assert.NoError(t, err)

	// Verify it's deleted
	_, err = manager.GetAllocation(ctx, allocation.ID)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")

	// Test deleting non-existent allocation
	err = manager.DeleteAllocation(ctx, uuid.New().String())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

// TestBudgetManager_GetBudgetSummary tests budget summary retrieval
func TestBudgetManager_GetBudgetSummary(t *testing.T) {
	manager := setupTestBudgetManager(t)
	ctx := context.Background()

	// Create test budget
	budget, err := manager.CreateBudget(ctx, &CreateBudgetRequest{
		Name:           "Test Budget",
		Description:    "Budget for summary testing",
		TotalAmount:    100000.0,
		Period:         "yearly",
		StartDate:      time.Now(),
		AlertThreshold: 0.8,
		CreatedBy:      "test-user",
	})
	require.NoError(t, err)

	// Create allocations
	_, err = manager.CreateAllocation(ctx, &CreateAllocationRequest{
		BudgetID:        budget.ID,
		ProjectID:       uuid.New().String(),
		AllocatedAmount: 30000.0,
		AllocatedBy:     "test-user",
	})
	require.NoError(t, err)

	_, err = manager.CreateAllocation(ctx, &CreateAllocationRequest{
		BudgetID:        budget.ID,
		ProjectID:       uuid.New().String(),
		AllocatedAmount: 20000.0,
		AllocatedBy:     "test-user",
	})
	require.NoError(t, err)

	// Get summary
	summary, err := manager.GetBudgetSummary(ctx, budget.ID)
	assert.NoError(t, err)
	assert.NotNil(t, summary)
	assert.Equal(t, budget.ID, summary.Budget.ID)
	assert.Equal(t, 100000.0, summary.Budget.TotalAmount)
	assert.Equal(t, 50000.0, summary.Budget.AllocatedAmount)
	assert.Equal(t, 50000.0, summary.RemainingAmount)

	// Test non-existent budget
	_, err = manager.GetBudgetSummary(ctx, uuid.New().String())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

// TestBudgetManager_ConcurrentOperations tests thread safety
func TestBudgetManager_ConcurrentOperations(t *testing.T) {
	manager := setupTestBudgetManager(t)
	ctx := context.Background()

	// Create test budget
	budget, err := manager.CreateBudget(ctx, &CreateBudgetRequest{
		Name:           "Concurrent Test Budget",
		Description:    "Budget for concurrency testing",
		TotalAmount:    1000000.0,
		Period:         "yearly",
		StartDate:      time.Now(),
		AlertThreshold: 0.8,
		CreatedBy:      "test-user",
	})
	require.NoError(t, err)

	const numGoroutines = 10
	done := make(chan bool, numGoroutines)
	errors := make(chan error, numGoroutines)

	// Concurrently create allocations
	for i := 0; i < numGoroutines; i++ {
		go func(index int) {
			defer func() { done <- true }()

			projectID := uuid.New().String()
			req := &CreateAllocationRequest{
				BudgetID:        budget.ID,
				ProjectID:       projectID,
				AllocatedAmount: 10000.0,
				AllocatedBy:     "test-user",
			}

			if _, err := manager.CreateAllocation(ctx, req); err != nil {
				errors <- err
			}
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < numGoroutines; i++ {
		<-done
	}
	close(errors)

	// Check for errors
	errorCount := 0
	for err := range errors {
		errorCount++
		t.Logf("Concurrent error: %v", err)
	}
	assert.Equal(t, 0, errorCount, "No errors should occur in concurrent operations")

	// Verify all allocations were created
	allocations, err := manager.GetBudgetAllocations(ctx, budget.ID)
	require.NoError(t, err)
	assert.Equal(t, numGoroutines, len(allocations))
}

// TestBudgetManager_Persistence tests data persistence
func TestBudgetManager_Persistence(t *testing.T) {
	// Create temporary directory for testing
	tempDir, err := os.MkdirTemp("", "prism-budget-persist-*")
	require.NoError(t, err)
	defer func() { _ = os.RemoveAll(tempDir) }()

	// Mock home directory
	originalHome := os.Getenv("HOME")
	defer func() { _ = os.Setenv("HOME", originalHome) }()
	_ = os.Setenv("HOME", tempDir)

	ctx := context.Background()

	// Create first manager and budget
	manager1, err := NewBudgetManager()
	require.NoError(t, err)

	budget, err := manager1.CreateBudget(ctx, &CreateBudgetRequest{
		Name:           "Persistent Budget",
		Description:    "Test budget for persistence",
		TotalAmount:    50000.0,
		Period:         "yearly",
		StartDate:      time.Now(),
		AlertThreshold: 0.8,
		CreatedBy:      "test-user",
		Tags: map[string]string{
			"test": "persistence",
		},
	})
	require.NoError(t, err)
	budgetID := budget.ID

	// Create allocation
	allocation, err := manager1.CreateAllocation(ctx, &CreateAllocationRequest{
		BudgetID:        budgetID,
		ProjectID:       uuid.New().String(),
		AllocatedAmount: 10000.0,
		AllocatedBy:     "test-user",
	})
	require.NoError(t, err)
	allocationID := allocation.ID

	// Close first manager
	err = manager1.Close()
	require.NoError(t, err)

	// Create second manager (should load from disk)
	manager2, err := NewBudgetManager()
	require.NoError(t, err)
	defer manager2.Close()

	// Verify budget was loaded
	loadedBudget, err := manager2.GetBudget(ctx, budgetID)
	require.NoError(t, err)
	assert.Equal(t, "Persistent Budget", loadedBudget.Name)
	assert.Equal(t, 50000.0, loadedBudget.TotalAmount)
	assert.Equal(t, map[string]string{"test": "persistence"}, loadedBudget.Tags)

	// Verify allocation was loaded
	loadedAllocation, err := manager2.GetAllocation(ctx, allocationID)
	require.NoError(t, err)
	assert.Equal(t, budgetID, loadedAllocation.BudgetID)
	assert.Equal(t, 10000.0, loadedAllocation.AllocatedAmount)
}

// TestBudgetManager_MigratesLegacyFiles verifies the pre-seam flat budgets.json /
// budget_allocations.json files are imported into the seam on first construction.
func TestBudgetManager_MigratesLegacyFiles(t *testing.T) {
	tempDir := t.TempDir()
	originalHome := os.Getenv("HOME")
	originalStateDir := os.Getenv("PRISM_STATE_DIR")
	t.Cleanup(func() {
		_ = os.Setenv("HOME", originalHome)
		_ = os.Setenv("PRISM_STATE_DIR", originalStateDir)
	})
	_ = os.Setenv("HOME", tempDir)
	_ = os.Unsetenv("PRISM_STATE_DIR")

	stateDir := tempDir + "/.prism"
	require.NoError(t, os.MkdirAll(stateDir, 0o755))
	require.NoError(t, os.WriteFile(stateDir+"/budgets.json",
		[]byte(`{"bud-legacy":{"id":"bud-legacy","name":"Legacy Grant","total_amount":1000}}`), 0o644))
	require.NoError(t, os.WriteFile(stateDir+"/budget_allocations.json",
		[]byte(`{"alloc-legacy":{"id":"alloc-legacy","budget_id":"bud-legacy","project_id":"proj-1","allocated_amount":250}}`), 0o644))

	ctx := context.Background()
	mgr, err := NewBudgetManager()
	require.NoError(t, err)

	b, err := mgr.GetBudget(ctx, "bud-legacy")
	require.NoError(t, err)
	assert.Equal(t, "Legacy Grant", b.Name)

	a, err := mgr.GetAllocation(ctx, "alloc-legacy")
	require.NoError(t, err)
	assert.Equal(t, "bud-legacy", a.BudgetID)

	// Legacy files retired; a second manager does not double-import.
	_, statErr := os.Stat(stateDir + "/budgets.json")
	assert.True(t, os.IsNotExist(statErr), "legacy budgets.json should be renamed aside")

	mgr2, err := NewBudgetManager()
	require.NoError(t, err)
	all, err := mgr2.ListBudgets(ctx)
	require.NoError(t, err)
	assert.Len(t, all, 1)
}

// TestBudgetManager_UpdateBudget_EdgeCases tests additional edge cases for budget updates
func TestBudgetManager_UpdateBudget_EdgeCases(t *testing.T) {
	manager := setupTestBudgetManager(t)
	ctx := context.Background()

	// Create a test budget
	budget, err := manager.CreateBudget(ctx, &CreateBudgetRequest{
		Name:           "Test Budget for Updates",
		Description:    "Test budget",
		TotalAmount:    100000.0,
		Period:         "yearly",
		StartDate:      time.Now(),
		AlertThreshold: 0.8,
		CreatedBy:      "test-user",
	})
	require.NoError(t, err)

	// Test updating with nil fields (should not change those fields)
	updateReq := &UpdateBudgetRequest{}
	updated, err := manager.UpdateBudget(ctx, budget.ID, updateReq)
	assert.NoError(t, err)
	assert.Equal(t, budget.Name, updated.Name, "name should not change")
	assert.Equal(t, budget.TotalAmount, updated.TotalAmount, "amount should not change")

	// Test updating description to empty string
	updateReq2 := &UpdateBudgetRequest{
		Description: stringPtr(""),
	}
	updated2, err := manager.UpdateBudget(ctx, budget.ID, updateReq2)
	assert.NoError(t, err)
	assert.Equal(t, "", updated2.Description)

	// Test updating with valid tags
	updateReq3 := &UpdateBudgetRequest{
		Tags: map[string]string{
			"updated": "true",
			"version": "2",
		},
	}
	updated3, err := manager.UpdateBudget(ctx, budget.ID, updateReq3)
	assert.NoError(t, err)
	assert.Equal(t, "true", updated3.Tags["updated"])
	assert.Equal(t, "2", updated3.Tags["version"])
}

// TestBudgetManager_CreateAllocation_EdgeCases tests edge cases for allocation creation
func TestBudgetManager_CreateAllocation_EdgeCases(t *testing.T) {
	manager := setupTestBudgetManager(t)
	ctx := context.Background()

	// Create test budget
	budget, err := manager.CreateBudget(ctx, &CreateBudgetRequest{
		Name:           "Test Budget",
		Description:    "For allocation edge cases",
		TotalAmount:    100000.0,
		Period:         "yearly",
		StartDate:      time.Now(),
		AlertThreshold: 0.8,
		CreatedBy:      "test-user",
	})
	require.NoError(t, err)

	// Test allocation with very large amount (within budget)
	largeAlloc, err := manager.CreateAllocation(ctx, &CreateAllocationRequest{
		BudgetID:        budget.ID,
		ProjectID:       "large-project",
		AllocatedAmount: 99999.99,
		AllocatedBy:     "test-user",
	})
	assert.NoError(t, err)
	assert.Equal(t, 99999.99, largeAlloc.AllocatedAmount)

	// Test allocation with custom alert threshold
	customAlloc, err := manager.CreateAllocation(ctx, &CreateAllocationRequest{
		BudgetID:        "non-existent-budget",
		ProjectID:       "custom-project",
		AllocatedAmount: 1000.0,
		AlertThreshold:  floatPtr(0.95),
		AllocatedBy:     "test-user",
	})
	assert.Error(t, err)
	assert.Nil(t, customAlloc)
	assert.Contains(t, err.Error(), "not found")
}

// TestBudgetManager_DeleteBudget_WithAllocations tests deleting budget with existing allocations
func TestBudgetManager_DeleteBudget_WithAllocations(t *testing.T) {
	manager := setupTestBudgetManager(t)
	ctx := context.Background()

	// Create budget
	budget, err := manager.CreateBudget(ctx, &CreateBudgetRequest{
		Name:           "Budget with Allocations",
		Description:    "Will have allocations",
		TotalAmount:    50000.0,
		Period:         "yearly",
		StartDate:      time.Now(),
		AlertThreshold: 0.8,
		CreatedBy:      "test-user",
	})
	require.NoError(t, err)

	// Create allocations
	_, err = manager.CreateAllocation(ctx, &CreateAllocationRequest{
		BudgetID:        budget.ID,
		ProjectID:       "project-1",
		AllocatedAmount: 10000.0,
		AllocatedBy:     "test-user",
	})
	require.NoError(t, err)

	_, err = manager.CreateAllocation(ctx, &CreateAllocationRequest{
		BudgetID:        budget.ID,
		ProjectID:       "project-2",
		AllocatedAmount: 15000.0,
		AllocatedBy:     "test-user",
	})
	require.NoError(t, err)

	// Try to delete budget (should fail because it has allocations)
	err = manager.DeleteBudget(ctx, budget.ID)
	assert.Error(t, err, "should not allow deleting budget with allocations")
	assert.Contains(t, err.Error(), "cannot delete budget with")

	// Verify budget still exists
	existingBudget, err := manager.GetBudget(ctx, budget.ID)
	assert.NoError(t, err)
	assert.NotNil(t, existingBudget)

	// Verify allocations still exist
	allocations, err := manager.GetBudgetAllocations(ctx, budget.ID)
	require.NoError(t, err)
	assert.Len(t, allocations, 2, "allocations should still exist")
}

// TestBudgetManager_ListBudgets_Filtering tests budget listing with various scenarios
func TestBudgetManager_ListBudgets_Filtering(t *testing.T) {
	manager := setupTestBudgetManager(t)
	ctx := context.Background()

	// Create several budgets
	budgets := []struct {
		name   string
		amount float64
		tags   map[string]string
	}{
		{
			name:   "NSF Grant 2024",
			amount: 100000.0,
			tags:   map[string]string{"agency": "NSF", "year": "2024"},
		},
		{
			name:   "DOE Grant 2024",
			amount: 75000.0,
			tags:   map[string]string{"agency": "DOE", "year": "2024"},
		},
		{
			name:   "Internal Funding",
			amount: 25000.0,
			tags:   map[string]string{"type": "internal"},
		},
	}

	for _, b := range budgets {
		_, err := manager.CreateBudget(ctx, &CreateBudgetRequest{
			Name:           b.name,
			Description:    "Test budget",
			TotalAmount:    b.amount,
			Period:         "yearly",
			StartDate:      time.Now(),
			AlertThreshold: 0.8,
			CreatedBy:      "test-user",
			Tags:           b.tags,
		})
		require.NoError(t, err)
	}

	// List all budgets
	allBudgets, err := manager.ListBudgets(ctx)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(allBudgets), 3, "should have at least 3 budgets")

	// Verify budgets are in the list
	budgetNames := make(map[string]bool)
	for _, budget := range allBudgets {
		budgetNames[budget.Name] = true
	}
	assert.True(t, budgetNames["NSF Grant 2024"])
	assert.True(t, budgetNames["DOE Grant 2024"])
	assert.True(t, budgetNames["Internal Funding"])
}

// TestBudgetManager_GetBudget_AfterUpdates tests getting budget after multiple updates
func TestBudgetManager_GetBudget_AfterUpdates(t *testing.T) {
	manager := setupTestBudgetManager(t)
	ctx := context.Background()

	// Create budget
	originalBudget, err := manager.CreateBudget(ctx, &CreateBudgetRequest{
		Name:           "Evolving Budget",
		Description:    "Will be updated",
		TotalAmount:    50000.0,
		Period:         "yearly",
		StartDate:      time.Now(),
		AlertThreshold: 0.7,
		CreatedBy:      "test-user",
	})
	require.NoError(t, err)

	// Update name
	_, err = manager.UpdateBudget(ctx, originalBudget.ID, &UpdateBudgetRequest{
		Name: stringPtr("Updated Budget Name"),
	})
	require.NoError(t, err)

	// Update amount
	_, err = manager.UpdateBudget(ctx, originalBudget.ID, &UpdateBudgetRequest{
		TotalAmount: floatPtr(75000.0),
	})
	require.NoError(t, err)

	// Update threshold
	_, err = manager.UpdateBudget(ctx, originalBudget.ID, &UpdateBudgetRequest{
		AlertThreshold: floatPtr(0.85),
	})
	require.NoError(t, err)

	// Get final budget and verify all updates
	finalBudget, err := manager.GetBudget(ctx, originalBudget.ID)
	require.NoError(t, err)
	assert.Equal(t, "Updated Budget Name", finalBudget.Name)
	assert.Equal(t, 75000.0, finalBudget.TotalAmount)
	assert.Equal(t, 0.85, finalBudget.AlertThreshold)
}

// TestBudgetManager_GetBudgetSummary_Complex tests summary with multiple allocations
func TestBudgetManager_GetBudgetSummary_Complex(t *testing.T) {
	manager := setupTestBudgetManager(t)
	ctx := context.Background()

	// Create budget
	budget, err := manager.CreateBudget(ctx, &CreateBudgetRequest{
		Name:           "Complex Summary Budget",
		Description:    "For testing summary calculations",
		TotalAmount:    100000.0,
		Period:         "yearly",
		StartDate:      time.Now(),
		AlertThreshold: 0.8,
		CreatedBy:      "test-user",
	})
	require.NoError(t, err)

	// Create multiple allocations
	projects := []struct {
		id     string
		amount float64
	}{
		{"project-alpha", 25000.0},
		{"project-beta", 30000.0},
		{"project-gamma", 20000.0},
	}

	for _, proj := range projects {
		_, err := manager.CreateAllocation(ctx, &CreateAllocationRequest{
			BudgetID:        budget.ID,
			ProjectID:       proj.id,
			AllocatedAmount: proj.amount,
			AllocatedBy:     "test-user",
		})
		require.NoError(t, err)
	}

	// Get summary
	summary, err := manager.GetBudgetSummary(ctx, budget.ID)
	require.NoError(t, err)

	// Verify summary calculations
	assert.Equal(t, budget.ID, summary.Budget.ID)
	assert.Equal(t, 100000.0, summary.Budget.TotalAmount)
	assert.Equal(t, 75000.0, summary.Budget.AllocatedAmount, "allocated should be sum of allocations")
	assert.Equal(t, 25000.0, summary.RemainingAmount, "remaining should be total - allocated")
	assert.Len(t, summary.Allocations, 3, "should have 3 allocations")
}

// TestBudgetManager_GetProjectAllocations_MultipleTest tests project with multiple budget sources
func TestBudgetManager_GetProjectAllocations_MultipleTest(t *testing.T) {
	manager := setupTestBudgetManager(t)
	ctx := context.Background()

	projectID := "multi-funded-project"

	// Create multiple budgets
	budget1, err := manager.CreateBudget(ctx, &CreateBudgetRequest{
		Name:           "Budget Source 1",
		Description:    "First funding source",
		TotalAmount:    50000.0,
		Period:         "yearly",
		StartDate:      time.Now(),
		AlertThreshold: 0.8,
		CreatedBy:      "test-user",
	})
	require.NoError(t, err)

	budget2, err := manager.CreateBudget(ctx, &CreateBudgetRequest{
		Name:           "Budget Source 2",
		Description:    "Second funding source",
		TotalAmount:    50000.0,
		Period:         "yearly",
		StartDate:      time.Now(),
		AlertThreshold: 0.8,
		CreatedBy:      "test-user",
	})
	require.NoError(t, err)

	// Allocate from both budgets to same project
	_, err = manager.CreateAllocation(ctx, &CreateAllocationRequest{
		BudgetID:        budget1.ID,
		ProjectID:       projectID,
		AllocatedAmount: 15000.0,
		AllocatedBy:     "test-user",
	})
	require.NoError(t, err)

	_, err = manager.CreateAllocation(ctx, &CreateAllocationRequest{
		BudgetID:        budget2.ID,
		ProjectID:       projectID,
		AllocatedAmount: 20000.0,
		AllocatedBy:     "test-user",
	})
	require.NoError(t, err)

	// Get all allocations for the project
	allocations, err := manager.GetProjectAllocations(ctx, projectID)
	require.NoError(t, err)
	assert.Len(t, allocations, 2, "project should have allocations from 2 budgets")

	// Verify total allocated to project
	totalAllocated := 0.0
	for _, alloc := range allocations {
		totalAllocated += alloc.AllocatedAmount
	}
	assert.Equal(t, 35000.0, totalAllocated, "total allocated from both sources")
}
