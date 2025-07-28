package components

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/lipgloss"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/scttfrdmn/cloudworkstation/pkg/profile"
)

// InvitationDashboard is a component for displaying and managing invitations
type InvitationDashboard struct {
	secureManager    *profile.SecureInvitationManager
	batchManager     *profile.BatchInvitationManager
	width            int
	height           int
	invitationTable  table.Model
	invitations      []*profile.InvitationToken
	selectedToken    string
	selectedIndex    int
	showDetails      bool
	detailsContent   string
	summaryStats     map[string]int
	keyMap           invitationKeyMap
	help             help.Model
}

// invitationKeyMap defines keybindings for the invitation dashboard
type invitationKeyMap struct {
	Up          key.Binding
	Down        key.Binding
	Details     key.Binding
	Revoke      key.Binding
	ExportBatch key.Binding
	Refresh     key.Binding
	Back        key.Binding
	Help        key.Binding
}

// NewInvitationKeyMap creates a new key map for the invitation dashboard
func NewInvitationKeyMap() invitationKeyMap {
	return invitationKeyMap{
		Up: key.NewBinding(
			key.WithKeys("up", "k"),
			key.WithHelp("↑/k", "up"),
		),
		Down: key.NewBinding(
			key.WithKeys("down", "j"),
			key.WithHelp("↓/j", "down"),
		),
		Details: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "view details"),
		),
		Revoke: key.NewBinding(
			key.WithKeys("r"),
			key.WithHelp("r", "revoke invitation"),
		),
		ExportBatch: key.NewBinding(
			key.WithKeys("e"),
			key.WithHelp("e", "export batch"),
		),
		Refresh: key.NewBinding(
			key.WithKeys("f5"),
			key.WithHelp("f5", "refresh"),
		),
		Back: key.NewBinding(
			key.WithKeys("esc"),
			key.WithHelp("esc", "back"),
		),
		Help: key.NewBinding(
			key.WithKeys("?"),
			key.WithHelp("?", "help"),
		),
	}
}

// NewInvitationDashboard creates a new invitation dashboard component
func NewInvitationDashboard(secureManager *profile.SecureInvitationManager) *InvitationDashboard {
	batchManager := profile.NewBatchInvitationManager(secureManager)
	
	columns := []table.Column{
		{Title: "Name", Width: 25},
		{Title: "Type", Width: 10},
		{Title: "Expires In", Width: 15},
		{Title: "Token", Width: 20},
		{Title: "Security", Width: 15},
	}

	t := table.New(
		table.WithColumns(columns),
		table.WithFocused(true),
		table.WithHeight(10),
	)

	// Set table styles
	s := table.DefaultStyles()
	s.Header = s.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderBottom(true).
		Bold(true)
	s.Selected = s.Selected.
		Foreground(lipgloss.Color("229")).
		Background(lipgloss.Color("57")).
		Bold(true)
	t.SetStyles(s)

	h := help.New()

	return &InvitationDashboard{
		secureManager:   secureManager,
		batchManager:    batchManager,
		invitationTable: t,
		keyMap:          NewInvitationKeyMap(),
		help:            h,
		summaryStats:    make(map[string]int),
	}
}

// Init initializes the invitation dashboard
func (d *InvitationDashboard) Init() tea.Cmd {
	return d.loadInvitations
}

// loadInvitations is a command that loads invitation data
func (d *InvitationDashboard) loadInvitations() tea.Msg {
	invitations := d.secureManager.ListInvitations()
	return invitationListMsg(invitations)
}

// invitationListMsg is a message containing invitation data
type invitationListMsg []*profile.InvitationToken

