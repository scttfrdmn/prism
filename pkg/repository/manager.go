package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"gopkg.in/yaml.v3"
)

const (
	// DefaultRepositoryURL is the URL of the default CloudWorkstation repository.
	DefaultRepositoryURL = "github.com/scttfrdmn/cloudworkstation-repository"

	// DefaultRepositoryBranch is the branch to use for the default repository.
	DefaultRepositoryBranch = "main"

	// CacheTTL is the time-to-live for the repository cache in hours.
	CacheTTL = 24

	// ConfigDirName is the name of the CloudWorkstation configuration directory.
	ConfigDirName = ".cloudworkstation"

	// ConfigFileName is the name of the configuration file.
	ConfigFileName = "config.json"

	// CacheDirName is the name of the cache directory.
	CacheDirName = "repositories"

	// CacheFileName is the name of the cache metadata file.
	CacheFileName = "cache.json"

	// RepositoryFileName is the name of the repository metadata file.
	RepositoryFileName = "repository.yaml"
)

// Manager handles repository operations.
type Manager struct {
	// configPath is the path to the configuration file
	configPath string

	// cachePath is the path to the cache directory
	cachePath string

	// cacheFilePath is the path to the cache metadata file
	cacheFilePath string

	// config contains the repository configuration
	config *Config

	// cache contains the repository cache
	cache *RepositoryCache
}

// NewManager creates a new repository manager.
func NewManager() (*Manager, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get user home directory: %w", err)
	}

	configDir := filepath.Join(homeDir, ConfigDirName)
	if err := ensureDir(configDir); err != nil {
		return nil, fmt.Errorf("failed to create config directory: %w", err)
	}

	cachePath := filepath.Join(configDir, CacheDirName)
	if err := ensureDir(cachePath); err != nil {
		return nil, fmt.Errorf("failed to create cache directory: %w", err)
	}

	configPath := filepath.Join(configDir, ConfigFileName)
	cacheFilePath := filepath.Join(cachePath, CacheFileName)

	manager := &Manager{
		configPath:    configPath,
		cachePath:     cachePath,
		cacheFilePath: cacheFilePath,
		config:        &Config{},
		cache:         &RepositoryCache{Repositories: make(map[string]RepositoryCacheEntry)},
	}

	// Load existing configuration or create default
	if err := manager.loadConfig(); err != nil {
		return nil, fmt.Errorf("failed to load repository configuration: %w", err)
	}

	// Ensure default repository exists
	if err := manager.ensureDefaultRepository(); err != nil {
		return nil, fmt.Errorf("failed to ensure default repository: %w", err)
	}

	// Load cache if it exists
	if err := manager.loadCache(); err != nil {
		return nil, fmt.Errorf("failed to load repository cache: %w", err)
	}

	return manager, nil
}

// loadConfig loads the repository configuration from disk.
func (m *Manager) loadConfig() error {
	// Check if config file exists
	if _, err := os.Stat(m.configPath); os.IsNotExist(err) {
		// Create default configuration
		m.config = &Config{
			Repositories: []Repository{},
		}
		return nil
	}

	// Read config file
	data, err := os.ReadFile(m.configPath)
	if err != nil {
		return fmt.Errorf("failed to read config file: %w", err)
	}

	// Parse JSON
	if err := json.Unmarshal(data, m.config); err != nil {
		return fmt.Errorf("failed to parse config file: %w", err)
	}

	return nil
}

