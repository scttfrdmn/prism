package marketplace

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// TestCommunityTemplate tests CommunityTemplate structure and validation
func TestCommunityTemplate(t *testing.T) {
	now := time.Now()
	template := &CommunityTemplate{
		TemplateID:        "test-template-123",
		Name:              "Test ML Template",
		Description:       "A comprehensive machine learning environment for research",
		Author:            "research-team",
		AuthorName:        "Research Team",
		Version:           "1.2.0",
		CreatedAt:         now.Add(-7 * 24 * time.Hour),
		UpdatedAt:         now,
		Category:          "machine-learning",
		Tags:              []string{"pytorch", "tensorflow", "jupyter"},
		Keywords:          []string{"deep-learning", "neural-networks"},
		ResearchDomain:    "artificial-intelligence",
		Architecture:      []string{"x86_64", "arm64"},
		SupportedRegions:  []string{"us-east-1", "us-west-2", "eu-west-1"},
		Rating:            4.5,
		ReviewCount:       25,
		DownloadCount:     1500,
		LaunchCount:       750,
		ForkCount:         15,
		Verified:          true,
		Featured:          true,
		LastTested:        now.Add(-12 * time.Hour),
		TestStatus:        "passed",
		SecurityScore:     88,
		MaintenanceStatus: "active",
		Documentation:     "# Test ML Template\nThis is a test template.",
		Screenshots:       []string{"screenshot1.png", "screenshot2.png"},
		VideoDemo:         "https://example.com/demo-video",
	}

	// Test basic fields
	assert.Equal(t, "test-template-123", template.TemplateID)
	assert.Equal(t, "Test ML Template", template.Name)
	assert.Equal(t, "research-team", template.Author)
	assert.Equal(t, "1.2.0", template.Version)
	assert.Equal(t, "machine-learning", template.Category)
	assert.Contains(t, template.Tags, "pytorch")
	assert.Contains(t, template.Architecture, "x86_64")
	assert.Contains(t, template.SupportedRegions, "us-east-1")

	// Test metrics
	assert.Equal(t, 4.5, template.Rating)
	assert.Equal(t, 25, template.ReviewCount)
	assert.Equal(t, 1500, template.DownloadCount)
	assert.Equal(t, 750, template.LaunchCount)
	assert.Equal(t, 15, template.ForkCount)

	// Test quality indicators
	assert.True(t, template.Verified)
	assert.True(t, template.Featured)
	assert.Equal(t, "passed", template.TestStatus)
	assert.Equal(t, 88, template.SecurityScore)
	assert.Equal(t, "active", template.MaintenanceStatus)

	// Test timestamps
	assert.True(t, template.UpdatedAt.After(template.CreatedAt))
	assert.True(t, template.LastTested.After(template.CreatedAt))
}

// TestTemplatePublication tests TemplatePublication structure
func TestTemplatePublication(t *testing.T) {
	publication := &TemplatePublication{
		SourceInstanceID: "i-1234567890abcdef0",
		Name:             "My Research Template",
		Description:      "Custom template for genomics research",
		Category:         "bioinformatics",
		Tags:             []string{"genomics", "gatk", "r"},
		Keywords:         []string{"variant-calling", "bioconductor"},
		Documentation:    "# My Research Template\nDetailed documentation here",
		Screenshots:      []string{"setup.png", "results.png"},
		VideoDemo:        "https://example.com/my-demo",
		Visibility:       "public",
		License:          "MIT",
		GenerateAMI:      true,
		TargetRegions:    []string{"us-east-1", "us-west-2"},
		Metadata:         map[string]string{"lab": "genomics", "grant": "NIH-123"},
		ResearchDomain:   "genomics",
		PaperDOI:         "10.1038/s41586-2024-example",
		FundingSource:    "NIH Grant R01-HG012345",
	}

	assert.Equal(t, "i-1234567890abcdef0", publication.SourceInstanceID)
	assert.Equal(t, "My Research Template", publication.Name)
	assert.Equal(t, "bioinformatics", publication.Category)
	assert.Contains(t, publication.Tags, "genomics")
	assert.Contains(t, publication.TargetRegions, "us-east-1")
	assert.Equal(t, "public", publication.Visibility)
	assert.Equal(t, "MIT", publication.License)
	assert.True(t, publication.GenerateAMI)
	assert.Equal(t, "10.1038/s41586-2024-example", publication.PaperDOI)
	assert.Equal(t, "genomics", publication.Metadata["lab"])
}

