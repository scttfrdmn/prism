// Package aws provides AMI caching functionality for the Universal AMI System
package aws

import (
	"sync"
	"time"

	"github.com/scttfrdmn/cloudworkstation/pkg/types"
)

// AMICache provides in-memory caching for AMI information to improve performance
type AMICache struct {
	// Cache storage
	cache      sync.Map // key -> *cachedAMIInfo

	// Cache configuration
	defaultTTL time.Duration
	maxSize    int

	// Statistics
	hits   int64
	misses int64
}

// cachedAMIInfo wraps AMI info with caching metadata
type cachedAMIInfo struct {
	ami       *types.AMIInfo
	cachedAt  time.Time
	ttl       time.Duration
	accessCount int64
	lastAccess time.Time
}

// NewAMICache creates a new AMI cache with default configuration
func NewAMICache() *AMICache {
	return &AMICache{
		defaultTTL: 30 * time.Minute, // Cache AMI info for 30 minutes
		maxSize:    1000,             // Maximum 1000 cached AMIs
	}
}

// NewAMICacheWithConfig creates a new AMI cache with custom configuration
func NewAMICacheWithConfig(ttl time.Duration, maxSize int) *AMICache {
	return &AMICache{
		defaultTTL: ttl,
		maxSize:    maxSize,
	}
}

// GetAMI retrieves AMI information from cache
func (c *AMICache) GetAMI(amiID, region string) *types.AMIInfo {
	key := c.makeKey(amiID, region)

	value, exists := c.cache.Load(key)
	if !exists {
		c.misses++
		return nil
	}

	cached := value.(*cachedAMIInfo)

	// Check if cache entry has expired
	if time.Since(cached.cachedAt) > cached.ttl {
		c.cache.Delete(key)
		c.misses++
		return nil
	}

	// Update access statistics
	cached.accessCount++
	cached.lastAccess = time.Now()
	c.hits++

	// Return a copy to prevent modification
	return c.copyAMIInfo(cached.ami)
}

// SetAMI stores AMI information in cache
func (c *AMICache) SetAMI(amiID, region string, ami *types.AMIInfo) {
	key := c.makeKey(amiID, region)

	cached := &cachedAMIInfo{
		ami:         c.copyAMIInfo(ami), // Store a copy to prevent modification
		cachedAt:    time.Now(),
		ttl:         c.defaultTTL,
		accessCount: 0,
		lastAccess:  time.Now(),
	}

	c.cache.Store(key, cached)

	// Enforce cache size limit
	c.enforceMaxSize()
}

// SetAMIWithTTL stores AMI information in cache with custom TTL
func (c *AMICache) SetAMIWithTTL(amiID, region string, ami *types.AMIInfo, ttl time.Duration) {
	key := c.makeKey(amiID, region)

	cached := &cachedAMIInfo{
		ami:         c.copyAMIInfo(ami),
		cachedAt:    time.Now(),
		ttl:         ttl,
		accessCount: 0,
		lastAccess:  time.Now(),
	}

	c.cache.Store(key, cached)
	c.enforceMaxSize()
}

// InvalidateAMI removes AMI information from cache
func (c *AMICache) InvalidateAMI(amiID, region string) {
	key := c.makeKey(amiID, region)
	c.cache.Delete(key)
}

// InvalidateRegion removes all AMI information for a specific region
func (c *AMICache) InvalidateRegion(region string) {
	c.cache.Range(func(key, value interface{}) bool {
		keyStr := key.(string)
		if c.keyMatchesRegion(keyStr, region) {
			c.cache.Delete(key)
		}
		return true
	})
}

// Clear removes all entries from the cache
func (c *AMICache) Clear() {
	c.cache.Range(func(key, value interface{}) bool {
		c.cache.Delete(key)
		return true
	})
	c.hits = 0
	c.misses = 0
}

// GetStats returns cache statistics
func (c *AMICache) GetStats() *AMICacheStats {
	size := c.getCurrentSize()

	return &AMICacheStats{
		Size:     size,
		MaxSize:  c.maxSize,
		Hits:     c.hits,
		Misses:   c.misses,
		HitRatio: c.calculateHitRatio(),
		TTL:      c.defaultTTL,
	}
}

// CleanupExpired removes expired entries from the cache
func (c *AMICache) CleanupExpired() int {
	var expiredCount int
	now := time.Now()

	c.cache.Range(func(key, value interface{}) bool {
		cached := value.(*cachedAMIInfo)
		if now.Sub(cached.cachedAt) > cached.ttl {
			c.cache.Delete(key)
			expiredCount++
		}
		return true
	})

	return expiredCount
}

