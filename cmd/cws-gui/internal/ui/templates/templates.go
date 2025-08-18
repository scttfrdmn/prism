package templates

import (
	"context"
	"fmt"
	"time"

	"fyne.io/fyne/v2"
	fynecontainer "fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"

	"github.com/scttfrdmn/cloudworkstation/pkg/api"
	pkgtemplates "github.com/scttfrdmn/cloudworkstation/pkg/templates"
	"github.com/scttfrdmn/cloudworkstation/pkg/types"
)

// Templates implements the Section interface for template management
type Templates struct {
	apiClient api.CloudWorkstationAPI
	window    fyne.Window
	templates map[string]types.RuntimeTemplate

	// UI components
	templateList *widget.List
	selectedKey  string
	launchButton *widget.Button
	infoButton   *widget.Button
	nameEntry    *widget.Entry
	sizeSelect   *widget.Select
}

// NewTemplates creates a new templates section
func NewTemplates(apiClient api.CloudWorkstationAPI, window fyne.Window) *Templates {
	t := &Templates{
		apiClient: apiClient,
		window:    window,
		templates: make(map[string]types.RuntimeTemplate),
	}

	t.setupUI()
	return t
}

// CreateView creates the templates browsing view
func (t *Templates) CreateView() fyne.CanvasObject {
	// Left side: template list
	listContainer := fynecontainer.NewVBox(
		widget.NewLabel("📋 Available Templates"),
		t.templateList,
	)

	// Right side: launch form
	launchForm := t.createLaunchForm()

	// Main layout
	mainView := fynecontainer.NewBorder(
		nil, nil, listContainer, launchForm, nil,
	)

	return mainView
}

// UpdateView refreshes the templates data
func (t *Templates) UpdateView() error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	templates, err := t.apiClient.ListTemplates(ctx)
	if err != nil {
		return fmt.Errorf("failed to fetch templates: %w", err)
	}

	t.templates = templates
	t.templateList.Refresh()
	t.updateLaunchForm()

	return nil
}

// GetTitle returns the section title
func (t *Templates) GetTitle() string {
	return "Templates"
}

// setupUI initializes the UI components
func (t *Templates) setupUI() {
	t.setupTemplateList()
	t.setupLaunchForm()
}

// setupTemplateList configures the template list widget
func (t *Templates) setupTemplateList() {
	t.templateList = widget.NewList(
		func() int { return len(t.templates) },
		func() fyne.CanvasObject {
			return t.createTemplateCard()
		},
		func(i widget.ListItemID, o fyne.CanvasObject) {
			keys := t.getTemplateKeys()
			if i < len(keys) {
				key := keys[i]
				template := t.templates[key]
				t.updateTemplateCard(o, key, template)
			}
		},
	)

	t.templateList.OnSelected = func(id widget.ListItemID) {
		keys := t.getTemplateKeys()
		if id < len(keys) {
			t.selectedKey = keys[id]
			t.updateLaunchForm()
		}
	}
}

// createTemplateCard creates a template for template display
func (t *Templates) createTemplateCard() fyne.CanvasObject {
	nameLabel := widget.NewLabel("")
	descLabel := widget.NewLabel("")
	costLabel := widget.NewLabel("")

	card := fynecontainer.NewVBox(
		nameLabel,
		descLabel,
		costLabel,
	)

	return card
}

// updateTemplateCard updates a template card with data
func (t *Templates) updateTemplateCard(obj fyne.CanvasObject, key string, template types.RuntimeTemplate) {
	card := obj.(*fyne.Container)

	nameLabel := card.Objects[0].(*widget.Label)
	descLabel := card.Objects[1].(*widget.Label)
	costLabel := card.Objects[2].(*widget.Label)

	nameLabel.SetText(fmt.Sprintf("🔧 %s", template.Name))
	descLabel.SetText(template.Description)

	// Calculate cost for x86_64 by default
	if cost, exists := template.EstimatedCostPerHour["x86_64"]; exists {
		costLabel.SetText(fmt.Sprintf("💰 $%.3f/hour", cost))
	} else {
		costLabel.SetText("💰 Cost: Unknown")
	}
}

