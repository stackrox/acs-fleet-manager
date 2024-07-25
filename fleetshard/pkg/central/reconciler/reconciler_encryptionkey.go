package reconciler

import (
	"context"
	"encoding/base64"
	"fmt"
	"github.com/stackrox/acs-fleet-manager/fleetshard/pkg/cipher"
	"github.com/stackrox/acs-fleet-manager/fleetshard/pkg/k8s"
	centralNotifierUtils "github.com/stackrox/rox/central/notifiers/utils"
	yaml "gopkg.in/yaml.v2"
	corev1 "k8s.io/api/core/v1"
	apiErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrlClient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	centralEncryptionKeySecretName = "central-encryption-key-chain" // pragma: allowlist secret
)

type encryptionKeyReconciler struct {
	encryptionKeyGenerator cipher.KeyGenerator
	client                 ctrlClient.Client
}

func newEncryptionKeyReconciler(client ctrlClient.Client, encryptionKeyGenerator cipher.KeyGenerator) reconciler {
	return &encryptionKeyReconciler{
		client:                 client,
		encryptionKeyGenerator: encryptionKeyGenerator,
	}
}

var _ reconciler = &encryptionKeyReconciler{}

func (e encryptionKeyReconciler) ensurePresent(ctx context.Context) (context.Context, error) {

	central, ok := managedCentralFromContext(ctx)
	if !ok {
		return ctx, fmt.Errorf("context does not contain a managed central")
	}

	namespace := central.Metadata.Namespace
	secret := &corev1.Secret{}
	secretName := centralEncryptionKeySecretName                              // pragma: allowlist secret
	secretKey := ctrlClient.ObjectKey{Name: secretName, Namespace: namespace} // pragma: allowlist secret

	err := e.client.Get(ctx, secretKey, secret) // pragma: allowlist secret
	if err != nil && !apiErrors.IsNotFound(err) {
		return ctx, fmt.Errorf("failed getting secret %v: %w", secretKey, err)
	}

	if err == nil {
		modificationErr := e.populateEncryptionKeySecret(secret)
		if modificationErr != nil {
			return ctx, fmt.Errorf("failed to update secret %v: %w", secretKey, modificationErr)
		}
		if updateErr := e.client.Update(ctx, secret); updateErr != nil { // pragma: allowlist secret
			return ctx, fmt.Errorf("failed to update secret %v: %w", secretKey, updateErr)
		}
		return ctx, nil
	}

	// Create secret if it does not exist.
	secret = &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{ // pragma: allowlist secret
			Name:      secretName,
			Namespace: namespace,
			Labels:    map[string]string{k8s.ManagedByLabelKey: k8s.ManagedByFleetshardValue},
			Annotations: map[string]string{
				managedServicesAnnotation: "true",
			},
		},
	}

	if modificationErr := e.populateEncryptionKeySecret(secret); modificationErr != nil {
		return ctx, fmt.Errorf("initializing %s/%s secret payload: %w", namespace, secretName, modificationErr)
	}

	if createErr := e.client.Create(ctx, secret); createErr != nil { // pragma: allowlist secret
		return ctx, fmt.Errorf("creating %s/%s secret: %w", namespace, secretName, createErr)
	}

	return ctx, nil
}

func (e encryptionKeyReconciler) ensureAbsent(ctx context.Context) (context.Context, error) {
	return ctx, nil
}

func (e encryptionKeyReconciler) populateEncryptionKeySecret(secret *corev1.Secret) error {
	const encryptionKeyChainFile = "key-chain.yaml"

	if secret.Data != nil {
		if _, ok := secret.Data[encryptionKeyChainFile]; ok {
			// secret already populated with encryption key skip operation
			return nil
		}
	}

	encryptionKey, err := e.encryptionKeyGenerator.Generate()
	if err != nil {
		return fmt.Errorf("generating encryption key: %w", err)
	}

	b64Key := base64.StdEncoding.EncodeToString(encryptionKey)
	keyChainFile, err := e.generateNewKeyChainFile(b64Key)
	if err != nil {
		return err
	}
	secret.Data = map[string][]byte{encryptionKeyChainFile: keyChainFile}
	return nil
}

func (e encryptionKeyReconciler) generateNewKeyChainFile(b64Key string) ([]byte, error) {
	keyMap := make(map[int]string)
	keyMap[0] = b64Key

	keyChain := centralNotifierUtils.KeyChain{
		KeyMap:         keyMap,
		ActiveKeyIndex: 0,
	}

	yamlBytes, err := yaml.Marshal(keyChain) // pragma: allowlist secret
	if err != nil {
		return []byte{}, fmt.Errorf("generating key-chain file: %w", err)
	}

	return yamlBytes, nil
}
