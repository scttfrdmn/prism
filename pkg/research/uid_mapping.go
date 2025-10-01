package research

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"sort"
	"sync"
	"time"
)

// UIDGIDAllocator manages consistent UID/GID allocation across instances
type UIDGIDAllocator struct {
	mu sync.RWMutex

	// Configuration
	baseUID int
	baseGID int
	maxUID  int
	maxGID  int

	// Allocation tracking
	allocations map[string]*UIDGIDAllocation // profileID:username -> allocation
	uidIndex    map[int]string               // UID -> profileID:username
	gidIndex    map[int]string               // GID -> profileID:username
}

// UIDGIDAllocation represents an allocated UID/GID pair
type UIDGIDAllocation struct {
	ProfileID     string   `json:"profile_id"`
	Username      string   `json:"username"`
	UID           int      `json:"uid"`
	GID           int      `json:"gid"`
	AllocatedAt   int64    `json:"allocated_at"`
	LastUsed      int64    `json:"last_used"`
	InstancesUsed []string `json:"instances_used,omitempty"`
}

// NewUIDGIDAllocator creates a new UID/GID allocator
func NewUIDGIDAllocator() *UIDGIDAllocator {
	return &UIDGIDAllocator{
		baseUID:     ResearchUserBaseUID,
		baseGID:     ResearchUserBaseGID,
		maxUID:      ResearchUserMaxUID,
		maxGID:      ResearchUserMaxGID,
		allocations: make(map[string]*UIDGIDAllocation),
		uidIndex:    make(map[int]string),
		gidIndex:    make(map[int]string),
	}
}

// AllocateUIDGID allocates a consistent UID/GID for a profile/username combination
func (allocator *UIDGIDAllocator) AllocateUIDGID(profileID, username string) (*UIDGIDAllocation, error) {
	allocator.mu.Lock()
	defer allocator.mu.Unlock()

	key := fmt.Sprintf("%s:%s", profileID, username)

	// Check if already allocated
	if existing, exists := allocator.allocations[key]; exists {
		return existing, nil
	}

	// Generate deterministic UID/GID based on profile and username
	uid, gid, err := allocator.generateDeterministicUIGID(profileID, username)
	if err != nil {
		return nil, fmt.Errorf("failed to generate UID/GID: %w", err)
	}

	// Create allocation
	allocation := &UIDGIDAllocation{
		ProfileID:   profileID,
		Username:    username,
		UID:         uid,
		GID:         gid,
		AllocatedAt: getCurrentTimestamp(),
		LastUsed:    getCurrentTimestamp(),
	}

	// Store allocation
	allocator.allocations[key] = allocation
	allocator.uidIndex[uid] = key
	allocator.gidIndex[gid] = key

	return allocation, nil
}

// GetAllocation retrieves an existing UID/GID allocation
func (allocator *UIDGIDAllocator) GetAllocation(profileID, username string) (*UIDGIDAllocation, error) {
	allocator.mu.RLock()
	defer allocator.mu.RUnlock()

	key := fmt.Sprintf("%s:%s", profileID, username)
	if allocation, exists := allocator.allocations[key]; exists {
		return allocation, nil
	}

	return nil, fmt.Errorf("no UID/GID allocation found for %s:%s", profileID, username)
}

// UpdateLastUsed updates the last used timestamp and instance list for an allocation
func (allocator *UIDGIDAllocator) UpdateLastUsed(profileID, username, instanceID string) error {
	allocator.mu.Lock()
	defer allocator.mu.Unlock()

	key := fmt.Sprintf("%s:%s", profileID, username)
	allocation, exists := allocator.allocations[key]
	if !exists {
		return fmt.Errorf("no allocation found for %s:%s", profileID, username)
	}

	allocation.LastUsed = getCurrentTimestamp()

	// Add instance to the list if not already present
	instanceExists := false
	for _, instance := range allocation.InstancesUsed {
		if instance == instanceID {
			instanceExists = true
			break
		}
	}

	if !instanceExists {
		allocation.InstancesUsed = append(allocation.InstancesUsed, instanceID)

		// Keep only the last 10 instances to avoid unbounded growth
		if len(allocation.InstancesUsed) > 10 {
			allocation.InstancesUsed = allocation.InstancesUsed[len(allocation.InstancesUsed)-10:]
		}
	}

	return nil
}

// ListAllocations returns all allocations for a profile
func (allocator *UIDGIDAllocator) ListAllocations(profileID string) ([]*UIDGIDAllocation, error) {
	allocator.mu.RLock()
	defer allocator.mu.RUnlock()

	var allocations []*UIDGIDAllocation
	for _, allocation := range allocator.allocations {
		if allocation.ProfileID == profileID {
			allocations = append(allocations, allocation)
		}
	}

	// Sort by username for consistent ordering
	sort.Slice(allocations, func(i, j int) bool {
		return allocations[i].Username < allocations[j].Username
	})

	return allocations, nil
}

// GetAllAllocations returns all allocations across all profiles
func (allocator *UIDGIDAllocator) GetAllAllocations() ([]*UIDGIDAllocation, error) {
	allocator.mu.RLock()
	defer allocator.mu.RUnlock()

	var allocations []*UIDGIDAllocation
	for _, allocation := range allocator.allocations {
		allocations = append(allocations, allocation)
	}

	// Sort by profile, then username
	sort.Slice(allocations, func(i, j int) bool {
		if allocations[i].ProfileID != allocations[j].ProfileID {
			return allocations[i].ProfileID < allocations[j].ProfileID
		}
		return allocations[i].Username < allocations[j].Username
	})

	return allocations, nil
}

