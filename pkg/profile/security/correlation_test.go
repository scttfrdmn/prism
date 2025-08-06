// Package security provides tests for security event correlation and analysis
package security

import (
	"testing"
	"time"
)

// TestNewSecurityCorrelationEngine validates correlation engine creation
func TestNewSecurityCorrelationEngine(t *testing.T) {
	engine, err := NewSecurityCorrelationEngine()
	if err != nil {
		t.Fatalf("Failed to create correlation engine: %v", err)
	}
	defer engine.Close()

	if engine.auditLogger == nil {
		t.Error("Audit logger should not be nil")
	}

	if engine.ruleEngine == nil {
		t.Error("Rule engine should not be nil")
	}

	if len(engine.ruleEngine.rules) == 0 {
		t.Error("Should have default correlation rules")
	}

	if len(engine.patterns) == 0 {
		t.Error("Should have default attack patterns")
	}

	if engine.ruleEngine.baselineMetrics == nil {
		t.Error("Baseline metrics should not be nil")
	}

	t.Log("✅ Security correlation engine created successfully")
}

// TestCorrelationRuleMatching validates rule-based event correlation
func TestCorrelationRuleMatching(t *testing.T) {
	engine, err := NewSecurityCorrelationEngine()
	if err != nil {
		t.Fatalf("Failed to create correlation engine: %v", err)
	}
	defer engine.Close()

	// Create test events that should trigger correlation rules
	events := []SecurityEvent{
		{
			EventType: "access_attempt",
			Success:   false,
			DeviceID:  "suspicious-device",
			Timestamp: time.Now(),
		},
		{
			EventType: "access_attempt",
			Success:   false,
			DeviceID:  "suspicious-device",
			Timestamp: time.Now(),
		},
		{
			EventType: "access_attempt",
			Success:   false,
			DeviceID:  "suspicious-device",
			Timestamp: time.Now(),
		},
		{
			EventType: "access_attempt",
			Success:   false,
			DeviceID:  "suspicious-device",
			Timestamp: time.Now(),
		},
		{
			EventType: "access_attempt",
			Success:   false,
			DeviceID:  "suspicious-device",
			Timestamp: time.Now(),
		},
	}

	// Update device profiles and perform correlation
	engine.updateDeviceProfiles(events)
	correlations := engine.performCorrelationAnalysis(events)

	// Should detect multiple authentication failures
	if len(correlations) == 0 {
		t.Error("Should detect correlation for multiple failed attempts")
	}

	foundFailedAuthRule := false
	for _, correlation := range correlations {
		if correlation.Pattern == "Multiple Authentication Failures" {
			foundFailedAuthRule = true
			if correlation.RiskScore <= 0 {
				t.Error("Risk score should be greater than 0")
			}
			if correlation.Confidence <= 0 {
				t.Error("Confidence should be greater than 0")
			}
		}
	}

	if !foundFailedAuthRule {
		t.Error("Should trigger 'Multiple Authentication Failures' rule")
	}

	t.Log("✅ Correlation rule matching validated")
}

// TestAttackPatternDetection validates attack pattern recognition
func TestAttackPatternDetection(t *testing.T) {
	engine, err := NewSecurityCorrelationEngine()
	if err != nil {
		t.Fatalf("Failed to create correlation engine: %v", err)
	}
	defer engine.Close()

	// Create events matching credential stuffing pattern
	events := []SecurityEvent{
		{
			EventType: "access_attempt",
			Success:   false,
			DeviceID:  "attacker-device",
			Timestamp: time.Now().Add(-2 * time.Minute),
		},
		{
			EventType: "access_attempt",
			Success:   false,
			DeviceID:  "attacker-device",
			Timestamp: time.Now().Add(-1 * time.Minute),
		},
		{
			EventType: "access_attempt",
			Success:   false,
			DeviceID:  "attacker-device",
			Timestamp: time.Now(),
		},
	}

	correlations := engine.detectAttackPatterns(events)

	// Should detect credential stuffing pattern
	foundCredentialStuffing := false
	for _, correlation := range correlations {
		if correlation.Pattern == "Credential Stuffing Attack" {
			foundCredentialStuffing = true
			if correlation.CorrelationType != "attack_pattern" {
				t.Error("Correlation type should be 'attack_pattern'")
			}
			if correlation.RiskScore != 80 {
				t.Errorf("Expected risk score 80, got %d", correlation.RiskScore)
			}
			if len(correlation.Recommendations) == 0 {
				t.Error("Should include countermeasures")
			}
		}
	}

	if !foundCredentialStuffing {
		t.Error("Should detect credential stuffing attack pattern")
	}

	t.Log("✅ Attack pattern detection validated")
}

