// Package ami provides Prism's AMI creation system.
package ami

import (
	"context"
	"fmt"
	"sort"
	"strings"
)

// DependencyResolver handles automatic resolution of template dependencies
type DependencyResolver struct {
	Manager *TemplateManager
}

// ResolvedDependency contains information about a resolved dependency
type ResolvedDependency struct {
	Name         string
	OriginalName string
	Version      string
	IsOptional   bool
	Status       string // "satisfied", "missing", "version-mismatch"
}

// NewDependencyResolver creates a new dependency resolver
func NewDependencyResolver(manager *TemplateManager) *DependencyResolver {
	return &DependencyResolver{
		Manager: manager,
	}
}

// ResolveDependencies resolves all dependencies for a template
//
// This function analyzes all dependencies and attempts to find
// compatible versions to satisfy the requirements.
//
// Parameters:
//   - templateName: The name of the template to resolve dependencies for
//
// Returns:
//   - map[string]*ResolvedDependency: Map of resolved dependencies
//   - []string: Ordered build list
//   - error: Any resolution errors
func (r *DependencyResolver) ResolveDependencies(templateName string) (map[string]*ResolvedDependency, []string, error) {
	// Get template
	template, err := r.Manager.GetTemplate(templateName)
	if err != nil {
		return nil, nil, err
	}

	// Get dependency graph to determine build order
	graph, err := r.Manager.GetDependencyGraph(templateName)
	if err != nil {
		return nil, nil, err
	}

	// Initialize result
	resolved := make(map[string]*ResolvedDependency)

	// Process each dependency
	for _, dep := range template.Dependencies {
		resolvedDep, err := r.resolveDependency(templateName, dep)
		if err != nil && !dep.Optional {
			return nil, nil, err
		}

		if resolvedDep != nil {
			resolved[dep.Name] = resolvedDep
		}
	}

	return resolved, graph, nil
}

// resolveDependency resolves a single dependency
func (r *DependencyResolver) resolveDependency(templateName string, dep TemplateDependency) (*ResolvedDependency, error) {
	// Check if dependency exists
	_, err := r.Manager.GetTemplate(dep.Name)
	if err != nil {
		return &ResolvedDependency{
			Name:         dep.Name,
			OriginalName: dep.Name,
			Version:      "",
			IsOptional:   dep.Optional,
			Status:       "missing",
		}, fmt.Errorf("dependent template '%s' not found", dep.Name)
	}

	// Get dependency version
	metadata, ok := r.Manager.TemplateMetadata[dep.Name]
	if !ok || metadata.Version == "" {
		return &ResolvedDependency{
			Name:         dep.Name,
			OriginalName: dep.Name,
			Version:      "",
			IsOptional:   dep.Optional,
			Status:       "missing-version",
		}, fmt.Errorf("dependent template '%s' has no version information", dep.Name)
	}

	// If no version constraint, we're done
	if dep.Version == "" {
		return &ResolvedDependency{
			Name:         dep.Name,
			OriginalName: dep.Name,
			Version:      metadata.Version,
			IsOptional:   dep.Optional,
			Status:       "satisfied",
		}, nil
	}

	// Check if version satisfies constraint
	satisfied, err := r.checkVersionConstraint(metadata.Version, dep.Version, dep.VersionOperator)
	if err != nil {
		return &ResolvedDependency{
			Name:         dep.Name,
			OriginalName: dep.Name,
			Version:      metadata.Version,
			IsOptional:   dep.Optional,
			Status:       "invalid-version",
		}, err
	}

	if satisfied {
		return &ResolvedDependency{
			Name:         dep.Name,
			OriginalName: dep.Name,
			Version:      metadata.Version,
			IsOptional:   dep.Optional,
			Status:       "satisfied",
		}, nil
	}

	// Version doesn't satisfy constraint
	return &ResolvedDependency{
			Name:         dep.Name,
			OriginalName: dep.Name,
			Version:      metadata.Version,
			IsOptional:   dep.Optional,
			Status:       "version-mismatch",
		}, fmt.Errorf("dependent template '%s' version %s doesn't satisfy constraint %s %s",
			dep.Name, metadata.Version, dep.VersionOperator, dep.Version)
}

