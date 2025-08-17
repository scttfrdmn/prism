package components

import (
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/scttfrdmn/cloudworkstation/internal/tui/styles"
)

// NotificationType represents different notification types
type NotificationType int

const (
	// NotificationInfo is for informational notifications
	NotificationInfo NotificationType = iota
	// NotificationSuccess is for success notifications
	NotificationSuccess
	// NotificationWarning is for warning notifications
	NotificationWarning
	// NotificationError is for error notifications
	NotificationError
)

// NotificationMsg is sent when a new notification is created
type NotificationMsg struct {
	ID      string
	Message string
	Type    NotificationType
}

// NotificationTimeoutMsg is sent when a notification times out
type NotificationTimeoutMsg struct {
	ID string
}

// Notification represents a notification
type Notification struct {
	ID        string
	Message   string
	Type      NotificationType
	CreatedAt time.Time
	Timeout   time.Duration
}

// NotificationCenter manages notifications
type NotificationCenter struct {
	notifications    []Notification
	width            int
	height           int
	visible          bool
	maxNotifications int
}

// NewNotificationCenter creates a new notification center
func NewNotificationCenter() NotificationCenter {
	return NotificationCenter{
		notifications:    []Notification{},
		width:            40,
		height:           5,
		visible:          false,
		maxNotifications: 5,
	}
}

// SetSize sets the size of the notification center
func (n *NotificationCenter) SetSize(width, height int) {
	n.width = width
	n.height = height
}

// AddNotification adds a new notification
func (n *NotificationCenter) AddNotification(message string, notificationType NotificationType) tea.Cmd {
	// Create a unique ID for the notification
	id := fmt.Sprintf("%d", time.Now().UnixNano())

	// Create notification
	notification := Notification{
		ID:        id,
		Message:   message,
		Type:      notificationType,
		CreatedAt: time.Now(),
		Timeout:   5 * time.Second, // Default timeout
	}

	// Add to list, limited by maxNotifications
	n.notifications = append(n.notifications, notification)
	if len(n.notifications) > n.maxNotifications {
		n.notifications = n.notifications[1:]
	}

	// Make notification center visible
	n.visible = true

	// Create notification timeout command
	return tea.Batch(
		func() tea.Msg {
			return NotificationMsg{
				ID:      id,
				Message: message,
				Type:    notificationType,
			}
		},
		n.timeoutCmd(id, notification.Timeout),
	)
}

// RemoveNotification removes a notification by ID
func (n *NotificationCenter) RemoveNotification(id string) tea.Cmd {
	for i, notification := range n.notifications {
		if notification.ID == id {
			n.notifications = append(n.notifications[:i], n.notifications[i+1:]...)
			break
		}
	}

	// Hide notification center if no notifications
	if len(n.notifications) == 0 {
		n.visible = false
	}

	return nil
}

// timeoutCmd creates a timeout command for a notification
func (n *NotificationCenter) timeoutCmd(id string, duration time.Duration) tea.Cmd {
	return tea.Tick(duration, func(time.Time) tea.Msg {
		return NotificationTimeoutMsg{ID: id}
	})
}

// Update handles messages and updates the model
func (n *NotificationCenter) Update(msg tea.Msg) (NotificationCenter, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case NotificationTimeoutMsg:
		cmd = n.RemoveNotification(msg.ID)
	}

	return *n, cmd
}

// ClearAllNotifications clears all notifications
func (n *NotificationCenter) ClearAllNotifications() {
	n.notifications = []Notification{}
	n.visible = false
}

// View renders the notification center
func (n NotificationCenter) View() string {
	if !n.visible || len(n.notifications) == 0 {
		return ""
	}

	theme := styles.CurrentTheme

	var notifications []string
	for _, notification := range n.notifications {
		// Choose style based on notification type
		var style lipgloss.Style
		switch notification.Type {
		case NotificationInfo:
			style = theme.Panel.BorderForeground(theme.PrimaryColor)
		case NotificationSuccess:
			style = theme.Panel.BorderForeground(theme.SuccessColor)
		case NotificationWarning:
			style = theme.Panel.BorderForeground(theme.WarningColor)
		case NotificationError:
			style = theme.Panel.BorderForeground(theme.ErrorColor)
		}

		// Create notification view
		notificationView := style.Width(n.width).Render(notification.Message)
		notifications = append(notifications, notificationView)
	}

	return lipgloss.JoinVertical(lipgloss.Left, notifications...)
}