// Update handles events and updates the invitation dashboard state
func (d *InvitationDashboard) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		d.width = msg.Width
		d.height = msg.Height
		d.invitationTable.SetHeight(d.height - 12) // Allow space for header, footer, stats
		d.invitationTable.SetWidth(d.width - 4)
		return d, nil

	case tea.KeyMsg:
		switch {
		case key.Matches(msg, d.keyMap.Up):
			if d.showDetails {
				return d, nil
			}
			d.invitationTable.MoveUp(1)
			index := d.invitationTable.Cursor()
			if index < len(d.invitations) {
				d.selectedIndex = index
				d.selectedToken = d.invitations[index].Token
			}
			return d, nil

		case key.Matches(msg, d.keyMap.Down):
			if d.showDetails {
				return d, nil
			}
			d.invitationTable.MoveDown(1)
			index := d.invitationTable.Cursor()
			if index < len(d.invitations) {
				d.selectedIndex = index
				d.selectedToken = d.invitations[index].Token
			}
			return d, nil

		case key.Matches(msg, d.keyMap.Details):
			if d.showDetails {
				d.showDetails = false
				return d, nil
			}

			if len(d.invitations) == 0 {
				return d, nil
			}

			d.showDetails = true
			selectedInvitation := d.invitations[d.selectedIndex]
			
			// Get device information if available
			deviceInfo := "Device information not available"
			devices, err := d.secureManager.GetInvitationDevices(selectedInvitation.Token)
			if err == nil && len(devices) > 0 {
				deviceInfo = fmt.Sprintf("Registered devices: %d\n", len(devices))
				for i, device := range devices {
					deviceInfo += fmt.Sprintf("  Device %d: %s\n", i+1, device["device_id"])
					if timestamp, ok := device["registered_at"].(string); ok {
						deviceInfo += fmt.Sprintf("    Registered: %s\n", timestamp)
					}
				}
			}

			// Build details view
			d.detailsContent = fmt.Sprintf(`
Invitation Details

Name: %s
Type: %s
Token: %s
Created: %s
Expires: %s (in %s)

Security Settings:
- Can Invite Others: %t
- Transferable: %t
- Device Bound: %t
- Maximum Devices: %d
- Parent Token: %s

%s

Press ESC or ENTER to close
`,
				selectedInvitation.Name,
				selectedInvitation.Type,
				selectedInvitation.Token,
				selectedInvitation.Created.Format("Jan 2, 2006 15:04:05"),
				selectedInvitation.Expires.Format("Jan 2, 2006 15:04:05"),
				selectedInvitation.GetExpirationDuration().Round(time.Hour),
				selectedInvitation.CanInvite,
				selectedInvitation.Transferable,
				selectedInvitation.DeviceBound,
				selectedInvitation.MaxDevices,
				selectedInvitation.ParentToken,
				deviceInfo,
			)

			return d, nil

		case key.Matches(msg, d.keyMap.Back):
			if d.showDetails {
				d.showDetails = false
				return d, nil
			}
			// Return to previous view handled by parent component

		case key.Matches(msg, d.keyMap.Refresh):
			return d, d.loadInvitations

		case key.Matches(msg, d.keyMap.Revoke):
			if d.showDetails || len(d.invitations) == 0 {
				return d, nil
			}
			
			token := d.invitations[d.selectedIndex].Token
			return d, func() tea.Msg {
				err := d.secureManager.RevokeInvitation(token)
				if err != nil {
					return errMsg{err}
				}
				return d.loadInvitations()
			}

		case key.Matches(msg, d.keyMap.ExportBatch):
			// This would typically open a file dialog in a real application
			// For TUI, we'll just initiate the export to a default file
			return d, func() tea.Msg {
				if len(d.invitations) == 0 {
					return errMsg{fmt.Errorf("no invitations to export")}
				}
				
				// Convert to batch invitations
				batchInvitations := make([]*profile.BatchInvitation, len(d.invitations))
				for i, inv := range d.invitations {
					// Create encoded form for sharing
					encodedData, _ := inv.EncodeToString()
					
					batchInvitations[i] = &profile.BatchInvitation{
						Name:        inv.Name,
						Type:        inv.Type,
						ValidDays:   int(inv.GetExpirationDuration().Hours() / 24),
						CanInvite:   inv.CanInvite,
						Transferable: inv.Transferable,
						DeviceBound: inv.DeviceBound,
						MaxDevices:  inv.MaxDevices,
						Token:       inv.Token,
						EncodedData: encodedData,
					}
				}
				
				// Create batch result
				results := &profile.BatchInvitationResult{
					Successful:     batchInvitations,
					Failed:         []*profile.BatchInvitation{},
					TotalProcessed: len(batchInvitations),
					TotalSuccessful: len(batchInvitations),
					TotalFailed:    0,
				}
				
				// Export to default file - in a real app, this would use a file picker
				err := d.batchManager.ExportBatchInvitationsToCSVFile(
					"invitations_export.csv", results, true)
				
				if err != nil {
					return errMsg{err}
				}
				
				return successMsg("Successfully exported invitations to invitations_export.csv")
			}
		}

	case invitationListMsg:
		d.invitations = msg
		d.updateTable()
		d.updateStats()
		d.selectedIndex = 0
		if len(d.invitations) > 0 {
			d.selectedToken = d.invitations[0].Token
		} else {
			d.selectedToken = ""
		}
		return d, nil

	case errMsg:
		// Error handling would be implemented here
		return d, nil

	case successMsg:
		// Success message handling
		return d, nil
	}

	// Handle table updates
	d.invitationTable, cmd = d.invitationTable.Update(msg)
	cmds = append(cmds, cmd)

	return d, tea.Batch(cmds...)
}

