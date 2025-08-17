package widgets

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	cwstheme "github.com/scttfrdmn/cloudworkstation/cmd/cws-gui/theme"
)

// StatusIndicator displays the status of an instance with a colored indicator
type StatusIndicator struct {
	widget.BaseWidget
	State    string
	showText bool
	color    color.Color
}

// NewStatusIndicator creates a new status indicator
func NewStatusIndicator(state string, showText bool) *StatusIndicator {
	i := &StatusIndicator{
		State:    state,
		showText: showText,
	}
	i.ExtendBaseWidget(i)
	return i
}

// CreateRenderer creates a renderer for the status indicator
func (i *StatusIndicator) CreateRenderer() fyne.WidgetRenderer {
	// Get the color based on state
	i.color = cwstheme.GetStateColor(i.State, fyne.CurrentApp().Settings().ThemeVariant())

	// Create the circle indicator
	circle := canvas.NewCircle(i.color)

	// Create the label if needed
	var objects []fyne.CanvasObject
	var label *widget.Label

	if i.showText {
		label = widget.NewLabel(i.State)
		objects = []fyne.CanvasObject{circle, label}
	} else {
		objects = []fyne.CanvasObject{circle}
	}

	return &statusIndicatorRenderer{
		indicator: i,
		circle:    circle,
		label:     label,
		objects:   objects,
	}
}

// statusIndicatorRenderer renders the status indicator
type statusIndicatorRenderer struct {
	indicator *StatusIndicator
	circle    *canvas.Circle
	label     *widget.Label
	objects   []fyne.CanvasObject
}

// Layout positions the components of the status indicator
func (r *statusIndicatorRenderer) Layout(size fyne.Size) {
	// Size the circle
	var circleSize float32
	if r.label == nil {
		circleSize = fyne.Min(size.Width, size.Height)
	} else {
		circleSize = fyne.Min(size.Height, theme.Padding()*2)
	}

	r.circle.Resize(fyne.NewSize(circleSize, circleSize))
	r.circle.Move(fyne.NewPos(0, (size.Height-circleSize)/2))

	// Position the label if present
	if r.label != nil {
		labelPos := fyne.NewPos(circleSize+theme.Padding(), 0)
		labelSize := fyne.NewSize(size.Width-circleSize-theme.Padding(), size.Height)
		r.label.Resize(labelSize)
		r.label.Move(labelPos)
	}
}

// MinSize calculates the minimum size of the status indicator
func (r *statusIndicatorRenderer) MinSize() fyne.Size {
	minHeight := theme.Padding() * 2
	minWidth := minHeight

	// Add space for label if needed
	if r.label != nil {
		labelMin := r.label.MinSize()
		minWidth += labelMin.Width + theme.Padding()
		if labelMin.Height > minHeight {
			minHeight = labelMin.Height
		}
	}

	return fyne.NewSize(minWidth, minHeight)
}

// Refresh updates the status indicator's display
func (r *statusIndicatorRenderer) Refresh() {
	// Update color based on current state
	r.circle.FillColor = cwstheme.GetStateColor(r.indicator.State, fyne.CurrentApp().Settings().ThemeVariant())

	// Update label if present
	if r.label != nil {
		r.label.SetText(r.indicator.State)
		r.label.Refresh()
	}

	r.circle.Refresh()
}

// Objects returns the objects that make up the status indicator
func (r *statusIndicatorRenderer) Objects() []fyne.CanvasObject {
	return r.objects
}

// Destroy cleans up resources when the status indicator is no longer needed
func (r *statusIndicatorRenderer) Destroy() {
	// No resources to clean up
}

// Update allows changing the status indicator's state
func (i *StatusIndicator) Update(state string) {
	i.State = state
	i.Refresh()
}
