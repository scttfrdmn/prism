package pricing

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewCalculator(t *testing.T) {
	config := DefaultPricingConfig()
	calculator := NewCalculator(config)

	assert.NotNil(t, calculator)
	assert.Equal(t, config, calculator.config)
}

func TestNewCalculator_NilConfig(t *testing.T) {
	calculator := NewCalculator(nil)

	assert.NotNil(t, calculator)
	assert.Nil(t, calculator.config)
}

func TestCalculateInstanceCost_NoDiscounts(t *testing.T) {
	calculator := NewCalculator(nil)

	result := calculator.CalculateInstanceCost("m5.large", 0.096, "us-east-1")

	assert.NotNil(t, result)
	assert.Equal(t, 0.096, result.ListPrice)
	assert.Equal(t, 0.096, result.DiscountedPrice)
	assert.Equal(t, 0.0, result.TotalDiscount)
	assert.InDelta(t, 0.096*24, result.DailyEstimate, 0.001)
	assert.InDelta(t, 0.096*24*30, result.MonthlyEstimate, 0.001)
	assert.Empty(t, result.AppliedDiscounts)
}

func TestCalculateInstanceCost_WithGlobalEC2Discount(t *testing.T) {
	config := DefaultPricingConfig()
	config.GlobalDiscounts.EC2Discount = 0.30 // 30% discount
	calculator := NewCalculator(config)

	result := calculator.CalculateInstanceCost("m5.large", 0.100, "us-east-1")

	assert.Equal(t, 0.100, result.ListPrice)
	assert.InDelta(t, 0.070, result.DiscountedPrice, 0.001) // 100 - 30% = 70
	assert.InDelta(t, 0.30, result.TotalDiscount, 0.001)
	assert.InDelta(t, 0.070*24, result.DailyEstimate, 0.001)
	assert.InDelta(t, 0.070*24*30, result.MonthlyEstimate, 0.001)

	// Verify applied discount
	require.Len(t, result.AppliedDiscounts, 1)
	discount := result.AppliedDiscounts[0]
	assert.Equal(t, "global_ec2", discount.Type)
	assert.Equal(t, 0.30, discount.Percentage)
	assert.Equal(t, 0.030, discount.Savings) // 0.100 * 0.30
	assert.Contains(t, discount.Description, "Global EC2 discount")
}

func TestCalculateInstanceCost_WithInstanceFamilyDiscount(t *testing.T) {
	config := DefaultPricingConfig()
	config.InstanceFamilyDiscounts = map[string]float64{
		"c5": 0.25, // 25% discount on c5 instances
		"m5": 0.20, // 20% discount on m5 instances
	}
	calculator := NewCalculator(config)

	result := calculator.CalculateInstanceCost("c5.xlarge", 0.200, "us-east-1")

	assert.Equal(t, 0.200, result.ListPrice)
	assert.InDelta(t, 0.150, result.DiscountedPrice, 0.001) // 200 - 25% = 150
	assert.InDelta(t, 0.25, result.TotalDiscount, 0.001)

	// Verify applied discount
	require.Len(t, result.AppliedDiscounts, 1)
	discount := result.AppliedDiscounts[0]
	assert.Equal(t, "instance_family", discount.Type)
	assert.Equal(t, 0.25, discount.Percentage)
	assert.Equal(t, 0.050, discount.Savings) // 0.200 * 0.25
	assert.Contains(t, discount.Description, "c5 instance family")
}

func TestCalculateInstanceCost_WithEducationalDiscount(t *testing.T) {
	config := DefaultPricingConfig()
	config.Programs.EducationalDiscount = 0.35 // 35% educational discount
	calculator := NewCalculator(config)

	result := calculator.CalculateInstanceCost("m5.large", 0.100, "us-east-1")

	assert.Equal(t, 0.100, result.ListPrice)
	assert.InDelta(t, 0.065, result.DiscountedPrice, 0.001) // 100 - 35% = 65
	assert.InDelta(t, 0.35, result.TotalDiscount, 0.001)

	// Verify applied discount
	require.Len(t, result.AppliedDiscounts, 1)
	discount := result.AppliedDiscounts[0]
	assert.Equal(t, "educational", discount.Type)
	assert.Equal(t, 0.35, discount.Percentage)
	assert.Contains(t, discount.Description, "Educational institution")
}

