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
	if qty == (resource.Quantity{}) {
		return ""
	}
	return (&qty).String()
}

// ConvertCoreV1ResourceRequirementsToPublic ...
func ConvertCoreV1ResourceRequirementsToPublic(res *v1.ResourceRequirements) public.ResourceRequirements {
	var resources public.ResourceRequirements
	limits := make(map[string]string)
	requests := make(map[string]string)

	for k, v := range res.Limits {
		limits[k.String()] = v.String()
	}
	if len(limits) > 0 {
		resources.Limits = limits
	}
	for k, v := range res.Requests {
		requests[k.String()] = v.String()
	}
	if len(requests) > 0 {
		resources.Requests = requests
	}

	return resources
}

// ConvertCoreV1ResourceRequirementsToPrivate ...
func ConvertCoreV1ResourceRequirementsToPrivate(res *v1.ResourceRequirements) private.ResourceRequirements {
	var resources private.ResourceRequirements
	limits := make(map[string]string)
	requests := make(map[string]string)

	for k, v := range res.Limits {
		limits[k.String()] = v.String()
	}
	if len(limits) > 0 {
		resources.Limits = limits
	}
	for k, v := range res.Requests {
		requests[k.String()] = v.String()
	}
	if len(requests) > 0 {
		resources.Requests = requests
	}

	return resources
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

func apiResourcesToCoreV1(resources map[string]string) (map[corev1.ResourceName]resource.Quantity, error) {
	var v1Resources map[corev1.ResourceName]resource.Quantity
	// := make(map[corev1.ResourceName]resource.Quantity)
	for name, qty := range resources {
		if qty == "" {
			continue
		}
		resourceQty, err := resource.ParseQuantity(qty)
		if err != nil {
			return nil, fmt.Errorf("parsing quantity %q for resource %s: %v", qty, name, err)
		}
		if v1Resources == nil {
			v1Resources = make(map[corev1.ResourceName]resource.Quantity)
		}
		v1Resources[corev1.ResourceName(name)] = resourceQty
	}
	return v1Resources, nil
}

// ConvertPrivateResourceRequirementsToCoreV1 ...
func ConvertPrivateResourceRequirementsToCoreV1(res *private.ResourceRequirements) (corev1.ResourceRequirements, error) {
	requests, err := apiResourcesToCoreV1(res.Requests)
	if err != nil {
		return corev1.ResourceRequirements{}, err
	}

	limits, err := apiResourcesToCoreV1(res.Limits)
	if err != nil {
		return corev1.ResourceRequirements{}, err
	}

	return corev1.ResourceRequirements{
		Limits:   limits,
		Requests: requests,
	}, nil
}
