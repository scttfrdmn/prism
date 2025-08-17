// Package security provides secure storage and verification for CloudWorkstation profiles.
package security

import (
	"errors"
	"fmt"
	"os"
	"runtime"
	"strings"
	"sync"
	"time"
)

// Common errors
var (
	ErrKeychainUnsupported = errors.New("keychain not supported on this platform")
	ErrKeychainNotFound    = errors.New("keychain entry not found")
	ErrKeychainAccess      = errors.New("access to keychain denied")
)

// BindingMaterial represents the secure data stored in keychain
type BindingMaterial struct {
	DeviceID        string    `json:"device_id"`
	ProfileID       string    `json:"profile_id"`
	InvitationToken string    `json:"invitation_token"`
	Created         time.Time `json:"created"`
	LastValidated   time.Time `json:"last_validated"`
}

// KeychainProvider defines the interface for secure storage systems
type KeychainProvider interface {
	// Store saves data in the secure storage
	Store(key string, data []byte) error

	// Retrieve gets data from the secure storage
	Retrieve(key string) ([]byte, error)

	// Exists checks if a key exists in the secure storage
	Exists(key string) bool

	// Delete removes data from the secure storage
	Delete(key string) error
}

// Global cached keychain provider to avoid multiple initialization prompts
var (
	globalProvider KeychainProvider
	initError      error
	initOnce       sync.Once
)

// isDevelopmentMode detects if we're running in development/test mode
// to reduce keychain password prompts during development
func isDevelopmentMode() bool {
	// Check for explicit development environment indicators
	if os.Getenv("GO_ENV") == "test" || os.Getenv("CLOUDWORKSTATION_DEV") == "true" {
		return true
	}

	// Check for testing context
	if os.Getenv("TESTING") == "1" {
		return true
	}

	// Check if running from test or temporary directories
	if executable, err := os.Executable(); err == nil {
		execPath := executable
		if strings.Contains(execPath, "/tmp/") ||
			strings.Contains(execPath, "test") ||
			strings.Contains(execPath, "___go_build_") ||
			strings.HasSuffix(execPath, ".test") {
			return true
		}
	}

	return false
}

// initializeGlobalProvider initializes the keychain provider once
func initializeGlobalProvider() {
	initOnce.Do(func() {
		switch runtime.GOOS {
		case "darwin":
			globalProvider, initError = initializeMacOSKeychain()
		case "windows":
			native, err := NewWindowsCredentialManagerNative()
			if err != nil {
				fmt.Fprintf(os.Stderr, "Warning: Failed to initialize native Windows Credential Manager, using secure file storage: %v\n", err)
				globalProvider, initError = NewFileSecureStorage()
			} else {
				globalProvider, initError = native, nil
			}
		case "linux":
			native, err := NewLinuxSecretServiceNative()
			if err != nil {
				fmt.Fprintf(os.Stderr, "Warning: Failed to initialize native Linux Secret Service, using secure file storage: %v\n", err)
				globalProvider, initError = NewFileSecureStorage()
			} else {
				globalProvider, initError = native, nil
			}
		default:
			// Fallback to file-based storage with warning
			fmt.Fprintf(os.Stderr, "Warning: Using fallback secure storage on platform: %s\n", runtime.GOOS)
			globalProvider, initError = NewFileSecureStorage()
		}
	})
}

// NewKeychainProvider returns the global keychain provider instance
// Initializes on first call, then returns cached instance
func NewKeychainProvider() (KeychainProvider, error) {
	initializeGlobalProvider()
	return globalProvider, initError
}

// MacOSKeychain implements KeychainProvider for macOS
type MacOSKeychain struct {
	// Fields needed for macOS keychain operations
}

// NewMacOSKeychain creates a new macOS keychain provider (deprecated - use NewKeychainProvider)
func NewMacOSKeychain() (KeychainProvider, error) {
	return NewKeychainProvider()
}

// WindowsCredentialManager implements KeychainProvider for Windows
type WindowsCredentialManager struct {
	// Fields needed for Windows credential operations
}

// NewWindowsCredentialManager creates a new Windows credential manager provider (deprecated - use NewKeychainProvider)
func NewWindowsCredentialManager() (KeychainProvider, error) {
	return NewKeychainProvider()
}

// LinuxSecretService implements KeychainProvider for Linux
type LinuxSecretService struct {
	// Fields needed for Secret Service operations
}

