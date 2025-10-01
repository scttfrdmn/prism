package cli

import (
	"fmt"
	"strconv"
	"strings"
)

// Marketplace processes marketplace-related commands
func (a *App) Marketplace(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("missing marketplace command (list, search, info, publish, review, fork, featured, trending, categories)")
	}

	subcommand := args[0]
	subargs := args[1:]

	switch subcommand {
	case "list":
		return a.handleMarketplaceList(subargs)
	case "search":
		return a.handleMarketplaceSearch(subargs)
	case "info":
		return a.handleMarketplaceInfo(subargs)
	case "publish":
		return a.handleMarketplacePublish(subargs)
	case "review":
		return a.handleMarketplaceReview(subargs)
	case "fork":
		return a.handleMarketplaceFork(subargs)
	case "featured":
		return a.handleMarketplaceFeatured(subargs)
	case "trending":
		return a.handleMarketplaceTrending(subargs)
	case "categories":
		return a.handleMarketplaceCategories(subargs)
	case "my-publications":
		return a.handleMyPublications(subargs)
	default:
		return fmt.Errorf("unknown marketplace command: %s", subcommand)
	}
}

// handleMarketplaceList lists community templates
func (a *App) handleMarketplaceList(args []string) error {
	cmdArgs := parseCmdArgs(args)

	// Build query parameters
	queryParams := make(map[string]string)
	if category := cmdArgs["category"]; category != "" {
		queryParams["category"] = category
	}
	if architecture := cmdArgs["architecture"]; architecture != "" {
		queryParams["architecture"] = architecture
	}
	if tags := cmdArgs["tags"]; tags != "" {
		queryParams["tags"] = tags
	}
	if verified := cmdArgs["verified-only"]; verified == "true" {
		queryParams["verified_only"] = "true"
	}
	if featured := cmdArgs["featured-only"]; featured == "true" {
		queryParams["featured_only"] = "true"
	}
	if ami := cmdArgs["ami-available"]; ami == "true" {
		queryParams["ami_available"] = "true"
	}
	if limit := cmdArgs["limit"]; limit != "" {
		queryParams["limit"] = limit
	} else {
		queryParams["limit"] = "20" // Default limit
	}

	// Make API request
	endpoint := "/api/v1/marketplace/templates"
	if len(queryParams) > 0 {
		endpoint += "?"
		var params []string
		for key, value := range queryParams {
			params = append(params, fmt.Sprintf("%s=%s", key, value))
		}
		endpoint += strings.Join(params, "&")
	}

	response, err := a.makeAPIRequest("GET", endpoint, nil)
	if err != nil {
		return fmt.Errorf("failed to list templates: %w", err)
	}

	// Parse response
	templates, ok := response["templates"].([]interface{})
	if !ok {
		return fmt.Errorf("invalid response format")
	}

	if len(templates) == 0 {
		fmt.Printf("📋 No templates found matching the criteria\n\n")
		fmt.Printf("💡 Try different filters or browse featured templates: cws marketplace featured\n")
		return nil
	}

	// Display templates
	fmt.Printf("🏪 Community Templates (%d results)\n\n", len(templates))

	for i, tmpl := range templates {
		template := tmpl.(map[string]interface{})
		fmt.Printf("📦 %d. %s\n", i+1, getString(template, "name"))
		fmt.Printf("   📝 %s\n", getString(template, "description"))
		fmt.Printf("   👤 Author: %s\n", getString(template, "author_name"))
		fmt.Printf("   🏷️  Category: %s | Version: %s\n",
			getString(template, "category"), getString(template, "version"))

		// Rating and stats
		rating := getFloat64(template, "rating")
		reviewCount := getInt(template, "review_count")
		downloadCount := getInt(template, "download_count")

		if rating > 0 {
			stars := strings.Repeat("⭐", int(rating))
			fmt.Printf("   %s %.1f (%d reviews) | 📥 %d downloads\n",
				stars, rating, reviewCount, downloadCount)
		} else {
			fmt.Printf("   ⭐ No ratings yet | 📥 %d downloads\n", downloadCount)
		}

		// Verification badges
		if getBool(template, "verified") {
			fmt.Printf("   ✅ Verified")
		}
		if getBool(template, "featured") {
			fmt.Printf("   🌟 Featured")
		}
		if ami, exists := template["ami_info"]; exists && ami != nil {
			amiInfo := ami.(map[string]interface{})
			if getBool(amiInfo, "available") {
				fmt.Printf("   🚀 AMI Available")
			}
		}
		fmt.Printf("\n")
		fmt.Printf("   💻 Launch: cws launch marketplace:%s my-project\n",
			getString(template, "template_id"))
		fmt.Printf("\n")
	}

	// Show pagination info if there might be more results
	if len(templates) == 20 { // Default limit
		fmt.Printf("💡 Use --limit <number> to see more results or --offset <number> for pagination\n")
	}

	return nil
}

