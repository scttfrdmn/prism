package usermgmt

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestProviderConstants tests all provider constants
func TestProviderConstants(t *testing.T) {
	providers := []struct {
		provider Provider
		expected string
	}{
		{ProviderAWSSO, "aws-sso"},
		{ProviderOkta, "okta"},
		{ProviderAzureAD, "azure-ad"},
		{ProviderGoogleWorkspace, "google-workspace"},
		{ProviderOneLogin, "onelogin"},
		{ProviderLocal, "local"},
		{ProviderOIDC, "oidc"},
	}

	for _, tc := range providers {
		t.Run(string(tc.provider), func(t *testing.T) {
			assert.Equal(t, tc.expected, string(tc.provider))
		})
	}
}

// TestUserRoleConstants tests all user role constants
func TestUserRoleConstants(t *testing.T) {
	roles := []struct {
		role     UserRole
		expected string
	}{
		{UserRoleAdmin, "admin"},
		{UserRolePowerUser, "power-user"},
		{UserRoleUser, "user"},
		{UserRoleReadOnly, "read-only"},
	}

	for _, tc := range roles {
		t.Run(string(tc.role), func(t *testing.T) {
			assert.Equal(t, tc.expected, string(tc.role))
		})
	}
}

// TestErrorConstants tests all error constants
func TestErrorConstants(t *testing.T) {
	errors := []struct {
		err      error
		expected string
	}{
		{ErrUserNotFound, "user not found"},
		{ErrDuplicateUsername, "username already exists"},
		{ErrDuplicateEmail, "email already exists"},
		{ErrGroupNotFound, "group not found"},
		{ErrDuplicateGroup, "group already exists"},
	}

	for _, tc := range errors {
		t.Run(tc.expected, func(t *testing.T) {
			assert.Equal(t, tc.expected, tc.err.Error())
		})
	}
}

// TestUserStructComplete tests User struct with all fields
func TestUserStructComplete(t *testing.T) {
	now := time.Now()
	lastLogin := now.Add(-24 * time.Hour)

	user := User{
		ID:          "user-123",
		Username:    "testuser",
		Email:       "test@example.com",
		DisplayName: "Test User",
		Roles:       []UserRole{UserRoleUser, UserRolePowerUser},
		Provider:    ProviderLocal,
		ProviderID:  "local-456",
		Attributes: map[string]interface{}{
			"department": "Engineering",
			"location":   "San Francisco",
		},
		Groups:    []string{"developers", "admins"},
		CreatedAt: now,
		UpdatedAt: now,
		LastLogin: &lastLogin,
		Enabled:   true,
	}

	assert.Equal(t, "user-123", user.ID)
	assert.Equal(t, "testuser", user.Username)
	assert.Equal(t, "test@example.com", user.Email)
	assert.Equal(t, "Test User", user.DisplayName)
	assert.Len(t, user.Roles, 2)
	assert.Contains(t, user.Roles, UserRoleUser)
	assert.Contains(t, user.Roles, UserRolePowerUser)
	assert.Equal(t, ProviderLocal, user.Provider)
	assert.Equal(t, "local-456", user.ProviderID)
	assert.Len(t, user.Attributes, 2)
	assert.Equal(t, "Engineering", user.Attributes["department"])
	assert.Equal(t, "San Francisco", user.Attributes["location"])
	assert.Len(t, user.Groups, 2)
	assert.Contains(t, user.Groups, "developers")
	assert.Contains(t, user.Groups, "admins")
	assert.Equal(t, now, user.CreatedAt)
	assert.Equal(t, now, user.UpdatedAt)
	assert.NotNil(t, user.LastLogin)
	assert.Equal(t, lastLogin, *user.LastLogin)
	assert.True(t, user.Enabled)
}

