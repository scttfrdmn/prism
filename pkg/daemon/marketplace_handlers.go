// Marketplace operations handlers for Template Marketplace Integration (Phase 5.1 Week 3)
package daemon

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/scttfrdmn/cloudworkstation/pkg/marketplace"
)

// RegisterMarketplaceRoutes registers all marketplace-related API routes
func (s *Server) RegisterMarketplaceRoutes(mux *http.ServeMux, applyMiddleware func(http.HandlerFunc) http.HandlerFunc) {
	// Discovery endpoints
	mux.HandleFunc("/api/v1/marketplace/templates", applyMiddleware(s.handleMarketplaceTemplates))
	mux.HandleFunc("/api/v1/marketplace/templates/", applyMiddleware(s.handleMarketplateTemplate))
	mux.HandleFunc("/api/v1/marketplace/categories", applyMiddleware(s.handleMarketplaceCategories))
	mux.HandleFunc("/api/v1/marketplace/featured", applyMiddleware(s.handleMarketplaceFeatured))
	mux.HandleFunc("/api/v1/marketplace/trending", applyMiddleware(s.handleMarketplaceTrending))

	// Publishing endpoints
	mux.HandleFunc("/api/v1/marketplace/publish", applyMiddleware(s.handleMarketplacePublish))
	mux.HandleFunc("/api/v1/marketplace/unpublish/", applyMiddleware(s.handleMarketplaceUnpublish))
	mux.HandleFunc("/api/v1/marketplace/update/", applyMiddleware(s.handleMarketplaceUpdate))
	mux.HandleFunc("/api/v1/marketplace/my-publications", applyMiddleware(s.handleMyPublications))

	// Community interaction endpoints
	mux.HandleFunc("/api/v1/marketplace/reviews/", applyMiddleware(s.handleMarketplaceReviews))
	mux.HandleFunc("/api/v1/marketplace/fork/", applyMiddleware(s.handleMarketplaceFork))
	mux.HandleFunc("/api/v1/marketplace/analytics/", applyMiddleware(s.handleMarketplaceAnalytics))
}

// handleMarketplaceTemplates handles template search/browse requests
// GET /api/v1/marketplace/templates?query=ml&category=machine-learning&limit=20
func (s *Server) handleMarketplaceTemplates(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	// Parse query parameters
	query := marketplace.SearchQuery{
		Query:        r.URL.Query().Get("query"),
		Category:     r.URL.Query().Get("category"),
		Author:       r.URL.Query().Get("author"),
		Architecture: r.URL.Query().Get("architecture"),
		Region:       r.URL.Query().Get("region"),
		SortBy:       r.URL.Query().Get("sort_by"),
		SortOrder:    r.URL.Query().Get("sort_order"),
	}

	// Parse tags
	if tagsStr := r.URL.Query().Get("tags"); tagsStr != "" {
		query.Tags = strings.Split(tagsStr, ",")
	}

	// Parse keywords
	if keywordsStr := r.URL.Query().Get("keywords"); keywordsStr != "" {
		query.Keywords = strings.Split(keywordsStr, ",")
	}

	// Parse numeric filters
	if minRatingStr := r.URL.Query().Get("min_rating"); minRatingStr != "" {
		if minRating, err := strconv.ParseFloat(minRatingStr, 64); err == nil {
			query.MinRating = minRating
		}
	}

	if minDownloadsStr := r.URL.Query().Get("min_downloads"); minDownloadsStr != "" {
		if minDownloads, err := strconv.Atoi(minDownloadsStr); err == nil {
			query.MinDownloads = minDownloads
		}
	}

	// Parse boolean filters
	query.VerifiedOnly = r.URL.Query().Get("verified_only") == "true"
	query.FeaturedOnly = r.URL.Query().Get("featured_only") == "true"
	query.AMIAvailable = r.URL.Query().Get("ami_available") == "true"

	// Parse pagination
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if limit, err := strconv.Atoi(limitStr); err == nil {
			query.Limit = limit
		}
	}

	if offsetStr := r.URL.Query().Get("offset"); offsetStr != "" {
		if offset, err := strconv.Atoi(offsetStr); err == nil {
			query.Offset = offset
		}
	}

	// Search templates using marketplace registry
	templates, err := s.marketplaceRegistry.SearchTemplates(query)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, fmt.Sprintf("search failed: %v", err))
		return
	}

	// Create response
	response := map[string]interface{}{
		"templates":   templates,
		"total_count": len(templates),
		"query":       query.Query,
		"category":    query.Category,
		"has_more":    len(templates) == query.Limit, // Simple approximation
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		s.writeError(w, http.StatusInternalServerError, err.Error())
	}
}

