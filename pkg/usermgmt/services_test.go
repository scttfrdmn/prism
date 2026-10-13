package usermgmt

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// TestNewMemoryUserStorage tests memory storage creation
func TestNewMemoryUserStorage(t *testing.T) {
	storage := NewMemoryUserStorage()
	assert.NotNil(t, storage)

	// Verify it implements UserStorage interface
	var _ UserStorage = storage
}

// TestNewUserManagementService tests service creation
func TestNewUserManagementService(t *testing.T) {
	storage := NewMemoryUserStorage()
	service := NewUserManagementService(storage)
	assert.NotNil(t, service)

	// Verify it implements UserManagementService interface
	var _ UserManagementService = service
}

// TestMemoryUserStorageUserOperations tests user operations on memory storage
func TestMemoryUserStorageUserOperations(t *testing.T) {
	storage := NewMemoryUserStorage()

	// Test StoreUser - should not error for placeholder implementation
	user := &User{
		ID:       "test-user-1",
		Username: "testuser",
		Email:    "test@example.com",
	}

	err := storage.StoreUser(user)
	assert.NoError(t, err)

	// Test RetrieveUser - should successfully retrieve stored user
	retrievedUser, err := storage.RetrieveUser("test-user-1")
	assert.NoError(t, err)
	assert.NotNil(t, retrievedUser)
	assert.Equal(t, "test-user-1", retrievedUser.ID)
	assert.Equal(t, "testuser", retrievedUser.Username)

	// Test UpdateUser - should update existing user
	user.Email = "updated@example.com"
	err = storage.UpdateUser(user)
	assert.NoError(t, err)

	// Verify update
	retrievedUser, err = storage.RetrieveUser("test-user-1")
	assert.NoError(t, err)
	assert.Equal(t, "updated@example.com", retrievedUser.Email)

	// Test ListUsers - should return stored user
	users, err := storage.ListUsers(&UserFilter{})
	assert.NoError(t, err)
	assert.NotNil(t, users)
	assert.Len(t, users, 1)

	// Test DeleteUser - should delete the user
	err = storage.DeleteUser("test-user-1")
	assert.NoError(t, err)

	// Verify deletion - should now return error
	retrievedUser, err = storage.RetrieveUser("test-user-1")
	assert.Error(t, err)
	assert.Equal(t, ErrUserNotFound, err)
	assert.Nil(t, retrievedUser)
}

// TestMemoryUserStorageGroupOperations tests group operations on memory storage
func TestMemoryUserStorageGroupOperations(t *testing.T) {
	storage := NewMemoryUserStorage()

	// Test StoreGroup - should not error for placeholder implementation
	group := &Group{
		ID:   "test-group-1",
		Name: "Test Group",
	}

	err := storage.StoreGroup(group)
	assert.NoError(t, err)

	// Test RetrieveGroup - should successfully retrieve stored group
	retrievedGroup, err := storage.RetrieveGroup("test-group-1")
	assert.NoError(t, err)
	assert.NotNil(t, retrievedGroup)
	assert.Equal(t, "test-group-1", retrievedGroup.ID)
	assert.Equal(t, "Test Group", retrievedGroup.Name)

	// Test UpdateGroup - should update existing group
	group.Description = "Updated description"
	err = storage.UpdateGroup(group)
	assert.NoError(t, err)

	// Verify update
	retrievedGroup, err = storage.RetrieveGroup("test-group-1")
	assert.NoError(t, err)
	assert.Equal(t, "Updated description", retrievedGroup.Description)

	// Test ListGroups - should return stored group
	groups, err := storage.ListGroups(&GroupFilter{})
	assert.NoError(t, err)
	assert.NotNil(t, groups)
	assert.Len(t, groups, 1)

	// Test DeleteGroup - should delete the group
	err = storage.DeleteGroup("test-group-1")
	assert.NoError(t, err)

	// Verify deletion - should now return error
	retrievedGroup, err = storage.RetrieveGroup("test-group-1")
	assert.Error(t, err)
	assert.Equal(t, ErrGroupNotFound, err)
	assert.Nil(t, retrievedGroup)
}

