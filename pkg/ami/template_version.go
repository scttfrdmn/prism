// Package ami provides CloudWorkstation's AMI creation system.
package ami

import (
	"fmt"
	"strconv"
	"strings"
)

// VersionInfo represents the semantic version of a template
type VersionInfo struct {
	Major int
	Minor int
	Patch int
}

// NewVersionInfo creates a new version info from a version string
//
// The version string must be in the format "x.y.z" where x, y, and z are integers.
//
// Parameters:
//   - version: The version string in the format "x.y.z"
//
// Returns:
//   - *VersionInfo: The parsed version info
//   - error: Any parsing errors
func NewVersionInfo(version string) (*VersionInfo, error) {
	// Split version string
	parts := strings.Split(version, ".")
	if len(parts) != 3 {
		return nil, fmt.Errorf("invalid version format: %s (expected x.y.z)", version)
	}

	// Parse each part
	major, err := strconv.Atoi(parts[0])
	if err != nil {
		return nil, fmt.Errorf("invalid major version: %s", parts[0])
	}

	minor, err := strconv.Atoi(parts[1])
	if err != nil {
		return nil, fmt.Errorf("invalid minor version: %s", parts[1])
	}

	patch, err := strconv.Atoi(parts[2])
	if err != nil {
		return nil, fmt.Errorf("invalid patch version: %s", parts[2])
	}

	return &VersionInfo{
		Major: major,
		Minor: minor,
		Patch: patch,
	}, nil
}

// String returns the string representation of the version
func (v *VersionInfo) String() string {
	return fmt.Sprintf("%d.%d.%d", v.Major, v.Minor, v.Patch)
}

// IsGreaterThan returns true if this version is greater than the other version
func (v *VersionInfo) IsGreaterThan(other *VersionInfo) bool {
	if v.Major > other.Major {
		return true
	}
	if v.Major < other.Major {
		return false
	}

	// Major versions are equal, check minor
	if v.Minor > other.Minor {
		return true
	}
	if v.Minor < other.Minor {
		return false
	}

	// Minor versions are equal, check patch
	return v.Patch > other.Patch
}

// IncrementMajor increments the major version number
func (v *VersionInfo) IncrementMajor() {
	v.Major++
	v.Minor = 0
	v.Patch = 0
}

// IncrementMinor increments the minor version number
func (v *VersionInfo) IncrementMinor() {
	v.Minor++
	v.Patch = 0
}

// IncrementPatch increments the patch version number
func (v *VersionInfo) IncrementPatch() {
	v.Patch++
}

// GetTemplateVersion gets the version of a template
//
// This method extracts the version from a template's metadata.
//
// Parameters:
//   - templateName: Name of the template
//
// Returns:
//   - *VersionInfo: The template version
//   - error: Any errors
func (m *TemplateManager) GetTemplateVersion(templateName string) (*VersionInfo, error) {
	// Get template metadata
	metadata, ok := m.TemplateMetadata[templateName]
	if !ok {
		return nil, TemplateManagementError(fmt.Sprintf("template '%s' not found", templateName), nil).
			WithContext("template_name", templateName)
	}

	// Check if version is set
	if metadata.Version == "" {
		// Default to 1.0.0
		return &VersionInfo{
			Major: 1,
			Minor: 0,
			Patch: 0,
		}, nil
	}

	// Parse version
	return NewVersionInfo(metadata.Version)
}

// SetTemplateVersion sets the version of a template
//
// This method updates the version in a template's metadata.
//
// Parameters:
//   - templateName: Name of the template
//   - version: The version to set
//
// Returns:
//   - error: Any errors
func (m *TemplateManager) SetTemplateVersion(templateName string, version *VersionInfo) error {
	// Get template
	_, err := m.GetTemplate(templateName)
	if err != nil {
		return TemplateManagementError(fmt.Sprintf("template '%s' not found", templateName), err).
			WithContext("template_name", templateName)
	}

	// Update metadata
	metadata, ok := m.TemplateMetadata[templateName]
	if !ok {
		// Create new metadata if not exists
		metadata = TemplateMetadata{
			LastModified: m.clock.Now(),
		}
	}

	// Update version
	metadata.Version = version.String()

	// Save metadata
	m.TemplateMetadata[templateName] = metadata

	return nil
}

