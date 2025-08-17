package models

import (
	"context"
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/scttfrdmn/cloudworkstation/internal/tui/api"
	"github.com/scttfrdmn/cloudworkstation/internal/tui/components"
	"github.com/scttfrdmn/cloudworkstation/internal/tui/styles"
)

// StorageModel represents the storage management view
type StorageModel struct {
	apiClient apiClient
	statusBar components.StatusBar
	spinner   components.Spinner
	width     int
	height    int
	loading   bool
	error     string
	volumes   map[string]api.VolumeResponse
	storage   map[string]api.StorageResponse
}

// NewStorageModel creates a new storage model
func NewStorageModel(apiClient apiClient) StorageModel {
	// Create status bar and spinner
	statusBar := components.NewStatusBar("CloudWorkstation Storage", "")
	spinner := components.NewSpinner("Loading storage information...")

	return StorageModel{
		apiClient: apiClient,
		statusBar: statusBar,
		spinner:   spinner,
		width:     80,
		height:    24,
		loading:   true,
		volumes:   make(map[string]api.VolumeResponse),
		storage:   make(map[string]api.StorageResponse),
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
	// Fetch both EFS volumes and EBS storage
	volumesResp, err := m.apiClient.ListVolumes(context.Background())
	if err != nil {
		return fmt.Errorf("failed to list EFS volumes: %w", err)
	}

	storageResp, err := m.apiClient.ListStorage(context.Background())
	if err != nil {
		return fmt.Errorf("failed to list EBS storage: %w", err)
	}

	return StorageDataMsg{
		Volumes: volumesResp.Volumes,
		Storage: storageResp.Storage,
	}
}

// StorageDataMsg represents storage data retrieved from the API
type StorageDataMsg struct {
	Volumes map[string]api.VolumeResponse
	Storage map[string]api.StorageResponse
}

// Update handles messages and updates the model
func (m StorageModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.statusBar.SetWidth(msg.Width)
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "r":
			m.loading = true
			m.error = ""
			return m, m.fetchStorage

		case "q", "esc":
			return m, tea.Quit
		}

	case RefreshMsg:
		m.loading = true
		m.error = ""
		return m, m.fetchStorage

	case error:
		m.loading = false
		m.error = msg.Error()
		m.statusBar.SetStatus(fmt.Sprintf("Error: %s", m.error), components.StatusError)

	case StorageDataMsg:
		m.loading = false
		m.volumes = msg.Volumes
		m.storage = msg.Storage

		volumeCount := len(m.volumes)
		storageCount := len(m.storage)
		m.statusBar.SetStatus(fmt.Sprintf("Loaded %d EFS volumes and %d EBS volumes", volumeCount, storageCount), components.StatusSuccess)
	}

	// Update components
	if m.loading {
		var spinnerCmd tea.Cmd
		m.spinner, spinnerCmd = m.spinner.Update(msg)
		cmds = append(cmds, spinnerCmd)
	}

	return m, tea.Batch(cmds...)
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
	} else {
		// EFS Volumes section
		efsContent := fmt.Sprintf("EFS Volumes (%d):\n", len(m.volumes))
		if len(m.volumes) == 0 {
			efsContent += "  No EFS volumes found\n"
		} else {
			for name, volume := range m.volumes {
				efsContent += fmt.Sprintf("  %s - %s (%.2f GB, $%.4f/GB/month)\n",
					name, volume.State, float64(volume.SizeBytes)/(1024*1024*1024), volume.EstimatedCostGB)
			}
		}

		// EBS Storage section
		ebsContent := fmt.Sprintf("\nEBS Volumes (%d):\n", len(m.storage))
		if len(m.storage) == 0 {
			ebsContent += "  No EBS volumes found\n"
		} else {
			for name, storage := range m.storage {
				attached := "unattached"
				if storage.AttachedTo != "" {
					attached = fmt.Sprintf("attached to %s", storage.AttachedTo)
				}
				ebsContent += fmt.Sprintf("  %s - %s (%d GB %s, %s, $%.4f/GB/month)\n",
					name, storage.State, storage.SizeGB, storage.VolumeType, attached, storage.EstimatedCostGB)
			}
		}

		// Commands help
		commandsContent := "\nAvailable Commands:\n"
		commandsContent += "  Create volumes and manage storage using CLI:\n"
		commandsContent += "  cws volumes create <name>        # Create EFS volume\n"
		commandsContent += "  cws ebs-volumes create <name>    # Create EBS volume\n"
		commandsContent += "  cws volumes delete <name>        # Delete volume\n"
		commandsContent += "  cws ebs-volumes delete <name>    # Delete EBS volume\n"

		// Combine content
		fullContent := efsContent + ebsContent + commandsContent

		content = lipgloss.NewStyle().
			Width(m.width-4).
			Padding(1, 2).
			Render(fullContent)
	}

	// Help text
	help := theme.Help.Render("r: refresh • q: quit")

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
