// Package ocm ...
package ocm

import (
	"fmt"
	"net/http"

	"github.com/pkg/errors"
	pkgerrors "github.com/pkg/errors"

	"github.com/golang/glog"
	sdkClient "github.com/openshift-online/ocm-sdk-go"
	amsv1 "github.com/openshift-online/ocm-sdk-go/accountsmgmt/v1"
	v1 "github.com/openshift-online/ocm-sdk-go/authorizations/v1"
	clustersmgmtv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
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
	GetAddon(clusterID string, addonID string) (*clustersmgmtv1.AddOnInstallation, error)
	CreateAddonWithParams(clusterID string, addonID string, parameters []Parameter) (*clustersmgmtv1.AddOnInstallation, error)
	CreateAddon(clusterID string, addonID string) (*clustersmgmtv1.AddOnInstallation, error)
	UpdateAddonParameters(clusterID string, addonID string, parameters []Parameter) (*clustersmgmtv1.AddOnInstallation, error)
	GetClusterDNS(clusterID string) (string, error)
	CreateSyncSet(clusterID string, syncset *clustersmgmtv1.Syncset) (*clustersmgmtv1.Syncset, error)
	UpdateSyncSet(clusterID string, syncSetID string, syncset *clustersmgmtv1.Syncset) (*clustersmgmtv1.Syncset, error)
	GetSyncSet(clusterID string, syncSetID string) (*clustersmgmtv1.Syncset, error)
	DeleteSyncSet(clusterID string, syncsetID string) (int, error)
	ScaleUpComputeNodes(clusterID string, increment int) (*clustersmgmtv1.Cluster, error)
	ScaleDownComputeNodes(clusterID string, decrement int) (*clustersmgmtv1.Cluster, error)
	SetComputeNodes(clusterID string, numNodes int) (*clustersmgmtv1.Cluster, error)
	CreateIdentityProvider(clusterID string, identityProvider *clustersmgmtv1.IdentityProvider) (*clustersmgmtv1.IdentityProvider, error)
	GetIdentityProviderList(clusterID string) (*clustersmgmtv1.IdentityProviderList, error)
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
func NewOCMConnection(ocmConfig *OCMConfig, BaseURL string) (*sdkClient.Connection, func(), error) {
	if ocmConfig.EnableMock && ocmConfig.MockMode != MockModeEmulateServer {
		return nil, func() {}, nil
	}

	builder := sdkClient.NewConnectionBuilder().
		URL(BaseURL).
		MetricsSubsystem("api_outbound")

	if !ocmConfig.EnableMock {
		// Create a logger that has the debug level enabled:
		logger, err := sdkClient.NewGoLoggerBuilder().
			Debug(ocmConfig.Debug).
			Build()
		if err != nil {
			return nil, nil, fmt.Errorf("creating logger for OCM client connection: %w", err)
		}
		builder = builder.Logger(logger)
	}

	if ocmConfig.ClientID != "" && ocmConfig.ClientSecret != "" {
		builder = builder.Client(ocmConfig.ClientID, ocmConfig.ClientSecret)
	} else if ocmConfig.SelfToken != "" {
		builder = builder.Tokens(ocmConfig.SelfToken)
	} else {
		return nil, nil, fmt.Errorf("Can't build OCM client connection. No Client/Secret or Token has been provided.")
	}

	connection, err := builder.Build()
	if err != nil {
		return nil, nil, fmt.Errorf("building OCM client connection: %w", err)
	}
	return connection, func() {
		_ = connection.Close()
	}, nil
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
			return org, errors.Wrap(err, "failed to build organisation")
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
func (c client) GetCluster(clusterID string) (*clustersmgmtv1.Cluster, error) {
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
func (c client) GetClusterStatus(id string) (*clustersmgmtv1.ClusterStatus, error) {
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

// CreateAddonWithParams ...
func (c client) CreateAddonWithParams(clusterID string, addonID string, params []Parameter) (*clustersmgmtv1.AddOnInstallation, error) {
	if c.connection == nil {
		return nil, serviceErrors.InvalidOCMConnection()
	}

	addon := clustersmgmtv1.NewAddOn().ID(addonID)
	addonParameters := newAddonParameterListBuilder(params)
	addonInstallationBuilder := clustersmgmtv1.NewAddOnInstallation().Addon(addon)
	if addonParameters != nil {
		addonInstallationBuilder = addonInstallationBuilder.Parameters(addonParameters)
	}
	addonInstallation, err := addonInstallationBuilder.Build()
	if err != nil {
		return nil, fmt.Errorf("building addon installation: %w", err)
	}
	resp, err := c.connection.ClustersMgmt().V1().Clusters().Cluster(clusterID).Addons().Add().Body(addonInstallation).Send()
	if err != nil {
		return nil, fmt.Errorf("sending AddOnInstallationAdd request: %w", err)
	}
	return resp.Body(), nil
}

// CreateAddon ...
func (c client) CreateAddon(clusterID string, addonID string) (*clustersmgmtv1.AddOnInstallation, error) {
	return c.CreateAddonWithParams(clusterID, addonID, []Parameter{})
}

// GetAddon ...
func (c client) GetAddon(clusterID string, addonID string) (*clustersmgmtv1.AddOnInstallation, error) {
	if c.connection == nil {
		return nil, serviceErrors.InvalidOCMConnection()
	}

	resp, err := c.connection.ClustersMgmt().V1().Clusters().Cluster(clusterID).Addons().List().Send()
	if err != nil {
		return nil, fmt.Errorf("sending AddOnInstallationList request: %w", err)
	}

	addon := &clustersmgmtv1.AddOnInstallation{}
	resp.Items().Each(func(addOnInstallation *clustersmgmtv1.AddOnInstallation) bool {
		if addOnInstallation.ID() == addonID {
			addon = addOnInstallation
			return false
		}
		return true
	})

	return addon, nil
}

// UpdateAddonParameters ...
func (c client) UpdateAddonParameters(clusterID string, addonInstallationID string, parameters []Parameter) (*clustersmgmtv1.AddOnInstallation, error) {
	if c.connection == nil {
		return nil, serviceErrors.InvalidOCMConnection()
	}

	addonInstallationResp, err := c.connection.ClustersMgmt().V1().Clusters().Cluster(clusterID).Addons().Addoninstallation(addonInstallationID).Get().Send()
	if err != nil {
		return nil, fmt.Errorf("sending AddOnInstallationGet request: %w", err)
	}
	if existingParameters, ok := addonInstallationResp.Body().GetParameters(); ok {
		if sameParameters(existingParameters, parameters) {
			return addonInstallationResp.Body(), nil
		}
	}
	addonInstallationBuilder := clustersmgmtv1.NewAddOnInstallation()
	updatedParamsListBuilder := newAddonParameterListBuilder(parameters)
	if updatedParamsListBuilder != nil {
		addonInstallation, err := addonInstallationBuilder.Parameters(updatedParamsListBuilder).Build()
		if err != nil {
			return nil, fmt.Errorf("building AddOnInstallation: %w", err)
		}
		resp, err := c.connection.ClustersMgmt().V1().Clusters().Cluster(clusterID).Addons().Addoninstallation(addonInstallationID).Update().Body(addonInstallation).Send()
		if err != nil {
			return nil, fmt.Errorf("sending AddOnInstallation update request: %w", err)
		}
		return resp.Body(), nil
	}
	return addonInstallationResp.Body(), nil
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

// CreateSyncSet ...
func (c client) CreateSyncSet(clusterID string, syncset *clustersmgmtv1.Syncset) (*clustersmgmtv1.Syncset, error) {
	if c.connection == nil {
		return nil, serviceErrors.InvalidOCMConnection()
	}

	clustersResource := c.connection.ClustersMgmt().V1().Clusters()
	response, syncsetErr := clustersResource.Cluster(clusterID).
		ExternalConfiguration().
		Syncsets().
		Add().
		Body(syncset).
		Send()
	var err error
	if syncsetErr != nil {
		err = serviceErrors.NewErrorFromHTTPStatusCode(response.Status(), "ocm client failed to create syncset: %s", syncsetErr)
	}
	return response.Body(), err
}

// UpdateSyncSet ...
func (c client) UpdateSyncSet(clusterID string, syncSetID string, syncset *clustersmgmtv1.Syncset) (*clustersmgmtv1.Syncset, error) {
	if c.connection == nil {
		return nil, serviceErrors.InvalidOCMConnection()
	}

	clustersResource := c.connection.ClustersMgmt().V1().Clusters()
	response, syncsetErr := clustersResource.Cluster(clusterID).
		ExternalConfiguration().
		Syncsets().
		Syncset(syncSetID).
		Update().
		Body(syncset).
		Send()

	var err error
	if syncsetErr != nil {
		err = serviceErrors.NewErrorFromHTTPStatusCode(response.Status(), "ocm client failed to update syncset '%s': %s", syncSetID, syncsetErr)
	}
	return response.Body(), err
}

// CreateIdentityProvider ...
func (c client) CreateIdentityProvider(clusterID string, identityProvider *clustersmgmtv1.IdentityProvider) (*clustersmgmtv1.IdentityProvider, error) {
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

// GetIdentityProviderList ...
func (c client) GetIdentityProviderList(clusterID string) (*clustersmgmtv1.IdentityProviderList, error) {
	if c.connection == nil {
		return nil, serviceErrors.InvalidOCMConnection()
	}

	clusterResource := c.connection.ClustersMgmt().V1().Clusters()
	response, getIDPErr := clusterResource.Cluster(clusterID).
		IdentityProviders().
		List().
		Send()

	if getIDPErr != nil {
		return nil, serviceErrors.NewErrorFromHTTPStatusCode(response.Status(), "ocm client failed to get list of identity providers, err: %s", getIDPErr.Error())
	}
	return response.Items(), nil
}

// GetSyncSet ...
func (c client) GetSyncSet(clusterID string, syncSetID string) (*clustersmgmtv1.Syncset, error) {
	if c.connection == nil {
		return nil, serviceErrors.InvalidOCMConnection()
	}

	clustersResource := c.connection.ClustersMgmt().V1().Clusters()
	response, syncsetErr := clustersResource.Cluster(clusterID).
		ExternalConfiguration().
		Syncsets().
		Syncset(syncSetID).
		Get().
		Send()

	var err error
	if syncsetErr != nil {
		err = serviceErrors.NewErrorFromHTTPStatusCode(response.Status(), "ocm client failed to get syncset '%s': %s", syncSetID, syncsetErr)
	}
	return response.Body(), err
}

// DeleteSyncSet Status returns the response status code.
func (c client) DeleteSyncSet(clusterID string, syncsetID string) (int, error) {
	if c.connection == nil {
		return 0, serviceErrors.InvalidOCMConnection()
	}

	clustersResource := c.connection.ClustersMgmt().V1().Clusters()
	response, syncsetErr := clustersResource.Cluster(clusterID).
		ExternalConfiguration().
		Syncsets().
		Syncset(syncsetID).
		Delete().
		Send()
	return response.Status(), syncsetErr
}

// ScaleUpComputeNodes scales up compute nodes by increment value
func (c client) ScaleUpComputeNodes(clusterID string, increment int) (*clustersmgmtv1.Cluster, error) {
	return c.scaleComputeNodes(clusterID, increment)
}

// ScaleDownComputeNodes scales down compute nodes by decrement value
func (c client) ScaleDownComputeNodes(clusterID string, decrement int) (*clustersmgmtv1.Cluster, error) {
	return c.scaleComputeNodes(clusterID, -decrement)
}

// scaleComputeNodes scales the Compute nodes up or down by the value of `numNodes`
func (c client) scaleComputeNodes(clusterID string, numNodes int) (*clustersmgmtv1.Cluster, error) {
	if c.connection == nil {
		return nil, serviceErrors.InvalidOCMConnection()
	}

	clusterClient := c.connection.ClustersMgmt().V1().Clusters().Cluster(clusterID)

	cluster, err := clusterClient.Get().Send()
	if err != nil {
		return nil, fmt.Errorf("retrieving cluster: %w", err)
	}

	// get current number of compute nodes
	currentNumOfNodes := cluster.Body().Nodes().Compute()

	// create a cluster object with updated number of compute nodes
	// NOTE - there is no need to handle whether the number of nodes is valid, as this is handled by OCM
	patch, err := clustersmgmtv1.NewCluster().Nodes(clustersmgmtv1.NewClusterNodes().Compute(currentNumOfNodes + numNodes)).
		Build()
	if err != nil {
		return nil, fmt.Errorf("scaling compute nodes by %d nodes: %w", numNodes, err)
	}

	// patch cluster with updated number of compute nodes
	resp, err := clusterClient.Update().Body(patch).Send()
	if err != nil {
		return nil, fmt.Errorf("patching cluster with updated number of compute nodes: %w", err)
	}

	return resp.Body(), nil
}

// SetComputeNodes ...
func (c client) SetComputeNodes(clusterID string, numNodes int) (*clustersmgmtv1.Cluster, error) {
	if c.connection == nil {
		return nil, serviceErrors.InvalidOCMConnection()
	}

	clusterClient := c.connection.ClustersMgmt().V1().Clusters().Cluster(clusterID)

	patch, err := clustersmgmtv1.NewCluster().Nodes(clustersmgmtv1.NewClusterNodes().Compute(numNodes)).
		Build()
	if err != nil {
		return nil, fmt.Errorf("building %d compute nodes: %w", numNodes, err)
	}

	// patch cluster with updated number of compute nodes
	resp, err := clusterClient.Update().Body(patch).Send()
	if err != nil {
		return nil, fmt.Errorf("patching cluster with updated number of compute nodes: %w", err)
	}

	return resp.Body(), nil
}

func newAddonParameterListBuilder(params []Parameter) *clustersmgmtv1.AddOnInstallationParameterListBuilder {
	if len(params) > 0 {
		var items []*clustersmgmtv1.AddOnInstallationParameterBuilder
		for _, p := range params {
			pb := clustersmgmtv1.NewAddOnInstallationParameter().ID(p.ID).Value(p.Value)
			items = append(items, pb)
		}
		return clustersmgmtv1.NewAddOnInstallationParameterList().Items(items...)
	}
	return nil
}

func sameParameters(parameterList *clustersmgmtv1.AddOnInstallationParameterList, params []Parameter) bool {
	if parameterList.Len() != len(params) {
		return false
	}
	paramsMap := map[string]string{}
	for _, p := range params {
		paramsMap[p.ID] = p.Value
	}
	match := true
	parameterList.Each(func(item *clustersmgmtv1.AddOnInstallationParameter) bool {
		if paramsMap[item.ID()] != item.Value() {
			match = false
			return false
		}
		return true
	})
	return match
}

// DeleteCluster ...
func (c client) DeleteCluster(clusterID string) (int, error) {
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
func (c client) ClusterAuthorization(cb *amsv1.ClusterAuthorizationRequest) (*amsv1.ClusterAuthorizationResponse, error) {
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
func (c client) DeleteSubscription(id string) (int, error) {
	if c.connection == nil {
		return 0, serviceErrors.InvalidOCMConnection()
	}

	r := c.connection.AccountsMgmt().V1().Subscriptions().Subscription(id).Delete()
	resp, err := r.Send()
	return resp.Status(), err
}

// FindSubscriptions ...
func (c client) FindSubscriptions(query string) (*amsv1.SubscriptionsListResponse, error) {
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
func (c client) GetQuotaCostsForProduct(organizationID, resourceName, product string) ([]*amsv1.QuotaCost, error) {
	if c.connection == nil {
		return nil, serviceErrors.InvalidOCMConnection()
	}

	var res []*amsv1.QuotaCost
	organizationClient := c.connection.AccountsMgmt().V1().Organizations()
	quotaCostClient := organizationClient.Organization(organizationID).QuotaCost()

	quotaCostList, err := quotaCostClient.List().Parameter("fetchRelatedResources", true).Send()
	if err != nil {
		return nil, fmt.Errorf("retrieving relatedResources from the QuotaCosts service: %w", err)
	}

	quotaCostList.Items().Each(func(qc *amsv1.QuotaCost) bool {
		relatedResourcesList := qc.RelatedResources()
		for _, relatedResource := range relatedResourcesList {
			if relatedResource.ResourceName() == resourceName && relatedResource.Product() == product {
				res = append(res, qc)
			}
		}
		return true
	})

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
