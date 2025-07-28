package models

import (
	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/scttfrdmn/cloudworkstation/internal/tui/api"
	"github.com/scttfrdmn/cloudworkstation/internal/tui/components"
)

// ProfilesKeyMap defines keybindings for the profiles page
type ProfilesKeyMap struct {
	Help key.Binding
	Quit key.Binding
	Back key.Binding
}

// ShortHelp returns keybindings to be shown in the mini help view
func (k ProfilesKeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Back, k.Help, k.Quit}
}

// FullHelp returns keybindings for the expanded help view
func (k ProfilesKeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Back, k.Help, k.Quit},
	}
}

// DefaultProfilesKeyMap returns a set of default keybindings
func DefaultProfilesKeyMap() ProfilesKeyMap {
	return ProfilesKeyMap{
		Help: key.NewBinding(
			key.WithKeys("?"),
			key.WithHelp("?", "help"),
		),
		Quit: key.NewBinding(
			key.WithKeys("q", "ctrl+c"),
			key.WithHelp("q", "quit"),
		),
		Back: key.NewBinding(
			key.WithKeys("esc"),
			key.WithHelp("esc", "back"),
		),
	}
}

// ProfilesModel represents the profiles page
type ProfilesModel struct {
	keys       ProfilesKeyMap
	help       help.Model
	client     api.Client
	width      int
	height     int
	ready      bool
	showHelp   bool
	profileMgr *components.ProfileManager
}

// NewProfilesModel creates a new profiles model
func NewProfilesModel(client api.Client) ProfilesModel {
	return ProfilesModel{
		keys:   DefaultProfilesKeyMap(),
		help:   help.New(),
		client: client,
	}
}

// Init initializes the profiles model
func (m ProfilesModel) Init() tea.Cmd {
	return nil
}

// SetSize sets the model's dimensions
func (m *ProfilesModel) SetSize(width, height int) {
	m.width = width
	m.height = height
	
	if m.profileMgr != nil {
		m.profileMgr.SetSize(width, height-4) // Reserve space for help bar
	}
	
	m.help.Width = width
	m.ready = true
}

// Update handles UI events
func (m ProfilesModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd
	
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, m.keys.Help):
			m.showHelp = !m.showHelp
		case key.Matches(msg, m.keys.Quit):
			return m, tea.Quit
		case key.Matches(msg, m.keys.Back):
			return m, func() tea.Msg { return BackMsg{} }
		}
	case ProfileInitMsg:
		// Create profile manager when we have the profile manager from the client
		if m.client.ProfileManager() != nil {
			m.profileMgr = components.NewProfileManager(m.client.ProfileManager())
			if m.ready {
				m.profileMgr.SetSize(m.width, m.height-4)
			}
			cmd = m.profileMgr.Init()
			cmds = append(cmds, cmd)
		}
	}
	
	// Pass messages to profile manager if available
	if m.profileMgr != nil {
		var profileCmd tea.Cmd
		_, profileCmd = m.profileMgr.Update(msg)
		if profileCmd != nil {
			cmds = append(cmds, profileCmd)
		}
	}
	
	return m, tea.Batch(cmds...)
}

// View renders the profiles model
func (m ProfilesModel) View() string {
	if !m.ready {
		return "Loading..."
	}
	
	content := ""
	if m.profileMgr != nil {
		content = m.profileMgr.View()
	} else {
		content = "Profile manager not initialized."
	}
	
	helpView := ""
	if m.showHelp {
		helpView = m.help.View(m.keys)
	} else {
		helpView = m.help.ShortHelp(m.keys)
	}
	
	return lipgloss.JoinVertical(
		lipgloss.Left,
		"# Profiles",
		"",
		content,
		"",
		helpView,
	)
}

// ProfileInitMsg is sent to initialize the profiles model
type ProfileInitMsg struct{}