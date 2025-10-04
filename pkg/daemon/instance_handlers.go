package daemon

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/scttfrdmn/cloudworkstation/pkg/aws"
	"github.com/scttfrdmn/cloudworkstation/pkg/profile"
	"github.com/scttfrdmn/cloudworkstation/pkg/templates"
	"github.com/scttfrdmn/cloudworkstation/pkg/types"
)

// handleInstances handles instance collection operations
func (s *Server) handleInstances(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.handleListInstances(w, r)
	case http.MethodPost:
		s.handleLaunchInstance(w, r)
	default:
		s.writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
	}
}

// handleListInstances lists all instances with real-time AWS status
func (s *Server) handleListInstances(w http.ResponseWriter, r *http.Request) {
	var instances []types.Instance
	totalCost := 0.0

	var awsErr error
	s.withAWSManager(w, r, func(awsManager *aws.Manager) error {
		var err error
		instances, err = awsManager.ListInstances()
		awsErr = err
		return err
	})

	// If AWS call failed, the withAWSManager already wrote the error response
	// Note: instances will be an empty slice when no instances exist, not nil
	if awsErr != nil {
		return
	}

	// Filter out terminated instances older than retention period (configurable)
	retentionDuration := s.config.GetRetentionDuration()
	filteredInstances := make([]types.Instance, 0)

	for _, instance := range instances {
		// Include non-terminated instances
		if instance.State != "terminated" {
			filteredInstances = append(filteredInstances, instance)
			continue
		}

		// For terminated instances, check deletion time against retention period
		if instance.DeletionTime != nil {
			// Include if less than retention period since deletion was initiated
			if time.Since(*instance.DeletionTime) < retentionDuration {
				filteredInstances = append(filteredInstances, instance)
			}
			// Otherwise, exclude (older than retention period)
		} else {
			// No deletion time recorded - use conservative approach for legacy instances
			// If retention is 0 (indefinite), always include terminated instances
			if s.config.InstanceRetentionMinutes == 0 {
				filteredInstances = append(filteredInstances, instance)
			} else {
				// Use launch time + startup buffer + retention period for legacy instances
				timeSinceLaunch := time.Since(instance.LaunchTime)
				conservativeRetention := (5 * time.Minute) + retentionDuration // 5min startup buffer
				if timeSinceLaunch < conservativeRetention {
					filteredInstances = append(filteredInstances, instance)
				}
			}
			// Otherwise, exclude old terminated instances without deletion timestamps
		}
	}

	// Calculate total cost for running instances
	for _, instance := range filteredInstances {
		if instance.State == "running" {
			// Use current spend to show actual accumulated cost
			totalCost += instance.CurrentSpend
		}
	}

	response := types.ListResponse{
		Instances: filteredInstances,
		TotalCost: totalCost,
	}

	_ = json.NewEncoder(w).Encode(response)
}

// handleLaunchInstance launches a new instance
func (s *Server) handleLaunchInstance(w http.ResponseWriter, r *http.Request) {
	var req types.LaunchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, "Invalid JSON request body")
		return
	}

	// Validate the launch request
	if err := s.validateLaunchRequest(&req, w); err != nil {
		return // Error response already written by validateLaunchRequest
	}

	// Check instance name uniqueness
	if s.checkInstanceNameUniqueness(&req, w, r) {
		return // Error response already written if name exists
	}

	// Handle SSH key management if not provided in request
	if req.SSHKeyName == "" {
		if err := s.setupSSHKeyForLaunch(&req); err != nil {
			s.writeError(w, http.StatusInternalServerError, fmt.Sprintf("SSH key setup failed: %v", err))
			return
		}
	}

	// Use AWS manager from request and handle launch
	var instance *types.Instance
	s.withAWSManager(w, r, func(awsManager *aws.Manager) error {
		// Ensure SSH key exists in AWS if specified
		if req.SSHKeyName != "" {
			if err := s.ensureSSHKeyInAWS(awsManager, &req); err != nil {
				return fmt.Errorf("failed to ensure SSH key in AWS: %w", err)
			}
		}

		// Track launch start time
		launchStart := time.Now()

		// Delegate to AWS manager
		var err error
		instance, err = awsManager.LaunchInstance(req)

		// Record usage stats
		launchDuration := int(time.Since(launchStart).Seconds())
		templates.GetUsageStats().RecordLaunch(req.Template, err == nil, launchDuration)

		return err
	})

	// If instance is nil, withAWSManager already wrote an error response
	if instance == nil {
		return
	}

	// Save state
	if err := s.stateManager.SaveInstance(*instance); err != nil {
		s.writeError(w, http.StatusInternalServerError, "Failed to save instance state")
		return
	}

	response := types.LaunchResponse{
		Instance:       *instance,
		Message:        fmt.Sprintf("Instance %s launched successfully", instance.Name),
		EstimatedCost:  fmt.Sprintf("$%.3f/hr (effective: $%.3f/hr)", instance.HourlyRate, instance.EffectiveRate),
		ConnectionInfo: fmt.Sprintf("ssh ubuntu@%s", instance.PublicIP),
	}

	_ = json.NewEncoder(w).Encode(response)
}

