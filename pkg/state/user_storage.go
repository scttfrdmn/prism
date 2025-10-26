package state

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/scttfrdmn/prism/pkg/usermgmt"
)

// UserState represents the user state stored on disk
type UserState struct {
	// Users is a map of user ID to user
	Users map[string]*usermgmt.User `json:"users"`

	// Groups is a map of group ID to group
	Groups map[string]*usermgmt.Group `json:"groups"`

	// UserGroups is a map of user ID to list of group IDs
	UserGroups map[string][]string `json:"user_groups"`

	// UsersByUsername is a map of username to user ID
	UsersByUsername map[string]string `json:"users_by_username"`

	// UsersByEmail is a map of email to user ID
	UsersByEmail map[string]string `json:"users_by_email"`

	// GroupsByName is a map of group name to group ID
	GroupsByName map[string]string `json:"groups_by_name"`
}

// LoadUserState loads the user state from disk
func (m *Manager) LoadUserState() (*UserState, error) {
	m.userMutex.RLock()
	defer m.userMutex.RUnlock()

	// Check if user state file exists
	if _, err := os.Stat(m.userPath); os.IsNotExist(err) {
		// Return empty state if file doesn't exist
		return &UserState{
			Users:           make(map[string]*usermgmt.User),
			Groups:          make(map[string]*usermgmt.Group),
			UserGroups:      make(map[string][]string),
			UsersByUsername: make(map[string]string),
			UsersByEmail:    make(map[string]string),
			GroupsByName:    make(map[string]string),
		}, nil
	}

	data, err := os.ReadFile(m.userPath)
	if err != nil {
		return nil, err
	}

	var state UserState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, err
	}

	// Ensure maps are initialized (backward compatibility)
	if state.Users == nil {
		state.Users = make(map[string]*usermgmt.User)
	}
	if state.Groups == nil {
		state.Groups = make(map[string]*usermgmt.Group)
	}
	if state.UserGroups == nil {
		state.UserGroups = make(map[string][]string)
	}
	if state.UsersByUsername == nil {
		state.UsersByUsername = make(map[string]string)
	}
	if state.UsersByEmail == nil {
		state.UsersByEmail = make(map[string]string)
	}
	if state.GroupsByName == nil {
		state.GroupsByName = make(map[string]string)
	}

	return &state, nil
}

// SaveUserState saves the user state to disk
func (m *Manager) SaveUserState(state *UserState) error {
	m.userMutex.Lock()
	defer m.userMutex.Unlock()

	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}

	// Write to temporary file first, then rename for atomicity
	tempPath := m.userPath + ".tmp"
	if err := os.WriteFile(tempPath, data, 0644); err != nil {
		return err
	}

	if err := os.Rename(tempPath, m.userPath); err != nil {
		return err
	}

	return nil
}

// Implementation of the usermgmt.UserStorage interface

// GetUser gets a user by ID
func (m *Manager) GetUser(ctx context.Context, id string) (*usermgmt.User, error) {
	state, err := m.LoadUserState()
	if err != nil {
		return nil, err
	}

	user, exists := state.Users[id]
	if !exists {
		return nil, usermgmt.ErrUserNotFound
	}

	return copyUser(user), nil
}

// GetUserByUsername gets a user by username
func (m *Manager) GetUserByUsername(ctx context.Context, username string) (*usermgmt.User, error) {
	state, err := m.LoadUserState()
	if err != nil {
		return nil, err
	}

	userID, exists := state.UsersByUsername[username]
	if !exists {
		return nil, usermgmt.ErrUserNotFound
	}

	return m.GetUser(ctx, userID)
}

// GetUserByEmail gets a user by email
func (m *Manager) GetUserByEmail(ctx context.Context, email string) (*usermgmt.User, error) {
	state, err := m.LoadUserState()
	if err != nil {
		return nil, err
	}

	userID, exists := state.UsersByEmail[email]
	if !exists {
		return nil, usermgmt.ErrUserNotFound
	}

	return m.GetUser(ctx, userID)
}