// handleMarketplaceSearch searches for templates
func (a *App) handleMarketplaceSearch(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: cws marketplace search <query> [--category <category>] [--tags <tags>]")
	}

	query := args[0]
	cmdArgs := parseCmdArgs(args[1:])

	// Build query parameters
	queryParams := map[string]string{
		"query": query,
		"limit": "20",
	}

	if category := cmdArgs["category"]; category != "" {
		queryParams["category"] = category
	}
	if tags := cmdArgs["tags"]; tags != "" {
		queryParams["tags"] = tags
	}
	if architecture := cmdArgs["architecture"]; architecture != "" {
		queryParams["architecture"] = architecture
	}
	if minRating := cmdArgs["min-rating"]; minRating != "" {
		queryParams["min_rating"] = minRating
	}

	// Build endpoint URL
	endpoint := "/api/v1/marketplace/templates?"
	var params []string
	for key, value := range queryParams {
		params = append(params, fmt.Sprintf("%s=%s", key, value))
	}
	endpoint += strings.Join(params, "&")

	response, err := a.makeAPIRequest("GET", endpoint, nil)
	if err != nil {
		return fmt.Errorf("failed to search templates: %w", err)
	}

	// Parse and display results (reuse list display logic)
	templates, ok := response["templates"].([]interface{})
	if !ok {
		return fmt.Errorf("invalid response format")
	}

	if len(templates) == 0 {
		fmt.Printf("🔍 No templates found for query: '%s'\n\n", query)
		fmt.Printf("💡 Try different search terms or browse categories: cws marketplace categories\n")
		return nil
	}

	fmt.Printf("🔍 Search Results for '%s' (%d found)\n\n", query, len(templates))

	// Display using same format as list
	return a.displayTemplateList(templates)
}

// handleMarketplaceInfo shows detailed information about a template
func (a *App) handleMarketplaceInfo(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: cws marketplace info <template-id>")
	}

	templateID := args[0]
	endpoint := fmt.Sprintf("/api/v1/marketplace/templates/%s", templateID)

	response, err := a.makeAPIRequest("GET", endpoint, nil)
	if err != nil {
		return fmt.Errorf("failed to get template info: %w", err)
	}

	// Display detailed template information
	a.displayTemplateInfo(response)

	return nil
}

// handleMarketplacePublish publishes a template to the marketplace
func (a *App) handleMarketplacePublish(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: cws marketplace publish <instance-name> --name <template-name> --category <category> [options]")
	}

	instanceName := args[0]
	cmdArgs := parseCmdArgs(args[1:])

	// Validate required parameters
	name := cmdArgs["name"]
	if name == "" {
		return fmt.Errorf("template name is required (use --name <name>)")
	}

	category := cmdArgs["category"]
	if category == "" {
		return fmt.Errorf("category is required (use --category <category>)")
	}

	description := cmdArgs["description"]
	if description == "" {
		return fmt.Errorf("description is required (use --description <description>)")
	}

	// Build publication request
	publication := map[string]interface{}{
		"source_instance_id": instanceName,
		"name":               name,
		"description":        description,
		"category":           category,
		"visibility":         cmdArgs["visibility"], // "public", "private"
		"license":            cmdArgs["license"],    // "MIT", "Apache-2.0", etc.
		"generate_ami":       cmdArgs["generate-ami"] == "true",
		"documentation":      cmdArgs["documentation"],
		"video_demo":         cmdArgs["video-demo"],
		"paper_doi":          cmdArgs["paper-doi"],
		"funding_source":     cmdArgs["funding"],
	}

	// Handle tags
	if tags := cmdArgs["tags"]; tags != "" {
		publication["tags"] = strings.Split(tags, ",")
	}

	// Handle target regions for AMI generation
	if regions := cmdArgs["regions"]; regions != "" {
		publication["target_regions"] = strings.Split(regions, ",")
	}

	fmt.Printf("🚀 Publishing template '%s' to marketplace...\n\n", name)

	response, err := a.makeAPIRequest("POST", "/api/v1/marketplace/publish", publication)
	if err != nil {
		return fmt.Errorf("failed to publish template: %w", err)
	}

	// Display publication result
	fmt.Printf("✅ Template published successfully!\n\n")
	fmt.Printf("🆔 Template ID: %s\n", getString(response, "template_id"))
	fmt.Printf("📝 Name: %s\n", name)
	fmt.Printf("🏷️  Category: %s\n", category)
	fmt.Printf("⚡ Status: %s\n", getString(response, "status"))
	fmt.Printf("🌐 URL: %s\n", getString(response, "publication_url"))

	if amiIDs := response["ami_creation_ids"]; amiIDs != nil {
		if ids, ok := amiIDs.([]interface{}); ok && len(ids) > 0 {
			fmt.Printf("\n🖼️  AMI creation initiated:\n")
			for _, id := range ids {
				fmt.Printf("   • %s\n", id)
			}
			fmt.Printf("\n💡 Check AMI creation status: cws ami status <creation-id>\n")
		}
	}

	fmt.Printf("\n💡 View your template: cws marketplace info %s\n", getString(response, "template_id"))
	fmt.Printf("💡 Launch your template: cws launch marketplace:%s my-project\n", getString(response, "template_id"))

	return nil
}

