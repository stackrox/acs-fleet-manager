// Package operator provides install/upgrade logic for ACS Operator
package operator

import (
	"context"
	"fmt"

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
)

// ACSOperatorManager ...
type ACSOperatorManager struct {
	client         ctrlClient.Client
	resourcesChart *chart.Chart
}

// Upgrade ...
func (u *ACSOperatorManager) Upgrade(ctx context.Context) error {
	vals := chartutil.Values{}
	u.resourcesChart = charts.MustGetChart("rhacs-operator")
	objs, err := charts.RenderToObjects(releaseName, operatorNamespace, u.resourcesChart, vals)
	if err != nil {
		return fmt.Errorf("installing operator chart: %w", err)
	}

	for _, obj := range objs {
		if obj.GetNamespace() == "" {
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

			continue
		}
		if !apiErrors.IsNotFound(err) {
			return fmt.Errorf("failed to retrieve object %s/%s of type %v %s", key.Namespace, key.Name, obj.GroupVersionKind(), err)
		}
		err = u.client.Create(ctx, obj)
		glog.V(10).Infof("Creating object %s/%s", obj.GetNamespace(), obj.GetName())
		if err != nil && !apiErrors.IsAlreadyExists(err) {
			return fmt.Errorf("failed to create object %s/%s of type %v: %w", key.Namespace, key.Name, obj.GroupVersionKind(), err)
		}
	}

	return nil

}

// NewACSOperatorManager ...
func NewACSOperatorManager(k8sClient ctrlClient.Client) *ACSOperatorManager {
	return &ACSOperatorManager{
		client: k8sClient,
	}
}
