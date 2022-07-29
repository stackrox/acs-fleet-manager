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

type dinosaurStatus string

const (
	statusInstalling         dinosaurStatus = "installing"
	statusReady              dinosaurStatus = "ready"
	statusError              dinosaurStatus = "error"
	statusRejected           dinosaurStatus = "rejected"
	statusDeleted            dinosaurStatus = "deleted"
	statusUnknown            dinosaurStatus = "unknown"
	dinosaurOperatorUpdating string         = "DinosaurOperatorUpdating"
	dinosaurUpdating         string         = "DinosaurUpdating"
)

// DataPlaneDinosaurService ...
type DataPlaneDinosaurService interface {
	UpdateDataPlaneDinosaurService(ctx context.Context, clusterID string, status []*dbapi.DataPlaneCentralStatus) *serviceError.ServiceError
}

type dataPlaneDinosaurService struct {
	dinosaurService DinosaurService
	clusterService  ClusterService
	dinosaurConfig  *config.CentralConfig
}

// NewDataPlaneDinosaurService ...
func NewDataPlaneDinosaurService(dinosaurSrv DinosaurService, clusterSrv ClusterService, dinosaurConfig *config.CentralConfig) *dataPlaneDinosaurService {
	return &dataPlaneDinosaurService{
		dinosaurService: dinosaurSrv,
		clusterService:  clusterSrv,
		dinosaurConfig:  dinosaurConfig,
	}
}

// UpdateDataPlaneDinosaurService ...
func (d *dataPlaneDinosaurService) UpdateDataPlaneDinosaurService(ctx context.Context, clusterID string, status []*dbapi.DataPlaneCentralStatus) *serviceError.ServiceError {
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
			glog.Error(errors.Wrapf(getErr, "failed to get dinosaur cluster by id %s", ks.CentralClusterID))
			continue
		}
		if dinosaur.ClusterID != clusterID {
			log.Warningf("clusterId for dinosaur cluster %s does not match clusterId. dinosaur clusterId = %s :: clusterId = %s", dinosaur.ID, dinosaur.ClusterID, clusterID)
			continue
		}
		var e *serviceError.ServiceError
		switch s := getStatus(ks); s {
		case statusReady:
			// Only store the routes (and create them) when the Dinosaurs are ready, as by the time they are ready,
			// the routes should definitely be there.
			e = d.persistDinosaurRoutes(dinosaur, ks, cluster)
			if e == nil {
				e = d.setDinosaurClusterReady(dinosaur)
			}
		case statusError:
			// when getStatus returns statusError we know that the ready
			// condition will be there so there's no need to check for it
			readyCondition, _ := ks.GetReadyCondition()
			e = d.setDinosaurClusterFailed(dinosaur, readyCondition.Message)
		case statusDeleted:
			e = d.setDinosaurClusterDeleting(dinosaur)
		case statusRejected:
			e = d.reassignDinosaurCluster(dinosaur)
		case statusUnknown:
			log.Infof("dinosaur cluster %s status is unknown", ks.CentralClusterID)
		default:
			log.V(5).Infof("dinosaur cluster %s is still installing", ks.CentralClusterID)
		}
		if e != nil {
			log.Error(errors.Wrapf(e, "Error updating dinosaur %s status", ks.CentralClusterID))
		}

		e = d.setDinosaurRequestVersionFields(dinosaur, ks)
		if e != nil {
			log.Error(errors.Wrapf(e, "Error updating dinosaur '%s' version fields", ks.CentralClusterID))
		}
	}

	return nil
}

