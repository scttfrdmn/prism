package aws

import (
	"math"
	"strings"
	"testing"

	ec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	ctypes "github.com/scttfrdmn/cloudworkstation/pkg/types"
)

// floatEquals checks if two floats are equal within a small tolerance
func floatEquals(a, b float64) bool {
	return math.Abs(a-b) < 0.001
}

func TestGetRegionPricingMultiplier(t *testing.T) {
	manager := &Manager{region: "us-east-1"}

	tests := []struct {
		region   string
		expected float64
		name     string
	}{
		{"us-east-1", 1.0, "US East 1 base pricing"},
		{"us-east-2", 0.98, "US East 2 slightly cheaper"},
		{"us-west-1", 1.05, "US West coast premium"},
		{"us-west-2", 1.05, "US West coast premium"},
		{"eu-west-1", 1.10, "Ireland pricing"},
		{"eu-west-2", 1.12, "London pricing"},
		{"eu-central-1", 1.18, "Frankfurt pricing"},
		{"ap-southeast-1", 1.20, "Singapore pricing"},
		{"ap-southeast-2", 1.25, "Sydney pricing"},
		{"ap-northeast-1", 1.22, "Tokyo pricing"},
		{"ap-south-1", 1.05, "Mumbai pricing"},
		{"ca-central-1", 1.08, "Canada pricing"},
		{"sa-east-1", 1.30, "São Paulo pricing"},
		{"unknown-region", 1.15, "Unknown region default"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manager.region = tt.region
			result := manager.getRegionPricingMultiplier()
			if result != tt.expected {
				t.Errorf("getRegionPricingMultiplier(%s) = %f, want %f", tt.region, result, tt.expected)
			}
		})
	}
}

func TestEstimateInstancePrice(t *testing.T) {
	manager := &Manager{}

	tests := []struct {
		instanceType string
		expected     float64
		name         string
	}{
		{"t3.micro", 0.0052, "t3.micro pricing"},       // 0.0104 * 0.5
		{"t3.small", 0.0104, "t3.small pricing"},       // 0.0104 * 1.0
		{"t3.medium", 0.0208, "t3.medium pricing"},     // 0.0104 * 2.0
		{"t3.large", 0.0416, "t3.large pricing"},       // 0.0104 * 4.0
		{"c5.large", 0.34, "c5.large pricing"},         // 0.085 * 4.0
		{"r5.xlarge", 1.008, "r5.xlarge pricing"},      // 0.126 * 8.0
		{"invalid", 0.10, "invalid instance fallback"}, // Conservative fallback
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := manager.estimateInstancePrice(tt.instanceType)
			if result != tt.expected {
				t.Errorf("estimateInstancePrice(%s) = %f, want %f", tt.instanceType, result, tt.expected)
			}
		})
	}
}

func TestParseSizeToGB(t *testing.T) {
	manager := &Manager{}

	tests := []struct {
		size        string
		expected    int
		expectError bool
		name        string
	}{
		{"XS", 100, false, "XS size"},
		{"xs", 100, false, "lowercase xs size"},
		{"S", 500, false, "S size"},
		{"M", 1000, false, "M size"},
		{"L", 2000, false, "L size"},
		{"XL", 4000, false, "XL size"},
		{"500", 500, false, "Direct GB value"},
		{"1000", 1000, false, "Direct GB value"},
		{"invalid", 0, true, "Invalid size"},
		{"0", 0, true, "Zero GB"},
		{"-100", 0, true, "Negative GB"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := manager.parseSizeToGB(tt.size)

			if tt.expectError {
				if err == nil {
					t.Errorf("parseSizeToGB(%s) expected error, got nil", tt.size)
				}
			} else {
				if err != nil {
					t.Errorf("parseSizeToGB(%s) unexpected error: %v", tt.size, err)
				}
				if result != tt.expected {
					t.Errorf("parseSizeToGB(%s) = %d, want %d", tt.size, result, tt.expected)
				}
			}
		})
	}
}

