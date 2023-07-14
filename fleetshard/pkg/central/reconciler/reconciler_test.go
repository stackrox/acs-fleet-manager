package reconciler

import (
	"context"
	"embed"
	"fmt"
	"github.com/stackrox/rox/pkg/utils"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws/credentials/stscreds"

	"github.com/aws/aws-sdk-go/aws/awserr"
	openshiftRouteV1 "github.com/openshift/api/route/v1"
	"github.com/pkg/errors"
	"github.com/stackrox/acs-fleet-manager/fleetshard/config"
	"github.com/stackrox/acs-fleet-manager/fleetshard/pkg/central/charts"
	"github.com/stackrox/acs-fleet-manager/fleetshard/pkg/central/cloudprovider"
	"github.com/stackrox/acs-fleet-manager/fleetshard/pkg/central/cloudprovider/awsclient"
	"github.com/stackrox/acs-fleet-manager/fleetshard/pkg/central/postgres"
	"github.com/stackrox/acs-fleet-manager/fleetshard/pkg/k8s"
	"github.com/stackrox/acs-fleet-manager/fleetshard/pkg/testutils"
	"github.com/stackrox/acs-fleet-manager/fleetshard/pkg/util"
	centralConstants "github.com/stackrox/acs-fleet-manager/internal/dinosaur/constants"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/api/private"
	"github.com/stackrox/acs-fleet-manager/pkg/features"
	"github.com/stackrox/rox/operator/apis/platform/v1alpha1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"helm.sh/helm/v3/pkg/chart/loader"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	centralName               = "test-central"
	centralID                 = "cb45idheg5ip6dq1jo4g"
	centralNamespace          = "rhacs-" + centralID
	centralReencryptRouteName = "managed-central-reencrypt"
	conditionTypeReady        = "Ready"
	clusterName               = "test-cluster"
	environment               = "test"
)

var simpleManagedCentral = private.ManagedCentral{
	Id: centralID,
	Metadata: private.ManagedCentralAllOfMetadata{
		Name:      centralName,
		Namespace: centralNamespace,
	},
	Spec: private.ManagedCentralAllOfSpec{
		Auth: private.ManagedCentralAllOfSpecAuth{
			OwnerOrgId:   "12345",
			OwnerOrgName: "org-name",
		},
		UiEndpoint: private.ManagedCentralAllOfSpecUiEndpoint{
			Host: fmt.Sprintf("acs-%s.acs.rhcloud.test", centralID),
		},
		DataEndpoint: private.ManagedCentralAllOfSpecDataEndpoint{
			Host: fmt.Sprintf("acs-data-%s.acs.rhcloud.test", centralID),
		},
		Central: private.ManagedCentralAllOfSpecCentral{
			InstanceType: "standard",
		},
	},
}

//go:embed testdata
var testdata embed.FS

func centralDBInitFunc(_ context.Context, _ postgres.DBConnection, _, _ string) error {
	return nil
}

func conditionForType(conditions []private.DataPlaneClusterUpdateStatusRequestConditions, conditionType string) (*private.DataPlaneClusterUpdateStatusRequestConditions, bool) {
	for _, c := range conditions {
		if c.Type == conditionType {
			return &c, true
		}
	}
	return nil, false
}

func TestReconcileCreate(t *testing.T) {
	fakeClient := testutils.NewFakeClientBuilder(t).Build()
	r := NewCentralReconciler(fakeClient,
		private.ManagedCentral{},
		nil,
		centralDBInitFunc,
		CentralReconcilerOptions{ClusterName: clusterName, Environment: environment, UseRoutes: true},
	)

	status, err := r.Reconcile(context.TODO(), simpleManagedCentral)
	require.NoError(t, err)

	readyCondition, ok := conditionForType(status.Conditions, conditionTypeReady)
	require.True(t, ok)
	assert.Equal(t, "True", readyCondition.Status, "Ready condition not found in conditions", status.Conditions)

	central := &v1alpha1.Central{}
	err = fakeClient.Get(context.TODO(), client.ObjectKey{Name: centralName, Namespace: centralNamespace}, central)
	require.NoError(t, err)
	assert.Equal(t, centralName, central.GetName())
	assert.Equal(t, simpleManagedCentral.Id, central.GetLabels()[tenantIDLabelKey])
	assert.Equal(t, simpleManagedCentral.Id, central.Spec.Customize.Labels[tenantIDLabelKey])
	assert.Equal(t, environment, central.Spec.Customize.Annotations[envAnnotationKey])
	assert.Equal(t, clusterName, central.Spec.Customize.Annotations[clusterNameAnnotationKey])
	assert.Equal(t, simpleManagedCentral.Spec.Auth.OwnerOrgName, central.Spec.Customize.Annotations[orgNameAnnotationKey])
	assert.Equal(t, simpleManagedCentral.Spec.Auth.OwnerOrgId, central.Spec.Customize.Labels[orgIDLabelKey])
	assert.Equal(t, simpleManagedCentral.Spec.Central.InstanceType, central.Spec.Customize.Labels[instanceTypeLabelKey])
	assert.Equal(t, "1", central.GetAnnotations()[util.RevisionAnnotationKey])
	assert.Equal(t, "true", central.GetAnnotations()[managedServicesAnnotation])
	assert.Equal(t, true, *central.Spec.Central.Exposure.Route.Enabled)

	route := &openshiftRouteV1.Route{}
	err = fakeClient.Get(context.TODO(), client.ObjectKey{Name: centralReencryptRouteName, Namespace: centralNamespace}, route)
	require.NoError(t, err)
	assert.Equal(t, centralReencryptRouteName, route.GetName())
	assert.Equal(t, openshiftRouteV1.TLSTerminationReencrypt, route.Spec.TLS.Termination)
	assert.Equal(t, testutils.CentralCA, route.Spec.TLS.DestinationCACertificate)
}

