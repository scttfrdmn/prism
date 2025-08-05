//go:build windows
// +build windows

// Package security provides Windows Credential Manager integration
package security

import (
	"fmt"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	advapi32                = windows.NewLazySystemDLL("advapi32.dll")
	procCredWriteW          = advapi32.NewProc("CredWriteW")
	procCredReadW           = advapi32.NewProc("CredReadW")
	procCredDeleteW         = advapi32.NewProc("CredDeleteW")
	procCredFree            = advapi32.NewProc("CredFree")
)

// CREDENTIAL structure for Windows Credential Manager
type CREDENTIAL struct {
	Flags              uint32
	Type               uint32
	TargetName         *uint16
	Comment            *uint16
	LastWritten        windows.Filetime
	CredentialBlobSize uint32
	CredentialBlob     *byte
	Persist            uint32
	AttributeCount     uint32
	Attributes         uintptr
	TargetAlias        *uint16
	UserName           *uint16
}

const (
	CRED_TYPE_GENERIC                = 1
	CRED_PERSIST_LOCAL_MACHINE       = 2
	CRED_PERSIST_ENTERPRISE          = 3
	CRED_MAX_STRING_LENGTH           = 256
	CRED_MAX_USERNAME_LENGTH         = 513
	CRED_MAX_GENERIC_TARGET_NAME_LENGTH = 32767
)

// WindowsCredentialManagerNative implements native Windows Credential Manager storage
type WindowsCredentialManagerNative struct {
	targetPrefix string
}

// NewWindowsCredentialManagerNative creates a new native Windows credential manager provider
func NewWindowsCredentialManagerNative() (*WindowsCredentialManagerNative, error) {
	return &WindowsCredentialManagerNative{
		targetPrefix: "CloudWorkstation",
	}, nil
}

// Store implements KeychainProvider.Store for Windows using Credential Manager API
func (w *WindowsCredentialManagerNative) Store(key string, data []byte) error {
	targetName := fmt.Sprintf("%s\\%s", w.targetPrefix, key)
	
	// Convert target name to UTF-16
	targetNamePtr, err := syscall.UTF16PtrFromString(targetName)
	if err != nil {
		return fmt.Errorf("failed to convert target name to UTF-16: %w", err)
	}

	// Create CREDENTIAL structure
	cred := &CREDENTIAL{
		Type:               CRED_TYPE_GENERIC,
		TargetName:         targetNamePtr,
		CredentialBlobSize: uint32(len(data)),
		CredentialBlob:     &data[0],
		Persist:            CRED_PERSIST_LOCAL_MACHINE,
	}

	// Call CredWriteW
	ret, _, err := procCredWriteW.Call(
		uintptr(unsafe.Pointer(cred)),
		uintptr(0), // flags
	)

	if ret == 0 {
		return fmt.Errorf("CredWriteW failed: %w", err)
	}

	return nil
}

// Retrieve implements KeychainProvider.Retrieve for Windows using Credential Manager API
func (w *WindowsCredentialManagerNative) Retrieve(key string) ([]byte, error) {
	targetName := fmt.Sprintf("%s\\%s", w.targetPrefix, key)
	
	// Convert target name to UTF-16
	targetNamePtr, err := syscall.UTF16PtrFromString(targetName)
	if err != nil {
		return nil, fmt.Errorf("failed to convert target name to UTF-16: %w", err)
	}

	var credPtr uintptr

	// Call CredReadW
	ret, _, err := procCredReadW.Call(
		uintptr(unsafe.Pointer(targetNamePtr)),
		uintptr(CRED_TYPE_GENERIC),
		uintptr(0), // flags
		uintptr(unsafe.Pointer(&credPtr)),
	)

	if ret == 0 {
		// Check if the error is "not found"
		if errno, ok := err.(syscall.Errno); ok && errno == syscall.ERROR_NOT_FOUND {
			return nil, ErrKeychainNotFound
		}
		return nil, fmt.Errorf("CredReadW failed: %w", err)
	}

	if credPtr == 0 {
		return nil, ErrKeychainNotFound
	}

	// Convert pointer to CREDENTIAL structure
	cred := (*CREDENTIAL)(unsafe.Pointer(credPtr))
	
	// Copy credential blob data
	data := make([]byte, cred.CredentialBlobSize)
	if cred.CredentialBlobSize > 0 && cred.CredentialBlob != nil {
		copy(data, (*[1 << 30]byte)(unsafe.Pointer(cred.CredentialBlob))[:cred.CredentialBlobSize:cred.CredentialBlobSize])
	}

	// Free the credential structure
	procCredFree.Call(credPtr)

	return data, nil
}

// Exists implements KeychainProvider.Exists for Windows using Credential Manager API
func (w *WindowsCredentialManagerNative) Exists(key string) bool {
	targetName := fmt.Sprintf("%s\\%s", w.targetPrefix, key)
	
	// Convert target name to UTF-16
	targetNamePtr, err := syscall.UTF16PtrFromString(targetName)
	if err != nil {
		return false
	}

	var credPtr uintptr

	// Call CredReadW
	ret, _, _ := procCredReadW.Call(
		uintptr(unsafe.Pointer(targetNamePtr)),
		uintptr(CRED_TYPE_GENERIC),
		uintptr(0), // flags
		uintptr(unsafe.Pointer(&credPtr)),
	)

	if ret != 0 && credPtr != 0 {
		// Free the credential structure
		procCredFree.Call(credPtr)
		return true
	}

	return false
}

// Delete implements KeychainProvider.Delete for Windows using Credential Manager API
func (w *WindowsCredentialManagerNative) Delete(key string) error {
	targetName := fmt.Sprintf("%s\\%s", w.targetPrefix, key)
	
	// Convert target name to UTF-16
	targetNamePtr, err := syscall.UTF16PtrFromString(targetName)
	if err != nil {
		return fmt.Errorf("failed to convert target name to UTF-16: %w", err)
	}

	// Call CredDeleteW
	ret, _, err := procCredDeleteW.Call(
		uintptr(unsafe.Pointer(targetNamePtr)),
		uintptr(CRED_TYPE_GENERIC),
		uintptr(0), // flags
	)

	if ret == 0 {
		// Check if the error is "not found"
		if errno, ok := err.(syscall.Errno); ok && errno == syscall.ERROR_NOT_FOUND {
			// Item doesn't exist, which is fine for delete operation
			return nil
		}
		return fmt.Errorf("CredDeleteW failed: %w", err)
	}

	return nil
}

// GetKeychainInfo returns information about the Windows Credential Manager integration
func (w *WindowsCredentialManagerNative) GetKeychainInfo() map[string]interface{} {
	return map[string]interface{}{
		"provider":      "Windows Credential Manager (Native)",
		"target_prefix": w.targetPrefix,
		"api":          "advapi32.dll",
		"persistence":  "CRED_PERSIST_LOCAL_MACHINE",
		"security_level": "Windows DPAPI encryption",
	}
}