// Package security provides secure registry communication with request signing and certificate pinning
package security

import (
	"bytes"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// SecureRegistryClient provides secure communication with the invitation registry
type SecureRegistryClient struct {
	config        S3RegistryConfig
	httpClient    *http.Client
	localMode     bool
	signer        *RequestSigner
	validator     *ResponseValidator
	certPinner    *CertificatePinner
	auditLogger   *SecurityAuditLogger
}

// RequestSigner handles HMAC-SHA256 request signing
type RequestSigner struct {
	secretKey []byte
}

// ResponseValidator validates response signatures and integrity
type ResponseValidator struct {
	secretKey []byte
}

// CertificatePinner implements certificate pinning for registry connections
type CertificatePinner struct {
	pinnedFingerprints []string
	allowSelfSigned    bool
}

// NewSecureRegistryClient creates a new secure registry client with enhanced security
func NewSecureRegistryClient(config S3RegistryConfig) (*SecureRegistryClient, error) {
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

	// Initialize request signer
	signer, err := NewRequestSigner()
	if err != nil {
		return nil, fmt.Errorf("failed to create request signer: %w", err)
	}

	// Initialize response validator
	validator, err := NewResponseValidator(signer.secretKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create response validator: %w", err)
	}

	// Initialize certificate pinner
	certPinner, err := NewCertificatePinner()
	if err != nil {
		return nil, fmt.Errorf("failed to create certificate pinner: %w", err)
	}

	// Initialize audit logger
	auditLogger, err := NewSecurityAuditLogger()
	if err != nil {
		return nil, fmt.Errorf("failed to create audit logger: %w", err)
	}

	// Create HTTP client with security enhancements
	httpClient := &http.Client{
		Timeout: 30 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				MinVersion: tls.VersionTLS12,
				VerifyPeerCertificate: certPinner.VerifyPeerCertificate,
			},
		},
	}

	client := &SecureRegistryClient{
		config:      config,
		httpClient:  httpClient,
		localMode:   !config.Enabled,
		signer:      signer,
		validator:   validator,
		certPinner:  certPinner,
		auditLogger: auditLogger,
	}

	return client, nil
}

// NewRequestSigner creates a new request signer with a secure key
func NewRequestSigner() (*RequestSigner, error) {
	// Generate or retrieve signing key
	secretKey, err := getOrCreateSigningKey()
	if err != nil {
		return nil, fmt.Errorf("failed to get signing key: %w", err)
	}

	return &RequestSigner{
		secretKey: secretKey,
	}, nil
}

// NewResponseValidator creates a new response validator
func NewResponseValidator(secretKey []byte) (*ResponseValidator, error) {
	return &ResponseValidator{
		secretKey: secretKey,
	}, nil
}

// NewCertificatePinner creates a new certificate pinner with default pinned certificates
func NewCertificatePinner() (*CertificatePinner, error) {
	// Load pinned certificate fingerprints from config or use defaults
	pinnedFingerprints := []string{
		// AWS S3 certificate fingerprints (these would be real in production)
		"sha256:AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=", // Example pinned cert
		"sha256:BBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBB=", // Backup cert
	}

	return &CertificatePinner{
		pinnedFingerprints: pinnedFingerprints,
		allowSelfSigned:    false, // Strict certificate validation
	}, nil
}

