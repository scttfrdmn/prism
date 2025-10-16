package models

import (
	"context"
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/scttfrdmn/cloudworkstation/internal/tui/components"
	"github.com/scttfrdmn/cloudworkstation/internal/tui/styles"
	"github.com/scttfrdmn/cloudworkstation/pkg/types"
)

// BrowserTemplateItem represents a template in the browser list
type BrowserTemplateItem struct {
	name        string
	description string
	costX86     float64
	costARM     float64
	ports       []int
}

// FilterValue returns the value to filter on in the list
func (t BrowserTemplateItem) FilterValue() string { return t.name }

// Title returns the name of the template
func (t BrowserTemplateItem) Title() string { return t.name }

// Description returns a short description of the template
func (t BrowserTemplateItem) Description() string { return t.description }

// TemplatesModel represents the templates view
type TemplatesModel struct {
	apiClient    apiClient
	templateList list.Model
	detailView   viewport.Model
	statusBar    components.StatusBar
	spinner      components.Spinner
	width        int
	height       int
	loading      bool
	error        string
	templates    map[string]types.Template
	selected     string
}

// NewTemplatesModel creates a new templates model
func NewTemplatesModel(apiClient apiClient) TemplatesModel {
	theme := styles.CurrentTheme

	// Set up template list
	templateList := list.New([]list.Item{}, list.NewDefaultDelegate(), 0, 0)
	templateList.Title = "Available Templates"
	templateList.Styles.Title = theme.Title
	templateList.Styles.PaginationStyle = theme.Pagination
	templateList.Styles.HelpStyle = theme.Help

	// Set up detail view for template information
	detailView := viewport.New(0, 0)
	detailView.Style = theme.Panel

	// Create status bar and spinner
	statusBar := components.NewStatusBar("", "")
	spinner := components.NewSpinner("Loading templates...")

	return TemplatesModel{
		apiClient:    apiClient,
		templateList: templateList,
		detailView:   detailView,
		statusBar:    statusBar,
		spinner:      spinner,
		width:        80,
		height:       24,
		loading:      true,
		templates:    make(map[string]types.Template),
	}
}

// Init initializes the model
func (m TemplatesModel) Init() tea.Cmd {
	return tea.Batch(
		m.spinner.InitialCmd(),
		m.fetchTemplates,
	)
}

// fetchTemplates retrieves template data from the API
func (m TemplatesModel) fetchTemplates() tea.Msg {
	resp, err := m.apiClient.ListTemplates(context.Background())
	if err != nil {
		return fmt.Errorf("failed to list templates: %w", err)
	}
	return resp.Templates
}