// GetUsers gets users matching the specified filter
func (m *Manager) GetUsers(ctx context.Context, filter *usermgmt.UserFilter, pagination *usermgmt.PaginationOptions) (*usermgmt.PaginatedUsers, error) {
	state, err := m.LoadUserState()
	if err != nil {
		return nil, err
	}

	var filteredUsers []*usermgmt.User

	// Apply filters
	for _, user := range state.Users {
		if !m.userMatchesFilter(state, user, filter) {
			continue
		}

		filteredUsers = append(filteredUsers, copyUser(user))
	}

	// Apply pagination
	result := &usermgmt.PaginatedUsers{
		Total:      len(filteredUsers),
		Page:       1,
		PageSize:   len(filteredUsers),
		TotalPages: 1,
	}

	if pagination != nil {
		// Calculate pagination
		if pagination.Page < 1 {
			pagination.Page = 1
		}

		if pagination.PageSize < 1 {
			pagination.PageSize = 10
		}

		result.Page = pagination.Page
		result.PageSize = pagination.PageSize
		result.TotalPages = (len(filteredUsers) + pagination.PageSize - 1) / pagination.PageSize

		// Apply pagination
		start := (pagination.Page - 1) * pagination.PageSize
		end := start + pagination.PageSize

		if start >= len(filteredUsers) {
			result.Users = []*usermgmt.User{}
		} else {
			if end > len(filteredUsers) {
				end = len(filteredUsers)
			}

			result.Users = filteredUsers[start:end]
		}
	} else {
		result.Users = filteredUsers
	}

	return result, nil
}

// userMatchesFilter checks if a user matches the filter (SOLID: Single Responsibility)
func (m *Manager) userMatchesFilter(state *UserState, user *usermgmt.User, filter *usermgmt.UserFilter) bool {
	if filter == nil {
		return true
	}

	return m.matchesBasicFilters(user, filter) &&
		m.matchesRoleFilter(user, filter) &&
		m.matchesGroupFilter(state, user, filter) &&
		m.matchesStatusFilters(user, filter) &&
		m.matchesTimeFilters(user, filter)
}

// matchesBasicFilters checks username, email, and provider filters
func (m *Manager) matchesBasicFilters(user *usermgmt.User, filter *usermgmt.UserFilter) bool {
	if filter.Username != "" && user.Username != filter.Username {
		return false
	}
	if filter.Email != "" && user.Email != filter.Email {
		return false
	}
	if filter.Provider != "" && user.Provider != filter.Provider {
		return false
	}
	return true
}

// matchesRoleFilter checks if user has the required role
func (m *Manager) matchesRoleFilter(user *usermgmt.User, filter *usermgmt.UserFilter) bool {
	if filter.Role == "" {
		return true
	}

	for _, role := range user.Roles {
		if role == filter.Role {
			return true
		}
	}
	return false
}

// matchesGroupFilter checks if user belongs to the required group
func (m *Manager) matchesGroupFilter(state *UserState, user *usermgmt.User, filter *usermgmt.UserFilter) bool {
	if filter.Group == "" {
		return true
	}

	groupID, exists := state.GroupsByName[filter.Group]
	if !exists {
		return false
	}

	userGroups, exists := state.UserGroups[user.ID]
	if !exists {
		return false
	}

	for _, g := range userGroups {
		if g == groupID {
			return true
		}
	}
	return false
}

// matchesStatusFilters checks enabled/disabled status filters
func (m *Manager) matchesStatusFilters(user *usermgmt.User, filter *usermgmt.UserFilter) bool {
	if filter.EnabledOnly && !user.Enabled {
		return false
	}
	if filter.DisabledOnly && user.Enabled {
		return false
	}
	return true
}

