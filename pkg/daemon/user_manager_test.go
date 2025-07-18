package daemon

import (
	"context"
	"testing"
	"time"

	"github.com/scttfrdmn/cloudworkstation/pkg/usermgmt"
	"github.com/stretchr/testify/assert"
)

// TestUserManager tests the user manager
func TestUserManager(t *testing.T) {
	// Create a new user manager
	manager := NewUserManager()
	
	// Initialize the user manager
	err := manager.Initialize()
	assert.NoError(t, err, "Expected Initialize to succeed")
	assert.True(t, manager.initialized, "Expected user manager to be initialized")
	assert.NotNil(t, manager.service, "Expected user manager service to be created")
	assert.NotNil(t, manager.storage, "Expected user manager storage to be created")
	
	// Test creating a user
	user := &usermgmt.User{
		ID:          "test-user",
		Username:    "testuser",
		Email:       "test@example.com",
		DisplayName: "Test User",
		Provider:    usermgmt.ProviderLocal,
		Roles:       []usermgmt.UserRole{usermgmt.UserRoleUser},
		Enabled:     true,
	}
	
	createdUser, err := manager.CreateUser(context.Background(), user)
	assert.NoError(t, err, "Expected CreateUser to succeed")
	assert.Equal(t, user.ID, createdUser.ID, "Expected created user ID to match")
	assert.Equal(t, user.Username, createdUser.Username, "Expected created user username to match")
	assert.Equal(t, user.Email, createdUser.Email, "Expected created user email to match")
	
	// Test getting a user by ID
	retrievedUser, err := manager.GetUser(context.Background(), user.ID)
	assert.NoError(t, err, "Expected GetUser to succeed")
	assert.Equal(t, user.ID, retrievedUser.ID, "Expected retrieved user ID to match")
	
	// Test getting a user by username
	retrievedUser, err = manager.GetUserByUsername(context.Background(), user.Username)
	assert.NoError(t, err, "Expected GetUserByUsername to succeed")
	assert.Equal(t, user.ID, retrievedUser.ID, "Expected retrieved user ID to match")
	
	// Test updating a user
	user.DisplayName = "Updated Test User"
	updatedUser, err := manager.UpdateUser(context.Background(), user)
	assert.NoError(t, err, "Expected UpdateUser to succeed")
	assert.Equal(t, "Updated Test User", updatedUser.DisplayName, "Expected updated user display name to match")
	
	// Test disabling a user
	err = manager.DisableUser(context.Background(), user.ID)
	assert.NoError(t, err, "Expected DisableUser to succeed")
	
	// Verify user is disabled
	retrievedUser, err = manager.GetUser(context.Background(), user.ID)
	assert.NoError(t, err, "Expected GetUser to succeed")
	assert.False(t, retrievedUser.Enabled, "Expected user to be disabled")
	
	// Test enabling a user
	err = manager.EnableUser(context.Background(), user.ID)
	assert.NoError(t, err, "Expected EnableUser to succeed")
	
	// Verify user is enabled
	retrievedUser, err = manager.GetUser(context.Background(), user.ID)
	assert.NoError(t, err, "Expected GetUser to succeed")
	assert.True(t, retrievedUser.Enabled, "Expected user to be enabled")
	
	// Test creating a group
	group := &usermgmt.Group{
		ID:          "test-group",
		Name:        "testgroup",
		Description: "Test Group",
		Provider:    usermgmt.ProviderLocal,
	}
	
	_, err = manager.service.CreateGroup(context.Background(), group)
	assert.NoError(t, err, "Expected CreateGroup to succeed")
	
	// Test adding user to group
	err = manager.service.AddUserToGroup(context.Background(), user.ID, group.ID)
	assert.NoError(t, err, "Expected AddUserToGroup to succeed")
	
	// Verify user is in group
	groups, err := manager.service.GetUserGroups(context.Background(), user.ID)
	assert.NoError(t, err, "Expected GetUserGroups to succeed")
	assert.Equal(t, 1, len(groups), "Expected user to be in 1 group")
	assert.Equal(t, group.ID, groups[0].ID, "Expected group ID to match")
	
	// Test getting users in a group
	users, err := manager.service.GetGroupUsers(context.Background(), group.ID, nil)
	assert.NoError(t, err, "Expected GetGroupUsers to succeed")
	assert.Equal(t, 1, len(users.Users), "Expected group to have 1 user")
	assert.Equal(t, user.ID, users.Users[0].ID, "Expected user ID to match")
	
	// Test removing user from group
	err = manager.service.RemoveUserFromGroup(context.Background(), user.ID, group.ID)
	assert.NoError(t, err, "Expected RemoveUserFromGroup to succeed")
	
	// Verify user is not in group
	groups, err = manager.service.GetUserGroups(context.Background(), user.ID)
	assert.NoError(t, err, "Expected GetUserGroups to succeed")
	assert.Equal(t, 0, len(groups), "Expected user to not be in any groups")
	
	// Test deleting a group
	err = manager.service.DeleteGroup(context.Background(), group.ID)
	assert.NoError(t, err, "Expected DeleteGroup to succeed")
	
	// Test deleting a user
	err = manager.DeleteUser(context.Background(), user.ID)
	assert.NoError(t, err, "Expected DeleteUser to succeed")
	
	// Verify user is deleted
	_, err = manager.GetUser(context.Background(), user.ID)
	assert.Error(t, err, "Expected GetUser to fail")
	assert.Equal(t, usermgmt.ErrUserNotFound, err, "Expected error to be ErrUserNotFound")
	
	// Test closing the user manager
	err = manager.Close()
	assert.NoError(t, err, "Expected Close to succeed")
	assert.False(t, manager.initialized, "Expected user manager to not be initialized")
}

