package models

import (
	"context"
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/scttfrdmn/cloudworkstation/internal/tui/api"
	"github.com/scttfrdmn/cloudworkstation/internal/tui/components"
	"github.com/scttfrdmn/cloudworkstation/internal/tui/styles"
)

// MarketplaceModel represents the marketplace view
type MarketplaceModel struct {
	apiClient         apiClient
	templatesTable    components.Table
	statusBar         components.StatusBar
	spinner           components.Spinner
	searchInput       textinput.Model
	width             int
	height            int
	loading           bool
	error             string
	templates         []api.MarketplaceTemplateResponse
	selectedTemplate  int
	selectedTab       int // 0=browse, 1=search, 2=categories, 3=registries
	showInstallDialog bool
	showDetailView    bool
	searchQuery       string
	selectedCategory  string
	selectedRegistry  string
	categories        []api.CategoryResponse
	registries        []api.RegistryResponse
}

// MarketplaceDataMsg represents marketplace data retrieved from the API
type MarketplaceDataMsg struct {
	Templates  []api.MarketplaceTemplateResponse
	Categories []api.CategoryResponse
	Registries []api.RegistryResponse
	Error      error
}

// MarketplaceSearchMsg represents search results
type MarketplaceSearchMsg struct {
	Templates []api.MarketplaceTemplateResponse
	Error     error
}

// MarketplaceInstallMsg represents template installation result
type MarketplaceInstallMsg struct {
	Success bool
	Message string
	Error   error
}

// NewMarketplaceModel creates a new marketplace model
func NewMarketplaceModel(apiClient apiClient) MarketplaceModel {
	// Create templates table
	columns := []table.Column{
		{Title: "NAME", Width: 25},
		{Title: "PUBLISHER", Width: 20},
		{Title: "CATEGORY", Width: 15},
		{Title: "RATING", Width: 8},
		{Title: "DOWNLOADS", Width: 10},
		{Title: "VERIFIED", Width: 8},
	}

	templatesTable := components.NewTable(columns, []table.Row{}, 80, 10, true)

	// Create status bar and spinner
	statusBar := components.NewStatusBar("CloudWorkstation Marketplace", "")
	spinner := components.NewSpinner("Loading marketplace...")

	// Create search input
	searchInput := textinput.New()
	searchInput.Placeholder = "Search templates..."
	searchInput.CharLimit = 100
	searchInput.Width = 50

	return MarketplaceModel{
		apiClient:      apiClient,
		templatesTable: templatesTable,
		statusBar:      statusBar,
		spinner:        spinner,
		searchInput:    searchInput,
		width:          80,
		height:         24,
		loading:        true,
		selectedTab:    0,
	}
}

// Init initializes the model
func (m MarketplaceModel) Init() tea.Cmd {
	return tea.Batch(
		m.spinner.InitialCmd(),
		m.fetchMarketplaceData,
	)
}

// fetchMarketplaceData retrieves marketplace data from the API
func (m MarketplaceModel) fetchMarketplaceData() tea.Msg {
	// Fetch marketplace templates
	templates, err := m.apiClient.ListMarketplaceTemplates(context.Background(), nil)
	if err != nil {
		return MarketplaceDataMsg{Error: fmt.Errorf("failed to list marketplace templates: %w", err)}
	}

	// Fetch categories
	categories, err := m.apiClient.ListMarketplaceCategories(context.Background())
	if err != nil {
		return MarketplaceDataMsg{Error: fmt.Errorf("failed to list categories: %w", err)}
	}

	// Fetch registries
	registries, err := m.apiClient.ListMarketplaceRegistries(context.Background())
	if err != nil {
		return MarketplaceDataMsg{Error: fmt.Errorf("failed to list registries: %w", err)}
	}

	return MarketplaceDataMsg{
		Templates:  templates.Templates,
		Categories: categories.Categories,
		Registries: registries.Registries,
		Error:      nil,
	}
}

// Update handles messages and updates the model
func (m MarketplaceModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		return m.handleWindowSize(msg)
	case MarketplaceDataMsg:
		return m.handleMarketplaceData(msg)
	case MarketplaceSearchMsg:
		return m.handleSearchData(msg)
	case MarketplaceInstallMsg:
		return m.handleInstallResult(msg)
	case tea.KeyMsg:
		return m.handleKeyPress(msg)
	case spinner.TickMsg:
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	}

	return m, nil
}

// handleWindowSize handles window resize events
func (m MarketplaceModel) handleWindowSize(msg tea.WindowSizeMsg) (tea.Model, tea.Cmd) {
	m.width = msg.Width
	m.height = msg.Height
	m.templatesTable.SetSize(msg.Width-4, msg.Height-15)
	return m, nil
}

