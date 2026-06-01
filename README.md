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
    - [Defaults](#defaults)
    - [Range queries](#range-queries)
    - [Colors](#colors)
    - [Value templates](#value-templates)
    - [History and charts](#history-and-charts)
    - [Badge font](#badge-font)
- [Index page](#index-page)
- [API reference](#api-reference)
- [Ports](#ports)
- [Rate limiting](#rate-limiting)
- [Caching](#caching)
- [Image verification](#image-verification)
- [Upgrading from kashalls/kromgo](#upgrading-from-kashallskromgo)
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

| Variable               | Required | Default   | Description                                 |
| ---------------------- | -------- | --------- | ------------------------------------------- |
| `PROMETHEUS_URL`       | yes      | —         | URL of your Prometheus instance             |
| `SERVER_HOST`          | no       | `0.0.0.0` | Host to bind the main server                |
| `SERVER_PORT`          | no       | `8080`    | Port for the main server                    |
| `HEALTH_HOST`          | no       | `0.0.0.0` | Host to bind the health server              |
| `HEALTH_PORT`          | no       | `8888`    | Port for the health/metrics server          |
| `SERVER_LOGGING`       | no       | `false`   | Enable HTTP request access logging          |
| `SERVER_READ_TIMEOUT`  | no       | —         | HTTP read timeout (e.g. `5s`)               |
| `SERVER_WRITE_TIMEOUT` | no       | —         | HTTP write timeout (e.g. `10s`)             |
| `QUERY_TIMEOUT`        | no       | `30s`     | Timeout applied to each Prometheus query    |
| `LOG_LEVEL`            | no       | `info`    | Log level: `debug`, `info`, `warn`, `error` |
| `LOG_FORMAT`           | no       | `json`    | Log format: `json` or `text`                |

### Metrics

Each entry under `metrics:` defines one queryable endpoint at `/{name}`.

| Field           | Required | Description                                                                                     |
| --------------- | -------- | ----------------------------------------------------------------------------------------------- |
| `name`          | yes      | URL path segment — `node_cpu_usage` → `GET /node_cpu_usage`                                     |
| `query`         | yes      | PromQL expression, must return a single scalar or vector value                                  |
| `type`          | no       | `instant` (default) or `range` — see [Range queries](#range-queries)                            |
| `range`         | no\*     | Range-query window when `type: range` — see [Range queries](#range-queries)                     |
| `title`         | no       | Display label in badge/endpoint responses (defaults to `name`)                                  |
| `label`         | no       | Extract value from this metric label instead of the sample value                                |
| `prefix`        | no       | String prepended to the value in the response (e.g. `v`)                                        |
| `suffix`        | no       | String appended to the value in the response (e.g. `%`)                                         |
| `valueTemplate` | no       | Go template applied to the value before prefix/suffix — see [Value templates](#value-templates) |
| `colors`        | no       | List of color ranges for the response — see [Colors](#colors)                                   |
| `hidden`        | no       | Override `defaults.hidden` for this metric — see [Index page](#index-page)                      |
| `timeseries`    | no       | Override `defaults.timeseries` for this metric — see [History and charts](#history-and-charts)  |
| `cacheSeconds`  | no       | Override `defaults.cacheSeconds` for this metric — see [Caching](#caching)                      |

### Defaults

`defaults` sets the baseline for the per-metric fields that support it; each metric overrides the
same-named field. All keys are optional.

```yaml
defaults:
    hidden: true # index visibility — true (default) hides every metric unless it opts in
    cacheSeconds: 0 # Cache-Control max-age in seconds; 0 disables caching
    timeseries: # gates the time-series output formats (format=history and format=chart)
        enabled: false
        maxDuration: 1h
```

### Range queries

By default a metric's value comes from an **instant** query at "now". Set `type: range` to instead
run a **range query** over a window and reduce it to a single value — useful for averages, peaks, or
comparing against an earlier period. The window is `end = now - offset`, `start = end - last`.

```yaml
metrics:
    - name: cpu_prev_week_avg
      type: range
      query: "cluster:node_cpu:ratio_rate5m * 100"
      range:
          last: "7d" # window length (required)
          offset: "7d" # shift the window back; here: 14d ago .. 7d ago (default: ends now)
          step: "1h" # resolution (default: last/100, min 1m)
          reduce: avg # last (default), first, avg, min, max, sum
      suffix: "%"
```

`reduce` collapses each series to one value; non-finite samples (NaN/Inf) are skipped. This is
independent of the [history/chart output formats](#history-and-charts) — a `range` metric still
returns a single value.

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

`valueTemplate` is always an inline template. To reuse one across metrics, use a YAML anchor:

```yaml
metrics:
    - name: cluster_age
      query: "floor((time() - k8s_cluster_created_timestamp) / 86400)"
      valueTemplate: &simplifyDays "{{ . | simplifyDays }}"
    - name: node_age
      query: "floor((time() - node_created_timestamp) / 86400)"
      valueTemplate: *simplifyDays
```

### History and charts

The `chart` and `history` output formats return a time series. They are **disabled by default** and
must be enabled — via `defaults.timeseries` and/or per metric — to limit what range data is exposed
publicly. (This is separate from a metric's [`type: range`](#range-queries), which only affects how
its single value is computed.)

```yaml
defaults:
    timeseries:
        enabled: true # allow format=history and format=chart
        maxDuration: "7d" # cap the requested time window (default "1h"; "0" = unlimited)

metrics:
    - name: node_cpu_usage
      query: "..."
      timeseries:
          maxDuration: "30d" # override just this metric (enabled inherited from defaults)
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

Set `defaults.hidden: false` to show all metrics, then opt individual ones out with `hidden: true`;
or keep the default and opt specific metrics in with `hidden: false`. When no metrics are visible,
the page displays _page intentionally blank_.

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

## Rate limiting

kromgo does not rate limit itself — it's meant to sit behind a reverse proxy on the public web, and
proxies do this better (shared limits across replicas, per-IP buckets, burst handling, `429`
responses). Configure it there. Examples for limiting `/` traffic to kromgo on `:8080`:

**nginx** — in the `http {}` block, then reference the zone in your `location`:

```nginx
limit_req_zone $binary_remote_addr zone=kromgo:10m rate=10r/s;

server {
    location / {
        limit_req zone=kromgo burst=20 nodelay;
        proxy_pass http://kromgo:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
    }
}
```

**Caddy** — requires the [caddy-ratelimit](https://github.com/mholt/caddy-ratelimit) module
(`xcaddy build --with github.com/mholt/caddy-ratelimit`):

```caddyfile
kromgo.example.com {
    rate_limit {
        zone kromgo {
            key    {remote_host}
            events 10
            window 1s
        }
    }
    reverse_proxy kromgo:8080
}
```

**Envoy** — the built-in [local rate limit](https://www.envoyproxy.io/docs/envoy/latest/configuration/http/http_filters/local_rate_limit_filter)
HTTP filter (100 requests/minute per listener):

```yaml
http_filters:
    - name: envoy.filters.http.local_ratelimit
      typed_config:
          "@type": type.googleapis.com/envoy.extensions.filters.http.local_ratelimit.v3.LocalRateLimit
          stat_prefix: kromgo_rate_limiter
          token_bucket:
              max_tokens: 100
              tokens_per_fill: 100
              fill_interval: 60s
          filter_enabled:
              default_value: { numerator: 100, denominator: HUNDRED }
          filter_enforced:
              default_value: { numerator: 100, denominator: HUNDRED }
```

**Traefik v3** — a `rateLimit` middleware attached to the router (dynamic file config; the
Kubernetes `Middleware` CRD takes the same `rateLimit` spec):

```yaml
http:
    middlewares:
        kromgo-ratelimit:
            rateLimit:
                average: 10
                burst: 20
                period: 1s
    routers:
        kromgo:
            rule: Host(`kromgo.example.com`)
            service: kromgo
            middlewares:
                - kromgo-ratelimit
```

## Caching

Caching has two halves, and kromgo owns the half only it can know: **how long a value stays fresh**.
Set `cacheSeconds` (globally and/or per metric) and kromgo emits `Cache-Control: public, max-age=N`
on successful responses and includes `cacheSeconds` in the shields.io endpoint JSON. Errors are always
sent `no-store`. Caching is off by default (`cacheSeconds: 0`).

```yaml
cacheSeconds: 300 # global default for every metric

metrics:
    - name: node_cpu_usage # changes every scrape — short TTL
      query: "..."
      cacheSeconds: 30
    - name: cluster_age # changes once a day — long TTL
      query: "..."
      cacheSeconds: 3600
```

The **other half — actually storing responses — is the edge's job**, and any cache that honors
`Cache-Control` (a CDN, Varnish, nginx `proxy_cache`) will then cache each metric for exactly the TTL
kromgo advertised, with no per-metric tuning at the proxy. shields.io already respects `cacheSeconds`,
so public badges are cached without any proxy at all.

If you want the reverse proxy itself to cache, enable its HTTP cache and let it honor the origin
headers — for example, nginx:

```nginx
proxy_cache_path /var/cache/nginx levels=1:2 keys_zone=kromgo:10m max_size=100m;

server {
    location / {
        proxy_cache kromgo;            # respects kromgo's Cache-Control
        add_header X-Cache-Status $upstream_cache_status;
        proxy_pass http://kromgo:8080;
    }
}
```

Caddy (via the [cache-handler](https://github.com/caddyserver/cache-handler) plugin), Traefik, and
Envoy can cache too, but generally need a plugin or an external cache/CDN; the simplest setup is to
front kromgo with a CDN and let `cacheSeconds` drive it.

## Image verification

Container images are signed with [Cosign](https://docs.sigstore.dev/cosign/overview/) keyless
signing. Verify an image before running it:

```bash
cosign verify ghcr.io/home-operations/kromgo:<tag> \
  --certificate-identity-regexp="https://github.com/home-operations/kromgo/" \
  --certificate-oidc-issuer="https://token.actions.githubusercontent.com"
```

## Upgrading from kashalls/kromgo

This fork is functionally compatible — metric config, query semantics, and response
formats are unchanged — but a few deployment details changed:

| Change                                                                                                                                                                                                                                                                                                                                                         | Action                                                                                                                                                                                |
| -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **Image moved** to `ghcr.io/home-operations/kromgo`.                                                                                                                                                                                                                                                                                                           | Update your image reference (and the cosign identity, if you verify).                                                                                                                 |
| **Badge font no longer bundled.** The image no longer ships `Verdana.ttf`; kromgo now uses an embedded default font. A config still pointing `badge.font` at `Verdana.ttf` will fail at startup.                                                                                                                                                               | Remove `badge.font` to use the embedded font, or mount a TrueType font and point `badge.font` at its path.                                                                            |
| **`LOG_FORMAT=test` renamed to `LOG_FORMAT=text`.**                                                                                                                                                                                                                                                                                                            | Set `LOG_FORMAT=text` for human-readable logs (default remains JSON).                                                                                                                 |
| **Built-in rate limiting removed** (`RATELIMIT_*` env vars).                                                                                                                                                                                                                                                                                                   | Rate limit at your reverse proxy — see [Rate limiting](#rate-limiting).                                                                                                               |
| **Config schema reorganized.** Top-level `hideAll`/`history`/`cacheSeconds` defaults moved under a `defaults:` block (`defaults.hidden`, `defaults.timeseries`, `defaults.cacheSeconds`); per-metric `history` is now `timeseries`; the named-`templates` map was removed. (`range` is now a metric query [type](#range-queries), not the history/chart gate.) | Move global defaults under `defaults:`, rename per-metric `history` → `timeseries`, and inline value templates (use a YAML anchor to reuse one). See [Configuration](#configuration). |
| **Missing `PROMETHEUS_URL` now fails fast** instead of starting degraded.                                                                                                                                                                                                                                                                                      | Ensure `PROMETHEUS_URL` (or `prometheus` in config) is set.                                                                                                                           |
| **Schema URL** in `config.yaml` examples.                                                                                                                                                                                                                                                                                                                      | Point `# yaml-language-server: $schema=` at `home-operations/kromgo`.                                                                                                                 |

Release tags drop the `v` prefix (e.g. `0.11.0`, not `v0.11.0`); pin image tags accordingly.

## Community

Thanks to everyone in the [Home Operations](https://discord.gg/home-operations) Discord community.
This project began as [kashalls/kromgo](https://github.com/kashalls/kromgo).
