// Package mocks ...
package mocks

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/http/httptest"
	"reflect"
	"sync"

	amsv1 "github.com/openshift-online/ocm-sdk-go/accountsmgmt/v1"
	authorizationsv1 "github.com/openshift-online/ocm-sdk-go/authorizations/v1"

	"k8s.io/apimachinery/pkg/util/wait"

	"time"

	"github.com/gorilla/mux"
	ocmErrors "github.com/stackrox/acs-fleet-manager/pkg/errors"

	clustersmgmtv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	v1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
)

// EndpointPathClusters ...
const (
	// EndpointPathClusters ocm clusters management service clusters endpoint
	EndpointPathClusters = "/api/clusters_mgmt/v1/clusters"
	// EndpointPathCluster ocm clusters management service clusters endpoint
	EndpointPathCluster = "/api/clusters_mgmt/v1/clusters/{id}"

	// EndpointPathClusterIdentityProviders ocm clusters management service clusters identity provider create endpoint
	EndpointPathClusterIdentityProviders = "/api/clusters_mgmt/v1/clusters/{id}/identity_providers"
	// EndpointPathClusterIdentityProvider ocm clusters management service clusters identity provider update endpoint
	EndpointPathClusterIdentityProvider = "/api/clusters_mgmt/v1/clusters/{id}/identity_providers/{idp_id}"

	// EndpointPathSyncsets ocm clusters management service syncset endpoint
	EndpointPathSyncsets = "/api/clusters_mgmt/v1/clusters/{id}/external_configuration/syncsets"
	// EndpointPathSyncset ocm clusters management service syncset endpoint
	EndpointPathSyncset = "/api/clusters_mgmt/v1/clusters/{id}/external_configuration/syncsets/{syncsetID}"
	// EndpointPathIngresses ocm cluster management ingress endpoint
	EndpointPathIngresses = "/api/clusters_mgmt/v1/clusters/{id}/ingresses"
	// EndpointPathCloudProviders ocm cluster management cloud providers endpoint
	EndpointPathCloudProviders = "/api/clusters_mgmt/v1/cloud_providers"
	// EndpointPathCloudProvider ocm cluster management cloud provider endpoint
	EndpointPathCloudProvider = "/api/clusters_mgmt/v1/cloud_providers/{id}"
	// EndpointPathCloudProviderRegions ocm cluster management cloud provider regions endpoint
	EndpointPathCloudProviderRegions = "/api/clusters_mgmt/v1/cloud_providers/{id}/regions"
	// EndpointPathCloudProviderRegion ocm cluster management cloud provider region endpoint
	EndpointPathCloudProviderRegion = "/api/clusters_mgmt/v1/cloud_providers/{providerID}/regions/{regionID}"
	// EndpointPathClusterStatus ocm cluster management cluster status endpoint
	EndpointPathClusterStatus = "/api/clusters_mgmt/v1/clusters/{id}/status"
	// EndpointPathClusterAddons ocm cluster management cluster addons endpoint
	EndpointPathClusterAddons = "/api/clusters_mgmt/v1/clusters/{id}/addons"
	// EndpointPathMachinePools ocm cluster management machine pools endpoint
	EndpointPathMachinePools = "/api/clusters_mgmt/v1/clusters/{id}/machine_pools"
	// EndpointPathMachinePool ocm cluster management machine pool endpoint
	EndpointPathMachinePool = "/api/clusters_mgmt/v1/clusters/{id}/machine_pools/{machinePoolId}"
	// EndpointPathAddonInstallations ocm cluster addon installations endpoint
	EndpointPathAddonInstallations = "/api/clusters_mgmt/v1/clusters/{id}/addons"
	// EndpointPathAddonInstallation ocm cluster addon installation endpoint
	EndpointPathAddonInstallation = "/api/clusters_mgmt/v1/clusters/{id}/addons/{addoninstallationId}"
	// EndpointPathFleetshardOperatorAddonInstallation ocm cluster fleetshard-operator-qe addon installation endpoint
	EndpointPathFleetshardOperatorAddonInstallation = "/api/clusters_mgmt/v1/clusters/{id}/addons/fleetshard-operator-qe"
	// EndpointPathClusterLoggingOperatorAddonInstallation ocm cluster cluster-logging-operator addon installation endpoint
	EndpointPathClusterLoggingOperatorAddonInstallation = "/api/clusters_mgmt/v1/clusters/{id}/addons/cluster-logging-operator"

	EndpointPathClusterAuthorization = "/api/accounts_mgmt/v1/cluster_authorizations"
	EndpointPathSubscription         = "/api/accounts_mgmt/v1/subscriptions/{id}"
	EndpointPathSubscriptionSearch   = "/api/accounts_mgmt/v1/subscriptions"

	EndpointPathTermsReview = "/api/authorizations/v1/terms_review"

	// Default values for getX functions

	// MockClusterID default mock cluster id used in the mock ocm server
	MockClusterID = "2aad9fc1-c40e-471f-8d57-fdaecc7d3686"
	// MockCloudProviderID default mock provider ID
	MockCloudProviderID = "aws"
	// MockClusterExternalID default mock cluster external ID
	MockClusterExternalID = "2aad9fc1-c40e-471f-8d57-fdaecc7d3686"
	// MockClusterState default mock cluster state
	MockClusterState = clustersmgmtv1.ClusterStateReady
	// MockCloudProviderDisplayName default mock provider display name
	MockCloudProviderDisplayName = "AWS"
	// MockCloudRegionID default mock cluster region
	MockCloudRegionID = "us-east-1"
	// MockCloudRegionDisplayName default mock cloud region display name
	MockCloudRegionDisplayName = "US East, N. Virginia"
	// MockSyncsetID default mock syncset id used in the mock ocm server
	MockSyncsetID = "ext-8a41f783-b5e4-4692-a7cd-c0b9c8eeede9"
	// MockIngressID default mock ingress id used in the mock ocm server
	MockIngressID = "s1h5"
	// MockIngressDNS default mock ingress dns used in the mock ocm server
	MockIngressDNS = "apps.mk-btq2d1h8d3b1.b3k3.s1.devshift.org"
	// MockIngressHref default mock ingress HREF used in the mock ocm server
	MockIngressHref = "/api/clusters_mgmt/v1/clusters/000/ingresses/i8y1"
	// MockIngressListening default mock ingress listening used in the mock ocm server
	MockIngressListening = clustersmgmtv1.ListeningMethodExternal
	// MockClusterAddonID default mock cluster addon ID
	MockClusterAddonID = "managed-central-qe"
	// MockFleetshardAddonID default mock ID for the Fleetshard Operator
	MockFleetshardAddonID = "fleetshard-operator-qe"
	// MockClusterLoggingOperatorAddonID default mock ID for the Cluster Logging Operator
	MockClusterLoggingOperatorAddonID = "cluster-logging-operator"
	// MockClusterAddonState default mock cluster addon state
	MockClusterAddonState = clustersmgmtv1.AddOnInstallationStateReady
	// MockClusterAddonDescription default mock cluster addon description
	MockClusterAddonDescription = "InstallWaiting"
	// MockMachinePoolID default machine pool ID
	MockMachinePoolID = "managed"
	// MockMachinePoolReplicas default number of machine pool replicas
	MockMachinePoolReplicas = 3
	// MockOpenshiftVersion default cluster openshift version
	MockOpenshiftVersion = "openshift-v4.6.1"
	// MockMultiAZ default value
	MockMultiAZ = true
	// MockClusterComputeNodes default nodes
	MockClusterComputeNodes = 3
	// MockIdentityProviderID default identity provider ID
	MockIdentityProviderID = "identity-provider-id"
	//
	MockSubID = "pphCb6sIQPqtjMtL0GQaX6i4bP"
)

