package pricing

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// InstitutionalPricingConfig represents pricing discounts provided by an institution
// This file can be distributed by institutions to their researchers for accurate cost estimation
type InstitutionalPricingConfig struct {
	// Metadata
	Institution string    `json:"institution"`
	CreatedAt   time.Time `json:"created_at"`
	ValidUntil  time.Time `json:"valid_until,omitempty"`
	Version     string    `json:"version"`
	Contact     string    `json:"contact,omitempty"` // Contact for pricing questions

	// Global Discounts (applied to all services)
	GlobalDiscounts struct {
		EC2Discount     float64 `json:"ec2_discount"`     // Percentage discount (0.0-1.0)
		EBSDiscount     float64 `json:"ebs_discount"`     // EBS storage discount
		EFSDiscount     float64 `json:"efs_discount"`     // EFS storage discount
		DataTransfer    float64 `json:"data_transfer"`    // Data transfer discount
		GeneralDiscount float64 `json:"general_discount"` // Catch-all discount
	} `json:"global_discounts"`

	// Instance Family Specific Discounts
	InstanceFamilyDiscounts map[string]float64 `json:"instance_family_discounts"` // family -> discount

	// Commitment-Based Discounts
	CommitmentPrograms struct {
		ReservedInstanceCoverage float64 `json:"reserved_instance_coverage"` // Percentage of usage covered
		SavingsPlanCoverage      float64 `json:"savings_plan_coverage"`      // Additional coverage
		SpotInstancePreference   float64 `json:"spot_instance_preference"`   // Preferred spot usage %
	} `json:"commitment_programs"`

	// Program-Specific Discounts
	Programs struct {
		EducationalDiscount float64 `json:"educational_discount"` // Academic institution discount
		StartupCredits      float64 `json:"startup_credits"`      // AWS Activate credits
		ResearchCredits     float64 `json:"research_credits"`     // AWS research credit program
		NonProfitDiscount   float64 `json:"nonprofit_discount"`   // Non-profit organization discount
	} `json:"programs"`

	// Enterprise Agreements
	Enterprise struct {
		EDPDiscount      float64 `json:"edp_discount"`      // Enterprise Discount Program
		VolumeDiscount   float64 `json:"volume_discount"`   // Volume-based discount tiers
		CommittedSpend   float64 `json:"committed_spend"`   // Annual committed spend
		CustomNegotiated float64 `json:"custom_negotiated"` // Custom negotiated rates
	} `json:"enterprise"`

	// Regional Variations
	RegionalDiscounts map[string]struct {
		AdditionalDiscount float64 `json:"additional_discount"` // Additional regional discount
		CreditMultiplier   float64 `json:"credit_multiplier"`   // Credit value multiplier
	} `json:"regional_discounts,omitempty"`

	// Budget and Cost Management
	CostManagement struct {
		BudgetAlerts     bool    `json:"budget_alerts"`     // Enable budget alerts
		CostOptimization bool    `json:"cost_optimization"` // Enable cost optimization suggestions
		SpendingLimit    float64 `json:"spending_limit"`    // Monthly spending limit
	} `json:"cost_management"`
}

// DefaultPricingConfig returns the default pricing configuration (no discounts)
func DefaultPricingConfig() *InstitutionalPricingConfig {
	return &InstitutionalPricingConfig{
		Institution: "Default",
		CreatedAt:   time.Now(),
		Version:     "1.0",
		// All discount fields default to 0.0 (no discount)
	}
}

// LoadInstitutionalPricing loads institutional pricing configuration
// Checks multiple locations in order of preference:
// 1. --pricing-config flag or PRICING_CONFIG env var
// 2. ~/.prism/institutional_pricing.json
// 3. ./institutional_pricing.json (current directory)
// 4. Returns default config (no discounts) if none found
func LoadInstitutionalPricing() (*InstitutionalPricingConfig, error) {
	// Check for explicit config path
	if configPath := os.Getenv("PRICING_CONFIG"); configPath != "" {
		return loadPricingConfigFromFile(configPath)
	}

	// Check standard locations
	configPaths := []string{
		getInstitutionalPricingPath(),
		"institutional_pricing.json",
	}

	for _, path := range configPaths {
		if _, err := os.Stat(path); err == nil {
			return loadPricingConfigFromFile(path)
		}
	}

	// Return default config (no discounts)
	return DefaultPricingConfig(), nil
}

