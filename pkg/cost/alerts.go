// Package cost provides advanced cost optimization and alerting features
package cost

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// AlertType defines the type of cost alert
type AlertType string

const (
	AlertTypeThreshold    AlertType = "threshold"    // Budget threshold exceeded
	AlertTypeAnomaly      AlertType = "anomaly"      // Unusual spending pattern
	AlertTypeProjection   AlertType = "projection"   // Projected to exceed budget
	AlertTypeTrend        AlertType = "trend"        // Concerning cost trend
	AlertTypeOptimization AlertType = "optimization" // Cost optimization opportunity
)

// AlertSeverity defines the severity of an alert
type AlertSeverity string

const (
	AlertSeverityInfo     AlertSeverity = "info"
	AlertSeverityWarning  AlertSeverity = "warning"
	AlertSeverityCritical AlertSeverity = "critical"
)

// Alert represents a cost alert
type Alert struct {
	ID           string                 `json:"id"`
	Type         AlertType              `json:"type"`
	Severity     AlertSeverity          `json:"severity"`
	ProjectID    string                 `json:"project_id"`
	InstanceID   string                 `json:"instance_id,omitempty"`
	Timestamp    time.Time              `json:"timestamp"`
	Message      string                 `json:"message"`
	Details      map[string]interface{} `json:"details"`
	Acknowledged bool                   `json:"acknowledged"`
	AutoResolved bool                   `json:"auto_resolved"`
	ResolvedAt   *time.Time             `json:"resolved_at,omitempty"`
	Actions      []AlertAction          `json:"actions"`
}

// AlertAction represents an action that can be taken for an alert
type AlertAction struct {
	Type        string     `json:"type"` // hibernate, stop, terminate, notify
	Description string     `json:"description"`
	Automated   bool       `json:"automated"` // Whether action is taken automatically
	ExecutedAt  *time.Time `json:"executed_at,omitempty"`
}

// AlertRule defines a rule for generating alerts
type AlertRule struct {
	ID            string          `json:"id"`
	Name          string          `json:"name"`
	Type          AlertType       `json:"type"`
	Enabled       bool            `json:"enabled"`
	Conditions    AlertConditions `json:"conditions"`
	Actions       []string        `json:"actions"`  // Actions to take when triggered
	Cooldown      time.Duration   `json:"cooldown"` // Minimum time between alerts
	LastTriggered *time.Time      `json:"last_triggered,omitempty"`
}

// AlertConditions defines conditions for triggering an alert
type AlertConditions struct {
	// Threshold conditions
	BudgetPercentage    *float64 `json:"budget_percentage,omitempty"`
	DailyCostThreshold  *float64 `json:"daily_cost_threshold,omitempty"`
	HourlyCostThreshold *float64 `json:"hourly_cost_threshold,omitempty"`

	// Trend conditions
	CostIncreasePercent *float64 `json:"cost_increase_percent,omitempty"`
	TrendWindow         string   `json:"trend_window,omitempty"` // 1h, 24h, 7d, 30d

	// Anomaly conditions
	StandardDeviations *float64 `json:"standard_deviations,omitempty"`
	BaselineWindow     string   `json:"baseline_window,omitempty"`
}

// AlertManager manages cost alerts
type AlertManager struct {
	mu          sync.RWMutex
	alerts      map[string]*Alert
	rules       map[string]*AlertRule
	subscribers []AlertSubscriber
	ctx         context.Context
	cancel      context.CancelFunc
}

// AlertSubscriber receives alert notifications
type AlertSubscriber interface {
	OnAlert(alert *Alert)
}

// NewAlertManager creates a new alert manager
func NewAlertManager() *AlertManager {
	ctx, cancel := context.WithCancel(context.Background())
	return &AlertManager{
		alerts:      make(map[string]*Alert),
		rules:       make(map[string]*AlertRule),
		subscribers: make([]AlertSubscriber, 0),
		ctx:         ctx,
		cancel:      cancel,
	}
}

// Start begins alert monitoring
func (am *AlertManager) Start() {
	go am.monitorAlerts()
}

// Stop stops alert monitoring
func (am *AlertManager) Stop() {
	am.cancel()
}

