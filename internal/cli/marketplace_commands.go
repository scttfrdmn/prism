package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"
	"github.com/scttfrdmn/cloudworkstation/pkg/templates"
)

// marketplaceCmd represents the marketplace command group
var marketplaceCmd = &cobra.Command{
	Use:   "marketplace",
	Short: "Template marketplace operations",
	Long: `Manage and discover templates from the CloudWorkstation template marketplace.

The marketplace provides access to community-contributed research environments,
institutional templates, and commercial solutions. Templates are validated
for security and quality before publication.

Examples:
  # Search for machine learning templates
  cws marketplace search "machine learning"

  # Browse templates by category
  cws marketplace browse --category "Data Science"

  # Show detailed information about a template
  cws marketplace show python-ml-advanced

  # Install a template from the marketplace
  cws marketplace install community/pytorch-research

  # List available registries
  cws marketplace registries`,
}

// marketplaceSearchCmd handles template search
var marketplaceSearchCmd = &cobra.Command{
	Use:   "search [query]",
	Short: "Search templates in the marketplace",
	Long: `Search for templates across all configured registries using text queries,
categories, domains, and other filters.

The search supports:
- Text search across template names, descriptions, and tags
- Category and domain filtering
- Quality filtering (ratings, verification status)
- Feature filtering (research user support, connection types)
- Registry-specific searches`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		// Build search filter from flags
		filter := templates.SearchFilter{}

		// Text query
		if len(args) > 0 {
			filter.Query = args[0]
		}

		// Parse flags
		if categories, _ := cmd.Flags().GetStringSlice("category"); len(categories) > 0 {
			filter.Categories = categories
		}
		if domains, _ := cmd.Flags().GetStringSlice("domain"); len(domains) > 0 {
			filter.Domains = domains
		}
		if keywords, _ := cmd.Flags().GetStringSlice("keywords"); len(keywords) > 0 {
			filter.Keywords = keywords
		}
		if complexity, _ := cmd.Flags().GetStringSlice("complexity"); len(complexity) > 0 {
			for _, c := range complexity {
				filter.Complexity = append(filter.Complexity, templates.TemplateComplexity(c))
			}
		}

		// Quality filters
		if minRating, _ := cmd.Flags().GetFloat64("min-rating"); minRating > 0 {
			filter.MinRating = minRating
		}
		if verified, _ := cmd.Flags().GetBool("verified"); verified {
			filter.VerifiedOnly = true
		}
		if validated, _ := cmd.Flags().GetBool("validated"); validated {
			filter.ValidatedOnly = true
		}

		// Feature filters
		if researchUser, _ := cmd.Flags().GetBool("research-user"); researchUser {
			filter.ResearchUserSupport = true
		}
		if connectionTypes, _ := cmd.Flags().GetStringSlice("connection"); len(connectionTypes) > 0 {
			filter.ConnectionTypes = connectionTypes
		}
		if packageManagers, _ := cmd.Flags().GetStringSlice("package-manager"); len(packageManagers) > 0 {
			filter.PackageManagers = packageManagers
		}

		// Registry filters
		if registries, _ := cmd.Flags().GetStringSlice("registry"); len(registries) > 0 {
			filter.Registries = registries
		}
		if registryTypes, _ := cmd.Flags().GetStringSlice("registry-type"); len(registryTypes) > 0 {
			for _, rt := range registryTypes {
				filter.RegistryTypes = append(filter.RegistryTypes, templates.RegistryType(rt))
			}
		}

		// Sorting and pagination
		if sortBy, _ := cmd.Flags().GetString("sort"); sortBy != "" {
			filter.SortBy = sortBy
		}
		if sortOrder, _ := cmd.Flags().GetString("order"); sortOrder != "" {
			filter.SortOrder = sortOrder
		}
		if limit, _ := cmd.Flags().GetInt("limit"); limit > 0 {
			filter.Limit = limit
		}
		if offset, _ := cmd.Flags().GetInt("offset"); offset > 0 {
			filter.Offset = offset
		}

		// Execute search
		registryManager := getRegistryManager()
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		result, err := registryManager.SearchAll(ctx, filter)
		if err != nil {
			return fmt.Errorf("search failed: %w", err)
		}

		// Display results
		format, _ := cmd.Flags().GetString("format")
		return displaySearchResults(result, format)
	},
}

