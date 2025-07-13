package models

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/scttfrdmn/cloudworkstation/internal/tui/components"
	"github.com/scttfrdmn/cloudworkstation/internal/tui/styles"
	"github.com/scttfrdmn/cloudworkstation/pkg/api"
	"github.com/scttfrdmn/cloudworkstation/pkg/types"
)

// IdleInstanceItem represents an instance with idle status
type IdleInstanceItem struct {
	name            string
	state           string
	idleEnabled     bool
	idleTime        int
	threshold       int
	action          string
	actionScheduled bool
	actionTime      time.Time
}

// FilterValue returns the value to filter on in the list
func (i IdleInstanceItem) FilterValue() string { return i.name }

// Title returns the name of the instance
func (i IdleInstanceItem) Title() string { return i.name }

// Description returns the idle status description
func (i IdleInstanceItem) Description() string {
	var status string
	if !i.idleEnabled {
		status = "Idle detection disabled"
	} else {
		status = fmt.Sprintf("Idle time: %d/%d min", i.idleTime, i.threshold)
		
		if i.actionScheduled {
			timeLeft := time.Until(i.actionTime)
			if timeLeft > 0 {
				minutes := int(timeLeft.Minutes())
				status += fmt.Sprintf(" | %s in %d min", strings.ToUpper(i.action), minutes)
			} else {
				status += fmt.Sprintf(" | %s pending", strings.ToUpper(i.action))
			}
		}
	}
	
	return fmt.Sprintf("State: %s | %s", strings.ToUpper(i.state), status)
}

// IdleInstancesModel represents the model for idle instance monitoring
type IdleInstancesModel struct {
	apiClient     apiClient
	instanceList  list.Model
	statusBar     components.StatusBar
	spinner       components.Spinner
	width         int
	height        int
	loading       bool
	error         string
	instances     []api.InstanceResponse
	selected      string
	confirmEnable bool
	confirmPolicy string
}

// NewIdleInstancesModel creates a new idle instances model
func NewIdleInstancesModel(apiClient apiClient) IdleInstancesModel {
	theme := styles.CurrentTheme
	
	// Set up instance list
	instanceList := list.New([]list.Item{}, list.NewDefaultDelegate(), 0, 0)
	instanceList.Title = "Instances Idle Status"
	instanceList.Styles.Title = theme.Title
	instanceList.Styles.PaginationStyle = theme.Pagination
	instanceList.Styles.HelpStyle = theme.Help
	
	// Create status bar and spinner
	statusBar := components.NewStatusBar("Idle Instance Monitoring", "")
	spinner := components.NewSpinner("Loading instance idle status...")
	
	return IdleInstancesModel{
		apiClient:    apiClient,
		instanceList: instanceList,
		statusBar:    statusBar,
		spinner:      spinner,
		width:        80,
		height:       24,
		loading:      true,
		instances:    []api.InstanceResponse{},
	}
}

// Init initializes the model
func (m IdleInstancesModel) Init() tea.Cmd {
	return tea.Batch(
		m.spinner.InitialCmd(),
		m.fetchInstances,
	)
}

// fetchInstances retrieves instance data from the API
func (m IdleInstancesModel) fetchInstances() tea.Msg {
	response, err := m.apiClient.ListInstances(context.Background())
	if err != nil {
		return fmt.Errorf("failed to list instances: %w", err)
	}
	return response
}

// enableIdleDetection enables idle detection for the selected instance
func (m IdleInstancesModel) enableIdleDetection() tea.Cmd {
	return func() tea.Msg {
		if m.selected == "" {
			return fmt.Errorf("no instance selected")
		}
		
		// In a real app, this would call the API
		
		// Create success message
		return IdleActionResult{
			Success: true,
			Message: fmt.Sprintf("Enabled idle detection for %s with policy %s", 
				m.selected, m.confirmPolicy),
		}
	}
}

// disableIdleDetection disables idle detection for the selected instance
func (m IdleInstancesModel) disableIdleDetection() tea.Cmd {
	return func() tea.Msg {
		if m.selected == "" {
			return fmt.Errorf("no instance selected")
		}
		
		// In a real app, this would call the API
		
		// Create success message
		return IdleActionResult{
			Success: true,
			Message: fmt.Sprintf("Disabled idle detection for %s", m.selected),
		}
	}
}

