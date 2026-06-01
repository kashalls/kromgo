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
			ID:    "cpu",
			Query: "node_cpu_usage",
			Value: `string(result) + "%"`,
			Color: `result <= 50.0 ? "green" : "red"`,
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

// An unknown badge path must fold into the bounded ("badge","unknown","svg")
// counter series rather than minting new label values.
func TestServeBadge_CounterCardinalityBounded(t *testing.T) {
	srv := mockProm(t, "1", nil)
	h := newHandlerForTest(t, baseConfig(), srv.URL)

	before := counterValue(t, requestsTotal.WithLabelValues("badge", "unknown", "svg"))
	promtest.Get(t, h.Mux(), "/badges/does-not-exist")
	promtest.Get(t, h.Mux(), "/badges/also-missing")
	after := counterValue(t, requestsTotal.WithLabelValues("badge", "unknown", "svg"))

	assert.Equal(t, before+2, after)
}

func TestServeBadge_SVGDefault(t *testing.T) {
	srv := mockProm(t, "17.5", nil)
	h := newHandlerForTest(t, baseConfig(), srv.URL)

	w := promtest.Get(t, h.Mux(), "/badges/cpu")

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "image/svg+xml", w.Header().Get("Content-Type"))
	assert.True(t, strings.HasPrefix(w.Body.String(), "<svg"))
}

func TestServeBadge_Shields(t *testing.T) {
	srv := mockProm(t, "17.5", nil)
	h := newHandlerForTest(t, baseConfig(), srv.URL)

	w := promtest.Get(t, h.Mux(), "/badges/cpu?format=shields")

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))
	var body map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.EqualValues(t, 1, body["schemaVersion"])
	assert.Equal(t, "cpu", body["label"])
	assert.Equal(t, "17.5%", body["message"])
	assert.Equal(t, "green", body["color"])
}

func TestServeBadge_JSON(t *testing.T) {
	srv := mockProm(t, "17.5", nil)
	h := newHandlerForTest(t, baseConfig(), srv.URL)

	w := promtest.Get(t, h.Mux(), "/badges/cpu?format=json")

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
}

func TestServeBadge_Styles(t *testing.T) {
	srv := mockProm(t, "17.5", nil)
	h := newHandlerForTest(t, baseConfig(), srv.URL)

	for _, style := range []string{"", "flat-square", "plastic"} {
		w := promtest.Get(t, h.Mux(), "/badges/cpu?style="+style)
		assert.Equal(t, http.StatusOK, w.Code, "style=%q", style)
		assert.Equal(t, "image/svg+xml", w.Header().Get("Content-Type"), "style=%q", style)
		assert.True(t, strings.HasPrefix(w.Body.String(), "<svg"), "style=%q", style)
	}
}

func TestServeBadge_NotFound(t *testing.T) {
	srv := mockProm(t, "17.5", nil)
	h := newHandlerForTest(t, baseConfig(), srv.URL)

	w := promtest.Get(t, h.Mux(), "/badges/does-not-exist")

	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.Contains(t, w.Body.String(), `"isError":true`)
}

func TestRoutes_NonGETRejected(t *testing.T) {
	srv := mockProm(t, "17.5", nil)
	h := newHandlerForTest(t, baseConfig(), srv.URL)

	for _, method := range []string{http.MethodPost, http.MethodPut, http.MethodDelete} {
		req := httptest.NewRequest(method, "/badges/cpu", nil)
		w := httptest.NewRecorder()
		h.Mux().ServeHTTP(w, req)
		assert.Equal(t, http.StatusMethodNotAllowed, w.Code, "method=%s", method)
	}
}

