// Package templates provides template diff calculation capabilities.
package templates

import (
	"fmt"
	"strings"
)

// TemplateDiffCalculator calculates differences between current state and desired template
type TemplateDiffCalculator struct{}

// NewTemplateDiffCalculator creates a new template diff calculator
func NewTemplateDiffCalculator() *TemplateDiffCalculator {
	return &TemplateDiffCalculator{}
}

// CalculateDiff calculates the differences between current instance state and desired template
func (d *TemplateDiffCalculator) CalculateDiff(currentState *InstanceState, template *Template) (*TemplateDiff, error) {
	diff := &TemplateDiff{
		PackagesToInstall:   []PackageDiff{},
		PackagesToRemove:    []PackageDiff{},
		ServicesToConfigure: []ServiceDiff{},
		ServicesToStop:      []ServiceDiff{},
		UsersToCreate:       []UserDiff{},
		UsersToModify:       []UserDiff{},
		PortsToOpen:         []int{},
		ConflictsFound:      []ConflictDiff{},
	}

	// Calculate package differences
	if err := d.calculatePackageDiffs(currentState, template, diff); err != nil {
		return nil, fmt.Errorf("failed to calculate package diffs: %w", err)
	}

	// Calculate service differences
	if err := d.calculateServiceDiffs(currentState, template, diff); err != nil {
		return nil, fmt.Errorf("failed to calculate service diffs: %w", err)
	}

	// Calculate user differences
	if err := d.calculateUserDiffs(currentState, template, diff); err != nil {
		return nil, fmt.Errorf("failed to calculate user diffs: %w", err)
	}

	// Calculate port differences
	if err := d.calculatePortDiffs(currentState, template, diff); err != nil {
		return nil, fmt.Errorf("failed to calculate port diffs: %w", err)
	}

	// Check for conflicts
	d.detectConflicts(currentState, template, diff)

	return diff, nil
}

// calculatePackageDiffs calculates package installation/removal differences
func (d *TemplateDiffCalculator) calculatePackageDiffs(currentState *InstanceState, template *Template, diff *TemplateDiff) error {
	// Determine package manager to use
	packageManager := template.PackageManager
	if packageManager == "" {
		packageManager = currentState.PackageManager
	}

	// Get template packages based on package manager
	var templatePackages []string
	switch packageManager {
	case "apt", "dnf":
		templatePackages = template.Packages.System
	case "conda":
		templatePackages = append(template.Packages.Conda, template.Packages.Pip...)
	case "spack":
		templatePackages = template.Packages.Spack
	case "ami":
		// AMI-based templates shouldn't have packages to install
		return nil
	default:
		return fmt.Errorf("unsupported package manager: %s", packageManager)
	}

	// Find packages to install
	for _, pkgName := range templatePackages {
		// Parse package name (might include version like "python=3.11")
		baseName := strings.Split(pkgName, "=")[0]
		baseName = strings.Split(baseName, ">")[0]
		baseName = strings.Split(baseName, "<")[0]
		baseName = strings.TrimSpace(baseName)

		if !currentState.HasPackage(baseName) {
			diff.PackagesToInstall = append(diff.PackagesToInstall, PackageDiff{
				Name:           baseName,
				TargetVersion:  d.extractVersion(pkgName),
				Action:         "install",
				PackageManager: packageManager,
			})
		} else {
			// Check for version upgrade
			currentPkg := currentState.GetPackage(baseName)
			targetVersion := d.extractVersion(pkgName)
			if targetVersion != "" && currentPkg.Version != targetVersion {
				diff.PackagesToInstall = append(diff.PackagesToInstall, PackageDiff{
					Name:           baseName,
					CurrentVersion: currentPkg.Version,
					TargetVersion:  targetVersion,
					Action:         "upgrade",
					PackageManager: packageManager,
				})
			}
		}
	}

	return nil
}

// calculateServiceDiffs calculates service configuration differences
func (d *TemplateDiffCalculator) calculateServiceDiffs(currentState *InstanceState, template *Template, diff *TemplateDiff) error {
	for _, svc := range template.Services {
		currentService := currentState.GetService(svc.Name)

		if currentService == nil {
			// Service doesn't exist, needs to be configured
			diff.ServicesToConfigure = append(diff.ServicesToConfigure, ServiceDiff{
				Name:         svc.Name,
				TargetStatus: "running",
				Action:       "configure",
				Port:         svc.Port,
			})
		} else {
			// Service exists, check if it needs to be started
			if svc.Enable && currentService.Status != "running" {
				diff.ServicesToConfigure = append(diff.ServicesToConfigure, ServiceDiff{
					Name:          svc.Name,
					CurrentStatus: currentService.Status,
					TargetStatus:  "running",
					Action:        "start",
					Port:          svc.Port,
				})
			}
		}
	}

	return nil
}

