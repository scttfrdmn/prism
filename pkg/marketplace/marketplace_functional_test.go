// Package marketplace provides comprehensive functional tests for community template marketplace
package marketplace

import (
	"testing"
	"time"
)

// TestMarketplaceRegistryFunctionalWorkflow validates complete marketplace functionality
func TestMarketplaceRegistryFunctionalWorkflow(t *testing.T) {
	registry := setupMarketplaceRegistry(t)

	// Test complete marketplace workflow
	testMarketplaceRegistryCreation(t, registry)
	testTemplateManagement(t, registry)
	testTemplateSearch(t, registry)
	testTemplateCategories(t, registry)
	testFeaturedTemplates(t, registry)
	testTemplateStatistics(t, registry)

	t.Log("✅ Marketplace registry functional workflow validated")
}

// setupMarketplaceRegistry creates and configures a marketplace registry for testing
func setupMarketplaceRegistry(t *testing.T) *Registry {
	config := createTestMarketplaceConfig()
	registry := NewRegistry(config)

	if registry == nil {
		t.Fatal("Failed to create marketplace registry")
	}

	// Populate with test templates
	populateTestTemplates(t, registry)

	return registry
}

// createTestMarketplaceConfig creates a test marketplace configuration
func createTestMarketplaceConfig() *MarketplaceConfig {
	return &MarketplaceConfig{
		Enabled:         true,
		CacheTimeout:    time.Hour,
		MaxTemplates:    1000,
		FeaturedCount:   5,
		DefaultCategory: "general",
		SortOptions:     []string{"popularity", "created", "updated", "name"},
		MinRating:       1.0,
		MaxRating:       5.0,
	}
}

// populateTestTemplates adds test templates to the registry
func populateTestTemplates(t *testing.T, registry *Registry) {
	testTemplates := createTestTemplates()

	for _, template := range testTemplates {
		err := registry.PublishTemplate(template)
		if err != nil {
			t.Errorf("Failed to publish test template %s: %v", template.ID, err)
		}
	}
}

// createTestTemplates creates a comprehensive set of test templates
func createTestTemplates() []*CommunityTemplate {
	now := time.Now()
	return []*CommunityTemplate{
		{
			ID:          "python-data-science",
			Name:        "Python Data Science Stack",
			Description: "Complete Python environment for data science with Jupyter, pandas, numpy, and scikit-learn",
			Author: &Author{
				ID:       "data-scientist-1",
				Name:     "Dr. Data Scientist",
				Email:    "data@example.com",
				Verified: true,
			},
			Version:   "1.2.0",
			Category:  "data-science",
			Tags:      []string{"python", "jupyter", "pandas", "machine-learning"},
			Rating:    4.8,
			Downloads: 15420,
			CreatedAt: now.Add(-30 * 24 * time.Hour),
			UpdatedAt: now.Add(-5 * 24 * time.Hour),
			Featured:  true,
			Verified:  true,
			Status:    TemplateStatusActive,
		},
		{
			ID:          "r-statistical-analysis",
			Name:        "R Statistical Analysis",
			Description: "R environment with RStudio, tidyverse, and statistical packages",
			Author: &Author{
				ID:       "statistician-1",
				Name:     "Prof. R. Statistician",
				Email:    "stats@example.com",
				Verified: true,
			},
			Version:   "2.1.3",
			Category:  "data-science",
			Tags:      []string{"r", "rstudio", "statistics", "tidyverse"},
			Rating:    4.6,
			Downloads: 8730,
			CreatedAt: now.Add(-45 * 24 * time.Hour),
			UpdatedAt: now.Add(-10 * 24 * time.Hour),
			Featured:  true,
			Verified:  true,
			Status:    TemplateStatusActive,
		},
		{
			ID:          "web-dev-fullstack",
			Name:        "Full-Stack Web Development",
			Description: "Complete web development stack with Node.js, React, and PostgreSQL",
			Author: &Author{
				ID:       "webdev-1",
				Name:     "Jane Developer",
				Email:    "webdev@example.com",
				Verified: false,
			},
			Version:   "3.0.1",
			Category:  "web-development",
			Tags:      []string{"nodejs", "react", "postgresql", "fullstack"},
			Rating:    4.4,
			Downloads: 12350,
			CreatedAt: now.Add(-20 * 24 * time.Hour),
			UpdatedAt: now.Add(-2 * 24 * time.Hour),
			Featured:  false,
			Verified:  false,
			Status:    TemplateStatusActive,
		},
		{
			ID:          "machine-learning-gpu",
			Name:        "GPU Machine Learning",
			Description: "CUDA-enabled environment with TensorFlow, PyTorch, and GPU acceleration",
			Author: &Author{
				ID:       "ml-engineer-1",
				Name:     "Alex ML Engineer",
				Email:    "ml@example.com",
				Verified: true,
			},
			Version:   "1.5.2",
			Category:  "machine-learning",
			Tags:      []string{"gpu", "cuda", "tensorflow", "pytorch", "deep-learning"},
			Rating:    4.9,
			Downloads: 25680,
			CreatedAt: now.Add(-60 * 24 * time.Hour),
			UpdatedAt: now.Add(-1 * 24 * time.Hour),
			Featured:  true,
			Verified:  true,
			Status:    TemplateStatusActive,
		},
		{
			ID:          "bioinformatics-toolkit",
			Name:        "Bioinformatics Toolkit",
			Description: "Specialized tools for genomics, proteomics, and bioinformatics research",
			Author: &Author{
				ID:       "bioinformatician-1",
				Name:     "Dr. Bio Research",
				Email:    "bio@example.com",
				Verified: true,
			},
			Version:   "2.3.1",
			Category:  "research",
			Tags:      []string{"bioinformatics", "genomics", "proteomics", "blast"},
			Rating:    4.3,
			Downloads: 3420,
			CreatedAt: now.Add(-90 * 24 * time.Hour),
			UpdatedAt: now.Add(-15 * 24 * time.Hour),
			Featured:  false,
			Verified:  true,
			Status:    TemplateStatusActive,
		},
	}
}

