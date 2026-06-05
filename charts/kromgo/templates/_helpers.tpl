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
Container image reference (tag defaults to the chart appVersion).
*/}}
{{- define "kromgo.image" -}}
{{- printf "%s:%s" .Values.image.repository (.Values.image.tag | default .Chart.AppVersion) }}
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
