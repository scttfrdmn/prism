// Package profile provides functionality for managing Prism profiles
package profile

import (
	"bytes"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/zalando/go-keyring"
	"golang.org/x/crypto/nacl/secretbox"
	"golang.org/x/crypto/pbkdf2"
	"golang.org/x/crypto/scrypt"
)

// Credentials holds AWS credentials data
type Credentials struct {
	AccessKeyID     string     `json:"access_key_id"`
	SecretAccessKey string     `json:"secret_access_key"`
	SessionToken    string     `json:"session_token,omitempty"`
	Expiration      *time.Time `json:"expiration,omitempty"`
}

// CredentialProvider defines the interface for storing and retrieving credentials
type CredentialProvider interface {
	// GetCredentials retrieves credentials for a profile
	GetCredentials(profileName string) (*Credentials, error)

	// StoreCredentials stores credentials for a profile
	StoreCredentials(profileName string, creds *Credentials) error

	// ClearCredentials removes credentials for a profile
	ClearCredentials(profileName string) error
}

// SecureCredentialProvider is a platform-specific secure credential provider
type SecureCredentialProvider struct {
	// Platform-specific implementation details
	service    string
	configPath string
}

// NewCredentialProvider creates a new secure credential provider
func NewCredentialProvider() (CredentialProvider, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}

	// Create Prism directory if it doesn't exist
	cwsDir := filepath.Join(homeDir, ".prism", "credentials")
	if err := os.MkdirAll(cwsDir, 0700); err != nil {
		return nil, fmt.Errorf("failed to create credentials directory: %w", err)
	}

	provider := &SecureCredentialProvider{
		service:    "Prism",
		configPath: cwsDir,
	}

	return provider, nil
}

// GetCredentials retrieves credentials for a profile using platform-specific secure storage
func (p *SecureCredentialProvider) GetCredentials(profileName string) (*Credentials, error) {
	// Attempt to use platform-specific secure storage
	creds, err := p.getFromSecureStorage(profileName)
	if err == nil {
		return creds, nil
	}

	// Fall back to local file if secure storage is not available
	return p.getFromFile(profileName)
}

// StoreCredentials stores credentials for a profile using platform-specific secure storage
func (p *SecureCredentialProvider) StoreCredentials(profileName string, creds *Credentials) error {
	// Try to use platform-specific secure storage
	err := p.storeInSecureStorage(profileName, creds)
	if err == nil {
		return nil
	}

	// Fall back to local file if secure storage is not available
	return p.storeInFile(profileName, creds)
}

// ClearCredentials removes credentials for a profile
func (p *SecureCredentialProvider) ClearCredentials(profileName string) error {
	// Try to remove from platform-specific secure storage
	err := p.removeFromSecureStorage(profileName)
	if err == nil {
		return nil
	}

	// Fall back to removing local file
	return p.removeFromFile(profileName)
}

// Platform-specific secure storage implementations

func (p *SecureCredentialProvider) getFromSecureStorage(profileName string) (*Credentials, error) {
	switch runtime.GOOS {
	case "darwin":
		return p.getFromKeychain(profileName)
	case "windows":
		return p.getFromCredentialManager(profileName)
	case "linux":
		return p.getFromSecretService(profileName)
	default:
		return nil, fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}
}

func (p *SecureCredentialProvider) storeInSecureStorage(profileName string, creds *Credentials) error {
	switch runtime.GOOS {
	case "darwin":
		return p.storeInKeychain(profileName, creds)
	case "windows":
		return p.storeInCredentialManager(profileName, creds)
	case "linux":
		return p.storeInSecretService(profileName, creds)
	default:
		return fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}
}

func (p *SecureCredentialProvider) removeFromSecureStorage(profileName string) error {
	switch runtime.GOOS {
	case "darwin":
		return p.removeFromKeychain(profileName)
	case "windows":
		return p.removeFromCredentialManager(profileName)
	case "linux":
		return p.removeFromSecretService(profileName)
	default:
		return fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}
}

// macOS Keychain implementations using go-keyring