func TestReconcileCreateWithManagedDB(t *testing.T) {
	fakeClient := testutils.NewFakeClientBuilder(t).Build()

	managedDBProvisioningClient := &cloudprovider.DBClientMock{}
	managedDBProvisioningClient.EnsureDBProvisionedFunc = func(_ context.Context, _ string, _ string, _ bool) error {
		return nil
	}
	managedDBProvisioningClient.GetDBConnectionFunc = func(_ string) (postgres.DBConnection, error) {
		connection, err := postgres.NewDBConnection("localhost", 5432, "rhacs", "postgres")
		if err != nil {
			return postgres.DBConnection{}, err
		}
		return connection, nil
	}

	r := NewCentralReconciler(fakeClient, private.ManagedCentral{}, managedDBProvisioningClient, centralDBInitFunc,
		CentralReconcilerOptions{
			UseRoutes:        true,
			ManagedDBEnabled: true,
		})

	status, err := r.Reconcile(context.TODO(), simpleManagedCentral)
	require.NoError(t, err)
	assert.Len(t, managedDBProvisioningClient.EnsureDBProvisionedCalls(), 1)

	readyCondition, ok := conditionForType(status.Conditions, conditionTypeReady)
	require.True(t, ok)
	assert.Equal(t, "True", readyCondition.Status, "Ready condition not found in conditions", status.Conditions)

	central := &v1alpha1.Central{}
	err = fakeClient.Get(context.TODO(), client.ObjectKey{Name: centralName, Namespace: centralNamespace}, central)
	require.NoError(t, err)

	route := &openshiftRouteV1.Route{}
	err = fakeClient.Get(context.TODO(), client.ObjectKey{Name: centralReencryptRouteName, Namespace: centralNamespace}, route)
	require.NoError(t, err)

	secret := &v1.Secret{}
	err = fakeClient.Get(context.TODO(), client.ObjectKey{Name: centralDbSecretName, Namespace: centralNamespace}, secret)
	require.NoError(t, err)
	password, ok := secret.Data["password"]
	require.True(t, ok)
	assert.NotEmpty(t, password)
}

func TestReconcileCreateWithLabelOperatorVersion(t *testing.T) {
	fakeClient := testutils.NewFakeClientBuilder(t).Build()
	t.Setenv(features.TargetedOperatorUpgrades.EnvVar(), "true")

	r := NewCentralReconciler(fakeClient, private.ManagedCentral{}, nil, centralDBInitFunc,
		CentralReconcilerOptions{
			UseRoutes: true,
		})

	exampleCentral := simpleManagedCentral
	exampleCentral.Spec.OperatorImage = "quay.io/rhacs-eng/stackrox-operator:4.1.0"
	status, err := r.Reconcile(context.TODO(), exampleCentral)
	require.NoError(t, err)
	readyCondition, ok := conditionForType(status.Conditions, conditionTypeReady)
	require.True(t, ok)
	assert.Equal(t, "True", readyCondition.Status, "Ready condition not found in conditions", status.Conditions)

	central := &v1alpha1.Central{}
	err = fakeClient.Get(context.TODO(), client.ObjectKey{Name: centralName, Namespace: centralNamespace}, central)
	require.NoError(t, err)
	assert.Equal(t, "4.1.0", central.ObjectMeta.Labels[ReconcileOperatorSelector])
}

func TestReconcileCreateWithManagedDBNoCredentials(t *testing.T) {
	t.Setenv("AWS_ACCESS_KEY", "")
	t.Setenv("AWS_SECRET_ACCESS_KEY", "")
	t.Setenv("AWS_REGION", "us-east-1")
	t.Setenv("AWS_ROLE_ARN", "arn:aws:iam::012456789:role/fake_role")
	t.Setenv("AWS_WEB_IDENTITY_TOKEN_FILE", "/var/run/secrets/tokens/aws-token")

	fakeClient := testutils.NewFakeClientBuilder(t).Build()

	managedDBProvisioningClient, err := awsclient.NewRDSClient(
		&config.Config{
			ManagedDB: config.ManagedDB{
				SecurityGroup: "security-group",
				SubnetGroup:   "db-group",
			},
		})
	require.NoError(t, err)

	r := NewCentralReconciler(fakeClient, private.ManagedCentral{}, managedDBProvisioningClient, centralDBInitFunc,
		CentralReconcilerOptions{
			UseRoutes:        true,
			ManagedDBEnabled: true,
		})

	_, err = r.Reconcile(context.TODO(), simpleManagedCentral)
	var awsErr awserr.Error
	require.ErrorAs(t, err, &awsErr)
	assert.Equal(t, stscreds.ErrCodeWebIdentity, awsErr.Code())
}

func TestReconcileUpdateSucceeds(t *testing.T) {
	fakeClient := testutils.NewFakeClientBuilder(t, &v1alpha1.Central{
		ObjectMeta: metav1.ObjectMeta{
			Name:        centralName,
			Namespace:   centralNamespace,
			Annotations: map[string]string{util.RevisionAnnotationKey: "3"},
		},
	}, centralDeploymentObject()).Build()

	r := NewCentralReconciler(fakeClient, private.ManagedCentral{}, nil, centralDBInitFunc, CentralReconcilerOptions{})

	status, err := r.Reconcile(context.TODO(), simpleManagedCentral)
	require.NoError(t, err)

	assert.Equal(t, "True", status.Conditions[0].Status)

	central := &v1alpha1.Central{}
	err = fakeClient.Get(context.TODO(), client.ObjectKey{Name: centralName, Namespace: centralNamespace}, central)
	require.NoError(t, err)
	assert.Equal(t, centralName, central.GetName())
	assert.Equal(t, "4", central.GetAnnotations()[util.RevisionAnnotationKey])
}

func TestReconcileLastHashNotUpdatedOnError(t *testing.T) {
	fakeClient := testutils.NewFakeClientBuilder(t, &v1alpha1.Central{
		ObjectMeta: metav1.ObjectMeta{
			Name:        centralName,
			Namespace:   centralNamespace,
			Annotations: map[string]string{util.RevisionAnnotationKey: "invalid annotation"},
		},
	}, centralDeploymentObject()).Build()

	r := CentralReconciler{
		status:         pointer.Int32(0),
		client:         fakeClient,
		central:        private.ManagedCentral{},
		resourcesChart: resourcesChart,
	}

	_, err := r.Reconcile(context.TODO(), simpleManagedCentral)
	require.Error(t, err)

	assert.Equal(t, [16]byte{}, r.lastCentralHash)
}

