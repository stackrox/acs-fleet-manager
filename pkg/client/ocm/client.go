package ocm

import (
	sdkClient "github.com/openshift-online/ocm-sdk-go"
	amsv1 "github.com/openshift-online/ocm-sdk-go/accountsmgmt/v1"
	clustersmgmtv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	serviceErrors "github.com/stackrox/acs-fleet-manager/pkg/errors"
)

//go:generate moq -rm -out mocks/client_moq.go -pkg mocks . Client

// Client is an interface to OCM
type Client interface {
	CreateCluster(cluster *clustersmgmtv1.Cluster) (*clustersmgmtv1.Cluster, error)
	GetClusterIngresses(clusterID string) (*clustersmgmtv1.IngressesListResponse, error)
	GetCluster(clusterID string) (*clustersmgmtv1.Cluster, error)
	GetClusterStatus(id string) (*clustersmgmtv1.ClusterStatus, error)
	GetCloudProviders() (*clustersmgmtv1.CloudProviderList, error)
	GetRegions(provider *clustersmgmtv1.CloudProvider) (*clustersmgmtv1.CloudRegionList, error)
	GetAddonInstallation(clusterID string, addonID string) (*clustersmgmtv1.AddOnInstallation, *serviceErrors.ServiceError)
	CreateAddonInstallation(clusterID string, addon *clustersmgmtv1.AddOnInstallation) error
	UpdateAddonInstallation(clusterID string, addon *clustersmgmtv1.AddOnInstallation) error
	DeleteAddonInstallation(clusterID string, addonID string) error
	GetAddon(addonID string) (*clustersmgmtv1.AddOn, error)
	GetAddonVersion(addonID string, version string) (*clustersmgmtv1.AddOnVersion, error)
	GetClusterDNS(clusterID string) (string, error)
	CreateIdentityProvider(clusterID string, identityProvider *clustersmgmtv1.IdentityProvider) (*clustersmgmtv1.IdentityProvider, error)
	DeleteCluster(clusterID string) (int, error)
	ClusterAuthorization(cb *amsv1.ClusterAuthorizationRequest) (*amsv1.ClusterAuthorizationResponse, error)
	DeleteSubscription(id string) (int, error)
	FindSubscriptions(query string) (*amsv1.SubscriptionsListResponse, error)
	GetRequiresTermsAcceptance(username string) (termsRequired bool, redirectURL string, err error)
	GetExistingClusterMetrics(clusterID string) (*amsv1.SubscriptionMetrics, error)
	GetOrganisationFromExternalID(externalID string) (*amsv1.Organization, error)
	Connection() *sdkClient.Connection
	GetQuotaCostsForProduct(organizationID, resourceName, product string) ([]*amsv1.QuotaCost, error)
	GetCustomerCloudAccounts(organizationID string, quotaIDs []string) ([]*amsv1.CloudAccount, error)
	// GetCurrentAccount returns the account information of the user to whom belongs the token
	GetCurrentAccount(userToken string) (int, *amsv1.Account, error)
}
