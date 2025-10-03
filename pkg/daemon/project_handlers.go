package daemon

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/scttfrdmn/cloudworkstation/pkg/project"
	"github.com/scttfrdmn/cloudworkstation/pkg/types"
)

// handleProjectOperations routes project-related requests
func (s *Server) handleProjectOperations(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.handleListProjects(w, r)
	case http.MethodPost:
		s.handleCreateProject(w, r)
	default:
		s.writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
	}
}

// handleProjectByID routes project-specific requests
func (s *Server) handleProjectByID(w http.ResponseWriter, r *http.Request) {
	// Parse project ID from path
	path := r.URL.Path[len("/api/v1/projects/"):]
	parts := splitPath(path)
	if len(parts) == 0 {
		s.writeError(w, http.StatusBadRequest, "Missing project ID")
		return
	}

	projectID := parts[0]

	if len(parts) == 1 {
		// Direct project operations
		switch r.Method {
		case http.MethodGet:
			s.handleGetProject(w, r, projectID)
		case http.MethodPut:
			s.handleUpdateProject(w, r, projectID)
		case http.MethodDelete:
			s.handleDeleteProject(w, r, projectID)
		default:
			s.writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		}
		return
	}

	// Sub-operations
	operation := parts[1]
	switch operation {
	case "members":
		s.handleProjectMembers(w, r, projectID, parts)
	case "budget":
		s.handleProjectBudget(w, r, projectID)
	case "costs":
		s.handleProjectCosts(w, r, projectID)
	case "usage":
		s.handleProjectUsage(w, r, projectID)
	default:
		s.writeError(w, http.StatusNotFound, "Unknown project operation")
	}
}

// handleListProjects lists projects with optional filtering
func (s *Server) handleListProjects(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	filter := s.parseProjectFilter(r)

	projects, err := s.projectManager.ListProjects(context.Background(), filter)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to list projects: %v", err))
		return
	}

	summaries := s.buildProjectSummaries(projects)

	response := project.ProjectListResponse{
		Projects:      summaries,
		TotalCount:    len(summaries),
		FilteredCount: len(summaries),
	}

	_ = json.NewEncoder(w).Encode(response)
}

func (s *Server) parseProjectFilter(r *http.Request) *project.ProjectFilter {
	filter := &project.ProjectFilter{}
	query := r.URL.Query()

	if owner := query.Get("owner"); owner != "" {
		filter.Owner = owner
	}

	if status := query.Get("status"); status != "" {
		projectStatus := types.ProjectStatus(status)
		filter.Status = &projectStatus
	}

	s.parseTimeFilters(query, filter)

	if hasBudget := query.Get("has_budget"); hasBudget != "" {
		if b, err := strconv.ParseBool(hasBudget); err == nil {
			filter.HasBudget = &b
		}
	}

	return filter
}

func (s *Server) parseTimeFilters(query url.Values, filter *project.ProjectFilter) {
	if createdAfter := query.Get("created_after"); createdAfter != "" {
		if t, err := time.Parse(time.RFC3339, createdAfter); err == nil {
			filter.CreatedAfter = &t
		}
	}

	if createdBefore := query.Get("created_before"); createdBefore != "" {
		if t, err := time.Parse(time.RFC3339, createdBefore); err == nil {
			filter.CreatedBefore = &t
		}
	}
}

func (s *Server) buildProjectSummaries(projects []*types.Project) []project.ProjectSummary {
	var summaries []project.ProjectSummary

	for _, proj := range projects {
		summary := s.buildProjectSummary(proj)
		summaries = append(summaries, summary)
	}

	return summaries
}

func (s *Server) buildProjectSummary(proj *types.Project) project.ProjectSummary {
	activeInstances := s.calculateActiveInstances()
	totalCost := s.calculateProjectCost(proj.ID)

	summary := project.ProjectSummary{
		ID:              proj.ID,
		Name:            proj.Name,
		Owner:           proj.Owner,
		Status:          proj.Status,
		MemberCount:     len(proj.Members),
		ActiveInstances: activeInstances,
		TotalCost:       totalCost,
		CreatedAt:       proj.CreatedAt,
		LastActivity:    proj.UpdatedAt,
	}

	if proj.Budget != nil {
		summary.BudgetStatus = s.buildBudgetStatusSummary(proj.Budget)
	}

	return summary
}