func TestReconcileLastHashSetOnSuccess(t *testing.T) {
	fakeClient := testutils.NewFakeClientBuilder(t, &v1alpha1.Central{
		ObjectMeta: metav1.ObjectMeta{
			Name:        centralName,
			Namespace:   centralNamespace,
			Annotations: map[string]string{util.RevisionAnnotationKey: "3"},
		},
	}, centralDeploymentObject()).Build()

	r := NewCentralReconciler(fakeClient, private.ManagedCentral{}, nil, centralDBInitFunc, CentralReconcilerOptions{})

	managedCentral := simpleManagedCentral
	managedCentral.RequestStatus = centralConstants.CentralRequestStatusReady.String()

	expectedHash, err := util.MD5SumFromJSONStruct(&managedCentral)
	require.NoError(t, err)

	_, err = r.Reconcile(context.TODO(), managedCentral)
	require.NoError(t, err)

	assert.Equal(t, expectedHash, r.lastCentralHash)

	status, err := r.Reconcile(context.TODO(), managedCentral)
	require.Nil(t, status)
	require.ErrorIs(t, err, ErrCentralNotChanged)

	central := &v1alpha1.Central{}
	err = fakeClient.Get(context.TODO(), client.ObjectKey{Name: centralName, Namespace: centralNamespace}, central)
	require.NoError(t, err)
	assert.Equal(t, "4", central.Annotations[util.RevisionAnnotationKey])
}

func TestIgnoreCacheForCentralNotReady(t *testing.T) {
	fakeClient := testutils.NewFakeClientBuilder(t, &v1alpha1.Central{
		ObjectMeta: metav1.ObjectMeta{
			Name:        centralName,
			Namespace:   centralNamespace,
			Annotations: map[string]string{util.RevisionAnnotationKey: "3"},
		},
	}, centralDeploymentObject()).Build()

	r := NewCentralReconciler(fakeClient, private.ManagedCentral{}, nil, centralDBInitFunc, CentralReconcilerOptions{})

	managedCentral := simpleManagedCentral
	managedCentral.RequestStatus = centralConstants.CentralRequestStatusProvisioning.String()

	expectedHash, err := util.MD5SumFromJSONStruct(&managedCentral)
	require.NoError(t, err)

	_, err = r.Reconcile(context.TODO(), managedCentral)
	require.NoError(t, err)
	assert.Equal(t, expectedHash, r.lastCentralHash)

	_, err = r.Reconcile(context.TODO(), managedCentral)
	require.NoError(t, err)
}

func TestIgnoreCacheForCentralForceReconcileAlways(t *testing.T) {
	fakeClient := testutils.NewFakeClientBuilder(t, &v1alpha1.Central{
		ObjectMeta: metav1.ObjectMeta{
			Name:        centralName,
			Namespace:   centralNamespace,
			Annotations: map[string]string{util.RevisionAnnotationKey: "3"},
		},
	}, centralDeploymentObject()).Build()

	r := NewCentralReconciler(fakeClient, private.ManagedCentral{}, nil, centralDBInitFunc, CentralReconcilerOptions{})

	managedCentral := simpleManagedCentral
	managedCentral.RequestStatus = centralConstants.CentralRequestStatusReady.String()
	managedCentral.ForceReconcile = "always"

	expectedHash, err := util.MD5SumFromJSONStruct(&managedCentral)
	require.NoError(t, err)

	_, err = r.Reconcile(context.TODO(), managedCentral)
	require.NoError(t, err)
	assert.Equal(t, expectedHash, r.lastCentralHash)

	_, err = r.Reconcile(context.TODO(), managedCentral)
	require.NoError(t, err)
}

func TestReconcileDelete(t *testing.T) {
	fakeClient := testutils.NewFakeClientBuilder(t).Build()
	r := NewCentralReconciler(fakeClient, private.ManagedCentral{}, nil, centralDBInitFunc, CentralReconcilerOptions{UseRoutes: true})

	_, err := r.Reconcile(context.TODO(), simpleManagedCentral)
	require.NoError(t, err)
	deletedCentral := simpleManagedCentral
	deletedCentral.Metadata.DeletionTimestamp = "2006-01-02T15:04:05Z07:00"

	// trigger deletion
	statusTrigger, err := r.Reconcile(context.TODO(), deletedCentral)
	require.Error(t, err, ErrDeletionInProgress)
	require.Nil(t, statusTrigger)

	// deletion completed needs second reconcile to check as deletion is async in a kubernetes cluster
	statusDeletion, err := r.Reconcile(context.TODO(), deletedCentral)
	require.NoError(t, err)
	require.NotNil(t, statusDeletion)

	readyCondition, ok := conditionForType(statusDeletion.Conditions, conditionTypeReady)
	require.True(t, ok, "Ready condition not found in conditions", statusDeletion.Conditions)
	assert.Equal(t, "False", readyCondition.Status)
	assert.Equal(t, "Deleted", readyCondition.Reason)

	central := &v1alpha1.Central{}
	err = fakeClient.Get(context.TODO(), client.ObjectKey{Name: centralName, Namespace: centralNamespace}, central)
	assert.True(t, k8sErrors.IsNotFound(err))

	route := &openshiftRouteV1.Route{}
	err = fakeClient.Get(context.TODO(), client.ObjectKey{Name: centralReencryptRouteName, Namespace: centralNamespace}, route)
	assert.True(t, k8sErrors.IsNotFound(err))
}

func TestDisablePauseAnnotation(t *testing.T) {
	fakeClient := testutils.NewFakeClientBuilder(t).Build()
	r := NewCentralReconciler(fakeClient, private.ManagedCentral{}, nil, centralDBInitFunc, CentralReconcilerOptions{UseRoutes: true})

	_, err := r.Reconcile(context.TODO(), simpleManagedCentral)
	require.NoError(t, err)

	central := &v1alpha1.Central{}
	err = fakeClient.Get(context.TODO(), client.ObjectKey{Name: centralName, Namespace: centralNamespace}, central)
	require.NoError(t, err)
	central.Annotations[PauseReconcileAnnotation] = "true"
	err = fakeClient.Update(context.TODO(), central)
	require.NoError(t, err)

	err = r.disablePauseReconcileIfPresent(context.TODO(), central)
	require.NoError(t, err)

	err = fakeClient.Get(context.TODO(), client.ObjectKey{Name: centralName, Namespace: centralNamespace}, central)
	require.NoError(t, err)
	require.Equal(t, "false", central.Annotations[PauseReconcileAnnotation])
}

