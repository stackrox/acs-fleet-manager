package reconciler

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	argocd "github.com/argoproj/argo-cd/v3/pkg/apis/application/v1alpha1"

	"github.com/aws/smithy-go"
	openshiftRouteV1 "github.com/openshift/api/route/v1"
	"github.com/pkg/errors"
	"github.com/stackrox/acs-fleet-manager/fleetshard/config"
	"github.com/stackrox/acs-fleet-manager/fleetshard/pkg/central/cloudprovider"
	"github.com/stackrox/acs-fleet-manager/fleetshard/pkg/central/cloudprovider/awsclient"
	"github.com/stackrox/acs-fleet-manager/fleetshard/pkg/central/postgres"
	"github.com/stackrox/acs-fleet-manager/fleetshard/pkg/cipher"
	"github.com/stackrox/acs-fleet-manager/fleetshard/pkg/k8s"
	"github.com/stackrox/acs-fleet-manager/fleetshard/pkg/testutils"
	"github.com/stackrox/acs-fleet-manager/fleetshard/pkg/util"
	centralConstants "github.com/stackrox/acs-fleet-manager/internal/central/constants"
	"github.com/stackrox/acs-fleet-manager/internal/central/pkg/api/private"
	"github.com/stackrox/acs-fleet-manager/pkg/client/fleetmanager"
	fmMocks "github.com/stackrox/acs-fleet-manager/pkg/client/fleetmanager/mocks"
	centralNotifierUtils "github.com/stackrox/rox/central/notifiers/utils"
	"github.com/stackrox/rox/pkg/declarativeconfig"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v2"
	v1 "k8s.io/api/core/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
)

const (
	centralName               = "test-central"
	centralID                 = "cb45idheg5ip6dq1jo4g"
	centralNamespace          = "rhacs-" + centralID
	centralArgoCDAppName      = "rhacs-" + centralID
	openshiftGitopsNamespace  = "openshift-gitops"
	centralReencryptRouteName = "managed-central-reencrypt"
	conditionTypeReady        = "Ready"
	clusterName               = "test-cluster"
	environment               = "test"
)

var (
	defaultReconcilerOptions = CentralReconcilerOptions{
		ClusterName: clusterName,
		Environment: environment,
		UseRoutes:   true,
		ArgoReconcilerOptions: ArgoReconcilerOptions{
			ArgoCdNamespace: "openshift-gitops",
		},
	}

	useRoutesReconcilerOptions = CentralReconcilerOptions{
		UseRoutes: true,
		ArgoReconcilerOptions: ArgoReconcilerOptions{
			ArgoCdNamespace: "openshift-gitops",
		},
	}

	defaultAuditLogConfig = config.AuditLogging{
		Enabled:            true,
		URLScheme:          "https",
		AuditLogTargetHost: "audit-logs-aggregator.rhacs-audit-logs",
		AuditLogTargetPort: 8888,
		SkipTLSVerify:      true,
	}

	vectorAuditLogConfig = config.AuditLogging{
		Enabled:            true,
		URLScheme:          "https",
		AuditLogTargetHost: "rhacs-vector.rhacs",
		AuditLogTargetPort: 8443,
		SkipTLSVerify:      true,
	}

	disabledAuditLogConfig = config.AuditLogging{
		Enabled:            false,
		URLScheme:          "https",
		AuditLogTargetHost: "audit-logs-aggregator.rhacs-audit-logs",
		AuditLogTargetPort: 8888,
		SkipTLSVerify:      false,
	}
)

var simpleManagedCentral = private.ManagedCentral{
	Id: centralID,
	Metadata: private.ManagedCentralAllOfMetadata{
		Name:      centralName,
		Namespace: centralNamespace,
	},
	Spec: private.ManagedCentralAllOfSpec{
		Auth: private.ManagedCentralAllOfSpecAuth{
			ClientSecret: "test-value", // pragma: allowlist secret
			ClientId:     "test-value",
			OwnerUserId:  "54321",
			OwnerOrgId:   "12345",
			OwnerOrgName: "org-name",
			Issuer:       "https://example.com",
		},
		UiHost:       fmt.Sprintf("acs-%s.acs.rhcloud.test", centralID),
		DataHost:     fmt.Sprintf("acs-data-%s.acs.rhcloud.test", centralID),
		InstanceType: "standard",
	},
}

func createBase64Cipher(t *testing.T) cipher.Cipher {
	b64Cipher, err := cipher.NewLocalBase64Cipher()
	require.NoError(t, err, "creating base64 cipher for test")
	return b64Cipher
}

func getClientTrackerAndReconciler(
	t *testing.T,
	managedDBClient cloudprovider.DBClient,
	reconcilerOptions CentralReconcilerOptions,
	k8sObjects ...client.Object,
) (client.WithWatch, *testutils.ReconcileTracker, *CentralReconciler) {
	fakeClient, tracker := testutils.NewFakeClientWithTracker(t, k8sObjects...)
	reconciler := NewCentralReconciler(
		fakeClient,
		fmMocks.NewClientMock().Client(),
		managedDBClient,
		centralDBInitFunc,
		createBase64Cipher(t),
		cipher.AES256KeyGenerator{},
		reconcilerOptions,
	)
	return fakeClient, tracker, reconciler
}

func centralDBInitFunc(_ context.Context, _ postgres.DBConnection, _, _ string) error {
	return nil
}

func centralTLSSecretObject() *v1.Secret {
	return &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "central-tls",
			Namespace: centralNamespace,
		},
		Data: map[string][]byte{
			"ca.pem": []byte("dummy-ca"),
		},
	}
}

func centralDBPasswordSecretObject() *v1.Secret {
	return &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "central-db-password",
			Namespace: centralNamespace,
		},
		Data: map[string][]byte{
			"password": []byte("dummy-password"),
		},
	}
}

func centralEncryptionKeySecretObject() *v1.Secret {
	return &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      centralEncryptionKeySecretName,
			Namespace: centralNamespace,
		},
	}
}

func conditionForType(conditions []private.DataPlaneCentralStatusConditions, conditionType string) (*private.DataPlaneCentralStatusConditions, bool) {
	for _, c := range conditions {
		if c.Type == conditionType {
			return &c, true
		}
	}
	return nil, false
}

func TestReconcileCreate(t *testing.T) {
	fakeClient, _, r := getClientTrackerAndReconciler(
		t,
		nil,
		defaultReconcilerOptions,
	)

	status, err := r.Reconcile(context.TODO(), simpleManagedCentral)
	require.NoError(t, err)

	readyCondition, ok := conditionForType(status.Conditions, conditionTypeReady)
	require.True(t, ok)
	assert.Equal(t, "True", readyCondition.Status, "Ready condition not found in conditions", status.Conditions)

	app := &argocd.Application{}
	err = fakeClient.Get(context.TODO(), client.ObjectKey{Name: centralArgoCDAppName, Namespace: openshiftGitopsNamespace}, app)
	require.NoError(t, err)

	route := &openshiftRouteV1.Route{}
	err = fakeClient.Get(context.TODO(), client.ObjectKey{Name: centralReencryptRouteName, Namespace: centralNamespace}, route)
	require.NoError(t, err)
	assert.Equal(t, centralReencryptRouteName, route.GetName())
	assert.Equal(t, openshiftRouteV1.TLSTerminationReencrypt, route.Spec.TLS.Termination)
	assert.Equal(t, testutils.CentralCA, route.Spec.TLS.DestinationCACertificate)
}

