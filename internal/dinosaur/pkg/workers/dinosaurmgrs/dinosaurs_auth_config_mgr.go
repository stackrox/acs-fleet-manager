package dinosaurmgrs

import (
	"context"
	"fmt"

	"github.com/stackrox/acs-fleet-manager/pkg/metrics"

	"github.com/stackrox/rox/pkg/stringutils"

	"github.com/golang/glog"
	"github.com/google/uuid"
	"github.com/pkg/errors"

	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/api/dbapi"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/config"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/services"
	"github.com/stackrox/acs-fleet-manager/pkg/client/iam"
	"github.com/stackrox/acs-fleet-manager/pkg/client/redhatsso/api"
	"github.com/stackrox/acs-fleet-manager/pkg/client/redhatsso/dynamicclients"
	"github.com/stackrox/acs-fleet-manager/pkg/workers"
	"github.com/stackrox/rox/pkg/ternary"
)

const (
	centralAuthConfigManagerWorkerType = "central_auth_config"
	oidcProviderCallbackPath           = "/sso/providers/oidc/callback"
	dynamicClientsNameMaxLength        = 50
)

// CentralAuthConfigManager updates CentralRequests with auth configuration.
type CentralAuthConfigManager struct {
	workers.BaseWorker
	centralService          services.DinosaurService
	centralConfig           *config.CentralConfig
	realmConfig             *iam.IAMRealmConfig
	dynamicClientsAPIClient *api.AcsTenantsApiService
}

var _ workers.Worker = (*CentralAuthConfigManager)(nil)

// NewCentralAuthConfigManager creates an instance of this worker.
// In case this function fails, fleet-manager will fail on the startup.
func NewCentralAuthConfigManager(centralService services.DinosaurService, iamConfig *iam.IAMConfig, centralConfig *config.CentralConfig) (*CentralAuthConfigManager, error) {
	realmConfig := iamConfig.RedhatSSORealm
	if !centralConfig.HasStaticAuth() && !realmConfig.IsConfigured() {
		return nil, errors.Errorf("failed to create %s worker: neither static nor dynamic auth configuration was provided", centralAuthConfigManagerWorkerType)
	}
	dynamicClientsAPI := dynamicclients.NewDynamicClientsAPI(realmConfig)

	metrics.InitReconcilerMetricsForType(centralAuthConfigManagerWorkerType)

	return &CentralAuthConfigManager{
		BaseWorker: workers.BaseWorker{
			ID:         uuid.New().String(),
			WorkerType: centralAuthConfigManagerWorkerType,
			Reconciler: workers.Reconciler{},
		},
		centralService:          centralService,
		centralConfig:           centralConfig,
		realmConfig:             realmConfig,
		dynamicClientsAPIClient: dynamicClientsAPI,
	}, nil
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
	var errs []error

	centralRequests, listErr := k.centralService.ListCentralsWithoutAuthConfig()
	if listErr != nil {
		errs = append(errs, errors.Wrap(listErr, "failed to list centrals without auth config"))
	}
	if len(centralRequests) > 0 {
		glog.V(5).Infof("%d central(s) need auth config to be added", len(centralRequests))
	}

	for _, cr := range centralRequests {
		err := k.reconcileCentralRequest(cr)
		if err != nil {
			errs = append(errs, err)
		}
	}

	return errs
}

func (k *CentralAuthConfigManager) reconcileCentralRequest(cr *dbapi.CentralRequest) error {
	glog.V(5).Infof("augmenting Central %q with auth config", cr.Meta.ID)
	// Auth config can either be:
	//   1) static, i.e., the same for all Centrals,
	//   2) dynamic, i.e., each Central has its own.
	// In case of 1), all necessary information should be provided in
	// CentralConfig. For 2), we need to request a dynamic client from the
	// RHSSO API.

	var err error
	if k.centralConfig.HasStaticAuth() {
		glog.V(7).Infoln("static config found; no dynamic client will be requested the IdP")
		err = augmentWithStaticAuthConfig(cr, k.centralConfig)
	} else {
		glog.V(7).Infoln("no static config found; attempting to obtain one from the IdP")
		err = augmentWithDynamicAuthConfig(cr, k.realmConfig, k.dynamicClientsAPIClient)
	}
	if err != nil {
		return errors.Wrap(err, "failed to augment central request with auth config")
	}

	cr.AuthConfig.ClientOrigin = ternary.String(k.centralConfig.HasStaticAuth(),
		dbapi.AuthConfigStaticClientOrigin, dbapi.AuthConfigDynamicClientOrigin)

	if err := k.centralService.Update(cr); err != nil {
		return errors.Wrapf(err, "failed to update central request %s", cr.ID)
	}

	return nil
}

// augmentWithStaticAuthConfig augments provided CentralRequest with static auth
// config information, i.e., the same for all Centrals.
func augmentWithStaticAuthConfig(r *dbapi.CentralRequest, centralConfig *config.CentralConfig) error {
	r.AuthConfig.ClientID = centralConfig.CentralIDPClientID
	r.AuthConfig.ClientSecret = centralConfig.CentralIDPClientSecret //pragma: allowlist secret
	r.AuthConfig.Issuer = centralConfig.CentralIDPIssuer

	return nil
}

// augmentWithDynamicAuthConfig performs all necessary rituals to obtain auth
// configuration via RHSSO API.
func augmentWithDynamicAuthConfig(r *dbapi.CentralRequest, realmConfig *iam.IAMRealmConfig, apiClient *api.AcsTenantsApiService) error {
	// There is a limit on name length of the dynamic client. To avoid unnecessary errors,
	// we truncate name here.
	name := stringutils.Truncate(fmt.Sprintf("acscs-%s", r.Name), dynamicClientsNameMaxLength)
	orgID := r.OrganisationID
	redirectURIs := []string{fmt.Sprintf("https://%s%s", r.GetUIHost(), oidcProviderCallbackPath)}

	dynamicClientData, _, err := apiClient.CreateAcsClient(context.Background(), api.AcsClientRequestData{
		Name:         name,
		OrgId:        orgID,
		RedirectUris: redirectURIs,
	})
	if err != nil {
		return errors.Wrapf(err, "failed to create RHSSO dynamic client for %s", r.ID)
	}

	r.AuthConfig.ClientID = dynamicClientData.ClientId
	r.AuthConfig.ClientSecret = dynamicClientData.Secret // pragma: allowlist secret
	r.AuthConfig.Issuer = realmConfig.ValidIssuerURI
	return nil
}
