package aws

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/scttfrdmn/cloudworkstation/pkg/types"
)

// TestSupportsHibernationExtended tests hibernation support with more cases
func TestSupportsHibernationExtended(t *testing.T) {
	manager := &Manager{}
	
	tests := []struct {
		instanceType string
		expected     bool
	}{
		{"m6i.large", true},     // Modern hibernation-capable
		{"c6a.large", true},     // AMD hibernation-capable
		{"r6g.large", true},     // Graviton hibernation-capable
		{"x1.16xlarge", true},   // Large hibernation-capable
		{"t4g.micro", false},    // Graviton T series (no hibernation)
		{"a1.medium", false},    // ARM instances without hibernation
		{"", false},             // Empty string
		{"invalid-type", false}, // Unknown type
	}
	
	for _, tt := range tests {
		t.Run(tt.instanceType, func(t *testing.T) {
			result := manager.supportsHibernation(tt.instanceType)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestAddEFSMountToUserDataComprehensive tests EFS mount addition
func TestAddEFSMountToUserDataComprehensive(t *testing.T) {
	manager := &Manager{}

	tests := []struct {
		name             string
		originalUserData string
		volumeName       string
		region           string
		expectedContains []string
	}{
		{
			name:             "Basic script with EFS mount",
			originalUserData: "#!/bin/bash\necho 'Original script'",
			volumeName:       "test-volume",
			region:           "us-east-1",
			expectedContains: []string{
				"Original script",
				"Mount EFS volume: test-volume",
				"/mnt/test-volume",
				"test-volume.efs.us-east-1.amazonaws.com",
				"/etc/fstab",
			},
		},
		{
			name:             "Empty script with EFS mount",
			originalUserData: "",
			volumeName:       "empty-volume",
			region:           "eu-west-1",
			expectedContains: []string{
				"Mount EFS volume: empty-volume",
				"/mnt/empty-volume",
				"empty-volume.efs.eu-west-1.amazonaws.com",
			},
		},
		{
			name:             "Complex script with EFS mount",
			originalUserData: "#!/bin/bash\n# Complex script\necho 'start'\nfor i in {1..10}; do echo $i; done",
			volumeName:       "complex-vol",
			region:           "ap-southeast-2",
			expectedContains: []string{
				"Complex script",
				"start",
				"Mount EFS volume: complex-vol",
				"complex-vol.efs.ap-southeast-2.amazonaws.com",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := manager.addEFSMountToUserData(tt.originalUserData, tt.volumeName, tt.region)

			for _, expected := range tt.expectedContains {
				assert.Contains(t, result, expected, "Result should contain: %s", expected)
			}

			// Verify EFS mount commands are properly formatted
			assert.Contains(t, result, "nfs-common")
		})
	}
}

// TestCalculatePerformanceParamsAllVolumeTypes tests performance calculation
func TestCalculatePerformanceParamsAllVolumeTypes(t *testing.T) {
	manager := &Manager{}

	tests := []struct {
		volumeType         string
		sizeGB             int
		expectPositiveIOPS bool
		expectThroughput   bool
		name               string
	}{
		{"gp3", 100, true, true, "gp3 with throughput"},
		{"gp3", 5000, true, true, "gp3 large volume"},
		{"io2", 500, true, false, "io2 with IOPS only"},
		{"io2", 10000, true, false, "io2 large volume"},
		{"gp2", 1000, false, false, "gp2 no performance config"},
		{"st1", 2000, false, false, "st1 no performance config"},
		{"sc1", 3000, false, false, "sc1 no performance config"},
		{"unknown", 1000, false, false, "unknown volume type"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			iops, throughput := manager.calculatePerformanceParams(tt.volumeType, tt.sizeGB)

			if tt.expectPositiveIOPS {
				assert.Greater(t, iops, 0, "IOPS should be positive for %s", tt.volumeType)
			} else {
				assert.Equal(t, 0, iops, "IOPS should be 0 for %s", tt.volumeType)
			}

			if tt.expectThroughput && tt.volumeType == "gp3" {
				assert.Greater(t, throughput, 0, "Throughput should be positive for gp3")
			}
		})
	}
}

// TestGetTemplateFromRealTemplates tests template retrieval with actual templates
func TestGetTemplateFromRealTemplates(t *testing.T) {
	manager := &Manager{
		templates: getTemplates(), // Use actual templates
	}
	
	// Test that we can get real templates
	templates := manager.GetTemplates()
	assert.NotEmpty(t, templates)
	
	// Test specific known templates exist
	expectedTemplates := []string{"r-research", "python-research"}
	for _, name := range expectedTemplates {
		template, err := manager.GetTemplate(name)
		assert.NoError(t, err, "Should find template %s", name)
		assert.NotNil(t, template)
		assert.NotEmpty(t, template.Name)
		assert.NotEmpty(t, template.Description)
	}
}

// TestGetTemplateForArchitectureBasic tests architecture template mapping
func TestGetTemplateForArchitectureBasic(t *testing.T) {
	manager := &Manager{
		templates:    getTemplates(),
		pricingCache: make(map[string]float64),
		discountConfig: types.DiscountConfig{},
	}
	
	// Get a real template for testing
	template, err := manager.GetTemplate("r-research")
	assert.NoError(t, err)
	
	tests := []struct {
		architecture string
		region       string
		shouldError  bool
		name         string
	}{
		{"x86_64", "us-east-1", false, "Valid x86_64 in us-east-1"},
		{"arm64", "us-west-2", false, "Valid arm64 in us-west-2"},
		{"invalid-arch", "us-east-1", true, "Invalid architecture"},
		{"x86_64", "invalid-region", true, "Invalid region"},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ami, instanceType, cost, err := manager.getTemplateForArchitecture(*template, tt.architecture, tt.region)
			
			if tt.shouldError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.NotEmpty(t, ami)
				assert.NotEmpty(t, instanceType)
				assert.Greater(t, cost, 0.0)
			}
		})
	}
}

