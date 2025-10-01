// Package marketplace provides the marketplace registry implementation
package marketplace

import (
	"fmt"
	"sort"
	"strings"
	"time"
)

// Registry implements the MarketplaceRegistry interface
type Registry struct {
	config        *MarketplaceConfig
	templateCache map[string]*CommunityTemplate
	categories    []TemplateCategory
	featured      []*CommunityTemplate
	lastSync      time.Time
}

// NewRegistry creates a new marketplace registry
func NewRegistry(config *MarketplaceConfig) *Registry {
	return &Registry{
		config:        config,
		templateCache: make(map[string]*CommunityTemplate),
		categories:    DefaultCategories(),
		featured:      make([]*CommunityTemplate, 0),
		lastSync:      time.Now(),
	}
}

// SearchTemplates searches for templates in the marketplace
func (r *Registry) SearchTemplates(query SearchQuery) ([]*CommunityTemplate, error) {
	// For now, implement in-memory search
	// In production, this would query DynamoDB with proper indexing
	var results []*CommunityTemplate

	for _, template := range r.templateCache {
		if r.matchesQuery(template, query) {
			results = append(results, template)
		}
	}

	// Sort results
	r.sortResults(results, query.SortBy, query.SortOrder)

	// Apply pagination
	if query.Offset > 0 && query.Offset < len(results) {
		results = results[query.Offset:]
	}
	if query.Limit > 0 && query.Limit < len(results) {
		results = results[:query.Limit]
	}

	return results, nil
}

// GetTemplate retrieves a specific template by ID
func (r *Registry) GetTemplate(templateID string) (*CommunityTemplate, error) {
	template, exists := r.templateCache[templateID]
	if !exists {
		// In production, this would fetch from DynamoDB
		return nil, fmt.Errorf("template not found: %s", templateID)
	}

	// Track access for analytics
	r.trackUsage(templateID, &UsageEvent{
		EventType:  "view",
		TemplateID: templateID,
		Timestamp:  time.Now(),
	})

	return template, nil
}

// ListCategories returns available template categories
func (r *Registry) ListCategories() ([]TemplateCategory, error) {
	// Update template counts
	counts := make(map[string]int)
	for _, template := range r.templateCache {
		counts[template.Category]++
	}

	for i := range r.categories {
		if count, exists := counts[r.categories[i].ID]; exists {
			r.categories[i].TemplateCount = count
		}
	}

	return r.categories, nil
}

// GetFeatured returns featured templates
func (r *Registry) GetFeatured() ([]*CommunityTemplate, error) {
	return r.featured, nil
}

// GetTrending returns trending templates for a specific timeframe
func (r *Registry) GetTrending(timeframe string) ([]*CommunityTemplate, error) {
	// For now, return based on recent downloads and high ratings
	var trending []*CommunityTemplate

	for _, template := range r.templateCache {
		// Simple trending algorithm: high rating + recent activity
		trendingScore := template.Rating * float64(template.DownloadCount) / 100
		if trendingScore > 10.0 {
			trending = append(trending, template)
		}
	}

	// Sort by trending score (downloads * rating)
	sort.Slice(trending, func(i, j int) bool {
		scoreI := trending[i].Rating * float64(trending[i].DownloadCount)
		scoreJ := trending[j].Rating * float64(trending[j].DownloadCount)
		return scoreI > scoreJ
	})

	// Return top 20 trending
	if len(trending) > 20 {
		trending = trending[:20]
	}

	return trending, nil
}

// PublishTemplate publishes a template to the marketplace
func (r *Registry) PublishTemplate(template *TemplatePublication) (*PublicationResult, error) {
	// Generate unique template ID
	templateID := r.generateTemplateID(template.Name)

	// Create community template from publication
	communityTemplate, err := r.createCommunityTemplate(templateID, template)
	if err != nil {
		return nil, fmt.Errorf("failed to create community template: %w", err)
	}

	// Store in cache (in production, this would save to DynamoDB)
	r.templateCache[templateID] = communityTemplate

	// Create publication result
	result := &PublicationResult{
		TemplateID:     templateID,
		PublicationURL: fmt.Sprintf("%s/templates/%s", r.config.CDNEndpoint, templateID),
		Status:         "published",
		Message:        "Template published successfully",
		CreatedAt:      time.Now(),
	}

	// Handle AMI generation if requested
	if template.GenerateAMI {
		result.AMICreationIDs = r.initiateAMIGeneration(templateID, template.TargetRegions)
	}

	return result, nil
}

