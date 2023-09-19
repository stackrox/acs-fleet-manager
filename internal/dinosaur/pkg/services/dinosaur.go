package services

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/rhsso"
	"github.com/stackrox/acs-fleet-manager/pkg/client/iam"
	dynamicClientAPI "github.com/stackrox/acs-fleet-manager/pkg/client/redhatsso/api"
	"github.com/stackrox/acs-fleet-manager/pkg/client/redhatsso/dynamicclients"

	dinosaurConstants "github.com/stackrox/acs-fleet-manager/internal/dinosaur/constants"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/api/dbapi"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/config"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/dinosaurs/types"
	"github.com/stackrox/acs-fleet-manager/pkg/services"
	coreServices "github.com/stackrox/acs-fleet-manager/pkg/services/queryparser"

	"github.com/golang/glog"

	"github.com/aws/aws-sdk-go/service/route53"
	"github.com/stackrox/acs-fleet-manager/pkg/api"
	"github.com/stackrox/acs-fleet-manager/pkg/auth"
	"github.com/stackrox/acs-fleet-manager/pkg/client/aws"
	"github.com/stackrox/acs-fleet-manager/pkg/client/ocm"
	"github.com/stackrox/acs-fleet-manager/pkg/db"
	"github.com/stackrox/acs-fleet-manager/pkg/errors"
	"github.com/stackrox/acs-fleet-manager/pkg/features"
	"github.com/stackrox/acs-fleet-manager/pkg/logger"
	"github.com/stackrox/acs-fleet-manager/pkg/metrics"
)

var (
	dinosaurDeletionStatuses = []string{
		dinosaurConstants.CentralRequestStatusDeleting.String(),
		dinosaurConstants.CentralRequestStatusDeprovision.String(),
	}

	dinosaurManagedCRStatuses = []string{
		dinosaurConstants.CentralRequestStatusProvisioning.String(),
		dinosaurConstants.CentralRequestStatusDeprovision.String(),
		dinosaurConstants.CentralRequestStatusReady.String(),
		dinosaurConstants.CentralRequestStatusFailed.String(),
	}
)

// DinosaurRoutesAction ...
type DinosaurRoutesAction string

// DinosaurRoutesActionCreate ...
const DinosaurRoutesActionCreate DinosaurRoutesAction = "CREATE"

// DinosaurRoutesActionDelete ...
const DinosaurRoutesActionDelete DinosaurRoutesAction = "DELETE"

// CNameRecordStatus ...
type CNameRecordStatus struct {
	ID     *string
	Status *string
}

// DinosaurService ...
//
//go:generate moq -out dinosaurservice_moq.go . DinosaurService
type DinosaurService interface {
	// HasAvailableCapacityInRegion checks if there is capacity in the clusters for a given region
	HasAvailableCapacityInRegion(dinosaurRequest *dbapi.CentralRequest) (bool, *errors.ServiceError)
	// AcceptCentralRequest transitions CentralRequest to 'Preparing'.
	AcceptCentralRequest(centralRequest *dbapi.CentralRequest) *errors.ServiceError
	// PrepareDinosaurRequest transitions CentralRequest to 'Provisioning'.
	PrepareDinosaurRequest(dinosaurRequest *dbapi.CentralRequest) *errors.ServiceError
	// Get method will retrieve the dinosaurRequest instance that the give ctx has access to from the database.
	// This should be used when you want to make sure the result is filtered based on the request context.
	Get(ctx context.Context, id string) (*dbapi.CentralRequest, *errors.ServiceError)
	// GetByID method will retrieve the DinosaurRequest instance from the database without checking any permissions.
	// You should only use this if you are sure permission check is not required.
	GetByID(id string) (*dbapi.CentralRequest, *errors.ServiceError)
	// Delete cleans up all dependencies for a Dinosaur request and soft deletes the Dinosaur Request record from the database.
	// The Dinosaur Request in the database will be updated with a deleted_at timestamp.
	Delete(centralRequest *dbapi.CentralRequest, force bool) *errors.ServiceError
	List(ctx context.Context, listArgs *services.ListArguments) (dbapi.CentralList, *api.PagingMeta, *errors.ServiceError)
	RegisterDinosaurJob(dinosaurRequest *dbapi.CentralRequest) *errors.ServiceError
	ListByStatus(status ...dinosaurConstants.CentralStatus) ([]*dbapi.CentralRequest, *errors.ServiceError)
	// UpdateStatus change the status of the Dinosaur cluster
	// The returned boolean is to be used to know if the update has been tried or not. An update is not tried if the
	// original status is 'deprovision' (cluster in deprovision state can't be change state) or if the final status is the
	// same as the original status. The error will contain any error encountered when attempting to update or the reason
	// why no attempt has been done
	UpdateStatus(id string, status dinosaurConstants.CentralStatus) (bool, *errors.ServiceError)
	Update(dinosaurRequest *dbapi.CentralRequest) *errors.ServiceError
	// Updates() updates the given fields of a dinosaur. This takes in a map so that even zero-fields can be updated.
	// Use this only when you want to update the multiple columns that may contain zero-fields, otherwise use the `DinosaurService.Update()` method.
	// See https://gorm.io/docs/update.html#Updates-multiple-columns for more info
	Updates(dinosaurRequest *dbapi.CentralRequest, values map[string]interface{}) *errors.ServiceError
	ChangeDinosaurCNAMErecords(dinosaurRequest *dbapi.CentralRequest, action DinosaurRoutesAction) (*route53.ChangeResourceRecordSetsOutput, *errors.ServiceError)
	GetCNAMERecordStatus(dinosaurRequest *dbapi.CentralRequest) (*CNameRecordStatus, error)
	DetectInstanceType(dinosaurRequest *dbapi.CentralRequest) types.DinosaurInstanceType
	RegisterDinosaurDeprovisionJob(ctx context.Context, id string) *errors.ServiceError
	// DeprovisionDinosaurForUsers registers all dinosaurs for deprovisioning given the list of owners
	DeprovisionDinosaurForUsers(users []string) *errors.ServiceError
	DeprovisionExpiredDinosaurs(dinosaurAgeInHours int) *errors.ServiceError
	CountByStatus(status []dinosaurConstants.CentralStatus) ([]DinosaurStatusCount, error)
	CountByRegionAndInstanceType() ([]DinosaurRegionCount, error)
	ListDinosaursWithRoutesNotCreated() ([]*dbapi.CentralRequest, *errors.ServiceError)
	ListCentralsWithoutAuthConfig() ([]*dbapi.CentralRequest, *errors.ServiceError)
	VerifyAndUpdateDinosaurAdmin(ctx context.Context, dinosaurRequest *dbapi.CentralRequest) *errors.ServiceError
	Restore(ctx context.Context, id string) *errors.ServiceError
	RotateCentralRHSSOClient(ctx context.Context, centralRequest *dbapi.CentralRequest) *errors.ServiceError
}

