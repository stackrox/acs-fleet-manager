// Package quota ...
package quota

import (
	"fmt"

	"github.com/golang/glog"
	amsv1 "github.com/openshift-online/ocm-sdk-go/accountsmgmt/v1"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/api/dbapi"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/dinosaurs/types"
	"github.com/stackrox/acs-fleet-manager/pkg/client/ocm"
	"github.com/stackrox/acs-fleet-manager/pkg/errors"
)

// RHACSMarketplaceQuotaID is default quota id used by ACS SKUs.
const RHACSMarketplaceQuotaID = "cluster|rhinfra|rhacs|marketplace"

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
	string(amsv1.BillingModelMarketplace): {},
	string(amsv1.BillingModelStandard):    {},
}

// CheckIfQuotaIsDefinedForInstanceType ...
func (q amsQuotaService) CheckIfQuotaIsDefinedForInstanceType(dinosaur *dbapi.CentralRequest, instanceType types.DinosaurInstanceType) (bool, *errors.ServiceError) {
	orgID, err := q.amsClient.GetOrganisationIDFromExternalID(dinosaur.OrganisationID)
	if err != nil {
		return false, errors.NewWithCause(errors.ErrorGeneral, err, fmt.Sprintf("Error checking quota: failed to get organization with external id %v", dinosaur.OrganisationID))
	}

	hasQuota, err := q.hasConfiguredQuotaCost(orgID, instanceType.GetQuotaType())
	if err != nil {
		return false, errors.NewWithCause(errors.ErrorGeneral, err, fmt.Sprintf("Error checking quota: failed to get assigned quota of type %v for organization with id %v", instanceType.GetQuotaType(), orgID))
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

	for _, qc := range quotaCosts {
		if qc.Allowed() > 0 {
			for _, rr := range qc.RelatedResources() {
				if _, isCompatibleBillingModel := supportedAMSBillingModels[rr.BillingModel()]; isCompatibleBillingModel {
					return true, nil
				}
				foundUnsupportedBillingModel = rr.BillingModel()
			}
		}
	}

	if foundUnsupportedBillingModel != "" {
		return false, errors.GeneralError("Product %s only has unsupported allowed billing models. Last one found: %s", quotaType.GetProduct(), foundUnsupportedBillingModel)
	}

	return false, nil
}

// getAvailableBillingModelFromDinosaurInstanceType gets the billing model of a
// dinosaur instance type by looking at the resource name and product of the
// instanceType. Only QuotaCosts that have available quota, or that contain a
// RelatedResource with "cost" 0 are considered. Only
// "standard" and "marketplace" billing models are considered. If both are
// detected "marketplace" is returned.
func (q amsQuotaService) getAvailableBillingModelFromDinosaurInstanceType(orgID string, instanceType types.DinosaurInstanceType) (string, error) {
	quotaCosts, err := q.amsClient.GetQuotaCostsForProduct(orgID, instanceType.GetQuotaType().GetResourceName(), instanceType.GetQuotaType().GetProduct())
	if err != nil {
		return "", errors.InsufficientQuotaError("%v: error getting quotas for product %s", err, instanceType.GetQuotaType().GetProduct())
	}

	billingModel := ""
	for _, qc := range quotaCosts {
		for _, rr := range qc.RelatedResources() {
			if qc.Consumed() < qc.Allowed() || rr.Cost() == 0 {
				if rr.BillingModel() == string(amsv1.BillingModelMarketplace) {
					return rr.BillingModel(), nil
				} else if rr.BillingModel() == string(amsv1.BillingModelStandard) {
					billingModel = rr.BillingModel()
				}
			}
		}
	}

	return billingModel, nil
}

// ReserveQuota ...
func (q amsQuotaService) ReserveQuota(dinosaur *dbapi.CentralRequest, instanceType types.DinosaurInstanceType) (string, *errors.ServiceError) {
	dinosaurID := dinosaur.ID

	rr := newBaseQuotaReservedResourceResourceBuilder()

	orgID, err := q.amsClient.GetOrganisationIDFromExternalID(dinosaur.OrganisationID)
	if err != nil {
		return "", errors.NewWithCause(errors.ErrorGeneral, err, fmt.Sprintf("Error checking quota: failed to get organization with external id %v", dinosaur.OrganisationID))
	}
	bm, err := q.getAvailableBillingModelFromDinosaurInstanceType(orgID, instanceType)
	if err != nil {
		svcErr := errors.ToServiceError(err)
		return "", errors.NewWithCause(svcErr.Code, svcErr, "Error getting billing model")
	}
	if bm == "" {
		return "", errors.InsufficientQuotaError("Error getting billing model: No available billing model found")
	}
	rr.BillingModel(amsv1.BillingModel(bm))
	glog.Infof("Billing model of Central request %s with quota type %s has been set to %s.", dinosaur.ID, instanceType.GetQuotaType(), bm)

	requestBuilder := amsv1.NewClusterAuthorizationRequest().
		AccountUsername(dinosaur.Owner).
		ProductID(instanceType.GetQuotaType().GetProduct()).
		Managed(true).
		ClusterID(dinosaurID).
		ExternalClusterID(dinosaurID).
		Disconnected(false).
		BYOC(false).
		AvailabilityZone("multi").
		Reserve(true).
		Resources(&rr)

	if bm != string(amsv1.BillingModelStandard) {
		if err := q.verifyCloudAccountInAMS(dinosaur, orgID); err != nil {
			return "", err
		}
		requestBuilder = requestBuilder.
			CloudProviderID(dinosaur.CloudProvider).
			CloudAccountID(dinosaur.CloudAccountID)
	}
	cb, err := requestBuilder.Build()
	if err != nil {
		return "", errors.NewWithCause(errors.ErrorGeneral, err, "Error reserving quota")
	}

	resp, err := q.amsClient.ClusterAuthorization(cb)
	if err != nil {
		return "", errors.NewWithCause(errors.ErrorGeneral, err, "Error reserving quota")
	}

	if resp.Allowed() {
		return resp.Subscription().ID(), nil
	}
	return "", errors.InsufficientQuotaError("Insufficient Quota")
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
	return errors.InvalidCloudAccountID("Request cloud account does not match organization cloud accounts", dinosaur.CloudAccountID)
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
