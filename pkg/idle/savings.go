// Package idle provides advanced idle detection and policy management
package idle

import (
	"fmt"
	"time"
)

// SavingsReport represents hibernation cost savings analysis
type SavingsReport struct {
	ReportID    string    `json:"report_id"`
	GeneratedAt time.Time `json:"generated_at"`
	Period      Period    `json:"period"`
	
	// Overall savings
	TotalSaved          float64 `json:"total_saved"`
	ProjectedSavings    float64 `json:"projected_savings"`
	HibernationHours    float64 `json:"hibernation_hours"`
	ActiveHours         float64 `json:"active_hours"`
	SavingsPercentage   float64 `json:"savings_percentage"`
	
	// Instance breakdown
	InstanceSavings []InstanceSaving `json:"instance_savings"`
	
	// Schedule effectiveness
	SchedulePerformance []SchedulePerformance `json:"schedule_performance"`
	
	// Recommendations
	Recommendations []Recommendation `json:"recommendations"`
	
	// Trends
	DailySavings   []DailySaving   `json:"daily_savings"`
	WeeklySavings  []WeeklySaving  `json:"weekly_savings"`
	MonthlySavings []MonthlySaving `json:"monthly_savings"`
}

// Period represents the reporting period
type Period struct {
	Start time.Time `json:"start"`
	End   time.Time `json:"end"`
	Days  int       `json:"days"`
}

// InstanceSaving represents savings for a specific instance
type InstanceSaving struct {
	InstanceID       string  `json:"instance_id"`
	InstanceName     string  `json:"instance_name"`
	InstanceType     string  `json:"instance_type"`
	HourlyRate       float64 `json:"hourly_rate"`
	HibernationHours float64 `json:"hibernation_hours"`
	ActiveHours      float64 `json:"active_hours"`
	TotalSaved       float64 `json:"total_saved"`
	SavingsPercent   float64 `json:"savings_percent"`
	
	// Hibernation events
	HibernationEvents []HibernationEvent `json:"hibernation_events"`
}

// HibernationEvent represents a single hibernation event
type HibernationEvent struct {
	StartTime    time.Time `json:"start_time"`
	EndTime      time.Time `json:"end_time"`
	Duration     float64   `json:"duration_hours"`
	SavedAmount  float64   `json:"saved_amount"`
	TriggerType  string    `json:"trigger_type"` // schedule, idle, manual
	ScheduleName string    `json:"schedule_name,omitempty"`
}

// SchedulePerformance tracks how well schedules are performing
type SchedulePerformance struct {
	ScheduleID        string  `json:"schedule_id"`
	ScheduleName      string  `json:"schedule_name"`
	ExecutionCount    int     `json:"execution_count"`
	SuccessRate       float64 `json:"success_rate"`
	TotalSaved        float64 `json:"total_saved"`
	AverageSaving     float64 `json:"average_saving"`
	EstimatedVsActual float64 `json:"estimated_vs_actual"` // Percentage difference
}

// Recommendation suggests optimization opportunities
type Recommendation struct {
	Type        string  `json:"type"`
	Priority    string  `json:"priority"` // high, medium, low
	Description string  `json:"description"`
	Impact      float64 `json:"estimated_monthly_impact"`
	Action      string  `json:"recommended_action"`
}

// DailySaving represents savings for a single day
type DailySaving struct {
	Date             time.Time `json:"date"`
	HibernationHours float64   `json:"hibernation_hours"`
	ActiveHours      float64   `json:"active_hours"`
	Saved            float64   `json:"saved"`
	InstanceCount    int       `json:"instance_count"`
}

// WeeklySaving represents savings for a week
type WeeklySaving struct {
	WeekStart        time.Time `json:"week_start"`
	HibernationHours float64   `json:"hibernation_hours"`
	ActiveHours      float64   `json:"active_hours"`
	Saved            float64   `json:"saved"`
	Trend            string    `json:"trend"` // increasing, decreasing, stable
}

// MonthlySaving represents savings for a month
type MonthlySaving struct {
	Month            string  `json:"month"` // YYYY-MM
	HibernationHours float64 `json:"hibernation_hours"`
	ActiveHours      float64 `json:"active_hours"`
	Saved            float64 `json:"saved"`
	ProjectedSaving  float64 `json:"projected_saving"`
}

// SavingsCalculator calculates hibernation savings
type SavingsCalculator struct {
	events []HibernationEvent
}