func TestReconcileDeleteWithManagedDB(t *testing.T) {
	fakeClient := testutils.NewFakeClientBuilder(t).Build()

	managedDBProvisioningClient := &cloudprovider.DBClientMock{}
	managedDBProvisioningClient.EnsureDBProvisionedFunc = func(_ context.Context, _ string, _ string, _ bool) error {
		return nil
	}
	managedDBProvisioningClient.EnsureDBDeprovisionedFunc = func(_ string, _ bool) error {
		return nil
	}
	managedDBProvisioningClient.GetDBConnectionFunc = func(_ string) (postgres.DBConnection, error) {
		connection, err := postgres.NewDBConnection("localhost", 5432, "rhacs", "postgres")
		if err != nil {
			return postgres.DBConnection{}, err
		}
		return connection, nil
	}

	r := NewCentralReconciler(fakeClient, private.ManagedCentral{}, managedDBProvisioningClient, centralDBInitFunc,
		CentralReconcilerOptions{
			UseRoutes:        true,
			ManagedDBEnabled: true,
		})

	_, err := r.Reconcile(context.TODO(), simpleManagedCentral)
	require.NoError(t, err)
	assert.Len(t, managedDBProvisioningClient.EnsureDBProvisionedCalls(), 1)

	deletedCentral := simpleManagedCentral
	deletedCentral.Metadata.DeletionTimestamp = "2006-01-02T15:04:05Z07:00"

	// trigger deletion
	managedDBProvisioningClient.EnsureDBProvisionedFunc = func(_ context.Context, _ string, _ string, _ bool) error {
		return nil
	}
	statusTrigger, err := r.Reconcile(context.TODO(), deletedCentral)
	require.Error(t, err, ErrDeletionInProgress)
	require.Nil(t, statusTrigger)
	assert.Len(t, managedDBProvisioningClient.EnsureDBProvisionedCalls(), 1)

	// deletion completed needs second reconcile to check as deletion is async in a kubernetes cluster
	statusDeletion, err := r.Reconcile(context.TODO(), deletedCentral)
	require.NoError(t, err)
	require.NotNil(t, statusDeletion)

	readyCondition, ok := conditionForType(statusDeletion.Conditions, conditionTypeReady)
	require.True(t, ok, "Ready condition not found in conditions", statusDeletion.Conditions)
	assert.Equal(t, "False", readyCondition.Status)
	assert.Equal(t, "Deleted", readyCondition.Reason)

	assert.Len(t, managedDBProvisioningClient.EnsureDBDeprovisionedCalls(), 2)

	central := &v1alpha1.Central{}
	err = fakeClient.Get(context.TODO(), client.ObjectKey{Name: centralName, Namespace: centralNamespace}, central)
	assert.True(t, k8sErrors.IsNotFound(err))

	route := &openshiftRouteV1.Route{}
	err = fakeClient.Get(context.TODO(), client.ObjectKey{Name: centralReencryptRouteName, Namespace: centralNamespace}, route)
	assert.True(t, k8sErrors.IsNotFound(err))

	secret := &v1.Secret{}
	err = fakeClient.Get(context.TODO(), client.ObjectKey{Name: centralDbSecretName, Namespace: centralNamespace}, secret)
	assert.True(t, k8sErrors.IsNotFound(err))
}

func TestCentralChanged(t *testing.T) {
	tests := []struct {
		name           string
		lastCentral    *private.ManagedCentral
		currentCentral private.ManagedCentral
		want           bool
	}{
		{
			name:           "return true when lastCentral was not set",
			lastCentral:    nil,
			currentCentral: simpleManagedCentral,
			want:           true,
		},
		{
			name:           "return false when lastCentral equal currentCentral",
			lastCentral:    &simpleManagedCentral,
			currentCentral: simpleManagedCentral,
			want:           false,
		},
		{
			name:        "return true when lastCentral not equal currentCentral",
			lastCentral: &simpleManagedCentral,
			currentCentral: private.ManagedCentral{
				Metadata: simpleManagedCentral.Metadata,
				Spec: private.ManagedCentralAllOfSpec{
					UiEndpoint: private.ManagedCentralAllOfSpecUiEndpoint{
						Host: "central.cluster.local",
					},
				},
			},
			want: true,
		},
	}

	fakeClient := testutils.NewFakeClientBuilder(t, centralDeploymentObject()).Build()

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			reconciler := NewCentralReconciler(fakeClient, test.currentCentral, nil, centralDBInitFunc, CentralReconcilerOptions{})

			if test.lastCentral != nil {
				err := reconciler.setLastCentralHash(*test.lastCentral)
				require.NoError(t, err)
			}

			got, err := reconciler.centralChanged(test.currentCentral)
			require.NoError(t, err)
			assert.Equal(t, test.want, got)
		})
	}
}

func TestNamespaceLabelsAreSet(t *testing.T) {
	fakeClient := testutils.NewFakeClientBuilder(t).Build()
	r := NewCentralReconciler(fakeClient, private.ManagedCentral{}, nil, centralDBInitFunc, CentralReconcilerOptions{UseRoutes: true})

	_, err := r.Reconcile(context.TODO(), simpleManagedCentral)
	require.NoError(t, err)

	namespace := &v1.Namespace{}
	err = fakeClient.Get(context.TODO(), client.ObjectKey{Name: centralNamespace}, namespace)
	require.NoError(t, err)
	assert.Equal(t, simpleManagedCentral.Id, namespace.GetLabels()[tenantIDLabelKey])
	assert.Equal(t, simpleManagedCentral.Spec.Auth.OwnerOrgId, namespace.GetLabels()[orgIDLabelKey])
}

