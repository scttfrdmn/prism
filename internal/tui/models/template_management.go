package models

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/scttfrdmn/cloudworkstation/internal/tui/components"
	"github.com/scttfrdmn/cloudworkstation/internal/tui/styles"
	"github.com/scttfrdmn/cloudworkstation/pkg/api"
)

// TemplateItem represents a template in the list
type TemplateItem struct {
	name        string
	description string
	baseAMI     string
	instanceType string
	capabilities []string
	tags        map[string]string
	costs       map[string]float64
}

// FilterValue returns the value to filter on in the list
func (t TemplateItem) FilterValue() string { return t.name }

// Title returns the name of the template
func (t TemplateItem) Title() string { return t.name }

// Description returns a short description of the template
func (t TemplateItem) Description() string {
	// Calculate the cost for display
	cost := 0.0
	if val, ok := t.costs["default"]; ok {
		cost = val
	}
	
	// Show tags if available
	var tagStr string
	if len(t.tags) > 0 {
		tags := make([]string, 0, len(t.tags))
		for k, v := range t.tags {
			tags = append(tags, fmt.Sprintf("%s: %s", k, v))
		}
		tagStr = " | Tags: " + strings.Join(tags[:1], ", ")
		if len(tags) > 1 {
			tagStr += fmt.Sprintf(" (+%d more)", len(tags)-1)
		}
	}
	
	// Show capabilities if available
	var capStr string
	if len(t.capabilities) > 0 {
		caps := t.capabilities
		if len(caps) > 2 {
			capStr = " | Caps: " + strings.Join(caps[:2], ", ")
			capStr += fmt.Sprintf(" (+%d more)", len(caps)-2)
		} else {
			capStr = " | Caps: " + strings.Join(caps, ", ")
		}
	}
	
	return fmt.Sprintf("%s | %s | $%.2f/day%s%s", 
		t.description, t.instanceType, cost, tagStr, capStr)
}

// TemplateManagementModel represents the template management view
type TemplateManagementModel struct {
	apiClient      apiClient
	templateList   list.Model
	statusBar      components.StatusBar
	spinner        components.Spinner
	tabs           components.TabBar
	width          int
	height         int
	loading        bool
	error          string
	templates      map[string]api.TemplateResponse
	selected       string
	mode           string // list, details, edit
	searchInput    textinput.Model
	searchActive   bool
}

// NewTemplateManagementModel creates a new template management model
func NewTemplateManagementModel(apiClient apiClient) TemplateManagementModel {
	theme := styles.CurrentTheme
	
	// Set up template list
	templateList := list.New([]list.Item{}, list.NewDefaultDelegate(), 0, 0)
	templateList.Title = "CloudWorkstation Templates"
	templateList.Styles.Title = theme.Title
	templateList.Styles.PaginationStyle = theme.Pagination
	templateList.Styles.HelpStyle = theme.Help
	
	// Create tabs
	tabs := components.NewTabBar(
		[]string{"Templates", "Details", "Repositories"},
		0,
	)
	
	// Create status bar and spinner
	statusBar := components.NewStatusBar("Template Management", "")
	spinner := components.NewSpinner("Loading templates...")
	
	// Create search input
	searchInput := textinput.New()
	searchInput.Placeholder = "Search templates..."
	searchInput.Width = 30
	
	return TemplateManagementModel{
		apiClient:    apiClient,
		templateList: templateList,
		statusBar:    statusBar,
		spinner:      spinner,
		tabs:         tabs,
		searchInput:  searchInput,
		width:        80,
		height:       24,
		loading:      true,
		mode:         "list",
		templates:    make(map[string]api.TemplateResponse),
	}
}

// Init initializes the model
func (m TemplateManagementModel) Init() tea.Cmd {
	return tea.Batch(
		m.spinner.InitialCmd(),
		m.fetchTemplates,
	)
}

// fetchTemplates retrieves template data from the API
func (m TemplateManagementModel) fetchTemplates() tea.Msg {
	response, err := m.apiClient.ListTemplates(context.Background())
	if err != nil {
		return fmt.Errorf("failed to list templates: %w", err)
	}
	return response
}