func TestCalculateInstanceCost_WithEDPDiscount(t *testing.T) {
	config := DefaultPricingConfig()
	config.Enterprise.EDPDiscount = 0.40 // 40% enterprise discount
	calculator := NewCalculator(config)

	result := calculator.CalculateInstanceCost("m5.large", 0.100, "us-east-1")

	assert.Equal(t, 0.100, result.ListPrice)
	assert.InDelta(t, 0.060, result.DiscountedPrice, 0.001) // 100 - 40% = 60
	assert.InDelta(t, 0.40, result.TotalDiscount, 0.001)

	// Verify applied discount
	require.Len(t, result.AppliedDiscounts, 1)
	discount := result.AppliedDiscounts[0]
	assert.Equal(t, "enterprise_edp", discount.Type)
	assert.Equal(t, 0.40, discount.Percentage)
	assert.Contains(t, discount.Description, "Enterprise Discount Program")
}

func TestCalculateInstanceCost_WithRegionalDiscount(t *testing.T) {
	config := DefaultPricingConfig()
	config.RegionalDiscounts = map[string]struct {
		AdditionalDiscount float64 `json:"additional_discount"`
		CreditMultiplier   float64 `json:"credit_multiplier"`
	}{
		"us-west-2": {
			AdditionalDiscount: 0.15, // 15% regional discount
			CreditMultiplier:   1.1,
		},
	}
	calculator := NewCalculator(config)

	result := calculator.CalculateInstanceCost("m5.large", 0.100, "us-west-2")

	assert.Equal(t, 0.100, result.ListPrice)
	assert.InDelta(t, 0.085, result.DiscountedPrice, 0.001) // 100 - 15% = 85
	assert.InDelta(t, 0.15, result.TotalDiscount, 0.001)

	// Verify applied discount
	require.Len(t, result.AppliedDiscounts, 1)
	discount := result.AppliedDiscounts[0]
	assert.Equal(t, "regional", discount.Type)
	assert.Equal(t, 0.15, discount.Percentage)
	assert.Contains(t, discount.Description, "us-west-2")
}

func TestCalculateInstanceCost_WithReservedInstanceDiscount(t *testing.T) {
	config := DefaultPricingConfig()
	config.CommitmentPrograms.ReservedInstanceCoverage = 0.75 // 75% RI coverage
	calculator := NewCalculator(config)

	result := calculator.CalculateInstanceCost("m5.large", 0.100, "us-east-1")

	// RI discount: 40% * 75% coverage = 30% effective discount
	expectedDiscount := 0.40 * 0.75
	expectedPrice := 0.100 * (1 - expectedDiscount)

	assert.Equal(t, 0.100, result.ListPrice)
	assert.InDelta(t, expectedPrice, result.DiscountedPrice, 0.001)
	assert.InDelta(t, expectedDiscount, result.TotalDiscount, 0.001)

	// Verify applied discount
	require.Len(t, result.AppliedDiscounts, 1)
	discount := result.AppliedDiscounts[0]
	assert.Equal(t, "reserved_instance", discount.Type)
	assert.Contains(t, discount.Description, "75% coverage")
}

