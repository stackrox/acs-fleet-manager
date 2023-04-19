package operator

import (
	"context"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/stackrox/acs-fleet-manager/fleetshard/pkg/testutils"
	appsv1 "k8s.io/api/apps/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	kindCRDName    = "CustomResourceDefinition"
	k8sAPIVersion  = "apiextensions.k8s.io/v1"
	operatorImage1 = "quay.io/rhacs-eng/stackrox-operator:3.74.1"
	operatorImage2 = "quay.io/rhacs-eng/stackrox-operator:3.74.2"
)

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
			"namespace": operatorNamespace,
		},
	},
}

var operatorDeployment1 = &appsv1.Deployment{
	ObjectMeta: metav1.ObjectMeta{
		Name:      "rhacs-operator-controller-manager-3.74.1",
		Namespace: operatorNamespace,
	},
}

var operatorDeployment2 = &appsv1.Deployment{
	ObjectMeta: metav1.ObjectMeta{
		Name:      "rhacs-operator-controller-manager-3.74.2",
		Namespace: operatorNamespace,
	},
}

var metricService = &unstructured.Unstructured{
	Object: map[string]interface{}{
		"kind":       "Service",
		"apiVersion": "v1",
		"metadata": map[string]interface{}{
			"name":      "rhacs-operator-controller-manager-metrics-service",
			"namespace": operatorNamespace,
		},
	},
}

func TestOperatorUpgradeFreshInstall(t *testing.T) {
	fakeClient := testutils.NewFakeClientBuilder(t).Build()
	u := NewACSOperatorManager(fakeClient)

	err := u.InstallOrUpgrade(context.Background(), []string{operatorImage1})
	require.NoError(t, err)

	// check Secured Cluster CRD exists and correct
	err = fakeClient.Get(context.Background(), client.ObjectKey{Namespace: "", Name: securedClusterCRD.GetName()}, securedClusterCRD)
	require.NoError(t, err)
	assert.Equal(t, k8sAPIVersion, securedClusterCRD.GetAPIVersion())
	assert.NotEmpty(t, securedClusterCRD.Object["metadata"])
	assert.NotEmpty(t, securedClusterCRD.Object["spec"])

	// check Central CRD exists and correct
	err = fakeClient.Get(context.Background(), client.ObjectKey{Namespace: "", Name: centralCRD.GetName()}, centralCRD)
	require.NoError(t, err)
	assert.Equal(t, k8sAPIVersion, centralCRD.GetAPIVersion())
	assert.NotEmpty(t, centralCRD.Object["metadata"])
	assert.NotEmpty(t, centralCRD.Object["spec"])

	// check serviceAccount exists
	err = fakeClient.Get(context.Background(), client.ObjectKey{Namespace: operatorNamespace, Name: serviceAccount.GetName()}, serviceAccount)
	require.NoError(t, err)
	assert.Equal(t, k8sAPIVersion, centralCRD.GetAPIVersion())
	assert.NotEmpty(t, centralCRD.Object["metadata"])
	assert.NotEmpty(t, centralCRD.Object["spec"])

	// check metric service exists
	err = fakeClient.Get(context.Background(), client.ObjectKey{Namespace: operatorNamespace, Name: metricService.GetName()}, metricService)
	require.NoError(t, err)
	assert.Equal(t, k8sAPIVersion, centralCRD.GetAPIVersion())
	assert.NotEmpty(t, metricService.Object["metadata"])
	assert.NotEmpty(t, metricService.Object["spec"])

	// check Operator Deployment exists
	err = fakeClient.Get(context.Background(), client.ObjectKey{Namespace: operatorNamespace, Name: operatorDeployment1.Name}, operatorDeployment1)
	require.NoError(t, err)
	containers := operatorDeployment1.Spec.Template.Spec.Containers
	assert.Len(t, containers, 2)
	managerContainer := containers[1]
	assert.Equal(t, managerContainer.Image, operatorImage1)
}

func TestOperatorUpgradeMultipleVersions(t *testing.T) {
	fakeClient := testutils.NewFakeClientBuilder(t).Build()
	u := NewACSOperatorManager(fakeClient)

	operatorImages := []string{operatorImage1, operatorImage2}
	err := u.InstallOrUpgrade(context.Background(), operatorImages)
	require.NoError(t, err)

	err = fakeClient.Get(context.Background(), client.ObjectKey{Namespace: operatorNamespace, Name: operatorDeployment1.Name}, operatorDeployment1)
	require.NoError(t, err)
	managerContainer := operatorDeployment1.Spec.Template.Spec.Containers[1]
	assert.Equal(t, managerContainer.Image, operatorImage1)

	err = fakeClient.Get(context.Background(), client.ObjectKey{Namespace: operatorNamespace, Name: operatorDeployment2.Name}, operatorDeployment2)
	require.NoError(t, err)
	managerContainer = operatorDeployment2.Spec.Template.Spec.Containers[1]
	assert.Equal(t, managerContainer.Image, operatorImage2)
}

func TestOperatorUpgradeDoNotInstallLongTagVersion(t *testing.T) {
	fakeClient := testutils.NewFakeClientBuilder(t).Build()
	u := NewACSOperatorManager(fakeClient)

	operatorImageWithLongTag := "quay.io/rhacs-eng/stackrox-operator:3.74.1-with-ridiculously-long-tag-version-name"
	err := u.InstallOrUpgrade(context.Background(), []string{operatorImageWithLongTag})
	require.Errorf(t, err, "zero tags parsed from images")

	deployments := &appsv1.DeploymentList{}
	err = fakeClient.List(context.Background(), deployments)
	require.NoError(t, err)
	assert.Len(t, deployments.Items, 0)
}
