package models

import (
	"context"
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/scttfrdmn/cloudworkstation/internal/tui/api"
	"github.com/scttfrdmn/cloudworkstation/internal/tui/components"
	"github.com/scttfrdmn/cloudworkstation/internal/tui/styles"
)

// StorageModel represents the storage management view
type StorageModel struct {
	apiClient         apiClient
	statusBar         components.StatusBar
	spinner           components.Spinner
	width             int
	height            int
	loading           bool
	error             string
	volumes           map[string]api.VolumeResponse
	storage           map[string]api.StorageResponse
	instances         []api.InstanceResponse
	selectedTab       int // 0=volumes, 1=storage
	selectedItem      int // Selected item in current tab
	showMountDialog   bool
	mountVolumeName   string
	mountInstanceName string
	mountPoint        string
}

// NewStorageModel creates a new storage model
func NewStorageModel(apiClient apiClient) StorageModel {
	// Create status bar and spinner
	statusBar := components.NewStatusBar("CloudWorkstation Storage", "")
	spinner := components.NewSpinner("Loading storage information...")

	return StorageModel{
		apiClient:    apiClient,
		statusBar:    statusBar,
		spinner:      spinner,
		width:        80,
		height:       24,
		loading:      true,
		volumes:      make(map[string]api.VolumeResponse),
		storage:      make(map[string]api.StorageResponse),
		instances:    []api.InstanceResponse{},
		selectedTab:  0,
		selectedItem: 0,
		mountPoint:   "/mnt/shared-volume", // Default mount point
	}
}

// Init initializes the model
func (m StorageModel) Init() tea.Cmd {
	return tea.Batch(
		m.spinner.InitialCmd(),
		m.fetchStorage,
	)
}

// fetchStorage retrieves storage data from the API
func (m StorageModel) fetchStorage() tea.Msg {
	// Fetch EFS volumes, EBS storage, and instances
	volumesResp, err := m.apiClient.ListVolumes(context.Background())
	if err != nil {
		return fmt.Errorf("failed to list EFS volumes: %w", err)
	}

	storageResp, err := m.apiClient.ListStorage(context.Background())
	if err != nil {
		return fmt.Errorf("failed to list EBS storage: %w", err)
	}

	instancesResp, err := m.apiClient.ListInstances(context.Background())
	if err != nil {
		return fmt.Errorf("failed to list instances: %w", err)
	}

	return StorageDataMsg{
		Volumes:   volumesResp.Volumes,
		Storage:   storageResp.Storage,
		Instances: instancesResp.Instances,
	}
}

// StorageDataMsg represents storage data retrieved from the API
type StorageDataMsg struct {
	Volumes   map[string]api.VolumeResponse
	Storage   map[string]api.StorageResponse
	Instances []api.InstanceResponse
}

// MountActionMsg represents a mount action result
type MountActionMsg struct {
	Success  bool
	Message  string
	Action   string // "mount" or "unmount"
	Volume   string
	Instance string
}

// Update handles messages and updates the model
func (m StorageModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		return m.handleWindowResize(msg)

	case tea.KeyMsg:
		return m.handleKeyboardInput(msg)

	case RefreshMsg:
		return m.handleRefresh()

	case error:
		return m.handleError(msg)

	case StorageDataMsg:
		return m.handleStorageData(msg)

	case MountActionMsg:
		return m.handleMountAction(msg)
	}

	// Update components
	if m.loading {
		var spinnerCmd tea.Cmd
		m.spinner, spinnerCmd = m.spinner.Update(msg)
		cmds = append(cmds, spinnerCmd)
	}

	return m, tea.Batch(cmds...)
}

// handleWindowResize processes window resize messages
func (m StorageModel) handleWindowResize(msg tea.WindowSizeMsg) (tea.Model, tea.Cmd) {
	m.width = msg.Width
	m.height = msg.Height
	m.statusBar.SetWidth(msg.Width)
	return m, nil
}

// handleKeyboardInput processes keyboard input messages
func (m StorageModel) handleKeyboardInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "r":
		return m.handleRefreshKey()

	case "tab":
		return m.handleTabKey()

	case "up", "k":
		return m.handleUpKey()

	case "down", "j":
		return m.handleDownKey()

	case "m":
		return m.handleMountKey()

	case "u":
		return m.handleUnmountKey()

	case "enter":
		return m.handleEnterKey()

	case "q", "esc":
		return m.handleQuitKey()
	}

	return m, nil
}

