package services

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/stackrox/acs-fleet-manager/pkg/services/sso"

	constants2 "github.com/stackrox/acs-fleet-manager/internal/dinosaur/constants"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/api/dbapi"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/config"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/dinosaurs/types"
	"github.com/stackrox/acs-fleet-manager/pkg/client/keycloak"
	"github.com/stackrox/acs-fleet-manager/pkg/services"

	"github.com/stackrox/acs-fleet-manager/pkg/services/authorization"
	coreServices "github.com/stackrox/acs-fleet-manager/pkg/services/queryparser"

	"time"

	"github.com/golang/glog"

	manageddinosaur "github.com/stackrox/acs-fleet-manager/pkg/api/manageddinosaurs.manageddinosaur.mas/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/aws/aws-sdk-go/service/route53"
	"github.com/stackrox/acs-fleet-manager/pkg/api"
	"github.com/stackrox/acs-fleet-manager/pkg/auth"
	"github.com/stackrox/acs-fleet-manager/pkg/client/aws"
	"github.com/stackrox/acs-fleet-manager/pkg/db"
	"github.com/stackrox/acs-fleet-manager/pkg/errors"
	"github.com/stackrox/acs-fleet-manager/pkg/logger"
	"github.com/stackrox/acs-fleet-manager/pkg/metrics"
)

var dinosaurDeletionStatuses = []string{constants2.DinosaurRequestStatusDeleting.String(), constants2.DinosaurRequestStatusDeprovision.String()}
var dinosaurManagedCRStatuses = []string{constants2.DinosaurRequestStatusProvisioning.String(), constants2.DinosaurRequestStatusDeprovision.String(), constants2.DinosaurRequestStatusReady.String(), constants2.DinosaurRequestStatusFailed.String()}

type DinosaurRoutesAction string

const DinosaurRoutesActionCreate DinosaurRoutesAction = "CREATE"
const DinosaurRoutesActionDelete DinosaurRoutesAction = "DELETE"

type CNameRecordStatus struct {
	Id     *string
	Status *string
}

//go:generate moq -out dinosaurservice_moq.go . DinosaurService
type DinosaurService interface {
	HasAvailableCapacity() (bool, *errors.ServiceError)
	// HasAvailableCapacityInRegion checks if there is capacity in the clusters for a given region
	HasAvailableCapacityInRegion(dinosaurRequest *dbapi.CentralRequest) (bool, *errors.ServiceError)
	// PrepareDinosaurRequest sets any required information (i.e. dinosaur host, sso client id and secret)
	// to the Dinosaur Request record in the database. The dinosaur request will also be updated with an updated_at
	// timestamp and the corresponding cluster identifier.
	PrepareDinosaurRequest(dinosaurRequest *dbapi.CentralRequest) *errors.ServiceError
	// Get method will retrieve the dinosaurRequest instance that the give ctx has access to from the database.
	// This should be used when you want to make sure the result is filtered based on the request context.
	Get(ctx context.Context, id string) (*dbapi.CentralRequest, *errors.ServiceError)
	// GetById method will retrieve the DinosaurRequest instance from the database without checking any permissions.
	// You should only use this if you are sure permission check is not required.
	GetById(id string) (*dbapi.CentralRequest, *errors.ServiceError)
	// Delete cleans up all dependencies for a Dinosaur request and soft deletes the Dinosaur Request record from the database.
	// The Dinosaur Request in the database will be updated with a deleted_at timestamp.
	Delete(*dbapi.CentralRequest) *errors.ServiceError
	List(ctx context.Context, listArgs *services.ListArguments) (dbapi.CentralList, *api.PagingMeta, *errors.ServiceError)
	GetManagedDinosaurByClusterID(clusterID string) ([]manageddinosaur.ManagedDinosaur, *errors.ServiceError)
	RegisterDinosaurJob(dinosaurRequest *dbapi.CentralRequest) *errors.ServiceError
	ListByStatus(status ...constants2.DinosaurStatus) ([]*dbapi.CentralRequest, *errors.ServiceError)
	// UpdateStatus change the status of the Dinosaur cluster
	// The returned boolean is to be used to know if the update has been tried or not. An update is not tried if the
	// original status is 'deprovision' (cluster in deprovision state can't be change state) or if the final status is the
	// same as the original status. The error will contain any error encountered when attempting to update or the reason
	// why no attempt has been done
	UpdateStatus(id string, status constants2.DinosaurStatus) (bool, *errors.ServiceError)
	Update(dinosaurRequest *dbapi.CentralRequest) *errors.ServiceError
	// Updates() updates the given fields of a dinosaur. This takes in a map so that even zero-fields can be updated.
	// Use this only when you want to update the multiple columns that may contain zero-fields, otherwise use the `DinosaurService.Update()` method.
	// See https://gorm.io/docs/update.html#Updates-multiple-columns for more info
	Updates(dinosaurRequest *dbapi.CentralRequest, values map[string]interface{}) *errors.ServiceError
	ChangeDinosaurCNAMErecords(dinosaurRequest *dbapi.CentralRequest, action DinosaurRoutesAction) (*route53.ChangeResourceRecordSetsOutput, *errors.ServiceError)
	GetCNAMERecordStatus(dinosaurRequest *dbapi.CentralRequest) (*CNameRecordStatus, error)
	DetectInstanceType(dinosaurRequest *dbapi.CentralRequest) (types.DinosaurInstanceType, *errors.ServiceError)
	RegisterDinosaurDeprovisionJob(ctx context.Context, id string) *errors.ServiceError
	// DeprovisionDinosaurForUsers registers all dinosaurs for deprovisioning given the list of owners
	DeprovisionDinosaurForUsers(users []string) *errors.ServiceError
	DeprovisionExpiredDinosaurs(dinosaurAgeInHours int) *errors.ServiceError
	CountByStatus(status []constants2.DinosaurStatus) ([]DinosaurStatusCount, error)
	CountByRegionAndInstanceType() ([]DinosaurRegionCount, error)
	ListDinosaursWithRoutesNotCreated() ([]*dbapi.CentralRequest, *errors.ServiceError)
	VerifyAndUpdateDinosaurAdmin(ctx context.Context, dinosaurRequest *dbapi.CentralRequest) *errors.ServiceError
	ListComponentVersions() ([]DinosaurComponentVersions, error)
}

