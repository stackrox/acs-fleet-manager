// Package operator provides install/upgrade logic for ACS Operator
package operator

import (
	"context"
	"fmt"
	"strings"

	"github.com/golang/glog"
	"github.com/stackrox/acs-fleet-manager/fleetshard/pkg/central/charts"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chartutil"
	apiErrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	ctrlClient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	operatorNamespace = "stackrox-operator"
	releaseName       = "rhacs-operator"
	operatorImage     = "quay.io/rhacs-eng/stackrox-operator:3.74.0"
	crdKind           = "CustomResourceDefinition"
)

// ACSOperatorManager keeps data necessary for managing ACS Operator
type ACSOperatorManager struct {
	client         ctrlClient.Client
	resourcesChart *chart.Chart
}

// InstallOrUpgrade provisions or upgrades an existing ACS Operator from helm chart template
func (u *ACSOperatorManager) InstallOrUpgrade(ctx context.Context, image string) error {
	chartVals := chartutil.Values{
		"operator": chartutil.Values{
			"image": image,
			"tag":   strings.Split(image, ":")[1],
		},
	}
	u.resourcesChart = charts.MustGetChart("rhacs-operator")
	objs, err := charts.RenderToObjects(releaseName, operatorNamespace, u.resourcesChart, chartVals)
	if err != nil {
		return fmt.Errorf("installing operator chart: %w", err)
	}

	// TODO(ROX-16338): handle namespace assigning with refactoring of chart deployment
	for _, obj := range objs {
		if obj.GetNamespace() == "" && obj.GetKind() != crdKind {
			obj.SetNamespace(operatorNamespace)
		}
		key := ctrlClient.ObjectKey{Namespace: obj.GetNamespace(), Name: obj.GetName()}
		var out unstructured.Unstructured
		out.SetGroupVersionKind(obj.GroupVersionKind())
		err := u.client.Get(ctx, key, &out)
		if err == nil {
			glog.V(10).Infof("Updating ACS Operator %s/%s", obj.GetNamespace(), obj.GetName())
			obj.SetResourceVersion(out.GetResourceVersion())
			err := u.client.Update(ctx, obj)
			if err != nil {
				return fmt.Errorf("failed to update ACS Operator %s/%s of type %v %s", key.Namespace, key.Name, obj.GroupVersionKind(), err)
			}
		}
		if !apiErrors.IsNotFound(err) {
			return fmt.Errorf("failed to retrieve object %s/%s of type %v %s", key.Namespace, key.Name, obj.GroupVersionKind(), err)
		}
		err = u.client.Create(ctx, obj)
		glog.Infof("Creating Operator %s/%s", obj.GetNamespace(), obj.GetName())
		if err != nil && !apiErrors.IsAlreadyExists(err) {
			return fmt.Errorf("failed to create object %s/%s of type %v: %w", key.Namespace, key.Name, obj.GroupVersionKind(), err)
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