func TestReconcileCreateWithManagedDB(t *testing.T) {
	managedDBProvisioningClient := &cloudprovider.DBClientMock{}
	managedDBProvisioningClient.EnsureDBProvisionedFunc = func(_ context.Context, _string, _ string, _ string, _ bool) error {
		return nil
	}
	managedDBProvisioningClient.GetDBConnectionFunc = func(_ string) (postgres.DBConnection, error) {
		connection, err := postgres.NewDBConnection("localhost", 5432, "rhacs", "postgres")
		if err != nil {
			return postgres.DBConnection{}, err
		}
		return connection, nil
	}

	reconcilerOptions := defaultReconcilerOptions
	reconcilerOptions.ManagedDBEnabled = true

	fakeClient, _, r := getClientTrackerAndReconciler(t, managedDBProvisioningClient, reconcilerOptions)

	status, err := r.Reconcile(context.TODO(), simpleManagedCentral)
	require.NoError(t, err)
	assert.Len(t, managedDBProvisioningClient.EnsureDBProvisionedCalls(), 1)

	readyCondition, ok := conditionForType(status.Conditions, conditionTypeReady)
	require.True(t, ok)
	assert.Equal(t, "True", readyCondition.Status, "Ready condition not found in conditions", status.Conditions)

	app := &argocd.Application{}
	err = fakeClient.Get(context.TODO(), client.ObjectKey{Name: centralArgoCDAppName, Namespace: openshiftGitopsNamespace}, app)
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

func TestReconcileCreateWithManagedDBNoCredentials(t *testing.T) {
	t.Setenv("AWS_ACCESS_KEY", "")
	t.Setenv("AWS_SECRET_ACCESS_KEY", "")
	t.Setenv("AWS_REGION", "us-east-1")
	t.Setenv("AWS_ROLE_ARN", "arn:aws:iam::012456789:role/fake_role")
	t.Setenv("AWS_WEB_IDENTITY_TOKEN_FILE", "/var/run/secrets/tokens/aws-token")

	managedDBProvisioningClient, err := awsclient.NewRDSClient(
		&config.Config{
			ManagedDB: config.ManagedDB{
				SecurityGroup: "security-group",
				SubnetGroup:   "db-group",
			},
		})
	require.NoError(t, err)

	reconcilerOptions := CentralReconcilerOptions{
		UseRoutes:        true,
		ManagedDBEnabled: true,
	}
	_, _, r := getClientTrackerAndReconciler(
		t,
		managedDBProvisioningClient,
		reconcilerOptions,
	)

	_, err = r.Reconcile(context.TODO(), simpleManagedCentral)
	var awsErr *smithy.OperationError
	require.ErrorAs(t, err, &awsErr)
	assert.Equal(t, awsErr.ServiceID, "RDS")
	assert.Equal(t, awsErr.OperationName, "DescribeDBClusters")
	assert.Contains(t, awsErr.Unwrap().Error(), "/var/run/secrets/tokens/aws-token: no such file")
}

func TestReconcileUpdateSucceeds(t *testing.T) {
	_, _, r := getClientTrackerAndReconciler(t, nil, defaultReconcilerOptions)
	status, err := r.Reconcile(context.TODO(), simpleManagedCentral)
	require.NoError(t, err)
	assert.Equal(t, "True", status.Conditions[0].Status)
}

func TestReconcileLastHashNotUpdatedOnError(t *testing.T) {
	_, _, r := getClientTrackerAndReconciler(t, nil, defaultReconcilerOptions)
	r.restoreCentralSecretsFunc = func(ctx context.Context, remoteCentral private.ManagedCentral) error {
		return errors.New("dummy")
	}

	_, err := r.Reconcile(context.TODO(), simpleManagedCentral)
	require.Error(t, err)

	assert.Equal(t, [16]byte{}, r.lastCentralHash)
}

func TestReconcileLastHashSetOnSuccess(t *testing.T) {
	_, _, r := getClientTrackerAndReconciler(t, nil, defaultReconcilerOptions, defaultObjects()...)

	managedCentral := simpleManagedCentral
	managedCentral.RequestStatus = centralConstants.CentralRequestStatusReady.String()
	managedCentral.Metadata.SecretsStored = r.secretBackup.GetWatchedSecrets()
	expectedHash, err := util.MD5SumFromJSONStruct(&managedCentral)
	require.NoError(t, err)

	_, err = r.Reconcile(context.TODO(), managedCentral)
	require.NoError(t, err)

	assert.Equal(t, expectedHash, r.lastCentralHash)

	status, err := r.Reconcile(context.TODO(), managedCentral)
	require.Nil(t, status)
	require.ErrorIs(t, err, ErrCentralNotChanged)
}

func TestReconcileLastHashSecretsOrderIndependent(t *testing.T) {
	_, _, r := getClientTrackerAndReconciler(t, nil, defaultReconcilerOptions, defaultObjects()...)

	managedCentral := simpleManagedCentral
	managedCentral.RequestStatus = centralConstants.CentralRequestStatusReady.String()
	managedCentral.Metadata.SecretsStored = []string{"central-tls", "central-db-password"}

	expectedHash, err := util.MD5SumFromJSONStruct(&managedCentral)
	require.NoError(t, err)

	_, err = r.Reconcile(context.TODO(), managedCentral)
	require.NoError(t, err)
	assert.Equal(t, expectedHash, r.lastCentralHash, "Order of stored secrets should not impact hash.")
}

func TestIgnoreCacheForCentralNotReady(t *testing.T) {
	_, _, r := getClientTrackerAndReconciler(t, nil, defaultReconcilerOptions)

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
	_, _, r := getClientTrackerAndReconciler(t, nil, defaultReconcilerOptions, defaultObjects()...)

	managedCentral := simpleManagedCentral
	managedCentral.RequestStatus = centralConstants.CentralRequestStatusReady.String()
	managedCentral.Spec.TenantResourcesValues = map[string]interface{}{
		"forceReconcile": true,
	}

	expectedHash, err := util.MD5SumFromJSONStruct(&managedCentral)
	require.NoError(t, err)

	_, err = r.Reconcile(context.TODO(), managedCentral)
	require.NoError(t, err)
	assert.Equal(t, expectedHash, r.lastCentralHash)

	_, err = r.Reconcile(context.TODO(), managedCentral)
	require.NoError(t, err)
}

func TestReconcileDelete(t *testing.T) {
	fakeClient, _, r := getClientTrackerAndReconciler(t, nil, defaultReconcilerOptions)

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

	app := &argocd.Application{}
	err = fakeClient.Get(context.TODO(), client.ObjectKey{Name: centralArgoCDAppName, Namespace: openshiftGitopsNamespace}, app)
	assert.True(t, k8sErrors.IsNotFound(err))

	namespace := &v1.Namespace{}
	err = fakeClient.Get(context.TODO(), client.ObjectKey{Name: centralNamespace}, namespace)
	assert.True(t, k8sErrors.IsNotFound(err))
}

func TestReconcileDeleteWithManagedDB(t *testing.T) {
	managedDBProvisioningClient := &cloudprovider.DBClientMock{}
	managedDBProvisioningClient.EnsureDBProvisionedFunc = func(_ context.Context, databaseID, acsInstanceID, _ string, _ bool) error {
		require.Equal(t, databaseID, acsInstanceID)
		require.Equal(t, databaseID, simpleManagedCentral.Id)
		return nil
	}
	managedDBProvisioningClient.EnsureDBDeprovisionedFunc = func(databaseID string, _ bool) error {
		require.Equal(t, databaseID, simpleManagedCentral.Id)
		return nil
	}
	managedDBProvisioningClient.GetDBConnectionFunc = func(databaseID string) (postgres.DBConnection, error) {
		require.Equal(t, databaseID, simpleManagedCentral.Id)
		connection, err := postgres.NewDBConnection("localhost", 5432, "rhacs", "postgres")
		if err != nil {
			return postgres.DBConnection{}, err
		}
		return connection, nil
	}

	reconcilerOptions := defaultReconcilerOptions
	reconcilerOptions.ManagedDBEnabled = true
	fakeClient, _, r := getClientTrackerAndReconciler(t, managedDBProvisioningClient, reconcilerOptions)

	_, err := r.Reconcile(context.TODO(), simpleManagedCentral)
	require.NoError(t, err)
	assert.Len(t, managedDBProvisioningClient.EnsureDBProvisionedCalls(), 1)

	deletedCentral := simpleManagedCentral
	deletedCentral.Metadata.DeletionTimestamp = "2006-01-02T15:04:05+00:00"

	// trigger deletion
	managedDBProvisioningClient.EnsureDBProvisionedFunc = func(_ context.Context, _ string, _ string, _ string, _ bool) error {
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

	app := &argocd.Application{}
	err = fakeClient.Get(context.TODO(), client.ObjectKey{Name: centralArgoCDAppName, Namespace: openshiftGitopsNamespace}, app)
	assert.True(t, k8sErrors.IsNotFound(err))

	namespace := &v1.Namespace{}
	err = fakeClient.Get(context.TODO(), client.ObjectKey{Name: centralNamespace}, namespace)
	assert.True(t, k8sErrors.IsNotFound(err))
}

func TestReconcileDeleteWithManagedDBOverride(t *testing.T) {
	dbOverrideId := "override-1234"

	managedDBProvisioningClient := &cloudprovider.DBClientMock{}
	managedDBProvisioningClient.EnsureDBProvisionedFunc = func(_ context.Context, databaseID, acsInstanceID, _ string, _ bool) error {
		require.Equal(t, databaseID, dbOverrideId)
		require.Equal(t, acsInstanceID, simpleManagedCentral.Id)
		return nil
	}
	managedDBProvisioningClient.EnsureDBDeprovisionedFunc = func(databaseID string, _ bool) error {
		require.Equal(t, databaseID, dbOverrideId)
		return nil
	}
	managedDBProvisioningClient.GetDBConnectionFunc = func(databaseID string) (postgres.DBConnection, error) {
		require.Equal(t, databaseID, dbOverrideId)
		connection, err := postgres.NewDBConnection("localhost", 5432, "rhacs", "postgres")
		if err != nil {
			return postgres.DBConnection{}, err
		}
		return connection, nil
	}

	reconcilerOptions := defaultReconcilerOptions
	reconcilerOptions.UseRoutes = true
	reconcilerOptions.ManagedDBEnabled = true

	_, _, r := getClientTrackerAndReconciler(
		t,
		managedDBProvisioningClient,
		reconcilerOptions,
	)

	namespace := &v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: simpleManagedCentral.Metadata.Namespace,
		},
	}
	err := r.client.Create(context.TODO(), namespace)
	require.NoError(t, err)

	dbOverrideConfigMap := &v1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: simpleManagedCentral.Metadata.Namespace,
			Name:      centralDbOverrideConfigMap,
		},
		Data: map[string]string{"databaseID": dbOverrideId},
	}
	err = r.client.Create(context.TODO(), dbOverrideConfigMap)
	require.NoError(t, err)

	_, err = r.Reconcile(context.TODO(), simpleManagedCentral)
	require.NoError(t, err)
	assert.Len(t, managedDBProvisioningClient.EnsureDBProvisionedCalls(), 1)

	deletedCentral := simpleManagedCentral
	deletedCentral.Metadata.DeletionTimestamp = "2006-01-02T15:04:05+00:00"

	// trigger deletion
	managedDBProvisioningClient.EnsureDBProvisionedFunc = func(_ context.Context, _ string, _ string, _ string, _ bool) error {
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
					UiHost: "central.cluster.local",
				},
			},
			want: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, _, reconciler := getClientTrackerAndReconciler(t, nil, defaultReconcilerOptions, defaultObjects()...)

			if test.lastCentral != nil {
				centralHash, err := reconciler.computeCentralHash(*test.lastCentral)
				require.NoError(t, err)
				reconciler.setLastCentralHash(centralHash)
			}

			centralHash, err := reconciler.computeCentralHash(test.currentCentral)
			require.NoError(t, err)
			got := reconciler.centralChanged(centralHash)
			assert.Equal(t, test.want, got)
		})
	}
}

