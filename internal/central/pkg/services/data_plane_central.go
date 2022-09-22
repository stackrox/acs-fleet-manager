package services

import (
	"context"
	"fmt"
	"strings"
	"time"

	constants2 "github.com/stackrox/acs-fleet-manager/internal/central/constants"

	"github.com/golang/glog"
	"github.com/pkg/errors"
	"github.com/stackrox/acs-fleet-manager/internal/central/pkg/api/dbapi"
	"github.com/stackrox/acs-fleet-manager/internal/central/pkg/config"
	"github.com/stackrox/acs-fleet-manager/pkg/api"
	serviceError "github.com/stackrox/acs-fleet-manager/pkg/errors"
	"github.com/stackrox/acs-fleet-manager/pkg/logger"
	"github.com/stackrox/acs-fleet-manager/pkg/metrics"
)

type centralStatus string

const (
	statusInstalling        centralStatus = "installing"
	statusReady             centralStatus = "ready"
	statusError             centralStatus = "error"
	statusRejected          centralStatus = "rejected"
	statusDeleted           centralStatus = "deleted"
	statusUnknown           centralStatus = "unknown"
	centralOperatorUpdating string        = "DinosaurOperatorUpdating"
	centralUpdating         string        = "DinosaurUpdating"
)

// DataPlaneCentralService ...
type DataPlaneCentralService interface {
	UpdateDataPlaneCentralService(ctx context.Context, clusterID string, status []*dbapi.DataPlaneCentralStatus) *serviceError.ServiceError
}

type dataPlaneCentralService struct {
	centralService CentralService
	clusterService ClusterService
	centralConfig  *config.CentralConfig
}

// NewDataPlaneCentralService ...
func NewDataPlaneCentralService(centralSrv CentralService, clusterSrv ClusterService, centralConfig *config.CentralConfig) *dataPlaneCentralService {
	return &dataPlaneCentralService{
		centralService: centralSrv,
		clusterService: clusterSrv,
		centralConfig:  centralConfig,
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
		central, getErr := d.centralService.GetByID(ks.CentralClusterID)
		if getErr != nil {
			glog.Error(errors.Wrapf(getErr, "failed to get central cluster by id %s", ks.CentralClusterID))
			continue
		}
		if central.ClusterID != clusterID {
			log.Warningf("clusterId for central cluster %s does not match clusterId. central clusterId = %s :: clusterId = %s", central.ID, central.ClusterID, clusterID)
			continue
		}
		var e *serviceError.ServiceError
		switch s := getStatus(ks); s {
		case statusReady:
			// Only store the routes (and create them) when the centrals are ready, as by the time they are ready,
			// the routes should definitely be there.
			e = d.persistCentralRoutes(central, ks, cluster)
			if e == nil {
				e = d.setCentralClusterReady(central)
			}
		case statusError:
			// when getStatus returns statusError we know that the ready
			// condition will be there so there's no need to check for it
			readyCondition, _ := ks.GetReadyCondition()
			e = d.setCentralClusterFailed(central, readyCondition.Message)
		case statusDeleted:
			e = d.setCentralClusterDeleting(central)
		case statusRejected:
			e = d.reassignCentralCluster(central)
		case statusUnknown:
			log.Infof("central cluster %s status is unknown", ks.CentralClusterID)
		default:
			log.V(5).Infof("central cluster %s is still installing", ks.CentralClusterID)
		}
		if e != nil {
			log.Error(errors.Wrapf(e, "Error updating central %s status", ks.CentralClusterID))
		}

		e = d.setCentralRequestVersionFields(central, ks)
		if e != nil {
			log.Error(errors.Wrapf(e, "Error updating central '%s' version fields", ks.CentralClusterID))
		}
	}

	return nil
}

func (d *dataPlaneCentralService) setCentralClusterReady(central *dbapi.CentralRequest) *serviceError.ServiceError {
	if !central.RoutesCreated {
		logger.Logger.V(10).Infof("routes for central %s are not created", central.ID)
		return nil
	}
	logger.Logger.Infof("routes for central %s are created", central.ID)

	// only send metrics data if the current central request is in "provisioning" status as this is the only case we want to report
	shouldSendMetric, err := d.checkCentralRequestCurrentStatus(central, constants2.CentralRequestStatusProvisioning)
	if err != nil {
		return err
	}

	err = d.centralService.Updates(central, map[string]interface{}{"failed_reason": "", "status": constants2.CentralRequestStatusReady.String()})
	if err != nil {
		return serviceError.NewWithCause(err.Code, err, "failed to update status %s for central cluster %s", constants2.CentralRequestStatusReady, central.ID)
	}
	if shouldSendMetric {
		metrics.UpdateCentralRequestsStatusSinceCreatedMetric(constants2.CentralRequestStatusReady, central.ID, central.ClusterID, time.Since(central.CreatedAt))
		metrics.UpdateCentralCreationDurationMetric(metrics.JobTypeCentralCreate, time.Since(central.CreatedAt))
		metrics.IncreaseCentralSuccessOperationsCountMetric(constants2.CentralOperationCreate)
		metrics.IncreaseCentralTotalOperationsCountMetric(constants2.CentralOperationCreate)
	}
	return nil
}

