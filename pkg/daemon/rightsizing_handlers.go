package daemon

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/scttfrdmn/cloudworkstation/pkg/types"
)

// registerRightsizingRoutes registers all rightsizing-related API endpoints
func (s *Server) registerRightsizingRoutes(mux *http.ServeMux, applyMiddleware func(http.HandlerFunc) http.HandlerFunc) {
	// Rightsizing analysis endpoints
	mux.HandleFunc("/api/v1/rightsizing/analyze", applyMiddleware(s.handleRightsizingAnalyze))
	mux.HandleFunc("/api/v1/rightsizing/recommendations", applyMiddleware(s.handleRightsizingRecommendations))
	mux.HandleFunc("/api/v1/rightsizing/stats", applyMiddleware(s.handleRightsizingStats))
	mux.HandleFunc("/api/v1/rightsizing/export", applyMiddleware(s.handleRightsizingExport))
	mux.HandleFunc("/api/v1/rightsizing/summary", applyMiddleware(s.handleRightsizingSummary))

	// Instance metrics endpoints
	mux.HandleFunc("/api/v1/instances/metrics", applyMiddleware(s.handleInstanceMetrics))
	mux.HandleFunc("/api/v1/rightsizing/instance/", applyMiddleware(s.handleInstanceMetricsOperations))
}