// EndpointClusterGet variables for endpoints
var (
	EndpointClusterGet                                   = Endpoint{EndpointPathCluster, http.MethodGet}
	EndpointClusterPatch                                 = Endpoint{EndpointPathCluster, http.MethodPatch}
	EndpointDinosaurDelete                               = Endpoint{EndpointPathSyncset, http.MethodDelete}
	EndpointClustersGet                                  = Endpoint{EndpointPathClusters, http.MethodGet}
	EndpointClustersPost                                 = Endpoint{EndpointPathClusters, http.MethodPost}
	EndpointClusterDelete                                = Endpoint{EndpointPathCluster, http.MethodDelete}
	EndpointClusterSyncsetsPost                          = Endpoint{EndpointPathSyncsets, http.MethodPost}
	EndpointClusterSyncsetGet                            = Endpoint{EndpointPathSyncset, http.MethodGet}
	EndpointClusterSyncsetPatch                          = Endpoint{EndpointPathSyncset, http.MethodPatch}
	EndpointClusterIngressGet                            = Endpoint{EndpointPathIngresses, http.MethodGet}
	EndpointCloudProvidersGet                            = Endpoint{EndpointPathCloudProviders, http.MethodGet}
	EndpointCloudProviderGet                             = Endpoint{EndpointPathCloudProvider, http.MethodGet}
	EndpointCloudProviderRegionsGet                      = Endpoint{EndpointPathCloudProviderRegions, http.MethodGet}
	EndpointCloudProviderRegionGet                       = Endpoint{EndpointPathCloudProviderRegion, http.MethodGet}
	EndpointClusterStatusGet                             = Endpoint{EndpointPathClusterStatus, http.MethodGet}
	EndpointClusterAddonsGet                             = Endpoint{EndpointPathClusterAddons, http.MethodGet}
	EndpointClusterAddonPost                             = Endpoint{EndpointPathClusterAddons, http.MethodPost}
	EndpointMachinePoolsGet                              = Endpoint{EndpointPathMachinePools, http.MethodGet}
	EndpointMachinePoolPost                              = Endpoint{EndpointPathMachinePools, http.MethodPost}
	EndpointMachinePoolPatch                             = Endpoint{EndpointPathMachinePool, http.MethodPatch}
	EndpointMachinePoolGet                               = Endpoint{EndpointPathMachinePool, http.MethodGet}
	EndpointIdentityProviderPost                         = Endpoint{EndpointPathClusterIdentityProviders, http.MethodPost}
	EndpointIdentityProviderPatch                        = Endpoint{EndpointPathClusterIdentityProvider, http.MethodPatch}
	EndpointAddonInstallationsPost                       = Endpoint{EndpointPathAddonInstallations, http.MethodPost}
	EndpointAddonInstallationGet                         = Endpoint{EndpointPathAddonInstallation, http.MethodGet}
	EndpointAddonInstallationPatch                       = Endpoint{EndpointPathAddonInstallation, http.MethodPatch}
	EndpointFleetshardOperatorAddonInstallationGet       = Endpoint{EndpointPathFleetshardOperatorAddonInstallation, http.MethodGet}
	EndpointFleetshardOperatorAddonInstallationPatch     = Endpoint{EndpointPathFleetshardOperatorAddonInstallation, http.MethodPatch}
	EndpointFleetshardOperatorAddonInstallationPost      = Endpoint{EndpointPathFleetshardOperatorAddonInstallation, http.MethodPost}
	EndpointClusterLoggingOperatorAddonInstallationGet   = Endpoint{EndpointPathClusterLoggingOperatorAddonInstallation, http.MethodGet}
	EndpointClusterLoggingOperatorAddonInstallationPatch = Endpoint{EndpointPathClusterLoggingOperatorAddonInstallation, http.MethodPatch}
	EndpointClusterLoggingOperatorAddonInstallationPost  = Endpoint{EndpointPathClusterLoggingOperatorAddonInstallation, http.MethodPost}
	EndpointClusterAuthorizationPost                     = Endpoint{EndpointPathClusterAuthorization, http.MethodPost}
	EndpointSubscriptionDelete                           = Endpoint{EndpointPathSubscription, http.MethodDelete}
	EndpointSubscriptionSearch                           = Endpoint{EndpointPathSubscriptionSearch, http.MethodGet}
	EndpointTermsReviewPost                              = Endpoint{EndpointPathTermsReview, http.MethodPost}
)

// MockIdentityProvider variables for mocked ocm types
//
// these are the default types that will be returned by the emulated ocm api
// to override these values, do not set them directly e.g. mocks.MockSyncset = ...
// instead use the Set*Response functions provided by MockConfigurableServerBuilder e.g. SetClusterGetResponse(...)
var (
	MockIdentityProvider                        *clustersmgmtv1.IdentityProvider
	MockSyncset                                 *clustersmgmtv1.Syncset
	MockIngressList                             *clustersmgmtv1.IngressList
	MockCloudProvider                           *clustersmgmtv1.CloudProvider
	MockCloudProviderList                       *clustersmgmtv1.CloudProviderList
	MockCloudProviderRegion                     *clustersmgmtv1.CloudRegion
	MockCloudProviderRegionList                 *clustersmgmtv1.CloudRegionList
	MockClusterStatus                           *clustersmgmtv1.ClusterStatus
	MockClusterAddonInstallation                *clustersmgmtv1.AddOnInstallation
	MockClusterAddonInstallationList            *clustersmgmtv1.AddOnInstallationList
	MockFleetshardOperatorAddonInstallation     *clustersmgmtv1.AddOnInstallation
	MockClusterLoggingOperatorAddonInstallation *clustersmgmtv1.AddOnInstallation
	MockMachinePoolList                         *clustersmgmtv1.MachinePoolList
	MockMachinePool                             *clustersmgmtv1.MachinePool
	MockCluster                                 *clustersmgmtv1.Cluster
	MockClusterAuthorization                    *amsv1.ClusterAuthorizationResponse
	MockSubscription                            *amsv1.Subscription
	MockSubscriptionSearch                      []*amsv1.Subscription
	MockTermsReview                             *authorizationsv1.TermsReviewResponse
)

// routerSwapper is an http.Handler that allows you to swap mux routers.
type routerSwapper struct {
	mu     sync.Mutex
	router *mux.Router
}

// Swap changes the old router with the new one.
func (rs *routerSwapper) Swap(newRouter *mux.Router) {
	rs.mu.Lock()
	rs.router = newRouter
	rs.mu.Unlock()
}

var router *mux.Router

// rSwapper is required if any change to the Router for mocked OCM server is needed
var rSwapper *routerSwapper

// Endpoint is a wrapper around an endpoint and the method used to interact with that endpoint e.g. GET /clusters
type Endpoint struct {
	Path   string
	Method string
}

// HandlerRegister is a cache that maps Endpoints to their handlers
type HandlerRegister map[Endpoint]func(w http.ResponseWriter, r *http.Request)

// MockConfigurableServerBuilder allows mock ocm api servers to be built
type MockConfigurableServerBuilder struct {
	// handlerRegister cache of endpoints and handlers to be used when the mock ocm api server is built
	handlerRegister HandlerRegister
}

// NewMockConfigurableServerBuilder returns a new builder that can be used to define a mock ocm api server
func NewMockConfigurableServerBuilder() *MockConfigurableServerBuilder {
	// get the default endpoint handlers that'll be used if they're not overridden
	handlerRegister, err := getDefaultHandlerRegister()
	if err != nil {
		panic(err)
	}
	return &MockConfigurableServerBuilder{
		handlerRegister: handlerRegister,
	}
}

// SetClusterGetResponse set a mock response cluster or error for the POST /api/clusters_mgmt/v1/clusters endpoint
func (b *MockConfigurableServerBuilder) SetClusterGetResponse(cluster *clustersmgmtv1.Cluster, err *ocmErrors.ServiceError) {
	b.handlerRegister[EndpointClusterGet] = buildMockRequestHandler(cluster, err)
}

// SetDinosaurDeleteResponse set a mock response cluster or error for the DELETE /api/clusters_mgmt/v1/clusters/{id}/external_configuration/syncsets/{syncsetID} endpoint
func (b *MockConfigurableServerBuilder) SetDinosaurDeleteResponse(syncset *clustersmgmtv1.Syncset, err *ocmErrors.ServiceError) {
	b.handlerRegister[EndpointDinosaurDelete] = buildMockRequestHandler(syncset, err)
}

