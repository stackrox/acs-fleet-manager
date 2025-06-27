// Package quota ...
package quota

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/golang/glog"
	amsv1 "github.com/openshift-online/ocm-sdk-go/accountsmgmt/v1"
	"github.com/openshift-online/ocm-sdk-go/authentication"
	"github.com/stackrox/acs-fleet-manager/internal/central/pkg/api/dbapi"
	"github.com/stackrox/acs-fleet-manager/internal/central/pkg/centrals/types"
	ocm "github.com/stackrox/acs-fleet-manager/pkg/client/ocm/impl"
	"github.com/stackrox/acs-fleet-manager/pkg/errors"
)

// RHACSMarketplaceQuotaID is default quota id used by ACS SKUs.
const RHACSMarketplaceQuotaID = "cluster|rhinfra|rhacs|marketplace"
const awsCloudProvider = "aws"

type amsQuotaService struct {
	amsClient ocm.AMSClient
}

func newBaseQuotaReservedResourceResourceBuilder() amsv1.ReservedResourceBuilder {
	rr := amsv1.ReservedResourceBuilder{}
	rr.ResourceType("cluster.aws")
	rr.BYOC(false)
	rr.ResourceName("rhacs")
	rr.AvailabilityZoneType("multi")
	rr.Count(1)
	return rr
}

var supportedAMSBillingModels = map[string]struct{}{
	string(amsv1.BillingModelMarketplace):    {},
	string(amsv1.BillingModelStandard):       {},
	string(amsv1.BillingModelMarketplaceAWS): {},
}

// HasQuotaAllowance checks if allowed quota is not zero for the given instance type.
func (q amsQuotaService) HasQuotaAllowance(central *dbapi.CentralRequest, instanceType types.CentralInstanceType) (bool, *errors.ServiceError) {
	org, err := q.amsClient.GetOrganisationFromExternalID(central.OrganisationID)
	if err != nil {
		return false, errors.OrganisationNotFound(central.OrganisationID, err)
	}

	quotaType := instanceType.GetQuotaType()
	quotaCosts, err := q.amsClient.GetQuotaCostsForProduct(org.ID(), quotaType.GetResourceName(), quotaType.GetProduct())
	if err != nil {
		return false, errors.NewWithCause(errors.ErrorGeneral, err, fmt.Sprintf(
			"failed to get assigned quota of type %q for organization with external id %q and id %q",
			quotaType, central.OrganisationID, org.ID()))
	}

	quotaCostsByModel, unsupportedModels := mapAllowedQuotaCosts(quotaCosts)
	if len(quotaCostsByModel) == 0 && len(unsupportedModels) > 0 {
		return false, errors.GeneralError("found only unsupported billing models %q for product %q", unsupportedModels, quotaType.GetProduct())
	}

	isCloudAccount := central.CloudAccountID != ""
	standardAccountIsActive := !isCloudAccount && len(quotaCostsByModel[amsv1.BillingModelStandard]) > 0

	// Entitlement is active if there's allowed quota for standard billing model
	// or there is cloud quota and the original cloud account is still active.
	entitled := standardAccountIsActive || cloudAccountIsActive(quotaCostsByModel, central)

	if !entitled {
		glog.Infof("Quota no longer entitled for organisation %q for cloud account %q. Quota Cost: %s",
			org.ID(), central.CloudAccountID, printQuotaCostMap(quotaCostsByModel))
		return false, nil
	}
	return true, nil
}