// ReleaseAllocation removes an allocation (for cleanup)
func (allocator *UIDGIDAllocator) ReleaseAllocation(profileID, username string) error {
	allocator.mu.Lock()
	defer allocator.mu.Unlock()

	key := fmt.Sprintf("%s:%s", profileID, username)
	allocation, exists := allocator.allocations[key]
	if !exists {
		return fmt.Errorf("no allocation found for %s:%s", profileID, username)
	}

	// Remove from all indexes
	delete(allocator.allocations, key)
	delete(allocator.uidIndex, allocation.UID)
	delete(allocator.gidIndex, allocation.GID)

	return nil
}

// ValidateAllocation checks if a UID/GID allocation is valid and available
func (allocator *UIDGIDAllocator) ValidateAllocation(uid, gid int, excludeKey string) error {
	allocator.mu.RLock()
	defer allocator.mu.RUnlock()

	// Check UID range
	if uid < allocator.baseUID || uid > allocator.maxUID {
		return fmt.Errorf("UID %d outside allowed range %d-%d", uid, allocator.baseUID, allocator.maxUID)
	}

	// Check GID range
	if gid < allocator.baseGID || gid > allocator.maxGID {
		return fmt.Errorf("GID %d outside allowed range %d-%d", gid, allocator.baseGID, allocator.maxGID)
	}

	// Check for conflicts
	if existingKey, exists := allocator.uidIndex[uid]; exists && existingKey != excludeKey {
		return fmt.Errorf("UID %d already allocated to %s", uid, existingKey)
	}

	if existingKey, exists := allocator.gidIndex[gid]; exists && existingKey != excludeKey {
		return fmt.Errorf("GID %d already allocated to %s", gid, existingKey)
	}

	return nil
}

// generateDeterministicUIGID generates consistent UID/GID based on profileID and username
func (allocator *UIDGIDAllocator) generateDeterministicUIGID(profileID, username string) (uid, gid int, err error) {
	// Create deterministic hash from profile + username
	input := fmt.Sprintf("%s:%s", profileID, username)
	hash := sha256.Sum256([]byte(input))

	// Use first 8 bytes to generate UID offset
	uidOffset := binary.BigEndian.Uint64(hash[:8])

	// Map to allowed UID range
	uidRange := uint64(allocator.maxUID - allocator.baseUID + 1)
	targetUID := allocator.baseUID + int(uidOffset%uidRange)

	// Find next available UID starting from target
	uid = targetUID
	for attempts := 0; attempts < int(uidRange); attempts++ {
		if _, exists := allocator.uidIndex[uid]; !exists {
			// Found available UID
			gid = uid // GID matches UID for simplicity
			if _, exists := allocator.gidIndex[gid]; !exists {
				return uid, gid, nil
			}
		}

		// Try next UID
		uid++
		if uid > allocator.maxUID {
			uid = allocator.baseUID
		}
	}

	return 0, 0, fmt.Errorf("no available UID/GID pairs in range %d-%d", allocator.baseUID, allocator.maxUID)
}

// ProfileUIDMapper provides a higher-level interface for profile-based UID mapping
type ProfileUIDMapper struct {
	allocator  *UIDGIDAllocator
	profileMgr ProfileManager
}

// NewProfileUIDMapper creates a new profile-based UID mapper
func NewProfileUIDMapper(profileMgr ProfileManager) *ProfileUIDMapper {
	return &ProfileUIDMapper{
		allocator:  NewUIDGIDAllocator(),
		profileMgr: profileMgr,
	}
}

// GetCurrentProfileUIDGID gets UID/GID for a username in the current profile
func (mapper *ProfileUIDMapper) GetCurrentProfileUIDGID(username string) (*UIDGIDAllocation, error) {
	profileID, err := mapper.profileMgr.GetCurrentProfile()
	if err != nil {
		return nil, fmt.Errorf("failed to get current profile: %w", err)
	}

	return mapper.allocator.AllocateUIDGID(profileID, username)
}

// GetProfileUIDGID gets UID/GID for a username in a specific profile
func (mapper *ProfileUIDMapper) GetProfileUIDGID(profileID, username string) (*UIDGIDAllocation, error) {
	return mapper.allocator.AllocateUIDGID(profileID, username)
}

// ListCurrentProfileAllocations lists all UID/GID allocations for the current profile
func (mapper *ProfileUIDMapper) ListCurrentProfileAllocations() ([]*UIDGIDAllocation, error) {
	profileID, err := mapper.profileMgr.GetCurrentProfile()
	if err != nil {
		return nil, fmt.Errorf("failed to get current profile: %w", err)
	}

	return mapper.allocator.ListAllocations(profileID)
}

// UpdateUsage updates usage tracking for a user allocation
func (mapper *ProfileUIDMapper) UpdateUsage(username, instanceID string) error {
	profileID, err := mapper.profileMgr.GetCurrentProfile()
	if err != nil {
		return fmt.Errorf("failed to get current profile: %w", err)
	}

	return mapper.allocator.UpdateLastUsed(profileID, username, instanceID)
}

// UID/GID Range Management

// IsResearchUserUID checks if a UID is in the research user range
func IsResearchUserUID(uid int) bool {
	return uid >= ResearchUserBaseUID && uid <= ResearchUserMaxUID
}

// IsResearchUserGID checks if a GID is in the research user range
func IsResearchUserGID(gid int) bool {
	return gid >= ResearchUserBaseGID && gid <= ResearchUserMaxGID
}

// GetResearchUserUIDRange returns the UID range for research users
func GetResearchUserUIDRange() (min, max int) {
	return ResearchUserBaseUID, ResearchUserMaxUID
}

// GetResearchUserGIDRange returns the GID range for research users
func GetResearchUserGIDRange() (min, max int) {
	return ResearchUserBaseGID, ResearchUserMaxGID
}

// Helper function to get current timestamp
func getCurrentTimestamp() int64 {
	return time.Now().Unix()
}
