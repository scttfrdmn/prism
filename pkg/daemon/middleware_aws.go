// Package daemon provides AWS credential middleware
package daemon

import (
	"log"
	"net/http"
)

// ReducedModeError represents an error when AWS credentials are required but unavailable
type ReducedModeError struct {
	Error       string `json:"error"`
	Message     string `json:"message"`
	ReducedMode bool   `json:"reduced_mode"`
	Help        string `json:"help"`
}

// requireAWSMiddleware checks if AWS credentials are available before allowing operation
// If in reduced mode, returns HTTP 503 Service Unavailable with helpful error message
func (s *Server) requireAWSMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Skip check if in test mode
		if s.testMode {
			next(w, r)
			return
		}

		// Check if running in reduced functionality mode
		if s.reducedMode {
			errResponse := ReducedModeError{
				Error:       "AWS credentials required",
				Message:     "This operation requires AWS credentials",
				ReducedMode: true,
				Help:        getCredentialErrorMessage(),
			}

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusServiceUnavailable)
			encodeJSONResponse(w, errResponse)
			return
		}

		// AWS credentials available, proceed with operation
		next(w, r)
	}
}

// IsReducedMode returns whether the server is running in reduced functionality mode
func (s *Server) IsReducedMode() bool {
	return s.reducedMode
}

// SetReducedMode sets the reduced functionality mode status
// This is primarily used during initialization when AWS credentials are unavailable
func (s *Server) SetReducedMode(enabled bool) {
	s.reducedMode = enabled
	if enabled {
		log.Println(getReducedModeBanner())
	}
}
