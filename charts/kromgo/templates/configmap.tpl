{{- if not .Values.existingConfigMap -}}
apiVersion: v1
kind: ConfigMap
metadata:
  name: {{ include "kromgo.fullname" . }}
  namespace: {{ .Release.Namespace }}
  labels:
    {{- include "kromgo.labels" . | nindent 4 }}
data:
  # Rendered through `tpl`, so config values may reference release metadata or other
  # values. PromQL/CEL braces pass through; escaping note is in values.yaml.
  config.yaml: |
    {{- tpl (toYaml .Values.config) $ | nindent 4 }}
{{- end }}
