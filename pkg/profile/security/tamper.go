// Package security provides tamper detection and file integrity monitoring
package security

import (
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// TamperProtection provides file integrity monitoring and tamper detection
type TamperProtection struct {
	checksums map[string]string // File path -> SHA-256 checksum
	metadata  map[string]*FileMetadata // File path -> metadata
	mutex     sync.RWMutex // Thread-safe access
}

// FileMetadata stores file integrity information
type FileMetadata struct {
	Path         string    `json:"path"`
	Checksum     string    `json:"checksum"`
	Size         int64     `json:"size"`
	ModTime      time.Time `json:"mod_time"`
	Protected    bool      `json:"protected"`
	CreatedAt    time.Time `json:"created_at"`
	LastChecked  time.Time `json:"last_checked"`
}

// NewTamperProtection creates a new tamper protection instance
func NewTamperProtection() *TamperProtection {
	return &TamperProtection{
		checksums: make(map[string]string),
		metadata:  make(map[string]*FileMetadata),
	}
}

// ProtectFile adds a file to tamper detection monitoring
func (t *TamperProtection) ProtectFile(filePath string) error {
	t.mutex.Lock()
	defer t.mutex.Unlock()

	// Get absolute path
	absPath, err := filepath.Abs(filePath)
	if err != nil {
		return fmt.Errorf("failed to get absolute path: %w", err)
	}

	// Calculate file checksum and metadata
	metadata, err := t.calculateFileMetadata(absPath)
	if err != nil {
		return fmt.Errorf("failed to calculate file metadata: %w", err)
	}

	// Store checksum and metadata
	t.checksums[absPath] = metadata.Checksum
	t.metadata[absPath] = metadata

	// Set file permissions to read-only for additional protection
	if err := os.Chmod(absPath, 0600); err != nil {
		// Non-fatal, log but continue
		fmt.Printf("Warning: Failed to set protective permissions on %s: %v\n", absPath, err)
	}

	return nil
}

// ValidateIntegrity checks if a protected file has been tampered with
func (t *TamperProtection) ValidateIntegrity(filePath string) error {
	t.mutex.RLock()
	defer t.mutex.RUnlock()

	// Get absolute path
	absPath, err := filepath.Abs(filePath)
	if err != nil {
		return fmt.Errorf("failed to get absolute path: %w", err)
	}

	// Check if file is protected
	expectedMetadata, exists := t.metadata[absPath]
	if !exists {
		return fmt.Errorf("file %s is not under tamper protection", filePath)
	}

	// Calculate current file metadata
	currentMetadata, err := t.calculateFileMetadata(absPath)
	if err != nil {
		return &TamperDetectionError{
			FilePath:  absPath,
			Operation: "integrity_check",
			Err:       fmt.Errorf("failed to read current file state: %w", err),
		}
	}

	// Compare checksums (primary integrity check)
	if expectedMetadata.Checksum != currentMetadata.Checksum {
		return &TamperDetectionError{
			FilePath:         absPath,
			Operation:        "checksum_verification",
			ExpectedChecksum: expectedMetadata.Checksum,
			ActualChecksum:   currentMetadata.Checksum,
			Err:              fmt.Errorf("file integrity violation detected"),
		}
	}

	// Compare file size (secondary check)
	if expectedMetadata.Size != currentMetadata.Size {
		return &TamperDetectionError{
			FilePath:  absPath,
			Operation: "size_verification",
			Err:       fmt.Errorf("file size changed from %d to %d bytes", expectedMetadata.Size, currentMetadata.Size),
		}
	}

	// Update last checked timestamp
	expectedMetadata.LastChecked = time.Now()

	return nil
}

// ValidateAllFiles checks integrity of all protected files
func (t *TamperProtection) ValidateAllFiles() []error {
	t.mutex.RLock()
	filePaths := make([]string, 0, len(t.metadata))
	for path := range t.metadata {
		filePaths = append(filePaths, path)
	}
	t.mutex.RUnlock()

	var errors []error
	for _, path := range filePaths {
		if err := t.ValidateIntegrity(path); err != nil {
			errors = append(errors, err)
		}
	}

	return errors
}

// RemoveProtection removes a file from tamper detection monitoring
func (t *TamperProtection) RemoveProtection(filePath string) error {
	t.mutex.Lock()
	defer t.mutex.Unlock()

	// Get absolute path
	absPath, err := filepath.Abs(filePath)
	if err != nil {
		return fmt.Errorf("failed to get absolute path: %w", err)
	}

	// Remove from tracking
	delete(t.checksums, absPath)
	delete(t.metadata, absPath)

	return nil
}

// GetProtectedFiles returns a list of all files under protection
func (t *TamperProtection) GetProtectedFiles() []*FileMetadata {
	t.mutex.RLock()
	defer t.mutex.RUnlock()

	files := make([]*FileMetadata, 0, len(t.metadata))
	for _, metadata := range t.metadata {
		// Return a copy to prevent external modification
		metadataCopy := *metadata
		files = append(files, &metadataCopy)
	}

	return files
}

// UpdateProtection recalculates protection metadata for a file
func (t *TamperProtection) UpdateProtection(filePath string) error {
	t.mutex.Lock()
	defer t.mutex.Unlock()

	// Get absolute path
	absPath, err := filepath.Abs(filePath)
	if err != nil {
		return fmt.Errorf("failed to get absolute path: %w", err)
	}

	// Check if file is currently protected
	if _, exists := t.metadata[absPath]; !exists {
		return fmt.Errorf("file %s is not currently protected", filePath)
	}

	// Recalculate metadata
	metadata, err := t.calculateFileMetadata(absPath)
	if err != nil {
		return fmt.Errorf("failed to recalculate file metadata: %w", err)
	}

	// Update tracking
	t.checksums[absPath] = metadata.Checksum
	t.metadata[absPath] = metadata

	return nil
}

// calculateFileMetadata computes SHA-256 checksum and file metadata
func (t *TamperProtection) calculateFileMetadata(filePath string) (*FileMetadata, error) {
	// Get file info
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to get file info: %w", err)
	}

	// Open file for reading
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	// Calculate SHA-256 checksum
	hasher := sha256.New()
	if _, err := io.Copy(hasher, file); err != nil {
		return nil, fmt.Errorf("failed to calculate checksum: %w", err)
	}

	checksum := fmt.Sprintf("%x", hasher.Sum(nil))

	return &FileMetadata{
		Path:        filePath,
		Checksum:    checksum,
		Size:        fileInfo.Size(),
		ModTime:     fileInfo.ModTime(),
		Protected:   true,
		CreatedAt:   time.Now(),
		LastChecked: time.Now(),
	}, nil
}