// handleInstanceOperations handles operations on specific instances
func (s *Server) handleInstanceOperations(w http.ResponseWriter, r *http.Request) {
	instanceName, pathParts, err := s.parseInstancePath(r.URL.Path)
	if err != nil {
		s.writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	s.routeInstanceOperation(w, r, instanceName, pathParts)
}

func (s *Server) parseInstancePath(urlPath string) (string, []string, error) {
	path := urlPath[len("/api/v1/instances/"):]
	parts := splitPath(path)

	if len(parts) == 0 {
		return "", nil, fmt.Errorf("missing instance name")
	}

	return parts[0], parts, nil
}

func (s *Server) routeInstanceOperation(w http.ResponseWriter, r *http.Request, instanceName string, parts []string) {
	switch len(parts) {
	case 1:
		s.handleDirectInstanceOperation(w, r, instanceName)
	case 2:
		s.handleInstanceSubOperation(w, r, instanceName, parts[1])
	case 3, 4:
		if parts[1] == "idle" && parts[2] == "policies" {
			s.handleIdlePolicyOperation(w, r, instanceName, parts)
		} else {
			s.writeError(w, http.StatusNotFound, "Invalid path")
		}
	default:
		s.writeError(w, http.StatusNotFound, "Invalid path")
	}
}

func (s *Server) handleDirectInstanceOperation(w http.ResponseWriter, r *http.Request, instanceName string) {
	switch r.Method {
	case http.MethodGet:
		s.handleGetInstance(w, r, instanceName)
	case http.MethodDelete:
		s.handleDeleteInstance(w, r, instanceName)
	default:
		s.writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
	}
}

func (s *Server) handleInstanceSubOperation(w http.ResponseWriter, r *http.Request, instanceName, operation string) {
	operationHandlers := map[string]func(http.ResponseWriter, *http.Request, string){
		"start":              s.handleStartInstance,
		"stop":               s.handleStopInstance,
		"hibernate":          s.handleHibernateInstance,
		"resume":             s.handleResumeInstance,
		"hibernation-status": s.handleInstanceHibernationStatus,
		"connect":            s.handleConnectInstance,
	}

	if handler, exists := operationHandlers[operation]; exists {
		handler(w, r, instanceName)
		return
	}

	// Special case handlers that don't take instanceName
	switch operation {
	case "layers":
		s.handleInstanceLayers(w, r)
	case "rollback":
		s.handleInstanceRollback(w, r)
	default:
		s.writeError(w, http.StatusNotFound, "Unknown operation")
	}
}

func (s *Server) handleIdlePolicyOperation(w http.ResponseWriter, r *http.Request, instanceName string, parts []string) {
	if len(parts) == 3 {
		// GET /instances/{name}/idle/policies
		s.handleInstanceIdlePolicies(w, r, instanceName)
	} else if len(parts) == 4 {
		// PUT/DELETE /instances/{name}/idle/policies/{policyId}
		policyID := parts[3]
		s.handleInstanceIdlePolicy(w, r, instanceName, policyID)
	} else {
		s.writeError(w, http.StatusNotFound, "Unknown idle operation")
	}
}

// handleGetInstance gets details of a specific instance
func (s *Server) handleGetInstance(w http.ResponseWriter, _ *http.Request, name string) {
	state, err := s.stateManager.LoadState()
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "Failed to load state")
		return
	}

	instance, exists := state.Instances[name]
	if !exists {
		s.writeError(w, http.StatusNotFound, "Instance not found")
		return
	}

	_ = json.NewEncoder(w).Encode(instance)
}

