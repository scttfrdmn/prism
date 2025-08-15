package daemon

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/scttfrdmn/cloudworkstation/pkg/templates"
	"github.com/scttfrdmn/cloudworkstation/pkg/types"
)

// handleTemplateApply handles applying templates to running instances
func (s *Server) handleTemplateApply(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		s.writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	var req templates.ApplyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, "Invalid request body: "+err.Error())
		return
	}

	// Validate request
	if req.InstanceName == "" {
		s.writeError(w, http.StatusBadRequest, "instance_name is required")
		return
	}
	if req.Template == nil {
		s.writeError(w, http.StatusBadRequest, "template is required")
		return
	}

	// Check if instance exists and is running
	state, err := s.stateManager.LoadState()
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "Failed to load state: "+err.Error())
		return
	}

	instance, exists := state.Instances[req.InstanceName]
	if !exists {
		s.writeError(w, http.StatusNotFound, "Instance not found: "+req.InstanceName)
		return
	}

	if instance.State != "running" {
		s.writeError(w, http.StatusBadRequest, "Instance must be running to apply templates")
		return
	}

	// Create template application engine with appropriate executor
	executor, err := s.createRemoteExecutor(instance)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "Failed to create remote executor: "+err.Error())
		return
	}

	engine := templates.NewTemplateApplicationEngine(executor)

	// Apply template
	response, err := engine.ApplyTemplate(r.Context(), req)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "Failed to apply template: "+err.Error())
		return
	}

	// Update instance state with applied template
	if !req.DryRun && response.Success {
		err = s.recordTemplateApplication(req.InstanceName, req.Template, response.RollbackCheckpoint)
		if err != nil {
			// Log warning but don't fail the operation
			fmt.Printf("Warning: failed to record template application: %v\n", err)
		}
	}

	json.NewEncoder(w).Encode(response)
}

// handleTemplateDiff handles calculating template differences
func (s *Server) handleTemplateDiff(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		s.writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	var req templates.DiffRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, "Invalid request body: "+err.Error())
		return
	}

	// Validate request
	if req.InstanceName == "" {
		s.writeError(w, http.StatusBadRequest, "instance_name is required")
		return
	}
	if req.Template == nil {
		s.writeError(w, http.StatusBadRequest, "template is required")
		return
	}

	// Check if instance exists and is running
	state, err := s.stateManager.LoadState()
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "Failed to load state: "+err.Error())
		return
	}

	instance, exists := state.Instances[req.InstanceName]
	if !exists {
		s.writeError(w, http.StatusNotFound, "Instance not found: "+req.InstanceName)
		return
	}

	if instance.State != "running" {
		s.writeError(w, http.StatusBadRequest, "Instance must be running to calculate template diff")
		return
	}

	// Create template application engine
	executor, err := s.createRemoteExecutor(instance)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "Failed to create remote executor: "+err.Error())
		return
	}

	// Create components for diff calculation
	stateInspector := templates.NewInstanceStateInspector(executor)
	diffCalculator := templates.NewTemplateDiffCalculator()

	// Inspect current instance state
	currentState, err := stateInspector.InspectInstance(r.Context(), req.InstanceName)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "Failed to inspect instance state: "+err.Error())
		return
	}

	// Calculate template differences
	diff, err := diffCalculator.CalculateDiff(currentState, req.Template)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "Failed to calculate template diff: "+err.Error())
		return
	}

	json.NewEncoder(w).Encode(diff)
}

// handleInstanceLayers handles listing applied template layers for an instance
func (s *Server) handleInstanceLayers(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	// Extract instance name from path: /api/v1/instances/{name}/layers
	path := strings.TrimPrefix(r.URL.Path, "/api/v1/instances/")
	pathParts := strings.Split(path, "/")
	if len(pathParts) != 2 || pathParts[1] != "layers" {
		s.writeError(w, http.StatusBadRequest, "Invalid path format")
		return
	}
	instanceName := pathParts[0]

	// Check if instance exists
	state, err := s.stateManager.LoadState()
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "Failed to load state: "+err.Error())
		return
	}

	instance, exists := state.Instances[instanceName]
	if !exists {
		s.writeError(w, http.StatusNotFound, "Instance not found: "+instanceName)
		return
	}

	// Get applied templates from instance state
	appliedTemplates := []templates.AppliedTemplate{}

	// Check if instance has applied templates field
	if len(instance.AppliedTemplates) > 0 {
		for _, applied := range instance.AppliedTemplates {
			appliedTemplates = append(appliedTemplates, templates.AppliedTemplate{
				Name:               applied.TemplateName,
				AppliedAt:          applied.AppliedAt,
				PackageManager:     applied.PackageManager,
				PackagesInstalled:  applied.PackagesInstalled,
				ServicesConfigured: applied.ServicesConfigured,
				UsersCreated:       applied.UsersCreated,
				RollbackCheckpoint: applied.RollbackCheckpoint,
			})
		}
	}

	json.NewEncoder(w).Encode(appliedTemplates)
}

