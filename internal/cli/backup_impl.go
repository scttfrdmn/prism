// Package cli - Backup Implementation Layer
//
// ARCHITECTURE NOTE: This file contains backup and restore command business logic.
// These commands are registered in root_command.go and called directly from the CLI.
//
// This follows Prism's command architecture pattern:
//   - Single-layer implementation for straightforward operations
//   - Direct integration with root command structure
//   - API-driven operations with consistent error handling
//
// DO NOT REMOVE THIS FILE - it is actively used by root_command.go and App methods.
package cli

import (
	"fmt"
	"os"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/scttfrdmn/prism/pkg/types"
)

// BackupCommands handles data backup and restore management operations (implementation layer)
type BackupCommands struct {
	app *App
}

// NewBackupCommands creates backup commands handler
func NewBackupCommands(app *App) *BackupCommands {
	return &BackupCommands{app: app}
}

// Backup handles the backup command with comprehensive data-level backup operations
func (bc *BackupCommands) Backup(args []string) error {
	if len(args) < 1 {
		return bc.showBackupUsage()
	}

	action := args[0]
	actionArgs := args[1:]

	// Ensure daemon is running (auto-start if needed)
	if err := bc.app.ensureDaemonRunning(); err != nil {
		return err
	}

	switch action {
	case "create":
		return bc.createBackup(actionArgs)
	case "list":
		return bc.listBackups(actionArgs)
	case "info", "show":
		return bc.getBackupInfo(actionArgs)
	case "delete":
		return bc.deleteBackup(actionArgs)
	case "contents":
		return bc.listBackupContents(actionArgs)
	case "verify":
		return bc.verifyBackup(actionArgs)
	default:
		return fmt.Errorf("unknown backup action: %s\n\n%s", action, bc.getBackupUsageText())
	}
}

// Restore handles the restore command with comprehensive data restoration operations
func (bc *BackupCommands) Restore(args []string) error {
	if len(args) < 1 {
		return bc.showRestoreUsage()
	}

	action := args[0]
	actionArgs := args[1:]

	// Ensure daemon is running (auto-start if needed)
	if err := bc.app.ensureDaemonRunning(); err != nil {
		return err
	}

	switch action {
	case "restore":
		// Handle both "prism restore backup-name instance-name" and "prism restore restore backup-name instance-name"
		if len(args) >= 2 {
			return bc.restoreFromBackup(args)
		}
		return bc.showRestoreUsage()
	case "list":
		return bc.listBackupContents(actionArgs)
	case "verify":
		return bc.verifyRestore(actionArgs)
	case "status":
		return bc.getRestoreStatus(actionArgs)
	case "operations":
		return bc.listRestoreOperations(actionArgs)
	default:
		// If first argument is not a subcommand, treat it as "restore backup-name instance-name"
		return bc.restoreFromBackup(args)
	}
}

// createBackup creates a data backup from an instance
func (bc *BackupCommands) createBackup(args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("usage: cws backup create <workspace-name> <backup-name> [options]")
	}

	req, err := bc.parseCreateBackupFlags(args)
	if err != nil {
		return err
	}

	bc.applyCreateBackupDefaults(&req)
	bc.displayCreateBackupInfo(req)

	result, err := bc.app.apiClient.CreateBackup(bc.app.ctx, req)
	if err != nil {
		return WrapAPIError("create backup", err)
	}

	bc.displayCreateBackupResult(result)

	if req.Wait {
		fmt.Printf("\n⏳ Monitoring backup creation progress...\n")
		return bc.monitorBackupProgress(result.BackupID, result.BackupName)
	}

	bc.displayCreateBackupNextSteps(req.BackupName)
	return nil
}

// parseCreateBackupFlags parses command-line flags for backup creation
func (bc *BackupCommands) parseCreateBackupFlags(args []string) (types.BackupCreateRequest, error) {
	req := types.BackupCreateRequest{
		InstanceName: args[0],
		BackupName:   args[1],
		StorageType:  "s3", // Default to S3
		Encrypted:    true, // Default to encrypted
	}

	for i := 2; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "--description" && i+1 < len(args):
			req.Description = args[i+1]
			i++
		case arg == "--include" && i+1 < len(args):
			paths := strings.Split(args[i+1], ",")
			req.IncludePaths = append(req.IncludePaths, paths...)
			i++
		case arg == "--exclude" && i+1 < len(args):
			paths := strings.Split(args[i+1], ",")
			req.ExcludePaths = append(req.ExcludePaths, paths...)
			i++
		case arg == "--full":
			req.Full = true
		case arg == "--incremental":
			req.Incremental = true
		case arg == "--storage" && i+1 < len(args):
			req.StorageType = args[i+1]
			i++
		case arg == "--no-encryption":
			req.Encrypted = false
		case arg == "--wait":
			req.Wait = true
		case strings.HasPrefix(arg, "--description="):
			req.Description = strings.TrimPrefix(arg, "--description=")
		case strings.HasPrefix(arg, "--include="):
			paths := strings.Split(strings.TrimPrefix(arg, "--include="), ",")
			req.IncludePaths = append(req.IncludePaths, paths...)
		case strings.HasPrefix(arg, "--exclude="):
			paths := strings.Split(strings.TrimPrefix(arg, "--exclude="), ",")
			req.ExcludePaths = append(req.ExcludePaths, paths...)
		case strings.HasPrefix(arg, "--storage="):
			req.StorageType = strings.TrimPrefix(arg, "--storage=")
		default:
			return req, fmt.Errorf("unknown option: %s", arg)
		}
	}

	return req, nil
}

