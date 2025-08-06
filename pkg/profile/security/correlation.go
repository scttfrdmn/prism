// Package security provides security event correlation and analysis
package security

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"
)

// SecurityCorrelationEngine provides advanced security event analysis and correlation
type SecurityCorrelationEngine struct {
	auditLogger   *SecurityAuditLogger
	ruleEngine    *CorrelationRuleEngine
	eventBuffer   []SecurityEvent
	correlations  []SecurityCorrelation
	patterns      map[string]*AttackPattern
}

// CorrelationRuleEngine manages correlation rules and pattern detection
type CorrelationRuleEngine struct {
	rules            []CorrelationRule
	attackPatterns   map[string]*AttackPattern
	deviceProfiles   map[string]*DeviceProfile
	baselineMetrics  *BaselineMetrics
}

// CorrelationRule defines rules for correlating security events
type CorrelationRule struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	EventTypes  []string               `json:"event_types"`
	TimeWindow  time.Duration          `json:"time_window"`
	Threshold   int                    `json:"threshold"`
	Severity    AlertSeverity         `json:"severity"`
	Conditions  map[string]interface{} `json:"conditions"`
	Actions     []string               `json:"actions"`
}

// SecurityCorrelation represents a correlation between multiple security events
type SecurityCorrelation struct {
	ID            string                 `json:"id"`
	Timestamp     time.Time              `json:"timestamp"`
	CorrelationType string               `json:"correlation_type"`
	Events        []SecurityEvent        `json:"events"`
	Pattern       string                 `json:"pattern"`
	RiskScore     int                    `json:"risk_score"`
	Confidence    float64                `json:"confidence"`
	Description   string                 `json:"description"`
	Recommendations []string             `json:"recommendations"`
	Metadata      map[string]interface{} `json:"metadata"`
}

// AttackPattern represents a known attack pattern for detection
type AttackPattern struct {
	Name           string                 `json:"name"`
	Description    string                 `json:"description"`
	EventSequence  []string               `json:"event_sequence"`
	TimeWindow     time.Duration          `json:"time_window"`
	RiskScore      int                    `json:"risk_score"`
	Indicators     []string               `json:"indicators"`
	Countermeasures []string              `json:"countermeasures"`
	Metadata       map[string]interface{} `json:"metadata"`
}

// DeviceProfile tracks normal behavior patterns for devices
type DeviceProfile struct {
	DeviceID           string            `json:"device_id"`
	FirstSeen          time.Time         `json:"first_seen"`
	LastSeen           time.Time         `json:"last_seen"`
	NormalOperations   map[string]int    `json:"normal_operations"`
	TypicalHours       []int             `json:"typical_hours"`
	AverageFrequency   float64           `json:"average_frequency"`
	SuspiciousActivity bool              `json:"suspicious_activity"`
	TrustScore         int               `json:"trust_score"` // 0-100
}

// BaselineMetrics tracks normal system behavior
type BaselineMetrics struct {
	StartTime                time.Time         `json:"start_time"`
	TotalEvents              int               `json:"total_events"`
	AverageEventsPerHour     float64           `json:"average_events_per_hour"`
	CommonEventTypes         map[string]int    `json:"common_event_types"`
	PeakActivityHours        []int             `json:"peak_activity_hours"`
	TypicalFailureRate       float64           `json:"typical_failure_rate"`
	NormalDeviceCount        int               `json:"normal_device_count"`
	LastUpdated              time.Time         `json:"last_updated"`
}

// ThreatIntelligence provides context for security correlations
type ThreatIntelligence struct {
	KnownAttackVectors    []string               `json:"known_attack_vectors"`
	CompromisedIndicators []string               `json:"compromised_indicators"`
	ThreatActorTTPs       map[string][]string    `json:"threat_actor_ttps"`
	IOCs                  []string               `json:"iocs"` // Indicators of Compromise
}

// NewSecurityCorrelationEngine creates a new correlation engine
func NewSecurityCorrelationEngine() (*SecurityCorrelationEngine, error) {
	auditLogger, err := NewSecurityAuditLogger()
	if err != nil {
		return nil, fmt.Errorf("failed to create audit logger: %w", err)
	}

	ruleEngine := &CorrelationRuleEngine{
		rules:           getDefaultCorrelationRules(),
		attackPatterns:  getKnownAttackPatterns(),
		deviceProfiles:  make(map[string]*DeviceProfile),
		baselineMetrics: &BaselineMetrics{
			StartTime:         time.Now(),
			CommonEventTypes:  make(map[string]int),
			PeakActivityHours: []int{9, 10, 11, 14, 15, 16}, // Default business hours
		},
	}

	engine := &SecurityCorrelationEngine{
		auditLogger:  auditLogger,
		ruleEngine:   ruleEngine,
		eventBuffer:  make([]SecurityEvent, 0),
		correlations: make([]SecurityCorrelation, 0),
		patterns:     ruleEngine.attackPatterns,
	}

	return engine, nil
}