// handleRefreshKey processes refresh key ('r')
func (m StorageModel) handleRefreshKey() (tea.Model, tea.Cmd) {
	m.loading = true
	m.error = ""
	return m, m.fetchStorage
}

// handleTabKey processes tab key for switching between EFS/EBS tabs
func (m StorageModel) handleTabKey() (tea.Model, tea.Cmd) {
	if !m.showMountDialog {
		m.selectedTab = (m.selectedTab + 1) % 2
		m.selectedItem = 0
	}
	return m, nil
}

// handleUpKey processes up arrow/k key for navigation
func (m StorageModel) handleUpKey() (tea.Model, tea.Cmd) {
	if !m.showMountDialog && m.selectedItem > 0 {
		m.selectedItem--
	}
	return m, nil
}

// handleDownKey processes down arrow/j key for navigation
func (m StorageModel) handleDownKey() (tea.Model, tea.Cmd) {
	if !m.showMountDialog {
		maxItems := len(m.volumes)
		if m.selectedTab == 1 {
			maxItems = len(m.storage)
		}
		if m.selectedItem < maxItems-1 {
			m.selectedItem++
		}
	}
	return m, nil
}

// handleMountKey processes mount key ('m') for EFS volumes
func (m StorageModel) handleMountKey() (tea.Model, tea.Cmd) {
	if volumeName := m.getSelectedVolumeName(); volumeName != "" {
		m.mountVolumeName = volumeName
		m.showMountDialog = true
	}
	return m, nil
}

// handleUnmountKey processes unmount key ('u') for EFS volumes
func (m StorageModel) handleUnmountKey() (tea.Model, tea.Cmd) {
	if volumeName := m.getSelectedVolumeName(); volumeName != "" {
		return m, m.performUnmount(volumeName)
	}
	return m, nil
}

// getSelectedVolumeName returns the currently selected volume name, or empty string if invalid
func (m StorageModel) getSelectedVolumeName() string {
	if m.showMountDialog || m.selectedTab != 0 || len(m.volumes) == 0 {
		return ""
	}

	volumeNames := make([]string, 0, len(m.volumes))
	for name := range m.volumes {
		volumeNames = append(volumeNames, name)
	}

	if m.selectedItem >= len(volumeNames) {
		return ""
	}

	return volumeNames[m.selectedItem]
}

// handleEnterKey processes enter key for confirming mount dialog
func (m StorageModel) handleEnterKey() (tea.Model, tea.Cmd) {
	if m.showMountDialog && m.mountVolumeName != "" && m.mountInstanceName != "" {
		m.showMountDialog = false
		return m, m.performMount(m.mountVolumeName, m.mountInstanceName, m.mountPoint)
	}
	return m, nil
}

// handleQuitKey processes quit keys ('q', 'esc')
func (m StorageModel) handleQuitKey() (tea.Model, tea.Cmd) {
	if m.showMountDialog {
		m.showMountDialog = false
		m.mountVolumeName = ""
		m.mountInstanceName = ""
	} else {
		return m, tea.Quit
	}
	return m, nil
}

// handleRefresh processes refresh messages
func (m StorageModel) handleRefresh() (tea.Model, tea.Cmd) {
	m.loading = true
	m.error = ""
	return m, m.fetchStorage
}

// handleError processes error messages
func (m StorageModel) handleError(err error) (tea.Model, tea.Cmd) {
	m.loading = false
	m.error = err.Error()
	m.statusBar.SetStatus(fmt.Sprintf("Error: %s", m.error), components.StatusError)
	return m, nil
}

// handleStorageData processes storage data messages
func (m StorageModel) handleStorageData(msg StorageDataMsg) (tea.Model, tea.Cmd) {
	m.loading = false
	m.volumes = msg.Volumes
	m.storage = msg.Storage
	m.instances = msg.Instances

	volumeCount := len(m.volumes)
	storageCount := len(m.storage)
	instanceCount := len(m.instances)
	m.statusBar.SetStatus(fmt.Sprintf("Loaded %d EFS volumes, %d EBS volumes, and %d instances", volumeCount, storageCount, instanceCount), components.StatusSuccess)
	return m, nil
}