// TestErrorMessages tests error message formatting
func TestErrorMessages(t *testing.T) {
	manager := &Manager{
		templates: getTemplates(),
	}
	
	// Test template not found error
	_, err := manager.GetTemplate("nonexistent-template")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "template nonexistent-template not found")
	
	// Test invalid size parsing error  
	_, err = manager.parseSizeToGB("invalid-size-xyz")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid size")
}

// TestDiscountConfigOperations tests discount configuration
func TestDiscountConfigOperations(t *testing.T) {
	manager := &Manager{
		pricingCache: make(map[string]float64),
	}
	
	// Test setting and getting discount config
	config := types.DiscountConfig{
		EC2Discount:         0.15,
		EducationalDiscount: 0.10,
		EBSDiscount:         0.05,
	}
	
	manager.SetDiscountConfig(config)
	
	result := manager.GetDiscountConfig()
	assert.Equal(t, 0.15, result.EC2Discount)
	assert.Equal(t, 0.10, result.EducationalDiscount)
	assert.Equal(t, 0.05, result.EBSDiscount)
	
	// Test that cache was cleared
	assert.Empty(t, manager.pricingCache)
}

// TestProcessIdleDetectionConfig tests idle detection configuration processing
func TestProcessIdleDetectionConfig(t *testing.T) {
	manager := &Manager{}
	
	tests := []struct {
		name             string
		userData         string
		template         *types.RuntimeTemplate
		expectedContains []string
	}{
		{
			name:     "Replace idle detection placeholders",
			userData: "#!/bin/bash\necho 'start'\n{{IDLE_THRESHOLD_MINUTES}}\n{{HIBERNATE_THRESHOLD_MINUTES}}\n{{CHECK_INTERVAL_MINUTES}}\necho 'end'",
			template: &types.RuntimeTemplate{
				IdleDetection: &types.IdleDetectionConfig{
					Enabled:                   true,
					CheckIntervalMinutes:      1,
					IdleThresholdMinutes:     5,
					HibernateThresholdMinutes: 10,
				},
			},
			expectedContains: []string{
				"start",
				"end",
				"1", // CHECK_INTERVAL_MINUTES replacement
				"5", // IDLE_THRESHOLD_MINUTES replacement
				"10", // HIBERNATE_THRESHOLD_MINUTES replacement
			},
		},
		{
			name:     "No idle detection config",
			userData: "#!/bin/bash\necho 'no config'",
			template: &types.RuntimeTemplate{
				IdleDetection: nil,
			},
			expectedContains: []string{
				"no config",
			},
		},
		{
			name:     "No placeholders in userdata",
			userData: "#!/bin/bash\necho 'no placeholder'",
			template: &types.RuntimeTemplate{
				IdleDetection: &types.IdleDetectionConfig{
					Enabled: true,
					CheckIntervalMinutes: 5,
				},
			},
			expectedContains: []string{
				"no placeholder",
			},
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := manager.processIdleDetectionConfig(tt.userData, tt.template)
			
			for _, expected := range tt.expectedContains {
				assert.Contains(t, result, expected, "Result should contain: %s", expected)
			}
		})
	}
}

