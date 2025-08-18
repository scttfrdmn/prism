package models

import (
	"fmt"
	"strings"
	
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	
	"github.com/scttfrdmn/cloudworkstation/internal/tui/components"
	"github.com/scttfrdmn/cloudworkstation/pkg/types"
)

// Command represents a UI command (Command Pattern - SOLID)
type Command interface {
	Execute(model tea.Model, msg tea.Msg) (tea.Model, tea.Cmd)
	CanExecute(msg tea.Msg) bool
}

// CommandDispatcher manages and executes commands (Single Responsibility)
type CommandDispatcher struct {
	commands []Command
}

// NewCommandDispatcher creates a new command dispatcher
func NewCommandDispatcher() *CommandDispatcher {
	return &CommandDispatcher{
		commands: make([]Command, 0),
	}
}

// RegisterCommand registers a new command (Open/Closed Principle)
func (cd *CommandDispatcher) RegisterCommand(cmd Command) {
	cd.commands = append(cd.commands, cmd)
}

// Dispatch finds and executes the appropriate command
func (cd *CommandDispatcher) Dispatch(msg tea.Msg, model tea.Model) (tea.Model, tea.Cmd) {
	for _, cmd := range cd.commands {
		if cmd.CanExecute(msg) {
			return cmd.Execute(model, msg)
		}
	}
	
	// Default: return model unchanged
	return model, nil
}

// WindowResizeCommand handles window resize events
type WindowResizeCommand struct{}

func (w WindowResizeCommand) CanExecute(msg tea.Msg) bool {
	_, ok := msg.(tea.WindowSizeMsg)
	return ok
}

func (w WindowResizeCommand) Execute(model tea.Model, msg tea.Msg) (tea.Model, tea.Cmd) {
	if repoModel, ok := model.(RepositoriesModel); ok {
		if resizeMsg, ok := msg.(tea.WindowSizeMsg); ok {
			return w.executeResize(repoModel, resizeMsg), nil
		}
	}
	return model, nil
}

func (w WindowResizeCommand) executeResize(m RepositoriesModel, msg tea.WindowSizeMsg) RepositoriesModel {
	m.width = msg.Width
	m.height = msg.Height
	contentHeight := m.height - 2 - 3
	m.repoList.SetSize(m.width-2, contentHeight)
	return m
}

// KeyCommand handles keyboard events with specific keys
type KeyCommand struct {
	Key     string
	Handler func(tea.Model) (tea.Model, tea.Cmd)
}

func (k KeyCommand) CanExecute(msg tea.Msg) bool {
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		return keyMsg.String() == k.Key
	}
	return false
}

func (k KeyCommand) Execute(model tea.Model, msg tea.Msg) (tea.Model, tea.Cmd) {
	return k.Handler(model)
}

// NewKeyCommand creates a new key command
func NewKeyCommand(key string, handler func(tea.Model) (tea.Model, tea.Cmd)) *KeyCommand {
	return &KeyCommand{
		Key:     key,
		Handler: handler,
	}
}

// RepositoryAddCommand handles adding new repositories
type RepositoryAddCommand struct{}

func (r RepositoryAddCommand) CanExecute(msg tea.Msg) bool {
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		return keyMsg.String() == "a"
	}
	return false
}

func (r RepositoryAddCommand) Execute(model tea.Model, msg tea.Msg) (tea.Model, tea.Cmd) {
	if repoModel, ok := model.(RepositoriesModel); ok && repoModel.mode == "view" {
		repoModel.mode = "add"
		repoModel.nameInput.Focus()
		repoModel.urlInput.Reset()
		repoModel.priorityInput.SetValue("50")
		repoModel.enabledInput.SetValue("true")
		repoModel.focusIndex = 0
		repoModel.statusBar.SetStatus("Enter: Submit • Esc: Cancel • Tab: Next Field", components.StatusInfo)
		return repoModel, nil
	}
	return model, nil
}

// RepositoryEditCommand handles editing repositories
type RepositoryEditCommand struct{}

func (r RepositoryEditCommand) CanExecute(msg tea.Msg) bool {
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		return keyMsg.String() == "e"
	}
	return false
}

func (r RepositoryEditCommand) Execute(model tea.Model, msg tea.Msg) (tea.Model, tea.Cmd) {
	if repoModel, ok := model.(RepositoriesModel); ok && repoModel.mode == "view" {
		if repoModel.repoList.SelectedItem() != nil {
			item := repoModel.repoList.SelectedItem().(RepositoryItem)
			repoModel.mode = "edit"
			repoModel.selected = item.Name
			repoModel.nameInput.SetValue(item.Name)
			repoModel.nameInput.Focus()
			repoModel.urlInput.SetValue(item.URL)
			repoModel.priorityInput.SetValue(fmt.Sprintf("%d", item.Priority))
			repoModel.enabledInput.SetValue(fmt.Sprintf("%t", item.Enabled))
			repoModel.focusIndex = 0
			repoModel.statusBar.SetStatus("Enter: Submit • Esc: Cancel • Tab: Next Field", components.StatusInfo)
		}
		return repoModel, nil
	}
	return model, nil
}

