// Package ocm ...
package ocm

import (
	"fmt"
	"net/http"

	"github.com/golang/glog"
	sdkClient "github.com/openshift-online/ocm-sdk-go"
	amsv1 "github.com/openshift-online/ocm-sdk-go/accountsmgmt/v1"
	addonsmgmtv1 "github.com/openshift-online/ocm-sdk-go/addonsmgmt/v1"
	v1 "github.com/openshift-online/ocm-sdk-go/authorizations/v1"
	clustersmgmtv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	"github.com/openshift-online/ocm-sdk-go/logging"
	pkgerrors "github.com/pkg/errors"
	serviceErrors "github.com/stackrox/acs-fleet-manager/pkg/errors"
)

// TermsSitecode ...
const TermsSitecode = "OCM"

// TermsEventcodeOnlineService ...
const TermsEventcodeOnlineService = "onlineService"

// TermsEventcodeRegister ...
const TermsEventcodeRegister = "register"

// Client ...
//
//go:generate moq -out client_moq.go . Client
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
	GetAddonVersion(addonID string, version string) (*addonsmgmtv1.AddonVersion, error)
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

var _ Client = &client{}

type client struct {
	connection *sdkClient.Connection
}

// AMSClient ...
type AMSClient Client

// ClusterManagementClient ...
type ClusterManagementClient Client

// NewOCMConnection ...
func NewOCMConnection(ocmConfig *OCMConfig, baseURL string) (*sdkClient.Connection, func(), error) {
	if ocmConfig.EnableMock && ocmConfig.MockMode != MockModeEmulateServer {
		return nil, func() {}, nil
	}

	builder := getBaseConnectionBuilder(baseURL)
	if !ocmConfig.EnableMock {
		// Create a logger that has the debug level enabled:
		logger, err := getLogger(ocmConfig.Debug)
		if err != nil {
			return nil, nil, err
		}
		builder = builder.Logger(logger)
	}

	if ocmConfig.ClientID != "" && ocmConfig.ClientSecret != "" {
		builder = builder.Client(ocmConfig.ClientID, ocmConfig.ClientSecret)
	} else if ocmConfig.SelfToken != "" {
		builder = builder.Tokens(ocmConfig.SelfToken)
	} else {
		return nil, nil, pkgerrors.New("Can't build OCM client connection. No Client/Secret or Token has been provided.")
	}

	connection, err := builder.Build()
	if err != nil {
		return nil, nil, fmt.Errorf("building OCM client connection: %w", err)
	}
	return connection, func() {
		_ = connection.Close()
	}, nil
}

func getBaseConnectionBuilder(baseURL string) *sdkClient.ConnectionBuilder {
	return sdkClient.NewConnectionBuilder().
		URL(baseURL).
		MetricsSubsystem("api_outbound")
}

func getLogger(isDebugEnabled bool) (*logging.GoLogger, error) {
	logger, err := sdkClient.NewGoLoggerBuilder().
		Debug(isDebugEnabled).
		Build()
	if err != nil {
		return nil, fmt.Errorf("creating logger for OCM client connection: %w", err)
	}
	return logger, nil
}

// NewClient ...
func NewClient(connection *sdkClient.Connection) Client {
	return &client{connection: connection}
}

// NewMockClient returns a new OCM client with stubbed responses.
func NewMockClient() Client {
	return &ClientMock{
		GetOrganisationFromExternalIDFunc: func(externalID string) (*amsv1.Organization, error) {
			org, err := amsv1.NewOrganization().
				ID("12345678").
				Name("stubbed-name").
				Build()
			return org, pkgerrors.Wrap(err, "failed to build organisation")
		},
	}
}

// Connection ...
func (c *client) Connection() *sdkClient.Connection {
	return c.connection
}

// Close ...
func (c *client) Close() {
	if c.connection != nil {
		_ = c.connection.Close()
	}
}

