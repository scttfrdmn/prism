// Package export provides functionality for exporting and importing Prism profiles.
package export

import (
	"archive/zip"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/scttfrdmn/prism/pkg/profile"
)

// maxExtractedFileSize caps the size of any single file extracted from an
// imported profile archive, guarding against decompression bombs. Profile
// exports contain only small JSON documents.
const maxExtractedFileSize = 64 << 20 // 64 MiB

// ExportFormat defines the format version of exported profiles
type ExportFormat struct {
	Version    string            `json:"version"`
	ExportedAt time.Time         `json:"exported_at"`
	Profiles   []profile.Profile `json:"profiles"`
	Metadata   map[string]string `json:"metadata"`
}

// ExportOptions defines options for profile export
type ExportOptions struct {
	IncludeCredentials bool
	IncludeInvitations bool
	Password           string
	Format             string
}

// DefaultExportOptions returns default export options
func DefaultExportOptions() ExportOptions {
	return ExportOptions{
		IncludeCredentials: false,
		IncludeInvitations: true,
		Password:           "",
		Format:             "zip",
	}
}

// ExportProfiles exports profiles to a file
func ExportProfiles(profileManager *profile.ManagerEnhanced, profiles []profile.Profile, outputPath string, options ExportOptions) error {
	// Create export format
	exportData := ExportFormat{
		Version:    "1.0.0",
		ExportedAt: time.Now(),
		Profiles:   profiles,
		Metadata: map[string]string{
			"app_version": "0.4.2",
			"platform":    detectPlatform(),
		},
	}

	// Create temporary directory for export files
	tempDir, err := os.MkdirTemp("", "prism-profile-export")
	if err != nil {
		return fmt.Errorf("failed to create temporary directory: %w", err)
	}
	defer func() { _ = os.RemoveAll(tempDir) }()

	// Create profiles JSON file
	profilesJSON, err := json.MarshalIndent(exportData, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal profiles: %w", err)
	}

	profilesFile := filepath.Join(tempDir, "profiles.json")
	if err := os.WriteFile(profilesFile, profilesJSON, 0600); err != nil {
		return fmt.Errorf("failed to write profiles file: %w", err)
	}

	// Export credentials if requested
	if options.IncludeCredentials {
		credsDir := filepath.Join(tempDir, "credentials")
		if err := os.MkdirAll(credsDir, 0700); err != nil {
			return fmt.Errorf("failed to create credentials directory: %w", err)
		}

		// Export credentials for each profile
		for _, prof := range profiles {
			// Skip invitation profiles
			if prof.Type == "invitation" {
				continue
			}

			// Get credentials
			creds, err := profileManager.GetProfileCredentials(prof.AWSProfile)
			if err != nil {
				// Skip profiles without credentials
				continue
			}

			// Marshal credentials (exporting secrets is the explicit purpose of IncludeCredentials)
			credsJSON, err := json.MarshalIndent(creds, "", "  ") //nolint:gosec // G117: credentials export is opt-in via IncludeCredentials; written 0600
			if err != nil {
				continue
			}

			// Write credentials file
			credsFile := filepath.Join(credsDir, fmt.Sprintf("%s.json", prof.AWSProfile))
			if err := os.WriteFile(credsFile, credsJSON, 0600); err != nil {
				continue
			}
		}
	}

	// Create zip archive if format is zip
	if options.Format == "zip" {
		if err := createZipArchive(tempDir, outputPath, options.Password); err != nil {
			return fmt.Errorf("failed to create zip archive: %w", err)
		}
	} else {
		// For JSON format, just copy the profiles.json file
		if err := copyFile(profilesFile, outputPath); err != nil {
			return fmt.Errorf("failed to copy profiles file: %w", err)
		}
	}

	return nil
}

// ImportProfiles imports profiles from a file
func ImportProfiles(profileManager *profile.ManagerEnhanced, inputPath string, options ImportOptions) (*ImportResult, error) {
	result := &ImportResult{
		Success:          true,
		ProfilesImported: 0,
		FailedProfiles:   make(map[string]string),
	}

	tempDir, err := prepareImportDirectory(inputPath, options)
	if err != nil {
		return nil, err
	}
	defer func() { _ = os.RemoveAll(tempDir) }()

	exportData, err := loadExportData(tempDir)
	if err != nil {
		return nil, err
	}

	if err := validateExportVersion(&exportData, result); err != nil {
		return result, nil
	}

	processProfiles(profileManager, &exportData, tempDir, options, result)

	finalizeImportResult(result)

	return result, nil
}

func prepareImportDirectory(inputPath string, options ImportOptions) (string, error) {
	tempDir, err := os.MkdirTemp("", "prism-profile-import")
	if err != nil {
		return "", fmt.Errorf("failed to create temporary directory: %w", err)
	}

	if filepath.Ext(inputPath) == ".zip" {
		if err := extractZipArchive(inputPath, tempDir, options.Password); err != nil {
			return "", fmt.Errorf("failed to extract zip archive: %w", err)
		}
	} else {
		if err := copyFile(inputPath, filepath.Join(tempDir, "profiles.json")); err != nil {
			return "", fmt.Errorf("failed to copy profiles file: %w", err)
		}
	}

	return tempDir, nil
}