var _ DinosaurService = &dinosaurService{}

type dinosaurService struct {
	connectionFactory            *db.ConnectionFactory
	clusterService               ClusterService
	dinosaurConfig               *config.CentralConfig
	awsConfig                    *config.AWSConfig
	quotaServiceFactory          QuotaServiceFactory
	mu                           sync.Mutex
	awsClientFactory             aws.ClientFactory
	dataplaneClusterConfig       *config.DataplaneClusterConfig
	clusterPlacementStrategy     ClusterPlacementStrategy
	amsClient                    ocm.AMSClient
	centralDefaultVersionService CentralDefaultVersionService
	iamConfig                    *iam.IAMConfig
	rhSSODynamicClientsAPI       *dynamicClientAPI.AcsTenantsApiService
}

// NewDinosaurService ...
func NewDinosaurService(connectionFactory *db.ConnectionFactory, clusterService ClusterService,
	iamConfig *iam.IAMConfig, dinosaurConfig *config.CentralConfig, dataplaneClusterConfig *config.DataplaneClusterConfig, awsConfig *config.AWSConfig,
	quotaServiceFactory QuotaServiceFactory, awsClientFactory aws.ClientFactory,
	clusterPlacementStrategy ClusterPlacementStrategy, amsClient ocm.AMSClient, centralDefaultVersionService CentralDefaultVersionService) DinosaurService {
	return &dinosaurService{
		connectionFactory:            connectionFactory,
		clusterService:               clusterService,
		iamConfig:                    iamConfig,
		dinosaurConfig:               dinosaurConfig,
		awsConfig:                    awsConfig,
		quotaServiceFactory:          quotaServiceFactory,
		awsClientFactory:             awsClientFactory,
		dataplaneClusterConfig:       dataplaneClusterConfig,
		clusterPlacementStrategy:     clusterPlacementStrategy,
		amsClient:                    amsClient,
		centralDefaultVersionService: centralDefaultVersionService,
		rhSSODynamicClientsAPI:       dynamicclients.NewDynamicClientsAPI(iamConfig.RedhatSSORealm),
	}
}

func (k *dinosaurService) RotateCentralRHSSOClient(ctx context.Context, centralRequest *dbapi.CentralRequest) *errors.ServiceError {
	realmConfig := k.iamConfig.RedhatSSORealm
	if k.dinosaurConfig.HasStaticAuth() {
		return errors.New(errors.ErrorDynamicClientsNotUsed, "RHSSO is configured via static configuration")
	}
	if !realmConfig.IsConfigured() {
		return errors.New(errors.ErrorDynamicClientsNotUsed, "RHSSO dynamic client configuration is not present")
	}

	previousAuthConfig := centralRequest.AuthConfig
	if err := rhsso.AugmentWithDynamicAuthConfig(ctx, centralRequest, k.iamConfig.RedhatSSORealm, k.rhSSODynamicClientsAPI); err != nil {
		return errors.NewWithCause(errors.ErrorClientRotationFailed, err, "failed to augment auth config")
	}
	if err := k.Update(centralRequest); err != nil {
		glog.Errorf("Rotating RHSSO client failed: created new RHSSO dynamic client, but failed to update central record, client ID is %s", centralRequest.AuthConfig.ClientID)
		return errors.NewWithCause(errors.ErrorClientRotationFailed, err, "failed to update database record")
	}
	if _, err := k.rhSSODynamicClientsAPI.DeleteAcsClient(ctx, previousAuthConfig.ClientID); err != nil {
		glog.Errorf("Rotating RHSSO client failed: failed to delete RHSSO dynamic client, client ID is %s", centralRequest.AuthConfig.ClientID)
		return errors.NewWithCause(errors.ErrorClientRotationFailed, err, "failed to delete previous RHSSO dynamic client")
	}
	return nil
}