// RepositoryDeleteCommand handles deleting repositories  
type RepositoryDeleteCommand struct{}

func (r RepositoryDeleteCommand) CanExecute(msg tea.Msg) bool {
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		return keyMsg.String() == "d"
	}
	return false
}

func (r RepositoryDeleteCommand) Execute(model tea.Model, msg tea.Msg) (tea.Model, tea.Cmd) {
	if repoModel, ok := model.(RepositoriesModel); ok && repoModel.mode == "view" {
		if repoModel.repoList.SelectedItem() != nil {
			item := repoModel.repoList.SelectedItem().(RepositoryItem)
			
			var updatedRepos []types.TemplateRepository
			for _, repo := range repoModel.repos {
				if repo.Name != item.Name {
					updatedRepos = append(updatedRepos, repo)
				}
			}
			repoModel.repos = updatedRepos
			repoModel.refreshRepositoryList()
			repoModel.statusBar.SetStatus("Repository deleted: "+item.Name, components.StatusSuccess)
		}
		return repoModel, nil
	}
	return model, nil
}

// RepositoryRefreshCommand handles repository refresh
type RepositoryRefreshCommand struct{}

func (r RepositoryRefreshCommand) CanExecute(msg tea.Msg) bool {
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		return keyMsg.String() == "r"
	}
	return false
}

func (r RepositoryRefreshCommand) Execute(model tea.Model, msg tea.Msg) (tea.Model, tea.Cmd) {
	if repoModel, ok := model.(RepositoriesModel); ok && repoModel.mode == "view" {
		repoModel.loading = true
		repoModel.statusBar.SetStatus("Refreshing repository list...", components.StatusInfo)
		return repoModel, tea.Batch(
			func() tea.Msg { return nil },
			repoModel.fetchRepositories,
		)
	}
	return model, nil
}

// RepositorySyncCommand handles repository sync
type RepositorySyncCommand struct{}

func (r RepositorySyncCommand) CanExecute(msg tea.Msg) bool {
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		return keyMsg.String() == "s"
	}
	return false
}

func (r RepositorySyncCommand) Execute(model tea.Model, msg tea.Msg) (tea.Model, tea.Cmd) {
	if repoModel, ok := model.(RepositoriesModel); ok && repoModel.mode == "view" {
		repoModel.loading = true
		repoModel.statusBar.SetStatus("Syncing repositories...", components.StatusInfo)
		return repoModel, tea.Batch(
			func() tea.Msg { return nil },
			repoModel.syncRepositories,
		)
	}
	return model, nil
}

// FormCancelCommand handles form cancellation
type FormCancelCommand struct{}

func (f FormCancelCommand) CanExecute(msg tea.Msg) bool {
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		return keyMsg.String() == "esc"
	}
	return false
}

func (f FormCancelCommand) Execute(model tea.Model, msg tea.Msg) (tea.Model, tea.Cmd) {
	if repoModel, ok := model.(RepositoriesModel); ok && (repoModel.mode == "add" || repoModel.mode == "edit") {
		repoModel.mode = "view"
		repoModel.statusBar.SetStatus("↑/↓: Navigate • Enter: Select • a: Add • e: Edit • r: Refresh • d: Delete", components.StatusInfo)
		return repoModel, nil
	}
	return model, nil
}

// FormSubmitCommand handles form submission
type FormSubmitCommand struct{}

func (f FormSubmitCommand) CanExecute(msg tea.Msg) bool {
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		return keyMsg.String() == "enter"
	}
	return false
}

func (f FormSubmitCommand) Execute(model tea.Model, msg tea.Msg) (tea.Model, tea.Cmd) {
	if repoModel, ok := model.(RepositoriesModel); ok && (repoModel.mode == "add" || repoModel.mode == "edit") {
		return f.executeFormSubmit(repoModel)
	}
	return model, nil
}