// UpdateTemplate updates a published template
func (r *Registry) UpdateTemplate(templateID string, update *TemplateUpdate) error {
	template, exists := r.templateCache[templateID]
	if !exists {
		return fmt.Errorf("template not found: %s", templateID)
	}

	// Apply updates
	if update.Name != "" {
		template.Name = update.Name
	}
	if update.Description != "" {
		template.Description = update.Description
	}
	if update.Documentation != "" {
		template.Documentation = update.Documentation
	}
	if len(update.Tags) > 0 {
		template.Tags = update.Tags
	}
	if len(update.Keywords) > 0 {
		template.Keywords = update.Keywords
	}
	if len(update.Screenshots) > 0 {
		template.Screenshots = update.Screenshots
	}
	if update.VideoDemo != "" {
		template.VideoDemo = update.VideoDemo
	}
	if update.Version != "" {
		template.Version = update.Version
	}

	template.UpdatedAt = time.Now()

	// In production, this would update DynamoDB
	r.templateCache[templateID] = template

	return nil
}

// UnpublishTemplate removes a template from the marketplace
func (r *Registry) UnpublishTemplate(templateID string) error {
	if _, exists := r.templateCache[templateID]; !exists {
		return fmt.Errorf("template not found: %s", templateID)
	}

	// Remove from cache (in production, this would mark as unpublished in DynamoDB)
	delete(r.templateCache, templateID)

	return nil
}

// GetUserPublications returns templates published by a specific user
func (r *Registry) GetUserPublications(userID string) ([]*CommunityTemplate, error) {
	var publications []*CommunityTemplate

	for _, template := range r.templateCache {
		if template.Author == userID {
			publications = append(publications, template)
		}
	}

	// Sort by creation date (newest first)
	sort.Slice(publications, func(i, j int) bool {
		return publications[i].CreatedAt.After(publications[j].CreatedAt)
	})

	return publications, nil
}

// AddReview adds a review for a template
func (r *Registry) AddReview(templateID string, review *TemplateReview) error {
	template, exists := r.templateCache[templateID]
	if !exists {
		return fmt.Errorf("template not found: %s", templateID)
	}

	// In production, this would store in DynamoDB reviews table
	// For now, just update aggregate metrics
	r.updateRatingMetrics(template, review.Rating)

	return nil
}

// GetReviews retrieves reviews for a template with pagination
func (r *Registry) GetReviews(templateID string, pagination *ReviewPagination) (*ReviewResponse, error) {
	// In production, this would query DynamoDB reviews table
	// For now, return mock reviews
	mockReviews := r.generateMockReviews(templateID)

	// Apply pagination
	start := 0
	if pagination.Offset > 0 {
		start = pagination.Offset
	}

	limit := 10 // Default limit
	if pagination.Limit > 0 {
		limit = pagination.Limit
	}

	end := start + limit
	if end > len(mockReviews) {
		end = len(mockReviews)
	}

	paginatedReviews := mockReviews[start:end]

	response := &ReviewResponse{
		Reviews:    paginatedReviews,
		TotalCount: len(mockReviews),
		Page:       (start / limit) + 1,
		TotalPages: (len(mockReviews) + limit - 1) / limit,
		HasMore:    end < len(mockReviews),
	}

	return response, nil
}

// TrackUsage tracks usage events for analytics
func (r *Registry) TrackUsage(templateID string, event *UsageEvent) error {
	// In production, this would write to analytics storage
	r.trackUsage(templateID, event)
	return nil
}

