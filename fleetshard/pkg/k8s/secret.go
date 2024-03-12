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
	CentralTLSSecretName              = "central-tls"                              // pragma: allowlist secret
	centralDBPasswordSecretName       = "central-db-password"                      // pragma: allowlist secret
	centralEncryptionKeySecretName    = "central-encryption-key"                   // pragma: allowlist secret
	manualDeclarativeConfigSecretName = "cloud-service-manual-declarative-configs" // pragma: allowlist secret
)

var defaultSecretsToWatch = map[string]bool{
	CentralTLSSecretName:              true,
	centralEncryptionKeySecretName:    true,
	manualDeclarativeConfigSecretName: false,
}

// SecretBackup is responsible for reading secrets to Backup for a tenant.
type SecretBackup struct {
	client         ctrlClient.Client
	secretsToWatch map[string]bool
}

// NewSecretBackup creates a new instance of SecretService.
func NewSecretBackup(client ctrlClient.Client, managedDB bool) *SecretBackup {
	secretsToWatch := defaultSecretsToWatch // pragma: allowlist secret
	if managedDB {
		secretsToWatch[centralDBPasswordSecretName] = true
	}

	return &SecretBackup{client: client, secretsToWatch: secretsToWatch} // pragma: allowlist secret
}

// GetWatchedSecrets return a sorted list of secrets watched by this package
func (s *SecretBackup) GetWatchedSecrets() []string {
	secrets := make([]string, 0, len(s.secretsToWatch))
	for secretName := range s.secretsToWatch {
		secrets = append(secrets, secretName)
	}
	sort.Strings(secrets)
	return secrets
}

// CollectSecrets returns a map of secret name to secret object for all secrets
// watched by SecretServices
func (s *SecretBackup) CollectSecrets(ctx context.Context, namespace string) (map[string]*corev1.Secret, error) {
	secrets := map[string]*corev1.Secret{}
	for secretName, required := range s.secretsToWatch { // pragma: allowlist secret
		secret, found, err := getSecret(ctx, s.client, required, secretName, namespace)
		if err != nil {
			return nil, err
		}
		if found {
			secrets[secretName] = secret // pragma: allowlist secret
		}
	}

	return secrets, nil
}

func getSecret(ctx context.Context, client ctrlClient.Client, required bool, secretname, namespace string) (*corev1.Secret, bool, error) {
	centralSecret := &corev1.Secret{}
	err := client.Get(ctx, ctrlClient.ObjectKey{Namespace: namespace, Name: secretname}, centralSecret)
	if err != nil {
		if apiErrors.IsNotFound(err) {
			if required {
				return centralSecret, false, fmt.Errorf("%s secret not found", secretname)
			}
			return centralSecret, false, nil
		}
		return centralSecret, false, fmt.Errorf("getting secret %s/%s: %w", namespace, secretname, err)
	}

	return centralSecret, true, nil
}
