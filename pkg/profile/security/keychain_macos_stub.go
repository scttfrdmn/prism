//go:build !darwin
// +build !darwin

// Package security provides macOS stub implementations for non-macOS platforms
package security

import "fmt"

// MacOSKeychainNative stub for non-macOS platforms
type MacOSKeychainNative struct{}

// NewMacOSKeychainNative is not available on non-macOS platforms
func NewMacOSKeychainNative() (*MacOSKeychainNative, error) {
	return nil, fmt.Errorf("macOS Keychain not available on this platform")
}

// Store implements KeychainProvider.Store (stub)
func (k *MacOSKeychainNative) Store(key string, data []byte) error {
	return fmt.Errorf("macOS Keychain not available on this platform")
}

// Retrieve implements KeychainProvider.Retrieve (stub)
func (k *MacOSKeychainNative) Retrieve(key string) ([]byte, error) {
	return nil, fmt.Errorf("macOS Keychain not available on this platform")
}

// Exists implements KeychainProvider.Exists (stub)
func (k *MacOSKeychainNative) Exists(key string) bool {
	return false
}

// Delete implements KeychainProvider.Delete (stub)
func (k *MacOSKeychainNative) Delete(key string) error {
	return fmt.Errorf("macOS Keychain not available on this platform")
}

// Close is a stub method
func (k *MacOSKeychainNative) Close() error {
	return nil
}