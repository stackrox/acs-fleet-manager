package operator

import (
	"context"
	"fmt"
	"strings"
	"testing"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/stackrox/acs-fleet-manager/fleetshard/pkg/testutils"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	kindCRDName     = "CustomResourceDefinition"
	k8sAPIVersion   = "apiextensions.k8s.io/v1"
	operatorImage1  = "quay.io/rhacs-eng/stackrox-operator:4.0.1"
	operatorImage2  = "quay.io/rhacs-eng/stackrox-operator:4.0.2"
	deploymentName1 = operatorDeploymentPrefix + "-4-0-1"
	deploymentName2 = operatorDeploymentPrefix + "-4-0-2"
)

func operatorConfigForVersion(version string) OperatorConfig {
	return OperatorConfig{
		"image":          fmt.Sprintf("quay.io/rhacs-eng/stackrox-operator:%s", version),
		"deploymentName": "rhacs-operator-" + strings.ReplaceAll(version, ".", "-"),
	}
}

var operatorConfig1 = operatorConfigForVersion("4.0.1")

var operatorConfig2 = operatorConfigForVersion("4.0.2")

var securedClusterCRD = &unstructured.Unstructured{
	Object: map[string]interface{}{
		"kind":       kindCRDName,
		"apiVersion": k8sAPIVersion,
		"metadata": map[string]interface{}{
			"name": "securedclusters.platform.stackrox.io",
		},
	},
}

var centralCRD = &unstructured.Unstructured{
	Object: map[string]interface{}{
		"kind":       kindCRDName,
		"apiVersion": k8sAPIVersion,
		"metadata": map[string]interface{}{
			"name": "centrals.platform.stackrox.io",
		},
	},
}

var serviceAccount = &unstructured.Unstructured{
	Object: map[string]interface{}{
		"kind":       "ServiceAccount",
		"apiVersion": "v1",
		"metadata": map[string]interface{}{
			"name":      "rhacs-operator-controller-manager",
			"namespace": ACSOperatorNamespace,
		},
	},
}

var operatorDeployment1 = createOperatorDeployment(deploymentName1, operatorImage1)

var operatorDeployment2 = createOperatorDeployment(deploymentName2, operatorImage2)

var metricService = &unstructured.Unstructured{
	Object: map[string]interface{}{
		"kind":       "Service",
		"apiVersion": "v1",
		"metadata": map[string]interface{}{
			"name":      "rhacs-operator-manager-metrics-service",
			"namespace": ACSOperatorNamespace,
		},
	},
}

func createOperatorDeployment(name string, image string) *appsv1.Deployment {
	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: ACSOperatorNamespace,
			Labels:    map[string]string{"app": "rhacs-operator"},
		},
		Spec: appsv1.DeploymentSpec{
			Template: v1.PodTemplateSpec{
				Spec: v1.PodSpec{
					Containers: []v1.Container{{Name: "manager", Image: image}},
				},
			},
		},
	}
}

func getExampleOperatorConfigs(configs ...OperatorConfig) OperatorConfigs {
	return OperatorConfigs{
		CRDURLs: []string{
			"https://raw.githubusercontent.com/stackrox/stackrox/4.1.2/operator/bundle/manifests/platform.stackrox.io_securedclusters.yaml",
			"https://raw.githubusercontent.com/stackrox/stackrox/4.1.2/operator/bundle/manifests/platform.stackrox.io_centrals.yaml",
		},
		Configs: configs,
	}
}

