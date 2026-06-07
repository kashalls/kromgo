# kromgo

![Version: 0.0.0](https://img.shields.io/badge/Version-0.0.0-informational?style=flat-square) ![Type: application](https://img.shields.io/badge/Type-application-informational?style=flat-square) ![AppVersion: 0.0.0](https://img.shields.io/badge/AppVersion-0.0.0-informational?style=flat-square)

Safely expose individual Prometheus metric values as SVG badges and graphs

**Homepage:** <https://github.com/home-operations/kromgo>

## Usage

kromgo ships as an OCI Helm chart. Point it at your Prometheus and define the
badge/graph endpoints under `config`:

```sh
helm install kromgo oci://ghcr.io/home-operations/charts/kromgo \
  --set config.prometheus=http://prometheus-operated.monitoring.svc.cluster.local:9090
```

Endpoints are declared under `config.badges` / `config.graphs` (see
[`values.yaml`](values.yaml) for the shape), or point `config.existingConfigMap`
at a ConfigMap you manage elsewhere. If the Prometheus URL embeds credentials,
set `secret.prometheusUrl` (or `secret.existingSecret`) so it stays out of the
ConfigMap. Expose the gallery with either `ingress` or a Gateway API `httpRoute`.

## Maintainers

| Name | Email | Url |
| ---- | ------ | --- |
| home-operations | <contact@home-operations.com> |  |

## Source Code

* <https://github.com/home-operations/kromgo>

## Requirements

Kubernetes: `>=1.25.0-0`

## Values

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| affinity | object | `{}` | Affinity rules for pod scheduling. |
| config.badges | list | `[]` | Instant-value endpoints served at /badges/{id}. |
| config.cache.enabled | bool | `true` | Send Cache-Control headers on badge/graph responses. |
| config.cache.maxAge | int | `300` | max-age + s-maxage in seconds (ignored when cache is disabled). |
| config.defaults | object | `{}` | Defaults applied to every endpoint, each overridable per badge/graph. |
| config.gallery.enabled | bool | `true` | Preview every endpoint at "/" with copy-paste Markdown; false serves a minimal landing page. |
| config.graphs | list | `[]` | Time-series endpoints served at /graphs/{id}. |
| config.prometheus | string | `"http://prometheus-operated.monitoring.svc.cluster.local:9090"` | Prometheus base URL kromgo queries; use `secret.prometheusUrl` instead when it embeds credentials. |
| existingConfigMap | string | `""` | Mount an existing ConfigMap (with a `config.yaml` key) instead of the inline `config`; takes precedence over it. |
| fullnameOverride | string | `""` | Override the full release name. |
| httpRoute.additionalRules | list | `[]` | Custom rules prepended before the default rule (templated). |
| httpRoute.annotations | object | `{}` | HTTPRoute annotations. |
| httpRoute.apiVersion | string | `""` | HTTPRoute apiVersion; empty defaults to gateway.networking.k8s.io/v1. |
| httpRoute.enabled | bool | `false` | Expose the UI via a Gateway API HTTPRoute (alternative to ingress). |
| httpRoute.filters | list | `[]` | Filters applied to the default rule. |
| httpRoute.hostnames | list | `[]` | Hostnames matched against the Host header (templated). |
| httpRoute.httpsRedirect | bool | `false` | Redirect HTTP→HTTPS (301) instead of routing to the backend (needs HTTP+HTTPS listeners). |
| httpRoute.kind | string | `""` | HTTPRoute kind; empty defaults to HTTPRoute. |
| httpRoute.labels | object | `{}` | HTTPRoute labels. |
| httpRoute.matches | list | `[{"path":{"type":"PathPrefix","value":"/"}}]` | Match conditions for the default rule. |
| httpRoute.parentRefs | list | `[]` | Gateways (and listeners) this route attaches to. |
| image.digest | string | `""` | Pin the image by digest (sha256:…); when set, overrides the tag. |
| image.pullPolicy | string | `"IfNotPresent"` | Image pull policy. |
| image.repository | string | `"ghcr.io/home-operations/kromgo"` | Image repository. |
| image.tag | string | `""` | Overrides the image tag; defaults to the chart appVersion. |
| imagePullSecrets | list | `[]` | Image pull secrets for private registries. |
| ingress.annotations | object | `{}` | Ingress annotations. |
| ingress.className | string | `""` | IngressClass name. |
| ingress.enabled | bool | `false` | Expose the UI via an Ingress. |
| ingress.hosts | list | `[{"host":"kromgo.example.com","paths":[{"path":"/","pathType":"Prefix"}]}]` | Ingress hosts and their paths. |
| ingress.tls | list | `[]` | Ingress TLS configuration. |
| livenessProbe | object | `{"httpGet":{"path":"/healthz","port":"health"},"initialDelaySeconds":10,"periodSeconds":20}` | Liveness probe. |
| monitoring.serviceMonitor.annotations | object | `{}` | ServiceMonitor annotations. |
| monitoring.serviceMonitor.enabled | bool | `false` | Create a Prometheus Operator ServiceMonitor (requires its CRDs). |
| monitoring.serviceMonitor.interval | string | `"30s"` | Scrape interval. |
| monitoring.serviceMonitor.labels | object | `{}` | ServiceMonitor labels. |
| monitoring.serviceMonitor.metricRelabelings | list | `[]` | Prometheus metric relabelings. |
| monitoring.serviceMonitor.path | string | `"/metrics"` | Metrics path. |
| monitoring.serviceMonitor.relabelings | list | `[]` | Prometheus relabelings. |
| monitoring.serviceMonitor.scrapeTimeout | string | `"10s"` | Scrape timeout. |
| nameOverride | string | `""` | Override the chart name used in resource names. |
| nodeSelector | object | `{}` | Node selector for pod scheduling. |
| podAnnotations | object | `{}` | Annotations added to the pod. |
| podLabels | object | `{}` | Labels added to the pod. |
| podSecurityContext | object | `{"fsGroup":65532,"runAsGroup":65532,"runAsNonRoot":true,"runAsUser":65532,"seccompProfile":{"type":"RuntimeDefault"}}` | Pod-level securityContext (runs as non-root uid/gid 65532 with the RuntimeDefault seccomp profile). |
| readinessProbe | object | `{"httpGet":{"path":"/readyz","port":"health"},"initialDelaySeconds":5,"periodSeconds":10}` | Readiness probe. |
| replicaCount | int | `1` | Number of kromgo replicas (it queries Prometheus per request and is stateless behind the Service). |
| resources | object | `{}` | Pod resource requests/limits. |
| secret.existingSecret | string | `""` | Existing Secret with a PROMETHEUS_URL key; takes precedence over the inline value below. |
| secret.prometheusUrl | string | `""` | PROMETHEUS_URL — sensitive Prometheus URL; overrides config.prometheus when set. |
| securityContext | object | `{"allowPrivilegeEscalation":false,"capabilities":{"drop":["ALL"]},"readOnlyRootFilesystem":true}` | Container securityContext (no privilege escalation, read-only root filesystem, drops ALL capabilities). |
| server.extraEnv | list | `[]` | Extra raw env vars merged into the container (advanced), e.g. GOMEMLIMIT. |
| server.logging | bool | `false` | SERVER_LOGGING — per-request access logging. |
| server.queryTimeout | string | `"30s"` | QUERY_TIMEOUT — bounds each outbound Prometheus query (Go duration). |
| server.readTimeout | string | `""` | SERVER_READ_TIMEOUT (Go duration); empty = Go's default (no timeout). |
| server.writeTimeout | string | `""` | SERVER_WRITE_TIMEOUT (Go duration); empty = Go's default (no timeout). |
| service.metricsPort | int | `8888` | Health + /metrics port. |
| service.port | int | `8080` | Badge / graph / gallery port. |
| service.type | string | `"ClusterIP"` | Service type. |
| serviceAccount.annotations | object | `{}` | Annotations for the ServiceAccount. |
| serviceAccount.automount | bool | `false` | Automount the ServiceAccount API token (off by default: kromgo talks to Prometheus, not the cluster API). |
| serviceAccount.create | bool | `true` | Create a ServiceAccount. |
| serviceAccount.name | string | `""` | ServiceAccount name; generated from the release name if empty. |
| tests.image.digest | string | `"sha256:9532d8c39891ca2ecde4d30d7710e01fb739c87a8b9299685c63704296b16028"` | `helm test` image digest (sha256:…); pins immutably and wins over the tag when set. |
| tests.image.pullPolicy | string | `"IfNotPresent"` | `helm test` image pull policy. |
| tests.image.repository | string | `"mirror.gcr.io/busybox"` | `helm test` pod image; needs a shell with wget (kromgo's own image is from scratch). |
| tests.image.tag | string | `"1.37.0"` | `helm test` image tag. |
| tolerations | list | `[]` | Tolerations for pod scheduling. |
| volumeMounts | list | `[]` | Additional volume mounts on the container. |
| volumes | list | `[]` | Additional volumes on the Deployment. |

---

_This README is generated by [helm-docs](https://github.com/norwoodj/helm-docs) from `Chart.yaml` and `values.yaml`. Edit those (or `README.md.gotmpl`) and run `mise run generate`._
