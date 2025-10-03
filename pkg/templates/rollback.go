// Package templates provides template rollback capabilities.
package templates

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// TemplateRollbackManager manages rollback checkpoints and restoration
type TemplateRollbackManager struct {
	executor RemoteExecutor
}

// NewTemplateRollbackManager creates a new template rollback manager
func NewTemplateRollbackManager(executor RemoteExecutor) *TemplateRollbackManager {
	return &TemplateRollbackManager{
		executor: executor,
	}
}

// RollbackCheckpoint represents a point-in-time snapshot of instance state
type RollbackCheckpoint struct {
	ID              string             `json:"id"`
	CreatedAt       time.Time          `json:"created_at"`
	InstanceName    string             `json:"instance_name"`
	Description     string             `json:"description"`
	PackageSnapshot []InstalledPackage `json:"package_snapshot"`
	ServiceSnapshot []RunningService   `json:"service_snapshot"`
	UserSnapshot    []ExistingUser     `json:"user_snapshot"`
	FilesBackedUp   []BackupFile       `json:"files_backed_up"`
	EnvironmentVars map[string]string  `json:"environment_vars"`
}

// BackupFile represents a backed up configuration file
type BackupFile struct {
	OriginalPath string `json:"original_path"`
	BackupPath   string `json:"backup_path"`
	Checksum     string `json:"checksum"`
}

// CreateCheckpoint creates a rollback checkpoint before applying template changes
func (r *TemplateRollbackManager) CreateCheckpoint(ctx context.Context, instanceName string) (string, error) {
	checkpointID := fmt.Sprintf("checkpoint-%d", time.Now().Unix())

	// 1. Inspect current state
	inspector := NewInstanceStateInspector(r.executor)
	currentState, err := inspector.InspectInstance(ctx, instanceName)
	if err != nil {
		return "", fmt.Errorf("failed to inspect instance state: %w", err)
	}

	// 2. Create checkpoint structure
	checkpoint := &RollbackCheckpoint{
		ID:              checkpointID,
		CreatedAt:       time.Now(),
		InstanceName:    instanceName,
		Description:     "Pre-template application checkpoint",
		PackageSnapshot: currentState.Packages,
		ServiceSnapshot: currentState.Services,
		UserSnapshot:    currentState.Users,
		FilesBackedUp:   []BackupFile{},
		EnvironmentVars: make(map[string]string),
	}

	// 3. Backup critical configuration files
	backupFiles, err := r.backupConfigurationFiles(ctx, instanceName, checkpointID)
	if err != nil {
		// Non-fatal - continue without file backups
		fmt.Printf("Warning: failed to backup configuration files: %v\n", err)
	} else {
		checkpoint.FilesBackedUp = backupFiles
	}

	// 4. Capture environment variables
	envVars, err := r.captureEnvironmentVariables(ctx, instanceName)
	if err != nil {
		// Non-fatal - continue without env vars
		fmt.Printf("Warning: failed to capture environment variables: %v\n", err)
	} else {
		checkpoint.EnvironmentVars = envVars
	}

	// 5. Save checkpoint to instance
	if err := r.saveCheckpoint(ctx, instanceName, checkpoint); err != nil {
		return "", fmt.Errorf("failed to save checkpoint: %w", err)
	}

	return checkpointID, nil
}