// ForkTemplate creates a fork of an existing template
func (r *Registry) ForkTemplate(templateID string, fork *TemplateFork) (*CommunityTemplate, error) {
	originalTemplate, exists := r.templateCache[templateID]
	if !exists {
		return nil, fmt.Errorf("template not found: %s", templateID)
	}

	// Create new template ID for the fork
	forkID := r.generateTemplateID(fork.NewName)

	// Create forked template
	forkedTemplate := &CommunityTemplate{
		TemplateID:        forkID,
		Name:              fork.NewName,
		Description:       fork.NewDescription,
		Author:            "current-user", // In production, this would be the authenticated user
		Version:           "1.0.0",
		CreatedAt:         time.Now(),
		UpdatedAt:         time.Now(),
		Category:          originalTemplate.Category,
		Tags:              originalTemplate.Tags,
		Keywords:          originalTemplate.Keywords,
		ResearchDomain:    originalTemplate.ResearchDomain,
		Architecture:      originalTemplate.Architecture,
		SupportedRegions:  originalTemplate.SupportedRegions,
		RequiredResources: originalTemplate.RequiredResources,
		EstimatedCost:     originalTemplate.EstimatedCost,
		Template:          originalTemplate.Template, // Copy template definition
		Publication: &PublicationMetadata{
			License:    originalTemplate.Publication.License,
			Visibility: "private", // Forks start as private
		},
	}

	// Store forked template
	r.templateCache[forkID] = forkedTemplate

	// Update fork count on original template
	originalTemplate.ForkCount++

	return forkedTemplate, nil
}

// GetTemplateAnalytics returns comprehensive analytics for a template
func (r *Registry) GetTemplateAnalytics(templateID string) (*TemplateAnalytics, error) {
	template, exists := r.templateCache[templateID]
	if !exists {
		return nil, fmt.Errorf("template not found: %s", templateID)
	}

	// In production, this would aggregate from analytics storage
	analytics := &TemplateAnalytics{
		TemplateID:        templateID,
		TotalDownloads:    template.DownloadCount,
		TotalLaunches:     template.LaunchCount,
		SuccessRate:       0.85, // Mock success rate
		AverageRating:     template.Rating,
		TotalReviews:      template.ReviewCount,
		TotalForks:        template.ForkCount,
		AverageLaunchTime: time.Duration(45) * time.Second,
		RegionUsage: map[string]int{
			"us-east-1": template.LaunchCount / 2,
			"us-west-2": template.LaunchCount / 3,
			"eu-west-1": template.LaunchCount / 4,
		},
		ArchitectureUsage: map[string]int{
			"x86_64": int(float64(template.LaunchCount) * 0.7),
			"arm64":  int(float64(template.LaunchCount) * 0.3),
		},
		LastUpdated: time.Now(),
	}

	return analytics, nil
}

// GetUsageStats returns usage statistics for a specific timeframe
func (r *Registry) GetUsageStats(templateID string, timeframe string) (*UsageStats, error) {
	template, exists := r.templateCache[templateID]
	if !exists {
		return nil, fmt.Errorf("template not found: %s", templateID)
	}

	// Calculate date range based on timeframe
	endDate := time.Now()
	var startDate time.Time
	switch timeframe {
	case "day":
		startDate = endDate.Add(-24 * time.Hour)
	case "week":
		startDate = endDate.Add(-7 * 24 * time.Hour)
	case "month":
		startDate = endDate.Add(-30 * 24 * time.Hour)
	case "year":
		startDate = endDate.Add(-365 * 24 * time.Hour)
	default:
		return nil, fmt.Errorf("invalid timeframe: %s", timeframe)
	}

	// In production, this would query analytics data for the timeframe
	stats := &UsageStats{
		TemplateID:        templateID,
		Timeframe:         timeframe,
		StartDate:         startDate,
		EndDate:           endDate,
		Downloads:         template.DownloadCount / 10, // Mock recent activity
		Launches:          template.LaunchCount / 10,
		Successes:         int(float64(template.LaunchCount/10) * 0.85),
		Failures:          int(float64(template.LaunchCount/10) * 0.15),
		SuccessRate:       0.85,
		AverageLaunchTime: time.Duration(45) * time.Second,
	}

	return stats, nil
}

