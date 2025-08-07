package project

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/scttfrdmn/cloudworkstation/pkg/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewManager(t *testing.T) {
	tests := []struct {
		name    string
		wantErr bool
	}{
		{
			name:    "successful manager creation",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary directory for testing
			tempDir, err := os.MkdirTemp("", "cws-project-test-*")
			require.NoError(t, err)
			defer os.RemoveAll(tempDir)

			// Mock home directory
			originalHome := os.Getenv("HOME")
			defer os.Setenv("HOME", originalHome)
			os.Setenv("HOME", tempDir)

			manager, err := NewManager()

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, manager)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, manager)
				assert.NotNil(t, manager.budgetTracker)
				assert.NotNil(t, manager.projects)
				
				// Verify state directory was created
				stateDir := filepath.Join(tempDir, ".cloudworkstation")
				assert.DirExists(t, stateDir)
			}
		})
	}
}

func TestManager_CreateProject(t *testing.T) {
	manager := setupTestManager(t)
	defer teardownTestManager(manager)

	tests := []struct {
		name    string
		req     *CreateProjectRequest
		wantErr bool
		errMsg  string
	}{
		{
			name: "successful project creation",
			req: &CreateProjectRequest{
				Name:        "Test Project",
				Description: "A test project for CloudWorkstation",
				Owner:       "test-user",
				Tags: map[string]string{
					"department": "research",
					"grant":      "NSF-12345",
				},
			},
			wantErr: false,
		},
		{
			name: "project with budget",
			req: &CreateProjectRequest{
				Name:        "Budgeted Project",
				Description: "A project with budget tracking",
				Owner:       "test-user",
				Budget: &CreateBudgetRequest{
					TotalBudget:  1000.0,
					MonthlyLimit: floatPtr(300.0),
					DailyLimit:   floatPtr(50.0),
					BudgetPeriod: types.BudgetPeriodMonthly,
					AlertThresholds: []types.BudgetAlert{
						{
							Threshold: 0.8,
							Type:      types.BudgetAlertEmail,
							Recipients: []string{"admin@example.com"},
						},
					},
					AutoActions: []types.BudgetAutoAction{
						{
							Threshold: 0.95,
							Action:    types.BudgetActionHibernateAll,
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "project with empty name",
			req: &CreateProjectRequest{
				Name:        "",
				Description: "Invalid project",
				Owner:       "test-user",
			},
			wantErr: true,
			errMsg:  "project name is required",
		},
		{
			name: "project with long name",
			req: &CreateProjectRequest{
				Name:        generateLongString(101),
				Description: "Invalid project",
				Owner:       "test-user",
			},
			wantErr: true,
			errMsg:  "project name cannot exceed 100 characters",
		},
		{
			name: "project with long description",
			req: &CreateProjectRequest{
				Name:        "Valid Project",
				Description: generateLongString(1001),
				Owner:       "test-user",
			},
			wantErr: true,
			errMsg:  "project description cannot exceed 1000 characters",
		},
		{
			name: "project with invalid budget",
			req: &CreateProjectRequest{
				Name:        "Invalid Budget Project",
				Description: "Project with invalid budget",
				Owner:       "test-user",
				Budget: &CreateBudgetRequest{
					TotalBudget: -100.0, // Invalid negative budget
				},
			},
			wantErr: true,
			errMsg:  "total budget must be greater than 0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			project, err := manager.CreateProject(ctx, tt.req)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, project)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, project)
				assert.NotEmpty(t, project.ID)
				assert.Equal(t, tt.req.Name, project.Name)
				assert.Equal(t, tt.req.Description, project.Description)
				assert.Equal(t, tt.req.Owner, project.Owner)
				assert.Equal(t, types.ProjectStatusActive, project.Status)
				assert.WithinDuration(t, time.Now(), project.CreatedAt, time.Second)
				assert.WithinDuration(t, time.Now(), project.UpdatedAt, time.Second)

				// Verify tags
				if tt.req.Tags != nil {
					assert.Equal(t, tt.req.Tags, project.Tags)
				}

				// Verify owner is added as project member
				if tt.req.Owner != "" {
					require.Len(t, project.Members, 1)
					assert.Equal(t, tt.req.Owner, project.Members[0].UserID)
					assert.Equal(t, types.ProjectRoleOwner, project.Members[0].Role)
					assert.Equal(t, tt.req.Owner, project.Members[0].AddedBy)
				}

				// Verify budget
				if tt.req.Budget != nil {
					assert.NotNil(t, project.Budget)
					assert.Equal(t, tt.req.Budget.TotalBudget, project.Budget.TotalBudget)
					assert.Equal(t, 0.0, project.Budget.SpentAmount)
					if tt.req.Budget.MonthlyLimit != nil {
						assert.Equal(t, *tt.req.Budget.MonthlyLimit, *project.Budget.MonthlyLimit)
					}
					if tt.req.Budget.DailyLimit != nil {
						assert.Equal(t, *tt.req.Budget.DailyLimit, *project.Budget.DailyLimit)
					}
				}
			}
		})
	}
}

func TestManager_CreateProject_DuplicateName(t *testing.T) {
	manager := setupTestManager(t)
	defer teardownTestManager(manager)

	ctx := context.Background()
	
	// Create first project
	req := &CreateProjectRequest{
		Name:        "Duplicate Test",
		Description: "First project",
		Owner:       "user1",
	}
	
	project1, err := manager.CreateProject(ctx, req)
	require.NoError(t, err)
	require.NotNil(t, project1)

	// Try to create second project with same name
	req2 := &CreateProjectRequest{
		Name:        "Duplicate Test",
		Description: "Second project",
		Owner:       "user2",
	}
	
	project2, err := manager.CreateProject(ctx, req2)
	assert.Error(t, err)
	assert.Nil(t, project2)
	assert.Contains(t, err.Error(), "already exists")
}

func TestManager_GetProject(t *testing.T) {
	manager := setupTestManager(t)
	defer teardownTestManager(manager)

	ctx := context.Background()
	
	// Create test project
	req := &CreateProjectRequest{
		Name:        "Test Project",
		Description: "Test project for retrieval",
		Owner:       "test-user",
		Tags: map[string]string{
			"type": "test",
		},
	}
	
	createdProject, err := manager.CreateProject(ctx, req)
	require.NoError(t, err)

	tests := []struct {
		name      string
		projectID string
		wantErr   bool
		wantName  string
	}{
		{
			name:      "get existing project",
			projectID: createdProject.ID,
			wantErr:   false,
			wantName:  "Test Project",
		},
		{
			name:      "get non-existent project",
			projectID: uuid.New().String(),
			wantErr:   true,
		},
		{
			name:      "get project with empty ID",
			projectID: "",
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			project, err := manager.GetProject(ctx, tt.projectID)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, project)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, project)
				assert.Equal(t, tt.wantName, project.Name)
				assert.Equal(t, tt.projectID, project.ID)
				
				// Verify it's a copy (changes don't affect original)
				project.Name = "Modified"
				retrievedAgain, err := manager.GetProject(ctx, tt.projectID)
				require.NoError(t, err)
				assert.Equal(t, tt.wantName, retrievedAgain.Name)
			}
		})
	}
}

