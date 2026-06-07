{{/*
Expand the name of the chart.
*/}}
{{- define "kromgo.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create a default fully qualified app name (truncated to the 63-char DNS limit).
*/}}
{{- define "kromgo.fullname" -}}
{{- if .Values.fullnameOverride }}
{{- .Values.fullnameOverride | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- $name := default .Chart.Name .Values.nameOverride }}
{{- if contains $name .Release.Name }}
{{- .Release.Name | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- printf "%s-%s" .Release.Name $name | trunc 63 | trimSuffix "-" }}
{{- end }}
{{- end }}
{{- end }}

{{/*
Chart name and version as used by the chart label.
*/}}
{{- define "kromgo.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Common labels
*/}}
{{- define "kromgo.labels" -}}
helm.sh/chart: {{ include "kromgo.chart" . }}
{{ include "kromgo.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

{{/*
Selector labels
*/}}
{{- define "kromgo.selectorLabels" -}}
app.kubernetes.io/name: {{ include "kromgo.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{/*
Service account name to use.
*/}}
{{- define "kromgo.serviceAccountName" -}}
{{- if .Values.serviceAccount.create }}
{{- default (include "kromgo.fullname" .) .Values.serviceAccount.name }}
{{- else }}
{{- default "default" .Values.serviceAccount.name }}
{{- end }}
{{- end }}

{{/*
Container image reference. A digest pins immutably and wins when set (the release
pipeline fills it with the published image's digest); otherwise it's
repository:tag, with tag defaulting to the chart appVersion.
*/}}
{{- define "kromgo.image" -}}
{{- if .Values.image.digest -}}
{{- printf "%s@%s" .Values.image.repository .Values.image.digest -}}
{{- else -}}
{{- printf "%s:%s" .Values.image.repository (.Values.image.tag | default .Chart.AppVersion) -}}
{{- end -}}
{{- end }}

{{/*
Image for the `helm test` connection pod (kromgo's own image is built FROM
scratch, so the test uses a small image with a shell). The tag is pinned as
`version@sha256:digest`, so Renovate updates the version and digest together.
*/}}
{{- define "kromgo.testImage" -}}
{{- $img := .Values.tests.image -}}
{{- printf "%s:%s" $img.repository $img.tag -}}
{{- end }}

{{/*
Name of the ConfigMap holding the kromgo config file (existingConfigMap wins;
otherwise the chart renders one).
*/}}
{{- define "kromgo.configMapName" -}}
{{- default (include "kromgo.fullname" .) .Values.existingConfigMap -}}
{{- end }}

{{/*
Name of the Secret holding PROMETHEUS_URL, or "" if none: an existing Secret
wins; otherwise a chart-managed Secret is used only when the inline value is set.
*/}}
{{- define "kromgo.secretName" -}}
{{- if .Values.secret.existingSecret -}}
{{- .Values.secret.existingSecret -}}
{{- else if .Values.secret.prometheusUrl -}}
{{- include "kromgo.fullname" . -}}
{{- end -}}
{{- end }}
