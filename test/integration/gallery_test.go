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
		desc := map[string]string{
			"cpu":     "Percentage with threshold color",
			"cpu_hot": "Threshold color — red when high",
			"mem":     "humanizeBytes + icon",
			"pods":    "humanizeNumber + icon",
			"uptime":  "humanizeDuration + icon",
			"age":     "humanizeDays + icon",
			"ver":     "Value from a label + icon",
			"ok":      "Icon only (no title)",
		}
		for _, id := range []string{"cpu", "cpu_hot", "mem", "pods", "uptime", "age", "ver", "ok"} {
			cell(t, h, &b, desc[id], "/badges/"+id, true, yamlFor("badges", byID[id]))
		}
	})
	group(&b, "Badge styles", func() {
		// A different badge per style so they're easy to tell apart.
		for _, sb := range []struct{ style, id string }{
			{"flat", "cpu"}, {"flat-square", "pods"}, {"plastic", "mem"},
		} {
			bd := byID[sb.id]
			bd.Style = sb.style
			cell(t, h, &b, "Style: "+sb.style, "/badges/"+sb.id+"?style="+sb.style, true, yamlFor("badges", bd))
		}
	})
	group(&b, "Graph themes (PNG)", func() {
		for _, th := range []string{"light", "dark", "grafana", "ocean", "slate", "gray",
			"catppuccin-latte", "catppuccin-mocha", "dracula", "monokai", "night-owl"} {
			g := config.Graph{ID: "cpu", Title: "CPU", Query: wave, Theme: th}
			cell(t, h, &b, "Theme: "+th, "/graphs/g?"+gdim+"&format=png&theme="+th, false, yamlFor("graphs", g))
		}
	})
	group(&b, "SVG vs PNG", func() {
		g := config.Graph{ID: "cpu", Title: "CPU", Query: wave, Theme: "grafana"}
		cell(t, h, &b, "SVG output (default)", "/graphs/g?"+gdim+"&theme=grafana", false, yamlFor("graphs", g))
		cell(t, h, &b, "PNG output (?format=png)", "/graphs/g?"+gdim+"&format=png&theme=grafana", false, yamlFor("graphs", g))
	})
	group(&b, "Graph fonts (PNG, dark)", func() {
		for _, f := range graphFonts {
			g := config.Graph{ID: "cpu", Title: "CPU", Query: wave, Font: f, Theme: "dark"}
			cell(t, h, &b, "Font: "+f, "/graphs/font_"+f+"?"+gdim+"&format=png", false, yamlFor("graphs", g))
		}
	})

	b.WriteString(`</main><script>
function copyCode(btn){
  var text = btn.parentElement.querySelector('code').textContent;
  var done = function(){ var o = btn.textContent; btn.textContent = 'Copied!'; setTimeout(function(){ btn.textContent = o; }, 1200); };
  if (navigator.clipboard && navigator.clipboard.writeText) { navigator.clipboard.writeText(text).then(done).catch(fallback); } else { fallback(); }
  function fallback(){ var ta = document.createElement('textarea'); ta.value = text; document.body.appendChild(ta); ta.select(); try { document.execCommand('copy'); done(); } finally { document.body.removeChild(ta); } }
}
hljs.highlightAll();
</script></body></html>`)

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
<link rel="stylesheet" href="https://cdnjs.cloudflare.com/ajax/libs/highlight.js/11.10.0/styles/github-dark.min.css">
<script src="https://cdnjs.cloudflare.com/ajax/libs/highlight.js/11.10.0/highlight.min.js"></script>
<style>.badge svg{height:42px;width:auto}
.cell img{max-width:100%;height:auto}
pre code.hljs{padding:0;background:transparent}</style>
</head>
<body class="bg-slate-950 text-slate-200 antialiased">
<main class="max-w-7xl mx-auto px-6 py-10">
<header class="mb-10">
  <h1 class="text-3xl font-bold tracking-tight text-white">kromgo gallery</h1>
  <p class="mt-1 text-slate-400">Badge &amp; graph variations, each with the config that produced it — synthetic data.</p>
</header>`

// group renders a titled section whose body emits cells in a uniform grid.
func group(b *strings.Builder, title string, body func()) {
	fmt.Fprintf(b, `<section class="mb-12">
<h2 class="text-sm font-semibold uppercase tracking-wider text-slate-400 mb-4">%s</h2>
<div class="grid gap-5 grid-cols-1 sm:grid-cols-2 lg:grid-cols-3">`, title)
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
	cls := "cell rounded-xl border border-slate-800 bg-slate-900 shadow-sm p-4 flex flex-col gap-3"
	if badge {
		cls += " badge"
	}
	markdown := fmt.Sprintf("![%s](%s%s)", idFromPath(path), exampleHost, path)

	fmt.Fprintf(b, `<figure class="%s">
<figcaption class="text-sm font-semibold text-slate-200">%s</figcaption>
<div class="flex items-center justify-center py-2">%s</div>`, cls, html.EscapeString(caption), media)
	codeBlock(b, "Markdown", "", markdown, "")
	codeBlock(b, "Config", "yaml", cfgYAML, "mt-auto")
	b.WriteString(`</figure>`)
}

// codeBlock writes a labelled, copy-able code block. extraClass is applied to the
// wrapper (e.g. "mt-auto" to pin the config block to the card's bottom).
func codeBlock(b *strings.Builder, label, lang, content, extraClass string) {
	langAttr := ""
	if lang != "" {
		langAttr = ` class="language-` + lang + `"`
	}
	fmt.Fprintf(b, `<div class="%s"><div class="text-[10px] font-semibold uppercase tracking-wider text-slate-500 mb-1">%s</div>
<div class="relative group">
<pre class="text-[11px] leading-snug bg-slate-800/60 border border-slate-700 rounded-lg p-3 overflow-x-auto"><code%s>%s</code></pre>
<button type="button" onclick="copyCode(this)" class="absolute top-2 right-2 rounded-md bg-slate-700/80 hover:bg-slate-600 text-slate-200 text-[10px] font-medium px-2 py-0.5 opacity-0 group-hover:opacity-100 transition">Copy</button>
</div></div>`, extraClass, label, langAttr, html.EscapeString(content))
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
