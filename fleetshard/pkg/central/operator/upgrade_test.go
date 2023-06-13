package operator

import (
	"context"
	"testing"

	"helm.sh/helm/v3/pkg/chartutil"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/stackrox/acs-fleet-manager/fleetshard/pkg/testutils"
	appsv1 "k8s.io/api/apps/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	kindCRDName        = "CustomResourceDefinition"
	k8sAPIVersion      = "apiextensions.k8s.io/v1"
	operatorRepository = "quay.io/rhacs-eng/stackrox-operator"
	operatorImage1     = "quay.io/rhacs-eng/stackrox-operator:3.74.1"
	operatorImage2     = "quay.io/rhacs-eng/stackrox-operator:3.74.2"
	crdURL             = "https://raw.githubusercontent.com/stackrox/stackrox/%s/operator/bundle/manifests/"
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
		Name:      "rhacs-operator-manager-3.74.1",
		Namespace: operatorNamespace,
	},
}

var operatorDeployment2 = &appsv1.Deployment{
	ObjectMeta: metav1.ObjectMeta{
		Name:      "rhacs-operator-manager-3.74.2",
		Namespace: operatorNamespace,
	},
}

var metricService = &unstructured.Unstructured{
	Object: map[string]interface{}{
		"kind":       "Service",
		"apiVersion": "v1",
		"metadata": map[string]interface{}{
			"name":      "rhacs-operator-manager-metrics-service",
			"namespace": operatorNamespace,
		},
	},
}

func TestOperatorUpgradeFreshInstall(t *testing.T) {
	fakeClient := testutils.NewFakeClientBuilder(t).Build()
	u := NewACSOperatorManager(fakeClient, crdURL)

	err := u.InstallOrUpgrade(context.Background(), []ACSOperatorImage{
		{
			Image:      operatorImage1,
			InstallCRD: true,
		},
	})
	require.NoError(t, err)

	// check Secured Cluster CRD exists and correct
	err = fakeClient.Get(context.Background(), client.ObjectKey{Namespace: operatorNamespace, Name: securedClusterCRD.GetName()}, securedClusterCRD)
	require.NoError(t, err)
	assert.Equal(t, k8sAPIVersion, securedClusterCRD.GetAPIVersion())
	assert.NotEmpty(t, securedClusterCRD.Object["metadata"])
	assert.NotEmpty(t, securedClusterCRD.Object["spec"])

	// check Central CRD exists and correct
	err = fakeClient.Get(context.Background(), client.ObjectKey{Namespace: operatorNamespace, Name: centralCRD.GetName()}, centralCRD)
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
	u := NewACSOperatorManager(fakeClient, crdURL)

	operatorImages := []ACSOperatorImage{
		{
			Image:      operatorImage1,
			InstallCRD: false,
		},
		{
			Image:      operatorImage2,
			InstallCRD: false,
		},
	}
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
	u := NewACSOperatorManager(fakeClient, crdURL)

	operatorImageWithLongTag := "quay.io/rhacs-eng/stackrox-operator:3.74.1-with-ridiculously-long-tag-version-name"
	err := u.InstallOrUpgrade(context.Background(), []ACSOperatorImage{
		{
			Image:      operatorImageWithLongTag,
			InstallCRD: false,
		}})
	require.Errorf(t, err, "zero tags parsed from images")

	deployments := &appsv1.DeploymentList{}
	err = fakeClient.List(context.Background(), deployments)
	require.NoError(t, err)
	assert.Len(t, deployments.Items, 0)
}

func TestParseOperatorImages(t *testing.T) {
	cases := map[string]struct {
		images             []ACSOperatorImage
		expected           []map[string]string
		expectedCrdVersion string
		shouldFail         bool
	}{
		"should parse one valid operator image": {
			images: []ACSOperatorImage{{
				Image:      operatorImage1,
				InstallCRD: false,
			}},
			expected: []map[string]string{
				{"repository": operatorRepository, "tag": "3.74.1"},
			},
		},
		"should parse two valid operator images": {
			images: []ACSOperatorImage{{
				Image:      operatorImage1,
				InstallCRD: false,
			}, {
				Image:      operatorImage2,
				InstallCRD: false,
			}},
			expected: []map[string]string{
				{"repository": operatorRepository, "tag": "3.74.1"},
				{"repository": operatorRepository, "tag": "3.74.2"},
			},
		},
		"should return correct desired CRD version": {
			images: []ACSOperatorImage{{
				Image:      operatorImage1,
				InstallCRD: false,
			}, {
				Image:      operatorImage2,
				InstallCRD: true,
			}},
			expected: []map[string]string{
				{"repository": operatorRepository, "tag": "3.74.1"},
				{"repository": operatorRepository, "tag": "3.74.2"},
			},
			expectedCrdVersion: "3.74.2",
		},
		"should ignore duplicate operator images": {
			images: []ACSOperatorImage{{
				Image:      operatorImage1,
				InstallCRD: false,
			}, {
				Image:      operatorImage1,
				InstallCRD: false,
			}},
			expected: []map[string]string{
				{"repository": operatorRepository, "tag": "3.74.1"},
			},
		},
		"do not fail if images list is empty": {
			images:     []ACSOperatorImage{},
			shouldFail: false,
		},
		"should accept images from multiple repositories with the same tag": {
			images: []ACSOperatorImage{{
				Image:      "repo1:tag",
				InstallCRD: false,
			}, {
				Image:      "repo2:tag",
				InstallCRD: false,
			}},
			expected: []map[string]string{
				{"repository": "repo1", "tag": "tag"},
				{"repository": "repo2", "tag": "tag"},
			},
		},
		"fail if image does contain colon": {
			images: []ACSOperatorImage{{
				Image:      "quay.io/without-colon-123-tag",
				InstallCRD: false,
			}},
			shouldFail: true,
		},
		"fail if image contains more than one colon": {
			images: []ACSOperatorImage{{
				Image:      "quay.io/image-name:1.2.3:",
				InstallCRD: false,
			}},
			shouldFail: true,
		},
		"fail if image tag is too long": {
			images: []ACSOperatorImage{{
				Image:      "quay.io/image-name:1.2.3-with-ridiculously-long-tag-version-name",
				InstallCRD: false,
			}},
			shouldFail: true,
		},
	}

	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			gotImages, gotCrdVersion, err := parseOperatorImages(c.images)
			if c.shouldFail {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				var expectedRepositoryAndTags []chartutil.Values
				for _, m := range c.expected {
					val := chartutil.Values{"repository": m["repository"], "tag": m["tag"]}
					expectedRepositoryAndTags = append(expectedRepositoryAndTags, val)
				}
				assert.Equal(t, c.expectedCrdVersion, gotCrdVersion)
				assert.Equal(t, expectedRepositoryAndTags, gotImages)
			}
		})
	}
}
