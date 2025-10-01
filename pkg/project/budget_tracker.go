package project

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/scttfrdmn/cloudworkstation/pkg/types"
)

// ActionExecutor defines the interface for executing budget auto actions
type ActionExecutor interface {
	// ExecuteHibernateAll hibernates all instances for a project
	ExecuteHibernateAll(projectID string) error
	// ExecuteStopAll stops all instances for a project
	ExecuteStopAll(projectID string) error
	// ExecutePreventLaunch sets a flag to prevent new launches for a project
	ExecutePreventLaunch(projectID string) error
}

// BudgetTracker handles budget tracking and cost analysis for projects
type BudgetTracker struct {
	budgetPath     string
	mutex          sync.RWMutex
	budgetData     map[string]*ProjectBudgetData
	costCalculator *CostCalculator
	actionExecutor ActionExecutor
}

// ProjectBudgetData stores budget tracking data for a project
type ProjectBudgetData struct {
	ProjectID    string               `json:"project_id"`
	Budget       *types.ProjectBudget `json:"budget"`
	CostHistory  []CostDataPoint      `json:"cost_history"`
	AlertHistory []AlertEvent         `json:"alert_history"`
	LastUpdated  time.Time            `json:"last_updated"`
}

// CostDataPoint represents a point-in-time cost measurement
type CostDataPoint struct {
	Timestamp     time.Time            `json:"timestamp"`
	TotalCost     float64              `json:"total_cost"`
	InstanceCosts []types.InstanceCost `json:"instance_costs"`
	StorageCosts  []types.StorageCost  `json:"storage_costs"`
	DailyCost     float64              `json:"daily_cost"`
}

// AlertEvent represents a budget alert event
type AlertEvent struct {
	Timestamp   time.Time             `json:"timestamp"`
	AlertType   types.BudgetAlertType `json:"alert_type"`
	Threshold   float64               `json:"threshold"`
	SpentAmount float64               `json:"spent_amount"`
	Message     string                `json:"message"`
	Resolved    bool                  `json:"resolved"`
}

// NewBudgetTracker creates a new budget tracker
func NewBudgetTracker() (*BudgetTracker, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}

	stateDir := filepath.Join(homeDir, ".cloudworkstation")
	budgetPath := filepath.Join(stateDir, "budget_data.json")

	costCalculator := &CostCalculator{}

	tracker := &BudgetTracker{
		budgetPath:     budgetPath,
		budgetData:     make(map[string]*ProjectBudgetData),
		costCalculator: costCalculator,
		actionExecutor: nil, // Will be set later by daemon
	}

	// Load existing budget data
	if err := tracker.loadBudgetData(); err != nil {
		return nil, fmt.Errorf("failed to load budget data: %w", err)
	}

	return tracker, nil
}

// SetActionExecutor sets the action executor for budget auto actions
func (bt *BudgetTracker) SetActionExecutor(executor ActionExecutor) {
	bt.mutex.Lock()
	defer bt.mutex.Unlock()
	bt.actionExecutor = executor
}

// InitializeProject initializes budget tracking for a new project
func (bt *BudgetTracker) InitializeProject(projectID string, budget *types.ProjectBudget) error {
	bt.mutex.Lock()
	defer bt.mutex.Unlock()

	budgetData := &ProjectBudgetData{
		ProjectID:    projectID,
		Budget:       budget,
		CostHistory:  []CostDataPoint{},
		AlertHistory: []AlertEvent{},
		LastUpdated:  time.Now(),
	}

	bt.budgetData[projectID] = budgetData
	return bt.saveBudgetData()
}

// RemoveProject removes budget tracking for a project
func (bt *BudgetTracker) RemoveProject(projectID string) error {
	bt.mutex.Lock()
	defer bt.mutex.Unlock()

	delete(bt.budgetData, projectID)
	return bt.saveBudgetData()
}

