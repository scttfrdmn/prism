package profile

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// InvitationManager handles the creation and storage of invitations
type InvitationManager struct {
	configPath     string
	invitations    map[string]InvitationToken
	generatedPath  string
	receivedPath   string
	mutex          sync.RWMutex
	profileManager *ManagerEnhanced
}

// NewInvitationManager creates a new invitation manager
func NewInvitationManager(profileManager *ManagerEnhanced) (*InvitationManager, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}
	
	// Create CloudWorkstation invitations directory if it doesn't exist
	invitationsDir := filepath.Join(homeDir, ".cloudworkstation", "invitations")
	generatedDir := filepath.Join(invitationsDir, "generated")
	receivedDir := filepath.Join(invitationsDir, "received")
	
	for _, dir := range []string{invitationsDir, generatedDir, receivedDir} {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create invitations directory %s: %w", dir, err)
		}
	}
	
	manager := &InvitationManager{
		configPath:     invitationsDir,
		generatedPath:  generatedDir,
		receivedPath:   receivedDir,
		invitations:    make(map[string]InvitationToken),
		profileManager: profileManager,
	}
	
	// Load existing invitations
	if err := manager.loadInvitations(); err != nil {
		return nil, fmt.Errorf("failed to load invitations: %w", err)
	}
	
	return manager, nil
}

// loadInvitations loads all invitation files from disk
func (m *InvitationManager) loadInvitations() error {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	
	// Clear existing invitations
	m.invitations = make(map[string]InvitationToken)
	
	// Load generated invitations
	generatedFiles, err := os.ReadDir(m.generatedPath)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to read generated invitations: %w", err)
	}
	
	for _, file := range generatedFiles {
		if file.IsDir() {
			continue
		}
		
		// Only process .json files
		if filepath.Ext(file.Name()) != ".json" {
			continue
		}
		
		filePath := filepath.Join(m.generatedPath, file.Name())
		data, err := os.ReadFile(filePath)
		if err != nil {
			continue // Skip files we can't read
		}
		
		var invitation InvitationToken
		if err := json.Unmarshal(data, &invitation); err != nil {
			continue // Skip invalid files
		}
		
		// Add to map
		m.invitations[invitation.Token] = invitation
	}
	
	return nil
}

// CreateInvitation generates and stores a new invitation
func (m *InvitationManager) CreateInvitation(name string, invType InvitationType, validDays int, s3ConfigPath string) (*InvitationToken, error) {
	// Get current profile
	currentProfile, err := m.profileManager.GetCurrentProfile()
	if err != nil {
		return nil, fmt.Errorf("failed to get current profile: %w", err)
	}
	
	// Generate the invitation token
	invitation, err := GenerateInvitationToken(
		currentProfile.AWSProfile,
		currentProfile.AWSProfile, // Using profile name as account ID for now
		name,
		invType,
		validDays,
		s3ConfigPath,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to generate invitation: %w", err)
	}
	
	// Save the invitation
	if err := m.saveInvitation(invitation); err != nil {
		return nil, fmt.Errorf("failed to save invitation: %w", err)
	}
	
	return invitation, nil
}

// saveInvitation writes an invitation to disk
func (m *InvitationManager) saveInvitation(invitation *InvitationToken) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	
	// Store in memory map
	m.invitations[invitation.Token] = *invitation
	
	// Write to file
	data, err := json.MarshalIndent(invitation, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal invitation: %w", err)
	}
	
	filename := fmt.Sprintf("%s.json", invitation.Token)
	filePath := filepath.Join(m.generatedPath, filename)
	
	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write invitation file: %w", err)
	}
	
	return nil
}

// GetInvitation retrieves an invitation by token
func (m *InvitationManager) GetInvitation(token string) (*InvitationToken, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	
	invitation, exists := m.invitations[token]
	if !exists {
		return nil, fmt.Errorf("invitation not found")
	}
	
	return &invitation, nil
}

// ListInvitations returns all valid invitations
func (m *InvitationManager) ListInvitations() []InvitationToken {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	
	var validInvitations []InvitationToken
	now := time.Now()
	
	for _, invitation := range m.invitations {
		if now.Before(invitation.Expires) {
			validInvitations = append(validInvitations, invitation)
		}
	}
	
	return validInvitations
}

// RevokeInvitation removes an invitation
func (m *InvitationManager) RevokeInvitation(token string) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	
	// Check if invitation exists
	if _, exists := m.invitations[token]; !exists {
		return fmt.Errorf("invitation not found")
	}
	
	// Remove from memory map
	delete(m.invitations, token)
	
	// Remove file
	filename := fmt.Sprintf("%s.json", token)
	filePath := filepath.Join(m.generatedPath, filename)
	
	if err := os.Remove(filePath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove invitation file: %w", err)
	}
	
	return nil
}

// AddToProfile accepts an invitation and creates a profile from it
func (m *InvitationManager) AddToProfile(encoded string, profileName string) error {
	// Decode the invitation
	invitation, err := DecodeFromString(encoded)
	if err != nil {
		return fmt.Errorf("invalid invitation: %w", err)
	}
	
	// Verify it's still valid
	if !invitation.IsValid() {
		return fmt.Errorf("invitation has expired")
	}
	
	// Create profile from invitation
	profile := Profile{
		Type:            ProfileTypeInvitation,
		Name:            profileName,
		AWSProfile:      profileName,
		InvitationToken: invitation.Token,
		OwnerAccount:    invitation.OwnerAccount,
		S3ConfigPath:    invitation.S3ConfigPath,
		Region:          "", // Use default region
		CreatedAt:       time.Now(),
	}
	
	// Add the profile
	if err := m.profileManager.AddProfile(profile); err != nil {
		return fmt.Errorf("failed to create profile from invitation: %w", err)
	}
	
	// Save the invitation in the received folder for reference
	data, err := json.MarshalIndent(invitation, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal invitation: %w", err)
	}
	
	filename := fmt.Sprintf("%s.json", invitation.Token)
	filePath := filepath.Join(m.receivedPath, filename)
	
	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return fmt.Errorf("failed to save received invitation: %w", err)
	}
	
	return nil
}