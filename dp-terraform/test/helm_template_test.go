package test

import (
	"path/filepath"
	"testing"

	"github.com/gruntwork-io/terratest/modules/helm"
	"github.com/gruntwork-io/terratest/modules/k8s"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
)

func TestHelmTemplate_FleetshardSyncDeployment_ServiceAccountTokenAuthType(t *testing.T) {
	t.Parallel()

	helmChartPath, err := filepath.Abs("../helm/rhacs-terraform")
	releaseName := "rhacs-terraform"
	require.NoError(t, err)

	namespaceName := "rhacs"

	options := &helm.Options{
		SetValues: map[string]string{
			"secured-cluster.enabled":          "false",
			"fleetshardSync.managedDB.enabled": "false",
		},
		KubectlOptions: k8s.NewKubectlOptions("", "", namespaceName),
	}

	output := helm.RenderTemplate(t, options, helmChartPath, releaseName, []string{"templates/fleetshard-sync.yaml"})

	var deployment appsv1.Deployment
	helm.UnmarshalK8SYaml(t, output, &deployment)

	container := deployment.Spec.Template.Spec.Containers[0]
	require.Equal(t, "fleetshard-sync", container.Name)

	volumes := deployment.Spec.Template.Spec.Volumes
	require.Equal(t, 2, len(volumes))
	volume := volumes[1]
	require.Equal(t, "fleet-manager-token", volume.Name)

	envVars := container.Env
	require.Equal(t, "SERVICE_ACCOUNT_TOKEN", findEnvVar("AUTH_TYPE", envVars))
	require.Empty(t, findEnvVar("RHSSO_SERVICE_ACCOUNT_CLIENT_ID", envVars))
	require.Empty(t, findEnvVar("RHSSO_SERVICE_ACCOUNT_CLIENT_SECRET", envVars))
	require.Empty(t, findEnvVar("RHSSO_REALM", envVars))
	require.Empty(t, findEnvVar("RHSSO_ENDPOINT", envVars))

	tokenFile := findEnvVar("FLEET_MANAGER_TOKEN_FILE", envVars)
	require.NotEmpty(t, tokenFile)
}

func findEnvVar(name string, envVars []corev1.EnvVar) string {
	for _, envVar := range envVars {
		if envVar.Name == name {
			return envVar.Value
		}
	}
	return ""
}
