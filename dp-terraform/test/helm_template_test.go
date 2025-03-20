package test

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/gruntwork-io/terratest/modules/helm"
	"github.com/gruntwork-io/terratest/modules/k8s"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
)

func TestHelmTemplate_FleetshardSyncDeployment_ServiceAccountTokenAuthType(t *testing.T) {
	t.Parallel()

	deployment := unmarshalFleetshardSyncDeployment(t, map[string]string{
		"secured-cluster.enabled":          "false",
		"fleetshardSync.managedDB.enabled": "false",
		"fleetshardSync.authType":          "SERVICE_ACCOUNT_TOKEN",
	})

	container := deployment.Spec.Template.Spec.Containers[0]
	require.Equal(t, "fleetshard-sync", container.Name)

	volumes := deployment.Spec.Template.Spec.Volumes
	require.Equal(t, 1, len(volumes))
	volume := volumes[0]
	require.Equal(t, "tokens", volume.Name)

	envVars := container.Env
	require.Equal(t, "SERVICE_ACCOUNT_TOKEN", findEnvVar("AUTH_TYPE", envVars).Value)
	require.Empty(t, findEnvVar("RHSSO_SERVICE_ACCOUNT_CLIENT_ID", envVars))
	require.Empty(t, findEnvVar("RHSSO_SERVICE_ACCOUNT_CLIENT_SECRET", envVars))
	require.Empty(t, findEnvVar("RHSSO_REALM", envVars))
	require.Empty(t, findEnvVar("RHSSO_ENDPOINT", envVars))

	tokenFile := findEnvVar("FLEET_MANAGER_TOKEN_FILE", envVars)
	require.NotEmpty(t, tokenFile.Value)
}

func renderTemplate(t *testing.T, values map[string]string, template string) string {
	helmChartPath, err := filepath.Abs("../helm/rhacs-terraform")
	releaseName := "rhacs-terraform"
	require.NoError(t, err)

	namespaceName := "rhacs"

	options := &helm.Options{
		SetValues:      values,
		KubectlOptions: k8s.NewKubectlOptions("", "", namespaceName),
	}

	extraHelmArgs := []string{
		"--api-versions", "external-secrets.io/v1beta1",
	}
	output := helm.RenderTemplate(t, options, helmChartPath, releaseName, []string{template}, extraHelmArgs...)
	return output
}

func unmarshalFleetshardSyncDeployment(t *testing.T, values map[string]string) appsv1.Deployment {
	output := renderTemplate(t, values, "templates/fleetshard-sync.yaml")
	var deployment appsv1.Deployment
	helm.UnmarshalK8SYaml(t, output, &deployment)
	return deployment
}

func findEnvVar(name string, envVars []corev1.EnvVar) *corev1.EnvVar {
	for _, envVar := range envVars {
		if envVar.Name == name {
			return &envVar
		}
	}
	return nil
}

func TestHelmTemplate_FleetshardSyncDeployment_Tenant(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name               string
		secretName         string
		wantNoEnvVar       bool
		expectedSecretName string
		key                string
		expectedKey        string
	}{
		{
			name:         "should not add env var if secret name value is not set",
			wantNoEnvVar: true,
		},
		{
			name:               "should add env var if secret name value is set",
			secretName:         "stackrox", // pragma: allowlist secret
			expectedSecretName: "stackrox", // pragma: allowlist secret
		},
		{
			name:        "should set default key if secret name value is set",
			secretName:  "stackrox", // pragma: allowlist secret
			expectedKey: ".dockerconfigjson",
		},
		{
			name:        "should set key if secret name and key is set",
			secretName:  "stackrox", // pragma: allowlist secret
			key:         "customkey",
			expectedKey: "customkey",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			values := map[string]string{
				"secured-cluster.enabled":          "false",
				"fleetshardSync.managedDB.enabled": "false",
			}

			if tt.secretName != "" {
				values["fleetshardSync.tenantImagePullSecret.name"] = tt.secretName
			}
			if tt.key != "" {
				values["fleetshardSync.tenantImagePullSecret.key"] = tt.key
			}

			deployment := unmarshalFleetshardSyncDeployment(t, values)
			container := deployment.Spec.Template.Spec.Containers[0]
			require.Equal(t, "fleetshard-sync", container.Name)

			envVars := container.Env

			tenantImagePullSecret := findEnvVar("TENANT_IMAGE_PULL_SECRET", envVars)
			if tt.wantNoEnvVar {
				require.Empty(t, tenantImagePullSecret)
				return
			}
			require.NotEmpty(t, tenantImagePullSecret)
			if tt.expectedSecretName != "" {
				require.Equal(t, tt.expectedSecretName, tenantImagePullSecret.ValueFrom.SecretKeyRef.Name)
			}
			if tt.expectedKey != "" {
				require.Equal(t, tt.expectedKey, tenantImagePullSecret.ValueFrom.SecretKeyRef.Key)
			}
		})
	}
}

