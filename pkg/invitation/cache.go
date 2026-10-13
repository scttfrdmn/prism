// Package invitation provides local invitation cache management for Prism v0.5.11+
//
// This file implements a local cache for storing received invitation tokens,
// allowing users to manage multiple invitations without dealing with long tokens.
package invitation

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/scttfrdmn/prism/pkg/types"
)

// CachedInvitation represents a cached invitation with token and metadata
type CachedInvitation struct {
	// Token is the full invitation token
	Token string `json:"token"`

	// Decoded invitation details (from token or API)
	InvitationID string                 `json:"invitation_id"`
	ProjectID    string                 `json:"project_id"`
	ProjectName  string                 `json:"project_name"`
	Email        string                 `json:"email"`
	Role         types.ProjectRole      `json:"role"`
	InvitedBy    string                 `json:"invited_by"`
	InvitedAt    time.Time              `json:"invited_at"`
	ExpiresAt    time.Time              `json:"expires_at"`
	Status       types.InvitationStatus `json:"status"`
	Message      string                 `json:"message,omitempty"`

	// Cache metadata
	AddedAt time.Time `json:"added_at"`
}

// InvitationCache manages locally cached invitations
type InvitationCache struct {
	cachePath   string
	mutex       sync.RWMutex
	invitations map[string]*CachedInvitation // projectName → CachedInvitation
}

// NewInvitationCache creates a new invitation cache
func NewInvitationCache() (*InvitationCache, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}

	stateDir := filepath.Join(homeDir, ".prism")
	if err := os.MkdirAll(stateDir, 0700); err != nil {
		return nil, fmt.Errorf("failed to create state directory: %w", err)
	}

	cachePath := filepath.Join(stateDir, "invitation_cache.json")

	cache := &InvitationCache{
		cachePath:   cachePath,
		invitations: make(map[string]*CachedInvitation),
	}

	// Load existing cache
	if err := cache.load(); err != nil {
		return nil, fmt.Errorf("failed to load invitation cache: %w", err)
	}

	return cache, nil
}

// Add adds an invitation token to the cache
func (c *InvitationCache) Add(ctx context.Context, token string, invitation *types.Invitation, projectName string) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	// Create cached invitation
	cached := &CachedInvitation{
		Token:        token,
		InvitationID: invitation.ID,
		ProjectID:    invitation.ProjectID,
		ProjectName:  projectName,
		Email:        invitation.Email,
		Role:         invitation.Role,
		InvitedBy:    invitation.InvitedBy,
		InvitedAt:    invitation.InvitedAt,
		ExpiresAt:    invitation.ExpiresAt,
		Status:       invitation.Status,
		Message:      invitation.Message,
		AddedAt:      time.Now(),
	}

	// Store by project name for easy lookup
	c.invitations[projectName] = cached

	// Save to disk
	if err := c.save(); err != nil {
		return fmt.Errorf("failed to save cache: %w", err)
	}

	return nil
}

// Get retrieves a cached invitation by project name
func (c *InvitationCache) Get(projectName string) (*CachedInvitation, error) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	cached, exists := c.invitations[projectName]
	if !exists {
		return nil, fmt.Errorf("no cached invitation for project: %s", projectName)
	}

	// Return copy to prevent external modification
	cachedCopy := *cached
	return &cachedCopy, nil
}

// List returns all cached invitations
func (c *InvitationCache) List() ([]*CachedInvitation, error) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	invitations := make([]*CachedInvitation, 0, len(c.invitations))
	for _, inv := range c.invitations {
		// Return copies
		invCopy := *inv
		invitations = append(invitations, &invCopy)
	}

	return invitations, nil
}

// Remove removes a cached invitation by project name
func (c *InvitationCache) Remove(projectName string) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if _, exists := c.invitations[projectName]; !exists {
		return fmt.Errorf("no cached invitation for project: %s", projectName)
	}

	delete(c.invitations, projectName)

	// Save to disk
	if err := c.save(); err != nil {
		return fmt.Errorf("failed to save cache: %w", err)
	}

	return nil
}

// CleanupExpired removes expired invitations from cache
func (c *InvitationCache) CleanupExpired() (int, error) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	now := time.Now()
	removed := 0

	for projectName, inv := range c.invitations {
		if now.After(inv.ExpiresAt) {
			delete(c.invitations, projectName)
			removed++
		}
	}

	if removed > 0 {
		if err := c.save(); err != nil {
			return 0, fmt.Errorf("failed to save cache: %w", err)
		}
	}

	return removed, nil
}

// UpdateStatus updates the status of a cached invitation
func (c *InvitationCache) UpdateStatus(projectName string, status types.InvitationStatus) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	inv, exists := c.invitations[projectName]
	if !exists {
		return fmt.Errorf("no cached invitation for project: %s", projectName)
	}

	inv.Status = status

	// Save to disk
	if err := c.save(); err != nil {
		return fmt.Errorf("failed to save cache: %w", err)
	}

	return nil
}

// GetSummary returns a summary of cached invitations
func (c *InvitationCache) GetSummary() (total, pending, accepted, declined, expired int) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	now := time.Now()
	total = len(c.invitations)

	for _, inv := range c.invitations {
		// Check if expired
		if now.After(inv.ExpiresAt) {
			expired++
		} else {
			// Check status
			switch inv.Status {
			case types.InvitationPending:
				pending++
			case types.InvitationAccepted:
				accepted++
			case types.InvitationDeclined:
				declined++
			}
		}
	}

	return
}

// load loads the cache from disk
func (c *InvitationCache) load() error {
	data, err := os.ReadFile(c.cachePath)
	if os.IsNotExist(err) {
		return nil // No cache file yet
	}
	if err != nil {
		return fmt.Errorf("failed to read cache file: %w", err)
	}

	var invitations []*CachedInvitation
	if err := json.Unmarshal(data, &invitations); err != nil {
		return fmt.Errorf("failed to unmarshal cache: %w", err)
	}

	// Build index by project name
	for _, inv := range invitations {
		c.invitations[inv.ProjectName] = inv
	}

	return nil
}

// save saves the cache to disk
func (c *InvitationCache) save() error {
	// Convert map to slice
	invitations := make([]*CachedInvitation, 0, len(c.invitations))
	for _, inv := range c.invitations {
		invitations = append(invitations, inv)
	}

	data, err := json.MarshalIndent(invitations, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal cache: %w", err)
	}

	if err := os.WriteFile(c.cachePath, data, 0600); err != nil {
		return fmt.Errorf("failed to write cache file: %w", err)
	}

	return nil
}

// Clear removes all cached invitations
func (c *InvitationCache) Clear() error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	c.invitations = make(map[string]*CachedInvitation)

	// Save empty cache
	if err := c.save(); err != nil {
		return fmt.Errorf("failed to save cache: %w", err)
	}

	return nil
}

// Size returns the number of cached invitations
func (c *InvitationCache) Size() int {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	return len(c.invitations)
}
