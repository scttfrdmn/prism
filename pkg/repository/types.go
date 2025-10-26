// Package repository provides multi-repository support for Prism templates.
//
// This package implements the core functionality for managing multiple template
// repositories with priority-based override capabilities. It handles repository
// configuration, caching, and template resolution across repositories.
package repository

import (
	"time"
)

// Repository represents a template repository configuration.
type Repository struct {
	// Name is the unique identifier for the repository
	Name string `json:"name"`

	// Type is the repository type (github, local, s3)
	Type string `json:"type"`

	// URL is the repository URL for github repositories
	URL string `json:"url,omitempty"`

	// Branch is the git branch to use for github repositories
	Branch string `json:"branch,omitempty"`

	// Path is the local filesystem path for local repositories
	Path string `json:"path,omitempty"`

	// Bucket is the S3 bucket name for s3 repositories
	Bucket string `json:"bucket,omitempty"`

	// Prefix is the S3 object prefix for s3 repositories
	Prefix string `json:"prefix,omitempty"`

	// Region is the AWS region for s3 repositories
	Region string `json:"region,omitempty"`

	// Priority determines the repository precedence (higher number = higher priority)
	Priority int `json:"priority"`

	// LastUpdated is the timestamp of the last repository update
	LastUpdated time.Time `json:"last_updated,omitempty"`

	// Metadata contains the parsed repository metadata
	Metadata *RepositoryMetadata `json:"-"`
}

// RepositoryMetadata represents the metadata for a repository.
type RepositoryMetadata struct {
	// Name is the human-readable name of the repository
	Name string `yaml:"name" json:"name"`

	// Description is the repository description
	Description string `yaml:"description" json:"description"`

	// Maintainer is the repository maintainer name
	Maintainer string `yaml:"maintainer" json:"maintainer"`

	// Website is the repository website URL
	Website string `yaml:"website" json:"website"`

	// ContactEmail is the repository contact email
	ContactEmail string `yaml:"contact_email" json:"contact_email"`

	// Version is the repository version
	Version string `yaml:"version" json:"version"`

	// LastUpdated is the timestamp of the last repository update
	LastUpdated string `yaml:"last_updated" json:"last_updated"`

	// Compatibility defines version requirements for Prism
	Compatibility struct {
		// MinVersion is the minimum required Prism version
		MinVersion string `yaml:"min_version" json:"min_version"`

		// MaxVersion is the maximum supported Prism version
		MaxVersion string `yaml:"max_version" json:"max_version"`
	} `yaml:"compatibility" json:"compatibility"`

	// Templates is a list of templates in the repository
	Templates []TemplateMetadata `yaml:"templates" json:"templates"`
}

// TemplateMetadata represents metadata for a template in a repository.
type TemplateMetadata struct {
	// Name is the template name
	Name string `yaml:"name" json:"name"`

	// Path is the relative path to the template file
	Path string `yaml:"path" json:"path"`

	// Versions lists available template versions
	Versions []TemplateVersion `yaml:"versions" json:"versions"`
}

// TemplateVersion represents a specific version of a template.
type TemplateVersion struct {
	// Version is the semantic version of the template
	Version string `yaml:"version" json:"version"`

	// Date is the release date of the template version
	Date string `yaml:"date" json:"date"`
}

// Config represents the repository configuration stored in config.json.
type Config struct {
	// Repositories is a list of configured repositories
	Repositories []Repository `json:"repositories"`
}

// TemplateReference specifies a template with optional repository and version.
type TemplateReference struct {
	// Repository is the repository name (optional)
	Repository string

	// Template is the template name
	Template string

	// Version is the template version (optional)
	Version string
}

// RepositoryCache represents the local cache of repositories.
type RepositoryCache struct {
	// LastUpdated is the timestamp of the last cache update
	LastUpdated time.Time `json:"last_updated"`

	// Repositories maps repository names to cache entries
	Repositories map[string]RepositoryCacheEntry `json:"repositories"`
}

// RepositoryCacheEntry represents a cached repository.
type RepositoryCacheEntry struct {
	// LastUpdated is the timestamp of the last repository update
	LastUpdated time.Time `json:"last_updated"`

	// Path is the local cache path
	Path string `json:"path"`

	// Metadata contains the parsed repository metadata
	Metadata *RepositoryMetadata `json:"metadata"`
}

// DependencyGraph represents a dependency graph for template resolution.
type DependencyGraph struct {
	// Nodes maps template references to dependency nodes
	Nodes map[string]*DependencyNode

	// Resolved is the list of resolved templates in correct order
	Resolved []*DependencyNode
}

// DependencyNode represents a node in the dependency graph.
type DependencyNode struct {
	// Reference is the template reference
	Reference TemplateReference

	// Dependencies are the template's dependencies
	Dependencies []TemplateReference

	// Visited tracks graph traversal for cycle detection
	Visited bool

	// Resolved indicates the node has been resolved
	Resolved bool
}