// TestUserJSONSerialization tests JSON serialization/deserialization
func TestUserJSONSerialization(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second) // Truncate for JSON compatibility
	lastLogin := now.Add(-24 * time.Hour)

	original := User{
		ID:          "user-123",
		Username:    "testuser",
		Email:       "test@example.com",
		DisplayName: "Test User",
		Roles:       []UserRole{UserRoleAdmin},
		Provider:    ProviderOkta,
		ProviderID:  "okta-789",
		Attributes: map[string]interface{}{
			"title": "Senior Engineer",
		},
		Groups:    []string{"engineering"},
		CreatedAt: now,
		UpdatedAt: now,
		LastLogin: &lastLogin,
		Enabled:   true,
	}

	// Serialize to JSON
	jsonData, err := json.Marshal(original)
	require.NoError(t, err)

	// Deserialize from JSON
	var deserialized User
	err = json.Unmarshal(jsonData, &deserialized)
	require.NoError(t, err)

	// Verify all fields match
	assert.Equal(t, original.ID, deserialized.ID)
	assert.Equal(t, original.Username, deserialized.Username)
	assert.Equal(t, original.Email, deserialized.Email)
	assert.Equal(t, original.DisplayName, deserialized.DisplayName)
	assert.Equal(t, original.Roles, deserialized.Roles)
	assert.Equal(t, original.Provider, deserialized.Provider)
	assert.Equal(t, original.ProviderID, deserialized.ProviderID)
	assert.Equal(t, original.Attributes, deserialized.Attributes)
	assert.Equal(t, original.Groups, deserialized.Groups)
	assert.True(t, original.CreatedAt.Equal(deserialized.CreatedAt))
	assert.True(t, original.UpdatedAt.Equal(deserialized.UpdatedAt))
	assert.NotNil(t, deserialized.LastLogin)
	assert.True(t, original.LastLogin.Equal(*deserialized.LastLogin))
	assert.Equal(t, original.Enabled, deserialized.Enabled)
}

// TestUserWithNilLastLogin tests User with nil LastLogin
func TestUserWithNilLastLogin(t *testing.T) {
	user := User{
		ID:       "user-456",
		Username: "newuser",
		Enabled:  false,
		// LastLogin is nil
	}

	jsonData, err := json.Marshal(user)
	require.NoError(t, err)

	var deserialized User
	err = json.Unmarshal(jsonData, &deserialized)
	require.NoError(t, err)

	assert.Nil(t, deserialized.LastLogin)
	assert.False(t, deserialized.Enabled)
}

// TestGroupStructComplete tests Group struct with all fields
func TestGroupStructComplete(t *testing.T) {
	now := time.Now()

	group := Group{
		ID:          "group-123",
		Name:        "Engineering",
		Description: "Engineering team members",
		Provider:    ProviderAzureAD,
		ProviderID:  "azure-456",
		Attributes: map[string]interface{}{
			"cost_center": "CC-100",
			"location":    "Remote",
		},
		CreatedAt: now,
		UpdatedAt: now,
	}

	assert.Equal(t, "group-123", group.ID)
	assert.Equal(t, "Engineering", group.Name)
	assert.Equal(t, "Engineering team members", group.Description)
	assert.Equal(t, ProviderAzureAD, group.Provider)
	assert.Equal(t, "azure-456", group.ProviderID)
	assert.Len(t, group.Attributes, 2)
	assert.Equal(t, "CC-100", group.Attributes["cost_center"])
	assert.Equal(t, "Remote", group.Attributes["location"])
	assert.Equal(t, now, group.CreatedAt)
	assert.Equal(t, now, group.UpdatedAt)
}

// TestGroupJSONSerialization tests JSON serialization for Group
func TestGroupJSONSerialization(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)

	original := Group{
		ID:          "group-789",
		Name:        "Marketing",
		Description: "Marketing department",
		Provider:    ProviderGoogleWorkspace,
		ProviderID:  "google-123",
		Attributes: map[string]interface{}{
			"budget": 50000,
		},
		CreatedAt: now,
		UpdatedAt: now,
	}

	jsonData, err := json.Marshal(original)
	require.NoError(t, err)

	var deserialized Group
	err = json.Unmarshal(jsonData, &deserialized)
	require.NoError(t, err)

	assert.Equal(t, original.ID, deserialized.ID)
	assert.Equal(t, original.Name, deserialized.Name)
	assert.Equal(t, original.Description, deserialized.Description)
	assert.Equal(t, original.Provider, deserialized.Provider)
	assert.Equal(t, original.ProviderID, deserialized.ProviderID)
	// JSON serialization converts int to float64, so check the budget value specifically
	assert.Len(t, deserialized.Attributes, 1)
	assert.Equal(t, float64(50000), deserialized.Attributes["budget"])
	assert.True(t, original.CreatedAt.Equal(deserialized.CreatedAt))
	assert.True(t, original.UpdatedAt.Equal(deserialized.UpdatedAt))
}