// TestPermissionChecking tests the permission checking system
func TestPermissionChecking(t *testing.T) {
	// Create a server with a user manager
	server := &Server{
		userManager: NewUserManager(),
	}
	
	// Initialize the user manager
	err := server.userManager.Initialize()
	assert.NoError(t, err, "Expected Initialize to succeed")
	
	// Create test users with different roles
	adminUser := &usermgmt.User{
		ID:          "admin-user",
		Username:    "admin",
		Email:       "admin@example.com",
		DisplayName: "Admin User",
		Provider:    usermgmt.ProviderLocal,
		Roles:       []usermgmt.UserRole{usermgmt.UserRoleAdmin},
		Enabled:     true,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	
	powerUser := &usermgmt.User{
		ID:          "power-user",
		Username:    "power",
		Email:       "power@example.com",
		DisplayName: "Power User",
		Provider:    usermgmt.ProviderLocal,
		Roles:       []usermgmt.UserRole{usermgmt.UserRolePowerUser},
		Enabled:     true,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	
	regularUser := &usermgmt.User{
		ID:          "regular-user",
		Username:    "regular",
		Email:       "regular@example.com",
		DisplayName: "Regular User",
		Provider:    usermgmt.ProviderLocal,
		Roles:       []usermgmt.UserRole{usermgmt.UserRoleUser},
		Enabled:     true,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	
	readOnlyUser := &usermgmt.User{
		ID:          "readonly-user",
		Username:    "readonly",
		Email:       "readonly@example.com",
		DisplayName: "Read-Only User",
		Provider:    usermgmt.ProviderLocal,
		Roles:       []usermgmt.UserRole{usermgmt.UserRoleReadOnly},
		Enabled:     true,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	
	disabledUser := &usermgmt.User{
		ID:          "disabled-user",
		Username:    "disabled",
		Email:       "disabled@example.com",
		DisplayName: "Disabled User",
		Provider:    usermgmt.ProviderLocal,
		Roles:       []usermgmt.UserRole{usermgmt.UserRoleAdmin}, // Even with admin role, should be denied
		Enabled:     false, // Disabled
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	
	// Create the users
	_, err = server.userManager.CreateUser(context.Background(), adminUser)
	assert.NoError(t, err, "Expected CreateUser to succeed for admin")
	
	_, err = server.userManager.CreateUser(context.Background(), powerUser)
	assert.NoError(t, err, "Expected CreateUser to succeed for power user")
	
	_, err = server.userManager.CreateUser(context.Background(), regularUser)
	assert.NoError(t, err, "Expected CreateUser to succeed for regular user")
	
	_, err = server.userManager.CreateUser(context.Background(), readOnlyUser)
	assert.NoError(t, err, "Expected CreateUser to succeed for read-only user")
	
	_, err = server.userManager.CreateUser(context.Background(), disabledUser)
	assert.NoError(t, err, "Expected CreateUser to succeed for disabled user")
	
	// Define test cases
	tests := []struct {
		name        string
		userID      string
		permission  Permission
		expectAllow bool
	}{
		// Admin user tests
		{
			name:   "Admin user can read instances",
			userID: adminUser.ID,
			permission: Permission{
				Resource:     ResourceInstance,
				Operation:    OperationRead,
				MinimumLevel: PermissionRead,
			},
			expectAllow: true,
		},
		{
			name:   "Admin user can write to instances",
			userID: adminUser.ID,
			permission: Permission{
				Resource:     ResourceInstance,
				Operation:    OperationCreate,
				MinimumLevel: PermissionWrite,
			},
			expectAllow: true,
		},
		{
			name:   "Admin user can manage system",
			userID: adminUser.ID,
			permission: Permission{
				Resource:     ResourceSystem,
				Operation:    OperationManage,
				MinimumLevel: PermissionAdmin,
			},
			expectAllow: true,
		},
		
		// Power user tests
		{
			name:   "Power user can read instances",
			userID: powerUser.ID,
			permission: Permission{
				Resource:     ResourceInstance,
				Operation:    OperationRead,
				MinimumLevel: PermissionRead,
			},
			expectAllow: true,
		},
		{
			name:   "Power user can write to instances",
			userID: powerUser.ID,
			permission: Permission{
				Resource:     ResourceInstance,
				Operation:    OperationCreate,
				MinimumLevel: PermissionWrite,
			},
			expectAllow: true,
		},
		{
			name:   "Power user can manage instances",
			userID: powerUser.ID,
			permission: Permission{
				Resource:     ResourceInstance,
				Operation:    OperationManage,
				MinimumLevel: PermissionAdmin,
			},
			expectAllow: true,
		},
		{
			name:   "Power user can read system",
			userID: powerUser.ID,
			permission: Permission{
				Resource:     ResourceSystem,
				Operation:    OperationRead,
				MinimumLevel: PermissionRead,
			},
			expectAllow: true,
		},
		{
			name:   "Power user cannot manage system",
			userID: powerUser.ID,
			permission: Permission{
				Resource:     ResourceSystem,
				Operation:    OperationManage,
				MinimumLevel: PermissionAdmin,
			},
			expectAllow: false,
		},
		
		// Regular user tests
		{
			name:   "Regular user can read instances",
			userID: regularUser.ID,
			permission: Permission{
				Resource:     ResourceInstance,
				Operation:    OperationRead,
				MinimumLevel: PermissionRead,
			},
			expectAllow: true,
		},
		{
			name:   "Regular user can write to instances",
			userID: regularUser.ID,
			permission: Permission{
				Resource:     ResourceInstance,
				Operation:    OperationCreate,
				MinimumLevel: PermissionWrite,
			},
			expectAllow: true,
		},
		{
			name:   "Regular user cannot manage instances",
			userID: regularUser.ID,
			permission: Permission{
				Resource:     ResourceInstance,
				Operation:    OperationManage,
				MinimumLevel: PermissionAdmin,
			},
			expectAllow: false,
		},
		{
			name:   "Regular user can read templates",
			userID: regularUser.ID,
			permission: Permission{
				Resource:     ResourceTemplate,
				Operation:    OperationRead,
				MinimumLevel: PermissionRead,
			},
			expectAllow: true,
		},
		{
			name:   "Regular user cannot write to templates",
			userID: regularUser.ID,
			permission: Permission{
				Resource:     ResourceTemplate,
				Operation:    OperationCreate,
				MinimumLevel: PermissionWrite,
			},
			expectAllow: false,
		},
		{
			name:   "Regular user cannot access users",
			userID: regularUser.ID,
			permission: Permission{
				Resource:     ResourceUser,
				Operation:    OperationRead,
				MinimumLevel: PermissionRead,
			},
			expectAllow: false,
		},
		
		// Read-only user tests
		{
			name:   "Read-only user can read instances",
			userID: readOnlyUser.ID,
			permission: Permission{
				Resource:     ResourceInstance,
				Operation:    OperationRead,
				MinimumLevel: PermissionRead,
			},
			expectAllow: true,
		},
		{
			name:   "Read-only user cannot write to instances",
			userID: readOnlyUser.ID,
			permission: Permission{
				Resource:     ResourceInstance,
				Operation:    OperationCreate,
				MinimumLevel: PermissionWrite,
			},
			expectAllow: false,
		},
		{
			name:   "Read-only user can read templates",
			userID: readOnlyUser.ID,
			permission: Permission{
				Resource:     ResourceTemplate,
				Operation:    OperationRead,
				MinimumLevel: PermissionRead,
			},
			expectAllow: true,
		},
		{
			name:   "Read-only user cannot write to templates",
			userID: readOnlyUser.ID,
			permission: Permission{
				Resource:     ResourceTemplate,
				Operation:    OperationCreate,
				MinimumLevel: PermissionWrite,
			},
			expectAllow: false,
		},
		
		// Disabled user tests
		{
			name:   "Disabled user cannot read instances",
			userID: disabledUser.ID,
			permission: Permission{
				Resource:     ResourceInstance,
				Operation:    OperationRead,
				MinimumLevel: PermissionRead,
			},
			expectAllow: false,
		},
		{
			name:   "Disabled user cannot write to instances",
			userID: disabledUser.ID,
			permission: Permission{
				Resource:     ResourceInstance,
				Operation:    OperationCreate,
				MinimumLevel: PermissionWrite,
			},
			expectAllow: false,
		},
	}
	
	// Run the tests
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			allowed, err := server.checkPermission(context.Background(), tt.userID, tt.permission)
			assert.NoError(t, err, "Expected checkPermission to succeed")
			assert.Equal(t, tt.expectAllow, allowed, "Permission check did not match expectation")
		})
	}
}