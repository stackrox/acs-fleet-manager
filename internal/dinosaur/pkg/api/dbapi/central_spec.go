package dbapi

import (
	"fmt"

	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/api/admin/private"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

// CentralSpec ...
type CentralSpec struct {
	Resources corev1.ResourceRequirements `json:"resources,omitempty"`
}

var (
	supportedResources = []corev1.ResourceName{corev1.ResourceCPU, corev1.ResourceMemory}
)

func isResourceSupported(name string) (corev1.ResourceName, bool) {
	resourceName := corev1.ResourceName(name)
	for _, supportedResource := range supportedResources {
		if supportedResource == resourceName {
			return resourceName, true
		}
	}
	return corev1.ResourceName(""), false
}

func updateResourcesList(to *corev1.ResourceList, from map[string]string) error {
	newResourceList := to.DeepCopy()
	for name, qty := range from {
		if qty == "" {
			continue
		}
		resourceName, isSupported := isResourceSupported(name)
		if !isSupported {
			// TODO(mclasmei): log
			continue
		}
		resourceQty, err := resource.ParseQuantity(qty)
		if err != nil {
			return fmt.Errorf("parsing %s quantity %q: %w", resourceName, qty, err)
		}
		if newResourceList == nil {
			newResourceList = corev1.ResourceList(make(map[corev1.ResourceName]resource.Quantity))
		}
		newResourceList[resourceName] = resourceQty
	}
	*to = newResourceList
	return nil
}

func updateCoreV1Resources(to *corev1.ResourceRequirements, from private.ResourceRequirements) error {
	newResources := to.DeepCopy()

	err := updateResourcesList(&newResources.Limits, from.Limits)
	if err != nil {
		return err
	}
	err = updateResourcesList(&newResources.Requests, from.Requests)
	if err != nil {
		return err
	}

	*to = *newResources
	return nil
}

// UpdateFromPrivateAPI updates the CentralSpec using the non-zero fields from the API's CentralSpec.
func (c *CentralSpec) UpdateFromPrivateAPI(apiCentralSpec *private.CentralSpec) error {
	err := updateCoreV1Resources(&c.Resources, apiCentralSpec.Resources)
	if err != nil {
		return fmt.Errorf("updating CentralSpec: %w", err)
	}
	return nil
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

func updateScannerAnalyzerScaling(s *ScannerAnalyzerScaling, apiScaling private.ScannerSpecAnalyzerScaling) error {
	new := *s

	if apiScaling.AutoScaling != "" {
		new.AutoScaling = apiScaling.AutoScaling
	}
	if apiScaling.MaxReplicas != 0 {
		new.MaxReplicas = apiScaling.MaxReplicas
	}
	if apiScaling.MinReplicas != 0 {
		new.MinReplicas = apiScaling.MinReplicas
	}
	if apiScaling.Replicas != 0 {
		new.Replicas = apiScaling.Replicas
	}

	// TODO: validation of the resulting configuration new.

	*s = new
	return nil
}

// UpdateFromPrivateAPI updates the ScannerSpec using the non-zero fields from the API's ScannerSpec.
func (s *ScannerSpec) UpdateFromPrivateAPI(apiSpec *private.ScannerSpec) error {
	var err error
	new := *s

	err = updateCoreV1Resources(&new.Analyzer.Resources, apiSpec.Analyzer.Resources)
	if err != nil {
		return fmt.Errorf("updating ScannerSpec Analyzer: %w", err)
	}
	err = updateScannerAnalyzerScaling(&new.Analyzer.Scaling, apiSpec.Analyzer.Scaling)
	if err != nil {
		return fmt.Errorf("updating ScannerSpec Analyzer Scaling: %w", err)
	}
	err = updateCoreV1Resources(&new.Db.Resources, apiSpec.Db.Resources)
	if err != nil {
		return fmt.Errorf("updating ScannerSpec DB: %w", err)
	}
	*s = new
	return nil
}
