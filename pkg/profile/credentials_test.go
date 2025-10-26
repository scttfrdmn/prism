package profile

import (
	"errors"
	"testing"
	"time"
)

// Define error for missing credentials
var ErrCredentialsNotFound = errors.New("credentials not found")

func TestAWSCredentialsProvider(t *testing.T) {
	// Create a mock credential provider
	mockProvider := NewMockCredentialProvider()

	// Store test credentials
	testProfile := "test-profile"
	expiry := time.Now().Add(1 * time.Hour)
	_ = mockProvider.StoreCredentials(testProfile, &Credentials{
		AccessKeyID:     "test-access-key",
		SecretAccessKey: "test-secret-key",
		SessionToken:    "test-session-token",
		Expiration:      &expiry,
	})

	// Create AWS credentials provider
	awsProvider := NewAWSCredentialsProvider(mockProvider, testProfile)

	// Test retrieving credentials
	creds, err := awsProvider.Retrieve(nil)
	if err != nil {
		t.Fatalf("Failed to retrieve AWS credentials: %v", err)
	}

	// Check retrieved credentials
	if creds.AccessKeyID != "test-access-key" {
		t.Errorf("Expected access key 'test-access-key', got '%s'", creds.AccessKeyID)
	}
	if creds.SecretAccessKey != "test-secret-key" {
		t.Errorf("Expected secret key 'test-secret-key', got '%s'", creds.SecretAccessKey)
	}
	if creds.SessionToken != "test-session-token" {
		t.Errorf("Expected session token 'test-session-token', got '%s'", creds.SessionToken)
	}
	if !creds.CanExpire {
		t.Errorf("Expected credentials to be marked as expirable")
	}
	if creds.Expires != expiry {
		t.Errorf("Expected expiry time %v, got %v", expiry, creds.Expires)
	}
	if creds.Source != "Prism" {
		t.Errorf("Expected source 'Prism', got '%s'", creds.Source)
	}

	// Test retrieving non-existent credentials
	nonExistentProvider := NewAWSCredentialsProvider(mockProvider, "non-existent")
	_, err = nonExistentProvider.Retrieve(nil)
	if err == nil {
		t.Errorf("Expected error retrieving non-existent credentials")
	}

	// Test credentials without expiration
	_ = mockProvider.StoreCredentials("no-expiry", &Credentials{
		AccessKeyID:     "test-access-key-2",
		SecretAccessKey: "test-secret-key-2",
		SessionToken:    "test-session-token-2",
		Expiration:      nil,
	})

	noExpiryProvider := NewAWSCredentialsProvider(mockProvider, "no-expiry")
	creds, err = noExpiryProvider.Retrieve(nil)
	if err != nil {
		t.Fatalf("Failed to retrieve AWS credentials: %v", err)
	}

	if creds.CanExpire {
		t.Errorf("Expected credentials to not be marked as expirable")
	}
}

func TestSecureCredentialProvider(t *testing.T) {
	// Create a new secure credential provider
	provider, err := NewCredentialProvider()
	if err != nil {
		t.Fatalf("Failed to create credential provider: %v", err)
	}

	// We don't test actual secure storage as it requires platform-specific setup
	// Instead, we just verify that the provider was created successfully
	if provider == nil {
		t.Fatalf("Expected non-nil credential provider")
	}

	// Test the fallback file implementation for credential removal
	// This is a smoke test that shouldn't fail but also won't do much
	err = provider.ClearCredentials("test-profile")
	if err != nil && !errors.Is(err, errors.New("unsupported platform")) {
		t.Errorf("Unexpected error clearing credentials: %v", err)
	}
}
