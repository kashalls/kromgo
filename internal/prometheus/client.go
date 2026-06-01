// Package prometheus wraps the Prometheus HTTP API with a small client that
// applies a per-query timeout and logs query warnings.
package prometheus

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/prometheus/client_golang/api"
	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"
)

// Client is a thin wrapper around the Prometheus v1 query API.
type Client struct {
	api     v1.API
	timeout time.Duration
}

// New builds a Client targeting address. timeout bounds each query (0 means no timeout).
func New(address string, timeout time.Duration) (*Client, error) {
	if address == "" {
		return nil, fmt.Errorf("no prometheus url provided")
	}
	c, err := api.NewClient(api.Config{Address: address})
	if err != nil {
		return nil, fmt.Errorf("creating prometheus client: %w", err)
	}
	return &Client{api: v1.NewAPI(c), timeout: timeout}, nil
}

// Query runs an instant query at t, logging any warnings.
func (c *Client) Query(ctx context.Context, query string, t time.Time) (model.Value, error) {
	ctx, cancel := c.withTimeout(ctx)
	defer cancel()
	value, warnings, err := c.api.Query(ctx, query, t)
	logWarnings(ctx, query, warnings)
	if err != nil {
		return nil, fmt.Errorf("prometheus query: %w", err)
	}
	return value, nil
}

// QueryRange runs a range query over r, logging any warnings.
func (c *Client) QueryRange(ctx context.Context, query string, r v1.Range) (model.Value, error) {
	ctx, cancel := c.withTimeout(ctx)
	defer cancel()
	value, warnings, err := c.api.QueryRange(ctx, query, r)
	logWarnings(ctx, query, warnings)
	if err != nil {
		return nil, fmt.Errorf("prometheus range query: %w", err)
	}
	return value, nil
}

func (c *Client) withTimeout(ctx context.Context) (context.Context, context.CancelFunc) {
	if c.timeout <= 0 {
		return ctx, func() {}
	}
	return context.WithTimeout(ctx, c.timeout)
}

func logWarnings(ctx context.Context, query string, warnings v1.Warnings) {
	for _, w := range warnings {
		slog.WarnContext(ctx, "prometheus query warning", "query", query, "warning", w)
	}
}