func (s *Server) calculateActiveInstances() int {
	activeInstances := 0
	if instances, err := s.awsManager.ListInstances(); err == nil {
		for _, instance := range instances {
			if instance.State == "running" {
				// TODO: Implement proper project-instance association
				activeInstances++
			}
		}
	}
	return activeInstances
}

func (s *Server) calculateProjectCost(projectID string) float64 {
	if s.budgetTracker == nil {
		return 0.0
	}

	budgetStatus, err := s.budgetTracker.CheckBudgetStatus(projectID)
	if err != nil || !budgetStatus.BudgetEnabled {
		return 0.0
	}

	return budgetStatus.SpentAmount
}

func (s *Server) buildBudgetStatusSummary(budget *types.ProjectBudget) *project.BudgetStatusSummary {
	return &project.BudgetStatusSummary{
		TotalBudget:     budget.TotalBudget,
		SpentAmount:     budget.SpentAmount,
		SpentPercentage: budget.SpentAmount / budget.TotalBudget,
		AlertCount:      len(budget.AlertThresholds),
	}
}

// handleCreateProject creates a new project
func (s *Server) handleCreateProject(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		s.writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	var req project.CreateProjectRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, fmt.Sprintf("Invalid JSON: %v", err))
		return
	}

	ctx := context.Background()
	proj, err := s.projectManager.CreateProject(ctx, &req)
	if err != nil {
		s.writeError(w, http.StatusBadRequest, fmt.Sprintf("Failed to create project: %v", err))
		return
	}

	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(proj)
}

// handleGetProject retrieves a specific project
func (s *Server) handleGetProject(w http.ResponseWriter, r *http.Request, projectID string) {
	ctx := context.Background()
	proj, err := s.projectManager.GetProject(ctx, projectID)
	if err != nil {
		s.writeError(w, http.StatusNotFound, fmt.Sprintf("Project not found: %v", err))
		return
	}

	_ = json.NewEncoder(w).Encode(proj)
}

// handleUpdateProject updates a project
func (s *Server) handleUpdateProject(w http.ResponseWriter, r *http.Request, projectID string) {
	var req project.UpdateProjectRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, fmt.Sprintf("Invalid JSON: %v", err))
		return
	}

	ctx := context.Background()
	proj, err := s.projectManager.UpdateProject(ctx, projectID, &req)
	if err != nil {
		s.writeError(w, http.StatusBadRequest, fmt.Sprintf("Failed to update project: %v", err))
		return
	}

	_ = json.NewEncoder(w).Encode(proj)
}

// handleDeleteProject deletes a project
func (s *Server) handleDeleteProject(w http.ResponseWriter, r *http.Request, projectID string) {
	ctx := context.Background()
	if err := s.projectManager.DeleteProject(ctx, projectID); err != nil {
		s.writeError(w, http.StatusBadRequest, fmt.Sprintf("Failed to delete project: %v", err))
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// handleProjectMembers manages project members
func (s *Server) handleProjectMembers(w http.ResponseWriter, r *http.Request, projectID string, parts []string) {
	// parts structure: [projectID, "members", userID (optional)]
	var userID string
	if len(parts) > 2 {
		userID = parts[2]
	}

	switch r.Method {
	case http.MethodGet:
		s.handleGetProjectMembers(w, r, projectID)
	case http.MethodPost:
		s.handleAddProjectMember(w, r, projectID)
	case http.MethodPut:
		if userID == "" {
			s.writeError(w, http.StatusBadRequest, "User ID required for member update")
			return
		}
		s.handleUpdateProjectMember(w, r, projectID, userID)
	case http.MethodDelete:
		if userID == "" {
			s.writeError(w, http.StatusBadRequest, "User ID required for member removal")
			return
		}
		s.handleRemoveProjectMember(w, r, projectID, userID)
	default:
		s.writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
	}
}

// handleGetProjectMembers retrieves project members
func (s *Server) handleGetProjectMembers(w http.ResponseWriter, r *http.Request, projectID string) {
	ctx := context.Background()
	proj, err := s.projectManager.GetProject(ctx, projectID)
	if err != nil {
		s.writeError(w, http.StatusNotFound, fmt.Sprintf("Project not found: %v", err))
		return
	}

	_ = json.NewEncoder(w).Encode(proj.Members)
}

// handleAddProjectMember adds a member to a project
func (s *Server) handleAddProjectMember(w http.ResponseWriter, r *http.Request, projectID string) {
	var req project.AddMemberRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, fmt.Sprintf("Invalid JSON: %v", err))
		return
	}

	if err := req.Validate(); err != nil {
		s.writeError(w, http.StatusBadRequest, fmt.Sprintf("Invalid request: %v", err))
		return
	}

	member := &types.ProjectMember{
		UserID:  req.UserID,
		Role:    req.Role,
		AddedBy: req.AddedBy,
	}

	ctx := context.Background()
	if err := s.projectManager.AddProjectMember(ctx, projectID, member); err != nil {
		s.writeError(w, http.StatusBadRequest, fmt.Sprintf("Failed to add member: %v", err))
		return
	}

	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(member)
}

