package clusters

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/clusters/types"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/config"

	"github.com/operator-framework/api/pkg/operators/v1alpha1"
	operatorsv1alpha1 "github.com/operator-framework/api/pkg/operators/v1alpha1"
	operatorsv1alpha2 "github.com/operator-framework/api/pkg/operators/v1alpha2"
	"github.com/stackrox/acs-fleet-manager/pkg/api"
	"github.com/stackrox/acs-fleet-manager/pkg/db"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/discovery/cached/memory"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/restmapper"
	"k8s.io/client-go/tools/clientcmd"
)

const (
	dinosaurOperatorCatalogSourceName      = "managed-dinosaur-cs"
	dinosaurOperatorOperatorGroupName      = "managed-dinosaur-og"
	dinosaurOperatorSubscriptionName       = "managed-dinosaur-sub"
	fleetShardOperatorCatalogSourceName    = "fleetshard-operator-cs"
	fleetShardOperatorOperatorGroupName    = "fleetshard-operator-og"
	fleetShardOperatorSubscriptionName     = "fleetshard-operator-sub"
	fleetShardOperatorParametersSecretName = "addon-fleetshard-operator-parameters"
)

// fieldManager indicates that the fleet-manager will be used as a field manager for conflict resolution
const fieldManager = "fleet-manager"

// lastAppliedConfigurationAnnotation is an annotation applied in a resources which tracks the last applied configuration of a resource.
// this is used to decide whether a new apply request should be taken into account or not
const lastAppliedConfigurationAnnotation = "fleet-manager/last-applied-resource-configuration"

// dinosaurSREOpenIDPSecretName is the secret name holding the clientSecret content
const dinosaurSREOpenIDPSecretName = "dinosaur-sre-idp-secret"

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

// blank assignment to verify that StandaloneProvider implements Provider
var _ Provider = &StandaloneProvider{}

// Create ...
func (s *StandaloneProvider) Create(request *types.ClusterRequest) (*types.ClusterSpec, error) {
	return nil, nil
}

// Delete ...
func (s *StandaloneProvider) Delete(spec *types.ClusterSpec) (bool, error) {
	return true, nil
}

// InstallDinosaurOperator ...
func (s *StandaloneProvider) InstallDinosaurOperator(clusterSpec *types.ClusterSpec) (bool, error) {
	_, err := s.ApplyResources(clusterSpec, types.ResourceSet{
		Resources: []interface{}{
			s.buildDinosaurOperatorNamespace(),
			s.buildDinosaurOperatorCatalogSource(),
			s.buildDinosaurOperatorOperatorGroup(),
			s.buildDinosaurOperatorSubscription(),
		},
	})

	return true, err
}

func (s *StandaloneProvider) buildDinosaurOperatorNamespace() *v1.Namespace {
	dinosaurOperatorOLMConfig := s.dataplaneClusterConfig.DinosaurOperatorOLMConfig
	return &v1.Namespace{
		TypeMeta: metav1.TypeMeta{
			APIVersion: v1.SchemeGroupVersion.String(),
			Kind:       "Namespace",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: dinosaurOperatorOLMConfig.Namespace,
		},
	}
}

func (s *StandaloneProvider) buildDinosaurOperatorCatalogSource() *operatorsv1alpha1.CatalogSource {
	dinosaurOperatorOLMConfig := s.dataplaneClusterConfig.DinosaurOperatorOLMConfig
	return &operatorsv1alpha1.CatalogSource{
		TypeMeta: metav1.TypeMeta{
			APIVersion: operatorsv1alpha1.SchemeGroupVersion.String(),
			Kind:       operatorsv1alpha1.CatalogSourceKind,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      dinosaurOperatorCatalogSourceName,
			Namespace: dinosaurOperatorOLMConfig.CatalogSourceNamespace,
		},
		Spec: operatorsv1alpha1.CatalogSourceSpec{
			SourceType: operatorsv1alpha1.SourceTypeGrpc,
			Image:      dinosaurOperatorOLMConfig.IndexImage,
		},
	}
}

func (s *StandaloneProvider) buildDinosaurOperatorOperatorGroup() *operatorsv1alpha2.OperatorGroup {
	dinosaurOperatorOLMConfig := s.dataplaneClusterConfig.DinosaurOperatorOLMConfig
	return &operatorsv1alpha2.OperatorGroup{
		TypeMeta: metav1.TypeMeta{
			APIVersion: operatorsv1alpha2.SchemeGroupVersion.String(),
			Kind:       operatorsv1alpha2.OperatorGroupKind,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      dinosaurOperatorOperatorGroupName,
			Namespace: dinosaurOperatorOLMConfig.Namespace,
		},
		// Spec.TargetNamespaces intentionally not set, which means "select all namespaces"
		Spec: operatorsv1alpha2.OperatorGroupSpec{},
	}
}

