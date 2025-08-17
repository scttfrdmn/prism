package project

import (
	"strings"
	"time"

	"github.com/scttfrdmn/cloudworkstation/pkg/types"
)

// CostCalculator calculates AWS costs for instances and storage
type CostCalculator struct {
	// AWS pricing data - in a production system, this would be loaded from AWS Pricing API
	// For now, we use estimated rates based on common instance types and regions
}

// Instance pricing data (USD per hour) - estimated rates for us-east-1
var instancePricing = map[string]float64{
	// General Purpose
	"t3.micro":    0.0104,
	"t3.small":    0.0208,
	"t3.medium":   0.0416,
	"t3.large":    0.0832,
	"t3.xlarge":   0.1664,
	"t3.2xlarge":  0.3328,
	"t3a.micro":   0.0094,
	"t3a.small":   0.0188,
	"t3a.medium":  0.0376,
	"t3a.large":   0.0752,
	"t3a.xlarge":  0.1504,
	"t3a.2xlarge": 0.3008,

	// Compute Optimized
	"c5.large":    0.085,
	"c5.xlarge":   0.17,
	"c5.2xlarge":  0.34,
	"c5.4xlarge":  0.68,
	"c5.9xlarge":  1.53,
	"c5.12xlarge": 2.04,
	"c5.18xlarge": 3.06,
	"c5.24xlarge": 4.08,

	// Memory Optimized
	"r5.large":    0.126,
	"r5.xlarge":   0.252,
	"r5.2xlarge":  0.504,
	"r5.4xlarge":  1.008,
	"r5.8xlarge":  2.016,
	"r5.12xlarge": 3.024,
	"r5.16xlarge": 4.032,
	"r5.24xlarge": 6.048,

	// GPU Instances
	"g4dn.xlarge":   0.526,
	"g4dn.2xlarge":  0.752,
	"g4dn.4xlarge":  1.204,
	"g4dn.8xlarge":  2.176,
	"g4dn.12xlarge": 3.912,
	"g4dn.16xlarge": 4.352,
	"p3.2xlarge":    3.06,
	"p3.8xlarge":    12.24,
	"p3.16xlarge":   24.48,
	"p4d.24xlarge":  32.77,
}

// Storage pricing (USD per GB per month)
var storagePricing = map[string]float64{
	"gp3":          0.08,   // General Purpose SSD (gp3)
	"gp2":          0.10,   // General Purpose SSD (gp2)
	"io1":          0.125,  // Provisioned IOPS SSD (io1)
	"io2":          0.125,  // Provisioned IOPS SSD (io2)
	"st1":          0.045,  // Throughput Optimized HDD
	"sc1":          0.025,  // Cold HDD
	"standard":     0.05,   // Magnetic
	"efs-standard": 0.30,   // EFS Standard
	"efs-ia":       0.0125, // EFS Infrequent Access
}

// CalculateInstanceCosts calculates costs for a list of instances
func (c *CostCalculator) CalculateInstanceCosts(instances []types.Instance) ([]types.InstanceCost, float64) {
	var instanceCosts []types.InstanceCost
	var totalCost float64

	for _, instance := range instances {
		cost := c.calculateSingleInstanceCost(instance)
		instanceCosts = append(instanceCosts, cost)
		totalCost += cost.TotalCost
	}

	return instanceCosts, totalCost
}

// CalculateStorageCosts calculates costs for EFS and EBS volumes
func (c *CostCalculator) CalculateStorageCosts(efsVolumes []types.EFSVolume, ebsVolumes []types.EBSVolume) ([]types.StorageCost, float64) {
	var storageCosts []types.StorageCost
	var totalCost float64

	// Calculate EFS costs
	for _, volume := range efsVolumes {
		cost := c.calculateEFSCost(volume)
		storageCosts = append(storageCosts, cost)
		totalCost += cost.Cost
	}

	// Calculate EBS costs
	for _, volume := range ebsVolumes {
		cost := c.calculateEBSCost(volume)
		storageCosts = append(storageCosts, cost)
		totalCost += cost.Cost
	}

	return storageCosts, totalCost
}

