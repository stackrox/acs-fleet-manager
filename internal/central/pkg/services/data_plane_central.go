package services

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/golang/glog"
	"github.com/pkg/errors"
	"github.com/stackrox/acs-fleet-manager/internal/central/constants"
	"github.com/stackrox/acs-fleet-manager/internal/central/pkg/api/dbapi"
	"github.com/stackrox/acs-fleet-manager/internal/central/pkg/config"
	"github.com/stackrox/acs-fleet-manager/pkg/api"
	"github.com/stackrox/acs-fleet-manager/pkg/db"
	serviceError "github.com/stackrox/acs-fleet-manager/pkg/errors"
	"github.com/stackrox/acs-fleet-manager/pkg/logger"
	"github.com/stackrox/acs-fleet-manager/pkg/metrics"
)

type centralStatus string

const (
	statusInstalling centralStatus = "installing"
	statusReady      centralStatus = "ready"
	statusError      centralStatus = "error"
	statusRejected   centralStatus = "rejected"
	statusDeleted    centralStatus = "deleted"
	statusUnknown    centralStatus = "unknown"
)

// DataPlaneCentralService ...
type DataPlaneCentralService interface {
	UpdateDataPlaneCentralService(ctx context.Context, clusterID string, status []*dbapi.DataPlaneCentralStatus) *serviceError.ServiceError
	ListByClusterID(clusterID string) (dbapi.CentralList, *serviceError.ServiceError)
}

type dataPlaneCentralService struct {
	centralService         CentralService
	clusterService         ClusterService
	connectionFactory      *db.ConnectionFactory
	dataplaneClusterConfig *config.DataplaneClusterConfig
}

// NewDataPlaneCentralService ...
func NewDataPlaneCentralService(
	centralSrv CentralService,
	clusterSrv ClusterService,
	connectionFactory *db.ConnectionFactory,
	dataplaneClusterConfig *config.DataplaneClusterConfig,
) DataPlaneCentralService {
	return &dataPlaneCentralService{
		centralService:         centralSrv,
		clusterService:         clusterSrv,
		connectionFactory:      connectionFactory,
		dataplaneClusterConfig: dataplaneClusterConfig,
	}
}

// UpdateDataPlaneCentralService ...
func (s *dataPlaneCentralService) UpdateDataPlaneCentralService(ctx context.Context, clusterID string, status []*dbapi.DataPlaneCentralStatus) *serviceError.ServiceError {
	cluster, err := s.clusterService.FindClusterByID(clusterID)
	log := logger.NewUHCLogger(ctx)
	if err != nil {
		return err
	}
	if cluster == nil {
		// 404 is used for authenticated requests. So to distinguish the errors, we use 400 here
		return serviceError.BadRequest("Cluster id %s not found", clusterID)
	}
	for _, ks := range status {
		central, getErr := s.centralService.GetByID(ks.CentralClusterID)
		if getErr != nil {
			glog.Error(errors.Wrapf(getErr, "failed to get central cluster by id %s", ks.CentralClusterID))
			continue
		}
		if central.ClusterID != clusterID {
			log.Warningf("clusterId for central cluster %s does not match clusterId. central clusterId = %s :: clusterId = %s", central.ID, central.ClusterID, clusterID)
			continue
		}
		var e *serviceError.ServiceError
		switch getStatus(ks) {
		case statusReady:
			// Persist values only known once central is in statusReady e.g. routes, secrets
			e = s.persistCentralValues(central, ks, cluster)
			if e == nil {
				e = s.setCentralClusterReady(central)
			}
		case statusError:
			// when getStatus returns statusError we know that the ready
			// condition will be there so there's no need to check for it
			readyCondition, _ := ks.GetReadyCondition()
			e = s.setCentralClusterFailed(central, readyCondition.Message)
		case statusDeleted:
			e = s.setCentralClusterDeleting(central)
		case statusRejected:
			e = s.reassignCentralCluster(central)
		case statusUnknown:
			log.Infof("central cluster %s status is unknown", ks.CentralClusterID)
		default:
			log.V(5).Infof("central cluster %s is still installing", ks.CentralClusterID)
		}
		if e != nil {
			log.Error(errors.Wrapf(e, "Error updating central %s status", ks.CentralClusterID))
		}
	}

	return nil
}

// ListByClusterID returns a list of CentralRequests with specified clusterID
func (s *dataPlaneCentralService) ListByClusterID(clusterID string) (dbapi.CentralList, *serviceError.ServiceError) {
	dbConn := s.connectionFactory.New().
		Where("cluster_id = ?", clusterID).
		Where("status IN (?)", centralManagedCRStatuses).
		Where("host != ''")

	var centralRequests dbapi.CentralList
	if err := dbConn.Find(&centralRequests).Error; err != nil {
		return nil, serviceError.NewWithCause(serviceError.ErrorGeneral, err, "unable to list central requests")
	}

	return centralRequests, nil
}

