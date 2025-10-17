package daemon

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/scttfrdmn/cloudworkstation/pkg/aws"
	"github.com/scttfrdmn/cloudworkstation/pkg/types"
)

// TunnelInfo represents tunnel information for API responses
type TunnelInfo struct {
	InstanceName string `json:"instance_name"`
	ServiceName  string `json:"service_name"`
	ServiceDesc  string `json:"service_description"`
	RemotePort   int    `json:"remote_port"`
	LocalPort    int    `json:"local_port"`
	LocalURL     string `json:"local_url"`
	AuthToken    string `json:"auth_token,omitempty"` // Authentication token (e.g., Jupyter)
	Status       string `json:"status"`
	StartTime    string `json:"start_time,omitempty"`
}

// CreateTunnelsRequest is the request to create tunnels
type CreateTunnelsRequest struct {
	InstanceName string   `json:"instance_name"`
	Services     []string `json:"services,omitempty"` // If empty, create all
}

// CreateTunnelsResponse is the response from creating tunnels
type CreateTunnelsResponse struct {
	Tunnels []TunnelInfo `json:"tunnels"`
	Message string       `json:"message"`
}

// handleTunnels handles /api/v1/tunnels requests
func (s *Server) handleTunnels(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.handleListTunnels(w, r)
	case http.MethodPost:
		s.handleCreateTunnels(w, r)
	case http.MethodDelete:
		s.handleCloseTunnels(w, r)
	default:
		s.writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
	}
}

// handleListTunnels lists all active tunnels
func (s *Server) handleListTunnels(w http.ResponseWriter, r *http.Request) {
	// Get instance name from query parameter if provided
	instanceName := r.URL.Query().Get("instance")

	var tunnelInfos []TunnelInfo

	if instanceName != "" {
		// List tunnels for specific instance
		tunnels := s.tunnelManager.GetInstanceTunnels(instanceName)
		for _, tunnel := range tunnels {
			tunnelInfos = append(tunnelInfos, TunnelInfo{
				InstanceName: tunnel.InstanceName,
				ServiceName:  tunnel.ServiceName,
				RemotePort:   tunnel.RemotePort,
				LocalPort:    tunnel.LocalPort,
				LocalURL:     fmt.Sprintf("http://localhost:%d", tunnel.LocalPort),
				AuthToken:    tunnel.AuthToken,
				Status:       tunnel.status,
				StartTime:    tunnel.startTime.Format("2006-01-02T15:04:05Z07:00"),
			})
		}
	} else {
		// List all tunnels
		s.tunnelManager.mu.RLock()
		for _, tunnel := range s.tunnelManager.tunnels {
			tunnelInfos = append(tunnelInfos, TunnelInfo{
				InstanceName: tunnel.InstanceName,
				ServiceName:  tunnel.ServiceName,
				RemotePort:   tunnel.RemotePort,
				LocalPort:    tunnel.LocalPort,
				LocalURL:     fmt.Sprintf("http://localhost:%d", tunnel.LocalPort),
				AuthToken:    tunnel.AuthToken,
				Status:       tunnel.status,
				StartTime:    tunnel.startTime.Format("2006-01-02T15:04:05Z07:00"),
			})
		}
		s.tunnelManager.mu.RUnlock()
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"tunnels": tunnelInfos,
		"count":   len(tunnelInfos),
	})
}

