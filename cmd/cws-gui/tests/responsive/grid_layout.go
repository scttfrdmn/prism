package responsive

import (
	"fyne.io/fyne/v2"
)

// GridLayout is a responsive grid layout that adjusts columns based on available width
type GridLayout struct {
	MinWidth   float32 // Minimum width before collapsing to single column
	MaxWidth   float32 // Maximum width to limit growth
	ColumnSize float32 // Target width of each column
}

// Layout implements fyne.Layout interface
func (g *GridLayout) Layout(objects []fyne.CanvasObject, size fyne.Size) {
	if len(objects) == 0 {
		return
	}

	// Calculate number of columns based on width
	columns := int(size.Width / g.ColumnSize)
	if columns < 1 {
		columns = 1
	}
	
	// Don't allow too many columns
	maxColumns := int(g.MaxWidth / g.ColumnSize)
	if columns > maxColumns && maxColumns > 0 {
		columns = maxColumns
	}
	
	// Calculate width of each cell
	cellWidth := size.Width / float32(columns)
	
	// Calculate required height based on number of rows
	rows := (len(objects) + columns - 1) / columns // Ceiling division
	cellHeight := cellWidth // Square cells by default, could be adjusted
	
	// Position each object in the grid
	for i, obj := range objects {
		if !obj.Visible() {
			continue
		}
		
		row := i / columns
		col := i % columns
		
		x := float32(col) * cellWidth
		y := float32(row) * cellHeight
		
		obj.Move(fyne.NewPos(x, y))
		obj.Resize(fyne.NewSize(cellWidth, cellHeight))
	}
}

// MinSize implements fyne.Layout interface
func (g *GridLayout) MinSize(objects []fyne.CanvasObject) fyne.Size {
	if len(objects) == 0 {
		return fyne.NewSize(0, 0)
	}
	
	// Default to single column
	columns := 1
	
	// Find minimum size needed for all objects
	minWidth := float32(0)
	minHeight := float32(0)
	
	for _, obj := range objects {
		if !obj.Visible() {
			continue
		}
		
		objMin := obj.MinSize()
		if objMin.Width > minWidth {
			minWidth = objMin.Width
		}
		minHeight += objMin.Height
	}
	
	// Ensure minimum width for grid
	if minWidth < g.MinWidth {
		minWidth = g.MinWidth
	}
	
	// Adjust height based on number of rows in minimum configuration
	rows := (len(objects) + columns - 1) / columns
	rowHeight := minHeight / float32(len(objects)) // Average row height
	totalHeight := rowHeight * float32(rows)
	
	return fyne.NewSize(minWidth, totalHeight)
}