// testMarketplaceRegistryCreation validates registry initialization
func testMarketplaceRegistryCreation(t *testing.T, registry *Registry) {
	if registry.config == nil {
		t.Error("Registry config should be initialized")
	}

	if registry.templateCache == nil {
		t.Error("Registry template cache should be initialized")
	}

	if registry.categories == nil {
		t.Error("Registry categories should be initialized")
	}

	if len(registry.categories) == 0 {
		t.Error("Registry should have default categories")
	}

	if registry.lastSync.IsZero() {
		t.Error("Registry last sync time should be set")
	}

	t.Log("Marketplace registry creation validated")
}

// testTemplateManagement validates template publishing, updating, and removal
func testTemplateManagement(t *testing.T, registry *Registry) {
	// Test template publishing
	newTemplate := &CommunityTemplate{
		ID:          "test-template-management",
		Name:        "Test Template",
		Description: "A template for testing management operations",
		Author: &Author{
			ID:       "test-author",
			Name:     "Test Author",
			Email:    "test@example.com",
			Verified: false,
		},
		Version:   "1.0.0",
		Category:  "testing",
		Tags:      []string{"test", "management"},
		Rating:    4.0,
		Downloads: 0,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Status:    TemplateStatusActive,
	}

	err := registry.PublishTemplate(newTemplate)
	if err != nil {
		t.Errorf("Failed to publish template: %v", err)
	}

	// Verify template was added
	template, err := registry.GetTemplate(newTemplate.ID)
	if err != nil {
		t.Errorf("Failed to retrieve published template: %v", err)
	}

	if template.ID != newTemplate.ID {
		t.Errorf("Retrieved template ID mismatch: expected %s, got %s", newTemplate.ID, template.ID)
	}

	// Test template updating
	updatedTemplate := *newTemplate
	updatedTemplate.Version = "1.1.0"
	updatedTemplate.Description = "Updated test template"

	err = registry.UpdateTemplate(&updatedTemplate)
	if err != nil {
		t.Errorf("Failed to update template: %v", err)
	}

	// Verify template was updated
	template, err = registry.GetTemplate(newTemplate.ID)
	if err != nil {
		t.Errorf("Failed to retrieve updated template: %v", err)
	}

	if template.Version != "1.1.0" {
		t.Errorf("Template version not updated: expected 1.1.0, got %s", template.Version)
	}

	// Test template removal
	err = registry.RemoveTemplate(newTemplate.ID)
	if err != nil {
		t.Errorf("Failed to remove template: %v", err)
	}

	// Verify template was removed
	_, err = registry.GetTemplate(newTemplate.ID)
	if err == nil {
		t.Error("Template should not exist after removal")
	}

	t.Log("Template management validated")
}

