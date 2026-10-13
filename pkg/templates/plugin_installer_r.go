// Package templates provides R package plugin installer implementation.
package templates

import (
	"context"
	"fmt"
	"strings"
	"text/template"
)

// RPackageInstaller installs R packages on remote instances
type RPackageInstaller struct {
	executor     RemoteExecutor
	instanceName string
}

// NewRPackageInstaller creates a new R package installer
func NewRPackageInstaller(executor RemoteExecutor, instanceName string) *RPackageInstaller {
	return &RPackageInstaller{
		executor:     executor,
		instanceName: instanceName,
	}
}

// Name returns the installer name
func (r *RPackageInstaller) Name() string {
	return "r"
}

// Install installs an R package with specified version and configuration
func (r *RPackageInstaller) Install(ctx context.Context, plugin *PluginManifest, version string, params map[string]interface{}) error {
	// Get R-specific installer from plugin manifest
	rInstaller, exists := plugin.Spec.Installers["r"]
	if !exists {
		return fmt.Errorf("plugin %s does not support R installer", plugin.Metadata.Name)
	}

	// Merge plugin configuration parameters with provided parameters
	mergedParams := r.mergeParameters(plugin.Spec.Configuration.Parameters, params)
	mergedParams["version"] = version

	// Install each package
	for _, pkg := range rInstaller.Packages {
		// Substitute parameters in install command
		installCmd, err := r.substituteParameters(pkg.InstallCommand, mergedParams)
		if err != nil {
			return fmt.Errorf("failed to substitute parameters for package %s: %w", pkg.Name, err)
		}

		// Execute installation command
		result, err := r.executor.Execute(ctx, r.instanceName, installCmd)
		if err != nil {
			return fmt.Errorf("failed to execute install command for package %s: %w", pkg.Name, err)
		}

		if result.ExitCode != 0 {
			return fmt.Errorf("install command failed for package %s (exit code %d): %s",
				pkg.Name, result.ExitCode, result.Stderr)
		}
	}

	// Create configuration files if specified
	for _, config := range rInstaller.Config {
		// Substitute parameters in config content
		content, err := r.substituteParameters(config.Content, mergedParams)
		if err != nil {
			return fmt.Errorf("failed to substitute parameters in config %s: %w", config.File, err)
		}

		// Create directory and write config file
		writeCmd := fmt.Sprintf("mkdir -p $(dirname %s) && cat > %s <<'EOF'\n%s\nEOF",
			config.Path, config.Path, content)

		result, err := r.executor.Execute(ctx, r.instanceName, writeCmd)
		if err != nil {
			return fmt.Errorf("failed to create config file %s: %w", config.File, err)
		}

		if result.ExitCode != 0 {
			return fmt.Errorf("failed to write config %s (exit code %d): %s",
				config.File, result.ExitCode, result.Stderr)
		}
	}

	// Execute post-install script if specified
	if rInstaller.PostInstall != "" {
		postInstall, err := r.substituteParameters(rInstaller.PostInstall, mergedParams)
		if err != nil {
			return fmt.Errorf("failed to substitute parameters in post-install script: %w", err)
		}

		result, err := r.executor.ExecuteScript(ctx, r.instanceName, postInstall)
		if err != nil {
			return fmt.Errorf("failed to execute post-install script: %w", err)
		}

		if result.ExitCode != 0 {
			return fmt.Errorf("post-install script failed (exit code %d): %s",
				result.ExitCode, result.Stderr)
		}
	}

	return nil
}

// Uninstall removes an R package
func (r *RPackageInstaller) Uninstall(ctx context.Context, plugin *PluginManifest) error {
	rInstaller, exists := plugin.Spec.Installers["r"]
	if !exists {
		return fmt.Errorf("plugin %s does not support R installer", plugin.Metadata.Name)
	}

	for _, pkg := range rInstaller.Packages {
		uninstallCmd := fmt.Sprintf("R -e \"remove.packages('%s')\"", pkg.Name)

		result, err := r.executor.Execute(ctx, r.instanceName, uninstallCmd)
		if err != nil {
			return fmt.Errorf("failed to uninstall package %s: %w", pkg.Name, err)
		}

		if result.ExitCode != 0 {
			// Log warning but continue - package might not be installed
			fmt.Printf("Warning: failed to uninstall package %s: %s\n", pkg.Name, result.Stderr)
		}
	}

	return nil
}

