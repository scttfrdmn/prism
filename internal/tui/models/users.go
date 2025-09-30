package models

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/scttfrdmn/cloudworkstation/internal/tui/components"
	"github.com/scttfrdmn/cloudworkstation/internal/tui/styles"
	"github.com/scttfrdmn/cloudworkstation/pkg/profile"
	"github.com/scttfrdmn/cloudworkstation/pkg/research"
)

// UsersModel represents the user management view
type UsersModel struct {
	apiClient        apiClient
	statusBar        components.StatusBar
	spinner          components.Spinner
	width            int
	height           int
	loading          bool
	error            string
	users            []*research.ResearchUserConfig
	selectedUser     int
	researchUserMgr  *research.ResearchUserManager
	showCreateDialog bool
	createUsername   string
	showDeleteDialog bool
	deleteUsername   string
}

// NewUsersModel creates a new users model
func NewUsersModel(apiClient apiClient) UsersModel {
	// Create status bar and spinner
	statusBar := components.NewStatusBar("CloudWorkstation Users", "")
	spinner := components.NewSpinner("Loading users...")

	// Initialize research user manager
	homeDir, _ := os.UserHomeDir()
	configDir := filepath.Join(homeDir, ".cloudworkstation")

	// Create a simplified profile adapter for TUI
	profileAdapter := &TUIProfileManagerAdapter{}
	researchUserMgr := research.NewResearchUserManager(profileAdapter, configDir)

	return UsersModel{
		apiClient:       apiClient,
		statusBar:       statusBar,
		spinner:         spinner,
		width:           80,
		height:          24,
		loading:         true,
		users:           []*research.ResearchUserConfig{},
		selectedUser:    0,
		researchUserMgr: researchUserMgr,
	}
}

// Init initializes the model
func (m UsersModel) Init() tea.Cmd {
	return tea.Batch(
		m.spinner.InitialCmd(),
		m.fetchUsers,
	)
}

// fetchUsers retrieves users from the manager
func (m UsersModel) fetchUsers() tea.Msg {
	users, err := m.researchUserMgr.ListResearchUsers()
	if err != nil {
		return fmt.Errorf("failed to list users: %w", err)
	}

	return UsersDataMsg{
		Users: users,
	}
}

// UsersDataMsg represents users data retrieved from the manager
type UsersDataMsg struct {
	Users []*research.ResearchUserConfig
}

// CreateUserMsg represents a user creation action
type CreateUserMsg struct {
	Username string
	Success  bool
	Message  string
}

// DeleteUserMsg represents a user deletion action
type DeleteUserMsg struct {
	Username string
	Success  bool
	Message  string
}

// Update handles messages and updates the model
func (m UsersModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.statusBar.SetWidth(msg.Width)

	case tea.KeyMsg:
		if m.showCreateDialog {
			return m.handleCreateDialog(msg)
		}
		if m.showDeleteDialog {
			return m.handleDeleteDialog(msg)
		}

		switch msg.String() {
		case "r":
			m.loading = true
			m.error = ""
			return m, m.fetchUsers

		case "c":
			// Show create user dialog
			m.showCreateDialog = true
			m.createUsername = ""

		case "d":
			// Show delete user dialog for selected user
			if len(m.users) > 0 && m.selectedUser < len(m.users) {
				m.showDeleteDialog = true
				m.deleteUsername = m.users[m.selectedUser].Username
			}

		case "up", "k":
			if m.selectedUser > 0 {
				m.selectedUser--
			}

		case "down", "j":
			if m.selectedUser < len(m.users)-1 {
				m.selectedUser++
			}

		case "s":
			// Show status for selected user
			if len(m.users) > 0 && m.selectedUser < len(m.users) {
				username := m.users[m.selectedUser].Username
				m.statusBar.SetStatus(fmt.Sprintf("User: %s (UID: %d)", username, m.users[m.selectedUser].UID), components.StatusInfo)
			}

		case "q", "esc":
			return m, tea.Quit
		}

	case RefreshMsg:
		m.loading = true
		m.error = ""
		return m, m.fetchUsers

	case error:
		m.loading = false
		m.error = msg.Error()
		m.statusBar.SetStatus(fmt.Sprintf("Error: %s", m.error), components.StatusError)

	case UsersDataMsg:
		m.loading = false
		m.users = msg.Users
		m.statusBar.SetStatus(fmt.Sprintf("Loaded %d users", len(m.users)), components.StatusSuccess)

	case CreateUserMsg:
		m.showCreateDialog = false
		if msg.Success {
			m.statusBar.SetStatus(fmt.Sprintf("Created user: %s", msg.Username), components.StatusSuccess)
			// Refresh the list
			return m, m.fetchUsers
		} else {
			m.statusBar.SetStatus(fmt.Sprintf("Failed to create user: %s", msg.Message), components.StatusError)
		}

	case DeleteUserMsg:
		m.showDeleteDialog = false
		if msg.Success {
			m.statusBar.SetStatus(fmt.Sprintf("Deleted user: %s", msg.Username), components.StatusSuccess)
			// Refresh the list
			return m, m.fetchUsers
		} else {
			m.statusBar.SetStatus(fmt.Sprintf("Failed to delete user: %s", msg.Message), components.StatusError)
		}
	}

	// Update components
	if m.loading {
		var spinnerCmd tea.Cmd
		m.spinner, spinnerCmd = m.spinner.Update(msg)
		cmds = append(cmds, spinnerCmd)
	}

	return m, tea.Batch(cmds...)
}

