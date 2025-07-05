// Package ami provides CloudWorkstation's AMI creation system.
package ami

import (
	"testing"
	"time"
)

// MockClock is a simple clock that returns a fixed time
type MockClock struct {
	CurrentTime time.Time
}

// Now returns the current time
func (m *MockClock) Now() time.Time {
	return m.CurrentTime
}

// TestNewVersionInfo tests parsing version strings
func TestNewVersionInfo(t *testing.T) {
	tests := []struct {
		version  string
		valid    bool
		expected VersionInfo
	}{
		{"1.2.3", true, VersionInfo{Major: 1, Minor: 2, Patch: 3}},
		{"0.1.0", true, VersionInfo{Major: 0, Minor: 1, Patch: 0}},
		{"10.20.30", true, VersionInfo{Major: 10, Minor: 20, Patch: 30}},
		{"1.2", false, VersionInfo{}},
		{"1.2.3.4", false, VersionInfo{}},
		{"1.2.x", false, VersionInfo{}},
		{"invalid", false, VersionInfo{}},
	}

	for _, test := range tests {
		version, err := NewVersionInfo(test.version)
		if test.valid {
			if err != nil {
				t.Errorf("Expected valid version %s, got error: %v", test.version, err)
			}
			if version == nil {
				t.Errorf("Expected non-nil version for %s", test.version)
				continue
			}
			if version.Major != test.expected.Major ||
				version.Minor != test.expected.Minor ||
				version.Patch != test.expected.Patch {
				t.Errorf("Expected version %v for %s, got %v", test.expected, test.version, *version)
			}
		} else {
			if err == nil {
				t.Errorf("Expected error for invalid version %s, got nil", test.version)
			}
		}
	}
}

// TestVersionCompare tests version comparison
func TestVersionCompare(t *testing.T) {
	tests := []struct {
		v1       VersionInfo
		v2       VersionInfo
		expected bool
	}{
		{VersionInfo{1, 0, 0}, VersionInfo{0, 9, 9}, true},
		{VersionInfo{1, 2, 3}, VersionInfo{1, 2, 2}, true},
		{VersionInfo{1, 2, 3}, VersionInfo{1, 1, 9}, true},
		{VersionInfo{1, 2, 3}, VersionInfo{1, 2, 3}, false},
		{VersionInfo{1, 2, 3}, VersionInfo{1, 2, 4}, false},
		{VersionInfo{1, 2, 3}, VersionInfo{1, 3, 0}, false},
		{VersionInfo{1, 2, 3}, VersionInfo{2, 0, 0}, false},
	}

	for _, test := range tests {
		result := test.v1.IsGreaterThan(&test.v2)
		if result != test.expected {
			t.Errorf("Expected %v > %v to be %v, got %v",
				test.v1, test.v2, test.expected, result)
		}
	}
}

// TestVersionEquals tests version equality
func TestVersionEquals(t *testing.T) {
	tests := []struct {
		v1       VersionInfo
		v2       VersionInfo
		expected bool
	}{
		{VersionInfo{1, 2, 3}, VersionInfo{1, 2, 3}, true},
		{VersionInfo{1, 2, 3}, VersionInfo{1, 2, 4}, false},
		{VersionInfo{1, 2, 3}, VersionInfo{1, 3, 3}, false},
		{VersionInfo{1, 2, 3}, VersionInfo{2, 2, 3}, false},
	}

	for _, test := range tests {
		result := test.v1.Equals(&test.v2)
		if result != test.expected {
			t.Errorf("Expected %v equals %v to be %v, got %v",
				test.v1, test.v2, test.expected, result)
		}
	}
}