// handleDeleteInstance deletes a specific instance
func (s *Server) handleDeleteInstance(w http.ResponseWriter, r *http.Request, name string) {
	// Mark deletion timestamp before initiating AWS deletion
	now := time.Now()
	state, err := s.stateManager.LoadState()
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "Failed to load state")
		return
	}

	if instance, exists := state.Instances[name]; exists {
		instance.DeletionTime = &now
		if err := s.stateManager.SaveInstance(instance); err != nil {
			s.writeError(w, http.StatusInternalServerError, "Failed to update instance state")
			return
		}
	}

	// Initiate AWS deletion
	s.withAWSManager(w, r, func(awsManager *aws.Manager) error {
		return awsManager.DeleteInstance(name)
	})

	w.WriteHeader(http.StatusNoContent)
}

// handleStartInstance starts a stopped instance
func (s *Server) handleStartInstance(w http.ResponseWriter, r *http.Request, name string) {
	if r.Method != http.MethodPost {
		s.writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	s.withAWSManager(w, r, func(awsManager *aws.Manager) error {
		return awsManager.StartInstance(name)
	})

	w.WriteHeader(http.StatusNoContent)
}

// handleStopInstance stops a running instance
func (s *Server) handleStopInstance(w http.ResponseWriter, r *http.Request, name string) {
	if r.Method != http.MethodPost {
		s.writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	// Validate instance exists before attempting AWS operation
	state, err := s.stateManager.LoadState()
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "Failed to load state")
		return
	}

	if _, exists := state.Instances[name]; !exists {
		s.writeError(w, http.StatusNotFound, "Instance not found")
		return
	}

	s.withAWSManager(w, r, func(awsManager *aws.Manager) error {
		return awsManager.StopInstance(name)
	})

	w.WriteHeader(http.StatusNoContent)
}

// handleHibernateInstance hibernates a running instance
func (s *Server) handleHibernateInstance(w http.ResponseWriter, r *http.Request, name string) {
	if r.Method != http.MethodPost {
		s.writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	// Validate instance exists before attempting AWS operation
	state, err := s.stateManager.LoadState()
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "Failed to load state")
		return
	}

	if _, exists := state.Instances[name]; !exists {
		s.writeError(w, http.StatusNotFound, "Instance not found")
		return
	}

	s.withAWSManager(w, r, func(awsManager *aws.Manager) error {
		return awsManager.HibernateInstance(name)
	})

	w.WriteHeader(http.StatusNoContent)
}