// calculateSingleInstanceCost calculates the cost for a single instance
func (c *CostCalculator) calculateSingleInstanceCost(instance types.Instance) types.InstanceCost {
	hourlyRate, exists := instancePricing[instance.InstanceType]
	if !exists {
		// Use a default rate for unknown instance types
		hourlyRate = c.estimateInstanceCost(instance.InstanceType)
	}

	// Calculate hours in different states
	now := time.Now()
	totalRuntime := now.Sub(instance.LaunchTime)
	totalHours := totalRuntime.Hours()

	// For simplicity, assume the instance has been running the entire time
	// In a real implementation, we would track state changes
	runningHours := totalHours
	hibernatedHours := 0.0
	stoppedHours := 0.0

	// Adjust based on current state
	switch strings.ToLower(instance.State) {
	case "stopped":
		// If stopped, it might have been running part of the time
		runningHours = totalHours * 0.7 // Estimate 70% uptime
		stoppedHours = totalHours * 0.3
	case "hibernated":
		runningHours = totalHours * 0.5 // Estimate 50% uptime before hibernation
		hibernatedHours = totalHours * 0.5
	}

	// Calculate compute cost (only for running hours)
	computeCost := runningHours * hourlyRate

	// Calculate storage cost (EBS root volume)
	storageCost := c.calculateInstanceStorageCost(instance)

	totalCost := computeCost + storageCost

	return types.InstanceCost{
		InstanceName:    instance.Name,
		InstanceType:    instance.InstanceType,
		ComputeCost:     computeCost,
		StorageCost:     storageCost,
		TotalCost:       totalCost,
		RunningHours:    runningHours,
		HibernatedHours: hibernatedHours,
		StoppedHours:    stoppedHours,
	}
}

// calculateInstanceStorageCost calculates the EBS storage cost for an instance
func (c *CostCalculator) calculateInstanceStorageCost(instance types.Instance) float64 {
	// Estimate root volume size based on instance type
	rootVolumeSize := c.estimateRootVolumeSize(instance.InstanceType)

	// Use gp3 pricing as default for root volumes
	pricePerGB := storagePricing["gp3"]

	// Calculate monthly cost, then pro-rate for actual runtime
	monthlyStorageCost := rootVolumeSize * pricePerGB

	// Calculate days since launch
	daysSinceLaunch := time.Since(instance.LaunchTime).Hours() / 24

	// Pro-rate the monthly cost
	return monthlyStorageCost * (daysSinceLaunch / 30.0)
}

// calculateEFSCost calculates the cost for an EFS volume
func (c *CostCalculator) calculateEFSCost(volume types.EFSVolume) types.StorageCost {
	pricePerGB := storagePricing["efs-standard"]

	// EFS size is not directly available in our volume type
	// In a real implementation, we would query AWS for actual usage
	estimatedSizeGB := 10.0 // Default estimate

	// Calculate monthly cost, then pro-rate for actual time
	monthlyStorageCost := estimatedSizeGB * pricePerGB
	daysSinceCreation := time.Since(volume.CreationTime).Hours() / 24
	actualCost := monthlyStorageCost * (daysSinceCreation / 30.0)

	return types.StorageCost{
		VolumeName: volume.Name,
		VolumeType: "EFS",
		SizeGB:     estimatedSizeGB,
		Cost:       actualCost,
		CostPerGB:  pricePerGB,
	}
}

// calculateEBSCost calculates the cost for an EBS volume
func (c *CostCalculator) calculateEBSCost(volume types.EBSVolume) types.StorageCost {
	pricePerGB := storagePricing["gp3"] // Default to gp3 pricing

	// Use volume type specific pricing if available
	if price, exists := storagePricing[volume.VolumeType]; exists {
		pricePerGB = price
	}

	// Calculate monthly cost, then pro-rate for actual time
	monthlyStorageCost := float64(volume.SizeGB) * pricePerGB
	daysSinceCreation := time.Since(volume.CreationTime).Hours() / 24
	actualCost := monthlyStorageCost * (daysSinceCreation / 30.0)

	return types.StorageCost{
		VolumeName: volume.Name,
		VolumeType: volume.VolumeType,
		SizeGB:     float64(volume.SizeGB),
		Cost:       actualCost,
		CostPerGB:  pricePerGB,
	}
}

