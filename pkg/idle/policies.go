// Package idle provides advanced idle detection and policy management
package idle

import (
	"fmt"
	"time"
)

// PolicyTemplate represents a pre-configured hibernation policy
type PolicyTemplate struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Category    PolicyCategory    `json:"category"`
	Schedules   []Schedule        `json:"schedules"`
	Tags        map[string]string `json:"tags"`

	// Estimated savings
	EstimatedSavingsPercent float64  `json:"estimated_savings_percent"`
	SuitableFor             []string `json:"suitable_for"`

	// Configuration
	AutoApply bool     `json:"auto_apply"`
	Priority  int      `json:"priority"`
	Conflicts []string `json:"conflicts"` // IDs of conflicting templates
}

// PolicyCategory categorizes policy templates
type PolicyCategory string

const (
	CategoryAggressive   PolicyCategory = "aggressive"
	CategoryBalanced     PolicyCategory = "balanced"
	CategoryConservative PolicyCategory = "conservative"
	CategoryDevelopment  PolicyCategory = "development"
	CategoryProduction   PolicyCategory = "production"
	CategoryResearch     PolicyCategory = "research"
	CategoryCustom       PolicyCategory = "custom"
)

// PolicyManager manages hibernation policies
type PolicyManager struct {
	templates map[string]*PolicyTemplate
	applied   map[string][]string // instance -> policy IDs
	scheduler *Scheduler          // Optional scheduler for automated execution
}

// NewPolicyManager creates a new policy manager with default templates
func NewPolicyManager() *PolicyManager {
	pm := &PolicyManager{
		templates: make(map[string]*PolicyTemplate),
		applied:   make(map[string][]string),
	}

	// Load default policy templates
	pm.loadDefaultTemplates()

	return pm
}

// SetScheduler sets the scheduler for automated policy execution
func (pm *PolicyManager) SetScheduler(scheduler *Scheduler) {
	pm.scheduler = scheduler
}

// loadDefaultTemplates loads pre-configured policy templates
func (pm *PolicyManager) loadDefaultTemplates() {
	// Aggressive Cost Optimization
	pm.templates["aggressive-cost"] = &PolicyTemplate{
		ID:          "aggressive-cost",
		Name:        "Aggressive Cost Optimization",
		Description: "Maximizes cost savings with frequent hibernation. Best for development and testing environments.",
		Category:    CategoryAggressive,
		Schedules: []Schedule{
			{
				Name:            "Business Hours Only",
				Type:            ScheduleTypeWorkHours,
				HibernateAction: "hibernate",
				WakeAction:      "resume",
				IdleMinutes:     10,
				CPUThreshold:    5.0,
				MemoryThreshold: 10.0,
			},
			{
				Name:            "Weekend Shutdown",
				Type:            ScheduleTypeWeekly,
				DaysOfWeek:      []DayOfWeek{Saturday, Sunday},
				StartTime:       "00:00",
				EndTime:         "23:59",
				HibernateAction: "stop",
			},
		},
		EstimatedSavingsPercent: 65,
		SuitableFor:             []string{"development", "testing", "staging"},
		AutoApply:               false,
		Priority:                1,
	}

	// Balanced Performance
	pm.templates["balanced"] = &PolicyTemplate{
		ID:          "balanced",
		Name:        "Balanced Performance",
		Description: "Balances cost savings with availability. Suitable for most workloads.",
		Category:    CategoryBalanced,
		Schedules: []Schedule{
			{
				Name:            "Night Hibernation",
				Type:            ScheduleTypeDaily,
				StartTime:       "20:00",
				EndTime:         "08:00",
				HibernateAction: "hibernate",
				WakeAction:      "resume",
			},
			{
				Name:            "Idle Detection",
				Type:            ScheduleTypeIdle,
				IdleMinutes:     30,
				CPUThreshold:    10.0,
				MemoryThreshold: 20.0,
				HibernateAction: "hibernate",
			},
		},
		EstimatedSavingsPercent: 40,
		SuitableFor:             []string{"general", "web", "api"},
		AutoApply:               true,
		Priority:                2,
	}

	// Conservative Availability
	pm.templates["conservative"] = &PolicyTemplate{
		ID:          "conservative",
		Name:        "Conservative Availability",
		Description: "Minimal hibernation for high-availability workloads.",
		Category:    CategoryConservative,
		Schedules: []Schedule{
			{
				Name:               "Extended Idle Only",
				Type:               ScheduleTypeIdle,
				IdleMinutes:        60,
				CPUThreshold:       5.0,
				MemoryThreshold:    10.0,
				HibernateAction:    "hibernate",
				GracePeriodMinutes: 15,
			},
		},
		EstimatedSavingsPercent: 15,
		SuitableFor:             []string{"production", "critical"},
		AutoApply:               false,
		Priority:                3,
	}

	// Research Workloads
	pm.templates["research"] = &PolicyTemplate{
		ID:          "research",
		Name:        "Research Optimization",
		Description: "Optimized for research workloads with batch processing patterns.",
		Category:    CategoryResearch,
		Schedules: []Schedule{
			{
				Name:            "Batch Window",
				Type:            ScheduleTypeDaily,
				StartTime:       "02:00",
				EndTime:         "06:00",
				HibernateAction: "hibernate",
				WakeAction:      "none", // Manual wake for batch jobs
			},
			{
				Name:             "GPU Idle Detection",
				Type:             ScheduleTypeIdle,
				IdleMinutes:      15,
				CPUThreshold:     20.0,
				MemoryThreshold:  30.0,
				NetworkThreshold: 10.0,
				HibernateAction:  "stop", // Stop GPU instances to save more
			},
		},
		EstimatedSavingsPercent: 45,
		SuitableFor:             []string{"ml", "datascience", "hpc"},
		AutoApply:               false,
		Priority:                2,
		Tags: map[string]string{
			"workload": "batch",
			"gpu":      "optimized",
		},
	}

	// Development Environment
	pm.templates["development"] = &PolicyTemplate{
		ID:          "development",
		Name:        "Development Environment",
		Description: "Aggressive hibernation for development instances.",
		Category:    CategoryDevelopment,
		Schedules: []Schedule{
			{
				Name:            "After Hours",
				Type:            ScheduleTypeDaily,
				StartTime:       "18:00",
				EndTime:         "09:00",
				HibernateAction: "stop",
				WakeAction:      "start",
			},
			{
				Name:            "Quick Idle",
				Type:            ScheduleTypeIdle,
				IdleMinutes:     5,
				CPUThreshold:    5.0,
				HibernateAction: "hibernate",
			},
			{
				Name:            "Weekend Off",
				Type:            ScheduleTypeWeekly,
				DaysOfWeek:      []DayOfWeek{Saturday, Sunday},
				HibernateAction: "terminate", // Terminate dev instances on weekends
			},
		},
		EstimatedSavingsPercent: 75,
		SuitableFor:             []string{"dev", "sandbox", "experiment"},
		AutoApply:               true,
		Priority:                1,
	}

	// Production Safeguard
	pm.templates["production"] = &PolicyTemplate{
		ID:          "production",
		Name:        "Production Safeguard",
		Description: "Minimal intervention for production workloads with safety checks.",
		Category:    CategoryProduction,
		Schedules: []Schedule{
			{
				Name:               "Emergency Idle",
				Type:               ScheduleTypeIdle,
				IdleMinutes:        120,
				CPUThreshold:       2.0,
				MemoryThreshold:    5.0,
				HibernateAction:    "alert", // Just alert, don't hibernate
				GracePeriodMinutes: 30,
				RequireTags:        []string{"env:production"},
			},
		},
		EstimatedSavingsPercent: 5,
		SuitableFor:             []string{"production", "critical", "database"},
		AutoApply:               false,
		Priority:                5,
		Conflicts:               []string{"aggressive-cost", "development"},
	}
}

