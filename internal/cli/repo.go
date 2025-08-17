package cli

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"text/tabwriter"

	"github.com/scttfrdmn/cloudworkstation/pkg/repository"
)

// Repo handles repository-related commands.
func (a *App) Repo(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("repo command requires a subcommand")
	}

	subcommand := args[0]
	subargs := args[1:]

	switch subcommand {
	case "list":
		return a.repoList(subargs)
	case "add":
		return a.repoAdd(subargs)
	case "remove":
		return a.repoRemove(subargs)
	case "update":
		return a.repoUpdate(subargs)
	case "info":
		return a.repoInfo(subargs)
	case "templates":
		return a.repoTemplates(subargs)
	case "search":
		return a.repoSearch(subargs)
	case "pull":
		return a.repoPull(subargs)
	case "push":
		return a.repoPush(subargs)
	default:
		return fmt.Errorf("unknown repo subcommand: %s", subcommand)
	}
}

// repoList lists all configured repositories.
func (a *App) repoList(args []string) error {
	// Create repository manager
	repoManager, err := repository.NewManager()
	if err != nil {
		return fmt.Errorf("failed to initialize repository manager: %w", err)
	}

	// Get repositories
	repos := repoManager.GetRepositories()

	// Print repositories
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintln(w, "NAME\tTYPE\tURL/PATH\tPRIORITY")
	for _, repo := range repos {
		var location string
		switch repo.Type {
		case "github":
			location = repo.URL
		case "local":
			location = repo.Path
		case "s3":
			location = fmt.Sprintf("s3://%s/%s", repo.Bucket, repo.Prefix)
		}
		_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%d\n", repo.Name, repo.Type, location, repo.Priority)
	}
	return w.Flush()
}

// repoAdd adds a new repository.
func (a *App) repoAdd(args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("repo add requires name and URL/path arguments")
	}

	name := args[0]
	location := args[1]
	repoType := "github"
	priority := 10 // Default medium priority
	branch := "main"

	// Parse flags
	for i := 2; i < len(args); i++ {
		switch {
		case args[i] == "--type" && i+1 < len(args):
			repoType = args[i+1]
			i++
		case args[i] == "--priority" && i+1 < len(args):
			p, err := strconv.Atoi(args[i+1])
			if err != nil {
				return fmt.Errorf("invalid priority: %s", args[i+1])
			}
			priority = p
			i++
		case args[i] == "--branch" && i+1 < len(args):
			branch = args[i+1]
			i++
		}
	}

	// Create repository manager
	repoManager, err := repository.NewManager()
	if err != nil {
		return fmt.Errorf("failed to initialize repository manager: %w", err)
	}

	// Create repository
	repo := repository.Repository{
		Name:     name,
		Type:     repoType,
		Priority: priority,
	}

	switch repoType {
	case "github":
		repo.URL = location
		repo.Branch = branch
	case "local":
		repo.Path = location
	case "s3":
		parts := strings.SplitN(strings.TrimPrefix(location, "s3://"), "/", 2)
		if len(parts) != 2 {
			return fmt.Errorf("invalid S3 URL: %s", location)
		}
		repo.Bucket = parts[0]
		repo.Prefix = parts[1]
	default:
		return fmt.Errorf("unsupported repository type: %s", repoType)
	}

	// Add repository
	if err := repoManager.AddRepository(repo); err != nil {
		return fmt.Errorf("failed to add repository: %w", err)
	}

	fmt.Printf("Added repository %q with priority %d\n", name, priority)
	return nil
}

// repoRemove removes a repository.
func (a *App) repoRemove(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("repo remove requires a name argument")
	}

	name := args[0]

	// Create repository manager
	repoManager, err := repository.NewManager()
	if err != nil {
		return fmt.Errorf("failed to initialize repository manager: %w", err)
	}

	// Remove repository
	if err := repoManager.RemoveRepository(name); err != nil {
		return fmt.Errorf("failed to remove repository: %w", err)
	}

	fmt.Printf("Removed repository %q\n", name)
	return nil
}