func TestOperatorUpgradeFreshInstall(t *testing.T) {
	fakeClient := testutils.NewFakeClientBuilder(t).Build()
	u := NewACSOperatorManager(fakeClient)

	err := u.InstallOrUpgrade(context.Background(), getExampleOperatorConfigs(operatorConfig1))
	require.NoError(t, err)

	// check Secured Cluster CRD exists and correct
	err = fakeClient.Get(context.Background(), client.ObjectKey{Namespace: ACSOperatorNamespace, Name: securedClusterCRD.GetName()}, securedClusterCRD)
	require.NoError(t, err)
	assert.Equal(t, k8sAPIVersion, securedClusterCRD.GetAPIVersion())
	assert.NotEmpty(t, securedClusterCRD.Object["metadata"])
	assert.NotEmpty(t, securedClusterCRD.Object["spec"])

	// check Central CRD exists and correct
	err = fakeClient.Get(context.Background(), client.ObjectKey{Namespace: ACSOperatorNamespace, Name: centralCRD.GetName()}, centralCRD)
	require.NoError(t, err)
	assert.Equal(t, k8sAPIVersion, centralCRD.GetAPIVersion())
	assert.NotEmpty(t, centralCRD.Object["metadata"])
	assert.NotEmpty(t, centralCRD.Object["spec"])

	// check serviceAccount exists
	err = fakeClient.Get(context.Background(), client.ObjectKey{Namespace: ACSOperatorNamespace, Name: serviceAccount.GetName()}, serviceAccount)
	require.NoError(t, err)
	assert.Equal(t, k8sAPIVersion, centralCRD.GetAPIVersion())
	assert.NotEmpty(t, centralCRD.Object["metadata"])
	assert.NotEmpty(t, centralCRD.Object["spec"])

	// check metric service exists
	err = fakeClient.Get(context.Background(), client.ObjectKey{Namespace: ACSOperatorNamespace, Name: metricService.GetName()}, metricService)
	require.NoError(t, err)
	assert.Equal(t, k8sAPIVersion, centralCRD.GetAPIVersion())
	assert.NotEmpty(t, metricService.Object["metadata"])
	assert.NotEmpty(t, metricService.Object["spec"])

	// check Operator Deployment exists
	err = fakeClient.Get(context.Background(), client.ObjectKey{Namespace: ACSOperatorNamespace, Name: operatorDeployment1.Name}, operatorDeployment1)
	require.NoError(t, err)
	containers := operatorDeployment1.Spec.Template.Spec.Containers
	assert.Len(t, containers, 2)
	managerContainer := containers[1]
	assert.Equal(t, managerContainer.Image, operatorImage1)
}

func TestOperatorUpgradeMultipleVersions(t *testing.T) {
	fakeClient := testutils.NewFakeClientBuilder(t).Build()
	u := NewACSOperatorManager(fakeClient)

	err := u.InstallOrUpgrade(context.Background(), getExampleOperatorConfigs(operatorConfig1, operatorConfig2))
	require.NoError(t, err)

	err = fakeClient.Get(context.Background(), client.ObjectKey{Namespace: ACSOperatorNamespace, Name: operatorDeployment1.Name}, operatorDeployment1)
	require.NoError(t, err)
	managerContainer := operatorDeployment1.Spec.Template.Spec.Containers[1]
	assert.Equal(t, managerContainer.Image, operatorImage1)

	err = fakeClient.Get(context.Background(), client.ObjectKey{Namespace: ACSOperatorNamespace, Name: operatorDeployment2.Name}, operatorDeployment2)
	require.NoError(t, err)
	managerContainer = operatorDeployment2.Spec.Template.Spec.Containers[1]
	assert.Equal(t, managerContainer.Image, operatorImage2)
}

func TestOperatorUpgradeImageWithDigest(t *testing.T) {
	fakeClient := testutils.NewFakeClientBuilder(t).Build()
	u := NewACSOperatorManager(fakeClient)

	digestedImage := "quay.io/rhacs-eng/stackrox-operator:4.0.1@sha256:232a180dbcbcfa7250917507f3827d88a9ae89bb1cdd8fe3ac4db7b764ebb25a"
	operatorConfig := operatorConfigForVersion("4.0.1")
	operatorConfig["image"] = digestedImage

	err := u.InstallOrUpgrade(context.Background(), getExampleOperatorConfigs(operatorConfig))
	require.NoError(t, err)

	err = fakeClient.Get(context.Background(), client.ObjectKey{Namespace: ACSOperatorNamespace, Name: operatorDeployment1.Name}, operatorDeployment1)
	require.NoError(t, err)
	managerContainer := operatorDeployment1.Spec.Template.Spec.Containers[1]
	assert.Equal(t, managerContainer.Image, digestedImage)
}

func TestRemoveUnusedEmpty(t *testing.T) {
	fakeClient := testutils.NewFakeClientBuilder(t).Build()
	u := NewACSOperatorManager(fakeClient)
	ctx := context.Background()

	err := u.RemoveUnusedOperators(ctx, []OperatorConfig{})
	require.NoError(t, err)
}

