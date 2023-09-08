// Package quota ...
package quota

import (
	"context"
	"fmt"
	"github.com/openshift-online/ocm-sdk-go/authentication"
	"time"

	"github.com/golang/glog"
	amsv1 "github.com/openshift-online/ocm-sdk-go/accountsmgmt/v1"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/api/dbapi"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/config"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/dinosaurs/types"
	"github.com/stackrox/acs-fleet-manager/pkg/client/ocm"
	"github.com/stackrox/acs-fleet-manager/pkg/errors"
)

// RHACSMarketplaceQuotaID is default quota id used by ACS SKUs.
const RHACSMarketplaceQuotaID = "cluster|rhinfra|rhacs|marketplace"
const awsCloudProvider = "aws"

type amsQuotaService struct {
	amsClient     ocm.AMSClient
	centralConfig *config.CentralConfig
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

// CheckIfQuotaIsDefinedForInstanceType ...
func (q amsQuotaService) CheckIfQuotaIsDefinedForInstanceType(dinosaur *dbapi.CentralRequest, instanceType types.DinosaurInstanceType) (bool, *errors.ServiceError) {
	org, err := q.amsClient.GetOrganisationFromExternalID(dinosaur.OrganisationID)
	if err != nil {
		return false, errors.OrganisationNotFound(dinosaur.OrganisationID, err)
	}

	hasQuota, err := q.hasConfiguredQuotaCost(org.ID(), instanceType.GetQuotaType())
	if err != nil {
		return false, errors.NewWithCause(errors.ErrorGeneral, err, fmt.Sprintf("failed to get assigned quota of type %v for organization with external id %v and id %v", instanceType.GetQuotaType(), dinosaur.OrganisationID, org.ID()))
	}

	return hasQuota, nil
}

// hasConfiguredQuotaCost returns true if the given organizationID has at least
// one AMS QuotaCost that complies with the following conditions:
//   - Matches the given input quotaType
//   - Contains at least one AMS RelatedResources whose billing model is one
//     of the supported Billing Models specified in supportedAMSBillingModels
//   - Has an "Allowed" value greater than 0
//
// An error is returned if the given organizationID has a QuotaCost
// with an unsupported billing model and there are no supported billing models
func (q amsQuotaService) hasConfiguredQuotaCost(organizationID string, quotaType ocm.DinosaurQuotaType) (bool, error) {
	quotaCosts, err := q.amsClient.GetQuotaCostsForProduct(organizationID, quotaType.GetResourceName(), quotaType.GetProduct())
	if err != nil {
		return false, fmt.Errorf("retrieving quota costs for product %s, organization ID %s, resource type %s: %w", quotaType.GetProduct(), organizationID, quotaType.GetResourceName(), err)
	}

	var foundUnsupportedBillingModel string

	var supported []*amsv1.QuotaCost

	for _, qc := range quotaCosts {
		if qc.Allowed() > 0 {
			for _, rr := range qc.RelatedResources() {
				if _, isCompatibleBillingModel := supportedAMSBillingModels[rr.BillingModel()]; isCompatibleBillingModel {
					supported = append(supported, qc)
				} else {
					foundUnsupportedBillingModel = rr.BillingModel()
				}
			}
		}
	}

	if foundUnsupportedBillingModel != "" {
		return false, errors.GeneralError("Product %s only has unsupported allowed billing models. Last one found: %s", quotaType.GetProduct(), foundUnsupportedBillingModel)
	}

	return false, nil
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

// IsQuotaEntitlementActive checks if an organisation has a SKU entitlement and that entitlement is still active in AMS.
func (q amsQuotaService) IsQuotaEntitlementActive(dinosaur *dbapi.CentralRequest) (bool, error) {
	org, err := q.amsClient.GetOrganisationFromExternalID(dinosaur.OrganisationID)
	if err != nil {
		return false, errors.OrganisationNotFound(dinosaur.OrganisationID, err)
	}

	organizationID := org.ID()
	quotaType := types.DinosaurInstanceType(dinosaur.InstanceType).GetQuotaType()
	quotaCosts, err := q.amsClient.GetQuotaCostsForProduct(org.ID(), quotaType.GetResourceName(), quotaType.GetProduct())
	if err != nil {
		return false, fmt.Errorf("retrieving quota costs for product %s, organization ID %s, resource type %s: %w", quotaType.GetProduct(), organizationID, quotaType.GetResourceName(), err)
	}

	// SKU not entitled to the organisation
	if len(quotaCosts) == 0 {
		glog.Infof("ams quota cost not found for Central instance %q in organisation %q with the following filter: {ResourceName: %q, Product: %q}",
			dinosaur.ID, organizationID, quotaType.GetResourceName(), quotaType.GetProduct())
		return false, nil
	}

	if len(quotaCosts) > 1 {
		return false, fmt.Errorf("more than 1 quota cost was returned for organisation %q with the following filter: {ResourceName: %q, Product: %q}",
			organizationID, quotaType.GetResourceName(), quotaType.GetProduct())
	}

	// result will always return one quota for the given org, ams related resource billing model, resource and product for rhosak quotas
	quotaCost := quotaCosts[0]

	// when a SKU entitlement expires in AMS, the allowed value for that quota cost is set back to 0
	// if the allowed value is 0 and consumed is greater than this, that denotes that the SKU entitlement
	// has expired and is no longer active.
	if quotaCost.Consumed() > quotaCost.Allowed() {
		if quotaCost.Allowed() == 0 {
			glog.Infof("quota no longer entitled for organisation %q (quotaid: %q, consumed: %q, allowed: %q)",
				organizationID, quotaCost.QuotaID(), quotaCost.Consumed(), quotaCost.Allowed())
			return false, nil
		}
		glog.Warningf("Organisation %q has exceeded their quota allowance (quotaid: %q, consumed %q, allowed: %q",
			organizationID, quotaCost.QuotaID(), quotaCost.Consumed(), quotaCost.Allowed())
	}
	return true, nil
}
