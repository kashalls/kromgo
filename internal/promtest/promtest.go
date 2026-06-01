// Package promtest provides a mock Prometheus HTTP API for use in tests across
// packages. It is imported only from test code, so it is never linked into the
// kromgo binary.
package promtest

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"
)

// Get issues a GET request against h and returns the recorded response.
func Get(t testing.TB, h http.Handler, target string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, target, nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	return w
}

// Sample is one instant-query result: a value and its labels.
type Sample struct {
	Value  string
	Labels map[string]string
}

// Scalar is a convenience for the common single-sample instant query.
func Scalar(value string, labels map[string]string) []Sample {
	return []Sample{{Value: value, Labels: labels}}
}

// Server returns an httptest.Server that answers /api/v1/query with vector and
// /api/v1/query_range with a single matrix stream (labels instance=a) built from
// matrix. It is closed automatically when the test finishes.
func Server(t testing.TB, vector []Sample, matrix []float64) *httptest.Server {
	t.Helper()
	now := time.Now().Unix()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/query":
			result := make([]any, len(vector))
			for i, s := range vector {
				result[i] = map[string]any{"metric": s.Labels, "value": []any{now, s.Value}}
			}
			writeResult(w, "vector", result)
		case "/api/v1/query_range":
			values := make([][]any, len(matrix))
			for i, v := range matrix {
				values[i] = []any{now + int64(i*60), formatFloat(v)}
			}
			stream := map[string]any{"metric": map[string]string{"instance": "a"}, "values": values}
			writeResult(w, "matrix", []any{stream})
		default:
			http.Error(w, "unexpected path "+r.URL.Path, http.StatusNotFound)
		}
	}))
	t.Cleanup(srv.Close)
	return srv
}

func writeResult(w http.ResponseWriter, resultType string, result any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"status": "success",
		"data":   map[string]any{"resultType": resultType, "result": result},
	})
}

func formatFloat(v float64) string {
	return strconv.FormatFloat(v, 'g', -1, 64)
}
