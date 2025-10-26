// Package templates provides Prism's unified template system.
package templates

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// UsageStats tracks template usage statistics
type UsageStats struct {
	mu    sync.RWMutex
	Stats map[string]*TemplateUsage `json:"stats"`
}

// TemplateUsage represents usage data for a single template
type TemplateUsage struct {
	TemplateName      string    `json:"template_name"`
	LaunchCount       int       `json:"launch_count"`
	LastUsed          time.Time `json:"last_used"`
	TotalLaunchTime   int       `json:"total_launch_time_seconds"`
	AverageLaunchTime int       `json:"average_launch_time_seconds"`
	SuccessRate       float64   `json:"success_rate"`
	SuccessCount      int       `json:"success_count"`
	FailureCount      int       `json:"failure_count"`
}

var (
	usageStats *UsageStats
	statsOnce  sync.Once
)

// GetUsageStats returns the singleton usage stats instance
func GetUsageStats() *UsageStats {
	statsOnce.Do(func() {
		usageStats = &UsageStats{
			Stats: make(map[string]*TemplateUsage),
		}
		usageStats.load()
	})
	return usageStats
}

// RecordLaunch records a template launch attempt
func (u *UsageStats) RecordLaunch(templateName string, success bool, launchTimeSeconds int) {
	u.mu.Lock()
	defer u.mu.Unlock()

	if u.Stats[templateName] == nil {
		u.Stats[templateName] = &TemplateUsage{
			TemplateName: templateName,
		}
	}

	stats := u.Stats[templateName]
	stats.LaunchCount++
	stats.LastUsed = time.Now()

	if success {
		stats.SuccessCount++
		if launchTimeSeconds > 0 {
			stats.TotalLaunchTime += launchTimeSeconds
			stats.AverageLaunchTime = stats.TotalLaunchTime / stats.SuccessCount
		}
	} else {
		stats.FailureCount++
	}

	// Calculate success rate
	totalAttempts := stats.SuccessCount + stats.FailureCount
	if totalAttempts > 0 {
		stats.SuccessRate = float64(stats.SuccessCount) / float64(totalAttempts)
	}

	// Save to disk asynchronously
	go u.save()
}

// GetPopularTemplates returns the most frequently used templates
func (u *UsageStats) GetPopularTemplates(limit int) []*TemplateUsage {
	u.mu.RLock()
	defer u.mu.RUnlock()

	// Create slice of all templates
	templates := make([]*TemplateUsage, 0, len(u.Stats))
	for _, stats := range u.Stats {
		templates = append(templates, stats)
	}

	// Sort by launch count (most popular first)
	for i := 0; i < len(templates)-1; i++ {
		for j := i + 1; j < len(templates); j++ {
			if templates[j].LaunchCount > templates[i].LaunchCount {
				templates[i], templates[j] = templates[j], templates[i]
			}
		}
	}

	// Return top N
	if limit > 0 && limit < len(templates) {
		return templates[:limit]
	}
	return templates
}

// GetRecentlyUsedTemplates returns templates used recently
func (u *UsageStats) GetRecentlyUsedTemplates(limit int) []*TemplateUsage {
	u.mu.RLock()
	defer u.mu.RUnlock()

	// Create slice of all templates
	templates := make([]*TemplateUsage, 0, len(u.Stats))
	for _, stats := range u.Stats {
		templates = append(templates, stats)
	}

	// Sort by last used time (most recent first)
	for i := 0; i < len(templates)-1; i++ {
		for j := i + 1; j < len(templates); j++ {
			if templates[j].LastUsed.After(templates[i].LastUsed) {
				templates[i], templates[j] = templates[j], templates[i]
			}
		}
	}

	// Return top N
	if limit > 0 && limit < len(templates) {
		return templates[:limit]
	}
	return templates
}

// GetTemplateUsage returns usage stats for a specific template
func (u *UsageStats) GetTemplateUsage(templateName string) *TemplateUsage {
	u.mu.RLock()
	defer u.mu.RUnlock()

	return u.Stats[templateName]
}

// getStatsPath returns the path to the usage stats file
func (u *UsageStats) getStatsPath() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return ""
	}

	return filepath.Join(homeDir, ".prism", "template_usage.json")
}

// load loads usage stats from disk
func (u *UsageStats) load() {
	statsPath := u.getStatsPath()
	if statsPath == "" {
		return
	}

	data, err := os.ReadFile(statsPath)
	if err != nil {
		// File doesn't exist yet, that's ok
		return
	}

	var loaded UsageStats
	if err := json.Unmarshal(data, &loaded); err != nil {
		// Corrupted file, start fresh
		return
	}

	u.Stats = loaded.Stats
	if u.Stats == nil {
		u.Stats = make(map[string]*TemplateUsage)
	}
}

// save saves usage stats to disk
func (u *UsageStats) save() {
	statsPath := u.getStatsPath()
	if statsPath == "" {
		return
	}

	// Ensure directory exists
	dir := filepath.Dir(statsPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return
	}

	u.mu.RLock()
	data, err := json.MarshalIndent(u, "", "  ")
	u.mu.RUnlock()

	if err != nil {
		return
	}

	// Write atomically
	tmpPath := statsPath + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0644); err != nil {
		return
	}

	os.Rename(tmpPath, statsPath)
}