// checkVersionConstraint checks if a version satisfies a constraint
// checkVersionConstraint validates version constraints using Strategy Pattern (SOLID: Single Responsibility)
func (r *DependencyResolver) checkVersionConstraint(version, constraint, operator string) (bool, error) {
	// Parse versions
	depVersion, reqVersion, err := r.parseVersionConstraintInputs(version, constraint)
	if err != nil {
		return false, err
	}

	// Use Strategy Pattern for version checking
	checker := r.getVersionChecker(operator)
	if checker == nil {
		return false, fmt.Errorf("unsupported version operator: %s", operator)
	}

	return checker.Check(depVersion, reqVersion), nil
}

// parseVersionConstraintInputs parses version strings for constraint checking (Single Responsibility)
func (r *DependencyResolver) parseVersionConstraintInputs(version, constraint string) (*VersionInfo, *VersionInfo, error) {
	v, err := NewVersionInfo(version)
	if err != nil {
		return nil, nil, err
	}

	cv, err := NewVersionInfo(constraint)
	if err != nil {
		return nil, nil, err
	}

	return v, cv, nil
}

// getVersionChecker returns the appropriate version checker using Strategy Pattern (SOLID: Open/Closed)
func (r *DependencyResolver) getVersionChecker(operator string) VersionChecker {
	// Default operator is >=
	if operator == "" {
		operator = ">="
	}

	switch operator {
	case "=", "==":
		return &ExactVersionChecker{}
	case ">=":
		return &GreaterThanOrEqualChecker{}
	case ">":
		return &GreaterThanChecker{}
	case "<=":
		return &LessThanOrEqualChecker{}
	case "<":
		return &LessThanChecker{}
	case "~>":
		return &CompatibleVersionChecker{}
	default:
		return nil
	}
}

// FindCompatibleVersions finds compatible versions for a dependency
//
// This function searches the registry for versions of a template that satisfy
// the version constraint. This is useful for suggesting alternatives when
// the current version doesn't satisfy the constraint.
//
// Parameters:
//   - templateName: The name of the template
//   - constraint: The version constraint
//   - operator: The operator for the constraint
//
// Returns:
//   - []string: List of compatible version strings
//   - error: Any errors
func (r *DependencyResolver) FindCompatibleVersions(templateName, constraint, operator string) ([]string, error) {
	// Check if registry is configured
	if r.Manager.Registry == nil {
		return nil, fmt.Errorf("registry not configured")
	}

	// List all versions
	ctx := context.Background()
	allVersions, err := r.Manager.Registry.ListSharedTemplateVersions(ctx, templateName)
	if err != nil {
		return nil, err
	}

	// Filter compatible versions
	var compatibleVersions []string
	for _, version := range allVersions {
		compatible, err := r.checkVersionConstraint(version, constraint, operator)
		if err != nil {
			// Skip invalid versions
			continue
		}

		if compatible {
			compatibleVersions = append(compatibleVersions, version)
		}
	}

	// Sort versions in descending order
	sort.Slice(compatibleVersions, func(i, j int) bool {
		v1, err1 := NewVersionInfo(compatibleVersions[i])
		v2, err2 := NewVersionInfo(compatibleVersions[j])
		if err1 != nil || err2 != nil {
			return compatibleVersions[i] > compatibleVersions[j]
		}
		return v1.IsGreaterThan(v2)
	})

	return compatibleVersions, nil
}

