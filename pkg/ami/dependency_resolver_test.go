// Package ami provides CloudWorkstation's AMI creation system.
package ami

import (
	"context"
	"fmt"
	"reflect"
	"testing"
	"time"
)

// MockRegistry implements a fake registry for testing
type MockRegistry struct {
	Templates      map[string]map[string]string // template -> version -> data
	VersionMap     map[string][]string          // template -> versions
	LatestVersions map[string]string            // template -> latest version
	FailLookup     bool
	FailList       bool
}

func (m *MockRegistry) ListSharedTemplateVersions(_ context.Context, templateName string) ([]string, error) {
	if m.FailList {
		return nil, fmt.Errorf("mock list error")
	}
	if versions, ok := m.VersionMap[templateName]; ok {
		return versions, nil
	}
	return []string{}, nil
}

func (m *MockRegistry) GetSharedTemplate(_ context.Context, templateName, version string) (*SharedTemplateEntry, error) {
	if m.FailLookup {
		return nil, fmt.Errorf("mock lookup error")
	}

	// Use latest version if not specified
	if version == "" {
		version = m.LatestVersions[templateName]
	}

	// Check if template exists
	templateVersions, ok := m.Templates[templateName]
	if !ok {
		return nil, fmt.Errorf("template not found")
	}

	// Check if version exists
	templateData, ok := templateVersions[version]
	if !ok {
		return nil, fmt.Errorf("version not found")
	}

	// Return entry
	return &SharedTemplateEntry{
		Name:        templateName,
		Version:     version,
		TemplateData: templateData,
		PublishedAt: time.Now(),
		PublishedBy: "test-user",
	}, nil
}

func (m *MockRegistry) ListSharedTemplates(_ context.Context) (map[string]*SharedTemplateEntry, error) {
	if m.FailLookup {
		return nil, fmt.Errorf("mock lookup error")
	}

	entries := make(map[string]*SharedTemplateEntry)
	for name, versions := range m.Templates {
		latestVersion := m.LatestVersions[name]
		templateData := versions[latestVersion]
		entries[name] = &SharedTemplateEntry{
			Name:        name,
			Version:     latestVersion,
			TemplateData: templateData,
			PublishedAt: time.Now(),
			PublishedBy: "test-user",
		}
	}
	return entries, nil
}

// TestClock implements a predictable clock for testing
type TestClock struct {
	CurrentTime time.Time
}

func (c *TestClock) Now() time.Time {
	return c.CurrentTime
}

