// Package ami provides Prism's AMI creation system.
package ami

import (
	"fmt"
)

// TemplateDependency represents a dependency on another template
type TemplateDependency struct {
	Name            string `json:"name" yaml:"name"`
	Version         string `json:"version,omitempty" yaml:"version,omitempty"`
	VersionOperator string `json:"version_operator,omitempty" yaml:"version_operator,omitempty"`
	Optional        bool   `json:"optional,omitempty" yaml:"optional,omitempty"`
}

// ValidateTemplateDependencies validates all dependencies for a template
//
// This function ensures all dependencies exist and version constraints are satisfied.
//
// Parameters:
//   - templateName: The name of the template being validated
//   - dependencies: List of dependencies to validate
//
// Returns:
//   - error: Any validation errors
func (m *TemplateManager) ValidateTemplateDependencies(templateName string, dependencies []TemplateDependency) error {
	if len(dependencies) == 0 {
		return nil
	}

	var errors []error
	for _, dep := range dependencies {
		err := m.validateDependency(templateName, dep)
		if err != nil && !dep.Optional {
			errors = append(errors, err)
		}
	}

	if len(errors) > 0 {
		errorMsg := fmt.Sprintf("template '%s' has dependency validation errors", templateName)
		return DependencyError(errorMsg, errors[0]).
			WithContext("template_name", templateName).
			WithContext("dependency_count", fmt.Sprintf("%d", len(dependencies))).
			WithContext("failed_dependencies", fmt.Sprintf("%d", len(errors)))
	}

	return nil
}

// validateDependency validates a single dependency
// validateDependency validates a single template dependency using Strategy Pattern (SOLID: Single Responsibility)
func (m *TemplateManager) validateDependency(templateName string, dep TemplateDependency) error {
	// Check if dependency exists
	if err := m.validateDependencyExists(dep); err != nil {
		return err
	}

	// Validate version constraints if specified
	if dep.Version != "" {
		return m.validateDependencyVersion(dep)
	}

	return nil
}

// validateDependencyExists checks if the dependency template exists (Single Responsibility)
func (m *TemplateManager) validateDependencyExists(dep TemplateDependency) error {
	_, err := m.GetTemplate(dep.Name)
	if err != nil {
		return DependencyError(
			fmt.Sprintf("dependent template '%s' not found", dep.Name),
			err,
		).WithContext("dependency_name", dep.Name)
	}
	return nil
}

// validateDependencyVersion validates dependency version constraints using Strategy Pattern (SOLID: Open/Closed)
func (m *TemplateManager) validateDependencyVersion(dep TemplateDependency) error {
	// Get dependency version information
	depVersion, reqVersion, err := m.parseDependencyVersions(dep)
	if err != nil {
		return err
	}

	// Use Strategy Pattern for version validation
	operator := dep.VersionOperator
	if operator == "" {
		operator = ">=" // Default is greater than or equal
	}

	validator := m.getVersionValidator(operator)
	if validator == nil {
		return DependencyError(
			fmt.Sprintf("invalid version operator: %s", operator),
			nil,
		).WithContext("dependency_name", dep.Name).
			WithContext("operator", operator).
			WithContext("valid_operators", "=, ==, >, >=, <, <=")
	}

	return validator.Validate(dep, depVersion, reqVersion)
}

// parseDependencyVersions parses and validates version information (Single Responsibility)
func (m *TemplateManager) parseDependencyVersions(dep TemplateDependency) (*VersionInfo, *VersionInfo, error) {
	// Get dependency version
	metadata, ok := m.TemplateMetadata[dep.Name]
	if !ok || metadata.Version == "" {
		if dep.Optional {
			return nil, nil, nil
		}
		return nil, nil, DependencyError(
			fmt.Sprintf("dependent template '%s' has no version information", dep.Name),
			nil,
		).WithContext("dependency_name", dep.Name)
	}

	// Parse dependency version
	depVersion, err := NewVersionInfo(metadata.Version)
	if err != nil {
		return nil, nil, DependencyError(
			fmt.Sprintf("dependent template '%s' has invalid version: %s", dep.Name, metadata.Version),
			err,
		).WithContext("dependency_name", dep.Name).
			WithContext("dependency_version", metadata.Version)
	}

	// Parse required version
	reqVersion, err := NewVersionInfo(dep.Version)
	if err != nil {
		return nil, nil, DependencyError(
			fmt.Sprintf("invalid version requirement: %s", dep.Version),
			err,
		).WithContext("dependency_name", dep.Name).
			WithContext("required_version", dep.Version)
	}

	return depVersion, reqVersion, nil
}

