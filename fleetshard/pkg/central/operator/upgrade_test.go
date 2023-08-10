package operator

import (
	"context"
	"testing"

	"helm.sh/helm/v3/pkg/chartutil"

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
	kindCRDName        = "CustomResourceDefinition"
	k8sAPIVersion      = "apiextensions.k8s.io/v1"
	operatorRepository = "quay.io/rhacs-eng/stackrox-operator"
	operatorImage1     = "quay.io/rhacs-eng/stackrox-operator:4.0.1"
	operatorImage2     = "quay.io/rhacs-eng/stackrox-operator:4.0.2"
	crdTag1            = "4.0.1"
	crdURL             = "https://raw.githubusercontent.com/stackrox/stackrox/%s/operator/bundle/manifests/"
	deploymentName1    = operatorDeploymentPrefix + "-4.0.1"
	deploymentName2    = operatorDeploymentPrefix + "-4.0.2"
)

var operatorConfig1 = DeploymentConfig{
	Image:  operatorImage1,
	GitRef: "4.0.1",
}

var operatorConfig2 = DeploymentConfig{
	Image:  operatorImage2,
	GitRef: "4.0.2",
}

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

func TestOperatorUpgradeFreshInstall(t *testing.T) {
	fakeClient := testutils.NewFakeClientBuilder(t).Build()
	u := NewACSOperatorManager(fakeClient, crdURL)

	err := u.InstallOrUpgrade(context.Background(), []DeploymentConfig{operatorConfig1}, crdTag1)
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
	u := NewACSOperatorManager(fakeClient, crdURL)

	operatorConfigs := []DeploymentConfig{operatorConfig1, operatorConfig2}

	err := u.InstallOrUpgrade(context.Background(), operatorConfigs, crdTag1)
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

func TestOperatorUpgradeDoNotInstallLongTagVersion(t *testing.T) {
	fakeClient := testutils.NewFakeClientBuilder(t).Build()
	u := NewACSOperatorManager(fakeClient, crdURL)

	longVersionName := "4.0.1-with-ridiculously-long-version-name-like-really-long-one-which-has-more-than-63-characters"
	operatorConfig := DeploymentConfig{
		Image:  operatorImage1,
		GitRef: longVersionName,
	}
	err := u.InstallOrUpgrade(context.Background(), []DeploymentConfig{operatorConfig}, crdTag1)
	require.Errorf(t, err, "zero tags parsed from images")

	deployments := &appsv1.DeploymentList{}
	err = fakeClient.List(context.Background(), deployments)
	require.NoError(t, err)
	assert.Len(t, deployments.Items, 0)
}

func TestOperatorUpgradeImageWithDigest(t *testing.T) {
	fakeClient := testutils.NewFakeClientBuilder(t).Build()
	u := NewACSOperatorManager(fakeClient, crdURL)

	digestedImage := "quay.io/rhacs-eng/stackrox-operator:4.0.1@sha256:232a180dbcbcfa7250917507f3827d88a9ae89bb1cdd8fe3ac4db7b764ebb25a"
	operatorConfig := DeploymentConfig{
		Image:  digestedImage,
		GitRef: "4.0.1",
	}
	err := u.InstallOrUpgrade(context.Background(), []DeploymentConfig{operatorConfig}, crdTag1)
	require.NoError(t, err)

	err = fakeClient.Get(context.Background(), client.ObjectKey{Namespace: ACSOperatorNamespace, Name: operatorDeployment1.Name}, operatorDeployment1)
	require.NoError(t, err)
	managerContainer := operatorDeployment1.Spec.Template.Spec.Containers[1]
	assert.Equal(t, managerContainer.Image, digestedImage)
}

func TestRemoveUnusedEmpty(t *testing.T) {
	fakeClient := testutils.NewFakeClientBuilder(t).Build()
	u := NewACSOperatorManager(fakeClient, crdURL)
	ctx := context.Background()

	err := u.RemoveUnusedOperators(ctx, []string{})
	require.NoError(t, err)
}

func TestRemoveOneUnusedOperator(t *testing.T) {
	fakeClient := testutils.NewFakeClientBuilder(t, operatorDeployment1, serviceAccount).Build()
	u := NewACSOperatorManager(fakeClient, crdURL)
	ctx := context.Background()

	err := fakeClient.Get(context.Background(), client.ObjectKey{Namespace: ACSOperatorNamespace, Name: operatorDeployment1.Name}, operatorDeployment1)
	require.NoError(t, err)
	err = fakeClient.Get(context.Background(), client.ObjectKey{Namespace: ACSOperatorNamespace, Name: serviceAccount.GetName()}, serviceAccount)
	require.NoError(t, err)

	err = u.RemoveUnusedOperators(ctx, []string{operatorImage2})
	require.NoError(t, err)
	// deployment is deleted but service account still persist
	err = fakeClient.Get(context.Background(), client.ObjectKey{Namespace: ACSOperatorNamespace, Name: operatorDeployment1.Name}, operatorDeployment1)
	require.True(t, errors.IsNotFound(err))
	err = fakeClient.Get(context.Background(), client.ObjectKey{Namespace: ACSOperatorNamespace, Name: serviceAccount.GetName()}, serviceAccount)
	require.NoError(t, err)
}

func TestRemoveOneUnusedOperatorFromMany(t *testing.T) {
	fakeClient := testutils.NewFakeClientBuilder(t, operatorDeployment1, operatorDeployment2, serviceAccount).Build()
	u := NewACSOperatorManager(fakeClient, crdURL)
	ctx := context.Background()

	err := u.RemoveUnusedOperators(ctx, []string{operatorImage2})
	require.NoError(t, err)
	err = fakeClient.Get(context.Background(), client.ObjectKey{Namespace: ACSOperatorNamespace, Name: operatorDeployment1.Name}, operatorDeployment1)
	require.True(t, errors.IsNotFound(err))
	err = fakeClient.Get(context.Background(), client.ObjectKey{Namespace: ACSOperatorNamespace, Name: operatorDeployment2.Name}, operatorDeployment2)
	require.NoError(t, err)
	err = fakeClient.Get(context.Background(), client.ObjectKey{Namespace: ACSOperatorNamespace, Name: serviceAccount.GetName()}, serviceAccount)
	require.NoError(t, err)

	// remove remaining
	err = u.RemoveUnusedOperators(ctx, []string{})
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
	u := NewACSOperatorManager(fakeClient, crdURL)
	ctx := context.Background()

	err := u.RemoveUnusedOperators(ctx, []string{})
	require.NoError(t, err)
	deployments := &appsv1.DeploymentList{}
	err = fakeClient.List(context.Background(), deployments)
	require.NoError(t, err)
	assert.Len(t, deployments.Items, 0)
	err = fakeClient.Get(context.Background(), client.ObjectKey{Namespace: ACSOperatorNamespace, Name: serviceAccount.GetName()}, serviceAccount)
	require.NoError(t, err)
}

func TestParseOperatorConfigs(t *testing.T) {
	cases := map[string]struct {
		operatorConfigs []DeploymentConfig
		expected        []map[string]string
		shouldFail      bool
	}{
		"should parse one valid operator image": {
			operatorConfigs: []DeploymentConfig{operatorConfig1},
			expected: []map[string]string{
				{"deploymentName": deploymentName1, "image": operatorImage1, "labelSelector": "4.0.1"},
			},
		},
		"should parse two valid operator images": {
			operatorConfigs: []DeploymentConfig{operatorConfig1, operatorConfig2},
			expected: []map[string]string{
				{"deploymentName": deploymentName1, "image": operatorImage1, "labelSelector": "4.0.1"},
				{"deploymentName": deploymentName2, "image": operatorImage2, "labelSelector": "4.0.2"},
			},
		},
		"should parse image with tag and digest": {
			operatorConfigs: []DeploymentConfig{{
				Image:  "quay.io/image-with-tag-and-digest:1.2.3@sha256:4ff5cb2dcddaaa2a4b702516870c85177e53ccc3566509c36c2d84b01ef8f783",
				GitRef: "version1",
			}},
			expected: []map[string]string{
				{
					"deploymentName": operatorDeploymentPrefix + "-version1",
					"image":          "quay.io/image-with-tag-and-digest:1.2.3@sha256:4ff5cb2dcddaaa2a4b702516870c85177e53ccc3566509c36c2d84b01ef8f783",
					"labelSelector":  "version1",
				},
			},
		},
		"should parse image without tag": {
			operatorConfigs: []DeploymentConfig{{
				Image:  "quay.io/image-without-tag",
				GitRef: "version1",
			}},
			expected: []map[string]string{
				{"deploymentName": operatorDeploymentPrefix + "-version1", "image": "quay.io/image-without-tag", "labelSelector": "version1"},
			},
		},
		"do not fail if images list is empty": {
			operatorConfigs: []DeploymentConfig{},
			shouldFail:      false,
		},
		"should accept images from multiple repositories with the same tag": {
			operatorConfigs: []DeploymentConfig{
				{
					Image:  "repo1:tag",
					GitRef: "version1",
				},
				{
					Image:  "repo2:tag",
					GitRef: "version2",
				}},
			expected: []map[string]string{
				{"deploymentName": operatorDeploymentPrefix + "-version1", "image": "repo1:tag", "labelSelector": "version1"},
				{"deploymentName": operatorDeploymentPrefix + "-version2", "image": "repo2:tag", "labelSelector": "version2"},
			},
		},
		"fail if image contains more than one colon": {
			operatorConfigs: []DeploymentConfig{{
				Image:  "quay.io/image-name:1.2.3:",
				GitRef: "version1",
			}},
			shouldFail: true,
		},
		"fail if GitRef is way too long for the DeploymentName": {
			operatorConfigs: []DeploymentConfig{{
				Image:  "quay.io/image-name:tag",
				GitRef: "1.2.3-with-ridiculously-long-version-name-like-really-long-one-which-has-more-than-63-characters",
			}},
			shouldFail: true,
		},
	}

	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			gotImages, err := parseOperatorConfigs(c.operatorConfigs)
			if c.shouldFail {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				var expectedRepositoryAndTags []chartutil.Values
				for _, m := range c.expected {
					val := chartutil.Values{
						"deploymentName": m["deploymentName"],
						"image":          m["image"],
						"labelSelector":  m["labelSelector"],
					}
					expectedRepositoryAndTags = append(expectedRepositoryAndTags, val)
				}
				assert.Equal(t, expectedRepositoryAndTags, gotImages)
			}
		})
	}
}
