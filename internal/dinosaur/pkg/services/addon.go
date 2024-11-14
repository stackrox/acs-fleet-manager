package services

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/golang/glog"
	clustersmgmtv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/api/dbapi"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/gitops"
	"github.com/stackrox/acs-fleet-manager/pkg/api"
	"github.com/stackrox/acs-fleet-manager/pkg/client/ocm"
	ocmImpl "github.com/stackrox/acs-fleet-manager/pkg/client/ocm/impl"
	"github.com/stackrox/acs-fleet-manager/pkg/features"
	"github.com/stackrox/acs-fleet-manager/pkg/metrics"
	"github.com/stackrox/acs-fleet-manager/pkg/shared"
	"golang.org/x/exp/maps"
)

const fleetshardImageTagParameter = "fleetshardSyncImageTag"

// This backoff was introduced to prevent reaching OCM service log limits
// by sending too many addon upgrade requests. The limit is 1000. A backoff
// of 20 minutes would result in 504 requests per week, which leaves some space
// for other openshift components to call the API as well.
const addonUpgradeBackoffSeconds = 1200

type updateAddonStatusMetricFunc func(addonID, clusterName string, status metrics.AddonStatus)

// AddonProvisioner keeps addon installations on the data plane clusters up-to-date
type AddonProvisioner struct {
	ocmClient                   ocm.Client
	customizations              []addonCustomization
	updateAddonStatusMetricFunc updateAddonStatusMetricFunc
	lastStatus                  metrics.AddonStatus
	lastUpgradeRequestTime      time.Time
}

// NewAddonProvisioner creates a new instance of AddonProvisioner
func NewAddonProvisioner(addonConfig *ocmImpl.AddonConfig, baseConfig *ocmImpl.OCMConfig) (*AddonProvisioner, error) {
	addonOCMConfig := *baseConfig

	addonOCMConfig.BaseURL = addonConfig.URL
	addonOCMConfig.ClientID = addonConfig.ClientID
	addonOCMConfig.ClientSecret = addonConfig.ClientSecret // pragma: allowlist secret
	addonOCMConfig.SelfToken = addonConfig.SelfToken

	conn, _, err := ocmImpl.NewOCMConnection(&addonOCMConfig, addonOCMConfig.BaseURL)
	if err != nil {
		return nil, fmt.Errorf("addon service ocm connection: %w", err)
	}
	return &AddonProvisioner{
		ocmClient:                   ocmImpl.NewClient(conn),
		customizations:              initCustomizations(*addonConfig),
		updateAddonStatusMetricFunc: metrics.UpdateClusterAddonStatusMetric,
	}, nil
}

func initCustomizations(config ocmImpl.AddonConfig) []addonCustomization {
	var customizations []addonCustomization

	if config.InheritFleetshardSyncImageTag {
		if config.FleetshardSyncImageTag == "" {
			glog.Error("fleetshard image tag should not be empty when inherit customization is enabled")
		} else {
			customizations = append(customizations, inheritFleetshardImageTag(config.FleetshardSyncImageTag))
		}
	}
	return customizations
}

type addonCustomization func(gitops.AddonConfig) gitops.AddonConfig

// Provision installs, upgrades or uninstalls the addons based on a given config
func (p *AddonProvisioner) Provision(cluster api.Cluster, dataplaneClusterConfig gitops.DataPlaneClusterConfig) error {
	var errs []error

	installedAddons, err := p.getInstalledAddons(cluster)
	if err != nil {
		errs = append(errs, err)
	}

	for _, expectedConfig := range dataplaneClusterConfig.Addons {
		for _, customization := range p.customizations {
			expectedConfig = customization(expectedConfig)
		}
		installedOnCluster, existOnCluster := installedAddons[expectedConfig.ID]
		if existOnCluster {
			delete(installedAddons, expectedConfig.ID) // retained installations are absent in GitOps - we need to uninstall them
		}
		errs = append(errs, p.provisionAddon(dataplaneClusterConfig, expectedConfig, installedOnCluster, existOnCluster))
	}

	for _, installedAddon := range installedAddons {
		// addon is installed on the cluster but not present in gitops config - uninstall it
		errs = append(errs, p.uninstallAddon(cluster.ClusterID, installedAddon.ID))
		p.updateAddonStatusMetric(installedAddon.ID, dataplaneClusterConfig.ClusterName, metrics.AddonHealthy)
	}

	return errors.Join(errs...)
}

