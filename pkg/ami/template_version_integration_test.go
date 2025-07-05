// Package ami provides CloudWorkstation's AMI creation system.
package ami

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestTemplateVersioningIntegration tests the full template versioning workflow
// This is an integration test that exercises the full template versioning system
func TestTemplateVersioningIntegration(t *testing.T) {
	// Skip in short mode
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Create a temporary directory for test templates
	tempDir, err := os.MkdirTemp("", "template-versioning-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create test base AMIs
	baseAMIs := map[string]map[string]string{
		"us-east-1": {
			"ubuntu-22.04-server-lts": "ami-12345678",
		},
	}

	// Create parser
	parser := NewParser(baseAMIs)

	// Create mock clock for deterministic testing
	mockClock := &MockClock{
		CurrentTime: time.Date(2023, 7, 1, 12, 0, 0, 0, time.UTC),
	}

	// Create template manager with mock clock
	manager := NewTemplateManager(parser, nil, tempDir)
	// Inject mock clock
	manager.clock = mockClock

	// PHASE 1: Create initial template
	t.Log("Phase 1: Creating initial template")
	template := createTestTemplate(t, manager, "versioning-test", "1.0.0")

	// Verify template was created with initial version
	version, err := manager.GetTemplateVersion("versioning-test")
	if err != nil {
		t.Fatalf("Failed to get template version: %v", err)
	}
	if version.String() != "1.0.0" {
		t.Errorf("Expected initial version 1.0.0, got %s", version.String())
	}

	// PHASE 2: Create minor version update
	t.Log("Phase 2: Creating minor version update")
	builder, err := manager.CreateTemplateVersion("versioning-test", "minor")
	if err != nil {
		t.Fatalf("Failed to create template version: %v", err)
	}

	// Add a new build step
	builder.AddBuildStep("install-tools", "apt-get install -y curl wget git")
	
	// Build the template
	minorTemplate, err := builder.Build()
	if err != nil {
		t.Fatalf("Failed to build minor version: %v", err)
	}

	// Verify version was incremented
	version, err = manager.GetTemplateVersion("versioning-test")
	if err != nil {
		t.Fatalf("Failed to get template version: %v", err)
	}
	if version.String() != "1.1.0" {
		t.Errorf("Expected minor version update to 1.1.0, got %s", version.String())
	}

	// Verify build steps were updated
	if len(minorTemplate.BuildSteps) != 2 {
		t.Errorf("Expected 2 build steps after minor update, got %d", len(minorTemplate.BuildSteps))
	}

	// PHASE 3: Create patch version update
	t.Log("Phase 3: Creating patch version update")
	builder, err = manager.CreateTemplateVersion("versioning-test", "patch")
	if err != nil {
		t.Fatalf("Failed to create template version: %v", err)
	}

	// Update a build step
	builder.AddBuildStepWithTimeout("optimize", "apt-get autoremove -y", 120)
	
	// Build the template
	patchTemplate, err := builder.Build()
	if err != nil {
		t.Fatalf("Failed to build patch version: %v", err)
	}

	// Verify version was incremented
	version, err = manager.GetTemplateVersion("versioning-test")
	if err != nil {
		t.Fatalf("Failed to get template version: %v", err)
	}
	if version.String() != "1.1.1" {
		t.Errorf("Expected patch version update to 1.1.1, got %s", version.String())
	}

	// Verify build steps were updated
	if len(patchTemplate.BuildSteps) != 3 {
		t.Errorf("Expected 3 build steps after patch update, got %d", len(patchTemplate.BuildSteps))
	}

	// PHASE 4: Create major version update
	t.Log("Phase 4: Creating major version update")
	builder, err = manager.CreateTemplateVersion("versioning-test", "major")
	if err != nil {
		t.Fatalf("Failed to create template version: %v", err)
	}

	// Change base and architecture
	builder.WithBase("ubuntu-22.04-server-lts-arm64")
	builder.WithArchitecture("arm64")
	
	// Build the template
	majorTemplate, err := builder.Build()
	if err != nil {
		t.Fatalf("Failed to build major version: %v", err)
	}

	// Verify version was incremented
	version, err = manager.GetTemplateVersion("versioning-test")
	if err != nil {
		t.Fatalf("Failed to get template version: %v", err)
	}
	if version.String() != "2.0.0" {
		t.Errorf("Expected major version update to 2.0.0, got %s", version.String())
	}

	// Verify base and architecture were updated
	if majorTemplate.Base != "ubuntu-22.04-server-lts-arm64" {
		t.Errorf("Expected base to be updated to ubuntu-22.04-server-lts-arm64, got %s", majorTemplate.Base)
	}
	if majorTemplate.Architecture != "arm64" {
		t.Errorf("Expected architecture to be updated to arm64, got %s", majorTemplate.Architecture)
	}

	// Export the templates to files to verify persistence
	t.Log("Exporting templates to files")
	outputPath := filepath.Join(tempDir, "versioning-test-2.0.0.yaml")
	err = manager.ExportToFile("versioning-test", outputPath, nil)
	if err != nil {
		t.Fatalf("Failed to export template: %v", err)
	}

	// Read the file and verify it contains the version
	content, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("Failed to read exported template file: %v", err)
	}
	if !strings.Contains(string(content), "version: 2.0.0") {
		t.Errorf("Exported template file does not contain version information")
	}
}

