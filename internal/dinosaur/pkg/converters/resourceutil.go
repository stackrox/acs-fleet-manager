package converters

import (
	"github.com/pkg/errors"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/api/private"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/api/public"
	"github.com/stackrox/rox/operator/apis/platform/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

// We need some helper functions for converting the generated types for `ResourceRequirements` from our own API into the
// proper `ResourceRequirements` type from the Kubernetes core/v1 API, because the latter is used within the `v1alpha1.Central` type,
// while at the same time OpenAPI does not provide a way for referencing official Kubernetes type schemas.

func convertPrivateResourceListToCoreV1(r *private.ResourceList) (corev1.ResourceList, error) {
	var cpuQty resource.Quantity
	var memQty resource.Quantity
	var err error
	if r == nil {
		return corev1.ResourceList{}, nil
	}
	if r.Cpu != "" {
		cpuQty, err = resource.ParseQuantity(r.Cpu)
		if err != nil {
			return corev1.ResourceList{}, errors.Wrapf(err, "parsing CPU quantity %q", r.Cpu)
		}
	}
	if r.Memory != "" {
		memQty, err = resource.ParseQuantity(r.Memory)
		if err != nil {
			return corev1.ResourceList{}, errors.Wrapf(err, "parsing memory quantity %q", r.Memory)
		}
	}
	return corev1.ResourceList{
		corev1.ResourceCPU:    cpuQty,
		corev1.ResourceMemory: memQty,
	}, nil
}

// ConvertPrivateResourceRequirementsToCoreV1 ...
func ConvertPrivateResourceRequirementsToCoreV1(res *private.ResourceRequirements) (*corev1.ResourceRequirements, error) {
	if res == nil {
		return nil, nil
	}
	requests, err := convertPrivateResourceListToCoreV1(&res.Requests)
	if err != nil {
		return nil, errors.Wrap(err, "parsing resource requests")
	}
	limits, err := convertPrivateResourceListToCoreV1(&res.Limits)
	if err != nil {
		return nil, errors.Wrap(err, "parsing resource limits")
	}
	return &corev1.ResourceRequirements{
		Requests: requests,
		Limits:   limits,
	}, nil
}

func convertPublicResourceListToCoreV1(r *public.ResourceList) (v1.ResourceList, error) {
	var cpuQty resource.Quantity
	var memQty resource.Quantity
	var err error
	if r == nil {
		return v1.ResourceList{}, nil
	}
	if r.Cpu != "" {
		cpuQty, err = resource.ParseQuantity(r.Cpu)
		if err != nil {
			return v1.ResourceList{}, errors.Wrapf(err, "parsing CPU quantity %q", r.Cpu)
		}
	}
	if r.Memory != "" {
		memQty, err = resource.ParseQuantity(r.Memory)
		if err != nil {
			return v1.ResourceList{}, errors.Wrapf(err, "parsing memory quantity %q", r.Memory)
		}
	}
	return v1.ResourceList{
		v1.ResourceCPU:    cpuQty,
		v1.ResourceMemory: memQty,
	}, nil
}

// ConvertPublicResourceRequirementsToCoreV1 ...
func ConvertPublicResourceRequirementsToCoreV1(res *public.ResourceRequirements) (*v1.ResourceRequirements, error) {
	if res == nil {
		return nil, nil
	}
	requests, err := convertPublicResourceListToCoreV1(&res.Requests)
	if err != nil {
		return nil, errors.Wrap(err, "parsing resource requests")
	}
	limits, err := convertPublicResourceListToCoreV1(&res.Limits)
	if err != nil {
		return nil, errors.Wrap(err, "parsing resource limits")
	}
	return &v1.ResourceRequirements{
		Requests: requests,
		Limits:   limits,
	}, nil
}

// ConvertPrivateScalingToV1 ...
func ConvertPrivateScalingToV1(scaling *private.ManagedCentralAllOfSpecScannerAnalyzerScaling) (*v1alpha1.ScannerAnalyzerScaling, error) {
	if scaling == nil {
		return nil, nil
	}
	autoScaling := scaling.AutoScaling
	return &v1alpha1.ScannerAnalyzerScaling{
		AutoScaling: (*v1alpha1.AutoScalingPolicy)(&autoScaling), // TODO(create-ticket): validate.
		Replicas:    &scaling.Replicas,
		MinReplicas: &scaling.MinReplicas,
		MaxReplicas: &scaling.MaxReplicas,
	}, nil
}

// ConvertPublicScalingToV1 ...
func ConvertPublicScalingToV1(scaling *public.ScannerSpecAnalyzerScaling) (*v1alpha1.ScannerAnalyzerScaling, error) {
	if scaling == nil {
		return nil, nil
	}
	autoScaling := scaling.AutoScaling
	return &v1alpha1.ScannerAnalyzerScaling{
		AutoScaling: (*v1alpha1.AutoScalingPolicy)(&autoScaling), // TODO(create-ticket): validate.
		Replicas:    &scaling.Replicas,
		MinReplicas: &scaling.MinReplicas,
		MaxReplicas: &scaling.MaxReplicas,
	}, nil
}
