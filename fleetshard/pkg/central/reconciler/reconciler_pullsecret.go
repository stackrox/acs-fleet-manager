package reconciler

import (
	"bytes"
	"context"
	"fmt"
	corev1 "k8s.io/api/core/v1"
	apiErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

const (
	tenantImagePullSecretName = "stackrox" // pragma: allowlist secret
)

type pullSecretReconciler struct {
	namespace        string
	dockerConfigJson []byte
	clientSet        kubernetes.Interface
}

func newPullSecretReconciler(clientSet kubernetes.Interface, namespace string, dockerConfigJson []byte) reconciler {
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
	existing, err := p.clientSet.CoreV1().Secrets(p.namespace).Get(ctx, tenantImagePullSecretName, metav1.GetOptions{})
	if err != nil {
		if apiErrors.IsNotFound(err) {
			return ctx, p.createPullSecret(ctx)
		}
		return ctx, fmt.Errorf("failed to get image pull secret %s/%s: %w", p.namespace, tenantImagePullSecretName, err)
	}
	return ctx, p.updatePullSecret(ctx, existing)
}

func (p pullSecretReconciler) ensureAbsent(ctx context.Context) (context.Context, error) {
	return ctx, p.deletePullSecret(ctx)
}

func (p pullSecretReconciler) deletePullSecret(ctx context.Context) error {
	err := p.clientSet.CoreV1().Secrets(p.namespace).Delete(ctx, tenantImagePullSecretName, metav1.DeleteOptions{})
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
	_, err := p.clientSet.CoreV1().Secrets(p.namespace).Create(ctx, secret, metav1.CreateOptions{
		FieldManager: fieldManager,
	})
	if err != nil {
		return fmt.Errorf("failed to create image pull secret %s/%s: %w", p.namespace, tenantImagePullSecretName, err)
	}
	return nil
}

func (p pullSecretReconciler) updatePullSecret(ctx context.Context, existing *corev1.Secret) error {
	if bytes.Equal(existing.Data[".dockerconfigjson"], p.dockerConfigJson) {
		return nil
	}
	existing.Data[".dockerconfigjson"] = p.dockerConfigJson
	_, err := p.clientSet.CoreV1().Secrets(p.namespace).Update(ctx, existing, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("failed to update image pull secret %s/%s: %w", p.namespace, tenantImagePullSecretName, err)
	}
	return nil
}