// HasAvailableCapacityInRegion ...
func (k *dinosaurService) HasAvailableCapacityInRegion(dinosaurRequest *dbapi.CentralRequest) (bool, *errors.ServiceError) {
	regionCapacity := int64(k.dataplaneClusterConfig.ClusterConfig.GetCapacityForRegion(dinosaurRequest.Region))
	if regionCapacity <= 0 {
		return false, nil
	}

	dbConn := k.connectionFactory.New()
	var count int64
	if err := dbConn.Model(&dbapi.CentralRequest{}).Where("region = ?", dinosaurRequest.Region).Count(&count).Error; err != nil {
		return false, errors.NewWithCause(errors.ErrorGeneral, err, "failed to count central request")
	}

	glog.Infof("%d of %d central clusters currently instantiated in region %v", count, regionCapacity, dinosaurRequest.Region)
	return count < regionCapacity, nil
}

// DetectInstanceType - returns standard instance type if quota is available. Otherwise falls back to eval instance type.
func (k *dinosaurService) DetectInstanceType(dinosaurRequest *dbapi.CentralRequest) types.DinosaurInstanceType {
	quotaType := api.QuotaType(k.dinosaurConfig.Quota.Type)
	quotaService, factoryErr := k.quotaServiceFactory.GetQuotaService(quotaType)
	if factoryErr != nil {
		glog.Error(errors.NewWithCause(errors.ErrorGeneral, factoryErr, "unable to get quota service"))
		return types.EVAL
	}

	hasQuota, err := quotaService.CheckIfQuotaIsDefinedForInstanceType(dinosaurRequest, types.STANDARD)
	if err != nil {
		glog.Error(errors.NewWithCause(errors.ErrorGeneral, err, "unable to check quota"))
		return types.EVAL
	}
	if hasQuota {
		glog.Infof("Quota detected for central request %s with quota type %s. Granting instance type %s.", dinosaurRequest.ID, quotaType, types.STANDARD)
		return types.STANDARD
	}

	glog.Infof("No quota detected for central request %s with quota type %s. Granting instance type %s.", dinosaurRequest.ID, quotaType, types.EVAL)
	return types.EVAL
}

// reserveQuota - reserves quota for the given dinosaur request. If a RHACS quota has been assigned, it will try to reserve RHACS quota, otherwise it will try with RHACSTrial
func (k *dinosaurService) reserveQuota(dinosaurRequest *dbapi.CentralRequest) (subscriptionID string, err *errors.ServiceError) {
	if dinosaurRequest.InstanceType == types.EVAL.String() {
		if !k.dinosaurConfig.Quota.AllowEvaluatorInstance {
			return "", errors.NewWithCause(errors.ErrorForbidden, err, "central eval instances are not allowed")
		}

		// Only one EVAL instance is admitted. Let's check if the user already owns one
		dbConn := k.connectionFactory.New()
		var count int64
		if err := dbConn.Model(&dbapi.CentralRequest{}).
			Where("instance_type = ?", types.EVAL).
			Where("owner = ?", dinosaurRequest.Owner).
			Where("organisation_id = ?", dinosaurRequest.OrganisationID).
			Count(&count).
			Error; err != nil {
			return "", errors.NewWithCause(errors.ErrorGeneral, err, "failed to count central eval instances")
		}

		if count > 0 {
			return "", errors.TooManyDinosaurInstancesReached("only one eval instance is allowed; increase your account quota")
		}
	}

	quotaService, factoryErr := k.quotaServiceFactory.GetQuotaService(api.QuotaType(k.dinosaurConfig.Quota.Type))
	if factoryErr != nil {
		return "", errors.NewWithCause(errors.ErrorGeneral, factoryErr, "unable to check quota")
	}
	subscriptionID, err = quotaService.ReserveQuota(dinosaurRequest, types.DinosaurInstanceType(dinosaurRequest.InstanceType))
	return subscriptionID, err
}

