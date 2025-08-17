package pricing

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultPricingConfig(t *testing.T) {
	config := DefaultPricingConfig()

	assert.NotNil(t, config)
	assert.Equal(t, "Default", config.Institution)
	assert.Equal(t, "1.0", config.Version)
	assert.True(t, config.CreatedAt.After(time.Now().Add(-time.Minute)))

	// Verify all discounts are zero by default
	assert.Equal(t, 0.0, config.GlobalDiscounts.EC2Discount)
	assert.Equal(t, 0.0, config.GlobalDiscounts.EBSDiscount)
	assert.Equal(t, 0.0, config.GlobalDiscounts.EFSDiscount)
	assert.Equal(t, 0.0, config.Programs.EducationalDiscount)
	assert.Equal(t, 0.0, config.Enterprise.EDPDiscount)
	assert.Empty(t, config.InstanceFamilyDiscounts)
}

func TestLoadInstitutionalPricing_Default(t *testing.T) {
	// Ensure no env var is set and back up/remove any existing config
	oldEnv := os.Getenv("PRICING_CONFIG")
	_ = os.Unsetenv("PRICING_CONFIG")
	defer func() {
		if oldEnv != "" {
			_ = os.Setenv("PRICING_CONFIG", oldEnv)
		}
	}()

	// Back up any existing current directory config
	currentConfig := "institutional_pricing.json"
	var backupData []byte
	var hadExistingConfig bool
	if data, err := os.ReadFile(currentConfig); err == nil {
		backupData = data
		hadExistingConfig = true
		_ = os.Remove(currentConfig) // Temporarily remove
	}

	defer func() {
		if hadExistingConfig {
			_ = os.WriteFile(currentConfig, backupData, 0644) // Restore
		}
	}()

	// Also back up home directory config
	homeConfig := getInstitutionalPricingPath()
	var homeBackupData []byte
	var hadExistingHomeConfig bool
	if data, err := os.ReadFile(homeConfig); err == nil {
		homeBackupData = data
		hadExistingHomeConfig = true
		_ = os.Remove(homeConfig) // Temporarily remove
	}

	defer func() {
		if hadExistingHomeConfig {
			_ = os.WriteFile(homeConfig, homeBackupData, 0644) // Restore
		}
	}()

	// Test with no config files present
	config, err := LoadInstitutionalPricing()
	assert.NoError(t, err)
	assert.NotNil(t, config)
	assert.Equal(t, "Default", config.Institution)
	assert.Equal(t, 0.0, config.GlobalDiscounts.EC2Discount)
}

func TestLoadInstitutionalPricing_FromFile(t *testing.T) {
	// Create temporary config file
	tempDir, err := os.MkdirTemp("", "pricing-config-test-*")
	require.NoError(t, err)
	defer func() { _ = os.RemoveAll(tempDir) }()

	configFile := filepath.Join(tempDir, "test_pricing.json")

	testConfig := &InstitutionalPricingConfig{
		Institution: "Test University",
		CreatedAt:   time.Now(),
		ValidUntil:  time.Now().AddDate(1, 0, 0),
		Version:     "1.0",
		Contact:     "test@university.edu",
	}
	testConfig.GlobalDiscounts.EC2Discount = 0.25
	testConfig.Programs.EducationalDiscount = 0.30

	// Write test config
	data, err := json.MarshalIndent(testConfig, "", "  ")
	require.NoError(t, err)
	err = os.WriteFile(configFile, data, 0644)
	require.NoError(t, err)

	// Set environment variable to point to test config
	oldEnv := os.Getenv("PRICING_CONFIG")
	_ = os.Setenv("PRICING_CONFIG", configFile)
	defer func() { _ = os.Setenv("PRICING_CONFIG", oldEnv) }()

	// Load config
	config, err := LoadInstitutionalPricing()
	assert.NoError(t, err)
	assert.NotNil(t, config)
	assert.Equal(t, "Test University", config.Institution)
	assert.Equal(t, 0.25, config.GlobalDiscounts.EC2Discount)
	assert.Equal(t, 0.30, config.Programs.EducationalDiscount)
	assert.Equal(t, "test@university.edu", config.Contact)
}