func TestServeBadge_ValueExpression(t *testing.T) {
	cfg := config.KromgoConfig{Badges: []config.Badge{{
		ID: "uptime", Query: "q", Value: "humanizeDuration(result)",
	}}}
	srv := mockProm(t, "9000", nil)
	h := newHandlerForTest(t, cfg, srv.URL)

	w := promtest.Get(t, h.Mux(), "/badges/uptime?format=json")

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"value":"2h30m"`)
}

func TestServeBadge_ValueFromLabel(t *testing.T) {
	srv := promtest.Server(t, promtest.Scalar("0", map[string]string{"version": "v1.2.3"}), nil)
	cfg := config.KromgoConfig{Badges: []config.Badge{{ID: "ver", Query: "q", Value: `labels["version"]`}}}
	h := newHandlerForTest(t, cfg, srv.URL)

	w := promtest.Get(t, h.Mux(), "/badges/ver?format=shields")

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"message":"v1.2.3"`)
}

func TestServeBadge_ValueExpressionError(t *testing.T) {
	// Indexing a missing label is a CEL runtime error → 500.
	srv := promtest.Server(t, promtest.Scalar("0", map[string]string{"other": "x"}), nil)
	cfg := config.KromgoConfig{Badges: []config.Badge{{ID: "ver", Query: "q", Value: `labels["version"]`}}}
	h := newHandlerForTest(t, cfg, srv.URL)

	w := promtest.Get(t, h.Mux(), "/badges/ver")

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Contains(t, w.Body.String(), `"isError":true`)
}

func TestServeBadge_StateExpressions(t *testing.T) {
	cfg := config.KromgoConfig{Badges: []config.Badge{{
		ID:    "ceph",
		Query: "q",
		Value: `result == 0.0 ? "Healthy" : "Critical"`,
		Color: `result == 0.0 ? "green" : "red"`,
	}}}
	srv := mockProm(t, "0", nil)
	h := newHandlerForTest(t, cfg, srv.URL)

	w := promtest.Get(t, h.Mux(), "/badges/ceph?format=shields")

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"message":"Healthy"`)
	assert.Contains(t, w.Body.String(), `"color":"green"`)
}

func TestServeBadge_EmptyVector_NoData(t *testing.T) {
	srv := promtest.Server(t, nil, nil) // empty instant vector
	h := newHandlerForTest(t, baseConfig(), srv.URL)

	w := promtest.Get(t, h.Mux(), "/badges/cpu?format=shields")

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"message":"no data"`)
}