// RegisterDinosaurJob registers a new job in the dinosaur table
func (k *dinosaurService) RegisterDinosaurJob(dinosaurRequest *dbapi.CentralRequest) *errors.ServiceError {
	k.mu.Lock()
	defer k.mu.Unlock()
	// we need to pre-populate the ID to be able to reserve the quota
	dinosaurRequest.ID = api.NewID()

	if hasCapacity, err := k.HasAvailableCapacityInRegion(dinosaurRequest); err != nil {
		return errors.NewWithCause(errors.ErrorGeneral, err, "failed to create central request")
	} else if !hasCapacity {
		errorMsg := fmt.Sprintf("Cluster capacity(%d) exhausted in %s region", int64(k.dataplaneClusterConfig.ClusterConfig.GetCapacityForRegion(dinosaurRequest.Region)), dinosaurRequest.Region)
		logger.Logger.Warningf(errorMsg)
		return errors.TooManyDinosaurInstancesReached(errorMsg)
	}

	instanceType := k.DetectInstanceType(dinosaurRequest)

	dinosaurRequest.InstanceType = instanceType.String()

	cluster, e := k.clusterPlacementStrategy.FindCluster(dinosaurRequest)
	if e != nil || cluster == nil {
		msg := fmt.Sprintf("No available cluster found for '%s' central instance in region: '%s'", dinosaurRequest.InstanceType, dinosaurRequest.Region)
		logger.Logger.Errorf(msg)
		return errors.TooManyDinosaurInstancesReached(fmt.Sprintf("Region %s cannot accept instance type: %s at this moment", dinosaurRequest.Region, dinosaurRequest.InstanceType))
	}
	dinosaurRequest.ClusterID = cluster.ClusterID
	subscriptionID, err := k.reserveQuota(dinosaurRequest)
	if err != nil {
		return err
	}

	if features.TargetedOperatorUpgrades.Enabled() {
		defaultVersion, serviceErr := k.centralDefaultVersionService.GetDefaultVersion()
		if serviceErr != nil {
			return err
		}
		dinosaurRequest.OperatorImage = defaultVersion
	}

	dbConn := k.connectionFactory.New()
	dinosaurRequest.Status = dinosaurConstants.CentralRequestStatusAccepted.String()
	dinosaurRequest.SubscriptionID = subscriptionID
	glog.Infof("Central request %s has been assigned the subscription %s.", dinosaurRequest.ID, subscriptionID)
	// Persist the QuotaType to be able to dynamically pick the right Quota service implementation even on restarts.
	// A typical usecase is when a dinosaur A is created, at the time of creation the quota-type was ams. At some point in the future
	// the API is restarted this time changing the --quota-type flag to quota-management-list, when dinosaur A is deleted at this point,
	// we want to use the correct quota to perform the deletion.
	dinosaurRequest.QuotaType = k.dinosaurConfig.Quota.Type
	if err := dbConn.Create(dinosaurRequest).Error; err != nil {
		return errors.NewWithCause(errors.ErrorGeneral, err, "failed to create central request") // hide the db error to http caller
	}
	metrics.UpdateCentralRequestsStatusSinceCreatedMetric(dinosaurConstants.CentralRequestStatusAccepted, dinosaurRequest.ID, dinosaurRequest.ClusterID, time.Since(dinosaurRequest.CreatedAt))
	return nil
}

// AcceptCentralRequest sets any information about Central that does not
// require blocking operations (deducing namespace or instance hostname). Upon
// success, CentralRequest is transitioned to 'Preparing' status and might not
// be fully prepared yet.
func (k *dinosaurService) AcceptCentralRequest(centralRequest *dbapi.CentralRequest) *errors.ServiceError {
	// Set namespace.
	namespace, formatErr := FormatNamespace(centralRequest.ID)
	if formatErr != nil {
		return errors.NewWithCause(errors.ErrorGeneral, formatErr, "invalid id format")
	}
	centralRequest.Namespace = namespace

	// Set host.
	if k.dinosaurConfig.EnableCentralExternalCertificate {
		// If we enable DinosaurTLS, the host should use the external domain name rather than the cluster domain
		centralRequest.Host = k.dinosaurConfig.CentralDomainName
	} else {
		clusterDNS, err := k.clusterService.GetClusterDNS(centralRequest.ClusterID)
		if err != nil {
			return errors.NewWithCause(errors.ErrorGeneral, err, "error retrieving cluster DNS")
		}
		centralRequest.Host = clusterDNS
	}

	// Update the fields of the CentralRequest record in the database.
	updatedDinosaurRequest := &dbapi.CentralRequest{
		Meta: api.Meta{
			ID: centralRequest.ID,
		},
		Host:        centralRequest.Host,
		PlacementID: api.NewID(),
		Status:      dinosaurConstants.CentralRequestStatusPreparing.String(),
		Namespace:   centralRequest.Namespace,
	}
	if err := k.Update(updatedDinosaurRequest); err != nil {
		return errors.NewWithCause(errors.ErrorGeneral, err, "failed to update central request")
	}

	return nil
}