func (p *SecureCredentialProvider) getFromKeychain(profileName string) (*Credentials, error) {
	// Retrieve from macOS Keychain using go-keyring
	data, err := keyring.Get(p.service, profileName)
	if err != nil {
		return nil, fmt.Errorf("failed to get credentials from keychain: %w", err)
	}

	// Decode JSON credentials
	var creds Credentials
	if err := json.Unmarshal([]byte(data), &creds); err != nil {
		return nil, fmt.Errorf("failed to decode credentials: %w", err)
	}

	return &creds, nil
}

func (p *SecureCredentialProvider) storeInKeychain(profileName string, creds *Credentials) error {
	// Encode credentials as JSON
	data, err := json.Marshal(creds) //nolint:gosec // G117: serializing credentials for the OS secure store is the purpose here
	if err != nil {
		return fmt.Errorf("failed to encode credentials: %w", err)
	}

	// Store in macOS Keychain using go-keyring
	if err := keyring.Set(p.service, profileName, string(data)); err != nil {
		return fmt.Errorf("failed to store credentials in keychain: %w", err)
	}

	return nil
}

func (p *SecureCredentialProvider) removeFromKeychain(profileName string) error {
	// Remove from macOS Keychain using go-keyring
	if err := keyring.Delete(p.service, profileName); err != nil {
		// If the item doesn't exist, that's okay
		if err == keyring.ErrNotFound {
			return nil
		}
		return fmt.Errorf("failed to remove credentials from keychain: %w", err)
	}

	return nil
}

// Windows Credential Manager implementations using go-keyring

func (p *SecureCredentialProvider) getFromCredentialManager(profileName string) (*Credentials, error) {
	// Retrieve from Windows Credential Manager using go-keyring
	data, err := keyring.Get(p.service, profileName)
	if err != nil {
		return nil, fmt.Errorf("failed to get credentials from credential manager: %w", err)
	}

	// Decode JSON credentials
	var creds Credentials
	if err := json.Unmarshal([]byte(data), &creds); err != nil {
		return nil, fmt.Errorf("failed to decode credentials: %w", err)
	}

	return &creds, nil
}

func (p *SecureCredentialProvider) storeInCredentialManager(profileName string, creds *Credentials) error {
	// Encode credentials as JSON
	data, err := json.Marshal(creds) //nolint:gosec // G117: serializing credentials for the OS secure store is the purpose here
	if err != nil {
		return fmt.Errorf("failed to encode credentials: %w", err)
	}

	// Store in Windows Credential Manager using go-keyring
	if err := keyring.Set(p.service, profileName, string(data)); err != nil {
		return fmt.Errorf("failed to store credentials in credential manager: %w", err)
	}

	return nil
}

func (p *SecureCredentialProvider) removeFromCredentialManager(profileName string) error {
	// Remove from Windows Credential Manager using go-keyring
	if err := keyring.Delete(p.service, profileName); err != nil {
		// If the item doesn't exist, that's okay
		if err == keyring.ErrNotFound {
			return nil
		}
		return fmt.Errorf("failed to remove credentials from credential manager: %w", err)
	}

	return nil
}

// Linux Secret Service implementations using go-keyring

func (p *SecureCredentialProvider) getFromSecretService(profileName string) (*Credentials, error) {
	// Retrieve from Linux Secret Service using go-keyring
	data, err := keyring.Get(p.service, profileName)
	if err != nil {
		return nil, fmt.Errorf("failed to get credentials from secret service: %w", err)
	}

	// Decode JSON credentials
	var creds Credentials
	if err := json.Unmarshal([]byte(data), &creds); err != nil {
		return nil, fmt.Errorf("failed to decode credentials: %w", err)
	}

	return &creds, nil
}

func (p *SecureCredentialProvider) storeInSecretService(profileName string, creds *Credentials) error {
	// Encode credentials as JSON
	data, err := json.Marshal(creds) //nolint:gosec // G117: serializing credentials for the OS secure store is the purpose here
	if err != nil {
		return fmt.Errorf("failed to encode credentials: %w", err)
	}

	// Store in Linux Secret Service using go-keyring
	if err := keyring.Set(p.service, profileName, string(data)); err != nil {
		return fmt.Errorf("failed to store credentials in secret service: %w", err)
	}

	return nil
}