// TestLocalArchitectureDetection tests local architecture detection
func TestLocalArchitectureDetection(t *testing.T) {
	manager := &Manager{}
	
	arch := manager.getLocalArchitecture()
	
	// Should return either x86_64 or arm64
	assert.True(t, arch == "x86_64" || arch == "arm64", "Architecture should be x86_64 or arm64, got: %s", arch)
	assert.NotEmpty(t, arch)
}

// TestGetBillingInfoStructure tests billing info structure
func TestGetBillingInfoStructure(t *testing.T) {
	manager := &Manager{}
	
	info, err := manager.GetBillingInfo()
	assert.NoError(t, err)
	assert.NotNil(t, info)
	
	// Check required fields
	assert.NotEmpty(t, info.BillingPeriod)
	assert.False(t, info.LastUpdated.IsZero())
	assert.NotNil(t, info.Credits)
	assert.NotEmpty(t, info.Credits)
	
	// Check first credit entry
	credit := info.Credits[0]
	assert.NotEmpty(t, credit.CreditType)
	assert.NotEmpty(t, credit.Description)
}

// TestDetectPotentialCreditsExtended tests credit detection with detailed validation
func TestDetectPotentialCreditsExtended(t *testing.T) {
	manager := &Manager{}
	
	credits := manager.detectPotentialCredits()
	assert.NotEmpty(t, credits)
	
	// Should have at least one mock credit
	assert.GreaterOrEqual(t, len(credits), 1)
	
	// Check credit structure
	for _, credit := range credits {
		assert.NotEmpty(t, credit.CreditType)
		assert.NotEmpty(t, credit.Description)
	}
	
	// Check that we have the expected AWS Credits entry
	found := false
	for _, credit := range credits {
		if credit.CreditType == "AWS Credits" {
			found = true
			break
		}
	}
	assert.True(t, found, "Should find AWS Credits entry")
}

// TestHibernationSupport tests hibernation support detection
func TestHibernationSupport(t *testing.T) {
	manager := &Manager{}
	
	tests := []struct {
		instanceType string
		expected     bool
		name         string
	}{
		// Hibernation-capable instances (based on actual AWS implementation)
		{"t2.micro", true, "T2 micro (supported)"},
		{"t3.medium", true, "T3 medium (supported)"},
		{"t3a.large", true, "T3a AMD (supported)"},
		{"m5.large", true, "M5 generation"},
		{"m6i.large", true, "M6i latest generation"},
		{"c5.large", true, "C5 compute optimized"},
		{"c6g.xlarge", true, "C6g Graviton2"},
		{"r5.large", true, "R5 memory optimized"},
		{"r6g.medium", true, "R6g Graviton2"},
		{"x1.16xlarge", true, "X1 high memory"},
		{"x1e.xlarge", true, "X1e enhanced"},
		{"g4dn.xlarge", true, "G4dn GPU (supported)"},
		
		// Non-hibernation instances (unsupported families)
		{"a1.medium", false, "A1 ARM family (not in supported list)"},
		{"i3.large", false, "I3 storage optimized (not supported)"},
		{"d2.xlarge", false, "D2 dense storage (not supported)"},
		{"f1.2xlarge", false, "F1 FPGA (not supported)"},
		{"p3.2xlarge", false, "P3 GPU (not supported)"},
		{"inf1.xlarge", false, "Inf1 inference (not supported)"},
		{"", false, "Empty instance type"},
		{"invalid-type", false, "Unknown instance type"},
		{"no-dot-type", false, "Type without dot separator"},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := manager.supportsHibernation(tt.instanceType)
			assert.Equal(t, tt.expected, result, "Instance type %s hibernation support", tt.instanceType)
		})
	}
}

