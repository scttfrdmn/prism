// Package cost provides functional tests for cost optimization and alerting system
package cost

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/scttfrdmn/cloudworkstation/pkg/types"
)

// TestCostOptimizerFunctionalWorkflow validates complete cost optimizer functionality
func TestCostOptimizerFunctionalWorkflow(t *testing.T) {
	optimizer := setupCostOptimizer(t)

	// Test complete cost optimization workflow
	testCostOptimizerCreation(t, optimizer)
	testInstanceAnalysisWorkflow(t, optimizer)
	testProjectAnalysisWorkflow(t, optimizer)
	testOptimizationReportGeneration(t, optimizer)
	testRecommendationPrioritization(t, optimizer)

	t.Log("✅ Cost optimizer functional workflow validated")
}

// setupCostOptimizer creates and configures a cost optimizer for testing
func setupCostOptimizer(t *testing.T) *CostOptimizer {
	optimizer := NewCostOptimizer()
	if optimizer == nil {
		t.Fatal("Failed to create cost optimizer")
	}

	// Verify initial state
	if optimizer.recommendations == nil {
		t.Error("Cost optimizer recommendations map should be initialized")
	}

	return optimizer
}

// testCostOptimizerCreation validates optimizer initialization
func testCostOptimizerCreation(t *testing.T, optimizer *CostOptimizer) {
	// Test recommendations map initialization
	if len(optimizer.recommendations) != 0 {
		t.Error("New optimizer should start with empty recommendations")
	}

	t.Log("Cost optimizer creation validated")
}

// testInstanceAnalysisWorkflow validates instance-level cost analysis
func testInstanceAnalysisWorkflow(t *testing.T, optimizer *CostOptimizer) {
	// Test instances with different optimization opportunities
	testCases := []struct {
		name             string
		instance         *types.Instance
		expectedRecCount int
		expectedTypes    []OptimizationType
	}{
		{
			name: "Underutilized instance",
			instance: &types.Instance{
				ID:                "i-underutilized",
				Name:              "test-underutilized",
				InstanceType:      "c5.2xlarge",
				State:             "running",
				EstimatedCost:     3.00,
				Architecture:      "x86_64",
				ARMCompatible:     true,
				SpotEligible:      true,
				IsSpot:            false,
				IdlePolicyEnabled: false,
				WorkloadType:      "development",
				AlwaysOn:          true,
			},
			expectedRecCount: 5, // rightsize, hibernation, spot, arm, schedule
			expectedTypes: []OptimizationType{
				OptimizationTypeRightSize,
				OptimizationTypeHibernation,
				OptimizationTypeSpot,
				OptimizationTypeArchitecture,
				OptimizationTypeSchedule,
			},
		},
		{
			name: "Development instance with optimization opportunities",
			instance: &types.Instance{
				ID:                "i-development",
				Name:              "test-development",
				InstanceType:      "t3.micro",
				State:             "running",
				EstimatedCost:     0.50,
				Architecture:      "x86_64",
				ARMCompatible:     false,
				SpotEligible:      false,
				IsSpot:            false,
				IdlePolicyEnabled: false,         // No idle policy, so hibernation will be recommended
				WorkloadType:      "development", // Development workload will trigger scheduling
				AlwaysOn:          true,          // Always on will trigger scheduling recommendation
			},
			expectedRecCount: 3, // Right-sizing (low CPU/memory) + Hibernation + scheduling
			expectedTypes:    []OptimizationType{OptimizationTypeRightSize, OptimizationTypeHibernation, OptimizationTypeSchedule},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			recommendations := optimizer.AnalyzeInstance(tc.instance)

			// Validate recommendation count
			if len(recommendations) != tc.expectedRecCount {
				t.Errorf("Expected %d recommendations, got %d", tc.expectedRecCount, len(recommendations))
			}

			// Validate recommendation types
			typeMap := make(map[OptimizationType]bool)
			for _, rec := range recommendations {
				typeMap[rec.Type] = true

				// Validate recommendation structure
				validateRecommendationStructure(t, rec, tc.instance)
			}

			for _, expectedType := range tc.expectedTypes {
				if !typeMap[expectedType] {
					t.Errorf("Expected recommendation type %s not found", expectedType)
				}
			}
		})
	}

	t.Log("Instance analysis workflow validated")
}