// saveConfig saves the repository configuration to disk.
func (m *Manager) saveConfig() error {
	// Marshal JSON
	data, err := json.MarshalIndent(m.config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	// Write config file
	if err := os.WriteFile(m.configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// loadCache loads the repository cache from disk.
func (m *Manager) loadCache() error {
	// Check if cache file exists
	if _, err := os.Stat(m.cacheFilePath); os.IsNotExist(err) {
		// Create empty cache
		m.cache = &RepositoryCache{
			LastUpdated:  time.Now(),
			Repositories: make(map[string]RepositoryCacheEntry),
		}
		return nil
	}

	// Read cache file
	data, err := os.ReadFile(m.cacheFilePath)
	if err != nil {
		return fmt.Errorf("failed to read cache file: %w", err)
	}

	// Parse JSON
	if err := json.Unmarshal(data, m.cache); err != nil {
		return fmt.Errorf("failed to parse cache file: %w", err)
	}

	return nil
}

// saveCache saves the repository cache to disk.
func (m *Manager) saveCache() error {
	// Marshal JSON
	data, err := json.MarshalIndent(m.cache, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal cache: %w", err)
	}

	// Write cache file
	if err := os.WriteFile(m.cacheFilePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write cache file: %w", err)
	}

	return nil
}

// ensureDefaultRepository ensures the default repository is configured.
func (m *Manager) ensureDefaultRepository() error {
	// Check if default repository exists
	for _, repo := range m.config.Repositories {
		if repo.Name == "default" {
			return nil
		}
	}

	// Add default repository
	defaultRepo := Repository{
		Name:     "default",
		Type:     "github",
		URL:      DefaultRepositoryURL,
		Branch:   DefaultRepositoryBranch,
		Priority: 1, // Lowest priority
	}

	m.config.Repositories = append(m.config.Repositories, defaultRepo)
	return m.saveConfig()
}

// GetRepositories returns the list of configured repositories sorted by priority.
func (m *Manager) GetRepositories() []Repository {
	repos := make([]Repository, len(m.config.Repositories))
	copy(repos, m.config.Repositories)

	// Sort by priority (highest to lowest)
	sort.Slice(repos, func(i, j int) bool {
		return repos[i].Priority > repos[j].Priority
	})

	return repos
}

// GetRepository returns the repository with the given name.
func (m *Manager) GetRepository(name string) (*Repository, error) {
	for _, repo := range m.config.Repositories {
		if repo.Name == name {
			return &repo, nil
		}
	}

	return nil, fmt.Errorf("repository %q not found", name)
}

// AddRepository adds a new repository to the configuration.
func (m *Manager) AddRepository(repo Repository) error {
	// Check if repository already exists
	for _, r := range m.config.Repositories {
		if r.Name == repo.Name {
			return fmt.Errorf("repository %q already exists", repo.Name)
		}
	}

	// Add repository
	m.config.Repositories = append(m.config.Repositories, repo)
	return m.saveConfig()
}

// RemoveRepository removes a repository from the configuration.
func (m *Manager) RemoveRepository(name string) error {
	// Check if it's the default repository
	if name == "default" {
		return fmt.Errorf("cannot remove the default repository")
	}

	// Find repository
	index := -1
	for i, repo := range m.config.Repositories {
		if repo.Name == name {
			index = i
			break
		}
	}

	if index == -1 {
		return fmt.Errorf("repository %q not found", name)
	}

	// Remove repository from configuration
	m.config.Repositories = append(m.config.Repositories[:index], m.config.Repositories[index+1:]...)

	// Remove from cache if exists
	delete(m.cache.Repositories, name)

	// Save config and cache
	if err := m.saveConfig(); err != nil {
		return err
	}

	return m.saveCache()
}

// UpdateRepository updates an existing repository in the configuration.
func (m *Manager) UpdateRepository(repo Repository) error {
	// Find repository
	index := -1
	for i, r := range m.config.Repositories {
		if r.Name == repo.Name {
			index = i
			break
		}
	}

	if index == -1 {
		return fmt.Errorf("repository %q not found", repo.Name)
	}

	// Update repository
	m.config.Repositories[index] = repo
	return m.saveConfig()
}

// ParseTemplateReference parses a template reference string into a TemplateReference.
// Format: [repo:]template[@version]
func (m *Manager) ParseTemplateReference(ref string) (TemplateReference, error) {
	result := TemplateReference{}

	// Check for repository prefix
	if parts := strings.SplitN(ref, ":", 2); len(parts) > 1 {
		result.Repository = parts[0]
		ref = parts[1]
	}

	// Check for version suffix
	if parts := strings.SplitN(ref, "@", 2); len(parts) > 1 {
		result.Template = parts[0]
		result.Version = parts[1]
	} else {
		result.Template = ref
	}

	// Validate
	if result.Template == "" {
		return result, fmt.Errorf("invalid template reference: template name cannot be empty")
	}

	return result, nil
}

// FindTemplate locates a template across repositories.
func (m *Manager) FindTemplate(ref TemplateReference) (*TemplateMetadata, *Repository, error) {
	// If repository is specified, only look in that repository
	if ref.Repository != "" {
		repo, err := m.GetRepository(ref.Repository)
		if err != nil {
			return nil, nil, err
		}

		// Ensure repository is cached
		if err := m.UpdateRepositoryCache(repo); err != nil {
			return nil, nil, err
		}

		// Get repository metadata
		metadata, err := m.GetRepositoryMetadata(repo.Name)
		if err != nil {
			return nil, nil, err
		}

		// Find template
		for _, t := range metadata.Templates {
			if t.Name == ref.Template {
				return &t, repo, nil
			}
		}

		return nil, nil, fmt.Errorf("template %q not found in repository %q", ref.Template, ref.Repository)
	}

	// Look in all repositories by priority
	repos := m.GetRepositories()
	for _, repo := range repos {
		// Ensure repository is cached
		if err := m.UpdateRepositoryCache(&repo); err != nil {
			// Just log the error and continue
			fmt.Fprintf(os.Stderr, "Warning: failed to update cache for repository %q: %v\n", repo.Name, err)
			continue
		}

		// Get repository metadata
		metadata, err := m.GetRepositoryMetadata(repo.Name)
		if err != nil {
			// Just log the error and continue
			fmt.Fprintf(os.Stderr, "Warning: failed to get metadata for repository %q: %v\n", repo.Name, err)
			continue
		}

		// Find template
		for _, t := range metadata.Templates {
			if t.Name == ref.Template {
				return &t, &repo, nil
			}
		}
	}

	return nil, nil, fmt.Errorf("template %q not found in any repository", ref.Template)
}

// getRepositoryMetadata retrieves the metadata for a repository from cache.
func (m *Manager) GetRepositoryMetadata(name string) (*RepositoryMetadata, error) {
	entry, ok := m.cache.Repositories[name]
	if !ok {
		return nil, fmt.Errorf("repository %q not found in cache", name)
	}

	if entry.Metadata == nil {
		return nil, fmt.Errorf("metadata for repository %q not available", name)
	}

	return entry.Metadata, nil
}

// updateRepositoryCache ensures a repository is cached and up-to-date.
func (m *Manager) UpdateRepositoryCache(repo *Repository) error {
	// Check if repository is in cache and up-to-date
	entry, ok := m.cache.Repositories[repo.Name]
	if ok {
		// Check if cache is still valid
		if time.Since(entry.LastUpdated).Hours() < CacheTTL {
			return nil
		}
	}

	// Update cache
	switch repo.Type {
	case "github":
		return m.updateGitHubCache(repo)
	case "local":
		return m.updateLocalCache(repo)
	case "s3":
		return m.updateS3Cache(repo)
	default:
		return fmt.Errorf("unsupported repository type: %s", repo.Type)
	}
}

// updateGitHubCache updates the cache for a GitHub repository.
func (m *Manager) updateGitHubCache(repo *Repository) error {
	// Parse GitHub URL to extract owner/repo
	// Expected format: https://github.com/owner/repo or github.com/owner/repo
	urlParts := strings.Split(strings.TrimPrefix(repo.URL, "https://"), "/")
	if len(urlParts) < 3 {
		urlParts = strings.Split(repo.URL, "/")
	}
	if len(urlParts) < 2 {
		return fmt.Errorf("invalid GitHub URL format: %s", repo.URL)
	}

	owner := urlParts[len(urlParts)-2]
	repoName := urlParts[len(urlParts)-1]
	branch := repo.Branch
	if branch == "" {
		branch = "main"
	}

	// Construct raw GitHub URL for repository.yaml
	// https://raw.githubusercontent.com/owner/repo/branch/repository.yaml
	rawURL := fmt.Sprintf("https://raw.githubusercontent.com/%s/%s/%s/%s",
		owner, repoName, branch, RepositoryFileName)

	// Fetch repository.yaml via HTTP
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", rawURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create HTTP request: %w", err)
	}

	// Add authentication if GitHub token is available
	if token := os.Getenv("GITHUB_TOKEN"); token != "" {
		req.Header.Set("Authorization", "token "+token)
	}

	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to fetch repository.yaml from GitHub: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to fetch repository.yaml: HTTP %d", resp.StatusCode)
	}

	// Read response body
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}

	// Parse YAML
	metadata := &RepositoryMetadata{}
	if err := yaml.Unmarshal(data, metadata); err != nil {
		return fmt.Errorf("failed to parse repository.yaml: %w", err)
	}

	// Create cache directory
	cachePath := filepath.Join(m.cachePath, repo.Name)
	if err := ensureDir(cachePath); err != nil {
		return fmt.Errorf("failed to create cache directory: %w", err)
	}

	// Update cache entry
	m.cache.Repositories[repo.Name] = RepositoryCacheEntry{
		LastUpdated: time.Now(),
		Path:        cachePath,
		Metadata:    metadata,
	}

	return m.saveCache()
}