var _ DinosaurService = &dinosaurService{}

type dinosaurService struct {
	connectionFactory        *db.ConnectionFactory
	clusterService           ClusterService
	keycloakService          sso.KeycloakService
	dinosaurConfig           *config.DinosaurConfig
	awsConfig                *config.AWSConfig
	quotaServiceFactory      QuotaServiceFactory
	mu                       sync.Mutex
	awsClientFactory         aws.ClientFactory
	authService              authorization.Authorization
	dataplaneClusterConfig   *config.DataplaneClusterConfig
	clusterPlacementStrategy ClusterPlacementStrategy
}

func NewDinosaurService(connectionFactory *db.ConnectionFactory, clusterService ClusterService, keycloakService sso.KeycloakService, dinosaurConfig *config.DinosaurConfig, dataplaneClusterConfig *config.DataplaneClusterConfig, awsConfig *config.AWSConfig, quotaServiceFactory QuotaServiceFactory, awsClientFactory aws.ClientFactory, authorizationService authorization.Authorization, clusterPlacementStrategy ClusterPlacementStrategy) *dinosaurService {
	return &dinosaurService{
		connectionFactory:        connectionFactory,
		clusterService:           clusterService,
		keycloakService:          keycloakService,
		dinosaurConfig:           dinosaurConfig,
		awsConfig:                awsConfig,
		quotaServiceFactory:      quotaServiceFactory,
		awsClientFactory:         awsClientFactory,
		authService:              authorizationService,
		dataplaneClusterConfig:   dataplaneClusterConfig,
		clusterPlacementStrategy: clusterPlacementStrategy,
	}
}

func (k *dinosaurService) HasAvailableCapacity() (bool, *errors.ServiceError) {
	dbConn := k.connectionFactory.New()
	var count int64

	if err := dbConn.Model(&dbapi.CentralRequest{}).Count(&count).Error; err != nil {
		return false, errors.NewWithCause(errors.ErrorGeneral, err, "failed to count dinosaur request")
	}

	glog.Infof("%d of %d dinosaur clusters currently instantiated", count, k.dinosaurConfig.MaxCapacity.MaxCapacity)
	return count < k.dinosaurConfig.MaxCapacity.MaxCapacity, nil
}

func (k *dinosaurService) HasAvailableCapacityInRegion(dinosaurRequest *dbapi.CentralRequest) (bool, *errors.ServiceError) {
	regionCapacity := int64(k.dataplaneClusterConfig.ClusterConfig.GetCapacityForRegion(dinosaurRequest.Region))
	if regionCapacity <= 0 {
		return false, nil
	}

	dbConn := k.connectionFactory.New()
	var count int64
	if err := dbConn.Model(&dbapi.CentralRequest{}).Where("region = ?", dinosaurRequest.Region).Count(&count).Error; err != nil {
		return false, errors.NewWithCause(errors.ErrorGeneral, err, "failed to count dinosaur request")
	}

	glog.Infof("%d of %d dinosaur clusters currently instantiated in region %v", count, regionCapacity, dinosaurRequest.Region)
	return count < regionCapacity, nil
}

func (k *dinosaurService) DetectInstanceType(dinosaurRequest *dbapi.CentralRequest) (types.DinosaurInstanceType, *errors.ServiceError) {
	quotaService, factoryErr := k.quotaServiceFactory.GetQuotaService(api.QuotaType(k.dinosaurConfig.Quota.Type))
	if factoryErr != nil {
		return "", errors.NewWithCause(errors.ErrorGeneral, factoryErr, "unable to check quota")
	}

	hasRhosakQuota, err := quotaService.CheckIfQuotaIsDefinedForInstanceType(dinosaurRequest, types.STANDARD)
	if err != nil {
		return "", err
	}
	if hasRhosakQuota {
		return types.STANDARD, nil
	}

	return types.EVAL, nil
}