// Helper methods

// matchesQuery determines if a template satisfies all search criteria
func (r *Registry) matchesQuery(template *CommunityTemplate, query SearchQuery) bool {
	return r.matchesTextSearch(template, query) &&
		r.matchesCategoryFilters(template, query) &&
		r.matchesArchitectureFilter(template, query) &&
		r.matchesRegionFilter(template, query) &&
		r.matchesQualityFilters(template, query) &&
		r.matchesAMIFilter(template, query)
}

// matchesTextSearch checks if template matches text query
func (r *Registry) matchesTextSearch(template *CommunityTemplate, query SearchQuery) bool {
	if query.Query == "" {
		return true
	}

	searchText := strings.ToLower(query.Query)
	return strings.Contains(strings.ToLower(template.Name), searchText) ||
		strings.Contains(strings.ToLower(template.Description), searchText)
}

// matchesCategoryFilters checks if template matches category and author filters
func (r *Registry) matchesCategoryFilters(template *CommunityTemplate, query SearchQuery) bool {
	if query.Category != "" && template.Category != query.Category {
		return false
	}
	if query.Author != "" && template.Author != query.Author {
		return false
	}
	return true
}

// matchesArchitectureFilter checks if template supports required architecture
func (r *Registry) matchesArchitectureFilter(template *CommunityTemplate, query SearchQuery) bool {
	if query.Architecture == "" {
		return true
	}

	for _, arch := range template.Architecture {
		if arch == query.Architecture {
			return true
		}
	}
	return false
}

// matchesRegionFilter checks if template supports required region
func (r *Registry) matchesRegionFilter(template *CommunityTemplate, query SearchQuery) bool {
	if query.Region == "" {
		return true
	}

	for _, region := range template.SupportedRegions {
		if region == query.Region {
			return true
		}
	}
	return false
}

// matchesQualityFilters checks if template meets quality requirements
func (r *Registry) matchesQualityFilters(template *CommunityTemplate, query SearchQuery) bool {
	if query.MinRating > 0 && template.Rating < query.MinRating {
		return false
	}
	if query.VerifiedOnly && !template.Verified {
		return false
	}
	if query.FeaturedOnly && !template.Featured {
		return false
	}
	if query.MinDownloads > 0 && template.DownloadCount < query.MinDownloads {
		return false
	}
	return true
}

// matchesAMIFilter checks if template has available AMI when required
func (r *Registry) matchesAMIFilter(template *CommunityTemplate, query SearchQuery) bool {
	if !query.AMIAvailable {
		return true
	}
	return template.AMIInfo != nil && template.AMIInfo.Available
}

func (r *Registry) sortResults(results []*CommunityTemplate, sortBy, sortOrder string) {
	if sortBy == "" {
		sortBy = "rating" // Default sort
	}
	if sortOrder == "" {
		sortOrder = "desc" // Default order
	}

	ascending := sortOrder == "asc"

	sort.Slice(results, func(i, j int) bool {
		var less bool
		switch sortBy {
		case "rating":
			less = results[i].Rating < results[j].Rating
		case "downloads":
			less = results[i].DownloadCount < results[j].DownloadCount
		case "updated":
			less = results[i].UpdatedAt.Before(results[j].UpdatedAt)
		case "created":
			less = results[i].CreatedAt.Before(results[j].CreatedAt)
		case "name":
			less = results[i].Name < results[j].Name
		default:
			less = results[i].Rating < results[j].Rating
		}
		if ascending {
			return less
		}
		return !less
	})
}

func (r *Registry) generateTemplateID(name string) string {
	// Create URL-safe template ID
	id := strings.ToLower(name)
	id = strings.ReplaceAll(id, " ", "-")
	id = strings.ReplaceAll(id, "_", "-")

	// Add timestamp to ensure uniqueness
	timestamp := time.Now().Unix()
	return fmt.Sprintf("%s-%d", id, timestamp%10000)
}

