package kromgo

import (
	"embed"
	"html/template"
	"io/fs"
	"net/http"
	"regexp"
	"strings"

	"github.com/home-operations/kromgo/internal/config"
)

// galleryAssets holds the vendored (marked, github-markdown-css) and first-party
// (gallery.css, gallery.js) files served under /assets/. They are embedded so the
// service stays self-contained and the index page keeps a strict script-src 'self'
// CSP — no external CDN. They are vendored via npm and built by cmd/genassets.
//
//go:embed assets/marked.js assets/github-markdown.css assets/gallery.css assets/gallery.js
var galleryAssets embed.FS

// assetsFS is galleryAssets rooted at the assets/ directory, for serving under /assets/.
var assetsFS = func() fs.FS {
	sub, err := fs.Sub(galleryAssets, "assets")
	if err != nil {
		panic(err) // embedded paths are fixed; this cannot fail at runtime
	}
	return sub
}()

// indexCSP relaxes the default strict policy just enough for the gallery page:
// scripts, styles, and images load only from this origin (the embedded assets and
// the badge/graph endpoints), and gallery.js is an external file (no inline
// scripts), so no 'unsafe-inline'/'unsafe-eval' is needed. frame-ancestors blocks
// clickjacking; base-uri/form-action are locked down for completeness.
const indexCSP = "default-src 'none'; script-src 'self'; style-src 'self'; img-src 'self'; base-uri 'none'; form-action 'none'; frame-ancestors 'none'"

var galleryTmpl = template.Must(template.New("gallery").Parse(`<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>kromgo</title>
<link rel="stylesheet" href="/assets/github-markdown.css">
<link rel="stylesheet" href="/assets/gallery.css">
</head>
<body>
<div class="wrap">
<header class="masthead">
<h1>kromgo</h1>
<p>Prometheus metrics as badges &amp; graphs &mdash; copy a snippet into your Markdown.</p>
</header>
{{- if or .Badges .Graphs}}
{{- if .Badges}}
<section>
<h2>Badges</h2>
<div class="grid">{{range .Badges}}{{template "card" .}}{{end}}</div>
</section>
{{- end}}
{{- if .Graphs}}
<section>
<h2>Graphs</h2>
<div class="grid">{{range .Graphs}}{{template "card" .}}{{end}}</div>
</section>
{{- end}}
{{- else}}
<p class="empty">No endpoints to show. Define entries under <code>badges:</code> / <code>graphs:</code> &mdash; each appears here unless its <code>gallery.hidden</code> is true.</p>
{{- end}}
</div>
<script src="/assets/marked.js"></script>
<script src="/assets/gallery.js"></script>
</body>
</html>
{{- define "card"}}<div class="card"><div class="preview"><div class="markdown-body"></div></div><div class="snippet"><button class="copy" type="button" aria-label="Copy Markdown">Copy</button><pre><code>{{.Markdown}}</code></pre></div></div>{{end}}`))

var landingTmpl = template.Must(template.New("landing").Parse(`<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>kromgo</title>
<link rel="stylesheet" href="/assets/gallery.css">
</head>
<body>
<div class="landing">
<div>
<h1>kromgo</h1>
<p>Running.</p>
</div>
</div>
</body>
</html>`))

// galleryView is the gallery template's data: visible badges and graphs as
// copy-pasteable Markdown snippets, badges first.
type galleryView struct {
	Badges []galleryItem
	Graphs []galleryItem
}

// galleryItem is one endpoint rendered as a Markdown image snippet.
type galleryItem struct {
	Markdown string
}

// index renders the gallery of visible badges and graphs (the default), or a
// minimal landing page when gallery.enabled is false.
func (h *Handler) index(w http.ResponseWriter, r *http.Request) {
	header := w.Header()
	header.Set("Content-Security-Policy", indexCSP) // override the strict default
	header.Set("Content-Type", mimeHTML)
	// Snippet URLs embed the request Host, so the page is per-Host and must never be
	// served from a shared cache.
	header.Set("Cache-Control", "no-store")

	if !firstSet(true, h.cfg.Gallery.Enabled) {
		_ = landingTmpl.Execute(w, nil)
		return
	}

	base := baseURL(r)
	view := galleryView{
		Badges: galleryItems(base, "badges", h.cfg.Badges, h.cfg.Defaults.Badge.Gallery.Hidden,
			func(b config.Badge) (string, string, *bool) {
				return b.ID, displayTitle(b.Title, b.ID), b.Gallery.Hidden
			}),
		Graphs: galleryItems(base, "graphs", h.cfg.Graphs, h.cfg.Defaults.Graph.Gallery.Hidden,
			func(g config.Graph) (string, string, *bool) {
				return g.ID, displayTitle(g.Title, g.ID), g.Gallery.Hidden
			}),
	}
	_ = galleryTmpl.Execute(w, view)
}

// galleryItems builds Markdown snippets for the visible endpoints of one kind.
// meta extracts each item's id, display title, and per-endpoint hidden override.
func galleryItems[T any](base, kind string, items []T, def *bool, meta func(T) (id, title string, hidden *bool)) []galleryItem {
	var out []galleryItem
	for _, it := range items {
		id, title, h := meta(it)
		if !hidden(h, def) {
			out = append(out, markdownItem(base, kind, id, title))
		}
	}
	return out
}

// assetsHandler serves the embedded gallery assets under /assets/ with a long
// cache lifetime (they only change when the binary does). Directory paths get a
// 404 rather than a listing.
func assetsHandler() http.Handler {
	fileServer := http.FileServerFS(assetsFS)
	return http.StripPrefix("/assets/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "" || strings.HasSuffix(r.URL.Path, "/") {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Cache-Control", "public, max-age=3600")
		fileServer.ServeHTTP(w, r)
	}))
}

// markdownItem builds a Markdown image snippet for an endpoint, e.g.
// "![CPU](https://host/badges/cpu)". A relative URL is used if the request Host is
// unusable (see baseURL).
func markdownItem(base, kind, id, alt string) galleryItem {
	return galleryItem{Markdown: "![" + mdEscapeAlt(alt) + "](" + base + "/" + kind + "/" + id + ")"}
}

// mdAltReplacer escapes characters that would break Markdown image alt text.
var mdAltReplacer = strings.NewReplacer(`\`, `\\`, "[", `\[`, "]", `\]`, "\n", " ", "\r", " ")

func mdEscapeAlt(s string) string { return mdAltReplacer.Replace(s) }

// validHost matches a hostname or bracketed IPv6 literal with an optional port,
// rejecting any character that could break out of the Markdown/URL context.
var validHost = regexp.MustCompile(`^[A-Za-z0-9.\-:\[\]]+$`)

// baseURL builds the absolute origin (scheme://host) for snippet URLs from the
// request, honoring an X-Forwarded-Proto from a trusted proxy. The Host header is
// validated so an attacker-controlled value cannot inject into the page; an
// unusable Host yields "" (snippets fall back to relative URLs).
func baseURL(r *http.Request) string {
	if !validHost.MatchString(r.Host) {
		return ""
	}
	scheme := "http"
	switch proto := strings.TrimSpace(strings.Split(r.Header.Get("X-Forwarded-Proto"), ",")[0]); {
	case proto == "https", proto == "http":
		scheme = proto
	case r.TLS != nil:
		scheme = "https"
	}
	return scheme + "://" + r.Host
}

// hidden reports whether an endpoint should be hidden from the gallery, given its
// own gallery.hidden override and the per-type default. Shown when neither is set.
func hidden(item, def *bool) bool {
	return firstSet(false, item, def)
}
