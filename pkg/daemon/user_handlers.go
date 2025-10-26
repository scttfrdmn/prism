package daemon

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/scttfrdmn/prism/pkg/usermgmt"
)

// handleUsers handles user management operations
func (s *Server) handleUsers(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.handleListUsers(w, r)
	case http.MethodPost:
		s.handleCreateUser(w, r)
	default:
		s.writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
	}
}

// handleUserOperations handles operations on a specific user
func (s *Server) handleUserOperations(w http.ResponseWriter, r *http.Request) {
	// Parse user ID from path
	path := r.URL.Path[len("/api/v1/users/"):]
	parts := splitPath(path)
	if len(parts) == 0 {
		s.writeError(w, http.StatusBadRequest, "Missing user ID")
		return
	}

	userID := parts[0]

	if len(parts) == 1 {
		// Operations on the user itself
		switch r.Method {
		case http.MethodGet:
			s.handleGetUser(w, r, userID)
		case http.MethodPut:
			s.handleUpdateUser(w, r, userID)
		case http.MethodDelete:
			s.handleDeleteUser(w, r, userID)
		default:
			s.writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		}
	} else if len(parts) == 2 {
		// Sub-operations
		operation := parts[1]
		switch operation {
		case "enable":
			s.handleEnableUser(w, r, userID)
		case "disable":
			s.handleDisableUser(w, r, userID)
		case "groups":
			s.handleUserGroups(w, r, userID)
		default:
			s.writeError(w, http.StatusNotFound, "Unknown operation")
		}
	} else {
		s.writeError(w, http.StatusNotFound, "Invalid path")
	}
}

// handleListUsers handles listing users
func (s *Server) handleListUsers(w http.ResponseWriter, r *http.Request) {
	// Parse query parameters for filter
	filter := &usermgmt.UserFilter{}

	if username := r.URL.Query().Get("username"); username != "" {
		filter.Username = username
	}

	if email := r.URL.Query().Get("email"); email != "" {
		filter.Email = email
	}

	if role := r.URL.Query().Get("role"); role != "" {
		filter.Role = usermgmt.UserRole(role)
	}

	if group := r.URL.Query().Get("group"); group != "" {
		filter.Group = group
	}

	if provider := r.URL.Query().Get("provider"); provider != "" {
		filter.Provider = usermgmt.Provider(provider)
	}

	if r.URL.Query().Get("enabled_only") == "true" {
		filter.EnabledOnly = true
	}

	if r.URL.Query().Get("disabled_only") == "true" {
		filter.DisabledOnly = true
	}

	// Parse pagination parameters
	pagination := &usermgmt.PaginationOptions{
		Page:     1,
		PageSize: 10,
	}

	if page := r.URL.Query().Get("page"); page != "" {
		if _, err := fmt.Sscanf(page, "%d", &pagination.Page); err != nil {
			s.writeError(w, http.StatusBadRequest, "Invalid page parameter")
			return
		}
	}

	if pageSize := r.URL.Query().Get("page_size"); pageSize != "" {
		if _, err := fmt.Sscanf(pageSize, "%d", &pagination.PageSize); err != nil {
			s.writeError(w, http.StatusBadRequest, "Invalid page_size parameter")
			return
		}
	}

	if sortBy := r.URL.Query().Get("sort_by"); sortBy != "" {
		pagination.SortBy = sortBy
	}

	if sortOrder := r.URL.Query().Get("sort_order"); sortOrder != "" {
		pagination.SortOrder = sortOrder
	}

	// Get users
	users, err := s.userManager.GetUsers(context.Background(), filter, pagination)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to list users: %v", err))
		return
	}

	_ = json.NewEncoder(w).Encode(users)
}

