package services

import (
	"encoding/json"
	"fmt"

	"github.com/golang/glog"
	"github.com/hashicorp/go-multierror"
	clustersmgmtv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/api/dbapi"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/gitops"
	"github.com/stackrox/acs-fleet-manager/pkg/api"
	"github.com/stackrox/acs-fleet-manager/pkg/client/ocm"
	"github.com/stackrox/acs-fleet-manager/pkg/features"
	"github.com/stackrox/acs-fleet-manager/pkg/shared"
	"github.com/stackrox/rox/pkg/utils"
	"golang.org/x/exp/maps"
)

const fleetshardImageTagParameter = "fleetshardSyncImageTag"

// AddonProvisioner keeps addon installations on the data plane clusters up-to-date
type AddonProvisioner struct {
	ocmClient      ocm.Client
	customizations []addonCustomization
}

// NewAddonProvisioner creates a new instance of AddonProvisioner
func NewAddonProvisioner(addonConfig *ocm.AddonConfig, baseConfig *ocm.OCMConfig) *AddonProvisioner {
	addonOCMConfig := *baseConfig

	addonOCMConfig.BaseURL = addonConfig.URL
	addonOCMConfig.ClientID = addonConfig.ClientID
	addonOCMConfig.ClientSecret = addonConfig.ClientSecret // pragma: allowlist secret
	addonOCMConfig.SelfToken = addonConfig.SelfToken

	conn, _, err := ocm.NewOCMConnection(&addonOCMConfig, addonOCMConfig.BaseURL)
	if err != nil {
		utils.Should(err, fmt.Errorf("addon service ocm connection: %w", err))
	}
	return &AddonProvisioner{
		ocmClient:      ocm.NewClient(conn),
		customizations: initCustomizations(*addonConfig),
	}
}

