package kromgo

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
)

// Response MIME types.
const (
	mimeJSON = "application/json"
	mimeSVG  = "image/svg+xml"
	mimePNG  = "image/png"
	mimeHTML = "text/html; charset=utf-8"
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
	w.Header().Set("Content-Type", mimeJSON)
	_, _ = w.Write(body)
	return nil
}

func writeSVG(w http.ResponseWriter, svg []byte) {
	w.Header().Set("Content-Type", mimeSVG)
	_, _ = w.Write(svg)
}

// setCache applies a successful response's cache policy; writeError later overrides
// it with no-store on failures.
func setCache(w http.ResponseWriter, cacheSeconds int) {
	if cacheSeconds > 0 {
		w.Header().Set("Cache-Control", fmt.Sprintf("public, max-age=%d", cacheSeconds))
	}
}

// writeJSONOr writes v as JSON, falling back to a 500 error response on marshal failure.
func writeJSONOr(w http.ResponseWriter, log *slog.Logger, id string, v any) {
	if err := writeJSON(w, v); err != nil {
		log.Error("error writing json response", "error", err)
		writeError(w, id, "Error", http.StatusInternalServerError)
	}
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
	w.Header().Set("Content-Type", mimeJSON)
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(code)
	_, _ = w.Write(body)
}
