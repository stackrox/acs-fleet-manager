// Package gitops contains the GitOps configuration.
package gitops

import (
	"github.com/stackrox/acs-fleet-manager/fleetshard/pkg/central/operator"
	"k8s.io/apimachinery/pkg/util/validation/field"
)

// Config represents the gitops configuration
type Config struct {
	Centrals          CentralsConfig           `json:"centrals"`
	RHACSOperators    operator.OperatorConfigs `json:"rhacsOperators"`
	DataPlaneClusters []DataPlaneClusterConfig `json:"dataPlaneClusters"`
}

// AuthProviderAddition represents tenant's additional auth provider gitops configuration
type AuthProviderAddition struct {
	InstanceID   string        `json:"instanceId"`
	AuthProvider *AuthProvider `json:"authProvider"`
}

// AuthProvider represents auth provider configuration
type AuthProvider struct {
	Name               string                          `json:"name,omitempty"`
	MinimumRoleName    string                          `json:"minimumRole,omitempty"`
	Groups             []AuthProviderGroup             `json:"groups,omitempty"`
	RequiredAttributes []AuthProviderRequiredAttribute `json:"requiredAttributes,omitempty"`
	ClaimMappings      []AuthProviderClaimMapping      `json:"claimMappings,omitempty"`
	OIDCConfig         *AuthProviderOIDCConfig         `json:"oidc,omitempty"`
}

// AuthProviderRequiredAttribute is representation of storage.AuthProvider_RequiredAttribute that supports transformation from YAML.
type AuthProviderRequiredAttribute struct {
	AttributeKey   string `json:"key,omitempty"`
	AttributeValue string `json:"value,omitempty"`
}

// AuthProviderClaimMapping represents a single entry in "claim_mappings" field in auth provider proto.
type AuthProviderClaimMapping struct {
	Path string `json:"path,omitempty"`
	Name string `json:"name,omitempty"`
}

// AuthProviderGroup is representation of storage.AuthProviderGroup that supports transformation from YAML.
type AuthProviderGroup struct {
	AttributeKey   string `json:"key,omitempty"`
	AttributeValue string `json:"value,omitempty"`
	RoleName       string `json:"role,omitempty"`
}

// AuthProviderOIDCConfig contains config values for OIDC auth provider.
type AuthProviderOIDCConfig struct {
	Issuer string `json:"issuer,omitempty"`
	// Depending on callback mode, different OAuth 2.0 would be preferred.
	// Possible values are: auto, post, query, fragment.
	CallbackMode string `json:"mode,omitempty"`
	ClientID     string `json:"clientID,omitempty"`
	ClientSecret string `json:"clientSecret,omitempty"`
	// Disables request for "offline_access" scope from OIDC identity provider.
	DisableOfflineAccessScope bool `json:"disableOfflineAccessScope,omitempty"`
}

// CentralsConfig represents the declarative configuration for Central instances defaults and overrides.
type CentralsConfig struct {
	// Overrides are the overrides for Central instances.
	Overrides               []CentralOverride      `json:"overrides"`
	AdditionalAuthProviders []AuthProviderAddition `json:"additionalAuthProviders"`
}

// CentralOverride represents the configuration for a Central instance override. The override
// will be applied on top of the default central instance configuration.
// See https://github.com/stackrox/stackrox/blob/master/operator/apis/platform/v1alpha1/overlay_types.go
type CentralOverride struct {
	// InstanceIDs are the Central instance IDs for the override.
	InstanceIDs []string `json:"instanceIds"`
	// Patch is the patch for the override, which will be applied as a strategic merge patch.
	Patch string `json:"patch"`
}

// DataPlaneClusterConfig represents the configuration to be applied for a data plane cluster.
type DataPlaneClusterConfig struct {
	ClusterID string        `json:"clusterID"`
	Addons    []AddonConfig `json:"addons"`
}

// AddonConfig represents the addon configuration to be installed on a cluster
type AddonConfig struct {
	ID         string            `json:"id"`
	Version    string            `json:"version"`
	Parameters map[string]string `json:"parameters"`
}

// ValidateConfig validates the GitOps configuration.
func ValidateConfig(config Config) field.ErrorList {
	var errs field.ErrorList
	errs = append(errs, validateCentralsConfig(field.NewPath("centrals"), config.Centrals)...)
	errs = append(errs, operator.Validate(field.NewPath("rhacsOperators"), config.RHACSOperators)...)
	errs = append(errs, validateDataPlaneClusterConfigs(field.NewPath("dataPlaneClusters"), config.DataPlaneClusters)...)
	return errs
}

func validateCentralsConfig(path *field.Path, config CentralsConfig) field.ErrorList {
	var errs field.ErrorList
	errs = append(errs, validateCentralOverrides(path.Child("overrides"), config.Overrides)...)
	return errs
}

