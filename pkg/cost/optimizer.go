package cost

import (
	"fmt"
	"sort"
	"time"

	"github.com/scttfrdmn/cloudworkstation/pkg/types"
)

// OptimizationType defines the type of cost optimization
type OptimizationType string

const (
	OptimizationTypeRightSize     OptimizationType = "right_size"
	OptimizationTypeSchedule      OptimizationType = "schedule"
	OptimizationTypeSpot          OptimizationType = "spot"
	OptimizationTypeReserved      OptimizationType = "reserved"
	OptimizationTypeHibernation   OptimizationType = "hibernation"
	OptimizationTypeStorage       OptimizationType = "storage"
	OptimizationTypeArchitecture  OptimizationType = "architecture"
)

// Recommendation represents a cost optimization recommendation
type Recommendation struct {
	ID               string            `json:"id"`
	Type             OptimizationType  `json:"type"`
	Priority         string            `json:"priority"` // high, medium, low
	Title            string            `json:"title"`
	Description      string            `json:"description"`
	EstimatedSavings float64           `json:"estimated_savings"` // Monthly savings in dollars
	SavingsPercent   float64           `json:"savings_percent"`
	Effort           string            `json:"effort"` // low, medium, high
	Risk             string            `json:"risk"`   // low, medium, high
	Implementation   string            `json:"implementation"`
	InstanceID       string            `json:"instance_id,omitempty"`
	ProjectID        string            `json:"project_id,omitempty"`
	Metrics          map[string]float64 `json:"metrics,omitempty"`
	CreatedAt        time.Time         `json:"created_at"`
	ExpiresAt        time.Time         `json:"expires_at"`
}

// CostOptimizer analyzes usage patterns and provides optimization recommendations
type CostOptimizer struct {
	recommendations map[string]*Recommendation
}

// NewCostOptimizer creates a new cost optimizer
func NewCostOptimizer() *CostOptimizer {
	return &CostOptimizer{
		recommendations: make(map[string]*Recommendation),
	}
}

// AnalyzeInstance analyzes an instance for optimization opportunities
func (co *CostOptimizer) AnalyzeInstance(instance *types.Instance) []*Recommendation {
	recommendations := make([]*Recommendation, 0)

	// Check for right-sizing opportunities
	if rec := co.checkRightSizing(instance); rec != nil {
		recommendations = append(recommendations, rec)
	}

	// Check for hibernation opportunities
	if rec := co.checkHibernationOpportunity(instance); rec != nil {
		recommendations = append(recommendations, rec)
	}

	// Check for spot instance opportunities
	if rec := co.checkSpotOpportunity(instance); rec != nil {
		recommendations = append(recommendations, rec)
	}

	// Check for architecture optimization (ARM)
	if rec := co.checkArchitectureOptimization(instance); rec != nil {
		recommendations = append(recommendations, rec)
	}

	// Check for scheduling opportunities
	if rec := co.checkSchedulingOpportunity(instance); rec != nil {
		recommendations = append(recommendations, rec)
	}

	return recommendations
}

// checkRightSizing checks if instance can be downsized
func (co *CostOptimizer) checkRightSizing(instance *types.Instance) *Recommendation {
	// Analyze CPU and memory utilization
	// This would use actual CloudWatch metrics in production
	avgCPU := 15.0    // Mock average CPU utilization
	avgMemory := 30.0 // Mock average memory utilization

	if avgCPU < 20 && avgMemory < 40 {
		currentCost := instance.EstimatedCost
		newCost := currentCost * 0.5 // Assume 50% cost reduction with smaller instance
		savings := currentCost - newCost

		return &Recommendation{
			ID:               fmt.Sprintf("rec-%s-rightsize", instance.ID),
			Type:             OptimizationTypeRightSize,
			Priority:         "high",
			Title:            "Right-size underutilized instance",
			Description:      fmt.Sprintf("Instance %s is using only %.1f%% CPU and %.1f%% memory on average", instance.Name, avgCPU, avgMemory),
			EstimatedSavings: savings * 30, // Monthly savings
			SavingsPercent:   50,
			Effort:           "low",
			Risk:             "low",
			Implementation:   fmt.Sprintf("Resize from %s to a smaller instance type", instance.InstanceType),
			InstanceID:       instance.ID,
			Metrics: map[string]float64{
				"avg_cpu":    avgCPU,
				"avg_memory": avgMemory,
			},
			CreatedAt: time.Now(),
			ExpiresAt: time.Now().Add(7 * 24 * time.Hour),
		}
	}

	return nil
}

