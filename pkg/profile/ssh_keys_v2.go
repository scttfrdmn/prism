package profile

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"golang.org/x/crypto/ssh"
)

// KeyMetadata tracks SSH key information and usage
type KeyMetadata struct {
	AWSKeyName string    `json:"aws_key_name"`
	Profile    string    `json:"profile"`
	Region     string    `json:"region"`
	CreatedAt  time.Time `json:"created_at"`
	KeyType    string    `json:"type"`
	Instances  []string  `json:"instances"`
	LocalPath  string    `json:"-"` // Not serialized, computed from profile
	PublicKey  string    `json:"-"` // Not serialized, loaded on demand
}

// KeyMetadataStore tracks all CloudWorkstation SSH keys
type KeyMetadataStore struct {
	Keys    map[string]*KeyMetadata `json:"keys"`
	Version string                  `json:"version"`
}

// SSHKeyManagerV2 manages SSH keys with normalized storage and naming
type SSHKeyManagerV2 struct {
	keysDir      string // ~/.cloudworkstation/keys
	metadataPath string // ~/.cloudworkstation/keys/metadata.json
}

// NewSSHKeyManagerV2 creates a new normalized SSH key manager
func NewSSHKeyManagerV2() (*SSHKeyManagerV2, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}

	keysDir := filepath.Join(homeDir, ".cloudworkstation", "keys")
	metadataPath := filepath.Join(keysDir, "metadata.json")

	// Ensure keys directory exists with proper permissions
	if err := os.MkdirAll(keysDir, 0700); err != nil {
		return nil, fmt.Errorf("failed to create keys directory: %w", err)
	}

	return &SSHKeyManagerV2{
		keysDir:      keysDir,
		metadataPath: metadataPath,
	}, nil
}

// GetOrCreateKeyForProfile gets or creates a normalized SSH key for a profile and region
func (m *SSHKeyManagerV2) GetOrCreateKeyForProfile(profileName, region string) (localPath, awsKeyName string, err error) {
	// Normalize profile name for filesystem and AWS
	safeName := m.normalizeProfileName(profileName)

	// Generate AWS key name with region for compliance/isolation
	awsKeyName = fmt.Sprintf("cws-%s-%s", safeName, region)
	localPath = filepath.Join(m.keysDir, safeName)
	publicPath := localPath + ".pub"

	// Check if key already exists
	if m.keyPairExists(localPath, publicPath) {
		// Update metadata to ensure it's current
		if err := m.updateMetadata(safeName, profileName, region, awsKeyName, nil); err != nil {
			return "", "", fmt.Errorf("failed to update metadata: %w", err)
		}
		return localPath, awsKeyName, nil
	}

	// Generate new key pair
	if err := m.generateKeyPair(localPath, publicPath); err != nil {
		return "", "", fmt.Errorf("failed to generate key pair: %w", err)
	}

	// Create metadata entry
	if err := m.updateMetadata(safeName, profileName, region, awsKeyName, []string{}); err != nil {
		return "", "", fmt.Errorf("failed to create metadata: %w", err)
	}

	return localPath, awsKeyName, nil
}

// GetKeyPath returns the local path for a profile's SSH key
func (m *SSHKeyManagerV2) GetKeyPath(profileName string) (string, error) {
	safeName := m.normalizeProfileName(profileName)
	keyPath := filepath.Join(m.keysDir, safeName)

	if _, err := os.Stat(keyPath); os.IsNotExist(err) {
		return "", fmt.Errorf("SSH key not found for profile '%s'", profileName)
	}

	return keyPath, nil
}

// GetKeyPathFromAWSKeyName converts AWS KeyName to local key path
func (m *SSHKeyManagerV2) GetKeyPathFromAWSKeyName(awsKeyName string) (string, error) {
	// Parse AWS key name: cws-<profile>-<region>
	if !strings.HasPrefix(awsKeyName, "cws-") {
		return "", fmt.Errorf("not a CloudWorkstation key: %s", awsKeyName)
	}

	// Extract profile name (everything between "cws-" and last "-<region>")
	parts := strings.SplitN(strings.TrimPrefix(awsKeyName, "cws-"), "-", 2)
	if len(parts) < 2 {
		return "", fmt.Errorf("invalid CloudWorkstation key name format: %s", awsKeyName)
	}

	profileName := parts[0]
	keyPath := filepath.Join(m.keysDir, profileName)

	if _, err := os.Stat(keyPath); os.IsNotExist(err) {
		return "", fmt.Errorf("SSH key not found for AWS key name '%s'", awsKeyName)
	}

	return keyPath, nil
}

