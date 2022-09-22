package services

import (
	"context"
	"fmt"
	"sync"
	"time"

	centralConstants "github.com/stackrox/acs-fleet-manager/internal/central/constants"
	"github.com/stackrox/acs-fleet-manager/internal/central/pkg/api/dbapi"
	"github.com/stackrox/acs-fleet-manager/internal/central/pkg/centrals/types"
	"github.com/stackrox/acs-fleet-manager/internal/central/pkg/config"
	"github.com/stackrox/acs-fleet-manager/pkg/services"
	"github.com/stackrox/acs-fleet-manager/pkg/services/sso"

	"github.com/stackrox/acs-fleet-manager/pkg/services/authorization"
	coreServices "github.com/stackrox/acs-fleet-manager/pkg/services/queryparser"

	"github.com/golang/glog"

	"github.com/aws/aws-sdk-go/service/route53"
	"github.com/stackrox/acs-fleet-manager/pkg/api"
	"github.com/stackrox/acs-fleet-manager/pkg/auth"
	"github.com/stackrox/acs-fleet-manager/pkg/client/aws"
	"github.com/stackrox/acs-fleet-manager/pkg/db"
	"github.com/stackrox/acs-fleet-manager/pkg/errors"
	"github.com/stackrox/acs-fleet-manager/pkg/logger"
	"github.com/stackrox/acs-fleet-manager/pkg/metrics"
)

var (
	centralDeletionStatuses = []string{
		centralConstants.CentralRequestStatusDeleting.String(),
		centralConstants.CentralRequestStatusDeprovision.String(),
	}

	centralManagedCRStatuses = []string{
		centralConstants.CentralRequestStatusProvisioning.String(),
		centralConstants.CentralRequestStatusDeprovision.String(),
		centralConstants.CentralRequestStatusReady.String(),
		centralConstants.CentralRequestStatusFailed.String(),
	}
)

// CentralRoutesAction ...
type CentralRoutesAction string

// CentralRoutesActionCreate ...
const CentralRoutesActionCreate CentralRoutesAction = "CREATE"

// CentralRoutesActionDelete ...
const CentralRoutesActionDelete CentralRoutesAction = "DELETE"

// CNameRecordStatus ...
type CNameRecordStatus struct {
	ID     *string
	Status *string
}

// CentralService ...
//
//go:generate moq -out centralservice_moq.go . CentralService
type CentralService interface {
	HasAvailableCapacity() (bool, *errors.ServiceError)
	// HasAvailableCapacityInRegion checks if there is capacity in the clusters for a given region
	HasAvailableCapacityInRegion(centralRequest *dbapi.CentralRequest) (bool, *errors.ServiceError)
	// AcceptCentralRequest transitions CentralRequest to 'Preparing'.
	AcceptCentralRequest(centralRequest *dbapi.CentralRequest) *errors.ServiceError
	// PrepareCentralRequest transitions CentralRequest to 'Provisioning'.
	PrepareCentralRequest(centralRequest *dbapi.CentralRequest) *errors.ServiceError
	// Get method will retrieve the centralRequest instance that the give ctx has access to from the database.
	// This should be used when you want to make sure the result is filtered based on the request context.
	Get(ctx context.Context, id string) (*dbapi.CentralRequest, *errors.ServiceError)
	// GetByID method will retrieve the CentralRequest instance from the database without checking any permissions.
	// You should only use this if you are sure permission check is not required.
	GetByID(id string) (*dbapi.CentralRequest, *errors.ServiceError)
	// Delete cleans up all dependencies for a Central request and soft deletes the Central Request record from the database.
	// The Central Request in the database will be updated with a deleted_at timestamp.
	Delete(*dbapi.CentralRequest) *errors.ServiceError
	List(ctx context.Context, listArgs *services.ListArguments) (dbapi.CentralList, *api.PagingMeta, *errors.ServiceError)
	ListByClusterID(clusterID string) ([]*dbapi.CentralRequest, *errors.ServiceError)
	RegisterCentralJob(centralRequest *dbapi.CentralRequest) *errors.ServiceError
	ListByStatus(status ...centralConstants.CentralStatus) ([]*dbapi.CentralRequest, *errors.ServiceError)
	// UpdateStatus change the status of the Central cluster
	// The returned boolean is to be used to know if the update has been tried or not. An update is not tried if the
	// original status is 'deprovision' (cluster in deprovision state can't be change state) or if the final status is the
	// same as the original status. The error will contain any error encountered when attempting to update or the reason
	// why no attempt has been done
	UpdateStatus(id string, status centralConstants.CentralStatus) (bool, *errors.ServiceError)
	Update(centralRequest *dbapi.CentralRequest) *errors.ServiceError
	// Updates updates the given fields of a central. This takes in a map so that even zero-fields can be updated.
	// Use this only when you want to update the multiple columns that may contain zero-fields, otherwise use the `CentralService.Update()` method.
	// See https://gorm.io/docs/update.html#Updates-multiple-columns for more info
	Updates(centralRequest *dbapi.CentralRequest, values map[string]interface{}) *errors.ServiceError
	ChangeCentralCNAMERecords(centralRequest *dbapi.CentralRequest, action CentralRoutesAction) (*route53.ChangeResourceRecordSetsOutput, *errors.ServiceError)
	GetCNAMERecordStatus(centralRequest *dbapi.CentralRequest) (*CNameRecordStatus, error)
	DetectInstanceType(centralRequest *dbapi.CentralRequest) (types.CentralInstanceType, *errors.ServiceError)
	RegisterCentralDeprovisionJob(ctx context.Context, id string) *errors.ServiceError
	// DeprovisionCentralForUsers registers all centrals for deprovisioning given the list of owners
	DeprovisionCentralForUsers(users []string) *errors.ServiceError
	DeprovisionExpiredCentrals(centralAgeInHours int) *errors.ServiceError
	CountByStatus(status []centralConstants.CentralStatus) ([]CentralStatusCount, error)
	CountByRegionAndInstanceType() ([]CentralRegionCount, error)
	ListCentralsWithRoutesNotCreated() ([]*dbapi.CentralRequest, *errors.ServiceError)
	ListCentralsWithoutAuthConfig() ([]*dbapi.CentralRequest, *errors.ServiceError)
	VerifyAndUpdateCentralAdmin(ctx context.Context, centralRequest *dbapi.CentralRequest) *errors.ServiceError
	ListComponentVersions() ([]CentralComponentVersions, error)
}

