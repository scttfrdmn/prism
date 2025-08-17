package state

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/scttfrdmn/cloudworkstation/pkg/usermgmt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestUserStorage tests the user storage implementation
func TestUserStorage(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "user-storage-test")
	require.NoError(t, err, "Failed to create temp directory")
	defer func() { _ = os.RemoveAll(tempDir) }()

	// Create a state manager with the temp directory
	manager := &Manager{
		statePath: filepath.Join(tempDir, "state.json"),
		userPath:  filepath.Join(tempDir, "users.json"),
	}

	// Test loading an empty state
	state, err := manager.LoadUserState()
	require.NoError(t, err, "Failed to load user state")
	assert.NotNil(t, state, "User state should not be nil")
	assert.Empty(t, state.Users, "Users should be empty")
	assert.Empty(t, state.Groups, "Groups should be empty")

	// Create a test user
	user := &usermgmt.User{
		ID:          "test-user",
		Username:    "testuser",
		Email:       "test@example.com",
		DisplayName: "Test User",
		Provider:    usermgmt.ProviderLocal,
		Roles:       []usermgmt.UserRole{usermgmt.UserRoleUser},
		Enabled:     true,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	// Test creating a user
	createdUser, err := manager.CreateUser(context.Background(), user)
	require.NoError(t, err, "Failed to create user")
	assert.Equal(t, user.ID, createdUser.ID, "User ID should match")
	assert.Equal(t, user.Username, createdUser.Username, "Username should match")
	assert.Equal(t, user.Email, createdUser.Email, "Email should match")

	// Test getting a user by ID
	retrievedUser, err := manager.GetUser(context.Background(), user.ID)
	require.NoError(t, err, "Failed to get user by ID")
	assert.Equal(t, user.ID, retrievedUser.ID, "User ID should match")

	// Test getting a user by username
	retrievedUser, err = manager.GetUserByUsername(context.Background(), user.Username)
	require.NoError(t, err, "Failed to get user by username")
	assert.Equal(t, user.ID, retrievedUser.ID, "User ID should match")

	// Test getting a user by email
	retrievedUser, err = manager.GetUserByEmail(context.Background(), user.Email)
	require.NoError(t, err, "Failed to get user by email")
	assert.Equal(t, user.ID, retrievedUser.ID, "User ID should match")

	// Test listing users
	users, err := manager.GetUsers(context.Background(), nil, nil)
	require.NoError(t, err, "Failed to list users")
	assert.Equal(t, 1, len(users.Users), "Should have one user")
	assert.Equal(t, user.ID, users.Users[0].ID, "User ID should match")

	// Test creating a group
	group := &usermgmt.Group{
		ID:          "test-group",
		Name:        "testgroup",
		Description: "Test Group",
		Provider:    usermgmt.ProviderLocal,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	createdGroup, err := manager.CreateGroup(context.Background(), group)
	require.NoError(t, err, "Failed to create group")
	assert.Equal(t, group.ID, createdGroup.ID, "Group ID should match")
	assert.Equal(t, group.Name, createdGroup.Name, "Group name should match")

	// Test getting a group by ID
	retrievedGroup, err := manager.GetGroup(context.Background(), group.ID)
	require.NoError(t, err, "Failed to get group by ID")
	assert.Equal(t, group.ID, retrievedGroup.ID, "Group ID should match")

	// Test getting a group by name
	retrievedGroup, err = manager.GetGroupByName(context.Background(), group.Name)
	require.NoError(t, err, "Failed to get group by name")
	assert.Equal(t, group.ID, retrievedGroup.ID, "Group ID should match")

	// Test listing groups
	groups, err := manager.GetGroups(context.Background(), nil, nil)
	require.NoError(t, err, "Failed to list groups")
	assert.Equal(t, 1, len(groups.Groups), "Should have one group")
	assert.Equal(t, group.ID, groups.Groups[0].ID, "Group ID should match")

	// Test adding a user to a group
	err = manager.AddUserToGroup(context.Background(), user.ID, group.ID)
	require.NoError(t, err, "Failed to add user to group")

	// Test checking if a user is in a group
	isInGroup, err := manager.IsUserInGroup(context.Background(), user.ID, group.ID)
	require.NoError(t, err, "Failed to check if user is in group")
	assert.True(t, isInGroup, "User should be in group")

	// Test getting groups for a user
	userGroups, err := manager.GetUserGroups(context.Background(), user.ID)
	require.NoError(t, err, "Failed to get user groups")
	assert.Equal(t, 1, len(userGroups), "User should be in one group")
	assert.Equal(t, group.ID, userGroups[0].ID, "Group ID should match")

	// Test getting users in a group
	groupUsers, err := manager.GetGroupUsers(context.Background(), group.ID, nil)
	require.NoError(t, err, "Failed to get group users")
	assert.Equal(t, 1, len(groupUsers.Users), "Group should have one user")
	assert.Equal(t, user.ID, groupUsers.Users[0].ID, "User ID should match")

	// Test removing a user from a group
	err = manager.RemoveUserFromGroup(context.Background(), user.ID, group.ID)
	require.NoError(t, err, "Failed to remove user from group")

	// Check that the user is no longer in the group
	isInGroup, err = manager.IsUserInGroup(context.Background(), user.ID, group.ID)
	require.NoError(t, err, "Failed to check if user is in group")
	assert.False(t, isInGroup, "User should not be in group")

	// Test updating a user
	user.DisplayName = "Updated User"
	updatedUser, err := manager.UpdateUser(context.Background(), user)
	require.NoError(t, err, "Failed to update user")
	assert.Equal(t, "Updated User", updatedUser.DisplayName, "Display name should be updated")

	// Test updating a group
	group.Description = "Updated Group"
	updatedGroup, err := manager.UpdateGroup(context.Background(), group)
	require.NoError(t, err, "Failed to update group")
	assert.Equal(t, "Updated Group", updatedGroup.Description, "Description should be updated")

	// Test deleting a group
	err = manager.DeleteGroup(context.Background(), group.ID)
	require.NoError(t, err, "Failed to delete group")

	// Check that the group is deleted
	_, err = manager.GetGroup(context.Background(), group.ID)
	assert.Equal(t, usermgmt.ErrGroupNotFound, err, "Group should be deleted")

	// Test deleting a user
	err = manager.DeleteUser(context.Background(), user.ID)
	require.NoError(t, err, "Failed to delete user")

	// Check that the user is deleted
	_, err = manager.GetUser(context.Background(), user.ID)
	assert.Equal(t, usermgmt.ErrUserNotFound, err, "User should be deleted")
}

// TestUserFilters tests the user filtering functionality
func TestUserFilters(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "user-filters-test")
	require.NoError(t, err, "Failed to create temp directory")
	defer func() { _ = os.RemoveAll(tempDir) }()

	// Create a state manager with the temp directory
	manager := &Manager{
		statePath: filepath.Join(tempDir, "state.json"),
		userPath:  filepath.Join(tempDir, "users.json"),
	}

	// Create test users
	adminUser := &usermgmt.User{
		ID:          "admin-user",
		Username:    "admin",
		Email:       "admin@example.com",
		DisplayName: "Admin User",
		Provider:    usermgmt.ProviderLocal,
		Roles:       []usermgmt.UserRole{usermgmt.UserRoleAdmin},
		Enabled:     true,
		CreatedAt:   time.Now().Add(-48 * time.Hour),
		UpdatedAt:   time.Now().Add(-24 * time.Hour),
	}

	regularUser := &usermgmt.User{
		ID:          "regular-user",
		Username:    "regular",
		Email:       "regular@example.com",
		DisplayName: "Regular User",
		Provider:    usermgmt.ProviderLocal,
		Roles:       []usermgmt.UserRole{usermgmt.UserRoleUser},
		Enabled:     true,
		CreatedAt:   time.Now().Add(-24 * time.Hour),
		UpdatedAt:   time.Now(),
	}

	disabledUser := &usermgmt.User{
		ID:          "disabled-user",
		Username:    "disabled",
		Email:       "disabled@example.com",
		DisplayName: "Disabled User",
		Provider:    usermgmt.ProviderLocal,
		Roles:       []usermgmt.UserRole{usermgmt.UserRoleUser},
		Enabled:     false,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	// Create the users
	_, err = manager.CreateUser(context.Background(), adminUser)
	require.NoError(t, err, "Failed to create admin user")

	_, err = manager.CreateUser(context.Background(), regularUser)
	require.NoError(t, err, "Failed to create regular user")

	_, err = manager.CreateUser(context.Background(), disabledUser)
	require.NoError(t, err, "Failed to create disabled user")

	// Create a test group
	group := &usermgmt.Group{
		ID:          "test-group",
		Name:        "testgroup",
		Description: "Test Group",
		Provider:    usermgmt.ProviderLocal,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	_, err = manager.CreateGroup(context.Background(), group)
	require.NoError(t, err, "Failed to create group")

	// Add admin user to the group
	err = manager.AddUserToGroup(context.Background(), adminUser.ID, group.ID)
	require.NoError(t, err, "Failed to add admin user to group")

	// Test filtering by username
	filter := &usermgmt.UserFilter{
		Username: "admin",
	}
	users, err := manager.GetUsers(context.Background(), filter, nil)
	require.NoError(t, err, "Failed to filter users by username")
	assert.Equal(t, 1, len(users.Users), "Should have one user")
	assert.Equal(t, adminUser.ID, users.Users[0].ID, "User ID should match")

	// Test filtering by email
	filter = &usermgmt.UserFilter{
		Email: "regular@example.com",
	}
	users, err = manager.GetUsers(context.Background(), filter, nil)
	require.NoError(t, err, "Failed to filter users by email")
	assert.Equal(t, 1, len(users.Users), "Should have one user")
	assert.Equal(t, regularUser.ID, users.Users[0].ID, "User ID should match")

	// Test filtering by role
	filter = &usermgmt.UserFilter{
		Role: usermgmt.UserRoleAdmin,
	}
	users, err = manager.GetUsers(context.Background(), filter, nil)
	require.NoError(t, err, "Failed to filter users by role")
	assert.Equal(t, 1, len(users.Users), "Should have one user")
	assert.Equal(t, adminUser.ID, users.Users[0].ID, "User ID should match")

	// Test filtering by group
	filter = &usermgmt.UserFilter{
		Group: "testgroup",
	}
	users, err = manager.GetUsers(context.Background(), filter, nil)
	require.NoError(t, err, "Failed to filter users by group")
	assert.Equal(t, 1, len(users.Users), "Should have one user")
	assert.Equal(t, adminUser.ID, users.Users[0].ID, "User ID should match")

	// Test filtering by enabled status
	filter = &usermgmt.UserFilter{
		EnabledOnly: true,
	}
	users, err = manager.GetUsers(context.Background(), filter, nil)
	require.NoError(t, err, "Failed to filter users by enabled status")
	assert.Equal(t, 2, len(users.Users), "Should have two users")

	// Test filtering by disabled status
	filter = &usermgmt.UserFilter{
		DisabledOnly: true,
	}
	users, err = manager.GetUsers(context.Background(), filter, nil)
	require.NoError(t, err, "Failed to filter users by disabled status")
	assert.Equal(t, 1, len(users.Users), "Should have one user")
	assert.Equal(t, disabledUser.ID, users.Users[0].ID, "User ID should match")

	// Test filtering by creation time
	createdAfter := regularUser.CreatedAt.Add(time.Hour) // Only disabledUser should match
	filter = &usermgmt.UserFilter{
		CreatedAfter: &createdAfter,
	}
	users, err = manager.GetUsers(context.Background(), filter, nil)
	require.NoError(t, err, "Failed to filter users by creation time")
	assert.Equal(t, 1, len(users.Users), "Should have one user")
	assert.Equal(t, disabledUser.ID, users.Users[0].ID, "User ID should match")

	// Test filtering by update time
	updatedAfter := regularUser.UpdatedAt.Add(-time.Hour)
	filter = &usermgmt.UserFilter{
		UpdatedAfter: &updatedAfter,
	}
	users, err = manager.GetUsers(context.Background(), filter, nil)
	require.NoError(t, err, "Failed to filter users by update time")
	assert.Equal(t, 2, len(users.Users), "Should have two users")

	// Test pagination
	pagination := &usermgmt.PaginationOptions{
		Page:      1,
		PageSize:  1,
		SortBy:    "username",
		SortOrder: "asc",
	}
	users, err = manager.GetUsers(context.Background(), nil, pagination)
	require.NoError(t, err, "Failed to paginate users")
	assert.Equal(t, 1, len(users.Users), "Should have one user per page")
	assert.Equal(t, 3, users.Total, "Total should be three users")
	assert.Equal(t, 3, users.TotalPages, "Should have three pages")
}