// TestUserFilterComplete tests UserFilter with all fields
func TestUserFilterComplete(t *testing.T) {
	createdAfter := time.Now().Add(-30 * 24 * time.Hour)
	createdBefore := time.Now()
	updatedAfter := time.Now().Add(-7 * 24 * time.Hour)
	updatedBefore := time.Now()
	lastLoginAfter := time.Now().Add(-24 * time.Hour)
	lastLoginBefore := time.Now()

	filter := UserFilter{
		Username:        "testuser",
		Email:           "test@example.com",
		Role:            UserRolePowerUser,
		Group:           "developers",
		Provider:        ProviderOkta,
		EnabledOnly:     true,
		DisabledOnly:    false,
		CreatedAfter:    &createdAfter,
		CreatedBefore:   &createdBefore,
		UpdatedAfter:    &updatedAfter,
		UpdatedBefore:   &updatedBefore,
		LastLoginAfter:  &lastLoginAfter,
		LastLoginBefore: &lastLoginBefore,
	}

	assert.Equal(t, "testuser", filter.Username)
	assert.Equal(t, "test@example.com", filter.Email)
	assert.Equal(t, UserRolePowerUser, filter.Role)
	assert.Equal(t, "developers", filter.Group)
	assert.Equal(t, ProviderOkta, filter.Provider)
	assert.True(t, filter.EnabledOnly)
	assert.False(t, filter.DisabledOnly)
	assert.NotNil(t, filter.CreatedAfter)
	assert.Equal(t, createdAfter, *filter.CreatedAfter)
	assert.NotNil(t, filter.CreatedBefore)
	assert.Equal(t, createdBefore, *filter.CreatedBefore)
	assert.NotNil(t, filter.UpdatedAfter)
	assert.Equal(t, updatedAfter, *filter.UpdatedAfter)
	assert.NotNil(t, filter.UpdatedBefore)
	assert.Equal(t, updatedBefore, *filter.UpdatedBefore)
	assert.NotNil(t, filter.LastLoginAfter)
	assert.Equal(t, lastLoginAfter, *filter.LastLoginAfter)
	assert.NotNil(t, filter.LastLoginBefore)
	assert.Equal(t, lastLoginBefore, *filter.LastLoginBefore)
}

// TestUserFilterJSONSerialization tests JSON serialization for UserFilter
func TestUserFilterJSONSerialization(t *testing.T) {
	createdAfter := time.Now().UTC().Truncate(time.Second)

	original := UserFilter{
		Username:     "filtertest",
		Email:        "filter@test.com",
		Role:         UserRoleAdmin,
		Provider:     ProviderLocal,
		EnabledOnly:  true,
		CreatedAfter: &createdAfter,
	}

	jsonData, err := json.Marshal(original)
	require.NoError(t, err)

	var deserialized UserFilter
	err = json.Unmarshal(jsonData, &deserialized)
	require.NoError(t, err)

	assert.Equal(t, original.Username, deserialized.Username)
	assert.Equal(t, original.Email, deserialized.Email)
	assert.Equal(t, original.Role, deserialized.Role)
	assert.Equal(t, original.Provider, deserialized.Provider)
	assert.Equal(t, original.EnabledOnly, deserialized.EnabledOnly)
	assert.NotNil(t, deserialized.CreatedAfter)
	assert.True(t, original.CreatedAfter.Equal(*deserialized.CreatedAfter))
}

// TestGroupFilterComplete tests GroupFilter with all fields
func TestGroupFilterComplete(t *testing.T) {
	createdAfter := time.Now().Add(-30 * 24 * time.Hour)
	createdBefore := time.Now()

	filter := GroupFilter{
		Name:          "Engineering",
		Provider:      ProviderAWSSO,
		CreatedAfter:  &createdAfter,
		CreatedBefore: &createdBefore,
	}

	assert.Equal(t, "Engineering", filter.Name)
	assert.Equal(t, ProviderAWSSO, filter.Provider)
	assert.NotNil(t, filter.CreatedAfter)
	assert.Equal(t, createdAfter, *filter.CreatedAfter)
	assert.NotNil(t, filter.CreatedBefore)
	assert.Equal(t, createdBefore, *filter.CreatedBefore)
}

// TestPaginationOptions tests PaginationOptions
func TestPaginationOptions(t *testing.T) {
	pagination := PaginationOptions{
		Page:      2,
		PageSize:  25,
		SortBy:    "username",
		SortOrder: "asc",
	}

	assert.Equal(t, 2, pagination.Page)
	assert.Equal(t, 25, pagination.PageSize)
	assert.Equal(t, "username", pagination.SortBy)
	assert.Equal(t, "asc", pagination.SortOrder)
}