// monitorAlerts continuously monitors for alert conditions
func (am *AlertManager) monitorAlerts() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-am.ctx.Done():
			return
		case <-ticker.C:
			am.checkAlertRules()
		}
	}
}

// checkAlertRules evaluates all alert rules
func (am *AlertManager) checkAlertRules() {
	am.mu.RLock()
	rules := make([]*AlertRule, 0, len(am.rules))
	for _, rule := range am.rules {
		if rule.Enabled {
			rules = append(rules, rule)
		}
	}
	am.mu.RUnlock()

	for _, rule := range rules {
		// Check if rule is in cooldown
		if rule.LastTriggered != nil {
			if time.Since(*rule.LastTriggered) < rule.Cooldown {
				continue
			}
		}

		// Evaluate rule conditions
		if am.evaluateRule(rule) {
			am.triggerAlert(rule)
		}
	}
}

// evaluateRule checks if a rule's conditions are met
func (am *AlertManager) evaluateRule(rule *AlertRule) bool {
	// This would integrate with actual cost data
	// For now, return false
	return false
}

// triggerAlert creates and sends an alert
func (am *AlertManager) triggerAlert(rule *AlertRule) {
	alert := &Alert{
		ID:        generateAlertID(),
		Type:      rule.Type,
		Severity:  am.determineSeverity(rule),
		Timestamp: time.Now(),
		Message:   fmt.Sprintf("Alert: %s triggered", rule.Name),
		Details:   make(map[string]interface{}),
		Actions:   am.determineActions(rule),
	}

	am.mu.Lock()
	am.alerts[alert.ID] = alert
	rule.LastTriggered = &alert.Timestamp
	am.mu.Unlock()

	// Notify subscribers
	am.notifySubscribers(alert)

	// Execute automated actions
	am.executeAutomatedActions(alert)
}

// determineSeverity determines alert severity based on conditions
func (am *AlertManager) determineSeverity(rule *AlertRule) AlertSeverity {
	conditions := rule.Conditions

	// Critical if budget exceeded by 90% or more
	if conditions.BudgetPercentage != nil && *conditions.BudgetPercentage >= 90 {
		return AlertSeverityCritical
	}

	// Warning if budget exceeded by 75% or more
	if conditions.BudgetPercentage != nil && *conditions.BudgetPercentage >= 75 {
		return AlertSeverityWarning
	}

	return AlertSeverityInfo
}

// determineActions determines what actions to take for an alert
func (am *AlertManager) determineActions(rule *AlertRule) []AlertAction {
	actions := make([]AlertAction, 0)

	for _, actionType := range rule.Actions {
		action := AlertAction{
			Type:        actionType,
			Description: getActionDescription(actionType),
			Automated:   isAutomatedAction(actionType),
		}
		actions = append(actions, action)
	}

	return actions
}

// executeAutomatedActions executes any automated actions for an alert
func (am *AlertManager) executeAutomatedActions(alert *Alert) {
	for i, action := range alert.Actions {
		if action.Automated {
			// Execute the action
			am.executeAction(alert, &action)

			// Update execution time
			now := time.Now()
			alert.Actions[i].ExecutedAt = &now
		}
	}
}

// executeAction executes a specific alert action
func (am *AlertManager) executeAction(alert *Alert, action *AlertAction) error {
	switch action.Type {
	case "hibernate":
		// Trigger instance hibernation
		fmt.Printf("Hibernating instance due to alert: %s\n", alert.ID)
	case "stop":
		// Stop instance
		fmt.Printf("Stopping instance due to alert: %s\n", alert.ID)
	case "notify":
		// Send notification
		fmt.Printf("Sending notification for alert: %s\n", alert.ID)
	}
	return nil
}

// notifySubscribers notifies all subscribers of an alert
func (am *AlertManager) notifySubscribers(alert *Alert) {
	for _, subscriber := range am.subscribers {
		go subscriber.OnAlert(alert)
	}
}

// Subscribe adds an alert subscriber
func (am *AlertManager) Subscribe(subscriber AlertSubscriber) {
	am.mu.Lock()
	defer am.mu.Unlock()
	am.subscribers = append(am.subscribers, subscriber)
}

