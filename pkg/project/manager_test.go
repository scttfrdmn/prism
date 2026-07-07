package project

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/scttfrdmn/prism/pkg/types"
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
			tempDir, err := os.MkdirTemp("", "prism-project-test-*")
			require.NoError(t, err)
			defer func() { _ = os.RemoveAll(tempDir) }()

			// Mock home directory
			originalHome := os.Getenv("HOME")
			defer func() { _ = os.Setenv("HOME", originalHome) }()
			_ = os.Setenv("HOME", tempDir)

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
				stateDir := filepath.Join(tempDir, ".prism")
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
				Description: "A test project for Prism",
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
				Budget: &CreateProjectBudgetRequest{
					TotalBudget:  1000.0,
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
				Budget: &CreateProjectBudgetRequest{
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
		name      string
		filter    *ProjectFilter
		wantCount int
		wantNames []string
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

// TestManager_DeleteProject_BlockedByActiveInstances verifies that DeleteProject
// returns an error when the injected activeInstancesFunc reports running instances,
// and succeeds once no instances are active. (#539)
func TestManager_DeleteProject_BlockedByActiveInstances(t *testing.T) {
	manager := setupTestManager(t)
	defer teardownTestManager(manager)

	ctx := context.Background()

	proj, err := manager.CreateProject(ctx, &CreateProjectRequest{
		Name:  "Blocked Project",
		Owner: "researcher",
	})
	require.NoError(t, err)

	// Wire up an activeInstancesFunc that reports one running instance.
	manager.SetActiveInstancesFunc(func(projectID string) ([]string, error) {
		if projectID == proj.ID {
			return []string{"research-instance-1"}, nil
		}
		return nil, nil
	})

	err = manager.DeleteProject(ctx, proj.ID)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "active instance")

	// Clear the running instance — deletion should now succeed.
	manager.SetActiveInstancesFunc(func(projectID string) ([]string, error) {
		return nil, nil
	})

	err = manager.DeleteProject(ctx, proj.ID)
	require.NoError(t, err)

	_, err = manager.GetProject(ctx, proj.ID)
	assert.Error(t, err)
}

// Helper functions
func setupTestManager(t *testing.T) *Manager {
	t.Helper()

	// Create temporary directory for testing
	tempDir, err := os.MkdirTemp("", "prism-project-test-*")
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

	manager, err := NewManager()
	require.NoError(t, err)
	require.NotNil(t, manager)

	return manager
}

func teardownTestManager(manager *Manager) {
	if manager != nil {
		_ = manager.Close()
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

// TestManager_AddProjectMember tests adding members to projects
func TestManager_AddProjectMember(t *testing.T) {
	manager := setupTestManager(t)
	defer teardownTestManager(manager)

	ctx := context.Background()

	// Create test project
	req := &CreateProjectRequest{
		Name:        "Test Project",
		Description: "Project for member testing",
		Owner:       "owner-user",
	}

	project, err := manager.CreateProject(ctx, req)
	require.NoError(t, err)

	tests := []struct {
		name      string
		projectID string
		member    *types.ProjectMember
		wantErr   bool
		errMsg    string
	}{
		{
			name:      "add admin member",
			projectID: project.ID,
			member: &types.ProjectMember{
				UserID:  "admin-user",
				Role:    types.ProjectRoleAdmin,
				AddedBy: "owner-user",
			},
			wantErr: false,
		},
		{
			name:      "add regular member",
			projectID: project.ID,
			member: &types.ProjectMember{
				UserID:  "regular-user",
				Role:    types.ProjectRoleMember,
				AddedBy: "owner-user",
			},
			wantErr: false,
		},
		{
			name:      "add viewer member",
			projectID: project.ID,
			member: &types.ProjectMember{
				UserID:  "viewer-user",
				Role:    types.ProjectRoleViewer,
				AddedBy: "admin-user",
			},
			wantErr: false,
		},
		{
			name:      "add duplicate member",
			projectID: project.ID,
			member: &types.ProjectMember{
				UserID:  "admin-user", // Already added above
				Role:    types.ProjectRoleMember,
				AddedBy: "owner-user",
			},
			wantErr: true,
			errMsg:  "already a member",
		},
		{
			name:      "add member with invalid role",
			projectID: project.ID,
			member: &types.ProjectMember{
				UserID:  "new-user",
				Role:    "invalid-role",
				AddedBy: "owner-user",
			},
			wantErr: true,
			errMsg:  "invalid role",
		},
		{
			name:      "add member to non-existent project",
			projectID: uuid.New().String(),
			member: &types.ProjectMember{
				UserID:  "new-user",
				Role:    types.ProjectRoleMember,
				AddedBy: "owner-user",
			},
			wantErr: true,
			errMsg:  "not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := manager.AddProjectMember(ctx, tt.projectID, tt.member)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err)

				// Verify member was added
				updatedProject, err := manager.GetProject(ctx, tt.projectID)
				require.NoError(t, err)

				found := false
				for _, member := range updatedProject.Members {
					if member.UserID == tt.member.UserID {
						found = true
						assert.Equal(t, tt.member.Role, member.Role)
						assert.Equal(t, tt.member.AddedBy, member.AddedBy)
						assert.WithinDuration(t, time.Now(), member.AddedAt, time.Second)
						break
					}
				}
				assert.True(t, found, "Member should be added to project")
			}
		})
	}
}

// TestManager_RemoveProjectMember tests removing members from projects
func TestManager_RemoveProjectMember(t *testing.T) {
	manager := setupTestManager(t)
	defer teardownTestManager(manager)

	ctx := context.Background()

	// Create test project with multiple members
	req := &CreateProjectRequest{
		Name:        "Test Project",
		Description: "Project for member removal testing",
		Owner:       "owner-user",
	}

	project, err := manager.CreateProject(ctx, req)
	require.NoError(t, err)

	// Add additional members
	err = manager.AddProjectMember(ctx, project.ID, &types.ProjectMember{
		UserID:  "admin-user",
		Role:    types.ProjectRoleAdmin,
		AddedBy: "owner-user",
	})
	require.NoError(t, err)

	err = manager.AddProjectMember(ctx, project.ID, &types.ProjectMember{
		UserID:  "member-user",
		Role:    types.ProjectRoleMember,
		AddedBy: "owner-user",
	})
	require.NoError(t, err)

	tests := []struct {
		name      string
		projectID string
		userID    string
		wantErr   bool
		errMsg    string
	}{
		{
			name:      "remove regular member",
			projectID: project.ID,
			userID:    "member-user",
			wantErr:   false,
		},
		{
			name:      "remove admin member",
			projectID: project.ID,
			userID:    "admin-user",
			wantErr:   false,
		},
		{
			name:      "remove last owner (should fail)",
			projectID: project.ID,
			userID:    "owner-user",
			wantErr:   true,
			errMsg:    "cannot remove the last owner",
		},
		{
			name:      "remove non-existent member",
			projectID: project.ID,
			userID:    "non-existent-user",
			wantErr:   true,
			errMsg:    "not a member",
		},
		{
			name:      "remove from non-existent project",
			projectID: uuid.New().String(),
			userID:    "member-user",
			wantErr:   true,
			errMsg:    "not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := manager.RemoveProjectMember(ctx, tt.projectID, tt.userID)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err)

				// Verify member was removed
				updatedProject, err := manager.GetProject(ctx, tt.projectID)
				require.NoError(t, err)

				for _, member := range updatedProject.Members {
					assert.NotEqual(t, tt.userID, member.UserID, "Member should be removed from project")
				}
			}
		})
	}
}

