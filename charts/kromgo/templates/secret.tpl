{{- if and (not .Values.secret.existingSecret) .Values.secret.prometheusUrl -}}
apiVersion: v1
kind: Secret
metadata:
  name: {{ include "kromgo.fullname" . }}
  namespace: {{ .Release.Namespace }}
  labels:
    {{- include "kromgo.labels" . | nindent 4 }}
type: Opaque
stringData:
  PROMETHEUS_URL: {{ .Values.secret.prometheusUrl | quote }}
{{- end }}
