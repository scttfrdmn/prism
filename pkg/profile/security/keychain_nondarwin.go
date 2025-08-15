//go:build !darwin
// +build !darwin

package security

// initializeMacOSKeychain provides a stub for non-macOS platforms
func initializeMacOSKeychain() (KeychainProvider, error) {
	return NewFileSecureStorage()
}

// checkMacOSKeychainType always returns false on non-macOS platforms
func checkMacOSKeychainType(provider KeychainProvider) (isMacOSNative bool, info map[string]interface{}) {
	return false, nil
}

// diagnoseMacOSKeychain is a no-op on non-macOS platforms
func diagnoseMacOSKeychain(diagnostics *KeychainDiagnostics) {
	// No-op on non-macOS platforms
}