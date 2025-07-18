package daemon

import (
	"context"
	"fmt"
	"net/http"

	"github.com/scttfrdmn/cloudworkstation/pkg/aws"
)

// createAWSManagerFromRequest creates a new AWS manager from the request context
func (s *Server) createAWSManagerFromRequest(r *http.Request) (*aws.Manager, error) {
	// Get AWS credentials from request context
	options := aws.ManagerOptions{}
	if profile := getAWSProfile(r.Context()); profile != "" {
		options.Profile = profile
	}
	if region := getAWSRegion(r.Context()); region != "" {
		options.Region = region
	}

	// Create AWS manager with credentials
	return aws.NewManager(options)
}

// withAWSManager runs the given function with an AWS manager created from the request
// and handles error cases, writing the error to the response
func (s *Server) withAWSManager(w http.ResponseWriter, r *http.Request, 
	fn func(*aws.Manager) error) {
	
	// Create AWS manager with credentials from the request
	awsManager, err := s.createAWSManagerFromRequest(r)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, 
			fmt.Sprintf("Failed to initialize AWS manager: %v", err))
		return
	}
	
	// Call the provided function with the AWS manager
	if err := fn(awsManager); err != nil {
		s.writeError(w, http.StatusInternalServerError, 
			fmt.Sprintf("AWS operation failed: %v", err))
		return
	}
}