var _ CentralService = &centralService{}

type centralService struct {
	connectionFactory        *db.ConnectionFactory
	clusterService           ClusterService
	iamService               sso.IAMService
	centralConfig            *config.CentralConfig
	awsConfig                *config.AWSConfig
	quotaServiceFactory      QuotaServiceFactory
	mu                       sync.Mutex
	awsClientFactory         aws.ClientFactory
	authService              authorization.Authorization
	dataplaneClusterConfig   *config.DataplaneClusterConfig
	clusterPlacementStrategy ClusterPlacementStrategy
}

// NewCentralService ...
func NewCentralService(connectionFactory *db.ConnectionFactory, clusterService ClusterService, iamService sso.IAMService, centralConfig *config.CentralConfig, dataplaneClusterConfig *config.DataplaneClusterConfig, awsConfig *config.AWSConfig, quotaServiceFactory QuotaServiceFactory, awsClientFactory aws.ClientFactory, authorizationService authorization.Authorization, clusterPlacementStrategy ClusterPlacementStrategy) *centralService {
	return &centralService{
		connectionFactory:        connectionFactory,
		clusterService:           clusterService,
		iamService:               iamService,
		centralConfig:            centralConfig,
		awsConfig:                awsConfig,
		quotaServiceFactory:      quotaServiceFactory,
		awsClientFactory:         awsClientFactory,
		authService:              authorizationService,
		dataplaneClusterConfig:   dataplaneClusterConfig,
		clusterPlacementStrategy: clusterPlacementStrategy,
	}
}

// HasAvailableCapacity ...
func (k *centralService) HasAvailableCapacity() (bool, *errors.ServiceError) {
	dbConn := k.connectionFactory.New()
	var count int64

	if err := dbConn.Model(&dbapi.CentralRequest{}).Count(&count).Error; err != nil {
		return false, errors.NewWithCause(errors.ErrorGeneral, err, "failed to count central request")
	}

	glog.Infof("%d of %d central clusters currently instantiated", count, k.centralConfig.MaxCapacity.MaxCapacity)
	return count < k.centralConfig.MaxCapacity.MaxCapacity, nil
}

// HasAvailableCapacityInRegion ...
func (k *centralService) HasAvailableCapacityInRegion(centralRequest *dbapi.CentralRequest) (bool, *errors.ServiceError) {
	regionCapacity := int64(k.dataplaneClusterConfig.ClusterConfig.GetCapacityForRegion(centralRequest.Region))
	if regionCapacity <= 0 {
		return false, nil
	}

	dbConn := k.connectionFactory.New()
	var count int64
	if err := dbConn.Model(&dbapi.CentralRequest{}).Where("region = ?", centralRequest.Region).Count(&count).Error; err != nil {
		return false, errors.NewWithCause(errors.ErrorGeneral, err, "failed to count central request")
	}

	glog.Infof("%d of %d central clusters currently instantiated in region %v", count, regionCapacity, centralRequest.Region)
	return count < regionCapacity, nil
}

// DetectInstanceType ...
func (k *centralService) DetectInstanceType(centralRequest *dbapi.CentralRequest) (types.CentralInstanceType, *errors.ServiceError) {
	quotaType := api.QuotaType(k.centralConfig.Quota.Type)
	quotaService, factoryErr := k.quotaServiceFactory.GetQuotaService(quotaType)
	if factoryErr != nil {
		return "", errors.NewWithCause(errors.ErrorGeneral, factoryErr, "unable to check quota")
	}

	hasQuota, err := quotaService.CheckIfQuotaIsDefinedForInstanceType(centralRequest, types.STANDARD)
	if err != nil {
		return "", err
	}
	if hasQuota {
		glog.Infof("Quota detected for central request %s with quota type %s. Granting instance type %s.", centralRequest.ID, quotaType, types.STANDARD)
		return types.STANDARD, nil
	}

	glog.Infof("No quota detected for central request %s with quota type %s. Granting instance type %s.", centralRequest.ID, quotaType, types.EVAL)
	return types.EVAL, nil
}