// SetClusterPatchResponse set a mock response cluster or error for the PATCH /api/clusters_mgmt/v1/clusters/{id} endpoint
func (b *MockConfigurableServerBuilder) SetClusterPatchResponse(cluster *clustersmgmtv1.Cluster, err *ocmErrors.ServiceError) {
	b.handlerRegister[EndpointClusterPatch] = buildMockRequestHandler(cluster, err)
}

// SetClustersPostResponse set a mock response cluster or error for the POST /api/clusters_mgmt/v1/clusters endpoint
func (b *MockConfigurableServerBuilder) SetClustersPostResponse(cluster *clustersmgmtv1.Cluster, err *ocmErrors.ServiceError) {
	b.handlerRegister[EndpointClustersPost] = buildMockRequestHandler(cluster, err)
}

// SetClustersGetResponse set a mock response cluster or error for the GET /api/clusters_mgmt/v1/clusters endpoint
func (b *MockConfigurableServerBuilder) SetClustersGetResponse(cluster *clustersmgmtv1.Cluster, err *ocmErrors.ServiceError) {
	b.handlerRegister[EndpointClustersGet] = buildMockRequestHandler(cluster, err)
}

// SetClusterDeleteResponse set a mock response cluster or error for the DELETE /api/clusters_mgmt/v1/clusters/{id} endpoint
func (b *MockConfigurableServerBuilder) SetClusterDeleteResponse(cluster *clustersmgmtv1.Cluster, err *ocmErrors.ServiceError) {
	b.handlerRegister[EndpointClusterDelete] = buildMockRequestHandler(cluster, err)
}

// SetClusterSyncsetGetResponse set a mock response syncset or error for the GET /api/clusters_mgmt/v1/clusters/{id}/external_configuration/syncsets/{syncsetID}
func (b *MockConfigurableServerBuilder) SetClusterSyncsetGetResponse(syncset *clustersmgmtv1.Syncset, err *ocmErrors.ServiceError) {
	b.handlerRegister[EndpointClusterSyncsetGet] = buildMockRequestHandler(syncset, err)
}

// SetClusterSyncsetPostResponse set a mock response syncset or error for the POST /api/clusters_mgmt/v1/clusters/{id}/syncsets endpoint
func (b *MockConfigurableServerBuilder) SetClusterSyncsetPostResponse(syncset *clustersmgmtv1.Syncset, err *ocmErrors.ServiceError) {
	b.handlerRegister[EndpointClusterSyncsetsPost] = buildMockRequestHandler(syncset, err)
}

// SetClusterSyncsetPatchResponse set a mock response syncset or error for the Patch /api/clusters_mgmt/v1/clusters/{id}/syncsets endpoint
func (b *MockConfigurableServerBuilder) SetClusterSyncsetPatchResponse(syncset *clustersmgmtv1.Syncset, err *ocmErrors.ServiceError) {
	b.handlerRegister[EndpointClusterSyncsetPatch] = buildMockRequestHandler(syncset, err)
}

// SetClusterSyncsetPostRequestHandler ...
func (b *MockConfigurableServerBuilder) SetClusterSyncsetPostRequestHandler(customMockRequestHandler func() func(w http.ResponseWriter, r *http.Request)) {
	b.handlerRegister[EndpointClusterSyncsetsPost] = customMockRequestHandler()
}

// SetClusterIngressGetResponse set a mock response ingress or error for the GET /api/clusters_mgmt/v1/clusters/{id}/ingresses endpoint
func (b *MockConfigurableServerBuilder) SetClusterIngressGetResponse(ingress *clustersmgmtv1.Ingress, err *ocmErrors.ServiceError) {
	b.handlerRegister[EndpointClusterIngressGet] = buildMockRequestHandler(ingress, err)
}

// SetCloudProvidersGetResponse set a mock response provider list or error for GET /api/clusters_mgmt/v1/cloud_providers
func (b *MockConfigurableServerBuilder) SetCloudProvidersGetResponse(providers *clustersmgmtv1.CloudProviderList, err *ocmErrors.ServiceError) {
	b.handlerRegister[EndpointCloudProvidersGet] = buildMockRequestHandler(providers, err)
}

// SetCloudRegionsGetResponse set a mock response region list or error for GET /api/clusters_mgmt/v1/cloud_providers/{id}/regions
func (b *MockConfigurableServerBuilder) SetCloudRegionsGetResponse(regions *clustersmgmtv1.CloudRegionList, err *ocmErrors.ServiceError) {
	b.handlerRegister[EndpointCloudProviderRegionsGet] = buildMockRequestHandler(regions, err)
}

// SetCloudRegionGetResponse set a mock response region or error for GET /api/clusters_mgmt/v1/cloud_providers/{id}/regions/{regionId}
func (b *MockConfigurableServerBuilder) SetCloudRegionGetResponse(region *clustersmgmtv1.CloudRegion, err *ocmErrors.ServiceError) {
	b.handlerRegister[EndpointCloudProviderRegionGet] = buildMockRequestHandler(region, err)
}

// SetClusterStatusGetResponse set a mock response cluster status or error for GET /api/clusters_mgmt/v1/clusters/{id}/status
func (b *MockConfigurableServerBuilder) SetClusterStatusGetResponse(status *clustersmgmtv1.ClusterStatus, err *ocmErrors.ServiceError) {
	b.handlerRegister[EndpointClusterStatusGet] = buildMockRequestHandler(status, err)
}

// SetClusterAddonsGetResponse set a mock response addon list or error for GET /api/clusters_mgmt/v1/clusters/{id}/addons
func (b *MockConfigurableServerBuilder) SetClusterAddonsGetResponse(addons *clustersmgmtv1.AddOnInstallationList, err *ocmErrors.ServiceError) {
	b.handlerRegister[EndpointClusterAddonsGet] = buildMockRequestHandler(addons, err)
}

// SetClusterAddonPostResponse set a mock response addon or error for POST /api/clusters_mgmt/v1/clusters/{id}/addons
func (b *MockConfigurableServerBuilder) SetClusterAddonPostResponse(addon *clustersmgmtv1.AddOnInstallation, err *ocmErrors.ServiceError) {
	b.handlerRegister[EndpointClusterAddonPost] = buildMockRequestHandler(addon, err)
}

// SetMachinePoolsGetResponse set a mock response machine pool or error for Get /api/clusters_mgmt/v1/clusters/{id}/machine_pools
func (b *MockConfigurableServerBuilder) SetMachinePoolsGetResponse(mp *clustersmgmtv1.MachinePoolList, err *ocmErrors.ServiceError) {
	b.handlerRegister[EndpointMachinePoolsGet] = buildMockRequestHandler(mp, err)
}

// SetMachinePoolGetResponse set a mock response machine pool list or error for Get /api/clusters_mgmt/v1/clusters/{id}/machine_pools/{machinePoolId}
func (b *MockConfigurableServerBuilder) SetMachinePoolGetResponse(mp *clustersmgmtv1.MachinePoolList, err *ocmErrors.ServiceError) {
	b.handlerRegister[EndpointMachinePoolGet] = buildMockRequestHandler(mp, err)
}

// SetMachinePoolPostResponse set a mock response for Post /api/clusters_mgmt/v1/clusters/{id}/machine_pools
func (b *MockConfigurableServerBuilder) SetMachinePoolPostResponse(mp *clustersmgmtv1.MachinePool, err *ocmErrors.ServiceError) {
	b.handlerRegister[EndpointMachinePoolPost] = buildMockRequestHandler(mp, err)
}

// SetMachinePoolPatchResponse set a mock response for Patch /api/clusters_mgmt/v1/clusters/{id}/machine_pools/{machinePoolId}
func (b *MockConfigurableServerBuilder) SetMachinePoolPatchResponse(mp *clustersmgmtv1.MachinePool, err *ocmErrors.ServiceError) {
	b.handlerRegister[EndpointMachinePoolPatch] = buildMockRequestHandler(mp, err)
}

