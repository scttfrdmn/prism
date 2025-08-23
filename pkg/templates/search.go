// Package templates provides CloudWorkstation's unified template system.
package templates

import (
	"sort"
	"strings"
)

// SearchOptions defines search and filter criteria for templates
type SearchOptions struct {
	Query         string   // Text search query
	Category      string   // Filter by category
	Domain        string   // Filter by domain
	Complexity    string   // Filter by complexity
	Tags          []string // Filter by tags
	Popular       *bool    // Filter by popular status
	Featured      *bool    // Filter by featured status
	HasGPU        *bool    // Filter by GPU support
	MaxLaunchTime int      // Maximum launch time in minutes
}

// SearchResult represents a template search result with relevance scoring
type SearchResult struct {
	Template *Template
	Score    float64  // Relevance score for ranking
	Matches  []string // What matched in the search
}

// SearchTemplates searches and filters templates based on criteria
func SearchTemplates(templates map[string]*Template, options SearchOptions) []SearchResult {
	var results []SearchResult

	for _, template := range templates {
		if result := evaluateTemplate(template, options); result != nil {
			results = append(results, *result)
		}
	}

	// Sort by score descending
	sort.Slice(results, func(i, j int) bool {
		// Featured templates always come first
		if results[i].Template.Featured != results[j].Template.Featured {
			return results[i].Template.Featured
		}
		// Then by score
		if results[i].Score != results[j].Score {
			return results[i].Score > results[j].Score
		}
		// Then popular templates
		if results[i].Template.Popular != results[j].Template.Popular {
			return results[i].Template.Popular
		}
		// Finally by name
		return results[i].Template.Name < results[j].Template.Name
	})

	return results
}

// evaluateTemplate checks if a template matches the search criteria
func evaluateTemplate(template *Template, options SearchOptions) *SearchResult {
	var score float64
	var matches []string

	// Apply filters first (these are binary - match or not)
	if !passesFilters(template, options) {
		return nil
	}

	// Text search scoring
	if options.Query != "" {
		queryLower := strings.ToLower(options.Query)

		// Exact name match (highest score)
		if strings.ToLower(template.Name) == queryLower {
			score += 10.0
			matches = append(matches, "exact name match")
		} else if strings.Contains(strings.ToLower(template.Name), queryLower) {
			score += 5.0
			matches = append(matches, "name contains query")
		}

		// Description match
		if strings.Contains(strings.ToLower(template.Description), queryLower) {
			score += 3.0
			matches = append(matches, "description contains query")
		}

		// Long description match
		if strings.Contains(strings.ToLower(template.LongDescription), queryLower) {
			score += 2.0
			matches = append(matches, "long description contains query")
		}

		// Category/domain match
		if strings.Contains(strings.ToLower(template.Category), queryLower) {
			score += 2.0
			matches = append(matches, "category contains query")
		}
		if strings.Contains(strings.ToLower(template.Domain), queryLower) {
			score += 2.0
			matches = append(matches, "domain contains query")
		}

		// Tag match
		for tagKey, tagValue := range template.Tags {
			if strings.Contains(strings.ToLower(tagKey), queryLower) ||
				strings.Contains(strings.ToLower(tagValue), queryLower) {
				score += 1.0
				matches = append(matches, "tag match: "+tagKey)
			}
		}

		// No text match at all - exclude
		if score == 0 && options.Query != "" {
			return nil
		}
	} else {
		// No query - include all that pass filters with base score
		score = 1.0
	}

	// Boost score for popular/featured templates
	if template.Popular {
		score *= 1.2
	}
	if template.Featured {
		score *= 1.5
	}

	return &SearchResult{
		Template: template,
		Score:    score,
		Matches:  matches,
	}
}

// passesFilters checks if template passes all specified filters
func passesFilters(template *Template, options SearchOptions) bool {
	// Category filter
	if options.Category != "" && !strings.EqualFold(template.Category, options.Category) {
		return false
	}

	// Domain filter
	if options.Domain != "" && !strings.EqualFold(template.Domain, options.Domain) {
		return false
	}

	// Complexity filter
	if options.Complexity != "" && string(template.Complexity) != options.Complexity {
		return false
	}

	// Popular filter
	if options.Popular != nil && template.Popular != *options.Popular {
		return false
	}

	// Featured filter
	if options.Featured != nil && template.Featured != *options.Featured {
		return false
	}

	// Launch time filter
	if options.MaxLaunchTime > 0 && template.EstimatedLaunchTime > options.MaxLaunchTime {
		return false
	}

	// GPU support filter (check instance defaults)
	if options.HasGPU != nil {
		hasGPU := strings.Contains(template.InstanceDefaults.Type, "g") ||
			strings.Contains(template.InstanceDefaults.Type, "p")
		if hasGPU != *options.HasGPU {
			return false
		}
	}

	// Tag filters
	if len(options.Tags) > 0 {
		for _, requiredTag := range options.Tags {
			found := false
			for tagKey := range template.Tags {
				if strings.EqualFold(tagKey, requiredTag) {
					found = true
					break
				}
			}
			if !found {
				return false
			}
		}
	}

	return true
}

// GetCategories returns all unique categories from templates
func GetCategories(templates map[string]*Template) []string {
	categoryMap := make(map[string]bool)
	for _, template := range templates {
		if template.Category != "" {
			categoryMap[template.Category] = true
		}
	}

	var categories []string
	for category := range categoryMap {
		categories = append(categories, category)
	}
	sort.Strings(categories)
	return categories
}

// GetDomains returns all unique domains from templates
func GetDomains(templates map[string]*Template) []string {
	domainMap := make(map[string]bool)
	for _, template := range templates {
		if template.Domain != "" {
			domainMap[template.Domain] = true
		}
	}

	var domains []string
	for domain := range domainMap {
		domains = append(domains, domain)
	}
	sort.Strings(domains)
	return domains
}

// GetTags returns all unique tags from templates
func GetTags(templates map[string]*Template) []string {
	tagMap := make(map[string]bool)
	for _, template := range templates {
		for tagKey := range template.Tags {
			tagMap[tagKey] = true
		}
	}

	var tags []string
	for tag := range tagMap {
		tags = append(tags, tag)
	}
	sort.Strings(tags)
	return tags
}

// RecommendTemplates suggests templates based on user history and preferences
func RecommendTemplates(templates map[string]*Template, userDomain string, maxResults int) []*Template {
	var recommendations []*Template

	// First, get templates from the same domain
	for _, template := range templates {
		if template.Domain == userDomain {
			recommendations = append(recommendations, template)
		}
	}

	// Sort by popularity and complexity
	sort.Slice(recommendations, func(i, j int) bool {
		// Featured first
		if recommendations[i].Featured != recommendations[j].Featured {
			return recommendations[i].Featured
		}
		// Then popular
		if recommendations[i].Popular != recommendations[j].Popular {
			return recommendations[i].Popular
		}
		// Then by complexity (simpler first for recommendations)
		return recommendations[i].Complexity.Level() < recommendations[j].Complexity.Level()
	})

	// Limit results
	if len(recommendations) > maxResults {
		recommendations = recommendations[:maxResults]
	}

	return recommendations
}
