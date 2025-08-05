//go:build !linux
// +build !linux

// Package security provides Linux stub implementations for non-Linux platforms
package security

import "fmt"

// LinuxSecretServiceNative stub for non-Linux platforms
type LinuxSecretServiceNative struct{}

// NewLinuxSecretServiceNative is not available on non-Linux platforms
func NewLinuxSecretServiceNative() (*LinuxSecretServiceNative, error) {
	return nil, fmt.Errorf("Linux Secret Service not available on this platform")
}

// Store implements KeychainProvider.Store (stub)
func (l *LinuxSecretServiceNative) Store(key string, data []byte) error {
	return fmt.Errorf("Linux Secret Service not available on this platform")
}

// Retrieve implements KeychainProvider.Retrieve (stub)
func (l *LinuxSecretServiceNative) Retrieve(key string) ([]byte, error) {
	return nil, fmt.Errorf("Linux Secret Service not available on this platform")
}

// Exists implements KeychainProvider.Exists (stub)
func (l *LinuxSecretServiceNative) Exists(key string) bool {
	return false
}

// Delete implements KeychainProvider.Delete (stub)
func (l *LinuxSecretServiceNative) Delete(key string) error {
	return fmt.Errorf("Linux Secret Service not available on this platform")
}

// GetKeychainInfo returns information about the Linux Secret Service integration (stub)
func (l *LinuxSecretServiceNative) GetKeychainInfo() map[string]interface{} {
	return map[string]interface{}{
		"provider": "Linux Secret Service (Stub)",
		"available": false,
	}
}

// Close is a stub method
func (l *LinuxSecretServiceNative) Close() error {
	return nil
}