package runtime

import (
	"context"
	"encoding/json"
	"github.com/stackrox/acs-fleet-manager/fleetshard/pkg/central/charts"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/api/private"
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
func (r *vpaReconciler) reconcile(ctx context.Context, config private.VerticalPodAutoscaling) error {
	params, err := r.getParamsForConfig(config)
	if err != nil {
		return err
	}
	return charts.Reconcile(ctx, params)
}

// getParamsForConfig returns the parameters for the Helm reconciler for the VPA chart.
func (r *vpaReconciler) getParamsForConfig(config private.VerticalPodAutoscaling) (charts.HelmReconcilerParams, error) {

	jsonBytes, err := json.Marshal(config)
	if err != nil {
		return charts.HelmReconcilerParams{}, err
	}
	values := make(map[string]interface{})
	if err := json.Unmarshal(jsonBytes, &values); err != nil {
		return charts.HelmReconcilerParams{}, err
	}

	return charts.HelmReconcilerParams{
		ReleaseName:     "rhacs-vpa",
		Namespace:       "rhacs-vertical-pod-autoscaling",
		ManagerName:     "fleetshard",
		Chart:           r.chart,
		Values:          values,
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
	}, nil
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
