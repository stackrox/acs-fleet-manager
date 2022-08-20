package defaults

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

var (
	// ScannerAnalyzerRequestMemory ...
	ScannerAnalyzerRequestMemory = resource.MustParse("100Mi")
	// ScannerAnalyzerRequestCPU ...
	ScannerAnalyzerRequestCPU = resource.MustParse("250m")
	// ScannerAnalyzerLimitMemory ...
	ScannerAnalyzerLimitMemory = resource.MustParse("2500Mi")
	// ScannerAnalyzerLimitCPU ...
	ScannerAnalyzerLimitCPU = resource.MustParse("2000m")
	// ScannerAnalyzerResources ...
	ScannerAnalyzerResources = corev1.ResourceRequirements{
		Requests: map[corev1.ResourceName]resource.Quantity{
			corev1.ResourceCPU:    ScannerAnalyzerRequestCPU,
			corev1.ResourceMemory: ScannerAnalyzerRequestMemory,
		},
		Limits: map[corev1.ResourceName]resource.Quantity{
			corev1.ResourceCPU:    ScannerAnalyzerLimitCPU,
			corev1.ResourceMemory: ScannerAnalyzerLimitMemory,
		},
	}

	// ScannerAnalyzerAutoScaling ...
	ScannerAnalyzerAutoScaling = "Enabled"
	// ScannerAnalyzerScalingReplicas ...
	ScannerAnalyzerScalingReplicas int32 = 1
	// ScannerAnalyzerScalingMinReplicas ...
	ScannerAnalyzerScalingMinReplicas int32 = 1
	// ScannerAnalyzerScalingMaxReplicas ...
	ScannerAnalyzerScalingMaxReplicas int32 = 3

	// ScannerDbRequestMemory ...
	ScannerDbRequestMemory = resource.MustParse("100Mi")
	// ScannerDbRequestCPU ...
	ScannerDbRequestCPU = resource.MustParse("250m")
	// ScannerDbLimitMemory ...
	ScannerDbLimitMemory = resource.MustParse("2500Mi")
	// ScannerDbLimitCPU ...
	ScannerDbLimitCPU = resource.MustParse("2000m")
	// ScannerDbResources ...
	ScannerDbResources = corev1.ResourceRequirements{
		Requests: map[corev1.ResourceName]resource.Quantity{
			corev1.ResourceCPU:    ScannerDbRequestCPU,
			corev1.ResourceMemory: ScannerDbRequestMemory,
		},
		Limits: map[corev1.ResourceName]resource.Quantity{
			corev1.ResourceCPU:    ScannerDbLimitCPU,
			corev1.ResourceMemory: ScannerDbLimitMemory,
		},
	}
)
