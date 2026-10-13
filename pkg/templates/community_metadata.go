package templates

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// CommunityMetadata extends template with community-specific information
type CommunityMetadata struct {
	// TemplateName is the canonical name of the template this metadata belongs to.
	// Populated automatically by SetMetadata so callers can identify returned records
	// in list/search results without a separate key lookup.
	TemplateName string `json:"template_name,omitempty"`

	// Author information
	Author          string   `json:"author"`
	MaintainerEmail string   `json:"maintainer_email,omitempty"`
	Repository      string   `json:"repository"`
	License         string   `json:"license"`
	Keywords        []string `json:"keywords,omitempty"`

	// Versioning
	TemplateVersion string    `json:"template_version"`
	PrismMinVersion string    `json:"prism_min_version"`
	LastUpdated     time.Time `json:"last_updated"`
	Changelog       string    `json:"changelog,omitempty"`

	// Support and documentation
	DocumentationURL string `json:"documentation_url,omitempty"`
	IssuesURL        string `json:"issues_url,omitempty"`
	HomepageURL      string `json:"homepage_url,omitempty"`

	// Community engagement
	Downloads      int       `json:"downloads"`
	Stars          int       `json:"stars"`
	Rating         float64   `json:"rating"` // 0.0-5.0
	RatingCount    int       `json:"rating_count"`
	Featured       bool      `json:"featured"`
	Verified       bool      `json:"verified"`
	LastDownloaded time.Time `json:"last_downloaded,omitempty"`

	// Security
	SecurityScanned  bool      `json:"security_scanned"`
	SecurityScanDate time.Time `json:"security_scan_date,omitempty"`
	SecurityScore    int       `json:"security_score"` // 0-100
	TrustLevel       string    `json:"trust_level"`    // verified/community/unverified

	// Dependencies and compatibility
	Dependencies      []string `json:"dependencies,omitempty"`
	ConflictsWith     []string `json:"conflicts_with,omitempty"`
	RecommendedWith   []string `json:"recommended_with,omitempty"`
	CompatibleRegions []string `json:"compatible_regions,omitempty"`

	// Usage statistics
	UsageStats UsageStatistics `json:"usage_stats,omitempty"`
}

// UsageStatistics tracks template usage patterns
type UsageStatistics struct {
	TotalLaunches      int       `json:"total_launches"`
	SuccessfulLaunches int       `json:"successful_launches"`
	FailedLaunches     int       `json:"failed_launches"`
	AverageLaunchTime  int       `json:"average_launch_time"` // seconds
	LastUsed           time.Time `json:"last_used,omitempty"`
	PopularityScore    float64   `json:"popularity_score"` // Derived from usage
}

// EnhancedTemplate combines Template with CommunityMetadata
type EnhancedTemplate struct {
	*Template
	Community *CommunityMetadata `json:"community,omitempty"`
}

// CommunityMetadataStore manages community metadata storage
type CommunityMetadataStore struct {
	metadataDir string
	mu          sync.Mutex
}

// NewCommunityMetadataStore creates a new metadata store
func NewCommunityMetadataStore() *CommunityMetadataStore {
	metadataDir := getCommunityMetadataDir()
	return &CommunityMetadataStore{
		metadataDir: metadataDir,
	}
}

// GetMetadata retrieves community metadata for a template
func (cms *CommunityMetadataStore) GetMetadata(templateName, sourceURL string) (*CommunityMetadata, error) {
	cms.mu.Lock()
	defer cms.mu.Unlock()
	return cms.getMetadataLocked(templateName, sourceURL)
}

// getMetadataLocked reads metadata from disk; caller must hold cms.mu
func (cms *CommunityMetadataStore) getMetadataLocked(templateName, sourceURL string) (*CommunityMetadata, error) {
	metadataPath := cms.getMetadataPath(templateName, sourceURL)

	// Check if metadata exists
	if _, err := os.Stat(metadataPath); os.IsNotExist(err) {
		// Return default metadata if file doesn't exist
		return cms.createDefaultMetadata(templateName, sourceURL), nil
	}

	// Read metadata file
	data, err := os.ReadFile(metadataPath) //nolint:gosec // G304: path is metadataDir + filename with path separators stripped (getMetadataPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read metadata file: %w", err)
	}

	// Parse metadata
	var metadata CommunityMetadata
	if err := json.Unmarshal(data, &metadata); err != nil {
		return nil, fmt.Errorf("failed to parse metadata: %w", err)
	}

	return &metadata, nil
}