func TestCalculateInstanceCost_WithSavingsPlanDiscount(t *testing.T) {
	config := DefaultPricingConfig()
	config.CommitmentPrograms.SavingsPlanCoverage = 0.80 // 80% SP coverage
	calculator := NewCalculator(config)

	result := calculator.CalculateInstanceCost("m5.large", 0.100, "us-east-1")

	// SP discount: 15% * 80% coverage = 12% effective discount
	expectedDiscount := 0.15 * 0.80
	expectedPrice := 0.100 * (1 - expectedDiscount)

	assert.Equal(t, 0.100, result.ListPrice)
	assert.InDelta(t, expectedPrice, result.DiscountedPrice, 0.001)
	assert.InDelta(t, expectedDiscount, result.TotalDiscount, 0.001)

	// Verify applied discount
	require.Len(t, result.AppliedDiscounts, 1)
	discount := result.AppliedDiscounts[0]
	assert.Equal(t, "savings_plan", discount.Type)
	assert.Contains(t, discount.Description, "80% coverage")
}

func TestCalculateInstanceCost_MultipleDiscounts(t *testing.T) {
	config := DefaultPricingConfig()
	config.Institution = "Test University"
	config.GlobalDiscounts.EC2Discount = 0.20  // 20% global discount
	config.Programs.EducationalDiscount = 0.25 // 25% educational discount
	config.Enterprise.EDPDiscount = 0.15       // 15% enterprise discount
	config.InstanceFamilyDiscounts = map[string]float64{
		"c5": 0.10, // 10% instance family discount
	}

	calculator := NewCalculator(config)

	result := calculator.CalculateInstanceCost("c5.large", 1.000, "us-east-1")

	// Discounts are applied sequentially:
	// Start: $1.000
	// Global EC2 (20%): $1.000 * 0.8 = $0.800
	// Instance family (10%): $0.800 * 0.9 = $0.720
	// Educational (25%): $0.720 * 0.75 = $0.540
	// Enterprise (15%): $0.540 * 0.85 = $0.459

	assert.Equal(t, 1.000, result.ListPrice)
	assert.InDelta(t, 0.459, result.DiscountedPrice, 0.001)

	// Total discount should be (1.000 - 0.459) / 1.000 = 0.541
	assert.InDelta(t, 0.541, result.TotalDiscount, 0.001)

	// Should have 4 applied discounts
	assert.Len(t, result.AppliedDiscounts, 4)

	// Verify discount types
	discountTypes := make(map[string]bool)
	for _, discount := range result.AppliedDiscounts {
		discountTypes[discount.Type] = true
	}
	assert.True(t, discountTypes["global_ec2"])
	assert.True(t, discountTypes["instance_family"])
	assert.True(t, discountTypes["educational"])
	assert.True(t, discountTypes["enterprise_edp"])
}

func TestCalculateStorageCost_NoDiscounts(t *testing.T) {
	calculator := NewCalculator(nil)

	// 100GB EBS at $0.10 per GB/month
	result := calculator.CalculateStorageCost("ebs", 100, 0.10, "us-east-1")

	assert.Equal(t, 10.0, result.ListPrice) // 100 * 0.10
	assert.Equal(t, 10.0, result.DiscountedPrice)
	assert.Equal(t, 0.0, result.TotalDiscount)
	assert.Equal(t, 10.0, result.MonthlyEstimate)
	assert.Empty(t, result.AppliedDiscounts)
}

func TestCalculateStorageCost_WithEBSDiscount(t *testing.T) {
	config := DefaultPricingConfig()
	config.GlobalDiscounts.EBSDiscount = 0.30 // 30% EBS discount
	calculator := NewCalculator(config)

	// 100GB EBS at $0.10 per GB/month
	result := calculator.CalculateStorageCost("ebs", 100, 0.10, "us-east-1")

	assert.Equal(t, 10.0, result.ListPrice)
	assert.InDelta(t, 7.0, result.DiscountedPrice, 0.001) // 10.0 - 30% = 7.0
	assert.InDelta(t, 0.30, result.TotalDiscount, 0.001)
	assert.InDelta(t, 7.0, result.MonthlyEstimate, 0.001)

	// Verify applied discount
	require.Len(t, result.AppliedDiscounts, 1)
	discount := result.AppliedDiscounts[0]
	assert.Equal(t, "global_ebs", discount.Type)
	assert.Equal(t, 0.30, discount.Percentage)
	assert.Contains(t, discount.Description, "Global EBS storage")
}