// reserveQuota - reserves quota for the given dinosaur request. If a RHOSAK quota has been assigned, it will try to reserve RHOSAK quota, otherwise it will try with RHOSAKTrial
func (k *dinosaurService) reserveQuota(dinosaurRequest *dbapi.CentralRequest) (subscriptionId string, err *errors.ServiceError) {
	if dinosaurRequest.InstanceType == types.EVAL.String() {
		if !k.dinosaurConfig.Quota.AllowEvaluatorInstance {
			return "", errors.NewWithCause(errors.ErrorForbidden, err, "dinosaur eval instances are not allowed")
		}

		// Only one EVAL instance is admitted. Let's check if the user already owns one
		dbConn := k.connectionFactory.New()
		var count int64
		if err := dbConn.Model(&dbapi.CentralRequest{}).
			Where("instance_type = ?", types.EVAL).
			Where("owner = ?", dinosaurRequest.Owner).
			Where("organisation_id = ?", dinosaurRequest.OrganisationId).
			Count(&count).
			Error; err != nil {
			return "", errors.NewWithCause(errors.ErrorGeneral, err, "failed to count dinosaur eval instances")
		}

		if count > 0 {
			return "", errors.TooManyDinosaurInstancesReached("only one eval instance is allowed")
		}
	}

	quotaService, factoryErr := k.quotaServiceFactory.GetQuotaService(api.QuotaType(k.dinosaurConfig.Quota.Type))
	if factoryErr != nil {
		return "", errors.NewWithCause(errors.ErrorGeneral, factoryErr, "unable to check quota")
	}
	subscriptionId, err = quotaService.ReserveQuota(dinosaurRequest, types.DinosaurInstanceType(dinosaurRequest.InstanceType))
	return subscriptionId, err
}

// RegisterDinosaurJob registers a new job in the dinosaur table
func (k *dinosaurService) RegisterDinosaurJob(dinosaurRequest *dbapi.CentralRequest) *errors.ServiceError {
	k.mu.Lock()
	defer k.mu.Unlock()
	// we need to pre-populate the ID to be able to reserve the quota
	dinosaurRequest.ID = api.NewID()

	if hasCapacity, err := k.HasAvailableCapacityInRegion(dinosaurRequest); err != nil {
		return errors.NewWithCause(errors.ErrorGeneral, err, "failed to create dinosaur request")
	} else if !hasCapacity {
		errorMsg := fmt.Sprintf("Cluster capacity(%d) exhausted in %s region", int64(k.dataplaneClusterConfig.ClusterConfig.GetCapacityForRegion(dinosaurRequest.Region)), dinosaurRequest.Region)
		logger.Logger.Warningf(errorMsg)
		return errors.TooManyDinosaurInstancesReached(errorMsg)
	}

	instanceType, err := k.DetectInstanceType(dinosaurRequest)
	if err != nil {
		return err
	}

	dinosaurRequest.InstanceType = instanceType.String()

	cluster, e := k.clusterPlacementStrategy.FindCluster(dinosaurRequest)
	if e != nil || cluster == nil {
		msg := fmt.Sprintf("No available cluster found for '%s' central instance in region: '%s'", dinosaurRequest.InstanceType, dinosaurRequest.Region)
		logger.Logger.Errorf(msg)
		return errors.TooManyDinosaurInstancesReached(fmt.Sprintf("Region %s cannot accept instance type: %s at this moment", dinosaurRequest.Region, dinosaurRequest.InstanceType))
	}
	dinosaurRequest.ClusterID = cluster.ClusterID
	subscriptionId, err := k.reserveQuota(dinosaurRequest)

	if err != nil {
		return err
	}

	dbConn := k.connectionFactory.New()
	dinosaurRequest.Status = constants2.DinosaurRequestStatusAccepted.String()
	dinosaurRequest.SubscriptionId = subscriptionId

	// Persist the QuotaTyoe to be able to dynamically pick the right Quota service implementation even on restarts.
	// A typical usecase is when a dinosaur A is created, at the time of creation the quota-type was ams. At some point in the future
	// the API is restarted this time changing the --quota-type flag to quota-management-list, when dinosaur A is deleted at this point,
	// we want to use the correct quota to perform the deletion.
	dinosaurRequest.QuotaType = k.dinosaurConfig.Quota.Type
	if err := dbConn.Create(dinosaurRequest).Error; err != nil {
		return errors.NewWithCause(errors.ErrorGeneral, err, "failed to create dinosaur request") //hide the db error to http caller
	}
	metrics.UpdateDinosaurRequestsStatusSinceCreatedMetric(constants2.DinosaurRequestStatusAccepted, dinosaurRequest.ID, dinosaurRequest.ClusterID, time.Since(dinosaurRequest.CreatedAt))
	return nil
}