// SetMetadata stores community metadata for a template
func (cms *CommunityMetadataStore) SetMetadata(templateName, sourceURL string, metadata *CommunityMetadata) error {
	cms.mu.Lock()
	defer cms.mu.Unlock()
	return cms.setMetadataLocked(templateName, sourceURL, metadata)
}

// setMetadataLocked writes metadata to disk; caller must hold cms.mu
func (cms *CommunityMetadataStore) setMetadataLocked(templateName, sourceURL string, metadata *CommunityMetadata) error {
	// Stamp the canonical template name so list/search results are self-identifying.
	metadata.TemplateName = templateName

	// Ensure metadata directory exists
	if err := os.MkdirAll(cms.metadataDir, 0700); err != nil {
		return fmt.Errorf("failed to create metadata directory: %w", err)
	}

	metadataPath := cms.getMetadataPath(templateName, sourceURL)

	// Marshal metadata to JSON
	data, err := json.MarshalIndent(metadata, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	// Write metadata file
	if err := os.WriteFile(metadataPath, data, 0600); err != nil {
		return fmt.Errorf("failed to write metadata file: %w", err)
	}

	return nil
}

// UpdateDownloadCount increments the download counter (thread-safe)
func (cms *CommunityMetadataStore) UpdateDownloadCount(templateName, sourceURL string) error {
	cms.mu.Lock()
	defer cms.mu.Unlock()

	metadata, err := cms.getMetadataLocked(templateName, sourceURL)
	if err != nil {
		return err
	}

	metadata.Downloads++
	metadata.LastDownloaded = time.Now()

	return cms.setMetadataLocked(templateName, sourceURL, metadata)
}

// UpdateUsageStats updates usage statistics after a launch (thread-safe)
func (cms *CommunityMetadataStore) UpdateUsageStats(templateName, sourceURL string, success bool, launchTime int) error {
	cms.mu.Lock()
	defer cms.mu.Unlock()

	metadata, err := cms.getMetadataLocked(templateName, sourceURL)
	if err != nil {
		return err
	}

	metadata.UsageStats.TotalLaunches++
	if success {
		metadata.UsageStats.SuccessfulLaunches++
	} else {
		metadata.UsageStats.FailedLaunches++
	}

	// Update average launch time (only for non-zero launch times to avoid skewing average with failures)
	if launchTime > 0 {
		if metadata.UsageStats.AverageLaunchTime == 0 {
			metadata.UsageStats.AverageLaunchTime = launchTime
		} else {
			metadata.UsageStats.AverageLaunchTime =
				(metadata.UsageStats.AverageLaunchTime + launchTime) / 2
		}
	}

	metadata.UsageStats.LastUsed = time.Now()

	// Calculate popularity score
	metadata.UsageStats.PopularityScore = cms.calculatePopularityScore(metadata)

	return cms.setMetadataLocked(templateName, sourceURL, metadata)
}

// UpdateSecurityScan updates security scan results (thread-safe)
func (cms *CommunityMetadataStore) UpdateSecurityScan(templateName, sourceURL string, score int) error {
	cms.mu.Lock()
	defer cms.mu.Unlock()

	metadata, err := cms.getMetadataLocked(templateName, sourceURL)
	if err != nil {
		return err
	}

	metadata.SecurityScanned = true
	metadata.SecurityScanDate = time.Now()
	metadata.SecurityScore = score

	return cms.setMetadataLocked(templateName, sourceURL, metadata)
}

// AddRating adds a user rating to the template (thread-safe)
func (cms *CommunityMetadataStore) AddRating(templateName, sourceURL string, rating float64) error {
	if rating < 0 || rating > 5 {
		return fmt.Errorf("rating must be between 0 and 5")
	}

	cms.mu.Lock()
	defer cms.mu.Unlock()

	metadata, err := cms.getMetadataLocked(templateName, sourceURL)
	if err != nil {
		return err
	}

	// Calculate new average rating
	totalRating := metadata.Rating * float64(metadata.RatingCount)
	metadata.RatingCount++
	metadata.Rating = (totalRating + rating) / float64(metadata.RatingCount)

	return cms.setMetadataLocked(templateName, sourceURL, metadata)
}

// ListAllMetadata returns metadata for all templates
func (cms *CommunityMetadataStore) ListAllMetadata() ([]*CommunityMetadata, error) {
	// Check if metadata directory exists
	if _, err := os.Stat(cms.metadataDir); os.IsNotExist(err) {
		return []*CommunityMetadata{}, nil
	}

	// Read all metadata files
	entries, err := os.ReadDir(cms.metadataDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read metadata directory: %w", err)
	}

	var allMetadata []*CommunityMetadata

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}

		metadataPath := filepath.Join(cms.metadataDir, entry.Name())
		data, err := os.ReadFile(metadataPath) //nolint:gosec // G304: entry from os.ReadDir of the Prism-owned metadata dir
		if err != nil {
			continue
		}

		var metadata CommunityMetadata
		if err := json.Unmarshal(data, &metadata); err != nil {
			continue
		}

		allMetadata = append(allMetadata, &metadata)
	}

	return allMetadata, nil
}