// handleCreateDialog handles input in the create user dialog
func (m UsersModel) handleCreateDialog(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		if m.createUsername != "" {
			return m, m.createUser(m.createUsername)
		}
		m.showCreateDialog = false

	case "esc":
		m.showCreateDialog = false

	case "backspace":
		if len(m.createUsername) > 0 {
			m.createUsername = m.createUsername[:len(m.createUsername)-1]
		}

	default:
		// Add character to username
		if len(msg.String()) == 1 && isValidUsernameChar(msg.String()[0]) {
			m.createUsername += msg.String()
		}
	}

	return m, nil
}

// handleDeleteDialog handles confirmation in the delete user dialog
func (m UsersModel) handleDeleteDialog(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "y", "Y":
		return m, m.deleteUser(m.deleteUsername)

	case "n", "N", "esc":
		m.showDeleteDialog = false
	}

	return m, nil
}

// createUser creates a new user
func (m UsersModel) createUser(username string) tea.Cmd {
	return func() tea.Msg {
		_, err := m.researchUserMgr.GetOrCreateResearchUser(username)
		if err != nil {
			return CreateUserMsg{
				Username: username,
				Success:  false,
				Message:  err.Error(),
			}
		}

		return CreateUserMsg{
			Username: username,
			Success:  true,
			Message:  "User created successfully",
		}
	}
}

// deleteUser deletes a user
func (m UsersModel) deleteUser(username string) tea.Cmd {
	return func() tea.Msg {
		// Get current profile
		currentProfile := "default" // Simplified for TUI

		err := m.researchUserMgr.DeleteResearchUser(currentProfile, username)
		if err != nil {
			return DeleteUserMsg{
				Username: username,
				Success:  false,
				Message:  err.Error(),
			}
		}

		return DeleteUserMsg{
			Username: username,
			Success:  true,
			Message:  "User deleted successfully",
		}
	}
}

// View renders the users view
func (m UsersModel) View() string {
	theme := styles.CurrentTheme

	// Title
	title := theme.Title.Render("👥 Users")

	var content string

	if m.showCreateDialog {
		content = m.renderCreateDialog()
	} else if m.showDeleteDialog {
		content = m.renderDeleteDialog()
	} else if m.loading {
		content = lipgloss.NewStyle().
			Width(m.width).
			Height(m.height-4).
			Align(lipgloss.Center, lipgloss.Center).
			Render(m.spinner.View())
	} else if m.error != "" {
		content = lipgloss.NewStyle().
			Width(m.width).
			Height(m.height-4).
			Align(lipgloss.Center, lipgloss.Center).
			Render(theme.StatusError.Render("Error: " + m.error))
	} else {
		content = m.renderUsersList()
	}

	// Help text
	var help string
	if m.showCreateDialog {
		help = theme.Help.Render("enter: create • esc: cancel")
	} else if m.showDeleteDialog {
		help = theme.Help.Render("y: confirm • n/esc: cancel")
	} else {
		help = theme.Help.Render("Navigation: 1-6 change page • Actions: ↑/↓: navigate • c: create • d: delete • s: status • r: refresh • q: quit")
	}

	// Join everything together
	return lipgloss.JoinVertical(
		lipgloss.Left,
		title,
		content,
		"",
		m.statusBar.View(),
		help,
	)
}

