package templates

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// CacheEntry represents a cached template
type CacheEntry struct {
	Template     *Template `json:"template"`
	CachedAt     time.Time `json:"cached_at"`
	SourceURL    string    `json:"source_url"`
	SourceBranch string    `json:"source_branch"`
	TemplatePath string    `json:"template_path"`
}

// CommunityCache manages local caching of community templates
type CommunityCache struct {
	cacheDir string
	ttl      time.Duration // Time-to-live for cache entries
}

// CacheOptions configures the cache behavior
type CacheOptions struct {
	CacheDir string        // Directory for cache storage
	TTL      time.Duration // Time-to-live for cached templates
}

// DefaultCacheOptions returns default cache options
func DefaultCacheOptions() CacheOptions {
	homeDir, err := os.UserHomeDir()
	cacheDir := "/tmp/prism-community-cache"
	if err == nil {
		cacheDir = filepath.Join(homeDir, ".prism", "community", "templates")
	}

	return CacheOptions{
		CacheDir: cacheDir,
		TTL:      1 * time.Hour, // Default 1 hour TTL
	}
}

// NewCommunityCache creates a new community cache
func NewCommunityCache(options CacheOptions) *CommunityCache {
	return &CommunityCache{
		cacheDir: options.CacheDir,
		ttl:      options.TTL,
	}
}

// Get retrieves a template from cache if it exists and is not expired
func (cc *CommunityCache) Get(sourceURL, branch, templatePath string) (*Template, bool, error) {
	cacheKey := cc.generateCacheKey(sourceURL, branch, templatePath)
	cachePath := cc.getCachePath(cacheKey)

	// Check if cache file exists
	if _, err := os.Stat(cachePath); os.IsNotExist(err) {
		return nil, false, nil // Not in cache
	}

	// Read cache entry
	data, err := os.ReadFile(cachePath) //nolint:gosec // G304: path is cacheDir + sha256 hex key, no user-controlled path segments
	if err != nil {
		return nil, false, fmt.Errorf("failed to read cache file: %w", err)
	}

	// Unmarshal cache entry
	var entry CacheEntry
	if err := json.Unmarshal(data, &entry); err != nil {
		// Cache file is corrupted, remove it
		_ = os.Remove(cachePath)
		return nil, false, nil
	}

	// Check if cache entry is expired
	if time.Since(entry.CachedAt) > cc.ttl {
		// Cache expired, remove it
		_ = os.Remove(cachePath)
		return nil, false, nil
	}

	return entry.Template, true, nil
}

// Set stores a template in the cache
func (cc *CommunityCache) Set(sourceURL, branch, templatePath string, template *Template) error {
	// Ensure cache directory exists
	if err := os.MkdirAll(cc.cacheDir, 0700); err != nil {
		return fmt.Errorf("failed to create cache directory: %w", err)
	}

	cacheKey := cc.generateCacheKey(sourceURL, branch, templatePath)
	cachePath := cc.getCachePath(cacheKey)

	// Create cache entry
	entry := CacheEntry{
		Template:     template,
		CachedAt:     time.Now(),
		SourceURL:    sourceURL,
		SourceBranch: branch,
		TemplatePath: templatePath,
	}

	// Marshal to JSON
	data, err := json.MarshalIndent(entry, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal cache entry: %w", err)
	}

	// Write to cache file
	if err := os.WriteFile(cachePath, data, 0600); err != nil {
		return fmt.Errorf("failed to write cache file: %w", err)
	}

	return nil
}

// Invalidate removes a specific template from cache
func (cc *CommunityCache) Invalidate(sourceURL, branch, templatePath string) error {
	cacheKey := cc.generateCacheKey(sourceURL, branch, templatePath)
	cachePath := cc.getCachePath(cacheKey)

	// Remove cache file
	if err := os.Remove(cachePath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove cache file: %w", err)
	}

	return nil
}

// InvalidateSource removes all templates from a specific source
func (cc *CommunityCache) InvalidateSource(sourceURL, branch string) error {
	// List all cache files
	entries, err := os.ReadDir(cc.cacheDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // Cache directory doesn't exist
		}
		return fmt.Errorf("failed to read cache directory: %w", err)
	}

	// Remove cache files matching the source
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		cachePath := filepath.Join(cc.cacheDir, entry.Name())

		// Read cache entry to check source
		data, err := os.ReadFile(cachePath) //nolint:gosec // G304: entry from os.ReadDir of the Prism-owned cache dir
		if err != nil {
			continue // Skip files that can't be read
		}

		var cacheEntry CacheEntry
		if err := json.Unmarshal(data, &cacheEntry); err != nil {
			continue // Skip malformed cache files
		}

		// Remove if source matches
		if cacheEntry.SourceURL == sourceURL && cacheEntry.SourceBranch == branch {
			_ = os.Remove(cachePath)
		}
	}

	return nil
}