// ListKeys returns all CloudWorkstation SSH keys
func (m *SSHKeyManagerV2) ListKeys() ([]*KeyMetadata, error) {
	store, err := m.loadMetadata()
	if err != nil {
		return nil, err
	}

	keys := make([]*KeyMetadata, 0, len(store.Keys))
	for _, key := range store.Keys {
		// Set computed fields
		key.LocalPath = filepath.Join(m.keysDir, key.Profile)

		// Load public key content
		publicKeyPath := key.LocalPath + ".pub"
		if content, err := os.ReadFile(publicKeyPath); err == nil {
			key.PublicKey = strings.TrimSpace(string(content))
		}

		keys = append(keys, key)
	}

	return keys, nil
}

// GetKeyMetadata returns metadata for a specific key
func (m *SSHKeyManagerV2) GetKeyMetadata(profileName string) (*KeyMetadata, error) {
	safeName := m.normalizeProfileName(profileName)

	store, err := m.loadMetadata()
	if err != nil {
		return nil, err
	}

	key, exists := store.Keys[safeName]
	if !exists {
		return nil, fmt.Errorf("key not found for profile '%s'", profileName)
	}

	// Set computed fields
	key.LocalPath = filepath.Join(m.keysDir, safeName)
	publicKeyPath := key.LocalPath + ".pub"
	if content, err := os.ReadFile(publicKeyPath); err == nil {
		key.PublicKey = strings.TrimSpace(string(content))
	}

	return key, nil
}

// ExportKey exports a key to a specified location
func (m *SSHKeyManagerV2) ExportKey(profileName, destPath string) error {
	safeName := m.normalizeProfileName(profileName)
	srcPath := filepath.Join(m.keysDir, safeName)

	// Verify source key exists
	if _, err := os.Stat(srcPath); os.IsNotExist(err) {
		return fmt.Errorf("SSH key not found for profile '%s'", profileName)
	}

	// Read source key
	keyContent, err := os.ReadFile(srcPath)
	if err != nil {
		return fmt.Errorf("failed to read key: %w", err)
	}

	// Ensure destination directory exists
	destDir := filepath.Dir(destPath)
	if err := os.MkdirAll(destDir, 0700); err != nil {
		return fmt.Errorf("failed to create destination directory: %w", err)
	}

	// Write to destination with proper permissions
	if err := os.WriteFile(destPath, keyContent, 0600); err != nil {
		return fmt.Errorf("failed to write key: %w", err)
	}

	return nil
}

