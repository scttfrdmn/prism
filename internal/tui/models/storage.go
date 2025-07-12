package models

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/scttfrdmn/cloudworkstation/internal/tui/components"
	"github.com/scttfrdmn/cloudworkstation/internal/tui/styles"
	"github.com/scttfrdmn/cloudworkstation/pkg/api"
	"github.com/scttfrdmn/cloudworkstation/pkg/types"
)

// TabChangeMsg is a temporary type until we integrate with bubbles/tab
type TabChangeMsg struct {
	Index int
}

// StorageType represents the different types of storage
type StorageType int

const (
	// EFSStorage is for EFS volumes
	EFSStorage StorageType = iota
	// EBSStorage is for EBS volumes
	EBSStorage
)

// EFSVolumeItem represents an EFS volume in the list
type EFSVolumeItem struct {
	name        string
	id          string
	state       string
	region      string
	sizeGB      float64
	costPerGB   float64
}

// FilterValue returns the value to filter on in the list
func (v EFSVolumeItem) FilterValue() string { return v.name }

// Title returns the name of the volume
func (v EFSVolumeItem) Title() string { return v.name }

// Description returns a short description of the volume
func (v EFSVolumeItem) Description() string { 
	return fmt.Sprintf("%s • %s • %.1f GB", v.id, v.state, v.sizeGB) 
}

// EBSVolumeItem represents an EBS volume in the list
type EBSVolumeItem struct {
	name        string
	id          string
	state       string
	region      string
	sizeGB      int32
	volumeType  string
	attachedTo  string
}

// FilterValue returns the value to filter on in the list
func (v EBSVolumeItem) FilterValue() string { return v.name }

// Title returns the name of the volume
func (v EBSVolumeItem) Title() string { return v.name }

// Description returns a short description of the volume
func (v EBSVolumeItem) Description() string { 
	attached := ""
	if v.attachedTo != "" {
		attached = fmt.Sprintf(" • Attached to: %s", v.attachedTo)
	}
	return fmt.Sprintf("%s • %s • %d GB%s", v.volumeType, v.state, v.sizeGB, attached) 
}

// Storage-specific message types

// Focus represents the component that has focus
type Focus int

const (
	// ListFocus means the list view has focus
	ListFocus Focus = iota
	// DetailFocus means the detail view has focus
	DetailFocus
)

// StorageModel represents the storage management view
type StorageModel struct {
	apiClient  api.CloudWorkstationAPI
	tabBar     components.TabBar
	efsVolumes list.Model
	ebsVolumes list.Model
	detailView viewport.Model
	statusBar  components.StatusBar
	spinner    components.Spinner
	search     components.Search
	width      int
	height     int
	loading    bool
	error      string
	
	// Data
	volumes       map[string]types.EFSVolume
	storage       map[string]types.EBSVolume
	selected      string
	activeType    StorageType
	focused       Focus
	searchActive  bool
	allEfsItems   []list.Item
	allEbsItems   []list.Item
	filterEfsItems []list.Item
	filterEbsItems []list.Item
}

