// Security API handlers for CloudWorkstation daemon
package daemon

import (
	"encoding/json"
	"fmt"
	"net/http"

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
			"client_ip":  r.RemoteAddr,
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
			"system_health":    status.SystemHealth,
			"keychain_info":    status.KeychainInfo,
			"last_check":       status.LastHealthCheck,
			"security_enabled": status.Enabled,
		}

		if err := json.NewEncoder(w).Encode(healthResponse); err != nil {
			http.Error(w, fmt.Sprintf("Failed to encode response: %v", err), http.StatusInternalServerError)
		}

	case "POST":
		// Trigger health check
		if err := s.securityManager.PerformHealthCheck(); err != nil {
			s.securityManager.LogSecurityEvent("health_check_triggered", false, "", map[string]interface{}{
				"error":     err.Error(),
				"client_ip": r.RemoteAddr,
			})
			http.Error(w, fmt.Sprintf("Health check failed: %v", err), http.StatusInternalServerError)
			return
		}

		s.securityManager.LogSecurityEvent("health_check_triggered", true, "", map[string]interface{}{
			"client_ip": r.RemoteAddr,
		})

		response := map[string]interface{}{
			"status":    "Health check completed successfully",
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
			"client_ip":  r.RemoteAddr,
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
			"correlations":        status.Correlations,
			"correlation_count":   len(status.Correlations),
			"correlation_enabled": status.Configuration.CorrelationEnabled,
		}

		s.securityManager.LogSecurityEvent("security_correlations_accessed", true, "", map[string]interface{}{
			"client_ip":         r.RemoteAddr,
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
			"info":        keychainInfo,
			"diagnostics": diagnostics,
		}

		s.securityManager.LogSecurityEvent("keychain_info_accessed", true, "", map[string]interface{}{
			"client_ip": r.RemoteAddr,
			"provider":  keychainInfo.Provider,
			"native":    keychainInfo.Native,
		})

		if err := json.NewEncoder(w).Encode(keychainResponse); err != nil {
			http.Error(w, fmt.Sprintf("Failed to encode response: %v", err), http.StatusInternalServerError)
		}

	case "POST":
		// Validate keychain provider
		if err := security.ValidateKeychainProvider(); err != nil {
			s.securityManager.LogSecurityEvent("keychain_validation_failed", false, "", map[string]interface{}{
				"error":     err.Error(),
				"client_ip": r.RemoteAddr,
			})
			http.Error(w, fmt.Sprintf("Keychain validation failed: %v", err), http.StatusInternalServerError)
			return
		}

		s.securityManager.LogSecurityEvent("keychain_validation_success", true, "", map[string]interface{}{
			"client_ip": r.RemoteAddr,
		})

		response := map[string]interface{}{
			"status":    "Keychain validation successful",
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
			"enabled":       status.Enabled,
			"running":       status.Running,
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
			"reason":    "not_implemented",
		})

		http.Error(w, "Security configuration updates not yet implemented", http.StatusNotImplemented)

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}
