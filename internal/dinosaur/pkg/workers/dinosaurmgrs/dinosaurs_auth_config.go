package dinosaurmgrs

import (
	"github.com/golang/glog"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/api/dbapi"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/config"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/services"
	"github.com/stackrox/acs-fleet-manager/pkg/workers"
)

// CentralAuthConfigManager updates CentralRequests with auth configuration.
type CentralAuthConfigManager struct {
	workers.BaseWorker
	centralService services.DinosaurService
	centralConfig  *config.CentralConfig
}

var _ workers.Worker = &CentralAuthConfigManager{}

func NewCentralAuthConfigManager(centralService services.DinosaurService, centralConfig *config.CentralConfig) *CentralAuthConfigManager {
	return &CentralAuthConfigManager{
		BaseWorker: workers.BaseWorker{
			ID:         uuid.New().String(),
			WorkerType: "central_auth_config",
			Reconciler: workers.Reconciler{},
		},
		centralService: centralService,
		centralConfig:  centralConfig,
	}
}

// Start uses base's Start()
func (k *CentralAuthConfigManager) Start() {
	k.StartWorker(k)
}

// Stop uses base's Stop()
func (k *CentralAuthConfigManager) Stop() {
	k.StopWorker(k)
}

// Reconcile fetches all CentralRequests without auth config from the DB and
// updates them.
func (k *CentralAuthConfigManager) Reconcile() []error {
	glog.Infoln("reconciling auth config for Centrals")
	var errs []error

	centralRequests, listErr := k.centralService.ListDinosaursWithoutAuthConfig()
	if listErr != nil {
		errs = append(errs, errors.Wrap(listErr, "failed to list centrals without auth config"))
	} else {
		glog.V(5).Infof("%d central(s) need auth config to be added", len(centralRequests))
	}

	for _, cr := range centralRequests {
		if err := augmentWithAuthConfig(cr, k.centralConfig); err != nil {
			errs = append(errs, err)
		}
	}

	return errs
}

// augmentWithAuthConfig augments provided CentralRequest with auth config
// information. For that, it performs all necessary rituals, if any.
func augmentWithAuthConfig(r *dbapi.CentralRequest, centralConfig *config.CentralConfig) error {
	glog.V(5).Infof("augmenting Central %q with auth config", r.Meta.ID)
	// Auth config can either be:
	//   1) static, i.e., the same for all Centrals,
	//   2) dynamic, i.e., each Central has its own.
	// In case of 1), all necessary information should be provided in
	// CentralConfig. For 2), we need to request a dynamic client from the
	// RHSSO API.

	// If CentralConfig contains auth info, take it and be done.
	if centralConfig.RhSsoClientID != "" && centralConfig.RhSsoIssuer != "" {
		glog.V(7).Infoln("static config found; no dynamic client will be requested from IdP")
		if centralConfig.RhSsoClientSecret == "" {
			glog.Warningf("no client_secret specified for static client_id %q;" +
				" auth configuration is either incorrect or insecure", centralConfig.RhSsoClientID)
		}
		if centralConfig.RhSsoIssuer == "" {
			glog.Errorf("no issuer specified for static client_id %q;" +
				" auth configuration will likely not work properly", centralConfig.RhSsoClientID)
		}

		r.AuthConfig.ClientID = centralConfig.RhSsoClientID
		r.AuthConfig.ClientSecret = centralConfig.RhSsoClientSecret
		r.AuthConfig.Issuer = centralConfig.RhSsoIssuer

		return nil
	}

	// Proceed with the dynamic path.
	glog.V(7).Infoln("no static config found; requesting dynamic client")

	// TODO(alexr): Talk to RHSSO dynamic client API.

	return nil
}
