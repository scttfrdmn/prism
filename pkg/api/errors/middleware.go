package errors

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/scttfrdmn/prism/pkg/types"
)

// ErrorHandler is a function that can handle errors
type ErrorHandler func(err error, w http.ResponseWriter, r *http.Request)

// ErrorHandlingMiddleware adds error handling to an HTTP handler
func ErrorHandlingMiddleware(next http.Handler, errorHandler ErrorHandler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var err error

		// Recover from panics
		defer func() {
			if panicValue := recover(); panicValue != nil {
				log.Printf("Panic recovered in request handler: %v", panicValue)
				err = types.NewAPIError(types.ErrServerError, "Internal server error", nil)
				errorHandler(err, w, r)
			}
		}()

		// Create a response recorder to capture the response
		start := time.Now()

		// Call the next handler
		next.ServeHTTP(w, r)

		// Log request duration
		duration := time.Since(start)
		log.Printf("%s %s completed in %v", r.Method, r.URL.Path, duration)
	})
}

// DefaultErrorHandler is a default implementation of ErrorHandler
func DefaultErrorHandler(err error, w http.ResponseWriter, r *http.Request) {
	var apiErr types.APIError
	var ok bool

	// Check if the error is an APIError
	if apiErr, ok = err.(types.APIError); !ok {
		// Convert to APIError
		apiErr = types.NewAPIError(types.ErrServerError, err.Error(), err)
	}

	// Set status code from error or default to 500
	statusCode := apiErr.StatusCode
	if statusCode == 0 {
		// Map error code to HTTP status
		statusCode = ErrorCodeToStatusCode(apiErr.Code)
	}

	// Set content type
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	// Add request ID if available
	if apiErr.RequestID == "" {
		apiErr.RequestID = r.Header.Get("X-Request-ID")
	}

	// Write error as JSON
	if err := json.NewEncoder(w).Encode(apiErr); err != nil {
		log.Printf("Failed to encode error response: %v", err)
		// Write a simple error message if JSON encoding fails
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

// ErrorCodeToStatusCode maps error codes to HTTP status codes
func ErrorCodeToStatusCode(code types.ErrorCode) int {
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
