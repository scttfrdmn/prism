// Package templates provides marketplace registry functionality for CloudWorkstation templates.
//
// The template marketplace enables centralized discovery, validation, and sharing of
// research environments through community, institutional, and commercial registries.
package templates

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"
)

// MarketplaceTemplateRegistry provides centralized template discovery and management
type MarketplaceTemplateRegistry struct {
	Name        string
	URL         string
	Type        RegistryType
	Credentials *RegistryCredentials
	Client      *http.Client
}

// RegistryType defines the type of template registry
type RegistryType string

const (
	RegistryTypeCommunity     RegistryType = "community"
	RegistryTypeInstitutional RegistryType = "institutional"
	RegistryTypePrivate       RegistryType = "private"
	RegistryTypeOfficial      RegistryType = "official"
)

// RegistryCredentials holds authentication information for private registries
type RegistryCredentials struct {
	Type     string `json:"type"` // token, basic, ssh_key
	Token    string `json:"token,omitempty"`
	Username string `json:"username,omitempty"`
	Password string `json:"password,omitempty"`
	SSHKey   string `json:"ssh_key,omitempty"`
}

// MarketplaceTemplateRegistryEntry represents a template entry in the registry
type MarketplaceTemplateRegistryEntry struct {
	*Template

	// Registry-specific metadata
	RegistryName string    `json:"registry_name"`
	RegistryType string    `json:"registry_type"`
	LastSynced   time.Time `json:"last_synced"`

	// Enhanced search metadata
	SearchTags     []string `json:"search_tags"`     // Computed from template content
	PopularityRank int      `json:"popularity_rank"` // Registry-specific ranking
}

// SearchFilter defines criteria for template marketplace searches
type SearchFilter struct {
	// Text search
	Query    string   `json:"query,omitempty"`
	Keywords []string `json:"keywords,omitempty"`

	// Classification filters
	Categories []string             `json:"categories,omitempty"`
	Domains    []string             `json:"domains,omitempty"`
	Complexity []TemplateComplexity `json:"complexity,omitempty"`

	// Quality filters
	MinRating     float64 `json:"min_rating,omitempty"`
	VerifiedOnly  bool    `json:"verified_only,omitempty"`
	ValidatedOnly bool    `json:"validated_only,omitempty"`

	// Registry filters
	Registries    []string       `json:"registries,omitempty"`
	RegistryTypes []RegistryType `json:"registry_types,omitempty"`

	// Feature filters
	ResearchUserSupport bool     `json:"research_user_support,omitempty"`
	ConnectionTypes     []string `json:"connection_types,omitempty"`
	PackageManagers     []string `json:"package_managers,omitempty"`

	// Sorting and pagination
	SortBy    string `json:"sort_by,omitempty"`    // popularity, rating, updated, name
	SortOrder string `json:"sort_order,omitempty"` // asc, desc
	Limit     int    `json:"limit,omitempty"`
	Offset    int    `json:"offset,omitempty"`
}

// SearchResult contains paginated search results
type MarketplaceSearchResult struct {
	Templates     []MarketplaceTemplateRegistryEntry `json:"templates"`
	TotalCount    int                                `json:"total_count"`
	FilteredCount int                                `json:"filtered_count"`
	Query         SearchFilter                       `json:"query"`
	ExecutionTime time.Duration                      `json:"execution_time"`
}

// NewMarketplaceTemplateRegistry creates a new registry client
func NewMarketplaceRegistry(name, url string, registryType RegistryType) *MarketplaceTemplateRegistry {
	return &MarketplaceTemplateRegistry{
		Name:   name,
		URL:    url,
		Type:   registryType,
		Client: &http.Client{Timeout: 30 * time.Second},
	}
}

// SetCredentials configures authentication for private registries
func (r *MarketplaceTemplateRegistry) SetCredentials(creds *RegistryCredentials) {
	r.Credentials = creds
}

