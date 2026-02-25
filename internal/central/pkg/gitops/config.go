// Package gitops contains the GitOps configuration.
package gitops

import (
	"fmt"

	argocd "github.com/stackrox/acs-fleet-manager/pkg/argocd/apis/application/v1alpha1"
	"k8s.io/apimachinery/pkg/util/validation/field"
)

// Config represents the gitops configuration
type Config struct {
	TenantResources TenantResourceConfig `json:"tenantResources"`
	Centrals        CentralsConfig       `json:"centrals"`
	Applications    []argocd.Application `json:"applications"`
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
	AdditionalAuthProviders []AuthProviderAddition `json:"additionalAuthProviders"`
}

// TenantResourceConfig represents the declarative configuration for tenant resource values defaults and overrides.
type TenantResourceConfig struct {
	Default   string                   `json:"default"`
	Overrides []TenantResourceOverride `json:"overrides"`
}

// TenantResourceOverride represents the configuration for a tenant resource override. The override
// will be applied on top of the default tenant resource values configuration.
type TenantResourceOverride struct {
	InstanceIDs []string `json:"instanceIds"`
	ClusterIDs  []string `json:"clusterIds"`
	Values      string   `json:"values"`
}

// ValidateConfig validates the GitOps configuration.
func ValidateConfig(config Config) field.ErrorList {
	var errs field.ErrorList
	errs = append(errs, validateCentralsConfig(field.NewPath("centrals"), config.Centrals)...)
	errs = append(errs, validateTenantResourcesConfig(field.NewPath("tenantResources"), config.TenantResources)...)
	errs = append(errs, validateApplications(field.NewPath("applications"), config.Applications)...)
	return errs
}

func validateCentralsConfig(path *field.Path, config CentralsConfig) field.ErrorList {
	var errs field.ErrorList
	errs = append(errs, validateAdditionalAuthProviders(path.Child("additionalAuthProviders"), config.AdditionalAuthProviders)...)
	return errs
}

func validateTenantResourcesConfig(path *field.Path, config TenantResourceConfig) field.ErrorList {
	var errs field.ErrorList
	errs = append(errs, validateTenantResourcesDefault(path.Child("default"), config.Default)...)
	errs = append(errs, validateTenantResourceOverrides(path.Child("overrides"), config.Overrides)...)
	return errs
}

func validateTenantResourcesDefault(path *field.Path, defaultValues string) field.ErrorList {
	var errs field.ErrorList
	if err := renderDummyValuesWithPatchForValidation(defaultValues); err != nil {
		errs = append(errs, field.Invalid(path, defaultValues, "invalid default values: "+err.Error()))
	}
	return errs
}

func validateTenantResourceOverrides(path *field.Path, overrides []TenantResourceOverride) field.ErrorList {
	var errs field.ErrorList
	for i, override := range overrides {
		errs = append(errs, validateTenantResourceOverride(path.Index(i), override)...)
	}
	return errs
}

func validateTenantResourceOverride(path *field.Path, override TenantResourceOverride) field.ErrorList {
	var errs field.ErrorList
	errs = append(errs, validateInstanceIDs(path.Child("instanceIds"), override.InstanceIDs)...)
	if err := renderDummyValuesWithPatchForValidation(override.Values); err != nil {
		errs = append(errs, field.Invalid(path.Child("values"), override.Values, "invalid values: "+err.Error()))
	}
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

// renderDummyValuesWithPatchForValidation renders a dummy tenant resource values with the given patch.
// useful to test that a values patch is valid.
func renderDummyValuesWithPatchForValidation(patch string) error {
	var dummyParams = getDummyCentralParams()
	dummyConfig := Config{
		TenantResources: TenantResourceConfig{
			Overrides: []TenantResourceOverride{
				{
					Values:      patch,
					InstanceIDs: []string{"*"},
				},
			},
		},
	}
	if _, err := RenderTenantResourceValues(dummyParams, dummyConfig); err != nil {
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

func validateApplications(path *field.Path, applications []argocd.Application) field.ErrorList {
	var errs field.ErrorList

	seenNames := make(map[string]struct{})

	for i, app := range applications {
		pathIndex := path.Index(i)
		if _, ok := seenNames[app.Name]; ok {
			errs = append(errs, field.Duplicate(path.Child("name"), app.Name))
			continue
		}
		seenNames[app.Name] = struct{}{}
		errs = append(errs, validateApplication(pathIndex, app)...)
	}
	return errs
}

func validateApplication(path *field.Path, app argocd.Application) field.ErrorList {
	var errs field.ErrorList
	if app.Name == "" {
		errs = append(errs, field.Required(path.Child("name"), "name is required"))
	}
	if app.Spec.Source.RepoURL == "" {
		errs = append(errs, field.Required(path.Child("repoURL"), "repoURL is required"))
	}
	if app.Spec.Source.TargetRevision == "" {
		errs = append(errs, field.Required(path.Child("targetRevision"), "targetRevision is required"))
	}
	if app.Spec.Destination.Namespace == "" {
		errs = append(errs, field.Required(path.Child("namespace"), "namespace is required"))
	}
	return errs
}