func TestReportRoutesStatuses(t *testing.T) {
	fakeClient := testutils.NewFakeClientBuilder(t).Build()
	r := NewCentralReconciler(fakeClient, private.ManagedCentral{}, nil, centralDBInitFunc, CentralReconcilerOptions{UseRoutes: true})

	status, err := r.Reconcile(context.TODO(), simpleManagedCentral)
	require.NoError(t, err)

	expected := []private.DataPlaneCentralStatusRoutes{
		{
			Domain: "acs-cb45idheg5ip6dq1jo4g.acs.rhcloud.test",
			Router: "router-default.apps.test.local",
		},
		{
			Domain: "acs-data-cb45idheg5ip6dq1jo4g.acs.rhcloud.test",
			Router: "router-default.apps.test.local",
		},
	}
	actual := status.Routes
	assert.ElementsMatch(t, expected, actual)
}

func TestChartResourcesAreAddedAndRemoved(t *testing.T) {
	chartFiles, err := charts.TraverseChart(testdata, "testdata/tenant-resources")
	require.NoError(t, err)
	chart, err := loader.LoadFiles(chartFiles)
	require.NoError(t, err)

	fakeClient := testutils.NewFakeClientBuilder(t).Build()
	r := NewCentralReconciler(fakeClient, private.ManagedCentral{}, nil, centralDBInitFunc, CentralReconcilerOptions{})
	r.resourcesChart = chart

	_, err = r.Reconcile(context.TODO(), simpleManagedCentral)
	require.NoError(t, err)

	var dummySvc v1.Service
	dummySvcKey := client.ObjectKey{Namespace: simpleManagedCentral.Metadata.Namespace, Name: "dummy"}
	err = fakeClient.Get(context.TODO(), dummySvcKey, &dummySvc)
	assert.NoError(t, err)

	assert.Equal(t, k8s.ManagedByFleetshardValue, dummySvc.GetLabels()[k8s.ManagedByLabelKey])

	deletedCentral := simpleManagedCentral
	deletedCentral.Metadata.DeletionTimestamp = time.Now().Format(time.RFC3339)

	_, err = r.Reconcile(context.TODO(), deletedCentral)
	for i := 0; i < 3 && errors.Is(err, ErrDeletionInProgress); i++ {
		_, err = r.Reconcile(context.TODO(), deletedCentral)
	}
	require.NoError(t, err)

	err = fakeClient.Get(context.TODO(), dummySvcKey, &dummySvc)
	assert.True(t, k8sErrors.IsNotFound(err))
}

func TestChartResourcesAreAddedAndUpdated(t *testing.T) {
	chartFiles, err := charts.TraverseChart(testdata, "testdata/tenant-resources")
	require.NoError(t, err)
	chart, err := loader.LoadFiles(chartFiles)
	require.NoError(t, err)

	fakeClient := testutils.NewFakeClientBuilder(t).Build()
	r := NewCentralReconciler(fakeClient, private.ManagedCentral{}, nil, centralDBInitFunc, CentralReconcilerOptions{})
	r.resourcesChart = chart

	_, err = r.Reconcile(context.TODO(), simpleManagedCentral)
	require.NoError(t, err)

	var dummySvc v1.Service
	dummySvcKey := client.ObjectKey{Namespace: simpleManagedCentral.Metadata.Namespace, Name: "dummy"}
	err = fakeClient.Get(context.TODO(), dummySvcKey, &dummySvc)
	assert.NoError(t, err)

	dummySvc.SetAnnotations(map[string]string{"dummy-annotation": "test"})
	err = fakeClient.Update(context.TODO(), &dummySvc)
	assert.NoError(t, err)

	err = fakeClient.Get(context.TODO(), dummySvcKey, &dummySvc)
	assert.NoError(t, err)
	assert.Equal(t, "test", dummySvc.GetAnnotations()["dummy-annotation"])

	_, err = r.Reconcile(context.TODO(), simpleManagedCentral)
	require.NoError(t, err)
	err = fakeClient.Get(context.TODO(), dummySvcKey, &dummySvc)
	assert.NoError(t, err)

	// verify that the chart resource was updated, by checking that the manually added annotation
	// is no longer present
	assert.Equal(t, "", dummySvc.GetAnnotations()["dummy-annotation"])
}

func TestEgressProxyIsDeployed(t *testing.T) {
	fakeClient := testutils.NewFakeClientBuilder(t).Build()
	r := NewCentralReconciler(fakeClient, private.ManagedCentral{}, nil, centralDBInitFunc, CentralReconcilerOptions{})

	_, err := r.Reconcile(context.TODO(), simpleManagedCentral)
	require.NoError(t, err)

	expectedObjs := []client.Object{
		&v1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: simpleManagedCentral.Metadata.Namespace,
				Name:      "egress-proxy-config",
			},
		},
		&v1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: simpleManagedCentral.Metadata.Namespace,
				Name:      "egress-proxy",
			},
		},
		&appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: simpleManagedCentral.Metadata.Namespace,
				Name:      "egress-proxy",
			},
		},
		&networkingv1.NetworkPolicy{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: simpleManagedCentral.Metadata.Namespace,
				Name:      "egress-proxy",
			},
		},
	}

	for _, expectedObj := range expectedObjs {
		actualObj := expectedObj.DeepCopyObject().(client.Object)
		if !assert.NoError(t, fakeClient.Get(context.TODO(), client.ObjectKeyFromObject(expectedObj), actualObj)) {
			continue
		}
		assert.Equal(t, k8s.ManagedByFleetshardValue, actualObj.GetLabels()[k8s.ManagedByLabelKey])

		if dep, ok := actualObj.(*appsv1.Deployment); ok {
			t.Run("verify deployment has desired properties", func(t *testing.T) {
				require.Len(t, dep.Spec.Template.Spec.Containers, 1, "expected exactly 1 container")
				assert.NotEmpty(t, dep.Spec.Template.Spec.Containers[0].Image, "container should define an image to be used")
			})
		}
	}
}

