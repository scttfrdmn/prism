//go:build !darwin && !windows && !linux
// +build !darwin,!windows,!linux

// Package security provides stub implementations for unsupported platforms
package security

import "fmt"

// NewMacOSKeychainNative is not available on this platform
func NewMacOSKeychainNative() (*MacOSKeychainNative, error) {
	return nil, fmt.Errorf("macOS Keychain not available on this platform")
}

// NewWindowsCredentialManagerNative is not available on this platform
func NewWindowsCredentialManagerNative() (*WindowsCredentialManagerNative, error) {
	return nil, fmt.Errorf("Windows Credential Manager not available on this platform")
}

// NewLinuxSecretServiceNative is not available on this platform
func NewLinuxSecretServiceNative() (*LinuxSecretServiceNative, error) {
	return nil, fmt.Errorf("Linux Secret Service not available on this platform")
}

// Stub types for unsupported platforms
type MacOSKeychainNative struct{}
type WindowsCredentialManagerNative struct{}
type LinuxSecretServiceNative struct{}