func TestCalculatePerformanceParams(t *testing.T) {
	manager := &Manager{}

	tests := []struct {
		volumeType         string
		sizeGB             int
		expectedIOPS       int
		expectedThroughput int
		name               string
	}{
		{"gp3", 100, 3000, 125, "gp3 small volume"},
		{"gp3", 1000, 3000, 250, "gp3 medium volume"},
		{"gp3", 10000, 16000, 1000, "gp3 large volume (capped)"},
		{"io2", 100, 1000, 0, "io2 small volume"},
		{"io2", 10000, 64000, 0, "io2 large volume (capped)"},
		{"gp2", 1000, 0, 0, "gp2 no IOPS config"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			iops, throughput := manager.calculatePerformanceParams(tt.volumeType, tt.sizeGB)

			if iops != tt.expectedIOPS {
				t.Errorf("calculatePerformanceParams(%s, %d) IOPS = %d, want %d",
					tt.volumeType, tt.sizeGB, iops, tt.expectedIOPS)
			}

			if throughput != tt.expectedThroughput {
				t.Errorf("calculatePerformanceParams(%s, %d) throughput = %d, want %d",
					tt.volumeType, tt.sizeGB, throughput, tt.expectedThroughput)
			}
		})
	}
}

func TestApplyDiscounts(t *testing.T) {
	manager := &Manager{}
	basePrice := 100.0

	t.Run("No discounts", func(t *testing.T) {
		result := manager.applyEC2Discounts(basePrice)
		if result != basePrice {
			t.Errorf("applyEC2Discounts(%f) with no discounts = %f, want %f", basePrice, result, basePrice)
		}
	})

	t.Run("Single discount", func(t *testing.T) {
		manager.discountConfig = ctypes.DiscountConfig{
			EC2Discount: 0.20, // 20% discount
		}
		expected := 80.0 // 100 * (1 - 0.20)
		result := manager.applyEC2Discounts(basePrice)
		if result != expected {
			t.Errorf("applyEC2Discounts(%f) with 20%% discount = %f, want %f", basePrice, result, expected)
		}
	})

	t.Run("Multiple discounts", func(t *testing.T) {
		manager.discountConfig = ctypes.DiscountConfig{
			EC2Discount:          0.10, // 10% discount
			SavingsPlansDiscount: 0.15, // 15% additional discount
			EducationalDiscount:  0.10, // 10% educational discount
		}
		// 100 * 0.9 * 0.85 * 0.9 = 68.85
		expected := 68.85
		result := manager.applyEC2Discounts(basePrice)
		if !floatEquals(result, expected) {
			t.Errorf("applyEC2Discounts(%f) with multiple discounts = %f, want %f", basePrice, result, expected)
		}
	})

	t.Run("EBS discounts", func(t *testing.T) {
		manager.discountConfig = ctypes.DiscountConfig{
			EBSDiscount:    0.15, // 15% EBS discount
			VolumeDiscount: 0.05, // 5% volume discount
		}
		// 100 * 0.85 * 0.95 = 80.75
		expected := 80.75
		result := manager.applyEBSDiscounts(basePrice)
		if !floatEquals(result, expected) {
			t.Errorf("applyEBSDiscounts(%f) with EBS discounts = %f, want %f", basePrice, result, expected)
		}
	})

	t.Run("EFS discounts", func(t *testing.T) {
		manager.discountConfig = ctypes.DiscountConfig{
			EFSDiscount:        0.10, // 10% EFS discount
			EnterpriseDiscount: 0.20, // 20% enterprise discount
		}
		// 100 * 0.9 * 0.8 = 72.0
		expected := 72.0
		result := manager.applyEFSDiscounts(basePrice)
		if !floatEquals(result, expected) {
			t.Errorf("applyEFSDiscounts(%f) with EFS discounts = %f, want %f", basePrice, result, expected)
		}
	})
}

func TestDiscountConfigManagement(t *testing.T) {
	manager := &Manager{}

	// Set some initial discount config
	manager.discountConfig = ctypes.DiscountConfig{
		EC2Discount: 0.10,
	}

	newConfig := ctypes.DiscountConfig{
		EC2Discount:         0.15,
		EducationalDiscount: 0.10,
	}

	// Test setting discount config
	manager.SetDiscountConfig(newConfig)

	// Check that config was set
	result := manager.GetDiscountConfig()
	if result.EC2Discount != 0.15 {
		t.Errorf("GetDiscountConfig() EC2Discount = %f, want %f", result.EC2Discount, 0.15)
	}
	if result.EducationalDiscount != 0.10 {
		t.Errorf("GetDiscountConfig() EducationalDiscount = %f, want %f", result.EducationalDiscount, 0.10)
	}
}

