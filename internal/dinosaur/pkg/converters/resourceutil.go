package converters

import (
	"encoding/json"
	"fmt"

	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/api/private"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/api/public"
	"github.com/stackrox/rox/operator/apis/platform/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

// ConvertPrivateScalingToV1 ...
func ConvertPrivateScalingToV1(scaling *private.ManagedCentralAllOfSpecScannerAnalyzerScaling) v1alpha1.ScannerAnalyzerScaling {
	if scaling == nil {
		return v1alpha1.ScannerAnalyzerScaling{}
	}
	autoScaling := scaling.AutoScaling
	return v1alpha1.ScannerAnalyzerScaling{
		AutoScaling: (*v1alpha1.AutoScalingPolicy)(&autoScaling), // TODO(create-ticket): validate.
		Replicas:    &scaling.Replicas,
		MinReplicas: &scaling.MinReplicas,
		MaxReplicas: &scaling.MaxReplicas,
	}

}

// ConvertPublicScalingToV1 ...
func ConvertPublicScalingToV1(scaling *public.ScannerSpecAnalyzerScaling) (v1alpha1.ScannerAnalyzerScaling, error) {
	if scaling == nil {
		return v1alpha1.ScannerAnalyzerScaling{}, nil
	}
	autoScaling := scaling.AutoScaling
	return v1alpha1.ScannerAnalyzerScaling{
		AutoScaling: (*v1alpha1.AutoScalingPolicy)(&autoScaling), // TODO(create-ticket): validate.
		Replicas:    &scaling.Replicas,
		MinReplicas: &scaling.MinReplicas,
		MaxReplicas: &scaling.MaxReplicas,
	}, nil
}

func qtyAsString(qty resource.Quantity) string {
	return (&qty).String()
}

// ConvertCoreV1ResourceRequirementsToPublic ...
func ConvertCoreV1ResourceRequirementsToPublic(res *v1.ResourceRequirements) public.ResourceRequirements {
	return public.ResourceRequirements{
		Limits: public.ResourceList{
			Cpu:    qtyAsString(res.Limits[corev1.ResourceCPU]),
			Memory: qtyAsString(res.Limits[corev1.ResourceMemory]),
		},
		Requests: public.ResourceList{
			Cpu:    qtyAsString(res.Requests[corev1.ResourceCPU]),
			Memory: qtyAsString(res.Requests[corev1.ResourceMemory]),
		},
	}
}

// ConvertPublicResourceRequirementsToCoreV1 ...
func ConvertPublicResourceRequirementsToCoreV1(res *public.ResourceRequirements) (corev1.ResourceRequirements, error) {
	val, err := json.Marshal(res)
	if err != nil {
		return corev1.ResourceRequirements{}, nil
	}
	var privateRes private.ResourceRequirements
	err = json.Unmarshal(val, &privateRes)
	if err != nil {
		return corev1.ResourceRequirements{}, nil
	}
	return ConvertPrivateResourceRequirementsToCoreV1(&privateRes)
}

// ConvertPrivateResourceRequirementsToCoreV1 ...
func ConvertPrivateResourceRequirementsToCoreV1(res *private.ResourceRequirements) (corev1.ResourceRequirements, error) {
	var limitsCPU, limitsMemory, requestsCPU, requestsMemory resource.Quantity
	var err error

	if res.Limits.Cpu != "" {
		limitsCPU, err = resource.ParseQuantity(res.Limits.Cpu)
		if err != nil {
			return corev1.ResourceRequirements{}, fmt.Errorf("parsing CPU limit %q: %v", res.Limits.Cpu, err)
		}
	}
	if res.Limits.Memory != "" {
		limitsMemory, err = resource.ParseQuantity(res.Limits.Memory)
		if err != nil {
			return corev1.ResourceRequirements{}, fmt.Errorf("parsing memory limit %q: %v", res.Limits.Memory, err)
		}
	}
	if res.Requests.Cpu != "" {
		requestsCPU, err = resource.ParseQuantity(res.Requests.Cpu)
		if err != nil {
			return corev1.ResourceRequirements{}, fmt.Errorf("parsing CPU request %q: %v", res.Requests.Cpu, err)
		}
	}
	if res.Requests.Memory != "" {
		requestsMemory, err = resource.ParseQuantity(res.Requests.Memory)
		if err != nil {
			return corev1.ResourceRequirements{}, fmt.Errorf("parsing memory requst %q: %v", res.Limits.Memory, err)
		}
	}

	return corev1.ResourceRequirements{
		Limits: corev1.ResourceList{
			corev1.ResourceCPU:    limitsCPU,
			corev1.ResourceMemory: limitsMemory,
		},
		Requests: corev1.ResourceList{
			corev1.ResourceCPU:    requestsCPU,
			corev1.ResourceMemory: requestsMemory,
		},
	}, nil
}
