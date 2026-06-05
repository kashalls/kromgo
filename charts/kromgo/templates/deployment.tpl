apiVersion: apps/v1
kind: Deployment
metadata:
  name: {{ include "kromgo.fullname" . }}
  namespace: {{ .Release.Namespace }}
  labels:
    {{- include "kromgo.labels" . | nindent 4 }}
spec:
  replicas: {{ .Values.replicaCount }}
  selector:
    matchLabels:
      {{- include "kromgo.selectorLabels" . | nindent 6 }}
  template:
    metadata:
      labels:
        {{- include "kromgo.labels" . | nindent 8 }}
        {{- with .Values.podLabels }}
        {{- toYaml . | nindent 8 }}
        {{- end }}
      annotations:
        {{- if not .Values.existingConfigMap }}
        # Roll the pods when the rendered config changes.
        checksum/config: {{ include (print $.Template.BasePath "/configmap.tpl") . | sha256sum }}
        {{- end }}
        {{- with .Values.podAnnotations }}
        {{- toYaml . | nindent 8 }}
        {{- end }}
    spec:
      {{- with .Values.imagePullSecrets }}
      imagePullSecrets:
        {{- toYaml . | nindent 8 }}
      {{- end }}
      serviceAccountName: {{ include "kromgo.serviceAccountName" . }}
      automountServiceAccountToken: {{ .Values.serviceAccount.automount }}
      securityContext:
        {{- toYaml .Values.podSecurityContext | nindent 8 }}
      containers:
        - name: kromgo
          image: {{ include "kromgo.image" . | quote }}
          imagePullPolicy: {{ .Values.image.pullPolicy }}
          securityContext:
            {{- toYaml .Values.securityContext | nindent 12 }}
          env:
            - name: QUERY_TIMEOUT
              value: {{ .Values.server.queryTimeout | quote }}
            {{- with .Values.server.readTimeout }}
            - name: SERVER_READ_TIMEOUT
              value: {{ . | quote }}
            {{- end }}
            {{- with .Values.server.writeTimeout }}
            - name: SERVER_WRITE_TIMEOUT
              value: {{ . | quote }}
            {{- end }}
            {{- if .Values.server.logging }}
            - name: SERVER_LOGGING
              value: "true"
            {{- end }}
            {{- with .Values.server.extraEnv }}
            {{- toYaml . | nindent 12 }}
            {{- end }}
          {{- with (include "kromgo.secretName" .) }}
          # PROMETHEUS_URL — overrides config.prometheus when present.
          envFrom:
            - secretRef:
                name: {{ . }}
          {{- end }}
          ports:
            - name: http
              containerPort: 8080
              protocol: TCP
            - name: health
              containerPort: 8888
              protocol: TCP
          livenessProbe:
            {{- toYaml .Values.livenessProbe | nindent 12 }}
          readinessProbe:
            {{- toYaml .Values.readinessProbe | nindent 12 }}
          {{- with .Values.resources }}
          resources:
            {{- toYaml . | nindent 12 }}
          {{- end }}
          volumeMounts:
            - name: config
              mountPath: /config
              readOnly: true
            {{- with .Values.volumeMounts }}
            {{- toYaml . | nindent 12 }}
            {{- end }}
      volumes:
        - name: config
          configMap:
            name: {{ include "kromgo.configMapName" . }}
        {{- with .Values.volumes }}
        {{- toYaml . | nindent 8 }}
        {{- end }}
      {{- with .Values.nodeSelector }}
      nodeSelector:
        {{- toYaml . | nindent 8 }}
      {{- end }}
      {{- with .Values.affinity }}
      affinity:
        {{- toYaml . | nindent 8 }}
      {{- end }}
      {{- with .Values.tolerations }}
      tolerations:
        {{- toYaml . | nindent 8 }}
      {{- end }}
