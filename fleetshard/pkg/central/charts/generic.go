package charts

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	jsonpatch "github.com/evanphx/json-patch/v5"
	"github.com/golang/glog"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chartutil"
	v1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/sets"
	ctrlClient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	labelManagedBy            = "app.kubernetes.io/managed-by"
	labelHelmReleaseName      = "meta.helm.sh/release-name"
	labelHelmReleaseNamespace = "meta.helm.sh/release-namespace"
	labelHelmChart            = "helm.sh/chart"
)

// HelmRecocilerParams contains the parameters required to reconcile a Helm release.
type HelmReconcilerParams struct {
	ReleaseName     string
	Namespace       string
	ManagerName     string
	Chart           *chart.Chart
	Values          chartutil.Values
	Client          ctrlClient.Client
	RestMapper      meta.RESTMapper
	AllowedGVKs     []schema.GroupVersionKind
	CreateNamespace bool
}

func validateParams(p HelmReconcilerParams) error {
	if p.ReleaseName == "" {
		return fmt.Errorf("ReleaseName cannot be empty")
	}
	if p.Namespace == "" {
		return fmt.Errorf("Namespace cannot be empty")
	}
	if p.ManagerName == "" {
		return fmt.Errorf("ManagerName cannot be empty")
	}
	if p.Chart == nil {
		return fmt.Errorf("Chart cannot be nil")
	}
	if p.Client == nil {
		return fmt.Errorf("Client cannot be nil")
	}
	if p.RestMapper == nil {
		return fmt.Errorf("RestMapper cannot be nil")
	}
	if len(p.AllowedGVKs) == 0 {
		return fmt.Errorf("AllowedGVKs cannot be empty")
	}
	return nil
}

// Reconcile reconciles a Helm release by ensuring that the objects in the Helm Chart are created, updated or garbage-collected in the cluster.
// This is a generic reconciliation method that can be used to reconcile any Helm release programmatically.
// This routine does not create a "helm release secret", but rather will reconcile objects based on the GVKs
// provided in HelmReconcilerParams.AllowedGVKs. It uses ownership labels to track ownership of objects, and will fail
// to update or delete objects that do not have those labels.
func Reconcile(ctx context.Context, p HelmReconcilerParams) error {

	// sanity checks
	if err := validateParams(p); err != nil {
		return err
	}

	if p.CreateNamespace {
		if err := ensureNamespaceExists(ctx, p.Client, p.Namespace); err != nil {
			return err
		}
	}

	// Creating a map of allowed GVKs for faster lookup
	allowedGvkMap := make(map[schema.GroupVersionKind]struct{})
	for _, gvk := range p.AllowedGVKs {
		allowedGvkMap[gvk] = struct{}{}
	}

	// Render the Helm chart
	renderedObjs, err := RenderToObjects(p.ReleaseName, p.Namespace, p.Chart, p.Values)
	if err != nil {
		return fmt.Errorf("failed to render objects from chart: %w", err)
	}

	// Grouping the rendered objects by GVK
	renderedObjsByGVK := make(map[schema.GroupVersionKind][]*unstructured.Unstructured)
	for _, renderedObj := range renderedObjs {
		gvk := renderedObj.GroupVersionKind()
		// Fail if the rendered object GVK is not in the allowed GVKs
		if _, ok := allowedGvkMap[gvk]; !ok {
			return fmt.Errorf("object %s has unexpected GVK %s", renderedObj.GetName(), gvk.String())
		}
		renderedObjsByGVK[gvk] = append(renderedObjsByGVK[gvk], renderedObj)
	}

	ownershipLabels := getOwnershipLabels(p.Chart, p.ReleaseName, p.Namespace, p.ManagerName)

	// Reconcile each allowedGVK separately
	for allowedGVK := range allowedGvkMap {
		renderedObjsForGvk := renderedObjsByGVK[allowedGVK]
		if err := reconcileGvk(ctx, p, allowedGVK, renderedObjsForGvk, ownershipLabels); err != nil {
			return err
		}
	}

	return nil

}