// SetIdentityProviderPostResponse set a mock response for Post /api/clusters_mgmt/v1/clusters/{id}/identity_providers
func (b *MockConfigurableServerBuilder) SetIdentityProviderPostResponse(idp *clustersmgmtv1.IdentityProvider, err *ocmErrors.ServiceError) {
	b.handlerRegister[EndpointIdentityProviderPost] = buildMockRequestHandler(idp, err)
}

// SetIdentityProviderPatchResponse set a mock response for Patch /api/clusters_mgmt/v1/clusters/{id}/identity_providers/{idp_id}
func (b *MockConfigurableServerBuilder) SetIdentityProviderPatchResponse(idp *clustersmgmtv1.IdentityProvider, err *ocmErrors.ServiceError) {
	b.handlerRegister[EndpointIdentityProviderPatch] = buildMockRequestHandler(idp, err)
}

// SetClusterAuthorizationResponse set a mock response for Post /api/accounts_mgmt/v1/cluster_authorizations
func (b *MockConfigurableServerBuilder) SetClusterAuthorizationResponse(idp *amsv1.ClusterAuthorizationResponse, err *ocmErrors.ServiceError) {
	b.handlerRegister[EndpointClusterAuthorizationPost] = buildMockRequestHandler(idp, err)
}

// SetAddonInstallationsPostResponse set a mock response for Post /api/clusters_mgmt/v1/clusters/{id}/addons
func (b *MockConfigurableServerBuilder) SetAddonInstallationsPostResponse(ai *clustersmgmtv1.AddOnInstallation, err *ocmErrors.ServiceError) {
	b.handlerRegister[EndpointAddonInstallationsPost] = buildMockRequestHandler(ai, err)
}

// SetAddonInstallationGetResponse set a mock response for Get /api/clusters_mgmt/v1/clusters/{id}/addons/{addoninstallationId}
func (b *MockConfigurableServerBuilder) SetAddonInstallationGetResponse(ai *clustersmgmtv1.AddOnInstallation, err *ocmErrors.ServiceError) {
	b.handlerRegister[EndpointAddonInstallationGet] = buildMockRequestHandler(ai, err)
}

// SetAddonInstallationPatchResponse set a mock response for Patch /api/clusters_mgmt/v1/clusters/{id}/addons/{addoninstallationId}
func (b *MockConfigurableServerBuilder) SetAddonInstallationPatchResponse(ai *clustersmgmtv1.AddOnInstallation, err *ocmErrors.ServiceError) {
	b.handlerRegister[EndpointAddonInstallationPatch] = buildMockRequestHandler(ai, err)
}

// SetFleetshardOperatorAddonInstallationGetResponse ...
func (b *MockConfigurableServerBuilder) SetFleetshardOperatorAddonInstallationGetResponse(ai *clustersmgmtv1.AddOnInstallation, err *ocmErrors.ServiceError) {
	b.handlerRegister[EndpointFleetshardOperatorAddonInstallationGet] = buildMockRequestHandler(ai, err)
}

// SetFleetshardOperatorAddonInstallationPatchResponse ...
func (b *MockConfigurableServerBuilder) SetFleetshardOperatorAddonInstallationPatchResponse(ai *clustersmgmtv1.AddOnInstallation, err *ocmErrors.ServiceError) {
	b.handlerRegister[EndpointFleetshardOperatorAddonInstallationPatch] = buildMockRequestHandler(ai, err)
}

// SetFleetshardOperatorAddonInstallationPostResponse ...
func (b *MockConfigurableServerBuilder) SetFleetshardOperatorAddonInstallationPostResponse(ai *clustersmgmtv1.AddOnInstallation, err *ocmErrors.ServiceError) {
	b.handlerRegister[EndpointFleetshardOperatorAddonInstallationPost] = buildMockRequestHandler(ai, err)
}

// SetClusterLoggingOperatorAddonInstallationGetResponse ...
func (b *MockConfigurableServerBuilder) SetClusterLoggingOperatorAddonInstallationGetResponse(ai *clustersmgmtv1.AddOnInstallation, err *ocmErrors.ServiceError) {
	b.handlerRegister[EndpointClusterLoggingOperatorAddonInstallationGet] = buildMockRequestHandler(ai, err)
}

// SetClusterLoggingOperatorAddonInstallationPatchResponse ...
func (b *MockConfigurableServerBuilder) SetClusterLoggingOperatorAddonInstallationPatchResponse(ai *clustersmgmtv1.AddOnInstallation, err *ocmErrors.ServiceError) {
	b.handlerRegister[EndpointClusterLoggingOperatorAddonInstallationPatch] = buildMockRequestHandler(ai, err)
}

// SetClusterLoggingOperatorAddonInstallationPostResponse ...
func (b *MockConfigurableServerBuilder) SetClusterLoggingOperatorAddonInstallationPostResponse(ai *clustersmgmtv1.AddOnInstallation, err *ocmErrors.ServiceError) {
	b.handlerRegister[EndpointClusterLoggingOperatorAddonInstallationPost] = buildMockRequestHandler(ai, err)
}

// SetSubscriptionPathDeleteResponse ...
func (b *MockConfigurableServerBuilder) SetSubscriptionPathDeleteResponse(idp *amsv1.Subscription, err *ocmErrors.ServiceError) {
	b.handlerRegister[EndpointSubscriptionDelete] = buildMockRequestHandler(idp, err)
}

// SetSubscriptionSearchResponse ...
func (b *MockConfigurableServerBuilder) SetSubscriptionSearchResponse(sl *amsv1.SubscriptionList, err *ocmErrors.ServiceError) {
	b.handlerRegister[EndpointSubscriptionSearch] = buildMockRequestHandler(sl, err)
}

// SetTermsReviewPostResponse ...
func (b *MockConfigurableServerBuilder) SetTermsReviewPostResponse(idp *authorizationsv1.TermsReviewResponse, err *ocmErrors.ServiceError) {
	b.handlerRegister[EndpointTermsReviewPost] = buildMockRequestHandler(idp, err)
}

// Build builds the mock ocm api server using the endpoint handlers that have been set in the builder
func (b *MockConfigurableServerBuilder) Build() *httptest.Server {
	router = mux.NewRouter()
	rSwapper = &routerSwapper{sync.Mutex{}, router}

	// set up handlers from the builder
	for endpoint, handleFn := range b.handlerRegister {
		router.HandleFunc(endpoint.Path, handleFn).Methods(endpoint.Method)
	}
	server := httptest.NewUnstartedServer(rSwapper)
	l, err := net.Listen("tcp", "127.0.0.1:9876")
	if err != nil {
		log.Fatal(err)
	}
	server.Listener = l
	server.Start()
	err = wait.PollImmediate(time.Second, 10*time.Second, func() (done bool, err error) {
		_, err = http.Get("http://127.0.0.1:9876/api/clusters_mgmt/v1/cloud_providers/aws/regions")
		return err == nil, nil
	})
	if err != nil {
		log.Fatal("Timed out waiting for mock server to start.")
		panic(err)
	}
	return server
}

// SwapRouterResponse and update the router to handle this response
func (b *MockConfigurableServerBuilder) SwapRouterResponse(path string, method string, successType interface{}, serviceErr *ocmErrors.ServiceError) {
	b.handlerRegister[Endpoint{
		Path:   path,
		Method: method,
	}] = buildMockRequestHandler(successType, serviceErr)

	router = mux.NewRouter()
	for endpoint, handleFn := range b.handlerRegister {
		router.HandleFunc(endpoint.Path, handleFn).Methods(endpoint.Method)
	}

	rSwapper.Swap(router)
}

// ServeHTTP makes the routerSwapper to implement the http.Handler interface
// so that routerSwapper can be used by httptest.NewServer()
func (rs *routerSwapper) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	rs.mu.Lock()
	router := rs.router
	rs.mu.Unlock()
	router.ServeHTTP(w, r)
}

