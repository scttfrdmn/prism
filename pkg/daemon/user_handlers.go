package daemon

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/scttfrdmn/prism/pkg/usermgmt"
)

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

// ==============================================================================
// User Management Features (Issue #330)
// ==============================================================================
