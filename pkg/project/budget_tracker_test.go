package project

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/scttfrdmn/cloudworkstation/pkg/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewBudgetTracker(t *testing.T) {
	// Create temporary directory for testing
	tempDir, err := os.MkdirTemp("", "cws-budget-test-*")
	require.NoError(t, err)
	defer func() { _ = os.RemoveAll(tempDir) }()

	// Mock home directory
	originalHome := os.Getenv("HOME")
	defer func() { _ = os.Setenv("HOME", originalHome) }()
	_ = os.Setenv("HOME", tempDir)

	tracker, err := NewBudgetTracker()
	assert.NoError(t, err)
	assert.NotNil(t, tracker)
	assert.NotNil(t, tracker.budgetData)
	assert.NotNil(t, tracker.costCalculator)
}

func TestBudgetTracker_InitializeProject(t *testing.T) {
	tracker := setupTestBudgetTracker(t)
	defer func() { _ = tracker.Close() }()

	projectID := uuid.New().String()
	budget := &types.ProjectBudget{
		TotalBudget:  1000.0,
		SpentAmount:  0.0,
		MonthlyLimit: floatPtr(300.0),
		DailyLimit:   floatPtr(50.0),
		AlertThresholds: []types.BudgetAlert{
			{
				Threshold:  0.8,
				Type:       types.BudgetAlertEmail,
				Recipients: []string{"admin@example.com"},
			},
		},
		AutoActions: []types.BudgetAutoAction{
			{
				Threshold: 0.95,
				Action:    types.BudgetActionHibernateAll,
			},
		},
		BudgetPeriod: types.BudgetPeriodMonthly,
		StartDate:    time.Now(),
		EndDate:      timePtr(time.Now().AddDate(0, 3, 0)), // 3 months from now
		LastUpdated:  time.Now(),
	}

	err := tracker.InitializeProject(projectID, budget)
	assert.NoError(t, err)

	// Verify project data was created
	tracker.mutex.RLock()
	budgetData, exists := tracker.budgetData[projectID]
	tracker.mutex.RUnlock()

	assert.True(t, exists)
	assert.NotNil(t, budgetData)
	assert.Equal(t, projectID, budgetData.ProjectID)
	assert.Equal(t, budget, budgetData.Budget)
	assert.Empty(t, budgetData.CostHistory)
	assert.Empty(t, budgetData.AlertHistory)
	assert.WithinDuration(t, time.Now(), budgetData.LastUpdated, time.Second)
}

func TestBudgetTracker_RemoveProject(t *testing.T) {
	tracker := setupTestBudgetTracker(t)
	defer func() { _ = tracker.Close() }()

	projectID := uuid.New().String()
	budget := &types.ProjectBudget{
		TotalBudget:  500.0,
		SpentAmount:  0.0,
		BudgetPeriod: types.BudgetPeriodMonthly,
		StartDate:    time.Now(),
		LastUpdated:  time.Now(),
	}

	// Initialize project
	err := tracker.InitializeProject(projectID, budget)
	require.NoError(t, err)

	// Verify project exists
	tracker.mutex.RLock()
	_, exists := tracker.budgetData[projectID]
	tracker.mutex.RUnlock()
	assert.True(t, exists)

	// Remove project
	err = tracker.RemoveProject(projectID)
	assert.NoError(t, err)

	// Verify project was removed
	tracker.mutex.RLock()
	_, exists = tracker.budgetData[projectID]
	tracker.mutex.RUnlock()
	assert.False(t, exists)
}

