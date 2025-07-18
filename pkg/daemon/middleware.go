package daemon

import (
	"log"
	"net/http"
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

// combineMiddleware combines multiple middleware functions into a single middleware
func (s *Server) combineMiddleware(handler http.HandlerFunc, middlewares ...func(http.HandlerFunc) http.HandlerFunc) http.HandlerFunc {
	// Apply middleware in reverse order (so the first middleware is executed first)
	for i := len(middlewares) - 1; i >= 0; i-- {
		handler = middlewares[i](handler)
	}
	return handler
}