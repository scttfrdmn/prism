package repository

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
)

// TestNewManager tests creation of a new repository manager.
func TestNewManager(t *testing.T) {
	// Create temporary directory for config and cache
	tempDir, err := ioutil.TempDir("", "repo-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Set up test environment
	setupTestEnvironment(t, tempDir)

	// Create manager with custom paths
	manager, err := createTestManager(tempDir)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}

	// Check that default repository exists
	repos := manager.GetRepositories()
	if len(repos) != 1 {
		t.Fatalf("Expected 1 repository, got %d", len(repos))
	}

	if repos[0].Name != "default" {
		t.Errorf("Expected repository name 'default', got %q", repos[0].Name)
	}

	if repos[0].Type != "github" {
		t.Errorf("Expected repository type 'github', got %q", repos[0].Type)
	}

	if repos[0].URL != DefaultRepositoryURL {
		t.Errorf("Expected repository URL %q, got %q", DefaultRepositoryURL, repos[0].URL)
	}
}

// TestAddRepository tests adding a new repository.
func TestAddRepository(t *testing.T) {
	// Create temporary directory for config and cache
	tempDir, err := ioutil.TempDir("", "repo-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Set up test environment
	setupTestEnvironment(t, tempDir)

	// Create manager with custom paths
	manager, err := createTestManager(tempDir)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}

	// Add a new repository
	repo := Repository{
		Name:     "test",
		Type:     "github",
		URL:      "github.com/test/repo",
		Branch:   "main",
		Priority: 10,
	}

	if err := manager.AddRepository(repo); err != nil {
		t.Fatalf("Failed to add repository: %v", err)
	}

	// Check that repository was added
	repos := manager.GetRepositories()
	if len(repos) != 2 {
		t.Fatalf("Expected 2 repositories, got %d", len(repos))
	}

	// Check that repositories are sorted by priority
	if repos[0].Name != "test" {
		t.Errorf("Expected first repository to be 'test', got %q", repos[0].Name)
	}

	if repos[0].Priority != 10 {
		t.Errorf("Expected repository priority 10, got %d", repos[0].Priority)
	}

	// Try to add a repository with the same name
	repo2 := Repository{
		Name:     "test",
		Type:     "github",
		URL:      "github.com/test/repo2",
		Branch:   "main",
		Priority: 20,
	}

	if err := manager.AddRepository(repo2); err == nil {
		t.Fatal("Expected error when adding repository with same name, got nil")
	}
}

// TestRemoveRepository tests removing a repository.
func TestRemoveRepository(t *testing.T) {
	// Create temporary directory for config and cache
	tempDir, err := ioutil.TempDir("", "repo-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Set up test environment
	setupTestEnvironment(t, tempDir)

	// Create manager with custom paths
	manager, err := createTestManager(tempDir)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}

	// Add a new repository
	repo := Repository{
		Name:     "test",
		Type:     "github",
		URL:      "github.com/test/repo",
		Branch:   "main",
		Priority: 10,
	}

	if err := manager.AddRepository(repo); err != nil {
		t.Fatalf("Failed to add repository: %v", err)
	}

	// Remove the repository
	if err := manager.RemoveRepository("test"); err != nil {
		t.Fatalf("Failed to remove repository: %v", err)
	}

	// Check that repository was removed
	repos := manager.GetRepositories()
	if len(repos) != 1 {
		t.Fatalf("Expected 1 repository, got %d", len(repos))
	}

	// Try to remove the default repository
	if err := manager.RemoveRepository("default"); err == nil {
		t.Fatal("Expected error when removing default repository, got nil")
	}

	// Try to remove a non-existent repository
	if err := manager.RemoveRepository("non-existent"); err == nil {
		t.Fatal("Expected error when removing non-existent repository, got nil")
	}
}