// testProjectAnalysisWorkflow validates project-level cost analysis
func testProjectAnalysisWorkflow(t *testing.T, optimizer *CostOptimizer) {
	projectID := "project-test-123"

	// Create test instances for project analysis
	instances := []*types.Instance{
		{
			ID:            "i-project-1",
			Name:          "project-instance-1",
			InstanceType:  "c5.large",
			State:         "running",
			EstimatedCost: 2.00,
			Runtime:       800, // Running for 800 hours (> 720 hours for RI)
			StorageGB:     200,
			StorageUsedGB: 80,
			Architecture:  "x86_64",
			ARMCompatible: true,
			SpotEligible:  true,
			WorkloadType:  "development",
			AlwaysOn:      true,
		},
		{
			ID:            "i-project-2",
			Name:          "project-instance-2",
			InstanceType:  "m5.xlarge",
			State:         "running",
			EstimatedCost: 3.50,
			Runtime:       900, // Running for 900 hours (> 720 hours for RI)
			StorageGB:     500,
			StorageUsedGB: 150,
			Architecture:  "x86_64",
			SpotEligible:  false,
		},
		{
			ID:            "i-project-3",
			Name:          "project-instance-3",
			InstanceType:  "t3.medium",
			State:         "running",
			EstimatedCost: 1.50,
			Runtime:       750, // Long running (> 720 hours for RI)
			StorageGB:     100,
			StorageUsedGB: 90,
		},
	}

	recommendations := optimizer.AnalyzeProject(projectID, instances)

	// Should have instance-level + project-level recommendations
	if len(recommendations) == 0 {
		t.Error("Project analysis should generate recommendations")
	}

	// Check for project-level recommendations
	projectRecTypes := make(map[OptimizationType]bool)
	for _, rec := range recommendations {
		projectRecTypes[rec.Type] = true

		// Validate recommendation belongs to project
		if rec.ProjectID != "" && rec.ProjectID != projectID {
			t.Errorf("Recommendation project ID mismatch: expected %s, got %s", projectID, rec.ProjectID)
		}

		// Validate recommendation structure
		validateRecommendationStructure(t, rec, nil)
	}

	// Should have reserved instance recommendations (3+ long-running instances)
	if !projectRecTypes[OptimizationTypeReserved] {
		t.Error("Project analysis should include reserved instance recommendations")
	}

	// Should have storage optimization recommendations (low utilization)
	if !projectRecTypes[OptimizationTypeStorage] {
		t.Error("Project analysis should include storage optimization recommendations")
	}

	t.Log("Project analysis workflow validated")
}

// testOptimizationReportGeneration validates optimization report generation
func testOptimizationReportGeneration(t *testing.T, optimizer *CostOptimizer) {
	projectID := "project-report-test"

	// Create test instances
	instances := []*types.Instance{
		{
			ID:                "i-report-1",
			Name:              "report-instance-1",
			InstanceType:      "c5.large",
			EstimatedCost:     2.50,
			Runtime:           800,
			StorageGB:         200,
			StorageUsedGB:     60,
			Architecture:      "x86_64",
			ARMCompatible:     true,
			SpotEligible:      true,
			IdlePolicyEnabled: false,
			WorkloadType:      "development",
			AlwaysOn:          true,
		},
		{
			ID:            "i-report-2",
			Name:          "report-instance-2",
			InstanceType:  "m5.2xlarge",
			EstimatedCost: 5.00,
			Runtime:       1000,
			StorageGB:     500,
			StorageUsedGB: 100,
		},
	}

	report := optimizer.GenerateOptimizationReport(projectID, instances)

	// Validate report structure
	if report.ProjectID != projectID {
		t.Errorf("Report project ID mismatch: expected %s, got %s", projectID, report.ProjectID)
	}

	if report.TotalInstances != len(instances) {
		t.Errorf("Report instance count mismatch: expected %d, got %d", len(instances), report.TotalInstances)
	}

	if len(report.Recommendations) == 0 {
		t.Error("Report should contain recommendations")
	}

	if report.TotalSavings <= 0 {
		t.Error("Report should show potential savings")
	}

	if len(report.TopRecommendations) == 0 {
		t.Error("Report should contain top recommendations")
	}

	// Validate top recommendations are subset of all recommendations
	if len(report.TopRecommendations) > len(report.Recommendations) {
		t.Error("Top recommendations cannot exceed total recommendations")
	}

	// Validate summary structure
	if report.Summary == nil {
		t.Error("Report summary should not be nil")
	}

	validateReportSummary(t, report.Summary, report.Recommendations)

	// Validate generation timestamp
	if report.GeneratedAt.IsZero() {
		t.Error("Report generation timestamp should be set")
	}

	if time.Since(report.GeneratedAt) > time.Minute {
		t.Error("Report generation timestamp should be recent")
	}

	t.Log("Optimization report generation validated")
}

