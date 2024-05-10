package reconciler

import (
	"bytes"
	"context"
	"embed"
	"encoding/base64"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/credentials/stscreds"
	openshiftRouteV1 "github.com/openshift/api/route/v1"
	"github.com/pkg/errors"
	"github.com/stackrox/acs-fleet-manager/fleetshard/config"
	"github.com/stackrox/acs-fleet-manager/fleetshard/pkg/central/charts"
	"github.com/stackrox/acs-fleet-manager/fleetshard/pkg/central/cloudprovider"
	"github.com/stackrox/acs-fleet-manager/fleetshard/pkg/central/cloudprovider/awsclient"
	"github.com/stackrox/acs-fleet-manager/fleetshard/pkg/central/postgres"
	"github.com/stackrox/acs-fleet-manager/fleetshard/pkg/cipher"
	"github.com/stackrox/acs-fleet-manager/fleetshard/pkg/k8s"
	"github.com/stackrox/acs-fleet-manager/fleetshard/pkg/testutils"
	"github.com/stackrox/acs-fleet-manager/fleetshard/pkg/util"
	centralConstants "github.com/stackrox/acs-fleet-manager/internal/dinosaur/constants"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/api/private"
	"github.com/stackrox/acs-fleet-manager/pkg/client/fleetmanager"
	fmMocks "github.com/stackrox/acs-fleet-manager/pkg/client/fleetmanager/mocks"
	centralNotifierUtils "github.com/stackrox/rox/central/notifiers/utils"
	"github.com/stackrox/rox/operator/apis/platform/v1alpha1"
	"github.com/stackrox/rox/pkg/declarativeconfig"
	"github.com/stackrox/rox/pkg/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v2"
	"helm.sh/helm/v3/pkg/chart/loader"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/utils/pointer"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
	yaml2 "sigs.k8s.io/yaml"
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

var (
	defaultCentralConfig = private.ManagedCentral{}

	defaultReconcilerOptions = CentralReconcilerOptions{}

	useRoutesReconcilerOptions           = CentralReconcilerOptions{UseRoutes: true}
	secureTenantNetworkReconcilerOptions = CentralReconcilerOptions{SecureTenantNetwork: true}

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

	defaultRouteConfig = config.RouteConfig{
		ConcurrentTCP: 32,
		RateHTTP:      128,
		RateTCP:       16,
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
		UiEndpoint: private.ManagedCentralAllOfSpecUiEndpoint{
			Host: fmt.Sprintf("acs-%s.acs.rhcloud.test", centralID),
		},
		DataEndpoint: private.ManagedCentralAllOfSpecDataEndpoint{
			Host: fmt.Sprintf("acs-data-%s.acs.rhcloud.test", centralID),
		},
		InstanceType: "standard",
		CentralCRYAML: `
metadata:
  name: ` + centralName + `
  namespace: ` + centralNamespace + `
`,
	},
}

//go:embed testdata
var testdata embed.FS

func createBase64Cipher(t *testing.T) cipher.Cipher {
	b64Cipher, err := cipher.NewLocalBase64Cipher()
	require.NoError(t, err, "creating base64 cipher for test")
	return b64Cipher
}

func getClientTrackerAndReconciler(
	t *testing.T,
	centralConfig private.ManagedCentral,
	managedDBClient cloudprovider.DBClient,
	reconcilerOptions CentralReconcilerOptions,
	k8sObjects ...client.Object,
) (client.WithWatch, *testutils.ReconcileTracker, *CentralReconciler) {
	fakeClient, tracker := testutils.NewFakeClientWithTracker(t, k8sObjects...)
	reconciler := NewCentralReconciler(
		fakeClient,
		fmMocks.NewClientMock().Client(),
		centralConfig,
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
	}
}

func centralDBPasswordSecretObject() *v1.Secret {
	return &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "central-db-password",
			Namespace: centralNamespace,
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
	reconcilerOptions := CentralReconcilerOptions{
		ClusterName:      clusterName,
		Environment:      environment,
		ManagedDBEnabled: false,
		UseRoutes:        true,
	}
	fakeClient, _, r := getClientTrackerAndReconciler(
		t,
		defaultCentralConfig,
		nil,
		reconcilerOptions,
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
	assert.Equal(t, "1", central.GetAnnotations()[util.RevisionAnnotationKey])
	assert.Equal(t, true, *central.Spec.Central.Exposure.Route.Enabled)

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

	reconcilerOptions := CentralReconcilerOptions{
		UseRoutes:        true,
		ManagedDBEnabled: true,
	}
	fakeClient, _, r := getClientTrackerAndReconciler(
		t,
		defaultCentralConfig,
		managedDBProvisioningClient,
		reconcilerOptions,
	)

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
		defaultCentralConfig,
		managedDBProvisioningClient,
		reconcilerOptions,
	)

	_, err = r.Reconcile(context.TODO(), simpleManagedCentral)
	var awsErr awserr.Error
	require.ErrorAs(t, err, &awsErr)
	assert.Equal(t, stscreds.ErrCodeWebIdentity, awsErr.Code())
}

func TestReconcileUpdateSucceeds(t *testing.T) {
	fakeClient, _, r := getClientTrackerAndReconciler(
		t,
		defaultCentralConfig,
		nil,
		defaultReconcilerOptions,
		&v1alpha1.Central{
			ObjectMeta: metav1.ObjectMeta{
				Name:        centralName,
				Namespace:   centralNamespace,
				Annotations: map[string]string{util.RevisionAnnotationKey: "3"},
			},
		},
		centralDeploymentObject(),
	)

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
		status:                 pointer.Int32(0),
		client:                 fakeClient,
		central:                private.ManagedCentral{},
		resourcesChart:         resourcesChart,
		encryptionKeyGenerator: cipher.AES256KeyGenerator{},
		secretBackup:           k8s.NewSecretBackup(fakeClient, false),
	}
	r.areSecretsStoredFunc = r.areSecretsStored //pragma: allowlist secret
	r.needsReconcileFunc = r.needsReconcile
	r.restoreCentralSecretsFunc = r.restoreCentralSecrets //pragma: allowlist secret

	_, err := r.Reconcile(context.TODO(), simpleManagedCentral)
	require.Error(t, err)

	assert.Equal(t, [16]byte{}, r.lastCentralHash)
}

