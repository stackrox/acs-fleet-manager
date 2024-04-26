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

{{- define "localNetworkCidrRanges" -}}
{{- tpl (.Files.Get "config/local-network-cidr-ranges.yaml.tpl") . -}}
{{- end -}}

{{- define "localNetworkCidrRangesIPv6" -}}
{{- tpl (.Files.Get "config/local-network-cidr-ranges-ipv6.yaml.tpl") . -}}
{{- end -}}