// handleMarketplaceData handles marketplace data response from API
func (m MarketplaceModel) handleMarketplaceData(msg MarketplaceDataMsg) (tea.Model, tea.Cmd) {
	if msg.Error != nil {
		m.error = msg.Error.Error()
		m.loading = false
		return m, nil
	}

	m.templates = msg.Templates
	m.categories = msg.Categories
	m.registries = msg.Registries
	m.loading = false
	m.error = ""

	// Update table with template data
	m.updateTemplatesTable()
	return m, nil
}

// handleSearchData handles search results response
func (m MarketplaceModel) handleSearchData(msg MarketplaceSearchMsg) (tea.Model, tea.Cmd) {
	if msg.Error != nil {
		m.error = msg.Error.Error()
		return m, nil
	}

	m.templates = msg.Templates
	m.updateTemplatesTable()
	return m, nil
}

// handleInstallResult handles template installation result
func (m MarketplaceModel) handleInstallResult(msg MarketplaceInstallMsg) (tea.Model, tea.Cmd) {
	m.showInstallDialog = false
	if msg.Error != nil {
		m.error = msg.Error.Error()
		return m, nil
	}
	// Refresh template list after installation
	return m, m.fetchMarketplaceData
}

// handleKeyPress handles keyboard input
func (m MarketplaceModel) handleKeyPress(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.loading {
		return m, nil
	}

	// Handle search input when in search tab
	if m.selectedTab == 1 && m.searchInput.Focused() {
		return m.handleSearchInput(msg)
	}

	switch msg.String() {
	case "ctrl+c", "q":
		return m, tea.Quit
	case "r", "f5":
		return m.handleRefresh()
	case "tab":
		return m.handleTabSwitch()
	case "/":
		return m.handleSearchFocus()
	case "i":
		return m.handleInstall()
	case "v":
		return m.handleViewDetails()
	case "enter":
		return m.handleEnterKey()
	case "esc":
		return m.handleEscKey()
	case "up", "k":
		return m.handleUpKey()
	case "down", "j":
		return m.handleDownKey()
	}

	return m, nil
}

// handleSearchInput handles text input in search field
func (m MarketplaceModel) handleSearchInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	switch msg.String() {
	case "enter":
		m.searchQuery = m.searchInput.Value()
		m.searchInput.Blur()
		return m, m.searchTemplates
	case "esc":
		m.searchInput.Blur()
		return m, nil
	default:
		m.searchInput, cmd = m.searchInput.Update(msg)
		return m, cmd
	}
}

// handleRefresh refreshes marketplace data
func (m MarketplaceModel) handleRefresh() (tea.Model, tea.Cmd) {
	m.loading = true
	return m, m.fetchMarketplaceData
}

// handleTabSwitch cycles through tabs
func (m MarketplaceModel) handleTabSwitch() (tea.Model, tea.Cmd) {
	m.selectedTab = (m.selectedTab + 1) % 4
	return m, nil
}

// handleSearchFocus focuses search input in search tab
func (m MarketplaceModel) handleSearchFocus() (tea.Model, tea.Cmd) {
	if m.selectedTab == 1 {
		m.searchInput.Focus()
	}
	return m, nil
}

// handleInstall shows install dialog for selected template
func (m MarketplaceModel) handleInstall() (tea.Model, tea.Cmd) {
	if m.selectedTemplate < len(m.templates) && !m.showDetailView {
		m.showInstallDialog = true
	}
	return m, nil
}

// handleViewDetails toggles detail view for selected template
func (m MarketplaceModel) handleViewDetails() (tea.Model, tea.Cmd) {
	if m.selectedTemplate < len(m.templates) {
		m.showDetailView = !m.showDetailView
	}
	return m, nil
}

// handleEnterKey handles Enter key press (dialog confirmation)
func (m MarketplaceModel) handleEnterKey() (tea.Model, tea.Cmd) {
	if m.showInstallDialog {
		template := m.templates[m.selectedTemplate]
		return m, m.installTemplate(template.Name)
	}
	return m, nil
}

// handleEscKey handles Escape key press (close dialogs)
func (m MarketplaceModel) handleEscKey() (tea.Model, tea.Cmd) {
	if m.showInstallDialog {
		m.showInstallDialog = false
		return m, nil
	}
	if m.showDetailView {
		m.showDetailView = false
		return m, nil
	}
	return m, nil
}

// handleUpKey handles up arrow navigation
func (m MarketplaceModel) handleUpKey() (tea.Model, tea.Cmd) {
	if m.selectedTemplate > 0 {
		m.selectedTemplate--
	}
	return m, nil
}

// handleDownKey handles down arrow navigation
func (m MarketplaceModel) handleDownKey() (tea.Model, tea.Cmd) {
	if m.selectedTemplate < len(m.templates)-1 {
		m.selectedTemplate++
	}
	return m, nil
}

