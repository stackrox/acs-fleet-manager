package reconciler

import (
	"context"
	"fmt"
	"github.com/golang/glog"
	"github.com/pkg/errors"
	"github.com/stackrox/acs-fleet-manager/fleetshard/pkg/central/charts"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/api/private"
	"github.com/stackrox/acs-fleet-manager/pkg/features"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chartutil"
	apiErrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	ctrlClient "sigs.k8s.io/controller-runtime/pkg/client"
	"time"
)

type tenantResourcesChartReconciler struct {
	client              ctrlClient.Client
	resourcesChart      *chart.Chart
	secureTenantNetwork bool
}

func newTenantResourcesChartReconciler(client ctrlClient.Client, resourcesChart *chart.Chart, secureTenantNetwork bool) reconciler {
	return &tenantResourcesChartReconciler{
		client:              client,
		resourcesChart:      resourcesChart,
		secureTenantNetwork: secureTenantNetwork,
	}
}

var _ reconciler = &tenantResourcesChartReconciler{}

func (t tenantResourcesChartReconciler) ensurePresent(ctx context.Context) (context.Context, error) {
	central, ok := managedCentralFromContext(ctx)
	if !ok {
		return ctx, fmt.Errorf("context does not contain a managed central")
	}
	if err := t.ensureChartResourcesExist(ctx, central); err != nil {
		return ctx, err
	}
	return ctx, nil
}

func (t tenantResourcesChartReconciler) ensureAbsent(ctx context.Context) (context.Context, error) {
	central, ok := managedCentralFromContext(ctx)
	if !ok {
		return ctx, fmt.Errorf("context does not contain a managed central")
	}
	if err := t.ensureChartResourcesDeleted(ctx, &central); err != nil {
		return ctx, err
	}
	return ctx, nil
}

func (t tenantResourcesChartReconciler) ensureChartResourcesExist(ctx context.Context, remoteCentral private.ManagedCentral) error {
	getObjectKey := func(obj *unstructured.Unstructured) string {
		return fmt.Sprintf("%s/%s/%s",
			obj.GetAPIVersion(),
			obj.GetKind(),
			obj.GetName(),
		)
	}

	vals, err := t.chartValues(remoteCentral)
	if err != nil {
		return fmt.Errorf("obtaining values for resources chart: %w", err)
	}

	if features.PrintTenantResourcesChartValues.Enabled() {
		glog.Infof("Tenant resources for central %q: %s", remoteCentral.Metadata.Name, vals)
	}

	objs, err := charts.RenderToObjects(helmReleaseName, remoteCentral.Metadata.Namespace, t.resourcesChart, vals)
	if err != nil {
		return fmt.Errorf("rendering resources chart: %w", err)
	}

	helmChartLabelValue := t.getTenantResourcesChartHelmLabelValue()

	// objectsThatShouldExist stores the keys of the objects we want to exist
	var objectsThatShouldExist = map[string]struct{}{}

	for _, obj := range objs {
		objectsThatShouldExist[getObjectKey(obj)] = struct{}{}

		if obj.GetNamespace() == "" {
			obj.SetNamespace(remoteCentral.Metadata.Namespace)
		}
		if obj.GetLabels() == nil {
			obj.SetLabels(map[string]string{})
		}
		labels := obj.GetLabels()
		labels[managedByLabelKey] = labelManagedByFleetshardValue
		labels[helmChartLabelKey] = helmChartLabelValue
		labels[helmChartNameLabel] = t.resourcesChart.Name()
		obj.SetLabels(labels)

		objectKey := ctrlClient.ObjectKey{Namespace: remoteCentral.Metadata.Namespace, Name: obj.GetName()}
		glog.Infof("Upserting object %v of type %v", objectKey, obj.GroupVersionKind())
		if err := charts.InstallOrUpdateChart(ctx, obj, t.client); err != nil {
			return fmt.Errorf("Failed to upsert object %v of type %v: %w", objectKey, obj.GroupVersionKind(), err)
		}
	}

	// Perform garbage collection
	for _, gvk := range tenantChartResourceGVKs {
		gvk := gvk
		var existingObjects unstructured.UnstructuredList
		existingObjects.SetGroupVersionKind(gvk)

		if err := t.client.List(ctx, &existingObjects,
			ctrlClient.InNamespace(remoteCentral.Metadata.Namespace),
			ctrlClient.MatchingLabels{helmChartNameLabel: t.resourcesChart.Name()},
		); err != nil {
			return fmt.Errorf("failed to list tenant resources chart objects %v: %w", gvk, err)
		}

		for _, existingObject := range existingObjects.Items {
			existingObject := &existingObject
			if _, shouldExist := objectsThatShouldExist[getObjectKey(existingObject)]; shouldExist {
				continue
			}

			// Re-check that the helm label is present & namespace matches.
			// Failsafe against some potential k8s-client bug when listing objects with a label selector
			if !t.isTenantResourcesChartObject(existingObject, &remoteCentral) {
				glog.Infof("Object %v of type %v is not managed by the resources chart", existingObject.GetName(), gvk)
				continue
			}

			if existingObject.GetDeletionTimestamp() != nil {
				glog.Infof("Object %v of type %v is already being deleted", existingObject.GetName(), gvk)
				continue
			}

			// The object exists but it should not. Delete it.
			glog.Infof("Deleting object %v of type %v", existingObject.GetName(), gvk)
			if err := t.client.Delete(ctx, existingObject); err != nil {
				if !apiErrors.IsNotFound(err) {
					return fmt.Errorf("failed to delete central tenant object %v %q in namespace %s: %w", gvk, existingObject.GetName(), remoteCentral.Metadata.Namespace, err)
				}
			}
		}
	}

	return nil
}