// repoUpdate updates one or all repositories.
func (a *App) repoUpdate(args []string) error {
	// Create repository manager
	repoManager, err := repository.NewManager()
	if err != nil {
		return fmt.Errorf("failed to initialize repository manager: %w", err)
	}

	// If a name is provided, update only that repository
	if len(args) > 0 {
		name := args[0]
		repo, err := repoManager.GetRepository(name)
		if err != nil {
			return fmt.Errorf("failed to get repository: %w", err)
		}

		fmt.Printf("Updating repository %q...\n", name)
		if err := repoManager.UpdateRepositoryCache(repo); err != nil {
			return fmt.Errorf("failed to update repository: %w", err)
		}

		fmt.Printf("Repository %q updated successfully\n", name)
		return nil
	}

	// Update all repositories
	fmt.Println("Updating all repositories...")
	repos := repoManager.GetRepositories()
	for _, repo := range repos {
		fmt.Printf("Updating repository %q...\n", repo.Name)
		if err := repoManager.UpdateRepositoryCache(&repo); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to update repository %q: %v\n", repo.Name, err)
			continue
		}
	}

	fmt.Println("Repositories updated successfully")
	return nil
}

// repoInfo shows information about a repository.
func (a *App) repoInfo(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("repo info requires a name argument")
	}

	name := args[0]

	// Create repository manager
	repoManager, err := repository.NewManager()
	if err != nil {
		return fmt.Errorf("failed to initialize repository manager: %w", err)
	}

	// Get repository
	repo, err := repoManager.GetRepository(name)
	if err != nil {
		return fmt.Errorf("failed to get repository: %w", err)
	}

	// Get metadata
	metadata, err := repoManager.GetRepositoryMetadata(name)
	if err != nil {
		return fmt.Errorf("failed to get repository metadata: %w", err)
	}

	// Print repository information
	fmt.Printf("Repository: %s\n", name)
	fmt.Printf("Type: %s\n", repo.Type)
	switch repo.Type {
	case "github":
		fmt.Printf("URL: %s\n", repo.URL)
		fmt.Printf("Branch: %s\n", repo.Branch)
	case "local":
		fmt.Printf("Path: %s\n", repo.Path)
	case "s3":
		fmt.Printf("Bucket: %s\n", repo.Bucket)
		fmt.Printf("Prefix: %s\n", repo.Prefix)
	}
	fmt.Printf("Priority: %d\n", repo.Priority)
	fmt.Printf("Description: %s\n", metadata.Description)
	fmt.Printf("Maintainer: %s\n", metadata.Maintainer)
	fmt.Printf("Website: %s\n", metadata.Website)
	fmt.Printf("Version: %s\n", metadata.Version)
	fmt.Printf("Last Updated: %s\n", metadata.LastUpdated)
	fmt.Printf("Templates: %d\n", len(metadata.Templates))
	return nil
}

// repoTemplates lists templates in a repository.
func (a *App) repoTemplates(args []string) error {
	// Create repository manager
	repoManager, err := repository.NewManager()
	if err != nil {
		return fmt.Errorf("failed to initialize repository manager: %w", err)
	}

	var repoName string
	if len(args) > 0 && args[0] != "--repo" {
		repoName = args[0]
	} else if len(args) > 1 && args[0] == "--repo" {
		repoName = args[1]
	}

	// If repository name is provided, list templates from that repository
	if repoName != "" {
		_, err := repoManager.GetRepository(repoName)
		if err != nil {
			return fmt.Errorf("failed to get repository: %w", err)
		}

		metadata, err := repoManager.GetRepositoryMetadata(repoName)
		if err != nil {
			return fmt.Errorf("failed to get repository metadata: %w", err)
		}

		// Print templates
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		_, _ = fmt.Fprintf(w, "Templates in repository %q:\n\n", repoName)
		_, _ = fmt.Fprintln(w, "NAME\tPATH\tVERSIONS")
		for _, template := range metadata.Templates {
			versions := []string{}
			for _, v := range template.Versions {
				versions = append(versions, v.Version)
			}
			_, _ = fmt.Fprintf(w, "%s\t%s\t%s\n", template.Name, template.Path, strings.Join(versions, ", "))
		}
		return w.Flush()
	}

	// List templates from all repositories
	repos := repoManager.GetRepositories()
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintln(w, "REPOSITORY\tTEMPLATE\tPATH\tVERSIONS")

	for _, repo := range repos {
		metadata, err := repoManager.GetRepositoryMetadata(repo.Name)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to get metadata for repository %q: %v\n", repo.Name, err)
			continue
		}

		for _, template := range metadata.Templates {
			versions := []string{}
			for _, v := range template.Versions {
				versions = append(versions, v.Version)
			}
			_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", repo.Name, template.Name, template.Path, strings.Join(versions, ", "))
		}
	}

	return w.Flush()
}