func TestLoadInstitutionalPricing_CurrentDirectory(t *testing.T) {
	// Create config in current directory using a unique filename
	configFile := "test_institutional_pricing_current_dir.json"

	testConfig := &InstitutionalPricingConfig{
		Institution: "Local Test",
		CreatedAt:   time.Now(),
		Version:     "1.0",
	}
	testConfig.GlobalDiscounts.EBSDiscount = 0.15

	// Write test config
	data, err := json.MarshalIndent(testConfig, "", "  ")
	require.NoError(t, err)
	err = os.WriteFile(configFile, data, 0644)
	require.NoError(t, err)
	defer func() { _ = os.Remove(configFile) }() // Clean up

	// Load config directly from the file
	config, err := loadPricingConfigFromFile(configFile)
	assert.NoError(t, err)
	assert.Equal(t, "Local Test", config.Institution)
	assert.Equal(t, 0.15, config.GlobalDiscounts.EBSDiscount)
}

func TestLoadPricingConfigFromFile(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "pricing-test-*")
	require.NoError(t, err)
	defer func() { _ = os.RemoveAll(tempDir) }()

	tests := []struct {
		name         string
		configData   string
		expectError  bool
		expectedInst string
	}{
		{
			name: "valid config",
			configData: `{
				"institution": "Valid University",
				"created_at": "2024-01-15T10:00:00Z",
				"version": "1.0",
				"global_discounts": {
					"ec2_discount": 0.20
				}
			}`,
			expectError:  false,
			expectedInst: "Valid University",
		},
		{
			name: "invalid JSON",
			configData: `{
				"institution": "Invalid JSON"
				"missing_comma": true
			}`,
			expectError: true,
		},
		{
			name: "invalid discount values",
			configData: `{
				"institution": "Invalid Discount",
				"created_at": "2024-01-15T10:00:00Z",
				"version": "1.0",
				"global_discounts": {
					"ec2_discount": 1.5
				}
			}`,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			configFile := filepath.Join(tempDir, "test_"+tt.name+".json")
			err := os.WriteFile(configFile, []byte(tt.configData), 0644)
			require.NoError(t, err)

			config, err := loadPricingConfigFromFile(configFile)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, config)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, config)
				assert.Equal(t, tt.expectedInst, config.Institution)
			}
		})
	}
}