func TestReconcileLastHashSetOnSuccess(t *testing.T) {
	fakeClient, _, r := getClientTrackerAndReconciler(
		t,
		defaultCentralConfig,
		nil,
		defaultReconcilerOptions,
		&v1alpha1.Central{
			ObjectMeta: metav1.ObjectMeta{
				Name:        centralName,
				Namespace:   centralNamespace,
				Annotations: map[string]string{util.RevisionAnnotationKey: "3"},
			},
		},
		centralDeploymentObject(),
		centralTLSSecretObject(),
		centralDBPasswordSecretObject(),
		centralEncryptionKeySecretObject(),
	)

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

	central := &v1alpha1.Central{}
	err = fakeClient.Get(context.TODO(), client.ObjectKey{Name: centralName, Namespace: centralNamespace}, central)
	require.NoError(t, err)
	assert.Equal(t, "4", central.Annotations[util.RevisionAnnotationKey])
}

func TestReconcileLastHashSecretsOrderIndependent(t *testing.T) {
	_, _, r := getClientTrackerAndReconciler(
		t,
		defaultCentralConfig,
		nil,
		defaultReconcilerOptions,
		&v1alpha1.Central{
			ObjectMeta: metav1.ObjectMeta{
				Name:        centralName,
				Namespace:   centralNamespace,
				Annotations: map[string]string{util.RevisionAnnotationKey: "3"},
			},
		},
		centralDeploymentObject(),
		centralTLSSecretObject(),
		centralDBPasswordSecretObject(),
	)

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
	_, _, r := getClientTrackerAndReconciler(
		t,
		defaultCentralConfig,
		nil,
		defaultReconcilerOptions,
		&v1alpha1.Central{
			ObjectMeta: metav1.ObjectMeta{
				Name:        centralName,
				Namespace:   centralNamespace,
				Annotations: map[string]string{util.RevisionAnnotationKey: "3"},
			},
		},
		centralDeploymentObject(),
	)

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
	_, _, r := getClientTrackerAndReconciler(
		t,
		defaultCentralConfig,
		nil,
		defaultReconcilerOptions,
		&v1alpha1.Central{
			ObjectMeta: metav1.ObjectMeta{
				Name:        centralName,
				Namespace:   centralNamespace,
				Annotations: map[string]string{util.RevisionAnnotationKey: "3"},
			},
		},
		centralDeploymentObject(),
		centralTLSSecretObject(),
		centralDBPasswordSecretObject(),
	)

	managedCentral := simpleManagedCentral
	managedCentral.RequestStatus = centralConstants.CentralRequestStatusReady.String()
	managedCentral.Spec.CentralCRYAML = `
metadata:
  name: ` + centralName + `
  namespace: ` + centralNamespace + `
  labels:
    rhacs.redhat.com/force-reconcile: "true"
`

	expectedHash, err := util.MD5SumFromJSONStruct(&managedCentral)
	require.NoError(t, err)

	_, err = r.Reconcile(context.TODO(), managedCentral)
	require.NoError(t, err)
	assert.Equal(t, expectedHash, r.lastCentralHash)

	_, err = r.Reconcile(context.TODO(), managedCentral)
	require.NoError(t, err)
}

func TestReconcileDelete(t *testing.T) {
	fakeClient, _, r := getClientTrackerAndReconciler(
		t,
		defaultCentralConfig,
		nil,
		useRoutesReconcilerOptions,
	)

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
	fakeClient, _, r := getClientTrackerAndReconciler(
		t,
		defaultCentralConfig,
		nil,
		useRoutesReconcilerOptions,
	)

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

	reconcilerOptions := CentralReconcilerOptions{
		UseRoutes:        true,
		ManagedDBEnabled: true,
	}
	fakeClient, _, r := getClientTrackerAndReconciler(
		t,
		defaultCentralConfig,
		managedDBProvisioningClient,
		reconcilerOptions,
	)

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

	reconcilerOptions := CentralReconcilerOptions{
		UseRoutes:        true,
		ManagedDBEnabled: true,
	}
	_, _, r := getClientTrackerAndReconciler(
		t,
		defaultCentralConfig,
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
					UiEndpoint: private.ManagedCentralAllOfSpecUiEndpoint{
						Host: "central.cluster.local",
					},
				},
			},
			want: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, _, reconciler := getClientTrackerAndReconciler(
				t,
				test.currentCentral,
				nil,
				defaultReconcilerOptions,
				centralDeploymentObject(),
			)

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
		t,
		defaultCentralConfig,
		nil,
		useRoutesReconcilerOptions,
	)

	_, err := r.Reconcile(context.TODO(), simpleManagedCentral)
	require.NoError(t, err)

	namespace := &v1.Namespace{}
	err = fakeClient.Get(context.TODO(), client.ObjectKey{Name: centralNamespace}, namespace)
	require.NoError(t, err)
	assert.Equal(t, simpleManagedCentral.Id, namespace.GetLabels()[tenantIDLabelKey])
	assert.Equal(t, simpleManagedCentral.Spec.Auth.OwnerOrgId, namespace.GetLabels()[orgIDLabelKey])
}