// TestTemplateDependencyIntegration tests the template dependency management system
func TestTemplateDependencyIntegration(t *testing.T) {
	// Skip in short mode
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Create a temporary directory for test templates
	tempDir, err := os.MkdirTemp("", "template-dependency-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create test base AMIs
	baseAMIs := map[string]map[string]string{
		"us-east-1": {
			"ubuntu-22.04-server-lts": "ami-12345678",
		},
	}

	// Create parser
	parser := NewParser(baseAMIs)

	// Create mock clock for deterministic testing
	mockClock := &MockClock{
		CurrentTime: time.Date(2023, 7, 1, 12, 0, 0, 0, time.UTC),
	}

	// Create template manager with mock clock
	manager := NewTemplateManager(parser, nil, tempDir)
	// Inject mock clock
	manager.clock = mockClock

	// PHASE 1: Create base templates
	t.Log("Phase 1: Creating base templates")
	createTestTemplate(t, manager, "base-template", "1.0.0")
	createTestTemplate(t, manager, "util-template", "1.2.0")
	createTestTemplate(t, manager, "app-template", "2.0.0")

	// PHASE 2: Add dependencies
	t.Log("Phase 2: Adding dependencies")
	
	// Add dependency: app-template depends on base-template >= 1.0.0
	err = manager.AddDependency("app-template", TemplateDependency{
		Name:            "base-template",
		Version:         "1.0.0",
		VersionOperator: ">=",
	})
	if err != nil {
		t.Fatalf("Failed to add dependency: %v", err)
	}

	// Add dependency: app-template depends on util-template >= 1.2.0
	err = manager.AddDependency("app-template", TemplateDependency{
		Name:            "util-template",
		Version:         "1.2.0",
		VersionOperator: ">=",
	})
	if err != nil {
		t.Fatalf("Failed to add dependency: %v", err)
	}

	// Verify dependencies were added
	appTemplate, err := manager.GetTemplate("app-template")
	if err != nil {
		t.Fatalf("Failed to get template: %v", err)
	}
	if len(appTemplate.Dependencies) != 2 {
		t.Errorf("Expected 2 dependencies, got %d", len(appTemplate.Dependencies))
	}

	// PHASE 3: Validate dependencies
	t.Log("Phase 3: Validating dependencies")
	err = manager.ValidateTemplateDependencies("app-template", appTemplate.Dependencies)
	if err != nil {
		t.Errorf("Dependency validation failed: %v", err)
	}

	// PHASE 4: Generate dependency graph
	t.Log("Phase 4: Generating dependency graph")
	graph, err := manager.GetDependencyGraph("app-template")
	if err != nil {
		t.Fatalf("Failed to get dependency graph: %v", err)
	}
	
	// Verify graph order (dependencies should come before the app template)
	if len(graph) != 3 {
		t.Errorf("Expected 3 templates in dependency graph, got %d", len(graph))
	}
	if graph[len(graph)-1] != "app-template" {
		t.Errorf("Expected app-template to be last in build order, got %s", graph[len(graph)-1])
	}

	// PHASE 5: Test version constraints
	t.Log("Phase 5: Testing version constraints")
	
	// Update base-template to incompatible version
	baseTemplate, err := manager.GetTemplate("base-template")
	if err != nil {
		t.Fatalf("Failed to get template: %v", err)
	}
	
	// Create builder for new version
	builder := manager.CreateTemplate("incompatible-base", "Incompatible Base Template").
		WithBase(baseTemplate.Base)
	for _, step := range baseTemplate.BuildSteps {
		builder.AddBuildStep(step.Name, step.Script)
	}
	
	// Build the template
	incompatibleTemplate, err := builder.Build()
	if err != nil {
		t.Fatalf("Failed to build incompatible template: %v", err)
	}
	
	// Set version to 0.9.0 (should be incompatible)
	incompatibleVersion, err := NewVersionInfo("0.9.0")
	if err != nil {
		t.Fatalf("Failed to create version: %v", err)
	}
	err = manager.SetTemplateVersion("incompatible-base", incompatibleVersion)
	if err != nil {
		t.Fatalf("Failed to set version: %v", err)
	}
	
	// Add dependency on incompatible version
	err = manager.AddDependency("app-template", TemplateDependency{
		Name:            "incompatible-base",
		Version:         "1.0.0",
		VersionOperator: ">=",
	})
	
	// This should fail validation
	err = manager.ValidateTemplateDependencies("app-template", appTemplate.Dependencies)
	if err == nil {
		t.Error("Expected validation to fail for incompatible version, but it passed")
	}
}

// TestTemplateVersioningWithRegistry tests template versioning with the registry
func TestTemplateVersioningWithRegistry(t *testing.T) {
	// Skip in short mode
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}
	
	// Skip if running in CI without AWS credentials
	if os.Getenv("CI") == "true" && os.Getenv("AWS_ACCESS_KEY_ID") == "" {
		t.Skip("Skipping registry test in CI without AWS credentials")
	}

	// Create a mock SSM client for testing
	mockSSM := &MockSSMClient{
		Parameters: make(map[string]string),
		Tags:       make(map[string]map[string]string),
	}

	// Create registry with mock client
	registry := NewRegistry(mockSSM, "/test/registry")

	// Create test base AMIs
	baseAMIs := map[string]map[string]string{
		"us-east-1": {
			"ubuntu-22.04-server-lts": "ami-12345678",
		},
	}

	// Create parser
	parser := NewParser(baseAMIs)

	// Create mock clock for deterministic testing
	mockClock := &MockClock{
		CurrentTime: time.Date(2023, 7, 1, 12, 0, 0, 0, time.UTC),
	}

	// Create template manager with registry and mock clock
	manager := NewTemplateManager(parser, registry, "")
	// Inject mock clock
	manager.clock = mockClock

	// Create test template
	template := createTestTemplate(t, manager, "registry-test", "1.0.0")

	// Share template to registry
	ctx := context.Background()
	err := manager.ShareTemplate("registry-test", ctx)
	if err != nil {
		t.Fatalf("Failed to share template: %v", err)
	}

	// Create a new version
	builder, err := manager.CreateTemplateVersion("registry-test", "minor")
	if err != nil {
		t.Fatalf("Failed to create template version: %v", err)
	}

	// Modify template
	builder.AddBuildStep("new-step", "echo 'New step'")
	
	// Build the template
	_, err = builder.Build()
	if err != nil {
		t.Fatalf("Failed to build new version: %v", err)
	}

	// Share new version
	err = manager.ShareTemplate("registry-test", ctx)
	if err != nil {
		t.Fatalf("Failed to share new version: %v", err)
	}

	// List versions
	versions, err := registry.ListSharedTemplateVersions(ctx, "registry-test")
	if err != nil {
		t.Fatalf("Failed to list versions: %v", err)
	}

	// Should have two versions
	if len(versions) != 2 {
		t.Errorf("Expected 2 versions, got %d", len(versions))
	}

	// Versions should include 1.0.0 and 1.1.0
	hasV1 := false
	hasV11 := false
	for _, v := range versions {
		if v == "1.0.0" {
			hasV1 = true
		}
		if v == "1.1.0" {
			hasV11 = true
		}
	}
	if !hasV1 || !hasV11 {
		t.Errorf("Expected versions 1.0.0 and 1.1.0, got %v", versions)
	}

	// Retrieve specific version
	entry, err := registry.GetSharedTemplate(ctx, "registry-test", "1.0.0")
	if err != nil {
		t.Fatalf("Failed to get shared template: %v", err)
	}
	if entry.Version != "1.0.0" {
		t.Errorf("Expected version 1.0.0, got %s", entry.Version)
	}

	// Retrieve latest version
	latest, err := registry.GetSharedTemplate(ctx, "registry-test", "")
	if err != nil {
		t.Fatalf("Failed to get latest template: %v", err)
	}
	if latest.Version != "1.1.0" {
		t.Errorf("Expected latest version 1.1.0, got %s", latest.Version)
	}
}

// Helper function to create a test template
func createTestTemplate(t *testing.T, manager *TemplateManager, name, initialVersion string) *Template {
	// Create builder
	builder := manager.CreateTemplate(name, "Test Template").
		WithBase("ubuntu-22.04-server-lts").
		AddBuildStep("update", "apt-get update -y")

	// Build the template
	template, err := builder.Build()
	if err != nil {
		t.Fatalf("Failed to build template: %v", err)
	}

	// Set initial version if provided
	if initialVersion != "" {
		version, err := NewVersionInfo(initialVersion)
		if err != nil {
			t.Fatalf("Failed to create version: %v", err)
		}
		err = manager.SetTemplateVersion(name, version)
		if err != nil {
			t.Fatalf("Failed to set version: %v", err)
		}
	}

	return template
}