// getVersionValidator returns the appropriate version validator using Strategy Pattern (SOLID: Open/Closed)
func (m *TemplateManager) getVersionValidator(operator string) VersionValidator {
	switch operator {
	case "=", "==":
		return &ExactVersionValidator{}
	case ">=":
		return &GreaterThanOrEqualValidator{}
	case ">":
		return &GreaterThanValidator{}
	case "<":
		return &LessThanValidator{}
	case "<=":
		return &LessThanOrEqualValidator{}
	default:
		return nil
	}
}

// AddDependency adds a dependency to a template
//
// This function adds a new dependency to a template.
//
// Parameters:
//   - templateName: The name of the template to modify
//   - dependency: The dependency to add
//
// Returns:
//   - error: Any errors
func (m *TemplateManager) AddDependency(templateName string, dependency TemplateDependency) error {
	template, err := m.GetTemplate(templateName)
	if err != nil {
		return err
	}

	// Check if dependency already exists
	for _, dep := range template.Dependencies {
		if dep.Name == dependency.Name {
			return DependencyError(
				fmt.Sprintf("template '%s' already has dependency '%s'", templateName, dependency.Name),
				nil,
			).WithContext("template_name", templateName).
				WithContext("dependency_name", dependency.Name)
		}
	}

	// Validate dependency
	if err := m.validateDependency(templateName, dependency); err != nil {
		if dependency.Optional {
			// Log warning but continue if dependency is optional
		} else {
			return err
		}
	}

	// Add dependency
	template.Dependencies = append(template.Dependencies, dependency)

	return nil
}

// RemoveDependency removes a dependency from a template
//
// Parameters:
//   - templateName: The name of the template to modify
//   - dependencyName: The name of the dependency to remove
//
// Returns:
//   - bool: True if dependency was found and removed
//   - error: Any errors
func (m *TemplateManager) RemoveDependency(templateName string, dependencyName string) (bool, error) {
	template, err := m.GetTemplate(templateName)
	if err != nil {
		return false, err
	}

	// Find dependency
	found := false
	newDependencies := []TemplateDependency{}
	for _, dep := range template.Dependencies {
		if dep.Name == dependencyName {
			found = true
		} else {
			newDependencies = append(newDependencies, dep)
		}
	}

	if !found {
		return false, nil
	}

	// Update dependencies
	template.Dependencies = newDependencies
	return true, nil
}

// GetDependencyGraph builds a dependency graph for a template
//
// This function returns a sorted list of templates in the order they should be built.
//
// Parameters:
//   - templateName: The name of the template to build graph for
//
// Returns:
//   - []string: Sorted list of template names in build order
//   - error: Any errors
func (m *TemplateManager) GetDependencyGraph(templateName string) ([]string, error) {
	visited := make(map[string]bool)
	graph := make(map[string][]string)
	sorted := []string{}

	// Build dependency graph
	if err := m.buildDependencyGraph(templateName, graph, visited); err != nil {
		return nil, err
	}

	// Check for cycles
	visitedInCurrentPath := make(map[string]bool)
	if m.hasCycle(templateName, graph, visited, visitedInCurrentPath) {
		return nil, DependencyError("circular dependency detected", nil).
			WithContext("template_name", templateName)
	}

	// Topologically sort the graph
	visited = make(map[string]bool)
	if err := m.topologicalSort(templateName, graph, visited, &sorted); err != nil {
		return nil, err
	}

	// Reverse the result to get build order
	for i, j := 0, len(sorted)-1; i < j; i, j = i+1, j-1 {
		sorted[i], sorted[j] = sorted[j], sorted[i]
	}

	return sorted, nil
}

// buildDependencyGraph builds a graph representation of template dependencies
func (m *TemplateManager) buildDependencyGraph(templateName string, graph map[string][]string, visited map[string]bool) error {
	if visited[templateName] {
		return nil
	}

	visited[templateName] = true
	graph[templateName] = []string{}

	template, err := m.GetTemplate(templateName)
	if err != nil {
		return err
	}

	for _, dep := range template.Dependencies {
		graph[templateName] = append(graph[templateName], dep.Name)

		if !visited[dep.Name] {
			if err := m.buildDependencyGraph(dep.Name, graph, visited); err != nil {
				return err
			}
		}
	}

	return nil
}

// hasCycle detects cycles in the dependency graph
func (m *TemplateManager) hasCycle(templateName string, graph map[string][]string, visited, visitedInCurrentPath map[string]bool) bool {
	visited[templateName] = true
	visitedInCurrentPath[templateName] = true

	for _, dep := range graph[templateName] {
		if !visited[dep] {
			if m.hasCycle(dep, graph, visited, visitedInCurrentPath) {
				return true
			}
		} else if visitedInCurrentPath[dep] {
			return true
		}
	}

	visitedInCurrentPath[templateName] = false
	return false
}

