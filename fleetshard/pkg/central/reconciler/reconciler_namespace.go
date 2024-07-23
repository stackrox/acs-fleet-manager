package reconciler

import (
	"context"
	"fmt"
	"github.com/golang/glog"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/api/private"
	corev1 "k8s.io/api/core/v1"
	apiErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"time"
)

// namespaceReconciler is a reconciler that ensures that the namespace for a managed central is present and up-to-date.
type namespaceReconciler struct {
	client kubernetes.Interface
}

var namespaceDeletionTimeout = 30 * time.Minute
var namespaceDeletionPollInterval = 5 * time.Second

func newNamespaceReconciler(client kubernetes.Interface) reconciler {
	return &namespaceReconciler{
		client: client,
	}
}

var _ reconciler = &namespaceReconciler{}

func (n namespaceReconciler) ensurePresent(ctx context.Context) (context.Context, error) {
	central, ok := managedCentralFromContext(ctx)
	if !ok {
		return ctx, fmt.Errorf("context does not contain a managed central")
	}
	desiredNamespace := n.makeNamespace(central)

	existingNamespace, err := n.client.CoreV1().Namespaces().Get(ctx, desiredNamespace.Name, metav1.GetOptions{})
	if err != nil {
		if apiErrors.IsNotFound(err) {
			glog.Infof("creating namespace %q", desiredNamespace.Name)
			if _, err := n.client.CoreV1().Namespaces().Create(ctx, desiredNamespace, metav1.CreateOptions{
				FieldManager: fieldManager,
			}); err != nil {
				return ctx, fmt.Errorf("failed to create namespace %q: %w", desiredNamespace.Name, err)
			}
			return ctx, nil
		}
		return ctx, fmt.Errorf("failed getting namespace %q: %w", desiredNamespace.Name, err)
	}

	if existingNamespace.DeletionTimestamp != nil {
		return ctx, fmt.Errorf("namespace %q is being deleted", desiredNamespace.Name)
	}

	if stringMapNeedsUpdating(desiredNamespace.Annotations, existingNamespace.Annotations) || stringMapNeedsUpdating(desiredNamespace.Labels, existingNamespace.Labels) {
		glog.Infof("updating namespace %q", desiredNamespace.Name)
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
		if _, err = n.client.CoreV1().Namespaces().Update(ctx, existingNamespace, metav1.UpdateOptions{
			FieldManager: fieldManager,
		}); err != nil {
			return ctx, fmt.Errorf("failed to update namespace %q: %w", desiredNamespace.Name, err)
		}
	}
	return ctx, nil
}

func (n namespaceReconciler) ensureAbsent(ctx context.Context) (context.Context, error) {
	central, ok := managedCentralFromContext(ctx)
	if !ok {
		return ctx, fmt.Errorf("context does not contain a managed central")
	}
	namespaceName := central.Metadata.Namespace

	ctx, cancel := context.WithTimeout(ctx, namespaceDeletionTimeout)
	defer cancel()

	ticker := time.NewTicker(namespaceDeletionPollInterval)
	start := time.Now()

	for {
		select {
		case <-ctx.Done():
			return ctx, fmt.Errorf("%v timeout reached while deleting namespace %q", namespaceDeletionTimeout, namespaceName)
		case <-ticker.C:

			namespace, err := n.client.CoreV1().Namespaces().Get(ctx, namespaceName, metav1.GetOptions{})
			if err != nil {
				if apiErrors.IsNotFound(err) {
					glog.Infof("namespace %q was successfully deleted after %v", namespaceName, time.Since(start))
					return ctx, nil
				}
				return ctx, fmt.Errorf("failed to delete namespace %q: %w", namespaceName, err)
			}

			if namespace.Status.Phase == corev1.NamespaceTerminating {
				glog.Infof("namespace %q is still terminating after %v", namespaceName, time.Since(start))
				continue
			}

			glog.Infof("deleting namespace %q", namespaceName)
			if err := n.client.CoreV1().Namespaces().Delete(ctx, namespaceName, metav1.DeleteOptions{}); err != nil {
				if apiErrors.IsNotFound(err) {
					glog.Infof("namespace %q was successfully deleted after %v", namespaceName, time.Since(start))
					return ctx, nil
				}
				return ctx, fmt.Errorf("failed to delete namespace %q: %w", namespaceName, err)
			}

			glog.Infof("namespace %s was marked for deletion", namespaceName)
		}
	}
}

func (n namespaceReconciler) makeNamespace(central private.ManagedCentral) *corev1.Namespace {
	return &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name:        central.Metadata.Namespace,
			Annotations: n.getNamespaceAnnotations(central),
			Labels:      n.getNamespaceLabels(central),
		},
	}
}

func (n namespaceReconciler) getNamespaceLabels(c private.ManagedCentral) map[string]string {
	return map[string]string{
		managedByLabelKey:    labelManagedByFleetshardValue,
		instanceLabelKey:     c.Metadata.Name,
		orgIDLabelKey:        c.Spec.Auth.OwnerOrgId,
		tenantIDLabelKey:     c.Id,
		instanceTypeLabelKey: c.Spec.InstanceType,
	}
}

func (n namespaceReconciler) getNamespaceAnnotations(c private.ManagedCentral) map[string]string {
	namespaceAnnotations := map[string]string{
		orgNameAnnotationKey: c.Spec.Auth.OwnerOrgName,
	}
	if c.Metadata.ExpiredAt != nil {
		namespaceAnnotations[centralExpiredAtKey] = c.Metadata.ExpiredAt.Format(time.RFC3339)
	}
	namespaceAnnotations[ovnACLLoggingAnnotationKey] = ovnACLLoggingAnnotationDefault
	return namespaceAnnotations
}
