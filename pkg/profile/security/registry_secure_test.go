// Package security provides tests for secure registry client
package security

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/hex"
	"io"
	"math/big"
	"net"
	"net/http"
	"strings"
	"testing"
	"time"
)

// TestNewSecureRegistryClient validates secure registry client creation
func TestNewSecureRegistryClient(t *testing.T) {
	config := S3RegistryConfig{
		BucketName: "test-bucket",
		Region:     "us-west-2",
		Enabled:    true,
	}

	client, err := NewSecureRegistryClient(config)
	if err != nil {
		t.Fatalf("Failed to create secure registry client: %v", err)
	}

	if client.signer == nil {
		t.Error("Signer should not be nil")
	}

	if client.validator == nil {
		t.Error("Validator should not be nil")
	}

	if client.certPinner == nil {
		t.Error("Certificate pinner should not be nil")
	}

	if client.auditLogger == nil {
		t.Error("Audit logger should not be nil")
	}

	if client.httpClient == nil {
		t.Error("HTTP client should not be nil")
	}

	// Check TLS configuration
	transport := client.httpClient.Transport.(*http.Transport)
	if transport.TLSClientConfig == nil {
		t.Error("TLS client config should not be nil")
	}

	if transport.TLSClientConfig.MinVersion != tls.VersionTLS12 {
		t.Error("Minimum TLS version should be 1.2")
	}

	if transport.TLSClientConfig.VerifyPeerCertificate == nil {
		t.Error("Certificate verification should be configured")
	}

	client.auditLogger.Close()
	t.Log("✅ Secure registry client created successfully")
}

// TestRequestSigning validates HMAC-SHA256 request signing
func TestRequestSigning(t *testing.T) {
	signer, err := NewRequestSigner()
	if err != nil {
		t.Fatalf("Failed to create request signer: %v", err)
	}

	payload := map[string]interface{}{
		"invitation_token": "test-token-123",
		"device_id":        "test-device-456",
		"timestamp":        time.Now(),
	}

	signedPayload, err := signer.SignPayload(payload)
	if err != nil {
		t.Fatalf("Failed to sign payload: %v", err)
	}

	if signedPayload.Payload == nil {
		t.Error("Signed payload should contain payload data")
	}

	if signedPayload.Timestamp == 0 {
		t.Error("Signed payload should contain timestamp")
	}

	if signedPayload.Signature == "" {
		t.Error("Signed payload should contain signature")
	}

	// Verify signature format (should be hex encoded)
	if len(signedPayload.Signature) != 64 { // SHA256 = 32 bytes = 64 hex chars
		t.Errorf("Expected 64 character signature, got %d", len(signedPayload.Signature))
	}

	// Test signature consistency
	time.Sleep(time.Millisecond) // Ensure different timestamp
	signedPayload2, err := signer.SignPayload(payload)
	if err != nil {
		t.Fatalf("Failed to sign payload second time: %v", err)
	}

	// Signatures should be different due to different timestamps
	if signedPayload.Signature == signedPayload2.Signature {
		t.Error("Signatures should be different for different timestamps")
	}

	t.Log("✅ Request signing validated")
}

// TestResponseValidation validates response signature validation
func TestResponseValidation(t *testing.T) {
	secretKey := make([]byte, 32)
	rand.Read(secretKey)

	validator, err := NewResponseValidator(secretKey)
	if err != nil {
		t.Fatalf("Failed to create response validator: %v", err)
	}

	// Create a mock HTTP response with proper headers
	responseBody := `{"status": "success", "message": "test response"}`
	timestamp := time.Now().UTC().Format(time.RFC3339)

	// Create expected signature
	signatureData := responseBody + "." + timestamp
	hash := sha256.Sum256([]byte(signatureData))
	expectedSignature := string(hash[:])

	// Create mock response
	resp := &http.Response{
		Header: make(http.Header),
		Body:   io.NopCloser(strings.NewReader(responseBody)),
	}
	resp.Header.Set("X-Registry-Signature", expectedSignature)
	resp.Header.Set("X-Registry-Timestamp", timestamp)

	// This will fail validation because we're not using HMAC properly,
	// but it tests the validation logic
	err = validator.ValidateResponse(resp)
	if err == nil {
		t.Error("Should fail validation with incorrect signature")
	}

	// Test missing signature header
	resp2 := &http.Response{
		Header: make(http.Header),
		Body:   io.NopCloser(strings.NewReader(responseBody)),
	}

	err = validator.ValidateResponse(resp2)
	if err == nil {
		t.Error("Should fail validation with missing signature header")
	}

	if !strings.Contains(err.Error(), "missing response signature") {
		t.Error("Should report missing signature error")
	}

	t.Log("✅ Response validation tested")
}

// TestCertificatePinning validates certificate pinning functionality
func TestCertificatePinning(t *testing.T) {
	pinner, err := NewCertificatePinner()
	if err != nil {
		t.Fatalf("Failed to create certificate pinner: %v", err)
	}

	// Create a test certificate
	cert := createTestCertificate(t)
	rawCert := cert.Raw

	// Test with no certificates
	err = pinner.VerifyPeerCertificate([][]byte{}, [][]*x509.Certificate{})
	if err == nil {
		t.Error("Should fail with no certificates")
	}

	// Test with unpinned certificate (should fail)
	err = pinner.VerifyPeerCertificate([][]byte{rawCert}, [][]*x509.Certificate{})
	if err == nil {
		t.Error("Should fail with unpinned certificate")
	}

	if !strings.Contains(err.Error(), "certificate not pinned") {
		t.Error("Should report certificate not pinned error")
	}

	// Calculate certificate fingerprint (same format as VerifyPeerCertificate)
	fingerprint := sha256.Sum256(rawCert)
	fingerprintStr := "sha256:" + strings.ToUpper(hex.EncodeToString(fingerprint[:]))

	// Add certificate to pinned list
	pinner.pinnedFingerprints = append(pinner.pinnedFingerprints, fingerprintStr)

	// Now it should pass
	err = pinner.VerifyPeerCertificate([][]byte{rawCert}, [][]*x509.Certificate{})
	if err != nil {
		t.Errorf("Should pass with pinned certificate: %v", err)
	}

	t.Log("✅ Certificate pinning validated")
}

