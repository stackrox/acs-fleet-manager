package services

import (
	"context"
	"database/sql"
	"fmt"
	"reflect"
	"slices"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/route53"
	route53Types "github.com/aws/aws-sdk-go-v2/service/route53/types"
	"github.com/golang/glog"
	"github.com/stackrox/acs-fleet-manager/internal/central/constants"
	"github.com/stackrox/acs-fleet-manager/internal/central/pkg/api/dbapi"
	"github.com/stackrox/acs-fleet-manager/internal/central/pkg/centrals/types"
	"github.com/stackrox/acs-fleet-manager/internal/central/pkg/config"
	"github.com/stackrox/acs-fleet-manager/internal/central/pkg/externaldns"
	"github.com/stackrox/acs-fleet-manager/internal/central/pkg/presenters"
	"github.com/stackrox/acs-fleet-manager/internal/central/pkg/rhsso"
	"github.com/stackrox/acs-fleet-manager/pkg/api"
	"github.com/stackrox/acs-fleet-manager/pkg/auth"
	"github.com/stackrox/acs-fleet-manager/pkg/client/aws"
	"github.com/stackrox/acs-fleet-manager/pkg/client/iam"
	ocm "github.com/stackrox/acs-fleet-manager/pkg/client/ocm/impl"
	dynamicClientAPI "github.com/stackrox/acs-fleet-manager/pkg/client/redhatsso/api"
	"github.com/stackrox/acs-fleet-manager/pkg/client/redhatsso/dynamicclients"
	"github.com/stackrox/acs-fleet-manager/pkg/db"
	"github.com/stackrox/acs-fleet-manager/pkg/environments"
	"github.com/stackrox/acs-fleet-manager/pkg/errors"
	"github.com/stackrox/acs-fleet-manager/pkg/logger"
	"github.com/stackrox/acs-fleet-manager/pkg/metrics"
	"github.com/stackrox/acs-fleet-manager/pkg/services"
	coreServices "github.com/stackrox/acs-fleet-manager/pkg/services/queryparser"
)

var (
	centralDeletionStatuses = []string{
		constants.CentralRequestStatusDeleting.String(),
		constants.CentralRequestStatusDeprovision.String(),
	}

	centralManagedCRStatuses = []string{
		constants.CentralRequestStatusProvisioning.String(),
		constants.CentralRequestStatusDeprovision.String(),
		constants.CentralRequestStatusReady.String(),
		constants.CentralRequestStatusFailed.String(),
	}
)

// CentralRoutesAction ...
type CentralRoutesAction string

// CentralRoutesActionUpsert ...
const CentralRoutesActionUpsert CentralRoutesAction = "UPSERT"

// CentralRoutesActionDelete ...
const CentralRoutesActionDelete CentralRoutesAction = "DELETE"

const gracePeriod = 14 * 24 * time.Hour

// CNameRecordStatus ...
type CNameRecordStatus struct {
	ID     *string
	Status *string
}

// CentralService ...
//
//go:generate moq -out centralservice_moq.go . CentralService
type CentralService interface {
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
	Delete(centralRequest *dbapi.CentralRequest, force bool) *errors.ServiceError
	List(ctx context.Context, listArgs *services.ListArguments) (dbapi.CentralList, *api.PagingMeta, *errors.ServiceError)
	RegisterCentralJob(ctx context.Context, centralRequest *dbapi.CentralRequest) *errors.ServiceError
	ListByStatus(status ...constants.CentralStatus) ([]*dbapi.CentralRequest, *errors.ServiceError)
	// UpdateStatus change the status of the Central cluster
	// The returned boolean is to be used to know if the update has been tried or not. An update is not tried if the
	// original status is 'deprovision' (cluster in deprovision state can't be change state) or if the final status is the
	// same as the original status. The error will contain any error encountered when attempting to update or the reason
	// why no attempt has been done
	UpdateStatus(id string, status constants.CentralStatus) (bool, *errors.ServiceError)
	// UpdateIgnoreNils does NOT update nullable fields when they're nil in the request. Use Updates() instead.
	UpdateIgnoreNils(centralRequest *dbapi.CentralRequest) *errors.ServiceError
	// Updates changes the given fields of a central. This takes in a map so that even zero-fields can be updated.
	// Use this only when you want to update the multiple columns that may contain zero-fields, otherwise use the `CentralService.Update()` method.
	// See https://gorm.io/docs/update.html#Updates-multiple-columns for more info
	Updates(centralRequest *dbapi.CentralRequest, values map[string]interface{}) *errors.ServiceError
	ChangeCentralCNAMErecords(centralRequest *dbapi.CentralRequest, action CentralRoutesAction) (*route53.ChangeResourceRecordSetsOutput, *errors.ServiceError)
	GetCNAMERecordStatus(centralRequest *dbapi.CentralRequest) (*CNameRecordStatus, error)
	DetectInstanceType(centralRequest *dbapi.CentralRequest) types.CentralInstanceType
	RegisterCentralDeprovisionJob(ctx context.Context, id string) *errors.ServiceError
	// DeprovisionCentralForUsers registers all centrals for deprovisioning given the list of owners
	DeprovisionCentralForUsers(users []string) *errors.ServiceError
	DeprovisionExpiredCentrals() *errors.ServiceError
	CountByStatus(status []constants.CentralStatus) ([]CentralStatusCount, error)
	CountByRegionAndInstanceType() ([]CentralRegionCount, error)
	ListCentralsWithRoutesNotCreated() ([]*dbapi.CentralRequest, *errors.ServiceError)
	ListCentralsWithoutAuthConfig() ([]*dbapi.CentralRequest, *errors.ServiceError)
	VerifyAndUpdateCentralAdmin(ctx context.Context, centralRequest *dbapi.CentralRequest) *errors.ServiceError
	Restore(ctx context.Context, id string) *errors.ServiceError
	RotateCentralRHSSOClient(ctx context.Context, centralRequest *dbapi.CentralRequest) *errors.ServiceError
	// ResetCentralSecretBackup resets the Secret field of centralReqest, which are the backed up secrets
	// of a tenant. By resetting the field the next update will store new secrets which enables manual rotation.
	// This is currently the only way to update secret backups, an automatic approach should be implemented
	// to accomated for regular processes like central TLS cert rotation.
	ResetCentralSecretBackup(ctx context.Context, centralRequest *dbapi.CentralRequest) *errors.ServiceError
	ChangeBillingParameters(ctx context.Context, centralID string, billingModel string, cloudAccountID string, cloudProvider string, product string) *errors.ServiceError
	AssignCluster(ctx context.Context, centralID string, clusterID string) *errors.ServiceError
	ChangeSubscription(ctx context.Context, centralID string, cloudAccountID string, cloudProvider string, subscriptionID string) *errors.ServiceError
}

