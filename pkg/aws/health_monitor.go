package aws

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/health"
	healthtypes "github.com/aws/aws-sdk-go-v2/service/health/types"
)

// HealthMonitor monitors AWS service health and provides proactive notifications
type HealthMonitor struct {
	healthClient *health.Client
	region       string

	// Cache for health events
	eventCache     map[string]*HealthEvent
	lastCheck      time.Time
	mu             sync.RWMutex
	cacheTTL       time.Duration
	supportEnabled bool // Whether AWS Health API is available (Business/Enterprise support)
}

// HealthEvent represents an AWS service health event
type HealthEvent struct {
	EventARN          string
	Service           string
	EventTypeCode     string
	EventTypeCategory string
	Region            string
	StartTime         time.Time
	EndTime           *time.Time
	LastUpdatedTime   time.Time
	StatusCode        string
	Description       string
	AffectedEntities  []string
}

// HealthStatus represents the overall health status for a service/region
type HealthStatus struct {
	Service           string
	Region            string
	IsHealthy         bool
	ActiveEvents      int
	ImpactLevel       string // "none", "low", "medium", "high"
	RecommendedAction string
	Events            []*HealthEvent
}

// RegionHealthSummary provides a summary of health across all monitored regions
type RegionHealthSummary struct {
	Regions         map[string]*HealthStatus
	HealthyRegions  []string
	AffectedRegions []string
	TotalEvents     int
}

// NewHealthMonitor creates a new AWS Health monitor
func NewHealthMonitor(cfg aws.Config, region string) *HealthMonitor {
	return &HealthMonitor{
		healthClient:   health.NewFromConfig(cfg),
		region:         region,
		eventCache:     make(map[string]*HealthEvent),
		cacheTTL:       5 * time.Minute, // Cache health events for 5 minutes
		supportEnabled: true,            // Assume enabled, will detect on first API call
	}
}

// CheckServiceHealth checks the health of EC2 service in a specific region
func (hm *HealthMonitor) CheckServiceHealth(ctx context.Context, region string) (*HealthStatus, error) {
	// Check if Health API is available
	if !hm.supportEnabled {
		return &HealthStatus{
			Service:           "ec2",
			Region:            region,
			IsHealthy:         true, // Assume healthy if we can't check
			ImpactLevel:       "none",
			RecommendedAction: "AWS Health API unavailable (requires Business or Enterprise Support). Assuming service is healthy.",
		}, nil
	}

	// Get active health events for EC2 in this region
	events, err := hm.getActiveEvents(ctx, "EC2", region)
	if err != nil {
		// Check if this is a subscription error
		if strings.Contains(err.Error(), "SubscriptionRequiredException") {
			hm.supportEnabled = false
			return hm.CheckServiceHealth(ctx, region) // Recurse with support disabled
		}
		return nil, fmt.Errorf("failed to get health events: %w", err)
	}

	status := &HealthStatus{
		Service:      "EC2",
		Region:       region,
		ActiveEvents: len(events),
		Events:       events,
	}

	// Determine health status and impact level
	if len(events) == 0 {
		status.IsHealthy = true
		status.ImpactLevel = "none"
		status.RecommendedAction = "Service is operating normally"
	} else {
		// Analyze events to determine impact
		status.IsHealthy, status.ImpactLevel = hm.analyzeEventImpact(events)
		status.RecommendedAction = hm.generateRecommendedAction(status)
	}

	return status, nil
}

// GetRegionHealthSummary returns health status for all supported regions
func (hm *HealthMonitor) GetRegionHealthSummary(ctx context.Context, regions []string) (*RegionHealthSummary, error) {
	summary := &RegionHealthSummary{
		Regions: make(map[string]*HealthStatus),
	}

	for _, region := range regions {
		status, err := hm.CheckServiceHealth(ctx, region)
		if err != nil {
			// Log error but continue checking other regions
			continue
		}

		summary.Regions[region] = status
		summary.TotalEvents += status.ActiveEvents

		if status.IsHealthy {
			summary.HealthyRegions = append(summary.HealthyRegions, region)
		} else {
			summary.AffectedRegions = append(summary.AffectedRegions, region)
		}
	}

	return summary, nil
}