// CreateCluster ...
func (c *client) CreateCluster(cluster *clustersmgmtv1.Cluster) (*clustersmgmtv1.Cluster, error) {
	if c.connection == nil {
		return nil, serviceErrors.InvalidOCMConnection()
	}

	clusterResource := c.connection.ClustersMgmt().V1().Clusters()
	response, err := clusterResource.Add().Body(cluster).Send()
	if err != nil {
		return &clustersmgmtv1.Cluster{}, serviceErrors.New(serviceErrors.ErrorGeneral, err.Error())
	}
	createdCluster := response.Body()

	return createdCluster, nil
}

// GetExistingClusterMetrics ...
func (c *client) GetExistingClusterMetrics(clusterID string) (*amsv1.SubscriptionMetrics, error) {
	if c.connection == nil {
		return nil, serviceErrors.InvalidOCMConnection()
	}

	subscriptions, err := c.connection.AccountsMgmt().V1().Subscriptions().List().Search(fmt.Sprintf("cluster_id='%s'", clusterID)).Send()
	if err != nil {
		return nil, fmt.Errorf("retrieving subscriptions: %w", err)
	}
	items := subscriptions.Items()
	if items == nil || items.Len() == 0 {
		return nil, nil
	}

	if items.Len() > 1 {
		return nil, fmt.Errorf("expected 1 subscription item, found %d", items.Len())
	}
	subscriptionsMetrics := subscriptions.Items().Get(0).Metrics()
	if len(subscriptionsMetrics) > 1 {
		// this should never happen: https://github.com/openshift-online/ocm-api-model/blob/9ca12df7763723903c0d1cd87e993995a2acda5f/model/accounts_mgmt/v1/subscription_type.model#L49-L50
		return nil, fmt.Errorf("expected 1 subscription metric, found %d", len(subscriptionsMetrics))
	}

	if len(subscriptionsMetrics) == 0 {
		return nil, nil
	}

	return subscriptionsMetrics[0], nil
}

// GetOrganisationFromExternalID takes the external org id as input, and returns the OCM org.
func (c *client) GetOrganisationFromExternalID(externalID string) (*amsv1.Organization, error) {
	if c.connection == nil {
		return nil, serviceErrors.InvalidOCMConnection()
	}

	request := c.connection.AccountsMgmt().V1().Organizations().List().Search(fmt.Sprintf("external_id='%s'", externalID))
	res, err := request.Send()
	if err != nil {
		return nil, fmt.Errorf("retrieving organizations: %w", err)
	}

	items := res.Items()
	if items.Len() < 1 {
		return nil, serviceErrors.New(serviceErrors.ErrorNotFound, "organisation with external id '%s' not found", externalID)
	}

	return items.Get(0), nil
}

// GetRequiresTermsAcceptance ...
func (c *client) GetRequiresTermsAcceptance(username string) (termsRequired bool, redirectURL string, err error) {
	if c.connection == nil {
		return false, "", serviceErrors.InvalidOCMConnection()
	}

	// Check for Appendix 4 Terms
	request, err := v1.NewTermsReviewRequest().AccountUsername(username).SiteCode(TermsSitecode).EventCode(TermsEventcodeRegister).Build()
	if err != nil {
		return false, "", fmt.Errorf("creating terms review request: %w", err)
	}
	selfTermsReview := c.connection.Authorizations().V1().TermsReview()
	postResp, err := selfTermsReview.Post().Request(request).Send()
	if err != nil {
		return false, "", fmt.Errorf("getting terms review: %w", err)
	}
	response, ok := postResp.GetResponse()
	if !ok {
		return false, "", fmt.Errorf("empty response from authorization post request")
	}

	redirectURL, _ = response.GetRedirectUrl()

	return response.TermsRequired(), redirectURL, nil
}

// GetClusterIngresses sends a GET request to ocm to retrieve the ingresses of an OSD cluster
func (c *client) GetClusterIngresses(clusterID string) (*clustersmgmtv1.IngressesListResponse, error) {
	if c.connection == nil {
		return nil, serviceErrors.InvalidOCMConnection()
	}

	clusterIngresses := c.connection.ClustersMgmt().V1().Clusters().Cluster(clusterID).Ingresses()
	ingressList, err := clusterIngresses.List().Send()
	if err != nil {
		return nil, fmt.Errorf("sending cluster ingresses list request: %w", err)
	}

	return ingressList, nil
}