// applyCreateBackupDefaults applies default values to backup request
func (bc *BackupCommands) applyCreateBackupDefaults(req *types.BackupCreateRequest) {
	if len(req.IncludePaths) == 0 {
		req.IncludePaths = []string{"/home", "/data", "/opt/research"}
	}
	if len(req.ExcludePaths) == 0 {
		req.ExcludePaths = []string{".cache", ".tmp", "*.log", "/proc", "/sys", "/dev"}
	}
}

// displayCreateBackupInfo displays backup creation information
func (bc *BackupCommands) displayCreateBackupInfo(req types.BackupCreateRequest) {
	fmt.Printf("💾 Creating data backup '%s' from workspace '%s'...\n", req.BackupName, req.InstanceName)

	if req.Incremental {
		fmt.Printf("📈 Incremental backup mode - only changed files since last backup\n")
	} else {
		fmt.Printf("📦 Full backup mode - all specified files and directories\n")
	}

	if len(req.IncludePaths) > 0 {
		fmt.Printf("📁 Including paths: %s\n", strings.Join(req.IncludePaths, ", "))
	}
	if len(req.ExcludePaths) > 0 {
		fmt.Printf("🚫 Excluding paths: %s\n", strings.Join(req.ExcludePaths, ", "))
	}

	fmt.Printf("💿 Storage: %s (%s)\n", strings.ToUpper(req.StorageType),
		map[bool]string{true: "encrypted", false: "unencrypted"}[req.Encrypted])
}

// displayCreateBackupResult displays backup creation result
func (bc *BackupCommands) displayCreateBackupResult(result *types.BackupCreateResult) {
	fmt.Printf("✅ Backup creation initiated\n")
	fmt.Printf("   Backup ID: %s\n", result.BackupID)
	fmt.Printf("   Source Instance: %s\n", result.SourceInstance)
	fmt.Printf("   Backup Type: %s\n", strings.Title(result.BackupType))
	fmt.Printf("   Storage Location: %s\n", result.StorageLocation)
	fmt.Printf("   Estimated Completion: %d minutes\n", result.EstimatedCompletionMinutes)
	if result.EstimatedSizeBytes > 0 {
		fmt.Printf("   Estimated Size: %s\n", bc.formatBytes(result.EstimatedSizeBytes))
	}
	fmt.Printf("   Monthly Storage Cost: $%.2f\n", result.StorageCostMonthly)
	fmt.Printf("   Created: %s\n", result.CreatedAt.Format("2006-01-02 15:04:05"))
}

// displayCreateBackupNextSteps displays next steps after backup creation
func (bc *BackupCommands) displayCreateBackupNextSteps(backupName string) {
	fmt.Printf("\n💡 Check progress with: cws backup info %s\n", backupName)
	fmt.Printf("💡 List backup contents: cws backup contents %s\n", backupName)
}

// listBackups lists all data backups
func (bc *BackupCommands) listBackups(args []string) error {
	response, err := bc.app.apiClient.ListBackups(bc.app.ctx)
	if err != nil {
		return WrapAPIError("list backups", err)
	}

	if len(response.Backups) == 0 {
		fmt.Println("No data backups found.")
		fmt.Println("Create one with: cws backup create <workspace-name> <backup-name>")
		return nil
	}

	fmt.Printf("💾 Data Backups (%d total):\n\n", response.Count)

	w := tabwriter.NewWriter(os.Stdout, TabWriterMinWidth, TabWriterTabWidth, TabWriterPadding, TabWriterPadChar, TabWriterFlags)
	_, _ = fmt.Fprintln(w, "NAME\tSOURCE INSTANCE\tTYPE\tSTORAGE\tSIZE\tCOST/MONTH\tCREATED")

	for _, backup := range response.Backups {
		backupType := strings.Title(backup.BackupType)
		if backup.BackupType == "" {
			backupType = "Full"
		}

		storageType := strings.ToUpper(backup.StorageType)
		if backup.Encrypted {
			storageType += "+ENC"
		}

		_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t$%.2f\t%s\n",
			backup.BackupName,
			backup.SourceInstance,
			backupType,
			storageType,
			bc.formatBytes(backup.SizeBytes),
			backup.StorageCostMonthly,
			backup.CreatedAt.Format("2006-01-02"),
		)
	}

	_ = w.Flush()

	// Show summary
	fmt.Printf("\n💰 Storage Summary:\n")
	fmt.Printf("   Total Size: %s\n", bc.formatBytes(response.TotalSize))
	fmt.Printf("   Monthly Cost: $%.2f\n", response.TotalCost)

	if len(response.StorageTypes) > 0 {
		fmt.Printf("   Storage Types: ")
		var types []string
		for storageType, count := range response.StorageTypes {
			types = append(types, fmt.Sprintf("%s (%d)", strings.ToUpper(storageType), count))
		}
		fmt.Printf("%s\n", strings.Join(types, ", "))
	}

	fmt.Printf("\n💡 Use 'cws backup info <name>' for detailed information\n")
	fmt.Printf("💡 Restore data with: cws restore <backup-name> <workspace-name>\n")

	return nil
}