// matchesTimeFilters checks all time-based filters
func (m *Manager) matchesTimeFilters(user *usermgmt.User, filter *usermgmt.UserFilter) bool {
	return m.matchesCreatedTimeFilters(user, filter) &&
		m.matchesUpdatedTimeFilters(user, filter) &&
		m.matchesLastLoginFilters(user, filter)
}

// matchesCreatedTimeFilters checks creation time filters
func (m *Manager) matchesCreatedTimeFilters(user *usermgmt.User, filter *usermgmt.UserFilter) bool {
	if filter.CreatedAfter != nil && user.CreatedAt.Before(*filter.CreatedAfter) {
		return false
	}
	if filter.CreatedBefore != nil && user.CreatedAt.After(*filter.CreatedBefore) {
		return false
	}
	return true
}

// matchesUpdatedTimeFilters checks update time filters
func (m *Manager) matchesUpdatedTimeFilters(user *usermgmt.User, filter *usermgmt.UserFilter) bool {
	if filter.UpdatedAfter != nil && user.UpdatedAt.Before(*filter.UpdatedAfter) {
		return false
	}
	if filter.UpdatedBefore != nil && user.UpdatedAt.After(*filter.UpdatedBefore) {
		return false
	}
	return true
}

// matchesLastLoginFilters checks last login time filters
func (m *Manager) matchesLastLoginFilters(user *usermgmt.User, filter *usermgmt.UserFilter) bool {
	if filter.LastLoginAfter != nil && (user.LastLogin == nil || user.LastLogin.Before(*filter.LastLoginAfter)) {
		return false
	}
	if filter.LastLoginBefore != nil && (user.LastLogin == nil || user.LastLogin.After(*filter.LastLoginBefore)) {
		return false
	}
	return true
}

// CreateUser creates a new user
func (m *Manager) CreateUser(ctx context.Context, user *usermgmt.User) (*usermgmt.User, error) {
	state, err := m.LoadUserState()
	if err != nil {
		return nil, err
	}

	// Check for duplicate username
	if _, exists := state.UsersByUsername[user.Username]; exists {
		return nil, usermgmt.ErrDuplicateUsername
	}

	// Check for duplicate email
	if user.Email != "" {
		if _, exists := state.UsersByEmail[user.Email]; exists {
			return nil, usermgmt.ErrDuplicateEmail
		}
	}

	// Copy user to prevent modification
	newUser := copyUser(user)

	// Set creation time if not set
	if newUser.CreatedAt.IsZero() {
		newUser.CreatedAt = time.Now()
	}

	if newUser.UpdatedAt.IsZero() {
		newUser.UpdatedAt = newUser.CreatedAt
	}

	// Store user
	state.Users[newUser.ID] = newUser
	state.UsersByUsername[newUser.Username] = newUser.ID

	if newUser.Email != "" {
		state.UsersByEmail[newUser.Email] = newUser.ID
	}

	// Initialize user group membership
	state.UserGroups[newUser.ID] = []string{}

	// Add user to specified groups
	if len(newUser.Groups) > 0 {
		for _, groupName := range newUser.Groups {
			groupID, exists := state.GroupsByName[groupName]
			if !exists {
				// Group doesn't exist, create it
				group := &usermgmt.Group{
					ID:          generateID(),
					Name:        groupName,
					Description: "Auto-created group",
					Provider:    newUser.Provider,
					CreatedAt:   newUser.CreatedAt,
					UpdatedAt:   newUser.CreatedAt,
				}

				if _, err := m.CreateGroup(ctx, group); err != nil {
					return nil, err
				}

				groupID = group.ID
			}

			// Add user to group
			if err := m.AddUserToGroup(ctx, newUser.ID, groupID); err != nil {
				return nil, err
			}
		}
	}

	// Save state
	if err := m.SaveUserState(state); err != nil {
		return nil, err
	}

	return copyUser(newUser), nil
}

// UserUpdateValidator validates user update operations (Single Responsibility - SOLID)
type UserUpdateValidator struct{}