// getDefaultHandlerRegister returns a set of default endpoints and handlers used in the mock ocm api server
func getDefaultHandlerRegister() (HandlerRegister, error) {
	// define a list of default endpoints and handlers in the mock ocm api server, when new endpoints are used in the
	// managed-services-api service, a default ocm response should also be added here
	return HandlerRegister{
		EndpointClusterGet:                                   buildMockRequestHandler(MockCluster, nil),
		EndpointClusterPatch:                                 buildMockRequestHandler(MockCluster, nil),
		EndpointDinosaurDelete:                               buildMockRequestHandler(MockSyncset, nil),
		EndpointClustersGet:                                  buildMockRequestHandler(MockCluster, nil),
		EndpointClustersPost:                                 buildMockRequestHandler(MockCluster, nil),
		EndpointClusterDelete:                                buildMockRequestHandler(MockCluster, ocmErrors.NotFound("setting this to not found to mimick a successul deletion")),
		EndpointClusterSyncsetsPost:                          buildMockRequestHandler(MockSyncset, nil),
		EndpointClusterSyncsetGet:                            buildMockRequestHandler(MockSyncset, nil),
		EndpointClusterSyncsetPatch:                          buildMockRequestHandler(MockSyncset, nil),
		EndpointClusterIngressGet:                            buildMockRequestHandler(MockIngressList, nil),
		EndpointCloudProvidersGet:                            buildMockRequestHandler(MockCloudProviderList, nil),
		EndpointCloudProviderGet:                             buildMockRequestHandler(MockCloudProvider, nil),
		EndpointCloudProviderRegionsGet:                      buildMockRequestHandler(MockCloudProviderRegionList, nil),
		EndpointCloudProviderRegionGet:                       buildMockRequestHandler(MockCloudProviderRegion, nil),
		EndpointClusterStatusGet:                             buildMockRequestHandler(MockClusterStatus, nil),
		EndpointClusterAddonsGet:                             buildMockRequestHandler(MockClusterAddonInstallationList, nil),
		EndpointClusterAddonPost:                             buildMockRequestHandler(MockClusterAddonInstallation, nil),
		EndpointMachinePoolsGet:                              buildMockRequestHandler(MockMachinePoolList, nil),
		EndpointMachinePoolGet:                               buildMockRequestHandler(MockMachinePool, nil),
		EndpointMachinePoolPatch:                             buildMockRequestHandler(MockMachinePool, nil),
		EndpointMachinePoolPost:                              buildMockRequestHandler(MockMachinePool, nil),
		EndpointIdentityProviderPatch:                        buildMockRequestHandler(MockIdentityProvider, nil),
		EndpointIdentityProviderPost:                         buildMockRequestHandler(MockIdentityProvider, nil),
		EndpointAddonInstallationsPost:                       buildMockRequestHandler(MockClusterAddonInstallation, nil),
		EndpointAddonInstallationGet:                         buildMockRequestHandler(MockClusterAddonInstallation, nil),
		EndpointAddonInstallationPatch:                       buildMockRequestHandler(MockClusterAddonInstallation, nil),
		EndpointFleetshardOperatorAddonInstallationGet:       buildMockRequestHandler(MockFleetshardOperatorAddonInstallation, nil),
		EndpointFleetshardOperatorAddonInstallationPatch:     buildMockRequestHandler(MockFleetshardOperatorAddonInstallation, nil),
		EndpointFleetshardOperatorAddonInstallationPost:      buildMockRequestHandler(MockFleetshardOperatorAddonInstallation, nil),
		EndpointClusterLoggingOperatorAddonInstallationGet:   buildMockRequestHandler(MockClusterLoggingOperatorAddonInstallation, nil),
		EndpointClusterLoggingOperatorAddonInstallationPatch: buildMockRequestHandler(MockClusterLoggingOperatorAddonInstallation, nil),
		EndpointClusterLoggingOperatorAddonInstallationPost:  buildMockRequestHandler(MockClusterLoggingOperatorAddonInstallation, nil),
		EndpointClusterAuthorizationPost:                     buildMockRequestHandler(MockClusterAuthorization, nil),
		EndpointSubscriptionDelete:                           buildMockRequestHandler(MockSubscription, nil),
		EndpointSubscriptionSearch:                           buildMockRequestHandler(MockSubscriptionSearch, nil),
		EndpointTermsReviewPost:                              buildMockRequestHandler(MockTermsReview, nil),
	}, nil
}

// buildMockRequestHandler builds a generic handler for all ocm api server responses
// one of successType of serviceErr should be defined
// if serviceErr is defined, it will be provided as an ocm error response
// if successType is defined, it will be provided as an ocm success response
func buildMockRequestHandler(successType interface{}, serviceErr *ocmErrors.ServiceError) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if serviceErr != nil {
			w.WriteHeader(serviceErr.HTTPCode)
			if err := marshalOCMType(serviceErr, w); err != nil {
				panic(err)
			}
		} else if successType != nil {
			if err := marshalOCMType(successType, w); err != nil {
				panic(err)
			}
		} else {
			panic("no response was defined")
		}
	}
}