// TestEstimateInstancePriceExtended tests instance price estimation edge cases
func TestEstimateInstancePriceExtended(t *testing.T) {
	manager := &Manager{}
	
	tests := []struct {
		instanceType string
		minPrice     float64 // Minimum expected price
		name         string
	}{
		// Test various instance families for positive pricing
		{"t2.nano", 0.001, "t2.nano pricing"},
		{"t2.micro", 0.001, "t2.micro pricing"}, 
		{"t2.small", 0.001, "t2.small pricing"},
		{"t2.medium", 0.001, "t2.medium pricing"},
		{"t2.large", 0.001, "t2.large pricing"},
		{"m4.large", 0.001, "m4.large pricing"},
		{"m4.xlarge", 0.001, "m4.xlarge pricing"},
		{"c4.large", 0.001, "c4.large pricing"},
		{"c4.xlarge", 0.001, "c4.xlarge pricing"},
		{"r4.large", 0.001, "r4.large pricing"},
		{"r4.xlarge", 0.001, "r4.xlarge pricing"},
		{"g3.4xlarge", 0.001, "g3.4xlarge pricing"},
		{"p2.xlarge", 0.001, "p2.xlarge pricing"},
		{"unknown.type", 0.001, "unknown type fallback"},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := manager.estimateInstancePrice(tt.instanceType)
			assert.GreaterOrEqual(t, result, tt.minPrice, "Price for %s should be at least %f", tt.instanceType, tt.minPrice)
		})
	}
}

// TestGetRegionalEC2PriceEdgeCases tests regional pricing edge cases
func TestGetRegionalEC2PriceEdgeCases(t *testing.T) {
	manager := &Manager{
		region:          "us-east-1",
		pricingCache:    make(map[string]float64),
		lastPriceUpdate: time.Time{},
		discountConfig:  types.DiscountConfig{},
	}
	
	tests := []struct {
		instanceType string
		region       string
		name         string
	}{
		{"t3.medium", "us-east-1", "US East pricing"},
		{"t3.medium", "eu-west-1", "EU West pricing"},
		{"t3.medium", "ap-southeast-1", "Asia Pacific pricing"},
		{"c5.large", "us-west-2", "US West pricing"},
		{"r5.xlarge", "ca-central-1", "Canada pricing"},
		{"unknown.type", "us-east-1", "Unknown instance fallback"},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manager.region = tt.region
			manager.pricingCache = make(map[string]float64) // Clear cache
			
			price := manager.getRegionalEC2Price(tt.instanceType)
			assert.Greater(t, price, 0.0, "Price should be positive for %s in %s", tt.instanceType, tt.region)
		})
	}
}

// TestApplyEC2DiscountsEdgeCases tests discount application edge cases
func TestApplyEC2DiscountsEdgeCases(t *testing.T) {
	manager := &Manager{}
	basePrice := 100.0
	
	tests := []struct {
		name           string
		discountConfig types.DiscountConfig
		expectedPrice  float64
	}{
		{
			name: "All EC2 discounts combined",
			discountConfig: types.DiscountConfig{
				EC2Discount:          0.10, // 10%
				SavingsPlansDiscount: 0.15, // 15%
				EducationalDiscount:  0.20, // 20%
				EnterpriseDiscount:   0.05, // 5%
			},
			expectedPrice: 100 * 0.9 * 0.85 * 0.8 * 0.95, // 58.14
		},
		{
			name: "Only educational discount",
			discountConfig: types.DiscountConfig{
				EducationalDiscount: 0.25, // 25%
			},
			expectedPrice: 75.0,
		},
		{
			name: "Only enterprise discount",
			discountConfig: types.DiscountConfig{
				EnterpriseDiscount: 0.15, // 15%
			},
			expectedPrice: 85.0,
		},
		{
			name: "Only savings plans discount",
			discountConfig: types.DiscountConfig{
				SavingsPlansDiscount: 0.30, // 30%
			},
			expectedPrice: 70.0,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manager.discountConfig = tt.discountConfig
			result := manager.applyEC2Discounts(basePrice)
			
			// Use a small tolerance for floating point comparison
			tolerance := 0.01
			if abs := result - tt.expectedPrice; abs < 0 {
				abs = -abs
			} else if abs > tolerance {
				t.Errorf("applyEC2Discounts() = %f, want %f (tolerance: %f)", result, tt.expectedPrice, tolerance)
			}
		})
	}
}

