package runtime

import (
	"context"
	"github.com/stackrox/acs-fleet-manager/fleetshard/pkg/central/charts"
	"helm.sh/helm/v3/pkg/chart"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime/schema"
	ctrlClient "sigs.k8s.io/controller-runtime/pkg/client"
)

// newVPAReconciler creates a new VPA reconciler.
func newVPAReconciler(cli ctrlClient.Client, restMapper meta.RESTMapper) *vpaReconciler {
	return &vpaReconciler{
		cli:        cli,
		restMapper: restMapper,
		chart:      vpaChart,
	}
}

// vpaReconciler is the reconciler for the VPA chart.
type vpaReconciler struct {
	cli        ctrlClient.Client
	restMapper meta.RESTMapper
	chart      *chart.Chart
}

// reconcile runs the reconciliation of the VPA chart.
func (r *vpaReconciler) reconcile(ctx context.Context, config map[string]interface{}) error {
	if config == nil {
		return nil
	}
	return charts.Reconcile(ctx, r.getParamsForConfig(config))
}

// getParamsForConfig returns the parameters for the Helm reconciler for the VPA chart.
func (r *vpaReconciler) getParamsForConfig(config map[string]interface{}) charts.HelmReconcilerParams {
	return charts.HelmReconcilerParams{
		ReleaseName:     "rhacs-vpa",
		Namespace:       "rhacs-vertical-pod-autoscaling",
		ManagerName:     "fleetshard",
		Chart:           r.chart,
		Values:          config,
		Client:          r.cli,
		RestMapper:      r.restMapper,
		CreateNamespace: true,
		AllowedGVKs: []schema.GroupVersionKind{
			{
				Kind:    "Deployment",
				Group:   "apps",
				Version: "v1",
			},
			{
				Kind:    "ServiceAccount",
				Group:   "",
				Version: "v1",
			},
			{
				Kind:    "ClusterRole",
				Group:   "rbac.authorization.k8s.io",
				Version: "v1",
			},
			{
				Kind:    "ClusterRoleBinding",
				Group:   "rbac.authorization.k8s.io",
				Version: "v1",
			},
		},
	}
}

// vpaChart is the Helm chart for the VPA configuration.
var vpaChart *chart.Chart

// init initializes the VPA chart.
func init() {
	var err error
	vpaChart, err = charts.GetChart("rhacs-vertical-pod-autoscaling", nil)
	if err != nil {
		panic(err)
	}
}
