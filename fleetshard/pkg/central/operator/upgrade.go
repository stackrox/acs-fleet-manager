// Package operator provides install/upgrade logic for ACS Operator
package operator

import (
	"context"
	"fmt"

	"github.com/stackrox/acs-fleet-manager/fleetshard/pkg/central/charts"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chartutil"
	ctrlClient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	operatorNamespace = "stackrox-operator"
	releaseName       = "rhacs-operator"
	operatorImage     = "quay.io/rhacs-eng/stackrox-operator:3.74.0"
)

// ACSOperatorManager keeps data necessary for managing ACS Operator
type ACSOperatorManager struct {
	client         ctrlClient.Client
	resourcesChart *chart.Chart
}

// InstallOrUpgrade provisions or upgrades an existing ACS Operator from helm chart template
func (u *ACSOperatorManager) InstallOrUpgrade(ctx context.Context) error {
	chartVals := chartutil.Values{
		"operator": chartutil.Values{
			"image": operatorImage,
		},
	}
	u.resourcesChart = charts.MustGetChart("rhacs-operator")
	objs, err := charts.RenderToObjects(releaseName, operatorNamespace, u.resourcesChart, chartVals)
	if err != nil {
		return fmt.Errorf("installing operator chart: %w", err)
	}

	for _, obj := range objs {
		if obj.GetNamespace() == "" {
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