func (k *dinosaurService) PrepareDinosaurRequest(dinosaurRequest *dbapi.CentralRequest) *errors.ServiceError {
	truncatedDinosaurIdentifier := buildTruncateDinosaurIdentifier(dinosaurRequest)
	truncatedDinosaurIdentifier, replaceErr := replaceHostSpecialChar(truncatedDinosaurIdentifier)
	if replaceErr != nil {
		return errors.NewWithCause(errors.ErrorGeneral, replaceErr, "generated host is not valid")
	}

	clusterDNS, err := k.clusterService.GetClusterDNS(dinosaurRequest.ClusterID)
	if err != nil {
		return errors.NewWithCause(errors.ErrorGeneral, err, "error retrieving cluster DNS")
	}

	namespace, formatErr := formatNamespace(dinosaurRequest.ID)
	if formatErr != nil {
		return errors.NewWithCause(errors.ErrorGeneral, formatErr, "invalid id format")
	}
	dinosaurRequest.Namespace = namespace
	clusterDNS = strings.Replace(clusterDNS, constants2.DefaultIngressDnsNamePrefix, constants2.ManagedDinosaurIngressDnsNamePrefix, 1)
	dinosaurRequest.Host = fmt.Sprintf("%s.%s", truncatedDinosaurIdentifier, clusterDNS)

	if k.dinosaurConfig.EnableDinosaurExternalCertificate {
		// If we enable DinosaurTLS, the host should use the external domain name rather than the cluster domain
		dinosaurRequest.Host = fmt.Sprintf("%s.%s", truncatedDinosaurIdentifier, k.dinosaurConfig.DinosaurDomainName)
	}

	// Update the Dinosaur Request record in the database
	// Only updates the fields below
	updatedDinosaurRequest := &dbapi.CentralRequest{
		Meta: api.Meta{
			ID: dinosaurRequest.ID,
		},
		Host:        dinosaurRequest.Host,
		PlacementId: api.NewID(),
		Status:      constants2.DinosaurRequestStatusProvisioning.String(),
		Namespace:   dinosaurRequest.Namespace,
	}
	if err := k.Update(updatedDinosaurRequest); err != nil {
		return errors.NewWithCause(errors.ErrorGeneral, err, "failed to update dinosaur request")
	}

	return nil
}

func (k *dinosaurService) ListByStatus(status ...constants2.DinosaurStatus) ([]*dbapi.CentralRequest, *errors.ServiceError) {
	if len(status) == 0 {
		return nil, errors.GeneralError("no status provided")
	}
	dbConn := k.connectionFactory.New()

	var dinosaurs []*dbapi.CentralRequest

	if err := dbConn.Model(&dbapi.CentralRequest{}).Where("status IN (?)", status).Scan(&dinosaurs).Error; err != nil {
		return nil, errors.NewWithCause(errors.ErrorGeneral, err, "failed to list by status")
	}

	return dinosaurs, nil
}

func (k *dinosaurService) Get(ctx context.Context, id string) (*dbapi.CentralRequest, *errors.ServiceError) {
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

		orgId, _ := claims.GetOrgId()
		filterByOrganisationId := auth.GetFilterByOrganisationFromContext(ctx)

		// filter by organisationId if a user is part of an organisation and is not allowed as a service account
		if filterByOrganisationId {
			dbConn = dbConn.Where("organisation_id = ?", orgId)
		} else {
			dbConn = dbConn.Where("owner = ?", user)
		}
	}

	var dinosaurRequest dbapi.CentralRequest
	if err := dbConn.First(&dinosaurRequest).Error; err != nil {
		resourceTypeStr := "DinosaurResource"
		if user != "" {
			resourceTypeStr = fmt.Sprintf("%s for user %s", resourceTypeStr, user)
		}
		return nil, services.HandleGetError(resourceTypeStr, "id", id, err)
	}
	return &dinosaurRequest, nil
}

func (k *dinosaurService) GetById(id string) (*dbapi.CentralRequest, *errors.ServiceError) {
	if id == "" {
		return nil, errors.Validation("id is undefined")
	}

	dbConn := k.connectionFactory.New()
	var dinosaurRequest dbapi.CentralRequest
	if err := dbConn.Where("id = ?", id).First(&dinosaurRequest).Error; err != nil {
		return nil, services.HandleGetError("DinosaurResource", "id", id, err)
	}
	return &dinosaurRequest, nil
}

// RegisterDinosaurDeprovisionJob registers a dinosaur deprovision job in the dinosaur table
func (k *dinosaurService) RegisterDinosaurDeprovisionJob(ctx context.Context, id string) *errors.ServiceError {
	if id == "" {
		return errors.Validation("id is undefined")
	}

	// filter dinosaur request by owner to only retrieve request of the current authenticated user
	claims, err := auth.GetClaimsFromContext(ctx)
	if err != nil {
		return errors.NewWithCause(errors.ErrorUnauthenticated, err, "user not authenticated")
	}

	dbConn := k.connectionFactory.New()

	if auth.GetIsAdminFromContext(ctx) {
		dbConn = dbConn.Where("id = ?", id)
	} else if claims.IsOrgAdmin() {
		orgId, _ := claims.GetOrgId()
		dbConn = dbConn.Where("id = ?", id).Where("organisation_id = ?", orgId)
	} else {
		user, _ := claims.GetUsername()
		dbConn = dbConn.Where("id = ?", id).Where("owner = ? ", user)
	}

	var dinosaurRequest dbapi.CentralRequest
	if err := dbConn.First(&dinosaurRequest).Error; err != nil {
		return services.HandleGetError("DinosaurResource", "id", id, err)
	}
	metrics.IncreaseDinosaurTotalOperationsCountMetric(constants2.DinosaurOperationDeprovision)

	deprovisionStatus := constants2.DinosaurRequestStatusDeprovision

	if executed, err := k.UpdateStatus(id, deprovisionStatus); executed {
		if err != nil {
			return services.HandleGetError("DinosaurResource", "id", id, err)
		}
		metrics.IncreaseDinosaurSuccessOperationsCountMetric(constants2.DinosaurOperationDeprovision)
		metrics.UpdateDinosaurRequestsStatusSinceCreatedMetric(deprovisionStatus, dinosaurRequest.ID, dinosaurRequest.ClusterID, time.Since(dinosaurRequest.CreatedAt))
	}

	return nil
}

