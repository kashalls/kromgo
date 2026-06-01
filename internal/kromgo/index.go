package kromgo

import (
	"html/template"
	"net/http"

	"github.com/home-operations/kromgo/internal/config"
)

var indexTmpl = template.Must(template.New("index").Parse(`<!DOCTYPE html>
<html>
<body>
{{- if .}}
{{- range .}}
<a href="/{{.Name}}">{{.Name}}</a><br>
{{- end}}
{{- else}}
<i>page intentionally blank</i>
{{- end}}
</body>
</html>`))

// index renders an HTML page listing all visible metrics.
func (h *Handler) index(w http.ResponseWriter, _ *http.Request) {
	var visible []config.Metric
	for _, m := range h.cfg.Metrics {
		if !isHidden(m, h.cfg.HideAll) {
			visible = append(visible, m)
		}
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_ = indexTmpl.Execute(w, visible)
}

// isHidden reports whether a metric should be hidden from the index page.
func isHidden(m config.Metric, hideAll *bool) bool {
	if m.Hidden != nil {
		return *m.Hidden
	}
	if hideAll != nil {
		return *hideAll
	}
	return true // default: hide all when not specified
}