// getBackupInfo gets detailed information about a backup
func (bc *BackupCommands) getBackupInfo(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: cws backup info <backup-name>")
	}

	backupName := args[0]

	backup, err := bc.app.apiClient.GetBackup(bc.app.ctx, backupName)
	if err != nil {
		return WrapAPIError("get backup info", err)
	}

	fmt.Printf("💾 Backup: %s\n", backup.BackupName)
	fmt.Printf("   Backup ID: %s\n", backup.BackupID)
	fmt.Printf("   Source Instance: %s\n", backup.SourceInstance)
	if backup.SourceInstanceId != "" {
		fmt.Printf("   Source Instance ID: %s\n", backup.SourceInstanceId)
	}
	if backup.Description != "" {
		fmt.Printf("   Description: %s\n", backup.Description)
	}
	fmt.Printf("   State: %s\n", strings.ToUpper(backup.State))
	fmt.Printf("   Backup Type: %s\n", strings.Title(backup.BackupType))
	fmt.Printf("   Storage Type: %s\n", strings.ToUpper(backup.StorageType))
	fmt.Printf("   Storage Location: %s\n", backup.StorageLocation)
	fmt.Printf("   Size: %s", bc.formatBytes(backup.SizeBytes))
	if backup.CompressedBytes > 0 && backup.CompressedBytes != backup.SizeBytes {
		compression := float64(backup.SizeBytes-backup.CompressedBytes) / float64(backup.SizeBytes) * 100
		fmt.Printf(" (%s compressed, %.1f%% savings)", bc.formatBytes(backup.CompressedBytes), compression)
	}
	fmt.Printf("\n")
	fmt.Printf("   File Count: %d\n", backup.FileCount)
	fmt.Printf("   Encrypted: %s\n", map[bool]string{true: "Yes", false: "No"}[backup.Encrypted])
	fmt.Printf("   Monthly Storage Cost: $%.2f\n", backup.StorageCostMonthly)
	fmt.Printf("   Created: %s\n", backup.CreatedAt.Format("2006-01-02 15:04:05"))

	if backup.CompletedAt != nil {
		fmt.Printf("   Completed: %s\n", backup.CompletedAt.Format("2006-01-02 15:04:05"))
		duration := backup.CompletedAt.Sub(backup.CreatedAt)
		fmt.Printf("   Duration: %s\n", bc.formatDuration(duration))
	}

	if backup.ExpiresAt != nil {
		fmt.Printf("   Expires: %s\n", backup.ExpiresAt.Format("2006-01-02 15:04:05"))
	}

	if backup.ParentBackup != "" {
		fmt.Printf("   Parent Backup: %s (incremental)\n", backup.ParentBackup)
	}

	if len(backup.IncludedPaths) > 0 {
		fmt.Printf("   Included Paths: %s\n", strings.Join(backup.IncludedPaths, ", "))
	}

	if len(backup.ExcludedPaths) > 0 {
		fmt.Printf("   Excluded Paths: %s\n", strings.Join(backup.ExcludedPaths, ", "))
	}

	if backup.ChecksumMD5 != "" {
		fmt.Printf("   Checksum (MD5): %s\n", backup.ChecksumMD5)
	}

	fmt.Printf("\n💡 Operations:\n")
	fmt.Printf("   List contents: cws backup contents %s\n", backup.BackupName)
	fmt.Printf("   Restore data: cws restore %s <workspace-name>\n", backup.BackupName)
	fmt.Printf("   Verify integrity: cws backup verify %s\n", backup.BackupName)
	fmt.Printf("   Delete backup: cws backup delete %s\n", backup.BackupName)

	return nil
}

// deleteBackup deletes a backup
func (bc *BackupCommands) deleteBackup(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: cws backup delete <backup-name>")
	}

	backupName := args[0]

	// Confirmation prompt
	fmt.Printf("⚠️  Are you sure you want to delete backup '%s'? This action cannot be undone.\n", backupName)
	fmt.Printf("Type 'yes' to confirm: ")

	var confirmation string
	_, _ = fmt.Scanln(&confirmation)

	if confirmation != "yes" {
		fmt.Println("❌ Deletion cancelled.")
		return nil
	}

	fmt.Printf("🗑️  Deleting backup '%s'...\n", backupName)

	result, err := bc.app.apiClient.DeleteBackup(bc.app.ctx, backupName)
	if err != nil {
		return WrapAPIError("delete backup", err)
	}

	fmt.Printf("✅ Backup deleted successfully\n")
	fmt.Printf("   Backup ID: %s\n", result.BackupID)
	fmt.Printf("   Storage Type: %s\n", strings.ToUpper(result.StorageType))
	fmt.Printf("   Storage Location: %s\n", result.StorageLocation)
	fmt.Printf("   Deleted Size: %s\n", bc.formatBytes(result.DeletedSizeBytes))
	fmt.Printf("   Monthly Storage Savings: $%.2f\n", result.StorageSavingsMonthly)
	fmt.Printf("   Deleted At: %s\n", result.DeletedAt.Format("2006-01-02 15:04:05"))

	return nil
}

