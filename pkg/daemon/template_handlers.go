package daemon

import (
	"encoding/json"
	"net/http"

	"github.com/scttfrdmn/cloudworkstation/pkg/aws"
	"github.com/scttfrdmn/cloudworkstation/pkg/types"
)

// handleTemplates handles template collection operations
func (s *Server) handleTemplates(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	var templates map[string]types.Template
	s.withAWSManager(w, r, func(awsManager *aws.Manager) error {
		templates = awsManager.GetTemplates()
		return nil
	})

	if templates != nil {
		json.NewEncoder(w).Encode(templates)
	}
}

// handleTemplateInfo handles operations on specific templates
func (s *Server) handleTemplateInfo(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	templateName := r.URL.Path[len("/api/v1/templates/"):]
	
	var template *types.Template
	s.withAWSManager(w, r, func(awsManager *aws.Manager) error {
		var err error
		template, err = awsManager.GetTemplate(templateName)
		return err
	})

	if template != nil {
		json.NewEncoder(w).Encode(template)
	}
}