func TestNamespaceLabelsAreSet(t *testing.T) {
	fakeClient, _, r := getClientTrackerAndReconciler(
		t, nil,
		useRoutesReconcilerOptions,
	)

	_, err := r.Reconcile(context.TODO(), simpleManagedCentral)
	require.NoError(t, err)

	namespace := &v1.Namespace{}
	err = fakeClient.Get(context.TODO(), client.ObjectKey{Name: centralNamespace}, namespace)
	require.NoError(t, err)
	assert.Equal(t, simpleManagedCentral.Id, namespace.GetLabels()[TenantIDLabelKey])
	assert.Equal(t, simpleManagedCentral.Spec.Auth.OwnerOrgId, namespace.GetLabels()[orgIDLabelKey])
}

func TestNamespaceAnnotationsAreSet(t *testing.T) {
	fakeClient, _, r := getClientTrackerAndReconciler(
		t, nil,
		useRoutesReconcilerOptions,
	)

	_, err := r.Reconcile(context.TODO(), simpleManagedCentral)
	require.NoError(t, err)

	namespace := &v1.Namespace{}
	err = fakeClient.Get(context.TODO(), client.ObjectKey{Name: centralNamespace}, namespace)
	require.NoError(t, err)
	assert.Equal(t, ovnACLLoggingAnnotationDefault, namespace.GetAnnotations()[ovnACLLoggingAnnotationKey])
}

