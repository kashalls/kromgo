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

	t.Run("badge svg (default)", func(t *testing.T) {
		resp := h.get("/badges/cpu")
		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.Equal(t, "image/svg+xml", resp.Header.Get("Content-Type"))
		assert.True(t, strings.HasPrefix(bodyString(t, resp), "<svg"))
	})

	t.Run("badge shields", func(t *testing.T) {
		resp := h.get("/badges/cpu?format=shields")
		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.Equal(t, "application/json", resp.Header.Get("Content-Type"))
		var body map[string]any
		require.NoError(t, json.Unmarshal([]byte(bodyString(t, resp)), &body))
		assert.Equal(t, "cpu", body["label"])
		assert.Equal(t, "17.5%", body["message"])
		assert.Equal(t, "green", body["color"])
	})

	t.Run("badge json", func(t *testing.T) {
		resp := h.get("/badges/cpu?format=json")
		assert.Equal(t, http.StatusOK, resp.StatusCode)
		body := bodyString(t, resp)
		assert.Contains(t, body, `"value":"17.5%"`)
		assert.Contains(t, body, `"job":"node"`)
	})

	t.Run("graph svg (default)", func(t *testing.T) {
		resp := h.get("/graphs/cpu?last=1h")
		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.Equal(t, "image/svg+xml", resp.Header.Get("Content-Type"))
		assert.True(t, strings.HasPrefix(bodyString(t, resp), "<svg"))
	})

	t.Run("graph json", func(t *testing.T) {
		resp := h.get("/graphs/cpu?format=json&last=1h")
		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.Contains(t, bodyString(t, resp), `"id":"cpu"`)
	})

	t.Run("index", func(t *testing.T) {
		resp := h.get("/")
		assert.Equal(t, http.StatusOK, resp.StatusCode)
		body := bodyString(t, resp)
		assert.Contains(t, body, `<a href="/badges/cpu">cpu</a>`)
		assert.Contains(t, body, `<a href="/graphs/cpu">cpu</a>`)
	})

	t.Run("security headers", func(t *testing.T) {
		resp := h.get("/badges/cpu")
		_ = bodyString(t, resp)
		assert.Equal(t, "nosniff", resp.Header.Get("X-Content-Type-Options"))
		assert.Contains(t, resp.Header.Get("Content-Security-Policy"), "default-src 'none'")
	})

	t.Run("not found", func(t *testing.T) {
		resp := h.get("/badges/nope")
		assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	})
}
