# Kromgo

[![Tests](https://github.com/home-operations/kromgo/actions/workflows/tests.yaml/badge.svg)](https://github.com/home-operations/kromgo/actions/workflows/tests.yaml)
[![E2E](https://github.com/home-operations/kromgo/actions/workflows/e2e.yaml/badge.svg)](https://github.com/home-operations/kromgo/actions/workflows/e2e.yaml)
[![Lint](https://github.com/home-operations/kromgo/actions/workflows/lint.yaml/badge.svg)](https://github.com/home-operations/kromgo/actions/workflows/lint.yaml)
[![Release](https://img.shields.io/github/v/release/home-operations/kromgo)](https://github.com/home-operations/kromgo/releases)
[![License](https://img.shields.io/github/license/home-operations/kromgo)](LICENSE)
[![Discord](https://img.shields.io/discord/673534664354430999?label=discord&logo=discord&logoColor=white&color=blue)](https://discord.gg/home-operations)

Safely expose individual Prometheus metric values to the public web. Define named metrics backed by PromQL queries and serve them as JSON, SVG badges, sparkline charts, or raw Prometheus data — without exposing your Prometheus instance directly.

Works out of the box with [shields.io Endpoint Badges](https://shields.io/badges/endpoint-badge).

## Contents

- [How it works](#how-it-works)
- [Quick start](#quick-start)
- [Configuration](#configuration)
    - [Environment variables](#environment-variables)
    - [Metrics](#metrics)
    - [Colors](#colors)
    - [Value templates](#value-templates)
    - [History and charts](#history-and-charts)
    - [Badge font](#badge-font)
- [Index page](#index-page)
- [API reference](#api-reference)
- [Ports](#ports)
- [Image verification](#image-verification)
- [Community](#community)

## How it works

kromgo sits between the public web and your Prometheus. Each configured metric maps a URL path
(`/{name}`) to a PromQL query; a request runs the query and renders the result in the format you ask
for (`json`, `raw`, `badge`, `chart`, or `history`). Only the metrics you define are reachable —
Prometheus itself is never exposed.

## Quick start

```bash
docker run -d \
  -e PROMETHEUS_URL=http://prometheus:9090 \
  -v /path/to/config.yaml:/kromgo/config.yaml \
  -p 8080:8080 \
  ghcr.io/home-operations/kromgo:latest
```

Then query a metric:

```
GET http://localhost:8080/node_cpu_usage
```

### Docker Compose

```yaml
services:
    kromgo:
        image: ghcr.io/home-operations/kromgo:latest
        environment:
            PROMETHEUS_URL: http://prometheus:9090
        volumes:
            - ./config.yaml:/kromgo/config.yaml:ro
        ports:
            - "8080:8080"
```

## Configuration

kromgo reads its metric definitions from `/kromgo/config.yaml` inside the container. Mount your
config file there (or pass `-config /path/to/config.yaml`).

**Minimal example:**

```yaml
metrics:
    - name: node_cpu_usage
      query: "round(cluster:node_cpu:ratio_rate5m * 100, 0.1)"
      suffix: "%"
```

See the sections below for the full set of options — colors, value templates, history/charts, and
badges. A JSON Schema for editor validation is published at [config.schema.json](./config.schema.json);
point your editor's YAML language server at it for inline completion and validation.

### Environment variables

| Variable                  | Required | Default   | Description                                 |
| ------------------------- | -------- | --------- | ------------------------------------------- |
| `PROMETHEUS_URL`          | yes      | —         | URL of your Prometheus instance             |
| `SERVER_HOST`             | no       | `0.0.0.0` | Host to bind the main server                |
| `SERVER_PORT`             | no       | `8080`    | Port for the main server                    |
| `HEALTH_HOST`             | no       | `0.0.0.0` | Host to bind the health server              |
| `HEALTH_PORT`             | no       | `8888`    | Port for the health/metrics server          |
| `SERVER_LOGGING`          | no       | `false`   | Enable HTTP request access logging          |
| `SERVER_READ_TIMEOUT`     | no       | —         | HTTP read timeout (e.g. `5s`)               |
| `SERVER_WRITE_TIMEOUT`    | no       | —         | HTTP write timeout (e.g. `10s`)             |
| `QUERY_TIMEOUT`           | no       | `30s`     | Timeout applied to each Prometheus query    |
| `RATELIMIT_ENABLE`        | no       | `false`   | Enable rate limiting                        |
| `RATELIMIT_ALL`           | no       | `false`   | Rate limit all requests globally            |
| `RATELIMIT_BY_REAL_IP`    | no       | `false`   | Rate limit by `X-Real-IP` header            |
| `RATELIMIT_REQUEST_LIMIT` | no       | `100`     | Max requests per window                     |
| `RATELIMIT_WINDOW_LENGTH` | no       | `1m`      | Rate limit window duration                  |
| `LOG_LEVEL`               | no       | `info`    | Log level: `debug`, `info`, `warn`, `error` |
| `LOG_FORMAT`              | no       | `json`    | Log format: `json` or `text`                |

### Metrics

Each entry under `metrics:` defines one queryable endpoint at `/{name}`.

| Field           | Required | Description                                                                                     |
| --------------- | -------- | ----------------------------------------------------------------------------------------------- |
| `name`          | yes      | URL path segment — `node_cpu_usage` → `GET /node_cpu_usage`                                     |
| `query`         | yes      | PromQL expression, must return a single scalar or vector value                                  |
| `title`         | no       | Display label in badge/endpoint responses (defaults to `name`)                                  |
| `label`         | no       | Extract value from this metric label instead of the sample value                                |
| `prefix`        | no       | String prepended to the value in the response (e.g. `v`)                                        |
| `suffix`        | no       | String appended to the value in the response (e.g. `%`)                                         |
| `valueTemplate` | no       | Go template applied to the value before prefix/suffix — see [Value templates](#value-templates) |
| `colors`        | no       | List of color ranges for the response — see [Colors](#colors)                                   |
| `hidden`        | no       | Hide from the index page (`GET /`) — see [Index page](#index-page)                              |
| `history`       | no       | Per-metric history/chart override — see [History and charts](#history-and-charts)               |

### Colors

Assign a badge color based on the numeric value. Use `valueOverride` to replace the displayed value
text entirely.

```yaml
metrics:
    - name: node_cpu_usage
      query: "round(cluster:node_cpu:ratio_rate5m * 100, 0.1)"
      suffix: "%"
      colors:
          - { color: "green", min: 0, max: 35 }
          - { color: "orange", min: 36, max: 75 }
          - { color: "red", min: 76, max: 1000 }

    - name: ceph_health
      query: "ceph_health_status{}"
      colors:
          - { color: "green", min: 0, max: 0, valueOverride: "Healthy" }
          - { color: "orange", min: 1, max: 1, valueOverride: "Warning" }
          - { color: "red", min: 2, max: 2, valueOverride: "Critical" }
```

Supported color names: `blue`, `brightgreen`, `green`, `grey`, `lightgrey`, `orange`, `red`,
`yellow`, `yellowgreen`, `success`, `important`, `critical`, `informational`, `inactive`. Hex values
(e.g. `#e05d44`) are also accepted.

### Value templates

The `valueTemplate` field applies a [Go template](https://pkg.go.dev/text/template) to the raw
Prometheus value before `prefix` and `suffix` are added.

| Function            | Example input | Example output | Description                                                     |
| ------------------- | ------------- | -------------- | --------------------------------------------------------------- |
| `simplifyDays`      | `"1159"`      | `3y64d`        | Converts a day count to years and days                          |
| `humanBytes`        | `"1572864"`   | `1.5MiB`       | Bytes → human size with IEC binary units (KiB, MiB, GiB...)     |
| `humanSIBytes`      | `"1500000"`   | `1.5MB`        | Bytes → human size with SI decimal units (÷1000, kB, MB, GB...) |
| `humanDuration`     | `"9000"`      | `2h30m`        | Converts seconds to a compact duration string                   |
| `humanizeThousands` | `"157121"`    | `157,121`      | Adds comma thousands separators                                 |
| `toUpper`           | `"v1.31.0"`   | `V1.31.0`      | Uppercases the string                                           |
| `toLower`           | `"HEALTHY"`   | `healthy`      | Lowercases the string                                           |
| `trim`              | `" ok "`      | `ok`           | Strips leading and trailing whitespace                          |

Define reusable snippets at the top level under `templates:` and reference them by name:

```yaml
templates:
    clusterAge: "{{ . | simplifyDays }}"
    uptime: "{{ . | humanDuration }}"

metrics:
    - name: cluster_age
      query: "floor((time() - k8s_cluster_created_timestamp) / 86400)"
      valueTemplate: clusterAge # resolved from templates map
    - name: node_memory_used
      query: "node_memory_MemTotal_bytes - node_memory_MemAvailable_bytes"
      valueTemplate: "{{ . | humanBytes }}" # inline template also works
```

### History and charts

The `chart` and `history` formats run a Prometheus **range** query and return a time series. They are
**disabled by default** and must be enabled — globally and/or per metric — to limit what range data is
exposed publicly.

```yaml
history:
    enabled: true # allow format=history and format=chart
    maxDuration: "7d" # cap the requested time window (default "1h"; "0" = unlimited)

metrics:
    - name: node_cpu_usage
      query: "..."
      history:
          enabled: true # override the global setting for this metric
          maxDuration: "30d"
```

Time-window query parameters (shared by `chart` and `history`):

| Parameter | Default    | Description                                                              |
| --------- | ---------- | ------------------------------------------------------------------------ |
| `last`    | —          | Shorthand window ending now, e.g. `last=7d` (supports `s/m/h/d/y` units) |
| `start`   | end − 1h   | Window start — Unix timestamp or RFC3339                                 |
| `end`     | now        | Window end — Unix timestamp or RFC3339                                   |
| `step`    | window/100 | Resolution between points (min `1m`); supports `s/m/h/d/y` units         |

### Badge font

Badges render with an embedded default font, so kromgo works out of the box with no font file. To use
a custom TrueType font, mount it and point `badge.font` at it:

```yaml
badge:
    font: /kromgo/Verdana.ttf # optional; defaults to an embedded font
    size: 12 # optional; defaults to 11
```

## Index page

`GET /` returns an HTML page listing all visible metrics as clickable links. By default all metrics
are hidden.

Set `hideAll: false` in your config to show all metrics, then opt individual ones out with
`hidden: true`; or keep `hideAll` at its default and opt specific metrics in with `hidden: false`.
When no metrics are visible, the page displays _page intentionally blank_.

## API reference

The format is selected with the `?format=` query parameter (default `json`).

### Endpoint / JSON (default)

Compatible with the [shields.io Endpoint Badge](https://shields.io/badges/endpoint-badge).

```
GET /node_cpu_usage
```

```json
{ "schemaVersion": 1, "label": "node_cpu_usage", "message": "17.5%", "color": "green" }
```

### Raw

Returns the raw Prometheus query result.

```
GET /node_cpu_usage?format=raw
```

```json
[{ "metric": {}, "value": [1702664619.78, "17.5"] }]
```

### Badge

Returns an SVG badge directly. Styles: `flat` (default), `flat-square`, `plastic`.

```
GET /node_cpu_usage?format=badge
GET /node_cpu_usage?format=badge&style=flat-square
```

### Chart

Returns an SVG sparkline of the metric over time (requires history enabled — see
[History and charts](#history-and-charts)). Extra parameters: `width` (default 300), `height`
(default 80), `stroke` (default 2), `color` (override line color), `legend` (`false` to hide).

```
GET /node_cpu_usage?format=chart&last=24h&width=400
```

### History

Returns the raw time series as JSON (requires history enabled).

```
GET /node_cpu_usage?format=history&last=24h
```

```json
{
    "metric": "node_cpu_usage",
    "title": "node_cpu_usage",
    "start": 1702578219,
    "end": 1702664619,
    "step": 60,
    "series": [{ "labels": { "instance": "node-1" }, "data": [{ "t": 1702578219, "v": 17.5 }] }]
}
```

## Ports

| Port   | Purpose                                                        |
| ------ | -------------------------------------------------------------- |
| `8080` | Main server — metric queries                                   |
| `8888` | Health server — `/healthz`, `/readyz`, `/metrics` (Prometheus) |

The health server's `/metrics` endpoint exposes Go runtime metrics plus
`kromgo_requests_total{metric, format}` — a counter of requests handled, broken down by metric name
and response format.

## Image verification

Container images are signed with [Cosign](https://docs.sigstore.dev/cosign/overview/) keyless
signing. Verify an image before running it:

```bash
cosign verify ghcr.io/home-operations/kromgo:<tag> \
  --certificate-identity-regexp="https://github.com/home-operations/kromgo/" \
  --certificate-oidc-issuer="https://token.actions.githubusercontent.com"
```

## Community

Thanks to everyone in the [Home Operations](https://discord.gg/home-operations) Discord community.
This project began as [kashalls/kromgo](https://github.com/kashalls/kromgo).
