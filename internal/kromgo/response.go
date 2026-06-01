package kromgo

import (
	"encoding/json"
	"net/http"
)

// EndpointResponse is the shields.io-compatible JSON envelope returned for both
// successful metric responses and errors.
type EndpointResponse struct {
	SchemaVersion int    `json:"schemaVersion"`
	Label         string `json:"label"`
	Message       string `json:"message"`
	Color         string `json:"color,omitempty"`
	Error         bool   `json:"isError,omitempty"`
	CacheSeconds  int    `json:"cacheSeconds,omitempty"`
}

func writeJSON(w http.ResponseWriter, v any) error {
	body, err := json.Marshal(v)
	if err != nil {
		return err
	}
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write(body)
	return nil
}

func writeSVG(w http.ResponseWriter, svg []byte) {
	w.Header().Set("Content-Type", "image/svg+xml")
	_, _ = w.Write(svg)
}

// writeError writes a shields.io-compatible error response with the given status code.
// Errors are never cached, even if a caller set a Cache-Control header earlier.
func writeError(w http.ResponseWriter, metric, reason string, code int) {
	body, err := json.Marshal(EndpointResponse{
		SchemaVersion: 1,
		Label:         metric,
		Message:       reason,
		Error:         true,
	})
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(code)
	_, _ = w.Write(body)
}
