package kromgo

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/home-operations/kromgo/internal/config"
	"github.com/home-operations/kromgo/internal/prometheus"
	"github.com/home-operations/kromgo/internal/promtest"
	promclient "github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockProm returns a mock Prometheus serving the given instant value (labelled
// job=node) and range matrix.
func mockProm(t *testing.T, vectorValue string, matrixValues []float64) *httptest.Server {
	t.Helper()
	return promtest.Server(t, promtest.Scalar(vectorValue, map[string]string{"job": "node"}), matrixValues)
}

func newHandlerForTest(t *testing.T, cfg config.KromgoConfig, srvURL string) *Handler {
	t.Helper()
	client, err := prometheus.New(srvURL, 0)
	require.NoError(t, err)
	h, err := New(cfg, client)
	require.NoError(t, err)
	return h
}

// baseConfig has one badge and one graph, both id "cpu".
func baseConfig() config.KromgoConfig {
	return config.KromgoConfig{
		Badges: []config.Badge{{
			ID:        "cpu",
			Query:     "node_cpu_usage",
			ValueExpr: `string(result) + "%"`,
			ColorExpr: `result <= 50.0 ? "green" : "red"`,
		}},
		Graphs: []config.Graph{{ID: "cpu", Query: "node_cpu_usage", MaxDuration: "24h"}},
	}
}

func counterValue(t *testing.T, c promclient.Counter) float64 {
	t.Helper()
	var m dto.Metric
	require.NoError(t, c.Write(&m))
	return m.GetCounter().GetValue()
}

// assertSVGOK asserts a 200 SVG image response.
func assertSVGOK(t *testing.T, w *httptest.ResponseRecorder) {
	t.Helper()
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "image/svg+xml", w.Header().Get("Content-Type"))
	assert.True(t, strings.HasPrefix(w.Body.String(), "<svg"))
}

// An unknown badge path must fold into the bounded ("badge","unknown","svg")
// counter series rather than minting new label values. Kept serial (no t.Parallel):
// the before/after delta is measured before the package's parallel tests run.
func TestServeBadge_CounterCardinalityBounded(t *testing.T) {
	srv := mockProm(t, "1", nil)
	h := newHandlerForTest(t, baseConfig(), srv.URL)

	before := counterValue(t, requestsTotal.WithLabelValues("badge", "unknown", "svg"))
	promtest.Get(t, h.Mux(), "/badges/does-not-exist")
	promtest.Get(t, h.Mux(), "/badges/also-missing")
	after := counterValue(t, requestsTotal.WithLabelValues("badge", "unknown", "svg"))

	assert.Equal(t, before+2, after)
}

// TestServeBadge_Output covers the badge output formats that share the standard
// config and instant value (17.5, labelled job=node).
func TestServeBadge_Output(t *testing.T) {
	t.Parallel()
	srv := mockProm(t, "17.5", nil)
	h := newHandlerForTest(t, baseConfig(), srv.URL)

	cases := []struct {
		name  string
		path  string
		check func(t *testing.T, w *httptest.ResponseRecorder)
	}{
		{"svg default", "/badges/cpu", assertSVGOK},
		{"style flat-square", "/badges/cpu?style=flat-square", assertSVGOK},
		{"style plastic", "/badges/cpu?style=plastic", assertSVGOK},
		{"unknown style falls back to svg", "/badges/cpu?style=", assertSVGOK},
		{"shields json", "/badges/cpu?format=shields", func(t *testing.T, w *httptest.ResponseRecorder) {
			assert.Equal(t, http.StatusOK, w.Code)
			assert.Equal(t, "application/json", w.Header().Get("Content-Type"))
			var body map[string]any
			require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
			assert.EqualValues(t, 1, body["schemaVersion"])
			assert.Equal(t, "cpu", body["label"])
			assert.Equal(t, "17.5%", body["message"])
			assert.Equal(t, "green", body["color"])
		}},
		{"native json", "/badges/cpu?format=json", func(t *testing.T, w *httptest.ResponseRecorder) {
			assert.Equal(t, http.StatusOK, w.Code)
			assert.Equal(t, "application/json", w.Header().Get("Content-Type"))
			var body BadgeJSON
			require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
			assert.Equal(t, "cpu", body.ID)
			assert.Equal(t, "17.5%", body.Value)
			assert.Equal(t, "green", body.Color)
			require.NotNil(t, body.Result)
			assert.InDelta(t, 17.5, *body.Result, 0.001)
			assert.Equal(t, "node", body.Labels["job"])
		}},
		{"not found svg", "/badges/does-not-exist", func(t *testing.T, w *httptest.ResponseRecorder) {
			// Default (svg) format renders a graceful error badge with HTTP 200, so an
			// <img> shows the error rather than a broken-image icon. 4xx → red.
			assert.Equal(t, http.StatusOK, w.Code)
			assert.Equal(t, "image/svg+xml", w.Header().Get("Content-Type"))
			assert.Equal(t, "no-store", w.Header().Get("Cache-Control"))
			assert.Contains(t, w.Body.String(), `aria-label="does-not-exist: Not Found"`)
			assert.Contains(t, w.Body.String(), "#e05d44") // red message segment
		}},
		{"not found json", "/badges/does-not-exist?format=json", func(t *testing.T, w *httptest.ResponseRecorder) {
			// Non-svg formats keep the JSON error and its status code.
			assert.Equal(t, http.StatusNotFound, w.Code)
			assert.Contains(t, w.Body.String(), `"isError":true`)
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			tc.check(t, promtest.Get(t, h.Mux(), tc.path))
		})
	}
}