// UpdateProjectCosts updates cost tracking for a project
func (bt *BudgetTracker) UpdateProjectCosts(projectID string, instances []types.Instance, volumes []types.EFSVolume, ebsVolumes []types.EBSVolume) error {
	bt.mutex.Lock()
	defer bt.mutex.Unlock()

	budgetData, exists := bt.budgetData[projectID]
	if !exists {
		return fmt.Errorf("budget data not found for project %q", projectID)
	}

	// Calculate current costs
	instanceCosts, totalInstanceCost := bt.costCalculator.CalculateInstanceCosts(instances)
	storageCosts, totalStorageCost := bt.costCalculator.CalculateStorageCosts(volumes, ebsVolumes)

	totalCost := totalInstanceCost + totalStorageCost

	// Create cost data point
	costPoint := CostDataPoint{
		Timestamp:     time.Now(),
		TotalCost:     totalCost,
		InstanceCosts: instanceCosts,
		StorageCosts:  storageCosts,
		DailyCost:     bt.calculateDailyCost(budgetData.CostHistory, totalCost),
	}

	// Add to history
	budgetData.CostHistory = append(budgetData.CostHistory, costPoint)

	// Keep only last 90 days of history
	cutoffTime := time.Now().AddDate(0, 0, -90)
	bt.trimCostHistory(budgetData, cutoffTime)

	// Update budget spent amount
	if budgetData.Budget != nil {
		budgetData.Budget.SpentAmount = totalCost
		budgetData.Budget.LastUpdated = time.Now()
	}

	budgetData.LastUpdated = time.Now()

	// Check for budget alerts
	if err := bt.checkBudgetAlerts(projectID, budgetData); err != nil {
		return fmt.Errorf("failed to check budget alerts: %w", err)
	}

	return bt.saveBudgetData()
}

// GetCostBreakdown retrieves detailed cost analysis for a project
func (bt *BudgetTracker) GetCostBreakdown(projectID string, startDate, endDate time.Time) (*types.ProjectCostBreakdown, error) {
	bt.mutex.RLock()
	defer bt.mutex.RUnlock()

	budgetData, exists := bt.budgetData[projectID]
	if !exists {
		return nil, fmt.Errorf("budget data not found for project %q", projectID)
	}

	// Find cost data points within the date range
	var relevantPoints []CostDataPoint
	for _, point := range budgetData.CostHistory {
		if (point.Timestamp.After(startDate) || point.Timestamp.Equal(startDate)) &&
			(point.Timestamp.Before(endDate) || point.Timestamp.Equal(endDate)) {
			relevantPoints = append(relevantPoints, point)
		}
	}

	if len(relevantPoints) == 0 {
		return &types.ProjectCostBreakdown{
			ProjectID:     projectID,
			TotalCost:     0.0,
			InstanceCosts: []types.InstanceCost{},
			StorageCosts:  []types.StorageCost{},
			PeriodStart:   startDate,
			PeriodEnd:     endDate,
			GeneratedAt:   time.Now(),
		}, nil
	}

	// Get the latest point for current costs
	latestPoint := relevantPoints[len(relevantPoints)-1]

	return &types.ProjectCostBreakdown{
		ProjectID:     projectID,
		TotalCost:     latestPoint.TotalCost,
		InstanceCosts: latestPoint.InstanceCosts,
		StorageCosts:  latestPoint.StorageCosts,
		PeriodStart:   startDate,
		PeriodEnd:     endDate,
		GeneratedAt:   time.Now(),
	}, nil
}

// GetResourceUsage retrieves resource utilization metrics for a project
func (bt *BudgetTracker) GetResourceUsage(projectID string, period time.Duration) (*types.ProjectResourceUsage, error) {
	bt.mutex.RLock()
	defer bt.mutex.RUnlock()

	budgetData, exists := bt.budgetData[projectID]
	if !exists {
		return nil, fmt.Errorf("budget data not found for project %q", projectID)
	}

	// Calculate metrics from cost history
	startTime := time.Now().Add(-period)

	var totalComputeHours float64
	var totalStorage float64
	var idleSavings float64
	activeInstances := 0
	totalInstances := 0

	// Find latest cost data point
	var latestPoint *CostDataPoint
	for i := len(budgetData.CostHistory) - 1; i >= 0; i-- {
		if budgetData.CostHistory[i].Timestamp.After(startTime) {
			latestPoint = &budgetData.CostHistory[i]
			break
		}
	}

	if latestPoint != nil {
		for _, instanceCost := range latestPoint.InstanceCosts {
			totalComputeHours += instanceCost.RunningHours
			idleSavings += bt.calculateIdleSavings(instanceCost)
			totalInstances++
			if instanceCost.RunningHours > 0 {
				activeInstances++
			}
		}

		for _, storageCost := range latestPoint.StorageCosts {
			totalStorage += storageCost.SizeGB
		}
	}

	return &types.ProjectResourceUsage{
		ProjectID:         projectID,
		ActiveInstances:   activeInstances,
		TotalInstances:    totalInstances,
		TotalStorage:      totalStorage,
		ComputeHours:      totalComputeHours,
		IdleSavings:       idleSavings,
		MeasurementPeriod: period,
		LastUpdated:       time.Now(),
	}, nil
}

