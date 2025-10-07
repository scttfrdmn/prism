// Package cli - Template Cobra Command Layer
//
// ARCHITECTURE NOTE: This file defines the user-facing CLI interface for template commands.
// The actual business logic is in template_impl.go (TemplateCommands).
//
// This separation follows the Facade/Adapter pattern:
//   - templates_cobra.go: CLI interface (THIS FILE - Cobra commands, flag parsing, help text)
//   - template_impl.go: Business logic (API calls, formatting, error handling)
//
// This Cobra layer is responsible for:
//   - Defining command structure and subcommands
//   - Parsing and validating flags
//   - Providing help text and examples
//   - Delegating to TemplateCommands for execution
package cli

import (
	"github.com/spf13/cobra"
)

// TemplateCobraCommands creates the templates command with proper Cobra subcommands (Cobra layer)
type TemplateCobraCommands struct {
	app              *App
	templateCommands *TemplateCommands
}

// NewTemplateCobraCommands creates new template cobra commands
func NewTemplateCobraCommands(app *App) *TemplateCobraCommands {
	return &TemplateCobraCommands{
		app:              app,
		templateCommands: NewTemplateCommands(app),
	}
}

// CreateTemplatesCommand creates the main templates command with all subcommands
func (tc *TemplateCobraCommands) CreateTemplatesCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "templates",
		Short: "Manage and explore CloudWorkstation templates",
		Long: `Manage CloudWorkstation templates including listing, searching, validating,
and testing templates. Use subcommands for specific operations.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Default action: list templates
			return tc.templateCommands.templatesList(args)
		},
	}

	// Add subcommands
	cmd.AddCommand(
		tc.createListCommand(),
		tc.createSearchCommand(),
		tc.createInfoCommand(),
		tc.createValidateCommand(),
		tc.createTestCommand(),
		tc.createDiscoverCommand(),
		tc.createUsageCommand(),
		tc.createInstallCommand(),
		tc.createVersionCommand(),
		tc.createSnapshotCommand(),
	)

	return cmd
}

// createListCommand creates the list subcommand
func (tc *TemplateCobraCommands) createListCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all available templates",
		Long:  "Display all available CloudWorkstation templates with their descriptions and costs.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return tc.templateCommands.templatesList(args)
		},
	}
}

// createSearchCommand creates the search subcommand
func (tc *TemplateCobraCommands) createSearchCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "search [query]",
		Short: "Search templates",
		Long: `Search for templates by name, description, category, or tags.
You can filter results using various flags.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Build args array with flags
			var searchArgs []string

			// Add query if provided
			if len(args) > 0 {
				searchArgs = append(searchArgs, args[0])
			}

			// Add flags
			if category, _ := cmd.Flags().GetString("category"); category != "" {
				searchArgs = append(searchArgs, "--category", category)
			}
			if domain, _ := cmd.Flags().GetString("domain"); domain != "" {
				searchArgs = append(searchArgs, "--domain", domain)
			}
			if complexity, _ := cmd.Flags().GetString("complexity"); complexity != "" {
				searchArgs = append(searchArgs, "--complexity", complexity)
			}
			if popular, _ := cmd.Flags().GetBool("popular"); popular {
				searchArgs = append(searchArgs, "--popular")
			}
			if featured, _ := cmd.Flags().GetBool("featured"); featured {
				searchArgs = append(searchArgs, "--featured")
			}

			return tc.templateCommands.templatesSearch(searchArgs)
		},
	}

	// Add search-specific flags
	cmd.Flags().String("category", "", "Filter by category")
	cmd.Flags().String("domain", "", "Filter by domain (ml, datascience, bio, web)")
	cmd.Flags().String("complexity", "", "Filter by complexity (simple, moderate, advanced)")
	cmd.Flags().Bool("popular", false, "Show only popular templates")
	cmd.Flags().Bool("featured", false, "Show only featured templates")

	return cmd
}

// createInfoCommand creates the info subcommand
func (tc *TemplateCobraCommands) createInfoCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "info <template-name>",
		Short: "Show detailed template information",
		Long:  "Display comprehensive information about a specific template including all metadata and configuration.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return tc.templateCommands.templatesInfo(args)
		},
	}
}

// createValidateCommand creates the validate subcommand
func (tc *TemplateCobraCommands) createValidateCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "validate [template-name]",
		Short: "Validate template configuration",
		Long: `Validate one or all templates for correctness, security, and best practices.
Without a template name, validates all templates.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Build args with flags
			var validateArgs []string

			if len(args) > 0 {
				validateArgs = append(validateArgs, args[0])
			}

			if verbose, _ := cmd.Flags().GetBool("verbose"); verbose {
				validateArgs = append(validateArgs, "--verbose")
			}
			if strict, _ := cmd.Flags().GetBool("strict"); strict {
				validateArgs = append(validateArgs, "--strict")
			}

			return tc.templateCommands.validateTemplates(validateArgs)
		},
	}

	// Add validation flags
	cmd.Flags().BoolP("verbose", "v", false, "Show detailed validation output")
	cmd.Flags().Bool("strict", false, "Treat warnings as errors")

	return cmd
}

// createTestCommand creates the test subcommand
func (tc *TemplateCobraCommands) createTestCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "test [template-name]",
		Short: "Run template tests",
		Long: `Run comprehensive test suites against templates to verify functionality,