func (f FormSubmitCommand) executeFormSubmit(m RepositoriesModel) (tea.Model, tea.Cmd) {
	newRepo := types.TemplateRepository{
		Name:     m.nameInput.Value(),
		URL:      m.urlInput.Value(),
		Priority: 50,
		Enabled:  true,
	}
	
	// Parse priority if provided
	if m.priorityInput.Value() != "" {
		var priority int
		_, err := fmt.Sscanf(m.priorityInput.Value(), "%d", &priority)
		if err == nil {
			newRepo.Priority = priority
		}
	}
	
	// Parse enabled if provided
	if strings.ToLower(m.enabledInput.Value()) == "false" {
		newRepo.Enabled = false
	}
	
	if m.mode == "add" {
		m.repos = append(m.repos, newRepo)
		m.statusBar.SetStatus("Repository added: "+newRepo.Name, components.StatusSuccess)
	} else {
		for i, repo := range m.repos {
			if repo.Name == m.selected {
				m.repos[i] = newRepo
				break
			}
		}
		m.statusBar.SetStatus("Repository updated: "+newRepo.Name, components.StatusSuccess)
	}
	
	m.refreshRepositoryList()
	m.mode = "view"
	m.statusBar.SetStatus("↑/↓: Navigate • Enter: Select • a: Add • e: Edit • r: Refresh • d: Delete", components.StatusInfo)
	return m, nil
}

// FormNavigationCommand handles tab navigation in forms
type FormNavigationCommand struct{}

func (f FormNavigationCommand) CanExecute(msg tea.Msg) bool {
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		return keyMsg.String() == "tab" || keyMsg.String() == "shift+tab"
	}
	return false
}

func (f FormNavigationCommand) Execute(model tea.Model, msg tea.Msg) (tea.Model, tea.Cmd) {
	if repoModel, ok := model.(RepositoriesModel); ok && (repoModel.mode == "add" || repoModel.mode == "edit") {
		if keyMsg, ok := msg.(tea.KeyMsg); ok {
			return f.executeNavigation(repoModel, keyMsg)
		}
	}
	return model, nil
}

func (f FormNavigationCommand) executeNavigation(m RepositoriesModel, keyMsg tea.KeyMsg) (tea.Model, tea.Cmd) {
	inputs := []textinput.Model{
		m.nameInput,
		m.urlInput,
		m.priorityInput,
		m.enabledInput,
	}
	
	// Determine direction
	if keyMsg.String() == "tab" {
		m.focusIndex = (m.focusIndex + 1) % len(inputs)
	} else {
		m.focusIndex = (m.focusIndex - 1 + len(inputs)) % len(inputs)
	}
	
	// Update focus
	for i := 0; i < len(inputs); i++ {
		if i == m.focusIndex {
			inputs[i].Focus()
		} else {
			inputs[i].Blur()
		}
	}
	
	m.nameInput = inputs[0]
	m.urlInput = inputs[1]
	m.priorityInput = inputs[2]
	m.enabledInput = inputs[3]
	
	return m, nil
}

// FormInputCommand handles input updates
type FormInputCommand struct{}

func (f FormInputCommand) CanExecute(msg tea.Msg) bool {
	_, ok := msg.(tea.KeyMsg)
	return ok
}

func (f FormInputCommand) Execute(model tea.Model, msg tea.Msg) (tea.Model, tea.Cmd) {
	if repoModel, ok := model.(RepositoriesModel); ok && (repoModel.mode == "add" || repoModel.mode == "edit") {
		if keyMsg, ok := msg.(tea.KeyMsg); ok {
			return f.executeInputUpdate(repoModel, keyMsg)
		}
	}
	return model, nil
}

func (f FormInputCommand) executeInputUpdate(m RepositoriesModel, keyMsg tea.KeyMsg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	switch m.focusIndex {
	case 0:
		m.nameInput, cmd = m.nameInput.Update(keyMsg)
	case 1:
		m.urlInput, cmd = m.urlInput.Update(keyMsg)
	case 2:
		m.priorityInput, cmd = m.priorityInput.Update(keyMsg)
	case 3:
		m.enabledInput, cmd = m.enabledInput.Update(keyMsg)
	}
	return m, cmd
}

// Instance Management Commands for InstancesModel

// InstanceWindowResizeCommand handles window resize for instances view
type InstanceWindowResizeCommand struct{}

func (i InstanceWindowResizeCommand) CanExecute(msg tea.Msg) bool {
	_, ok := msg.(tea.WindowSizeMsg)
	return ok
}

func (i InstanceWindowResizeCommand) Execute(model tea.Model, msg tea.Msg) (tea.Model, tea.Cmd) {
	if instanceModel, ok := model.(InstancesModel); ok {
		if resizeMsg, ok := msg.(tea.WindowSizeMsg); ok {
			instanceModel.width = resizeMsg.Width
			instanceModel.height = resizeMsg.Height
			instanceModel.statusBar.SetWidth(resizeMsg.Width)
			
			// Update table dimensions
			tableHeight := instanceModel.height - 6 // Account for title, help, and status
			instanceModel.instancesTable.SetSize(instanceModel.width-4, tableHeight)
			
			return instanceModel, nil
		}
	}
	return model, nil
}

