// Package templates provides instance state inspection capabilities.
package templates

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// InstanceStateInspector inspects the current state of running instances
type InstanceStateInspector struct {
	executor RemoteExecutor
}

// NewInstanceStateInspector creates a new instance state inspector
func NewInstanceStateInspector(executor RemoteExecutor) *InstanceStateInspector {
	return &InstanceStateInspector{
		executor: executor,
	}
}

// InspectInstance inspects the current state of a running instance
func (i *InstanceStateInspector) InspectInstance(ctx context.Context, instanceName string) (*InstanceState, error) {
	state := &InstanceState{
		LastInspected: time.Now(),
	}
	
	// Inspect installed packages
	packages, err := i.inspectPackages(ctx, instanceName)
	if err != nil {
		return nil, fmt.Errorf("failed to inspect packages: %w", err)
	}
	state.Packages = packages
	
	// Detect package manager
	packageManager, err := i.detectPackageManager(ctx, instanceName)
	if err != nil {
		return nil, fmt.Errorf("failed to detect package manager: %w", err)
	}
	state.PackageManager = packageManager
	
	// Inspect running services
	services, err := i.inspectServices(ctx, instanceName)
	if err != nil {
		return nil, fmt.Errorf("failed to inspect services: %w", err)
	}
	state.Services = services
	
	// Inspect users
	users, err := i.inspectUsers(ctx, instanceName)
	if err != nil {
		return nil, fmt.Errorf("failed to inspect users: %w", err)
	}
	state.Users = users
	
	// Inspect open ports
	ports, err := i.inspectPorts(ctx, instanceName)
	if err != nil {
		return nil, fmt.Errorf("failed to inspect ports: %w", err)
	}
	state.Ports = ports
	
	// Load applied templates history
	appliedTemplates, err := i.loadAppliedTemplates(ctx, instanceName)
	if err != nil {
		// Non-fatal - instance might not have template history yet
		appliedTemplates = []AppliedTemplate{}
	}
	state.AppliedTemplates = appliedTemplates
	
	return state, nil
}

// inspectPackages inspects installed packages on the instance
func (i *InstanceStateInspector) inspectPackages(ctx context.Context, instanceName string) ([]InstalledPackage, error) {
	var packages []InstalledPackage
	
	// Try different package managers
	packageManagers := []struct {
		name    string
		command string
		parser  func(string) ([]InstalledPackage, error)
	}{
		{"apt", "dpkg -l", i.parseAptPackages},
		{"dnf", "dnf list installed", i.parseDnfPackages},
		{"conda", "conda list --json", i.parseCondaPackages},
		{"pip", "pip list --format=json", i.parsePipPackages},
	}
	
	for _, pm := range packageManagers {
		result, err := i.executor.Execute(ctx, instanceName, pm.command)
		if err != nil || result.ExitCode != 0 {
			// Package manager not available, skip
			continue
		}
		
		pmPackages, err := pm.parser(result.Stdout)
		if err != nil {
			// Failed to parse, skip
			continue
		}
		
		packages = append(packages, pmPackages...)
	}
	
	return packages, nil
}

// parseAptPackages parses dpkg -l output
func (i *InstanceStateInspector) parseAptPackages(output string) ([]InstalledPackage, error) {
	var packages []InstalledPackage
	lines := strings.Split(output, "\n")
	
	for _, line := range lines {
		if !strings.HasPrefix(line, "ii ") {
			continue
		}
		
		fields := strings.Fields(line)
		if len(fields) < 3 {
			continue
		}
		
		packages = append(packages, InstalledPackage{
			Name:           fields[1],
			Version:        fields[2],
			PackageManager: "apt",
			Source:         "unknown", // Would need additional logic to determine
		})
	}
	
	return packages, nil
}