// NewStorageModel creates a new storage model
func NewStorageModel(apiClient api.CloudWorkstationAPI) StorageModel {
	theme := styles.CurrentTheme
	
	// Set up tabs
	tabBar := components.NewTabBar([]string{"EFS Volumes", "EBS Storage"}, 0)
	
	// Set up EFS volume list
	efsVolumes := list.New([]list.Item{}, list.NewDefaultDelegate(), 0, 0)
	efsVolumes.Title = "EFS Volumes"
	efsVolumes.Styles.Title = theme.Title
	efsVolumes.Styles.PaginationStyle = theme.Pagination
	efsVolumes.Styles.HelpStyle = theme.Help
	
	// Set up EBS volume list
	ebsVolumes := list.New([]list.Item{}, list.NewDefaultDelegate(), 0, 0)
	ebsVolumes.Title = "EBS Storage Volumes"
	ebsVolumes.Styles.Title = theme.Title
	ebsVolumes.Styles.PaginationStyle = theme.Pagination
	ebsVolumes.Styles.HelpStyle = theme.Help
	
	// Set up detail view for volume information
	detailView := viewport.New(0, 0)
	detailView.Style = theme.Panel
	
	// Create status bar and spinner
	statusBar := components.NewStatusBar("", "")
	spinner := components.NewSpinner("Loading storage information...")
	
	// Create search component
	search := components.NewSearch()
	
	return StorageModel{
		apiClient:     apiClient,
		tabBar:        tabBar,
		efsVolumes:    efsVolumes,
		ebsVolumes:    ebsVolumes,
		detailView:    detailView,
		statusBar:     statusBar,
		spinner:       spinner,
		search:        search,
		width:         80,
		height:        24,
		loading:       true,
		volumes:       make(map[string]types.EFSVolume),
		storage:       make(map[string]types.EBSVolume),
		activeType:    EFSStorage,
		focused:       ListFocus,
		searchActive:  false,
		allEfsItems:   []list.Item{},
		allEbsItems:   []list.Item{},
		filterEfsItems: []list.Item{},
		filterEbsItems: []list.Item{},
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
	// Get both EFS volumes and EBS storage in parallel
	efsVolumesCh := make(chan map[string]types.EFSVolume)
	ebsVolumesCh := make(chan map[string]types.EBSVolume)
	efsErrCh := make(chan error)
	ebsErrCh := make(chan error)
	
	// Fetch EFS volumes in goroutine
	go func() {
		volumes, err := m.apiClient.ListVolumes()
		if err != nil {
			efsErrCh <- fmt.Errorf("failed to list EFS volumes: %w", err)
			return
		}
		efsVolumesCh <- volumes
	}()
	
	// Fetch EBS volumes in goroutine
	go func() {
		storage, err := m.apiClient.ListStorage()
		if err != nil {
			ebsErrCh <- fmt.Errorf("failed to list EBS volumes: %w", err)
			return
		}
		ebsVolumesCh <- storage
	}()
	
	// Wait for both results or errors
	var efsVolumes map[string]types.EFSVolume
	var ebsVolumes map[string]types.EBSVolume
	var efsErr, ebsErr error
	
	select {
	case efsVolumes = <-efsVolumesCh:
	case efsErr = <-efsErrCh:
	}
	
	select {
	case ebsVolumes = <-ebsVolumesCh:
	case ebsErr = <-ebsErrCh:
	}
	
	// Return an error if either failed
	if efsErr != nil {
		return efsErr
	}
	if ebsErr != nil {
		return ebsErr
	}
	
	// Return combined storage results
	return struct {
		EFSVolumes map[string]types.EFSVolume
		EBSVolumes map[string]types.EBSVolume
	}{
		EFSVolumes: efsVolumes,
		EBSVolumes: ebsVolumes,
	}
}

// Update handles messages and updates the model
func (m StorageModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.statusBar.SetWidth(msg.Width)
		
		// Update list and detail view dimensions
		leftWidth := m.width / 3
		rightWidth := m.width - leftWidth - 2 // Account for separator
		contentHeight := m.height - 6 // Account for title, tabs and status
		
		m.efsVolumes.SetSize(leftWidth, contentHeight)
		m.ebsVolumes.SetSize(leftWidth, contentHeight)
		m.detailView.Width = rightWidth
		m.detailView.Height = contentHeight
		
		return m, nil
		
	case tea.KeyMsg:
		// Handle key presses
		switch msg.String() {
		case "r":
			m.loading = true
			m.error = ""
			return m, m.fetchStorage
			
		case "q", "esc":
			return m, tea.Quit
			
		case "tab":
			// Switch between EFS and EBS tabs
			cmd := m.tabBar.Next()
			m.activeType = StorageType(m.tabBar.ActiveTab())
			return m, cmd
			
		case "shift+tab":
			// Switch between EFS and EBS tabs (reverse)
			cmd := m.tabBar.Prev()
			m.activeType = StorageType(m.tabBar.ActiveTab())
			return m, cmd
			
		case "right", "l":
			if m.focused == ListFocus {
				m.focused = DetailFocus
				return m, nil
			}
			
		case "left", "h":
			if m.focused == DetailFocus {
				m.focused = ListFocus
				return m, nil
			}
		}
		
		// Only process list inputs when not loading
		if !m.loading {
			var cmd tea.Cmd
			
			// Update active list based on tab
			if m.activeType == EFSStorage {
				m.efsVolumes, cmd = m.efsVolumes.Update(msg)
				cmds = append(cmds, cmd)
				
				// Handle selection changes
				if i, ok := m.efsVolumes.SelectedItem().(EFSVolumeItem); ok {
					if i.name != m.selected || m.activeType == EFSStorage {
						m.selected = i.name
						m.updateDetailView()
					}
				}
			} else {
				m.ebsVolumes, cmd = m.ebsVolumes.Update(msg)
				cmds = append(cmds, cmd)
				
				// Handle selection changes
				if i, ok := m.ebsVolumes.SelectedItem().(EBSVolumeItem); ok {
					if i.name != m.selected || m.activeType == EBSStorage {
						m.selected = i.name
						m.updateDetailView()
					}
				}
			}
			
			// Update detail view on scroll
			m.detailView, cmd = m.detailView.Update(msg)
			cmds = append(cmds, cmd)
		}
		
	case RefreshMsg:
		m.loading = true
		m.error = ""
		return m, m.fetchStorage
		
	case error:
		m.loading = false
		m.error = msg.Error()
		m.statusBar.SetStatus(fmt.Sprintf("Error: %s", m.error), components.StatusError)
		
	case struct {
		EFSVolumes map[string]types.EFSVolume
		EBSVolumes map[string]types.EBSVolume
	}:
		m.loading = false
		m.volumes = msg.EFSVolumes
		m.storage = msg.EBSVolumes
		
		// Update EFS volume list items
		var efsItems []list.Item
		for name, volume := range m.volumes {
			sizeGB := float64(volume.SizeBytes) / (1024 * 1024 * 1024)
			if sizeGB < 0.1 {
				sizeGB = 0.1 // Minimum display size
			}
			
			efsItems = append(efsItems, EFSVolumeItem{
				name:      name,
				id:        volume.FileSystemId,
				state:     volume.State,
				region:    volume.Region,
				sizeGB:    sizeGB,
				costPerGB: volume.EstimatedCostGB,
			})
		}
		m.efsVolumes.SetItems(efsItems)
		
		// Update EBS volume list items
		var ebsItems []list.Item
		for name, volume := range m.storage {
			ebsItems = append(ebsItems, EBSVolumeItem{
				name:       name,
				id:         volume.VolumeID,
				state:      volume.State,
				region:     volume.Region,
				sizeGB:     volume.SizeGB,
				volumeType: volume.VolumeType,
				attachedTo: volume.AttachedTo,
			})
		}
		m.ebsVolumes.SetItems(ebsItems)
		
		m.statusBar.SetStatus("Storage information loaded", components.StatusSuccess)
		
		// Select first item if available
		if m.activeType == EFSStorage && len(efsItems) > 0 {
			m.selected = efsItems[0].(EFSVolumeItem).name
			m.updateDetailView()
		} else if m.activeType == EBSStorage && len(ebsItems) > 0 {
			m.selected = ebsItems[0].(EBSVolumeItem).name
			m.updateDetailView()
		}
		
	case TabChangeMsg:
		// Handle tab changes
		m.activeType = StorageType(m.tabBar.ActiveTab())
		
		// Reset selected item
		m.selected = ""
		
		// Select first item in the active list
		if m.activeType == EFSStorage && m.efsVolumes.Items() != nil && len(m.efsVolumes.Items()) > 0 {
			m.selected = m.efsVolumes.Items()[0].(EFSVolumeItem).name
			m.updateDetailView()
		} else if m.activeType == EBSStorage && m.ebsVolumes.Items() != nil && len(m.ebsVolumes.Items()) > 0 {
			m.selected = m.ebsVolumes.Items()[0].(EBSVolumeItem).name
			m.updateDetailView()
		}
	}
	
	// Update components
	if m.loading {
		var spinnerCmd tea.Cmd
		m.spinner, spinnerCmd = m.spinner.Update(msg)
		cmds = append(cmds, spinnerCmd)
	}
	
	return m, tea.Batch(cmds...)
}

