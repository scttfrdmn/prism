// Package templates provides plugin registry for discovery and management.
package templates

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

// PluginRegistry manages plugin manifests and provides discovery
type PluginRegistry struct {
	plugins     map[string]*PluginManifest
	pluginPaths []string
}

// NewPluginRegistry creates a new plugin registry
func NewPluginRegistry() *PluginRegistry {
	return &PluginRegistry{
		plugins:     make(map[string]*PluginManifest),
		pluginPaths: make([]string, 0),
	}
}

// DefaultPluginDirs returns the default plugin directories to search
func DefaultPluginDirs() []string {
	dirs := []string{}

	// Environment variable override (for tests and custom setups)
	if envDir := os.Getenv("PRISM_PLUGIN_DIR"); envDir != "" {
		envDir = filepath.Clean(envDir)
		if info, err := os.Stat(envDir); err == nil && info.IsDir() { //nolint:gosec // G703: PRISM_PLUGIN_DIR is the local user's own config, only stat'd
			dirs = append(dirs, envDir)
			return dirs // Use ONLY the environment variable path when set
		}
	}

	// Binary-relative path (development and production)
	if exe, err := os.Executable(); err == nil {
		exeDir := filepath.Dir(exe)
		binaryRelativePath := filepath.Join(exeDir, "..", "plugins")
		if absPath, err := filepath.Abs(binaryRelativePath); err == nil {
			if info, err := os.Stat(absPath); err == nil && info.IsDir() {
				dirs = append(dirs, absPath)
			}
		}
	}

	// System-wide installation path (Homebrew, system packages)
	systemPaths := []string{
		"/opt/homebrew/share/prism/plugins",
		"/usr/local/share/prism/plugins",
		"/usr/share/prism/plugins",
	}
	for _, path := range systemPaths {
		if info, err := os.Stat(path); err == nil && info.IsDir() {
			dirs = append(dirs, path)
		}
	}

	return dirs
}

// LoadPluginsFromDirectory loads all plugin manifests from a directory
func (r *PluginRegistry) LoadPluginsFromDirectory(dir string) error {
	files, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("failed to read plugin directory %s: %w", dir, err)
	}

	loadedCount := 0
	for _, file := range files {
		if file.IsDir() {
			continue
		}

		ext := filepath.Ext(file.Name())
		if ext != ".yml" && ext != ".yaml" {
			continue
		}

		pluginPath := filepath.Join(dir, file.Name())
		if err := r.LoadPlugin(pluginPath); err != nil {
			// Log warning but continue loading other plugins
			fmt.Printf("Warning: failed to load plugin %s: %v\n", file.Name(), err)
			continue
		}
		loadedCount++
	}

	if loadedCount == 0 {
		return fmt.Errorf("no valid plugins found in directory %s", dir)
	}

	return nil
}

// LoadPlugin loads a single plugin manifest from a file
func (r *PluginRegistry) LoadPlugin(path string) error {
	data, err := os.ReadFile(path) //nolint:gosec // G304: plugin manifest discovered in local plugin dirs (user/system config)
	if err != nil {
		return fmt.Errorf("failed to read plugin file: %w", err)
	}

	var manifest PluginManifest
	if err := yaml.Unmarshal(data, &manifest); err != nil {
		return fmt.Errorf("failed to parse plugin manifest: %w", err)
	}

	// Validate manifest
	if err := r.validateManifest(&manifest); err != nil {
		return fmt.Errorf("invalid plugin manifest: %w", err)
	}

	// Store plugin
	r.plugins[manifest.Metadata.Name] = &manifest
	r.pluginPaths = append(r.pluginPaths, path)

	return nil
}

// GetPlugin retrieves a plugin by name
func (r *PluginRegistry) GetPlugin(name string) (*PluginManifest, error) {
	plugin, exists := r.plugins[name]
	if !exists {
		return nil, fmt.Errorf("plugin %s not found", name)
	}
	return plugin, nil
}

// ListPlugins returns all registered plugins
func (r *PluginRegistry) ListPlugins() []*PluginManifest {
	plugins := make([]*PluginManifest, 0, len(r.plugins))
	for _, plugin := range r.plugins {
		plugins = append(plugins, plugin)
	}
	return plugins
}

