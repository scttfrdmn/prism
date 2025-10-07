package cli

import (
	"github.com/spf13/cobra"
)

// RepoCobraCommands handles repository management commands
type RepoCobraCommands struct {
	app *App
}

// NewRepoCobraCommands creates new repository cobra commands
func NewRepoCobraCommands(app *App) *RepoCobraCommands {
	return &RepoCobraCommands{app: app}
}

// CreateRepoCommand creates the repo command with subcommands
func (rc *RepoCobraCommands) CreateRepoCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "repo",
		Short: "Manage template repositories",
		Long: `Add, remove, and manage CloudWorkstation template repositories.

Repositories allow you to access templates from different sources including
institutional repositories, research group collections, and private repositories.`,
	}

	// Add all repository subcommands
	cmd.AddCommand(
		rc.createListCommand(),
		rc.createAddCommand(),
		rc.createRemoveCommand(),
		rc.createUpdateCommand(),
		rc.createInfoCommand(),
		rc.createTemplatesCommand(),
		rc.createSearchCommand(),
		rc.createPullCommand(),
		rc.createPushCommand(),
	)

	return cmd
}

func (rc *RepoCobraCommands) createListCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List configured repositories",
		Long:  `List all currently configured template repositories.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			verbose, _ := cmd.Flags().GetBool("verbose")

			repoArgs := []string{"list"}
			if verbose {
				repoArgs = append(repoArgs, "--verbose")
			}

			return rc.app.Repo(repoArgs)
		},
	}

	cmd.Flags().BoolP("verbose", "v", false, "Show detailed repository information")

	return cmd
}

func (rc *RepoCobraCommands) createAddCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "add <name> <url>",
		Short: "Add a new template repository",
		Long: `Add a new template repository to your configuration.

The repository URL can be:
- Git repository URL (https://github.com/org/templates.git)
- HTTP/HTTPS URL to a template index
- Local file path to a repository directory`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			branch, _ := cmd.Flags().GetString("branch")
			token, _ := cmd.Flags().GetString("token")
			public, _ := cmd.Flags().GetBool("public")

			repoArgs := []string{"add", args[0], args[1]}
			if branch != "" {
				repoArgs = append(repoArgs, "--branch", branch)
			}
			if token != "" {
				repoArgs = append(repoArgs, "--token", token)
			}
			if public {
				repoArgs = append(repoArgs, "--public")
			}

			return rc.app.Repo(repoArgs)
		},
	}

	cmd.Flags().String("branch", "", "Git branch to use (default: main)")
	cmd.Flags().String("token", "", "Authentication token for private repositories")
	cmd.Flags().Bool("public", false, "Repository is publicly accessible")

	return cmd
}

func (rc *RepoCobraCommands) createRemoveCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "remove <name>",
		Aliases: []string{"rm", "delete"},
		Short:   "Remove a template repository",
		Long:    `Remove a template repository from your configuration.`,
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			force, _ := cmd.Flags().GetBool("force")

			repoArgs := []string{"remove", args[0]}
			if force {
				repoArgs = append(repoArgs, "--force")
			}

			return rc.app.Repo(repoArgs)
		},
	}

	cmd.Flags().Bool("force", false, "Force removal without confirmation")

	return cmd
}

func (rc *RepoCobraCommands) createUpdateCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "update [name]",
		Aliases: []string{"sync", "refresh"},
		Short:   "Update repository templates",
		Long: `Update templates from repositories.

If no repository name is specified, all repositories will be updated.`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			force, _ := cmd.Flags().GetBool("force")

			repoArgs := []string{"update"}
			if len(args) > 0 {
				repoArgs = append(repoArgs, args[0])
			}
			if force {
				repoArgs = append(repoArgs, "--force")
			}

			return rc.app.Repo(repoArgs)
		},
	}

	cmd.Flags().Bool("force", false, "Force update even if repository is up to date")

	return cmd
}

func (rc *RepoCobraCommands) createInfoCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "info <name>",
		Short: "Show detailed repository information",
		Long:  `Show detailed information about a specific repository.`,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return rc.app.Repo([]string{"info", args[0]})
		},
	}
}

func (rc *RepoCobraCommands) createTemplatesCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "templates <name>",
		Short: "List templates in a repository",
		Long:  `List all templates available in a specific repository.`,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			category, _ := cmd.Flags().GetString("category")

			repoArgs := []string{"templates", args[0]}
			if category != "" {
				repoArgs = append(repoArgs, "--category", category)
			}

			return rc.app.Repo(repoArgs)
		},
	}

	cmd.Flags().String("category", "", "Filter templates by category")

	return cmd
}

func (rc *RepoCobraCommands) createSearchCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "search <query>",
		Short: "Search templates across repositories",
		Long:  `Search for templates across all configured repositories.`,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			repo, _ := cmd.Flags().GetString("repo")
			category, _ := cmd.Flags().GetString("category")

			repoArgs := []string{"search", args[0]}
			if repo != "" {
				repoArgs = append(repoArgs, "--repo", repo)
			}
			if category != "" {
				repoArgs = append(repoArgs, "--category", category)
			}

			return rc.app.Repo(repoArgs)
		},
	}

	cmd.Flags().String("repo", "", "Search only in specific repository")
	cmd.Flags().String("category", "", "Filter by template category")

	return cmd
}

func (rc *RepoCobraCommands) createPullCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "pull <template-name>",
		Short: "Pull specific template from repository",
		Long:  `Pull a specific template from its source repository.`,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return rc.app.Repo([]string{"pull", args[0]})
		},
	}
}

func (rc *RepoCobraCommands) createPushCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "push <template-name> <repo-name>",
		Short: "Push template to repository",
		Long:  `Push a local template to a configured repository.`,
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			message, _ := cmd.Flags().GetString("message")

			repoArgs := []string{"push", args[0], args[1]}
			if message != "" {
				repoArgs = append(repoArgs, "--message", message)
			}

			return rc.app.Repo(repoArgs)
		},
	}

	cmd.Flags().StringP("message", "m", "", "Commit message for push")

	return cmd
}
