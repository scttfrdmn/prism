package workshop

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/google/uuid"
)

// Manager handles workshop lifecycle, participant management, and config templates.
type Manager struct {
	statePath string
	mutex     sync.RWMutex
	workshops map[string]*WorkshopEvent
	configs   map[string]*WorkshopConfig
}

// NewManager creates a new workshop manager.
func NewManager() (*Manager, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}

	stateDir := filepath.Join(homeDir, ".prism")
	if err := os.MkdirAll(stateDir, 0700); err != nil {
		return nil, fmt.Errorf("failed to create state directory: %w", err)
	}

	m := &Manager{
		statePath: filepath.Join(stateDir, "workshops.json"),
		workshops: make(map[string]*WorkshopEvent),
		configs:   make(map[string]*WorkshopConfig),
	}

	if err := m.load(); err != nil {
		return nil, fmt.Errorf("failed to load workshops: %w", err)
	}

	return m, nil
}

// ── Persistence ───────────────────────────────────────────────────────────────

func (m *Manager) load() error {
	data, err := os.ReadFile(m.statePath)
	if os.IsNotExist(err) {
		return nil // first run
	}
	if err != nil {
		return fmt.Errorf("failed to read workshops file: %w", err)
	}

	var state persistedState
	if err := json.Unmarshal(data, &state); err != nil {
		return fmt.Errorf("failed to parse workshops file: %w", err)
	}

	for i := range state.Workshops {
		w := state.Workshops[i]
		m.workshops[w.ID] = &w
	}
	for i := range state.Configs {
		c := state.Configs[i]
		m.configs[c.Name] = &c
	}
	return nil
}

func (m *Manager) save() error {
	ws := make([]WorkshopEvent, 0, len(m.workshops))
	for _, w := range m.workshops {
		ws = append(ws, *w)
	}
	cs := make([]WorkshopConfig, 0, len(m.configs))
	for _, c := range m.configs {
		cs = append(cs, *c)
	}

	data, err := json.MarshalIndent(persistedState{Workshops: ws, Configs: cs}, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal workshops: %w", err)
	}
	return os.WriteFile(m.statePath, data, 0600)
}

// ── CRUD ──────────────────────────────────────────────────────────────────────

// CreateWorkshop creates a new workshop and generates its join token.
func (m *Manager) CreateWorkshop(ctx context.Context, req *CreateWorkshopRequest) (*WorkshopEvent, error) {
	if err := req.Validate(); err != nil {
		return nil, fmt.Errorf("invalid workshop request: %w", err)
	}

	m.mutex.Lock()
	defer m.mutex.Unlock()

	now := time.Now()

	// Determine initial status
	status := WorkshopStatusDraft
	if now.After(req.StartTime) && now.Before(req.EndTime) {
		status = WorkshopStatusActive
	}

	w := &WorkshopEvent{
		ID:                   uuid.New().String(),
		Title:                req.Title,
		Description:          req.Description,
		Owner:                req.Owner,
		Template:             req.Template,
		ApprovedTemplates:    req.ApprovedTemplates,
		MaxParticipants:      req.MaxParticipants,
		BudgetPerParticipant: req.BudgetPerParticipant,
		StartTime:            req.StartTime,
		EndTime:              req.EndTime,
		EarlyAccessHours:     req.EarlyAccessHours,
		PolicyRestrictions:   req.PolicyRestrictions,
		Status:               status,
		JoinToken:            generateJoinToken(req.Title),
		Participants:         []WorkshopParticipant{},
		CreatedAt:            now,
		UpdatedAt:            now,
	}

	m.workshops[w.ID] = w
	if err := m.save(); err != nil {
		delete(m.workshops, w.ID)
		return nil, fmt.Errorf("failed to save workshop: %w", err)
	}

	cp := *w
	return &cp, nil
}

// GetWorkshop retrieves a workshop by ID.
func (m *Manager) GetWorkshop(ctx context.Context, workshopID string) (*WorkshopEvent, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	w, ok := m.workshops[workshopID]
	if !ok {
		return nil, ErrWorkshopNotFound
	}
	cp := *w
	return &cp, nil
}

