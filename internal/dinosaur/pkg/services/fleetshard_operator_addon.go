package services

import (
	"github.com/goava/di"
	"github.com/golang/glog"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/clusters"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/clusters/types"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/config"
	"github.com/stackrox/acs-fleet-manager/pkg/api"
	"github.com/stackrox/acs-fleet-manager/pkg/client/keycloak"
	"github.com/stackrox/acs-fleet-manager/pkg/client/ocm"
	"github.com/stackrox/acs-fleet-manager/pkg/errors"
	"github.com/stackrox/acs-fleet-manager/pkg/server"
	"github.com/stackrox/acs-fleet-manager/pkg/services/sso"
)

const (
	FleetshardOperatorRoleName = "fleetshard_operator"

	//parameter names for the fleetshard-operator service account
	fleetshardOperatorParamSSOBaseUrl           = "sso-auth-server-url"
	fleetshardOperatorParamServiceAccountId     = "sso-client-id"
	fleetshardOperatorParamServiceAccountSecret = "sso-secret"
	// parameter names for the cluster id
	fleetshardOperatorParamClusterId = "cluster-id"
	// parameter names for the control plane url
	fleetshardOperatorParamControlPlaneBaseURL = "control-plane-url"
	//parameter names for fleetshardoperator synchronizer
	fleetshardOperatorParamPollinterval   = "poll-interval"
	fleetshardOperatorParamResyncInterval = "resync-interval"
)

//go:generate moq -out fleetshard_operator_addon_moq.go . FleetshardOperatorAddon
type FleetshardOperatorAddon interface {
	Provision(cluster api.Cluster) (bool, *errors.ServiceError)
	ReconcileParameters(cluster api.Cluster) *errors.ServiceError
	RemoveServiceAccount(cluster api.Cluster) *errors.ServiceError
}

func NewFleetshardOperatorAddon(o fleetshardOperatorAddon) FleetshardOperatorAddon {
	return &o
}

type fleetshardOperatorAddon struct {
	di.Inject
	SsoService       sso.ACSKeycloakService
	ProviderFactory  clusters.ProviderFactory
	ServerConfig     *server.ServerConfig
	FleetShardConfig *config.FleetshardConfig
	OCMConfig        *ocm.OCMConfig
	KeycloakConfig   *keycloak.KeycloakConfig
}

func (o *fleetshardOperatorAddon) Provision(cluster api.Cluster) (bool, *errors.ServiceError) {
	fleetshardAddonID := o.OCMConfig.FleetshardAddonID
	params, paramsErr := o.getAddonParams(cluster)
	if paramsErr != nil {
		return false, paramsErr
	}
	p, err := o.ProviderFactory.GetProvider(cluster.ProviderType)
	if err != nil {
		return false, errors.NewWithCause(errors.ErrorGeneral, err, "failed to get provider implementation")
	}
	glog.V(5).Infof("Provision addon %s for cluster %s", fleetshardAddonID, cluster.ClusterID)
	spec := &types.ClusterSpec{
		InternalID:     cluster.ClusterID,
		ExternalID:     cluster.ExternalID,
		Status:         cluster.Status,
		AdditionalInfo: cluster.ClusterSpec,
	}
	if ready, err := p.InstallFleetshard(spec, params); err != nil {
		return false, errors.NewWithCause(errors.ErrorGeneral, err, "failed to install addon %s for cluster %s", fleetshardAddonID, cluster.ClusterID)
	} else {
		return ready, nil
	}
}

func (o *fleetshardOperatorAddon) ReconcileParameters(cluster api.Cluster) *errors.ServiceError {
	fleetshardAddonID := o.OCMConfig.FleetshardAddonID
	params, paramsErr := o.getAddonParams(cluster)
	if paramsErr != nil {
		return paramsErr
	}
	p, err := o.ProviderFactory.GetProvider(cluster.ProviderType)
	if err != nil {
		return errors.NewWithCause(errors.ErrorGeneral, err, "failed to get provider implementation")
	}

	glog.V(5).Infof("Reconcile parameters for addon %s on cluster %s", fleetshardAddonID, cluster.ClusterID)
	spec := &types.ClusterSpec{
		InternalID:     cluster.ClusterID,
		ExternalID:     cluster.ExternalID,
		Status:         cluster.Status,
		AdditionalInfo: cluster.ClusterSpec,
	}
	if updated, err := p.InstallFleetshard(spec, params); err != nil {
		return errors.NewWithCause(errors.ErrorGeneral, err, "failed to update parameters for addon %s for cluster %s", fleetshardAddonID, cluster.ClusterID)
	} else if updated {
		glog.V(5).Infof("Addon parameters for addon %s on cluster %s are updated", fleetshardAddonID, cluster.ClusterID)
		return nil
	} else {
		glog.V(5).Infof("Addon parameters for addon %s on cluster %s are not updated", fleetshardAddonID, cluster.ClusterID)
		return nil
	}
}

func (o *fleetshardOperatorAddon) getAddonParams(cluster api.Cluster) ([]types.Parameter, *errors.ServiceError) {
	acc, pErr := o.provisionServiceAccount(cluster.ClusterID)
	if pErr != nil {
		return nil, errors.GeneralError("failed to create service account for cluster %s due to error: %v", cluster.ClusterID, pErr)
	}
	params := o.buildAddonParams(acc, cluster.ClusterID)
	return params, nil
}

func (o *fleetshardOperatorAddon) provisionServiceAccount(clusterId string) (*api.ServiceAccount, *errors.ServiceError) {
	glog.V(5).Infof("Provisioning service account for cluster %s", clusterId)
	return o.SsoService.RegisterAcsFleetshardOperatorServiceAccount(clusterId)
}

func (o *fleetshardOperatorAddon) buildAddonParams(serviceAccount *api.ServiceAccount, clusterId string) []types.Parameter {
	p := []types.Parameter{

		{
			Id:    fleetshardOperatorParamSSOBaseUrl,
			Value: o.KeycloakConfig.RedhatSSORealm.ValidIssuerURI,
		},
		{
			Id:    fleetshardOperatorParamServiceAccountId,
			Value: serviceAccount.ClientID,
		},
		{
			Id:    fleetshardOperatorParamServiceAccountSecret,
			Value: serviceAccount.ClientSecret,
		},
		{
			Id:    fleetshardOperatorParamControlPlaneBaseURL,
			Value: o.ServerConfig.PublicHostURL,
		},
		{
			Id:    fleetshardOperatorParamClusterId,
			Value: clusterId,
		},
		{
			Id:    fleetshardOperatorParamPollinterval,
			Value: o.FleetShardConfig.PollInterval,
		},
		{
			Id:    fleetshardOperatorParamResyncInterval,
			Value: o.FleetShardConfig.ResyncInterval,
		},
	}
	return p
}

func (o *fleetshardOperatorAddon) RemoveServiceAccount(cluster api.Cluster) *errors.ServiceError {
	glog.V(5).Infof("Removing fleetshard-operator service account for cluster %s", cluster.ClusterID)
	return o.SsoService.DeRegisterAcsFleetshardOperatorServiceAccount(cluster.ClusterID)
}
