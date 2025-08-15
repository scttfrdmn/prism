// Package security provides keychain information and diagnostics
package security

import (
	"fmt"
	"runtime"
)

// KeychainInfo provides information about the keychain provider in use
type KeychainInfo struct {
	Provider        string                 `json:"provider"`
	Platform        string                 `json:"platform"`
	Native          bool                   `json:"native"`
	Available       bool                   `json:"available"`
	SecurityLevel   string                 `json:"security_level"`
	Details         map[string]interface{} `json:"details"`
	FallbackReason  string                 `json:"fallback_reason,omitempty"`
}

// GetKeychainInfo returns detailed information about the current keychain provider
func GetKeychainInfo() (*KeychainInfo, error) {
	provider, err := NewKeychainProvider()
	if err != nil {
		return &KeychainInfo{
			Provider:       "failed",
			Platform:       runtime.GOOS,
			Native:         false,
			Available:      false,
			SecurityLevel:  "none",
			FallbackReason: err.Error(),
		}, err
	}

	info := &KeychainInfo{
		Platform:  runtime.GOOS,
		Available: true,
	}

	// Try to determine provider type based on what we got
	if isMacOS, macOSInfo := checkMacOSKeychainType(provider); isMacOS {
		info.Provider = "macOS Keychain (Native)"
		info.Native = true
		info.SecurityLevel = "Hardware-backed secure enclave when available"
		info.Details = macOSInfo
	} else {
		switch p := provider.(type) {
		case *WindowsCredentialManagerNative:
		info.Provider = "Windows Credential Manager (Native)"
		info.Native = true
		info.SecurityLevel = "Windows DPAPI encryption"
		info.Details = p.GetKeychainInfo()
		
	case *LinuxSecretServiceNative:
		info.Provider = "Linux Secret Service (Native)"
		info.Native = true
		info.SecurityLevel = "Desktop environment keyring"
		info.Details = p.GetKeychainInfo()
		
	case *FileSecureStorage:
		info.Provider = "File-based Secure Storage (Fallback)"
		info.Native = false
		info.SecurityLevel = "AES-256-GCM with device-specific key derivation"
		info.Details = map[string]interface{}{
			"encryption":    "AES-256-GCM",
			"key_derivation": "PBKDF2 with device-specific entropy",
			"file_permissions": "0600 (owner read/write only)",
			"tamper_protection": "SHA-256 checksums",
		}
		info.FallbackReason = "Native keychain not available or failed initialization"
		
		default:
			info.Provider = "Unknown"
			info.Native = false
			info.SecurityLevel = "Unknown"
			info.Details = map[string]interface{}{}
		}
	}

	return info, nil
}

// ValidateKeychainProvider performs comprehensive validation of the keychain provider
func ValidateKeychainProvider() error {
	provider, err := NewKeychainProvider()
	if err != nil {
		return fmt.Errorf("keychain provider creation failed: %w", err)
	}

	// Test basic operations
	testKey := "validation-test-key"
	testData := []byte("validation test data")

	// Test Store
	if err := provider.Store(testKey, testData); err != nil {
		return fmt.Errorf("keychain Store operation failed: %w", err)
	}

	// Test Exists
	if !provider.Exists(testKey) {
		return fmt.Errorf("keychain Exists operation failed: key should exist after Store")
	}

	// Test Retrieve
	retrievedData, err := provider.Retrieve(testKey)
	if err != nil {
		// Clean up before returning error
		provider.Delete(testKey)
		return fmt.Errorf("keychain Retrieve operation failed: %w", err)
	}

	if string(retrievedData) != string(testData) {
		// Clean up before returning error
		provider.Delete(testKey)
		return fmt.Errorf("keychain Retrieve operation failed: data mismatch")
	}

	// Test Delete
	if err := provider.Delete(testKey); err != nil {
		return fmt.Errorf("keychain Delete operation failed: %w", err)
	}

	// Verify deletion
	if provider.Exists(testKey) {
		return fmt.Errorf("keychain Delete operation failed: key still exists")
	}

	return nil
}

// DiagnoseKeychainIssues provides diagnostic information for keychain problems
func DiagnoseKeychainIssues() *KeychainDiagnostics {
	diagnostics := &KeychainDiagnostics{
		Platform: runtime.GOOS,
		Issues:   []string{},
		Warnings: []string{},
		Recommendations: []string{},
	}

	// Get keychain info
	info, err := GetKeychainInfo()
	if err != nil {
		diagnostics.Issues = append(diagnostics.Issues, fmt.Sprintf("Failed to get keychain info: %v", err))
		diagnostics.Recommendations = append(diagnostics.Recommendations, "Check system permissions and keychain availability")
		return diagnostics
	}

	diagnostics.Info = info

	// Platform-specific diagnostics
	switch runtime.GOOS {
	case "darwin":
		diagnoseMacOSKeychain(diagnostics)
	case "windows":
		diagnoseWindowsCredentialManager(diagnostics)
	case "linux":
		diagnoseLinuxSecretService(diagnostics)
	default:
		diagnostics.Warnings = append(diagnostics.Warnings, "Platform not officially supported, using fallback storage")
	}

	// General recommendations
	if !info.Native {
		diagnostics.Warnings = append(diagnostics.Warnings, "Using fallback file-based storage instead of native keychain")
		diagnostics.Recommendations = append(diagnostics.Recommendations, "Consider installing appropriate keychain software for better security")
	}

	return diagnostics
}

// KeychainDiagnostics provides diagnostic information about keychain status
type KeychainDiagnostics struct {
	Platform        string           `json:"platform"`
	Info            *KeychainInfo    `json:"info,omitempty"`
	Issues          []string         `json:"issues"`
	Warnings        []string         `json:"warnings"`
	Recommendations []string         `json:"recommendations"`
}


func diagnoseWindowsCredentialManager(diagnostics *KeychainDiagnostics) {
	// Test if we can create native Windows credential manager
	_, err := NewWindowsCredentialManagerNative()
	if err != nil {
		diagnostics.Issues = append(diagnostics.Issues, fmt.Sprintf("Windows Credential Manager unavailable: %v", err))
		diagnostics.Recommendations = append(diagnostics.Recommendations, "Ensure Windows Credential Manager service is running")
	} else {
		diagnostics.Recommendations = append(diagnostics.Recommendations, "Windows Credential Manager provides DPAPI-protected storage")
	}
}

func diagnoseLinuxSecretService(diagnostics *KeychainDiagnostics) {
	// Test if we can create native Linux Secret Service
	native, err := NewLinuxSecretServiceNative()
	if err != nil {
		diagnostics.Issues = append(diagnostics.Issues, fmt.Sprintf("Linux Secret Service unavailable: %v", err))
		diagnostics.Recommendations = append(diagnostics.Recommendations, 
			"Install and configure a Secret Service provider (GNOME Keyring, KDE Wallet, etc.)")
	} else {
		native.Close()
		diagnostics.Recommendations = append(diagnostics.Recommendations, "Linux Secret Service provides desktop keyring integration")
	}
}