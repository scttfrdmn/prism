// Package security provides secure storage and verification for CloudWorkstation profiles.
package security

import (
	"errors"
	"fmt"
	"os"
	"runtime"
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

// NewKeychainProvider creates the appropriate keychain provider for the current platform
func NewKeychainProvider() (KeychainProvider, error) {
	switch runtime.GOOS {
	case "darwin":
		return NewMacOSKeychain()
	case "windows":
		return NewWindowsCredentialManager()
	case "linux":
		return NewLinuxSecretService()
	default:
		// Fallback to file-based storage with warning
		fmt.Fprintf(os.Stderr, "Warning: Using fallback secure storage on platform: %s\n", runtime.GOOS)
		return NewFileSecureStorage()
	}
}

// MacOSKeychain implements KeychainProvider for macOS
type MacOSKeychain struct {
	// Fields needed for macOS keychain operations
	serviceName string
}

// NewMacOSKeychain creates a new macOS keychain provider
func NewMacOSKeychain() (*MacOSKeychain, error) {
	return &MacOSKeychain{
		serviceName: "com.cloudworkstation.profiles",
	}, nil
}

// Store implements KeychainProvider.Store for macOS
func (k *MacOSKeychain) Store(key string, data []byte) error {
	// On macOS, we would use the keychain API
	// This is a placeholder for the actual implementation
	
	// macOSKeychainAdd would be a CGO function calling the Security framework
	// err := macOSKeychainAdd(k.serviceName, key, data)
	// if err != nil {
	//     return fmt.Errorf("failed to store in keychain: %w", err)
	// }
	
	// For now, fall back to secure file storage
	fs, err := NewFileSecureStorage()
	if err != nil {
		return err
	}
	return fs.Store(key, data)
}

// Retrieve implements KeychainProvider.Retrieve for macOS
func (k *MacOSKeychain) Retrieve(key string) ([]byte, error) {
	// On macOS, we would use the keychain API
	// This is a placeholder for the actual implementation
	
	// data, err := macOSKeychainFind(k.serviceName, key)
	// if err != nil {
	//     return nil, fmt.Errorf("failed to retrieve from keychain: %w", err)
	// }
	
	// For now, fall back to secure file storage
	fs, err := NewFileSecureStorage()
	if err != nil {
		return nil, err
	}
	return fs.Retrieve(key)
}

// Exists implements KeychainProvider.Exists for macOS
func (k *MacOSKeychain) Exists(key string) bool {
	// On macOS, we would check if the item exists in keychain
	// This is a placeholder for the actual implementation
	
	// exists, _ := macOSKeychainExists(k.serviceName, key)
	// return exists
	
	// For now, fall back to secure file storage
	fs, err := NewFileSecureStorage()
	if err != nil {
		return false
	}
	return fs.Exists(key)
}

// Delete implements KeychainProvider.Delete for macOS
func (k *MacOSKeychain) Delete(key string) error {
	// On macOS, we would use the keychain API
	// This is a placeholder for the actual implementation
	
	// err := macOSKeychainDelete(k.serviceName, key)
	// if err != nil {
	//     return fmt.Errorf("failed to delete from keychain: %w", err)
	// }
	
	// For now, fall back to secure file storage
	fs, err := NewFileSecureStorage()
	if err != nil {
		return err
	}
	return fs.Delete(key)
}

// WindowsCredentialManager implements KeychainProvider for Windows
type WindowsCredentialManager struct {
	// Fields needed for Windows credential operations
	targetName string
}

// NewWindowsCredentialManager creates a new Windows credential manager provider
func NewWindowsCredentialManager() (*WindowsCredentialManager, error) {
	return &WindowsCredentialManager{
		targetName: "CloudWorkstationProfiles",
	}, nil
}

// Store implements KeychainProvider.Store for Windows
func (w *WindowsCredentialManager) Store(key string, data []byte) error {
	// On Windows, we would use the Credential Manager API
	// This is a placeholder for the actual implementation
	
	// credKey := w.targetName + "/" + key
	// err := windowsCredWrite(credKey, data)
	// if err != nil {
	//     return fmt.Errorf("failed to store in credential manager: %w", err)
	// }
	
	// For now, fall back to secure file storage
	fs, err := NewFileSecureStorage()
	if err != nil {
		return err
	}
	return fs.Store(key, data)
}

// Retrieve implements KeychainProvider.Retrieve for Windows
func (w *WindowsCredentialManager) Retrieve(key string) ([]byte, error) {
	// On Windows, we would use the Credential Manager API
	// This is a placeholder for the actual implementation
	
	// credKey := w.targetName + "/" + key
	// data, err := windowsCredRead(credKey)
	// if err != nil {
	//     return nil, fmt.Errorf("failed to retrieve from credential manager: %w", err)
	// }
	
	// For now, fall back to secure file storage
	fs, err := NewFileSecureStorage()
	if err != nil {
		return nil, err
	}
	return fs.Retrieve(key)
}

// Exists implements KeychainProvider.Exists for Windows
func (w *WindowsCredentialManager) Exists(key string) bool {
	// On Windows, we would check if the credential exists
	// This is a placeholder for the actual implementation
	
	// credKey := w.targetName + "/" + key
	// exists, _ := windowsCredExists(credKey)
	// return exists
	
	// For now, fall back to secure file storage
	fs, err := NewFileSecureStorage()
	if err != nil {
		return false
	}
	return fs.Exists(key)
}

// Delete implements KeychainProvider.Delete for Windows
func (w *WindowsCredentialManager) Delete(key string) error {
	// On Windows, we would use the Credential Manager API
	// This is a placeholder for the actual implementation
	
	// credKey := w.targetName + "/" + key
	// err := windowsCredDelete(credKey)
	// if err != nil {
	//     return fmt.Errorf("failed to delete from credential manager: %w", err)
	// }
	
	// For now, fall back to secure file storage
	fs, err := NewFileSecureStorage()
	if err != nil {
		return err
	}
	return fs.Delete(key)
}

// LinuxSecretService implements KeychainProvider for Linux
type LinuxSecretService struct {
	// Fields needed for Secret Service operations
	collection string
}

// NewLinuxSecretService creates a new Linux Secret Service provider
func NewLinuxSecretService() (*LinuxSecretService, error) {
	return &LinuxSecretService{
		collection: "cloudworkstation",
	}, nil
}

// Store implements KeychainProvider.Store for Linux
func (l *LinuxSecretService) Store(key string, data []byte) error {
	// On Linux, we would use the Secret Service API
	// This is a placeholder for the actual implementation
	
	// err := secretServiceStore(l.collection, key, data)
	// if err != nil {
	//     return fmt.Errorf("failed to store in secret service: %w", err)
	// }
	
	// For now, fall back to secure file storage
	fs, err := NewFileSecureStorage()
	if err != nil {
		return err
	}
	return fs.Store(key, data)
}

// Retrieve implements KeychainProvider.Retrieve for Linux
func (l *LinuxSecretService) Retrieve(key string) ([]byte, error) {
	// On Linux, we would use the Secret Service API
	// This is a placeholder for the actual implementation
	
	// data, err := secretServiceRetrieve(l.collection, key)
	// if err != nil {
	//     return nil, fmt.Errorf("failed to retrieve from secret service: %w", err)
	// }
	
	// For now, fall back to secure file storage
	fs, err := NewFileSecureStorage()
	if err != nil {
		return nil, err
	}
	return fs.Retrieve(key)
}

// Exists implements KeychainProvider.Exists for Linux
func (l *LinuxSecretService) Exists(key string) bool {
	// On Linux, we would check if the secret exists
	// This is a placeholder for the actual implementation
	
	// exists, _ := secretServiceExists(l.collection, key)
	// return exists
	
	// For now, fall back to secure file storage
	fs, err := NewFileSecureStorage()
	if err != nil {
		return false
	}
	return fs.Exists(key)
}

// Delete implements KeychainProvider.Delete for Linux
func (l *LinuxSecretService) Delete(key string) error {
	// On Linux, we would use the Secret Service API
	// This is a placeholder for the actual implementation
	
	// err := secretServiceDelete(l.collection, key)
	// if err != nil {
	//     return fmt.Errorf("failed to delete from secret service: %w", err)
	// }
	
	// For now, fall back to secure file storage
	fs, err := NewFileSecureStorage()
	if err != nil {
		return err
	}
	return fs.Delete(key)
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
	if err := os.MkdirAll(baseDir, 0700); err != nil {  // Owner only
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
	if err := os.WriteFile(filePath, encryptedData, 0600); err != nil {  // Owner read/write only
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