func (d *dataPlaneCentralService) setCentralRequestVersionFields(central *dbapi.CentralRequest, status *dbapi.DataPlaneCentralStatus) *serviceError.ServiceError {
	needsUpdate := false
	prevActualCentralVersion := status.CentralVersion
	if status.CentralVersion != "" && status.CentralVersion != central.ActualCentralVersion {
		logger.Logger.Infof("Updating Central version for Central ID '%s' from '%s' to '%s'", central.ID, prevActualCentralVersion, status.CentralVersion)
		central.ActualCentralVersion = status.CentralVersion
		needsUpdate = true
	}

	prevActualCentralOperatorVersion := status.CentralOperatorVersion
	if status.CentralOperatorVersion != "" && status.CentralOperatorVersion != central.ActualCentralOperatorVersion {
		logger.Logger.Infof("Updating Central operator version for Central ID '%s' from '%s' to '%s'", central.ID, prevActualCentralOperatorVersion, status.CentralOperatorVersion)
		central.ActualCentralOperatorVersion = status.CentralOperatorVersion
		needsUpdate = true
	}

	readyCondition, found := status.GetReadyCondition()
	if found {
		// TODO is this really correct? What happens if there is a CentralOperatorUpdating reason
		// but the 'status' is false? What does that mean and how should we behave?
		prevCentralOperatorUpgrading := central.CentralOperatorUpgrading
		centralOperatorUpdatingReasonIsSet := readyCondition.Reason == centralOperatorUpdating
		if centralOperatorUpdatingReasonIsSet && !prevCentralOperatorUpgrading {
			logger.Logger.Infof("Central operator version for Central ID '%s' upgrade state changed from %t to %t", central.ID, prevCentralOperatorUpgrading, centralOperatorUpdatingReasonIsSet)
			central.CentralOperatorUpgrading = true
			needsUpdate = true
		}
		if !centralOperatorUpdatingReasonIsSet && prevCentralOperatorUpgrading {
			logger.Logger.Infof("Central operator version for Central ID '%s' upgrade state changed from %t to %t", central.ID, prevCentralOperatorUpgrading, centralOperatorUpdatingReasonIsSet)
			central.CentralOperatorUpgrading = false
			needsUpdate = true
		}

		prevCentralUpgrading := central.CentralUpgrading
		centralUpdatingReasonIsSet := readyCondition.Reason == centralUpdating
		if centralUpdatingReasonIsSet && !prevCentralUpgrading {
			logger.Logger.Infof("Central version for Central ID '%s' upgrade state changed from %t to %t", central.ID, prevCentralUpgrading, centralUpdatingReasonIsSet)
			central.CentralUpgrading = true
			needsUpdate = true
		}
		if !centralUpdatingReasonIsSet && prevCentralUpgrading {
			logger.Logger.Infof("Central version for Central ID '%s' upgrade state changed from %t to %t", central.ID, prevCentralUpgrading, centralUpdatingReasonIsSet)
			central.CentralUpgrading = false
			needsUpdate = true
		}

	}

	if needsUpdate {
		versionFields := map[string]interface{}{
			"actual_central_operator_version": central.ActualCentralOperatorVersion,
			"actual_central_version":          central.ActualCentralVersion,
			"central_operator_upgrading":      central.CentralOperatorUpgrading,
			"central_upgrading":               central.CentralUpgrading,
		}

		if err := d.centralService.Updates(central, versionFields); err != nil {
			return serviceError.NewWithCause(err.Code, err, "failed to update actual version fields for central cluster %s", central.ID)
		}
	}

	return nil
}

func (d *dataPlaneCentralService) setCentralClusterFailed(central *dbapi.CentralRequest, errMessage string) *serviceError.ServiceError {
	// if central was already reported as failed we don't do anything
	if central.Status == string(constants2.CentralRequestStatusFailed) {
		return nil
	}

	// only send metrics data if the current central request is in "provisioning" status as this is the only case we want to report
	shouldSendMetric, err := d.checkCentralRequestCurrentStatus(central, constants2.CentralRequestStatusProvisioning)
	if err != nil {
		return err
	}

	central.Status = string(constants2.CentralRequestStatusFailed)
	central.FailedReason = fmt.Sprintf("Central reported as failed: '%s'", errMessage)
	err = d.centralService.Update(central)
	if err != nil {
		return serviceError.NewWithCause(err.Code, err, "failed to update central cluster to %s status for central cluster %s", constants2.CentralRequestStatusFailed, central.ID)
	}
	if shouldSendMetric {
		metrics.UpdateCentralRequestsStatusSinceCreatedMetric(constants2.CentralRequestStatusFailed, central.ID, central.ClusterID, time.Since(central.CreatedAt))
		metrics.IncreaseCentralTotalOperationsCountMetric(constants2.CentralOperationCreate)
	}
	logger.Logger.Errorf("Central status for Central ID '%s' in ClusterID '%s' reported as failed by Fleet Shard Operator: '%s'", central.ID, central.ClusterID, errMessage)

	return nil
}

