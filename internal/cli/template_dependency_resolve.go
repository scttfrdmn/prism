// Package cli implements CloudWorkstation's command-line interface application.
package cli

import (
	"fmt"
	"os"
	"text/tabwriter"
	
	"github.com/scttfrdmn/cloudworkstation/pkg/ami"
)

// handleTemplateDependencyResolve handles automatic dependency resolution
func (a *App) handleTemplateDependencyResolve(args []string, manager *ami.TemplateManager) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: cws ami template dependency resolve <template-name> [--fetch] [--format <format>]")
	}

	templateName := args[0]
	cmdArgs := parseCmdArgs(args[1:])
	
	// Create dependency resolver
	resolver := ami.NewDependencyResolver(manager)
	
	// Determine whether to fetch missing dependencies
	fetchMissing := cmdArgs["fetch"] != ""
	
	// Default output format is table
	outputFormat := cmdArgs["format"]
	if outputFormat == "" {
		outputFormat = "table"
	}

	fetchMsg := ""
	if fetchMissing {
		fetchMsg = " (with fetching)"
	}
	fmt.Printf("🔍 Resolving dependencies for template '%s'%s\n", 
		templateName, fetchMsg)
		
	// Resolve dependencies
	resolved, fetched, err := resolver.ResolveAndFetchDependencies(templateName, fetchMissing)
	if err != nil {
		return fmt.Errorf("dependency resolution failed: %w", err)
	}
	
	// Get build order
	graph, err := manager.GetDependencyGraph(templateName)
	if err != nil {
		fmt.Printf("⚠️  Warning: Unable to determine build order: %v\n", err)
	}

	// Display results based on format
	switch outputFormat {
	case "table":
		printResolvedDependenciesTable(templateName, resolved, graph, fetched)
	case "json":
		printResolvedDependenciesJSON(templateName, resolved, graph, fetched)
	default:
		return fmt.Errorf("unsupported output format: %s (supported: table, json)", outputFormat)
	}
	
	return nil
}

// printResolvedDependenciesTable prints resolved dependencies in a tabular format
func printResolvedDependenciesTable(templateName string, resolved map[string]*ami.ResolvedDependency, 
	graph []string, fetched []string) {
	
	fmt.Printf("📋 Resolved dependencies for '%s':\n\n", templateName)
	
	// Create tabwriter for aligned output
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "DEPENDENCY\tVERSION\tSTATUS\tOPTIONAL\tNOTES")
	
	// Print in build order if available
	if len(graph) > 0 {
		// Remove the target template from the graph
		depGraph := graph[:len(graph)-1]
		
		for _, name := range depGraph {
			if dep, ok := resolved[name]; ok {
				printDependencyRow(w, dep, fetched)
			}
		}
	} else {
		// Print in any order
		for _, dep := range resolved {
			printDependencyRow(w, dep, fetched)
		}
	}
	
	w.Flush()
	
	// Print build order if available
	if len(graph) > 1 {
		fmt.Printf("\n📦 Build Order:\n")
		for i, name := range graph {
			if i == len(graph)-1 {
				fmt.Printf("  %d. %s (target template)\n", i+1, name)
			} else {
				fmt.Printf("  %d. %s\n", i+1, name)
			}
		}
	}
}

// printDependencyRow prints a single dependency row in the table
func printDependencyRow(w *tabwriter.Writer, dep *ami.ResolvedDependency, fetched []string) {
	// Check if this dependency was fetched
	wasFetched := false
	for _, name := range fetched {
		if name == dep.Name {
			wasFetched = true
			break
		}
	}
	
	// Determine notes
	notes := ""
	if wasFetched {
		notes = "fetched from registry"
	}
	
	// Pretty status
	status := dep.Status
	if status == "satisfied" {
		status = "✅ satisfied"
	} else if status == "missing" {
		status = "❌ missing"
	} else if status == "version-mismatch" {
		status = "⚠️ version mismatch"
	}
	
	fmt.Fprintf(w, "%s\t%s\t%s\t%v\t%s\n", 
		dep.Name, 
		dep.Version, 
		status, 
		dep.IsOptional,
		notes)
}

// printResolvedDependenciesJSON prints resolved dependencies in JSON format
func printResolvedDependenciesJSON(templateName string, resolved map[string]*ami.ResolvedDependency,
	graph []string, fetched []string) {
	
	// TODO: Implement JSON output format
	fmt.Printf("JSON output format not implemented yet\n")
}

// handleTemplateDependencyAnalyze analyzes dependencies for a template
func (a *App) handleTemplateDependencyAnalyze(args []string, manager *ami.TemplateManager) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: cws ami template dependency analyze <template-name>")
	}

	templateName := args[0]
	fmt.Printf("🔬 Analyzing dependencies for template '%s'\n", templateName)
	
	// Create dependency resolver
	resolver := ami.NewDependencyResolver(manager)
	
	// Resolve dependencies (without fetching)
	resolved, _, err := resolver.ResolveDependencies(templateName)
	if err != nil && len(resolved) == 0 {
		return fmt.Errorf("dependency analysis failed: %w", err)
	}
	
	// Count dependency statuses
	var satisfied, missing, mismatch, optional int
	for _, dep := range resolved {
		if dep.Status == "satisfied" {
			satisfied++
		} else if dep.Status == "missing" {
			missing++
			if dep.IsOptional {
				optional++
			}
		} else if dep.Status == "version-mismatch" {
			mismatch++
		}
	}
	
	// Print summary
	fmt.Printf("\n📊 Dependency Analysis Summary:\n")
	fmt.Printf("  Total dependencies:   %d\n", len(resolved))
	fmt.Printf("  Satisfied:            %d\n", satisfied)
	fmt.Printf("  Missing (required):   %d\n", missing - optional)
	fmt.Printf("  Missing (optional):   %d\n", optional)
	fmt.Printf("  Version mismatch:     %d\n", mismatch)
	
	// Template is buildable if all required dependencies are satisfied
	buildable := (missing - optional) == 0 && mismatch == 0
	if buildable {
		fmt.Printf("\n✅ Template is buildable - all required dependencies are satisfied\n")
	} else {
		fmt.Printf("\n❌ Template is not buildable - missing required dependencies or version mismatches\n")
		fmt.Printf("\nTo resolve these issues:\n")
		fmt.Printf("  - Use 'cws ami template dependency resolve %s --fetch' to fetch missing dependencies\n", templateName)
		fmt.Printf("  - Manually update dependency versions with 'cws ami template dependency add'\n")
	}
	
	return nil
}