// updateLocalCache updates the cache for a local repository.
func (m *Manager) updateLocalCache(repo *Repository) error {
	// Validate local path
	if repo.Path == "" {
		return fmt.Errorf("local repository must have a path")
	}

	if _, err := os.Stat(repo.Path); os.IsNotExist(err) {
		return fmt.Errorf("repository path %q does not exist", repo.Path)
	}

	// Read repository.yaml
	repoFilePath := filepath.Join(repo.Path, RepositoryFileName)
	if _, err := os.Stat(repoFilePath); os.IsNotExist(err) {
		return fmt.Errorf("repository.yaml not found in %q", repo.Path)
	}

	data, err := os.ReadFile(repoFilePath)
	if err != nil {
		return fmt.Errorf("failed to read repository.yaml: %w", err)
	}

	// Parse YAML
	metadata := &RepositoryMetadata{}
	if err := yaml.Unmarshal(data, metadata); err != nil {
		return fmt.Errorf("failed to parse repository.yaml: %w", err)
	}

	// Copy to cache
	cachePath := filepath.Join(m.cachePath, repo.Name)
	if err := ensureDir(cachePath); err != nil {
		return fmt.Errorf("failed to create cache directory: %w", err)
	}

	// Update cache entry
	m.cache.Repositories[repo.Name] = RepositoryCacheEntry{
		LastUpdated: time.Now(),
		Path:        cachePath,
		Metadata:    metadata,
	}

	return m.saveCache()
}