// GetPopularTemplates returns the most popular templates
func (cms *CommunityMetadataStore) GetPopularTemplates(limit int) ([]*CommunityMetadata, error) {
	allMetadata, err := cms.ListAllMetadata()
	if err != nil {
		return nil, err
	}

	// Sort by popularity score
	for i := 0; i < len(allMetadata); i++ {
		for j := i + 1; j < len(allMetadata); j++ {
			if allMetadata[j].UsageStats.PopularityScore > allMetadata[i].UsageStats.PopularityScore {
				allMetadata[i], allMetadata[j] = allMetadata[j], allMetadata[i]
			}
		}
	}

	// Return top N
	if limit > len(allMetadata) {
		limit = len(allMetadata)
	}

	return allMetadata[:limit], nil
}

// GetFeaturedTemplates returns featured templates
func (cms *CommunityMetadataStore) GetFeaturedTemplates() ([]*CommunityMetadata, error) {
	allMetadata, err := cms.ListAllMetadata()
	if err != nil {
		return nil, err
	}

	var featured []*CommunityMetadata
	for _, metadata := range allMetadata {
		if metadata.Featured {
			featured = append(featured, metadata)
		}
	}

	return featured, nil
}

// GetHighRatedTemplates returns templates with ratings above threshold
func (cms *CommunityMetadataStore) GetHighRatedTemplates(minRating float64, minRatingCount int) ([]*CommunityMetadata, error) {
	allMetadata, err := cms.ListAllMetadata()
	if err != nil {
		return nil, err
	}

	var highRated []*CommunityMetadata
	for _, metadata := range allMetadata {
		if metadata.Rating >= minRating && metadata.RatingCount >= minRatingCount {
			highRated = append(highRated, metadata)
		}
	}

	// Sort by rating
	for i := 0; i < len(highRated); i++ {
		for j := i + 1; j < len(highRated); j++ {
			if highRated[j].Rating > highRated[i].Rating {
				highRated[i], highRated[j] = highRated[j], highRated[i]
			}
		}
	}

	return highRated, nil
}

// SearchByKeywords searches templates by keywords
func (cms *CommunityMetadataStore) SearchByKeywords(keywords []string) ([]*CommunityMetadata, error) {
	allMetadata, err := cms.ListAllMetadata()
	if err != nil {
		return nil, err
	}

	var matches []*CommunityMetadata

	for _, metadata := range allMetadata {
		for _, searchKeyword := range keywords {
			searchLower := strings.ToLower(searchKeyword)

			// Check template keywords
			for _, templateKeyword := range metadata.Keywords {
				if strings.Contains(strings.ToLower(templateKeyword), searchLower) {
					matches = append(matches, metadata)
					goto nextTemplate
				}
			}
		}
	nextTemplate:
	}

	return matches, nil
}

// Helper functions

// getMetadataPath returns the filesystem path for template metadata
func (cms *CommunityMetadataStore) getMetadataPath(templateName, sourceURL string) string {
	// Create a safe filename from template name and source
	safeName := strings.ReplaceAll(templateName, " ", "_")
	safeName = strings.ReplaceAll(safeName, "/", "_")

	// Include source in filename to avoid collisions
	sourceHash := fmt.Sprintf("%x", []byte(sourceURL))[:8]
	filename := fmt.Sprintf("%s_%s.json", safeName, sourceHash)

	return filepath.Join(cms.metadataDir, filename)
}

// createDefaultMetadata creates default metadata for a template
func (cms *CommunityMetadataStore) createDefaultMetadata(templateName, sourceURL string) *CommunityMetadata {
	return &CommunityMetadata{
		Author:          "Unknown",
		Repository:      sourceURL,
		License:         "Unknown",
		Keywords:        []string{},
		TemplateVersion: "1.0.0",
		PrismMinVersion: "0.1.0",
		LastUpdated:     time.Now(),
		Downloads:       0,
		Stars:           0,
		Rating:          0.0,
		RatingCount:     0,
		Featured:        false,
		Verified:        false,
		SecurityScanned: false,
		SecurityScore:   0,
		TrustLevel:      "unverified",
		UsageStats: UsageStatistics{
			TotalLaunches:      0,
			SuccessfulLaunches: 0,
			FailedLaunches:     0,
			AverageLaunchTime:  0,
			PopularityScore:    0.0,
		},
	}
}