func TestEgressProxyCustomImage(t *testing.T) {
	fakeClient := testutils.NewFakeClientBuilder(t).Build()
	r := NewCentralReconciler(fakeClient, private.ManagedCentral{}, nil, centralDBInitFunc,
		CentralReconcilerOptions{
			EgressProxyImage: "registry.redhat.io/openshift4/ose-egress-http-proxy:version-for-test",
		})

	_, err := r.Reconcile(context.TODO(), simpleManagedCentral)
	require.NoError(t, err)

	dep := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: simpleManagedCentral.Metadata.Namespace,
			Name:      "egress-proxy",
		},
	}

	err = fakeClient.Get(context.TODO(), client.ObjectKeyFromObject(dep), dep)
	require.NoError(t, err)

	containers := dep.Spec.Template.Spec.Containers
	require.Len(t, containers, 1)

	assert.Equal(t, "registry.redhat.io/openshift4/ose-egress-http-proxy:version-for-test", containers[0].Image)
}

func TestNoRoutesSentWhenOneNotCreated(t *testing.T) {
	fakeClient, tracker := testutils.NewFakeClientWithTracker(t)
	tracker.AddRouteError(centralReencryptRouteName, errors.New("fake error"))
	r := NewCentralReconciler(fakeClient, private.ManagedCentral{}, nil, centralDBInitFunc, CentralReconcilerOptions{UseRoutes: true})
	_, err := r.Reconcile(context.TODO(), simpleManagedCentral)
	require.Errorf(t, err, "fake error")
}

func TestNoRoutesSentWhenOneNotAdmitted(t *testing.T) {
	fakeClient, tracker := testutils.NewFakeClientWithTracker(t)
	tracker.SetRouteAdmitted(centralReencryptRouteName, false)
	r := NewCentralReconciler(fakeClient, private.ManagedCentral{}, nil, centralDBInitFunc, CentralReconcilerOptions{UseRoutes: true})
	_, err := r.Reconcile(context.TODO(), simpleManagedCentral)
	require.Errorf(t, err, "unable to find admitted ingress")
}

func TestNoRoutesSentWhenOneNotCreatedYet(t *testing.T) {
	fakeClient, tracker := testutils.NewFakeClientWithTracker(t)
	tracker.SetSkipRoute(centralReencryptRouteName, true)
	r := NewCentralReconciler(fakeClient, private.ManagedCentral{}, nil, centralDBInitFunc, CentralReconcilerOptions{UseRoutes: true})
	_, err := r.Reconcile(context.TODO(), simpleManagedCentral)
	require.Errorf(t, err, "unable to find admitted ingress")
}

func centralDeploymentObject() *appsv1.Deployment {
	return testutils.NewCentralDeployment(centralNamespace)
}

func TestTelemetryOptionsAreSetInCR(t *testing.T) {
	tt := []struct {
		testName  string
		telemetry config.Telemetry
		enabled   bool
	}{
		{
			testName:  "endpoint and storage key not empty",
			telemetry: config.Telemetry{StorageEndpoint: "https://dummy.endpoint", StorageKey: "dummy-key"},
			enabled:   true,
		},
		{
			testName:  "endpoint not empty; storage key empty",
			telemetry: config.Telemetry{StorageEndpoint: "https://dummy.endpoint", StorageKey: ""},
			enabled:   false,
		},
		{
			testName:  "endpoint empty; storage key not empty",
			telemetry: config.Telemetry{StorageEndpoint: "", StorageKey: "dummy-key"},
			enabled:   true,
		},
	}
	for _, tc := range tt {
		t.Run(tc.testName, func(t *testing.T) {
			fakeClient := testutils.NewFakeClientBuilder(t).Build()
			r := NewCentralReconciler(fakeClient, private.ManagedCentral{}, nil, centralDBInitFunc, CentralReconcilerOptions{Telemetry: tc.telemetry})

			_, err := r.Reconcile(context.TODO(), simpleManagedCentral)
			require.NoError(t, err)
			central := &v1alpha1.Central{}
			err = fakeClient.Get(context.TODO(), client.ObjectKey{Name: centralName, Namespace: centralNamespace}, central)
			require.NoError(t, err)

			require.NotNil(t, central.Spec.Central.Telemetry.Enabled)
			assert.Equal(t, tc.enabled, *central.Spec.Central.Telemetry.Enabled)
			require.NotNil(t, central.Spec.Central.Telemetry.Storage.Endpoint)
			assert.Equal(t, tc.telemetry.StorageEndpoint, *central.Spec.Central.Telemetry.Storage.Endpoint)
			require.NotNil(t, central.Spec.Central.Telemetry.Storage.Key)
			assert.Equal(t, tc.telemetry.StorageKey, *central.Spec.Central.Telemetry.Storage.Key)
		})
	}
}

