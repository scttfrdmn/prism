package widgets

import (
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"

	"github.com/scttfrdmn/cloudworkstation/pkg/types"
)

// InstanceCard represents a card displaying instance information
type InstanceCard struct {
	widget.Card
	Instance      types.Instance
	OnConnect     func(name string)
	OnStart       func(name string)
	OnStop        func(name string)
	OnDelete      func(name string)
	StatusDisplay *StatusIndicator
	CostDisplay   *CostBadge
	Responsive    bool
}

// NewInstanceCard creates a new instance card
func NewInstanceCard(instance types.Instance, responsive bool) *InstanceCard {
	card := &InstanceCard{
		Instance:   instance,
		Responsive: responsive,
	}

	// Create status indicator
	card.StatusDisplay = NewStatusIndicator(instance.State, true)

	// Create cost badge
	card.CostDisplay = NewCostBadge(instance.EstimatedDailyCost, false, false)

	// Set up the content
	card.updateContent()

	return card
}

// updateContent updates the card's content based on the current instance
func (c *InstanceCard) updateContent() {
	// Create instance details
	details := container.NewVBox(
		widget.NewLabelWithStyle(c.Instance.Name, fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		widget.NewLabel(fmt.Sprintf("Template: %s", c.Instance.Template)),
		container.NewHBox(
			widget.NewLabel("Cost:"),
			c.CostDisplay,
		),
	)

	// Add launch time if available
	if !c.Instance.LaunchTime.IsZero() {
		details.Add(widget.NewLabel(fmt.Sprintf("Launched: %s", c.Instance.LaunchTime.Format("Jan 2, 2006 15:04"))))
	}

	// Create status display
	status := container.NewVBox(
		container.NewHBox(
			c.StatusDisplay,
		),
	)

	// Create action buttons
	actions := container.NewVBox()

	// Add appropriate action buttons based on instance state
	switch c.Instance.State {
	case "running":
		connectBtn := widget.NewButton("Connect", func() {
			if c.OnConnect != nil {
				c.OnConnect(c.Instance.Name)
			}
		})
		stopBtn := widget.NewButton("Stop", func() {
			if c.OnStop != nil {
				c.OnStop(c.Instance.Name)
			}
		})
		stopBtn.Importance = widget.WarningImportance

		actions.Add(connectBtn)
		actions.Add(stopBtn)
	case "stopped":
		startBtn := widget.NewButton("Start", func() {
			if c.OnStart != nil {
				c.OnStart(c.Instance.Name)
			}
		})
		actions.Add(startBtn)
	}

	// Always add delete button
	deleteBtn := widget.NewButton("Delete", func() {
		if c.OnDelete != nil {
			c.OnDelete(c.Instance.Name)
		}
	})
	deleteBtn.Importance = widget.DangerImportance
	actions.Add(deleteBtn)

	// Lay out the card based on responsive setting
	var content fyne.CanvasObject
	if c.Responsive {
		// Responsive layout that will adjust based on width
		content = container.NewHBox(
			details,
			layout.NewSpacer(),
			status,
			layout.NewSpacer(),
			actions,
		)
	} else {
		// Compact layout (vertical for narrow screens)
		content = container.NewVBox(
			container.NewHBox(
				c.StatusDisplay,
				widget.NewLabelWithStyle(c.Instance.Name, fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
				layout.NewSpacer(),
				c.CostDisplay,
			),
			widget.NewLabel(fmt.Sprintf("Template: %s", c.Instance.Template)),
			container.NewHBox(
				layout.NewSpacer(),
				actions,
			),
		)
	}

	// Set the card content
	c.SetContent(content)
}

// SetInstance updates the instance displayed in the card
func (c *InstanceCard) SetInstance(instance types.Instance) {
	c.Instance = instance
	c.StatusDisplay.Update(instance.State)
	c.CostDisplay.Update(instance.EstimatedDailyCost)
	c.updateContent()
}

// Uses the responsive layout from responsive_layout.go
