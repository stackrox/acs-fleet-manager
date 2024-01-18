package clusters

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/clusters/types"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/config"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/wellknown"

	"github.com/stackrox/acs-fleet-manager/pkg/api"
	"github.com/stackrox/acs-fleet-manager/pkg/db"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/restmapper"
)

// fieldManager indicates that the fleet-manager will be used as a field manager for conflict resolution
const fieldManager = "fleet-manager"

// lastAppliedConfigurationAnnotation is an annotation applied in a resources which tracks the last applied configuration of a resource.
// this is used to decide whether a new apply request should be taken into account or not
const lastAppliedConfigurationAnnotation = "fleet-manager/last-applied-resource-configuration"

// dinosaurSREOpenIDPSecretName is the secret name holding the clientSecret content
const dinosaurSREOpenIDPSecretName = "dinosaur-sre-idp-secret" // pragma: allowlist secret

var ctx = context.Background()

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

func applyResource(dynamicClient dynamic.Interface, mapper *restmapper.DeferredDiscoveryRESTMapper, resource interface{}) (runtime.Object, error) {
	// parse resource obj to unstructure.Unstructered
	data, err := json.Marshal(resource)
	if err != nil {
		return nil, fmt.Errorf("marshalling resource into JSON: %w", err)
	}

	var obj unstructured.Unstructured
	err = json.Unmarshal(data, &obj)
	if err != nil {
		return nil, fmt.Errorf("unmarshalling data from JSON: %w", err)
	}

	newConfiguration := string(data)
	newAnnotations := obj.GetAnnotations()
	if newAnnotations == nil {
		newAnnotations = map[string]string{}
		obj.SetAnnotations(newAnnotations)
	}
	// add last configuration annotation with contents pointing to latest marshalled resources
	// this is needed to see if new changes will need to be applied during reconciliation
	newAnnotations[lastAppliedConfigurationAnnotation] = newConfiguration
	obj.SetAnnotations(newAnnotations)

	// Find Group Version resource for rest mapping
	gvk := obj.GroupVersionKind()
	mapping, err := mapper.RESTMapping(gvk.GroupKind(), gvk.Version)
	if err != nil {
		return nil, fmt.Errorf("getting REST mapping: %w", err)
	}

	desiredObj := &obj
	namespace, err := meta.NewAccessor().Namespace(desiredObj)
	if err != nil {
		return nil, fmt.Errorf("creating namespace accessor: %w", err)
	}

	var dr dynamic.ResourceInterface
	if namespace != "" && mapping.Scope.Name() == meta.RESTScopeNameNamespace {
		// namespaced resources should specify the namespace
		dr = dynamicClient.Resource(mapping.Resource).Namespace(namespace)
	} else {
		// for cluster-wide resources
		dr = dynamicClient.Resource(mapping.Resource)
	}

	name, err := meta.NewAccessor().Name(desiredObj)
	if err != nil {
		return nil, fmt.Errorf("creating name accessor: %w", err)
	}

	// check if resources needs to be applied
	existingObj, _ := dr.Get(ctx, name, metav1.GetOptions{})
	applyChanges := shouldApplyChanges(dr, existingObj, newConfiguration)

	if !applyChanges { // no need to apply changes as resource has not changed
		return existingObj, nil
	}

	// apply new changes which will lead to creation of new resources
	return applyChangesFn(dr, desiredObj, existingObj)
}

func shouldApplyChanges(dynamicClient dynamic.ResourceInterface, existingObj *unstructured.Unstructured, newConfiguration string) bool {
	if existingObj == nil {
		return true
	}

	originalAnnotations := existingObj.GetAnnotations()
	if originalAnnotations != nil {
		lastApplied, ok := originalAnnotations[lastAppliedConfigurationAnnotation]
		if !ok {
			return true // new object, create it
		}
		return newConfiguration != lastApplied // check if configuration has changed before applying changes
	}

	return true
}

func applyChangesFn(client dynamic.ResourceInterface, desiredObj *unstructured.Unstructured, existingObj *unstructured.Unstructured) (runtime.Object, error) {
	if existingObj == nil { // create object if it does not exist
		newObj, err := client.Create(ctx, desiredObj, metav1.CreateOptions{
			FieldManager: fieldManager,
		})
		if err != nil {
			return newObj, fmt.Errorf("creating new object: %w", err)
		}
		return newObj, nil
	}

	desiredObj.SetResourceVersion(existingObj.GetResourceVersion())

	// we are replacing the whole object instead of using server-side apply which is in beta
	// the object is set to exactly desired object
	updatedObj, err := client.Update(ctx, desiredObj, metav1.UpdateOptions{
		FieldManager: fieldManager,
	})
	if err != nil {
		return updatedObj, fmt.Errorf("updating object: %w", err)
	}
	return updatedObj, nil
}
