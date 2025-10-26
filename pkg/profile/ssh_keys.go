package profile

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/crypto/ssh"
)

// SSHKeyManager handles SSH key operations for profiles
type SSHKeyManager struct {
	homeDir string
}

// NewSSHKeyManager creates a new SSH key manager
func NewSSHKeyManager() (*SSHKeyManager, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}

	return &SSHKeyManager{
		homeDir: homeDir,
	}, nil
}

// GetSSHKeyForProfile returns the SSH key configuration for a profile
func (m *SSHKeyManager) GetSSHKeyForProfile(profile *Profile) (keyPath string, keyName string, err error) {
	if profile.UseDefaultKey {
		return m.getDefaultSSHKey()
	}

	if profile.SSHKeyPath != "" && profile.SSHKeyName != "" {
		// Use explicitly configured key
		return profile.SSHKeyPath, profile.SSHKeyName, nil
	}

	// Generate Prism-specific key for this profile
	return m.getOrCreateProfileKey(profile)
}

// getDefaultSSHKey finds the user's default SSH key
func (m *SSHKeyManager) getDefaultSSHKey() (string, string, error) {
	sshDir := filepath.Join(m.homeDir, ".ssh")

	// Check common default key types in order of preference
	keyTypes := []string{"id_ed25519", "id_rsa", "id_ecdsa"}

	for _, keyType := range keyTypes {
		privateKeyPath := filepath.Join(sshDir, keyType)
		publicKeyPath := privateKeyPath + ".pub"

		if m.keyPairExists(privateKeyPath, publicKeyPath) {
			// Extract key name from public key for AWS
			keyName, err := m.extractKeyNameFromPublicKey(publicKeyPath)
			if err != nil {
				continue // Try next key type
			}
			return privateKeyPath, keyName, nil
		}
	}

	return "", "", fmt.Errorf("no default SSH key found in %s", sshDir)
}

// getOrCreateProfileKey gets or creates a Prism-specific key for the profile
func (m *SSHKeyManager) getOrCreateProfileKey(profile *Profile) (string, string, error) {
	keyName := m.generateKeyName(profile)
	// Use the same safe naming for file paths as for AWS key names
	safeName := strings.ToLower(profile.Name)
	safeName = strings.ReplaceAll(safeName, " ", "-")
	safeName = strings.ReplaceAll(safeName, "_", "-")
	safeName = strings.ReplaceAll(safeName, "'", "")
	safeName = strings.ReplaceAll(safeName, "\"", "")

	privateKeyPath := filepath.Join(m.homeDir, ".ssh", fmt.Sprintf("cws-%s-key", safeName))
	publicKeyPath := privateKeyPath + ".pub"

	if m.keyPairExists(privateKeyPath, publicKeyPath) {
		return privateKeyPath, keyName, nil
	}

	// Generate new key pair
	if err := m.generateKeyPair(privateKeyPath, publicKeyPath); err != nil {
		return "", "", fmt.Errorf("failed to generate key pair: %w", err)
	}

	return privateKeyPath, keyName, nil
}

// generateKeyName creates a unique AWS key pair name for the profile
func (m *SSHKeyManager) generateKeyName(profile *Profile) string {
	// Create a safe key name for AWS and filesystem
	safeName := strings.ToLower(profile.Name)
	safeName = strings.ReplaceAll(safeName, " ", "-")
	safeName = strings.ReplaceAll(safeName, "_", "-")
	// Remove any other problematic characters
	safeName = strings.ReplaceAll(safeName, "'", "")
	safeName = strings.ReplaceAll(safeName, "\"", "")
	return fmt.Sprintf("cws-%s-key", safeName)
}

// keyPairExists checks if both private and public key files exist
func (m *SSHKeyManager) keyPairExists(privateKeyPath, publicKeyPath string) bool {
	if _, err := os.Stat(privateKeyPath); os.IsNotExist(err) {
		return false
	}
	if _, err := os.Stat(publicKeyPath); os.IsNotExist(err) {
		return false
	}
	return true
}

// generateKeyPair generates a new RSA key pair
func (m *SSHKeyManager) generateKeyPair(privateKeyPath, publicKeyPath string) error {
	// Ensure .ssh directory exists
	sshDir := filepath.Dir(privateKeyPath)
	if err := os.MkdirAll(sshDir, 0700); err != nil {
		return fmt.Errorf("failed to create SSH directory: %w", err)
	}

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

	// Write private key
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
	publicFile, err := os.Create(publicKeyPath)
	if err != nil {
		return fmt.Errorf("failed to create public key file: %w", err)
	}
	defer publicFile.Close()

	publicKeyLine := fmt.Sprintf("%s cloudworkstation-generated\n",
		strings.TrimSpace(string(ssh.MarshalAuthorizedKey(publicKey))))

	if _, err := publicFile.WriteString(publicKeyLine); err != nil {
		return fmt.Errorf("failed to write public key: %w", err)
	}

	return nil
}

// extractKeyNameFromPublicKey extracts a usable key name from a public key file
func (m *SSHKeyManager) extractKeyNameFromPublicKey(publicKeyPath string) (string, error) {
	content, err := os.ReadFile(publicKeyPath)
	if err != nil {
		return "", fmt.Errorf("failed to read public key: %w", err)
	}

	// Parse the public key
	parts := strings.Fields(string(content))
	if len(parts) < 2 {
		return "", fmt.Errorf("invalid public key format")
	}

	// Use filename as base for key name
	filename := filepath.Base(publicKeyPath)
	filename = strings.TrimSuffix(filename, ".pub")

	return fmt.Sprintf("cws-default-%s-key", filename), nil
}

// GetPublicKeyContent returns the content of the public key for AWS upload
func (m *SSHKeyManager) GetPublicKeyContent(publicKeyPath string) (string, error) {
	content, err := os.ReadFile(publicKeyPath)
	if err != nil {
		return "", fmt.Errorf("failed to read public key: %w", err)
	}

	return strings.TrimSpace(string(content)), nil
}