// updateTable refreshes the invitation table data
func (d *InvitationDashboard) updateTable() {
	rows := []table.Row{}

	sort.Slice(d.invitations, func(i, j int) bool {
		// Sort by expiration date (ascending)
		return d.invitations[i].Expires.Before(d.invitations[j].Expires)
	})

	for _, inv := range d.invitations {
		// Create security summary
		security := []string{}
		if inv.DeviceBound {
			security = append(security, "Device-bound")
		}
		if inv.CanInvite {
			security = append(security, "Can invite")
		}
		if inv.Transferable {
			security = append(security, "Transferable")
		}
		securityStr := strings.Join(security, ", ")
		if len(securityStr) > 15 {
			securityStr = securityStr[:12] + "..."
		}

		// Format expiration time
		expiresIn := inv.GetExpirationDuration()
		expiresStr := "Expired"
		if expiresIn > 0 {
			if expiresIn < 24*time.Hour {
				expiresStr = fmt.Sprintf("%d hours", int(expiresIn.Hours()))
			} else if expiresIn < 48*time.Hour {
				expiresStr = "1 day"
			} else {
				expiresStr = fmt.Sprintf("%d days", int(expiresIn.Hours()/24))
			}
		}

		// Truncate token for display
		tokenDisplay := inv.Token
		if len(tokenDisplay) > 18 {
			tokenDisplay = tokenDisplay[:15] + "..."
		}

		rows = append(rows, table.Row{
			inv.Name,
			string(inv.Type),
			expiresStr,
			tokenDisplay,
			securityStr,
		})
	}

	d.invitationTable.SetRows(rows)
}

// updateStats calculates summary statistics for the dashboard
func (d *InvitationDashboard) updateStats() {
	// Reset stats
	d.summaryStats = map[string]int{
		"total":        0,
		"admin":        0,
		"read_write":   0,
		"read_only":    0,
		"device_bound": 0,
		"expiring_soon": 0,
	}

	for _, inv := range d.invitations {
		d.summaryStats["total"]++
		
		// Count by type
		d.summaryStats[string(inv.Type)]++
		
		// Count security features
		if inv.DeviceBound {
			d.summaryStats["device_bound"]++
		}
		
		// Count expiring soon (within 7 days)
		if inv.GetExpirationDuration() < 7*24*time.Hour {
			d.summaryStats["expiring_soon"]++
		}
	}
}

// View renders the invitation dashboard
func (d *InvitationDashboard) View() string {
	if len(d.invitations) == 0 {
		return d.renderEmptyState()
	}

	if d.showDetails {
		return d.renderDetailsView()
	}

	return d.renderDashboardView()
}

// renderEmptyState renders the view when no invitations exist
func (d *InvitationDashboard) renderEmptyState() string {
	emptyStyle := lipgloss.NewStyle().
		Align(lipgloss.Center).
		Foreground(lipgloss.Color("240")).
		MarginTop(2)

	return fmt.Sprintf(`
%s

No invitations found

Press F5 to refresh
Press E to create a batch export
Press ? for help
`,
		emptyStyle.Render("Invitation Dashboard"),
	)
}

// renderDetailsView renders the detailed view of a selected invitation
func (d *InvitationDashboard) renderDetailsView() string {
	detailsStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("240")).
		Padding(1).
		Width(d.width - 8)

	return detailsStyle.Render(d.detailsContent)
}

// renderDashboardView renders the main dashboard view
func (d *InvitationDashboard) renderDashboardView() string {
	headerStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("39")).
		MarginBottom(1)

	statsStyle := lipgloss.NewStyle().
		Border(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("240")).
		Padding(0, 1).
		MarginTop(1).
		MarginBottom(1)

	// Build stats display
	statsContent := fmt.Sprintf(
		"Total: %d | Admin: %d | Read/Write: %d | Read-Only: %d | Device-Bound: %d | Expiring Soon: %d",
		d.summaryStats["total"],
		d.summaryStats["admin"],
		d.summaryStats["read_write"],
		d.summaryStats["read_only"],
		d.summaryStats["device_bound"],
		d.summaryStats["expiring_soon"],
	)

	helpView := d.help.View(d.keyMap)
	
	return fmt.Sprintf("%s\n%s\n%s\n%s",
		headerStyle.Render("Invitation Dashboard"),
		statsStyle.Render(statsContent),
		d.invitationTable.View(),
		helpView,
	)
}

// errMsg represents an error message
type errMsg struct {
	error
}

// successMsg represents a success message
type successMsg string