package templates

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// CommunitySource represents a community template repository
type CommunitySource struct {
	Name       string    `json:"name"`
	URL        string    `json:"url"`    // GitHub repo URL
	Branch     string    `json:"branch"` // main/master/etc
	Enabled    bool      `json:"enabled"`
	TrustLevel string    `json:"trust_level"` // verified/community/unverified
	Added      time.Time `json:"added"`
	LastSync   time.Time `json:"last_sync,omitempty"`
}

// SourceManager manages community template sources
type SourceManager struct {
	sources    map[string]*CommunitySource
	configPath string
}

// NewSourceManager creates a new source manager
func NewSourceManager() *SourceManager {
	configPath := getSourcesConfigPath()
	sm := &SourceManager{
		sources:    make(map[string]*CommunitySource),
		configPath: configPath,
	}

	// Load existing sources
	// A missing file is not an error (first run); anything else is a corrupt or
	// unreadable config, so start with empty sources but say so.
	if err := sm.Load(); err != nil {
		log.Printf("warning: ignoring community template sources config: %v", err)
	}

	return sm
}

// Add adds a new community source
func (sm *SourceManager) Add(url, branch string) (*CommunitySource, error) {
	// Normalize URL
	url = normalizeGitHubURL(url)

	// Extract repository name for source name
	name, err := extractRepoName(url)
	if err != nil {
		return nil, fmt.Errorf("failed to extract repository name: %w", err)
	}

	// Check if source already exists
	if _, exists := sm.sources[name]; exists {
		return nil, fmt.Errorf("source '%s' already exists", name)
	}

	// Default to main branch if not specified
	if branch == "" {
		branch = "main"
	}

	source := &CommunitySource{
		Name:       name,
		URL:        url,
		Branch:     branch,
		Enabled:    true,
		TrustLevel: "community", // Default trust level
		Added:      time.Now(),
	}

	sm.sources[name] = source

	// Save to disk
	if err := sm.Save(); err != nil {
		return nil, fmt.Errorf("failed to save sources: %w", err)
	}

	return source, nil
}

// Remove removes a community source
func (sm *SourceManager) Remove(name string) error {
	if _, exists := sm.sources[name]; !exists {
		return fmt.Errorf("source '%s' not found", name)
	}

	delete(sm.sources, name)

	// Save to disk
	if err := sm.Save(); err != nil {
		return fmt.Errorf("failed to save sources: %w", err)
	}

	return nil
}

// Get retrieves a community source by name
func (sm *SourceManager) Get(name string) (*CommunitySource, error) {
	source, exists := sm.sources[name]
	if !exists {
		return nil, fmt.Errorf("source '%s' not found", name)
	}
	return source, nil
}

// List returns all community sources
func (sm *SourceManager) List() []*CommunitySource {
	sources := make([]*CommunitySource, 0, len(sm.sources))
	for _, source := range sm.sources {
		sources = append(sources, source)
	}
	return sources
}

// Enable enables a community source
func (sm *SourceManager) Enable(name string) error {
	source, exists := sm.sources[name]
	if !exists {
		return fmt.Errorf("source '%s' not found", name)
	}

	source.Enabled = true

	// Save to disk
	if err := sm.Save(); err != nil {
		return fmt.Errorf("failed to save sources: %w", err)
	}

	return nil
}

// Disable disables a community source
func (sm *SourceManager) Disable(name string) error {
	source, exists := sm.sources[name]
	if !exists {
		return fmt.Errorf("source '%s' not found", name)
	}

	source.Enabled = false

	// Save to disk
	if err := sm.Save(); err != nil {
		return fmt.Errorf("failed to save sources: %w", err)
	}

	return nil
}

// SetTrustLevel sets the trust level for a source
func (sm *SourceManager) SetTrustLevel(name, trustLevel string) error {
	source, exists := sm.sources[name]
	if !exists {
		return fmt.Errorf("source '%s' not found", name)
	}

	// Validate trust level
	validLevels := map[string]bool{
		"verified":   true,
		"community":  true,
		"unverified": true,
	}
	if !validLevels[trustLevel] {
		return fmt.Errorf("invalid trust level: %s (must be verified, community, or unverified)", trustLevel)
	}

	source.TrustLevel = trustLevel

	// Save to disk
	if err := sm.Save(); err != nil {
		return fmt.Errorf("failed to save sources: %w", err)
	}

	return nil
}