// selectBillingModel selects the billing model of an instance by looking
// at the resource name and product, cloudProviderID and cloudAccountID.
// Only QuotaCosts that have available quota, or that contain a RelatedResource
// with "cost" 0 are considered.
// Only "standard", "marketplace" and "marketplace-aws" billing models are
// considered. If both "marketplace" and "standard" billing models are
// available, "marketplace" will be given preference.
func (q amsQuotaService) selectBillingModel(orgID, cloudProviderID, cloudAccountID string, resourceName string, product string) (string, error) {
	quotaCosts, err := q.amsClient.GetQuotaCostsForProduct(orgID, resourceName, product)
	if err != nil {
		return "", errors.InsufficientQuotaError("%v: error getting quotas for product %s", err, product)
	}

	hasBillingModelMarketplace := false
	hasBillingModelMarketplaceAWS := false
	hasBillingModelStandard := false
	for _, qc := range quotaCosts {
		for _, rr := range qc.RelatedResources() {
			if qc.Consumed() < qc.Allowed() || rr.Cost() == 0 {
				hasBillingModelMarketplace = hasBillingModelMarketplace || rr.BillingModel() == string(amsv1.BillingModelMarketplace)
				hasBillingModelMarketplaceAWS = hasBillingModelMarketplaceAWS || rr.BillingModel() == string(amsv1.BillingModelMarketplaceAWS)
				hasBillingModelStandard = hasBillingModelStandard || rr.BillingModel() == string(amsv1.BillingModelStandard)
			}
		}
	}

	if cloudAccountID != "" && cloudProviderID == awsCloudProvider {
		if hasBillingModelMarketplaceAWS || hasBillingModelMarketplace {
			return string(amsv1.BillingModelMarketplaceAWS), nil
		}
		return "", errors.InvalidCloudAccountID("No subscription available for cloud account %s", cloudAccountID)
	}
	if hasBillingModelMarketplace {
		return string(amsv1.BillingModelMarketplace), nil
	}
	if hasBillingModelStandard {
		return string(amsv1.BillingModelStandard), nil
	}
	return "", errors.InsufficientQuotaError("No available billing model found")
}

// ReserveQuota calls AMS to reserve quota for the central request. It computes
// the central billing parameters if they're not forced.
func (q amsQuotaService) ReserveQuota(ctx context.Context, central *dbapi.CentralRequest, forcedBillingModel string, forcedProduct string) (string, *errors.ServiceError) {
	instanceType := types.CentralInstanceType(central.InstanceType)
	centralID := central.ID
	rr := newBaseQuotaReservedResourceResourceBuilder()

	// The reason to call /current_account here is how AMS functions.
	// In case customer just created the account, AMS might miss information about their quota.
	// Calling /current_account endpoint results in this data being populated.
	// Since this is a non-requirement for successful quota reservation, errors are logged but ignored here.
	q.callCurrentAccount(ctx)

	org, err := q.amsClient.GetOrganisationFromExternalID(central.OrganisationID)
	if err != nil {
		return "", errors.OrganisationNotFound(central.OrganisationID, err)
	}

	product := instanceType.GetQuotaType().GetProduct()
	if forcedProduct != "" {
		product = forcedProduct
	}

	var bm string
	if forcedBillingModel == "" {
		resourceName := instanceType.GetQuotaType().GetResourceName()
		bm, err = q.selectBillingModel(org.ID(), central.CloudProvider, central.CloudAccountID, resourceName, product)
		if err != nil {
			svcErr := errors.ToServiceError(err)
			return "", errors.NewWithCause(svcErr.Code, svcErr, "Error getting billing model")
		}
	} else {
		bm = forcedBillingModel
	}
	rr.BillingModel(amsv1.BillingModel(bm))
	glog.Infof("Billing model of Central request %q with quota type %q has been set to %q.", central.ID, instanceType.GetQuotaType(), bm)

	if bm != string(amsv1.BillingModelStandard) {
		if err := q.verifyCloudAccountInAMS(central, org.ID()); err != nil {
			return "", err
		}
		if bm != string(amsv1.BillingModelMarketplace) &&
			bm != string(amsv1.BillingModelMarketplaceRHM) {
			rr.BillingMarketplaceAccount(central.CloudAccountID)
		}
	}

	requestBuilder := amsv1.NewClusterAuthorizationRequest().
		AccountUsername(central.Owner).
		CloudProviderID(central.CloudProvider).
		ProductID(product).
		Managed(true).
		ClusterID(centralID).
		ExternalClusterID(centralID).
		Disconnected(false).
		BYOC(false).
		AvailabilityZone("multi").
		Reserve(true).
		Resources(&rr)

	cb, err := requestBuilder.Build()
	if err != nil {
		return "", errors.NewWithCause(errors.ErrorGeneral, err, "Error reserving quota")
	}

	resp, err := q.amsClient.ClusterAuthorization(cb)
	if err != nil {
		return "", errors.FailedClusterAuthorization(err)
	}

	if resp.Allowed() {
		return resp.Subscription().ID(), nil
	}
	return "", errors.InsufficientQuotaError("Insufficient Quota")
}

