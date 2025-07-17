package components

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/scttfrdmn/cloudworkstation/pkg/profile"
)

var (
	profileListStyle = lipgloss.NewStyle().
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("240")).
		Padding(1, 0).
		Width(60)

	profileSelectedStyle = lipgloss.NewStyle().
		Border(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("36")).
		Padding(1, 2)

	profileDetailStyle = lipgloss.NewStyle().
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("240")).
		Padding(1).
		Width(80)

	profileTitleStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("170")).
		Bold(true)

	profileLabelStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("241")).
		Width(16)

	profileValueStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("252"))

	profileTypePersonalStyle = lipgloss.NewStyle().
		Background(lipgloss.Color("25")).
		Foreground(lipgloss.Color("15")).
		Padding(0, 1)

	profileTypeInvitationStyle = lipgloss.NewStyle().
		Background(lipgloss.Color("127")).
		Foreground(lipgloss.Color("15")).
		Padding(0, 1)

	profileSecureStyle = lipgloss.NewStyle().
		Background(lipgloss.Color("34")).
		Foreground(lipgloss.Color("15")).
		Padding(0, 1)

	profileUnsecureStyle = lipgloss.NewStyle().
		Background(lipgloss.Color("160")).
		Foreground(lipgloss.Color("15")).
		Padding(0, 1)
)

// profileItem represents a profile in the list
type profileItem struct {
	profile profile.Profile
	active  bool
}

// ProfileManager is a component for managing profiles
type ProfileManager struct {
	profileManager      *profile.ManagerEnhanced
	secureManager       *profile.SecureInvitationManager
	profiles            []profile.Profile
	currentProfile      profile.Profile
	list                list.Model
	selectedProfile     *profile.Profile
	state               string
	addPersonalInput    textinput.Model
	addInvitationInput  textinput.Model
	addInvitationName   textinput.Model
	validationMsg       string
	width              int
	height             int
}

// FilterValue implements list.Item interface
func (i profileItem) FilterValue() string {
	return i.profile.Name
}

// Title implements list.Item interface
func (i profileItem) Title() string {
	if i.active {
		return "✓ " + i.profile.Name
	}
	return "  " + i.profile.Name
}

// Description implements list.Item interface
func (i profileItem) Description() string {
	var typeStr string
	if i.profile.Type == profile.ProfileTypePersonal {
		typeStr = profileTypePersonalStyle.Render("Personal")
	} else {
		typeStr = profileTypeInvitationStyle.Render("Invitation")
	}
	
	var secureStr string
	if i.profile.DeviceBound {
		secureStr = profileSecureStyle.Render("Secure")
	} else if i.profile.Type == profile.ProfileTypeInvitation {
		secureStr = profileUnsecureStyle.Render("Unsecure")
	}
	
	var parts []string
	parts = append(parts, typeStr)
	if i.profile.Region != "" {
		parts = append(parts, i.profile.Region)
	}
	if secureStr != "" {
		parts = append(parts, secureStr)
	}
	
	return strings.Join(parts, " | ")
}

// NewProfileManager creates a new profile manager component
func NewProfileManager(pm *profile.ManagerEnhanced) *ProfileManager {
	l := list.New([]list.Item{}, list.NewDefaultDelegate(), 0, 0)
	l.Title = "Profiles"
	l.SetShowHelp(false)
	
	nameInput := textinput.New()
	nameInput.Placeholder = "AWS Profile Name"
	nameInput.CharLimit = 50
	nameInput.Width = 30
	
	invInput := textinput.New()
	invInput.Placeholder = "Invitation Token (starts with inv-)"
	invInput.CharLimit = 200
	invInput.Width = 50
	
	invNameInput := textinput.New()
	invNameInput.Placeholder = "Profile Name"
	invNameInput.CharLimit = 50
	invNameInput.Width = 30
	
	// Create secure invitation manager
	var secureManager *profile.SecureInvitationManager
	if sm, err := profile.NewSecureInvitationManager(pm); err == nil {
		secureManager = sm
	}
	
	return &ProfileManager{
		profileManager:     pm,
		secureManager:      secureManager,
		list:               l,
		state:              "list",
		addPersonalInput:   nameInput,
		addInvitationInput: invInput,
		addInvitationName:  invNameInput,
	}
}

