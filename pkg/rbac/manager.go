package rbac

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/scttfrdmn/prism/pkg/seam"
	"github.com/scttfrdmn/prism/pkg/seam/filestore"
)

// Manager handles role-based access control for Prism.
// User identities are profile names (as returned by profile.GetCurrentProfile().Name).
// Assignments are persisted through the seam (design §5) — file-backed on the desktop, and the
// same logic backs the shared cloud, so role bindings are part of the shared state (§4).
type Manager struct {
	roles     map[string]*Role
	userRoles map[string]string // userID -> roleID
	mutex     sync.RWMutex
	store     seam.Store[State]
	scope     seam.Scope // tenancy key; zero Scope on the desktop, per-Principal in the cloud
}

// rbacStateID is the fixed record id for the single RBAC state object.
const rbacStateID = "rbac"

// NewManager creates a Manager with built-in default roles and loads any persisted assignments.
func NewManager() (*Manager, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}

	stateDir := filepath.Join(homeDir, ".prism")
	m := &Manager{
		roles:     make(map[string]*Role),
		userRoles: make(map[string]string),
		store:     filestore.New[State](filepath.Join(stateDir, "rbac")),
	}

	m.registerDefaultRoles()
	// One-time migration of a legacy rbac.json, then load (best-effort: empty on first run).
	if err := m.migrateLegacy(filepath.Join(stateDir, "rbac.json")); err != nil {
		return nil, fmt.Errorf("failed to migrate legacy rbac: %w", err)
	}
	_ = m.load()

	return m, nil
}

// NewManagerWithStore builds a Manager over an injected seam store under the zero Scope (tests /
// single-tenant callers).
func NewManagerWithStore(store seam.Store[State]) *Manager {
	return NewManagerForScope(store, seam.Scope{})
}

// NewManagerForScope builds a Manager over an injected store scoped to a Principal — the
// cloud/multi-tenant entry point. The role-binding state partitions by scope; logic is unchanged.
func NewManagerForScope(store seam.Store[State], scope seam.Scope) *Manager {
	m := &Manager{
		roles:     make(map[string]*Role),
		userRoles: make(map[string]string),
		store:     store,
		scope:     scope,
	}
	m.registerDefaultRoles()
	_ = m.load()
	return m
}

// migrateLegacy imports a pre-seam rbac.json into the store, then retires it. Absent → no-op.
func (m *Manager) migrateLegacy(legacyPath string) error {
	// #nosec G304 G703 -- legacyPath is the manager's own ~/.prism/rbac.json, composed internally.
	data, err := os.ReadFile(legacyPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	var state State
	if err := json.Unmarshal(data, &state); err != nil {
		return fmt.Errorf("parse legacy rbac: %w", err)
	}
	if err := m.store.Put(context.Background(), m.scope, rbacStateID, state); err != nil {
		return err
	}
	return os.Rename(legacyPath, legacyPath+".migrated")
}

func (m *Manager) registerDefaultRoles() {
	now := time.Now()

	m.roles[RoleAdmin] = &Role{
		ID:          RoleAdmin,
		Name:        "Administrator",
		Description: "Full access to all resources and operations",
		Permissions: []string{"*"},
		CreatedAt:   now,
	}

	m.roles[RoleResearcher] = &Role{
		ID:          RoleResearcher,
		Name:        "Researcher",
		Description: "Launch and manage instances and storage",
		Permissions: []string{
			ActionInstancesLaunch, ActionInstancesStop, ActionInstancesStart,
			ActionInstancesTerminate, ActionInstancesView, ActionInstancesConnect,
			ActionStorageCreate, ActionStorageAttach, ActionStorageDetach, ActionStorageView,
			ActionTemplatesView, ActionProfilesView, ActionProfilesManage,
		},
		CreatedAt: now,
	}

	m.roles[RoleStudent] = &Role{
		ID:          RoleStudent,
		Name:        "Student",
		Description: "Launch and stop instances using approved templates; no termination or storage deletion",
		Permissions: []string{
			ActionInstancesLaunch, ActionInstancesStop, ActionInstancesStart,
			ActionInstancesView, ActionInstancesConnect,
			ActionStorageAttach, ActionStorageDetach, ActionStorageView,
			ActionTemplatesView,
		},
		CreatedAt: now,
	}

	m.roles[RoleViewer] = &Role{
		ID:          RoleViewer,
		Name:        "Viewer",
		Description: "Read-only access to all resources",
		Permissions: []string{
			ActionInstancesView, ActionStorageView, ActionTemplatesView, ActionProfilesView,
		},
		CreatedAt: now,
	}
}

// AssignRole assigns a named role to a user ID, persisting the assignment.
func (m *Manager) AssignRole(userID, roleID, assignedBy string) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if _, exists := m.roles[roleID]; !exists {
		return fmt.Errorf("role %q not found", roleID)
	}
	m.userRoles[userID] = roleID
	return m.save()
}