// listBackupContents lists the contents of a backup
func (bc *BackupCommands) listBackupContents(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: cws backup contents <backup-name> [path]")
	}

	backupName := args[0]
	path := "/"
	if len(args) >= 2 {
		path = args[1]
	}

	req := types.BackupContentsRequest{
		BackupName: backupName,
		Path:       path,
		Recursive:  false, // Default to non-recursive
	}

	// Check for recursive flag
	for i := 2; i < len(args); i++ {
		if args[i] == "--recursive" || args[i] == "-r" {
			req.Recursive = true
		}
	}

	response, err := bc.app.apiClient.GetBackupContents(bc.app.ctx, req)
	if err != nil {
		return WrapAPIError("list backup contents", err)
	}

	fmt.Printf("📁 Contents of backup '%s' at path '%s':\n\n", response.BackupName, response.Path)

	if len(response.Files) == 0 {
		fmt.Printf("No files found at path '%s'\n", response.Path)
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, TabWriterMinWidth, TabWriterTabWidth, TabWriterPadding, TabWriterPadChar, TabWriterFlags)
	_, _ = fmt.Fprintln(w, "TYPE\tNAME\tSIZE\tOWNER\tMODIFIED")

	for _, file := range response.Files {
		fileType := "FILE"
		if file.IsDir {
			fileType = "DIR"
		}

		size := bc.formatBytes(file.Size)
		if file.IsDir {
			size = "-"
		}

		owner := file.Owner
		if file.Group != "" && file.Group != file.Owner {
			owner = fmt.Sprintf("%s:%s", file.Owner, file.Group)
		}

		_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
			fileType,
			file.Path,
			size,
			owner,
			file.ModTime.Format("2006-01-02 15:04"),
		)
	}

	_ = w.Flush()

	fmt.Printf("\nSummary: %d items, %s total size\n", response.Count, bc.formatBytes(response.TotalSize))

	if !req.Recursive {
		fmt.Printf("💡 Use '--recursive' to list all files recursively\n")
	}
	fmt.Printf("💡 Restore files: cws restore %s <workspace-name> --path %s\n", backupName, path)

	return nil
}

// verifyBackup verifies backup integrity
func (bc *BackupCommands) verifyBackup(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: cws backup verify <backup-name> [options]")
	}

	req, err := bc.parseVerifyBackupFlags(args)
	if err != nil {
		return err
	}

	bc.displayVerifyBackupStart(req)

	result, err := bc.app.apiClient.VerifyBackup(bc.app.ctx, req)
	if err != nil {
		return WrapAPIError("verify backup", err)
	}

	bc.displayVerifyBackupResults(result)
	return nil
}

// parseVerifyBackupFlags parses verify backup command-line flags
func (bc *BackupCommands) parseVerifyBackupFlags(args []string) (types.BackupVerifyRequest, error) {
	req := types.BackupVerifyRequest{
		BackupName: args[0],
		QuickCheck: false,
	}

	for i := 1; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "--quick":
			req.QuickCheck = true
		case arg == "--paths" && i+1 < len(args):
			paths := strings.Split(args[i+1], ",")
			req.SelectivePaths = paths
			i++
		default:
			return req, fmt.Errorf("unknown option: %s", arg)
		}
	}

	return req, nil
}

// displayVerifyBackupStart displays verification start information
func (bc *BackupCommands) displayVerifyBackupStart(req types.BackupVerifyRequest) {
	fmt.Printf("🔍 Verifying backup '%s'...\n", req.BackupName)
	if req.QuickCheck {
		fmt.Printf("⚡ Quick verification mode - checking metadata only\n")
	} else {
		fmt.Printf("🔒 Full verification mode - checking file integrity and checksums\n")
	}
}

// displayVerifyBackupResults displays verification results
func (bc *BackupCommands) displayVerifyBackupResults(result *types.BackupVerifyResult) {
	fmt.Printf("\n✅ Backup verification completed\n")
	fmt.Printf("   Verification State: %s\n", strings.ToUpper(result.VerificationState))
	fmt.Printf("   Files Checked: %d\n", result.CheckedFileCount)
	fmt.Printf("   Verified Size: %s\n", bc.formatBytes(result.VerifiedBytes))

	bc.displayCorruptFiles(result)
	bc.displayMissingFiles(result)

	if result.CorruptFileCount == 0 && result.MissingFileCount == 0 {
		fmt.Printf("   ✅ All files verified successfully\n")
	}

	duration := time.Since(result.VerificationStarted)
	if result.VerificationCompleted != nil {
		duration = result.VerificationCompleted.Sub(result.VerificationStarted)
	}
	fmt.Printf("   Verification Time: %s\n", bc.formatDuration(duration))
}

// displayCorruptFiles displays corrupt files from verification
func (bc *BackupCommands) displayCorruptFiles(result *types.BackupVerifyResult) {
	if result.CorruptFileCount > 0 {
		fmt.Printf("   ❌ Corrupt Files: %d\n", result.CorruptFileCount)
		if len(result.CorruptFiles) > 0 {
			fmt.Printf("   Corrupt Files: %s\n", strings.Join(result.CorruptFiles[:minInt(5, len(result.CorruptFiles))], ", "))
			if len(result.CorruptFiles) > 5 {
				fmt.Printf("   ... and %d more\n", len(result.CorruptFiles)-5)
			}
		}
	}
}

