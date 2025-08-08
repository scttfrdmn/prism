package daemon

import (
	"encoding/json"
	"net/http"

	"github.com/scttfrdmn/cloudworkstation/pkg/templates"
)

// handleTemplates handles template collection operations
func (s *Server) handleTemplates(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	// Get region and architecture from query params or headers
	region := r.URL.Query().Get("region")
	if region == "" {
		region = "us-east-1" // Default region
	}
	
	architecture := r.URL.Query().Get("architecture")
	if architecture == "" {
		architecture = "x86_64" // Default architecture
	}

	// Use the new unified template system
	templates, err := templates.GetTemplatesForDaemonHandler(region, architecture)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "Failed to load templates: "+err.Error())
		return
	}

	json.NewEncoder(w).Encode(templates)
}

// handleTemplateInfo handles operations on specific templates
func (s *Server) handleTemplateInfo(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	templateName := r.URL.Path[len("/api/v1/templates/"):]
	
	// Get region and architecture from query params or headers
	region := r.URL.Query().Get("region")
	if region == "" {
		region = "us-east-1" // Default region
	}
	
	architecture := r.URL.Query().Get("architecture")
	if architecture == "" {
		architecture = "x86_64" // Default architecture
	}
	
	// Get package manager override from query params
	packageManager := r.URL.Query().Get("package_manager")
	
	// Get size for instance type scaling from query params
	size := r.URL.Query().Get("size")

	// Use the new unified template system with package manager and size support
	template, err := templates.GetTemplateWithPackageManager(templateName, region, architecture, packageManager, size)
	if err != nil {
		s.writeError(w, http.StatusNotFound, "Template not found: "+err.Error())
		return
	}

	json.NewEncoder(w).Encode(template)
}