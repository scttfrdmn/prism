package models

import (
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/scttfrdmn/cloudworkstation/internal/tui/components"
	"github.com/scttfrdmn/cloudworkstation/internal/tui/styles"
	"github.com/scttfrdmn/cloudworkstation/pkg/api"
)

// SettingsModel represents the settings view
type SettingsModel struct {
	apiClient      api.CloudWorkstationAPI
	statusBar      components.StatusBar
	width          int
	height         int
	loading        bool
	error          string
	focusedInput   int
	awsProfile     textinput.Model
	awsRegion      textinput.Model
	apiEndpoint    textinput.Model
	defaultSize    textinput.Model
	darkMode       bool
	saveInProgress bool
	saveSuccess    bool
}

// SettingsRefreshMsg refreshes the settings data
type SettingsRefreshMsg struct{}

// SaveSettingsMsg is sent when settings are saved
type SaveSettingsMsg struct {
	Success bool
	Error   error
}

// NewSettingsModel creates a new settings model
func NewSettingsModel(apiClient api.CloudWorkstationAPI) SettingsModel {
	// Create text inputs
	awsProfile := textinput.New()
	awsProfile.Placeholder = "default"
	awsProfile.Prompt = "› "
	awsProfile.CharLimit = 50
	awsProfile.Width = 30
	
	awsRegion := textinput.New()
	awsRegion.Placeholder = "us-west-2"
	awsRegion.Prompt = "› "
	awsRegion.CharLimit = 20
	awsRegion.Width = 30
	
	apiEndpoint := textinput.New()
	apiEndpoint.Placeholder = "http://localhost:8080"
	apiEndpoint.Prompt = "› "
	apiEndpoint.CharLimit = 100
	apiEndpoint.Width = 50
	
	defaultSize := textinput.New()
	defaultSize.Placeholder = "M"
	defaultSize.Prompt = "› "
	defaultSize.CharLimit = 2
	defaultSize.Width = 10
	
	// Set first input as focused
	awsProfile.Focus()
	
	// Create status bar
	statusBar := components.NewStatusBar("", "")
	
	// Set default theme
	darkMode := styles.CurrentThemeMode == styles.DarkTheme
	
	return SettingsModel{
		apiClient:    apiClient,
		awsProfile:   awsProfile,
		awsRegion:    awsRegion,
		apiEndpoint:  apiEndpoint,
		defaultSize:  defaultSize,
		statusBar:    statusBar,
		focusedInput: 0,
		darkMode:     darkMode,
	}
}

// Init initializes the settings model
func (m SettingsModel) Init() tea.Cmd {
	return tea.Batch(
		m.fetchSettings,
	)
}

// fetchSettings retrieves current settings
func (m *SettingsModel) fetchSettings() tea.Msg {
	// Try to get AWS profile from environment or config
	if profile := os.Getenv("AWS_PROFILE"); profile != "" {
		m.awsProfile.SetValue(profile)
	}
	
	// Try to get AWS region from environment or config
	if region := os.Getenv("AWS_REGION"); region != "" {
		m.awsRegion.SetValue(region)
	} else if region := os.Getenv("AWS_DEFAULT_REGION"); region != "" {
		m.awsRegion.SetValue(region)
	}
	
	// Try to get API endpoint from environment
	if endpoint := os.Getenv("CWSD_URL"); endpoint != "" {
		m.apiEndpoint.SetValue(endpoint)
	}
	
	// Get theme setting
	m.darkMode = styles.CurrentThemeMode == styles.DarkTheme
	
	// In a real implementation, we would get other settings from config file
	// For now, we just return a RefreshMsg
	return SettingsRefreshMsg{}
}

// saveSettings saves the current settings
func (m SettingsModel) saveSettings() tea.Cmd {
	return func() tea.Msg {
		// In a real implementation, we would save settings to config file
		// For now, just simulate saving and apply the theme setting
		
		// Apply theme setting
		if m.darkMode {
			styles.SetThemeMode(styles.DarkTheme)
		} else {
			styles.SetThemeMode(styles.LightTheme)
		}
		
		// Simulate writing to config file
		success := true
		var saveError error
		
		// Return result
		return SaveSettingsMsg{
			Success: success,
			Error:   saveError,
		}
	}
}