var _ CentralService = &centralService{}

type centralService struct {
	connectionFactory        *db.ConnectionFactory
	clusterService           ClusterService
	centralConfig            *config.CentralConfig
	awsConfig                *config.AWSConfig
	quotaServiceFactory      QuotaServiceFactory
	mu                       sync.Mutex
	awsClientFactory         aws.ClientFactory
	dataplaneClusterConfig   *config.DataplaneClusterConfig
	clusterPlacementStrategy ClusterPlacementStrategy
	amsClient                ocm.AMSClient
	iamConfig                *iam.IAMConfig
	rhSSODynamicClientsAPI   *dynamicClientAPI.AcsTenantsApiService
	telemetry                *Telemetry
	managedCentralPresenter  *presenters.ManagedCentralPresenter
}

// NewCentralService ...
func NewCentralService(connectionFactory *db.ConnectionFactory, clusterService ClusterService,
	iamConfig *iam.IAMConfig, centralConfig *config.CentralConfig, dataplaneClusterConfig *config.DataplaneClusterConfig, awsConfig *config.AWSConfig,
	quotaServiceFactory QuotaServiceFactory, awsClientFactory aws.ClientFactory,
	clusterPlacementStrategy ClusterPlacementStrategy, amsClient ocm.AMSClient, telemetry *Telemetry, managedCentralPresenter *presenters.ManagedCentralPresenter) CentralService {
	return &centralService{
		connectionFactory:        connectionFactory,
		clusterService:           clusterService,
		iamConfig:                iamConfig,
		centralConfig:            centralConfig,
		awsConfig:                awsConfig,
		quotaServiceFactory:      quotaServiceFactory,
		awsClientFactory:         awsClientFactory,
		dataplaneClusterConfig:   dataplaneClusterConfig,
		clusterPlacementStrategy: clusterPlacementStrategy,
		amsClient:                amsClient,
		rhSSODynamicClientsAPI:   dynamicclients.NewDynamicClientsAPI(iamConfig.RedhatSSORealm),
		telemetry:                telemetry,
		managedCentralPresenter:  managedCentralPresenter,
	}
}