// ValidateUpdate performs all validation checks for user updates
func (v *UserUpdateValidator) ValidateUpdate(state *UserState, user *usermgmt.User) error {
	// Check if user exists
	if _, exists := state.Users[user.ID]; !exists {
		return usermgmt.ErrUserNotFound
	}

	// Check for duplicate username
	if currentID, exists := state.UsersByUsername[user.Username]; exists && currentID != user.ID {
		return usermgmt.ErrDuplicateUsername
	}

	// Check for duplicate email
	if user.Email != "" {
		if currentID, exists := state.UsersByEmail[user.Email]; exists && currentID != user.ID {
			return usermgmt.ErrDuplicateEmail
		}
	}

	return nil
}

// UserMappingUpdater updates username and email mappings (Single Responsibility - SOLID)
type UserMappingUpdater struct{}

// UpdateMappings updates username and email mappings in state
func (u *UserMappingUpdater) UpdateMappings(state *UserState, oldUser, newUser *usermgmt.User) {
	// Update username mapping
	if oldUser.Username != newUser.Username {
		delete(state.UsersByUsername, oldUser.Username)
		state.UsersByUsername[newUser.Username] = newUser.ID
	}

	// Update email mapping
	if oldUser.Email != newUser.Email {
		if oldUser.Email != "" {
			delete(state.UsersByEmail, oldUser.Email)
		}

		if newUser.Email != "" {
			state.UsersByEmail[newUser.Email] = newUser.ID
		}
	}
}

// UserGroupMembershipManager manages user group relationships (Single Responsibility - SOLID)
type UserGroupMembershipManager struct {
	manager *Manager
}

// UpdateGroupMembership synchronizes user group memberships
func (g *UserGroupMembershipManager) UpdateGroupMembership(ctx context.Context, state *UserState, user *usermgmt.User) error {
	if len(user.Groups) == 0 {
		return nil
	}

	// Get current groups
	currentGroups, _ := g.manager.GetUserGroups(ctx, user.ID)
	currentGroupMap := make(map[string]bool)

	for _, group := range currentGroups {
		currentGroupMap[group.Name] = true
	}

	// Add to new groups
	for _, groupName := range user.Groups {
		if !currentGroupMap[groupName] {
			if err := g.addUserToGroup(ctx, state, user, groupName); err != nil {
				return err
			}
		}
		// Mark as processed
		delete(currentGroupMap, groupName)
	}

	// Remove from groups not in new list
	for _, group := range currentGroups {
		if currentGroupMap[group.Name] {
			if err := g.manager.RemoveUserFromGroup(ctx, user.ID, group.ID); err != nil {
				return err
			}
		}
	}

	return nil
}

func (g *UserGroupMembershipManager) addUserToGroup(ctx context.Context, state *UserState, user *usermgmt.User, groupName string) error {
	groupID, exists := state.GroupsByName[groupName]
	if !exists {
		// Group doesn't exist, create it
		group := &usermgmt.Group{
			ID:          generateID(),
			Name:        groupName,
			Description: "Auto-created group",
			Provider:    user.Provider,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		}

		if _, err := g.manager.CreateGroup(ctx, group); err != nil {
			return err
		}

		groupID = group.ID
	}

	// Add user to group
	return g.manager.AddUserToGroup(ctx, user.ID, groupID)
}

// UserUpdateOrchestrator coordinates user update operations (Strategy Pattern - SOLID)
type UserUpdateOrchestrator struct {
	validator      *UserUpdateValidator
	mappingUpdater *UserMappingUpdater
	groupManager   *UserGroupMembershipManager
}

// NewUserUpdateOrchestrator creates user update orchestrator
func NewUserUpdateOrchestrator(manager *Manager) *UserUpdateOrchestrator {
	return &UserUpdateOrchestrator{
		validator:      &UserUpdateValidator{},
		mappingUpdater: &UserMappingUpdater{},
		groupManager:   &UserGroupMembershipManager{manager: manager},
	}
}