func TestBudgetTracker_CheckBudgetStatus(t *testing.T) {
	tracker := setupTestBudgetTracker(t)
	defer func() { _ = tracker.Close() }()

	projectID := uuid.New().String()
	budget := &types.ProjectBudget{
		TotalBudget:  1000.0,
		SpentAmount:  300.0, // 30% spent
		BudgetPeriod: types.BudgetPeriodMonthly,
		StartDate:    time.Now().AddDate(0, 0, -15), // Started 15 days ago
		LastUpdated:  time.Now(),
		AlertThresholds: []types.BudgetAlert{
			{
				Threshold:  0.5,
				Type:       types.BudgetAlertEmail,
				Recipients: []string{"admin@example.com"},
			},
			{
				Threshold:  0.8,
				Type:       types.BudgetAlertSlack,
				Recipients: []string{"#alerts"},
			},
		},
		AutoActions: []types.BudgetAutoAction{
			{
				Threshold: 0.9,
				Action:    types.BudgetActionStopAll,
			},
		},
	}

	err := tracker.InitializeProject(projectID, budget)
	require.NoError(t, err)

	tests := []struct {
		name        string
		projectID   string
		wantErr     bool
		checkFields func(t *testing.T, status *BudgetStatus)
	}{
		{
			name:      "check existing project budget status",
			projectID: projectID,
			wantErr:   false,
			checkFields: func(t *testing.T, status *BudgetStatus) {
				assert.Equal(t, projectID, status.ProjectID)
				assert.True(t, status.BudgetEnabled)
				assert.Equal(t, 1000.0, status.TotalBudget)
				assert.Equal(t, 300.0, status.SpentAmount)
				assert.Equal(t, 700.0, status.RemainingBudget)
				assert.InDelta(t, 0.3, status.SpentPercentage, 0.01)
				assert.WithinDuration(t, time.Now(), status.LastUpdated, time.Second)
			},
		},
		{
			name:      "check non-existent project",
			projectID: uuid.New().String(),
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			status, err := tracker.CheckBudgetStatus(tt.projectID)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, status)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, status)
				if tt.checkFields != nil {
					tt.checkFields(t, status)
				}
			}
		})
	}
}

func TestBudgetTracker_UpdateProjectSpending(t *testing.T) {
	tracker := setupTestBudgetTracker(t)
	defer func() { _ = tracker.Close() }()

	projectID := uuid.New().String()
	budget := &types.ProjectBudget{
		TotalBudget:  1000.0,
		SpentAmount:  100.0,
		BudgetPeriod: types.BudgetPeriodMonthly,
		StartDate:    time.Now().AddDate(0, 0, -10),
		LastUpdated:  time.Now(),
		AlertThresholds: []types.BudgetAlert{
			{
				Threshold:  0.5,
				Type:       types.BudgetAlertEmail,
				Recipients: []string{"admin@example.com"},
			},
		},
	}

	err := tracker.InitializeProject(projectID, budget)
	require.NoError(t, err)

	// Test spending updates
	tests := []struct {
		name          string
		instanceCosts []types.InstanceCost
		storageCosts  []types.StorageCost
		expectedTotal float64
		expectedDaily float64
	}{
		{
			name: "update with instance costs only",
			instanceCosts: []types.InstanceCost{
				{
					InstanceName:    "test-instance",
					InstanceType:    "t3.medium",
					ComputeCost:     25.0,
					StorageCost:     0.0,
					TotalCost:       25.0,
					RunningHours:    24.0,
					HibernatedHours: 0.0,
					StoppedHours:    0.0,
				},
				{
					InstanceName:    "ml-instance",
					InstanceType:    "p3.2xlarge",
					ComputeCost:     100.0,
					StorageCost:     0.0,
					TotalCost:       100.0,
					RunningHours:    24.0,
					HibernatedHours: 0.0,
					StoppedHours:    0.0,
				},
			},
			storageCosts:  []types.StorageCost{},
			expectedTotal: 225.0, // 100 (initial) + 25 + 100
			expectedDaily: 125.0, // 25 + 100
		},
		{
			name:          "update with storage costs only",
			instanceCosts: []types.InstanceCost{},
			storageCosts: []types.StorageCost{
				{
					VolumeName: "data-volume",
					VolumeType: "gp3",
					SizeGB:     1000,
					Cost:       5.0,
					CostPerGB:  0.005,
				},
			},
			expectedTotal: 230.0, // Previous total + 5
			expectedDaily: 5.0,
		},
		{
			name: "update with both instance and storage costs",
			instanceCosts: []types.InstanceCost{
				{
					InstanceName:    "mixed-instance",
					InstanceType:    "r5.large",
					ComputeCost:     50.0,
					StorageCost:     0.0,
					TotalCost:       50.0,
					RunningHours:    24.0,
					HibernatedHours: 0.0,
					StoppedHours:    0.0,
				},
			},
			storageCosts: []types.StorageCost{
				{
					VolumeName: "mixed-volume",
					VolumeType: "io2",
					SizeGB:     500,
					Cost:       10.0,
					CostPerGB:  0.02,
				},
			},
			expectedTotal: 290.0, // Previous total + 50 + 10
			expectedDaily: 60.0,  // 50 + 10
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tracker.UpdateProjectSpending(projectID, tt.instanceCosts, tt.storageCosts)
			assert.NoError(t, err)

			// Verify budget was updated
			tracker.mutex.RLock()
			budgetData := tracker.budgetData[projectID]
			tracker.mutex.RUnlock()

			assert.Equal(t, tt.expectedTotal, budgetData.Budget.SpentAmount)
			assert.NotEmpty(t, budgetData.CostHistory)

			// Check the latest cost data point
			latestCost := budgetData.CostHistory[len(budgetData.CostHistory)-1]
			assert.Equal(t, tt.expectedTotal, latestCost.TotalCost)
			assert.Equal(t, tt.expectedDaily, latestCost.DailyCost)
			assert.Equal(t, len(tt.instanceCosts), len(latestCost.InstanceCosts))
			assert.Equal(t, len(tt.storageCosts), len(latestCost.StorageCosts))
			assert.WithinDuration(t, time.Now(), latestCost.Timestamp, time.Second)
		})
	}
}

