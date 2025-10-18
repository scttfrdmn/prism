// Package templates provides template dependency management for version resolution.
package templates

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// DependencyResolver handles template dependency resolution and version constraints
type DependencyResolver struct {
	versionResolver *VersionResolver
}

// NewDependencyResolver creates a new dependency resolver
func NewDependencyResolver(versionResolver *VersionResolver) *DependencyResolver {
	return &DependencyResolver{
		versionResolver: versionResolver,
	}
}

// ResolvedDependencies contains resolved dependency information
type ResolvedDependencies struct {
	BaseOS           string // Resolved base OS (e.g., "ubuntu")
	Version          string // Resolved version (e.g., "24.04")
	ResolvedAMI      string // Resolved AMI ID
	VersionSource    string // How version was determined (user_override, template_requirement, default)
	SatisfiesRequest bool   // Whether resolution satisfied user's request
}

// ResolveDependencies resolves template dependencies with user overrides
//
// Parameters:
//   - template: The template to resolve dependencies for
//   - userVersion: User-provided version override via --version flag
//   - region: AWS region for AMI resolution
//   - architecture: Instance architecture (x86_64, arm64)
//
// Returns resolved dependencies or an error if resolution fails
func (dr *DependencyResolver) ResolveDependencies(
	template *Template,
	userVersion, region, architecture string,
) (*ResolvedDependencies, error) {
	// Extract base OS from template
	baseOS := template.Base
	if baseOS == "" {
		return nil, fmt.Errorf("template missing base OS specification")
	}

	var resolvedVersion string
	var versionSource string

	// Priority 1: User override via --version flag
	if userVersion != "" {
		// Validate user-provided version
		if err := dr.versionResolver.ValidateVersion(baseOS, userVersion); err != nil {
			return nil, fmt.Errorf("invalid version override: %w", err)
		}
		resolvedVersion = userVersion
		versionSource = "user_override"
	} else {
		// Priority 2: Template dependency requirements
		if templateVersion, err := dr.getTemplateVersionRequirement(template); err == nil && templateVersion != "" {
			resolvedVersion = templateVersion
			versionSource = "template_requirement"
		} else {
			// Priority 3: Default version for distro
			resolvedVersion = dr.versionResolver.getDefaultVersion(baseOS)
			versionSource = "default"
		}
	}

	// Resolve AMI based on baseOS, version, region, architecture
	ami, err := dr.versionResolver.ResolveAMI(baseOS, resolvedVersion, region, architecture)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve AMI: %w", err)
	}

	return &ResolvedDependencies{
		BaseOS:           baseOS,
		Version:          resolvedVersion,
		ResolvedAMI:      ami,
		VersionSource:    versionSource,
		SatisfiesRequest: true,
	}, nil
}

// getTemplateVersionRequirement extracts version requirement from template dependencies
func (dr *DependencyResolver) getTemplateVersionRequirement(template *Template) (string, error) {
	// Check marketplace dependencies for base_os type
	if template.Marketplace != nil && len(template.Marketplace.Dependencies) > 0 {
		for _, dep := range template.Marketplace.Dependencies {
			if dep.Type == "base_os" && dep.Version != "" {
				// Parse version requirement
				return dr.parseVersionConstraint(dep.Version)
			}
		}
	}

	return "", fmt.Errorf("no version requirement found")
}

// parseVersionConstraint parses version constraints like ">=24.04", "^9", "~9.5"
//
// Supported formats:
//   - Exact: "24.04", "9", "2023"
//   - Minimum: ">=24.04", ">9"
//   - Compatible: "^24" (24.x), "^9.5" (9.5.x)
//   - Approximate: "~24.04" (~24.04.x), "~9" (~9.x)
//   - Aliases: "latest", "lts", "previous-lts"
func (dr *DependencyResolver) parseVersionConstraint(constraint string) (string, error) {
	constraint = strings.TrimSpace(constraint)

	// Exact version (no operators)
	if !strings.ContainsAny(constraint, ">=<^~*") {
		return constraint, nil
	}

	// Minimum version: >=24.04, >22
	if strings.HasPrefix(constraint, ">=") || strings.HasPrefix(constraint, ">") {
		// Extract version number
		versionStr := strings.TrimPrefix(strings.TrimPrefix(constraint, ">="), ">")
		return versionStr, nil // Return the minimum version
	}

	// Compatible version: ^24 (24.x), ^9.5 (9.5.x)
	if strings.HasPrefix(constraint, "^") {
		versionStr := strings.TrimPrefix(constraint, "^")
		return versionStr, nil // Return base version for now
	}

	// Approximate version: ~24.04 (~24.04.x), ~9 (~9.x)
	if strings.HasPrefix(constraint, "~") {
		versionStr := strings.TrimPrefix(constraint, "~")
		return versionStr, nil // Return base version for now
	}

	return "", fmt.Errorf("unsupported version constraint format: %s", constraint)
}

