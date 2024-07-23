package reconciler

import (
	"context"
	"fmt"
	"github.com/golang/glog"
	"github.com/pkg/errors"
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

var errNamespaceTerminating = errors.New("namespace is being deleted")
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
			if _, err := n.client.CoreV1().Namespaces().Create(ctx, desiredNamespace, metav1.CreateOptions{
				FieldManager: fieldManager,
			}); err != nil {
				return ctx, fmt.Errorf("creating namespace %q: %w", desiredNamespace.Name, err)
			}
			return ctx, nil
		}
		return ctx, fmt.Errorf("getting namespace %q: %w", desiredNamespace.Name, err)
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
		if _, err = n.client.CoreV1().Namespaces().Update(ctx, existingNamespace, metav1.UpdateOptions{
			FieldManager: fieldManager,
		}); err != nil {
			return ctx, fmt.Errorf("updating namespace %q: %w", desiredNamespace.Name, err)
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

	for {
		select {
		case <-ctx.Done():
			return ctx, fmt.Errorf("timeout deleting central namespace %s", namespaceName)
		case <-ticker.C:
			namespace, err := n.client.CoreV1().Namespaces().Get(ctx, namespaceName, metav1.GetOptions{})
			if err != nil {
				if apiErrors.IsNotFound(err) {
					return ctx, nil
				}
				return ctx, errors.Wrapf(err, "deleting central namespace %s", namespaceName)
			}
			if namespace.Status.Phase == corev1.NamespaceTerminating {
				continue
			}
			if err := n.client.CoreV1().Namespaces().Delete(ctx, namespaceName, metav1.DeleteOptions{}); err != nil {
				return ctx, errors.Wrapf(err, "delete central namespace %s", namespaceName)
			}
			glog.Infof("Central namespace %s is marked for deletion", namespaceName)
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