// TestPaginationOptionsJSON tests JSON serialization for PaginationOptions
func TestPaginationOptionsJSON(t *testing.T) {
	original := PaginationOptions{
		Page:      1,
		PageSize:  50,
		SortBy:    "created_at",
		SortOrder: "desc",
	}

	jsonData, err := json.Marshal(original)
	require.NoError(t, err)

	var deserialized PaginationOptions
	err = json.Unmarshal(jsonData, &deserialized)
	require.NoError(t, err)

	assert.Equal(t, original, deserialized)
}

// TestUserProvisionOptions tests UserProvisionOptions with all fields
func TestUserProvisionOptions(t *testing.T) {
	options := UserProvisionOptions{
		DefaultRole:      UserRoleUser,
		AutoCreateGroups: true,
		GroupRoleMapping: map[string]UserRole{
			"admins":     UserRoleAdmin,
			"developers": UserRolePowerUser,
			"users":      UserRoleUser,
		},
		AttributeMapping: map[string]string{
			"given_name":  "first_name",
			"family_name": "last_name",
		},
		AutoProvision:  true,
		RequireGroup:   false,
		AllowedDomains: []string{"company.com", "contractor.com"},
	}

	assert.Equal(t, UserRoleUser, options.DefaultRole)
	assert.True(t, options.AutoCreateGroups)
	assert.Len(t, options.GroupRoleMapping, 3)
	assert.Equal(t, UserRoleAdmin, options.GroupRoleMapping["admins"])
	assert.Equal(t, UserRolePowerUser, options.GroupRoleMapping["developers"])
	assert.Equal(t, UserRoleUser, options.GroupRoleMapping["users"])
	assert.Len(t, options.AttributeMapping, 2)
	assert.Equal(t, "first_name", options.AttributeMapping["given_name"])
	assert.Equal(t, "last_name", options.AttributeMapping["family_name"])
	assert.True(t, options.AutoProvision)
	assert.False(t, options.RequireGroup)
	assert.Len(t, options.AllowedDomains, 2)
	assert.Contains(t, options.AllowedDomains, "company.com")
	assert.Contains(t, options.AllowedDomains, "contractor.com")
}

// TestSyncOptions tests SyncOptions with all fields
func TestSyncOptions(t *testing.T) {
	options := SyncOptions{
		SyncGroups:          true,
		SyncRoles:           true,
		SyncAttributes:      false,
		DisableUnknownUsers: true,
		DisableDeletedUsers: true,
		CreateMissingUsers:  false,
		BatchSize:           100,
	}

	assert.True(t, options.SyncGroups)
	assert.True(t, options.SyncRoles)
	assert.False(t, options.SyncAttributes)
	assert.True(t, options.DisableUnknownUsers)
	assert.True(t, options.DisableDeletedUsers)
	assert.False(t, options.CreateMissingUsers)
	assert.Equal(t, 100, options.BatchSize)
}

// TestPaginatedUsers tests PaginatedUsers struct
func TestPaginatedUsers(t *testing.T) {
	users := []*User{
		{ID: "user1", Username: "user1"},
		{ID: "user2", Username: "user2"},
		{ID: "user3", Username: "user3"},
	}

	paginated := PaginatedUsers{
		Users:      users,
		Total:      150,
		Page:       2,
		PageSize:   25,
		TotalPages: 6,
	}

	assert.Len(t, paginated.Users, 3)
	assert.Equal(t, "user1", paginated.Users[0].Username)
	assert.Equal(t, "user2", paginated.Users[1].Username)
	assert.Equal(t, "user3", paginated.Users[2].Username)
	assert.Equal(t, 150, paginated.Total)
	assert.Equal(t, 2, paginated.Page)
	assert.Equal(t, 25, paginated.PageSize)
	assert.Equal(t, 6, paginated.TotalPages)
}

