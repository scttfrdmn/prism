package daemon

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/scttfrdmn/cloudworkstation/pkg/cost"
	"github.com/scttfrdmn/cloudworkstation/pkg/types"
)

// Cost optimization and alert handlers

// handleGetCostAlerts returns all cost alerts
func (s *Server) handleGetCostAlerts(w http.ResponseWriter, r *http.Request) {
	// Get project ID from query params if provided
	projectID := r.URL.Query().Get("project_id")
	
	alerts := s.alertManager.GetAlerts()
	
	// Filter by project if specified
	if projectID != "" {
		filtered := make([]*cost.Alert, 0)
		for _, alert := range alerts {
			if alert.ProjectID == projectID {
				filtered = append(filtered, alert)
			}
		}
		alerts = filtered
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"alerts": alerts,
		"count":  len(alerts),
	})
}

// handleGetActiveAlerts returns only active (unresolved) alerts
func (s *Server) handleGetActiveAlerts(w http.ResponseWriter, r *http.Request) {
	alerts := s.alertManager.GetActiveAlerts()
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"alerts": alerts,
		"count":  len(alerts),
	})
}

// handleAcknowledgeAlert marks an alert as acknowledged
func (s *Server) handleAcknowledgeAlert(w http.ResponseWriter, r *http.Request) {
	// Extract alert ID from path
	path := strings.TrimPrefix(r.URL.Path, "/api/v1/cost/alerts/")
	parts := strings.Split(path, "/")
	if len(parts) < 2 || parts[1] != "acknowledge" {
		http.Error(w, "Invalid path", http.StatusBadRequest)
		return
	}
	alertID := parts[0]
	
	if err := s.alertManager.AcknowledgeAlert(alertID); err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status":   "success",
		"alert_id": alertID,
	})
}

// handleResolveAlert marks an alert as resolved
func (s *Server) handleResolveAlert(w http.ResponseWriter, r *http.Request) {
	// Extract alert ID from path
	path := strings.TrimPrefix(r.URL.Path, "/api/v1/cost/alerts/")
	parts := strings.Split(path, "/")
	if len(parts) < 2 || parts[1] != "resolve" {
		http.Error(w, "Invalid path", http.StatusBadRequest)
		return
	}
	alertID := parts[0]
	
	if err := s.alertManager.ResolveAlert(alertID); err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status":   "success",
		"alert_id": alertID,
	})
}

// handleAddAlertRule adds a new alert rule
func (s *Server) handleAddAlertRule(w http.ResponseWriter, r *http.Request) {
	var rule cost.AlertRule
	if err := json.NewDecoder(r.Body).Decode(&rule); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	
	if err := s.alertManager.AddRule(&rule); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "success",
		"rule_id": rule.ID,
	})
}

// handleGetOptimizationReport generates an optimization report for a project
func (s *Server) handleGetOptimizationReport(w http.ResponseWriter, r *http.Request) {
	projectID := r.URL.Query().Get("project_id")
	if projectID == "" {
		http.Error(w, "project_id is required", http.StatusBadRequest)
		return
	}
	
	// Get instances for the project
	instances, err := s.getProjectInstances(projectID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	
	// Generate optimization report
	optimizer := cost.NewCostOptimizer()
	report := optimizer.GenerateOptimizationReport(projectID, instances)
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(report)
}

// handleGetRecommendations returns optimization recommendations
func (s *Server) handleGetRecommendations(w http.ResponseWriter, r *http.Request) {
	instanceID := r.URL.Query().Get("instance_id")
	projectID := r.URL.Query().Get("project_id")
	
	optimizer := cost.NewCostOptimizer()
	recommendations := make([]*cost.Recommendation, 0)
	
	if instanceID != "" {
		// Get recommendations for specific instance
		instance, err := s.getInstance(instanceID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		recommendations = optimizer.AnalyzeInstance(instance)
	} else if projectID != "" {
		// Get recommendations for project
		instances, err := s.getProjectInstances(projectID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		recommendations = optimizer.AnalyzeProject(projectID, instances)
	} else {
		http.Error(w, "instance_id or project_id is required", http.StatusBadRequest)
		return
	}
	
	totalSavings := optimizer.CalculateTotalSavings(recommendations)
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"recommendations": recommendations,
		"count":           len(recommendations),
		"total_savings":   totalSavings,
	})
}

// handleGetCostTrends returns cost trends for analysis
func (s *Server) handleGetCostTrends(w http.ResponseWriter, r *http.Request) {
	projectID := r.URL.Query().Get("project_id")
	period := r.URL.Query().Get("period") // 7d, 30d, 90d
	
	if projectID == "" {
		http.Error(w, "project_id is required", http.StatusBadRequest)
		return
	}
	
	if period == "" {
		period = "30d"
	}
	
	// Get cost trends from budget tracker
	trends, err := s.budgetTracker.GetCostTrends(projectID, period)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(trends)
}

// handleGetBudgetStatus returns current budget status for a project
func (s *Server) handleGetBudgetStatus(w http.ResponseWriter, r *http.Request) {
	projectID := r.URL.Query().Get("project_id")
	if projectID == "" {
		http.Error(w, "project_id is required", http.StatusBadRequest)
		return
	}
	
	status, err := s.budgetTracker.GetBudgetStatus(projectID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}

// handleUpdateBudgetAlert updates budget alert settings
func (s *Server) handleUpdateBudgetAlert(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ProjectID   string                `json:"project_id"`
		AlertType   types.BudgetAlertType `json:"alert_type"`
		Threshold   float64               `json:"threshold"`
		Enabled     bool                  `json:"enabled"`
		Actions     []string              `json:"actions"`
	}
	
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	
	// Create or update alert rule
	rule := &cost.AlertRule{
		Name:    string(req.AlertType),
		Type:    cost.AlertTypeThreshold,
		Enabled: req.Enabled,
		Conditions: cost.AlertConditions{
			BudgetPercentage: &req.Threshold,
		},
		Actions: req.Actions,
	}
	
	if err := s.alertManager.AddRule(rule); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status": "success",
	})
}

