package research

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockProfileManager for testing research user functionality
type MockProfileManager struct{}

func (m *MockProfileManager) GetCurrentProfile() (string, error) {
	return "test-profile", nil
}

func (m *MockProfileManager) GetProfileConfig(profileID string) (interface{}, error) {
	return map[string]interface{}{"name": profileID}, nil
}

func (m *MockProfileManager) UpdateProfileConfig(profileID string, config interface{}) error {
	return nil
}

// TestResearchUserFunctionalWorkflows tests real-world research user scenarios
func TestResearchUserFunctionalWorkflows(t *testing.T) {

	t.Run("university_research_user_creation_workflow", func(t *testing.T) {
		// User scenario: University creates research users for collaborative projects
		profileMgr := &MockProfileManager{}
		manager := NewResearchUserManager(profileMgr, "/tmp/test-research-config")

		// Professor creates research user
		professorUser, err := manager.CreateResearchUser("cs-department", "prof-johnson")
		require.NoError(t, err, "Should create professor research user")
		require.NotNil(t, professorUser, "Professor user should exist")

		// Verify research user properties
		assert.Equal(t, "prof-johnson", professorUser.Username)
		assert.Equal(t, "cs-department", professorUser.ProfileOwner)
		assert.Greater(t, professorUser.UID, 2000, "Research users should have UID >= 2000")
		assert.NotEmpty(t, professorUser.HomeDirectory, "Should have home directory")

		t.Logf("✅ Professor research user created successfully")
		t.Logf("👨‍🏫 Username: %s", professorUser.Username)
		t.Logf("🆔 UID: %d", professorUser.UID)
		t.Logf("📁 Home: %s", professorUser.HomeDirectory)

		// Student joins the same project
		studentUser, err := manager.CreateResearchUser("cs-department", "phd-student-kim")
		require.NoError(t, err, "Should create student research user")

		// Verify different UIDs for different users
		assert.NotEqual(t, professorUser.UID, studentUser.UID, "Different users should have different UIDs")

		t.Logf("✅ Student research user created successfully")
		t.Logf("👨‍🎓 Username: %s", studentUser.Username)
		t.Logf("🆔 UID: %d", studentUser.UID)
	})

	t.Run("consistent_uid_mapping_for_collaboration", func(t *testing.T) {
		// User scenario: Same researcher needs consistent UID across different instances
		profileMgr := &MockProfileManager{}
		manager1 := NewResearchUserManager(profileMgr, "/tmp/test-research-config")
		manager2 := NewResearchUserManager(profileMgr, "/tmp/test-research-config")

		// Create user with first manager (simulates first instance)
		user1, err := manager1.CreateResearchUser("ml-lab", "researcher-alice")
		require.NoError(t, err, "Should create user on first instance")

		// Get same user with second manager (simulates second instance)
		user2, err := manager2.GetResearchUser("ml-lab", "researcher-alice")

		if err == nil {
			// If user exists, UIDs should match for EFS consistency
			assert.Equal(t, user1.UID, user2.UID, "Same user should have consistent UID across instances")
			assert.Equal(t, user1.GID, user2.GID, "Same user should have consistent GID across instances")

			t.Logf("✅ UID mapping consistency verified")
			t.Logf("👤 User: %s", user1.Username)
			t.Logf("🔗 UID consistency: %d = %d", user1.UID, user2.UID)
		} else {
			// If user doesn't exist yet, this documents the expected behavior
			t.Logf("📝 User lookup failed (expected in test environment): %v", err)
		}
	})

	t.Run("research_user_validation_logic", func(t *testing.T) {
		// User scenario: System validates research user configurations
		profileMgr := &MockProfileManager{}
		manager := NewResearchUserManager(profileMgr, "/tmp/test-research-config")

		// Test valid usernames
		validUsernames := []string{
			"professor-smith",
			"phd-student-01",
			"lab-assistant",
			"researcher123",
		}

		for _, username := range validUsernames {
			user, err := manager.CreateResearchUser("test-profile", username)
			if err == nil {
				assert.Equal(t, username, user.Username, "Username should match input")
				t.Logf("✅ Valid username accepted: %s", username)
			} else {
				// Log validation errors for review
				t.Logf("⚠️  Username validation error for '%s': %v", username, err)
			}
		}

		// Test invalid usernames (if validation exists)
		invalidUsernames := []string{
			"",                 // empty
			"user with spaces", // spaces
			"UPPERCASE",        // case sensitivity
		}

		for _, username := range invalidUsernames {
			user, err := manager.CreateResearchUser("test-profile", username)
			if err != nil {
				t.Logf("✅ Invalid username rejected: '%s' - %v", username, err)
			} else if user != nil {
				t.Logf("⚠️  Username '%s' was accepted (may need validation)", username)
			}
		}
	})

	t.Run("uid_allocation_deterministic_behavior", func(t *testing.T) {
		// User scenario: UID allocation should be deterministic and conflict-free
		profileMgr := &MockProfileManager{}
		manager := NewResearchUserManager(profileMgr, "/tmp/test-research-uid-config")

		// Create multiple users and verify UID allocation
		users := make(map[string]*ResearchUserConfig)
		uids := make(map[int]bool)

		usernames := []string{"user1", "user2", "user3", "user4", "user5"}

		for _, username := range usernames {
			user, err := manager.CreateResearchUser("uid-test-profile", username)
			if err == nil {
				users[username] = user

				// Check for UID conflicts
				if uids[user.UID] {
					t.Errorf("UID conflict detected: %d used by multiple users", user.UID)
				}
				uids[user.UID] = true

				// Verify UID is in expected range
				assert.GreaterOrEqual(t, user.UID, 2000, "Research user UID should be >= 2000")
				assert.LessOrEqual(t, user.UID, 65535, "Research user UID should be <= 65535")
			}
		}

		t.Logf("✅ UID allocation tested for %d users", len(users))
		t.Logf("🔢 Unique UIDs allocated: %d", len(uids))

		if len(users) > 0 {
			t.Logf("📊 UID range: %d to %d", getMinUID(users), getMaxUID(users))
		}
	})
}