func (d *dataPlaneDinosaurService) setDinosaurClusterReady(dinosaur *dbapi.CentralRequest) *serviceError.ServiceError {
	if !dinosaur.RoutesCreated {
		logger.Logger.V(10).Infof("routes for dinosaur %s are not created", dinosaur.ID)
		return nil
	}
	logger.Logger.Infof("routes for dinosaur %s are created", dinosaur.ID)

	// only send metrics data if the current dinosaur request is in "provisioning" status as this is the only case we want to report
	shouldSendMetric, err := d.checkDinosaurRequestCurrentStatus(dinosaur, constants2.DinosaurRequestStatusProvisioning)
	if err != nil {
		return err
	}

	err = d.dinosaurService.Updates(dinosaur, map[string]interface{}{"failed_reason": "", "status": constants2.DinosaurRequestStatusReady.String()})
	if err != nil {
		return serviceError.NewWithCause(err.Code, err, "failed to update status %s for dinosaur cluster %s", constants2.DinosaurRequestStatusReady, dinosaur.ID)
	}
	if shouldSendMetric {
		metrics.UpdateDinosaurRequestsStatusSinceCreatedMetric(constants2.DinosaurRequestStatusReady, dinosaur.ID, dinosaur.ClusterID, time.Since(dinosaur.CreatedAt))
		metrics.UpdateDinosaurCreationDurationMetric(metrics.JobTypeDinosaurCreate, time.Since(dinosaur.CreatedAt))
		metrics.IncreaseDinosaurSuccessOperationsCountMetric(constants2.DinosaurOperationCreate)
		metrics.IncreaseDinosaurTotalOperationsCountMetric(constants2.DinosaurOperationCreate)
	}
	return nil
}

func (d *dataPlaneDinosaurService) setDinosaurRequestVersionFields(dinosaur *dbapi.CentralRequest, status *dbapi.DataPlaneCentralStatus) *serviceError.ServiceError {
	needsUpdate := false
	prevActualDinosaurVersion := status.CentralVersion
	if status.CentralVersion != "" && status.CentralVersion != dinosaur.ActualCentralVersion {
		logger.Logger.Infof("Updating Dinosaur version for Dinosaur ID '%s' from '%s' to '%s'", dinosaur.ID, prevActualDinosaurVersion, status.CentralVersion)
		dinosaur.ActualCentralVersion = status.CentralVersion
		needsUpdate = true
	}

	prevActualDinosaurOperatorVersion := status.CentralOperatorVersion
	if status.CentralOperatorVersion != "" && status.CentralOperatorVersion != dinosaur.ActualCentralOperatorVersion {
		logger.Logger.Infof("Updating Dinosaur operator version for Dinosaur ID '%s' from '%s' to '%s'", dinosaur.ID, prevActualDinosaurOperatorVersion, status.CentralOperatorVersion)
		dinosaur.ActualCentralOperatorVersion = status.CentralOperatorVersion
		needsUpdate = true
	}

	readyCondition, found := status.GetReadyCondition()
	if found {
		// TODO is this really correct? What happens if there is a DinosaurOperatorUpdating reason
		// but the 'status' is false? What does that mean and how should we behave?
		prevDinosaurOperatorUpgrading := dinosaur.CentralOperatorUpgrading
		dinosaurOperatorUpdatingReasonIsSet := readyCondition.Reason == dinosaurOperatorUpdating
		if dinosaurOperatorUpdatingReasonIsSet && !prevDinosaurOperatorUpgrading {
			logger.Logger.Infof("Dinosaur operator version for Dinosaur ID '%s' upgrade state changed from %t to %t", dinosaur.ID, prevDinosaurOperatorUpgrading, dinosaurOperatorUpdatingReasonIsSet)
			dinosaur.CentralOperatorUpgrading = true
			needsUpdate = true
		}
		if !dinosaurOperatorUpdatingReasonIsSet && prevDinosaurOperatorUpgrading {
			logger.Logger.Infof("Dinosaur operator version for Dinosaur ID '%s' upgrade state changed from %t to %t", dinosaur.ID, prevDinosaurOperatorUpgrading, dinosaurOperatorUpdatingReasonIsSet)
			dinosaur.CentralOperatorUpgrading = false
			needsUpdate = true
		}

		prevDinosaurUpgrading := dinosaur.CentralUpgrading
		dinosaurUpdatingReasonIsSet := readyCondition.Reason == dinosaurUpdating
		if dinosaurUpdatingReasonIsSet && !prevDinosaurUpgrading {
			logger.Logger.Infof("Dinosaur version for Dinosaur ID '%s' upgrade state changed from %t to %t", dinosaur.ID, prevDinosaurUpgrading, dinosaurUpdatingReasonIsSet)
			dinosaur.CentralUpgrading = true
			needsUpdate = true
		}
		if !dinosaurUpdatingReasonIsSet && prevDinosaurUpgrading {
			logger.Logger.Infof("Dinosaur version for Dinosaur ID '%s' upgrade state changed from %t to %t", dinosaur.ID, prevDinosaurUpgrading, dinosaurUpdatingReasonIsSet)
			dinosaur.CentralUpgrading = false
			needsUpdate = true
		}

	}

	if needsUpdate {
		versionFields := map[string]interface{}{
			"actual_central_operator_version": dinosaur.ActualCentralOperatorVersion,
			"actual_central_version":          dinosaur.ActualCentralVersion,
			"central_operator_upgrading":      dinosaur.CentralOperatorUpgrading,
			"central_upgrading":               dinosaur.CentralUpgrading,
		}

		if err := d.dinosaurService.Updates(dinosaur, versionFields); err != nil {
			return serviceError.NewWithCause(err.Code, err, "failed to update actual version fields for dinosaur cluster %s", dinosaur.ID)
		}
	}

	return nil
}

