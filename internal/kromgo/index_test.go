package kromgo

import (
	"crypto/tls"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/home-operations/kromgo/internal/config"
	"github.com/stretchr/testify/assert"
)

// --- hidden ---

func TestHidden(t *testing.T) {
	cases := []struct {
		name string
		item *bool
		def  *bool
		want bool
	}{
		{"no item, no default → hidden", nil, nil, true},
		{"default false, no item → visible", nil, new(false), false},
		{"default true, no item → hidden", nil, new(true), true},
		{"item true over default false → hidden", new(true), new(false), true},
		{"item false over default true → visible", new(false), new(true), false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, hidden(tc.item, tc.def))
		})
	}
}

// --- baseURL ---

func TestBaseURL(t *testing.T) {
	req := func(host, xfp string, withTLS bool) *http.Request {
		r := httptest.NewRequest(http.MethodGet, "/", nil)
		r.Host = host
		if xfp != "" {
			r.Header.Set("X-Forwarded-Proto", xfp)
		}
		if withTLS {
			r.TLS = &tls.ConnectionState{}
		}
		return r
	}
	cases := []struct {
		name string
		r    *http.Request
		want string
	}{
		{"plain http", req("example.com", "", false), "http://example.com"},
		{"host with port", req("example.com:8080", "", false), "http://example.com:8080"},
		{"forwarded https", req("example.com", "https", false), "https://example.com"},
		{"forwarded proto list takes first", req("example.com", "https, http", false), "https://example.com"},
		{"tls connection", req("example.com", "", true), "https://example.com"},
		{"ipv6 literal", req("[::1]:8080", "", false), "http://[::1]:8080"},
		{"invalid host → relative", req("ex ample.com", "", false), ""},
		{"injection host → relative", req("evil.com)![x](http://x", "", false), ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, baseURL(tc.r))
		})
	}
}

// --- markdownItem / mdEscapeAlt ---

func TestMarkdownItem(t *testing.T) {
	assert.Equal(t, "![CPU](http://example.com/badges/cpu)",
		markdownItem("http://example.com", "badges", "cpu", "CPU").Markdown)
	// Relative URL when the host is unusable.
	assert.Equal(t, "![CPU](/badges/cpu)", markdownItem("", "badges", "cpu", "CPU").Markdown)
	// Brackets in the alt text are escaped so they can't break the image syntax.
	assert.Equal(t, `![a\[b\]](http://h/graphs/g)`, markdownItem("http://h", "graphs", "g", "a[b]").Markdown)
}

// --- index handler ---

func newTestHandler(cfg config.KromgoConfig) *Handler {
	return &Handler{cfg: cfg}
}

func getIndex(h *Handler) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, "/", nil) // Host defaults to example.com
	w := httptest.NewRecorder()
	h.index(w, req)
	return w
}

func TestIndexHandler_GalleryHeadersAndAssets(t *testing.T) {
	h := newTestHandler(config.KromgoConfig{
		Badges:   []config.Badge{{ID: "cpu", Title: "CPU"}},
		Defaults: config.Defaults{Hidden: new(false)},
	})
	w := getIndex(h)
	body := w.Body.String()

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, mimeHTML, w.Header().Get("Content-Type"))
	// CSP is relaxed for the page but stays free of unsafe-inline/eval and CDNs.
	csp := w.Header().Get("Content-Security-Policy")
	assert.Contains(t, csp, "script-src 'self'")
	assert.Contains(t, csp, "frame-ancestors 'none'")
	assert.NotContains(t, csp, "unsafe-inline")
	assert.NotContains(t, csp, "unsafe-eval")
	// Per-Host page must not be shared-cached.
	assert.Equal(t, "no-store", w.Header().Get("Cache-Control"))
	// Self-hosted assets, no external origins.
	assert.Contains(t, body, `/assets/marked.min.js`)
	assert.Contains(t, body, `/assets/github-markdown.css`)
	assert.Contains(t, body, `/assets/gallery.js`)
	assert.NotContains(t, body, "cdn.")
}

