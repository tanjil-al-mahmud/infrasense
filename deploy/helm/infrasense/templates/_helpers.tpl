{{/*
Expand the name of the chart.
*/}}
{{- define "infrasense.name" -}}
{{- .Chart.Name | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Common labels
*/}}
{{- define "infrasense.labels" -}}
helm.sh/chart: {{ .Chart.Name }}-{{ .Chart.Version }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
app.kubernetes.io/instance: {{ .Release.Name }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}

{{/*
Resolve image with optional global registry prefix
*/}}
{{- define "infrasense.image" -}}
{{- $img := . -}}
{{- if $img.registry -}}
{{ $img.registry }}/{{ $img.name }}:{{ $img.tag }}
{{- else -}}
{{ $img.name }}:{{ $img.tag }}
{{- end -}}
{{- end }}

{{/*
Namespace helper
*/}}
{{- define "infrasense.namespace" -}}
{{ .Values.global.namespace | default .Release.Namespace }}
{{- end }}
