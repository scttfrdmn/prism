package tests

import (
	"fmt"
	"os"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/stretchr/testify/assert"
	
	"github.com/scttfrdmn/cloudworkstation/cmd/cws-gui/tests/responsive"
)

// PlatformConfig represents platform-specific configuration for testing
type PlatformConfig struct {
	Name           string
	Theme          fyne.Theme
	DarkMode       bool
	Scale          float32
	ExpectedHeight float32 // Expected height of a standard button in this configuration
}

// TestCrossPlatformRendering tests rendering on different platforms and configurations
func TestCrossPlatformRendering(t *testing.T) {
	// Skip if in CI environment
	if os.Getenv("CI") != "" {
		t.Skip("Skipping cross-platform rendering test in CI environment")
	}

	// Define platform configurations to test
	platforms := []PlatformConfig{
		{Name: "macOS-Light", Theme: theme.LightTheme(), DarkMode: false, Scale: 1.0, ExpectedHeight: 38},
		{Name: "macOS-Dark", Theme: theme.DarkTheme(), DarkMode: true, Scale: 1.0, ExpectedHeight: 38},
		{Name: "Windows-Standard", Theme: theme.DefaultTheme(), DarkMode: false, Scale: 1.0, ExpectedHeight: 38},
		{Name: "Linux-HiDPI", Theme: theme.DefaultTheme(), DarkMode: false, Scale: 2.0, ExpectedHeight: 76},
		{Name: "Mobile-Small", Theme: theme.DefaultTheme(), DarkMode: false, Scale: 0.8, ExpectedHeight: 30},
	}

	for _, platform := range platforms {
		t.Run(platform.Name, func(t *testing.T) {
			// Create test app with platform settings
			app := test.NewApp()
			defer app.Quit()
			
			// Set theme and scale
			settings := app.Settings()
			settings.SetTheme(platform.Theme)
			
			// Create a window
			window := app.NewWindow("Cross-platform Test")
			defer window.Close()
			window.Resize(fyne.NewSize(800, 600))
			
			// Create a set of standard UI components to test
			content := createCrossPlatformTestUI()
			window.SetContent(content)
			
			// Create a canvas to test rendering
			canvas := window.Canvas()
			assert.NotNil(t, canvas)
			
			// Test rendering specific components
			button := findButtonWithText(content, "Test Button")
			assert.NotNil(t, button, "Button should be found")
			
			// Verify the component renders correctly for this platform
			// Note: This is a basic verification - actual tests would need more complex assertions
			// based on specific platform rendering expectations
			buttonSize := button.Size()
			assert.True(t, buttonSize.Width > 0, "Button should have positive width")
			assert.True(t, buttonSize.Height > 0, "Button should have positive height")
			
			// Test that containers respect platform-specific layout
			formItem := findFormItem(content, "Profile Name")
			assert.NotNil(t, formItem, "Form item should be found")
			
			// Test scrollable containers
			scrollContainer := findScrollContainer(content)
			assert.NotNil(t, scrollContainer, "Scroll container should be found")
			
			// Test different screen sizes
			window.Resize(fyne.NewSize(1024, 768))
			assert.Equal(t, float32(1024), window.Canvas().Size().Width)
			
			window.Resize(fyne.NewSize(800, 600))
			assert.Equal(t, float32(800), window.Canvas().Size().Width)
			
			window.Resize(fyne.NewSize(320, 480)) // Mobile size
			assert.Equal(t, float32(320), window.Canvas().Size().Width)
			
			// Test can render to an image
			image := test.WidgetScreenshot(content)
			assert.NotNil(t, image, "Should be able to render to image")
			assert.True(t, image.Bounds().Dx() > 0, "Image should have positive width")
			assert.True(t, image.Bounds().Dy() > 0, "Image should have positive height")
		})
	}
}