// TestMemoryUserStorageGroupMembership tests group membership operations
func TestMemoryUserStorageGroupMembership(t *testing.T) {
	storage := NewMemoryUserStorage()

	// Create user and group first
	user := &User{
		ID:       "user-1",
		Username: "testuser",
		Email:    "test@example.com",
	}
	err := storage.StoreUser(user)
	assert.NoError(t, err)

	group := &Group{
		ID:   "group-1",
		Name: "Test Group",
	}
	err = storage.StoreGroup(group)
	assert.NoError(t, err)

	// Test StoreUserGroupMembership - should add membership
	err = storage.StoreUserGroupMembership("user-1", "group-1")
	assert.NoError(t, err)

	// Test GetUserGroups - should return the group
	userGroups, err := storage.GetUserGroups("user-1")
	assert.NoError(t, err)
	assert.NotNil(t, userGroups)
	assert.Len(t, userGroups, 1)

	// Test GetGroupUsers - should return the user
	groupUsers, err := storage.GetGroupUsers("group-1")
	assert.NoError(t, err)
	assert.NotNil(t, groupUsers)
	assert.Len(t, groupUsers, 1)

	// Test RemoveUserGroupMembership - should remove membership
	err = storage.RemoveUserGroupMembership("user-1", "group-1")
	assert.NoError(t, err)

	// Verify removal
	userGroups, err = storage.GetUserGroups("user-1")
	assert.NoError(t, err)
	assert.Empty(t, userGroups)
}

// TestUserManagementServiceUserOperations tests user operations on service
func TestUserManagementServiceUserOperations(t *testing.T) {
	storage := NewMemoryUserStorage()
	service := NewUserManagementService(storage)

	// Test CreateUser - should not error for placeholder implementation
	user := &User{
		ID:       "service-user-1",
		Username: "serviceuser",
		Email:    "service@example.com",
		Enabled:  true,
	}

	err := service.CreateUser(user)
	assert.NoError(t, err)

	// Test GetUser - should successfully retrieve created user
	retrievedUser, err := service.GetUser("service-user-1")
	assert.NoError(t, err)
	assert.NotNil(t, retrievedUser)
	assert.Equal(t, "service-user-1", retrievedUser.ID)

	// Test GetUserByUsername - should retrieve by username
	userByUsername, err := service.GetUserByUsername("serviceuser")
	assert.NoError(t, err)
	assert.NotNil(t, userByUsername)
	assert.Equal(t, "serviceuser", userByUsername.Username)

	// Test GetUserByEmail - should retrieve by email
	userByEmail, err := service.GetUserByEmail("service@example.com")
	assert.NoError(t, err)
	assert.NotNil(t, userByEmail)
	assert.Equal(t, "service@example.com", userByEmail.Email)

	// Test UpdateUser - should update the user
	user.Email = "updated@example.com"
	err = service.UpdateUser(user)
	assert.NoError(t, err)

	// Verify update
	retrievedUser, err = service.GetUser("service-user-1")
	assert.NoError(t, err)
	assert.Equal(t, "updated@example.com", retrievedUser.Email)

	// Test ListUsers - should return created user
	pagination := &PaginationOptions{
		Page:     1,
		PageSize: 10,
	}
	paginatedUsers, err := service.ListUsers(&UserFilter{}, pagination)
	assert.NoError(t, err)
	assert.NotNil(t, paginatedUsers)
	assert.Len(t, paginatedUsers.Users, 1)
	assert.Equal(t, 1, paginatedUsers.Total)

	// Test DeleteUser - should delete the user
	err = service.DeleteUser("service-user-1")
	assert.NoError(t, err)

	// Verify deletion
	retrievedUser, err = service.GetUser("service-user-1")
	assert.Error(t, err)
	assert.Equal(t, ErrUserNotFound, err)
	assert.Nil(t, retrievedUser, "a deleted user must not be returned")
}

