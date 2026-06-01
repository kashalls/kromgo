package kromgo

import (
	"html/template"
	"net/http"
)

var indexTmpl = template.Must(template.New("index").Parse(`<!DOCTYPE html>
<html>
<body>
{{- if .}}
{{- range .}}
<a href="{{.Href}}">{{.Label}}</a><br>
{{- end}}
{{- else}}
<i>page intentionally blank</i>
{{- end}}
</body>
</html>`))

type indexLink struct {
	Href  string
	Label string
}

// index renders an HTML page listing all visible badges and graphs.
func (h *Handler) index(w http.ResponseWriter, _ *http.Request) {
	var links []indexLink
	for _, b := range h.cfg.Badges {
		if !hidden(b.Hidden, h.cfg.Defaults.Hidden) {
			links = append(links, indexLink{Href: "/badges/" + b.ID, Label: displayTitle(b.Title, b.ID)})
		}
	}
	for _, g := range h.cfg.Graphs {
		if !hidden(g.Hidden, h.cfg.Defaults.Hidden) {
			links = append(links, indexLink{Href: "/graphs/" + g.ID, Label: displayTitle(g.Title, g.ID)})
		}
	}
	w.Header().Set("Content-Type", mimeHTML)
	_ = indexTmpl.Execute(w, links)
}

// hidden reports whether an endpoint should be hidden from the index, given its own
// override and the default. Defaults to hidden when neither is set.
func hidden(item, def *bool) bool {
	if item != nil {
		return *item
	}
	if def != nil {
		return *def
	}
	return true // default: hide all when not specified
}
