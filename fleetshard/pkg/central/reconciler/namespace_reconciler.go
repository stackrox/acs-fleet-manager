package reconciler

import (
	"context"
	"fmt"

	"github.com/golang/glog"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	apiErrors "k8s.io/apimachinery/pkg/api/errors"
	ctrlClient "sigs.k8s.io/controller-runtime/pkg/client"
)

// namespaceReconciler provides methods to reconcile the namespace required for central tenants
type namespaceReconciler struct {
	// client is the controller runtime client used for namespace reconciliation
	client ctrlClient.Client
}

// newNamespaceReconciler creates a NamespaceReconciler with given arguments
func newNamespaceReconciler(client ctrlClient.Client) *namespaceReconciler {
	return &namespaceReconciler{client: client}
}

// ensureDeleted sends a delete request for K8s namespace with given name
// it return true if the namespace doesn't exist anymore.
// It will not send additional delete requests if the NS is already in state terminating
func (r *namespaceReconciler) ensureDeleted(ctx context.Context, name string) (bool, error) {
	namespace, err := r.getNamespaceObj(name)
	if err != nil {
		if apiErrors.IsNotFound(err) {
			return true, nil
		}
		return false, errors.Wrapf(err, "delete central namespace %s", name)
	}
	if namespace.Status.Phase == corev1.NamespaceTerminating {
		return false, nil // Deletion is already in progress, skipping deletion request
	}
	if err = r.client.Delete(ctx, namespace); err != nil {
		return false, errors.Wrapf(err, "delete central namespace %s", name)
	}
	glog.Infof("Central namespace %s is marked for deletion", name)
	return false, nil
}

// reconcile reconciles the given namespace in cluster to fit to the given desired namespace
func (r *namespaceReconciler) reconcile(ctx context.Context, desiredNamespace *corev1.Namespace) error {
	existingNamespace, err := r.getNamespaceObj(desiredNamespace.Name)
	if err != nil {
		if apiErrors.IsNotFound(err) {
			if err := r.client.Create(ctx, desiredNamespace); err != nil {
				return fmt.Errorf("creating namespace %q: %w", desiredNamespace.Name, err)
			}
			return nil
		}
		return fmt.Errorf("getting namespace %q: %w", desiredNamespace.Name, err)
	}

	if stringMapNeedsUpdating(desiredNamespace.Annotations, existingNamespace.Annotations) || stringMapNeedsUpdating(desiredNamespace.Labels, existingNamespace.Labels) {
		glog.Infof("Updating namespace %q", desiredNamespace.Name)
		if existingNamespace.Annotations == nil {
			existingNamespace.Annotations = map[string]string{}
		}
		for k, v := range desiredNamespace.Annotations {
			existingNamespace.Annotations[k] = v
		}
		if existingNamespace.Labels == nil {
			existingNamespace.Labels = map[string]string{}
		}
		for k, v := range desiredNamespace.Labels {
			existingNamespace.Labels[k] = v
		}
		if err = r.client.Update(ctx, existingNamespace, &ctrlClient.UpdateOptions{
			FieldManager: "fleetshard-sync",
		}); err != nil {
			return fmt.Errorf("updating namespace %q: %w", desiredNamespace.Name, err)
		}
	}

	return nil
}

// getNamespaceObj gets the *corev1.Namespace object for namespace with given name from the cluster
func (r *namespaceReconciler) getNamespaceObj(name string) (*corev1.Namespace, error) {
	var namespace corev1.Namespace
	if err := r.client.Get(context.Background(), ctrlClient.ObjectKey{Name: name}, &namespace); err != nil {
		return nil, fmt.Errorf("getting namespace %q: %w", name, err)
	}
	return &namespace, nil
}