// ImportKey imports an existing SSH key for a profile
func (m *SSHKeyManagerV2) ImportKey(profileName, region, keyFilePath string) error {
	safeName := m.normalizeProfileName(profileName)

	// Check if key already exists
	destPath := filepath.Join(m.keysDir, safeName)
	if _, err := os.Stat(destPath); err == nil {
		return fmt.Errorf("key already exists for profile '%s'", profileName)
	}

	// Validate and read the private key file
	privateKeyContent, err := os.ReadFile(keyFilePath)
	if err != nil {
		return fmt.Errorf("failed to read key file: %w", err)
	}

	// Validate it's a valid PEM-encoded private key
	block, _ := pem.Decode(privateKeyContent)
	if block == nil {
		return fmt.Errorf("invalid private key: not PEM encoded")
	}

	// Try to parse as RSA key
	var privateKey *rsa.PrivateKey
	switch block.Type {
	case "RSA PRIVATE KEY":
		privateKey, err = x509.ParsePKCS1PrivateKey(block.Bytes)
		if err != nil {
			return fmt.Errorf("failed to parse RSA private key: %w", err)
		}
	case "PRIVATE KEY":
		// PKCS8 format
		parsedKey, err := x509.ParsePKCS8PrivateKey(block.Bytes)
		if err != nil {
			return fmt.Errorf("failed to parse PKCS8 private key: %w", err)
		}
		var ok bool
		privateKey, ok = parsedKey.(*rsa.PrivateKey)
		if !ok {
			return fmt.Errorf("private key is not RSA format")
		}
	default:
		return fmt.Errorf("unsupported private key type: %s", block.Type)
	}

	// Write private key to normalized location
	if err := os.WriteFile(destPath, privateKeyContent, 0600); err != nil {
		return fmt.Errorf("failed to write private key: %w", err)
	}

	// Generate and write public key
	publicKey, err := ssh.NewPublicKey(&privateKey.PublicKey)
	if err != nil {
		// Clean up private key on failure
		os.Remove(destPath)
		return fmt.Errorf("failed to generate public key: %w", err)
	}

	publicPath := destPath + ".pub"
	publicKeyLine := fmt.Sprintf("%s cloudworkstation-imported\n",
		strings.TrimSpace(string(ssh.MarshalAuthorizedKey(publicKey))))

	if err := os.WriteFile(publicPath, []byte(publicKeyLine), 0644); err != nil {
		// Clean up on failure
		os.Remove(destPath)
		return fmt.Errorf("failed to write public key: %w", err)
	}

	// Create metadata entry
	awsKeyName := fmt.Sprintf("cws-%s-%s", safeName, region)
	if err := m.updateMetadata(safeName, profileName, region, awsKeyName, []string{}); err != nil {
		// Clean up on failure
		os.Remove(destPath)
		os.Remove(publicPath)
		return fmt.Errorf("failed to create metadata: %w", err)
	}

	return nil
}

// DeleteKey deletes a key (with safety checks for instance associations)
func (m *SSHKeyManagerV2) DeleteKey(profileName string) error {
	safeName := m.normalizeProfileName(profileName)

	// Check metadata for instance associations
	metadata, err := m.GetKeyMetadata(profileName)
	if err != nil {
		return err
	}

	if len(metadata.Instances) > 0 {
		return fmt.Errorf("key '%s' is associated with %d instance(s): %v",
			profileName, len(metadata.Instances), metadata.Instances)
	}

	// Delete key files
	keyPath := filepath.Join(m.keysDir, safeName)
	publicPath := keyPath + ".pub"

	if err := os.Remove(keyPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to delete private key: %w", err)
	}

	if err := os.Remove(publicPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to delete public key: %w", err)
	}

	// Remove from metadata
	store, err := m.loadMetadata()
	if err != nil {
		return err
	}

	delete(store.Keys, safeName)
	return m.saveMetadata(store)
}

// AddInstanceToKey associates an instance with a key
func (m *SSHKeyManagerV2) AddInstanceToKey(profileName, instanceName string) error {
	safeName := m.normalizeProfileName(profileName)

	store, err := m.loadMetadata()
	if err != nil {
		return err
	}

	key, exists := store.Keys[safeName]
	if !exists {
		return fmt.Errorf("key not found for profile '%s'", profileName)
	}

	// Check if instance already associated
	for _, inst := range key.Instances {
		if inst == instanceName {
			return nil // Already associated
		}
	}

	// Add instance
	key.Instances = append(key.Instances, instanceName)
	return m.saveMetadata(store)
}

// RemoveInstanceFromKey removes an instance association from a key
func (m *SSHKeyManagerV2) RemoveInstanceFromKey(profileName, instanceName string) error {
	safeName := m.normalizeProfileName(profileName)

	store, err := m.loadMetadata()
	if err != nil {
		return err
	}

	key, exists := store.Keys[safeName]
	if !exists {
		return nil // Key doesn't exist, nothing to do
	}

	// Remove instance
	newInstances := make([]string, 0, len(key.Instances))
	for _, inst := range key.Instances {
		if inst != instanceName {
			newInstances = append(newInstances, inst)
		}
	}

	key.Instances = newInstances
	return m.saveMetadata(store)
}

// GetPublicKeyContent returns the public key content for AWS upload
func (m *SSHKeyManagerV2) GetPublicKeyContent(profileName string) (string, error) {
	safeName := m.normalizeProfileName(profileName)
	publicKeyPath := filepath.Join(m.keysDir, safeName) + ".pub"

	content, err := os.ReadFile(publicKeyPath)
	if err != nil {
		return "", fmt.Errorf("failed to read public key: %w", err)
	}

	return strings.TrimSpace(string(content)), nil
}