func (p *AddonProvisioner) provisionAddon(dataplaneClusterConfig gitops.DataPlaneClusterConfig, expectedConfig gitops.AddonConfig, installedOnCluster dbapi.AddonInstallation, existOnCluster bool) (provisionError error) {
	var errs []error
	clusterID := dataplaneClusterConfig.ClusterID
	installedInOCM, addonErr := p.ocmClient.GetAddonInstallation(clusterID, expectedConfig.ID)
	status := metrics.AddonHealthy

	defer func() {
		provisionError = errors.Join(errs...)
		if provisionError != nil {
			status = metrics.AddonUnhealthy
		}
		p.updateAddonStatusMetric(expectedConfig.ID, dataplaneClusterConfig.ClusterName, status)
	}()

	if addonErr != nil {
		if addonErr.Is404() {
			// addon does not exist, install it
			errs = append(errs, p.installAddon(clusterID, expectedConfig))
			status = metrics.AddonUpgrade
		} else {
			errs = append(errs, fmt.Errorf("failed to get addon %s: %w", expectedConfig.ID, addonErr))
		}
		return
	}
	if updateInProgress(installedInOCM) {
		glog.V(10).Infof("Addon %s is not in a final state: %s, skip until the next worker iteration", installedInOCM.ID(), installedInOCM.State())
		status = metrics.AddonUpgrade
		return
	}
	if expectedConfig.Version == "" {
		addon, err := p.ocmClient.GetAddon(expectedConfig.ID)
		if err != nil {
			errs = append(errs, fmt.Errorf("get addon %s with the latest version: %w", expectedConfig.ID, err))
			return
		}
		expectedConfig.Version = addon.Version().ID()
	}
	if gitOpsConfigDifferent(expectedConfig, installedInOCM) {
		errs = append(errs, p.updateAddon(clusterID, expectedConfig))
		status = metrics.AddonUpgrade
		return
	}
	if !existOnCluster {
		glog.V(10).Infof("Addon %s is not installed on the data plane", installedInOCM.ID())
		status = metrics.AddonUnhealthy
		return
	}
	versionInstalledInOCM, err := p.ocmClient.GetAddonVersion(installedInOCM.ID(), installedInOCM.AddonVersion().ID())
	if err != nil {
		errs = append(errs, fmt.Errorf("get addon version object for addon %s with version %s: %w",
			installedInOCM.ID(), installedInOCM.AddonVersion().ID(), err))
		return
	}
	if clusterInstallationDifferent(installedOnCluster, versionInstalledInOCM) {
		errs = append(errs, p.updateAddon(clusterID, expectedConfig))
		status = metrics.AddonUpgrade
	} else {
		glog.V(10).Infof("Addon %s is already up-to-date", installedOnCluster.ID)
		errs = append(errs, validateUpToDateAddon(installedInOCM, installedOnCluster))
	}
	return
}

func validateUpToDateAddon(ocmInstallation *clustersmgmtv1.AddOnInstallation, dataPlaneInstallation dbapi.AddonInstallation) error {
	var errs []error
	if ocmInstallation.State() == clustersmgmtv1.AddOnInstallationStateFailed {
		// addon is already up-to-date with gitops config and still failed
		errs = append(errs, fmt.Errorf("addon %s is in a failed state", ocmInstallation.ID()))
	}
	if ocmInstallation.AddonVersion().ID() != dataPlaneInstallation.Version {
		errs = append(errs, fmt.Errorf("addon %s version mismatch: ocm - %s, data plane - %s",
			ocmInstallation.ID(), ocmInstallation.AddonVersion().ID(), dataPlaneInstallation.Version))
	}
	if ocmSHA256Sum := convertParametersFromOCMAPI(ocmInstallation.Parameters()).SHA256Sum(); ocmSHA256Sum != dataPlaneInstallation.ParametersSHA256Sum {
		errs = append(errs, fmt.Errorf("addon %s parameters mismatch: ocm sha256sum - %s, data plane sha256sum - %s",
			ocmInstallation.ID(), ocmSHA256Sum, dataPlaneInstallation.ParametersSHA256Sum))
	}
	return errors.Join(errs...)
}

func (p *AddonProvisioner) getInstalledAddons(cluster api.Cluster) (map[string]dbapi.AddonInstallation, error) {
	if !features.AddonAutoUpgrade.Enabled() {
		glog.V(10).Info("Addon auto upgrade feature is disabled, the existing addon installations will NOT be updated")
		return map[string]dbapi.AddonInstallation{}, nil
	}
	if len(cluster.Addons) == 0 {
		return map[string]dbapi.AddonInstallation{}, nil
	}
	var installedAddons []dbapi.AddonInstallation
	if err := json.Unmarshal(cluster.Addons, &installedAddons); err != nil {
		return map[string]dbapi.AddonInstallation{}, fmt.Errorf("unmarshal installed addons: %w", err)
	}
	result := make(map[string]dbapi.AddonInstallation)
	for _, addon := range installedAddons {
		result[addon.ID] = addon
	}
	return result, nil
}

