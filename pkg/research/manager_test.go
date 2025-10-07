package research

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Use the existing MockProfileManager from functional_test.go

// TestNewResearchUserManager tests research user manager initialization
func TestNewResearchUserManager(t *testing.T) {
	profileMgr := &MockProfileManager{}
	configDir := t.TempDir()

	manager := NewResearchUserManager(profileMgr, configDir)

	assert.NotNil(t, manager, "Manager should not be nil")
	assert.Equal(t, ResearchUserBaseUID, manager.baseUID, "Base UID should be set correctly")
	assert.Equal(t, ResearchUserBaseGID, manager.baseGID, "Base GID should be set correctly")
	assert.NotNil(t, manager.uidAllocations, "UID allocations should be initialized")
	assert.Equal(t, configDir, manager.configPath, "Config path should be set correctly")
}

// TestGetOrCreateResearchUser tests research user creation and retrieval
func TestGetOrCreateResearchUser(t *testing.T) {
	profileMgr := &MockProfileManager{}
	configDir := t.TempDir()
	manager := NewResearchUserManager(profileMgr, configDir)

	tests := []struct {
		name        string
		username    string
		profile     string
		expectError bool
		setup       func()
	}{
		{
			name:        "create_new_research_user",
			username:    "researcher1",
			profile:     "test-profile",
			expectError: false,
			setup: func() {
				profileMgr.SetCurrentProfile("test-profile")
			},
		},
		{
			name:        "get_existing_research_user",
			username:    "researcher1",
			profile:     "test-profile",
			expectError: false,
			setup: func() {
				profileMgr.SetCurrentProfile("test-profile")
			},
		},
		{
			name:        "create_user_different_profile",
			username:    "researcher2",
			profile:     "other-profile",
			expectError: false,
			setup: func() {
				profileMgr.SetCurrentProfile("other-profile")
			},
		},
		{
			name:        "invalid_username",
			username:    "",
			profile:     "test-profile",
			expectError: true,
			setup: func() {
				profileMgr.SetCurrentProfile("test-profile")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setup()

			user, err := manager.GetOrCreateResearchUser(tt.username)

			if tt.expectError {
				assert.Error(t, err, "Expected error for test case: %s", tt.name)
				assert.Nil(t, user, "User should be nil on error")
			} else {
				assert.NoError(t, err, "Expected no error for test case: %s", tt.name)
				assert.NotNil(t, user, "User should not be nil on success")

				// Verify user properties
				assert.Equal(t, tt.username, user.Username, "Username should match")
				assert.Equal(t, tt.profile, user.ProfileOwner, "Profile should match")
				assert.Greater(t, user.UID, ResearchUserBaseUID-1, "UID should be in research range")
				assert.Greater(t, user.GID, ResearchUserBaseGID-1, "GID should be in research range")
				assert.False(t, user.CreatedAt.IsZero(), "Created timestamp should be set")

				// Test default values
				assert.Equal(t, "/bin/bash", user.Shell, "Default shell should be bash")
				assert.True(t, user.CreateHomeDir, "Should create home directory by default")
				assert.NotNil(t, user.SSHPublicKeys, "SSH keys should be initialized")
				assert.NotNil(t, user.SecondaryGroups, "Secondary groups should be initialized")
				assert.NotNil(t, user.DefaultEnvironment, "Default environment should be initialized")
			}
		})
	}
}

// TestManagerListResearchUsers tests listing research users
func TestManagerListResearchUsers(t *testing.T) {
	profileMgr := &MockProfileManager{}
	configDir := t.TempDir()
	manager := NewResearchUserManager(profileMgr, configDir)

	// Setup test data
	profileMgr.SetCurrentProfile("profile1")
	user1, err := manager.GetOrCreateResearchUser("user1")
	require.NoError(t, err)

	user2, err := manager.GetOrCreateResearchUser("user2")
	require.NoError(t, err)

	profileMgr.SetCurrentProfile("profile2")
	user3, err := manager.GetOrCreateResearchUser("user3")
	require.NoError(t, err)
	assert.NotNil(t, user3)

	tests := []struct {
		name          string
		profile       string
		expectedCount int
		expectedUsers []string
	}{
		{
			name:          "list_profile1_users",
			profile:       "profile1",
			expectedCount: 2,
			expectedUsers: []string{"user1", "user2"},
		},
		{
			name:          "list_profile2_users",
			profile:       "profile2",
			expectedCount: 1,
			expectedUsers: []string{"user3"},
		},
		{
			name:          "list_empty_profile",
			profile:       "empty-profile",
			expectedCount: 0,
			expectedUsers: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			profileMgr.SetCurrentProfile(tt.profile)
			users, err := manager.ListResearchUsers()

			assert.NoError(t, err, "Expected no error listing users")
			assert.Len(t, users, tt.expectedCount, "Expected %d users", tt.expectedCount)

			usernames := make([]string, len(users))
			for i, user := range users {
				usernames[i] = user.Username
			}

			for _, expectedUser := range tt.expectedUsers {
				assert.Contains(t, usernames, expectedUser, "Expected user %s in results", expectedUser)
			}
		})
	}

	// Verify specific users exist with correct data
	profileMgr.SetCurrentProfile("profile1")
	users, err := manager.ListResearchUsers()
	require.NoError(t, err)
	require.Len(t, users, 2)

	assert.Equal(t, user1.Username, users[0].Username)
	assert.Equal(t, user1.UID, users[0].UID)
	assert.Equal(t, user2.Username, users[1].Username)
	assert.Equal(t, user2.UID, users[1].UID)
}