// StartCleanupRoutine starts a background goroutine to periodically clean up expired entries
func (c *AMICache) StartCleanupRoutine(interval time.Duration) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for range ticker.C {
			c.CleanupExpired()
		}
	}()
}

// Private helper methods

func (c *AMICache) makeKey(amiID, region string) string {
	return amiID + ":" + region
}

func (c *AMICache) keyMatchesRegion(key, region string) bool {
	return key[len(key)-len(region):] == region
}

func (c *AMICache) copyAMIInfo(ami *types.AMIInfo) *types.AMIInfo {
	if ami == nil {
		return nil
	}

	// Create a deep copy of AMI info
	copy := *ami

	// Copy maps if they exist
	if ami.Tags != nil {
		copy.Tags = make(map[string]string)
		for k, v := range ami.Tags {
			copy.Tags[k] = v
		}
	}

	// Copy community info if it exists
	if ami.CommunityInfo != nil {
		communityInfo := *ami.CommunityInfo
		copy.CommunityInfo = &communityInfo

		// Copy reviews slice if it exists
		if ami.CommunityInfo.Reviews != nil {
			copy.CommunityInfo.Reviews = make([]types.AMIReview, len(ami.CommunityInfo.Reviews))
			for i, review := range ami.CommunityInfo.Reviews {
				copy.CommunityInfo.Reviews[i] = review
			}
		}
	}

	return &copy
}

func (c *AMICache) getCurrentSize() int {
	size := 0
	c.cache.Range(func(key, value interface{}) bool {
		size++
		return true
	})
	return size
}

func (c *AMICache) calculateHitRatio() float64 {
	total := c.hits + c.misses
	if total == 0 {
		return 0.0
	}
	return float64(c.hits) / float64(total)
}

func (c *AMICache) enforceMaxSize() {
	if c.maxSize <= 0 {
		return // No size limit
	}

	currentSize := c.getCurrentSize()
	if currentSize <= c.maxSize {
		return // Within limit
	}

	// Collect all entries with access information for LRU eviction
	type entryInfo struct {
		key         interface{}
		lastAccess  time.Time
		accessCount int64
	}

	var entries []entryInfo
	c.cache.Range(func(key, value interface{}) bool {
		cached := value.(*cachedAMIInfo)
		entries = append(entries, entryInfo{
			key:         key,
			lastAccess:  cached.lastAccess,
			accessCount: cached.accessCount,
		})
		return true
	})

	// Sort by last access time (least recently used first)
	// In production, this would use a more sophisticated sorting algorithm
	toRemove := currentSize - c.maxSize
	for i := 0; i < toRemove && i < len(entries); i++ {
		// Find least recently used entry
		oldestIndex := i
		for j := i + 1; j < len(entries); j++ {
			if entries[j].lastAccess.Before(entries[oldestIndex].lastAccess) {
				oldestIndex = j
			}
		}

		// Remove the oldest entry
		if oldestIndex != i {
			entries[i], entries[oldestIndex] = entries[oldestIndex], entries[i]
		}
		c.cache.Delete(entries[i].key)
	}
}

// AMICacheStats represents cache statistics
type AMICacheStats struct {
	Size     int           `json:"size"`      // Current number of cached entries
	MaxSize  int           `json:"max_size"`  // Maximum cache size
	Hits     int64         `json:"hits"`      // Cache hits
	Misses   int64         `json:"misses"`    // Cache misses
	HitRatio float64       `json:"hit_ratio"` // Hit ratio (0.0 to 1.0)
	TTL      time.Duration `json:"ttl"`       // Default TTL
}

// AMICacheConfig represents cache configuration
type AMICacheConfig struct {
	DefaultTTL     time.Duration `json:"default_ttl"`     // Default time-to-live for cached entries
	MaxSize        int           `json:"max_size"`        // Maximum number of cached entries
	CleanupInterval time.Duration `json:"cleanup_interval"` // How often to clean up expired entries
}

// ConfigureCache applies configuration to the cache
func (c *AMICache) ConfigureCache(config *AMICacheConfig) {
	if config.DefaultTTL > 0 {
		c.defaultTTL = config.DefaultTTL
	}
	if config.MaxSize > 0 {
		c.maxSize = config.MaxSize
	}
	if config.CleanupInterval > 0 {
		c.StartCleanupRoutine(config.CleanupInterval)
	}
}