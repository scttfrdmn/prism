package daemon

import (
	"context"
	"errors"
	"sync"

	"github.com/scttfrdmn/prism/pkg/usermgmt"
)

var ErrUserManagerNotInitialized = errors.New("user manager not initialized")

// UserManager manages user-related operations in the daemon
type UserManager struct {
	// service is the user management service
	service usermgmt.UserManagementService

	// storage is the user storage
	storage usermgmt.UserStorage

	// initialized tracks if the user manager has been initialized
	initialized bool

	// mutex protects concurrent access
	mutex sync.RWMutex
}

// NewUserManager creates a new user manager
func NewUserManager() *UserManager {
	return &UserManager{
		initialized: false,
	}
}

// Initialize initializes the user manager
func (m *UserManager) Initialize() error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if m.initialized {
		return nil
	}

	// Create in-memory storage for now
	// In a production environment, this would use persistent storage
	storage := usermgmt.NewMemoryUserStorage()

	// Create user management service
	service := usermgmt.NewUserManagementService(storage)

	m.storage = storage
	m.service = service
	m.initialized = true

	return nil
}

// RegisterProvider registers a user management provider
func (m *UserManager) RegisterProvider(provider usermgmt.UserManagementProvider) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if !m.initialized {
		return ErrUserManagerNotInitialized
	}

	return m.service.RegisterProvider(provider)
}

// UnregisterProvider unregisters a user management provider
func (m *UserManager) UnregisterProvider(providerType usermgmt.Provider) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if !m.initialized {
		return ErrUserManagerNotInitialized
	}

	return m.service.UnregisterProvider(providerType)
}

// Authenticate authenticates a user with the given credentials
func (m *UserManager) Authenticate(ctx context.Context, username, password string) (*usermgmt.AuthenticationResult, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	if !m.initialized {
		return nil, ErrUserManagerNotInitialized
	}

	return m.service.Authenticate(username, password)
}

// GetUser gets a user by ID
func (m *UserManager) GetUser(ctx context.Context, id string) (*usermgmt.User, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	if !m.initialized {
		return nil, ErrUserManagerNotInitialized
	}

	return m.service.GetUser(id)
}

// GetUserByUsername gets a user by username
func (m *UserManager) GetUserByUsername(ctx context.Context, username string) (*usermgmt.User, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	if !m.initialized {
		return nil, ErrUserManagerNotInitialized
	}

	return m.service.GetUserByUsername(username)
}

// GetUsers gets users matching the specified filter
func (m *UserManager) GetUsers(ctx context.Context, filter *usermgmt.UserFilter, pagination *usermgmt.PaginationOptions) (*usermgmt.PaginatedUsers, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	if !m.initialized {
		return nil, ErrUserManagerNotInitialized
	}

	return m.service.ListUsers(filter, pagination)
}

// CreateUser creates a new user
func (m *UserManager) CreateUser(ctx context.Context, user *usermgmt.User) (*usermgmt.User, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	if !m.initialized {
		return nil, ErrUserManagerNotInitialized
	}

	err := m.service.CreateUser(user)
	return user, err
}

// UpdateUser updates an existing user
func (m *UserManager) UpdateUser(ctx context.Context, user *usermgmt.User) (*usermgmt.User, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	if !m.initialized {
		return nil, ErrUserManagerNotInitialized
	}

	err := m.service.UpdateUser(user)
	return user, err
}

// DeleteUser deletes a user
func (m *UserManager) DeleteUser(ctx context.Context, id string) error {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	if !m.initialized {
		return ErrUserManagerNotInitialized
	}

	return m.service.DeleteUser(id)
}

// EnableUser enables a user
func (m *UserManager) EnableUser(ctx context.Context, id string) error {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	if !m.initialized {
		return ErrUserManagerNotInitialized
	}

	return m.service.EnableUser(id)
}

// DisableUser disables a user
func (m *UserManager) DisableUser(ctx context.Context, id string) error {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	if !m.initialized {
		return ErrUserManagerNotInitialized
	}

	return m.service.DisableUser(id)
}

// SynchronizeUsers synchronizes users from all providers
func (m *UserManager) SynchronizeUsers(ctx context.Context, options *usermgmt.SyncOptions) (*usermgmt.SyncResult, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	if !m.initialized {
		return nil, ErrUserManagerNotInitialized
	}

	return m.service.SynchronizeUsers(options)
}

// SetDefaultProvisionOptions sets the default options for provisioning users
func (m *UserManager) SetDefaultProvisionOptions(options *usermgmt.UserProvisionOptions) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if !m.initialized {
		return ErrUserManagerNotInitialized
	}

	_ = m.service.SetDefaultProvisionOptions(options)
	return nil
}

// GetDefaultProvisionOptions gets the default options for provisioning users
func (m *UserManager) GetDefaultProvisionOptions() (*usermgmt.UserProvisionOptions, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	if !m.initialized {
		return nil, ErrUserManagerNotInitialized
	}

	return m.service.GetDefaultProvisionOptions()
}

// IsUserInRole checks if a user has the specified role
func (m *UserManager) IsUserInRole(ctx context.Context, userID string, role usermgmt.UserRole) (bool, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	if !m.initialized {
		return false, ErrUserManagerNotInitialized
	}

	// Get user
	user, err := m.service.GetUser(userID)
	if err != nil {
		return false, err
	}

	// Check if user has role
	for _, userRole := range user.Roles {
		if userRole == role {
			return true, nil
		}
	}

	return false, nil
}

// Close closes the user manager and releases resources
func (m *UserManager) Close() error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if !m.initialized {
		return nil
	}

	if err := m.service.Close(); err != nil {
		return err
	}

	m.initialized = false
	return nil
}