// AnalyzeSecurityEvents performs comprehensive analysis and correlation of security events
func (e *SecurityCorrelationEngine) AnalyzeSecurityEvents() ([]SecurityCorrelation, error) {
	// Load recent security events
	events, err := e.loadRecentEvents()
	if err != nil {
		return nil, fmt.Errorf("failed to load events: %w", err)
	}

	// Update device profiles and baseline metrics
	e.updateDeviceProfiles(events)
	e.updateBaselineMetrics(events)

	// Perform correlation analysis
	correlations := e.performCorrelationAnalysis(events)

	// Detect attack patterns
	patternCorrelations := e.detectAttackPatterns(events)
	correlations = append(correlations, patternCorrelations...)

	// Analyze anomalies
	anomalyCorrelations := e.detectAnomalies(events)
	correlations = append(correlations, anomalyCorrelations...)

	// Update correlation history
	e.correlations = append(e.correlations, correlations...)

	// Keep only recent correlations (last 7 days)
	e.pruneOldCorrelations()

	return correlations, nil
}

// performCorrelationAnalysis applies correlation rules to detect related events
func (e *SecurityCorrelationEngine) performCorrelationAnalysis(events []SecurityEvent) []SecurityCorrelation {
	correlations := make([]SecurityCorrelation, 0)

	for _, rule := range e.ruleEngine.rules {
		matches := e.findRuleMatches(rule, events)
		if len(matches) >= rule.Threshold {
			correlation := SecurityCorrelation{
				ID:              fmt.Sprintf("%s-%d", rule.ID, time.Now().Unix()),
				Timestamp:       time.Now(),
				CorrelationType: "rule_based",
				Events:          matches,
				Pattern:         rule.Name,
				RiskScore:       e.calculateRiskScore(matches, rule),
				Confidence:      e.calculateConfidence(matches, rule),
				Description:     fmt.Sprintf("Correlation rule '%s' triggered with %d events", rule.Name, len(matches)),
				Recommendations: rule.Actions,
				Metadata: map[string]interface{}{
					"rule_id":    rule.ID,
					"threshold":  rule.Threshold,
					"time_window": rule.TimeWindow.String(),
				},
			}
			correlations = append(correlations, correlation)
		}
	}

	return correlations
}

// detectAttackPatterns identifies known attack patterns in event sequences
func (e *SecurityCorrelationEngine) detectAttackPatterns(events []SecurityEvent) []SecurityCorrelation {
	correlations := make([]SecurityCorrelation, 0)

	for _, pattern := range e.patterns {
		matches := e.findPatternMatches(pattern, events)
		if len(matches) > 0 {
			for _, match := range matches {
				correlation := SecurityCorrelation{
					ID:              fmt.Sprintf("pattern-%s-%d", pattern.Name, time.Now().Unix()),
					Timestamp:       time.Now(),
					CorrelationType: "attack_pattern",
					Events:          match,
					Pattern:         pattern.Name,
					RiskScore:       pattern.RiskScore,
					Confidence:      0.85, // High confidence for known patterns
					Description:     fmt.Sprintf("Detected attack pattern: %s", pattern.Description),
					Recommendations: pattern.Countermeasures,
					Metadata: map[string]interface{}{
						"pattern_name": pattern.Name,
						"indicators":   pattern.Indicators,
					},
				}
				correlations = append(correlations, correlation)
			}
		}
	}

	return correlations
}

// detectAnomalies identifies anomalous behavior patterns
func (e *SecurityCorrelationEngine) detectAnomalies(events []SecurityEvent) []SecurityCorrelation {
	correlations := make([]SecurityCorrelation, 0)

	// Device behavior anomalies
	deviceAnomalies := e.detectDeviceBehaviorAnomalies(events)
	correlations = append(correlations, deviceAnomalies...)

	// Temporal anomalies
	temporalAnomalies := e.detectTemporalAnomalies(events)
	correlations = append(correlations, temporalAnomalies...)

	// Frequency anomalies
	frequencyAnomalies := e.detectFrequencyAnomalies(events)
	correlations = append(correlations, frequencyAnomalies...)

	return correlations
}