func (k *dinosaurService) DeprovisionDinosaurForUsers(users []string) *errors.ServiceError {
	dbConn := k.connectionFactory.New().
		Model(&dbapi.CentralRequest{}).
		Where("owner IN (?)", users).
		Where("status NOT IN (?)", dinosaurDeletionStatuses).
		Update("status", constants2.DinosaurRequestStatusDeprovision)

	err := dbConn.Error
	if err != nil {
		return errors.NewWithCause(errors.ErrorGeneral, err, "Unable to deprovision dinosaur requests for users")
	}

	if dbConn.RowsAffected >= 1 {
		glog.Infof("%v dinosaurs are now deprovisioning for users %v", dbConn.RowsAffected, users)
		var counter int64 = 0
		for ; counter < dbConn.RowsAffected; counter++ {
			metrics.IncreaseDinosaurTotalOperationsCountMetric(constants2.DinosaurOperationDeprovision)
			metrics.IncreaseDinosaurSuccessOperationsCountMetric(constants2.DinosaurOperationDeprovision)
		}
	}

	return nil
}

func (k *dinosaurService) DeprovisionExpiredDinosaurs(dinosaurAgeInHours int) *errors.ServiceError {
	dbConn := k.connectionFactory.New().
		Model(&dbapi.CentralRequest{}).
		Where("instance_type = ?", types.EVAL.String()).
		Where("created_at  <=  ?", time.Now().Add(-1*time.Duration(dinosaurAgeInHours)*time.Hour)).
		Where("status NOT IN (?)", dinosaurDeletionStatuses)

	db := dbConn.Update("status", constants2.DinosaurRequestStatusDeprovision)
	err := db.Error
	if err != nil {
		return errors.NewWithCause(errors.ErrorGeneral, err, "unable to deprovision expired dinosaurs")
	}

	if db.RowsAffected >= 1 {
		glog.Infof("%v dinosaur_request's lifespans are over %d hours and have had their status updated to deprovisioning", db.RowsAffected, dinosaurAgeInHours)
		var counter int64 = 0
		for ; counter < db.RowsAffected; counter++ {
			metrics.IncreaseDinosaurTotalOperationsCountMetric(constants2.DinosaurOperationDeprovision)
			metrics.IncreaseDinosaurSuccessOperationsCountMetric(constants2.DinosaurOperationDeprovision)
		}
	}

	return nil
}

func (k *dinosaurService) Delete(dinosaurRequest *dbapi.CentralRequest) *errors.ServiceError {
	dbConn := k.connectionFactory.New()

	// if the we don't have the clusterID we can only delete the row from the database
	if dinosaurRequest.ClusterID != "" {
		routes, err := dinosaurRequest.GetRoutes()
		if err != nil {
			return errors.NewWithCause(errors.ErrorGeneral, err, "failed to get routes")
		}
		// Only delete the routes when they are set
		if routes != nil && k.dinosaurConfig.EnableDinosaurExternalCertificate {
			_, err := k.ChangeDinosaurCNAMErecords(dinosaurRequest, DinosaurRoutesActionDelete)
			if err != nil {
				return err
			}
		}
	}

	// soft delete the dinosaur request
	if err := dbConn.Delete(dinosaurRequest).Error; err != nil {
		return errors.NewWithCause(errors.ErrorGeneral, err, "unable to delete dinosaur request with id %s", dinosaurRequest.ID)
	}

	metrics.IncreaseDinosaurTotalOperationsCountMetric(constants2.DinosaurOperationDelete)
	metrics.IncreaseDinosaurSuccessOperationsCountMetric(constants2.DinosaurOperationDelete)

	return nil
}