func TestNamespaceAnnotationsAreSet(t *testing.T) {
	fakeClient, _, r := getClientTrackerAndReconciler(
		t,
		defaultCentralConfig,
		nil,
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
		t,
		defaultCentralConfig,
		nil,
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

func TestChartResourcesAreAddedAndRemoved(t *testing.T) {
	chartFiles, err := charts.TraverseChart(testdata, "testdata/tenant-resources")
	require.NoError(t, err)
	chart, err := loader.LoadFiles(chartFiles)
	require.NoError(t, err)

	fakeClient, _, r := getClientTrackerAndReconciler(
		t,
		defaultCentralConfig,
		nil,
		defaultReconcilerOptions,
	)
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

func TestCentralEncryptionKeyIsGenerated(t *testing.T) {
	fakeClient, _, r := getClientTrackerAndReconciler(
		t,
		defaultCentralConfig,
		nil,
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

func TestChartResourcesAreAddedAndUpdated(t *testing.T) {
	chartFiles, err := charts.TraverseChart(testdata, "testdata/tenant-resources")
	require.NoError(t, err)
	chart, err := loader.LoadFiles(chartFiles)
	require.NoError(t, err)

	fakeClient, _, r := getClientTrackerAndReconciler(
		t,
		defaultCentralConfig,
		nil,
		defaultReconcilerOptions,
	)
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
	fakeClient, _, r := getClientTrackerAndReconciler(
		t,
		defaultCentralConfig,
		nil,
		defaultReconcilerOptions,
	)

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

func TestEgressProxyIsNotDeployedWhenSecureTenantNetwork(t *testing.T) {
	fakeClient, _, r := getClientTrackerAndReconciler(
		t,
		defaultCentralConfig,
		nil,
		secureTenantNetworkReconcilerOptions,
	)

	_, err := r.Reconcile(context.TODO(), simpleManagedCentral)
	require.NoError(t, err)

	unexpectedObjs := []client.Object{
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

	for _, unexpectedObj := range unexpectedObjs {
		actualObj := unexpectedObj.DeepCopyObject().(client.Object)
		assert.Error(t, fakeClient.Get(context.TODO(), client.ObjectKeyFromObject(unexpectedObj), actualObj))
	}
}

func TestTenantNetworkIsSecured(t *testing.T) {
	fakeClient, _, r := getClientTrackerAndReconciler(
		t,
		defaultCentralConfig,
		nil,
		secureTenantNetworkReconcilerOptions,
	)

	_, err := r.Reconcile(context.TODO(), simpleManagedCentral)
	require.NoError(t, err)

	expectedObjs := []client.Object{
		&networkingv1.NetworkPolicy{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: simpleManagedCentral.Metadata.Namespace,
				Name:      "default-deny-all-except-dns",
			},
		},
		&networkingv1.NetworkPolicy{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: simpleManagedCentral.Metadata.Namespace,
				Name:      "tenant-central",
			},
		},
		&networkingv1.NetworkPolicy{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: simpleManagedCentral.Metadata.Namespace,
				Name:      "tenant-scanner",
			},
		},
		&networkingv1.NetworkPolicy{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: simpleManagedCentral.Metadata.Namespace,
				Name:      "tenant-scanner-db",
			},
		},
		&networkingv1.NetworkPolicy{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: simpleManagedCentral.Metadata.Namespace,
				Name:      "tenant-scanner-v4-db",
			},
		},
		&networkingv1.NetworkPolicy{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: simpleManagedCentral.Metadata.Namespace,
				Name:      "tenant-scanner-v4-indexer",
			},
		},
		&networkingv1.NetworkPolicy{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: simpleManagedCentral.Metadata.Namespace,
				Name:      "tenant-scanner-v4-matcher",
			},
		},
	}

	for _, expectedObj := range expectedObjs {
		actualObj := expectedObj.DeepCopyObject().(client.Object)
		if !assert.NoError(t, fakeClient.Get(context.TODO(), client.ObjectKeyFromObject(expectedObj), actualObj)) {
			continue
		}
		assert.Equal(t, k8s.ManagedByFleetshardValue, actualObj.GetLabels()[k8s.ManagedByLabelKey])
	}
}

func TestEgressProxyCustomImage(t *testing.T) {
	reconcilerOptions := CentralReconcilerOptions{
		EgressProxyImage: "registry.redhat.io/openshift4/ose-egress-http-proxy:version-for-test",
	}
	fakeClient, _, r := getClientTrackerAndReconciler(
		t,
		defaultCentralConfig,
		nil,
		reconcilerOptions,
	)

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
	_, tracker, r := getClientTrackerAndReconciler(
		t,
		defaultCentralConfig,
		nil,
		useRoutesReconcilerOptions,
	)
	tracker.AddRouteError(centralReencryptRouteName, errors.New("fake error"))
	_, err := r.Reconcile(context.TODO(), simpleManagedCentral)
	require.Errorf(t, err, "fake error")
}