// estimateInstanceCost estimates the hourly cost for unknown instance types
func (c *CostCalculator) estimateInstanceCost(instanceType string) float64 {
	// Extract instance family and size
	parts := strings.Split(instanceType, ".")
	if len(parts) != 2 {
		return 0.10 // Default fallback rate
	}

	family := parts[0]
	size := parts[1]

	// Base rates by instance family
	familyRates := map[string]float64{
		"t3":   0.0104, // t3.micro base rate
		"t3a":  0.0094, // t3a.micro base rate
		"c5":   0.085,  // c5.large base rate
		"c5n":  0.108,  // c5n.large base rate
		"r5":   0.126,  // r5.large base rate
		"r5a":  0.113,  // r5a.large base rate
		"m5":   0.096,  // m5.large base rate
		"m5a":  0.086,  // m5a.large base rate
		"g4dn": 0.526,  // g4dn.xlarge base rate
		"p3":   3.06,   // p3.2xlarge base rate
		"p4d":  32.77,  // p4d.24xlarge base rate
	}

	baseRate, exists := familyRates[family]
	if !exists {
		baseRate = 0.10 // Default rate
	}

	// Size multipliers
	sizeMultipliers := map[string]float64{
		"nano":     0.25,
		"micro":    0.5,
		"small":    1.0,
		"medium":   2.0,
		"large":    4.0,
		"xlarge":   8.0,
		"2xlarge":  16.0,
		"3xlarge":  24.0,
		"4xlarge":  32.0,
		"6xlarge":  48.0,
		"8xlarge":  64.0,
		"9xlarge":  72.0,
		"12xlarge": 96.0,
		"16xlarge": 128.0,
		"18xlarge": 144.0,
		"24xlarge": 192.0,
		"32xlarge": 256.0,
	}

	multiplier, exists := sizeMultipliers[size]
	if !exists {
		multiplier = 4.0 // Default to large equivalent
	}

	return baseRate * multiplier
}

// estimateRootVolumeSize estimates the root EBS volume size for an instance type
func (c *CostCalculator) estimateRootVolumeSize(instanceType string) float64 {
	// Most instances have 8-20 GB root volumes
	// GPU instances typically have larger root volumes
	if strings.Contains(instanceType, "g4") || strings.Contains(instanceType, "p3") || strings.Contains(instanceType, "p4") {
		return 50.0 // GPU instances often need larger root volumes for drivers
	}

	return 20.0 // Standard root volume size
}

// GetInstanceHourlyRate returns the hourly rate for an instance type
func (c *CostCalculator) GetInstanceHourlyRate(instanceType string) float64 {
	if rate, exists := instancePricing[instanceType]; exists {
		return rate
	}
	return c.estimateInstanceCost(instanceType)
}

// GetStorageMonthlyRate returns the monthly rate per GB for a storage type
func (c *CostCalculator) GetStorageMonthlyRate(storageType string) float64 {
	if rate, exists := storagePricing[storageType]; exists {
		return rate
	}
	return storagePricing["gp3"] // Default to gp3 pricing
}

// EstimateMonthlyCost estimates the monthly cost for running an instance continuously
func (c *CostCalculator) EstimateMonthlyCost(instanceType string, storageGB int) float64 {
	hourlyRate := c.GetInstanceHourlyRate(instanceType)
	storageRate := c.GetStorageMonthlyRate("gp3")

	// 24 hours * 30 days = 720 hours per month
	monthlyComputeCost := hourlyRate * 720
	monthlyStorageCost := float64(storageGB) * storageRate

	return monthlyComputeCost + monthlyStorageCost
}

// EstimateHibernationSavings estimates the cost savings from hibernating vs running
func (c *CostCalculator) EstimateHibernationSavings(instanceType string, hibernatedHours float64) float64 {
	hourlyRate := c.GetInstanceHourlyRate(instanceType)

	// Hibernation saves compute costs but storage costs continue
	// Assume hibernation saves 90% of compute costs (some overhead remains)
	return hourlyRate * hibernatedHours * 0.90
}