// NewLinuxSecretService creates a new Linux Secret Service provider (deprecated - use NewKeychainProvider)
func NewLinuxSecretService() (KeychainProvider, error) {
	return NewKeychainProvider()
}

// FileSecureStorage is a fallback implementation using encrypted files
type FileSecureStorage struct {
	// Base directory for secure storage
	baseDir string
	// Crypto provider for encryption
	crypto *CryptoProvider
	// Tamper protection for file integrity
	tamperProtection *TamperProtection
}

// NewFileSecureStorage creates a new file-based secure storage provider
func NewFileSecureStorage() (*FileSecureStorage, error) {
	// Determine base directory
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}

	baseDir := fmt.Sprintf("%s/.cloudworkstation/secure", homeDir)

	// Create directory if it doesn't exist with restrictive permissions
	if err := os.MkdirAll(baseDir, 0700); err != nil { // Owner only
		return nil, fmt.Errorf("failed to create secure storage directory: %w", err)
	}

	// Initialize crypto provider
	crypto, err := NewCryptoProvider()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize crypto provider: %w", err)
	}

	// Initialize tamper protection
	tamperProtection := NewTamperProtection()

	return &FileSecureStorage{
		baseDir:          baseDir,
		crypto:           crypto,
		tamperProtection: tamperProtection,
	}, nil
}

// Store implements KeychainProvider.Store for file-based storage
func (f *FileSecureStorage) Store(key string, data []byte) error {
	// Encrypt the data using AES-256-GCM
	encryptedData, err := f.crypto.Encrypt(data)
	if err != nil {
		return fmt.Errorf("failed to encrypt data: %w", err)
	}

	// Write to file with restrictive permissions
	filePath := f.getFilePath(key)
	if err := os.WriteFile(filePath, encryptedData, 0600); err != nil { // Owner read/write only
		return fmt.Errorf("failed to write encrypted file: %w", err)
	}

	// Add tamper protection to the newly created file
	if err := f.tamperProtection.ProtectFile(filePath); err != nil {
		// Non-fatal error, log but continue
		fmt.Fprintf(os.Stderr, "Warning: Failed to add tamper protection to %s: %v\n", filePath, err)
	}

	return nil
}

// Retrieve implements KeychainProvider.Retrieve for file-based storage
func (f *FileSecureStorage) Retrieve(key string) ([]byte, error) {
	filePath := f.getFilePath(key)

	// Check if file exists
	if !f.Exists(key) {
		return nil, ErrKeychainNotFound
	}

	// Validate file integrity before reading
	if err := f.tamperProtection.ValidateIntegrity(filePath); err != nil {
		return nil, fmt.Errorf("file integrity violation detected: %w", err)
	}

	// Read encrypted data
	encryptedData, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read secure file: %w", err)
	}

	// Decrypt data using AES-256-GCM
	plaintext, err := f.crypto.Decrypt(encryptedData)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt data: %w", err)
	}

	return plaintext, nil
}

// Exists implements KeychainProvider.Exists for file-based storage
func (f *FileSecureStorage) Exists(key string) bool {
	filePath := f.getFilePath(key)
	_, err := os.Stat(filePath)
	return err == nil
}

// Delete implements KeychainProvider.Delete for file-based storage
func (f *FileSecureStorage) Delete(key string) error {
	filePath := f.getFilePath(key)

	// Check if file exists
	if !f.Exists(key) {
		return nil
	}

	// Remove tamper protection before deleting
	if err := f.tamperProtection.RemoveProtection(filePath); err != nil {
		// Non-fatal error, log but continue
		fmt.Fprintf(os.Stderr, "Warning: Failed to remove tamper protection from %s: %v\n", filePath, err)
	}

	return os.Remove(filePath)
}

// getFilePath returns the file path for a key
func (f *FileSecureStorage) getFilePath(key string) string {
	// Sanitize key for use as filename
	safeKey := sanitizeKey(key)
	return fmt.Sprintf("%s/%s.bin", f.baseDir, safeKey)
}

// Placeholder functions removed - using real AES-256-GCM encryption in FileSecureStorage

func sanitizeKey(key string) string {
	// Simple sanitization for demonstration
	// A real implementation would be more thorough
	result := ""
	for _, c := range key {
		if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '-' || c == '_' || c == '.' {
			result += string(c)
		} else {
			result += "_"
		}
	}
	return result
}