// PrepareDinosaurRequest ensures that any required information (e.g.,
// CentralRequest's host, RHSSO auth config, etc) has been set. Upon success,
// the request is transitioned to 'Provisioning' status.
func (k *dinosaurService) PrepareDinosaurRequest(dinosaurRequest *dbapi.CentralRequest) *errors.ServiceError {
	// Check if the request is ready to be transitioned to provisioning.

	// Check IdP config is ready.
	//
	// TODO(alexr): Shall this go into "preparing_dinosaurs_mgr.go"? Ideally,
	//     all CentralRequest updating logic is in one place, either in this
	//     service or workers.
	if dinosaurRequest.AuthConfig.ClientID == "" {
		// We can't provision this request, skip
		return nil
	}

	// Obtain organisation name from AMS to store in central request.
	org, err := k.amsClient.GetOrganisationFromExternalID(dinosaurRequest.OrganisationID)
	if err != nil {
		return errors.OrganisationNotFound(dinosaurRequest.OrganisationID, err)
	}
	orgName := org.Name()
	if orgName == "" {
		return errors.OrganisationNameInvalid(dinosaurRequest.OrganisationID, orgName)
	}

	// Update the fields of the CentralRequest record in the database.
	updatedCentralRequest := &dbapi.CentralRequest{
		Meta: api.Meta{
			ID: dinosaurRequest.ID,
		},
		OrganisationName: orgName,
		Status:           dinosaurConstants.CentralRequestStatusProvisioning.String(),
	}
	if err := k.Update(updatedCentralRequest); err != nil {
		return errors.NewWithCause(errors.ErrorGeneral, err, "failed to update central request")
	}

	return nil
}

// ListByStatus ...
func (k *dinosaurService) ListByStatus(status ...dinosaurConstants.CentralStatus) ([]*dbapi.CentralRequest, *errors.ServiceError) {
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

// Get ...
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

		orgID, _ := claims.GetOrgID()
		filterByOrganisationID := auth.GetFilterByOrganisationFromContext(ctx)

		// filter by organisationId if a user is part of an organisation and is not allowed as a service account
		if filterByOrganisationID {
			dbConn = dbConn.Where("organisation_id = ?", orgID)
		} else {
			dbConn = dbConn.Where("owner = ?", user)
		}
	}

	var dinosaurRequest dbapi.CentralRequest
	if err := dbConn.First(&dinosaurRequest).Error; err != nil {
		resourceTypeStr := "CentralResource"
		if user != "" {
			resourceTypeStr = fmt.Sprintf("%s for user %s", resourceTypeStr, user)
		}
		return nil, services.HandleGetError(resourceTypeStr, "id", id, err)
	}
	return &dinosaurRequest, nil
}

// GetByID ...
func (k *dinosaurService) GetByID(id string) (*dbapi.CentralRequest, *errors.ServiceError) {
	if id == "" {
		return nil, errors.Validation("id is undefined")
	}

	dbConn := k.connectionFactory.New()
	var dinosaurRequest dbapi.CentralRequest
	if err := dbConn.Where("id = ?", id).First(&dinosaurRequest).Error; err != nil {
		return nil, services.HandleGetError("CentralResource", "id", id, err)
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
		orgID, _ := claims.GetOrgID()
		dbConn = dbConn.Where("id = ?", id).Where("organisation_id = ?", orgID)
	} else {
		user, _ := claims.GetUsername()
		dbConn = dbConn.Where("id = ?", id).Where("owner = ? ", user)
	}

	var dinosaurRequest dbapi.CentralRequest
	if err := dbConn.First(&dinosaurRequest).Error; err != nil {
		return services.HandleGetError("CentralResource", "id", id, err)
	}
	metrics.IncreaseCentralTotalOperationsCountMetric(dinosaurConstants.CentralOperationDeprovision)

	deprovisionStatus := dinosaurConstants.CentralRequestStatusDeprovision

	if executed, err := k.UpdateStatus(id, deprovisionStatus); executed {
		if err != nil {
			return services.HandleGetError("CentralResource", "id", id, err)
		}
		metrics.IncreaseCentralSuccessOperationsCountMetric(dinosaurConstants.CentralOperationDeprovision)
		metrics.UpdateCentralRequestsStatusSinceCreatedMetric(deprovisionStatus, dinosaurRequest.ID, dinosaurRequest.ClusterID, time.Since(dinosaurRequest.CreatedAt))
	}

	return nil
}

// DeprovisionDinosaurForUsers registers all dinosaurs for deprovisioning given the list of owners
func (k *dinosaurService) DeprovisionDinosaurForUsers(users []string) *errors.ServiceError {
	now := time.Now()
	dbConn := k.connectionFactory.New().
		Model(&dbapi.CentralRequest{}).
		Where("owner IN (?)", users).
		Where("status NOT IN (?)", dinosaurDeletionStatuses).
		Updates(map[string]interface{}{
			"status":             dinosaurConstants.CentralRequestStatusDeprovision,
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
			metrics.IncreaseCentralTotalOperationsCountMetric(dinosaurConstants.CentralOperationDeprovision)
			metrics.IncreaseCentralSuccessOperationsCountMetric(dinosaurConstants.CentralOperationDeprovision)
		}
	}

	return nil
}