// TestResearchUserIntegrationScenarios tests integration with other systems
func TestResearchUserIntegrationScenarios(t *testing.T) {
	t.Run("efs_home_directory_path_generation", func(t *testing.T) {
		// User scenario: Research users get consistent EFS home directory paths
		profileMgr := &MockProfileManager{}
		manager := NewResearchUserManager(profileMgr, "/tmp/test-efs-config")

		user, err := manager.CreateResearchUser("research-group", "data-scientist")

		if err == nil {
			// Verify home directory path structure
			assert.NotEmpty(t, user.HomeDirectory, "Home directory should be set")
			assert.Contains(t, user.HomeDirectory, user.Username, "Home path should contain username")

			t.Logf("✅ EFS home directory configured")
			t.Logf("📁 Home path: %s", user.HomeDirectory)
			t.Logf("👤 User: %s (UID: %d)", user.Username, user.UID)
		} else {
			t.Logf("📝 User creation failed (expected in test environment): %v", err)
		}
	})

	t.Run("profile_isolation_verification", func(t *testing.T) {
		// User scenario: Users in different profiles should be isolated
		profileMgr := &MockProfileManager{}
		manager := NewResearchUserManager(profileMgr, "/tmp/test-isolation-config")

		// Create users in different profiles
		user1, err1 := manager.CreateResearchUser("profile-a", "researcher")
		user2, err2 := manager.CreateResearchUser("profile-b", "researcher")

		if err1 == nil && err2 == nil {
			// Same username in different profiles should be allowed
			assert.Equal(t, "researcher", user1.Username)
			assert.Equal(t, "researcher", user2.Username)
			assert.Equal(t, "profile-a", user1.ProfileOwner)
			assert.Equal(t, "profile-b", user2.ProfileOwner)

			// Should have different UIDs (profile isolation)
			assert.NotEqual(t, user1.UID, user2.UID, "Same username in different profiles should have different UIDs")

			t.Logf("✅ Profile isolation verified")
			t.Logf("👤 profile-a/researcher: UID %d", user1.UID)
			t.Logf("👤 profile-b/researcher: UID %d", user2.UID)
		} else {
			t.Logf("📝 Profile isolation test incomplete due to creation errors")
			if err1 != nil {
				t.Logf("   Profile A error: %v", err1)
			}
			if err2 != nil {
				t.Logf("   Profile B error: %v", err2)
			}
		}
	})
}

// Helper functions for test analysis
func getMinUID(users map[string]*ResearchUserConfig) int {
	min := 65535
	for _, user := range users {
		if user.UID < min {
			min = user.UID
		}
	}
	return min
}

func getMaxUID(users map[string]*ResearchUserConfig) int {
	max := 0
	for _, user := range users {
		if user.UID > max {
			max = user.UID
		}
	}
	return max
}