// backupConfigurationFiles backs up critical configuration files
func (r *TemplateRollbackManager) backupConfigurationFiles(ctx context.Context, instanceName, checkpointID string) ([]BackupFile, error) {
	// Define critical files to backup
	criticalFiles := []string{
		"/etc/passwd",
		"/etc/group",
		"/etc/sudoers",
		"/etc/systemd/system",
		"/etc/apt/sources.list",
		"/etc/yum.repos.d",
		"/home/*/.bashrc",
		"/home/*/.profile",
	}

	var backupFiles []BackupFile
	backupDir := fmt.Sprintf("/opt/cloudworkstation/checkpoints/%s", checkpointID)

	// Create backup directory
	script := fmt.Sprintf("mkdir -p %s", backupDir)
	if _, err := r.executor.Execute(ctx, instanceName, script); err != nil {
		return nil, fmt.Errorf("failed to create backup directory: %w", err)
	}

	// Backup each file
	for _, filePath := range criticalFiles {
		backupPath := fmt.Sprintf("%s%s", backupDir, filePath)

		// Create backup subdirectories
		script := fmt.Sprintf("mkdir -p $(dirname %s)", backupPath)
		if _, err := r.executor.Execute(ctx, instanceName, script); err != nil {
			continue // Skip this file
		}

		// Copy file if it exists
		script = fmt.Sprintf("if [ -f %s ]; then cp %s %s; fi", filePath, filePath, backupPath)
		result, err := r.executor.Execute(ctx, instanceName, script)
		if err != nil || result.ExitCode != 0 {
			continue // Skip this file
		}

		// Calculate checksum
		checksumScript := fmt.Sprintf("if [ -f %s ]; then md5sum %s | cut -d' ' -f1; fi", backupPath, backupPath)
		checksumResult, err := r.executor.Execute(ctx, instanceName, checksumScript)
		checksum := ""
		if err == nil && checksumResult.ExitCode == 0 {
			checksum = checksumResult.Stdout
		}

		backupFiles = append(backupFiles, BackupFile{
			OriginalPath: filePath,
			BackupPath:   backupPath,
			Checksum:     checksum,
		})
	}

	return backupFiles, nil
}

// captureEnvironmentVariables captures important environment variables
func (r *TemplateRollbackManager) captureEnvironmentVariables(ctx context.Context, instanceName string) (map[string]string, error) {
	envVars := make(map[string]string)

	// Capture important environment variables
	importantVars := []string{
		"PATH",
		"LD_LIBRARY_PATH",
		"PYTHONPATH",
		"CONDA_DEFAULT_ENV",
		"VIRTUAL_ENV",
		"JAVA_HOME",
		"GOPATH",
		"SPACK_ROOT",
	}

	for _, varName := range importantVars {
		script := fmt.Sprintf("echo $%s", varName)
		result, err := r.executor.Execute(ctx, instanceName, script)
		if err == nil && result.ExitCode == 0 {
			envVars[varName] = result.Stdout
		}
	}

	return envVars, nil
}

// saveCheckpoint saves the checkpoint to the instance
func (r *TemplateRollbackManager) saveCheckpoint(ctx context.Context, instanceName string, checkpoint *RollbackCheckpoint) error {
	// Serialize checkpoint to JSON
	checkpointJSON, err := json.MarshalIndent(checkpoint, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to serialize checkpoint: %w", err)
	}

	// Create cloudworkstation directory
	script := "mkdir -p /opt/cloudworkstation/checkpoints"
	if _, err := r.executor.Execute(ctx, instanceName, script); err != nil {
		return fmt.Errorf("failed to create checkpoints directory: %w", err)
	}

	// Write checkpoint file
	checkpointPath := fmt.Sprintf("/opt/cloudworkstation/checkpoints/%s.json", checkpoint.ID)
	script = fmt.Sprintf("cat > %s << 'EOF'\n%s\nEOF", checkpointPath, string(checkpointJSON))

	result, err := r.executor.ExecuteScript(ctx, instanceName, script)
	if err != nil {
		return fmt.Errorf("failed to write checkpoint file: %w", err)
	}

	if result.ExitCode != 0 {
		return fmt.Errorf("checkpoint write failed (exit code %d): %s", result.ExitCode, result.Stderr)
	}

	return nil
}