// detectDeviceBehaviorAnomalies identifies devices exhibiting unusual behavior
func (e *SecurityCorrelationEngine) detectDeviceBehaviorAnomalies(events []SecurityEvent) []SecurityCorrelation {
	correlations := make([]SecurityCorrelation, 0)

	for deviceID, profile := range e.ruleEngine.deviceProfiles {
		deviceEvents := e.filterEventsByDevice(events, deviceID)
		if len(deviceEvents) == 0 {
			continue
		}

		// Check for behavior anomalies
		if e.isDeviceBehaviorAnomalous(profile, deviceEvents) {
			correlation := SecurityCorrelation{
				ID:              fmt.Sprintf("device-anomaly-%s-%d", deviceID, time.Now().Unix()),
				Timestamp:       time.Now(),
				CorrelationType: "behavioral_anomaly",
				Events:          deviceEvents,
				Pattern:         "unusual_device_behavior",
				RiskScore:       60,
				Confidence:      0.70,
				Description:     fmt.Sprintf("Device %s exhibiting unusual behavior patterns", deviceID),
				Recommendations: []string{
					"Review device activity logs",
					"Verify device identity and authorization",
					"Check for compromise indicators",
				},
				Metadata: map[string]interface{}{
					"device_id":    deviceID,
					"trust_score":  profile.TrustScore,
					"normal_ops":   len(profile.NormalOperations),
				},
			}
			correlations = append(correlations, correlation)
		}
	}

	return correlations
}

// detectTemporalAnomalies identifies activity outside normal time patterns
func (e *SecurityCorrelationEngine) detectTemporalAnomalies(events []SecurityEvent) []SecurityCorrelation {
	correlations := make([]SecurityCorrelation, 0)

	// Group events by hour
	hourlyActivity := make(map[int][]SecurityEvent)
	for _, event := range events {
		hour := event.Timestamp.Hour()
		hourlyActivity[hour] = append(hourlyActivity[hour], event)
	}

	// Check for activity during unusual hours
	for hour, hourEvents := range hourlyActivity {
		if !e.isTypicalActivityHour(hour) && len(hourEvents) > 5 {
			correlation := SecurityCorrelation{
				ID:              fmt.Sprintf("temporal-anomaly-%d-%d", hour, time.Now().Unix()),
				Timestamp:       time.Now(),
				CorrelationType: "temporal_anomaly",
				Events:          hourEvents,
				Pattern:         "unusual_time_activity",
				RiskScore:       40,
				Confidence:      0.60,
				Description:     fmt.Sprintf("Unusual activity detected during hour %d:00", hour),
				Recommendations: []string{
					"Review after-hours access policies",
					"Verify legitimacy of off-hours operations",
					"Check for automated processes or scheduled tasks",
				},
				Metadata: map[string]interface{}{
					"hour":        hour,
					"event_count": len(hourEvents),
				},
			}
			correlations = append(correlations, correlation)
		}
	}

	return correlations
}

// detectFrequencyAnomalies identifies unusual event frequency patterns
func (e *SecurityCorrelationEngine) detectFrequencyAnomalies(events []SecurityEvent) []SecurityCorrelation {
	correlations := make([]SecurityCorrelation, 0)

	// Calculate event frequency by type
	eventTypeFreq := make(map[string]int)
	for _, event := range events {
		eventTypeFreq[event.EventType]++
	}

	// Compare against baseline
	for eventType, count := range eventTypeFreq {
		baseline := e.ruleEngine.baselineMetrics.CommonEventTypes[eventType]
		if baseline > 0 && count > baseline*3 { // 3x normal frequency
			correlation := SecurityCorrelation{
				ID:              fmt.Sprintf("frequency-anomaly-%s-%d", eventType, time.Now().Unix()),
				Timestamp:       time.Now(),
				CorrelationType: "frequency_anomaly",
				Events:          e.filterEventsByType(events, eventType),
				Pattern:         "unusual_event_frequency",
				RiskScore:       50,
				Confidence:      0.65,
				Description:     fmt.Sprintf("Unusual frequency of %s events: %d vs baseline %d", eventType, count, baseline),
				Recommendations: []string{
					"Investigate cause of increased activity",
					"Check for automated attacks or misconfigurations",
					"Review system performance and capacity",
				},
				Metadata: map[string]interface{}{
					"event_type":      eventType,
					"current_count":   count,
					"baseline_count":  baseline,
					"frequency_ratio": float64(count) / float64(baseline),
				},
			}
			correlations = append(correlations, correlation)
		}
	}

	return correlations
}

// Helper methods