func TestManager_GetProjectByName(t *testing.T) {
	manager := setupTestManager(t)
	defer teardownTestManager(manager)

	ctx := context.Background()
	
	// Create test projects
	req1 := &CreateProjectRequest{
		Name:        "Project Alpha",
		Description: "First test project",
		Owner:       "user1",
	}
	
	req2 := &CreateProjectRequest{
		Name:        "Project Beta",
		Description: "Second test project", 
		Owner:       "user2",
	}
	
	project1, err := manager.CreateProject(ctx, req1)
	require.NoError(t, err)
	
	project2, err := manager.CreateProject(ctx, req2)
	require.NoError(t, err)

	tests := []struct {
		name        string
		projectName string
		wantErr     bool
		wantID      string
	}{
		{
			name:        "get existing project by name",
			projectName: "Project Alpha",
			wantErr:     false,
			wantID:      project1.ID,
		},
		{
			name:        "get second project by name",
			projectName: "Project Beta",
			wantErr:     false,
			wantID:      project2.ID,
		},
		{
			name:        "get non-existent project by name",
			projectName: "Non-existent Project",
			wantErr:     true,
		},
		{
			name:        "get project with empty name",
			projectName: "",
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			project, err := manager.GetProjectByName(ctx, tt.projectName)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, project)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, project)
				assert.Equal(t, tt.projectName, project.Name)
				assert.Equal(t, tt.wantID, project.ID)
			}
		})
	}
}

