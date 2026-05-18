package reconciler

import (
	"context"
	"fmt"

	"github.com/stackrox/acs-fleet-manager/fleetshard/pkg/k8s"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	appsv1 "k8s.io/api/apps/v1"
	apiErrors "k8s.io/apimachinery/pkg/api/errors"

	"github.com/pkg/errors"
	core "k8s.io/api/core/v1"
	ctrlClient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	centralDeploymentName = "central"
)

func isCentralDeploymentReady(ctx context.Context, client ctrlClient.Client, namespace string) (bool, error) {
	deployment := &appsv1.Deployment{}
	err := client.Get(ctx,
		ctrlClient.ObjectKey{Name: centralDeploymentName, Namespace: namespace},
		deployment)
	if err != nil {
		if apiErrors.IsNotFound(err) {
			return false, nil
		}
		return false, errors.Wrap(err, "retrieving central deployment resource from Kubernetes")
	}
	return deployment.Status.AvailableReplicas > 0 && deployment.Status.UnavailableReplicas == 0, nil
}

func checkSecretExists(
	ctx context.Context,
	client ctrlClient.Client,
	remoteCentralNamespace string,
	secretName string,
) (bool, error) {
	secret := &core.Secret{}
	err := client.Get(ctx, ctrlClient.ObjectKey{Namespace: remoteCentralNamespace, Name: secretName}, secret)
	if err != nil {
		if apiErrors.IsNotFound(err) {
			return false, nil
		}

		return false, fmt.Errorf("getting secret %s/%s: %w", remoteCentralNamespace, secretName, err)
	}

	return true, nil
}

func ensureSecretExists(
	ctx context.Context,
	client ctrlClient.Client,
	namespace string,
	secretName string,
	secretModifyFunc func(secret *core.Secret) error,
) error {
	secret := &core.Secret{}
	secretKey := ctrlClient.ObjectKey{Name: secretName, Namespace: namespace} // pragma: allowlist secret

	err := client.Get(ctx, secretKey, secret) // pragma: allowlist secret
	if err != nil && !apiErrors.IsNotFound(err) {
		return fmt.Errorf("getting %s/%s secret: %w", namespace, secretName, err)
	}
	if err == nil {
		modificationErr := secretModifyFunc(secret)
		if modificationErr != nil {
			return fmt.Errorf("updating %s/%s secret content: %w", namespace, secretName, modificationErr)
		}
		if updateErr := client.Update(ctx, secret); updateErr != nil { // pragma: allowlist secret
			return fmt.Errorf("updating %s/%s secret: %w", namespace, secretName, updateErr)
		}

		return nil
	}

	// Create secret if it does not exist.
	secret = &core.Secret{
		ObjectMeta: metav1.ObjectMeta{ // pragma: allowlist secret
			Name:      secretName,
			Namespace: namespace,
			Labels:    map[string]string{k8s.ManagedByLabelKey: k8s.ManagedByFleetshardValue},
			Annotations: map[string]string{
				managedServicesAnnotation: "true",
			},
		},
	}

	if modificationErr := secretModifyFunc(secret); modificationErr != nil {
		return fmt.Errorf("initializing %s/%s secret payload: %w", namespace, secretName, modificationErr)
	}
	if createErr := client.Create(ctx, secret); createErr != nil { // pragma: allowlist secret
		return fmt.Errorf("creating %s/%s secret: %w", namespace, secretName, createErr)
	}
	return nil
}