// displayMissingFiles displays missing files from verification
func (bc *BackupCommands) displayMissingFiles(result *types.BackupVerifyResult) {
	if result.MissingFileCount > 0 {
		fmt.Printf("   ❌ Missing Files: %d\n", result.MissingFileCount)
		if len(result.MissingFiles) > 0 {
			fmt.Printf("   Missing Files: %s\n", strings.Join(result.MissingFiles[:minInt(5, len(result.MissingFiles))], ", "))
			if len(result.MissingFiles) > 5 {
				fmt.Printf("   ... and %d more\n", len(result.MissingFiles)-5)
			}
		}
	}
}

// restoreFromBackup restores data from a backup
func (bc *BackupCommands) restoreFromBackup(args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("usage: cws restore <backup-name> <target-instance> [options]")
	}

	req, err := bc.parseRestoreFlags(args)
	if err != nil {
		return err
	}

	bc.displayRestoreStart(req)

	result, err := bc.app.apiClient.RestoreBackup(bc.app.ctx, req)
	if err != nil {
		return WrapAPIError("restore backup", err)
	}

	bc.displayRestoreResult(req, result)

	if req.Wait && !req.DryRun {
		fmt.Printf("\n⏳ Monitoring restore progress...\n")
		return bc.monitorRestoreProgress(result.RestoreID, result.BackupName, result.TargetInstance)
	}

	bc.displayRestoreNextSteps(req, result)
	return nil
}

// parseRestoreFlags parses restore command-line flags
func (bc *BackupCommands) parseRestoreFlags(args []string) (types.RestoreRequest, error) {
	req := types.RestoreRequest{
		BackupName:      args[0],
		TargetInstance:  args[1],
		RestorePath:     "/", // Default to root
		PreservePerms:   true,
		PreserveOwner:   true,
		VerifyIntegrity: true,
	}

	for i := 2; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "--path" && i+1 < len(args):
			req.RestorePath = args[i+1]
			i++
		case arg == "--selective" && i+1 < len(args):
			paths := strings.Split(args[i+1], ",")
			req.SelectivePaths = paths
			i++
		case arg == "--overwrite":
			req.Overwrite = true
		case arg == "--merge":
			req.Merge = true
		case arg == "--dry-run":
			req.DryRun = true
		case arg == "--no-preserve-perms":
			req.PreservePerms = false
		case arg == "--no-preserve-owner":
			req.PreserveOwner = false
		case arg == "--no-verify":
			req.VerifyIntegrity = false
		case arg == "--wait":
			req.Wait = true
		case strings.HasPrefix(arg, "--path="):
			req.RestorePath = strings.TrimPrefix(arg, "--path=")
		case strings.HasPrefix(arg, "--selective="):
			paths := strings.Split(strings.TrimPrefix(arg, "--selective="), ",")
			req.SelectivePaths = paths
		default:
			return req, fmt.Errorf("unknown option: %s", arg)
		}
	}

	return req, nil
}

// displayRestoreStart displays restore operation start information
func (bc *BackupCommands) displayRestoreStart(req types.RestoreRequest) {
	if req.DryRun {
		fmt.Printf("🔍 Dry-run: Previewing restore operation...\n")
	} else {
		fmt.Printf("🔄 Restoring data from backup '%s' to workspace '%s'...\n", req.BackupName, req.TargetInstance)
	}

	if req.RestorePath != "/" {
		fmt.Printf("📁 Restore path: %s\n", req.RestorePath)
	}

	if len(req.SelectivePaths) > 0 {
		fmt.Printf("📂 Selective paths: %s\n", strings.Join(req.SelectivePaths, ", "))
	}

	bc.displayRestoreMode(req)
}

// displayRestoreMode displays the restore mode being used
func (bc *BackupCommands) displayRestoreMode(req types.RestoreRequest) {
	if req.Overwrite {
		fmt.Printf("⚠️  Overwrite mode: Existing files will be replaced\n")
	} else if req.Merge {
		fmt.Printf("🔄 Merge mode: Files will be merged with existing data\n")
	} else {
		fmt.Printf("🔒 Safe mode: Existing files will be preserved\n")
	}
}

// displayRestoreResult displays restore operation result
func (bc *BackupCommands) displayRestoreResult(req types.RestoreRequest, result *types.RestoreResult) {
	if req.DryRun {
		fmt.Printf("✅ Dry-run completed - no actual restore performed\n")
	} else {
		fmt.Printf("✅ Restore operation initiated\n")
	}

	fmt.Printf("   Restore ID: %s\n", result.RestoreID)
	fmt.Printf("   Backup: %s\n", result.BackupName)
	fmt.Printf("   Target Instance: %s\n", result.TargetInstance)
	fmt.Printf("   Restore Path: %s\n", result.RestorePath)
	fmt.Printf("   State: %s\n", strings.ToUpper(result.State))

	if !req.DryRun {
		fmt.Printf("   Estimated Completion: %d minutes\n", result.EstimatedCompletion)
		fmt.Printf("   Started At: %s\n", result.StartedAt.Format("2006-01-02 15:04:05"))

		if result.Message != "" {
			fmt.Printf("   Message: %s\n", result.Message)
		}
	}
}

// displayRestoreNextSteps displays next steps after restore initiation
func (bc *BackupCommands) displayRestoreNextSteps(req types.RestoreRequest, result *types.RestoreResult) {
	if !req.DryRun {
		fmt.Printf("\n💡 Check progress with: cws restore status %s\n", result.RestoreID)
		fmt.Printf("💡 View all operations: cws restore operations\n")
	}
}

