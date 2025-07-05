// Package ami provides CloudWorkstation's AMI creation system.
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
func (r *DependencyResolver) checkVersionConstraint(version, constraint, operator string) (bool, error) {
	// Parse versions
	v, err := NewVersionInfo(version)
	if err != nil {
		return false, err
	}

	cv, err := NewVersionInfo(constraint)
	if err != nil {
		return false, err
	}

	// Default operator is >=
	if operator == "" {
		operator = ">="
	}

	// Check constraint
	switch operator {
	case "=", "==":
		return v.Major == cv.Major && v.Minor == cv.Minor && v.Patch == cv.Patch, nil
	case ">=":
		return v.IsGreaterThan(cv) || (v.Major == cv.Major && v.Minor == cv.Minor && v.Patch == cv.Patch), nil
	case ">":
		return v.IsGreaterThan(cv), nil
	case "<=":
		return !v.IsGreaterThan(cv) || (v.Major == cv.Major && v.Minor == cv.Minor && v.Patch == cv.Patch), nil
	case "<":
		return !v.IsGreaterThan(cv) && !(v.Major == cv.Major && v.Minor == cv.Minor && v.Patch == cv.Patch), nil
	case "~>":
		// Compatible version: same major version, greater than or equal to specified minor version
		return v.Major == cv.Major && (v.Minor > cv.Minor || (v.Minor == cv.Minor && v.Patch >= cv.Patch)), nil
	default:
		return false, fmt.Errorf("unsupported version operator: %s", operator)
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
		constraints := make(map[string][]string)
		for _, dep := range deps {
			operator := dep.VersionOperator
			if operator == "" {
				operator = ">="
			}
			constraints[operator] = append(constraints[operator], dep.Version)
		}

		// Get the most restrictive version for each operator type
		var eqVersion, gtVersion, ltVersion string
		for op, versions := range constraints {
			// Sort versions
			sort.Slice(versions, func(i, j int) bool {
				v1, err1 := NewVersionInfo(versions[i])
				v2, err2 := NewVersionInfo(versions[j])
				if err1 != nil || err2 != nil {
					return versions[i] < versions[j]
				}
				return v1.IsGreaterThan(v2)
			})

			// Get most restrictive version
			if op == "=" || op == "==" {
				// For equality, all versions must be the same
				if eqVersion == "" {
					eqVersion = versions[0]
				} else if eqVersion != versions[0] {
					return nil, fmt.Errorf("conflicting exact version requirements for %s: %s vs %s", 
						template, eqVersion, versions[0])
				}
			} else if op == ">=" || op == ">" {
				// For greater than, take the highest lower bound
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
				gtVersion = highest
			} else if op == "<=" || op == "<" {
				// For less than, take the lowest upper bound
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
				ltVersion = lowest
			}
		}

		// Resolve conflicts
		if eqVersion != "" {
			// If there's an equality constraint, check if it satisfies other constraints
			if gtVersion != "" {
				gtv, _ := NewVersionInfo(gtVersion)
				eqv, _ := NewVersionInfo(eqVersion)
				if !eqv.IsGreaterThan(gtv) && !(eqv.Major == gtv.Major && eqv.Minor == gtv.Minor && eqv.Patch == gtv.Patch) {
					return nil, fmt.Errorf("conflicting version requirements for %s: equal to %s but must be >= %s", 
						template, eqVersion, gtVersion)
				}
			}
			if ltVersion != "" {
				ltv, _ := NewVersionInfo(ltVersion)
				eqv, _ := NewVersionInfo(eqVersion)
				if eqv.IsGreaterThan(ltv) {
					return nil, fmt.Errorf("conflicting version requirements for %s: equal to %s but must be <= %s", 
						template, eqVersion, ltVersion)
				}
			}
			result[template] = eqVersion
		} else if gtVersion != "" && ltVersion != "" {
			// Check if there's a version that satisfies both bounds
			gtv, _ := NewVersionInfo(gtVersion)
			ltv, _ := NewVersionInfo(ltVersion)
			if gtv.IsGreaterThan(ltv) {
				return nil, fmt.Errorf("conflicting version requirements for %s: >= %s and <= %s", 
					template, gtVersion, ltVersion)
			}
			
			// Find a version that satisfies both bounds
			compatibleVersions, err := r.FindCompatibleVersions(template, gtVersion, ">=")
			if err != nil || len(compatibleVersions) == 0 {
				// If no compatible versions found, use the lower bound
				result[template] = gtVersion
				continue
			}
			
			// Filter versions that also satisfy the upper bound
			for _, v := range compatibleVersions {
				compatible, _ := r.checkVersionConstraint(v, ltVersion, "<=")
				if compatible {
					result[template] = v
					break
				}
			}
			
			// If no version satisfies both bounds, use the lower bound
			if _, ok := result[template]; !ok {
				result[template] = gtVersion
			}
		} else if gtVersion != "" {
			// Only lower bound
			result[template] = gtVersion
		} else if ltVersion != "" {
			// Only upper bound, use the exact upper bound
			result[template] = ltVersion
		}
	}

	return result, nil
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
				
				// For now we'll use a simple mock approach
				template := &Template{
					Name: name,
					Description: "Imported from registry",
				}
				// TODO: Implement proper template parsing from string
				
				// Add template directly instead of importing
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