{{/*
Namespace for the observability stack.
*/}}
{{- define "observability.namespace" -}}
{{- printf "%s-%s" .Release.Namespace "observability" }}
{{- end }}