// ListWorkshops returns all workshops matching the optional filter.
func (m *Manager) ListWorkshops(ctx context.Context, filter *WorkshopFilter) ([]*WorkshopEvent, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	var results []*WorkshopEvent
	for _, w := range m.workshops {
		if filter != nil && !filter.Matches(w) {
			continue
		}
		cp := *w
		results = append(results, &cp)
	}
	return results, nil
}

// UpdateWorkshop applies a partial update to an existing workshop.
func (m *Manager) UpdateWorkshop(ctx context.Context, workshopID string, req *UpdateWorkshopRequest) (*WorkshopEvent, error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	w, ok := m.workshops[workshopID]
	if !ok {
		return nil, ErrWorkshopNotFound
	}

	if req.Title != nil {
		w.Title = *req.Title
	}
	if req.Description != nil {
		w.Description = *req.Description
	}
	if req.Template != nil {
		w.Template = *req.Template
	}
	if req.ApprovedTemplates != nil {
		w.ApprovedTemplates = req.ApprovedTemplates
	}
	if req.MaxParticipants != nil {
		w.MaxParticipants = *req.MaxParticipants
	}
	if req.BudgetPerParticipant != nil {
		w.BudgetPerParticipant = *req.BudgetPerParticipant
	}
	if req.EndTime != nil {
		w.EndTime = *req.EndTime
	}
	if req.EarlyAccessHours != nil {
		w.EarlyAccessHours = *req.EarlyAccessHours
	}
	if req.PolicyRestrictions != nil {
		w.PolicyRestrictions = req.PolicyRestrictions
	}
	w.UpdatedAt = time.Now()

	if err := m.save(); err != nil {
		return nil, fmt.Errorf("failed to save workshop: %w", err)
	}

	cp := *w
	return &cp, nil
}

// DeleteWorkshop removes a workshop permanently.
func (m *Manager) DeleteWorkshop(ctx context.Context, workshopID string) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if _, ok := m.workshops[workshopID]; !ok {
		return ErrWorkshopNotFound
	}

	delete(m.workshops, workshopID)
	return m.save()
}

// ── Participant Management ────────────────────────────────────────────────────

// AddParticipant adds a participant who has redeemed the join token.
func (m *Manager) AddParticipant(ctx context.Context, workshopID string, p WorkshopParticipant) (*WorkshopEvent, error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	w, ok := m.workshops[workshopID]
	if !ok {
		return nil, ErrWorkshopNotFound
	}
	if w.Status == WorkshopStatusEnded || w.Status == WorkshopStatusArchived {
		return nil, ErrWorkshopEnded
	}
	if w.MaxParticipants > 0 && len(w.Participants) >= w.MaxParticipants {
		return nil, ErrWorkshopFull
	}

	// Prevent duplicate
	for _, existing := range w.Participants {
		if existing.UserID == p.UserID || (existing.Email != "" && existing.Email == p.Email) {
			cp := *w
			return &cp, nil // idempotent
		}
	}

	if p.JoinedAt.IsZero() {
		p.JoinedAt = time.Now()
	}
	if p.Status == "" {
		p.Status = "pending"
	}
	w.Participants = append(w.Participants, p)
	w.UpdatedAt = time.Now()

	if err := m.save(); err != nil {
		return nil, fmt.Errorf("failed to save workshop: %w", err)
	}
	cp := *w
	return &cp, nil
}

// RemoveParticipant removes a participant from the workshop.
func (m *Manager) RemoveParticipant(ctx context.Context, workshopID, userID string) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	w, ok := m.workshops[workshopID]
	if !ok {
		return ErrWorkshopNotFound
	}

	found := false
	filtered := w.Participants[:0]
	for _, p := range w.Participants {
		if p.UserID == userID {
			found = true
		} else {
			filtered = append(filtered, p)
		}
	}
	if !found {
		return ErrParticipantNotFound
	}
	w.Participants = filtered
	w.UpdatedAt = time.Now()
	return m.save()
}

