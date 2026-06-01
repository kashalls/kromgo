// Package kromgo implements the HTTP handlers that turn Prometheus query results
// into SVG badges, shields.io / kromgo JSON, and SVG sparkline graphs.
package kromgo

import (
	"fmt"
	"net/http"

	"github.com/home-operations/kromgo/internal/config"
	"github.com/home-operations/kromgo/internal/prometheus"
	"github.com/prometheus/common/model"
)

// Output format query-parameter values.
const (
	formatSVG     = "svg"
	formatJSON    = "json"
	formatShields = "shields"
)

// Handler serves badge and graph endpoints backed by Prometheus queries.
type Handler struct {
	cfg    config.KromgoConfig
	badges map[string]*resolvedBadge
	graphs map[string]*resolvedGraph
	prom   *prometheus.Client
	gen    *badgeRenderer
}

// New builds a Handler from config and a Prometheus client. Per-endpoint CEL
// expressions and durations are compiled/parsed here, so malformed config fails
// at startup rather than on a request.
func New(cfg config.KromgoConfig, prom *prometheus.Client) (*Handler, error) {
	gen, err := newBadgeRenderer(cfg.Defaults.Badge)
	if err != nil {
		return nil, err
	}

	env, err := newCELEnv()
	if err != nil {
		return nil, fmt.Errorf("building CEL environment: %w", err)
	}

	badges := make(map[string]*resolvedBadge, len(cfg.Badges))
	for _, b := range cfg.Badges {
		rb, err := resolveBadge(b, cfg.Defaults, env)
		if err != nil {
			return nil, err
		}
		badges[b.ID] = rb
	}

	graphs := make(map[string]*resolvedGraph, len(cfg.Graphs))
	for _, g := range cfg.Graphs {
		rg, err := resolveGraph(g, cfg.Defaults)
		if err != nil {
			return nil, err
		}
		graphs[g.ID] = rg
	}

	return &Handler{cfg: cfg, badges: badges, graphs: graphs, prom: prom, gen: gen}, nil
}

// Mux returns the application router: the index page and per-type endpoints.
func (h *Handler) Mux() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /{$}", h.index)
	mux.Handle("GET /assets/", assetsHandler())
	mux.HandleFunc("GET /badges/{id}", h.serveBadge)
	mux.HandleFunc("GET /graphs/{id}", h.serveGraph)
	return mux
}

// labelMap converts a Prometheus label set to a plain string map for CEL/JSON use.
func labelMap(m model.Metric) map[string]string {
	out := make(map[string]string, len(m))
	for k, v := range m {
		out[string(k)] = string(v)
	}
	return out
}
