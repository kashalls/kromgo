package kromgo

import (
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

// --- index ---

func newTestHandler(cfg config.KromgoConfig) *Handler {
	return &Handler{cfg: cfg}
}

func TestIndexHandler_AllHidden_ShowsIntentionallyBlank(t *testing.T) {
	h := newTestHandler(config.KromgoConfig{
		Badges: []config.Badge{{ID: "cpu"}, {ID: "mem"}},
		// Defaults.Hidden nil → defaults to true (hidden)
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	h.index(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "text/html; charset=utf-8", w.Header().Get("Content-Type"))
	assert.Contains(t, w.Body.String(), "page intentionally blank")
	assert.NotContains(t, w.Body.String(), "<a href")
}

func TestIndexHandler_BadgesAndGraphs_LinksPresent(t *testing.T) {
	h := newTestHandler(config.KromgoConfig{
		Badges:   []config.Badge{{ID: "cpu", Title: "CPU"}},
		Graphs:   []config.Graph{{ID: "cpu"}},
		Defaults: config.Defaults{Hidden: new(false)},
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	h.index(w, req)

	body := w.Body.String()
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, body, `<a href="/badges/cpu">CPU</a>`)
	assert.Contains(t, body, `<a href="/graphs/cpu">cpu</a>`)
	assert.NotContains(t, body, "page intentionally blank")
}

func TestIndexHandler_MixedVisibility(t *testing.T) {
	h := newTestHandler(config.KromgoConfig{
		Badges: []config.Badge{
			{ID: "cpu", Hidden: new(false)},
			{ID: "mem"}, // hidden by global default
		},
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	h.index(w, req)

	body := w.Body.String()
	assert.Contains(t, body, `<a href="/badges/cpu">cpu</a>`)
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

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	h.index(w, req)

	body := w.Body.String()
	assert.Contains(t, body, `<a href="/badges/cpu">cpu</a>`)
	assert.NotContains(t, body, `/badges/secret`)
}

func TestIndexHandler_NoEndpoints_ShowsIntentionallyBlank(t *testing.T) {
	h := newTestHandler(config.KromgoConfig{Defaults: config.Defaults{Hidden: new(false)}})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	h.index(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "page intentionally blank")
	assert.NotContains(t, w.Body.String(), "<a href")
}