// searchTemplates filters templates based on search query
func (m *TemplateManagementModel) searchTemplates(query string) {
	query = strings.ToLower(query)
	
	// If query is empty, show all templates
	if query == "" {
		var items []list.Item
		for name, template := range m.templates {
			items = append(items, createTemplateItem(name, template))
		}
		
		// Sort alphabetically
		sort.Slice(items, func(i, j int) bool {
			return items[i].(TemplateItem).name < items[j].(TemplateItem).name
		})
		
		m.templateList.SetItems(items)
		return
	}
	
	// Filter templates based on query
	var items []list.Item
	for name, template := range m.templates {
		// Check if name or description contains query
		if strings.Contains(strings.ToLower(name), query) || 
		   strings.Contains(strings.ToLower(template.Description), query) {
			items = append(items, createTemplateItem(name, template))
		}
		
		// Check capabilities
		for _, cap := range template.Capabilities {
			if strings.Contains(strings.ToLower(cap), query) {
				items = append(items, createTemplateItem(name, template))
				break
			}
		}
		
		// Check tags
		for k, v := range template.Tags {
			if strings.Contains(strings.ToLower(k), query) || 
			   strings.Contains(strings.ToLower(v), query) {
				items = append(items, createTemplateItem(name, template))
				break
			}
		}
	}
	
	// Sort alphabetically
	sort.Slice(items, func(i, j int) bool {
		return items[i].(TemplateItem).name < items[j].(TemplateItem).name
	})
	
	m.templateList.SetItems(items)
}

// createTemplateItem creates a template item from a template response
func createTemplateItem(name string, template api.TemplateResponse) TemplateItem {
	return TemplateItem{
		name:         name,
		description:  template.Description,
		baseAMI:      template.BaseAMI,
		instanceType: template.InstanceType,
		capabilities: template.Capabilities,
		tags:         template.Tags,
		costs:        template.Costs,
	}
}

// Update handles messages and updates the model
func (m TemplateManagementModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.statusBar.SetWidth(msg.Width)
		m.tabs.SetWidth(msg.Width)
		
		// Update list height to account for title, tabs, and search
		listHeight := m.height - 12 // title + tabs + search + status bar + help
		if listHeight < 3 {
			listHeight = 3
		}
		m.templateList.SetHeight(listHeight)
		m.templateList.SetWidth(m.width - 4)
		
		return m, nil
		
	case tea.KeyMsg:
		// Handle search input first if active
		if m.searchActive {
			switch msg.String() {
			case "esc":
				m.searchActive = false
				m.searchInput.Blur()
				return m, nil
				
			case "enter":
				m.searchActive = false
				m.searchInput.Blur()
				m.searchTemplates(m.searchInput.Value())
				return m, nil
			}
			
			// Update search input
			var cmd tea.Cmd
			m.searchInput, cmd = m.searchInput.Update(msg)
			return m, cmd
		}
		
		// Handle normal key presses
		switch msg.String() {
		case "/":
			// Activate search
			m.searchActive = true
			m.searchInput.Focus()
			return m, nil
			
		case "tab":
			// Cycle through tabs
			cmd := m.tabs.Next()
			
			// Update mode based on tab
			switch m.tabs.ActiveTab() {
			case 0:
				m.mode = "list"
			case 1:
				m.mode = "details"
			case 2:
				m.mode = "repos"
			}
			
			cmds = append(cmds, cmd)
			
		case "r":
			m.loading = true
			m.error = ""
			return m, m.fetchTemplates
			
		case "l":
			// Launch template
			if m.selected != "" {
				// In a real app, this would navigate to a launch form
				m.statusBar.SetStatus(fmt.Sprintf("Launching template: %s", m.selected), components.StatusInfo)
			}
			
		case "c":
			// Clone template (for customization)
			if m.selected != "" {
				// In a real app, this would create a new template based on the selected one
				m.statusBar.SetStatus(fmt.Sprintf("Cloning template: %s", m.selected), components.StatusInfo)
			}
			
		case "q", "esc":
			return m, tea.Quit
		}
		
		// Update list selection
		if m.mode == "list" && !m.loading {
			var cmd tea.Cmd
			m.templateList, cmd = m.templateList.Update(msg)
			cmds = append(cmds, cmd)
			
			// Update selected template
			if i, ok := m.templateList.SelectedItem().(TemplateItem); ok {
				m.selected = i.name
			}
		}
		
	case api.ListTemplatesResponse:
		m.loading = false
		m.templates = msg.Templates
		
		// Update template list items
		var items []list.Item
		for name, template := range m.templates {
			items = append(items, createTemplateItem(name, template))
		}
		
		// Sort alphabetically
		sort.Slice(items, func(i, j int) bool {
			return items[i].(TemplateItem).name < items[j].(TemplateItem).name
		})
		
		m.templateList.SetItems(items)
		m.statusBar.SetStatus(fmt.Sprintf("Loaded %d templates", len(items)), components.StatusSuccess)
		
		// Select first item if none selected
		if len(items) > 0 && m.selected == "" {
			m.selected = items[0].(TemplateItem).name
		}
		
		// Schedule periodic refresh
		return m, tea.Tick(5*time.Minute, func(t time.Time) tea.Msg {
			return RefreshMsg{}
		})
		
	case RefreshMsg:
		m.loading = true
		m.error = ""
		return m, m.fetchTemplates
		
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
	
	return m, tea.Batch(cmds...)
}