// parseDnfPackages parses dnf list installed output
func (i *InstanceStateInspector) parseDnfPackages(output string) ([]InstalledPackage, error) {
	var packages []InstalledPackage
	lines := strings.Split(output, "\n")
	
	// Skip header lines
	inPackageList := false
	for _, line := range lines {
		if strings.Contains(line, "Installed Packages") {
			inPackageList = true
			continue
		}
		
		if !inPackageList || strings.TrimSpace(line) == "" {
			continue
		}
		
		// Parse package.arch version repo
		parts := strings.Fields(line)
		if len(parts) < 2 {
			continue
		}
		
		nameParts := strings.Split(parts[0], ".")
		name := nameParts[0]
		
		packages = append(packages, InstalledPackage{
			Name:           name,
			Version:        parts[1],
			PackageManager: "dnf",
			Source:         "unknown",
		})
	}
	
	return packages, nil
}

// parseCondaPackages parses conda list --json output
func (i *InstanceStateInspector) parseCondaPackages(output string) ([]InstalledPackage, error) {
	var condaPackages []struct {
		Name    string `json:"name"`
		Version string `json:"version"`
		Channel string `json:"channel"`
	}
	
	if err := json.Unmarshal([]byte(output), &condaPackages); err != nil {
		return nil, err
	}
	
	var packages []InstalledPackage
	for _, pkg := range condaPackages {
		packages = append(packages, InstalledPackage{
			Name:           pkg.Name,
			Version:        pkg.Version,
			PackageManager: "conda",
			Source:         "unknown",
		})
	}
	
	return packages, nil
}

// parsePipPackages parses pip list --format=json output
func (i *InstanceStateInspector) parsePipPackages(output string) ([]InstalledPackage, error) {
	var pipPackages []struct {
		Name    string `json:"name"`
		Version string `json:"version"`
	}
	
	if err := json.Unmarshal([]byte(output), &pipPackages); err != nil {
		return nil, err
	}
	
	var packages []InstalledPackage
	for _, pkg := range pipPackages {
		packages = append(packages, InstalledPackage{
			Name:           pkg.Name,
			Version:        pkg.Version,
			PackageManager: "pip",
			Source:         "unknown",
		})
	}
	
	return packages, nil
}

// detectPackageManager detects the primary package manager on the instance
func (i *InstanceStateInspector) detectPackageManager(ctx context.Context, instanceName string) (string, error) {
	// Check for package managers in order of preference
	managers := []struct {
		name    string
		command string
	}{
		{"conda", "which conda"},
		{"apt", "which apt-get"},
		{"dnf", "which dnf"},
		{"spack", "which spack"},
	}
	
	for _, mgr := range managers {
		result, err := i.executor.Execute(ctx, instanceName, mgr.command)
		if err == nil && result.ExitCode == 0 {
			return mgr.name, nil
		}
	}
	
	return "unknown", nil
}

// inspectServices inspects running services on the instance
func (i *InstanceStateInspector) inspectServices(ctx context.Context, instanceName string) ([]RunningService, error) {
	var services []RunningService
	
	// Use systemctl to list services
	result, err := i.executor.Execute(ctx, instanceName, "systemctl list-units --type=service --all --no-pager --plain")
	if err != nil {
		return services, nil // Non-fatal, system might not use systemd
	}
	
	lines := strings.Split(result.Stdout, "\n")
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}
		
		fields := strings.Fields(line)
		if len(fields) < 4 {
			continue
		}
		
		serviceName := strings.TrimSuffix(fields[0], ".service")
		status := "stopped"
		if fields[1] == "loaded" && fields[2] == "active" {
			status = "running"
		}
		
		services = append(services, RunningService{
			Name:    serviceName,
			Status:  status,
			Port:    0, // Would need additional logic to determine port
			Enabled: fields[1] == "loaded",
			Source:  "unknown",
		})
	}
	
	return services, nil
}