// SetSize sets the component size
func (p *ProfileManager) SetSize(width, height int) {
	p.width = width
	p.height = height
	p.list.SetSize(width-4, height-6)
}

// loadProfiles loads profiles from the profile manager
func (p *ProfileManager) loadProfiles() error {
	// Load profiles
	profiles, err := p.profileManager.ListProfiles()
	if err != nil {
		return err
	}
	p.profiles = profiles
	
	// Get current profile
	currentProfile, err := p.profileManager.GetCurrentProfile()
	if err == nil {
		p.currentProfile = currentProfile
	}
	
	// Convert to list items
	var items []list.Item
	for _, prof := range profiles {
		active := prof.AWSProfile == p.currentProfile.AWSProfile
		items = append(items, profileItem{profile: prof, active: active})
	}
	
	p.list.SetItems(items)
	return nil
}

// Init initializes the component
func (p *ProfileManager) Init() tea.Cmd {
	_ = p.loadProfiles()
	return nil
}

// Update handles component updates
func (p *ProfileManager) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "enter":
			if p.state == "list" && p.list.SelectedItem() != nil {
				item := p.list.SelectedItem().(profileItem)
				p.selectedProfile = &item.profile
				p.state = "detail"
			} else if p.state == "addPersonal" {
				// Add personal profile
				if p.addPersonalInput.Value() != "" {
					profile := profile.Profile{
						Type:       profile.ProfileTypePersonal,
						Name:       p.addPersonalInput.Value(),
						AWSProfile: p.addPersonalInput.Value(),
						Region:     "", // Default region
					}
					err := p.profileManager.AddProfile(profile)
					if err != nil {
						p.validationMsg = fmt.Sprintf("Error: %v", err)
					} else {
						p.validationMsg = fmt.Sprintf("Added profile: %s", profile.Name)
						p.addPersonalInput.Reset()
						p.state = "list"
						_ = p.loadProfiles()
					}
				}
			} else if p.state == "addInvitation" {
				// Add invitation profile
				token := p.addInvitationInput.Value()
				name := p.addInvitationName.Value()
				
				if token == "" || name == "" {
					p.validationMsg = "Error: Both token and name are required"
				} else if !strings.HasPrefix(token, "inv-") {
					p.validationMsg = "Error: Token must start with 'inv-'"
				} else {
					var err error
					if p.secureManager != nil {
						// Use secure invitation manager if available
						err = p.secureManager.SecureAddToProfile(token, name)
					} else {
						// Fall back to regular invitation manager
						invManager, ierr := profile.NewInvitationManager(p.profileManager)
						if ierr != nil {
							p.validationMsg = fmt.Sprintf("Error: %v", ierr)
							break
						}
						err = invManager.AddToProfile(token, name)
					}
					
					if err != nil {
						p.validationMsg = fmt.Sprintf("Error: %v", err)
					} else {
						p.validationMsg = fmt.Sprintf("Added profile from invitation: %s", name)
						p.addInvitationInput.Reset()
						p.addInvitationName.Reset()
						p.state = "list"
						_ = p.loadProfiles()
					}
				}
			}
			
		case "esc":
			if p.state == "detail" || p.state == "addPersonal" || p.state == "addInvitation" {
				p.state = "list"
				p.selectedProfile = nil
				p.validationMsg = ""
			}
			
		case "a":
			if p.state == "list" {
				p.state = "addPersonal"
				p.validationMsg = ""
				p.addPersonalInput.Focus()
			}
			
		case "i":
			if p.state == "list" {
				p.state = "addInvitation"
				p.validationMsg = ""
				p.addInvitationInput.Focus()
			}
			
		case "tab":
			if p.state == "addInvitation" {
				if p.addInvitationInput.Focused() {
					p.addInvitationInput.Blur()
					p.addInvitationName.Focus()
				} else {
					p.addInvitationName.Blur()
					p.addInvitationInput.Focus()
				}
			}
			
		case "s":
			if p.state == "detail" && p.selectedProfile != nil {
				// Switch to selected profile
				err := p.profileManager.SwitchProfile(p.selectedProfile.AWSProfile)
				if err != nil {
					p.validationMsg = fmt.Sprintf("Error switching profile: %v", err)
				} else {
					p.validationMsg = fmt.Sprintf("Switched to profile: %s", p.selectedProfile.Name)
					p.currentProfile = *p.selectedProfile
					p.state = "list"
					_ = p.loadProfiles()
				}
			}
			
		case "v":
			if p.state == "detail" && p.selectedProfile != nil {
				// Validate profile
				p.validationMsg = "Validating profile..."
				
				if p.selectedProfile.Type == profile.ProfileTypeInvitation && p.selectedProfile.DeviceBound {
					// Validate secure profile
					if p.secureManager != nil {
						err := p.secureManager.ValidateSecureProfile(p.selectedProfile)
						if err != nil {
							p.validationMsg = fmt.Sprintf("Validation failed: %v", err)
						} else {
							p.validationMsg = "Profile validated successfully"
						}
					} else {
						p.validationMsg = "Secure validation not available"
					}
				} else {
					// Basic validation
					err := p.profileManager.ValidateProfile(p.selectedProfile.AWSProfile)
					if err != nil {
						p.validationMsg = fmt.Sprintf("Validation failed: %v", err)
					} else {
						p.validationMsg = "Profile validated successfully"
					}
				}
			}
			
		case "d", "backspace", "delete":
			if p.state == "detail" && p.selectedProfile != nil {
				// Don't allow deleting the current profile
				if p.selectedProfile.AWSProfile == p.currentProfile.AWSProfile {
					p.validationMsg = "Cannot delete the active profile"
				} else {
					// Delete profile
					err := p.profileManager.RemoveProfile(p.selectedProfile.AWSProfile)
					if err != nil {
						p.validationMsg = fmt.Sprintf("Error removing profile: %v", err)
					} else {
						p.validationMsg = fmt.Sprintf("Removed profile: %s", p.selectedProfile.Name)
						p.selectedProfile = nil
						p.state = "list"
						_ = p.loadProfiles()
					}
				}
			}
		}
	}
	
	// Handle list updates
	if p.state == "list" {
		var listCmd tea.Cmd
		p.list, listCmd = p.list.Update(msg)
		cmd = listCmd
	} else if p.state == "addPersonal" {
		var inputCmd tea.Cmd
		p.addPersonalInput, inputCmd = p.addPersonalInput.Update(msg)
		cmd = inputCmd
	} else if p.state == "addInvitation" {
		var inputCmd tea.Cmd
		if p.addInvitationInput.Focused() {
			p.addInvitationInput, inputCmd = p.addInvitationInput.Update(msg)
		} else {
			p.addInvitationName, inputCmd = p.addInvitationName.Update(msg)
		}
		cmd = inputCmd
	}
	
	return p, cmd
}