func TestHelmTemplate_FleetshardSyncDeployment_Image(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		ref       string
		repo      string
		tag       string
		wantImage string
	}{
		{
			name:      "should set default image repo and tag when no values set",
			wantImage: "quay.io/redhat-services-prod/acscs-rhacs-tenant/acscs-main/acs-fleet-manager:latest",
		},
		{
			name:      "should set default image repo when tag is set",
			tag:       "custom",
			wantImage: "quay.io/redhat-services-prod/acscs-rhacs-tenant/acscs-main/acs-fleet-manager:custom",
		},
		{
			name:      "should set image when repo and tag are set",
			repo:      "quay.io/johndoe/my-fleet-manager",
			tag:       "feature1",
			wantImage: "quay.io/johndoe/my-fleet-manager:feature1",
		},
		{
			name:      "should set image when ref is set",
			ref:       "fleet-manager@sha256:12345abcdef",
			wantImage: "fleet-manager@sha256:12345abcdef",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			values := map[string]string{
				"secured-cluster.enabled":          "false",
				"fleetshardSync.managedDB.enabled": "false",
			}

			if tt.repo != "" {
				values["fleetshardSync.image.repo"] = tt.repo
			}
			if tt.tag != "" {
				values["fleetshardSync.image.tag"] = tt.tag
			}
			if tt.ref != "" {
				values["fleetshardSync.image.ref"] = tt.ref
			}

			deployment := unmarshalFleetshardSyncDeployment(t, values)
			container := deployment.Spec.Template.Spec.Containers[0]
			require.Equal(t, "fleetshard-sync", container.Name)
			require.Equal(t, tt.wantImage, container.Image)
		})
	}
}

func TestHelmTemplate_FleetshardSync_ImagePullSecret(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name             string
		pullSecret       string
		createPullSecret string
		wantPullSecret   bool
	}{
		{
			name:             "should not create secret when pull secret is not set and createPullSecret is false",
			pullSecret:       "",
			createPullSecret: "false",
			wantPullSecret:   false,
		},
		{
			name:             "should not create secret when pull secret is set and createPullSecret is false",
			pullSecret:       "quay-image-pull-secret",
			createPullSecret: "false",
			wantPullSecret:   false,
		},
		{
			name:             "should not create secret when pull secret is not set and createPullSecret is true",
			pullSecret:       "",
			createPullSecret: "true",
			wantPullSecret:   false,
		},
		{
			name:             "should create secret when pull secret is set and createPullSecret is true",
			pullSecret:       "quay-image-pull-secret",
			createPullSecret: "true",
			wantPullSecret:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			values := map[string]string{
				"secured-cluster.enabled":                     "false",
				"fleetshardSync.managedDB.enabled":            "false",
				"fleetshardSync.tenantImagePullSecret.create": tt.createPullSecret,
			}
			if tt.pullSecret != "" {
				values["fleetshardSync.tenantImagePullSecret.name"] = tt.pullSecret // pragma: allowlist secret
			}

			output := renderTemplate(t, values, "templates/fleetshard-sync-secret.yaml")
			allRange := strings.Split(output, "---")
			for _, rawOutput := range allRange[1:] {
				var secret corev1.Secret
				helm.UnmarshalK8SYaml(t, rawOutput, &secret)
				if secret.Name == tt.pullSecret {
					require.True(t, tt.wantPullSecret)
					return
				}
			}
			require.False(t, tt.wantPullSecret)
		})
	}
}

func TestHelmTemplate_SecuredCluster_ImagePullSecret(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name             string
		pullSecret       string
		createPullSecret string
		wantPullSecret   bool
	}{
		{
			name:             "should not create secret when pull secret is not set and createPullSecret is false",
			pullSecret:       "",
			createPullSecret: "false",
			wantPullSecret:   false,
		},
		{
			name:             "should not create secret when pull secret is set and createPullSecret is false",
			pullSecret:       "quay-image-pull-secret",
			createPullSecret: "false",
			wantPullSecret:   false,
		},
		{
			name:             "should not create secret when pull secret is not set and createPullSecret is true",
			pullSecret:       "",
			createPullSecret: "true",
			wantPullSecret:   false,
		},
		{
			name:             "should create secret when pull secret is set and createPullSecret is true",
			pullSecret:       "quay-image-pull-secret",
			createPullSecret: "true",
			wantPullSecret:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			values := map[string]string{
				"secured-cluster.clusterName":      "test-cluster",
				"secured-cluster.centralEndpoint":  "https://localhost:8443",
				"secured-cluster.createPullSecret": tt.createPullSecret,
				"fleetshardSync.managedDB.enabled": "false",
			}
			if tt.pullSecret != "" {
				values["secured-cluster.pullSecret"] = tt.pullSecret // pragma: allowlist secret
			}

			output := renderTemplate(t, values, "charts/secured-cluster/templates/secured-cluster-secrets.yaml")
			allRange := strings.Split(output, "---")
			for _, rawOutput := range allRange[1:] {
				var secret corev1.Secret
				helm.UnmarshalK8SYaml(t, rawOutput, &secret)
				if secret.Name == tt.pullSecret {
					require.True(t, tt.wantPullSecret)
					return
				}
			}
			require.False(t, tt.wantPullSecret)
		})
	}
}