// UpdateProjectSpending updates project spending with instance and storage costs
func (bt *BudgetTracker) UpdateProjectSpending(projectID string, instanceCosts []types.InstanceCost, storageCosts []types.StorageCost) error {
	bt.mutex.Lock()
	defer bt.mutex.Unlock()

	budgetData, exists := bt.budgetData[projectID]
	if !exists {
		return fmt.Errorf("budget data not found for project %q", projectID)
	}

	// Calculate total costs
	var totalInstanceCost, totalStorageCost float64

	for _, instanceCost := range instanceCosts {
		totalInstanceCost += instanceCost.TotalCost
	}

	for _, storageCost := range storageCosts {
		totalStorageCost += storageCost.Cost
	}

	dailyTotalCost := totalInstanceCost + totalStorageCost

	// Add to previous spending (cumulative)
	previousSpent := budgetData.Budget.SpentAmount
	newTotalSpent := previousSpent + dailyTotalCost

	// Create cost data point
	costPoint := CostDataPoint{
		Timestamp:     time.Now(),
		TotalCost:     newTotalSpent,
		InstanceCosts: instanceCosts,
		StorageCosts:  storageCosts,
		DailyCost:     dailyTotalCost,
	}

	// Add to history
	budgetData.CostHistory = append(budgetData.CostHistory, costPoint)

	// Keep only last 90 days of history
	cutoffTime := time.Now().AddDate(0, 0, -90)
	bt.trimCostHistory(budgetData, cutoffTime)

	// Update budget spent amount (cumulative)
	if budgetData.Budget != nil {
		budgetData.Budget.SpentAmount = newTotalSpent
		budgetData.Budget.LastUpdated = time.Now()
	}

	budgetData.LastUpdated = time.Now()

	// Check for budget alerts
	if err := bt.checkBudgetAlerts(projectID, budgetData); err != nil {
		return fmt.Errorf("failed to check budget alerts: %w", err)
	}

	return bt.saveBudgetData()
}

// CheckBudgetStatus checks the current budget status and returns detailed information
func (bt *BudgetTracker) CheckBudgetStatus(projectID string) (*BudgetStatus, error) {
	bt.mutex.RLock()
	defer bt.mutex.RUnlock()

	budgetData, exists := bt.budgetData[projectID]
	if !exists {
		return nil, fmt.Errorf("budget data not found for project %q", projectID)
	}

	if budgetData.Budget == nil {
		return &BudgetStatus{
			ProjectID:     projectID,
			BudgetEnabled: false,
		}, nil
	}

	budget := budgetData.Budget
	spentPercentage := 0.0
	if budget.TotalBudget > 0 {
		spentPercentage = budget.SpentAmount / budget.TotalBudget
	}

	remainingBudget := budget.TotalBudget - budget.SpentAmount
	if remainingBudget < 0 {
		remainingBudget = 0
	}

	// Calculate projected monthly spend
	projectedMonthlySpend := bt.calculateProjectedMonthlySpend(budgetData.CostHistory)

	// Calculate days until budget exhausted
	var daysUntilExhausted *int
	if projectedMonthlySpend > 0 && remainingBudget > 0 {
		monthsRemaining := remainingBudget / projectedMonthlySpend
		daysRemaining := int(monthsRemaining * 30)
		daysUntilExhausted = &daysRemaining
	}

	// Get active alerts
	activeAlerts := bt.getActiveAlerts(budgetData)
	triggeredActions := bt.getTriggeredActions(budgetData)

	return &BudgetStatus{
		ProjectID:                projectID,
		BudgetEnabled:            true,
		TotalBudget:              budget.TotalBudget,
		SpentAmount:              budget.SpentAmount,
		RemainingBudget:          remainingBudget,
		SpentPercentage:          spentPercentage,
		ProjectedMonthlySpend:    projectedMonthlySpend,
		DaysUntilBudgetExhausted: daysUntilExhausted,
		ActiveAlerts:             activeAlerts,
		TriggeredActions:         triggeredActions,
		LastUpdated:              time.Now(),
	}, nil
}

