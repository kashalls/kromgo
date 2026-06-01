//go:build integration

package integration

import (
	"encoding/base64"
	"fmt"
	"html"
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
	"go.yaml.in/yaml/v4"
)

func boolPtr(b bool) *bool { return &b }

const (
	// wave is a 0..100 sawtooth so the chart line has a visible shape.
	wave      = "(time() % 1800) / 18"
	threshold = `result <= 35 ? "green" : result <= 75 ? "orange" : "red"`
	gdim      = "last=1h&width=480&height=200"
)

var graphFonts = []string{"roboto", "notosans", "notosans-bold", "go-regular", "go-bold", "go-medium", "go-mono"}

// TestGallery renders a wide matrix of badge/graph variations through the real
// handler and writes a self-contained, Tailwind-styled HTML page — each item shown
// with the kromgo config that produced it. Run:
//
//	PROMETHEUS_URL=https://prom.example mise run gallery   # then open kromgo-gallery.html
//
// Queries are synthetic (vector()/time()), so it works against any Prometheus.
func TestGallery(t *testing.T) {
	url := os.Getenv("PROMETHEUS_URL")
	if url == "" {
		t.Skip("PROMETHEUS_URL not set")
	}
	client, err := prometheus.New(url, 30*time.Second)
	require.NoError(t, err)

	badges := []config.Badge{
		{ID: "cpu", Title: "CPU", Query: "vector(4.6)", Value: `string(result) + "%"`, Color: threshold, Icon: "mdi:cpu-64-bit"},
		{ID: "cpu_hot", Title: "CPU", Query: "vector(82)", Value: `string(result) + "%"`, Color: threshold},
		{ID: "mem", Title: "Memory", Query: "vector(56012345678)", Value: "humanizeBytes(result)", Icon: "mdi:memory"},
		{ID: "pods", Title: "Pods", Query: "vector(1204)", Value: "humanizeNumber(result)", Icon: "mdi:kubernetes"},
		{ID: "uptime", Title: "Uptime", Query: "vector(40348800)", Value: "humanizeDuration(result)", Icon: "mdi:clock-outline"},
		{ID: "age", Title: "Age", Query: "vector(5961600)", Value: "humanizeDays(result)", Icon: "mdi:server-outline"},
		{ID: "ver", Title: "Kubernetes", Query: `label_replace(vector(1), "v", "1.36.1", "", "")`, Value: `labels["v"]`, Color: `"blue"`, Icon: "mdi:kubernetes"},
		{ID: "ok", Query: "vector(1)", Value: `"online"`, Color: `"green"`, Icon: "mdi:check-circle-outline"},
	}
	byID := map[string]config.Badge{}
	for _, b := range badges {
		byID[b.ID] = b
	}

	cfg := config.KromgoConfig{
		Defaults: config.Defaults{Hidden: boolPtr(false)},
		Badges:   badges,
		Graphs:   []config.Graph{{ID: "g", Title: "CPU", Query: wave}},
	}
	for _, f := range graphFonts {
		cfg.Graphs = append(cfg.Graphs, config.Graph{ID: "font_" + f, Title: "CPU", Query: wave, Font: f, Theme: "dark"})
	}
	h, err := kromgo.New(cfg, client)
	require.NoError(t, err)

	var b strings.Builder
	b.WriteString(galleryHead)

	group(&b, "Badges — value formatters & icons", func() {
		for _, id := range []string{"cpu", "cpu_hot", "mem", "pods", "uptime", "age", "ver", "ok"} {
			cell(t, h, &b, id, "/badges/"+id, true, yamlFor("badges", byID[id]))
		}
	})
	group(&b, "Badge styles", func() {
		for _, style := range []string{"flat", "flat-square", "plastic"} {
			bd := byID["mem"]
			bd.Style = style
			cell(t, h, &b, style, "/badges/mem?style="+style, true, yamlFor("badges", bd))
		}
	})
	group(&b, "Graph themes (PNG)", func() {
		for _, th := range []string{"light", "dark", "grafana", "ocean", "slate", "gray",
			"catppuccin-latte", "catppuccin-mocha", "dracula", "monokai", "night-owl"} {
			g := config.Graph{ID: "cpu", Title: "CPU", Query: wave, Theme: th}
			cell(t, h, &b, th, "/graphs/g?"+gdim+"&format=png&theme="+th, false, yamlFor("graphs", g))
		}
	})
	group(&b, "SVG vs PNG", func() {
		g := config.Graph{ID: "cpu", Title: "CPU", Query: wave, Theme: "grafana"}
		cell(t, h, &b, "grafana · SVG", "/graphs/g?"+gdim+"&theme=grafana", false, yamlFor("graphs", g))
		cell(t, h, &b, "grafana · PNG (?format=png)", "/graphs/g?"+gdim+"&format=png&theme=grafana", false, yamlFor("graphs", g))
	})
	group(&b, "Graph fonts (PNG, dark)", func() {
		for _, f := range graphFonts {
			g := config.Graph{ID: "cpu", Title: "CPU", Query: wave, Font: f, Theme: "dark"}
			cell(t, h, &b, f, "/graphs/font_"+f+"?"+gdim+"&format=png", false, yamlFor("graphs", g))
		}
	})

	b.WriteString("</main></body></html>")

	out := os.Getenv("GALLERY_OUT")
	if out == "" {
		out = filepath.Join(moduleRoot(t), "kromgo-gallery.html")
	}
	require.NoError(t, os.WriteFile(out, []byte(b.String()), 0o644))
	t.Logf("gallery written: open %q", out)
}