// handleMountAction processes mount/unmount action messages
func (m StorageModel) handleMountAction(msg MountActionMsg) (tea.Model, tea.Cmd) {
	if msg.Success {
		m.statusBar.SetStatus(fmt.Sprintf("%s %s to %s successful", msg.Action, msg.Volume, msg.Instance), components.StatusSuccess)
		// Refresh data to show updated mount status
		return m, m.fetchStorage
	} else {
		m.statusBar.SetStatus(fmt.Sprintf("%s failed: %s", msg.Action, msg.Message), components.StatusError)
	}
	return m, nil
}

// View renders the storage view
func (m StorageModel) View() string {
	theme := styles.CurrentTheme

	// Title section
	title := theme.Title.Render("CloudWorkstation Storage")

	// Content area
	var content string
	if m.loading {
		content = lipgloss.NewStyle().
			Width(m.width).
			Height(m.height-4). // Account for title and status bar
			Align(lipgloss.Center, lipgloss.Center).
			Render(m.spinner.View())
	} else if m.error != "" {
		content = lipgloss.NewStyle().
			Width(m.width).
			Height(m.height-4).
			Align(lipgloss.Center, lipgloss.Center).
			Render(theme.StatusError.Render("Error: " + m.error))
	} else if m.showMountDialog {
		content = m.renderMountDialog()
	} else {
		content = m.renderMainView()
	}

	// Help text based on current state
	var help string
	if m.showMountDialog {
		help = theme.Help.Render("enter: mount • esc: cancel")
	} else {
		help = theme.Help.Render("tab: switch tabs • ↑/↓: navigate • m: mount • u: unmount • r: refresh • q: quit")
	}

	// Join everything together
	return lipgloss.JoinVertical(
		lipgloss.Left,
		title,
		content,
		"",
		m.statusBar.View(),
		help,
	)
}

// renderMainView renders the main storage view with tabs
func (m StorageModel) renderMainView() string {
	theme := styles.CurrentTheme

	// Tab headers
	efsTab := "EFS Volumes"
	ebsTab := "EBS Volumes"
	if m.selectedTab == 0 {
		efsTab = theme.Tab.Active.Render(efsTab)
		ebsTab = theme.Tab.Inactive.Render(ebsTab)
	} else {
		efsTab = theme.Tab.Inactive.Render(efsTab)
		ebsTab = theme.Tab.Active.Render(ebsTab)
	}

	tabs := lipgloss.JoinHorizontal(lipgloss.Top, efsTab, "  ", ebsTab)

	// Tab content
	var tabContent string
	if m.selectedTab == 0 {
		tabContent = m.renderEFSVolumes()
	} else {
		tabContent = m.renderEBSVolumes()
	}

	// Combine tabs and content
	return lipgloss.JoinVertical(lipgloss.Left, tabs, "", tabContent)
}

// renderEFSVolumes renders the EFS volumes tab
func (m StorageModel) renderEFSVolumes() string {
	theme := styles.CurrentTheme

	if len(m.volumes) == 0 {
		return lipgloss.NewStyle().
			Padding(2).
			Render("No EFS volumes found\n\nUse CLI to create volumes:\n  cws volumes create <name>")
	}

	var lines []string
	volumeNames := make([]string, 0, len(m.volumes))
	for name := range m.volumes {
		volumeNames = append(volumeNames, name)
	}

	for i, name := range volumeNames {
		volume := m.volumes[name]
		line := fmt.Sprintf("%s - %s (%.2f GB, $%.4f/GB/month)",
			name, volume.State, float64(volume.SizeBytes)/(1024*1024*1024), volume.EstimatedCostGB)

		if i == m.selectedItem {
			line = theme.Selection.Render("> " + line)
		} else {
			line = "  " + line
		}
		lines = append(lines, line)
	}

	content := strings.Join(lines, "\n")
	content += "\n\nActions:\n  m: Mount selected volume to instance\n  u: Unmount selected volume from all instances"

	return lipgloss.NewStyle().Padding(1, 2).Render(content)
}

