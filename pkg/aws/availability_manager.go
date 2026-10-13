package aws

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	ec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
)

// AvailabilityManager handles AZ availability tracking and intelligent failover
type AvailabilityManager struct {
	ec2Client *ec2.Client
	region    string

	// Track AZ health for instance types
	azHealthMap map[string]*AZHealthTracker
	mu          sync.RWMutex
}

// AZHealthTracker tracks success/failure rates for instance launches in an AZ
type AZHealthTracker struct {
	AvailabilityZone string
	InstanceType     string
	SuccessCount     int
	FailureCount     int
	LastSuccess      time.Time
	LastFailure      time.Time
	SuccessRate      float64
}

// AZRecommendation provides AZ selection recommendations
type AZRecommendation struct {
	PreferredAZ      string
	AlternateAZs     []string
	Reason           string
	AvailabilityZone *AZHealthTracker
}

// NewAvailabilityManager creates a new availability manager
func NewAvailabilityManager(cfg aws.Config, region string) *AvailabilityManager {
	return &AvailabilityManager{
		ec2Client:   ec2.NewFromConfig(cfg),
		region:      region,
		azHealthMap: make(map[string]*AZHealthTracker),
	}
}

// isStandardAZ determines if an AZ is a standard availability zone (not a local zone or wavelength zone)
// Standard AZs follow the pattern: {region}{letter} (e.g., us-west-2a, eu-west-1b)
// Local zones have extra components: {region}-{city}-{number}{letter} (e.g., us-west-2-lax-1a)
// Wavelength zones contain "wl": {region}-wl1-{city}-wlz-1 (e.g., us-east-1-wl1-nyc-wlz-1)
func (am *AvailabilityManager) isStandardAZ(az string) bool {
	// Wavelength zones contain "wl" - exclude them
	if strings.Contains(az, "-wl") {
		return false
	}

	// Local zones have more than one hyphen after the region
	// Standard AZ: us-west-2a (region "us-west-2" + letter "a")
	// Local zone: us-west-2-lax-1a (region "us-west-2" + extra parts)

	// Remove the region prefix to check what's left
	// For us-west-2a, after removing us-west-2, we have "a" (single letter)
	// For us-west-2-lax-1a, after removing us-west-2, we have "-lax-1a" (extra components)

	if !strings.HasPrefix(az, am.region) {
		return false
	}

	suffix := strings.TrimPrefix(az, am.region)

	// Standard AZ should have just a single letter (a-z) after the region
	// It might have no prefix (unlikely but handle it), or just the letter
	if len(suffix) == 0 {
		return false
	}

	// Should be just a single lowercase letter (no hyphens or additional characters)
	return len(suffix) == 1 && suffix >= "a" && suffix <= "z"
}

// GetAvailableAZsForInstanceType returns available standard AZs for a given instance type
// Filters out local zones and wavelength zones to avoid subnet mismatches
func (am *AvailabilityManager) GetAvailableAZsForInstanceType(ctx context.Context, instanceType string) ([]string, error) {
	input := &ec2.DescribeInstanceTypeOfferingsInput{
		LocationType: ec2types.LocationTypeAvailabilityZone,
		Filters: []ec2types.Filter{
			{
				Name:   aws.String("instance-type"),
				Values: []string{instanceType},
			},
		},
	}

	output, err := am.ec2Client.DescribeInstanceTypeOfferings(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to describe instance type offerings: %w", err)
	}

	var azs []string
	for _, offering := range output.InstanceTypeOfferings {
		if offering.Location != nil {
			az := *offering.Location
			// Only include standard AZs (exclude local zones and wavelength zones)
			if am.isStandardAZ(az) {
				azs = append(azs, az)
			}
		}
	}

	if len(azs) == 0 {
		return nil, fmt.Errorf("instance type %s not available in any standard AZ in region %s (local zones and wavelength zones are excluded)", instanceType, am.region)
	}

	return azs, nil
}