// calculateUserDiffs calculates user account differences
func (d *TemplateDiffCalculator) calculateUserDiffs(currentState *InstanceState, template *Template, diff *TemplateDiff) error {
	for _, user := range template.Users {
		currentUser := currentState.GetUser(user.Name)

		if currentUser == nil {
			// User doesn't exist, needs to be created
			diff.UsersToCreate = append(diff.UsersToCreate, UserDiff{
				Name:         user.Name,
				TargetGroups: user.Groups,
				Action:       "create",
			})
		} else {
			// User exists, check if groups need to be modified
			if !d.groupsEqual(currentUser.Groups, user.Groups) {
				diff.UsersToModify = append(diff.UsersToModify, UserDiff{
					Name:          user.Name,
					CurrentGroups: currentUser.Groups,
					TargetGroups:  user.Groups,
					Action:        "modify",
				})
			}
		}
	}

	return nil
}

// calculatePortDiffs calculates port opening differences
func (d *TemplateDiffCalculator) calculatePortDiffs(currentState *InstanceState, template *Template, diff *TemplateDiff) error {
	currentPortSet := make(map[int]bool)
	for _, port := range currentState.Ports {
		currentPortSet[port] = true
	}

	// Check template service ports
	for _, svc := range template.Services {
		if svc.Port > 0 && !currentPortSet[svc.Port] {
			diff.PortsToOpen = append(diff.PortsToOpen, svc.Port)
		}
	}

	// Check template instance default ports
	for _, port := range template.InstanceDefaults.Ports {
		if !currentPortSet[port] {
			diff.PortsToOpen = append(diff.PortsToOpen, port)
		}
	}

	return nil
}

// detectConflicts detects potential conflicts in the template application
func (d *TemplateDiffCalculator) detectConflicts(currentState *InstanceState, template *Template, diff *TemplateDiff) {
	// Check for package manager conflicts
	if template.PackageManager != "" && currentState.PackageManager != "" &&
		template.PackageManager != currentState.PackageManager {
		diff.ConflictsFound = append(diff.ConflictsFound, ConflictDiff{
			Type: "package_manager",
			Description: fmt.Sprintf("Template uses %s but instance has %s",
				template.PackageManager, currentState.PackageManager),
			Resolution: "force",
		})
	}

	// Check for port conflicts
	currentPortSet := make(map[int]bool)
	for _, port := range currentState.Ports {
		currentPortSet[port] = true
	}

	for _, svc := range template.Services {
		if svc.Port > 0 && currentPortSet[svc.Port] {
			// Check if it's the same service
			currentService := currentState.GetService(svc.Name)
			if currentService == nil || currentService.Port != svc.Port {
				diff.ConflictsFound = append(diff.ConflictsFound, ConflictDiff{
					Type:        "port",
					Description: fmt.Sprintf("Port %d is already in use", svc.Port),
					Resolution:  "skip",
				})
			}
		}
	}

	// Check for user conflicts (users with different configurations)
	for _, user := range template.Users {
		currentUser := currentState.GetUser(user.Name)
		if currentUser != nil && !d.groupsEqual(currentUser.Groups, user.Groups) {
			diff.ConflictsFound = append(diff.ConflictsFound, ConflictDiff{
				Type:        "user",
				Description: fmt.Sprintf("User %s exists with different group membership", user.Name),
				Resolution:  "merge",
			})
		}
	}
}

// extractVersion extracts version from package specification like "python=3.11"
func (d *TemplateDiffCalculator) extractVersion(pkgSpec string) string {
	if strings.Contains(pkgSpec, "=") {
		parts := strings.Split(pkgSpec, "=")
		if len(parts) > 1 {
			return parts[1]
		}
	}
	return ""
}

// groupsEqual checks if two group slices are equal
func (d *TemplateDiffCalculator) groupsEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}

	aSet := make(map[string]bool)
	for _, group := range a {
		aSet[group] = true
	}

	for _, group := range b {
		if !aSet[group] {
			return false
		}
	}

	return true
}

// HasChanges returns true if the diff contains any changes to apply
func (diff *TemplateDiff) HasChanges() bool {
	return len(diff.PackagesToInstall) > 0 ||
		len(diff.PackagesToRemove) > 0 ||
		len(diff.ServicesToConfigure) > 0 ||
		len(diff.ServicesToStop) > 0 ||
		len(diff.UsersToCreate) > 0 ||
		len(diff.UsersToModify) > 0 ||
		len(diff.PortsToOpen) > 0
}

// Summary returns a human-readable summary of the diff
func (diff *TemplateDiff) Summary() string {
	var parts []string

	if len(diff.PackagesToInstall) > 0 {
		parts = append(parts, fmt.Sprintf("%d packages to install", len(diff.PackagesToInstall)))
	}

	if len(diff.ServicesToConfigure) > 0 {
		parts = append(parts, fmt.Sprintf("%d services to configure", len(diff.ServicesToConfigure)))
	}

	if len(diff.UsersToCreate) > 0 {
		parts = append(parts, fmt.Sprintf("%d users to create", len(diff.UsersToCreate)))
	}

	if len(diff.UsersToModify) > 0 {
		parts = append(parts, fmt.Sprintf("%d users to modify", len(diff.UsersToModify)))
	}

	if len(diff.PortsToOpen) > 0 {
		parts = append(parts, fmt.Sprintf("%d ports to open", len(diff.PortsToOpen)))
	}

	if len(diff.ConflictsFound) > 0 {
		parts = append(parts, fmt.Sprintf("%d conflicts detected", len(diff.ConflictsFound)))
	}

	if len(parts) == 0 {
		return "No changes needed"
	}

	return strings.Join(parts, ", ")
}