compatibility, performance, and security.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Build args with flags
			var testArgs []string

			if len(args) > 0 {
				testArgs = append(testArgs, args[0])
			}

			if suite, _ := cmd.Flags().GetString("suite"); suite != "" {
				testArgs = append(testArgs, "--suite", suite)
			}
			if verbose, _ := cmd.Flags().GetBool("verbose"); verbose {
				testArgs = append(testArgs, "--verbose")
			}

			return tc.templateCommands.templatesTest(testArgs)
		},
	}

	// Add test flags
	cmd.Flags().String("suite", "", "Run specific test suite (syntax, compatibility, performance, security, integration)")
	cmd.Flags().BoolP("verbose", "v", false, "Show detailed test output")

	return cmd
}

// createDiscoverCommand creates the discover subcommand
func (tc *TemplateCobraCommands) createDiscoverCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "discover",
		Short: "Discover templates by category",
		Long:  "Browse and discover CloudWorkstation templates organized by category and research domain.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return tc.templateCommands.templatesDiscover(args)
		},
	}
}

// createUsageCommand creates the usage subcommand
func (tc *TemplateCobraCommands) createUsageCommand() *cobra.Command {
	return &cobra.Command{
		Use:     "usage",
		Aliases: []string{"stats"},
		Short:   "Show template usage statistics",
		Long:    "Display template usage statistics including popularity, recent usage, and recommendations.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return tc.templateCommands.templatesUsage(args)
		},
	}
}

// createInstallCommand creates the install subcommand
func (tc *TemplateCobraCommands) createInstallCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "install <repository:template>",
		Short: "Install template from repository",
		Long:  "Install a template from a CloudWorkstation repository.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			// Build args with flags
			installArgs := []string{args[0]}

			if force, _ := cmd.Flags().GetBool("force"); force {
				installArgs = append(installArgs, "--force")
			}
			if version, _ := cmd.Flags().GetString("version"); version != "" {
				installArgs = append(installArgs, "--version", version)
			}

			return tc.templateCommands.templatesInstall(installArgs)
		},
	}

	// Add install flags
	cmd.Flags().Bool("force", false, "Force reinstall even if template exists")
	cmd.Flags().String("version", "", "Install specific template version")

	return cmd
}

// createVersionCommand creates the version subcommand
func (tc *TemplateCobraCommands) createVersionCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "version",
		Short: "Manage template versions",
		Long:  "Manage template versions including listing, getting, setting, and validating versions.",
	}

	// Add version subcommands
	cmd.AddCommand(
		&cobra.Command{
			Use:   "list <template>",
			Short: "List all versions of a template",
			Args:  cobra.ExactArgs(1),
			RunE: func(cmd *cobra.Command, args []string) error {
				return tc.templateCommands.templatesVersionList(args)
			},
		},
		&cobra.Command{
			Use:   "get <template>",
			Short: "Get current version of a template",
			Args:  cobra.ExactArgs(1),
			RunE: func(cmd *cobra.Command, args []string) error {
				return tc.templateCommands.templatesVersionGet(args)
			},
		},
		&cobra.Command{
			Use:   "validate",
			Short: "Validate all template versions",
			RunE: func(cmd *cobra.Command, args []string) error {
				return tc.templateCommands.templatesVersionValidate(args)
			},
		},
	)

	return cmd
}

// createSnapshotCommand creates the snapshot subcommand
func (tc *TemplateCobraCommands) createSnapshotCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "snapshot <instance-name>",
		Short: "Create template from running instance",
		Long: `Create a new template by capturing the current state of a running instance,
including installed packages, services, and users.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			// Build args with flags
			snapshotArgs := []string{args[0]}

			if name, _ := cmd.Flags().GetString("name"); name != "" {
				snapshotArgs = append(snapshotArgs, "--name", name)
			}
			if description, _ := cmd.Flags().GetString("description"); description != "" {
				snapshotArgs = append(snapshotArgs, "--description", description)
			}
			if save, _ := cmd.Flags().GetBool("save"); save {
				snapshotArgs = append(snapshotArgs, "--save")
			}

			return tc.templateCommands.templatesSnapshot(snapshotArgs)
		},
	}

	// Add snapshot flags
	cmd.Flags().String("name", "", "Name for the new template")
	cmd.Flags().String("description", "", "Description for the new template")
	cmd.Flags().Bool("save", false, "Save template to templates directory")

	return cmd
}
