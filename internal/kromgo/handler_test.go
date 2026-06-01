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

func baseConfig() config.KromgoConfig {
	return config.KromgoConfig{
		Metrics: []config.Metric{
			{
				Name:  "cpu",
				Query: "node_cpu_usage",
				Value: `string(result) + "%"`,
				Color: `result <= 50.0 ? "green" : "red"`,
			},
		},
		Defaults: config.Defaults{Timeseries: config.TimeseriesConfig{Enabled: true, MaxDuration: "24h"}},
	}
}

func doGet(t *testing.T, h *Handler, target string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, target, nil)
	w := httptest.NewRecorder()
	h.Mux().ServeHTTP(w, req)
	return w
}

func counterValue(t *testing.T, c promclient.Counter) float64 {
	t.Helper()
	var m dto.Metric
	require.NoError(t, c.Write(&m))
	return m.GetCounter().GetValue()
}

// An unknown metric path and an unknown ?format= value must both fold into the
// bounded ("unknown","json") counter series rather than minting new label values.
func TestServeMetric_CounterCardinalityBounded(t *testing.T) {
	srv := mockProm(t, "1", nil)
	h := newHandlerForTest(t, baseConfig(), srv.URL)

	before := counterValue(t, requestsTotal.WithLabelValues("unknown", "json"))
	doGet(t, h, "/does-not-exist")
	doGet(t, h, "/also-missing?format=bogus")
	after := counterValue(t, requestsTotal.WithLabelValues("unknown", "json"))

	assert.Equal(t, before+2, after)
}

func TestServeMetric_JSON(t *testing.T) {
	srv := mockProm(t, "17.5", nil)
	h := newHandlerForTest(t, baseConfig(), srv.URL)

	w := doGet(t, h, "/cpu")

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))
	var body map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.EqualValues(t, 1, body["schemaVersion"])
	assert.Equal(t, "cpu", body["label"])
	assert.Equal(t, "17.5%", body["message"])
	assert.Equal(t, "green", body["color"])
}

func TestServeMetric_QueryParamForm(t *testing.T) {
	srv := mockProm(t, "17.5", nil)
	h := newHandlerForTest(t, baseConfig(), srv.URL)

	w := doGet(t, h, "/query?metric=cpu")

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"label":"cpu"`)
}

func TestServeMetric_Raw(t *testing.T) {
	srv := mockProm(t, "17.5", nil)
	h := newHandlerForTest(t, baseConfig(), srv.URL)

	w := doGet(t, h, "/cpu?format=raw")

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))
	// Raw is the Prometheus vector marshalled directly.
	assert.Contains(t, w.Body.String(), `"job":"node"`)
}

func TestServeMetric_Badge(t *testing.T) {
	srv := mockProm(t, "17.5", nil)
	h := newHandlerForTest(t, baseConfig(), srv.URL)

	for _, style := range []string{"", "flat-square", "plastic"} {
		w := doGet(t, h, "/cpu?format=badge&style="+style)
		assert.Equal(t, http.StatusOK, w.Code, "style=%q", style)
		assert.Equal(t, "image/svg+xml", w.Header().Get("Content-Type"), "style=%q", style)
		assert.True(t, strings.HasPrefix(w.Body.String(), "<svg"), "style=%q", style)
	}
}

func TestServeMetric_NonGETRejected(t *testing.T) {
	// Only safe (idempotent) GET requests are routed; the ServeMux returns 405 otherwise.
	srv := mockProm(t, "17.5", nil)
	h := newHandlerForTest(t, baseConfig(), srv.URL)

	for _, method := range []string{http.MethodPost, http.MethodPut, http.MethodDelete} {
		req := httptest.NewRequest(method, "/cpu", nil)
		w := httptest.NewRecorder()
		h.Mux().ServeHTTP(w, req)
		assert.Equal(t, http.StatusMethodNotAllowed, w.Code, "method=%s", method)
	}
}

func TestServeMetric_NotFound(t *testing.T) {
	srv := mockProm(t, "17.5", nil)
	h := newHandlerForTest(t, baseConfig(), srv.URL)

	w := doGet(t, h, "/does-not-exist")

	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.Contains(t, w.Body.String(), `"isError":true`)
}

func TestServeMetric_ValueExpression(t *testing.T) {
	cfg := config.KromgoConfig{
		Metrics: []config.Metric{{
			Name:  "uptime",
			Query: "q",
			Value: "humanizeDuration(result)",
		}},
	}
	srv := mockProm(t, "9000", nil)
	h := newHandlerForTest(t, cfg, srv.URL)

	w := doGet(t, h, "/uptime")

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"message":"2h30m"`)
}

func TestServeMetric_History(t *testing.T) {
	srv := mockProm(t, "0", []float64{1, 2, 3})
	h := newHandlerForTest(t, baseConfig(), srv.URL)

	w := doGet(t, h, "/cpu?format=history&last=1h")

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))
	var resp HistoryResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "cpu", resp.Metric)
	require.Len(t, resp.Series, 1)
	assert.Len(t, resp.Series[0].Data, 3)
}

func TestServeMetric_HistoryDisabled(t *testing.T) {
	cfg := baseConfig()
	cfg.Defaults.Timeseries.Enabled = false
	srv := mockProm(t, "0", []float64{1, 2, 3})
	h := newHandlerForTest(t, cfg, srv.URL)

	w := doGet(t, h, "/cpu?format=history")

	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestServeMetric_Chart(t *testing.T) {
	srv := mockProm(t, "0", []float64{10, 20, 15, 30})
	h := newHandlerForTest(t, baseConfig(), srv.URL)

	w := doGet(t, h, "/cpu?format=chart&last=1h")

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "image/svg+xml", w.Header().Get("Content-Type"))
	assert.True(t, strings.HasPrefix(w.Body.String(), "<svg"))
}