// GetCluster ...
func (c *client) GetCluster(clusterID string) (*clustersmgmtv1.Cluster, error) {
	if c.connection == nil {
		return nil, serviceErrors.InvalidOCMConnection()
	}

	resp, err := c.connection.ClustersMgmt().V1().Clusters().Cluster(clusterID).Get().Send()
	if err != nil {
		return nil, fmt.Errorf("sending get cluster request: %w", err)
	}
	return resp.Body(), nil
}

// GetClusterStatus ...
func (c *client) GetClusterStatus(id string) (*clustersmgmtv1.ClusterStatus, error) {
	if c.connection == nil {
		return nil, serviceErrors.InvalidOCMConnection()
	}

	resp, err := c.connection.ClustersMgmt().V1().Clusters().Cluster(id).Status().Get().Send()
	if err != nil {
		return nil, fmt.Errorf("sending cluster status request: %w", err)
	}
	return resp.Body(), nil
}

// GetCloudProviders ...
func (c *client) GetCloudProviders() (*clustersmgmtv1.CloudProviderList, error) {
	if c.connection == nil {
		return nil, serviceErrors.InvalidOCMConnection()
	}

	providersCollection := c.connection.ClustersMgmt().V1().CloudProviders()
	providersResponse, err := providersCollection.List().Send()
	if err != nil {
		return nil, pkgerrors.Wrap(err, "error retrieving cloud provider list")
	}
	cloudProviderList := providersResponse.Items()
	return cloudProviderList, nil
}

// GetRegions ...
func (c *client) GetRegions(provider *clustersmgmtv1.CloudProvider) (*clustersmgmtv1.CloudRegionList, error) {
	if c.connection == nil {
		return nil, serviceErrors.InvalidOCMConnection()
	}

	regionsCollection := c.connection.ClustersMgmt().V1().CloudProviders().CloudProvider(provider.ID()).Regions()
	regionsResponse, err := regionsCollection.List().Send()
	if err != nil {
		return nil, pkgerrors.Wrap(err, "error retrieving cloud region list")
	}

	regionList := regionsResponse.Items()
	return regionList, nil
}

func (c *client) GetAddonVersion(addonID string, versionID string) (*addonsmgmtv1.AddonVersion, error) {
	if c.connection == nil {
		return nil, serviceErrors.InvalidOCMConnection()
	}

	resp, err := c.connection.AddonsMgmt().V1().Addons().Addon(addonID).Versions().Version(versionID).Get().Send()
	if err != nil {
		if resp != nil && resp.Status() == http.StatusNotFound {
			return nil, serviceErrors.NotFound("")
		}
		return nil, serviceErrors.GeneralError("sending GetAddon request: %v", err)
	}

	return resp.Body(), nil
}

// CreateAddonInstallation creates a new addon for a cluster with given ID
func (c *client) CreateAddonInstallation(clusterID string, addon *clustersmgmtv1.AddOnInstallation) error {
	if c.connection == nil {
		return serviceErrors.InvalidOCMConnection()
	}
	_, err := c.connection.ClustersMgmt().V1().Clusters().Cluster(clusterID).Addons().Add().Body(addon).Send()
	if err != nil {
		return fmt.Errorf("sending CreateAddonInstallation request: %w", err)
	}
	return nil
}

// GetAddonInstallation returns the addon installed on a cluster with given ID
func (c *client) GetAddonInstallation(clusterID string, addonID string) (*clustersmgmtv1.AddOnInstallation, *serviceErrors.ServiceError) {
	if c.connection == nil {
		return nil, serviceErrors.InvalidOCMConnection()
	}
	resp, err := c.connection.ClustersMgmt().V1().Clusters().Cluster(clusterID).Addons().Addoninstallation(addonID).Get().Send()
	if err != nil {
		if resp != nil && resp.Status() == http.StatusNotFound {
			return nil, serviceErrors.NotFound("")
		}
		return nil, serviceErrors.GeneralError("sending GetAddonInstallation request: %v", err)
	}

	return resp.Body(), nil
}