func (k *centralService) RotateCentralRHSSOClient(ctx context.Context, centralRequest *dbapi.CentralRequest) *errors.ServiceError {
	realmConfig := k.iamConfig.RedhatSSORealm
	if k.centralConfig.HasStaticAuth() {
		return errors.New(errors.ErrorDynamicClientsNotUsed, "RHSSO is configured via static configuration")
	}
	if !realmConfig.IsConfigured() {
		return errors.New(errors.ErrorDynamicClientsNotUsed, "RHSSO dynamic client configuration is not present")
	}

	previousAuthConfig := centralRequest.AuthConfig
	if err := rhsso.AugmentWithDynamicAuthConfig(ctx, centralRequest, k.iamConfig.RedhatSSORealm, k.rhSSODynamicClientsAPI); err != nil {
		return errors.NewWithCause(errors.ErrorClientRotationFailed, err, "failed to augment auth config")
	}
	if err := k.UpdateIgnoreNils(centralRequest); err != nil {
		glog.Errorf("Rotating RHSSO client failed: created new RHSSO dynamic client, but failed to update central record, client ID is %s", centralRequest.AuthConfig.ClientID)
		return errors.NewWithCause(errors.ErrorClientRotationFailed, err, "failed to update database record")
	}
	if _, err := k.rhSSODynamicClientsAPI.DeleteAcsClient(ctx, previousAuthConfig.ClientID); err != nil {
		glog.Errorf("Rotating RHSSO client failed: failed to delete RHSSO dynamic client, client ID is %s", centralRequest.AuthConfig.ClientID)
		return errors.NewWithCause(errors.ErrorClientRotationFailed, err, "failed to delete previous RHSSO dynamic client")
	}
	return nil
}

func (k *centralService) ResetCentralSecretBackup(ctx context.Context, centralRequest *dbapi.CentralRequest) *errors.ServiceError {
	centralRequest.Secrets = nil // pragma: allowlist secret
	centralRequest.SecretDataSha256Sum = ""
	logStateChange("reset secrets", centralRequest.ID, nil)

	dbConn := k.connectionFactory.New()
	if err := dbConn.Model(centralRequest).Select("secrets", "secret_data_sha256_sum").Updates(centralRequest).Error; err != nil {
		return errors.NewWithCause(errors.ErrorGeneral, err, "Unable to reset secrets for central request")
	}

	return nil
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

	glog.Infof("%d of %d central tenants currently instantiated in region %v", count, regionCapacity, centralRequest.Region)
	return count < regionCapacity, nil
}

// DetectInstanceType - returns standard instance type if quota is available. Otherwise falls back to eval instance type.
func (k *centralService) DetectInstanceType(centralRequest *dbapi.CentralRequest) types.CentralInstanceType {
	quotaType := api.QuotaType(k.centralConfig.Quota.Type)
	quotaService, factoryErr := k.quotaServiceFactory.GetQuotaService(quotaType)
	if factoryErr != nil {
		glog.Error(errors.NewWithCause(errors.ErrorGeneral, factoryErr, "unable to get quota service"))
		return types.EVAL
	}

	hasQuota, err := quotaService.HasQuotaAllowance(centralRequest, types.STANDARD)
	if err != nil {
		glog.Error(errors.NewWithCause(errors.ErrorGeneral, err, "unable to check quota"))
		return types.EVAL
	}
	if hasQuota {
		glog.Infof("Quota detected for central request %s with quota type %s. Granting instance type %s.", centralRequest.ID, quotaType, types.STANDARD)
		return types.STANDARD
	}

	glog.Infof("No quota detected for central request %s with quota type %s. Granting instance type %s.", centralRequest.ID, quotaType, types.EVAL)
	return types.EVAL
}

// reserveQuota - reserves quota for the given central request. If a RHACS quota has been assigned, it will try to reserve RHACS quota, otherwise it will try with RHACSTrial
func (k *centralService) reserveQuota(ctx context.Context, centralRequest *dbapi.CentralRequest, bm string, product string) (subscriptionID string, err *errors.ServiceError) {
	if centralRequest.InstanceType == types.EVAL.String() &&
		!(environments.GetEnvironmentStrFromEnv() == environments.DevelopmentEnv || environments.GetEnvironmentStrFromEnv() == environments.TestingEnv) {
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
			return "", errors.TooManyCentralInstancesReached("only one eval instance is allowed; increase your account quota")
		}
	}

	quotaService, factoryErr := k.quotaServiceFactory.GetQuotaService(api.QuotaType(k.centralConfig.Quota.Type))
	if factoryErr != nil {
		return "", errors.NewWithCause(errors.ErrorGeneral, factoryErr, "unable to check quota")
	}
	subscriptionID, err = quotaService.ReserveQuota(ctx, centralRequest, bm, product)
	return subscriptionID, err
}