// renderUsersList renders the list of users
func (m UsersModel) renderUsersList() string {
	theme := styles.CurrentTheme

	if len(m.users) == 0 {
		return lipgloss.NewStyle().
			Padding(2).
			Render("No users found\n\nPress 'c' to create a new user.")
	}

	var lines []string
	for i, user := range m.users {
		createdDate := user.CreatedAt.Format("2006-01-02")
		sshKeyCount := len(user.SSHPublicKeys)

		line := fmt.Sprintf("%s (UID: %d) | SSH Keys: %d | Created: %s",
			user.Username, user.UID, sshKeyCount, createdDate)

		if i == m.selectedUser {
			line = theme.Selection.Render("> " + line)
		} else {
			line = "  " + line
		}
		lines = append(lines, line)
	}

	// Add details for selected user
	if len(m.users) > 0 && m.selectedUser < len(m.users) {
		selectedUser := m.users[m.selectedUser]
		lines = append(lines, "")
		lines = append(lines, "📋 Selected User Details:")
		lines = append(lines, fmt.Sprintf("  Full Name: %s", selectedUser.FullName))
		lines = append(lines, fmt.Sprintf("  Email: %s", selectedUser.Email))
		lines = append(lines, fmt.Sprintf("  Home Directory: %s", selectedUser.HomeDirectory))
		lines = append(lines, fmt.Sprintf("  Shell: %s", selectedUser.Shell))
		lines = append(lines, fmt.Sprintf("  Sudo Access: %t", selectedUser.SudoAccess))
		lines = append(lines, fmt.Sprintf("  Docker Access: %t", selectedUser.DockerAccess))
	}

	content := strings.Join(lines, "\n")
	return lipgloss.NewStyle().Padding(1, 2).Render(content)
}

// renderCreateDialog renders the create user dialog
func (m UsersModel) renderCreateDialog() string {
	theme := styles.CurrentTheme

	dialog := fmt.Sprintf("Create New User\n\nUsername: %s_", m.createUsername)
	dialog += "\n\nEnter username and press Enter to create.\nPress Esc to cancel."

	return lipgloss.NewStyle().
		Width(m.width-4).
		Height(m.height-6).
		Padding(2).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(theme.Border).
		Align(lipgloss.Center, lipgloss.Center).
		Render(dialog)
}

// renderDeleteDialog renders the delete confirmation dialog
func (m UsersModel) renderDeleteDialog() string {
	theme := styles.CurrentTheme

	dialog := fmt.Sprintf("Delete User\n\nAre you sure you want to delete '%s'?\n\nThis will only remove the local configuration.\nEFS files and provisioned instances are not affected.\n\nPress 'y' to confirm or 'n' to cancel.", m.deleteUsername)

	return lipgloss.NewStyle().
		Width(m.width-4).
		Height(m.height-6).
		Padding(2).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(theme.Border).
		Align(lipgloss.Center, lipgloss.Center).
		Render(dialog)
}

// Helper functions

func isValidUsernameChar(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '-' || c == '_'
}

// TUIProfileManagerAdapter is a simplified profile manager adapter for TUI
type TUIProfileManagerAdapter struct{}

func (t *TUIProfileManagerAdapter) GetCurrentProfile() (string, error) {
	// For TUI, we'll use a simplified approach
	profileManager, err := profile.NewManagerEnhanced()
	if err != nil {
		return "default", nil
	}

	profile, err := profileManager.GetCurrentProfile()
	if err != nil {
		return "default", nil
	}

	return profile.Name, nil
}

func (t *TUIProfileManagerAdapter) GetProfileConfig(profileID string) (interface{}, error) {
	return nil, fmt.Errorf("not implemented in TUI adapter")
}

func (t *TUIProfileManagerAdapter) UpdateProfileConfig(profileID string, config interface{}) error {
	return fmt.Errorf("not implemented in TUI adapter")
}
