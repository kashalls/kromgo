package prometheus

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/home-operations/kromgo/internal/promtest"
	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNew_EmptyAddress(t *testing.T) {
	t.Parallel()
	_, err := New("", 0)
	assert.Error(t, err)
}

func TestNew_Valid(t *testing.T) {
	t.Parallel()
	c, err := New("http://localhost:9090", 0)
	require.NoError(t, err)
	assert.NotNil(t, c)
}

func TestQuery_Vector(t *testing.T) {
	t.Parallel()
	srv := promtest.Server(t, promtest.Scalar("17.5", map[string]string{"job": "node"}), nil)
	c, err := New(srv.URL, 0)
	require.NoError(t, err)

	value, err := c.Query(context.Background(), "up", time.Now())
	require.NoError(t, err)

	vec, ok := value.(model.Vector)
	require.True(t, ok)
	require.Len(t, vec, 1)
	assert.Equal(t, 17.5, float64(vec[0].Value))
}

func TestQueryRange_Matrix(t *testing.T) {
	t.Parallel()
	srv := promtest.Server(t, nil, []float64{1, 2, 3})
	c, err := New(srv.URL, 0)
	require.NoError(t, err)

	rng := v1.Range{Start: time.Now().Add(-time.Hour), End: time.Now(), Step: time.Minute}
	value, err := c.QueryRange(context.Background(), "up", rng)
	require.NoError(t, err)

	m, ok := value.(model.Matrix)
	require.True(t, ok)
	require.Len(t, m, 1)
	assert.Len(t, m[0].Values, 3)
}

func TestQuery_ServerError(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	}))
	t.Cleanup(srv.Close)

	c, err := New(srv.URL, 0)
	require.NoError(t, err)

	_, err = c.Query(context.Background(), "up", time.Now())
	assert.Error(t, err)
}

func TestQuery_TimeoutApplied(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		time.Sleep(200 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(srv.Close)

	c, err := New(srv.URL, 10*time.Millisecond)
	require.NoError(t, err)

	_, err = c.Query(context.Background(), "up", time.Now())
	assert.Error(t, err, "expected the per-query timeout to fire")
}