// View renders the model
func (m MarketplaceModel) View() string {
	if m.loading {
		return m.spinner.View()
	}

	var b strings.Builder
	theme := styles.CurrentTheme

	// Header
	header := theme.Title.Render("📦 Template Marketplace")
	b.WriteString(header)
	b.WriteString("\n\n")

	// Tabs
	b.WriteString(m.renderTabs())
	b.WriteString("\n\n")

	// Content based on selected tab
	switch m.selectedTab {
	case 0:
		b.WriteString(m.renderBrowse())
	case 1:
		b.WriteString(m.renderSearch())
	case 2:
		b.WriteString(m.renderCategories())
	case 3:
		b.WriteString(m.renderRegistries())
	}

	// Show install dialog if active
	if m.showInstallDialog {
		dialog := m.renderInstallDialog()
		b.WriteString("\n\n")
		b.WriteString(dialog)
	}

	// Show detail view if active
	if m.showDetailView {
		detail := m.renderDetailView()
		b.WriteString("\n\n")
		b.WriteString(detail)
	}

	// Error display
	if m.error != "" {
		b.WriteString("\n\n")
		errorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("9")).Bold(true)
		b.WriteString(errorStyle.Render("Error: " + m.error))
	}

	// Help text
	b.WriteString("\n\n")
	helpText := m.renderHelp()
	b.WriteString(helpText)

	return b.String()
}

// renderTabs displays the tab navigation
func (m MarketplaceModel) renderTabs() string {
	theme := styles.CurrentTheme
	tabs := []string{"Browse", "Search", "Categories", "Registries"}

	var renderedTabs []string
	for i, tab := range tabs {
		if i == m.selectedTab {
			renderedTabs = append(renderedTabs, theme.Tab.Active.Render(" "+tab+" "))
		} else {
			renderedTabs = append(renderedTabs, theme.Tab.Inactive.Render(" "+tab+" "))
		}
	}

	return strings.Join(renderedTabs, " ")
}

// renderBrowse displays the browse templates view
func (m MarketplaceModel) renderBrowse() string {
	var b strings.Builder
	theme := styles.CurrentTheme

	b.WriteString(theme.SectionTitle.Render("Browse Templates"))
	b.WriteString("\n\n")

	if len(m.templates) == 0 {
		b.WriteString("No templates available in marketplace\n")
		return b.String()
	}

	// Templates table
	b.WriteString(m.templatesTable.View())

	return b.String()
}

// renderSearch displays the search view
func (m MarketplaceModel) renderSearch() string {
	var b strings.Builder
	theme := styles.CurrentTheme

	b.WriteString(theme.SectionTitle.Render("Search Templates"))
	b.WriteString("\n\n")

	// Search input
	b.WriteString("Search: ")
	b.WriteString(m.searchInput.View())
	b.WriteString("\n\n")

	if m.searchQuery != "" {
		b.WriteString(fmt.Sprintf("Results for: %s\n\n", m.searchQuery))
	}

	if len(m.templates) == 0 {
		b.WriteString("No templates found\n")
		return b.String()
	}

	// Templates table
	b.WriteString(m.templatesTable.View())

	return b.String()
}

// renderCategories displays the categories view
func (m MarketplaceModel) renderCategories() string {
	var b strings.Builder
	theme := styles.CurrentTheme

	b.WriteString(theme.SectionTitle.Render("Template Categories"))
	b.WriteString("\n\n")

	if len(m.categories) == 0 {
		b.WriteString("No categories available\n")
		return b.String()
	}

	for _, category := range m.categories {
		categoryStyle := theme.SubTitle
		if category.Name == m.selectedCategory {
			categoryStyle = theme.SubTitle.Bold(true)
		}

		b.WriteString(categoryStyle.Render(fmt.Sprintf("📁 %s", category.Name)))
		b.WriteString(fmt.Sprintf(" (%d templates)\n", category.TemplateCount))

		if category.Description != "" {
			b.WriteString(fmt.Sprintf("   %s\n", category.Description))
		}
		b.WriteString("\n")
	}

	return b.String()
}

// renderRegistries displays the registries view
func (m MarketplaceModel) renderRegistries() string {
	var b strings.Builder
	theme := styles.CurrentTheme

	b.WriteString(theme.SectionTitle.Render("Template Registries"))
	b.WriteString("\n\n")

	if len(m.registries) == 0 {
		b.WriteString("No registries configured\n")
		return b.String()
	}

	for _, registry := range m.registries {
		registryStyle := theme.SubTitle
		if registry.Name == m.selectedRegistry {
			registryStyle = theme.SubTitle.Bold(true)
		}

		b.WriteString(registryStyle.Render(fmt.Sprintf("🏛️  %s", registry.Name)))
		b.WriteString(fmt.Sprintf(" (%s)\n", registry.Type))

		if registry.URL != "" {
			b.WriteString(fmt.Sprintf("   URL: %s\n", registry.URL))
		}
		b.WriteString(fmt.Sprintf("   Templates: %d | Status: %s\n", registry.TemplateCount, registry.Status))
		b.WriteString("\n")
	}

	return b.String()
}