// getRestoreStatus gets the status of a restore operation
func (bc *BackupCommands) getRestoreStatus(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: cws restore status <restore-id>")
	}

	restoreID := args[0]

	result, err := bc.app.apiClient.GetRestoreStatus(bc.app.ctx, restoreID)
	if err != nil {
		return WrapAPIError("get restore status", err)
	}

	fmt.Printf("🔄 Restore Operation: %s\n", result.RestoreID)
	fmt.Printf("   Backup: %s\n", result.BackupName)
	fmt.Printf("   Target Instance: %s\n", result.TargetInstance)
	fmt.Printf("   Restore Path: %s\n", result.RestorePath)
	fmt.Printf("   State: %s\n", strings.ToUpper(result.State))

	fmt.Printf("   Progress:\n")
	fmt.Printf("     Files Restored: %d\n", result.RestoredFileCount)
	fmt.Printf("     Data Restored: %s\n", bc.formatBytes(result.RestoredBytes))
	fmt.Printf("     Files Skipped: %d\n", result.SkippedFileCount)

	if result.ErrorCount > 0 {
		fmt.Printf("     Errors: %d\n", result.ErrorCount)
		if len(result.Errors) > 0 {
			fmt.Printf("     Recent Errors: %s\n", strings.Join(result.Errors[:minInt(3, len(result.Errors))], ", "))
		}
	}

	fmt.Printf("   Started: %s\n", result.StartedAt.Format("2006-01-02 15:04:05"))

	if result.CompletedAt != nil {
		fmt.Printf("   Completed: %s\n", result.CompletedAt.Format("2006-01-02 15:04:05"))
		duration := result.CompletedAt.Sub(result.StartedAt)
		fmt.Printf("   Duration: %s\n", bc.formatDuration(duration))
	} else {
		elapsed := time.Since(result.StartedAt)
		fmt.Printf("   Elapsed: %s\n", bc.formatDuration(elapsed))
		if result.EstimatedCompletion > 0 {
			fmt.Printf("   Estimated Remaining: %d minutes\n", result.EstimatedCompletion)
		}
	}

	if result.IntegrityVerified {
		fmt.Printf("   ✅ Integrity verified\n")
	}

	return nil
}

// listRestoreOperations lists all restore operations
func (bc *BackupCommands) listRestoreOperations(args []string) error {
	operations, err := bc.app.apiClient.ListRestoreOperations(bc.app.ctx)
	if err != nil {
		return WrapAPIError("list restore operations", err)
	}

	if len(operations) == 0 {
		fmt.Println("No restore operations found.")
		return nil
	}

	fmt.Printf("🔄 Restore Operations (%d total):\n\n", len(operations))

	w := tabwriter.NewWriter(os.Stdout, TabWriterMinWidth, TabWriterTabWidth, TabWriterPadding, TabWriterPadChar, TabWriterFlags)
	_, _ = fmt.Fprintln(w, "RESTORE ID\tBACKUP\tTARGET INSTANCE\tSTATE\tPROGRESS\tSTARTED")

	for _, op := range operations {
		progress := fmt.Sprintf("%d files", op.RestoredFileCount)
		if op.RestoredBytes > 0 {
			progress += fmt.Sprintf(" (%s)", bc.formatBytes(op.RestoredBytes))
		}

		restoreID := op.RestoreID
		if len(restoreID) > 12 {
			restoreID = restoreID[:12] + "..."
		}

		_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\n",
			restoreID,
			op.BackupName,
			op.TargetInstance,
			strings.ToUpper(op.State),
			progress,
			op.StartedAt.Format("2006-01-02 15:04"),
		)
	}

	_ = w.Flush()

	fmt.Printf("\n💡 Check operation status: cws restore status <restore-id>\n")

	return nil
}

// verifyRestore verifies a completed restore operation
func (bc *BackupCommands) verifyRestore(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: cws restore verify <restore-id>")
	}

	restoreID := args[0]

	// Get restore status and perform verification
	result, err := bc.app.apiClient.GetRestoreStatus(bc.app.ctx, restoreID)
	if err != nil {
		return WrapAPIError("verify restore", err)
	}

	fmt.Printf("🔍 Verifying restore operation: %s\n", result.RestoreID)
	fmt.Printf("   Backup: %s\n", result.BackupName)
	fmt.Printf("   Target Instance: %s\n", result.TargetInstance)
	fmt.Printf("   State: %s\n", strings.ToUpper(result.State))

	if result.State != "completed" {
		fmt.Printf("⚠️  Restore operation is not completed (state: %s)\n", result.State)
		return nil
	}

	fmt.Printf("✅ Restore verification summary:\n")
	fmt.Printf("   Files Restored: %d\n", result.RestoredFileCount)
	fmt.Printf("   Data Restored: %s\n", bc.formatBytes(result.RestoredBytes))
	fmt.Printf("   Files Skipped: %d\n", result.SkippedFileCount)
	fmt.Printf("   Errors: %d\n", result.ErrorCount)
	fmt.Printf("   Integrity Verified: %s\n", map[bool]string{true: "Yes", false: "No"}[result.IntegrityVerified])

	if result.ErrorCount > 0 && len(result.Errors) > 0 {
		fmt.Printf("\n⚠️  Errors encountered:\n")
		for i, err := range result.Errors[:min(5, len(result.Errors))] {
			fmt.Printf("   %d. %s\n", i+1, err)
		}
		if len(result.Errors) > 5 {
			fmt.Printf("   ... and %d more errors\n", len(result.Errors)-5)
		}
	}

	return nil
}