// RegisterCentralJob registers a new job in the central table
func (k *centralService) RegisterCentralJob(ctx context.Context, centralRequest *dbapi.CentralRequest) *errors.ServiceError {
	k.mu.Lock()
	defer k.mu.Unlock()
	// we need to pre-populate the ID to be able to reserve the quota
	centralRequest.ID = api.NewID()

	if hasCapacity, err := k.HasAvailableCapacityInRegion(centralRequest); err != nil {
		return errors.NewWithCause(errors.ErrorGeneral, err, "failed to create central request")
	} else if !hasCapacity {
		errorMsg := fmt.Sprintf("Cluster capacity(%d) exhausted in %s region", int64(k.dataplaneClusterConfig.ClusterConfig.GetCapacityForRegion(centralRequest.Region)), centralRequest.Region)
		logger.Logger.Warningf(errorMsg)
		return errors.TooManyCentralInstancesReached("%s", errorMsg)
	}

	instanceType := k.DetectInstanceType(centralRequest)

	centralRequest.InstanceType = instanceType.String()

	cluster, e := k.clusterPlacementStrategy.FindCluster(centralRequest)
	if e != nil || cluster == nil {
		msg := fmt.Sprintf("No available cluster found for '%s' central instance in region: '%s'", centralRequest.InstanceType, centralRequest.Region)
		logger.Logger.Errorf(msg)
		return errors.TooManyCentralInstancesReached("Region %s cannot accept instance type: %s at this moment", centralRequest.Region, centralRequest.InstanceType)
	}
	centralRequest.ClusterID = cluster.ClusterID
	subscriptionID, err := k.reserveQuota(ctx, centralRequest, "", "")
	if err != nil {
		return err
	}

	dbConn := k.connectionFactory.New()
	centralRequest.Status = constants.CentralRequestStatusAccepted.String()
	centralRequest.SubscriptionID = subscriptionID
	glog.Infof("Central request %s has been assigned the subscription %s.", centralRequest.ID, subscriptionID)
	// Persist the QuotaType to be able to dynamically pick the right Quota service implementation even on restarts.
	// A typical usecase is when a central A is created, at the time of creation the quota-type was ams. At some point in the future
	// the API is restarted this time changing the --quota-type flag to quota-management-list, when central A is deleted at this point,
	// we want to use the correct quota to perform the deletion.
	centralRequest.QuotaType = k.centralConfig.Quota.Type

	logStateChange("register central job", centralRequest.ID, centralRequest)

	if err := dbConn.Create(centralRequest).Error; err != nil {
		return errors.NewWithCause(errors.ErrorGeneral, err, "failed to create central request") // hide the db error to http caller
	}
	metrics.UpdateCentralRequestsStatusSinceCreatedMetric(constants.CentralRequestStatusAccepted, centralRequest.ID, centralRequest.ClusterID, time.Since(centralRequest.CreatedAt))
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
	if k.centralConfig.EnableCentralExternalDomain {
		// the host should use the external domain name rather than the cluster domain
		centralRequest.Host = k.centralConfig.CentralDomainName
	} else {
		clusterDNS, err := k.clusterService.GetClusterDNS(centralRequest.ClusterID)
		if err != nil {
			return errors.NewWithCause(errors.ErrorGeneral, err, "error retrieving cluster DNS")
		}
		centralRequest.Host = clusterDNS
	}

	// UpdateIgnoreNils the fields of the CentralRequest record in the database.
	updatedCentralRequest := &dbapi.CentralRequest{
		Meta: api.Meta{
			ID: centralRequest.ID,
		},
		Host:        centralRequest.Host,
		PlacementID: api.NewID(),
		Status:      constants.CentralRequestStatusPreparing.String(),
		Namespace:   centralRequest.Namespace,
	}
	if err := k.UpdateIgnoreNils(updatedCentralRequest); err != nil {
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
	// TODO(alexr): Shall this go into "preparing_centrals_mgr.go"? Ideally,
	//     all CentralRequest updating logic is in one place, either in this
	//     service or workers.
	if centralRequest.AuthConfig.ClientID == "" {
		// We can't provision this request, skip
		return nil
	}

	// Obtain organisation name from AMS to store in central request.
	org, err := k.amsClient.GetOrganisationFromExternalID(centralRequest.OrganisationID)
	if err != nil {
		return errors.OrganisationNotFound(centralRequest.OrganisationID, err)
	}
	orgName := org.Name()
	if orgName == "" {
		return errors.OrganisationNameInvalid(centralRequest.OrganisationID, orgName)
	}

	// UpdateIgnoreNils the fields of the CentralRequest record in the database.
	now := time.Now()
	updatedCentralRequest := &dbapi.CentralRequest{
		Meta: api.Meta{
			ID: centralRequest.ID,
		},
		OrganisationName:      orgName,
		Status:                constants.CentralRequestStatusProvisioning.String(),
		EnteredProvisioningAt: dbapi.TimePtrToNullTime(&now),
	}
	if err := k.UpdateIgnoreNils(updatedCentralRequest); err != nil {
		return errors.NewWithCause(errors.ErrorGeneral, err, "failed to update central request")
	}

	return nil
}

// ListByStatus ...
func (k *centralService) ListByStatus(status ...constants.CentralStatus) ([]*dbapi.CentralRequest, *errors.ServiceError) {
	if len(status) == 0 {
		return nil, errors.GeneralError("no status provided")
	}
	dbConn := k.connectionFactory.New()

	var requests []*dbapi.CentralRequest

	if err := dbConn.Model(&dbapi.CentralRequest{}).Where("status IN (?)", status).Scan(&requests).Error; err != nil {
		return nil, errors.NewWithCause(errors.ErrorGeneral, err, "failed to list by status")
	}

	return requests, nil
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
	metrics.IncreaseCentralTotalOperationsCountMetric(constants.CentralOperationDeprovision)

	deprovisionStatus := constants.CentralRequestStatusDeprovision

	if executed, err := k.UpdateStatus(id, deprovisionStatus); executed {
		if err != nil {
			return services.HandleGetError("CentralResource", "id", id, err)
		}
		metrics.IncreaseCentralSuccessOperationsCountMetric(constants.CentralOperationDeprovision)
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
			"status":             constants.CentralRequestStatusDeprovision,
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
			metrics.IncreaseCentralTotalOperationsCountMetric(constants.CentralOperationDeprovision)
			metrics.IncreaseCentralSuccessOperationsCountMetric(constants.CentralOperationDeprovision)
		}
	}

	return nil
}