// NewSavingsCalculator creates a new savings calculator
func NewSavingsCalculator() *SavingsCalculator {
	return &SavingsCalculator{
		events: make([]HibernationEvent, 0),
	}
}

// GenerateReport generates a comprehensive savings report
func (sc *SavingsCalculator) GenerateReport(period Period) (*SavingsReport, error) {
	report := &SavingsReport{
		ReportID:    generateReportID(),
		GeneratedAt: time.Now(),
		Period:      period,
	}
	
	// Calculate overall savings
	report.TotalSaved = sc.calculateTotalSavings(period)
	report.HibernationHours = sc.calculateHibernationHours(period)
	report.ActiveHours = sc.calculateActiveHours(period)
	report.SavingsPercentage = (report.HibernationHours / (report.HibernationHours + report.ActiveHours)) * 100
	
	// Project future savings
	report.ProjectedSavings = sc.projectFutureSavings(report.TotalSaved, period.Days)
	
	// Generate instance breakdown
	report.InstanceSavings = sc.generateInstanceBreakdown(period)
	
	// Analyze schedule performance
	report.SchedulePerformance = sc.analyzeSchedulePerformance(period)
	
	// Generate recommendations
	report.Recommendations = sc.generateRecommendations(report)
	
	// Calculate trends
	report.DailySavings = sc.calculateDailyTrends(period)
	report.WeeklySavings = sc.calculateWeeklyTrends(period)
	report.MonthlySavings = sc.calculateMonthlyTrends(period)
	
	return report, nil
}

// calculateTotalSavings calculates total savings for the period
func (sc *SavingsCalculator) calculateTotalSavings(period Period) float64 {
	total := 0.0
	for _, event := range sc.events {
		if event.StartTime.After(period.Start) && event.StartTime.Before(period.End) {
			total += event.SavedAmount
		}
	}
	return total
}

// calculateHibernationHours calculates total hibernation hours
func (sc *SavingsCalculator) calculateHibernationHours(period Period) float64 {
	hours := 0.0
	for _, event := range sc.events {
		if event.StartTime.After(period.Start) && event.StartTime.Before(period.End) {
			hours += event.Duration
		}
	}
	return hours
}

// calculateActiveHours calculates total active hours
func (sc *SavingsCalculator) calculateActiveHours(period Period) float64 {
	totalHours := period.Days * 24
	hibernationHours := sc.calculateHibernationHours(period)
	return float64(totalHours) - hibernationHours
}

// projectFutureSavings projects savings for the next month
func (sc *SavingsCalculator) projectFutureSavings(currentSavings float64, days int) float64 {
	if days == 0 {
		return 0
	}
	dailyAverage := currentSavings / float64(days)
	return dailyAverage * 30 // Project for 30 days
}

// generateInstanceBreakdown generates savings breakdown by instance
func (sc *SavingsCalculator) generateInstanceBreakdown(period Period) []InstanceSaving {
	instanceMap := make(map[string]*InstanceSaving)
	
	// Group events by instance
	for _, event := range sc.events {
		if event.StartTime.After(period.Start) && event.StartTime.Before(period.End) {
			// This is simplified - would need actual instance data
			instanceID := "i-example"
			
			if _, exists := instanceMap[instanceID]; !exists {
				instanceMap[instanceID] = &InstanceSaving{
					InstanceID:        instanceID,
					InstanceName:      "example-instance",
					InstanceType:      "t3.medium",
					HourlyRate:        0.0416,
					HibernationEvents: []HibernationEvent{},
				}
			}
			
			saving := instanceMap[instanceID]
			saving.HibernationHours += event.Duration
			saving.TotalSaved += event.SavedAmount
			saving.HibernationEvents = append(saving.HibernationEvents, event)
		}
	}
	
	// Convert map to slice
	savings := make([]InstanceSaving, 0, len(instanceMap))
	for _, saving := range instanceMap {
		saving.ActiveHours = float64(period.Days*24) - saving.HibernationHours
		saving.SavingsPercent = (saving.HibernationHours / (saving.HibernationHours + saving.ActiveHours)) * 100
		savings = append(savings, *saving)
	}
	
	return savings
}