// loadPricingConfigFromFile loads pricing config from a specific file
func loadPricingConfigFromFile(path string) (*InstitutionalPricingConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read pricing config file %s: %w", path, err)
	}

	config := &InstitutionalPricingConfig{}
	if err := json.Unmarshal(data, config); err != nil {
		return nil, fmt.Errorf("failed to parse pricing config file %s: %w", path, err)
	}

	// Validate configuration
	if err := validatePricingConfig(config); err != nil {
		return nil, fmt.Errorf("invalid pricing config in %s: %w", path, err)
	}

	return config, nil
}

// validatePricingConfig validates that discount values are reasonable
func validatePricingConfig(config *InstitutionalPricingConfig) error {
	// Validate discount ranges (0.0 to 1.0)
	discounts := []struct {
		name  string
		value float64
	}{
		{"ec2_discount", config.GlobalDiscounts.EC2Discount},
		{"ebs_discount", config.GlobalDiscounts.EBSDiscount},
		{"efs_discount", config.GlobalDiscounts.EFSDiscount},
		{"data_transfer", config.GlobalDiscounts.DataTransfer},
		{"general_discount", config.GlobalDiscounts.GeneralDiscount},
		{"educational_discount", config.Programs.EducationalDiscount},
		{"startup_credits", config.Programs.StartupCredits},
		{"research_credits", config.Programs.ResearchCredits},
		{"nonprofit_discount", config.Programs.NonProfitDiscount},
		{"edp_discount", config.Enterprise.EDPDiscount},
		{"volume_discount", config.Enterprise.VolumeDiscount},
		{"custom_negotiated", config.Enterprise.CustomNegotiated},
	}

	for _, discount := range discounts {
		if discount.value < 0.0 || discount.value > 1.0 {
			return fmt.Errorf("%s must be between 0.0 and 1.0, got %.3f", discount.name, discount.value)
		}
	}

	// Validate coverage percentages
	if config.CommitmentPrograms.ReservedInstanceCoverage < 0.0 || config.CommitmentPrograms.ReservedInstanceCoverage > 1.0 {
		return fmt.Errorf("reserved_instance_coverage must be between 0.0 and 1.0")
	}
	if config.CommitmentPrograms.SavingsPlanCoverage < 0.0 || config.CommitmentPrograms.SavingsPlanCoverage > 1.0 {
		return fmt.Errorf("savings_plan_coverage must be between 0.0 and 1.0")
	}

	// Check expiration date
	if !config.ValidUntil.IsZero() && time.Now().After(config.ValidUntil) {
		return fmt.Errorf("pricing configuration expired on %s", config.ValidUntil.Format("2006-01-02"))
	}

	return nil
}

// getInstitutionalPricingPath returns the standard path for institutional pricing config
func getInstitutionalPricingPath() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "institutional_pricing.json"
	}
	return filepath.Join(homeDir, ".prism", "institutional_pricing.json")
}

// SaveExampleConfig saves an example institutional pricing configuration
func SaveExampleConfig(path string) error {
	config := &InstitutionalPricingConfig{
		Institution: "Example University",
		CreatedAt:   time.Now(),
		ValidUntil:  time.Now().AddDate(1, 0, 0), // Valid for 1 year
		Version:     "1.0",
		Contact:     "cloudcomputing@example.university.edu",
	}

	// Example academic discounts
	config.GlobalDiscounts.EC2Discount = 0.30  // 30% discount on EC2
	config.GlobalDiscounts.EBSDiscount = 0.20  // 20% discount on EBS
	config.GlobalDiscounts.EFSDiscount = 0.15  // 15% discount on EFS
	config.GlobalDiscounts.DataTransfer = 0.25 // 25% discount on data transfer

	// Instance family discounts
	config.InstanceFamilyDiscounts = map[string]float64{
		"c5":   0.35, // 35% discount on c5 instances
		"m5":   0.30, // 30% discount on m5 instances
		"r5":   0.32, // 32% discount on r5 instances
		"p3":   0.40, // 40% discount on GPU instances
		"g4dn": 0.45, // 45% discount on gaming/ML instances
	}

	// Academic programs
	config.Programs.EducationalDiscount = 0.30
	config.Programs.ResearchCredits = 0.50 // 50% covered by research credits

	// Commitment preferences
	config.CommitmentPrograms.ReservedInstanceCoverage = 0.60 // 60% of usage via RIs
	config.CommitmentPrograms.SpotInstancePreference = 0.30   // Prefer spot for 30% of workloads

	// Cost management
	config.CostManagement.BudgetAlerts = true
	config.CostManagement.CostOptimization = true

	// Marshal to JSON
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal example config: %w", err)
	}

	// Write to file
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write example config to %s: %w", path, err)
	}

	return nil
}
