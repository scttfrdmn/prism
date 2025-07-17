// Package profile provides functionality for managing CloudWorkstation profiles
package profile

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"time"
	
	"github.com/aws/aws-sdk-go-v2/aws"
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
	
	// Create CloudWorkstation directory if it doesn't exist
	cwsDir := filepath.Join(homeDir, ".cloudworkstation", "credentials")
	if err := os.MkdirAll(cwsDir, 0700); err != nil {
		return nil, fmt.Errorf("failed to create credentials directory: %w", err)
	}
	
	provider := &SecureCredentialProvider{
		service:    "CloudWorkstation",
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

// Local file fallback implementations

func (p *SecureCredentialProvider) getFromFile(profileName string) (*Credentials, error) {
	// In a real implementation, this would decrypt the stored credentials
	return nil, fmt.Errorf("credential file does not exist for profile %s", profileName)
}

func (p *SecureCredentialProvider) storeInFile(profileName string, creds *Credentials) error {
	// In a real implementation, this would encrypt the credentials before storing
	return fmt.Errorf("secure storage not available and encrypted file storage not yet implemented")
}

func (p *SecureCredentialProvider) removeFromFile(profileName string) error {
	credFile := filepath.Join(p.configPath, profileName+".creds")
	if _, err := os.Stat(credFile); os.IsNotExist(err) {
		return nil // Already removed
	}
	return os.Remove(credFile)
}

// Platform-specific implementations (stubs)

func (p *SecureCredentialProvider) getFromKeychain(profileName string) (*Credentials, error) {
	// Stub: In real implementation, this would use the macOS Keychain API
	return nil, fmt.Errorf("keychain access not implemented")
}

func (p *SecureCredentialProvider) storeInKeychain(profileName string, creds *Credentials) error {
	// Stub: In real implementation, this would use the macOS Keychain API
	return fmt.Errorf("keychain access not implemented")
}

func (p *SecureCredentialProvider) removeFromKeychain(profileName string) error {
	// Stub: In real implementation, this would use the macOS Keychain API
	return fmt.Errorf("keychain access not implemented")
}

func (p *SecureCredentialProvider) getFromCredentialManager(profileName string) (*Credentials, error) {
	// Stub: In real implementation, this would use the Windows Credential Manager API
	return nil, fmt.Errorf("credential manager access not implemented")
}

func (p *SecureCredentialProvider) storeInCredentialManager(profileName string, creds *Credentials) error {
	// Stub: In real implementation, this would use the Windows Credential Manager API
	return fmt.Errorf("credential manager access not implemented")
}

func (p *SecureCredentialProvider) removeFromCredentialManager(profileName string) error {
	// Stub: In real implementation, this would use the Windows Credential Manager API
	return fmt.Errorf("credential manager access not implemented")
}

func (p *SecureCredentialProvider) getFromSecretService(profileName string) (*Credentials, error) {
	// Stub: In real implementation, this would use the Linux Secret Service API
	return nil, fmt.Errorf("secret service access not implemented")
}

func (p *SecureCredentialProvider) storeInSecretService(profileName string, creds *Credentials) error {
	// Stub: In real implementation, this would use the Linux Secret Service API
	return fmt.Errorf("secret service access not implemented")
}

func (p *SecureCredentialProvider) removeFromSecretService(profileName string) error {
	// Stub: In real implementation, this would use the Linux Secret Service API
	return fmt.Errorf("secret service access not implemented")
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
		Source:          "CloudWorkstation",
		CanExpire:       creds.Expiration != nil,
		Expires:         expiry,
	}, nil
}