// TestDeviceBehaviorProfiling validates device behavior analysis
func TestDeviceBehaviorProfiling(t *testing.T) {
	engine, err := NewSecurityCorrelationEngine()
	if err != nil {
		t.Fatalf("Failed to create correlation engine: %v", err)
	}
	defer engine.Close()

	// Create normal behavior events for a device
	normalEvents := []SecurityEvent{
		{
			EventType: "device_registration",
			Success:   true,
			DeviceID:  "normal-device",
			Timestamp: time.Now().Add(-3 * time.Hour),
		},
		{
			EventType: "keychain_operation",
			Success:   true,
			DeviceID:  "normal-device",
			Timestamp: time.Now().Add(-2 * time.Hour),
		},
		{
			EventType: "access_attempt",
			Success:   true,
			DeviceID:  "normal-device",
			Timestamp: time.Now().Add(-1 * time.Hour),
		},
	}

	// Update device profile with normal behavior
	engine.updateDeviceProfiles(normalEvents)

	profile := engine.ruleEngine.deviceProfiles["normal-device"]
	if profile == nil {
		t.Fatal("Device profile should be created")
	}

	if profile.TrustScore <= 75 {
		t.Error("Trust score should increase with successful operations")
	}

	if len(profile.NormalOperations) != 3 {
		t.Errorf("Expected 3 normal operations, got %d", len(profile.NormalOperations))
	}

	// Create suspicious behavior events
	suspiciousEvents := []SecurityEvent{
		{
			EventType: "tamper_detected", // New operation type
			Success:   false,
			DeviceID:  "normal-device",
			Timestamp: time.Now(),
		},
	}

	// Check if behavior is considered anomalous
	if !engine.isDeviceBehaviorAnomalous(profile, suspiciousEvents) {
		t.Error("Should detect anomalous behavior for new operation type")
	}

	t.Log("✅ Device behavior profiling validated")
}

// TestTemporalAnomalyDetection validates detection of unusual activity times
func TestTemporalAnomalyDetection(t *testing.T) {
	engine, err := NewSecurityCorrelationEngine()
	if err != nil {
		t.Fatalf("Failed to create correlation engine: %v", err)
	}
	defer engine.Close()

	// Create events during unusual hours (2 AM)
	nightEvents := make([]SecurityEvent, 6) // More than threshold of 5
	for i := range nightEvents {
		nightEvents[i] = SecurityEvent{
			EventType: "access_attempt",
			Success:   true,
			DeviceID:  "night-device",
			Timestamp: time.Date(2025, 1, 1, 2, i*5, 0, 0, time.UTC), // 2 AM
		}
	}

	correlations := engine.detectTemporalAnomalies(nightEvents)

	// Should detect unusual time activity
	foundTemporalAnomaly := false
	for _, correlation := range correlations {
		if correlation.Pattern == "unusual_time_activity" {
			foundTemporalAnomaly = true
			if correlation.CorrelationType != "temporal_anomaly" {
				t.Error("Correlation type should be 'temporal_anomaly'")
			}
			if metadata, ok := correlation.Metadata["hour"].(int); !ok || metadata != 2 {
				t.Error("Should report correct hour in metadata")
			}
		}
	}

	if !foundTemporalAnomaly {
		t.Error("Should detect temporal anomaly for night activity")
	}

	t.Log("✅ Temporal anomaly detection validated")
}