// reserveQuota - reserves quota for the given central request. If a RHACS quota has been assigned, it will try to reserve RHACS quota, otherwise it will try with RHACSTrial
func (k *centralService) reserveQuota(centralRequest *dbapi.CentralRequest) (subscriptionID string, err *errors.ServiceError) {
	if centralRequest.InstanceType == types.EVAL.String() {
		if !k.centralConfig.Quota.AllowEvaluatorInstance {
			return "", errors.NewWithCause(errors.ErrorForbidden, err, "central eval instances are not allowed")
		}

		// Only one EVAL instance is admitted. Let's check if the user already owns one
		dbConn := k.connectionFactory.New()
		var count int64
		if err := dbConn.Model(&dbapi.CentralRequest{}).
			Where("instance_type = ?", types.EVAL).
			Where("owner = ?", centralRequest.Owner).
			Where("organisation_id = ?", centralRequest.OrganisationID).
			Count(&count).
			Error; err != nil {
			return "", errors.NewWithCause(errors.ErrorGeneral, err, "failed to count central eval instances")
		}

		if count > 0 {
			return "", errors.TooManyCentralInstancesReached("only one eval instance is allowed")
		}
	}

	quotaService, factoryErr := k.quotaServiceFactory.GetQuotaService(api.QuotaType(k.centralConfig.Quota.Type))
	if factoryErr != nil {
		return "", errors.NewWithCause(errors.ErrorGeneral, factoryErr, "unable to check quota")
	}
	subscriptionID, err = quotaService.ReserveQuota(centralRequest, types.CentralInstanceType(centralRequest.InstanceType))
	return subscriptionID, err
}

// RegisterCentralJob registers a new job in the central table
func (k *centralService) RegisterCentralJob(centralRequest *dbapi.CentralRequest) *errors.ServiceError {
	k.mu.Lock()
	defer k.mu.Unlock()
	// we need to pre-populate the ID to be able to reserve the quota
	centralRequest.ID = api.NewID()

	if hasCapacity, err := k.HasAvailableCapacityInRegion(centralRequest); err != nil {
		return errors.NewWithCause(errors.ErrorGeneral, err, "failed to create central request")
	} else if !hasCapacity {
		errorMsg := fmt.Sprintf("Cluster capacity(%d) exhausted in %s region", int64(k.dataplaneClusterConfig.ClusterConfig.GetCapacityForRegion(centralRequest.Region)), centralRequest.Region)
		logger.Logger.Warningf(errorMsg)
		return errors.TooManyCentralInstancesReached(errorMsg)
	}

	instanceType, err := k.DetectInstanceType(centralRequest)
	if err != nil {
		return err
	}

	centralRequest.InstanceType = instanceType.String()

	cluster, e := k.clusterPlacementStrategy.FindCluster(centralRequest)
	if e != nil || cluster == nil {
		msg := fmt.Sprintf("No available cluster found for '%s' central instance in region: '%s'", centralRequest.InstanceType, centralRequest.Region)
		logger.Logger.Errorf(msg)
		return errors.TooManyCentralInstancesReached(fmt.Sprintf("Region %s cannot accept instance type: %s at this moment", centralRequest.Region, centralRequest.InstanceType))
	}
	centralRequest.ClusterID = cluster.ClusterID
	subscriptionID, err := k.reserveQuota(centralRequest)
	if err != nil {
		return err
	}

	dbConn := k.connectionFactory.New()
	centralRequest.Status = centralConstants.CentralRequestStatusAccepted.String()
	centralRequest.SubscriptionID = subscriptionID
	glog.Infof("Central request %s has been assigned the subscription %s.", centralRequest.ID, subscriptionID)
	// Persist the QuotaType to be able to dynamically pick the right Quota service implementation even on restarts.
	// A typical usecase is when a central A is created, at the time of creation the quota-type was ams. At some point in the future
	// the API is restarted this time changing the --quota-type flag to quota-management-list, when central A is deleted at this point,
	// we want to use the correct quota to perform the deletion.
	centralRequest.QuotaType = k.centralConfig.Quota.Type
	if err := dbConn.Create(centralRequest).Error; err != nil {
		return errors.NewWithCause(errors.ErrorGeneral, err, "failed to create central request") // hide the db error to http caller
	}
	metrics.UpdateCentralRequestsStatusSinceCreatedMetric(centralConstants.CentralRequestStatusAccepted, centralRequest.ID, centralRequest.ClusterID, time.Since(centralRequest.CreatedAt))
	return nil
}