func TestServeBadge_Icon(t *testing.T) {
	t.Parallel()
	cfg := config.KromgoConfig{Badges: []config.Badge{{
		ID: "cpu", Query: "q", Icon: "mdi:server-outline", ValueExpr: `string(result) + "%"`,
	}}}
	srv := mockProm(t, "17.5", nil)
	h := newHandlerForTest(t, cfg, srv.URL)

	w := promtest.Get(t, h.Mux(), "/badges/cpu")

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "image/svg+xml", w.Header().Get("Content-Type"))
	assert.Contains(t, w.Body.String(), mdiIcons()["server-outline"], "icon path embedded")
}

func TestNew_InvalidIconFailsFast(t *testing.T) {
	t.Parallel()
	cfg := config.KromgoConfig{Badges: []config.Badge{{ID: "x", Query: "q", Icon: "mdi:does-not-exist"}}}
	srv := mockProm(t, "1", nil)
	client, err := prometheus.New(srv.URL, 0)
	require.NoError(t, err)

	_, err = New(cfg, client)
	assert.Error(t, err)
}

func TestRoutes_NonGETRejected(t *testing.T) {
	t.Parallel()
	srv := mockProm(t, "17.5", nil)
	h := newHandlerForTest(t, baseConfig(), srv.URL)

	for _, method := range []string{http.MethodPost, http.MethodPut, http.MethodDelete} {
		t.Run(method, func(t *testing.T) {
			t.Parallel()
			req := httptest.NewRequest(method, "/badges/cpu", nil)
			w := httptest.NewRecorder()
			h.Mux().ServeHTTP(w, req)
			assert.Equal(t, http.StatusMethodNotAllowed, w.Code)
		})
	}
}