// setupTestTemplateManager creates a template manager with test templates
func setupTestTemplateManager(t *testing.T) *TemplateManager {
	t.Helper()

	// Create template manager
	manager := &TemplateManager{
		Templates:        make(map[string]*Template),
		TemplateMetadata: make(map[string]TemplateMetadata),
	}

	// Set up test clock
	testTime := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	manager.clock = &TestClock{CurrentTime: testTime}

	// Add test templates
	templates := map[string]*Template{
		"base": {
			Name:         "base",
			Description:  "Base template",
			Dependencies: []TemplateDependency{},
		},
		"python": {
			Name:        "python",
			Description: "Python template",
			Dependencies: []TemplateDependency{
				{
					Name:            "base",
					Version:         "1.0.0",
					VersionOperator: ">=",
				},
			},
		},
		"ml": {
			Name:        "ml",
			Description: "Machine learning template",
			Dependencies: []TemplateDependency{
				{
					Name:            "python",
					Version:         "2.0.0",
					VersionOperator: ">=",
				},
				{
					Name:            "r-base",
					Version:         "1.5.0",
					VersionOperator: ">=",
					Optional:        true,
				},
			},
		},
		"data-science": {
			Name:        "data-science",
			Description: "Data science template",
			Dependencies: []TemplateDependency{
				{
					Name:            "python",
					Version:         "2.0.0",
					VersionOperator: ">=",
				},
				{
					Name:            "r-base",
					Version:         "1.0.0",
					VersionOperator: ">=",
				},
			},
		},
		"r-base": {
			Name:        "r-base",
			Description: "R base template",
			Dependencies: []TemplateDependency{
				{
					Name:            "base",
					Version:         "1.0.0",
					VersionOperator: ">=",
				},
			},
		},
		"empty": {
			Name:         "empty",
			Description:  "Empty template",
			Dependencies: []TemplateDependency{},
		},
	}

	// Add templates to manager
	for name, template := range templates {
		manager.Templates[name] = template
		manager.TemplateMetadata[name] = TemplateMetadata{
			Version:      "1.0.0",
			LastModified: manager.clock.Now(),
			SourceURL:    "local://" + name,
		}
	}

	// Update specific versions
	metadata := manager.TemplateMetadata["base"]
	metadata.Version = "2.0.0"
	manager.TemplateMetadata["base"] = metadata
	
	metadata = manager.TemplateMetadata["python"]
	metadata.Version = "3.0.0"
	manager.TemplateMetadata["python"] = metadata
	
	metadata = manager.TemplateMetadata["r-base"]
	metadata.Version = "1.5.0"
	manager.TemplateMetadata["r-base"] = metadata

	// Create mock registry
	mockRegistry := &MockRegistry{
		Templates: map[string]map[string]string{
			"base": {
				"1.0.0": "name: base\nversion: 1.0.0",
				"2.0.0": "name: base\nversion: 2.0.0",
			},
			"python": {
				"2.0.0": "name: python\nversion: 2.0.0",
				"3.0.0": "name: python\nversion: 3.0.0",
			},
			"r-base": {
				"1.0.0": "name: r-base\nversion: 1.0.0",
				"1.5.0": "name: r-base\nversion: 1.5.0",
				"2.0.0": "name: r-base\nversion: 2.0.0",
			},
			"extra": {
				"1.0.0": "name: extra\nversion: 1.0.0",
			},
		},
		VersionMap: map[string][]string{
			"base":    {"1.0.0", "2.0.0"},
			"python":  {"2.0.0", "3.0.0"},
			"r-base":  {"1.0.0", "1.5.0", "2.0.0"},
			"extra":   {"1.0.0"},
		},
		LatestVersions: map[string]string{
			"base":    "2.0.0",
			"python":  "3.0.0",
			"r-base":  "2.0.0",
			"extra":   "1.0.0",
		},
	}

	// Set registry (type casting)
	r := Registry(*mockRegistry)
	manager.Registry = &r
	
	// Add Parser (simple mock)
	manager.Parser = &Parser{}

	return manager
}

// MockParser is a simple parser for testing
func (p *Parser) ParseTemplateFromString(data string) (*Template, error) {
	// Simple mock implementation for testing
	if data == "" {
		return nil, fmt.Errorf("empty template data")
	}
	return &Template{
		Name: "mock-template",
		Description: "Mock template from string",
		Dependencies: []TemplateDependency{},
	}, nil
}

// TestResolveDependencies tests dependency resolution
func TestResolveDependencies(t *testing.T) {
	manager := setupTestTemplateManager(t)
	resolver := NewDependencyResolver(manager)

	tests := []struct {
		name            string
		templateName    string
		expectError     bool
		expectedDeps    int
		expectedBuildOrder []string
	}{
		{
			name:            "No dependencies",
			templateName:    "base",
			expectError:     false,
			expectedDeps:    0,
			expectedBuildOrder: []string{"base"},
		},
		{
			name:            "Simple dependency",
			templateName:    "python",
			expectError:     false,
			expectedDeps:    1,
			expectedBuildOrder: []string{"base", "python"},
		},
		{
			name:            "Multiple dependencies with optional",
			templateName:    "ml",
			expectError:     false,
			expectedDeps:    2,
			expectedBuildOrder: []string{"base", "python", "r-base", "ml"},
		},
		{
			name:            "Non-existent template",
			templateName:    "nonexistent",
			expectError:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resolved, graph, err := resolver.ResolveDependencies(tt.templateName)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if len(resolved) != tt.expectedDeps {
				t.Errorf("Expected %d dependencies, got %d", tt.expectedDeps, len(resolved))
			}

			if !reflect.DeepEqual(graph, tt.expectedBuildOrder) {
				t.Errorf("Expected build order %v, got %v", tt.expectedBuildOrder, graph)
			}

			// Check dependency status
			for name, dep := range resolved {
				if dep.Status != "satisfied" {
					t.Errorf("Expected dependency %s to be satisfied, got %s", name, dep.Status)
				}
			}
		})
	}
}