func (p *SecureCredentialProvider) removeFromSecretService(profileName string) error {
	// Remove from Linux Secret Service using go-keyring
	if err := keyring.Delete(p.service, profileName); err != nil {
		// If the item doesn't exist, that's okay
		if err == keyring.ErrNotFound {
			return nil
		}
		return fmt.Errorf("failed to remove credentials from secret service: %w", err)
	}

	return nil
}

// Local file fallback implementations with NaCl secretbox encryption
//
// The fallback file is used only when the OS keychain can't hold the credentials themselves. Its
// key is never derived from public data (#738): the original scheme used PBKDF2 over the home
// directory path and service name with a static salt, so anyone who could read a .creds file could
// derive the key -- obfuscation, not encryption. Files in that legacy format are still read, and
// rewritten in the current format on first read. The key is now one of:
//
//   - a random 256-bit key held in the OS keychain (mode 'k'), or
//   - a key stretched by scrypt, with a per-file random salt, from a passphrase the user supplies
//     in PRISM_CREDENTIALS_PASSPHRASE (mode 'p') -- for headless hosts with no keychain.
//
// With neither available, storing to the file fails rather than writing a derivable ciphertext.
//
// File format: credFileMagic | mode byte | [16-byte salt if mode 'p'] | 24-byte nonce | ciphertext.

const nonceSize = 24
const keySize = 32

const (
	credFileMagic                  = "PRISMCRED2" //nolint:gosec // G101: file-format marker, not a credential
	keyModeKeychain           byte = 'k'
	keyModePassphrase         byte = 'p'
	passphraseSaltSize             = 16
	credentialsPassEnv             = "PRISM_CREDENTIALS_PASSPHRASE" //nolint:gosec // G101: name of the environment variable, not a credential
	credentialsFileKeyAccount      = "credentials-file-key"
)

// keychainFileKey returns the random file-encryption key held in the OS keychain, creating it when
// create is set and none exists yet.
func (p *SecureCredentialProvider) keychainFileKey(create bool) ([keySize]byte, error) {
	var key [keySize]byte
	stored, err := keyring.Get(p.service, credentialsFileKeyAccount)
	if err == nil {
		raw, decodeErr := hex.DecodeString(stored)
		if decodeErr != nil || len(raw) != keySize {
			return key, fmt.Errorf("credentials file key in the keychain is malformed")
		}
		copy(key[:], raw)
		return key, nil
	}
	if !create || err != keyring.ErrNotFound {
		return key, fmt.Errorf("keychain unavailable: %w", err)
	}
	if _, err := io.ReadFull(rand.Reader, key[:]); err != nil {
		return key, fmt.Errorf("failed to generate credentials file key: %w", err)
	}
	if err := keyring.Set(p.service, credentialsFileKeyAccount, hex.EncodeToString(key[:])); err != nil {
		return key, fmt.Errorf("keychain unavailable: %w", err)
	}
	return key, nil
}

// passphraseFileKey stretches the user's passphrase into a file key with scrypt.
func passphraseFileKey(passphrase string, salt []byte) ([keySize]byte, error) {
	var key [keySize]byte
	derived, err := scrypt.Key([]byte(passphrase), salt, 1<<15, 8, 1, keySize)
	if err != nil {
		return key, fmt.Errorf("failed to derive key from %s: %w", credentialsPassEnv, err)
	}
	copy(key[:], derived)
	return key, nil
}

// legacyEncryptionKey is the derivable key of the original file format. It is used only to read
// files written before #738 so they can be migrated.
func (p *SecureCredentialProvider) legacyEncryptionKey() ([keySize]byte, error) {
	var key [keySize]byte
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return key, fmt.Errorf("failed to get home directory: %w", err)
	}
	derivedKey := pbkdf2.Key([]byte(homeDir+p.service), []byte("prism-v1"), 100000, 32, sha256.New)
	copy(key[:], derivedKey)
	return key, nil
}

