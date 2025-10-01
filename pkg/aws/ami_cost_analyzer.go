// Package aws provides AMI cost analysis functionality for the Universal AMI System
package aws

import (
	"time"

	"github.com/scttfrdmn/cloudworkstation/pkg/types"
)

// AMICostAnalyzer provides cost analysis for AMI vs script deployment strategies
type AMICostAnalyzer struct {
	// Pricing data (in production, this would come from AWS Pricing API)
	instancePricing   map[string]map[string]float64 // instance_type -> region -> hourly_cost
	storagePricing    map[string]float64             // region -> monthly_cost_per_gb
	transferPricing   map[string]float64             // region -> cost_per_gb
	marketplacePricing map[string]float64            // product_code -> hourly_cost

	// Cost calculation parameters
	averageAMISize     float64 // Average AMI size in GB
	setupTimeOverhead  float64 // Additional overhead for setup in minutes
}

// NewAMICostAnalyzer creates a new AMI cost analyzer with default pricing data
func NewAMICostAnalyzer() *AMICostAnalyzer {
	analyzer := &AMICostAnalyzer{
		instancePricing:    make(map[string]map[string]float64),
		storagePricing:     make(map[string]float64),
		transferPricing:    make(map[string]float64),
		marketplacePricing: make(map[string]float64),
		averageAMISize:     8.0,  // 8GB average AMI size
		setupTimeOverhead:  0.5,  // 30 seconds overhead
	}

	// Initialize with sample pricing data (production would load from AWS Pricing API)
	analyzer.initializePricingData()

	return analyzer
}

// CalculateAMICost calculates the cost of using an AMI for deployment
func (a *AMICostAnalyzer) CalculateAMICost(ami *types.AMIInfo, region string) float64 {
	cost := 0.0

	// AMI storage cost (monthly, prorated to hourly)
	storageCostPerHour := a.getStorageCost(region) * a.averageAMISize / (24 * 30)
	cost += storageCostPerHour

	// Marketplace cost if applicable
	if ami.MarketplaceCost > 0 {
		cost += ami.MarketplaceCost
	}

	return cost
}

// AnalyzeCosts performs comprehensive cost analysis for AMI vs script deployment
func (a *AMICostAnalyzer) AnalyzeCosts(templateName, region string, amiInfo *types.AMIInfo, scriptTime time.Duration) *types.AMICostAnalysis {
	analysis := &types.AMICostAnalysis{
		TemplateName: templateName,
		Region:       region,
	}

	// Base instance cost (same for both AMI and script)
	baseInstanceCost := a.getInstanceCost("t4g.medium", region) // Default instance type

	// AMI deployment costs
	analysis.AMILaunchCost = baseInstanceCost
	analysis.AMIStorageCost = a.getStorageCost(region) * a.averageAMISize // Monthly cost
	analysis.AMISetupCost = baseInstanceCost * (30.0 / 3600.0) // 30 seconds setup time

	// Script deployment costs
	analysis.ScriptLaunchCost = baseInstanceCost
	analysis.ScriptSetupCost = baseInstanceCost * (float64(scriptTime.Minutes()) / 60.0) // Setup time cost
	analysis.ScriptSetupTime = int(scriptTime.Minutes())

	// Calculate cost comparisons
	analysis.CostSavings1Hour = a.calculateSavings(1.0, analysis)
	analysis.CostSavings8Hour = a.calculateSavings(8.0, analysis)
	analysis.BreakEvenPoint = a.calculateBreakEvenPoint(analysis)
	analysis.TimeSavings = analysis.ScriptSetupTime // Time saved in minutes

	// Generate recommendation
	analysis.Recommendation, analysis.Reasoning = a.generateRecommendation(analysis)

	return analysis
}

