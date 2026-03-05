package kromgo

import (
	"html/template"
	"net/http"

	"github.com/kashalls/kromgo/cmd/kromgo/init/configuration"
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

func (h *KromgoHandler) IndexHandler(w http.ResponseWriter, r *http.Request) {
	var visible []configuration.Metric
	for _, m := range h.Config.Metrics {
		if !isHidden(m, h.Config.HideAll) {
			visible = append(visible, m)
		}
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	indexTmpl.Execute(w, visible)
}

func isHidden(m configuration.Metric, hideAll *bool) bool {
	globalDefault := true // default: hide all when not specified
	if hideAll != nil {
		globalDefault = *hideAll
	}
	if m.Hidden != nil {
		return *m.Hidden
	}
	return globalDefault
}