func TestBudgetTracker_GetCostBreakdown(t *testing.T) {
	tracker := setupTestBudgetTracker(t)
	defer func() { _ = tracker.Close() }()

	projectID := uuid.New().String()
	budget := &types.ProjectBudget{
		TotalBudget:  1000.0,
		SpentAmount:  0.0,
		BudgetPeriod: types.BudgetPeriodMonthly,
		StartDate:    time.Now().AddDate(0, 0, -30),
		LastUpdated:  time.Now(),
	}

	err := tracker.InitializeProject(projectID, budget)
	require.NoError(t, err)

	// Add some cost history
	instanceCosts := []types.InstanceCost{
		{
			InstanceName:    "test-instance",
			InstanceType:    "t3.medium",
			ComputeCost:     25.0,
			StorageCost:     0.0,
			TotalCost:       25.0,
			RunningHours:    24.0,
			HibernatedHours: 0.0,
			StoppedHours:    0.0,
		},
	}

	storageCosts := []types.StorageCost{
		{
			VolumeName: "data-volume",
			VolumeType: "gp3",
			SizeGB:     1000,
			Cost:       5.0,
			CostPerGB:  0.005,
		},
	}

	err = tracker.UpdateProjectSpending(projectID, instanceCosts, storageCosts)
	require.NoError(t, err)

	startDate := time.Now().AddDate(0, 0, -7) // 7 days ago
	endDate := time.Now()

	breakdown, err := tracker.GetCostBreakdown(projectID, startDate, endDate)
	assert.NoError(t, err)
	assert.NotNil(t, breakdown)

	// Verify basic fields
	assert.Equal(t, projectID, breakdown.ProjectID)
	assert.Equal(t, startDate, breakdown.PeriodStart)
	assert.Equal(t, endDate, breakdown.PeriodEnd)
	assert.True(t, breakdown.TotalCost > 0)

	// Verify cost breakdowns
	assert.NotNil(t, breakdown.InstanceCosts)
	assert.NotNil(t, breakdown.StorageCosts)

	// Should have at least some data
	assert.True(t, len(breakdown.InstanceCosts) > 0 || len(breakdown.StorageCosts) > 0)
}

func TestBudgetTracker_GetResourceUsage(t *testing.T) {
	tracker := setupTestBudgetTracker(t)
	defer func() { _ = tracker.Close() }()

	projectID := uuid.New().String()
	budget := &types.ProjectBudget{
		TotalBudget:  1000.0,
		SpentAmount:  0.0,
		BudgetPeriod: types.BudgetPeriodMonthly,
		StartDate:    time.Now().AddDate(0, 0, -30),
		LastUpdated:  time.Now(),
	}

	err := tracker.InitializeProject(projectID, budget)
	require.NoError(t, err)

	period := 7 * 24 * time.Hour // 7 days

	usage, err := tracker.GetResourceUsage(projectID, period)
	assert.NoError(t, err)
	assert.NotNil(t, usage)

	// Verify basic fields
	assert.Equal(t, projectID, usage.ProjectID)
	assert.Equal(t, period, usage.MeasurementPeriod)
	assert.WithinDuration(t, time.Now(), usage.LastUpdated, time.Minute)

	// Verify usage metrics structure
	assert.GreaterOrEqual(t, usage.ActiveInstances, 0)
	assert.GreaterOrEqual(t, usage.TotalInstances, 0)
	assert.GreaterOrEqual(t, usage.TotalStorage, 0.0)
	assert.GreaterOrEqual(t, usage.ComputeHours, 0.0)
	assert.GreaterOrEqual(t, usage.IdleSavings, 0.0)
}