func TestReconcileUpdatesRoutes(t *testing.T) {

	tt := []struct {
		testName                string
		expectedReencryptHost   string
		expectedPassthroughHost string
		expectedTLSCert         string
		expectedTLSKey          string
	}{
		{
			testName:                "should update reencrypt route with TLS cert changes",
			expectedReencryptHost:   simpleManagedCentral.Spec.UiEndpoint.Host,
			expectedPassthroughHost: simpleManagedCentral.Spec.DataEndpoint.Host,
			expectedTLSCert:         "new-tls-cert-data",
			expectedTLSKey:          simpleManagedCentral.Spec.UiEndpoint.Tls.Key,
		},
		{
			testName:                "should update reencrypt route with TLS key changes",
			expectedReencryptHost:   simpleManagedCentral.Spec.UiEndpoint.Host,
			expectedPassthroughHost: simpleManagedCentral.Spec.DataEndpoint.Host,
			expectedTLSCert:         simpleManagedCentral.Spec.UiEndpoint.Tls.Cert,
			expectedTLSKey:          "new-tls-key-data",
		},
		{
			testName:                "should update reencrypt route with host name changes",
			expectedReencryptHost:   "new-hostname.acs.test",
			expectedPassthroughHost: simpleManagedCentral.Spec.DataEndpoint.Host,
			expectedTLSCert:         simpleManagedCentral.Spec.UiEndpoint.Tls.Cert,
			expectedTLSKey:          simpleManagedCentral.Spec.UiEndpoint.Tls.Key,
		},
		{
			testName:                "should update passthrough route with host name changes",
			expectedReencryptHost:   simpleManagedCentral.Spec.UiEndpoint.Host,
			expectedPassthroughHost: "new-hostname.acs.test",
			expectedTLSCert:         simpleManagedCentral.Spec.UiEndpoint.Tls.Cert,
			expectedTLSKey:          simpleManagedCentral.Spec.UiEndpoint.Tls.Key,
		},
	}

	for _, tc := range tt {
		t.Run(tc.testName, func(t *testing.T) {
			fakeClient := testutils.NewFakeClientBuilder(t).Build()
			r := NewCentralReconciler(fakeClient, private.ManagedCentral{}, nil, centralDBInitFunc, CentralReconcilerOptions{UseRoutes: true})
			r.routeService = k8s.NewRouteService(fakeClient)
			central := simpleManagedCentral

			// create the initial reencrypt route
			_, err := r.Reconcile(context.Background(), central)
			require.NoError(t, err)

			// test that initial routes were created to make sure we update and not create in the next step
			reencryptRoute := &openshiftRouteV1.Route{}
			err = fakeClient.Get(context.Background(), client.ObjectKey{Namespace: central.Metadata.Namespace, Name: "managed-central-reencrypt"}, reencryptRoute)
			require.NoError(t, err)
			passthroughRoute := &openshiftRouteV1.Route{}
			err = fakeClient.Get(context.Background(), client.ObjectKey{Namespace: central.Metadata.Namespace, Name: "managed-central-passthrough"}, passthroughRoute)
			require.NoError(t, err)

			central.Spec.UiEndpoint.Host = tc.expectedReencryptHost
			central.Spec.UiEndpoint.Tls.Cert = tc.expectedTLSCert
			central.Spec.UiEndpoint.Tls.Key = tc.expectedTLSKey
			central.Spec.DataEndpoint.Host = tc.expectedPassthroughHost

			// run another reconcile to update the route
			_, err = r.Reconcile(context.Background(), central)
			require.NoError(t, err)

			err = fakeClient.Get(context.Background(), client.ObjectKey{Namespace: central.Metadata.Namespace, Name: "managed-central-reencrypt"}, reencryptRoute)
			require.NoError(t, err)
			err = fakeClient.Get(context.Background(), client.ObjectKey{Namespace: central.Metadata.Namespace, Name: "managed-central-passthrough"}, passthroughRoute)
			require.NoError(t, err)

			require.Equal(t, tc.expectedReencryptHost, reencryptRoute.Spec.Host)
			require.Equal(t, tc.expectedTLSCert, reencryptRoute.Spec.TLS.Certificate)
			require.Equal(t, tc.expectedTLSKey, reencryptRoute.Spec.TLS.Key)
			require.Equal(t, tc.expectedPassthroughHost, passthroughRoute.Spec.Host)

		})
	}
}

func Test_centralNeedsUpdating(t *testing.T) {
	var scheme = runtime.NewScheme()
	utils.Must(clientgoscheme.AddToScheme(scheme))
	utils.Must(v1alpha1.AddToScheme(scheme))

	var central1 *v1alpha1.Central
	var central2 *v1alpha1.Central

	setup := func() {
		objectMeta := metav1.ObjectMeta{
			Name:      "test",
			Namespace: "test",
		}
		centralSpec1 := v1alpha1.CentralSpec{
			Central: &v1alpha1.CentralComponentSpec{
				DeploymentSpec: v1alpha1.DeploymentSpec{
					Resources: &v1.ResourceRequirements{
						Limits: v1.ResourceList{
							v1.ResourceCPU: resource.MustParse("1"),
						},
					},
				},
			},
		}
		centralSpec2 := v1alpha1.CentralSpec{
			Central: &v1alpha1.CentralComponentSpec{
				DeploymentSpec: v1alpha1.DeploymentSpec{
					Resources: &v1.ResourceRequirements{
						Limits: v1.ResourceList{
							v1.ResourceCPU: resource.MustParse("2"),
						},
					},
				},
			},
		}
		central1 = &v1alpha1.Central{
			ObjectMeta: objectMeta,
			Spec:       centralSpec1,
		}
		central2 = &v1alpha1.Central{
			ObjectMeta: objectMeta,
			Spec:       centralSpec2,
		}
	}

	t.Run("when desired is equal to existing Central, no upgrades are required", func(t *testing.T) {
		setup()
		existing := central1.DeepCopy()
		existing.Annotations = map[string]string{"test": "test"}
		existing.Labels = map[string]string{"test": "test"}
		desired := central1.DeepCopy()
		desired.Annotations = map[string]string{"test": "test"}
		desired.Labels = map[string]string{"test": "test"}
		cli := fake.NewClientBuilder().WithScheme(scheme).WithObjects(existing).Build()

		got, err := centralNeedsUpdating(context.Background(), cli, existing, desired)

		require.NoError(t, err)
		require.False(t, got)
	})
	t.Run("when desired Central spec is different from existing spec, an upgrade is required", func(t *testing.T) {
		setup()
		existing := central1.DeepCopy()
		desired := central2.DeepCopy()
		cli := fake.NewClientBuilder().WithScheme(scheme).WithObjects(central1).Build()

		got, err := centralNeedsUpdating(context.Background(), cli, existing, desired)

		require.NoError(t, err)
		require.True(t, got)
	})
	t.Run("when existing central is missing an annotation, an upgrade is required", func(t *testing.T) {
		setup()
		existing := central1.DeepCopy()
		existing.Annotations = map[string]string{"foo": "bar"}
		desired := central1.DeepCopy()
		desired.Annotations = map[string]string{"foo": "bar", "bar": "baz"}
		cli := fake.NewClientBuilder().WithScheme(scheme).WithObjects(central1).Build()

		got, err := centralNeedsUpdating(context.Background(), cli, existing, desired)

		require.NoError(t, err)
		require.True(t, got)
	})
	t.Run("when existing central is missing a label, an update is required", func(t *testing.T) {
		setup()
		existing := central1.DeepCopy()
		existing.Labels = map[string]string{"foo": "bar"}
		desired := central1.DeepCopy()
		desired.Labels = map[string]string{"foo": "bar", "bar": "baz"}
		cli := fake.NewClientBuilder().WithScheme(scheme).WithObjects(central1).Build()

		got, err := centralNeedsUpdating(context.Background(), cli, existing, desired)

		require.NoError(t, err)
		require.True(t, got)
	})
	t.Run("when existing central has extra annotations, no upgrade is required", func(t *testing.T) {
		setup()
		existing := central1.DeepCopy()
		existing.Annotations = map[string]string{"foo": "bar", "bar": "baz"}
		desired := central1.DeepCopy()
		desired.Annotations = map[string]string{"foo": "bar"}
		cli := fake.NewClientBuilder().WithScheme(scheme).WithObjects(central1).Build()

		got, err := centralNeedsUpdating(context.Background(), cli, existing, desired)

		require.NoError(t, err)
		require.False(t, got)
	})
	t.Run("when existing central has extra labels, no upgrade is required", func(t *testing.T) {
		setup()
		existing := central1.DeepCopy()
		existing.Labels = map[string]string{"foo": "bar", "bar": "baz"}
		desired := central1.DeepCopy()
		desired.Labels = map[string]string{"foo": "bar"}
		cli := fake.NewClientBuilder().WithScheme(scheme).WithObjects(central1).Build()

		got, err := centralNeedsUpdating(context.Background(), cli, existing, desired)

		require.NoError(t, err)
		require.False(t, got)
	})
	t.Run("when existing central is not missing labels, no upgrade is required", func(t *testing.T) {
		setup()
		existing := central1.DeepCopy()
		existing.Labels = map[string]string{"foo": "bar"}
		desired := central1.DeepCopy()
		desired.Labels = map[string]string{"foo": "bar"}
		cli := fake.NewClientBuilder().WithScheme(scheme).WithObjects(central1).Build()

		got, err := centralNeedsUpdating(context.Background(), cli, existing, desired)

		require.NoError(t, err)
		require.False(t, got)
	})

	t.Run("when existing central is not missing annotations, no upgrade is required", func(t *testing.T) {
		setup()
		existing := central1.DeepCopy()
		existing.Annotations = map[string]string{"foo": "bar"}
		desired := central1.DeepCopy()
		desired.Annotations = map[string]string{"foo": "bar"}
		cli := fake.NewClientBuilder().WithScheme(scheme).WithObjects(central1).Build()

		got, err := centralNeedsUpdating(context.Background(), cli, existing, desired)

		require.NoError(t, err)
		require.False(t, got)
	})

}