// ensureNamespaceExists ensures that the namespace with the given name exists in the cluster.
func ensureNamespaceExists(ctx context.Context, cli ctrlClient.Client, name string) error {
	ns := &v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
	}
	var existing v1.Namespace
	if err := cli.Get(ctx, ctrlClient.ObjectKeyFromObject(ns), &existing); err != nil {
		if !k8serrors.IsNotFound(err) {
			return fmt.Errorf("failed to get namespace %s: %w", name, err)
		}
	} else {
		if existing.DeletionTimestamp != nil {
			return fmt.Errorf("namespace %s is being deleted", name)
		}
		return nil
	}

	if err := cli.Create(ctx, ns); err != nil {
		if !k8serrors.IsAlreadyExists(err) {
			return fmt.Errorf("failed to create namespace %s: %w", name, err)
		}
	}
	return nil
}

// reconcileGvk will reconcile objects with the given GroupVersionKind.
func reconcileGvk(ctx context.Context, params HelmReconcilerParams, gvk schema.GroupVersionKind, wantObjs []*unstructured.Unstructured, ownershipLabels map[string]string) error {

	restMapping, err := params.RestMapper.RESTMapping(gvk.GroupKind(), gvk.Version)
	if err != nil {
		return fmt.Errorf("failed to get rest mapping for %s: %w", gvk.String(), err)
	}

	// Checks if the GVK is cluster-scoped or namespaced-scoped
	isNamespacedGVK := restMapping.Scope.Name() == meta.RESTScopeNameNamespace

	existingObjs := &unstructured.UnstructuredList{}

	{
		// List existing objects of the given GVK

		var listOptions []ctrlClient.ListOption
		listOptions = append(listOptions, ctrlClient.MatchingLabels{
			labelManagedBy:            params.ManagerName,
			labelHelmReleaseNamespace: params.Namespace,
			labelHelmReleaseName:      params.ReleaseName,

			// Do not include the labelHelmChart, because it includes the chart version.
			// For example, "helm.sh/chart": "my-chart-0.0.0"
			// If the chart version changes, we still assume that the objects are managed by the release.
		})

		// If the GVK is namespaced, we list objects in the namespace
		if isNamespacedGVK {
			listOptions = append(listOptions, ctrlClient.InNamespace(params.Namespace))
		}

		existingObjs.SetGroupVersionKind(gvk)
		if err := params.Client.List(ctx, existingObjs, listOptions...); err != nil {
			return fmt.Errorf("failed to list existing objects of kind %s: %w", gvk.String(), err)
		}
	}

	// Objects we want
	wantNames := sets.NewString()
	wantByName := make(map[string]*unstructured.Unstructured)
	for _, obj := range wantObjs {
		obj := obj
		wantNames.Insert(obj.GetName())
		wantByName[obj.GetName()] = obj
	}

	// Objects we have
	existingNames := sets.NewString()
	existingByName := make(map[string]*unstructured.Unstructured)
	for _, existingObj := range existingObjs.Items {
		existingObj := existingObj
		existingNames.Insert(existingObj.GetName())
		existingByName[existingObj.GetName()] = &existingObj
	}

	// Objects to delete
	namesToDelete := existingNames.Difference(wantNames)

	// Delete phase
	for _, nameToDelete := range namesToDelete.List() {
		objToDelete := existingByName[nameToDelete]

		glog.Infof("deleting object %q of type %v", nameToDelete, gvk)

		// Do not delete object that are not managed by us
		if err := checkOwnership(objToDelete, params.ManagerName, params.ReleaseName, params.Namespace); err != nil {
			return fmt.Errorf("cannot delete object %q of type %v: %w", nameToDelete, gvk, err)
		}
		// Do not delete object that is already being deleted
		if objToDelete.GetDeletionTimestamp() != nil {
			continue
		}
		if err := params.Client.Delete(ctx, objToDelete); err != nil {
			if !k8serrors.IsNotFound(err) {
				return fmt.Errorf("failed to delete object %s: %w", nameToDelete, err)
			}
		}
	}

	// Create / Update
	for _, wantObj := range wantObjs {
		objectName := wantObj.GetName()
		applyLabelsToObject(wantObj, ownershipLabels)
		if isNamespacedGVK {
			wantObj.SetNamespace(params.Namespace)
		}
		if existingObject, alreadyExists := existingByName[objectName]; alreadyExists {

			// Do not update object that are not managed by us
			if err := checkOwnership(existingObject, params.ManagerName, params.ReleaseName, params.Namespace); err != nil {
				return fmt.Errorf("cannot update object %q of type %v: %w", objectName, gvk, err)
			}

			// Do not update object that is being deleted
			if existingObject.GetDeletionTimestamp() != nil {
				return fmt.Errorf("cannot update object %q of type %v because it is being deleted", objectName, gvk)
			}

			wantObj.SetResourceVersion(existingObject.GetResourceVersion())

			patch, err := createPatch(existingObject.Object, wantObj.Object)
			if err != nil {
				return fmt.Errorf("failed to create patch for object %q of type %v: %w", objectName, gvk, err)
			}

			if len(patch) == 0 {
				glog.Infof("object %q of type %v is up-to-date", objectName, gvk)
				continue
			} else {
				glog.Infof("object %q of type %v is not up-to-date", objectName, gvk)
				glog.Infof("diff: %v", string(patch))
			}

			if err := params.Client.Update(ctx, wantObj); err != nil {
				return fmt.Errorf("failed to update object %q of type %v: %w", objectName, gvk, err)
			}
		} else {
			// The object doesn't exist, create it

			glog.Infof("creating object %q of type %v", objectName, gvk)

			if err := params.Client.Create(ctx, wantObj); err != nil {
				if k8serrors.IsAlreadyExists(err) {

					return fmt.Errorf("cannot create object %q of type %v because it already exists and is not managed by %q or is not part of release %q", objectName, gvk, params.ManagerName, params.ReleaseName)
				} else {
					return fmt.Errorf("failed to create object %s: %w", objectName, err)
				}
			}
		}
	}

	return nil
}