// TestUserManagementServiceGroupOperations tests group operations on service
func TestUserManagementServiceGroupOperations(t *testing.T) {
	storage := NewMemoryUserStorage()
	service := NewUserManagementService(storage)

	// Test CreateGroup - should not error for placeholder implementation
	group := &Group{
		ID:   "service-group-1",
		Name: "Service Group",
	}

	err := service.CreateGroup(group)
	assert.NoError(t, err)

	// Test GetGroup - should successfully retrieve created group
	retrievedGroup, err := service.GetGroup("service-group-1")
	assert.NoError(t, err)
	assert.NotNil(t, retrievedGroup)
	assert.Equal(t, "service-group-1", retrievedGroup.ID)

	// Test GetGroupByName - should retrieve by name
	groupByName, err := service.GetGroupByName("Service Group")
	assert.NoError(t, err)
	assert.NotNil(t, groupByName)
	assert.Equal(t, "Service Group", groupByName.Name)

	// Test UpdateGroup - should update the group
	group.Description = "Updated description"
	err = service.UpdateGroup(group)
	assert.NoError(t, err)

	// Verify update
	retrievedGroup, err = service.GetGroup("service-group-1")
	assert.NoError(t, err)
	assert.Equal(t, "Updated description", retrievedGroup.Description)

	// Test ListGroups - should return created group
	pagination := &PaginationOptions{
		Page:     1,
		PageSize: 10,
	}
	paginatedGroups, err := service.ListGroups(&GroupFilter{}, pagination)
	assert.NoError(t, err)
	assert.NotNil(t, paginatedGroups)
	assert.Len(t, paginatedGroups.Groups, 1)
	assert.Equal(t, 1, paginatedGroups.Total)

	// Test GetGroups - simplified list method
	groups, err := service.GetGroups()
	assert.NoError(t, err)
	assert.NotNil(t, groups)
	assert.Len(t, groups, 1)

	// Test DeleteGroup - should delete the group
	err = service.DeleteGroup("service-group-1")
	assert.NoError(t, err)

	// Verify deletion
	retrievedGroup, err = service.GetGroup("service-group-1")
	assert.Error(t, err)
	assert.Equal(t, ErrGroupNotFound, err)
	assert.Nil(t, retrievedGroup, "a deleted group must not be returned")
}

// TestUserManagementServiceUserGroupOperations tests user-group operations
func TestUserManagementServiceUserGroupOperations(t *testing.T) {
	storage := NewMemoryUserStorage()
	service := NewUserManagementService(storage)

	// Create user and group first
	user := &User{
		ID:       "user-1",
		Username: "testuser",
		Email:    "test@example.com",
		Enabled:  true,
	}
	err := service.CreateUser(user)
	assert.NoError(t, err)

	group := &Group{
		ID:   "group-1",
		Name: "Test Group",
	}
	err = service.CreateGroup(group)
	assert.NoError(t, err)

	// Test AddUserToGroup - should add user to group
	err = service.AddUserToGroup("user-1", "group-1")
	assert.NoError(t, err)

	// Test GetUserGroups - should return the group
	userGroups, err := service.GetUserGroups("user-1")
	assert.NoError(t, err)
	assert.NotNil(t, userGroups)
	assert.Len(t, userGroups, 1)

	// Test GetGroupUsers - should return the user
	groupUsers, err := service.GetGroupUsers("group-1")
	assert.NoError(t, err)
	assert.NotNil(t, groupUsers)
	assert.Len(t, groupUsers, 1)

	// Test RemoveUserFromGroup - should remove user from group
	err = service.RemoveUserFromGroup("user-1", "group-1")
	assert.NoError(t, err)

	// Verify removal
	userGroups, err = service.GetUserGroups("user-1")
	assert.NoError(t, err)
	assert.Empty(t, userGroups)
}