func TestRemoveOneUnusedOperator(t *testing.T) {
	fakeClient := testutils.NewFakeClientBuilder(t, operatorDeployment1, serviceAccount).Build()
	u := NewACSOperatorManager(fakeClient)
	ctx := context.Background()

	err := fakeClient.Get(context.Background(), client.ObjectKey{Namespace: ACSOperatorNamespace, Name: operatorDeployment1.Name}, operatorDeployment1)
	require.NoError(t, err)
	err = fakeClient.Get(context.Background(), client.ObjectKey{Namespace: ACSOperatorNamespace, Name: serviceAccount.GetName()}, serviceAccount)
	require.NoError(t, err)

	err = u.RemoveUnusedOperators(ctx, []OperatorConfig{{
		"deploymentName": deploymentName2,
		"image":          operatorImage2,
	}})
	require.NoError(t, err)
	// deployment is deleted but service account still persist
	err = fakeClient.Get(context.Background(), client.ObjectKey{Namespace: ACSOperatorNamespace, Name: operatorDeployment1.Name}, operatorDeployment1)
	require.True(t, errors.IsNotFound(err))
	err = fakeClient.Get(context.Background(), client.ObjectKey{Namespace: ACSOperatorNamespace, Name: serviceAccount.GetName()}, serviceAccount)
	require.NoError(t, err)
}

func TestRemoveOneUnusedOperatorFromMany(t *testing.T) {
	fakeClient := testutils.NewFakeClientBuilder(t, operatorDeployment1, operatorDeployment2, serviceAccount).Build()
	u := NewACSOperatorManager(fakeClient)
	ctx := context.Background()

	err := u.RemoveUnusedOperators(ctx, []OperatorConfig{{
		"deploymentName": deploymentName2,
		"image":          operatorImage2,
	}})
	require.NoError(t, err)
	err = fakeClient.Get(context.Background(), client.ObjectKey{Namespace: ACSOperatorNamespace, Name: operatorDeployment1.Name}, operatorDeployment1)
	require.True(t, errors.IsNotFound(err))
	err = fakeClient.Get(context.Background(), client.ObjectKey{Namespace: ACSOperatorNamespace, Name: operatorDeployment2.Name}, operatorDeployment2)
	require.NoError(t, err)
	err = fakeClient.Get(context.Background(), client.ObjectKey{Namespace: ACSOperatorNamespace, Name: serviceAccount.GetName()}, serviceAccount)
	require.NoError(t, err)

	// remove remaining
	err = u.RemoveUnusedOperators(ctx, []OperatorConfig{})
	require.NoError(t, err)
	err = fakeClient.Get(context.Background(), client.ObjectKey{Namespace: ACSOperatorNamespace, Name: operatorDeployment1.Name}, operatorDeployment1)
	require.True(t, errors.IsNotFound(err))
	err = fakeClient.Get(context.Background(), client.ObjectKey{Namespace: ACSOperatorNamespace, Name: operatorDeployment2.Name}, operatorDeployment2)
	require.True(t, errors.IsNotFound(err))
	err = fakeClient.Get(context.Background(), client.ObjectKey{Namespace: ACSOperatorNamespace, Name: serviceAccount.GetName()}, serviceAccount)
	require.NoError(t, err)
}

func TestRemoveMultipleUnusedOperators(t *testing.T) {
	fakeClient := testutils.NewFakeClientBuilder(t, operatorDeployment1, operatorDeployment2, serviceAccount).Build()
	u := NewACSOperatorManager(fakeClient)
	ctx := context.Background()

	err := u.RemoveUnusedOperators(ctx, []OperatorConfig{})
	require.NoError(t, err)
	deployments := &appsv1.DeploymentList{}
	err = fakeClient.List(context.Background(), deployments)
	require.NoError(t, err)
	assert.Len(t, deployments.Items, 0)
	err = fakeClient.Get(context.Background(), client.ObjectKey{Namespace: ACSOperatorNamespace, Name: serviceAccount.GetName()}, serviceAccount)
	require.NoError(t, err)
}