func (s *StandaloneProvider) buildDinosaurOperatorSubscription() *operatorsv1alpha1.Subscription {
	dinosaurOperatorOLMConfig := s.dataplaneClusterConfig.DinosaurOperatorOLMConfig
	return &operatorsv1alpha1.Subscription{
		TypeMeta: metav1.TypeMeta{
			APIVersion: operatorsv1alpha1.SchemeGroupVersion.String(),
			Kind:       operatorsv1alpha1.SubscriptionKind,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      dinosaurOperatorSubscriptionName,
			Namespace: dinosaurOperatorOLMConfig.Namespace,
		},
		Spec: &operatorsv1alpha1.SubscriptionSpec{
			CatalogSource:          dinosaurOperatorCatalogSourceName,
			Channel:                dinosaurOperatorOLMConfig.SubscriptionChannel,
			CatalogSourceNamespace: dinosaurOperatorOLMConfig.CatalogSourceNamespace,
			InstallPlanApproval:    operatorsv1alpha1.ApprovalAutomatic,
			Package:                dinosaurOperatorOLMConfig.Package,
		},
	}
}

// InstallFleetshard ...
func (s *StandaloneProvider) InstallFleetshard(clusterSpec *types.ClusterSpec, params []types.Parameter) (bool, error) {
	_, err := s.ApplyResources(clusterSpec, types.ResourceSet{
		Resources: []interface{}{
			s.buildFleetShardOperatorNamespace(),
			s.buildFleetShardSyncSecret(params),
			s.buildFleetShardOperatorCatalogSource(),
			s.buildFleetShardOperatorOperatorGroup(),
			s.buildFleetShardOperatorSubscription(),
		},
	})

	return true, err
}

func (s *StandaloneProvider) buildFleetShardOperatorNamespace() *v1.Namespace {
	fleetshardOLMConfig := s.dataplaneClusterConfig.FleetshardOperatorOLMConfig
	return &v1.Namespace{
		TypeMeta: metav1.TypeMeta{
			APIVersion: v1.SchemeGroupVersion.String(),
			Kind:       "Namespace",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: fleetshardOLMConfig.Namespace,
		},
	}
}

func (s *StandaloneProvider) buildFleetShardOperatorCatalogSource() *operatorsv1alpha1.CatalogSource {
	fleetshardOLMConfig := s.dataplaneClusterConfig.FleetshardOperatorOLMConfig
	return &operatorsv1alpha1.CatalogSource{
		TypeMeta: metav1.TypeMeta{
			APIVersion: operatorsv1alpha1.SchemeGroupVersion.String(),
			Kind:       operatorsv1alpha1.CatalogSourceKind,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      fleetShardOperatorCatalogSourceName,
			Namespace: fleetshardOLMConfig.CatalogSourceNamespace,
		},
		Spec: v1alpha1.CatalogSourceSpec{
			SourceType: v1alpha1.SourceTypeGrpc,
			Image:      fleetshardOLMConfig.IndexImage,
		},
	}
}

func (s *StandaloneProvider) buildFleetShardOperatorOperatorGroup() *operatorsv1alpha2.OperatorGroup {
	fleetshardOLMConfig := s.dataplaneClusterConfig.FleetshardOperatorOLMConfig
	return &operatorsv1alpha2.OperatorGroup{
		TypeMeta: metav1.TypeMeta{
			APIVersion: operatorsv1alpha2.SchemeGroupVersion.String(),
			Kind:       operatorsv1alpha2.OperatorGroupKind,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      fleetShardOperatorOperatorGroupName,
			Namespace: fleetshardOLMConfig.Namespace,
		},
		// Spec.TargetNamespaces intentionally not set, which means "select all namespaces"
		Spec: operatorsv1alpha2.OperatorGroupSpec{},
	}
}

func (s *StandaloneProvider) buildFleetShardOperatorSubscription() *operatorsv1alpha1.Subscription {
	fleetshardOLMConfig := s.dataplaneClusterConfig.FleetshardOperatorOLMConfig
	return &operatorsv1alpha1.Subscription{
		TypeMeta: metav1.TypeMeta{
			APIVersion: operatorsv1alpha1.SchemeGroupVersion.String(),
			Kind:       operatorsv1alpha1.SubscriptionKind,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      fleetShardOperatorSubscriptionName,
			Namespace: fleetshardOLMConfig.Namespace,
		},
		Spec: &operatorsv1alpha1.SubscriptionSpec{
			CatalogSource:          fleetShardOperatorCatalogSourceName,
			Channel:                fleetshardOLMConfig.SubscriptionChannel,
			CatalogSourceNamespace: fleetshardOLMConfig.CatalogSourceNamespace,
			InstallPlanApproval:    operatorsv1alpha1.ApprovalAutomatic,
			Package:                fleetshardOLMConfig.Package,
		},
	}
}