// ShouldBlockLaunch determines if a launch should be blocked due to service health issues
func (hm *HealthMonitor) ShouldBlockLaunch(ctx context.Context, region string) (bool, string) {
	status, err := hm.CheckServiceHealth(ctx, region)
	if err != nil {
		// If we can't check health, don't block (fail open)
		return false, ""
	}

	// Block launches if impact is high
	if status.ImpactLevel == "high" {
		return true, fmt.Sprintf(
			"EC2 service in %s is experiencing issues (Impact: %s). %d active events. %s",
			region, status.ImpactLevel, status.ActiveEvents, status.RecommendedAction,
		)
	}

	// Warn but don't block for medium impact
	if status.ImpactLevel == "medium" {
		return false, fmt.Sprintf(
			"Warning: EC2 service in %s has %d active events (Impact: %s). Launch may experience issues.",
			region, status.ActiveEvents, status.ImpactLevel,
		)
	}

	return false, ""
}

// SuggestAlternativeRegions suggests alternative healthy regions
func (hm *HealthMonitor) SuggestAlternativeRegions(ctx context.Context, currentRegion string, allRegions []string) ([]string, error) {
	summary, err := hm.GetRegionHealthSummary(ctx, allRegions)
	if err != nil {
		return nil, err
	}

	// Filter out the current region and return only healthy alternatives
	var alternatives []string
	for _, region := range summary.HealthyRegions {
		if region != currentRegion {
			alternatives = append(alternatives, region)
		}
	}

	return alternatives, nil
}

// getActiveEvents retrieves active health events for a service/region
func (hm *HealthMonitor) getActiveEvents(ctx context.Context, service, region string) ([]*HealthEvent, error) {
	// Check cache first
	hm.mu.RLock()
	if time.Since(hm.lastCheck) < hm.cacheTTL {
		// Cache is still valid
		var events []*HealthEvent
		for _, event := range hm.eventCache {
			if event.Service == service && event.Region == region {
				events = append(events, event)
			}
		}
		hm.mu.RUnlock()
		return events, nil
	}
	hm.mu.RUnlock()

	// Cache miss or expired, fetch from API
	filter := &healthtypes.EventFilter{
		Services: []string{service},
		Regions:  []string{region},
		EventStatusCodes: []healthtypes.EventStatusCode{
			healthtypes.EventStatusCodeOpen,
			healthtypes.EventStatusCodeUpcoming,
		},
	}

	input := &health.DescribeEventsInput{
		Filter: filter,
	}

	output, err := hm.healthClient.DescribeEvents(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to describe events: %w", err)
	}

	// Convert to our event format
	var events []*HealthEvent
	for _, awsEvent := range output.Events {
		event := &HealthEvent{
			EventARN:          aws.ToString(awsEvent.Arn),
			Service:           aws.ToString(awsEvent.Service),
			EventTypeCode:     aws.ToString(awsEvent.EventTypeCode),
			EventTypeCategory: string(awsEvent.EventTypeCategory),
			Region:            aws.ToString(awsEvent.Region),
			StartTime:         aws.ToTime(awsEvent.StartTime),
			LastUpdatedTime:   aws.ToTime(awsEvent.LastUpdatedTime),
			StatusCode:        string(awsEvent.StatusCode),
		}

		if awsEvent.EndTime != nil {
			endTime := aws.ToTime(awsEvent.EndTime)
			event.EndTime = &endTime
		}

		// Note: Event descriptions require separate DescribeEventDetails call
		// For now, use EventTypeCode as description for performance
		event.Description = aws.ToString(awsEvent.EventTypeCode)

		events = append(events, event)

		// Update cache
		hm.mu.Lock()
		hm.eventCache[event.EventARN] = event
		hm.mu.Unlock()
	}

	// Update last check time
	hm.mu.Lock()
	hm.lastCheck = time.Now()
	hm.mu.Unlock()

	return events, nil
}