// TestPublicationResult tests PublicationResult structure
func TestPublicationResult(t *testing.T) {
	now := time.Now()
	result := &PublicationResult{
		TemplateID:     "my-template-456",
		PublicationURL: "https://marketplace.cloudworkstation.com/templates/my-template-456",
		Status:         "published",
		Message:        "Template published successfully",
		CreatedAt:      now,
		AMICreationIDs: []string{"ami-creation-123", "ami-creation-456"},
	}

	assert.Equal(t, "my-template-456", result.TemplateID)
	assert.Equal(t, "published", result.Status)
	assert.Equal(t, "Template published successfully", result.Message)
	assert.Equal(t, now, result.CreatedAt)
	assert.Len(t, result.AMICreationIDs, 2)
	assert.Contains(t, result.AMICreationIDs, "ami-creation-123")
}

// TestTemplateUpdate tests TemplateUpdate structure
func TestTemplateUpdate(t *testing.T) {
	update := &TemplateUpdate{
		Name:          "Updated Template Name",
		Description:   "Updated description with new features",
		Documentation: "# Updated Documentation\nNew sections added",
		Tags:          []string{"updated", "improved", "v2"},
		Keywords:      []string{"enhanced", "optimized"},
		Screenshots:   []string{"new-screenshot.png"},
		VideoDemo:     "https://example.com/updated-demo",
		Metadata:      map[string]string{"version": "2.0", "updated": "true"},
		Version:       "2.0.0",
	}

	assert.Equal(t, "Updated Template Name", update.Name)
	assert.Equal(t, "Updated description with new features", update.Description)
	assert.Contains(t, update.Tags, "updated")
	assert.Contains(t, update.Keywords, "enhanced")
	assert.Equal(t, "2.0.0", update.Version)
	assert.Equal(t, "2.0", update.Metadata["version"])
}

// TestSearchQuery tests SearchQuery structure and validation
func TestSearchQuery(t *testing.T) {
	tests := []struct {
		name    string
		query   SearchQuery
		isValid bool
	}{
		{
			name: "basic_text_search",
			query: SearchQuery{
				Query: "machine learning",
			},
			isValid: true,
		},
		{
			name: "category_filter_search",
			query: SearchQuery{
				Category: "bioinformatics",
				Tags:     []string{"genomics", "gatk"},
			},
			isValid: true,
		},
		{
			name: "advanced_search_with_filters",
			query: SearchQuery{
				Query:        "deep learning",
				Category:     "machine-learning",
				Tags:         []string{"pytorch", "tensorflow"},
				Architecture: "x86_64",
				Region:       "us-east-1",
				MinRating:    4.0,
				VerifiedOnly: true,
				FeaturedOnly: false,
				MinDownloads: 100,
				SortBy:       "rating",
				SortOrder:    "desc",
				Limit:        20,
				Offset:       0,
			},
			isValid: true,
		},
		{
			name: "pagination_search",
			query: SearchQuery{
				Query:     "analysis",
				SortBy:    "downloads",
				SortOrder: "desc",
				Limit:     50,
				Offset:    100,
			},
			isValid: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.isValid {
				// Test that all fields are accessible and have expected types
				assert.IsType(t, "", tt.query.Query)
				assert.IsType(t, []string{}, tt.query.Keywords)
				assert.IsType(t, "", tt.query.Category)
				assert.IsType(t, []string{}, tt.query.Tags)
				assert.IsType(t, "", tt.query.Author)
				assert.IsType(t, "", tt.query.Architecture)
				assert.IsType(t, "", tt.query.Region)
				assert.IsType(t, false, tt.query.AMIAvailable)
				assert.IsType(t, float64(0), tt.query.MinRating)
				assert.IsType(t, false, tt.query.VerifiedOnly)
				assert.IsType(t, false, tt.query.FeaturedOnly)
				assert.IsType(t, 0, tt.query.MinDownloads)
				assert.IsType(t, "", tt.query.SortBy)
				assert.IsType(t, "", tt.query.SortOrder)
				assert.IsType(t, 0, tt.query.Limit)
				assert.IsType(t, 0, tt.query.Offset)
			}
		})
	}
}