// ValidateVersionConstraint validates if a resolved version satisfies a constraint
//
// This is used for future validation of complex version requirements
func (dr *DependencyResolver) ValidateVersionConstraint(resolvedVersion, constraint string) (bool, error) {
	constraint = strings.TrimSpace(constraint)

	// Exact match
	if !strings.ContainsAny(constraint, ">=<^~*") {
		return resolvedVersion == constraint, nil
	}

	// Parse versions for comparison
	resolved, err := parseVersion(resolvedVersion)
	if err != nil {
		return false, fmt.Errorf("invalid resolved version: %w", err)
	}

	// Handle different constraint types
	if strings.HasPrefix(constraint, ">=") {
		required, err := parseVersion(strings.TrimPrefix(constraint, ">="))
		if err != nil {
			return false, fmt.Errorf("invalid constraint version: %w", err)
		}
		return compareVersions(resolved, required) >= 0, nil
	}

	if strings.HasPrefix(constraint, ">") {
		required, err := parseVersion(strings.TrimPrefix(constraint, ">"))
		if err != nil {
			return false, fmt.Errorf("invalid constraint version: %w", err)
		}
		return compareVersions(resolved, required) > 0, nil
	}

	if strings.HasPrefix(constraint, "^") {
		// Compatible version: major version must match
		baseVersion := strings.TrimPrefix(constraint, "^")
		base, err := parseVersion(baseVersion)
		if err != nil {
			return false, fmt.Errorf("invalid constraint version: %w", err)
		}
		// Check if major version matches
		return resolved[0] == base[0], nil
	}

	if strings.HasPrefix(constraint, "~") {
		// Approximate version: major and minor must match
		baseVersion := strings.TrimPrefix(constraint, "~")
		base, err := parseVersion(baseVersion)
		if err != nil {
			return false, fmt.Errorf("invalid constraint version: %w", err)
		}
		// Check if major and minor match
		return resolved[0] == base[0] && resolved[1] == base[1], nil
	}

	return false, fmt.Errorf("unsupported constraint type: %s", constraint)
}

// parseVersion parses a version string into numeric components
// Examples: "24.04" -> [24, 4], "9" -> [9, 0], "2023" -> [2023, 0]
func parseVersion(version string) ([]int, error) {
	// Remove any non-numeric suffixes (e.g., "24.04-lts" -> "24.04")
	re := regexp.MustCompile(`^[\d.]+`)
	numericPart := re.FindString(version)
	if numericPart == "" {
		return nil, fmt.Errorf("no numeric version found in: %s", version)
	}

	parts := strings.Split(numericPart, ".")
	result := make([]int, 0, len(parts))

	for _, part := range parts {
		num, err := strconv.Atoi(part)
		if err != nil {
			return nil, fmt.Errorf("invalid version component '%s': %w", part, err)
		}
		result = append(result, num)
	}

	// Ensure at least 2 components (major.minor)
	for len(result) < 2 {
		result = append(result, 0)
	}

	return result, nil
}

// compareVersions compares two parsed versions
// Returns: -1 if v1 < v2, 0 if v1 == v2, 1 if v1 > v2
func compareVersions(v1, v2 []int) int {
	maxLen := len(v1)
	if len(v2) > maxLen {
		maxLen = len(v2)
	}

	for i := 0; i < maxLen; i++ {
		val1 := 0
		if i < len(v1) {
			val1 = v1[i]
		}
		val2 := 0
		if i < len(v2) {
			val2 = v2[i]
		}

		if val1 < val2 {
			return -1
		}
		if val1 > val2 {
			return 1
		}
	}

	return 0
}

// GetDependencyInfo returns human-readable dependency information for a template
func (dr *DependencyResolver) GetDependencyInfo(template *Template) string {
	if template.Marketplace == nil || len(template.Marketplace.Dependencies) == 0 {
		return "No specific version requirements"
	}

	var info []string
	for _, dep := range template.Marketplace.Dependencies {
		if dep.Type == "base_os" {
			info = append(info, fmt.Sprintf("Requires: %s %s", template.Base, dep.Version))
		}
	}

	if len(info) > 0 {
		return strings.Join(info, ", ")
	}

	return "No specific version requirements"
}
