// Package gitops contains the GitOps configuration.
package gitops

import (
	argocd "github.com/stackrox/acs-fleet-manager/pkg/argocd/apis/application/v1alpha1"
	"k8s.io/apimachinery/pkg/util/validation/field"
)

// Config represents the gitops configuration
type Config struct {
	TenantResources TenantResourceConfig `json:"tenantResources"`
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
	errs = append(errs, validateTenantResourcesConfig(field.NewPath("tenantResources"), config.TenantResources)...)
	errs = append(errs, validateApplications(field.NewPath("applications"), config.Applications)...)
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
