package daemon

import (
	"context"
	"crypto/subtle"
	"log"
	"net/http"
	"os"
	"strings"
)

const (
	// Additional context keys for middleware
	authenticatedKey contextKey = iota + 2 // Start after awsRegionKey
)

// AWSHeadersMiddleware extracts AWS-related headers from the request
// and adds them to the request context
func (s *Server) awsHeadersMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Extract AWS profile header
		awsProfile := r.Header.Get("X-AWS-Profile")
		if awsProfile != "" {
			log.Printf("Using AWS profile: %s", awsProfile)
			// Store in context for later use
			ctx := setAWSProfile(r.Context(), awsProfile)
			r = r.WithContext(ctx)
		}

		// Extract AWS region header
		awsRegion := r.Header.Get("X-AWS-Region")
		if awsRegion != "" {
			log.Printf("Using AWS region: %s", awsRegion)
			// Store in context for later use
			ctx := setAWSRegion(r.Context(), awsRegion)
			r = r.WithContext(ctx)
		}

		// Call the next handler
		next(w, r)
	}
}

// authMiddleware validates API key in the request header
func (s *Server) authMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Skip authentication for the ping endpoint (health check)
		if r.URL.Path == "/api/v1/ping" {
			next(w, r)
			return
		}

		// Skip authentication for the auth endpoint
		if r.URL.Path == "/api/v1/auth" && r.Method == http.MethodPost {
			next(w, r)
			return
		}

		// Skip authentication for the authentication endpoint
		if r.URL.Path == "/api/v1/authenticate" {
			next(w, r)
			return
		}

		// Load state to get the API key
		state, err := s.stateManager.LoadState()
		if err != nil {
			s.writeError(w, http.StatusInternalServerError, "Failed to load server configuration")
			return
		}

		// Check if running in test mode (bypasses authentication)
		if os.Getenv("PRISM_TEST_MODE") == "true" {
			next(w, r)
			return
		}

		// Check if API key is enabled (exists in config)
		if state.Config.APIKey == "" {
			// No API key set, allow access without authentication
			// This maintains backward compatibility for existing setups
			next(w, r)
			return
		}

		// Get API key from header
		providedKey := r.Header.Get("X-API-Key")
		if providedKey == "" {
			s.writeError(w, http.StatusUnauthorized, "API key required")
			return
		}

		// Constant-time comparison to prevent timing attacks
		if subtle.ConstantTimeCompare([]byte(providedKey), []byte(state.Config.APIKey)) != 1 {
			s.writeError(w, http.StatusUnauthorized, "Invalid API key")
			return
		}

		// Mark the request as authenticated in the context
		ctx := context.WithValue(r.Context(), authenticatedKey, true)

		// Check for user authentication token
		token := r.Header.Get("Authorization")
		if token != "" && strings.HasPrefix(token, "Bearer ") {
			// Extract token from header
			token = strings.TrimPrefix(token, "Bearer ")

			// Check if user manager is initialized
			if s.userManager != nil && s.userManager.initialized {
				// Future Enhancement: Token validation for multi-user authentication
				// When implementing institutional deployments with OAuth/LDAP/SAML:
				//   1. Call s.userManager.ValidateToken(token) to verify token
				//   2. Extract user ID and permissions from validated token
				//   3. Apply role-based access control (RBAC) based on user permissions
				// Current behavior: Uses token directly as user ID for single-user/development mode
				userID := token

				// Add user ID to context
				ctx = setUserID(ctx, userID)
			}
		}

		next(w, r.WithContext(ctx))
	}
}

// combineMiddleware combines multiple middleware functions into a single middleware
func (s *Server) combineMiddleware(handler http.HandlerFunc, middlewares ...func(http.HandlerFunc) http.HandlerFunc) http.HandlerFunc {
	// Apply middleware in reverse order (so the first middleware is executed first)
	for i := len(middlewares) - 1; i >= 0; i-- {
		handler = middlewares[i](handler)
	}
	return handler
}

// Context helper functions (setAWSProfile, getAWSProfile, setAWSRegion, getAWSRegion are defined in context.go)

func isAuthenticated(ctx context.Context) bool {
	value := ctx.Value(authenticatedKey)
	if value == nil {
		return false
	}
	return value.(bool)
}