// updateDetailView updates the content of the detail view with the selected storage
func (m *StorageModel) updateDetailView() {
	theme := styles.CurrentTheme
	
	if m.selected == "" {
		return
	}
	
	var content strings.Builder
	
	if m.activeType == EFSStorage {
		// Show EFS volume details
		if volume, ok := m.volumes[m.selected]; ok {
			sizeGB := float64(volume.SizeBytes) / (1024 * 1024 * 1024)
			if sizeGB < 0.1 {
				sizeGB = 0.1 // Minimum display size
			}
			monthlyCost := sizeGB * volume.EstimatedCostGB
			
			content.WriteString(theme.SectionTitle.Render(m.selected) + "\n\n")
			content.WriteString(theme.SubTitle.Render("EFS Volume Details:") + "\n")
			content.WriteString(fmt.Sprintf("File System ID: %s\n", volume.FileSystemId))
			content.WriteString(fmt.Sprintf("State: %s\n", strings.ToUpper(volume.State)))
			content.WriteString(fmt.Sprintf("Region: %s\n", volume.Region))
			content.WriteString(fmt.Sprintf("Performance Mode: %s\n", volume.PerformanceMode))
			content.WriteString(fmt.Sprintf("Throughput Mode: %s\n\n", volume.ThroughputMode))
			
			content.WriteString(theme.SubTitle.Render("Storage Information:") + "\n")
			content.WriteString(fmt.Sprintf("Size: %.1f GB\n", sizeGB))
			content.WriteString(fmt.Sprintf("Cost per GB: $%.2f/month\n", volume.EstimatedCostGB))
			content.WriteString(fmt.Sprintf("Estimated Monthly Cost: $%.2f\n\n", monthlyCost))
			
			content.WriteString(theme.SubTitle.Render("Mount Information:") + "\n")
			content.WriteString(fmt.Sprintf("Mount Targets: %d\n", len(volume.MountTargets)))
			content.WriteString("Mount Command:\n")
			content.WriteString(fmt.Sprintf("  sudo mount -t nfs4 %s.efs.%s.amazonaws.com:/ /mnt/efs\n\n", 
				volume.FileSystemId, volume.Region))
			
			content.WriteString(theme.SubTitle.Render("Actions:") + "\n")
			content.WriteString("Launch with Volume:\n")
			content.WriteString(fmt.Sprintf("  cws launch <template> <name> --volume %s\n", m.selected))
		}
	} else {
		// Show EBS volume details
		if volume, ok := m.storage[m.selected]; ok {
			monthlyCost := float64(volume.SizeGB) * volume.EstimatedCostGB
			
			content.WriteString(theme.SectionTitle.Render(m.selected) + "\n\n")
			content.WriteString(theme.SubTitle.Render("EBS Volume Details:") + "\n")
			content.WriteString(fmt.Sprintf("Volume ID: %s\n", volume.VolumeID))
			content.WriteString(fmt.Sprintf("State: %s\n", strings.ToUpper(volume.State)))
			content.WriteString(fmt.Sprintf("Region: %s\n", volume.Region))
			content.WriteString(fmt.Sprintf("Volume Type: %s\n", volume.VolumeType))
			
			if volume.IOPS > 0 {
				content.WriteString(fmt.Sprintf("IOPS: %d\n", volume.IOPS))
			}
			if volume.Throughput > 0 {
				content.WriteString(fmt.Sprintf("Throughput: %d MB/s\n", volume.Throughput))
			}
			content.WriteString("\n")
			
			content.WriteString(theme.SubTitle.Render("Storage Information:") + "\n")
			content.WriteString(fmt.Sprintf("Size: %d GB\n", volume.SizeGB))
			content.WriteString(fmt.Sprintf("Cost per GB: $%.2f/month\n", volume.EstimatedCostGB))
			content.WriteString(fmt.Sprintf("Estimated Monthly Cost: $%.2f\n\n", monthlyCost))
			
			content.WriteString(theme.SubTitle.Render("Attachment:") + "\n")
			if volume.AttachedTo != "" {
				content.WriteString(fmt.Sprintf("Attached to: %s\n\n", volume.AttachedTo))
				content.WriteString("Detach Command:\n")
				content.WriteString(fmt.Sprintf("  cws storage detach %s\n\n", m.selected))
			} else {
				content.WriteString("Not attached to any instance\n\n")
				content.WriteString("Attach Command:\n")
				content.WriteString(fmt.Sprintf("  cws storage attach %s <instance>\n\n", m.selected))
			}
			
			content.WriteString(theme.SubTitle.Render("Actions:") + "\n")
			content.WriteString("Delete Volume:\n")
			content.WriteString(fmt.Sprintf("  cws storage delete %s\n", m.selected))
		}
	}
	
	m.detailView.SetContent(content.String())
	m.detailView.GotoTop()
}