func TestReportRoutesStatuses(t *testing.T) {
	_, _, r := getClientTrackerAndReconciler(
		t, nil,
		useRoutesReconcilerOptions,
	)

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

func TestCentralEncryptionKeyIsGenerated(t *testing.T) {
	fakeClient, _, r := getClientTrackerAndReconciler(
		t, nil,
		defaultReconcilerOptions,
	)

	_, err := r.Reconcile(context.TODO(), simpleManagedCentral)
	require.NoError(t, err)

	var centralEncryptionSecret v1.Secret
	key := client.ObjectKey{Namespace: simpleManagedCentral.Metadata.Namespace, Name: centralEncryptionKeySecretName}
	err = fakeClient.Get(context.TODO(), key, &centralEncryptionSecret)
	require.NoError(t, err)
	require.Contains(t, centralEncryptionSecret.Data, "key-chain.yaml")

	var keyChain centralNotifierUtils.KeyChain
	err = yaml.Unmarshal(centralEncryptionSecret.Data["key-chain.yaml"], &keyChain)
	require.NoError(t, err)
	require.Equal(t, 0, keyChain.ActiveKeyIndex)
	require.Equal(t, 1, len(keyChain.KeyMap))

	encKey, err := base64.StdEncoding.DecodeString(keyChain.KeyMap[keyChain.ActiveKeyIndex])
	require.NoError(t, err)
	expectedKeyLen := 32 // 256 bits key
	require.Equal(t, expectedKeyLen, len(encKey))
}

func TestNoRoutesSentWhenOneNotCreated(t *testing.T) {
	_, tracker, r := getClientTrackerAndReconciler(
		t, nil,
		useRoutesReconcilerOptions,
	)
	tracker.AddRouteError(centralReencryptRouteName, errors.New("fake error"))
	_, err := r.Reconcile(context.TODO(), simpleManagedCentral)
	require.Errorf(t, err, "fake error")
}

func TestNoRoutesSentWhenOneNotAdmitted(t *testing.T) {
	_, tracker, r := getClientTrackerAndReconciler(
		t, nil,
		useRoutesReconcilerOptions,
	)
	tracker.SetRouteAdmitted(centralReencryptRouteName, false)
	_, err := r.Reconcile(context.TODO(), simpleManagedCentral)
	require.Errorf(t, err, "unable to find admitted ingress")
}

func TestNoRoutesSentWhenOneNotCreatedYet(t *testing.T) {
	_, tracker, r := getClientTrackerAndReconciler(
		t, nil,
		useRoutesReconcilerOptions,
	)
	tracker.SetSkipRoute(centralReencryptRouteName, true)
	_, err := r.Reconcile(context.TODO(), simpleManagedCentral)
	require.Errorf(t, err, "unable to find admitted ingress")
}

func centralAppObject() *argocd.Application {
	return &argocd.Application{
		ObjectMeta: metav1.ObjectMeta{
			Name:      centralArgoCDAppName,
			Namespace: openshiftGitopsNamespace,
		},
		Spec: argocd.ApplicationSpec{
			Destination: argocd.ApplicationDestination{
				Namespace: centralNamespace,
			},
			Source: &argocd.ApplicationSource{
				Helm: &argocd.ApplicationSourceHelm{
					ValuesObject: &runtime.RawExtension{
						Raw: []byte(`{"instanceName":"test-central"}`),
					},
				},
			},
		},
	}
}

func defaultObjects() []client.Object {
	centralApp := centralAppObject()
	tenantResources, _ := testutils.NewTenantResources(centralApp)

	var objects []client.Object

	objects = append(objects, centralApp)
	objects = append(objects, tenantResources.Objects()...)
	objects = append(objects, centralDBPasswordSecretObject())
	objects = append(objects, centralEncryptionKeySecretObject())

	return objects
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

			reconcilerOptions := defaultReconcilerOptions
			reconcilerOptions.ArgoReconcilerOptions.Telemetry = tc.telemetry
			fakeClient, _, r := getClientTrackerAndReconciler(t, nil, reconcilerOptions)

			_, err := r.Reconcile(context.TODO(), simpleManagedCentral)
			require.NoError(t, err)
			app := &argocd.Application{}
			err = fakeClient.Get(context.TODO(), client.ObjectKey{Name: centralArgoCDAppName, Namespace: openshiftGitopsNamespace}, app)
			require.NoError(t, err)

			values := map[string]interface{}{}
			require.NoError(t, json.Unmarshal(app.Spec.Source.Helm.ValuesObject.Raw, &values))

			assert.Equal(t, tc.telemetry.StorageEndpoint, values["telemetryStorageEndpoint"])
			if len(tc.telemetry.StorageKey) > 0 {
				assert.Equal(t, tc.telemetry.StorageKey, values["telemetryStorageKey"])
			} else {
				assert.Equal(t, "DISABLED", values["telemetryStorageKey"])
			}
		})
	}
}

func TestReconcileUpdatesRoutes(t *testing.T) {

	tt := []struct {
		testName                string
		expectedReencryptHost   string
		expectedPassthroughHost string
	}{
		{
			testName:                "should update reencrypt route with TLS cert changes",
			expectedReencryptHost:   simpleManagedCentral.Spec.UiHost,
			expectedPassthroughHost: simpleManagedCentral.Spec.DataHost,
		},
		{
			testName:                "should update reencrypt route with TLS key changes",
			expectedReencryptHost:   simpleManagedCentral.Spec.UiHost,
			expectedPassthroughHost: simpleManagedCentral.Spec.DataHost,
		},
		{
			testName:                "should update reencrypt route with host name changes",
			expectedReencryptHost:   "new-hostname.acs.test",
			expectedPassthroughHost: simpleManagedCentral.Spec.DataHost,
		},
		{
			testName:                "should update passthrough route with host name changes",
			expectedReencryptHost:   simpleManagedCentral.Spec.UiHost,
			expectedPassthroughHost: "new-hostname.acs.test",
		},
	}

	for _, tc := range tt {
		t.Run(tc.testName, func(t *testing.T) {
			fakeClient, _, r := getClientTrackerAndReconciler(
				t, nil,
				useRoutesReconcilerOptions,
			)
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

			central.Spec.UiHost = tc.expectedReencryptHost
			central.Spec.DataHost = tc.expectedPassthroughHost

			// run another reconcile to update the route
			_, err = r.Reconcile(context.Background(), central)
			require.NoError(t, err)

			err = fakeClient.Get(context.Background(), client.ObjectKey{Namespace: central.Metadata.Namespace, Name: "managed-central-reencrypt"}, reencryptRoute)
			require.NoError(t, err)
			err = fakeClient.Get(context.Background(), client.ObjectKey{Namespace: central.Metadata.Namespace, Name: "managed-central-passthrough"}, passthroughRoute)
			require.NoError(t, err)

			require.Equal(t, tc.expectedReencryptHost, reencryptRoute.Spec.Host)
			require.Equal(t, tc.expectedPassthroughHost, passthroughRoute.Spec.Host)

		})
	}
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

func getSecret(name string, namespace string, data map[string][]byte) *v1.Secret {
	return &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{ // pragma: allowlist secret
			Name:      name,
			Namespace: namespace,
		},
		Data: data,
	}
}

const (
	emptySecretName                   = "emptySecret"                   // pragma: allowlist secret
	secretWithOtherKeyName            = "secretWithOtherKey"            // pragma: allowlist secret
	secretWithKeyDataToChangeName     = "secretWithKeyDataToChange"     // pragma: allowlist secret
	secretWithExpectedKeyDataOnlyName = "secretWithExpectedKeyDataOnly" // pragma: allowlist secret

	entryKey = "some.key"
	otherKey = "other.key"
)

