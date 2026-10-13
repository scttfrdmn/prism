//go:build integration
// +build integration

package chaos

import (
	"github.com/scttfrdmn/prism/pkg/templates"
)

// Thin adapters from the template-edge-case tests onto the current
// pkg/templates API.

// newTemplateRegistry returns an empty, in-memory template registry (no
// template directories to scan).
func newTemplateRegistry() *templates.TemplateRegistry {
	return templates.NewTemplateRegistry(nil)
}

// registerTemplate adds a parsed template to the registry, indexed by name and
// slug, mirroring what TemplateRegistry.ScanTemplates does for files on disk.
func registerTemplate(registry *templates.TemplateRegistry, tmpl *templates.Template) {
	if tmpl == nil {
		return
	}
	registry.Templates[tmpl.Name] = tmpl
	if tmpl.Slug != "" {
		registry.SlugIndex[tmpl.Slug] = tmpl.Name
	}
}

// validateTemplate runs the comprehensive validator over a template and
// returns its individual findings. A nil template (failed parse) has none.
func validateTemplate(validator *templates.ComprehensiveValidator, tmpl *templates.Template) []templates.ValidationResult {
	if tmpl == nil {
		return nil
	}
	return validator.ValidateTemplate(tmpl).Results
}

// resolveTemplate resolves inheritance across the registry and returns the
// merged template. Templates whose inheritance cannot be resolved (e.g. a
// missing parent) are dropped by ResolveInheritance, so they surface here as
// a "template not found" error.
func resolveTemplate(registry *templates.TemplateRegistry, name string) (*templates.Template, error) {
	if err := registry.ResolveInheritance(); err != nil {
		return nil, err
	}
	return registry.GetTemplate(name)
}
