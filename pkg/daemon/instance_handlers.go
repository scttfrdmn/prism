package daemon

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/scttfrdmn/cloudworkstation/pkg/aws"
	"github.com/scttfrdmn/cloudworkstation/pkg/profile"
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
	
	s.withAWSManager(w, r, func(awsManager *aws.Manager) error {
		var err error
		instances, err = awsManager.ListInstances()
		return err
	})
	
	// If AWS call failed, the withAWSManager already wrote the error response
	if instances == nil {
		return
	}

	// Filter out terminated instances older than 5 minutes (if we have deletion time)
	filteredInstances := make([]types.Instance, 0)
	for _, instance := range instances {
		// Include non-terminated instances
		if instance.State != "terminated" {
			filteredInstances = append(filteredInstances, instance)
			continue
		}
		
		// For terminated instances, check deletion time
		if instance.DeletionTime != nil {
			// Include if less than 5 minutes since deletion was initiated
			if time.Since(*instance.DeletionTime) < 5*time.Minute {
				filteredInstances = append(filteredInstances, instance)
			}
			// Otherwise, exclude (older than 5 minutes)
		} else {
			// No deletion time recorded - assume terminated instances older than 5 minutes should be cleaned up
			// Use a conservative approach: if terminated for more than 5 minutes based on launch time + reasonable startup time
			// This handles legacy instances without deletion timestamps
			timeSinceLaunch := time.Since(instance.LaunchTime)
			if timeSinceLaunch < 10*time.Minute { // Conservative: assume startup + 5min retention
				filteredInstances = append(filteredInstances, instance)
			}
			// Otherwise, exclude old terminated instances without deletion timestamps
		}
	}

	// Calculate total cost for running instances
	for _, instance := range filteredInstances {
		if instance.State == "running" {
			totalCost += instance.EstimatedDailyCost
		}
	}

	response := types.ListResponse{
		Instances: filteredInstances,
		TotalCost: totalCost,
	}

	json.NewEncoder(w).Encode(response)
}

// handleLaunchInstance launches a new instance
func (s *Server) handleLaunchInstance(w http.ResponseWriter, r *http.Request) {
	var req types.LaunchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	
	// Validate that instance name is unique
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
		return
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
		
		// Delegate to AWS manager
		var err error
		instance, err = awsManager.LaunchInstance(req)
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
		EstimatedCost:  fmt.Sprintf("$%.2f/day", instance.EstimatedDailyCost),
		ConnectionInfo: fmt.Sprintf("ssh ubuntu@%s", instance.PublicIP),
	}

	json.NewEncoder(w).Encode(response)
}

// handleInstanceOperations handles operations on specific instances
func (s *Server) handleInstanceOperations(w http.ResponseWriter, r *http.Request) {
	// Parse instance name from path
	path := r.URL.Path[len("/api/v1/instances/"):]
	parts := splitPath(path)
	if len(parts) == 0 {
		s.writeError(w, http.StatusBadRequest, "Missing instance name")
		return
	}

	instanceName := parts[0]

	if len(parts) == 1 {
		// Operations on the instance itself
		switch r.Method {
		case http.MethodGet:
			s.handleGetInstance(w, r, instanceName)
		case http.MethodDelete:
			s.handleDeleteInstance(w, r, instanceName)
		default:
			s.writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		}
	} else if len(parts) == 2 {
		// Sub-operations
		operation := parts[1]
		switch operation {
		case "start":
			s.handleStartInstance(w, r, instanceName)
		case "stop":
			s.handleStopInstance(w, r, instanceName)
		case "hibernate":
			s.handleHibernateInstance(w, r, instanceName)
		case "resume":
			s.handleResumeInstance(w, r, instanceName)
		case "hibernation-status":
			s.handleInstanceHibernationStatus(w, r, instanceName)
		case "connect":
			s.handleConnectInstance(w, r, instanceName)
		case "layers":
			s.handleInstanceLayers(w, r)
		case "rollback":
			s.handleInstanceRollback(w, r)
		default:
			s.writeError(w, http.StatusNotFound, "Unknown operation")
		}
	} else {
		s.writeError(w, http.StatusNotFound, "Invalid path")
	}
}

// handleGetInstance gets details of a specific instance
func (s *Server) handleGetInstance(w http.ResponseWriter, r *http.Request, name string) {
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

	json.NewEncoder(w).Encode(instance)
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

	var hibernationSupported, isHibernated bool
	s.withAWSManager(w, r, func(awsManager *aws.Manager) error {
		var err error
		hibernationSupported, isHibernated, err = awsManager.GetInstanceHibernationStatus(name)
		return err
	})

	response := map[string]interface{}{
		"hibernation_supported": hibernationSupported,
		"is_hibernated":        isHibernated,
		"instance_name":        name,
	}
	
	json.NewEncoder(w).Encode(response)
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
	json.NewEncoder(w).Encode(response)
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
func (s *Server) ensureSSHKeyInAWS(awsManager *aws.Manager, req *types.LaunchRequest) error {
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