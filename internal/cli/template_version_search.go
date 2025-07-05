// Package cli implements CloudWorkstation's command-line interface application.
package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"text/tabwriter"

	"github.com/scttfrdmn/cloudworkstation/pkg/ami"
)

// handleTemplateVersionSearch handles searching for template versions
func (a *App) handleTemplateVersionSearch(args []string, manager *ami.TemplateManager) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: cws ami template version search <template-name> [--format <format>] [--min-version <version>]")
	}

	templateName := args[0]
	cmdArgs := parseCmdArgs(args[1:])
	
	// Default output format is table
	outputFormat := cmdArgs["format"]
	if outputFormat == "" {
		outputFormat = "table"
	}

	// Optional minimum version filter
	minVersion := cmdArgs["min-version"]

	// Prepare minimum version filter if specified
	var minVersionInfo *ami.VersionInfo
	var err error
	if minVersion != "" {
		minVersionInfo, err = ami.NewVersionInfo(minVersion)
		if err != nil {
			return fmt.Errorf("invalid minimum version format: %w", err)
		}
	}

	// Get template
	fmt.Printf("🔍 Searching for versions of template '%s'\n", templateName)

	// Check if registry is configured
	if manager.Registry == nil {
		return fmt.Errorf("template registry not configured")
	}

	// List available versions
	ctx := a.ctx
	versions, err := manager.Registry.ListSharedTemplateVersions(ctx, templateName)
	if err != nil {
		return fmt.Errorf("failed to list template versions: %w", err)
	}

	// If no versions found
	if len(versions) == 0 {
		fmt.Printf("No versions found for template '%s'\n", templateName)
		return nil
	}

	// Sort versions semantically
	sort.Slice(versions, func(i, j int) bool {
		// Try to parse as semantic versions
		vi, err1 := ami.NewVersionInfo(versions[i])
		vj, err2 := ami.NewVersionInfo(versions[j])
		
		// If parsing fails, fall back to string comparison
		if err1 != nil || err2 != nil {
			return versions[i] < versions[j]
		}
		
		// If both are valid semantic versions, compare properly
		return vj.IsGreaterThan(vi)
	})

	// Filter by minimum version if specified
	if minVersionInfo != nil {
		filteredVersions := make([]string, 0, len(versions))
		for _, v := range versions {
			// Parse version
			vInfo, err := ami.NewVersionInfo(v)
			if err != nil {
				// Skip invalid versions
				continue
			}
			
			// Check if version is greater than or equal to min version
			if vInfo.IsGreaterThan(minVersionInfo) || 
			   (vInfo.Major == minVersionInfo.Major && 
				vInfo.Minor == minVersionInfo.Minor && 
				vInfo.Patch == minVersionInfo.Patch) {
				filteredVersions = append(filteredVersions, v)
			}
		}
		versions = filteredVersions
	}

	// Display results based on format
	switch outputFormat {
	case "table":
		printVersionsTable(templateName, versions, manager, ctx)
	case "json":
		printVersionsJSON(templateName, versions, manager, ctx)
	case "simple":
		printVersionsSimple(versions)
	default:
		return fmt.Errorf("unsupported output format: %s (supported: table, json, simple)", outputFormat)
	}

	return nil
}

// printVersionsTable prints versions in a tabular format
func printVersionsTable(templateName string, versions []string, manager *ami.TemplateManager, ctx context.Context) {
	fmt.Printf("📋 Available versions for '%s':\n\n", templateName)
	
	// Create tabwriter for aligned output
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "VERSION\tPUBLISHED\tDESCRIPTION")
	
	for _, version := range versions {
		// Get template details
		entry, err := manager.Registry.GetSharedTemplate(ctx, templateName, version)
		if err != nil {
			// If can't get details, just print version
			fmt.Fprintf(w, "%s\t-\t-\n", version)
			continue
		}
		
		// Format published date
		publishedDate := "-"
		if !entry.PublishedAt.IsZero() {
			publishedDate = entry.PublishedAt.Format("2006-01-02 15:04")
		}
		
		// Format description
		description := entry.Description
		if len(description) > 40 {
			description = description[:37] + "..."
		}
		
		fmt.Fprintf(w, "%s\t%s\t%s\n", version, publishedDate, description)
	}
	
	w.Flush()
}