// TestFrequencyAnomalyDetection validates detection of unusual event frequencies
func TestFrequencyAnomalyDetection(t *testing.T) {
	engine, err := NewSecurityCorrelationEngine()
	if err != nil {
		t.Fatalf("Failed to create correlation engine: %v", err)
	}
	defer engine.Close()

	// Set baseline metrics
	engine.ruleEngine.baselineMetrics.CommonEventTypes["test_event"] = 10

	// Create events with 3x frequency
	highFreqEvents := make([]SecurityEvent, 30)
	for i := range highFreqEvents {
		highFreqEvents[i] = SecurityEvent{
			EventType: "test_event",
			Success:   true,
			Timestamp: time.Now(),
		}
	}

	correlations := engine.detectFrequencyAnomalies(highFreqEvents)

	// Should detect frequency anomaly
	foundFreqAnomaly := false
	for _, correlation := range correlations {
		if correlation.Pattern == "unusual_event_frequency" {
			foundFreqAnomaly = true
			if correlation.CorrelationType != "frequency_anomaly" {
				t.Error("Correlation type should be 'frequency_anomaly'")
			}
			if metadata, ok := correlation.Metadata["frequency_ratio"].(float64); !ok || metadata < 3.0 {
				t.Error("Should report correct frequency ratio in metadata")
			}
		}
	}

	if !foundFreqAnomaly {
		t.Error("Should detect frequency anomaly for high event count")
	}

	t.Log("✅ Frequency anomaly detection validated")
}

// TestBaselineMetricsUpdate validates baseline metrics calculation
func TestBaselineMetricsUpdate(t *testing.T) {
	engine, err := NewSecurityCorrelationEngine()
	if err != nil {
		t.Fatalf("Failed to create correlation engine: %v", err)
	}
	defer engine.Close()

	// Create test events
	events := []SecurityEvent{
		{
			EventType: "test_event_1",
			Success:   true,
			Timestamp: time.Now(),
		},
		{
			EventType: "test_event_1",
			Success:   true,
			Timestamp: time.Now(),
		},
		{
			EventType: "test_event_2",
			Success:   false,
			Timestamp: time.Now(),
		},
	}

	initialTotal := engine.ruleEngine.baselineMetrics.TotalEvents

	engine.updateBaselineMetrics(events)

	// Check updates
	if engine.ruleEngine.baselineMetrics.TotalEvents != initialTotal+3 {
		t.Errorf("Expected total events %d, got %d", initialTotal+3, engine.ruleEngine.baselineMetrics.TotalEvents)
	}

	if engine.ruleEngine.baselineMetrics.CommonEventTypes["test_event_1"] != 2 {
		t.Error("Should track event type frequencies")
	}

	if engine.ruleEngine.baselineMetrics.CommonEventTypes["test_event_2"] != 1 {
		t.Error("Should track event type frequencies")
	}

	if engine.ruleEngine.baselineMetrics.TypicalFailureRate == 0 {
		t.Error("Should calculate failure rate")
	}

	t.Log("✅ Baseline metrics update validated")
}

// TestRiskScoreCalculation validates risk score calculation
func TestRiskScoreCalculation(t *testing.T) {
	engine, err := NewSecurityCorrelationEngine()
	if err != nil {
		t.Fatalf("Failed to create correlation engine: %v", err)
	}
	defer engine.Close()

	rule := CorrelationRule{Threshold: 5}

	// Test with all successful events
	successfulEvents := []SecurityEvent{
		{Success: true}, {Success: true}, {Success: true},
	}
	
	riskScore1 := engine.calculateRiskScore(successfulEvents, rule)

	// Test with all failed events
	failedEvents := []SecurityEvent{
		{Success: false}, {Success: false}, {Success: false},
	}
	
	riskScore2 := engine.calculateRiskScore(failedEvents, rule)

	// Failed events should have higher risk score
	if riskScore2 <= riskScore1 {
		t.Errorf("Failed events should have higher risk score: %d vs %d", riskScore2, riskScore1)
	}

	// Mixed events
	mixedEvents := []SecurityEvent{
		{Success: true}, {Success: false},
	}
	
	riskScore3 := engine.calculateRiskScore(mixedEvents, rule)
	
	if riskScore3 <= riskScore1 || riskScore3 >= riskScore2 {
		t.Error("Mixed events should have risk score between all-success and all-failure")
	}

	t.Log("✅ Risk score calculation validated")
}