// yamlFor marshals a single endpoint under its top-level key (badges:/graphs:).
func yamlFor(key string, v any) string {
	out, err := yaml.Marshal(map[string]any{key: []any{v}})
	if err != nil {
		return err.Error()
	}
	return strings.TrimRight(string(out), "\n")
}

const galleryHead = `<!DOCTYPE html><html lang="en"><head><meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>kromgo gallery</title>
<script src="https://unpkg.com/@tailwindcss/browser@4"></script>
<style>.badge svg{height:42px;width:auto}</style>
</head>
<body class="bg-slate-100 text-slate-800 antialiased">
<main class="max-w-7xl mx-auto px-6 py-10">
<header class="mb-10">
  <h1 class="text-3xl font-bold tracking-tight text-slate-900">kromgo gallery</h1>
  <p class="mt-1 text-slate-500">Badge &amp; graph variations, each with the config that produced it — synthetic data.</p>
</header>`

// group renders a titled section whose body emits cells, with valid nesting.
func group(b *strings.Builder, title string, body func()) {
	fmt.Fprintf(b, `<section class="mb-12">
<h2 class="text-sm font-semibold uppercase tracking-wider text-slate-500 mb-4">%s</h2>
<div class="flex flex-wrap gap-5 items-start">`, title)
	body()
	b.WriteString(`</div></section>`)
}

func cell(t *testing.T, h *kromgo.Handler, b *strings.Builder, caption, path string, badge bool, cfgYAML string) {
	t.Helper()
	w := promtest.Get(t, h.Mux(), path)
	if w.Code != 200 {
		t.Fatalf("%s -> %d: %s", path, w.Code, w.Body.String())
	}
	var media string
	if strings.Contains(w.Header().Get("Content-Type"), "png") {
		media = fmt.Sprintf(`<img class="block rounded" src="data:image/png;base64,%s">`, base64.StdEncoding.EncodeToString(w.Body.Bytes()))
	} else {
		media = w.Body.String() // inline SVG (our own trusted output)
	}
	cls := "rounded-xl border border-slate-200 bg-white shadow-sm p-4 flex flex-col gap-3 max-w-lg"
	if badge {
		cls += " badge"
	}
	markdown := fmt.Sprintf("![%s](%s%s)", idFromPath(path), exampleHost, path)

	fmt.Fprintf(b, `<figure class="%s">
<div class="flex items-center gap-3"><div>%s</div><figcaption class="text-xs font-mono text-slate-400">%s</figcaption></div>
<div><div class="text-[10px] font-semibold uppercase tracking-wider text-slate-400 mb-1">Markdown</div>
<pre class="text-[11px] leading-snug bg-slate-50 border border-slate-200 text-slate-700 rounded-lg p-3 overflow-x-auto"><code>%s</code></pre></div>
<div><div class="text-[10px] font-semibold uppercase tracking-wider text-slate-400 mb-1">Config</div>
<pre class="text-[11px] leading-snug bg-slate-900 text-slate-100 rounded-lg p-3 overflow-x-auto"><code>%s</code></pre></div>
</figure>`, cls, media, caption, html.EscapeString(markdown), html.EscapeString(cfgYAML))
}

// exampleHost is a placeholder origin for the copy-paste Markdown snippets.
const exampleHost = "https://kromgo.example.com"

// idFromPath extracts the endpoint id from a request path for Markdown alt text.
func idFromPath(path string) string {
	p := strings.TrimPrefix(strings.TrimPrefix(path, "/badges/"), "/graphs/")
	if i := strings.IndexByte(p, '?'); i >= 0 {
		p = p[:i]
	}
	return p
}

// moduleRoot walks up from the test's working directory to the directory holding go.mod.
func moduleRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	require.NoError(t, err)
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		require.NotEqual(t, parent, dir, "could not locate go.mod")
		dir = parent
	}
}