func TestManager_ListProjects(t *testing.T) {
	manager := setupTestManager(t)
	defer teardownTestManager(manager)

	ctx := context.Background()
	
	// Create test projects
	projects := []*CreateProjectRequest{
		{
			Name:        "Active Project 1",
			Description: "First active project",
			Owner:       "user1",
			Tags: map[string]string{
				"department": "research",
				"priority":   "high",
			},
		},
		{
			Name:        "Active Project 2",
			Description: "Second active project",
			Owner:       "user2",
			Tags: map[string]string{
				"department": "engineering",
				"priority":   "medium",
			},
		},
		{
			Name:        "User1 Second Project",
			Description: "Another project by user1",
			Owner:       "user1",
			Tags: map[string]string{
				"department": "research",
				"priority":   "low",
			},
		},
	}

	createdProjects := make([]*types.Project, len(projects))
	for i, req := range projects {
		project, err := manager.CreateProject(ctx, req)
		require.NoError(t, err)
		createdProjects[i] = project
	}

	// Update one project to archived status
	archiveStatus := types.ProjectStatusArchived
	_, err := manager.UpdateProject(ctx, createdProjects[1].ID, &UpdateProjectRequest{
		Status: &archiveStatus,
	})
	require.NoError(t, err)

	tests := []struct {
		name        string
		filter      *ProjectFilter
		wantCount   int
		wantNames   []string
	}{
		{
			name:      "list all projects",
			filter:    nil,
			wantCount: 3,
			wantNames: []string{"Active Project 1", "Active Project 2", "User1 Second Project"},
		},
		{
			name: "filter by owner",
			filter: &ProjectFilter{
				Owner: "user1",
			},
			wantCount: 2,
			wantNames: []string{"Active Project 1", "User1 Second Project"},
		},
		{
			name: "filter by status",
			filter: &ProjectFilter{
				Status: func() *types.ProjectStatus { s := types.ProjectStatusActive; return &s }(),
			},
			wantCount: 2,
			wantNames: []string{"Active Project 1", "User1 Second Project"},
		},
		{
			name: "filter by archived status",
			filter: &ProjectFilter{
				Status: &archiveStatus,
			},
			wantCount: 1,
			wantNames: []string{"Active Project 2"},
		},
		{
			name: "filter by tags",
			filter: &ProjectFilter{
				Tags: map[string]string{
					"department": "research",
				},
			},
			wantCount: 2,
			wantNames: []string{"Active Project 1", "User1 Second Project"},
		},
		{
			name: "filter by multiple tags",
			filter: &ProjectFilter{
				Tags: map[string]string{
					"department": "research",
					"priority":   "high",
				},
			},
			wantCount: 1,
			wantNames: []string{"Active Project 1"},
		},
		{
			name: "filter with no matches",
			filter: &ProjectFilter{
				Owner: "non-existent-user",
			},
			wantCount: 0,
			wantNames: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			projects, err := manager.ListProjects(ctx, tt.filter)
			assert.NoError(t, err)
			assert.Len(t, projects, tt.wantCount)

			actualNames := make([]string, len(projects))
			for i, project := range projects {
				actualNames[i] = project.Name
			}

			for _, expectedName := range tt.wantNames {
				assert.Contains(t, actualNames, expectedName)
			}
		})
	}
}