// updateS3Cache updates the cache for an S3 repository.
func (m *Manager) updateS3Cache(repo *Repository) error {
	// Parse S3 URL to extract bucket and prefix
	// Expected format: s3://bucket-name/path/to/repository
	s3URL := strings.TrimPrefix(repo.URL, "s3://")
	urlParts := strings.SplitN(s3URL, "/", 2)

	if len(urlParts) < 1 {
		return fmt.Errorf("invalid S3 URL format: %s", repo.URL)
	}

	bucket := urlParts[0]
	prefix := ""
	if len(urlParts) > 1 {
		prefix = urlParts[1]
	}

	// Construct S3 object key for repository.yaml
	objectKey := prefix
	if prefix != "" && !strings.HasSuffix(prefix, "/") {
		objectKey += "/"
	}
	objectKey += RepositoryFileName

	// Create AWS S3 client
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Load AWS configuration with support for AWS_PROFILE environment variable
	cfg, err := config.LoadDefaultConfig(ctx,
		config.WithRegion(os.Getenv("AWS_REGION")),
	)
	if err != nil {
		return fmt.Errorf("failed to load AWS config: %w", err)
	}

	// Create S3 client
	s3Client := s3.NewFromConfig(cfg)

	// Fetch repository.yaml from S3
	getObjectInput := &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(objectKey),
	}

	result, err := s3Client.GetObject(ctx, getObjectInput)
	if err != nil {
		return fmt.Errorf("failed to fetch repository.yaml from S3 (s3://%s/%s): %w", bucket, objectKey, err)
	}
	defer result.Body.Close()

	// Read object content
	data, err := io.ReadAll(result.Body)
	if err != nil {
		return fmt.Errorf("failed to read S3 object body: %w", err)
	}

	// Parse YAML
	metadata := &RepositoryMetadata{}
	if err := yaml.Unmarshal(data, metadata); err != nil {
		return fmt.Errorf("failed to parse repository.yaml: %w", err)
	}

	// Create cache directory
	cachePath := filepath.Join(m.cachePath, repo.Name)
	if err := ensureDir(cachePath); err != nil {
		return fmt.Errorf("failed to create cache directory: %w", err)
	}

	// Update cache entry
	m.cache.Repositories[repo.Name] = RepositoryCacheEntry{
		LastUpdated: time.Now(),
		Path:        cachePath,
		Metadata:    metadata,
	}

	return m.saveCache()
}

