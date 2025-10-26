// Package security provides cryptographic operations for Prism profiles
package security

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"fmt"
	"io"
	"net"
	"os"
	"os/user"
	"runtime"
	"strings"
	"time"

	"golang.org/x/crypto/pbkdf2"
)

// CryptoProvider handles encryption and decryption operations
type CryptoProvider struct {
	key [32]byte // AES-256 key
}

// NewCryptoProvider creates a new crypto provider with device-specific key derivation
func NewCryptoProvider() (*CryptoProvider, error) {
	// Generate device-specific key using multiple entropy sources
	key, err := deriveDeviceKey()
	if err != nil {
		return nil, fmt.Errorf("failed to derive device key: %w", err)
	}

	return &CryptoProvider{
		key: key,
	}, nil
}

// Encrypt encrypts data using AES-256-GCM with random nonce
func (c *CryptoProvider) Encrypt(plaintext []byte) ([]byte, error) {
	// Create AES cipher
	block, err := aes.NewCipher(c.key[:])
	if err != nil {
		return nil, fmt.Errorf("failed to create AES cipher: %w", err)
	}

	// Create GCM mode
	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	// Generate random nonce
	nonce := make([]byte, aesGCM.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("failed to generate nonce: %w", err)
	}

	// Encrypt and authenticate
	ciphertext := aesGCM.Seal(nonce, nonce, plaintext, nil)
	return ciphertext, nil
}

// Decrypt decrypts data using AES-256-GCM
func (c *CryptoProvider) Decrypt(ciphertext []byte) ([]byte, error) {
	// Create AES cipher
	block, err := aes.NewCipher(c.key[:])
	if err != nil {
		return nil, fmt.Errorf("failed to create AES cipher: %w", err)
	}

	// Create GCM mode
	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	// Check minimum ciphertext size
	nonceSize := aesGCM.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, fmt.Errorf("ciphertext too short")
	}

	// Extract nonce and ciphertext
	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]

	// Decrypt and verify
	plaintext, err := aesGCM.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("decryption failed: %w", err)
	}

	return plaintext, nil
}

// deriveDeviceKey derives encryption key from device-specific information
func deriveDeviceKey() ([32]byte, error) {
	var key [32]byte

	// Collect device entropy sources
	entropy := []string{
		getDeviceEntropy(),
		getUserEntropy(),
		getSystemEntropy(),
		getInstallationEntropy(),
	}

	// Combine all entropy sources
	combined := strings.Join(entropy, "|")

	// Use PBKDF2 to derive key from combined entropy
	// Salt is derived from hostname and user to be consistent across invocations
	// but unique per device/user combination
	salt := deriveSalt()
	derived := pbkdf2.Key([]byte(combined), salt, 100000, 32, sha256.New)

	copy(key[:], derived)
	return key, nil
}

// getDeviceEntropy collects device-specific entropy
func getDeviceEntropy() string {
	var parts []string

	// Hostname
	if hostname, err := os.Hostname(); err == nil {
		parts = append(parts, "hostname:"+hostname)
	}

	// MAC addresses from primary interfaces
	if macs := getPrimaryMACAddresses(); len(macs) > 0 {
		parts = append(parts, "macs:"+strings.Join(macs, ","))
	}

	// Platform information
	parts = append(parts, "os:"+runtime.GOOS)
	parts = append(parts, "arch:"+runtime.GOARCH)

	return strings.Join(parts, ";")
}

// getUserEntropy collects user-specific entropy
func getUserEntropy() string {
	var parts []string

	// Current user information
	if currentUser, err := user.Current(); err == nil {
		parts = append(parts, "uid:"+currentUser.Uid)
		parts = append(parts, "gid:"+currentUser.Gid)
		parts = append(parts, "username:"+currentUser.Username)
		parts = append(parts, "homedir:"+currentUser.HomeDir)
	}

	return strings.Join(parts, ";")
}