// RegisterDevice securely registers a device with the registry
func (c *SecureRegistryClient) RegisterDevice(invitationToken, deviceID string) error {
	// Log security event
	event := SecurityEvent{
		EventType: "device_registration_attempt",
		DeviceID:  deviceID,
		Details: map[string]interface{}{
			"invitation_token": maskToken(invitationToken),
			"device_id":       deviceID,
		},
	}

	// If in local mode, use local storage
	if c.localMode {
		event.Success = true
		c.auditLogger.LogSecurityEvent(event)
		return c.saveLocalRegistration(invitationToken, deviceID, map[string]interface{}{
			"invitation_token": invitationToken,
			"device_id":        deviceID,
			"timestamp":        time.Now().UTC().Format(time.RFC3339),
			"hostname":         getHostnameOrUnknown(),
			"username":         getUserName(),
		})
	}

	// Create signed registration payload
	payload := RegistrationPayload{
		InvitationToken:   invitationToken,
		DeviceID:          deviceID,
		Timestamp:         time.Now().UTC(),
		DeviceFingerprint: generateSecureDeviceFingerprint(),
	}

	signedPayload, err := c.signer.SignPayload(payload)
	if err != nil {
		event.Success = false
		event.ErrorCode = "signing_failed"
		c.auditLogger.LogSecurityEvent(event)
		return fmt.Errorf("failed to sign registration payload: %w", err)
	}

	// Send secure request
	apiURL := os.Getenv("CWS_REGISTRY_API")
	if apiURL == "" {
		// Fall back to local mode
		event.Success = true
		event.Details["fallback_reason"] = "no_api_url"
		c.auditLogger.LogSecurityEvent(event)
		return c.saveLocalRegistration(invitationToken, deviceID, payload)
	}

	resp, err := c.securePost(apiURL+"/register", signedPayload)
	if err != nil {
		event.Success = false
		event.ErrorCode = "request_failed"
		c.auditLogger.LogSecurityEvent(event)
		// Fall back to local registration
		c.saveLocalRegistration(invitationToken, deviceID, payload)
		return fmt.Errorf("secure registration request failed: %w", err)
	}
	defer resp.Body.Close()

	// Validate response
	if err := c.validator.ValidateResponse(resp); err != nil {
		event.Success = false
		event.ErrorCode = "response_validation_failed"
		c.auditLogger.LogSecurityEvent(event)
		return fmt.Errorf("response validation failed: %w", err)
	}

	event.Success = true
	c.auditLogger.LogSecurityEvent(event)
	return nil
}

// SignPayload creates an HMAC-SHA256 signature for a payload
func (s *RequestSigner) SignPayload(payload interface{}) (*SignedPayload, error) {
	// Marshal payload to JSON
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal payload: %w", err)
	}

	// Create timestamp (with nanosecond precision for uniqueness)
	timestamp := time.Now().UTC().UnixNano()

	// Create signature data (payload + timestamp)
	signatureData := fmt.Sprintf("%s.%d", string(payloadBytes), timestamp)

	// Generate HMAC-SHA256 signature
	mac := hmac.New(sha256.New, s.secretKey)
	mac.Write([]byte(signatureData))
	signature := hex.EncodeToString(mac.Sum(nil))

	return &SignedPayload{
		Payload:   payloadBytes,
		Timestamp: timestamp,
		Signature: signature,
	}, nil
}

// ValidateResponse validates the integrity and authenticity of a response
func (v *ResponseValidator) ValidateResponse(resp *http.Response) error {
	// Check response signature header
	signature := resp.Header.Get("X-Registry-Signature")
	if signature == "" {
		return fmt.Errorf("missing response signature")
	}

	// Check timestamp header
	timestampStr := resp.Header.Get("X-Registry-Timestamp")
	if timestampStr == "" {
		return fmt.Errorf("missing response timestamp")
	}

	// Validate timestamp (reject old responses)
	timestamp, err := time.Parse(time.RFC3339, timestampStr)
	if err != nil {
		return fmt.Errorf("invalid timestamp format: %w", err)
	}

	if time.Since(timestamp) > 5*time.Minute {
		return fmt.Errorf("response timestamp too old")
	}

	// Read response body for signature validation
	var buf bytes.Buffer
	if _, err := buf.ReadFrom(resp.Body); err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}

	// Validate response signature
	signatureData := fmt.Sprintf("%s.%s", buf.String(), timestampStr)
	mac := hmac.New(sha256.New, v.secretKey)
	mac.Write([]byte(signatureData))
	expectedSignature := hex.EncodeToString(mac.Sum(nil))

	if !hmac.Equal([]byte(signature), []byte(expectedSignature)) {
		return fmt.Errorf("invalid response signature")
	}

	return nil
}

