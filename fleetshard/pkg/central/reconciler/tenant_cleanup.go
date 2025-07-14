package reconciler

import (
	"context"
	"fmt"

	"github.com/golang/glog"
	"github.com/stackrox/acs-fleet-manager/fleetshard/pkg/k8s"
	"github.com/stackrox/acs-fleet-manager/internal/central/pkg/api/private"
	corev1 "k8s.io/api/core/v1"

	ctrlClient "sigs.k8s.io/controller-runtime/pkg/client"
)

const crNameLabelKey = "app.kubernetes.io/instance"

// TenantCleanup defines methods to cleanup Kubernetes resources and namespaces for tenants
// that are no longer in the list of tenants fleetshard-sync schould run on a cluster
type TenantCleanup struct {
	k8sClient      ctrlClient.Client
	nsReconciler   *namespaceReconciler
	argoReconciler *argoReconciler
}

// TenantCleanupOptions defines configuration options for the TenantCleanup logic
type TenantCleanupOptions struct {
	ArgoReconcilerOptions ArgoReconcilerOptions
}

// NewTenantCleanup returns a new TenantCleanup using given arguments
func NewTenantCleanup(k8sClient ctrlClient.Client, opts TenantCleanupOptions) *TenantCleanup {
	return &TenantCleanup{
		k8sClient:      k8sClient,
		nsReconciler:   newNamespaceReconciler(k8sClient),
		argoReconciler: newArgoReconciler(k8sClient, opts.ArgoReconcilerOptions),
	}
}

// DeleteStaleTenantK8sResources deletes all namespaces on the cluster that are labeled
// as tenant namespaces but are not in the given list of ManagedCentrals
func (t *TenantCleanup) DeleteStaleTenantK8sResources(ctx context.Context, centralListFromFmAPI *private.ManagedCentralList) error {
	namespaceList := corev1.NamespaceList{}
	matchLabels := ctrlClient.MatchingLabels{k8s.ManagedByLabelKey: k8s.ManagedByFleetshardValue}
	hasLabels := ctrlClient.HasLabels{TenantIDLabelKey, crNameLabelKey}
	if err := t.k8sClient.List(ctx, &namespaceList, matchLabels, hasLabels); err != nil {
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
		delete(namespaceNameToCrName, central.Metadata.Namespace)
	}

	for namespace, crName := range namespaceNameToCrName {
		glog.Infof("delete resources for stale tenant in namespace: %s", namespace)
		if crName == "" {
			glog.Infof("namespace %q was not propperly labeled with a tenant name, skipping deletion", namespace)
			continue
		}
		if _, err := t.DeleteK8sResources(ctx, namespace, crName); err != nil {
			glog.Errorf("failed to delete k8s resources for central: %s: %s", namespace, err.Error())
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

	// TODO(ROX-26277): This has to go into the tenantCleanup implementation
	// it is here for now for merge conflict resolution, and need additional impl before mergin ROX-26277 PR.
	argoCdAppDeleted, err := t.argoReconciler.ensureApplicationDeleted(ctx, namespace)
	if err != nil {
		return false, err
	}
	globalDeleted = globalDeleted && argoCdAppDeleted

	namespaceDeleted, err := t.nsReconciler.ensureDeleted(ctx, namespace)
	if err != nil {
		return false, fmt.Errorf("Failed to delete namespace for tenant in namespace %q: %w", namespace, err)
	}
	globalDeleted = globalDeleted && namespaceDeleted

	return globalDeleted, nil
}