func (e *SecurityCorrelationEngine) loadRecentEvents() ([]SecurityEvent, error) {
	logPath := e.auditLogger.GetAuditLogPath()
	content, err := os.ReadFile(logPath)
	if err != nil {
		if os.IsNotExist(err) {
			return []SecurityEvent{}, nil
		}
		return nil, fmt.Errorf("failed to read audit log: %w", err)
	}

	lines := strings.Split(strings.TrimSpace(string(content)), "\n")
	events := make([]SecurityEvent, 0, len(lines))
	cutoff := time.Now().Add(-24 * time.Hour)

	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}

		var event SecurityEvent
		if err := json.Unmarshal([]byte(line), &event); err != nil {
			continue
		}

		if event.Timestamp.After(cutoff) {
			events = append(events, event)
		}
	}

	return events, nil
}

func (e *SecurityCorrelationEngine) updateDeviceProfiles(events []SecurityEvent) {
	for _, event := range events {
		if event.DeviceID == "" {
			continue
		}

		profile, exists := e.ruleEngine.deviceProfiles[event.DeviceID]
		if !exists {
			profile = &DeviceProfile{
				DeviceID:         event.DeviceID,
				FirstSeen:        event.Timestamp,
				NormalOperations: make(map[string]int),
				TypicalHours:     make([]int, 0),
				TrustScore:       75, // Start with neutral trust
			}
			e.ruleEngine.deviceProfiles[event.DeviceID] = profile
		}

		profile.LastSeen = event.Timestamp
		profile.NormalOperations[event.EventType]++

		// Update typical hours
		hour := event.Timestamp.Hour()
		if !contains(profile.TypicalHours, hour) {
			profile.TypicalHours = append(profile.TypicalHours, hour)
		}

		// Adjust trust score based on event success/failure
		if event.Success {
			if profile.TrustScore < 100 {
				profile.TrustScore++
			}
		} else {
			if profile.TrustScore > 0 {
				profile.TrustScore--
			}
		}
	}
}

func (e *SecurityCorrelationEngine) updateBaselineMetrics(events []SecurityEvent) {
	baseline := e.ruleEngine.baselineMetrics
	baseline.TotalEvents += len(events)
	baseline.LastUpdated = time.Now()

	// Update event type frequencies
	for _, event := range events {
		baseline.CommonEventTypes[event.EventType]++
	}

	// Calculate average events per hour
	duration := time.Since(baseline.StartTime)
	if duration.Hours() > 0 {
		baseline.AverageEventsPerHour = float64(baseline.TotalEvents) / duration.Hours()
	}

	// Update failure rate
	successCount := 0
	for _, event := range events {
		if event.Success {
			successCount++
		}
	}
	if len(events) > 0 {
		baseline.TypicalFailureRate = float64(len(events)-successCount) / float64(len(events))
	}
}

func (e *SecurityCorrelationEngine) findRuleMatches(rule CorrelationRule, events []SecurityEvent) []SecurityEvent {
	matches := make([]SecurityEvent, 0)
	cutoff := time.Now().Add(-rule.TimeWindow)

	for _, event := range events {
		if event.Timestamp.Before(cutoff) {
			continue
		}

		for _, eventType := range rule.EventTypes {
			if event.EventType == eventType {
				matches = append(matches, event)
				break
			}
		}
	}

	return matches
}

func (e *SecurityCorrelationEngine) findPatternMatches(pattern *AttackPattern, events []SecurityEvent) [][]SecurityEvent {
	matches := make([][]SecurityEvent, 0)
	cutoff := time.Now().Add(-pattern.TimeWindow)

	// Simple pattern matching - look for event sequences
	for i := 0; i < len(events)-len(pattern.EventSequence)+1; i++ {
		if events[i].Timestamp.Before(cutoff) {
			continue
		}

		match := make([]SecurityEvent, 0, len(pattern.EventSequence))
		sequenceIndex := 0

		for j := i; j < len(events) && sequenceIndex < len(pattern.EventSequence); j++ {
			if events[j].EventType == pattern.EventSequence[sequenceIndex] {
				match = append(match, events[j])
				sequenceIndex++
			}
		}

		if sequenceIndex == len(pattern.EventSequence) {
			matches = append(matches, match)
		}
	}

	return matches
}

func (e *SecurityCorrelationEngine) calculateRiskScore(events []SecurityEvent, rule CorrelationRule) int {
	baseScore := 50
	
	// Increase score for failed events
	failedCount := 0
	for _, event := range events {
		if !event.Success {
			failedCount++
		}
	}
	
	riskMultiplier := 1 + (float64(failedCount) / float64(len(events)))
	return int(float64(baseScore) * riskMultiplier)
}

