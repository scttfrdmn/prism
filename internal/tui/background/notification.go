package background

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
)

// NotificationType defines the different types of system notifications
type NotificationType int

const (
	// NotificationInfo is for informational messages
	NotificationInfo NotificationType = iota

	// NotificationWarning is for warning messages
	NotificationWarning

	// NotificationError is for error messages
	NotificationError
)

// NotificationOptions holds configuration for a system notification
type NotificationOptions struct {
	// Title is the notification title
	Title string

	// Message is the notification body
	Message string

	// Type is the notification severity level
	Type NotificationType

	// Icon is an optional path to an icon image
	Icon string

	// Timeout is how long the notification should remain visible (in seconds)
	Timeout int

	// Actions are optional notification actions (only supported on some platforms)
	Actions []string
}

// DefaultOptions returns default notification options
func DefaultOptions() NotificationOptions {
	return NotificationOptions{
		Title:   "CloudWorkstation",
		Type:    NotificationInfo,
		Timeout: 5,
	}
}

// NotificationManager handles system notifications across different platforms
type NotificationManager struct {
	// DefaultOptions contains default settings for notifications
	DefaultOptions NotificationOptions

	// MonitorEvents determines if the manager should show notifications for monitor events
	MonitorEvents bool

	// MinSeverity filters notifications below this severity level
	MinSeverity NotificationType
}

// NewNotificationManager creates a new notification manager
func NewNotificationManager() *NotificationManager {
	return &NotificationManager{
		DefaultOptions: DefaultOptions(),
		MonitorEvents:  true,
		MinSeverity:    NotificationWarning, // Only show warnings and errors by default
	}
}

// Notify displays a system notification with the given options
func (n *NotificationManager) Notify(options NotificationOptions) error {
	if options.Type < n.MinSeverity {
		// Skip notifications below minimum severity
		return nil
	}

	// Merge with default options
	if options.Title == "" {
		options.Title = n.DefaultOptions.Title
	}

	if options.Timeout == 0 {
		options.Timeout = n.DefaultOptions.Timeout
	}

	// Choose notification method based on platform
	switch runtime.GOOS {
	case "darwin":
		return n.notifyMacOS(options)
	case "linux":
		return n.notifyLinux(options)
	case "windows":
		return n.notifyWindows(options)
	default:
		return fmt.Errorf("notifications not supported on this platform")
	}
}

// NotifyFromEvent creates a notification from an instance event
func (n *NotificationManager) NotifyFromEvent(event InstanceEvent) error {
	if !n.MonitorEvents {
		return nil
	}

	// Convert event level to notification type
	var notifType NotificationType
	switch event.Level {
	case EventLevelInfo:
		notifType = NotificationInfo
	case EventLevelWarning:
		notifType = NotificationWarning
	case EventLevelError:
		notifType = NotificationError
	default:
		notifType = NotificationInfo
	}

	// Skip if below minimum severity
	if notifType < n.MinSeverity {
		return nil
	}

	// Create notification options
	options := NotificationOptions{
		Title:   "CloudWorkstation",
		Message: event.Message,
		Type:    notifType,
		Timeout: 5,
	}

	// Customize based on event type
	switch event.Type {
	case EventTypeStateChange:
		options.Title = fmt.Sprintf("Instance State: %s", event.Instance)

	case EventTypeIdleWarning:
		options.Title = fmt.Sprintf("Idle Warning: %s", event.Instance)
		options.Timeout = 10 // Longer timeout for important warnings

	case EventTypeCostAlert:
		options.Title = "Cost Alert"
		options.Timeout = 10 // Longer timeout for important alerts

	case EventTypeError:
		options.Title = "CloudWorkstation Error"
	}

	return n.Notify(options)
}

// notifyMacOS sends a notification on macOS using AppleScript
func (n *NotificationManager) notifyMacOS(options NotificationOptions) error {
	// Escape double quotes in title and message
	title := strings.ReplaceAll(options.Title, "\"", "\\\"")
	message := strings.ReplaceAll(options.Message, "\"", "\\\"")

	script := fmt.Sprintf(`display notification "%s" with title "%s"`, message, title)

	// Add subtitle if needed
	if options.Type == NotificationWarning {
		script += ` subtitle "Warning"`
	} else if options.Type == NotificationError {
		script += ` subtitle "Error"`
	}

	// Add sound based on severity
	if options.Type == NotificationWarning {
		script += " sound name \"Basso\""
	} else if options.Type == NotificationError {
		script += " sound name \"Sosumi\""
	}

	cmd := exec.Command("osascript", "-e", script)
	return cmd.Run()
}

