package defaults

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

var (
	// CentralRequestMemory ...
	CentralRequestMemory = resource.MustParse("250Mi")
	// CentralRequestCPU ...
	CentralRequestCPU = resource.MustParse("250m")
	// CentralLimitMemory ...
	CentralLimitMemory = resource.MustParse("4Gi")
	// CentralLimitCPU ...
	CentralLimitCPU = resource.MustParse("1000m")
	// CentralResources ...
	CentralResources = corev1.ResourceRequirements{
		Requests: map[corev1.ResourceName]resource.Quantity{
			corev1.ResourceCPU:    CentralRequestCPU,
			corev1.ResourceMemory: CentralRequestMemory,
		},
		Limits: map[corev1.ResourceName]resource.Quantity{
			corev1.ResourceCPU:    CentralLimitCPU,
			corev1.ResourceMemory: CentralLimitMemory,
		},
	}
)
