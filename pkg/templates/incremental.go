// Package templates provides incremental template application capabilities.
package templates

import (
	"context"
	"fmt"
	"strings"
)

// IncrementalApplyEngine applies template changes incrementally to running instances
type IncrementalApplyEngine struct {
	executor RemoteExecutor
}

// NewIncrementalApplyEngine creates a new incremental apply engine
func NewIncrementalApplyEngine(executor RemoteExecutor) *IncrementalApplyEngine {
	return &IncrementalApplyEngine{
		executor: executor,
	}
}

// ApplyChanges applies the calculated template differences to a running instance
func (e *IncrementalApplyEngine) ApplyChanges(ctx context.Context, instanceName string, diff *TemplateDiff, template *Template) (*ApplyResult, error) {
	result := &ApplyResult{
		Warnings: []string{},
	}
	
	// 1. Install packages
	if len(diff.PackagesToInstall) > 0 {
		installed, err := e.installPackages(ctx, instanceName, diff.PackagesToInstall)
		if err != nil {
			return nil, fmt.Errorf("failed to install packages: %w", err)
		}
		result.PackagesInstalled = installed
	}
	
	// 2. Configure services
	if len(diff.ServicesToConfigure) > 0 {
		configured, err := e.configureServices(ctx, instanceName, diff.ServicesToConfigure)
		if err != nil {
			return nil, fmt.Errorf("failed to configure services: %w", err)
		}
		result.ServicesConfigured = configured
	}
	
	// 3. Create/modify users
	if len(diff.UsersToCreate) > 0 || len(diff.UsersToModify) > 0 {
		created, err := e.manageUsers(ctx, instanceName, diff.UsersToCreate, diff.UsersToModify)
		if err != nil {
			return nil, fmt.Errorf("failed to manage users: %w", err)
		}
		result.UsersCreated = created
	}
	
	// 4. Open ports (handled by security group updates)
	if len(diff.PortsToOpen) > 0 {
		if err := e.openPorts(ctx, instanceName, diff.PortsToOpen); err != nil {
			// Non-fatal - add warning
			result.Warnings = append(result.Warnings, 
				fmt.Sprintf("Failed to open ports %v: %v", diff.PortsToOpen, err))
		}
	}
	
	return result, nil
}

// installPackages installs the required packages using the appropriate package manager
func (e *IncrementalApplyEngine) installPackages(ctx context.Context, instanceName string, packages []PackageDiff) (int, error) {
	if len(packages) == 0 {
		return 0, nil
	}
	
	// Group packages by package manager
	packagesByManager := make(map[string][]PackageDiff)
	for _, pkg := range packages {
		packagesByManager[pkg.PackageManager] = append(packagesByManager[pkg.PackageManager], pkg)
	}
	
	totalInstalled := 0
	
	// Install packages for each package manager
	for manager, pkgs := range packagesByManager {
		installed, err := e.installPackagesWithManager(ctx, instanceName, manager, pkgs)
		if err != nil {
			return totalInstalled, fmt.Errorf("failed to install packages with %s: %w", manager, err)
		}
		totalInstalled += installed
	}
	
	return totalInstalled, nil
}

// installPackagesWithManager installs packages using a specific package manager
func (e *IncrementalApplyEngine) installPackagesWithManager(ctx context.Context, instanceName string, manager string, packages []PackageDiff) (int, error) {
	if len(packages) == 0 {
		return 0, nil
	}
	
	// Generate installation script based on package manager
	script, err := e.generateInstallationScript(manager, packages)
	if err != nil {
		return 0, fmt.Errorf("failed to generate installation script: %w", err)
	}
	
	// Execute installation script
	result, err := e.executor.ExecuteScript(ctx, instanceName, script)
	if err != nil {
		return 0, fmt.Errorf("failed to execute installation script: %w", err)
	}
	
	if result.ExitCode != 0 {
		return 0, fmt.Errorf("installation script failed (exit code %d): %s", result.ExitCode, result.Stderr)
	}
	
	return len(packages), nil
}