func validateCentralOverrides(path *field.Path, config []CentralOverride) field.ErrorList {
	var errs field.ErrorList
	for i, override := range config {
		errs = append(errs, validateCentralOverride(path.Index(i), override)...)
	}
	return errs
}

func validateCentralOverride(path *field.Path, config CentralOverride) field.ErrorList {
	var errs field.ErrorList
	errs = append(errs, validateInstanceIDs(path.Child("instanceIds"), config.InstanceIDs)...)
	errs = append(errs, validatePatch(path.Child("patch"), config.Patch)...)
	return errs
}

func validatePatch(path *field.Path, patch string) field.ErrorList {
	var errs field.ErrorList
	if len(patch) == 0 {
		errs = append(errs, field.Required(path, "patch is required"))
		return errs
	}
	if err := tryRenderDummyCentralWithPatch(patch); err != nil {
		errs = append(errs, field.Invalid(path, patch, "invalid patch: "+err.Error()))
	}
	return errs
}

// tryRenderDummyCentralWithPatch renders a dummy Central instance with the given patch.
// useful to test that a patch is valid.
func tryRenderDummyCentralWithPatch(patch string) error {
	var dummyParams = getDummyCentralParams()
	dummyConfig := Config{
		Centrals: CentralsConfig{
			Overrides: []CentralOverride{
				{
					Patch:       patch,
					InstanceIDs: []string{"*"},
				},
			},
		},
	}
	if _, err := RenderCentral(dummyParams, dummyConfig); err != nil {
		return err
	}
	return nil
}

func getDummyCentralParams() CentralParams {
	return CentralParams{
		ID:               "id",
		Name:             "name",
		Namespace:        "namespace",
		Region:           "region",
		ClusterID:        "clusterId",
		CloudProvider:    "cloudProvider",
		CloudAccountID:   "cloudAccountId",
		SubscriptionID:   "subscriptionId",
		Owner:            "owner",
		OwnerAccountID:   "ownerAccountId",
		OwnerUserID:      "ownerUserId",
		Host:             "host",
		OrganizationID:   "organizationId",
		OrganizationName: "organizationName",
		InstanceType:     "instanceType",
		IsInternal:       false,
	}
}

func validateInstanceIDs(path *field.Path, instanceIDs []string) field.ErrorList {
	var errs field.ErrorList
	var seenInstanceIDs = make(map[string]struct{})
	for i, instanceID := range instanceIDs {
		if _, ok := seenInstanceIDs[instanceID]; ok {
			errs = append(errs, field.Duplicate(path, instanceID))
		}
		seenInstanceIDs[instanceID] = struct{}{}
		errs = append(errs, validateInstanceID(path.Index(i), instanceID)...)
	}
	return errs
}

func validateInstanceID(path *field.Path, instanceID string) field.ErrorList {
	var errs field.ErrorList
	if len(instanceID) == 0 {
		errs = append(errs, field.Required(path, "instance ID is required"))
	}
	return errs
}

func validateDataPlaneClusterConfigs(path *field.Path, clusters []DataPlaneClusterConfig) field.ErrorList {
	var errs field.ErrorList
	var seenCluster = make(map[string]struct{})
	for i, cluster := range clusters {
		errs = append(errs, validateClusterID(path.Index(i).Child("clusterID"), cluster.ClusterID)...)
		if _, ok := seenCluster[cluster.ClusterID]; ok {
			errs = append(errs, field.Duplicate(path, cluster))
		}
		seenCluster[cluster.ClusterID] = struct{}{}
		errs = append(errs, validateAddons(path.Index(i).Child("addons"), cluster.Addons)...)
	}
	return errs
}

func validateClusterID(path *field.Path, clusterID string) field.ErrorList {
	var errs field.ErrorList
	if len(clusterID) == 0 {
		errs = append(errs, field.Required(path, "clusterID is required"))
	}
	return errs
}

func validateAddons(path *field.Path, addons []AddonConfig) field.ErrorList {
	var errs field.ErrorList
	var seenAddon = make(map[string]struct{})
	for i, addon := range addons {
		errs = append(errs, validateAddonID(path.Index(i).Child("id"), addon.ID)...)
		if _, ok := seenAddon[addon.ID]; ok {
			errs = append(errs, field.Duplicate(path, addon))
		}
		seenAddon[addon.ID] = struct{}{}
	}
	return errs
}

func validateAddonID(path *field.Path, addonID string) field.ErrorList {
	var errs field.ErrorList
	if len(addonID) == 0 {
		errs = append(errs, field.Required(path, "id is required"))
	}
	return errs
}