// TestUserManagementServiceSyncOperations tests sync operations
func TestUserManagementServiceSyncOperations(t *testing.T) {
	storage := NewMemoryUserStorage()
	service := NewUserManagementService(storage)

	// Test SyncUsers - placeholder returns empty result
	syncOptions := &SyncOptions{
		SyncGroups:         true,
		SyncRoles:          true,
		CreateMissingUsers: true,
		BatchSize:          50,
	}

	syncResult, err := service.SyncUsers(syncOptions)
	assert.NoError(t, err)
	assert.NotNil(t, syncResult)
	assert.Equal(t, 0, syncResult.Created)
	assert.Equal(t, 0, syncResult.Updated)

	// Test SynchronizeUsers (alternative method) - placeholder returns empty result
	syncResult2, err := service.SynchronizeUsers(syncOptions)
	assert.NoError(t, err)
	assert.NotNil(t, syncResult2)

	// Test ProvisionUser - returns error for unimplemented provisioning
	provisionedUser, err := service.ProvisionUser(map[string]interface{}{
		"username": "provisioned",
		"email":    "provisioned@example.com",
	}, &UserProvisionOptions{
		DefaultRole: UserRoleUser,
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not implemented")
	assert.Nil(t, provisionedUser)
}

// TestUserManagementServiceProviderOperations tests provider operations
func TestUserManagementServiceProviderOperations(t *testing.T) {
	storage := NewMemoryUserStorage()
	service := NewUserManagementService(storage)

	// Create a mock provider for testing
	mockProvider := &MockUserManagementProvider{
		providerType: ProviderOkta,
	}

	// Test RegisterProvider - should not error for placeholder implementation
	err := service.RegisterProvider(mockProvider)
	assert.NoError(t, err)

	// Test UnregisterProvider - should not error for placeholder implementation
	err = service.UnregisterProvider(ProviderOkta)
	assert.NoError(t, err)
}

// TestUserManagementServiceAuthentication tests authentication operations
func TestUserManagementServiceAuthentication(t *testing.T) {
	storage := NewMemoryUserStorage()
	service := NewUserManagementService(storage)

	// Test Authenticate - returns failure for invalid credentials
	authResult, err := service.Authenticate("testuser", "password123")
	assert.NoError(t, err)
	assert.NotNil(t, authResult)
	assert.False(t, authResult.Success)
	assert.Contains(t, authResult.ErrorMessage, "invalid username or password")
	assert.Nil(t, authResult.User)
	assert.Empty(t, authResult.Token)
}

// TestUserManagementServiceUserManagement tests user management operations
func TestUserManagementServiceUserManagement(t *testing.T) {
	storage := NewMemoryUserStorage()
	service := NewUserManagementService(storage)

	// Create user first
	user := &User{
		ID:       "user-1",
		Username: "testuser",
		Email:    "test@example.com",
		Enabled:  true,
	}
	err := service.CreateUser(user)
	assert.NoError(t, err)

	// Test DisableUser - should disable the user
	err = service.DisableUser("user-1")
	assert.NoError(t, err)

	// Verify user is disabled
	retrievedUser, err := service.GetUser("user-1")
	assert.NoError(t, err)
	assert.False(t, retrievedUser.Enabled)

	// Test EnableUser - should enable the user
	err = service.EnableUser("user-1")
	assert.NoError(t, err)

	// Verify user is enabled
	retrievedUser, err = service.GetUser("user-1")
	assert.NoError(t, err)
	assert.True(t, retrievedUser.Enabled)
}

// TestUserManagementServiceProvisionOptions tests provision options management
func TestUserManagementServiceProvisionOptions(t *testing.T) {
	storage := NewMemoryUserStorage()
	service := NewUserManagementService(storage)

	// Test SetDefaultProvisionOptions - should not error for placeholder implementation
	options := &UserProvisionOptions{
		DefaultRole:    UserRoleUser,
		AutoProvision:  true,
		RequireGroup:   false,
		AllowedDomains: []string{"company.com"},
	}

	err := service.SetDefaultProvisionOptions(options)
	assert.NoError(t, err)

	// Test GetDefaultProvisionOptions - placeholder returns empty options
	retrievedOptions, err := service.GetDefaultProvisionOptions()
	assert.NoError(t, err)
	assert.NotNil(t, retrievedOptions)
	// Placeholder returns empty options
	assert.Equal(t, UserRole(""), retrievedOptions.DefaultRole)
}

// TestUserManagementServiceClose tests service close
func TestUserManagementServiceClose(t *testing.T) {
	storage := NewMemoryUserStorage()
	service := NewUserManagementService(storage)

	// Test Close - should not error for placeholder implementation
	err := service.Close()
	assert.NoError(t, err)
}

// MockUserManagementProvider for testing provider interface
type MockUserManagementProvider struct {
	providerType Provider
}

func (m *MockUserManagementProvider) GetProviderType() Provider {
	return m.providerType
}

func (m *MockUserManagementProvider) AuthenticateUser(username, password string) (*AuthenticationResult, error) {
	return &AuthenticationResult{
		Success: true,
		User: &User{
			ID:       "mock-user-1",
			Username: username,
			Provider: m.providerType,
		},
		Token: "mock-token",
	}, nil
}

func (m *MockUserManagementProvider) SyncUsers(options *SyncOptions) (*SyncResult, error) {
	return &SyncResult{
		Created:   5,
		Updated:   10,
		Disabled:  2,
		Failed:    0,
		Started:   time.Now(),
		Completed: time.Now().Add(30 * time.Second),
		Duration:  30.0,
	}, nil
}

func (m *MockUserManagementProvider) SyncGroups() error {
	return nil
}

func (m *MockUserManagementProvider) ValidateConfiguration() error {
	return nil
}

func (m *MockUserManagementProvider) TestConnection() error {
	return nil
}

// TestMockProvider tests the mock provider implementation
func TestMockProvider(t *testing.T) {
	provider := &MockUserManagementProvider{
		providerType: ProviderOkta,
	}

	// Test GetProviderType
	assert.Equal(t, ProviderOkta, provider.GetProviderType())

	// Test AuthenticateUser
	authResult, err := provider.AuthenticateUser("testuser", "password")
	assert.NoError(t, err)
	assert.True(t, authResult.Success)
	assert.NotNil(t, authResult.User)
	assert.Equal(t, "testuser", authResult.User.Username)
	assert.Equal(t, ProviderOkta, authResult.User.Provider)
	assert.Equal(t, "mock-token", authResult.Token)

	// Test SyncUsers
	syncResult, err := provider.SyncUsers(&SyncOptions{})
	assert.NoError(t, err)
	assert.NotNil(t, syncResult)
	assert.Equal(t, 5, syncResult.Created)
	assert.Equal(t, 10, syncResult.Updated)
	assert.Equal(t, 2, syncResult.Disabled)
	assert.Equal(t, 0, syncResult.Failed)
	assert.Equal(t, 30.0, syncResult.Duration)

	// Test SyncGroups
	err = provider.SyncGroups()
	assert.NoError(t, err)

	// Test ValidateConfiguration
	err = provider.ValidateConfiguration()
	assert.NoError(t, err)

	// Test TestConnection
	err = provider.TestConnection()
	assert.NoError(t, err)
}

// TestUserManagementProviderInterface tests provider interface compliance
func TestUserManagementProviderInterface(t *testing.T) {
	provider := &MockUserManagementProvider{providerType: ProviderLocal}

	// Verify it implements UserManagementProvider interface
	var _ UserManagementProvider = provider
}

// TestServiceInterfaceCompliance tests interface compliance
func TestServiceInterfaceCompliance(t *testing.T) {
	storage := NewMemoryUserStorage()
	service := NewUserManagementService(storage)

	// Test all interface methods exist and can be called
	// This ensures the placeholder implementation satisfies the interface

	// UserManagementService interface compliance
	var _ UserManagementService = service

	// UserStorage interface compliance
	var _ UserStorage = storage

	// All methods should be callable without panic
	user := &User{ID: "test", Username: "test"}
	group := &Group{ID: "test", Name: "test"}
	filter := &UserFilter{}
	groupFilter := &GroupFilter{}
	pagination := &PaginationOptions{Page: 1, PageSize: 10}
	syncOptions := &SyncOptions{}
	provisionOptions := &UserProvisionOptions{}

	// Test all UserManagementService methods
	service.CreateUser(user)
	service.GetUser("test")
	service.GetUserByUsername("test")
	service.GetUserByEmail("test@example.com")
	service.UpdateUser(user)
	service.DeleteUser("test")
	service.ListUsers(filter, pagination)
	service.CreateGroup(group)
	service.GetGroup("test")
	service.GetGroupByName("test")
	service.UpdateGroup(group)
	service.DeleteGroup("test")
	service.ListGroups(groupFilter, pagination)
	service.GetGroups()
	service.AddUserToGroup("user", "group")
	service.RemoveUserFromGroup("user", "group")
	service.GetUserGroups("user")
	service.GetGroupUsers("group")
	service.SyncUsers(syncOptions)
	service.ProvisionUser(nil, provisionOptions)
	service.RegisterProvider(&MockUserManagementProvider{})
	service.UnregisterProvider(ProviderLocal)
	service.Authenticate("user", "pass")
	service.EnableUser("user")
	service.DisableUser("user")
	service.SynchronizeUsers(syncOptions)
	service.SetDefaultProvisionOptions(provisionOptions)
	service.GetDefaultProvisionOptions()
	service.Close()

	// Test all UserStorage methods
	storage.StoreUser(user)
	storage.RetrieveUser("test")
	storage.UpdateUser(user)
	storage.DeleteUser("test")
	storage.ListUsers(filter)
	storage.StoreGroup(group)
	storage.RetrieveGroup("test")
	storage.UpdateGroup(group)
	storage.DeleteGroup("test")
	storage.ListGroups(groupFilter)
	storage.StoreUserGroupMembership("user", "group")
	storage.RemoveUserGroupMembership("user", "group")
	storage.GetUserGroups("user")
	storage.GetGroupUsers("group")
}