// TestGetRegionPricingMultiplierComplete tests region pricing logic
func TestGetRegionPricingMultiplierComplete(t *testing.T) {
	manager := &Manager{}
	
	tests := []struct {
		region      string
		minPrice    float64
		maxPrice    float64
		name        string
	}{
		// Test key regions with expected ranges
		{"us-east-1", 1.0, 1.0, "US East 1 base"},
		{"us-east-2", 0.98, 0.98, "US East 2"},
		{"us-west-1", 1.05, 1.05, "US West 1"},
		{"us-west-2", 1.05, 1.05, "US West 2"},
		{"eu-west-1", 1.10, 1.10, "EU Ireland"},
		{"eu-west-2", 1.12, 1.12, "EU London"}, 
		{"eu-west-3", 1.15, 1.15, "EU Paris"},
		{"eu-central-1", 1.18, 1.18, "EU Frankfurt"},
		{"ap-southeast-1", 1.20, 1.20, "Asia Singapore"},
		{"ap-southeast-2", 1.25, 1.25, "Asia Sydney"},
		{"ap-northeast-1", 1.22, 1.22, "Asia Tokyo"},
		{"ap-northeast-2", 1.18, 1.18, "Asia Seoul"},
		{"ap-south-1", 1.05, 1.05, "Asia Mumbai"},
		{"ca-central-1", 1.08, 1.08, "Canada"},
		{"sa-east-1", 1.30, 1.30, "South America"},
		{"unknown-region-xyz", 1.15, 1.15, "Unknown region fallback"},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manager.region = tt.region
			result := manager.getRegionPricingMultiplier()
			assert.GreaterOrEqual(t, result, tt.minPrice, "Region multiplier for %s should be at least %f", tt.region, tt.minPrice)
			assert.LessOrEqual(t, result, tt.maxPrice, "Region multiplier for %s should be at most %f", tt.region, tt.maxPrice)
		})
	}
}

// TestCacheExpirationLogic tests cache expiration behavior
func TestCacheExpirationLogic(t *testing.T) {
	manager := &Manager{
		region:          "us-east-1",
		pricingCache:    make(map[string]float64),
		lastPriceUpdate: time.Time{},
		discountConfig:  types.DiscountConfig{},
	}
	
	t.Run("Fresh cache behavior", func(t *testing.T) {
		// Set fresh cache
		manager.lastPriceUpdate = time.Now()
		manager.pricingCache["test-key"] = 1.23
		
		// Verify cache is considered fresh
		assert.True(t, time.Since(manager.lastPriceUpdate) < 24*time.Hour)
	})
	
	t.Run("Expired cache behavior", func(t *testing.T) {
		// Set expired cache  
		manager.lastPriceUpdate = time.Now().Add(-25 * time.Hour)
		
		// Verify cache is considered expired
		assert.True(t, time.Since(manager.lastPriceUpdate) > 24*time.Hour)
	})
	
	t.Run("Cache operations with regional pricing", func(t *testing.T) {
		manager.pricingCache = make(map[string]float64)
		manager.lastPriceUpdate = time.Time{}
		
		// First call should populate cache
		price1 := manager.getRegionalEC2Price("t3.medium")
		assert.Greater(t, price1, 0.0)
		
		// Cache should be populated
		assert.NotEmpty(t, manager.pricingCache)
		assert.False(t, manager.lastPriceUpdate.IsZero())
		
		// Second call should use cache (same result)
		price2 := manager.getRegionalEC2Price("t3.medium")
		assert.Equal(t, price1, price2)
	})
}

// TestEBSVolumeTypeHandling tests EBS volume type handling edge cases
func TestEBSVolumeTypeHandling(t *testing.T) {
	manager := &Manager{
		region:         "us-east-1",
		pricingCache:   make(map[string]float64),
		discountConfig: types.DiscountConfig{},
	}
	
	volumeTypes := []string{"gp3", "gp2", "io2", "st1", "sc1", "unknown-type"}
	
	for _, volumeType := range volumeTypes {
		t.Run(fmt.Sprintf("Volume type %s", volumeType), func(t *testing.T) {
			// Test regional EBS pricing
			regionalPrice := manager.getRegionalEBSPrice(volumeType)
			assert.Greater(t, regionalPrice, 0.0, "Regional EBS price should be positive for %s", volumeType)
			
			// Test cost per GB calculation
			costPerGB := manager.getEBSCostPerGB(volumeType)
			assert.Greater(t, costPerGB, 0.0, "Cost per GB should be positive for %s", volumeType)
		})
	}
}