func TestNoRoutesSentWhenOneNotAdmitted(t *testing.T) {
	_, tracker, r := getClientTrackerAndReconciler(
		t,
		defaultCentralConfig,
		nil,
		useRoutesReconcilerOptions,
	)
	tracker.SetRouteAdmitted(centralReencryptRouteName, false)
	_, err := r.Reconcile(context.TODO(), simpleManagedCentral)
	require.Errorf(t, err, "unable to find admitted ingress")
}

func TestNoRoutesSentWhenOneNotCreatedYet(t *testing.T) {
	_, tracker, r := getClientTrackerAndReconciler(
		t,
		defaultCentralConfig,
		nil,
		useRoutesReconcilerOptions,
	)
	tracker.SetSkipRoute(centralReencryptRouteName, true)
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
			reconcilerOptions := CentralReconcilerOptions{Telemetry: tc.telemetry}
			fakeClient, _, r := getClientTrackerAndReconciler(
				t,
				defaultCentralConfig,
				nil,
				reconcilerOptions,
			)

			_, err := r.Reconcile(context.TODO(), simpleManagedCentral)
			require.NoError(t, err)
			central := &v1alpha1.Central{}
			err = fakeClient.Get(context.TODO(), client.ObjectKey{Name: centralName, Namespace: centralNamespace}, central)
			require.NoError(t, err)

			require.NotNil(t, central.Spec.Central.Telemetry.Enabled)
			assert.True(t, *central.Spec.Central.Telemetry.Enabled)
			require.NotNil(t, central.Spec.Central.Telemetry.Storage.Endpoint)
			assert.Equal(t, tc.telemetry.StorageEndpoint, *central.Spec.Central.Telemetry.Storage.Endpoint)
			require.NotNil(t, central.Spec.Central.Telemetry.Storage.Key)
			if tc.telemetry.StorageKey == "" {
				assert.Equal(t, "DISABLED", *central.Spec.Central.Telemetry.Storage.Key)
			} else {
				assert.Equal(t, tc.telemetry.StorageKey, *central.Spec.Central.Telemetry.Storage.Key)
			}
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
			fakeClient, _, r := getClientTrackerAndReconciler(
				t,
				defaultCentralConfig,
				nil,
				useRoutesReconcilerOptions,
			)
			r.routeService = k8s.NewRouteService(fakeClient, &defaultRouteConfig)
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
	fakeClient, _, r := getClientTrackerAndReconciler(
		t,
		defaultCentralConfig,
		nil,
		defaultReconcilerOptions,
	)
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
			assert.NoError(t, r.ensureSecretExists(ctx, centralNamespace, tc.secretName, secretModifyFunc))
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
		assert.NoError(t, r.ensureSecretExists(ctx, centralNamespace, secretName, secretModifyFunc))
		assert.NoError(t, fakeClient.Get(ctx, client.ObjectKey{Namespace: centralNamespace, Name: secretName}, fetchedSecret))
		compareSecret(t, getSecret(secretName, centralNamespace, expectedData), fetchedSecret, true)
	})
}

func TestGetInstanceConfigSetsNoProxyEnvVarsForAuditLog(t *testing.T) {
	testCases := []struct {
		auditLoggingConfig config.AuditLogging
	}{
		{
			auditLoggingConfig: defaultAuditLogConfig,
		},
		{
			auditLoggingConfig: vectorAuditLogConfig,
		},
		{
			auditLoggingConfig: disabledAuditLogConfig,
		},
	}
	for _, testCase := range testCases {
		reconcilerOptions := CentralReconcilerOptions{
			AuditLogging: testCase.auditLoggingConfig,
		}
		_, _, r := getClientTrackerAndReconciler(
			t,
			simpleManagedCentral,
			nil,
			reconcilerOptions,
		)
		centralConfig, err := r.getInstanceConfig(&simpleManagedCentral)
		assert.NoError(t, err)
		require.NotNil(t, centralConfig)
		require.NotNil(t, centralConfig.Spec.Customize)
		noProxyEnvLowerCaseFound := false
		noProxyEnvUpperCaseFound := false
		for _, envVar := range centralConfig.Spec.Customize.EnvVars {
			switch envVar.Name {
			case "no_proxy":
				noProxyEnvLowerCaseFound = true
				assert.Contains(t, strings.Split(envVar.Value, ","), testCase.auditLoggingConfig.Endpoint(false))
			case "NO_PROXY":
				noProxyEnvUpperCaseFound = true
				assert.Contains(t, strings.Split(envVar.Value, ","), testCase.auditLoggingConfig.Endpoint(false))
			}
		}
		assert.True(t, noProxyEnvLowerCaseFound)
		assert.True(t, noProxyEnvUpperCaseFound)
	}
}

