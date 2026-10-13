// Package templates provides Vim plugin installer implementation.
package templates

import (
	"context"
	"fmt"
	"strings"
	"text/template"
)

// VimPluginInstaller installs Vim plugins on remote instances
type VimPluginInstaller struct {
	executor     RemoteExecutor
	instanceName string
}

// NewVimPluginInstaller creates a new Vim plugin installer
func NewVimPluginInstaller(executor RemoteExecutor, instanceName string) *VimPluginInstaller {
	return &VimPluginInstaller{
		executor:     executor,
		instanceName: instanceName,
	}
}

// Name returns the installer name
func (v *VimPluginInstaller) Name() string {
	return "vim"
}

// Install installs a Vim plugin with specified version and configuration
func (v *VimPluginInstaller) Install(ctx context.Context, plugin *PluginManifest, version string, params map[string]interface{}) error {
	// Get Vim-specific installer from plugin manifest
	vimInstaller, exists := plugin.Spec.Installers["vim"]
	if !exists {
		return fmt.Errorf("plugin %s does not support Vim installer", plugin.Metadata.Name)
	}

	// Merge plugin configuration parameters with provided parameters
	mergedParams := v.mergeParameters(plugin.Spec.Configuration.Parameters, params)
	mergedParams["version"] = version

	// Install each plugin
	for _, pkg := range vimInstaller.Packages {
		// Substitute parameters in install command
		installCmd, err := v.substituteParameters(pkg.InstallCommand, mergedParams)
		if err != nil {
			return fmt.Errorf("failed to substitute parameters for plugin %s: %w", pkg.Name, err)
		}

		// Execute installation command
		result, err := v.executor.Execute(ctx, v.instanceName, installCmd)
		if err != nil {
			return fmt.Errorf("failed to execute install command for plugin %s: %w", pkg.Name, err)
		}

		if result.ExitCode != 0 {
			return fmt.Errorf("install command failed for plugin %s (exit code %d): %s",
				pkg.Name, result.ExitCode, result.Stderr)
		}
	}

	// Create configuration files if specified
	for _, config := range vimInstaller.Config {
		// Substitute parameters in config content
		content, err := v.substituteParameters(config.Content, mergedParams)
		if err != nil {
			return fmt.Errorf("failed to substitute parameters in config %s: %w", config.File, err)
		}

		// Create directory and write config file
		writeCmd := fmt.Sprintf("mkdir -p $(dirname %s) && cat > %s <<'EOF'\n%s\nEOF",
			config.Path, config.Path, content)

		result, err := v.executor.Execute(ctx, v.instanceName, writeCmd)
		if err != nil {
			return fmt.Errorf("failed to create config file %s: %w", config.File, err)
		}

		if result.ExitCode != 0 {
			return fmt.Errorf("failed to write config %s (exit code %d): %s",
				config.File, result.ExitCode, result.Stderr)
		}
	}

	// Execute post-install script if specified
	if vimInstaller.PostInstall != "" {
		postInstall, err := v.substituteParameters(vimInstaller.PostInstall, mergedParams)
		if err != nil {
			return fmt.Errorf("failed to substitute parameters in post-install script: %w", err)
		}

		result, err := v.executor.ExecuteScript(ctx, v.instanceName, postInstall)
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

// Uninstall removes a Vim plugin
func (v *VimPluginInstaller) Uninstall(ctx context.Context, plugin *PluginManifest) error {
	vimInstaller, exists := plugin.Spec.Installers["vim"]
	if !exists {
		return fmt.Errorf("plugin %s does not support Vim installer", plugin.Metadata.Name)
	}

	for _, pkg := range vimInstaller.Packages {
		// Basic uninstall - remove plugin directory
		// This is plugin-manager specific and simplified here
		uninstallCmd := fmt.Sprintf("rm -rf ~/.vim/plugged/%s", pkg.Name)

		result, err := v.executor.Execute(ctx, v.instanceName, uninstallCmd)
		if err != nil {
			return fmt.Errorf("failed to uninstall plugin %s: %w", pkg.Name, err)
		}

		if result.ExitCode != 0 {
			// Log warning but continue - plugin might not be installed
			fmt.Printf("Warning: failed to uninstall plugin %s: %s\n", pkg.Name, result.Stderr)
		}
	}

	return nil
}

// Validate checks if Vim is available and prerequisites are met
func (v *VimPluginInstaller) Validate(ctx context.Context, plugin *PluginManifest) error {
	// Check if Vim is installed
	result, err := v.executor.Execute(ctx, v.instanceName, "vim --version")
	if err != nil {
		return fmt.Errorf("vim is not installed: %w", err)
	}

	if result.ExitCode != 0 {
		return fmt.Errorf("vim is not available (exit code %d): %s", result.ExitCode, result.Stderr)
	}

	// Check for plugin manager (vim-plug is common)
	plugCheck, err := v.executor.Execute(ctx, v.instanceName, "test -f ~/.vim/autoload/plug.vim")
	if err != nil || plugCheck.ExitCode != 0 {
		// Warn but don't fail - plugin manager might be installed differently
		fmt.Printf("Warning: vim-plug not detected at ~/.vim/autoload/plug.vim\n")
	}

	// Check compatibility constraints if specified
	if plugin.Spec.Compatibility.Tools != nil {
		if constraint, exists := plugin.Spec.Compatibility.Tools["vim"]; exists {
			if installed := parseVersionFromOutput(result.Stdout); installed != "" {
				if constraint.MinVersion != "" && !versionSatisfiesConstraint(installed, ">="+constraint.MinVersion) {
					return fmt.Errorf("installed Vim version %s is below minimum required %s",
						installed, constraint.MinVersion)
				}
				if constraint.MaxVersion != "" && !versionSatisfiesConstraint(installed, "<="+constraint.MaxVersion) {
					return fmt.Errorf("installed Vim version %s exceeds maximum allowed %s",
						installed, constraint.MaxVersion)
				}
			}
		}
	}

	return nil
}

// GetInstalledVersion returns the installed version of a plugin
func (v *VimPluginInstaller) GetInstalledVersion(ctx context.Context, plugin *PluginManifest) (string, error) {
	vimInstaller, exists := plugin.Spec.Installers["vim"]
	if !exists {
		return "", fmt.Errorf("plugin %s does not support Vim installer", plugin.Metadata.Name)
	}

	if len(vimInstaller.Packages) == 0 {
		return "", fmt.Errorf("no plugins defined for plugin %s", plugin.Metadata.Name)
	}

	// Check if plugin directory exists (basic check)
	pkg := vimInstaller.Packages[0]
	checkCmd := fmt.Sprintf("test -d ~/.vim/plugged/%s && echo 'installed' || echo 'not installed'", pkg.Name)

	result, err := v.executor.Execute(ctx, v.instanceName, checkCmd)
	if err != nil {
		return "", fmt.Errorf("failed to check version: %w", err)
	}

	output := strings.TrimSpace(result.Stdout)
	if output == "installed" {
		// Run vim --version to extract the actual version number
		versionResult, err := v.executor.Execute(ctx, v.instanceName, "vim --version")
		if err == nil && versionResult.ExitCode == 0 {
			if v := parseVersionFromOutput(versionResult.Stdout); v != "" {
				return v, nil
			}
		}
		return "installed", nil
	}

	return "", nil
}

// IsCompatible checks if the plugin is compatible with the environment
func (v *VimPluginInstaller) IsCompatible(ctx context.Context, plugin *PluginManifest) (bool, error) {
	// Check if Vim installer is defined
	_, exists := plugin.Spec.Installers["vim"]
	if !exists {
		return false, nil
	}

	// Validate Vim is available
	if err := v.Validate(ctx, plugin); err != nil {
		return false, nil
	}

	return true, nil
}

// mergeParameters merges plugin configuration parameters with provided parameters
func (v *VimPluginInstaller) mergeParameters(pluginParams map[string]TemplateParameter, providedParams map[string]interface{}) map[string]interface{} {
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
func (v *VimPluginInstaller) substituteParameters(text string, params map[string]interface{}) (string, error) {
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
