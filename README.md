# Kromgo

Safely expose individual Prometheus metric values to the public web. Define named metrics backed by PromQL queries and serve them as JSON, SVG badges, or raw Prometheus data — without exposing your Prometheus instance directly.

Works out of the box with [shields.io Endpoint Badges](https://shields.io/badges/endpoint-badge).

## Quick Start

```bash
docker run -d \
  -e PROMETHEUS_URL=http://prometheus:9090 \
  -v /path/to/config.yaml:/kromgo/config.yaml \
  -p 8080:8080 \
  ghcr.io/kashalls/kromgo:latest
```

Then query a metric:

```
GET http://localhost:8080/node_cpu_usage
```

## Configuration

Kromgo reads its metric definitions from `/kromgo/config.yaml` inside the container. Mount your config file there.

**Minimal example:**

```yaml
metrics:
  - name: node_cpu_usage
    query: "round(cluster:node_cpu:ratio_rate5m * 100, 0.1)"
    suffix: "%"
```

See [config.yaml.example](./config.yaml.example) for a full example with colors, templates, and badges.

### Docker Compose

```yaml
services:
  kromgo:
    image: ghcr.io/kashalls/kromgo:latest
    environment:
      PROMETHEUS_URL: http://prometheus:9090
    volumes:
      - ./config.yaml:/kromgo/config.yaml:ro
    ports:
      - "8080:8080"
```

## Environment Variables

| Variable | Required | Default | Description |
|---|---|---|---|
| `PROMETHEUS_URL` | yes | — | URL of your Prometheus instance |
| `SERVER_HOST` | no | `0.0.0.0` | Host to bind the main server |
| `SERVER_PORT` | no | `8080` | Port for the main server |
| `HEALTH_HOST` | no | `0.0.0.0` | Host to bind the health server |
| `HEALTH_PORT` | no | `8888` | Port for the health/metrics server |
| `SERVER_LOGGING` | no | `false` | Enable HTTP request logging |
| `SERVER_READ_TIMEOUT` | no | — | HTTP read timeout (e.g. `5s`) |
| `SERVER_WRITE_TIMEOUT` | no | — | HTTP write timeout (e.g. `10s`) |
| `RATELIMIT_ENABLE` | no | `false` | Enable rate limiting |
| `RATELIMIT_ALL` | no | `false` | Rate limit all requests globally |
| `RATELIMIT_BY_REAL_IP` | no | `false` | Rate limit by `X-Real-IP` header |
| `RATELIMIT_REQUEST_LIMIT` | no | `100` | Max requests per window |
| `RATELIMIT_WINDOW_LENGTH` | no | `1m` | Rate limit window duration |
| `LOG_LEVEL` | no | `info` | Log level: `debug`, `info`, `warn`, `error` |
| `LOG_FORMAT` | no | `json` | Log format: `json` or `text` |

## Metric Configuration

Each entry under `metrics:` defines one queryable endpoint at `/{name}`.

| Field | Required | Description |
|---|---|---|
| `name` | yes | URL path segment — `node_cpu_usage` → `GET /node_cpu_usage` |
| `query` | yes | PromQL expression, must return a single scalar or vector value |
| `title` | no | Display label in badge/endpoint responses (defaults to `name`) |
| `label` | no | Extract value from this metric label instead of the sample value |
| `prefix` | no | String prepended to the value in the response (e.g. `v`) |
| `suffix` | no | String appended to the value in the response (e.g. `%`) |
| `valueTemplate` | no | Go template applied to the value before prefix/suffix — see [Value Templates](#value-templates) |
| `colors` | no | List of color ranges for the response — see [Colors](#colors) |
| `hidden` | no | Hide from the index page (`GET /`) — see [Index Page](#index-page) |

### Colors

Assign a badge color based on the numeric value. Use `valueOverride` to replace the displayed value text entirely.

```yaml
metrics:
  - name: node_cpu_usage
    query: "round(cluster:node_cpu:ratio_rate5m * 100, 0.1)"
    suffix: "%"
    colors:
      - { color: "green",  min: 0,  max: 35  }
      - { color: "orange", min: 36, max: 75  }
      - { color: "red",    min: 76, max: 1000 }

  - name: ceph_health
    query: "ceph_health_status{}"
    colors:
      - { color: "green",  min: 0, max: 0, valueOverride: "Healthy"  }
      - { color: "orange", min: 1, max: 1, valueOverride: "Warning"  }
      - { color: "red",    min: 2, max: 2, valueOverride: "Critical" }
```

Supported color names: `blue`, `brightgreen`, `green`, `grey`, `lightgrey`, `orange`, `red`, `yellow`, `yellowgreen`, `success`, `important`, `critical`, `informational`, `inactive`. Hex values (e.g. `#e05d44`) are also accepted.

## Value Templates

The `valueTemplate` field applies a [Go template](https://pkg.go.dev/text/template) to the raw Prometheus value before `prefix` and `suffix` are added.

### Built-in functions

| Function | Example input | Example output | Description |
|---|---|---|---|
| `simplifyDays` | `"1159"` | `3y64d` | Converts a day count to years and days |
| `humanBytes` | `"1572864"` | `1.5MB` | Converts bytes to a human-readable size |
| `humanDuration` | `"9000"` | `2h30m` | Converts seconds to a compact duration string |
| `toUpper` | `"v1.31.0"` | `V1.31.0` | Uppercases the string |
| `toLower` | `"HEALTHY"` | `healthy` | Lowercases the string |
| `trim` | `" ok "` | `ok` | Strips leading and trailing whitespace |

### Inline template

```yaml
metrics:
  - name: cluster_age
    query: "floor((time() - k8s_cluster_created_timestamp) / 86400)"
    valueTemplate: "{{ . | simplifyDays }}"   # 1159 → 3y64d

  - name: node_memory_used
    query: "node_memory_MemTotal_bytes - node_memory_MemAvailable_bytes"
    valueTemplate: "{{ . | humanBytes }}"     # 1572864 → 1.5MB
```

### Named templates

Define reusable snippets at the top level and reference them by name:

```yaml
templates:
  clusterAge: "{{ . | simplifyDays }}"
  uptime:     "{{ . | humanDuration }}"

metrics:
  - name: cluster_age
    query: "floor((time() - k8s_cluster_created_timestamp) / 86400)"
    valueTemplate: clusterAge   # resolved from templates map

  - name: node_uptime
    query: "time() - node_boot_time_seconds"
    valueTemplate: uptime
```

## Index Page

`GET /` returns an HTML page listing all visible metrics as clickable links. By default all metrics are hidden.

Set `hideAll: false` in your config to show all metrics, then opt individual ones out with `hidden: true`, or keep `hideAll` at its default and opt specific metrics in with `hidden: false`.

```yaml
hideAll: false   # show all metrics on the index page by default

metrics:
  - name: node_cpu_usage
    query: "..."
  - name: internal_metric
    query: "..."
    hidden: true   # this one won't appear on the index page
```

When no metrics are visible, the page displays *page intentionally blank*.

## API Reference

### Endpoint format (default)

Compatible with [shields.io Endpoint Badge](https://shields.io/badges/endpoint-badge).

```
GET /node_cpu_usage
```

```json
{
    "schemaVersion": 1,
    "label": "node_cpu_usage",
    "message": "17.5%",
    "color": "green"
}
```

### Raw format

Returns the raw Prometheus query result.

```
GET /node_cpu_usage?format=raw
```

```json
[{"metric": {}, "value": [1702664619.78, "17.5"]}]
```

### Badge format

Returns an SVG badge directly.

```
GET /node_cpu_usage?format=badge
GET /node_cpu_usage?format=badge&style=flat-square
GET /node_cpu_usage?format=badge&style=plastic
```

```
content-type: image/svg+xml
<svg xmlns="http://www.w3.org/2000/svg" ...
```

## Ports

| Port | Purpose |
|---|---|
| `8080` | Main server — metric queries |
| `8888` | Health server — `/healthz`, `/readyz`, `/metrics` (Prometheus) |

The health server's `/metrics` endpoint exposes Go runtime metrics plus `kromgo_requests_total{metric, format}` — a counter of requests handled, broken down by metric name and response format.

## Image Verification

Container images are signed with [Cosign](https://docs.sigstore.dev/cosign/overview/) keyless signing. Verify an image before running it:

```bash
cosign verify ghcr.io/kashalls/kromgo:<tag> \
  --certificate-identity-regexp="https://github.com/kashalls/kromgo/" \
  --certificate-oidc-issuer="https://token.actions.githubusercontent.com"
```

## 🤝 Gratitude and Thanks

Thanks to all of the people at the [Home Operations](https://discord.gg/home-operations) Discord community. Be sure to check it out, it's a blast!