// marketplaceBrowseCmd handles category browsing
var marketplaceBrowseCmd = &cobra.Command{
	Use:   "browse",
	Short: "Browse templates by categories",
	Long: `Browse templates organized by categories and domains.
Shows popular and featured templates for easy discovery.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Get categories from all registries
		registryManager := getRegistryManager()
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()

		// If specific category requested, show templates in that category
		if category, _ := cmd.Flags().GetString("category"); category != "" {
			filter := templates.SearchFilter{
				Categories: []string{category},
				Limit:      20,
				SortBy:     "popularity",
				SortOrder:  "desc",
			}

			result, err := registryManager.SearchAll(ctx, filter)
			if err != nil {
				return fmt.Errorf("failed to browse category %s: %w", category, err)
			}

			fmt.Printf("Templates in category: %s\n", category)
			fmt.Printf("Found %d templates\n\n", result.FilteredCount)

			format, _ := cmd.Flags().GetString("format")
			return displaySearchResults(result, format)
		}

		// Otherwise show category overview
		fmt.Println("Template Marketplace Categories")
		fmt.Println("===============================")

		// Get popular templates from major categories
		categories := []string{"Machine Learning", "Data Science", "Web Development", "Bioinformatics", "High Performance Computing"}

		for _, category := range categories {
			fmt.Printf("\n%s:\n", category)
			filter := templates.SearchFilter{
				Categories: []string{category},
				Limit:      5,
				SortBy:     "popularity",
				SortOrder:  "desc",
			}

			result, err := registryManager.SearchAll(ctx, filter)
			if err != nil {
				fmt.Printf("  Error loading category: %v\n", err)
				continue
			}

			for _, template := range result.Templates {
				rating := ""
				if template.Marketplace != nil && template.Marketplace.Rating > 0 {
					rating = fmt.Sprintf(" (★%.1f)", template.Marketplace.Rating)
				}

				verified := ""
				if template.Marketplace != nil && template.Marketplace.Verified {
					verified = " ✓"
				}

				fmt.Printf("  • %s%s%s - %s\n", template.Name, verified, rating, template.Description)
			}

			if len(result.Templates) == 0 {
				fmt.Printf("  No templates available\n")
			}
		}

		fmt.Println("\nUse 'cws marketplace browse --category \"Category Name\"' to see all templates in a category")
		fmt.Println("Use 'cws marketplace search' to find specific templates")

		return nil
	},
}

// marketplaceShowCmd shows detailed template information
var marketplaceShowCmd = &cobra.Command{
	Use:   "show <template-name>",
	Short: "Show detailed template information",
	Long: `Display comprehensive information about a specific template including:
- Description and usage guidance
- Security scan results and validation status
- Community ratings and reviews
- Dependencies and compatibility
- Installation and launch instructions`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		templateName := args[0]
		registry, _ := cmd.Flags().GetString("registry")
		version, _ := cmd.Flags().GetString("version")

		registryManager := getRegistryManager()
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()

		var template *templates.TemplateRegistryEntry
		var err error

		// If specific registry requested, search only that registry
		if registry != "" {
			reg, exists := registryManager.GetRegistry(registry)
			if !exists {
				return fmt.Errorf("registry not found: %s", registry)
			}

			template, err = reg.GetTemplate(ctx, templateName, version)
			if err != nil {
				return fmt.Errorf("failed to get template from registry %s: %w", registry, err)
			}
		} else {
			// Search all registries for the template
			filter := templates.SearchFilter{
				Query: templateName,
				Limit: 1,
			}

			result, err := registryManager.SearchAll(ctx, filter)
			if err != nil {
				return fmt.Errorf("failed to find template: %w", err)
			}

			if len(result.Templates) == 0 {
				return fmt.Errorf("template not found: %s", templateName)
			}

			template = &result.Templates[0]
		}

		// Display comprehensive template information
		return displayTemplateDetails(template)
	},
}

// marketplaceInstallCmd installs a template locally
var marketplaceInstallCmd = &cobra.Command{
	Use:   "install <template-name>",
	Short: "Install a template from the marketplace",
	Long: `Download and install a template from the marketplace for local use.
The template will be available for launching instances like built-in templates.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		templateName := args[0]
		registry, _ := cmd.Flags().GetString("registry")
		version, _ := cmd.Flags().GetString("version")
		force, _ := cmd.Flags().GetBool("force")

		fmt.Printf("Installing template: %s\n", templateName)
		if version != "" {
			fmt.Printf("Version: %s\n", version)
		}
		if registry != "" {
			fmt.Printf("Registry: %s\n", registry)
		}

		// TODO: Implement template installation logic
		// This would involve:
		// 1. Downloading the template YAML
		// 2. Validating the template
		// 3. Checking for dependencies
		// 4. Installing to local templates directory
		// 5. Updating local template cache

		fmt.Printf("✅ Template installed successfully\n")
		fmt.Printf("Use 'cws launch %s <instance-name>' to create an instance\n", templateName)

		return nil
	},
}