func TestServeBadge_RangeType(t *testing.T) {
	// type: range reduces a range query to one value (avg of 10,20,30 = 20).
	cfg := config.KromgoConfig{Badges: []config.Badge{{
		ID:    "cpu_avg",
		Type:  config.TypeRange,
		Query: "q",
		Value: `string(result) + "%"`,
		Range: &config.RangeQuery{Last: "1h", Reduce: config.ReduceAvg},
	}}}
	srv := promtest.Server(t, nil, []float64{10, 20, 30})
	h := newHandlerForTest(t, cfg, srv.URL)

	w := promtest.Get(t, h.Mux(), "/badges/cpu_avg?format=shields")

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"message":"20%"`)
}

func TestCacheControl_PerEndpoint(t *testing.T) {
	cfg := config.KromgoConfig{
		Defaults: config.Defaults{CacheSeconds: 60}, // global default
		Badges: []config.Badge{
			{ID: "fast", Query: "q"},
			{ID: "slow", Query: "q", CacheSeconds: new(3600)},
		},
	}
	srv := mockProm(t, "1", nil)
	h := newHandlerForTest(t, cfg, srv.URL)

	t.Run("global default", func(t *testing.T) {
		w := promtest.Get(t, h.Mux(), "/badges/fast?format=shields")
		assert.Equal(t, "public, max-age=60", w.Header().Get("Cache-Control"))
		assert.Contains(t, w.Body.String(), `"cacheSeconds":60`)
	})

	t.Run("per-endpoint override", func(t *testing.T) {
		w := promtest.Get(t, h.Mux(), "/badges/slow?format=shields")
		assert.Equal(t, "public, max-age=3600", w.Header().Get("Cache-Control"))
		assert.Contains(t, w.Body.String(), `"cacheSeconds":3600`)
	})
}

func TestCacheControl_DisabledByDefault(t *testing.T) {
	srv := mockProm(t, "17.5", nil)
	h := newHandlerForTest(t, baseConfig(), srv.URL) // no CacheSeconds

	w := promtest.Get(t, h.Mux(), "/badges/cpu")

	assert.Empty(t, w.Header().Get("Cache-Control"))
}

func TestCacheControl_ErrorsNotCached(t *testing.T) {
	cfg := config.KromgoConfig{
		Defaults: config.Defaults{CacheSeconds: 60},
		Graphs:   []config.Graph{{ID: "cpu", Query: "q", MaxDuration: "24h"}},
	}
	srv := mockProm(t, "1", nil)
	h := newHandlerForTest(t, cfg, srv.URL)

	// Window exceeds maxDuration → 400; the cache header must be no-store, not max-age.
	w := promtest.Get(t, h.Mux(), "/graphs/cpu?format=json&last=7d")

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Equal(t, "no-store", w.Header().Get("Cache-Control"))
}

func TestServeGraph_JSON(t *testing.T) {
	srv := mockProm(t, "0", []float64{1, 2, 3})
	h := newHandlerForTest(t, baseConfig(), srv.URL)

	w := promtest.Get(t, h.Mux(), "/graphs/cpu?format=json&last=1h")

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))
	var resp HistoryResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "cpu", resp.ID)
	require.Len(t, resp.Series, 1)
	assert.Len(t, resp.Series[0].Data, 3)
}

func TestServeGraph_SVGDefault(t *testing.T) {
	srv := mockProm(t, "0", []float64{10, 20, 15, 30})
	h := newHandlerForTest(t, baseConfig(), srv.URL)

	w := promtest.Get(t, h.Mux(), "/graphs/cpu?last=1h")

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "image/svg+xml", w.Header().Get("Content-Type"))
	assert.True(t, strings.HasPrefix(w.Body.String(), "<svg"))
}

func TestServeGraph_PNG(t *testing.T) {
	srv := mockProm(t, "0", []float64{10, 20, 15, 30})
	h := newHandlerForTest(t, baseConfig(), srv.URL)

	w := promtest.Get(t, h.Mux(), "/graphs/cpu?format=png&last=1h")

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "image/png", w.Header().Get("Content-Type"))
	assert.Equal(t, []byte{0x89, 'P', 'N', 'G'}, w.Body.Bytes()[:4])
}

func TestServeGraph_Theme(t *testing.T) {
	srv := mockProm(t, "0", []float64{10, 20, 15, 30})
	h := newHandlerForTest(t, baseConfig(), srv.URL)

	w := promtest.Get(t, h.Mux(), "/graphs/cpu?theme=dracula&last=1h")

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "rgb(40,42,54)") // dracula background
}

func TestServeGraph_WindowTooLarge(t *testing.T) {
	srv := mockProm(t, "0", []float64{1, 2})
	h := newHandlerForTest(t, baseConfig(), srv.URL) // graph MaxDuration 24h

	w := promtest.Get(t, h.Mux(), "/graphs/cpu?format=json&last=7d")

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestServeGraph_NotFound(t *testing.T) {
	srv := mockProm(t, "0", nil)
	h := newHandlerForTest(t, baseConfig(), srv.URL)

	w := promtest.Get(t, h.Mux(), "/graphs/nope")

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestIndexRoute(t *testing.T) {
	cfg := baseConfig()
	cfg.Defaults.Hidden = new(false)
	srv := mockProm(t, "0", nil)
	h := newHandlerForTest(t, cfg, srv.URL)

	w := promtest.Get(t, h.Mux(), "/")

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `<a href="/badges/cpu">cpu</a>`)
	assert.Contains(t, w.Body.String(), `<a href="/graphs/cpu">cpu</a>`)
}