// DeprovisionExpiredDinosaurs cleaning up expired dinosaurs
func (k *dinosaurService) DeprovisionExpiredDinosaurs(dinosaurAgeInHours int) *errors.ServiceError {
	now := time.Now()
	dbConn := k.connectionFactory.New().
		Model(&dbapi.CentralRequest{}).
		Where("instance_type = ?", types.EVAL.String()).
		Where("created_at  <=  ?", now.Add(-1*time.Duration(dinosaurAgeInHours)*time.Hour)).
		Where("status NOT IN (?)", dinosaurDeletionStatuses)

	db := dbConn.Updates(map[string]interface{}{
		"status":             dinosaurConstants.CentralRequestStatusDeprovision,
		"deletion_timestamp": now,
	})
	err := db.Error
	if err != nil {
		return errors.NewWithCause(errors.ErrorGeneral, err, "unable to deprovision expired centrals")
	}

	if db.RowsAffected >= 1 {
		glog.Infof("%v central_request's lifespans are over %d hours and have had their status updated to deprovisioning", db.RowsAffected, dinosaurAgeInHours)
		var counter int64
		for ; counter < db.RowsAffected; counter++ {
			metrics.IncreaseCentralTotalOperationsCountMetric(dinosaurConstants.CentralOperationDeprovision)
			metrics.IncreaseCentralSuccessOperationsCountMetric(dinosaurConstants.CentralOperationDeprovision)
		}
	}

	return nil
}

// Delete a CentralRequest from the database.
// The implementation uses soft-deletion (via GORM).
// If the force flag is true, then any errors prior to the final deletion of the CentralRequest will be logged as warnings
// but do not interrupt the deletion flow.
func (k *dinosaurService) Delete(centralRequest *dbapi.CentralRequest, force bool) *errors.ServiceError {
	dbConn := k.connectionFactory.New()

	// if the we don't have the clusterID we can only delete the row from the database
	if centralRequest.ClusterID != "" {
		routes, err := centralRequest.GetRoutes()
		if err != nil {
			return errors.NewWithCause(errors.ErrorGeneral, err, "failed to get routes")
		}
		// Only delete the routes when they are set
		if routes != nil && k.dinosaurConfig.EnableCentralExternalCertificate {
			_, err := k.ChangeDinosaurCNAMErecords(centralRequest, DinosaurRoutesActionDelete)
			if err != nil {
				if force {
					glog.Warningf("Failed to delete CNAME records for Central tenant %q: %v", centralRequest.ID, err)
					glog.Warning("Continuing with deletion of Central tenant because force-deletion is specified")
				} else {
					return err
				}
			}
			glog.Infof("Successfully deleted CNAME records for Central tenant %q", centralRequest.ID)
		}
	}

	// soft delete the dinosaur request
	if err := dbConn.Delete(centralRequest).Error; err != nil {
		return errors.NewWithCause(errors.ErrorGeneral, err, "unable to delete central request with id %s", centralRequest.ID)
	}

	glog.Infof("Successfully deleted Central tenant %q in the database.", centralRequest.ID)
	if force {
		glog.Infof("Make sure any other resources belonging to the Central tenant %q are manually deleted.", centralRequest.ID)
	}
	metrics.IncreaseCentralTotalOperationsCountMetric(dinosaurConstants.CentralOperationDelete)
	metrics.IncreaseCentralSuccessOperationsCountMetric(dinosaurConstants.CentralOperationDelete)

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

		orgID, _ := claims.GetOrgID()
		filterByOrganisationID := auth.GetFilterByOrganisationFromContext(ctx)

		// filter by organisationId if a user is part of an organisation and is not allowed as a service account
		if filterByOrganisationID {
			// filter dinosaur requests by organisation_id since the user is allowed to see all dinosaur requests of my id
			dbConn = dbConn.Where("organisation_id = ?", orgID)
		} else {
			// filter dinosaur requests by owner as we are dealing with service accounts which may not have an org id
			dbConn = dbConn.Where("owner = ?", user)
		}
	}

	// Apply search query
	if len(listArgs.Search) > 0 {
		searchDbQuery, err := coreServices.NewQueryParser().Parse(listArgs.Search)
		if err != nil {
			return dinosaurRequestList, pagingMeta, errors.NewWithCause(errors.ErrorFailedToParseSearch, err, "Unable to list central requests: %s", err.Error())
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
		return dinosaurRequestList, pagingMeta, errors.NewWithCause(errors.ErrorGeneral, err, "Unable to list central requests")
	}

	return dinosaurRequestList, pagingMeta, nil
}

// Update ...
func (k *dinosaurService) Update(dinosaurRequest *dbapi.CentralRequest) *errors.ServiceError {
	dbConn := k.connectionFactory.New().
		Model(dinosaurRequest).
		Where("status not IN (?)", dinosaurDeletionStatuses) // ignore updates of dinosaur under deletion

	if err := dbConn.Updates(dinosaurRequest).Error; err != nil {
		return errors.NewWithCause(errors.ErrorGeneral, err, "Failed to update central")
	}

	return nil
}

