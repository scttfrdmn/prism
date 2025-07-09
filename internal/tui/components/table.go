package components

import (
	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/scttfrdmn/cloudworkstation/internal/tui/styles"
)

// Table is a styled table component
type Table struct {
	table        table.Model
	focusable    bool
	focused      bool
	selectedItem string
}

// NewTable creates a new table component
func NewTable(columns []table.Column, rows []table.Row, 
	          width, height int, focusable bool) Table {
	t := table.New(
		table.WithColumns(columns),
		table.WithRows(rows),
		table.WithFocused(false),
		table.WithWidth(width),
		table.WithHeight(height),
	)

	// Apply styles from theme
	theme := styles.CurrentTheme
	t.SetStyles(table.Styles{
		Header: theme.TableHeader,
		Selected: theme.TableRow.Copy().
			Foreground(theme.TextColor).
			Background(theme.PrimaryColor),
		Cell: theme.TableRow,
	})

	return Table{
		table:     t,
		focusable: focusable,
		focused:   false,
	}
}

// SelectedRow returns the currently selected row
func (t *Table) SelectedRow() table.Row {
	if len(t.table.Rows()) == 0 {
		return table.Row{}
	}
	return t.table.SelectedRow()
}

// SetRows updates the table rows
func (t *Table) SetRows(rows []table.Row) {
	t.table.SetRows(rows)
}

// Focus puts the table in focus
func (t *Table) Focus() {
	if t.focusable {
		t.focused = true
		t.table.Focus()
	}
}

// Blur removes focus from the table
func (t *Table) Blur() {
	t.focused = false
	t.table.Blur()
}

// Focused returns whether the table is focused
func (t *Table) Focused() bool {
	return t.focused
}

// Update handles messages for the table
func (t *Table) Update(msg tea.Msg) (Table, tea.Cmd) {
	if !t.focused {
		return *t, nil
	}

	var cmd tea.Cmd
	t.table, cmd = t.table.Update(msg)
	
	// Save the selected item for later reference
	if len(t.table.Rows()) > 0 {
		selectedRow := t.table.SelectedRow()
		if len(selectedRow) > 0 {
			t.selectedItem = selectedRow[0] // Assuming first column is the ID/name
		}
	}
	
	return *t, cmd
}

// View renders the table
func (t *Table) View() string {
	return t.table.View()
}