func TestIndexHandler_BadgesAndGraphs_SnippetsPresent(t *testing.T) {
	h := newTestHandler(config.KromgoConfig{
		Badges:   []config.Badge{{ID: "cpu", Title: "CPU"}},
		Graphs:   []config.Graph{{ID: "cpu"}}, // no title → falls back to id
		Defaults: config.Defaults{Hidden: new(false)},
	})
	body := getIndex(h).Body.String()

	assert.Contains(t, body, `![CPU](http://example.com/badges/cpu)`)
	assert.Contains(t, body, `![cpu](http://example.com/graphs/cpu)`)
	assert.Contains(t, body, "<h2>Badges</h2>")
	assert.Contains(t, body, "<h2>Graphs</h2>")
	assert.Contains(t, body, `class="copy"`)
	assert.NotContains(t, body, "No endpoints are visible")
}

func TestIndexHandler_AllHidden_ShowsEmptyState(t *testing.T) {
	h := newTestHandler(config.KromgoConfig{
		Badges: []config.Badge{{ID: "cpu"}, {ID: "mem"}},
		// Defaults.Hidden nil → defaults to true (hidden)
	})
	body := getIndex(h).Body.String()

	assert.Contains(t, body, "No endpoints are visible")
	assert.NotContains(t, body, "/badges/cpu)")
	assert.NotContains(t, body, "<h2>Badges</h2>")
}

func TestIndexHandler_MixedVisibility(t *testing.T) {
	h := newTestHandler(config.KromgoConfig{
		Badges: []config.Badge{
			{ID: "cpu", Hidden: new(false)},
			{ID: "mem"}, // hidden by global default
		},
	})
	body := getIndex(h).Body.String()

	assert.Contains(t, body, `![cpu](http://example.com/badges/cpu)`)
	assert.NotContains(t, body, `/badges/mem`)
}

func TestIndexHandler_GlobalFalse_PerEndpointOverrideHidden(t *testing.T) {
	h := newTestHandler(config.KromgoConfig{
		Badges: []config.Badge{
			{ID: "cpu"},
			{ID: "secret", Hidden: new(true)},
		},
		Defaults: config.Defaults{Hidden: new(false)},
	})
	body := getIndex(h).Body.String()

	assert.Contains(t, body, `![cpu](http://example.com/badges/cpu)`)
	assert.NotContains(t, body, `/badges/secret`)
}

func TestIndexHandler_NoEndpoints_ShowsEmptyState(t *testing.T) {
	h := newTestHandler(config.KromgoConfig{Defaults: config.Defaults{Hidden: new(false)}})
	body := getIndex(h).Body.String()
	assert.Contains(t, body, "No endpoints are visible")
}

func TestIndexHandler_GalleryDisabled_ShowsLanding(t *testing.T) {
	h := newTestHandler(config.KromgoConfig{
		Badges:   []config.Badge{{ID: "cpu", Title: "CPU"}},
		Defaults: config.Defaults{Gallery: new(false), Hidden: new(false)},
	})
	w := getIndex(h)
	body := w.Body.String()

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, mimeHTML, w.Header().Get("Content-Type"))
	assert.Contains(t, body, `class="landing"`)
	// The landing page lists no endpoints and renders no gallery grid.
	assert.NotContains(t, body, "<h2>Badges</h2>")
	assert.NotContains(t, body, "/badges/cpu")
}

// --- assets handler ---

func TestAssetsHandler(t *testing.T) {
	get := func(path string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		w := httptest.NewRecorder()
		assetsHandler().ServeHTTP(w, req)
		return w
	}

	css := get("/assets/gallery.css")
	assert.Equal(t, http.StatusOK, css.Code)
	assert.Contains(t, css.Header().Get("Content-Type"), "text/css")
	assert.Contains(t, css.Header().Get("Cache-Control"), "max-age=")
	assert.Contains(t, css.Body.String(), ".grid")

	js := get("/assets/marked.min.js")
	assert.Equal(t, http.StatusOK, js.Code)
	assert.Contains(t, js.Header().Get("Content-Type"), "javascript")

	assert.Equal(t, http.StatusNotFound, get("/assets/ATTRIBUTION.md").Code, "non-embedded files are not served")
	assert.Equal(t, http.StatusNotFound, get("/assets/nope.txt").Code)
	// No directory listing, and a traversal attempt cannot escape the embedded FS.
	assert.Equal(t, http.StatusNotFound, get("/assets/").Code, "no directory listing")
	assert.NotEqual(t, http.StatusOK, get("/assets/../handler.go").Code, "no path traversal")
}