// Helper methods

func (bt *BudgetTracker) calculateDailyCost(costHistory []CostDataPoint, currentCost float64) float64 {
	if len(costHistory) == 0 {
		return 0.0
	}

	// Find cost from 24 hours ago
	yesterday := time.Now().AddDate(0, 0, -1)
	var yesterdayCost float64
	for i := len(costHistory) - 1; i >= 0; i-- {
		if costHistory[i].Timestamp.Before(yesterday) {
			yesterdayCost = costHistory[i].TotalCost
			break
		}
	}

	return currentCost - yesterdayCost
}

func (bt *BudgetTracker) calculateProjectedMonthlySpend(costHistory []CostDataPoint) float64 {
	if len(costHistory) < 2 {
		return 0.0
	}

	// Calculate average daily spend over last 7 days
	weekAgo := time.Now().AddDate(0, 0, -7)
	var recentPoints []CostDataPoint

	for _, point := range costHistory {
		if point.Timestamp.After(weekAgo) {
			recentPoints = append(recentPoints, point)
		}
	}

	if len(recentPoints) < 2 {
		return 0.0
	}

	// Calculate daily average
	totalDailyCost := 0.0
	validPoints := 0

	for _, point := range recentPoints {
		if point.DailyCost > 0 {
			totalDailyCost += point.DailyCost
			validPoints++
		}
	}

	if validPoints == 0 {
		return 0.0
	}

	avgDailyCost := totalDailyCost / float64(validPoints)
	return avgDailyCost * 30 // Project monthly spend
}

func (bt *BudgetTracker) calculateIdleSavings(instanceCost types.InstanceCost) float64 {
	// Hibernation saves compute costs but not storage costs
	// Estimate savings based on hibernated hours vs running hours
	if instanceCost.HibernatedHours > 0 {
		totalHours := instanceCost.RunningHours + instanceCost.HibernatedHours + instanceCost.StoppedHours
		if totalHours > 0 {
			hibernationRatio := instanceCost.HibernatedHours / totalHours
			return instanceCost.ComputeCost * hibernationRatio
		}
	}
	return 0.0
}

func (bt *BudgetTracker) trimCostHistory(budgetData *ProjectBudgetData, cutoffTime time.Time) {
	var trimmedHistory []CostDataPoint
	for _, point := range budgetData.CostHistory {
		if point.Timestamp.After(cutoffTime) {
			trimmedHistory = append(trimmedHistory, point)
		}
	}
	budgetData.CostHistory = trimmedHistory
}

