{{- define "vertical-pod-autoscaler.namespace" }}
{{- printf "%s-%s" .Release.Namespace "vertical-pod-autoscaler" }}
{{- end }}
