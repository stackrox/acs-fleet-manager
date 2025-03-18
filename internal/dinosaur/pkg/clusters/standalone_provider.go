package clusters

import (
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/clusters/types"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/config"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/wellknown"

	"github.com/stackrox/acs-fleet-manager/pkg/api"
	"github.com/stackrox/acs-fleet-manager/pkg/db"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// dinosaurSREOpenIDPSecretName is the secret name holding the clientSecret content
const dinosaurSREOpenIDPSecretName = "dinosaur-sre-idp-secret" // pragma: allowlist secret

// StandaloneProvider ...
type StandaloneProvider struct {
	connectionFactory      *db.ConnectionFactory
	dataplaneClusterConfig *config.DataplaneClusterConfig
}

var _ Provider = &StandaloneProvider{}

func newStandaloneProvider(connectionFactory *db.ConnectionFactory, dataplaneClusterConfig *config.DataplaneClusterConfig) *StandaloneProvider {
	return &StandaloneProvider{
		connectionFactory:      connectionFactory,
		dataplaneClusterConfig: dataplaneClusterConfig,
	}
}

// Create ...
func (s *StandaloneProvider) Create(request *types.ClusterRequest) (*types.ClusterSpec, error) {
	return nil, nil
}

// Delete ...
func (s *StandaloneProvider) Delete(spec *types.ClusterSpec) (bool, error) {
	return true, nil
}

// CheckClusterStatus ...
func (s *StandaloneProvider) CheckClusterStatus(spec *types.ClusterSpec) (*types.ClusterSpec, error) {
	spec.Status = api.ClusterProvisioned
	return spec, nil
}

// GetClusterDNS ...
func (s *StandaloneProvider) GetClusterDNS(clusterSpec *types.ClusterSpec) (string, error) {
	return "", nil // NOOP for now
}

// buildOpenIDPClientSecret builds the k8s secret which holds OpenIDP clientSecret value
// The clientSecret as indicated in https://docs.openshift.com/container-platform/4.7/authentication/identity_providers/configuring-oidc-identity-provider.html#identity-provider-creating-secret_configuring-oidc-identity-provider
func (s *StandaloneProvider) buildOpenIDPClientSecret(identityProvider types.IdentityProviderInfo) *v1.Secret {
	return &v1.Secret{
		TypeMeta: metav1.TypeMeta{
			APIVersion: metav1.SchemeGroupVersion.Version,
			Kind:       "Secret",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      dinosaurSREOpenIDPSecretName,
			Namespace: "openshift-config",
		},
		Type: v1.SecretTypeOpaque,
		StringData: map[string]string{
			"clientSecret": identityProvider.OpenID.ClientSecret, // pragma: allowlist secret
		},
	}
}

// buildIdentityProviderResource builds the identity provider resource to be applied
// The resource is taken from https://docs.openshift.com/container-platform/4.7/authentication/identity_providers/configuring-oidc-identity-provider.html#identity-provider-oidc-CR_configuring-oidc-identity-provider
func (s *StandaloneProvider) buildIdentityProviderResource(identityProvider types.IdentityProviderInfo) map[string]interface{} {
	// Using unstructured type for now.
	// we might want to pull the type information from github.com/openshift/api at a later stage
	return map[string]interface{}{
		"apiVersion": "config.openshift.io/v1",
		"kind":       "OAuth",
		"metadata": map[string]string{
			"name": "cluster",
		},
		"spec": map[string]interface{}{
			"identityProviders": []map[string]interface{}{
				{
					"name":          identityProvider.OpenID.Name,
					"mappingMethod": "claim",
					"type":          "OpenID",
					"openID": map[string]interface{}{
						"clientID": identityProvider.OpenID.ClientID,
						"issuer":   identityProvider.OpenID.Issuer,
						"clientSecret": map[string]string{
							"name": dinosaurSREOpenIDPSecretName,
						},
						"claims": map[string][]string{
							"email":             {"email"},
							"preferredUsername": {"preferred_username"},
							"last_name":         {"preferred_username"},
						},
					},
				},
			},
		},
	}
}

// ScaleUp ...
func (s *StandaloneProvider) ScaleUp(clusterSpec *types.ClusterSpec, increment int) (*types.ClusterSpec, error) {
	return clusterSpec, nil // NOOP
}

// ScaleDown ...
func (s *StandaloneProvider) ScaleDown(clusterSpec *types.ClusterSpec, decrement int) (*types.ClusterSpec, error) {
	return clusterSpec, nil // NOOP
}

// SetComputeNodes ...
func (s *StandaloneProvider) SetComputeNodes(clusterSpec *types.ClusterSpec, numNodes int) (*types.ClusterSpec, error) {
	return clusterSpec, nil // NOOP
}

// GetComputeNodes ...
func (s *StandaloneProvider) GetComputeNodes(spec *types.ClusterSpec) (*types.ComputeNodesInfo, error) {
	return &types.ComputeNodesInfo{}, nil // NOOP
}

// GetCloudProviders ...
func (s *StandaloneProvider) GetCloudProviders() (*types.CloudProviderInfoList, error) {
	type Cluster struct {
		CloudProvider string
	}
	dbConn := s.connectionFactory.New().
		Model(&Cluster{}).
		Distinct("cloud_provider").
		Where("provider_type = ?", api.ClusterProviderStandalone.String()).
		Where("status NOT IN (?)", api.ClusterDeletionStatuses)

	var results []Cluster
	err := dbConn.Find(&results).Error
	if err != nil {
		return nil, err
	}

	items := []types.CloudProviderInfo{}
	for _, result := range results {
		items = append(items, types.CloudProviderInfo{
			ID:          result.CloudProvider,
			Name:        result.CloudProvider,
			DisplayName: result.CloudProvider,
		})
	}

	return &types.CloudProviderInfoList{Items: items}, nil
}

// GetCloudProviderRegions ...
func (s *StandaloneProvider) GetCloudProviderRegions(providerInf types.CloudProviderInfo) (*types.CloudProviderRegionInfoList, error) {
	type Cluster struct {
		Region  string
		MultiAZ bool
	}
	dbConn := s.connectionFactory.New().
		Model(&Cluster{}).
		Distinct("region", "multi_az").
		Where("cloud_provider = ?", providerInf.ID).
		Where("provider_type = ?", api.ClusterProviderStandalone.String()).
		Where("status NOT IN (?)", api.ClusterDeletionStatuses)

	var results []Cluster
	err := dbConn.Find(&results).Error
	if err != nil {
		return nil, err
	}

	var items = make([]types.CloudProviderRegionInfo, len(results))
	for i, result := range results {
		items[i] = types.CloudProviderRegionInfo{
			ID:              result.Region,
			Name:            result.Region,
			DisplayName:     wellknown.GetCloudRegionDisplayName(providerInf.ID, result.Region),
			SupportsMultiAZ: result.MultiAZ,
			CloudProviderID: providerInf.ID,
		}
	}

	return &types.CloudProviderRegionInfoList{Items: items}, nil
}