// getOwnershipLabels returns the labels that should be applied to objects created by the Helm release.
// The presence of those labels on an object means that the object is owned by the Helm release.
func getOwnershipLabels(chart *chart.Chart, releaseName, releaseNamespace, managerName string) map[string]string {
	result := make(map[string]string)
	result[labelHelmChart] = fmt.Sprintf("%s-%s", chart.Metadata.Name, chart.Metadata.Version)
	result[labelHelmReleaseNamespace] = releaseNamespace
	result[labelHelmReleaseName] = releaseName
	result[labelManagedBy] = managerName
	return result
}

// checkOwnership checks that a given object is managed by the given Helm release.
func checkOwnership(obj *unstructured.Unstructured, managerName, releaseName, releaseNamespace string) error {

	labels := obj.GetLabels()
	if labels == nil {
		labels = make(map[string]string)
	}

	var errs []error
	if err := requireValue(labels, labelManagedBy, managerName); err != nil {
		errs = append(errs, fmt.Errorf("label validation error: %s", err))
	}
	if err := requireValue(labels, labelHelmReleaseName, releaseName); err != nil {
		errs = append(errs, fmt.Errorf("label validation error: %s", err))
	}
	if err := requireValue(labels, labelHelmReleaseNamespace, releaseNamespace); err != nil {
		errs = append(errs, fmt.Errorf("label validation error: %s", err))
	}

	if len(errs) > 0 {
		return fmt.Errorf("invalid ownership metadata: %w", errors.Join(errs...))
	}

	return nil

}

// requireValue checks that a given key in a map has a specific value.
func requireValue(meta map[string]string, k, v string) error {
	actual, ok := meta[k]
	if !ok {
		return fmt.Errorf("missing key %q: must be set to %q", k, v)
	}
	if actual != v {
		return fmt.Errorf("key %q must be set to %q: current value is %q", k, v, actual)
	}
	return nil
}

// applyLabelsToObject applies the given labels to the given object
func applyLabelsToObject(obj *unstructured.Unstructured, labels map[string]string) {
	existing := obj.GetLabels()
	if existing == nil {
		existing = make(map[string]string)
	}
	for k, v := range labels {
		existing[k] = v
	}
	obj.SetLabels(existing)
}

// getPatchData will return difference between original and modified document
func createPatch(originalObj, modifiedObj interface{}) ([]byte, error) {
	originalData, err := json.Marshal(originalObj)
	if err != nil {
		return nil, fmt.Errorf("failed marshal original data: %w", err)
	}
	modifiedData, err := json.Marshal(modifiedObj)
	if err != nil {
		return nil, fmt.Errorf("failed marshal modified data: %w", err)
	}

	patchBytes, err := jsonpatch.CreateMergePatch(originalData, modifiedData)
	if err != nil {
		return nil, fmt.Errorf("CreateMergePatch failed: %w", err)
	}
	return patchBytes, nil
}