func (p *AddonProvisioner) installAddon(clusterID string, config gitops.AddonConfig) error {
	addonInstallation, err := p.newInstallation(config)
	if err != nil {
		return err
	}
	if err = p.ocmClient.CreateAddonInstallation(clusterID, addonInstallation); err != nil {
		return fmt.Errorf("create addon %s in ocm: %w", config.ID, err)
	}
	glog.V(5).Infof("Addon %s has been installed on the cluster %s", config.ID, clusterID)
	return nil
}

func (p *AddonProvisioner) newInstallation(config gitops.AddonConfig) (*clustersmgmtv1.AddOnInstallation, error) {
	builder := clustersmgmtv1.NewAddOnInstallation().
		ID(config.ID).
		Addon(clustersmgmtv1.NewAddOn().ID(config.ID)).
		Parameters(convertParametersToOCMAPI(config.Parameters))

	if config.Version != "" {
		builder = builder.AddonVersion(clustersmgmtv1.NewAddOnVersion().ID(config.Version))
	}

	installation, err := builder.Build()

	if err != nil {
		return nil, fmt.Errorf("build new addon installation %s: %w", config.ID, err)
	}

	return installation, nil
}

func (p *AddonProvisioner) updateAddon(clusterID string, config gitops.AddonConfig) error {
	if p.backoffUpgradeRequest() {
		glog.V(5).Infof("update addon request backoff for cluster: %s", clusterID)
		return nil
	}

	update, err := p.newInstallation(config)
	if err != nil {
		return err
	}
	p.lastUpgradeRequestTime = time.Now()
	if err := p.ocmClient.UpdateAddonInstallation(clusterID, update); err != nil {
		return fmt.Errorf("update addon %s: %w", update.ID(), err)
	}
	glog.V(5).Infof("Addon %s has been updated on the cluster %s", config.ID, clusterID)
	return nil
}

func (p *AddonProvisioner) backoffUpgradeRequest() bool {
	return p.lastStatus == metrics.AddonUpgrade && time.Since(p.lastUpgradeRequestTime) < time.Second*addonUpgradeBackoffSeconds
}

func (p *AddonProvisioner) uninstallAddon(clusterID string, addonID string) error {
	if err := p.ocmClient.DeleteAddonInstallation(clusterID, addonID); err != nil {
		return fmt.Errorf("uninstall addon %s: %w", addonID, err)
	}
	glog.V(5).Infof("Addon %s has been uninstalled from the cluster %s", addonID, clusterID)
	return nil
}

func (p *AddonProvisioner) updateAddonStatusMetric(addonID string, clusterName string, status metrics.AddonStatus) {
	if p.updateAddonStatusMetricFunc != nil {
		p.updateAddonStatusMetricFunc(addonID, clusterName, status)
	}
}

func isFinalState(state clustersmgmtv1.AddOnInstallationState) bool {
	return state == clustersmgmtv1.AddOnInstallationStateFailed || state == clustersmgmtv1.AddOnInstallationStateReady
}

func updateInProgress(installedInOCM *clustersmgmtv1.AddOnInstallation) bool {
	return !isFinalState(installedInOCM.State())
}

func gitOpsConfigDifferent(expectedConfig gitops.AddonConfig, installedInOCM *clustersmgmtv1.AddOnInstallation) bool {
	return installedInOCM.AddonVersion().ID() != expectedConfig.Version || !maps.Equal(convertParametersFromOCMAPI(installedInOCM.Parameters()), expectedConfig.Parameters)
}

func clusterInstallationDifferent(current dbapi.AddonInstallation, addonVersion *clustersmgmtv1.AddOnVersion) bool {
	return current.SourceImage != addonVersion.SourceImage() || current.PackageImage != addonVersion.PackageImage()
}

func convertParametersToOCMAPI(parameters map[string]string) *clustersmgmtv1.AddOnInstallationParameterListBuilder {
	var values []*clustersmgmtv1.AddOnInstallationParameterBuilder
	for key, value := range parameters {
		values = append(values, clustersmgmtv1.NewAddOnInstallationParameter().ID(key).Value(value))
	}
	return clustersmgmtv1.NewAddOnInstallationParameterList().Items(values...)
}

func convertParametersFromOCMAPI(parameters *clustersmgmtv1.AddOnInstallationParameterList) shared.AddonParameters {
	result := make(map[string]string)
	parameters.Each(func(item *clustersmgmtv1.AddOnInstallationParameter) bool {
		result[item.ID()] = item.Value()
		return true
	})
	return result
}

func inheritFleetshardImageTag(imageTag string) addonCustomization {
	return func(addon gitops.AddonConfig) gitops.AddonConfig {
		if param := addon.Parameters[fleetshardImageTagParameter]; param == "inherit" {
			addon.Parameters[fleetshardImageTagParameter] = imageTag
		}
		return addon
	}
}
