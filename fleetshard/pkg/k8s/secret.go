package k8s

import (
	"context"
	"fmt"
	"sort"

	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	apiErrors "k8s.io/apimachinery/pkg/api/errors"
	ctrlClient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	centralTLSSecretName        = "central-tls"         // pragma: allowlist secret
	centralDBPasswordSecretName = "central-db-password" // pragma: allowlist secret
)

var secretsToWatch = []string{
	centralTLSSecretName,
	centralDBPasswordSecretName,
}

// SecretService is responsible for reading and writing secrets
// This servise is specific to ACS Managed Services and provides methods work on specific secrets
type SecretService struct {
	client ctrlClient.Client
}

// NewSecretService creates a new instance of SecretService.
func NewSecretService(client ctrlClient.Client) *SecretService {
	return &SecretService{client: client}
}

// GetWatchedSecrets return a sorted list of secrets watched by this package
func GetWatchedSecrets() []string {
	secrets := make([]string, len(secretsToWatch))
	copy(secrets, secretsToWatch)
	sort.Strings(secrets)
	return secrets
}

// CollectSecrets return a map of secret name to secret object for all secrets
// watched by SecretServices
func (s *SecretService) CollectSecrets(ctx context.Context, namespace string) (map[string]*corev1.Secret, error) {
	secrets := map[string]*corev1.Secret{}
	for _, secretname := range secretsToWatch { // pragma: allowlist secret
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
		return centralSecret, errors.Wrapf(err, "getting secret %s/%s", namespace, secretname)
	}

	return centralSecret, nil
}
