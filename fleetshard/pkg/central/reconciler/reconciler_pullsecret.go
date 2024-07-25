package reconciler

import (
	"bytes"
	"context"
	"fmt"
	corev1 "k8s.io/api/core/v1"
	apiErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrlClient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	tenantImagePullSecretName = "stackrox" // pragma: allowlist secret
)

type pullSecretReconciler struct {
	namespace        string
	dockerConfigJson []byte
	clientSet        ctrlClient.Client
}

func newPullSecretReconciler(clientSet ctrlClient.Client, namespace string, dockerConfigJson []byte) reconciler {
	return &pullSecretReconciler{
		namespace:        namespace,
		dockerConfigJson: dockerConfigJson,
		clientSet:        clientSet,
	}
}

var _ reconciler = &pullSecretReconciler{}

func (p pullSecretReconciler) ensurePresent(ctx context.Context) (context.Context, error) {
	if len(p.dockerConfigJson) == 0 {
		return ctx, p.deletePullSecret(ctx)
	}
	var existing corev1.Secret
	err := p.clientSet.Get(ctx, p.getObjectKey(), &existing)
	if err != nil {
		if apiErrors.IsNotFound(err) {
			return ctx, p.createPullSecret(ctx)
		}
		return ctx, fmt.Errorf("failed to get image pull secret %v: %w", p.getObjectKey(), err)
	}
	return ctx, p.updatePullSecret(ctx, existing)
}

func (p pullSecretReconciler) ensureAbsent(ctx context.Context) (context.Context, error) {
	return ctx, p.deletePullSecret(ctx)
}

func (p pullSecretReconciler) deletePullSecret(ctx context.Context) error {
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: p.namespace,
			Name:      tenantImagePullSecretName,
		},
	}
	err := p.clientSet.Delete(ctx, secret)
	if err != nil && !apiErrors.IsNotFound(err) {
		return fmt.Errorf("failed to delete image pull secret %s/%s: %w", p.namespace, tenantImagePullSecretName, err)
	}
	return nil
}

func (p pullSecretReconciler) createPullSecret(ctx context.Context) error {
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: p.namespace,
			Name:      tenantImagePullSecretName,
		},
		Type: "kubernetes.io/dockerconfigjson",
		Data: map[string][]byte{
			".dockerconfigjson": p.dockerConfigJson,
		},
	}
	err := p.clientSet.Create(ctx, secret, ctrlClient.FieldOwner(fieldManager))
	if err != nil {
		return fmt.Errorf("failed to create image pull secret %v: %w", p.getObjectKey(), err)
	}
	return nil
}

func (p pullSecretReconciler) updatePullSecret(ctx context.Context, existing corev1.Secret) error {
	if bytes.Equal(existing.Data[".dockerconfigjson"], p.dockerConfigJson) {
		return nil
	}
	existing.Data[".dockerconfigjson"] = p.dockerConfigJson
	err := p.clientSet.Update(ctx, &existing, ctrlClient.FieldOwner(fieldManager))
	if err != nil {
		return fmt.Errorf("failed to update image pull secret %v: %w", p.getObjectKey(), err)
	}
	return nil
}

func (p pullSecretReconciler) getObjectKey() ctrlClient.ObjectKey {
	return ctrlClient.ObjectKey{Name: tenantImagePullSecretName, Namespace: p.namespace}
}