// checkHibernationOpportunity checks for hibernation optimization
func (co *CostOptimizer) checkHibernationOpportunity(instance *types.Instance) *Recommendation {
	// Check if instance has idle periods
	idleHoursPerDay := 16.0 // Mock: instance idle 16 hours per day

	if idleHoursPerDay > 8 && !instance.IdlePolicyEnabled {
		dailyCost := instance.EstimatedCost
		savingsPerDay := dailyCost * (idleHoursPerDay / 24)
		monthlySavings := savingsPerDay * 30

		return &Recommendation{
			ID:               fmt.Sprintf("rec-%s-hibernate", instance.ID),
			Type:             OptimizationTypeHibernation,
			Priority:         "high",
			Title:            "Enable hibernation for idle periods",
			Description:      fmt.Sprintf("Instance %s is idle %.0f hours per day on average", instance.Name, idleHoursPerDay),
			EstimatedSavings: monthlySavings,
			SavingsPercent:   (idleHoursPerDay / 24) * 100,
			Effort:           "low",
			Risk:             "low",
			Implementation:   "Enable hibernation policy to automatically hibernate during idle periods",
			InstanceID:       instance.ID,
			Metrics: map[string]float64{
				"idle_hours_per_day": idleHoursPerDay,
			},
			CreatedAt: time.Now(),
			ExpiresAt: time.Now().Add(7 * 24 * time.Hour),
		}
	}

	return nil
}

// checkSpotOpportunity checks if spot instances would be beneficial
func (co *CostOptimizer) checkSpotOpportunity(instance *types.Instance) *Recommendation {
	// Check if workload is suitable for spot instances
	if instance.SpotEligible && !instance.IsSpot {
		spotDiscount := 0.7 // Assume 70% discount for spot instances
		monthlySavings := instance.EstimatedCost * 30 * spotDiscount

		return &Recommendation{
			ID:               fmt.Sprintf("rec-%s-spot", instance.ID),
			Type:             OptimizationTypeSpot,
			Priority:         "medium",
			Title:            "Use Spot instances for fault-tolerant workloads",
			Description:      fmt.Sprintf("Instance %s appears suitable for Spot pricing", instance.Name),
			EstimatedSavings: monthlySavings,
			SavingsPercent:   spotDiscount * 100,
			Effort:           "medium",
			Risk:             "medium",
			Implementation:   "Launch as Spot instance with appropriate interruption handling",
			InstanceID:       instance.ID,
			CreatedAt:        time.Now(),
			ExpiresAt:        time.Now().Add(7 * 24 * time.Hour),
		}
	}

	return nil
}

// checkArchitectureOptimization checks for ARM optimization opportunities
func (co *CostOptimizer) checkArchitectureOptimization(instance *types.Instance) *Recommendation {
	// Check if instance could benefit from ARM architecture
	if instance.Architecture == "x86_64" && instance.ARMCompatible {
		armDiscount := 0.2 // ARM instances typically 20% cheaper
		monthlySavings := instance.EstimatedCost * 30 * armDiscount

		return &Recommendation{
			ID:               fmt.Sprintf("rec-%s-arm", instance.ID),
			Type:             OptimizationTypeArchitecture,
			Priority:         "medium",
			Title:            "Migrate to ARM-based instances",
			Description:      fmt.Sprintf("Instance %s workload is compatible with ARM architecture", instance.Name),
			EstimatedSavings: monthlySavings,
			SavingsPercent:   armDiscount * 100,
			Effort:           "medium",
			Risk:             "low",
			Implementation:   "Migrate to Graviton (ARM) instance for better price-performance",
			InstanceID:       instance.ID,
			CreatedAt:        time.Now(),
			ExpiresAt:        time.Now().Add(7 * 24 * time.Hour),
		}
	}

	return nil
}