// normalizeProfileName converts profile name to safe filesystem/AWS name
func (m *SSHKeyManagerV2) normalizeProfileName(profileName string) string {
	safeName := strings.ToLower(profileName)
	safeName = strings.ReplaceAll(safeName, " ", "-")
	safeName = strings.ReplaceAll(safeName, "_", "-")
	safeName = strings.ReplaceAll(safeName, "'", "")
	safeName = strings.ReplaceAll(safeName, "\"", "")
	return safeName
}

// keyPairExists checks if both private and public key files exist
func (m *SSHKeyManagerV2) keyPairExists(privateKeyPath, publicKeyPath string) bool {
	if _, err := os.Stat(privateKeyPath); os.IsNotExist(err) {
		return false
	}
	if _, err := os.Stat(publicKeyPath); os.IsNotExist(err) {
		return false
	}
	return true
}

// generateKeyPair generates a new RSA 2048 key pair
func (m *SSHKeyManagerV2) generateKeyPair(privateKeyPath, publicKeyPath string) error {
	// Generate RSA private key
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return fmt.Errorf("failed to generate RSA key: %w", err)
	}

	// Encode private key to PEM format
	privateKeyPEM := &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(privateKey),
	}

	// Write private key with secure permissions
	privateFile, err := os.OpenFile(privateKeyPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return fmt.Errorf("failed to create private key file: %w", err)
	}
	defer privateFile.Close()

	if err := pem.Encode(privateFile, privateKeyPEM); err != nil {
		return fmt.Errorf("failed to write private key: %w", err)
	}

	// Generate public key
	publicKey, err := ssh.NewPublicKey(&privateKey.PublicKey)
	if err != nil {
		return fmt.Errorf("failed to generate public key: %w", err)
	}

	// Write public key
	publicFile, err := os.OpenFile(publicKeyPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return fmt.Errorf("failed to create public key file: %w", err)
	}
	defer publicFile.Close()

	publicKeyLine := fmt.Sprintf("%s cloudworkstation-managed\n",
		strings.TrimSpace(string(ssh.MarshalAuthorizedKey(publicKey))))

	if _, err := publicFile.WriteString(publicKeyLine); err != nil {
		return fmt.Errorf("failed to write public key: %w", err)
	}

	return nil
}

// loadMetadata loads the key metadata store
func (m *SSHKeyManagerV2) loadMetadata() (*KeyMetadataStore, error) {
	// If metadata file doesn't exist, return empty store
	if _, err := os.Stat(m.metadataPath); os.IsNotExist(err) {
		return &KeyMetadataStore{
			Keys:    make(map[string]*KeyMetadata),
			Version: "1.0",
		}, nil
	}

	content, err := os.ReadFile(m.metadataPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read metadata: %w", err)
	}

	var store KeyMetadataStore
	if err := json.Unmarshal(content, &store); err != nil {
		return nil, fmt.Errorf("failed to parse metadata: %w", err)
	}

	// Ensure Keys map is initialized
	if store.Keys == nil {
		store.Keys = make(map[string]*KeyMetadata)
	}

	return &store, nil
}

// saveMetadata saves the key metadata store
func (m *SSHKeyManagerV2) saveMetadata(store *KeyMetadataStore) error {
	content, err := json.MarshalIndent(store, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	if err := os.WriteFile(m.metadataPath, content, 0600); err != nil {
		return fmt.Errorf("failed to write metadata: %w", err)
	}

	return nil
}

// updateMetadata updates or creates a metadata entry
func (m *SSHKeyManagerV2) updateMetadata(safeName, profileName, region, awsKeyName string, instances []string) error {
	store, err := m.loadMetadata()
	if err != nil {
		return err
	}

	key, exists := store.Keys[safeName]
	if !exists {
		// Create new entry
		key = &KeyMetadata{
			AWSKeyName: awsKeyName,
			Profile:    safeName,
			Region:     region,
			CreatedAt:  time.Now(),
			KeyType:    "rsa-2048",
			Instances:  []string{},
		}
		if instances != nil {
			key.Instances = instances
		}
		store.Keys[safeName] = key
	} else {
		// Update existing entry
		key.AWSKeyName = awsKeyName
		key.Region = region
	}

	return m.saveMetadata(store)
}