// testRecommendationPrioritization validates recommendation sorting and prioritization
func testRecommendationPrioritization(t *testing.T, optimizer *CostOptimizer) {
	// Create recommendations with different priorities and savings
	recommendations := []*Recommendation{
		{
			Type:             OptimizationTypeRightSize,
			Priority:         "high",
			EstimatedSavings: 100.0,
		},
		{
			Type:             OptimizationTypeSpot,
			Priority:         "medium",
			EstimatedSavings: 200.0,
		},
		{
			Type:             OptimizationTypeStorage,
			Priority:         "low",
			EstimatedSavings: 50.0,
		},
		{
			Type:             OptimizationTypeHibernation,
			Priority:         "high",
			EstimatedSavings: 80.0,
		},
		{
			Type:             OptimizationTypeArchitecture,
			Priority:         "medium",
			EstimatedSavings: 30.0,
		},
	}

	// Test sorting
	optimizer.sortRecommendations(recommendations)

	// First should be high priority with highest savings
	if recommendations[0].Priority != "high" {
		t.Error("First recommendation should be high priority")
	}

	if recommendations[0].EstimatedSavings != 100.0 {
		t.Error("First recommendation should have highest savings among high priority")
	}

	// Validate priority ordering
	lastPriorityValue := 999
	for _, rec := range recommendations {
		priorityValue := getPriorityValue(rec.Priority)
		if priorityValue > lastPriorityValue {
			t.Error("Recommendations should be sorted by priority")
		}
		lastPriorityValue = priorityValue
	}

	// Test GetTopRecommendations
	top3 := optimizer.GetTopRecommendations(recommendations, 3)
	if len(top3) != 3 {
		t.Errorf("Expected 3 top recommendations, got %d", len(top3))
	}

	// Test CalculateTotalSavings
	totalSavings := optimizer.CalculateTotalSavings(recommendations)
	expectedTotal := 100.0 + 200.0 + 50.0 + 80.0 + 30.0
	if totalSavings != expectedTotal {
		t.Errorf("Total savings calculation incorrect: expected %f, got %f", expectedTotal, totalSavings)
	}

	t.Log("Recommendation prioritization validated")
}

// TestCostAlertManagerBasicFunctionality validates basic alert manager functionality
func TestCostAlertManagerBasicFunctionality(t *testing.T) {
	manager := NewAlertManager(nil)
	defer manager.Stop()

	// Test basic creation
	if manager == nil {
		t.Fatal("Failed to create alert manager")
	}

	// Test subscriber functionality
	subscriber := &TestAlertSubscriber{}
	manager.Subscribe(subscriber)

	// Test rule addition
	rule := &AlertRule{
		ID:         "test-rule",
		Name:       "Test Rule",
		Type:       AlertTypeThreshold,
		Enabled:    true,
		Conditions: AlertConditions{},
		Actions:    []string{"notify"},
		Cooldown:   time.Minute,
	}

	err := manager.AddRule(rule)
	if err != nil {
		t.Fatalf("Failed to add rule: %v", err)
	}

	// Test alert triggering
	manager.triggerAlert(rule)

	// Allow time for notification
	time.Sleep(20 * time.Millisecond)

	// Verify alert was received
	alerts := subscriber.GetAlerts()
	if len(alerts) != 1 {
		t.Errorf("Expected 1 alert, got %d", len(alerts))
	}

	if len(alerts) > 0 && alerts[0].Type != AlertTypeThreshold {
		t.Error("Alert type mismatch")
	}

	t.Log("✅ Cost alert manager basic functionality validated")
}