// handleUpdateProjectMember updates a project member's role
func (s *Server) handleUpdateProjectMember(w http.ResponseWriter, r *http.Request, projectID, userID string) {
	var req project.UpdateMemberRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, fmt.Sprintf("Invalid JSON: %v", err))
		return
	}

	if err := req.Validate(); err != nil {
		s.writeError(w, http.StatusBadRequest, fmt.Sprintf("Invalid request: %v", err))
		return
	}

	ctx := context.Background()
	if err := s.projectManager.UpdateProjectMember(ctx, projectID, userID, req.Role); err != nil {
		s.writeError(w, http.StatusBadRequest, fmt.Sprintf("Failed to update member: %v", err))
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// handleRemoveProjectMember removes a member from a project
func (s *Server) handleRemoveProjectMember(w http.ResponseWriter, r *http.Request, projectID, userID string) {
	ctx := context.Background()
	if err := s.projectManager.RemoveProjectMember(ctx, projectID, userID); err != nil {
		s.writeError(w, http.StatusBadRequest, fmt.Sprintf("Failed to remove member: %v", err))
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// handleProjectBudget manages project budget information
func (s *Server) handleProjectBudget(w http.ResponseWriter, r *http.Request, projectID string) {
	switch r.Method {
	case http.MethodGet:
		s.handleGetProjectBudgetStatus(w, r, projectID)
	default:
		s.writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
	}
}

// handleGetProjectBudgetStatus retrieves budget status for a project
func (s *Server) handleGetProjectBudgetStatus(w http.ResponseWriter, r *http.Request, projectID string) {
	ctx := context.Background()
	budgetStatus, err := s.projectManager.CheckBudgetStatus(ctx, projectID)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to get budget status: %v", err))
		return
	}

	_ = json.NewEncoder(w).Encode(budgetStatus)
}

// handleProjectCosts manages project cost analysis
func (s *Server) handleProjectCosts(w http.ResponseWriter, r *http.Request, projectID string) {
	if r.Method != http.MethodGet {
		s.writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	// Parse date range parameters
	startDate := time.Now().AddDate(0, -1, 0) // Default to last month
	endDate := time.Now()

	if startStr := r.URL.Query().Get("start_date"); startStr != "" {
		if t, err := time.Parse(time.RFC3339, startStr); err == nil {
			startDate = t
		}
	}

	if endStr := r.URL.Query().Get("end_date"); endStr != "" {
		if t, err := time.Parse(time.RFC3339, endStr); err == nil {
			endDate = t
		}
	}

	ctx := context.Background()
	costBreakdown, err := s.projectManager.GetProjectCostBreakdown(ctx, projectID, startDate, endDate)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to get cost breakdown: %v", err))
		return
	}

	_ = json.NewEncoder(w).Encode(costBreakdown)
}

// handleProjectUsage manages project resource usage metrics
func (s *Server) handleProjectUsage(w http.ResponseWriter, r *http.Request, projectID string) {
	if r.Method != http.MethodGet {
		s.writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	// Parse period parameter
	period := time.Hour * 24 * 30 // Default to 30 days

	if periodStr := r.URL.Query().Get("period"); periodStr != "" {
		if d, err := time.ParseDuration(periodStr); err == nil {
			period = d
		}
	}

	ctx := context.Background()
	usage, err := s.projectManager.GetProjectResourceUsage(ctx, projectID, period)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to get resource usage: %v", err))
		return
	}

	_ = json.NewEncoder(w).Encode(usage)
}