// inspectUsers inspects user accounts on the instance
func (i *InstanceStateInspector) inspectUsers(ctx context.Context, instanceName string) ([]ExistingUser, error) {
	var users []ExistingUser
	
	// Get user list from /etc/passwd
	result, err := i.executor.Execute(ctx, instanceName, "cat /etc/passwd")
	if err != nil {
		return nil, err
	}
	
	lines := strings.Split(result.Stdout, "\n")
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}
		
		fields := strings.Split(line, ":")
		if len(fields) < 7 {
			continue
		}
		
		username := fields[0]
		shell := fields[6]
		
		// Skip system users (UID < 1000)
		uidStr := fields[2]
		uid, err := strconv.Atoi(uidStr)
		if err != nil || uid < 1000 {
			continue
		}
		
		// Get user groups
		groupResult, err := i.executor.Execute(ctx, instanceName, fmt.Sprintf("groups %s", username))
		var groups []string
		if err == nil {
			// Parse "username : group1 group2 group3"
			parts := strings.Split(groupResult.Stdout, ":")
			if len(parts) > 1 {
				groupList := strings.TrimSpace(parts[1])
				groups = strings.Fields(groupList)
			}
		}
		
		users = append(users, ExistingUser{
			Name:   username,
			Groups: groups,
			Shell:  shell,
			Source: "unknown",
		})
	}
	
	return users, nil
}

// inspectPorts inspects open ports on the instance
func (i *InstanceStateInspector) inspectPorts(ctx context.Context, instanceName string) ([]int, error) {
	var ports []int
	
	// Use netstat to find listening ports
	result, err := i.executor.Execute(ctx, instanceName, "netstat -tlnp 2>/dev/null || ss -tlnp")
	if err != nil {
		return ports, nil // Non-fatal
	}
	
	// Parse netstat/ss output to extract ports
	lines := strings.Split(result.Stdout, "\n")
	portRegex := regexp.MustCompile(`:(\d+)\s`)
	
	portSet := make(map[int]bool)
	for _, line := range lines {
		matches := portRegex.FindAllStringSubmatch(line, -1)
		for _, match := range matches {
			if len(match) > 1 {
				port, err := strconv.Atoi(match[1])
				if err == nil && port > 0 && port < 65536 {
					portSet[port] = true
				}
			}
		}
	}
	
	// Convert set to slice
	for port := range portSet {
		ports = append(ports, port)
	}
	
	return ports, nil
}

// loadAppliedTemplates loads the history of applied templates
func (i *InstanceStateInspector) loadAppliedTemplates(ctx context.Context, instanceName string) ([]AppliedTemplate, error) {
	// Try to load template application history from a known location
	result, err := i.executor.Execute(ctx, instanceName, "cat /opt/cloudworkstation/applied-templates.json 2>/dev/null")
	if err != nil || result.ExitCode != 0 {
		// No history file, return empty list
		return []AppliedTemplate{}, nil
	}
	
	var templates []AppliedTemplate
	if err := json.Unmarshal([]byte(result.Stdout), &templates); err != nil {
		// Corrupted history file, return empty list
		return []AppliedTemplate{}, nil
	}
	
	return templates, nil
}

// HasPackage checks if a package is installed
func (state *InstanceState) HasPackage(packageName string) bool {
	for _, pkg := range state.Packages {
		if pkg.Name == packageName {
			return true
		}
	}
	return false
}

// HasService checks if a service exists
func (state *InstanceState) HasService(serviceName string) bool {
	for _, svc := range state.Services {
		if svc.Name == serviceName {
			return true
		}
	}
	return false
}

// HasUser checks if a user exists
func (state *InstanceState) HasUser(username string) bool {
	for _, user := range state.Users {
		if user.Name == username {
			return true
		}
	}
	return false
}

// GetPackage returns a package by name if it exists
func (state *InstanceState) GetPackage(packageName string) *InstalledPackage {
	for _, pkg := range state.Packages {
		if pkg.Name == packageName {
			return &pkg
		}
	}
	return nil
}

// GetService returns a service by name if it exists
func (state *InstanceState) GetService(serviceName string) *RunningService {
	for _, svc := range state.Services {
		if svc.Name == serviceName {
			return &svc
		}
	}
	return nil
}

// GetUser returns a user by name if it exists
func (state *InstanceState) GetUser(username string) *ExistingUser {
	for _, user := range state.Users {
		if user.Name == username {
			return &user
		}
	}
	return nil
}