// handleMarketplaceReview adds a review for a template
func (a *App) handleMarketplaceReview(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: cws marketplace review <template-id> --rating <1-5> --title <title> --comment <comment>")
	}

	templateID := args[0]
	cmdArgs := parseCmdArgs(args[1:])

	// Validate required parameters
	ratingStr := cmdArgs["rating"]
	if ratingStr == "" {
		return fmt.Errorf("rating is required (use --rating <1-5>)")
	}

	rating, err := strconv.Atoi(ratingStr)
	if err != nil || rating < 1 || rating > 5 {
		return fmt.Errorf("rating must be between 1 and 5")
	}

	title := cmdArgs["title"]
	if title == "" {
		return fmt.Errorf("review title is required (use --title <title>)")
	}

	comment := cmdArgs["comment"]
	if comment == "" {
		return fmt.Errorf("review comment is required (use --comment <comment>)")
	}

	// Build review request
	review := map[string]interface{}{
		"rating":   rating,
		"title":    title,
		"content":  comment,
		"use_case": cmdArgs["use-case"],
	}

	endpoint := fmt.Sprintf("/api/v1/marketplace/reviews/%s", templateID)

	fmt.Printf("⭐ Adding review for template...\n\n")

	response, err := a.makeAPIRequest("POST", endpoint, review)
	if err != nil {
		return fmt.Errorf("failed to add review: %w", err)
	}

	// Display review result
	stars := strings.Repeat("⭐", rating)
	fmt.Printf("✅ Review added successfully!\n\n")
	fmt.Printf("📦 Template: %s\n", templateID)
	fmt.Printf("⭐ Rating: %s (%d/5)\n", stars, rating)
	fmt.Printf("📝 Title: %s\n", title)
	fmt.Printf("💬 Comment: %s\n", comment)
	fmt.Printf("🆔 Review ID: %s\n", getString(response, "review_id"))

	return nil
}

// handleMarketplaceFork forks a template
func (a *App) handleMarketplaceFork(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: cws marketplace fork <template-id> --name <new-name> --description <new-description>")
	}

	templateID := args[0]
	cmdArgs := parseCmdArgs(args[1:])

	// Validate required parameters
	newName := cmdArgs["name"]
	if newName == "" {
		return fmt.Errorf("new template name is required (use --name <name>)")
	}

	newDescription := cmdArgs["description"]
	if newDescription == "" {
		return fmt.Errorf("new template description is required (use --description <description>)")
	}

	// Build fork request
	fork := map[string]interface{}{
		"new_name":        newName,
		"new_description": newDescription,
		"private":         cmdArgs["private"] == "true",
	}

	endpoint := fmt.Sprintf("/api/v1/marketplace/fork/%s", templateID)

	fmt.Printf("🍴 Forking template...\n\n")

	response, err := a.makeAPIRequest("POST", endpoint, fork)
	if err != nil {
		return fmt.Errorf("failed to fork template: %w", err)
	}

	// Display fork result
	fmt.Printf("✅ Template forked successfully!\n\n")
	fmt.Printf("🆔 Original: %s\n", getString(response, "original_template_id"))
	fmt.Printf("🆔 Forked: %s\n", getString(response, "forked_template_id"))
	fmt.Printf("📝 Name: %s\n", getString(response, "forked_template_name"))

	fmt.Printf("\n💡 Launch your fork: cws launch marketplace:%s my-project\n",
		getString(response, "forked_template_id"))

	return nil
}

