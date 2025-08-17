package widgets

import (
	"fyne.io/fyne/v2"
)

// ResponsiveLayout implements a layout that changes based on available width
type ResponsiveLayout struct {
	wideLayout      fyne.CanvasObject
	narrowLayout    fyne.CanvasObject
	breakpointWidth float32
}

// NewResponsiveLayout creates a new responsive layout
func NewResponsiveLayout(wideLayout, narrowLayout fyne.CanvasObject, breakpointWidth float32) *ResponsiveLayout {
	return &ResponsiveLayout{
		wideLayout:      wideLayout,
		narrowLayout:    narrowLayout,
		breakpointWidth: breakpointWidth,
	}
}

// Layout positions the responsive layout's content
func (l *ResponsiveLayout) Layout(objects []fyne.CanvasObject, size fyne.Size) {
	// Show/hide based on available width
	if size.Width >= l.breakpointWidth {
		l.wideLayout.Show()
		l.narrowLayout.Hide()
		l.wideLayout.Resize(size)
	} else {
		l.wideLayout.Hide()
		l.narrowLayout.Show()
		l.narrowLayout.Resize(size)
	}
}

// MinSize calculates the minimum size of the responsive layout
func (l *ResponsiveLayout) MinSize(objects []fyne.CanvasObject) fyne.Size {
	wideMin := l.wideLayout.MinSize()
	narrowMin := l.narrowLayout.MinSize()

	// Use the larger of the two minimum widths and heights
	return fyne.NewSize(
		fyne.Max(wideMin.Width, narrowMin.Width),
		fyne.Max(wideMin.Height, narrowMin.Height),
	)
}

// Objects returns the objects in the responsive layout
func (l *ResponsiveLayout) Objects() []fyne.CanvasObject {
	return []fyne.CanvasObject{l.wideLayout, l.narrowLayout}
}