// VerifyPeerCertificate implements certificate pinning
func (p *CertificatePinner) VerifyPeerCertificate(rawCerts [][]byte, verifiedChains [][]*x509.Certificate) error {
	if len(rawCerts) == 0 {
		return fmt.Errorf("no certificates provided")
	}

	// Parse the first certificate
	cert, err := x509.ParseCertificate(rawCerts[0])
	if err != nil {
		return fmt.Errorf("failed to parse certificate: %w", err)
	}

	// Calculate certificate fingerprint
	fingerprint := sha256.Sum256(cert.Raw)
	fingerprintStr := "sha256:" + strings.ToUpper(hex.EncodeToString(fingerprint[:]))

	// Check against pinned certificates
	for _, pinned := range p.pinnedFingerprints {
		if strings.EqualFold(fingerprintStr, pinned) {
			return nil // Certificate is pinned
		}
	}

	// If no pinned certificates match, reject
	return fmt.Errorf("certificate not pinned: %s", fingerprintStr)
}

// securePost performs a secure POST request with signing
func (c *SecureRegistryClient) securePost(url string, payload *SignedPayload) (*http.Response, error) {
	// Create request
	reqBody, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal signed payload: %w", err)
	}

	req, err := http.NewRequest("POST", url, bytes.NewReader(reqBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "CloudWorkstation-Secure/1.0")
	req.Header.Set("X-Request-ID", generateRequestID())

	// Perform request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}

	return resp, nil
}

// Helper types and functions

type RegistrationPayload struct {
	InvitationToken   string                 `json:"invitation_token"`
	DeviceID          string                 `json:"device_id"`
	Timestamp         time.Time              `json:"timestamp"`
	DeviceFingerprint map[string]interface{} `json:"device_fingerprint"`
}

type SignedPayload struct {
	Payload   json.RawMessage `json:"payload"`
	Timestamp int64           `json:"timestamp"`
	Signature string          `json:"signature"`
}

// getOrCreateSigningKey retrieves or creates a signing key for request authentication
func getOrCreateSigningKey() ([]byte, error) {
	// Try to get key from keychain
	keychain, err := NewKeychainProvider()
	if err != nil {
		return nil, fmt.Errorf("failed to create keychain provider: %w", err)
	}

	// IMPROVED UX: Use consistent service name to avoid multiple keychain prompts
	keyName := "CloudWorkstation.registry.signing-key"
	
	// Try to retrieve existing key
	existingKey, err := keychain.Retrieve(keyName)
	if err == nil {
		return existingKey, nil
	}

	// Generate new key
	key := make([]byte, 32) // 256-bit key
	if _, err := rand.Read(key); err != nil {
		return nil, fmt.Errorf("failed to generate signing key: %w", err)
	}

	// Store key in keychain
	if err := keychain.Store(keyName, key); err != nil {
		return nil, fmt.Errorf("failed to store signing key: %w", err)
	}

	return key, nil
}

// generateSecureDeviceFingerprint creates a secure device fingerprint for registration
func generateSecureDeviceFingerprint() map[string]interface{} {
	fingerprint, err := GenerateDeviceFingerprint()
	if err != nil {
		return map[string]interface{}{"error": "fingerprint_generation_failed"}
	}

	return map[string]interface{}{
		"hostname":      fingerprint.Hostname,
		"os_version":    fingerprint.OSVersion,
		"architecture":  fingerprint.Architecture,
		"user_id":       fingerprint.UserID,
		"username":      fingerprint.Username,
		"mac_addresses": fingerprint.MACAddresses,
		"hash":          fingerprint.Hash,
		"created":       fingerprint.Created,
	}
}

// generateRequestID creates a unique request ID for tracking
func generateRequestID() string {
	requestID := make([]byte, 16)
	rand.Read(requestID)
	return hex.EncodeToString(requestID)
}

// maskToken masks sensitive token data for logging
func maskToken(token string) string {
	if len(token) <= 8 {
		return "***"
	}
	return token[:4] + "***" + token[len(token)-4:]
}

// Local storage implementation (inherited from original registry.go)
func (c *SecureRegistryClient) saveLocalRegistration(invitationToken, deviceID string, data interface{}) error {
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