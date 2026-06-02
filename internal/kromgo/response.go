package kromgo

import (
	"cmp"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/home-operations/kromgo/internal/config"
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
	LabelColor    string `json:"labelColor,omitempty"`
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

// defaultCacheMaxAge is the Cache-Control max-age / s-maxage (in seconds) applied
// when caching is enabled and cache.maxAge is unset.
const defaultCacheMaxAge = 300

// cachePolicy is the global, precomputed Cache-Control policy: the header value sent
// on successful responses and the cacheSeconds advertised in the shields.io JSON. It
// is resolved once from config (see resolveCache), the same for every endpoint.
type cachePolicy struct {
	control string // Cache-Control header value for successful responses
	seconds int    // cacheSeconds reported in the shields.io JSON; 0 when caching is off
}

// resolveCache turns the global cache config into a fixed policy. Caching is on by
// default; enabled: false sends an explicit no-store. Sending no header would NOT
// mean "no caching" — it lets GitHub's camo proxy / CDNs apply their own aggressive
// default, which is why badges go stale. A header still isn't a hard guarantee
// against camo (badges/shields#221), but it's the strongest signal we can send.
func resolveCache(c config.Cache) cachePolicy {
	if c.Enabled != nil && !*c.Enabled {
		return cachePolicy{control: "no-cache, no-store, must-revalidate, max-age=0"}
	}
	maxAge := cmp.Or(c.MaxAge, defaultCacheMaxAge)
	// max-age governs browser caches; s-maxage governs shared caches (CDNs and GitHub's
	// camo image proxy) — the ones that cache README badges. shields.io sets both.
	return cachePolicy{
		control: fmt.Sprintf("public, max-age=%d, s-maxage=%d", maxAge, maxAge),
		seconds: maxAge,
	}
}

// apply sets the resolved Cache-Control header on a successful response; writeError
// later overrides it with no-store on failures.
func (p cachePolicy) apply(w http.ResponseWriter) {
	w.Header().Set("Cache-Control", p.control)
}

// writeJSONOr writes v as JSON, falling back to a 500 error response on marshal failure.
func writeJSONOr(w http.ResponseWriter, log *slog.Logger, id string, v any) {
	if err := writeJSON(w, v); err != nil {
		log.Error("error writing json response", "error", err)
		writeError(w, id, "Error", http.StatusInternalServerError)
	}
}

// errorResponse renders a failure. For an SVG request it returns a self-describing
// error badge with HTTP 200 — so an <img> shows the error instead of a broken-image
// icon — colored red for client errors (4xx) and grey for server/upstream (5xx).
// Other formats get the JSON error with its status code. Errors are never cached.
func (h *Handler) errorResponse(w http.ResponseWriter, format, id, reason string, code int) {
	if format != formatSVG {
		writeError(w, id, reason, code)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	writeSVG(w, h.gen.renderError(id, reason, code))
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