func loadExportData(tempDir string) (ExportFormat, error) {
	profilesFile := filepath.Join(tempDir, "profiles.json")
	profilesJSON, err := os.ReadFile(profilesFile) //nolint:gosec // G304: fixed file name inside our own MkdirTemp directory
	if err != nil {
		return ExportFormat{}, fmt.Errorf("failed to read profiles file: %w", err)
	}

	var exportData ExportFormat
	if err := json.Unmarshal(profilesJSON, &exportData); err != nil {
		return ExportFormat{}, fmt.Errorf("invalid profiles format: %w", err)
	}

	return exportData, nil
}

func validateExportVersion(exportData *ExportFormat, result *ImportResult) error {
	if exportData.Version != "1.0.0" {
		result.Success = false
		result.Error = fmt.Sprintf("unsupported export version: %s", exportData.Version)
		return fmt.Errorf("unsupported version")
	}
	return nil
}

func processProfiles(profileManager *profile.ManagerEnhanced, exportData *ExportFormat, tempDir string, options ImportOptions, result *ImportResult) {
	for _, prof := range exportData.Profiles {
		if !shouldImportProfile(&prof, options) {
			continue
		}

		profileID := handleProfileNameConflicts(&prof, profileManager, options)
		importProfileCredentials(profileManager, prof.AWSProfile, profileID, tempDir, options)

		if err := profileManager.AddProfile(prof); err != nil {
			result.FailedProfiles[prof.Name] = err.Error()
			continue
		}

		result.ProfilesImported++
	}
}

func handleProfileNameConflicts(prof *profile.Profile, profileManager *profile.ManagerEnhanced, options ImportOptions) string {
	profileID := prof.AWSProfile
	if options.ImportMode == ImportModeRename && profileManager.ProfileExists(profileID) {
		profileID = fmt.Sprintf("%s-imported-%d", profileID, time.Now().Unix())
		prof.AWSProfile = profileID
	}
	return profileID
}

func importProfileCredentials(profileManager *profile.ManagerEnhanced, originalProfileID, profileID, tempDir string, options ImportOptions) {
	if !options.ImportCredentials {
		return
	}

	// The profile ID comes from the imported (untrusted) archive; reject IDs
	// that would resolve outside the extracted credentials directory.
	credsDir := filepath.Join(tempDir, "credentials")
	credsFile := filepath.Join(credsDir, fmt.Sprintf("%s.json", originalProfileID))
	if !isWithinDir(credsDir, credsFile) {
		return
	}
	if _, err := os.Stat(credsFile); err != nil {
		return
	}

	credsJSON, err := os.ReadFile(credsFile) //nolint:gosec // G304: path validated to stay within the extracted credentials directory
	if err != nil {
		return
	}

	var creds profile.Credentials
	if err := json.Unmarshal(credsJSON, &creds); err != nil {
		return
	}

	_ = profileManager.StoreProfileCredentials(profileID, &creds)
}

func finalizeImportResult(result *ImportResult) {
	if result.ProfilesImported == 0 && len(result.FailedProfiles) > 0 {
		result.Success = false
		result.Error = "failed to import any profiles"
	}
}

// ImportOptions defines options for profile import
type ImportOptions struct {
	ImportMode        ImportMode
	ProfileFilter     []string
	ImportCredentials bool
	Password          string
}

// ImportMode defines how to handle conflicting profiles during import
type ImportMode string

const (
	// ImportModeSkip skips profiles that already exist
	ImportModeSkip ImportMode = "skip"
	// ImportModeOverwrite overwrites existing profiles
	ImportModeOverwrite ImportMode = "overwrite"
	// ImportModeRename renames imported profiles to avoid conflicts
	ImportModeRename ImportMode = "rename"
)

// DefaultImportOptions returns default import options
func DefaultImportOptions() ImportOptions {
	return ImportOptions{
		ImportMode:        ImportModeRename,
		ProfileFilter:     []string{},
		ImportCredentials: false,
		Password:          "",
	}
}

// ImportResult contains the results of a profile import operation
type ImportResult struct {
	Success          bool
	ProfilesImported int
	FailedProfiles   map[string]string
	Error            string
}

// Helper functions

// shouldImportProfile determines if a profile should be imported based on filters
func shouldImportProfile(profile *profile.Profile, options ImportOptions) bool {
	// If no filter is set, import all profiles
	if len(options.ProfileFilter) == 0 {
		return true
	}

	// Check if profile is in filter
	for _, filter := range options.ProfileFilter {
		if filter == profile.Name || filter == profile.AWSProfile {
			return true
		}
	}

	return false
}

// isWithinDir reports whether target, after cleaning, is located inside baseDir.
func isWithinDir(baseDir, target string) bool {
	rel, err := filepath.Rel(filepath.Clean(baseDir), filepath.Clean(target))
	if err != nil {
		return false
	}
	return rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) && !filepath.IsAbs(rel)
}