// TestUpdateResearchUser tests research user updates
func TestUpdateResearchUser(t *testing.T) {
	profileMgr := &MockProfileManager{}
	configDir := t.TempDir()
	manager := NewResearchUserManager(profileMgr, configDir)

	// Create initial user
	profileMgr.SetCurrentProfile("test-profile")
	user, err := manager.GetOrCreateResearchUser("testuser")
	require.NoError(t, err)
	require.NotNil(t, user)

	originalUID := user.UID
	originalCreatedAt := user.CreatedAt

	tests := []struct {
		name     string
		updates  ResearchUserConfig
		validate func(*testing.T, *ResearchUserConfig)
	}{
		{
			name: "update_basic_info",
			updates: ResearchUserConfig{
				FullName: "Updated Full Name",
				Email:    "updated@example.com",
			},
			validate: func(t *testing.T, user *ResearchUserConfig) {
				assert.Equal(t, "Updated Full Name", user.FullName)
				assert.Equal(t, "updated@example.com", user.Email)
			},
		},
		{
			name: "update_ssh_keys",
			updates: ResearchUserConfig{
				SSHPublicKeys:     []string{"ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAInew new@key"},
				SSHKeyFingerprint: "SHA256:NewFingerprint",
			},
			validate: func(t *testing.T, user *ResearchUserConfig) {
				assert.Len(t, user.SSHPublicKeys, 1)
				assert.Contains(t, user.SSHPublicKeys[0], "new@key")
				assert.Equal(t, "SHA256:NewFingerprint", user.SSHKeyFingerprint)
			},
		},
		{
			name: "update_groups_and_permissions",
			updates: ResearchUserConfig{
				SecondaryGroups: []string{"docker", "research", "admin"},
				SudoAccess:      true,
				DockerAccess:    true,
			},
			validate: func(t *testing.T, user *ResearchUserConfig) {
				assert.Contains(t, user.SecondaryGroups, "docker")
				assert.Contains(t, user.SecondaryGroups, "research")
				assert.Contains(t, user.SecondaryGroups, "admin")
				assert.True(t, user.SudoAccess)
				assert.True(t, user.DockerAccess)
			},
		},
		{
			name: "update_environment",
			updates: ResearchUserConfig{
				DefaultEnvironment: map[string]string{
					"CUSTOM_VAR": "custom_value",
					"PATH":       "/custom/bin:/usr/bin:/bin",
				},
			},
			validate: func(t *testing.T, user *ResearchUserConfig) {
				assert.Equal(t, "custom_value", user.DefaultEnvironment["CUSTOM_VAR"])
				assert.Equal(t, "/custom/bin:/usr/bin:/bin", user.DefaultEnvironment["PATH"])
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Update the user with the test profile
			tt.updates.Username = "testuser"
			err := manager.UpdateResearchUser("test-profile", &tt.updates)
			assert.NoError(t, err, "Expected no error updating user")

			// Get the updated user
			updatedUser, err := manager.GetResearchUser("test-profile", "testuser")
			assert.NoError(t, err, "Expected no error getting updated user")
			assert.NotNil(t, updatedUser, "Updated user should not be nil")

			// Verify core properties are preserved
			assert.Equal(t, "testuser", updatedUser.Username)
			assert.Equal(t, originalUID, updatedUser.UID, "UID should not change")
			assert.Equal(t, originalCreatedAt, updatedUser.CreatedAt, "CreatedAt should not change")
			assert.Equal(t, "test-profile", updatedUser.ProfileOwner)

			// Run specific validation for this test
			tt.validate(t, updatedUser)

			// Verify LastUsed is updated
			assert.NotNil(t, updatedUser.LastUsed, "LastUsed should be set after update")
			assert.True(t, updatedUser.LastUsed.After(originalCreatedAt), "LastUsed should be after creation")
		})
	}
}

