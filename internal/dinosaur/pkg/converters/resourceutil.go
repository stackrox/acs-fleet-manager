package converters

import (
	"fmt"

	admin "github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/api/admin/private"
	adminPrivate "github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/api/admin/private"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/api/dbapi"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/api/private"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/api/public"
	"github.com/stackrox/rox/operator/apis/platform/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

// ConvertPublicScalingToV1 converts public API Scanner Scaling configuration into v1alpha1 Scanner Scaling configuration.
func ConvertPublicScalingToV1(scaling *public.ScannerSpecAnalyzerScaling) (v1alpha1.ScannerAnalyzerScaling, error) {
	if scaling == nil {
		return v1alpha1.ScannerAnalyzerScaling{}, nil
	}
	autoScaling := scaling.AutoScaling
	return v1alpha1.ScannerAnalyzerScaling{
		AutoScaling: (*v1alpha1.AutoScalingPolicy)(&autoScaling), // TODO: validate.
		Replicas:    &scaling.Replicas,
		MinReplicas: &scaling.MinReplicas,
		MaxReplicas: &scaling.MaxReplicas,
	}, nil
}

// ConvertAdminPrivateScalingToV1 converts admin API Scanner Scaling configuration into v1alpha1 Scanner Scaling configuration.
func ConvertAdminPrivateScalingToV1(scaling *adminPrivate.ScannerSpecAnalyzerScaling) (v1alpha1.ScannerAnalyzerScaling, error) {
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

// ConvertPrivateScalingToV1 converts private API Scanner Scaling configuration into v1alpha1 Scanner Scaling configuration.
func ConvertPrivateScalingToV1(scaling *private.ManagedCentralAllOfSpecScannerAnalyzerScaling) v1alpha1.ScannerAnalyzerScaling {
	if scaling == nil {
		return v1alpha1.ScannerAnalyzerScaling{}
	}
	autoScaling := scaling.AutoScaling
	return v1alpha1.ScannerAnalyzerScaling{
		AutoScaling: (*v1alpha1.AutoScalingPolicy)(&autoScaling), // TODO: validate.
		Replicas:    &scaling.Replicas,
		MinReplicas: &scaling.MinReplicas,
		MaxReplicas: &scaling.MaxReplicas,
	}
}

// ConvertScalingToPublic converts the internal dbapi ScannerAnalyzerScaling model into the ScannerSpecAnalyzerScaling model from the public API.
func ConvertScalingToPublic(from *dbapi.ScannerAnalyzerScaling) public.ScannerSpecAnalyzerScaling {
	return public.ScannerSpecAnalyzerScaling{
		AutoScaling: from.AutoScaling,
		Replicas:    from.Replicas,
		MinReplicas: from.MinReplicas,
		MaxReplicas: from.MaxReplicas,
	}
}

// convertCoreV1ResourceListToMap converts corev1 ResourceList into generic map.
func convertCoreV1ResourceListToMap(v1ResourceList corev1.ResourceList) map[string]string {
	v1Resources := (map[corev1.ResourceName]resource.Quantity)(v1ResourceList)
	if v1Resources == nil {
		return nil
	}
	resources := make(map[string]string)

	for name, qty := range v1Resources {
		if qtyString := qtyAsString(qty); qtyString != "" {
			resources[name.String()] = qtyString
		}
	}
	if len(resources) == 0 {
		return nil
	}
	return resources
}

// ConvertCoreV1ResourceRequirementsToPublic converts corev1 ResourceRequirements into public API ResourceRequirements.
func ConvertCoreV1ResourceRequirementsToPublic(v1Resources *corev1.ResourceRequirements) public.ResourceRequirements {
	return public.ResourceRequirements{
		Limits:   convertCoreV1ResourceListToMap(v1Resources.Limits),
		Requests: convertCoreV1ResourceListToMap(v1Resources.Requests),
	}
}

// ConvertCoreV1ResourceRequirementsToPrivate converts corev1 ResourceRequirements into private API ResourceRequirements.
func ConvertCoreV1ResourceRequirementsToPrivate(v1Resources *corev1.ResourceRequirements) private.ResourceRequirements {
	return private.ResourceRequirements{
		Limits:   convertCoreV1ResourceListToMap(v1Resources.Limits),
		Requests: convertCoreV1ResourceListToMap(v1Resources.Requests),
	}
}

// ConvertCoreV1ResourceRequirementsToAdmin converts corev1 ResourceRequirements into private admin API ResourceRequirements.
func ConvertCoreV1ResourceRequirementsToAdmin(v1Resources *corev1.ResourceRequirements) admin.ResourceRequirements {
	return admin.ResourceRequirements{
		Limits:   convertCoreV1ResourceListToMap(v1Resources.Limits),
		Requests: convertCoreV1ResourceListToMap(v1Resources.Requests),
	}
}

// ConvertPublicResourceRequirementsToCoreV1 converts public API ResourceRequirements into corev1 ResourceRequirements.
func ConvertPublicResourceRequirementsToCoreV1(res *public.ResourceRequirements) (corev1.ResourceRequirements, error) {
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

// ConvertPrivateResourceRequirementsToCoreV1 converts private API ResourceRequirements into corev1 ResourceRequirements.
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

// ConvertAdminPrivateRequirementsToCoreV1 converts admin API ResourceRequirements into corev1 ResourceRequirements.
func ConvertAdminPrivateRequirementsToCoreV1(res *adminPrivate.ResourceRequirements) (corev1.ResourceRequirements, error) {
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

func apiResourcesToCoreV1(resources map[string]string) (map[corev1.ResourceName]resource.Quantity, error) {
	var v1Resources map[corev1.ResourceName]resource.Quantity
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

func qtyAsString(qty resource.Quantity) string {
	if qty == (resource.Quantity{}) {
		// Otherwise a zero-value Quantity would produce the non-zero-value string "0".
		return ""
	}
	return (&qty).String()
}