// AddRule adds a new alert rule
func (am *AlertManager) AddRule(rule *AlertRule) error {
	am.mu.Lock()
	defer am.mu.Unlock()

	if rule.ID == "" {
		rule.ID = generateRuleID()
	}

	am.rules[rule.ID] = rule
	return nil
}

// GetAlerts returns all alerts
func (am *AlertManager) GetAlerts() []*Alert {
	am.mu.RLock()
	defer am.mu.RUnlock()

	alerts := make([]*Alert, 0, len(am.alerts))
	for _, alert := range am.alerts {
		alerts = append(alerts, alert)
	}
	return alerts
}

// GetActiveAlerts returns unresolved alerts
func (am *AlertManager) GetActiveAlerts() []*Alert {
	am.mu.RLock()
	defer am.mu.RUnlock()

	alerts := make([]*Alert, 0)
	for _, alert := range am.alerts {
		if alert.ResolvedAt == nil {
			alerts = append(alerts, alert)
		}
	}
	return alerts
}

// AcknowledgeAlert marks an alert as acknowledged
func (am *AlertManager) AcknowledgeAlert(alertID string) error {
	am.mu.Lock()
	defer am.mu.Unlock()

	alert, exists := am.alerts[alertID]
	if !exists {
		return fmt.Errorf("alert not found: %s", alertID)
	}

	alert.Acknowledged = true
	return nil
}

// ResolveAlert marks an alert as resolved
func (am *AlertManager) ResolveAlert(alertID string) error {
	am.mu.Lock()
	defer am.mu.Unlock()

	alert, exists := am.alerts[alertID]
	if !exists {
		return fmt.Errorf("alert not found: %s", alertID)
	}

	now := time.Now()
	alert.ResolvedAt = &now
	return nil
}

// CreateDefaultRules creates default alert rules
func (am *AlertManager) CreateDefaultRules() {
	// Budget threshold rules
	am.AddRule(&AlertRule{
		Name:    "Budget 75% Warning",
		Type:    AlertTypeThreshold,
		Enabled: true,
		Conditions: AlertConditions{
			BudgetPercentage: &[]float64{75}[0],
		},
		Actions:  []string{"notify"},
		Cooldown: 6 * time.Hour,
	})

	am.AddRule(&AlertRule{
		Name:    "Budget 90% Critical",
		Type:    AlertTypeThreshold,
		Enabled: true,
		Conditions: AlertConditions{
			BudgetPercentage: &[]float64{90}[0],
		},
		Actions:  []string{"notify", "hibernate"},
		Cooldown: 1 * time.Hour,
	})

	// Cost anomaly rule
	am.AddRule(&AlertRule{
		Name:    "Cost Anomaly Detection",
		Type:    AlertTypeAnomaly,
		Enabled: true,
		Conditions: AlertConditions{
			StandardDeviations: &[]float64{2.5}[0],
			BaselineWindow:     "7d",
		},
		Actions:  []string{"notify"},
		Cooldown: 24 * time.Hour,
	})

	// Daily cost threshold
	am.AddRule(&AlertRule{
		Name:    "Daily Cost Threshold",
		Type:    AlertTypeThreshold,
		Enabled: true,
		Conditions: AlertConditions{
			DailyCostThreshold: &[]float64{50.0}[0], // $50/day
		},
		Actions:  []string{"notify"},
		Cooldown: 24 * time.Hour,
	})
}

// Helper functions

func generateAlertID() string {
	return fmt.Sprintf("alert-%d", time.Now().Unix())
}

func generateRuleID() string {
	return fmt.Sprintf("rule-%d", time.Now().Unix())
}

func getActionDescription(actionType string) string {
	descriptions := map[string]string{
		"hibernate": "Hibernate instance to save costs",
		"stop":      "Stop instance",
		"terminate": "Terminate instance",
		"notify":    "Send notification",
	}
	return descriptions[actionType]
}

func isAutomatedAction(actionType string) bool {
	// Only notifications are automated by default
	// Other actions require manual confirmation
	return actionType == "notify"
}