// marshalOCMType marshals known ocm types to a provided io.Writer using the ocm sdk marshallers
func marshalOCMType(t interface{}, w io.Writer) error {
	switch v := t.(type) {
	// handle cluster types
	case *clustersmgmtv1.Cluster:
		err := clustersmgmtv1.MarshalCluster(v, w)
		if err != nil {
			return fmt.Errorf("marshalling Cluster: %w", err)
		}
		return nil
	// handle cluster status types
	case *clustersmgmtv1.ClusterStatus:
		err := clustersmgmtv1.MarshalClusterStatus(v, w)
		if err != nil {
			return fmt.Errorf("marshalling ClusterStatus: %w", err)
		}
		return nil
	// handle syncset types
	case *clustersmgmtv1.Syncset:
		err := clustersmgmtv1.MarshalSyncset(v, w)
		if err != nil {
			return fmt.Errorf("marshalling Syncset: %w", err)
		}
		return nil
	// handle identiy provider types
	case *clustersmgmtv1.IdentityProvider:
		err := clustersmgmtv1.MarshalIdentityProvider(v, w)
		if err != nil {
			return fmt.Errorf("marshalling IdentityProvider: %w", err)
		}
		return nil
	// handle ingress types
	case *clustersmgmtv1.Ingress:
		err := clustersmgmtv1.MarshalIngress(v, w)
		if err != nil {
			return fmt.Errorf("marshalling Ingress: %w", err)
		}
		return nil
	case []*clustersmgmtv1.Ingress:
		err := clustersmgmtv1.MarshalIngressList(v, w)
		if err != nil {
			return fmt.Errorf("marshalling IngressList: %w", err)
		}
		return nil
	// for any <type>List ocm type we'll need to follow this pattern to ensure the array of objects
	// is wrapped with an OCMList object
	case *clustersmgmtv1.IngressList:
		ocmList, err := NewOCMList().WithItems(v.Slice())
		if err != nil {
			return err
		}
		err = json.NewEncoder(w).Encode(ocmList)
		if err != nil {
			return fmt.Errorf("encoding IngressList: %w", err)
		}
		return nil
	// handle cloud provider types
	case *clustersmgmtv1.CloudProvider:
		err := clustersmgmtv1.MarshalCloudProvider(v, w)
		if err != nil {
			return fmt.Errorf("marshalling CloudProvider: %w", err)
		}
		return nil
	case []*clustersmgmtv1.CloudProvider:
		err := clustersmgmtv1.MarshalCloudProviderList(v, w)
		if err != nil {
			return fmt.Errorf("marshalling CloudProviderList: %w", err)
		}
		return nil
	case *clustersmgmtv1.CloudProviderList:
		ocmList, err := NewOCMList().WithItems(v.Slice())
		if err != nil {
			return err
		}
		err = json.NewEncoder(w).Encode(ocmList)
		if err != nil {
			return fmt.Errorf("encoding CloudProviderList: %w", err)
		}
		return nil
	// handle cloud region types
	case *clustersmgmtv1.CloudRegion:
		err := clustersmgmtv1.MarshalCloudRegion(v, w)
		if err != nil {
			return fmt.Errorf("marshalling CloudRegion: %w", err)
		}
		return nil
	case []*clustersmgmtv1.CloudRegion:
		err := clustersmgmtv1.MarshalCloudRegionList(v, w)
		if err != nil {
			return fmt.Errorf("marshalling CloudRegionList: %w", err)
		}
		return nil
	case *clustersmgmtv1.CloudRegionList:
		ocmList, err := NewOCMList().WithItems(v.Slice())
		if err != nil {
			return err
		}
		err = json.NewEncoder(w).Encode(ocmList)
		if err != nil {
			return fmt.Errorf("encoding CloudRegionList: %w", err)
		}
		return nil
	// handle cluster addon installations
	case *clustersmgmtv1.AddOnInstallation:
		err := clustersmgmtv1.MarshalAddOnInstallation(v, w)
		if err != nil {
			return fmt.Errorf("marshalling AddOnInstallation: %w", err)
		}
		return nil
	case []*clustersmgmtv1.AddOnInstallation:
		err := clustersmgmtv1.MarshalAddOnInstallationList(v, w)
		if err != nil {
			return fmt.Errorf("marshalling AddOnInstallationList: %w", err)
		}
		return nil
	case *clustersmgmtv1.AddOnInstallationList:
		ocmList, err := NewOCMList().WithItems(v.Slice())
		if err != nil {
			return err
		}
		err = json.NewEncoder(w).Encode(ocmList)
		if err != nil {
			return fmt.Errorf("encoding AddOnInstallationList: %w", err)
		}
		return nil
	case *clustersmgmtv1.MachinePool:
		err := clustersmgmtv1.MarshalMachinePool(v, w)
		if err != nil {
			return fmt.Errorf("marshalling MachinePool: %w", err)
		}
		return nil
	case []*clustersmgmtv1.MachinePool:
		err := clustersmgmtv1.MarshalMachinePoolList(v, w)
		if err != nil {
			return fmt.Errorf("marshalling MachinePoolList: %w", err)
		}
		return nil
	case *clustersmgmtv1.MachinePoolList:
		ocmList, err := NewOCMList().WithItems(v.Slice())
		if err != nil {
			return err
		}
		err = json.NewEncoder(w).Encode(ocmList)
		if err != nil {
			return fmt.Errorf("encoding MachinePoolList: %w", err)
		}
		return nil
	// handle the generic ocm list type
	case *ocmList:
		err := json.NewEncoder(w).Encode(t)
		if err != nil {
			return fmt.Errorf("encoding ocm list: %w", err)
		}
		return nil
	case *amsv1.ClusterAuthorizationResponse:
		err := amsv1.MarshalClusterAuthorizationResponse(v, w)
		if err != nil {
			return fmt.Errorf("marshalling ClusterAuthorizationResponse: %w", err)
		}
		return nil
	case *amsv1.Subscription:
		err := amsv1.MarshalSubscription(t.(*amsv1.Subscription), w)
		if err != nil {
			return fmt.Errorf("marshalling Subscription: %w", err)
		}
		return nil
	case *authorizationsv1.TermsReviewResponse:
		err := authorizationsv1.MarshalTermsReviewResponse(v, w)
		if err != nil {
			return fmt.Errorf("marshalling TermsReviewResponse: %w", err)
		}
		return nil
	case []*amsv1.Subscription:
		err := amsv1.MarshalSubscriptionList(v, w)
		if err != nil {
			return fmt.Errorf("marshalling SubscriptionList: %w", err)
		}
		return nil
	case *amsv1.SubscriptionList:
		subscList, err := NewSubscriptionList().WithItems(v.Slice())
		if err != nil {
			return err
		}
		err = json.NewEncoder(w).Encode(subscList)
		if err != nil {
			return fmt.Errorf("encoding SubscriptionList: %w", err)
		}
		return nil
		// list := t.(*amsv1.SubscriptionList)
		// return amsv1.MarshalSubscriptionList(list.Slice(), w)
	// handle ocm error type
	case *ocmErrors.ServiceError:
		err := json.NewEncoder(w).Encode(v.AsOpenapiError("", ""))
		if err != nil {
			return fmt.Errorf("encoding ServiceError: %w", err)
		}
		return nil
	}
	return fmt.Errorf("could not recognise type %s in ocm type marshaller", reflect.TypeOf(t).String())
}

// basic wrapper to emulate the the ocm list types as they're private
type ocmList struct {
	HREF  *string         `json:"href"`
	Link  bool            `json:"link"`
	Items json.RawMessage `json:"items"`
}

// NewOCMList ...
func NewOCMList() *ocmList {
	return &ocmList{
		HREF:  nil,
		Link:  false,
		Items: nil,
	}
}

// WithHREF ...
func (l *ocmList) WithHREF(href string) *ocmList {
	l.HREF = &href
	return l
}

// WithLink ...
func (l *ocmList) WithLink(link bool) *ocmList {
	l.Link = link
	return l
}

// WithItems ...
func (l *ocmList) WithItems(items interface{}) (*ocmList, error) {
	var b bytes.Buffer
	if err := marshalOCMType(items, &b); err != nil {
		return l, err
	}
	l.Items = b.Bytes()
	return l, nil
}

type subscriptionList struct {
	Page  int             `json:"page"`
	Size  int             `json:"size"`
	Total int             `json:"total"`
	Items json.RawMessage `json:"items"`
}

// WithItems ...
func (l *subscriptionList) WithItems(items interface{}) (*subscriptionList, error) {
	var b bytes.Buffer
	if err := marshalOCMType(items, &b); err != nil {
		return l, err
	}
	l.Items = b.Bytes()
	return l, nil
}

// NewSubscriptionList ...
func NewSubscriptionList() *subscriptionList {
	return &subscriptionList{
		Page:  0,
		Size:  0,
		Total: 0,
		Items: nil,
	}
}

// init the shared mock types, panic if we fail, this should never fail
func init() {
	var err error
	// mock syncsets
	mockMockSyncsetBuilder := GetMockSyncsetBuilder(nil)
	MockSyncset, err = GetMockSyncset(mockMockSyncsetBuilder)
	if err != nil {
		panic(err)
	}

	// mock ingresses
	MockIngressList, err = GetMockIngressList(nil)
	if err != nil {
		panic(err)
	}

	// mock cloud providers
	MockCloudProvider, err = GetMockCloudProvider(nil)
	if err != nil {
		panic(err)
	}
	MockCloudProviderList, err = GetMockCloudProviderList(nil)
	if err != nil {
		panic(err)
	}

	// mock cloud provider regions/cloud regions
	MockCloudProviderRegion, err = GetMockCloudProviderRegion(nil)
	if err != nil {
		panic(err)
	}
	MockCloudProviderRegionList, err = GetMockCloudProviderRegionList(nil)
	if err != nil {
		panic(err)
	}

	// mock cluster status
	MockClusterStatus, err = GetMockClusterStatus(nil)
	if err != nil {
		panic(err)
	}
	MockClusterAddonInstallation, err = GetMockClusterAddonInstallation(nil, "")
	if err != nil {
		panic(err)
	}
	MockClusterAddonInstallationList, err = GetMockClusterAddonInstallationList(nil)
	if err != nil {
		panic(err)
	}
	MockFleetshardOperatorAddonInstallation, err = GetMockClusterAddonInstallation(nil, MockFleetshardAddonID)
	if err != nil {
		panic(err)
	}
	MockClusterLoggingOperatorAddonInstallation, err = GetMockClusterAddonInstallation(nil, MockClusterLoggingOperatorAddonID)
	if err != nil {
		panic(err)
	}
	MockCluster, err = GetMockCluster(nil)
	if err != nil {
		panic(err)
	}

	// Mock machine pools
	MockMachinePoolList, err = GetMachinePoolList(nil)
	if err != nil {
		panic(err)
	}
	MockMachinePool, err = GetMockMachinePool(nil)
	if err != nil {
		panic(err)
	}

	// Identity provider
	MockIdentityProvider, err = GetMockIdentityProvider(nil)
	if err != nil {
		panic(err)
	}

	MockClusterAuthorization, err = GetMockClusterAuthorization(nil)
	if err != nil {
		panic(err)
	}
	MockSubscription, err = GetMockSubscription(nil)
	if err != nil {
		panic(err)
	}
}