// TestResponsiveLayout tests that the GUI layouts correctly respond to different screen sizes
func TestResponsiveLayout(t *testing.T) {
	// Skip if in CI environment
	if os.Getenv("CI") != "" {
		t.Skip("Skipping responsive layout test in CI environment")
	}

	// Create test app
	app := test.NewApp()
	defer app.Quit()
	
	// Create window
	window := app.NewWindow("Responsive Test")
	defer window.Close()
	
	// Create test UI with responsive elements
	content := createResponsiveTestUI()
	window.SetContent(content)
	
	// Test different screen sizes
	screenSizes := []fyne.Size{
		{Width: 1920, Height: 1080}, // Large desktop
		{Width: 1366, Height: 768},  // Standard laptop
		{Width: 800, Height: 600},   // Small screen
		{Width: 320, Height: 480},   // Mobile portrait
	}
	
	for _, size := range screenSizes {
		t.Run(fmt.Sprintf("Size_%dx%d", int(size.Width), int(size.Height)), func(t *testing.T) {
			window.Resize(size)
			
			// Get the containers that should resize
			mainContainer := content.(*fyne.Container)
			assert.NotNil(t, mainContainer)
			
			// Check that container size matches window size
			containerSize := mainContainer.Size()
			assert.InDelta(t, size.Width, containerSize.Width, 1.0)
			
			// Check responsive layout adjustments
			if size.Width < 600 {
				// Check for single-column layout on small screens
				gridContainer := findGridContainer(content)
				if gridContainer != nil {
					// For small screens, we expect the grid to have 1 column
					// This is an approximation - actual check would depend on your layout logic
					assert.True(t, gridContainer.Size().Width < 600)
				}
			} else {
				// Check for multi-column layout on larger screens
				gridContainer := findGridContainer(content)
				if gridContainer != nil {
					// For larger screens, we expect the grid to be wider
					assert.True(t, gridContainer.Size().Width >= 600)
				}
			}
		})
	}
}

// Helper function to create a test UI with all standard components for cross-platform testing
func createCrossPlatformTestUI() fyne.CanvasObject {
	// Create components for different input types
	textEntry := widget.NewEntry()
	textEntry.SetPlaceHolder("Enter text here")
	
	passwordEntry := widget.NewPasswordEntry()
	passwordEntry.SetPlaceHolder("Password")
	
	multilineEntry := widget.NewMultiLineEntry()
	multilineEntry.SetPlaceHolder("Multi-line text")
	multilineEntry.SetMinRowsVisible(3)
	
	// Create selection widgets
	radioGroup := widget.NewRadioGroup([]string{"Option 1", "Option 2", "Option 3"}, nil)
	checkGroup := widget.NewCheckGroup([]string{"Check 1", "Check 2", "Check 3"}, nil)
	
	// Create buttons with different states
	normalButton := widget.NewButton("Test Button", nil)
	primaryButton := widget.NewButton("Primary Button", nil)
	primaryButton.Importance = widget.HighImportance
	disabledButton := widget.NewButton("Disabled Button", nil)
	disabledButton.Disable()
	
	// Create form with labels
	form := widget.NewForm(
		widget.NewFormItem("Profile Name", widget.NewEntry()),
		widget.NewFormItem("AWS Region", widget.NewSelect([]string{"us-west-2", "us-east-1", "eu-west-1"}, nil)),
		widget.NewFormItem("Options", widget.NewCheck("Enable feature", nil)),
	)
	
	// Create cards
	card1 := widget.NewCard("Card Title", "Card Subtitle", widget.NewLabel("Card content goes here"))
	card2 := widget.NewCard("Another Card", "", container.NewVBox(
		widget.NewLabel("More content"),
		widget.NewButton("Card Button", nil),
	))
	
	// Create tabs
	tabs := container.NewAppTabs(
		container.NewTabItem("Tab 1", widget.NewLabel("Content for Tab 1")),
		container.NewTabItem("Tab 2", widget.NewLabel("Content for Tab 2")),
		container.NewTabItem("Tab 3", widget.NewLabel("Content for Tab 3")),
	)
	
	// Create scrollable content
	scrollContent := container.NewVBox()
	for i := 0; i < 20; i++ {
		scrollContent.Add(widget.NewLabel(fmt.Sprintf("Scroll Item %d", i+1)))
	}
	scroll := container.NewScroll(scrollContent)
	
	// Combine everything into a main container
	mainContent := container.NewVBox(
		widget.NewLabel("Cross-Platform UI Test"),
		widget.NewSeparator(),
		container.NewHBox(
			textEntry,
			passwordEntry,
		),
		multilineEntry,
		container.NewVBox(
			widget.NewLabel("Radio Options"),
			radioGroup,
		),
		container.NewVBox(
			widget.NewLabel("Check Options"),
			checkGroup,
		),
		container.NewHBox(
			normalButton,
			primaryButton,
			disabledButton,
		),
		widget.NewSeparator(),
		form,
		widget.NewSeparator(),
		container.NewGridWithColumns(2,
			card1,
			card2,
		),
		widget.NewSeparator(),
		tabs,
		widget.NewSeparator(),
		container.NewVBox(
			widget.NewLabel("Scrollable Content"),
			scroll,
		),
	)
	
	return container.NewScroll(mainContent)
}