// printVersionsJSON prints versions in JSON format
func printVersionsJSON(templateName string, versions []string, manager *ami.TemplateManager, ctx context.Context) {
	// Collect detailed information for each version
	versionDetails := []map[string]string{}
	
	for _, version := range versions {
		// Get template details
		entry, err := manager.Registry.GetSharedTemplate(ctx, templateName, version)
		if err != nil {
			// If can't get details, just include version
			versionDetails = append(versionDetails, map[string]string{
				"version": version,
			})
			continue
		}
		
		// Format published date
		publishedDate := ""
		if !entry.PublishedAt.IsZero() {
			publishedDate = entry.PublishedAt.Format("2006-01-02T15:04:05Z07:00")
		}
		
		// Add details
		versionDetails = append(versionDetails, map[string]string{
			"version":     version,
			"published":   publishedDate,
			"description": entry.Description,
			"publisher":   entry.PublishedBy,
			"format":      entry.Format,
		})
	}
	
	// Convert to JSON and print
	output, err := json.Marshal(versionDetails)
	if err != nil {
		fmt.Printf("Error generating JSON output: %v\n", err)
		return
	}
	
	fmt.Println(string(output))
}

// printVersionsSimple prints just the version strings
func printVersionsSimple(versions []string) {
	for _, version := range versions {
		fmt.Println(version)
	}
}

// handleTemplateDependencyGraph shows a visual representation of the dependency graph
func (a *App) handleTemplateDependencyGraph(args []string, manager *ami.TemplateManager) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: cws ami template dependency graph <template-name> [--format <format>]")
	}

	templateName := args[0]
	cmdArgs := parseCmdArgs(args[1:])
	
	// Default output format is text
	outputFormat := cmdArgs["format"]
	if outputFormat == "" {
		outputFormat = "text"
	}

	// Get dependency graph
	fmt.Printf("🔍 Generating dependency graph for template '%s'\n", templateName)
	graph, err := manager.GetDependencyGraph(templateName)
	if err != nil {
		return fmt.Errorf("failed to get dependency graph: %w", err)
	}

	if len(graph) <= 1 {
		fmt.Printf("Template '%s' has no dependencies\n", templateName)
		return nil
	}

	// Display graph based on format
	switch outputFormat {
	case "text":
		fmt.Printf("📋 Build order for template '%s':\n\n", templateName)
		for i, name := range graph {
			if i == len(graph)-1 {
				fmt.Printf("%d. %s (target template)\n", i+1, name)
			} else {
				fmt.Printf("%d. %s\n", i+1, name)
			}
		}
	case "dot":
		printDependencyGraphDot(templateName, graph, manager)
	default:
		return fmt.Errorf("unsupported output format: %s (supported: text, dot)", outputFormat)
	}

	return nil
}

// printDependencyGraphDot prints the dependency graph in Graphviz DOT format
func printDependencyGraphDot(templateName string, graph []string, manager *ami.TemplateManager) {
	fmt.Println("digraph G {")
	fmt.Println("  rankdir=\"LR\";")
	fmt.Println("  node [shape=box, style=filled, fillcolor=lightblue];")
	
	// Map of templates and their dependencies
	deps := make(map[string][]string)
	
	// Build dependency map
	for _, name := range graph {
		template, err := manager.GetTemplate(name)
		if err != nil {
			continue
		}
		
		deps[name] = make([]string, 0, len(template.Dependencies))
		for _, dep := range template.Dependencies {
			deps[name] = append(deps[name], dep.Name)
		}
	}
	
	// Output nodes
	for _, name := range graph {
		label := name
		if name == templateName {
			fmt.Printf("  \"%s\" [fillcolor=lightgreen, fontcolor=black];\n", name)
		} else {
			fmt.Printf("  \"%s\";\n", name)
		}
	}
	
	// Output edges
	for src, targets := range deps {
		for _, target := range targets {
			fmt.Printf("  \"%s\" -> \"%s\";\n", target, src)
		}
	}
	
	fmt.Println("}")
}