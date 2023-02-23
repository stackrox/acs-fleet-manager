{{/*
Namespace for the observability stack.
*/}}
{{- define "cloudwatch.namespace" }}
{{- printf "%s-%s" .Release.Namespace "cloudwatch" }}
{{- end }}