// setupAlertManager creates and configures an alert manager for testing
func setupAlertManager(t *testing.T) *AlertManager {
	manager := NewAlertManager(nil)
	if manager == nil {
		t.Fatal("Failed to create alert manager")
	}

	// Verify initial state
	if manager.alerts == nil {
		t.Error("Alert manager alerts map should be initialized")
	}

	if manager.rules == nil {
		t.Error("Alert manager rules map should be initialized")
	}

	if manager.subscribers == nil {
		t.Error("Alert manager subscribers should be initialized")
	}

	return manager
}

// testAlertManagerCreation validates alert manager initialization
func testAlertManagerCreation(t *testing.T, manager *AlertManager) {
	// Test initial empty state
	if len(manager.alerts) != 0 {
		t.Error("New alert manager should start with no alerts")
	}

	if len(manager.rules) != 0 {
		t.Error("New alert manager should start with no rules")
	}

	if len(manager.subscribers) != 0 {
		t.Error("New alert manager should start with no subscribers")
	}

	// Test context setup
	if manager.ctx == nil {
		t.Error("Alert manager context should be initialized")
	}

	t.Log("Alert manager creation validated")
}

// testAlertRuleManagement validates alert rule management
func testAlertRuleManagement(t *testing.T, manager *AlertManager) {
	// Test adding custom rule
	customRule := &AlertRule{
		Name:    "Test Budget Alert",
		Type:    AlertTypeThreshold,
		Enabled: true,
		Conditions: AlertConditions{
			BudgetPercentage: &[]float64{80.0}[0],
		},
		Actions:  []string{"notify", "hibernate"},
		Cooldown: 2 * time.Hour,
	}

	err := manager.AddRule(customRule)
	if err != nil {
		t.Errorf("Failed to add custom rule: %v", err)
	}

	// Verify rule was added
	if len(manager.rules) != 1 {
		t.Error("Rule should be added to manager")
	}

	// Verify rule ID was generated
	if customRule.ID == "" {
		t.Error("Rule ID should be generated if not provided")
	}

	// Test creating default rules
	manager.CreateDefaultRules()

	// Should have custom rule + default rules (at least 2 total)
	if len(manager.rules) < 2 {
		t.Error("Should have custom rule plus some default rules")
	}

	// Validate that default rules exist and are properly configured
	hasEnabledRules := false
	for _, rule := range manager.rules {
		if rule.Enabled && rule.Cooldown > 0 && len(rule.Actions) > 0 {
			hasEnabledRules = true
			break
		}
	}
	if !hasEnabledRules {
		t.Error("Should have at least one properly configured enabled rule")
	}

	t.Log("Alert rule management validated")
}

// testAlertGeneration validates alert generation workflow
func testAlertGeneration(t *testing.T, manager *AlertManager) {
	// Create a test rule for manual triggering
	testRule := &AlertRule{
		ID:      "test-trigger-rule",
		Name:    "Test Trigger Rule",
		Type:    AlertTypeThreshold,
		Enabled: true,
		Conditions: AlertConditions{
			BudgetPercentage: &[]float64{85.0}[0],
		},
		Actions:  []string{"notify"},
		Cooldown: 1 * time.Minute,
	}

	manager.AddRule(testRule)

	// Manually trigger alert for testing
	manager.triggerAlert(testRule)

	// Verify alert was created
	alerts := manager.GetAlerts()
	if len(alerts) != 1 {
		t.Error("Alert should be generated")
	}

	alert := alerts[0]
	validateAlertStructure(t, alert, testRule)

	// Test cooldown behavior
	initialTriggerTime := testRule.LastTriggered
	if initialTriggerTime == nil {
		t.Error("Rule should have last triggered time set")
	}

	t.Log("Alert generation validated")
}

