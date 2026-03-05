# Kromgo

A simple go project that allows you to expose prometheus metrics "safely" to a public source. Uses the official prometheus go api client. Better than exposing a grafana image rendering instance to the WWW.

It allows you to define your own metric names and your own prometheus queries as long as they return a single value at the end. There is config support to allow you to format the response with strings before and after the value.

You can use [shields.io](https://shields.io) and use either the [Dynamic JSON Badge](https://shields.io/badges/dynamic-json-badge) or the [Endpoint Badge](https://shields.io/badges/endpoint-badge) and add dynamic coloring with ranges you set.

[Config Example](./config.yaml.example)
[Configuration Structure](./cmd/kromgo/init/configuration/configuration.go)

- Reads configuration file from `/kromgo/config.yaml`
- Requires `PROMETHEUS_URL` be set in ENV.
- Optional `SERVER_PORT` to change server port.

## Value Templates

Metrics support a `valueTemplate` field that applies a [Go template](https://pkg.go.dev/text/template) to the raw Prometheus value before `prefix` and `suffix` are added.

### Built-in template functions

| Function | Input | Output | Description |
|---|---|---|---|
| `simplifyDays` | `"1159"` | `3y64d` | Converts a day count to years and days |
| `humanBytes` | `"1572864"` | `1.5MB` | Converts a byte count to a human-readable size |
| `humanDuration` | `"9000"` | `2h30m` | Converts seconds to a compact duration string |
| `toUpper` | `"v1.31.0"` | `V1.31.0` | Converts the string to uppercase |
| `toLower` | `"HEALTHY"` | `healthy` | Converts the string to lowercase |
| `trim` | `" ok "` | `ok` | Strips leading and trailing whitespace |

### Inline template

Set `valueTemplate` directly on a metric to a Go template string:

```yaml
metrics:
  - name: cluster_age
    query: "floor((time() - k8s_cluster_created_timestamp) / 86400)"
    valueTemplate: "{{ . | simplifyDays }}"   # 1159 → 3y64d

  - name: node_memory_used
    query: "node_memory_MemTotal_bytes - node_memory_MemAvailable_bytes"
    valueTemplate: "{{ . | humanBytes }}"     # 1572864 → 1.5MB

  - name: node_uptime
    query: "time() - node_boot_time_seconds"
    valueTemplate: "{{ . | humanDuration }}"  # 9000 → 2h30m
```

### Named templates

Define reusable template snippets at the top level under `templates` and reference them by name across multiple metrics:

```yaml
templates:
  clusterAge: "{{ . | simplifyDays }}"
  uptime: "{{ . | humanDuration }}"

metrics:
  - name: cluster_age
    query: "floor((time() - k8s_cluster_created_timestamp) / 86400)"
    valueTemplate: "clusterAge"   # resolved from templates map

  - name: node_uptime
    query: "time() - node_boot_time_seconds"
    valueTemplate: "uptime"
```

If `valueTemplate` matches a key in `templates`, that snippet is used; otherwise the value is treated as a literal Go template string. Named templates and inline templates can be mixed freely.

## Performance

Queries take around 5ms ~ 75ms to complete depending on how many breaks my prometheus server takes. This was running on my [home-cluster](https://github.com/kashalls/home-cluster) and runs 3 instances, so depending on the query YMMV.

## Example Request

### Endpoint Response

This format is provided to support Shield.io's [Endpoint Badge](https://shields.io/badges/endpoint-badge) endpoint.

`HTTP GET localhost:8080/node_cpu_usage`

```json
{
    "color": "green",
    "label": "node_cpu_usage",
    "message": "17.5",
    "schemaVersion": 1
}
```

### Raw Response

`HTTP GET localhost:8080/node_cpu_usage?format=raw`

```json
[
    {
        "metric": {},
        "value": [
            1702664619.78,
            "17.5"
        ]
    }
]
```

### Badge Response

Like the `endpoint` format but serves an svg badge with `label` and `message`

`HTTP GET localhost:8080/node_cpu_usage?format=badge`

```
content-type: image/svg+xml
<svg xmlns="http://www.w3.org/2000/svg" ...
```

### 🤝 Gratitude and Thanks

Thanks to all of the people at the [Home Operations](https://discord.gg/home-operations) Discord community. Be sure to check it out, its a blast!
