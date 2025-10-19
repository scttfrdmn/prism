// Package marketplace provides the marketplace registry implementation
package marketplace

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

// Registry implements the MarketplaceRegistry interface with DynamoDB backend
type Registry struct {
	config        *MarketplaceConfig
	dynamoClient  *dynamodb.Client
	templateCache map[string]*CommunityTemplate // Optional cache for performance
	categories    []TemplateCategory
	featured      []*CommunityTemplate
	lastSync      time.Time

	// DynamoDB table names
	templatesTable string
	reviewsTable   string
	analyticsTable string
}

// NewRegistry creates a new marketplace registry with DynamoDB backend
func NewRegistry(config *MarketplaceConfig) *Registry {
	return &Registry{
		config:         config,
		dynamoClient:   nil, // Set via SetDynamoClient
		templateCache:  make(map[string]*CommunityTemplate),
		categories:     DefaultCategories(),
		featured:       make([]*CommunityTemplate, 0),
		lastSync:       time.Now(),
		templatesTable: "cloudworkstation-templates",
		reviewsTable:   "cloudworkstation-reviews",
		analyticsTable: "cloudworkstation-analytics",
	}
}

// NewRegistryWithDynamoDB creates a registry with DynamoDB client
func NewRegistryWithDynamoDB(config *MarketplaceConfig, dynamoClient *dynamodb.Client) *Registry {
	r := NewRegistry(config)
	r.dynamoClient = dynamoClient
	return r
}

// SetDynamoClient sets the DynamoDB client for the registry
func (r *Registry) SetDynamoClient(client *dynamodb.Client) {
	r.dynamoClient = client
}