// ExecuteUpdate performs complete user update using SOLID strategy pattern
func (o *UserUpdateOrchestrator) ExecuteUpdate(ctx context.Context, state *UserState, user *usermgmt.User, manager *Manager) (*usermgmt.User, error) {
	// Validate update
	if err := o.validator.ValidateUpdate(state, user); err != nil {
		return nil, err
	}

	// Get old user for mapping updates
	oldUser := state.Users[user.ID]

	// Copy user to prevent modification
	updatedUser := copyUser(user)

	// Set update time if not set
	if updatedUser.UpdatedAt.IsZero() {
		updatedUser.UpdatedAt = time.Now()
	}

	// Update mappings
	o.mappingUpdater.UpdateMappings(state, oldUser, updatedUser)

	// Store updated user
	state.Users[updatedUser.ID] = updatedUser

	// Update group membership
	if err := o.groupManager.UpdateGroupMembership(ctx, state, updatedUser); err != nil {
		return nil, err
	}

	// Save state
	if err := manager.SaveUserState(state); err != nil {
		return nil, err
	}

	return copyUser(updatedUser), nil
}

// UpdateUser updates an existing user using Strategy Pattern (SOLID: Single Responsibility)
func (m *Manager) UpdateUser(ctx context.Context, user *usermgmt.User) (*usermgmt.User, error) {
	state, err := m.LoadUserState()
	if err != nil {
		return nil, err
	}

	orchestrator := NewUserUpdateOrchestrator(m)
	return orchestrator.ExecuteUpdate(ctx, state, user, m)
}

// DeleteUser deletes a user
func (m *Manager) DeleteUser(ctx context.Context, id string) error {
	state, err := m.LoadUserState()
	if err != nil {
		return err
	}

	user, exists := state.Users[id]
	if !exists {
		return usermgmt.ErrUserNotFound
	}

	// Remove from all groups
	if groups, err := m.GetUserGroups(ctx, id); err == nil {
		for _, group := range groups {
			_ = m.RemoveUserFromGroup(ctx, id, group.ID)
		}
	}

	// Remove from mappings
	delete(state.UsersByUsername, user.Username)

	if user.Email != "" {
		delete(state.UsersByEmail, user.Email)
	}

	// Remove user
	delete(state.Users, id)
	delete(state.UserGroups, id)

	// Save state
	if err := m.SaveUserState(state); err != nil {
		return err
	}

	return nil
}

// GetGroup gets a group by ID
func (m *Manager) GetGroup(ctx context.Context, id string) (*usermgmt.Group, error) {
	state, err := m.LoadUserState()
	if err != nil {
		return nil, err
	}

	group, exists := state.Groups[id]
	if !exists {
		return nil, usermgmt.ErrGroupNotFound
	}

	return copyGroup(group), nil
}

// GetGroupByName gets a group by name
func (m *Manager) GetGroupByName(ctx context.Context, name string) (*usermgmt.Group, error) {
	state, err := m.LoadUserState()
	if err != nil {
		return nil, err
	}

	groupID, exists := state.GroupsByName[name]
	if !exists {
		return nil, usermgmt.ErrGroupNotFound
	}

	return m.GetGroup(ctx, groupID)
}