// handleResumeInstance resumes a hibernated instance
func (s *Server) handleResumeInstance(w http.ResponseWriter, r *http.Request, name string) {
	if r.Method != http.MethodPost {
		s.writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	s.withAWSManager(w, r, func(awsManager *aws.Manager) error {
		return awsManager.ResumeInstance(name)
	})

	w.WriteHeader(http.StatusNoContent)
}

// handleInstanceHibernationStatus gets hibernation status for an instance
func (s *Server) handleInstanceHibernationStatus(w http.ResponseWriter, r *http.Request, name string) {
	if r.Method != http.MethodGet {
		s.writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	var hibernationSupported bool
	var instanceState string
	var possiblyHibernated bool

	s.withAWSManager(w, r, func(awsManager *aws.Manager) error {
		var err error
		hibernationSupported, instanceState, possiblyHibernated, err = awsManager.GetInstanceHibernationStatus(name)
		return err
	})

	response := map[string]interface{}{
		"hibernation_supported": hibernationSupported,
		"instance_state":        instanceState,
		"possibly_hibernated":   possiblyHibernated,
		"instance_name":         name,
		"is_hibernated":         possiblyHibernated, // Deprecated field for backward compatibility
		"note":                  "possibly_hibernated is true when instance is stopped and hibernation is supported",
	}

	_ = json.NewEncoder(w).Encode(response)
}

// handleConnectInstance gets connection information for an instance
func (s *Server) handleConnectInstance(w http.ResponseWriter, r *http.Request, name string) {
	if r.Method != http.MethodGet {
		s.writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	var connectionInfo string
	s.withAWSManager(w, r, func(awsManager *aws.Manager) error {
		var err error
		connectionInfo, err = awsManager.GetConnectionInfo(name)
		return err
	})

	if connectionInfo == "" {
		// Error was already handled by withAWSManager
		return
	}

	response := map[string]string{
		"connection_info": connectionInfo,
	}
	_ = json.NewEncoder(w).Encode(response)
}

// setupSSHKeyForLaunch sets up SSH key configuration for a launch request
func (s *Server) setupSSHKeyForLaunch(req *types.LaunchRequest) error {
	// Get current profile (this would be extracted from request context in production)
	profileManager, err := profile.NewManagerEnhanced()
	if err != nil {
		return fmt.Errorf("failed to create profile manager: %w", err)
	}

	currentProfile, err := profileManager.GetCurrentProfile()
	if err != nil {
		return fmt.Errorf("failed to get current profile: %w", err)
	}

	// Create SSH key manager
	sshKeyManager, err := profile.NewSSHKeyManager()
	if err != nil {
		return fmt.Errorf("failed to create SSH key manager: %w", err)
	}

	// Get SSH key configuration for current profile
	_, keyName, err := sshKeyManager.GetSSHKeyForProfile(currentProfile)
	if err != nil {
		return fmt.Errorf("failed to get SSH key for profile: %w", err)
	}

	// Set SSH key in launch request
	req.SSHKeyName = keyName

	return nil
}

// ensureSSHKeyInAWS ensures the SSH key exists in AWS
func (s *Server) ensureSSHKeyInAWS(awsManager *aws.Manager, _ *types.LaunchRequest) error {
	// Get current profile
	profileManager, err := profile.NewManagerEnhanced()
	if err != nil {
		return fmt.Errorf("failed to create profile manager: %w", err)
	}

	currentProfile, err := profileManager.GetCurrentProfile()
	if err != nil {
		return fmt.Errorf("failed to get current profile: %w", err)
	}

	// Create SSH key manager
	sshKeyManager, err := profile.NewSSHKeyManager()
	if err != nil {
		return fmt.Errorf("failed to create SSH key manager: %w", err)
	}

	// Get SSH key configuration
	keyPath, keyName, err := sshKeyManager.GetSSHKeyForProfile(currentProfile)
	if err != nil {
		return fmt.Errorf("failed to get SSH key for profile: %w", err)
	}

	// Get public key content
	publicKeyPath := keyPath + ".pub"
	publicKeyContent, err := sshKeyManager.GetPublicKeyContent(publicKeyPath)
	if err != nil {
		return fmt.Errorf("failed to get public key content: %w", err)
	}

	// Ensure key exists in AWS
	if err := awsManager.EnsureKeyPairExists(keyName, publicKeyContent); err != nil {
		return fmt.Errorf("failed to ensure key pair exists in AWS: %w", err)
	}

	return nil
}

// validateLaunchRequest validates the launch request and writes error response if needed
// Returns nil if validation passes, error if validation fails (response already written)
func (s *Server) validateLaunchRequest(req *types.LaunchRequest, w http.ResponseWriter) error {
	// Validate required fields
	if req.Template == "" {
		s.writeError(w, http.StatusBadRequest, "Missing required field: template")
		return fmt.Errorf("missing template")
	}

	if req.Name == "" {
		s.writeError(w, http.StatusBadRequest, "Missing required field: name")
		return fmt.Errorf("missing name")
	}

	// Validate instance size if provided
	if req.Size != "" {
		if err := s.validateInstanceSize(req.Size, w); err != nil {
			return err
		}
	}

	return nil
}

// validateInstanceSize validates the instance size parameter
func (s *Server) validateInstanceSize(size string, w http.ResponseWriter) error {
	validSizes := []string{"XS", "S", "M", "L", "XL", "GPU-S", "GPU-M", "GPU-L", "GPU-XL"}
	for _, validSize := range validSizes {
		if size == validSize {
			return nil
		}
	}

	s.writeError(w, http.StatusBadRequest, fmt.Sprintf("Invalid size '%s'. Valid sizes: %v", size, validSizes))
	return fmt.Errorf("invalid size")
}

// checkInstanceNameUniqueness checks if the instance name is already taken
// Returns true if name exists (not available), false if available
func (s *Server) checkInstanceNameUniqueness(req *types.LaunchRequest, w http.ResponseWriter, r *http.Request) bool {
	var nameExists bool
	s.withAWSManager(w, r, func(awsManager *aws.Manager) error {
		instances, err := awsManager.ListInstances()
		if err != nil {
			return fmt.Errorf("failed to check existing instances: %w", err)
		}

		for _, existingInstance := range instances {
			if existingInstance.Name == req.Name {
				// Check if instance is in a terminal state (terminated/terminating)
				if existingInstance.State != "terminated" && existingInstance.State != "terminating" {
					nameExists = true
					break
				}
			}
		}
		return nil
	})

	if nameExists {
		s.writeError(w, http.StatusConflict, fmt.Sprintf("Instance with name '%s' already exists. Please choose a different name.", req.Name))
		return true
	}
	return false
}
