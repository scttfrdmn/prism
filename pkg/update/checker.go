package update

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"golang.org/x/mod/semver"

	"github.com/scttfrdmn/prism/pkg/version"
)

const (
	githubAPIURL    = "https://api.github.com/repos/scttfrdmn/prism/releases/latest"
	defaultCacheTTL = 24 * time.Hour
	cacheFileName   = "update_check.json"
)

// Checker handles version checking and update notifications
type Checker struct {
	cacheDir   string
	cacheTTL   time.Duration
	channel    UpdateChannel
	httpClient *http.Client
}

// NewChecker creates a new update checker
func NewChecker() (*Checker, error) {
	// Get cache directory
	cacheDir, err := getCacheDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get cache directory: %w", err)
	}

	return &Checker{
		cacheDir:   cacheDir,
		cacheTTL:   defaultCacheTTL,
		channel:    ChannelStable,
		httpClient: &http.Client{Timeout: 10 * time.Second},
	}, nil
}

// CheckForUpdates checks if a new version is available
func (c *Checker) CheckForUpdates(ctx context.Context) (*UpdateInfo, error) {
	// Try to get cached result first
	if cached := c.getCachedUpdate(); cached != nil && !cached.IsExpired() {
		// The cache stores the CurrentVersion captured at the time of the
		// check, but the running binary may have been upgraded since. Recompute
		// against the actual running version so we don't replay a stale
		// "0.30.0 -> 0.35.4" notification after the user has already updated.
		info := *cached.UpdateInfo
		info.CurrentVersion = version.Version
		info.IsUpdateAvailable = compareVersions(version.Version, info.LatestVersion)
		return &info, nil
	}

	// Fetch latest version from GitHub
	latestVersion, releaseURL, releaseNotes, publishedAt, err := c.fetchLatestVersion(ctx)
	if err != nil {
		// If we have a cached result, return it even if expired
		if cached := c.getCachedUpdate(); cached != nil {
			return cached.UpdateInfo, nil
		}
		return nil, fmt.Errorf("failed to fetch latest version: %w", err)
	}

	// Get current version
	currentVersion := version.Version

	// Compare versions
	isUpdateAvailable := compareVersions(currentVersion, latestVersion)

	// Detect installation method
	installMethod := DetectInstallMethod()
	updateCommand := GetUpdateCommand(installMethod)

	updateInfo := &UpdateInfo{
		CurrentVersion:    currentVersion,
		LatestVersion:     latestVersion,
		IsUpdateAvailable: isUpdateAvailable,
		ReleaseURL:        releaseURL,
		ReleaseNotes:      releaseNotes,
		PublishedAt:       publishedAt,
		InstallMethod:     installMethod,
		UpdateCommand:     updateCommand,
		LastChecked:       time.Now(),
	}

	// Cache the result
	c.cacheUpdate(updateInfo)

	return updateInfo, nil
}