// TestServeBadge_Expressions covers value/color CEL evaluation, including the
// no-data and runtime-error paths, each with its own config.
func TestServeBadge_Expressions(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name     string
		badge    config.Badge
		sample   []promtest.Sample
		matrix   []float64
		path     string
		wantCode int
		contains []string
	}{
		{
			name:     "humanize duration value",
			badge:    config.Badge{ID: "uptime", Query: "q", ValueExpr: "humanizeDuration(result)"},
			sample:   promtest.Scalar("9000", map[string]string{"job": "node"}),
			path:     "/badges/uptime?format=json",
			wantCode: http.StatusOK,
			contains: []string{`"value":"2h30m"`},
		},
		{
			name:     "value from label",
			badge:    config.Badge{ID: "ver", Query: "q", ValueExpr: `labels["version"]`},
			sample:   promtest.Scalar("0", map[string]string{"version": "v1.2.3"}),
			path:     "/badges/ver?format=shields",
			wantCode: http.StatusOK,
			contains: []string{`"message":"v1.2.3"`},
		},
		{
			// Indexing a missing label is a CEL runtime error. In JSON it surfaces as a
			// 500 error response (the svg error badge is covered in TestServeBadge_Output).
			name:     "missing label is runtime error",
			badge:    config.Badge{ID: "ver", Query: "q", ValueExpr: `labels["version"]`},
			sample:   promtest.Scalar("0", map[string]string{"other": "x"}),
			path:     "/badges/ver?format=json",
			wantCode: http.StatusInternalServerError,
			contains: []string{`"isError":true`},
		},
		{
			name: "state expressions",
			badge: config.Badge{
				ID:        "ceph",
				Query:     "q",
				ValueExpr: `result == 0.0 ? "Healthy" : "Critical"`,
				ColorExpr: `result == 0.0 ? "green" : "red"`,
			},
			sample:   promtest.Scalar("0", map[string]string{"job": "node"}),
			path:     "/badges/ceph?format=shields",
			wantCode: http.StatusOK,
			contains: []string{`"message":"Healthy"`, `"color":"green"`},
		},
		{
			name:     "empty vector renders no data",
			badge:    baseConfig().Badges[0],
			sample:   nil, // empty instant vector
			path:     "/badges/cpu?format=shields",
			wantCode: http.StatusOK,
			contains: []string{`"message":"no data"`},
		},
		{
			// type: range reduces a range query to one value (avg of 10,20,30 = 20).
			name: "range type reduces to one value",
			badge: config.Badge{
				ID:        "cpu_avg",
				Type:      config.TypeRange,
				Query:     "q",
				ValueExpr: `string(result) + "%"`,
				Range:     &config.RangeQuery{Last: "1h", Reduce: config.ReduceAvg},
			},
			matrix:   []float64{10, 20, 30},
			path:     "/badges/cpu_avg?format=shields",
			wantCode: http.StatusOK,
			contains: []string{`"message":"20%"`},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			srv := promtest.Server(t, tc.sample, tc.matrix)
			h := newHandlerForTest(t, config.KromgoConfig{Badges: []config.Badge{tc.badge}}, srv.URL)

			w := promtest.Get(t, h.Mux(), tc.path)

			assert.Equal(t, tc.wantCode, w.Code)
			for _, want := range tc.contains {
				assert.Contains(t, w.Body.String(), want)
			}
		})
	}
}

func TestCacheControl(t *testing.T) {
	t.Parallel()

	// Caching is global (not per endpoint) and on by default.
	t.Run("enabled", func(t *testing.T) {
		t.Parallel()
		cases := []struct {
			name      string
			cache     config.Cache
			wantCache string
			wantBody  string
		}{
			{"on by default at the default max-age", config.Cache{}, "public, max-age=300, s-maxage=300", `"cacheSeconds":300`},
			{"custom max-age", config.Cache{MaxAge: 3600}, "public, max-age=3600, s-maxage=3600", `"cacheSeconds":3600`},
			{"enabled with max-age unset falls back to default", config.Cache{Enabled: new(true)}, "public, max-age=300, s-maxage=300", `"cacheSeconds":300`},
		}
		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				cfg := config.KromgoConfig{Cache: tc.cache, Badges: []config.Badge{{ID: "cpu", Query: "q"}}}
				srv := mockProm(t, "1", nil)
				h := newHandlerForTest(t, cfg, srv.URL)
				w := promtest.Get(t, h.Mux(), "/badges/cpu?format=shields")
				assert.Equal(t, tc.wantCache, w.Header().Get("Cache-Control"))
				assert.Contains(t, w.Body.String(), tc.wantBody)
			})
		}
	})

	t.Run("disabled sends no-store", func(t *testing.T) {
		t.Parallel()
		cfg := config.KromgoConfig{
			Cache:  config.Cache{Enabled: new(false)},
			Badges: []config.Badge{{ID: "cpu", Query: "q"}},
		}
		srv := mockProm(t, "17.5", nil)
		h := newHandlerForTest(t, cfg, srv.URL)
		w := promtest.Get(t, h.Mux(), "/badges/cpu?format=shields")
		// enabled: false sends an explicit no-store (not an empty header) so camo/CDNs don't cache.
		assert.Equal(t, "no-cache, no-store, must-revalidate, max-age=0", w.Header().Get("Cache-Control"))
		// cacheSeconds is 0 (omitempty) in the JSON when caching is off.
		assert.NotContains(t, w.Body.String(), "cacheSeconds")
	})

	t.Run("errors not cached", func(t *testing.T) {
		t.Parallel()
		cfg := config.KromgoConfig{
			Graphs: []config.Graph{{ID: "cpu", Query: "q", MaxDuration: "24h"}},
		}
		srv := mockProm(t, "1", nil)
		h := newHandlerForTest(t, cfg, srv.URL)

		// Window exceeds maxDuration → 400; the cache header must be no-store, not max-age.
		w := promtest.Get(t, h.Mux(), "/graphs/cpu?format=json&last=7d")

		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Equal(t, "no-store", w.Header().Get("Cache-Control"))
	})
}