// Validate checks if R is available and prerequisites are met
func (r *RPackageInstaller) Validate(ctx context.Context, plugin *PluginManifest) error {
	// Check if R is installed
	result, err := r.executor.Execute(ctx, r.instanceName, "R --version")
	if err != nil {
		return fmt.Errorf("no R installation found: %w", err)
	}

	if result.ExitCode != 0 {
		return fmt.Errorf("no working R installation (exit code %d): %s", result.ExitCode, result.Stderr)
	}

	// Check compatibility constraints if specified
	if plugin.Spec.Compatibility.Tools != nil {
		if constraint, exists := plugin.Spec.Compatibility.Tools["r"]; exists {
			if installed := parseVersionFromOutput(result.Stdout); installed != "" {
				if constraint.MinVersion != "" && !versionSatisfiesConstraint(installed, ">="+constraint.MinVersion) {
					return fmt.Errorf("installed R version %s is below minimum required %s",
						installed, constraint.MinVersion)
				}
				if constraint.MaxVersion != "" && !versionSatisfiesConstraint(installed, "<="+constraint.MaxVersion) {
					return fmt.Errorf("installed R version %s exceeds maximum allowed %s",
						installed, constraint.MaxVersion)
				}
			}
		}
	}

	return nil
}

// GetInstalledVersion returns the installed version of a plugin
func (r *RPackageInstaller) GetInstalledVersion(ctx context.Context, plugin *PluginManifest) (string, error) {
	rInstaller, exists := plugin.Spec.Installers["r"]
	if !exists {
		return "", fmt.Errorf("plugin %s does not support R installer", plugin.Metadata.Name)
	}

	if len(rInstaller.Packages) == 0 {
		return "", fmt.Errorf("no packages defined for plugin %s", plugin.Metadata.Name)
	}

	// Check version of first package
	pkg := rInstaller.Packages[0]
	checkCmd := fmt.Sprintf("R -e \"packageVersion('%s')\"", pkg.Name)

	result, err := r.executor.Execute(ctx, r.instanceName, checkCmd)
	if err != nil {
		return "", fmt.Errorf("failed to check version: %w", err)
	}

	if result.ExitCode != 0 {
		return "", nil // Not installed
	}

	// Parse version from output (format: [1] '0.3.16')
	output := strings.TrimSpace(result.Stdout)
	if strings.Contains(output, "'") {
		parts := strings.Split(output, "'")
		if len(parts) >= 2 {
			return parts[1], nil
		}
	}

	return "", nil
}

// IsCompatible checks if the plugin is compatible with the environment
func (r *RPackageInstaller) IsCompatible(ctx context.Context, plugin *PluginManifest) (bool, error) {
	// Check if R installer is defined
	_, exists := plugin.Spec.Installers["r"]
	if !exists {
		return false, nil
	}

	// Validate R is available
	if err := r.Validate(ctx, plugin); err != nil {
		return false, nil
	}

	return true, nil
}

// mergeParameters merges plugin configuration parameters with provided parameters
func (r *RPackageInstaller) mergeParameters(pluginParams map[string]TemplateParameter, providedParams map[string]interface{}) map[string]interface{} {
	merged := make(map[string]interface{})

	// Start with plugin defaults
	for key, param := range pluginParams {
		merged[key] = param.Default
	}

	// Override with provided parameters
	for key, value := range providedParams {
		merged[key] = value
	}

	return merged
}

// substituteParameters performs template variable substitution
func (r *RPackageInstaller) substituteParameters(text string, params map[string]interface{}) (string, error) {
	tmpl, err := template.New("install").Parse(text)
	if err != nil {
		return "", fmt.Errorf("failed to parse template: %w", err)
	}

	var result strings.Builder
	if err := tmpl.Execute(&result, params); err != nil {
		return "", fmt.Errorf("failed to execute template: %w", err)
	}

	return result.String(), nil
}