// ensureDir ensures a directory exists, creating it if necessary.
func ensureDir(path string) error {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return os.MkdirAll(path, 0755)
	}
	return nil
}

// DownloadTemplate downloads a template from a repository to the local templates directory
func (m *Manager) DownloadTemplate(ref TemplateReference, destDir string) (string, error) {
	// Find the template in repositories
	templateMeta, repo, err := m.FindTemplate(ref)
	if err != nil {
		return "", fmt.Errorf("failed to find template: %w", err)
	}

	// Ensure destination directory exists
	if err := ensureDir(destDir); err != nil {
		return "", fmt.Errorf("failed to create destination directory: %w", err)
	}

	// Download based on repository type
	var templateContent []byte
	switch repo.Type {
	case "local":
		templateContent, err = m.downloadFromLocal(repo, templateMeta.Path)
	case "github":
		templateContent, err = m.downloadFromGitHub(repo, templateMeta.Path)
	case "s3":
		templateContent, err = m.downloadFromS3(repo, templateMeta.Path)
	default:
		return "", fmt.Errorf("unsupported repository type: %s", repo.Type)
	}

	if err != nil {
		return "", fmt.Errorf("failed to download template: %w", err)
	}

	// Write template to destination
	destFile := filepath.Join(destDir, filepath.Base(templateMeta.Path))
	if err := os.WriteFile(destFile, templateContent, 0644); err != nil {
		return "", fmt.Errorf("failed to write template file: %w", err)
	}

	return destFile, nil
}

// downloadFromLocal reads a template from a local repository
func (m *Manager) downloadFromLocal(repo *Repository, templatePath string) ([]byte, error) {
	fullPath := filepath.Join(repo.Path, templatePath)
	content, err := os.ReadFile(fullPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read local template: %w", err)
	}
	return content, nil
}

// downloadFromGitHub downloads a template from a GitHub repository
func (m *Manager) downloadFromGitHub(repo *Repository, templatePath string) ([]byte, error) {
	// Parse GitHub URL to get owner and repo
	// Expected format: github.com/owner/repo
	parts := strings.Split(strings.TrimPrefix(repo.URL, "https://"), "/")
	if len(parts) < 3 {
		parts = strings.Split(repo.URL, "/")
	}
	if len(parts) < 2 {
		return nil, fmt.Errorf("invalid GitHub URL format: %s", repo.URL)
	}

	owner := parts[len(parts)-2]
	repoName := parts[len(parts)-1]
	branch := repo.Branch
	if branch == "" {
		branch = "main"
	}

	// Construct raw GitHub URL for template file
	rawURL := fmt.Sprintf("https://raw.githubusercontent.com/%s/%s/%s/%s",
		owner, repoName, branch, templatePath)

	// Fetch template via HTTP
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", rawURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}

	// Add authentication if GitHub token is available
	if token := os.Getenv("GITHUB_TOKEN"); token != "" {
		req.Header.Set("Authorization", "token "+token)
	}

	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch template from GitHub: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch template: HTTP %d", resp.StatusCode)
	}

	// Read response body
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	return data, nil
}

// downloadFromS3 downloads a template from an S3 bucket
func (m *Manager) downloadFromS3(repo *Repository, templatePath string) ([]byte, error) {
	// Parse S3 URL to extract bucket
	s3URL := strings.TrimPrefix(repo.URL, "s3://")
	urlParts := strings.SplitN(s3URL, "/", 2)

	if len(urlParts) < 1 {
		return nil, fmt.Errorf("invalid S3 URL format: %s", repo.URL)
	}

	bucket := urlParts[0]
	prefix := ""
	if len(urlParts) > 1 {
		prefix = urlParts[1]
	}

	// Construct S3 object key
	objectKey := prefix
	if prefix != "" && !strings.HasSuffix(prefix, "/") {
		objectKey += "/"
	}
	objectKey += templatePath

	// Create AWS S3 client
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Load AWS configuration
	cfg, err := config.LoadDefaultConfig(ctx,
		config.WithRegion(os.Getenv("AWS_REGION")),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	// Create S3 client
	s3Client := s3.NewFromConfig(cfg)

	// Fetch template from S3
	getObjectInput := &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(objectKey),
	}

	result, err := s3Client.GetObject(ctx, getObjectInput)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch template from S3 (s3://%s/%s): %w", bucket, objectKey, err)
	}
	defer result.Body.Close()

	// Read object content
	data, err := io.ReadAll(result.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read S3 object body: %w", err)
	}

	return data, nil
}