// TestServeGraph_Output covers the graph output formats; each case supplies its own
// range matrix and request.
func TestServeGraph_Output(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name   string
		matrix []float64
		path   string
		check  func(t *testing.T, w *httptest.ResponseRecorder)
	}{
		{"json", []float64{1, 2, 3}, "/graphs/cpu?format=json&last=1h", func(t *testing.T, w *httptest.ResponseRecorder) {
			assert.Equal(t, http.StatusOK, w.Code)
			assert.Equal(t, "application/json", w.Header().Get("Content-Type"))
			var resp HistoryResponse
			require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
			assert.Equal(t, "cpu", resp.ID)
			require.Len(t, resp.Series, 1)
			assert.Len(t, resp.Series[0].Data, 3)
		}},
		{"svg default", []float64{10, 20, 15, 30}, "/graphs/cpu?last=1h", assertSVGOK},
		{"png", []float64{10, 20, 15, 30}, "/graphs/cpu?format=png&last=1h", func(t *testing.T, w *httptest.ResponseRecorder) {
			assert.Equal(t, http.StatusOK, w.Code)
			assert.Equal(t, "image/png", w.Header().Get("Content-Type"))
			assert.Equal(t, []byte{0x89, 'P', 'N', 'G'}, w.Body.Bytes()[:4])
		}},
		{"theme", []float64{10, 20, 15, 30}, "/graphs/cpu?theme=dracula&last=1h", func(t *testing.T, w *httptest.ResponseRecorder) {
			assert.Equal(t, http.StatusOK, w.Code)
			assert.Contains(t, w.Body.String(), "rgb(40,42,54)") // dracula background
		}},
		{"window too large", []float64{1, 2}, "/graphs/cpu?format=json&last=7d", func(t *testing.T, w *httptest.ResponseRecorder) {
			assert.Equal(t, http.StatusBadRequest, w.Code)
		}},
		{"not found", nil, "/graphs/nope", func(t *testing.T, w *httptest.ResponseRecorder) {
			// svg (default) → graceful error badge with HTTP 200, not a broken image.
			assert.Equal(t, http.StatusOK, w.Code)
			assert.Equal(t, "image/svg+xml", w.Header().Get("Content-Type"))
			assert.Contains(t, w.Body.String(), `aria-label="nope: Not Found"`)
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			srv := mockProm(t, "0", tc.matrix)
			h := newHandlerForTest(t, baseConfig(), srv.URL)
			tc.check(t, promtest.Get(t, h.Mux(), tc.path))
		})
	}
}

func TestIndexRoute(t *testing.T) {
	t.Parallel()
	cfg := baseConfig() // endpoints are shown in the gallery by default
	srv := mockProm(t, "0", nil)
	h := newHandlerForTest(t, cfg, srv.URL)

	w := promtest.Get(t, h.Mux(), "/")

	assert.Equal(t, http.StatusOK, w.Code)
	body := w.Body.String()
	assert.Contains(t, body, `![cpu](http://example.com/badges/cpu)`)
	assert.Contains(t, body, `![cpu](http://example.com/graphs/cpu)`)
}

func TestAssetsRoute(t *testing.T) {
	t.Parallel()
	srv := mockProm(t, "0", nil)
	h := newHandlerForTest(t, baseConfig(), srv.URL)

	w := promtest.Get(t, h.Mux(), "/assets/gallery.js")

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Header().Get("Content-Type"), "javascript")
}