// TestTemplateCategory tests TemplateCategory structure
func TestTemplateCategory(t *testing.T) {
	category := TemplateCategory{
		ID:            "machine-learning",
		Name:          "Machine Learning & AI",
		Description:   "Deep learning, neural networks, and AI research environments",
		Icon:          "🤖",
		Color:         "#FF6B6B",
		TemplateCount: 45,
		Featured:      true,
	}

	assert.Equal(t, "machine-learning", category.ID)
	assert.Equal(t, "Machine Learning & AI", category.Name)
	assert.Equal(t, "🤖", category.Icon)
	assert.Equal(t, "#FF6B6B", category.Color)
	assert.Equal(t, 45, category.TemplateCount)
	assert.True(t, category.Featured)
}

// TestTemplateReview tests TemplateReview structure
func TestTemplateReview(t *testing.T) {
	now := time.Now()
	review := &TemplateReview{
		ReviewID:       "review-123",
		TemplateID:     "template-456",
		Reviewer:       "researcher-789",
		ReviewerName:   "Dr. Jane Smith",
		Rating:         5,
		Title:          "Excellent template for our research",
		Content:        "This template saved us weeks of setup time. Everything worked perfectly.",
		UseCase:        "genomics-analysis",
		VerifiedUsage:  true,
		VerifiedBy:     "admin-user",
		HelpfulVotes:   25,
		UnhelpfulVotes: 2,
		CreatedAt:      now.Add(-7 * 24 * time.Hour),
		UpdatedAt:      now,
	}

	assert.Equal(t, "review-123", review.ReviewID)
	assert.Equal(t, "template-456", review.TemplateID)
	assert.Equal(t, "researcher-789", review.Reviewer)
	assert.Equal(t, "Dr. Jane Smith", review.ReviewerName)
	assert.Equal(t, 5, review.Rating)
	assert.Equal(t, "Excellent template for our research", review.Title)
	assert.Equal(t, "genomics-analysis", review.UseCase)
	assert.True(t, review.VerifiedUsage)
	assert.Equal(t, "admin-user", review.VerifiedBy)
	assert.Equal(t, 25, review.HelpfulVotes)
	assert.Equal(t, 2, review.UnhelpfulVotes)
	assert.True(t, review.UpdatedAt.After(review.CreatedAt))
}

// TestReply tests Reply structure
func TestReply(t *testing.T) {
	now := time.Now()
	reply := Reply{
		ReplyID:     "reply-123",
		Replier:     "template-author",
		ReplierName: "Template Author",
		Content:     "Thank you for the feedback! We're glad it helped your research.",
		CreatedAt:   now,
	}

	assert.Equal(t, "reply-123", reply.ReplyID)
	assert.Equal(t, "template-author", reply.Replier)
	assert.Equal(t, "Template Author", reply.ReplierName)
	assert.Contains(t, reply.Content, "feedback")
	assert.Equal(t, now, reply.CreatedAt)
}

// TestReviewPagination tests ReviewPagination structure
func TestReviewPagination(t *testing.T) {
	pagination := &ReviewPagination{
		Limit:  20,
		Offset: 40,
		SortBy: "helpful",
	}

	assert.Equal(t, 20, pagination.Limit)
	assert.Equal(t, 40, pagination.Offset)
	assert.Equal(t, "helpful", pagination.SortBy)
}

// TestReviewResponse tests ReviewResponse structure
func TestReviewResponse(t *testing.T) {
	reviews := []*TemplateReview{
		{ReviewID: "review-1", Rating: 5},
		{ReviewID: "review-2", Rating: 4},
		{ReviewID: "review-3", Rating: 5},
	}

	response := &ReviewResponse{
		Reviews:    reviews,
		TotalCount: 150,
		Page:       3,
		TotalPages: 8,
		HasMore:    true,
	}

	assert.Len(t, response.Reviews, 3)
	assert.Equal(t, 150, response.TotalCount)
	assert.Equal(t, 3, response.Page)
	assert.Equal(t, 8, response.TotalPages)
	assert.True(t, response.HasMore)
}