// View renders the storage management view
func (m StorageModel) View() string {
	theme := styles.CurrentTheme
	
	// Title section
	title := theme.Title.Render("CloudWorkstation Storage Management")
	
	// Tab bar
	tabs := m.tabBar.View()
	
	// Content area
	var content string
	if m.loading {
		content = lipgloss.NewStyle().
			Width(m.width).
			Height(m.height - 6). // Account for title, tabs, and status bar
			Align(lipgloss.Center, lipgloss.Center).
			Render(m.spinner.View())
	} else if m.error != "" {
		content = lipgloss.NewStyle().
			Width(m.width).
			Height(m.height - 6).
			Align(lipgloss.Center, lipgloss.Center).
			Render(theme.StatusError.Render("Error: " + m.error))
	} else {
		// Split view with storage list on left and details on right
		var leftPane string
		if m.activeType == EFSStorage {
			leftPane = m.efsVolumes.View()
		} else {
			leftPane = m.ebsVolumes.View()
		}
		
		rightPane := m.detailView.View()
		
		// Add focus indicators
		leftBorder := lipgloss.NormalBorder()
		rightBorder := lipgloss.NormalBorder()
		
		if m.focused == ListFocus {
			leftBorder = lipgloss.DoubleBorder()
		} else if m.focused == DetailFocus {
			rightBorder = lipgloss.DoubleBorder()
		}
		
		leftStyle := lipgloss.NewStyle().
			Border(leftBorder).
			BorderForeground(theme.PrimaryColor).
			Padding(0, 1)
			
		rightStyle := lipgloss.NewStyle().
			Border(rightBorder).
			BorderForeground(theme.SecondaryColor).
			Padding(0, 1)
		
		leftPane = leftStyle.Render(leftPane)
		rightPane = rightStyle.Render(rightPane)
		
		separator := lipgloss.NewStyle().
			Foreground(theme.MutedColor).
			Width(1).
			Height(m.height - 6).
			Render("│")
		
		content = lipgloss.JoinHorizontal(lipgloss.Top, leftPane, separator, rightPane)
	}
	
	// Help text
	help := components.CompactHelpView(components.HelpStorage)
	
	// Join everything together
	return lipgloss.JoinVertical(
		lipgloss.Left,
		title,
		tabs,
		content,
		"",
		m.statusBar.View(),
		help,
	)
}