// Helper methods

// monitorBackupProgress monitors backup creation progress
func (bc *BackupCommands) monitorBackupProgress(backupID, backupName string) error {
	startTime := time.Now()
	maxDuration := 60 * time.Minute // Maximum monitoring time for backups

	fmt.Printf("📊 Monitoring backup creation: %s (%s)\n", backupName, backupID)

	for {
		elapsed := time.Since(startTime)

		// Check for timeout
		if elapsed > maxDuration {
			fmt.Printf("⚠️  Backup monitoring timeout (%s). Backup may still be creating.\n",
				bc.formatDuration(maxDuration))
			fmt.Printf("💡 Check status with: cws backup info %s\n", backupName)
			return nil
		}

		// Get current backup status
		backup, err := bc.app.apiClient.GetBackup(bc.app.ctx, backupName)
		if err != nil {
			fmt.Printf("⚠️  Unable to get backup status: %v\n", err)
			time.Sleep(10 * time.Second)
			continue
		}

		// Update progress display
		fmt.Printf("⏳ Backup state: %s (%s elapsed)\n", strings.ToUpper(backup.State), bc.formatDuration(elapsed))

		// Check for completion
		if backup.State == "available" {
			fmt.Printf("✅ Backup creation completed!\n")
			fmt.Printf("   Backup: %s\n", backup.BackupName)
			fmt.Printf("   Total Size: %s\n", bc.formatBytes(backup.SizeBytes))
			fmt.Printf("   File Count: %d\n", backup.FileCount)
			fmt.Printf("   Total Time: %s\n", bc.formatDuration(elapsed))
			fmt.Printf("   Monthly Cost: $%.2f\n", backup.StorageCostMonthly)
			return nil
		}

		// Check for error states
		if backup.State == "error" || backup.State == "failed" {
			fmt.Printf("❌ Backup creation failed\n")
			return fmt.Errorf("backup creation failed with state: %s", backup.State)
		}

		// Wait before next check
		time.Sleep(15 * time.Second)
	}
}

// monitorRestoreProgress monitors restore operation progress
func (bc *BackupCommands) monitorRestoreProgress(restoreID, backupName, targetInstance string) error {
	startTime := time.Now()
	maxDuration := 45 * time.Minute // Maximum monitoring time for restores

	fmt.Printf("📊 Monitoring restore operation: %s\n", restoreID)

	for {
		elapsed := time.Since(startTime)

		// Check for timeout
		if elapsed > maxDuration {
			fmt.Printf("⚠️  Restore monitoring timeout (%s). Restore may still be running.\n",
				bc.formatDuration(maxDuration))
			fmt.Printf("💡 Check status with: cws restore status %s\n", restoreID)
			return nil
		}

		// Get current restore status
		result, err := bc.app.apiClient.GetRestoreStatus(bc.app.ctx, restoreID)
		if err != nil {
			fmt.Printf("⚠️  Unable to get restore status: %v\n", err)
			time.Sleep(10 * time.Second)
			continue
		}

		// Update progress display
		progress := fmt.Sprintf("%d files (%s)", result.RestoredFileCount, bc.formatBytes(result.RestoredBytes))
		fmt.Printf("⏳ Restore progress: %s - %s (%s elapsed)\n",
			strings.ToUpper(result.State), progress, bc.formatDuration(elapsed))

		if result.ErrorCount > 0 {
			fmt.Printf("   ⚠️  %d errors encountered\n", result.ErrorCount)
		}

		// Check for completion
		if result.State == "completed" {
			fmt.Printf("✅ Restore operation completed!\n")
			fmt.Printf("   Files Restored: %d\n", result.RestoredFileCount)
			fmt.Printf("   Data Restored: %s\n", bc.formatBytes(result.RestoredBytes))
			if result.SkippedFileCount > 0 {
				fmt.Printf("   Files Skipped: %d\n", result.SkippedFileCount)
			}
			if result.ErrorCount > 0 {
				fmt.Printf("   Errors: %d\n", result.ErrorCount)
			}
			fmt.Printf("   Total Time: %s\n", bc.formatDuration(elapsed))
			fmt.Printf("   Integrity Verified: %s\n", map[bool]string{true: "Yes", false: "No"}[result.IntegrityVerified])
			return nil
		}

		// Check for error states
		if result.State == "error" || result.State == "failed" {
			fmt.Printf("❌ Restore operation failed\n")
			if len(result.Errors) > 0 {
				fmt.Printf("   Errors: %s\n", strings.Join(result.Errors[:minInt(3, len(result.Errors))], ", "))
			}
			return fmt.Errorf("restore operation failed with state: %s", result.State)
		}

		// Wait before next check
		time.Sleep(10 * time.Second)
	}
}

// formatBytes formats byte size into human readable format
func (bc *BackupCommands) formatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