// TestUsageEvent tests UsageEvent structure
func TestUsageEvent(t *testing.T) {
	now := time.Now()
	event := &UsageEvent{
		EventType:    "launch",
		TemplateID:   "template-123",
		UserID:       "user-456",
		InstanceID:   "i-1234567890abcdef0",
		Region:       "us-east-1",
		Architecture: "x86_64",
		LaunchTime:   45 * time.Second,
		Metadata:     map[string]string{"instance_type": "m5.xlarge", "success": "true"},
		Timestamp:    now,
	}

	assert.Equal(t, "launch", event.EventType)
	assert.Equal(t, "template-123", event.TemplateID)
	assert.Equal(t, "user-456", event.UserID)
	assert.Equal(t, "i-1234567890abcdef0", event.InstanceID)
	assert.Equal(t, "us-east-1", event.Region)
	assert.Equal(t, "x86_64", event.Architecture)
	assert.Equal(t, 45*time.Second, event.LaunchTime)
	assert.Equal(t, "m5.xlarge", event.Metadata["instance_type"])
	assert.Equal(t, now, event.Timestamp)
}

// TestTemplateFork tests TemplateFork structure
func TestTemplateFork(t *testing.T) {
	modifications := []Modification{
		{
			Type:        "package_add",
			Description: "Added custom genomics packages",
			Details:     "samtools, bcftools, htslib",
		},
		{
			Type:        "config_change",
			Description: "Modified memory settings",
			Details:     "Increased Java heap size for GATK",
		},
	}

	fork := &TemplateFork{
		NewName:        "Custom Genomics Pipeline",
		NewDescription: "Modified pipeline with additional tools for our lab",
		Modifications:  modifications,
		Private:        false,
	}

	assert.Equal(t, "Custom Genomics Pipeline", fork.NewName)
	assert.Equal(t, "Modified pipeline with additional tools for our lab", fork.NewDescription)
	assert.Len(t, fork.Modifications, 2)
	assert.Equal(t, "package_add", fork.Modifications[0].Type)
	assert.Equal(t, "config_change", fork.Modifications[1].Type)
	assert.False(t, fork.Private)
}

// TestModification tests Modification structure
func TestModification(t *testing.T) {
	modification := Modification{
		Type:        "package_remove",
		Description: "Removed unused packages to reduce size",
		Details:     "Removed LibreOffice, games, and development tools",
	}

	assert.Equal(t, "package_remove", modification.Type)
	assert.Equal(t, "Removed unused packages to reduce size", modification.Description)
	assert.Contains(t, modification.Details, "LibreOffice")
}

// TestResourceRequirements tests ResourceRequirements structure
func TestResourceRequirements(t *testing.T) {
	requirements := &ResourceRequirements{
		MinCPU:       8,
		MinMemoryGB:  32,
		MinStorageGB: 100,
		RequiresGPU:  true,
		GPUType:      "V100",
		NetworkBW:    "10Gbps",
	}

	assert.Equal(t, 8, requirements.MinCPU)
	assert.Equal(t, 32, requirements.MinMemoryGB)
	assert.Equal(t, 100, requirements.MinStorageGB)
	assert.True(t, requirements.RequiresGPU)
	assert.Equal(t, "V100", requirements.GPUType)
	assert.Equal(t, "10Gbps", requirements.NetworkBW)
}

// TestCostEstimate tests CostEstimate structure
func TestCostEstimate(t *testing.T) {
	now := time.Now()
	estimate := &CostEstimate{
		HourlyCost:   2.50,
		DailyCost:    60.00,
		MonthlyCost:  1800.00,
		Region:       "us-east-1",
		InstanceType: "m5.2xlarge",
		Currency:     "USD",
		LastUpdated:  now,
	}

	assert.Equal(t, 2.50, estimate.HourlyCost)
	assert.Equal(t, 60.00, estimate.DailyCost)
	assert.Equal(t, 1800.00, estimate.MonthlyCost)
	assert.Equal(t, "us-east-1", estimate.Region)
	assert.Equal(t, "m5.2xlarge", estimate.InstanceType)
	assert.Equal(t, "USD", estimate.Currency)
	assert.Equal(t, now, estimate.LastUpdated)
}

