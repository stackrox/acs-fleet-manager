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
)

var SecuredClusterCRD = &unstructured.Unstructured{
	Object: map[string]interface{}{
		"kind":       kindCRDName,
		"apiVersion": k8sAPIVersion,
		"metadata": map[string]interface{}{
			"name": "securedclusters.platform.stackrox.io",
		},
	},
}

var CentralCRD = &unstructured.Unstructured{
	Object: map[string]interface{}{
		"kind":       kindCRDName,
		"apiVersion": k8sAPIVersion,
		"metadata": map[string]interface{}{
			"name": "centrals.platform.stackrox.io",
		},
	},
}

var ServiceAccount = &unstructured.Unstructured{
	Object: map[string]interface{}{
		"kind":       "ServiceAccount",
		"apiVersion": "v1",
		"metadata": map[string]interface{}{
			"name":      "rhacs-operator-controller-manager",
			"namespace": operatorNamespace,
		},
	},
}

var OperatorDeployment = &unstructured.Unstructured{
	Object: map[string]interface{}{
		"kind":       "Deployment",
		"apiVersion": "apps/v1",
		"metadata": map[string]interface{}{
			"name":      "rhacs-operator-controller-manager",
			"namespace": operatorNamespace,
		},
	},
}

var OperatorConfigMap = &unstructured.Unstructured{
	Object: map[string]interface{}{
		"kind":       "ConfigMap",
		"apiVersion": "v1",
		"metadata": map[string]interface{}{
			"name": "rhacs-operator-manager-config",
		},
	},
}

func TestOperatorUpgradeFreshInstall(t *testing.T) {
	fakeClient := testutils.NewFakeClientBuilder(t).Build()
	u := NewACSOperatorManager(fakeClient)

	err := u.Upgrade(context.TODO())

	require.NoError(t, err)

	// check Secured Cluster CRD exists and correct
	err = fakeClient.Get(context.TODO(), client.ObjectKey{Namespace: operatorNamespace, Name: SecuredClusterCRD.GetName()}, SecuredClusterCRD)
	require.NoError(t, err)
	assert.Equal(t, k8sAPIVersion, SecuredClusterCRD.GetAPIVersion())
	assert.NotEmpty(t, SecuredClusterCRD.Object["metadata"])
	assert.NotEmpty(t, SecuredClusterCRD.Object["spec"])

	// check Central CRD exists and correct
	err = fakeClient.Get(context.TODO(), client.ObjectKey{Namespace: operatorNamespace, Name: CentralCRD.GetName()}, CentralCRD)
	require.NoError(t, err)
	assert.Equal(t, k8sAPIVersion, CentralCRD.GetAPIVersion())
	assert.NotEmpty(t, CentralCRD.Object["metadata"])
	assert.NotEmpty(t, CentralCRD.Object["spec"])

	// check ServiceAccount exists
	err = fakeClient.Get(context.TODO(), client.ObjectKey{Namespace: operatorNamespace, Name: ServiceAccount.GetName()}, ServiceAccount)
	require.NoError(t, err)
	assert.Equal(t, k8sAPIVersion, CentralCRD.GetAPIVersion())
	assert.NotEmpty(t, CentralCRD.Object["metadata"])
	assert.NotEmpty(t, CentralCRD.Object["spec"])

	// check Operator Deployment exists
	err = fakeClient.Get(context.TODO(), client.ObjectKey{Namespace: operatorNamespace, Name: OperatorDeployment.GetName()}, OperatorDeployment)
	require.NoError(t, err)
	assert.Equal(t, "apps/v1", OperatorDeployment.GetAPIVersion())
	assert.NotEmpty(t, OperatorDeployment.Object["metadata"])
	assert.NotEmpty(t, OperatorDeployment.Object["spec"])

	// check Operator ConfigMap exists
	err = fakeClient.Get(context.TODO(), client.ObjectKey{Namespace: operatorNamespace, Name: OperatorConfigMap.GetName()}, OperatorConfigMap)
	require.NoError(t, err)
	assert.Equal(t, "v1", OperatorConfigMap.GetAPIVersion())
	assert.NotEmpty(t, OperatorConfigMap.Object["metadata"])

}
