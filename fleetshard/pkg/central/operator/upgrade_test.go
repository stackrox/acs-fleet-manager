package operator

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/stackrox/acs-fleet-manager/fleetshard/pkg/testutils"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	kindCRDName   = "CustomResourceDefinition"
	k8sAPIVersion = "apiextensions.k8s.io/v1"
	operatorImage = "quay.io/rhacs-eng/stackrox-operator:3.74.0"
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

var operatorDeployment = &unstructured.Unstructured{
	Object: map[string]interface{}{
		"kind":       "Deployment",
		"apiVersion": "apps/v1",
		"metadata": map[string]interface{}{
			"name":      "rhacs-operator-controller-manager-3.74.0",
			"namespace": operatorNamespace,
		},
	},
}

func TestOperatorUpgradeFreshInstall(t *testing.T) {
	fakeClient := testutils.NewFakeClientBuilder(t).Build()
	u := NewACSOperatorManager(fakeClient)

	err := u.InstallOrUpgrade(context.Background(), operatorImage)

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

	// check Operator Deployment exists
	err = fakeClient.Get(context.Background(), client.ObjectKey{Namespace: operatorNamespace, Name: operatorDeployment.GetName()}, operatorDeployment)
	require.NoError(t, err)
	assert.Equal(t, "apps/v1", operatorDeployment.GetAPIVersion())
	assert.NotEmpty(t, operatorDeployment.Object["metadata"])
	assert.NotEmpty(t, operatorDeployment.Object["spec"])
	templateSpec := operatorDeployment.Object["spec"].(map[string]interface{})["template"].(map[string]interface{})["spec"]
	assert.NotEmpty(t, templateSpec)
	assert.Contains(t, templateSpec, "containers")
	containers := templateSpec.(map[string]interface{})["containers"].([]interface{})
	assert.Len(t, containers, 2)
	managerContainer := containers[1].(map[string]interface{})
	assert.Equal(t, managerContainer["image"], operatorImage)

}