// calculatePopularityScore computes a popularity score
func (cms *CommunityMetadataStore) calculatePopularityScore(metadata *CommunityMetadata) float64 {
	// Weighted formula for popularity
	// Downloads: 40%, Success Rate: 30%, Rating: 20%, Stars: 10%

	downloadsScore := float64(metadata.Downloads) * 0.4

	successRate := 0.0
	if metadata.UsageStats.TotalLaunches > 0 {
		successRate = float64(metadata.UsageStats.SuccessfulLaunches) /
			float64(metadata.UsageStats.TotalLaunches)
	}
	successScore := successRate * 100 * 0.3

	ratingScore := metadata.Rating * 20 * 0.2

	starsScore := float64(metadata.Stars) * 0.1

	return downloadsScore + successScore + ratingScore + starsScore
}

// getCommunityMetadataDir returns the metadata directory path
func getCommunityMetadataDir() string {
	// Try environment variable first
	if customPath := os.Getenv("PRISM_COMMUNITY_METADATA_DIR"); customPath != "" {
		return customPath
	}

	// Use default location: ~/.prism/community/metadata
	homeDir, err := os.UserHomeDir()
	if err != nil {
		// Fallback to /tmp
		return "/tmp/prism-community-metadata"
	}

	return filepath.Join(homeDir, ".prism", "community", "metadata")
}

// EnhanceTemplate adds community metadata to a template
func EnhanceTemplate(template *Template, sourceURL string) (*EnhancedTemplate, error) {
	store := NewCommunityMetadataStore()
	metadata, err := store.GetMetadata(template.Name, sourceURL)
	if err != nil {
		return nil, fmt.Errorf("failed to get metadata: %w", err)
	}

	return &EnhancedTemplate{
		Template:  template,
		Community: metadata,
	}, nil
}

// GetCompatibilityInfo returns compatibility information for a template
func (metadata *CommunityMetadata) GetCompatibilityInfo() string {
	var info strings.Builder

	if len(metadata.Dependencies) > 0 {
		info.WriteString(fmt.Sprintf("Dependencies: %s\n", strings.Join(metadata.Dependencies, ", ")))
	}

	if len(metadata.ConflictsWith) > 0 {
		info.WriteString(fmt.Sprintf("Conflicts with: %s\n", strings.Join(metadata.ConflictsWith, ", ")))
	}

	if len(metadata.RecommendedWith) > 0 {
		info.WriteString(fmt.Sprintf("Recommended with: %s\n", strings.Join(metadata.RecommendedWith, ", ")))
	}

	if len(metadata.CompatibleRegions) > 0 {
		info.WriteString(fmt.Sprintf("Compatible regions: %s\n", strings.Join(metadata.CompatibleRegions, ", ")))
	}

	return info.String()
}

// GetTrustBadge returns a visual representation of trust level
func (metadata *CommunityMetadata) GetTrustBadge() string {
	switch metadata.TrustLevel {
	case "verified":
		return "🛡️ VERIFIED"
	case "community":
		return "🤝 COMMUNITY"
	case "unverified":
		return "⚠️ UNVERIFIED"
	default:
		return "❓ UNKNOWN"
	}
}

// GetSecurityBadge returns a visual representation of security status
func (metadata *CommunityMetadata) GetSecurityBadge() string {
	if !metadata.SecurityScanned {
		return "⚪ NOT SCANNED"
	}

	score := metadata.SecurityScore
	switch {
	case score >= 90:
		return "🟢 EXCELLENT"
	case score >= 75:
		return "🟢 GOOD"
	case score >= 60:
		return "🟡 FAIR"
	case score >= 40:
		return "🟠 POOR"
	default:
		return "🔴 CRITICAL"
	}
}

// GetRatingDisplay returns a star rating display
func (metadata *CommunityMetadata) GetRatingDisplay() string {
	if metadata.RatingCount == 0 {
		return "☆☆☆☆☆ (no ratings)"
	}

	fullStars := int(metadata.Rating)
	halfStar := metadata.Rating-float64(fullStars) >= 0.5

	display := strings.Repeat("★", fullStars)
	if halfStar {
		display += "½"
	}
	emptyStars := 5 - fullStars
	if halfStar {
		emptyStars--
	}
	display += strings.Repeat("☆", emptyStars)

	return fmt.Sprintf("%s (%.1f from %d ratings)", display, metadata.Rating, metadata.RatingCount)
}
