package tests

import (
	"testing"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/stretchr/testify/assert"

	"github.com/scttfrdmn/cloudworkstation/pkg/types"
)

// TestUIComponents tests various UI components in isolation
func TestUIComponents(t *testing.T) {
	app := test.NewApp()
	defer app.Quit()

	t.Run("InstanceCard", func(t *testing.T) {
		// Create a test instance
		instance := types.Instance{
			ID:                "i-12345",
			Name:              "test-instance",
			Template:          "r-research",
			InstanceType:      "t3.medium",
			State:             "running",
			PublicIP:          "1.2.3.4",
			LaunchTime:        time.Now(),
			EstimatedDailyCost: 2.40,
		}

		// Call the function to create an instance card
		card := createTestInstanceCard(instance)
		
		// Create a window to render the card
		window := app.NewWindow("Test")
		window.SetContent(card)
		window.Resize(fyne.NewSize(600, 200))
		
		// Basic assertions
		assert.NotNil(t, card)
		assert.Equal(t, "", card.Title) // Title should be empty as per the implementation
		
		// Test rendering
		test.AssertRendersToMarkup(t, "instance_card.xml", card)
	})

	t.Run("NotificationSystem", func(t *testing.T) {
		// Create notification container
		notification := container.NewVBox()
		
		// Create notification content
		content := createNotification("success", "Test Title", "Test Message")
		notification.Add(content)
		
		// Create a window to render
		window := app.NewWindow("Test")
		window.SetContent(notification)
		window.Resize(fyne.NewSize(400, 100))
		
		// Basic assertions
		assert.NotNil(t, notification)
		assert.Equal(t, 1, len(notification.Objects))
		
		// Test rendering
		test.AssertRendersToMarkup(t, "notification.xml", notification)
	})

	t.Run("DashboardView", func(t *testing.T) {
		// Create test instances
		instances := CreateMockInstances()
		
		// Create dashboard view
		dashboard := createTestDashboardView(instances, 2.40)
		
		// Create a window to render
		window := app.NewWindow("Test")
		window.SetContent(dashboard)
		window.Resize(fyne.NewSize(800, 600))
		
		// Basic assertions
		assert.NotNil(t, dashboard)
		
		// Test rendering
		test.AssertRendersToMarkup(t, "dashboard.xml", dashboard)
	})
}

// Helper functions to create test UI components

