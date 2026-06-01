# Kromgo

[![Tests](https://github.com/home-operations/kromgo/actions/workflows/tests.yaml/badge.svg)](https://github.com/home-operations/kromgo/actions/workflows/tests.yaml)
[![E2E](https://github.com/home-operations/kromgo/actions/workflows/e2e.yaml/badge.svg)](https://github.com/home-operations/kromgo/actions/workflows/e2e.yaml)
[![Lint](https://github.com/home-operations/kromgo/actions/workflows/lint.yaml/badge.svg)](https://github.com/home-operations/kromgo/actions/workflows/lint.yaml)
[![Release](https://img.shields.io/github/v/release/home-operations/kromgo)](https://github.com/home-operations/kromgo/releases)
[![License](https://img.shields.io/github/license/home-operations/kromgo)](LICENSE)
[![Discord](https://img.shields.io/discord/673534664354430999?label=discord&logo=discord&logoColor=white&color=blue)](https://discord.gg/home-operations)

Safely expose individual Prometheus metric values to the public web. Define named endpoints backed by PromQL queries and serve them as SVG badges, themed SVG/PNG graphs, or JSON — without exposing your Prometheus instance directly.

Badges render as shields.io-style SVG, so you can embed `/badges/{id}` straight into an `<img>` tag — no shields.io round-trip required (though it's still supported via `?format=shields`).

## How it works

kromgo sits between the public web and your Prometheus. You define two kinds of endpoint:

- **Badges** (`/badges/{id}`) render an instant value as an SVG badge, shields.io JSON, or kromgo JSON.
- **Graphs** (`/graphs/{id}`) render a time series as a themed SVG/PNG chart or JSON.

Each maps a URL path to a PromQL query. Only the endpoints you define are reachable — Prometheus itself is never exposed.

The root path `/` serves a **gallery** that previews every endpoint next to its copy-paste Markdown snippet — handy for grabbing a badge for a README.

## Quick start

```bash
docker run -d \
  -e PROMETHEUS_URL=http://prometheus:9090 \
  -v /path/to/config.yaml:/config/config.yaml \
  -p 8080:8080 \
  ghcr.io/home-operations/kromgo:latest
```

Then embed or query a badge:

```html
<img src="http://localhost:8080/badges/node_cpu_usage" />
```

### Docker Compose

```yaml
services:
    kromgo:
        image: ghcr.io/home-operations/kromgo:latest
        environment:
            PROMETHEUS_URL: http://prometheus:9090
        volumes:
            - ./config.yaml:/config/config.yaml:ro
        ports:
            - "8080:8080"
```

## Configuration

kromgo reads its endpoint definitions from `/config/config.yaml` inside the container. Mount your
config file there (or pass `-config /path/to/config.yaml`).

**Minimal example:**

```yaml
badges:
    - id: node_cpu_usage
      query: "round(cluster:node_cpu:ratio_rate5m * 100, 0.1)"
      value: string(result) + "%"
```

A JSON Schema for editor validation is published at [config.schema.json](./config.schema.json);
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

### Defaults

`defaults` sets the baseline for the per-endpoint fields that support it; each endpoint overrides the
same-named field. All keys are optional.

```yaml
defaults:
    gallery: true # serve the HTML gallery at "/" (default); false serves a minimal landing page
    hidden: true # gallery visibility — true (default) hides every endpoint unless it opts in
    cacheSeconds: 0 # Cache-Control max-age in seconds; 0 disables caching
    badge:
        font: go-regular # go-regular (default), go-bold, go-medium, go-mono
        size: 11 # badge font size in points
        style: flat # flat (default), flat-square, or plastic
    graph:
        maxDuration: 1h # cap on a graph's requested window ("0" = unlimited)
        width: 600 # image width in px
        height: 200 # image height in px
        legend: true # show the series legend
        theme: light # color theme — see Themes below
        font: roboto # text font — see Themes below
```

### Badges

Each entry under `badges:` defines an instant-value endpoint at `/badges/{id}`.

| Field          | Required | Description                                                                       |
| -------------- | -------- | --------------------------------------------------------------------------------- |
| `id`           | yes      | URL path segment — `cpu` → `GET /badges/cpu`                                      |
| `query`        | yes      | PromQL expression returning a single scalar or vector value                       |
| `title`        | no       | Display label on the badge (defaults to `id`)                                     |
| `type`         | no       | `instant` (default) or `range` — see [Range badges](#range-badges)                |
| `range`        | no\*     | Range-query window when `type: range`                                             |
| `value`        | no       | CEL expression for the displayed string — see [Value and color](#value-and-color) |
| `color`        | no       | CEL expression for the color — see [Value and color](#value-and-color)            |
| `style`        | no       | `flat` (default), `flat-square`, or `plastic`                                     |
| `icon`         | no       | A Material Design Icon on the SVG badge, e.g. `mdi:server-outline` — see below    |
| `hidden`       | no       | Override `defaults.hidden` for this badge                                         |
| `cacheSeconds` | no       | Override `defaults.cacheSeconds` for this badge                                   |

#### Icons

`icon` renders a [Material Design Icon](https://pictogrammers.com/library/mdi/) on the left of the SVG
badge, written as `mdi:<name>` (e.g. `mdi:server-outline`). It is **SVG-only** — the `shields` and
`json` formats have no icon field and ignore it. The icon sits left of the `title`; with an icon and
no `title`, the badge shows just the icon and the value (the `id` fallback is suppressed).

```yaml
badges:
    - id: nodes
      query: count(kube_node_info)
      icon: mdi:server-outline
      title: Nodes
```

The **entire** Material Design Icons set (~7,400 glyphs) is embedded in the binary — no network or
disk access at runtime — so any `mdi:<name>` from [the library](https://pictogrammers.com/library/mdi/)
works (e.g. `mdi:kubernetes`, `mdi:database-outline`, `mdi:rocket-launch`). The set is stored
compressed (~0.8 MB) and decoded into memory on first use. An unknown name fails fast at startup. The
icon data is built from the `@mdi/svg` npm package **at build time** (not committed) — see
[Building from source](#building-from-source).

#### Range badges

By default a badge's value comes from an **instant** query at "now". Set `type: range` to instead run
a **range query** over a window and reduce it to a single value — useful for averages, peaks, or
comparing against an earlier period. The window is `end = now - offset`, `start = end - last`.

```yaml
badges:
    - id: cpu_prev_week_avg
      type: range
      query: "cluster:node_cpu:ratio_rate5m * 100"
      range:
          last: "7d" # window length (required)
          offset: "7d" # shift the window back; here: 14d ago .. 7d ago (default: ends now)
          step: "1h" # resolution (default: last/100, min 1m)
          reduce: avg # last (default), first, avg, min, max, sum
      value: string(result) + "%"
```

`reduce` collapses each series to one value; non-finite samples (NaN/Inf) are skipped.

### Value and color

`value` and `color` are [CEL](https://cel.dev) expressions. CEL is sandboxed (no environment, file,
or network access) and compiled once at startup, so a malformed expression fails fast rather than per
request. Each expression receives two variables:

| Variable | Type                  | Description                                              |
| -------- | --------------------- | -------------------------------------------------------- |
| `result` | `double`              | The sample value (for `type: range`, the reduced value). |
| `labels` | `map(string, string)` | The sample's labels, e.g. `labels["instance"]`.          |

- **`value`** must return a string — the message shown on the badge. Defaults to `string(result)`.
- **`color`** must return a string — a [shields.io color name](https://shields.io) (`green`,
  `orange`, `red`, `blue`, `grey`, …) or a hex value like `"#e05d44"`. Omit for no color.

```yaml
badges:
    # numeric value with a unit + threshold coloring
    - id: cpu
      query: "round(avg(...) * 100, 0.1)"
      value: string(result) + "%"
      color: 'result < 35 ? "green" : result < 75 ? "orange" : "red"'

    # value taken from a label, falling back if it's absent
    - id: version
      query: 'label_replace(build_info, "v", "$1", "version", "v(.+)")'
      value: labels[?"v"].orValue("unknown")

    # guard a possibly-NaN ratio (e.g. divide-by-zero) before formatting
    - id: hit_ratio
      query: cache_hits / (cache_hits + cache_misses)
      value: 'math.isNaN(result) ? "n/a" : humanizeFloat(math.round(result * 100.0)) + "%"'

    # enum → text + color
    - id: ceph_health
      query: ceph_health_status
      value: 'result == 0.0 ? "Healthy" : result == 1.0 ? "Warning" : "Critical"'
      color: 'result == 0.0 ? "green" : result == 1.0 ? "orange" : "red"'
```

Besides CEL's built-ins (arithmetic, comparisons, ternary `?:`, `in`) the environment enables:

- the **`strings`** extension — `startsWith`, `matches`, `replace`, `substring`, `upperAscii`, …
- the **`math`** extension — `math.round`, `math.abs`, `math.floor`/`ceil`, `math.least`/`greatest`
  (clamping), and `math.isNaN`/`isInf`/`isFinite` to guard non-finite values (Prometheus returns
  `NaN` for e.g. division by zero, which would otherwise render literally on the badge);
- **optional types** — `labels[?"k"].orValue("default")` for a label that may be absent.

On top of those, these humanizer functions are available (byte and number formatting come from
[go-humanize](https://github.com/dustin/go-humanize)):

| Function                   | Example                      | Result    | Notes                                  |
| -------------------------- | ---------------------------- | --------- | -------------------------------------- |
| `humanizeBytes(result)`    | `humanizeBytes(1572864.0)`   | `1.5 MiB` | IEC binary units                       |
| `humanizeSIBytes(result)`  | `humanizeSIBytes(1500000.0)` | `1.5 MB`  | SI decimal units                       |
| `humanizeNumber(result)`   | `humanizeNumber(157121.0)`   | `157,121` | comma grouping                         |
| `humanizeFloat(result)`    | `humanizeFloat(2.50)`        | `2.5`     | plain decimal, trailing zeros stripped |
| `humanizeDuration(result)` | `humanizeDuration(9000.0)`   | `2h30m`   | **seconds** → compact time span        |
| `humanizeDays(result)`     | `humanizeDays(5961600.0)`    | `69d`     | **seconds** → whole days, no roll-up   |

`humanizeDuration` takes **seconds** (so it drops onto a `time() - created_ts` query directly) and
adapts to the magnitude, emitting the up-to-three most-significant units — `90` → `1m30s`, `9000` →
`2h30m`, `40348800` → `1y3mo12d`. Months render as `mo` so they never collide with minutes (`m`) in
the same string.

Two gotchas around `result` (a `double`):

- **Numeric literals.** Ordered comparisons accept plain integers — `result < 35` works (kromgo
  enables CEL's cross-type numeric comparisons). Equality and arithmetic do **not**: write a decimal
  literal there, e.g. `result == 0.0` (not `== 0`) and `result * 100.0` (not `* 100`). A mismatch is
  a compile error caught at startup, not a runtime surprise.
- **Missing labels.** Indexing a label that isn't present errors. Use optional indexing —
  `labels[?"k"].orValue("n/a")` — or the ternary `"k" in labels ? labels["k"] : "n/a"`.

### Graphs

Each entry under `graphs:` defines a time-series endpoint at `/graphs/{id}`. Defining a graph is the
opt-in to expose range data for that query — there is no separate enable flag. Charts are rendered by
[go-analyze/charts](https://github.com/go-analyze/charts) as **SVG** (default) or **PNG**
(`?format=png`).

| Field          | Required | Description                                                                      |
| -------------- | -------- | -------------------------------------------------------------------------------- |
| `id`           | yes      | URL path segment — `cpu` → `GET /graphs/cpu`                                     |
| `query`        | yes      | PromQL expression run as a range query                                           |
| `title`        | no       | Display label (defaults to `id`)                                                 |
| `maxDuration`  | no       | Cap on the requested window (overrides `defaults.graph.maxDuration`)             |
| `width`        | no       | Image width in px (overrides `defaults.graph.width`)                             |
| `height`       | no       | Image height in px (overrides `defaults.graph.height`)                           |
| `legend`       | no       | Show the series legend (overrides `defaults.graph.legend`)                       |
| `theme`        | no       | Color theme (overrides `defaults.graph.theme`) — see [Themes](#themes-and-fonts) |
| `font`         | no       | Text font (overrides `defaults.graph.font`) — see [Themes](#themes-and-fonts)    |
| `hidden`       | no       | Override `defaults.hidden` for this graph                                        |
| `cacheSeconds` | no       | Override `defaults.cacheSeconds` for this graph                                  |

```yaml
graphs:
    - id: node_cpu_usage
      query: "cluster:node_cpu:ratio_rate5m * 100"
      maxDuration: "30d"
      width: 800
      theme: catppuccin-mocha
```

The time window is chosen by these query parameters:

| Parameter | Default    | Description                                                              |
| --------- | ---------- | ------------------------------------------------------------------------ |
| `last`    | —          | Shorthand window ending now, e.g. `last=7d` (supports `s/m/h/d/y` units) |
| `start`   | end − 1h   | Window start — Unix timestamp or RFC3339                                 |
| `end`     | now        | Window end — Unix timestamp or RFC3339                                   |
| `step`    | window/100 | Resolution between points (min `1m`); supports `s/m/h/d/y` units         |

The rendering fields `width`, `height`, `legend`, and `theme`, plus the output `format` (`svg`/`png`),
may also be overridden per request via query parameters, e.g.
`/graphs/node_cpu_usage?theme=dracula&format=png&width=800&last=24h`. (`font` is config-only — it's
resolved once at startup.)

#### Themes and fonts

`theme` accepts a [go-analyze/charts](https://github.com/go-analyze/charts) built-in or one of
kromgo's bundled palettes (an unknown value falls back to the default):

- **Built-in:** `light` (default), `dark`, `vivid-light`, `vivid-dark`, `grafana`, `ant`,
  `nature-light`, `nature-dark`, `retro`, `ocean`, `slate`, `gray`, `winter`, `spring`, `summer`,
  `fall`.
- **Bundled:** `catppuccin-latte`, `catppuccin-frappe`, `catppuccin-macchiato`, `catppuccin-mocha`
  (via the official [catppuccin/go](https://github.com/catppuccin/go) palette), `dracula`, `monokai`,
  `night-owl`.

`font` accepts a built-in name: `roboto` (default), `notosans`, `notosans-bold`, or the embedded Go
family `go-regular` / `go-bold` / `go-medium` / `go-mono`. (Badges use the Go family; `defaults.badge.font`
defaults to `go-regular`.) Fonts are compiled into the binary — there's no reading from disk, so add a
new face by PRing it into the registry. An unknown name fails fast at startup.

## Gallery

`GET /` serves a gallery: a responsive page (up to three columns, collapsing to one on mobile) that
previews every visible badge and graph and shows the copy-pasteable Markdown snippet for each — the
preview is rendered from that same snippet with [marked](https://github.com/markedjs/marked), so what
you see is what a GitHub README will show. Snippet URLs are absolute, built from the request host (a
reverse proxy's `X-Forwarded-Proto` is honored for the scheme).

The page is self-contained: its JavaScript and CSS are embedded in the binary and served from
`/assets/` — no external CDN — so it works air-gapped and keeps a strict `script-src 'self'`
Content-Security-Policy. See [Building from source](#building-from-source) for how the assets are
vendored.

**Which endpoints appear.** By default all endpoints are hidden. Set `defaults.hidden: false` to list
everything, then opt individual endpoints out with `hidden: true`; or keep the default and opt
specific ones in with `hidden: false`. When nothing is visible the gallery shows a short hint instead.

**Turning it off.** Set `defaults.gallery: false` to serve a minimal landing page at `/` instead of
the gallery (the badge and graph endpoints are unaffected).

## API reference

| Route              | Default response        | Variants                                                           |
| ------------------ | ----------------------- | ------------------------------------------------------------------ |
| `GET /badges/{id}` | SVG badge (`?style=…`)  | `?format=shields` → shields.io JSON · `?format=json` → kromgo JSON |
| `GET /graphs/{id}` | SVG chart (`?theme=…`)  | `?format=png` → PNG image · `?format=json` → time-series data      |
| `GET /`            | HTML gallery            | landing page when `defaults.gallery: false`                        |
| `GET /assets/…`    | Embedded gallery JS/CSS |                                                                    |

**`/badges/{id}`** (default SVG):

```html
<img src="http://localhost:8080/badges/node_cpu_usage" />
```

**`?format=shields`** — the [shields.io Endpoint Badge](https://shields.io/badges/endpoint-badge) schema:

```json
{ "schemaVersion": 1, "label": "node_cpu_usage", "message": "17.5%", "color": "green" }
```

**`?format=json`** — kromgo's native JSON (rendered string plus the raw number and labels):

```json
{
    "id": "node_cpu_usage",
    "title": "CPU",
    "value": "17.5%",
    "color": "green",
    "result": 17.5,
    "labels": {}
}
```

**`/graphs/{id}?format=json`** — the raw time series:

```json
{
    "id": "node_cpu_usage",
    "title": "CPU",
    "start": 1702578219,
    "end": 1702664619,
    "step": 60,
    "series": [{ "labels": { "instance": "node-1" }, "data": [{ "t": 1702578219, "v": 17.5 }] }]
}
```

## Ports

| Port   | Purpose                                                        |
| ------ | -------------------------------------------------------------- |
| `8080` | Main server — badge and graph endpoints                        |
| `8888` | Health server — `/healthz`, `/readyz`, `/metrics` (Prometheus) |

The health server's `/metrics` endpoint exposes Go runtime metrics plus
`kromgo_requests_total{kind, id, format}` — a counter of requests handled, broken down by endpoint
kind (`badge`/`graph`), id, and response format.

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
Set `cacheSeconds` (globally under `defaults` and/or per endpoint) and kromgo emits
`Cache-Control: public, max-age=N` on successful responses and includes `cacheSeconds` in the
shields.io endpoint JSON. Errors are always sent `no-store`. Caching is off by default
(`cacheSeconds: 0`).

```yaml
defaults:
    cacheSeconds: 300 # default for every endpoint

badges:
    - id: node_cpu_usage # changes every scrape — short TTL
      query: "..."
      cacheSeconds: 30
    - id: cluster_age # changes once a day — long TTL
      query: "..."
      cacheSeconds: 3600
```

The **other half — actually storing responses — is the edge's job**, and any cache that honors
`Cache-Control` (a CDN, Varnish, nginx `proxy_cache`) will then cache each endpoint for exactly the
TTL kromgo advertised. shields.io already respects `cacheSeconds`, so badges served through it are
cached without any proxy at all.

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

## Security

kromgo is built to face the public web. Its posture:

- **Prometheus is never exposed.** Only the endpoints you define are reachable; query parameters are
  parsed as durations/timestamps/enums and never interpolated into PromQL.
- **SVG output is safe.** Badge text and graph labels (which can derive from attacker-influenceable
  metric label values) are HTML-escaped, and badge/graph/JSON responses carry
  `Content-Security-Policy: default-src 'none'; style-src 'unsafe-inline'` and
  `X-Content-Type-Options: nosniff`, so an SVG can't execute script even when opened directly.
- **The gallery loads nothing external.** Its JS/CSS are embedded and served from `/assets/`, so the
  page ships a tightened-but-still-locked-down CSP (`script-src 'self'`, no `unsafe-inline`/`unsafe-eval`,
  no CDN). The Host header used to build snippet URLs is validated before use.
- **Bounded work.** Each Prometheus query is bounded by `QUERY_TIMEOUT` (default 30s); graph windows
  are capped by `maxDuration` and image dimensions are clamped. A 10s `ReadHeaderTimeout` guards
  against Slowloris; tune `SERVER_READ_TIMEOUT`/`SERVER_WRITE_TIMEOUT` to your proxy.
- **Minimal image.** A `scratch` image with just the static binary and a CA bundle (kromgo dials
  Prometheus over HTTPS) — no shell, package manager, or writable filesystem. It pins no user; set one
  via your Kubernetes `securityContext` or `docker run --user`. Images are cosign-signed (below).

Operational guidance:

- **Expose only the main port (`8080`).** The health port (`8888`) serves `/metrics` and probes —
  keep it on the internal network.
- **Terminate TLS and rate limit at your reverse proxy** (see [Rate limiting](#rate-limiting)).
- Treat the config as trusted (it's operator-controlled). Fonts are compiled-in (never read from
  disk), and CEL expressions run sandboxed (no env/file/network access).

## Image verification

Container images are signed with [Cosign](https://docs.sigstore.dev/cosign/overview/) keyless
signing. Verify an image before running it:

```bash
cosign verify ghcr.io/home-operations/kromgo:<tag> \
  --certificate-identity-regexp="https://github.com/home-operations/kromgo/" \
  --certificate-oidc-issuer="https://token.actions.githubusercontent.com"
```

## Building from source

The gallery's `marked.js` / `github-markdown.css` and the full Material Design Icons set are vendored
via npm (`package.json` + `package-lock.json`) and baked into the binary with `//go:embed` rather than
committed. [`cmd/genassets`](cmd/genassets/main.go) reads `node_modules` and writes the embedded files,
so a build runs `npm ci` once (network) and the resulting binary is self-contained (nothing fetched at
runtime).

```bash
mise run assets   # npm ci + go run ./cmd/genassets (re-runs only when the lockfile changes)
go build ./cmd/kromgo
```

`mise run test` / `lint` / `test-e2e` depend on `assets`, so they build it automatically; CI and the
Docker build (a dedicated `node` stage) do the same. [Renovate](https://docs.renovatebot.com) keeps
`marked`, `github-markdown-css`, and `@mdi/svg` current via PRs against `package.json`.

## Upgrading 0.11 → 0.12

0.12 splits the flat `metrics:` list into `badges:` and `graphs:` sections, with REST-style routes.
A pre-0.12 config fails fast at startup with a pointer to this guide.

| Change                                                                                                                                         | Action                                                                                                                                           |
| ---------------------------------------------------------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------ |
| **`metrics:` split into `badges:` and `graphs:`.** Instant-value endpoints go under `badges:`; time-series endpoints under `graphs:`.          | Move each metric to the section(s) it needs. A metric you served as both a badge and a chart becomes one entry in each (with the same `id`).     |
| **`name` → `id`.**                                                                                                                             | Rename the key on every endpoint.                                                                                                                |
| **Routes are namespaced.** `GET /{name}` + `?format=` → `GET /badges/{id}` and `GET /graphs/{id}`.                                             | Update embed URLs and shields.io endpoint URLs.                                                                                                  |
| **Badge default is now the SVG image.** `?format=badge` → default; `?format=json` (shields schema) → `?format=shields`; `?format=raw` removed. | Embed `/badges/{id}` directly; point shields.io at `?format=shields`. `?format=json` now returns kromgo's native JSON (value + result + labels). |
| **Graph formats.** `?format=chart` → `/graphs/{id}` (SVG default); `?format=history` → `/graphs/{id}?format=json`.                             | Switch to the `/graphs/` routes.                                                                                                                 |
| **`defaults.timeseries` removed.** The `enabled` gate is gone — defining a `graphs:` entry _is_ the opt-in.                                    | Drop `timeseries.enabled`; move `maxDuration` to `defaults.graph.maxDuration` or per-graph `maxDuration`.                                        |
| **Global `badge:` (font/size) → `defaults.badge`.** Badge `style` is now a config field too.                                                   | Move `badge.font`/`badge.size` under `defaults.badge`.                                                                                           |

Release tags drop the `v` prefix (e.g. `0.12.0`, not `v0.12.0`); pin image tags accordingly.

## Upgrading from kashalls/kromgo

This fork began as [kashalls/kromgo](https://github.com/kashalls/kromgo). Beyond the schema changes
above, note: the image moved to `ghcr.io/home-operations/kromgo`; the badge font is no longer bundled
(an embedded font is used, with `defaults.badge.font` to override); `LOG_FORMAT=test` was corrected
to `LOG_FORMAT=text`; built-in rate limiting was removed (see [Rate limiting](#rate-limiting)); and a
missing `PROMETHEUS_URL` now fails fast instead of starting degraded.

## Community

Thanks to everyone in the [Home Operations](https://discord.gg/home-operations) Discord community.
This project began as [kashalls/kromgo](https://github.com/kashalls/kromgo).