func (d *dataPlaneDinosaurService) setDinosaurClusterFailed(dinosaur *dbapi.CentralRequest, errMessage string) *serviceError.ServiceError {
	// if dinosaur was already reported as failed we don't do anything
	if dinosaur.Status == string(constants2.DinosaurRequestStatusFailed) {
		return nil
	}

	// only send metrics data if the current dinosaur request is in "provisioning" status as this is the only case we want to report
	shouldSendMetric, err := d.checkDinosaurRequestCurrentStatus(dinosaur, constants2.DinosaurRequestStatusProvisioning)
	if err != nil {
		return err
	}

	dinosaur.Status = string(constants2.DinosaurRequestStatusFailed)
	dinosaur.FailedReason = fmt.Sprintf("Dinosaur reported as failed: '%s'", errMessage)
	err = d.dinosaurService.Update(dinosaur)
	if err != nil {
		return serviceError.NewWithCause(err.Code, err, "failed to update dinosaur cluster to %s status for dinosaur cluster %s", constants2.DinosaurRequestStatusFailed, dinosaur.ID)
	}
	if shouldSendMetric {
		metrics.UpdateDinosaurRequestsStatusSinceCreatedMetric(constants2.DinosaurRequestStatusFailed, dinosaur.ID, dinosaur.ClusterID, time.Since(dinosaur.CreatedAt))
		metrics.IncreaseDinosaurTotalOperationsCountMetric(constants2.DinosaurOperationCreate)
	}
	logger.Logger.Errorf("Dinosaur status for Dinosaur ID '%s' in ClusterID '%s' reported as failed by Fleet Shard Operator: '%s'", dinosaur.ID, dinosaur.ClusterID, errMessage)

	return nil
}

func (d *dataPlaneDinosaurService) setDinosaurClusterDeleting(dinosaur *dbapi.CentralRequest) *serviceError.ServiceError {
	// If the Dinosaur cluster is deleted from the data plane cluster, we will make it as "deleting" in db and the reconcilier will ensure it is cleaned up properly
	if ok, updateErr := d.dinosaurService.UpdateStatus(dinosaur.ID, constants2.DinosaurRequestStatusDeleting); ok {
		if updateErr != nil {
			return serviceError.NewWithCause(updateErr.Code, updateErr, "failed to update status %s for dinosaur cluster %s", constants2.DinosaurRequestStatusDeleting, dinosaur.ID)
		}
		metrics.UpdateDinosaurRequestsStatusSinceCreatedMetric(constants2.DinosaurRequestStatusDeleting, dinosaur.ID, dinosaur.ClusterID, time.Since(dinosaur.CreatedAt))
	}
	return nil
}