// GetTemplate retrieves a policy template
func (pm *PolicyManager) GetTemplate(id string) (*PolicyTemplate, error) {
	template, exists := pm.templates[id]
	if !exists {
		return nil, fmt.Errorf("policy template not found: %s", id)
	}
	return template, nil
}

// ListTemplates returns all available policy templates
func (pm *PolicyManager) ListTemplates() []*PolicyTemplate {
	templates := make([]*PolicyTemplate, 0, len(pm.templates))
	for _, template := range pm.templates {
		templates = append(templates, template)
	}
	return templates
}

// ListTemplatesByCategory returns templates filtered by category
func (pm *PolicyManager) ListTemplatesByCategory(category PolicyCategory) []*PolicyTemplate {
	var templates []*PolicyTemplate
	for _, template := range pm.templates {
		if template.Category == category {
			templates = append(templates, template)
		}
	}
	return templates
}

// ApplyTemplate applies a policy template to an instance
func (pm *PolicyManager) ApplyTemplate(instanceID string, templateID string) error {
	template, err := pm.GetTemplate(templateID)
	if err != nil {
		return err
	}

	// Check for conflicts
	if err := pm.checkConflicts(instanceID, template); err != nil {
		return err
	}

	// Apply the template
	if pm.applied[instanceID] == nil {
		pm.applied[instanceID] = []string{}
	}
	pm.applied[instanceID] = append(pm.applied[instanceID], templateID)

	// Apply schedules to the instance if scheduler is available
	if pm.scheduler != nil {
		for _, schedule := range template.Schedules {
			// Add schedule to scheduler if not already present
			if _, err := pm.scheduler.GetSchedule(schedule.ID); err != nil {
				// Schedule doesn't exist, add it
				scheduleCopy := schedule // Copy to avoid modifying template
				scheduleCopy.ID = generateScheduleID()
				if err := pm.scheduler.AddSchedule(&scheduleCopy); err != nil {
					return fmt.Errorf("failed to add schedule: %w", err)
				}
			}

			// Assign schedule to instance
			if err := pm.scheduler.AssignScheduleToInstance(schedule.ID, instanceID); err != nil {
				return fmt.Errorf("failed to assign schedule to instance: %w", err)
			}
		}
	}

	return nil
}