// handleMarketplaceFeatured shows featured templates
func (a *App) handleMarketplaceFeatured(args []string) error {
	response, err := a.makeAPIRequest("GET", "/api/v1/marketplace/featured", nil)
	if err != nil {
		return fmt.Errorf("failed to get featured templates: %w", err)
	}

	templates, ok := response["templates"].([]interface{})
	if !ok {
		return fmt.Errorf("invalid response format")
	}

	if len(templates) == 0 {
		fmt.Printf("🌟 No featured templates currently available\n")
		return nil
	}

	fmt.Printf("🌟 Featured Templates (%d)\n\n", len(templates))
	return a.displayTemplateList(templates)
}

// handleMarketplaceTrending shows trending templates
func (a *App) handleMarketplaceTrending(args []string) error {
	cmdArgs := parseCmdArgs(args)
	timeframe := cmdArgs["timeframe"]
	if timeframe == "" {
		timeframe = "week"
	}

	endpoint := fmt.Sprintf("/api/v1/marketplace/trending?timeframe=%s", timeframe)
	response, err := a.makeAPIRequest("GET", endpoint, nil)
	if err != nil {
		return fmt.Errorf("failed to get trending templates: %w", err)
	}

	templates, ok := response["templates"].([]interface{})
	if !ok {
		return fmt.Errorf("invalid response format")
	}

	if len(templates) == 0 {
		fmt.Printf("📈 No trending templates for timeframe: %s\n", timeframe)
		return nil
	}

	fmt.Printf("📈 Trending Templates (%s) - %d found\n\n", timeframe, len(templates))
	return a.displayTemplateList(templates)
}

// handleMarketplaceCategories lists available categories
func (a *App) handleMarketplaceCategories(args []string) error {
	response, err := a.makeAPIRequest("GET", "/api/v1/marketplace/categories", nil)
	if err != nil {
		return fmt.Errorf("failed to get categories: %w", err)
	}

	categories, ok := response["categories"].([]interface{})
	if !ok {
		return fmt.Errorf("invalid response format")
	}

	fmt.Printf("📚 Marketplace Categories (%d)\n\n", len(categories))

	for i, cat := range categories {
		category := cat.(map[string]interface{})
		icon := getString(category, "icon")
		if icon == "" {
			icon = "📦"
		}

		fmt.Printf("%s %d. %s\n", icon, i+1, getString(category, "name"))
		fmt.Printf("   📝 %s\n", getString(category, "description"))
		fmt.Printf("   📊 %d templates\n", getInt(category, "template_count"))

		if getBool(category, "featured") {
			fmt.Printf("   🌟 Featured category\n")
		}
		fmt.Printf("\n")
	}

	fmt.Printf("💡 Browse category: cws marketplace list --category <category-id>\n")
	return nil
}

// handleMyPublications shows user's published templates
func (a *App) handleMyPublications(args []string) error {
	response, err := a.makeAPIRequest("GET", "/api/v1/marketplace/my-publications", nil)
	if err != nil {
		return fmt.Errorf("failed to get publications: %w", err)
	}

	publications, ok := response["publications"].([]interface{})
	if !ok {
		return fmt.Errorf("invalid response format")
	}

	if len(publications) == 0 {
		fmt.Printf("📚 You haven't published any templates yet\n\n")
		fmt.Printf("💡 Publish a template: cws marketplace publish <instance> --name <name> --category <category> --description <desc>\n")
		return nil
	}

	fmt.Printf("📚 My Published Templates (%d)\n\n", len(publications))
	return a.displayTemplateList(publications)
}

// Helper methods

