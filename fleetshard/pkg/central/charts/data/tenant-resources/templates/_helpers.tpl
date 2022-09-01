{{- define "labels" -}}
{{- $labels := tpl (.Files.Get "config/default-labels.yaml.tpl") . | fromYaml -}}
{{- $labels = merge (deepCopy .Values.labels) $labels -}}
{{- $labels | toYaml | nindent 0 }}
{{- end -}}

{{- define "annotations" -}}
{{- $annotations := tpl (.Files.Get "config/default-annotations.yaml.tpl") . | fromYaml -}}
{{- $annotations = merge (deepCopy .Values.annotations) $annotations -}}
{{- $annotations | toYaml | nindent 0 }}
{{- end -}}
