// Package cluster provides access to cluster resources not related with centrals
package cluster

import (
	"context"
	"fmt"

	addonsV1alpha1 "github.com/openshift/addon-operator/apis/addons/v1alpha1"

	"github.com/stackrox/acs-fleet-manager/pkg/shared"
	corev1 "k8s.io/api/core/v1"
	ctrlClient "sigs.k8s.io/controller-runtime/pkg/client"
)

// AddonService allows to access addons installed on a cluster
type AddonService interface {
	// GetAddon returns the addon with the specified ID or error in case service fails to find such an addon
	GetAddon(ctx context.Context, id string) (shared.Addon, error)
}

var (
	_ AddonService = (*addonService)(nil)
)

type addonService struct {
	k8sClient ctrlClient.Client
}

// NewAddonService return a new instance of AddonService
func NewAddonService(k8sClient ctrlClient.Client) AddonService {
	return &addonService{
		k8sClient: k8sClient,
	}
}

func (s *addonService) GetAddon(ctx context.Context, id string) (shared.Addon, error) {
	addonObj := addonsV1alpha1.Addon{}
	addon := shared.Addon{}
	if err := s.k8sClient.Get(ctx, ctrlClient.ObjectKey{Name: id}, &addonObj); err != nil {
		return addon, fmt.Errorf("get addon %s: %w", id, err)
	}

	addon.ID = addonObj.Name
	addon.Version = addonObj.Spec.Version

	if addonObj.Spec.Install.OLMOwnNamespace == nil {
		return addon, fmt.Errorf("addon with install mode other than OLMOwnNamespace is not supported")
	}
	addon.SourceImage = addonObj.Spec.Install.OLMOwnNamespace.CatalogSourceImage

	if addonObj.Spec.AddonPackageOperator == nil {
		return addon, fmt.Errorf("addon without AddonPackageOperator is not supported")
	}
	addon.PackageImage = addonObj.Spec.AddonPackageOperator.Image

	secret, err := s.getAddonParametersSecret(ctx, addonObj.Spec.Install.OLMOwnNamespace.Namespace, id)
	if err != nil {
		return addon, err
	}
	parameters := make(map[string]string, len(secret.Data))
	for k, v := range secret.Data {
		parameters[k] = string(v[:])
	}
	addon.Parameters = parameters

	return addon, nil
}

func (s *addonService) getAddonParametersSecret(ctx context.Context, namespace string, addonID string) (corev1.Secret, error) {
	secretName := fmt.Sprintf("addon-%s-parameters", addonID)
	secret := corev1.Secret{} // pragma: allowlist secret
	err := s.k8sClient.Get(ctx, ctrlClient.ObjectKey{Namespace: namespace, Name: secretName}, &secret)

	if err != nil {
		return secret, fmt.Errorf("get addon parameters secret %s: %w", secretName, err)
	}

	return secret, nil
}