// GetMockSubscription ...
func GetMockSubscription(modifyFn func(b *amsv1.Subscription)) (*amsv1.Subscription, error) {
	builder, err := amsv1.NewSubscription().ID(MockSubID).Build()
	if modifyFn != nil {
		modifyFn(builder)
	}
	if err != nil {
		return builder, fmt.Errorf("building Subscription: %w", err)
	}
	return builder, nil
}

// GetMockClusterAuthorization ...
func GetMockClusterAuthorization(modifyFn func(b *amsv1.ClusterAuthorizationResponse)) (*amsv1.ClusterAuthorizationResponse, error) {
	sub := amsv1.SubscriptionBuilder{}
	sub.ID(MockSubID)
	sub.ClusterID(MockClusterExternalID)
	sub.Status("Active")
	builder, err := amsv1.NewClusterAuthorizationResponse().Subscription(&sub).Allowed(true).Build()
	if modifyFn != nil {
		modifyFn(builder)
	}

	if err != nil {
		return builder, fmt.Errorf("building ClusterAuthorizationResponse: %w", err)
	}
	return builder, nil
}

// GetMockTermsReview ...
func GetMockTermsReview(modifyFn func(b *authorizationsv1.TermsReviewResponse)) (*authorizationsv1.TermsReviewResponse, error) {
	resp, err := authorizationsv1.NewTermsReviewResponse().TermsRequired(true).Build()
	if err != nil {
		return resp, fmt.Errorf("building TermsReviewResponse: %w", err)
	}
	return resp, nil
}

// GetMockSyncsetBuilder for emulated OCM server
func GetMockSyncsetBuilder(modifyFn func(b *clustersmgmtv1.SyncsetBuilder)) *clustersmgmtv1.SyncsetBuilder {
	builder := clustersmgmtv1.NewSyncset().
		ID(MockSyncsetID).
		HREF(fmt.Sprintf("/api/clusters_mgmt/v1/clusters/%s/external_configuration/syncsets/%s", MockClusterID, MockSyncsetID))

	if modifyFn != nil {
		modifyFn(builder)
	}
	return builder
}

// GetMockSyncset for emulated OCM server
func GetMockSyncset(syncsetBuilder *clustersmgmtv1.SyncsetBuilder) (*clustersmgmtv1.Syncset, error) {
	s, err := syncsetBuilder.Build()
	if err != nil {
		return s, fmt.Errorf("building Syncset: %w", err)
	}
	return s, nil
}

// GetMockIngressList for emulated OCM server
func GetMockIngressList(modifyFn func(l *v1.IngressList, err error)) (*clustersmgmtv1.IngressList, error) {
	list, err := clustersmgmtv1.NewIngressList().Items(
		clustersmgmtv1.NewIngress().ID(MockIngressID).DNSName(MockIngressDNS).Default(true).Listening(MockIngressListening).HREF(MockIngressHref)).Build()

	if modifyFn != nil {
		modifyFn(list, err)
	}
	if err != nil {
		return list, fmt.Errorf("building IngressList: %w", err)
	}
	return list, nil
}

// GetMockCloudProviderBuilder for emulated OCM server
func GetMockCloudProviderBuilder(modifyFn func(builder *clustersmgmtv1.CloudProviderBuilder)) *clustersmgmtv1.CloudProviderBuilder {
	builder := clustersmgmtv1.NewCloudProvider().
		ID(MockCloudProviderID).
		Name(MockCloudProviderID).
		DisplayName(MockCloudProviderDisplayName).
		HREF(fmt.Sprintf("/api/clusters_mgmt/v1/cloud_providers/%s", MockCloudProviderID))

	if modifyFn != nil {
		modifyFn(builder)
	}
	return builder
}

// GetMockCloudProvider for emulated OCM server
func GetMockCloudProvider(modifyFn func(*clustersmgmtv1.CloudProvider, error)) (*clustersmgmtv1.CloudProvider, error) {
	cloudProvider, err := GetMockCloudProviderBuilder(nil).Build()
	if modifyFn != nil {
		modifyFn(cloudProvider, err)
	}
	if err != nil {
		return cloudProvider, fmt.Errorf("building CloudProvier: %w", err)
	}
	return cloudProvider, nil
}

// GetMockCloudProviderList for emulated OCM server
func GetMockCloudProviderList(modifyFn func(*clustersmgmtv1.CloudProviderList, error)) (*clustersmgmtv1.CloudProviderList, error) {
	list, err := clustersmgmtv1.NewCloudProviderList().
		Items(GetMockCloudProviderBuilder(nil)).
		Build()
	if modifyFn != nil {
		modifyFn(list, err)
	}
	if err != nil {
		return list, fmt.Errorf("building CloudProviderList: %w", err)
	}
	return list, nil
}

// GetMockCloudProviderRegionBuilder for emulated OCM server
func GetMockCloudProviderRegionBuilder(modifyFn func(*clustersmgmtv1.CloudRegionBuilder)) *clustersmgmtv1.CloudRegionBuilder {
	builder := clustersmgmtv1.NewCloudRegion().
		ID(MockCloudRegionID).
		HREF(fmt.Sprintf("/api/clusters_mgmt/v1/cloud_providers/%s/regions/%s", MockCloudProviderID, MockCloudRegionID)).
		DisplayName(MockCloudRegionDisplayName).
		CloudProvider(GetMockCloudProviderBuilder(nil)).
		Enabled(true).
		SupportsMultiAZ(true)

	if modifyFn != nil {
		modifyFn(builder)
	}
	return builder
}

// GetMockCloudProviderRegion for emulated OCM server
func GetMockCloudProviderRegion(modifyFn func(*clustersmgmtv1.CloudRegion, error)) (*clustersmgmtv1.CloudRegion, error) {
	cloudRegion, err := GetMockCloudProviderRegionBuilder(nil).Build()
	if modifyFn != nil {
		modifyFn(cloudRegion, err)
	}
	if err != nil {
		return cloudRegion, fmt.Errorf("building CloudRegion: %w", err)
	}
	return cloudRegion, nil
}

// GetMockCloudProviderRegionList for emulated OCM server
func GetMockCloudProviderRegionList(modifyFn func(*clustersmgmtv1.CloudRegionList, error)) (*clustersmgmtv1.CloudRegionList, error) {
	list, err := clustersmgmtv1.NewCloudRegionList().Items(GetMockCloudProviderRegionBuilder(nil)).Build()
	if modifyFn != nil {
		modifyFn(list, err)
	}
	if err != nil {
		return list, fmt.Errorf("building CloudRegionList: %w", err)
	}
	return list, nil
}

// GetMockClusterStatus for emulated OCM server
func GetMockClusterStatus(modifyFn func(*clustersmgmtv1.ClusterStatus, error)) (*clustersmgmtv1.ClusterStatus, error) {
	status, err := GetMockClusterStatusBuilder(nil).Build()
	if modifyFn != nil {
		modifyFn(status, err)
	}
	if err != nil {
		return status, fmt.Errorf("building ClusterStatus: %w", err)
	}
	return status, nil
}

// GetMockClusterStatusBuilder for emulated OCM server
func GetMockClusterStatusBuilder(modifyFn func(*clustersmgmtv1.ClusterStatusBuilder)) *clustersmgmtv1.ClusterStatusBuilder {
	builder := clustersmgmtv1.NewClusterStatus().
		ID(MockClusterID).
		HREF(fmt.Sprintf("/api/clusters_mgmt/v1/clusters/%s/status", MockClusterID)).
		State(MockClusterState).
		Description("")
	if modifyFn != nil {
		modifyFn(builder)
	}
	return builder
}