func TestGetInstanceConfigSetsDeclarativeConfigSecretInCentralCR(t *testing.T) {
	reconcilerOptions := CentralReconcilerOptions{
		AuditLogging: defaultAuditLogConfig,
	}
	_, _, r := getClientTrackerAndReconciler(
		t,
		simpleManagedCentral,
		nil,
		reconcilerOptions,
	)
	centralConfig, err := r.getInstanceConfig(&simpleManagedCentral)
	assert.NoError(t, err)
	require.NotNil(t, centralConfig)
	require.NotNil(t, centralConfig.Spec.Central)
	require.NotNil(t, centralConfig.Spec.Central.DeclarativeConfiguration)
	centralCRDeclarativeConfig := centralConfig.Spec.Central.DeclarativeConfiguration
	assert.NotZero(t, len(centralCRDeclarativeConfig.Secrets))
	expectedReconciledSecretReference := v1alpha1.LocalSecretReference{ // pragma: allowlist secret
		Name: sensibleDeclarativeConfigSecretName,
	}
	expectedManualSecretReference := v1alpha1.LocalSecretReference{ // pragma: allowlist secret
		Name: manualDeclarativeConfigSecretName,
	}
	assert.Contains(t, centralCRDeclarativeConfig.Secrets, expectedReconciledSecretReference)
	assert.Contains(t, centralCRDeclarativeConfig.Secrets, expectedManualSecretReference)
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
				simpleManagedCentral,
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
			fakeClient, _, r := getClientTrackerAndReconciler(t, simpleManagedCentral, nil, defaultReconcilerOptions, tc.mockObjects...)
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

func Test_getCentralConfig_telemetry(t *testing.T) {

	type args struct {
		isInternal bool
		storageKey string
	}

	tcs := []struct {
		name   string
		args   args
		assert func(t *testing.T, c *v1alpha1.Central)
	}{
		{
			name: "telemetry enabled, but DISABLED when no storage key is set",
			args: args{
				isInternal: false,
				storageKey: "",
			},
			assert: func(t *testing.T, c *v1alpha1.Central) {
				assert.True(t, *c.Spec.Central.Telemetry.Enabled)
				assert.Equal(t, "DISABLED", *c.Spec.Central.Telemetry.Storage.Key)
			},
		},
		{
			name: "should DISABLE telemetry key when managed central is internal",
			args: args{
				isInternal: true,
				storageKey: "foo",
			},
			assert: func(t *testing.T, c *v1alpha1.Central) {
				assert.True(t, *c.Spec.Central.Telemetry.Enabled)
				assert.Equal(t, "DISABLED", *c.Spec.Central.Telemetry.Storage.Key)
			},
		},
		{
			name: "should enable telemetry when storage key is set and managed central is not internal",
			args: args{
				isInternal: false,
				storageKey: "foo",
			},
			assert: func(t *testing.T, c *v1alpha1.Central) {
				assert.True(t, *c.Spec.Central.Telemetry.Enabled)
				assert.Equal(t, "foo", *c.Spec.Central.Telemetry.Storage.Key)
			},
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			r := &CentralReconciler{
				telemetry: config.Telemetry{
					StorageKey: tc.args.storageKey,
				},
			}
			c := &v1alpha1.Central{}
			mc := &private.ManagedCentral{
				Metadata: private.ManagedCentralAllOfMetadata{
					Internal: tc.args.isInternal,
				},
			}
			r.applyTelemetry(mc, c)
			tc.assert(t, c)
		})
	}
}

func TestReconciler_applyRoutes(t *testing.T) {
	type args struct {
		useRoutes bool
	}

	tcs := []struct {
		name   string
		args   args
		assert func(t *testing.T, c *v1alpha1.Central)
	}{
		{
			name: "should DISABLE routes when useRoutes is false",
			args: args{
				useRoutes: false,
			},
			assert: func(t *testing.T, c *v1alpha1.Central) {
				assert.False(t, *c.Spec.Central.Exposure.Route.Enabled)
			},
		}, {
			name: "should ENABLE routes when useRoutes is true",
			args: args{
				useRoutes: true,
			},
			assert: func(t *testing.T, c *v1alpha1.Central) {
				assert.True(t, *c.Spec.Central.Exposure.Route.Enabled)
			},
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			r := &CentralReconciler{
				useRoutes: tc.args.useRoutes,
			}
			c := &v1alpha1.Central{}
			r.applyRoutes(c)
			tc.assert(t, c)
		})
	}
}

func TestReconciler_applyProxyConfig(t *testing.T) {

	r := &CentralReconciler{
		auditLogging: config.AuditLogging{
			AuditLogTargetHost: "host",
			AuditLogTargetPort: 9000,
		},
	}
	c := &v1alpha1.Central{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "namespace",
		},
	}
	r.applyProxyConfig(c)

	assert.Equal(t, c.Spec.Customize.EnvVars, []v1.EnvVar{
		{
			Name:  "http_proxy",
			Value: "http://egress-proxy.namespace.svc:3128",
		},
		{
			Name:  "HTTP_PROXY",
			Value: "http://egress-proxy.namespace.svc:3128",
		},
		{
			Name:  "https_proxy",
			Value: "http://egress-proxy.namespace.svc:3128",
		},
		{
			Name:  "HTTPS_PROXY",
			Value: "http://egress-proxy.namespace.svc:3128",
		},
		{
			Name:  "all_proxy",
			Value: "http://egress-proxy.namespace.svc:3128",
		},
		{
			Name:  "ALL_PROXY",
			Value: "http://egress-proxy.namespace.svc:3128",
		},
		{
			Name:  "no_proxy",
			Value: "central.namespace.svc:443,central.namespace:443,central:443,host:9000,kubernetes.default.svc.cluster.local.:443,scanner-db.namespace.svc:5432,scanner-db.namespace:5432,scanner-db:5432,scanner-v4-db.namespace.svc:5432,scanner-v4-db.namespace:5432,scanner-v4-db:5432,scanner-v4-indexer.namespace.svc:8443,scanner-v4-indexer.namespace:8443,scanner-v4-indexer:8443,scanner-v4-matcher.namespace.svc:8443,scanner-v4-matcher.namespace:8443,scanner-v4-matcher:8443,scanner.namespace.svc:8080,scanner.namespace.svc:8443,scanner.namespace:8080,scanner.namespace:8443,scanner:8080,scanner:8443",
		},
		{
			Name:  "NO_PROXY",
			Value: "central.namespace.svc:443,central.namespace:443,central:443,host:9000,kubernetes.default.svc.cluster.local.:443,scanner-db.namespace.svc:5432,scanner-db.namespace:5432,scanner-db:5432,scanner-v4-db.namespace.svc:5432,scanner-v4-db.namespace:5432,scanner-v4-db:5432,scanner-v4-indexer.namespace.svc:8443,scanner-v4-indexer.namespace:8443,scanner-v4-indexer:8443,scanner-v4-matcher.namespace.svc:8443,scanner-v4-matcher.namespace:8443,scanner-v4-matcher:8443,scanner.namespace.svc:8080,scanner.namespace.svc:8443,scanner.namespace:8080,scanner.namespace:8443,scanner:8080,scanner:8443",
		},
	})
}