func (s *dataPlaneCentralService) setCentralClusterReady(centralRequest *dbapi.CentralRequest) *serviceError.ServiceError {
	if !centralRequest.RoutesCreated {
		logger.Logger.V(10).Infof("routes for central %s are not created", centralRequest.ID)
		return nil
	}
	logger.Logger.Infof("routes for central %s are created", centralRequest.ID)

	// only send metrics data if the current central request is in "provisioning" status as this is the only case we want to report
	shouldSendMetric, err := s.checkCentralRequestCurrentStatus(centralRequest, constants.CentralRequestStatusProvisioning)
	if err != nil {
		return err
	}

	err = s.centralService.Updates(centralRequest, map[string]interface{}{"failed_reason": "", "status": constants.CentralRequestStatusReady.String()})
	if err != nil {
		return serviceError.NewWithCause(err.Code, err, "failed to update status %s for central cluster %s", constants.CentralRequestStatusReady, centralRequest.ID)
	}
	if shouldSendMetric {
		metrics.UpdateCentralRequestsStatusSinceCreatedMetric(constants.CentralRequestStatusReady, centralRequest.ID, centralRequest.ClusterID, time.Since(centralRequest.CreatedAt))
		metrics.UpdateCentralCreationDurationMetric(metrics.JobTypeCentralCreate, time.Since(centralRequest.CreatedAt))
		metrics.IncreaseCentralSuccessOperationsCountMetric(constants.CentralOperationCreate)
		metrics.IncreaseCentralTotalOperationsCountMetric(constants.CentralOperationCreate)
	}
	return nil
}

func (s *dataPlaneCentralService) setCentralClusterFailed(centralRequest *dbapi.CentralRequest, errMessage string) *serviceError.ServiceError {
	// if central was already reported as failed we don't do anything
	if centralRequest.Status == string(constants.CentralRequestStatusFailed) {
		return nil
	}

	// only send metrics data if the current central request is in "provisioning" status as this is the only case we want to report
	shouldSendMetric, err := s.checkCentralRequestCurrentStatus(centralRequest, constants.CentralRequestStatusProvisioning)
	if err != nil {
		return err
	}

	centralRequest.Status = string(constants.CentralRequestStatusFailed)
	centralRequest.FailedReason = fmt.Sprintf("Central reported as failed: '%s'", errMessage)
	err = s.centralService.UpdateIgnoreNils(centralRequest)
	if err != nil {
		return serviceError.NewWithCause(err.Code, err, "failed to update central cluster to %s status for central cluster %s", constants.CentralRequestStatusFailed, centralRequest.ID)
	}
	if shouldSendMetric {
		metrics.UpdateCentralRequestsStatusSinceCreatedMetric(constants.CentralRequestStatusFailed, centralRequest.ID, centralRequest.ClusterID, time.Since(centralRequest.CreatedAt))
		metrics.IncreaseCentralTotalOperationsCountMetric(constants.CentralOperationCreate)
	}
	clusterName := ""
	if s.dataplaneClusterConfig != nil {
		clusterName = s.dataplaneClusterConfig.FindClusterNameByClusterID(centralRequest.ClusterID)
	}
	logger.Logger.Errorf("Central status for Central ID '%s' in ClusterID '%s' (%s) reported as failed by Fleet Shard Operator: '%s'", centralRequest.ID, centralRequest.ClusterID, clusterName, errMessage)

	return nil
}

func (s *dataPlaneCentralService) setCentralClusterDeleting(centralRequest *dbapi.CentralRequest) *serviceError.ServiceError {
	// If the Central cluster is deleted from the data plane cluster, we will make it as "deleting" in db and the reconcilier will ensure it is cleaned up properly
	if ok, updateErr := s.centralService.UpdateStatus(centralRequest.ID, constants.CentralRequestStatusDeleting); ok {
		if updateErr != nil {
			return serviceError.NewWithCause(updateErr.Code, updateErr, "failed to update status %s for central cluster %s", constants.CentralRequestStatusDeleting, centralRequest.ID)
		}
		metrics.UpdateCentralRequestsStatusSinceCreatedMetric(constants.CentralRequestStatusDeleting, centralRequest.ID, centralRequest.ClusterID, time.Since(centralRequest.CreatedAt))
	}
	return nil
}

func (s *dataPlaneCentralService) reassignCentralCluster(centralRequest *dbapi.CentralRequest) *serviceError.ServiceError {
	if centralRequest.Status == constants.CentralRequestStatusProvisioning.String() {
		// If Central is rejected by the fleetshard-sync, it should be assigned to another Data Plane cluster (via some scheduler service in the future).
		// But now we only have one Data Plane cluster, so we need to change the placementId field so that the fleetshard-sync will try it again
		// In the future, we may consider adding a new table to track the placement history for central clusters if there are multiple Data Plane clusters and the value here can be the key of that table
		centralRequest.PlacementID = api.NewID()
		if err := s.centralService.UpdateIgnoreNils(centralRequest); err != nil {
			return err
		}
		metrics.UpdateCentralRequestsStatusSinceCreatedMetric(constants.CentralRequestStatusProvisioning, centralRequest.ID, centralRequest.ClusterID, time.Since(centralRequest.CreatedAt))
	} else {
		logger.Logger.Infof("central cluster %s is rejected and current status is %s", centralRequest.ID, centralRequest.Status)
	}

	return nil
}