// TestVersionIncrement tests version incrementing
func TestVersionIncrement(t *testing.T) {
	v1 := VersionInfo{1, 2, 3}
	v1.IncrementMajor()
	expected1 := VersionInfo{2, 0, 0}
	if v1.Major != expected1.Major || v1.Minor != expected1.Minor || v1.Patch != expected1.Patch {
		t.Errorf("Expected %v after IncrementMajor, got %v", expected1, v1)
	}

	v2 := VersionInfo{1, 2, 3}
	v2.IncrementMinor()
	expected2 := VersionInfo{1, 3, 0}
	if v2.Major != expected2.Major || v2.Minor != expected2.Minor || v2.Patch != expected2.Patch {
		t.Errorf("Expected %v after IncrementMinor, got %v", expected2, v2)
	}

	v3 := VersionInfo{1, 2, 3}
	v3.IncrementPatch()
	expected3 := VersionInfo{1, 2, 4}
	if v3.Major != expected3.Major || v3.Minor != expected3.Minor || v3.Patch != expected3.Patch {
		t.Errorf("Expected %v after IncrementPatch, got %v", expected3, v3)
	}
}

// TestGetSetTemplateVersion tests getting and setting template versions
func TestGetSetTemplateVersion(t *testing.T) {
	// Create test base AMIs
	baseAMIs := map[string]map[string]string{
		"us-east-1": {
			"ubuntu-22.04-server-lts": "ami-12345678",
		},
	}

	// Create parser
	parser := NewParser(baseAMIs)

	// Create mock clock
	mockClock := &MockClock{
		CurrentTime: time.Date(2023, 7, 1, 12, 0, 0, 0, time.UTC),
	}

	// Create template manager with mock clock
	manager := NewTemplateManager(parser, nil, "")
	// Inject mock clock
	manager.clock = mockClock

	// Create a test template
	template := &Template{
		Name:        "version-test",
		Base:        "ubuntu-22.04-server-lts",
		Description: "Version Test Template",
		BuildSteps: []BuildStep{
			{
				Name:   "update",
				Script: "apt-get update -y",
			},
		},
	}

	// Import the template
	_, err := manager.ImportFromTemplate(template, nil)
	if err != nil {
		t.Fatalf("Failed to import template: %v", err)
	}

	// Test default version
	version, err := manager.GetTemplateVersion("version-test")
	if err != nil {
		t.Fatalf("Failed to get template version: %v", err)
	}
	if version.String() != "1.0.0" {
		t.Errorf("Expected default version 1.0.0, got %s", version.String())
	}

	// Test setting version
	newVersion := &VersionInfo{Major: 2, Minor: 1, Patch: 3}
	err = manager.SetTemplateVersion("version-test", newVersion)
	if err != nil {
		t.Fatalf("Failed to set template version: %v", err)
	}

	// Verify version was set
	version, err = manager.GetTemplateVersion("version-test")
	if err != nil {
		t.Fatalf("Failed to get template version: %v", err)
	}
	if version.String() != "2.1.3" {
		t.Errorf("Expected version 2.1.3, got %s", version.String())
	}

	// Test incrementing version
	version, err = manager.IncrementTemplateVersion("version-test", "minor")
	if err != nil {
		t.Fatalf("Failed to increment template version: %v", err)
	}
	if version.String() != "2.2.0" {
		t.Errorf("Expected version 2.2.0, got %s", version.String())
	}

	// Verify version was incremented
	version, err = manager.GetTemplateVersion("version-test")
	if err != nil {
		t.Fatalf("Failed to get template version: %v", err)
	}
	if version.String() != "2.2.0" {
		t.Errorf("Expected version 2.2.0, got %s", version.String())
	}

	// Test invalid version component
	_, err = manager.IncrementTemplateVersion("version-test", "invalid")
	if err == nil {
		t.Error("Expected error for invalid version component, got nil")
	}
}

