apiVersion: v1
kind: Pod
metadata:
  name: {{ include "kromgo.fullname" . }}-test-connection
  namespace: {{ .Release.Namespace }}
  labels:
    {{- include "kromgo.labels" . | nindent 4 }}
  annotations:
    helm.sh/hook: test
    # Recreate on each run; keep the pod on failure so `helm test --logs` (and a
    # manual `kubectl logs`) can show what happened.
    helm.sh/hook-delete-policy: before-hook-creation,hook-succeeded
spec:
  restartPolicy: Never
  securityContext:
    runAsNonRoot: true
    runAsUser: 65532
    runAsGroup: 65532
    seccompProfile:
      type: RuntimeDefault
  containers:
    - name: connection
      image: {{ include "kromgo.testImage" . | quote }}
      imagePullPolicy: {{ .Values.tests.image.pullPolicy }}
      securityContext:
        allowPrivilegeEscalation: false
        readOnlyRootFilesystem: true
        capabilities:
          drop:
            - ALL
      # /readyz on the health/metrics port returns a static 200 (no Prometheus
      # dependency), so this checks purely that the Service routes to a running,
      # listening pod. wget writes to stdout (-O-) so the rootfs stays read-only; a
      # non-2xx or refused connection exits non-zero and fails `helm test`.
      command:
        - wget
      args:
        - -q
        - -O-
        - http://{{ include "kromgo.fullname" . }}:{{ .Values.service.metricsPort }}/readyz