func TestBudgetTracker_AlertTriggering(t *testing.T) {
	tracker := setupTestBudgetTracker(t)
	defer func() { _ = tracker.Close() }()

	projectID := uuid.New().String()
	budget := &types.ProjectBudget{
		TotalBudget:  1000.0,
		SpentAmount:  0.0,
		BudgetPeriod: types.BudgetPeriodMonthly,
		StartDate:    time.Now().AddDate(0, 0, -10),
		LastUpdated:  time.Now(),
		AlertThresholds: []types.BudgetAlert{
			{
				Threshold:  0.5, // 50%
				Type:       types.BudgetAlertEmail,
				Recipients: []string{"admin@example.com"},
			},
			{
				Threshold:  0.8, // 80%
				Type:       types.BudgetAlertSlack,
				Recipients: []string{"#alerts"},
			},
		},
		AutoActions: []types.BudgetAutoAction{
			{
				Threshold: 0.9, // 90%
				Action:    types.BudgetActionHibernateAll,
			},
		},
	}

	err := tracker.InitializeProject(projectID, budget)
	require.NoError(t, err)

	tests := []struct {
		name                string
		spendingAmount      float64
		expectedAlertCount  int
		expectedActionCount int
	}{
		{
			name:                "spending below first threshold",
			spendingAmount:      400.0, // 40%
			expectedAlertCount:  0,
			expectedActionCount: 0,
		},
		{
			name:                "spending triggers first alert",
			spendingAmount:      600.0, // 60% (triggers 50% alert)
			expectedAlertCount:  1,
			expectedActionCount: 0,
		},
		{
			name:                "spending triggers second alert",
			spendingAmount:      850.0, // 85% (triggers both 50% and 80% alerts)
			expectedAlertCount:  2,
			expectedActionCount: 0,
		},
		{
			name:                "spending triggers auto action",
			spendingAmount:      950.0, // 95% (triggers all alerts and auto action)
			expectedAlertCount:  2,
			expectedActionCount: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset budget data for this test by reinitializing the project
			testTracker := setupTestBudgetTracker(t)

			// Create a fresh budget with zero spending for isolated testing
			freshBudget := *budget
			freshBudget.SpentAmount = 0.0

			err := testTracker.InitializeProject(projectID, &freshBudget)
			require.NoError(t, err)

			// Update spending to target amount
			instanceCosts := []types.InstanceCost{
				{
					InstanceName:    "test-instance",
					InstanceType:    "t3.large",
					ComputeCost:     tt.spendingAmount,
					StorageCost:     0.0,
					TotalCost:       tt.spendingAmount,
					RunningHours:    24.0,
					HibernatedHours: 0.0,
					StoppedHours:    0.0,
				},
			}

			err = testTracker.UpdateProjectSpending(projectID, instanceCosts, []types.StorageCost{})
			assert.NoError(t, err)

			// Check budget status
			status, err := testTracker.CheckBudgetStatus(projectID)
			assert.NoError(t, err)
			assert.NotNil(t, status)

			// Verify alert and action counts
			assert.Len(t, status.ActiveAlerts, tt.expectedAlertCount)
			assert.Len(t, status.TriggeredActions, tt.expectedActionCount)

			// Verify spending percentage
			expectedPercentage := tt.spendingAmount / 1000.0
			assert.InDelta(t, expectedPercentage, status.SpentPercentage, 0.01)
		})
	}
}

