// Package export provides functionality for exporting and importing CloudWorkstation profiles.
package export

import (
	"archive/zip"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/scttfrdmn/cloudworkstation/pkg/profile"
)

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
	tempDir, err := os.MkdirTemp("", "cws-profile-export")
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
	if err := os.WriteFile(profilesFile, profilesJSON, 0644); err != nil {
		return fmt.Errorf("failed to write profiles file: %w", err)
	}

	// Export credentials if requested
	if options.IncludeCredentials {
		credsDir := filepath.Join(tempDir, "credentials")
		if err := os.MkdirAll(credsDir, 0755); err != nil {
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

			// Marshal credentials
			credsJSON, err := json.MarshalIndent(creds, "", "  ")
			if err != nil {
				continue
			}

			// Write credentials file
			credsFile := filepath.Join(credsDir, fmt.Sprintf("%s.json", prof.AWSProfile))
			if err := os.WriteFile(credsFile, credsJSON, 0644); err != nil {
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
	tempDir, err := os.MkdirTemp("", "cws-profile-import")
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
	profilesJSON, err := os.ReadFile(profilesFile)
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

	credsFile := filepath.Join(tempDir, "credentials", fmt.Sprintf("%s.json", originalProfileID))
	if _, err := os.Stat(credsFile); err != nil {
		return
	}

	credsJSON, err := os.ReadFile(credsFile)
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

// createZipArchive creates a zip archive from a directory
func createZipArchive(sourceDir, outputPath, password string) error {
	// Create output file
	outFile, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer outFile.Close()

	// Create zip writer
	zipWriter := zip.NewWriter(outFile)
	defer zipWriter.Close()

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
		srcFile, err := os.Open(path)
		if err != nil {
			return fmt.Errorf("failed to open source file: %w", err)
		}
		defer srcFile.Close()

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
	defer reader.Close()

	// Extract files
	for _, file := range reader.File {
		// Skip directories
		if file.FileInfo().IsDir() {
			continue
		}

		// Create output file path
		outPath := filepath.Join(outputDir, file.Name)

		// Create parent directory if needed
		if err := os.MkdirAll(filepath.Dir(outPath), 0755); err != nil {
			return fmt.Errorf("failed to create directory: %w", err)
		}

		// Open source file in zip
		srcFile, err := file.Open()
		if err != nil {
			return fmt.Errorf("failed to open file in zip: %w", err)
		}

		// Create output file
		outFile, err := os.Create(outPath)
		if err != nil {
			_ = srcFile.Close()
			return fmt.Errorf("failed to create output file: %w", err)
		}

		// Copy content
		_, err = io.Copy(outFile, srcFile)
		_ = srcFile.Close()
		_ = outFile.Close()

		if err != nil {
			return fmt.Errorf("failed to copy file content: %w", err)
		}
	}

	return nil
}

// copyFile copies a file from src to dst
func copyFile(src, dst string) error {
	// Open source file
	srcFile, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("failed to open source file: %w", err)
	}
	defer srcFile.Close()

	// Create destination file
	dstFile, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("failed to create destination file: %w", err)
	}
	defer dstFile.Close()

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