// DeprovisionExpiredCentrals cleaning up expired centrals
func (k *centralService) DeprovisionExpiredCentrals() *errors.ServiceError {
	now := time.Now()
	dbConn := k.connectionFactory.New().Model(&dbapi.CentralRequest{}).
		Where("expired_at IS NOT NULL").Where("expired_at < ?", now.Add(-gracePeriod))

	if k.centralConfig.CentralLifespan.EnableDeletionOfExpiredCentral {
		dbConn = dbConn.Where(dbConn.
			Or("instance_type = ?", types.EVAL.String()).
			Where("created_at <= ?", now.Add(
				-time.Duration(k.centralConfig.CentralLifespan.CentralLifespanInHours)*time.Hour)))
	}

	dbConn = dbConn.Where("status NOT IN (?)", centralDeletionStatuses)

	db := dbConn.Updates(map[string]interface{}{
		"status":             constants.CentralRequestStatusDeprovision,
		"deletion_timestamp": now,
	})
	err := db.Error
	if err != nil {
		return errors.NewWithCause(errors.ErrorGeneral, err, "unable to deprovision expired centrals")
	}

	if db.RowsAffected >= 1 {
		glog.Infof("%v central_request's have had their status updated to deprovisioning", db.RowsAffected)
		var counter int64
		for ; counter < db.RowsAffected; counter++ {
			metrics.IncreaseCentralTotalOperationsCountMetric(constants.CentralOperationDeprovision)
			metrics.IncreaseCentralSuccessOperationsCountMetric(constants.CentralOperationDeprovision)
		}
	}

	return nil
}