// GetGroups gets groups matching the specified filter
func (m *Manager) GetGroups(ctx context.Context, filter *usermgmt.GroupFilter, pagination *usermgmt.PaginationOptions) (*usermgmt.PaginatedGroups, error) {
	state, err := m.LoadUserState()
	if err != nil {
		return nil, err
	}

	var filteredGroups []*usermgmt.Group

	// Apply filters
	for _, group := range state.Groups {
		if !m.groupMatchesFilter(group, filter) {
			continue
		}

		filteredGroups = append(filteredGroups, copyGroup(group))
	}

	// Apply pagination
	result := &usermgmt.PaginatedGroups{
		Total:      len(filteredGroups),
		Page:       1,
		PageSize:   len(filteredGroups),
		TotalPages: 1,
	}

	if pagination != nil {
		// Calculate pagination
		if pagination.Page < 1 {
			pagination.Page = 1
		}

		if pagination.PageSize < 1 {
			pagination.PageSize = 10
		}

		result.Page = pagination.Page
		result.PageSize = pagination.PageSize
		result.TotalPages = (len(filteredGroups) + pagination.PageSize - 1) / pagination.PageSize

		// Apply pagination
		start := (pagination.Page - 1) * pagination.PageSize
		end := start + pagination.PageSize

		if start >= len(filteredGroups) {
			result.Groups = []*usermgmt.Group{}
		} else {
			if end > len(filteredGroups) {
				end = len(filteredGroups)
			}

			result.Groups = filteredGroups[start:end]
		}
	} else {
		result.Groups = filteredGroups
	}

	return result, nil
}

// groupMatchesFilter checks if a group matches the filter
func (m *Manager) groupMatchesFilter(group *usermgmt.Group, filter *usermgmt.GroupFilter) bool {
	if filter == nil {
		return true
	}

	// Name filter
	if filter.Name != "" && group.Name != filter.Name {
		return false
	}

	// Provider filter
	if filter.Provider != "" && group.Provider != filter.Provider {
		return false
	}

	// Time filters
	if filter.CreatedAfter != nil && group.CreatedAt.Before(*filter.CreatedAfter) {
		return false
	}

	if filter.CreatedBefore != nil && group.CreatedAt.After(*filter.CreatedBefore) {
		return false
	}

	return true
}

// CreateGroup creates a new group
func (m *Manager) CreateGroup(ctx context.Context, group *usermgmt.Group) (*usermgmt.Group, error) {
	state, err := m.LoadUserState()
	if err != nil {
		return nil, err
	}

	// Check for duplicate name
	if _, exists := state.GroupsByName[group.Name]; exists {
		return nil, usermgmt.ErrDuplicateGroup
	}

	// Copy group to prevent modification
	newGroup := copyGroup(group)

	// Set creation time if not set
	if newGroup.CreatedAt.IsZero() {
		newGroup.CreatedAt = time.Now()
	}

	if newGroup.UpdatedAt.IsZero() {
		newGroup.UpdatedAt = newGroup.CreatedAt
	}

	// Store group
	state.Groups[newGroup.ID] = newGroup
	state.GroupsByName[newGroup.Name] = newGroup.ID

	// Save state
	if err := m.SaveUserState(state); err != nil {
		return nil, err
	}

	return copyGroup(newGroup), nil
}

// UpdateGroup updates an existing group
func (m *Manager) UpdateGroup(ctx context.Context, group *usermgmt.Group) (*usermgmt.Group, error) {
	state, err := m.LoadUserState()
	if err != nil {
		return nil, err
	}

	// Check if group exists
	if _, exists := state.Groups[group.ID]; !exists {
		return nil, usermgmt.ErrGroupNotFound
	}

	// Check for duplicate name
	if currentID, exists := state.GroupsByName[group.Name]; exists && currentID != group.ID {
		return nil, usermgmt.ErrDuplicateGroup
	}

	// Get old group
	oldGroup := state.Groups[group.ID]

	// Update name mapping
	if oldGroup.Name != group.Name {
		delete(state.GroupsByName, oldGroup.Name)
		state.GroupsByName[group.Name] = group.ID
	}

	// Copy group to prevent modification
	updatedGroup := copyGroup(group)

	// Set update time if not set
	if updatedGroup.UpdatedAt.IsZero() {
		updatedGroup.UpdatedAt = time.Now()
	}

	// Store updated group
	state.Groups[updatedGroup.ID] = updatedGroup

	// Save state
	if err := m.SaveUserState(state); err != nil {
		return nil, err
	}

	return copyGroup(updatedGroup), nil
}

