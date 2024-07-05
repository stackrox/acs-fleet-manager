/*
 * Red Hat Advanced Cluster Security Service Fleet Manager
 *
 * Red Hat Advanced Cluster Security (RHACS) Service Fleet Manager APIs that are used by internal services e.g fleetshard operators.
 *
 * API version: 1.4.0
 * Generated by: OpenAPI Generator (https://openapi-generator.tech)
 */

// Code generated by OpenAPI Generator (https://openapi-generator.tech). DO NOT EDIT.
package private

// VpaRecommenderConfig struct for VpaRecommenderConfig
type VpaRecommenderConfig struct {
	Name                                     string                 `json:"name"`
	Image                                    string                 `json:"image,omitempty"`
	ImagePullSecrets                         []LocalObjectReference `json:"imagePullSecrets,omitempty"`
	Resources                                ResourceRequirements   `json:"resources,omitempty"`
	RecommendationMarginFraction             float32                `json:"recommendationMarginFraction,omitempty"`
	PodRecommendationMinCpuMillicores        float32                `json:"podRecommendationMinCpuMillicores,omitempty"`
	PodRecommendationMinMemoryMb             float32                `json:"podRecommendationMinMemoryMb,omitempty"`
	TargetCpuPercentile                      float32                `json:"targetCpuPercentile,omitempty"`
	RecommendationLowerBoundCpuPercentile    float32                `json:"recommendationLowerBoundCpuPercentile,omitempty"`
	RecommendationUpperBoundCpuPercentile    float32                `json:"recommendationUpperBoundCpuPercentile,omitempty"`
	TargetMemoryPercentile                   float32                `json:"targetMemoryPercentile,omitempty"`
	RecommendationLowerBoundMemoryPercentile float32                `json:"recommendationLowerBoundMemoryPercentile,omitempty"`
	RecommendationUpperBoundMemoryPercentile float32                `json:"recommendationUpperBoundMemoryPercentile,omitempty"`
	CheckpointsTimeout                       string                 `json:"checkpointsTimeout,omitempty"`
	MinCheckpoints                           int32                  `json:"minCheckpoints,omitempty"`
	MemorySaver                              bool                   `json:"memorySaver,omitempty"`
	RecommenderInterval                      string                 `json:"recommenderInterval,omitempty"`
	CheckpointsGcInterval                    string                 `json:"checkpointsGcInterval,omitempty"`
	PrometheusAddress                        string                 `json:"prometheusAddress,omitempty"`
	PrometheusCadvisorJobName                string                 `json:"prometheusCadvisorJobName,omitempty"`
	Address                                  string                 `json:"address,omitempty"`
	Kubeconfig                               string                 `json:"kubeconfig,omitempty"`
	KubeApiQps                               float32                `json:"kubeApiQps,omitempty"`
	KubeApiBurst                             int32                  `json:"kubeApiBurst,omitempty"`
	Storage                                  string                 `json:"storage,omitempty"`
	HistoryLength                            string                 `json:"historyLength,omitempty"`
	HistoryResolution                        string                 `json:"historyResolution,omitempty"`
	PrometheusQueryTimeout                   string                 `json:"prometheusQueryTimeout,omitempty"`
	PodLabelPrefix                           string                 `json:"podLabelPrefix,omitempty"`
	MetricForPodLabels                       string                 `json:"metricForPodLabels,omitempty"`
	PodNamespaceLabel                        string                 `json:"podNamespaceLabel,omitempty"`
	PodNameLabel                             string                 `json:"podNameLabel,omitempty"`
	ContainerNamespaceLabel                  string                 `json:"containerNamespaceLabel,omitempty"`
	ContainerPodNameLabel                    string                 `json:"containerPodNameLabel,omitempty"`
	ContainerNameLabel                       string                 `json:"containerNameLabel,omitempty"`
	VpaObjectNamespace                       string                 `json:"vpaObjectNamespace,omitempty"`
	MemoryAggregationInterval                string                 `json:"memoryAggregationInterval,omitempty"`
	MemoryAggregationIntervalCount           int32                  `json:"memoryAggregationIntervalCount,omitempty"`
	MemoryHistogramDecayHalfLife             string                 `json:"memoryHistogramDecayHalfLife,omitempty"`
	CpuHistogramDecayHalfLife                string                 `json:"cpuHistogramDecayHalfLife,omitempty"`
	CpuIntegerPostProcessorEnabled           bool                   `json:"cpuIntegerPostProcessorEnabled,omitempty"`
	UseExternalMetrics                       bool                   `json:"useExternalMetrics,omitempty"`
	ExternalMetricsCpuMetric                 string                 `json:"externalMetricsCpuMetric,omitempty"`
	ExternalMetricsMemoryMetric              string                 `json:"externalMetricsMemoryMetric,omitempty"`
	OomBumpUpRatio                           float32                `json:"oomBumpUpRatio,omitempty"`
	OomMinBumpUpBytes                        float32                `json:"oomMinBumpUpBytes,omitempty"`
	Tolerations                              []Toleration           `json:"tolerations,omitempty"`
	NodeSelector                             map[string]string      `json:"nodeSelector,omitempty"`
	UseProxy                                 bool                   `json:"useProxy,omitempty"`
	ProxyImage                               string                 `json:"proxyImage,omitempty"`
	LogLevel                                 float32                `json:"logLevel,omitempty"`
}