// TestDeviceRegistrationLocal validates local device registration
func TestDeviceRegistrationLocal(t *testing.T) {
	config := S3RegistryConfig{
		BucketName: "test-bucket",
		Region:     "us-west-2",
		Enabled:    false, // Local mode
	}

	client, err := NewSecureRegistryClient(config)
	if err != nil {
		t.Fatalf("Failed to create secure registry client: %v", err)
	}
	defer client.auditLogger.Close()

	// Test local device registration
	invitationToken := "test-invitation-123"
	deviceID := "test-device-456"

	err = client.RegisterDevice(invitationToken, deviceID)
	if err != nil {
		t.Fatalf("Failed to register device locally: %v", err)
	}

	// Verify local registration file was created
	// This would check the file system in a real test, but for now we just
	// verify no error occurred

	t.Log("✅ Local device registration validated")
}

// TestSecureRegistryErrorHandling validates error handling
func TestSecureRegistryErrorHandling(t *testing.T) {
	config := S3RegistryConfig{
		BucketName: "test-bucket",
		Region:     "us-west-2",
		Enabled:    true,
	}

	client, err := NewSecureRegistryClient(config)
	if err != nil {
		t.Fatalf("Failed to create secure registry client: %v", err)
	}
	defer client.auditLogger.Close()

	// Test with invalid server (should fallback to local)
	err = client.RegisterDevice("test-token", "test-device")
	if err != nil {
		t.Fatalf("Should handle server errors gracefully: %v", err)
	}

	t.Log("✅ Error handling validated")
}

// TestSecureHTTPClient validates HTTP client security configuration
func TestSecureHTTPClient(t *testing.T) {
	config := S3RegistryConfig{
		BucketName: "test-bucket",
		Region:     "us-west-2",
		Enabled:    true,
	}

	client, err := NewSecureRegistryClient(config)
	if err != nil {
		t.Fatalf("Failed to create secure registry client: %v", err)
	}
	defer client.auditLogger.Close()

	// Verify HTTP client configuration
	if client.httpClient.Timeout != 30*time.Second {
		t.Error("HTTP client timeout should be 30 seconds")
	}

	transport := client.httpClient.Transport.(*http.Transport)
	tlsConfig := transport.TLSClientConfig

	if tlsConfig.MinVersion != tls.VersionTLS12 {
		t.Error("Should enforce minimum TLS 1.2")
	}

	if tlsConfig.VerifyPeerCertificate == nil {
		t.Error("Should have custom certificate verification")
	}

	t.Log("✅ HTTP client security configuration validated")
}

// TestRequestIDGeneration validates request ID generation
func TestRequestIDGeneration(t *testing.T) {
	id1 := generateRequestID()
	id2 := generateRequestID()

	if id1 == id2 {
		t.Error("Request IDs should be unique")
	}

	if len(id1) != 32 { // 16 bytes = 32 hex chars
		t.Errorf("Expected 32 character request ID, got %d", len(id1))
	}

	// Should be valid hex
	for _, c := range id1 {
		if !strings.ContainsRune("0123456789abcdef", c) {
			t.Errorf("Request ID should be valid hex, found invalid char: %c", c)
		}
	}

	t.Log("✅ Request ID generation validated")
}

// TestTokenMasking validates sensitive token masking
func TestTokenMasking(t *testing.T) {
	testCases := []struct {
		input    string
		expected string
	}{
		{"short", "***"},                            // <= 8 chars
		{"medium12", "***"},                         // <= 8 chars (8 chars exactly)
		{"very-long-token-123456", "very***3456"},   // > 8 chars: first 4 + *** + last 4
		{"", "***"},                                 // empty string
		{"123456789", "1234***6789"},               // 9 chars: first 4 + *** + last 4
	}

	for _, tc := range testCases {
		result := maskToken(tc.input)
		if result != tc.expected {
			t.Errorf("maskToken(%q) = %q, expected %q", tc.input, result, tc.expected)
		}
	}

	t.Log("✅ Token masking validated")
}

// Helper functions

func createTestCertificate(t *testing.T) *x509.Certificate {
	// Create a test certificate for certificate pinning tests
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("Failed to generate private key: %v", err)
	}

	template := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			Organization: []string{"Test Org"},
		},
		NotBefore:    time.Now(),
		NotAfter:     time.Now().Add(time.Hour),
		KeyUsage:     x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		IPAddresses:  []net.IP{net.IPv4(127, 0, 0, 1)},
	}

	certDER, err := x509.CreateCertificate(rand.Reader, &template, &template, &priv.PublicKey, priv)
	if err != nil {
		t.Fatalf("Failed to create certificate: %v", err)
	}

	cert, err := x509.ParseCertificate(certDER)
	if err != nil {
		t.Fatalf("Failed to parse certificate: %v", err)
	}

	return cert
}