// GetRecommendedAZ returns the recommended AZ for launching an instance
func (am *AvailabilityManager) GetRecommendedAZ(ctx context.Context, instanceType string) (*AZRecommendation, error) {
	// Get all available AZs for this instance type
	availableAZs, err := am.GetAvailableAZsForInstanceType(ctx, instanceType)
	if err != nil {
		return nil, err
	}

	if len(availableAZs) == 0 {
		return nil, fmt.Errorf("no available AZs for instance type %s", instanceType)
	}

	// Calculate success rates for each AZ
	am.mu.RLock()
	azScores := make(map[string]float64)
	azTrackers := make(map[string]*AZHealthTracker)

	for _, az := range availableAZs {
		key := am.getHealthKey(az, instanceType)
		tracker := am.azHealthMap[key]

		if tracker == nil {
			// New AZ, give it a neutral score
			azScores[az] = 0.5
		} else {
			// Calculate score based on success rate and recency
			azScores[az] = am.calculateAZScore(tracker)
			azTrackers[az] = tracker
		}
	}
	am.mu.RUnlock()

	// Find the best AZ
	var bestAZ string
	bestScore := -1.0

	for az, score := range azScores {
		if score > bestScore {
			bestScore = score
			bestAZ = az
		}
	}

	// Build alternate AZs list (excluding the best one)
	var alternates []string
	for _, az := range availableAZs {
		if az != bestAZ {
			alternates = append(alternates, az)
		}
	}

	recommendation := &AZRecommendation{
		PreferredAZ:  bestAZ,
		AlternateAZs: alternates,
		Reason:       am.generateRecommendationReason(bestAZ, azTrackers[bestAZ]),
	}

	if tracker := azTrackers[bestAZ]; tracker != nil {
		recommendation.AvailabilityZone = tracker
	}

	return recommendation, nil
}

// RecordLaunchSuccess records a successful launch in an AZ
func (am *AvailabilityManager) RecordLaunchSuccess(az, instanceType string) {
	am.mu.Lock()
	defer am.mu.Unlock()

	key := am.getHealthKey(az, instanceType)
	tracker := am.azHealthMap[key]

	if tracker == nil {
		tracker = &AZHealthTracker{
			AvailabilityZone: az,
			InstanceType:     instanceType,
		}
		am.azHealthMap[key] = tracker
	}

	tracker.SuccessCount++
	tracker.LastSuccess = time.Now()
	tracker.SuccessRate = am.calculateSuccessRate(tracker)
}

// RecordLaunchFailure records a failed launch in an AZ
func (am *AvailabilityManager) RecordLaunchFailure(az, instanceType string, reason string) {
	am.mu.Lock()
	defer am.mu.Unlock()

	key := am.getHealthKey(az, instanceType)
	tracker := am.azHealthMap[key]

	if tracker == nil {
		tracker = &AZHealthTracker{
			AvailabilityZone: az,
			InstanceType:     instanceType,
		}
		am.azHealthMap[key] = tracker
	}

	tracker.FailureCount++
	tracker.LastFailure = time.Now()
	tracker.SuccessRate = am.calculateSuccessRate(tracker)
}

// AttemptLaunchWithFailover attempts to launch an instance with automatic AZ failover
func (am *AvailabilityManager) AttemptLaunchWithFailover(
	ctx context.Context,
	instanceType string,
	launchFunc func(ctx context.Context, az string) (string, error),
) (instanceID string, az string, err error) {
	// Get recommended AZ ordering
	recommendation, err := am.GetRecommendedAZ(ctx, instanceType)
	if err != nil {
		return "", "", fmt.Errorf("failed to get AZ recommendation: %w", err)
	}

	// Try preferred AZ first
	azOrder := append([]string{recommendation.PreferredAZ}, recommendation.AlternateAZs...)

	var lastErr error
	for i, targetAZ := range azOrder {
		// Attempt launch in this AZ
		instanceID, err := launchFunc(ctx, targetAZ)

		if err == nil {
			// Success!
			am.RecordLaunchSuccess(targetAZ, instanceType)
			return instanceID, targetAZ, nil
		}

		// Record failure
		am.RecordLaunchFailure(targetAZ, instanceType, err.Error())
		lastErr = err

		// Check if this is a capacity error
		if am.isCapacityError(err) {
			// Try next AZ
			if i < len(azOrder)-1 {
				// Log that we're failing over
				fmt.Printf("InsufficientInstanceCapacity in %s, trying %s...\n", targetAZ, azOrder[i+1])
				continue
			}
		} else {
			// Not a capacity error, don't retry
			return "", "", err
		}
	}

	// All AZs exhausted
	return "", "", fmt.Errorf("failed to launch in all available AZs: %w", lastErr)
}

