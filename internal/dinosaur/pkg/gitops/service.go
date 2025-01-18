package gitops

import (
	"strings"
	"text/template"

	"helm.sh/helm/v3/pkg/chartutil"

	"github.com/pkg/errors"
	"sigs.k8s.io/yaml"
)

// RenderTenantResourceValues renders the values for tenant resources helm chart for the given GitOps configuration and parameters.
func RenderTenantResourceValues(params CentralParams, config Config) (map[string]interface{}, error) {
	values := map[string]interface{}{}
	if len(config.TenantResources.Default) > 0 {
		renderedDefault, err := renderPatchTemplate(config.TenantResources.Default, params)
		if err != nil {
			return nil, err
		}
		if err := yaml.Unmarshal([]byte(renderedDefault), &values); err != nil {
			return nil, errors.Wrap(err, "failed to unmarshal default tenant resource values")
		}
	}

	for _, override := range config.TenantResources.Overrides {
		if !shouldApplyOverride(override.InstanceIDs, params) {
			continue
		}
		rendered, err := renderPatchTemplate(override.Values, params)
		if err != nil {
			return nil, err
		}
		patchValues := map[string]interface{}{}
		if err := yaml.Unmarshal([]byte(rendered), &patchValues); err != nil {
			return nil, errors.Wrap(err, "failed to unmarshal override patch")
		}
		values = chartutil.CoalesceTables(patchValues, values)
	}
	return values, nil
}

// CentralParams represents the parameters for a Central instance.
type CentralParams struct {
	// ID is the ID of the Central instance.
	ID string
	// Name is the name of the Central instance.
	Name string
	// Namespace is the namespace of the Central instance.
	Namespace string
	// Region is the region of the Central instance.
	Region string
	// ClusterID is the ID of the cluster of the Central instance.
	ClusterID string
	// CloudProvider is the cloud provider of the Central instance.
	CloudProvider string
	// CloudAccountID	is the cloud account ID of the Central instance.
	CloudAccountID string
	// SubscriptionID is the subscription ID of the Central instance.
	SubscriptionID string
	// Owner is the owner of the Central instance.
	Owner string
	// OwnerAccountID is the owner account ID of the Central instance.
	OwnerAccountID string
	// OwnerUserID is the owner user ID of the Central instance.
	OwnerUserID string
	// Host is the host of the Central instance.
	Host string
	// OrganizationID is the organization ID of the Central instance.
	OrganizationID string
	// OrganizationName is the organization name of the Central instance.
	OrganizationName string
	// InstanceType is the instance type of the Central instance.
	InstanceType string
	// IsInternal is true if the Central instance is internal.
	IsInternal bool
}

// shouldApplyOverride returns true if the given Central override should be applied to the given Central instance.
func shouldApplyOverride(instanceIDs []string, ctx CentralParams) bool {
	for _, d := range instanceIDs {
		if d == "*" {
			return true
		}
		if d == ctx.ID {
			return true
		}
	}
	return false
}

func renderPatchTemplate(patchTemplate string, ctx CentralParams) (string, error) {
	tpl, err := template.New("patch").Parse(patchTemplate)
	if err != nil {
		return "", errors.Wrap(err, "failed to parse patch template")
	}
	var writer = new(strings.Builder)
	if err := tpl.Execute(writer, ctx); err != nil {
		return "", errors.Wrap(err, "failed to render patch template")
	}
	return writer.String(), nil
}