// IncrementTemplateVersion increments the version of a template
//
// This method increments the major, minor, or patch version of a template.
//
// Parameters:
//   - templateName: Name of the template
//   - component: Which component to increment ("major", "minor", or "patch")
//
// Returns:
//   - *VersionInfo: The new version
//   - error: Any errors
func (m *TemplateManager) IncrementTemplateVersion(templateName, component string) (*VersionInfo, error) {
	// Get current version
	version, err := m.GetTemplateVersion(templateName)
	if err != nil {
		return nil, err
	}

	// Increment the specified component
	switch component {
	case "major":
		version.IncrementMajor()
	case "minor":
		version.IncrementMinor()
	case "patch":
		version.IncrementPatch()
	default:
		return nil, ValidationError(fmt.Sprintf("invalid version component: %s", component), nil).
			WithContext("component", component).
			WithContext("valid_components", "major, minor, patch")
	}

	// Update template version
	err = m.SetTemplateVersion(templateName, version)
	if err != nil {
		return nil, err
	}

	return version, nil
}

// VersionTemplate creates a new versioned copy of a template
//
// This method creates a new template with a new version number.
//
// Parameters:
//   - templateName: Name of the template to version
//   - newVersion: New version info (or nil to auto-increment)
//   - component: Component to increment if newVersion is nil ("major", "minor", or "patch")
//
// Returns:
//   - *Template: The new versioned template
//   - error: Any errors
func (m *TemplateManager) VersionTemplate(templateName string, newVersion *VersionInfo, component string) (*Template, error) {
	// Get template
	template, err := m.GetTemplate(templateName)
	if err != nil {
		return nil, err
	}

	// Determine new version
	var version *VersionInfo
	if newVersion != nil {
		version = newVersion
	} else {
		// Auto-increment
		version, err = m.IncrementTemplateVersion(templateName, component)
		if err != nil {
			return nil, err
		}
	}

	// Create a deep copy of the template
	newTemplate := *template

	// Update metadata
	if m.TemplateMetadata != nil {
		metadata, ok := m.TemplateMetadata[templateName]
		if ok {
			metadata.Version = version.String()
			metadata.LastModified = m.clock.Now()
			m.TemplateMetadata[templateName] = metadata
		}
	}

	return &newTemplate, nil
}

// CreateTemplateVersion creates a new version of a template with changes
//
// This method creates a new version of a template with changes applied via the builder pattern.
//
// Parameters:
//   - templateName: Name of the template to version
//   - component: Component to increment ("major", "minor", or "patch")
//
// Returns:
//   - *TemplateBuilder: Builder for the new version
//   - error: Any errors
func (m *TemplateManager) CreateTemplateVersion(templateName, component string) (*TemplateBuilder, error) {
	// Get original template
	template, err := m.GetTemplate(templateName)
	if err != nil {
		return nil, err
	}

	// Get current version
	version, err := m.GetTemplateVersion(templateName)
	if err != nil {
		return nil, err
	}

	// Increment version
	switch component {
	case "major":
		version.IncrementMajor()
	case "minor":
		version.IncrementMinor()
	case "patch":
		version.IncrementPatch()
	default:
		return nil, ValidationError(fmt.Sprintf("invalid version component: %s", component), nil).
			WithContext("component", component).
			WithContext("valid_components", "major, minor, patch")
	}

	// Create a deep copy of the template
	newTemplate := *template

	// Create builder
	builder := &TemplateBuilder{
		manager:     m,
		template:    &newTemplate,
		hasModified: true, // Treat as modified to force overwrite
	}
	
	// Set version in metadata
	metadata := TemplateMetadata{
		LastModified: m.clock.Now(),
		Version:      version.String(),
	}
	m.TemplateMetadata[newTemplate.Name] = metadata

	return builder, nil
}

// Note: Equals method is defined in template_dependency.go