package instances

import (
	"context"
	"fmt"
	"time"

	"fyne.io/fyne/v2"
	fynecontainer "fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"

	"github.com/scttfrdmn/cloudworkstation/pkg/api"
	"github.com/scttfrdmn/cloudworkstation/pkg/types"
)

// Instances implements the Section interface for instance management
type Instances struct {
	apiClient api.CloudWorkstationAPI
	window    fyne.Window
	instances []types.Instance

	// UI components
	instanceList  *widget.List
	selectedIndex int
	actionButtons *fyne.Container
}

// NewInstances creates a new instances section
func NewInstances(apiClient api.CloudWorkstationAPI, window fyne.Window) *Instances {
	i := &Instances{
		apiClient:     apiClient,
		window:        window,
		selectedIndex: -1,
	}

	i.setupInstanceList()
	i.setupActionButtons()

	return i
}

// CreateView creates the instances management view
func (i *Instances) CreateView() fyne.CanvasObject {
	// Main layout with list on left, actions on right
	listContainer := fynecontainer.NewVBox(
		widget.NewLabel("🖥️ Your Instances"),
		i.instanceList,
	)

	actionsContainer := fynecontainer.NewVBox(
		widget.NewLabel("⚡ Actions"),
		i.actionButtons,
	)

	mainView := fynecontainer.NewBorder(
		nil, nil, listContainer, actionsContainer, nil,
	)

	return mainView
}

// UpdateView refreshes the instances data
func (i *Instances) UpdateView() error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	resp, err := i.apiClient.ListInstances(ctx)
	if err != nil {
		return fmt.Errorf("failed to fetch instances: %w", err)
	}

	i.instances = resp.Instances
	i.instanceList.Refresh()
	i.updateActionButtons()

	return nil
}

// GetTitle returns the section title
func (i *Instances) GetTitle() string {
	return "Instances"
}

// setupInstanceList configures the instance list widget
func (i *Instances) setupInstanceList() {
	i.instanceList = widget.NewList(
		func() int { return len(i.instances) },
		func() fyne.CanvasObject {
			return i.createInstanceCard()
		},
		func(id widget.ListItemID, o fyne.CanvasObject) {
			if id < len(i.instances) {
				i.updateInstanceCard(o, i.instances[id])
			}
		},
	)

	i.instanceList.OnSelected = func(id widget.ListItemID) {
		i.selectedIndex = id
		i.updateActionButtons()
	}
}

// createInstanceCard creates a template for instance display
func (i *Instances) createInstanceCard() fyne.CanvasObject {
	nameLabel := widget.NewLabel("")
	stateLabel := widget.NewLabel("")
	typeLabel := widget.NewLabel("")
	costLabel := widget.NewLabel("")

	card := fynecontainer.NewVBox(
		fynecontainer.NewHBox(nameLabel, stateLabel),
		fynecontainer.NewHBox(typeLabel, costLabel),
	)

	return card
}

// updateInstanceCard updates an instance card with data
func (i *Instances) updateInstanceCard(obj fyne.CanvasObject, instance types.Instance) {
	card := obj.(*fyne.Container)

	// Top row: name and state
	topRow := card.Objects[0].(*fyne.Container)
	nameLabel := topRow.Objects[0].(*widget.Label)
	stateLabel := topRow.Objects[1].(*widget.Label)

	// Bottom row: type and cost
	bottomRow := card.Objects[1].(*fyne.Container)
	typeLabel := bottomRow.Objects[0].(*widget.Label)
	costLabel := bottomRow.Objects[1].(*widget.Label)

	// Update labels
	nameLabel.SetText(instance.Name)
	stateLabel.SetText(fmt.Sprintf("State: %s", instance.State))
	typeLabel.SetText(fmt.Sprintf("Type: %s", instance.InstanceType))
	costLabel.SetText(fmt.Sprintf("Cost: $%.2f/day", instance.EstimatedDailyCost))
}

// setupActionButtons creates action buttons for instance management
func (i *Instances) setupActionButtons() {
	startBtn := widget.NewButton("▶️ Start", i.startInstance)
	stopBtn := widget.NewButton("⏹️ Stop", i.stopInstance)
	hibernateBtn := widget.NewButton("💤 Hibernate", i.hibernateInstance)
	resumeBtn := widget.NewButton("🔄 Resume", i.resumeInstance)
	connectBtn := widget.NewButton("🔗 Connect", i.connectInstance)
	deleteBtn := widget.NewButton("🗑️ Delete", i.deleteInstance)

	i.actionButtons = fynecontainer.NewVBox(
		startBtn,
		stopBtn,
		hibernateBtn,
		resumeBtn,
		connectBtn,
		widget.NewSeparator(),
		deleteBtn,
	)

	i.updateActionButtons()
}

// updateActionButtons enables/disables buttons based on selection and state
func (i *Instances) updateActionButtons() {
	hasSelection := i.selectedIndex >= 0 && i.selectedIndex < len(i.instances)

	for _, obj := range i.actionButtons.Objects {
		if btn, ok := obj.(*widget.Button); ok {
			btn.Enable()
			if !hasSelection {
				btn.Disable()
			}
		}
	}
}

// Instance action methods (Single Responsibility Principle)

func (i *Instances) startInstance() {
	i.performInstanceAction("start", i.apiClient.StartInstance)
}

func (i *Instances) stopInstance() {
	i.performInstanceAction("stop", i.apiClient.StopInstance)
}

func (i *Instances) hibernateInstance() {
	i.performInstanceAction("hibernate", i.apiClient.HibernateInstance)
}

func (i *Instances) resumeInstance() {
	i.performInstanceAction("resume", i.apiClient.ResumeInstance)
}

func (i *Instances) deleteInstance() {
	if i.selectedIndex < 0 || i.selectedIndex >= len(i.instances) {
		return
	}

	instance := i.instances[i.selectedIndex]

	// Confirmation dialog
	dialog.ShowConfirm(
		"Delete Instance",
		fmt.Sprintf("Are you sure you want to delete '%s'?", instance.Name),
		func(confirm bool) {
			if confirm {
				i.performInstanceAction("delete", i.apiClient.DeleteInstance)
			}
		},
		i.window,
	)
}

func (i *Instances) connectInstance() {
	if i.selectedIndex < 0 || i.selectedIndex >= len(i.instances) {
		return
	}

	instance := i.instances[i.selectedIndex]

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	connInfo, err := i.apiClient.ConnectInstance(ctx, instance.Name)
	if err != nil {
		dialog.ShowError(err, i.window)
		return
	}

	// Show connection info dialog
	dialog.ShowInformation(
		"Connection Info",
		fmt.Sprintf("Connect to %s:\n\n%s", instance.Name, connInfo),
		i.window,
	)
}

// performInstanceAction performs a generic instance action
func (i *Instances) performInstanceAction(action string, apiFunc func(context.Context, string) error) {
	if i.selectedIndex < 0 || i.selectedIndex >= len(i.instances) {
		return
	}

	instance := i.instances[i.selectedIndex]

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err := apiFunc(ctx, instance.Name)
	if err != nil {
		dialog.ShowError(fmt.Errorf("failed to %s instance: %w", action, err), i.window)
		return
	}

	// Refresh the view after action
	go func() {
		time.Sleep(2 * time.Second) // Give AWS time to process
		i.UpdateView()
	}()
}
