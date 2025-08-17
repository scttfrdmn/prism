// Package ami provides CloudWorkstation's AMI creation system.
package ami

import (
	"time"
)

// SharedTemplateEntry represents a template entry in the shared registry
type SharedTemplateEntry struct {
	Name         string
	Version      string
	Description  string
	TemplateData string
	PublishedAt  time.Time
	PublishedBy  string
	Format       string // "yaml", "json", etc.
	Tags         map[string]string
}

// ImportTemplateOptions contains options for template importing
type TemplateImportOptions struct {
	Validate      bool
	Force         bool
	OverwriteName string
}

// DefaultClock implements the Clock interface using system time
type DefaultClock struct{}

// Now returns the current system time
func (c *DefaultClock) Now() time.Time {
	return time.Now()
}