func getStatus(status *dbapi.DataPlaneCentralStatus) centralStatus {
	for _, c := range status.Conditions {
		if strings.EqualFold(c.Type, "Ready") {
			if strings.EqualFold(c.Status, "True") {
				return statusReady
			}
			if strings.EqualFold(c.Status, "Unknown") {
				return statusUnknown
			}
			if strings.EqualFold(c.Reason, "Installing") {
				return statusInstalling
			}
			if strings.EqualFold(c.Reason, "Deleted") {
				return statusDeleted
			}
			if strings.EqualFold(c.Reason, "Error") {
				return statusError
			}
			if strings.EqualFold(c.Reason, "Rejected") {
				return statusRejected
			}
		}
	}
	return statusInstalling
}
func (s *dataPlaneCentralService) checkCentralRequestCurrentStatus(centralRequest *dbapi.CentralRequest, status constants.CentralStatus) (bool, *serviceError.ServiceError) {
	matchStatus := false
	if currentInstance, err := s.centralService.GetByID(centralRequest.ID); err != nil {
		return matchStatus, err
	} else if currentInstance.Status == status.String() {
		matchStatus = true
	}
	return matchStatus, nil
}

func (s *dataPlaneCentralService) persistCentralValues(centralRequest *dbapi.CentralRequest, centralStatus *dbapi.DataPlaneCentralStatus, cluster *api.Cluster) *serviceError.ServiceError {
	if err := s.addRoutesToRequest(centralRequest, centralStatus, cluster); err != nil {
		return err
	}

	if err := s.addSecretsToRequest(centralRequest, centralStatus, cluster); err != nil {
		return err
	}

	if err := s.centralService.UpdateIgnoreNils(centralRequest); err != nil {
		return serviceError.NewWithCause(err.Code, err, "failed to update routes for central cluster %s", centralRequest.ID)
	}

	return nil
}

func (s *dataPlaneCentralService) addRoutesToRequest(centralRequest *dbapi.CentralRequest, centralStatus *dbapi.DataPlaneCentralStatus, cluster *api.Cluster) *serviceError.ServiceError {
	if centralRequest.Routes != nil {
		logger.Logger.V(10).Infof("skip persisting routes for Central %s as they are already stored", centralRequest.ID)
		return nil
	}
	logger.Logger.Infof("store routes information for central %s", centralRequest.ID)
	clusterDNS, err := s.clusterService.GetClusterDNS(cluster.ClusterID)
	if err != nil {
		return serviceError.NewWithCause(err.Code, err, "failed to get DNS entry for cluster %s", cluster.ClusterID)
	}

	routesInRequest := centralStatus.Routes

	if routesErr := validateRouters(routesInRequest, centralRequest, clusterDNS); routesErr != nil {
		return serviceError.NewWithCause(serviceError.ErrorBadRequest, routesErr, "routes are not valid")
	}

	if err := centralRequest.SetRoutes(routesInRequest); err != nil {
		return serviceError.NewWithCause(serviceError.ErrorGeneral, err, "failed to set routes for central %s", centralRequest.ID)
	}

	return nil
}

func (d *dataPlaneCentralService) addSecretsToRequest(centralRequest *dbapi.CentralRequest, centralStatus *dbapi.DataPlaneCentralStatus, cluster *api.Cluster) *serviceError.ServiceError {
	if centralStatus.Secrets == nil || len(centralStatus.Secrets) == 0 { // pragma: allowlist secret
		logger.Logger.V(10).Infof("skip persisting secrets for Central %s, report is empty or nil", centralRequest.ID)
		return nil
	}

	if centralStatus.SecretDataSha256Sum == "" {
		// TODO: change this to send a bad request later, once we are sure no FS version without SecretDataSum feature is running
		logger.Logger.V(10).Warningf("persisting secret but no secret data sum. this might be a request of a outdated fleetshard version")
	}

	logger.Logger.Infof("store secret information for central %s", centralRequest.ID)

	if err := centralRequest.SetSecrets(centralStatus.Secrets); err != nil {
		return serviceError.NewWithCause(serviceError.ErrorGeneral, err, "failed to set secrets for central %s", centralRequest.ID)
	}
	centralRequest.SecretDataSha256Sum = centralStatus.SecretDataSha256Sum // pragma: allowlist secret

	return nil
}

func validateRouters(routesInRequest []dbapi.DataPlaneCentralRoute, centralRequest *dbapi.CentralRequest, clusterDNS string) error {
	for _, r := range routesInRequest {
		if !strings.HasSuffix(r.Router, clusterDNS) {
			return errors.Errorf("cluster router is not valid. router = %s, expected = %s", r.Router, clusterDNS)
		}
		if !strings.HasSuffix(r.Domain, centralRequest.Host) {
			return errors.Errorf("exposed domain is not valid. domain = %s, expected = %s", r.Domain, centralRequest.Host)
		}
	}
	return nil
}