// UploadTemplate uploads a template to a repository
func (m *Manager) UploadTemplate(templatePath string, ref TemplateReference) error {
	// Get the target repository
	repo, err := m.GetRepository(ref.Repository)
	if err != nil {
		return fmt.Errorf("failed to get repository: %w", err)
	}

	// Read template file
	templateContent, err := os.ReadFile(templatePath)
	if err != nil {
		return fmt.Errorf("failed to read template file: %w", err)
	}

	// Upload based on repository type
	switch repo.Type {
	case "local":
		return m.uploadToLocal(repo, ref.Template, templateContent)
	case "github":
		return m.uploadToGitHub(repo, ref.Template, templateContent)
	case "s3":
		return m.uploadToS3(repo, ref.Template, templateContent)
	default:
		return fmt.Errorf("unsupported repository type: %s", repo.Type)
	}
}

// uploadToLocal writes a template to a local repository
func (m *Manager) uploadToLocal(repo *Repository, templateName string, content []byte) error {
	// Ensure repository directory exists
	if err := ensureDir(repo.Path); err != nil {
		return fmt.Errorf("failed to create repository directory: %w", err)
	}

	// Construct destination path
	destPath := filepath.Join(repo.Path, templateName+".yml")

	// Write template file
	if err := os.WriteFile(destPath, content, 0644); err != nil {
		return fmt.Errorf("failed to write template: %w", err)
	}

	// Update repository cache
	if err := m.UpdateRepositoryCache(repo); err != nil {
		return fmt.Errorf("failed to update repository cache: %w", err)
	}

	return nil
}

// uploadToGitHub uploads a template to a GitHub repository
func (m *Manager) uploadToGitHub(repo *Repository, templateName string, content []byte) error {
	// GitHub uploads require direct push access or pull request workflow
	// This is intentionally not implemented as template uploads to GitHub
	// should go through proper git workflows (clone, commit, push)
	// For automated template sharing, use S3 or local repositories
	return fmt.Errorf("GitHub uploads must use git workflows (clone/commit/push) - use 'git push' or S3/local repositories for automated uploads")
}

// uploadToS3 uploads a template to an S3 bucket
func (m *Manager) uploadToS3(repo *Repository, templateName string, content []byte) error {
	// Parse S3 URL to extract bucket
	s3URL := strings.TrimPrefix(repo.URL, "s3://")
	urlParts := strings.SplitN(s3URL, "/", 2)

	if len(urlParts) < 1 {
		return fmt.Errorf("invalid S3 URL format: %s", repo.URL)
	}

	bucket := urlParts[0]
	prefix := ""
	if len(urlParts) > 1 {
		prefix = urlParts[1]
	}

	// Construct S3 object key
	objectKey := prefix
	if prefix != "" && !strings.HasSuffix(prefix, "/") {
		objectKey += "/"
	}
	objectKey += templateName + ".yml"

	// Create AWS S3 client
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Load AWS configuration
	cfg, err := config.LoadDefaultConfig(ctx,
		config.WithRegion(os.Getenv("AWS_REGION")),
	)
	if err != nil {
		return fmt.Errorf("failed to load AWS config: %w", err)
	}

	// Create S3 client
	s3Client := s3.NewFromConfig(cfg)

	// Upload template to S3
	putObjectInput := &s3.PutObjectInput{
		Bucket:      aws.String(bucket),
		Key:         aws.String(objectKey),
		Body:        strings.NewReader(string(content)),
		ContentType: aws.String("application/x-yaml"),
	}

	_, err = s3Client.PutObject(ctx, putObjectInput)
	if err != nil {
		return fmt.Errorf("failed to upload template to S3 (s3://%s/%s): %w", bucket, objectKey, err)
	}

	return nil
}