func openSealed(data []byte, key *[keySize]byte) ([]byte, error) {
	if len(data) < nonceSize {
		return nil, fmt.Errorf("invalid encrypted data: too short")
	}
	var nonce [nonceSize]byte
	copy(nonce[:], data[:nonceSize])
	plaintext, ok := secretbox.Open(nil, data[nonceSize:], &nonce, key)
	if !ok {
		return nil, fmt.Errorf("failed to decrypt credentials")
	}
	return plaintext, nil
}

func (p *SecureCredentialProvider) getFromFile(profileName string) (*Credentials, error) {
	credFile := filepath.Join(p.configPath, profileName+".creds")

	// Read encrypted file
	encryptedData, err := os.ReadFile(credFile) //nolint:gosec // G304: profile name from local config/CLI, joined under the user's credentials dir
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("credentials not found for profile %s", profileName)
		}
		return nil, fmt.Errorf("failed to read credentials file: %w", err)
	}

	var plaintext []byte
	legacy := !bytes.HasPrefix(encryptedData, []byte(credFileMagic))
	if legacy {
		key, err := p.legacyEncryptionKey()
		if err != nil {
			return nil, fmt.Errorf("failed to get encryption key: %w", err)
		}
		if plaintext, err = openSealed(encryptedData, &key); err != nil {
			return nil, err
		}
	} else {
		rest := encryptedData[len(credFileMagic):]
		if len(rest) < 1 {
			return nil, fmt.Errorf("invalid encrypted data: too short")
		}
		mode, rest := rest[0], rest[1:]
		var key [keySize]byte
		switch mode {
		case keyModeKeychain:
			if key, err = p.keychainFileKey(false); err != nil {
				return nil, fmt.Errorf("credentials for %s are encrypted with a key held in the OS keychain, which is not readable: %w", profileName, err)
			}
		case keyModePassphrase:
			passphrase := os.Getenv(credentialsPassEnv)
			if passphrase == "" {
				return nil, fmt.Errorf("credentials for %s are passphrase-encrypted: set %s", profileName, credentialsPassEnv)
			}
			if len(rest) < passphraseSaltSize {
				return nil, fmt.Errorf("invalid encrypted data: too short")
			}
			if key, err = passphraseFileKey(passphrase, rest[:passphraseSaltSize]); err != nil {
				return nil, err
			}
			rest = rest[passphraseSaltSize:]
		default:
			return nil, fmt.Errorf("invalid encrypted data: unknown key mode %q", mode)
		}
		if plaintext, err = openSealed(rest, &key); err != nil {
			return nil, err
		}
	}

	// Decode JSON credentials
	var creds Credentials
	if err := json.Unmarshal(plaintext, &creds); err != nil {
		return nil, fmt.Errorf("failed to decode credentials: %w", err)
	}

	if legacy {
		// Rewrite with a key that can't be derived. If no such key is available yet (no keychain,
		// no passphrase), keep the readable legacy file and say why it wasn't migrated.
		if err := p.storeInFile(profileName, &creds); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: credentials for %s use the old, weakly protected file format and could not be re-encrypted: %v\n", profileName, err)
		}
	}

	return &creds, nil
}

func (p *SecureCredentialProvider) storeInFile(profileName string, creds *Credentials) error {
	credFile := filepath.Join(p.configPath, profileName+".creds")

	// Encode credentials as JSON
	plaintext, err := json.Marshal(creds) //nolint:gosec // G117: plaintext is encrypted with secretbox before being written
	if err != nil {
		return fmt.Errorf("failed to encode credentials: %w", err)
	}

	// Choose the key: an explicit passphrase wins; otherwise a random key held in the keychain.
	header := []byte(credFileMagic)
	var key [keySize]byte
	if passphrase := os.Getenv(credentialsPassEnv); passphrase != "" {
		salt := make([]byte, passphraseSaltSize)
		if _, err := io.ReadFull(rand.Reader, salt); err != nil {
			return fmt.Errorf("failed to generate salt: %w", err)
		}
		if key, err = passphraseFileKey(passphrase, salt); err != nil {
			return err
		}
		header = append(append(header, keyModePassphrase), salt...)
	} else {
		if key, err = p.keychainFileKey(true); err != nil {
			return fmt.Errorf("no secure key storage for the credentials file (%w); set %s to encrypt it with a passphrase, or use AWS CLI profiles", err, credentialsPassEnv)
		}
		header = append(header, keyModeKeychain)
	}

	// Generate random nonce
	var nonce [nonceSize]byte
	if _, err := io.ReadFull(rand.Reader, nonce[:]); err != nil {
		return fmt.Errorf("failed to generate nonce: %w", err)
	}

	// Encrypt using NaCl secretbox
	header = append(header, nonce[:]...)
	encryptedData := secretbox.Seal(header, plaintext, &nonce, &key)

	// Write to file with restrictive permissions
	if err := os.WriteFile(credFile, encryptedData, 0600); err != nil {
		return fmt.Errorf("failed to write credentials file: %w", err)
	}

	return nil
}