// GetPluginsByCategory returns plugins in a specific category
func (r *PluginRegistry) GetPluginsByCategory(category string) []*PluginManifest {
	plugins := make([]*PluginManifest, 0)
	for _, plugin := range r.plugins {
		if plugin.Metadata.Category == category {
			plugins = append(plugins, plugin)
		}
	}
	return plugins
}

// HasPlugin checks if a plugin exists
func (r *PluginRegistry) HasPlugin(name string) bool {
	_, exists := r.plugins[name]
	return exists
}

// validateManifest validates a plugin manifest structure
func (r *PluginRegistry) validateManifest(manifest *PluginManifest) error {
	if manifest.Metadata.Name == "" {
		return fmt.Errorf("plugin name is required")
	}
	if manifest.Metadata.Version == "" {
		return fmt.Errorf("plugin version is required")
	}
	if manifest.APIVersion == "" {
		return fmt.Errorf("apiVersion is required")
	}
	if manifest.Kind == "" {
		return fmt.Errorf("kind is required")
	}
	if len(manifest.Spec.Installers) == 0 {
		return fmt.Errorf("at least one installer is required")
	}

	// Validate installer specs
	for toolName, installer := range manifest.Spec.Installers {
		if len(installer.Packages) == 0 {
			return fmt.Errorf("installer %s must have at least one package", toolName)
		}
		for _, pkg := range installer.Packages {
			if pkg.Name == "" {
				return fmt.Errorf("package name is required in installer %s", toolName)
			}
			if pkg.InstallCommand == "" {
				return fmt.Errorf("install_command is required for package %s in installer %s", pkg.Name, toolName)
			}
		}
	}

	return nil
}

// ResolveDependencies resolves all dependencies for a plugin (recursive)
func (r *PluginRegistry) ResolveDependencies(plugin *PluginManifest) ([]*PluginManifest, error) {
	resolved := make([]*PluginManifest, 0)
	visited := make(map[string]bool)

	if err := r.resolveDependenciesRecursive(plugin, &resolved, visited); err != nil {
		return nil, err
	}

	return resolved, nil
}

// resolveDependenciesRecursive resolves dependencies recursively with cycle detection
func (r *PluginRegistry) resolveDependenciesRecursive(plugin *PluginManifest, resolved *[]*PluginManifest, visited map[string]bool) error {
	// Check for cycles
	if visited[plugin.Metadata.Name] {
		return nil // Already processed
	}

	visited[plugin.Metadata.Name] = true

	// Resolve required dependencies first
	for _, dep := range plugin.Spec.Dependencies.Required {
		depPlugin, err := r.GetPlugin(dep.Plugin)
		if err != nil {
			return fmt.Errorf("required dependency %s not found: %w", dep.Plugin, err)
		}

		// Check version constraint
		if !r.versionSatisfies(depPlugin.Metadata.Version, dep.Version) {
			return fmt.Errorf("dependency %s version %s does not satisfy constraint %s",
				dep.Plugin, depPlugin.Metadata.Version, dep.Version)
		}

		// Recursively resolve dependencies
		if err := r.resolveDependenciesRecursive(depPlugin, resolved, visited); err != nil {
			return err
		}
	}

	// Add this plugin to resolved list
	*resolved = append(*resolved, plugin)

	return nil
}

// CheckConflicts checks for conflicting plugins in a list
func (r *PluginRegistry) CheckConflicts(plugins []*PluginManifest) error {
	installed := make(map[string]*PluginManifest)

	for _, plugin := range plugins {
		// Check if this plugin conflicts with any already installed
		for _, conflict := range plugin.Spec.Conflicts {
			if conflictingPlugin, exists := installed[conflict.Plugin]; exists {
				return fmt.Errorf("plugin %s conflicts with %s: %s",
					plugin.Metadata.Name, conflictingPlugin.Metadata.Name, conflict.Reason)
			}
		}

		// Check if any already installed plugin conflicts with this one
		for _, installedPlugin := range installed {
			for _, conflict := range installedPlugin.Spec.Conflicts {
				if conflict.Plugin == plugin.Metadata.Name {
					return fmt.Errorf("plugin %s conflicts with %s: %s",
						installedPlugin.Metadata.Name, plugin.Metadata.Name, conflict.Reason)
				}
			}
		}

		installed[plugin.Metadata.Name] = plugin
	}

	return nil
}

