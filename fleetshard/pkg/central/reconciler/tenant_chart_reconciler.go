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
)

// TenantChartReconciler provides methods to reconcile additional kubernetes resources
// required for a central tenant and defined by a helm chart
type TenantChartReconciler struct {
	// client is the controller runtime client used for chart reconciliations
	client              ctrlClient.Client
	chart               *chart.Chart
	secureTenantNetwork bool
}

// NewTenantChartReconciler creates a TenantChartReconciler with given arguments
// This function uses the resourceChart default
func NewTenantChartReconciler(client ctrlClient.Client, secureTenantNetwork bool) *TenantChartReconciler {
	return &TenantChartReconciler{client: client, chart: resourcesChart, secureTenantNetwork: secureTenantNetwork}
}

// WithChart overrides the default chart used by the TenantChartReconciler
func (r *TenantChartReconciler) WithChart(c *chart.Chart) *TenantChartReconciler {
	r.chart = c
	return r
}

// EnsureResourcesExist installs or updates the chart for remoteCentral on the Kuberntes cluster
func (r *TenantChartReconciler) EnsureResourcesExist(ctx context.Context, remoteCentral private.ManagedCentral) error {
	getObjectKey := func(obj *unstructured.Unstructured) string {
		return fmt.Sprintf("%s/%s/%s",
			obj.GetAPIVersion(),
			obj.GetKind(),
			obj.GetName(),
		)
	}

	vals, err := r.chartValues(remoteCentral)
	if err != nil {
		return fmt.Errorf("obtaining values for resources chart: %w", err)
	}

	if features.PrintTenantResourcesChartValues.Enabled() {
		glog.Infof("Tenant resources for central %q: %s", remoteCentral.Metadata.Name, vals)
	}

	objs, err := charts.RenderToObjects(helmReleaseName, remoteCentral.Metadata.Namespace, r.chart, vals)
	if err != nil {
		return fmt.Errorf("rendering resources chart: %w", err)
	}

	helmChartLabelValue := r.getTenantResourcesChartHelmLabelValue()

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
		labels[helmChartNameLabel] = r.chart.Name()
		obj.SetLabels(labels)

		objectKey := ctrlClient.ObjectKey{Namespace: remoteCentral.Metadata.Namespace, Name: obj.GetName()}
		glog.Infof("Upserting object %v of type %v", objectKey, obj.GroupVersionKind())
		if err := charts.InstallOrUpdateChart(ctx, obj, r.client); err != nil {
			return fmt.Errorf("Failed to upsert object %v of type %v: %w", objectKey, obj.GroupVersionKind(), err)
		}
	}

	// Perform garbage collection
	for _, gvk := range tenantChartResourceGVKs {
		gvk := gvk
		var existingObjects unstructured.UnstructuredList
		existingObjects.SetGroupVersionKind(gvk)

		if err := r.client.List(ctx, &existingObjects,
			ctrlClient.InNamespace(remoteCentral.Metadata.Namespace),
			ctrlClient.MatchingLabels{helmChartNameLabel: r.chart.Name()},
		); err != nil {
			return fmt.Errorf("failed to list tenant resources chart objects %v: %w", gvk, err)
		}

		for _, existingObject := range existingObjects.Items {
			// prevents erros for using address of the shared loop variable (gosec 601) by making a copy, no longer necessary once updated to Go 1.22
			existingObject := existingObject
			if _, shouldExist := objectsThatShouldExist[getObjectKey(&existingObject)]; shouldExist {
				continue
			}

			// Re-check that the helm label is present & namespace matches.
			// Failsafe against some potential k8s-client bug when listing objects with a label selector
			if !r.isTenantResourcesChartObject(&existingObject, remoteCentral.Metadata.Namespace) {
				glog.Infof("Object %v of type %v is not managed by the resources chart", existingObject.GetName(), gvk)
				continue
			}

			if existingObject.GetDeletionTimestamp() != nil {
				glog.Infof("Object %v of type %v is already being deleted", existingObject.GetName(), gvk)
				continue
			}

			// The object exists but it should not. Delete it.
			glog.Infof("Deleting object %v of type %v", existingObject.GetName(), gvk)
			if err := r.client.Delete(ctx, &existingObject); err != nil {
				if !apiErrors.IsNotFound(err) {
					return fmt.Errorf("failed to delete central tenant object %v %q in namespace %s: %w", gvk, existingObject.GetName(), remoteCentral.Metadata.Namespace, err)
				}
			}
		}
	}

	return nil
}