// Updates ...
func (k *dinosaurService) Updates(dinosaurRequest *dbapi.CentralRequest, fields map[string]interface{}) *errors.ServiceError {
	dbConn := k.connectionFactory.New().
		Model(dinosaurRequest).
		Where("status not IN (?)", dinosaurDeletionStatuses) // ignore updates of dinosaur under deletion

	if err := dbConn.Updates(fields).Error; err != nil {
		return errors.NewWithCause(errors.ErrorGeneral, err, "Failed to update central")
	}

	return nil
}

// VerifyAndUpdateDinosaurAdmin ...
func (k *dinosaurService) VerifyAndUpdateDinosaurAdmin(ctx context.Context, dinosaurRequest *dbapi.CentralRequest) *errors.ServiceError {
	if !auth.GetIsAdminFromContext(ctx) {
		return errors.New(errors.ErrorUnauthenticated, "User not authenticated")
	}

	cluster, svcErr := k.clusterService.FindClusterByID(dinosaurRequest.ClusterID)
	if svcErr != nil {
		return errors.NewWithCause(errors.ErrorGeneral, svcErr, "Unable to find cluster associated with central request: %s", dinosaurRequest.ID)
	}
	if cluster == nil {
		return errors.New(errors.ErrorValidation, fmt.Sprintf("Unable to get cluster for central %s", dinosaurRequest.ID))
	}

	return k.Update(dinosaurRequest)
}

// UpdateStatus ...
func (k *dinosaurService) UpdateStatus(id string, status dinosaurConstants.CentralStatus) (bool, *errors.ServiceError) {
	dbConn := k.connectionFactory.New()

	dinosaur, err := k.GetByID(id)
	if err != nil {
		return true, errors.NewWithCause(errors.ErrorGeneral, err, "failed to update status")
	}
	// only allow to change the status to "deleting" if the cluster is already in "deprovision" status
	if dinosaur.Status == dinosaurConstants.CentralRequestStatusDeprovision.String() && status != dinosaurConstants.CentralRequestStatusDeleting {
		return false, errors.GeneralError("failed to update status: cluster is deprovisioning")
	}

	if dinosaur.Status == status.String() {
		// no update needed
		return false, errors.GeneralError("failed to update status: the cluster %s is already in %s state", id, status.String())
	}

	update := &dbapi.CentralRequest{Status: status.String()}
	if status.String() == dinosaurConstants.CentralRequestStatusDeprovision.String() {
		now := time.Now()
		update.DeletionTimestamp = &now
	}

	if err := dbConn.Model(&dbapi.CentralRequest{Meta: api.Meta{ID: id}}).Updates(update).Error; err != nil {
		return true, errors.NewWithCause(errors.ErrorGeneral, err, "Failed to update central status")
	}

	return true, nil
}

// ChangeDinosaurCNAMErecords ...
func (k *dinosaurService) ChangeDinosaurCNAMErecords(dinosaurRequest *dbapi.CentralRequest, action DinosaurRoutesAction) (*route53.ChangeResourceRecordSetsOutput, *errors.ServiceError) {
	routes, err := dinosaurRequest.GetRoutes()
	if routes == nil || err != nil {
		return nil, errors.NewWithCause(errors.ErrorGeneral, err, "failed to get routes")
	}

	domainRecordBatch := buildDinosaurClusterCNAMESRecordBatch(routes, string(action))

	// Create AWS client with the region of this Dinosaur Cluster
	awsConfig := aws.Config{
		AccessKeyID:     k.awsConfig.Route53AccessKey,
		SecretAccessKey: k.awsConfig.Route53SecretAccessKey, // pragma: allowlist secret
	}
	awsClient, err := k.awsClientFactory.NewClient(awsConfig, dinosaurRequest.Region)
	if err != nil {
		return nil, errors.NewWithCause(errors.ErrorGeneral, err, "Unable to create aws client")
	}

	changeRecordsOutput, err := awsClient.ChangeResourceRecordSets(k.dinosaurConfig.CentralDomainName, domainRecordBatch)
	if err != nil {
		return nil, errors.NewWithCause(errors.ErrorGeneral, err, "Unable to create domain record sets")
	}

	return changeRecordsOutput, nil
}

// GetCNAMERecordStatus ...
func (k *dinosaurService) GetCNAMERecordStatus(dinosaurRequest *dbapi.CentralRequest) (*CNameRecordStatus, error) {
	awsConfig := aws.Config{
		AccessKeyID:     k.awsConfig.Route53AccessKey,
		SecretAccessKey: k.awsConfig.Route53SecretAccessKey, // pragma: allowlist secret
	}
	awsClient, err := k.awsClientFactory.NewClient(awsConfig, dinosaurRequest.Region)
	if err != nil {
		return nil, errors.NewWithCause(errors.ErrorGeneral, err, "Unable to create aws client")
	}

	changeOutput, err := awsClient.GetChange(dinosaurRequest.RoutesCreationID)
	if err != nil {
		return nil, errors.NewWithCause(errors.ErrorGeneral, err, "Unable to CNAME record status")
	}

	return &CNameRecordStatus{
		ID:     changeOutput.ChangeInfo.Id,
		Status: changeOutput.ChangeInfo.Status,
	}, nil
}