// setupLaunchForm configures the launch form
func (t *Templates) setupLaunchForm() {
	t.nameEntry = widget.NewEntry()
	t.nameEntry.SetPlaceHolder("Enter instance name...")

	t.sizeSelect = widget.NewSelect([]string{"XS", "S", "M", "L", "XL"}, nil)
	t.sizeSelect.SetSelected("M") // Default size

	t.launchButton = widget.NewButton("🚀 Launch Instance", t.launchInstance)
	t.infoButton = widget.NewButton("ℹ️ Template Info", t.showTemplateInfo)

	t.updateLaunchForm()
}

// createLaunchForm creates the launch form container
func (t *Templates) createLaunchForm() fyne.CanvasObject {
	form := fynecontainer.NewVBox(
		widget.NewLabel("🚀 Launch Configuration"),

		widget.NewLabel("Instance Name:"),
		t.nameEntry,

		widget.NewLabel("Size:"),
		t.sizeSelect,

		fynecontainer.NewGridWithColumns(2,
			t.launchButton,
			t.infoButton,
		),
	)

	return form
}

// updateLaunchForm updates the launch form based on selection
func (t *Templates) updateLaunchForm() {
	hasSelection := t.selectedKey != ""

	t.launchButton.Enable()
	t.infoButton.Enable()

	if !hasSelection {
		t.launchButton.Disable()
		t.infoButton.Disable()
	}
}

// getTemplateKeys returns sorted template keys
func (t *Templates) getTemplateKeys() []string {
	keys := make([]string, 0, len(t.templates))
	for key := range t.templates {
		keys = append(keys, key)
	}
	return keys
}

// launchInstance launches the selected template
func (t *Templates) launchInstance() {
	if t.selectedKey == "" {
		return
	}

	instanceName := t.nameEntry.Text
	if instanceName == "" {
		dialog.ShowError(fmt.Errorf("please enter an instance name"), t.window)
		return
	}

	size := t.sizeSelect.Selected
	if size == "" {
		size = "M" // Default
	}

	// Create launch request
	req := types.LaunchRequest{
		Name:     instanceName,
		Template: t.selectedKey,
		Size:     size,
	}

	// Show progress dialog
	progress := dialog.NewProgressInfinite(
		"Launching Instance",
		fmt.Sprintf("Launching '%s' with template '%s'...", instanceName, t.selectedKey),
		t.window,
	)
	progress.Show()

	// Launch in background
	go func() {
		defer progress.Hide()

		ctx, cancel := context.WithTimeout(context.Background(), 300*time.Second)
		defer cancel()

		_, err := t.apiClient.LaunchInstance(ctx, req)
		if err != nil {
			dialog.ShowError(fmt.Errorf("failed to launch instance: %w", err), t.window)
			return
		}

		dialog.ShowInformation(
			"Success",
			fmt.Sprintf("Instance '%s' launched successfully!", instanceName),
			t.window,
		)

		// Clear form
		t.nameEntry.SetText("")
	}()
}

// showTemplateInfo displays detailed template information
func (t *Templates) showTemplateInfo() {
	if t.selectedKey == "" {
		return
	}

	// Get detailed template info
	templateInfo, err := pkgtemplates.GetTemplateInfo(t.selectedKey)
	if err != nil {
		dialog.ShowError(fmt.Errorf("failed to get template info: %w", err), t.window)
		return
	}

	template := t.templates[t.selectedKey]

	// Build info text
	info := fmt.Sprintf("📋 %s\n\n", template.Name)
	info += fmt.Sprintf("📝 %s\n\n", template.Description)
	info += fmt.Sprintf("🖥️ Base OS: %s\n", templateInfo.Base)
	info += fmt.Sprintf("📦 Package Manager: %s\n\n", templateInfo.PackageManager)

	// Add cost information
	if cost, exists := template.EstimatedCostPerHour["x86_64"]; exists {
		info += fmt.Sprintf("💰 Estimated Cost: $%.3f/hour ($%.2f/day)\n\n", cost, cost*24)
	}

	// Add instance type info
	if instanceType, exists := template.InstanceType["x86_64"]; exists {
		info += fmt.Sprintf("🔧 Instance Type: %s\n", instanceType)
	}

	dialog.ShowInformation("Template Information", info, t.window)
}
