package gitops

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"sigs.k8s.io/yaml"
)

func TestValidateGitOpsConfig(t *testing.T) {
	type tc struct {
		name   string
		yaml   string
		assert func(t *testing.T, c *Config, err field.ErrorList)
	}

	tests := []tc{
		{
			name: "valid",
			assert: func(t *testing.T, c *Config, err field.ErrorList) {
				require.Empty(t, err)
			},
			yaml: `
rhacsOperators:
  crdURls:
    - https://raw.githubusercontent.com/stackrox/stackrox/4.1.2/operator/bundle/manifests/platform.stackrox.io_securedclusters.yaml
  operators:
    - image: "quay.io/rhacs-eng/stackrox-operator:4.1.1"
      deploymentName: "stackrox-operator"
      centralLabelSelector: "app.kubernetes.io/name=central"
      securedClusterLabelSelector: "app.kubernetes.io/name=securedCluster"
centrals:
  additionalAuthProviders:
    - instanceId: "id1"
      authProvider:
        name: "good-name"
        groups:
          - key: "a"
            value: "b"
            role: "Admin"
  overrides:
  - instanceIds:
    - id1
    patch: |
      {}`,
		}, {
			name: "invalid auth provider oidc config",
			assert: func(t *testing.T, c *Config, err field.ErrorList) {
				require.Len(t, err, 3)
			},
			yaml: `
centrals:
  additionalAuthProviders:
    - instanceId: "id1"
      authProvider:
        name: "good-name"
        oidc:
          clientSecret: "donttellanyonemysecret!"
        groups:
          - key: "a"
            value: "b"
            role: "Admin"
`,
		}, {
			name: "invalid yaml in patch",
			assert: func(t *testing.T, c *Config, err field.ErrorList) {
				require.Len(t, err, 1)
				assert.Equal(t, field.Invalid(field.NewPath("centrals", "overrides").Index(0).Child("patch"), "foo", "invalid patch: failed to apply override to Central instance: invalid JSON document"), err[0])
			},
			yaml: `
centrals:
  overrides:
  - instanceIds:
    - id1
    patch: foo
`,
		}, {
			name: "patch contains un-mergeable fields",
			assert: func(t *testing.T, c *Config, err field.ErrorList) {
				require.Len(t, err, 1)
				assert.Equal(t, field.Invalid(field.NewPath("centrals", "overrides").Index(0).Child("patch"), "spec: 123\n", "invalid patch: failed to unmarshal Central instance: json: cannot unmarshal number into Go struct field Central.spec of type v1alpha1.CentralSpec"), err[0])
			},
			yaml: `
centrals:
  overrides:
  - instanceIds:
    patch: |
      spec: 123
`,
		}, {
			name: "invalid operator config and central config",
			assert: func(t *testing.T, c *Config, err field.ErrorList) {
				require.Len(t, err, 1)
				assert.Contains(t, err.ToAggregate().Errors()[0].Error(), "invalid patch: failed to apply override to Central instance: invalid JSON document", "central config was not validated")
			},
			yaml: `
centrals:
  overrides:
  - instanceIds:
    - id1
    patch: invalid
`,
		}, {
			name: "invalid memory request",
			assert: func(t *testing.T, c *Config, err field.ErrorList) {
				require.Len(t, err, 1)
				assert.Contains(t, err.ToAggregate().Errors()[0].Error(), "invalid patch: failed to unmarshal Central instance: quantities must match the regular expression", "central config was not validated")
			},
			yaml: `
centrals:
  overrides:
  - instanceIds:
    - id1
    patch: |
      spec:
        central:
          resources:
            requests:
              memory: "a"
`,
		},
		{
			name: "valid cluster config",
			assert: func(t *testing.T, c *Config, err field.ErrorList) {
				require.Empty(t, err)
			},
			yaml: `
dataPlaneClusters:
  - clusterID: 1234567890abcdef1234567890abcdef
    addons:
      - id: acs-fleetshard
`,
		},
		{
			name: "invalid cluster config when no clusterID",
			assert: func(t *testing.T, c *Config, err field.ErrorList) {
				require.Len(t, err, 1)
				assert.Equal(t, field.Required(field.NewPath("dataPlaneClusters").Index(0).Child("clusterID"), "clusterID is required"), err[0])
			},
			yaml: `
dataPlaneClusters:
  - addons:
      - id: acs-fleetshard
`,
		},
		{
			name: "invalid cluster config when no addon ID",
			assert: func(t *testing.T, c *Config, err field.ErrorList) {
				require.Len(t, err, 1)
				assert.Equal(t, field.Required(field.NewPath("dataPlaneClusters").Index(0).Child("addons").Index(0).Child("id"), "id is required"), err[0])
			},
			yaml: `
dataPlaneClusters:
  - clusterID: 1234567890abcdef1234567890abcdef
    addons:
      - version: 0.2.0
`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var g Config
			require.NoError(t, yaml.Unmarshal([]byte(tt.yaml), &g))
			err := ValidateConfig(g)
			tt.assert(t, &g, err)
		})
	}
}

