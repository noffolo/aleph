package handler

import (
	"encoding/json"
	"net/http"
)

// writeError sends a JSON error response with the given HTTP status code.
// The response body is {"error": "message"}.
func writeError(w http.ResponseWriter, message string, code int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]string{"error": message})
}

// writeJSON sends a JSON response with the given HTTP status code and body.
func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}