func TestCalculateStorageCost_WithEFSDiscount(t *testing.T) {
	config := DefaultPricingConfig()
	config.GlobalDiscounts.EFSDiscount = 0.25 // 25% EFS discount
	calculator := NewCalculator(config)

	// 200GB EFS at $0.30 per GB/month
	result := calculator.CalculateStorageCost("efs", 200, 0.30, "us-east-1")

	assert.Equal(t, 60.0, result.ListPrice)                // 200 * 0.30
	assert.InDelta(t, 45.0, result.DiscountedPrice, 0.001) // 60.0 - 25% = 45.0
	assert.InDelta(t, 0.25, result.TotalDiscount, 0.001)
	assert.InDelta(t, 45.0, result.MonthlyEstimate, 0.001)

	// Verify applied discount
	require.Len(t, result.AppliedDiscounts, 1)
	discount := result.AppliedDiscounts[0]
	assert.Equal(t, "global_efs", discount.Type)
	assert.Contains(t, discount.Description, "Global EFS storage")
}

func TestCalculateStorageCost_WithGeneralDiscount(t *testing.T) {
	config := DefaultPricingConfig()
	config.GlobalDiscounts.GeneralDiscount = 0.15 // 15% general discount
	calculator := NewCalculator(config)

	// Unknown storage type should use general discount
	result := calculator.CalculateStorageCost("s3", 500, 0.023, "us-east-1")

	assert.Equal(t, 11.5, result.ListPrice)                 // 500 * 0.023
	assert.InDelta(t, 9.775, result.DiscountedPrice, 0.001) // 11.5 - 15% = 9.775
	assert.InDelta(t, 0.15, result.TotalDiscount, 0.001)

	// Verify applied discount
	require.Len(t, result.AppliedDiscounts, 1)
	discount := result.AppliedDiscounts[0]
	assert.Equal(t, "general", discount.Type)
	assert.Contains(t, discount.Description, "General storage")
}

func TestCalculateStorageCost_WithVolumeDiscount(t *testing.T) {
	config := DefaultPricingConfig()
	config.GlobalDiscounts.EBSDiscount = 0.20 // 20% EBS discount
	config.Enterprise.VolumeDiscount = 0.10   // 10% volume discount
	calculator := NewCalculator(config)

	// 2000GB (2TB) storage should trigger volume discount
	result := calculator.CalculateStorageCost("ebs", 2000, 0.10, "us-east-1")

	// Start: 2000 * 0.10 = $200
	// EBS discount (20%): $200 * 0.8 = $160
	// Volume discount (10%): $160 * 0.9 = $144

	assert.Equal(t, 200.0, result.ListPrice)
	assert.InDelta(t, 144.0, result.DiscountedPrice, 0.001)
	assert.InDelta(t, 0.28, result.TotalDiscount, 0.001) // (200-144)/200 = 0.28

	// Should have 2 applied discounts
	require.Len(t, result.AppliedDiscounts, 2)

	discountTypes := make(map[string]bool)
	for _, discount := range result.AppliedDiscounts {
		discountTypes[discount.Type] = true
	}
	assert.True(t, discountTypes["global_ebs"])
	assert.True(t, discountTypes["volume"])
}

func TestCalculateStorageCost_SmallStorageNoVolumeDiscount(t *testing.T) {
	config := DefaultPricingConfig()
	config.Enterprise.VolumeDiscount = 0.10 // 10% volume discount
	calculator := NewCalculator(config)

	// 500GB storage should NOT trigger volume discount (< 1000GB)
	result := calculator.CalculateStorageCost("ebs", 500, 0.10, "us-east-1")

	assert.Equal(t, 50.0, result.ListPrice)
	assert.Equal(t, 50.0, result.DiscountedPrice) // No discounts applied
	assert.InDelta(t, 0.0, result.TotalDiscount, 0.001)
	assert.Empty(t, result.AppliedDiscounts)
}