func (k *dinosaurService) Restore(ctx context.Context, id string) *errors.ServiceError {
	dbConn := k.connectionFactory.New()
	var centralRequest dbapi.CentralRequest
	if err := dbConn.Unscoped().Where("id = ?", id).First(&centralRequest).Error; err != nil {
		return services.HandleGetError("CentralRequest", "id", id, err)
	}

	if !centralRequest.DeletedAt.Valid {
		return errors.BadRequest("CentralRequests not marked as deleted.")
	}

	timeSinceDeletion := time.Since(centralRequest.DeletedAt.Time)
	if timeSinceDeletion.Hours()/24 > float64(k.dinosaurConfig.CentralRetentionPeriodDays) {
		return errors.BadRequest("CentralRequests retention period already expired")
	}

	// Reset all values up to provisioning
	columnsToReset := []string{
		"Routes",
		"Status",
		"RoutesCreated",
		"RoutesCreationID",
		"DeletedAt",
		"DeletionTimestamp",
		"ClientID",
		"ClientOrigin",
		"ClientSecret",
		"CreatedAt",
	}

	// use a new central request, so that unset field for columnsToReset will automatically be set to the zero value
	// this Update only changes columns listed in columnsToReset
	resetRequest := &dbapi.CentralRequest{}
	resetRequest.Status = dinosaurConstants.CentralRequestStatusPreparing.String()
	resetRequest.CreatedAt = time.Now()

	if err := dbConn.Unscoped().Model(resetRequest).Select(columnsToReset).Updates(resetRequest).Error; err != nil {
		return errors.NewWithCause(errors.ErrorGeneral, err, "Unable to reset CentralRequest status")
	}

	return nil
}

// DinosaurStatusCount ...
type DinosaurStatusCount struct {
	Status dinosaurConstants.CentralStatus
	Count  int
}

// DinosaurRegionCount ...
type DinosaurRegionCount struct {
	Region       string
	InstanceType string `gorm:"column:instance_type"`
	ClusterID    string `gorm:"column:cluster_id"`
	Count        int
}

// CountByRegionAndInstanceType ...
func (k *dinosaurService) CountByRegionAndInstanceType() ([]DinosaurRegionCount, error) {
	dbConn := k.connectionFactory.New()
	var results []DinosaurRegionCount

	if err := dbConn.Model(&dbapi.CentralRequest{}).Select("region as Region, instance_type, cluster_id, count(1) as Count").Group("region,instance_type,cluster_id").Scan(&results).Error; err != nil {
		return nil, errors.NewWithCause(errors.ErrorGeneral, err, "Failed to count centrals")
	}

	return results, nil
}

// CountByStatus ...
func (k *dinosaurService) CountByStatus(status []dinosaurConstants.CentralStatus) ([]DinosaurStatusCount, error) {
	dbConn := k.connectionFactory.New()
	var results []DinosaurStatusCount
	if err := dbConn.Model(&dbapi.CentralRequest{}).Select("status as Status, count(1) as Count").Where("status in (?)", status).Group("status").Scan(&results).Error; err != nil {
		return nil, errors.NewWithCause(errors.ErrorGeneral, err, "Failed to count centrals")
	}

	// if there is no count returned for a status from the above query because there is no dinosaurs in such a status,
	// we should return the count for these as well to avoid any confusion
	if len(status) > 0 {
		countersMap := map[dinosaurConstants.CentralStatus]int{}
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

// ListDinosaursWithRoutesNotCreated ...
func (k *dinosaurService) ListDinosaursWithRoutesNotCreated() ([]*dbapi.CentralRequest, *errors.ServiceError) {
	dbConn := k.connectionFactory.New()
	var results []*dbapi.CentralRequest
	if err := dbConn.Where("routes IS NOT NULL").Where("routes_created = ?", "no").Find(&results).Error; err != nil {
		return nil, errors.NewWithCause(errors.ErrorGeneral, err, "failed to list central requests")
	}
	return results, nil
}

// ListCentralsWithoutAuthConfig returns all _relevant_ central requests with
// no auth config. For central requests without host set, we cannot compute
// redirect_uri and hence cannot set up auth config.
func (k *dinosaurService) ListCentralsWithoutAuthConfig() ([]*dbapi.CentralRequest, *errors.ServiceError) {
	dbQuery := k.connectionFactory.New().
		Where("client_id = ''").
		Where("host != ''")

	var results []*dbapi.CentralRequest
	if err := dbQuery.Find(&results).Error; err != nil {
		return nil, errors.NewWithCause(errors.ErrorGeneral, err, "failed to list Central requests")
	}

	// Central requests beyond 'Preparing' should have already been augmented.
	filteredResults := make([]*dbapi.CentralRequest, 0, len(results))
	for _, r := range results {
		if dinosaurConstants.CentralStatus(r.Status).CompareTo(dinosaurConstants.CentralRequestStatusPreparing) <= 0 {
			filteredResults = append(filteredResults, r)
		} else {
			glog.Warningf("Central request %s in status %q lacks auth config which should have been set up earlier", r.ID, r.Status)
		}
	}

	return filteredResults, nil
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