// Update handles messages and updates the model
func (m TemplatesModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.statusBar.SetWidth(msg.Width)

		// Update list and detail view dimensions
		leftWidth := m.width / 3
		rightWidth := m.width - leftWidth - 2 // Account for separator
		contentHeight := m.height - 4         // Account for title and status

		m.templateList.SetSize(leftWidth, contentHeight)
		m.detailView.Width = rightWidth
		m.detailView.Height = contentHeight

		return m, nil

	case tea.KeyMsg:
		// Handle key presses
		switch msg.String() {
		case "r":
			m.loading = true
			m.error = ""
			return m, m.fetchTemplates

		case "q", "esc":
			return m, tea.Quit
		}

		// Only process list inputs when not loading
		if !m.loading {
			var cmd tea.Cmd
			m.templateList, cmd = m.templateList.Update(msg)
			cmds = append(cmds, cmd)

			// Handle template selection changes
			if i, ok := m.templateList.SelectedItem().(BrowserTemplateItem); ok {
				if i.name != m.selected {
					m.selected = i.name
					m.updateDetailView()
				}
			}

			// Update detail view on scroll
			m.detailView, cmd = m.detailView.Update(msg)
			cmds = append(cmds, cmd)
		}

	case RefreshMsg:
		m.loading = true
		m.error = ""
		return m, m.fetchTemplates

	case error:
		m.loading = false
		m.error = msg.Error()
		m.statusBar.SetStatus(fmt.Sprintf("Error: %s", m.error), components.StatusError)

	case map[string]types.Template:
		m.loading = false
		m.templates = msg

		// Update template list items
		var items []list.Item
		for name, template := range m.templates {
			items = append(items, BrowserTemplateItem{
				name:        name,
				description: template.Description,
				costX86:     template.EstimatedCostPerHour["x86_64"],
				costARM:     template.EstimatedCostPerHour["arm64"],
				ports:       template.Ports,
			})
		}

		m.templateList.SetItems(items)
		m.statusBar.SetStatus("Templates loaded", components.StatusSuccess)

		// Select first item
		if len(items) > 0 {
			m.selected = items[0].(BrowserTemplateItem).name
			m.updateDetailView()
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

// updateDetailView updates the content of the detail view with the selected template
func (m *TemplatesModel) updateDetailView() {
	if template, ok := m.templates[m.selected]; ok {
		theme := styles.CurrentTheme

		// Format the template details
		var content strings.Builder

		content.WriteString(theme.SectionTitle.Render(m.selected) + "\n\n")
		content.WriteString(template.Description + "\n\n")

		content.WriteString(theme.SubTitle.Render("Cost Information:") + "\n")
		content.WriteString(fmt.Sprintf("x86_64: $%.4f/hour ($%.2f/day)\n",
			template.EstimatedCostPerHour["x86_64"],
			template.EstimatedCostPerHour["x86_64"]*24))
		content.WriteString(fmt.Sprintf("arm64:  $%.4f/hour ($%.2f/day)\n\n",
			template.EstimatedCostPerHour["arm64"],
			template.EstimatedCostPerHour["arm64"]*24))

		content.WriteString(theme.SubTitle.Render("Instance Types:") + "\n")
		content.WriteString(fmt.Sprintf("x86_64: %s\n", template.InstanceType["x86_64"]))
		content.WriteString(fmt.Sprintf("arm64:  %s\n\n", template.InstanceType["arm64"]))

		content.WriteString(theme.SubTitle.Render("Open Ports:") + "\n")
		portStrings := make([]string, len(template.Ports))
		for i, port := range template.Ports {
			portStrings[i] = fmt.Sprintf("%d", port)
		}
		content.WriteString(strings.Join(portStrings, ", ") + "\n\n")

		// Research User Integration (Phase 5A+)
		if template.ResearchUser != nil {
			content.WriteString(theme.SubTitle.Render("Research User Support:") + "\n")
			if template.ResearchUser.AutoCreate {
				content.WriteString("✅ Automatic research user creation during launch\n")
			}
			if template.ResearchUser.RequireEFS {
				content.WriteString("✅ Persistent EFS home directories\n")
			}
			if template.ResearchUser.InstallSSHKeys {
				content.WriteString("✅ Automatic SSH key generation and installation\n")
			}
			if len(template.ResearchUser.UserIntegration.SharedDirectories) > 0 {
				content.WriteString("✅ Multi-user collaboration support\n")
			}
			content.WriteString("\n")
		}

		content.WriteString(theme.SubTitle.Render("Launch Command:") + "\n")
		content.WriteString(fmt.Sprintf("cws launch %s instance-name\n", m.selected))
		content.WriteString("cws launch " + m.selected + " instance-name --size L\n")
		content.WriteString("cws launch " + m.selected + " instance-name --volume data-volume\n")

		// Add research user launch example if supported
		if template.ResearchUser != nil && template.ResearchUser.AutoCreate {
			content.WriteString("cws launch " + m.selected + " instance-name --research-user alice\n")
		}

		m.detailView.SetContent(content.String())
		m.detailView.GotoTop()
	}
}

// View renders the templates view
func (m TemplatesModel) View() string {
	theme := styles.CurrentTheme

	// Title section
	title := theme.Title.Render("CloudWorkstation Templates")

	// Content area
	var content string
	if m.loading {
		content = lipgloss.NewStyle().
			Width(m.width).
			Height(m.height-4). // Account for title and status bar
			Align(lipgloss.Center, lipgloss.Center).
			Render(m.spinner.View())
	} else if m.error != "" {
		content = lipgloss.NewStyle().
			Width(m.width).
			Height(m.height-4).
			Align(lipgloss.Center, lipgloss.Center).
			Render(theme.StatusError.Render("Error: " + m.error))
	} else {
		// Split view with templates list on left and details on right
		leftPane := m.templateList.View()
		rightPane := m.detailView.View()

		separator := lipgloss.NewStyle().
			Foreground(theme.MutedColor).
			Width(1).
			Height(m.height - 4).
			Render("│")

		content = lipgloss.JoinHorizontal(lipgloss.Top, leftPane, separator, rightPane)
	}

	// Help text
	help := theme.Help.Render("r: refresh • q: quit • ↑/↓: navigate")

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