// testTemplateSearch validates search functionality
func testTemplateSearch(t *testing.T, registry *Registry) {
	testCases := []struct {
		name          string
		query         SearchQuery
		expectedCount int
		expectedFirst string
		description   string
	}{
		{
			name: "search_by_keyword",
			query: SearchQuery{
				Keywords: "python",
				Limit:    10,
			},
			expectedCount: 2,  // python-data-science and machine-learning-gpu (has python in tags)
			expectedFirst: "", // Order may vary
			description:   "Search by keyword 'python'",
		},
		{
			name: "search_by_category",
			query: SearchQuery{
				Category: "data-science",
				Limit:    10,
			},
			expectedCount: 2, // python-data-science and r-statistical-analysis
			expectedFirst: "",
			description:   "Search by category 'data-science'",
		},
		{
			name: "search_by_author",
			query: SearchQuery{
				Author: "Dr. Data Scientist",
				Limit:  10,
			},
			expectedCount: 1, // python-data-science
			expectedFirst: "python-data-science",
			description:   "Search by author name",
		},
		{
			name: "search_verified_only",
			query: SearchQuery{
				VerifiedOnly: true,
				Limit:        10,
			},
			expectedCount: 4, // All except web-dev-fullstack
			expectedFirst: "",
			description:   "Search verified templates only",
		},
		{
			name: "search_featured_only",
			query: SearchQuery{
				FeaturedOnly: true,
				Limit:        10,
			},
			expectedCount: 3, // python-data-science, r-statistical-analysis, machine-learning-gpu
			expectedFirst: "",
			description:   "Search featured templates only",
		},
		{
			name: "search_with_tag_filter",
			query: SearchQuery{
				Tags:  []string{"jupyter"},
				Limit: 10,
			},
			expectedCount: 1, // python-data-science
			expectedFirst: "python-data-science",
			description:   "Search by specific tag",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			results, err := registry.SearchTemplates(tc.query)
			if err != nil {
				t.Errorf("Search failed: %v", err)
				return
			}

			if len(results) != tc.expectedCount {
				t.Errorf("%s: expected %d results, got %d", tc.description, tc.expectedCount, len(results))
			}

			if tc.expectedFirst != "" && len(results) > 0 {
				if results[0].ID != tc.expectedFirst {
					t.Errorf("%s: expected first result %s, got %s", tc.description, tc.expectedFirst, results[0].ID)
				}
			}
		})
	}

	t.Log("Template search validated")
}

// testTemplateCategories validates category management
func testTemplateCategories(t *testing.T, registry *Registry) {
	categories := registry.GetCategories()

	if len(categories) == 0 {
		t.Error("Registry should have categories")
	}

	// Verify default categories exist
	expectedCategories := []string{"data-science", "web-development", "machine-learning", "research"}
	categoryMap := make(map[string]bool)

	for _, cat := range categories {
		categoryMap[cat.ID] = true
	}

	for _, expected := range expectedCategories {
		if !categoryMap[expected] {
			t.Errorf("Expected category %s not found", expected)
		}
	}

	// Test category statistics
	stats := registry.GetCategoryStats()
	if len(stats) == 0 {
		t.Error("Category statistics should be available")
	}

	for _, stat := range stats {
		if stat.Count < 0 {
			t.Errorf("Category %s has negative count: %d", stat.Category.ID, stat.Count)
		}
	}

	t.Log("Template categories validated")
}

// testFeaturedTemplates validates featured template management
func testFeaturedTemplates(t *testing.T, registry *Registry) {
	featured := registry.GetFeaturedTemplates()

	if len(featured) == 0 {
		t.Error("Should have featured templates")
	}

	// Verify featured templates are actually marked as featured
	for _, template := range featured {
		if !template.Featured {
			t.Errorf("Template %s in featured list but not marked as featured", template.ID)
		}
	}

	// Verify featured templates are sorted by relevance (rating * downloads)
	if len(featured) > 1 {
		for i := 1; i < len(featured); i++ {
			prev := featured[i-1]
			current := featured[i]

			prevScore := prev.Rating * float64(prev.Downloads)
			currentScore := current.Rating * float64(current.Downloads)

			if currentScore > prevScore {
				t.Errorf("Featured templates not properly sorted: %s (%.2f) should be before %s (%.2f)",
					current.ID, currentScore, prev.ID, prevScore)
			}
		}
	}

	t.Log("Featured templates validated")
}