// TestPublicationMetadata tests PublicationMetadata structure
func TestPublicationMetadata(t *testing.T) {
	metadata := &PublicationMetadata{
		License:          "MIT",
		Visibility:       "public",
		PaperDOI:         "10.1038/s41586-2024-example",
		FundingSource:    "NSF Grant 123456",
		DocumentationURL: "https://docs.example.com/template",
		RepositoryURL:    "https://github.com/example/template",
		ContactEmail:     "author@university.edu",
	}

	assert.Equal(t, "MIT", metadata.License)
	assert.Equal(t, "public", metadata.Visibility)
	assert.Equal(t, "10.1038/s41586-2024-example", metadata.PaperDOI)
	assert.Equal(t, "NSF Grant 123456", metadata.FundingSource)
	assert.Equal(t, "https://docs.example.com/template", metadata.DocumentationURL)
	assert.Equal(t, "https://github.com/example/template", metadata.RepositoryURL)
	assert.Equal(t, "author@university.edu", metadata.ContactEmail)
}

// TestAMIAvailability tests AMIAvailability structure
func TestAMIAvailability(t *testing.T) {
	now := time.Now()
	amiInfo := &AMIAvailability{
		Available: true,
		Regions: map[string]string{
			"us-east-1": "ami-12345678",
			"us-west-2": "ami-87654321",
			"eu-west-1": "ami-11223344",
		},
		LastUpdated:    now,
		CreationStatus: "available",
	}

	assert.True(t, amiInfo.Available)
	assert.Len(t, amiInfo.Regions, 3)
	assert.Equal(t, "ami-12345678", amiInfo.Regions["us-east-1"])
	assert.Equal(t, "ami-87654321", amiInfo.Regions["us-west-2"])
	assert.Equal(t, "ami-11223344", amiInfo.Regions["eu-west-1"])
	assert.Equal(t, now, amiInfo.LastUpdated)
	assert.Equal(t, "available", amiInfo.CreationStatus)
}

// TestUsageExample tests UsageExample structure
func TestUsageExample(t *testing.T) {
	example := UsageExample{
		Title:       "Basic ML Training",
		Description: "How to train a simple neural network using this template",
		Command:     "python train.py --model resnet --epochs 10",
		UseCase:     "image-classification",
		Difficulty:  "beginner",
	}

	assert.Equal(t, "Basic ML Training", example.Title)
	assert.Equal(t, "How to train a simple neural network using this template", example.Description)
	assert.Equal(t, "python train.py --model resnet --epochs 10", example.Command)
	assert.Equal(t, "image-classification", example.UseCase)
	assert.Equal(t, "beginner", example.Difficulty)
}

// TestResearchPaper tests ResearchPaper structure
func TestResearchPaper(t *testing.T) {
	now := time.Now()
	paper := ResearchPaper{
		DOI:         "10.1038/s41586-2024-ml",
		Title:       "Deep Learning Advances in Genomic Analysis",
		Authors:     []string{"Dr. Jane Smith", "Dr. John Doe", "Dr. Alice Johnson"},
		Journal:     "Nature",
		Year:        2024,
		URL:         "https://nature.com/articles/s41586-2024-ml",
		Abstract:    "This paper presents novel approaches to genomic analysis using deep learning...",
		PublishedAt: now,
	}

	assert.Equal(t, "10.1038/s41586-2024-ml", paper.DOI)
	assert.Equal(t, "Deep Learning Advances in Genomic Analysis", paper.Title)
	assert.Len(t, paper.Authors, 3)
	assert.Contains(t, paper.Authors, "Dr. Jane Smith")
	assert.Equal(t, "Nature", paper.Journal)
	assert.Equal(t, 2024, paper.Year)
	assert.Contains(t, paper.Abstract, "genomic analysis")
	assert.Equal(t, now, paper.PublishedAt)
}

