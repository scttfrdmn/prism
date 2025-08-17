package tests

import (
	"testing"

	"github.com/scttfrdmn/cloudworkstation/internal/tui/components"
	"github.com/stretchr/testify/assert"
)

// TestNotificationCreation tests creation of the notification center
func TestNotificationCreation(t *testing.T) {
	nc := components.NewNotificationCenter()

	// Initial state should have no notifications
	view := nc.View()
	assert.Empty(t, view, "New notification center should have empty view")
}

// TestAddNotification tests adding a notification
func TestAddNotification(t *testing.T) {
	nc := components.NewNotificationCenter()

	// Add a notification
	message := "Test notification"
	cmd := nc.AddNotification(message, components.NotificationInfo)

	// Command should be returned
	assert.NotNil(t, cmd, "Command should be returned when adding notification")

	// View should not be empty now
	view := nc.View()
	assert.NotEmpty(t, view, "View should not be empty after adding notification")
}

// TestNotificationTypes tests different notification types
func TestNotificationTypes(t *testing.T) {
	nc := components.NewNotificationCenter()

	// Add notifications of different types
	nc.AddNotification("Info notification", components.NotificationInfo)
	nc.AddNotification("Success notification", components.NotificationSuccess)
	nc.AddNotification("Warning notification", components.NotificationWarning)
	nc.AddNotification("Error notification", components.NotificationError)

	// View should contain all notifications
	view := nc.View()
	assert.NotEmpty(t, view, "View should not be empty with notifications")
}

// TestRemoveNotification tests removing a notification
func TestRemoveNotification(t *testing.T) {
	nc := components.NewNotificationCenter()

	// Add a notification
	nc.AddNotification("Test notification", components.NotificationInfo)

	// Add notification
	_ = nc.AddNotification("Test notification", components.NotificationInfo)

	// Force remove all notifications
	nc.ClearAllNotifications()

	// View should now be empty
	view := nc.View()
	assert.Empty(t, view, "View should be empty after removing notification")
}

// TestNotificationUpdate tests updating the notification center
func TestNotificationUpdate(t *testing.T) {
	nc := components.NewNotificationCenter()

	// Add a notification
	_ = nc.AddNotification("Test notification", components.NotificationInfo)

	// Clear notifications manually since we can't easily simulate the timeout
	nc.ClearAllNotifications()

	// View should be empty
	view := nc.View()
	assert.Empty(t, view, "View should be empty after timeout")
}

// TestNotificationSize tests setting the size of the notification center
func TestNotificationSize(t *testing.T) {
	nc := components.NewNotificationCenter()

	// Set size
	nc.SetSize(100, 50)

	// Add a notification
	nc.AddNotification("Test notification", components.NotificationInfo)

	// The actual effect is on rendering, but we can at least ensure it doesn't crash
	view := nc.View()
	assert.NotEmpty(t, view, "View should not be empty after setting size")
}

// TestMultipleNotifications tests adding multiple notifications
func TestMultipleNotifications(t *testing.T) {
	nc := components.NewNotificationCenter()

	// Add multiple notifications
	nc.AddNotification("First notification", components.NotificationInfo)
	nc.AddNotification("Second notification", components.NotificationSuccess)
	nc.AddNotification("Third notification", components.NotificationWarning)

	// View should contain all notifications
	view := nc.View()
	assert.NotEmpty(t, view, "View should not be empty with multiple notifications")
}

// TestNotificationTimeoutCmd tests the timeout command
func TestNotificationTimeoutCmd(t *testing.T) {
	// This is more of a smoke test since we can't easily test the timing
	nc := components.NewNotificationCenter()

	// Add a notification with a very short timeout
	cmd := nc.AddNotification("Test notification", components.NotificationInfo)
	assert.NotNil(t, cmd, "Command should be returned with timeout")
}