func TestValidatePricingConfig(t *testing.T) {
	tests := []struct {
		name          string
		modifyConfig  func(*InstitutionalPricingConfig)
		expectError   bool
		errorContains string
	}{
		{
			name: "valid config",
			modifyConfig: func(c *InstitutionalPricingConfig) {
				c.GlobalDiscounts.EC2Discount = 0.30
				c.Programs.EducationalDiscount = 0.25
			},
			expectError: false,
		},
		{
			name: "negative discount",
			modifyConfig: func(c *InstitutionalPricingConfig) {
				c.GlobalDiscounts.EC2Discount = -0.10
			},
			expectError:   true,
			errorContains: "ec2_discount must be between 0.0 and 1.0",
		},
		{
			name: "discount too high",
			modifyConfig: func(c *InstitutionalPricingConfig) {
				c.GlobalDiscounts.EBSDiscount = 1.5
			},
			expectError:   true,
			errorContains: "ebs_discount must be between 0.0 and 1.0",
		},
		{
			name: "invalid coverage percentage",
			modifyConfig: func(c *InstitutionalPricingConfig) {
				c.CommitmentPrograms.ReservedInstanceCoverage = 1.2
			},
			expectError:   true,
			errorContains: "reserved_instance_coverage must be between 0.0 and 1.0",
		},
		{
			name: "invalid savings plan coverage",
			modifyConfig: func(c *InstitutionalPricingConfig) {
				c.CommitmentPrograms.SavingsPlanCoverage = -0.1
			},
			expectError:   true,
			errorContains: "savings_plan_coverage must be between 0.0 and 1.0",
		},
		{
			name: "expired configuration",
			modifyConfig: func(c *InstitutionalPricingConfig) {
				c.ValidUntil = time.Now().AddDate(-1, 0, 0) // Expired 1 year ago
			},
			expectError:   true,
			errorContains: "pricing configuration expired",
		},
		{
			name: "educational discount validation",
			modifyConfig: func(c *InstitutionalPricingConfig) {
				c.Programs.EducationalDiscount = 2.0
			},
			expectError:   true,
			errorContains: "educational_discount must be between 0.0 and 1.0",
		},
		{
			name: "enterprise discount validation",
			modifyConfig: func(c *InstitutionalPricingConfig) {
				c.Enterprise.EDPDiscount = -0.5
			},
			expectError:   true,
			errorContains: "edp_discount must be between 0.0 and 1.0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &InstitutionalPricingConfig{
				Institution: "Test University",
				CreatedAt:   time.Now(),
				Version:     "1.0",
			}

			tt.modifyConfig(config)

			err := validatePricingConfig(config)

			if tt.expectError {
				assert.Error(t, err)
				if tt.errorContains != "" {
					assert.Contains(t, err.Error(), tt.errorContains)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestGetInstitutionalPricingPath(t *testing.T) {
	path := getInstitutionalPricingPath()
	assert.NotEmpty(t, path)

	// Should contain .cloudworkstation directory
	assert.Contains(t, path, ".cloudworkstation")
	assert.Contains(t, path, "institutional_pricing.json")
}

func TestSaveExampleConfig(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "example-config-test-*")
	require.NoError(t, err)
	defer func() { _ = os.RemoveAll(tempDir) }()

	configFile := filepath.Join(tempDir, "example.json")

	// Save example config
	err = SaveExampleConfig(configFile)
	assert.NoError(t, err)

	// Verify file was created
	_, err = os.Stat(configFile)
	assert.NoError(t, err)

	// Load and verify the config
	config, err := loadPricingConfigFromFile(configFile)
	assert.NoError(t, err)
	assert.NotNil(t, config)

	// Verify example values
	assert.Equal(t, "Example University", config.Institution)
	assert.Equal(t, "cloudcomputing@example.university.edu", config.Contact)
	assert.Equal(t, 0.30, config.GlobalDiscounts.EC2Discount)
	assert.Equal(t, 0.20, config.GlobalDiscounts.EBSDiscount)
	assert.Equal(t, 0.30, config.Programs.EducationalDiscount)

	// Verify instance family discounts
	assert.Contains(t, config.InstanceFamilyDiscounts, "c5")
	assert.Equal(t, 0.35, config.InstanceFamilyDiscounts["c5"])
	assert.Contains(t, config.InstanceFamilyDiscounts, "p3")
	assert.Equal(t, 0.40, config.InstanceFamilyDiscounts["p3"])

	// Verify commitment programs
	assert.Equal(t, 0.60, config.CommitmentPrograms.ReservedInstanceCoverage)
	assert.Equal(t, 0.30, config.CommitmentPrograms.SpotInstancePreference)

	// Verify cost management
	assert.True(t, config.CostManagement.BudgetAlerts)
	assert.True(t, config.CostManagement.CostOptimization)

	// Verify validity period
	assert.True(t, config.ValidUntil.After(time.Now()))
	assert.True(t, config.ValidUntil.Before(time.Now().AddDate(2, 0, 0))) // Within 2 years
}

func TestSaveExampleConfig_WriteError(t *testing.T) {
	// Try to write to invalid path
	err := SaveExampleConfig("/root/invalid/path/config.json")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to write example config")
}

func TestLoadInstitutionalPricing_FileReadError(t *testing.T) {
	// Set env var to non-existent file
	oldEnv := os.Getenv("PRICING_CONFIG")
	_ = os.Setenv("PRICING_CONFIG", "/nonexistent/path/config.json")
	defer func() {
		if oldEnv == "" {
			_ = os.Unsetenv("PRICING_CONFIG")
		} else {
			_ = os.Setenv("PRICING_CONFIG", oldEnv)
		}
	}()

	config, err := LoadInstitutionalPricing()

	// Should error when trying to read the specified file
	assert.Error(t, err)
	assert.Nil(t, config)
	assert.Contains(t, err.Error(), "failed to read pricing config file")
}

func TestValidatePricingConfig_AllDiscountTypes(t *testing.T) {
	config := &InstitutionalPricingConfig{
		Institution: "Complete Test",
		CreatedAt:   time.Now(),
		Version:     "1.0",
	}

	// Set all discount types to valid values
	config.GlobalDiscounts.EC2Discount = 0.25
	config.GlobalDiscounts.EBSDiscount = 0.20
	config.GlobalDiscounts.EFSDiscount = 0.15
	config.GlobalDiscounts.DataTransfer = 0.10
	config.GlobalDiscounts.GeneralDiscount = 0.05
	config.Programs.EducationalDiscount = 0.30
	config.Programs.StartupCredits = 0.50
	config.Programs.ResearchCredits = 0.40
	config.Programs.NonProfitDiscount = 0.20
	config.Enterprise.EDPDiscount = 0.35
	config.Enterprise.VolumeDiscount = 0.15
	config.Enterprise.CustomNegotiated = 0.25
	config.CommitmentPrograms.ReservedInstanceCoverage = 0.70
	config.CommitmentPrograms.SavingsPlanCoverage = 0.80

	err := validatePricingConfig(config)
	assert.NoError(t, err)
}

func TestConfigurationSerialization(t *testing.T) {
	originalConfig := &InstitutionalPricingConfig{
		Institution: "Serialization Test",
		CreatedAt:   time.Now().Truncate(time.Second), // Truncate for comparison
		ValidUntil:  time.Now().AddDate(1, 0, 0).Truncate(time.Second),
		Version:     "2.0",
		Contact:     "admin@test.edu",
	}

	// Set various discount values
	originalConfig.GlobalDiscounts.EC2Discount = 0.30
	originalConfig.InstanceFamilyDiscounts = map[string]float64{
		"m5": 0.25,
		"c5": 0.30,
	}
	originalConfig.Programs.EducationalDiscount = 0.35

	// Serialize to JSON
	data, err := json.MarshalIndent(originalConfig, "", "  ")
	require.NoError(t, err)

	// Deserialize back
	deserializedConfig := &InstitutionalPricingConfig{}
	err = json.Unmarshal(data, deserializedConfig)
	require.NoError(t, err)

	// Compare key fields
	assert.Equal(t, originalConfig.Institution, deserializedConfig.Institution)
	assert.Equal(t, originalConfig.Version, deserializedConfig.Version)
	assert.Equal(t, originalConfig.Contact, deserializedConfig.Contact)
	assert.Equal(t, originalConfig.GlobalDiscounts.EC2Discount, deserializedConfig.GlobalDiscounts.EC2Discount)
	assert.Equal(t, originalConfig.Programs.EducationalDiscount, deserializedConfig.Programs.EducationalDiscount)
	assert.Equal(t, originalConfig.InstanceFamilyDiscounts["m5"], deserializedConfig.InstanceFamilyDiscounts["m5"])
	assert.Equal(t, originalConfig.InstanceFamilyDiscounts["c5"], deserializedConfig.InstanceFamilyDiscounts["c5"])
}

func TestConfigurationWithRegionalDiscounts(t *testing.T) {
	config := &InstitutionalPricingConfig{
		Institution: "Regional Test",
		CreatedAt:   time.Now(),
		Version:     "1.0",
	}

	// Add regional discounts
	config.RegionalDiscounts = map[string]struct {
		AdditionalDiscount float64 `json:"additional_discount"`
		CreditMultiplier   float64 `json:"credit_multiplier"`
	}{
		"us-east-1": {
			AdditionalDiscount: 0.05,
			CreditMultiplier:   1.2,
		},
		"eu-west-1": {
			AdditionalDiscount: 0.10,
			CreditMultiplier:   1.1,
		},
	}

	// Should validate without errors
	err := validatePricingConfig(config)
	assert.NoError(t, err)

	// Test serialization
	data, err := json.MarshalIndent(config, "", "  ")
	require.NoError(t, err)

	// Should contain regional discount data
	assert.Contains(t, string(data), "us-east-1")
	assert.Contains(t, string(data), "additional_discount")
	assert.Contains(t, string(data), "credit_multiplier")
}