// analyzeEventImpact analyzes events to determine impact level
func (hm *HealthMonitor) analyzeEventImpact(events []*HealthEvent) (bool, string) {
	if len(events) == 0 {
		return true, "none"
	}

	// Count events by category
	issueCount := 0
	scheduledChangeCount := 0

	for _, event := range events {
		switch event.EventTypeCategory {
		case "issue":
			issueCount++
		case "scheduledChange":
			scheduledChangeCount++
		}
	}

	// Determine impact level
	if issueCount > 0 {
		// Any active issue is at least medium impact
		if issueCount >= 3 {
			return false, "high" // Multiple concurrent issues
		}
		return false, "medium"
	}

	if scheduledChangeCount > 0 {
		return true, "low" // Scheduled changes are low impact
	}

	return true, "none"
}

// generateRecommendedAction generates recommended actions based on health status
func (hm *HealthMonitor) generateRecommendedAction(status *HealthStatus) string {
	if status.IsHealthy {
		if status.ActiveEvents > 0 {
			return fmt.Sprintf("Service is healthy but has %d scheduled maintenance events. Monitor for updates.", status.ActiveEvents)
		}
		return "Service is operating normally"
	}

	// Service is unhealthy
	var actions []string

	switch status.ImpactLevel {
	case "high":
		actions = append(actions, "Consider using an alternative region")
		actions = append(actions, "Run: prism admin aws-health --all-regions")
		actions = append(actions, "Monitor AWS Service Health Dashboard for updates")

	case "medium":
		actions = append(actions, "Launch may succeed but could experience delays")
		actions = append(actions, "Consider waiting 15-30 minutes if not urgent")
		actions = append(actions, "Monitor AWS Service Health Dashboard")
	}

	return strings.Join(actions, ". ")
}

// ClearCache clears the event cache (useful for testing)
func (hm *HealthMonitor) ClearCache() {
	hm.mu.Lock()
	defer hm.mu.Unlock()

	hm.eventCache = make(map[string]*HealthEvent)
	hm.lastCheck = time.Time{}
}

// FormatHealthReport generates a formatted health report
func (hm *HealthMonitor) FormatHealthReport(summary *RegionHealthSummary) string {
	var builder strings.Builder

	builder.WriteString("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
	builder.WriteString("🏥 AWS Service Health Report\n")
	builder.WriteString("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n\n")

	builder.WriteString(fmt.Sprintf("Total Regions Checked: %d\n", len(summary.Regions)))
	builder.WriteString(fmt.Sprintf("Healthy Regions: %d\n", len(summary.HealthyRegions)))
	builder.WriteString(fmt.Sprintf("Affected Regions: %d\n", len(summary.AffectedRegions)))
	builder.WriteString(fmt.Sprintf("Total Active Events: %d\n\n", summary.TotalEvents))

	if len(summary.HealthyRegions) > 0 {
		builder.WriteString("✅ Healthy Regions:\n")
		for _, region := range summary.HealthyRegions {
			builder.WriteString(fmt.Sprintf("  • %s\n", region))
		}
		builder.WriteString("\n")
	}

	if len(summary.AffectedRegions) > 0 {
		builder.WriteString("⚠️  Affected Regions:\n")
		for _, region := range summary.AffectedRegions {
			status := summary.Regions[region]
			builder.WriteString(fmt.Sprintf("  • %s - Impact: %s (%d events)\n", region, status.ImpactLevel, status.ActiveEvents))
		}
		builder.WriteString("\n")
	}

	builder.WriteString("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")

	return builder.String()
}
