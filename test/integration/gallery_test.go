//go:build integration

package integration

import (
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/home-operations/kromgo/internal/config"
	"github.com/home-operations/kromgo/internal/kromgo"
	"github.com/home-operations/kromgo/internal/prometheus"
	"github.com/home-operations/kromgo/internal/promtest"
	"github.com/stretchr/testify/require"
)

func boolPtr(b bool) *bool { return &b }

// TestGallery renders a wide matrix of badge/graph variations through the real
// handler and writes a self-contained HTML page to inspect the styling. Run:
//
//	PROMETHEUS_URL=https://prom.example go test -tags integration -run TestGallery ./test/integration/...
//
// then open the logged file. Queries are synthetic (vector()/time()), so it works
// against any Prometheus and is deterministic.
func TestGallery(t *testing.T) {
	url := os.Getenv("PROMETHEUS_URL")
	if url == "" {
		t.Skip("PROMETHEUS_URL not set")
	}
	client, err := prometheus.New(url, 30*time.Second)
	require.NoError(t, err)

	const wave = "(time() % 1800) / 18" // 0..100 sawtooth — shows the line shape

	cfg := config.KromgoConfig{
		Defaults: config.Defaults{Hidden: boolPtr(false)},
		Badges: []config.Badge{
			{ID: "cpu", Title: "CPU", Query: "vector(4.6)", Value: `string(result) + "%"`, Color: threshold, Icon: "mdi:cpu-64-bit"},
			{ID: "cpu_hot", Title: "CPU", Query: "vector(82)", Value: `string(result) + "%"`, Color: threshold},
			{ID: "mem", Title: "Memory", Query: "vector(56012345678)", Value: "humanizeBytes(result)", Icon: "mdi:memory"},
			{ID: "pods", Title: "Pods", Query: "vector(1204)", Value: "humanizeNumber(result)", Icon: "mdi:kubernetes"},
			{ID: "uptime", Title: "Uptime", Query: "vector(40348800)", Value: "humanizeDuration(result)", Icon: "mdi:clock-outline"},
			{ID: "age", Title: "Age", Query: "vector(5961600)", Value: "humanizeDays(result)", Icon: "mdi:server-outline"},
			{ID: "ver", Title: "Kubernetes", Query: `label_replace(vector(1), "v", "1.36.1", "", "")`, Value: `labels["v"]`, Color: `"blue"`, Icon: "mdi:kubernetes"},
			{ID: "ok", Query: "vector(1)", Value: `"online"`, Color: `"green"`, Icon: "mdi:check-circle-outline"}, // icon, no title
		},
		Graphs: []config.Graph{
			{ID: "g", Title: "CPU", Query: wave},
			{ID: "g_bold", Title: "CPU", Query: wave, Font: "go-bold", Theme: "grafana"},
			{ID: "g_noto", Title: "CPU", Query: wave, Font: "notosans", Theme: "slate"},
		},
	}
	h, err := kromgo.New(cfg, client)
	require.NoError(t, err)

	var b strings.Builder
	b.WriteString(galleryHead)

	// Badges across formatters + icons.
	section(&b, "Badges — value formatters & icons")
	for _, id := range []string{"cpu", "cpu_hot", "mem", "pods", "uptime", "age", "ver", "ok"} {
		cell(t, h, &b, id, "/badges/"+id)
	}

	// Badge styles on one badge.
	section(&b, "Badge styles")
	for _, style := range []string{"flat", "flat-square", "plastic"} {
		cell(t, h, &b, style, "/badges/mem?style="+style)
	}

	// Graph themes.
	section(&b, "Graph themes")
	themes := []string{"light", "dark", "grafana", "ocean", "slate", "gray",
		"catppuccin-latte", "catppuccin-mocha", "dracula", "monokai", "night-owl"}
	for _, th := range themes {
		cell(t, h, &b, th, "/graphs/g?last=1h&width=360&height=120&theme="+th)
	}

	// Graph fonts (config-level) + a PNG.
	section(&b, "Graph fonts & PNG")
	cell(t, h, &b, "go-bold (grafana)", "/graphs/g_bold?last=1h&width=360&height=120")
	cell(t, h, &b, "notosans (slate)", "/graphs/g_noto?last=1h&width=360&height=120")
	cell(t, h, &b, "dark · PNG", "/graphs/g?last=1h&width=360&height=120&theme=dark&format=png")

	b.WriteString("</body></html>")

	out := filepath.Join(os.TempDir(), "kromgo-gallery.html")
	require.NoError(t, os.WriteFile(out, []byte(b.String()), 0o644))
	t.Logf("gallery written: open file://%s", out)
}

const threshold = `result <= 35 ? "green" : result <= 75 ? "orange" : "red"`

const galleryHead = `<!DOCTYPE html><html><head><meta charset="utf-8"><title>kromgo gallery</title>
<style>body{font:14px sans-serif;background:#fafafa;color:#222;margin:24px}
h2{margin-top:32px;border-bottom:1px solid #ddd;padding-bottom:4px}
.grid{display:flex;flex-wrap:wrap;gap:18px;align-items:flex-start}
.cell{background:#fff;border:1px solid #e3e3e3;border-radius:6px;padding:12px}
.cap{font-size:12px;color:#888;margin-top:8px;font-family:monospace}</style></head><body>
<h1>kromgo gallery</h1>`

func section(b *strings.Builder, title string) {
	fmt.Fprintf(b, `<h2>%s</h2><div class="grid">`, title)
}

func cell(t *testing.T, h *kromgo.Handler, b *strings.Builder, caption, path string) {
	t.Helper()
	w := promtest.Get(t, h.Mux(), path)
	if w.Code != 200 {
		t.Fatalf("%s -> %d: %s", path, w.Code, w.Body.String())
	}
	ct := w.Header().Get("Content-Type")
	var media string
	if strings.Contains(ct, "png") {
		media = fmt.Sprintf(`<img src="data:image/png;base64,%s">`, base64.StdEncoding.EncodeToString(w.Body.Bytes()))
	} else {
		media = w.Body.String() // inline SVG (our own trusted output)
	}
	fmt.Fprintf(b, `<div class="cell">%s<div class="cap">%s</div></div>`, media, caption)
}