// InstanceRefreshCommand handles instance refresh
type InstanceRefreshCommand struct{}

func (i InstanceRefreshCommand) CanExecute(msg tea.Msg) bool {
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		return keyMsg.String() == "r"
	}
	return false
}

func (i InstanceRefreshCommand) Execute(model tea.Model, msg tea.Msg) (tea.Model, tea.Cmd) {
	if instanceModel, ok := model.(InstancesModel); ok && !instanceModel.showingActions {
		instanceModel.loading = true
		instanceModel.error = ""
		return instanceModel, instanceModel.fetchInstances
	}
	return model, nil
}

// InstanceActionsCommand handles showing action menu
type InstanceActionsCommand struct{}

func (i InstanceActionsCommand) CanExecute(msg tea.Msg) bool {
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		return keyMsg.String() == "a"
	}
	return false
}

func (i InstanceActionsCommand) Execute(model tea.Model, msg tea.Msg) (tea.Model, tea.Cmd) {
	if instanceModel, ok := model.(InstancesModel); ok && !instanceModel.showingActions && len(instanceModel.instances) > 0 {
		instanceModel.showingActions = true
		return instanceModel, nil
	}
	return model, nil
}

// InstanceConnectionCommand handles showing connection info
type InstanceConnectionCommand struct{}

func (i InstanceConnectionCommand) CanExecute(msg tea.Msg) bool {
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		return keyMsg.String() == "c"
	}
	return false
}

func (i InstanceConnectionCommand) Execute(model tea.Model, msg tea.Msg) (tea.Model, tea.Cmd) {
	if instanceModel, ok := model.(InstancesModel); ok && !instanceModel.showingActions {
		if len(instanceModel.instances) > 0 && instanceModel.selected < len(instanceModel.instances) {
			instance := instanceModel.instances[instanceModel.selected]
			instanceModel.actionMessage = fmt.Sprintf("SSH: ssh %s@%s", "ubuntu", instance.PublicIP)
			return instanceModel, nil
		}
	}
	return model, nil
}

// InstanceActionExecuteCommand handles action execution (s, p, d in action mode)
type InstanceActionExecuteCommand struct{}

func (i InstanceActionExecuteCommand) CanExecute(msg tea.Msg) bool {
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		return keyMsg.String() == "s" || keyMsg.String() == "p" || keyMsg.String() == "d"
	}
	return false
}

func (i InstanceActionExecuteCommand) Execute(model tea.Model, msg tea.Msg) (tea.Model, tea.Cmd) {
	if instanceModel, ok := model.(InstancesModel); ok && instanceModel.showingActions {
		if keyMsg, ok := msg.(tea.KeyMsg); ok {
			instanceModel.showingActions = false
			switch keyMsg.String() {
			case "s":
				return instanceModel, instanceModel.performAction("start")
			case "p":
				return instanceModel, instanceModel.performAction("stop") 
			case "d":
				return instanceModel, instanceModel.performAction("delete")
			}
		}
	}
	return model, nil
}

// InstanceActionCancelCommand handles canceling action menu
type InstanceActionCancelCommand struct{}

func (i InstanceActionCancelCommand) CanExecute(msg tea.Msg) bool {
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		return keyMsg.String() == "esc" || keyMsg.String() == "q"
	}
	return false
}

func (i InstanceActionCancelCommand) Execute(model tea.Model, msg tea.Msg) (tea.Model, tea.Cmd) {
	if instanceModel, ok := model.(InstancesModel); ok && instanceModel.showingActions {
		instanceModel.showingActions = false
		return instanceModel, nil
	}
	return model, nil
}

// InstanceTableNavigationCommand handles table navigation
type InstanceTableNavigationCommand struct{}

func (i InstanceTableNavigationCommand) CanExecute(msg tea.Msg) bool {
	// Handle any key message that's not a specific command
	_, ok := msg.(tea.KeyMsg)
	return ok
}

func (i InstanceTableNavigationCommand) Execute(model tea.Model, msg tea.Msg) (tea.Model, tea.Cmd) {
	if instanceModel, ok := model.(InstancesModel); ok && !instanceModel.loading && !instanceModel.showingActions && len(instanceModel.instances) > 0 {
		var cmd tea.Cmd
		instanceModel.instancesTable, cmd = instanceModel.instancesTable.Update(msg)
		instanceModel.updateSelectedIndex()
		return instanceModel, cmd
	}
	return model, nil
}