// TestConfidenceCalculation validates confidence calculation
func TestConfidenceCalculation(t *testing.T) {
	engine, err := NewSecurityCorrelationEngine()
	if err != nil {
		t.Fatalf("Failed to create correlation engine: %v", err)
	}
	defer engine.Close()

	rule := CorrelationRule{Threshold: 5}

	// Test with threshold events
	thresholdEvents := make([]SecurityEvent, 5)
	confidence1 := engine.calculateConfidence(thresholdEvents, rule)

	// Test with double threshold events
	doubleEvents := make([]SecurityEvent, 10)
	confidence2 := engine.calculateConfidence(doubleEvents, rule)

	// More events should have higher confidence
	if confidence2 <= confidence1 {
		t.Errorf("More events should increase confidence: %f vs %f", confidence2, confidence1)
	}

	// Confidence should not exceed 1.0
	if confidence1 > 1.0 || confidence2 > 1.0 {
		t.Error("Confidence should not exceed 1.0")
	}

	// Confidence should be at least 0.5 for threshold events
	if confidence1 < 0.5 {
		t.Error("Confidence should be at least 0.5 for threshold events")
	}

	t.Log("✅ Confidence calculation validated")
}

// TestCorrelationPruning validates old correlation removal
func TestCorrelationPruning(t *testing.T) {
	engine, err := NewSecurityCorrelationEngine()
	if err != nil {
		t.Fatalf("Failed to create correlation engine: %v", err)
	}
	defer engine.Close()

	// Add old and new correlations
	oldCorrelation := SecurityCorrelation{
		ID:        "old",
		Timestamp: time.Now().Add(-8 * 24 * time.Hour), // 8 days ago
	}
	newCorrelation := SecurityCorrelation{
		ID:        "new",
		Timestamp: time.Now().Add(-1 * time.Hour), // 1 hour ago
	}

	engine.correlations = append(engine.correlations, oldCorrelation, newCorrelation)

	engine.pruneOldCorrelations()

	// Should keep only new correlation
	if len(engine.correlations) != 1 {
		t.Errorf("Expected 1 correlation after pruning, got %d", len(engine.correlations))
	}

	if engine.correlations[0].ID != "new" {
		t.Error("Should keep the new correlation")
	}

	t.Log("✅ Correlation pruning validated")
}

// TestDefaultRulesAndPatterns validates default configuration
func TestDefaultRulesAndPatterns(t *testing.T) {
	rules := getDefaultCorrelationRules()
	patterns := getKnownAttackPatterns()

	if len(rules) == 0 {
		t.Error("Should have default correlation rules")
	}

	if len(patterns) == 0 {
		t.Error("Should have default attack patterns")
	}

	// Validate rule structure
	for _, rule := range rules {
		if rule.ID == "" {
			t.Error("Rule should have ID")
		}
		if rule.Name == "" {
			t.Error("Rule should have name")
		}
		if len(rule.EventTypes) == 0 {
			t.Error("Rule should have event types")
		}
		if rule.Threshold <= 0 {
			t.Error("Rule should have positive threshold")
		}
	}

	// Validate pattern structure
	for _, pattern := range patterns {
		if pattern.Name == "" {
			t.Error("Pattern should have name")
		}
		if len(pattern.EventSequence) == 0 {
			t.Error("Pattern should have event sequence")
		}
		if pattern.RiskScore <= 0 {
			t.Error("Pattern should have positive risk score")
		}
	}

	t.Log("✅ Default rules and patterns validated")
}

// TestComprehensiveAnalysis validates full analysis workflow
func TestComprehensiveAnalysis(t *testing.T) {
	engine, err := NewSecurityCorrelationEngine()
	if err != nil {
		t.Fatalf("Failed to create correlation engine: %v", err)
	}
	defer engine.Close()

	// This test would typically load events from audit logs,
	// but since we're in a test environment, we'll simulate
	// the analysis with an empty event set

	correlations, err := engine.AnalyzeSecurityEvents()
	if err != nil {
		t.Fatalf("Failed to analyze security events: %v", err)
	}

	// Should not error even with no events
	if correlations == nil {
		t.Error("Correlations should not be nil")
	}

	t.Log("✅ Comprehensive analysis validated")
}