// RemoveTemplate removes a policy template from an instance
func (pm *PolicyManager) RemoveTemplate(instanceID string, templateID string) error {
	policies, exists := pm.applied[instanceID]
	if !exists {
		return fmt.Errorf("no policies applied to instance: %s", instanceID)
	}

	// Remove the template ID from the list
	newPolicies := []string{}
	found := false
	for _, id := range policies {
		if id != templateID {
			newPolicies = append(newPolicies, id)
		} else {
			found = true
		}
	}

	if !found {
		return fmt.Errorf("template %s not applied to instance %s", templateID, instanceID)
	}

	pm.applied[instanceID] = newPolicies

	// Remove schedules from the instance if scheduler is available
	if pm.scheduler != nil {
		template, exists := pm.templates[templateID]
		if exists {
			for _, schedule := range template.Schedules {
				// Remove schedule assignment from instance
				if err := pm.scheduler.RemoveScheduleFromInstance(schedule.ID, instanceID); err != nil {
					// Log error but don't fail the removal
					fmt.Printf("Warning: failed to remove schedule %s from instance: %v\n", schedule.ID, err)
				}
			}
		}
	}

	return nil
}

// GetAppliedTemplates returns templates applied to an instance
func (pm *PolicyManager) GetAppliedTemplates(instanceID string) ([]*PolicyTemplate, error) {
	policyIDs, exists := pm.applied[instanceID]
	if !exists {
		return []*PolicyTemplate{}, nil
	}

	templates := make([]*PolicyTemplate, 0, len(policyIDs))
	for _, id := range policyIDs {
		if template, err := pm.GetTemplate(id); err == nil {
			templates = append(templates, template)
		}
	}

	return templates, nil
}

// RecommendTemplate recommends a policy template based on instance characteristics
func (pm *PolicyManager) RecommendTemplate(instanceType string, tags map[string]string) (*PolicyTemplate, error) {
	// Check environment tag
	env := tags["env"]
	if env == "" {
		env = tags["environment"]
	}

	switch env {
	case "production", "prod":
		return pm.GetTemplate("production")
	case "development", "dev":
		return pm.GetTemplate("development")
	case "research", "ml", "datascience":
		return pm.GetTemplate("research")
	default:
		// Default to balanced
		return pm.GetTemplate("balanced")
	}
}

// checkConflicts checks if a template conflicts with existing policies
func (pm *PolicyManager) checkConflicts(instanceID string, template *PolicyTemplate) error {
	existingIDs, exists := pm.applied[instanceID]
	if !exists {
		return nil
	}

	for _, existingID := range existingIDs {
		// Check if new template conflicts with existing
		for _, conflictID := range template.Conflicts {
			if conflictID == existingID {
				return fmt.Errorf("template %s conflicts with existing template %s", template.ID, existingID)
			}
		}

		// Check if existing conflicts with new template
		if existing, err := pm.GetTemplate(existingID); err == nil {
			for _, conflictID := range existing.Conflicts {
				if conflictID == template.ID {
					return fmt.Errorf("existing template %s conflicts with template %s", existingID, template.ID)
				}
			}
		}
	}

	return nil
}

// CreateCustomTemplate creates a custom policy template
func (pm *PolicyManager) CreateCustomTemplate(name, description string, schedules []Schedule) (*PolicyTemplate, error) {
	id := fmt.Sprintf("custom-%d", time.Now().Unix())

	template := &PolicyTemplate{
		ID:          id,
		Name:        name,
		Description: description,
		Category:    CategoryCustom,
		Schedules:   schedules,
		AutoApply:   false,
		Priority:    10, // Low priority for custom templates
	}

	// Calculate estimated savings
	template.EstimatedSavingsPercent = pm.estimateSavings(schedules)

	pm.templates[id] = template

	return template, nil
}

// estimateSavings estimates the savings percentage for a set of schedules
func (pm *PolicyManager) estimateSavings(schedules []Schedule) float64 {
	totalHours := 24 * 7 // Hours in a week
	hibernationHours := 0.0

	for _, schedule := range schedules {
		switch schedule.Type {
		case ScheduleTypeDaily:
			// Calculate daily hibernation hours
			hours := calculateHoursBetween(schedule.StartTime, schedule.EndTime)
			hibernationHours += hours * 7 // 7 days a week
		case ScheduleTypeWeekly:
			hours := calculateHoursBetween(schedule.StartTime, schedule.EndTime)
			hibernationHours += hours * float64(len(schedule.DaysOfWeek))
		case ScheduleTypeWorkHours:
			// Nights and weekends
			hibernationHours += 15 * 5 // 15 hours/day * 5 weekdays
			hibernationHours += 24 * 2 // 24 hours/day * 2 weekend days
		case ScheduleTypeIdle:
			// Estimate based on idle threshold
			hibernationHours += float64(schedule.IdleMinutes) / 60 * 24 // Rough estimate
		}
	}

	if hibernationHours > float64(totalHours) {
		hibernationHours = float64(totalHours)
	}

	return (hibernationHours / float64(totalHours)) * 100
}
