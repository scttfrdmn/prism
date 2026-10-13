package daemon

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/scttfrdmn/prism/pkg/rbac"
)

// handleRBACRoles returns all available roles.
// GET /api/v1/rbac/roles
func (s *Server) handleRBACRoles(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	roles := s.rbacManager.ListRoles()
	w.Header().Set("Content-Type", "application/json")
	encodeJSONResponse(w, map[string]interface{}{"roles": roles})
}

// handleRBACAssign assigns a role to a user.
// POST /api/v1/rbac/assign
func (s *Server) handleRBACAssign(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		s.writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	var req rbac.AssignRoleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, "Invalid JSON request body")
		return
	}
	if req.UserID == "" || req.RoleID == "" {
		s.writeError(w, http.StatusBadRequest, "user_id and role_id are required")
		return
	}

	assignedBy := r.Header.Get("X-AWS-Profile")
	if assignedBy == "" {
		assignedBy = "admin"
	}

	if err := s.rbacManager.AssignRole(req.UserID, req.RoleID, assignedBy); err != nil {
		s.writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	// Audit the role assignment.
	s.securityManager.LogOperationalEvent("rbac.assign", req.UserID, assignedBy, true, "", map[string]interface{}{
		"role_id": req.RoleID,
	})

	w.Header().Set("Content-Type", "application/json")
	encodeJSONResponse(w, map[string]string{
		"message": "Role assigned successfully",
		"user_id": req.UserID,
		"role_id": req.RoleID,
	})
}

// handleRBACUsers returns all user role assignments.
// GET /api/v1/rbac/users
func (s *Server) handleRBACUsers(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	assignments := s.rbacManager.ListUserAssignments()
	w.Header().Set("Content-Type", "application/json")
	encodeJSONResponse(w, map[string]interface{}{"assignments": assignments})
}

// handleRBACOperations dispatches /api/v1/rbac/ sub-paths.
func (s *Server) handleRBACOperations(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/v1/rbac/")

	switch path {
	case "roles":
		s.handleRBACRoles(w, r)
	case "assign":
		s.handleRBACAssign(w, r)
	case "users":
		s.handleRBACUsers(w, r)
	default:
		s.writeError(w, http.StatusNotFound, "Unknown RBAC endpoint")
	}
}