// checkSchedulingOpportunity checks for scheduling optimization
func (co *CostOptimizer) checkSchedulingOpportunity(instance *types.Instance) *Recommendation {
	// Check if instance runs 24/7 but only needed during work hours
	if instance.AlwaysOn && instance.WorkloadType == "development" {
		// Assume 40 work hours per week vs 168 total hours
		workHoursRatio := 40.0 / 168.0
		potentialSavings := instance.EstimatedCost * 30 * (1 - workHoursRatio)

		return &Recommendation{
			ID:               fmt.Sprintf("rec-%s-schedule", instance.ID),
			Type:             OptimizationTypeSchedule,
			Priority:         "high",
			Title:            "Implement workday scheduling",
			Description:      fmt.Sprintf("Instance %s runs 24/7 but may only be needed during work hours", instance.Name),
			EstimatedSavings: potentialSavings,
			SavingsPercent:   (1 - workHoursRatio) * 100,
			Effort:           "low",
			Risk:             "low",
			Implementation:   "Set up automatic start/stop schedule for work hours only",
			InstanceID:       instance.ID,
			CreatedAt:        time.Now(),
			ExpiresAt:        time.Now().Add(7 * 24 * time.Hour),
		}
	}

	return nil
}

// AnalyzeProject analyzes a project for optimization opportunities
func (co *CostOptimizer) AnalyzeProject(projectID string, instances []*types.Instance) []*Recommendation {
	recommendations := make([]*Recommendation, 0)

	// Analyze each instance
	for _, instance := range instances {
		instanceRecs := co.AnalyzeInstance(instance)
		recommendations = append(recommendations, instanceRecs...)
	}

	// Add project-level recommendations
	if rec := co.checkProjectReservedInstances(projectID, instances); rec != nil {
		recommendations = append(recommendations, rec)
	}

	if rec := co.checkProjectStorageOptimization(projectID, instances); rec != nil {
		recommendations = append(recommendations, rec)
	}

	// Sort by priority and potential savings
	co.sortRecommendations(recommendations)

	return recommendations
}

// checkProjectReservedInstances checks for Reserved Instance opportunities
func (co *CostOptimizer) checkProjectReservedInstances(projectID string, instances []*types.Instance) *Recommendation {
	// Count long-running instances
	longRunning := 0
	totalCost := 0.0

	for _, instance := range instances {
		if instance.Runtime > 720 { // Running for more than 30 days
			longRunning++
			totalCost += instance.EstimatedCost
		}
	}

	if longRunning >= 3 {
		riDiscount := 0.3 // Assume 30% discount with Reserved Instances
		monthlySavings := totalCost * 30 * riDiscount

		return &Recommendation{
			ID:               fmt.Sprintf("rec-%s-reserved", projectID),
			Type:             OptimizationTypeReserved,
			Priority:         "medium",
			Title:            "Purchase Reserved Instances for long-running workloads",
			Description:      fmt.Sprintf("Project has %d long-running instances that could benefit from Reserved Instance pricing", longRunning),
			EstimatedSavings: monthlySavings,
			SavingsPercent:   riDiscount * 100,
			Effort:           "low",
			Risk:             "low",
			Implementation:   "Purchase 1-year Reserved Instances for predictable workloads",
			ProjectID:        projectID,
			Metrics: map[string]float64{
				"long_running_instances": float64(longRunning),
				"total_monthly_cost":     totalCost * 30,
			},
			CreatedAt: time.Now(),
			ExpiresAt: time.Now().Add(30 * 24 * time.Hour),
		}
	}

	return nil
}

