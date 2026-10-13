package daemon

import (
	"net/http"
	"strconv"
	"time"
)

// handleAuditEvents queries the operational audit log.
// GET /api/v1/audit/events?limit=100&action=instance.launch&since=2026-01-01T00:00:00Z
func (s *Server) handleAuditEvents(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	q := r.URL.Query()

	limit := 100
	if l := q.Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 {
			limit = parsed
		}
	}

	action := q.Get("action")

	var since time.Time
	if s := q.Get("since"); s != "" {
		if t, err := time.Parse(time.RFC3339, s); err == nil {
			since = t
		}
	}

	events, err := s.securityManager.QueryOperationalEvents(limit, action, since)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/json")
	encodeJSONResponse(w, map[string]interface{}{
		"events": events,
		"count":  len(events),
	})
}
