//go:build e2e

package e2e

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func bodyString(t *testing.T, resp *http.Response) string {
	t.Helper()
	defer func() { _ = resp.Body.Close() }()
	b, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	return string(b)
}

func TestE2E(t *testing.T) {
	h := start(t)

	t.Run("json", func(t *testing.T) {
		resp := h.get("/cpu")
		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.Equal(t, "application/json", resp.Header.Get("Content-Type"))
		var body map[string]any
		require.NoError(t, json.Unmarshal([]byte(bodyString(t, resp)), &body))
		assert.Equal(t, "cpu", body["label"])
		assert.Equal(t, "17.5%", body["message"])
		assert.Equal(t, "green", body["color"])
	})

	t.Run("raw", func(t *testing.T) {
		resp := h.get("/cpu?format=raw")
		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.Contains(t, bodyString(t, resp), `"job":"node"`)
	})

	t.Run("badge", func(t *testing.T) {
		resp := h.get("/cpu?format=badge")
		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.Equal(t, "image/svg+xml", resp.Header.Get("Content-Type"))
		assert.True(t, strings.HasPrefix(bodyString(t, resp), "<svg"))
	})

	t.Run("history", func(t *testing.T) {
		resp := h.get("/cpu?format=history&last=1h")
		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.Contains(t, bodyString(t, resp), `"metric":"cpu"`)
	})

	t.Run("chart", func(t *testing.T) {
		resp := h.get("/cpu?format=chart&last=1h")
		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.Equal(t, "image/svg+xml", resp.Header.Get("Content-Type"))
	})

	t.Run("index", func(t *testing.T) {
		resp := h.get("/")
		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.Contains(t, bodyString(t, resp), `<a href="/cpu">cpu</a>`)
	})

	t.Run("not found", func(t *testing.T) {
		resp := h.get("/nope")
		assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	})
}
