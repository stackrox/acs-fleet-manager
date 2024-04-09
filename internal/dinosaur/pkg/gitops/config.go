// Package gitops contains the GitOps configuration.
package gitops

import (
	"fmt"

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
	MinimumRole        string                          `json:"minimumRole,omitempty"`
	Groups             []AuthProviderGroup             `json:"groups,omitempty"`
	RequiredAttributes []AuthProviderRequiredAttribute `json:"requiredAttributes,omitempty"`
	ClaimMappings      []AuthProviderClaimMapping      `json:"claimMappings,omitempty"`
	OIDC               *AuthProviderOIDCConfig         `json:"oidc,omitempty"`
}

// AuthProviderRequiredAttribute is representation of storage.AuthProvider_RequiredAttribute that supports transformation from YAML.
type AuthProviderRequiredAttribute struct {
	Key   string `json:"key,omitempty"`
	Value string `json:"value,omitempty"`
}

// AuthProviderClaimMapping represents a single entry in "claim_mappings" field in auth provider proto.
type AuthProviderClaimMapping struct {
	Path string `json:"path,omitempty"`
	Name string `json:"name,omitempty"`
}

// AuthProviderGroup is representation of storage.AuthProviderGroup that supports transformation from YAML.
type AuthProviderGroup struct {
	Key   string `json:"key,omitempty"`
	Value string `json:"value,omitempty"`
	Role  string `json:"role,omitempty"`
}

// AuthProviderOIDCConfig contains config values for OIDC auth provider.
type AuthProviderOIDCConfig struct {
	Issuer string `json:"issuer,omitempty"`
	// Depending on callback mode, different OAuth 2.0 would be preferred.
	// Possible values are: auto, post, query, fragment.
	Mode         string `json:"mode,omitempty"`
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
	errs = append(errs, validateAdditionalAuthProviders(path.Child("additionalAuthProviders"), config.AdditionalAuthProviders)...)
	return errs
}

func validateAdditionalAuthProviders(path *field.Path, providers []AuthProviderAddition) field.ErrorList {
	var errs field.ErrorList
	for i, additionalProvider := range providers {
		errs = append(errs, validateAdditionalAuthProvider(path.Index(i), additionalProvider)...)
	}
	return errs
}

func validateAdditionalAuthProvider(path *field.Path, provider AuthProviderAddition) field.ErrorList {
	var errs field.ErrorList

	if provider.InstanceID == "" {
		errs = append(errs, field.Required(path.Child("instanceId"), "instance ID is required"))
	}
	authProviderPath := path.Child("authProvider")
	errs = append(errs, validateAuthProvider(authProviderPath, provider.AuthProvider)...)

	return errs
}

func validateAuthProvider(path *field.Path, provider *AuthProvider) field.ErrorList {
	var errs field.ErrorList
	if provider == nil {
		errs = append(errs, field.Required(path, "auth provider spec is required"))
		return errs
	}
	errs = append(errs, validateAuthProviderName(path.Child("name"), provider.Name)...)
	errs = append(errs, validateAuthProviderGroups(path.Child("groups"), provider.Groups)...)
	errs = append(errs, validateAuthProviderRequiredAttributes(path.Child("requiredAttributes"), provider.RequiredAttributes)...)
	errs = append(errs, validateAuthProviderClaimMappings(path.Child("claimMappings"), provider.ClaimMappings)...)
	// Empty config means that the config will be copied over from the default auth provider.
	if provider.OIDC != nil {
		errs = append(errs, validateAuthProviderOIDCConfig(path.Child("oidc"), provider.OIDC)...)
	}
	return errs
}

func validateAuthProviderClaimMappings(path *field.Path, claimMappings []AuthProviderClaimMapping) field.ErrorList {
	var errs field.ErrorList
	for i, claimMapping := range claimMappings {
		errs = append(errs, validateAuthProviderClaimMapping(path.Index(i), claimMapping)...)
	}
	return errs
}

func validateAuthProviderRequiredAttributes(path *field.Path, requiredAttributes []AuthProviderRequiredAttribute) field.ErrorList {
	var errs field.ErrorList
	for i, requiredAttribute := range requiredAttributes {
		errs = append(errs, validateAuthProviderRequiredAttribute(path.Index(i), requiredAttribute)...)
	}
	return errs
}

func validateAuthProviderGroups(path *field.Path, groups []AuthProviderGroup) field.ErrorList {
	var errs field.ErrorList
	var seenGroups = make(map[AuthProviderGroup]struct{}, len(groups))
	for i, group := range groups {
		groupPath := path.Index(i)
		if _, ok := seenGroups[group]; ok {
			errs = append(errs, field.Duplicate(groupPath, fmt.Sprintf("duplicate group %v", group)))
			continue
		}
		seenGroups[group] = struct{}{}
		errs = append(errs, validateAuthProviderGroup(groupPath, group)...)
	}
	return errs
}

func validateAuthProviderName(path *field.Path, name string) field.ErrorList {
	var errs field.ErrorList
	if name == "" {
		errs = append(errs, field.Required(path, "name is required"))
	}
	return errs
}

func validateAuthProviderOIDCConfig(path *field.Path, config *AuthProviderOIDCConfig) field.ErrorList {
	var errs field.ErrorList
	if config.ClientID == "" {
		errs = append(errs, field.Required(path.Child("clientID"), "clientID is required"))
	}
	if config.Issuer == "" {
		errs = append(errs, field.Required(path.Child("issuer"), "issuer is required"))
	}
	if config.Mode == "" {
		errs = append(errs, field.Required(path.Child("mode"), "callbackMode is required"))
	}
	return errs
}

func validateAuthProviderClaimMapping(path *field.Path, claimMapping AuthProviderClaimMapping) field.ErrorList {
	var errs field.ErrorList
	if claimMapping.Path == "" {
		errs = append(errs, field.Required(path.Child("path"), "path is required"))
	}
	if claimMapping.Name == "" {
		errs = append(errs, field.Required(path.Child("name"), "name is required"))
	}
	return errs
}

func validateAuthProviderRequiredAttribute(path *field.Path, attribute AuthProviderRequiredAttribute) field.ErrorList {
	var errs field.ErrorList
	if attribute.Key == "" {
		errs = append(errs, field.Required(path.Child("key"), "key is required"))
	}
	if attribute.Value == "" {
		errs = append(errs, field.Required(path.Child("value"), "value is required"))
	}
	return errs
}

func validateAuthProviderGroup(path *field.Path, group AuthProviderGroup) field.ErrorList {
	var errs field.ErrorList
	if group.Role == "" {
		errs = append(errs, field.Required(path.Child("role"), "role name is required"))
	}
	if group.Key == "" {
		errs = append(errs, field.Required(path.Child("key"), "key is required"))
	}
	if group.Value == "" {
		errs = append(errs, field.Required(path.Child("value"), "value is required"))
	}
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