// notifyLinux sends a notification on Linux using notify-send
func (n *NotificationManager) notifyLinux(options NotificationOptions) error {
	// Check if notify-send is available
	if _, err := exec.LookPath("notify-send"); err != nil {
		return fmt.Errorf("notify-send not found: %w", err)
	}

	// Build command arguments
	args := []string{
		"--app-name=CloudWorkstation",
		fmt.Sprintf("--expire-time=%d", options.Timeout*1000),
	}

	// Add urgency based on notification type
	switch options.Type {
	case NotificationInfo:
		args = append(args, "--urgency=low")
	case NotificationWarning:
		args = append(args, "--urgency=normal")
	case NotificationError:
		args = append(args, "--urgency=critical")
	}

	// Add icon if specified
	if options.Icon != "" {
		args = append(args, fmt.Sprintf("--icon=%s", options.Icon))
	} else {
		// Use default icon based on type
		switch options.Type {
		case NotificationWarning:
			args = append(args, "--icon=dialog-warning")
		case NotificationError:
			args = append(args, "--icon=dialog-error")
		default:
			args = append(args, "--icon=dialog-information")
		}
	}

	// Add title and message
	args = append(args, options.Title, options.Message)

	cmd := exec.Command("notify-send", args...)
	return cmd.Run()
}

// notifyWindows sends a notification on Windows using PowerShell
func (n *NotificationManager) notifyWindows(options NotificationOptions) error {
	// Script to create a Windows notification
	script := `[Windows.UI.Notifications.ToastNotificationManager, Windows.UI.Notifications, ContentType = WindowsRuntime] | Out-Null
[Windows.UI.Notifications.ToastNotification, Windows.UI.Notifications, ContentType = WindowsRuntime] | Out-Null
[Windows.Data.Xml.Dom.XmlDocument, Windows.Data.Xml.Dom.XmlDocument, ContentType = WindowsRuntime] | Out-Null

$app = '{1AC14E77-02E7-4E5D-B744-2EB1AE5198B7}\WindowsPowerShell\v1.0\powershell.exe'
$template = @"
<toast>
    <visual>
        <binding template="ToastText02">
            <text id="1">%s</text>
            <text id="2">%s</text>
        </binding>
    </visual>
</toast>
"@

$xml = New-Object Windows.Data.Xml.Dom.XmlDocument
$xml.LoadXml($template)
$toast = New-Object Windows.UI.Notifications.ToastNotification $xml
[Windows.UI.Notifications.ToastNotificationManager]::CreateToastNotifier($app).Show($toast)
`

	// Escape any characters that might cause issues
	title := strings.ReplaceAll(options.Title, "\"", "'")
	message := strings.ReplaceAll(options.Message, "\"", "'")

	// Fill in the template
	script = fmt.Sprintf(script, title, message)

	// Create a temporary file for the PowerShell script
	tmpFile, err := os.CreateTemp("", "cwsnotify-*.ps1")
	if err != nil {
		return fmt.Errorf("failed to create temporary file: %w", err)
	}
	defer func() {
		if err := os.Remove(tmpFile.Name()); err != nil {
			// Log but don't fail on cleanup error
		}
	}()

	if _, err := tmpFile.WriteString(script); err != nil {
		return fmt.Errorf("failed to write script file: %w", err)
	}
	if err := tmpFile.Close(); err != nil {
		return fmt.Errorf("failed to close script file: %w", err)
	}

	// Execute the PowerShell script
	cmd := exec.Command("powershell", "-ExecutionPolicy", "Bypass", "-File", tmpFile.Name())
	return cmd.Run()
}

// MonitorAndNotify subscribes to monitor events and sends notifications
func (n *NotificationManager) MonitorAndNotify(monitor *InstanceMonitor) func() {
	// Subscribe to monitor events
	events, unsubscribe := monitor.Subscribe()

	// Start goroutine to handle events
	stopCh := make(chan struct{})
	go func() {
		for {
			select {
			case event, ok := <-events:
				if !ok {
					return
				}

				// Convert event to notification
				err := n.NotifyFromEvent(event)
				if err != nil {
					fmt.Printf("Failed to show notification: %v\n", err)
				}

			case <-stopCh:
				return
			}
		}
	}()

	// Return function to stop monitoring
	return func() {
		close(stopCh)
		unsubscribe()
	}
}