func TestGetBillingInfo(t *testing.T) {
	manager := &Manager{}

	info, err := manager.GetBillingInfo()
	if err != nil {
		t.Errorf("GetBillingInfo() unexpected error: %v", err)
	}

	if info == nil {
		t.Fatal("GetBillingInfo() returned nil")
	}

	// Check basic structure
	if info.BillingPeriod == "" {
		t.Error("GetBillingInfo() BillingPeriod should not be empty")
	}

	if info.LastUpdated.IsZero() {
		t.Error("GetBillingInfo() LastUpdated should not be zero")
	}

	if info.Credits == nil {
		t.Error("GetBillingInfo() Credits should not be nil")
	}

	// Should have at least one mock credit entry
	if len(info.Credits) == 0 {
		t.Error("GetBillingInfo() should return at least one credit entry")
	}
}

func TestGetLocalArchitecture(t *testing.T) {
	manager := &Manager{}

	arch := manager.getLocalArchitecture()

	// Should return either x86_64 or arm64
	if arch != "x86_64" && arch != "arm64" {
		t.Errorf("getLocalArchitecture() = %s, want x86_64 or arm64", arch)
	}
}

func TestGetTemplates(t *testing.T) {
	manager := &Manager{
		templates: getTemplates(),
	}

	templates := manager.GetTemplates()

	if len(templates) == 0 {
		t.Error("GetTemplates() should return at least one template")
	}

	// Check for expected templates
	expectedTemplates := []string{"r-research", "python-research", "basic-ubuntu"}
	for _, expected := range expectedTemplates {
		if _, exists := templates[expected]; !exists {
			t.Errorf("GetTemplates() missing expected template: %s", expected)
		}
	}

	// Test GetTemplate function
	template, err := manager.GetTemplate("r-research")
	if err != nil {
		t.Errorf("GetTemplate(r-research) unexpected error: %v", err)
	}
	if template == nil {
		t.Error("GetTemplate(r-research) returned nil")
	}

	// Test non-existent template
	_, err = manager.GetTemplate("non-existent")
	if err == nil {
		t.Error("GetTemplate(non-existent) should return error")
	}
}

func TestManagerCreation(t *testing.T) {
	// Test that NewManager would create a properly initialized manager
	// Note: We can't actually test NewManager() without AWS credentials
	// but we can test the initialization logic

	manager := &Manager{
		discountConfig: ctypes.DiscountConfig{},
		templates:      getTemplates(),
		region:         "us-east-1",
	}

	// Test that manager is properly initialized
	if len(manager.templates) == 0 {
		t.Error("Manager should have templates loaded")
	}

	if manager.region == "" {
		t.Error("Manager should have a region set")
	}
}

func TestPricingCacheLogic(t *testing.T) {
	manager := &Manager{
		region:         "us-east-1",
		discountConfig: ctypes.DiscountConfig{},
	}

	t.Run("Pricing consistency", func(t *testing.T) {
		// First call
		price1 := manager.getRegionalEC2Price("t3.medium")

		// Second call should return consistent price
		price2 := manager.getRegionalEC2Price("t3.medium")

		if !floatEquals(price1, price2) {
			t.Errorf("Price should be consistent: %f vs %f", price1, price2)
		}
	})

	t.Run("Regional pricing differences", func(t *testing.T) {
		manager.region = "us-east-1"
		priceUSEast := manager.getRegionalEC2Price("t3.medium")

		manager.region = "eu-west-1"
		priceEUWest := manager.getRegionalEC2Price("t3.medium")

		if floatEquals(priceUSEast, priceEUWest) {
			t.Error("Regional pricing should differ between us-east-1 and eu-west-1")
		}

		// EU should be more expensive
		if priceEUWest <= priceUSEast {
			t.Errorf("EU pricing (%f) should be higher than US East (%f)", priceEUWest, priceUSEast)
		}
	})
}

func TestGetEBSCostPerGB(t *testing.T) {
	manager := &Manager{
		region:         "us-east-1",
		discountConfig: ctypes.DiscountConfig{},
	}

	tests := []struct {
		volumeType string
		name       string
	}{
		{"gp3", "gp3 pricing"},
		{"gp2", "gp2 pricing"},
		{"io2", "io2 pricing"},
		{"st1", "st1 pricing"},
		{"sc1", "sc1 pricing"},
		{"unknown", "unknown volume type fallback"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			price := manager.getEBSCostPerGB(tt.volumeType)
			if price <= 0 {
				t.Errorf("getEBSCostPerGB(%s) should return positive price, got %f", tt.volumeType, price)
			}
		})
	}

	t.Run("EBS pricing with discounts", func(t *testing.T) {
		manager.discountConfig = ctypes.DiscountConfig{
			EBSDiscount: 0.20, // 20% discount
		}

		basePriceNoDiscount := manager.getRegionalEBSPrice("gp3")
		priceWithDiscount := manager.getEBSCostPerGB("gp3")

		expectedWithDiscount := basePriceNoDiscount * 0.8 // 20% off
		if !floatEquals(priceWithDiscount, expectedWithDiscount) {
			t.Errorf("EBS price with 20%% discount = %f, want %f", priceWithDiscount, expectedWithDiscount)
		}
	})
}