// ── Lifecycle Operations ──────────────────────────────────────────────────────

// ProvisionWorkshop marks all pending participants as provisioned and returns a summary.
// Actual instance launch is handled by the daemon handler (which calls the AWS manager).
// This method updates participant statuses and returns the list needing provisioning.
func (m *Manager) ProvisionWorkshop(ctx context.Context, workshopID string) (*ProvisionResult, []*WorkshopParticipant, error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	w, ok := m.workshops[workshopID]
	if !ok {
		return nil, nil, ErrWorkshopNotFound
	}

	var toProvision []*WorkshopParticipant
	for i := range w.Participants {
		if w.Participants[i].Status == "pending" {
			toProvision = append(toProvision, &w.Participants[i])
		}
	}

	result := &ProvisionResult{
		WorkshopID: workshopID,
		Skipped:    len(w.Participants) - len(toProvision),
	}

	return result, toProvision, nil
}

// UpdateParticipantInstance sets the instance details for a provisioned participant.
func (m *Manager) UpdateParticipantInstance(ctx context.Context, workshopID, userID, instanceID, instanceName string, status string) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	w, ok := m.workshops[workshopID]
	if !ok {
		return ErrWorkshopNotFound
	}
	for i := range w.Participants {
		if w.Participants[i].UserID == userID {
			w.Participants[i].InstanceID = instanceID
			w.Participants[i].InstanceName = instanceName
			w.Participants[i].Status = status
			w.UpdatedAt = time.Now()
			return m.save()
		}
	}
	return ErrParticipantNotFound
}

// GetDashboard builds the live dashboard view for a workshop (#179).
func (m *Manager) GetDashboard(ctx context.Context, workshopID string) (*WorkshopDashboard, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	w, ok := m.workshops[workshopID]
	if !ok {
		return nil, ErrWorkshopNotFound
	}

	dash := &WorkshopDashboard{
		WorkshopID:        w.ID,
		Title:             w.Title,
		TotalParticipants: len(w.Participants),
		Status:            w.Status,
		Participants:      append([]WorkshopParticipant{}, w.Participants...),
	}

	for _, p := range w.Participants {
		switch p.Status {
		case "running":
			dash.ActiveInstances++
		case "stopped":
			dash.StoppedInstances++
		case "pending", "provisioned":
			dash.PendingInstances++
		}
	}

	now := time.Now()
	if w.Status != WorkshopStatusEnded && w.Status != WorkshopStatusArchived && now.Before(w.EndTime) {
		remaining := w.EndTime.Sub(now)
		h := int(remaining.Hours())
		m2 := int(remaining.Minutes()) % 60
		if h > 0 {
			dash.TimeRemaining = fmt.Sprintf("%dh %dm", h, m2)
		} else {
			dash.TimeRemaining = fmt.Sprintf("%dm", m2)
		}
	} else {
		dash.TimeRemaining = "ended"
	}

	return dash, nil
}

// EndWorkshop transitions the workshop to ended status and returns participants with instances (#135).
func (m *Manager) EndWorkshop(ctx context.Context, workshopID string) (*EndResult, []WorkshopParticipant, error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	w, ok := m.workshops[workshopID]
	if !ok {
		return nil, nil, ErrWorkshopNotFound
	}

	// Collect participants that have running instances
	var withInstances []WorkshopParticipant
	for _, p := range w.Participants {
		if p.InstanceName != "" {
			withInstances = append(withInstances, p)
		}
	}

	w.Status = WorkshopStatusEnded
	w.UpdatedAt = time.Now()

	if err := m.save(); err != nil {
		return nil, nil, fmt.Errorf("failed to save workshop: %w", err)
	}

	return &EndResult{WorkshopID: workshopID}, withInstances, nil
}