// generateInstallationScript generates package installation script for a specific package manager
func (e *IncrementalApplyEngine) generateInstallationScript(manager string, packages []PackageDiff) (string, error) {
	var script strings.Builder
	
	script.WriteString("#!/bin/bash\n")
	script.WriteString("set -e\n\n")
	script.WriteString("# Package installation script\n")
	script.WriteString(fmt.Sprintf("# Package manager: %s\n", manager))
	script.WriteString(fmt.Sprintf("# Packages to install: %d\n\n", len(packages)))
	
	switch manager {
	case "apt":
		script.WriteString("# Update package index\n")
		script.WriteString("apt-get update -y\n\n")
		
		script.WriteString("# Install packages\n")
		for _, pkg := range packages {
			if pkg.TargetVersion != "" {
				script.WriteString(fmt.Sprintf("apt-get install -y %s=%s\n", pkg.Name, pkg.TargetVersion))
			} else {
				script.WriteString(fmt.Sprintf("apt-get install -y %s\n", pkg.Name))
			}
		}
		
	case "dnf":
		script.WriteString("# Install packages\n")
		for _, pkg := range packages {
			if pkg.TargetVersion != "" {
				script.WriteString(fmt.Sprintf("dnf install -y %s-%s\n", pkg.Name, pkg.TargetVersion))
			} else {
				script.WriteString(fmt.Sprintf("dnf install -y %s\n", pkg.Name))
			}
		}
		
	case "conda":
		script.WriteString("# Install conda packages\n")
		for _, pkg := range packages {
			if pkg.TargetVersion != "" {
				script.WriteString(fmt.Sprintf("conda install -y %s=%s\n", pkg.Name, pkg.TargetVersion))
			} else {
				script.WriteString(fmt.Sprintf("conda install -y %s\n", pkg.Name))
			}
		}
		
	case "pip":
		script.WriteString("# Install pip packages\n")
		for _, pkg := range packages {
			if pkg.TargetVersion != "" {
				script.WriteString(fmt.Sprintf("pip install %s==%s\n", pkg.Name, pkg.TargetVersion))
			} else {
				script.WriteString(fmt.Sprintf("pip install %s\n", pkg.Name))
			}
		}
		
	case "spack":
		script.WriteString("# Install spack packages\n")
		for _, pkg := range packages {
			if pkg.TargetVersion != "" {
				script.WriteString(fmt.Sprintf("spack install %s@%s\n", pkg.Name, pkg.TargetVersion))
			} else {
				script.WriteString(fmt.Sprintf("spack install %s\n", pkg.Name))
			}
		}
		
	default:
		return "", fmt.Errorf("unsupported package manager: %s", manager)
	}
	
	script.WriteString("\necho 'Package installation completed successfully'\n")
	
	return script.String(), nil
}

// configureServices configures and starts the required services
func (e *IncrementalApplyEngine) configureServices(ctx context.Context, instanceName string, services []ServiceDiff) (int, error) {
	if len(services) == 0 {
		return 0, nil
	}
	
	// Generate service configuration script
	script := e.generateServiceScript(services)
	
	// Execute service configuration
	result, err := e.executor.ExecuteScript(ctx, instanceName, script)
	if err != nil {
		return 0, fmt.Errorf("failed to execute service script: %w", err)
	}
	
	if result.ExitCode != 0 {
		return 0, fmt.Errorf("service configuration failed (exit code %d): %s", result.ExitCode, result.Stderr)
	}
	
	return len(services), nil
}