func TestServeMetric_HistoryWindowTooLarge(t *testing.T) {
	srv := mockProm(t, "0", []float64{1, 2})
	h := newHandlerForTest(t, baseConfig(), srv.URL) // MaxDuration 24h

	w := doGet(t, h, "/cpu?format=history&last=7d")

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestServeMetric_ValueFromLabel(t *testing.T) {
	srv := promtest.Server(t, promtest.Scalar("0", map[string]string{"version": "v1.2.3"}), nil)
	cfg := config.KromgoConfig{Metrics: []config.Metric{{Name: "ver", Query: "q", Value: `labels["version"]`}}}
	h := newHandlerForTest(t, cfg, srv.URL)

	w := doGet(t, h, "/ver")

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"message":"v1.2.3"`)
}

func TestServeMetric_ValueExpressionError(t *testing.T) {
	// Indexing a missing label is a CEL runtime error → 500 (write a safe expr instead).
	srv := promtest.Server(t, promtest.Scalar("0", map[string]string{"other": "x"}), nil)
	cfg := config.KromgoConfig{Metrics: []config.Metric{{Name: "ver", Query: "q", Value: `labels["version"]`}}}
	h := newHandlerForTest(t, cfg, srv.URL)

	w := doGet(t, h, "/ver")

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Contains(t, w.Body.String(), `"isError":true`)
}

func TestServeMetric_StateExpressions(t *testing.T) {
	cfg := config.KromgoConfig{Metrics: []config.Metric{{
		Name:  "ceph",
		Query: "q",
		Value: `result == 0.0 ? "Healthy" : "Critical"`,
		Color: `result == 0.0 ? "green" : "red"`,
	}}}
	srv := mockProm(t, "0", nil)
	h := newHandlerForTest(t, cfg, srv.URL)

	w := doGet(t, h, "/ceph")

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"message":"Healthy"`)
	assert.Contains(t, w.Body.String(), `"color":"green"`)
}

func TestServeMetric_EmptyVector_NoData(t *testing.T) {
	srv := promtest.Server(t, nil, nil) // empty instant vector
	h := newHandlerForTest(t, baseConfig(), srv.URL)

	w := doGet(t, h, "/cpu")

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "metric returned no data")
}

func TestCacheControl_PerMetric(t *testing.T) {
	cfg := config.KromgoConfig{
		Defaults: config.Defaults{CacheSeconds: 60}, // global default
		Metrics: []config.Metric{
			{Name: "fast", Query: "q"},
			{Name: "slow", Query: "q", CacheSeconds: new(3600)}, // overrides default
		},
	}
	srv := mockProm(t, "1", nil)
	h := newHandlerForTest(t, cfg, srv.URL)

	t.Run("global default", func(t *testing.T) {
		w := doGet(t, h, "/fast")
		assert.Equal(t, "public, max-age=60", w.Header().Get("Cache-Control"))
		assert.Contains(t, w.Body.String(), `"cacheSeconds":60`)
	})

	t.Run("per-metric override", func(t *testing.T) {
		w := doGet(t, h, "/slow")
		assert.Equal(t, "public, max-age=3600", w.Header().Get("Cache-Control"))
		assert.Contains(t, w.Body.String(), `"cacheSeconds":3600`)
	})
}

func TestCacheControl_DisabledByDefault(t *testing.T) {
	srv := mockProm(t, "17.5", nil)
	h := newHandlerForTest(t, baseConfig(), srv.URL) // no CacheSeconds

	w := doGet(t, h, "/cpu")

	assert.Empty(t, w.Header().Get("Cache-Control"))
	assert.NotContains(t, w.Body.String(), "cacheSeconds")
}

func TestCacheControl_ErrorsNotCached(t *testing.T) {
	cfg := config.KromgoConfig{
		Defaults: config.Defaults{CacheSeconds: 60},
		Metrics:  []config.Metric{{Name: "cpu", Query: "q"}},
	}
	srv := mockProm(t, "1", nil)
	h := newHandlerForTest(t, cfg, srv.URL)

	// Range queries are disabled → 403 error; the cache header must be no-store, not max-age.
	w := doGet(t, h, "/cpu?format=history")

	assert.Equal(t, http.StatusForbidden, w.Code)
	assert.Equal(t, "no-store", w.Header().Get("Cache-Control"))
}

func TestServeMetric_RangeType(t *testing.T) {
	// type: range reduces a range query to one value (avg of 10,20,30 = 20).
	cfg := config.KromgoConfig{
		Metrics: []config.Metric{{
			Name:  "cpu_avg",
			Type:  config.TypeRange,
			Query: "q",
			Value: `string(result) + "%"`,
			Range: &config.RangeQuery{Last: "1h", Reduce: config.ReduceAvg},
		}},
	}
	srv := promtest.Server(t, nil, []float64{10, 20, 30})
	h := newHandlerForTest(t, cfg, srv.URL)

	w := doGet(t, h, "/cpu_avg")

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"message":"20%"`)
}

func TestIndexRoute(t *testing.T) {
	cfg := baseConfig()
	cfg.Defaults.Hidden = new(false)
	srv := mockProm(t, "0", nil)
	h := newHandlerForTest(t, cfg, srv.URL)

	w := doGet(t, h, "/")

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `<a href="/cpu">cpu</a>`)
}
