package widgets

import (
	"fmt"
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	cwstheme "github.com/scttfrdmn/cloudworkstation/cmd/cws-gui/theme"
)

// CostBadge displays the cost information in a badge format
type CostBadge struct {
	widget.BaseWidget
	Cost        float64 // Daily cost
	ShowMonthly bool    // Whether to also show monthly cost
	ShowLabel   bool    // Whether to show "Cost:" label
}

// NewCostBadge creates a new cost badge
func NewCostBadge(cost float64, showMonthly, showLabel bool) *CostBadge {
	badge := &CostBadge{
		Cost:        cost,
		ShowMonthly: showMonthly,
		ShowLabel:   showLabel,
	}
	badge.ExtendBaseWidget(badge)
	return badge
}

// CreateRenderer creates a renderer for the cost badge
func (b *CostBadge) CreateRenderer() fyne.WidgetRenderer {
	// Create background rectangle
	background := canvas.NewRectangle(theme.Color(theme.ColorNameBackground))
	background.CornerRadius = 4

	// Create cost text
	var costText string
	if b.ShowLabel {
		costText = fmt.Sprintf("Cost: $%.2f/day", b.Cost)
		if b.ShowMonthly {
			costText += fmt.Sprintf(" (~$%.2f/mo)", b.Cost*30)
		}
	} else {
		costText = fmt.Sprintf("$%.2f/day", b.Cost)
		if b.ShowMonthly {
			costText += fmt.Sprintf(" (~$%.2f/mo)", b.Cost*30)
		}
	}

	// Create the label
	costLabel := widget.NewLabel(costText)

	// Set text style based on cost
	if b.Cost > 10 {
		costLabel.TextStyle = fyne.TextStyle{Bold: true}
	}

	return &costBadgeRenderer{
		badge:      b,
		background: background,
		label:      costLabel,
		objects:    []fyne.CanvasObject{background, costLabel},
	}
}

// costBadgeRenderer renders the cost badge
type costBadgeRenderer struct {
	badge      *CostBadge
	background *canvas.Rectangle
	label      *widget.Label
	objects    []fyne.CanvasObject
}

// Layout positions the components of the cost badge
func (r *costBadgeRenderer) Layout(size fyne.Size) {
	// Position background
	r.background.Resize(size)

	// Center the label
	labelSize := r.label.MinSize()
	r.label.Resize(labelSize)
	r.label.Move(fyne.NewPos(
		(size.Width-labelSize.Width)/2,
		(size.Height-labelSize.Height)/2,
	))
}

// MinSize calculates the minimum size of the cost badge
func (r *costBadgeRenderer) MinSize() fyne.Size {
	padding := theme.Padding() * 1.5
	labelSize := r.label.MinSize()

	return fyne.NewSize(
		labelSize.Width+padding*2,
		labelSize.Height+padding,
	)
}

// Refresh updates the cost badge's display
func (r *costBadgeRenderer) Refresh() {
	// Update background color based on cost
	r.updateBackgroundColor()

	// Update cost text
	var costText string
	if r.badge.ShowLabel {
		costText = fmt.Sprintf("Cost: $%.2f/day", r.badge.Cost)
		if r.badge.ShowMonthly {
			costText += fmt.Sprintf(" (~$%.2f/mo)", r.badge.Cost*30)
		}
	} else {
		costText = fmt.Sprintf("$%.2f/day", r.badge.Cost)
		if r.badge.ShowMonthly {
			costText += fmt.Sprintf(" (~$%.2f/mo)", r.badge.Cost*30)
		}
	}
	r.label.SetText(costText)

	// Update text style based on cost
	if r.badge.Cost > 10 {
		r.label.TextStyle = fyne.TextStyle{Bold: true}
	} else {
		r.label.TextStyle = fyne.TextStyle{}
	}

	// Refresh components
	r.label.Refresh()
	r.background.Refresh()
}

// updateBackgroundColor updates the background color based on cost
func (r *costBadgeRenderer) updateBackgroundColor() {
	variant := fyne.CurrentApp().Settings().ThemeVariant()
	var bgColor color.Color

	if variant == theme.VariantDark {
		// Dark mode
		if r.badge.Cost == 0 {
			// No cost (stopped instance)
			bgColor = theme.Color(theme.ColorNameBackground)
		} else if r.badge.Cost < 5 {
			// Low cost - subtle green
			bgColor = blendColors(cwstheme.CloudWorkstationColors.RunningDark, theme.Color(theme.ColorNameBackground), 0.3)
		} else if r.badge.Cost < 10 {
			// Medium cost - subtle orange
			bgColor = blendColors(cwstheme.CloudWorkstationColors.PendingDark, theme.Color(theme.ColorNameBackground), 0.3)
		} else {
			// High cost - subtle red
			bgColor = blendColors(cwstheme.CloudWorkstationColors.TerminatedDark, theme.Color(theme.ColorNameBackground), 0.3)
		}
	} else {
		// Light mode
		if r.badge.Cost == 0 {
			// No cost (stopped instance)
			bgColor = theme.Color(theme.ColorNameBackground)
		} else if r.badge.Cost < 5 {
			// Low cost - subtle green
			bgColor = blendColors(cwstheme.CloudWorkstationColors.RunningLight, theme.Color(theme.ColorNameBackground), 0.2)
		} else if r.badge.Cost < 10 {
			// Medium cost - subtle orange
			bgColor = blendColors(cwstheme.CloudWorkstationColors.PendingLight, theme.Color(theme.ColorNameBackground), 0.2)
		} else {
			// High cost - subtle red
			bgColor = blendColors(cwstheme.CloudWorkstationColors.TerminatedLight, theme.Color(theme.ColorNameBackground), 0.2)
		}
	}

	r.background.FillColor = bgColor
}

// Objects returns the objects that make up the cost badge
func (r *costBadgeRenderer) Objects() []fyne.CanvasObject {
	return r.objects
}

// Destroy cleans up resources when the cost badge is no longer needed
func (r *costBadgeRenderer) Destroy() {
	// No resources to clean up
}

// Update allows changing the cost badge's cost
func (b *CostBadge) Update(cost float64) {
	b.Cost = cost
	b.Refresh()
}

// Helper function to blend two colors
func blendColors(c1, c2 color.Color, factor float64) color.Color {
	r1, g1, b1, a1 := c1.RGBA()
	r2, g2, b2, a2 := c2.RGBA()

	// Convert from 16-bit to 8-bit color components
	r1, g1, b1, a1 = r1>>8, g1>>8, b1>>8, a1>>8
	r2, g2, b2, a2 = r2>>8, g2>>8, b2>>8, a2>>8

	// Blend
	r := uint8(float64(r1)*factor + float64(r2)*(1-factor))
	g := uint8(float64(g1)*factor + float64(g2)*(1-factor))
	b := uint8(float64(b1)*factor + float64(b2)*(1-factor))
	a := uint8(float64(a1)*factor + float64(a2)*(1-factor))

	return color.NRGBA{R: r, G: g, B: b, A: a}
}