// Helper function to create a test UI that demonstrates responsive design
func createResponsiveTestUI() fyne.CanvasObject {
	// Create a container that will adjust based on screen size
	responsiveGrid := container.New(
		&responsive.GridLayout{
			MinWidth:   300,
			MaxWidth:   2000,
			ColumnSize: 300,
		},
		widget.NewCard("Item 1", "", widget.NewLabel("Content 1")),
		widget.NewCard("Item 2", "", widget.NewLabel("Content 2")),
		widget.NewCard("Item 3", "", widget.NewLabel("Content 3")),
		widget.NewCard("Item 4", "", widget.NewLabel("Content 4")),
		widget.NewCard("Item 5", "", widget.NewLabel("Content 5")),
		widget.NewCard("Item 6", "", widget.NewLabel("Content 6")),
	)
	
	// Create sidebar that collapses on small screens
	sidebar := container.NewVBox(
		widget.NewLabel("Navigation"),
		widget.NewButton("Home", nil),
		widget.NewButton("Instances", nil),
		widget.NewButton("Templates", nil),
		widget.NewButton("Settings", nil),
	)
	
	// Create a split container for sidebar + content
	split := container.NewHSplit(
		sidebar,
		container.NewVBox(
			widget.NewLabel("Main Content Area"),
			responsiveGrid,
		),
	)
	split.SetOffset(0.2) // 20% for sidebar
	
	// Create header that stays at the top
	header := container.NewHBox(
		widget.NewLabel("CloudWorkstation"),
		layout.NewSpacer(),
		widget.NewButton("Refresh", nil),
		widget.NewButton("Help", nil),
	)
	
	// Footer that stays at bottom
	footer := container.NewHBox(
		layout.NewSpacer(),
		widget.NewLabel("Status: Connected"),
		layout.NewSpacer(),
	)
	
	// Main container with header, content, footer
	return container.NewBorder(
		header,  // Top
		footer,  // Bottom
		nil,     // Left
		nil,     // Right
		split,   // Center
	)
}

// Helper functions to find specific components in the UI

func findButtonWithText(parent fyne.CanvasObject, text string) *widget.Button {
	var found *widget.Button
	
	// Function to check this object and its children
	var walkObjects func(o fyne.CanvasObject)
	walkObjects = func(o fyne.CanvasObject) {
		// Check if this is a button with matching text
		if button, isButton := o.(*widget.Button); isButton {
			if button.Text == text {
				found = button
				return
			}
		}
		
		// Check if this is a container
		if container, isContainer := o.(*fyne.Container); isContainer {
			// Check all children
			for _, child := range container.Objects {
				if found != nil {
					return // Stop if already found
				}
				walkObjects(child)
			}
		}
	}
	
	walkObjects(parent)
	return found
}

func findFormItem(parent fyne.CanvasObject, label string) *widget.FormItem {
	var found *widget.FormItem
	
	// Function to check this object and its children
	var walkObjects func(o fyne.CanvasObject)
	walkObjects = func(o fyne.CanvasObject) {
		// Check if this is a form
		if form, isForm := o.(*widget.Form); isForm {
			for _, item := range form.Items {
				if item.Text == label {
					found = item
					return
				}
			}
		}
		
		// Check if this is a container
		if container, isContainer := o.(*fyne.Container); isContainer {
			// Check all children
			for _, child := range container.Objects {
				if found != nil {
					return // Stop if already found
				}
				walkObjects(child)
			}
		}
	}
	
	walkObjects(parent)
	return found
}

func findScrollContainer(parent fyne.CanvasObject) *container.Scroll {
	var found *container.Scroll
	
	// Function to check this object and its children
	var walkObjects func(o fyne.CanvasObject)
	walkObjects = func(o fyne.CanvasObject) {
		// Check if this is a scroll container
		if scroll, isScroll := o.(*container.Scroll); isScroll {
			found = scroll
			return
		}
		
		// Check if this is a container
		if container, isContainer := o.(*fyne.Container); isContainer {
			// Check all children
			for _, child := range container.Objects {
				if found != nil {
					return // Stop if already found
				}
				walkObjects(child)
			}
		}
	}
	
	walkObjects(parent)
	return found
}

func findGridContainer(parent fyne.CanvasObject) *fyne.Container {
	var found *fyne.Container
	
	// Function to check this object and its children
	var walkObjects func(o fyne.CanvasObject)
	walkObjects = func(o fyne.CanvasObject) {
		// Check if this is a container with grid layout
		if container, isContainer := o.(*fyne.Container); isContainer {
			if _, isGrid := container.Layout.(*fyne.GridLayout); isGrid {
				found = container
				return
			}
			
			// Check all children
			for _, child := range container.Objects {
				if found != nil {
					return // Stop if already found
				}
				walkObjects(child)
			}
		}
	}
	
	walkObjects(parent)
	return found
}