func TestProjectFilter_Matches(t *testing.T) {
	now := time.Now()
	project := &types.Project{
		ID:          "test-id",
		Name:        "Test Project",
		Description: "A test project",
		Owner:       "test-user",
		Status:      types.ProjectStatusActive,
		Tags: map[string]string{
			"department": "research",
			"priority":   "high",
		},
		CreatedAt: now.AddDate(0, 0, -7), // 7 days ago
		Budget: &types.ProjectBudget{
			TotalBudget: 1000.0,
		},
	}

	tests := []struct {
		name    string
		filter  *ProjectFilter
		matches bool
	}{
		{
			name:    "nil filter matches all",
			filter:  nil,
			matches: true,
		},
		{
			name: "owner filter matches",
			filter: &ProjectFilter{
				Owner: "test-user",
			},
			matches: true,
		},
		{
			name: "owner filter doesn't match",
			filter: &ProjectFilter{
				Owner: "other-user",
			},
			matches: false,
		},
		{
			name: "status filter matches",
			filter: &ProjectFilter{
				Status: func() *types.ProjectStatus { s := types.ProjectStatusActive; return &s }(),
			},
			matches: true,
		},
		{
			name: "status filter doesn't match",
			filter: &ProjectFilter{
				Status: func() *types.ProjectStatus { s := types.ProjectStatusArchived; return &s }(),
			},
			matches: false,
		},
		{
			name: "has budget filter matches (has budget)",
			filter: &ProjectFilter{
				HasBudget: boolPtr(true),
			},
			matches: true,
		},
		{
			name: "has budget filter doesn't match (expects no budget)",
			filter: &ProjectFilter{
				HasBudget: boolPtr(false),
			},
			matches: false,
		},
		{
			name: "created after filter matches",
			filter: &ProjectFilter{
				CreatedAfter: timePtr(now.AddDate(0, 0, -10)), // 10 days ago
			},
			matches: true,
		},
		{
			name: "created after filter doesn't match",
			filter: &ProjectFilter{
				CreatedAfter: timePtr(now.AddDate(0, 0, -5)), // 5 days ago
			},
			matches: false,
		},
		{
			name: "created before filter matches",
			filter: &ProjectFilter{
				CreatedBefore: timePtr(now.AddDate(0, 0, -5)), // 5 days ago
			},
			matches: true,
		},
		{
			name: "created before filter doesn't match",
			filter: &ProjectFilter{
				CreatedBefore: timePtr(now.AddDate(0, 0, -10)), // 10 days ago
			},
			matches: false,
		},
		{
			name: "tags filter matches single tag",
			filter: &ProjectFilter{
				Tags: map[string]string{
					"department": "research",
				},
			},
			matches: true,
		},
		{
			name: "tags filter matches multiple tags",
			filter: &ProjectFilter{
				Tags: map[string]string{
					"department": "research",
					"priority":   "high",
				},
			},
			matches: true,
		},
		{
			name: "tags filter doesn't match",
			filter: &ProjectFilter{
				Tags: map[string]string{
					"department": "engineering",
				},
			},
			matches: false,
		},
		{
			name: "tags filter doesn't match missing tag",
			filter: &ProjectFilter{
				Tags: map[string]string{
					"nonexistent": "value",
				},
			},
			matches: false,
		},
		{
			name: "complex filter matches all conditions",
			filter: &ProjectFilter{
				Owner:         "test-user",
				Status:        func() *types.ProjectStatus { s := types.ProjectStatusActive; return &s }(),
				HasBudget:     boolPtr(true),
				CreatedAfter:  timePtr(now.AddDate(0, 0, -10)),
				CreatedBefore: timePtr(now.AddDate(0, 0, -5)),
				Tags: map[string]string{
					"department": "research",
				},
			},
			matches: true,
		},
		{
			name: "complex filter fails on one condition",
			filter: &ProjectFilter{
				Owner:         "test-user",
				Status:        func() *types.ProjectStatus { s := types.ProjectStatusArchived; return &s }(), // Wrong status
				HasBudget:     boolPtr(true),
				CreatedAfter:  timePtr(now.AddDate(0, 0, -10)),
				CreatedBefore: timePtr(now.AddDate(0, 0, -5)),
				Tags: map[string]string{
					"department": "research",
				},
			},
			matches: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.filter == nil || tt.filter.Matches(project)
			assert.Equal(t, tt.matches, result)
		})
	}
}

// Helper functions
func setupTestBudgetTracker(t *testing.T) *BudgetTracker {
	t.Helper()

	// Create temporary directory for testing
	tempDir, err := os.MkdirTemp("", "cws-budget-test-*")
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = os.RemoveAll(tempDir)
	})

	// Create the .cloudworkstation directory
	stateDir := filepath.Join(tempDir, ".cloudworkstation")
	err = os.MkdirAll(stateDir, 0755)
	require.NoError(t, err)

	// Mock home directory
	originalHome := os.Getenv("HOME")
	t.Cleanup(func() {
		_ = os.Setenv("HOME", originalHome)
	})
	_ = os.Setenv("HOME", tempDir)

	tracker, err := NewBudgetTracker()
	require.NoError(t, err)
	require.NotNil(t, tracker)

	return tracker
}

func timePtr(t time.Time) *time.Time {
	return &t
}

func boolPtr(b bool) *bool {
	return &b
}