// RollbackToCheckpoint rolls back the instance to a previous checkpoint
func (r *TemplateRollbackManager) RollbackToCheckpoint(ctx context.Context, instanceName, checkpointID string) error {
	// 1. Load checkpoint
	checkpoint, err := r.loadCheckpoint(ctx, instanceName, checkpointID)
	if err != nil {
		return fmt.Errorf("failed to load checkpoint: %w", err)
	}

	// 2. Restore configuration files
	if err := r.restoreConfigurationFiles(ctx, instanceName, checkpoint.FilesBackedUp); err != nil {
		return fmt.Errorf("failed to restore configuration files: %w", err)
	}

	// 3. Restore services to previous state
	if err := r.restoreServices(ctx, instanceName, checkpoint.ServiceSnapshot); err != nil {
		return fmt.Errorf("failed to restore services: %w", err)
	}

	// 4. Remove packages that were added after checkpoint
	if err := r.removeAddedPackages(ctx, instanceName, checkpoint); err != nil {
		// Non-fatal - some packages might be difficult to remove
		fmt.Printf("Warning: failed to remove some packages: %v\n", err)
	}

	// 5. Restore environment variables
	if err := r.restoreEnvironmentVariables(ctx, instanceName, checkpoint.EnvironmentVars); err != nil {
		// Non-fatal - env vars will be restored on next login
		fmt.Printf("Warning: failed to restore environment variables: %v\n", err)
	}

	return nil
}

// loadCheckpoint loads a checkpoint from the instance
func (r *TemplateRollbackManager) loadCheckpoint(ctx context.Context, instanceName, checkpointID string) (*RollbackCheckpoint, error) {
	checkpointPath := fmt.Sprintf("/opt/cloudworkstation/checkpoints/%s.json", checkpointID)

	result, err := r.executor.Execute(ctx, instanceName, fmt.Sprintf("cat %s", checkpointPath))
	if err != nil {
		return nil, fmt.Errorf("failed to read checkpoint file: %w", err)
	}

	if result.ExitCode != 0 {
		return nil, fmt.Errorf("checkpoint file not found or unreadable: %s", result.Stderr)
	}

	var checkpoint RollbackCheckpoint
	if err := json.Unmarshal([]byte(result.Stdout), &checkpoint); err != nil {
		return nil, fmt.Errorf("failed to parse checkpoint: %w", err)
	}

	return &checkpoint, nil
}

// restoreConfigurationFiles restores backed up configuration files
func (r *TemplateRollbackManager) restoreConfigurationFiles(ctx context.Context, instanceName string, backupFiles []BackupFile) error {
	for _, backup := range backupFiles {
		// Restore file from backup
		script := fmt.Sprintf("if [ -f %s ]; then cp %s %s; fi", backup.BackupPath, backup.BackupPath, backup.OriginalPath)
		result, err := r.executor.Execute(ctx, instanceName, script)
		if err != nil || result.ExitCode != 0 {
			// Continue with other files even if one fails
			continue
		}
	}

	return nil
}

// restoreServices restores services to their checkpoint state
func (r *TemplateRollbackManager) restoreServices(ctx context.Context, instanceName string, serviceSnapshot []RunningService) error {
	var script strings.Builder

	script.WriteString("#!/bin/bash\n")
	script.WriteString("# Service restoration script\n\n")

	for _, svc := range serviceSnapshot {
		if svc.Status == "running" {
			script.WriteString(fmt.Sprintf("systemctl start %s 2>/dev/null || true\n", svc.Name))
		} else {
			script.WriteString(fmt.Sprintf("systemctl stop %s 2>/dev/null || true\n", svc.Name))
		}

		if svc.Enabled {
			script.WriteString(fmt.Sprintf("systemctl enable %s 2>/dev/null || true\n", svc.Name))
		} else {
			script.WriteString(fmt.Sprintf("systemctl disable %s 2>/dev/null || true\n", svc.Name))
		}
	}

	result, err := r.executor.ExecuteScript(ctx, instanceName, script.String())
	if err != nil {
		return fmt.Errorf("failed to execute service restoration script: %w", err)
	}

	if result.ExitCode != 0 {
		// Non-fatal - some services might not exist anymore
		fmt.Printf("Warning: some services could not be restored: %s\n", result.Stderr)
	}

	return nil
}