// UpdateLastSync updates the last sync time for a source
func (sm *SourceManager) UpdateLastSync(name string) error {
	source, exists := sm.sources[name]
	if !exists {
		return fmt.Errorf("source '%s' not found", name)
	}

	source.LastSync = time.Now()

	// Save to disk
	if err := sm.Save(); err != nil {
		return fmt.Errorf("failed to save sources: %w", err)
	}

	return nil
}

// Save persists sources to disk
func (sm *SourceManager) Save() error {
	// Ensure directory exists
	dir := filepath.Dir(sm.configPath)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Marshal sources to JSON
	data, err := json.MarshalIndent(sm.sources, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal sources: %w", err)
	}

	// Write to file
	if err := os.WriteFile(sm.configPath, data, 0600); err != nil {
		return fmt.Errorf("failed to write sources file: %w", err)
	}

	return nil
}

// Load loads sources from disk
func (sm *SourceManager) Load() error {
	// Check if file exists
	if _, err := os.Stat(sm.configPath); os.IsNotExist(err) {
		// File doesn't exist, return empty sources
		return nil
	}

	// Read file
	data, err := os.ReadFile(sm.configPath)
	if err != nil {
		return fmt.Errorf("failed to read sources file: %w", err)
	}

	// Unmarshal JSON
	if err := json.Unmarshal(data, &sm.sources); err != nil {
		return fmt.Errorf("failed to unmarshal sources: %w", err)
	}

	return nil
}

// getSourcesConfigPath returns the path to the sources configuration file
func getSourcesConfigPath() string {
	// Try environment variable first
	if customPath := os.Getenv("PRISM_SOURCES_CONFIG"); customPath != "" {
		return customPath
	}

	// Use default location: ~/.prism/community/sources.json
	homeDir, err := os.UserHomeDir()
	if err != nil {
		// Fallback to /tmp
		return "/tmp/prism-community-sources.json"
	}

	return filepath.Join(homeDir, ".prism", "community", "sources.json")
}

// normalizeGitHubURL normalizes a GitHub URL to standard format
func normalizeGitHubURL(url string) string {
	// Remove trailing slash
	url = strings.TrimSuffix(url, "/")

	// Ensure https:// prefix for full URLs
	if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
		// If it looks like owner/repo format, add github.com prefix
		if len(strings.Split(url, "/")) == 2 && !strings.Contains(url, ".") {
			url = "https://github.com/" + url
		}
	}

	// Remove .git suffix
	url = strings.TrimSuffix(url, ".git")

	return url
}

// extractRepoName extracts the repository name from a GitHub URL
func extractRepoName(url string) (string, error) {
	// Normalize URL first
	url = normalizeGitHubURL(url)

	// Remove common prefixes
	url = strings.TrimPrefix(url, "https://github.com/")
	url = strings.TrimPrefix(url, "http://github.com/")
	url = strings.TrimPrefix(url, "github.com/")

	// Split into parts
	parts := strings.Split(url, "/")
	if len(parts) < 2 {
		return "", fmt.Errorf("invalid GitHub URL format")
	}

	// Use owner/repo as the name
	return fmt.Sprintf("%s/%s", parts[0], parts[1]), nil
}

// AddDefaultSources adds the official Prism community repository as a default source
func (sm *SourceManager) AddDefaultSources() error {
	// Check if official community repo is already added
	officialName := "scttfrdmn/prism-template-registry"
	if _, exists := sm.sources[officialName]; exists {
		return nil // Already exists
	}

	source := &CommunitySource{
		Name:       officialName,
		URL:        "https://github.com/scttfrdmn/prism-template-registry",
		Branch:     "main",
		Enabled:    true,
		TrustLevel: "verified", // Official repo is verified
		Added:      time.Now(),
	}

	sm.sources[officialName] = source

	// Save to disk
	if err := sm.Save(); err != nil {
		return fmt.Errorf("failed to save sources: %w", err)
	}

	return nil
}