var (
	entryData = []byte("content")
	otherData = []byte("something else")
)

func compareSecret(t *testing.T, expectedSecret *v1.Secret, secret *v1.Secret, created bool) {
	if expectedSecret == nil { // pragma: allowlist secret
		assert.Nil(t, secret)
		return
	}
	require.NotNil(t, secret)
	assert.Equal(t, expectedSecret.ObjectMeta.Namespace, secret.ObjectMeta.Namespace) // pragma: allowlist secret
	assert.Equal(t, expectedSecret.ObjectMeta.Name, secret.ObjectMeta.Name)           // pragma: allowlist secret
	if created {
		require.NotZero(t, len(secret.ObjectMeta.Labels))
		labelVal, labelFound := secret.ObjectMeta.Labels[k8s.ManagedByLabelKey]
		require.True(t, labelFound)
		assert.Equal(t, labelVal, k8s.ManagedByFleetshardValue)
		require.NotZero(t, len(secret.ObjectMeta.Annotations))
		annotationVal, annotationFound := secret.ObjectMeta.Annotations[managedServicesAnnotation]
		require.True(t, annotationFound)
		assert.Equal(t, annotationVal, "true")
	}
	assert.Equal(t, expectedSecret.Data, secret.Data)
}

func TestEnsureSecretExists(t *testing.T) {
	fakeClient, _ := testutils.NewFakeClientWithTracker(t)
	secretModifyFunc := func(secret *v1.Secret) error {
		if secret.Data == nil {
			secret.Data = make(map[string][]byte)
		}
		payload, found := secret.Data[entryKey]
		if found {
			if bytes.Equal(payload, entryData) {
				return nil
			}
		}
		secret.Data[entryKey] = entryData
		return nil
	}
	testCases := []struct {
		secretName   string
		initialData  map[string][]byte
		expectedData map[string][]byte
	}{
		{
			secretName:  emptySecretName, // pragma: allowlist secret
			initialData: nil,
			expectedData: map[string][]byte{
				entryKey: entryData,
			},
		},
		{
			secretName: secretWithOtherKeyName, // pragma: allowlist secret
			initialData: map[string][]byte{
				otherKey: otherData,
			},
			expectedData: map[string][]byte{
				entryKey: entryData,
				otherKey: otherData,
			},
		},
		{
			secretName: secretWithKeyDataToChangeName, // pragma: allowlist secret
			initialData: map[string][]byte{
				entryKey: otherData,
			},
			expectedData: map[string][]byte{
				entryKey: entryData,
			},
		},
		{
			secretName: secretWithExpectedKeyDataOnlyName, // pragma: allowlist secret
			initialData: map[string][]byte{
				entryKey: entryData,
			},
			expectedData: map[string][]byte{
				entryKey: entryData,
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.secretName, func(t *testing.T) {
			ctx := context.TODO()
			fetchedSecret := &v1.Secret{}
			initialSecret := getSecret(tc.secretName, centralNamespace, tc.initialData)
			assert.NoError(t, fakeClient.Create(ctx, initialSecret))
			assert.NoError(t, ensureSecretExists(ctx, fakeClient, centralNamespace, tc.secretName, secretModifyFunc))
			assert.NoError(t, fakeClient.Get(ctx, client.ObjectKey{Namespace: centralNamespace, Name: tc.secretName}, fetchedSecret))
			compareSecret(t, getSecret(tc.secretName, centralNamespace, tc.expectedData), fetchedSecret, false)
		})
	}

	t.Run("missing secret", func(t *testing.T) {
		secretName := "missingSecret" // pragma: allowlist secret
		expectedData := map[string][]byte{
			entryKey: entryData,
		}
		ctx := context.TODO()
		fetchedSecret := &v1.Secret{}
		assert.NoError(t, ensureSecretExists(ctx, fakeClient, centralNamespace, secretName, secretModifyFunc))
		assert.NoError(t, fakeClient.Get(ctx, client.ObjectKey{Namespace: centralNamespace, Name: secretName}, fetchedSecret))
		compareSecret(t, getSecret(secretName, centralNamespace, expectedData), fetchedSecret, true)
	})
}

func TestGetAuditLogNotifierConfig(t *testing.T) {
	testCases := []struct {
		namespace      string
		auditLogging   config.AuditLogging
		auditLogTarget string
		auditLogPort   int
		expectedConfig *declarativeconfig.Notifier
	}{
		{
			namespace:      centralNamespace,
			auditLogging:   defaultAuditLogConfig,
			auditLogTarget: defaultAuditLogConfig.AuditLogTargetHost,
			auditLogPort:   defaultAuditLogConfig.AuditLogTargetPort,
			expectedConfig: &declarativeconfig.Notifier{
				Name: auditLogNotifierName,
				GenericConfig: &declarativeconfig.GenericConfig{
					Endpoint: fmt.Sprintf(
						"https://%s:%d",
						defaultAuditLogConfig.AuditLogTargetHost,
						defaultAuditLogConfig.AuditLogTargetPort,
					),
					SkipTLSVerify:       true,
					AuditLoggingEnabled: true,
					ExtraFields: []declarativeconfig.KeyValuePair{
						{
							Key:   auditLogTenantIDKey,
							Value: centralNamespace,
						},
					},
				},
			},
		},
		{
			namespace:      "rhacs",
			auditLogging:   vectorAuditLogConfig,
			auditLogTarget: vectorAuditLogConfig.AuditLogTargetHost,
			auditLogPort:   vectorAuditLogConfig.AuditLogTargetPort,
			expectedConfig: &declarativeconfig.Notifier{
				Name: auditLogNotifierName,
				GenericConfig: &declarativeconfig.GenericConfig{
					Endpoint: fmt.Sprintf(
						"https://%s:%d",
						vectorAuditLogConfig.AuditLogTargetHost,
						vectorAuditLogConfig.AuditLogTargetPort,
					),
					SkipTLSVerify:       true,
					AuditLoggingEnabled: true,
					ExtraFields: []declarativeconfig.KeyValuePair{
						{
							Key:   auditLogTenantIDKey,
							Value: "rhacs",
						},
					},
				},
			},
		},
	}
	for _, testCase := range testCases {
		notifierConfig := getAuditLogNotifierConfig(testCase.auditLogging, testCase.namespace)
		assert.Equal(t, testCase.expectedConfig, notifierConfig)
	}
}

func populateDeclarativeConfigSecrets(
	t *testing.T,
	namespace string,
	payload map[string][]declarativeconfig.Configuration,
) *v1.Secret {
	if len(payload) == 0 {
		return getSecret(sensibleDeclarativeConfigSecretName, namespace, nil)
	}
	secretData := make(map[string][]byte, len(payload))
	for dataKey, configList := range payload {
		switch len(configList) {
		case 0:
			secretData[dataKey] = nil
		case 1:
			encodedBytes, encodingErr := yaml.Marshal(configList[0])
			assert.NoError(t, encodingErr)
			secretData[dataKey] = encodedBytes
		default:
			encodedBytes, encodingErr := yaml.Marshal(configList)
			assert.NoError(t, encodingErr)
			secretData[dataKey] = encodedBytes
		}
	}
	return getSecret(sensibleDeclarativeConfigSecretName, centralNamespace, secretData)
}

func TestReconcileDeclarativeConfigurationData(t *testing.T) {
	defaultNotifierConfig := getAuditLogNotifierConfig(
		defaultAuditLogConfig,
		centralNamespace,
	)
	faultyVectorNotifierConfig := getAuditLogNotifierConfig(
		vectorAuditLogConfig,
		"rhacs",
	)
	correctVectorNotifierConfig := getAuditLogNotifierConfig(
		vectorAuditLogConfig,
		centralNamespace,
	)

	authProviderConfig := getAuthProviderConfig(simpleManagedCentral)

	const otherItemKey = "other.item.key"

	testCases := []struct {
		name                       string
		auditLogConfig             config.AuditLogging
		preExistingSecret          bool
		initialDeclarativeConfigs  map[string][]declarativeconfig.Configuration
		expectedDeclarativeConfigs map[string][]declarativeconfig.Configuration
		wantsAuthProvider          bool
	}{
		{
			name:              "Missing default secret gets created",
			auditLogConfig:    defaultAuditLogConfig,
			preExistingSecret: false, // pragma: allowlist secret
			expectedDeclarativeConfigs: map[string][]declarativeconfig.Configuration{
				auditLogNotifierKey:              {defaultNotifierConfig},
				authProviderDeclarativeConfigKey: {authProviderConfig},
			},
			wantsAuthProvider: true,
		},
		{
			name:              "Missing vector secret gets created",
			auditLogConfig:    vectorAuditLogConfig,
			preExistingSecret: false, // pragma: allowlist secret
			expectedDeclarativeConfigs: map[string][]declarativeconfig.Configuration{
				auditLogNotifierKey: {correctVectorNotifierConfig},
			},
		},
		{
			name:              "Empty secret when audit logging and auth provider creation disabled",
			auditLogConfig:    disabledAuditLogConfig,
			preExistingSecret: false, // pragma: allowlist secret
		},
		{
			name:              "No secret modification when audit logging and auth provider creation disabled",
			auditLogConfig:    disabledAuditLogConfig,
			preExistingSecret: true, // pragma: allowlist secret
			initialDeclarativeConfigs: map[string][]declarativeconfig.Configuration{
				auditLogNotifierKey:              {defaultNotifierConfig},
				authProviderDeclarativeConfigKey: {authProviderConfig},
				otherItemKey:                     {faultyVectorNotifierConfig},
			},
			expectedDeclarativeConfigs: map[string][]declarativeconfig.Configuration{
				auditLogNotifierKey:              {defaultNotifierConfig},
				authProviderDeclarativeConfigKey: {authProviderConfig},
				otherItemKey:                     {faultyVectorNotifierConfig},
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			ctx := context.TODO()
			reconcilerOptions := CentralReconcilerOptions{
				AuditLogging:      testCase.auditLogConfig,
				WantsAuthProvider: testCase.wantsAuthProvider,
			}
			fakeClient, _, r := getClientTrackerAndReconciler(
				t,
				nil,
				reconcilerOptions,
			)
			if testCase.preExistingSecret {
				secret := populateDeclarativeConfigSecrets(t, centralNamespace, testCase.initialDeclarativeConfigs)
				require.NoError(t, fakeClient.Create(ctx, secret))
			}
			assert.NoError(t, r.reconcileDeclarativeConfigurationData(ctx, simpleManagedCentral))
			fetchedSecret := &v1.Secret{}
			secretKey := client.ObjectKey{ // pragma: allowlist secret
				Name:      sensibleDeclarativeConfigSecretName,
				Namespace: centralNamespace,
			}
			postFetchErr := fakeClient.Get(ctx, secretKey, fetchedSecret)
			assert.NoError(t, postFetchErr)
			expectedSecret := populateDeclarativeConfigSecrets(t, centralNamespace, testCase.expectedDeclarativeConfigs)
			compareSecret(t, expectedSecret, fetchedSecret, !testCase.preExistingSecret)
		})
	}
}

func TestRestoreCentralSecrets(t *testing.T) {
	testCases := []struct {
		name                     string
		buildCentral             func() private.ManagedCentral
		mockObjects              []client.Object
		buildFMClient            func() *fleetmanager.Client
		expectedErrorMsgContains string
		expectedObjects          []client.Object
	}{
		{
			name: "no error for SecretsStored not set",
			buildCentral: func() private.ManagedCentral {
				return simpleManagedCentral
			},
		},
		{
			name: "no error for existing secrets in SecretsStored",
			buildCentral: func() private.ManagedCentral {
				newCentral := simpleManagedCentral
				newCentral.Metadata.SecretsStored = []string{"central-tls", "central-db-password"}
				return newCentral
			},
			mockObjects: []client.Object{
				centralTLSSecretObject(),
				centralDBPasswordSecretObject(),
			},
		},
		{
			name: "return errors from fleetmanager",
			buildCentral: func() private.ManagedCentral {
				newCentral := simpleManagedCentral
				newCentral.Metadata.SecretsStored = []string{"central-tls", "central-db-password"}
				return newCentral
			},
			mockObjects: []client.Object{
				centralTLSSecretObject(),
			},
			buildFMClient: func() *fleetmanager.Client {
				mockClient := fmMocks.NewClientMock()
				mockClient.PrivateAPIMock.GetCentralFunc = func(ctx context.Context, centralID string) (private.ManagedCentral, *http.Response, error) {
					return private.ManagedCentral{}, nil, errors.New("test error")
				}
				return mockClient.Client()
			},
			expectedErrorMsgContains: "loading secrets for central cb45idheg5ip6dq1jo4g: test error",
		},
		{
			// force encrypt error by using non base64 value for central-db-password
			name: "return errors from decryptSecrets",
			buildCentral: func() private.ManagedCentral {
				newCentral := simpleManagedCentral
				newCentral.Metadata.SecretsStored = []string{"central-tls", "central-db-password"}
				return newCentral
			},
			mockObjects: []client.Object{
				centralTLSSecretObject(),
			},
			buildFMClient: func() *fleetmanager.Client {
				mockClient := fmMocks.NewClientMock()
				mockClient.PrivateAPIMock.GetCentralFunc = func(ctx context.Context, centralID string) (private.ManagedCentral, *http.Response, error) {
					returnCentral := simpleManagedCentral
					returnCentral.Metadata.Secrets = map[string]string{"central-db-password": "testpw"}
					return returnCentral, nil, nil
				}
				return mockClient.Client()
			},
			expectedErrorMsgContains: "decrypting secrets for central",
		},
		{
			name: "expect secrets to exist after secret restore",
			buildCentral: func() private.ManagedCentral {
				newCentral := simpleManagedCentral
				newCentral.Metadata.SecretsStored = []string{"central-tls", "central-db-password"}
				return newCentral
			},
			buildFMClient: func() *fleetmanager.Client {
				mockClient := fmMocks.NewClientMock()
				mockClient.PrivateAPIMock.GetCentralFunc = func(ctx context.Context, centralID string) (private.ManagedCentral, *http.Response, error) {
					returnCentral := simpleManagedCentral
					centralTLS := `{"metadata":{"name":"central-tls","namespace":"rhacs-cb45idheg5ip6dq1jo4g","creationTimestamp":null}}`
					centralDBPW := `{"metadata":{"name":"central-db-password","namespace":"rhacs-cb45idheg5ip6dq1jo4g","creationTimestamp":null}}`

					encode := base64.StdEncoding.EncodeToString
					// we need to encode twice, once for b64 test cipher used
					// once for the b64 encoding done to transfer secret data via API
					returnCentral.Metadata.Secrets = map[string]string{
						"central-tls":         encode([]byte(encode([]byte(centralTLS)))),
						"central-db-password": encode([]byte(encode([]byte(centralDBPW)))),
					}
					return returnCentral, nil, nil
				}
				return mockClient.Client()
			},
			expectedObjects: []client.Object{
				centralTLSSecretObject(),
				centralDBPasswordSecretObject(),
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			fakeClient, _, r := getClientTrackerAndReconciler(t, nil, defaultReconcilerOptions, tc.mockObjects...)
			managedCentral := tc.buildCentral()

			if tc.buildFMClient != nil {
				r.fleetmanagerClient = tc.buildFMClient()
			}

			err := r.restoreCentralSecrets(context.Background(), managedCentral)

			if err != nil && tc.expectedErrorMsgContains != "" {
				require.Contains(t, err.Error(), tc.expectedErrorMsgContains)
			} else {
				require.NoError(t, err)
			}

			for _, obj := range tc.expectedObjects {
				s := v1.Secret{}
				err := fakeClient.Get(context.Background(), client.ObjectKey{Namespace: obj.GetNamespace(), Name: obj.GetName()}, &s)
				require.NoErrorf(t, err, "finding expected object %s/%s", obj.GetNamespace(), obj.GetName())
			}

		})
	}
}

func TestReconciler_reconcileNamespace(t *testing.T) {
	tests := []struct {
		name              string
		existingNamespace *v1.Namespace
		wantErr           bool
		wantNamespace     *v1.Namespace
		expectUpdate      bool
		expectCreate      bool
	}{
		{
			name:         "namespace should be created if it doesn't exist",
			expectCreate: true,
			wantNamespace: &v1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: simpleManagedCentral.Metadata.Namespace,
					Labels: map[string]string{
						"app.kubernetes.io/instance":     "test-central",
						"app.kubernetes.io/managed-by":   "rhacs-fleetshard",
						"rhacs.redhat.com/instance-type": "standard",
						"rhacs.redhat.com/org-id":        "12345",
						"rhacs.redhat.com/tenant":        "cb45idheg5ip6dq1jo4g",
						"argocd.argoproj.io/managed-by":  "openshift-gitops",
					},
					Annotations: map[string]string{
						"rhacs.redhat.com/org-name": "org-name",
						ovnACLLoggingAnnotationKey:  ovnACLLoggingAnnotationDefault,
					},
				},
			},
		},
		{
			name:         "namespace with wrong labels or annotations should be updated",
			expectUpdate: true,
			existingNamespace: &v1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: simpleManagedCentral.Metadata.Namespace,
					Labels: map[string]string{
						"app.kubernetes.io/instance":     "wrong",
						"app.kubernetes.io/managed-by":   "wrong",
						"rhacs.redhat.com/instance-type": "wrong",
						"rhacs.redhat.com/org-id":        "wrong",
						"rhacs.redhat.com/tenant":        "wrong",
						"argocd.argoproj.io/managed-by":  "wrong",
					},
					Annotations: map[string]string{
						"rhacs.redhat.com/org-name": "wrong",
						ovnACLLoggingAnnotationKey:  "{\"allow\": \"wrong\"}",
					},
				},
			},
			wantNamespace: &v1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: simpleManagedCentral.Metadata.Namespace,
					Labels: map[string]string{
						"app.kubernetes.io/instance":     "test-central",
						"app.kubernetes.io/managed-by":   "rhacs-fleetshard",
						"rhacs.redhat.com/instance-type": "standard",
						"rhacs.redhat.com/org-id":        "12345",
						"rhacs.redhat.com/tenant":        "cb45idheg5ip6dq1jo4g",
						"argocd.argoproj.io/managed-by":  "openshift-gitops",
					},
					Annotations: map[string]string{
						"rhacs.redhat.com/org-name": "org-name",
						ovnACLLoggingAnnotationKey:  ovnACLLoggingAnnotationDefault,
					},
				},
			},
		},
		{
			name: "extra labels/annotations should remain untouched",
			existingNamespace: &v1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: simpleManagedCentral.Metadata.Namespace,
					Labels: map[string]string{
						"app.kubernetes.io/instance":     "test-central",
						"app.kubernetes.io/managed-by":   "rhacs-fleetshard",
						"rhacs.redhat.com/instance-type": "standard",
						"rhacs.redhat.com/org-id":        "12345",
						"rhacs.redhat.com/tenant":        "cb45idheg5ip6dq1jo4g",
						"argocd.argoproj.io/managed-by":  "openshift-gitops",
						"extra":                          "extra",
					},
					Annotations: map[string]string{
						"rhacs.redhat.com/org-name": "org-name",
						ovnACLLoggingAnnotationKey:  ovnACLLoggingAnnotationDefault,
						"extra":                     "extra",
					},
				},
			},
			wantNamespace: &v1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: simpleManagedCentral.Metadata.Namespace,
					Labels: map[string]string{
						"app.kubernetes.io/instance":     "test-central",
						"app.kubernetes.io/managed-by":   "rhacs-fleetshard",
						"rhacs.redhat.com/instance-type": "standard",
						"rhacs.redhat.com/org-id":        "12345",
						"rhacs.redhat.com/tenant":        "cb45idheg5ip6dq1jo4g",
						"argocd.argoproj.io/managed-by":  "openshift-gitops",
						"extra":                          "extra",
					},
					Annotations: map[string]string{
						"rhacs.redhat.com/org-name": "org-name",
						ovnACLLoggingAnnotationKey:  ovnACLLoggingAnnotationDefault,
						"extra":                     "extra",
					},
				},
			},
		},
		{
			name: "namespace should not be updated if it's already correct",
			existingNamespace: &v1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: simpleManagedCentral.Metadata.Namespace,
					Labels: map[string]string{
						"app.kubernetes.io/instance":     "test-central",
						"app.kubernetes.io/managed-by":   "rhacs-fleetshard",
						"rhacs.redhat.com/instance-type": "standard",
						"rhacs.redhat.com/org-id":        "12345",
						"rhacs.redhat.com/tenant":        "cb45idheg5ip6dq1jo4g",
						"argocd.argoproj.io/managed-by":  "openshift-gitops",
					},
					Annotations: map[string]string{
						"rhacs.redhat.com/org-name": "org-name",
						ovnACLLoggingAnnotationKey:  ovnACLLoggingAnnotationDefault,
					},
				},
			},
			wantNamespace: &v1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: simpleManagedCentral.Metadata.Namespace,
					Labels: map[string]string{
						"app.kubernetes.io/instance":     "test-central",
						"app.kubernetes.io/managed-by":   "rhacs-fleetshard",
						"rhacs.redhat.com/instance-type": "standard",
						"rhacs.redhat.com/org-id":        "12345",
						"rhacs.redhat.com/tenant":        "cb45idheg5ip6dq1jo4g",
						"argocd.argoproj.io/managed-by":  "openshift-gitops",
					},
					Annotations: map[string]string{
						"rhacs.redhat.com/org-name": "org-name",
						ovnACLLoggingAnnotationKey:  ovnACLLoggingAnnotationDefault,
					},
				},
			},
		},
	}

	managedCentral := simpleManagedCentral

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fakeClient, _, r := getClientTrackerAndReconciler(t, nil, defaultReconcilerOptions)
			if tt.existingNamespace != nil {
				require.NoError(t, fakeClient.Create(context.Background(), tt.existingNamespace))
			}
			updateCount := 0
			createCount := 0
			r.namespaceReconciler.client = interceptor.NewClient(fakeClient, interceptor.Funcs{
				Update: func(ctx context.Context, client client.WithWatch, obj client.Object, opts ...client.UpdateOption) error {
					updateCount++
					return client.Update(ctx, obj, opts...)
				},
				Create: func(ctx context.Context, client client.WithWatch, obj client.Object, opts ...client.CreateOption) error {
					createCount++
					return client.Create(ctx, obj, opts...)
				},
			})
			err := r.namespaceReconciler.reconcile(context.Background(), r.getDesiredNamespace(managedCentral))
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				var gotNamespace v1.Namespace
				err := fakeClient.Get(context.Background(), client.ObjectKey{Name: simpleManagedCentral.Metadata.Namespace}, &gotNamespace)
				require.NoError(t, err)
				assert.Equal(t, tt.wantNamespace.Name, gotNamespace.Name)
				assert.Equal(t, tt.wantNamespace.Labels, gotNamespace.Labels)
				assert.Equal(t, tt.wantNamespace.Annotations, gotNamespace.Annotations)
				if tt.expectUpdate {
					assert.Equal(t, 1, updateCount, "update should be called")
				} else {
					assert.Equal(t, 0, updateCount, "update should not be called")
				}

				if tt.expectCreate {
					assert.Equal(t, 1, createCount, "create should be called")
				} else {
					assert.Equal(t, 0, createCount, "create should not be called")
				}
			}
		})
	}
}