// AcceptCentralRequest sets any information about Central that does not
// require blocking operations (deducing namespace or instance hostname). Upon
// success, CentralRequest is transitioned to 'Preparing' status and might not
// be fully prepared yet.
func (k *centralService) AcceptCentralRequest(centralRequest *dbapi.CentralRequest) *errors.ServiceError {
	// Set namespace.
	namespace, formatErr := FormatNamespace(centralRequest.ID)
	if formatErr != nil {
		return errors.NewWithCause(errors.ErrorGeneral, formatErr, "invalid id format")
	}
	centralRequest.Namespace = namespace

	// Set host.
	if k.centralConfig.EnableCentralExternalCertificate {
		// If we enable centralTLS, the host should use the external domain name rather than the cluster domain
		centralRequest.Host = k.centralConfig.CentralDomainName
	} else {
		clusterDNS, err := k.clusterService.GetClusterDNS(centralRequest.ClusterID)
		if err != nil {
			return errors.NewWithCause(errors.ErrorGeneral, err, "error retrieving cluster DNS")
		}
		centralRequest.Host = clusterDNS
	}

	// Update the fields of the CentralRequest record in the database.
	updatedCentralRequest := &dbapi.CentralRequest{
		Meta: api.Meta{
			ID: centralRequest.ID,
		},
		Host:        centralRequest.Host,
		PlacementID: api.NewID(),
		Status:      centralConstants.CentralRequestStatusPreparing.String(),
		Namespace:   centralRequest.Namespace,
	}
	if err := k.Update(updatedCentralRequest); err != nil {
		return errors.NewWithCause(errors.ErrorGeneral, err, "failed to update central request")
	}

	return nil
}

// PrepareCentralRequest ensures that any required information (e.g.,
// CentralRequest's host, RHSSO auth config, etc) has been set. Upon success,
// the request is transitioned to 'Provisioning' status.
func (k *centralService) PrepareCentralRequest(centralRequest *dbapi.CentralRequest) *errors.ServiceError {
	// Check if the request is ready to be transitioned to provisioning.

	// Check IdP config is ready.
	//
	// TODO(alexr): Shall this go into "preparing_dinosaurs_mgr.go"? Ideally,
	//     all CentralRequest updating logic is in one place, either in this
	//     service or workers.
	if centralRequest.AuthConfig.ClientID == "" {
		// We can't provision this request, skip
		return nil
	}

	// Update the fields of the CentralRequest record in the database.
	updatedCentralRequest := &dbapi.CentralRequest{
		Meta: api.Meta{
			ID: centralRequest.ID,
		},
		Status: centralConstants.CentralRequestStatusProvisioning.String(),
	}
	if err := k.Update(updatedCentralRequest); err != nil {
		return errors.NewWithCause(errors.ErrorGeneral, err, "failed to update central request")
	}

	return nil
}

// ListByStatus ...
func (k *centralService) ListByStatus(status ...centralConstants.CentralStatus) ([]*dbapi.CentralRequest, *errors.ServiceError) {
	if len(status) == 0 {
		return nil, errors.GeneralError("no status provided")
	}
	dbConn := k.connectionFactory.New()

	var centrals []*dbapi.CentralRequest

	if err := dbConn.Model(&dbapi.CentralRequest{}).Where("status IN (?)", status).Scan(&centrals).Error; err != nil {
		return nil, errors.NewWithCause(errors.ErrorGeneral, err, "failed to list by status")
	}

	return centrals, nil
}

// Get ...
func (k *centralService) Get(ctx context.Context, id string) (*dbapi.CentralRequest, *errors.ServiceError) {
	if id == "" {
		return nil, errors.Validation("id is undefined")
	}

	claims, err := auth.GetClaimsFromContext(ctx)
	if err != nil {
		return nil, errors.NewWithCause(errors.ErrorUnauthenticated, err, "user not authenticated")
	}

	dbConn := k.connectionFactory.New().Where("id = ?", id)

	var user string
	if !auth.GetIsAdminFromContext(ctx) {
		user, _ = claims.GetUsername()
		if user == "" {
			return nil, errors.Unauthenticated("user not authenticated")
		}

		orgID, _ := claims.GetOrgID()
		filterByOrganisationID := auth.GetFilterByOrganisationFromContext(ctx)

		// filter by organisationId if a user is part of an organisation and is not allowed as a service account
		if filterByOrganisationID {
			dbConn = dbConn.Where("organisation_id = ?", orgID)
		} else {
			dbConn = dbConn.Where("owner = ?", user)
		}
	}

	var centralRequest dbapi.CentralRequest
	if err := dbConn.First(&centralRequest).Error; err != nil {
		resourceTypeStr := "CentralResource"
		if user != "" {
			resourceTypeStr = fmt.Sprintf("%s for user %s", resourceTypeStr, user)
		}
		return nil, services.HandleGetError(resourceTypeStr, "id", id, err)
	}
	return &centralRequest, nil
}

// GetByID ...
func (k *centralService) GetByID(id string) (*dbapi.CentralRequest, *errors.ServiceError) {
	if id == "" {
		return nil, errors.Validation("id is undefined")
	}

	dbConn := k.connectionFactory.New()
	var centralRequest dbapi.CentralRequest
	if err := dbConn.Where("id = ?", id).First(&centralRequest).Error; err != nil {
		return nil, services.HandleGetError("CentralResource", "id", id, err)
	}
	return &centralRequest, nil
}