// handleMarketplaceTemplate handles individual template requests
// GET /api/v1/marketplace/templates/{template_id}
func (s *Server) handleMarketplateTemplate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	// Extract template ID from URL path
	pathParts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(pathParts) < 4 || pathParts[3] == "" {
		s.writeError(w, http.StatusBadRequest, "template ID required in URL path")
		return
	}

	templateID := pathParts[3]

	// Get template from marketplace registry
	template, err := s.marketplaceRegistry.GetTemplate(templateID)
	if err != nil {
		s.writeError(w, http.StatusNotFound, fmt.Sprintf("template not found: %v", err))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(template); err != nil {
		s.writeError(w, http.StatusInternalServerError, err.Error())
	}
}

// handleMarketplaceCategories lists available template categories
// GET /api/v1/marketplace/categories
func (s *Server) handleMarketplaceCategories(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	categories, err := s.marketplaceRegistry.ListCategories()
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, fmt.Sprintf("failed to list categories: %v", err))
		return
	}

	response := map[string]interface{}{
		"categories": categories,
		"total":      len(categories),
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		s.writeError(w, http.StatusInternalServerError, err.Error())
	}
}

// handleMarketplaceFeatured returns featured templates
// GET /api/v1/marketplace/featured
func (s *Server) handleMarketplaceFeatured(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	featured, err := s.marketplaceRegistry.GetFeatured()
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, fmt.Sprintf("failed to get featured templates: %v", err))
		return
	}

	response := map[string]interface{}{
		"templates": featured,
		"count":     len(featured),
		"updated":   "recently", // Could be actual timestamp
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		s.writeError(w, http.StatusInternalServerError, err.Error())
	}
}

// handleMarketplaceTrending returns trending templates
// GET /api/v1/marketplace/trending?timeframe=week
func (s *Server) handleMarketplaceTrending(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	timeframe := r.URL.Query().Get("timeframe")
	if timeframe == "" {
		timeframe = "week" // Default timeframe
	}

	trending, err := s.marketplaceRegistry.GetTrending(timeframe)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, fmt.Sprintf("failed to get trending templates: %v", err))
		return
	}

	response := map[string]interface{}{
		"templates": trending,
		"count":     len(trending),
		"timeframe": timeframe,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		s.writeError(w, http.StatusInternalServerError, err.Error())
	}
}

// handleMarketplacePublish publishes a template to the marketplace
// POST /api/v1/marketplace/publish
func (s *Server) handleMarketplacePublish(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		s.writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var publication marketplace.TemplatePublication
	if err := json.NewDecoder(r.Body).Decode(&publication); err != nil {
		s.writeError(w, http.StatusBadRequest, fmt.Sprintf("invalid request body: %v", err))
		return
	}

	// Validate required fields
	if publication.Name == "" {
		s.writeError(w, http.StatusBadRequest, "template name is required")
		return
	}
	if publication.Description == "" {
		s.writeError(w, http.StatusBadRequest, "template description is required")
		return
	}
	if publication.Category == "" {
		s.writeError(w, http.StatusBadRequest, "template category is required")
		return
	}

	// Publish template using marketplace registry
	result, err := s.marketplaceRegistry.PublishTemplate(&publication)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, fmt.Sprintf("publication failed: %v", err))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(result); err != nil {
		s.writeError(w, http.StatusInternalServerError, err.Error())
	}
}