// DeleteGroup deletes a group
func (m *Manager) DeleteGroup(ctx context.Context, id string) error {
	state, err := m.LoadUserState()
	if err != nil {
		return err
	}

	group, exists := state.Groups[id]
	if !exists {
		return usermgmt.ErrGroupNotFound
	}

	// Remove all users from group
	for userID, groupIDs := range state.UserGroups {
		var newGroups []string
		for _, gID := range groupIDs {
			if gID != id {
				newGroups = append(newGroups, gID)
			}
		}
		state.UserGroups[userID] = newGroups
	}

	// Remove from mappings
	delete(state.GroupsByName, group.Name)

	// Remove group
	delete(state.Groups, id)

	// Save state
	if err := m.SaveUserState(state); err != nil {
		return err
	}

	return nil
}

// AddUserToGroup adds a user to a group
func (m *Manager) AddUserToGroup(ctx context.Context, userID, groupID string) error {
	state, err := m.LoadUserState()
	if err != nil {
		return err
	}

	// Check if user exists
	if _, exists := state.Users[userID]; !exists {
		return usermgmt.ErrUserNotFound
	}

	// Check if group exists
	if _, exists := state.Groups[groupID]; !exists {
		return usermgmt.ErrGroupNotFound
	}

	// Initialize user groups if needed
	userGroups, exists := state.UserGroups[userID]
	if !exists {
		userGroups = []string{}
	}

	// Check if user is already in group
	for _, id := range userGroups {
		if id == groupID {
			// User is already in group
			return nil
		}
	}

	// Add user to group
	state.UserGroups[userID] = append(userGroups, groupID)

	// Save state
	if err := m.SaveUserState(state); err != nil {
		return err
	}

	return nil
}

// RemoveUserFromGroup removes a user from a group
func (m *Manager) RemoveUserFromGroup(ctx context.Context, userID, groupID string) error {
	state, err := m.LoadUserState()
	if err != nil {
		return err
	}

	// Check if user exists
	if _, exists := state.Users[userID]; !exists {
		return usermgmt.ErrUserNotFound
	}

	// Check if group exists
	if _, exists := state.Groups[groupID]; !exists {
		return usermgmt.ErrGroupNotFound
	}

	// Check if user is in group
	userGroups, exists := state.UserGroups[userID]
	if !exists {
		return nil
	}

	var newGroups []string
	for _, id := range userGroups {
		if id != groupID {
			newGroups = append(newGroups, id)
		}
	}

	// Update user groups
	state.UserGroups[userID] = newGroups

	// Save state
	if err := m.SaveUserState(state); err != nil {
		return err
	}

	return nil
}

// GetUserGroups gets the groups a user belongs to
func (m *Manager) GetUserGroups(ctx context.Context, userID string) ([]*usermgmt.Group, error) {
	state, err := m.LoadUserState()
	if err != nil {
		return nil, err
	}

	// Check if user exists
	if _, exists := state.Users[userID]; !exists {
		return nil, usermgmt.ErrUserNotFound
	}

	var groups []*usermgmt.Group

	// Get user groups
	userGroups, exists := state.UserGroups[userID]
	if !exists {
		return groups, nil
	}

	// Get groups
	for _, groupID := range userGroups {
		group, exists := state.Groups[groupID]
		if exists {
			groups = append(groups, copyGroup(group))
		}
	}

	return groups, nil
}