func initCustomizations(config ocm.AddonConfig) []addonCustomization {
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

type updateDecision struct {
	installedInOCM *clustersmgmtv1.AddOnInstallation
	expectedConfig gitops.AddonConfig
	ocmClient      ocm.Client
	multiErr       *multierror.Error
}

type addonCustomization func(gitops.AddonConfig) gitops.AddonConfig

// Provision installs, upgrades or uninstalls the addons based on a given config
func (p *AddonProvisioner) Provision(cluster api.Cluster, addons []gitops.AddonConfig) error {
	var multiErr *multierror.Error
	clusterID := cluster.ClusterID

	updateDecisions := make(map[string]*updateDecision)
	for _, addon := range addons {
		addonInstallation, addonErr := p.ocmClient.GetAddonInstallation(clusterID, addon.ID)
		if addonErr != nil {
			if addonErr.Is404() {
				// addon does not exist, install it
				multiErr = multierror.Append(multiErr, p.installAddon(clusterID, addon))
			} else {
				multiErr = multierror.Append(multiErr, fmt.Errorf("failed to get addon %s: %w", addon.ID, addonErr))
			}
		} else {
			updateDecisions[addonInstallation.ID()] = &updateDecision{
				installedInOCM: addonInstallation,
				expectedConfig: addon,
				ocmClient:      p.ocmClient,
				multiErr:       multiErr,
			}
		}
	}
	installedAddons, err := p.getInstalledAddons(cluster)
	if err != nil {
		multiErr = multierror.Append(multiErr, err)
	}
	for _, installedAddon := range installedAddons {
		decision, exists := updateDecisions[installedAddon.ID]
		if !exists {
			// addon is installed on the cluster but not present in gitops config - uninstall it
			multiErr = multierror.Append(multiErr, p.uninstallAddon(clusterID, installedAddon.ID))
		} else {
			if decision.updateInProgress() {
				glog.V(10).Infof("Addon %s is not in a final state: %s, skip until the next worker iteration",
					decision.installedInOCM.ID(), decision.installedInOCM.State())
			} else if decision.needsUpdate(installedAddon) {
				multiErr = multierror.Append(multiErr, p.updateAddon(clusterID, decision.expectedConfig))
			} else {
				glog.V(10).Infof("Addon %s is already up-to-date", installedAddon.ID)
				multiErr = validateUpToDateAddon(multiErr, decision.installedInOCM, installedAddon)
			}
		}
	}
	return errorOrNil(multiErr)
}

func validateUpToDateAddon(multiErr *multierror.Error, ocmInstallation *clustersmgmtv1.AddOnInstallation, dataPlaneInstallation dbapi.AddonInstallation) *multierror.Error {
	if ocmInstallation.State() == clustersmgmtv1.AddOnInstallationStateFailed {
		// addon is already up-to-date with gitops config and still failed
		multiErr = multierror.Append(multiErr, fmt.Errorf("addon %s is in a failed state", ocmInstallation.ID()))
	}
	if ocmInstallation.AddonVersion().ID() != dataPlaneInstallation.Version {
		multiErr = multierror.Append(multiErr, fmt.Errorf("addon %s version mismatch: ocm - %s, data plane - %s",
			ocmInstallation.ID(), ocmInstallation.AddonVersion().ID(), dataPlaneInstallation.Version))
	}
	if ocmSHA256Sum := convertParametersFromOCMAPI(ocmInstallation.Parameters()).SHA256Sum(); ocmSHA256Sum != dataPlaneInstallation.ParametersSHA256Sum {
		multiErr = multierror.Append(multiErr, fmt.Errorf("addon %s parameters mismatch: ocm sha256sum - %s, data plane sha256sum - %s",
			ocmInstallation.ID(), ocmSHA256Sum, dataPlaneInstallation.ParametersSHA256Sum))
	}
	return multiErr
}

func (p *AddonProvisioner) getInstalledAddons(cluster api.Cluster) ([]dbapi.AddonInstallation, error) {
	if !features.AddonAutoUpgrade.Enabled() {
		glog.V(10).Info("Addon auto upgrade feature is disabled, the existing addon installations will NOT be updated")
		return []dbapi.AddonInstallation{}, nil
	}
	if len(cluster.Addons) == 0 {
		glog.V(10).Info("No addons installed on the cluster, skipping")
		return []dbapi.AddonInstallation{}, nil
	}
	var installedAddons []dbapi.AddonInstallation
	if err := json.Unmarshal(cluster.Addons, &installedAddons); err != nil {
		return []dbapi.AddonInstallation{}, fmt.Errorf("unmarshal installed addons: %w", err)
	}
	return installedAddons, nil
}

func errorOrNil(multiErr *multierror.Error) error {
	if err := multiErr.ErrorOrNil(); err != nil {
		return fmt.Errorf("provision addons: %w", err)
	}
	return nil
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
	for _, customization := range p.customizations {
		config = customization(config)
	}

	installation, err := clustersmgmtv1.NewAddOnInstallation().
		Addon(clustersmgmtv1.NewAddOn().ID(config.ID)).
		AddonVersion(clustersmgmtv1.NewAddOnVersion().ID(config.Version)).
		Parameters(convertParametersToOCMAPI(config.Parameters)).
		Build()

	if err != nil {
		return nil, fmt.Errorf("build new addon installation %s: %w", config.ID, err)
	}

	return installation, nil
}

func (p *AddonProvisioner) updateAddon(clusterID string, config gitops.AddonConfig) error {
	update, err := p.newInstallation(config)
	if err != nil {
		return err
	}
	if err := p.ocmClient.UpdateAddonInstallation(clusterID, update); err != nil {
		return fmt.Errorf("update addon %s: %w", update.ID(), err)
	}
	glog.V(5).Infof("Addon %s has been updated on the cluster %s", config.ID, clusterID)
	return nil
}

func (p *AddonProvisioner) uninstallAddon(clusterID string, addonID string) error {
	if err := p.ocmClient.DeleteAddonInstallation(clusterID, addonID); err != nil {
		return fmt.Errorf("uninstall addon %s: %w", addonID, err)
	}
	glog.V(5).Infof("Addon %s has been uninstalled from the cluster %s", addonID, clusterID)
	return nil
}

func isFinalState(state clustersmgmtv1.AddOnInstallationState) bool {
	return state == clustersmgmtv1.AddOnInstallationStateFailed || state == clustersmgmtv1.AddOnInstallationStateReady
}

func (c *updateDecision) updateInProgress() bool {
	return !isFinalState(c.installedInOCM.State())
}

func (c *updateDecision) needsUpdate(current dbapi.AddonInstallation) bool {
	if c.installedInOCM.AddonVersion().ID() != c.expectedConfig.Version ||
		!maps.Equal(convertParametersFromOCMAPI(c.installedInOCM.Parameters()), c.expectedConfig.Parameters) {
		return true
	}

	addonVersion, err := c.ocmClient.GetAddonVersion(c.expectedConfig.ID, c.expectedConfig.Version)
	if err != nil {
		c.multiErr = multierror.Append(c.multiErr, fmt.Errorf("get addon version object for addon %s with version %s: %w",
			c.expectedConfig.ID, c.expectedConfig.Version, err))
		return false
	}

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