// GetMockClusterAddonBuilder for emulated OCM server
func GetMockClusterAddonBuilder(modifyFn func(*clustersmgmtv1.AddOnBuilder), addonID string) *clustersmgmtv1.AddOnBuilder {
	if addonID == "" {
		addonID = MockClusterAddonID
	}

	builder := clustersmgmtv1.NewAddOn().
		ID(addonID).
		HREF(fmt.Sprintf("/api/clusters_mgmt/v1/addons/%s", addonID))
	if modifyFn != nil {
		modifyFn(builder)
	}
	return builder
}

// GetMockClusterAddonInstallationBuilder for emulated OCM server
func GetMockClusterAddonInstallationBuilder(modifyFn func(*clustersmgmtv1.AddOnInstallationBuilder), addonID string) *clustersmgmtv1.AddOnInstallationBuilder {
	if addonID == "" {
		addonID = MockClusterAddonID
	}
	addonInstallation := clustersmgmtv1.NewAddOnInstallation().
		ID(addonID).
		HREF(fmt.Sprintf("/api/clusters_mgmt/v1/clusters/%s/addons/%s", MockClusterID, addonID)).
		Addon(GetMockClusterAddonBuilder(nil, addonID)).
		State(MockClusterAddonState).
		StateDescription(MockClusterAddonDescription)

	if modifyFn != nil {
		modifyFn(addonInstallation)
	}
	return addonInstallation
}

// GetMockClusterAddonInstallation for emulated OCM server
func GetMockClusterAddonInstallation(modifyFn func(*clustersmgmtv1.AddOnInstallation, error), addonID string) (*clustersmgmtv1.AddOnInstallation, error) {
	addonInstall, err := GetMockClusterAddonInstallationBuilder(nil, addonID).Build()
	if modifyFn != nil {
		modifyFn(addonInstall, err)
	}
	if err != nil {
		return addonInstall, fmt.Errorf("building AddOnInstallation: %w", err)
	}
	return addonInstall, nil
}

// GetMockClusterAddonInstallationList for emulated OCM server
func GetMockClusterAddonInstallationList(modifyFn func(*clustersmgmtv1.AddOnInstallationList, error)) (*clustersmgmtv1.AddOnInstallationList, error) {
	list, err := clustersmgmtv1.NewAddOnInstallationList().Items(
		GetMockClusterAddonInstallationBuilder(nil, MockClusterAddonID),
		GetMockClusterAddonInstallationBuilder(nil, MockClusterLoggingOperatorAddonID),
		GetMockClusterAddonInstallationBuilder(nil, MockFleetshardAddonID)).
		Build()
	if modifyFn != nil {
		modifyFn(list, err)
	}
	if err != nil {
		return list, fmt.Errorf("building AddOnInstallationList: %w", err)
	}
	return list, nil
}

// GetMockClusterNodesBuilder for emulated OCM server
func GetMockClusterNodesBuilder(modifyFn func(*clustersmgmtv1.ClusterNodesBuilder)) *clustersmgmtv1.ClusterNodesBuilder {
	builder := clustersmgmtv1.NewClusterNodes().
		Compute(MockClusterComputeNodes).
		ComputeMachineType(clustersmgmtv1.NewMachineType().ID("m5.2xlarge"))
	if modifyFn != nil {
		modifyFn(builder)
	}
	return builder
}

// GetMockClusterBuilder for emulated OCM server
func GetMockClusterBuilder(modifyFn func(*clustersmgmtv1.ClusterBuilder)) *clustersmgmtv1.ClusterBuilder {
	mockClusterStatusBuilder := GetMockClusterStatusBuilder(nil)
	builder := clustersmgmtv1.NewCluster().
		ID(MockClusterID).
		ExternalID(MockClusterExternalID).
		State(MockClusterState).
		Status(mockClusterStatusBuilder).
		MultiAZ(MockMultiAZ).
		Nodes(GetMockClusterNodesBuilder(nil)).
		CloudProvider(GetMockCloudProviderBuilder(nil)).
		Region(GetMockCloudProviderRegionBuilder(nil)).
		Version(GetMockOpenshiftVersionBuilder(nil))
	if modifyFn != nil {
		modifyFn(builder)
	}
	return builder
}

// GetMockTermsReviewBuilder ...
func GetMockTermsReviewBuilder(modifyFn func(builder *authorizationsv1.TermsReviewResponseBuilder)) *authorizationsv1.TermsReviewResponseBuilder {
	builder := authorizationsv1.NewTermsReviewResponse()
	if modifyFn != nil {
		modifyFn(builder)
	}
	return builder
}

// GetMockCluster for emulated OCM server
func GetMockCluster(modifyFn func(*clustersmgmtv1.Cluster, error)) (*clustersmgmtv1.Cluster, error) {
	cluster, err := GetMockClusterBuilder(nil).Build()
	if modifyFn != nil {
		modifyFn(cluster, err)
	}
	if err != nil {
		return cluster, fmt.Errorf("building Cluster: %w", err)
	}
	return cluster, nil
}

// GetMockMachineBuilder for emulated OCM server
func GetMockMachineBuilder(modifyFn func(*clustersmgmtv1.MachinePoolBuilder)) *clustersmgmtv1.MachinePoolBuilder {
	builder := clustersmgmtv1.NewMachinePool().
		ID(MockMachinePoolID).
		HREF(fmt.Sprintf("/api/clusters_mgmt/v1/clusters/%s/machine_pools/%s", MockClusterID, MockMachinePoolID)).
		Replicas(MockMachinePoolReplicas).
		Cluster(GetMockClusterBuilder(nil))
	if modifyFn != nil {
		modifyFn(builder)
	}
	return builder
}

// GetMachinePoolList for emulated OCM server
func GetMachinePoolList(modifyFn func(*clustersmgmtv1.MachinePoolList, error)) (*clustersmgmtv1.MachinePoolList, error) {
	list, err := clustersmgmtv1.NewMachinePoolList().Items(GetMockMachineBuilder(nil)).Build()
	if modifyFn != nil {
		modifyFn(list, err)
	}
	if err != nil {
		return list, fmt.Errorf("building MachinePoolList: %w", err)
	}
	return list, nil
}

// GetMockMachinePool for emulated OCM server
func GetMockMachinePool(modifyFn func(*clustersmgmtv1.MachinePool, error)) (*clustersmgmtv1.MachinePool, error) {
	machinePool, err := GetMockMachineBuilder(nil).Build()
	if modifyFn != nil {
		modifyFn(machinePool, err)
	}
	if err != nil {
		return machinePool, fmt.Errorf("building MachinePool: %w", err)
	}
	return machinePool, nil
}

// GetMockOpenshiftVersionBuilder for emulated OCM server
func GetMockOpenshiftVersionBuilder(modifyFn func(*clustersmgmtv1.VersionBuilder)) *clustersmgmtv1.VersionBuilder {
	builder := clustersmgmtv1.NewVersion().ID(MockOpenshiftVersion)
	if modifyFn != nil {
		modifyFn(builder)
	}
	return builder
}

// GetMockIdentityProviderBuilder for emulated OCM server
func GetMockIdentityProviderBuilder(modifyFn func(*clustersmgmtv1.IdentityProviderBuilder)) *clustersmgmtv1.IdentityProviderBuilder {
	builder := clustersmgmtv1.NewIdentityProvider().
		ID(MockIdentityProviderID).
		HREF(fmt.Sprintf("/api/clusters_mgmt/v1/clusters/%s/identity_providers/%s", MockClusterID, MockIdentityProviderID)).
		OpenID(clustersmgmtv1.NewOpenIDIdentityProvider())
	if modifyFn != nil {
		modifyFn(builder)
	}
	return builder
}

// GetMockIdentityProvider for emulated OCM server
func GetMockIdentityProvider(modifyFn func(*clustersmgmtv1.IdentityProvider, error)) (*clustersmgmtv1.IdentityProvider, error) {
	identityProvider, err := GetMockIdentityProviderBuilder(nil).Build()
	if modifyFn != nil {
		modifyFn(identityProvider, err)
	}
	if err != nil {
		return identityProvider, fmt.Errorf("building IdentityProvider: %w", err)
	}
	return identityProvider, nil
}