// Clear removes all cached templates
func (cc *CommunityCache) Clear() error {
	// Remove entire cache directory
	if err := os.RemoveAll(cc.cacheDir); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to clear cache: %w", err)
	}

	return nil
}

// CleanExpired removes all expired cache entries
func (cc *CommunityCache) CleanExpired() error {
	// List all cache files
	entries, err := os.ReadDir(cc.cacheDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // Cache directory doesn't exist
		}
		return fmt.Errorf("failed to read cache directory: %w", err)
	}

	// Check each cache file
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		cachePath := filepath.Join(cc.cacheDir, entry.Name())

		// Read cache entry
		data, err := os.ReadFile(cachePath) //nolint:gosec // G304: entry from os.ReadDir of the Prism-owned cache dir
		if err != nil {
			continue // Skip files that can't be read
		}

		var cacheEntry CacheEntry
		if err := json.Unmarshal(data, &cacheEntry); err != nil {
			// Malformed cache file, remove it
			_ = os.Remove(cachePath)
			continue
		}

		// Remove if expired
		if time.Since(cacheEntry.CachedAt) > cc.ttl {
			_ = os.Remove(cachePath)
		}
	}

	return nil
}

// GetCacheInfo returns statistics about the cache
func (cc *CommunityCache) GetCacheInfo() (totalEntries, expiredEntries int, totalSize int64, err error) {
	// List all cache files
	entries, err := os.ReadDir(cc.cacheDir)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, 0, 0, nil // Cache directory doesn't exist
		}
		return 0, 0, 0, fmt.Errorf("failed to read cache directory: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		totalEntries++

		cachePath := filepath.Join(cc.cacheDir, entry.Name())

		// Get file size
		info, err := entry.Info()
		if err == nil {
			totalSize += info.Size()
		}

		// Check if expired
		data, err := os.ReadFile(cachePath) //nolint:gosec // G304: entry from os.ReadDir of the Prism-owned cache dir
		if err != nil {
			continue
		}

		var cacheEntry CacheEntry
		if err := json.Unmarshal(data, &cacheEntry); err != nil {
			expiredEntries++ // Count malformed as expired
			continue
		}

		if time.Since(cacheEntry.CachedAt) > cc.ttl {
			expiredEntries++
		}
	}

	return totalEntries, expiredEntries, totalSize, nil
}

// generateCacheKey generates a unique cache key for a template
func (cc *CommunityCache) generateCacheKey(sourceURL, branch, templatePath string) string {
	// Create a unique key by hashing source + branch + path
	key := fmt.Sprintf("%s:%s:%s", sourceURL, branch, templatePath)
	hash := sha256.Sum256([]byte(key))
	return hex.EncodeToString(hash[:])
}

// getCachePath returns the filesystem path for a cache key
func (cc *CommunityCache) getCachePath(cacheKey string) string {
	return filepath.Join(cc.cacheDir, cacheKey+".json")
}

// ListCachedTemplates returns a list of all cached template entries
func (cc *CommunityCache) ListCachedTemplates() ([]CacheEntry, error) {
	// List all cache files
	entries, err := os.ReadDir(cc.cacheDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []CacheEntry{}, nil // Cache directory doesn't exist
		}
		return nil, fmt.Errorf("failed to read cache directory: %w", err)
	}

	var cachedTemplates []CacheEntry

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		cachePath := filepath.Join(cc.cacheDir, entry.Name())

		// Read cache entry
		data, err := os.ReadFile(cachePath) //nolint:gosec // G304: entry from os.ReadDir of the Prism-owned cache dir
		if err != nil {
			continue // Skip files that can't be read
		}

		var cacheEntry CacheEntry
		if err := json.Unmarshal(data, &cacheEntry); err != nil {
			continue // Skip malformed cache files
		}

		cachedTemplates = append(cachedTemplates, cacheEntry)
	}

	return cachedTemplates, nil
}