func (e *SecurityCorrelationEngine) calculateConfidence(events []SecurityEvent, rule CorrelationRule) float64 {
	// Base confidence on event count vs threshold
	ratio := float64(len(events)) / float64(rule.Threshold)
	confidence := 0.5 + (ratio * 0.3) // Base 0.5, increase with more events
	
	if confidence > 1.0 {
		confidence = 1.0
	}
	
	return confidence
}

func (e *SecurityCorrelationEngine) isDeviceBehaviorAnomalous(profile *DeviceProfile, events []SecurityEvent) bool {
	// Simple heuristics for anomaly detection
	if profile.TrustScore < 30 {
		return true
	}

	// Check for unusual operation types
	currentOps := make(map[string]int)
	for _, event := range events {
		currentOps[event.EventType]++
	}

	for eventType := range currentOps {
		if _, exists := profile.NormalOperations[eventType]; !exists {
			return true // New operation type
		}
	}

	return false
}

func (e *SecurityCorrelationEngine) isTypicalActivityHour(hour int) bool {
	for _, typicalHour := range e.ruleEngine.baselineMetrics.PeakActivityHours {
		if hour == typicalHour {
			return true
		}
	}
	return false
}

func (e *SecurityCorrelationEngine) filterEventsByDevice(events []SecurityEvent, deviceID string) []SecurityEvent {
	filtered := make([]SecurityEvent, 0)
	for _, event := range events {
		if event.DeviceID == deviceID {
			filtered = append(filtered, event)
		}
	}
	return filtered
}

func (e *SecurityCorrelationEngine) filterEventsByType(events []SecurityEvent, eventType string) []SecurityEvent {
	filtered := make([]SecurityEvent, 0)
	for _, event := range events {
		if event.EventType == eventType {
			filtered = append(filtered, event)
		}
	}
	return filtered
}

func (e *SecurityCorrelationEngine) pruneOldCorrelations() {
	cutoff := time.Now().Add(-7 * 24 * time.Hour)
	pruned := make([]SecurityCorrelation, 0)
	
	for _, correlation := range e.correlations {
		if correlation.Timestamp.After(cutoff) {
			pruned = append(pruned, correlation)
		}
	}
	
	e.correlations = pruned
}

// Default correlation rules
func getDefaultCorrelationRules() []CorrelationRule {
	return []CorrelationRule{
		{
			ID:          "failed-auth-burst",
			Name:        "Multiple Authentication Failures",
			Description: "Multiple authentication failures in short time period",
			EventTypes:  []string{"access_attempt", "device_registration"},
			TimeWindow:  15 * time.Minute,
			Threshold:   5,
			Severity:    AlertSeverityHigh,
			Actions: []string{
				"Block suspicious source",
				"Require additional authentication",
				"Alert security team",
			},
		},
		{
			ID:          "tamper-followed-by-access",
			Name:        "Tamper Detection with Subsequent Access",
			Description: "File tampering followed by access attempts",
			EventTypes:  []string{"tamper_detected", "access_attempt"},
			TimeWindow:  30 * time.Minute,
			Threshold:   2,
			Severity:    AlertSeverityCritical,
			Actions: []string{
				"Immediate security review",
				"Block all access from device",
				"Forensic analysis",
			},
		},
	}
}

// Known attack patterns
func getKnownAttackPatterns() map[string]*AttackPattern {
	return map[string]*AttackPattern{
		"credential-stuffing": {
			Name:          "Credential Stuffing Attack",
			Description:   "Automated attempts to access accounts using lists of known passwords",
			EventSequence: []string{"access_attempt", "access_attempt", "access_attempt"},
			TimeWindow:    5 * time.Minute,
			RiskScore:     80,
			Indicators:    []string{"rapid_failed_attempts", "multiple_devices", "common_passwords"},
			Countermeasures: []string{
				"Implement rate limiting",
				"Enable account lockout policies",
				"Deploy CAPTCHA challenges",
			},
		},
		"privilege-escalation": {
			Name:          "Privilege Escalation Attempt",
			Description:   "Attempt to gain elevated system privileges",
			EventSequence: []string{"keychain_operation", "access_attempt", "tamper_detected"},
			TimeWindow:    10 * time.Minute,
			RiskScore:     90,
			Indicators:    []string{"keychain_access", "file_modifications", "privilege_changes"},
			Countermeasures: []string{
				"Review access controls",
				"Monitor privileged operations",
				"Enable detailed audit logging",
			},
		},
	}
}

// Utility functions
func contains(slice []int, item int) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// Close closes the correlation engine and associated resources
func (e *SecurityCorrelationEngine) Close() error {
	if e.auditLogger != nil {
		return e.auditLogger.Close()
	}
	return nil
}