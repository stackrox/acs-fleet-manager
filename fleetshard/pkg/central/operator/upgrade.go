// Package operator provides install/upgrade logic for ACS Operator
package operator

import (
	"context"
	"fmt"
	"strings"

	"github.com/stackrox/acs-fleet-manager/fleetshard/pkg/central/charts"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chartutil"
	ctrlClient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	operatorNamespace = "stackrox-operator"
	releaseName       = "rhacs-operator"
	crdKind           = "CustomResourceDefinition"
)

// Image represents operator image
type Image struct {
	repository string
	tag        string
}

// ImageFromString creates Image from image string
func ImageFromString(image string) Image {
	s := strings.Split(image, ":")
	return Image{s[0], s[1]}
}

// ACSOperatorManager keeps data necessary for managing ACS Operator
type ACSOperatorManager struct {
	client         ctrlClient.Client
	resourcesChart *chart.Chart
}

// InstallOrUpgrade provisions or upgrades an existing ACS Operator from helm chart template
func (u *ACSOperatorManager) InstallOrUpgrade(ctx context.Context, image Image) error {
	chartVals := chartutil.Values{
		"operatorImage": chartutil.Values{
			"repository": image.repository,
			"tag":        image.tag,
		},
	}
	u.resourcesChart = charts.MustGetChart("rhacs-operator")
	objs, err := charts.RenderToObjects(releaseName, operatorNamespace, u.resourcesChart, chartVals)
	if err != nil {
		return fmt.Errorf("failed rendering operator chart: %w", err)
	}

	for _, obj := range objs {
		if obj.GetNamespace() == "" && obj.GetKind() != crdKind {
			obj.SetNamespace(operatorNamespace)
		}
		err := charts.InstallOrUpdateChart(ctx, obj, u.client)
		if err != nil {
			return fmt.Errorf("failed to update operator object %w", err)
		}
	}

	return nil

}

// NewACSOperatorManager creates a new ACS Operator Manager
func NewACSOperatorManager(k8sClient ctrlClient.Client) *ACSOperatorManager {
	return &ACSOperatorManager{
		client: k8sClient,
	}
}