// ResolveDependencyConflicts attempts to resolve conflicts between dependencies
//
// This function analyzes conflicts between different version requirements for the
// same dependency and tries to find a version that satisfies all constraints.
//
// Parameters:
//   - conflicts: Map of template name to list of conflicting dependencies
//
// Returns:
//   - map[string]string: Map of template name to resolved version
//   - error: Any resolution errors
func (r *DependencyResolver) ResolveDependencyConflicts(conflicts map[string][]TemplateDependency) (map[string]string, error) {
	result := make(map[string]string)

	for template, deps := range conflicts {
		// Group constraints by operator
		constraints := r.groupConstraintsByOperator(deps)

		// Get the most restrictive version for each operator type
		resolvedVersions, err := r.findMostRestrictiveVersions(constraints, template)
		if err != nil {
			return nil, err
		}

		// Validate version compatibility and find final version
		finalVersion, err := r.validateAndResolveVersions(template, resolvedVersions)
		if err != nil {
			return nil, err
		}

		if finalVersion != "" {
			result[template] = finalVersion
		}
	}

	return result, nil
}

// groupConstraintsByOperator groups version constraints by their operators
func (r *DependencyResolver) groupConstraintsByOperator(deps []TemplateDependency) map[string][]string {
	constraints := make(map[string][]string)
	for _, dep := range deps {
		operator := dep.VersionOperator
		if operator == "" {
			operator = ">="
		}
		constraints[operator] = append(constraints[operator], dep.Version)
	}
	return constraints
}

// RestrictiveVersions holds the most restrictive versions for each operator type
type RestrictiveVersions struct {
	EqVersion string
	GtVersion string
	LtVersion string
}

// findMostRestrictiveVersions finds the most restrictive version for each operator type
func (r *DependencyResolver) findMostRestrictiveVersions(constraints map[string][]string, template string) (*RestrictiveVersions, error) {
	result := &RestrictiveVersions{}

	for op, versions := range constraints {
		// Sort versions in descending order
		sortedVersions := r.sortVersionsDescending(versions)

		switch op {
		case "=", "==":
			eqVersion, err := r.processEqualityConstraints(sortedVersions, template, result.EqVersion)
			if err != nil {
				return nil, err
			}
			result.EqVersion = eqVersion

		case ">=", ">":
			result.GtVersion = r.findHighestVersion(sortedVersions)

		case "<=", "<":
			result.LtVersion = r.findLowestVersion(sortedVersions)
		}
	}

	return result, nil
}

// sortVersionsDescending sorts version strings in descending order
func (r *DependencyResolver) sortVersionsDescending(versions []string) []string {
	sorted := make([]string, len(versions))
	copy(sorted, versions)

	sort.Slice(sorted, func(i, j int) bool {
		v1, err1 := NewVersionInfo(sorted[i])
		v2, err2 := NewVersionInfo(sorted[j])
		if err1 != nil || err2 != nil {
			return sorted[i] < sorted[j]
		}
		return v1.IsGreaterThan(v2)
	})

	return sorted
}

// processEqualityConstraints processes equality constraints and ensures all versions are the same
func (r *DependencyResolver) processEqualityConstraints(versions []string, template, currentEqVersion string) (string, error) {
	if len(versions) == 0 {
		return currentEqVersion, nil
	}

	newEqVersion := versions[0]
	if currentEqVersion == "" {
		return newEqVersion, nil
	}

	if currentEqVersion != newEqVersion {
		return "", fmt.Errorf("conflicting exact version requirements for %s: %s vs %s",
			template, currentEqVersion, newEqVersion)
	}

	return currentEqVersion, nil
}

// findHighestVersion finds the highest version from a list of versions
func (r *DependencyResolver) findHighestVersion(versions []string) string {
	if len(versions) == 0 {
		return ""
	}

	highest := versions[0]
	for _, v := range versions[1:] {
		v1, err1 := NewVersionInfo(highest)
		v2, err2 := NewVersionInfo(v)
		if err1 != nil || err2 != nil {
			continue
		}
		if v2.IsGreaterThan(v1) {
			highest = v
		}
	}
	return highest
}

