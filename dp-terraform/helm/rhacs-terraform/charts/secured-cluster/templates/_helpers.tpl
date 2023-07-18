{{- define "secured-cluster.namespace" }}
{{- printf "%s-%s" .Release.Namespace "secured-cluster" }}
{{- end }}