func (s *StandaloneProvider) buildFleetShardSyncSecret(params []types.Parameter) *v1.Secret {
	secretStringData := map[string]string{}
	for _, param := range params {
		secretStringData[param.ID] = param.Value
	}

	fleetshardOLMConfig := s.dataplaneClusterConfig.FleetshardOperatorOLMConfig
	return &v1.Secret{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Secret",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      fleetShardOperatorParametersSecretName,
			Namespace: fleetshardOLMConfig.Namespace,
		},
		StringData: secretStringData,
	}
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

// AddIdentityProvider ...
func (s *StandaloneProvider) AddIdentityProvider(clusterSpec *types.ClusterSpec, identityProvider types.IdentityProviderInfo) (*types.IdentityProviderInfo, error) {
	// setup identity provider
	_, err := s.ApplyResources(clusterSpec, types.ResourceSet{
		Resources: []interface{}{
			s.buildOpenIDPClientSecret(identityProvider),
			s.buildIdentityProviderResource(identityProvider),
		},
	})

	return &identityProvider, err
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
			"clientSecret": identityProvider.OpenID.ClientSecret,
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

// ApplyResources ...
func (s *StandaloneProvider) ApplyResources(clusterSpec *types.ClusterSpec, resources types.ResourceSet) (*types.ResourceSet, error) {
	if s.dataplaneClusterConfig.RawKubernetesConfig == nil {
		return &resources, nil // no kubeconfig read, do nothing.
	}

	contextName := s.dataplaneClusterConfig.FindClusterNameByClusterID(clusterSpec.InternalID)
	override := &clientcmd.ConfigOverrides{CurrentContext: contextName}
	config := *s.dataplaneClusterConfig.RawKubernetesConfig
	restConfig, err := clientcmd.NewNonInteractiveClientConfig(config, override.CurrentContext, override, &clientcmd.ClientConfigLoadingRules{}).
		ClientConfig()

	if err != nil {
		return nil, fmt.Errorf("creating config: %w", err)
	}

	dynamicClient, err := dynamic.NewForConfig(restConfig)
	if err != nil {
		return nil, fmt.Errorf("creating dynamic client: %w", err)
	}

	// Create a REST mapper that tracks information about the available resources in the cluster.
	dc, err := discovery.NewDiscoveryClientForConfig(restConfig)
	if err != nil {
		return nil, fmt.Errorf("creating discovery client: %w", err)
	}

	discoveryCachedClient := memory.NewMemCacheClient(dc)
	mapper := restmapper.NewDeferredDiscoveryRESTMapper(discoveryCachedClient)

	for _, resource := range resources.Resources {
		_, err = applyResource(dynamicClient, mapper, resource)
		if err != nil {
			return nil, err
		}
	}

	return &resources, nil
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

	items := []types.CloudProviderRegionInfo{}
	for _, result := range results {
		items = append(items, types.CloudProviderRegionInfo{
			ID:              result.Region,
			Name:            result.Region,
			DisplayName:     result.Region,
			SupportsMultiAZ: result.MultiAZ,
			CloudProviderID: providerInf.ID,
		})
	}

	return &types.CloudProviderRegionInfoList{Items: items}, nil
}

func applyResource(dynamicClient dynamic.Interface, mapper *restmapper.DeferredDiscoveryRESTMapper, resource interface{}) (runtime.Object, error) {
	// parse resource obj to unstructure.Unstructered
	data, err := json.Marshal(resource)
	if err != nil {
		return nil, fmt.Errorf("marshalling resource of type %T with value %v into JSON: %w", resource, resource, err)
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
			return nil, fmt.Errorf("creating new object: %w", err)
		}
		return c, nil
	}

	desiredObj.SetResourceVersion(existingObj.GetResourceVersion())

	// we are replacing the whole object instead of using server-side apply which is in beta
	// the object is set to exactly desired object
	updatedObj, err := client.Update(ctx, desiredObj, metav1.UpdateOptions{
		FieldManager: fieldManager,
	})
	if err != nil {
		return nil, fmt.Errorf("updating object: %w", err)
	}
	return updatedObj, nil
}