// Helper methods

// getInstance retrieves an instance by ID (mock implementation)
func (s *Server) getInstance(instanceID string) (*types.Instance, error) {
	// This would fetch from AWS in production
	return &types.Instance{
		ID:                 instanceID,
		Name:               "test-instance",
		InstanceType:       "t3.medium",
		EstimatedCost:      2.50,
		HibernationEnabled: false,
		SpotEligible:       true,
		IsSpot:             false,
		ARMCompatible:      true,
		Architecture:       "x86_64",
		AlwaysOn:           true,
		WorkloadType:       "development",
		Runtime:            1440, // 60 days
		StorageGB:          100,
		StorageUsedGB:      35,
	}, nil
}

// getProjectInstances retrieves all instances for a project (mock implementation)
func (s *Server) getProjectInstances(projectID string) ([]*types.Instance, error) {
	// This would fetch from AWS in production
	return []*types.Instance{
		{
			ID:                 "i-1234567890",
			Name:               "dev-server",
			InstanceType:       "t3.large",
			EstimatedCost:      5.00,
			HibernationEnabled: false,
			SpotEligible:       true,
			IsSpot:             false,
			ARMCompatible:      true,
			Architecture:       "x86_64",
			AlwaysOn:           true,
			WorkloadType:       "development",
			Runtime:            2880, // 120 days
			StorageGB:          200,
			StorageUsedGB:      80,
		},
		{
			ID:                 "i-0987654321",
			Name:               "ml-training",
			InstanceType:       "g4dn.xlarge",
			EstimatedCost:      25.00,
			HibernationEnabled: false,
			SpotEligible:       true,
			IsSpot:             false,
			ARMCompatible:      false,
			Architecture:       "x86_64",
			AlwaysOn:           false,
			WorkloadType:       "ml-training",
			Runtime:            720, // 30 days
			StorageGB:          500,
			StorageUsedGB:      450,
		},
	}, nil
}

// RegisterCostHandlers registers all cost-related HTTP handlers
func (s *Server) RegisterCostHandlers(mux *http.ServeMux, applyMiddleware func(http.HandlerFunc) http.HandlerFunc) {
	// Alert endpoints
	mux.HandleFunc("/api/v1/cost/alerts", applyMiddleware(s.handleGetCostAlerts))
	mux.HandleFunc("/api/v1/cost/alerts/active", applyMiddleware(s.handleGetActiveAlerts))
	mux.HandleFunc("/api/v1/cost/alerts/", applyMiddleware(s.handleAlertAction)) // Handles acknowledge and resolve
	mux.HandleFunc("/api/v1/cost/alerts/rules", applyMiddleware(s.handleAddAlertRule))
	
	// Optimization endpoints
	mux.HandleFunc("/api/v1/cost/optimization/report", applyMiddleware(s.handleGetOptimizationReport))
	mux.HandleFunc("/api/v1/cost/optimization/recommendations", applyMiddleware(s.handleGetRecommendations))
	
	// Budget and trends
	mux.HandleFunc("/api/v1/cost/trends", applyMiddleware(s.handleGetCostTrends))
	mux.HandleFunc("/api/v1/cost/budget/status", applyMiddleware(s.handleGetBudgetStatus))
	mux.HandleFunc("/api/v1/cost/budget/alerts", applyMiddleware(s.handleUpdateBudgetAlert))
}

// handleAlertAction routes alert actions (acknowledge, resolve)
func (s *Server) handleAlertAction(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	path := r.URL.Path
	if strings.Contains(path, "/acknowledge") {
		s.handleAcknowledgeAlert(w, r)
	} else if strings.Contains(path, "/resolve") {
		s.handleResolveAlert(w, r)
	} else {
		http.Error(w, "Unknown action", http.StatusBadRequest)
	}
}