// RegisterCentralDeprovisionJob registers a central deprovision job in the central table
func (k *centralService) RegisterCentralDeprovisionJob(ctx context.Context, id string) *errors.ServiceError {
	if id == "" {
		return errors.Validation("id is undefined")
	}

	// filter central request by owner to only retrieve request of the current authenticated user
	claims, err := auth.GetClaimsFromContext(ctx)
	if err != nil {
		return errors.NewWithCause(errors.ErrorUnauthenticated, err, "user not authenticated")
	}

	dbConn := k.connectionFactory.New()

	if auth.GetIsAdminFromContext(ctx) {
		dbConn = dbConn.Where("id = ?", id)
	} else if claims.IsOrgAdmin() {
		orgID, _ := claims.GetOrgID()
		dbConn = dbConn.Where("id = ?", id).Where("organisation_id = ?", orgID)
	} else {
		user, _ := claims.GetUsername()
		dbConn = dbConn.Where("id = ?", id).Where("owner = ? ", user)
	}

	var centralRequest dbapi.CentralRequest
	if err := dbConn.First(&centralRequest).Error; err != nil {
		return services.HandleGetError("CentralResource", "id", id, err)
	}
	metrics.IncreaseCentralTotalOperationsCountMetric(centralConstants.CentralOperationDeprovision)

	deprovisionStatus := centralConstants.CentralRequestStatusDeprovision

	if executed, err := k.UpdateStatus(id, deprovisionStatus); executed {
		if err != nil {
			return services.HandleGetError("CentralResource", "id", id, err)
		}
		metrics.IncreaseCentralSuccessOperationsCountMetric(centralConstants.CentralOperationDeprovision)
		metrics.UpdateCentralRequestsStatusSinceCreatedMetric(deprovisionStatus, centralRequest.ID, centralRequest.ClusterID, time.Since(centralRequest.CreatedAt))
	}

	return nil
}

// DeprovisionCentralForUsers registers all centrals for deprovisioning given the list of owners
func (k *centralService) DeprovisionCentralForUsers(users []string) *errors.ServiceError {
	now := time.Now()
	dbConn := k.connectionFactory.New().
		Model(&dbapi.CentralRequest{}).
		Where("owner IN (?)", users).
		Where("status NOT IN (?)", centralDeletionStatuses).
		Updates(map[string]interface{}{
			"status":             centralConstants.CentralRequestStatusDeprovision,
			"deletion_timestamp": now,
		})

	err := dbConn.Error
	if err != nil {
		return errors.NewWithCause(errors.ErrorGeneral, err, "Unable to deprovision central requests for users")
	}

	if dbConn.RowsAffected >= 1 {
		glog.Infof("%v centrals are now deprovisioning for users %v", dbConn.RowsAffected, users)
		var counter int64
		for ; counter < dbConn.RowsAffected; counter++ {
			metrics.IncreaseCentralTotalOperationsCountMetric(centralConstants.CentralOperationDeprovision)
			metrics.IncreaseCentralSuccessOperationsCountMetric(centralConstants.CentralOperationDeprovision)
		}
	}

	return nil
}

// DeprovisionExpiredCentrals cleaning up expired centrals
func (k *centralService) DeprovisionExpiredCentrals(centralAgeInHours int) *errors.ServiceError {
	now := time.Now()
	dbConn := k.connectionFactory.New().
		Model(&dbapi.CentralRequest{}).
		Where("instance_type = ?", types.EVAL.String()).
		Where("created_at  <=  ?", now.Add(-1*time.Duration(centralAgeInHours)*time.Hour)).
		Where("status NOT IN (?)", centralDeletionStatuses)

	db := dbConn.Updates(map[string]interface{}{
		"status":             centralConstants.CentralRequestStatusDeprovision,
		"deletion_timestamp": now,
	})
	err := db.Error
	if err != nil {
		return errors.NewWithCause(errors.ErrorGeneral, err, "unable to deprovision expired centrals")
	}

	if db.RowsAffected >= 1 {
		glog.Infof("%v central_request's lifespans are over %d hours and have had their status updated to deprovisioning", db.RowsAffected, centralAgeInHours)
		var counter int64
		for ; counter < db.RowsAffected; counter++ {
			metrics.IncreaseCentralTotalOperationsCountMetric(centralConstants.CentralOperationDeprovision)
			metrics.IncreaseCentralSuccessOperationsCountMetric(centralConstants.CentralOperationDeprovision)
		}
	}

	return nil
}

