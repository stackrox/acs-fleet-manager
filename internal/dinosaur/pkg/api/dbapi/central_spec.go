package dbapi

import corev1 "k8s.io/api/core/v1"

// CentralSpec ...
type CentralSpec struct {
	Resources corev1.ResourceRequirements `json:"resources"`
}

// ScannerAnalyzerScaling ...
type ScannerAnalyzerScaling struct {
	AutoScaling string `json:"autoScaling,omitempty"`
	Replicas    int32  `json:"replicas,omitempty"`
	MinReplicas int32  `json:"minReplicas,omitempty"`
	MaxReplicas int32  `json:"maxReplicas,omitempty"`
}

// ScannerAnalyzerSpec ...
type ScannerAnalyzerSpec struct {
	Resources corev1.ResourceRequirements `json:"resources,omitempty"`
	Scaling   ScannerAnalyzerScaling      `json:"scaling,omitempty"`
}

// ScannerDbSpec ...
type ScannerDbSpec struct {
	Resources corev1.ResourceRequirements `json:"resources,omitempty"`
}

// ScannerSpec ...
type ScannerSpec struct {
	Analyzer ScannerAnalyzerSpec `json:"analyzer,omitempty"`
	Db       ScannerDbSpec       `json:"db,omitempty"`
}