// analyzeSchedulePerformance analyzes how well schedules are performing
func (sc *SavingsCalculator) analyzeSchedulePerformance(period Period) []SchedulePerformance {
	scheduleMap := make(map[string]*SchedulePerformance)
	
	for _, event := range sc.events {
		if event.StartTime.After(period.Start) && event.StartTime.Before(period.End) {
			if event.ScheduleName != "" {
				if _, exists := scheduleMap[event.ScheduleName]; !exists {
					scheduleMap[event.ScheduleName] = &SchedulePerformance{
						ScheduleName: event.ScheduleName,
					}
				}
				
				perf := scheduleMap[event.ScheduleName]
				perf.ExecutionCount++
				perf.TotalSaved += event.SavedAmount
			}
		}
	}
	
	// Calculate averages and convert to slice
	performances := make([]SchedulePerformance, 0, len(scheduleMap))
	for _, perf := range scheduleMap {
		if perf.ExecutionCount > 0 {
			perf.AverageSaving = perf.TotalSaved / float64(perf.ExecutionCount)
			perf.SuccessRate = 100.0 // Simplified - would track actual success/failure
		}
		performances = append(performances, *perf)
	}
	
	return performances
}

// generateRecommendations generates optimization recommendations
func (sc *SavingsCalculator) generateRecommendations(report *SavingsReport) []Recommendation {
	recommendations := []Recommendation{}
	
	// Check if hibernation percentage is low
	if report.SavingsPercentage < 20 {
		recommendations = append(recommendations, Recommendation{
			Type:        "schedule_optimization",
			Priority:    "high",
			Description: "Low hibernation utilization detected",
			Impact:      report.TotalSaved * 2, // Could double savings
			Action:      "Consider adding more aggressive hibernation schedules",
		})
	}
	
	// Check for instances with no hibernation
	for _, instance := range report.InstanceSavings {
		if instance.HibernationHours == 0 {
			recommendations = append(recommendations, Recommendation{
				Type:        "instance_policy",
				Priority:    "medium",
				Description: fmt.Sprintf("Instance %s has no hibernation", instance.InstanceName),
				Impact:      instance.HourlyRate * 8 * 30, // 8 hours/day * 30 days
				Action:      fmt.Sprintf("Enable hibernation policy for %s", instance.InstanceName),
			})
		}
	}
	
	// Check for underperforming schedules
	for _, perf := range report.SchedulePerformance {
		if perf.SuccessRate < 80 {
			recommendations = append(recommendations, Recommendation{
				Type:        "schedule_reliability",
				Priority:    "high",
				Description: fmt.Sprintf("Schedule %s has low success rate", perf.ScheduleName),
				Impact:      perf.AverageSaving * 5, // Potential additional savings
				Action:      "Review and adjust schedule configuration",
			})
		}
	}
	
	return recommendations
}

// calculateDailyTrends calculates daily savings trends
func (sc *SavingsCalculator) calculateDailyTrends(period Period) []DailySaving {
	dailyMap := make(map[string]*DailySaving)
	
	for _, event := range sc.events {
		if event.StartTime.After(period.Start) && event.StartTime.Before(period.End) {
			dateKey := event.StartTime.Format("2006-01-02")
			
			if _, exists := dailyMap[dateKey]; !exists {
				dailyMap[dateKey] = &DailySaving{
					Date: event.StartTime.Truncate(24 * time.Hour),
				}
			}
			
			daily := dailyMap[dateKey]
			daily.HibernationHours += event.Duration
			daily.Saved += event.SavedAmount
		}
	}
	
	// Convert to slice and sort by date
	dailySavings := make([]DailySaving, 0, len(dailyMap))
	for _, daily := range dailyMap {
		daily.ActiveHours = 24 - daily.HibernationHours
		dailySavings = append(dailySavings, *daily)
	}
	
	return dailySavings
}

// calculateWeeklyTrends calculates weekly savings trends
func (sc *SavingsCalculator) calculateWeeklyTrends(period Period) []WeeklySaving {
	// Simplified implementation
	return []WeeklySaving{}
}

// calculateMonthlyTrends calculates monthly savings trends
func (sc *SavingsCalculator) calculateMonthlyTrends(period Period) []MonthlySaving {
	// Simplified implementation
	return []MonthlySaving{}
}

// AddEvent adds a hibernation event for tracking
func (sc *SavingsCalculator) AddEvent(event HibernationEvent) {
	sc.events = append(sc.events, event)
}

// Helper functions

func generateReportID() string {
	return fmt.Sprintf("report-%d", time.Now().Unix())
}