func TestReconciler_needsReconcile(t *testing.T) {
	tests := []struct {
		name string
		// central (hash) has changed
		changed bool
		// the central to reconcile
		central private.ManagedCentral
		// mocking the areSecretsStoredFunc
		secretsStoredFunc areSecretsStoredFunc
		// how long since the last hash was stored
		timePassed time.Duration
		// desired output
		want bool
	}{
		{
			name:              "no change",
			changed:           false,
			central:           private.ManagedCentral{},
			secretsStoredFunc: func([]string) bool { return true },
			timePassed:        0,
			want:              false,
		}, {
			name:              "central changed",
			changed:           true,
			central:           private.ManagedCentral{},
			secretsStoredFunc: func([]string) bool { return true },
			timePassed:        0,
			want:              true,
		}, {
			name:              "secrets not stored",
			changed:           false,
			central:           private.ManagedCentral{},
			secretsStoredFunc: func([]string) bool { return false },
			timePassed:        0,
			want:              true,
		}, {
			name:              "time passed",
			changed:           false,
			central:           private.ManagedCentral{},
			secretsStoredFunc: func([]string) bool { return true },
			timePassed:        1 * time.Hour,
			want:              true,
		}, {
			name:    "force reconcile",
			changed: false,
			central: private.ManagedCentral{
				Spec: private.ManagedCentralAllOfSpec{
					TenantResourcesValues: map[string]interface{}{
						"forceReconcile": true,
					},
				},
			},
			secretsStoredFunc: func([]string) bool { return true },
			timePassed:        0,
			want:              true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, r := getClientTrackerAndReconciler(t, nil, defaultReconcilerOptions)
			r.areSecretsStoredFunc = tt.secretsStoredFunc //pragma: allowlist secret
			r.clock = fakeClock{
				NowTime: time.Now(),
			}
			r.lastCentralHashTime = r.clock.Now().Add(-tt.timePassed)
			got := r.needsReconcile(tt.changed, tt.central, []string{})
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestEncryptionShaSum(t *testing.T) {
	reconciler := &CentralReconciler{
		secretCipher: cipher.LocalBase64Cipher{}, // pragma: allowlist secret
	}

	testSecrets := map[string]*v1.Secret{
		"testsecret1": {
			ObjectMeta: metav1.ObjectMeta{
				Name:      "testsecret1",
				Namespace: centralNamespace,
			},
			Data: map[string][]byte{
				"test1": []byte("test1-secretdata1"),
				"test2": []byte("test1-secretdata2"),
			},
		},
		"testsecret2": {
			ObjectMeta: metav1.ObjectMeta{
				Name:      "testsecret2",
				Namespace: centralNamespace,
			},
			Data: map[string][]byte{
				"test1": []byte("test2-secretdata1"),
				"test2": []byte("test2-secretdata2"),
			},
		},
	}

	enc1, err := reconciler.encryptSecrets(testSecrets)
	require.NoError(t, err)
	enc2, err := reconciler.encryptSecrets(testSecrets)
	require.NoError(t, err)

	testSecrets["testsecret1"].Data["test3"] = []byte("test3")
	encChanged, err := reconciler.encryptSecrets(testSecrets)
	require.NoError(t, err)

	require.NoError(t, err)
	require.Equal(t, enc1.sha256Sum, enc2.sha256Sum, "hash of equal secrets was not equal")
	require.NotEqual(t, enc1.sha256Sum, encChanged.sha256Sum, "hash of unequal secrets was equal")
}

func TestEncyrptionSHASumSameObject(t *testing.T) {
	// This test is important, since it helped catch a bug discovered during e2e testing
	// of this feature that would cause the calculated hash to be not equal for the same secrets
	// because the function was looping over keys of Go maps, which is not guaranteed to loop in the
	// same order on every invokation
	reconciler := &CentralReconciler{
		secretCipher: cipher.LocalBase64Cipher{}, // pragma: allowlist secret
	}

	testSecrets := map[string]*v1.Secret{
		"testsecret1": {
			ObjectMeta: metav1.ObjectMeta{
				Name:      "testsecret1",
				Namespace: centralNamespace,
			},
			Data: map[string][]byte{
				"test1": []byte("test1-secretdata1"),
				"test2": []byte("test1-secretdata2"),
			},
		},
		"testsecret2": {
			ObjectMeta: metav1.ObjectMeta{
				Name:      "testsecret2",
				Namespace: centralNamespace,
			},
			Data: map[string][]byte{
				"test1": []byte("test2-secretdata1"),
				"test2": []byte("test2-secretdata2"),
			},
		},
	}

	amount := 1000
	sums := make([]string, 1000)
	for i := range amount {
		enc, err := reconciler.encryptSecrets(testSecrets)
		require.NoError(t, err)
		sums[i] = enc.sha256Sum
	}

	for i := range amount - 1 {
		require.Equal(t, sums[i], sums[i+1], "hash of the same object should always be equal but was not")
	}
}