func (bt *BudgetTracker) checkBudgetAlerts(projectID string, budgetData *ProjectBudgetData) error {
	if budgetData.Budget == nil {
		return nil
	}

	budget := budgetData.Budget
	spentPercentage := 0.0
	if budget.TotalBudget > 0 {
		spentPercentage = budget.SpentAmount / budget.TotalBudget
	}

	// Check alert thresholds - only trigger each threshold once
	for _, alert := range budget.AlertThresholds {
		if spentPercentage >= alert.Threshold {
			// Check if we've already sent this specific alert threshold ever
			alreadyTriggered := false
			for _, event := range budgetData.AlertHistory {
				if event.Threshold == alert.Threshold && event.AlertType == alert.Type {
					alreadyTriggered = true
					break
				}
			}

			if !alreadyTriggered {
				alertEvent := AlertEvent{
					Timestamp:   time.Now(),
					AlertType:   alert.Type,
					Threshold:   alert.Threshold,
					SpentAmount: budget.SpentAmount,
					Message:     fmt.Sprintf("Budget alert: %.1f%% of budget spent", alert.Threshold*100),
					Resolved:    false,
				}

				budgetData.AlertHistory = append(budgetData.AlertHistory, alertEvent)

				// Deliver alert based on alert type
				if err := bt.deliverAlert(projectID, alertEvent, alert.Recipients); err != nil {
					// Don't fail budget processing for alert delivery errors
					fmt.Printf("Failed to deliver budget alert: %v\n", err)
					// Still log the alert even if delivery fails
					if logErr := bt.logAlert(projectID, alertEvent); logErr != nil {
						fmt.Printf("Failed to log budget alert after delivery failure: %v\n", logErr)
					}
				}
			}
		}
	}

	// Check auto actions - only trigger each action once
	for _, action := range budget.AutoActions {
		if spentPercentage >= action.Threshold {
			// Check if we've already triggered this specific action threshold ever
			alreadyTriggered := false
			actionAlertType := types.BudgetAlertType(fmt.Sprintf("action_%s", action.Action))
			for _, event := range budgetData.AlertHistory {
				if event.Threshold == action.Threshold && event.AlertType == actionAlertType {
					alreadyTriggered = true
					break
				}
			}

			if !alreadyTriggered {
				// Execute the auto action if executor is available
				var actionErr error
				actionMessage := fmt.Sprintf("Auto action triggered: %s at %.1f%% budget", action.Action, action.Threshold*100)

				if bt.actionExecutor != nil {
					switch action.Action {
					case types.BudgetActionHibernateAll:
						actionErr = bt.actionExecutor.ExecuteHibernateAll(projectID)
						if actionErr != nil {
							actionMessage = fmt.Sprintf("Auto action failed: hibernate_all at %.1f%% budget - %v", action.Threshold*100, actionErr)
						} else {
							actionMessage = fmt.Sprintf("Auto action executed: hibernated all instances at %.1f%% budget", action.Threshold*100)
						}
					case types.BudgetActionStopAll:
						actionErr = bt.actionExecutor.ExecuteStopAll(projectID)
						if actionErr != nil {
							actionMessage = fmt.Sprintf("Auto action failed: stop_all at %.1f%% budget - %v", action.Threshold*100, actionErr)
						} else {
							actionMessage = fmt.Sprintf("Auto action executed: stopped all instances at %.1f%% budget", action.Threshold*100)
						}
					case types.BudgetActionPreventLaunch:
						actionErr = bt.actionExecutor.ExecutePreventLaunch(projectID)
						if actionErr != nil {
							actionMessage = fmt.Sprintf("Auto action failed: prevent_launch at %.1f%% budget - %v", action.Threshold*100, actionErr)
						} else {
							actionMessage = fmt.Sprintf("Auto action executed: prevented new launches at %.1f%% budget", action.Threshold*100)
						}
					case types.BudgetActionNotifyOnly:
						// No action to execute, just notification
						actionMessage = fmt.Sprintf("Budget notification: %.1f%% budget reached", action.Threshold*100)
					default:
						actionErr = fmt.Errorf("unknown action type: %s", action.Action)
						actionMessage = fmt.Sprintf("Auto action failed: unknown action %s at %.1f%% budget", action.Action, action.Threshold*100)
					}
				} else {
					// No executor available - log warning but still record the event
					fmt.Printf("Warning: Budget auto action executor not set, skipping action %s for project %s\n", action.Action, projectID)
					actionMessage = fmt.Sprintf("Auto action skipped: %s at %.1f%% budget (no executor)", action.Action, action.Threshold*100)
				}

				// Record the action attempt in alert history
				actionEvent := AlertEvent{
					Timestamp:   time.Now(),
					AlertType:   actionAlertType,
					Threshold:   action.Threshold,
					SpentAmount: budget.SpentAmount,
					Message:     actionMessage,
					Resolved:    actionErr == nil, // Mark resolved if action succeeded
				}
				budgetData.AlertHistory = append(budgetData.AlertHistory, actionEvent)
			}
		}
	}

	return nil
}