// TestVersionTemplate tests creating versioned templates
func TestVersionTemplate(t *testing.T) {
	// Create test base AMIs
	baseAMIs := map[string]map[string]string{
		"us-east-1": {
			"ubuntu-22.04-server-lts": "ami-12345678",
		},
	}

	// Create parser
	parser := NewParser(baseAMIs)

	// Create mock clock
	mockClock := &MockClock{
		CurrentTime: time.Date(2023, 7, 1, 12, 0, 0, 0, time.UTC),
	}

	// Create template manager with mock clock
	manager := NewTemplateManager(parser, nil, "")
	// Inject mock clock
	manager.clock = mockClock

	// Create a test template
	template := &Template{
		Name:        "version-template-test",
		Base:        "ubuntu-22.04-server-lts",
		Description: "Version Template Test",
		BuildSteps: []BuildStep{
			{
				Name:   "update",
				Script: "apt-get update -y",
			},
		},
	}

	// Import the template
	_, err := manager.ImportFromTemplate(template, nil)
	if err != nil {
		t.Fatalf("Failed to import template: %v", err)
	}

	// Create a new versioned template
	newTemplate, err := manager.VersionTemplate("version-template-test", nil, "minor")
	if err != nil {
		t.Fatalf("Failed to version template: %v", err)
	}

	// Verify template properties
	if newTemplate.Name != "version-template-test" {
		t.Errorf("Expected template name 'version-template-test', got '%s'", newTemplate.Name)
	}
	if len(newTemplate.BuildSteps) != 1 {
		t.Errorf("Expected 1 build step, got %d", len(newTemplate.BuildSteps))
	}

	// Verify version was incremented
	version, err := manager.GetTemplateVersion("version-template-test")
	if err != nil {
		t.Fatalf("Failed to get template version: %v", err)
	}
	if version.String() != "1.1.0" {
		t.Errorf("Expected version 1.1.0, got %s", version.String())
	}
}

// TestCreateTemplateVersion tests creating a new template version with builder
func TestCreateTemplateVersion(t *testing.T) {
	// Create test base AMIs
	baseAMIs := map[string]map[string]string{
		"us-east-1": {
			"ubuntu-22.04-server-lts": "ami-12345678",
		},
	}

	// Create parser
	parser := NewParser(baseAMIs)

	// Create mock clock
	mockClock := &MockClock{
		CurrentTime: time.Date(2023, 7, 1, 12, 0, 0, 0, time.UTC),
	}

	// Create template manager with mock clock
	manager := NewTemplateManager(parser, nil, "")
	// Inject mock clock
	manager.clock = mockClock

	// Create a test template
	template := &Template{
		Name:        "create-version-test",
		Base:        "ubuntu-22.04-server-lts",
		Description: "Create Version Test",
		BuildSteps: []BuildStep{
			{
				Name:   "update",
				Script: "apt-get update -y",
			},
		},
	}

	// Import the template
	_, err := manager.ImportFromTemplate(template, nil)
	if err != nil {
		t.Fatalf("Failed to import template: %v", err)
	}

	// Create a new version with builder
	builder, err := manager.CreateTemplateVersion("create-version-test", "minor")
	if err != nil {
		t.Fatalf("Failed to create template version: %v", err)
	}

	// Modify the template
	builder.AddBuildStep("install-tools", "apt-get install -y curl wget git")
	builder.WithTag("version", "modified")

	// Build the template
	newTemplate, err := builder.Build()
	if err != nil {
		t.Fatalf("Failed to build template version: %v", err)
	}

	// Verify template properties
	if newTemplate.Name != "create-version-test" {
		t.Errorf("Expected template name 'create-version-test', got '%s'", newTemplate.Name)
	}
	if len(newTemplate.BuildSteps) != 2 {
		t.Errorf("Expected 2 build steps, got %d", len(newTemplate.BuildSteps))
	}
	if newTemplate.Tags["version"] != "modified" {
		t.Errorf("Expected tag 'version: modified', got '%s'", newTemplate.Tags["version"])
	}

	// Verify version was incremented
	version, err := manager.GetTemplateVersion("create-version-test")
	if err != nil {
		t.Fatalf("Failed to get template version: %v", err)
	}
	if version.String() != "1.1.0" {
		t.Errorf("Expected version 1.1.0, got %s", version.String())
	}
}