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
		Code:      errorCode,
		Message:   message,
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

// writeAPIError writes a types.APIError to the response writer
func (s *Server) writeAPIError(w http.ResponseWriter, apiError types.APIError) {
	// Set status code
	statusCode := apiError.StatusCode
	if statusCode == 0 {
		// Map error code to status code
		statusCode = getStatusCodeFromErrorCode(apiError.Code)
	}

	// Set headers and status
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	// Write error response
	if err := json.NewEncoder(w).Encode(apiError); err != nil {
		log.Printf("Error encoding error response: %v", err)
	}
}

// getStatusCodeFromErrorCode maps error codes to HTTP status codes
func getStatusCodeFromErrorCode(code types.ErrorCode) int {
	switch code {
	case types.ErrNotFound:
		return http.StatusNotFound
	case types.ErrUnauthorized:
		return http.StatusUnauthorized
	case types.ErrForbidden, types.ErrPermissionDenied:
		return http.StatusForbidden
	case types.ErrInvalidParameters, types.ErrValidationFailed, types.ErrInvalidFormat, types.ErrMissingRequired:
		return http.StatusBadRequest
	case types.ErrResourceExists, types.ErrConflict:
		return http.StatusConflict
	case types.ErrRateLimited:
		return http.StatusTooManyRequests
	case types.ErrTimeout:
		return http.StatusGatewayTimeout
	default:
		return http.StatusInternalServerError
	}
}

// errorResponseMiddleware adds error handling to routes
func (s *Server) errorResponseMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Recover from panics
		defer func() {
			if r := recover(); r != nil {
				log.Printf("Panic in request handler: %v", r)
				s.writeError(w, http.StatusInternalServerError, "Internal server error")
			}
		}()

		// Call the next handler
		next(w, r)
	}
}

// convertAWSError converts AWS errors to API errors
func (s *Server) convertAWSError(err error, operation string) types.APIError {
	// TODO: Implement AWS error mapping for better client error handling
	// This would detect specific AWS error types and convert them to appropriate API errors
	
	// For now, just return a generic AWS error
	return types.NewAPIError(types.ErrAWSError, 
		"AWS operation failed", 
		err,
	).WithOperation(operation)
}