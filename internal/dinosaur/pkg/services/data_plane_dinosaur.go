package services

import (
	"context"
	"fmt"
	"strings"
	"time"

	constants2 "github.com/stackrox/acs-fleet-manager/internal/dinosaur/constants"

	"github.com/golang/glog"
	"github.com/pkg/errors"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/api/dbapi"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/config"
	"github.com/stackrox/acs-fleet-manager/pkg/api"
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
}

type dataPlaneCentralService struct {
	dinosaurService DinosaurService
	clusterService  ClusterService
	dinosaurConfig  *config.CentralConfig
}

// NewDataPlaneCentralService ...
func NewDataPlaneCentralService(dinosaurSrv DinosaurService, clusterSrv ClusterService, dinosaurConfig *config.CentralConfig) *dataPlaneCentralService {
	return &dataPlaneCentralService{
		dinosaurService: dinosaurSrv,
		clusterService:  clusterSrv,
		dinosaurConfig:  dinosaurConfig,
	}
}

// UpdateDataPlaneCentralService ...
func (d *dataPlaneCentralService) UpdateDataPlaneCentralService(ctx context.Context, clusterID string, status []*dbapi.DataPlaneCentralStatus) *serviceError.ServiceError {
	cluster, err := d.clusterService.FindClusterByID(clusterID)
	log := logger.NewUHCLogger(ctx)
	if err != nil {
		return err
	}
	if cluster == nil {
		// 404 is used for authenticated requests. So to distinguish the errors, we use 400 here
		return serviceError.BadRequest("Cluster id %s not found", clusterID)
	}
	for _, ks := range status {
		dinosaur, getErr := d.dinosaurService.GetByID(ks.CentralClusterID)
		if getErr != nil {
			glog.Error(errors.Wrapf(getErr, "failed to get central cluster by id %s", ks.CentralClusterID))
			continue
		}
		if dinosaur.ClusterID != clusterID {
			log.Warningf("clusterId for central cluster %s does not match clusterId. central clusterId = %s :: clusterId = %s", dinosaur.ID, dinosaur.ClusterID, clusterID)
			continue
		}
		var e *serviceError.ServiceError
		switch s := getStatus(ks); s {
		case statusReady:
			// Persist values only known once central is in statusReady e.g. routes, secrets
			e = d.persistCentralValues(dinosaur, ks, cluster)
			if e == nil {
				e = d.setCentralClusterReady(dinosaur)
			}
		case statusError:
			// when getStatus returns statusError we know that the ready
			// condition will be there so there's no need to check for it
			readyCondition, _ := ks.GetReadyCondition()
			e = d.setCentralClusterFailed(dinosaur, readyCondition.Message)
		case statusDeleted:
			e = d.setCentralClusterDeleting(dinosaur)
		case statusRejected:
			e = d.reassignCentralCluster(dinosaur)
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

func (d *dataPlaneCentralService) setCentralClusterReady(centralRequest *dbapi.CentralRequest) *serviceError.ServiceError {
	if !centralRequest.RoutesCreated {
		logger.Logger.V(10).Infof("routes for central %s are not created", centralRequest.ID)
		return nil
	}
	logger.Logger.Infof("routes for central %s are created", centralRequest.ID)

	// only send metrics data if the current dinosaur request is in "provisioning" status as this is the only case we want to report
	shouldSendMetric, err := d.checkCentralRequestCurrentStatus(centralRequest, constants2.CentralRequestStatusProvisioning)
	if err != nil {
		return err
	}

	err = d.dinosaurService.Updates(centralRequest, map[string]interface{}{"failed_reason": "", "status": constants2.CentralRequestStatusReady.String()})
	if err != nil {
		return serviceError.NewWithCause(err.Code, err, "failed to update status %s for central cluster %s", constants2.CentralRequestStatusReady, centralRequest.ID)
	}
	if shouldSendMetric {
		metrics.UpdateCentralRequestsStatusSinceCreatedMetric(constants2.CentralRequestStatusReady, centralRequest.ID, centralRequest.ClusterID, time.Since(centralRequest.CreatedAt))
		metrics.UpdateCentralCreationDurationMetric(metrics.JobTypeCentralCreate, time.Since(centralRequest.CreatedAt))
		metrics.IncreaseCentralSuccessOperationsCountMetric(constants2.CentralOperationCreate)
		metrics.IncreaseCentralTotalOperationsCountMetric(constants2.CentralOperationCreate)
	}
	return nil
}

func (d *dataPlaneCentralService) setCentralClusterFailed(centralRequest *dbapi.CentralRequest, errMessage string) *serviceError.ServiceError {
	// if dinosaur was already reported as failed we don't do anything
	if centralRequest.Status == string(constants2.CentralRequestStatusFailed) {
		return nil
	}

	// only send metrics data if the current dinosaur request is in "provisioning" status as this is the only case we want to report
	shouldSendMetric, err := d.checkCentralRequestCurrentStatus(centralRequest, constants2.CentralRequestStatusProvisioning)
	if err != nil {
		return err
	}

	centralRequest.Status = string(constants2.CentralRequestStatusFailed)
	centralRequest.FailedReason = fmt.Sprintf("Central reported as failed: '%s'", errMessage)
	err = d.dinosaurService.Update(centralRequest)
	if err != nil {
		return serviceError.NewWithCause(err.Code, err, "failed to update central cluster to %s status for central cluster %s", constants2.CentralRequestStatusFailed, centralRequest.ID)
	}
	if shouldSendMetric {
		metrics.UpdateCentralRequestsStatusSinceCreatedMetric(constants2.CentralRequestStatusFailed, centralRequest.ID, centralRequest.ClusterID, time.Since(centralRequest.CreatedAt))
		metrics.IncreaseCentralTotalOperationsCountMetric(constants2.CentralOperationCreate)
	}
	logger.Logger.Errorf("Central status for Central ID '%s' in ClusterID '%s' reported as failed by Fleet Shard Operator: '%s'", centralRequest.ID, centralRequest.ClusterID, errMessage)

	return nil
}

func (d *dataPlaneCentralService) setCentralClusterDeleting(centralRequest *dbapi.CentralRequest) *serviceError.ServiceError {
	// If the Dinosaur cluster is deleted from the data plane cluster, we will make it as "deleting" in db and the reconcilier will ensure it is cleaned up properly
	if ok, updateErr := d.dinosaurService.UpdateStatus(centralRequest.ID, constants2.CentralRequestStatusDeleting); ok {
		if updateErr != nil {
			return serviceError.NewWithCause(updateErr.Code, updateErr, "failed to update status %s for central cluster %s", constants2.CentralRequestStatusDeleting, centralRequest.ID)
		}
		metrics.UpdateCentralRequestsStatusSinceCreatedMetric(constants2.CentralRequestStatusDeleting, centralRequest.ID, centralRequest.ClusterID, time.Since(centralRequest.CreatedAt))
	}
	return nil
}

func (d *dataPlaneCentralService) reassignCentralCluster(centralRequest *dbapi.CentralRequest) *serviceError.ServiceError {
	if centralRequest.Status == constants2.CentralRequestStatusProvisioning.String() {
		// If a Dinosaur cluster is rejected by the fleetshard-operator, it should be assigned to another OSD cluster (via some scheduler service in the future).
		// But now we only have one OSD cluster, so we need to change the placementId field so that the fleetshard-operator will try it again
		// In the future, we may consider adding a new table to track the placement history for dinosaur clusters if there are multiple OSD clusters and the value here can be the key of that table
		centralRequest.PlacementID = api.NewID()
		if err := d.dinosaurService.Update(centralRequest); err != nil {
			return err
		}
		metrics.UpdateCentralRequestsStatusSinceCreatedMetric(constants2.CentralRequestStatusProvisioning, centralRequest.ID, centralRequest.ClusterID, time.Since(centralRequest.CreatedAt))
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
func (d *dataPlaneCentralService) checkCentralRequestCurrentStatus(centralRequest *dbapi.CentralRequest, status constants2.CentralStatus) (bool, *serviceError.ServiceError) {
	matchStatus := false
	if currentInstance, err := d.dinosaurService.GetByID(centralRequest.ID); err != nil {
		return matchStatus, err
	} else if currentInstance.Status == status.String() {
		matchStatus = true
	}
	return matchStatus, nil
}

func (d *dataPlaneCentralService) persistCentralValues(centralRequest *dbapi.CentralRequest, centralStatus *dbapi.DataPlaneCentralStatus, cluster *api.Cluster) *serviceError.ServiceError {
	if err := d.addRoutesToRequest(centralRequest, centralStatus, cluster); err != nil {
		return err
	}

	if err := d.addSecretsToRequest(centralRequest, centralStatus, cluster); err != nil {
		return err
	}

	if err := d.dinosaurService.Update(centralRequest); err != nil {
		return serviceError.NewWithCause(err.Code, err, "failed to update routes for central cluster %s", centralRequest.ID)
	}

	return nil
}

func (d *dataPlaneCentralService) addRoutesToRequest(centralRequest *dbapi.CentralRequest, centralStatus *dbapi.DataPlaneCentralStatus, cluster *api.Cluster) *serviceError.ServiceError {
	if centralRequest.Routes != nil {
		logger.Logger.V(10).Infof("skip persisting routes for Central %s as they are already stored", centralRequest.ID)
		return nil
	}
	logger.Logger.Infof("store routes information for central %s", centralRequest.ID)
	clusterDNS, err := d.clusterService.GetClusterDNS(cluster.ClusterID)
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
	if centralRequest.Secrets != nil { // pragma: allowlist secret
		logger.Logger.V(10).Infof("skip persisting secrets for Central %s as they are already stored", centralRequest.ID)
		return nil
	}
	logger.Logger.Infof("store secret information for central %s", centralRequest.ID)

	if err := centralRequest.SetSecrets(centralStatus.Secrets); err != nil {
		return serviceError.NewWithCause(serviceError.ErrorGeneral, err, "failed to set secrets for central %s", centralRequest.ID)
	}

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
