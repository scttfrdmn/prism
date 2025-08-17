package security

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/user"
	"path/filepath"
	"time"
)

// S3RegistryConfig contains configuration for the S3 registry
type S3RegistryConfig struct {
	BucketName string
	Region     string
	LocalCache string
	Enabled    bool
}

// RegistryClient handles communication with the invitation registry
type RegistryClient struct {
	config     S3RegistryConfig
	httpClient *http.Client
	localMode  bool
}

// NewRegistryClient creates a new registry client
func NewRegistryClient(config S3RegistryConfig) (*RegistryClient, error) {
	// Ensure local cache directory exists
	if config.LocalCache == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("failed to get home directory: %w", err)
		}

		config.LocalCache = filepath.Join(homeDir, ".cloudworkstation", "registry-cache")
	}

	if err := os.MkdirAll(config.LocalCache, 0755); err != nil {
		return nil, fmt.Errorf("failed to create registry cache directory: %w", err)
	}

	client := &RegistryClient{
		config:     config,
		httpClient: &http.Client{Timeout: 10 * time.Second},
		localMode:  !config.Enabled,
	}

	return client, nil
}

// RegisterDevice registers a device with the registry
func (c *RegistryClient) RegisterDevice(invitationToken, deviceID string) error {
	// Create registration data
	registration := map[string]interface{}{
		"invitation_token": invitationToken,
		"device_id":        deviceID,
		"timestamp":        time.Now().UTC().Format(time.RFC3339),
		"hostname":         getHostnameOrUnknown(),
		"username":         getUserName(),
	}

	// If in local mode, just cache the registration
	if c.localMode {
		return c.saveLocalRegistration(invitationToken, deviceID, registration)
	}

	// Otherwise, send to S3 registry
	data, err := json.Marshal(registration)
	if err != nil {
		return fmt.Errorf("failed to marshal registration data: %w", err)
	}

	// API endpoint would be provided by a config environment variable
	apiURL := os.Getenv("CWS_REGISTRY_API")
	if apiURL == "" {
		// Fall back to local mode
		return c.saveLocalRegistration(invitationToken, deviceID, registration)
	}

	// Send registration to API
	req, err := http.NewRequest("POST", fmt.Sprintf("%s/register", apiURL), bytes.NewBuffer(data))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		// If API call fails, fall back to local cache
		_ = c.saveLocalRegistration(invitationToken, deviceID, registration)
		return fmt.Errorf("failed to register with API (using local cache): %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		// If API returns error, fall back to local cache
		_ = c.saveLocalRegistration(invitationToken, deviceID, registration)
		return fmt.Errorf("API returned error (using local cache): %s", resp.Status)
	}

	return nil
}

// ValidateDevice checks if a device is registered
func (c *RegistryClient) ValidateDevice(invitationToken, deviceID string) (bool, error) {
	// If in local mode, check local registration
	if c.localMode {
		exists, _ := c.checkLocalRegistration(invitationToken, deviceID)
		return exists, nil
	}

	// Otherwise, check with S3 registry
	apiURL := os.Getenv("CWS_REGISTRY_API")
	if apiURL == "" {
		// Fall back to local mode
		exists, _ := c.checkLocalRegistration(invitationToken, deviceID)
		return exists, nil
	}

	// Send validation request to API
	url := fmt.Sprintf("%s/validate?token=%s&device=%s", apiURL, invitationToken, deviceID)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return false, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		// If API call fails, fall back to local validation
		exists, _ := c.checkLocalRegistration(invitationToken, deviceID)
		return exists, nil
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		// If API returns error, fall back to local validation
		exists, _ := c.checkLocalRegistration(invitationToken, deviceID)
		return exists, nil
	}

	// Parse response
	var result struct {
		Valid bool `json:"valid"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return false, fmt.Errorf("failed to parse API response: %w", err)
	}

	return result.Valid, nil
}

// GetInvitationDevices gets the list of devices registered for an invitation
func (c *RegistryClient) GetInvitationDevices(invitationToken string) ([]map[string]interface{}, error) {
	// If in local mode, read from local cache
	if c.localMode {
		return c.getLocalDevices(invitationToken)
	}

	// Otherwise, get from S3 registry
	apiURL := os.Getenv("CWS_REGISTRY_API")
	if apiURL == "" {
		// Fall back to local mode
		return c.getLocalDevices(invitationToken)
	}

	// Send request to API
	url := fmt.Sprintf("%s/devices?token=%s", apiURL, invitationToken)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		// If API call fails, fall back to local data
		devices, _ := c.getLocalDevices(invitationToken)
		return devices, nil
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		// If API returns error, fall back to local data
		devices, _ := c.getLocalDevices(invitationToken)
		return devices, nil
	}

	// Parse response
	var result struct {
		Devices []map[string]interface{} `json:"devices"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to parse API response: %w", err)
	}

	return result.Devices, nil
}