// handleMarketplaceUpdate updates a published template
// PUT /api/v1/marketplace/update/{template_id}
func (s *Server) handleMarketplaceUpdate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		s.writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	// Extract template ID from URL path
	pathParts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(pathParts) < 4 || pathParts[3] == "" {
		s.writeError(w, http.StatusBadRequest, "template ID required in URL path")
		return
	}

	templateID := pathParts[3]

	var update marketplace.TemplateUpdate
	if err := json.NewDecoder(r.Body).Decode(&update); err != nil {
		s.writeError(w, http.StatusBadRequest, fmt.Sprintf("invalid request body: %v", err))
		return
	}

	// Update template using marketplace registry
	if err := s.marketplaceRegistry.UpdateTemplate(templateID, &update); err != nil {
		s.writeError(w, http.StatusInternalServerError, fmt.Sprintf("update failed: %v", err))
		return
	}

	response := map[string]interface{}{
		"template_id": templateID,
		"status":      "updated",
		"message":     "Template updated successfully",
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		s.writeError(w, http.StatusInternalServerError, err.Error())
	}
}

// handleMarketplaceUnpublish unpublishes a template
// DELETE /api/v1/marketplace/unpublish/{template_id}
func (s *Server) handleMarketplaceUnpublish(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		s.writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	// Extract template ID from URL path
	pathParts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(pathParts) < 4 || pathParts[3] == "" {
		s.writeError(w, http.StatusBadRequest, "template ID required in URL path")
		return
	}

	templateID := pathParts[3]

	// Unpublish template using marketplace registry
	if err := s.marketplaceRegistry.UnpublishTemplate(templateID); err != nil {
		s.writeError(w, http.StatusInternalServerError, fmt.Sprintf("unpublish failed: %v", err))
		return
	}

	response := map[string]interface{}{
		"template_id": templateID,
		"status":      "unpublished",
		"message":     "Template unpublished successfully",
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		s.writeError(w, http.StatusInternalServerError, err.Error())
	}
}

// handleMyPublications returns templates published by the current user
// GET /api/v1/marketplace/my-publications
func (s *Server) handleMyPublications(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	// In production, this would get the authenticated user ID
	userID := "current-user" // Placeholder

	publications, err := s.marketplaceRegistry.GetUserPublications(userID)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, fmt.Sprintf("failed to get publications: %v", err))
		return
	}

	response := map[string]interface{}{
		"publications": publications,
		"count":        len(publications),
		"user_id":      userID,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		s.writeError(w, http.StatusInternalServerError, err.Error())
	}
}

// handleMarketplaceReviews handles reviews for templates
// GET /api/v1/marketplace/reviews/{template_id}?limit=10&offset=0
// POST /api/v1/marketplace/reviews/{template_id}
func (s *Server) handleMarketplaceReviews(w http.ResponseWriter, r *http.Request) {
	// Extract template ID from URL path
	pathParts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(pathParts) < 4 || pathParts[3] == "" {
		s.writeError(w, http.StatusBadRequest, "template ID required in URL path")
		return
	}

	templateID := pathParts[3]

	switch r.Method {
	case http.MethodGet:
		s.handleGetReviews(w, r, templateID)
	case http.MethodPost:
		s.handleAddReview(w, r, templateID)
	default:
		s.writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

// handleGetReviews gets reviews for a template
func (s *Server) handleGetReviews(w http.ResponseWriter, r *http.Request, templateID string) {
	// Parse pagination parameters
	pagination := &marketplace.ReviewPagination{}

	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if limit, err := strconv.Atoi(limitStr); err == nil {
			pagination.Limit = limit
		}
	}

	if offsetStr := r.URL.Query().Get("offset"); offsetStr != "" {
		if offset, err := strconv.Atoi(offsetStr); err == nil {
			pagination.Offset = offset
		}
	}

	pagination.SortBy = r.URL.Query().Get("sort_by")

	// Get reviews using marketplace registry
	reviewResponse, err := s.marketplaceRegistry.GetReviews(templateID, pagination)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, fmt.Sprintf("failed to get reviews: %v", err))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(reviewResponse); err != nil {
		s.writeError(w, http.StatusInternalServerError, err.Error())
	}
}