// Delete ...
func (k *centralService) Delete(centralRequest *dbapi.CentralRequest) *errors.ServiceError {
	dbConn := k.connectionFactory.New()

	// if the we don't have the clusterID we can only delete the row from the database
	if centralRequest.ClusterID != "" {
		routes, err := centralRequest.GetRoutes()
		if err != nil {
			return errors.NewWithCause(errors.ErrorGeneral, err, "failed to get routes")
		}
		// Only delete the routes when they are set
		if routes != nil && k.centralConfig.EnableCentralExternalCertificate {
			_, err := k.ChangeCentralCNAMERecords(centralRequest, CentralRoutesActionDelete)
			if err != nil {
				return err
			}
		}
	}

	// soft delete the central request
	if err := dbConn.Delete(centralRequest).Error; err != nil {
		return errors.NewWithCause(errors.ErrorGeneral, err, "unable to delete central request with id %s", centralRequest.ID)
	}

	metrics.IncreaseCentralTotalOperationsCountMetric(centralConstants.CentralOperationDelete)
	metrics.IncreaseCentralSuccessOperationsCountMetric(centralConstants.CentralOperationDelete)

	return nil
}

// List returns all central requests belonging to a user.
func (k *centralService) List(ctx context.Context, listArgs *services.ListArguments) (dbapi.CentralList, *api.PagingMeta, *errors.ServiceError) {
	var centralRequestList dbapi.CentralList
	dbConn := k.connectionFactory.New()
	pagingMeta := &api.PagingMeta{
		Page: listArgs.Page,
		Size: listArgs.Size,
	}

	claims, err := auth.GetClaimsFromContext(ctx)
	if err != nil {
		return nil, nil, errors.NewWithCause(errors.ErrorUnauthenticated, err, "user not authenticated")
	}

	if !auth.GetIsAdminFromContext(ctx) {
		user, _ := claims.GetUsername()
		if user == "" {
			return nil, nil, errors.Unauthenticated("user not authenticated")
		}

		orgID, _ := claims.GetOrgID()
		filterByOrganisationID := auth.GetFilterByOrganisationFromContext(ctx)

		// filter by organisationId if a user is part of an organisation and is not allowed as a service account
		if filterByOrganisationID {
			// filter central requests by organisation_id since the user is allowed to see all central requests of my id
			dbConn = dbConn.Where("organisation_id = ?", orgID)
		} else {
			// filter central requests by owner as we are dealing with service accounts which may not have an org id
			dbConn = dbConn.Where("owner = ?", user)
		}
	}

	// Apply search query
	if len(listArgs.Search) > 0 {
		searchDbQuery, err := coreServices.NewQueryParser().Parse(listArgs.Search)
		if err != nil {
			return centralRequestList, pagingMeta, errors.NewWithCause(errors.ErrorFailedToParseSearch, err, "Unable to list central requests: %s", err.Error())
		}
		dbConn = dbConn.Where(searchDbQuery.Query, searchDbQuery.Values...)
	}

	if len(listArgs.OrderBy) == 0 {
		// default orderBy name
		dbConn = dbConn.Order("name")
	}

	// Set the order by arguments if any
	for _, orderByArg := range listArgs.OrderBy {
		dbConn = dbConn.Order(orderByArg)
	}

	// set total, limit and paging (based on https://gitlab.cee.redhat.com/service/api-guidelines#user-content-paging)
	total := int64(pagingMeta.Total)
	dbConn.Model(&centralRequestList).Count(&total)
	pagingMeta.Total = int(total)
	if pagingMeta.Size > pagingMeta.Total {
		pagingMeta.Size = pagingMeta.Total
	}
	dbConn = dbConn.Offset((pagingMeta.Page - 1) * pagingMeta.Size).Limit(pagingMeta.Size)

	// execute query
	if err := dbConn.Find(&centralRequestList).Error; err != nil {
		return centralRequestList, pagingMeta, errors.NewWithCause(errors.ErrorGeneral, err, "Unable to list central requests")
	}

	return centralRequestList, pagingMeta, nil
}

// ListByClusterID returns a list of CentralRequests with specified clusterID
func (k *centralService) ListByClusterID(clusterID string) ([]*dbapi.CentralRequest, *errors.ServiceError) {
	dbConn := k.connectionFactory.New().
		Where("cluster_id = ?", clusterID).
		Where("status IN (?)", centralManagedCRStatuses).
		Where("host != ''")

	var centralRequestList dbapi.CentralList
	if err := dbConn.Find(&centralRequestList).Error; err != nil {
		return nil, errors.NewWithCause(errors.ErrorGeneral, err, "unable to list central requests")
	}

	return centralRequestList, nil
}

// Update ...
func (k *centralService) Update(centralRequest *dbapi.CentralRequest) *errors.ServiceError {
	dbConn := k.connectionFactory.New().
		Model(centralRequest).
		Where("status not IN (?)", centralDeletionStatuses) // ignore updates of central under deletion

	if err := dbConn.Updates(centralRequest).Error; err != nil {
		return errors.NewWithCause(errors.ErrorGeneral, err, "Failed to update central")
	}

	return nil
}

// Updates ...
func (k *centralService) Updates(centralRequest *dbapi.CentralRequest, fields map[string]interface{}) *errors.ServiceError {
	dbConn := k.connectionFactory.New().
		Model(centralRequest).
		Where("status not IN (?)", centralDeletionStatuses) // ignore updates of central under deletion

	if err := dbConn.Updates(fields).Error; err != nil {
		return errors.NewWithCause(errors.ErrorGeneral, err, "Failed to update central")
	}

	return nil
}