func Test_mergeLabelsAndAnnotations(t *testing.T) {

	var from *v1alpha1.Central
	var into *v1alpha1.Central

	setup := func() {
		from = &v1alpha1.Central{}
		into = &v1alpha1.Central{}
	}

	t.Run("when from annotations is nil", func(t *testing.T) {
		setup()
		from.Annotations = nil
		into.Annotations = map[string]string{"bar": "baz"}
		mergeLabelsAndAnnotations(from, into)
		require.Equal(t, map[string]string{"bar": "baz"}, into.Annotations)
	})
	t.Run("when from annotations is empty", func(t *testing.T) {
		setup()
		from.Annotations = map[string]string{}
		into.Annotations = map[string]string{"bar": "baz"}
		mergeLabelsAndAnnotations(from, into)
		require.Equal(t, map[string]string{"bar": "baz"}, into.Annotations)
	})
	t.Run("when from annotations has values", func(t *testing.T) {
		setup()
		from.Annotations = map[string]string{"foo": "bar"}
		into.Annotations = map[string]string{"bar": "baz"}
		mergeLabelsAndAnnotations(from, into)
		require.Equal(t, map[string]string{"foo": "bar", "bar": "baz"}, into.Annotations)
	})
	t.Run("when from labels is nil", func(t *testing.T) {
		setup()
		from.Labels = nil
		into.Labels = map[string]string{"bar": "baz"}
		mergeLabelsAndAnnotations(from, into)
		require.Equal(t, map[string]string{"bar": "baz"}, into.Labels)
	})
	t.Run("when from labels is empty", func(t *testing.T) {
		setup()
		from.Labels = map[string]string{}
		into.Labels = map[string]string{"bar": "baz"}
		mergeLabelsAndAnnotations(from, into)
		require.Equal(t, map[string]string{"bar": "baz"}, into.Labels)
	})
	t.Run("when from labels has values", func(t *testing.T) {
		setup()
		from.Labels = map[string]string{"foo": "bar"}
		into.Labels = map[string]string{"bar": "baz"}
		mergeLabelsAndAnnotations(from, into)
		require.Equal(t, map[string]string{"foo": "bar", "bar": "baz"}, into.Labels)
	})

}

func Test_stringMapNeedsUpdating(t *testing.T) {
	type tc struct {
		name    string
		actual  map[string]string
		desired map[string]string
		want    bool
	}
	tests := []tc{
		{
			name:    "both nil",
			desired: nil,
			actual:  nil,
			want:    false,
		}, {
			name:    "both empty",
			desired: map[string]string{},
			actual:  map[string]string{},
			want:    false,
		}, {
			name:    "desired nil",
			desired: nil,
			actual:  map[string]string{"foo": "bar"},
			want:    false,
		}, {
			name:    "actual nil",
			desired: map[string]string{"foo": "bar"},
			actual:  nil,
			want:    true,
		}, {
			name:    "desired empty",
			desired: map[string]string{},
			actual:  map[string]string{"foo": "bar"},
			want:    false,
		}, {
			name:    "actual empty",
			desired: map[string]string{"foo": "bar"},
			actual:  map[string]string{},
			want:    true,
		}, {
			name:    "desired has more keys",
			desired: map[string]string{"foo": "bar", "bar": "baz"},
			actual:  map[string]string{"foo": "bar"},
			want:    true,
		}, {
			name:    "actual has more keys",
			desired: map[string]string{"foo": "bar"},
			actual:  map[string]string{"foo": "bar", "bar": "baz"},
			want:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := stringMapNeedsUpdating(tt.desired, tt.actual)
			assert.Equal(t, tt.want, got, tt.name)
		})
	}
}