func (bt *BudgetTracker) getActiveAlerts(budgetData *ProjectBudgetData) []string {
	var activeAlerts []string

	// Get alerts from last 24 hours - only actual alerts, not actions
	dayAgo := time.Now().AddDate(0, 0, -1)

	for _, event := range budgetData.AlertHistory {
		if event.Timestamp.After(dayAgo) && !event.Resolved {
			// Only include actual alert types (email, slack, webhook), not actions
			if event.AlertType == types.BudgetAlertEmail || event.AlertType == types.BudgetAlertSlack || event.AlertType == types.BudgetAlertWebhook {
				activeAlerts = append(activeAlerts, event.Message)
			}
		}
	}

	return activeAlerts
}

func (bt *BudgetTracker) getTriggeredActions(budgetData *ProjectBudgetData) []string {
	var triggeredActions []string

	// Get actions from last 24 hours
	dayAgo := time.Now().AddDate(0, 0, -1)

	for _, event := range budgetData.AlertHistory {
		if event.Timestamp.After(dayAgo) && !event.Resolved {
			// Check if this is an action event (not a regular alert)
			if event.AlertType != types.BudgetAlertEmail && event.AlertType != types.BudgetAlertSlack && event.AlertType != types.BudgetAlertWebhook {
				triggeredActions = append(triggeredActions, event.Message)
			}
		}
	}

	return triggeredActions
}

func (bt *BudgetTracker) loadBudgetData() error {
	// Check if budget data file exists
	if _, err := os.Stat(bt.budgetPath); os.IsNotExist(err) {
		// No budget data file exists yet, start with empty map
		return nil
	}

	data, err := os.ReadFile(bt.budgetPath)
	if err != nil {
		return fmt.Errorf("failed to read budget data file: %w", err)
	}

	var budgetData map[string]*ProjectBudgetData
	if err := json.Unmarshal(data, &budgetData); err != nil {
		return fmt.Errorf("failed to parse budget data file: %w", err)
	}

	bt.budgetData = budgetData
	return nil
}

func (bt *BudgetTracker) saveBudgetData() error {
	data, err := json.MarshalIndent(bt.budgetData, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal budget data: %w", err)
	}

	// Write to temporary file first, then rename for atomicity
	tempPath := bt.budgetPath + ".tmp"
	if err := os.WriteFile(tempPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write temporary budget data file: %w", err)
	}

	if err := os.Rename(tempPath, bt.budgetPath); err != nil {
		return fmt.Errorf("failed to rename budget data file: %w", err)
	}

	return nil
}

// GetCostTrends returns cost trends for a project over a specified period
func (bt *BudgetTracker) GetCostTrends(projectID string, period string) (map[string]interface{}, error) {
	bt.mutex.RLock()
	defer bt.mutex.RUnlock()

	data, exists := bt.budgetData[projectID]
	if !exists {
		return nil, fmt.Errorf("project not found: %s", projectID)
	}

	// Parse period (7d, 30d, 90d)
	days := 30
	switch period {
	case "7d":
		days = 7
	case "90d":
		days = 90
	}

	// Filter cost history by period
	cutoff := time.Now().AddDate(0, 0, -days)
	trends := make([]CostDataPoint, 0)

	for _, point := range data.CostHistory {
		if point.Timestamp.After(cutoff) {
			trends = append(trends, point)
		}
	}

	return map[string]interface{}{
		"project_id": projectID,
		"period":     period,
		"days":       days,
		"trends":     trends,
		"count":      len(trends),
	}, nil
}

