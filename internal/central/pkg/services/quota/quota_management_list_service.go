package quota

import (
	"fmt"

	"github.com/stackrox/acs-fleet-manager/pkg/quotamanagement"

	"github.com/stackrox/acs-fleet-manager/internal/central/pkg/api/dbapi"
	"github.com/stackrox/acs-fleet-manager/internal/central/pkg/centrals/types"
	"github.com/stackrox/acs-fleet-manager/pkg/db"
	"github.com/stackrox/acs-fleet-manager/pkg/errors"
)

// QuotaManagementListService ...
type QuotaManagementListService struct {
	connectionFactory   *db.ConnectionFactory
	quotaManagementList *quotamanagement.QuotaManagementListConfig
}

// CheckIfQuotaIsDefinedForInstanceType ...
func (q QuotaManagementListService) CheckIfQuotaIsDefinedForInstanceType(dinosaur *dbapi.CentralRequest, instanceType types.CentralInstanceType) (bool, *errors.ServiceError) {
	username := dinosaur.Owner
	orgID := dinosaur.OrganisationID
	org, orgFound := q.quotaManagementList.QuotaList.Organisations.GetByID(orgID)
	userIsRegistered := false
	if orgFound && org.IsUserRegistered(username) {
		userIsRegistered = true
	} else {
		_, userFound := q.quotaManagementList.QuotaList.ServiceAccounts.GetByUsername(username)
		userIsRegistered = userFound
	}

	// allow user defined in quota list to create standard instances
	if userIsRegistered && instanceType == types.STANDARD {
		return true, nil
	} else if !userIsRegistered && instanceType == types.EVAL { // allow user who are not in quota list to create eval instances
		return true, nil
	}

	return false, nil
}

// ReserveQuota ...
func (q QuotaManagementListService) ReserveQuota(dinosaur *dbapi.CentralRequest, instanceType types.CentralInstanceType) (string, *errors.ServiceError) {
	if !q.quotaManagementList.EnableInstanceLimitControl {
		return "", nil
	}

	username := dinosaur.Owner
	orgID := dinosaur.OrganisationID
	var quotaManagementListItem quotamanagement.QuotaManagementListItem
	message := fmt.Sprintf("User '%s' has reached a maximum number of %d allowed instances.", username, quotamanagement.GetDefaultMaxAllowedInstances())
	org, orgFound := q.quotaManagementList.QuotaList.Organisations.GetByID(orgID)
	filterByOrd := false
	if orgFound && org.IsUserRegistered(username) {
		quotaManagementListItem = org
		message = fmt.Sprintf("Organization '%s' has reached a maximum number of %d allowed instances.", orgID, org.GetMaxAllowedInstances())
		filterByOrd = true
	} else {
		user, userFound := q.quotaManagementList.QuotaList.ServiceAccounts.GetByUsername(username)
		if userFound {
			quotaManagementListItem = user
			message = fmt.Sprintf("User '%s' has reached a maximum number of %d allowed instances.", username, user.GetMaxAllowedInstances())
		}
	}

	var count int64
	dbConn := q.connectionFactory.New().
		Model(&dbapi.CentralRequest{}).
		Where("instance_type = ?", instanceType.String())

	if instanceType == types.STANDARD && filterByOrd {
		dbConn = dbConn.Where("organisation_id = ?", orgID)
	} else {
		dbConn = dbConn.Where("owner = ?", username)
	}

	if err := dbConn.Count(&count).Error; err != nil {
		return "", errors.GeneralError("count failed from database")
	}

	totalInstanceCount := int(count)
	if quotaManagementListItem != nil && instanceType == types.STANDARD {
		if quotaManagementListItem.IsInstanceCountWithinLimit(totalInstanceCount) {
			return "", nil
		}
		return "", errors.MaximumAllowedInstanceReached(message)
	}

	if instanceType == types.EVAL && quotaManagementListItem == nil {
		if totalInstanceCount >= quotamanagement.GetDefaultMaxAllowedInstances() {
			return "", errors.MaximumAllowedInstanceReached(message)
		}
		return "", nil
	}

	return "", errors.InsufficientQuotaError("Insufficient Quota")
}

// DeleteQuota ...
func (q QuotaManagementListService) DeleteQuota(SubscriptionID string) *errors.ServiceError {
	return nil // NOOP
}