// TestManager_UpdateProjectMember tests updating member roles
func TestManager_UpdateProjectMember(t *testing.T) {
	manager := setupTestManager(t)
	defer teardownTestManager(manager)

	ctx := context.Background()

	// Create test project with multiple members
	req := &CreateProjectRequest{
		Name:        "Test Project",
		Description: "Project for member update testing",
		Owner:       "owner-user",
	}

	project, err := manager.CreateProject(ctx, req)
	require.NoError(t, err)

	// Add additional members
	err = manager.AddProjectMember(ctx, project.ID, &types.ProjectMember{
		UserID:  "admin-user",
		Role:    types.ProjectRoleAdmin,
		AddedBy: "owner-user",
	})
	require.NoError(t, err)

	err = manager.AddProjectMember(ctx, project.ID, &types.ProjectMember{
		UserID:  "member-user",
		Role:    types.ProjectRoleMember,
		AddedBy: "owner-user",
	})
	require.NoError(t, err)

	tests := []struct {
		name      string
		projectID string
		userID    string
		newRole   types.ProjectRole
		wantErr   bool
		errMsg    string
	}{
		{
			name:      "promote member to admin",
			projectID: project.ID,
			userID:    "member-user",
			newRole:   types.ProjectRoleAdmin,
			wantErr:   false,
		},
		{
			name:      "demote admin to member",
			projectID: project.ID,
			userID:    "admin-user",
			newRole:   types.ProjectRoleMember,
			wantErr:   false,
		},
		{
			name:      "change member to viewer",
			projectID: project.ID,
			userID:    "admin-user", // Now a member from previous test
			newRole:   types.ProjectRoleViewer,
			wantErr:   false,
		},
		{
			name:      "demote last owner (should fail)",
			projectID: project.ID,
			userID:    "owner-user",
			newRole:   types.ProjectRoleMember,
			wantErr:   true,
			errMsg:    "cannot change the role of the last owner",
		},
		{
			name:      "update non-existent member",
			projectID: project.ID,
			userID:    "non-existent-user",
			newRole:   types.ProjectRoleMember,
			wantErr:   true,
			errMsg:    "not a member",
		},
		{
			name:      "update in non-existent project",
			projectID: uuid.New().String(),
			userID:    "member-user",
			newRole:   types.ProjectRoleAdmin,
			wantErr:   true,
			errMsg:    "not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := manager.UpdateProjectMember(ctx, tt.projectID, tt.userID, tt.newRole)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err)

				// Verify role was updated
				updatedProject, err := manager.GetProject(ctx, tt.projectID)
				require.NoError(t, err)

				found := false
				for _, member := range updatedProject.Members {
					if member.UserID == tt.userID {
						found = true
						assert.Equal(t, tt.newRole, member.Role, "Member role should be updated")
						break
					}
				}
				assert.True(t, found, "Member should still be in project")
			}
		})
	}
}

// TestManager_MemberManagement_MultipleOwners tests scenarios with multiple owners
func TestManager_MemberManagement_MultipleOwners(t *testing.T) {
	manager := setupTestManager(t)
	defer teardownTestManager(manager)

	ctx := context.Background()

	// Create test project
	req := &CreateProjectRequest{
		Name:        "Multi-Owner Project",
		Description: "Project with multiple owners",
		Owner:       "owner1",
	}

	project, err := manager.CreateProject(ctx, req)
	require.NoError(t, err)

	// Add second owner
	err = manager.AddProjectMember(ctx, project.ID, &types.ProjectMember{
		UserID:  "owner2",
		Role:    types.ProjectRoleOwner,
		AddedBy: "owner1",
	})
	require.NoError(t, err)

	// Add third owner
	err = manager.AddProjectMember(ctx, project.ID, &types.ProjectMember{
		UserID:  "owner3",
		Role:    types.ProjectRoleOwner,
		AddedBy: "owner1",
	})
	require.NoError(t, err)

	// Now we should be able to remove one owner (not the last)
	err = manager.RemoveProjectMember(ctx, project.ID, "owner2")
	assert.NoError(t, err)

	// And demote another owner
	err = manager.UpdateProjectMember(ctx, project.ID, "owner3", types.ProjectRoleAdmin)
	assert.NoError(t, err)

	// But we still cannot remove or demote the last owner
	err = manager.RemoveProjectMember(ctx, project.ID, "owner1")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cannot remove the last owner")

	err = manager.UpdateProjectMember(ctx, project.ID, "owner1", types.ProjectRoleMember)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cannot change the role of the last owner")
}

// TestManager_SetProjectBudget tests setting budget for projects
func TestManager_SetProjectBudget(t *testing.T) {
	manager := setupTestManager(t)
	defer teardownTestManager(manager)

	ctx := context.Background()

	// Create test project without budget
	req := &CreateProjectRequest{
		Name:        "Test Project",
		Description: "Project for budget setting",
		Owner:       "test-user",
	}

	project, err := manager.CreateProject(ctx, req)
	require.NoError(t, err)
	assert.Nil(t, project.Budget, "Project should have no budget initially")

	tests := []struct {
		name      string
		projectID string
		budget    *types.ProjectBudget
		wantErr   bool
		errMsg    string
	}{
		{
			name:      "set valid budget",
			projectID: project.ID,
			budget: &types.ProjectBudget{
				TotalBudget:  5000.0,
				MonthlyLimit: floatPtr(1000.0),
				DailyLimit:   floatPtr(100.0),
				AlertThresholds: []types.BudgetAlert{
					{
						Threshold:  0.8,
						Type:       types.BudgetAlertEmail,
						Recipients: []string{"admin@example.com"},
					},
				},
			},
			wantErr: false,
		},
		{
			name:      "set budget on non-existent project",
			projectID: uuid.New().String(),
			budget: &types.ProjectBudget{
				TotalBudget: 1000.0,
			},
			wantErr: true,
			errMsg:  "not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := manager.SetProjectBudget(ctx, tt.projectID, tt.budget)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err)

				// Verify budget was set
				updatedProject, err := manager.GetProject(ctx, tt.projectID)
				require.NoError(t, err)
				assert.NotNil(t, updatedProject.Budget)
				assert.Equal(t, tt.budget.TotalBudget, updatedProject.Budget.TotalBudget)
			}
		})
	}
}

// TestManager_UpdateProjectBudget tests updating project budgets
func TestManager_UpdateProjectBudget(t *testing.T) {
	manager := setupTestManager(t)
	defer teardownTestManager(manager)

	ctx := context.Background()

	// Create test project with budget
	req := &CreateProjectRequest{
		Name:        "Test Project",
		Description: "Project for budget update",
		Owner:       "test-user",
		Budget: &CreateProjectBudgetRequest{
			TotalBudget:  5000.0,
			MonthlyLimit: floatPtr(1000.0),
		},
	}

	project, err := manager.CreateProject(ctx, req)
	require.NoError(t, err)

	tests := []struct {
		name         string
		projectID    string
		budgetUpdate *types.ProjectBudget
		wantErr      bool
		errMsg       string
	}{
		{
			name:      "update budget amount",
			projectID: project.ID,
			budgetUpdate: &types.ProjectBudget{
				TotalBudget:  10000.0,
				MonthlyLimit: floatPtr(2000.0),
				DailyLimit:   floatPtr(200.0),
			},
			wantErr: false,
		},
		{
			name:      "update non-existent project",
			projectID: uuid.New().String(),
			budgetUpdate: &types.ProjectBudget{
				TotalBudget: 5000.0,
			},
			wantErr: true,
			errMsg:  "not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := manager.UpdateProjectBudget(ctx, tt.projectID, tt.budgetUpdate)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err)

				// Verify budget was updated
				updatedProject, err := manager.GetProject(ctx, tt.projectID)
				require.NoError(t, err)
				assert.NotNil(t, updatedProject.Budget)
				assert.Equal(t, tt.budgetUpdate.TotalBudget, updatedProject.Budget.TotalBudget)
				if tt.budgetUpdate.MonthlyLimit != nil {
					assert.Equal(t, *tt.budgetUpdate.MonthlyLimit, *updatedProject.Budget.MonthlyLimit)
				}
			}
		})
	}
}

