//go:build darwin
// +build darwin

package security

import (
	"fmt"
	"os"
)

// initializeMacOSKeychain initializes the macOS-specific keychain provider
func initializeMacOSKeychain() (KeychainProvider, error) {
	if isDevelopmentMode() {
		fmt.Fprintf(os.Stderr, "Development mode detected - using secure file storage to avoid keychain prompts\n")
		return NewFileSecureStorage()
	}
	
	// Production mode - use native keychain
	native, err := NewMacOSKeychainNative()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: Failed to initialize native macOS Keychain, using secure file storage: %v\n", err)
		return NewFileSecureStorage()
	}
	return native, nil
}

// checkMacOSKeychainType checks if provider is macOS native keychain
func checkMacOSKeychainType(provider KeychainProvider) (isMacOSNative bool, info map[string]interface{}) {
	if p, ok := provider.(*MacOSKeychainNative); ok {
		return true, map[string]interface{}{
			"provider": "macOS Keychain (Native)",
			"framework": "Security.framework",
			"instance": p,
		}
	}
	return false, nil
}

// diagnoseMacOSKeychain performs macOS-specific keychain diagnostics
func diagnoseMacOSKeychain(diagnostics *KeychainDiagnostics) {
	// Test if we can create native macOS keychain
	_, err := NewMacOSKeychainNative()
	if err != nil {
		diagnostics.Issues = append(diagnostics.Issues, fmt.Sprintf("macOS Keychain unavailable: %v", err))
		diagnostics.Recommendations = append(diagnostics.Recommendations, "Ensure Security framework is available and keychain is unlocked")
	} else {
		diagnostics.Recommendations = append(diagnostics.Recommendations, "macOS Keychain provides hardware-backed security when available")
	}
}