// TestDeleteResearchUser tests research user deletion
func TestDeleteResearchUser(t *testing.T) {
	profileMgr := &MockProfileManager{}
	configDir := t.TempDir()
	manager := NewResearchUserManager(profileMgr, configDir)

	// Create test users
	profileMgr.SetCurrentProfile("test-profile")
	user1, err := manager.GetOrCreateResearchUser("user1")
	require.NoError(t, err)
	assert.NotNil(t, user1)

	user2, err := manager.GetOrCreateResearchUser("user2")
	require.NoError(t, err)
	assert.NotNil(t, user2)

	tests := []struct {
		name        string
		username    string
		expectError bool
		validate    func(*testing.T)
	}{
		{
			name:        "delete_existing_user",
			username:    "user1",
			expectError: false,
			validate: func(t *testing.T) {
				// Verify user is deleted
				users, err := manager.ListResearchUsers()
				assert.NoError(t, err)
				assert.Len(t, users, 1, "Should have one user remaining")
				assert.Equal(t, "user2", users[0].Username, "Remaining user should be user2")
			},
		},
		{
			name:        "delete_nonexistent_user",
			username:    "nonexistent",
			expectError: true,
			validate: func(t *testing.T) {
				// Verify other users are unaffected
				users, err := manager.ListResearchUsers()
				assert.NoError(t, err)
				assert.Len(t, users, 1, "Should still have one user")
			},
		},
		{
			name:        "delete_remaining_user",
			username:    "user2",
			expectError: false,
			validate: func(t *testing.T) {
				// Verify no users remain
				users, err := manager.ListResearchUsers()
				assert.NoError(t, err)
				assert.Len(t, users, 0, "Should have no users")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := manager.DeleteResearchUser("test-profile", tt.username)

			if tt.expectError {
				assert.Error(t, err, "Expected error deleting user: %s", tt.username)
			} else {
				assert.NoError(t, err, "Expected no error deleting user: %s", tt.username)
			}

			tt.validate(t)
		})
	}
}

// TestResearchUserPersistence tests user data persistence
func TestResearchUserPersistence(t *testing.T) {
	profileMgr := &MockProfileManager{}
	configDir := t.TempDir()

	// Create first manager and user
	manager1 := NewResearchUserManager(profileMgr, configDir)
	profileMgr.SetCurrentProfile("test-profile")

	user1, err := manager1.GetOrCreateResearchUser("persistentuser")
	require.NoError(t, err)
	require.NotNil(t, user1)

	originalUID := user1.UID
	originalCreatedAt := user1.CreatedAt

	// Update user data
	updates := &ResearchUserConfig{
		FullName:   "Persistent User",
		Email:      "persistent@example.com",
		SudoAccess: true,
	}
	updates.Username = "persistentuser"
	err = manager1.UpdateResearchUser("test-profile", updates)
	require.NoError(t, err)

	// Get the updated user
	updatedUser, err := manager1.GetResearchUser("test-profile", "persistentuser")
	require.NoError(t, err)
	assert.NotNil(t, updatedUser)

	// Create second manager instance (simulating restart)
	manager2 := NewResearchUserManager(profileMgr, configDir)

	// Retrieve user with second manager
	retrievedUser, err := manager2.GetOrCreateResearchUser("persistentuser")
	require.NoError(t, err)
	require.NotNil(t, retrievedUser)

	// Verify data persistence
	assert.Equal(t, "persistentuser", retrievedUser.Username)
	assert.Equal(t, originalUID, retrievedUser.UID, "UID should persist")
	assert.Equal(t, originalCreatedAt, retrievedUser.CreatedAt, "CreatedAt should persist")
	assert.Equal(t, "Persistent User", retrievedUser.FullName, "FullName should persist")
	assert.Equal(t, "persistent@example.com", retrievedUser.Email, "Email should persist")
	assert.True(t, retrievedUser.SudoAccess, "SudoAccess should persist")

	// Verify both managers see the same data
	users1, err := manager1.ListResearchUsers()
	require.NoError(t, err)

	users2, err := manager2.ListResearchUsers()
	require.NoError(t, err)

	assert.Equal(t, len(users1), len(users2), "Both managers should see same number of users")
	assert.Equal(t, users1[0].UID, users2[0].UID, "Both managers should see same user data")
}