func createTestInstanceCard(instance types.Instance) *widget.Card {
	// Instance details
	details := container.NewVBox(
		widget.NewLabelWithStyle(instance.Name, fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		widget.NewLabel(fmt.Sprintf("Template: %s", instance.Template)),
		widget.NewLabel(fmt.Sprintf("Cost: $%.2f/day", instance.EstimatedDailyCost)),
		widget.NewLabel(fmt.Sprintf("Launched: %s", instance.LaunchTime.Format("Jan 2, 2006 15:04"))),
	)

	// Status
	status := container.NewVBox(
		container.NewHBox(
			widget.NewLabel(getTestStatusIcon(instance.State)),
			widget.NewLabelWithStyle(instance.State, fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		),
	)

	// Actions
	actions := container.NewVBox()

	if instance.State == "running" {
		actions.Add(widget.NewButton("Connect", func() {}))
		actions.Add(widget.NewButton("Stop", func() {}))
	} else if instance.State == "stopped" {
		actions.Add(widget.NewButton("Start", func() {}))
	}

	actions.Add(widget.NewButton("Delete", func() {}))

	// Card content
	cardContent := container.NewHBox(
		details,
		layout.NewSpacer(),
		status,
		layout.NewSpacer(),
		actions,
	)

	return widget.NewCard("", "", cardContent)
}

func getTestStatusIcon(state string) string {
	switch state {
	case "running":
		return "🟢"
	case "stopped":
		return "🟡"
	case "pending":
		return "🟠"
	case "stopping":
		return "🟠"
	case "terminated":
		return "🔴"
	default:
		return "⚫"
	}
}

func createNotification(notificationType, title, message string) fyne.CanvasObject {
	var icon fyne.Resource
	switch notificationType {
	case "success":
		icon = theme.ConfirmIcon()
	case "error":
		icon = theme.ErrorIcon()
	case "info":
		icon = theme.InfoIcon()
	default:
		icon = theme.InfoIcon()
	}

	// Create notification content
	var content *fyne.Container
	if message != "" {
		content = container.NewHBox(
			widget.NewIcon(icon),
			container.NewVBox(
				widget.NewLabelWithStyle(title, fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
				widget.NewLabel(message),
			),
			layout.NewSpacer(),
			widget.NewButton("×", func() {}),
		)
	} else {
		content = container.NewHBox(
			widget.NewIcon(icon),
			widget.NewLabelWithStyle(title, fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
			layout.NewSpacer(),
			widget.NewButton("×", func() {}),
		)
	}

	return widget.NewCard("", "", content)
}

func createTestDashboardView(instances []types.Instance, totalCost float64) fyne.CanvasObject {
	// Header
	header := container.NewHBox(
		widget.NewLabelWithStyle("Dashboard", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		layout.NewSpacer(),
		widget.NewButton("Refresh", func() {}),
	)

	// Overview cards
	overviewCards := container.NewGridWithColumns(3,
		widget.NewCard("Active Instances", "",
			container.NewVBox(
				widget.NewLabelWithStyle(fmt.Sprintf("%d", countRunningInstances(instances)), 
					fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
				widget.NewLabel("Currently running"),
			),
		),
		widget.NewCard("Daily Cost", "",
			container.NewVBox(
				widget.NewLabelWithStyle(fmt.Sprintf("$%.2f", totalCost), 
					fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
				widget.NewLabel("Estimated per day"),
			),
		),
		widget.NewCard("Total Instances", "",
			container.NewVBox(
				widget.NewLabelWithStyle(fmt.Sprintf("%d", len(instances)), 
					fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
				widget.NewLabel("All instances"),
			),
		),
	)

	// Quick launch section
	quickLaunchForm := createTestQuickLaunchForm()
	quickLaunchCard := widget.NewCard("Quick Launch", "Launch a new research environment",
		quickLaunchForm,
	)

	// Recent instances
	recentInstancesList := createTestRecentInstancesList(instances)
	recentInstancesCard := widget.NewCard("Recent Instances", "Your latest cloud workstations",
		recentInstancesList,
	)

	// Layout
	content := container.NewVBox(
		header,
		widget.NewSeparator(),
		overviewCards,
		widget.NewSeparator(),
		container.NewGridWithColumns(2,
			quickLaunchCard,
			recentInstancesCard,
		),
	)

	return container.NewScroll(content)
}

func countRunningInstances(instances []types.Instance) int {
	count := 0
	for _, instance := range instances {
		if instance.State == "running" {
			count++
		}
	}
	return count
}

func createTestQuickLaunchForm() *fyne.Container {
	// Template selection
	templateSelect := widget.NewSelect([]string{"r-research", "python-research", "basic-ubuntu"}, nil)
	templateSelect.SetSelected("r-research")

	// Instance name
	nameEntry := widget.NewEntry()
	nameEntry.SetPlaceHolder("my-workspace")

	// Size selection
	sizeSelect := widget.NewSelect([]string{"XS", "S", "M", "L", "XL"}, nil)
	sizeSelect.SetSelected("M")

	// Launch button
	launchBtn := widget.NewButton("🚀 Launch Environment", func() {})
	launchBtn.Importance = widget.HighImportance

	form := container.NewVBox(
		widget.NewForm(
			widget.NewFormItem("Template", templateSelect),
			widget.NewFormItem("Name", nameEntry),
			widget.NewFormItem("Size", sizeSelect),
		),
		launchBtn,
	)

	return form
}

func createTestRecentInstancesList(instances []types.Instance) *fyne.Container {
	if len(instances) == 0 {
		return container.NewVBox(
			widget.NewLabelWithStyle("No instances yet", fyne.TextAlignCenter, fyne.TextStyle{Italic: true}),
			widget.NewLabel("Launch your first environment using Quick Launch"),
		)
	}

	// Show up to 3 most recent instances
	items := make([]fyne.CanvasObject, 0)
	count := 0
	for _, instance := range instances {
		if count >= 3 {
			break
		}

		statusIcon := getTestStatusIcon(instance.State)

		instanceItem := container.NewHBox(
			widget.NewLabel(statusIcon),
			container.NewVBox(
				widget.NewLabelWithStyle(instance.Name, fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
				widget.NewLabel(fmt.Sprintf("%s • $%.2f/day", instance.Template, instance.EstimatedDailyCost)),
			),
			layout.NewSpacer(),
			widget.NewButton("Manage", func() {}),
		)

		items = append(items, instanceItem)
		count++
	}

	return container.NewVBox(items...)
}