// CompareDeploymentCosts compares costs between different deployment strategies
func (a *AMICostAnalyzer) CompareDeploymentCosts(strategies []string, duration float64, region string) map[string]float64 {
	costs := make(map[string]float64)

	baseInstanceCost := a.getInstanceCost("t4g.medium", region)

	for _, strategy := range strategies {
		switch strategy {
		case "ami":
			// AMI deployment cost
			storageCost := a.getStorageCost(region) * a.averageAMISize / (24 * 30) // Hourly storage cost
			setupCost := baseInstanceCost * (30.0 / 3600.0) // 30 seconds setup
			runtimeCost := baseInstanceCost * duration
			costs[strategy] = setupCost + runtimeCost + (storageCost * duration)

		case "script":
			// Script deployment cost
			setupCost := baseInstanceCost * (6.0 / 60.0) // 6 minutes average setup
			runtimeCost := baseInstanceCost * duration
			costs[strategy] = setupCost + runtimeCost

		case "marketplace":
			// Marketplace AMI cost
			marketplaceCost := a.marketplacePricing["default"] // Default marketplace cost
			setupCost := baseInstanceCost * (45.0 / 3600.0) // 45 seconds setup
			runtimeCost := (baseInstanceCost + marketplaceCost) * duration
			costs[strategy] = setupCost + runtimeCost
		}
	}

	return costs
}

// EstimateCrossRegionCost estimates the cost of copying an AMI across regions
func (a *AMICostAnalyzer) EstimateCrossRegionCost(amiSize float64, sourceRegion, targetRegion string) float64 {
	if amiSize == 0 {
		amiSize = a.averageAMISize
	}

	// Data transfer cost for cross-region copy
	transferCost := a.getTransferCost(sourceRegion) * amiSize

	// Additional storage cost in target region
	targetStorageCost := a.getStorageCost(targetRegion) * amiSize / (24 * 30) // Monthly cost prorated

	return transferCost + targetStorageCost
}

// GetOptimizedInstanceType recommends the most cost-effective instance type for a template
func (a *AMICostAnalyzer) GetOptimizedInstanceType(templateDomain string, region string) (string, float64) {
	// Instance type recommendations based on template domain
	recommendations := map[string][]string{
		"ml":   {"t4g.medium", "m6i.large", "c6i.large"},
		"data": {"r6i.large", "m6i.xlarge", "r5.large"},
		"web":  {"t4g.small", "t4g.medium", "t3.medium"},
		"hpc":  {"c6i.xlarge", "c5n.2xlarge", "m6i.2xlarge"},
		"default": {"t4g.medium", "t3.medium", "m6i.large"},
	}

	candidates := recommendations[templateDomain]
	if candidates == nil {
		candidates = recommendations["default"]
	}

	// Find the most cost-effective option
	bestInstance := candidates[0]
	bestCost := a.getInstanceCost(bestInstance, region)

	for _, instanceType := range candidates[1:] {
		cost := a.getInstanceCost(instanceType, region)
		if cost < bestCost {
			bestInstance = instanceType
			bestCost = cost
		}
	}

	return bestInstance, bestCost
}

// Private helper methods

