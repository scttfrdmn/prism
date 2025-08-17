package daemon

import (
	"context"
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

// Context key for user ID
type userIDContextKey int

const (
	userIDKey userIDContextKey = iota
)

// setUserID adds the user ID to the context
func setUserID(ctx context.Context, userID string) context.Context {
	return context.WithValue(ctx, userIDKey, userID)
}