// UpdateAddonInstallation updates the existing addon on a cluster with given ID
func (c *client) UpdateAddonInstallation(clusterID string, addonInstallation *clustersmgmtv1.AddOnInstallation) error {
	if c.connection == nil {
		return serviceErrors.InvalidOCMConnection()
	}
	_, err := c.connection.ClustersMgmt().V1().Clusters().Cluster(clusterID).Addons().Addoninstallation(addonInstallation.ID()).Update().Body(addonInstallation).Send()
	if err != nil {
		return fmt.Errorf("sending UpdateAddonInstallation request: %w", err)
	}
	return nil
}

// DeleteAddonInstallation deletes the addon on a cluster with given ID
func (c *client) DeleteAddonInstallation(clusterID string, addonInstallationID string) error {
	if c.connection == nil {
		return serviceErrors.InvalidOCMConnection()
	}
	_, err := c.connection.ClustersMgmt().V1().Clusters().Cluster(clusterID).Addons().Addoninstallation(addonInstallationID).Delete().Send()
	if err != nil {
		return fmt.Errorf("sending DeleteAddonInstallation request: %w", err)
	}
	return nil
}

// GetClusterDNS ...
func (c *client) GetClusterDNS(clusterID string) (string, error) {
	if clusterID == "" {
		return "", serviceErrors.Validation("clusterID cannot be empty")
	}
	ingresses, err := c.GetClusterIngresses(clusterID)
	if err != nil {
		return "", err
	}

	var clusterDNS string
	ingresses.Items().Each(func(ingress *clustersmgmtv1.Ingress) bool {
		if ingress.Default() {
			clusterDNS = ingress.DNSName()
			return false
		}
		return true
	})

	if clusterDNS == "" {
		return "", serviceErrors.NotFound("Cluster %s: DNS is empty", clusterID)
	}

	return clusterDNS, nil
}

// CreateIdentityProvider ...
func (c *client) CreateIdentityProvider(clusterID string, identityProvider *clustersmgmtv1.IdentityProvider) (*clustersmgmtv1.IdentityProvider, error) {
	if c.connection == nil {
		return nil, serviceErrors.InvalidOCMConnection()
	}

	clustersResource := c.connection.ClustersMgmt().V1().Clusters()
	response, identityProviderErr := clustersResource.Cluster(clusterID).
		IdentityProviders().
		Add().
		Body(identityProvider).
		Send()
	var err error
	if identityProviderErr != nil {
		err = serviceErrors.NewErrorFromHTTPStatusCode(response.Status(), "ocm client failed to create identity provider: %s", identityProviderErr)
	}
	return response.Body(), err
}

// DeleteCluster ...
func (c *client) DeleteCluster(clusterID string) (int, error) {
	if c.connection == nil {
		return 0, serviceErrors.InvalidOCMConnection()
	}

	clustersResource := c.connection.ClustersMgmt().V1().Clusters()
	response, deleteClusterError := clustersResource.Cluster(clusterID).Delete().Send()

	var err error
	if deleteClusterError != nil {
		err = serviceErrors.NewErrorFromHTTPStatusCode(response.Status(), "OCM client failed to delete cluster '%s': %s", clusterID, deleteClusterError)
	}
	return response.Status(), err
}

// ClusterAuthorization ...
func (c *client) ClusterAuthorization(cb *amsv1.ClusterAuthorizationRequest) (*amsv1.ClusterAuthorizationResponse, error) {
	if c.connection == nil {
		return nil, serviceErrors.InvalidOCMConnection()
	}

	glog.V(10).Infof("Sending request to OCM '%v'", *cb)

	r, err := c.connection.AccountsMgmt().V1().
		ClusterAuthorizations().
		Post().Request(cb).Send()
	if err != nil && r.Status() != http.StatusTooManyRequests {
		glog.Warningf("OCM client responded with '%v: %v' for request '%v'", r.Status(), err, *cb)
		return nil, serviceErrors.NewErrorFromHTTPStatusCode(r.Status(), "OCM client failed to create cluster authorization")
	}
	resp, _ := r.GetResponse()
	return resp, nil
}