// Delete a CentralRequest from the database.
// The implementation uses soft-deletion (via GORM).
// If the force flag is true, then any errors prior to the final deletion of the CentralRequest will be logged as warnings
// but do not interrupt the deletion flow.
func (k *centralService) Delete(centralRequest *dbapi.CentralRequest, force bool) *errors.ServiceError {
	dbConn := k.connectionFactory.New()

	// if the we don't have the clusterID we can only delete the row from the database
	if centralRequest.ClusterID != "" {
		routes, err := centralRequest.GetRoutes()
		if err != nil {
			return errors.NewWithCause(errors.ErrorGeneral, err, "failed to get routes")
		}
		managedCentral, err := k.managedCentralPresenter.PresentManagedCentral(centralRequest)
		if err != nil {
			return errors.NewWithCause(errors.ErrorGeneral, err, "failed to present managed central")
		}
		// Only delete the routes when they are set
		if routes != nil && k.centralConfig.EnableCentralExternalDomain && !externaldns.IsEnabled(managedCentral) {
			_, err := k.ChangeCentralCNAMErecords(centralRequest, CentralRoutesActionDelete)
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

	logStateChange("delete request", centralRequest.ID, nil)
	// soft delete the central request
	if err := dbConn.Delete(centralRequest).Error; err != nil {
		return errors.NewWithCause(errors.ErrorGeneral, err, "unable to delete central request with id %s", centralRequest.ID)
	}

	if force {
		glog.Infof("Make sure any other resources belonging to the Central tenant %q are manually deleted.", centralRequest.ID)
	}
	metrics.IncreaseCentralTotalOperationsCountMetric(constants.CentralOperationDelete)
	metrics.IncreaseCentralSuccessOperationsCountMetric(constants.CentralOperationDelete)

	return nil
}

// List returns all Central requests belonging to a user.
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

// Update ...
func (k *centralService) UpdateIgnoreNils(centralRequest *dbapi.CentralRequest) *errors.ServiceError {
	dbConn := k.connectionFactory.New().
		Model(centralRequest).
		Where("status not IN (?)", centralDeletionStatuses) // ignore updates of central under deletion

	logStateChange("updates", centralRequest.ID, centralRequest)

	if err := dbConn.Updates(centralRequest).Error; err != nil {
		return errors.NewWithCause(errors.ErrorGeneral, err, "Failed to update central")
	}
	k.telemetry.UpdateTenantProperties(centralRequest)
	return nil
}

// Updates ...
func (k *centralService) Updates(centralRequest *dbapi.CentralRequest, fields map[string]interface{}) *errors.ServiceError {
	dbConn := k.connectionFactory.New().
		Model(centralRequest).
		Where("status not IN (?)", centralDeletionStatuses) // ignore updates of central under deletion

	glog.Infof("instance state change: id=%q: fields=%+v", centralRequest.ID, fields)

	if err := dbConn.Updates(fields).Error; err != nil {
		return errors.NewWithCause(errors.ErrorGeneral, err, "Failed to update central")
	}
	// Get all request properties, not only the ones provided with fields.
	if centralRequest, svcErr := k.GetByID(centralRequest.ID); svcErr == nil {
		k.telemetry.UpdateTenantProperties(centralRequest)
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
		return errors.New(errors.ErrorValidation, "Unable to get cluster for central %s", centralRequest.ID)
	}

	return k.UpdateIgnoreNils(centralRequest)
}

// UpdateStatus ...
func (k *centralService) UpdateStatus(id string, status constants.CentralStatus) (bool, *errors.ServiceError) {
	dbConn := k.connectionFactory.New()

	central, err := k.GetByID(id)
	if err != nil {
		return true, errors.NewWithCause(errors.ErrorGeneral, err, "failed to update status")
	}
	// only allow to change the status to "deleting" if the cluster is already in "deprovision" status
	if central.Status == constants.CentralRequestStatusDeprovision.String() && status != constants.CentralRequestStatusDeleting {
		return false, errors.GeneralError("failed to update status: cluster is deprovisioning")
	}

	if central.Status == status.String() {
		// no update needed
		return false, errors.GeneralError("failed to update status: the cluster %s is already in %s state", id, status.String())
	}

	update := &dbapi.CentralRequest{Status: status.String()}
	if status.String() == constants.CentralRequestStatusDeprovision.String() {
		now := time.Now()
		update.DeletionTimestamp = sql.NullTime{Time: now, Valid: true}
	}

	logStateChange(fmt.Sprintf("change status to %q", status.String()), id, nil)

	if err := dbConn.Model(&dbapi.CentralRequest{Meta: api.Meta{ID: id}}).Updates(update).Error; err != nil {
		return true, errors.NewWithCause(errors.ErrorGeneral, err, "Failed to update central status")
	}
	k.telemetry.UpdateTenantProperties(central)
	return true, nil
}

// ChangeCentralCNAMErecords ...
func (k *centralService) ChangeCentralCNAMErecords(centralRequest *dbapi.CentralRequest, action CentralRoutesAction) (*route53.ChangeResourceRecordSetsOutput, *errors.ServiceError) {
	routes, err := centralRequest.GetRoutes()
	if routes == nil || err != nil {
		return nil, errors.NewWithCause(errors.ErrorGeneral, err, "failed to get routes")
	}

	changeAction, err := CentralRoutesActionToRoute53ChangeAction(action)
	domainRecordBatch := buildCentralClusterCNAMESRecordBatch(routes, changeAction)

	// Create AWS client with the region of this Central Cluster
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

	status := string(changeOutput.ChangeInfo.Status)
	return &CNameRecordStatus{
		ID:     changeOutput.ChangeInfo.Id,
		Status: &status,
	}, nil
}

func (k *centralService) Restore(ctx context.Context, id string) *errors.ServiceError {
	dbConn := k.connectionFactory.New()
	var centralRequest dbapi.CentralRequest
	if err := dbConn.Unscoped().Where("id = ?", id).First(&centralRequest).Error; err != nil {
		return services.HandleGetError("CentralRequest", "id", id, err)
	}

	if !centralRequest.DeletedAt.Valid {
		return errors.BadRequest("CentralRequests not marked as deleted.")
	}

	timeSinceDeletion := time.Since(centralRequest.DeletedAt.Time)
	if timeSinceDeletion.Hours()/24 > float64(k.centralConfig.CentralRetentionPeriodDays) {
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
		// reset expired_at to null: it may later be updated by the next run
		// of the expiration manager. If there is still no quota, the grace
		// period will start over.
		"ExpiredAt",
		"EnteredProvisioningAt",
	}

	// use a new central request, so that unset field for columnsToReset will automatically be set to the zero value
	// this UpdateIgnoreNils only changes columns listed in columnsToReset
	resetRequest := &dbapi.CentralRequest{}
	resetRequest.ID = centralRequest.ID
	resetRequest.Status = constants.CentralRequestStatusPreparing.String()
	now := time.Now()
	resetRequest.CreatedAt = now
	resetRequest.EnteredProvisioningAt = dbapi.TimePtrToNullTime(&now)

	logStateChange("restore", resetRequest.ID, resetRequest)

	if err := dbConn.Unscoped().Model(resetRequest).Select(columnsToReset).Updates(resetRequest).Error; err != nil {
		return errors.NewWithCause(errors.ErrorGeneral, err, "Unable to reset CentralRequest status")
	}

	return nil
}

func (k *centralService) AssignCluster(ctx context.Context, centralID string, clusterID string) *errors.ServiceError {
	central, serviceErr := k.GetByID(centralID)
	if serviceErr != nil {
		return serviceErr
	}

	readyStatus := constants.CentralRequestStatusReady.String()
	provisioningStatus := constants.CentralRequestStatusProvisioning.String()
	if central.Status != readyStatus && central.Status != provisioningStatus {
		return errors.BadRequest("Cannot assing cluster_id for tenant in status: %q, status %q is required", central.Status, readyStatus)
	}

	clusters, err := AllMatchingClustersForCentral(central, k.clusterService)
	if err != nil {
		glog.Errorf("internal error getting all matching cluster for central: %q, err: %s", centralID, err.Error())
		return errors.GeneralError("error getting matching clusters for central: %q", centralID)
	}

	if !slices.ContainsFunc(clusters, func(c *api.Cluster) bool { return c.ClusterID == clusterID }) {
		return errors.BadRequest("Given cluster_id: %q not found in list of matching clusters for central: %q.", clusterID, centralID)
	}

	central.ClusterID = clusterID
	central.RoutesCreated = false
	central.Routes = nil
	central.RoutesCreationID = ""
	central.Status = constants.CentralRequestStatusProvisioning.String()
	now := time.Now()
	central.EnteredProvisioningAt = dbapi.TimePtrToNullTime(&now)

	return k.Updates(central, map[string]interface{}{
		"cluster_id":              central.ClusterID,
		"routes_created":          central.RoutesCreated,
		"routes":                  central.Routes,
		"status":                  central.Status,
		"routes_creation_id":      central.RoutesCreationID,
		"entered_provisioning_at": central.EnteredProvisioningAt,
	})
}

// CentralStatusCount ...
type CentralStatusCount struct {
	Status constants.CentralStatus
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
func (k *centralService) CountByStatus(status []constants.CentralStatus) ([]CentralStatusCount, error) {
	dbConn := k.connectionFactory.New()
	var results []CentralStatusCount
	if err := dbConn.Model(&dbapi.CentralRequest{}).Select("status as Status, count(1) as Count").Where("status in (?)", status).Group("status").Scan(&results).Error; err != nil {
		return nil, errors.NewWithCause(errors.ErrorGeneral, err, "Failed to count centrals")
	}

	// if there is no count returned for a status from the above query because there is no centrals in such a status,
	// we should return the count for these as well to avoid any confusion
	if len(status) > 0 {
		countersMap := map[constants.CentralStatus]int{}
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
// no auth config. For central requests without host set, we cannot compute
// redirect_uri and hence cannot set up auth config.
func (k *centralService) ListCentralsWithoutAuthConfig() ([]*dbapi.CentralRequest, *errors.ServiceError) {
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
		if constants.CentralStatus(r.Status).CompareTo(constants.CentralRequestStatusPreparing) <= 0 {
			filteredResults = append(filteredResults, r)
		} else {
			glog.Warningf("Central request %s in status %q lacks auth config which should have been set up earlier", r.ID, r.Status)
		}
	}

	return filteredResults, nil
}

func buildCentralClusterCNAMESRecordBatch(routes []dbapi.DataPlaneCentralRoute, action route53Types.ChangeAction) *route53Types.ChangeBatch {
	var changes []route53Types.Change
	for _, r := range routes {
		c := buildResourceRecordChange(r.Domain, r.Router, action)
		changes = append(changes, c)
	}
	recordChangeBatch := &route53Types.ChangeBatch{
		Changes: changes,
	}

	return recordChangeBatch
}

func buildResourceRecordChange(recordName string, clusterIngress string, action route53Types.ChangeAction) route53Types.Change {
	recordTTL := int64(300)

	resourceRecordChange := route53Types.Change{
		Action: action,
		ResourceRecordSet: &route53Types.ResourceRecordSet{
			Name: &recordName,
			Type: route53Types.RRTypeCname,
			TTL:  &recordTTL,
			ResourceRecords: []route53Types.ResourceRecord{
				{
					Value: &clusterIngress,
				},
			},
		},
	}

	return resourceRecordChange
}

func logStateChange(msg, id string, req *dbapi.CentralRequest) {
	if req != nil {
		glog.Infof("instance state change: id=%q: message=%s: request=%+v", id, msg, convertCentralRequestToString(req))
	} else {
		glog.Infof("instance state change: id=%q: message=%s", id, msg)
	}
}

func convertCentralRequestToString(req *dbapi.CentralRequest) string {
	traits, _ := req.Traits.Value()
	requestAsMap := map[string]interface{}{
		"id":                      req.ID,
		"created_at":              req.CreatedAt,
		"updated_at":              req.UpdatedAt,
		"deleted_at":              req.DeletedAt,
		"region":                  req.Region,
		"cluster_id":              req.ClusterID,
		"cloud_provider":          req.CloudProvider,
		"cloud_account_id":        req.CloudAccountID,
		"multi_az":                req.MultiAZ,
		"name":                    req.Name,
		"status":                  req.Status,
		"subscription_id":         req.SubscriptionID,
		"owner":                   req.Owner,
		"owner_account_id":        req.OwnerAccountID,
		"owner_user_id":           req.OwnerUserID,
		"owner_alternate_user_id": req.OwnerAlternateUserID,
		"host":                    req.Host,
		"organisation_id":         req.OrganisationID,
		"organisation_name":       req.OrganisationName,
		"failed_reason":           req.FailedReason,
		"placement_id":            req.PlacementID,
		"instance_type":           req.InstanceType,
		"qouta_type":              req.QuotaType,
		"routes_created":          req.RoutesCreated,
		"namespace":               req.Namespace,
		"routes_creation_id":      req.RoutesCreationID,
		"deletion_timestamp":      req.DeletionTimestamp,
		"internal":                req.Internal,
		"expired_at":              req.ExpiredAt,
		"traits":                  traits,
		"issuer":                  req.Issuer,
		"client_origin":           req.ClientOrigin,
	}
	return fmt.Sprintf("%+v", requestAsMap)
}

type billingParameters struct {
	cloudAccountID string
	cloudProvider  string
	subscriptionID string
	instanceType   string
	product        string
}

func makeBillingParameters(central *dbapi.CentralRequest) *billingParameters {
	return &billingParameters{
		cloudAccountID: central.CloudAccountID,
		cloudProvider:  central.CloudProvider,
		subscriptionID: central.SubscriptionID,
		instanceType:   central.InstanceType,
		product:        types.CentralInstanceType(central.InstanceType).GetQuotaType().GetProduct(),
	}
}

func (k *centralService) ChangeBillingParameters(ctx context.Context, centralID string, billingModel string, cloudAccountID string, cloudProvider string, product string) *errors.ServiceError {
	centralRequest, svcErr := k.GetByID(centralID)
	if svcErr != nil {
		return svcErr
	}

	original := makeBillingParameters(centralRequest)

	centralRequest.CloudAccountID = cloudAccountID
	centralRequest.CloudProvider = cloudProvider

	// Changing product is allowed (by OCM) only from RHACSTrial to RHACS today.
	// This change should also change the instance type.
	if original.product == string(ocm.RHACSTrialProduct) && product == string(ocm.RHACSProduct) {
		centralRequest.InstanceType = string(types.STANDARD)
	}

	newSubscriptionID, svcErr := k.reserveQuota(ctx, centralRequest, billingModel, product)
	updated := makeBillingParameters(centralRequest)
	if svcErr != nil {
		glog.Errorf("Failed to reserve quota with updated billing parameters (%+v): %v", updated, svcErr)
		return svcErr
	}
	updated.subscriptionID = newSubscriptionID
	centralRequest.SubscriptionID = newSubscriptionID

	if !reflect.DeepEqual(original, updated) {
		if svcErr = k.UpdateIgnoreNils(centralRequest); svcErr != nil {
			glog.Errorf("Failed to update central %q record with updated billing parameters (%v): %v", centralID, updated, svcErr)
			return svcErr
		}
		glog.Infof("Central %q billing parameters have been changed from %v to %v", centralID, original, updated)
	} else {
		glog.Infof("Central %q has no change in billing parameters")
	}
	return nil
}

// ChangeSubscription implements CentralService.
func (k *centralService) ChangeSubscription(ctx context.Context, centralID string, cloudAccountID string, cloudProvider string, subscriptionID string) *errors.ServiceError {
	centralRequest, svcErr := k.GetByID(centralID)
	if svcErr != nil {
		return svcErr
	}

	centralRequest.CloudProvider = cloudProvider
	centralRequest.CloudAccountID = cloudAccountID
	centralRequest.SubscriptionID = subscriptionID

	if svcErr = k.UpdateIgnoreNils(centralRequest); svcErr != nil {
		glog.Errorf("Failed to update central %q record with subscription_id %q and updated cloud account %q: %v", centralID, subscriptionID, cloudAccountID, svcErr)
		return svcErr
	}

	glog.Infof("Central %q cloud account parameters have been changed to %q with id %q", centralID, cloudProvider, cloudAccountID)
	return nil
}

// CentralRoutesActionToRoute53ChangeAction converts a CentralRoutesAction to a route53 types ChangeAction
func CentralRoutesActionToRoute53ChangeAction(a CentralRoutesAction) (route53Types.ChangeAction, error) {
	changeAction := route53Types.ChangeAction(a)
	switch changeAction {
	case route53Types.ChangeActionCreate, route53Types.ChangeActionUpsert, route53Types.ChangeAction(CentralRoutesActionDelete):
		return changeAction, nil
	default:
		return "", fmt.Errorf("invalid CentralChangeAction: %q, cannot convert to Route53 action", changeAction)
	}
}