func (p *SecureCredentialProvider) removeFromFile(profileName string) error {
	credFile := filepath.Join(p.configPath, profileName+".creds")
	if _, err := os.Stat(credFile); os.IsNotExist(err) {
		return nil // Already removed
	}
	return os.Remove(credFile)
}

// AWSCredentialsProvider adapts our credentials to AWS SDK
type AWSCredentialsProvider struct {
	provider      CredentialProvider
	profileName   string
	expirationCh  chan bool
	lastRefreshed time.Time
}

// NewAWSCredentialsProvider creates a new AWS credentials provider for a profile
func NewAWSCredentialsProvider(provider CredentialProvider, profileName string) *AWSCredentialsProvider {
	return &AWSCredentialsProvider{
		provider:     provider,
		profileName:  profileName,
		expirationCh: make(chan bool, 1),
	}
}

// Retrieve implements the aws.CredentialsProvider interface
func (p *AWSCredentialsProvider) Retrieve(ctx interface{}) (aws.Credentials, error) {
	creds, err := p.provider.GetCredentials(p.profileName)
	if err != nil {
		return aws.Credentials{}, err
	}

	p.lastRefreshed = time.Now()

	var expiry time.Time
	if creds.Expiration != nil {
		expiry = *creds.Expiration

		// Set up expiration notification
		go func() {
			timer := time.NewTimer(time.Until(expiry) - 5*time.Minute)
			<-timer.C
			select {
			case p.expirationCh <- true:
				// Sent notification
			default:
				// Channel already has notification
			}
		}()
	}

	return aws.Credentials{
		AccessKeyID:     creds.AccessKeyID,
		SecretAccessKey: creds.SecretAccessKey,
		SessionToken:    creds.SessionToken,
		Source:          "Prism",
		CanExpire:       creds.Expiration != nil,
		Expires:         expiry,
	}, nil
}

// EncodeCredsForDisplay encodes credentials for display (masking sensitive data)
func EncodeCredsForDisplay(creds *Credentials) string {
	if creds == nil {
		return "No credentials"
	}

	masked := fmt.Sprintf("AccessKeyID: %s****", creds.AccessKeyID[:min(4, len(creds.AccessKeyID))])
	if creds.SessionToken != "" {
		masked += " (with session token)"
	}
	if creds.Expiration != nil {
		masked += fmt.Sprintf(" expires: %s", creds.Expiration.Format(time.RFC3339))
	}

	return masked
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// Base64EncodeCredentials encodes credentials to base64 for transport
func Base64EncodeCredentials(creds *Credentials) (string, error) {
	data, err := json.Marshal(creds) //nolint:gosec // G117: explicit credential encoding for transport is this function's purpose
	if err != nil {
		return "", fmt.Errorf("failed to marshal credentials: %w", err)
	}
	return base64.StdEncoding.EncodeToString(data), nil
}

// Base64DecodeCredentials decodes credentials from base64
func Base64DecodeCredentials(encoded string) (*Credentials, error) {
	data, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return nil, fmt.Errorf("failed to decode base64: %w", err)
	}

	var creds Credentials
	if err := json.Unmarshal(data, &creds); err != nil {
		return nil, fmt.Errorf("failed to unmarshal credentials: %w", err)
	}

	return &creds, nil
}