// findLowestVersion finds the lowest version from a list of versions
func (r *DependencyResolver) findLowestVersion(versions []string) string {
	if len(versions) == 0 {
		return ""
	}

	lowest := versions[0]
	for _, v := range versions[1:] {
		v1, err1 := NewVersionInfo(lowest)
		v2, err2 := NewVersionInfo(v)
		if err1 != nil || err2 != nil {
			continue
		}
		if !v2.IsGreaterThan(v1) {
			lowest = v
		}
	}
	return lowest
}

// validateAndResolveVersions validates version compatibility and finds the final resolved version
func (r *DependencyResolver) validateAndResolveVersions(template string, versions *RestrictiveVersions) (string, error) {
	// If there's an equality constraint, validate it against other constraints
	if versions.EqVersion != "" {
		return r.validateEqualityConstraint(template, versions)
	}

	// Handle range constraints (both lower and upper bounds)
	if versions.GtVersion != "" && versions.LtVersion != "" {
		return r.resolveRangeConstraints(template, versions.GtVersion, versions.LtVersion)
	}

	// Handle single-bound constraints
	if versions.GtVersion != "" {
		return versions.GtVersion, nil
	}
	if versions.LtVersion != "" {
		return versions.LtVersion, nil
	}

	return "", nil
}

// validateEqualityConstraint validates that an equality constraint satisfies other constraints
func (r *DependencyResolver) validateEqualityConstraint(template string, versions *RestrictiveVersions) (string, error) {
	eqVersion := versions.EqVersion
	eqv, _ := NewVersionInfo(eqVersion)

	// Check against lower bound
	if versions.GtVersion != "" {
		gtv, _ := NewVersionInfo(versions.GtVersion)
		if gtv.IsGreaterThan(eqv) {
			return "", fmt.Errorf("conflicting version requirements for %s: equal to %s but must be >= %s",
				template, eqVersion, versions.GtVersion)
		}
	}

	// Check against upper bound
	if versions.LtVersion != "" {
		ltv, _ := NewVersionInfo(versions.LtVersion)
		if eqv.IsGreaterThan(ltv) {
			return "", fmt.Errorf("conflicting version requirements for %s: equal to %s but must be <= %s",
				template, eqVersion, versions.LtVersion)
		}
	}

	return eqVersion, nil
}

// resolveRangeConstraints resolves version constraints with both lower and upper bounds
func (r *DependencyResolver) resolveRangeConstraints(template, gtVersion, ltVersion string) (string, error) {
	// Check if there's a valid range
	gtv, _ := NewVersionInfo(gtVersion)
	ltv, _ := NewVersionInfo(ltVersion)
	if gtv.IsGreaterThan(ltv) {
		return "", fmt.Errorf("conflicting version requirements for %s: >= %s and <= %s",
			template, gtVersion, ltVersion)
	}

	// Try to find a version that satisfies both bounds
	compatibleVersions, err := r.FindCompatibleVersions(template, gtVersion, ">=")
	if err != nil || len(compatibleVersions) == 0 {
		// If no compatible versions found, use the lower bound
		return gtVersion, nil
	}

	// Filter versions that also satisfy the upper bound
	for _, v := range compatibleVersions {
		compatible, _ := r.checkVersionConstraint(v, ltVersion, "<=")
		if compatible {
			return v, nil
		}
	}

	// If no version satisfies both bounds, use the lower bound
	return gtVersion, nil
}

