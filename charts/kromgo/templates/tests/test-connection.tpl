apiVersion: v1
kind: Pod
metadata:
  name: {{ include "kromgo.fullname" . }}-test-connection
  namespace: {{ .Release.Namespace }}
  labels:
    {{- include "kromgo.labels" . | nindent 4 }}
  annotations:
    helm.sh/hook: test
    # before-hook-creation only (no hook-succeeded): `helm test` then never deletes
    # the pod itself, so it can't block on Helm 4's wait-for-delete (kstatus) after a
    # green run — which otherwise stalled `helm test` ~5m. The pod is recreated on
    # the next run, and a failed run's pod stays for `helm test --logs` / kubectl.
    helm.sh/hook-delete-policy: before-hook-creation
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