// testAlertManagement validates alert state management
func testAlertManagement(t *testing.T, manager *AlertManager) {
	// Create test alert
	testRule := &AlertRule{
		Name: "Test Management Rule",
		Type: AlertTypeThreshold,
	}

	manager.triggerAlert(testRule)
	alerts := manager.GetAlerts()
	if len(alerts) == 0 {
		t.Fatal("Need at least one alert for management testing")
	}

	alert := alerts[len(alerts)-1] // Get the last alert

	// Test acknowledging alert
	err := manager.AcknowledgeAlert(alert.ID)
	if err != nil {
		t.Errorf("Failed to acknowledge alert: %v", err)
	}

	if !alert.Acknowledged {
		t.Error("Alert should be marked as acknowledged")
	}

	// Test resolving alert
	err = manager.ResolveAlert(alert.ID)
	if err != nil {
		t.Errorf("Failed to resolve alert: %v", err)
	}

	if alert.ResolvedAt == nil {
		t.Error("Alert should have resolution timestamp")
	}

	// Test GetActiveAlerts
	activeAlerts := manager.GetActiveAlerts()
	for _, activeAlert := range activeAlerts {
		if activeAlert.ID == alert.ID {
			t.Error("Resolved alert should not appear in active alerts")
		}
	}

	// Test error handling for non-existent alert
	err = manager.AcknowledgeAlert("non-existent")
	if err == nil {
		t.Error("Should return error for non-existent alert")
	}

	t.Log("Alert management validated")
}

// testAlertSubscriptions validates alert subscription system
func testAlertSubscriptions(t *testing.T, manager *AlertManager) {
	// Create test subscriber
	subscriber := &TestAlertSubscriber{
		alerts: make([]*Alert, 0),
	}

	manager.Subscribe(subscriber)

	// Verify subscriber was added
	if len(manager.subscribers) != 1 {
		t.Error("Subscriber should be added")
	}

	// Trigger alert and verify subscriber receives it
	testRule := &AlertRule{
		Name: "Subscription Test Rule",
		Type: AlertTypeThreshold,
	}

	manager.triggerAlert(testRule)

	// Give time for async notification
	time.Sleep(10 * time.Millisecond)

	alerts := subscriber.GetAlerts()
	if len(alerts) != 1 {
		t.Error("Subscriber should receive alert notification")
	}

	receivedAlert := alerts[0]
	if receivedAlert.Type != AlertTypeThreshold {
		t.Error("Received alert should match triggered alert")
	}

	t.Log("Alert subscriptions validated")
}

// Helper functions for validation

func validateRecommendationStructure(t *testing.T, rec *Recommendation, instance *types.Instance) {
	if rec.ID == "" {
		t.Error("Recommendation should have ID")
	}

	if rec.Type == "" {
		t.Error("Recommendation should have type")
	}

	if rec.Priority == "" {
		t.Error("Recommendation should have priority")
	}

	if rec.Title == "" {
		t.Error("Recommendation should have title")
	}

	if rec.Description == "" {
		t.Error("Recommendation should have description")
	}

	if rec.EstimatedSavings <= 0 {
		t.Error("Recommendation should have positive estimated savings")
	}

	if rec.SavingsPercent <= 0 {
		t.Error("Recommendation should have positive savings percentage")
	}

	if rec.CreatedAt.IsZero() {
		t.Error("Recommendation should have creation timestamp")
	}

	if rec.ExpiresAt.IsZero() {
		t.Error("Recommendation should have expiration timestamp")
	}

	if instance != nil && rec.InstanceID != instance.ID {
		t.Error("Recommendation instance ID should match instance")
	}
}

func validateReportSummary(t *testing.T, summary map[string]interface{}, recommendations []*Recommendation) {
	if summary["total_recommendations"] != len(recommendations) {
		t.Error("Summary should have correct total recommendations count")
	}

	// Count actual priority distribution
	highCount, mediumCount, lowCount := 0, 0, 0
	typeCount := make(map[string]int)

	for _, rec := range recommendations {
		switch rec.Priority {
		case "high":
			highCount++
		case "medium":
			mediumCount++
		case "low":
			lowCount++
		}
		typeCount[string(rec.Type)]++
	}

	if summary["high_priority"] != highCount {
		t.Error("Summary should have correct high priority count")
	}

	if summary["medium_priority"] != mediumCount {
		t.Error("Summary should have correct medium priority count")
	}

	if summary["low_priority"] != lowCount {
		t.Error("Summary should have correct low priority count")
	}

	// Validate type breakdown
	if byType, ok := summary["by_type"].(map[string]int); ok {
		for recType, count := range typeCount {
			if byType[recType] != count {
				t.Errorf("Summary type count mismatch for %s: expected %d, got %d", recType, count, byType[recType])
			}
		}
	} else {
		t.Error("Summary should have by_type breakdown")
	}
}