// fetchPolicies retrieves idle policies from the API
func (m IdleInstancesModel) fetchPolicies() tea.Msg {
	// In a real app, this would call the API
	// For now, return sample data
	return []types.IdlePolicy{
		{
			Name:        "default",
			Description: "Default idle policy for all instances",
			Threshold:   30,
			Action:      "stop",
			AppliesTo:   []string{"all"},
		},
		{
			Name:        "gpu-instances",
			Description: "Aggressive idle policy for cost-intensive GPU instances",
			Threshold:   15,
			Action:      "stop",
			AppliesTo:   []string{"g4dn", "p3", "p4d"},
		},
		{
			Name:        "development",
			Description: "Lenient policy for development instances",
			Threshold:   60,
			Action:      "notify",
			AppliesTo:   []string{"t3", "t4g", "m5", "m6g"},
		},
	}
}

// IdleActionResult represents the result of an idle action
type IdleActionResult struct {
	Success bool
	Message string
	Error   error
}

// Update handles messages and updates the model
func (m IdleInstancesModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.statusBar.SetWidth(msg.Width)
		
		// Update list dimensions
		listHeight := m.height - 16 // Account for title, details, status
		if listHeight < 3 {
			listHeight = 3
		}
		m.instanceList.SetHeight(listHeight)
		m.instanceList.SetWidth(m.width - 4)
		
		return m, nil
		
	case tea.KeyMsg:
		// Handle policy selection
		if m.confirmEnable {
			switch msg.String() {
			case "1", "2", "3", "d", "g", "l":
				// Map key to policy
				var policy string
				switch msg.String() {
				case "1", "d":
					policy = "default"
				case "2", "g":
					policy = "gpu-instances"
				case "3", "l":
					policy = "development"
				}
				
				m.confirmPolicy = policy
				m.confirmEnable = false
				return m, m.enableIdleDetection()
				
			case "esc", "q":
				m.confirmEnable = false
				return m, nil
			}
			
			return m, nil
		}
		
		// Handle normal keys
		switch msg.String() {
		case "e":
			// Enable idle detection
			if m.selected != "" {
				m.confirmEnable = true
				return m, m.fetchPolicies
			}
			
		case "d":
			// Disable idle detection
			if m.selected != "" {
				// Find if the selected instance has idle detection enabled
				var hasIdleEnabled bool
				for _, instance := range m.instances {
					if instance.Name == m.selected && 
						instance.IdleDetection != nil && 
						instance.IdleDetection.Enabled {
						hasIdleEnabled = true
						break
					}
				}
				
				if hasIdleEnabled {
					return m, m.disableIdleDetection()
				}
			}
			
		case "r":
			// Refresh instance list
			m.loading = true
			m.error = ""
			return m, m.fetchInstances
			
		case "q", "esc":
			return m, tea.Quit
		}
		
		// Update list selection
		if !m.loading {
			var cmd tea.Cmd
			m.instanceList, cmd = m.instanceList.Update(msg)
			cmds = append(cmds, cmd)
			
			// Update selected instance
			if i, ok := m.instanceList.SelectedItem().(IdleInstanceItem); ok {
				m.selected = i.name
			}
		}
		
	case IdleActionResult:
		m.loading = false
		if msg.Success {
			m.statusBar.SetStatus(msg.Message, components.StatusSuccess)
			return m, m.fetchInstances
		} else if msg.Error != nil {
			m.error = msg.Error.Error()
			m.statusBar.SetStatus(fmt.Sprintf("Error: %s", m.error), components.StatusError)
		}
		
	case []types.IdlePolicy:
		// For policy selection dialog
		return m, nil
		
	case *api.ListInstancesResponse:
		m.loading = false
		m.instances = msg.Instances
		
		// Create idle instance items
		var items []list.Item
		for _, instance := range msg.Instances {
			item := IdleInstanceItem{
				name:  instance.Name,
				state: instance.State,
			}
			
			// Add idle information if available
			if instance.IdleDetection != nil {
				item.idleEnabled = instance.IdleDetection.Enabled
				item.idleTime = instance.IdleDetection.IdleTime
				item.threshold = instance.IdleDetection.Threshold
				item.action = instance.IdleDetection.Policy
				item.actionScheduled = instance.IdleDetection.ActionPending
				item.actionTime = instance.IdleDetection.ActionSchedule
			}
			
			items = append(items, item)
		}
		
		// Sort items by idle time (highest first)
		sort.Slice(items, func(i, j int) bool {
			return items[i].(IdleInstanceItem).idleTime > items[j].(IdleInstanceItem).idleTime
		})
		
		m.instanceList.SetItems(items)
		m.statusBar.SetStatus(fmt.Sprintf("Loaded %d instances", len(items)), components.StatusSuccess)
		
		// Select first item if none selected
		if len(items) > 0 && m.selected == "" {
			m.selected = items[0].(IdleInstanceItem).name
		}
		
	case error:
		m.loading = false
		m.error = msg.Error()
		m.statusBar.SetStatus(fmt.Sprintf("Error: %s", m.error), components.StatusError)
	}
	
	// Update spinner when loading
	if m.loading {
		var spinnerCmd tea.Cmd
		m.spinner, spinnerCmd = m.spinner.Update(msg)
		cmds = append(cmds, spinnerCmd)
	}
	
	// Add auto-refresh command (every 30 seconds)
	cmds = append(cmds, tea.Tick(30*time.Second, func(t time.Time) tea.Msg {
		return RefreshMsg{}
	}))
	
	return m, tea.Batch(cmds...)
}