// focusNextInput focuses the next input field
func (m *SettingsModel) focusNextInput() {
	// Blur current input
	switch m.focusedInput {
	case 0:
		m.awsProfile.Blur()
	case 1:
		m.awsRegion.Blur()
	case 2:
		m.apiEndpoint.Blur()
	case 3:
		m.defaultSize.Blur()
	}
	
	// Focus next input - now we have 5 inputs (including theme toggle)
	m.focusedInput = (m.focusedInput + 1) % 5
	
	// Focus the newly selected input
	switch m.focusedInput {
	case 0:
		m.awsProfile.Focus()
	case 1:
		m.awsRegion.Focus()
	case 2:
		m.apiEndpoint.Focus()
	case 3:
		m.defaultSize.Focus()
	case 4:
		// Theme toggle is not a text input, nothing to focus
	}
}

// focusPrevInput focuses the previous input field
func (m *SettingsModel) focusPrevInput() {
	// Blur current input
	switch m.focusedInput {
	case 0:
		m.awsProfile.Blur()
	case 1:
		m.awsRegion.Blur()
	case 2:
		m.apiEndpoint.Blur()
	case 3:
		m.defaultSize.Blur()
	}
	
	// Focus prev input - now we have 5 inputs (including theme toggle)
	m.focusedInput = (m.focusedInput - 1 + 5) % 5
	
	// Focus the newly selected input
	switch m.focusedInput {
	case 0:
		m.awsProfile.Focus()
	case 1:
		m.awsRegion.Focus()
	case 2:
		m.apiEndpoint.Focus()
	case 3:
		m.defaultSize.Focus()
	case 4:
		// Theme toggle is not a text input, nothing to focus
	}
}

// Update handles messages and updates the model
func (m SettingsModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.statusBar.SetWidth(msg.Width)
		
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+s":
			if !m.saveInProgress {
				m.saveInProgress = true
				m.saveSuccess = false
				m.statusBar.SetStatus("Saving settings...", components.StatusWarning)
				return m, m.saveSettings()
			}
			
		case "tab":
			m.focusNextInput()
			return m, nil
			
		case "shift+tab":
			m.focusPrevInput()
			return m, nil
			
		case "enter":
			// Handle enter key press based on focused element
			switch m.focusedInput {
			case 3: // If on default size input
				// Move to theme toggle
				m.focusNextInput()
				return m, nil
			case 4: // If on theme toggle
				// Toggle theme
				m.darkMode = !m.darkMode
				// Apply theme
				if m.darkMode {
					styles.SetThemeMode(styles.DarkTheme)
				} else {
					styles.SetThemeMode(styles.LightTheme)
				}
				m.statusBar.SetStatus(fmt.Sprintf("%s mode activated", styles.GetCurrentThemeMode()), components.StatusSuccess)
				return m, nil
			default:
				// Move to next input
				m.focusNextInput()
				return m, nil
			}
			
		case "t":
			// Shortcut for toggling theme
			if m.focusedInput == 4 {
				m.darkMode = !m.darkMode
				// Apply theme
				if m.darkMode {
					styles.SetThemeMode(styles.DarkTheme)
				} else {
					styles.SetThemeMode(styles.LightTheme)
				}
				m.statusBar.SetStatus(fmt.Sprintf("%s mode activated", styles.GetCurrentThemeMode()), components.StatusSuccess)
				return m, nil
			}
			
		case "q", "esc":
			return m, tea.Quit
		}
		
	case SettingsRefreshMsg:
		m.loading = false
		m.statusBar.SetStatus("Settings loaded", components.StatusSuccess)
		
	case SaveSettingsMsg:
		m.saveInProgress = false
		if msg.Success {
			m.saveSuccess = true
			m.statusBar.SetStatus("Settings saved successfully", components.StatusSuccess)
		} else {
			m.error = msg.Error.Error()
			m.statusBar.SetStatus(fmt.Sprintf("Error saving settings: %s", m.error), components.StatusError)
		}
		return m, nil
		
	case error:
		m.loading = false
		m.error = msg.Error()
		m.statusBar.SetStatus(fmt.Sprintf("Error: %s", m.error), components.StatusError)
	}
	
	// Handle input updates
	var cmd tea.Cmd
	m.awsProfile, cmd = m.awsProfile.Update(msg)
	cmds = append(cmds, cmd)
	
	m.awsRegion, cmd = m.awsRegion.Update(msg)
	cmds = append(cmds, cmd)
	
	m.apiEndpoint, cmd = m.apiEndpoint.Update(msg)
	cmds = append(cmds, cmd)
	
	m.defaultSize, cmd = m.defaultSize.Update(msg)
	cmds = append(cmds, cmd)
	
	return m, tea.Batch(cmds...)
}