// generateServiceScript generates service configuration script
func (e *IncrementalApplyEngine) generateServiceScript(services []ServiceDiff) string {
	var script strings.Builder
	
	script.WriteString("#!/bin/bash\n")
	script.WriteString("set -e\n\n")
	script.WriteString("# Service configuration script\n")
	script.WriteString(fmt.Sprintf("# Services to configure: %d\n\n", len(services)))
	
	for _, svc := range services {
		switch svc.Action {
		case "configure":
			script.WriteString(fmt.Sprintf("# Configure service: %s\n", svc.Name))
			script.WriteString(fmt.Sprintf("systemctl enable %s\n", svc.Name))
			script.WriteString(fmt.Sprintf("systemctl start %s\n", svc.Name))
			
		case "start":
			script.WriteString(fmt.Sprintf("# Start service: %s\n", svc.Name))
			script.WriteString(fmt.Sprintf("systemctl start %s\n", svc.Name))
			
		case "restart":
			script.WriteString(fmt.Sprintf("# Restart service: %s\n", svc.Name))
			script.WriteString(fmt.Sprintf("systemctl restart %s\n", svc.Name))
			
		case "stop":
			script.WriteString(fmt.Sprintf("# Stop service: %s\n", svc.Name))
			script.WriteString(fmt.Sprintf("systemctl stop %s\n", svc.Name))
		}
		
		script.WriteString("\n")
	}
	
	script.WriteString("echo 'Service configuration completed successfully'\n")
	
	return script.String()
}

// manageUsers creates and modifies user accounts
func (e *IncrementalApplyEngine) manageUsers(ctx context.Context, instanceName string, usersToCreate []UserDiff, usersToModify []UserDiff) (int, error) {
	totalUsers := len(usersToCreate) + len(usersToModify)
	if totalUsers == 0 {
		return 0, nil
	}
	
	// Generate user management script
	script := e.generateUserScript(usersToCreate, usersToModify)
	
	// Execute user management
	result, err := e.executor.ExecuteScript(ctx, instanceName, script)
	if err != nil {
		return 0, fmt.Errorf("failed to execute user script: %w", err)
	}
	
	if result.ExitCode != 0 {
		return 0, fmt.Errorf("user management failed (exit code %d): %s", result.ExitCode, result.Stderr)
	}
	
	return len(usersToCreate), nil // Only count new users created
}

// generateUserScript generates user management script
func (e *IncrementalApplyEngine) generateUserScript(usersToCreate []UserDiff, usersToModify []UserDiff) string {
	var script strings.Builder
	
	script.WriteString("#!/bin/bash\n")
	script.WriteString("set -e\n\n")
	script.WriteString("# User management script\n")
	script.WriteString(fmt.Sprintf("# Users to create: %d\n", len(usersToCreate)))
	script.WriteString(fmt.Sprintf("# Users to modify: %d\n\n", len(usersToModify)))
	
	// Create users
	for _, user := range usersToCreate {
		script.WriteString(fmt.Sprintf("# Create user: %s\n", user.Name))
		script.WriteString(fmt.Sprintf("useradd -m -s /bin/bash %s\n", user.Name))
		
		// Add to groups
		for _, group := range user.TargetGroups {
			script.WriteString(fmt.Sprintf("usermod -a -G %s %s\n", group, user.Name))
		}
		
		script.WriteString("\n")
	}
	
	// Modify users
	for _, user := range usersToModify {
		script.WriteString(fmt.Sprintf("# Modify user: %s\n", user.Name))
		
		// Update group membership
		if len(user.TargetGroups) > 0 {
			groupList := strings.Join(user.TargetGroups, ",")
			script.WriteString(fmt.Sprintf("usermod -G %s %s\n", groupList, user.Name))
		}
		
		script.WriteString("\n")
	}
	
	script.WriteString("echo 'User management completed successfully'\n")
	
	return script.String()
}

// openPorts handles port opening (placeholder for security group integration)
func (e *IncrementalApplyEngine) openPorts(ctx context.Context, instanceName string, ports []int) error {
	// This would integrate with AWS security group management
	// For now, it's a placeholder that logs the ports that need to be opened
	
	script := fmt.Sprintf(`#!/bin/bash
# Port opening placeholder
echo "Ports to open: %v"
echo "Note: Port opening requires security group updates (not implemented in this prototype)"
`, ports)
	
	result, err := e.executor.ExecuteScript(ctx, instanceName, script)
	if err != nil {
		return fmt.Errorf("failed to execute port script: %w", err)
	}
	
	if result.ExitCode != 0 {
		return fmt.Errorf("port script failed (exit code %d): %s", result.ExitCode, result.Stderr)
	}
	
	return nil
}