func (a *AMICostAnalyzer) initializePricingData() {
	// Initialize US regions pricing
	usRegions := []string{"us-east-1", "us-east-2", "us-west-1", "us-west-2"}
	for _, region := range usRegions {
		a.storagePricing[region] = 0.10 // $0.10 per GB per month
		a.transferPricing[region] = 0.02 // $0.02 per GB transfer
	}

	// Initialize EU regions pricing (slightly higher)
	euRegions := []string{"eu-west-1", "eu-west-2", "eu-central-1"}
	for _, region := range euRegions {
		a.storagePricing[region] = 0.11 // $0.11 per GB per month
		a.transferPricing[region] = 0.02 // $0.02 per GB transfer
	}

	// Initialize APAC regions pricing
	apacRegions := []string{"ap-south-1", "ap-southeast-1", "ap-northeast-1"}
	for _, region := range apacRegions {
		a.storagePricing[region] = 0.12 // $0.12 per GB per month
		a.transferPricing[region] = 0.03 // $0.03 per GB transfer
	}

	// Initialize instance type pricing for us-east-1
	usEast1Pricing := map[string]float64{
		"t4g.small":   0.0168,
		"t4g.medium":  0.0336,
		"t4g.large":   0.0672,
		"t4g.xlarge":  0.1344,
		"t3.small":    0.0208,
		"t3.medium":   0.0416,
		"t3.large":    0.0832,
		"m6i.large":   0.0864,
		"m6i.xlarge":  0.1728,
		"c6i.large":   0.0765,
		"c6i.xlarge":  0.1530,
		"r6i.large":   0.1008,
		"r6i.xlarge":  0.2016,
	}
	a.instancePricing["us-east-1"] = usEast1Pricing

	// Copy pricing to other regions with small adjustments
	for _, region := range usRegions[1:] {
		a.instancePricing[region] = make(map[string]float64)
		for instanceType, cost := range usEast1Pricing {
			a.instancePricing[region][instanceType] = cost * 1.02 // 2% higher
		}
	}

	for _, region := range euRegions {
		a.instancePricing[region] = make(map[string]float64)
		for instanceType, cost := range usEast1Pricing {
			a.instancePricing[region][instanceType] = cost * 1.05 // 5% higher
		}
	}

	for _, region := range apacRegions {
		a.instancePricing[region] = make(map[string]float64)
		for instanceType, cost := range usEast1Pricing {
			a.instancePricing[region][instanceType] = cost * 1.08 // 8% higher
		}
	}

	// Default marketplace pricing
	a.marketplacePricing["default"] = 0.05 // $0.05 per hour average
}

func (a *AMICostAnalyzer) getInstanceCost(instanceType, region string) float64 {
	if regionPricing, exists := a.instancePricing[region]; exists {
		if cost, exists := regionPricing[instanceType]; exists {
			return cost
		}
	}
	return 0.05 // Default fallback cost
}

func (a *AMICostAnalyzer) getStorageCost(region string) float64 {
	if cost, exists := a.storagePricing[region]; exists {
		return cost
	}
	return 0.10 // Default fallback cost
}

func (a *AMICostAnalyzer) getTransferCost(region string) float64 {
	if cost, exists := a.transferPricing[region]; exists {
		return cost
	}
	return 0.02 // Default fallback cost
}

func (a *AMICostAnalyzer) calculateSavings(hours float64, analysis *types.AMICostAnalysis) float64 {
	amiTotalCost := analysis.AMISetupCost + (analysis.AMILaunchCost * hours)
	scriptTotalCost := analysis.ScriptSetupCost + (analysis.ScriptLaunchCost * hours)
	return scriptTotalCost - amiTotalCost
}

func (a *AMICostAnalyzer) calculateBreakEvenPoint(analysis *types.AMICostAnalysis) float64 {
	// Calculate when AMI storage cost equals script setup savings
	setupSavings := analysis.ScriptSetupCost - analysis.AMISetupCost
	storageMonthly := analysis.AMIStorageCost

	if storageMonthly <= 0 {
		return 0 // Always beneficial if no storage cost
	}

	// Break-even point in hours
	return setupSavings / (storageMonthly / (24 * 30))
}

func (a *AMICostAnalyzer) generateRecommendation(analysis *types.AMICostAnalysis) (string, string) {
	// Decision logic for AMI vs script recommendation
	if analysis.CostSavings1Hour > 0.01 { // Save more than 1 cent per hour
		if analysis.TimeSavings > 3 { // Save more than 3 minutes
			return "ami_recommended", "AMI provides significant time and cost savings"
		}
		return "ami_recommended", "AMI provides cost savings and faster deployment"
	}

	if analysis.CostSavings8Hour < -0.05 { // Costs more than 5 cents for 8-hour session
		return "script_recommended", "Script provisioning is more cost-effective for longer sessions"
	}

	if analysis.TimeSavings > 5 { // Save more than 5 minutes
		return "ami_recommended", "AMI provides significant time savings worth the small cost increase"
	}

	return "neutral", "Both AMI and script provisioning have similar cost/benefit profiles"
}