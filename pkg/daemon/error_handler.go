package daemon

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/scttfrdmn/cloudworkstation/pkg/types"
)

// writeError writes a JSON error response to the response writer
func (s *Server) writeError(w http.ResponseWriter, statusCode int, message string) {
	// Create standardized error response
	errorCode := types.GetErrorCodeFromStatusCode(statusCode)
	apiError := types.APIError{
		Code:       errorCode,
		Message:    message,
		StatusCode: statusCode,
	}

	// Set headers and status
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	// Write error response
	if err := json.NewEncoder(w).Encode(apiError); err != nil {
		log.Printf("Error encoding error response: %v", err)
	}
}

// writeJSON writes a JSON response to the response writer
func (s *Server) writeJSON(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	if err := json.NewEncoder(w).Encode(data); err != nil {
		log.Printf("Error encoding JSON response: %v", err)
	}
}