// ResolveAndFetchDependencies resolves dependencies and fetches missing templates
//
// This function analyzes dependencies, resolves conflicts, and attempts to
// fetch missing templates from the registry.
//
// Parameters:
//   - templateName: The name of the template to resolve dependencies for
//   - fetchMissing: Whether to fetch missing templates from the registry
//
// Returns:
//   - map[string]*ResolvedDependency: Map of resolved dependencies
//   - []string: List of fetched templates
//   - error: Any resolution errors
func (r *DependencyResolver) ResolveAndFetchDependencies(templateName string, fetchMissing bool) (map[string]*ResolvedDependency, []string, error) {
	// Resolve dependencies
	resolved, _, err := r.ResolveDependencies(templateName)
	if err != nil && !strings.Contains(err.Error(), "missing") {
		return resolved, nil, err
	}

	if !fetchMissing {
		return resolved, nil, err
	}

	// Check registry availability
	if r.Manager.Registry == nil {
		return resolved, nil, fmt.Errorf("registry not configured, cannot fetch missing dependencies")
	}

	// List of templates that were fetched
	fetched := []string{}

	// Try to fetch missing dependencies
	for name, dep := range resolved {
		if dep.Status == "missing" {
			// Try to fetch from registry
			ctx := context.Background()
			entries, err := r.Manager.Registry.ListSharedTemplates(ctx)
			if err != nil {
				continue
			}

			if entry, ok := entries[name]; ok {
				// Found in registry, fetch it
				templateData := entry.TemplateData
				if templateData == "" {
					continue
				}

				// Parse template from string data using Parser
				template, err := r.Manager.Parser.ParseTemplate(templateData)
				if err != nil {
					// Log error but continue with other templates
					continue
				}

				// Ensure the template has the correct name
				template.Name = name

				// Add template to manager
				r.Manager.Templates[name] = template

				// Add metadata
				r.Manager.TemplateMetadata[name] = TemplateMetadata{
					Version:      entry.Version,
					LastModified: entry.PublishedAt,
					SourceURL:    "registry://" + name,
				}
				// Update resolved dependency status
				dep.Status = "satisfied"
				dep.Version = entry.Version
				fetched = append(fetched, name)
			}
		}
	}

	return resolved, fetched, nil
}

// VersionChecker defines the interface for version constraint checking using Strategy Pattern (SOLID: Open/Closed)
type VersionChecker interface {
	Check(version *VersionInfo, constraint *VersionInfo) bool
}

// ExactVersionChecker validates exact version matches (Strategy Pattern)
type ExactVersionChecker struct{}

func (c *ExactVersionChecker) Check(version *VersionInfo, constraint *VersionInfo) bool {
	return version.Major == constraint.Major && version.Minor == constraint.Minor && version.Patch == constraint.Patch
}

// GreaterThanOrEqualChecker validates >= version constraints (Strategy Pattern)
type GreaterThanOrEqualChecker struct{}

func (c *GreaterThanOrEqualChecker) Check(version *VersionInfo, constraint *VersionInfo) bool {
	return version.IsGreaterThan(constraint) || version.Equals(constraint)
}

// GreaterThanChecker validates > version constraints (Strategy Pattern)
type GreaterThanChecker struct{}

func (c *GreaterThanChecker) Check(version *VersionInfo, constraint *VersionInfo) bool {
	return version.IsGreaterThan(constraint)
}

// LessThanOrEqualChecker validates <= version constraints (Strategy Pattern)
type LessThanOrEqualChecker struct{}

func (c *LessThanOrEqualChecker) Check(version *VersionInfo, constraint *VersionInfo) bool {
	return !version.IsGreaterThan(constraint) || version.Equals(constraint)
}

// LessThanChecker validates < version constraints (Strategy Pattern)
type LessThanChecker struct{}

func (c *LessThanChecker) Check(version *VersionInfo, constraint *VersionInfo) bool {
	return constraint.IsGreaterThan(version)
}

// CompatibleVersionChecker validates ~> version constraints (Strategy Pattern)
type CompatibleVersionChecker struct{}

func (c *CompatibleVersionChecker) Check(version *VersionInfo, constraint *VersionInfo) bool {
	// Compatible version: same major version, greater than or equal to specified minor version
	return version.Major == constraint.Major && (version.Minor > constraint.Minor || (version.Minor == constraint.Minor && version.Patch >= constraint.Patch))
}