func TestGetSpotInstanceDiscount_NoConfig(t *testing.T) {
	calculator := NewCalculator(nil)

	discount := calculator.GetSpotInstanceDiscount()

	assert.Equal(t, 0.70, discount) // Default 70% spot discount
}

func TestGetSpotInstanceDiscount_WithPreference(t *testing.T) {
	config := DefaultPricingConfig()
	config.CommitmentPrograms.SpotInstancePreference = 0.50 // 50% preference
	calculator := NewCalculator(config)

	discount := calculator.GetSpotInstanceDiscount()

	// 70% * 50% preference = 35% effective discount
	assert.Equal(t, 0.35, discount)
}

func TestGetSpotInstanceDiscount_ZeroPreference(t *testing.T) {
	config := DefaultPricingConfig()
	config.CommitmentPrograms.SpotInstancePreference = 0.0 // No preference
	calculator := NewCalculator(config)

	discount := calculator.GetSpotInstanceDiscount()

	assert.Equal(t, 0.70, discount) // Falls back to default
}

func TestExtractInstanceFamily(t *testing.T) {
	tests := []struct {
		instanceType   string
		expectedFamily string
	}{
		{"c5.large", "c5"},
		{"m5a.xlarge", "m5a"},
		{"r5.2xlarge", "r5"},
		{"p3.8xlarge", "p3"},
		{"g4dn.medium", "g4dn"},
		{"t3.micro", "t3"},
		{"invalid-type", "invalid-type"}, // No dot, return as-is
		{"", ""},                         // Empty string
		{"x1e.xlarge", "x1e"},
	}

	for _, tt := range tests {
		t.Run(tt.instanceType, func(t *testing.T) {
			family := extractInstanceFamily(tt.instanceType)
			assert.Equal(t, tt.expectedFamily, family)
		})
	}
}

func TestGetPricingInfo_NoConfig(t *testing.T) {
	calculator := NewCalculator(nil)

	info := calculator.GetPricingInfo()

	assert.Equal(t, "None", info["institution"])
	assert.Equal(t, false, info["discounts_available"])
	assert.Equal(t, "list_price", info["pricing_model"])
}

func TestGetPricingInfo_WithConfig(t *testing.T) {
	config := &InstitutionalPricingConfig{
		Institution: "Test University",
		Version:     "2.0",
		Contact:     "pricing@test.edu",
		ValidUntil:  time.Now().AddDate(1, 0, 0),
	}
	config.GlobalDiscounts.EC2Discount = 0.25
	config.Programs.EducationalDiscount = 0.30
	config.Enterprise.EDPDiscount = 0.35

	calculator := NewCalculator(config)

	info := calculator.GetPricingInfo()

	assert.Equal(t, "Test University", info["institution"])
	assert.Equal(t, true, info["discounts_available"])
	assert.Equal(t, "2.0", info["version"])
	assert.Equal(t, "pricing@test.edu", info["contact"])
	assert.Equal(t, "25.0%", info["ec2_discount"])
	assert.Equal(t, "30.0%", info["educational_discount"])
	assert.Equal(t, "35.0%", info["enterprise_discount"])
}

func TestCalculateInstanceCost_ZeroPrice(t *testing.T) {
	config := DefaultPricingConfig()
	config.GlobalDiscounts.EC2Discount = 0.30
	calculator := NewCalculator(config)

	result := calculator.CalculateInstanceCost("m5.large", 0.0, "us-east-1")

	assert.Equal(t, 0.0, result.ListPrice)
	assert.Equal(t, 0.0, result.DiscountedPrice)
	assert.Equal(t, 0.0, result.TotalDiscount) // No total discount calculation for zero price
	assert.Equal(t, 0.0, result.DailyEstimate)
	assert.Equal(t, 0.0, result.MonthlyEstimate)
}

