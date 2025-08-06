// Security API handlers for CloudWorkstation daemon
package daemon

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/scttfrdmn/cloudworkstation/pkg/profile/security"
)

// handleSecurityStatus handles GET requests to /api/v1/security/status
func (s *Server) handleSecurityStatus(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		status, err := s.securityManager.GetSecurityStatus()
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to get security status: %v", err), http.StatusInternalServerError)
			return
		}

		// Log security status request
		s.securityManager.LogSecurityEvent("security_status_requested", true, "", map[string]interface{}{
			"client_ip": r.RemoteAddr,
			"user_agent": r.UserAgent(),
		})

		if err := json.NewEncoder(w).Encode(status); err != nil {
			http.Error(w, fmt.Sprintf("Failed to encode response: %v", err), http.StatusInternalServerError)
		}

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleSecurityHealth handles GET/POST requests to /api/v1/security/health  
func (s *Server) handleSecurityHealth(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		// Get current health status
		status, err := s.securityManager.GetSecurityStatus()
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to get security status: %v", err), http.StatusInternalServerError)
			return
		}

		healthResponse := map[string]interface{}{
			"system_health": status.SystemHealth,
			"keychain_info": status.KeychainInfo,
			"last_check": status.LastHealthCheck,
			"security_enabled": status.Enabled,
		}

		if err := json.NewEncoder(w).Encode(healthResponse); err != nil {
			http.Error(w, fmt.Sprintf("Failed to encode response: %v", err), http.StatusInternalServerError)
		}

	case "POST":
		// Trigger health check
		if err := s.securityManager.PerformHealthCheck(); err != nil {
			s.securityManager.LogSecurityEvent("health_check_triggered", false, "", map[string]interface{}{
				"error": err.Error(),
				"client_ip": r.RemoteAddr,
			})
			http.Error(w, fmt.Sprintf("Health check failed: %v", err), http.StatusInternalServerError)
			return
		}

		s.securityManager.LogSecurityEvent("health_check_triggered", true, "", map[string]interface{}{
			"client_ip": r.RemoteAddr,
		})

		response := map[string]interface{}{
			"status": "Health check completed successfully",
			"timestamp": "now",
		}

		if err := json.NewEncoder(w).Encode(response); err != nil {
			http.Error(w, fmt.Sprintf("Failed to encode response: %v", err), http.StatusInternalServerError)
		}

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleSecurityDashboard handles GET requests to /api/v1/security/dashboard
func (s *Server) handleSecurityDashboard(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		status, err := s.securityManager.GetSecurityStatus()
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to get security status: %v", err), http.StatusInternalServerError)
			return
		}

		if status.Dashboard == nil {
			http.Error(w, "Security dashboard not available", http.StatusServiceUnavailable)
			return
		}

		s.securityManager.LogSecurityEvent("security_dashboard_accessed", true, "", map[string]interface{}{
			"client_ip": r.RemoteAddr,
			"user_agent": r.UserAgent(),
		})

		if err := json.NewEncoder(w).Encode(status.Dashboard); err != nil {
			http.Error(w, fmt.Sprintf("Failed to encode response: %v", err), http.StatusInternalServerError)
		}

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleSecurityCorrelations handles GET requests to /api/v1/security/correlations
func (s *Server) handleSecurityCorrelations(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		status, err := s.securityManager.GetSecurityStatus()
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to get security status: %v", err), http.StatusInternalServerError)
			return
		}

		correlationResponse := map[string]interface{}{
			"correlations": status.Correlations,
			"correlation_count": len(status.Correlations),
			"correlation_enabled": status.Configuration.CorrelationEnabled,
		}

		s.securityManager.LogSecurityEvent("security_correlations_accessed", true, "", map[string]interface{}{
			"client_ip": r.RemoteAddr,
			"correlation_count": len(status.Correlations),
		})

		if err := json.NewEncoder(w).Encode(correlationResponse); err != nil {
			http.Error(w, fmt.Sprintf("Failed to encode response: %v", err), http.StatusInternalServerError)
		}

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleSecurityKeychain handles GET requests to /api/v1/security/keychain
func (s *Server) handleSecurityKeychain(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		// Get keychain information
		keychainInfo, err := security.GetKeychainInfo()
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to get keychain info: %v", err), http.StatusInternalServerError)
			return
		}

		// Get keychain diagnostics
		diagnostics := security.DiagnoseKeychainIssues()

		keychainResponse := map[string]interface{}{
			"info": keychainInfo,
			"diagnostics": diagnostics,
		}

		s.securityManager.LogSecurityEvent("keychain_info_accessed", true, "", map[string]interface{}{
			"client_ip": r.RemoteAddr,
			"provider": keychainInfo.Provider,
			"native": keychainInfo.Native,
		})

		if err := json.NewEncoder(w).Encode(keychainResponse); err != nil {
			http.Error(w, fmt.Sprintf("Failed to encode response: %v", err), http.StatusInternalServerError)
		}

	case "POST":
		// Validate keychain provider
		if err := security.ValidateKeychainProvider(); err != nil {
			s.securityManager.LogSecurityEvent("keychain_validation_failed", false, "", map[string]interface{}{
				"error": err.Error(),
				"client_ip": r.RemoteAddr,
			})
			http.Error(w, fmt.Sprintf("Keychain validation failed: %v", err), http.StatusInternalServerError)
			return
		}

		s.securityManager.LogSecurityEvent("keychain_validation_success", true, "", map[string]interface{}{
			"client_ip": r.RemoteAddr,
		})

		response := map[string]interface{}{
			"status": "Keychain validation successful",
			"timestamp": "now",
		}

		if err := json.NewEncoder(w).Encode(response); err != nil {
			http.Error(w, fmt.Sprintf("Failed to encode response: %v", err), http.StatusInternalServerError)
		}

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleSecurityConfig handles GET/PUT requests to /api/v1/security/config
func (s *Server) handleSecurityConfig(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		// Get current security configuration
		status, err := s.securityManager.GetSecurityStatus()
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to get security status: %v", err), http.StatusInternalServerError)
			return
		}

		configResponse := map[string]interface{}{
			"configuration": status.Configuration,
			"enabled": status.Enabled,
			"running": status.Running,
		}

		s.securityManager.LogSecurityEvent("security_config_accessed", true, "", map[string]interface{}{
			"client_ip": r.RemoteAddr,
		})

		if err := json.NewEncoder(w).Encode(configResponse); err != nil {
			http.Error(w, fmt.Sprintf("Failed to encode response: %v", err), http.StatusInternalServerError)
		}

	case "PUT":
		// Update security configuration (placeholder for future implementation)
		s.securityManager.LogSecurityEvent("security_config_update_attempted", false, "", map[string]interface{}{
			"client_ip": r.RemoteAddr,
			"reason": "not_implemented",
		})

		http.Error(w, "Security configuration updates not yet implemented", http.StatusNotImplemented)

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// Additional security handler for device registration (integrated with existing invitation system)
func (s *Server) handleSecureDeviceRegistration(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var registrationRequest struct {
		InvitationToken string `json:"invitation_token"`
		DeviceID        string `json:"device_id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&registrationRequest); err != nil {
		http.Error(w, fmt.Sprintf("Invalid request body: %v", err), http.StatusBadRequest)
		return
	}

	// Validate required fields
	if registrationRequest.InvitationToken == "" || registrationRequest.DeviceID == "" {
		http.Error(w, "Missing required fields: invitation_token and device_id", http.StatusBadRequest)
		return
	}

	// Register device using security manager
	if err := s.securityManager.RegisterDevice(registrationRequest.InvitationToken, registrationRequest.DeviceID); err != nil {
		s.securityManager.LogDeviceRegistration(registrationRequest.DeviceID, registrationRequest.InvitationToken, false, "registration_failed", map[string]interface{}{
			"error": err.Error(),
			"client_ip": r.RemoteAddr,
		})
		
		// Check if it's a security-related error
		if strings.Contains(err.Error(), "device binding") || strings.Contains(err.Error(), "tamper") {
			http.Error(w, "Device registration failed: security violation", http.StatusForbidden)
		} else {
			http.Error(w, fmt.Sprintf("Device registration failed: %v", err), http.StatusInternalServerError)
		}
		return
	}

	s.securityManager.LogDeviceRegistration(registrationRequest.DeviceID, registrationRequest.InvitationToken, true, "", map[string]interface{}{
		"client_ip": r.RemoteAddr,
	})

	response := map[string]interface{}{
		"status": "Device registered successfully",
		"device_id": registrationRequest.DeviceID,
		"timestamp": "now",
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, fmt.Sprintf("Failed to encode response: %v", err), http.StatusInternalServerError)
	}
}

// Security middleware for sensitive endpoints
func (s *Server) securityMiddleware(handler http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Log access attempt
		s.securityManager.LogSecurityEvent("api_access_attempt", true, "", map[string]interface{}{
			"endpoint": r.URL.Path,
			"method": r.Method,
			"client_ip": r.RemoteAddr,
			"user_agent": r.UserAgent(),
		})

		// Add security headers
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("X-XSS-Protection", "1; mode=block")

		// Call the handler
		handler(w, r)
	}
}