// TestCheckVersionConstraint tests version constraint checking
func TestCheckVersionConstraint(t *testing.T) {
	manager := setupTestTemplateManager(t)
	resolver := NewDependencyResolver(manager)

	tests := []struct {
		name       string
		version    string
		constraint string
		operator   string
		expected   bool
		expectError bool
	}{
		{
			name:       "Equal versions",
			version:    "1.0.0",
			constraint: "1.0.0",
			operator:   "=",
			expected:   true,
		},
		{
			name:       "Greater than",
			version:    "2.0.0",
			constraint: "1.0.0",
			operator:   ">",
			expected:   true,
		},
		{
			name:       "Not greater than",
			version:    "1.0.0",
			constraint: "2.0.0",
			operator:   ">",
			expected:   false,
		},
		{
			name:       "Greater than or equal (equal)",
			version:    "1.0.0",
			constraint: "1.0.0",
			operator:   ">=",
			expected:   true,
		},
		{
			name:       "Greater than or equal (greater)",
			version:    "2.0.0",
			constraint: "1.0.0",
			operator:   ">=",
			expected:   true,
		},
		{
			name:       "Less than",
			version:    "1.0.0",
			constraint: "2.0.0",
			operator:   "<",
			expected:   true,
		},
		{
			name:       "Not less than",
			version:    "2.0.0",
			constraint: "1.0.0",
			operator:   "<",
			expected:   false,
		},
		{
			name:       "Less than or equal (equal)",
			version:    "1.0.0",
			constraint: "1.0.0",
			operator:   "<=",
			expected:   true,
		},
		{
			name:       "Less than or equal (less)",
			version:    "1.0.0",
			constraint: "2.0.0",
			operator:   "<=",
			expected:   true,
		},
		{
			name:       "Compatible version (same major, greater minor)",
			version:    "1.5.0",
			constraint: "1.0.0",
			operator:   "~>",
			expected:   true,
		},
		{
			name:       "Compatible version (different major)",
			version:    "2.0.0",
			constraint: "1.0.0",
			operator:   "~>",
			expected:   false,
		},
		{
			name:       "Default operator (>=)",
			version:    "2.0.0",
			constraint: "1.0.0",
			operator:   "",
			expected:   true,
		},
		{
			name:       "Invalid version",
			version:    "invalid",
			constraint: "1.0.0",
			operator:   ">=",
			expectError: true,
		},
		{
			name:       "Invalid constraint",
			version:    "1.0.0",
			constraint: "invalid",
			operator:   ">=",
			expectError: true,
		},
		{
			name:       "Invalid operator",
			version:    "1.0.0",
			constraint: "1.0.0",
			operator:   "invalid",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := resolver.checkVersionConstraint(tt.version, tt.constraint, tt.operator)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

// TestFindCompatibleVersions tests finding compatible versions
func TestFindCompatibleVersions(t *testing.T) {
	manager := setupTestTemplateManager(t)
	resolver := NewDependencyResolver(manager)

	tests := []struct {
		name            string
		templateName    string
		constraint      string
		operator        string
		expectError     bool
		expectedVersions []string
		mockRegistry    *MockRegistry
	}{
		{
			name:            "Find compatible versions (>=)",
			templateName:    "r-base",
			constraint:      "1.0.0",
			operator:        ">=",
			expectedVersions: []string{"2.0.0", "1.5.0", "1.0.0"},
		},
		{
			name:            "Find compatible versions (<)",
			templateName:    "r-base",
			constraint:      "2.0.0",
			operator:        "<",
			expectedVersions: []string{"1.5.0", "1.0.0"},
		},
		{
			name:            "Find compatible versions (~>)",
			templateName:    "r-base",
			constraint:      "1.0.0",
			operator:        "~>",
			expectedVersions: []string{"1.5.0", "1.0.0"},
		},
		{
			name:            "No compatible versions",
			templateName:    "r-base",
			constraint:      "3.0.0",
			operator:        "=",
			expectedVersions: []string{},
		},
		{
			name:            "Registry list error",
			templateName:    "r-base",
			constraint:      "1.0.0",
			operator:        ">=",
			expectError:     true,
			mockRegistry: &MockRegistry{
				FailList: true,
			},
		},
		{
			name:            "Non-existent template",
			templateName:    "nonexistent",
			constraint:      "1.0.0",
			operator:        ">=",
			expectedVersions: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Use custom mock registry if provided
			if tt.mockRegistry != nil {
				originalRegistry := manager.Registry
				r := Registry(*tt.mockRegistry)
				manager.Registry = &r
				defer func() { manager.Registry = originalRegistry }()
			}

			versions, err := resolver.FindCompatibleVersions(tt.templateName, tt.constraint, tt.operator)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if !reflect.DeepEqual(versions, tt.expectedVersions) {
				t.Errorf("Expected versions %v, got %v", tt.expectedVersions, versions)
			}
		})
	}
}

// TestResolveDependencyConflicts tests resolving dependency conflicts
func TestResolveDependencyConflicts(t *testing.T) {
	manager := setupTestTemplateManager(t)
	resolver := NewDependencyResolver(manager)

	tests := []struct {
		name          string
		conflicts     map[string][]TemplateDependency
		expectError   bool
		expectedResolution map[string]string
	}{
		{
			name: "No conflicts",
			conflicts: map[string][]TemplateDependency{
				"base": {
					{
						Name:            "base",
						Version:         "1.0.0",
						VersionOperator: ">=",
					},
				},
			},
			expectedResolution: map[string]string{
				"base": "1.0.0",
			},
		},
		{
			name: "Compatible version requirements",
			conflicts: map[string][]TemplateDependency{
				"python": {
					{
						Name:            "python",
						Version:         "2.0.0",
						VersionOperator: ">=",
					},
					{
						Name:            "python",
						Version:         "2.5.0",
						VersionOperator: ">=",
					},
				},
			},
			expectedResolution: map[string]string{
				"python": "2.5.0",
			},
		},
		{
			name: "Incompatible exact versions",
			conflicts: map[string][]TemplateDependency{
				"base": {
					{
						Name:            "base",
						Version:         "1.0.0",
						VersionOperator: "=",
					},
					{
						Name:            "base",
						Version:         "2.0.0",
						VersionOperator: "=",
					},
				},
			},
			expectError: true,
		},
		{
			name: "Incompatible bounds",
			conflicts: map[string][]TemplateDependency{
				"base": {
					{
						Name:            "base",
						Version:         "2.0.0",
						VersionOperator: ">=",
					},
					{
						Name:            "base",
						Version:         "1.0.0",
						VersionOperator: "<=",
					},
				},
			},
			expectError: true,
		},
		{
			name: "Mixed constraints",
			conflicts: map[string][]TemplateDependency{
				"python": {
					{
						Name:            "python",
						Version:         "2.0.0",
						VersionOperator: ">=",
					},
					{
						Name:            "python",
						Version:         "3.0.0",
						VersionOperator: "<",
					},
				},
			},
			expectedResolution: map[string]string{
				"python": "2.0.0",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resolution, err := resolver.ResolveDependencyConflicts(tt.conflicts)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if !reflect.DeepEqual(resolution, tt.expectedResolution) {
				t.Errorf("Expected resolution %v, got %v", tt.expectedResolution, resolution)
			}
		})
	}
}

// TestResolveAndFetchDependencies tests resolving and fetching dependencies
func TestResolveAndFetchDependencies(t *testing.T) {
	manager := setupTestTemplateManager(t)
	resolver := NewDependencyResolver(manager)

	// Add missing dependency to ml template
	mlTemplate := manager.Templates["ml"]
	mlTemplate.Dependencies = append(mlTemplate.Dependencies, TemplateDependency{
		Name:     "extra",
		Optional: true,
	})

	tests := []struct {
		name          string
		templateName  string
		fetchMissing  bool
		expectError   bool
		expectedFetch int
	}{
		{
			name:         "Resolve without fetching",
			templateName: "ml",
			fetchMissing: false,
			expectedFetch: 0,
		},
		{
			name:         "Resolve with fetching",
			templateName: "ml",
			fetchMissing: true,
			expectedFetch: 1,
		},
		{
			name:         "Resolve with registry error",
			templateName: "ml",
			fetchMissing: true,
			expectError:  true,
		},
		{
			name:         "Resolve non-existent template",
			templateName: "nonexistent",
			fetchMissing: true,
			expectError:  true,
		},
	}

	for i, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set up registry failure for specific test
			if i == 2 {
				originalRegistry := manager.Registry
				mockReg := MockRegistry{FailLookup: true}
				r := Registry(mockReg)
				manager.Registry = &r
				defer func() { manager.Registry = originalRegistry }()
			}

			resolved, fetched, err := resolver.ResolveAndFetchDependencies(tt.templateName, tt.fetchMissing)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if len(fetched) != tt.expectedFetch {
				t.Errorf("Expected %d fetched templates, got %d", tt.expectedFetch, len(fetched))
			}

			// Check if extra dependency was fetched
			if tt.fetchMissing && len(fetched) > 0 {
				if fetched[0] != "extra" {
					t.Errorf("Expected to fetch 'extra' dependency, got %s", fetched[0])
				}

				if resolved["extra"].Status != "satisfied" {
					t.Errorf("Expected 'extra' dependency to be satisfied, got %s", resolved["extra"].Status)
				}
			}
		})
	}
}

