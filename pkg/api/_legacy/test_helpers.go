package api

import (
	"encoding/json"
	"io"
	"net/http"
)

// Helper functions for testing

// renderJSON renders a JSON response
func renderJSON(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}

// decodeJSON decodes a JSON request
func decodeJSON(r *http.Request, v interface{}) error {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return err
	}
	defer r.Body.Close()
	
	return json.Unmarshal(body, v)
}