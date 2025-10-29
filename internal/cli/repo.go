package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"text/tabwriter"

	"github.com/scttfrdmn/prism/pkg/repository"
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

// RepositoryAddRequest represents repository creation parameters (Factory Pattern - SOLID)
type RepositoryAddRequest struct {
	Name     string
	Location string
	Type     string
	Priority int
	Branch   string
}

// RepositoryConfigParser parses CLI arguments for repository addition (Single Responsibility)
type RepositoryConfigParser struct{}

// Parse extracts repository configuration from CLI arguments
func (p *RepositoryConfigParser) Parse(args []string) (*RepositoryAddRequest, error) {
	if len(args) < 2 {
		return nil, fmt.Errorf("repo add requires name and URL/path arguments")
	}

	req := &RepositoryAddRequest{
		Name:     args[0],
		Location: args[1],
		Type:     "github",
		Priority: 10, // Default medium priority
		Branch:   "main",
	}

	return p.parseFlags(req, args[2:])
}

func (p *RepositoryConfigParser) parseFlags(req *RepositoryAddRequest, flagArgs []string) (*RepositoryAddRequest, error) {
	for i := 0; i < len(flagArgs); i++ {
		switch {
		case flagArgs[i] == "--type" && i+1 < len(flagArgs):
			req.Type = flagArgs[i+1]
			i++
		case flagArgs[i] == "--priority" && i+1 < len(flagArgs):
			p, err := strconv.Atoi(flagArgs[i+1])
			if err != nil {
				return nil, fmt.Errorf("invalid priority: %s", flagArgs[i+1])
			}
			req.Priority = p
			i++
		case flagArgs[i] == "--branch" && i+1 < len(flagArgs):
			req.Branch = flagArgs[i+1]
			i++
		}
	}
	return req, nil
}

// RepositoryFactory creates repository objects based on type (Factory Pattern - SOLID)
type RepositoryFactory struct{}

// CreateRepository creates repository object from request
func (f *RepositoryFactory) CreateRepository(req *RepositoryAddRequest) (*repository.Repository, error) {
	repo := &repository.Repository{
		Name:     req.Name,
		Type:     req.Type,
		Priority: req.Priority,
	}

	return f.configureRepositoryType(repo, req)
}

func (f *RepositoryFactory) configureRepositoryType(repo *repository.Repository, req *RepositoryAddRequest) (*repository.Repository, error) {
	switch req.Type {
	case "github":
		repo.URL = req.Location
		repo.Branch = req.Branch
	case "local":
		repo.Path = req.Location
	case "s3":
		return f.configureS3Repository(repo, req.Location)
	default:
		return nil, fmt.Errorf("unsupported repository type: %s", req.Type)
	}
	return repo, nil
}

func (f *RepositoryFactory) configureS3Repository(repo *repository.Repository, location string) (*repository.Repository, error) {
	parts := strings.SplitN(strings.TrimPrefix(location, "s3://"), "/", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid S3 URL: %s", location)
	}
	repo.Bucket = parts[0]
	repo.Prefix = parts[1]
	return repo, nil
}

// repoAdd adds a new repository using Factory Pattern (SOLID: Single Responsibility)
func (a *App) repoAdd(args []string) error {
	// Parse request
	parser := &RepositoryConfigParser{}
	req, err := parser.Parse(args)
	if err != nil {
		return err
	}

	// Create repository
	factory := &RepositoryFactory{}
	repo, err := factory.CreateRepository(req)
	if err != nil {
		return err
	}

	// Add to manager
	repoManager, err := repository.NewManager()
	if err != nil {
		return fmt.Errorf("failed to initialize repository manager: %w", err)
	}

	if err := repoManager.AddRepository(*repo); err != nil {
		return fmt.Errorf("failed to add repository: %w", err)
	}

	fmt.Printf("Added repository %q with priority %d\n", req.Name, req.Priority)
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

	// Get templates directory
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}
	templatesDir := filepath.Join(homeDir, ".prism", "templates")

	// Download template
	destFile, err := repoManager.DownloadTemplate(ref, templatesDir)
	if err != nil {
		return fmt.Errorf("failed to download template: %w", err)
	}

	fmt.Printf("\n✅ Template downloaded successfully\n")
	fmt.Printf("   File: %s\n", destFile)
	fmt.Printf("   Use: prism launch %s <workspace-name>\n", ref.Template)

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

	// Verify template file exists
	if _, err := os.Stat(templateFile); err != nil {
		return fmt.Errorf("template file not found: %w", err)
	}

	// Parse template reference for upload
	ref := repository.TemplateReference{
		Repository: repoName,
		Template:   filepath.Base(templateFile),
	}

	// Upload template
	if err := repoManager.UploadTemplate(templateFile, ref); err != nil {
		return fmt.Errorf("failed to upload template: %w", err)
	}

	fmt.Printf("\n✅ Template uploaded successfully\n")
	fmt.Printf("   Repository: %s\n", repoName)
	fmt.Printf("   Template: %s\n", filepath.Base(templateFile))

	return nil
}
