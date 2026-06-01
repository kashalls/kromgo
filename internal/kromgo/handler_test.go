package kromgo

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/home-operations/kromgo/internal/config"
	"github.com/home-operations/kromgo/internal/prometheus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockProm returns an httptest server that speaks just enough of the Prometheus
// HTTP API for the handler: an instant vector for /query and a matrix for /query_range.
func mockProm(t *testing.T, vectorValue string, matrixValues []float64) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		now := time.Now().Unix()
		switch r.URL.Path {
		case "/api/v1/query":
			_, _ = fmt.Fprintf(w, `{"status":"success","data":{"resultType":"vector","result":[{"metric":{"job":"node"},"value":[%d,%q]}]}}`, now, vectorValue)
		case "/api/v1/query_range":
			points := make([]string, len(matrixValues))
			for i, v := range matrixValues {
				points[i] = fmt.Sprintf("[%d,%q]", now+int64(i*60), fmt.Sprintf("%g", v))
			}
			_, _ = fmt.Fprintf(w, `{"status":"success","data":{"resultType":"matrix","result":[{"metric":{"instance":"a"},"values":[%s]}]}}`, strings.Join(points, ","))
		default:
			http.Error(w, "unexpected path "+r.URL.Path, http.StatusNotFound)
		}
	}))
	t.Cleanup(srv.Close)
	return srv
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
				Name:   "cpu",
				Query:  "node_cpu_usage",
				Suffix: "%",
				Colors: []config.MetricColor{{Min: 0, Max: 50, Color: "green"}},
			},
		},
		History: config.HistoryConfig{Enabled: true, MaxDuration: "24h"},
	}
}

func doGet(t *testing.T, h *Handler, target string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, target, nil)
	w := httptest.NewRecorder()
	h.Mux().ServeHTTP(w, req)
	return w
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

func TestServeMetric_NotFound(t *testing.T) {
	srv := mockProm(t, "17.5", nil)
	h := newHandlerForTest(t, baseConfig(), srv.URL)

	w := doGet(t, h, "/does-not-exist")

	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.Contains(t, w.Body.String(), `"isError":true`)
}

func TestServeMetric_ValueTemplateAndOverride(t *testing.T) {
	cfg := config.KromgoConfig{
		Metrics: []config.Metric{{
			Name:          "uptime",
			Query:         "q",
			ValueTemplate: "{{ . | humanDuration }}",
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
	cfg.History.Enabled = false
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

func TestIndexRoute(t *testing.T) {
	cfg := baseConfig()
	cfg.HideAll = boolPtr(false)
	srv := mockProm(t, "0", nil)
	h := newHandlerForTest(t, cfg, srv.URL)

	w := doGet(t, h, "/")

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `<a href="/cpu">cpu</a>`)
}