// TestUpdateRepository tests updating a repository.
func TestUpdateRepository(t *testing.T) {
	// Create temporary directory for config and cache
	tempDir, err := ioutil.TempDir("", "repo-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Set up test environment
	setupTestEnvironment(t, tempDir)

	// Create manager with custom paths
	manager, err := createTestManager(tempDir)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}

	// Add a new repository
	repo := Repository{
		Name:     "test",
		Type:     "github",
		URL:      "github.com/test/repo",
		Branch:   "main",
		Priority: 10,
	}

	if err := manager.AddRepository(repo); err != nil {
		t.Fatalf("Failed to add repository: %v", err)
	}

	// Update the repository
	repo.URL = "github.com/test/repo2"
	repo.Priority = 20

	if err := manager.UpdateRepository(repo); err != nil {
		t.Fatalf("Failed to update repository: %v", err)
	}

	// Check that repository was updated
	updatedRepo, err := manager.GetRepository("test")
	if err != nil {
		t.Fatalf("Failed to get repository: %v", err)
	}

	if updatedRepo.URL != "github.com/test/repo2" {
		t.Errorf("Expected repository URL 'github.com/test/repo2', got %q", updatedRepo.URL)
	}

	if updatedRepo.Priority != 20 {
		t.Errorf("Expected repository priority 20, got %d", updatedRepo.Priority)
	}

	// Try to update a non-existent repository
	nonExistentRepo := Repository{
		Name:     "non-existent",
		Type:     "github",
		URL:      "github.com/test/repo",
		Branch:   "main",
		Priority: 10,
	}

	if err := manager.UpdateRepository(nonExistentRepo); err == nil {
		t.Fatal("Expected error when updating non-existent repository, got nil")
	}
}

// TestParseTemplateReference tests parsing template references.
func TestParseTemplateReference(t *testing.T) {
	// Create temporary directory for config and cache
	tempDir, err := ioutil.TempDir("", "repo-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Set up test environment
	setupTestEnvironment(t, tempDir)

	// Create manager with custom paths
	manager, err := createTestManager(tempDir)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}

	// Test cases
	testCases := []struct {
		input         string
		expectedRepo  string
		expectedTemp  string
		expectedVer   string
		expectedError bool
	}{
		{
			input:        "template",
			expectedTemp: "template",
		},
		{
			input:        "repo:template",
			expectedRepo: "repo",
			expectedTemp: "template",
		},
		{
			input:        "template@1.0.0",
			expectedTemp: "template",
			expectedVer:  "1.0.0",
		},
		{
			input:        "repo:template@1.0.0",
			expectedRepo: "repo",
			expectedTemp: "template",
			expectedVer:  "1.0.0",
		},
		{
			input:         "",
			expectedError: true,
		},
		{
			input:         "repo:",
			expectedError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.input, func(t *testing.T) {
			ref, err := manager.ParseTemplateReference(tc.input)

			// Check error
			if tc.expectedError && err == nil {
				t.Fatal("Expected error, got nil")
			} else if !tc.expectedError && err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			// If error expected, no need to check other fields
			if tc.expectedError {
				return
			}

			// Check fields
			if ref.Repository != tc.expectedRepo {
				t.Errorf("Expected repository %q, got %q", tc.expectedRepo, ref.Repository)
			}

			if ref.Template != tc.expectedTemp {
				t.Errorf("Expected template %q, got %q", tc.expectedTemp, ref.Template)
			}

			if ref.Version != tc.expectedVer {
				t.Errorf("Expected version %q, got %q", tc.expectedVer, ref.Version)
			}
		})
	}
}

// setupTestEnvironment sets up a test environment with custom paths.
func setupTestEnvironment(t *testing.T, tempDir string) {
	// Create config directory
	configDir := filepath.Join(tempDir, ConfigDirName)
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatalf("Failed to create config directory: %v", err)
	}

	// Create cache directory
	cacheDir := filepath.Join(configDir, CacheDirName)
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		t.Fatalf("Failed to create cache directory: %v", err)
	}
}

// createTestManager creates a new manager with custom paths.
func createTestManager(tempDir string) (*Manager, error) {
	configDir := filepath.Join(tempDir, ConfigDirName)
	cacheDir := filepath.Join(configDir, CacheDirName)
	configPath := filepath.Join(configDir, ConfigFileName)
	cacheFilePath := filepath.Join(cacheDir, CacheFileName)

	manager := &Manager{
		configPath:    configPath,
		cachePath:     cacheDir,
		cacheFilePath: cacheFilePath,
		config:        &Config{},
		cache:         &RepositoryCache{Repositories: make(map[string]RepositoryCacheEntry)},
	}

	// Ensure default repository exists
	if err := manager.ensureDefaultRepository(); err != nil {
		return nil, err
	}

	return manager, nil
}