func (r *Registry) createCommunityTemplate(templateID string, publication *TemplatePublication) (*CommunityTemplate, error) {
	// In production, this would integrate with the existing template system
	template := &CommunityTemplate{
		TemplateID:        templateID,
		Name:              publication.Name,
		Description:       publication.Description,
		Author:            "current-user", // Would be authenticated user
		AuthorName:        "Current User",
		Version:           "1.0.0",
		CreatedAt:         time.Now(),
		UpdatedAt:         time.Now(),
		Category:          publication.Category,
		Tags:              publication.Tags,
		Keywords:          publication.Keywords,
		ResearchDomain:    publication.ResearchDomain,
		Architecture:      []string{"x86_64", "arm64"}, // Default support
		SupportedRegions:  publication.TargetRegions,
		Rating:            0.0, // No ratings yet
		ReviewCount:       0,
		DownloadCount:     0,
		LaunchCount:       0,
		ForkCount:         0,
		Verified:          false,
		Featured:          false,
		LastTested:        time.Now(),
		TestStatus:        "unknown",
		SecurityScore:     75, // Default security score
		MaintenanceStatus: "active",
		Documentation:     publication.Documentation,
		Screenshots:       publication.Screenshots,
		VideoDemo:         publication.VideoDemo,
		Publication: &PublicationMetadata{
			License:       publication.License,
			Visibility:    publication.Visibility,
			PaperDOI:      publication.PaperDOI,
			FundingSource: publication.FundingSource,
		},
	}

	// Initialize AMI info if AMI generation is requested
	if publication.GenerateAMI {
		template.AMIInfo = &AMIAvailability{
			Available:      false, // Will be updated when AMI creation completes
			Regions:        make(map[string]string),
			LastUpdated:    time.Now(),
			CreationStatus: "creating",
		}
	}

	return template, nil
}

func (r *Registry) initiateAMIGeneration(templateID string, regions []string) []string {
	// In production, this would integrate with the AMI creation system
	var creationIDs []string
	for _, region := range regions {
		creationID := fmt.Sprintf("ami-creation-%s-%s", templateID, region)
		creationIDs = append(creationIDs, creationID)
	}
	return creationIDs
}

func (r *Registry) updateRatingMetrics(template *CommunityTemplate, newRating int) {
	// Simple rating calculation
	totalRating := template.Rating * float64(template.ReviewCount)
	template.ReviewCount++
	template.Rating = (totalRating + float64(newRating)) / float64(template.ReviewCount)
}

func (r *Registry) trackUsage(templateID string, event *UsageEvent) {
	// Update template metrics based on event
	template, exists := r.templateCache[templateID]
	if !exists {
		return
	}

	switch event.EventType {
	case "download":
		template.DownloadCount++
	case "launch":
		template.LaunchCount++
	case "view":
		// Just tracking, no metric update
	}
}

func (r *Registry) generateMockReviews(templateID string) []*TemplateReview {
	// Generate mock reviews for demonstration
	reviews := []*TemplateReview{
		{
			ReviewID:      "review-1",
			TemplateID:    templateID,
			Reviewer:      "researcher-123",
			ReviewerName:  "Dr. Jane Smith",
			Rating:        5,
			Title:         "Excellent for our research",
			Content:       "This template saved us weeks of setup time. Everything worked perfectly out of the box.",
			UseCase:       "genomics-analysis",
			VerifiedUsage: true,
			HelpfulVotes:  15,
			CreatedAt:     time.Now().Add(-7 * 24 * time.Hour),
		},
		{
			ReviewID:      "review-2",
			TemplateID:    templateID,
			Reviewer:      "grad-student-456",
			ReviewerName:  "Alex Chen",
			Rating:        4,
			Title:         "Good template with minor issues",
			Content:       "Works well overall, but had to tweak some configurations for our specific use case.",
			UseCase:       "machine-learning",
			VerifiedUsage: true,
			HelpfulVotes:  8,
			CreatedAt:     time.Now().Add(-3 * 24 * time.Hour),
		},
	}
	return reviews
}