// formatDuration formats duration for display
func (bc *BackupCommands) formatDuration(d time.Duration) string {
	hours := int(d.Hours())
	minutes := int(d.Minutes()) % 60
	seconds := int(d.Seconds()) % 60

	if hours > 0 {
		return fmt.Sprintf("%dh%dm%ds", hours, minutes, seconds)
	}
	if minutes > 0 {
		return fmt.Sprintf("%dm%ds", minutes, seconds)
	}
	return fmt.Sprintf("%ds", seconds)
}

// minInt returns the minimum of two integers
func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// showBackupUsage shows backup command usage
func (bc *BackupCommands) showBackupUsage() error {
	fmt.Print(bc.getBackupUsageText())
	return nil
}

// showRestoreUsage shows restore command usage
func (bc *BackupCommands) showRestoreUsage() error {
	fmt.Print(bc.getRestoreUsageText())
	return nil
}

// getBackupUsageText returns the backup usage text
func (bc *BackupCommands) getBackupUsageText() string {
	return `Usage: prism backup <action> [arguments]

Actions:
  create <workspace> <backup-name> [options]    Create a data backup from workspace
  list                                         List all data backups
  info <backup-name>                          Show detailed backup information
  delete <backup-name>                        Delete a backup
  contents <backup-name> [path]               List backup contents
  verify <backup-name> [options]              Verify backup integrity

Create Options:
  --description <text>         Add a description to the backup
  --include <paths>           Comma-separated paths to include (default: /home,/data,/opt/research)
  --exclude <paths>           Comma-separated paths to exclude (default: .cache,.tmp,*.log)
  --full                      Create full backup (default)
  --incremental              Create incremental backup
  --storage <type>           Storage type: s3, efs, ebs (default: s3)
  --no-encryption            Disable encryption (enabled by default)
  --wait                     Wait and monitor backup creation progress

Verify Options:
  --quick                    Quick verification (metadata only)
  --paths <paths>            Verify only specific paths

Contents Options:
  --recursive, -r            List contents recursively

Examples:
  cws backup create my-workspace daily-backup
  cws backup create gpu-training checkpoint-data --include /data,/results --incremental
  cws backup list
  cws backup info daily-backup
  cws backup contents daily-backup /home
  cws backup verify daily-backup --quick
  cws backup delete old-backup

Storage Options:
  • S3: Cost-effective, encrypted, cross-region replication
  • EFS: Fast access, shared across instances
  • EBS: High-performance, instance-local

Cost Information:
  Data backups are stored with compression and deduplication
  S3: ~$0.023/GB/month, EFS: ~$0.30/GB/month, EBS: ~$0.10/GB/month

💡 Data backups complement instance snapshots:
   • Backups: User data, configurations, research files (selective)
   • Snapshots: Complete system images (full instance state)

⚠️  Important Notes:
   • Backups are region-specific but can be replicated
   • Incremental backups depend on previous full backup
   • Encrypted backups use AES-256 encryption at rest and in transit
   • Backup names must be unique within your account
`
}

// getRestoreUsageText returns the restore usage text
func (bc *BackupCommands) getRestoreUsageText() string {
	return `Usage: prism restore <backup-name> <target-instance> [options]
       cws restore <action> [arguments]

Direct Restore:
  <backup-name> <target-instance>    Restore backup to workspace

Actions:
  list <backup-name> [path]          List backup contents (same as backup contents)
  verify <restore-id>                Verify completed restore operation
  status <restore-id>                Check restore operation status
  operations                         List all restore operations

Restore Options:
  --path <path>                      Target path for restore (default: /)
  --selective <paths>                Restore only specific paths (comma-separated)
  --overwrite                       Overwrite existing files
  --merge                           Merge with existing data (default: preserve existing)
  --dry-run                         Preview restore operation without executing
  --no-preserve-perms               Don't preserve file permissions
  --no-preserve-owner               Don't preserve file ownership
  --no-verify                       Skip integrity verification
  --wait                            Wait and monitor restore progress

Examples:
  cws restore daily-backup my-workspace
  cws restore checkpoint-data gpu-training --path /data --overwrite
  cws restore daily-backup my-workspace --selective /home/user,/data/project
  cws restore daily-backup my-workspace --dry-run
  cws restore operations
  cws restore status restore-123456
  cws restore verify restore-123456

Restore Modes:
  • Safe (default): Preserve existing files, restore only missing files
  • Merge: Merge backup data with existing data, newer files take precedence
  • Overwrite: Replace all existing files with backup data

Selective Restore:
  Use --selective to restore only specific files or directories:
  • --selective /home/user               Restore only user home directory
  • --selective /data/project,/config    Restore multiple specific paths
  • --path /restore --selective /data    Restore /data to /restore directory

Verification:
  • Automatic integrity verification during restore (disable with --no-verify)
  • Post-restore verification with 'cws restore verify <restore-id>'
  • Checksum validation for data integrity assurance

💡 Restoration Features:
   • Cross-instance restore: Restore backup to any instance
   • Point-in-time recovery: Restore from any available backup
   • Selective restore: Choose specific files/directories
   • Integrity verification: Ensure data consistency
   • Progress monitoring: Real-time restore status

⚠️  Important Notes:
   • Target instance must be running and accessible
   • Sufficient storage space required on target instance
   • File permissions preserved by default (requires appropriate user access)
   • Large restores may take significant time depending on data size
   • Dry-run recommended for large or complex restore operations
`
}