func TestValidateAdditionalAuthProvider(t *testing.T) {
	path := field.NewPath("additionalAuthProvider")
	authProviderPath := path.Child("authProvider")
	err := validateAdditionalAuthProvider(path, AuthProviderAddition{
		AuthProvider: nil,
		InstanceID:   "",
	})
	assert.Len(t, err.ToAggregate().Errors(), 2)
	assert.Equal(t, field.Required(path.Child("instanceId"), "instance ID is required"), err[0])
	assert.Equal(t, field.Required(authProviderPath, "auth provider spec is required"), err[1])

	err = validateAdditionalAuthProvider(path, AuthProviderAddition{
		AuthProvider: nil,
		InstanceID:   "non-nil",
	})
	require.Len(t, err.ToAggregate().Errors(), 1)
	assert.Equal(t, field.Required(authProviderPath, "auth provider spec is required"), err[0])

	err = validateAdditionalAuthProvider(path, AuthProviderAddition{
		AuthProvider: &AuthProvider{},
		InstanceID:   "non-nil",
	})
	require.Len(t, err.ToAggregate().Errors(), 1)
	assert.Equal(t, field.Required(authProviderPath.Child("name"), "name is required"), err[0])

	err = validateAdditionalAuthProvider(path, AuthProviderAddition{
		AuthProvider: &AuthProvider{
			Name: "all too well",
		},
		InstanceID: "non-nil",
	})
	assert.Nil(t, err)

	err = validateAdditionalAuthProvider(path, AuthProviderAddition{
		AuthProvider: &AuthProvider{
			Name: "all too well",
			OIDC: &AuthProviderOIDCConfig{},
		},
		InstanceID: "non-nil",
	})
	require.Len(t, err.ToAggregate().Errors(), 3)
	oidcPath := authProviderPath.Child("oidc")
	assert.Equal(t, field.Required(oidcPath.Child("clientID"), "clientID is required"), err[0])
	assert.Equal(t, field.Required(oidcPath.Child("issuer"), "issuer is required"), err[1])
	assert.Equal(t, field.Required(oidcPath.Child("mode"), "callbackMode is required"), err[2])

	err = validateAdditionalAuthProvider(path, AuthProviderAddition{
		AuthProvider: &AuthProvider{
			Name: "all too well",
			OIDC: &AuthProviderOIDCConfig{
				ClientID: "clientID",
				Issuer:   "issuer",
				Mode:     "post",
			},
		},
		InstanceID: "non-nil",
	})
	assert.Nil(t, err)

	err = validateAdditionalAuthProvider(path, AuthProviderAddition{
		AuthProvider: &AuthProvider{
			Name: "all too well",
			OIDC: &AuthProviderOIDCConfig{
				ClientID: "clientID",
				Issuer:   "issuer",
				Mode:     "post",
			},
			Groups: []AuthProviderGroup{
				{},
			},
		},
		InstanceID: "non-nil",
	})
	require.Len(t, err.ToAggregate().Errors(), 3)
	groupsPath := authProviderPath.Child("groups")
	firstGroupPath := groupsPath.Index(0)
	assert.Equal(t, field.Required(firstGroupPath.Child("role"), "role name is required"), err[0])
	assert.Equal(t, field.Required(firstGroupPath.Child("key"), "key is required"), err[1])
	assert.Equal(t, field.Required(firstGroupPath.Child("value"), "value is required"), err[2])

	err = validateAdditionalAuthProvider(path, AuthProviderAddition{
		AuthProvider: &AuthProvider{
			Name: "all too well",
			OIDC: &AuthProviderOIDCConfig{
				ClientID: "clientID",
				Issuer:   "issuer",
				Mode:     "post",
			},
			Groups: []AuthProviderGroup{
				{
					Role:  "role",
					Key:   "key",
					Value: "value",
				},
			},
		},
		InstanceID: "non-nil",
	})
	assert.Nil(t, err)

	err = validateAdditionalAuthProvider(path, AuthProviderAddition{
		AuthProvider: &AuthProvider{
			Name: "all too well",
			Groups: []AuthProviderGroup{
				{
					Role:  "role",
					Key:   "key",
					Value: "value",
				},
				{
					Role:  "role",
					Key:   "key",
					Value: "value",
				},
			},
		},
		InstanceID: "non-nil",
	})
	secondGroupPath := groupsPath.Index(1)
	assert.Equal(t, field.Duplicate(secondGroupPath, "duplicate group {key value role}"), err[0])

	err = validateAdditionalAuthProvider(path, AuthProviderAddition{
		AuthProvider: &AuthProvider{
			Name: "all too well",
			RequiredAttributes: []AuthProviderRequiredAttribute{
				{},
			},
		},
		InstanceID: "non-nil",
	})
	require.Len(t, err.ToAggregate().Errors(), 2)
	requiredAttributesPath := authProviderPath.Child("requiredAttributes")
	firstAttributePath := requiredAttributesPath.Index(0)
	assert.Equal(t, field.Required(firstAttributePath.Child("key"), "key is required"), err[0])
	assert.Equal(t, field.Required(firstAttributePath.Child("value"), "value is required"), err[1])

	err = validateAdditionalAuthProvider(path, AuthProviderAddition{
		AuthProvider: &AuthProvider{
			Name: "all too well",
			RequiredAttributes: []AuthProviderRequiredAttribute{
				{
					Key:   "key",
					Value: "value",
				},
			},
		},
		InstanceID: "non-nil",
	})
	assert.Nil(t, err)

	err = validateAdditionalAuthProvider(path, AuthProviderAddition{
		AuthProvider: &AuthProvider{
			Name: "all too well",
			ClaimMappings: []AuthProviderClaimMapping{
				{},
			},
		},
		InstanceID: "non-nil",
	})
	require.Len(t, err.ToAggregate().Errors(), 2)
	claimMappingsPath := authProviderPath.Child("claimMappings")
	firstMappingPath := claimMappingsPath.Index(0)
	assert.Equal(t, field.Required(firstMappingPath.Child("path"), "path is required"), err[0])
	assert.Equal(t, field.Required(firstMappingPath.Child("name"), "name is required"), err[1])

	err = validateAdditionalAuthProvider(path, AuthProviderAddition{
		AuthProvider: &AuthProvider{
			Name: "all too well",
			ClaimMappings: []AuthProviderClaimMapping{
				{
					Path: "path",
					Name: "name",
				},
			},
		},
		InstanceID: "non-nil",
	})
	assert.Nil(t, err)
}
