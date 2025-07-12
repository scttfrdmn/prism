package components

import (
	"github.com/charmbracelet/lipgloss"
	"github.com/scttfrdmn/cloudworkstation/internal/tui/styles"
)

// TabItem represents a single tab in the tab bar
type TabItem struct {
	ID    string
	Title string
}

// TabBar represents a tab bar component
type TabBar struct {
	tabs        []TabItem
	activeTab   string
	width       int
	showBorder  bool
	tabWidth    int // if 0, tabs are sized automatically
	height      int // typically 1 or 3 (for borders)
}

// NewTabBar creates a new tab bar with the specified tabs
func NewTabBar(tabs []TabItem, activeTab string) TabBar {
	return TabBar{
		tabs:      tabs,
		activeTab: activeTab,
		width:     80,
		height:    3,
		showBorder: true,
	}
}

// SetActiveTab changes the active tab
func (t *TabBar) SetActiveTab(tabID string) {
	t.activeTab = tabID
}

// GetActiveTab returns the ID of the active tab
func (t *TabBar) GetActiveTab() string {
	return t.activeTab
}

// SetWidth sets the width of the tab bar
func (t *TabBar) SetWidth(width int) {
	t.width = width
}

// SetTabWidth sets a fixed width for each tab
func (t *TabBar) SetTabWidth(width int) {
	t.tabWidth = width
}

// SetShowBorder sets whether to show borders
func (t *TabBar) SetShowBorder(show bool) {
	t.showBorder = show
	if show {
		t.height = 3
	} else {
		t.height = 1
	}
}

// ClickTab handles a tab click at the given x position
func (t *TabBar) ClickTab(x int) (clicked bool) {
	tabWidth := t.calculateTabWidth()
	
	// Calculate which tab was clicked
	pos := 0
	for _, tab := range t.tabs {
		if x >= pos && x < pos+tabWidth {
			if t.activeTab != tab.ID {
				t.activeTab = tab.ID
				return true
			}
			return false
		}
		pos += tabWidth
	}
	return false
}

// calculateTabWidth returns the width each tab should have
func (t *TabBar) calculateTabWidth() int {
	if t.tabWidth > 0 {
		return t.tabWidth
	}
	
	// Divide available space evenly among tabs
	tabCount := len(t.tabs)
	if tabCount == 0 {
		return 10 // default
	}
	return t.width / tabCount
}

// View renders the tab bar
func (t *TabBar) View() string {
	theme := styles.CurrentTheme
	
	// Calculate tab width
	tabWidth := t.calculateTabWidth()
	
	// Create tab styles
	activeTabStyle := lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(theme.AccentColor).
		BorderBottom(false).
		Padding(0, 1).
		Background(theme.PrimaryColor).
		Foreground(theme.TextColor).
		Bold(true).
		Width(tabWidth - 2) // account for borders
		
	inactiveTabStyle := lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(theme.BorderColor).
		BorderBottom(false).
		Padding(0, 1).
		Foreground(theme.MutedColor).
		Width(tabWidth - 2) // account for borders
	
	// Set to simple style if not showing borders
	if !t.showBorder {
		activeTabStyle = lipgloss.NewStyle().
			Background(theme.PrimaryColor).
			Foreground(theme.TextColor).
			Bold(true).
			Padding(0, 1).
			Width(tabWidth - 2)
			
		inactiveTabStyle = lipgloss.NewStyle().
			Foreground(theme.MutedColor).
			Padding(0, 1).
			Width(tabWidth - 2)
	}
	
	// Build tab string
	var renderedTabs []string
	
	for _, tab := range t.tabs {
		if tab.ID == t.activeTab {
			renderedTabs = append(renderedTabs, activeTabStyle.Render(tab.Title))
		} else {
			renderedTabs = append(renderedTabs, inactiveTabStyle.Render(tab.Title))
		}
	}
	
	// Join tabs together
	tabs := lipgloss.JoinHorizontal(lipgloss.Top, renderedTabs...)
	
	// If showing borders, create a bottom line that spans the entire row
	if t.showBorder {
		bottom := lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderTop(false).
			BorderLeft(true).
			BorderRight(true).
			BorderBottom(true).
			BorderForeground(theme.AccentColor).
			Width(t.width - 2).
			Render("")
			
		return lipgloss.JoinVertical(lipgloss.Left, tabs, bottom)
	}
	
	return tabs
}