// handleCreateTunnels creates tunnels for an instance
func (s *Server) handleCreateTunnels(w http.ResponseWriter, r *http.Request) {
	log.Printf("[DEBUG] handleCreateTunnels: START")

	var req CreateTunnelsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("[DEBUG] handleCreateTunnels: Invalid request body: %v", err)
		s.writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	log.Printf("[DEBUG] handleCreateTunnels: Parsed request - InstanceName=%s, Services=%v", req.InstanceName, req.Services)

	if req.InstanceName == "" {
		log.Printf("[DEBUG] handleCreateTunnels: instance_name is required")
		s.writeError(w, http.StatusBadRequest, "instance_name is required")
		return
	}

	// Get fresh instance data from AWS (includes KeyName for SSH)
	var instance types.Instance

	s.withAWSManager(w, r, func(awsManager *aws.Manager) error {
		instances, err := awsManager.ListInstances()
		if err != nil {
			log.Printf("[DEBUG] handleCreateTunnels: Failed to list instances: %v", err)
			return fmt.Errorf("failed to list instances: %w", err)
		}

		// Find the requested instance
		found := false
		for _, inst := range instances {
			if inst.Name == req.InstanceName {
				instance = inst
				found = true
				break
			}
		}

		if !found {
			log.Printf("[DEBUG] handleCreateTunnels: Instance not found: %s", req.InstanceName)
			return fmt.Errorf("instance not found: %s", req.InstanceName)
		}

		log.Printf("[DEBUG] handleCreateTunnels: Found instance - Name=%s, State=%s, IP=%s, KeyName=%s",
			instance.Name, instance.State, instance.PublicIP, instance.KeyName)
		return nil
	})

	// Check if instance is running
	if instance.State != "running" {
		s.writeError(w, http.StatusBadRequest, fmt.Sprintf("Instance must be running (current state: %s)", instance.State))
		return
	}

	// Get services to tunnel
	var servicesToTunnel []types.Service

	// DEBUG LOGGING
	log.Printf("[DEBUG] CreateTunnels: instance.Services=%v (len=%d)", instance.Services, len(instance.Services))
	log.Printf("[DEBUG] CreateTunnels: req.Services=%v (len=%d)", req.Services, len(req.Services))

	if len(req.Services) == 0 {
		// Create tunnels for all services
		servicesToTunnel = instance.Services
	} else {
		// Create tunnels for specified services only
		for _, svcName := range req.Services {
			for _, svc := range instance.Services {
				if svc.Name == svcName {
					servicesToTunnel = append(servicesToTunnel, svc)
					break
				}
			}
		}
	}

	log.Printf("[DEBUG] CreateTunnels: servicesToTunnel=%v (len=%d)", servicesToTunnel, len(servicesToTunnel))

	if len(servicesToTunnel) == 0 {
		s.writeError(w, http.StatusBadRequest, "No services found to tunnel")
		return
	}

	// Create tunnels
	var tunnelInfos []TunnelInfo
	var errors []string

	for _, service := range servicesToTunnel {
		log.Printf("[DEBUG] CreateTunnels: Creating tunnel for service %s (port %d)", service.Name, service.Port)
		tunnel, err := s.tunnelManager.CreateTunnel(&instance, service)
		if err != nil {
			log.Printf("[DEBUG] CreateTunnels: Failed to create tunnel for %s: %v", service.Name, err)
			errors = append(errors, fmt.Sprintf("%s: %v", service.Name, err))
			continue
		}
		log.Printf("[DEBUG] CreateTunnels: Tunnel created successfully: %+v", tunnel)

		tunnelInfos = append(tunnelInfos, TunnelInfo{
			InstanceName: tunnel.InstanceName,
			ServiceName:  tunnel.ServiceName,
			ServiceDesc:  service.Description,
			RemotePort:   tunnel.RemotePort,
			LocalPort:    tunnel.LocalPort,
			LocalURL:     fmt.Sprintf("http://localhost:%d", tunnel.LocalPort),
			AuthToken:    tunnel.AuthToken,
			Status:       tunnel.status,
			StartTime:    tunnel.startTime.Format("2006-01-02T15:04:05Z07:00"),
		})
	}

	log.Printf("[DEBUG] CreateTunnels: Created %d tunnels, %d errors", len(tunnelInfos), len(errors))

	// Build response
	response := CreateTunnelsResponse{
		Tunnels: tunnelInfos,
	}

	if len(errors) > 0 {
		response.Message = fmt.Sprintf("Created %d tunnels with %d errors: %s",
			len(tunnelInfos), len(errors), strings.Join(errors, "; "))
	} else {
		response.Message = fmt.Sprintf("Created %d tunnels successfully", len(tunnelInfos))
	}

	w.Header().Set("Content-Type", "application/json")
	if len(errors) > 0 && len(tunnelInfos) == 0 {
		w.WriteHeader(http.StatusInternalServerError)
	}
	_ = json.NewEncoder(w).Encode(response)
}

// handleCloseTunnels closes tunnels
func (s *Server) handleCloseTunnels(w http.ResponseWriter, r *http.Request) {
	instanceName := r.URL.Query().Get("instance")
	serviceName := r.URL.Query().Get("service")

	if instanceName == "" {
		s.writeError(w, http.StatusBadRequest, "instance parameter required")
		return
	}

	if serviceName != "" {
		// Close specific tunnel
		if err := s.tunnelManager.CloseTunnel(instanceName, serviceName); err != nil {
			s.writeError(w, http.StatusNotFound, err.Error())
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{
			"message": fmt.Sprintf("Closed tunnel for %s/%s", instanceName, serviceName),
		})
	} else {
		// Close all tunnels for instance
		s.tunnelManager.CloseInstanceTunnels(instanceName)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{
			"message": fmt.Sprintf("Closed all tunnels for %s", instanceName),
		})
	}
}
