package repository

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// ResolveDependencies builds a dependency graph and resolves dependencies.
func (m *Manager) ResolveDependencies(ref TemplateReference) ([]*DependencyNode, error) {
	// Create dependency graph
	graph := &DependencyGraph{
		Nodes: make(map[string]*DependencyNode),
	}

	// Start with the main template
	if err := m.buildDependencyGraph(graph, ref); err != nil {
		return nil, err
	}

	// Resolve dependencies
	if err := m.resolveDependencyGraph(graph); err != nil {
		return nil, err
	}

	return graph.Resolved, nil
}

// buildDependencyGraph builds a dependency graph starting from the given template.
func (m *Manager) buildDependencyGraph(graph *DependencyGraph, ref TemplateReference) error {
	// Convert reference to string key
	key := refToKey(ref)

	// Check if already in graph
	if _, ok := graph.Nodes[key]; ok {
		return nil
	}

	// Find the template
	template, repo, err := m.FindTemplate(ref)
	if err != nil {
		return err
	}

	// Get template path to read dependencies
	templatePath := getTemplatePath(template, repo, m.cache)

	// Read template file to get dependencies
	dependencies, err := m.readTemplateDependencies(templatePath, ref.Repository)
	if err != nil {
		return fmt.Errorf("failed to read template dependencies: %w", err)
	}

	// Create node
	node := &DependencyNode{
		Reference:    ref,
		Dependencies: dependencies,
		Visited:      false,
		Resolved:     false,
	}

	// Add to graph
	graph.Nodes[key] = node

	// Process dependencies
	for _, depRef := range dependencies {
		if err := m.buildDependencyGraph(graph, depRef); err != nil {
			return err
		}
	}

	return nil
}

// getTemplatePath gets the path to a template file.
func getTemplatePath(template *TemplateMetadata, repo *Repository, cache *RepositoryCache) string {
	if repo.Type == "local" {
		return fmt.Sprintf("%s/%s", repo.Path, template.Path)
	}

	// Use cached path
	if entry, ok := cache.Repositories[repo.Name]; ok {
		return fmt.Sprintf("%s/%s", entry.Path, template.Path)
	}

	return template.Path
}

// resolveDependencyGraph resolves dependencies in the graph.
func (m *Manager) resolveDependencyGraph(graph *DependencyGraph) error {
	// Process all nodes
	for _, node := range graph.Nodes {
		if !node.Resolved {
			if err := m.resolveNode(graph, node); err != nil {
				return err
			}
		}
	}

	return nil
}

// resolveNode resolves a single node in the dependency graph.
func (m *Manager) resolveNode(graph *DependencyGraph, node *DependencyNode) error {
	// Check for circular dependencies
	if node.Visited {
		return fmt.Errorf("circular dependency detected: %s", refToKey(node.Reference))
	}

	node.Visited = true

	// Resolve dependencies first
	for _, depRef := range node.Dependencies {
		depKey := refToKey(depRef)
		depNode, ok := graph.Nodes[depKey]
		if !ok {
			return fmt.Errorf("dependency not found in graph: %s", depKey)
		}

		if !depNode.Resolved {
			if err := m.resolveNode(graph, depNode); err != nil {
				return err
			}
		}
	}

	// Mark as resolved and add to resolved list
	node.Resolved = true
	graph.Resolved = append(graph.Resolved, node)

	return nil
}

// refToKey converts a template reference to a string key.
func refToKey(ref TemplateReference) string {
	repo := ref.Repository
	if repo == "" {
		repo = "default"
	}

	version := ref.Version
	if version == "" {
		version = "latest"
	}

	return fmt.Sprintf("%s:%s@%s", repo, ref.Template, version)
}

// readTemplateDependencies reads a template file and extracts its dependencies
func (m *Manager) readTemplateDependencies(templatePath string, defaultRepo string) ([]TemplateReference, error) {
	// Read template file
	data, err := os.ReadFile(templatePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read template file: %w", err)
	}

	// Parse YAML to extract inherits field
	var templateData struct {
		Inherits []string `yaml:"inherits"`
	}

	if err := yaml.Unmarshal(data, &templateData); err != nil {
		return nil, fmt.Errorf("failed to parse template YAML: %w", err)
	}

	// Convert inherits to TemplateReferences
	dependencies := make([]TemplateReference, 0, len(templateData.Inherits))
	for _, inherit := range templateData.Inherits {
		// Parse the inherit string as a template reference
		// Format: [repo:]template[@version]
		ref := TemplateReference{
			Repository: defaultRepo, // Default to same repository
			Template:   inherit,
		}

		// Parse if it contains repository or version
		// Simple parsing: check for ':' (repository) and '@' (version)
		// For simplicity, assume inherit is just the template name
		// More complex parsing can be added if needed

		dependencies = append(dependencies, ref)
	}

	return dependencies, nil
}