// EnsureResourcesDeleted deletes all resources associated with the chart and namepsace from the Kubernetes cluster
func (r *TenantChartReconciler) EnsureResourcesDeleted(ctx context.Context, namespace string) (bool, error) {
	allObjectsDeleted := true

	for _, gvk := range tenantChartResourceGVKs {
		gvk := gvk
		var existingObjects unstructured.UnstructuredList
		existingObjects.SetGroupVersionKind(gvk)

		if err := r.client.List(ctx, &existingObjects,
			ctrlClient.InNamespace(namespace),
			ctrlClient.MatchingLabels{helmChartNameLabel: r.chart.Name()},
		); err != nil {
			return false, fmt.Errorf("failed to list tenant resources chart objects %v in namespace %s: %w", gvk, namespace, err)
		}

		for _, existingObject := range existingObjects.Items {
			// prevents erros for using address of the shared loop variable (gosec 601) by making a copy, no longer necessary once updated to Go 1.22
			existingObject := existingObject

			// Re-check that the helm label is present & namespace matches.
			// Failsafe against some potential k8s-client bug when listing objects with a label selector
			if !r.isTenantResourcesChartObject(&existingObject, namespace) {
				continue
			}

			if existingObject.GetDeletionTimestamp() != nil {
				allObjectsDeleted = false
				continue
			}

			if err := r.client.Delete(ctx, &existingObject); err != nil {
				if !apiErrors.IsNotFound(err) {
					return false, fmt.Errorf("failed to delete central tenant object %v in namespace %q: %w", gvk, namespace, err)
				}
			}
		}
	}

	return allObjectsDeleted, nil
}

func (r *TenantChartReconciler) chartValues(c private.ManagedCentral) (chartutil.Values, error) {
	if r.chart == nil {
		return nil, errors.New("resources chart is not set")
	}
	src := r.chart.Values

	// We are introducing the passing of helm values from fleetManager (and gitops). If the managed central
	// includes the tenant resource values, we will use them. Otherwise, defaults to the previous
	// implementation.
	if len(c.Spec.TenantResourcesValues) > 0 {
		values := chartutil.CoalesceTables(c.Spec.TenantResourcesValues, src)
		glog.Infof("Values: %v", values)
		return values, nil
	}

	dst := map[string]interface{}{
		"labels":      stringMapToMapInterface(getTenantLabels(c)),
		"annotations": stringMapToMapInterface(getTenantAnnotations(c)),
	}
	dst["secureTenantNetwork"] = r.secureTenantNetwork
	return chartutil.CoalesceTables(dst, src), nil
}

func (r *TenantChartReconciler) isTenantResourcesChartObject(existingObject *unstructured.Unstructured, namespace string) bool {
	return existingObject.GetLabels() != nil &&
		existingObject.GetLabels()[helmChartNameLabel] == r.chart.Name() &&
		existingObject.GetLabels()[managedByLabelKey] == labelManagedByFleetshardValue &&
		existingObject.GetNamespace() == namespace
}

func (r *TenantChartReconciler) getTenantResourcesChartHelmLabelValue() string {
	// the objects rendered by the helm chart will have a label in the format
	// helm.sh/chart: <chart-name>-<chart-version>
	return fmt.Sprintf("%s-%s", r.chart.Name(), r.chart.Metadata.Version)
}
