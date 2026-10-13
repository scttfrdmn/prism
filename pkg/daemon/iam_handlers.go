package daemon

import (
	"net/http"

	"github.com/scttfrdmn/prism/pkg/aws"
)

// handleIAMValidate validates the current AWS credentials and checks required IAM permissions.
// GET /api/v1/iam/validate
func (s *Server) handleIAMValidate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	var result *aws.IAMValidationResult

	s.withAWSManager(w, r, func(awsManager *aws.Manager) error {
		var err error
		result, err = awsManager.ValidateIAMPermissions(r.Context())
		return err
	})

	if result == nil {
		return // withAWSManager already wrote error
	}

	w.Header().Set("Content-Type", "application/json")
	encodeJSONResponse(w, result)
}

// handleIAMPermissions lists the required AWS permissions for launching instances.
// GET /api/v1/iam/permissions
func (s *Server) handleIAMPermissions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	resp := map[string]interface{}{
		"required_permissions": aws.GetRequiredLaunchPermissions(),
		"description":          "Minimum IAM permissions required to launch Prism instances",
	}

	w.Header().Set("Content-Type", "application/json")
	encodeJSONResponse(w, resp)
}