func TestCalculateStorageCost_ZeroSize(t *testing.T) {
	config := DefaultPricingConfig()
	config.GlobalDiscounts.EBSDiscount = 0.25
	calculator := NewCalculator(config)

	result := calculator.CalculateStorageCost("ebs", 0, 0.10, "us-east-1")

	assert.Equal(t, 0.0, result.ListPrice)
	assert.Equal(t, 0.0, result.DiscountedPrice)
	assert.Equal(t, 0.0, result.TotalDiscount)
	assert.Equal(t, 0.0, result.MonthlyEstimate)
}

func TestCalculateInstanceCost_EdgeCaseDiscounts(t *testing.T) {
	config := DefaultPricingConfig()

	// Test with very small discount
	config.GlobalDiscounts.EC2Discount = 0.001 // 0.1% discount
	calculator := NewCalculator(config)

	result := calculator.CalculateInstanceCost("m5.large", 1.000, "us-east-1")

	assert.Equal(t, 1.000, result.ListPrice)
	assert.InDelta(t, 0.999, result.DiscountedPrice, 0.001)
	assert.InDelta(t, 0.001, result.TotalDiscount, 0.001)

	// Test with maximum valid discount
	config.GlobalDiscounts.EC2Discount = 1.0 // 100% discount (free)
	result2 := calculator.CalculateInstanceCost("m5.large", 1.000, "us-east-1")

	assert.Equal(t, 1.000, result2.ListPrice)
	assert.Equal(t, 0.000, result2.DiscountedPrice)
	assert.Equal(t, 1.0, result2.TotalDiscount)
}

func TestCalculateInstanceCost_ComplexScenario(t *testing.T) {
	// Test a realistic academic pricing scenario
	config := &InstitutionalPricingConfig{
		Institution: "Academic Research University",
		CreatedAt:   time.Now(),
		Version:     "2.0",
		Contact:     "cloud-pricing@university.edu",
	}

	// Set up comprehensive academic discounts
	config.GlobalDiscounts.EC2Discount = 0.35  // 35% university EC2 discount
	config.Programs.EducationalDiscount = 0.20 // 20% additional educational discount
	config.Programs.ResearchCredits = 0.40     // 40% covered by research grants

	// Instance family discounts for common research workloads
	config.InstanceFamilyDiscounts = map[string]float64{
		"p3": 0.25, // 25% discount on GPU instances for ML research
		"r5": 0.15, // 15% discount on memory-optimized for data analysis
		"c5": 0.20, // 20% discount on compute-optimized for simulations
	}

	// Commitment modeling
	config.CommitmentPrograms.ReservedInstanceCoverage = 0.60 // 60% RI coverage
	config.CommitmentPrograms.SavingsPlanCoverage = 0.30      // 30% SP coverage
	config.CommitmentPrograms.SpotInstancePreference = 0.40   // 40% spot preference

	calculator := NewCalculator(config)

	// Test GPU instance for ML research
	result := calculator.CalculateInstanceCost("p3.2xlarge", 3.06, "us-east-1")

	assert.Equal(t, 3.06, result.ListPrice)
	assert.True(t, result.DiscountedPrice < result.ListPrice)
	assert.True(t, result.TotalDiscount > 0.5) // Should have > 50% total discount

	// Should have multiple discounts applied
	assert.True(t, len(result.AppliedDiscounts) >= 4) // Global, family, educational, RI

	// Verify it includes both daily and monthly estimates
	assert.Equal(t, result.DiscountedPrice*24, result.DailyEstimate)
	assert.Equal(t, result.DiscountedPrice*24*30, result.MonthlyEstimate)

	// Test spot discount functionality
	spotDiscount := calculator.GetSpotInstanceDiscount()
	expectedSpotDiscount := 0.70 * 0.40 // 70% base * 40% preference
	assert.InDelta(t, expectedSpotDiscount, spotDiscount, 0.001)
}