// getSystemEntropy collects system-specific entropy
func getSystemEntropy() string {
	var parts []string

	// System-specific identifiers
	switch runtime.GOOS {
	case "darwin":
		// macOS system UUID
		if uuid := getMacOSSystemUUID(); uuid != "" {
			parts = append(parts, "system_uuid:"+uuid)
		}
	case "windows":
		// Windows machine GUID
		if guid := getWindowsMachineGUID(); guid != "" {
			parts = append(parts, "machine_guid:"+guid)
		}
	case "linux":
		// Linux machine ID
		if machineID := getLinuxMachineID(); machineID != "" {
			parts = append(parts, "machine_id:"+machineID)
		}
	}

	return strings.Join(parts, ";")
}

// getInstallationEntropy provides temporal entropy
func getInstallationEntropy() string {
	// This provides a timestamp that's consistent for this installation
	// but different if the profile is copied to another system
	installTime := getPrismInstallTime()
	return "install_time:" + installTime.Format(time.RFC3339)
}

// deriveSalt creates a consistent salt for PBKDF2
func deriveSalt() []byte {
	// Use hostname and username to create consistent but unique salt
	hostname, _ := os.Hostname()
	currentUser, _ := user.Current()
	username := ""
	if currentUser != nil {
		username = currentUser.Username
	}

	saltString := fmt.Sprintf("cloudworkstation-salt-%s-%s", hostname, username)
	hash := sha256.Sum256([]byte(saltString))
	return hash[:]
}

// Platform-specific system identifier functions

func getMacOSSystemUUID() string {
	// Get macOS system UUID from IOPlatformUUID
	// In a real implementation, this would use system calls
	// For now, return a placeholder that would be replaced with actual implementation
	return "macos-system-uuid-placeholder"
}

func getWindowsMachineGUID() string {
	// Get Windows machine GUID from registry
	// In a real implementation, this would read from Windows registry
	// For now, return a placeholder that would be replaced with actual implementation
	return "windows-machine-guid-placeholder"
}

func getLinuxMachineID() string {
	// Try to read /etc/machine-id
	if data, err := os.ReadFile("/etc/machine-id"); err == nil {
		return strings.TrimSpace(string(data))
	}

	// Fallback to /var/lib/dbus/machine-id
	if data, err := os.ReadFile("/var/lib/dbus/machine-id"); err == nil {
		return strings.TrimSpace(string(data))
	}

	return "linux-machine-id-unavailable"
}

// getPrimaryMACAddresses gets MAC addresses from primary network interfaces
func getPrimaryMACAddresses() []string {
	var addresses []string

	interfaces, err := net.Interfaces()
	if err != nil {
		return addresses
	}

	for _, iface := range interfaces {
		// Skip loopback interfaces
		if iface.Flags&net.FlagLoopback != 0 {
			continue
		}

		// Skip interfaces that are down
		if iface.Flags&net.FlagUp == 0 {
			continue
		}

		// Skip interfaces without a hardware address
		if len(iface.HardwareAddr) == 0 {
			continue
		}

		// Add the MAC address
		addresses = append(addresses, iface.HardwareAddr.String())
	}

	return addresses
}

func getPrismInstallTime() time.Time {
	// Try to get the creation time of the Prism config directory
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return time.Now()
	}

	configDir := fmt.Sprintf("%s/.prism", homeDir)
	if stat, err := os.Stat(configDir); err == nil {
		return stat.ModTime()
	}

	// If config directory doesn't exist, return current time
	// This will be consistent for this session but different if copied
	return time.Now()
}

// EncryptionError represents encryption-related errors
type EncryptionError struct {
	Operation string
	Err       error
}

func (e *EncryptionError) Error() string {
	return fmt.Sprintf("encryption %s failed: %v", e.Operation, e.Err)
}

func (e *EncryptionError) Unwrap() error {
	return e.Err
}

// Common encryption errors
var (
	ErrInvalidCiphertext = &EncryptionError{Operation: "decrypt", Err: fmt.Errorf("invalid ciphertext")}
	ErrKeyDerivation     = &EncryptionError{Operation: "key_derivation", Err: fmt.Errorf("key derivation failed")}
)