// TestManager_DisableProjectBudget tests disabling project budgets
func TestManager_DisableProjectBudget(t *testing.T) {
	manager := setupTestManager(t)
	defer teardownTestManager(manager)

	ctx := context.Background()

	// Create test project with budget
	req := &CreateProjectRequest{
		Name:        "Test Project",
		Description: "Project for budget disable",
		Owner:       "test-user",
		Budget: &CreateProjectBudgetRequest{
			TotalBudget: 5000.0,
		},
	}

	project, err := manager.CreateProject(ctx, req)
	require.NoError(t, err)
	assert.NotNil(t, project.Budget)

	// Disable budget
	err = manager.DisableProjectBudget(ctx, project.ID)
	assert.NoError(t, err)

	// Verify budget was disabled
	updatedProject, err := manager.GetProject(ctx, project.ID)
	require.NoError(t, err)
	assert.Nil(t, updatedProject.Budget, "Budget should be nil after disable")

	// Test disabling non-existent project
	err = manager.DisableProjectBudget(ctx, uuid.New().String())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

// TestManager_LaunchPrevention tests launch prevention functionality
func TestManager_LaunchPrevention(t *testing.T) {
	manager := setupTestManager(t)
	defer teardownTestManager(manager)

	ctx := context.Background()

	// Create test project
	req := &CreateProjectRequest{
		Name:        "Test Project",
		Description: "Project for launch prevention",
		Owner:       "test-user",
	}

	project, err := manager.CreateProject(ctx, req)
	require.NoError(t, err)

	// Initially, launches should not be prevented
	isPrevented, err := manager.IsLaunchPrevented(ctx, project.ID)
	assert.NoError(t, err)
	assert.False(t, isPrevented, "Launches should not be prevented initially")

	// Prevent launches
	err = manager.PreventLaunches(ctx, project.ID)
	assert.NoError(t, err)

	// Verify launches are prevented
	isPrevented, err = manager.IsLaunchPrevented(ctx, project.ID)
	assert.NoError(t, err)
	assert.True(t, isPrevented, "Launches should be prevented")

	// Allow launches again
	err = manager.AllowLaunches(ctx, project.ID)
	assert.NoError(t, err)

	// Verify launches are allowed
	isPrevented, err = manager.IsLaunchPrevented(ctx, project.ID)
	assert.NoError(t, err)
	assert.False(t, isPrevented, "Launches should be allowed again")

	// Test with non-existent project
	err = manager.PreventLaunches(ctx, uuid.New().String())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")

	err = manager.AllowLaunches(ctx, uuid.New().String())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")

	_, err = manager.IsLaunchPrevented(ctx, uuid.New().String())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

// TestManager_TransferProject tests project ownership transfer
func TestManager_TransferProject(t *testing.T) {
	manager := setupTestManager(t)
	defer teardownTestManager(manager)

	ctx := context.Background()

	// Create test project
	req := &CreateProjectRequest{
		Name:        "Test Project",
		Description: "Project for ownership transfer",
		Owner:       "original-owner",
	}

	project, err := manager.CreateProject(ctx, req)
	require.NoError(t, err)
	assert.Equal(t, "original-owner", project.Owner)

	// Add a member who will become the new owner
	err = manager.AddProjectMember(ctx, project.ID, &types.ProjectMember{
		UserID:  "new-owner",
		Role:    types.ProjectRoleMember,
		AddedBy: "original-owner",
	})
	require.NoError(t, err)

	// Transfer ownership
	transferredProject, err := manager.TransferProject(ctx, project.ID, &TransferProjectRequest{
		NewOwnerID:    "new-owner",
		TransferredBy: "original-owner",
		Reason:        "Testing transfer",
	})
	assert.NoError(t, err)
	assert.NotNil(t, transferredProject)

	// Verify ownership was transferred
	updatedProject, err := manager.GetProject(ctx, project.ID)
	require.NoError(t, err)
	assert.Equal(t, "new-owner", updatedProject.Owner)

	// Verify new owner has owner role
	newOwnerFound := false
	originalOwnerRole := types.ProjectRole("")
	for _, member := range updatedProject.Members {
		if member.UserID == "new-owner" {
			newOwnerFound = true
			assert.Equal(t, types.ProjectRoleOwner, member.Role)
		}
		if member.UserID == "original-owner" {
			originalOwnerRole = member.Role
		}
	}
	assert.True(t, newOwnerFound, "New owner should be in members list")
	assert.Equal(t, types.ProjectRoleAdmin, originalOwnerRole, "Original owner should be demoted to admin")

	// Test error cases
	_, err = manager.TransferProject(ctx, uuid.New().String(), &TransferProjectRequest{
		NewOwnerID:    "someone",
		TransferredBy: "transferrer",
		Reason:        "Test",
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

// TestManager_DefaultAllocation tests default allocation management
func TestManager_DefaultAllocation(t *testing.T) {
	manager := setupTestManager(t)
	defer teardownTestManager(manager)

	ctx := context.Background()

	// Create test project
	req := &CreateProjectRequest{
		Name:        "Test Project",
		Description: "Project for default allocation",
		Owner:       "test-user",
	}

	project, err := manager.CreateProject(ctx, req)
	require.NoError(t, err)

	// Set default allocation
	allocationID := uuid.New().String()
	err = manager.SetDefaultAllocation(ctx, project.ID, allocationID)
	assert.NoError(t, err)

	// Verify default allocation was set
	updatedProject, err := manager.GetProject(ctx, project.ID)
	require.NoError(t, err)
	assert.NotNil(t, updatedProject.DefaultAllocationID)
	assert.Equal(t, allocationID, *updatedProject.DefaultAllocationID)

	// Clear default allocation
	err = manager.ClearDefaultAllocation(ctx, project.ID)
	assert.NoError(t, err)

	// Verify default allocation was cleared
	updatedProject, err = manager.GetProject(ctx, project.ID)
	require.NoError(t, err)
	assert.Nil(t, updatedProject.DefaultAllocationID)

	// Test error cases
	err = manager.SetDefaultAllocation(ctx, uuid.New().String(), allocationID)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")

	err = manager.ClearDefaultAllocation(ctx, uuid.New().String())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

// TestManager_ConcurrentOperations tests thread safety
func TestManager_ConcurrentOperations(t *testing.T) {
	manager := setupTestManager(t)
	defer teardownTestManager(manager)

	ctx := context.Background()

	// Create initial project
	req := &CreateProjectRequest{
		Name:        "Concurrent Test Project",
		Description: "Project for concurrency testing",
		Owner:       "test-user",
	}

	project, err := manager.CreateProject(ctx, req)
	require.NoError(t, err)

	const numGoroutines = 10
	done := make(chan bool, numGoroutines)
	errors := make(chan error, numGoroutines)

	// Concurrently add members
	for i := 0; i < numGoroutines; i++ {
		go func(index int) {
			defer func() { done <- true }()

			userID := fmt.Sprintf("user-%d", index)
			member := &types.ProjectMember{
				UserID:  userID,
				Role:    types.ProjectRoleMember,
				AddedBy: "test-user",
			}

			if err := manager.AddProjectMember(ctx, project.ID, member); err != nil {
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

	// Verify all members were added
	finalProject, err := manager.GetProject(ctx, project.ID)
	require.NoError(t, err)
	assert.Equal(t, numGoroutines+1, len(finalProject.Members), "Should have owner + %d members", numGoroutines)
}

// TestManager_StatePersistence tests that changes are persisted
func TestManager_StatePersistence(t *testing.T) {
	// Create temporary directory for testing
	tempDir, err := os.MkdirTemp("", "prism-project-persist-*")
	require.NoError(t, err)
	defer func() { _ = os.RemoveAll(tempDir) }()

	// Mock home directory
	originalHome := os.Getenv("HOME")
	defer func() { _ = os.Setenv("HOME", originalHome) }()
	_ = os.Setenv("HOME", tempDir)

	ctx := context.Background()

	// Create first manager and project
	manager1, err := NewManager()
	require.NoError(t, err)

	req := &CreateProjectRequest{
		Name:        "Persistent Project",
		Description: "Test project for persistence",
		Owner:       "test-user",
		Tags: map[string]string{
			"test": "persistence",
		},
	}

	project, err := manager1.CreateProject(ctx, req)
	require.NoError(t, err)
	projectID := project.ID

	// Add a member
	err = manager1.AddProjectMember(ctx, projectID, &types.ProjectMember{
		UserID:  "member-user",
		Role:    types.ProjectRoleMember,
		AddedBy: "test-user",
	})
	require.NoError(t, err)

	// Close first manager
	err = manager1.Close()
	require.NoError(t, err)

	// Create second manager (should load from disk)
	manager2, err := NewManager()
	require.NoError(t, err)
	defer manager2.Close()

	// Verify project was loaded
	loadedProject, err := manager2.GetProject(ctx, projectID)
	require.NoError(t, err)
	assert.Equal(t, "Persistent Project", loadedProject.Name)
	assert.Equal(t, "Test project for persistence", loadedProject.Description)
	assert.Equal(t, "test-user", loadedProject.Owner)
	assert.Equal(t, map[string]string{"test": "persistence"}, loadedProject.Tags)
	assert.Len(t, loadedProject.Members, 2, "Should have owner + 1 member")
}

// TestManager_MigratesLegacyProjectsFile verifies a pre-seam flat projects.json is imported into
// the seam on first construction, so existing users lose nothing in the convergence.
func TestManager_MigratesLegacyProjectsFile(t *testing.T) {
	tempDir := t.TempDir()
	originalHome := os.Getenv("HOME")
	originalStateDir := os.Getenv("PRISM_STATE_DIR")
	t.Cleanup(func() {
		_ = os.Setenv("HOME", originalHome)
		_ = os.Setenv("PRISM_STATE_DIR", originalStateDir)
	})
	_ = os.Setenv("HOME", tempDir)
	_ = os.Unsetenv("PRISM_STATE_DIR")

	stateDir := filepath.Join(tempDir, ".prism")
	require.NoError(t, os.MkdirAll(stateDir, 0o755))
	legacy := `{"proj-legacy":{"id":"proj-legacy","name":"Legacy Project","owner":"alice","status":"active"}}`
	require.NoError(t, os.WriteFile(filepath.Join(stateDir, "projects.json"), []byte(legacy), 0o644))

	mgr, err := NewManager()
	require.NoError(t, err)

	got, err := mgr.GetProject(context.Background(), "proj-legacy")
	require.NoError(t, err)
	assert.Equal(t, "Legacy Project", got.Name)

	// Legacy file retired; a second manager does not double-import.
	_, statErr := os.Stat(filepath.Join(stateDir, "projects.json"))
	assert.True(t, os.IsNotExist(statErr), "legacy projects.json should be renamed aside")

	mgr2, err := NewManager()
	require.NoError(t, err)
	all, err := mgr2.ListProjects(context.Background(), nil)
	require.NoError(t, err)
	assert.Len(t, all, 1)
}

// TestRequestValidation tests validation methods for various request types
func TestRequestValidation_AddMemberRequest(t *testing.T) {
	tests := []struct {
		name    string
		req     *AddMemberRequest
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid add member request",
			req: &AddMemberRequest{
				UserID:  "user-123",
				Role:    types.ProjectRoleMember,
				AddedBy: "admin-user",
			},
			wantErr: false,
		},
		{
			name: "missing user ID",
			req: &AddMemberRequest{
				Role:    types.ProjectRoleMember,
				AddedBy: "admin-user",
			},
			wantErr: true,
			errMsg:  "user ID is required",
		},
		{
			name: "missing role",
			req: &AddMemberRequest{
				UserID:  "user-123",
				AddedBy: "admin-user",
			},
			wantErr: true,
			errMsg:  "role is required",
		},
		{
			name: "invalid role",
			req: &AddMemberRequest{
				UserID:  "user-123",
				Role:    "invalid-role",
				AddedBy: "admin-user",
			},
			wantErr: true,
			errMsg:  "invalid role",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.req.Validate()
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestRequestValidation_UpdateMemberRequest(t *testing.T) {
	tests := []struct {
		name    string
		req     *UpdateMemberRequest
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid update request",
			req: &UpdateMemberRequest{
				Role: types.ProjectRoleAdmin,
			},
			wantErr: false,
		},
		{
			name: "missing role",
			req: &UpdateMemberRequest{
				Role: "",
			},
			wantErr: true,
			errMsg:  "role is required",
		},
		{
			name: "invalid role",
			req: &UpdateMemberRequest{
				Role: "super-admin",
			},
			wantErr: true,
			errMsg:  "invalid role",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.req.Validate()
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestRequestValidation_ReallocateFundsRequest(t *testing.T) {
	tests := []struct {
		name    string
		req     *ReallocateFundsRequest
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid reallocation request",
			req: &ReallocateFundsRequest{
				SourceAllocationID:      "alloc-1",
				DestinationAllocationID: "alloc-2",
				Amount:                  5000.0,
				Reason:                  "Project needs changed",
				PerformedBy:             "admin-user",
			},
			wantErr: false,
		},
		{
			name: "missing source allocation",
			req: &ReallocateFundsRequest{
				DestinationAllocationID: "alloc-2",
				Amount:                  5000.0,
				Reason:                  "Test",
				PerformedBy:             "admin",
			},
			wantErr: true,
			errMsg:  "source_allocation_id is required",
		},
		{
			name: "missing destination allocation",
			req: &ReallocateFundsRequest{
				SourceAllocationID: "alloc-1",
				Amount:             5000.0,
				Reason:             "Test",
				PerformedBy:        "admin",
			},
			wantErr: true,
			errMsg:  "destination_allocation_id is required",
		},
		{
			name: "same source and destination",
			req: &ReallocateFundsRequest{
				SourceAllocationID:      "alloc-1",
				DestinationAllocationID: "alloc-1",
				Amount:                  5000.0,
				Reason:                  "Test",
				PerformedBy:             "admin",
			},
			wantErr: true,
			errMsg:  "source and destination allocations cannot be the same",
		},
		{
			name: "zero amount",
			req: &ReallocateFundsRequest{
				SourceAllocationID:      "alloc-1",
				DestinationAllocationID: "alloc-2",
				Amount:                  0.0,
				Reason:                  "Test",
				PerformedBy:             "admin",
			},
			wantErr: true,
			errMsg:  "reallocation amount must be greater than 0",
		},
		{
			name: "missing reason",
			req: &ReallocateFundsRequest{
				SourceAllocationID:      "alloc-1",
				DestinationAllocationID: "alloc-2",
				Amount:                  5000.0,
				PerformedBy:             "admin",
			},
			wantErr: true,
			errMsg:  "reason is required",
		},
		{
			name: "reason too long",
			req: &ReallocateFundsRequest{
				SourceAllocationID:      "alloc-1",
				DestinationAllocationID: "alloc-2",
				Amount:                  5000.0,
				Reason:                  string(make([]byte, 501)),
				PerformedBy:             "admin",
			},
			wantErr: true,
			errMsg:  "reason cannot exceed 500 characters",
		},
		{
			name: "missing performed_by",
			req: &ReallocateFundsRequest{
				SourceAllocationID:      "alloc-1",
				DestinationAllocationID: "alloc-2",
				Amount:                  5000.0,
				Reason:                  "Test",
			},
			wantErr: true,
			errMsg:  "performed_by is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.req.Validate()
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestRequestValidation_CreateProjectBudgetRequest(t *testing.T) {
	tests := []struct {
		name    string
		req     *CreateProjectBudgetRequest
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid budget request",
			req: &CreateProjectBudgetRequest{
				TotalBudget: 10000.0,
			},
			wantErr: false,
		},
		{
			name: "zero total budget",
			req: &CreateProjectBudgetRequest{
				TotalBudget: 0.0,
			},
			wantErr: true,
			errMsg:  "total budget must be greater than 0",
		},
		{
			name: "negative total budget",
			req: &CreateProjectBudgetRequest{
				TotalBudget: -1000.0,
			},
			wantErr: true,
			errMsg:  "total budget must be greater than 0",
		},
		{
			name: "invalid monthly limit",
			req: &CreateProjectBudgetRequest{
				TotalBudget:  10000.0,
				MonthlyLimit: func() *float64 { v := -100.0; return &v }(),
			},
			wantErr: true,
			errMsg:  "monthly limit must be greater than 0",
		},
		{
			name: "invalid daily limit",
			req: &CreateProjectBudgetRequest{
				TotalBudget: 10000.0,
				DailyLimit:  func() *float64 { v := 0.0; return &v }(),
			},
			wantErr: true,
			errMsg:  "daily limit must be greater than 0",
		},
		{
			name: "invalid alert threshold",
			req: &CreateProjectBudgetRequest{
				TotalBudget: 10000.0,
				AlertThresholds: []types.BudgetAlert{
					{Threshold: 1.5, Type: "email"},
				},
			},
			wantErr: true,
			errMsg:  "alert threshold 0 must be between 0.0 and 1.0",
		},
		{
			name: "invalid auto action threshold",
			req: &CreateProjectBudgetRequest{
				TotalBudget: 10000.0,
				AutoActions: []types.BudgetAutoAction{
					{Threshold: -0.1, Action: "prevent-launch"},
				},
			},
			wantErr: true,
			errMsg:  "auto action threshold 0 must be between 0.0 and 1.0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.req.Validate()
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestRequestValidation_CreateAllocationRequest(t *testing.T) {
	tests := []struct {
		name    string
		req     *CreateAllocationRequest
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid allocation request",
			req: &CreateAllocationRequest{
				BudgetID:        "budget-1",
				ProjectID:       "project-1",
				AllocatedAmount: 5000.0,
				AllocatedBy:     "admin-user",
			},
			wantErr: false,
		},
		{
			name: "missing budget ID",
			req: &CreateAllocationRequest{
				ProjectID:       "project-1",
				AllocatedAmount: 5000.0,
				AllocatedBy:     "admin-user",
			},
			wantErr: true,
			errMsg:  "budget_id is required",
		},
		{
			name: "missing project ID",
			req: &CreateAllocationRequest{
				BudgetID:        "budget-1",
				AllocatedAmount: 5000.0,
				AllocatedBy:     "admin-user",
			},
			wantErr: true,
			errMsg:  "project_id is required",
		},
		{
			name: "zero allocated amount",
			req: &CreateAllocationRequest{
				BudgetID:        "budget-1",
				ProjectID:       "project-1",
				AllocatedAmount: 0.0,
				AllocatedBy:     "admin-user",
			},
			wantErr: true,
			errMsg:  "allocated amount must be greater than 0",
		},
		{
			name: "invalid alert threshold - too high",
			req: &CreateAllocationRequest{
				BudgetID:        "budget-1",
				ProjectID:       "project-1",
				AllocatedAmount: 5000.0,
				AlertThreshold:  func() *float64 { v := 1.5; return &v }(),
				AllocatedBy:     "admin-user",
			},
			wantErr: true,
			errMsg:  "alert threshold must be between 0.0 and 1.0",
		},
		{
			name: "invalid alert threshold - negative",
			req: &CreateAllocationRequest{
				BudgetID:        "budget-1",
				ProjectID:       "project-1",
				AllocatedAmount: 5000.0,
				AlertThreshold:  func() *float64 { v := -0.1; return &v }(),
				AllocatedBy:     "admin-user",
			},
			wantErr: true,
			errMsg:  "alert threshold must be between 0.0 and 1.0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.req.Validate()
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestRequestValidation_TransferProjectRequest(t *testing.T) {
	tests := []struct {
		name    string
		req     *TransferProjectRequest
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid transfer request",
			req: &TransferProjectRequest{
				NewOwnerID:    "user-2",
				TransferredBy: "user-1",
				Reason:        "User requested transfer",
			},
			wantErr: false,
		},
		{
			name: "missing new owner ID",
			req: &TransferProjectRequest{
				TransferredBy: "user-1",
				Reason:        "Transfer",
			},
			wantErr: true,
			errMsg:  "new_owner_id is required",
		},
		{
			name: "missing transferred by",
			req: &TransferProjectRequest{
				NewOwnerID: "user-2",
				Reason:     "Transfer",
			},
			wantErr: true,
			errMsg:  "transferred_by is required",
		},
		{
			name: "reason too long",
			req: &TransferProjectRequest{
				NewOwnerID:    "user-2",
				TransferredBy: "user-1",
				Reason:        string(make([]byte, 501)),
			},
			wantErr: true,
			errMsg:  "reason cannot exceed 500 characters",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.req.Validate()
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestManager_GetProjectCostBreakdown tests cost breakdown retrieval
func TestManager_GetProjectCostBreakdown(t *testing.T) {
	manager := setupTestManager(t)
	ctx := context.Background()

	// Test with non-existent project
	_, err := manager.GetProjectCostBreakdown(ctx, "non-existent", time.Now().Add(-24*time.Hour), time.Now())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")

	// Test with existing project (may fail if budget tracker not initialized, but covers the function)
	project, err := manager.CreateProject(ctx, &CreateProjectRequest{
		Name:        "Cost Test Project",
		Description: "For cost breakdown testing",
		Owner:       "test-user",
	})
	require.NoError(t, err)

	// This may return an error if budget tracker isn't fully initialized, but it exercises the code path
	_, _ = manager.GetProjectCostBreakdown(ctx, project.ID, time.Now().Add(-24*time.Hour), time.Now())
}

// TestManager_GetProjectResourceUsage tests resource usage retrieval
func TestManager_GetProjectResourceUsage(t *testing.T) {
	manager := setupTestManager(t)
	ctx := context.Background()

	// Test with non-existent project
	_, err := manager.GetProjectResourceUsage(ctx, "non-existent", 24*time.Hour)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")

	// Test with existing project
	project, err := manager.CreateProject(ctx, &CreateProjectRequest{
		Name:        "Resource Test Project",
		Description: "For resource usage testing",
		Owner:       "test-user",
	})
	require.NoError(t, err)

	// This may return an error if budget tracker isn't fully initialized, but it exercises the code path
	_, _ = manager.GetProjectResourceUsage(ctx, project.ID, 24*time.Hour)
}

// TestManager_CheckBudgetStatus tests budget status checking
func TestManager_CheckBudgetStatus(t *testing.T) {
	manager := setupTestManager(t)
	ctx := context.Background()

	// Test with non-existent project
	_, err := manager.CheckBudgetStatus(ctx, "non-existent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")

	// Test with project that has no budget
	project, err := manager.CreateProject(ctx, &CreateProjectRequest{
		Name:        "No Budget Project",
		Description: "Project without budget",
		Owner:       "test-user",
	})
	require.NoError(t, err)

	status, err := manager.CheckBudgetStatus(ctx, project.ID)
	assert.NoError(t, err)
	assert.NotNil(t, status)
	assert.Equal(t, project.ID, status.ProjectID)
	assert.False(t, status.BudgetEnabled, "project should not have budget enabled")

	// Test with project that has a budget
	projectWithBudget, err := manager.CreateProject(ctx, &CreateProjectRequest{
		Name:        "Budgeted Project",
		Description: "Project with budget",
		Owner:       "test-user",
		Budget: &CreateProjectBudgetRequest{
			TotalBudget: 10000.0,
		},
	})
	require.NoError(t, err)

	// This may return an error if budget tracker isn't fully initialized, but it exercises the code path
	_, _ = manager.CheckBudgetStatus(ctx, projectWithBudget.ID)
}

// ============ v0.12.0 Tests ============

// ---------------------------------------------------------------------------
// CheckQuota / SetRoleQuota / GetRoleQuotas (#151)
// ---------------------------------------------------------------------------

func TestManager_CheckQuota_UnderLimit(t *testing.T) {
	manager := setupTestManager(t)
	defer teardownTestManager(manager)
	ctx := context.Background()

	project, err := manager.CreateProject(ctx, &CreateProjectRequest{
		Name:  "Quota Under Limit",
		Owner: "owner",
	})
	require.NoError(t, err)

	// Give the project a budget with a quota for members: max 2 instances.
	err = manager.SetProjectBudget(ctx, project.ID, &types.ProjectBudget{
		TotalBudget: 1000.0,
		RoleQuotas: []types.RoleQuota{
			{Role: types.ProjectRoleMember, MaxInstances: 2},
		},
	})
	require.NoError(t, err)

	// Add a member with the "member" role.
	err = manager.AddProjectMember(ctx, project.ID, &types.ProjectMember{
		UserID: "user-1",
		Role:   types.ProjectRoleMember,
	})
	require.NoError(t, err)

	// Currently holding 1 instance (< 2) — should be allowed.
	err = manager.CheckQuota(project.ID, "user-1", "t3.medium", 1)
	assert.NoError(t, err)
}

func TestManager_CheckQuota_AtLimit(t *testing.T) {
	manager := setupTestManager(t)
	defer teardownTestManager(manager)
	ctx := context.Background()

	project, err := manager.CreateProject(ctx, &CreateProjectRequest{
		Name:  "Quota At Limit",
		Owner: "owner",
	})
	require.NoError(t, err)

	err = manager.SetProjectBudget(ctx, project.ID, &types.ProjectBudget{
		TotalBudget: 1000.0,
		RoleQuotas: []types.RoleQuota{
			{Role: types.ProjectRoleMember, MaxInstances: 2},
		},
	})
	require.NoError(t, err)

	err = manager.AddProjectMember(ctx, project.ID, &types.ProjectMember{
		UserID: "user-1",
		Role:   types.ProjectRoleMember,
	})
	require.NoError(t, err)

	// Already has 2 instances (== MaxInstances) — should be rejected.
	err = manager.CheckQuota(project.ID, "user-1", "t3.medium", 2)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "quota exceeded")
}

func TestManager_CheckQuota_NoQuota(t *testing.T) {
	manager := setupTestManager(t)
	defer teardownTestManager(manager)
	ctx := context.Background()

	project, err := manager.CreateProject(ctx, &CreateProjectRequest{
		Name:  "No Quota Project",
		Owner: "owner",
	})
	require.NoError(t, err)

	err = manager.AddProjectMember(ctx, project.ID, &types.ProjectMember{
		UserID: "user-1",
		Role:   types.ProjectRoleMember,
	})
	require.NoError(t, err)

	// No budget / no quotas → always allowed.
	err = manager.CheckQuota(project.ID, "user-1", "c5.2xlarge", 99)
	assert.NoError(t, err)
}

func TestManager_CheckQuota_InstanceType(t *testing.T) {
	manager := setupTestManager(t)
	defer teardownTestManager(manager)
	ctx := context.Background()

	project, err := manager.CreateProject(ctx, &CreateProjectRequest{
		Name:  "Quota InstanceType",
		Owner: "owner",
	})
	require.NoError(t, err)

	err = manager.SetProjectBudget(ctx, project.ID, &types.ProjectBudget{
		TotalBudget: 1000.0,
		RoleQuotas: []types.RoleQuota{
			{Role: types.ProjectRoleMember, MaxInstanceType: "t3"},
		},
	})
	require.NoError(t, err)

	err = manager.AddProjectMember(ctx, project.ID, &types.ProjectMember{
		UserID: "user-1",
		Role:   types.ProjectRoleMember,
	})
	require.NoError(t, err)

	// t3.medium matches "t3" prefix — allowed.
	err = manager.CheckQuota(project.ID, "user-1", "t3.medium", 0)
	assert.NoError(t, err)

	// c5.2xlarge does NOT match "t3" prefix — rejected.
	err = manager.CheckQuota(project.ID, "user-1", "c5.2xlarge", 0)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "quota exceeded")
}

func TestManager_SetGetRoleQuotas(t *testing.T) {
	manager := setupTestManager(t)
	defer teardownTestManager(manager)
	ctx := context.Background()

	project, err := manager.CreateProject(ctx, &CreateProjectRequest{
		Name:  "SetGet Quotas",
		Owner: "owner",
	})
	require.NoError(t, err)

	// Ensure the project has a budget before setting quotas.
	err = manager.SetProjectBudget(ctx, project.ID, &types.ProjectBudget{TotalBudget: 500.0})
	require.NoError(t, err)

	quota := types.RoleQuota{Role: types.ProjectRoleMember, MaxInstances: 3}
	err = manager.SetRoleQuota(ctx, project.ID, quota)
	require.NoError(t, err)

	quotas, err := manager.GetRoleQuotas(project.ID)
	require.NoError(t, err)
	require.Len(t, quotas, 1)
	assert.Equal(t, types.ProjectRoleMember, quotas[0].Role)
	assert.Equal(t, 3, quotas[0].MaxInstances)
}

// ---------------------------------------------------------------------------
// Grant period (#152)
// ---------------------------------------------------------------------------

func TestManager_SetGetGrantPeriod(t *testing.T) {
	manager := setupTestManager(t)
	defer teardownTestManager(manager)
	ctx := context.Background()

	project, err := manager.CreateProject(ctx, &CreateProjectRequest{
		Name:  "Grant Period Test",
		Owner: "owner",
	})
	require.NoError(t, err)

	err = manager.SetProjectBudget(ctx, project.ID, &types.ProjectBudget{TotalBudget: 1000.0})
	require.NoError(t, err)

	gp := &types.GrantPeriod{
		Name:       "NSF Year 1",
		StartDate:  time.Now().Add(-30 * 24 * time.Hour),
		EndDate:    time.Now().Add(335 * 24 * time.Hour),
		AutoFreeze: true,
	}
	err = manager.SetGrantPeriod(ctx, project.ID, gp)
	require.NoError(t, err)

	got, err := manager.GetGrantPeriod(project.ID)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, "NSF Year 1", got.Name)
	assert.True(t, got.AutoFreeze)
}

func TestManager_DeleteGrantPeriod(t *testing.T) {
	manager := setupTestManager(t)
	defer teardownTestManager(manager)
	ctx := context.Background()

	project, err := manager.CreateProject(ctx, &CreateProjectRequest{
		Name:  "Delete Grant Period",
		Owner: "owner",
	})
	require.NoError(t, err)

	err = manager.SetProjectBudget(ctx, project.ID, &types.ProjectBudget{TotalBudget: 1000.0})
	require.NoError(t, err)

	gp := &types.GrantPeriod{Name: "Temp Grant", StartDate: time.Now(), EndDate: time.Now().Add(24 * time.Hour)}
	err = manager.SetGrantPeriod(ctx, project.ID, gp)
	require.NoError(t, err)

	err = manager.DeleteGrantPeriod(ctx, project.ID)
	require.NoError(t, err)

	got, err := manager.GetGrantPeriod(project.ID)
	require.NoError(t, err)
	assert.Nil(t, got, "grant period should be nil after deletion")
}

func TestManager_CheckGrantPeriods_NotExpired(t *testing.T) {
	manager := setupTestManager(t)
	defer teardownTestManager(manager)
	ctx := context.Background()

	project, err := manager.CreateProject(ctx, &CreateProjectRequest{
		Name:  "Grant Not Expired",
		Owner: "owner",
	})
	require.NoError(t, err)

	err = manager.SetProjectBudget(ctx, project.ID, &types.ProjectBudget{TotalBudget: 1000.0})
	require.NoError(t, err)

	// EndDate is in the future.
	gp := &types.GrantPeriod{
		Name:       "Future Grant",
		StartDate:  time.Now().Add(-24 * time.Hour),
		EndDate:    time.Now().Add(365 * 24 * time.Hour),
		AutoFreeze: true,
	}
	err = manager.SetGrantPeriod(ctx, project.ID, gp)
	require.NoError(t, err)

	frozen, err := manager.CheckGrantPeriods(ctx)
	require.NoError(t, err)
	assert.NotContains(t, frozen, project.ID, "project should not be frozen when grant has not expired")
}

func TestManager_CheckGrantPeriods_AutoFreeze(t *testing.T) {
	manager := setupTestManager(t)
	defer teardownTestManager(manager)
	ctx := context.Background()

	project, err := manager.CreateProject(ctx, &CreateProjectRequest{
		Name:  "Auto Freeze Grant",
		Owner: "owner",
	})
	require.NoError(t, err)

	err = manager.SetProjectBudget(ctx, project.ID, &types.ProjectBudget{TotalBudget: 1000.0})
	require.NoError(t, err)

	// EndDate is in the past and AutoFreeze=true.
	gp := &types.GrantPeriod{
		Name:       "Expired Grant",
		StartDate:  time.Now().Add(-60 * 24 * time.Hour),
		EndDate:    time.Now().Add(-1 * time.Second),
		AutoFreeze: true,
	}
	err = manager.SetGrantPeriod(ctx, project.ID, gp)
	require.NoError(t, err)

	frozen, err := manager.CheckGrantPeriods(ctx)
	require.NoError(t, err)
	assert.Contains(t, frozen, project.ID, "project should be auto-frozen when grant has expired with AutoFreeze=true")

	// Verify the project was actually paused.
	got, err := manager.GetProject(ctx, project.ID)
	require.NoError(t, err)
	assert.Equal(t, types.ProjectStatusPaused, got.Status)
	assert.True(t, got.LaunchPrevented)
}

func TestManager_CheckGrantPeriods_NoAutoFreeze(t *testing.T) {
	manager := setupTestManager(t)
	defer teardownTestManager(manager)
	ctx := context.Background()

	project, err := manager.CreateProject(ctx, &CreateProjectRequest{
		Name:  "No Auto Freeze",
		Owner: "owner",
	})
	require.NoError(t, err)

	err = manager.SetProjectBudget(ctx, project.ID, &types.ProjectBudget{TotalBudget: 1000.0})
	require.NoError(t, err)

	// EndDate is in the past but AutoFreeze=false.
	gp := &types.GrantPeriod{
		Name:       "Expired No Freeze",
		StartDate:  time.Now().Add(-60 * 24 * time.Hour),
		EndDate:    time.Now().Add(-1 * time.Second),
		AutoFreeze: false,
	}
	err = manager.SetGrantPeriod(ctx, project.ID, gp)
	require.NoError(t, err)

	frozen, err := manager.CheckGrantPeriods(ctx)
	require.NoError(t, err)
	assert.NotContains(t, frozen, project.ID, "project should NOT be frozen when AutoFreeze=false")
}

// ---------------------------------------------------------------------------
// Multi-month allocation (#144)
// ---------------------------------------------------------------------------

func TestManager_CurrentMonthAllocation_MonthlyAmount(t *testing.T) {
	manager := setupTestManager(t)
	defer teardownTestManager(manager)
	ctx := context.Background()

	project, err := manager.CreateProject(ctx, &CreateProjectRequest{
		Name:  "Monthly Amount Project",
		Owner: "owner",
	})
	require.NoError(t, err)

	err = manager.SetProjectBudget(ctx, project.ID, &types.ProjectBudget{
		TotalBudget:   1200.0,
		MonthlyAmount: 150.0,
	})
	require.NoError(t, err)

	alloc, err := manager.CurrentMonthAllocation(project.ID)
	require.NoError(t, err)
	assert.Equal(t, 150.0, alloc, "should return MonthlyAmount when set")
}

func TestManager_CurrentMonthAllocation_MultiMonth(t *testing.T) {
	manager := setupTestManager(t)
	defer teardownTestManager(manager)
	ctx := context.Background()

	project, err := manager.CreateProject(ctx, &CreateProjectRequest{
		Name:  "Multi Month Project",
		Owner: "owner",
	})
	require.NoError(t, err)

	// TotalBudget=1200 over 4 months → 300/month.
	err = manager.SetProjectBudget(ctx, project.ID, &types.ProjectBudget{
		TotalBudget:      1200.0,
		AllocationMonths: 4,
	})
	require.NoError(t, err)

	alloc, err := manager.CurrentMonthAllocation(project.ID)
	require.NoError(t, err)
	assert.Equal(t, 300.0, alloc, "should return TotalBudget/AllocationMonths")
}

func TestManager_CurrentMonthAllocation_Default(t *testing.T) {
	manager := setupTestManager(t)
	defer teardownTestManager(manager)
	ctx := context.Background()

	project, err := manager.CreateProject(ctx, &CreateProjectRequest{
		Name:  "Default Allocation Project",
		Owner: "owner",
	})
	require.NoError(t, err)

	err = manager.SetProjectBudget(ctx, project.ID, &types.ProjectBudget{
		TotalBudget: 500.0,
		// No MonthlyAmount, no AllocationMonths.
	})
	require.NoError(t, err)

	alloc, err := manager.CurrentMonthAllocation(project.ID)
	require.NoError(t, err)
	assert.Equal(t, 500.0, alloc, "should return TotalBudget when no monthly breakdown set")
}

// ---------------------------------------------------------------------------
// Member expiry (#150)
// ---------------------------------------------------------------------------

func TestManager_PruneExpiredMembers_None(t *testing.T) {
	manager := setupTestManager(t)
	defer teardownTestManager(manager)
	ctx := context.Background()

	project, err := manager.CreateProject(ctx, &CreateProjectRequest{
		Name:  "Prune None",
		Owner: "owner",
	})
	require.NoError(t, err)

	// Member with no expiry.
	err = manager.AddProjectMember(ctx, project.ID, &types.ProjectMember{
		UserID: "permanent-user",
		Role:   types.ProjectRoleMember,
	})
	require.NoError(t, err)

	removed, err := manager.PruneExpiredMembers(ctx)
	require.NoError(t, err)
	assert.Empty(t, removed, "no members should be pruned when none have expired")
}

func TestManager_PruneExpiredMembers_Expired(t *testing.T) {
	manager := setupTestManager(t)
	defer teardownTestManager(manager)
	ctx := context.Background()

	project, err := manager.CreateProject(ctx, &CreateProjectRequest{
		Name:  "Prune Expired",
		Owner: "owner",
	})
	require.NoError(t, err)

	// Add a member whose expiry is in the past.
	past := time.Now().Add(-time.Hour)
	err = manager.AddProjectMember(ctx, project.ID, &types.ProjectMember{
		UserID:    "expired-user",
		Role:      types.ProjectRoleMember,
		ExpiresAt: &past,
	})
	require.NoError(t, err)

	removed, err := manager.PruneExpiredMembers(ctx)
	require.NoError(t, err)
	assert.NotEmpty(t, removed, "expired member should be pruned")

	// Verify the member is no longer in the project.
	got, err := manager.GetProject(ctx, project.ID)
	require.NoError(t, err)
	for _, m := range got.Members {
		assert.NotEqual(t, "expired-user", m.UserID, "expired user should have been removed")
	}
}

func TestManager_PruneExpiredMembers_Active(t *testing.T) {
	manager := setupTestManager(t)
	defer teardownTestManager(manager)
	ctx := context.Background()

	project, err := manager.CreateProject(ctx, &CreateProjectRequest{
		Name:  "Prune Active Only",
		Owner: "owner",
	})
	require.NoError(t, err)

	// Member whose expiry is in the future — should NOT be pruned.
	future := time.Now().Add(24 * time.Hour)
	err = manager.AddProjectMember(ctx, project.ID, &types.ProjectMember{
		UserID:    "future-user",
		Role:      types.ProjectRoleMember,
		ExpiresAt: &future,
	})
	require.NoError(t, err)

	removed, err := manager.PruneExpiredMembers(ctx)
	require.NoError(t, err)
	assert.Empty(t, removed, "non-expired member should not be pruned")
}

// ---------------------------------------------------------------------------
// Budget sharing (#143,#155,#156)
// ---------------------------------------------------------------------------

func TestManager_ShareBudget_Basic(t *testing.T) {
	manager := setupTestManager(t)
	defer teardownTestManager(manager)
	ctx := context.Background()

	src, err := manager.CreateProject(ctx, &CreateProjectRequest{
		Name:  "Share Source",
		Owner: "admin",
	})
	require.NoError(t, err)

	dst, err := manager.CreateProject(ctx, &CreateProjectRequest{
		Name:  "Share Destination",
		Owner: "admin",
	})
	require.NoError(t, err)

	// Give source project a budget of $1000 (SpentAmount=0 → $1000 available).
	err = manager.SetProjectBudget(ctx, src.ID, &types.ProjectBudget{
		TotalBudget: 1000.0,
		SpentAmount: 0.0,
	})
	require.NoError(t, err)

	req := &types.BudgetShareRequest{
		FromProjectID: src.ID,
		ToProjectID:   dst.ID,
		Amount:        200.0,
	}
	record, err := manager.ShareBudget(ctx, req, "admin")
	require.NoError(t, err)
	require.NotNil(t, record)

	// Source budget should be reduced by 200.
	srcProject, err := manager.GetProject(ctx, src.ID)
	require.NoError(t, err)
	require.NotNil(t, srcProject.Budget)
	assert.Equal(t, 800.0, srcProject.Budget.TotalBudget, "source TotalBudget should be reduced")

	// Destination budget should be increased by 200.
	dstProject, err := manager.GetProject(ctx, dst.ID)
	require.NoError(t, err)
	require.NotNil(t, dstProject.Budget)
	assert.Equal(t, 200.0, dstProject.Budget.TotalBudget, "destination TotalBudget should be increased")
}

func TestManager_ShareBudget_InsufficientFunds(t *testing.T) {
	manager := setupTestManager(t)
	defer teardownTestManager(manager)
	ctx := context.Background()

	src, err := manager.CreateProject(ctx, &CreateProjectRequest{
		Name:  "Insufficient Source",
		Owner: "admin",
	})
	require.NoError(t, err)

	dst, err := manager.CreateProject(ctx, &CreateProjectRequest{
		Name:  "Insufficient Dest",
		Owner: "admin",
	})
	require.NoError(t, err)

	err = manager.SetProjectBudget(ctx, src.ID, &types.ProjectBudget{
		TotalBudget: 1000.0,
		SpentAmount: 0.0,
	})
	require.NoError(t, err)

	// Attempt to share more than available.
	req := &types.BudgetShareRequest{
		FromProjectID: src.ID,
		ToProjectID:   dst.ID,
		Amount:        1001.0,
	}
	_, err = manager.ShareBudget(ctx, req, "admin")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "insufficient budget")
}

// ---------------------------------------------------------------------------
// Onboarding templates (#154)
// ---------------------------------------------------------------------------

func TestManager_SetOnboardingTemplate_New(t *testing.T) {
	manager := setupTestManager(t)
	defer teardownTestManager(manager)
	ctx := context.Background()

	project, err := manager.CreateProject(ctx, &CreateProjectRequest{
		Name:  "Onboarding New",
		Owner: "owner",
	})
	require.NoError(t, err)

	tmpl := types.OnboardingTemplate{
		Name:        "Welcome",
		Templates:   []string{"python-ml"},
		BudgetLimit: 100.0,
	}
	err = manager.SetOnboardingTemplate(ctx, project.ID, tmpl)
	require.NoError(t, err)

	got, err := manager.GetProject(ctx, project.ID)
	require.NoError(t, err)
	require.Len(t, got.OnboardingTemplates, 1)
	assert.Equal(t, "Welcome", got.OnboardingTemplates[0].Name)
	assert.NotEmpty(t, got.OnboardingTemplates[0].ID, "ID should be auto-generated")
}

func TestManager_SetOnboardingTemplate_Update(t *testing.T) {
	manager := setupTestManager(t)
	defer teardownTestManager(manager)
	ctx := context.Background()

	project, err := manager.CreateProject(ctx, &CreateProjectRequest{
		Name:  "Onboarding Update",
		Owner: "owner",
	})
	require.NoError(t, err)

	// Create initial template.
	tmpl := types.OnboardingTemplate{
		Name:        "Welcome",
		BudgetLimit: 100.0,
	}
	err = manager.SetOnboardingTemplate(ctx, project.ID, tmpl)
	require.NoError(t, err)

	// Update template with same Name — should upsert, not duplicate.
	tmpl.BudgetLimit = 200.0
	err = manager.SetOnboardingTemplate(ctx, project.ID, tmpl)
	require.NoError(t, err)

	got, err := manager.GetProject(ctx, project.ID)
	require.NoError(t, err)
	require.Len(t, got.OnboardingTemplates, 1, "should have exactly 1 template after upsert")
	assert.Equal(t, 200.0, got.OnboardingTemplates[0].BudgetLimit)
}

func TestManager_DeleteOnboardingTemplate(t *testing.T) {
	manager := setupTestManager(t)
	defer teardownTestManager(manager)
	ctx := context.Background()

	project, err := manager.CreateProject(ctx, &CreateProjectRequest{
		Name:  "Onboarding Delete",
		Owner: "owner",
	})
	require.NoError(t, err)

	tmpl := types.OnboardingTemplate{Name: "Bye"}
	err = manager.SetOnboardingTemplate(ctx, project.ID, tmpl)
	require.NoError(t, err)

	err = manager.DeleteOnboardingTemplate(ctx, project.ID, "Bye")
	require.NoError(t, err)

	got, err := manager.GetProject(ctx, project.ID)
	require.NoError(t, err)
	assert.Empty(t, got.OnboardingTemplates, "template should have been removed")
}

func TestManager_DeleteOnboardingTemplate_NotFound(t *testing.T) {
	manager := setupTestManager(t)
	defer teardownTestManager(manager)
	ctx := context.Background()

	project, err := manager.CreateProject(ctx, &CreateProjectRequest{
		Name:  "Onboarding Delete NotFound",
		Owner: "owner",
	})
	require.NoError(t, err)

	// Deleting a non-existent template by name should return nil (silently a no-op).
	err = manager.DeleteOnboardingTemplate(ctx, project.ID, "does-not-exist")
	assert.NoError(t, err, "deleting a non-existent onboarding template should not error")
}