// handleCreateUser handles creating a new user
func (s *Server) handleCreateUser(w http.ResponseWriter, r *http.Request) {
	// Parse request body
	var user usermgmt.User
	if err := json.NewDecoder(r.Body).Decode(&user); err != nil {
		s.writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Create user
	newUser, err := s.userManager.CreateUser(context.Background(), &user)
	if err != nil {
		if err == usermgmt.ErrDuplicateUsername || err == usermgmt.ErrDuplicateEmail {
			s.writeError(w, http.StatusConflict, err.Error())
		} else {
			s.writeError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to create user: %v", err))
		}
		return
	}

	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(newUser)
}

// handleGetUser handles getting a user
func (s *Server) handleGetUser(w http.ResponseWriter, r *http.Request, id string) {
	// Get user
	user, err := s.userManager.GetUser(context.Background(), id)
	if err != nil {
		if err == usermgmt.ErrUserNotFound {
			s.writeError(w, http.StatusNotFound, "User not found")
		} else {
			s.writeError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to get user: %v", err))
		}
		return
	}

	_ = json.NewEncoder(w).Encode(user)
}

// handleUpdateUser handles updating a user
func (s *Server) handleUpdateUser(w http.ResponseWriter, r *http.Request, id string) {
	// Parse request body
	var user usermgmt.User
	if err := json.NewDecoder(r.Body).Decode(&user); err != nil {
		s.writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Ensure ID in URL matches ID in body
	if id != user.ID {
		s.writeError(w, http.StatusBadRequest, "User ID in URL does not match ID in body")
		return
	}

	// Update user
	updatedUser, err := s.userManager.UpdateUser(context.Background(), &user)
	if err != nil {
		if err == usermgmt.ErrUserNotFound {
			s.writeError(w, http.StatusNotFound, "User not found")
		} else if err == usermgmt.ErrDuplicateUsername || err == usermgmt.ErrDuplicateEmail {
			s.writeError(w, http.StatusConflict, err.Error())
		} else {
			s.writeError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to update user: %v", err))
		}
		return
	}

	_ = json.NewEncoder(w).Encode(updatedUser)
}

// handleDeleteUser handles deleting a user
func (s *Server) handleDeleteUser(w http.ResponseWriter, r *http.Request, id string) {
	// Delete user
	err := s.userManager.DeleteUser(context.Background(), id)
	if err != nil {
		if err == usermgmt.ErrUserNotFound {
			s.writeError(w, http.StatusNotFound, "User not found")
		} else {
			s.writeError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to delete user: %v", err))
		}
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// handleEnableUser handles enabling a user
func (s *Server) handleEnableUser(w http.ResponseWriter, r *http.Request, id string) {
	if r.Method != http.MethodPost {
		s.writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	// Enable user
	err := s.userManager.EnableUser(context.Background(), id)
	if err != nil {
		if err == usermgmt.ErrUserNotFound {
			s.writeError(w, http.StatusNotFound, "User not found")
		} else {
			s.writeError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to enable user: %v", err))
		}
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// handleDisableUser handles disabling a user
func (s *Server) handleDisableUser(w http.ResponseWriter, r *http.Request, id string) {
	if r.Method != http.MethodPost {
		s.writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	// Disable user
	err := s.userManager.DisableUser(context.Background(), id)
	if err != nil {
		if err == usermgmt.ErrUserNotFound {
			s.writeError(w, http.StatusNotFound, "User not found")
		} else {
			s.writeError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to disable user: %v", err))
		}
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// handleUserGroups handles user group operations
func (s *Server) handleUserGroups(w http.ResponseWriter, r *http.Request, id string) {
	switch r.Method {
	case http.MethodGet:
		// Get user groups
		groups, err := s.userManager.service.GetUserGroups(id)
		if err != nil {
			if err == usermgmt.ErrUserNotFound {
				s.writeError(w, http.StatusNotFound, "User not found")
			} else {
				s.writeError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to get user groups: %v", err))
			}
			return
		}

		_ = json.NewEncoder(w).Encode(groups)
	case http.MethodPut:
		// Update user groups
		var groupNames []string
		if err := json.NewDecoder(r.Body).Decode(&groupNames); err != nil {
			s.writeError(w, http.StatusBadRequest, "Invalid request body")
			return
		}

		// Get user
		user, err := s.userManager.GetUser(context.Background(), id)
		if err != nil {
			if err == usermgmt.ErrUserNotFound {
				s.writeError(w, http.StatusNotFound, "User not found")
			} else {
				s.writeError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to get user: %v", err))
			}
			return
		}

		// Update user groups
		user.Groups = groupNames

		// Update user
		_, err = s.userManager.UpdateUser(context.Background(), user)
		if err != nil {
			s.writeError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to update user groups: %v", err))
			return
		}

		w.WriteHeader(http.StatusNoContent)
	default:
		s.writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
	}
}

// handleGroups handles group management operations
func (s *Server) handleGroups(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.handleListGroups(w, r)
	case http.MethodPost:
		s.handleCreateGroup(w, r)
	default:
		s.writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
	}
}

// handleGroupOperations handles operations on a specific group
func (s *Server) handleGroupOperations(w http.ResponseWriter, r *http.Request) {
	// Parse group ID from path
	path := r.URL.Path[len("/api/v1/groups/"):]
	parts := splitPath(path)
	if len(parts) == 0 {
		s.writeError(w, http.StatusBadRequest, "Missing group ID")
		return
	}

	groupID := parts[0]

	if len(parts) == 1 {
		// Operations on the group itself
		switch r.Method {
		case http.MethodGet:
			s.handleGetGroup(w, r, groupID)
		case http.MethodPut:
			s.handleUpdateGroup(w, r, groupID)
		case http.MethodDelete:
			s.handleDeleteGroup(w, r, groupID)
		default:
			s.writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		}
	} else if len(parts) == 2 {
		// Sub-operations
		operation := parts[1]
		switch operation {
		case "users":
			s.handleGroupUsers(w, r, groupID)
		default:
			s.writeError(w, http.StatusNotFound, "Unknown operation")
		}
	} else {
		s.writeError(w, http.StatusNotFound, "Invalid path")
	}
}

// handleListGroups handles listing groups
func (s *Server) handleListGroups(w http.ResponseWriter, r *http.Request) {
	// Parse query parameters for filter
	filter := &usermgmt.GroupFilter{}

	if name := r.URL.Query().Get("name"); name != "" {
		filter.Name = name
	}

	if provider := r.URL.Query().Get("provider"); provider != "" {
		filter.Provider = usermgmt.Provider(provider)
	}

	// Parse pagination parameters
	pagination := &usermgmt.PaginationOptions{
		Page:     1,
		PageSize: 10,
	}

	if page := r.URL.Query().Get("page"); page != "" {
		if _, err := fmt.Sscanf(page, "%d", &pagination.Page); err != nil {
			s.writeError(w, http.StatusBadRequest, "Invalid page parameter")
			return
		}
	}

	if pageSize := r.URL.Query().Get("page_size"); pageSize != "" {
		if _, err := fmt.Sscanf(pageSize, "%d", &pagination.PageSize); err != nil {
			s.writeError(w, http.StatusBadRequest, "Invalid page_size parameter")
			return
		}
	}

	if sortBy := r.URL.Query().Get("sort_by"); sortBy != "" {
		pagination.SortBy = sortBy
	}

	if sortOrder := r.URL.Query().Get("sort_order"); sortOrder != "" {
		pagination.SortOrder = sortOrder
	}

	// Get groups
	groups, err := s.userManager.service.GetGroups()
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to list groups: %v", err))
		return
	}

	_ = json.NewEncoder(w).Encode(groups)
}

// handleCreateGroup handles creating a new group
func (s *Server) handleCreateGroup(w http.ResponseWriter, r *http.Request) {
	// Parse request body
	var group usermgmt.Group
	if err := json.NewDecoder(r.Body).Decode(&group); err != nil {
		s.writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Create group
	err := s.userManager.service.CreateGroup(&group)
	if err != nil {
		if err == usermgmt.ErrDuplicateGroup {
			s.writeError(w, http.StatusConflict, err.Error())
		} else {
			s.writeError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to create group: %v", err))
		}
		return
	}

	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(group)
}

// handleGetGroup handles getting a group
func (s *Server) handleGetGroup(w http.ResponseWriter, r *http.Request, id string) {
	// Get group
	group, err := s.userManager.service.GetGroup(id)
	if err != nil {
		if err == usermgmt.ErrGroupNotFound {
			s.writeError(w, http.StatusNotFound, "Group not found")
		} else {
			s.writeError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to get group: %v", err))
		}
		return
	}

	_ = json.NewEncoder(w).Encode(group)
}

// handleUpdateGroup handles updating a group
func (s *Server) handleUpdateGroup(w http.ResponseWriter, r *http.Request, id string) {
	// Parse request body
	var group usermgmt.Group
	if err := json.NewDecoder(r.Body).Decode(&group); err != nil {
		s.writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Ensure ID in URL matches ID in body
	if id != group.ID {
		s.writeError(w, http.StatusBadRequest, "Group ID in URL does not match ID in body")
		return
	}

	// Update group
	err := s.userManager.service.UpdateGroup(&group)
	if err != nil {
		if err == usermgmt.ErrGroupNotFound {
			s.writeError(w, http.StatusNotFound, "Group not found")
		} else if err == usermgmt.ErrDuplicateGroup {
			s.writeError(w, http.StatusConflict, err.Error())
		} else {
			s.writeError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to update group: %v", err))
		}
		return
	}

	_ = json.NewEncoder(w).Encode(group)
}

// handleDeleteGroup handles deleting a group
func (s *Server) handleDeleteGroup(w http.ResponseWriter, r *http.Request, id string) {
	// Delete group
	err := s.userManager.service.DeleteGroup(id)
	if err != nil {
		if err == usermgmt.ErrGroupNotFound {
			s.writeError(w, http.StatusNotFound, "Group not found")
		} else {
			s.writeError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to delete group: %v", err))
		}
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// GroupUsersHandler interface for different HTTP methods (Strategy Pattern - SOLID)
type GroupUsersHandler interface {
	CanHandle(method string) bool
	Handle(w http.ResponseWriter, r *http.Request, id string, server *Server) error
}

// GetGroupUsersHandler handles GET requests for group users
type GetGroupUsersHandler struct{}

func (h *GetGroupUsersHandler) CanHandle(method string) bool {
	return method == http.MethodGet
}

func (h *GetGroupUsersHandler) Handle(w http.ResponseWriter, r *http.Request, id string, server *Server) error {
	// Parse pagination parameters
	pagination := &usermgmt.PaginationOptions{
		Page:     1,
		PageSize: 10,
	}

	if err := h.parsePaginationParams(r, pagination); err != nil {
		server.writeError(w, http.StatusBadRequest, err.Error())
		return err
	}

	// Get group users
	users, err := server.userManager.service.GetGroupUsers(id)
	if err != nil {
		if err == usermgmt.ErrGroupNotFound {
			server.writeError(w, http.StatusNotFound, "Group not found")
		} else {
			server.writeError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to get group users: %v", err))
		}
		return err
	}

	return json.NewEncoder(w).Encode(users)
}

func (h *GetGroupUsersHandler) parsePaginationParams(r *http.Request, pagination *usermgmt.PaginationOptions) error {
	if page := r.URL.Query().Get("page"); page != "" {
		if _, err := fmt.Sscanf(page, "%d", &pagination.Page); err != nil {
			return fmt.Errorf("invalid page parameter")
		}
	}

	if pageSize := r.URL.Query().Get("page_size"); pageSize != "" {
		if _, err := fmt.Sscanf(pageSize, "%d", &pagination.PageSize); err != nil {
			return fmt.Errorf("invalid page_size parameter")
		}
	}

	return nil
}

// PutGroupUsersHandler handles PUT requests for updating group users
type PutGroupUsersHandler struct{}

func (h *PutGroupUsersHandler) CanHandle(method string) bool {
	return method == http.MethodPut
}

func (h *PutGroupUsersHandler) Handle(w http.ResponseWriter, r *http.Request, id string, server *Server) error {
	// Parse request body
	var userIDs []string
	if err := json.NewDecoder(r.Body).Decode(&userIDs); err != nil {
		server.writeError(w, http.StatusBadRequest, "Invalid request body")
		return err
	}

	// Get current users in group
	currentUsers, err := server.userManager.service.GetGroupUsers(id)
	if err != nil {
		if err == usermgmt.ErrGroupNotFound {
			server.writeError(w, http.StatusNotFound, "Group not found")
		} else {
			server.writeError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to get current group users: %v", err))
		}
		return err
	}

	// Update group membership
	if err := h.updateGroupMembership(currentUsers, userIDs, id, server); err != nil {
		return err
	}

	w.WriteHeader(http.StatusNoContent)
	return nil
}

func (h *PutGroupUsersHandler) updateGroupMembership(currentUsers []*usermgmt.User, userIDs []string, groupID string, server *Server) error {
	// Build maps for efficient lookup
	currentUserMap := make(map[string]bool)
	for _, user := range currentUsers {
		currentUserMap[user.ID] = true
	}

	newUserMap := make(map[string]bool)
	for _, userID := range userIDs {
		newUserMap[userID] = true
	}

	// Remove users not in the new list
	for _, user := range currentUsers {
		if !newUserMap[user.ID] {
			err := server.userManager.service.RemoveUserFromGroup(user.ID, groupID)
			if err != nil && err != usermgmt.ErrUserNotFound && err != usermgmt.ErrGroupNotFound {
				return fmt.Errorf("failed to remove user from group: %v", err)
			}
		}
	}

	// Add new users
	for _, userID := range userIDs {
		if !currentUserMap[userID] {
			err := server.userManager.service.AddUserToGroup(userID, groupID)
			if err != nil && err != usermgmt.ErrUserNotFound && err != usermgmt.ErrGroupNotFound {
				return fmt.Errorf("failed to add user to group: %v", err)
			}
		}
	}

	return nil
}

// GroupUsersMethodDispatcher manages HTTP method handlers (Strategy Pattern - SOLID)
type GroupUsersMethodDispatcher struct {
	handlers []GroupUsersHandler
}

// NewGroupUsersMethodDispatcher creates group users method dispatcher
func NewGroupUsersMethodDispatcher() *GroupUsersMethodDispatcher {
	return &GroupUsersMethodDispatcher{
		handlers: []GroupUsersHandler{
			&GetGroupUsersHandler{},
			&PutGroupUsersHandler{},
		},
	}
}

// Dispatch handles request using appropriate handler
func (d *GroupUsersMethodDispatcher) Dispatch(w http.ResponseWriter, r *http.Request, id string, server *Server) {
	for _, handler := range d.handlers {
		if handler.CanHandle(r.Method) {
			if err := handler.Handle(w, r, id, server); err != nil {
				return // Error already handled by handler
			}
			return
		}
	}

	// Method not allowed
	server.writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
}

// handleGroupUsers handles group user operations using Strategy Pattern (SOLID: Single Responsibility)
func (s *Server) handleGroupUsers(w http.ResponseWriter, r *http.Request, id string) {
	dispatcher := NewGroupUsersMethodDispatcher()
	dispatcher.Dispatch(w, r, id, s)
}

// handleAuthenticate handles user authentication
func (s *Server) handleAuthenticate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		s.writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	// Parse request
	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Authenticate user
	result, err := s.userManager.Authenticate(context.Background(), req.Username, req.Password)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, fmt.Sprintf("Authentication error: %v", err))
		return
	}

	if !result.Success {
		s.writeError(w, http.StatusUnauthorized, "Invalid credentials")
		return
	}

	// Create token response
	response := struct {
		Token     string         `json:"token"`
		User      *usermgmt.User `json:"user"`
		ExpiresAt *time.Time     `json:"expires_at"`
	}{
		Token:     result.Token,
		User:      result.User,
		ExpiresAt: result.ExpiresAt,
	}

	_ = json.NewEncoder(w).Encode(response)
}
