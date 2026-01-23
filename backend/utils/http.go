package utils

import (
	"encoding/json"
	"log/slog"
	"net/http"
)

// WriteJSONResponse writes a JSON response with the given status code and data.
func WriteJSONResponse(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		slog.Error("failed to encode JSON response", "error", err)
	}
}

// WriteJSONError writes a JSON error response with the given status code and message.
func WriteJSONError(w http.ResponseWriter, status int, message string) {
	WriteJSONResponse(w, status, map[string]string{
		"error": message,
	})
}