// LoadSampleData loads sample marketplace data for development and testing
func (r *Registry) LoadSampleData() {
	// Sample categories are already loaded in DefaultCategories()

	// Load sample templates
	sampleTemplates := []*CommunityTemplate{
		{
			TemplateID:        "genomics-pipeline-v3",
			Name:              "Advanced Genomics Analysis Pipeline",
			Description:       "Complete genomics workflow with GATK, BWA, and Bioconductor",
			Author:            "research-lab-genomics",
			AuthorName:        "Genomics Research Lab",
			Version:           "3.2.1",
			CreatedAt:         time.Now().Add(-30 * 24 * time.Hour),
			UpdatedAt:         time.Now().Add(-7 * 24 * time.Hour),
			Category:          "bioinformatics",
			Tags:              []string{"genomics", "gatk", "bioconductor", "ngs"},
			Keywords:          []string{"whole-genome", "variant-calling", "R"},
			ResearchDomain:    "genomics",
			Architecture:      []string{"x86_64", "arm64"},
			SupportedRegions:  []string{"us-east-1", "us-west-2", "eu-west-1"},
			Rating:            4.7,
			ReviewCount:       23,
			DownloadCount:     1547,
			LaunchCount:       892,
			ForkCount:         12,
			Verified:          true,
			Featured:          true,
			LastTested:        time.Now().Add(-24 * time.Hour),
			TestStatus:        "passed",
			SecurityScore:     92,
			MaintenanceStatus: "active",
			Documentation:     "# Advanced Genomics Analysis Pipeline\n\nThis template provides a complete genomics analysis environment...",
			Screenshots:       []string{"screenshot1.png", "screenshot2.png"},
			Publication: &PublicationMetadata{
				License:       "MIT",
				Visibility:    "public",
				PaperDOI:      "10.1038/s41586-2024-genomics",
				FundingSource: "NIH Grant R01-HG012345",
			},
			AMIInfo: &AMIAvailability{
				Available: true,
				Regions: map[string]string{
					"us-east-1": "ami-genomics-123456",
					"us-west-2": "ami-genomics-789012",
					"eu-west-1": "ami-genomics-345678",
				},
				LastUpdated:    time.Now().Add(-24 * time.Hour),
				CreationStatus: "available",
			},
		},
		{
			TemplateID:        "machine-learning-gpu",
			Name:              "GPU-Accelerated ML Environment",
			Description:       "PyTorch, TensorFlow, and CUDA toolkit for deep learning research",
			Author:            "ai-research-team",
			AuthorName:        "AI Research Team",
			Version:           "2.1.0",
			CreatedAt:         time.Now().Add(-45 * 24 * time.Hour),
			UpdatedAt:         time.Now().Add(-14 * 24 * time.Hour),
			Category:          "machine-learning",
			Tags:              []string{"pytorch", "tensorflow", "cuda", "gpu"},
			Keywords:          []string{"deep-learning", "neural-networks", "transformers"},
			ResearchDomain:    "artificial-intelligence",
			Architecture:      []string{"x86_64"},
			SupportedRegions:  []string{"us-east-1", "us-west-2", "eu-west-1", "ap-south-1"},
			Rating:            4.5,
			ReviewCount:       67,
			DownloadCount:     2341,
			LaunchCount:       1456,
			ForkCount:         28,
			Verified:          true,
			Featured:          true,
			LastTested:        time.Now().Add(-12 * time.Hour),
			TestStatus:        "passed",
			SecurityScore:     88,
			MaintenanceStatus: "active",
			Documentation:     "# GPU-Accelerated ML Environment\n\nOptimized for deep learning research with latest frameworks...",
			Screenshots:       []string{"ml-screenshot1.png", "ml-screenshot2.png"},
			VideoDemo:         "https://example.com/ml-demo-video",
			Publication: &PublicationMetadata{
				License:    "Apache-2.0",
				Visibility: "public",
			},
			AMIInfo: &AMIAvailability{
				Available: true,
				Regions: map[string]string{
					"us-east-1":  "ami-ml-gpu-123456",
					"us-west-2":  "ami-ml-gpu-789012",
					"eu-west-1":  "ami-ml-gpu-345678",
					"ap-south-1": "ami-ml-gpu-901234",
				},
				LastUpdated:    time.Now().Add(-12 * time.Hour),
				CreationStatus: "available",
			},
		},
		{
			TemplateID:        "r-statistical-analysis",
			Name:              "R Statistical Analysis Workbench",
			Description:       "RStudio Server with tidyverse, statistical packages, and visualization tools",
			Author:            "stats-department",
			AuthorName:        "Statistics Department",
			Version:           "1.8.5",
			CreatedAt:         time.Now().Add(-60 * 24 * time.Hour),
			UpdatedAt:         time.Now().Add(-21 * 24 * time.Hour),
			Category:          "statistics",
			Tags:              []string{"r", "rstudio", "tidyverse", "statistics"},
			Keywords:          []string{"data-analysis", "visualization", "biostatistics"},
			ResearchDomain:    "statistics",
			Architecture:      []string{"x86_64", "arm64"},
			SupportedRegions:  []string{"us-east-1", "us-west-2", "eu-west-1"},
			Rating:            4.3,
			ReviewCount:       34,
			DownloadCount:     987,
			LaunchCount:       543,
			ForkCount:         8,
			Verified:          false,
			Featured:          false,
			LastTested:        time.Now().Add(-48 * time.Hour),
			TestStatus:        "passed",
			SecurityScore:     79,
			MaintenanceStatus: "active",
			Documentation:     "# R Statistical Analysis Workbench\n\nComprehensive R environment for statistical analysis...",
			Publication: &PublicationMetadata{
				License:    "GPL-3.0",
				Visibility: "public",
			},
		},
	}

	// Store sample templates in cache
	for _, template := range sampleTemplates {
		r.templateCache[template.TemplateID] = template
		if template.Featured {
			r.featured = append(r.featured, template)
		}
	}
}