// DeleteSubscription ...
func (c *client) DeleteSubscription(id string) (int, error) {
	if c.connection == nil {
		return 0, serviceErrors.InvalidOCMConnection()
	}

	r := c.connection.AccountsMgmt().V1().Subscriptions().Subscription(id).Delete()
	resp, err := r.Send()
	return resp.Status(), err
}

// FindSubscriptions ...
func (c *client) FindSubscriptions(query string) (*amsv1.SubscriptionsListResponse, error) {
	if c.connection == nil {
		return nil, serviceErrors.InvalidOCMConnection()
	}

	r, err := c.connection.AccountsMgmt().V1().Subscriptions().List().Search(query).Send()
	if err != nil {
		return nil, fmt.Errorf("querying the accounts management service for subscriptions: %w", err)
	}
	return r, nil
}

// GetQuotaCostsForProduct gets the AMS QuotaCosts in the given organizationID
// whose relatedResources contains at least a relatedResource that has the
// given resourceName and product
func (c *client) GetQuotaCostsForProduct(organizationID, resourceName, product string) ([]*amsv1.QuotaCost, error) {
	if c.connection == nil {
		return nil, serviceErrors.InvalidOCMConnection()
	}

	var res []*amsv1.QuotaCost
	organizationClient := c.connection.AccountsMgmt().V1().Organizations()
	quotaCostClient := organizationClient.Organization(organizationID).QuotaCost()

	req := quotaCostClient.List().Parameter("fetchRelatedResources", true).Parameter("fetchCloudAccounts", true)

	// TODO: go 1.21 can infer the following generic arguments from req
	//       automatically, so this indirection becomes unnecessary.
	fetchQuotaCosts := fetchPages[*amsv1.QuotaCostListRequest,
		*amsv1.QuotaCostListResponse,
		*amsv1.QuotaCostList,
		*amsv1.QuotaCost,
	]
	err := fetchQuotaCosts(req, 100, 1000, func(qc *amsv1.QuotaCost) bool {
		relatedResourcesList := qc.RelatedResources()
		for _, relatedResource := range relatedResourcesList {
			if relatedResource.ResourceName() == resourceName && relatedResource.Product() == product {
				res = append(res, qc)
				break
			}
		}
		return true
	})
	if err != nil {
		return nil, pkgerrors.Wrap(err, "error listing QuotaCosts")
	}
	return res, nil
}

func (c *client) GetCustomerCloudAccounts(organizationID string, quotaIDs []string) ([]*amsv1.CloudAccount, error) {
	if c.connection == nil {
		return nil, serviceErrors.InvalidOCMConnection()
	}

	var res []*amsv1.CloudAccount
	organizationClient := c.connection.AccountsMgmt().V1().Organizations()
	quotaCostClient := organizationClient.Organization(organizationID).QuotaCost()

	quotaCostList, err := quotaCostClient.List().Parameter("fetchCloudAccounts", true).Send()
	if err != nil {
		return nil, fmt.Errorf("error getting cloud accounts: %w", err)
	}

	quotaCostList.Items().Each(func(qc *amsv1.QuotaCost) bool {
		for _, quotaID := range quotaIDs {
			if qc.QuotaID() == quotaID {
				res = append(res, qc.CloudAccounts()...)
				break
			}
		}
		return true
	})

	return res, nil
}

// GetCurrentAccount returns the account information of the user to whom belongs the token
func (c *client) GetCurrentAccount(userToken string) (int, *amsv1.Account, error) {
	logger, err := getLogger(c.connection.Logger().DebugEnabled())
	if err != nil {
		return 0, nil, fmt.Errorf("couldn't create logger for modified OCM connection: %w", err)
	}
	modifiedConnection, err := getBaseConnectionBuilder(c.connection.URL()).
		Logger(logger).
		Tokens(userToken).
		Build()
	if err != nil {
		return 0, nil, fmt.Errorf("couldn't build modified OCM connection: %w", err)
	}
	defer modifiedConnection.Close()
	response, err := modifiedConnection.AccountsMgmt().V1().CurrentAccount().Get().Send()
	if err != nil {
		return response.Status(), nil, fmt.Errorf("unsuccessful call to current account endpoint: %w", err)
	}

	currentAccount := response.Body()
	return response.Status(), currentAccount, nil
}
