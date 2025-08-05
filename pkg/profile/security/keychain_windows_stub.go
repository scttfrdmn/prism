//go:build !windows
// +build !windows

// Package security provides Windows stub implementations for non-Windows platforms
package security

import "fmt"

// WindowsCredentialManagerNative stub for non-Windows platforms
type WindowsCredentialManagerNative struct{}

// NewWindowsCredentialManagerNative is not available on non-Windows platforms
func NewWindowsCredentialManagerNative() (*WindowsCredentialManagerNative, error) {
	return nil, fmt.Errorf("Windows Credential Manager not available on this platform")
}

// Store implements KeychainProvider.Store (stub)
func (w *WindowsCredentialManagerNative) Store(key string, data []byte) error {
	return fmt.Errorf("Windows Credential Manager not available on this platform")
}

// Retrieve implements KeychainProvider.Retrieve (stub)
func (w *WindowsCredentialManagerNative) Retrieve(key string) ([]byte, error) {
	return nil, fmt.Errorf("Windows Credential Manager not available on this platform")
}

// Exists implements KeychainProvider.Exists (stub)
func (w *WindowsCredentialManagerNative) Exists(key string) bool {
	return false
}

// Delete implements KeychainProvider.Delete (stub)
func (w *WindowsCredentialManagerNative) Delete(key string) error {
	return fmt.Errorf("Windows Credential Manager not available on this platform")
}

// GetKeychainInfo returns information about the Windows Credential Manager integration (stub)
func (w *WindowsCredentialManagerNative) GetKeychainInfo() map[string]interface{} {
	return map[string]interface{}{
		"provider": "Windows Credential Manager (Stub)",
		"available": false,
	}
}

// Close is a stub method
func (w *WindowsCredentialManagerNative) Close() error {
	return nil
}