func (t tenantResourcesChartReconciler) ensureChartResourcesDeleted(ctx context.Context, remoteCentral *private.ManagedCentral) error {

	ctx, cancel := context.WithTimeout(ctx, 15*time.Minute)
	defer cancel()

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("timed out waiting for tenant resources chart objects to be deleted")
		case <-ticker.C:
			allObjectsDeleted := true
			for _, gvk := range tenantChartResourceGVKs {
				gvk := gvk
				var existingObjects unstructured.UnstructuredList
				existingObjects.SetGroupVersionKind(gvk)

				if err := t.client.List(ctx, &existingObjects,
					ctrlClient.InNamespace(remoteCentral.Metadata.Namespace),
					ctrlClient.MatchingLabels{helmChartNameLabel: t.resourcesChart.Name()},
				); err != nil {
					return fmt.Errorf("failed to list tenant resources chart objects %v in namespace %s: %w", gvk, remoteCentral.Metadata.Namespace, err)
				}

				for _, existingObject := range existingObjects.Items {
					existingObject := &existingObject

					// Re-check that the helm label is present & namespace matches.
					// Failsafe against some potential k8s-client bug when listing objects with a label selector
					if !t.isTenantResourcesChartObject(existingObject, remoteCentral) {
						continue
					}

					if existingObject.GetDeletionTimestamp() != nil {
						allObjectsDeleted = false
						continue
					}

					if err := t.client.Delete(ctx, existingObject); err != nil {
						if !apiErrors.IsNotFound(err) {
							return fmt.Errorf("failed to delete central tenant object %v in namespace %q: %w", gvk, remoteCentral.Metadata.Namespace, err)
						}
					}
				}
			}
			if allObjectsDeleted {
				return nil
			}
		}
	}

}

func (t tenantResourcesChartReconciler) chartValues(c private.ManagedCentral) (chartutil.Values, error) {
	if t.resourcesChart == nil {
		return nil, errors.New("resources chart is not set")
	}
	src := t.resourcesChart.Values

	// We are introducing the passing of helm values from fleetManager (and gitops). If the managed central
	// includes the tenant resource values, we will use them. Otherwise, defaults to the previous
	// implementation.
	if len(c.Spec.TenantResourcesValues) > 0 {
		values := chartutil.CoalesceTables(c.Spec.TenantResourcesValues, src)
		glog.Infof("Values: %v", values)
		return values, nil
	}

	dst := map[string]interface{}{
		"labels":      t.stringMapToMapInterface(t.getTenantLabels(c)),
		"annotations": t.stringMapToMapInterface(t.getTenantAnnotations(c)),
	}
	dst["secureTenantNetwork"] = t.secureTenantNetwork
	return chartutil.CoalesceTables(dst, src), nil
}

func (t tenantResourcesChartReconciler) isTenantResourcesChartObject(existingObject *unstructured.Unstructured, remoteCentral *private.ManagedCentral) bool {
	return existingObject.GetLabels() != nil &&
		existingObject.GetLabels()[helmChartNameLabel] == t.resourcesChart.Name() &&
		existingObject.GetLabels()[managedByLabelKey] == labelManagedByFleetshardValue &&
		existingObject.GetNamespace() == remoteCentral.Metadata.Namespace
}

func (t tenantResourcesChartReconciler) getTenantResourcesChartHelmLabelValue() string {
	// the objects rendered by the helm chart will have a label in the format
	// helm.sh/chart: <chart-name>-<chart-version>
	return fmt.Sprintf("%s-%s", t.resourcesChart.Name(), t.resourcesChart.Metadata.Version)
}

func (t tenantResourcesChartReconciler) getTenantLabels(c private.ManagedCentral) map[string]string {
	return map[string]string{
		managedByLabelKey:    labelManagedByFleetshardValue,
		instanceLabelKey:     c.Metadata.Name,
		orgIDLabelKey:        c.Spec.Auth.OwnerOrgId,
		tenantIDLabelKey:     c.Id,
		instanceTypeLabelKey: c.Spec.InstanceType,
	}
}

func (t tenantResourcesChartReconciler) getTenantAnnotations(c private.ManagedCentral) map[string]string {
	return map[string]string{
		orgNameAnnotationKey: c.Spec.Auth.OwnerOrgName,
	}
}

func (t tenantResourcesChartReconciler) stringMapToMapInterface(m map[string]string) map[string]interface{} {
	result := make(map[string]interface{}, len(m))
	for k, v := range m {
		result[k] = v
	}
	return result
}