func (d *dataPlaneDinosaurService) reassignDinosaurCluster(dinosaur *dbapi.CentralRequest) *serviceError.ServiceError {
	if dinosaur.Status == constants2.DinosaurRequestStatusProvisioning.String() {
		// If a Dinosaur cluster is rejected by the fleetshard-operator, it should be assigned to another OSD cluster (via some scheduler service in the future).
		// But now we only have one OSD cluster, so we need to change the placementId field so that the fleetshard-operator will try it again
		// In the future, we may consider adding a new table to track the placement history for dinosaur clusters if there are multiple OSD clusters and the value here can be the key of that table
		dinosaur.PlacementID = api.NewID()
		if err := d.dinosaurService.Update(dinosaur); err != nil {
			return err
		}
		metrics.UpdateDinosaurRequestsStatusSinceCreatedMetric(constants2.DinosaurRequestStatusProvisioning, dinosaur.ID, dinosaur.ClusterID, time.Since(dinosaur.CreatedAt))
	} else {
		logger.Logger.Infof("dinosaur cluster %s is rejected and current status is %s", dinosaur.ID, dinosaur.Status)
	}

	return nil
}

func getStatus(status *dbapi.DataPlaneCentralStatus) dinosaurStatus {
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
func (d *dataPlaneDinosaurService) checkDinosaurRequestCurrentStatus(dinosaur *dbapi.CentralRequest, status constants2.DinosaurStatus) (bool, *serviceError.ServiceError) {
	matchStatus := false
	if currentInstance, err := d.dinosaurService.GetByID(dinosaur.ID); err != nil {
		return matchStatus, err
	} else if currentInstance.Status == status.String() {
		matchStatus = true
	}
	return matchStatus, nil
}

func (d *dataPlaneDinosaurService) persistDinosaurRoutes(dinosaur *dbapi.CentralRequest, dinosaurStatus *dbapi.DataPlaneCentralStatus, cluster *api.Cluster) *serviceError.ServiceError {
	if dinosaur.Routes != nil {
		logger.Logger.V(10).Infof("skip persisting routes for Dinosaur %s as they are already stored", dinosaur.ID)
		return nil
	}
	logger.Logger.Infof("store routes information for dinosaur %s", dinosaur.ID)
	clusterDNS, err := d.clusterService.GetClusterDNS(cluster.ClusterID)
	if err != nil {
		return serviceError.NewWithCause(err.Code, err, "failed to get DNS entry for cluster %s", cluster.ClusterID)
	}

	routesInRequest := dinosaurStatus.Routes
	var routes []dbapi.DataPlaneCentralRoute

	var routesErr error
	baseClusterDomain := strings.TrimPrefix(clusterDNS, fmt.Sprintf("%s.", constants2.DefaultIngressDNSNamePrefix))
	if routes, routesErr = buildRoutes(routesInRequest, dinosaur, baseClusterDomain); routesErr != nil {
		return serviceError.NewWithCause(serviceError.ErrorBadRequest, routesErr, "routes are not valid")
	}

	if err := dinosaur.SetRoutes(routes); err != nil {
		return serviceError.NewWithCause(serviceError.ErrorGeneral, err, "failed to set routes for dinosaur %s", dinosaur.ID)
	}

	if err := d.dinosaurService.Update(dinosaur); err != nil {
		return serviceError.NewWithCause(err.Code, err, "failed to update routes for dinosaur cluster %s", dinosaur.ID)
	}
	return nil
}

func buildRoutes(routesInRequest []dbapi.DataPlaneCentralRouteRequest, dinosaur *dbapi.CentralRequest, clusterDNS string) ([]dbapi.DataPlaneCentralRoute, error) {
	routes := []dbapi.DataPlaneCentralRoute{}
	dinosaurHost := dinosaur.Host
	for _, r := range routesInRequest {
		if strings.HasSuffix(r.Router, clusterDNS) {
			router := dbapi.DataPlaneCentralRoute{
				Router: r.Router,
			}
			if r.Prefix != "" {
				router.Domain = fmt.Sprintf("%s-%s", r.Prefix, dinosaurHost)
			} else {
				router.Domain = dinosaurHost
			}
			routes = append(routes, router)
		} else {
			return nil, errors.Errorf("router domain is not valid. router = %s, expected domain = %s", r.Router, clusterDNS)
		}
	}
	return routes, nil
}
