{{- define "msa.fullname" -}}
{{- if .Values.fullnameOverride }}{{ .Values.fullnameOverride }}{{- else }}{{- if .Values.nameOverride }}{{ .Values.nameOverride }}{{- else }}{{ .Chart.Name }}{{- end }}{{- end }}
{{- end }}

{{- define "msa.labels" -}}
app.kubernetes.io/name: {{ include "msa.fullname" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
app.kubernetes.io/version: {{ .Chart.AppVersion }}
{{- end }}