// View renders the settings view
func (m SettingsModel) View() string {
	theme := styles.CurrentTheme
	
	// Title section
	title := theme.Title.Render("CloudWorkstation Settings")
	
	// Settings form
	formStyle := lipgloss.NewStyle().
		Border(lipgloss.NormalBorder()).
		BorderForeground(theme.BorderColor).
		Padding(1, 2).
		Width(m.width - 4) // Account for borders
	
	// Create form content
	var formContent strings.Builder
	
	// AWS Profile
	formContent.WriteString(theme.Label.Render("AWS Profile:") + "\n")
	formContent.WriteString(m.awsProfile.View() + "\n\n")
	
	// AWS Region
	formContent.WriteString(theme.Label.Render("AWS Region:") + "\n")
	formContent.WriteString(m.awsRegion.View() + "\n\n")
	
	// API Endpoint
	formContent.WriteString(theme.Label.Render("API Endpoint:") + "\n")
	formContent.WriteString(m.apiEndpoint.View() + "\n\n")
	
	// Default Instance Size
	formContent.WriteString(theme.Label.Render("Default Instance Size:") + "\n")
	formContent.WriteString(m.defaultSize.View() + "\n\n")
	
	// Theme Toggle
	formContent.WriteString(theme.Label.Render("Theme Mode:") + "\n")
	
	// Create theme toggle buttons
	darkButtonStyle := theme.Button.Copy()
	lightButtonStyle := theme.Button.Copy()
	
	// Highlight active theme
	if m.darkMode {
		darkButtonStyle = darkButtonStyle.Background(theme.AccentColor)
	} else {
		lightButtonStyle = lightButtonStyle.Background(theme.AccentColor)
	}
	
	// Render buttons
	darkButton := darkButtonStyle.Render(" Dark ")
	lightButton := lightButtonStyle.Render(" Light ")
	
	// Add focus indicator if this input is selected
	if m.focusedInput == 4 {
		formContent.WriteString(fmt.Sprintf("%s %s %s", darkButton, lightButton, theme.MutedColor.Render("← Press Enter to toggle or 't'")) + "\n\n")
	} else {
		formContent.WriteString(fmt.Sprintf("%s %s", darkButton, lightButton) + "\n\n")
	}
	
	// Save button
	saveButton := theme.Button.Render(" Save Settings ")
	if m.saveInProgress {
		saveButton = theme.Button.Copy().Background(theme.MutedColor).Render(" Saving... ")
	} else if m.saveSuccess {
		saveButton = theme.Button.Copy().Background(theme.SuccessColor).Render(" Saved ✓ ")
	}
	
	formContent.WriteString(saveButton + " (Ctrl+S)")
	
	// Render form
	form := formStyle.Render(formContent.String())
	
	// Help text
	help := components.CompactHelpView(components.HelpSettings)
	
	// Join everything together
	return lipgloss.JoinVertical(
		lipgloss.Left,
		title,
		"",
		form,
		"",
		m.statusBar.View(),
		help,
	)
}