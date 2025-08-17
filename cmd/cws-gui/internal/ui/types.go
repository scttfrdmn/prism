package ui

import (
	"fmt"
	"time"

	"fyne.io/fyne/v2"

	"github.com/scttfrdmn/cloudworkstation/pkg/api"
	"github.com/scttfrdmn/cloudworkstation/pkg/profile"
	"github.com/scttfrdmn/cloudworkstation/pkg/types"
)

// NavigationSection represents different GUI sections
type NavigationSection int

const (
	SectionDashboard NavigationSection = iota
	SectionInstances
	SectionTemplates
	SectionVolumes
	SectionSettings
)

// CloudWorkstationGUI represents the main GUI application
type CloudWorkstationGUI struct {
	app       fyne.App
	window    fyne.Window
	apiClient api.CloudWorkstationAPI
	
	// Navigation
	currentSection NavigationSection
	sidebar        *fyne.Container
	content        *fyne.Container
	notification   *fyne.Container
	
	// Data
	instances  []types.Instance
	totalCost  float64
	lastUpdate time.Time
	
	// Profile Management
	profileManager *profile.ManagerEnhanced
	stateManager   *profile.ProfileAwareStateManager
	activeProfile  *profile.Profile
}

// Section represents a GUI section with its own view and behavior
type Section interface {
	// CreateView creates the section's main view
	CreateView() fyne.CanvasObject
	
	// UpdateView refreshes the section's data and view
	UpdateView() error
	
	// GetTitle returns the section's display title
	GetTitle() string
}

// SectionManager manages GUI sections using Dependency Inversion Principle
type SectionManager struct {
	sections map[NavigationSection]Section
	gui      *CloudWorkstationGUI
}

// NewSectionManager creates a new section manager
func NewSectionManager(gui *CloudWorkstationGUI) *SectionManager {
	return &SectionManager{
		sections: make(map[NavigationSection]Section),
		gui:      gui,
	}
}

// RegisterSection registers a section (Open/Closed Principle)
func (sm *SectionManager) RegisterSection(nav NavigationSection, section Section) {
	sm.sections[nav] = section
}

// GetSection retrieves a section
func (sm *SectionManager) GetSection(nav NavigationSection) Section {
	return sm.sections[nav]
}

// NavigateToSection changes the active section
func (sm *SectionManager) NavigateToSection(nav NavigationSection) error {
	section := sm.sections[nav]
	if section == nil {
		return fmt.Errorf("section not found: %d", nav)
	}
	
	// Update the content
	sm.gui.content.Objects = []fyne.CanvasObject{section.CreateView()}
	sm.gui.content.Refresh()
	sm.gui.currentSection = nav
	
	return section.UpdateView()
}