// GetGroupUsers gets the users in a group
func (m *Manager) GetGroupUsers(ctx context.Context, groupID string, pagination *usermgmt.PaginationOptions) (*usermgmt.PaginatedUsers, error) {
	state, err := m.LoadUserState()
	if err != nil {
		return nil, err
	}

	// Check if group exists
	if _, exists := state.Groups[groupID]; !exists {
		return nil, usermgmt.ErrGroupNotFound
	}

	var users []*usermgmt.User

	// Find all users in group
	for userID, groupIDs := range state.UserGroups {
		isInGroup := false
		for _, id := range groupIDs {
			if id == groupID {
				isInGroup = true
				break
			}
		}

		if isInGroup {
			user, exists := state.Users[userID]
			if exists {
				users = append(users, copyUser(user))
			}
		}
	}

	// Apply pagination
	result := &usermgmt.PaginatedUsers{
		Total:      len(users),
		Page:       1,
		PageSize:   len(users),
		TotalPages: 1,
	}

	if pagination != nil {
		// Calculate pagination
		if pagination.Page < 1 {
			pagination.Page = 1
		}

		if pagination.PageSize < 1 {
			pagination.PageSize = 10
		}

		result.Page = pagination.Page
		result.PageSize = pagination.PageSize
		result.TotalPages = (len(users) + pagination.PageSize - 1) / pagination.PageSize

		// Apply pagination
		start := (pagination.Page - 1) * pagination.PageSize
		end := start + pagination.PageSize

		if start >= len(users) {
			result.Users = []*usermgmt.User{}
		} else {
			if end > len(users) {
				end = len(users)
			}

			result.Users = users[start:end]
		}
	} else {
		result.Users = users
	}

	return result, nil
}

// IsUserInGroup checks if a user is in a group
func (m *Manager) IsUserInGroup(ctx context.Context, userID, groupID string) (bool, error) {
	state, err := m.LoadUserState()
	if err != nil {
		return false, err
	}

	// Check if user exists
	if _, exists := state.Users[userID]; !exists {
		return false, usermgmt.ErrUserNotFound
	}

	// Check if group exists
	if _, exists := state.Groups[groupID]; !exists {
		return false, usermgmt.ErrGroupNotFound
	}

	// Check if user is in group
	userGroups, exists := state.UserGroups[userID]
	if !exists {
		return false, nil
	}

	for _, id := range userGroups {
		if id == groupID {
			return true, nil
		}
	}

	return false, nil
}

// Helper functions

// copyUser creates a deep copy of a user
func copyUser(user *usermgmt.User) *usermgmt.User {
	if user == nil {
		return nil
	}

	copy := &usermgmt.User{
		ID:          user.ID,
		Username:    user.Username,
		Email:       user.Email,
		DisplayName: user.DisplayName,
		Provider:    user.Provider,
		ProviderID:  user.ProviderID,
		CreatedAt:   user.CreatedAt,
		UpdatedAt:   user.UpdatedAt,
		Enabled:     user.Enabled,
	}

	// Copy last login
	if user.LastLogin != nil {
		lastLogin := *user.LastLogin
		copy.LastLogin = &lastLogin
	}

	// Copy roles
	if user.Roles != nil {
		copy.Roles = make([]usermgmt.UserRole, len(user.Roles))
		for i, role := range user.Roles {
			copy.Roles[i] = role
		}
	}

	// Copy groups
	if user.Groups != nil {
		copy.Groups = make([]string, len(user.Groups))
		for i, group := range user.Groups {
			copy.Groups[i] = group
		}
	}

	// Copy attributes
	if user.Attributes != nil {
		copy.Attributes = make(map[string]interface{})
		for k, v := range user.Attributes {
			copy.Attributes[k] = v
		}
	}

	return copy
}

// copyGroup creates a deep copy of a group
func copyGroup(group *usermgmt.Group) *usermgmt.Group {
	if group == nil {
		return nil
	}

	copy := &usermgmt.Group{
		ID:          group.ID,
		Name:        group.Name,
		Description: group.Description,
		Provider:    group.Provider,
		ProviderID:  group.ProviderID,
		CreatedAt:   group.CreatedAt,
		UpdatedAt:   group.UpdatedAt,
	}

	// Copy attributes
	if group.Attributes != nil {
		copy.Attributes = make(map[string]interface{})
		for k, v := range group.Attributes {
			copy.Attributes[k] = v
		}
	}

	return copy
}

// generateID generates a unique ID
func generateID() string {
	return fmt.Sprintf("id-%d", time.Now().UnixNano())
}
