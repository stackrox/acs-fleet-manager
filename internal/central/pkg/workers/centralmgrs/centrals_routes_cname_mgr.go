package centralmgrs

import (
	"github.com/golang/glog"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	"github.com/stackrox/acs-fleet-manager/internal/central/pkg/config"
	"github.com/stackrox/acs-fleet-manager/internal/central/pkg/services"
	"github.com/stackrox/acs-fleet-manager/pkg/metrics"
	"github.com/stackrox/acs-fleet-manager/pkg/workers"
)

const centralDNSWorkerType = "central_dns"

// CentralRoutesCNAMEManager ...
type CentralRoutesCNAMEManager struct {
	workers.BaseWorker
	centralService services.CentralService
	centralConfig  *config.CentralConfig
}

var _ workers.Worker = &CentralRoutesCNAMEManager{}

// NewCentralCNAMEManager ...
func NewCentralCNAMEManager(centralService services.CentralService, centralConfig *config.CentralConfig) *CentralRoutesCNAMEManager {
	metrics.InitReconcilerMetricsForType(centralDNSWorkerType)
	return &CentralRoutesCNAMEManager{
		BaseWorker: workers.BaseWorker{
			ID:         uuid.New().String(),
			WorkerType: centralDNSWorkerType,
			Reconciler: workers.Reconciler{},
		},
		centralService: centralService,
		centralConfig:  centralConfig,
	}
}

// Start ...
func (k *CentralRoutesCNAMEManager) Start() {
	k.StartWorker(k)
}

// Stop ...
func (k *CentralRoutesCNAMEManager) Stop() {
	k.StopWorker(k)
}

// Reconcile ...
func (k *CentralRoutesCNAMEManager) Reconcile() []error {
	var errs []error

	centrals, listErr := k.centralService.ListCentralsWithRoutesNotCreated()
	if listErr != nil {
		errs = append(errs, errors.Wrap(listErr, "failed to list centrals whose routes are not created"))
	}
	if len(centrals) > 0 {
		glog.Infof("centrals need routes created count = %d", len(centrals))
	}

	for _, central := range centrals {
		if k.centralConfig.EnableCentralExternalDomain {
			if central.RoutesCreationID == "" {
				glog.Infof("creating CNAME records for central %s", central.ID)

				changeOutput, err := k.centralService.ChangeCentralCNAMErecords(central, services.CentralRoutesActionUpsert)

				if err != nil {
					errs = append(errs, err)
					continue
				}

				switch {
				case changeOutput == nil:
					glog.Infof("creating CNAME records failed with nil result")
					continue
				case changeOutput.ChangeInfo == nil || changeOutput.ChangeInfo.Id == nil || changeOutput.ChangeInfo.Status == nil:
					glog.Infof("creating CNAME records failed with nil info")
					continue
				}

				central.RoutesCreationID = *changeOutput.ChangeInfo.Id
				central.RoutesCreated = *changeOutput.ChangeInfo.Status == "INSYNC"
			} else {
				recordStatus, err := k.centralService.GetCNAMERecordStatus(central)
				if err != nil {
					errs = append(errs, err)
					continue
				}
				central.RoutesCreated = *recordStatus.Status == "INSYNC"
			}
		} else {
			glog.Infof("external certificate is disabled, skip CNAME creation for Central %s", central.ID)
			central.RoutesCreated = true
		}

		if err := k.centralService.UpdateIgnoreNils(central); err != nil {
			errs = append(errs, err)
			continue
		}
	}

	return errs
}
