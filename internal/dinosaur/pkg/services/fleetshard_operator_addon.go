package services

import (
	"github.com/goava/di"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/clusters"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/clusters/types"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/config"
	"github.com/stackrox/acs-fleet-manager/pkg/api"
	"github.com/stackrox/acs-fleet-manager/pkg/client/iam"
	"github.com/stackrox/acs-fleet-manager/pkg/client/ocm"
	"github.com/stackrox/acs-fleet-manager/pkg/errors"
	glog "github.com/stackrox/acs-fleet-manager/pkg/logging"
	"github.com/stackrox/acs-fleet-manager/pkg/server"
	"github.com/stackrox/acs-fleet-manager/pkg/services/sso"
)

// FleetshardOperatorRoleName ...
const (
	FleetshardOperatorRoleName = "fleetshard_operator"

	// parameter names for the fleetshard-operator service account
	fleetshardOperatorParamSSOBaseURL           = "sso-auth-server-url"
	fleetshardOperatorParamServiceAccountID     = "sso-client-id"
	fleetshardOperatorParamServiceAccountSecret = "sso-secret" // pragma: allowlist secret
	// parameter names for the cluster id
	fleetshardOperatorParamClusterID = "cluster-id"
	// parameter names for the control plane url
	fleetshardOperatorParamControlPlaneBaseURL = "control-plane-url"
	// parameter names for fleetshardoperator synchronizer
	fleetshardOperatorParamPollinterval   = "poll-interval"
	fleetshardOperatorParamResyncInterval = "resync-interval"
)

// FleetshardOperatorAddon ...
//
//go:generate moq -out fleetshard_operator_addon_moq.go . FleetshardOperatorAddon
type FleetshardOperatorAddon interface {
	Provision(cluster api.Cluster) (bool, *errors.ServiceError)
	ReconcileParameters(cluster api.Cluster) *errors.ServiceError
	RemoveServiceAccount(cluster api.Cluster) *errors.ServiceError
}

// NewFleetshardOperatorAddon ...
func NewFleetshardOperatorAddon(o fleetshardOperatorAddon) FleetshardOperatorAddon {
	return &o
}

type fleetshardOperatorAddon struct {
	di.Inject
	IAMService       sso.IAMService
	ProviderFactory  clusters.ProviderFactory
	ServerConfig     *server.ServerConfig
	FleetShardConfig *config.FleetshardConfig
	OCMConfig        *ocm.OCMConfig
	IAMConfig        *iam.IAMConfig
}

// Provision ...
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
	glog.Infof("Provision addon %s for cluster %s", fleetshardAddonID, cluster.ClusterID)
	spec := &types.ClusterSpec{
		InternalID:     cluster.ClusterID,
		ExternalID:     cluster.ExternalID,
		Status:         cluster.Status,
		AdditionalInfo: cluster.ClusterSpec,
	}
	ready, err := p.InstallFleetshard(spec, params)
	if err != nil {
		return false, errors.NewWithCause(errors.ErrorGeneral, err, "failed to install addon %s for cluster %s", fleetshardAddonID, cluster.ClusterID)
	}
	return ready, nil
}

// ReconcileParameters ...
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

	glog.Infof("Reconcile parameters for addon %s on cluster %s", fleetshardAddonID, cluster.ClusterID)
	spec := &types.ClusterSpec{
		InternalID:     cluster.ClusterID,
		ExternalID:     cluster.ExternalID,
		Status:         cluster.Status,
		AdditionalInfo: cluster.ClusterSpec,
	}
	if updated, err := p.InstallFleetshard(spec, params); err != nil {
		return errors.NewWithCause(errors.ErrorGeneral, err, "failed to update parameters for addon %s for cluster %s", fleetshardAddonID, cluster.ClusterID)
	} else if updated {
		glog.Infof("Addon parameters for addon %s on cluster %s are updated", fleetshardAddonID, cluster.ClusterID)
		return nil
	} else {
		glog.Infof("Addon parameters for addon %s on cluster %s are not updated", fleetshardAddonID, cluster.ClusterID)
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

func (o *fleetshardOperatorAddon) provisionServiceAccount(clusterID string) (*api.ServiceAccount, *errors.ServiceError) {
	glog.Infof("Provisioning service account for cluster %s", clusterID)
	return o.IAMService.RegisterAcsFleetshardOperatorServiceAccount(clusterID)
}

func (o *fleetshardOperatorAddon) buildAddonParams(serviceAccount *api.ServiceAccount, clusterID string) []types.Parameter {
	p := []types.Parameter{

		{
			ID:    fleetshardOperatorParamSSOBaseURL,
			Value: o.IAMConfig.RedhatSSORealm.ValidIssuerURI,
		},
		{
			ID:    fleetshardOperatorParamServiceAccountID,
			Value: serviceAccount.ClientID,
		},
		{
			ID:    fleetshardOperatorParamServiceAccountSecret,
			Value: serviceAccount.ClientSecret,
		},
		{
			ID:    fleetshardOperatorParamControlPlaneBaseURL,
			Value: o.ServerConfig.PublicHostURL,
		},
		{
			ID:    fleetshardOperatorParamClusterID,
			Value: clusterID,
		},
		{
			ID:    fleetshardOperatorParamPollinterval,
			Value: o.FleetShardConfig.PollInterval,
		},
		{
			ID:    fleetshardOperatorParamResyncInterval,
			Value: o.FleetShardConfig.ResyncInterval,
		},
	}
	return p
}

// RemoveServiceAccount ...
func (o *fleetshardOperatorAddon) RemoveServiceAccount(cluster api.Cluster) *errors.ServiceError {
	glog.Infof("Removing fleetshard-operator service account for cluster %s", cluster.ClusterID)
	return o.IAMService.DeRegisterAcsFleetshardOperatorServiceAccount(cluster.ClusterID)
}
