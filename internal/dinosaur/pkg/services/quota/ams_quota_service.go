// Package quota ...
package quota

import (
	"context"
	"fmt"
	"time"

	"github.com/golang/glog"
	amsv1 "github.com/openshift-online/ocm-sdk-go/accountsmgmt/v1"
	"github.com/openshift-online/ocm-sdk-go/authentication"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/api/dbapi"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/dinosaurs/types"
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
func (q amsQuotaService) HasQuotaAllowance(central *dbapi.CentralRequest, instanceType types.DinosaurInstanceType) (bool, *errors.ServiceError) {
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
		glog.Infof("Quota no longer entitled for organisation %q", org.ID)
		return false, nil
	}
	return true, nil
}

// selectBillingModelFromDinosaurInstanceType select the billing model of a
// dinosaur instance type by looking at the resource name and product of the
// instanceType, as well as cloudAccountID and cloudProviderID. Only QuotaCosts that have available quota, or that contain a
// RelatedResource with "cost" 0 are considered. Only
// "standard" and "marketplace" and "marketplace-aws" billing models are considered.
// If both marketplace and standard billing models are available, marketplace will be given preference.
func (q amsQuotaService) selectBillingModelFromDinosaurInstanceType(orgID, cloudProviderID, cloudAccountID string, instanceType types.DinosaurInstanceType) (string, error) {
	quotaCosts, err := q.amsClient.GetQuotaCostsForProduct(orgID, instanceType.GetQuotaType().GetResourceName(), instanceType.GetQuotaType().GetProduct())
	if err != nil {
		return "", errors.InsufficientQuotaError("%v: error getting quotas for product %s", err, instanceType.GetQuotaType().GetProduct())
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

// ReserveQuota ...
func (q amsQuotaService) ReserveQuota(ctx context.Context, dinosaur *dbapi.CentralRequest, instanceType types.DinosaurInstanceType) (string, *errors.ServiceError) {
	dinosaurID := dinosaur.ID
	rr := newBaseQuotaReservedResourceResourceBuilder()

	// The reason to call /current_account here is how AMS functions.
	// In case customer just created the account, AMS might miss information about their quota.
	// Calling /current_account endpoint results in this data being populated.
	// Since this is a non-requirement for successful quota reservation, errors are logged but ignored here.
	q.callCurrentAccount(ctx)

	org, err := q.amsClient.GetOrganisationFromExternalID(dinosaur.OrganisationID)
	if err != nil {
		return "", errors.OrganisationNotFound(dinosaur.OrganisationID, err)
	}
	bm, err := q.selectBillingModelFromDinosaurInstanceType(org.ID(), dinosaur.CloudProvider, dinosaur.CloudAccountID, instanceType)
	if err != nil {
		svcErr := errors.ToServiceError(err)
		return "", errors.NewWithCause(svcErr.Code, svcErr, "Error getting billing model")
	}
	rr.BillingModel(amsv1.BillingModel(bm))
	glog.Infof("Billing model of Central request %s with quota type %s has been set to %s.", dinosaur.ID, instanceType.GetQuotaType(), bm)

	if bm != string(amsv1.BillingModelStandard) {
		if err := q.verifyCloudAccountInAMS(dinosaur, org.ID()); err != nil {
			return "", err
		}
		rr.BillingMarketplaceAccount(dinosaur.CloudAccountID)
	}

	requestBuilder := amsv1.NewClusterAuthorizationRequest().
		AccountUsername(dinosaur.Owner).
		CloudProviderID(dinosaur.CloudProvider).
		ProductID(instanceType.GetQuotaType().GetProduct()).
		Managed(true).
		ClusterID(dinosaurID).
		ExternalClusterID(dinosaurID).
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

func (q amsQuotaService) verifyCloudAccountInAMS(dinosaur *dbapi.CentralRequest, orgID string) *errors.ServiceError {
	cloudAccounts, err := q.amsClient.GetCustomerCloudAccounts(orgID, []string{RHACSMarketplaceQuotaID})
	if err != nil {
		svcErr := errors.ToServiceError(err)
		return errors.NewWithCause(svcErr.Code, svcErr, "Error getting cloud accounts")
	}

	if dinosaur.CloudAccountID == "" {
		if len(cloudAccounts) != 0 {
			return errors.InvalidCloudAccountID("Missing cloud account id in creation request")
		}
		return nil
	}
	for _, cloudAccount := range cloudAccounts {
		if cloudAccount.CloudAccountID() == dinosaur.CloudAccountID && cloudAccount.CloudProviderID() == dinosaur.CloudProvider {
			return nil
		}
	}
	return errors.InvalidCloudAccountID("Request cloud account %s does not match organization cloud accounts", dinosaur.CloudAccountID)
}

// DeleteQuota ...
func (q amsQuotaService) DeleteQuota(subscriptionID string) *errors.ServiceError {
	if subscriptionID == "" {
		return nil
	}

	_, err := q.amsClient.DeleteSubscription(subscriptionID)
	if err != nil {
		return errors.GeneralError("failed to delete the quota: %v", err)
	}
	return nil
}

func mapAllowedQuotaCosts(quotaCosts []*amsv1.QuotaCost) (map[amsv1.BillingModel][]*amsv1.QuotaCost, []string) {
	costsMap := make(map[amsv1.BillingModel][]*amsv1.QuotaCost)
	var foundUnsupportedBillingModels []string
	for _, qc := range quotaCosts {
		// When an SKU entitlement expires in AMS, the allowed value for that quota cost is set back to 0.
		if qc.Allowed() == 0 {
			continue
		}
		for _, rr := range qc.RelatedResources() {
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