// VerifyAndUpdateCentralAdmin ...
func (k *centralService) VerifyAndUpdateCentralAdmin(ctx context.Context, centralRequest *dbapi.CentralRequest) *errors.ServiceError {
	if !auth.GetIsAdminFromContext(ctx) {
		return errors.New(errors.ErrorUnauthenticated, "User not authenticated")
	}

	cluster, svcErr := k.clusterService.FindClusterByID(centralRequest.ClusterID)
	if svcErr != nil {
		return errors.NewWithCause(errors.ErrorGeneral, svcErr, "Unable to find cluster associated with central request: %s", centralRequest.ID)
	}
	if cluster == nil {
		return errors.New(errors.ErrorValidation, fmt.Sprintf("Unable to get cluster for central %s", centralRequest.ID))
	}

	return k.Update(centralRequest)
}

// UpdateStatus ...
func (k *centralService) UpdateStatus(id string, status centralConstants.CentralStatus) (bool, *errors.ServiceError) {
	dbConn := k.connectionFactory.New()

	central, err := k.GetByID(id)
	if err != nil {
		return true, errors.NewWithCause(errors.ErrorGeneral, err, "failed to update status")
	}
	// only allow to change the status to "deleting" if the cluster is already in "deprovision" status
	if central.Status == centralConstants.CentralRequestStatusDeprovision.String() && status != centralConstants.CentralRequestStatusDeleting {
		return false, errors.GeneralError("failed to update status: cluster is deprovisioning")
	}

	if central.Status == status.String() {
		// no update needed
		return false, errors.GeneralError("failed to update status: the cluster %s is already in %s state", id, status.String())
	}

	update := &dbapi.CentralRequest{Status: status.String()}
	if status.String() == centralConstants.CentralRequestStatusDeprovision.String() {
		now := time.Now()
		update.DeletionTimestamp = &now
	}

	if err := dbConn.Model(&dbapi.CentralRequest{Meta: api.Meta{ID: id}}).Updates(update).Error; err != nil {
		return true, errors.NewWithCause(errors.ErrorGeneral, err, "Failed to update central status")
	}

	return true, nil
}

// ChangeCentralCNAMERecords ...
func (k *centralService) ChangeCentralCNAMERecords(centralRequest *dbapi.CentralRequest, action CentralRoutesAction) (*route53.ChangeResourceRecordSetsOutput, *errors.ServiceError) {
	routes, err := centralRequest.GetRoutes()
	if routes == nil || err != nil {
		return nil, errors.NewWithCause(errors.ErrorGeneral, err, "failed to get routes")
	}

	domainRecordBatch := buildCentralClusterCNAMESRecordBatch(routes, string(action))

	// Create AWS client with the region of this central Cluster
	awsConfig := aws.Config{
		AccessKeyID:     k.awsConfig.Route53AccessKey,
		SecretAccessKey: k.awsConfig.Route53SecretAccessKey, // pragma: allowlist secret
	}
	awsClient, err := k.awsClientFactory.NewClient(awsConfig, centralRequest.Region)
	if err != nil {
		return nil, errors.NewWithCause(errors.ErrorGeneral, err, "Unable to create aws client")
	}

	changeRecordsOutput, err := awsClient.ChangeResourceRecordSets(k.centralConfig.CentralDomainName, domainRecordBatch)
	if err != nil {
		return nil, errors.NewWithCause(errors.ErrorGeneral, err, "Unable to create domain record sets")
	}

	return changeRecordsOutput, nil
}

// GetCNAMERecordStatus ...
func (k *centralService) GetCNAMERecordStatus(centralRequest *dbapi.CentralRequest) (*CNameRecordStatus, error) {
	awsConfig := aws.Config{
		AccessKeyID:     k.awsConfig.Route53AccessKey,
		SecretAccessKey: k.awsConfig.Route53SecretAccessKey, // pragma: allowlist secret
	}
	awsClient, err := k.awsClientFactory.NewClient(awsConfig, centralRequest.Region)
	if err != nil {
		return nil, errors.NewWithCause(errors.ErrorGeneral, err, "Unable to create aws client")
	}

	changeOutput, err := awsClient.GetChange(centralRequest.RoutesCreationID)
	if err != nil {
		return nil, errors.NewWithCause(errors.ErrorGeneral, err, "Unable to CNAME record status")
	}

	return &CNameRecordStatus{
		ID:     changeOutput.ChangeInfo.Id,
		Status: changeOutput.ChangeInfo.Status,
	}, nil
}

// CentralStatusCount ...
type CentralStatusCount struct {
	Status centralConstants.CentralStatus
	Count  int
}

// CentralRegionCount ...
type CentralRegionCount struct {
	Region       string
	InstanceType string `gorm:"column:instance_type"`
	ClusterID    string `gorm:"column:cluster_id"`
	Count        int
}