// RemoveRole removes any role assignment for the user, reverting to the default researcher role.
func (m *Manager) RemoveRole(userID string) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	delete(m.userRoles, userID)
	return m.save()
}

// GetUserRole returns the role for a user. Defaults to the researcher role if no assignment exists.
func (m *Manager) GetUserRole(userID string) *Role {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	roleID, exists := m.userRoles[userID]
	if !exists {
		return m.roles[RoleResearcher]
	}
	if role, ok := m.roles[roleID]; ok {
		return role
	}
	return m.roles[RoleResearcher]
}

// CanPerformAction returns (allowed, reason) for the given user and action string.
// action format: "instances:launch", "storage:delete", etc.
func (m *Manager) CanPerformAction(userID, action string) (bool, string) {
	role := m.GetUserRole(userID)
	if role == nil {
		return false, "no role assigned"
	}

	for _, perm := range role.Permissions {
		if perm == "*" {
			return true, fmt.Sprintf("allowed by role %q (full access)", role.Name)
		}
		if perm == action {
			return true, fmt.Sprintf("allowed by role %q", role.Name)
		}
		// Wildcard segment: "instances:*" matches "instances:launch"
		if strings.HasSuffix(perm, ":*") {
			prefix := strings.TrimSuffix(perm, ":*")
			if strings.HasPrefix(action, prefix+":") {
				return true, fmt.Sprintf("allowed by role %q", role.Name)
			}
		}
	}

	return false, fmt.Sprintf("action %q not permitted for role %q — contact your administrator", action, role.Name)
}

// ListRoles returns all available roles (built-in and any custom ones).
func (m *Manager) ListRoles() []*Role {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	roles := make([]*Role, 0, len(m.roles))
	for _, r := range m.roles {
		copy := *r
		roles = append(roles, &copy)
	}
	return roles
}

// ListUserAssignments returns all explicit user→role mappings.
func (m *Manager) ListUserAssignments() map[string]string {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	result := make(map[string]string, len(m.userRoles))
	for k, v := range m.userRoles {
		result[k] = v
	}
	return result
}

// persistence

// State is the persisted RBAC record: the single role-binding object stored through the seam. It
// is exported so seam wiring (e.g. the DynamoDB-backed daemon path) can name the store's type
// parameter — `seam.Store[rbac.State]`.
type State struct {
	UserRoles map[string]string `json:"user_roles"`
}

func (m *Manager) save() error {
	return m.store.Put(context.Background(), m.scope, rbacStateID, State{UserRoles: m.userRoles})
}

func (m *Manager) load() error {
	state, err := m.store.Get(context.Background(), m.scope, rbacStateID)
	if err != nil {
		if errors.Is(err, seam.ErrNotFound) {
			return nil
		}
		return err
	}
	if state.UserRoles != nil {
		m.userRoles = state.UserRoles
	}
	return nil
}
