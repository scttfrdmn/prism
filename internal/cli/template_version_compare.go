// Package cli implements CloudWorkstation's command-line interface application.
package cli

import (
	"fmt"
	
	"github.com/scttfrdmn/cloudworkstation/pkg/ami"
)

// handleTemplateVersionCompare handles comparing template versions
func (a *App) handleTemplateVersionCompare(args []string, manager *ami.TemplateManager) error {
	if len(args) < 2 {
		return fmt.Errorf("usage: cws ami template version compare <version1> <version2>")
	}

	version1 := args[0]
	version2 := args[1]

	fmt.Printf("🔍 Comparing versions: %s vs %s\n", version1, version2)

	// Parse versions
	v1, err := ami.NewVersionInfo(version1)
	if err != nil {
		return fmt.Errorf("invalid version 1 format: %w", err)
	}

	v2, err := ami.NewVersionInfo(version2)
	if err != nil {
		return fmt.Errorf("invalid version 2 format: %w", err)
	}

	// Compare versions
	if v1.IsGreaterThan(v2) {
		fmt.Printf("Result: %s is greater than %s\n", version1, version2)
	} else if v2.IsGreaterThan(v1) {
		fmt.Printf("Result: %s is less than %s\n", version1, version2)
	} else {
		fmt.Printf("Result: %s is equal to %s\n", version1, version2)
	}

	// Show details about the comparison
	fmt.Printf("\nBreakdown:\n")
	fmt.Printf("  Major: %d vs %d\n", v1.Major, v2.Major)
	fmt.Printf("  Minor: %d vs %d\n", v1.Minor, v2.Minor)
	fmt.Printf("  Patch: %d vs %d\n", v1.Patch, v2.Patch)

	return nil
}