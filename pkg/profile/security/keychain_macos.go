//go:build darwin
// +build darwin

// Package security provides macOS Keychain integration using the Security framework
package security

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework Security -framework Foundation
#include <Security/Security.h>
#include <CoreFoundation/CoreFoundation.h>
#include <stdio.h>

// Helper function to store data in macOS Keychain
OSStatus storeInKeychain(const char* serviceName, const char* accountName, 
                        const void* data, UInt32 dataLength, Boolean updateExisting) {
    CFStringRef service = CFStringCreateWithCString(NULL, serviceName, kCFStringEncodingUTF8);
    CFStringRef account = CFStringCreateWithCString(NULL, accountName, kCFStringEncodingUTF8);
    
    if (!service || !account) {
        if (service) CFRelease(service);
        if (account) CFRelease(account);
        return errSecParam;
    }
    
    OSStatus status;
    
    if (updateExisting) {
        // Try to update existing item first
        CFMutableDictionaryRef query = CFDictionaryCreateMutable(NULL, 0, 
            &kCFTypeDictionaryKeyCallBacks, &kCFTypeDictionaryValueCallBacks);
        CFDictionarySetValue(query, kSecClass, kSecClassGenericPassword);
        CFDictionarySetValue(query, kSecAttrService, service);
        CFDictionarySetValue(query, kSecAttrAccount, account);
        
        CFMutableDictionaryRef attributes = CFDictionaryCreateMutable(NULL, 0,
            &kCFTypeDictionaryKeyCallBacks, &kCFTypeDictionaryValueCallBacks);
        CFDataRef valueData = CFDataCreate(NULL, (const UInt8*)data, dataLength);
        CFDictionarySetValue(attributes, kSecValueData, valueData);
        
        status = SecItemUpdate(query, attributes);
        
        CFRelease(query);
        CFRelease(attributes);
        CFRelease(valueData);
        
        if (status == errSecSuccess) {
            CFRelease(service);
            CFRelease(account);
            return status;
        }
        // Fall through to create new item if update failed
    }
    
    // Create new keychain item
    CFMutableDictionaryRef attributes = CFDictionaryCreateMutable(NULL, 0,
        &kCFTypeDictionaryKeyCallBacks, &kCFTypeDictionaryValueCallBacks);
    
    CFDataRef valueData = CFDataCreate(NULL, (const UInt8*)data, dataLength);
    
    CFDictionarySetValue(attributes, kSecClass, kSecClassGenericPassword);
    CFDictionarySetValue(attributes, kSecAttrService, service);
    CFDictionarySetValue(attributes, kSecAttrAccount, account);
    CFDictionarySetValue(attributes, kSecValueData, valueData);
    CFDictionarySetValue(attributes, kSecAttrAccessible, kSecAttrAccessibleWhenUnlockedThisDeviceOnly);
    
    status = SecItemAdd(attributes, NULL);
    
    CFRelease(service);
    CFRelease(account);
    CFRelease(attributes);
    CFRelease(valueData);
    
    return status;
}

// Helper function to retrieve data from macOS Keychain
OSStatus retrieveFromKeychain(const char* serviceName, const char* accountName,
                             void** data, UInt32* dataLength) {
    CFStringRef service = CFStringCreateWithCString(NULL, serviceName, kCFStringEncodingUTF8);
    CFStringRef account = CFStringCreateWithCString(NULL, accountName, kCFStringEncodingUTF8);
    
    if (!service || !account) {
        if (service) CFRelease(service);
        if (account) CFRelease(account);
        return errSecParam;
    }
    
    CFMutableDictionaryRef query = CFDictionaryCreateMutable(NULL, 0,
        &kCFTypeDictionaryKeyCallBacks, &kCFTypeDictionaryValueCallBacks);
    
    CFDictionarySetValue(query, kSecClass, kSecClassGenericPassword);
    CFDictionarySetValue(query, kSecAttrService, service);
    CFDictionarySetValue(query, kSecAttrAccount, account);
    CFDictionarySetValue(query, kSecReturnData, kCFBooleanTrue);
    CFDictionarySetValue(query, kSecMatchLimit, kSecMatchLimitOne);
    
    CFDataRef result = NULL;
    OSStatus status = SecItemCopyMatching(query, (CFTypeRef*)&result);
    
    CFRelease(service);
    CFRelease(account);
    CFRelease(query);
    
    if (status == errSecSuccess && result) {
        CFIndex length = CFDataGetLength(result);
        *dataLength = (UInt32)length;
        *data = malloc(length);
        if (*data) {
            CFDataGetBytes(result, CFRangeMake(0, length), (UInt8*)*data);
        } else {
            status = errSecAllocate;
        }
        CFRelease(result);
    }
    
    return status;
}