// fetchLatestVersion queries GitHub API for the latest release
func (c *Checker) fetchLatestVersion(ctx context.Context) (version, url, notes string, publishedAt time.Time, err error) {
	req, err := http.NewRequestWithContext(ctx, "GET", githubAPIURL, nil)
	if err != nil {
		return "", "", "", time.Time{}, err
	}

	// Add User-Agent header (GitHub API requires it)
	req.Header.Set("User-Agent", "Prism-Update-Checker")
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", "", "", time.Time{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", "", "", time.Time{}, fmt.Errorf("GitHub API returned status %d: %s", resp.StatusCode, string(body))
	}

	var release struct {
		TagName     string    `json:"tag_name"`
		HTMLURL     string    `json:"html_url"`
		Body        string    `json:"body"`
		PublishedAt time.Time `json:"published_at"`
		Prerelease  bool      `json:"prerelease"`
		Draft       bool      `json:"draft"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return "", "", "", time.Time{}, err
	}

	// Skip draft or pre-release versions if on stable channel
	if c.channel == ChannelStable && (release.Draft || release.Prerelease) {
		return "", "", "", time.Time{}, fmt.Errorf("latest release is a draft or prerelease")
	}

	// Strip 'v' prefix from version if present
	version = strings.TrimPrefix(release.TagName, "v")

	return version, release.HTMLURL, release.Body, release.PublishedAt, nil
}

// compareVersions returns true if latest > current using semver ordering.
// Dev/unknown current versions are never treated as upgradable, since the user
// is running an unreleased build and we have no meaningful comparison point.
func compareVersions(current, latest string) bool {
	current = strings.TrimPrefix(current, "v")
	latest = strings.TrimPrefix(latest, "v")
	if current == "" || current == "dev" || current == "unknown" {
		return false
	}
	// semver.Compare requires a "v" prefix and treats invalid versions as less
	// than valid ones, so guard explicitly to avoid false update prompts.
	cv, lv := "v"+current, "v"+latest
	if !semver.IsValid(cv) || !semver.IsValid(lv) {
		return false
	}
	return semver.Compare(lv, cv) > 0
}

// getCachedUpdate retrieves cached update check result
func (c *Checker) getCachedUpdate() *CachedUpdateCheck {
	cachePath := filepath.Join(c.cacheDir, cacheFileName)

	data, err := os.ReadFile(cachePath)
	if err != nil {
		return nil
	}

	var cached CachedUpdateCheck
	if err := json.Unmarshal(data, &cached); err != nil {
		return nil
	}

	cached.CacheTTL = c.cacheTTL
	return &cached
}

// cacheUpdate stores update check result in cache
func (c *Checker) cacheUpdate(info *UpdateInfo) {
	cachePath := filepath.Join(c.cacheDir, cacheFileName)

	cached := CachedUpdateCheck{
		UpdateInfo: info,
		CheckedAt:  time.Now(),
		CacheTTL:   c.cacheTTL,
	}

	data, err := json.MarshalIndent(cached, "", "  ")
	if err != nil {
		return
	}

	// Ensure cache directory exists
	os.MkdirAll(c.cacheDir, 0755)

	// Write cache file
	os.WriteFile(cachePath, data, 0644)
}

// ClearCache removes cached update check
func (c *Checker) ClearCache() error {
	cachePath := filepath.Join(c.cacheDir, cacheFileName)
	return os.Remove(cachePath)
}

// getCacheDir returns the cache directory for update checks
func getCacheDir() (string, error) {
	// Use ~/.prism/cache for update checks
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	cacheDir := filepath.Join(homeDir, ".prism", "cache")
	return cacheDir, nil
}

// FormatUpdateMessage formats a user-friendly update notification message
func FormatUpdateMessage(info *UpdateInfo) string {
	if !info.IsUpdateAvailable {
		return fmt.Sprintf("✓ You're running the latest version (%s)", info.CurrentVersion)
	}

	var message strings.Builder

	message.WriteString("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
	message.WriteString(fmt.Sprintf("🎉 New Prism version available: %s → %s\n", info.CurrentVersion, info.LatestVersion))
	message.WriteString("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n\n")

	// Show installation method
	message.WriteString(fmt.Sprintf("Installation method: %s\n\n", info.InstallMethod))

	// Show update command
	message.WriteString("To update:\n")
	message.WriteString(fmt.Sprintf("  %s\n\n", GetUpdateInstructions(info.InstallMethod, info.LatestVersion)))

	// Show release info
	message.WriteString(fmt.Sprintf("📋 Release: %s\n", info.ReleaseURL))
	message.WriteString(fmt.Sprintf("📅 Published: %s\n\n", info.PublishedAt.Format("January 2, 2006")))

	// Show truncated release notes if available
	if info.ReleaseNotes != "" {
		notes := info.ReleaseNotes
		if len(notes) > 300 {
			notes = notes[:300] + "..."
		}
		message.WriteString("Release Notes:\n")
		message.WriteString(notes)
		message.WriteString("\n\n")
	}

	message.WriteString("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")

	return message.String()
}

// FormatShortUpdateMessage formats a brief update notification
func FormatShortUpdateMessage(info *UpdateInfo) string {
	if !info.IsUpdateAvailable {
		return ""
	}

	return fmt.Sprintf("💡 New version available: %s → %s. Run: %s",
		info.CurrentVersion, info.LatestVersion, info.UpdateCommand)
}