// handleAddReview adds a review for a template
func (s *Server) handleAddReview(w http.ResponseWriter, r *http.Request, templateID string) {
	var review marketplace.TemplateReview
	if err := json.NewDecoder(r.Body).Decode(&review); err != nil {
		s.writeError(w, http.StatusBadRequest, fmt.Sprintf("invalid request body: %v", err))
		return
	}

	// Validate required fields
	if review.Rating < 1 || review.Rating > 5 {
		s.writeError(w, http.StatusBadRequest, "rating must be between 1 and 5")
		return
	}
	if review.Title == "" {
		s.writeError(w, http.StatusBadRequest, "review title is required")
		return
	}
	if review.Content == "" {
		s.writeError(w, http.StatusBadRequest, "review content is required")
		return
	}

	// Set template ID and reviewer info
	review.TemplateID = templateID
	review.Reviewer = "current-user"     // In production, get from authentication
	review.ReviewerName = "Current User" // In production, get from user profile

	// Add review using marketplace registry
	if err := s.marketplaceRegistry.AddReview(templateID, &review); err != nil {
		s.writeError(w, http.StatusInternalServerError, fmt.Sprintf("failed to add review: %v", err))
		return
	}

	response := map[string]interface{}{
		"template_id": templateID,
		"review_id":   fmt.Sprintf("review-%s-%d", templateID, review.Rating),
		"status":      "added",
		"message":     "Review added successfully",
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		s.writeError(w, http.StatusInternalServerError, err.Error())
	}
}

// handleMarketplaceFork handles template forking
// POST /api/v1/marketplace/fork/{template_id}
func (s *Server) handleMarketplaceFork(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		s.writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	// Extract template ID from URL path
	pathParts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(pathParts) < 4 || pathParts[3] == "" {
		s.writeError(w, http.StatusBadRequest, "template ID required in URL path")
		return
	}

	templateID := pathParts[3]

	var fork marketplace.TemplateFork
	if err := json.NewDecoder(r.Body).Decode(&fork); err != nil {
		s.writeError(w, http.StatusBadRequest, fmt.Sprintf("invalid request body: %v", err))
		return
	}

	// Validate required fields
	if fork.NewName == "" {
		s.writeError(w, http.StatusBadRequest, "new template name is required")
		return
	}
	if fork.NewDescription == "" {
		s.writeError(w, http.StatusBadRequest, "new template description is required")
		return
	}

	// Fork template using marketplace registry
	forkedTemplate, err := s.marketplaceRegistry.ForkTemplate(templateID, &fork)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, fmt.Sprintf("fork failed: %v", err))
		return
	}

	response := map[string]interface{}{
		"original_template_id": templateID,
		"forked_template_id":   forkedTemplate.TemplateID,
		"forked_template_name": forkedTemplate.Name,
		"status":               "forked",
		"message":              "Template forked successfully",
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		s.writeError(w, http.StatusInternalServerError, err.Error())
	}
}

// handleMarketplaceAnalytics provides analytics for templates
// GET /api/v1/marketplace/analytics/{template_id}?timeframe=week
func (s *Server) handleMarketplaceAnalytics(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	// Extract template ID from URL path
	pathParts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(pathParts) < 4 || pathParts[3] == "" {
		s.writeError(w, http.StatusBadRequest, "template ID required in URL path")
		return
	}

	templateID := pathParts[3]
	timeframe := r.URL.Query().Get("timeframe")
	if timeframe == "" {
		timeframe = "week" // Default timeframe
	}

	// Get analytics based on query type
	if timeframe == "overview" {
		// Get comprehensive analytics
		analytics, err := s.marketplaceRegistry.GetTemplateAnalytics(templateID)
		if err != nil {
			s.writeError(w, http.StatusInternalServerError, fmt.Sprintf("failed to get analytics: %v", err))
			return
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(analytics); err != nil {
			s.writeError(w, http.StatusInternalServerError, err.Error())
		}
	} else {
		// Get usage statistics for specific timeframe
		stats, err := s.marketplaceRegistry.GetUsageStats(templateID, timeframe)
		if err != nil {
			s.writeError(w, http.StatusInternalServerError, fmt.Sprintf("failed to get usage stats: %v", err))
			return
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(stats); err != nil {
			s.writeError(w, http.StatusInternalServerError, err.Error())
		}
	}
}