func (d *dataPlaneCentralService) setCentralClusterDeleting(central *dbapi.CentralRequest) *serviceError.ServiceError {
	// If the central cluster is deleted from the data plane cluster, we will make it as "deleting" in db and the reconcilier will ensure it is cleaned up properly
	if ok, updateErr := d.centralService.UpdateStatus(central.ID, constants2.CentralRequestStatusDeleting); ok {
		if updateErr != nil {
			return serviceError.NewWithCause(updateErr.Code, updateErr, "failed to update status %s for central cluster %s", constants2.CentralRequestStatusDeleting, central.ID)
		}
		metrics.UpdateCentralRequestsStatusSinceCreatedMetric(constants2.CentralRequestStatusDeleting, central.ID, central.ClusterID, time.Since(central.CreatedAt))
	}
	return nil
}

func (d *dataPlaneCentralService) reassignCentralCluster(central *dbapi.CentralRequest) *serviceError.ServiceError {
	if central.Status == constants2.CentralRequestStatusProvisioning.String() {
		// If a central cluster is rejected by the fleetshard-operator, it should be assigned to another OSD cluster (via some scheduler service in the future).
		// But now we only have one OSD cluster, so we need to change the placementId field so that the fleetshard-operator will try it again
		// In the future, we may consider adding a new table to track the placement history for central clusters if there are multiple OSD clusters and the value here can be the key of that table
		central.PlacementID = api.NewID()
		if err := d.centralService.Update(central); err != nil {
			return err
		}
		metrics.UpdateCentralRequestsStatusSinceCreatedMetric(constants2.CentralRequestStatusProvisioning, central.ID, central.ClusterID, time.Since(central.CreatedAt))
	} else {
		logger.Logger.Infof("central cluster %s is rejected and current status is %s", central.ID, central.Status)
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
func (d *dataPlaneCentralService) checkCentralRequestCurrentStatus(central *dbapi.CentralRequest, status constants2.CentralStatus) (bool, *serviceError.ServiceError) {
	matchStatus := false
	if currentInstance, err := d.centralService.GetByID(central.ID); err != nil {
		return matchStatus, err
	} else if currentInstance.Status == status.String() {
		matchStatus = true
	}
	return matchStatus, nil
}

func (d *dataPlaneCentralService) persistCentralRoutes(central *dbapi.CentralRequest, centralStatus *dbapi.DataPlaneCentralStatus, cluster *api.Cluster) *serviceError.ServiceError {
	if central.Routes != nil {
		logger.Logger.V(10).Infof("skip persisting routes for Central %s as they are already stored", central.ID)
		return nil
	}
	logger.Logger.Infof("store routes information for central %s", central.ID)
	clusterDNS, err := d.clusterService.GetClusterDNS(cluster.ClusterID)
	if err != nil {
		return serviceError.NewWithCause(err.Code, err, "failed to get DNS entry for cluster %s", cluster.ClusterID)
	}

	routesInRequest := centralStatus.Routes

	if routesErr := validateRouters(routesInRequest, central, clusterDNS); routesErr != nil {
		return serviceError.NewWithCause(serviceError.ErrorBadRequest, routesErr, "routes are not valid")
	}

	if err := central.SetRoutes(routesInRequest); err != nil {
		return serviceError.NewWithCause(serviceError.ErrorGeneral, err, "failed to set routes for central %s", central.ID)
	}

	if err := d.centralService.Update(central); err != nil {
		return serviceError.NewWithCause(err.Code, err, "failed to update routes for central cluster %s", central.ID)
	}
	return nil
}

func validateRouters(routesInRequest []dbapi.DataPlaneCentralRoute, central *dbapi.CentralRequest, clusterDNS string) error {
	for _, r := range routesInRequest {
		if !strings.HasSuffix(r.Router, clusterDNS) {
			return errors.Errorf("cluster router is not valid. router = %s, expected = %s", r.Router, clusterDNS)
		}
		if !strings.HasSuffix(r.Domain, central.Host) {
			return errors.Errorf("exposed domain is not valid. domain = %s, expected = %s", r.Domain, central.Host)
		}
	}
	return nil
}