// TestConcurrentUserAccess tests concurrent access to research users
func TestConcurrentUserAccess(t *testing.T) {
	profileMgr := &MockProfileManager{}
	configDir := t.TempDir()
	manager := NewResearchUserManager(profileMgr, configDir)

	profileMgr.SetCurrentProfile("test-profile")

	// Create test user
	user, err := manager.GetOrCreateResearchUser("concurrent-user")
	require.NoError(t, err)
	require.NotNil(t, user)

	// Test concurrent reads
	t.Run("concurrent_reads", func(t *testing.T) {
		done := make(chan bool, 5)

		for i := 0; i < 5; i++ {
			go func() {
				defer func() { done <- true }()
				retrievedUser, err := manager.GetOrCreateResearchUser("concurrent-user")
				assert.NoError(t, err)
				assert.NotNil(t, retrievedUser)
				assert.Equal(t, "concurrent-user", retrievedUser.Username)
				assert.Equal(t, user.UID, retrievedUser.UID)
			}()
		}

		// Wait for all goroutines to complete
		for i := 0; i < 5; i++ {
			<-done
		}
	})

	// Test concurrent updates (note: this is a simplified test)
	t.Run("concurrent_updates", func(t *testing.T) {
		done := make(chan bool, 3)

		for i := 0; i < 3; i++ {
			go func(iteration int) {
				defer func() { done <- true }()
				updates := &ResearchUserConfig{
					FullName: fmt.Sprintf("Updated Name %d", iteration),
				}
				updates.Username = "concurrent-user"
				err := manager.UpdateResearchUser("test-profile", updates)
				assert.NoError(t, err)

				// Get the updated user
				updatedUser, err := manager.GetResearchUser("test-profile", "concurrent-user")
				assert.NoError(t, err)
				assert.NotNil(t, updatedUser)
				assert.Equal(t, "concurrent-user", updatedUser.Username)
			}(i)
		}

		// Wait for all goroutines to complete
		for i := 0; i < 3; i++ {
			<-done
		}

		// Verify final state is consistent
		finalUser, err := manager.GetOrCreateResearchUser("concurrent-user")
		assert.NoError(t, err)
		assert.NotNil(t, finalUser)
		assert.Equal(t, "concurrent-user", finalUser.Username)
		assert.Equal(t, user.UID, finalUser.UID)
	})
}

// TestResearchUserManagerErrorHandling tests error handling scenarios
func TestResearchUserManagerErrorHandling(t *testing.T) {
	tests := []struct {
		name        string
		setup       func() (*ResearchUserManager, *MockProfileManager)
		action      func(*ResearchUserManager) error
		expectError bool
		errorCheck  func(*testing.T, error)
	}{
		{
			name: "no_current_profile",
			setup: func() (*ResearchUserManager, *MockProfileManager) {
				profileMgr := &MockProfileManager{}
				profileMgr.SetCurrentProfile("") // No current profile
				configDir := t.TempDir()
				manager := NewResearchUserManager(profileMgr, configDir)
				return manager, profileMgr
			},
			action: func(manager *ResearchUserManager) error {
				_, err := manager.GetOrCreateResearchUser("testuser")
				return err
			},
			expectError: true,
			errorCheck: func(t *testing.T, err error) {
				assert.Contains(t, err.Error(), "profile", "Error should mention profile")
			},
		},
		{
			name: "invalid_config_directory",
			setup: func() (*ResearchUserManager, *MockProfileManager) {
				profileMgr := &MockProfileManager{}
				profileMgr.SetCurrentProfile("test-profile")
				// Use a path that doesn't exist and can't be created
				invalidPath := "/root/nonexistent/path"
				manager := NewResearchUserManager(profileMgr, invalidPath)
				return manager, profileMgr
			},
			action: func(manager *ResearchUserManager) error {
				_, err := manager.GetOrCreateResearchUser("testuser")
				return err
			},
			expectError: true,
			errorCheck: func(t *testing.T, err error) {
				// This might not always fail depending on permissions, so we just check structure
				assert.NotNil(t, err, "Should have some error with invalid path")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manager, _ := tt.setup()
			err := tt.action(manager)

			if tt.expectError {
				assert.Error(t, err, "Expected error for test case: %s", tt.name)
				if tt.errorCheck != nil {
					tt.errorCheck(t, err)
				}
			} else {
				assert.NoError(t, err, "Expected no error for test case: %s", tt.name)
			}
		})
	}
}