// handleInstanceRollback handles rolling back template applications
func (s *Server) handleInstanceRollback(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		s.writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	var req types.RollbackRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, "Invalid request body: "+err.Error())
		return
	}

	// Validate request
	if req.InstanceName == "" {
		s.writeError(w, http.StatusBadRequest, "instance_name is required")
		return
	}
	if req.CheckpointID == "" {
		s.writeError(w, http.StatusBadRequest, "checkpoint_id is required")
		return
	}

	// Check if instance exists and is running
	state, err := s.stateManager.LoadState()
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "Failed to load state: "+err.Error())
		return
	}

	instance, exists := state.Instances[req.InstanceName]
	if !exists {
		s.writeError(w, http.StatusNotFound, "Instance not found: "+req.InstanceName)
		return
	}

	if instance.State != "running" {
		s.writeError(w, http.StatusBadRequest, "Instance must be running to perform rollback")
		return
	}

	// Create rollback manager
	executor, err := s.createRemoteExecutor(instance)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "Failed to create remote executor: "+err.Error())
		return
	}

	rollbackManager := templates.NewTemplateRollbackManager(executor)

	// Perform rollback
	err = rollbackManager.RollbackToCheckpoint(r.Context(), req.InstanceName, req.CheckpointID)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "Failed to rollback instance: "+err.Error())
		return
	}

	// Update instance state to remove template applications after the checkpoint
	err = s.removeTemplateApplicationsAfterCheckpoint(req.InstanceName, req.CheckpointID)
	if err != nil {
		// Log warning but don't fail the operation
		fmt.Printf("Warning: failed to update instance state after rollback: %v\n", err)
	}

	// Return success response
	response := map[string]interface{}{
		"success": true,
		"message": fmt.Sprintf("Successfully rolled back instance '%s' to checkpoint '%s'", req.InstanceName, req.CheckpointID),
	}
	json.NewEncoder(w).Encode(response)
}

// createRemoteExecutor creates an appropriate remote executor for the instance
func (s *Server) createRemoteExecutor(instance types.Instance) (templates.RemoteExecutor, error) {
	// Determine the best connection method based on instance configuration
	if instance.PublicIP != "" {
		// Use SSH for instances with public IPs
		keyPath := s.getSSHKeyPath()
		username := s.getSSHUsername(instance)

		return templates.NewSSHRemoteExecutor(keyPath, username), nil
	} else {
		// Use Systems Manager for private instances
		region := s.getAWSRegion()

		return templates.NewSystemsManagerExecutor(region), nil
	}
}

// getSSHKeyPath returns the path to the SSH key for instance connections
func (s *Server) getSSHKeyPath() string {
	// In a full implementation, this would:
	// 1. Check for key in CloudWorkstation config
	// 2. Look for default AWS key pairs
	// 3. Handle per-instance key management

	// For now, return a placeholder that would be configured
	return "~/.ssh/cloudworkstation-key.pem"
}

// getSSHUsername returns the appropriate SSH username for the instance
func (s *Server) getSSHUsername(instance types.Instance) string {
	// Determine username based on instance template/AMI
	// Common usernames: ubuntu, ec2-user, centos, admin

	// For now, use ubuntu as default (most common for research workstations)
	return "ubuntu"
}

// getAWSRegion returns the AWS region for Systems Manager connections

// recordTemplateApplication records a successful template application in instance state
func (s *Server) recordTemplateApplication(instanceName string, template *templates.Template, checkpointID string) error {
	state, err := s.stateManager.LoadState()
	if err != nil {
		return err
	}

	instance, exists := state.Instances[instanceName]
	if !exists {
		return fmt.Errorf("instance not found: %s", instanceName)
	}

	// Create applied template record
	appliedTemplate := types.AppliedTemplateRecord{
		TemplateName:       template.Name,
		AppliedAt:          time.Now(),
		PackageManager:     template.PackageManager,
		PackagesInstalled:  getPackageNames(template),
		ServicesConfigured: getServiceNames(template),
		UsersCreated:       getUserNames(template),
		RollbackCheckpoint: checkpointID,
	}

	// Add to instance's applied templates
	instance.AppliedTemplates = append(instance.AppliedTemplates, appliedTemplate)
	state.Instances[instanceName] = instance

	// Save updated state
	return s.stateManager.SaveState(state)
}

// removeTemplateApplicationsAfterCheckpoint removes template applications after a specific checkpoint
func (s *Server) removeTemplateApplicationsAfterCheckpoint(instanceName, checkpointID string) error {
	state, err := s.stateManager.LoadState()
	if err != nil {
		return err
	}

	instance, exists := state.Instances[instanceName]
	if !exists {
		return fmt.Errorf("instance not found: %s", instanceName)
	}

	// Find the checkpoint index
	checkpointIndex := -1
	for i, applied := range instance.AppliedTemplates {
		if applied.RollbackCheckpoint == checkpointID {
			checkpointIndex = i
			break
		}
	}

	if checkpointIndex == -1 {
		return fmt.Errorf("checkpoint not found: %s", checkpointID)
	}

	// Remove all template applications after the checkpoint
	instance.AppliedTemplates = instance.AppliedTemplates[:checkpointIndex+1]
	state.Instances[instanceName] = instance

	// Save updated state
	return s.stateManager.SaveState(state)
}

// Helper functions to extract names from template
func getPackageNames(template *templates.Template) []string {
	var packages []string
	packages = append(packages, template.Packages.System...)
	packages = append(packages, template.Packages.Conda...)
	packages = append(packages, template.Packages.Spack...)
	packages = append(packages, template.Packages.Pip...)
	return packages
}

func getServiceNames(template *templates.Template) []string {
	var services []string
	for _, svc := range template.Services {
		services = append(services, svc.Name)
	}
	return services
}

func getUserNames(template *templates.Template) []string {
	var users []string
	for _, user := range template.Users {
		users = append(users, user.Name)
	}
	return users
}
