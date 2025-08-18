package dashboard

import (
	"context"
	"fmt"
	"time"

	"fyne.io/fyne/v2"
	fynecontainer "fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"

	"github.com/scttfrdmn/cloudworkstation/pkg/api"
	"github.com/scttfrdmn/cloudworkstation/pkg/types"
)

// Dashboard implements the Section interface for the dashboard view
type Dashboard struct {
	apiClient  api.CloudWorkstationAPI
	instances  []types.Instance
	totalCost  float64
	lastUpdate time.Time

	// UI components
	costLabel      *widget.Label
	instancesLabel *widget.Label
	statusLabel    *widget.Label
	refreshButton  *widget.Button
}

// NewDashboard creates a new dashboard section
func NewDashboard(apiClient api.CloudWorkstationAPI) *Dashboard {
	d := &Dashboard{
		apiClient:      apiClient,
		costLabel:      widget.NewLabel("Total Cost: Loading..."),
		instancesLabel: widget.NewLabel("Instances: Loading..."),
		statusLabel:    widget.NewLabel("Status: Connecting..."),
	}

	d.refreshButton = widget.NewButton("Refresh", d.refresh)

	return d
}

// CreateView creates the dashboard's main view (Interface Segregation)
func (d *Dashboard) CreateView() fyne.CanvasObject {
	// Create summary cards
	costCard := d.createCostSummaryCard()
	instanceCard := d.createInstanceSummaryCard()
	statusCard := d.createStatusCard()

	// Layout dashboard components
	topRow := fynecontainer.NewGridWithColumns(3, costCard, instanceCard, statusCard)

	// Recent activity section
	activityCard := d.createRecentActivityCard()

	// Quick actions
	actionsCard := d.createQuickActionsCard()

	// Combine all sections
	dashboard := fynecontainer.NewVBox(
		widget.NewLabel("Dashboard"),
		topRow,
		activityCard,
		actionsCard,
		d.refreshButton,
	)

	return fynecontainer.NewScroll(dashboard)
}

// UpdateView refreshes the dashboard data (Single Responsibility)
func (d *Dashboard) UpdateView() error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Fetch instances
	resp, err := d.apiClient.ListInstances(ctx)
	if err != nil {
		d.statusLabel.SetText("Status: Error connecting to daemon")
		return fmt.Errorf("failed to fetch instances: %w", err)
	}

	d.instances = resp.Instances
	d.calculateTotalCost()
	d.lastUpdate = time.Now()

	// Update UI labels
	d.updateLabels()

	return nil
}

// GetTitle returns the section title
func (d *Dashboard) GetTitle() string {
	return "Dashboard"
}

// createCostSummaryCard creates the cost overview card
func (d *Dashboard) createCostSummaryCard() fyne.CanvasObject {
	card := fynecontainer.NewVBox(
		widget.NewLabel("💰 Total Cost"),
		d.costLabel,
		widget.NewLabel("(per hour)"),
	)

	return fynecontainer.NewBorder(nil, nil, nil, nil, card)
}

// createInstanceSummaryCard creates the instance overview card
func (d *Dashboard) createInstanceSummaryCard() fyne.CanvasObject {
	card := fynecontainer.NewVBox(
		widget.NewLabel("🖥️ Instances"),
		d.instancesLabel,
		widget.NewLabel("(active)"),
	)

	return fynecontainer.NewBorder(nil, nil, nil, nil, card)
}

// createStatusCard creates the connection status card
func (d *Dashboard) createStatusCard() fyne.CanvasObject {
	card := fynecontainer.NewVBox(
		widget.NewLabel("🔗 Connection"),
		d.statusLabel,
		widget.NewLabel(fmt.Sprintf("Updated: %s", d.lastUpdate.Format("15:04:05"))),
	)

	return fynecontainer.NewBorder(nil, nil, nil, nil, card)
}

// createRecentActivityCard creates the recent activity section
func (d *Dashboard) createRecentActivityCard() fyne.CanvasObject {
	activityList := widget.NewList(
		func() int { return len(d.instances) },
		func() fyne.CanvasObject {
			return widget.NewLabel("Instance Activity")
		},
		func(i widget.ListItemID, o fyne.CanvasObject) {
			if i < len(d.instances) {
				instance := d.instances[i]
				label := o.(*widget.Label)
				label.SetText(fmt.Sprintf("%s - %s", instance.Name, instance.State))
			}
		},
	)

	return fynecontainer.NewVBox(
		widget.NewLabel("📊 Recent Activity"),
		activityList,
	)
}

// createQuickActionsCard creates quick action buttons
func (d *Dashboard) createQuickActionsCard() fyne.CanvasObject {
	launchBtn := widget.NewButton("🚀 Quick Launch", func() {
		// TODO: Implement quick launch dialog
	})

	connectBtn := widget.NewButton("🔗 Quick Connect", func() {
		// TODO: Implement quick connect
	})

	actions := fynecontainer.NewGridWithColumns(2, launchBtn, connectBtn)

	return fynecontainer.NewVBox(
		widget.NewLabel("⚡ Quick Actions"),
		actions,
	)
}

// calculateTotalCost calculates total cost of running instances
func (d *Dashboard) calculateTotalCost() {
	d.totalCost = 0
	for _, instance := range d.instances {
		if instance.State == "running" {
			d.totalCost += instance.EstimatedDailyCost / 24 // Convert to hourly
		}
	}
}

// updateLabels updates the dashboard labels
func (d *Dashboard) updateLabels() {
	runningCount := 0
	for _, instance := range d.instances {
		if instance.State == "running" {
			runningCount++
		}
	}

	d.costLabel.SetText(fmt.Sprintf("$%.3f/hour", d.totalCost))
	d.instancesLabel.SetText(fmt.Sprintf("%d running", runningCount))
	d.statusLabel.SetText("Status: Connected")
}

// refresh refreshes dashboard data
func (d *Dashboard) refresh() {
	go func() {
		if err := d.UpdateView(); err != nil {
			d.statusLabel.SetText("Status: Refresh failed")
		}
	}()
}