// removeAddedPackages attempts to remove packages that were added after the checkpoint
func (r *TemplateRollbackManager) removeAddedPackages(ctx context.Context, instanceName string, checkpoint *RollbackCheckpoint) error {
	// Get current package state
	currentState, err := r.inspectCurrentPackageState(ctx, instanceName)
	if err != nil {
		return err
	}

	// Find packages that were added after checkpoint
	packagesToRemove := r.identifyPackagesToRemove(currentState, checkpoint)
	if len(packagesToRemove) == 0 {
		return nil
	}

	// Generate and execute removal script
	return r.executePackageRemoval(ctx, instanceName, currentState.PackageManager, packagesToRemove)
}

// inspectCurrentPackageState gets the current state of packages on the instance
func (r *TemplateRollbackManager) inspectCurrentPackageState(ctx context.Context, instanceName string) (*InstanceState, error) {
	inspector := NewInstanceStateInspector(r.executor)
	currentState, err := inspector.InspectInstance(ctx, instanceName)
	if err != nil {
		return nil, fmt.Errorf("failed to inspect current state: %w", err)
	}
	return currentState, nil
}

// identifyPackagesToRemove compares current packages with checkpoint to find additions
func (r *TemplateRollbackManager) identifyPackagesToRemove(currentState *InstanceState, checkpoint *RollbackCheckpoint) []string {
	// Create lookup map for checkpoint packages
	checkpointPackages := make(map[string]bool)
	for _, pkg := range checkpoint.PackageSnapshot {
		checkpointPackages[pkg.Name] = true
	}

	// Find packages that were added after checkpoint
	var packagesToRemove []string
	for _, pkg := range currentState.Packages {
		if !checkpointPackages[pkg.Name] {
			packagesToRemove = append(packagesToRemove, pkg.Name)
		}
	}

	return packagesToRemove
}

// executePackageRemoval generates and executes the package removal script
func (r *TemplateRollbackManager) executePackageRemoval(ctx context.Context, instanceName, packageManager string, packagesToRemove []string) error {
	// Generate removal script
	script := r.generatePackageRemovalScript(packageManager, packagesToRemove)

	// Execute removal script
	result, err := r.executor.ExecuteScript(ctx, instanceName, script)
	if err != nil {
		return fmt.Errorf("failed to execute package removal script: %w", err)
	}

	// Handle execution results
	r.handleRemovalResults(result)
	return nil
}

// generatePackageRemovalScript creates the removal script based on package manager
func (r *TemplateRollbackManager) generatePackageRemovalScript(packageManager string, packagesToRemove []string) string {
	var script strings.Builder

	// Script header
	script.WriteString("#!/bin/bash\n")
	script.WriteString("# Package removal script\n")
	script.WriteString("set +e  # Continue on errors\n\n")

	// Generate removal commands by package manager type
	r.appendRemovalCommandsByManager(&script, packageManager, packagesToRemove)

	return script.String()
}

// appendRemovalCommandsByManager adds package-manager-specific removal commands
func (r *TemplateRollbackManager) appendRemovalCommandsByManager(script *strings.Builder, packageManager string, packagesToRemove []string) {
	switch packageManager {
	case "apt":
		r.appendAptRemovalCommands(script, packagesToRemove)
	case "dnf":
		r.appendDnfRemovalCommands(script, packagesToRemove)
	case "conda":
		r.appendCondaRemovalCommands(script, packagesToRemove)
	case "pip":
		r.appendPipRemovalCommands(script, packagesToRemove)
	}
}

// appendAptRemovalCommands adds APT-specific removal commands
func (r *TemplateRollbackManager) appendAptRemovalCommands(script *strings.Builder, packagesToRemove []string) {
	for _, pkg := range packagesToRemove {
		script.WriteString(fmt.Sprintf("apt-get remove -y %s 2>/dev/null || true\n", pkg))
	}
	script.WriteString("apt-get autoremove -y 2>/dev/null || true\n")
}

