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
        {{- tpl (toYaml .) $ | nindent 8 }}
        {{- end }}
      annotations:
        {{- if not .Values.existingConfigMap }}
        # Roll the pods when the rendered config changes.
        checksum/config: {{ include (print $.Template.BasePath "/configmap.tpl") . | sha256sum }}
        {{- end }}
        {{- with .Values.podAnnotations }}
        {{- tpl (toYaml .) $ | nindent 8 }}
        {{- end }}
    spec:
      {{- with .Values.imagePullSecrets }}
      imagePullSecrets:
        {{- tpl (toYaml .) $ | nindent 8 }}
      {{- end }}
      serviceAccountName: {{ include "kromgo.serviceAccountName" . }}
      automountServiceAccountToken: {{ .Values.serviceAccount.automount }}
      securityContext:
        {{- tpl (toYaml .Values.podSecurityContext) $ | nindent 8 }}
      containers:
        - name: kromgo
          image: {{ include "kromgo.image" . | quote }}
          imagePullPolicy: {{ .Values.image.pullPolicy }}
          securityContext:
            {{- tpl (toYaml .Values.securityContext) $ | nindent 12 }}
          env:
            - name: QUERY_TIMEOUT
              value: {{ tpl .Values.server.queryTimeout $ | quote }}
            {{- with .Values.server.readTimeout }}
            - name: SERVER_READ_TIMEOUT
              value: {{ tpl . $ | quote }}
            {{- end }}
            {{- with .Values.server.writeTimeout }}
            - name: SERVER_WRITE_TIMEOUT
              value: {{ tpl . $ | quote }}
            {{- end }}
            {{- if .Values.server.logging }}
            - name: SERVER_LOGGING
              value: "true"
            {{- end }}
            {{- with .Values.server.extraEnv }}
            {{- tpl (toYaml .) $ | nindent 12 }}
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
            {{- tpl (toYaml .Values.livenessProbe) $ | nindent 12 }}
          readinessProbe:
            {{- tpl (toYaml .Values.readinessProbe) $ | nindent 12 }}
          {{- with .Values.resources }}
          resources:
            {{- tpl (toYaml .) $ | nindent 12 }}
          {{- end }}
          volumeMounts:
            - name: config
              mountPath: /config
              readOnly: true
            {{- with .Values.volumeMounts }}
            {{- tpl (toYaml .) $ | nindent 12 }}
            {{- end }}
      volumes:
        - name: config
          configMap:
            name: {{ include "kromgo.configMapName" . }}
        {{- with .Values.volumes }}
        {{- tpl (toYaml .) $ | nindent 8 }}
        {{- end }}
      {{- with .Values.nodeSelector }}
      nodeSelector:
        {{- tpl (toYaml .) $ | nindent 8 }}
      {{- end }}
      {{- with .Values.affinity }}
      affinity:
        {{- tpl (toYaml .) $ | nindent 8 }}
      {{- end }}
      {{- with .Values.tolerations }}
      tolerations:
        {{- tpl (toYaml .) $ | nindent 8 }}
      {{- end }}