// repoSearch searches for templates across all repositories.
func (a *App) repoSearch(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("repo search requires a query argument")
	}

	query := strings.ToLower(args[0])

	// Create repository manager
	repoManager, err := repository.NewManager()
	if err != nil {
		return fmt.Errorf("failed to initialize repository manager: %w", err)
	}

	// Search all repositories
	repos := repoManager.GetRepositories()
	found := false

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintf(w, "Search results for %q:\n\n", query)
	_, _ = fmt.Fprintln(w, "REPOSITORY\tTEMPLATE\tPATH\tVERSIONS")

	for _, repo := range repos {
		metadata, err := repoManager.GetRepositoryMetadata(repo.Name)
		if err != nil {
			continue
		}

		for _, template := range metadata.Templates {
			// Check if template name or path contains query
			if strings.Contains(strings.ToLower(template.Name), query) || strings.Contains(strings.ToLower(template.Path), query) {
				versions := []string{}
				for _, v := range template.Versions {
					versions = append(versions, v.Version)
				}
				_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", repo.Name, template.Name, template.Path, strings.Join(versions, ", "))
				found = true
			}
		}
	}

	if !found {
		_, _ = fmt.Fprintf(w, "No templates found matching %q\n", query)
	}

	return w.Flush()
}

// repoPull downloads a template from a repository.
func (a *App) repoPull(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("repo pull requires a template reference argument")
	}

	refStr := args[0]

	// Create repository manager
	repoManager, err := repository.NewManager()
	if err != nil {
		return fmt.Errorf("failed to initialize repository manager: %w", err)
	}

	// Parse template reference
	ref, err := repoManager.ParseTemplateReference(refStr)
	if err != nil {
		return fmt.Errorf("invalid template reference: %w", err)
	}

	// Find template
	template, repo, err := repoManager.FindTemplate(ref)
	if err != nil {
		return fmt.Errorf("failed to find template: %w", err)
	}

	fmt.Printf("Found template %q in repository %q\n", ref.Template, repo.Name)
	fmt.Printf("Path: %s\n", template.Path)

	// TODO: Implement template downloading
	fmt.Println("Template pull not fully implemented yet")

	return nil
}

// repoPush uploads a template to a repository.
func (a *App) repoPush(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("repo push requires a template file argument")
	}

	templateFile := args[0]
	repoName := "default"

	// Parse flags
	for i := 1; i < len(args); i++ {
		switch {
		case args[i] == "--repo" && i+1 < len(args):
			repoName = args[i+1]
			i++
		}
	}

	// Create repository manager
	repoManager, err := repository.NewManager()
	if err != nil {
		return fmt.Errorf("failed to initialize repository manager: %w", err)
	}

	// Get repository
	_, err = repoManager.GetRepository(repoName)
	if err != nil {
		return fmt.Errorf("failed to get repository: %w", err)
	}

	fmt.Printf("Pushing template %q to repository %q\n", templateFile, repoName)

	// TODO: Implement template uploading
	fmt.Println("Template push not fully implemented yet")

	return nil
}