// TamperDetectionError represents tamper detection related errors
type TamperDetectionError struct {
	FilePath         string
	Operation        string
	ExpectedChecksum string
	ActualChecksum   string
	Err              error
}

func (e *TamperDetectionError) Error() string {
	if e.ExpectedChecksum != "" && e.ActualChecksum != "" {
		return fmt.Sprintf("tamper detection %s failed for %s: expected checksum %s, got %s: %v",
			e.Operation, e.FilePath, e.ExpectedChecksum[:8], e.ActualChecksum[:8], e.Err)
	}
	return fmt.Sprintf("tamper detection %s failed for %s: %v", e.Operation, e.FilePath, e.Err)
}

func (e *TamperDetectionError) Unwrap() error {
	return e.Err
}

// Common tamper detection errors
var (
	ErrFileNotProtected = &TamperDetectionError{Operation: "protection_check", Err: fmt.Errorf("file not under protection")}
	ErrIntegrityViolation = &TamperDetectionError{Operation: "integrity_check", Err: fmt.Errorf("file integrity violation")}
)

// ProtectSecurityFiles applies tamper protection to critical security files
func ProtectSecurityFiles() (*TamperProtection, error) {
	protection := NewTamperProtection()

	// Get CloudWorkstation directory
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}

	secureDir := filepath.Join(homeDir, ".cloudworkstation", "secure")

	// Find all .bin files in secure directory (encrypted binding files)
	err = filepath.Walk(secureDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip files we can't access
		}

		// Protect all .bin files (encrypted device bindings)
		if filepath.Ext(path) == ".bin" {
			if err := protection.ProtectFile(path); err != nil {
				fmt.Printf("Warning: Failed to protect file %s: %v\n", path, err)
			}
		}

		return nil
	})

	if err != nil {
		fmt.Printf("Warning: Failed to walk secure directory: %v\n", err)
	}

	// Protect profiles.json if it exists
	profilesPath := filepath.Join(homeDir, ".cloudworkstation", "profiles.json")
	if _, err := os.Stat(profilesPath); err == nil {
		if err := protection.ProtectFile(profilesPath); err != nil {
			fmt.Printf("Warning: Failed to protect profiles.json: %v\n", err)
		}
	}

	return protection, nil
}

// ValidateSystemIntegrity performs a comprehensive integrity check
func ValidateSystemIntegrity() error {
	protection, err := ProtectSecurityFiles()
	if err != nil {
		return fmt.Errorf("failed to initialize tamper protection: %w", err)
	}

	errors := protection.ValidateAllFiles()
	if len(errors) > 0 {
		// Return the first critical error
		return errors[0]
	}

	return nil
}

// InitializeSecuritySystem sets up tamper protection for all security files
func InitializeSecuritySystem() (*TamperProtection, error) {
	// Create tamper protection and protect critical files
	protection, err := ProtectSecurityFiles()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize security system: %w", err)
	}

	// Perform initial integrity validation
	if err := ValidateSystemIntegrity(); err != nil {
		// Log warning but don't fail initialization (files might be new)
		fmt.Printf("Warning: Security integrity check found issues: %v\n", err)
	}

	return protection, nil
}

// PeriodicIntegrityCheck performs regular integrity validation
func PeriodicIntegrityCheck(protection *TamperProtection) {
	if protection == nil {
		return
	}

	errors := protection.ValidateAllFiles()
	if len(errors) > 0 {
		fmt.Printf("SECURITY ALERT: File integrity violations detected:\n")
		for _, err := range errors {
			fmt.Printf("  - %v\n", err)
		}
	}
}