// List returns all Dinosaur requests belonging to a user.
func (k *dinosaurService) List(ctx context.Context, listArgs *services.ListArguments) (dbapi.CentralList, *api.PagingMeta, *errors.ServiceError) {
	var dinosaurRequestList dbapi.CentralList
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

		orgId, _ := claims.GetOrgId()
		filterByOrganisationId := auth.GetFilterByOrganisationFromContext(ctx)

		// filter by organisationId if a user is part of an organisation and is not allowed as a service account
		if filterByOrganisationId {
			// filter dinosaur requests by organisation_id since the user is allowed to see all dinosaur requests of my id
			dbConn = dbConn.Where("organisation_id = ?", orgId)
		} else {
			// filter dinosaur requests by owner as we are dealing with service accounts which may not have an org id
			dbConn = dbConn.Where("owner = ?", user)
		}
	}

	// Apply search query
	if len(listArgs.Search) > 0 {
		searchDbQuery, err := coreServices.NewQueryParser().Parse(listArgs.Search)
		if err != nil {
			return dinosaurRequestList, pagingMeta, errors.NewWithCause(errors.ErrorFailedToParseSearch, err, "Unable to list dinosaur requests: %s", err.Error())
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
	dbConn.Model(&dinosaurRequestList).Count(&total)
	pagingMeta.Total = int(total)
	if pagingMeta.Size > pagingMeta.Total {
		pagingMeta.Size = pagingMeta.Total
	}
	dbConn = dbConn.Offset((pagingMeta.Page - 1) * pagingMeta.Size).Limit(pagingMeta.Size)

	// execute query
	if err := dbConn.Find(&dinosaurRequestList).Error; err != nil {
		return dinosaurRequestList, pagingMeta, errors.NewWithCause(errors.ErrorGeneral, err, "Unable to list dinosaur requests")
	}

	return dinosaurRequestList, pagingMeta, nil
}

func (k *dinosaurService) GetManagedDinosaurByClusterID(clusterID string) ([]manageddinosaur.ManagedDinosaur, *errors.ServiceError) {
	dbConn := k.connectionFactory.New().
		Where("cluster_id = ?", clusterID).
		Where("status IN (?)", dinosaurManagedCRStatuses).
		Where("host != ''")

	var dinosaurRequestList dbapi.CentralList
	if err := dbConn.Find(&dinosaurRequestList).Error; err != nil {
		return nil, errors.NewWithCause(errors.ErrorGeneral, err, "unable to list dinosaur requests")
	}

	var res []manageddinosaur.ManagedDinosaur
	// convert dinosaur requests to managed dinosaur
	for _, dinosaurRequest := range dinosaurRequestList {
		mk := BuildManagedDinosaurCR(dinosaurRequest, k.dinosaurConfig, k.keycloakService.GetConfig())
		res = append(res, *mk)
	}

	return res, nil
}

func (k *dinosaurService) Update(dinosaurRequest *dbapi.CentralRequest) *errors.ServiceError {
	dbConn := k.connectionFactory.New().
		Model(dinosaurRequest).
		Where("status not IN (?)", dinosaurDeletionStatuses) // ignore updates of dinosaur under deletion

	if err := dbConn.Updates(dinosaurRequest).Error; err != nil {
		return errors.NewWithCause(errors.ErrorGeneral, err, "Failed to update dinosaur")
	}

	return nil
}

func (k *dinosaurService) Updates(dinosaurRequest *dbapi.CentralRequest, fields map[string]interface{}) *errors.ServiceError {
	dbConn := k.connectionFactory.New().
		Model(dinosaurRequest).
		Where("status not IN (?)", dinosaurDeletionStatuses) // ignore updates of dinosaur under deletion

	if err := dbConn.Updates(fields).Error; err != nil {
		return errors.NewWithCause(errors.ErrorGeneral, err, "Failed to update dinosaur")
	}

	return nil
}

func (k *dinosaurService) VerifyAndUpdateDinosaurAdmin(ctx context.Context, dinosaurRequest *dbapi.CentralRequest) *errors.ServiceError {
	if auth.GetIsAdminFromContext(ctx) {
		cluster, err := k.clusterService.FindClusterByID(dinosaurRequest.ClusterID)
		if err != nil {
			return errors.NewWithCause(errors.ErrorGeneral, err, "Unable to find cluster associated with dinosaur request: %s", dinosaurRequest.ID)
		}
		if cluster == nil {
			return errors.New(errors.ErrorValidation, fmt.Sprintf("Unable to get cluster for dinosaur %s", dinosaurRequest.ID))
		}

		dinosaurVersionAvailable, err2 := k.clusterService.IsDinosaurVersionAvailableInCluster(cluster, dinosaurRequest.DesiredCentralOperatorVersion, dinosaurRequest.DesiredCentralVersion)
		if err2 != nil {
			return errors.Validation(err2.Error())
		}

		if !dinosaurVersionAvailable {
			return errors.New(errors.ErrorValidation, fmt.Sprintf("Unable to update dinosaur: %s with dinosaur version: %s", dinosaurRequest.ID, dinosaurRequest.DesiredCentralVersion))
		}

		dinosaruOperatorVersionReady, err2 := k.clusterService.CheckDinosaurOperatorVersionReady(cluster, dinosaurRequest.DesiredCentralOperatorVersion)
		if err2 != nil {
			return errors.Validation(err2.Error())
		}

		if !dinosaruOperatorVersionReady {
			return errors.New(errors.ErrorValidation, fmt.Sprintf("Unable to update dinosaur: %s with dinosaur operator version: %s", dinosaurRequest.ID, dinosaurRequest.DesiredCentralOperatorVersion))
		}

		vCompDinosaur, ek := api.CompareSemanticVersionsMajorAndMinor(dinosaurRequest.ActualCentralVersion, dinosaurRequest.DesiredCentralVersion)

		if ek != nil {
			return errors.New(errors.ErrorValidation, fmt.Sprintf("Unable to compare desired dinosaur version: %s with actual dinosaur version: %s", dinosaurRequest.DesiredCentralVersion, dinosaurRequest.ActualCentralVersion))
		}

		// no minor/ major version downgrades allowed for dinosaur version
		if vCompDinosaur > 0 {
			return errors.New(errors.ErrorValidation, fmt.Sprintf("Unable to downgrade dinosaur: %s version: %s to the following dinosaur version: %s", dinosaurRequest.ID, dinosaurRequest.ActualCentralVersion, dinosaurRequest.DesiredCentralVersion))
		}

		return k.Update(dinosaurRequest)
	} else {
		return errors.New(errors.ErrorUnauthenticated, "User not authenticated")
	}
}

func (k *dinosaurService) UpdateStatus(id string, status constants2.DinosaurStatus) (bool, *errors.ServiceError) {
	dbConn := k.connectionFactory.New()

	if dinosaur, err := k.GetById(id); err != nil {
		return true, errors.NewWithCause(errors.ErrorGeneral, err, "failed to update status")
	} else {
		// only allow to change the status to "deleting" if the cluster is already in "deprovision" status
		if dinosaur.Status == constants2.DinosaurRequestStatusDeprovision.String() && status != constants2.DinosaurRequestStatusDeleting {
			return false, errors.GeneralError("failed to update status: cluster is deprovisioning")
		}

		if dinosaur.Status == status.String() {
			// no update needed
			return false, errors.GeneralError("failed to update status: the cluster %s is already in %s state", id, status.String())
		}
	}

	update := &dbapi.CentralRequest{Status: status.String()}
	if status.String() == constants2.DinosaurRequestStatusDeprovision.String() {
		now := time.Now()
		update.DeletionTimestamp = &now
	}

	if err := dbConn.Model(&dbapi.CentralRequest{Meta: api.Meta{ID: id}}).Updates(update).Error; err != nil {
		return true, errors.NewWithCause(errors.ErrorGeneral, err, "Failed to update dinosaur status")
	}

	return true, nil
}

func (k *dinosaurService) ChangeDinosaurCNAMErecords(dinosaurRequest *dbapi.CentralRequest, action DinosaurRoutesAction) (*route53.ChangeResourceRecordSetsOutput, *errors.ServiceError) {
	routes, err := dinosaurRequest.GetRoutes()
	if routes == nil || err != nil {
		return nil, errors.NewWithCause(errors.ErrorGeneral, err, "failed to get routes")
	}

	domainRecordBatch := buildDinosaurClusterCNAMESRecordBatch(routes, string(action))

	// Create AWS client with the region of this Dinosaur Cluster
	awsConfig := aws.Config{
		AccessKeyID:     k.awsConfig.Route53AccessKey,
		SecretAccessKey: k.awsConfig.Route53SecretAccessKey,
	}
	awsClient, err := k.awsClientFactory.NewClient(awsConfig, dinosaurRequest.Region)
	if err != nil {
		return nil, errors.NewWithCause(errors.ErrorGeneral, err, "Unable to create aws client")
	}

	changeRecordsOutput, err := awsClient.ChangeResourceRecordSets(k.dinosaurConfig.DinosaurDomainName, domainRecordBatch)
	if err != nil {
		return nil, errors.NewWithCause(errors.ErrorGeneral, err, "Unable to create domain record sets")
	}

	return changeRecordsOutput, nil
}

func (k *dinosaurService) GetCNAMERecordStatus(dinosaurRequest *dbapi.CentralRequest) (*CNameRecordStatus, error) {
	awsConfig := aws.Config{
		AccessKeyID:     k.awsConfig.Route53AccessKey,
		SecretAccessKey: k.awsConfig.Route53SecretAccessKey,
	}
	awsClient, err := k.awsClientFactory.NewClient(awsConfig, dinosaurRequest.Region)
	if err != nil {
		return nil, errors.NewWithCause(errors.ErrorGeneral, err, "Unable to create aws client")
	}

	changeOutput, err := awsClient.GetChange(dinosaurRequest.RoutesCreationId)
	if err != nil {
		return nil, errors.NewWithCause(errors.ErrorGeneral, err, "Unable to CNAME record status")
	}

	return &CNameRecordStatus{
		Id:     changeOutput.ChangeInfo.Id,
		Status: changeOutput.ChangeInfo.Status,
	}, nil
}

type DinosaurStatusCount struct {
	Status constants2.DinosaurStatus
	Count  int
}

type DinosaurRegionCount struct {
	Region       string
	InstanceType string `gorm:"column:instance_type"`
	ClusterId    string `gorm:"column:cluster_id"`
	Count        int
}

func (k *dinosaurService) CountByRegionAndInstanceType() ([]DinosaurRegionCount, error) {
	dbConn := k.connectionFactory.New()
	var results []DinosaurRegionCount

	if err := dbConn.Model(&dbapi.CentralRequest{}).Select("region as Region, instance_type, cluster_id, count(1) as Count").Group("region,instance_type,cluster_id").Scan(&results).Error; err != nil {
		return nil, errors.NewWithCause(errors.ErrorGeneral, err, "Failed to count dinosaurs")
	}

	return results, nil
}

func (k *dinosaurService) CountByStatus(status []constants2.DinosaurStatus) ([]DinosaurStatusCount, error) {
	dbConn := k.connectionFactory.New()
	var results []DinosaurStatusCount
	if err := dbConn.Model(&dbapi.CentralRequest{}).Select("status as Status, count(1) as Count").Where("status in (?)", status).Group("status").Scan(&results).Error; err != nil {
		return nil, errors.NewWithCause(errors.ErrorGeneral, err, "Failed to count dinosaurs")
	}

	// if there is no count returned for a status from the above query because there is no dinosaurs in such a status,
	// we should return the count for these as well to avoid any confusion
	if len(status) > 0 {
		countersMap := map[constants2.DinosaurStatus]int{}
		for _, r := range results {
			countersMap[r.Status] = r.Count
		}
		for _, s := range status {
			if _, ok := countersMap[s]; !ok {
				results = append(results, DinosaurStatusCount{Status: s, Count: 0})
			}
		}
	}

	return results, nil
}

type DinosaurComponentVersions struct {
	ID                             string
	ClusterID                      string
	DesiredDinosaurOperatorVersion string
	ActualDinosaurOperatorVersion  string
	DinosaurOperatorUpgrading      bool
	DesiredDinosaurVersion         string
	ActualDinosaurVersion          string
	DinosaurUpgrading              bool
}

func (k *dinosaurService) ListComponentVersions() ([]DinosaurComponentVersions, error) {
	dbConn := k.connectionFactory.New()
	var results []DinosaurComponentVersions
	if err := dbConn.Model(&dbapi.CentralRequest{}).Select("id", "cluster_id", "desired_central_operator_version", "actual_central_operator_version", "central_operator_upgrading", "desired_central_version", "actual_central_version", "central_upgrading").Scan(&results).Error; err != nil {
		return nil, errors.NewWithCause(errors.ErrorGeneral, err, "failed to list component versions")
	}
	return results, nil
}

func (k *dinosaurService) ListDinosaursWithRoutesNotCreated() ([]*dbapi.CentralRequest, *errors.ServiceError) {
	dbConn := k.connectionFactory.New()
	var results []*dbapi.CentralRequest
	if err := dbConn.Where("routes IS NOT NULL").Where("routes_created = ?", "no").Find(&results).Error; err != nil {
		return nil, errors.NewWithCause(errors.ErrorGeneral, err, "failed to list dinosaur requests")
	}
	return results, nil
}

func BuildManagedDinosaurCR(dinosaurRequest *dbapi.CentralRequest, dinosaurConfig *config.DinosaurConfig, keycloakConfig *keycloak.KeycloakConfig) *manageddinosaur.ManagedDinosaur {
	managedDinosaurCR := &manageddinosaur.ManagedDinosaur{
		Id: dinosaurRequest.ID,
		TypeMeta: metav1.TypeMeta{
			Kind:       "ManagedCentral",
			APIVersion: "manageddinosaur.mas/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      dinosaurRequest.Name,
			Namespace: dinosaurRequest.Namespace,
			Annotations: map[string]string{
				"mas/id":          dinosaurRequest.ID,
				"mas/placementId": dinosaurRequest.PlacementId,
			},
		},
		Spec: manageddinosaur.ManagedDinosaurSpec{
			Endpoint: manageddinosaur.EndpointSpec{
				Host: dinosaurRequest.Host,
				Tls: &manageddinosaur.TlsSpec{
					Cert: dinosaurConfig.DinosaurTLSCert,
					Key:  dinosaurConfig.DinosaurTLSKey,
				},
			},
			Versions: manageddinosaur.VersionsSpec{
				Dinosaur:         dinosaurRequest.DesiredCentralVersion,
				DinosaurOperator: dinosaurRequest.DesiredCentralOperatorVersion,
			},
			Deleted: dinosaurRequest.Status == constants2.DinosaurRequestStatusDeprovision.String(),
			Owners: []string{
				dinosaurRequest.Owner,
			},
		},
		Status: manageddinosaur.ManagedDinosaurStatus{},
	}

	if dinosaurConfig.EnableDinosaurExternalCertificate {
		managedDinosaurCR.Spec.Endpoint.Tls = &manageddinosaur.TlsSpec{
			Cert: dinosaurConfig.DinosaurTLSCert,
			Key:  dinosaurConfig.DinosaurTLSKey,
		}
	}

	if dinosaurRequest.DeletionTimestamp != nil {
		managedDinosaurCR.DeletionTimestamp = &metav1.Time{Time: *dinosaurRequest.DeletionTimestamp}
	}

	return managedDinosaurCR
}

func buildDinosaurClusterCNAMESRecordBatch(routes []dbapi.DataPlaneCentralRoute, action string) *route53.ChangeBatch {
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
