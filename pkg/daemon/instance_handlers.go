package daemon

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/scttfrdmn/cloudworkstation/pkg/aws"
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

// handleListInstances lists all instances
func (s *Server) handleListInstances(w http.ResponseWriter, r *http.Request) {
	state, err := s.stateManager.LoadState()
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "Failed to load state")
		return
	}

	instances := make([]types.Instance, 0, len(state.Instances))
	totalCost := 0.0

	for _, instance := range state.Instances {
		instances = append(instances, instance)
		if instance.State == "running" {
			totalCost += instance.EstimatedDailyCost
		}
	}

	response := types.ListResponse{
		Instances: instances,
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
	
	// Use AWS manager from request and handle launch
	var instance *types.Instance
	s.withAWSManager(w, r, func(awsManager *aws.Manager) error {
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
	s.withAWSManager(w, r, func(awsManager *aws.Manager) error {
		return awsManager.DeleteInstance(name)
	})

	// Remove from state - only if we didn't error out above
	if err := s.stateManager.RemoveInstance(name); err != nil {
		s.writeError(w, http.StatusInternalServerError, "Failed to update state")
		return
	}

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