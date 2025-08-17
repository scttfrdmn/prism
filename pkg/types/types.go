// Package types provides CloudWorkstation's core type definitions.
//
// This package is organized into logical modules:
//   - runtime.go: Instance and template runtime definitions
//   - storage.go: EFS and EBS volume types
//   - config.go: Configuration and state management
//   - requests.go: API request/response types
//   - api_version.go: API versioning types
//   - errors.go: Error handling types
//   - idle.go: Idle detection types
//   - instance.go: Instance-specific types
//   - repository.go: Repository management types
//
// For backward compatibility, the main types are also available
// through this file via type aliases.
package types

// Backward compatibility aliases
// Template is aliased to RuntimeTemplate to distinguish from AMI build templates
type Template = RuntimeTemplate

// SimpleAPIError represents a simple API error response (legacy)
type SimpleAPIError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Details string `json:"details,omitempty"`
}

func (e SimpleAPIError) Error() string {
	return e.Message
}
