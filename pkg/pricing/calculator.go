package pricing

import (
	"fmt"
	"strings"
)

// Calculator applies institutional pricing discounts to AWS list prices
type Calculator struct {
	config *InstitutionalPricingConfig
}

// NewCalculator creates a new pricing calculator with institutional discounts
func NewCalculator(config *InstitutionalPricingConfig) *Calculator {
	return &Calculator{config: config}
}

// InstanceCostResult represents the result of instance cost calculation
type InstanceCostResult struct {
	ListPrice       float64 `json:"list_price"`       // AWS list price per hour
	DiscountedPrice float64 `json:"discounted_price"` // Price after discounts
	TotalDiscount   float64 `json:"total_discount"`   // Total discount percentage
	DailyEstimate   float64 `json:"daily_estimate"`   // Estimated daily cost
	MonthlyEstimate float64 `json:"monthly_estimate"` // Estimated monthly cost
	AppliedDiscounts []DiscountApplied `json:"applied_discounts"` // List of applied discounts
}

// DiscountApplied represents a specific discount that was applied
type DiscountApplied struct {
	Type        string  `json:"type"`        // Type of discount (e.g., "global_ec2", "instance_family")
	Description string  `json:"description"` // Human-readable description
	Percentage  float64 `json:"percentage"`  // Discount percentage
	Savings     float64 `json:"savings"`     // Absolute savings per hour
}

// CalculateInstanceCost calculates the discounted cost for an EC2 instance
func (c *Calculator) CalculateInstanceCost(instanceType string, listPricePerHour float64, region string) *InstanceCostResult {
	if c.config == nil {
		// No discounts available
		return &InstanceCostResult{
			ListPrice:       listPricePerHour,
			DiscountedPrice: listPricePerHour,
			TotalDiscount:   0.0,
			DailyEstimate:   listPricePerHour * 24,
			MonthlyEstimate: listPricePerHour * 24 * 30,
		}
	}

	discountedPrice := listPricePerHour
	appliedDiscounts := []DiscountApplied{}

	// Apply global EC2 discount
	if c.config.GlobalDiscounts.EC2Discount > 0 {
		discount := c.config.GlobalDiscounts.EC2Discount
		savings := discountedPrice * discount
		discountedPrice = discountedPrice * (1 - discount)
		
		appliedDiscounts = append(appliedDiscounts, DiscountApplied{
			Type:        "global_ec2",
			Description: fmt.Sprintf("Global EC2 discount (%s)", c.config.Institution),
			Percentage:  discount,
			Savings:     savings,
		})
	}

	// Apply instance family discount
	instanceFamily := extractInstanceFamily(instanceType)
	if familyDiscount, exists := c.config.InstanceFamilyDiscounts[instanceFamily]; exists && familyDiscount > 0 {
		savings := discountedPrice * familyDiscount
		discountedPrice = discountedPrice * (1 - familyDiscount)
		
		appliedDiscounts = append(appliedDiscounts, DiscountApplied{
			Type:        "instance_family",
			Description: fmt.Sprintf("%s instance family discount", instanceFamily),
			Percentage:  familyDiscount,
			Savings:     savings,
		})
	}

	// Apply educational discount if configured
	if c.config.Programs.EducationalDiscount > 0 {
		discount := c.config.Programs.EducationalDiscount
		savings := discountedPrice * discount
		discountedPrice = discountedPrice * (1 - discount)
		
		appliedDiscounts = append(appliedDiscounts, DiscountApplied{
			Type:        "educational",
			Description: "Educational institution discount",
			Percentage:  discount,
			Savings:     savings,
		})
	}

	// Apply EDP (Enterprise Discount Program) discount
	if c.config.Enterprise.EDPDiscount > 0 {
		discount := c.config.Enterprise.EDPDiscount
		savings := discountedPrice * discount
		discountedPrice = discountedPrice * (1 - discount)
		
		appliedDiscounts = append(appliedDiscounts, DiscountApplied{
			Type:        "enterprise_edp",
			Description: "Enterprise Discount Program",
			Percentage:  discount,
			Savings:     savings,
		})
	}

	// Apply regional discount if configured
	if regionalConfig, exists := c.config.RegionalDiscounts[region]; exists && regionalConfig.AdditionalDiscount > 0 {
		discount := regionalConfig.AdditionalDiscount
		savings := discountedPrice * discount
		discountedPrice = discountedPrice * (1 - discount)
		
		appliedDiscounts = append(appliedDiscounts, DiscountApplied{
			Type:        "regional",
			Description: fmt.Sprintf("Regional discount for %s", region),
			Percentage:  discount,
			Savings:     savings,
		})
	}

	// Apply Reserved Instance discount modeling
	if c.config.CommitmentPrograms.ReservedInstanceCoverage > 0 {
		// Model RI savings (typically 30-75% depending on commitment)
		riDiscount := 0.40 * c.config.CommitmentPrograms.ReservedInstanceCoverage // Assume 40% RI discount
		savings := discountedPrice * riDiscount
		discountedPrice = discountedPrice * (1 - riDiscount)
		
		appliedDiscounts = append(appliedDiscounts, DiscountApplied{
			Type:        "reserved_instance",
			Description: fmt.Sprintf("Reserved Instance modeling (%.0f%% coverage)", c.config.CommitmentPrograms.ReservedInstanceCoverage*100),
			Percentage:  riDiscount,
			Savings:     savings,
		})
	}

	// Apply Savings Plan discount modeling
	if c.config.CommitmentPrograms.SavingsPlanCoverage > 0 {
		// Model Savings Plan additional savings
		spDiscount := 0.15 * c.config.CommitmentPrograms.SavingsPlanCoverage // Additional 15% from Savings Plans
		savings := discountedPrice * spDiscount
		discountedPrice = discountedPrice * (1 - spDiscount)
		
		appliedDiscounts = append(appliedDiscounts, DiscountApplied{
			Type:        "savings_plan",
			Description: fmt.Sprintf("Savings Plan modeling (%.0f%% coverage)", c.config.CommitmentPrograms.SavingsPlanCoverage*100),
			Percentage:  spDiscount,
			Savings:     savings,
		})
	}

	// Calculate total discount percentage
	totalDiscount := 0.0
	if listPricePerHour > 0 {
		totalDiscount = (listPricePerHour - discountedPrice) / listPricePerHour
	}

	return &InstanceCostResult{
		ListPrice:        listPricePerHour,
		DiscountedPrice:  discountedPrice,
		TotalDiscount:    totalDiscount,
		DailyEstimate:    discountedPrice * 24,
		MonthlyEstimate:  discountedPrice * 24 * 30,
		AppliedDiscounts: appliedDiscounts,
	}
}