// GetDownloadManifest returns a manifest of participant work for bulk download (#180).
func (m *Manager) GetDownloadManifest(ctx context.Context, workshopID string) (*DownloadManifest, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	w, ok := m.workshops[workshopID]
	if !ok {
		return nil, ErrWorkshopNotFound
	}

	manifest := &DownloadManifest{
		WorkshopID:  w.ID,
		GeneratedAt: time.Now(),
	}
	for _, p := range w.Participants {
		note := "no workspace provisioned"
		if p.InstanceName != "" {
			note = fmt.Sprintf("ssh into %s to download work, or use: prism workspace ssh %s", p.InstanceName, p.InstanceName)
		}
		manifest.Participants = append(manifest.Participants, ParticipantWork{
			UserID:       p.UserID,
			Email:        p.Email,
			DisplayName:  p.DisplayName,
			InstanceName: p.InstanceName,
			DownloadNote: note,
		})
	}
	return manifest, nil
}

// ── Config Templates (#183) ───────────────────────────────────────────────────

// SaveConfig saves the current workshop settings as a reusable config template.
func (m *Manager) SaveConfig(ctx context.Context, workshopID, configName string) (*WorkshopConfig, error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	w, ok := m.workshops[workshopID]
	if !ok {
		return nil, ErrWorkshopNotFound
	}

	durationHours := int(w.EndTime.Sub(w.StartTime).Hours())

	config := &WorkshopConfig{
		Name:                 configName,
		Template:             w.Template,
		MaxParticipants:      w.MaxParticipants,
		BudgetPerParticipant: w.BudgetPerParticipant,
		DurationHours:        durationHours,
		EarlyAccessHours:     w.EarlyAccessHours,
		Description:          fmt.Sprintf("Saved from workshop: %s", w.Title),
		CreatedAt:            time.Now(),
	}

	m.configs[configName] = config
	if err := m.save(); err != nil {
		delete(m.configs, configName)
		return nil, fmt.Errorf("failed to save config: %w", err)
	}

	cp := *config
	return &cp, nil
}

// ListConfigs returns all saved workshop config templates.
func (m *Manager) ListConfigs(ctx context.Context) ([]*WorkshopConfig, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	var results []*WorkshopConfig
	for _, c := range m.configs {
		cp := *c
		results = append(results, &cp)
	}
	return results, nil
}

// GetConfig retrieves a single config by name.
func (m *Manager) GetConfig(ctx context.Context, name string) (*WorkshopConfig, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	c, ok := m.configs[name]
	if !ok {
		return nil, ErrConfigNotFound
	}
	cp := *c
	return &cp, nil
}

// CreateFromConfig creates a new workshop pre-populated from a saved config.
func (m *Manager) CreateFromConfig(ctx context.Context, configName string, req *CreateWorkshopRequest) (*WorkshopEvent, error) {
	m.mutex.RLock()
	config, ok := m.configs[configName]
	m.mutex.RUnlock()

	if !ok {
		return nil, ErrConfigNotFound
	}

	// Fill in blanks from config
	if req.Template == "" {
		req.Template = config.Template
	}
	if req.MaxParticipants == 0 {
		req.MaxParticipants = config.MaxParticipants
	}
	if req.BudgetPerParticipant == 0 {
		req.BudgetPerParticipant = config.BudgetPerParticipant
	}
	if req.EarlyAccessHours == 0 {
		req.EarlyAccessHours = config.EarlyAccessHours
	}
	// If end time not provided but duration is known, derive from start
	if req.EndTime.IsZero() && !req.StartTime.IsZero() && config.DurationHours > 0 {
		req.EndTime = req.StartTime.Add(time.Duration(config.DurationHours) * time.Hour)
	}

	return m.CreateWorkshop(ctx, req)
}

// ── Helpers ───────────────────────────────────────────────────────────────────

// generateJoinToken creates a short, uppercase token code from the workshop title.
// Example: "NeurIPS DL Tutorial" → "WORKSHOP-NEURIPS-<8 random chars>"
func generateJoinToken(title string) string {
	id := uuid.New().String()[:8]
	return fmt.Sprintf("WS-%s", id)
}