// Helper function to check if item exists in macOS Keychain
OSStatus existsInKeychain(const char* serviceName, const char* accountName) {
    CFStringRef service = CFStringCreateWithCString(NULL, serviceName, kCFStringEncodingUTF8);
    CFStringRef account = CFStringCreateWithCString(NULL, accountName, kCFStringEncodingUTF8);
    
    if (!service || !account) {
        if (service) CFRelease(service);
        if (account) CFRelease(account);
        return errSecParam;
    }
    
    CFMutableDictionaryRef query = CFDictionaryCreateMutable(NULL, 0,
        &kCFTypeDictionaryKeyCallBacks, &kCFTypeDictionaryValueCallBacks);
    
    CFDictionarySetValue(query, kSecClass, kSecClassGenericPassword);
    CFDictionarySetValue(query, kSecAttrService, service);
    CFDictionarySetValue(query, kSecAttrAccount, account);
    CFDictionarySetValue(query, kSecReturnAttributes, kCFBooleanTrue);
    CFDictionarySetValue(query, kSecMatchLimit, kSecMatchLimitOne);
    
    CFDictionaryRef result = NULL;
    OSStatus status = SecItemCopyMatching(query, (CFTypeRef*)&result);
    
    CFRelease(service);
    CFRelease(account);
    CFRelease(query);
    
    if (result) {
        CFRelease(result);
    }
    
    return status;
}

// Helper function to delete data from macOS Keychain
OSStatus deleteFromKeychain(const char* serviceName, const char* accountName) {
    CFStringRef service = CFStringCreateWithCString(NULL, serviceName, kCFStringEncodingUTF8);
    CFStringRef account = CFStringCreateWithCString(NULL, accountName, kCFStringEncodingUTF8);
    
    if (!service || !account) {
        if (service) CFRelease(service);
        if (account) CFRelease(account);
        return errSecParam;
    }
    
    CFMutableDictionaryRef query = CFDictionaryCreateMutable(NULL, 0,
        &kCFTypeDictionaryKeyCallBacks, &kCFTypeDictionaryValueCallBacks);
    
    CFDictionarySetValue(query, kSecClass, kSecClassGenericPassword);
    CFDictionarySetValue(query, kSecAttrService, service);
    CFDictionarySetValue(query, kSecAttrAccount, account);
    
    OSStatus status = SecItemDelete(query);
    
    CFRelease(service);
    CFRelease(account);
    CFRelease(query);
    
    return status;
}
*/
import "C"
import (
	"fmt"
	"unsafe"
)

// MacOSKeychainNative implements native macOS Keychain storage
type MacOSKeychainNative struct {
	serviceName string
}

// NewMacOSKeychainNative creates a new native macOS keychain provider
func NewMacOSKeychainNative() (*MacOSKeychainNative, error) {
	return &MacOSKeychainNative{
		serviceName: "com.cloudworkstation.profiles",
	}, nil
}

// Store implements KeychainProvider.Store for macOS using Security framework
func (k *MacOSKeychainNative) Store(key string, data []byte) error {
	cService := C.CString(k.serviceName)
	cAccount := C.CString(key)
	defer C.free(unsafe.Pointer(cService))
	defer C.free(unsafe.Pointer(cAccount))

	status := C.storeInKeychain(cService, cAccount, 
		unsafe.Pointer(&data[0]), C.UInt32(len(data)), C.Boolean(1))

	if status != C.errSecSuccess {
		return fmt.Errorf("failed to store in macOS Keychain: OSStatus %d", int(status))
	}

	return nil
}

// Retrieve implements KeychainProvider.Retrieve for macOS using Security framework
func (k *MacOSKeychainNative) Retrieve(key string) ([]byte, error) {
	cService := C.CString(k.serviceName)
	cAccount := C.CString(key)
	defer C.free(unsafe.Pointer(cService))
	defer C.free(unsafe.Pointer(cAccount))

	var data unsafe.Pointer
	var dataLength C.UInt32

	status := C.retrieveFromKeychain(cService, cAccount, &data, &dataLength)

	if status == C.errSecItemNotFound {
		return nil, ErrKeychainNotFound
	}

	if status != C.errSecSuccess {
		return nil, fmt.Errorf("failed to retrieve from macOS Keychain: OSStatus %d", int(status))
	}

	if data == nil {
		return nil, ErrKeychainNotFound
	}

	// Convert C data to Go slice
	result := C.GoBytes(data, C.int(dataLength))
	C.free(data) // Free the allocated C memory

	return result, nil
}

// Exists implements KeychainProvider.Exists for macOS using Security framework
func (k *MacOSKeychainNative) Exists(key string) bool {
	cService := C.CString(k.serviceName)
	cAccount := C.CString(key)
	defer C.free(unsafe.Pointer(cService))
	defer C.free(unsafe.Pointer(cAccount))

	status := C.existsInKeychain(cService, cAccount)
	return status == C.errSecSuccess
}

// Delete implements KeychainProvider.Delete for macOS using Security framework
func (k *MacOSKeychainNative) Delete(key string) error {
	cService := C.CString(k.serviceName)
	cAccount := C.CString(key)
	defer C.free(unsafe.Pointer(cService))
	defer C.free(unsafe.Pointer(cAccount))

	status := C.deleteFromKeychain(cService, cAccount)

	if status == C.errSecItemNotFound {
		// Item doesn't exist, which is fine for delete operation
		return nil
	}

	if status != C.errSecSuccess {
		return fmt.Errorf("failed to delete from macOS Keychain: OSStatus %d", int(status))
	}

	return nil
}

// GetKeychainInfo returns information about the macOS Keychain integration
func (k *MacOSKeychainNative) GetKeychainInfo() map[string]interface{} {
	return map[string]interface{}{
		"provider":     "macOS Keychain (Native)",
		"service_name": k.serviceName,
		"framework":   "Security.framework",
		"accessibility": "kSecAttrAccessibleWhenUnlockedThisDeviceOnly",
		"security_level": "Hardware-backed secure enclave when available",
	}
}