func TestManager_UpdateProject(t *testing.T) {
	manager := setupTestManager(t)
	defer teardownTestManager(manager)

	ctx := context.Background()
	
	// Create test project
	req := &CreateProjectRequest{
		Name:        "Original Project",
		Description: "Original description",
		Owner:       "test-user",
		Tags: map[string]string{
			"version": "1.0",
		},
	}
	
	project, err := manager.CreateProject(ctx, req)
	require.NoError(t, err)

	tests := []struct {
		name      string
		projectID string
		updateReq *UpdateProjectRequest
		wantErr   bool
		errMsg    string
	}{
		{
			name:      "update project name",
			projectID: project.ID,
			updateReq: &UpdateProjectRequest{
				Name: stringPtr("Updated Project Name"),
			},
			wantErr: false,
		},
		{
			name:      "update project description",
			projectID: project.ID,
			updateReq: &UpdateProjectRequest{
				Description: stringPtr("Updated description"),
			},
			wantErr: false,
		},
		{
			name:      "update project status",
			projectID: project.ID,
			updateReq: &UpdateProjectRequest{
				Status: func() *types.ProjectStatus { s := types.ProjectStatusPaused; return &s }(),
			},
			wantErr: false,
		},
		{
			name:      "update project tags",
			projectID: project.ID,
			updateReq: &UpdateProjectRequest{
				Tags: map[string]string{
					"version": "2.0",
					"updated": "true",
				},
			},
			wantErr: false,
		},
		{
			name:      "update non-existent project",
			projectID: uuid.New().String(),
			updateReq: &UpdateProjectRequest{
				Name: stringPtr("Should Fail"),
			},
			wantErr: true,
			errMsg:  "not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			updatedProject, err := manager.UpdateProject(ctx, tt.projectID, tt.updateReq)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, updatedProject)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, updatedProject)
				
				// Verify updates
				if tt.updateReq.Name != nil {
					assert.Equal(t, *tt.updateReq.Name, updatedProject.Name)
				}
				if tt.updateReq.Description != nil {
					assert.Equal(t, *tt.updateReq.Description, updatedProject.Description)
				}
				if tt.updateReq.Status != nil {
					assert.Equal(t, *tt.updateReq.Status, updatedProject.Status)
				}
				if tt.updateReq.Tags != nil {
					assert.Equal(t, tt.updateReq.Tags, updatedProject.Tags)
				}

				// Verify UpdatedAt was changed (or at least not before the original)
				assert.True(t, updatedProject.UpdatedAt.After(project.UpdatedAt) || updatedProject.UpdatedAt.Equal(project.UpdatedAt))
			}
		})
	}
}

func TestManager_DeleteProject(t *testing.T) {
	manager := setupTestManager(t)
	defer teardownTestManager(manager)

	ctx := context.Background()
	
	// Create test projects
	req1 := &CreateProjectRequest{
		Name:        "Project to Delete",
		Description: "This project will be deleted",
		Owner:       "test-user",
	}
	
	req2 := &CreateProjectRequest{
		Name:        "Project to Keep",
		Description: "This project will remain",
		Owner:       "test-user",
	}
	
	projectToDelete, err := manager.CreateProject(ctx, req1)
	require.NoError(t, err)
	
	projectToKeep, err := manager.CreateProject(ctx, req2)
	require.NoError(t, err)

	tests := []struct {
		name      string
		projectID string
		wantErr   bool
	}{
		{
			name:      "delete existing project",
			projectID: projectToDelete.ID,
			wantErr:   false,
		},
		{
			name:      "delete non-existent project",
			projectID: uuid.New().String(),
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := manager.DeleteProject(ctx, tt.projectID)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				
				// Verify project is deleted
				_, err := manager.GetProject(ctx, tt.projectID)
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "not found")
			}
		})
	}

	// Verify other project still exists
	remainingProject, err := manager.GetProject(ctx, projectToKeep.ID)
	assert.NoError(t, err)
	assert.NotNil(t, remainingProject)
	assert.Equal(t, "Project to Keep", remainingProject.Name)
}

// Helper functions
func setupTestManager(t *testing.T) *Manager {
	t.Helper()
	
	// Create temporary directory for testing
	tempDir, err := os.MkdirTemp("", "cws-project-test-*")
	require.NoError(t, err)
	t.Cleanup(func() {
		os.RemoveAll(tempDir)
	})

	// Mock home directory
	originalHome := os.Getenv("HOME")
	t.Cleanup(func() {
		os.Setenv("HOME", originalHome)
	})
	os.Setenv("HOME", tempDir)

	manager, err := NewManager()
	require.NoError(t, err)
	require.NotNil(t, manager)
	
	return manager
}

func teardownTestManager(manager *Manager) {
	if manager != nil {
		manager.Close()
	}
}

func stringPtr(s string) *string {
	return &s
}

func floatPtr(f float64) *float64 {
	return &f
}

func generateLongString(length int) string {
	result := make([]byte, length)
	for i := range result {
		result[i] = 'a'
	}
	return string(result)
}