func TestGetRegionalEFSPrice(t *testing.T) {
	manager := &Manager{
		region:         "us-east-1",
		discountConfig: ctypes.DiscountConfig{},
	}

	t.Run("EFS pricing without discounts", func(t *testing.T) {
		price := manager.getRegionalEFSPrice()
		expectedBase := 0.30 // US East 1 base price
		if !floatEquals(price, expectedBase) {
			t.Errorf("getRegionalEFSPrice() in us-east-1 = %f, want %f", price, expectedBase)
		}
	})

	t.Run("EFS pricing with discounts", func(t *testing.T) {
		manager.discountConfig = ctypes.DiscountConfig{
			EFSDiscount: 0.15, // 15% discount
		}

		price := manager.getRegionalEFSPrice()
		expected := 0.30 * 0.85 // 15% off base price
		if !floatEquals(price, expected) {
			t.Errorf("getRegionalEFSPrice() with 15%% discount = %f, want %f", price, expected)
		}
	})

	t.Run("EFS regional pricing", func(t *testing.T) {
		manager.discountConfig = ctypes.DiscountConfig{} // Reset discounts
		manager.region = "ap-southeast-2"

		price := manager.getRegionalEFSPrice()
		expected := 0.30 * 1.25 // Sydney multiplier
		if !floatEquals(price, expected) {
			t.Errorf("getRegionalEFSPrice() in ap-southeast-2 = %f, want %f", price, expected)
		}
	})
}

func TestAddEFSMountToUserData(t *testing.T) {
	manager := &Manager{}

	originalUserData := "#!/bin/bash\necho 'Original script'"
	volumeName := "test-volume"
	region := "us-east-1"

	result := manager.addEFSMountToUserData(originalUserData, volumeName, region)

	// Check that original data is preserved
	if !strings.Contains(result, "Original script") {
		t.Error("Original user data should be preserved")
	}

	// Check that EFS mount commands are added
	if !strings.Contains(result, "Mount EFS volume: test-volume") {
		t.Error("EFS mount comment should be added")
	}

	if !strings.Contains(result, "/mnt/test-volume") {
		t.Error("Mount directory should be created")
	}

	if !strings.Contains(result, "test-volume.efs.us-east-1.amazonaws.com") {
		t.Error("EFS mount target should be correctly formatted")
	}

	if !strings.Contains(result, "/etc/fstab") {
		t.Error("fstab entry should be added")
	}
}

func TestGetTemplateForArchitecture(t *testing.T) {
	manager := &Manager{
		templates: getTemplates(),
	}

	// Get a known template for testing
	template := manager.templates["r-research"]

	t.Run("Valid architecture and region", func(t *testing.T) {
		ami, instanceType, cost, err := manager.getTemplateForArchitecture(template, "x86_64", "us-east-1")

		if err != nil {
			t.Errorf("getTemplateForArchitecture() unexpected error: %v", err)
		}

		if ami == "" {
			t.Error("AMI should not be empty")
		}

		if instanceType == "" {
			t.Error("Instance type should not be empty")
		}

		if cost <= 0 {
			t.Error("Cost should be positive")
		}
	})

	t.Run("Invalid region", func(t *testing.T) {
		_, _, _, err := manager.getTemplateForArchitecture(template, "x86_64", "invalid-region")

		if err == nil {
			t.Error("getTemplateForArchitecture() should return error for invalid region")
		}
	})

	t.Run("Invalid architecture", func(t *testing.T) {
		_, _, _, err := manager.getTemplateForArchitecture(template, "invalid-arch", "us-east-1")

		if err == nil {
			t.Error("getTemplateForArchitecture() should return error for invalid architecture")
		}
	})
}

func TestDetectPotentialCredits(t *testing.T) {
	manager := &Manager{}

	credits := manager.detectPotentialCredits()

	if len(credits) == 0 {
		t.Error("detectPotentialCredits() should return at least one credit entry")
	}

	// Check the mock credit entry
	credit := credits[0]
	if credit.CreditType != "AWS Credits" {
		t.Errorf("Credit type = %s, want 'AWS Credits'", credit.CreditType)
	}

	if credit.Description == "" {
		t.Error("Credit description should not be empty")
	}
}

