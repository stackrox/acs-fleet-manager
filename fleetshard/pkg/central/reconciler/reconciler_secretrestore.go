package reconciler

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/golang/glog"
	"github.com/stackrox/acs-fleet-manager/fleetshard/pkg/cipher"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/api/private"
	corev1 "k8s.io/api/core/v1"
	apiErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"net/http"
)

type centralGetter interface {
	GetCentral(ctx context.Context, centralID string) (private.ManagedCentral, *http.Response, error)
}

type secretRestoreReconciler struct {
	client             kubernetes.Interface
	fleetManagerClient centralGetter
	secretCipher       cipher.Cipher
}

func newSecretRestoreReconciler(client kubernetes.Interface, fleetManagerClient centralGetter, secretCipher cipher.Cipher) reconciler {
	return &secretRestoreReconciler{
		client:             client,
		fleetManagerClient: fleetManagerClient,
		secretCipher:       secretCipher, // pragma: allowlist secret
	}
}

func (s secretRestoreReconciler) ensurePresent(ctx context.Context) (context.Context, error) {
	central, ok := managedCentralFromContext(ctx)
	if !ok {
		return ctx, fmt.Errorf("context does not contain a managed central")
	}

	centralID := central.Id
	namespace := central.Metadata.Namespace

	restoreSecrets := []string{}

	for _, secretName := range central.Metadata.SecretsStored { // pragma: allowlist secret
		_, err := s.client.CoreV1().Secrets(namespace).Get(ctx, secretName, metav1.GetOptions{})
		if err == nil {
			// secret already exists
			continue
		}
		if !apiErrors.IsNotFound(err) {
			return ctx, fmt.Errorf("failed to get secret %s/%s: %w", namespace, secretName, err)
		}
		restoreSecrets = append(restoreSecrets, secretName)
	}

	if len(restoreSecrets) == 0 {
		// nothing to restore
		return ctx, nil
	}

	glog.Info(fmt.Sprintf("Restoring secrets for tenant: %s/%s", centralID, namespace), restoreSecrets)
	centralWithSecrets, _, err := s.fleetManagerClient.GetCentral(ctx, centralID)
	if err != nil {
		return ctx, fmt.Errorf("failed to load secrets for central %s: %w", centralID, err)
	}

	decryptedSecrets, err := s.decryptSecrets(centralWithSecrets.Metadata.Secrets)
	if err != nil {
		return ctx, fmt.Errorf("failed to decrypt secrets for central %s: %w", centralID, err)
	}

	for _, secretName := range restoreSecrets { // pragma: allowlist secret
		secretToRestore, secretFound := decryptedSecrets[secretName]
		if !secretFound {
			return ctx, fmt.Errorf("failed to find secret %s in decrypted secret map", secretName)
		}

		_, err = s.client.CoreV1().Secrets(namespace).Create(ctx, secretToRestore, metav1.CreateOptions{})
		if err != nil {
			return ctx, fmt.Errorf("failed to recreate secret %s for central %s: %w", secretName, centralID, err)
		}

	}

	return ctx, nil
}

func (s secretRestoreReconciler) ensureAbsent(ctx context.Context) (context.Context, error) {
	return ctx, nil
}

func (s secretRestoreReconciler) decryptSecrets(secrets map[string]string) (map[string]*corev1.Secret, error) {
	decryptedSecrets := map[string]*corev1.Secret{}

	for secretName, ciphertext := range secrets {
		decodedCipher, err := base64.StdEncoding.DecodeString(ciphertext)
		if err != nil {
			return nil, fmt.Errorf("failed to decode secret %s: %w", secretName, err)
		}

		plaintextSecret, err := s.secretCipher.Decrypt(decodedCipher)
		if err != nil {
			return nil, fmt.Errorf("failed to decrypt secret %s: %w", secretName, err)
		}

		var secret corev1.Secret
		if err := json.Unmarshal(plaintextSecret, &secret); err != nil {
			return nil, fmt.Errorf("failed to unmarshal secret %s: %w", secretName, err)
		}

		decryptedSecrets[secretName] = &secret // pragma: allowlist secret
	}

	return decryptedSecrets, nil
}

var _ reconciler = &secretRestoreReconciler{}