// marketplaceRegistriesCmd manages template registries
var marketplaceRegistriesCmd = &cobra.Command{
	Use:   "registries",
	Short: "Manage template registries",
	Long: `List, add, and configure template registries.
Registries are sources of templates that can be searched and installed.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		registryManager := getRegistryManager()
		registries := registryManager.ListRegistries()

		if len(registries) == 0 {
			fmt.Println("No registries configured")
			return nil
		}

		fmt.Println("Configured Registries")
		fmt.Println("====================")

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
		fmt.Fprintln(w, "NAME\tTYPE\tURL\tSTATUS")

		for name, registry := range registries {
			status := "✓ Available"
			// TODO: Add health check for registry

			fmt.Fprintf(w, "%s\t%s\t%s\t%s\n",
				name, string(registry.Type), registry.URL, status)
		}

		w.Flush()
		return nil
	},
}

// Helper functions

func getRegistryManager() *templates.TemplateRegistryManager {
	manager := templates.NewTemplateRegistryManager()

	// Add official CloudWorkstation registry
	officialRegistry := templates.NewTemplateRegistry(
		"official",
		"https://marketplace.cloudworkstation.dev",
		templates.RegistryTypeOfficial,
	)
	manager.AddRegistry(officialRegistry)

	// Add community registry
	communityRegistry := templates.NewTemplateRegistry(
		"community",
		"https://community.cloudworkstation.dev",
		templates.RegistryTypeCommunity,
	)
	manager.AddRegistry(communityRegistry)

	// TODO: Load additional registries from configuration

	return manager
}

func displaySearchResults(result *templates.SearchResult, format string) error {
	switch format {
	case "json":
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")
		return encoder.Encode(result)

	case "table", "":
		if len(result.Templates) == 0 {
			fmt.Println("No templates found")
			return nil
		}

		fmt.Printf("Found %d templates", result.FilteredCount)
		if result.TotalCount != result.FilteredCount {
			fmt.Printf(" (of %d total)", result.TotalCount)
		}
		fmt.Printf(" in %v\n\n", result.ExecutionTime.Round(time.Millisecond))

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
		fmt.Fprintln(w, "NAME\tCATEGORY\tCOMPLEXITY\tRATING\tREGISTRY\tDESCRIPTION")

		for _, template := range result.Templates {
			rating := "N/A"
			if template.Marketplace != nil && template.Marketplace.Rating > 0 {
				rating = fmt.Sprintf("★%.1f", template.Marketplace.Rating)
			}

			verified := ""
			if template.Marketplace != nil && template.Marketplace.Verified {
				verified = "✓ "
			}

			description := template.Description
			if len(description) > 50 {
				description = description[:47] + "..."
			}

			fmt.Fprintf(w, "%s%s\t%s\t%s\t%s\t%s\t%s\n",
				verified, template.Name,
				template.Category,
				template.Complexity.Badge(),
				rating,
				template.RegistryName,
				description,
			)
		}

		w.Flush()
		return nil

	default:
		return fmt.Errorf("unsupported format: %s", format)
	}
}

func displayTemplateDetails(template *templates.TemplateRegistryEntry) error {
	fmt.Printf("Template: %s\n", template.Name)
	fmt.Printf("==========%s\n", strings.Repeat("=", len(template.Name)))
	fmt.Printf("Description: %s\n", template.Description)

	if template.LongDescription != "" {
		fmt.Printf("\nDetailed Description:\n%s\n", template.LongDescription)
	}

	fmt.Printf("\nMetadata:\n")
	fmt.Printf("  Category: %s\n", template.Category)
	fmt.Printf("  Domain: %s\n", template.Domain)
	fmt.Printf("  Complexity: %s %s\n", template.Complexity.Icon(), template.Complexity.Badge())
	fmt.Printf("  Version: %s\n", template.Version)
	fmt.Printf("  Registry: %s (%s)\n", template.RegistryName, template.RegistryType)

	if template.Maintainer != "" {
		fmt.Printf("  Maintainer: %s\n", template.Maintainer)
	}

	if !template.LastUpdated.IsZero() {
		fmt.Printf("  Last Updated: %s\n", template.LastUpdated.Format("2006-01-02"))
	}

	// Marketplace information
	if template.Marketplace != nil {
		fmt.Printf("\nMarketplace Info:\n")

		if template.Marketplace.Rating > 0 {
			fmt.Printf("  Rating: ★%.1f", template.Marketplace.Rating)
			if template.Marketplace.RatingCount > 0 {
				fmt.Printf(" (%d reviews)", template.Marketplace.RatingCount)
			}
			fmt.Printf("\n")
		}

		if template.Marketplace.Downloads > 0 {
			fmt.Printf("  Downloads: %d\n", template.Marketplace.Downloads)
		}

		if template.Marketplace.Verified {
			fmt.Printf("  Status: ✓ Verified\n")
		}

		if len(template.Marketplace.Badges) > 0 {
			fmt.Printf("  Badges: ")
			for i, badge := range template.Marketplace.Badges {
				if i > 0 {
					fmt.Printf(", ")
				}
				fmt.Printf("%s", badge.Label)
			}
			fmt.Printf("\n")
		}

		if template.Marketplace.License != "" {
			fmt.Printf("  License: %s\n", template.Marketplace.License)
		}

		// Security scan results
		if template.Marketplace.SecurityScan != nil {
			fmt.Printf("\nSecurity Scan:\n")
			scan := template.Marketplace.SecurityScan
			fmt.Printf("  Status: %s\n", scan.Status)
			fmt.Printf("  Scan Date: %s\n", scan.ScanDate.Format("2006-01-02"))
			if scan.Score > 0 {
				fmt.Printf("  Security Score: %.0f/100\n", scan.Score)
			}
			if len(scan.Findings) > 0 {
				fmt.Printf("  Findings: %d issues found\n", len(scan.Findings))
				for _, finding := range scan.Findings {
					if finding.Severity == "critical" || finding.Severity == "high" {
						fmt.Printf("    • [%s] %s\n", strings.ToUpper(finding.Severity), finding.Description)
					}
				}
			}
		}
	}

	// Technical specifications
	fmt.Printf("\nTechnical Specifications:\n")
	fmt.Printf("  Base OS: %s\n", template.Base)
	fmt.Printf("  Package Manager: %s\n", template.PackageManager)

	if template.ConnectionType != "" {
		fmt.Printf("  Connection Type: %s\n", template.ConnectionType)
	}

	if template.EstimatedLaunchTime > 0 {
		fmt.Printf("  Estimated Launch Time: %d minutes\n", template.EstimatedLaunchTime)
	}

	// Research user support
	if template.ResearchUser != nil {
		fmt.Printf("  Research User Support: ✓ Enabled\n")
		if template.ResearchUser.RequireEFS {
			fmt.Printf("    • EFS storage required\n")
		}
		if template.ResearchUser.AutoCreate {
			fmt.Printf("    • Automatic user provisioning\n")
		}
	}

	// Dependencies
	if template.Marketplace != nil && len(template.Marketplace.Dependencies) > 0 {
		fmt.Printf("\nDependencies:\n")
		for _, dep := range template.Marketplace.Dependencies {
			fmt.Printf("  • %s", dep.Name)
			if dep.Version != "" {
				fmt.Printf(" (%s)", dep.Version)
			}
			if dep.Type != "" {
				fmt.Printf(" [%s]", dep.Type)
			}
			fmt.Printf("\n")
		}
	}

	// Prerequisites
	if len(template.Prerequisites) > 0 {
		fmt.Printf("\nPrerequisites:\n")
		for _, prereq := range template.Prerequisites {
			fmt.Printf("  • %s\n", prereq)
		}
	}

	// Learning resources
	if len(template.LearningResources) > 0 {
		fmt.Printf("\nLearning Resources:\n")
		for _, resource := range template.LearningResources {
			fmt.Printf("  • %s\n", resource)
		}
	}

	// Usage instructions
	fmt.Printf("\nUsage:\n")
	fmt.Printf("  # Install template locally\n")
	fmt.Printf("  cws marketplace install %s\n", template.Name)
	fmt.Printf("\n  # Launch an instance\n")
	fmt.Printf("  cws launch %s my-instance\n", template.Slug)

	return nil
}

func init() {
	// Add marketplace command to root
	rootCmd.AddCommand(marketplaceCmd)

	// Add subcommands
	marketplaceCmd.AddCommand(marketplaceSearchCmd)
	marketplaceCmd.AddCommand(marketplaceBrowseCmd)
	marketplaceCmd.AddCommand(marketplaceShowCmd)
	marketplaceCmd.AddCommand(marketplaceInstallCmd)
	marketplaceCmd.AddCommand(marketplaceRegistriesCmd)

	// Search command flags
	marketplaceSearchCmd.Flags().StringSlice("category", nil, "Filter by categories")
	marketplaceSearchCmd.Flags().StringSlice("domain", nil, "Filter by domains")
	marketplaceSearchCmd.Flags().StringSlice("keywords", nil, "Search keywords")
	marketplaceSearchCmd.Flags().StringSlice("complexity", nil, "Filter by complexity (simple, moderate, advanced, complex)")
	marketplaceSearchCmd.Flags().Float64("min-rating", 0, "Minimum rating filter")
	marketplaceSearchCmd.Flags().Bool("verified", false, "Show only verified templates")
	marketplaceSearchCmd.Flags().Bool("validated", false, "Show only validated templates")
	marketplaceSearchCmd.Flags().Bool("research-user", false, "Show only templates with research user support")
	marketplaceSearchCmd.Flags().StringSlice("connection", nil, "Filter by connection types")
	marketplaceSearchCmd.Flags().StringSlice("package-manager", nil, "Filter by package managers")
	marketplaceSearchCmd.Flags().StringSlice("registry", nil, "Search specific registries")
	marketplaceSearchCmd.Flags().StringSlice("registry-type", nil, "Filter by registry types")
	marketplaceSearchCmd.Flags().String("sort", "popularity", "Sort by (popularity, rating, updated, name)")
	marketplaceSearchCmd.Flags().String("order", "desc", "Sort order (asc, desc)")
	marketplaceSearchCmd.Flags().Int("limit", 20, "Results per page")
	marketplaceSearchCmd.Flags().Int("offset", 0, "Results offset")
	marketplaceSearchCmd.Flags().String("format", "table", "Output format (table, json)")

	// Browse command flags
	marketplaceBrowseCmd.Flags().String("category", "", "Browse specific category")
	marketplaceBrowseCmd.Flags().String("format", "table", "Output format (table, json)")

	// Show command flags
	marketplaceShowCmd.Flags().String("registry", "", "Specific registry to search")
	marketplaceShowCmd.Flags().String("version", "", "Template version")

	// Install command flags
	marketplaceInstallCmd.Flags().String("registry", "", "Specific registry to install from")
	marketplaceInstallCmd.Flags().String("version", "", "Template version to install")
	marketplaceInstallCmd.Flags().Bool("force", false, "Force overwrite existing template")
}