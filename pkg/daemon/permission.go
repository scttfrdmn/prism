package daemon

import (
	"context"
	"fmt"
	"net/http"
	
	"github.com/scttfrdmn/cloudworkstation/pkg/usermgmt"
)

// PermissionLevel represents an access level for an operation
type PermissionLevel int

const (
	// PermissionNone represents no access
	PermissionNone PermissionLevel = iota
	
	// PermissionRead represents read access
	PermissionRead
	
	// PermissionWrite represents write access (includes read)
	PermissionWrite
	
	// PermissionAdmin represents administrative access (includes read and write)
	PermissionAdmin
)

// Resource represents a resource type
type Resource string

const (
	// ResourceInstance represents an instance resource
	ResourceInstance Resource = "instance"
	
	// ResourceTemplate represents a template resource
	ResourceTemplate Resource = "template"
	
	// ResourceVolume represents a volume resource
	ResourceVolume Resource = "volume"
	
	// ResourceStorage represents a storage resource
	ResourceStorage Resource = "storage"
	
	// ResourceUser represents a user resource
	ResourceUser Resource = "user"
	
	// ResourceGroup represents a group resource
	ResourceGroup Resource = "group"
	
	// ResourceSystem represents system-level resources
	ResourceSystem Resource = "system"
)

// Operation represents an operation type
type Operation string

const (
	// OperationCreate represents a create operation
	OperationCreate Operation = "create"
	
	// OperationRead represents a read operation
	OperationRead Operation = "read"
	
	// OperationUpdate represents an update operation
	OperationUpdate Operation = "update"
	
	// OperationDelete represents a delete operation
	OperationDelete Operation = "delete"
	
	// OperationList represents a list operation
	OperationList Operation = "list"
	
	// OperationManage represents a management operation
	OperationManage Operation = "manage"
)

// Permission represents a permission check
type Permission struct {
	// Resource is the resource type
	Resource Resource
	
	// Operation is the operation type
	Operation Operation
	
	// MinimumLevel is the minimum permission level required
	MinimumLevel PermissionLevel
}

// User permission middleware
func (s *Server) permissionMiddleware(permission Permission) func(http.HandlerFunc) http.HandlerFunc {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			// Skip permission check if no user manager
			if s.userManager == nil || !s.userManager.initialized {
				// No user management, allow access
				next(w, r)
				return
			}
			
			// Get user from context (set by auth middleware)
			userID := getUserID(r.Context())
			if userID == "" {
				// No user in context, deny access
				s.writeError(w, http.StatusUnauthorized, "Authentication required")
				return
			}
			
			// Check permission
			hasPermission, err := s.checkPermission(r.Context(), userID, permission)
			if err != nil {
				s.writeError(w, http.StatusInternalServerError, fmt.Sprintf("Error checking permission: %v", err))
				return
			}
			
			if !hasPermission {
				s.writeError(w, http.StatusForbidden, "Permission denied")
				return
			}
			
			// User has permission, continue
			next(w, r)
		}
	}
}

// checkPermission checks if a user has the specified permission
func (s *Server) checkPermission(ctx context.Context, userID string, permission Permission) (bool, error) {
	// Get user
	user, err := s.userManager.GetUser(ctx, userID)
	if err != nil {
		return false, err
	}
	
	// Check if user is enabled
	if !user.Enabled {
		return false, nil
	}
	
	// Check if user has admin role (admins can do anything)
	for _, role := range user.Roles {
		if role == usermgmt.UserRoleAdmin {
			return true, nil
		}
	}
	
	// Check specific permissions based on roles
	permissionLevel := s.getRolePermissionLevel(user.Roles, permission.Resource)
	
	// Check if user has sufficient permission level
	return permissionLevel >= permission.MinimumLevel, nil
}

// getRolePermissionLevel gets the permission level for a user's roles
func (s *Server) getRolePermissionLevel(roles []usermgmt.UserRole, resource Resource) PermissionLevel {
	// Start with no permission
	level := PermissionNone
	
	// Check each role and use the highest permission level
	for _, role := range roles {
		roleLevel := PermissionNone
		
		switch role {
		case usermgmt.UserRoleAdmin:
			// Admins have admin access to everything
			roleLevel = PermissionAdmin
			
		case usermgmt.UserRolePowerUser:
			// Power users have write access to most resources, admin access to some
			switch resource {
			case ResourceInstance, ResourceVolume, ResourceStorage:
				roleLevel = PermissionAdmin
			case ResourceTemplate, ResourceUser, ResourceGroup:
				roleLevel = PermissionWrite
			case ResourceSystem:
				roleLevel = PermissionRead
			}
			
		case usermgmt.UserRoleUser:
			// Regular users have write access to instances, volumes, storage,
			// read access to templates, and no access to users, groups, or system
			switch resource {
			case ResourceInstance, ResourceVolume, ResourceStorage:
				roleLevel = PermissionWrite
			case ResourceTemplate:
				roleLevel = PermissionRead
			case ResourceUser, ResourceGroup, ResourceSystem:
				roleLevel = PermissionNone
			}
			
		case usermgmt.UserRoleReadOnly:
			// Read-only users have read access to most resources
			switch resource {
			case ResourceInstance, ResourceTemplate, ResourceVolume, ResourceStorage:
				roleLevel = PermissionRead
			case ResourceUser, ResourceGroup, ResourceSystem:
				roleLevel = PermissionNone
			}
		}
		
		// Use the highest permission level from all roles
		if roleLevel > level {
			level = roleLevel
		}
	}
	
	return level
}

// Context key for user ID
type userIDContextKey int

const (
	userIDKey userIDContextKey = iota
)

// setUserID adds the user ID to the context
func setUserID(ctx context.Context, userID string) context.Context {
	return context.WithValue(ctx, userIDKey, userID)
}

// getUserID gets the user ID from the context
func getUserID(ctx context.Context) string {
	if id, ok := ctx.Value(userIDKey).(string); ok {
		return id
	}
	return ""
}