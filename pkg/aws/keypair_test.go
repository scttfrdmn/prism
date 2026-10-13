package aws

import (
	"context"
	"errors"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	ec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestEnsureKeyPairExists tests the SSH key pair management with force update logic
func TestEnsureKeyPairExists(t *testing.T) {
	t.Run("key_does_not_exist_imports_new_key", func(t *testing.T) {
		// User scenario: Fresh launch with no existing AWS key pair
		mockEC2 := &MockEC2Client{}

		manager := &Manager{
			ec2:    mockEC2,
			region: "us-west-2",
		}

		// Mock DescribeKeyPairs returning "not found" error
		mockEC2.DescribeKeyPairsFunc = func(ctx context.Context, params *ec2.DescribeKeyPairsInput) (*ec2.DescribeKeyPairsOutput, error) {
			return nil, errors.New("InvalidKeyPair.NotFound: The key pair 'test-key' does not exist")
		}

		// Track that ImportKeyPair was called
		importCalled := false
		mockEC2.ImportKeyPairFunc = func(ctx context.Context, params *ec2.ImportKeyPairInput) (*ec2.ImportKeyPairOutput, error) {
			importCalled = true
			assert.Equal(t, "test-key", *params.KeyName, "Should import with correct key name")
			assert.Equal(t, []byte("ssh-rsa AAAAB3... test-key"), params.PublicKeyMaterial, "Should import correct public key")

			// Verify tags are set
			require.Len(t, params.TagSpecifications, 1)
			assert.Equal(t, ec2types.ResourceTypeKeyPair, params.TagSpecifications[0].ResourceType)
			assert.Contains(t, params.TagSpecifications[0].Tags, ec2types.Tag{
				Key:   aws.String("Prism"),
				Value: aws.String("true"),
			})

			return &ec2.ImportKeyPairOutput{}, nil
		}

		// Test key pair creation
		err := manager.EnsureKeyPairExists("test-key", "ssh-rsa AAAAB3... test-key")

		assert.NoError(t, err, "Should successfully import new key pair")
		assert.True(t, importCalled, "ImportKeyPair should be called")
		t.Logf("✅ New key pair imported successfully")
	})

	t.Run("key_exists_deletes_and_reimports", func(t *testing.T) {
		// User scenario: Key name exists in AWS but content doesn't match local key
		// This is the bug we're fixing - should delete and reimport
		mockEC2 := &MockEC2Client{}

		manager := &Manager{
			ec2:    mockEC2,
			region: "us-west-2",
		}

		// Mock DescribeKeyPairs returning existing key
		mockEC2.DescribeKeyPairsFunc = func(ctx context.Context, params *ec2.DescribeKeyPairsInput) (*ec2.DescribeKeyPairsOutput, error) {
			return &ec2.DescribeKeyPairsOutput{
				KeyPairs: []ec2types.KeyPairInfo{
					{
						KeyName:        aws.String("test-key"),
						KeyFingerprint: aws.String("dc:4d:f4:01:50:0a:e0:33:b5:ba:b2:bc:6e:72:1a:8a"),
					},
				},
			}, nil
		}

		// Track that DeleteKeyPair was called
		deleteCalled := false
		mockEC2.DeleteKeyPairFunc = func(ctx context.Context, params *ec2.DeleteKeyPairInput) (*ec2.DeleteKeyPairOutput, error) {
			deleteCalled = true
			assert.Equal(t, "test-key", *params.KeyName, "Should delete the existing key")
			return &ec2.DeleteKeyPairOutput{}, nil
		}

		// Track that ImportKeyPair was called after delete
		importCalled := false
		mockEC2.ImportKeyPairFunc = func(ctx context.Context, params *ec2.ImportKeyPairInput) (*ec2.ImportKeyPairOutput, error) {
			importCalled = true
			assert.Equal(t, "test-key", *params.KeyName, "Should import with correct key name")
			assert.Equal(t, []byte("ssh-rsa AAAAB3... new-content"), params.PublicKeyMaterial, "Should import new key content")
			return &ec2.ImportKeyPairOutput{}, nil
		}

		// Test key pair replacement
		err := manager.EnsureKeyPairExists("test-key", "ssh-rsa AAAAB3... new-content")

		assert.NoError(t, err, "Should successfully replace existing key pair")
		assert.True(t, deleteCalled, "DeleteKeyPair should be called to remove old key")
		assert.True(t, importCalled, "ImportKeyPair should be called to add new key")
		t.Logf("✅ Existing key pair deleted and reimported with new content")
	})

	t.Run("describe_error_not_not_found_returns_error", func(t *testing.T) {
		// User scenario: AWS API error (permissions, network, etc.) - not "not found"
		mockEC2 := &MockEC2Client{}

		manager := &Manager{
			ec2:    mockEC2,
			region: "us-west-2",
		}

		// Mock DescribeKeyPairs returning permission error
		mockEC2.DescribeKeyPairsFunc = func(ctx context.Context, params *ec2.DescribeKeyPairsInput) (*ec2.DescribeKeyPairsOutput, error) {
			return nil, errors.New("UnauthorizedOperation: You are not authorized to perform this operation")
		}

		// Test error handling
		err := manager.EnsureKeyPairExists("test-key", "ssh-rsa AAAAB3... test-key")

		assert.Error(t, err, "Should return error for AWS API failures")
		assert.Contains(t, err.Error(), "failed to check if key pair exists", "Should wrap error with context")
		assert.Contains(t, err.Error(), "UnauthorizedOperation", "Should include original error")
		t.Logf("✅ AWS API errors handled correctly: %v", err)
	})

	t.Run("delete_error_returns_error", func(t *testing.T) {
		// User scenario: Key exists but deletion fails (permissions, etc.)
		mockEC2 := &MockEC2Client{}

		manager := &Manager{
			ec2:    mockEC2,
			region: "us-west-2",
		}

		// Mock DescribeKeyPairs returning existing key
		mockEC2.DescribeKeyPairsFunc = func(ctx context.Context, params *ec2.DescribeKeyPairsInput) (*ec2.DescribeKeyPairsOutput, error) {
			return &ec2.DescribeKeyPairsOutput{
				KeyPairs: []ec2types.KeyPairInfo{
					{
						KeyName: aws.String("test-key"),
					},
				},
			}, nil
		}

		// Mock DeleteKeyPair returning permission error
		mockEC2.DeleteKeyPairFunc = func(ctx context.Context, params *ec2.DeleteKeyPairInput) (*ec2.DeleteKeyPairOutput, error) {
			return nil, errors.New("UnauthorizedOperation: You are not authorized to delete this key pair")
		}

		// Test error handling
		err := manager.EnsureKeyPairExists("test-key", "ssh-rsa AAAAB3... test-key")

		assert.Error(t, err, "Should return error when deletion fails")
		assert.Contains(t, err.Error(), "failed to delete existing key pair", "Should wrap error with context")
		assert.Contains(t, err.Error(), "UnauthorizedOperation", "Should include original error")
		t.Logf("✅ Key deletion errors handled correctly: %v", err)
	})

	t.Run("import_error_returns_error", func(t *testing.T) {
		// User scenario: Import fails (invalid key format, quota exceeded, etc.)
		mockEC2 := &MockEC2Client{}

		manager := &Manager{
			ec2:    mockEC2,
			region: "us-west-2",
		}

		// Mock DescribeKeyPairs returning "not found"
		mockEC2.DescribeKeyPairsFunc = func(ctx context.Context, params *ec2.DescribeKeyPairsInput) (*ec2.DescribeKeyPairsOutput, error) {
			return nil, errors.New("InvalidKeyPair.NotFound: The key pair does not exist")
		}

		// Mock ImportKeyPair returning error
		mockEC2.ImportKeyPairFunc = func(ctx context.Context, params *ec2.ImportKeyPairInput) (*ec2.ImportKeyPairOutput, error) {
			return nil, errors.New("InvalidKeyPair.Format: Invalid public key format")
		}

		// Test error handling
		err := manager.EnsureKeyPairExists("test-key", "invalid-key-content")

		assert.Error(t, err, "Should return error when import fails")
		assert.Contains(t, err.Error(), "failed to import key pair", "Should wrap error with context")
		assert.Contains(t, err.Error(), "InvalidKeyPair.Format", "Should include original error")
		t.Logf("✅ Key import errors handled correctly: %v", err)
	})

	t.Run("idempotent_multiple_calls_with_same_content", func(t *testing.T) {
		// User scenario: Calling EnsureKeyPairExists multiple times should work
		mockEC2 := &MockEC2Client{}

		manager := &Manager{
			ec2:    mockEC2,
			region: "us-west-2",
		}

		callCount := 0
		publicKeyContent := "ssh-rsa AAAAB3... test-key"

		// Mock DescribeKeyPairs - first call returns not found, subsequent calls return key exists
		mockEC2.DescribeKeyPairsFunc = func(ctx context.Context, params *ec2.DescribeKeyPairsInput) (*ec2.DescribeKeyPairsOutput, error) {
			if callCount == 0 {
				callCount++
				return nil, errors.New("InvalidKeyPair.NotFound: The key pair does not exist")
			}
			return &ec2.DescribeKeyPairsOutput{
				KeyPairs: []ec2types.KeyPairInfo{
					{
						KeyName: aws.String("test-key"),
					},
				},
			}, nil
		}

		deleteCount := 0
		mockEC2.DeleteKeyPairFunc = func(ctx context.Context, params *ec2.DeleteKeyPairInput) (*ec2.DeleteKeyPairOutput, error) {
			deleteCount++
			return &ec2.DeleteKeyPairOutput{}, nil
		}

		importCount := 0
		mockEC2.ImportKeyPairFunc = func(ctx context.Context, params *ec2.ImportKeyPairInput) (*ec2.ImportKeyPairOutput, error) {
			importCount++
			return &ec2.ImportKeyPairOutput{}, nil
		}

		// First call - should import
		err := manager.EnsureKeyPairExists("test-key", publicKeyContent)
		assert.NoError(t, err)
		assert.Equal(t, 0, deleteCount, "Should not delete on first call")
		assert.Equal(t, 1, importCount, "Should import on first call")

		// Second call - should delete and reimport
		err = manager.EnsureKeyPairExists("test-key", publicKeyContent)
		assert.NoError(t, err)
		assert.Equal(t, 1, deleteCount, "Should delete on second call")
		assert.Equal(t, 2, importCount, "Should import again on second call")

		t.Logf("✅ Idempotent behavior verified: import=%d, delete=%d", importCount, deleteCount)
	})
}
