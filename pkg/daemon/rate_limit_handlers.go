// Package daemon provides rate limiting API handlers
package daemon

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// handleRateLimitStatus returns the current rate limit status
// GET /api/v1/rate-limit/status
func (s *Server) handleRateLimitStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	if s.rateLimiter == nil {
		s.writeError(w, http.StatusServiceUnavailable, "Rate limiter not initialized")
		return
	}

	status := s.rateLimiter.GetStatus()

	// Calculate percentage of quota used
	var quotaUsedPercent float64
	if status.MaxLaunches > 0 {
		quotaUsedPercent = float64(status.Current) / float64(status.MaxLaunches) * 100
	}

	response := map[string]interface{}{
		"enabled":            status.Enabled,
		"max_launches":       status.MaxLaunches,
		"window_minutes":     int(status.Window.Minutes()),
		"current_launches":   status.Current,
		"remaining_launches": status.Remaining,
		"quota_used_percent": quotaUsedPercent,
	}

	// Add reset time if available
	if !status.ResetTime.IsZero() {
		response["reset_time"] = status.ResetTime
		response["seconds_until_reset"] = int(time.Until(status.ResetTime).Seconds())
	}

	w.Header().Set("Content-Type", "application/json")
	encodeJSONResponse(w, response)
}

// handleRateLimitConfigure updates rate limiter configuration
// POST /api/v1/rate-limit/configure
func (s *Server) handleRateLimitConfigure(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		s.writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	if s.rateLimiter == nil {
		s.writeError(w, http.StatusServiceUnavailable, "Rate limiter not initialized")
		return
	}

	var req struct {
		MaxLaunches   *int  `json:"max_launches"`
		WindowMinutes *int  `json:"window_minutes"`
		Enabled       *bool `json:"enabled"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, "Invalid JSON request body")
		return
	}

	// Validate configuration
	if req.MaxLaunches != nil {
		if *req.MaxLaunches < 1 || *req.MaxLaunches > 100 {
			s.writeError(w, http.StatusBadRequest, "max_launches must be between 1 and 100")
			return
		}
	}

	if req.WindowMinutes != nil {
		if *req.WindowMinutes < 1 || *req.WindowMinutes > 60 {
			s.writeError(w, http.StatusBadRequest, "window_minutes must be between 1 and 60")
			return
		}
	}

	// Apply configuration changes
	if req.Enabled != nil {
		s.rateLimiter.SetEnabled(*req.Enabled)
	}

	if req.MaxLaunches != nil && req.WindowMinutes != nil {
		window := time.Duration(*req.WindowMinutes) * time.Minute
		s.rateLimiter.UpdateConfig(*req.MaxLaunches, window)
	} else if req.MaxLaunches != nil {
		// Update max launches only, keep existing window
		status := s.rateLimiter.GetStatus()
		s.rateLimiter.UpdateConfig(*req.MaxLaunches, status.Window)
	} else if req.WindowMinutes != nil {
		// Update window only, keep existing max launches
		status := s.rateLimiter.GetStatus()
		window := time.Duration(*req.WindowMinutes) * time.Minute
		s.rateLimiter.UpdateConfig(status.MaxLaunches, window)
	}

	// Return updated status
	status := s.rateLimiter.GetStatus()
	response := map[string]interface{}{
		"enabled":          status.Enabled,
		"max_launches":     status.MaxLaunches,
		"window_minutes":   int(status.Window.Minutes()),
		"current_launches": status.Current,
		"remaining":        status.Remaining,
		"message":          fmt.Sprintf("Rate limiter updated: %d launches per %d minute(s), enabled=%v", status.MaxLaunches, int(status.Window.Minutes()), status.Enabled),
	}

	w.Header().Set("Content-Type", "application/json")
	encodeJSONResponse(w, response)
}

// handleRateLimitReset resets the rate limiter state
// POST /api/v1/rate-limit/reset
func (s *Server) handleRateLimitReset(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		s.writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	if s.rateLimiter == nil {
		s.writeError(w, http.StatusServiceUnavailable, "Rate limiter not initialized")
		return
	}

	// Reset by creating a new rate limiter with same configuration
	status := s.rateLimiter.GetStatus()
	s.rateLimiter = NewRateLimiter(status.MaxLaunches, status.Window)
	s.rateLimiter.SetEnabled(status.Enabled)

	response := map[string]interface{}{
		"message":          "Rate limiter reset successfully",
		"max_launches":     status.MaxLaunches,
		"window_minutes":   int(status.Window.Minutes()),
		"enabled":          status.Enabled,
		"current_launches": 0,
		"remaining":        status.MaxLaunches,
	}

	w.Header().Set("Content-Type", "application/json")
	encodeJSONResponse(w, response)
}
