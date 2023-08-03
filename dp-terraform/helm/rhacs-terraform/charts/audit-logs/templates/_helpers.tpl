{{/*
Namespace for the audit-logs stack.
*/}}
{{- define "aggregator.namespace" }}
{{- printf "%s-%s" .Release.Namespace "audit-logs" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Full name for audit-log aggregator.
*/}}
{{- define "aggregator.fullname" -}}
{{- printf "%s-%s" .Chart.Name "aggregator" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Labels for to match related Kubernetes resources (i.e. Service->Pod)
*/}}
{{- define "aggregator.selectorLabels" -}}
app.kubernetes.io/name: {{ include "aggregator.fullname" . }}
{{- end }}