// DefaultCategories returns the default template categories
func DefaultCategories() []TemplateCategory {
	return []TemplateCategory{
		{
			ID:          "machine-learning",
			Name:        "Machine Learning & AI",
			Description: "Deep learning, neural networks, and AI research environments",
			Icon:        "🤖",
			Color:       "#FF6B6B",
			Featured:    true,
		},
		{
			ID:          "bioinformatics",
			Name:        "Bioinformatics",
			Description: "Genomics, proteomics, and computational biology tools",
			Icon:        "🧬",
			Color:       "#4ECDC4",
			Featured:    true,
		},
		{
			ID:          "statistics",
			Name:        "Statistics & Data Science",
			Description: "Statistical analysis, data visualization, and modeling",
			Icon:        "📊",
			Color:       "#45B7D1",
			Featured:    true,
		},
		{
			ID:          "physics",
			Name:        "Physics & Astronomy",
			Description: "Computational physics, simulations, and astronomical data analysis",
			Icon:        "🔬",
			Color:       "#96CEB4",
		},
		{
			ID:          "chemistry",
			Name:        "Chemistry",
			Description: "Molecular modeling, quantum chemistry, and drug discovery",
			Icon:        "⚗️",
			Color:       "#FECA57",
		},
		{
			ID:          "economics",
			Name:        "Economics & Finance",
			Description: "Econometric analysis, financial modeling, and market research",
			Icon:        "💹",
			Color:       "#FF9FF3",
		},
		{
			ID:          "engineering",
			Name:        "Engineering",
			Description: "CAD/CAE, simulations, and engineering analysis tools",
			Icon:        "⚙️",
			Color:       "#54A0FF",
		},
		{
			ID:          "social-sciences",
			Name:        "Social Sciences",
			Description: "Survey analysis, behavioral research, and social network analysis",
			Icon:        "👥",
			Color:       "#5F27CD",
		},
		{
			ID:          "web-development",
			Name:        "Web Development",
			Description: "Full-stack development environments and web frameworks",
			Icon:        "💻",
			Color:       "#00D2D3",
		},
		{
			ID:          "data-processing",
			Name:        "Data Processing",
			Description: "ETL pipelines, big data processing, and data engineering",
			Icon:        "🗃️",
			Color:       "#FF7675",
		},
	}
}