// View renders the component
func (p *ProfileManager) View() string {
	switch p.state {
	case "list":
		helpText := "\n[↑/↓] Navigate • [enter] Select • [a] Add Personal • [i] Add Invitation"
		listView := p.list.View()
		return profileListStyle.Render(listView + helpText)
		
	case "detail":
		if p.selectedProfile != nil {
			profile := p.selectedProfile
			
			// Build detail view
			var sb strings.Builder
			sb.WriteString(profileTitleStyle.Render(profile.Name) + "\n\n")
			
			// Basic properties
			sb.WriteString(fmt.Sprintf("%s %s\n", 
				profileLabelStyle.Render("Type:"), 
				profileValueStyle.Render(string(profile.Type))))
				
			sb.WriteString(fmt.Sprintf("%s %s\n", 
				profileLabelStyle.Render("AWS Profile:"), 
				profileValueStyle.Render(profile.AWSProfile)))
				
			if profile.Region != "" {
				sb.WriteString(fmt.Sprintf("%s %s\n", 
					profileLabelStyle.Render("Region:"), 
					profileValueStyle.Render(profile.Region)))
			}
			
			// Invitation-specific properties
			if profile.Type == profile.ProfileTypeInvitation {
				sb.WriteString("\n" + profileTitleStyle.Render("Invitation Details") + "\n\n")
				
				sb.WriteString(fmt.Sprintf("%s %s\n", 
					profileLabelStyle.Render("Owner Account:"), 
					profileValueStyle.Render(profile.OwnerAccount)))
					
				sb.WriteString(fmt.Sprintf("%s %s\n", 
					profileLabelStyle.Render("Token:"), 
					profileValueStyle.Render(profile.InvitationToken)))
					
				// Security properties
				sb.WriteString("\n" + profileTitleStyle.Render("Security") + "\n\n")
				
				var deviceStatus string
				if profile.DeviceBound {
					deviceStatus = profileSecureStyle.Render("Yes")
				} else {
					deviceStatus = profileUnsecureStyle.Render("No")
				}
				sb.WriteString(fmt.Sprintf("%s %s\n", 
					profileLabelStyle.Render("Device Bound:"), 
					deviceStatus))
					
				var transferStatus string
				if profile.Transferable {
					transferStatus = profileUnsecureStyle.Render("Yes")
				} else {
					transferStatus = profileSecureStyle.Render("No")
				}
				sb.WriteString(fmt.Sprintf("%s %s\n", 
					profileLabelStyle.Render("Transferable:"), 
					transferStatus))
					
				if profile.BindingRef != "" {
					sb.WriteString(fmt.Sprintf("%s %s\n", 
						profileLabelStyle.Render("Binding:"), 
						profileValueStyle.Render("Active")))
				}
			}
			
			// Created time
			sb.WriteString("\n" + fmt.Sprintf("%s %s\n", 
				profileLabelStyle.Render("Created:"), 
				profileValueStyle.Render(profile.CreatedAt.Format("2006-01-02 15:04:05"))))
				
			// Last used
			if !profile.LastUsed.IsZero() {
				sb.WriteString(fmt.Sprintf("%s %s\n", 
					profileLabelStyle.Render("Last Used:"), 
					profileValueStyle.Render(profile.LastUsed.Format("2006-01-02 15:04:05"))))
			}
			
			// Add help text
			helpText := "\n[esc] Back • [s] Switch to profile • [v] Validate • [d] Delete"
			
			// Add validation message if any
			if p.validationMsg != "" {
				helpText = "\n" + p.validationMsg + helpText
			}
			
			return profileDetailStyle.Render(sb.String() + helpText)
		}
		return "No profile selected"
		
	case "addPersonal":
		var sb strings.Builder
		sb.WriteString(profileTitleStyle.Render("Add Personal Profile") + "\n\n")
		sb.WriteString("Enter the name of your AWS profile:\n\n")
		sb.WriteString(p.addPersonalInput.View() + "\n\n")
		sb.WriteString("This should match an AWS profile in your ~/.aws/credentials file.\n")
		
		helpText := "\n[enter] Add Profile • [esc] Cancel"
		
		// Add validation message if any
		if p.validationMsg != "" {
			helpText = "\n" + p.validationMsg + helpText
		}
		
		return profileDetailStyle.Render(sb.String() + helpText)
		
	case "addInvitation":
		var sb strings.Builder
		sb.WriteString(profileTitleStyle.Render("Add Invitation Profile") + "\n\n")
		sb.WriteString("Enter the invitation token:\n")
		sb.WriteString(p.addInvitationInput.View() + "\n\n")
		sb.WriteString("Enter a name for this profile:\n")
		sb.WriteString(p.addInvitationName.View() + "\n\n")
		
		helpText := "\n[tab] Switch fields • [enter] Add Profile • [esc] Cancel"
		
		// Add validation message if any
		if p.validationMsg != "" {
			helpText = "\n" + p.validationMsg + helpText
		}
		
		return profileDetailStyle.Render(sb.String() + helpText)
	}
	
	return ""
}