// TestTemplateAnalytics tests TemplateAnalytics structure
func TestTemplateAnalytics(t *testing.T) {
	now := time.Now()
	failureReasons := []FailureReason{
		{
			Reason:         "Insufficient memory",
			Count:          15,
			Percentage:     12.5,
			LastOccurrence: now.Add(-24 * time.Hour),
		},
		{
			Reason:         "Network timeout",
			Count:          8,
			Percentage:     6.7,
			LastOccurrence: now.Add(-48 * time.Hour),
		},
	}

	dailyUsage := []DailyUsagePoint{
		{Date: now.Add(-2 * 24 * time.Hour), Downloads: 25, Launches: 15, Successes: 13},
		{Date: now.Add(-1 * 24 * time.Hour), Downloads: 30, Launches: 18, Successes: 16},
		{Date: now, Downloads: 22, Launches: 12, Successes: 11},
	}

	monthlyTrend := []MonthlyTrendPoint{
		{Month: now.Add(-2 * 30 * 24 * time.Hour), Downloads: 450, Launches: 280, Rating: 4.2},
		{Month: now.Add(-1 * 30 * 24 * time.Hour), Downloads: 520, Launches: 320, Rating: 4.4},
		{Month: now, Downloads: 610, Launches: 380, Rating: 4.5},
	}

	analytics := &TemplateAnalytics{
		TemplateID:        "template-analytics-test",
		TotalDownloads:    1580,
		TotalLaunches:     980,
		SuccessRate:       0.87,
		AverageRating:     4.5,
		TotalReviews:      45,
		TotalForks:        18,
		AverageLaunchTime: 42 * time.Second,
		FailureReasons:    failureReasons,
		RegionUsage: map[string]int{
			"us-east-1": 450,
			"us-west-2": 320,
			"eu-west-1": 210,
		},
		ArchitectureUsage: map[string]int{
			"x86_64": 686,
			"arm64":  294,
		},
		DailyUsage:   dailyUsage,
		MonthlyTrend: monthlyTrend,
		LastUpdated:  now,
	}

	assert.Equal(t, "template-analytics-test", analytics.TemplateID)
	assert.Equal(t, 1580, analytics.TotalDownloads)
	assert.Equal(t, 980, analytics.TotalLaunches)
	assert.Equal(t, 0.87, analytics.SuccessRate)
	assert.Equal(t, 4.5, analytics.AverageRating)
	assert.Equal(t, 45, analytics.TotalReviews)
	assert.Equal(t, 18, analytics.TotalForks)
	assert.Equal(t, 42*time.Second, analytics.AverageLaunchTime)

	// Test failure reasons
	assert.Len(t, analytics.FailureReasons, 2)
	assert.Equal(t, "Insufficient memory", analytics.FailureReasons[0].Reason)
	assert.Equal(t, 15, analytics.FailureReasons[0].Count)

	// Test region usage
	assert.Equal(t, 450, analytics.RegionUsage["us-east-1"])
	assert.Equal(t, 320, analytics.RegionUsage["us-west-2"])

	// Test architecture usage
	assert.Equal(t, 686, analytics.ArchitectureUsage["x86_64"])
	assert.Equal(t, 294, analytics.ArchitectureUsage["arm64"])

	// Test daily usage
	assert.Len(t, analytics.DailyUsage, 3)
	assert.Equal(t, 25, analytics.DailyUsage[0].Downloads)

	// Test monthly trend
	assert.Len(t, analytics.MonthlyTrend, 3)
	assert.Equal(t, 4.5, analytics.MonthlyTrend[2].Rating)

	assert.Equal(t, now, analytics.LastUpdated)
}