func TestReconciler_applyDeclarativeConfig(t *testing.T) {
	r := &CentralReconciler{}
	c := &v1alpha1.Central{}
	r.applyDeclarativeConfig(c)
	assert.Equal(t, c.Spec.Central.DeclarativeConfiguration.Secrets, []v1alpha1.LocalSecretReference{
		{
			Name: "cloud-service-sensible-declarative-configs",
		}, {
			Name: "cloud-service-manual-declarative-configs",
		},
	})
}

func TestReconciler_applyAnnotations(t *testing.T) {
	r := &CentralReconciler{
		environment: "test",
		clusterName: "test",
	}
	c := &v1alpha1.Central{
		Spec: v1alpha1.CentralSpec{
			Customize: &v1alpha1.CustomizeSpec{
				Annotations: map[string]string{
					"foo": "bar",
				},
			},
		},
	}
	date := time.Date(2024, 01, 01, 0, 0, 0, 0, time.UTC)
	rc := &private.ManagedCentral{
		Metadata: private.ManagedCentralAllOfMetadata{
			ExpiredAt: &date,
		},
	}
	r.applyAnnotations(rc, c)
	assert.Equal(t, map[string]string{
		"rhacs.redhat.com/environment":  "test",
		"rhacs.redhat.com/cluster-name": "test",
		"foo":                           "bar",
		"rhacs.redhat.com/expired-at":   "2024-01-01T00:00:00Z",
	}, c.Spec.Customize.Annotations)
}

