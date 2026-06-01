package kromgo

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/home-operations/kromgo/internal/config"
	"github.com/stretchr/testify/assert"
)

// --- isHidden ---

func TestIsHidden(t *testing.T) {
	cases := []struct {
		name          string
		metric        config.Metric
		defaultHidden *bool
		want          bool
	}{
		{"no global, no per-metric defaults to hidden", config.Metric{Name: "foo"}, nil, true},
		{"global false, no per-metric is visible", config.Metric{Name: "foo"}, new(false), false},
		{"global true, no per-metric is hidden", config.Metric{Name: "foo"}, new(true), true},
		{"per-metric true overrides global false", config.Metric{Name: "foo", Hidden: new(true)}, new(false), true},
		{"per-metric false overrides global true", config.Metric{Name: "foo", Hidden: new(false)}, new(true), false},
		{"per-metric false, no global", config.Metric{Name: "foo", Hidden: new(false)}, nil, false},
		{"per-metric true, no global", config.Metric{Name: "foo", Hidden: new(true)}, nil, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, isHidden(tc.metric, tc.defaultHidden))
		})
	}
}

// --- index ---

func newTestHandler(cfg config.KromgoConfig) *Handler {
	return &Handler{cfg: cfg}
}

func TestIndexHandler_AllHidden_ShowsIntentionallyBlank(t *testing.T) {
	h := newTestHandler(config.KromgoConfig{
		Metrics: []config.Metric{{Name: "cpu"}, {Name: "mem"}},
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

func TestIndexHandler_AllVisible_AllLinksPresent(t *testing.T) {
	h := newTestHandler(config.KromgoConfig{
		Metrics:  []config.Metric{{Name: "cpu"}, {Name: "mem"}},
		Defaults: config.Defaults{Hidden: new(false)},
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	h.index(w, req)

	body := w.Body.String()
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, body, `<a href="/cpu">cpu</a>`)
	assert.Contains(t, body, `<a href="/mem">mem</a>`)
	assert.NotContains(t, body, "page intentionally blank")
}

func TestIndexHandler_MixedVisibility(t *testing.T) {
	h := newTestHandler(config.KromgoConfig{
		Metrics: []config.Metric{
			{Name: "cpu", Hidden: new(false)},
			{Name: "mem"}, // hidden by global default
		},
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	h.index(w, req)

	body := w.Body.String()
	assert.Contains(t, body, `<a href="/cpu">cpu</a>`)
	assert.NotContains(t, body, `<a href="/mem">`)
}

func TestIndexHandler_GlobalFalse_PerMetricOverrideHidden(t *testing.T) {
	h := newTestHandler(config.KromgoConfig{
		Metrics: []config.Metric{
			{Name: "cpu"},
			{Name: "secret", Hidden: new(true)},
		},
		Defaults: config.Defaults{Hidden: new(false)},
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	h.index(w, req)

	body := w.Body.String()
	assert.Contains(t, body, `<a href="/cpu">cpu</a>`)
	assert.NotContains(t, body, `<a href="/secret">`)
}

func TestIndexHandler_NoMetrics_ShowsIntentionallyBlank(t *testing.T) {
	h := newTestHandler(config.KromgoConfig{
		Metrics:  []config.Metric{},
		Defaults: config.Defaults{Hidden: new(false)},
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	h.index(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "page intentionally blank")
	assert.NotContains(t, w.Body.String(), "<a href")
}