// Search performs a template search across the registry
func (r *MarketplaceTemplateRegistry) Search(ctx context.Context, filter SearchFilter) (*MarketplaceSearchResult, error) {
	if filter.Limit == 0 {
		filter.Limit = 50 // Default page size
	}
	if filter.SortBy == "" {
		filter.SortBy = "popularity" // Default sort
	}
	if filter.SortOrder == "" {
		filter.SortOrder = "desc" // Most popular first
	}

	// Build search URL with parameters
	searchURL, err := r.buildSearchURL(filter)
	if err != nil {
		return nil, fmt.Errorf("failed to build search URL: %w", err)
	}

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, "GET", searchURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create search request: %w", err)
	}

	// Add authentication if required
	if err := r.addAuth(req); err != nil {
		return nil, fmt.Errorf("failed to add authentication: %w", err)
	}

	// Execute request
	startTime := time.Now()
	resp, err := r.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute search request: %w", err)
	}
	defer resp.Body.Close()

	// Handle response
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("registry returned error status: %d", resp.StatusCode)
	}

	// Parse response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var result MarketplaceSearchResult
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse search response: %w", err)
	}

	result.ExecutionTime = time.Since(startTime)
	result.Query = filter

	return &result, nil
}

// GetTemplate retrieves a specific template from the registry
func (r *MarketplaceTemplateRegistry) GetTemplate(ctx context.Context, name, version string) (*MarketplaceTemplateRegistryEntry, error) {
	templateURL := fmt.Sprintf("%s/api/v1/templates/%s", r.URL, url.PathEscape(name))
	if version != "" {
		templateURL += "?version=" + url.QueryEscape(version)
	}

	req, err := http.NewRequestWithContext(ctx, "GET", templateURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	if err := r.addAuth(req); err != nil {
		return nil, fmt.Errorf("failed to add authentication: %w", err)
	}

	resp, err := r.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch template: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("template not found: %s", name)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("registry returned error status: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var entry MarketplaceTemplateRegistryEntry
	if err := json.Unmarshal(body, &entry); err != nil {
		return nil, fmt.Errorf("failed to parse template response: %w", err)
	}

	return &entry, nil
}

// ListCategories returns available template categories in the registry
func (r *MarketplaceTemplateRegistry) ListCategories(ctx context.Context) ([]string, error) {
	categoriesURL := fmt.Sprintf("%s/api/v1/categories", r.URL)

	req, err := http.NewRequestWithContext(ctx, "GET", categoriesURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	if err := r.addAuth(req); err != nil {
		return nil, fmt.Errorf("failed to add authentication: %w", err)
	}

	resp, err := r.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch categories: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("registry returned error status: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var categories []string
	if err := json.Unmarshal(body, &categories); err != nil {
		return nil, fmt.Errorf("failed to parse categories response: %w", err)
	}

	return categories, nil
}

// PublishTemplate uploads a template to the registry (for writable registries)
func (r *MarketplaceTemplateRegistry) PublishTemplate(ctx context.Context, template *Template) error {
	if r.Type == RegistryTypeOfficial {
		return fmt.Errorf("cannot publish to official registry")
	}

	publishURL := fmt.Sprintf("%s/api/v1/templates", r.URL)

	templateJSON, err := json.Marshal(template)
	if err != nil {
		return fmt.Errorf("failed to marshal template: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", publishURL, strings.NewReader(string(templateJSON)))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	if err := r.addAuth(req); err != nil {
		return fmt.Errorf("failed to add authentication: %w", err)
	}

	resp, err := r.Client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to publish template: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("registry returned error status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// buildSearchURL constructs the search API URL with filters
func (r *MarketplaceTemplateRegistry) buildSearchURL(filter SearchFilter) (string, error) {
	baseURL, err := url.Parse(fmt.Sprintf("%s/api/v1/search", r.URL))
	if err != nil {
		return "", err
	}

	params := url.Values{}

	// Add text search parameters
	if filter.Query != "" {
		params.Add("q", filter.Query)
	}
	if len(filter.Keywords) > 0 {
		params.Add("keywords", strings.Join(filter.Keywords, ","))
	}

	// Add classification filters
	if len(filter.Categories) > 0 {
		params.Add("categories", strings.Join(filter.Categories, ","))
	}
	if len(filter.Domains) > 0 {
		params.Add("domains", strings.Join(filter.Domains, ","))
	}
	if len(filter.Complexity) > 0 {
		complexityStrs := make([]string, len(filter.Complexity))
		for i, c := range filter.Complexity {
			complexityStrs[i] = string(c)
		}
		params.Add("complexity", strings.Join(complexityStrs, ","))
	}

	// Add quality filters
	if filter.MinRating > 0 {
		params.Add("min_rating", fmt.Sprintf("%.1f", filter.MinRating))
	}
	if filter.VerifiedOnly {
		params.Add("verified", "true")
	}
	if filter.ValidatedOnly {
		params.Add("validated", "true")
	}

	// Add feature filters
	if filter.ResearchUserSupport {
		params.Add("research_user", "true")
	}
	if len(filter.ConnectionTypes) > 0 {
		params.Add("connection_types", strings.Join(filter.ConnectionTypes, ","))
	}
	if len(filter.PackageManagers) > 0 {
		params.Add("package_managers", strings.Join(filter.PackageManagers, ","))
	}

	// Add sorting and pagination
	params.Add("sort_by", filter.SortBy)
	params.Add("sort_order", filter.SortOrder)
	params.Add("limit", fmt.Sprintf("%d", filter.Limit))
	params.Add("offset", fmt.Sprintf("%d", filter.Offset))

	baseURL.RawQuery = params.Encode()
	return baseURL.String(), nil
}

// addAuth adds authentication headers to the request
func (r *MarketplaceTemplateRegistry) addAuth(req *http.Request) error {
	if r.Credentials == nil {
		return nil // No authentication required
	}

	switch r.Credentials.Type {
	case "token":
		req.Header.Set("Authorization", "Bearer "+r.Credentials.Token)
	case "basic":
		req.SetBasicAuth(r.Credentials.Username, r.Credentials.Password)
	default:
		return fmt.Errorf("unsupported authentication type: %s", r.Credentials.Type)
	}

	return nil
}

// MarketplaceTemplateRegistryManager manages multiple template registries
type MarketplaceTemplateRegistryManager struct {
	registries      map[string]*MarketplaceTemplateRegistry
	defaultRegistry string
}

// NewMarketplaceTemplateRegistryManager creates a new registry manager
func NewMarketplaceTemplateRegistryManager() *MarketplaceTemplateRegistryManager {
	return &MarketplaceTemplateRegistryManager{
		registries: make(map[string]*MarketplaceTemplateRegistry),
	}
}

// AddRegistry adds a registry to the manager
func (m *MarketplaceTemplateRegistryManager) AddRegistry(registry *MarketplaceTemplateRegistry) {
	m.registries[registry.Name] = registry

	// Set first official registry as default
	if m.defaultRegistry == "" && registry.Type == RegistryTypeOfficial {
		m.defaultRegistry = registry.Name
	}
	// Set first registry as default if no official registry exists
	if m.defaultRegistry == "" {
		m.defaultRegistry = registry.Name
	}
}

// SearchAll searches across all configured registries
func (m *MarketplaceTemplateRegistryManager) SearchAll(ctx context.Context, filter SearchFilter) (*MarketplaceSearchResult, error) {
	if len(m.registries) == 0 {
		return m.emptySearchResult(), nil
	}

	startTime := time.Now()
	allResults := make([]MarketplaceTemplateRegistryEntry, 0)
	totalCount := 0

	// Search each matching registry
	for _, registry := range m.registries {
		if !m.shouldSearchRegistry(registry, filter) {
			continue
		}

		results, count := m.searchRegistry(ctx, registry, filter)
		allResults = append(allResults, results...)
		totalCount += count
	}

	// Sort and paginate combined results
	m.sortResults(allResults, filter.SortBy, filter.SortOrder)
	paginatedResults := m.paginateResults(allResults, filter)

	return &MarketplaceSearchResult{
		Templates:     paginatedResults,
		TotalCount:    totalCount,
		FilteredCount: len(allResults),
		Query:         filter,
		ExecutionTime: time.Since(startTime),
	}, nil
}

// emptySearchResult returns an empty search result
func (m *MarketplaceTemplateRegistryManager) emptySearchResult() *MarketplaceSearchResult {
	return &MarketplaceSearchResult{
		Templates:  []MarketplaceTemplateRegistryEntry{},
		TotalCount: 0,
	}
}

// shouldSearchRegistry checks if registry matches filter criteria
func (m *MarketplaceTemplateRegistryManager) shouldSearchRegistry(registry *MarketplaceTemplateRegistry, filter SearchFilter) bool {
	// Check registry name filter
	if len(filter.Registries) > 0 && !m.registryNameMatches(registry.Name, filter.Registries) {
		return false
	}

	// Check registry type filter
	if len(filter.RegistryTypes) > 0 && !m.registryTypeMatches(registry.Type, filter.RegistryTypes) {
		return false
	}

	return true
}

// registryNameMatches checks if registry name is in the filter list
func (m *MarketplaceTemplateRegistryManager) registryNameMatches(name string, allowedNames []string) bool {
	for _, allowedName := range allowedNames {
		if name == allowedName {
			return true
		}
	}
	return false
}

// registryTypeMatches checks if registry type is in the filter list
func (m *MarketplaceTemplateRegistryManager) registryTypeMatches(registryType RegistryType, allowedTypes []RegistryType) bool {
	for _, allowedType := range allowedTypes {
		if registryType == allowedType {
			return true
		}
	}
	return false
}

// searchRegistry executes search on a single registry
func (m *MarketplaceTemplateRegistryManager) searchRegistry(ctx context.Context, registry *MarketplaceTemplateRegistry, filter SearchFilter) ([]MarketplaceTemplateRegistryEntry, int) {
	result, err := registry.Search(ctx, filter)
	if err != nil {
		// Log error but continue with other registries
		return nil, 0
	}

	// Add registry metadata to results
	for i := range result.Templates {
		result.Templates[i].RegistryName = registry.Name
		result.Templates[i].RegistryType = string(registry.Type)
	}

	return result.Templates, result.TotalCount
}

// paginateResults applies pagination to search results
func (m *MarketplaceTemplateRegistryManager) paginateResults(results []MarketplaceTemplateRegistryEntry, filter SearchFilter) []MarketplaceTemplateRegistryEntry {
	offset := filter.Offset
	limit := filter.Limit
	if limit == 0 {
		limit = 50
	}

	if offset >= len(results) {
		return []MarketplaceTemplateRegistryEntry{}
	}

	end := offset + limit
	if end > len(results) {
		end = len(results)
	}

	return results[offset:end]
}

// GetRegistry returns a specific registry by name
func (m *MarketplaceTemplateRegistryManager) GetRegistry(name string) (*MarketplaceTemplateRegistry, bool) {
	registry, exists := m.registries[name]
	return registry, exists
}

// ListRegistries returns all configured registries
func (m *MarketplaceTemplateRegistryManager) ListRegistries() map[string]*MarketplaceTemplateRegistry {
	return m.registries
}

// sortResults sorts template results by the specified criteria
func (m *MarketplaceTemplateRegistryManager) sortResults(results []MarketplaceTemplateRegistryEntry, sortBy, sortOrder string) {
	switch sortBy {
	case "popularity":
		sort.Slice(results, func(i, j int) bool {
			if sortOrder == "asc" {
				return results[i].PopularityRank < results[j].PopularityRank
			}
			return results[i].PopularityRank > results[j].PopularityRank
		})
	case "rating":
		sort.Slice(results, func(i, j int) bool {
			iRating := float64(0)
			jRating := float64(0)
			if results[i].Marketplace != nil {
				iRating = results[i].Marketplace.Rating
			}
			if results[j].Marketplace != nil {
				jRating = results[j].Marketplace.Rating
			}

			if sortOrder == "asc" {
				return iRating < jRating
			}
			return iRating > jRating
		})
	case "updated":
		sort.Slice(results, func(i, j int) bool {
			if sortOrder == "asc" {
				return results[i].LastUpdated.Before(results[j].LastUpdated)
			}
			return results[i].LastUpdated.After(results[j].LastUpdated)
		})
	case "name":
		sort.Slice(results, func(i, j int) bool {
			if sortOrder == "asc" {
				return results[i].Name < results[j].Name
			}
			return results[i].Name > results[j].Name
		})
	}
}