// createZipArchive creates a zip archive from a directory
func createZipArchive(sourceDir, outputPath, password string) (retErr error) {
	// Create output file. The archive may contain credentials, so restrict it to the owner.
	outFile, err := os.OpenFile(outputPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0600) //nolint:gosec // G304: output path chosen by the local user (CLI argument)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer func() {
		if cerr := outFile.Close(); cerr != nil && retErr == nil {
			retErr = fmt.Errorf("failed to close output file: %w", cerr)
		}
	}()

	// Create zip writer
	zipWriter := zip.NewWriter(outFile)
	defer func() {
		if cerr := zipWriter.Close(); cerr != nil && retErr == nil {
			retErr = fmt.Errorf("failed to finalize zip archive: %w", cerr)
		}
	}()

	// Walk through the source directory
	return filepath.Walk(sourceDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories
		if info.IsDir() {
			return nil
		}

		// Calculate relative path
		relPath, err := filepath.Rel(sourceDir, path)
		if err != nil {
			return fmt.Errorf("failed to get relative path: %w", err)
		}

		// Create file in zip
		zipFile, err := zipWriter.Create(relPath)
		if err != nil {
			return fmt.Errorf("failed to create file in zip: %w", err)
		}

		// Open source file
		srcFile, err := os.Open(path) //nolint:gosec // G304: path produced by filepath.Walk over our own temp directory
		if err != nil {
			return fmt.Errorf("failed to open source file: %w", err)
		}
		defer func() { _ = srcFile.Close() }() // read-only; close error is not actionable

		// Copy file content
		_, err = io.Copy(zipFile, srcFile)
		if err != nil {
			return fmt.Errorf("failed to copy file content: %w", err)
		}

		return nil
	})
}

// extractZipArchive extracts a zip archive to a directory
func extractZipArchive(zipPath, outputDir, password string) error {
	// Open zip file
	reader, err := zip.OpenReader(zipPath)
	if err != nil {
		return fmt.Errorf("failed to open zip file: %w", err)
	}
	defer func() { _ = reader.Close() }() // read-only; close error is not actionable

	// Extract files
	for _, file := range reader.File {
		// Skip directories
		if file.FileInfo().IsDir() {
			continue
		}

		// Create output file path, rejecting entries that escape outputDir (zip slip)
		outPath := filepath.Join(outputDir, file.Name) //nolint:gosec // G305: validated by isWithinDir below
		if !isWithinDir(outputDir, outPath) {
			return fmt.Errorf("invalid file path in archive: %s", file.Name)
		}

		// Create parent directory if needed
		if err := os.MkdirAll(filepath.Dir(outPath), 0700); err != nil {
			return fmt.Errorf("failed to create directory: %w", err)
		}

		// Open source file in zip
		srcFile, err := file.Open()
		if err != nil {
			return fmt.Errorf("failed to open file in zip: %w", err)
		}

		// Create output file
		outFile, err := os.OpenFile(outPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0600) //nolint:gosec // G304: path validated to stay within outputDir
		if err != nil {
			_ = srcFile.Close()
			return fmt.Errorf("failed to create output file: %w", err)
		}

		// Copy content, capped to guard against decompression bombs
		n, err := io.Copy(outFile, io.LimitReader(srcFile, maxExtractedFileSize+1))
		_ = srcFile.Close() // read-only zip entry
		closeErr := outFile.Close()

		if err != nil {
			return fmt.Errorf("failed to copy file content: %w", err)
		}
		if n > maxExtractedFileSize {
			return fmt.Errorf("file %s in archive exceeds maximum size of %d bytes", file.Name, maxExtractedFileSize)
		}
		if closeErr != nil {
			return fmt.Errorf("failed to close output file: %w", closeErr)
		}
	}

	return nil
}

// copyFile copies a file from src to dst
func copyFile(src, dst string) (retErr error) {
	// Open source file
	srcFile, err := os.Open(src) //nolint:gosec // G304: path is a local CLI argument or our own temp file
	if err != nil {
		return fmt.Errorf("failed to open source file: %w", err)
	}
	defer func() { _ = srcFile.Close() }() // read-only; close error is not actionable

	// Create destination file
	dstFile, err := os.Create(dst) //nolint:gosec // G304: path is a local CLI argument or our own temp file
	if err != nil {
		return fmt.Errorf("failed to create destination file: %w", err)
	}
	defer func() {
		if cerr := dstFile.Close(); cerr != nil && retErr == nil {
			retErr = fmt.Errorf("failed to close destination file: %w", cerr)
		}
	}()

	// Copy content
	_, err = io.Copy(dstFile, srcFile)
	if err != nil {
		return fmt.Errorf("failed to copy file content: %w", err)
	}

	return nil
}

// detectPlatform detects the current platform
func detectPlatform() string {
	// Simple platform detection
	if os.PathSeparator == '\\' {
		return "windows"
	} else if _, err := os.Stat("/Applications"); err == nil {
		return "macos"
	} else {
		return "linux"
	}
}