// SearchTemplates searches for templates using DynamoDB with fallback to cache
func (r *Registry) SearchTemplates(query SearchQuery) ([]*CommunityTemplate, error) {
	ctx := context.Background()

	// If DynamoDB client is configured, use it for search
	if r.dynamoClient != nil {
		return r.searchTemplatesWithDynamoDB(ctx, query)
	}

	// Fallback to in-memory cache for development/testing
	results := make([]*CommunityTemplate, 0)
	for _, template := range r.templateCache {
		if r.matchesQuery(template, query) {
			results = append(results, template)
		}
	}

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

// searchTemplatesWithDynamoDB performs DynamoDB scan with filters
func (r *Registry) searchTemplatesWithDynamoDB(ctx context.Context, query SearchQuery) ([]*CommunityTemplate, error) {
	// Build DynamoDB filter expression
	filterExpr, exprAttrValues := r.buildDynamoDBFilterExpression(query)

	// Execute DynamoDB scan
	templates, err := r.executeDynamoDBScan(ctx, filterExpr, exprAttrValues)
	if err != nil {
		return nil, err
	}

	// Apply client-side filters
	templates = r.applyClientSideFilters(templates, query)

	// Sort and paginate results
	templates = r.sortAndPaginate(templates, query)

	return templates, nil
}

// buildDynamoDBFilterExpression constructs filter expression for DynamoDB
func (r *Registry) buildDynamoDBFilterExpression(query SearchQuery) (string, map[string]types.AttributeValue) {
	filterExpr := "visibility = :public"
	exprAttrValues := map[string]types.AttributeValue{
		":public": &types.AttributeValueMemberS{Value: "public"},
	}

	// Add category filter
	if query.Category != "" {
		filterExpr += " AND category = :category"
		exprAttrValues[":category"] = &types.AttributeValueMemberS{Value: query.Category}
	}

	// Add author filter
	if query.Author != "" {
		filterExpr += " AND author = :author"
		exprAttrValues[":author"] = &types.AttributeValueMemberS{Value: query.Author}
	}

	// Add verified filter
	if query.VerifiedOnly {
		filterExpr += " AND verified = :verified"
		exprAttrValues[":verified"] = &types.AttributeValueMemberBOOL{Value: true}
	}

	// Add featured filter
	if query.FeaturedOnly {
		filterExpr += " AND featured = :featured"
		exprAttrValues[":featured"] = &types.AttributeValueMemberBOOL{Value: true}
	}

	// Add rating filter
	if query.MinRating > 0 {
		filterExpr += " AND rating >= :minRating"
		exprAttrValues[":minRating"] = &types.AttributeValueMemberN{Value: fmt.Sprintf("%.2f", query.MinRating)}
	}

	return filterExpr, exprAttrValues
}

// executeDynamoDBScan performs DynamoDB scan and unmarshals results
func (r *Registry) executeDynamoDBScan(ctx context.Context, filterExpr string, exprAttrValues map[string]types.AttributeValue) ([]*CommunityTemplate, error) {
	scanInput := &dynamodb.ScanInput{
		TableName:                 aws.String(r.templatesTable),
		FilterExpression:          aws.String(filterExpr),
		ExpressionAttributeValues: exprAttrValues,
	}

	result, err := r.dynamoClient.Scan(ctx, scanInput)
	if err != nil {
		return nil, fmt.Errorf("DynamoDB scan failed: %w", err)
	}

	var templates []*CommunityTemplate
	for _, item := range result.Items {
		var template CommunityTemplate
		if err := attributevalue.UnmarshalMap(item, &template); err != nil {
			continue // Skip malformed items
		}
		templates = append(templates, &template)
	}

	return templates, nil
}

// applyClientSideFilters applies filters not supported by DynamoDB
func (r *Registry) applyClientSideFilters(templates []*CommunityTemplate, query SearchQuery) []*CommunityTemplate {
	var filtered []*CommunityTemplate

	for _, template := range templates {
		// Apply text search filter
		if !r.matchesTextQuery(template, query.Query) {
			continue
		}

		// Apply tag filter
		if !r.matchesTagFilter(template, query.Tags) {
			continue
		}

		filtered = append(filtered, template)
	}

	return filtered
}

// matchesTextQuery checks if template matches text search
func (r *Registry) matchesTextQuery(template *CommunityTemplate, query string) bool {
	if query == "" {
		return true
	}

	searchText := strings.ToLower(query)
	return strings.Contains(strings.ToLower(template.Name), searchText) ||
		strings.Contains(strings.ToLower(template.Description), searchText)
}

// matchesTagFilter checks if template has all required tags
func (r *Registry) matchesTagFilter(template *CommunityTemplate, requiredTags []string) bool {
	if len(requiredTags) == 0 {
		return true
	}

	templateTags := make(map[string]bool)
	for _, tag := range template.Tags {
		templateTags[strings.ToLower(tag)] = true
	}

	for _, requiredTag := range requiredTags {
		if !templateTags[strings.ToLower(requiredTag)] {
			return false
		}
	}

	return true
}

// sortAndPaginate sorts and paginates template results
func (r *Registry) sortAndPaginate(templates []*CommunityTemplate, query SearchQuery) []*CommunityTemplate {
	r.sortResults(templates, query.SortBy, query.SortOrder)

	// Apply offset
	if query.Offset > 0 && query.Offset < len(templates) {
		templates = templates[query.Offset:]
	}

	// Apply limit
	if query.Limit > 0 && query.Limit < len(templates) {
		templates = templates[:query.Limit]
	}

	return templates
}

// GetTemplate retrieves a specific template by ID from DynamoDB or cache
func (r *Registry) GetTemplate(templateID string) (*CommunityTemplate, error) {
	ctx := context.Background()

	// If DynamoDB client is configured, fetch from DynamoDB
	if r.dynamoClient != nil {
		input := &dynamodb.GetItemInput{
			TableName: aws.String(r.templatesTable),
			Key: map[string]types.AttributeValue{
				"template_id": &types.AttributeValueMemberS{Value: templateID},
			},
		}

		result, err := r.dynamoClient.GetItem(ctx, input)
		if err != nil {
			return nil, fmt.Errorf("DynamoDB GetItem failed: %w", err)
		}

		if result.Item == nil {
			return nil, fmt.Errorf("template not found: %s", templateID)
		}

		var template CommunityTemplate
		if err := attributevalue.UnmarshalMap(result.Item, &template); err != nil {
			return nil, fmt.Errorf("failed to unmarshal template: %w", err)
		}

		// Track access for analytics
		r.trackUsage(templateID, &UsageEvent{
			EventType:  "view",
			TemplateID: templateID,
			Timestamp:  time.Now(),
		})

		return &template, nil
	}

	// Fallback to cache
	template, exists := r.templateCache[templateID]
	if !exists {
		return nil, fmt.Errorf("template not found: %s", templateID)
	}

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

// GetTrending returns trending templates using rating and download metrics
// DynamoDB integration: Query analytics table with timeframe-based aggregations
func (r *Registry) GetTrending(timeframe string) ([]*CommunityTemplate, error) {
	var trending []*CommunityTemplate

	for _, template := range r.templateCache {
		// Trending algorithm: rating × download velocity
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

// PublishTemplate publishes a template to the marketplace using DynamoDB
func (r *Registry) PublishTemplate(template *TemplatePublication) (*PublicationResult, error) {
	ctx := context.Background()

	// Generate unique template ID
	templateID := r.generateTemplateID(template.Name)

	// Create community template from publication
	communityTemplate, err := r.createCommunityTemplate(templateID, template)
	if err != nil {
		return nil, fmt.Errorf("failed to create community template: %w", err)
	}

	// If DynamoDB client is configured, store in DynamoDB
	if r.dynamoClient != nil {
		// Marshal template to DynamoDB attribute values
		item, err := attributevalue.MarshalMap(communityTemplate)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal template: %w", err)
		}

		// Put item in DynamoDB
		putInput := &dynamodb.PutItemInput{
			TableName: aws.String(r.templatesTable),
			Item:      item,
		}

		if _, err := r.dynamoClient.PutItem(ctx, putInput); err != nil {
			return nil, fmt.Errorf("DynamoDB PutItem failed: %w", err)
		}
	}

	// Also store in cache for performance
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

	// Update cache
	// DynamoDB integration: UpdateItem on cloudworkstation-templates table
	r.templateCache[templateID] = template

	return nil
}

// UnpublishTemplate removes a template from the marketplace
func (r *Registry) UnpublishTemplate(templateID string) error {
	if _, exists := r.templateCache[templateID]; !exists {
		return fmt.Errorf("template not found: %s", templateID)
	}

	// Remove from cache
	// DynamoDB integration: UpdateItem to set visibility=unpublished
	delete(r.templateCache, templateID)

	return nil
}

// GetUserPublications returns templates published by a specific user
func (r *Registry) GetUserPublications(userID string) ([]*CommunityTemplate, error) {
	publications := make([]*CommunityTemplate, 0)

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

// AddReview adds a review for a template to DynamoDB
func (r *Registry) AddReview(templateID string, review *TemplateReview) error {
	ctx := context.Background()

	// Verify template exists before adding review
	template, err := r.GetTemplate(templateID)
	if err != nil || template == nil {
		return fmt.Errorf("template '%s' not found", templateID)
	}

	// Generate review ID if not provided
	if review.ReviewID == "" {
		review.ReviewID = fmt.Sprintf("review-%s-%d", templateID, time.Now().Unix())
	}
	review.TemplateID = templateID
	review.CreatedAt = time.Now()

	// If DynamoDB client is configured, store review
	if r.dynamoClient != nil {
		// Marshal review to DynamoDB
		item, err := attributevalue.MarshalMap(review)
		if err != nil {
			return fmt.Errorf("failed to marshal review: %w", err)
		}

		// Put review in DynamoDB
		putInput := &dynamodb.PutItemInput{
			TableName: aws.String(r.reviewsTable),
			Item:      item,
		}

		if _, err := r.dynamoClient.PutItem(ctx, putInput); err != nil {
			return fmt.Errorf("DynamoDB PutItem failed: %w", err)
		}

		// Update template rating in templates table
		updateExpr := "SET review_count = review_count + :inc, rating = :newRating"
		exprAttrValues := map[string]types.AttributeValue{
			":inc":       &types.AttributeValueMemberN{Value: "1"},
			":newRating": &types.AttributeValueMemberN{Value: fmt.Sprintf("%.2f", float64(review.Rating))},
		}

		updateInput := &dynamodb.UpdateItemInput{
			TableName: aws.String(r.templatesTable),
			Key: map[string]types.AttributeValue{
				"template_id": &types.AttributeValueMemberS{Value: templateID},
			},
			UpdateExpression:          aws.String(updateExpr),
			ExpressionAttributeValues: exprAttrValues,
		}

		if _, err := r.dynamoClient.UpdateItem(ctx, updateInput); err != nil {
			// Log error but don't fail the review submission
			fmt.Printf("Warning: failed to update template rating: %v\n", err)
		}
	}

	// Update cache if template exists
	if template, exists := r.templateCache[templateID]; exists {
		r.updateRatingMetrics(template, review.Rating)
	}

	return nil
}

// GetReviews retrieves reviews for a template with pagination from DynamoDB
func (r *Registry) GetReviews(templateID string, pagination *ReviewPagination) (*ReviewResponse, error) {
	ctx := context.Background()

	// If DynamoDB client is configured, query reviews
	if r.dynamoClient != nil {
		// Query reviews using template_id GSI
		queryInput := &dynamodb.QueryInput{
			TableName:              aws.String(r.reviewsTable),
			IndexName:              aws.String("template_id-index"),
			KeyConditionExpression: aws.String("template_id = :templateID"),
			ExpressionAttributeValues: map[string]types.AttributeValue{
				":templateID": &types.AttributeValueMemberS{Value: templateID},
			},
			Limit:            aws.Int32(int32(pagination.Limit)),
			ScanIndexForward: aws.Bool(false), // Newest first
		}

		result, err := r.dynamoClient.Query(ctx, queryInput)
		if err != nil {
			return nil, fmt.Errorf("DynamoDB Query failed: %w", err)
		}

		// Unmarshal reviews
		var reviews []*TemplateReview
		for _, item := range result.Items {
			var review TemplateReview
			if err := attributevalue.UnmarshalMap(item, &review); err != nil {
				continue // Skip malformed items
			}
			reviews = append(reviews, &review)
		}

		// Calculate pagination info
		totalCount := len(reviews)
		limit := pagination.Limit
		if limit == 0 {
			limit = 10
		}

		response := &ReviewResponse{
			Reviews:    reviews,
			TotalCount: totalCount,
			Page:       (pagination.Offset / limit) + 1,
			TotalPages: (totalCount + limit - 1) / limit,
			HasMore:    result.LastEvaluatedKey != nil,
		}

		return response, nil
	}

	// Fallback to mock reviews for development
	mockReviews := r.generateMockReviews(templateID)

	start := 0
	if pagination.Offset > 0 {
		start = pagination.Offset
	}

	limit := 10
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

// TrackUsage tracks usage events for analytics in DynamoDB
func (r *Registry) TrackUsage(templateID string, event *UsageEvent) error {
	ctx := context.Background()

	// Ensure event has required fields
	event.TemplateID = templateID
	event.Timestamp = time.Now()

	// If DynamoDB client is configured, store analytics event
	if r.dynamoClient != nil {
		// Marshal event to DynamoDB
		item, err := attributevalue.MarshalMap(event)
		if err != nil {
			return fmt.Errorf("failed to marshal usage event: %w", err)
		}

		// Add composite key for efficient querying
		// PK: template_id, SK: timestamp
		eventID := fmt.Sprintf("%s#%d", templateID, event.Timestamp.Unix())
		item["event_id"] = &types.AttributeValueMemberS{Value: eventID}

		// Put event in DynamoDB
		putInput := &dynamodb.PutItemInput{
			TableName: aws.String(r.analyticsTable),
			Item:      item,
		}

		if _, err := r.dynamoClient.PutItem(ctx, putInput); err != nil {
			// Log error but don't fail tracking
			fmt.Printf("Warning: failed to track usage in DynamoDB: %v\n", err)
		}
	}

	// Update local cache metrics
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
		Author:            "current-user", // Integration: Replace with authenticated user ID from request context
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
	r.templateCache[templateID] = originalTemplate

	return forkedTemplate, nil
}

// GetTemplateAnalytics returns comprehensive analytics for a template
// DynamoDB integration: Query cloudworkstation-analytics with aggregation functions
func (r *Registry) GetTemplateAnalytics(templateID string) (*TemplateAnalytics, error) {
	template, exists := r.templateCache[templateID]
	if !exists {
		return nil, fmt.Errorf("template not found: %s", templateID)
	}

	// Aggregate analytics from current metrics
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

	// Build usage statistics from current metrics
	// DynamoDB integration: Query analytics table with time-range filter
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

	// Check if template has all required tags
	if len(query.Tags) > 0 {
		templateTags := make(map[string]bool)
		for _, tag := range template.Tags {
			templateTags[strings.ToLower(tag)] = true
		}

		for _, requiredTag := range query.Tags {
			if !templateTags[strings.ToLower(requiredTag)] {
				return false // Template doesn't have this required tag
			}
		}
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
	// Create community template from publication metadata
	template := &CommunityTemplate{
		TemplateID:        templateID,
		Name:              publication.Name,
		Description:       publication.Description,
		Author:            "current-user", // Integration: Replace with authenticated user ID from request context
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
	// Generate AMI creation job IDs for tracking
	// Integration point: pkg/ami system for actual AMI generation
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

	// Update the cache with modified template
	r.templateCache[template.TemplateID] = template
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

	// Update the cache with modified template
	r.templateCache[templateID] = template
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