// View renders the idle instances view
func (m IdleInstancesModel) View() string {
	theme := styles.CurrentTheme
	
	// Title section
	title := theme.Title.Render("CloudWorkstation Idle Monitoring")
	
	// Content area
	var content string
	
	if m.loading {
		content = lipgloss.NewStyle().
			Width(m.width).
			Height(m.height - 8). // Account for title and status
			Align(lipgloss.Center, lipgloss.Center).
			Render(m.spinner.View())
	} else if m.error != "" {
		content = lipgloss.NewStyle().
			Width(m.width).
			Height(m.height - 8).
			Align(lipgloss.Center, lipgloss.Center).
			Render(theme.StatusError.Render("Error: " + m.error))
	} else if m.confirmEnable {
		// Policy selection dialog
		policyPanel := theme.Panel.Copy().Width(m.width - 20).Render(
			lipgloss.JoinVertical(
				lipgloss.Left,
				theme.PanelHeader.Render("Select Idle Policy"),
				"",
				fmt.Sprintf("Select a policy for instance: %s", m.selected),
				"",
				"1. default - Default idle policy (30 min → stop)",
				"2. gpu-instances - GPU instance policy (15 min → stop)",
				"3. development - Development policy (60 min → notify)",
				"",
				"Press the key for your desired policy, Esc to cancel",
			),
		)
		
		content = lipgloss.NewStyle().
			Width(m.width).
			Height(m.height - 8).
			Align(lipgloss.Center, lipgloss.Center).
			Render(policyPanel)
	} else {
		// List view
		listPanel := theme.Panel.Copy().Width(m.width - 4).Render(
			lipgloss.JoinVertical(
				lipgloss.Left,
				m.instanceList.View(),
			),
		)
		
		// Detail panel for selected instance
		var detailPanel string
		for _, instance := range m.instances {
			if instance.Name == m.selected {
				// Build idle information
				var idleInfo string
				if instance.IdleDetection == nil || !instance.IdleDetection.Enabled {
					idleInfo = theme.Warning.Render("Idle detection is disabled for this instance")
				} else {
					idleInfo = fmt.Sprintf("Idle time: %d minutes\n", instance.IdleDetection.IdleTime)
					idleInfo += fmt.Sprintf("Policy: %s\n", instance.IdleDetection.Policy)
					idleInfo += fmt.Sprintf("Threshold: %d minutes\n", instance.IdleDetection.Threshold)
					
					if instance.IdleDetection.ActionPending {
						timeLeft := time.Until(instance.IdleDetection.ActionSchedule)
						if timeLeft > 0 {
							idleInfo += fmt.Sprintf("Action scheduled: %s in %d minutes\n", 
								strings.ToUpper(instance.IdleDetection.Policy),
								int(timeLeft.Minutes()))
						} else {
							idleInfo += fmt.Sprintf("Action pending: %s\n", 
								strings.ToUpper(instance.IdleDetection.Policy))
						}
					}
				}
				
				// Build action text
				actionText := ""
				if instance.IdleDetection == nil || !instance.IdleDetection.Enabled {
					actionText = "Press 'e' to enable idle detection"
				} else {
					actionText = "Press 'd' to disable idle detection"
				}
				
				detailPanel = theme.Panel.Copy().Width(m.width - 4).Render(
					lipgloss.JoinVertical(
						lipgloss.Left,
						theme.PanelHeader.Render("Idle Status: " + instance.Name),
						"",
						fmt.Sprintf("Instance State: %s", strings.ToUpper(instance.State)),
						fmt.Sprintf("Instance Type: %s", instance.InstanceType),
						"",
						idleInfo,
						"",
						actionText,
					),
				)
				break
			}
		}
		
		content = lipgloss.JoinVertical(
			lipgloss.Left,
			listPanel,
			detailPanel,
		)
	}
	
	// Help text
	var help string
	if m.confirmEnable {
		help = theme.Help.Render("1-3: select policy • esc: cancel")
	} else {
		help = theme.Help.Render("e: enable • d: disable • r: refresh • q: quit • ↑/↓: navigate")
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