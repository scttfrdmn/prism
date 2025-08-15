//go:build darwin && crosscompile
// +build darwin,crosscompile

package security

import (
	"fmt"
	"os"
)

// initializeMacOSKeychain for cross-compilation always uses file storage
func initializeMacOSKeychain() (KeychainProvider, error) {
	fmt.Fprintf(os.Stderr, "Cross-compiled macOS build - using secure file storage (keychain unavailable)\n")
	return NewFileSecureStorage()
}

// checkMacOSKeychainType always returns false for cross-compilation
func checkMacOSKeychainType(provider KeychainProvider) (isMacOSNative bool, info map[string]interface{}) {
	return false, nil
}

// diagnoseMacOSKeychain is a no-op for cross-compilation
func diagnoseMacOSKeychain(diagnostics *KeychainDiagnostics) {
	diagnostics.Recommendations = append(diagnostics.Recommendations, "Cross-compiled build: native macOS Keychain not available")
}