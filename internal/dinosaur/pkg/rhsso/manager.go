package rhsso

import (
	"context"

	"github.com/golang/glog"
	"github.com/pkg/errors"

	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/api/dbapi"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/config"
	"github.com/stackrox/acs-fleet-manager/pkg/client/iam"
	"github.com/stackrox/acs-fleet-manager/pkg/client/redhatsso/api"
	"github.com/stackrox/acs-fleet-manager/pkg/client/redhatsso/dynamicclients"
	"github.com/stackrox/rox/pkg/ternary"
)

// CentralAuthConfigManager updates CentralRequests with auth configuration.
type CentralAuthConfigManager struct {
	centralConfig           *config.CentralConfig
	realmConfig             *iam.IAMRealmConfig
	dynamicClientsAPIClient *api.AcsTenantsApiService
}

// NewCentralAuthConfigManager creates an instance of this worker.
// In case this function fails, fleet-manager will fail on the startup.
func NewCentralAuthConfigManager(iamConfig *iam.IAMConfig, centralConfig *config.CentralConfig) (*CentralAuthConfigManager, error) {
	realmConfig := iamConfig.RedhatSSORealm
	if !centralConfig.HasStaticAuth() && !realmConfig.IsConfigured() {
		return nil, errors.Errorf("failed to create CentralAuthConfigManager: neither static nor dynamic auth configuration was provided")
	}
	dynamicClientsAPI := dynamicclients.NewDynamicClientsAPI(realmConfig)

	return &CentralAuthConfigManager{
		centralConfig:           centralConfig,
		realmConfig:             realmConfig,
		dynamicClientsAPIClient: dynamicClientsAPI,
	}, nil
}

// AddAuthConfig adds the auth config to a central request
func (k *CentralAuthConfigManager) AddAuthConfig(cr *dbapi.CentralRequest) error {
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
		err = AugmentWithDynamicAuthConfig(context.Background(), cr, k.realmConfig, k.dynamicClientsAPIClient)
	}
	if err != nil {
		return errors.Wrap(err, "failed to augment central request with auth config")
	}

	cr.AuthConfig.ClientOrigin = ternary.String(k.centralConfig.HasStaticAuth(),
		dbapi.AuthConfigStaticClientOrigin, dbapi.AuthConfigDynamicClientOrigin)

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
