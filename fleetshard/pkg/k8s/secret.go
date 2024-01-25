package k8s

import (
	"context"
	"fmt"
	"sort"

	corev1 "k8s.io/api/core/v1"
	apiErrors "k8s.io/apimachinery/pkg/api/errors"
	ctrlClient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	CentralTLSSecretName           = "central-tls"            // pragma: allowlist secret
	centralDBPasswordSecretName    = "central-db-password"    // pragma: allowlist secret
	centralEncryptionKeySecretName = "central-encryption-key" // pragma: allowlist secret
)

var defaultSecretsToWatch = []string{
	CentralTLSSecretName,
	centralEncryptionKeySecretName,
}

// SecretBackup is responsible for reading secrets to Backup for a tenant.
type SecretBackup struct {
	client         ctrlClient.Client
	secretsToWatch []string
}

// NewSecretBackup creates a new instance of SecretService.
func NewSecretBackup(client ctrlClient.Client, managedDB bool) *SecretBackup {
	secretsToWatch := defaultSecretsToWatch // pragma: allowlist secret
	if managedDB {
		secretsToWatch = append(secretsToWatch, centralDBPasswordSecretName)
	}

	return &SecretBackup{client: client, secretsToWatch: secretsToWatch} // pragma: allowlist secret
}

// GetWatchedSecrets return a sorted list of secrets watched by this package
func (s *SecretBackup) GetWatchedSecrets() []string {
	secrets := make([]string, len(s.secretsToWatch))
	copy(secrets, s.secretsToWatch)
	sort.Strings(secrets)
	return secrets
}

// CollectSecrets returns a map of secret name to secret object for all secrets
// watched by SecretServices
func (s *SecretBackup) CollectSecrets(ctx context.Context, namespace string) (map[string]*corev1.Secret, error) {
	secrets := map[string]*corev1.Secret{}
	for _, secretname := range s.secretsToWatch { // pragma: allowlist secret
		secret, err := getSecret(ctx, s.client, secretname, namespace)
		if err != nil {
			return nil, err
		}
		secrets[secretname] = secret // pragma: allowlist secret
	}

	return secrets, nil
}

func getSecret(ctx context.Context, client ctrlClient.Client, secretname, namespace string) (*corev1.Secret, error) {
	centralSecret := &corev1.Secret{}
	err := client.Get(ctx, ctrlClient.ObjectKey{Namespace: namespace, Name: secretname}, centralSecret)
	if err != nil {
		if apiErrors.IsNotFound(err) {
			return centralSecret, fmt.Errorf("%s secret not found", secretname)
		}
		return centralSecret, fmt.Errorf("getting secret %s/%s: %w", namespace, secretname, err)
	}

	return centralSecret, nil
}
