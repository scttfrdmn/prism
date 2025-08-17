//go:build darwin && crosscompile
// +build darwin,crosscompile

package security

import (
	"fmt"
	"os"
)

// MacOSKeychainNative stub for cross-compilation
type MacOSKeychainNative struct{}

// NewMacOSKeychainNative is not available during cross-compilation
func NewMacOSKeychainNative() (*MacOSKeychainNative, error) {
	return nil, fmt.Errorf("macOS Keychain not available during cross-compilation")
}

// Store implements KeychainProvider.Store (stub)
func (k *MacOSKeychainNative) Store(key string, data []byte) error {
	return fmt.Errorf("macOS Keychain not available during cross-compilation")
}

// Retrieve implements KeychainProvider.Retrieve (stub)
func (k *MacOSKeychainNative) Retrieve(key string) ([]byte, error) {
	return nil, fmt.Errorf("macOS Keychain not available during cross-compilation")
}

// Exists implements KeychainProvider.Exists (stub)
func (k *MacOSKeychainNative) Exists(key string) bool {
	return false
}

// Delete implements KeychainProvider.Delete (stub)
func (k *MacOSKeychainNative) Delete(key string) error {
	return fmt.Errorf("macOS Keychain not available during cross-compilation")
}

// Close is a stub method
func (k *MacOSKeychainNative) Close() error {
	return nil
}

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