func (a *App) displayTemplateList(templates []interface{}) error {
	for i, tmpl := range templates {
		template := tmpl.(map[string]interface{})
		fmt.Printf("📦 %d. %s\n", i+1, getString(template, "name"))
		fmt.Printf("   📝 %s\n", getString(template, "description"))
		fmt.Printf("   👤 Author: %s\n", getString(template, "author_name"))
		fmt.Printf("   🏷️  Category: %s | Version: %s\n",
			getString(template, "category"), getString(template, "version"))

		// Rating and stats
		rating := getFloat64(template, "rating")
		reviewCount := getInt(template, "review_count")
		downloadCount := getInt(template, "download_count")

		if rating > 0 {
			stars := strings.Repeat("⭐", int(rating))
			fmt.Printf("   %s %.1f (%d reviews) | 📥 %d downloads\n",
				stars, rating, reviewCount, downloadCount)
		} else {
			fmt.Printf("   ⭐ No ratings yet | 📥 %d downloads\n", downloadCount)
		}

		// Verification badges
		badges := []string{}
		if getBool(template, "verified") {
			badges = append(badges, "✅ Verified")
		}
		if getBool(template, "featured") {
			badges = append(badges, "🌟 Featured")
		}
		if ami, exists := template["ami_info"]; exists && ami != nil {
			amiInfo := ami.(map[string]interface{})
			if getBool(amiInfo, "available") {
				badges = append(badges, "🚀 AMI Available")
			}
		}

		if len(badges) > 0 {
			fmt.Printf("   %s\n", strings.Join(badges, " "))
		}

		fmt.Printf("   💻 Launch: cws launch marketplace:%s my-project\n",
			getString(template, "template_id"))
		fmt.Printf("\n")
	}

	return nil
}

func (a *App) displayTemplateInfo(template map[string]interface{}) {
	fmt.Printf("📦 Template Details\n\n")

	// Basic information
	fmt.Printf("🆔 ID: %s\n", getString(template, "template_id"))
	fmt.Printf("📝 Name: %s\n", getString(template, "name"))
	fmt.Printf("📖 Description: %s\n", getString(template, "description"))
	fmt.Printf("👤 Author: %s\n", getString(template, "author_name"))
	fmt.Printf("🏷️  Category: %s\n", getString(template, "category"))
	fmt.Printf("🔖 Version: %s\n", getString(template, "version"))

	// Tags and keywords
	if tags := getStringSlice(template, "tags"); len(tags) > 0 {
		fmt.Printf("🏷️  Tags: %s\n", strings.Join(tags, ", "))
	}

	// Ratings and stats
	rating := getFloat64(template, "rating")
	reviewCount := getInt(template, "review_count")
	downloadCount := getInt(template, "download_count")
	launchCount := getInt(template, "launch_count")

	if rating > 0 {
		stars := strings.Repeat("⭐", int(rating))
		fmt.Printf("⭐ Rating: %s %.1f/5 (%d reviews)\n", stars, rating, reviewCount)
	} else {
		fmt.Printf("⭐ Rating: No ratings yet\n")
	}

	fmt.Printf("📊 Stats: %d downloads | %d launches\n", downloadCount, launchCount)

	// Technical specs
	if arch := getStringSlice(template, "architecture"); len(arch) > 0 {
		fmt.Printf("🏗️  Architecture: %s\n", strings.Join(arch, ", "))
	}
	if regions := getStringSlice(template, "supported_regions"); len(regions) > 0 {
		fmt.Printf("🌍 Regions: %s\n", strings.Join(regions, ", "))
	}

	// AMI availability
	if ami, exists := template["ami_info"]; exists && ami != nil {
		amiInfo := ami.(map[string]interface{})
		if getBool(amiInfo, "available") {
			fmt.Printf("🚀 AMI: Available for fast launch (30 seconds)\n")
		} else {
			fmt.Printf("⏱️  AMI: Not available, will build from script (5-8 minutes)\n")
		}
	}

	// Quality indicators
	badges := []string{}
	if getBool(template, "verified") {
		badges = append(badges, "✅ Verified")
	}
	if getBool(template, "featured") {
		badges = append(badges, "🌟 Featured")
	}

	securityScore := getInt(template, "security_score")
	if securityScore > 0 {
		badges = append(badges, fmt.Sprintf("🔒 Security: %d/100", securityScore))
	}

	if len(badges) > 0 {
		fmt.Printf("🏆 Quality: %s\n", strings.Join(badges, " "))
	}

	// Documentation
	if docs := getString(template, "documentation"); docs != "" {
		fmt.Printf("\n📚 Documentation:\n%s\n", docs)
	}

	// Usage examples
	fmt.Printf("\n💻 Usage:\n")
	fmt.Printf("   Launch: cws launch marketplace:%s my-project\n", getString(template, "template_id"))
	fmt.Printf("   Info: cws marketplace info %s\n", getString(template, "template_id"))
	fmt.Printf("   Review: cws marketplace review %s --rating 5 --title \"Great!\" --comment \"Works perfectly\"\n",
		getString(template, "template_id"))
	fmt.Printf("   Fork: cws marketplace fork %s --name \"My Custom Version\" --description \"Customized for my needs\"\n",
		getString(template, "template_id"))
}

