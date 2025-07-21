package test

import (
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"
	"github.com/stretchr/testify/assert"

	"github.com/scttfrdmn/cloudworkstation/cmd/cws-gui/widgets"
)

// TestResponsiveLayout tests the responsive layout functionality
func TestResponsiveLayout(t *testing.T) {
	// Create test app
	app := test.NewApp()
	defer app.Quit()

	// Create wide layout (horizontal)
	wideContent := container.NewHBox(
		widget.NewLabel("Left"),
		widget.NewLabel("Center"),
		widget.NewLabel("Right"),
	)

	// Create narrow layout (vertical)
	narrowContent := container.NewVBox(
		widget.NewLabel("Top"),
		widget.NewLabel("Middle"),
		widget.NewLabel("Bottom"),
	)

	// Create responsive layout with 500px breakpoint
	breakpoint := float32(500)
	respLayout := widgets.NewResponsiveLayout(wideContent, narrowContent, breakpoint)
	content := container.New(respLayout)

	// Create a window to test with
	window := app.NewWindow("Test")
	window.SetContent(content)

	t.Run("WideLayout", func(t *testing.T) {
		// Set window size above breakpoint
		window.Resize(fyne.NewSize(600, 400))

		// Let the canvas update
		test.Sync()

		// Verify wide layout is visible and narrow is hidden
		assert.True(t, wideContent.Visible())
		assert.False(t, narrowContent.Visible())
	})

	t.Run("NarrowLayout", func(t *testing.T) {
		// Set window size below breakpoint
		window.Resize(fyne.NewSize(400, 400))

		// Let the canvas update
		test.Sync()

		// Verify narrow layout is visible and wide is hidden
		assert.True(t, narrowContent.Visible())
		assert.False(t, wideContent.Visible())
	})

	t.Run("SwitchingLayouts", func(t *testing.T) {
		// Start with wide layout
		window.Resize(fyne.NewSize(600, 400))
		test.Sync()
		assert.True(t, wideContent.Visible())
		assert.False(t, narrowContent.Visible())

		// Switch to narrow layout
		window.Resize(fyne.NewSize(400, 400))
		test.Sync()
		assert.True(t, narrowContent.Visible())
		assert.False(t, wideContent.Visible())

		// Back to wide layout
		window.Resize(fyne.NewSize(600, 400))
		test.Sync()
		assert.True(t, wideContent.Visible())
		assert.False(t, narrowContent.Visible())
	})

	t.Run("MinSize", func(t *testing.T) {
		// Test that minimum size is calculated correctly
		minSize := respLayout.MinSize(nil)
		
		wideMin := wideContent.MinSize()
		narrowMin := narrowContent.MinSize()
		
		// Should be the larger of the two layouts' minimum sizes
		expectedWidth := fyne.Max(wideMin.Width, narrowMin.Width)
		expectedHeight := fyne.Max(wideMin.Height, narrowMin.Height)
		
		assert.Equal(t, expectedWidth, minSize.Width)
		assert.Equal(t, expectedHeight, minSize.Height)
	})
}