func TestGetDefaultRegion(t *testing.T) {
	manager := &Manager{region: "us-west-2"}

	region := manager.GetDefaultRegion()
	if region != "us-west-2" {
		t.Errorf("GetDefaultRegion() = %s, want us-west-2", region)
	}
}

func TestParseSizeToGBError(t *testing.T) {
	manager := &Manager{}

	tests := []struct {
		size        string
		expectError bool
		name        string
	}{
		{"invalid", true, "Invalid string should error"},
		{"0", true, "Zero GB should error"},
		{"-100", true, "Negative GB should error"},
		{"abc123", true, "Mixed alphanumeric should error"},
		{"XS", false, "XS should not error"},
		{"100", false, "Valid number should not error"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := manager.parseSizeToGB(tt.size)
			if tt.expectError && err == nil {
				t.Errorf("parseSizeToGB(%s) expected error, got nil", tt.size)
			}
			if !tt.expectError && err != nil {
				t.Errorf("parseSizeToGB(%s) unexpected error: %v", tt.size, err)
			}
		})
	}
}

// Additional comprehensive tests to reach 85% coverage

func TestGetRegionalEBSPrice(t *testing.T) {
	manager := &Manager{
		region:         "us-east-1",
		discountConfig: ctypes.DiscountConfig{},
	}

	tests := []struct {
		volumeType string
		name       string
	}{
		{"gp3", "gp3 pricing"},
		{"gp2", "gp2 pricing"},
		{"io2", "io2 pricing"},
		{"st1", "st1 pricing"},
		{"sc1", "sc1 pricing"},
		{"unknown", "unknown volume type fallback"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			price := manager.getRegionalEBSPrice(tt.volumeType)
			if price <= 0 {
				t.Errorf("getRegionalEBSPrice(%s) should return positive price, got %f", tt.volumeType, price)
			}
		})
	}
}

func TestAddEFSMountToUserDataDetailed(t *testing.T) {
	manager := &Manager{}

	tests := []struct {
		originalUserData string
		volumeName       string
		region           string
		name             string
	}{
		{"#!/bin/bash\necho 'test'", "volume1", "us-east-1", "basic script"},
		{"", "empty-volume", "eu-west-1", "empty script"},
		{"#!/bin/bash\n# Complex script\necho 'start'\n", "complex-vol", "ap-southeast-2", "complex script"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := manager.addEFSMountToUserData(tt.originalUserData, tt.volumeName, tt.region)

			// Check EFS mount is added
			if !strings.Contains(result, tt.volumeName) {
				t.Error("Result should contain volume name")
			}

			if !strings.Contains(result, tt.region) {
				t.Error("Result should contain region")
			}

			if !strings.Contains(result, "efs") {
				t.Error("Result should contain EFS mount commands")
			}
		})
	}
}

func TestEstimateInstancePriceComprehensive(t *testing.T) {
	manager := &Manager{}

	tests := []struct {
		instanceType string
		name         string
	}{
		{"t2.micro", "t2.micro pricing"},
		{"t2.small", "t2.small pricing"},
		{"m5.large", "m5.large pricing"},
		{"c5.xlarge", "c5.xlarge pricing"},
		{"r5.2xlarge", "r5.2xlarge pricing"},
		{"unknown-instance", "unknown instance fallback"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			price := manager.estimateInstancePrice(tt.instanceType)
			if price <= 0 {
				t.Errorf("estimateInstancePrice(%s) should return positive price, got %f", tt.instanceType, price)
			}
		})
	}
}

func TestCalculatePerformanceParamsComprehensive(t *testing.T) {
	manager := &Manager{}

	tests := []struct {
		volumeType         string
		sizeGB             int
		expectPositiveIOPS bool
		name               string
	}{
		{"gp3", 50, true, "gp3 small volume"},
		{"gp3", 5000, true, "gp3 large volume"},
		{"io2", 100, true, "io2 volume"},
		{"io2", 5000, true, "io2 large volume"},
		{"gp2", 1000, false, "gp2 volume (no IOPS config)"},
		{"st1", 1000, false, "st1 volume (no IOPS config)"},
		{"sc1", 1000, false, "sc1 volume (no IOPS config)"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			iops, throughput := manager.calculatePerformanceParams(tt.volumeType, tt.sizeGB)

			if tt.expectPositiveIOPS && iops <= 0 {
				t.Errorf("calculatePerformanceParams(%s, %d) IOPS should be positive, got %d", tt.volumeType, tt.sizeGB, iops)
			}

			if !tt.expectPositiveIOPS && iops != 0 {
				t.Errorf("calculatePerformanceParams(%s, %d) IOPS should be 0, got %d", tt.volumeType, tt.sizeGB, iops)
			}

			// gp3 should have throughput, others shouldn't
			if tt.volumeType == "gp3" && throughput <= 0 {
				t.Errorf("gp3 volumes should have positive throughput, got %d", throughput)
			}
		})
	}
}