func (q amsQuotaService) callCurrentAccount(ctx context.Context) {
	userToken, err := authentication.TokenFromContext(ctx)
	if err != nil {
		glog.Warningf("Couldn't extract user token from context: %w", err)
		return
	}
	if userToken == nil {
		return
	}
	status, acc, err := q.amsClient.GetCurrentAccount(userToken.Raw)
	if err != nil {
		glog.Warningf("Failed to query current account (%v): %w", status, err)
	} else {
		glog.Infof("Succeeded to query current account (%v): <%s> created at %v, belongs to %q created at %v", status, acc.Email(), acc.CreatedAt().Format(time.RFC3339), acc.Organization().Name(), acc.Organization().CreatedAt().Format(time.RFC3339))
	}
}

func (q amsQuotaService) verifyCloudAccountInAMS(central *dbapi.CentralRequest, orgID string) *errors.ServiceError {
	cloudAccounts, err := q.amsClient.GetCustomerCloudAccounts(orgID, []string{RHACSMarketplaceQuotaID})
	if err != nil {
		svcErr := errors.ToServiceError(err)
		return errors.NewWithCause(svcErr.Code, svcErr, "Error getting cloud accounts")
	}

	if central.CloudAccountID == "" {
		if len(cloudAccounts) != 0 {
			return errors.InvalidCloudAccountID("Missing cloud account id in creation request")
		}
		return nil
	}
	for _, cloudAccount := range cloudAccounts {
		if cloudAccount.CloudAccountID() == central.CloudAccountID && cloudAccount.CloudProviderID() == central.CloudProvider {
			return nil
		}
	}
	return errors.InvalidCloudAccountID("Request cloud account %s does not match organization cloud accounts", central.CloudAccountID)
}

// DeleteQuota ...
func (q amsQuotaService) DeleteQuota(subscriptionID string) *errors.ServiceError {
	if subscriptionID == "" {
		return nil
	}

	status, err := q.amsClient.DeleteSubscription(subscriptionID)
	if err != nil {
		if status == http.StatusNotFound {
			glog.Infof("quota for subscription: %v not found, asuming it's already deleted", subscriptionID)
			return nil
		}
		return errors.GeneralError("failed to delete the quota: %v", err)
	}
	return nil
}

func mapAllowedQuotaCosts(quotaCosts []*amsv1.QuotaCost) (map[amsv1.BillingModel][]*amsv1.QuotaCost, []string) {
	costsMap := make(map[amsv1.BillingModel][]*amsv1.QuotaCost)
	var foundUnsupportedBillingModels []string
	for _, qc := range quotaCosts {
		for _, rr := range qc.RelatedResources() {
			// When an SKU entitlement expires in AMS, the allowed value for that quota cost is set back to 0.
			// Ignore allowance for zero-cost resources.
			if qc.Allowed() == 0 && rr.Cost() != 0 {
				continue
			}
			bm := amsv1.BillingModel(rr.BillingModel())
			if _, isCompatibleBillingModel := supportedAMSBillingModels[rr.BillingModel()]; isCompatibleBillingModel {
				costsMap[bm] = append(costsMap[bm], qc)
			} else {
				foundUnsupportedBillingModels = append(foundUnsupportedBillingModels, rr.BillingModel())
			}
		}
	}
	return costsMap, foundUnsupportedBillingModels
}

func cloudAccountIsActive(costsMap map[amsv1.BillingModel][]*amsv1.QuotaCost, central *dbapi.CentralRequest) bool {
	for model, quotaCosts := range costsMap {
		if model == amsv1.BillingModelStandard {
			continue
		}
		for _, qc := range quotaCosts {
			for _, account := range qc.CloudAccounts() {
				if account.CloudAccountID() == central.CloudAccountID &&
					account.CloudProviderID() == central.CloudProvider {
					return true
				}
			}
		}
	}
	return false
}
