{{- if .Values.ingress.enabled -}}
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: {{ include "kromgo.fullname" . }}
  namespace: {{ .Release.Namespace }}
  labels:
    {{- include "kromgo.labels" . | nindent 4 }}
  {{- with .Values.ingress.annotations }}
  annotations:
    {{- tpl (toYaml .) $ | nindent 4 }}
  {{- end }}
spec:
  {{- with .Values.ingress.className }}
  ingressClassName: {{ tpl . $ }}
  {{- end }}
  {{- with .Values.ingress.tls }}
  tls:
    {{- tpl (toYaml .) $ | nindent 4 }}
  {{- end }}
  rules:
    {{- range .Values.ingress.hosts }}
    - host: {{ tpl .host $ | quote }}
      http:
        paths:
          {{- range .paths }}
          - path: {{ .path }}
            pathType: {{ .pathType | default "Prefix" }}
            backend:
              service:
                name: {{ include "kromgo.fullname" $ }}
                port:
                  name: http
          {{- end }}
    {{- end }}
{{- end }}