// checkProjectStorageOptimization checks for storage optimization
func (co *CostOptimizer) checkProjectStorageOptimization(projectID string, instances []*types.Instance) *Recommendation {
	// Check for unused or oversized storage
	totalStorageGB := 0.0
	usedStorageGB := 0.0

	for _, instance := range instances {
		totalStorageGB += instance.StorageGB
		usedStorageGB += instance.StorageUsedGB
	}

	utilizationPercent := (usedStorageGB / totalStorageGB) * 100

	if utilizationPercent < 50 && totalStorageGB > 100 {
		wastedStorage := totalStorageGB - usedStorageGB
		storageCoastPerGB := 0.10 // $0.10 per GB per month
		monthlySavings := wastedStorage * storageCoastPerGB

		return &Recommendation{
			ID:               fmt.Sprintf("rec-%s-storage", projectID),
			Type:             OptimizationTypeStorage,
			Priority:         "low",
			Title:            "Optimize storage allocation",
			Description:      fmt.Sprintf("Project is using only %.1f%% of allocated storage", utilizationPercent),
			EstimatedSavings: monthlySavings,
			SavingsPercent:   (wastedStorage / totalStorageGB) * 100,
			Effort:           "medium",
			Risk:             "low",
			Implementation:   "Reduce storage allocation or use lifecycle policies for old data",
			ProjectID:        projectID,
			Metrics: map[string]float64{
				"total_storage_gb":   totalStorageGB,
				"used_storage_gb":    usedStorageGB,
				"utilization_percent": utilizationPercent,
			},
			CreatedAt: time.Now(),
			ExpiresAt: time.Now().Add(30 * 24 * time.Hour),
		}
	}

	return nil
}

// sortRecommendations sorts recommendations by priority and savings
func (co *CostOptimizer) sortRecommendations(recommendations []*Recommendation) {
	sort.Slice(recommendations, func(i, j int) bool {
		// First sort by priority
		priorityOrder := map[string]int{"high": 3, "medium": 2, "low": 1}
		if priorityOrder[recommendations[i].Priority] != priorityOrder[recommendations[j].Priority] {
			return priorityOrder[recommendations[i].Priority] > priorityOrder[recommendations[j].Priority]
		}
		
		// Then by estimated savings
		return recommendations[i].EstimatedSavings > recommendations[j].EstimatedSavings
	})
}

// GetTopRecommendations returns the top N recommendations
func (co *CostOptimizer) GetTopRecommendations(recommendations []*Recommendation, n int) []*Recommendation {
	if len(recommendations) <= n {
		return recommendations
	}
	return recommendations[:n]
}

// CalculateTotalSavings calculates total potential savings
func (co *CostOptimizer) CalculateTotalSavings(recommendations []*Recommendation) float64 {
	total := 0.0
	for _, rec := range recommendations {
		total += rec.EstimatedSavings
	}
	return total
}

// GenerateOptimizationReport generates a comprehensive optimization report
func (co *CostOptimizer) GenerateOptimizationReport(projectID string, instances []*types.Instance) *OptimizationReport {
	recommendations := co.AnalyzeProject(projectID, instances)
	
	return &OptimizationReport{
		ProjectID:          projectID,
		GeneratedAt:        time.Now(),
		TotalInstances:     len(instances),
		Recommendations:    recommendations,
		TotalSavings:       co.CalculateTotalSavings(recommendations),
		TopRecommendations: co.GetTopRecommendations(recommendations, 5),
		Summary:            co.generateSummary(recommendations),
	}
}

// generateSummary generates a summary of optimization opportunities
func (co *CostOptimizer) generateSummary(recommendations []*Recommendation) map[string]interface{} {
	summary := map[string]interface{}{
		"total_recommendations": len(recommendations),
		"high_priority":         0,
		"medium_priority":       0,
		"low_priority":          0,
		"by_type":               make(map[string]int),
	}

	for _, rec := range recommendations {
		switch rec.Priority {
		case "high":
			summary["high_priority"] = summary["high_priority"].(int) + 1
		case "medium":
			summary["medium_priority"] = summary["medium_priority"].(int) + 1
		case "low":
			summary["low_priority"] = summary["low_priority"].(int) + 1
		}

		byType := summary["by_type"].(map[string]int)
		byType[string(rec.Type)]++
	}

	return summary
}

// OptimizationReport represents a complete optimization analysis
type OptimizationReport struct {
	ProjectID          string                 `json:"project_id"`
	GeneratedAt        time.Time              `json:"generated_at"`
	TotalInstances     int                    `json:"total_instances"`
	Recommendations    []*Recommendation      `json:"recommendations"`
	TotalSavings       float64                `json:"total_savings"`
	TopRecommendations []*Recommendation      `json:"top_recommendations"`
	Summary            map[string]interface{} `json:"summary"`
}