func (a *App) makeAPIRequest(method, endpoint string, body interface{}) (map[string]interface{}, error) {
	// This is a placeholder - in production, this would make actual HTTP requests to the daemon
	// For now, return mock responses based on the endpoint

	switch {
	case strings.Contains(endpoint, "/api/v1/marketplace/templates") && method == "GET":
		return a.mockTemplateListResponse(), nil
	case strings.Contains(endpoint, "/api/v1/marketplace/categories"):
		return a.mockCategoriesResponse(), nil
	case strings.Contains(endpoint, "/api/v1/marketplace/featured"):
		return a.mockFeaturedResponse(), nil
	case strings.Contains(endpoint, "/api/v1/marketplace/trending"):
		return a.mockTrendingResponse(), nil
	case strings.Contains(endpoint, "/api/v1/marketplace/publish") && method == "POST":
		return a.mockPublishResponse(), nil
	default:
		return map[string]interface{}{
			"status":  "success",
			"message": "Mock response",
		}, nil
	}
}

// Mock response helpers (for development/testing)

func (a *App) mockTemplateListResponse() map[string]interface{} {
	return map[string]interface{}{
		"templates": []map[string]interface{}{
			{
				"template_id":    "genomics-pipeline-v3",
				"name":           "Advanced Genomics Analysis Pipeline",
				"description":    "Complete genomics workflow with GATK, BWA, and Bioconductor",
				"author":         "research-lab-genomics",
				"author_name":    "Genomics Research Lab",
				"category":       "bioinformatics",
				"version":        "3.2.1",
				"rating":         4.7,
				"review_count":   23,
				"download_count": 1547,
				"verified":       true,
				"featured":       true,
				"ami_info": map[string]interface{}{
					"available": true,
				},
			},
			{
				"template_id":    "machine-learning-gpu",
				"name":           "GPU-Accelerated ML Environment",
				"description":    "PyTorch, TensorFlow, and CUDA toolkit for deep learning research",
				"author":         "ai-research-team",
				"author_name":    "AI Research Team",
				"category":       "machine-learning",
				"version":        "2.1.0",
				"rating":         4.5,
				"review_count":   67,
				"download_count": 2341,
				"verified":       true,
				"featured":       true,
				"ami_info": map[string]interface{}{
					"available": true,
				},
			},
		},
		"total_count": 2,
	}
}

func (a *App) mockCategoriesResponse() map[string]interface{} {
	return map[string]interface{}{
		"categories": []map[string]interface{}{
			{
				"id":             "machine-learning",
				"name":           "Machine Learning & AI",
				"description":    "Deep learning, neural networks, and AI research environments",
				"icon":           "🤖",
				"template_count": 15,
				"featured":       true,
			},
			{
				"id":             "bioinformatics",
				"name":           "Bioinformatics",
				"description":    "Genomics, proteomics, and computational biology tools",
				"icon":           "🧬",
				"template_count": 12,
				"featured":       true,
			},
		},
		"total": 2,
	}
}

func (a *App) mockFeaturedResponse() map[string]interface{} {
	return map[string]interface{}{
		"templates": []map[string]interface{}{
			{
				"template_id":    "genomics-pipeline-v3",
				"name":           "Advanced Genomics Analysis Pipeline",
				"description":    "Complete genomics workflow with GATK, BWA, and Bioconductor",
				"author_name":    "Genomics Research Lab",
				"category":       "bioinformatics",
				"version":        "3.2.1",
				"rating":         4.7,
				"review_count":   23,
				"download_count": 1547,
				"verified":       true,
				"featured":       true,
			},
		},
		"count": 1,
	}
}

func (a *App) mockTrendingResponse() map[string]interface{} {
	return a.mockFeaturedResponse() // Same data for now
}

func (a *App) mockPublishResponse() map[string]interface{} {
	return map[string]interface{}{
		"template_id":     "my-custom-template-12345",
		"publication_url": "https://marketplace.cloudworkstation.org/templates/my-custom-template-12345",
		"status":          "published",
		"message":         "Template published successfully",
	}
}