// GetBudgetStatus returns the current budget status for a project
func (bt *BudgetTracker) GetBudgetStatus(projectID string) (map[string]interface{}, error) {
	bt.mutex.RLock()
	defer bt.mutex.RUnlock()

	data, exists := bt.budgetData[projectID]
	if !exists {
		return nil, fmt.Errorf("project not found: %s", projectID)
	}

	// Calculate current spend
	currentSpend := 0.0
	if len(data.CostHistory) > 0 {
		currentSpend = data.CostHistory[len(data.CostHistory)-1].TotalCost
	}

	// Calculate budget usage
	budgetLimit := 0.0
	if data.Budget != nil && data.Budget.MonthlyLimit != nil {
		budgetLimit = *data.Budget.MonthlyLimit
	}

	usagePercent := 0.0
	if budgetLimit > 0 {
		usagePercent = (currentSpend / budgetLimit) * 100
	}

	return map[string]interface{}{
		"project_id":     projectID,
		"budget_limit":   budgetLimit,
		"current_spend":  currentSpend,
		"usage_percent":  usagePercent,
		"budget":         data.Budget,
		"last_updated":   data.LastUpdated,
		"alerts_enabled": data.Budget != nil && len(data.Budget.AlertThresholds) > 0,
		"recent_alerts":  len(data.AlertHistory),
	}, nil
}

// Close cleanly shuts down the budget tracker
func (bt *BudgetTracker) Close() error {
	bt.mutex.Lock()
	defer bt.mutex.Unlock()

	// Save any pending data
	return bt.saveBudgetData()
}

// deliverAlert delivers budget alerts via configured delivery methods
func (bt *BudgetTracker) deliverAlert(projectID string, alertEvent AlertEvent, recipients []string) error {
	switch alertEvent.AlertType {
	case types.BudgetAlertEmail:
		return bt.sendEmailAlert(projectID, alertEvent, recipients)
	case types.BudgetAlertSlack:
		return bt.sendSlackAlert(projectID, alertEvent, recipients)
	case types.BudgetAlertWebhook:
		return bt.sendWebhookAlert(projectID, alertEvent, recipients)
	default:
		// Fallback to logging for unknown alert types
		return bt.logAlert(projectID, alertEvent)
	}
}

// sendEmailAlert sends budget alert via email
func (bt *BudgetTracker) sendEmailAlert(projectID string, alertEvent AlertEvent, recipients []string) error {
	// For production deployment, integrate with email service (AWS SES, SendGrid, etc.)
	// For now, log the email that would be sent
	fmt.Printf("📧 EMAIL ALERT to %v: [%s] Project %s budget alert - $%.2f spent (%.1f%% threshold)\n",
		recipients,
		alertEvent.Timestamp.Format("2006-01-02 15:04:05"),
		projectID,
		alertEvent.SpentAmount,
		alertEvent.Threshold*100)
	return nil
}

// sendSlackAlert sends budget alert to Slack
func (bt *BudgetTracker) sendSlackAlert(projectID string, alertEvent AlertEvent, recipients []string) error {
	// For production deployment, integrate with Slack API
	// For now, log the Slack message that would be sent
	fmt.Printf("💬 SLACK ALERT to %v: 🚨 Budget Alert - Project %s has spent $%.2f (%.1f%% threshold reached)\n",
		recipients,
		projectID,
		alertEvent.SpentAmount,
		alertEvent.Threshold*100)
	return nil
}

// sendWebhookAlert sends budget alert via webhook
func (bt *BudgetTracker) sendWebhookAlert(projectID string, alertEvent AlertEvent, recipients []string) error {
	// For production deployment, send HTTP POST to webhook URLs in recipients
	// For now, log the webhook payload that would be sent
	fmt.Printf("🔗 WEBHOOK ALERT to %v: POST {\"project\": \"%s\", \"spent\": %.2f, \"threshold\": %.1f, \"timestamp\": \"%s\"}\n",
		recipients,
		projectID,
		alertEvent.SpentAmount,
		alertEvent.Threshold*100,
		alertEvent.Timestamp.Format(time.RFC3339))
	return nil
}

// logAlert logs a budget alert event (fallback method)
func (bt *BudgetTracker) logAlert(projectID string, alertEvent AlertEvent) error {
	fmt.Printf("🚨 BUDGET ALERT [%s] Project: %s, Type: %s, Current: $%.2f, Threshold: %.1f%%\n",
		alertEvent.Timestamp.Format("2006-01-02 15:04:05"),
		projectID,
		alertEvent.AlertType,
		alertEvent.SpentAmount,
		alertEvent.Threshold*100)
	return nil
}