// topologicalSort performs a topological sort of the dependency graph
func (m *TemplateManager) topologicalSort(templateName string, graph map[string][]string, visited map[string]bool, sorted *[]string) error {
	if visited[templateName] {
		return nil
	}

	visited[templateName] = true

	for _, dep := range graph[templateName] {
		if !visited[dep] {
			if err := m.topologicalSort(dep, graph, visited, sorted); err != nil {
				return err
			}
		}
	}

	*sorted = append(*sorted, templateName)
	return nil
}

// Equals checks if two versions are equal
func (v *VersionInfo) Equals(other *VersionInfo) bool {
	return v.Major == other.Major &&
		v.Minor == other.Minor &&
		v.Patch == other.Patch
}

// VersionValidator defines the interface for version constraint validation using Strategy Pattern (SOLID: Open/Closed)
type VersionValidator interface {
	Validate(dep TemplateDependency, depVersion *VersionInfo, reqVersion *VersionInfo) error
}

// ExactVersionValidator validates exact version matches (Strategy Pattern)
type ExactVersionValidator struct{}

func (v *ExactVersionValidator) Validate(dep TemplateDependency, depVersion *VersionInfo, reqVersion *VersionInfo) error {
	if !depVersion.Equals(reqVersion) {
		return DependencyError(
			fmt.Sprintf("dependent template '%s' version %s doesn't match required version %s",
				dep.Name, depVersion.String(), reqVersion.String()),
			nil,
		).WithContext("dependency_name", dep.Name).
			WithContext("dependency_version", depVersion.String()).
			WithContext("required_version", reqVersion.String()).
			WithContext("operator", "=")
	}
	return nil
}

// GreaterThanOrEqualValidator validates >= version constraints (Strategy Pattern)
type GreaterThanOrEqualValidator struct{}

func (v *GreaterThanOrEqualValidator) Validate(dep TemplateDependency, depVersion *VersionInfo, reqVersion *VersionInfo) error {
	if !depVersion.IsGreaterThan(reqVersion) && !depVersion.Equals(reqVersion) {
		return DependencyError(
			fmt.Sprintf("dependent template '%s' version %s is less than required version %s",
				dep.Name, depVersion.String(), reqVersion.String()),
			nil,
		).WithContext("dependency_name", dep.Name).
			WithContext("dependency_version", depVersion.String()).
			WithContext("required_version", reqVersion.String()).
			WithContext("operator", ">=")
	}
	return nil
}

// GreaterThanValidator validates > version constraints (Strategy Pattern)
type GreaterThanValidator struct{}

func (v *GreaterThanValidator) Validate(dep TemplateDependency, depVersion *VersionInfo, reqVersion *VersionInfo) error {
	if !depVersion.IsGreaterThan(reqVersion) {
		return DependencyError(
			fmt.Sprintf("dependent template '%s' version %s is not greater than required version %s",
				dep.Name, depVersion.String(), reqVersion.String()),
			nil,
		).WithContext("dependency_name", dep.Name).
			WithContext("dependency_version", depVersion.String()).
			WithContext("required_version", reqVersion.String()).
			WithContext("operator", ">")
	}
	return nil
}

// LessThanValidator validates < version constraints (Strategy Pattern)
type LessThanValidator struct{}

func (v *LessThanValidator) Validate(dep TemplateDependency, depVersion *VersionInfo, reqVersion *VersionInfo) error {
	if depVersion.IsGreaterThan(reqVersion) || depVersion.Equals(reqVersion) {
		return DependencyError(
			fmt.Sprintf("dependent template '%s' version %s is not less than required version %s",
				dep.Name, depVersion.String(), reqVersion.String()),
			nil,
		).WithContext("dependency_name", dep.Name).
			WithContext("dependency_version", depVersion.String()).
			WithContext("required_version", reqVersion.String()).
			WithContext("operator", "<")
	}
	return nil
}

// LessThanOrEqualValidator validates <= version constraints (Strategy Pattern)
type LessThanOrEqualValidator struct{}

func (v *LessThanOrEqualValidator) Validate(dep TemplateDependency, depVersion *VersionInfo, reqVersion *VersionInfo) error {
	if depVersion.IsGreaterThan(reqVersion) {
		return DependencyError(
			fmt.Sprintf("dependent template '%s' version %s is greater than required version %s",
				dep.Name, depVersion.String(), reqVersion.String()),
			nil,
		).WithContext("dependency_name", dep.Name).
			WithContext("dependency_version", depVersion.String()).
			WithContext("required_version", reqVersion.String()).
			WithContext("operator", "<=")
	}
	return nil
}
