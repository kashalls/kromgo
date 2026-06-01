package kromgo

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/home-operations/kromgo/internal/config"
	"github.com/stretchr/testify/assert"
)

// boolPtr makes *bool literals concise in tests.
func boolPtr(b bool) *bool { return &b }

// --- isHidden ---

func TestIsHidden_NoGlobal_NoPerMetric_DefaultsTrue(t *testing.T) {
	m := config.Metric{Name: "foo"}
	assert.True(t, isHidden(m, nil))
}

func TestIsHidden_GlobalFalse_NoPerMetric_Visible(t *testing.T) {
	m := config.Metric{Name: "foo"}
	assert.False(t, isHidden(m, boolPtr(false)))
}

func TestIsHidden_GlobalTrue_NoPerMetric_Hidden(t *testing.T) {
	m := config.Metric{Name: "foo"}
	assert.True(t, isHidden(m, boolPtr(true)))
}

func TestIsHidden_GlobalFalse_PerMetricTrue_Hidden(t *testing.T) {
	m := config.Metric{Name: "foo", Hidden: boolPtr(true)}
	assert.True(t, isHidden(m, boolPtr(false)))
}

func TestIsHidden_GlobalTrue_PerMetricFalse_Visible(t *testing.T) {
	m := config.Metric{Name: "foo", Hidden: boolPtr(false)}
	assert.False(t, isHidden(m, boolPtr(true)))
}

func TestIsHidden_NoGlobal_PerMetricFalse_Visible(t *testing.T) {
	m := config.Metric{Name: "foo", Hidden: boolPtr(false)}
	assert.False(t, isHidden(m, nil))
}

func TestIsHidden_NoGlobal_PerMetricTrue_Hidden(t *testing.T) {
	m := config.Metric{Name: "foo", Hidden: boolPtr(true)}
	assert.True(t, isHidden(m, nil))
}

// --- index ---

func newTestHandler(cfg config.KromgoConfig) *Handler {
	return &Handler{cfg: cfg, metrics: cfg.MetricsByName()}
}

func TestIndexHandler_AllHidden_ShowsIntentionallyBlank(t *testing.T) {
	h := newTestHandler(config.KromgoConfig{
		Metrics: []config.Metric{{Name: "cpu"}, {Name: "mem"}},
		// HideAll nil → defaults to true
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
		Metrics: []config.Metric{{Name: "cpu"}, {Name: "mem"}},
		HideAll: boolPtr(false),
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
			{Name: "cpu", Hidden: boolPtr(false)},
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
			{Name: "secret", Hidden: boolPtr(true)},
		},
		HideAll: boolPtr(false),
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
		Metrics: []config.Metric{},
		HideAll: boolPtr(false),
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	h.index(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "page intentionally blank")
	assert.False(t, strings.Contains(w.Body.String(), "<a href"))
}