// View renders the template management view
func (m TemplateManagementModel) View() string {
	theme := styles.CurrentTheme
	
	// Title section
	title := theme.Title.Render("CloudWorkstation Template Management")
	
	// Tab bar
	tabBar := m.tabs.View()
	
	// Search bar
	searchBar := lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(theme.MutedColor).
		Padding(0, 1).
		Width(m.width - 4).
		Render(m.searchInput.View())
		
	// Content area
	var content string
	
	if m.loading {
		content = lipgloss.NewStyle().
			Width(m.width).
			Height(m.height - 12). // Account for title, tabs, search, status bar, help
			Align(lipgloss.Center, lipgloss.Center).
			Render(m.spinner.View())
	} else if m.error != "" {
		content = lipgloss.NewStyle().
			Width(m.width).
			Height(m.height - 12).
			Align(lipgloss.Center, lipgloss.Center).
			Render(theme.StatusError.Render("Error: " + m.error))
	} else {
		switch m.mode {
		case "list":
			content = m.renderTemplateList()
		case "details":
			content = m.renderTemplateDetails()
		case "repos":
			content = m.renderRepositories()
		}
	}
	
	// Help text
	var help string
	switch m.mode {
	case "list":
		help = theme.Help.Render("↑/↓: navigate • /: search • l: launch • c: clone • tab: details • r: refresh • q: quit")
	case "details":
		help = theme.Help.Render("tab: repositories • esc: back")
	case "repos":
		help = theme.Help.Render("tab: templates • esc: back")
	}
	
	// Join everything together
	return lipgloss.JoinVertical(
		lipgloss.Left,
		title,
		"",
		tabBar,
		searchBar,
		content,
		"",
		m.statusBar.View(),
		help,
	)
}

// renderTemplateList renders the template list view
func (m TemplateManagementModel) renderTemplateList() string {
	theme := styles.CurrentTheme
	
	// Template list
	listPanel := theme.Panel.Copy().
		Width(m.width - 4).
		Render(m.templateList.View())
	
	return listPanel
}

// renderTemplateDetails renders the template details view
func (m TemplateManagementModel) renderTemplateDetails() string {
	theme := styles.CurrentTheme
	
	// Get the selected template
	template, ok := m.templates[m.selected]
	if !ok {
		return theme.Panel.Copy().Width(m.width - 4).Render("No template selected")
	}
	
	// Format the details
	var details []string
	
	// Basic info
	details = append(details, theme.SectionTitle.Render(m.selected))
	details = append(details, "")
	details = append(details, theme.Label.Render("Description: ") + theme.Text.Render(template.Description))
	details = append(details, theme.Label.Render("Instance Type: ") + theme.Text.Render(template.InstanceType))
	details = append(details, theme.Label.Render("Base AMI: ") + theme.Text.Render(template.BaseAMI))
	details = append(details, "")
	
	// Cost information
	details = append(details, theme.SubTitle.Render("Cost Information:"))
	for region, cost := range template.Costs {
		details = append(details, fmt.Sprintf("%s: $%.2f per day", region, cost))
	}
	details = append(details, "")
	
	// Capabilities
	details = append(details, theme.SubTitle.Render("Capabilities:"))
	for _, cap := range template.Capabilities {
		details = append(details, "• "+cap)
	}
	details = append(details, "")
	
	// Tags
	details = append(details, theme.SubTitle.Render("Tags:"))
	for k, v := range template.Tags {
		details = append(details, fmt.Sprintf("%s: %s", k, v))
	}
	details = append(details, "")
	
	// Actions
	details = append(details, theme.SubTitle.Render("Actions:"))
	details = append(details, "• Press 'l' to launch this template")
	details = append(details, "• Press 'c' to clone and customize this template")
	
	detailsPanel := theme.Panel.Copy().
		Width(m.width - 4).
		Render(lipgloss.JoinVertical(lipgloss.Left, details...))
	
	return detailsPanel
}

// renderRepositories renders the repositories view
func (m TemplateManagementModel) renderRepositories() string {
	theme := styles.CurrentTheme
	
	// For now, just display a placeholder
	return theme.Panel.Copy().
		Width(m.width - 4).
		Render(
			lipgloss.JoinVertical(
				lipgloss.Left,
				theme.PanelHeader.Render("Template Repositories"),
				"",
				"Repository management is available in the Repositories section.",
				"",
				"From there, you can:",
				"• Add new template repositories",
				"• Manage existing repositories",
				"• Set repository priorities",
				"• Sync templates from repositories",
			),
		)
}