// testTemplateStatistics validates statistics generation
func testTemplateStatistics(t *testing.T, registry *Registry) {
	stats := registry.GetMarketplaceStats()

	if stats.TotalTemplates <= 0 {
		t.Error("Should have templates in statistics")
	}

	if stats.TotalDownloads <= 0 {
		t.Error("Should have downloads in statistics")
	}

	if stats.AverageRating <= 0 || stats.AverageRating > 5 {
		t.Errorf("Average rating should be between 0 and 5, got %f", stats.AverageRating)
	}

	if len(stats.TopCategories) == 0 {
		t.Error("Should have top categories in statistics")
	}

	if len(stats.TopAuthors) == 0 {
		t.Error("Should have top authors in statistics")
	}

	// Validate category breakdown
	totalByCategory := 0
	for _, count := range stats.CategoryBreakdown {
		totalByCategory += count
	}

	if totalByCategory != stats.TotalTemplates {
		t.Errorf("Category breakdown sum (%d) doesn't match total templates (%d)",
			totalByCategory, stats.TotalTemplates)
	}

	t.Log("Template statistics validated")
}

// TestMarketplaceRegistryConcurrency validates thread-safe operations
func TestMarketplaceRegistryConcurrency(t *testing.T) {
	registry := setupMarketplaceRegistry(t)
	done := make(chan bool, 3)

	// Concurrent searches
	go func() {
		for i := 0; i < 50; i++ {
			query := SearchQuery{
				Keywords: "test",
				Limit:    10,
			}
			registry.SearchTemplates(query)
		}
		done <- true
	}()

	// Concurrent template retrieval
	go func() {
		templateIDs := []string{"python-data-science", "r-statistical-analysis", "web-dev-fullstack"}
		for i := 0; i < 50; i++ {
			for _, id := range templateIDs {
				registry.GetTemplate(id)
			}
		}
		done <- true
	}()

	// Concurrent statistics generation
	go func() {
		for i := 0; i < 25; i++ {
			registry.GetMarketplaceStats()
			registry.GetFeaturedTemplates()
			registry.GetCategories()
		}
		done <- true
	}()

	// Wait for all goroutines to complete
	for i := 0; i < 3; i++ {
		select {
		case <-done:
			// Success
		case <-time.After(10 * time.Second):
			t.Error("Concurrent operations timed out")
		}
	}

	t.Log("✅ Marketplace registry concurrency validated")
}

// TestMarketplaceTemplateValidation validates template validation logic
func TestMarketplaceTemplateValidation(t *testing.T) {
	registry := setupMarketplaceRegistry(t)

	testCases := []struct {
		name        string
		template    *CommunityTemplate
		expectError bool
		description string
	}{
		{
			name: "valid_template",
			template: &CommunityTemplate{
				ID:          "valid-template",
				Name:        "Valid Template",
				Description: "A valid template for testing",
				Author: &Author{
					ID:    "valid-author",
					Name:  "Valid Author",
					Email: "valid@example.com",
				},
				Version:  "1.0.0",
				Category: "testing",
				Tags:     []string{"valid", "test"},
				Rating:   4.5,
				Status:   TemplateStatusActive,
			},
			expectError: false,
			description: "Valid template should pass validation",
		},
		{
			name: "empty_id",
			template: &CommunityTemplate{
				ID:          "",
				Name:        "Template with Empty ID",
				Description: "Template missing ID",
				Author: &Author{
					ID:    "author",
					Name:  "Author",
					Email: "author@example.com",
				},
				Version:  "1.0.0",
				Category: "testing",
				Status:   TemplateStatusActive,
			},
			expectError: true,
			description: "Template with empty ID should fail validation",
		},
		{
			name: "invalid_rating",
			template: &CommunityTemplate{
				ID:          "invalid-rating",
				Name:        "Invalid Rating Template",
				Description: "Template with invalid rating",
				Author: &Author{
					ID:    "author",
					Name:  "Author",
					Email: "author@example.com",
				},
				Version:  "1.0.0",
				Category: "testing",
				Rating:   6.0, // Invalid: > 5.0
				Status:   TemplateStatusActive,
			},
			expectError: true,
			description: "Template with rating > 5.0 should fail validation",
		},
		{
			name: "empty_author",
			template: &CommunityTemplate{
				ID:          "no-author",
				Name:        "No Author Template",
				Description: "Template without author",
				Author:      nil,
				Version:     "1.0.0",
				Category:    "testing",
				Rating:      4.0,
				Status:      TemplateStatusActive,
			},
			expectError: true,
			description: "Template without author should fail validation",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := registry.PublishTemplate(tc.template)

			if tc.expectError && err == nil {
				t.Errorf("%s: expected error but got none", tc.description)
			}

			if !tc.expectError && err != nil {
				t.Errorf("%s: unexpected error: %v", tc.description, err)
			}
		})
	}

	t.Log("✅ Marketplace template validation tested")
}