// parseVersionFromOutput extracts the first X.Y.Z or X.Y version string from command output.
// Used by plugin installers to parse tool versions from stdout.
var versionPattern = regexp.MustCompile(`\d+\.\d+(?:\.\d+)?`)

func parseVersionFromOutput(output string) string {
	return versionPattern.FindString(output)
}

// compareVersionStrings compares two version strings numerically by component.
// Returns: -1 if v1 < v2, 0 if v1 == v2, 1 if v1 > v2
func compareVersionStrings(v1, v2 string) int {
	parts1 := strings.Split(strings.TrimSpace(v1), ".")
	parts2 := strings.Split(strings.TrimSpace(v2), ".")

	maxLen := len(parts1)
	if len(parts2) > maxLen {
		maxLen = len(parts2)
	}

	for i := 0; i < maxLen; i++ {
		var n1, n2 int
		if i < len(parts1) {
			if _, err := fmt.Sscanf(parts1[i], "%d", &n1); err != nil {
				n1 = 0 // non-numeric component compares as 0
			}
		}
		if i < len(parts2) {
			if _, err := fmt.Sscanf(parts2[i], "%d", &n2); err != nil {
				n2 = 0 // non-numeric component compares as 0
			}
		}
		if n1 < n2 {
			return -1
		}
		if n1 > n2 {
			return 1
		}
	}
	return 0
}

// versionSatisfiesConstraint checks if a version satisfies a constraint string.
// Supports: exact match, >=, >, <=, <, ~X.Y.Z (patch range), ^X.Y.Z (minor range).
func versionSatisfiesConstraint(version, constraint string) bool {
	if constraint == "" || constraint == "latest" {
		return true
	}

	if !strings.ContainsAny(constraint, "><=~^") {
		return version == constraint
	}

	if strings.HasPrefix(constraint, ">=") {
		return compareVersionStrings(version, strings.TrimPrefix(constraint, ">=")) >= 0
	}
	if strings.HasPrefix(constraint, ">") {
		return compareVersionStrings(version, strings.TrimPrefix(constraint, ">")) > 0
	}
	if strings.HasPrefix(constraint, "<=") {
		return compareVersionStrings(version, strings.TrimPrefix(constraint, "<=")) <= 0
	}
	if strings.HasPrefix(constraint, "<") {
		return compareVersionStrings(version, strings.TrimPrefix(constraint, "<")) < 0
	}

	// ~X.Y.Z — patch-level range: >=X.Y.Z <X.Y+1.0
	if strings.HasPrefix(constraint, "~") {
		base := strings.TrimPrefix(constraint, "~")
		parts := strings.SplitN(base, ".", 3)
		if len(parts) >= 2 {
			minor, _ := strconv.Atoi(parts[1])
			upper := fmt.Sprintf("%s.%d.0", parts[0], minor+1)
			return compareVersionStrings(version, base) >= 0 &&
				compareVersionStrings(version, upper) < 0
		}
		return compareVersionStrings(version, base) >= 0
	}

	// ^X.Y.Z — minor-level range: >=X.Y.Z <X+1.0.0
	if strings.HasPrefix(constraint, "^") {
		base := strings.TrimPrefix(constraint, "^")
		parts := strings.SplitN(base, ".", 3)
		major, _ := strconv.Atoi(parts[0])
		upper := fmt.Sprintf("%d.0.0", major+1)
		return compareVersionStrings(version, base) >= 0 &&
			compareVersionStrings(version, upper) < 0
	}

	return true // permissive default
}

// versionSatisfies checks if a version satisfies a constraint (method wrapper).
func (r *PluginRegistry) versionSatisfies(version, constraint string) bool {
	return versionSatisfiesConstraint(version, constraint)
}

// compareVersions compares two version strings (method wrapper).
func (r *PluginRegistry) compareVersions(v1, v2 string) int {
	return compareVersionStrings(v1, v2)
}

// Clear removes all plugins from the registry (useful for testing)
func (r *PluginRegistry) Clear() {
	r.plugins = make(map[string]*PluginManifest)
	r.pluginPaths = make([]string, 0)
}