// IsCapacityError checks if an error is due to insufficient capacity
func (am *AvailabilityManager) isCapacityError(err error) bool {
	if err == nil {
		return false
	}

	errStr := err.Error()
	capacityErrors := []string{
		"InsufficientInstanceCapacity",
		"Insufficient capacity",
		"capacity not available",
	}

	for _, pattern := range capacityErrors {
		if strings.Contains(errStr, pattern) {
			return true
		}
	}

	return false
}

// GetAZHealthReport returns a health report for all tracked AZs
func (am *AvailabilityManager) GetAZHealthReport() map[string]*AZHealthTracker {
	am.mu.RLock()
	defer am.mu.RUnlock()

	// Create a copy to avoid concurrent access issues
	report := make(map[string]*AZHealthTracker)
	for key, tracker := range am.azHealthMap {
		// Create a copy of the tracker
		trackerCopy := *tracker
		report[key] = &trackerCopy
	}

	return report
}

// ClearHealthData clears all health tracking data (useful for testing)
func (am *AvailabilityManager) ClearHealthData() {
	am.mu.Lock()
	defer am.mu.Unlock()

	am.azHealthMap = make(map[string]*AZHealthTracker)
}

// getHealthKey creates a unique key for AZ + instance type combination
func (am *AvailabilityManager) getHealthKey(az, instanceType string) string {
	return fmt.Sprintf("%s:%s", az, instanceType)
}

// calculateSuccessRate calculates the success rate for an AZ tracker
func (am *AvailabilityManager) calculateSuccessRate(tracker *AZHealthTracker) float64 {
	total := tracker.SuccessCount + tracker.FailureCount
	if total == 0 {
		return 0.5 // Neutral for new AZs
	}

	return float64(tracker.SuccessCount) / float64(total)
}

// calculateAZScore calculates a score for AZ selection (0.0 to 1.0)
// Factors: success rate, recency of success, failure recency
func (am *AvailabilityManager) calculateAZScore(tracker *AZHealthTracker) float64 {
	baseScore := tracker.SuccessRate

	// Boost score for recent successes
	if !tracker.LastSuccess.IsZero() {
		timeSinceSuccess := time.Since(tracker.LastSuccess)
		if timeSinceSuccess < 1*time.Hour {
			baseScore += 0.1 // Recent success is good
		}
	}

	// Penalize for recent failures
	if !tracker.LastFailure.IsZero() {
		timeSinceFailure := time.Since(tracker.LastFailure)
		if timeSinceFailure < 1*time.Hour {
			baseScore -= 0.2 // Recent failure is bad
		}
	}

	// Clamp score to [0, 1]
	if baseScore < 0 {
		baseScore = 0
	}
	if baseScore > 1 {
		baseScore = 1
	}

	return baseScore
}

// generateRecommendationReason creates a human-readable reason for AZ recommendation
func (am *AvailabilityManager) generateRecommendationReason(az string, tracker *AZHealthTracker) string {
	if tracker == nil {
		return fmt.Sprintf("No prior launch history in %s", az)
	}

	if tracker.SuccessRate >= 0.9 {
		return fmt.Sprintf("High success rate (%.0f%%) in %s", tracker.SuccessRate*100, az)
	}

	if tracker.SuccessRate >= 0.7 {
		return fmt.Sprintf("Good success rate (%.0f%%) in %s", tracker.SuccessRate*100, az)
	}

	if !tracker.LastSuccess.IsZero() && time.Since(tracker.LastSuccess) < 1*time.Hour {
		return fmt.Sprintf("Recent successful launch in %s", az)
	}

	return fmt.Sprintf("Available in %s", az)
}
