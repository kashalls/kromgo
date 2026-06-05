{{- if .Values.httpRoute.enabled -}}
{{- $route := .Values.httpRoute -}}
apiVersion: {{ $route.apiVersion | default "gateway.networking.k8s.io/v1" }}
kind: {{ $route.kind | default "HTTPRoute" }}
metadata:
  name: {{ include "kromgo.fullname" . }}
  namespace: {{ .Release.Namespace }}
  labels:
    {{- include "kromgo.labels" . | nindent 4 }}
    {{- with $route.labels }}
    {{- tpl (toYaml .) $ | nindent 4 }}
    {{- end }}
  {{- with $route.annotations }}
  annotations:
    {{- tpl (toYaml .) $ | nindent 4 }}
  {{- end }}
spec:
  {{- with $route.parentRefs }}
  parentRefs:
    {{- tpl (toYaml .) $ | nindent 4 }}
  {{- end }}
  {{- with $route.hostnames }}
  hostnames:
    {{- tpl (toYaml .) $ | nindent 4 }}
  {{- end }}
  rules:
    {{- with $route.additionalRules }}
    {{- tpl (toYaml .) $ | nindent 4 }}
    {{- end }}
    {{- if $route.httpsRedirect }}
    - filters:
        - type: RequestRedirect
          requestRedirect:
            scheme: https
            statusCode: 301
    {{- else }}
    - backendRefs:
        - name: {{ include "kromgo.fullname" . }}
          port: {{ .Values.service.port }}
      {{- with $route.filters }}
      filters:
        {{- tpl (toYaml .) $ | nindent 8 }}
      {{- end }}
      {{- with $route.matches }}
      matches:
        {{- tpl (toYaml .) $ | nindent 8 }}
      {{- end }}
    {{- end }}
{{- end }}
