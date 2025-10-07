package cli

import (
	"github.com/spf13/cobra"
)

// MarketplaceCobraCommands handles template marketplace commands
type MarketplaceCobraCommands struct {
	app *App
}

// NewMarketplaceCobraCommands creates new marketplace cobra commands
func NewMarketplaceCobraCommands(app *App) *MarketplaceCobraCommands {
	return &MarketplaceCobraCommands{app: app}
}

// CreateMarketplaceCommand creates the marketplace command with subcommands
func (mc *MarketplaceCobraCommands) CreateMarketplaceCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "marketplace",
		Short: "Browse and manage community templates",
		Long: `Browse, publish, and manage community templates from the CloudWorkstation marketplace.

The marketplace provides access to community-contributed research environments,
allowing you to share and discover specialized templates for different research domains.`,
	}

	// Add all marketplace subcommands
	cmd.AddCommand(
		mc.createListCommand(),
		mc.createSearchCommand(),
		mc.createInfoCommand(),
		mc.createInstallCommand(),
		mc.createPublishCommand(),
		mc.createReviewCommand(),
		mc.createForkCommand(),
		mc.createFeaturedCommand(),
		mc.createTrendingCommand(),
		mc.createCategoriesCommand(),
		mc.createMyPublicationsCommand(),
	)

	return cmd
}

func (mc *MarketplaceCobraCommands) createListCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List available marketplace templates",
		Long:  `List templates available in the community marketplace.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			category, _ := cmd.Flags().GetString("category")
			limit, _ := cmd.Flags().GetInt("limit")

			marketplaceArgs := []string{"list"}
			if category != "" {
				marketplaceArgs = append(marketplaceArgs, "--category", category)
			}
			if limit > 0 {
				marketplaceArgs = append(marketplaceArgs, "--limit", string(rune(limit)))
			}

			return mc.app.Marketplace(marketplaceArgs)
		},
	}

	cmd.Flags().String("category", "", "Filter by category")
	cmd.Flags().Int("limit", 20, "Limit number of results")

	return cmd
}

func (mc *MarketplaceCobraCommands) createSearchCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "search <query>",
		Short: "Search marketplace templates",
		Long:  `Search for templates in the community marketplace.`,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			category, _ := cmd.Flags().GetString("category")
			tag, _ := cmd.Flags().GetString("tag")
			author, _ := cmd.Flags().GetString("author")

			marketplaceArgs := []string{"search", args[0]}
			if category != "" {
				marketplaceArgs = append(marketplaceArgs, "--category", category)
			}
			if tag != "" {
				marketplaceArgs = append(marketplaceArgs, "--tag", tag)
			}
			if author != "" {
				marketplaceArgs = append(marketplaceArgs, "--author", author)
			}

			return mc.app.Marketplace(marketplaceArgs)
		},
	}

	cmd.Flags().String("category", "", "Filter by category")
	cmd.Flags().String("tag", "", "Filter by tag")
	cmd.Flags().String("author", "", "Filter by author")

	return cmd
}

func (mc *MarketplaceCobraCommands) createInfoCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "info <template-name>",
		Short: "Show detailed template information",
		Long:  `Show detailed information about a marketplace template.`,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return mc.app.Marketplace([]string{"info", args[0]})
		},
	}
}

func (mc *MarketplaceCobraCommands) createInstallCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "install <template-name>",
		Short: "Install marketplace template locally",
		Long:  `Install a marketplace template to your local template library.`,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			version, _ := cmd.Flags().GetString("version")
			force, _ := cmd.Flags().GetBool("force")

			marketplaceArgs := []string{"install", args[0]}
			if version != "" {
				marketplaceArgs = append(marketplaceArgs, "--version", version)
			}
			if force {
				marketplaceArgs = append(marketplaceArgs, "--force")
			}

			return mc.app.Marketplace(marketplaceArgs)
		},
	}

	cmd.Flags().String("version", "", "Specific version to install")
	cmd.Flags().Bool("force", false, "Force reinstall if already installed")

	return cmd
}

func (mc *MarketplaceCobraCommands) createPublishCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "publish <template-path>",
		Short: "Publish template to marketplace",
		Long: `Publish your template to the community marketplace for others to use.

This will validate your template and make it available for discovery and installation.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			public, _ := cmd.Flags().GetBool("public")
			category, _ := cmd.Flags().GetString("category")
			description, _ := cmd.Flags().GetString("description")
			tags, _ := cmd.Flags().GetStringSlice("tags")

			marketplaceArgs := []string{"publish", args[0]}
			if public {
				marketplaceArgs = append(marketplaceArgs, "--public")
			}
			if category != "" {
				marketplaceArgs = append(marketplaceArgs, "--category", category)
			}
			if description != "" {
				marketplaceArgs = append(marketplaceArgs, "--description", description)
			}
			for _, tag := range tags {
				marketplaceArgs = append(marketplaceArgs, "--tag", tag)
			}

			return mc.app.Marketplace(marketplaceArgs)
		},
	}

	cmd.Flags().Bool("public", false, "Make template publicly available")
	cmd.Flags().String("category", "", "Template category")
	cmd.Flags().String("description", "", "Template description")
	cmd.Flags().StringSlice("tags", []string{}, "Template tags")

	return cmd
}

func (mc *MarketplaceCobraCommands) createReviewCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "review <template-name>",
		Short: "Review and rate a marketplace template",
		Long:  `Review and rate a template you have used from the marketplace.`,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			rating, _ := cmd.Flags().GetInt("rating")
			comment, _ := cmd.Flags().GetString("comment")

			marketplaceArgs := []string{"review", args[0]}
			if rating > 0 {
				marketplaceArgs = append(marketplaceArgs, "--rating", string(rune(rating)))
			}
			if comment != "" {
				marketplaceArgs = append(marketplaceArgs, "--comment", comment)
			}

			return mc.app.Marketplace(marketplaceArgs)
		},
	}

	cmd.Flags().Int("rating", 0, "Rating (1-5 stars)")
	cmd.Flags().String("comment", "", "Review comment")

	return cmd
}

func (mc *MarketplaceCobraCommands) createForkCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "fork <template-name> <new-name>",
		Short: "Fork a marketplace template",
		Long:  `Create a fork of a marketplace template for customization.`,
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			return mc.app.Marketplace([]string{"fork", args[0], args[1]})
		},
	}
}

func (mc *MarketplaceCobraCommands) createFeaturedCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "featured",
		Short: "Show featured marketplace templates",
		Long:  `Show templates that are currently featured in the marketplace.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return mc.app.Marketplace([]string{"featured"})
		},
	}
}

func (mc *MarketplaceCobraCommands) createTrendingCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "trending",
		Short: "Show trending marketplace templates",
		Long:  `Show templates that are currently trending in downloads and usage.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return mc.app.Marketplace([]string{"trending"})
		},
	}
}

func (mc *MarketplaceCobraCommands) createCategoriesCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "categories",
		Short: "List available template categories",
		Long:  `List all available categories for marketplace templates.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return mc.app.Marketplace([]string{"categories"})
		},
	}
}

func (mc *MarketplaceCobraCommands) createMyPublicationsCommand() *cobra.Command {
	return &cobra.Command{
		Use:     "my-publications",
		Aliases: []string{"my-pubs", "mine"},
		Short:   "Show your published templates",
		Long:    `Show templates that you have published to the marketplace.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return mc.app.Marketplace([]string{"my-publications"})
		},
	}
}