// CountByRegionAndInstanceType ...
func (k *centralService) CountByRegionAndInstanceType() ([]CentralRegionCount, error) {
	dbConn := k.connectionFactory.New()
	var results []CentralRegionCount

	if err := dbConn.Model(&dbapi.CentralRequest{}).Select("region as Region, instance_type, cluster_id, count(1) as Count").Group("region,instance_type,cluster_id").Scan(&results).Error; err != nil {
		return nil, errors.NewWithCause(errors.ErrorGeneral, err, "Failed to count centrals")
	}

	return results, nil
}

// CountByStatus ...
func (k *centralService) CountByStatus(status []centralConstants.CentralStatus) ([]CentralStatusCount, error) {
	dbConn := k.connectionFactory.New()
	var results []CentralStatusCount
	if err := dbConn.Model(&dbapi.CentralRequest{}).Select("status as Status, count(1) as Count").Where("status in (?)", status).Group("status").Scan(&results).Error; err != nil {
		return nil, errors.NewWithCause(errors.ErrorGeneral, err, "Failed to count centrals")
	}

	// if there is no count returned for a status from the above query because there is no centrals in such a status,
	// we should return the count for these as well to avoid any confusion
	if len(status) > 0 {
		countersMap := map[centralConstants.CentralStatus]int{}
		for _, r := range results {
			countersMap[r.Status] = r.Count
		}
		for _, s := range status {
			if _, ok := countersMap[s]; !ok {
				results = append(results, CentralStatusCount{Status: s, Count: 0})
			}
		}
	}

	return results, nil
}

// CentralComponentVersions ...
type CentralComponentVersions struct {
	ID                            string
	ClusterID                     string
	DesiredCentralOperatorVersion string
	ActualCentralOperatorVersion  string
	CentralOperatorUpgrading      bool
	DesiredCentralVersion         string
	ActualCentralVersion          string
	CentralUpgrading              bool
}

// ListComponentVersions ...
func (k *centralService) ListComponentVersions() ([]CentralComponentVersions, error) {
	dbConn := k.connectionFactory.New()
	var results []CentralComponentVersions
	if err := dbConn.Model(&dbapi.CentralRequest{}).Select("id", "cluster_id", "desired_central_operator_version", "actual_central_operator_version", "central_operator_upgrading", "desired_central_version", "actual_central_version", "central_upgrading").Scan(&results).Error; err != nil {
		return nil, errors.NewWithCause(errors.ErrorGeneral, err, "failed to list component versions")
	}
	return results, nil
}

// ListCentralsWithRoutesNotCreated ...
func (k *centralService) ListCentralsWithRoutesNotCreated() ([]*dbapi.CentralRequest, *errors.ServiceError) {
	dbConn := k.connectionFactory.New()
	var results []*dbapi.CentralRequest
	if err := dbConn.Where("routes IS NOT NULL").Where("routes_created = ?", "no").Find(&results).Error; err != nil {
		return nil, errors.NewWithCause(errors.ErrorGeneral, err, "failed to list central requests")
	}
	return results, nil
}

// ListCentralsWithoutAuthConfig returns all _relevant_ central requests with
// no auth config.
func (k *centralService) ListCentralsWithoutAuthConfig() ([]*dbapi.CentralRequest, *errors.ServiceError) {
	dbQuery := k.connectionFactory.New().
		Where("client_id = ''")

	var results []*dbapi.CentralRequest
	if err := dbQuery.Find(&results).Error; err != nil {
		return nil, errors.NewWithCause(errors.ErrorGeneral, err, "failed to list Central requests")
	}

	// Central requests beyond 'Preparing' should have already been augmented.
	filteredResults := make([]*dbapi.CentralRequest, 0, len(results))
	for _, r := range results {
		if centralConstants.CentralStatus(r.Status).CompareTo(centralConstants.CentralRequestStatusPreparing) <= 0 {
			filteredResults = append(filteredResults, r)
		} else {
			glog.Warningf("Central request %s in status %q lacks auth config which should have been set up earlier", r.ID, r.Status)
		}
	}

	return filteredResults, nil
}

func buildCentralClusterCNAMESRecordBatch(routes []dbapi.DataPlaneCentralRoute, action string) *route53.ChangeBatch {
	var changes []*route53.Change
	for _, r := range routes {
		c := buildResourceRecordChange(r.Domain, r.Router, action)
		changes = append(changes, c)
	}
	recordChangeBatch := &route53.ChangeBatch{
		Changes: changes,
	}

	return recordChangeBatch
}

func buildResourceRecordChange(recordName string, clusterIngress string, action string) *route53.Change {
	recordType := "CNAME"
	recordTTL := int64(300)

	resourceRecordChange := &route53.Change{
		Action: &action,
		ResourceRecordSet: &route53.ResourceRecordSet{
			Name: &recordName,
			Type: &recordType,
			TTL:  &recordTTL,
			ResourceRecords: []*route53.ResourceRecord{
				{
					Value: &clusterIngress,
				},
			},
		},
	}

	return resourceRecordChange
}