// CalculateStorageCost calculates discounted storage costs
func (c *Calculator) CalculateStorageCost(storageType string, sizeGB int, pricePerGBMonth float64, region string) *InstanceCostResult {
	if c.config == nil {
		monthlyListPrice := float64(sizeGB) * pricePerGBMonth
		return &InstanceCostResult{
			ListPrice:       monthlyListPrice,
			DiscountedPrice: monthlyListPrice,
			TotalDiscount:   0.0,
			MonthlyEstimate: monthlyListPrice,
		}
	}

	monthlyListPrice := float64(sizeGB) * pricePerGBMonth
	discountedPrice := monthlyListPrice
	appliedDiscounts := []DiscountApplied{}

	// Apply storage-specific discounts
	var discount float64
	var discountType, description string

	switch storageType {
	case "ebs":
		discount = c.config.GlobalDiscounts.EBSDiscount
		discountType = "global_ebs"
		description = "Global EBS storage discount"
	case "efs":
		discount = c.config.GlobalDiscounts.EFSDiscount
		discountType = "global_efs"
		description = "Global EFS storage discount"
	default:
		discount = c.config.GlobalDiscounts.GeneralDiscount
		discountType = "general"
		description = "General storage discount"
	}

	if discount > 0 {
		savings := discountedPrice * discount
		discountedPrice = discountedPrice * (1 - discount)
		
		appliedDiscounts = append(appliedDiscounts, DiscountApplied{
			Type:        discountType,
			Description: description,
			Percentage:  discount,
			Savings:     savings,
		})
	}

	// Apply volume discount for large storage
	if c.config.Enterprise.VolumeDiscount > 0 && sizeGB > 1000 { // Apply to storage > 1TB
		volumeDiscount := c.config.Enterprise.VolumeDiscount
		savings := discountedPrice * volumeDiscount
		discountedPrice = discountedPrice * (1 - volumeDiscount)
		
		appliedDiscounts = append(appliedDiscounts, DiscountApplied{
			Type:        "volume",
			Description: "Volume discount for large storage",
			Percentage:  volumeDiscount,
			Savings:     savings,
		})
	}

	// Calculate total discount
	totalDiscount := 0.0
	if monthlyListPrice > 0 {
		totalDiscount = (monthlyListPrice - discountedPrice) / monthlyListPrice
	}

	return &InstanceCostResult{
		ListPrice:        monthlyListPrice,
		DiscountedPrice:  discountedPrice,
		TotalDiscount:    totalDiscount,
		MonthlyEstimate:  discountedPrice,
		AppliedDiscounts: appliedDiscounts,
	}
}

// GetSpotInstanceDiscount returns the effective spot instance discount
func (c *Calculator) GetSpotInstanceDiscount() float64 {
	if c.config == nil {
		return 0.70 // Default spot discount (70%)
	}
	
	// Use preference as a proxy for expected spot savings
	preference := c.config.CommitmentPrograms.SpotInstancePreference
	if preference > 0 {
		return 0.70 * preference // Scale spot discount by preference
	}
	
	return 0.70 // Default spot discount
}

// extractInstanceFamily extracts the instance family from instance type
// e.g., "c5.large" -> "c5", "m5a.xlarge" -> "m5a"
func extractInstanceFamily(instanceType string) string {
	parts := strings.Split(instanceType, ".")
	if len(parts) > 0 {
		return parts[0]
	}
	return instanceType
}

// GetPricingInfo returns detailed pricing information for display
func (c *Calculator) GetPricingInfo() map[string]interface{} {
	if c.config == nil {
		return map[string]interface{}{
			"institution": "None",
			"discounts_available": false,
			"pricing_model": "list_price",
		}
	}

	info := map[string]interface{}{
		"institution": c.config.Institution,
		"discounts_available": true,
		"version": c.config.Version,
		"contact": c.config.Contact,
		"valid_until": c.config.ValidUntil,
	}

	// Add discount summary
	if c.config.GlobalDiscounts.EC2Discount > 0 {
		info["ec2_discount"] = fmt.Sprintf("%.1f%%", c.config.GlobalDiscounts.EC2Discount*100)
	}
	if c.config.Programs.EducationalDiscount > 0 {
		info["educational_discount"] = fmt.Sprintf("%.1f%%", c.config.Programs.EducationalDiscount*100)
	}
	if c.config.Enterprise.EDPDiscount > 0 {
		info["enterprise_discount"] = fmt.Sprintf("%.1f%%", c.config.Enterprise.EDPDiscount*100)
	}

	return info
}