// handleRightsizingAnalyze handles POST /api/v1/rightsizing/analyze
func (s *Server) handleRightsizingAnalyze(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		s.performRightsizingAnalysis(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// performRightsizingAnalysis performs rightsizing analysis for a specific instance
func (s *Server) performRightsizingAnalysis(w http.ResponseWriter, r *http.Request) {
	var req types.RightsizingAnalysisRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.InstanceName == "" {
		http.Error(w, "Instance name is required", http.StatusBadRequest)
		return
	}

	// Get instance information
	instances, err := s.awsManager.ListInstances()
	if err != nil {
		log.Printf("Failed to list instances for rightsizing analysis: %v", err)
		http.Error(w, "Failed to retrieve instance information", http.StatusInternalServerError)
		return
	}

	var targetInstance *types.Instance
	for i := range instances {
		if instances[i].Name == req.InstanceName {
			targetInstance = &instances[i]
			break
		}
	}

	if targetInstance == nil {
		http.Error(w, fmt.Sprintf("Instance '%s' not found", req.InstanceName), http.StatusNotFound)
		return
	}

	// Check if instance is running for meaningful analysis
	if targetInstance.State != "running" {
		response := types.RightsizingAnalysisResponse{
			Recommendation:      nil,
			MetricsAvailable:    false,
			DataPointsCount:     0,
			AnalysisPeriodHours: 0,
			LastUpdated:         time.Now(),
			Message:             fmt.Sprintf("Instance '%s' is not running. Analysis requires a running instance with collected metrics.", req.InstanceName),
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
		return
	}

	// Generate rightsizing recommendation
	recommendation := s.generateRightsizingRecommendation(targetInstance, req.AnalysisPeriodHours, req.ForceRefresh)

	// Simulate some metrics availability (in real implementation, this would query actual metrics)
	dataPointsCount := s.calculateDataPointsCount(targetInstance, req.AnalysisPeriodHours)

	response := types.RightsizingAnalysisResponse{
		Recommendation:      recommendation,
		MetricsAvailable:    dataPointsCount > 0,
		DataPointsCount:     dataPointsCount,
		AnalysisPeriodHours: req.AnalysisPeriodHours,
		LastUpdated:         time.Now(),
		Message:             "",
	}

	if dataPointsCount == 0 {
		response.Message = "Insufficient metrics data for analysis. Allow more runtime for meaningful recommendations."
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleRightsizingRecommendations handles GET /api/v1/rightsizing/recommendations
func (s *Server) handleRightsizingRecommendations(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.getAllRightsizingRecommendations(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// getAllRightsizingRecommendations retrieves recommendations for all instances
func (s *Server) getAllRightsizingRecommendations(w http.ResponseWriter, r *http.Request) {
	instances, err := s.awsManager.ListInstances()
	if err != nil {
		log.Printf("Failed to list instances for rightsizing recommendations: %v", err)
		http.Error(w, "Failed to retrieve instances", http.StatusInternalServerError)
		return
	}

	var recommendations []types.RightsizingRecommendation
	var activeInstances int
	var potentialSavings float64

	for _, instance := range instances {
		if instance.State == "running" {
			activeInstances++
			recommendation := s.generateRightsizingRecommendation(&instance, 24, false)
			if recommendation != nil {
				recommendations = append(recommendations, *recommendation)
				if recommendation.CostImpact.DailyDifference < 0 { // Negative means savings
					potentialSavings += -recommendation.CostImpact.DailyDifference * 30 // Monthly savings
				}
			}
		}
	}

	response := types.RightsizingRecommendationsResponse{
		Recommendations:  recommendations,
		TotalInstances:   len(instances),
		ActiveInstances:  activeInstances,
		PotentialSavings: potentialSavings,
		GeneratedAt:      time.Now(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleRightsizingStats handles GET /api/v1/rightsizing/stats?instance=<name>
func (s *Server) handleRightsizingStats(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.getRightsizingStats(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// getRightsizingStats retrieves detailed stats for a specific instance
func (s *Server) getRightsizingStats(w http.ResponseWriter, r *http.Request) {
	instanceName := r.URL.Query().Get("instance")
	if instanceName == "" {
		http.Error(w, "Instance name parameter is required", http.StatusBadRequest)
		return
	}

	instances, err := s.awsManager.ListInstances()
	if err != nil {
		log.Printf("Failed to list instances for rightsizing stats: %v", err)
		http.Error(w, "Failed to retrieve instance information", http.StatusInternalServerError)
		return
	}

	var targetInstance *types.Instance
	for i := range instances {
		if instances[i].Name == instanceName {
			targetInstance = &instances[i]
			break
		}
	}

	if targetInstance == nil {
		http.Error(w, fmt.Sprintf("Instance '%s' not found", instanceName), http.StatusNotFound)
		return
	}

	// Generate detailed stats
	statsResponse := s.generateRightsizingStats(targetInstance)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(statsResponse)
}

// handleRightsizingExport handles GET /api/v1/rightsizing/export?instance=<name>
func (s *Server) handleRightsizingExport(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.exportRightsizingData(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// exportRightsizingData exports usage data for a specific instance
func (s *Server) exportRightsizingData(w http.ResponseWriter, r *http.Request) {
	instanceName := r.URL.Query().Get("instance")
	if instanceName == "" {
		http.Error(w, "Instance name parameter is required", http.StatusBadRequest)
		return
	}

	instances, err := s.awsManager.ListInstances()
	if err != nil {
		log.Printf("Failed to list instances for rightsizing export: %v", err)
		http.Error(w, "Failed to retrieve instance information", http.StatusInternalServerError)
		return
	}

	var targetInstance *types.Instance
	for i := range instances {
		if instances[i].Name == instanceName {
			targetInstance = &instances[i]
			break
		}
	}

	if targetInstance == nil {
		http.Error(w, fmt.Sprintf("Instance '%s' not found", instanceName), http.StatusNotFound)
		return
	}

	// Generate sample metrics for export
	metrics := s.generateSampleMetrics(targetInstance, 100)

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s-metrics.json", instanceName))
	json.NewEncoder(w).Encode(metrics)
}

// handleRightsizingSummary handles GET /api/v1/rightsizing/summary
func (s *Server) handleRightsizingSummary(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.getRightsizingSummary(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// getRightsizingSummary retrieves fleet-wide rightsizing summary
func (s *Server) getRightsizingSummary(w http.ResponseWriter, r *http.Request) {
	instances, err := s.awsManager.ListInstances()
	if err != nil {
		log.Printf("Failed to list instances for rightsizing summary: %v", err)
		http.Error(w, "Failed to retrieve instances", http.StatusInternalServerError)
		return
	}

	summary := s.generateFleetRightsizingSummary(instances)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(summary)
}

// handleInstanceMetrics handles GET /api/v1/instances/metrics
func (s *Server) handleInstanceMetrics(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.getInstanceMetrics(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// getInstanceMetrics retrieves metrics for all instances
func (s *Server) getInstanceMetrics(w http.ResponseWriter, r *http.Request) {
	instances, err := s.awsManager.ListInstances()
	if err != nil {
		log.Printf("Failed to list instances for metrics: %v", err)
		http.Error(w, "Failed to retrieve instances", http.StatusInternalServerError)
		return
	}

	var allMetrics []types.InstanceMetrics
	for _, instance := range instances {
		if instance.State == "running" {
			// Generate recent metrics for running instances
			metrics := s.generateSampleMetrics(&instance, 10) // Last 10 data points
			allMetrics = append(allMetrics, metrics...)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(allMetrics)
}

// handleInstanceMetricsOperations handles GET /api/v1/instances/{name}/metrics
func (s *Server) handleInstanceMetricsOperations(w http.ResponseWriter, r *http.Request) {
	// Extract instance name from path
	path := strings.TrimPrefix(r.URL.Path, "/api/v1/instances/")
	pathParts := strings.Split(path, "/")

	if len(pathParts) < 2 || pathParts[1] != "metrics" {
		// Not a metrics operation, let the main instance handler handle it
		s.handleInstanceOperations(w, r)
		return
	}

	instanceName := pathParts[0]

	switch r.Method {
	case http.MethodGet:
		s.getInstanceSpecificMetrics(w, r, instanceName)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// getInstanceSpecificMetrics retrieves metrics for a specific instance
func (s *Server) getInstanceSpecificMetrics(w http.ResponseWriter, r *http.Request, instanceName string) {
	instances, err := s.awsManager.ListInstances()
	if err != nil {
		log.Printf("Failed to list instances for specific metrics: %v", err)
		http.Error(w, "Failed to retrieve instance information", http.StatusInternalServerError)
		return
	}

	var targetInstance *types.Instance
	for i := range instances {
		if instances[i].Name == instanceName {
			targetInstance = &instances[i]
			break
		}
	}

	if targetInstance == nil {
		http.Error(w, fmt.Sprintf("Instance '%s' not found", instanceName), http.StatusNotFound)
		return
	}

	// Parse query parameters for pagination/filtering
	limitStr := r.URL.Query().Get("limit")
	limit := 50 // Default limit
	if limitStr != "" {
		if parsedLimit, err := strconv.Atoi(limitStr); err == nil && parsedLimit > 0 {
			limit = parsedLimit
		}
	}

	metrics := s.generateSampleMetrics(targetInstance, limit)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(metrics)
}

// Helper methods for generating rightsizing data

// generateRightsizingRecommendation generates a rightsizing recommendation for an instance
func (s *Server) generateRightsizingRecommendation(instance *types.Instance, analysisPeriodHours float64, forceRefresh bool) *types.RightsizingRecommendation {
	if instance.State != "running" {
		return nil
	}

	// Simulate recommendation logic based on instance type and template
	currentSize := s.parseInstanceSize(instance.InstanceType)
	recommendedSize := s.predictOptimalSize(instance, analysisPeriodHours)

	recommendationType := types.RightsizingOptimal
	if recommendedSize != currentSize {
		if s.isSizeSmaller(recommendedSize, currentSize) {
			recommendationType = types.RightsizingDownsize
		} else {
			recommendationType = types.RightsizingUpsize
		}
	}

	// Determine confidence based on runtime
	confidence := s.calculateConfidence(instance, analysisPeriodHours)

	// Calculate cost impact
	currentCost := instance.HourlyRate * 24 // Daily cost
	recommendedCost := s.estimateCostForSize(recommendedSize) * 24

	costImpact := types.CostImpact{
		CurrentDailyCost:     currentCost,
		RecommendedDailyCost: recommendedCost,
		DailyDifference:      recommendedCost - currentCost,
		PercentageChange:     ((recommendedCost - currentCost) / currentCost) * 100,
		MonthlySavings:       (currentCost - recommendedCost) * 30,
		AnnualSavings:        (currentCost - recommendedCost) * 365,
		IsIncrease:           recommendedCost > currentCost,
	}

	// Generate resource analysis
	resourceAnalysis := s.generateResourceAnalysis(instance)

	return &types.RightsizingRecommendation{
		InstanceID:              instance.ID,
		InstanceName:            instance.Name,
		CurrentInstanceType:     instance.InstanceType,
		CurrentSize:             currentSize,
		RecommendedInstanceType: s.getInstanceTypeForSize(recommendedSize),
		RecommendedSize:         recommendedSize,
		RecommendationType:      recommendationType,
		Confidence:              confidence,
		Reasoning:               s.generateRecommendationReasoning(instance, currentSize, recommendedSize, recommendationType),
		CostImpact:              costImpact,
		ResourceAnalysis:        resourceAnalysis,
		CreatedAt:               time.Now(),
		DataPointsAnalyzed:      s.calculateDataPointsCount(instance, analysisPeriodHours),
		AnalysisPeriodHours:     analysisPeriodHours,
	}
}

// parseInstanceSize converts instance type to t-shirt size
func (s *Server) parseInstanceSize(instanceType string) string {
	sizeMap := map[string]string{
		"t3.nano":     "XS",
		"t3.micro":    "XS",
		"t3.small":    "S",
		"t3.medium":   "M",
		"t3.large":    "L",
		"t3.xlarge":   "XL",
		"t3.2xlarge":  "XL",
		"t3a.nano":    "XS",
		"t3a.micro":   "XS",
		"t3a.small":   "S",
		"t3a.medium":  "M",
		"t3a.large":   "L",
		"t3a.xlarge":  "XL",
		"t3a.2xlarge": "XL",
		"t4g.nano":    "XS",
		"t4g.micro":   "XS",
		"t4g.small":   "S",
		"t4g.medium":  "M",
		"t4g.large":   "L",
		"t4g.xlarge":  "XL",
		"t4g.2xlarge": "XL",
	}

	if size, exists := sizeMap[instanceType]; exists {
		return size
	}
	return "M" // Default to medium
}

// predictOptimalSize predicts optimal size based on instance template and runtime
func (s *Server) predictOptimalSize(instance *types.Instance, analysisPeriodHours float64) string {
	currentSize := s.parseInstanceSize(instance.InstanceType)
	template := strings.ToLower(instance.Template)

	// Simple heuristic based on template type and runtime
	switch {
	case strings.Contains(template, "ml") || strings.Contains(template, "gpu") || strings.Contains(template, "machine learning"):
		// ML workloads typically need more resources
		if currentSize == "XS" {
			return "S"
		} else if currentSize == "S" {
			return "M"
		}
	case strings.Contains(template, "r-") || strings.Contains(template, "r ") || strings.Contains(template, "research"):
		// R workloads are memory intensive
		if currentSize == "XS" {
			return "S"
		} else if currentSize == "S" {
			return "M"
		}
	case strings.Contains(template, "simple") || strings.Contains(template, "basic") || strings.Contains(template, "ubuntu"):
		// Simple workloads might be over-provisioned
		if currentSize == "L" || currentSize == "XL" {
			return "M"
		} else if currentSize == "M" {
			return "S"
		}
	}

	// If instance has been running for a while with likely low utilization, consider downsizing
	runtime := time.Since(instance.LaunchTime).Hours()
	if runtime > 24 && analysisPeriodHours > 12 {
		// Simulate some downsizing opportunities for long-running instances
		if currentSize == "L" {
			return "M"
		} else if currentSize == "M" {
			return "S"
		}
	}

	return currentSize // Default: current size is optimal
}

// calculateConfidence determines confidence level based on available data
func (s *Server) calculateConfidence(instance *types.Instance, analysisPeriodHours float64) types.ConfidenceLevel {
	runtime := time.Since(instance.LaunchTime).Hours()

	switch {
	case runtime < 1:
		return types.ConfidenceLow
	case runtime < 24:
		return types.ConfidenceMedium
	case runtime < 168: // 1 week
		return types.ConfidenceHigh
	default:
		return types.ConfidenceVeryHigh
	}
}

// generateResourceAnalysis creates detailed resource analysis
func (s *Server) generateResourceAnalysis(instance *types.Instance) types.ResourceAnalysis {
	// Simulate resource analysis based on instance type and template
	template := strings.ToLower(instance.Template)
	currentSize := s.parseInstanceSize(instance.InstanceType)

	// Generate simulated CPU analysis
	cpuAnalysis := types.CPUAnalysis{
		AverageUtilization: s.simulateCPUUtilization(template, currentSize),
		PeakUtilization:    s.simulateCPUUtilization(template, currentSize) * 1.8,
		P95Utilization:     s.simulateCPUUtilization(template, currentSize) * 1.5,
		P99Utilization:     s.simulateCPUUtilization(template, currentSize) * 1.7,
		IdlePercentage:     100 - s.simulateCPUUtilization(template, currentSize),
		IsBottleneck:       s.simulateCPUUtilization(template, currentSize) > 80,
		Recommendation:     s.generateCPURecommendation(template, currentSize),
	}

	// Generate simulated memory analysis
	memoryAnalysis := types.MemoryAnalysis{
		AverageUtilization: s.simulateMemoryUtilization(template, currentSize),
		PeakUtilization:    s.simulateMemoryUtilization(template, currentSize) * 1.3,
		P95Utilization:     s.simulateMemoryUtilization(template, currentSize) * 1.2,
		P99Utilization:     s.simulateMemoryUtilization(template, currentSize) * 1.25,
		SwapUsage:          0.0, // Assume no swap usage
		IsBottleneck:       s.simulateMemoryUtilization(template, currentSize) > 85,
		Recommendation:     s.generateMemoryRecommendation(template, currentSize),
	}

	// Generate storage analysis
	storageAnalysis := types.StorageAnalysis{
		AverageIOPS:       100.0,
		PeakIOPS:          300.0,
		AverageThroughput: 50.0,
		PeakThroughput:    150.0,
		SpaceUtilization:  45.0,
		IsBottleneck:      false,
		Recommendation:    "Storage performance is adequate for current workload",
	}

	// Generate network analysis
	networkAnalysis := types.NetworkAnalysis{
		AverageThroughput: 25.0,
		PeakThroughput:    100.0,
		PacketRate:        1000.0,
		IsBottleneck:      false,
		Recommendation:    "Network performance is sufficient",
	}

	// Generate workload pattern
	workloadPattern := types.WorkloadPattern{
		Type:                types.WorkloadPatternSteady,
		ConsistencyScore:    0.8,
		PeakHours:           []int{9, 10, 11, 14, 15, 16},
		SeasonalityDetected: false,
		GrowthTrend:         0.05, // 5% growth trend
		BurstFrequency:      0.2,  // 20% burst frequency
		Description:         "Steady workload with predictable daily patterns",
	}

	return types.ResourceAnalysis{
		CPUAnalysis:     cpuAnalysis,
		MemoryAnalysis:  memoryAnalysis,
		StorageAnalysis: storageAnalysis,
		NetworkAnalysis: networkAnalysis,
		WorkloadPattern: workloadPattern,
	}
}

// simulateCPUUtilization generates simulated CPU utilization based on template
func (s *Server) simulateCPUUtilization(template, size string) float64 {
	base := 30.0 // Base CPU utilization

	switch {
	case strings.Contains(template, "ml") || strings.Contains(template, "gpu"):
		base = 60.0
	case strings.Contains(template, "r-") || strings.Contains(template, "research"):
		base = 45.0
	case strings.Contains(template, "simple") || strings.Contains(template, "basic"):
		base = 20.0
	}

	// Adjust based on size (larger instances might have lower utilization)
	switch size {
	case "XL":
		base *= 0.7
	case "L":
		base *= 0.8
	case "S":
		base *= 1.2
	case "XS":
		base *= 1.4
	}

	return base
}

// simulateMemoryUtilization generates simulated memory utilization
func (s *Server) simulateMemoryUtilization(template, size string) float64 {
	base := 50.0 // Base memory utilization

	switch {
	case strings.Contains(template, "ml") || strings.Contains(template, "gpu"):
		base = 70.0
	case strings.Contains(template, "r-") || strings.Contains(template, "research"):
		base = 65.0
	case strings.Contains(template, "simple") || strings.Contains(template, "basic"):
		base = 35.0
	}

	// Adjust based on size
	switch size {
	case "XL":
		base *= 0.6
	case "L":
		base *= 0.7
	case "S":
		base *= 1.3
	case "XS":
		base *= 1.6
	}

	return base
}

// generateCPURecommendation generates CPU-specific recommendations
func (s *Server) generateCPURecommendation(template, size string) string {
	utilization := s.simulateCPUUtilization(template, size)

	switch {
	case utilization > 80:
		return "CPU utilization is high. Consider upgrading to a larger instance size."
	case utilization < 20:
		return "CPU utilization is low. Consider downsizing to reduce costs."
	default:
		return "CPU utilization is within optimal range."
	}
}

// generateMemoryRecommendation generates memory-specific recommendations
func (s *Server) generateMemoryRecommendation(template, size string) string {
	utilization := s.simulateMemoryUtilization(template, size)

	switch {
	case utilization > 85:
		return "Memory utilization is high. Consider upgrading to a memory-optimized instance."
	case utilization < 30:
		return "Memory utilization is low. Consider downsizing to reduce costs."
	default:
		return "Memory utilization is within optimal range."
	}
}

// generateRecommendationReasoning provides reasoning for recommendations
func (s *Server) generateRecommendationReasoning(instance *types.Instance, currentSize, recommendedSize string, recType types.RightsizingType) string {
	template := strings.ToLower(instance.Template)

	switch recType {
	case types.RightsizingDownsize:
		return fmt.Sprintf("Instance is over-provisioned for %s workload. Current %s size shows low resource utilization. Downsizing to %s will reduce costs while maintaining performance.",
			template, currentSize, recommendedSize)
	case types.RightsizingUpsize:
		return fmt.Sprintf("Instance is under-provisioned for %s workload. Current %s size shows high resource utilization. Upgrading to %s will improve performance and prevent bottlenecks.",
			template, currentSize, recommendedSize)
	case types.RightsizingOptimal:
		return fmt.Sprintf("Instance size %s is optimal for %s workload. Resource utilization is balanced across CPU, memory, and storage.",
			currentSize, template)
	default:
		return "Resource utilization analysis suggests current sizing is appropriate."
	}
}

// isSizeSmaller checks if one size is smaller than another
func (s *Server) isSizeSmaller(size1, size2 string) bool {
	sizeOrder := map[string]int{"XS": 1, "S": 2, "M": 3, "L": 4, "XL": 5}
	return sizeOrder[size1] < sizeOrder[size2]
}

// getInstanceTypeForSize returns the instance type for a given size
func (s *Server) getInstanceTypeForSize(size string) string {
	typeMap := map[string]string{
		"XS": "t4g.nano",
		"S":  "t4g.small",
		"M":  "t4g.medium",
		"L":  "t4g.large",
		"XL": "t4g.xlarge",
	}

	if instanceType, exists := typeMap[size]; exists {
		return instanceType
	}
	return "t4g.medium" // Default
}

// estimateCostForSize estimates hourly cost for a given size
func (s *Server) estimateCostForSize(size string) float64 {
	costMap := map[string]float64{
		"XS": 0.021, // t4g.nano
		"S":  0.042, // t4g.small
		"M":  0.084, // t4g.medium
		"L":  0.168, // t4g.large
		"XL": 0.336, // t4g.xlarge
	}

	if cost, exists := costMap[size]; exists {
		return cost
	}
	return 0.084 // Default medium cost
}

// calculateDataPointsCount calculates expected number of data points
func (s *Server) calculateDataPointsCount(instance *types.Instance, analysisPeriodHours float64) int {
	runtime := time.Since(instance.LaunchTime).Hours()
	if analysisPeriodHours == 0 {
		analysisPeriodHours = runtime
	}

	effectivePeriod := analysisPeriodHours
	if effectivePeriod > runtime {
		effectivePeriod = runtime
	}

	// Assume metrics collected every 2 minutes (30 per hour)
	return int(effectivePeriod * 30)
}

// generateRightsizingStats creates detailed stats response
func (s *Server) generateRightsizingStats(instance *types.Instance) types.RightsizingStatsResponse {
	currentSize := s.parseInstanceSize(instance.InstanceType)

	// Current configuration
	config := types.InstanceConfiguration{
		InstanceType:       instance.InstanceType,
		Size:               currentSize,
		VCPUs:              s.getCPUsForSize(currentSize),
		MemoryGB:           s.getMemoryForSize(currentSize),
		StorageGB:          instance.StorageGB,
		NetworkPerformance: s.getNetworkPerformanceForSize(currentSize),
		DailyCost:          instance.HourlyRate * 24,
	}

	// Metrics summary
	template := strings.ToLower(instance.Template)
	metricsSummary := types.MetricsSummary{
		CPUSummary:     s.generateResourceSummary("cpu", template, currentSize),
		MemorySummary:  s.generateResourceSummary("memory", template, currentSize),
		StorageSummary: s.generateResourceSummary("storage", template, currentSize),
		NetworkSummary: s.generateResourceSummary("network", template, currentSize),
	}

	// Recent metrics
	recentMetrics := s.generateSampleMetrics(instance, 10)

	// Collection status
	collectionStatus := types.MetricsCollectionStatus{
		IsActive:           instance.State == "running",
		LastCollectionTime: time.Now().Add(-2 * time.Minute),
		CollectionInterval: "2 minutes",
		TotalDataPoints:    s.calculateDataPointsCount(instance, 0),
		DataRetentionDays:  7,
		StorageLocation:    "/var/log/cloudworkstation-metrics.json",
	}

	var recommendation *types.RightsizingRecommendation
	if instance.State == "running" {
		recommendation = s.generateRightsizingRecommendation(instance, 24, false)
	}

	return types.RightsizingStatsResponse{
		InstanceName:         instance.Name,
		CurrentConfiguration: config,
		MetricsSummary:       metricsSummary,
		RecentMetrics:        recentMetrics,
		Recommendation:       recommendation,
		CollectionStatus:     collectionStatus,
	}
}

// Helper methods for generating stats
func (s *Server) getCPUsForSize(size string) int {
	cpuMap := map[string]int{"XS": 1, "S": 2, "M": 2, "L": 4, "XL": 8}
	if cpus, exists := cpuMap[size]; exists {
		return cpus
	}
	return 2
}

func (s *Server) getMemoryForSize(size string) float64 {
	memMap := map[string]float64{"XS": 2, "S": 4, "M": 8, "L": 16, "XL": 32}
	if mem, exists := memMap[size]; exists {
		return mem
	}
	return 8
}

func (s *Server) getNetworkPerformanceForSize(size string) string {
	perfMap := map[string]string{
		"XS": "Low",
		"S":  "Low to Moderate",
		"M":  "Moderate",
		"L":  "High",
		"XL": "High",
	}
	if perf, exists := perfMap[size]; exists {
		return perf
	}
	return "Moderate"
}

// generateResourceSummary creates summary for a resource type
func (s *Server) generateResourceSummary(resourceType, template, size string) types.ResourceSummary {
	var average, peak float64

	switch resourceType {
	case "cpu":
		average = s.simulateCPUUtilization(template, size)
		peak = average * 1.8
	case "memory":
		average = s.simulateMemoryUtilization(template, size)
		peak = average * 1.3
	case "storage":
		average = 45.0
		peak = 80.0
	case "network":
		average = 25.0
		peak = 100.0
	default:
		average = 50.0
		peak = 80.0
	}

	return types.ResourceSummary{
		Average:           average,
		Peak:              peak,
		P95:               peak * 0.9,
		P99:               peak * 0.95,
		Minimum:           average * 0.5,
		StandardDeviation: average * 0.2,
		TrendDirection:    "stable",
		Bottleneck:        average > 80,
		Underutilized:     average < 20,
	}
}

// generateSampleMetrics generates sample metrics for an instance
func (s *Server) generateSampleMetrics(instance *types.Instance, count int) []types.InstanceMetrics {
	var metrics []types.InstanceMetrics
	template := strings.ToLower(instance.Template)
	size := s.parseInstanceSize(instance.InstanceType)

	for i := 0; i < count; i++ {
		timestamp := time.Now().Add(-time.Duration(i*2) * time.Minute)

		metric := types.InstanceMetrics{
			InstanceID:   instance.ID,
			InstanceName: instance.Name,
			Timestamp:    timestamp,
			CPU: types.CPUMetrics{
				UtilizationPercent: s.simulateCPUUtilization(template, size) + (float64(i%10) - 5),
				Load1Min:           1.2,
				Load5Min:           1.0,
				Load15Min:          0.8,
				CoreCount:          s.getCPUsForSize(size),
				IdlePercent:        100 - s.simulateCPUUtilization(template, size),
				WaitPercent:        5.0,
			},
			Memory: types.MemoryMetrics{
				TotalMB:            s.getMemoryForSize(size) * 1024,
				UsedMB:             s.getMemoryForSize(size) * 1024 * s.simulateMemoryUtilization(template, size) / 100,
				FreeMB:             s.getMemoryForSize(size) * 1024 * (100 - s.simulateMemoryUtilization(template, size)) / 100,
				AvailableMB:        s.getMemoryForSize(size) * 1024 * 0.8,
				CachedMB:           s.getMemoryForSize(size) * 1024 * 0.1,
				BufferedMB:         s.getMemoryForSize(size) * 1024 * 0.05,
				UtilizationPercent: s.simulateMemoryUtilization(template, size),
				SwapTotalMB:        0,
				SwapUsedMB:         0,
			},
			Storage: types.StorageMetrics{
				TotalGB:             instance.StorageGB,
				UsedGB:              instance.StorageGB * 0.45,
				AvailableGB:         instance.StorageGB * 0.55,
				UtilizationPercent:  45.0,
				ReadIOPS:            100.0,
				WriteIOPS:           50.0,
				ReadThroughputMBps:  25.0,
				WriteThroughputMBps: 15.0,
			},
			Network: types.NetworkMetrics{
				RxBytesPerSec:   1024 * 25, // 25 KB/s
				TxBytesPerSec:   1024 * 20, // 20 KB/s
				RxPacketsPerSec: 100,
				TxPacketsPerSec: 80,
				TotalRxBytes:    1024 * 1024 * 100, // 100 MB
				TotalTxBytes:    1024 * 1024 * 80,  // 80 MB
			},
			System: types.SystemMetrics{
				ProcessCount:    120,
				LoggedInUsers:   1,
				UptimeSeconds:   time.Since(instance.LaunchTime).Seconds(),
				LastActivity:    time.Now().Add(-time.Duration(i*30) * time.Second),
				LoadAverage1Min: 1.2,
			},
		}

		// Add GPU metrics for ML templates
		if strings.Contains(template, "ml") || strings.Contains(template, "gpu") {
			metric.GPU = &types.GPUMetrics{
				Count:                    1,
				UtilizationPercent:       75.0,
				MemoryTotalMB:            8192, // 8GB VRAM
				MemoryUsedMB:             6144, // 6GB used
				MemoryUtilizationPercent: 75.0,
				TemperatureCelsius:       65.0,
				PowerDrawWatts:           200.0,
			}
		}

		metrics = append(metrics, metric)
	}

	// Sort by timestamp (newest first)
	sort.Slice(metrics, func(i, j int) bool {
		return metrics[i].Timestamp.After(metrics[j].Timestamp)
	})

	return metrics
}

// generateFleetRightsizingSummary generates fleet-wide summary
func (s *Server) generateFleetRightsizingSummary(instances []types.Instance) types.RightsizingSummaryResponse {
	var runningInstances, stoppedInstances, instancesWithMetrics int
	var totalDailyCost float64
	var potentialSavings float64
	var overprovisioned, underprovisioned, optimal int
	var totalCPU, totalMemory, totalStorage float64
	var cpuCount, memoryCount, storageCount int

	for _, instance := range instances {
		if instance.State == "running" {
			runningInstances++
			totalDailyCost += instance.HourlyRate * 24
			instancesWithMetrics++

			// Simulate rightsizing analysis
			template := strings.ToLower(instance.Template)
			currentSize := s.parseInstanceSize(instance.InstanceType)
			recommendedSize := s.predictOptimalSize(&instance, 24)

			if s.isSizeSmaller(recommendedSize, currentSize) {
				overprovisioned++
				// Calculate potential savings
				currentCost := instance.HourlyRate * 24
				recommendedCost := s.estimateCostForSize(recommendedSize) * 24
				potentialSavings += (currentCost - recommendedCost) * 30 // Monthly
			} else if currentSize != recommendedSize {
				underprovisioned++
			} else {
				optimal++
			}

			// Add to utilization averages
			cpuUtil := s.simulateCPUUtilization(template, currentSize)
			memUtil := s.simulateMemoryUtilization(template, currentSize)
			storageUtil := 45.0 // Fixed storage utilization

			totalCPU += cpuUtil
			cpuCount++
			totalMemory += memUtil
			memoryCount++
			totalStorage += storageUtil
			storageCount++
		} else {
			stoppedInstances++
		}
	}

	fleetOverview := types.FleetOverview{
		TotalInstances:       len(instances),
		RunningInstances:     runningInstances,
		StoppedInstances:     stoppedInstances,
		TotalDailyCost:       totalDailyCost,
		TotalMonthlyCost:     totalDailyCost * 30,
		InstancesWithMetrics: instancesWithMetrics,
	}

	costOptimization := types.CostOptimizationSummary{
		PotentialDailySavings:         potentialSavings / 30,
		PotentialMonthlySavings:       potentialSavings,
		PotentialAnnualSavings:        potentialSavings * 12,
		SavingsPercentage:             (potentialSavings * 12 / (totalDailyCost * 365)) * 100,
		OverprovisionedInstances:      overprovisioned,
		UnderprovisionedInstances:     underprovisioned,
		OptimallyProvisionedInstances: optimal,
	}

	resourceUtilization := types.ResourceUtilizationSummary{
		AverageCPUUtilization:     totalCPU / max(float64(cpuCount), 1),
		AverageMemoryUtilization:  totalMemory / max(float64(memoryCount), 1),
		AverageStorageUtilization: totalStorage / max(float64(storageCount), 1),
		InstancesWithLowCPU:       0, // Would need actual analysis
		InstancesWithHighCPU:      0,
		InstancesWithLowMemory:    0,
		InstancesWithHighMemory:   0,
	}

	recommendations := types.RecommendationsSummary{
		TotalRecommendations:    overprovisioned + underprovisioned,
		DownsizeRecommendations: overprovisioned,
		UpsizeRecommendations:   underprovisioned,
		OptimizeRecommendations: 0,
		HighConfidenceCount:     (overprovisioned + underprovisioned) / 2,
		MediumConfidenceCount:   (overprovisioned + underprovisioned) / 3,
		LowConfidenceCount:      (overprovisioned + underprovisioned) / 6,
	}

	return types.RightsizingSummaryResponse{
		FleetOverview:       fleetOverview,
		CostOptimization:    costOptimization,
		ResourceUtilization: resourceUtilization,
		Recommendations:     recommendations,
		GeneratedAt:         time.Now(),
	}
}

// max returns the maximum of two float64 values
func max(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}