func validateAlertStructure(t *testing.T, alert *Alert, rule *AlertRule) {
	if alert.ID == "" {
		t.Error("Alert should have ID")
	}

	if alert.Type != rule.Type {
		t.Error("Alert type should match rule type")
	}

	if alert.Severity == "" {
		t.Error("Alert should have severity")
	}

	if alert.Timestamp.IsZero() {
		t.Error("Alert should have timestamp")
	}

	if alert.Message == "" {
		t.Error("Alert should have message")
	}

	if alert.Details == nil {
		t.Error("Alert details should be initialized")
	}

	if len(alert.Actions) != len(rule.Actions) {
		t.Error("Alert should have same number of actions as rule")
	}

	// Validate actions
	for _, action := range alert.Actions {
		if action.Type == "" {
			t.Error("Alert action should have type")
		}

		if action.Description == "" {
			t.Error("Alert action should have description")
		}
	}
}

func getPriorityValue(priority string) int {
	switch priority {
	case "high":
		return 3
	case "medium":
		return 2
	case "low":
		return 1
	default:
		return 0
	}
}

// TestAlertSubscriber implements AlertSubscriber for testing
type TestAlertSubscriber struct {
	mu     sync.Mutex
	alerts []*Alert
}

func (t *TestAlertSubscriber) OnAlert(alert *Alert) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.alerts = append(t.alerts, alert)
}

func (t *TestAlertSubscriber) GetAlerts() []*Alert {
	t.mu.Lock()
	defer t.mu.Unlock()
	// Return a copy to avoid concurrent access issues
	result := make([]*Alert, len(t.alerts))
	copy(result, t.alerts)
	return result
}

// TestCostOptimizationIntegration validates integration between optimizer and alert manager
func TestCostOptimizationIntegration(t *testing.T) {
	optimizer := NewCostOptimizer()
	alertManager := NewAlertManager(nil)
	defer alertManager.Stop()

	// Create test instance with optimization opportunities
	instance := &types.Instance{
		ID:                "i-integration-test",
		Name:              "integration-test-instance",
		InstanceType:      "c5.2xlarge",
		EstimatedCost:     4.00,
		Architecture:      "x86_64",
		ARMCompatible:     true,
		SpotEligible:      true,
		IdlePolicyEnabled: false,
		WorkloadType:      "development",
		AlwaysOn:          true,
	}

	// Analyze instance for recommendations
	recommendations := optimizer.AnalyzeInstance(instance)
	if len(recommendations) == 0 {
		t.Error("Should generate recommendations for test instance")
	}

	// Create alert rules based on optimization opportunities
	for _, rec := range recommendations {
		rule := &AlertRule{
			Name:    fmt.Sprintf("Optimization Alert: %s", rec.Title),
			Type:    AlertTypeOptimization,
			Enabled: true,
			Conditions: AlertConditions{
				// Trigger if potential savings exceed $20/month
				DailyCostThreshold: &[]float64{0.67}[0], // ~$20/month
			},
			Actions:  []string{"notify"},
			Cooldown: 24 * time.Hour,
		}

		if rec.EstimatedSavings > 20.0 {
			alertManager.AddRule(rule)
		}
	}

	// Generate optimization report
	report := optimizer.GenerateOptimizationReport("integration-project", []*types.Instance{instance})

	// Validate integration
	if report.TotalSavings <= 0 {
		t.Error("Integration should show potential savings")
	}

	if len(report.TopRecommendations) == 0 {
		t.Error("Integration should identify top recommendations")
	}

	// Check that high-value recommendations trigger alert rules
	highValueRecs := 0
	for _, rec := range recommendations {
		if rec.EstimatedSavings > 20.0 {
			highValueRecs++
		}
	}

	if highValueRecs > 0 && len(alertManager.rules) == 0 {
		t.Error("High-value recommendations should create alert rules")
	}

	t.Log("✅ Cost optimization integration validated")
}