func TestReconciler_getInstanceConfig(t *testing.T) {

	tcs := []struct {
		name          string
		yaml          string
		expectErr     bool
		expectCentral *v1alpha1.Central
	}{
		{
			name:      "should return error when yaml is invalid",
			yaml:      "invalid yaml",
			expectErr: true,
		}, {
			name: "should unmashal yaml to central",
			yaml: `
apiVersion: platform.stackrox.io/v1alpha1
kind: Central
metadata:
  name: central
  namespace: rhacs
`,
			expectCentral: &v1alpha1.Central{
				TypeMeta: metav1.TypeMeta{
					Kind:       "Central",
					APIVersion: "platform.stackrox.io/v1alpha1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "central",
					Namespace: "rhacs",
				},
				Spec: v1alpha1.CentralSpec{
					Central: &v1alpha1.CentralComponentSpec{
						Exposure: &v1alpha1.Exposure{
							Route: &v1alpha1.ExposureRoute{
								Enabled: pointer.Bool(false),
							},
						},
						DeclarativeConfiguration: &v1alpha1.DeclarativeConfiguration{
							Secrets: []v1alpha1.LocalSecretReference{
								{
									Name: "cloud-service-sensible-declarative-configs",
								}, {
									Name: "cloud-service-manual-declarative-configs",
								},
							},
						},
						Telemetry: &v1alpha1.Telemetry{
							Enabled: pointer.Bool(true),
							Storage: &v1alpha1.TelemetryStorage{
								Endpoint: pointer.String(""),
								Key:      pointer.String("DISABLED"),
							},
						},
					},
					Customize: &v1alpha1.CustomizeSpec{
						Annotations: map[string]string{
							"rhacs.redhat.com/environment":  "",
							"rhacs.redhat.com/cluster-name": "",
						},
						EnvVars: []v1.EnvVar{
							{
								Name:  "http_proxy",
								Value: "http://egress-proxy.rhacs.svc:3128",
							}, {
								Name:  "HTTP_PROXY",
								Value: "http://egress-proxy.rhacs.svc:3128",
							}, {
								Name:  "https_proxy",
								Value: "http://egress-proxy.rhacs.svc:3128",
							}, {
								Name:  "HTTPS_PROXY",
								Value: "http://egress-proxy.rhacs.svc:3128",
							}, {
								Name:  "all_proxy",
								Value: "http://egress-proxy.rhacs.svc:3128",
							}, {
								Name:  "ALL_PROXY",
								Value: "http://egress-proxy.rhacs.svc:3128",
							}, {
								Name:  "no_proxy",
								Value: ":0,central.rhacs.svc:443,central.rhacs:443,central:443,kubernetes.default.svc.cluster.local.:443,scanner-db.rhacs.svc:5432,scanner-db.rhacs:5432,scanner-db:5432,scanner-v4-db.rhacs.svc:5432,scanner-v4-db.rhacs:5432,scanner-v4-db:5432,scanner-v4-indexer.rhacs.svc:8443,scanner-v4-indexer.rhacs:8443,scanner-v4-indexer:8443,scanner-v4-matcher.rhacs.svc:8443,scanner-v4-matcher.rhacs:8443,scanner-v4-matcher:8443,scanner.rhacs.svc:8080,scanner.rhacs.svc:8443,scanner.rhacs:8080,scanner.rhacs:8443,scanner:8080,scanner:8443",
							}, {
								Name:  "NO_PROXY",
								Value: ":0,central.rhacs.svc:443,central.rhacs:443,central:443,kubernetes.default.svc.cluster.local.:443,scanner-db.rhacs.svc:5432,scanner-db.rhacs:5432,scanner-db:5432,scanner-v4-db.rhacs.svc:5432,scanner-v4-db.rhacs:5432,scanner-v4-db:5432,scanner-v4-indexer.rhacs.svc:8443,scanner-v4-indexer.rhacs:8443,scanner-v4-indexer:8443,scanner-v4-matcher.rhacs.svc:8443,scanner-v4-matcher.rhacs:8443,scanner-v4-matcher:8443,scanner.rhacs.svc:8080,scanner.rhacs.svc:8443,scanner.rhacs:8080,scanner.rhacs:8443,scanner:8080,scanner:8443",
							},
						},
					},
				},
			},
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			r := &CentralReconciler{}
			mc := &private.ManagedCentral{
				Spec: private.ManagedCentralAllOfSpec{
					CentralCRYAML: tc.yaml,
				},
			}
			c, err := r.getInstanceConfig(mc)
			if tc.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.expectCentral, c)
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
			fakeClient, _, r := getClientTrackerAndReconciler(t, simpleManagedCentral, nil, defaultReconcilerOptions)
			if tt.existingNamespace != nil {
				require.NoError(t, fakeClient.Create(context.Background(), tt.existingNamespace))
			}
			updateCount := 0
			createCount := 0
			r.client = interceptor.NewClient(fakeClient, interceptor.Funcs{
				Update: func(ctx context.Context, client client.WithWatch, obj client.Object, opts ...client.UpdateOption) error {
					updateCount++
					return client.Update(ctx, obj, opts...)
				},
				Create: func(ctx context.Context, client client.WithWatch, obj client.Object, opts ...client.CreateOption) error {
					createCount++
					return client.Create(ctx, obj, opts...)
				},
			})
			err := r.reconcileNamespace(context.Background(), managedCentral)
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

func TestReconciler_ensureChartResourcesExist_labelsAndAnnotations(t *testing.T) {
	fakeClient, _, r := getClientTrackerAndReconciler(t, simpleManagedCentral, nil, defaultReconcilerOptions)
	require.NoError(t, r.ensureChartResourcesExist(context.Background(), simpleManagedCentral))

	var egressProxyDeployment appsv1.Deployment
	err := fakeClient.Get(context.Background(), client.ObjectKey{
		Namespace: simpleManagedCentral.Metadata.Namespace,
		Name:      "egress-proxy",
	}, &egressProxyDeployment)
	require.NoError(t, err)

	assert.Equal(t, map[string]string{
		"app.kubernetes.io/instance":     "test-central",
		"app.kubernetes.io/managed-by":   "rhacs-fleetshard",
		"rhacs.redhat.com/instance-type": "standard",
		"rhacs.redhat.com/org-id":        "12345",
		"rhacs.redhat.com/tenant":        "cb45idheg5ip6dq1jo4g",
		"app.kubernetes.io/component":    "egress-proxy",
		"app.kubernetes.io/name":         "central-tenant-resources",
		"helm.sh/chart":                  "central-tenant-resources-0.0.0",
	}, egressProxyDeployment.ObjectMeta.Labels)

	assert.Equal(t, map[string]string{
		"rhacs.redhat.com/org-name": "org-name",
	}, egressProxyDeployment.ObjectMeta.Annotations)

}

func TestReconciler_needsReconcile(t *testing.T) {
	tests := []struct {
		name string
		// central (hash) has changed
		changed bool
		// the central to reconcile
		central *v1alpha1.Central
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
			central:           &v1alpha1.Central{},
			secretsStoredFunc: func([]string) bool { return true },
			timePassed:        0,
			want:              false,
		}, {
			name:              "central changed",
			changed:           true,
			central:           &v1alpha1.Central{},
			secretsStoredFunc: func([]string) bool { return true },
			timePassed:        0,
			want:              true,
		}, {
			name:              "secrets not stored",
			changed:           false,
			central:           &v1alpha1.Central{},
			secretsStoredFunc: func([]string) bool { return false },
			timePassed:        0,
			want:              true,
		}, {
			name:              "time passed",
			changed:           false,
			central:           &v1alpha1.Central{},
			secretsStoredFunc: func([]string) bool { return true },
			timePassed:        1 * time.Hour,
			want:              true,
		}, {
			name:    "force reconcile",
			changed: false,
			central: &v1alpha1.Central{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"rhacs.redhat.com/force-reconcile": "true",
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
			_, _, r := getClientTrackerAndReconciler(t, simpleManagedCentral, nil, defaultReconcilerOptions)
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

// TestReconcilerRaceCondition tests that reconciling a central that changes in quick
// succession will still be able to accurately reconcile the central.
// The reason for this test is that the reconciler will exit early if the deployment
// is not ready.
func TestReconcilerRaceCondition(t *testing.T) {
	var managedCentral = simpleManagedCentral
	managedCentral.RequestStatus = centralConstants.CentralRequestStatusReady.String()
	// Creating 2 "ready" centrals, with 2 different cpu limit values
	var central1 = withCpuLimit(t, managedCentral, "100m")
	var central2 = withCpuLimit(t, managedCentral, "200m")

	cli, _, r := getClientTrackerAndReconciler(t, central1, nil, defaultReconcilerOptions)
	ctx := context.Background()
	namespace := central1.Metadata.Namespace
	name := central1.Metadata.Name

	// we mock the "needsReconcileFunc" to only return true when the hash changes
	r.needsReconcileFunc = func(changed bool, central *v1alpha1.Central, storedSecrets []string) bool {
		return changed
	}
	// we mock the "restoreCentralSecrets" to always succeed
	r.restoreCentralSecretsFunc = func(ctx context.Context, remoteCentral private.ManagedCentral) error {
		return nil
	}

	// Perform first reconciliation
	_, err := r.Reconcile(ctx, central1)
	require.NoError(t, err)
	printCentralHash(t, r)
	assertCentralCpuLimit(t, ctx, cli, namespace, name, "100m")

	// Perform second reconciliation
	_, err = r.Reconcile(ctx, central2)
	require.NoError(t, err)
	printCentralHash(t, r)
	assertCentralCpuLimit(t, ctx, cli, namespace, name, "200m")

	makeDeploymentNotReady(t, ctx, cli, namespace)

	// Reconcile with first central again
	_, err = r.Reconcile(ctx, central1)
	require.NoError(t, err)
	printCentralHash(t, r)
	assertCentralCpuLimit(t, ctx, cli, namespace, name, "100m")

	// Then reconcile with second central
	_, err = r.Reconcile(ctx, central2)
	require.NoError(t, err)
	printCentralHash(t, r)
	assertCentralCpuLimit(t, ctx, cli, namespace, name, "200m")

	makeDeploymentReady(t, ctx, cli, namespace)

	// Reconcile with first central again
	_, err = r.Reconcile(ctx, central1)
	require.NoError(t, err)
	printCentralHash(t, r)
	assertCentralCpuLimit(t, ctx, cli, namespace, name, "100m")

	// Then reconcile with second central
	_, err = r.Reconcile(ctx, central2)
	require.NoError(t, err)
	printCentralHash(t, r)
	assertCentralCpuLimit(t, ctx, cli, namespace, name, "200m")
}

func printCentralHash(t *testing.T, reconciler *CentralReconciler) {
	t.Logf("Last central hash: %x", reconciler.lastCentralHash[:])
}

func makeDeploymentNotReady(t *testing.T, ctx context.Context, cli client.WithWatch, namespace string) {
	deployment := &appsv1.Deployment{}
	require.NoError(t, cli.Get(ctx, client.ObjectKey{Namespace: namespace, Name: centralDeploymentName}, deployment))
	deployment.Status.AvailableReplicas = 0
	require.NoError(t, cli.Status().Update(ctx, deployment))
	// ensure deployment is not ready
	assertDeploymentNotReady(t, ctx, cli, namespace)
}

func makeDeploymentReady(t *testing.T, ctx context.Context, cli client.WithWatch, namespace string) {
	deployment := &appsv1.Deployment{}
	require.NoError(t, cli.Get(ctx, client.ObjectKey{Namespace: namespace, Name: centralDeploymentName}, deployment))
	deployment.Status.AvailableReplicas = 1
	require.NoError(t, cli.Status().Update(ctx, deployment))
	// ensure deployment is not ready
	assertDeploymentReady(t, ctx, cli, namespace)
}

func assertDeploymentReady(t *testing.T, ctx context.Context, cli client.WithWatch, namespace string) {
	isReady, err := isCentralDeploymentReady(ctx, cli, namespace)
	require.NoError(t, err)
	require.True(t, isReady)
}

func assertDeploymentNotReady(t *testing.T, ctx context.Context, cli client.WithWatch, namespace string) {
	isReady, err := isCentralDeploymentReady(ctx, cli, namespace)
	require.NoError(t, err)
	require.False(t, isReady)
}

func assertCentralCpuLimit(t *testing.T, ctx context.Context, cli client.WithWatch, namespace, name string, cpuLimit string) {
	var central v1alpha1.Central
	require.NoError(t, cli.Get(ctx, client.ObjectKey{Namespace: namespace, Name: name}, &central))
	require.NotNil(t, central.Spec.Central)
	require.NotNil(t, central.Spec.Central.Resources)
	require.NotNil(t, central.Spec.Central.Resources.Requests)
	require.NotNil(t, central.Spec.Central.Resources.Requests.Cpu())
	assert.Equal(t, cpuLimit, central.Spec.Central.Resources.Requests.Cpu().String())
}

func withCpuLimit(t *testing.T, central private.ManagedCentral, cpuLimit string) private.ManagedCentral {
	var cr v1alpha1.Central
	err := yaml2.Unmarshal([]byte(central.Spec.CentralCRYAML), &cr)
	require.NoError(t, err)

	if cr.Spec.Central == nil {
		cr.Spec.Central = &v1alpha1.CentralComponentSpec{}
	}
	if cr.Spec.Central.Resources == nil {
		cr.Spec.Central.Resources = &v1.ResourceRequirements{}
	}
	if cr.Spec.Central.Resources.Requests == nil {
		cr.Spec.Central.Resources.Requests = make(v1.ResourceList)
	}
	cr.Spec.Central.Resources.Requests["cpu"] = resource.MustParse(cpuLimit)
	central2Yaml, err := yaml2.Marshal(cr)
	require.NoError(t, err)
	var clone private.ManagedCentral = central
	clone.Spec.CentralCRYAML = string(central2Yaml)
	return clone
}