// renderInstallDialog displays the template installation dialog
func (m MarketplaceModel) renderInstallDialog() string {
	theme := styles.CurrentTheme

	dialogStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("62")).
		Padding(1, 2).
		Width(60)

	var content strings.Builder
	if m.selectedTemplate < len(m.templates) {
		template := m.templates[m.selectedTemplate]
		content.WriteString(theme.SubTitle.Render("Install Template") + "\n\n")
		content.WriteString(fmt.Sprintf("Name: %s\n", template.Name))
		content.WriteString(fmt.Sprintf("Publisher: %s\n", template.Publisher))
		content.WriteString(fmt.Sprintf("Category: %s\n\n", template.Category))
		content.WriteString("Press Enter to confirm, Esc to cancel\n")
	}

	return dialogStyle.Render(content.String())
}

// renderDetailView displays detailed template information
func (m MarketplaceModel) renderDetailView() string {
	theme := styles.CurrentTheme

	dialogStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("62")).
		Padding(1, 2).
		Width(70)

	var content strings.Builder
	if m.selectedTemplate < len(m.templates) {
		template := m.templates[m.selectedTemplate]
		content.WriteString(theme.SubTitle.Render("Template Details") + "\n\n")
		content.WriteString(fmt.Sprintf("Name: %s\n", template.Name))
		content.WriteString(fmt.Sprintf("Publisher: %s\n", template.Publisher))
		content.WriteString(fmt.Sprintf("Category: %s\n", template.Category))
		content.WriteString(fmt.Sprintf("Rating: %.1f (%d ratings)\n", template.Rating, template.RatingCount))
		content.WriteString(fmt.Sprintf("Downloads: %d\n", template.Downloads))
		content.WriteString(fmt.Sprintf("Verified: %v\n\n", template.Verified))

		if template.Description != "" {
			content.WriteString(fmt.Sprintf("Description:\n%s\n\n", template.Description))
		}

		if len(template.Keywords) > 0 {
			content.WriteString(fmt.Sprintf("Keywords: %s\n", strings.Join(template.Keywords, ", ")))
		}

		content.WriteString("\nPress Esc to close\n")
	}

	return dialogStyle.Render(content.String())
}

// renderHelp displays help text
func (m MarketplaceModel) renderHelp() string {
	theme := styles.CurrentTheme

	var helps []string
	if m.showInstallDialog {
		helps = []string{"enter: confirm", "esc: cancel"}
	} else if m.showDetailView {
		helps = []string{"esc: close"}
	} else if m.selectedTab == 1 && m.searchInput.Focused() {
		helps = []string{"enter: search", "esc: cancel"}
	} else {
		helps = []string{
			"↑/↓: select",
			"tab: switch tabs",
			"i: install",
			"v: view details",
			"/: search",
			"r: refresh",
			"q: quit",
		}
	}

	return theme.Help.Render(strings.Join(helps, " • "))
}

// searchTemplates searches for templates
func (m MarketplaceModel) searchTemplates() tea.Msg {
	filter := &api.MarketplaceFilter{
		Query: m.searchQuery,
	}

	templates, err := m.apiClient.ListMarketplaceTemplates(context.Background(), filter)
	if err != nil {
		return MarketplaceSearchMsg{Error: fmt.Errorf("failed to search templates: %w", err)}
	}

	return MarketplaceSearchMsg{Templates: templates.Templates, Error: nil}
}

// installTemplate installs a marketplace template
func (m MarketplaceModel) installTemplate(templateName string) tea.Cmd {
	return func() tea.Msg {
		err := m.apiClient.InstallMarketplaceTemplate(context.Background(), templateName)
		if err != nil {
			return MarketplaceInstallMsg{Error: fmt.Errorf("failed to install template: %w", err)}
		}

		return MarketplaceInstallMsg{
			Success: true,
			Message: fmt.Sprintf("Successfully installed template: %s", templateName),
			Error:   nil,
		}
	}
}

// updateTemplatesTable updates the templates table with current data
func (m *MarketplaceModel) updateTemplatesTable() {
	rows := []table.Row{}
	for i, template := range m.templates {
		// Selection indicator
		name := template.Name
		if i == m.selectedTemplate {
			name = "> " + name
		}

		// Verified status
		verified := "No"
		if template.Verified {
			verified = "✓ Yes"
		}

		row := table.Row{
			name,
			template.Publisher,
			template.Category,
			fmt.Sprintf("%.1f", template.Rating),
			fmt.Sprintf("%d", template.Downloads),
			verified,
		}
		rows = append(rows, row)
	}

	m.templatesTable.SetRows(rows)
}
