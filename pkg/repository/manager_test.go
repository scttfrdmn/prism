package repository

import (
	"os"
	"path/filepath"
	"testing"
	
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNewManager tests creating a new repository manager
func TestNewManager(t *testing.T) {
	// Create temporary directory for testing
	tempDir, err := os.MkdirTemp("", "repo-manager-test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)
	
	// Create manager with test directory
	manager, err := NewManager(WithBasePath(tempDir))
	require.NoError(t, err)
	assert.NotNil(t, manager)
	
	// Should have no repositories initially
	repos, err := manager.ListRepositories()
	require.NoError(t, err)
	assert.Empty(t, repos)
}

// TestAddRepository tests adding a repository
func TestAddRepository(t *testing.T) {
	// Create temporary directory for testing
	tempDir, err := os.MkdirTemp("", "repo-manager-test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)
	
	// Create manager with test directory
	manager, err := NewManager(WithBasePath(tempDir))
	require.NoError(t, err)
	
	// Add a repository
	repo := Repository{
		Name:     "test-repo",
		URL:      "https://github.com/test/repo",
		Type:     "github",
		Priority: 1,
	}
	
	err = manager.AddRepository(repo)
	require.NoError(t, err)
	
	// Check that repository was added
	repos, err := manager.ListRepositories()
	require.NoError(t, err)
	assert.Len(t, repos, 1)
	assert.Equal(t, "test-repo", repos[0].Name)
	assert.Equal(t, "https://github.com/test/repo", repos[0].URL)
	assert.Equal(t, "github", repos[0].Type)
	assert.Equal(t, 1, repos[0].Priority)
	
	// Add another repository
	repo2 := Repository{
		Name:     "test-repo2",
		URL:      "https://github.com/test/repo2",
		Type:     "github",
		Priority: 2,
	}
	
	err = manager.AddRepository(repo2)
	require.NoError(t, err)
	
	// Check that both repositories are present
	repos, err = manager.ListRepositories()
	require.NoError(t, err)
	assert.Len(t, repos, 2)
	
	// Repositories should be sorted by priority (highest first)
	assert.Equal(t, "test-repo2", repos[0].Name)
	assert.Equal(t, "test-repo", repos[1].Name)
}

// TestRemoveRepository tests removing a repository
func TestRemoveRepository(t *testing.T) {
	// Create temporary directory for testing
	tempDir, err := os.MkdirTemp("", "repo-manager-test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)
	
	// Create manager with test directory
	manager, err := NewManager(WithBasePath(tempDir))
	require.NoError(t, err)
	
	// Add repositories
	repo1 := Repository{
		Name:     "test-repo1",
		URL:      "https://github.com/test/repo1",
		Type:     "github",
		Priority: 1,
	}
	
	repo2 := Repository{
		Name:     "test-repo2",
		URL:      "https://github.com/test/repo2",
		Type:     "github",
		Priority: 2,
	}
	
	err = manager.AddRepository(repo1)
	require.NoError(t, err)
	
	err = manager.AddRepository(repo2)
	require.NoError(t, err)
	
	// Remove first repository
	err = manager.RemoveRepository("test-repo1")
	require.NoError(t, err)
	
	// Check that only second repository remains
	repos, err := manager.ListRepositories()
	require.NoError(t, err)
	assert.Len(t, repos, 1)
	assert.Equal(t, "test-repo2", repos[0].Name)
	
	// Try removing non-existent repository
	err = manager.RemoveRepository("non-existent")
	assert.Error(t, err)
}

// TestGetRepository tests getting a specific repository
func TestGetRepository(t *testing.T) {
	// Create temporary directory for testing
	tempDir, err := os.MkdirTemp("", "repo-manager-test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)
	
	// Create manager with test directory
	manager, err := NewManager(WithBasePath(tempDir))
	require.NoError(t, err)
	
	// Add repositories
	repo1 := Repository{
		Name:     "test-repo1",
		URL:      "https://github.com/test/repo1",
		Type:     "github",
		Priority: 1,
	}
	
	repo2 := Repository{
		Name:     "test-repo2",
		URL:      "https://github.com/test/repo2",
		Type:     "github",
		Priority: 2,
	}
	
	err = manager.AddRepository(repo1)
	require.NoError(t, err)
	
	err = manager.AddRepository(repo2)
	require.NoError(t, err)
	
	// Get specific repository
	repo, err := manager.GetRepository("test-repo1")
	require.NoError(t, err)
	assert.Equal(t, "test-repo1", repo.Name)
	assert.Equal(t, "https://github.com/test/repo1", repo.URL)
	
	// Try getting non-existent repository
	_, err = manager.GetRepository("non-existent")
	assert.Error(t, err)
}

// TestUpdateRepository tests updating a repository
func TestUpdateRepository(t *testing.T) {
	// Create temporary directory for testing
	tempDir, err := os.MkdirTemp("", "repo-manager-test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)
	
	// Create manager with test directory
	manager, err := NewManager(WithBasePath(tempDir))
	require.NoError(t, err)
	
	// Add repository
	repo := Repository{
		Name:     "test-repo",
		URL:      "https://github.com/test/repo",
		Type:     "github",
		Priority: 1,
	}
	
	err = manager.AddRepository(repo)
	require.NoError(t, err)
	
	// Create local repository directory
	repoDir := filepath.Join(tempDir, "repositories", "test-repo")
	err = os.MkdirAll(repoDir, 0755)
	require.NoError(t, err)
	
	// Create a dummy file to simulate repository content
	dummyFile := filepath.Join(repoDir, "dummy.txt")
	err = os.WriteFile(dummyFile, []byte("dummy content"), 0644)
	require.NoError(t, err)
	
	// Mock repo update - in a real test this would actually clone/pull from git
	err = manager.UpdateRepository("test-repo")
	require.NoError(t, err)
	
	// Try updating non-existent repository
	err = manager.UpdateRepository("non-existent")
	assert.Error(t, err)
}

// TestTemplateResolution tests template resolution
func TestTemplateResolution(t *testing.T) {
	// Create temporary directory for testing
	tempDir, err := os.MkdirTemp("", "repo-manager-test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)
	
	// Create manager with test directory
	manager, err := NewManager(WithBasePath(tempDir))
	require.NoError(t, err)
	
	// Add repositories
	repo1 := Repository{
		Name:     "test-repo1",
		URL:      "https://github.com/test/repo1",
		Type:     "github",
		Priority: 1,
	}
	
	repo2 := Repository{
		Name:     "test-repo2",
		URL:      "https://github.com/test/repo2",
		Type:     "github",
		Priority: 2,
	}
	
	err = manager.AddRepository(repo1)
	require.NoError(t, err)
	
	err = manager.AddRepository(repo2)
	require.NoError(t, err)
	
	// Create repository directories
	repo1Dir := filepath.Join(tempDir, "repositories", "test-repo1")
	repo2Dir := filepath.Join(tempDir, "repositories", "test-repo2")
	
	err = os.MkdirAll(filepath.Join(repo1Dir, "domains", "test"), 0755)
	require.NoError(t, err)
	
	err = os.MkdirAll(filepath.Join(repo2Dir, "domains", "test"), 0755)
	require.NoError(t, err)
	
	// Create template files
	template1Path := filepath.Join(repo1Dir, "domains", "test", "template1.yaml")
	template2Path := filepath.Join(repo2Dir, "domains", "test", "template1.yaml")
	
	// Template in repo1 with same name but lower priority
	err = os.WriteFile(template1Path, []byte("name: template1\nversion: 1.0.0"), 0644)
	require.NoError(t, err)
	
	// Template in repo2 with same name but higher priority
	err = os.WriteFile(template2Path, []byte("name: template1\nversion: 2.0.0"), 0644)
	require.NoError(t, err)
	
	// Test repository-specified template resolution
	templatePath, err := manager.ResolveTemplate("test-repo1:template1", "")
	require.NoError(t, err)
	assert.Equal(t, template1Path, templatePath)
	
	templatePath, err = manager.ResolveTemplate("test-repo2:template1", "")
	require.NoError(t, err)
	assert.Equal(t, template2Path, templatePath)
	
	// Test priority-based resolution (should use repo2 version as it has higher priority)
	templatePath, err = manager.ResolveTemplate("template1", "")
	require.NoError(t, err)
	assert.Equal(t, template2Path, templatePath)
}

// TestDependencyResolution tests dependency resolution
func TestDependencyResolution(t *testing.T) {
	// Create temporary directory for testing
	tempDir, err := os.MkdirTemp("", "repo-manager-test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)
	
	// Create manager with test directory
	manager, err := NewManager(WithBasePath(tempDir))
	require.NoError(t, err)
	
	// Add repository
	repo := Repository{
		Name:     "test-repo",
		URL:      "https://github.com/test/repo",
		Type:     "github",
		Priority: 1,
	}
	
	err = manager.AddRepository(repo)
	require.NoError(t, err)
	
	// Create repository directory structure
	repoDir := filepath.Join(tempDir, "repositories", "test-repo")
	baseDir := filepath.Join(repoDir, "base")
	stacksDir := filepath.Join(repoDir, "stacks")
	
	err = os.MkdirAll(baseDir, 0755)
	require.NoError(t, err)
	
	err = os.MkdirAll(stacksDir, 0755)
	require.NoError(t, err)
	
	// Create template files
	baseTemplate := filepath.Join(baseDir, "ubuntu.yaml")
	stackTemplate := filepath.Join(stacksDir, "python.yaml")
	appTemplate := filepath.Join(stacksDir, "django.yaml")
	
	// Base template with no dependencies
	err = os.WriteFile(baseTemplate, []byte("name: ubuntu\nversion: 1.0.0"), 0644)
	require.NoError(t, err)
	
	// Stack template that depends on base
	pythonTemplateContent := `name: python
version: 1.0.0
dependencies:
  - name: ubuntu
    version: 1.0.0
`
	err = os.WriteFile(stackTemplate, []byte(pythonTemplateContent), 0644)
	require.NoError(t, err)
	
	// App template that depends on python stack
	djangoTemplateContent := `name: django
version: 1.0.0
dependencies:
  - name: python
    version: 1.0.0
`
	err = os.WriteFile(appTemplate, []byte(djangoTemplateContent), 0644)
	require.NoError(t, err)
	
	// Resolve dependencies for django template
	deps, err := manager.ResolveDependencies("test-repo:django", "")
	require.NoError(t, err)
	assert.Len(t, deps, 3) // Should include ubuntu, python, and django itself
	
	// Dependencies should be ordered correctly - base first, then dependencies
	assert.Equal(t, "ubuntu", deps[0].Name)
	assert.Equal(t, "python", deps[1].Name)
	assert.Equal(t, "django", deps[2].Name)
}