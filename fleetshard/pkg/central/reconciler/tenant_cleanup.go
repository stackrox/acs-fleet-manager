package reconciler

import (
	"context"
	"fmt"

	"github.com/golang/glog"
	"github.com/stackrox/acs-fleet-manager/fleetshard/pkg/k8s"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/api/private"
	"helm.sh/helm/v3/pkg/chart"
	"k8s.io/kubernetes/pkg/apis/core"
	ctrlClient "sigs.k8s.io/controller-runtime/pkg/client"
)

const crNameLabelKey = "app.kubernetes.io/instance"

// TenantCleanup defines methods to cleanup Kubernetes resources and namespaces for tenants
// that are no longer in the list of tenants fleetshard-sync schould run on a cluster
type TenantCleanup struct {
	k8sClient           ctrlClient.Client
	secureTenantNetwork bool
	chart               *chart.Chart
}

// NewTenantCleanup returns a new TenantCleanup using given arguments
func NewTenantCleanup(k8sClient ctrlClient.Client, secureTenantNetwork bool) *TenantCleanup {
	return &TenantCleanup{k8sClient: k8sClient, secureTenantNetwork: secureTenantNetwork}
}

// DeleteStaleTenantK8sResources deletes all namespaces on the cluster that are labeled
// as tenant namespaces but are not in the given list of ManagedCentrals
func (t *TenantCleanup) DeleteStaleTenantK8sResources(ctx context.Context, centralListFromFmAPI *private.ManagedCentralList) error {
	namespaceList := core.NamespaceList{}
	labels := map[string]string{k8s.ManagedByLabelKey: k8s.ManagedByFleetshardValue}
	if err := t.k8sClient.List(ctx, &namespaceList, ctrlClient.MatchingLabels(labels)); err != nil {
		return fmt.Errorf("Failed to list all tenant namespaces: %w", err)
	}

	if len(namespaceList.Items) == 0 {
		return nil
	}

	namespaceNameToCrName := make(map[string]string, len(namespaceList.Items))
	for _, ns := range namespaceList.Items {
		namespaceNameToCrName[ns.Name] = ns.Labels[crNameLabelKey]
	}

	for _, central := range centralListFromFmAPI.Items {
		// the namespaceName is equal to the ID of a central
		delete(namespaceNameToCrName, central.Id)
	}

	for namespace, crName := range namespaceNameToCrName {
		if _, err := t.DeleteK8sResources(ctx, namespace, crName); err != nil {
			glog.Errorf("Failed to delete k8s resources for central: %s: %s", namespace, err.Error())
		}
	}

	return nil
}

// DeleteK8sResources deletes all associated resources for a managed central from the cluster.
// Returns potential errors and a bool indicating whether deletion went through successfully
func (t *TenantCleanup) DeleteK8sResources(ctx context.Context, namespace string, tenantName string) (bool, error) {
	// Deleting the NS is not enough to cleanup a tenant as there could be non-namespaced resources
	// within the tenant resource chart or created by the CR. Because of that we delete chart and CR first
	// to allow propper cleanup of such resources throug helm/ACS operator before removing the namespace.
	// If any resources wouldn't be deleted by namespace deletion add them here.
	globalDeleted := true

	chartReconciler := NewTenantChartReconciler(t.k8sClient, t.secureTenantNetwork)
	if t.chart != nil {
		chartReconciler = chartReconciler.WithChart(t.chart)
	}

	deleted, err := chartReconciler.EnsureResourcesDeleted(ctx, namespace)
	if err != nil {
		return false, fmt.Errorf("Failed to delete chart resources in namespace %q: %w", namespace, err)
	}
	globalDeleted = globalDeleted && deleted

	crReconciler := NewCentralCrReconciler(t.k8sClient)

	deleted, err = crReconciler.EnsureDeleted(ctx, namespace, tenantName)
	if err != nil {
		return false, fmt.Errorf("Failed to delete central CR in namespace %q: %w", namespace, err)
	}
	globalDeleted = globalDeleted && deleted

	nsReconciler := NewNamespaceReconciler(t.k8sClient)
	deleted, err = nsReconciler.EnsureDeleted(ctx, namespace)
	if err != nil {
		return false, fmt.Errorf("Failed to delete namespace for tenant in namespace %q: %w", namespace, err)
	}
	globalDeleted = globalDeleted && deleted

	return globalDeleted, nil
}