// TestDependencyResolverEdgeCases tests edge cases
func TestDependencyResolverEdgeCases(t *testing.T) {
	manager := setupTestTemplateManager(t)
	resolver := NewDependencyResolver(manager)

	// Test circular dependency
	circularTemplate := &Template{
		Name:        "circular",
		Description: "Circular dependency template",
		Dependencies: []TemplateDependency{
			{
				Name: "circular2",
			},
		},
	}
	manager.Templates["circular"] = circularTemplate

	circularTemplate2 := &Template{
		Name:        "circular2",
		Description: "Circular dependency template 2",
		Dependencies: []TemplateDependency{
			{
				Name: "circular",
			},
		},
	}
	manager.Templates["circular2"] = circularTemplate2

	// Test with only optional dependencies
	optionalTemplate := &Template{
		Name:        "optional-only",
		Description: "Template with only optional dependencies",
		Dependencies: []TemplateDependency{
			{
				Name:     "nonexistent",
				Optional: true,
			},
		},
	}
	manager.Templates["optional-only"] = optionalTemplate

	// Test with no registry
	t.Run("No registry configured", func(t *testing.T) {
		originalRegistry := manager.Registry
		manager.Registry = nil
		defer func() { manager.Registry = originalRegistry }()

		_, _, err := resolver.ResolveAndFetchDependencies("python", true)
		if err == nil {
			t.Errorf("Expected error when no registry configured, got none")
		}
	})

	// Test circular dependency
	t.Run("Circular dependency", func(t *testing.T) {
		_, _, err := resolver.ResolveDependencies("circular")
		if err == nil {
			t.Errorf("Expected error for circular dependency, got none")
		}
	})

	// Test with only optional dependencies
	t.Run("Only optional dependencies", func(t *testing.T) {
		resolved, _, err := resolver.ResolveDependencies("optional-only")
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
			return
		}

		if len(resolved) != 1 {
			t.Errorf("Expected 1 resolved dependency, got %d", len(resolved))
		}

		if resolved["nonexistent"].Status != "missing" {
			t.Errorf("Expected optional dependency status to be 'missing', got %s", 
				resolved["nonexistent"].Status)
		}
	})
}

// TestDependencyGraphGeneration tests generating dependency graphs
func TestDependencyGraphGeneration(t *testing.T) {
	manager := setupTestTemplateManager(t)

	tests := []struct {
		name            string
		templateName    string
		expectError     bool
		expectedOrder   []string
	}{
		{
			name:          "Simple dependency",
			templateName:  "python",
			expectedOrder: []string{"base", "python"},
		},
		{
			name:          "Multiple dependencies",
			templateName:  "data-science",
			expectedOrder: []string{"base", "r-base", "python", "data-science"},
		},
		{
			name:          "No dependencies",
			templateName:  "base",
			expectedOrder: []string{"base"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			graph, err := manager.GetDependencyGraph(tt.templateName)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if !reflect.DeepEqual(graph, tt.expectedOrder) {
				t.Errorf("Expected order %v, got %v", tt.expectedOrder, graph)
			}
		})
	}
}