// RevokeDevice revokes a device from the registry
func (c *RegistryClient) RevokeDevice(invitationToken, deviceID string) error {
	// If in local mode, remove from local cache
	if c.localMode {
		return c.removeLocalRegistration(invitationToken, deviceID)
	}

	// Otherwise, revoke from S3 registry
	apiURL := os.Getenv("CWS_REGISTRY_API")
	if apiURL == "" {
		// Fall back to local mode
		return c.removeLocalRegistration(invitationToken, deviceID)
	}

	// Send revocation to API
	url := fmt.Sprintf("%s/revoke?token=%s&device=%s", apiURL, invitationToken, deviceID)
	req, err := http.NewRequest("DELETE", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		// If API call fails, fall back to local revocation
		_ = c.removeLocalRegistration(invitationToken, deviceID)
		return fmt.Errorf("failed to revoke with API (using local cache): %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		// If API returns error, fall back to local revocation
		_ = c.removeLocalRegistration(invitationToken, deviceID)
		return fmt.Errorf("API returned error (using local cache): %s", resp.Status)
	}

	return nil
}

// RevokeInvitation revokes an entire invitation
func (c *RegistryClient) RevokeInvitation(invitationToken string) error {
	// If in local mode, remove from local cache
	if c.localMode {
		return c.removeAllLocalRegistrations(invitationToken)
	}

	// Otherwise, revoke from S3 registry
	apiURL := os.Getenv("CWS_REGISTRY_API")
	if apiURL == "" {
		// Fall back to local mode
		return c.removeAllLocalRegistrations(invitationToken)
	}

	// Send revocation to API
	url := fmt.Sprintf("%s/revoke-all?token=%s", apiURL, invitationToken)
	req, err := http.NewRequest("DELETE", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		// If API call fails, fall back to local revocation
		_ = c.removeAllLocalRegistrations(invitationToken)
		return fmt.Errorf("failed to revoke with API (using local cache): %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		// If API returns error, fall back to local revocation
		_ = c.removeAllLocalRegistrations(invitationToken)
		return fmt.Errorf("API returned error (using local cache): %s", resp.Status)
	}

	return nil
}

// Local cache implementation

func (c *RegistryClient) saveLocalRegistration(invitationToken, deviceID string, data interface{}) error {
	// Create invitation directory if needed
	invitationDir := filepath.Join(c.config.LocalCache, invitationToken)
	if err := os.MkdirAll(invitationDir, 0755); err != nil {
		return fmt.Errorf("failed to create invitation directory: %w", err)
	}

	// Convert data to JSON
	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal registration data: %w", err)
	}

	// Write to file
	deviceFile := filepath.Join(invitationDir, fmt.Sprintf("%s.json", deviceID))
	if err := os.WriteFile(deviceFile, jsonData, 0644); err != nil {
		return fmt.Errorf("failed to write registration file: %w", err)
	}

	return nil
}

func (c *RegistryClient) checkLocalRegistration(invitationToken, deviceID string) (bool, error) {
	deviceFile := filepath.Join(c.config.LocalCache, invitationToken, fmt.Sprintf("%s.json", deviceID))
	_, err := os.Stat(deviceFile)
	return err == nil, nil
}

func (c *RegistryClient) removeLocalRegistration(invitationToken, deviceID string) error {
	deviceFile := filepath.Join(c.config.LocalCache, invitationToken, fmt.Sprintf("%s.json", deviceID))
	err := os.Remove(deviceFile)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove device registration: %w", err)
	}
	return nil
}

func (c *RegistryClient) removeAllLocalRegistrations(invitationToken string) error {
	invitationDir := filepath.Join(c.config.LocalCache, invitationToken)
	err := os.RemoveAll(invitationDir)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove invitation registrations: %w", err)
	}
	return nil
}

func (c *RegistryClient) getLocalDevices(invitationToken string) ([]map[string]interface{}, error) {
	invitationDir := filepath.Join(c.config.LocalCache, invitationToken)

	// Check if invitation directory exists
	if _, err := os.Stat(invitationDir); os.IsNotExist(err) {
		return []map[string]interface{}{}, nil
	}

	// Read directory
	files, err := os.ReadDir(invitationDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read invitation directory: %w", err)
	}

	// Parse device files
	var devices []map[string]interface{}

	for _, file := range files {
		if filepath.Ext(file.Name()) != ".json" {
			continue
		}

		// Read file
		data, err := os.ReadFile(filepath.Join(invitationDir, file.Name()))
		if err != nil {
			continue
		}

		// Parse JSON
		var device map[string]interface{}
		if err := json.Unmarshal(data, &device); err != nil {
			continue
		}

		devices = append(devices, device)
	}

	return devices, nil
}

// Helper function to get hostname or "unknown"
func getHostnameOrUnknown() string {
	hostname, err := os.Hostname()
	if err != nil {
		return "unknown"
	}
	return hostname
}

// Helper function to get username or "unknown"
func getUserName() string {
	currentUser, err := user.Current()
	if err != nil {
		return "unknown"
	}
	return currentUser.Username
}