func TestErrorHandling(t *testing.T) {
	manager := &Manager{}

	t.Run("Invalid size parsing", func(t *testing.T) {
		_, err := manager.parseSizeToGB("invalid-size")
		if err == nil {
			t.Error("parseSizeToGB should return error for invalid size")
		}
	})

	t.Run("Template validation", func(t *testing.T) {
		_, err := manager.GetTemplate("non-existent-template")
		if err == nil {
			t.Error("GetTemplate should return error for non-existent template")
		}
	})
}

// TestCacheInvalidation is no longer relevant since pricingCache
// is now managed internally by PricingClient

func TestRegionalPricingWithCache(t *testing.T) {
	manager := &Manager{
		region:         "us-east-1",
		discountConfig: ctypes.DiscountConfig{},
	}

	t.Run("EC2 price consistency", func(t *testing.T) {
		price1 := manager.getRegionalEC2Price("t3.medium")
		price2 := manager.getRegionalEC2Price("t3.medium")

		if !floatEquals(price1, price2) {
			t.Errorf("Prices should be consistent: %f vs %f", price1, price2)
		}
	})

	t.Run("EBS price consistency", func(t *testing.T) {
		price1 := manager.getRegionalEBSPrice("gp3")
		price2 := manager.getRegionalEBSPrice("gp3")

		if !floatEquals(price1, price2) {
			t.Errorf("EBS prices should be consistent: %f vs %f", price1, price2)
		}
	})

	t.Run("EFS price consistency", func(t *testing.T) {
		price1 := manager.getRegionalEFSPrice()
		price2 := manager.getRegionalEFSPrice()

		if !floatEquals(price1, price2) {
			t.Errorf("EFS prices should be consistent: %f vs %f", price1, price2)
		}
	})
}

// TestInstanceStorageGBIsSet verifies that StorageGB field is set during instance creation (Issue #68)
func TestInstanceStorageGBIsSet(t *testing.T) {
	launcher := &InstanceLauncher{
		region: "us-west-2",
	}

	// Create a mock EC2 instance response
	mockEC2Instance := &ec2types.Instance{
		InstanceId:   strPtr("i-1234567890abcdef0"),
		InstanceType: ec2types.InstanceTypeT3Medium,
		State: &ec2types.InstanceState{
			Name: ec2types.InstanceStateNameRunning,
		},
		Placement: &ec2types.Placement{
			AvailabilityZone: strPtr("us-west-2a"),
		},
	}

	req := ctypes.LaunchRequest{
		Name:     "test-instance",
		Template: "python-ml",
	}

	hourlyRate := 0.0416
	services := []ctypes.Service{}
	primaryUsername := "ubuntu"

	testCases := []struct {
		name            string
		rootVolumeGB    int
		expectedStorage float64
	}{
		{
			name:            "Default root volume size",
			rootVolumeGB:    20,
			expectedStorage: 20.0,
		},
		{
			name:            "Custom root volume size",
			rootVolumeGB:    100,
			expectedStorage: 100.0,
		},
		{
			name:            "Large root volume size",
			rootVolumeGB:    500,
			expectedStorage: 500.0,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			instance := launcher.buildInstanceFromEC2(mockEC2Instance, req, hourlyRate, services, primaryUsername, tc.rootVolumeGB)

			if instance == nil {
				t.Fatal("buildInstanceFromEC2 returned nil")
			}

			if !floatEquals(instance.StorageGB, tc.expectedStorage) {
				t.Errorf("Expected StorageGB to be %.0f GB, got %.0f GB", tc.expectedStorage, instance.StorageGB)
			}

			t.Logf("✅ StorageGB correctly set to %.0f GB for %s", instance.StorageGB, tc.name)
		})
	}
}

// Helper function to create string pointers
func strPtr(s string) *string {
	return &s
}