// renderEBSVolumes renders the EBS volumes tab
func (m StorageModel) renderEBSVolumes() string {
	theme := styles.CurrentTheme

	if len(m.storage) == 0 {
		return lipgloss.NewStyle().
			Padding(2).
			Render("No EBS volumes found\n\nUse CLI to create volumes:\n  cws ebs-volumes create <name>")
	}

	var lines []string
	storageNames := make([]string, 0, len(m.storage))
	for name := range m.storage {
		storageNames = append(storageNames, name)
	}

	for i, name := range storageNames {
		storage := m.storage[name]
		attached := "unattached"
		if storage.AttachedTo != "" {
			attached = fmt.Sprintf("attached to %s", storage.AttachedTo)
		}
		line := fmt.Sprintf("%s - %s (%d GB %s, %s, $%.4f/GB/month)",
			name, storage.State, storage.SizeGB, storage.VolumeType, attached, storage.EstimatedCostGB)

		if i == m.selectedItem {
			line = theme.Selection.Render("> " + line)
		} else {
			line = "  " + line
		}
		lines = append(lines, line)
	}

	content := strings.Join(lines, "\n")
	content += "\n\nEBS volumes are managed via CLI:\n  cws ebs-volumes attach/detach <name> <instance>"

	return lipgloss.NewStyle().Padding(1, 2).Render(content)
}

// renderMountDialog renders the mount dialog
func (m StorageModel) renderMountDialog() string {
	theme := styles.CurrentTheme

	dialog := fmt.Sprintf("Mount EFS Volume: %s\n\n", m.mountVolumeName)

	if len(m.instances) == 0 {
		dialog += "No running instances found.\nStart an instance first, then try mounting."
	} else {
		dialog += "Select instance to mount to:\n\n"

		for _, instance := range m.instances {
			line := fmt.Sprintf("  %s (%s)", instance.Name, instance.State)
			if instance.State == "running" {
				line += " ✓"
			}
			dialog += line + "\n"
		}

		if len(m.instances) > 0 && m.mountInstanceName == "" {
			// Auto-select first running instance
			for _, instance := range m.instances {
				if instance.State == "running" {
					m.mountInstanceName = instance.Name
					break
				}
			}
			if m.mountInstanceName == "" {
				m.mountInstanceName = m.instances[0].Name // Fallback to first instance
			}
		}

		dialog += fmt.Sprintf("\nSelected: %s", m.mountInstanceName)
		dialog += fmt.Sprintf("\nMount point: %s", m.mountPoint)
		dialog += "\n\nPress Enter to mount, Esc to cancel"
	}

	return lipgloss.NewStyle().
		Width(m.width-4).
		Height(m.height-6).
		Padding(2).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(theme.Border).
		Align(lipgloss.Center, lipgloss.Center).
		Render(dialog)
}

// performMount mounts a volume to an instance
func (m StorageModel) performMount(volumeName, instanceName, mountPoint string) tea.Cmd {
	return func() tea.Msg {
		err := m.apiClient.MountVolume(context.Background(), volumeName, instanceName, mountPoint)
		if err != nil {
			return MountActionMsg{
				Success:  false,
				Message:  err.Error(),
				Action:   "Mount",
				Volume:   volumeName,
				Instance: instanceName,
			}
		}
		return MountActionMsg{
			Success:  true,
			Message:  "Successfully mounted volume",
			Action:   "Mount",
			Volume:   volumeName,
			Instance: instanceName,
		}
	}
}

// performUnmount unmounts a volume from all instances
func (m StorageModel) performUnmount(volumeName string) tea.Cmd {
	return func() tea.Msg {
		// For simplicity, unmount from all instances that might have this volume
		// In a real implementation, we'd want to be more specific
		for _, instance := range m.instances {
			err := m.apiClient.UnmountVolume(context.Background(), volumeName, instance.Name)
			if err != nil {
				return MountActionMsg{
					Success:  false,
					Message:  err.Error(),
					Action:   "Unmount",
					Volume:   volumeName,
					Instance: instance.Name,
				}
			}
		}
		return MountActionMsg{
			Success:  true,
			Message:  "Successfully unmounted volume",
			Action:   "Unmount",
			Volume:   volumeName,
			Instance: "all",
		}
	}
}
