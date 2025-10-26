package components

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/scttfrdmn/prism/internal/tui/styles"
)

// TabChangeMsg is sent when the active tab changes
type TabChangeMsg struct {
	Index int
}

// Tab represents a tab in the tab bar
type Tab struct {
	title string
	style lipgloss.Style
}

// TabBar is a simple tab bar component
type TabBar struct {
	tabs                []Tab
	activeTabIndex      int
	activeBorderColor   lipgloss.Color
	inactiveBorderColor lipgloss.Color
	width               int
}

// NewTabBar creates a new tab bar with the given tab titles
func NewTabBar(tabs []string, activeIndex int) TabBar {
	theme := styles.CurrentTheme

	// Create tab objects
	tabItems := make([]Tab, len(tabs))
	for i, title := range tabs {
		tabItems[i] = Tab{
			title: title,
			style: lipgloss.NewStyle().
				Border(lipgloss.NormalBorder()).
				BorderForeground(theme.BorderColor).
				Padding(0, 2),
		}
	}

	return TabBar{
		tabs:                tabItems,
		activeTabIndex:      activeIndex,
		activeBorderColor:   theme.PrimaryColor,
		inactiveBorderColor: theme.MutedColor,
		width:               80,
	}
}

// ActiveTab returns the index of the active tab
func (t TabBar) ActiveTab() int {
	return t.activeTabIndex
}

// Next activates the next tab
func (t *TabBar) Next() tea.Cmd {
	if t.activeTabIndex < len(t.tabs)-1 {
		t.activeTabIndex++
	} else {
		t.activeTabIndex = 0
	}
	return func() tea.Msg {
		return TabChangeMsg{Index: t.activeTabIndex}
	}
}

// Prev activates the previous tab
func (t *TabBar) Prev() tea.Cmd {
	if t.activeTabIndex > 0 {
		t.activeTabIndex--
	} else {
		t.activeTabIndex = len(t.tabs) - 1
	}
	return func() tea.Msg {
		return TabChangeMsg{Index: t.activeTabIndex}
	}
}

// SetWidth sets the width of the tab bar
func (t *TabBar) SetWidth(width int) {
	t.width = width
}

// View renders the tab bar
func (t TabBar) View() string {
	var renderedTabs []string

	// Calculate approximate width for each tab
	tabWidth := (t.width / len(t.tabs)) - 4 // Account for borders and spacing

	for i, tab := range t.tabs {
		// Set border color based on active status
		borderColor := t.inactiveBorderColor
		if i == t.activeTabIndex {
			borderColor = t.activeBorderColor
		}

		// Set tab style
		style := tab.style.
			BorderForeground(borderColor).
			Width(tabWidth)

		if i == t.activeTabIndex {
			style = style.Bold(true)
		}

		// Render tab
		renderedTabs = append(renderedTabs, style.Render(tab.title))
	}

	return lipgloss.JoinHorizontal(lipgloss.Top, renderedTabs...)
}