// TestPaginatedGroups tests PaginatedGroups struct
func TestPaginatedGroups(t *testing.T) {
	groups := []*Group{
		{ID: "group1", Name: "Engineering"},
		{ID: "group2", Name: "Marketing"},
	}

	paginated := PaginatedGroups{
		Groups:     groups,
		Total:      25,
		Page:       1,
		PageSize:   10,
		TotalPages: 3,
	}

	assert.Len(t, paginated.Groups, 2)
	assert.Equal(t, "Engineering", paginated.Groups[0].Name)
	assert.Equal(t, "Marketing", paginated.Groups[1].Name)
	assert.Equal(t, 25, paginated.Total)
	assert.Equal(t, 1, paginated.Page)
	assert.Equal(t, 10, paginated.PageSize)
	assert.Equal(t, 3, paginated.TotalPages)
}

// TestSyncResult tests SyncResult struct
func TestSyncResult(t *testing.T) {
	started := time.Now()
	completed := started.Add(5 * time.Minute)

	result := SyncResult{
		Created:       10,
		Updated:       25,
		Disabled:      5,
		Failed:        2,
		FailedUsers:   []string{"user1", "user2"},
		GroupsCreated: 3,
		GroupsUpdated: 7,
		Started:       started,
		Completed:     completed,
		Duration:      300.5,
	}

	assert.Equal(t, 10, result.Created)
	assert.Equal(t, 25, result.Updated)
	assert.Equal(t, 5, result.Disabled)
	assert.Equal(t, 2, result.Failed)
	assert.Len(t, result.FailedUsers, 2)
	assert.Contains(t, result.FailedUsers, "user1")
	assert.Contains(t, result.FailedUsers, "user2")
	assert.Equal(t, 3, result.GroupsCreated)
	assert.Equal(t, 7, result.GroupsUpdated)
	assert.Equal(t, started, result.Started)
	assert.Equal(t, completed, result.Completed)
	assert.Equal(t, 300.5, result.Duration)
}

// TestAuthenticationResult tests AuthenticationResult
func TestAuthenticationResult(t *testing.T) {
	user := &User{
		ID:       "auth-user",
		Username: "authtest",
	}
	expiresAt := time.Now().Add(1 * time.Hour)

	// Test successful authentication
	successResult := AuthenticationResult{
		Success:   true,
		User:      user,
		Token:     "jwt-token-123",
		ExpiresAt: &expiresAt,
		Attributes: map[string]interface{}{
			"session_id": "sess-456",
		},
	}

	assert.True(t, successResult.Success)
	assert.NotNil(t, successResult.User)
	assert.Equal(t, "auth-user", successResult.User.ID)
	assert.Equal(t, "authtest", successResult.User.Username)
	assert.Equal(t, "jwt-token-123", successResult.Token)
	assert.NotNil(t, successResult.ExpiresAt)
	assert.Equal(t, expiresAt, *successResult.ExpiresAt)
	assert.Len(t, successResult.Attributes, 1)
	assert.Equal(t, "sess-456", successResult.Attributes["session_id"])
	assert.Empty(t, successResult.ErrorMessage)

	// Test failed authentication
	failureResult := AuthenticationResult{
		Success:      false,
		ErrorMessage: "invalid credentials",
	}

	assert.False(t, failureResult.Success)
	assert.Nil(t, failureResult.User)
	assert.Empty(t, failureResult.Token)
	assert.Nil(t, failureResult.ExpiresAt)
	assert.Equal(t, "invalid credentials", failureResult.ErrorMessage)
}

// TestAuthenticationResultJSON tests JSON serialization for AuthenticationResult
func TestAuthenticationResultJSON(t *testing.T) {
	user := &User{ID: "json-user", Username: "jsontest"}
	expiresAt := time.Now().UTC().Truncate(time.Second)

	original := AuthenticationResult{
		Success:   true,
		User:      user,
		Token:     "token-789",
		ExpiresAt: &expiresAt,
		Attributes: map[string]interface{}{
			"role": "admin",
		},
	}

	jsonData, err := json.Marshal(original)
	require.NoError(t, err)

	var deserialized AuthenticationResult
	err = json.Unmarshal(jsonData, &deserialized)
	require.NoError(t, err)

	assert.Equal(t, original.Success, deserialized.Success)
	assert.NotNil(t, deserialized.User)
	assert.Equal(t, original.User.ID, deserialized.User.ID)
	assert.Equal(t, original.User.Username, deserialized.User.Username)
	assert.Equal(t, original.Token, deserialized.Token)
	assert.NotNil(t, deserialized.ExpiresAt)
	assert.True(t, original.ExpiresAt.Equal(*deserialized.ExpiresAt))
	assert.Equal(t, original.Attributes, deserialized.Attributes)
}