// appendDnfRemovalCommands adds DNF-specific removal commands
func (r *TemplateRollbackManager) appendDnfRemovalCommands(script *strings.Builder, packagesToRemove []string) {
	for _, pkg := range packagesToRemove {
		script.WriteString(fmt.Sprintf("dnf remove -y %s 2>/dev/null || true\n", pkg))
	}
}

// appendCondaRemovalCommands adds Conda-specific removal commands
func (r *TemplateRollbackManager) appendCondaRemovalCommands(script *strings.Builder, packagesToRemove []string) {
	for _, pkg := range packagesToRemove {
		script.WriteString(fmt.Sprintf("conda remove -y %s 2>/dev/null || true\n", pkg))
	}
}

// appendPipRemovalCommands adds pip-specific removal commands
func (r *TemplateRollbackManager) appendPipRemovalCommands(script *strings.Builder, packagesToRemove []string) {
	for _, pkg := range packagesToRemove {
		script.WriteString(fmt.Sprintf("pip uninstall -y %s 2>/dev/null || true\n", pkg))
	}
}

// handleRemovalResults processes the results from package removal execution
func (r *TemplateRollbackManager) handleRemovalResults(result *ExecutionResult) {
	// Package removal is best-effort, so don't fail on non-zero exit codes
	if result.ExitCode != 0 {
		fmt.Printf("Warning: some packages could not be removed: %s\n", result.Stderr)
	}
}

// restoreEnvironmentVariables restores environment variables
func (r *TemplateRollbackManager) restoreEnvironmentVariables(ctx context.Context, instanceName string, envVars map[string]string) error {
	if len(envVars) == 0 {
		return nil
	}

	// Create environment restoration script
	var script strings.Builder

	script.WriteString("#!/bin/bash\n")
	script.WriteString("# Environment variable restoration\n")
	script.WriteString("# These will take effect on next shell session\n\n")

	// Write environment variables to profile
	script.WriteString("cat >> /etc/environment << 'EOF'\n")
	for varName, varValue := range envVars {
		if varValue != "" {
			script.WriteString(fmt.Sprintf("%s=%s\n", varName, varValue))
		}
	}
	script.WriteString("EOF\n")

	result, err := r.executor.ExecuteScript(ctx, instanceName, script.String())
	if err != nil {
		return fmt.Errorf("failed to restore environment variables: %w", err)
	}

	if result.ExitCode != 0 {
		return fmt.Errorf("environment restoration failed: %s", result.Stderr)
	}

	return nil
}

// ListCheckpoints lists available rollback checkpoints for an instance
func (r *TemplateRollbackManager) ListCheckpoints(ctx context.Context, instanceName string) ([]RollbackCheckpoint, error) {
	// List checkpoint files
	result, err := r.executor.Execute(ctx, instanceName, "ls -1 /opt/cloudworkstation/checkpoints/*.json 2>/dev/null || true")
	if err != nil {
		return nil, fmt.Errorf("failed to list checkpoints: %w", err)
	}

	if result.ExitCode != 0 || result.Stdout == "" {
		// No checkpoints found
		return []RollbackCheckpoint{}, nil
	}

	checkpointFiles := strings.Split(strings.TrimSpace(result.Stdout), "\n")
	var checkpoints []RollbackCheckpoint

	for _, filePath := range checkpointFiles {
		// Extract checkpoint ID from filename
		parts := strings.Split(filePath, "/")
		filename := parts[len(parts)-1]
		checkpointID := strings.TrimSuffix(filename, ".json")

		checkpoint, err := r.loadCheckpoint(ctx, instanceName, checkpointID)
		if err != nil {
			// Skip corrupted checkpoints
			continue
		}

		checkpoints = append(checkpoints, *checkpoint)
	}

	return checkpoints, nil
}