// TestUsageStats tests UsageStats structure
func TestUsageStats(t *testing.T) {
	now := time.Now()
	startDate := now.Add(-7 * 24 * time.Hour)

	periodComparison := &PeriodComparison{
		DownloadChange:    15.5,
		LaunchChange:      -5.2,
		SuccessRateChange: 2.1,
	}

	stats := &UsageStats{
		TemplateID:        "template-stats-test",
		Timeframe:         "week",
		StartDate:         startDate,
		EndDate:           now,
		Downloads:         150,
		Launches:          85,
		Successes:         72,
		Failures:          13,
		SuccessRate:       0.847,
		AverageLaunchTime: 38 * time.Second,
		PeriodComparison:  periodComparison,
	}

	assert.Equal(t, "template-stats-test", stats.TemplateID)
	assert.Equal(t, "week", stats.Timeframe)
	assert.Equal(t, startDate, stats.StartDate)
	assert.Equal(t, now, stats.EndDate)
	assert.Equal(t, 150, stats.Downloads)
	assert.Equal(t, 85, stats.Launches)
	assert.Equal(t, 72, stats.Successes)
	assert.Equal(t, 13, stats.Failures)
	assert.Equal(t, 0.847, stats.SuccessRate)
	assert.Equal(t, 38*time.Second, stats.AverageLaunchTime)

	// Test period comparison
	assert.NotNil(t, stats.PeriodComparison)
	assert.Equal(t, 15.5, stats.PeriodComparison.DownloadChange)
	assert.Equal(t, -5.2, stats.PeriodComparison.LaunchChange)
	assert.Equal(t, 2.1, stats.PeriodComparison.SuccessRateChange)
}

// TestMarketplaceConfig tests MarketplaceConfig structure
func TestMarketplaceConfig(t *testing.T) {
	config := &MarketplaceConfig{
		RegistryEndpoint:      "https://marketplace-api.cloudworkstation.com",
		S3Bucket:              "cloudworkstation-marketplace",
		DynamoDBTable:         "marketplace-templates",
		CDNEndpoint:           "https://cdn.cloudworkstation.com",
		AutoAMIGeneration:     true,
		DefaultRegions:        []string{"us-east-1", "us-west-2", "eu-west-1"},
		RequireModeration:     false,
		MinRatingForFeatured:  4.0,
		MinReviewsForFeatured: 10,
		PublishRateLimit:      5,
		ReviewRateLimit:       10,
		SearchRateLimit:       60,
	}

	assert.Equal(t, "https://marketplace-api.cloudworkstation.com", config.RegistryEndpoint)
	assert.Equal(t, "cloudworkstation-marketplace", config.S3Bucket)
	assert.Equal(t, "marketplace-templates", config.DynamoDBTable)
	assert.Equal(t, "https://cdn.cloudworkstation.com", config.CDNEndpoint)
	assert.True(t, config.AutoAMIGeneration)
	assert.Len(t, config.DefaultRegions, 3)
	assert.Contains(t, config.DefaultRegions, "us-east-1")
	assert.False(t, config.RequireModeration)
	assert.Equal(t, 4.0, config.MinRatingForFeatured)
	assert.Equal(t, 10, config.MinReviewsForFeatured)
	assert.Equal(t, 5, config.PublishRateLimit)
	assert.Equal(t, 10, config.ReviewRateLimit)
	assert.Equal(t, 60, config.SearchRateLimit)
}

// TestDefaultCategories tests the DefaultCategories function
func TestDefaultCategories(t *testing.T) {
	categories := DefaultCategories()

	assert.NotEmpty(t, categories, "Should have default categories")
	assert.GreaterOrEqual(t, len(categories), 5, "Should have multiple categories")

	// Test that expected categories exist
	categoryIDs := make(map[string]bool)
	for _, category := range categories {
		categoryIDs[category.ID] = true

		// Validate category structure
		assert.NotEmpty(t, category.ID, "Category ID should not be empty")
		assert.NotEmpty(t, category.Name, "Category name should not be empty")
		assert.NotEmpty(t, category.Description, "Category description should not be empty")
		assert.GreaterOrEqual(t, category.TemplateCount, 0, "Template count should be non-negative")
	}

	// Check for expected categories
	expectedCategories := []string{
		"machine-learning",
		"bioinformatics",
		"statistics",
		"physics",
		"chemistry",
		"economics",
		"engineering",
		"social-sciences",
		"web-development",
		"data-processing",
	}

	for _, expectedID := range expectedCategories {
		assert.True(t, categoryIDs[expectedID], "Should contain category: %s", expectedID)
	}
}
